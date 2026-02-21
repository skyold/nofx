package chaos

import (
	"fmt"
	"regexp"
	"strings"

	"nofx/logger"
	"nofx/store"
)

// Manager handles Chaos trading mode specific logic
type Manager struct{}

// NewManager creates a new Chaos Manager
func NewManager() *Manager {
	return &Manager{}
}

// IsChaosMode checks if the current prompt indicates Chaos mode
// It supports both legacy string matching and new prompt_meta JSON check
func (m *Manager) IsChaosMode(customPrompt string) bool {
	if customPrompt == "" {
		return false
	}

	// 1. Legacy Check
	cp := strings.TrimSpace(customPrompt)
	chaosTags := []string{"这是一个Chaos策略", "Chaos Trader"}
	for _, tag := range chaosTags {
		if strings.HasPrefix(cp, tag) || strings.Contains(cp, tag) {
			return true
		}
	}

	// 2. New Metadata Check (prompt_meta)
	// Regex to find "prompt_meta": { ... "type": "chaos" ... }
	// We look for a specific "type": "chaos" field within prompt_meta to explicitly mark it
	// This is more robust than relying on the name
	reType := regexp.MustCompile(`"type"\s*:\s*"chaos"`)
	if reType.MatchString(cp) {
		return true
	}

	// Fallback to name check if type is not present (for backward compatibility with early chaos prompts)
	reMeta := regexp.MustCompile(`"prompt_name"\s*:\s*"([^"]+)"`)
	matches := reMeta.FindStringSubmatch(cp)
	if len(matches) > 1 {
		promptName := matches[1]
		if strings.Contains(promptName, "Chaos Trader") {
			return true
		}
	}

	return false
}

// GetVariantParams returns the parameters for a specific variant
func (m *Manager) GetVariantParams(variant string) map[string]string {
	params := make(map[string]string)
	v := strings.ToLower(strings.TrimSpace(variant))

	// Default empty params
	params["PRIMARY_TIMEFRAME"] = "N/A"
	params["STRUCTURE_VALIDATION_TF"] = "N/A"
	params["ENTRY_TIMEFRAME"] = "N/A"
	params["TIME_DECAY_N"] = "N/A"
	params["MIN_RR"] = "N/A"

	switch v {
	case "s1", "swing_core":
		params["PRIMARY_TIMEFRAME"] = "1h"
		params["STRUCTURE_VALIDATION_TF"] = "4h"
		params["ENTRY_TIMEFRAME"] = "15m"
		params["TIME_DECAY_N"] = "5"
		params["MIN_RR"] = "1.5"

	case "t1", "trend_follow_slow":
		params["PRIMARY_TIMEFRAME"] = "4h"
		params["STRUCTURE_VALIDATION_TF"] = "1d"
		params["ENTRY_TIMEFRAME"] = "1h"
		params["TIME_DECAY_N"] = "3"
		params["MIN_RR"] = "2.0"

	case "d1", "intraday_swing":
		params["PRIMARY_TIMEFRAME"] = "15m"
		params["STRUCTURE_VALIDATION_TF"] = "1h"
		params["ENTRY_TIMEFRAME"] = "5m"
		params["TIME_DECAY_N"] = "4"
		params["MIN_RR"] = "1.5"

	case "r1", "reversal_hunter":
		params["PRIMARY_TIMEFRAME"] = "1h"
		params["STRUCTURE_VALIDATION_TF"] = "4h"
		params["ENTRY_TIMEFRAME"] = "15m"
		params["TIME_DECAY_N"] = "3"
		params["MIN_RR"] = "2.5" // Higher RR for reversals

	case "x1", "scalp_turbo":
		params["PRIMARY_TIMEFRAME"] = "5m"
		params["STRUCTURE_VALIDATION_TF"] = "15m"
		params["ENTRY_TIMEFRAME"] = "1m"
		params["TIME_DECAY_N"] = "3"
		params["MIN_RR"] = "1.2" // Lower RR for scalping

	case "default", "":
		// Fallback to S1
		params["PRIMARY_TIMEFRAME"] = "1h"
		params["STRUCTURE_VALIDATION_TF"] = "4h"
		params["ENTRY_TIMEFRAME"] = "15m"
		params["TIME_DECAY_N"] = "5"
		params["MIN_RR"] = "1.5"
	}

	return params
}

