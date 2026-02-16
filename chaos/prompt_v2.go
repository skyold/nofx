package chaos

import (
	"encoding/json"
	"fmt"
	"math"
	"nofx/market"
	"regexp"
	"strings"
	"time"
)

// BuildUserPromptFromChaosContext_v2 使用 ChaosContext 构建 V2 版本的 User Prompt。
// 它采用与 V1 相同的构成逻辑，但使用新的 JSON 格式。
func (e *ChaosEngine) BuildUserPromptFromChaosContext_v2(ctx *ChaosContext) string {
	if ctx == nil {
		return ""
	}
	var sb strings.Builder

	// 添加 Chaos 头部以增加可见性
	sb.WriteString("# 🌀 Chaos 模式用户提示 (V2)\n\n")

	sb.WriteString(e.getTechnicalIndicatorsReference())
	sb.WriteString("\n\n")
	sb.WriteString("---\n\n")

	// 1. 系统状态与环境
	sb.WriteString(e.buildHeader(ctx))

	// 2. 全球市场背景 (BTC)
	sb.WriteString(e.buildGlobalContext(ctx))

	// 3. 账户信息
	sb.WriteString(e.buildAccountStatus(ctx))

	// 4. 交易表现 (统计与历史)
	sb.WriteString(e.buildTradingPerformance(ctx))

	// 5. 当前持仓
	sb.WriteString(e.buildPositions(ctx))

	// 6. 市场数据 (JSON 格式)
	sb.WriteString("## 市场数据 (JSON 格式)\n\n")
	sb.WriteString(e.BuildJsonMarketData(ctx))

	sb.WriteString("---\n\n")

	return sb.String()
}

// BuildJsonMarketData generates the JSON market data part of User Prompt.
func (e *ChaosEngine) BuildJsonMarketData(ctx *ChaosContext) string {
	data := e.buildCompleteMarketData(ctx)
	// Use json.MarshalIndent for pretty printing.
	prettyJSON, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Sprintf("Error building JSON: %v", err)
	}

	// Compress arrays (numeric or string) to single line using regex
	// Numeric pattern: optional sign, digits, optional decimal part, optional exponent
	num := `-?(?:0|[1-9]\d*)(?:\.\d+)?(?:[eE][+-]?\d+)?`
	// String pattern: double quotes surrounding non-quote characters
	str := `"[^"]*"`
	// Value pattern: number or string
	val := `(?:` + num + `|` + str + `)`

	// Pattern: [ space? val (space? , space? val)* space? ]
	pattern := `\[\s*` + val + `(?:\s*,\s*` + val + `)*\s*\]`
	re := regexp.MustCompile(pattern)

	// Regex to match newline and following indentation
	whitespaceRe := regexp.MustCompile(`\n\s*`)

	compactJSON := re.ReplaceAllStringFunc(string(prettyJSON), func(match string) string {
		// Remove newlines and indentation, keeping value content intact
		clean := whitespaceRe.ReplaceAllString(match, "")
		// Add space after comma for readability (safe for these data types)
		clean = strings.ReplaceAll(clean, ",", ", ")
		return clean
	})

	return compactJSON
}

