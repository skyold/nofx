// =============================================================================
// Chaos Trading System - Prompt Builders
// =============================================================================
//
// Builder 模式：负责构建 Prompt 数据
// - BasicBuilder: 基础数据（V1 使用）
// - EnhancedBuilder: 增强数据（V2 使用，包含机构分类器 + LLM 简报）
// =============================================================================

package chaos

import (
	"nofx/market"
	"nofx/store"
)

// ============================================================================
// Builder 接口
// ============================================================================

// MarketDataBuilder 市场数据构建器接口
type MarketDataBuilder interface {
	BuildMarketData(symbol string, data *market.Data, indicators store.IndicatorConfig) *MarketPromptData
}

// ============================================================================
// BasicBuilder - 基础数据构建器
// ============================================================================

// BasicBuilder 基础数据构建器
// 只构建基础市场数据，不包含机构分类器等增强功能
type BasicBuilder struct {
	Manager *Manager
}

// BuildMarketData 构建基础市场数据
func (b *BasicBuilder) BuildMarketData(symbol string, data *market.Data, indicators store.IndicatorConfig) *MarketPromptData {
	result := &MarketPromptData{
		Symbol:        symbol,
		CurrentPrice:  data.CurrentPrice,
		LocalSupport:  data.LocalSupport,
		LocalSupportTime: data.LocalSupportTime,
		DailyLow:      data.DailyLow,
		FundingRate:   data.FundingRate,
		Timeframes:    make(map[string]*TimeframeData),
	}

	// 构建持仓量数据
	if data.OpenInterest != nil {
		result.OpenInterest = &OpenInterestData{
			Latest:  data.OpenInterest.Latest,
			Average: data.OpenInterest.Average,
		}
	}

	// 构建多时间周期数据
	for tf, tfData := range data.TimeframeData {
		result.Timeframes[tf] = b.buildTimeframeData(tfData, indicators)
	}

	// 构建物理结构锚点
	result.SwingFeatures = b.buildSwingFeatures(data)

	// 构建主要区间边界
	b.buildMajorLevels(result, data)

	// 构建动态参考
	result.DynamicReferences = b.buildDynamicReferences(data)

	return result
}

// buildTimeframeData 构建单个时间周期数据
func (b *BasicBuilder) buildTimeframeData(tfData *market.TimeframeSeriesData, indicators store.IndicatorConfig) *TimeframeData {
	return &TimeframeData{
		Timeframe:   tfData.Timeframe,
		Klines:      tfData.Klines,
		EMA20Values: b.getIndicatorValues(indicators.EnableEMA, tfData.EMA20Values),
		EMA50Values: b.getIndicatorValues(indicators.EnableEMA, tfData.EMA50Values),
		MACDValues:  b.getIndicatorValues(indicators.EnableMACD, tfData.MACDValues),
		RSI7Values:  b.getIndicatorValues(indicators.EnableRSI, tfData.RSI7Values),
		RSI14Values: b.getIndicatorValues(indicators.EnableRSI, tfData.RSI14Values),
		ATR14Values: b.getIndicatorValues(indicators.EnableATR, tfData.ATR14Values),
		BOLLUpper:   b.getIndicatorValues(indicators.EnableBOLL, tfData.BOLLUpper),
		BOLLMiddle:  b.getIndicatorValues(indicators.EnableBOLL, tfData.BOLLMiddle),
		BOLLLower:   b.getIndicatorValues(indicators.EnableBOLL, tfData.BOLLLower),
	}
}

// getIndicatorValues 根据配置返回指标值
func (b *BasicBuilder) getIndicatorValues(enabled bool, values []float64) []float64 {
	if !enabled {
		return nil
	}
	return values
}

// buildSwingFeatures 构建摆动点特征
func (b *BasicBuilder) buildSwingFeatures(data *market.Data) *TechnicalFeatures {
	// 确定主要时间周期
	var mainTf string
	if _, ok := data.TimeframeData["1h"]; ok {
		mainTf = "1h"
	} else if _, ok := data.TimeframeData["4h"]; ok {
		mainTf = "4h"
	}

	if mainTf == "" {
		return nil
	}

	tfData := data.TimeframeData[mainTf]
	return GenerateTechnicalFeatures(tfData.Klines, swingWindowSize)
}

