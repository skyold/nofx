package chaos

import (
	"fmt"
	"regexp"
	"strings"

	"nofx/logger"
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

// BuildPrompt builds the system prompt for Chaos mode
func (m *Manager) BuildPrompt(variant string, customPrompt string, availableIndicatorsFunc func(*strings.Builder)) string {
	var sb strings.Builder

	// 1. Available Indicators (Context)
	sb.WriteString("\n\n你拥有以下市场数据和指标:\n")
	availableIndicatorsFunc(&sb)
	sb.WriteString("\n\n")

	// 2. Resolve Variant Parameters
	// Instead of appending hardcoded strings, we prepare a map of parameters to inject into the template
	params := make(map[string]string)
	v := strings.ToLower(strings.TrimSpace(variant))

	// Default empty params
	params["PRIMARY_TIMEFRAME"] = "N/A"
	params["STRUCTURE_VALIDATION_TF"] = "N/A"
	params["POSITION_STYLE"] = "N/A"
	params["TIME_DECAY_N"] = "N/A"
	params["MIN_RR"] = "N/A"
	params["PROFILE_NAME"] = "CUSTOM"
	params["PROFILE_DESC"] = "Custom Profile"

	switch v {
	case "s1", "swing_core":
		params["PRIMARY_TIMEFRAME"] = "1h"
		params["STRUCTURE_VALIDATION_TF"] = "4h"
		params["POSITION_STYLE"] = "SWING"
		params["TIME_DECAY_N"] = "5"
		params["MIN_RR"] = "1.5"
		params["PROFILE_NAME"] = "S1 — SWING_CORE"
		params["PROFILE_DESC"] = "主力实盘账户 | 稳定性最高 | 适合长期跑"

	case "t1", "trend_follow_slow":
		params["PRIMARY_TIMEFRAME"] = "4h"
		params["STRUCTURE_VALIDATION_TF"] = "1d"
		params["POSITION_STYLE"] = "TREND"
		params["TIME_DECAY_N"] = "3"
		params["MIN_RR"] = "2.0"
		params["PROFILE_NAME"] = "T1 — TREND_FOLLOW_SLOW"
		params["PROFILE_DESC"] = "中长期趋势 | 牛市单边 | 极少交易"

	case "d1", "intraday_swing":
		params["PRIMARY_TIMEFRAME"] = "15m"
		params["STRUCTURE_VALIDATION_TF"] = "1h"
		params["POSITION_STYLE"] = "SWING"
		params["TIME_DECAY_N"] = "4"
		params["MIN_RR"] = "1.5"
		params["PROFILE_NAME"] = "D1 — INTRADAY_SWING"
		params["PROFILE_DESC"] = "日内波段 | 鲁棒性验证 | 较活跃"

	case "r1", "range_defensive":
		params["PRIMARY_TIMEFRAME"] = "30m"
		params["STRUCTURE_VALIDATION_TF"] = "2h"
		params["POSITION_STYLE"] = "SWING"
		params["TIME_DECAY_N"] = "3"
		params["MIN_RR"] = "1.2"
		params["PROFILE_NAME"] = "R1 — RANGE_DEFENSIVE"
		params["PROFILE_DESC"] = "震荡防御 | 盈利周期短 | Regime敏感"

	case "x1", "scalp_experiment":
		params["PRIMARY_TIMEFRAME"] = "5m"
		params["STRUCTURE_VALIDATION_TF"] = "15m"
		params["POSITION_STYLE"] = "SCALP"
		params["TIME_DECAY_N"] = "2"
		params["MIN_RR"] = "1.2"
		params["PROFILE_NAME"] = "X1 — SCALP_EXPERIMENT"
		params["PROFILE_DESC"] = "微结构研究 | 不稳定 | Token消耗高"

	case "none", "":
		// No injection, parameters remain as defaults or placeholders
		// This allows for pure custom prompts without forced parameter injection
	}

	// 3. Inject Parameters into Custom Prompt (Template Replacement)
	// We replace placeholders like {PRIMARY_TIMEFRAME} with actual values
	finalPrompt := customPrompt
	for key, val := range params {
		placeholder := "{" + key + "}"
		finalPrompt = strings.ReplaceAll(finalPrompt, placeholder, val)
	}

	// 4. Append Profile Header (Optional, for transparency if not in "none" mode)
	if v != "none" && v != "" {
		hasPlaceholders := strings.Contains(customPrompt, "{PRIMARY_TIMEFRAME}")

		if !hasPlaceholders {
			// Legacy mode: Append the profile info manually since template tags are missing
			// We reconstruct the info string from params to keep it DRY
			sb.WriteString(fmt.Sprintf("## Profile %s\n\n", params["PROFILE_NAME"]))
			sb.WriteString(fmt.Sprintf("### 使用信息\n%s\n\n", params["PROFILE_DESC"]))
			sb.WriteString("━━━━━━━━━━━━━━━━━━━━\nGLOBAL SYSTEM PARAMETERS (READ-ONLY)\n━━━━━━━━━━━━━━━━━━━━\n\n")
			sb.WriteString(fmt.Sprintf("PRIMARY_TIMEFRAME = %s\n", params["PRIMARY_TIMEFRAME"]))
			sb.WriteString(fmt.Sprintf("STRUCTURE_VALIDATION_TF = %s\n", params["STRUCTURE_VALIDATION_TF"]))
			sb.WriteString(fmt.Sprintf("POSITION_STYLE = %s\n", params["POSITION_STYLE"]))
			sb.WriteString(fmt.Sprintf("TIME_DECAY_N = %s\n", params["TIME_DECAY_N"]))
			sb.WriteString(fmt.Sprintf("MIN_RR = %s\n\n", params["MIN_RR"]))
			sb.WriteString("The LLM MUST NOT modify or reinterpret these parameters.\n\n")
		}
	}

	sb.WriteString(finalPrompt)

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
	accountEquity float64,
	btcEthLeverage, altcoinLeverage int,
	btcEthPosRatio, altcoinPosRatio float64,
) (float64, error) {
	decisionInfo := func() string {
		return fmt.Sprintf(
			"decision[symbol=%s action=%s lev=%d entry=%.8f sl=%.8f tp=%.8f risk_r=%.2f conf=%d]",
			d.Symbol, d.Action, d.Leverage, d.EntryPrice, d.StopLoss, d.TakeProfit, d.RiskR, d.Confidence,
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

	if d.Action != "open_long" && d.Action != "open_short" {
		// For non-opening actions (wait, hold, close_long, close_short), we skip RiskR/Pricing validation
		logger.Infof("✓ Chaos decision validated (non-opening) | %s %s", d.Action, d.Symbol)
		return 0, nil
	}

	// 1. RiskR hard constraints
	const MaxRiskR = 1.5
	const baseRiskPercent = 0.01 // 1R = 1% equity

	if d.RiskR <= 0 {
		return 0, fmt.Errorf("%s: RiskR must be greater than 0 in Chaos decision", decisionInfo())
	}
	if d.RiskR > MaxRiskR {
		return 0, fmt.Errorf("%s: RiskR %.2f exceeds hard limit %.2f", decisionInfo(), d.RiskR, MaxRiskR)
	}

	// 2. Mandatory price anchors
	if d.EntryPrice <= 0 {
		return 0, fmt.Errorf("%s: entry price required for RiskR decision", decisionInfo())
	}
	if d.StopLoss <= 0 {
		return 0, fmt.Errorf("%s: stop loss required for RiskR decision", decisionInfo())
	}
	if d.TakeProfit <= 0 {
		return 0, fmt.Errorf("%s: take profit required for RiskR decision", decisionInfo())
	}

	// Directional price logic
	if d.Action == "open_long" {
		if !(d.StopLoss < d.EntryPrice && d.EntryPrice < d.TakeProfit) {
			return 0, fmt.Errorf("%s: invalid price structure for open_long (SL < Entry < TP)", decisionInfo())
		}
	} else {
		if !(d.TakeProfit < d.EntryPrice && d.EntryPrice < d.StopLoss) {
			return 0, fmt.Errorf("%s: invalid price structure for open_short (TP < Entry < SL)", decisionInfo())
		}
	}

	// 3. R:R validation (Chaos requires edge)
	var risk, reward float64
	if d.Action == "open_long" {
		risk = d.EntryPrice - d.StopLoss
		reward = d.TakeProfit - d.EntryPrice
	} else {
		risk = d.StopLoss - d.EntryPrice
		reward = d.EntryPrice - d.TakeProfit
	}

	if risk <= 0 || reward <= 0 {
		return 0, fmt.Errorf("%s: invalid risk/reward distances (risk=%.4f reward=%.4f)", decisionInfo(), risk, reward)
	}

	riskRewardRatio := reward / risk
	if riskRewardRatio < 2 {
		return 0, fmt.Errorf("%s: Chaos decision requires R:R ≥ 2 (got %.2f). Params: Entry=%.2f, SL=%.2f, TP=%.2f, Risk=%.2f, Reward=%.2f",
			decisionInfo(), riskRewardRatio, d.EntryPrice, d.StopLoss, d.TakeProfit, risk, reward)
	}

	// 4. Position sizing via RiskR or RiskUSD
	// If RiskUSD is provided and valid, use it directly (converting to equivalent RiskR logic)
	// Otherwise use RiskR based on account equity
	var riskAmount float64
	if d.RiskUSD > 0 {
		riskAmount = d.RiskUSD
		// Back-calculate RiskR for consistency in logging/audit
		// RiskAmount = Equity * 1% * RiskR => RiskR = RiskAmount / (Equity * 0.01)
		if accountEquity > 0 {
			calculatedRiskR := riskAmount / (accountEquity * baseRiskPercent)
			if calculatedRiskR > MaxRiskR {
				logger.Infof("⚠️  [RiskR Adjustment] Calculated RiskR %.2f (from RiskUSD %.2f) exceeds limit %.2f, clamping RiskUSD",
					calculatedRiskR, d.RiskUSD, MaxRiskR)

				// Clamp risk amount to MaxRiskR limit
				// MaxRiskAmount = Equity * 1% * MaxRiskR
				maxRiskAmount := accountEquity * baseRiskPercent * MaxRiskR
				riskAmount = maxRiskAmount
				d.RiskR = MaxRiskR
			} else {
				d.RiskR = calculatedRiskR
			}
		}
	} else {
		riskAmount = accountEquity * baseRiskPercent * d.RiskR
	}

	quantity := riskAmount / risk
	// Calculate PositionSizeUSD but return it instead of modifying d (as d doesn't have the field anymore)
	// Wait, validateDecision modifies d directly for adjustments.
	// But d no longer has PositionSizeUSD.
	// We need to return the calculated PositionSizeUSD or store it elsewhere?
	// The validation logic seems to assume d carries the state.
	// If chaos.Decision doesn't have PositionSizeUSD, we can't store it there.
	// But chaos/engine.go needs it to map to kernel.Decision.

	// Re-reading user request: "PositionSizeUSD is output of Nofx normal mode, unused, currently calculated from RiskR."
	// This implies chaos.Decision shouldn't have it as an INPUT from LLM.
	// BUT, does it need it as an internal field for passing to kernel?
	// User said: "remove it".
	// If I remove it from chaos.Decision, I need to calculate it in engine.go before mapping to kernel.Decision?
	// Or maybe ValidateDecision should return the calculated position size?

	positionSizeUSD := quantity * d.EntryPrice

	if positionSizeUSD <= 0 {
		return 0, fmt.Errorf("%s: calculated position size invalid: %.2f", decisionInfo(), positionSizeUSD)
	}

	const minPositionSize = 100.0
	if positionSizeUSD < minPositionSize {
		logger.Infof("⚠️  [Position Size Adjustment] %s calculated size %.2f < min %.2f, adjusting to %.2f",
			decisionInfo(), positionSizeUSD, minPositionSize, minPositionSize)
		positionSizeUSD = minPositionSize

		if positionSizeUSD > accountEquity {
			return 0, fmt.Errorf("%s: adjusted position size %.2f exceeds account equity %.2f", decisionInfo(), positionSizeUSD, accountEquity)
		}
	}

	// 5. Symbol-based caps
	maxLeverage := altcoinLeverage
	maxPosValue := accountEquity * altcoinPosRatio

	if d.Symbol == "BTCUSDT" || d.Symbol == "ETHUSDT" {
		maxLeverage = btcEthLeverage
		maxPosValue = accountEquity * btcEthPosRatio
	}

	if d.Leverage <= 0 {
		return 0, fmt.Errorf("%s: leverage must be provided for Chaos decision", decisionInfo())
	}
	if d.Leverage > maxLeverage {
		logger.Infof("⚠️  [Leverage Adjustment] %s leverage %dx exceeds limit %dx for %s, adjusting to %dx",
			decisionInfo(), d.Leverage, maxLeverage, d.Symbol, maxLeverage)
		d.Leverage = maxLeverage
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
		d.Action, d.Symbol, d.RiskR, positionSizeUSD, riskRewardRatio,
	)

	return positionSizeUSD, nil
}