// getTechnicalIndicatorsReference returns the technical indicators reference documentation.
func (e *ChaosEngine) getTechnicalIndicatorsReference() string {
	return `# Chaos Trading System - 技术指标参考

---

## 标准技术指标（无需说明）

以下指标使用行业标准计算方法：
- **EMA20, EMA50**: 指数移动平均线
- **RSI7, RSI14**: 相对强弱指数
- **MACD**: 指数平滑异同移动平均线 (12,26,9)
- **BOLL**: 布林带 (20周期, 2倍标准差)
- **ATR14**: 平均真实波幅

*这些指标仅提供数值，无需额外说明。*

---

## 自定义指标（必须说明）

---

### 1. market_regime (市场环境)

**用途**：分类市场环境，用于策略选择

**取值与含义**：

| 值 | 条件 | 策略 |
|-------|-----------|----------|
| trending_bullish | 价格 > EMA20 > EMA50, ADX > 25 | 跟随趋势，逢低买入 |
| trending_bearish | 价格 < EMA20 < EMA50, ADX > 25 | 跟随趋势，逢高卖出 |
| ranging | ADX < 20 | 均值回归，高抛低吸 |
| transition | 均线交叉或 20 < ADX < 25 | 等待明确，降低仓位 |

**使用建议**：
- 仅在明确的市场环境（trending_* 或 ranging）中交易
- 在 transition 状态时退出/减仓

---

### 2. trend_strength (趋势强度)

**用途**：量化趋势强度 (0-1 范围)

**解读**：

| 范围 | 标签 | 含义 | 仓位大小 |
|-------|-------|---------|---------------|
| 0.00-0.30 | weak | 无明确趋势 | 30-50% |
| 0.30-0.50 | moderate | 趋势正在形成 | 50-70% |
| 0.50-0.70 | strong | 明确趋势 | 70-100% |
| 0.70-1.00 | very_strong | 强劲趋势 | 100%，注意衰竭 |

**使用建议**：
- 仅在 > 0.30 时开新仓
- > 0.50 时用满仓
- > 0.75 时要谨慎（可能超买）

---

### 3. momentum (动量状态)

**用途**：描述价格动量方向和加速度

**取值**：

| 值 | 含义 |
|-------|---------|
| accelerating_up | 强烈向上加速 |
| rising | 平稳上升 |
| choppy | 无明确方向 |
| falling | 平稳下降 |
| accelerating_down | 强烈向下加速 |

**使用建议**：
- accelerating_*：强烈波动，可能超买/超卖
- rising/falling：适合趋势跟随
- choppy：避免方向性交易

---

### 4. volume_price_relationship (量价关系)

**用途**：通过成交量验证价格走势

**取值与含义**：

| 值 | 价格 | 成交量 | 含义 |
|-------|-------|--------|---------|
| bullish_confirmation | 上涨 | 增加 (+20%+) | 强势买入，可持续 |
| weak_rally | 上涨 | 减少 | 缺乏信心，可能失败 |
| bearish_confirmation | 下跌 | 增加 (+20%+) | 强势卖出，可持续 |
| weak_selloff | 下跌 | 减少 | 缺乏抛压，可能反弹 |
| neutral | 任意 | 正常 | 无明确信号 |

**使用建议**：
- 仅在有 *_confirmation 时信任突破
- 在阻力位 fade weak_rally
- 在支撑位 fade weak_selloff

---

### 5. price_position (价格位置)

**用途**：价格在最近 20 根 K 线区间中的位置

**计算**：
position_pct = (当前价格 - 20根K线最低价) / (20根K线最高价 - 20根K线最低价) × 100

**区域**：

| 范围 | 区域 | 解读 |
|-------|------|----------------|
| 80-100% | near_high | 在顶部，考虑止盈 |
| 60-80% | upper_range | 上部区域，注意阻力 |
| 40-60% | mid_range | 中性区域 |
| 20-40% | lower_range | 下部区域，注意支撑 |
| 0-20% | near_low | 在底部，考虑买入 |

---

### 6. volatility_state (波动率状态)

**用途**：评估当前市场波动率

**分类**：

| 值 | 条件 | 行动 |
|-------|-----------|--------|
| very_low | 布林带宽度 < 2%, ATR < 0.8×均值 | 突破即将来临 |
| low | 布林带宽度 < 3%, ATR < 1.2×均值 | 小仓位 |
| medium | 正常条件 | 标准风险 |
| high | 布林带宽度 > 6%, ATR > 1.5×均值 | 扩大止损 |

**趋势**：
- expanding：波动率增加，可能有大波动
- contracting：波动率降低，潜在突破
- stable：稳定波动率

**挤压警报**：
- true：布林带宽度 < 2% 且 contracting → 突破即将来临
- false：正常条件

---

## 决策框架

### 高置信度做多信号

✓ market_regime == "trending_bullish"
✓ trend_strength > 0.50
✓ momentum == "rising" 或 "accelerating_up"
✓ volume_price_relationship == "bullish_confirmation"
✓ price_position.pct < 60% (未超买)
→ 强烈做多

### 高置信度做空信号

✓ market_regime == "trending_bearish"
✓ trend_strength > 0.50
✓ momentum == "falling" 或 "accelerating_down"
✓ volume_price_relationship == "bearish_confirmation"
✓ price_position.pct > 40% (未超卖)
→ 强烈做空

### 避免/等待条件

✗ market_regime == "transition"
✗ trend_strength < 0.30
✗ volume_price_relationship 包含 "weak_"
✗ volatility_state.squeeze_alert == true
→ 不交易，等待明确

---

## 总结表格

| 指标 | 类型 | 需要定义 | 计算方式 |
|-----------|------|------------------|-------------|
| EMA20, EMA50 | 标准 | 否 | 行业标准 |
| RSI7, RSI14 | 标准 | 否 | 行业标准 |
| MACD | 标准 | 否 | 行业标准 |
| BOLL | 标准 | 否 | 行业标准 |
| ATR14 | 标准 | 否 | 行业标准 |
| market_regime | 自定义 | 是 | 基于 ADX + EMA |
| trend_strength | 自定义 | 是 | 复合：ADX + EMA + 动量 |
| momentum | 自定义 | 是 | 5根和10根 vs ATR |
| volume_price_relationship | 自定义 | 是 | 价格变化 + 成交量变化 |
| price_position | 自定义 | 是 | 20根K线区间内的百分比 |
| volatility_state | 自定义 | 是 | ATR比率 + 布林带宽度 |
`
}

