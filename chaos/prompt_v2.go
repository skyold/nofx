package chaos

import (
	"encoding/json"
	"fmt"
	"math"
	"nofx/market"
	"strings"
	"time"
)

// BuildCandidatesJSON generates a JSON representation of candidate coins for LLM reasoning.
// Every timeframe (15m, 1h, 4h) follows the exact same structure for consistency.
func (e *ChaosEngine) BuildCandidatesJSON(ctx *ChaosContext) string {
	if len(ctx.CandidateCoins) == 0 {
		return "[]"
	}

	var candidates []CandidateJSON
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

		candidate := CandidateJSON{
			Symbol:     coin.Symbol,
			Timeframes: make(map[string]TimeframeJSON),
		}

		for _, tf := range targetTimeframes {
			if tfData, hasData := marketData.TimeframeData[tf]; hasData {
				candidate.Timeframes[tf] = TimeframeJSON{
					Signals:    e.buildSignals(marketData, tfData, ctx.Config),
					Indicators: e.buildIndicators(tfData, ctx.Config),
					Klines:     e.buildKlines(tfData),
				}
			}
		}

		if len(candidate.Timeframes) > 0 {
			candidates = append(candidates, candidate)
		}
	}

	jsonData, err := json.MarshalIndent(candidates, "", "  ")
	if err != nil {
		return fmt.Sprintf("Error building JSON: %v", err)
	}
	return string(jsonData)
}

// ============================================================================
// Data Structures
// ============================================================================

type CandidateJSON struct {
	Symbol     string                   `json:"symbol"`
	Timeframes map[string]TimeframeJSON `json:"timeframes"`
}

type TimeframeJSON struct {
	Signals    SignalsJSON    `json:"signals"`
	Indicators IndicatorsJSON `json:"indicators"`
	Klines     KlinesJSON     `json:"klines"`
}

type SignalsJSON struct {
	MarketRegime  string        `json:"market_regime,omitempty"` // Only relevant for 1h/4h
	TrendStrength float64       `json:"trend_strength"`
	EMACross      EMACrossJSON  `json:"ema_cross"`
	RSI           RSIStateJSON  `json:"rsi"`
	Momentum      string        `json:"momentum"`
	KeyLevels     KeyLevelsJSON `json:"key_levels"`
}

type EMACrossJSON struct {
	Status   string `json:"status"`   // "golden_cross", "death_cross", "none"
	BarsAgo  int    `json:"bars_ago"` // How many bars ago the cross happened
	Strength string `json:"strength"` // "strong", "weak"
}

type RSIStateJSON struct {
	Current float64 `json:"current"`
	State   string  `json:"state"` // "overbought", "oversold", "neutral"
	Trend   string  `json:"trend"` // "rising", "falling", "flat"
}

type KeyLevelsJSON struct {
	Resistance       []float64 `json:"resistance"`
	Support          []float64 `json:"support"`
	LocalSupport     float64   `json:"local_support,omitempty"`      // Intraday micro-level support
	LocalSupportTime string    `json:"local_support_time,omitempty"` // UTC HH:mm
	DailyLow         float64   `json:"daily_low,omitempty"`          // 24h session low
}

type IndicatorsJSON struct {
	EMA20    []float64 `json:"ema20,omitempty"`
	EMA50    []float64 `json:"ema50,omitempty"`
	RSI7     []float64 `json:"rsi7,omitempty"`
	RSI14    []float64 `json:"rsi14,omitempty"`
	MACD     []float64 `json:"macd,omitempty"`
	ATR      float64   `json:"atr,omitempty"`
	BOLLUpper  []float64 `json:"boll_upper,omitempty"`
	BOLLMiddle []float64 `json:"boll_middle,omitempty"`
	BOLLLower  []float64 `json:"boll_lower,omitempty"`
}

type KlinesJSON struct {
	Columns         []string        `json:"columns"`
	Values          [][]interface{} `json:"values"`
	CurrentBarIndex int             `json:"current_bar_index"`
}

// ============================================================================
// Builder Logic
// ============================================================================

