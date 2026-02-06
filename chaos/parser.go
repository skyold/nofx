package chaos

import (
	"encoding/json"
	"fmt"
	"math"
	"regexp"
	"strings"

	"nofx/logger"
)

var (
	// Safe regex: precisely match ```json code blocks
	reJSONFence      = regexp.MustCompile(`(?is)` + "```json\\s*(\\[\\s*\\{.*?\\}\\s*\\])\\s*```")
	reJSONArray      = regexp.MustCompile(`(?is)\[\s*\{.*?\}\s*\]`)
	reArrayHead      = regexp.MustCompile(`^\[\s*\{`)
	reArrayOpenSpace = regexp.MustCompile(`^\[\s+\{`)
	reInvisibleRunes = regexp.MustCompile("[\u200B\u200C\u200D\uFEFF]")
	reDecisionTag    = regexp.MustCompile(`(?s)<decision>(.*?)</decision>`)
	// Support both <execution_reasoning> (Standard) and <reasoning> (Legacy/Fallback)
	reReasoningTag = regexp.MustCompile(`(?s)<(execution_)?reasoning>(.*?)</(execution_)?reasoning>`)
)

// RawDecision represents the structure expected from AI JSON
type RawDecision struct {
	Symbol     string      `json:"symbol"`
	Action     string      `json:"action"`
	Leverage   interface{} `json:"leverage"`    // Support int or string (e.g., "10x")
	EntryPrice interface{} `json:"entry"`       // Support float or string
	StopLoss   interface{} `json:"stop_loss"`   // Support float or string
	TakeProfit interface{} `json:"take_profit"` // Support float or string
	RiskR      interface{} `json:"risk_r"`      // Support float or string
	TotalScore interface{} `json:"total_score"` // Support float or string
}

// Reasoning represents the structured reasoning output from AI
// Now flexible to support different strategy structures
type Reasoning struct {
	Raw            map[string]interface{}
	SystemRiskFlag bool
	Opportunities  []Opportunity
}

// Opportunity for validation (Audit Path)
type Opportunity struct {
	Symbol    string
	AuditPath string
}

func extractDecisions(response string) ([]Decision, error) {
	s := removeInvisibleRunes(response)
	s = strings.TrimSpace(s)
	s = fixMissingQuotes(s)

	var jsonPart string
	if match := reDecisionTag.FindStringSubmatch(s); match != nil && len(match) > 1 {
		jsonPart = strings.TrimSpace(match[1])
		logger.Infof("✓ [Format Audit] Extracted JSON using <decision> tag")
	} else {
		// Fallback: try to find JSON array directly if tag is missing
		jsonPart = s
	}

	jsonPart = fixMissingQuotes(jsonPart)

	// Try to find JSON array
	var jsonContent string
	if m := reJSONFence.FindStringSubmatch(jsonPart); m != nil && len(m) > 1 {
		jsonContent = strings.TrimSpace(m[1])
	} else {
		jsonContent = strings.TrimSpace(reJSONArray.FindString(jsonPart))
	}

	if jsonContent == "" {
		logger.Warnf("⚠️  [Format Audit] AI didn't output JSON decision, entering safe wait mode")
		fallbackDecision := Decision{
			Symbol: "ALL",
			Action: "wait",
		}
		return []Decision{fallbackDecision}, nil
	}

	jsonContent = compactArrayOpen(jsonContent)
	jsonContent = fixMissingQuotes(jsonContent)

	if err := validateJSONFormat(jsonContent); err != nil {
		return nil, fmt.Errorf("JSON format validation failed: %w\nJSON content: %s\nFull response:\n%s", err, jsonContent, response)
	}

	var rawDecisions []RawDecision
	if err := json.Unmarshal([]byte(jsonContent), &rawDecisions); err != nil {
		return nil, fmt.Errorf("JSON parsing failed: %w\nJSON content: %s", err, jsonContent)
	}

	return convertDecisions(rawDecisions), nil
}

