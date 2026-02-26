package chaos

import (
	"fmt"
	"strings"

	"nofx/logger"
	"nofx/store"
)

// =============================================================================
// Manager - 内部业务逻辑层
// =============================================================================
// Manager 处理 Chaos 模式的核心业务逻辑，不直接对外暴露。
// 所有外部调用都通过 ChaosEngine 进行。
// =============================================================================

// Manager handles Chaos trading mode specific logic
type Manager struct{}

// NewManager creates a new Chaos Manager
func NewManager() *Manager {
	return &Manager{}
}

// =============================================================================
// 模式检测与参数
// =============================================================================

// IsChaosMode checks if the current prompt indicates Chaos mode
func (m *Manager) IsChaosMode(customPrompt string) bool {
	if customPrompt == "" {
		return false
	}

	cp := strings.TrimSpace(customPrompt)
	chaosTags := []string{"这是一个Chaos策略", "Chaos Trader"}
	for _, tag := range chaosTags {
		if strings.HasPrefix(cp, tag) || strings.Contains(cp, tag) {
			return true
		}
	}

	// Also check for JSON-style type indicator
	if strings.Contains(customPrompt, `"type": "chaos"`) {
		return true
	}

	return false
}

// GetVariantParams returns the parameters for a specific variant
func (m *Manager) GetVariantParams(variant string) map[string]string {
	params := make(map[string]string)
	v := strings.ToLower(strings.TrimSpace(variant))

	// Default values
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
		params["MIN_RR"] = "2.5"

	case "x1", "scalp_turbo":
		params["PRIMARY_TIMEFRAME"] = "5m"
		params["STRUCTURE_VALIDATION_TF"] = "15m"
		params["ENTRY_TIMEFRAME"] = "1m"
		params["TIME_DECAY_N"] = "3"
		params["MIN_RR"] = "1.2"

	case "default", "":
		params["PRIMARY_TIMEFRAME"] = "1h"
		params["STRUCTURE_VALIDATION_TF"] = "4h"
		params["ENTRY_TIMEFRAME"] = "15m"
		params["TIME_DECAY_N"] = "5"
		params["MIN_RR"] = "1.5"
	}

	return params
}

// =============================================================================
// 决策验证
// =============================================================================

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

	if isOpenAction {
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
		if d.Leverage == nil || *d.Leverage <= 0 {
			return 0, fmt.Errorf("%s: leverage required for %s", decisionInfo(), d.Action)
		}
	} else {
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
		logger.Infof("✓ Chaos decision validated (non-opening) | %s %s", d.Action, d.Symbol)
		return 0, nil
	}

	entryPrice := *d.EntryPrice
	stopLoss := *d.StopLoss
	takeProfit := *d.TakeProfit
	riskR := *d.RiskR
	leverage := *d.Leverage

	if reasoning != nil && reasoning.SystemRiskFlag {
		return 0, fmt.Errorf("SYSTEM_RISK_FLAG_TRIGGERED: open_* actions forbidden when system_risk_flag is true")
	}

	const MaxRiskR = 1.5

	baseRiskPercent := riskConfig.BaseRiskPercent
	if baseRiskPercent <= 0 {
		baseRiskPercent = 0.01
	}

	const MinRiskR = 0.1
	if riskR < MinRiskR || riskR > MaxRiskR {
		return 0, fmt.Errorf("%s: RiskR %.2f must be between %.2f and %.2f", decisionInfo(), riskR, MinRiskR, MaxRiskR)
	}

	if reasoning != nil {
		var matchedOpp *Opportunity
		for _, opp := range reasoning.Opportunities {
			if opp.Symbol == d.Symbol {
				matchedOpp = &opp
				break
			}
		}
		if matchedOpp != nil {
			expectedStr := fmt.Sprintf("Final %.1fR", riskR)
			expectedStr2 := fmt.Sprintf("Final %.2fR", riskR)

			if !strings.Contains(matchedOpp.AuditPath, expectedStr) && !strings.Contains(matchedOpp.AuditPath, expectedStr2) {
				logger.Warnf("%s: RiskR consistency check failed. Decision=%.2f, AuditPath='%s' (Expected '%s' or '%s')",
					decisionInfo(), riskR, matchedOpp.AuditPath, expectedStr, expectedStr2)
			}
		}
	}

	if entryPrice <= 0 {
		return 0, fmt.Errorf("%s: entry price must be positive", decisionInfo())
	}
	if stopLoss <= 0 {
		return 0, fmt.Errorf("%s: stop loss must be positive", decisionInfo())
	}
	if takeProfit <= 0 {
		return 0, fmt.Errorf("%s: take profit must be positive", decisionInfo())
	}

	if d.Action == "open_long" {
		if !(stopLoss < entryPrice && entryPrice < takeProfit) {
			return 0, fmt.Errorf("%s: invalid price structure for open_long (SL < Entry < TP)", decisionInfo())
		}
	} else {
		if !(takeProfit < entryPrice && entryPrice < stopLoss) {
			return 0, fmt.Errorf("%s: invalid price structure for open_short (TP < Entry < SL)", decisionInfo())
		}
	}

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

	minRR := riskConfig.MinRiskRewardRatio
	if minRR <= 0 {
		minRR = 1.0
	}

	if riskRewardRatio < minRR {
		return 0, fmt.Errorf("%s: Chaos decision requires R:R ≥ %.2f (got %.4f). Params: Entry=%.4f, SL=%.4f, TP=%.4f, Risk=%.4f, Reward=%.4f",
			decisionInfo(), minRR, riskRewardRatio, entryPrice, stopLoss, takeProfit, risk, reward)
	}

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

		if positionSizeUSD > accountEquity {
			return 0, fmt.Errorf("%s: adjusted position size %.2f exceeds account equity %.2f", decisionInfo(), positionSizeUSD, accountEquity)
		}
	}

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
		*d.Leverage = leverage
	}

	if positionSizeUSD > maxPosValue {
		logger.Infof("⚠️  [Position Size Clamped] %s position size %.2f exceeds max allowed %.2f, clamping to max",
			decisionInfo(), positionSizeUSD, maxPosValue)
		positionSizeUSD = maxPosValue
	}

	if positionSizeUSD <= 0 {
		return 0, fmt.Errorf("%s: final position size %.2f is invalid (likely due to zero equity or strict limits)", decisionInfo(), positionSizeUSD)
	}

	logger.Infof(
		"✓ Chaos decision validated | %s %s | RiskR=%.2f | Size=%.2f USDT | R:R=%.2f",
		d.Action, d.Symbol, riskR, positionSizeUSD, riskRewardRatio,
	)

	return positionSizeUSD, nil
}
