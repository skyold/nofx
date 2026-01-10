package kernel

import (
	"math"
	"testing"
)

// TestLeverageFallback tests automatic correction when leverage exceeds limit
func TestLeverageFallback(t *testing.T) {
	tests := []struct {
		name            string
		decision        Decision
		accountEquity   float64
		btcEthLeverage  int
		altcoinLeverage int
		wantLeverage    int // Expected leverage after correction
		wantError       bool
	}{
		{
			name: "Altcoin leverage exceeded - auto-correct to limit",
			decision: Decision{
				Symbol:          "SOLUSDT",
				Action:          "open_long",
				Leverage:        20, // Exceeds limit
				PositionSizeUSD: 100,
				StopLoss:        50,
				TakeProfit:      200,
			},
			accountEquity:   100,
			btcEthLeverage:  10,
			altcoinLeverage: 5, // Limit 5x
			wantLeverage:    5, // Should be corrected to 5
			wantError:       false,
		},
		{
			name: "BTC leverage exceeded - auto-correct to limit",
			decision: Decision{
				Symbol:          "BTCUSDT",
				Action:          "open_long",
				Leverage:        20, // Exceeds limit
				PositionSizeUSD: 1000,
				StopLoss:        90000,
				TakeProfit:      110000,
			},
			accountEquity:   100,
			btcEthLeverage:  10, // Limit 10x
			altcoinLeverage: 5,
			wantLeverage:    10, // Should be corrected to 10
			wantError:       false,
		},
		{
			name: "Leverage within limit - no correction",
			decision: Decision{
				Symbol:          "ETHUSDT",
				Action:          "open_short",
				Leverage:        5, // Not exceeded
				PositionSizeUSD: 500,
				StopLoss:        4000,
				TakeProfit:      3000,
			},
			accountEquity:   100,
			btcEthLeverage:  10,
			altcoinLeverage: 5,
			wantLeverage:    5, // Stays unchanged
			wantError:       false,
		},
		{
			name: "Leverage is 0 - should error",
			decision: Decision{
				Symbol:          "SOLUSDT",
				Action:          "open_long",
				Leverage:        0, // Invalid
				PositionSizeUSD: 100,
				StopLoss:        50,
				TakeProfit:      200,
			},
			accountEquity:   100,
			btcEthLeverage:  10,
			altcoinLeverage: 5,
			wantLeverage:    0,
			wantError:       true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Use default position value ratios for testing (10x for BTC/ETH, 1.5x for altcoins)
			err := validateDecision(&tt.decision, tt.accountEquity, tt.btcEthLeverage, tt.altcoinLeverage, 10.0, 1.5)

			// Check error status
			if (err != nil) != tt.wantError {
				t.Errorf("validateDecision() error = %v, wantError %v", err, tt.wantError)
				return
			}

			// If shouldn't error, check if leverage was correctly corrected
			if !tt.wantError && tt.decision.Leverage != tt.wantLeverage {
				t.Errorf("Leverage not corrected: got %d, want %d", tt.decision.Leverage, tt.wantLeverage)
			}
		})
	}
}

func TestValidateDecisionsSliceUpdate(t *testing.T) {
	decisions := []Decision{
		{
			Symbol:     "BTCUSDT",
			Action:     "open_long",
			RiskR:      1.0,
			EntryPrice: 100,
			StopLoss:   90,
			TakeProfit: 130,
			Leverage:   5,
		},
	}

	err := validateDecisions(decisions, 10000, 10, 5, 10.0, 1.5)
	if err != nil {
		t.Fatalf("validateDecisions failed: %v", err)
	}

	if decisions[0].PositionSizeUSD <= 0 {
		t.Fatalf("PositionSizeUSD was not updated in slice, got %.8f", decisions[0].PositionSizeUSD)
	}

	if math.Abs(decisions[0].PositionSizeUSD-1000) > 1e-6 {
		t.Fatalf("PositionSizeUSD mismatch, got %.8f, want 1000", decisions[0].PositionSizeUSD)
	}
}

// TestChaosPositionSizeClamping tests automatic clamping when position size exceeds limit in Chaos mode
func TestChaosPositionSizeClamping(t *testing.T) {
	// Setup scenario matching the user error
	// decision[symbol=ETHUSDT action=open_long lev=5 entry=3092.16000000 sl=3086.00000000 tp=3120.00000000 risk_r=0.25 pos_usd=1324.28 conf=75]: position size 1324.28 exceeds max allowed 1055.26 for ETHUSDT

	accountEquity := 211.052
	btcEthPosRatio := 1.0                        // Reduced from 5.0 to force clamping with lower base risk
	maxAllowed := accountEquity * btcEthPosRatio // 211.052

	decision := Decision{
		Symbol:     "ETHUSDT",
		Action:     "open_long",
		Leverage:   5,
		EntryPrice: 3092.16,
		StopLoss:   3086.00,
		TakeProfit: 3120.00,
		RiskR:      1.0, // Increased to 1.0 (1% risk) to generate enough size
		Confidence: 75,
	}

	// With baseRiskPercent = 0.01:
	// RiskAmount = 211.052 * 0.01 * 1.0 = 2.11052
	// RiskPerUnit = 3092.16 - 3086.00 = 6.16
	// Qty = 2.11052 / 6.16 = 0.3426
	// PosSize = 0.3426 * 3092.16 = 1059.43
	// MaxAllowed = 211.052
	// Should clamp.

	// Should not return error, but clamp position size
	err := validateDecision(&decision, accountEquity, 5, 5, btcEthPosRatio, 1.0)
	if err != nil {
		t.Fatalf("validateDecision failed: %v", err)
	}

	// Check if position size is clamped
	if decision.PositionSizeUSD > maxAllowed+0.01 { // Allow tiny float error
		t.Errorf("PositionSizeUSD not clamped: got %.2f, want <= %.2f", decision.PositionSizeUSD, maxAllowed)
	}

	if math.Abs(decision.PositionSizeUSD-maxAllowed) > 0.01 {
		t.Errorf("PositionSizeUSD should be clamped to maxAllowed: got %.2f, want %.2f", decision.PositionSizeUSD, maxAllowed)
	}
}

// contains checks if string contains substring (helper function)
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(substr) == 0 ||
		(len(s) > 0 && len(substr) > 0 && stringContains(s, substr)))
}

func stringContains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
