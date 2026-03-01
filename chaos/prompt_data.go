// =============================================================================
// Chaos Trading System - Prompt Data Structures
// =============================================================================
//
// 定义用于 Prompt 构建的数据结构
// 这些结构体是 Builder 的输出，也是 Formatter 的输入
// =============================================================================

package chaos

import (
	"nofx/market"
)

// ============================================================================
// 核心数据结构
// ============================================================================

// MarketPromptData 市场提示词数据
// 包含所有可能需要的市场相关信息，由 Builder 填充，由 Formatter 格式化
type MarketPromptData struct {
	// 基础价格数据
	Symbol        string
	CurrentPrice  float64
	LocalSupport  float64
	LocalSupportTime int64
	DailyLow      float64
	FundingRate   float64

	// 时间周期数据
	Timeframes map[string]*TimeframeData // 多时间周期数据

	// 持仓量数据
	OpenInterest *OpenInterestData

	// 机构市场状态分类器数据（可选，由 EnhancedBuilder 填充）
	InstitutionalRegime *InstitutionalRegimeSignal

	// LLM 战术简报（可选，由 EnhancedBuilder 填充）
	LLMBriefing string

	// 物理结构锚点（摆动点等）
	SwingFeatures *TechnicalFeatures

	// 主要区间边界
	MajorSupport    float64
	MajorResistance float64
	SupportTests    int
	ResistanceTests int

	// 动态参考
	DynamicReferences []string
}

// TimeframeData 单个时间周期的数据
type TimeframeData struct {
	Timeframe   string
	Klines      []market.KlineBar
	EMA20Values []float64
	EMA50Values []float64
	MACDValues  []float64
	RSI7Values  []float64
	RSI14Values []float64
	ATR14Values []float64
	BOLLUpper   []float64
	BOLLMiddle  []float64
	BOLLLower   []float64
}

// OpenInterestData 持仓量数据
type OpenInterestData struct {
	Latest  float64
	Average float64
}

// ============================================================================
// 账户和交易数据
// ============================================================================

// AccountPromptData 账户提示词数据
type AccountPromptData struct {
	TotalEquity        float64
	AvailableBalance   float64
	TotalPnLPct        float64
	MarginUsedPct      float64
	PositionCount      int
	BalancePercent     float64
}

// PositionPromptData 持仓提示词数据
type PositionPromptData struct {
	Symbol           string
	Side             string
	EntryPrice       float64
	MarkPrice        float64
	Quantity         float64
	PositionValue    float64
	UnrealizedPnLPct float64
	UnrealizedPnL    float64
	PeakPnLPct       float64
	Leverage         int
	MarginUsed       float64
	LiquidationPrice float64
	HoldingDuration  string
	MarketData       *MarketPromptData // 持仓对应的市场数据
	QuantData        *QuantPromptData  // 量化数据（可选）
}

// QuantPromptData 量化数据
type QuantPromptData struct {
	Symbol      string
	PriceChange map[string]float64 // 时间周期 -> 价格变化
	Netflow     *NetflowData
	OI          map[string]*OIData // 交易所 -> OI 数据
}

// NetflowData 资金流向数据
type NetflowData struct {
	Institution *FlowData
	Personal    *FlowData
}

// FlowData 流向数据
type FlowData struct {
	Future map[string]float64 // 时间周期 -> 流向金额
	Spot   map[string]float64 // 时间周期 -> 流向金额
}

// OIData 持仓量变化数据
type OIData struct {
	Delta       map[string]float64 // 时间周期 -> OI 变化值
	DeltaPercent map[string]float64 // 时间周期 -> OI 变化百分比
}

// ============================================================================
// 排行榜数据
// ============================================================================

// RankingPromptData 排行榜提示词数据
type RankingPromptData struct {
	OIIncrease1h  []RankingItem
	OIDecrease1h  []RankingItem
	FundInflow1h  []RankingItem
	FundOutflow1h []RankingItem
	TopGainers1h  []RankingItem
	TopLosers1h   []RankingItem
}

// RankingItem 排行榜项
type RankingItem struct {
	Symbol    string
	OIChange  float64
	OIPct     float64
	PricePct  float64
	Inflow    float64
	Outflow   float64
	Price     float64
	FundFlow  float64
	ChangePct float64
}

// ============================================================================
// 全局上下文数据
// ============================================================================

// GlobalContextData 全局上下文数据（BTC 行情）
type GlobalContextData struct {
	Symbol        string
	CurrentPrice  float64
	PriceChange1h float64
	PriceChange4h float64
	CurrentMACD   float64
	CurrentRSI7   float64
}

// ============================================================================
// 交易统计
// ============================================================================

// TradingStatsData 交易统计数据
type TradingStatsData struct {
	TotalTrades     int
	ProfitFactor    float64
	SharpeRatio     float64
	WinLossRatio    float64
	TotalPnL        float64
	AvgWin          float64
	AvgLoss         float64
	MaxDrawdownPct  float64
	PerformanceNote string // 表现评价
}

// RecentTradeData 最近交易数据
type RecentTradeData struct {
	Symbol        string
	Side          string
	EntryPrice    float64
	ExitPrice     float64
	ResultStr     string // "Profit" or "Loss"
	RealizedPnL   float64
	PnLPct        float64
	EntryTime     string
	ExitTime      string
	HoldDuration  string
}
