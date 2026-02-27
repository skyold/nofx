// =============================================================================
// Chaos Trading System - User Prompt Legacy (Text Format)
// =============================================================================
//
// User Prompt 是发送给 LLM 的用户提示词的一部分。
// 完整提示词 = System Prompt + User Prompt
//
// Legacy 特点：
// - 文本格式输出
// - 包含方向性标签（bullish/bearish）
// - K线数据以表格形式展示
// - 适合人工Review
//
// 版本切换：通过配置 prompt_version = "v1" 或 "legacy" 启用
// =============================================================================

package chaos

import (
	"fmt"
	"strings"
	"time"

	"nofx/kernel"
	"nofx/market"
	"nofx/provider/nofxos"
	"nofx/store"
)

const (
	sourceAI500        = "ai500"
	sourceOITop        = "oi_top"
	sourceOILow        = "oi_low"
	sourceStatic       = "static"
	langEnglish        = "en"
	langChinese        = "zh"
	timeFormatUTC      = "01-02 15:04"
	timeFormatTime     = "15:04"
	mainTF1h           = "1h"
	mainTF4h           = "4h"
	mainTF1d           = "1d"
	mainTF5m           = "5m"
	winLossRatioMin    = 0.0
	levelTestThreshold = 0.002
	swingWindowSize    = 3
	maxSwingFeatures   = 5
)

// buildUserPromptV1 使用 ChaosContext 构建用户提示词（Legacy 版本）
//
// 参数说明:
//   - ctx: ChaosContext 上下文对象，包含所有交易相关数据
//
// 返回值:
//   - 完整的用户提示词字符串，包含所有市场数据、账户信息和交易统计
//
// 构建流程：
//  1. Header - 时间、周期、运行时长
//  2. GlobalContext - BTC 行情概览
//  3. AccountStatus - 账户状态
//  4. TradingPerformance - 历史交易统计
//  5. Positions - 当前持仓
//  6. Candidates - 候选币种市场数据 (文本格式)
//  7. Rankings - 排行榜数据 (OI、资金流向、涨跌幅)
func (m *Manager) buildUserPromptV1(ctx *ChaosContext) string {
	if ctx == nil {
		return ""
	}
	var sb strings.Builder

	// 1. 标题
	sb.WriteString("# 🌀 Chaos 模式用户提示 (Legacy - Text Format)\n\n")

	// 2. 公共信息部分
	sb.WriteString(m.buildHeader(ctx))             // 时间、周期、运行时长
	sb.WriteString(m.buildGlobalContext(ctx))      // BTC 行情概览
	sb.WriteString(m.buildAccountStatus(ctx))      // 账户状态
	sb.WriteString(m.buildTradingPerformance(ctx)) // 历史交易统计
	sb.WriteString(m.buildPositions(ctx))          // 当前持仓

	// 3. 市场数据 (文本格式)
	sb.WriteString(m.buildCandidates(ctx)) // 候选币种市场数据
	sb.WriteString(m.buildRankings(ctx))   // 排行榜数据

	sb.WriteString("---\n\n")

	return sb.String()
}

// buildHeader 构建头部信息，包含时间、调用次数和运行时长
//
// 参数说明:
//   - ctx: ChaosContext 上下文对象
//
// 返回值:
//   - 格式化的头部信息字符串
func (m *Manager) buildHeader(ctx *ChaosContext) string {
	return fmt.Sprintf("Time: %s | Period: #%d | Runtime: %d minutes\n\n",
		ctx.CurrentTime, ctx.CallCount, ctx.RuntimeMinutes)
}

// buildGlobalContext 构建 BTC 全局行情概览
//
// 参数说明:
//   - ctx: ChaosContext 上下文对象
//
// 返回值:
//   - BTC 行情概览字符串（如果有 BTC 数据）
func (m *Manager) buildGlobalContext(ctx *ChaosContext) string {
	var sb strings.Builder
	if btcData, hasBTC := ctx.MarketDataMap["BTCUSDT"]; hasBTC {
		sb.WriteString(fmt.Sprintf("BTC: %.2f (1h: %+.2f%%, 4h: %+.2f%%) | MACD: %.4f | RSI: %.2f\n\n",
			btcData.CurrentPrice, btcData.PriceChange1h, btcData.PriceChange4h,
			btcData.CurrentMACD, btcData.CurrentRSI7))
	}
	return sb.String()
}

