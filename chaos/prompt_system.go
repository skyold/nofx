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
	"strings"

	"nofx/store"
)

// 常量定义 - 消除魔法字符串
const (
	variantNone    = "none"
	variantDefault = "default"
	placeholderFmt = "{%s}"

	// 参数键名常量
	paramStructureValidationTF = "STRUCTURE_VALIDATION_TF"
	paramPrimaryTimeframe      = "PRIMARY_TIMEFRAME"
	paramEntryTimeframe        = "ENTRY_TIMEFRAME"
	paramTimeDecayN            = "TIME_DECAY_N"
	paramMinRR                 = "MIN_RR"
)

// BuildSystemPrompt 构建 Chaos 模式的系统提示词
//
// 参数说明:
//   - variant: 配置变体名称，用于获取预定义的交易参数
//   - customPrompt: 用户自定义的提示词模板，可包含占位符如 {PRIMARY_TIMEFRAME}
//   - indicators: 市场数据和技术指标的配置信息
//
// 返回值:
//   - 完整的系统提示词字符串，包含输出格式规范、系统参数、自定义提示词和可用数据列表
//
// 功能流程:
//  1. 首先写入输出格式规范（GenerateOutputSchema）
//  2. 根据 variant 获取对应的参数配置
//  3. 将 customPrompt 中的占位符替换为实际参数值
//  4. 如果 variant 不是 none/default/空，且 customPrompt 中没有占位符，则写入强制执行的系统参数
//  5. 写入替换后的自定义提示词
//  6. 写入可用的市场数据和技术指标列表
//  7. 返回最终构建的提示词
func (m *Manager) BuildSystemPrompt(variant string, customPrompt string, indicators store.IndicatorConfig) string {
	var sb strings.Builder

	// 1. 写入输出格式规范
	sb.WriteString(GenerateOutputSchema())
	sb.WriteString("\n\n")

	// 2. 获取配置变体对应的参数
	params := m.GetVariantParams(variant)
	v := strings.ToLower(strings.TrimSpace(variant))

	// 3. 替换自定义提示词中的占位符
	finalPrompt := customPrompt
	for key, val := range params {
		placeholder := fmt.Sprintf(placeholderFmt, key)
		finalPrompt = strings.ReplaceAll(finalPrompt, placeholder, val)
	}

	// 4. 如果配置变体有效且提示词中没有占位符，则写入强制执行的系统参数
	if v != variantNone && v != variantDefault && v != "" {
		hasPlaceholders := hasAnyPlaceholder(customPrompt, params)

		if !hasPlaceholders {
			writeSystemParameters(&sb, params)
		}
	}

	// 5. 写入替换后的自定义提示词
	sb.WriteString(finalPrompt)

	// 6. 写入可用的市场数据和技术指标列表
	sb.WriteString("\n\n你拥有以下市场数据和指标:\n")

	// 收集 K线周期，使用 map 去重
	klines := collectUniqueKlines(indicators)
	if len(klines) > 0 {
		sb.WriteString(fmt.Sprintf("- %s K-line series\n", strings.Join(klines, " + ")))
	} else {
		sb.WriteString("- K-line series\n")
	}

	// 根据配置写入技术指标
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

// hasAnyPlaceholder 检查 customPrompt 中是否包含 params 中任意键的占位符
func hasAnyPlaceholder(customPrompt string, params map[string]string) bool {
	for key := range params {
		placeholder := fmt.Sprintf(placeholderFmt, key)
		if strings.Contains(customPrompt, placeholder) {
			return true
		}
	}
	return false
}

// writeSystemParameters 写入强制执行的系统参数
func writeSystemParameters(sb *strings.Builder, params map[string]string) {
	sb.WriteString("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n")
	sb.WriteString("SYSTEM PARAMETERS — 强制执行，不可覆盖\n")
	sb.WriteString("━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━\n\n")

	// 安全地获取参数值，带默认值
	writeParam(sb, paramStructureValidationTF, params[paramStructureValidationTF])
	writeParam(sb, paramPrimaryTimeframe, params[paramPrimaryTimeframe])
	writeParam(sb, paramEntryTimeframe, params[paramEntryTimeframe])
	writeParam(sb, paramTimeDecayN, params[paramTimeDecayN])
	writeParam(sb, paramMinRR, params[paramMinRR])

	sb.WriteString("\n")
	sb.WriteString("所有决策必须以上述参数为准。\n")
	sb.WriteString("若市场数据与参数定义不符，以参数为准，不得自行调整。\n\n")
}

// writeParam 写入单个参数，带对齐格式
func writeParam(sb *strings.Builder, key, value string) {
	sb.WriteString(fmt.Sprintf("%-24s = %s\n", key, value))
}

// collectUniqueKlines 收集并去重 K线周期
func collectUniqueKlines(indicators store.IndicatorConfig) []string {
	seen := make(map[string]bool)
	klines := []string{}

	// 添加主时间周期
	if indicators.Klines.PrimaryTimeframe != "" {
		klines = append(klines, indicators.Klines.PrimaryTimeframe)
		seen[indicators.Klines.PrimaryTimeframe] = true
	}

	// 添加多时间周期（去重）
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

// writeIndicatorWithPeriods 写入带周期参数的技术指标
func writeIndicatorWithPeriods(sb *strings.Builder, enabled bool, name string, periods interface{}) {
	if enabled {
		sb.WriteString(fmt.Sprintf("- %s indicators (periods: %v)\n", name, periods))
	}
}

// writeSimpleIndicator 写入简单的技术指标（无周期参数）
func writeSimpleIndicator(sb *strings.Builder, enabled bool, name string) {
	if enabled {
		sb.WriteString(fmt.Sprintf("- %s data\n", name))
	}
}