// ============================================================================
// Data Structures (JSON Design)
// ============================================================================

type MarketData struct {
	Timestamp      string         `json:"timestamp"`
	Account        AccountInfo    `json:"account"`
	Candidates     []Candidate    `json:"candidates"`
	MarketRankings MarketRankings `json:"market_rankings,omitempty"`
}

type AccountInfo struct {
	Equity         float64 `json:"equity"`
	Balance        float64 `json:"balance"`
	PnlPct         float64 `json:"pnl_pct"`
	MarginUsedPct  float64 `json:"margin_used_pct"`
	PositionsCount int     `json:"positions_count"`
}

type Candidate struct {
	Symbol     string               `json:"symbol"`
	Timeframes map[string]Timeframe `json:"timeframes"`
}

type Timeframe struct {
	Signals    Signals    `json:"signals"`
	Indicators Indicators `json:"indicators"`
	Klines     Klines     `json:"klines"`
}

type Signals struct {
	MarketRegime            string          `json:"market_regime,omitempty"`
	MarketRegimeConfidence  string          `json:"market_regime_confidence,omitempty"`
	TrendStrength           float64         `json:"trend_strength,omitempty"`
	TrendStrengthLabel      string          `json:"trend_strength_label,omitempty"`
	Momentum                string          `json:"momentum,omitempty"`
	VolumePriceRelationship string          `json:"volume_price_relationship,omitempty"`
	PricePosition           PricePosition   `json:"price_position,omitempty"`
	VolatilityState         VolatilityState `json:"volatility_state,omitempty"`
	KeyLevels               KeyLevels       `json:"key_levels,omitempty"`
}

type PricePosition struct {
	PctOfRange float64 `json:"pct_of_range,omitempty"`
	Zone       string  `json:"zone,omitempty"`
}

type VolatilityState struct {
	Classification string `json:"classification,omitempty"`
	Trend          string `json:"trend,omitempty"`
	SqueezeAlert   bool   `json:"squeeze_alert,omitempty"`
}