// buildAccountStatus 构建账户状态信息
//
// 参数说明:
//   - ctx: ChaosContext 上下文对象
//
// 返回值:
//   - 包含账户权益、余额、盈亏、保证金和持仓数量的字符串
func (m *Manager) buildAccountStatus(ctx *ChaosContext) string {
	balancePercent := 0.0
	if ctx.Account.TotalEquity > 0 {
		balancePercent = (ctx.Account.AvailableBalance / ctx.Account.TotalEquity) * 100
	}
	return fmt.Sprintf("Account: Equity %.2f | Balance %.2f (%.1f%%) | PnL %+.2f%% | Margin %.1f%% | Positions %d\n\n",
		ctx.Account.TotalEquity,
		ctx.Account.AvailableBalance,
		balancePercent,
		ctx.Account.TotalPnLPct,
		ctx.Account.MarginUsedPct,
		ctx.Account.PositionCount)
}

// buildTradingPerformance 构建交易性能统计
//
// 参数说明:
//   - ctx: ChaosContext 上下文对象
//
// 返回值:
//   - 包含历史统计和最近交易的字符串
func (m *Manager) buildTradingPerformance(ctx *ChaosContext) string {
	var sb strings.Builder
	sb.WriteString(m.buildHistoricalStats(ctx))
	sb.WriteString(m.buildRecentTrades(ctx))
	return sb.String()
}

// buildRecentTrades 构建最近完成的交易列表
//
// 参数说明:
//   - ctx: ChaosContext 上下文对象
//
// 返回值:
//   - 最近交易列表字符串（如果有交易）
func (m *Manager) buildRecentTrades(ctx *ChaosContext) string {
	if len(ctx.RecentOrders) == 0 {
		return ""
	}
	var sb strings.Builder
	sb.WriteString("## Recent Completed Trades\n")
	for i, order := range ctx.RecentOrders {
		resultStr := "Profit"
		if order.RealizedPnL < 0 {
			resultStr = "Loss"
		}
		sb.WriteString(fmt.Sprintf("%d. %s %s | Entry %.4f Exit %.4f | %s: %+.2f USDT (%+.2f%%) | %s→%s (%s)\n",
			i+1, order.Symbol, order.Side,
			order.EntryPrice, order.ExitPrice,
			resultStr, order.RealizedPnL, order.PnLPct,
			order.EntryTime, order.ExitTime, order.HoldDuration))
	}
	sb.WriteString("\n")
	return sb.String()
}

