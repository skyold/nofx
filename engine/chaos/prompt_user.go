// =============================================================================
// Chaos Trading System - User Prompt Builder
// =============================================================================
//
// User Prompt 是发送给 LLM 的用户提示词的一部分。
// 完整提示词 = System Prompt + User Prompt
//
// 此文件负责构建 User Prompt，包含：
// 1. 公共信息 (头部、全局上下文、账户状态等)
// 2. 目标交易对数据 (基础市场数据 + 技术指标 + 信号数据)
// 3. 市场排行榜数据
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

// BuildUserPrompt 构建 User Prompt
//
// 构建流程:
//  1. 标题
//  2. 公共信息 (头部、全局上下文、账户状态、交易表现、持仓)
//  3. 目标交易对数据 (市场数据 + 量化数据)
//  4. 市场排行榜数据
func (m *Manager) BuildUserPrompt(ctx *ChaosContext) string {
	if ctx == nil {
		return ""
	}

	var sb strings.Builder

	// 1. 标题
	sb.WriteString("# 🌀 Chaos 模式用户提示\n\n")

	// 2. 公共信息
	sb.WriteString(m.buildHeader(ctx))
	sb.WriteString(m.buildGlobalContext(ctx))
	sb.WriteString(m.buildAccountStatus(ctx))
	sb.WriteString(m.buildHistoricalStats(ctx))
	sb.WriteString(m.buildRecentTrades(ctx))
	sb.WriteString(m.buildPositions(ctx))

	// 3. 目标交易对数据 (包括基础市场数据 + 指标数据 + 信号数据)
	sb.WriteString(m.buildTargetSymbols(ctx))

	// 4. 市场排行榜数据
	sb.WriteString(m.buildMarketRankings(ctx))

	sb.WriteString("---\n\n")

	return sb.String()
}

// buildHeader 构建头部信息
func (m *Manager) buildHeader(ctx *ChaosContext) string {
	return fmt.Sprintf("Time: %s | Period: #%d | Runtime: %d minutes\n\n",
		ctx.CurrentTime, ctx.CallCount, ctx.RuntimeMinutes)
}

// buildGlobalContext 构建 BTC 全局行情概览
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

// buildHistoricalStats 构建历史交易统计
func (m *Manager) buildHistoricalStats(ctx *ChaosContext) string {
	if ctx.TradingStats == nil || ctx.TradingStats.TotalTrades == 0 {
		return ""
	}
	var sb strings.Builder

	var winLossRatio float64
	if ctx.TradingStats.AvgLoss > 0 {
		winLossRatio = ctx.TradingStats.AvgWin / ctx.TradingStats.AvgLoss
	}

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
	sb.WriteString("\n")
	return sb.String()
}

// buildRecentTrades 构建最近完成的交易列表
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

// buildPositions 构建当前持仓列表
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

// buildTargetSymbols 构建目标交易对数据
// 包含：基础市场数据 + 技术指标 + 加工信号
// 数据来源：ctx.MarketDataMap 中排除已持仓的交易对
func (m *Manager) buildTargetSymbols(ctx *ChaosContext) string {
	var sb strings.Builder

	// 收集已持仓的币种 (避免重复展示)
	positionSymbols := make(map[string]bool)
	for _, pos := range ctx.Positions {
		positionSymbols[market.Normalize(pos.Symbol)] = true
	}

	sb.WriteString(fmt.Sprintf("## Target Symbols (%d symbols)\n\n", len(ctx.MarketDataMap)))
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

// buildMarketRankings 构建市场排行榜数据
func (m *Manager) buildMarketRankings(ctx *ChaosContext) string {
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
func (m *Manager) formatCoinSourceTag(sources []string) string {
	if len(sources) > 1 {
		hasAI500 := false
		hasOITop := false
		hasOILow := false
		for _, s := range sources {
			switch s {
			case "ai500":
				hasAI500 = true
			case "oi_top":
				hasOITop = true
			case "oi_low":
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
		case "ai500":
			return " (AI500)"
		case "oi_top":
			return " (OI_Top 持仓增加)"
		case "oi_low":
			return " (OI_Low 持仓减少)"
		case "static":
			return " (Manual selection)"
		}
	}
	return ""
}

// formatMarketData 格式化市场数据为文本
// 直接从 market.Data 读取并格式化，不再使用 Builder/Formatter 模式
func (m *Manager) formatMarketData(data *market.Data, indicators store.IndicatorConfig) string {
	var sb strings.Builder

	// === 第一部分：基础价格和技术指标 ===
	sb.WriteString(fmt.Sprintf("=== %s Market Data ===\n\n", data.Symbol))
	sb.WriteString(fmt.Sprintf("current_price = %.4f", data.CurrentPrice))

	// 从时间周期数据中获取指标
	if indicators.EnableEMA || indicators.EnableMACD || indicators.EnableRSI {
		for _, tfData := range data.TimeframeData {
			if len(tfData.EMA20Values) > 0 && indicators.EnableEMA {
				sb.WriteString(fmt.Sprintf(", current_ema20 = %.3f", tfData.EMA20Values[len(tfData.EMA20Values)-1]))
			}
			if len(tfData.MACDValues) > 0 && indicators.EnableMACD {
				sb.WriteString(fmt.Sprintf(", current_macd = %.3f", tfData.MACDValues[len(tfData.MACDValues)-1]))
			}
			if len(tfData.RSI7Values) > 0 && indicators.EnableRSI {
				sb.WriteString(fmt.Sprintf(", current_rsi7 = %.3f", tfData.RSI7Values[len(tfData.RSI7Values)-1]))
			}
			break
		}
	}

	sb.WriteString("\n\n")

	// === 第二部分：局部支撑位和日内低点 ===
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

	// === 第三部分：附加数据 (OI、资金费率) ===
	if data.OpenInterest != nil || data.FundingRate != 0 {
		sb.WriteString(fmt.Sprintf("Additional data for %s:\n\n", data.Symbol))

		if data.OpenInterest != nil {
			sb.WriteString(fmt.Sprintf("Open Interest: Latest: %.2f 5-Period-Ago: %.2f\n\n",
				data.OpenInterest.Latest, data.OpenInterest.Before5Period))
		}

		if data.FundingRate != 0 {
			sb.WriteString(fmt.Sprintf("Funding Rate: %.2e\n\n", data.FundingRate))
		}
	}

	// === 第四部分：多时间周期 K 线数据 ===
	if len(data.TimeframeData) > 0 {
		timeframeOrder := []string{"1m", "3m", "5m", "15m", "30m", "1h", "2h", "4h", "6h", "8h", "12h", "1d", "3d", "1w"}
		for _, tf := range timeframeOrder {
			if tfData, ok := data.TimeframeData[tf]; ok {
				sb.WriteString(fmt.Sprintf("=== %s Timeframe (oldest → latest) ===\n\n", strings.ToUpper(tf)))
				m.formatTimeframeSeriesData(&sb, tfData, indicators)
			}
		}
	}

	return sb.String()
}

// formatTimeframeSeriesData 格式化单个时间周期的序列数据
func (m *Manager) formatTimeframeSeriesData(sb *strings.Builder, data *market.TimeframeSeriesData, indicators store.IndicatorConfig) {
	if len(data.Klines) > 0 {
		sb.WriteString("Time(UTC)      Open      High      Low       Close     Volume\n")
		for i, k := range data.Klines {
			t := time.Unix(k.Time/1000, 0).UTC()
			timeStr := t.Format("01-02 15:04")
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

// formatQuantData 格式化量化数据
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

	return sb.String()
}