func extractReasoningJSON(response string) (*Reasoning, error) {
	s := removeInvisibleRunes(response)
	var jsonContent string

	if match := reReasoningTag.FindStringSubmatch(s); match != nil {
		// match[0] is full string
		// match[1] is "execution_" or "" (prefix)
		// match[2] is content
		// match[3] is "execution_" or "" (suffix)
		if len(match) > 2 {
			jsonContent = strings.TrimSpace(match[2])
		}
	} else {
		// Fallback: Try to find JSON object directly if tag is missing
		// This handles cases where prompt returns raw JSON without XML tags
		firstBrace := strings.Index(s, "{")
		lastBrace := strings.LastIndex(s, "}")
		if firstBrace >= 0 && lastBrace > firstBrace {
			jsonContent = s[firstBrace : lastBrace+1]
		} else {
			return nil, fmt.Errorf("<execution_reasoning> tag not found and no valid JSON object detected")
		}
	}

	jsonContent = fixMissingQuotes(jsonContent)

	// Parse into generic map
	var rawMap map[string]interface{}
	if err := json.Unmarshal([]byte(jsonContent), &rawMap); err != nil {
		return nil, fmt.Errorf("reasoning JSON parsing failed: %w", err)
	}

	reasoning := &Reasoning{
		Raw: rawMap,
	}

	// 1. Extract SystemRiskFlag (Flexible Location)
	// Try root level
	if v, ok := rawMap["system_risk_flag"]; ok {
		if boolVal, ok := v.(bool); ok {
			reasoning.SystemRiskFlag = boolVal
		}
	} else if v, ok := rawMap["market_context"]; ok {
		// Try nested in market_context (Standard Prompt style)
		if mcMap, ok := v.(map[string]interface{}); ok {
			if flag, ok := mcMap["system_risk_flag"]; ok {
				if boolVal, ok := flag.(bool); ok {
					reasoning.SystemRiskFlag = boolVal
				}
			}
		}
	}

	// 2. Extract Opportunities for Audit Path (Flexible)
	// Only if "opportunities" key exists and is array
	if v, ok := rawMap["opportunities"]; ok {
		if oppsArray, ok := v.([]interface{}); ok {
			for _, item := range oppsArray {
				if oppMap, ok := item.(map[string]interface{}); ok {
					opp := Opportunity{}
					if s, ok := oppMap["symbol"].(string); ok {
						opp.Symbol = s
					}
					if ap, ok := oppMap["audit_path"].(string); ok {
						opp.AuditPath = ap
					}
					reasoning.Opportunities = append(reasoning.Opportunities, opp)
				}
			}
		}
	}

	return reasoning, nil
}

func convertDecisions(raw []RawDecision) []Decision {
	decisions := make([]Decision, len(raw))
	for i, r := range raw {
		d := Decision{
			Symbol: r.Symbol,
			Action: r.Action,
		}

		if r.Leverage != nil {
			if val, ok := toInt(r.Leverage); ok {
				d.Leverage = &val
			}
		}

		if r.EntryPrice != nil {
			if val, ok := toFloat(r.EntryPrice); ok {
				d.EntryPrice = &val
			}
		}

		if r.StopLoss != nil {
			if val, ok := toFloat(r.StopLoss); ok {
				d.StopLoss = &val
			}
		}

		if r.TakeProfit != nil {
			if val, ok := toFloat(r.TakeProfit); ok {
				d.TakeProfit = &val
			}
		}

		if r.RiskR != nil {
			if val, ok := toFloat(r.RiskR); ok {
				d.RiskR = &val
			}
		}

		if r.TotalScore != nil {
			if val, ok := toFloat(r.TotalScore); ok {
				score := int(math.Round(val))
				d.TotalScore = &score
			}
		}

		decisions[i] = d
	}
	return decisions
}