// buildMajorLevels 构建主要区间边界
func (b *BasicBuilder) buildMajorLevels(result *MarketPromptData, data *market.Data) {
	tf1d, ok := data.TimeframeData["1d"]
	if !ok || len(tf1d.Klines) == 0 {
		return
	}

	// 使用昨天的 K 线
	var last market.KlineBar
	if len(tf1d.Klines) >= 2 {
		last = tf1d.Klines[len(tf1d.Klines)-2]
	} else {
		last = tf1d.Klines[len(tf1d.Klines)-1]
	}

	result.MajorSupport = last.Low
	result.MajorResistance = last.High

	// 计算测试次数
	var mainTf string
	if _, ok := data.TimeframeData["1h"]; ok {
		mainTf = "1h"
	} else if _, ok := data.TimeframeData["4h"]; ok {
		mainTf = "4h"
	}

	if mainTf != "" {
		result.SupportTests = countLevelTests(result.MajorSupport, SwingLow, data.TimeframeData[mainTf].Klines, 0, levelTestThreshold)
		result.ResistanceTests = countLevelTests(result.MajorResistance, SwingHigh, data.TimeframeData[mainTf].Klines, 0, levelTestThreshold)
	}
}

// buildDynamicReferences 构建动态参考
func (b *BasicBuilder) buildDynamicReferences(data *market.Data) []string {
	references := []string{}

	// 5M 布林带下轨
	if tf, ok := data.TimeframeData["5m"]; ok && len(tf.BOLLLower) > 0 {
		references = append(references, formatPriceForPrompt(tf.BOLLLower[len(tf.BOLLLower)-1]))
	}

	// 1H EMA50
	if tf, ok := data.TimeframeData["1h"]; ok && len(tf.EMA50Values) > 0 {
		references = append(references, formatFloatSlice([]float64{tf.EMA50Values[len(tf.EMA50Values)-1]}))
	}

	return references
}

// ============================================================================
// EnhancedBuilder - 增强数据构建器
// ============================================================================

// EnhancedBuilder 增强数据构建器
// 在基础数据上增加机构分类器和 LLM 简报
type EnhancedBuilder struct {
	*BasicBuilder // 继承基础功能
}

// BuildMarketData 构建增强市场数据
func (b *EnhancedBuilder) BuildMarketData(symbol string, data *market.Data, indicators store.IndicatorConfig) *MarketPromptData {
	// 先构建基础数据
	result := b.BasicBuilder.BuildMarketData(symbol, data, indicators)

	// 添加 Regime 分类器
	b.buildRegime(result, data, indicators)

	// 生成 LLM 战术简报
	b.buildLLMBriefing(result, symbol)

	return result
}

// buildRegime 构建 Regime 分类器
func (b *EnhancedBuilder) buildRegime(result *MarketPromptData, data *market.Data, indicators store.IndicatorConfig) {
	// 确定主要时间周期
	var mainTf string
	if _, ok := data.TimeframeData["1h"]; ok {
		mainTf = "1h"
	} else if _, ok := data.TimeframeData["4h"]; ok {
		mainTf = "4h"
	}

	if mainTf == "" {
		return
	}

	tfData := data.TimeframeData[mainTf]
	oiLatest := 0.0
	oiAverage := 0.0
	if data.OpenInterest != nil {
		oiLatest = data.OpenInterest.Latest
		oiAverage = data.OpenInterest.Average
	}

	regimeSignal := GenerateRegimeSignal(
		tfData.Klines,
		tfData.EMA20Values,
		tfData.EMA50Values,
		tfData.ATR14Values,
		oiLatest,
		oiAverage,
		data.FundingRate,
	)

	result.Regime = regimeSignal
}

// buildLLMBriefing 生成 LLM 战术简报
func (b *EnhancedBuilder) buildLLMBriefing(result *MarketPromptData, symbol string) {
	if result.Regime == nil {
		return
	}

	cvdSlope := 0.0 // 暂时跳过 CVD，预留接口
	result.LLMBriefing = result.Regime.GenerateLLMBriefing(symbol, result.CurrentPrice, cvdSlope)
}