// buildHistoricalStats 构建历史交易统计
//
// 参数说明:
//   - ctx: ChaosContext 上下文对象
//
// 返回值:
//   - 历史交易统计字符串（支持中英文）
func (m *Manager) buildHistoricalStats(ctx *ChaosContext) string {
	if ctx.TradingStats == nil || ctx.TradingStats.TotalTrades == 0 {
		return ""
	}
	var sb strings.Builder

	lang := "en"

	var winLossRatio float64
	if ctx.TradingStats.AvgLoss > 0 {
		winLossRatio = ctx.TradingStats.AvgWin / ctx.TradingStats.AvgLoss
	}

	if lang == "zh" {
		sb.WriteString("## 历史交易统计\n")
		sb.WriteString(fmt.Sprintf("总交易: %d 笔 | 盈利因子: %.2f | 夏普比率: %.2f | 盈亏比: %.2f\n",
			ctx.TradingStats.TotalTrades,
			ctx.TradingStats.ProfitFactor,
			ctx.TradingStats.SharpeRatio,
			winLossRatio))
		sb.WriteString(fmt.Sprintf("总盈亏: %+.2f USDT | 平均盈利: +%.2f | 平均亏损: -%.2f | 最大回撤: %.1f%%\n",
			ctx.TradingStats.TotalPnL,
			ctx.TradingStats.AvgWin,
			ctx.TradingStats.AvgLoss,
			ctx.TradingStats.MaxDrawdownPct))

		if ctx.TradingStats.ProfitFactor >= 1.5 && ctx.TradingStats.SharpeRatio >= 1 {
			sb.WriteString("表现: 良好 - 保持当前策略\n")
		} else if ctx.TradingStats.ProfitFactor < 1 {
			sb.WriteString("表现: 需改进 - 提高盈亏比，优化止盈止损\n")
		} else if ctx.TradingStats.MaxDrawdownPct > 30 {
			sb.WriteString("表现: 风险偏高 - 减少仓位，控制回撤\n")
		} else {
			sb.WriteString("表现: 正常 - 有优化空间\n")
		}
	} else {
		sb.WriteString("## Historical Trading Statistics\n")
		sb.WriteString(fmt.Sprintf("Total Trades: %d | Profit Factor: %.2f | Sharpe: %.2f | Win/Loss Ratio: %.2f\n",
			ctx.TradingStats.TotalTrades,
			ctx.TradingStats.ProfitFactor,
			ctx.TradingStats.SharpeRatio,
			winLossRatio))
		sb.WriteString(fmt.Sprintf("Total PnL: %+.2f USDT | Avg Win: +%.2f | Avg Loss: -%.2f | Max Drawdown: %.1f%%\n",
			ctx.TradingStats.TotalPnL,
			ctx.TradingStats.AvgWin,
			ctx.TradingStats.AvgLoss,
			ctx.TradingStats.MaxDrawdownPct))

		if ctx.TradingStats.ProfitFactor >= 1.5 && ctx.TradingStats.SharpeRatio >= 1 {
			sb.WriteString("Performance: GOOD - maintain current strategy\n")
		} else if ctx.TradingStats.ProfitFactor < 1 {
			sb.WriteString("Performance: NEEDS IMPROVEMENT - improve win/loss ratio, optimize TP/SL\n")
		} else if ctx.TradingStats.MaxDrawdownPct > 30 {
			sb.WriteString("Performance: HIGH RISK - reduce position size, control drawdown\n")
		} else {
			sb.WriteString("Performance: NORMAL - room for optimization\n")
		}
	}
	sb.WriteString("\n")
	return sb.String()
}

// buildPositions 构建当前持仓列表
//
// 参数说明:
//   - ctx: ChaosContext 上下文对象
//
// 返回值:
//   - 当前持仓列表字符串
func (m *Manager) buildPositions(ctx *ChaosContext) string {
	var sb strings.Builder
	if len(ctx.Positions) > 0 {
		sb.WriteString("## Current Positions\n")
		for i, pos := range ctx.Positions {
			sb.WriteString(m.formatPositionInfoFromContext(i+1, pos, ctx))
		}
	} else {
		sb.WriteString("Current Positions: None\n\n")
	}
	return sb.String()
}

// formatPositionInfoFromContext 格式化单个持仓信息
//
// 参数说明:
//   - index: 持仓索引（从 1 开始）
//   - pos: 持仓信息对象
//   - ctx: ChaosContext 上下文对象
//
// 返回值:
//   - 格式化的持仓信息字符串，包含价格、数量、盈亏、杠杆等
func (m *Manager) formatPositionInfoFromContext(index int, pos kernel.PositionInfo, ctx *ChaosContext) string {
	var sb strings.Builder

	holdingDuration := ""
	if pos.UpdateTime > 0 {
		durationMs := time.Now().UnixMilli() - pos.UpdateTime
		durationMin := durationMs / (1000 * 60)
		if durationMin < 60 {
			holdingDuration = fmt.Sprintf(" | Holding Duration %d min", durationMin)
		} else {
			durationHour := durationMin / 60
			durationMinRemainder := durationMin % 60
			holdingDuration = fmt.Sprintf(" | Holding Duration %dh %dm", durationHour, durationMinRemainder)
		}
	}

	positionValue := pos.Quantity * pos.MarkPrice
	if positionValue < 0 {
		positionValue = -positionValue
	}

	sb.WriteString(fmt.Sprintf("%d. %s %s | Entry %.4f Current %.4f | Qty %.4f | Position Value %.2f USDT | PnL%+.2f%% | PnL Amount%+.2f USDT | Peak PnL%.2f%% | Leverage %dx | Margin %.0f | Liq Price %.4f%s\n\n",
		index, pos.Symbol, strings.ToUpper(pos.Side),
		pos.EntryPrice, pos.MarkPrice, pos.Quantity, positionValue, pos.UnrealizedPnLPct, pos.UnrealizedPnL, pos.PeakPnLPct,
		pos.Leverage, pos.MarginUsed, pos.LiquidationPrice, holdingDuration))

	if marketData, ok := ctx.MarketDataMap[pos.Symbol]; ok {
		sb.WriteString(m.formatMarketData(marketData, ctx.Config.Indicators))

		if ctx.QuantDataMap != nil {
			if quantData, hasQuant := ctx.QuantDataMap[pos.Symbol]; hasQuant {
				sb.WriteString(m.formatQuantData(quantData, ctx.Config.Indicators))
			}
		}
		sb.WriteString("\n")
	}

	return sb.String()
}