// Helper functions for type conversion
func toFloat(v interface{}) (float64, bool) {
	switch val := v.(type) {
	case float64:
		return val, true
	case int:
		return float64(val), true
	case string:
		// Try to parse string as float
		// Remove quotes if present (though JSON unmarshal usually handles this)
		// Remove non-numeric characters except dot and minus
		// For simplicity, let's use a basic regex or just simple parsing
		// Here we assume standard number format in string
		var f float64
		if _, err := fmt.Sscanf(val, "%f", &f); err == nil {
			return f, true
		}
	}
	return 0, false
}

func toInt(v interface{}) (int, bool) {
	switch val := v.(type) {
	case float64:
		return int(val), true
	case int:
		return val, true
	case string:
		// Handle "10x" or "10"
		s := strings.TrimSuffix(strings.ToLower(val), "x")
		var i int
		if _, err := fmt.Sscanf(s, "%d", &i); err == nil {
			return i, true
		}
	}
	return 0, false
}

func fixMissingQuotes(jsonStr string) string {
	jsonStr = strings.ReplaceAll(jsonStr, "\u201c", "\"")
	jsonStr = strings.ReplaceAll(jsonStr, "\u201d", "\"")
	jsonStr = strings.ReplaceAll(jsonStr, "\u2018", "'")
	jsonStr = strings.ReplaceAll(jsonStr, "\u2019", "'")

	jsonStr = strings.ReplaceAll(jsonStr, "［", "[")
	jsonStr = strings.ReplaceAll(jsonStr, "］", "]")
	jsonStr = strings.ReplaceAll(jsonStr, "｛", "{")
	jsonStr = strings.ReplaceAll(jsonStr, "｝", "}")
	jsonStr = strings.ReplaceAll(jsonStr, "：", ":")
	jsonStr = strings.ReplaceAll(jsonStr, "，", ",")

	jsonStr = strings.ReplaceAll(jsonStr, "【", "[")
	jsonStr = strings.ReplaceAll(jsonStr, "】", "]")
	jsonStr = strings.ReplaceAll(jsonStr, "〔", "[")
	jsonStr = strings.ReplaceAll(jsonStr, "〕", "]")
	jsonStr = strings.ReplaceAll(jsonStr, "、", ",")

	jsonStr = strings.ReplaceAll(jsonStr, "　", " ")

	return jsonStr
}

func validateJSONFormat(jsonStr string) error {
	trimmed := strings.TrimSpace(jsonStr)

	// Allow JSON that starts with [ but check for basic validity
	if !strings.HasPrefix(trimmed, "[") {
		return fmt.Errorf("JSON must start with [, actual: %s", trimmed[:min(20, len(trimmed))])
	}

	// Basic check for object start
	if !strings.Contains(trimmed, "{") {
		return fmt.Errorf("JSON array must contain objects {}, actual content: %s", trimmed[:min(50, len(trimmed))])
	}

	if strings.Contains(jsonStr, "~") {
		return fmt.Errorf("JSON cannot contain range symbol ~, all numbers must be precise single values")
	}

	// Relaxed thousand separator check: only if surrounded by digits
	// Regex would be better but keeping simple loop for now
	for i := 0; i < len(jsonStr)-4; i++ {
		if jsonStr[i] >= '0' && jsonStr[i] <= '9' &&
			jsonStr[i+1] == ',' &&
			jsonStr[i+2] >= '0' && jsonStr[i+2] <= '9' &&
			jsonStr[i+3] >= '0' && jsonStr[i+3] <= '9' &&
			jsonStr[i+4] >= '0' && jsonStr[i+4] <= '9' {
			return fmt.Errorf("JSON numbers cannot contain thousand separator comma, found: %s", jsonStr[i:min(i+10, len(jsonStr))])
		}
	}

	return nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func removeInvisibleRunes(s string) string {
	return reInvisibleRunes.ReplaceAllString(s, "")
}

func compactArrayOpen(s string) string {
	return reArrayOpenSpace.ReplaceAllString(strings.TrimSpace(s), "[{")
}
