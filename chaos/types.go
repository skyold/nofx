package chaos

import (
	"nofx/kernel"
	"nofx/market"
	"nofx/provider/nofxos"
	"nofx/store"
	"time"
)

// Decision is a copy of the essential fields from kernel.Decision
// defined here to avoid circular dependencies and allow independent evolution
type Decision struct {
	Symbol     string   `json:"symbol"`
	Action     string   `json:"action"`
	Leverage   *int     `json:"leverage,omitempty"`
	EntryPrice *float64 `json:"entry,omitempty"`
	StopLoss   *float64 `json:"stop_loss,omitempty"`
	TakeProfit *float64 `json:"take_profit,omitempty"`
	RiskR      *float64 `json:"risk_r,omitempty"`
	TotalScore *int     `json:"total_score,omitempty"`
	// PositionSizeUSD is calculated by the system based on risk management rules, NOT output by AI.
	PositionSizeUSD *float64 `json:"position_size_usd,omitempty"`
}

// DecisionResult is the independent result structure for Chaos mode
type DecisionResult struct {
	SystemPrompt        string      `json:"system_prompt"`
	UserPrompt          string      `json:"user_prompt"`
	CoTTrace            string      `json:"cot_trace"`
	Decisions           []Decision  `json:"decisions"`
	RawDecisions        interface{} `json:"raw_decisions,omitempty"`
	DecisionJSON        string      `json:"decision_json"`
	RawResponse         string      `json:"raw_response"`
	Timestamp           time.Time   `json:"timestamp"`
	AIRequestDurationMs int64       `json:"ai_request_duration_ms,omitempty"`
}

// ChaosContext contains all information needed for Chaos mode execution
// independent from kernel.Context
type ChaosContext struct {
    CurrentTime     string                             `json:"current_time"`
	RuntimeMinutes  int                                `json:"runtime_minutes"`
	CallCount       int                                `json:"call_count"`
	Account         kernel.AccountInfo                 `json:"account"`
	Positions       []kernel.PositionInfo              `json:"positions"`
	CandidateCoins  []kernel.CandidateCoin             `json:"candidate_coins"`
	TradingStats    *kernel.TradingStats               `json:"trading_stats,omitempty"`
	RecentOrders    []kernel.RecentOrder               `json:"recent_orders,omitempty"`
	MarketDataMap   map[string]*market.Data            `json:"-"`
	MultiTFMarket   map[string]map[string]*market.Data `json:"-"`
	OITopDataMap    map[string]*kernel.OITopData       `json:"-"`
	QuantDataMap    map[string]*kernel.QuantData       `json:"-"`
	OIRankingData      *nofxos.OIRankingData           `json:"-"` // Market-wide OI ranking data
	NetFlowRankingData *nofxos.NetFlowRankingData      `json:"-"` // Market-wide fund flow ranking data
	PriceRankingData   *nofxos.PriceRankingData        `json:"-"` // Market-wide price gainers/losers
	Timeframes         []string                        `json:"-"`
	Config             *ChaosConfig                    `json:"config,omitempty"`

}

// ChaosConfig mirrors the store.ChaosStrategyConfig but can be extended
type ChaosConfig struct {
	ChaosPrompt         string
	RiskControl         store.RiskControlConfig
	SystemPromptVariant string
	UserPromptVersion   string
	Indicators          store.IndicatorConfig
}


