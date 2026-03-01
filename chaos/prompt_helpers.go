// =============================================================================
// Chaos Trading System - Prompt Helpers
// =============================================================================
//
// 辅助函数和常量定义
// 供 Builder 和 Formatter 使用
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

// ============================================================================
// 常量定义
// ============================================================================

const (
	swingWindowSize    = 3
	maxSwingFeatures   = 5
	levelTestThreshold = 0.002
	mainTF1h           = "1h"
	mainTF4h           = "4h"
	mainTF1d           = "1d"
	mainTF5m           = "5m"
	timeFormatUTC      = "01-02 15:04"
	timeFormatTime     = "15:04"
)

// ============================================================================
// V1/V2 需要的辅助函数
// ============================================================================

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

// buildTradingPerformance 构建交易性能统计
func (m *Manager) buildTradingPerformance(ctx *ChaosContext) string {
	var sb strings.Builder
	sb.WriteString(m.buildHistoricalStats(ctx))
	sb.WriteString(m.buildRecentTrades(ctx))
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
		sb.WriteString(m.formatMarketData(marketData, ctx.Config.Indicators, ctx.Config.UserPromptVersion))

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
		sb.WriteString(m.formatMarketData(marketData, ctx.Config.Indicators, ctx.Config.UserPromptVersion))

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

// formatMarketData 根据版本选择正确的格式化函数
func (m *Manager) formatMarketData(data *market.Data, indicators store.IndicatorConfig, version string) string {
	switch version {
	case "v2":
		return m.formatMarketDataV2(data, indicators)
	case "v1", "legacy", "":
		return m.formatMarketDataV1(data, indicators)
	default:
		// 默认使用 V1
		return m.formatMarketDataV1(data, indicators)
	}
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

	// 简化处理，详细实现可以参考原代码
	return sb.String()
}
