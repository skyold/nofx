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

	at.callCount++

	// Create decision record
	record := &store.DecisionRecord{
		ExecutionLog: []string{},
		Success:      true,
	}

	logger.Infof("🌀 Starting Chaos Cycle #%d...", at.callCount)

	// 1. Build Chaos Context
	ctx, err := at.buildChaosContext()
	if err != nil {
		logger.Errorf("❌ Failed to build Chaos context: %v", err)
		record.Success = false
		record.ErrorMessage = fmt.Sprintf("Failed to build Chaos context: %v", err)
		record.ExecutionLog = append(record.ExecutionLog, fmt.Sprintf("❌ Error: %v", err))
		if saveErr := at.saveDecision(record); saveErr != nil {
			logger.Errorf("Failed to save error decision: %v", saveErr)
		}
		return err
	}

	// Save equity snapshot (Align with nofx AutoTrader behavior)
	at.saveChaosEquitySnapshot(ctx)

	// 2. Execute Chaos Engine (LLM call)
	chaosDecision, err := chaos.GetChaosDecisions(ctx, at.mcpClient)
	if err != nil {
		logger.Errorf("❌ Chaos Engine execution failed: %v", err)
		record.Success = false
		record.ErrorMessage = fmt.Sprintf("Chaos Engine execution failed: %v", err)
		record.ExecutionLog = append(record.ExecutionLog, fmt.Sprintf("❌ Engine Error: %v", err))
		if saveErr := at.saveDecision(record); saveErr != nil {
			logger.Errorf("Failed to save error decision: %v", saveErr)
		}
		return err
	}

	// Fill record with AI decision results
	record.SystemPrompt = chaosDecision.SystemPrompt
	record.InputPrompt = chaosDecision.UserPrompt
	record.CoTTrace = chaosDecision.CoTTrace
	record.RawResponse = chaosDecision.RawResponse
	record.AIRequestDurationMs = chaosDecision.AIRequestDurationMs
	record.DecisionJSON = chaosDecision.DecisionJSON

	// 3. Execute Decisions
	actionRecord, executionLogs := at.executeChaosDecision(chaosDecision, ctx)
	record.Decisions = actionRecord
	record.ExecutionLog = append(record.ExecutionLog, executionLogs...)

	// Save final decision record
	if err := at.saveDecision(record); err != nil {
		logger.Errorf("Failed to save final decision record: %v", err)
	}

	logger.Infof("✅ Chaos Cycle #%d completed", at.callCount)
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
	var positionSnapshots []kernel.PositionInfo
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

		positionSnapshots = append(positionSnapshots, kernel.PositionInfo{
			Symbol:           symbol,
			Side:             side,
			EntryPrice:       entryPrice,
			MarkPrice:        markPrice,
			Quantity:         quantity,
			Leverage:         leverage,
			UnrealizedPnL:    unrealizedPnl,
			UnrealizedPnLPct: pnlPct,
			PeakPnLPct:       pnlPct, // Approximate as current PnL for now
			MarginUsed:       marginUsed,
			LiquidationPrice: liquidationPrice,
			UpdateTime:       time.Now().UnixMilli(), // Approximate
		})
	}

	// 3. Prepare Candidate Coins
	candidateCoins := []kernel.CandidateCoin{}
	existingCandidateMap := make(map[string]bool)

	// 3.1 Fetch candidates from strategy
	if at.strategyEngine != nil {
		candidates, err := at.strategyEngine.GetCandidateCoins()
		if err == nil {
			for _, c := range candidates {
				candidateCoins = append(candidateCoins, kernel.CandidateCoin{
					Symbol:  c.Symbol,
					Sources: c.Sources,
				})
				existingCandidateMap[c.Symbol] = true
			}
		}
	}

	// 3.2 Merge existing positions into candidate coins
	// If we hold a position, we MUST treat it as a candidate to allow the AI to manage it (close/adjust)
	for _, pos := range positionSnapshots {
		if !existingCandidateMap[pos.Symbol] {
			logger.Infof("➕ Merging held position %s into candidate coins for management", pos.Symbol)
			candidateCoins = append(candidateCoins, kernel.CandidateCoin{
				Symbol:  pos.Symbol,
				Sources: []string{"Existing Position"},
			})
			existingCandidateMap[pos.Symbol] = true
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
	indicatorsConfig := chaosConfig.Indicators

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
	oiTopMap := make(map[string]*kernel.OITopData)
	if chaosConfig.CoinSource.UseOITop {
		apiKey := chaosConfig.Indicators.NofxOSAPIKey
		if apiKey == "" {
			apiKey = nofxos.DefaultAuthKey
		}
		client := nofxos.NewClient(nofxos.DefaultBaseURL, apiKey)
		oiPositions, err := client.GetOITopPositions()
		if err == nil {
			for _, p := range oiPositions {
				oiTopMap[p.Symbol] = &kernel.OITopData{
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
		CallCount:      at.callCount,
		Config: &chaos.ChaosConfig{
			ChaosPrompt:   chaosConfig.ChaosPrompt,
			RiskControl:   chaosConfig.RiskControl,
			PromptVariant: chaosConfig.PromptVariant,
			Indicators:    indicatorsConfig,
		},
		Account: kernel.AccountInfo{
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

// saveChaosEquitySnapshot saves equity snapshot for chaos mode (matches auto_trader behavior)
func (at *AutoTrader) saveChaosEquitySnapshot(ctx *chaos.ChaosContext) {
	if at.store == nil || ctx == nil {
		return
	}

	snapshot := &store.EquitySnapshot{
		TraderID:      at.id,
		Timestamp:     time.Now().UTC(),
		TotalEquity:   ctx.Account.TotalEquity,
		Balance:       ctx.Account.TotalEquity - ctx.Account.UnrealizedPnL,
		UnrealizedPnL: ctx.Account.UnrealizedPnL,
		PositionCount: ctx.Account.PositionCount,
		MarginUsedPct: ctx.Account.MarginUsedPct,
	}

	if err := at.store.Equity().Save(snapshot); err != nil {
		logger.Infof("⚠️ Failed to save equity snapshot: %v", err)
	}
}

func (at *AutoTrader) executeChaosDecision(result *chaos.DecisionResult, ctx *chaos.ChaosContext) ([]store.DecisionAction, []string) {

	logger.Infof("🤖 Chaos AI Decision: %d decisions generated", len(result.Decisions))

	var actionRecord []store.DecisionAction
	var executionLogs []string

	// Create Executor
	executor := chaos.NewChaosExecutor(
		at.trader, // AutoTrader.trader implements TraderInterface
		at.store,
		at.id,
		at.exchange,
		at.exchangeID,
		ctx.Config,
	)
	actionRecord, executionLogs = executor.Execute(result.Decisions)

	return actionRecord, executionLogs
}
