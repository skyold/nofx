package trader

import (
	"fmt"
	"nofx/chaos"
	"nofx/kernel"
	"nofx/logger"
	"nofx/market"
	"nofx/provider/nofxos"
	"nofx/store"
	"time"
)

// IsChaosStrategy checks if the current strategy is Chaos
func (at *AutoTrader) IsChaosStrategy() bool {
	if at.config.StrategyConfig == nil {
		return false
	}
	return at.config.StrategyConfig.StrategyType == "chaos_trading"
}

// RunChaosCycle runs a single chaos trading cycle
func (at *AutoTrader) RunChaosCycle() error {
	if !at.IsChaosStrategy() {
		return nil
	}

	at.cycleNumber++ // Increment cycle number

	logger.Infof("🌀 Starting Chaos Cycle #%d...", at.cycleNumber)

	// 1. Build Chaos Context
	ctx, err := at.buildChaosContext()
	if err != nil {
		logger.Errorf("❌ Failed to build Chaos context: %v", err)
		return err
	}

	// 2. Execute Chaos Engine
	result, err := chaos.GetDecisions(ctx, at.mcpClient)
	if err != nil {
		logger.Errorf("❌ Chaos Engine execution failed: %v", err)
		return err
	}

	// 3. Process Decisions
	at.processChaosResult(result, ctx)

	logger.Infof("✅ Chaos Cycle #%d completed", at.cycleNumber)
	return nil
}

func (at *AutoTrader) buildChaosContext() (*chaos.ChaosContext, error) {
	// 1. Get Account Info (Reuse logic from auto_trader.go)
	balance, err := at.trader.GetBalance()
	if err != nil {
		return nil, fmt.Errorf("failed to get account balance: %w", err)
	}

	totalWalletBalance := 0.0
	totalUnrealizedProfit := 0.0
	availableBalance := 0.0
	totalEquity := 0.0

	if wallet, ok := balance["totalWalletBalance"].(float64); ok {
		totalWalletBalance = wallet
	}
	if unrealized, ok := balance["totalUnrealizedProfit"].(float64); ok {
		totalUnrealizedProfit = unrealized
	}
	if avail, ok := balance["availableBalance"].(float64); ok {
		availableBalance = avail
	}
	if eq, ok := balance["totalEquity"].(float64); ok && eq > 0 {
		totalEquity = eq
	} else {
		totalEquity = totalWalletBalance + totalUnrealizedProfit
	}

	// 2. Get Positions
	positions, err := at.trader.GetPositions()
	if err != nil {
		return nil, fmt.Errorf("failed to get positions: %w", err)
	}

	// Convert positions to Chaos format
	var positionSnapshots []chaos.PositionSnapshot
	for _, pos := range positions {
		symbol := pos["symbol"].(string)
		side := pos["side"].(string)
		entryPrice := pos["entryPrice"].(float64)
		markPrice := pos["markPrice"].(float64)
		quantity := pos["positionAmt"].(float64)
		if quantity < 0 {
			quantity = -quantity
		}
		if quantity == 0 {
			continue
		}
		// unrealizedPnl := pos["unRealizedProfit"].(float64)
		liquidationPrice := pos["liquidationPrice"].(float64)

		leverage := 10
		if lev, ok := pos["leverage"].(float64); ok {
			leverage = int(lev)
		}

		// Calculate PnL Pct (use same logic as auto_trader.go)
		marginUsed := (quantity * markPrice) / float64(leverage)
		pnlPct := 0.0
		// Need to recalculate unrealizedPnl based on side to be safe, or use what's provided
		unrealizedPnl := pos["unRealizedProfit"].(float64)
		if marginUsed > 0 {
			pnlPct = (unrealizedPnl / marginUsed) * 100
		}

		positionSnapshots = append(positionSnapshots, chaos.PositionSnapshot{
			Symbol:           symbol,
			Side:             side,
			EntryPrice:       entryPrice,
			MarkPrice:        markPrice,
			Quantity:         quantity,
			Leverage:         leverage,
			UnrealizedPnLPct: pnlPct,
			LiquidationPrice: liquidationPrice,
			UpdateTime:       time.Now().UnixMilli(), // Approximate
		})
	}

	// 3. Prepare Candidate Coins
	candidateCoins := []chaos.CandidateCoin{}
	if at.strategyEngine != nil {
		candidates, err := at.strategyEngine.GetCandidateCoins()
		if err == nil {
			for _, c := range candidates {
				candidateCoins = append(candidateCoins, chaos.CandidateCoin{
					Symbol:  c.Symbol,
					Sources: c.Sources,
				})
			}
		}
	}

	// 4. Fetch Market Data (Chaos specific fetching)
	marketDataMap := make(map[string]*market.Data)

	// Config access
	var chaosConfig *store.ChaosStrategyConfig
	if at.config.StrategyConfig.ChaosConfig != nil {
		chaosConfig = at.config.StrategyConfig.ChaosConfig
	} else {
		// Default config if nil
		chaosConfig = &store.ChaosStrategyConfig{}
	}
	indicatorsConfig := at.config.StrategyConfig.Indicators

	// Using default logic if config missing
	primaryTimeframe := "1h"
	klineCount := 100
	if indicatorsConfig.Klines.PrimaryTimeframe != "" {
		primaryTimeframe = indicatorsConfig.Klines.PrimaryTimeframe
	}
	if indicatorsConfig.Klines.PrimaryCount > 0 {
		klineCount = indicatorsConfig.Klines.PrimaryCount
	}

	symbolsToFetch := make(map[string]bool)
	for _, p := range positionSnapshots {
		symbolsToFetch[p.Symbol] = true
	}
	for _, c := range candidateCoins {
		symbolsToFetch[c.Symbol] = true
	}

	for symbol := range symbolsToFetch {
		data, err := market.GetWithTimeframes(symbol, indicatorsConfig.Klines.SelectedTimeframes, primaryTimeframe, klineCount)
		if err != nil {
			logger.Warnf("Failed to fetch market data for %s: %v", symbol, err)
			continue
		}
		marketDataMap[symbol] = data
	}

	// 5. Get OI Top Data
	oiTopMap := make(map[string]*chaos.OITopData)
	if at.config.StrategyConfig.CoinSource.UseOITop {
		apiKey := at.config.StrategyConfig.Indicators.NofxOSAPIKey
		if apiKey == "" {
			apiKey = nofxos.DefaultAuthKey
		}
		client := nofxos.NewClient(nofxos.DefaultBaseURL, apiKey)
		oiPositions, err := client.GetOITopPositions()
		if err == nil {
			for _, p := range oiPositions {
				oiTopMap[p.Symbol] = &chaos.OITopData{
					Rank:              p.Rank,
					OIDeltaPercent:    p.OIDeltaPercent,
					OIDeltaValue:      p.OIDeltaValue,
					PriceDeltaPercent: p.PriceDeltaPercent,
				}
			}
		}
	}

	// 7. Assemble Context
	chaosCtx := &chaos.ChaosContext{
		CurrentTime:    time.Now().Format("2006-01-02 15:04:05"),
		RuntimeMinutes: int(time.Since(at.startTime).Minutes()),
		CallCount:      at.cycleNumber,
		Config: &chaos.ChaosConfig{
			ChaosPrompt:        chaosConfig.ChaosPrompt,
			RiskControl:        chaosConfig.RiskControl,
			PromptVariant:      chaosConfig.PromptVariant,
			FaultInjectionRate: chaosConfig.FaultInjectionRate,
			DataNoiseLevel:     chaosConfig.DataNoiseLevel,
			StressTestMode:     chaosConfig.StressTestMode,
			Indicators:         indicatorsConfig,
		},
		Account: chaos.AccountSnapshot{
			TotalEquity:      totalEquity,
			AvailableBalance: availableBalance,
			UnrealizedPnL:    totalUnrealizedProfit,
			PositionCount:    len(positionSnapshots),
		},
		Positions:      positionSnapshots,
		CandidateCoins: candidateCoins,
		MarketDataMap:  marketDataMap,
		OITopDataMap:   oiTopMap,
	}

	if totalEquity > 0 {
		chaosCtx.Account.TotalPnLPct = (totalUnrealizedProfit / totalEquity) * 100
		chaosCtx.Account.MarginUsedPct = ((totalEquity - availableBalance) / totalEquity) * 100
	}

	return chaosCtx, nil
}

