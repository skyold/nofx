package ai

import (
	"fmt"
	"strings"

	"nofx/engine"
	"nofx/store"
)

type PromptBuilder struct {
	config *store.StrategyConfig
}

func NewPromptBuilder(config *store.StrategyConfig) *PromptBuilder {
	return &PromptBuilder{
		config: config,
	}
}

func (p *PromptBuilder) BuildSystemPrompt(ctx *Context) string {
	var sb strings.Builder

	sb.WriteString(generateOutputSchema())
	sb.WriteString("\n\n")

	var customPrompt string
	var indicators store.IndicatorConfig
	var variant string

	if p.config != nil {
		if p.config.ChaosConfig != nil {
			customPrompt = p.config.ChaosConfig.ChaosPrompt
			indicators = p.config.ChaosConfig.Indicators
			variant = p.config.ChaosConfig.SystemPromptVariant
		} else {
			customPrompt = p.config.CustomPrompt
			indicators = p.config.Indicators
			variant = p.config.PromptVariant
		}
	}

	params := getVariantParams(variant)
	v := strings.ToLower(strings.TrimSpace(variant))

	finalPrompt := customPrompt
	for key, val := range params {
		placeholder := fmt.Sprintf("{%s}", key)
		finalPrompt = strings.ReplaceAll(finalPrompt, placeholder, val)
	}

	if v != "none" && v != "default" && v != "" {
		hasPlaceholders := hasAnyPlaceholder(customPrompt, params)
		if !hasPlaceholders {
			writeSystemParameters(&sb, params)
		}
	}

	sb.WriteString(finalPrompt)

	sb.WriteString("\n\n你拥有以下市场数据和指标:\n")

	klines := collectUniqueKlines(indicators)
	if len(klines) > 0 {
		sb.WriteString(fmt.Sprintf("- %s K-line series\n", strings.Join(klines, " + ")))
	} else {
		sb.WriteString("- K-line series\n")
	}

	writeIndicatorWithPeriods(&sb, indicators.EnableEMA, "EMA", indicators.EMAPeriods)
	writeIndicatorWithPeriods(&sb, indicators.EnableRSI, "RSI", indicators.RSIPeriods)
	writeIndicatorWithPeriods(&sb, indicators.EnableATR, "ATR", indicators.ATRPeriods)
	writeIndicatorWithPeriods(&sb, indicators.EnableBOLL, "BOLL", indicators.BOLLPeriods)
	writeSimpleIndicator(&sb, indicators.EnableVolume, "Volume")
	writeSimpleIndicator(&sb, indicators.EnableOI, "Open Interest (OI)")
	writeSimpleIndicator(&sb, indicators.EnableFundingRate, "Funding rate")

	sb.WriteString("- AI500 / OI_Top filter tags (if available)\n")

	if indicators.EnableQuantData {
		sb.WriteString("- Quantitative data (institutional/retail fund flow, position changes, multi-period price changes)\n")
	}

	sb.WriteString("\n\n")

	return sb.String()
}

func (p *PromptBuilder) BuildUserPrompt(ctx *Context) string {
	if ctx == nil {
		return ""
	}

	var sb strings.Builder

	sb.WriteString("# AI 模式用户提示\n\n")

	sb.WriteString(buildHeader(ctx))
	sb.WriteString(buildAccountStatus(ctx))
	sb.WriteString(buildPositions(ctx))

	sb.WriteString("---\n\n")

	return sb.String()
}

func generateOutputSchema() string {
	return `## 输出格式

你必须输出一个 JSON 数组，每个元素代表一个交易决策。

### 决策格式

[
  {
    "symbol": "BTCUSDT",           // 交易品种
    "action": "open_long",         // 操作: open_long, open_short, close_long, close_short, hold, wait
    "leverage": 10,                // 杠杆倍数 (仅开仓时需要)
    "entry": 50000.00,             // 入场价格 (仅开仓时需要)
    "stop_loss": 49000.00,         // 止损价格 (仅开仓时需要)
    "take_profit": 52000.00,       // 止盈价格 (仅开仓时需要)
    "risk_r": 1.0,                 // 风险倍数 R (仅开仓时需要)
    "total_score": 80              // 信心分数 0-100 (仅开仓时需要)
  }
]

### 重要约束

1. **开仓决策** 必须包含: symbol, action, leverage, entry, stop_loss, take_profit, risk_r, total_score
2. **平仓/观望** 只需: symbol, action
3. **风险收益比** 必须 >= 1.5 (MinRiskR)
4. **价格结构**:
   - open_long: stop_loss < entry < take_profit
   - open_short: take_profit < entry < stop_loss
5. **决策数量**: 每次最多 3 个

### Chain of Thought

在输出 JSON 之前，请先输出你的分析过程，使用 <reasoning> 标签包裹:

<reasoning>
{
  "market_context": {
    "system_risk_flag": false,
    "overall_trend": "bullish"
  },
  "opportunities": [
    {
      "symbol": "BTCUSDT",
      "audit_path": "Final 1.5R"
    }
  ]
}
</reasoning>

然后输出决策 JSON:
<decision>
[...]
</decision>`
}