// buildCandidates 构建候选币种列表
//
// 参数说明:
//   - ctx: ChaosContext 上下文对象
//
// 返回值:
//   - 候选币种的市场数据字符串（排除已有持仓的币种）
func (m *Manager) buildCandidates(ctx *ChaosContext) string {
	var sb strings.Builder

	positionSymbols := make(map[string]bool)
	for _, pos := range ctx.Positions {
		positionSymbols[market.Normalize(pos.Symbol)] = true
	}

	sb.WriteString(fmt.Sprintf("## Candidate Coins (%d coins)\n\n", len(ctx.MarketDataMap)))
	displayedCount := 0

	for _, coin := range ctx.CandidateCoins {
		normalizedCoinSymbol := market.Normalize(coin.Symbol)
		if positionSymbols[normalizedCoinSymbol] {
			continue
		}

		marketData, hasData := ctx.MarketDataMap[coin.Symbol]
		if !hasData {
			continue
		}
		displayedCount++

		sourceTags := m.formatCoinSourceTag(coin.Sources)
		sb.WriteString(fmt.Sprintf("### %d. %s%s\n\n", displayedCount, coin.Symbol, sourceTags))
		sb.WriteString(m.formatMarketData(marketData, ctx.Config.Indicators))

		if ctx.QuantDataMap != nil {
			if quantData, hasQuant := ctx.QuantDataMap[coin.Symbol]; hasQuant {
				sb.WriteString(m.formatQuantData(quantData, ctx.Config.Indicators))
			}
		}
		sb.WriteString("\n")
	}
	sb.WriteString("\n")
	return sb.String()
}

// buildRankings 构建排行榜数据
//
// 参数说明:
//   - ctx: ChaosContext 上下文对象
//
// 返回值:
//   - OI、资金流向和价格涨跌幅排行榜数据字符串
func (m *Manager) buildRankings(ctx *ChaosContext) string {
	var sb strings.Builder
	nofxosLang := nofxos.LangEnglish

	if ctx.OIRankingData != nil {
		sb.WriteString(nofxos.FormatOIRankingForAI(ctx.OIRankingData, nofxosLang))
	}

	if ctx.NetFlowRankingData != nil {
		sb.WriteString(nofxos.FormatNetFlowRankingForAI(ctx.NetFlowRankingData, nofxosLang))
	}

	if ctx.PriceRankingData != nil {
		sb.WriteString(nofxos.FormatPriceRankingForAI(ctx.PriceRankingData, nofxosLang))
	}
	return sb.String()
}

// formatCoinSourceTag 格式化币种来源标签
//
// 参数说明:
//   - sources: 来源字符串数组
//
// 返回值:
//   - 格式化的来源标签字符串，如 (AI500)、(OI_Top 持仓增加) 等
func (m *Manager) formatCoinSourceTag(sources []string) string {
	if len(sources) > 1 {
		hasAI500 := false
		hasOITop := false
		hasOILow := false
		for _, s := range sources {
			switch s {
			case sourceAI500:
				hasAI500 = true
			case sourceOITop:
				hasOITop = true
			case sourceOILow:
				hasOILow = true
			}
		}
		if hasAI500 && hasOITop {
			return " (AI500+OI_Top dual signal)"
		}
		if hasAI500 && hasOILow {
			return " (AI500+OI_Low dual signal)"
		}
		if hasOITop && hasOILow {
			return " (OI_Top+OI_Low)"
		}
		return " (Multiple sources)"
	} else if len(sources) == 1 {
		switch sources[0] {
		case sourceAI500:
			return " (AI500)"
		case sourceOITop:
			return " (OI_Top 持仓增加)"
		case sourceOILow:
			return " (OI_Low 持仓减少)"
		case sourceStatic:
			return " (Manual selection)"
		}
	}
	return ""
}

// ============================================================================
// Market Data Formatting
// ============================================================================

