// =============================================================================
// Chaos Trading System - User Prompt V4 (Structural Facts Edition)
// =============================================================================
//
// User Prompt 是发送给 LLM 的用户提示词的一部分。
// 完整提示词 = System Prompt + User Prompt
//
// V4 特点：
// - 纯结构事实（无方向词）
// - 无综合评分（无conviction_score）
// - 完全可验证（每个值都可复算）
// - 判断权在LLM（不预判方向）
// - 6层 Signals 结构：Structure, Momentum, Volatility, Liquidity, Positioning, Ranking
//
// 版本切换：通过配置 prompt_version = "v4" 启用
// =============================================================================

package chaos

import (
	"encoding/json"
	"fmt"
	"math"
	"nofx/market"
	"regexp"
	"sort"
	"strings"
	"time"
)

// buildUserPromptV4 uses ChaosContext to build V4 version User Prompt.
// V4 provides pure structural facts, letting LLM do real structural reasoning.
//
// 构建流程：
//  1. V4 技术指标参考文档 (getV4TechnicalIndicatorsReference) - 6层 Signals 结构说明
//  2. Header - 时间、周期、运行时长
//  3. GlobalContext - BTC 行情概览
//  4. AccountStatus - 账户状态
//  5. TradingPerformance - 历史交易统计
//  6. Positions - 当前持仓
//  7. 市场数据 JSON (V4) - 候选币种多时间框架数据 + 6层结构事实 Signals
func (m *Manager) buildUserPromptV4(ctx *ChaosContext) string {
	if ctx == nil {
		return ""
	}
	var sb strings.Builder

	// 1. 标题
	sb.WriteString("# 🌀 Chaos 模式用户提示 (V4 - Structural Facts Edition)\n\n")

	// 2. V4 技术指标参考文档 (6层 Signals 结构说明)
	sb.WriteString(m.getV4TechnicalIndicatorsReference())
	sb.WriteString("\n\n")
	sb.WriteString("---\n\n")

	// 3. 公共信息部分
	sb.WriteString(m.buildHeader(ctx))             // 时间、周期、运行时长
	sb.WriteString(m.buildGlobalContext(ctx))      // BTC 行情概览
	sb.WriteString(m.buildAccountStatus(ctx))      // 账户状态
	sb.WriteString(m.buildTradingPerformance(ctx)) // 历史交易统计
	sb.WriteString(m.buildPositions(ctx))          // 当前持仓

	// 4. 市场数据 (JSON 格式 - V4 结构事实版)
	sb.WriteString("## 市场数据 (JSON 格式 - V4 结构事实版)\n\n")
	sb.WriteString(m.BuildV4JsonMarketData(ctx))

	sb.WriteString("---\n\n")

	return sb.String()
}

