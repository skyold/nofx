package chaos

// Decision is a copy of the essential fields from kernel.Decision
// defined here to avoid circular dependencies and allow independent evolution
type Decision struct {
	Symbol     string  `json:"symbol"`
	Action     string  `json:"action"`
	Leverage   int     `json:"leverage,omitempty"`
	StopLoss   float64 `json:"stop_loss,omitempty"`
	TakeProfit float64 `json:"take_profit,omitempty"`
	EntryPrice float64 `json:"entry,omitempty"`
	RiskR      float64 `json:"risk_r,omitempty"`
	RiskUSD    float64 `json:"risk_usd,omitempty"`
	Confidence int     `json:"confidence,omitempty"`
}
