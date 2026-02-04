package chaos

import (
	"strings"
	"testing"
	"time"

	"nofx/kernel"
	"nofx/market"
	"nofx/store"
)

func TestBuildUserPrompt(t *testing.T) {
	// Setup mock context
	ctx := &kernel.Context{
		CurrentTime:    time.Now().Format(time.RFC3339),
		CallCount:      1,
		RuntimeMinutes: 5,
		Account: kernel.AccountInfo{
			TotalEquity:      1000.0,
			AvailableBalance: 1000.0,
			TotalPnLPct:      0.0,
			MarginUsedPct:    0.0,
			PositionCount:    0,
		},
		MarketDataMap:  make(map[string]*market.Data),
		CandidateCoins: []kernel.CandidateCoin{},
	}

	// Create engine
	config := store.GetDefaultStrategyConfig("en")
	engine := NewChaosEngine(&config)

	// Test BuildUserPrompt
	prompt := engine.BuildUserPrompt(ctx)

	// Verify essential components
	expectedParts := []string{
		"Time:",
		"Account:",
		"Candidate Coins",
	}

	for _, part := range expectedParts {
		if !strings.Contains(prompt, part) {
			t.Errorf("Prompt missing expected part: %s", part)
		}
	}
}
