// =============================================================================
// Chaos Trading System - System Prompt (Manager Methods)
// =============================================================================
//
// System Prompt 是发送给 LLM 的系统提示词的一部分。
// 完整提示词 = System Prompt + User Prompt
//
// 此文件包含 Manager 的 System Prompt 构建方法。
// =============================================================================

package chaos

import (
	"fmt"
	"regexp"
	"strings"

	"nofx/store"
)

// BuildSystemPrompt builds the system prompt for Chaos mode
func (m *Manager) BuildSystemPrompt(variant string, customPrompt string, indicators store.IndicatorConfig) string {
	var sb strings.Builder

	sb.WriteString(GenerateOutputSchema())
	sb.WriteString("\n\n")

	params := m.GetVariantParams(variant)
	v := strings.ToLower(strings.TrimSpace(variant))

	finalPrompt := customPrompt
	for key, val := range params {
		placeholder := "{" + key + "}"
		finalPrompt = strings.ReplaceAll(finalPrompt, placeholder, val)
	}

	if v != "none" && v != "default" && v != "" {
		hasPlaceholders := strings.Contains(customPrompt, "{PRIMARY_TIMEFRAME}")

		if !hasPlaceholders {
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

	sb.WriteString("\n\n你拥有以下市场数据和指标:\n")

	klines := []string{}
	if indicators.Klines.PrimaryTimeframe != "" {
		klines = append(klines, indicators.Klines.PrimaryTimeframe)
	}
	if indicators.Klines.EnableMultiTimeframe {
		for _, tf := range indicators.Klines.SelectedTimeframes {
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

	if indicators.EnableEMA {
		sb.WriteString(fmt.Sprintf("- EMA indicators (periods: %v)\n", indicators.EMAPeriods))
	}

	if indicators.EnableRSI {
		sb.WriteString(fmt.Sprintf("- RSI indicators (periods: %v)\n", indicators.RSIPeriods))
	}

	if indicators.EnableATR {
		sb.WriteString(fmt.Sprintf("- ATR indicators (periods: %v)\n", indicators.ATRPeriods))
	}

	if indicators.EnableBOLL {
		sb.WriteString(fmt.Sprintf("- BOLL indicators (periods: %v)\n", indicators.BOLLPeriods))
	}

	if indicators.EnableVolume {
		sb.WriteString("- Volume data\n")
	}

	if indicators.EnableOI {
		sb.WriteString("- Open Interest (OI) data\n")
	}

	if indicators.EnableFundingRate {
		sb.WriteString("- Funding rate\n")
	}

	sb.WriteString("- AI500 / OI_Top filter tags (if available)\n")

	if indicators.EnableQuantData {
		sb.WriteString("- Quantitative data (institutional/retail fund flow, position changes, multi-period price changes)\n")
	}

	sb.WriteString("\n\n")

	return sb.String()
}

// ExtractReasoning extracts the Chain of Thought from the AI response
func (m *Manager) ExtractReasoning(response string) string {
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
