package chaos

// Decision is a copy of the essential fields from kernel.Decision
// defined here to avoid circular dependencies and allow independent evolution
type Decision struct {
	Symbol          string
	Action          string
	Leverage        int
	PositionSizeUSD float64
	StopLoss        float64
	TakeProfit      float64
	EntryPrice      float64
	RiskR           float64
	RiskUSD         float64 // Optional: explicit risk amount in USD
	Confidence      int
	Reasoning       string
}
