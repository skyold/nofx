// =============================================================================
// Chaos Trading System - JSON Type Definitions
// =============================================================================
//
// 这些类型定义供 V4 (JSON Formatter) 使用
// =============================================================================

package chaos

import "math"

// ============================================================================
// Data Structures (JSON Design)
// ============================================================================

// AccountInfo JSON 账户信息
type AccountInfo struct {
	Equity         float64 `json:"equity"`
	Balance        float64 `json:"balance"`
	PnlPct         float64 `json:"pnl_pct"`
	MarginUsedPct  float64 `json:"margin_used_pct"`
	PositionsCount int     `json:"positions_count"`
}

// Indicators JSON 指标数据
type Indicators struct {
	EMA20     []float64     `json:"ema20,omitempty"`
	EMA50     []float64     `json:"ema50,omitempty"`
	RSI7      []float64     `json:"rsi7,omitempty"`
	RSI14     []float64     `json:"rsi14,omitempty"`
	MACD      MACDData      `json:"macd,omitempty"`
	ATR14     []float64     `json:"atr14,omitempty"`
	Bollinger BollingerData `json:"bollinger,omitempty"`
	Volume    []float64     `json:"volume,omitempty"`
}

// MACDData JSON MACD 数据
type MACDData struct {
	Line      []float64 `json:"line,omitempty"`
	Signal    []float64 `json:"signal,omitempty"`
	Histogram []float64 `json:"histogram,omitempty"`
}

// BollingerData JSON 布林带数据
type BollingerData struct {
	Upper    []float64 `json:"upper,omitempty"`
	Middle   []float64 `json:"middle,omitempty"`
	Lower    []float64 `json:"lower,omitempty"`
	WidthPct float64   `json:"width_pct,omitempty"`
}

// Klines JSON K 线数据
type Klines struct {
	Date            string          `json:"date"`
	Columns         []string        `json:"columns"`
	Values          [][]interface{} `json:"values"`
	CurrentBarIndex int             `json:"current_bar_index"`
}

// ============================================================================
// Helpers
// ============================================================================

// roundFloat 四舍五入到指定精度
func roundFloat(val float64, precision uint) float64 {
	ratio := math.Pow(10, float64(precision))
	return math.Round(val*ratio) / ratio
}