// formatMarketData 格式化市场数据
//
// 参数说明:
//   - data: 市场数据对象
//   - indicators: 指标配置
//
// 返回值:
//   - 格式化的市场数据字符串，包含价格、技术指标、K线等
func (m *Manager) formatMarketData(data *market.Data, indicators store.IndicatorConfig) string {
	var sb strings.Builder
	// indicators are passed as argument

	sb.WriteString(fmt.Sprintf("=== %s Market Data ===\n\n", data.Symbol))
	sb.WriteString(fmt.Sprintf("current_price = %.4f", data.CurrentPrice))

	if indicators.EnableEMA {
		sb.WriteString(fmt.Sprintf(", current_ema20 = %.3f", data.CurrentEMA20))
	}

	if indicators.EnableMACD {
		sb.WriteString(fmt.Sprintf(", current_macd = %.3f", data.CurrentMACD))
	}

	if indicators.EnableRSI {
		sb.WriteString(fmt.Sprintf(", current_rsi7 = %.3f", data.CurrentRSI7))
	}

	sb.WriteString("\n\n")

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

	if indicators.EnableOI || indicators.EnableFundingRate {
		sb.WriteString(fmt.Sprintf("Additional data for %s:\n\n", data.Symbol))

		if indicators.EnableOI && data.OpenInterest != nil {
			sb.WriteString(fmt.Sprintf("Open Interest: Latest: %.2f Average: %.2f\n\n",
				data.OpenInterest.Latest, data.OpenInterest.Average))
		}

		if indicators.EnableFundingRate {
			sb.WriteString(fmt.Sprintf("Funding Rate: %.2e\n\n", data.FundingRate))
		}
	}

	if len(data.TimeframeData) > 0 {
		timeframeOrder := []string{"1m", "3m", "5m", "15m", "30m", "1h", "2h", "4h", "6h", "8h", "12h", "1d", "3d", "1w"}
		for _, tf := range timeframeOrder {
			if tfData, ok := data.TimeframeData[tf]; ok {
				sb.WriteString(fmt.Sprintf("=== %s Timeframe (oldest → latest) ===\n\n", strings.ToUpper(tf)))
				m.formatTimeframeSeriesData(&sb, tfData, indicators)
			}
		}
		var mainTf string
		if _, ok := data.TimeframeData[mainTF1h]; ok {
			mainTf = mainTF1h
		} else if _, ok := data.TimeframeData[mainTF4h]; ok {
			mainTf = mainTF4h
		}

		// 1. Structural Anchors (Swings with Topology)
		if mainTf != "" {
			tfData := data.TimeframeData[mainTf]
			// Use window 3 for better sensitivity with limited data
			features := GenerateTechnicalFeatures(tfData.Klines, swingWindowSize)
			sb.WriteString(features.FormatFeaturesToText(maxSwingFeatures))
		}

		// 2. Major Boundaries (Daily Levels)
		tf1d, ok := data.TimeframeData[mainTF1d]
		if ok && len(tf1d.Klines) > 0 {
			// Use yesterday's completed candle if available (more reliable "Major" level)
			var last market.KlineBar
			var dayDesc string

			if len(tf1d.Klines) >= 2 {
				last = tf1d.Klines[len(tf1d.Klines)-2]
				dayDesc = "Yesterday's Daily"
			} else {
				last = tf1d.Klines[len(tf1d.Klines)-1]
				dayDesc = "Today's Intraday"
			}

			sb.WriteString("### Range Boundaries & Tests:\n")

			// Daily Low (Major Support)
			supportLevel := last.Low
			supportTests := 0
			if mainTf != "" {
				// Count tests on the main timeframe (e.g. 1h)
				supportTests = countLevelTests(supportLevel, SwingLow, data.TimeframeData[mainTf].Klines, 0, levelTestThreshold)
			}
			sb.WriteString(fmt.Sprintf("- [Major Support]: %s | Tested: %d times (%s Low)\n",
				formatPriceForPrompt(supportLevel), supportTests, dayDesc))

			// Daily High (Major Resistance)
			resistanceLevel := last.High
			resistanceTests := 0
			if mainTf != "" {
				resistanceTests = countLevelTests(resistanceLevel, SwingHigh, data.TimeframeData[mainTf].Klines, 0, levelTestThreshold)
			}
			sb.WriteString(fmt.Sprintf("- [Major Resistance]: %s | Tested: %d times (%s High)\n",
				formatPriceForPrompt(resistanceLevel), resistanceTests, dayDesc))

			sb.WriteString("\n")
		} else {
			// Fallback: use existing ComputeAnchors if explicit feature generation failed or no 1d data?
			// But ComputeAnchors also depends on data.
			// If we entered here, we might have skipped GenerateTechnicalFeatures if no 1h/4h.
			// Let's keep the old logic as a fallback if mainTf is empty.
			if mainTf == "" {
				anchors := market.ComputeAnchors(data.TimeframeData)
				if len(anchors) > 0 {
					sb.WriteString("### 物理结构锚点 (Physical Structural Anchors):\n")
					for _, a := range anchors {
						priceStr := formatPriceForPrompt(a.Price)
						t := time.Unix(a.Time/1000, 0).UTC().Format("01-02 15:04")
						src := ""
						switch a.Timeframe {
						case "1d":
							if a.Type == "Major Support" {
								src = "24H Daily Low"
							} else {
								src = "24H Daily High"
							}
						case "1h", "4h":
							src = a.Timeframe + " Structure"
						default:
							src = a.Timeframe + " Pivot"
						}
						sb.WriteString(fmt.Sprintf("- [%s]: %s (%s, %s)\n", a.Type, priceStr, src, t))
					}
					sb.WriteString("\n")
				}
			}
		}

		// Dynamic References (BB, EMA)
		refParts := []string{}
		if tf, ok := data.TimeframeData[mainTF5m]; ok && len(tf.BOLLLower) > 0 {
			refParts = append(refParts, fmt.Sprintf("BB_Lower (5M): %s", formatPriceForPrompt(tf.BOLLLower[len(tf.BOLLLower)-1])))
		}
		if tf, ok := data.TimeframeData[mainTF1h]; ok && len(tf.EMA50Values) > 0 {
			refParts = append(refParts, fmt.Sprintf("EMA50 (1H): %.4f", tf.EMA50Values[len(tf.EMA50Values)-1]))
		}
		if len(refParts) > 0 {
			sb.WriteString("### 动态参考 (Dynamic References):\n")
			for _, p := range refParts {
				sb.WriteString(fmt.Sprintf("- %s\n", p))
			}
			sb.WriteString("\n")
		}
	} else {
		if data.IntradaySeries != nil {
			klineConfig := indicators.Klines
			sb.WriteString(fmt.Sprintf("Intraday series (%s intervals, oldest → latest):\n\n", klineConfig.PrimaryTimeframe))

			if len(data.IntradaySeries.MidPrices) > 0 {
				sb.WriteString(fmt.Sprintf("Mid prices: %s\n\n", formatFloatSlice(data.IntradaySeries.MidPrices)))
			}

			if indicators.EnableEMA && len(data.IntradaySeries.EMA20Values) > 0 {
				sb.WriteString(fmt.Sprintf("EMA indicators (20-period): %s\n\n", formatFloatSlice(data.IntradaySeries.EMA20Values)))
			}

			if indicators.EnableMACD && len(data.IntradaySeries.MACDValues) > 0 {
				sb.WriteString(fmt.Sprintf("MACD indicators: %s\n\n", formatFloatSlice(data.IntradaySeries.MACDValues)))
			}

			if indicators.EnableRSI {
				if len(data.IntradaySeries.RSI7Values) > 0 {
					sb.WriteString(fmt.Sprintf("RSI indicators (7-Period): %s\n\n", formatFloatSlice(data.IntradaySeries.RSI7Values)))
				}
				if len(data.IntradaySeries.RSI14Values) > 0 {
					sb.WriteString(fmt.Sprintf("RSI indicators (14-Period): %s\n\n", formatFloatSlice(data.IntradaySeries.RSI14Values)))
				}
			}

			if indicators.EnableVolume && len(data.IntradaySeries.Volume) > 0 {
				sb.WriteString(fmt.Sprintf("Volume: %s\n\n", formatFloatSlice(data.IntradaySeries.Volume)))
			}

			if indicators.EnableATR {
				sb.WriteString(fmt.Sprintf("3m ATR (14-period): %.3f\n\n", data.IntradaySeries.ATR14))
			}
		}

		if data.LongerTermContext != nil && indicators.Klines.EnableMultiTimeframe {
			sb.WriteString(fmt.Sprintf("Longer-term context (%s timeframe):\n\n", indicators.Klines.LongerTimeframe))

			if indicators.EnableEMA {
				sb.WriteString(fmt.Sprintf("20-Period EMA: %.3f vs. 50-Period EMA: %.3f\n\n",
					data.LongerTermContext.EMA20, data.LongerTermContext.EMA50))
			}

			if indicators.EnableATR {
				sb.WriteString(fmt.Sprintf("3-Period ATR: %.3f vs. 14-Period ATR: %.3f\n\n",
					data.LongerTermContext.ATR3, data.LongerTermContext.ATR14))
			}

			if indicators.EnableVolume {
				sb.WriteString(fmt.Sprintf("Current Volume: %.3f vs. Average Volume: %.3f\n\n",
					data.LongerTermContext.CurrentVolume, data.LongerTermContext.AverageVolume))
			}

			if indicators.EnableMACD && len(data.LongerTermContext.MACDValues) > 0 {
				sb.WriteString(fmt.Sprintf("MACD indicators: %s\n\n", formatFloatSlice(data.LongerTermContext.MACDValues)))
			}

			if indicators.EnableRSI && len(data.LongerTermContext.RSI14Values) > 0 {
				sb.WriteString(fmt.Sprintf("RSI indicators (14-Period): %s\n\n", formatFloatSlice(data.LongerTermContext.RSI14Values)))
			}
		}
	}

	return sb.String()
}

