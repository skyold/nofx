package chaos

import (
	"context"
	"fmt"
	"nofx/kernel"
	"nofx/market"
	"nofx/provider/nofxos"
	"nofx/store"
	"nofx/trader/types"
	"time"
)

// ContextBuilder 负责构建 Chaos 引擎的上下文
// 这个结构体封装了所有构建上下文所需的依赖
type ContextBuilder struct {
	trader         types.Trader
	strategyEngine *kernel.StrategyEngine
	nofxosClient   *nofxos.Client
	config         *ChaosConfig
	store          *store.Store
	traderID       string
	startTime      time.Time
	callCount      int
}

// NewContextBuilder 创建上下文构建器
func NewContextBuilder(
	trader types.Trader,
	strategyEngine *kernel.StrategyEngine,
	nofxosClient *nofxos.Client,
	config *ChaosConfig,
	store *store.Store,
	traderID string,
	startTime time.Time,
	callCount int,
) *ContextBuilder {
	return &ContextBuilder{
		trader:         trader,
		strategyEngine: strategyEngine,
		nofxosClient:   nofxosClient,
		config:         config,
		store:          store,
		traderID:       traderID,
		startTime:      startTime,
		callCount:      callCount,
	}
}

// RuntimeInfo 运行时信息
type RuntimeInfo struct {
	CurrentTime    time.Time `json:"current_time"`
	RuntimeMinutes int       `json:"runtime_minutes"`
	CallCount      int       `json:"call_count"`
}