// getV4TechnicalIndicatorsReference returns the V4 technical indicators reference documentation.
func (m *Manager) getV4TechnicalIndicatorsReference() string {
	return `# Chaos Trading System V4 - 技术指标参考
## Structural Facts Edition (结构事实版本)

---

## 🎯 V4 设计哲学

**核心原则**：Signal只负责"计算外包"，不负责"方向判断"

- ❌ **禁止方向词**（bullish/bearish/long/short）
- ❌ **禁止综合评分**（conviction_score/trend_score）
- ✅ **只提供结构事实**（higher_high_count/rsi_value）
- ✅ **完全可验证**（每个值都可复算）
- ✅ **判断权在LLM**（不预判方向）

---

## 📐 Signal 6层分层结构

### 1️⃣ Structure Layer (结构层)
- higher_high_count_N: 最近N根K线创新高次数
- lower_low_count_N: 最近N根K线创新低次数
- structure_break_high: 是否突破N根高点
- structure_break_low: 是否突破N根低点
- range_compression_ratio: 区间压缩度
- swing_high_count_N: N根内明显高点数
- swing_low_count_N: N根内明显低点数

### 2️⃣ Momentum Layer (动量层)
- rsi_value: RSI原始值
- rsi_percentile_200: RSI在200根中的百分位
- rsi_over_70: RSI是否超过70
- rsi_below_30: RSI是否低于30
- ema20_above_ema50: EMA20是否在EMA50上方
- ema20_slope: EMA20斜率
- ema50_slope: EMA50斜率
- ema_distance_percent: EMA距离百分比
- macd_line_value: MACD线值
- macd_signal_value: MACD信号线值
- macd_histogram_value: MACD柱状图值
- macd_histogram_positive: MACD柱状图是否为正
- consecutive_up_bars: 连续阳线数
- consecutive_down_bars: 连续阴线数
- up_bars_count_N: 最近N根阳线数
- down_bars_count_N: 最近N根阴线数

### 3️⃣ Volatility Layer (波动层)
- atr_value: ATR值
- atr_percentile_200: ATR在200根中的百分位
- body_ratio: 实体占比
- bb_width_percent: BB宽度百分比
- bb_width_percentile_200: BB宽度百分位
- price_bb_position: 价格在BB中的位置(0-100)

### 4️⃣ Liquidity Layer (流动性层)
- liquidity_sweep_high: 是否扫了高点
- liquidity_sweep_low: 是否扫了低点
- volume_value: 当前成交量
- volume_percentile_200: 成交量百分位
- volume_spike: 是否成交量突增
- volume_ma_ratio: 成交量/MA比率
- price_volume_sync: 价格和成交量方向是否一致

### 5️⃣ Positioning Layer (持仓结构层) - 可选
- long_short_ratio: 多空持仓比
- long_account_percent: 多头账户占比
- short_account_percent: 空头账户占比
- oi_value: 当前持仓量
- oi_change_percent: 持仓量变化百分比
- oi_change_percentile_100: 持仓量变化百分位
- funding_rate: 资金费率
- funding_rate_percentile_100: 资金费率百分位

### 6️⃣ Ranking Layer (横向排名层) - 可选
- volume_rank_24h: 24小时成交量排名
- volume_rank_total_symbols: 总币种数
- volatility_rank_24h: 24小时波动率排名
- volatility_rank_total_symbols: 总币种数
- relative_strength_rank_24h: 24小时涨幅排名
- price_change_percent_24h: 24小时价格变化百分比

---

## 💡 V4 vs V2/Legacy

Legacy: LLM作为规则匹配器 → "看到'bullish'，考虑做多"
V2: LLM作为规则匹配器 → "看到'bullish'，考虑做多"
V4: LLM作为结构推理器 → "HH=3 vs LL=1 + RSI=68 → 可能上升结构"

**V4是真正的"结构推理时代"！**
`
}

// ============================================================================
// V4 Data Structures
// ============================================================================

type V4MarketData struct {
	Timestamp  string        `json:"timestamp"`
	Account    AccountInfo   `json:"account"`
	Candidates []V4Candidate `json:"candidates"`
	SignalMeta V4SignalMeta  `json:"signal_meta,omitempty"`
}

type V4Candidate struct {
	Symbol     string                 `json:"symbol"`
	Timeframes map[string]V4Timeframe `json:"timeframes"`
}

type V4Timeframe struct {
	Signals    V4Signals  `json:"signals"`
	Indicators Indicators `json:"indicators"`
	Klines     Klines     `json:"klines"`
}

type V4Signals struct {
	Structure   V4StructureSignals   `json:"structure"`
	Momentum    V4MomentumSignals    `json:"momentum"`
	Volatility  V4VolatilitySignals  `json:"volatility"`
	Liquidity   V4LiquiditySignals   `json:"liquidity"`
	Positioning V4PositioningSignals `json:"positioning,omitempty"`
	Ranking     V4RankingSignals     `json:"ranking,omitempty"`
}

type V4StructureSignals struct {
	HigherHighCount5      int     `json:"higher_high_count_5"`
	HigherHighCount10     int     `json:"higher_high_count_10"`
	LowerLowCount5        int     `json:"lower_low_count_5"`
	LowerLowCount10       int     `json:"lower_low_count_10"`
	StructureBreakHigh    bool    `json:"structure_break_high"`
	StructureBreakLow     bool    `json:"structure_break_low"`
	RangeCompressionRatio float64 `json:"range_compression_ratio"`
	SwingHighCount20      int     `json:"swing_high_count_20"`
	SwingLowCount20       int     `json:"swing_low_count_20"`
}

