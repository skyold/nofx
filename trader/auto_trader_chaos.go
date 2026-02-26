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

// IsChaosStrategy 检查当前策略是否为混沌交易策略
// 返回值：如果策略类型为 "chaos_trading" 则返回 true，否则返回 false
func (at *AutoTrader) IsChaosStrategy() bool {
	if at.config.StrategyConfig == nil {
		return false
	}
	return at.config.StrategyConfig.StrategyType == "chaos_trading"
}

// RunChaosCycle 执行一次混沌交易周期
// 主要流程：
// 1. 构建混沌上下文（包含账户信息、持仓、市场数据等）
// 2. 保存权益快照
// 3. 执行混沌引擎（构建提示词、调用AI、解析决策、验证决策、执行决策）
// 4. 保存决策记录
// 返回值：执行过程中的错误，如果成功则返回 nil
func (at *AutoTrader) RunChaosCycle() error {
	// 检查是否为混沌策略，如果不是则直接返回
	if !at.IsChaosStrategy() {
		return nil
	}

	// 增加调用计数
	at.callCount++

	// 创建决策记录对象，用于记录本次周期的执行情况
	record := &store.DecisionRecord{
		ExecutionLog: []string{},
		Success:      true,
	}

	logger.Infof("🌀 开始混沌交易周期 #%d...", at.callCount)

	// ========================================
	// 第一步：构建混沌上下文
	// ========================================
	ctx, err := at.buildChaosContext()
	if err != nil {
		logger.Errorf("❌ 构建混沌上下文失败: %v", err)
		record.Success = false
		record.ErrorMessage = fmt.Sprintf("构建混沌上下文失败: %v", err)
		record.ExecutionLog = append(record.ExecutionLog, fmt.Sprintf("❌ 错误: %v", err))
		if saveErr := at.saveDecision(record); saveErr != nil {
			logger.Errorf("保存错误决策记录失败: %v", saveErr)
		}
		return err
	}

	// 记录构建好的上下文信息（用于调试）
	if ctxBytes, err := json.MarshalIndent(ctx, "", "  "); err == nil {
		logger.Infof("🔍 混沌上下文已构建:\n%s\n"+
			"📊 数据摘要:\n"+
			"- 市场数据映射大小: %d\n"+
			"- 多时间框架市场大小: %d\n"+
			"- 持仓量排名数据大小: %d\n"+
			"- 量化数据大小: %d",
			string(ctxBytes),
			len(ctx.MarketDataMap),
			len(ctx.MultiTFMarket),
			len(ctx.OITopDataMap),
			len(ctx.QuantDataMap),
		)
	} else {
		logger.Errorf("序列化混沌上下文用于日志记录失败: %v", err)
	}

	// 保存权益快照（与 nofx 自动交易器行为保持一致）
	at.saveChaosEquitySnapshot(ctx)

	// ========================================
	// 第二步：执行混沌引擎（分解为多个子步骤）
	// ========================================

	// 子步骤 1：构建 AI 提示词
	systemPrompt := at.chaosEngine.BuildSystemPromptWithContext(ctx)
	userPrompt := at.chaosEngine.BuildUserPrompt(ctx)

	// 子步骤 2：调用 AI 模型
	aiCallStart := time.Now()
	aiResponse, err := at.mcpClient.CallWithMessages(systemPrompt, userPrompt)
	aiCallDuration := time.Since(aiCallStart)

	if err != nil {
		logger.Errorf("❌ 混沌引擎执行失败（LLM 调用）: %v", err)
		record.Success = false
		record.ErrorMessage = fmt.Sprintf("LLM 调用失败: %v", err)
		record.ExecutionLog = append(record.ExecutionLog, fmt.Sprintf("❌ LLM 错误: %v", err))
		if saveErr := at.saveDecision(record); saveErr != nil {
			logger.Errorf("保存错误决策记录失败: %v", saveErr)
		}
		return err
	}

	// 子步骤 3：解析和提取 AI 决策
	decisions, decisionJSON, extractErr := at.chaosEngine.ExtractDecisions(aiResponse)

	// 同时提取推理过程和思维链跟踪
	reasoning, _ := at.chaosEngine.ExtractReasoningJSON(aiResponse)
	cotTrace := at.chaosEngine.ExtractCoTTrace(aiResponse)

	// 构建初始结果对象，用于日志记录和错误处理
	chaosDecision := &chaos.DecisionResult{
		SystemPrompt:        systemPrompt,
		UserPrompt:          userPrompt,
		CoTTrace:            cotTrace,
		Decisions:           nil, // 将在验证后填充
		RawDecisions:        decisions,
		DecisionJSON:        decisionJSON,
		RawResponse:         aiResponse,
		Timestamp:           time.Now(),
		AIRequestDurationMs: aiCallDuration.Milliseconds(),
	}

	if extractErr != nil {
		logger.Errorf("❌ 混沌引擎执行失败（格式审核）: %v", extractErr)
		record.Success = false
		record.ErrorMessage = fmt.Sprintf("格式审核失败: %v", extractErr)
		record.ExecutionLog = append(record.ExecutionLog, fmt.Sprintf("❌ 格式错误: %v", extractErr))

		// 保存包含可用信息的记录
		record.SystemPrompt = chaosDecision.SystemPrompt
		record.InputPrompt = chaosDecision.UserPrompt
		record.CoTTrace = chaosDecision.CoTTrace
		record.RawResponse = chaosDecision.RawResponse
		record.AIRequestDurationMs = chaosDecision.AIRequestDurationMs
		record.DecisionJSON = chaosDecision.DecisionJSON

		if saveErr := at.saveDecision(record); saveErr != nil {
			logger.Errorf("保存错误决策记录失败: %v", saveErr)
		}
		return extractErr
	}

	// 子步骤 4：验证决策
	// 手动验证决策，以便优雅地处理错误（支持部分失败）
	var validatedDecisions []chaos.Decision
	var failedDecisions []store.DecisionAction
	riskConfig := ctx.Config.RiskControl

	for _, d := range decisions {
		// 验证单个决策，并计算仓位大小
		positionSizeUSD, err := at.chaosEngine.ValidateDecision(&d, reasoning, ctx.Account.TotalEquity, riskConfig)
		if err != nil {
			logger.Warnf("⚠️ %s 的混沌决策验证失败: %v", d.Symbol, err)

			// 创建失败的操作记录
			failedAction := store.DecisionAction{
				Symbol:    d.Symbol,
				Action:    "wait",
				Timestamp: time.Now().UTC(),
				Success:   false,
				Error:     err.Error(),
				Reasoning: fmt.Sprintf("验证失败: %v", err),
			}
			if d.TotalScore != nil {
				failedAction.Confidence = *d.TotalScore
			}
			failedDecisions = append(failedDecisions, failedAction)
			continue
		}

		// 存储验证通过的仓位大小
		d.PositionSizeUSD = &positionSizeUSD
		validatedDecisions = append(validatedDecisions, d)
	}

	// 将验证通过的决策赋值给结果对象
	chaosDecision.Decisions = validatedDecisions

	// 用 AI 决策结果填充记录
	record.SystemPrompt = chaosDecision.SystemPrompt
	record.InputPrompt = chaosDecision.UserPrompt
	record.CoTTrace = chaosDecision.CoTTrace
	record.RawResponse = chaosDecision.RawResponse
	record.AIRequestDurationMs = chaosDecision.AIRequestDurationMs
	record.DecisionJSON = chaosDecision.DecisionJSON

	// 子步骤 5：执行决策
	actionRecord, executionLogs := at.executeChaosDecision(chaosDecision, ctx)

	// 将验证失败的操作追加到记录和日志中
	if len(failedDecisions) > 0 {
		actionRecord = append(actionRecord, failedDecisions...)
		for _, fd := range failedDecisions {
			errMsg := fmt.Sprintf("❌ 验证失败 (%s): %s", fd.Symbol, fd.Error)
			executionLogs = append(executionLogs, errMsg)
		}
	}

	// 更新决策记录
	record.Decisions = actionRecord
	record.ExecutionLog = append(record.ExecutionLog, executionLogs...)

	// 保存最终的决策记录
	if err := at.saveDecision(record); err != nil {
		logger.Errorf("保存最终决策记录失败: %v", err)
	}

	logger.Infof("✅ 混沌交易周期 #%d 完成", at.callCount)
	return nil
}

