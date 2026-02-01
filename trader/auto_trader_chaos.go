package trader

import (
	"encoding/json"
	"fmt"
	"nofx/chaos"
	"nofx/logger"
	"nofx/store"
	"strings"
	"time"
)

// RunChaosCycle runs one trading cycle using Chaos Engine (Plan B)
// This is a standalone evolution of the trading logic specifically for Chaos mode
func (at *AutoTrader) RunChaosCycle() error {
	at.callCount++

	logger.Info("\n" + strings.Repeat("=", 70) + "\n")
	logger.Infof("🌀 %s - Chaos Cycle #%d", time.Now().Format("2006-01-02 15:04:05"), at.callCount)
	logger.Info(strings.Repeat("=", 70))

	// 0. Check if trader is stopped
	at.isRunningMutex.RLock()
	running := at.isRunning
	at.isRunningMutex.RUnlock()
	if !running {
		logger.Infof("⏹ Trader is stopped, aborting Chaos cycle #%d", at.callCount)
		return nil
	}

	// Create decision record
	record := &store.DecisionRecord{
		ExecutionLog: []string{},
		Success:      true,
		TraderID:     at.id,
		CycleNumber:  at.callCount,
		Timestamp:    time.Now().UTC(),
	}

	// 1. Check if trading needs to be stopped (Risk Control)
	if time.Now().Before(at.stopUntil) {
		remaining := at.stopUntil.Sub(time.Now())
		logger.Infof("⏸ Risk control: Trading paused, remaining %.0f minutes", remaining.Minutes())
		record.Success = false
		record.ErrorMessage = fmt.Sprintf("Risk control paused, remaining %.0f minutes", remaining.Minutes())
		at.saveDecision(record)
		return nil
	}

	// 2. Reset daily P&L
	if time.Since(at.lastResetTime) > 24*time.Hour {
		at.dailyPnL = 0
		at.lastResetTime = time.Now()
		logger.Info("📅 Daily P&L reset")
	}

	// 3. Collect trading context
	ctx, err := at.buildTradingContext()
	if err != nil {
		record.Success = false
		record.ErrorMessage = fmt.Sprintf("Failed to build trading context: %v", err)
		at.saveDecision(record)
		return fmt.Errorf("failed to build trading context: %w", err)
	}

	// Skip if no candidate coins
	if len(ctx.CandidateCoins) == 0 {
		logger.Infof("ℹ️  No candidate coins available, skipping this cycle")
		return nil
	}

	// Save equity snapshot
	at.saveEquitySnapshot(ctx)

	// Log account status
	logger.Infof("📊 Account equity: %.2f USDT | Available: %.2f USDT | Positions: %d",
		ctx.Account.TotalEquity, ctx.Account.AvailableBalance, ctx.Account.PositionCount)

	// 4. Execute Chaos Engine
	logger.Infof("🌀 Requesting Chaos AI analysis...")
	
	// Initialize Chaos Engine
	chaosEngine := chaos.NewChaosEngine(at.config.StrategyConfig)
	
	// Execute AI Decision
	aiDecision, err := chaosEngine.Execute(ctx, at.mcpClient)
	
	// Record metrics
	if aiDecision != nil && aiDecision.AIRequestDurationMs > 0 {
		record.AIRequestDurationMs = aiDecision.AIRequestDurationMs
		logger.Infof("⏱️ AI call duration: %.2f seconds", float64(record.AIRequestDurationMs)/1000)
		record.ExecutionLog = append(record.ExecutionLog,
			fmt.Sprintf("AI call duration: %d ms", record.AIRequestDurationMs))
	}

	// Save detailed decision info
	if aiDecision != nil {
		record.SystemPrompt = aiDecision.SystemPrompt
		record.InputPrompt = aiDecision.UserPrompt
		record.CoTTrace = aiDecision.CoTTrace
		record.RawResponse = aiDecision.RawResponse
		
		if aiDecision.RawDecisions != nil {
			decisionJSON, _ := json.MarshalIndent(aiDecision.RawDecisions, "", "  ")
			record.DecisionJSON = string(decisionJSON)
		} else if len(aiDecision.Decisions) > 0 {
			decisionJSON, _ := json.MarshalIndent(aiDecision.Decisions, "", "  ")
			record.DecisionJSON = string(decisionJSON)
		}
	}

	// Handle AI errors
	if err != nil {
		record.Success = false
		record.ErrorMessage = fmt.Sprintf("Chaos Engine failed: %v", err)
		
		// Debug logging for error case
		if aiDecision != nil {
			logger.Info("\n" + strings.Repeat("=", 70))
			logger.Infof("📋 System prompt (error case)")
			logger.Info(aiDecision.SystemPrompt)
			
			if aiDecision.CoTTrace != "" {
				logger.Info("\n" + strings.Repeat("-", 70))
				logger.Info("💭 AI chain of thought (error case):")
				logger.Info(aiDecision.CoTTrace)
			}
		}
		
		at.saveDecision(record)
		return fmt.Errorf("Chaos Engine execution failed: %w", err)
	}

	// 5. Execution Logic
	logger.Info(strings.Repeat("-", 70))
	
	// Sort decisions: Close first, then Open
	sortedDecisions := sortDecisionsByPriority(aiDecision.Decisions)
	
	logger.Info("🔄 Execution order: Close positions first → Open positions later")
	for i, d := range sortedDecisions {
		logger.Infof("  [%d] %s %s", i+1, d.Symbol, d.Action)
	}
	logger.Info()

	// Check stopped status again
	at.isRunningMutex.RLock()
	running = at.isRunning
	at.isRunningMutex.RUnlock()
	if !running {
		logger.Infof("⏹ Trader stopped before execution, aborting")
		return nil
	}

	// Execute decisions
	for _, d := range sortedDecisions {
		at.isRunningMutex.RLock()
		running = at.isRunning
		at.isRunningMutex.RUnlock()
		if !running {
			logger.Infof("⏹ Trader stopped during execution")
			break
		}

		actionRecord := store.DecisionAction{
			Action:     d.Action,
			Symbol:     d.Symbol,
			Quantity:   0,
			Leverage:   d.Leverage,
			Price:      0,
			StopLoss:   d.StopLoss,
			TakeProfit: d.TakeProfit,
			Confidence: d.Confidence,
			Reasoning:  d.Reasoning,
			Timestamp:  time.Now().UTC(),
			Success:    false,
		}

		if err := at.executeDecisionWithRecord(&d, &actionRecord); err != nil {
			logger.Infof("❌ Failed to execute (%s %s): %v", d.Symbol, d.Action, err)
			actionRecord.Error = err.Error()
			record.ExecutionLog = append(record.ExecutionLog, fmt.Sprintf("❌ %s %s failed: %v", d.Symbol, d.Action, err))
		} else {
			actionRecord.Success = true
			record.ExecutionLog = append(record.ExecutionLog, fmt.Sprintf("✓ %s %s succeeded", d.Symbol, d.Action))
			time.Sleep(1 * time.Second)
		}

		record.Decisions = append(record.Decisions, actionRecord)
	}

	// 6. Save final record
	if err := at.saveDecision(record); err != nil {
		logger.Infof("⚠ Failed to save decision record: %v", err)
	}

	return nil
}

// IsChaosStrategy checks if the current configuration corresponds to a Chaos strategy
func (at *AutoTrader) IsChaosStrategy() bool {
	if at.config.StrategyConfig == nil {
		return false
	}
	// Check prompt for Chaos signature
	chaosManager := chaos.NewManager()
	return chaosManager.IsChaosMode(at.config.StrategyConfig.CustomPrompt)
}