type V4MomentumSignals struct {
	RsiValue              float64 `json:"rsi_value"`
	RsiPercentile200      int     `json:"rsi_percentile_200"`
	RsiOver70             bool    `json:"rsi_over_70"`
	RsiBelow30            bool    `json:"rsi_below_30"`
	Ema20AboveEma50       bool    `json:"ema20_above_ema50"`
	Ema20Slope            float64 `json:"ema20_slope"`
	Ema50Slope            float64 `json:"ema50_slope"`
	EmaDistancePercent    float64 `json:"ema_distance_percent"`
	MacdLineValue         float64 `json:"macd_line_value"`
	MacdSignalValue       float64 `json:"macd_signal_value"`
	MacdHistogramValue    float64 `json:"macd_histogram_value"`
	MacdHistogramPositive bool    `json:"macd_histogram_positive"`
	ConsecutiveUpBars     int     `json:"consecutive_up_bars"`
	ConsecutiveDownBars   int     `json:"consecutive_down_bars"`
	UpBarsCount10         int     `json:"up_bars_count_10"`
	DownBarsCount10       int     `json:"down_bars_count_10"`
}

type V4VolatilitySignals struct {
	AtrValue             float64 `json:"atr_value"`
	AtrPercentile200     int     `json:"atr_percentile_200"`
	BodyRatio            float64 `json:"body_ratio"`
	BbWidthPercent       float64 `json:"bb_width_percent"`
	BbWidthPercentile200 int     `json:"bb_width_percentile_200"`
	PriceBbPosition      float64 `json:"price_bb_position"`
}

type V4LiquiditySignals struct {
	LiquiditySweepHigh  bool    `json:"liquidity_sweep_high"`
	LiquiditySweepLow   bool    `json:"liquidity_sweep_low"`
	VolumeValue         float64 `json:"volume_value"`
	VolumePercentile200 int     `json:"volume_percentile_200"`
	VolumeSpike         bool    `json:"volume_spike"`
	VolumeMaRatio       float64 `json:"volume_ma_ratio"`
	PriceVolumeSync     bool    `json:"price_volume_sync"`
}

type V4PositioningSignals struct {
	LongShortRatio           float64 `json:"long_short_ratio,omitempty"`
	LongAccountPercent       float64 `json:"long_account_percent,omitempty"`
	ShortAccountPercent      float64 `json:"short_account_percent,omitempty"`
	OiValue                  float64 `json:"oi_value,omitempty"`
	OiChangePercent          float64 `json:"oi_change_percent,omitempty"`
	OiChangePercentile100    int     `json:"oi_change_percentile_100,omitempty"`
	FundingRate              float64 `json:"funding_rate,omitempty"`
	FundingRatePercentile100 int     `json:"funding_rate_percentile_100,omitempty"`
}

type V4RankingSignals struct {
	VolumeRank24h              int     `json:"volume_rank_24h,omitempty"`
	VolumeRankTotalSymbols     int     `json:"volume_rank_total_symbols,omitempty"`
	VolatilityRank24h          int     `json:"volatility_rank_24h,omitempty"`
	VolatilityRankTotalSymbols int     `json:"volatility_rank_total_symbols,omitempty"`
	RelativeStrengthRank24h    int     `json:"relative_strength_rank_24h,omitempty"`
	PriceChangePercent24h      float64 `json:"price_change_percent_24h,omitempty"`
}

type V4SignalMeta struct {
	Version           string              `json:"version"`
	LookbackPeriods   V4LookbackPeriods   `json:"lookback_periods,omitempty"`
	IndicatorPeriods  V4IndicatorPeriods  `json:"indicator_periods,omitempty"`
	PercentileWindows V4PercentileWindows `json:"percentile_windows,omitempty"`
	Thresholds        V4Thresholds        `json:"thresholds,omitempty"`
}