// formatTimeframeSeriesData 格式化单个时间周期的序列数据
//
// 参数说明:
//   - sb: strings.Builder 对象，用于写入格式化后的数据
//   - data: 时间周期序列数据
//   - indicators: 指标配置
func (m *Manager) formatTimeframeSeriesData(sb *strings.Builder, data *market.TimeframeSeriesData, indicators store.IndicatorConfig) {
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
	} else if len(data.MidPrices) > 0 {
		sb.WriteString(fmt.Sprintf("Mid prices: %s\n\n", formatFloatSlice(data.MidPrices)))
		if indicators.EnableVolume && len(data.Volume) > 0 {
			sb.WriteString(fmt.Sprintf("Volume: %s\n\n", formatFloatSlice(data.Volume)))
		}
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
		} else if data.ATR14 > 0 {
			sb.WriteString(fmt.Sprintf("ATR14: %.4f\n", data.ATR14))
		}
	}

	if indicators.EnableBOLL && len(data.BOLLUpper) > 0 {
		sb.WriteString(fmt.Sprintf("BOLL Upper: %s\n", formatFloatSlice(data.BOLLUpper)))
		sb.WriteString(fmt.Sprintf("BOLL Middle: %s\n", formatFloatSlice(data.BOLLMiddle)))
		sb.WriteString(fmt.Sprintf("BOLL Lower: %s\n", formatFloatSlice(data.BOLLLower)))
	}

	sb.WriteString("\n")
}

