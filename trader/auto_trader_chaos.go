package trader

import (
	"encoding/json"
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

	if ctxBytes, err := json.MarshalIndent(ctx, "", "  "); err == nil {
		logger.Infof("🔍 Chaos Context Built:\n%s\n"+
			"📊 Data Summary:\n"+
			"- MarketDataMap Size: %d\n"+
			"- MultiTFMarket Size: %d\n"+
			"- OITopDataMap Size: %d\n"+
			"- QuantDataMap Size: %d",
			string(ctxBytes),
			len(ctx.MarketDataMap),
			len(ctx.MultiTFMarket),
			len(ctx.OITopDataMap),
			len(ctx.QuantDataMap),
		)
	} else {
		logger.Errorf("Failed to marshal Chaos Context for logging: %v", err)
	}

	// Save equity snapshot (Align with nofx AutoTrader behavior)
	at.saveChaosEquitySnapshot(ctx)

	// 2. Execute Chaos Engine (Decomposed)

	// Step 1: Build Prompts
	systemPrompt := at.chaosEngine.BuildSystemPromptWithContext(ctx)
	userPrompt := at.chaosEngine.BuildUserPrompt(ctx)

	// Step 2: Call AI
	aiCallStart := time.Now()
	aiResponse, err := at.mcpClient.CallWithMessages(systemPrompt, userPrompt)
	aiCallDuration := time.Since(aiCallStart)

	if err != nil {
		logger.Errorf("❌ Chaos Engine execution failed (LLM Call): %v", err)
		record.Success = false
		record.ErrorMessage = fmt.Sprintf("LLM Call failed: %v", err)
		record.ExecutionLog = append(record.ExecutionLog, fmt.Sprintf("❌ LLM Error: %v", err))
		if saveErr := at.saveDecision(record); saveErr != nil {
			logger.Errorf("Failed to save error decision: %v", saveErr)
		}
		return err
	}

	// Step 3: Parse & Extract
	decisions, decisionJSON, extractErr := chaos.ExtractDecisions(aiResponse)

	reasoning, _ := chaos.ExtractReasoningJSON(aiResponse)
	cotTrace := at.chaosEngine.ExtractCoTTrace(aiResponse)

	// Construct initial result for logging/error handling
	chaosDecision := &chaos.DecisionResult{
		SystemPrompt:        systemPrompt,
		UserPrompt:          userPrompt,
		CoTTrace:            cotTrace,
		Decisions:           nil, // Will be filled after validation
		RawDecisions:        decisions,
		DecisionJSON:        decisionJSON,
		RawResponse:         aiResponse,
		Timestamp:           time.Now(),
		AIRequestDurationMs: aiCallDuration.Milliseconds(),
	}

	if extractErr != nil {
		logger.Errorf("❌ Chaos Engine execution failed (Format Audit): %v", extractErr)
		record.Success = false
		record.ErrorMessage = fmt.Sprintf("Format Audit failed: %v", extractErr)
		record.ExecutionLog = append(record.ExecutionLog, fmt.Sprintf("❌ Format Error: %v", extractErr))

		// Save record with available info
		record.SystemPrompt = chaosDecision.SystemPrompt
		record.InputPrompt = chaosDecision.UserPrompt
		record.CoTTrace = chaosDecision.CoTTrace
		record.RawResponse = chaosDecision.RawResponse
		record.AIRequestDurationMs = chaosDecision.AIRequestDurationMs
		record.DecisionJSON = chaosDecision.DecisionJSON

		if saveErr := at.saveDecision(record); saveErr != nil {
			logger.Errorf("Failed to save error decision: %v", saveErr)
		}
		return extractErr
	}

	// Step 4: Validate
	// Manually validate decisions to handle errors gracefully (partial failure)
	manager := chaos.NewManager()
	var validatedDecisions []chaos.Decision
	var failedDecisions []store.DecisionAction
	riskConfig := ctx.Config.RiskControl

	for _, d := range decisions {
		positionSizeUSD, err := manager.ValidateDecision(&d, reasoning, ctx.Account.TotalEquity, riskConfig)
		if err != nil {
			logger.Warnf("⚠️ Chaos decision validation failed for %s: %v", d.Symbol, err)

			// Create failed action record
			failedAction := store.DecisionAction{
				Symbol:    d.Symbol,
				Action:    "wait",
				Timestamp: time.Now().UTC(),
				Success:   false,
				Error:     err.Error(),
				Reasoning: fmt.Sprintf("Validation failed: %v", err),
			}
			if d.TotalScore != nil {
				failedAction.Confidence = *d.TotalScore
			}
			failedDecisions = append(failedDecisions, failedAction)
			continue
		}

		// Store validated position size
		d.PositionSizeUSD = &positionSizeUSD
		validatedDecisions = append(validatedDecisions, d)
	}

	chaosDecision.Decisions = validatedDecisions

	// Fill record with AI decision results
	record.SystemPrompt = chaosDecision.SystemPrompt
	record.InputPrompt = chaosDecision.UserPrompt
	record.CoTTrace = chaosDecision.CoTTrace
	record.RawResponse = chaosDecision.RawResponse
	record.AIRequestDurationMs = chaosDecision.AIRequestDurationMs
	record.DecisionJSON = chaosDecision.DecisionJSON

	// Step 5: Execute Decisions
	actionRecord, executionLogs := at.executeChaosDecision(chaosDecision, ctx)

	// Append failed validation actions to record and logs
	if len(failedDecisions) > 0 {
		actionRecord = append(actionRecord, failedDecisions...)
		for _, fd := range failedDecisions {
			errMsg := fmt.Sprintf("❌ Validation failed (%s): %s", fd.Symbol, fd.Error)
			executionLogs = append(executionLogs, errMsg)
		}
	}

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
	if at.chaosEngine != nil {
		candidates, err := at.chaosEngine.GetCandidateCoins()
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

	// 5. Get OI Top Data (Legacy, kept for compatibility if needed)
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

	// 6. Fetch Market Rankings (New NoFxOS Data)
	// Respect Chaos-specific indicator switches for OI/NetFlow/Price rankings
	var oiRankingData *nofxos.OIRankingData
	var netFlowRankingData *nofxos.NetFlowRankingData
	var priceRankingData *nofxos.PriceRankingData
	var quantDataMap map[string]*kernel.QuantData

	if at.strategyEngine != nil {
		indicators := indicatorsConfig

		if indicators.EnableQuantData {
			symbols := make([]string, 0, len(symbolsToFetch))
			for s := range symbolsToFetch {
				symbols = append(symbols, s)
			}
			logger.Infof("📊 [%s] Fetching Quant Data for %d symbols", at.name, len(symbols))
			quantDataMap = at.strategyEngine.FetchQuantDataBatch(symbols)
		}

		if indicators.EnableOIRanking {
			logger.Infof("📊 [%s] Fetching OI ranking data (Chaos)", at.name)
			oiRankingData = at.strategyEngine.FetchOIRankingData()
			if oiRankingData != nil {
				logger.Infof("📊 [%s] OI ranking data ready (Chaos): %d top, %d low positions",
					at.name, len(oiRankingData.TopPositions), len(oiRankingData.LowPositions))
			}
		}

		if indicators.EnableNetFlowRanking {
			logger.Infof("💰 [%s] Fetching NetFlow ranking data (Chaos)", at.name)
			netFlowRankingData = at.strategyEngine.FetchNetFlowRankingData()
			if netFlowRankingData != nil {
				logger.Infof("💰 [%s] NetFlow ranking data ready (Chaos): inst_in=%d, inst_out=%d",
					at.name, len(netFlowRankingData.InstitutionFutureTop), len(netFlowRankingData.InstitutionFutureLow))
			}
		}

		if indicators.EnablePriceRanking {
			logger.Infof("📈 [%s] Fetching Price ranking data (Chaos)", at.name)
			priceRankingData = at.strategyEngine.FetchPriceRankingData()
			if priceRankingData != nil {
				logger.Infof("📈 [%s] Price ranking data ready (Chaos) for %d durations",
					at.name, len(priceRankingData.Durations))
			}
		}
	}

	// 6.5 Fetch Trading History (Recent Orders & Stats)
	var recentOrders []kernel.RecentOrder
	var tradingStats *kernel.TradingStats

	if at.store != nil {
		// Get recent 10 closed trades for AI context
		recentTrades, err := at.store.Position().GetRecentTrades(at.id, 10)
		if err != nil {
			logger.Infof("⚠️ [%s] Failed to get recent trades: %v", at.name, err)
		} else {
			for _, trade := range recentTrades {
				// Convert Unix timestamps to formatted strings for AI readability
				entryTimeStr := ""
				if trade.EntryTime > 0 {
					entryTimeStr = time.Unix(trade.EntryTime, 0).UTC().Format("01-02 15:04 UTC")
				}
				exitTimeStr := ""
				if trade.ExitTime > 0 {
					exitTimeStr = time.Unix(trade.ExitTime, 0).UTC().Format("01-02 15:04 UTC")
				}

				recentOrders = append(recentOrders, kernel.RecentOrder{
					Symbol:       trade.Symbol,
					Side:         trade.Side,
					EntryPrice:   trade.EntryPrice,
					ExitPrice:    trade.ExitPrice,
					RealizedPnL:  trade.RealizedPnL,
					PnLPct:       trade.PnLPct,
					EntryTime:    entryTimeStr,
					ExitTime:     exitTimeStr,
					HoldDuration: trade.HoldDuration,
				})
			}
		}

		// Get trading statistics for AI context
		stats, err := at.store.Position().GetFullStats(at.id)
		if err != nil {
			logger.Infof("⚠️ [%s] Failed to get trading stats: %v", at.name, err)
		} else if stats != nil && stats.TotalTrades > 0 {
			tradingStats = &kernel.TradingStats{
				TotalTrades:    stats.TotalTrades,
				WinRate:        stats.WinRate,
				ProfitFactor:   stats.ProfitFactor,
				SharpeRatio:    stats.SharpeRatio,
				TotalPnL:       stats.TotalPnL,
				AvgWin:         stats.AvgWin,
				AvgLoss:        stats.AvgLoss,
				MaxDrawdownPct: stats.MaxDrawdownPct,
			}
		}
	}

	// 7. Assemble Context
	chaosCtx := &chaos.ChaosContext{
		CurrentTime:    time.Now().Format("2006-01-02 15:04:05"),
		RuntimeMinutes: int(time.Since(at.startTime).Minutes()),
		CallCount:      at.callCount,
		Timeframes:     indicatorsConfig.Klines.SelectedTimeframes,
		Config: &chaos.ChaosConfig{
			ChaosPrompt:         chaosConfig.ChaosPrompt,
			RiskControl:         chaosConfig.RiskControl,
			SystemPromptVariant: chaosConfig.SystemPromptVariant,
			UserPromptVersion:   chaosConfig.UserPromptVersion,
			Indicators:          indicatorsConfig,
		},
		Account: kernel.AccountInfo{
			TotalEquity:      totalEquity,
			AvailableBalance: availableBalance,
			UnrealizedPnL:    totalUnrealizedProfit,
			PositionCount:    len(positionSnapshots),
		},
		Positions:          positionSnapshots,
		CandidateCoins:     candidateCoins,
		RecentOrders:       recentOrders,
		TradingStats:       tradingStats,
		MarketDataMap:      marketDataMap,
		QuantDataMap:       quantDataMap,
		OITopDataMap:       oiTopMap,
		OIRankingData:      oiRankingData,
		NetFlowRankingData: netFlowRankingData,
		PriceRankingData:   priceRankingData,
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