type V4LookbackPeriods struct {
	StructureBreak   int `json:"structure_break"`
	RangeCompression int `json:"range_compression"`
	SwingDetection   int `json:"swing_detection"`
	LiquiditySweep   int `json:"liquidity_sweep"`
}

type V4IndicatorPeriods struct {
	Rsi        int `json:"rsi"`
	EmaShort   int `json:"ema_short"`
	EmaLong    int `json:"ema_long"`
	Atr        int `json:"atr"`
	Bollinger  int `json:"bollinger"`
	MacdFast   int `json:"macd_fast"`
	MacdSlow   int `json:"macd_slow"`
	MacdSignal int `json:"macd_signal"`
}

type V4PercentileWindows struct {
	Rsi         int `json:"rsi"`
	Atr         int `json:"atr"`
	Volume      int `json:"volume"`
	BbWidth     int `json:"bb_width"`
	OiChange    int `json:"oi_change"`
	FundingRate int `json:"funding_rate"`
}

type V4Thresholds struct {
	RsiOverbought         float64 `json:"rsi_overbought"`
	RsiOversold           float64 `json:"rsi_oversold"`
	VolumeSpikeRatio      float64 `json:"volume_spike_ratio"`
	RangeCompressionTight float64 `json:"range_compression_tight"`
	RangeCompressionWide  float64 `json:"range_compression_wide"`
}

// BuildV4JsonMarketData generates the V4 JSON market data part of User Prompt.
func (m *Manager) BuildV4JsonMarketData(ctx *ChaosContext) string {
	data := m.buildV4CompleteMarketData(ctx)
	prettyJSON, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Sprintf("Error building JSON: %v", err)
	}

	num := `-?(?:0|[1-9]\d*)(?:\.\d+)?(?:[eE][+-]?\d+)?`
	str := `"[^"]*"`
	val := `(?:` + num + `|` + str + `)`
	pattern := `\[\s*` + val + `(?:\s*,\s*` + val + `)*\s*\]`
	re := regexp.MustCompile(pattern)
	whitespaceRe := regexp.MustCompile(`\n\s*`)

	compactJSON := re.ReplaceAllStringFunc(string(prettyJSON), func(match string) string {
		clean := whitespaceRe.ReplaceAllString(match, "")
		clean = strings.ReplaceAll(clean, ",", ", ")
		return clean
	})

	return compactJSON
}

func (m *Manager) buildV4CompleteMarketData(ctx *ChaosContext) V4MarketData {
	result := V4MarketData{
		Timestamp: ctx.CurrentTime,
		Account: AccountInfo{
			Equity:         ctx.Account.TotalEquity,
			Balance:        ctx.Account.AvailableBalance,
			PnlPct:         ctx.Account.TotalPnLPct,
			MarginUsedPct:  ctx.Account.MarginUsedPct,
			PositionsCount: ctx.Account.PositionCount,
		},
		Candidates: []V4Candidate{},
		SignalMeta: m.buildV4SignalMeta(),
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

		candidate := V4Candidate{
			Symbol:     coin.Symbol,
			Timeframes: make(map[string]V4Timeframe),
		}

		for _, tf := range targetTimeframes {
			if tfData, hasData := marketData.TimeframeData[tf]; hasData {
				candidate.Timeframes[tf] = V4Timeframe{
					Signals:    m.buildV4Signals(marketData, tfData, ctx),
					Indicators: m.buildV4Indicators(tfData, ctx.Config),
					Klines:     m.buildV4Klines(tfData),
				}
			}
		}

		if len(candidate.Timeframes) > 0 {
			result.Candidates = append(result.Candidates, candidate)
		}
	}

	return result
}

