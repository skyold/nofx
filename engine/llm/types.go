package ai

import (
	"nofx/store"
)

type Decision struct {
	Symbol          string   `json:"symbol"`
	Action          string   `json:"action"`
	Leverage        *int     `json:"leverage,omitempty"`
	EntryPrice      *float64 `json:"entry,omitempty"`
	StopLoss        *float64 `json:"stop_loss,omitempty"`
	TakeProfit      *float64 `json:"take_profit,omitempty"`
	RiskR           *float64 `json:"risk_r,omitempty"`
	TotalScore      *int     `json:"total_score,omitempty"`
	PositionSizeUSD *float64 `json:"position_size_usd,omitempty"`
	Reasoning       string   `json:"reasoning,omitempty"`
}

type Context struct {
	CurrentTime     string                  `json:"current_time"`
	RuntimeMinutes  int                    `json:"runtime_minutes"`
	CallCount       int                    `json:"call_count"`
	Account         interface{}            `json:"account"`
	Positions       interface{}            `json:"positions"`
	CandidateCoins  interface{}            `json:"candidate_coins"`
	MarketDataMap   interface{}            `json:"market_data_map"`
	QuantDataMap    interface{}            `json:"quant_data_map"`
	OIRankingData   interface{}            `json:"oi_ranking_data"`
	TradingStats    interface{}            `json:"trading_stats"`
	RecentOrders   interface{}            `json:"recent_orders"`
	Config          *store.StrategyConfig  `json:"config,omitempty"`
}

type Reasoning struct {
	Raw             map[string]interface{}
	SystemRiskFlag  bool
	Opportunities   []Opportunity
}

type Opportunity struct {
	Symbol    string
	AuditPath string
}

type LLMConfig struct {
	EnginePrompt    string
	RiskControl     store.RiskControlConfig
	SystemVariant   string
	UserVersion     string
	Indicators      store.IndicatorConfig
}