type KeyLevels struct {
	Resistance   []KeyLevel `json:"resistance,omitempty"`
	Support      []KeyLevel `json:"support,omitempty"`
	CurrentPrice float64    `json:"current_price,omitempty"`
}

type KeyLevel struct {
	Price    float64 `json:"price"`
	Type     string  `json:"type,omitempty"`
	Strength string  `json:"strength,omitempty"`
	Tests    int     `json:"tests,omitempty"`
}

type Indicators struct {
	EMA20     []float64     `json:"ema20,omitempty"`
	EMA50     []float64     `json:"ema50,omitempty"`
	RSI7      []float64     `json:"rsi7,omitempty"`
	RSI14     []float64     `json:"rsi14,omitempty"`
	MACD      MACDData      `json:"macd,omitempty"`
	ATR14     float64       `json:"atr14,omitempty"`
	Bollinger BollingerData `json:"bollinger,omitempty"`
	Volume    []float64     `json:"volume,omitempty"`
}

type MACDData struct {
	Line      []float64 `json:"line,omitempty"`
	Signal    []float64 `json:"signal,omitempty"`
	Histogram []float64 `json:"histogram,omitempty"`
}

type BollingerData struct {
	Upper    []float64 `json:"upper,omitempty"`
	Middle   []float64 `json:"middle,omitempty"`
	Lower    []float64 `json:"lower,omitempty"`
	WidthPct float64   `json:"width_pct,omitempty"`
}

type Klines struct {
	Date            string          `json:"date"`
	Columns         []string        `json:"columns"`
	Values          [][]interface{} `json:"values"`
	CurrentBarIndex int             `json:"current_bar_index"`
}

type MarketRankings struct {
	OIIncrease1h  []RankingItem `json:"oi_increase_1h,omitempty"`
	OIDecrease1h  []RankingItem `json:"oi_decrease_1h,omitempty"`
	FundInflow1h  []RankingItem `json:"fund_inflow_1h,omitempty"`
	FundOutflow1h []RankingItem `json:"fund_outflow_1h,omitempty"`
	TopGainers1h  []RankingItem `json:"top_gainers_1h,omitempty"`
	TopLosers1h   []RankingItem `json:"top_losers_1h,omitempty"`
}

type RankingItem struct {
	Symbol    string  `json:"symbol"`
	OIChange  float64 `json:"oi_change,omitempty"`
	OIPct     float64 `json:"oi_pct,omitempty"`
	PricePct  float64 `json:"price_pct,omitempty"`
	Inflow    float64 `json:"inflow,omitempty"`
	Outflow   float64 `json:"outflow,omitempty"`
	Price     float64 `json:"price,omitempty"`
	FundFlow  float64 `json:"fund_flow,omitempty"`
	ChangePct float64 `json:"change_pct,omitempty"`
}

// ============================================================================
// Builder Logic
// ============================================================================