func (m *Manager) buildV4SignalMeta() V4SignalMeta {
	return V4SignalMeta{
		Version: "4.0",
		LookbackPeriods: V4LookbackPeriods{
			StructureBreak:   20,
			RangeCompression: 20,
			SwingDetection:   2,
			LiquiditySweep:   10,
		},
		IndicatorPeriods: V4IndicatorPeriods{
			Rsi:        14,
			EmaShort:   20,
			EmaLong:    50,
			Atr:        14,
			Bollinger:  20,
			MacdFast:   12,
			MacdSlow:   26,
			MacdSignal: 9,
		},
		PercentileWindows: V4PercentileWindows{
			Rsi:         200,
			Atr:         200,
			Volume:      200,
			BbWidth:     200,
			OiChange:    100,
			FundingRate: 100,
		},
		Thresholds: V4Thresholds{
			RsiOverbought:         70,
			RsiOversold:           30,
			VolumeSpikeRatio:      1.5,
			RangeCompressionTight: 2.0,
			RangeCompressionWide:  4.0,
		},
	}
}

func (m *Manager) buildV4Signals(md *market.Data, tf *market.TimeframeSeriesData, ctx *ChaosContext) V4Signals {
	return V4Signals{
		Structure:   m.buildV4StructureSignals(tf),
		Momentum:    m.buildV4MomentumSignals(tf, md),
		Volatility:  m.buildV4VolatilitySignals(tf, md),
		Liquidity:   m.buildV4LiquiditySignals(tf, md),
		Positioning: m.buildV4PositioningSignals(md),
		Ranking:     m.buildV4RankingSignals(md, ctx),
	}
}

func (m *Manager) buildV4StructureSignals(tf *market.TimeframeSeriesData) V4StructureSignals {
	signals := V4StructureSignals{}

	if len(tf.Klines) < 5 {
		return signals
	}

	signals.HigherHighCount5 = countHigherHigh(tf.Klines, 5)
	signals.LowerLowCount5 = countLowerLow(tf.Klines, 5)

	if len(tf.Klines) >= 10 {
		signals.HigherHighCount10 = countHigherHigh(tf.Klines, 10)
		signals.LowerLowCount10 = countLowerLow(tf.Klines, 10)
	}

	lookback := 20
	if len(tf.Klines) >= lookback+1 {
		lookbackHigh := 0.0
		lookbackLow := math.MaxFloat64
		for i := len(tf.Klines) - lookback - 1; i < len(tf.Klines)-1; i++ {
			if tf.Klines[i].High > lookbackHigh {
				lookbackHigh = tf.Klines[i].High
			}
			if tf.Klines[i].Low < lookbackLow {
				lookbackLow = tf.Klines[i].Low
			}
		}
		currentKline := tf.Klines[len(tf.Klines)-1]
		signals.StructureBreakHigh = currentKline.High > lookbackHigh
		signals.StructureBreakLow = currentKline.Low < lookbackLow

		rangeWidth := lookbackHigh - lookbackLow
		if tf.ATR14 > 0 {
			signals.RangeCompressionRatio = roundFloat(rangeWidth/tf.ATR14, 2)
		}
	}

	signals.SwingHighCount20 = countSwingHigh(tf.Klines, 20, 2)
	signals.SwingLowCount20 = countSwingLow(tf.Klines, 20, 2)

	return signals
}