// buildChaosContext 构建混沌交易所需的上下文信息
// 上下文包含：账户信息、持仓、候选币种、市场数据、排名数据、交易历史等
// 返回值：构建好的混沌上下文对象和可能的错误
func (at *AutoTrader) buildChaosContext() (*chaos.ChaosContext, error) {

	// ========================================
	// 第一步：获取账户信息（复用 auto_trader.go 中的逻辑）
	// ========================================
	balance, err := at.trader.GetBalance()
	if err != nil {
		return nil, fmt.Errorf("获取账户余额失败: %w", err)
	}

	// 解析账户余额数据
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
		// 如果没有 totalEquity，则通过钱包余额和未实现盈亏计算
		totalEquity = totalWalletBalance + totalUnrealizedProfit
	}

	// ========================================
	// 第二步：获取当前持仓
	// ========================================
	positions, err := at.trader.GetPositions()
	if err != nil {
		return nil, fmt.Errorf("获取持仓失败: %w", err)
	}

	// 将持仓转换为混沌格式
	var positionSnapshots []kernel.PositionInfo
	for _, pos := range positions {
		symbol := pos["symbol"].(string)
		side := pos["side"].(string)
		entryPrice := pos["entryPrice"].(float64)
		markPrice := pos["markPrice"].(float64)
		quantity := pos["positionAmt"].(float64)
		
		// 确保数量为正数
		if quantity < 0 {
			quantity = -quantity
		}
		// 跳过空仓位
		if quantity == 0 {
			continue
		}
		
		liquidationPrice := pos["liquidationPrice"].(float64)

		// 获取杠杆倍数，默认为 10
		leverage := 10
		if lev, ok := pos["leverage"].(float64); ok {
			leverage = int(lev)
		}

		// 计算盈亏百分比（使用与 auto_trader.go 相同的逻辑）
		marginUsed := (quantity * markPrice) / float64(leverage)
		pnlPct := 0.0
		unrealizedPnl := pos["unRealizedProfit"].(float64)
		if marginUsed > 0 {
			pnlPct = (unrealizedPnl / marginUsed) * 100
		}

		// 构建持仓快照对象
		positionSnapshots = append(positionSnapshots, kernel.PositionInfo{
			Symbol:           symbol,
			Side:             side,
			EntryPrice:       entryPrice,
			MarkPrice:        markPrice,
			Quantity:         quantity,
			Leverage:         leverage,
			UnrealizedPnL:    unrealizedPnl,
			UnrealizedPnLPct: pnlPct,
			PeakPnLPct:       pnlPct, // 暂时用当前盈亏近似
			MarginUsed:       marginUsed,
			LiquidationPrice: liquidationPrice,
			UpdateTime:       time.Now().UnixMilli(), // 近似值
		})
	}

	// ========================================
	// 第三步：准备候选币种列表
	// ========================================
	candidateCoins := []kernel.CandidateCoin{}
	existingCandidateMap := make(map[string]bool)

	// 3.1 从策略引擎获取候选币种
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

	// 3.2 将现有持仓合并到候选币种中
	// 如果我们持有某个币种的仓位，必须将其视为候选币种，以便 AI 可以管理它（平仓或调整）
	for _, pos := range positionSnapshots {
		if !existingCandidateMap[pos.Symbol] {
			logger.Infof("➕ 将持有的 %s 仓位合并到候选币种中进行管理", pos.Symbol)
			candidateCoins = append(candidateCoins, kernel.CandidateCoin{
				Symbol:  pos.Symbol,
				Sources: []string{"现有持仓"},
			})
			existingCandidateMap[pos.Symbol] = true
		}
	}

	// ========================================
	// 第四步：获取市场数据（混沌特定的数据获取）
	// ========================================
	marketDataMap := make(map[string]*market.Data)

	// 获取配置
	var chaosConfig *store.ChaosStrategyConfig
	if at.config.StrategyConfig.ChaosConfig != nil {
		chaosConfig = at.config.StrategyConfig.ChaosConfig
	} else {
		// 如果配置为空，使用默认配置
		chaosConfig = &store.ChaosStrategyConfig{}
	}
	indicatorsConfig := chaosConfig.Indicators

	// 如果配置缺失，使用默认值
	primaryTimeframe := "1h"
	klineCount := 100
	if indicatorsConfig.Klines.PrimaryTimeframe != "" {
		primaryTimeframe = indicatorsConfig.Klines.PrimaryTimeframe
	}
	if indicatorsConfig.Klines.PrimaryCount > 0 {
		klineCount = indicatorsConfig.Klines.PrimaryCount
	}

	// 确定需要获取数据的币种列表
	symbolsToFetch := make(map[string]bool)
	for _, p := range positionSnapshots {
		symbolsToFetch[p.Symbol] = true
	}
	for _, c := range candidateCoins {
		symbolsToFetch[c.Symbol] = true
	}

	// 获取每个币种的市场数据
	for symbol := range symbolsToFetch {
		data, err := market.GetWithTimeframes(symbol, indicatorsConfig.Klines.SelectedTimeframes, primaryTimeframe, klineCount)
		if err != nil {
			logger.Warnf("获取 %s 的市场数据失败: %v", symbol, err)
			continue
		}
		marketDataMap[symbol] = data
	}

	// ========================================
	// 第五步：获取持仓量排名数据（旧版，保留以保持兼容性）
	// ========================================
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

	// ========================================
	// 第六步：获取市场排名数据（新版 NoFxOS 数据）
	// 尊重混沌特定的指标开关，用于 OI/资金流/价格排名
	// ========================================
	var oiRankingData *nofxos.OIRankingData
	var netFlowRankingData *nofxos.NetFlowRankingData
	var priceRankingData *nofxos.PriceRankingData
	var quantDataMap map[string]*kernel.QuantData

	if at.strategyEngine != nil {
		indicators := indicatorsConfig

		// 获取量化数据
		if indicators.EnableQuantData {
			symbols := make([]string, 0, len(symbolsToFetch))
			for s := range symbolsToFetch {
				symbols = append(symbols, s)
			}
			logger.Infof("📊 [%s] 正在为 %d 个币种获取量化数据", at.name, len(symbols))
			quantDataMap = at.strategyEngine.FetchQuantDataBatch(symbols)
		}

		// 获取持仓量排名数据
		if indicators.EnableOIRanking {
			logger.Infof("📊 [%s] 正在获取持仓量排名数据（混沌）", at.name)
			oiRankingData = at.strategyEngine.FetchOIRankingData()
			if oiRankingData != nil {
				logger.Infof("📊 [%s] 持仓量排名数据已就绪（混沌）: %d 个高位, %d 个低位",
					at.name, len(oiRankingData.TopPositions), len(oiRankingData.LowPositions))
			}
		}

		// 获取资金流排名数据
		if indicators.EnableNetFlowRanking {
			logger.Infof("💰 [%s] 正在获取资金流排名数据（混沌）", at.name)
			netFlowRankingData = at.strategyEngine.FetchNetFlowRankingData()
			if netFlowRankingData != nil {
				logger.Infof("💰 [%s] 资金流排名数据已就绪（混沌）: 机构流入=%d, 机构流出=%d",
					at.name, len(netFlowRankingData.InstitutionFutureTop), len(netFlowRankingData.InstitutionFutureLow))
			}
		}

		// 获取价格排名数据
		if indicators.EnablePriceRanking {
			logger.Infof("📈 [%s] 正在获取价格排名数据（混沌）", at.name)
			priceRankingData = at.strategyEngine.FetchPriceRankingData()
			if priceRankingData != nil {
				logger.Infof("📈 [%s] 价格排名数据已就绪（混沌），共 %d 个时间段",
					at.name, len(priceRankingData.Durations))
			}
		}
	}

	// ========================================
	// 第六步半：获取交易历史（最近订单和统计数据）
	// ========================================
	var recentOrders []kernel.RecentOrder
	var tradingStats *kernel.TradingStats

	if at.store != nil {
		// 获取最近 10 笔已平仓交易，用于 AI 上下文
		recentTrades, err := at.store.Position().GetRecentTrades(at.id, 10)
		if err != nil {
			logger.Infof("⚠️ [%s] 获取最近交易失败: %v", at.name, err)
		} else {
			for _, trade := range recentTrades {
				// 将 Unix 时间戳转换为格式化字符串，便于 AI 阅读
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

		// 获取交易统计数据，用于 AI 上下文
		stats, err := at.store.Position().GetFullStats(at.id)
		if err != nil {
			logger.Infof("⚠️ [%s] 获取交易统计失败: %v", at.name, err)
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

	// ========================================
	// 第七步：组装上下文
	// ========================================
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

	// 计算总盈亏百分比和已使用保证金百分比
	if totalEquity > 0 {
		chaosCtx.Account.TotalPnLPct = (totalUnrealizedProfit / totalEquity) * 100
		chaosCtx.Account.MarginUsedPct = ((totalEquity - availableBalance) / totalEquity) * 100
	}

	return chaosCtx, nil
}

// saveChaosEquitySnapshot 为混沌模式保存权益快照（与 auto_trader 行为一致）
// 参数：ctx - 混沌上下文对象，包含账户信息
func (at *AutoTrader) saveChaosEquitySnapshot(ctx *chaos.ChaosContext) {
	// 如果存储对象或上下文为空，则直接返回
	if at.store == nil || ctx == nil {
		return
	}

	// 创建权益快照对象
	snapshot := &store.EquitySnapshot{
		TraderID:      at.id,
		Timestamp:     time.Now().UTC(),
		TotalEquity:   ctx.Account.TotalEquity,
		Balance:       ctx.Account.TotalEquity - ctx.Account.UnrealizedPnL,
		UnrealizedPnL: ctx.Account.UnrealizedPnL,
		PositionCount: ctx.Account.PositionCount,
		MarginUsedPct: ctx.Account.MarginUsedPct,
	}

	// 保存权益快照
	if err := at.store.Equity().Save(snapshot); err != nil {
		logger.Infof("⚠️ 保存权益快照失败: %v", err)
	}
}

// executeChaosDecision 执行混沌交易决策
// 参数：
//   result - 混沌决策结果对象，包含验证通过的决策
//   ctx - 混沌上下文对象
// 返回值：
//   操作记录数组和执行日志数组
func (at *AutoTrader) executeChaosDecision(result *chaos.DecisionResult, ctx *chaos.ChaosContext) ([]store.DecisionAction, []string) {

	logger.Infof("🤖 混沌 AI 决策: 生成了 %d 个决策", len(result.Decisions))

	var actionRecord []store.DecisionAction
	var executionLogs []string

	// 创建混沌执行器
	executor := chaos.NewChaosExecutor(
		at.trader, // AutoTrader.trader 实现了 TraderInterface
		at.store,
		at.id,
		at.exchange,
		at.exchangeID,
		ctx.Config,
	)
	// 执行决策
	actionRecord, executionLogs = executor.ExecuteDecisions(result.Decisions)

	return actionRecord, executionLogs
}