func (e *ChaosEngine) buildCompleteMarketData(ctx *ChaosContext) MarketData {
	result := MarketData{
		Timestamp: ctx.CurrentTime,
		Account: AccountInfo{
			Equity:         ctx.Account.TotalEquity,
			Balance:        ctx.Account.AvailableBalance,
			PnlPct:         ctx.Account.TotalPnLPct,
			MarginUsedPct:  ctx.Account.MarginUsedPct,
			PositionsCount: ctx.Account.PositionCount,
		},
		Candidates:     []Candidate{},
		MarketRankings: MarketRankings{},
	}

	positionSymbols := make(map[string]bool)
	for _, pos := range ctx.Positions {
		positionSymbols[market.Normalize(pos.Symbol)] = true
	}

	targetTimeframes := ctx.Timeframes
	if len(targetTimeframes) == 0 && ctx.Config != nil && len(ctx.Config.Indicators.Klines.SelectedTimeframes) > 0 {
		targetTimeframes = ctx.Config.Indicators.Klines.SelectedTimeframes
	}
	if len(targetTimeframes) == 0 {
		targetTimeframes = []string{"15m", "1h", "4h"}
	}

	for _, coin := range ctx.CandidateCoins {
		normalizedSymbol := market.Normalize(coin.Symbol)
		if positionSymbols[normalizedSymbol] {
			continue
		}

		marketData, ok := ctx.MarketDataMap[coin.Symbol]
		if !ok {
			continue
		}

		candidate := Candidate{
			Symbol:     coin.Symbol,
			Timeframes: make(map[string]Timeframe),
		}

		for _, tf := range targetTimeframes {
			if tfData, hasData := marketData.TimeframeData[tf]; hasData {
				candidate.Timeframes[tf] = Timeframe{
					Signals:    e.buildSignals(marketData, tfData, ctx.Config),
					Indicators: e.buildIndicators(tfData, ctx.Config),
					Klines:     e.buildKlines(tfData),
				}
			}
		}

		if len(candidate.Timeframes) > 0 {
			result.Candidates = append(result.Candidates, candidate)
		}
	}

	result.MarketRankings = e.buildMarketRankings(ctx)

	return result
}

func (e *ChaosEngine) buildSignals(md *market.Data, tf *market.TimeframeSeriesData, cfg *ChaosConfig) Signals {
	signals := Signals{}

	enableEMA := cfg == nil || cfg.Indicators.EnableEMA
	enableRSI := cfg == nil || cfg.Indicators.EnableRSI

	if enableEMA && len(tf.EMA20Values) > 0 && len(tf.EMA50Values) > 0 {
		ema20 := tf.EMA20Values[len(tf.EMA20Values)-1]
		ema50 := tf.EMA50Values[len(tf.EMA50Values)-1]
		price := md.CurrentPrice

		diffPct := (ema20 - ema50) / ema50 * 100
		if diffPct > 0.5 && price > ema20 {
			signals.MarketRegime = "trending_bullish"
			signals.MarketRegimeConfidence = "high"
			signals.TrendStrength = roundFloat(math.Min(diffPct, 3.0)/3.0, 2)
		} else if diffPct < -0.5 && price < ema20 {
			signals.MarketRegime = "trending_bearish"
			signals.MarketRegimeConfidence = "high"
			signals.TrendStrength = roundFloat(math.Min(-diffPct, 3.0)/3.0, 2)
		} else {
			signals.MarketRegime = "ranging"
			signals.MarketRegimeConfidence = "medium"
			signals.TrendStrength = roundFloat(math.Abs(diffPct)/1.0, 2)
		}

		signals.TrendStrengthLabel = getLabelForTrendStrength(signals.TrendStrength)
	}

	if enableRSI {
		if len(tf.Klines) >= 10 && tf.ATR14 > 0 {
			signals.Momentum = e.calculateMomentum(tf)
		} else {
			if md.PriceChange1h > 0.5 {
				signals.Momentum = "rising"
			} else if md.PriceChange1h < -0.5 {
				signals.Momentum = "falling"
			} else {
				signals.Momentum = "choppy"
			}
		}
	}

	signals.VolumePriceRelationship = e.calculateVolumePriceRelationship(tf)

	if len(tf.Klines) >= 20 {
		high20 := 0.0
		low20 := math.MaxFloat64
		start := len(tf.Klines) - 20
		if start < 0 {
			start = 0
		}
		for i := start; i < len(tf.Klines); i++ {
			if tf.Klines[i].High > high20 {
				high20 = tf.Klines[i].High
			}
			if tf.Klines[i].Low < low20 {
				low20 = tf.Klines[i].Low
			}
		}
		if high20 > low20 {
			pct := (md.CurrentPrice - low20) / (high20 - low20) * 100
			signals.PricePosition = PricePosition{
				PctOfRange: roundFloat(pct, 1),
				Zone:       getZoneForPricePosition(pct),
			}
		}
	}

	signals.VolatilityState = e.calculateVolatilityState(tf)

	signals.KeyLevels = e.buildKeyLevels(md, tf)

	return signals
}

