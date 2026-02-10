package mcp

import (
	"encoding/json"
	"fmt"
	"math/rand"
	"regexp"
	"time"
)

// MockClient simulates an LLM for testing
type MockClient struct {
}

// NewMockClient creates a new MockClient
func NewMockClient() *MockClient {
	return &MockClient{}
}

func (c *MockClient) SetAPIKey(apiKey string, customURL string, customModel string) {
	// No-op
}

func (c *MockClient) SetTimeout(timeout time.Duration) {
	// No-op
}

func (c *MockClient) CallWithMessages(systemPrompt, userPrompt string) (string, error) {
	// 1. Parse symbols from prompt
	// Look for pattern like "=== BTCUSDT Market Data ===" or "### 1. BTCUSDT"
	re := regexp.MustCompile(`=== (\w+) Market Data ===`)
	matches := re.FindAllStringSubmatch(userPrompt, -1)

	var symbols []string
	seen := make(map[string]bool)
	for _, m := range matches {
		if len(m) > 1 {
			symbol := m[1]
			if !seen[symbol] {
				symbols = append(symbols, symbol)
				seen[symbol] = true
			}
		}
	}

	// If no symbols found, try candidate coins section format
	if len(symbols) == 0 {
		reCand := regexp.MustCompile(`### \d+\. (\w+)`)
		matchesCand := reCand.FindAllStringSubmatch(userPrompt, -1)
		for _, m := range matchesCand {
			if len(m) > 1 {
				symbol := m[1]
				if !seen[symbol] {
					symbols = append(symbols, symbol)
					seen[symbol] = true
				}
			}
		}
	}

	// 2. Generate decisions
	var opportunities []map[string]interface{}
	var decisions []map[string]interface{}

	// Random source
	r := rand.New(rand.NewSource(time.Now().UnixNano()))

	for i, symbol := range symbols {
		// Random action: 10% buy, 10% sell, 80% wait
		actionRoll := r.Float64()
		var action string
		var direction string

		// Set realistic prices based on symbol
		var entry float64 = 100.0 // Default
		if symbol == "BTCUSDT" {
			entry = 65000.0 + r.Float64()*1000.0
		} else if symbol == "ETHUSDT" {
			entry = 2500.0 + r.Float64()*100.0
		} else if symbol == "SOLUSDT" {
			entry = 150.0 + r.Float64()*10.0
		} else {
			entry = 10.0 + r.Float64()*5.0
		}

		var stopLoss float64
		var takeProfit float64
		var leverage = 5
		var confidence = 75.0 + r.Float64()*20.0

		// Mock reasoning
		logicSummary := fmt.Sprintf("Price action for %s shows potential setup.", symbol)

		if actionRoll < 0.1 {
			action = "open_long"
			direction = "LONG"
			stopLoss = entry * 0.98
			takeProfit = entry * 1.05
			logicSummary = "Price bouncing off support with strong volume."
		} else if actionRoll < 0.2 {
			action = "open_short"
			direction = "SHORT"
			stopLoss = entry * 1.02
			takeProfit = entry * 0.95
			logicSummary = "Price rejected at resistance, bearish divergence."
		} else {
			action = "wait"
			direction = "NEUTRAL"
			logicSummary = "Market is choppy, waiting for clearer signal."
		}

		// Opportunity object (for execution_reasoning)
		opp := map[string]interface{}{
			"symbol":        symbol,
			"regime":        "RANGE_REVERSION",
			"logic_summary": logicSummary,
			"direction":     direction,
			"price_levels": map[string]interface{}{
				"entry":       entry,
				"stop_loss":   stopLoss,
				"take_profit": takeProfit,
				"risk_reward": 2.5,
			},
			"scoring": map[string]interface{}{
				"Trend":         r.Intn(25),
				"Structure":     r.Intn(25),
				"Participation": r.Intn(25),
				"Timing":        r.Intn(25),
			},
			"total_score":   int(confidence),
			"veto_flags":    []string{},
			"audit_path":    "Score 92 -> Base 1.0R -> Final 1.0R",
			"priority_rank": i + 1,
		}
		opportunities = append(opportunities, opp)

		// Decision object (for final JSON array)
		decision := map[string]interface{}{
			"symbol": symbol,
			"action": action,
		}
		if action != "wait" && action != "hold" {
			decision["leverage"] = leverage
			decision["entry"] = entry
			decision["stop_loss"] = stopLoss
			decision["take_profit"] = takeProfit
			decision["risk_r"] = 1.0
			decision["total_score"] = int(confidence)
		}
		decisions = append(decisions, decision)
	}

	// 3. Construct response
	executionReasoning := map[string]interface{}{
		"system_risk_flag": false,
		"prompt_meta": map[string]string{
			"prompt_name": "Virtual LLM Mock Prompt",
		},
		"market_context": map[string]interface{}{
			"market_regime":          "CONSOLIDATING",
			"volatility_profile":     "STABLE",
			"data_integrity_warning": []string{},
		},
		"opportunities":          opportunities,
		"position_authorization": []map[string]interface{}{},
	}

	reasoningJSON, _ := json.MarshalIndent(executionReasoning, "", "  ")
	decisionsJSON, _ := json.MarshalIndent(decisions, "", "  ")

	response := fmt.Sprintf("<execution_reasoning>\n%s\n</execution_reasoning>\n<decision>\n%s\n</decision>", string(reasoningJSON), string(decisionsJSON))
	return response, nil
}

func (c *MockClient) CallWithRequest(req *Request) (string, error) {
	// Delegate to CallWithMessages for simplicity
	var systemPrompt string
	var userPrompt string
	for _, msg := range req.Messages {
		if msg.Role == "system" {
			systemPrompt += msg.Content + "\n"
		} else if msg.Role == "user" {
			userPrompt += msg.Content + "\n"
		}
	}
	return c.CallWithMessages(systemPrompt, userPrompt)
}