// BuildContext 构建完整的 Chaos 上下文
// 这是 Engine 接口的核心方法之一
func (cb *ContextBuilder) BuildContext(ctx context.Context, runtime interface{}) (*ChaosContext, error) {
	runtimeInfo, ok := runtime.(RuntimeInfo)
	if !ok {
		return nil, fmt.Errorf("invalid runtime info type")
	}

	// 第一步：获取账户信息
	accountInfo, totalEquity, availableBalance, totalUnrealizedProfit, err := cb.buildAccountInfo(ctx)
	if err != nil {
		return nil, fmt.Errorf("获取账户信息失败：%w", err)
	}

	// 第二步：获取持仓信息
	positionSnapshots, err := cb.buildPositions(ctx)
	if err != nil {
		return nil, fmt.Errorf("获取持仓信息失败：%w", err)
	}

	// 第三步：准备候选币种列表
	candidateCoins, err := cb.buildCandidateCoins(positionSnapshots)
	if err != nil {
		return nil, fmt.Errorf("准备候选币种失败：%w", err)
	}

	// 第四步：获取市场数据
	marketDataMap, err := cb.buildMarketData(candidateCoins)
	if err != nil {
		return nil, fmt.Errorf("获取市场数据失败：%w", err)
	}

	// 第五步：获取持仓量排名数据（旧版）
	oiTopMap := cb.buildOITopData()

	// 第六步：获取市场排名数据（新版）
	quantDataMap, oiRankingData, netFlowRankingData, priceRankingData := cb.buildRankingData(candidateCoins)

	// 第六步半：获取交易历史
	recentOrders, tradingStats := cb.buildTradingHistory()

	// 第七步：组装上下文
	chaosCtx := &ChaosContext{
		CurrentTime:        time.Now().Format("2006-01-02 15:04:05"),
		RuntimeMinutes:     runtimeInfo.RuntimeMinutes,
		CallCount:          runtimeInfo.CallCount,
		Timeframes:         cb.config.Indicators.Klines.SelectedTimeframes,
		Config:             cb.config,
		Account:            *accountInfo,
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

// buildAccountInfo 构建账户信息
func (cb *ContextBuilder) buildAccountInfo(ctx context.Context) (*kernel.AccountInfo, float64, float64, float64, error) {
	account, err := cb.trader.GetAccountInfo(ctx)
	if err != nil {
		return nil, 0, 0, 0, err
	}

	accountInfo := &kernel.AccountInfo{
		TotalEquity:      account.TotalEquity,
		AvailableBalance: account.AvailableBalance,
		UnrealizedPnL:    account.TotalUnrealizedPnL,
		MarginUsed:       account.MarginUsed,
		PositionCount:    0, // 将在后面设置
	}

	return accountInfo, account.TotalEquity, account.AvailableBalance, account.TotalUnrealizedPnL, nil
}

// buildPositions 构建持仓信息
func (cb *ContextBuilder) buildPositions(ctx context.Context) ([]kernel.PositionInfo, error) {
	positions, err := cb.trader.GetPositions(ctx)
	if err != nil {
		return nil, err
	}

	var positionSnapshots []kernel.PositionInfo
	for _, pos := range positions {
		quantity := pos.Quantity
		if quantity == 0 {
			continue
		}

		marginUsed := (quantity * pos.MarkPrice) / float64(pos.Leverage)
		unrealizedPnLPct := 0.0
		if marginUsed > 0 {
			unrealizedPnLPct = (pos.UnrealizedPnL / marginUsed) * 100
		}

		positionSnapshots = append(positionSnapshots, kernel.PositionInfo{
			Symbol:           pos.Symbol,
			Side:             pos.Side,
			EntryPrice:       pos.EntryPrice,
			MarkPrice:        pos.MarkPrice,
			Quantity:         quantity,
			Leverage:         pos.Leverage,
			UnrealizedPnL:    pos.UnrealizedPnL,
			UnrealizedPnLPct: unrealizedPnLPct,
			LiquidationPrice: pos.LiquidationPrice,
			MarginUsed:       marginUsed,
		})
	}

	return positionSnapshots, nil
}

// buildCandidateCoins 构建候选币种列表
func (cb *ContextBuilder) buildCandidateCoins(positions []kernel.PositionInfo) ([]kernel.CandidateCoin, error) {
	var candidateCoins []kernel.CandidateCoin
	existingCandidateMap := make(map[string]bool)

	// 从策略引擎获取候选币种
	// 使用 cb.config.Indicators 创建临时的 ChaosEngine
	tempConfig := &store.StrategyConfig{
		ChaosConfig: &store.ChaosStrategyConfig{
			Indicators: cb.config.Indicators,
		},
	}
	engine := NewChaosEngine(tempConfig)
	if engine != nil {
		candidates, err := engine.GetCandidateCoins()
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

	// 将现有持仓合并到候选币种中
	for _, pos := range positions {
		if !existingCandidateMap[pos.Symbol] {
			candidateCoins = append(candidateCoins, kernel.CandidateCoin{
				Symbol:  pos.Symbol,
				Sources: []string{"现有持仓"},
			})
			existingCandidateMap[pos.Symbol] = true
		}
	}

	return candidateCoins, nil
}

// buildMarketData 构建市场数据
func (cb *ContextBuilder) buildMarketData(candidateCoins []kernel.CandidateCoin) (map[string]*market.Data, error) {
	marketDataMap := make(map[string]*market.Data)

	indicatorsConfig := cb.config.Indicators
	primaryTimeframe := "1h"
	klineCount := 100
	if indicatorsConfig.Klines.PrimaryTimeframe != "" {
		primaryTimeframe = indicatorsConfig.Klines.PrimaryTimeframe
	}
	if indicatorsConfig.Klines.PrimaryCount > 0 {
		klineCount = indicatorsConfig.Klines.PrimaryCount
	}

	symbolsToFetch := make(map[string]bool)
	for _, c := range candidateCoins {
		symbolsToFetch[c.Symbol] = true
	}

	for symbol := range symbolsToFetch {
		data, err := market.GetWithTimeframes(symbol, indicatorsConfig.Klines.SelectedTimeframes, primaryTimeframe, klineCount)
		if err != nil {
			continue
		}
		marketDataMap[symbol] = data
	}

	return marketDataMap, nil
}

// buildOITopData 构建持仓量排名数据（旧版）
func (cb *ContextBuilder) buildOITopData() map[string]*kernel.OITopData {
	oiTopMap := make(map[string]*kernel.OITopData)

	// 检查是否启用了 OI Top 数据
	if cb.config.Indicators.EnableOI {
		apiKey := cb.config.Indicators.NofxOSAPIKey
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

	return oiTopMap
}

// buildRankingData 构建市场排名数据（新版）
func (cb *ContextBuilder) buildRankingData(candidateCoins []kernel.CandidateCoin) (map[string]*kernel.QuantData, *nofxos.OIRankingData, *nofxos.NetFlowRankingData, *nofxos.PriceRankingData) {
	var quantDataMap map[string]*kernel.QuantData
	var oiRankingData *nofxos.OIRankingData
	var netFlowRankingData *nofxos.NetFlowRankingData
	var priceRankingData *nofxos.PriceRankingData

	if cb.strategyEngine != nil {
		indicators := cb.config.Indicators

		// 获取量化数据
		if indicators.EnableQuantData {
			symbols := make([]string, 0, len(candidateCoins))
			for _, c := range candidateCoins {
				symbols = append(symbols, c.Symbol)
			}
			quantDataMap = cb.strategyEngine.FetchQuantDataBatch(symbols)
		}

		// 获取持仓量排名数据
		if indicators.EnableOIRanking {
			oiRankingData = cb.strategyEngine.FetchOIRankingData()
		}

		// 获取资金流排名数据
		if indicators.EnableNetFlowRanking {
			netFlowRankingData = cb.strategyEngine.FetchNetFlowRankingData()
		}

		// 获取价格排名数据
		if indicators.EnablePriceRanking {
			priceRankingData = cb.strategyEngine.FetchPriceRankingData()
		}
	}

	return quantDataMap, oiRankingData, netFlowRankingData, priceRankingData
}

// buildTradingHistory 构建交易历史
func (cb *ContextBuilder) buildTradingHistory() ([]kernel.RecentOrder, *kernel.TradingStats) {
	var recentOrders []kernel.RecentOrder
	var tradingStats *kernel.TradingStats

	if cb.store != nil {
		// 获取最近 10 笔已平仓交易
		recentTrades, err := cb.store.Position().GetRecentTrades(cb.traderID, 10)
		if err == nil {
			for _, trade := range recentTrades {
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

		// 获取交易统计数据
		stats, err := cb.store.Position().GetFullStats(cb.traderID)
		if err == nil && stats != nil && stats.TotalTrades > 0 {
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

	return recentOrders, tradingStats
}
