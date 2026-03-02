// =============================================================================
// Chaos Trading System - Text Formatter
// =============================================================================
//
// Formatter 模式：负责将数据格式化为文本格式
// 用于 V1 和 V2 版本
// =============================================================================

package chaos

import (
	"fmt"
	"strings"
	"time"

	"nofx/store"
)

// ============================================================================
// TextFormatter - 文本格式器
// ============================================================================

// TextFormatter 文本格式器
// 将 MarketPromptData 格式化为文本格式
type TextFormatter struct {
	IncludeRegime              bool // 是否包含 Regime 分类器
	IncludeLLMBriefing         bool // 是否包含 LLM 简报
}

// FormatMarketData 格式化市场数据为文本
func (f *TextFormatter) FormatMarketData(data *MarketPromptData, indicators store.IndicatorConfig) string {
	var sb strings.Builder

	// =========================================================================
	// 第一部分：基础价格和技术指标
	// =========================================================================
	sb.WriteString(fmt.Sprintf("=== %s Market Data ===\n\n", data.Symbol))
	sb.WriteString(fmt.Sprintf("current_price = %.4f", data.CurrentPrice))

	// 从时间周期数据中获取指标
	if indicators.EnableEMA || indicators.EnableMACD || indicators.EnableRSI {
		for _, tfData := range data.Timeframes {
			if len(tfData.EMA20Values) > 0 && indicators.EnableEMA {
				sb.WriteString(fmt.Sprintf(", current_ema20 = %.3f", tfData.EMA20Values[len(tfData.EMA20Values)-1]))
			}
			if len(tfData.MACDValues) > 0 && indicators.EnableMACD {
				sb.WriteString(fmt.Sprintf(", current_macd = %.3f", tfData.MACDValues[len(tfData.MACDValues)-1]))
			}
			if len(tfData.RSI7Values) > 0 && indicators.EnableRSI {
				sb.WriteString(fmt.Sprintf(", current_rsi7 = %.3f", tfData.RSI7Values[len(tfData.RSI7Values)-1]))
			}
			break // 只取第一个时间周期的最新值
		}
	}

	sb.WriteString("\n\n")

	// =========================================================================
	// 第二部分：局部支撑位和日内低点
	// =========================================================================
	if data.LocalSupport > 0 || data.DailyLow > 0 {
		localSupportStr := ""
		if data.LocalSupport > 0 {
			timePart := ""
			if data.LocalSupportTime > 0 {
				timePart = fmt.Sprintf(" (%s Low)", time.Unix(data.LocalSupportTime/1000, 0).UTC().Format("15:04"))
			}
			localSupportStr = fmt.Sprintf("Local_Support: %s%s", formatPriceForPrompt(data.LocalSupport), timePart)
		}
		dailyLowStr := ""
		if data.DailyLow > 0 {
			dailyLowStr = fmt.Sprintf("Daily_Low: %s", formatPriceForPrompt(data.DailyLow))
		}

		switch {
		case localSupportStr != "" && dailyLowStr != "":
			sb.WriteString(localSupportStr + ", " + dailyLowStr + "\n\n")
		case localSupportStr != "":
			sb.WriteString(localSupportStr + "\n\n")
		case dailyLowStr != "":
			sb.WriteString(dailyLowStr + "\n\n")
		}
	}

	// =========================================================================
	// 第三部分：附加数据（OI、资金费率、机构分类器、物理结构锚点）
	// =========================================================================
	if data.OpenInterest != nil || data.FundingRate != 0 || data.Regime != nil {
		sb.WriteString(fmt.Sprintf("Additional data for %s:\n\n", data.Symbol))

		// 3.1 持仓量数据
		if data.OpenInterest != nil {
			sb.WriteString(fmt.Sprintf("Open Interest: Latest: %.2f Average: %.2f\n\n",
				data.OpenInterest.Latest, data.OpenInterest.Average))
		}

		// 3.2 资金费率
		if data.FundingRate != 0 {
			sb.WriteString(fmt.Sprintf("Funding Rate: %.2e\n\n", data.FundingRate))
		}

		// 3.3 Regime 分类器
		if f.IncludeRegime && data.Regime != nil {
			sb.WriteString(data.Regime.FormatToText())

			// 3.3.1 LLM 战术简报
			if f.IncludeLLMBriefing && data.LLMBriefing != "" {
				sb.WriteString(data.LLMBriefing)
			}
		}

		// 3.4 物理结构锚点（摆动点 + 拓扑标记）
		if data.SwingFeatures != nil {
			sb.WriteString(data.SwingFeatures.FormatFeaturesToText(maxSwingFeatures))
		}
	}

	// =========================================================================
	// 第四部分：多时间周期 K 线数据
	// =========================================================================
	if len(data.Timeframes) > 0 {
		timeframeOrder := []string{"1m", "3m", "5m", "15m", "30m", "1h", "2h", "4h", "6h", "8h", "12h", "1d", "3d", "1w"}
		for _, tf := range timeframeOrder {
			if tfData, ok := data.Timeframes[tf]; ok {
				sb.WriteString(fmt.Sprintf("=== %s Timeframe (oldest → latest) ===\n\n", strings.ToUpper(tf)))
				f.formatTimeframeSeriesData(&sb, tfData, indicators)
			}
		}

		// =====================================================================
		// 第五部分：主要区间边界和测试次数
		// =====================================================================
		if data.MajorSupport > 0 || data.MajorResistance > 0 {
			sb.WriteString("### Range Boundaries & Tests:\n")

			if data.MajorSupport > 0 {
				dayDesc := "Yesterday's Daily"
				sb.WriteString(fmt.Sprintf("- [Major Support]: %s | Tested: %d times (%s Low)\n",
					formatPriceForPrompt(data.MajorSupport), data.SupportTests, dayDesc))
			}

			if data.MajorResistance > 0 {
				dayDesc := "Yesterday's Daily"
				sb.WriteString(fmt.Sprintf("- [Major Resistance]: %s | Tested: %d times (%s High)\n",
					formatPriceForPrompt(data.MajorResistance), data.ResistanceTests, dayDesc))
			}

			sb.WriteString("\n")
		}

		// =====================================================================
		// 第六部分：动态参考
		// =====================================================================
		if len(data.DynamicReferences) > 0 {
			sb.WriteString("### 动态参考 (Dynamic References):\n")
			for _, p := range data.DynamicReferences {
				sb.WriteString(fmt.Sprintf("- %s\n", p))
			}
			sb.WriteString("\n")
		}
	}

	return sb.String()
}