func getVariantParams(variant string) map[string]string {
	params := make(map[string]string)
	v := strings.ToLower(strings.TrimSpace(variant))

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

func hasAnyPlaceholder(customPrompt string, params map[string]string) bool {
	for key := range params {
		placeholder := fmt.Sprintf("{%s}", key)
		if strings.Contains(customPrompt, placeholder) {
			return true
		}
	}
	return false
}

func writeSystemParameters(sb *strings.Builder, params map[string]string) {
	sb.WriteString("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
	sb.WriteString("SYSTEM PARAMETERS — 强制执行，不可覆盖\n")
	sb.WriteString("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n\n")

	writeParam(sb, "STRUCTURE_VALIDATION_TF", params["STRUCTURE_VALIDATION_TF"])
	writeParam(sb, "PRIMARY_TIMEFRAME", params["PRIMARY_TIMEFRAME"])
	writeParam(sb, "ENTRY_TIMEFRAME", params["ENTRY_TIMEFRAME"])
	writeParam(sb, "TIME_DECAY_N", params["TIME_DECAY_N"])
	writeParam(sb, "MIN_RR", params["MIN_RR"])

	sb.WriteString("\n")
	sb.WriteString("所有决策必须以上述参数为准。\n")
	sb.WriteString("若市场数据与参数定义不符，以参数为准，不得自行调整。\n\n")
}

func writeParam(sb *strings.Builder, key, value string) {
	sb.WriteString(fmt.Sprintf("%-24s = %s\n", key, value))
}

func collectUniqueKlines(indicators store.IndicatorConfig) []string {
	seen := make(map[string]bool)
	klines := []string{}

	if indicators.Klines.PrimaryTimeframe != "" {
		klines = append(klines, indicators.Klines.PrimaryTimeframe)
		seen[indicators.Klines.PrimaryTimeframe] = true
	}

	if indicators.Klines.EnableMultiTimeframe {
		for _, tf := range indicators.Klines.SelectedTimeframes {
			if !seen[tf] {
				klines = append(klines, tf)
				seen[tf] = true
			}
		}
	}

	return klines
}

func writeIndicatorWithPeriods(sb *strings.Builder, enabled bool, name string, periods interface{}) {
	if enabled {
		sb.WriteString(fmt.Sprintf("- %s indicators\n", name))
	}
}

func writeSimpleIndicator(sb *strings.Builder, enabled bool, name string) {
	if enabled {
		sb.WriteString(fmt.Sprintf("- %s data\n", name))
	}
}

func buildHeader(ctx *Context) string {
	return fmt.Sprintf("Time: %s | Period: #%d | Runtime: %d minutes\n\n",
		ctx.CurrentTime, ctx.CallCount, ctx.RuntimeMinutes)
}

func buildAccountStatus(ctx *Context) string {
	if ctx.Account == nil {
		return ""
	}

	account, ok := ctx.Account.(engine.AccountInfo)
	if !ok {
		return ""
	}

	balancePercent := 0.0
	if account.TotalEquity > 0 {
		balancePercent = (account.AvailableBalance / account.TotalEquity) * 100
	}

	var totalPnLPct float64
	if account.TotalEquity > 0 {
		totalPnLPct = (account.UnrealizedPnL / account.TotalEquity) * 100
	}

	var marginUsedPct float64
	if account.TotalEquity > 0 {
		marginUsedPct = ((account.TotalEquity - account.AvailableBalance) / account.TotalEquity) * 100
	}

	return fmt.Sprintf("Account: Equity %.2f | Balance %.2f (%.1f%%) | PnL %+.2f%% | Margin %.1f%% | Positions %d\n\n",
		account.TotalEquity,
		account.AvailableBalance,
		balancePercent,
		totalPnLPct,
		marginUsedPct,
		account.PositionCount)
}

func buildPositions(ctx *Context) string {
	if ctx.Positions == nil {
		return "Current Positions: None\n\n"
	}

	positions, ok := ctx.Positions.([]engine.PositionInfo)
	if !ok || len(positions) == 0 {
		return "Current Positions: None\n\n"
	}

	var sb strings.Builder
	sb.WriteString("## Current Positions\n")

	for i, pos := range positions {
		sb.WriteString(fmt.Sprintf("%d. %s %s | Entry %.4f Current %.4f | PnL %+.2f%% | Leverage %dx\n\n",
			i+1, pos.Symbol, strings.ToUpper(pos.Side),
			pos.EntryPrice, pos.MarkPrice, pos.UnrealizedPnLPct, pos.Leverage))
	}

	return sb.String()
}
