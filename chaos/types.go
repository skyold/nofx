package chaos

import (
	"nofx/market"
	"nofx/store"
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
}

// ChaosContext contains all information needed for Chaos mode execution
// independent from kernel.Context
type ChaosContext struct {
	// Basic info
	CurrentTime    string
	RuntimeMinutes int
	CallCount      int

	// Configuration
	Config *ChaosConfig

	// Account & Positions
	Account   AccountSnapshot
	Positions []PositionSnapshot

	// Market Data
	CandidateCoins []CandidateCoin
	MarketDataMap  map[string]*market.Data
	OITopDataMap   map[string]*OITopData

	// Chaos specific fields
	InjectedAnomalies []string // List of anomalies injected into this context
}

// ChaosConfig mirrors the store.ChaosStrategyConfig but can be extended
type ChaosConfig struct {
	ChaosPrompt        string
	RiskControl        store.RiskControlConfig
	PromptVariant      string
	FaultInjectionRate float64
	DataNoiseLevel     float64
	StressTestMode     bool
	Indicators         store.IndicatorConfig
}

// AccountSnapshot independent account info
type AccountSnapshot struct {
	TotalEquity      float64 `json:"total_equity"`
	AvailableBalance float64 `json:"available_balance"`
	UnrealizedPnL    float64 `json:"unrealized_pnl"`
	TotalPnLPct      float64 `json:"total_pnl_pct"`
	MarginUsedPct    float64 `json:"margin_used_pct"`
	PositionCount    int     `json:"position_count"`
}

// PositionSnapshot independent position info
type PositionSnapshot struct {
	Symbol           string  `json:"symbol"`
	Side             string  `json:"side"`
	EntryPrice       float64 `json:"entry_price"`
	MarkPrice        float64 `json:"mark_price"`
	Quantity         float64 `json:"quantity"`
	Leverage         int     `json:"leverage"`
	UnrealizedPnLPct float64 `json:"unrealized_pnl_pct"`
	LiquidationPrice float64 `json:"liquidation_price"`
	UpdateTime       int64   `json:"update_time"`
}

// CandidateCoin independent candidate coin
type CandidateCoin struct {
	Symbol  string   `json:"symbol"`
	Sources []string `json:"sources"`
}

// OITopData independent OI Top data
type OITopData struct {
	Rank              int     `json:"rank"`
	OIDeltaPercent    float64 `json:"oi_delta_percent"`
	OIDeltaValue      float64 `json:"oi_delta_value"`
	PriceDeltaPercent float64 `json:"price_delta_percent"`
}
