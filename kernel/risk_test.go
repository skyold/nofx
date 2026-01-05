package kernel

import (
	"math"
	"testing"
)

func TestRiskRCalculation(t *testing.T) {
	// Setup mock data
	accountEquity := 10000.0

	// Case 1: 1R Risk, 1% Stop Loss Distance -> Should be ~1x Leverage
	d1 := Decision{
		Symbol:     "BTCUSDT",
		Action:     "open_long",
		RiskR:      1.0,
		EntryPrice: 50000,
		StopLoss:   49500, // 500 diff (1%)
		TakeProfit: 51500,
		Leverage:   5, // Exchange setting
	}

	// Expected:
	// Risk Amount = 10000 * 0.01 * 1.0 = 100 USD
	// Price Diff = 500
	// Quantity = 100 / 500 = 0.2 BTC
	// Position Size = 0.2 * 50000 = 10000 USD (1x Equity)

	err := validateDecision(&d1, accountEquity, 10, 10, 10.0, 10.0)
	if err != nil {
		t.Errorf("Validation failed: %v", err)
	}

	if math.Abs(d1.PositionSizeUSD-10000.0) > 0.01 {
		t.Errorf("Case 1 Failed: Expected 10000 position size, got %.2f", d1.PositionSizeUSD)
	}

	// Case 2: 1R Risk, 0.5% Stop Loss Distance -> Should be ~2x Leverage
	d2 := Decision{
		Symbol:     "BTCUSDT",
		Action:     "open_long",
		RiskR:      1.0,
		EntryPrice: 50000,
		StopLoss:   49750, // 250 diff (0.5%)
		TakeProfit: 51500,
		Leverage:   5,
	}

	// Expected:
	// Risk Amount = 100
	// Price Diff = 250
	// Quantity = 100 / 250 = 0.4 BTC
	// Position Size = 0.4 * 50000 = 20000 USD (2x Equity)

	err = validateDecision(&d2, accountEquity, 10, 10, 10.0, 10.0)
	if err != nil {
		t.Errorf("Validation failed: %v", err)
	}

	if math.Abs(d2.PositionSizeUSD-20000.0) > 0.01 {
		t.Errorf("Case 2 Failed: Expected 20000 position size, got %.2f", d2.PositionSizeUSD)
	}

	t.Logf("Case 1 Size: %.2f", d1.PositionSizeUSD)
	t.Logf("Case 2 Size: %.2f", d2.PositionSizeUSD)
}