func (at *AutoTrader) processChaosResult(result *chaos.DecisionResult, ctx *chaos.ChaosContext) {
	logger.Infof("🤖 Chaos AI Decision: %d decisions generated", len(result.Decisions))

	kernelDecisions := []kernel.Decision{}
	for _, d := range result.Decisions {
		kd := kernel.Decision{
			Symbol: d.Symbol,
			Action: d.Action,
		}
		if d.Leverage != nil {
			kd.Leverage = *d.Leverage
		}
		if d.EntryPrice != nil {
			kd.EntryPrice = *d.EntryPrice
		}
		if d.StopLoss != nil {
			kd.StopLoss = *d.StopLoss
		}
		if d.TakeProfit != nil {
			kd.TakeProfit = *d.TakeProfit
		}
		if d.RiskR != nil {
			kd.RiskR = *d.RiskR
		}
		if d.TotalScore != nil {
			kd.Confidence = *d.TotalScore
		}
		if d.Reasoning != nil {
			kd.Reasoning = *d.Reasoning
		}

		kernelDecisions = append(kernelDecisions, kd)
	}

	// Save to DB
	if at.store != nil {
		record := &store.DecisionRecord{
			TraderID:            at.id,
			CycleNumber:         at.cycleNumber,
			Timestamp:           result.Timestamp,
			SystemPrompt:        result.SystemPrompt,
			InputPrompt:         result.UserPrompt,
			CoTTrace:            result.CoTTrace,
			RawResponse:         result.RawResponse,
			AIRequestDurationMs: result.AIRequestDurationMs,
			Success:             true,
		}
		// Add decisions to record... (simplified for now, full detail is in RawResponse)
		if err := at.store.Decision().LogDecision(record); err != nil {
			logger.Errorf("Failed to save chaos decision: %v", err)
		}
	}

	if !ctx.Config.StressTestMode {
		at.executeChaosDecisions(kernelDecisions)
	} else {
		logger.Infof("🧪 Stress Test Mode: Skipping execution for %d decisions", len(kernelDecisions))
	}
}

func (at *AutoTrader) executeChaosDecisions(decisions []kernel.Decision) {
	// Sort decisions
	sortedDecisions := sortDecisionsByPriority(decisions)

	for _, d := range sortedDecisions {
		// Create action record
		actionRecord := store.DecisionAction{
			Action:     d.Action,
			Symbol:     d.Symbol,
			Leverage:   d.Leverage,
			StopLoss:   d.StopLoss,
			TakeProfit: d.TakeProfit,
			Confidence: d.Confidence,
			Reasoning:  d.Reasoning,
			Timestamp:  time.Now().UTC(),
		}

		// Execute
		if err := at.executeDecisionWithRecord(&d, &actionRecord); err != nil {
			logger.Infof("❌ Chaos execution failed (%s %s): %v", d.Symbol, d.Action, err)
		} else {
			logger.Infof("✓ Chaos execution succeeded (%s %s)", d.Symbol, d.Action)
		}
	}
}