// formatTimeframeSeriesData 格式化单个时间周期的序列数据
func (f *TextFormatter) formatTimeframeSeriesData(sb *strings.Builder, data *TimeframeData, indicators store.IndicatorConfig) {
	if len(data.Klines) > 0 {
		sb.WriteString("Time(UTC)      Open      High      Low       Close     Volume\n")
		for i, k := range data.Klines {
			t := time.Unix(k.Time/1000, 0).UTC()
			timeStr := t.Format(timeFormatUTC)
			marker := ""
			if i == len(data.Klines)-1 {
				marker = "  <- current"
			}
			sb.WriteString(fmt.Sprintf("%-14s %-9.4f %-9.4f %-9.4f %-9.4f %-12.2f%s\n",
				timeStr, k.Open, k.High, k.Low, k.Close, k.Volume, marker))
		}
		sb.WriteString("\n")
	}

	if indicators.EnableEMA {
		if len(data.EMA20Values) > 0 {
			sb.WriteString(fmt.Sprintf("EMA20: %s\n", formatFloatSlice(data.EMA20Values)))
		}
		if len(data.EMA50Values) > 0 {
			sb.WriteString(fmt.Sprintf("EMA50: %s\n", formatFloatSlice(data.EMA50Values)))
		}
	}

	if indicators.EnableMACD && len(data.MACDValues) > 0 {
		sb.WriteString(fmt.Sprintf("MACD: %s\n", formatFloatSlice(data.MACDValues)))
	}

	if indicators.EnableRSI {
		if len(data.RSI7Values) > 0 {
			sb.WriteString(fmt.Sprintf("RSI7: %s\n", formatFloatSlice(data.RSI7Values)))
		}
		if len(data.RSI14Values) > 0 {
			sb.WriteString(fmt.Sprintf("RSI14: %s\n", formatFloatSlice(data.RSI14Values)))
		}
	}

	if indicators.EnableATR {
		if len(data.ATR14Values) > 0 {
			sb.WriteString(fmt.Sprintf("ATR14: %s\n", formatFloatSlice(data.ATR14Values)))
		}
	}

	if indicators.EnableBOLL && len(data.BOLLUpper) > 0 {
		sb.WriteString(fmt.Sprintf("BOLL Upper: %s\n", formatFloatSlice(data.BOLLUpper)))
		sb.WriteString(fmt.Sprintf("BOLL Middle: %s\n", formatFloatSlice(data.BOLLMiddle)))
		sb.WriteString(fmt.Sprintf("BOLL Lower: %s\n", formatFloatSlice(data.BOLLLower)))
	}

	sb.WriteString("\n")
}