func (e *ChaosEngine) calculateMomentum(tf *market.TimeframeSeriesData) string {
	if len(tf.Klines) < 10 || tf.ATR14 <= 0 {
		return "choppy"
	}

	currentIdx := len(tf.Klines) - 1
	if currentIdx < 10 {
		return "choppy"
	}

	price5barAgo := tf.Klines[currentIdx-5].Close
	price10barAgo := tf.Klines[currentIdx-10].Close
	currentPrice := tf.Klines[currentIdx].Close

	change5bar := (currentPrice - price5barAgo) / price5barAgo * 100
	change10bar := (currentPrice - price10barAgo) / price10barAgo * 100

	atrRatio5 := math.Abs(change5bar) / tf.ATR14
	atrRatio10 := math.Abs(change10bar) / tf.ATR14

	switch {
	case change5bar > 1.5 && change10bar > 1.5 && atrRatio5 > 1.5 && atrRatio10 > 1.5:
		return "accelerating_up"
	case change5bar > 0.5 && change10bar > 0:
		return "rising"
	case change5bar < -1.5 && change10bar < -1.5 && atrRatio5 > 1.5 && atrRatio10 > 1.5:
		return "accelerating_down"
	case change5bar < -0.5 && change10bar < 0:
		return "falling"
	default:
		return "choppy"
	}
}

func (e *ChaosEngine) calculateVolumePriceRelationship(tf *market.TimeframeSeriesData) string {
	if len(tf.Klines) < 2 {
		return "neutral"
	}

	currentIdx := len(tf.Klines) - 1
	prevIdx := currentIdx - 1

	currentKline := tf.Klines[currentIdx]
	prevKline := tf.Klines[prevIdx]

	priceChange := currentKline.Close - prevKline.Close
	volumeChange := 0.0
	if prevKline.Volume > 0 {
		volumeChange = (currentKline.Volume - prevKline.Volume) / prevKline.Volume * 100
	}

	switch {
	case priceChange > 0 && volumeChange >= 20:
		return "bullish_confirmation"
	case priceChange > 0 && volumeChange < 0:
		return "weak_rally"
	case priceChange < 0 && volumeChange >= 20:
		return "bearish_confirmation"
	case priceChange < 0 && volumeChange < 0:
		return "weak_selloff"
	default:
		return "neutral"
	}
}

func (e *ChaosEngine) calculateVolatilityState(tf *market.TimeframeSeriesData) VolatilityState {
	state := VolatilityState{
		Classification: "medium",
		Trend:          "stable",
		SqueezeAlert:   false,
	}

	if len(tf.BOLLUpper) < 2 || len(tf.BOLLMiddle) < 2 || len(tf.BOLLLower) < 2 {
		return state
	}

	idx := len(tf.BOLLMiddle) - 1
	if tf.BOLLMiddle[idx] <= 0 {
		return state
	}

	bbWidth := (tf.BOLLUpper[idx] - tf.BOLLLower[idx]) / tf.BOLLMiddle[idx] * 100
	prevBBWidth := (tf.BOLLUpper[idx-1] - tf.BOLLLower[idx-1]) / tf.BOLLMiddle[idx-1] * 100

	switch {
	case bbWidth < 2:
		state.Classification = "very_low"
	case bbWidth < 3:
		state.Classification = "low"
	case bbWidth > 6:
		state.Classification = "high"
	default:
		state.Classification = "medium"
	}

	widthDiff := bbWidth - prevBBWidth
	switch {
	case widthDiff > 0.5:
		state.Trend = "expanding"
	case widthDiff < -0.5:
		state.Trend = "contracting"
	default:
		state.Trend = "stable"
	}

	state.SqueezeAlert = bbWidth < 2 && state.Trend == "contracting"

	return state
}