func (m *Manager) buildV4MomentumSignals(tf *market.TimeframeSeriesData, md *market.Data) V4MomentumSignals {
	signals := V4MomentumSignals{}

	if len(tf.RSI14Values) > 0 {
		rsiValue := tf.RSI14Values[len(tf.RSI14Values)-1]
		signals.RsiValue = roundFloat(rsiValue, 1)
		signals.RsiOver70 = rsiValue > 70
		signals.RsiBelow30 = rsiValue < 30

		if len(tf.RSI14Values) >= 200 {
			signals.RsiPercentile200 = calculatePercentile(rsiValue, tf.RSI14Values[len(tf.RSI14Values)-200:])
		} else if len(tf.RSI14Values) > 1 {
			signals.RsiPercentile200 = calculatePercentile(rsiValue, tf.RSI14Values)
		}
	}

	if len(tf.EMA20Values) > 0 && len(tf.EMA50Values) > 0 {
		ema20 := tf.EMA20Values[len(tf.EMA20Values)-1]
		ema50 := tf.EMA50Values[len(tf.EMA50Values)-1]
		signals.Ema20AboveEma50 = ema20 > ema50

		if len(tf.EMA20Values) >= 2 {
			signals.Ema20Slope = roundFloat(ema20-tf.EMA20Values[len(tf.EMA20Values)-2], 2)
		}
		if len(tf.EMA50Values) >= 2 {
			signals.Ema50Slope = roundFloat(ema50-tf.EMA50Values[len(tf.EMA50Values)-2], 2)
		}

		if ema50 != 0 && md.CurrentPrice > 0 {
			signals.EmaDistancePercent = roundFloat(math.Abs(ema20-ema50)/md.CurrentPrice*100, 2)
		}
	}

	if len(tf.MACDValues) >= 30 {
		limit := 10
		signals.MacdLineValue = roundFloat(tf.MACDValues[0:limit][limit-1], 2)
		signals.MacdSignalValue = roundFloat(tf.MACDValues[limit : 2*limit][limit-1], 2)
		signals.MacdHistogramValue = roundFloat(tf.MACDValues[2*limit : 3*limit][limit-1], 2)
		signals.MacdHistogramPositive = signals.MacdHistogramValue > 0
	}

	if len(tf.Klines) >= 10 {
		signals.ConsecutiveUpBars, signals.ConsecutiveDownBars = countConsecutiveBars(tf.Klines)
		signals.UpBarsCount10, signals.DownBarsCount10 = countBarsInPeriod(tf.Klines, 10)
	}

	return signals
}

func (m *Manager) buildV4VolatilitySignals(tf *market.TimeframeSeriesData, md *market.Data) V4VolatilitySignals {
	signals := V4VolatilitySignals{}

	if tf.ATR14 > 0 {
		signals.AtrValue = roundFloat(tf.ATR14, 2)
	}

	if len(tf.Klines) >= 2 {
		current := tf.Klines[len(tf.Klines)-1]
		body := math.Abs(current.Close - current.Open)
		fullRange := current.High - current.Low
		if fullRange > 0 {
			signals.BodyRatio = roundFloat(body/fullRange, 2)
		}
	}

	if len(tf.BOLLUpper) > 0 && len(tf.BOLLMiddle) > 0 && len(tf.BOLLLower) > 0 {
		upper := tf.BOLLUpper[len(tf.BOLLUpper)-1]
		middle := tf.BOLLMiddle[len(tf.BOLLMiddle)-1]
		lower := tf.BOLLLower[len(tf.BOLLLower)-1]

		if middle > 0 {
			signals.BbWidthPercent = roundFloat((upper-lower)/middle*100, 2)
		}

		if upper != lower {
			signals.PriceBbPosition = roundFloat((md.CurrentPrice-lower)/(upper-lower)*100, 1)
		}

		if len(tf.BOLLMiddle) >= 200 {
			bbWidths := make([]float64, 0)
			for i := len(tf.BOLLMiddle) - 200; i < len(tf.BOLLMiddle); i++ {
				if tf.BOLLMiddle[i] > 0 {
					bbWidths = append(bbWidths, (tf.BOLLUpper[i]-tf.BOLLLower[i])/tf.BOLLMiddle[i]*100)
				}
			}
			if len(bbWidths) > 0 {
				signals.BbWidthPercentile200 = calculatePercentile(signals.BbWidthPercent, bbWidths)
			}
		}
	}

	return signals
}