// formatQuantData 格式化量化数据
//
// 参数说明:
//   - data: 量化数据对象
//   - indicators: 指标配置
//
// 返回值:
//   - 格式化的量化数据字符串，包含价格变化、资金流向、持仓量变化等
func (m *Manager) formatQuantData(data *kernel.QuantData, indicators store.IndicatorConfig) string {
	if data == nil {
		return ""
	}

	if !indicators.EnableQuantOI && !indicators.EnableQuantNetflow {
		return ""
	}

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("📊 %s Quantitative Data:\n", data.Symbol))

	if len(data.PriceChange) > 0 {
		sb.WriteString("Price Change: ")
		timeframes := []string{"5m", "15m", "1h", "4h", "12h", "24h"}
		parts := []string{}
		for _, tf := range timeframes {
			if v, ok := data.PriceChange[tf]; ok {
				parts = append(parts, fmt.Sprintf("%s: %+.4f%%", tf, v*100))
			}
		}
		sb.WriteString(strings.Join(parts, " | "))
		sb.WriteString("\n")
	}

	if indicators.EnableQuantNetflow && data.Netflow != nil {
		sb.WriteString("Fund Flow (Netflow):\n")
		timeframes := []string{"5m", "15m", "1h", "4h", "12h", "24h"}

		if data.Netflow.Institution != nil {
			if data.Netflow.Institution.Future != nil && len(data.Netflow.Institution.Future) > 0 {
				sb.WriteString("  Institutional Futures:\n")
				for _, tf := range timeframes {
					if v, ok := data.Netflow.Institution.Future[tf]; ok {
						sb.WriteString(fmt.Sprintf("    %s: %s\n", tf, formatFlowValue(v)))
					}
				}
			}
			if data.Netflow.Institution.Spot != nil && len(data.Netflow.Institution.Spot) > 0 {
				sb.WriteString("  Institutional Spot:\n")
				for _, tf := range timeframes {
					if v, ok := data.Netflow.Institution.Spot[tf]; ok {
						sb.WriteString(fmt.Sprintf("    %s: %s\n", tf, formatFlowValue(v)))
					}
				}
			}
		}

		if data.Netflow.Personal != nil {
			if data.Netflow.Personal.Future != nil && len(data.Netflow.Personal.Future) > 0 {
				sb.WriteString("  Retail Futures:\n")
				for _, tf := range timeframes {
					if v, ok := data.Netflow.Personal.Future[tf]; ok {
						sb.WriteString(fmt.Sprintf("    %s: %s\n", tf, formatFlowValue(v)))
					}
				}
			}
			if data.Netflow.Personal.Spot != nil && len(data.Netflow.Personal.Spot) > 0 {
				sb.WriteString("  Retail Spot:\n")
				for _, tf := range timeframes {
					if v, ok := data.Netflow.Personal.Spot[tf]; ok {
						sb.WriteString(fmt.Sprintf("    %s: %s\n", tf, formatFlowValue(v)))
					}
				}
			}
		}
	}

	if indicators.EnableQuantOI && len(data.OI) > 0 {
		for exchange, oiData := range data.OI {
			if len(oiData.Delta) > 0 {
				sb.WriteString(fmt.Sprintf("Open Interest (%s):\n", exchange))
				for _, tf := range []string{"5m", "15m", "1h", "4h", "12h", "24h"} {
					if d, ok := oiData.Delta[tf]; ok {
						sb.WriteString(fmt.Sprintf("    %s: %+.4f%% (%s)\n", tf, d.OIDeltaPercent, formatFlowValue(d.OIDeltaValue)))
					}
				}
			}
		}
	}

	return sb.String()
}