// BuildSystemPrompt builds the system prompt for Chaos mode
func (m *Manager) BuildSystemPrompt(variant string, customPrompt string, indicators store.IndicatorConfig) string {
	var sb strings.Builder

	// BLOCK A: System Execution Contract (Code Generated)
	sb.WriteString(GenerateOutputSchema())
	sb.WriteString("\n\n")

	// BLOCK B: Strategy Logic (from Prompt File)
	// We inject parameters into the custom prompt (which acts as the Strategy Block)

	// 2. Resolve Variant Parameters
	// Instead of appending hardcoded strings, we prepare a map of parameters to inject into the template
	params := m.GetVariantParams(variant)
	v := strings.ToLower(strings.TrimSpace(variant))

	// 3. Inject Parameters into Custom Prompt (Template Replacement)
	// We replace placeholders like {PRIMARY_TIMEFRAME} with actual values
	finalPrompt := customPrompt
	for key, val := range params {
		placeholder := "{" + key + "}"
		finalPrompt = strings.ReplaceAll(finalPrompt, placeholder, val)
	}

	// 4. Append Profile Header (Optional, for transparency if not in "none" mode)
	if v != "none" && v != "default" && v != "" {
		hasPlaceholders := strings.Contains(customPrompt, "{PRIMARY_TIMEFRAME}")

		if !hasPlaceholders {
			// Legacy mode: Append the profile info manually since template tags are missing
			sb.WriteString("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
			sb.WriteString("SYSTEM PARAMETERS — 强制执行，不可覆盖\n")
			sb.WriteString("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n\n")
			sb.WriteString(fmt.Sprintf("STRUCTURE_VALIDATION_TF = %s\n", params["STRUCTURE_VALIDATION_TF"]))
			sb.WriteString(fmt.Sprintf("PRIMARY_TIMEFRAME       = %s\n", params["PRIMARY_TIMEFRAME"]))
			sb.WriteString(fmt.Sprintf("ENTRY_TIMEFRAME         = %s\n", params["ENTRY_TIMEFRAME"]))
			sb.WriteString(fmt.Sprintf("TIME_DECAY_N            = %s\n", params["TIME_DECAY_N"]))
			sb.WriteString(fmt.Sprintf("MIN_RR                  = %s\n\n", params["MIN_RR"]))
			sb.WriteString("所有决策必须以上述参数为准。\n")
			sb.WriteString("若市场数据与参数定义不符，以参数为准，不得自行调整。\n\n")
		}
	}

	sb.WriteString(finalPrompt)

	// BLOCK C: Context (Indicators)
	// This part is injected at the end of System Prompt to provide context about available data
	sb.WriteString("\n\n你拥有以下市场数据和指标:\n")

	// 1. K-line Series
	klines := []string{}
	if indicators.Klines.PrimaryTimeframe != "" {
		klines = append(klines, indicators.Klines.PrimaryTimeframe)
	}
	if indicators.Klines.EnableMultiTimeframe {
		for _, tf := range indicators.Klines.SelectedTimeframes {
			// Avoid duplicates if primary is in selected
			isDup := false
			for _, k := range klines {
				if k == tf {
					isDup = true
					break
				}
			}
			if !isDup {
				klines = append(klines, tf)
			}
		}
	}
	if len(klines) > 0 {
		sb.WriteString(fmt.Sprintf("- %s K-line series\n", strings.Join(klines, " + ")))
	} else {
		sb.WriteString("- K-line series\n")
	}

	// 2. EMA
	if indicators.EnableEMA {
		sb.WriteString(fmt.Sprintf("- EMA indicators (periods: %v)\n", indicators.EMAPeriods))
	}

	// 3. RSI
	if indicators.EnableRSI {
		sb.WriteString(fmt.Sprintf("- RSI indicators (periods: %v)\n", indicators.RSIPeriods))
	}

	// 4. ATR
	if indicators.EnableATR {
		sb.WriteString(fmt.Sprintf("- ATR indicators (periods: %v)\n", indicators.ATRPeriods))
	}

	// 5. BOLL (if enabled, though not in user example)
	if indicators.EnableBOLL {
		sb.WriteString(fmt.Sprintf("- BOLL indicators (periods: %v)\n", indicators.BOLLPeriods))
	}

	// 6. Volume
	if indicators.EnableVolume {
		sb.WriteString("- Volume data\n")
	}

	// 7. Open Interest
	if indicators.EnableOI {
		sb.WriteString("- Open Interest (OI) data\n")
	}

	// 8. Funding Rate
	if indicators.EnableFundingRate {
		sb.WriteString("- Funding rate\n")
	}

	// 9. Filter Tags (AI500 / OI_Top) - Hardcoded for now as "if available" matches user request
	sb.WriteString("- AI500 / OI_Top filter tags (if available)\n")

	// 10. Quantitative Data
	if indicators.EnableQuantData {
		sb.WriteString("- Quantitative data (institutional/retail fund flow, position changes, multi-period price changes)\n")
	}

	sb.WriteString("\n\n")

	return sb.String()
}

// ExtractReasoning extracts the Chain of Thought from the AI response
func (m *Manager) ExtractReasoning(response string) string {
	// Logic copied from engine.go extractCoTTrace but dedicated for Chaos
	// Chaos mode might favor specific tags or formats in the future
	reReasoningTag := regexp.MustCompile(`(?s)<reasoning>(.*?)</reasoning>`)
	if match := reReasoningTag.FindStringSubmatch(response); match != nil && len(match) > 1 {
		return strings.TrimSpace(match[1])
	}

	if decisionIdx := strings.Index(response, "<decision>"); decisionIdx > 0 {
		return strings.TrimSpace(response[:decisionIdx])
	}

	jsonStart := strings.Index(response, "[")
	if jsonStart > 0 {
		return strings.TrimSpace(response[:jsonStart])
	}

	return strings.TrimSpace(response)
}

// ValidateDecision validates a decision made in Chaos mode
func (m *Manager) ValidateDecision(
	d *Decision,
	reasoning *Reasoning,
	accountEquity float64,
	riskConfig store.RiskControlConfig,
) (float64, error) {
	decisionInfo := func() string {
		score := 0
		if d.TotalScore != nil {
			score = *d.TotalScore
		}
		lev := 0
		if d.Leverage != nil {
			lev = *d.Leverage
		}
		riskR := 0.0
		if d.RiskR != nil {
			riskR = *d.RiskR
		}
		entry := 0.0
		if d.EntryPrice != nil {
			entry = *d.EntryPrice
		}

		return fmt.Sprintf(
			"decision[symbol=%s action=%s lev=%d entry=%.8f risk_r=%.2f score=%d]",
			d.Symbol, d.Action, lev, entry, riskR, score,
		)
	}

	// 0. Action sanity check
	validActions := map[string]bool{
		"open_long":   true,
		"open_short":  true,
		"close_long":  true,
		"close_short": true,
		"hold":        true,
		"wait":        true,
	}
	if !validActions[d.Action] {
		return 0, fmt.Errorf("%s: invalid action '%s'", decisionInfo(), d.Action)
	}

	isOpenAction := d.Action == "open_long" || d.Action == "open_short"

	// 0.1 Strict Field Validation (Schema enforcement)
	if isOpenAction {
		// All fields required for open actions
		if d.EntryPrice == nil {
			return 0, fmt.Errorf("%s: entry required for %s", decisionInfo(), d.Action)
		}
		if d.StopLoss == nil {
			return 0, fmt.Errorf("%s: stop_loss required for %s", decisionInfo(), d.Action)
		}
		if d.TakeProfit == nil {
			return 0, fmt.Errorf("%s: take_profit required for %s", decisionInfo(), d.Action)
		}
		if d.RiskR == nil {
			return 0, fmt.Errorf("%s: risk_r required for %s", decisionInfo(), d.Action)
		}
		if d.TotalScore == nil {
			return 0, fmt.Errorf("%s: total_score required for %s", decisionInfo(), d.Action)
		}
		// Leverage is technically optional in schema but practically required for sizing?
		// Schema doesn't list leverage in required for open_* in the user snippet, but code logic uses it.
		// Assuming leverage is handled separately or defaulted if missing?
		// Existing code checked d.Leverage <= 0.
		if d.Leverage == nil || *d.Leverage <= 0 {
			return 0, fmt.Errorf("%s: leverage required for %s", decisionInfo(), d.Action)
		}
	} else {
		// No extra fields allowed for non-open actions
		var disallowed []string
		if d.Leverage != nil {
			disallowed = append(disallowed, "leverage")
		}
		if d.EntryPrice != nil {
			disallowed = append(disallowed, "entry")
		}
		if d.StopLoss != nil {
			disallowed = append(disallowed, "stop_loss")
		}
		if d.TakeProfit != nil {
			disallowed = append(disallowed, "take_profit")
		}
		if d.RiskR != nil {
			disallowed = append(disallowed, "risk_r")
		}
		if d.TotalScore != nil {
			disallowed = append(disallowed, "total_score")
		}
		if len(disallowed) > 0 {
			return 0, fmt.Errorf("%s: non-open action '%s' must include ONLY symbol+action (disallowed: %s)", decisionInfo(), d.Action, strings.Join(disallowed, ", "))
		}
		// Logic short-circuit for non-open
		logger.Infof("✓ Chaos decision validated (non-opening) | %s %s", d.Action, d.Symbol)
		return 0, nil
	}

	// Dereference pointers for easier usage below (safe because we checked nil above)
	entryPrice := *d.EntryPrice
	stopLoss := *d.StopLoss
	takeProfit := *d.TakeProfit
	riskR := *d.RiskR
	leverage := *d.Leverage

	// 0.2 System Risk Flag Check
	// Note: With flexible reasoning, if the flag is missing, it defaults to false (safe for simple prompts)
	// But if present and true, it MUST trigger.
	if reasoning != nil && reasoning.SystemRiskFlag {
		return 0, fmt.Errorf("SYSTEM_RISK_FLAG_TRIGGERED: open_* actions forbidden when system_risk_flag is true")
	}

	// 1. RiskR hard constraints
	const MaxRiskR = 1.5

	// Determine base risk percent (default 1%)
	baseRiskPercent := riskConfig.BaseRiskPercent
	if baseRiskPercent <= 0 {
		baseRiskPercent = 0.01
	}

	// Range validation (0.1 to 1.5)
	const MinRiskR = 0.1
	if riskR < MinRiskR || riskR > MaxRiskR {
		return 0, fmt.Errorf("%s: RiskR %.2f must be between %.2f and %.2f", decisionInfo(), riskR, MinRiskR, MaxRiskR)
	}

	// 1.1 RiskR Consistency Check with Audit Path
	if reasoning != nil {
		// Find matching opportunity
		var matchedOpp *Opportunity
		for _, opp := range reasoning.Opportunities {
			if opp.Symbol == d.Symbol {
				matchedOpp = &opp
				break
			}
		}
		if matchedOpp != nil {
			// Parse "Final X.XR" from audit_path
			// Expected format: "... Final 0.5R"
			// Simple check: does it contain the formatted RiskR string?
			expectedStr := fmt.Sprintf("Final %.1fR", riskR)
			expectedStr2 := fmt.Sprintf("Final %.2fR", riskR) // Also allow 2 decimal places (e.g. 0.50R)

			// If 0.5, fmt gives 0.5.
			// Let's use flexible check or regex if needed.
			// User example: "Final 0.5R" or "Final 0.50R"
			if !strings.Contains(matchedOpp.AuditPath, expectedStr) && !strings.Contains(matchedOpp.AuditPath, expectedStr2) {
				return 0, fmt.Errorf("%s: RiskR consistency check failed. Decision=%.2f, AuditPath='%s' (Expected '%s' or '%s')",
					decisionInfo(), riskR, matchedOpp.AuditPath, expectedStr, expectedStr2)
			}
		}
	}

	// 2. Mandatory price anchors
	if entryPrice <= 0 {
		return 0, fmt.Errorf("%s: entry price must be positive", decisionInfo())
	}
	if stopLoss <= 0 {
		return 0, fmt.Errorf("%s: stop loss must be positive", decisionInfo())
	}
	if takeProfit <= 0 {
		return 0, fmt.Errorf("%s: take profit must be positive", decisionInfo())
	}

	// Directional price logic
	if d.Action == "open_long" {
		if !(stopLoss < entryPrice && entryPrice < takeProfit) {
			return 0, fmt.Errorf("%s: invalid price structure for open_long (SL < Entry < TP)", decisionInfo())
		}
	} else {
		if !(takeProfit < entryPrice && entryPrice < stopLoss) {
			return 0, fmt.Errorf("%s: invalid price structure for open_short (TP < Entry < SL)", decisionInfo())
		}
	}

	// 3. R:R validation (Chaos requires edge)
	var risk, reward float64
	if d.Action == "open_long" {
		risk = entryPrice - stopLoss
		reward = takeProfit - entryPrice
	} else {
		risk = stopLoss - entryPrice
		reward = entryPrice - takeProfit
	}

	if risk <= 0 || reward <= 0 {
		return 0, fmt.Errorf("%s: invalid risk/reward distances (risk=%.4f reward=%.4f)", decisionInfo(), risk, reward)
	}

	riskRewardRatio := reward / risk

	// Use configured min risk/reward ratio, default to 1.0 if not set or invalid
	minRR := riskConfig.MinRiskRewardRatio
	if minRR <= 0 {
		minRR = 1.0
	}

	if riskRewardRatio < minRR {
		return 0, fmt.Errorf("%s: Chaos decision requires R:R ≥ %.2f (got %.4f). Params: Entry=%.4f, SL=%.4f, TP=%.4f, Risk=%.4f, Reward=%.4f",
			decisionInfo(), minRR, riskRewardRatio, entryPrice, stopLoss, takeProfit, risk, reward)
	}

	// 4. Position sizing via RiskR
	// Chaos mode uses RiskR based sizing: Risk Amount = Equity * 1% * RiskR
	var riskAmount float64
	riskAmount = accountEquity * baseRiskPercent * riskR

	quantity := riskAmount / risk

	positionSizeUSD := quantity * entryPrice

	if positionSizeUSD <= 0 {
		return 0, fmt.Errorf("%s: calculated position size invalid: %.2f", decisionInfo(), positionSizeUSD)
	}

	minPositionSize := riskConfig.MinPositionSize
	if positionSizeUSD < minPositionSize {
		logger.Infof("⚠️  [Position Size Adjustment] %s calculated size %.2f < min %.2f, adjusting to %.2f",
			decisionInfo(), positionSizeUSD, minPositionSize, minPositionSize)
		positionSizeUSD = minPositionSize

		if positionSizeUSD > accountEquity {
			return 0, fmt.Errorf("%s: adjusted position size %.2f exceeds account equity %.2f", decisionInfo(), positionSizeUSD, accountEquity)
		}
	}

	// 5. Symbol-based caps
	maxLeverage := riskConfig.AltcoinMaxLeverage
	maxPosValue := accountEquity * riskConfig.AltcoinMaxPositionValueRatio

	if d.Symbol == "BTCUSDT" || d.Symbol == "ETHUSDT" {
		maxLeverage = riskConfig.BTCETHMaxLeverage
		maxPosValue = accountEquity * riskConfig.BTCETHMaxPositionValueRatio
	}

	if leverage > maxLeverage {
		logger.Infof("⚠️  [Leverage Adjustment] %s leverage %dx exceeds limit %dx for %s, adjusting to %dx",
			decisionInfo(), leverage, maxLeverage, d.Symbol, maxLeverage)
		leverage = maxLeverage
		// Update the pointer value
		*d.Leverage = leverage
	}

	if positionSizeUSD > maxPosValue {
		logger.Infof("⚠️  [Position Size Clamped] %s position size %.2f exceeds max allowed %.2f, clamping to max",
			decisionInfo(), positionSizeUSD, maxPosValue)
		positionSizeUSD = maxPosValue
	}

	// Final check: Position size must be valid after all adjustments
	if positionSizeUSD <= 0 {
		return 0, fmt.Errorf("%s: final position size %.2f is invalid (likely due to zero equity or strict limits)", decisionInfo(), positionSizeUSD)
	}

	// 6. Logging (audit trail)
	logger.Infof(
		"✓ Chaos decision validated | %s %s | RiskR=%.2f | Size=%.2f USDT | R:R=%.2f",
		d.Action, d.Symbol, riskR, positionSizeUSD, riskRewardRatio,
	)

	return positionSizeUSD, nil
}

func abs(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}