func (m *Manager) buildV4LiquiditySignals(tf *market.TimeframeSeriesData, md *market.Data) V4LiquiditySignals {
	signals := V4LiquiditySignals{}

	lookback := 10
	if len(tf.Klines) >= lookback+1 {
		lookbackHigh := 0.0
		lookbackLow := math.MaxFloat64
		for i := len(tf.Klines) - lookback - 1; i < len(tf.Klines)-1; i++ {
			if tf.Klines[i].High > lookbackHigh {
				lookbackHigh = tf.Klines[i].High
			}
			if tf.Klines[i].Low < lookbackLow {
				lookbackLow = tf.Klines[i].Low
			}
		}

		current := tf.Klines[len(tf.Klines)-1]
		signals.LiquiditySweepHigh = current.High > lookbackHigh && current.Close < lookbackHigh
		signals.LiquiditySweepLow = current.Low < lookbackLow && current.Close > lookbackLow
	}

	if len(tf.Klines) > 0 {
		current := tf.Klines[len(tf.Klines)-1]
		signals.VolumeValue = current.Volume

		if len(tf.Klines) >= 200 {
			volumes := make([]float64, 200)
			for i := 0; i < 200; i++ {
				volumes[i] = tf.Klines[len(tf.Klines)-200+i].Volume
			}
			signals.VolumePercentile200 = calculatePercentile(current.Volume, volumes)
		}

		if len(tf.Klines) >= 20 {
			volumeMA := 0.0
			for i := len(tf.Klines) - 20; i < len(tf.Klines); i++ {
				volumeMA += tf.Klines[i].Volume
			}
			volumeMA /= 20
			if volumeMA > 0 {
				signals.VolumeMaRatio = roundFloat(current.Volume/volumeMA, 2)
			}
		}

		signals.VolumeSpike = signals.VolumeMaRatio > 1.5

		if len(tf.Klines) >= 2 {
			prev := tf.Klines[len(tf.Klines)-2]
			priceUp := current.Close > prev.Close
			volumeUp := current.Volume > prev.Volume
			signals.PriceVolumeSync = priceUp == volumeUp
		}
	}

	return signals
}

func (m *Manager) buildV4PositioningSignals(md *market.Data) V4PositioningSignals {
	signals := V4PositioningSignals{}

	if md.OpenInterest != nil {
		signals.OiValue = md.OpenInterest.Latest
		if md.OpenInterest.Before5Period > 0 {
			signals.OiChangePercent = roundFloat((md.OpenInterest.Latest-md.OpenInterest.Before5Period)/md.OpenInterest.Before5Period*100, 2)
		}
	}

	if md.FundingRate != 0 {
		signals.FundingRate = md.FundingRate
	}

	return signals
}

func (m *Manager) buildV4RankingSignals(md *market.Data, ctx *ChaosContext) V4RankingSignals {
	signals := V4RankingSignals{}

	// FIXME: market.Data does not have PriceChange24h. Using PriceChange4h as a placeholder or 0.
	// To strictly follow V4 design, we need to add 24h change to market.Data or calculate it from daily klines.
	// For now, we use 0 to avoid compilation error.
	signals.PriceChangePercent24h = 0 // roundFloat(md.PriceChange24h*100, 2)

	return signals
}