// formatFlowValue 格式化资金流向数值（带单位）
//
// 参数说明:
//   - v: 资金流向数值
//
// 返回值:
//   - 格式化的字符串，如 +1.23M、-45.67K 等
func formatFlowValue(v float64) string {
	sign := ""
	if v >= 0 {
		sign = "+"
	}
	absV := v
	if absV < 0 {
		absV = -absV
	}
	if absV >= 1e9 {
		return fmt.Sprintf("%s%.2fB", sign, v/1e9)
	} else if absV >= 1e6 {
		return fmt.Sprintf("%s%.2fM", sign, v/1e6)
	} else if absV >= 1e3 {
		return fmt.Sprintf("%s%.2fK", sign, v/1e3)
	}
	return fmt.Sprintf("%s%.2f", sign, v)
}

// formatPriceForPrompt 根据价格大小格式化价格字符串
//
// 参数说明:
//   - price: 价格数值
//
// 返回值:
//   - 根据价格范围返回不同精度的格式化字符串
func formatPriceForPrompt(price float64) string {
	switch {
	case price < 0.0001:
		return fmt.Sprintf("%.8f", price)
	case price < 0.001:
		return fmt.Sprintf("%.6f", price)
	case price < 0.01:
		return fmt.Sprintf("%.6f", price)
	case price < 1.0:
		return fmt.Sprintf("%.4f", price)
	case price < 100:
		return fmt.Sprintf("%.4f", price)
	default:
		return fmt.Sprintf("%.2f", price)
	}
}

// formatFloatSlice 格式化浮点数数组
//
// 参数说明:
//   - values: 浮点数数组
//
// 返回值:
//   - 格式化为 [1.2345, 6.7890, ...] 形式的字符串
func formatFloatSlice(values []float64) string {
	strValues := make([]string, len(values))
	for i, v := range values {
		strValues[i] = fmt.Sprintf("%.4f", v)
	}
	return "[" + strings.Join(strValues, ", ") + "]"
}