func (e *ChaosEngine) buildKeyLevels(md *market.Data, tf *market.TimeframeSeriesData) KeyLevels {
	keyLevels := KeyLevels{
		CurrentPrice: md.CurrentPrice,
	}

	anchors := market.ComputeAnchors(map[string]*market.TimeframeSeriesData{tf.Timeframe: tf})
	for _, a := range anchors {
		if a.Timeframe == tf.Timeframe {
			level := KeyLevel{
				Price:    a.Price,
				Type:     a.Type,
				Strength: "medium",
				Tests:    1,
			}
			if strings.Contains(strings.ToLower(a.Type), "resistance") || strings.Contains(strings.ToLower(a.Type), "high") {
				keyLevels.Resistance = append(keyLevels.Resistance, level)
			} else if strings.Contains(strings.ToLower(a.Type), "support") || strings.Contains(strings.ToLower(a.Type), "low") {
				keyLevels.Support = append(keyLevels.Support, level)
			}
		}
	}

	if md.LocalSupport > 0 {
		keyLevels.Support = append(keyLevels.Support, KeyLevel{
			Price:    md.LocalSupport,
			Type:     "local_support",
			Strength: "strong",
			Tests:    1,
		})
	}
	if md.DailyLow > 0 {
		keyLevels.Support = append(keyLevels.Support, KeyLevel{
			Price:    md.DailyLow,
			Type:     "daily_low",
			Strength: "strong",
			Tests:    1,
		})
	}

	return keyLevels
}

func (e *ChaosEngine) buildIndicators(tf *market.TimeframeSeriesData, cfg *ChaosConfig) Indicators {
	limit := 10
	indicators := Indicators{}

	if cfg == nil || cfg.Indicators.EnableEMA {
		indicators.EMA20 = getLastN(tf.EMA20Values, limit)
		indicators.EMA50 = getLastN(tf.EMA50Values, limit)
	}

	if cfg == nil || cfg.Indicators.EnableRSI {
		indicators.RSI7 = getLastN(tf.RSI7Values, limit)
		indicators.RSI14 = getLastN(tf.RSI14Values, limit)
	}

	if cfg == nil || cfg.Indicators.EnableMACD {
		macdVals := getLastN(tf.MACDValues, limit*3)
		if len(macdVals) >= limit*3 {
			indicators.MACD = MACDData{
				Line:      macdVals[0:limit],
				Signal:    macdVals[limit : 2*limit],
				Histogram: macdVals[2*limit : 3*limit],
			}
		}
	}

	if cfg == nil || cfg.Indicators.EnableATR {
		indicators.ATR14 = roundFloat(tf.ATR14, 4)
	}

	if cfg == nil || cfg.Indicators.EnableBOLL {
		upper := getLastN(tf.BOLLUpper, limit)
		middle := getLastN(tf.BOLLMiddle, limit)
		lower := getLastN(tf.BOLLLower, limit)
		indicators.Bollinger = BollingerData{
			Upper:  upper,
			Middle: middle,
			Lower:  lower,
		}
		if len(middle) > 0 && len(upper) > 0 && len(lower) > 0 && middle[len(middle)-1] > 0 {
			widthPct := (upper[len(upper)-1] - lower[len(lower)-1]) / middle[len(middle)-1] * 100
			indicators.Bollinger.WidthPct = roundFloat(widthPct, 2)
		}
	}

	volumes := make([]float64, 0, limit)
	start := len(tf.Klines) - limit
	if start < 0 {
		start = 0
	}
	for i := start; i < len(tf.Klines); i++ {
		volumes = append(volumes, tf.Klines[i].Volume)
	}
	indicators.Volume = volumes

	return indicators
}