func (m *Manager) buildV4Indicators(tf *market.TimeframeSeriesData, cfg *ChaosConfig) Indicators {
	limit := 10
	indicators := Indicators{}

	if cfg == nil || cfg.Indicators.EnableEMA {
		indicators.EMA20 = getLastNFloat(tf.EMA20Values, limit)
		indicators.EMA50 = getLastNFloat(tf.EMA50Values, limit)
	}

	if cfg == nil || cfg.Indicators.EnableRSI {
		indicators.RSI14 = getLastNFloat(tf.RSI14Values, limit)
	}

	if cfg == nil || cfg.Indicators.EnableMACD {
		macdVals := getLastNFloat(tf.MACDValues, limit*3)
		if len(macdVals) >= limit*3 {
			indicators.MACD = MACDData{
				Line:      macdVals[0:limit],
				Signal:    macdVals[limit : 2*limit],
				Histogram: macdVals[2*limit : 3*limit],
			}
		}
	}

	if cfg == nil || cfg.Indicators.EnableATR {
		indicators.ATR14 = getLastNFloat(tf.ATR14Values, limit)
	}

	if cfg == nil || cfg.Indicators.EnableBOLL {
		indicators.Bollinger = BollingerData{
			Upper:  getLastNFloat(tf.BOLLUpper, limit),
			Middle: getLastNFloat(tf.BOLLMiddle, limit),
			Lower:  getLastNFloat(tf.BOLLLower, limit),
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

func (m *Manager) buildV4Klines(tf *market.TimeframeSeriesData) Klines {
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

// ============================================================================
// Helper Functions for V4 Calculations
// ============================================================================

func countHigherHigh(klines []market.KlineBar, n int) int {
	if len(klines) < n+1 {
		return 0
	}
	count := 0
	for i := len(klines) - n; i < len(klines); i++ {
		if i > 0 && klines[i].High > klines[i-1].High {
			count++
		}
	}
	return count
}

func countLowerLow(klines []market.KlineBar, n int) int {
	if len(klines) < n+1 {
		return 0
	}
	count := 0
	for i := len(klines) - n; i < len(klines); i++ {
		if i > 0 && klines[i].Low < klines[i-1].Low {
			count++
		}
	}
	return count
}

func countSwingHigh(klines []market.KlineBar, n int, lookback int) int {
	if len(klines) < n+2*lookback {
		return 0
	}
	count := 0
	start := len(klines) - n
	if start < lookback {
		start = lookback
	}
	end := len(klines) - lookback

	for i := start; i < end; i++ {
		if isSwingHigh(klines, i, lookback) {
			count++
		}
	}
	return count
}

func countSwingLow(klines []market.KlineBar, n int, lookback int) int {
	if len(klines) < n+2*lookback {
		return 0
	}
	count := 0
	start := len(klines) - n
	if start < lookback {
		start = lookback
	}
	end := len(klines) - lookback

	for i := start; i < end; i++ {
		if isSwingLow(klines, i, lookback) {
			count++
		}
	}
	return count
}

func isSwingHigh(klines []market.KlineBar, i int, lookback int) bool {
	if i < lookback || i >= len(klines)-lookback {
		return false
	}
	currentHigh := klines[i].High

	for j := i - lookback; j < i; j++ {
		if klines[j].High >= currentHigh {
			return false
		}
	}
	for j := i + 1; j <= i+lookback; j++ {
		if klines[j].High >= currentHigh {
			return false
		}
	}
	return true
}

func isSwingLow(klines []market.KlineBar, i int, lookback int) bool {
	if i < lookback || i >= len(klines)-lookback {
		return false
	}
	currentLow := klines[i].Low

	for j := i - lookback; j < i; j++ {
		if klines[j].Low <= currentLow {
			return false
		}
	}
	for j := i + 1; j <= i+lookback; j++ {
		if klines[j].Low <= currentLow {
			return false
		}
	}
	return true
}

func countConsecutiveBars(klines []market.KlineBar) (up int, down int) {
	if len(klines) == 0 {
		return 0, 0
	}

	last := klines[len(klines)-1]
	isUp := last.Close > last.Open
	isDown := last.Close < last.Open

	if !isUp && !isDown {
		return 0, 0
	}

	count := 0
	for i := len(klines) - 1; i >= 0; i-- {
		k := klines[i]
		currentUp := k.Close > k.Open
		currentDown := k.Close < k.Open

		if isUp {
			if currentUp {
				count++
			} else {
				break
			}
		} else { // isDown
			if currentDown {
				count++
			} else {
				break
			}
		}
	}

	if isUp {
		return count, 0
	}
	return 0, count
}

func countBarsInPeriod(klines []market.KlineBar, n int) (up int, down int) {
	up = 0
	down = 0
	start := len(klines) - n
	if start < 0 {
		start = 0
	}

	for i := start; i < len(klines); i++ {
		if klines[i].Close > klines[i].Open {
			up++
		} else if klines[i].Close < klines[i].Open {
			down++
		}
	}
	return up, down
}

func calculatePercentile(value float64, history []float64) int {
	if len(history) == 0 {
		return 50
	}

	sorted := make([]float64, len(history))
	copy(sorted, history)
	sort.Float64s(sorted)

	count := 0
	for _, v := range sorted {
		if v <= value {
			count++
		}
	}

	percentile := float64(count) / float64(len(sorted)) * 100
	return int(math.Round(percentile))
}

func getLastNFloat(slice []float64, n int) []float64 {
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