func (e *ChaosEngine) buildSignals(md *market.Data, tf *market.TimeframeSeriesData, cfg *ChaosConfig) SignalsJSON {
	signals := SignalsJSON{}

	enableEMA := cfg == nil || cfg.Indicators.EnableEMA
	enableRSI := cfg == nil || cfg.Indicators.EnableRSI

	if enableRSI {
		rsiVal := 50.0
		if len(tf.RSI14Values) > 0 {
			rsiVal = tf.RSI14Values[len(tf.RSI14Values)-1]
		}
		signals.RSI.Current = roundFloat(rsiVal, 1)
		if rsiVal > 70 {
			signals.RSI.State = "overbought"
		} else if rsiVal < 30 {
			signals.RSI.State = "oversold"
		} else {
			signals.RSI.State = "neutral"
		}

		if len(tf.RSI14Values) >= 3 {
			prev := tf.RSI14Values[len(tf.RSI14Values)-2]
			prev2 := tf.RSI14Values[len(tf.RSI14Values)-3]
			if rsiVal > prev && prev > prev2 {
				signals.RSI.Trend = "rising"
			} else if rsiVal < prev && prev < prev2 {
				signals.RSI.Trend = "falling"
			} else {
				signals.RSI.Trend = "flat"
			}
		}
	}

	if enableEMA {
		signals.EMACross = detectEMACross(tf.EMA20Values, tf.EMA50Values)

		if len(tf.EMA20Values) > 0 && len(tf.EMA50Values) > 0 {
			ema20 := tf.EMA20Values[len(tf.EMA20Values)-1]
			ema50 := tf.EMA50Values[len(tf.EMA50Values)-1]
			price := md.CurrentPrice

			diffPct := (ema20 - ema50) / ema50 * 100
			if diffPct > 0.5 && price > ema20 {
				signals.MarketRegime = "trending_bullish"
				signals.TrendStrength = roundFloat(math.Min(diffPct, 3.0)/3.0, 2)
			} else if diffPct < -0.5 && price < ema20 {
				signals.MarketRegime = "trending_bearish"
				signals.TrendStrength = roundFloat(math.Min(-diffPct, 3.0)/3.0, 2)
			} else {
				signals.MarketRegime = "ranging"
				signals.TrendStrength = roundFloat(math.Abs(diffPct)/1.0, 2)
			}
		}
	}

	if md.PriceChange1h > 0.5 {
		signals.Momentum = "strong_up"
	} else if md.PriceChange1h < -0.5 {
		signals.Momentum = "strong_down"
	} else if md.PriceChange1h > 0 {
		signals.Momentum = "weak_up"
	} else {
		signals.Momentum = "weak_down"
	}

	anchors := market.ComputeAnchors(md.TimeframeData)
	for _, a := range anchors {
		if a.Timeframe == tf.Timeframe {
			if strings.Contains(strings.ToLower(a.Type), "resistance") || strings.Contains(strings.ToLower(a.Type), "high") {
				signals.KeyLevels.Resistance = append(signals.KeyLevels.Resistance, a.Price)
			} else if strings.Contains(strings.ToLower(a.Type), "support") || strings.Contains(strings.ToLower(a.Type), "low") {
				signals.KeyLevels.Support = append(signals.KeyLevels.Support, a.Price)
			}
		}
	}

	if md.LocalSupport > 0 {
		signals.KeyLevels.LocalSupport = md.LocalSupport
		if md.LocalSupportTime > 0 {
			signals.KeyLevels.LocalSupportTime = time.Unix(md.LocalSupportTime/1000, 0).UTC().Format("15:04")
		}
	}
	if md.DailyLow > 0 {
		signals.KeyLevels.DailyLow = md.DailyLow
	}

	return signals
}

func (e *ChaosEngine) buildIndicators(tf *market.TimeframeSeriesData, cfg *ChaosConfig) IndicatorsJSON {
	limit := 10
	indicators := IndicatorsJSON{}

	if cfg == nil || cfg.Indicators.EnableEMA {
		indicators.EMA20 = getLastN(tf.EMA20Values, limit)
		indicators.EMA50 = getLastN(tf.EMA50Values, limit)
	}

	if cfg == nil || cfg.Indicators.EnableRSI {
		indicators.RSI7 = getLastN(tf.RSI7Values, limit)
		indicators.RSI14 = getLastN(tf.RSI14Values, limit)
	}

	if cfg == nil || cfg.Indicators.EnableMACD {
		indicators.MACD = getLastN(tf.MACDValues, limit)
	}

	if cfg == nil || cfg.Indicators.EnableATR {
		indicators.ATR = roundFloat(tf.ATR14, 4)
	}

	if cfg == nil || cfg.Indicators.EnableBOLL {
		indicators.BOLLUpper = getLastN(tf.BOLLUpper, limit)
		indicators.BOLLMiddle = getLastN(tf.BOLLMiddle, limit)
		indicators.BOLLLower = getLastN(tf.BOLLLower, limit)
	}

	return indicators
}

func (e *ChaosEngine) buildKlines(tf *market.TimeframeSeriesData) KlinesJSON {
	klines := KlinesJSON{
		Columns:         []string{"time", "o", "h", "l", "c", "v"},
		Values:          make([][]interface{}, 0),
		CurrentBarIndex: len(tf.Klines) - 1,
	}

	// Limit to last 10-15 candles to save tokens
	startIdx := len(tf.Klines) - 15
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
			int(k.Volume), // Volume as int to save space
		}
		klines.Values = append(klines.Values, row)
	}

	return klines
}

// ============================================================================
// Helpers
// ============================================================================

func detectEMACross(ema20, ema50 []float64) EMACrossJSON {
	res := EMACrossJSON{Status: "none", BarsAgo: -1, Strength: "none"}
	if len(ema20) < 5 || len(ema50) < 5 || len(ema20) != len(ema50) {
		return res
	}

	// Look back 5 bars
	for i := len(ema20) - 1; i >= len(ema20)-5; i-- {
		if i == 0 {
			break
		}
		currDiff := ema20[i] - ema50[i]
		prevDiff := ema20[i-1] - ema50[i-1]

		if currDiff > 0 && prevDiff <= 0 {
			res.Status = "golden_cross"
			res.BarsAgo = len(ema20) - 1 - i
			res.Strength = "standard" // Can be refined
			return res
		} else if currDiff < 0 && prevDiff >= 0 {
			res.Status = "death_cross"
			res.BarsAgo = len(ema20) - 1 - i
			res.Strength = "standard"
			return res
		}
	}
	return res
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