func (e *ChaosEngine) buildKlines(tf *market.TimeframeSeriesData) Klines {
	klines := Klines{
		Columns:         []string{"time", "o", "h", "l", "c", "v"},
		Values:          make([][]interface{}, 0),
		CurrentBarIndex: len(tf.Klines) - 1,
	}

	if len(tf.Klines) > 0 {
		klines.Date = time.Unix(tf.Klines[len(tf.Klines)-1].Time/1000, 0).UTC().Format("2006-01-02")
	}

	startIdx := len(tf.Klines) - 10
	if startIdx < 0 {
		startIdx = 0
	}

	for i := startIdx; i < len(tf.Klines); i++ {
		k := tf.Klines[i]
		tStr := time.Unix(k.Time/1000, 0).UTC().Format("15:04")
		row := []interface{}{
			tStr,
			roundFloat(k.Open, 2),
			roundFloat(k.High, 2),
			roundFloat(k.Low, 2),
			roundFloat(k.Close, 2),
			int(k.Volume),
		}
		klines.Values = append(klines.Values, row)
	}

	return klines
}

func (e *ChaosEngine) buildMarketRankings(ctx *ChaosContext) MarketRankings {
	rankings := MarketRankings{}

	if ctx.OIRankingData != nil {
		for _, item := range ctx.OIRankingData.TopPositions {
			rankings.OIIncrease1h = append(rankings.OIIncrease1h, RankingItem{
				Symbol:   item.Symbol,
				OIChange: item.OIDelta,
				OIPct:    item.OIDeltaPercent,
				PricePct: item.PriceDeltaPercent,
			})
		}
		for _, item := range ctx.OIRankingData.LowPositions {
			rankings.OIDecrease1h = append(rankings.OIDecrease1h, RankingItem{
				Symbol:   item.Symbol,
				OIChange: item.OIDelta,
				OIPct:    item.OIDeltaPercent,
				PricePct: item.PriceDeltaPercent,
			})
		}
	}

	if ctx.PriceRankingData != nil && ctx.PriceRankingData.Durations != nil {
		if data1h, ok := ctx.PriceRankingData.Durations["1h"]; ok {
			for _, item := range data1h.Top {
				rankings.TopGainers1h = append(rankings.TopGainers1h, RankingItem{
					Symbol:    item.Symbol,
					ChangePct: item.PriceDelta * 100,
					Price:     item.Price,
				})
			}
			for _, item := range data1h.Low {
				rankings.TopLosers1h = append(rankings.TopLosers1h, RankingItem{
					Symbol:    item.Symbol,
					ChangePct: item.PriceDelta * 100,
					Price:     item.Price,
				})
			}
		}
	}

	return rankings
}

// ============================================================================
// Helpers
// ============================================================================

func getLabelForTrendStrength(value float64) string {
	switch {
	case value < 0.30:
		return "weak"
	case value < 0.50:
		return "moderate"
	case value < 0.70:
		return "strong"
	default:
		return "very_strong"
	}
}

func getZoneForPricePosition(pct float64) string {
	switch {
	case pct >= 80:
		return "near_high"
	case pct >= 60:
		return "upper_range"
	case pct >= 40:
		return "mid_range"
	case pct >= 20:
		return "lower_range"
	default:
		return "near_low"
	}
}

func getLastN(slice []float64, n int) []float64 {
	if len(slice) == 0 {
		return []float64{}
	}
	if len(slice) <= n {
		res := make([]float64, len(slice))
		for i, v := range slice {
			res[i] = roundFloat(v, 2)
		}
		return res
	}
	raw := slice[len(slice)-n:]
	res := make([]float64, n)
	for i, v := range raw {
		res[i] = roundFloat(v, 2)
	}
	return res
}

func roundFloat(val float64, precision uint) float64 {
	ratio := math.Pow(10, float64(precision))
	return math.Round(val*ratio) / ratio
}
