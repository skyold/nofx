package chaos

import (
	"fmt"
	"math"
	"strings"
	"time"

	"nofx/market"
)

// =============================================================================
// 特征工程：摆动点拓扑结构 & 支撑阻力位测试
// =============================================================================
//
// 主要功能：
// 1. 摆动点检测：识别局部高点和低点
// 2. 拓扑标记：HH/HL/LH/LL 标记
// 3. 支撑阻力位测试次数统计
// 4. 机构市场状态分类：识别机构主导的市场状态
// 5. 特征格式化：生成用于 LLM 的文本格式化输出
// =============================================================================

// =============================================================================
// 摆动点相关类型定义
// =============================================================================

// SwingType 摆动点类型枚举
type SwingType int

const (
	// SwingHigh 高点摆动点（局部高点）
	SwingHigh SwingType = iota
	// SwingLow 低点摆动点（局部低点）
	SwingLow
)

// SwingPoint 摆动点结构
// 描述价格波动中的局部高点或低点
type SwingPoint struct {
	Price     float64   // 价格
	Time      int64     // Unix 时间戳（毫秒）
	Index     int       // 在 K线切片中的索引
	Type      SwingType // 类型：高点或低点
	Label     string    // 拓扑标签：HH, LH, HL, LL, High, Low
	TestCount int       // 该价位被测试的次数
}

// =============================================================================
// 机构市场状态信号
// =============================================================================

// InstitutionalRegimeSignal 机构市场状态信号
// 用于识别当前市场处于哪种机构主导状态
type InstitutionalRegimeSignal struct {
	// 动力学因子
	Ema20AboveEma50Count20 int     // 最近 20 根 K 线中 EMA20 高于 EMA50 的次数
	OiChangePercent5       float64 // 近 5 周期持仓量（OI）变化百分比
	AtrPct                 float64 // ATR（平均真实波幅）百分比

	// 价格结构
	PriceMakingHigh50 bool    // 当前价格是否创 50 周期新高
	PriceMakingLow50  bool    // 当前价格是否创 50 周期新低
	BoxHeight50       float64 // 50 周期价格区间高度（相对百分比）

	// 衍生品状态
	FundingRate        float64 // 资金费率
	FundingRateExtreme bool    // 资金费率是否极端（超过阈值）

	// 状态分类
	RegimeClassification string // 市场状态分类结果
}

// =============================================================================
// Regime 分类器配置
// =============================================================================

// RegimeClassifierConfig Regime 分类器配置
// 用于集中管理所有可调整的阈值参数
type RegimeClassifierConfig struct {
	// OI 变化阈值
	OiGrowthThreshold float64 // OI 增长/下降阈值（默认 0.05 = 5%）

	// 资金费率阈值
	FundingExtreme float64 // 资金费率极端阈值（默认 0.0003 = 0.03%）

	// 趋势确认阈值
	TrendThreshold int // 趋势确认所需的 EMA 计数（默认 16/20）

	// 动能特征阈值
	RecoveryRatioThreshold     float64 // 收复比例阈值（默认 0.4 = 40%）
	ShortSqueezePriceThreshold float64 // 轧空行情价格涨幅阈值（默认 1.01 = 1%）
	ShortSqueezeOiThreshold    float64 // 轧空行情 OI 降幅阈值（默认 -0.02 = -2%）
	CapitulationAtrThreshold   float64 // 恐慌探底 ATR 阈值（默认 0.03 = 3%）
}

// NewRegimeClassifierConfig_Balanced 创建平衡配置（默认）
// 适用于 BTC/ETH 1H 级别，在准确率和覆盖率之间取得平衡
func NewRegimeClassifierConfig_Balanced() *RegimeClassifierConfig {
	return &RegimeClassifierConfig{
		OiGrowthThreshold:          0.05,   // 5%
		FundingExtreme:             0.0003, // 0.03%
		TrendThreshold:             16,     // 16/20
		RecoveryRatioThreshold:     0.4,    // 40%
		ShortSqueezePriceThreshold: 1.01,   // 1%
		ShortSqueezeOiThreshold:    -0.02,  // -2%
		CapitulationAtrThreshold:   0.03,   // 3%
	}
}

// NewRegimeClassifierConfig_Conservative 创建保守配置（高准确率）
// 适用于追求高胜率、低频率交易的策略
func NewRegimeClassifierConfig_Conservative() *RegimeClassifierConfig {
	return &RegimeClassifierConfig{
		OiGrowthThreshold:          0.08,   // 8%（提高，只在机构确定入场时动作）
		FundingExtreme:             0.0004, // 0.04%（提高）
		TrendThreshold:             18,     // 18/20（提高，更严格的趋势确认）
		RecoveryRatioThreshold:     0.6,    // 60%（提高，只在强力反弹时触发）
		ShortSqueezePriceThreshold: 1.02,   // 2%（提高）
		ShortSqueezeOiThreshold:    -0.03,  // -3%（提高）
		CapitulationAtrThreshold:   0.04,   // 4%（提高，更严格的恐慌底判定）
	}
}

// NewRegimeClassifierConfig_Aggressive 创建激进配置（高覆盖率）
// 适用于追求高频率、捕捉更多机会的策略
func NewRegimeClassifierConfig_Aggressive() *RegimeClassifierConfig {
	return &RegimeClassifierConfig{
		OiGrowthThreshold:          0.03,   // 3%（降低，对更多 OI 变化有反应）
		FundingExtreme:             0.0002, // 0.02%（降低）
		TrendThreshold:             14,     // 14/20（降低，更快确认趋势）
		RecoveryRatioThreshold:     0.3,    // 30%（降低，对任何风吹草动都有反应）
		ShortSqueezePriceThreshold: 1.005,  // 0.5%（降低）
		ShortSqueezeOiThreshold:    -0.01,  // -1%（降低）
		CapitulationAtrThreshold:   0.02,   // 2%（降低，更容易触发恐慌底）
	}
}

// TechnicalFeatures 技术特征
// 封装特征工程过程的所有结果
type TechnicalFeatures struct {
	SwingPoints         []SwingPoint               // 摆动点列表
	Trend               string                     // 基于拓扑结构的简单趋势描述
	InstitutionalRegime *InstitutionalRegimeSignal // 机构市场状态信号
}

// =============================================================================
// 机构市场状态信号生成
// =============================================================================

// getPriceRange 获取指定周期内的价格区间
func getPriceRange(klines []market.KlineBar, period int) (float64, float64) {
	if len(klines) < period {
		return 0, 0
	}
	sub := klines[len(klines)-period:]
	h, l := 0.0, math.MaxFloat64
	for _, k := range sub {
		if k.High > h {
			h = k.High
		}
		if k.Low < l {
			l = k.Low
		}
	}
	return h, l
}

// GenerateInstitutionalRegimeSignal 计算机构市场状态信号（使用默认平衡配置）
//
// 参数说明：
//   - klines: K 线数据列表
//   - ema20Values: EMA20 指标值列表
//   - ema50Values: EMA50 指标值列表
//   - atr14Values: ATR14 指标值列表
//   - oiLatest: 最新持仓量（OI）
//   - oiAverage: 平均持仓量（OI）
//   - fundingRate: 当前资金费率
//
// 返回值：
//   - 机构市场状态信号对象
//
// 分类结果：
//   - REVERSING_POTENTIAL_TOP：顶部反转预警
//   - BULLISH_SHORT_SQUEEZE：轧空（空头平仓）
//   - IMPULSIVE_RECOVERY：脉冲式修复
//   - TRENDING_UP_STRONG：强多头趋势
//   - TRENDING_UP_OVERHEATED：多头过热
//   - TRENDING_DOWN_STRONG：强空头趋势
//   - CAPITULATION_BOTTOM：恐慌探底
//   - RANGE_BOUND：震荡区间
//   - TRANSITIONAL：过渡状态
func GenerateInstitutionalRegimeSignal(
	klines []market.KlineBar,
	ema20Values []float64,
	ema50Values []float64,
	atr14Values []float64,
	oiLatest float64,
	oiAverage float64,
	fundingRate float64,
) *InstitutionalRegimeSignal {
	return GenerateInstitutionalRegimeSignalWithConfig(
		klines,
		ema20Values,
		ema50Values,
		atr14Values,
		oiLatest,
		oiAverage,
		fundingRate,
		NewRegimeClassifierConfig_Balanced(),
	)
}

// GenerateInstitutionalRegimeSignalWithConfig 计算机构市场状态信号（使用自定义配置）
//
// 参数说明：
//   - klines: K 线数据列表
//   - ema20Values: EMA20 指标值列表
//   - ema50Values: EMA50 指标值列表
//   - atr14Values: ATR14 指标值列表
//   - oiLatest: 最新持仓量（OI）
//   - oiAverage: 平均持仓量（OI）
//   - fundingRate: 当前资金费率
//   - config: 分类器配置（如果为 nil，使用默认平衡配置）
//
// 返回值：
//   - 机构市场状态信号对象
func GenerateInstitutionalRegimeSignalWithConfig(
	klines []market.KlineBar,
	ema20Values []float64,
	ema50Values []float64,
	atr14Values []float64,
	oiLatest float64,
	oiAverage float64,
	fundingRate float64,
	config *RegimeClassifierConfig,
) *InstitutionalRegimeSignal {
	if config == nil {
		config = NewRegimeClassifierConfig_Balanced()
	}

	signal := &InstitutionalRegimeSignal{
		FundingRate: fundingRate,
	}

	if oiAverage > 0 {
		signal.OiChangePercent5 = (oiLatest - oiAverage) / oiAverage
	}
	signal.FundingRateExtreme = math.Abs(fundingRate) > config.FundingExtreme

	lastIdx := len(klines) - 1
	if lastIdx < 0 {
		signal.RegimeClassification = "TRANSITIONAL"
		return signal
	}

	currentPrice := klines[lastIdx].Close
	currentEma20 := 0.0
	currentAtr := 0.0

	if len(ema20Values) > 0 {
		currentEma20 = ema20Values[len(ema20Values)-1]
	}
	if len(atr14Values) > 0 {
		currentAtr = atr14Values[len(atr14Values)-1]
	}

	countUp := 0
	countDown := 0
	lookback := 20
	if len(ema20Values) >= lookback && len(ema50Values) >= lookback {
		start := len(ema20Values) - lookback
		for i := start; i < len(ema20Values); i++ {
			if ema20Values[i] > ema50Values[i] {
				countUp++
			}
			if ema20Values[i] < ema50Values[i] {
				countDown++
			}
		}
	}
	signal.Ema20AboveEma50Count20 = countUp

	high50, low50 := getPriceRange(klines, 50)
	signal.PriceMakingHigh50 = currentPrice >= high50
	signal.PriceMakingLow50 = currentPrice <= low50
	if low50 > 0 {
		signal.BoxHeight50 = (high50 - low50) / low50
	}
	if currentPrice > 0 {
		signal.AtrPct = currentAtr / currentPrice
	}

	recoveryRatio := 0.0
	if high50 > low50 {
		recoveryRatio = (currentPrice - low50) / (high50 - low50)
	}

	isImpulsiveCross := false
	if lastIdx >= 1 {
		isImpulsiveCross = currentPrice > currentEma20 && klines[lastIdx-1].Close < currentEma20
	}

	isShortSqueeze := false
	if lastIdx >= 1 {
		isShortSqueeze = (currentPrice > klines[lastIdx-1].Close*config.ShortSqueezePriceThreshold) &&
			(signal.OiChangePercent5 < config.ShortSqueezeOiThreshold)
	}

	switch {
	case signal.PriceMakingHigh50 && (signal.OiChangePercent5 < -config.OiGrowthThreshold || signal.FundingRateExtreme):
		signal.RegimeClassification = "REVERSING_POTENTIAL_TOP"

	case isImpulsiveCross && recoveryRatio > config.RecoveryRatioThreshold:
		if isShortSqueeze {
			signal.RegimeClassification = "BULLISH_SHORT_SQUEEZE"
		} else {
			signal.RegimeClassification = "IMPULSIVE_RECOVERY"
		}

	case countUp >= config.TrendThreshold:
		if fundingRate > config.FundingExtreme {
			signal.RegimeClassification = "TRENDING_UP_OVERHEATED"
		} else {
			signal.RegimeClassification = "TRENDING_UP_STRONG"
		}

	case countDown >= config.TrendThreshold:
		signal.RegimeClassification = "TRENDING_DOWN_STRONG"

	case signal.PriceMakingLow50 && signal.AtrPct > config.CapitulationAtrThreshold:
		signal.RegimeClassification = "CAPITULATION_BOTTOM"

	case signal.BoxHeight50 < signal.AtrPct*4 && math.Abs(signal.OiChangePercent5) < config.OiGrowthThreshold:
		signal.RegimeClassification = "RANGE_BOUND"

	default:
		signal.RegimeClassification = "TRANSITIONAL"
	}

	return signal
}

// =============================================================================
// 格式化输出函数
// =============================================================================

// FormatToText 格式化机构市场状态信号为文本（简洁版）
//
// 返回值：
//   - 简洁的机构市场状态信号文本，适合放在 LLM
func (s *InstitutionalRegimeSignal) FormatToText() string {
	if s == nil {
		return ""
	}

	var sb strings.Builder

	sb.WriteString("### Institutional Regime Classifier:\n")

	// 分类结果
	sb.WriteString(fmt.Sprintf("- Regime: %s\n", s.RegimeClassification))

	// 动力学因子
	sb.WriteString(fmt.Sprintf("- EMA20 > EMA50 (last 20): %d/20\n", s.Ema20AboveEma50Count20))
	sb.WriteString(fmt.Sprintf("- OI Change (5-period): %.2f%%\n", s.OiChangePercent5*100))
	if s.PriceMakingLow50 {
		sb.WriteString("- Price Making 50-period Low: Yes\n")
	}

	// 异常分析
	anomalies := []string{}
	if strings.Contains(s.RegimeClassification, "TRENDING_UP") && s.OiChangePercent5 < 0 {
		anomalies = append(anomalies, "- 价格上涨但持仓量 (OI) 下降，暗示上涨动力来自【空头平仓】，而非新多头入场。")
	}
	if s.RegimeClassification == "BULLISH_SHORT_SQUEEZE" {
		anomalies = append(anomalies, "- 检测到【轧空行情】：价格大涨 + 持仓量 (OI) 下降，空头正在平仓。注意：这种上涨可能快拉快跌，持续性较弱。")
	}
	if s.RegimeClassification == "IMPULSIVE_RECOVERY" {
		anomalies = append(anomalies, "- 检测到【脉冲式修复】：价格快速收复 40% 以上的失地，可能预示 V 型反转，但需等待均线确认。")
	}
	if s.RegimeClassification == "CAPITULATION_BOTTOM" {
		anomalies = append(anomalies, "- 检测到【恐慌探底】：价格创 50 周期新低 + 波动率 (ATR) 异常放大，可能是抄底机会。")
	}
	if s.FundingRateExtreme {
		anomalies = append(anomalies, "- 资金费率过高，多头杠杆极度拥挤，注意【多杀多/闪崩】风险。")
	}

	if len(anomalies) > 0 {
		sb.WriteString("\n** Anomalies Detected **\n")
		for _, a := range anomalies {
			sb.WriteString(a + "\n")
		}
	}

	sb.WriteString("\n")
	return sb.String()
}

// =============================================================================
// LLM 战术简报
// =============================================================================

// GenerateLLMBriefing 生成给 LLM 的战术简报（详细版）
//
// 参数说明：
//   - symbol: 交易对名称（如 BTCUSDT）
//   - currentPrice: 当前价格
//   - cvdSlope: CVD 斜率（暂时跳过，传 0 即可）
//
// 返回值：
//   - 格式化的 LLM 战术简报字符串
//
// 简报结构：
//  1. 核心观测数据：价格、OI 变化、CVD 趋势、资金费率
//  2. 算法诊断：分类结果和异常分析
//  3. 复核需求：引导 LLM 进一步思考
func (s *InstitutionalRegimeSignal) GenerateLLMBriefing(symbol string, currentPrice float64, cvdSlope float64) string {
	if s == nil {
		return ""
	}

	var sb strings.Builder

	// 标题
	sb.WriteString(fmt.Sprintf("### 📊 市场战术简报 | 资产：%s | 当前状态：**%s**\n\n", symbol, s.RegimeClassification))

	// 1. 核心观测数据
	sb.WriteString("**1. 核心观测数据：\n")
	sb.WriteString(fmt.Sprintf("- **当前价格：%.2f\n", currentPrice))
	sb.WriteString(fmt.Sprintf("- **持仓量 (OI) 变化：%.2f%% (近 5 周期)\n", s.OiChangePercent5*100))

	// CVD 方向（暂时跳过，预留接口）
	cvdDir := "流入 (买方主动)"
	if cvdSlope < 0 {
		cvdDir = "流出 (卖方主动)"
	}
	sb.WriteString(fmt.Sprintf("- **主动成交 (CVD) 趋势：%s\n", cvdDir))
	sb.WriteString(fmt.Sprintf("- **资金费率：%.4f%% (当前周期)\n\n", s.FundingRate*100))

	// 2. 算法诊断
	sb.WriteString("**2. 算法诊断：\n")
	sb.WriteString(fmt.Sprintf("由逻辑分类器判定当前市场处于 **%s**。\n", s.RegimeClassification))

	// 异常分析
	anomalies := s.detectAnomalies(cvdSlope)
	if len(anomalies) > 0 {
		for _, a := range anomalies {
			sb.WriteString(a + "\n")
		}
	} else {
		sb.WriteString("- 数据表现一致，暂未发现明显的衍生品背离。\n")
	}
	sb.WriteString("\n")

	// 3. 复核需求
	sb.WriteString("**3. 复核需求：\n")
	sb.WriteString("请结合以上【异常分析】和其他数据，判断此信号是否为【假突破】或【不可持续的挤压行情】？\n")

	// 根据不同的分类给出特定的复核提示
	if strings.Contains(s.RegimeClassification, "REVERSING") {
		sb.WriteString("如果是 REVERSING，结合持仓量急剧变动，是否意味着趋势已经彻底反转？\n")
	} else if strings.Contains(s.RegimeClassification, "TRENDING_UP_OVERHEATED") {
		sb.WriteString("当前为过热上涨状态，需警惕资金费率过高导致的回调风险。\n")
	} else if strings.Contains(s.RegimeClassification, "RANGE_BOUND") {
		sb.WriteString("当前为区间震荡状态，等待突破信号或区间边界交易机会。\n")
	} else if s.RegimeClassification == "BULLISH_SHORT_SQUEEZE" {
		sb.WriteString("当前为【轧空行情】，空头正在平仓推动价格上涨。请判断：这种上涨是否具有持续性？还是会快拉快跌？\n")
	} else if s.RegimeClassification == "IMPULSIVE_RECOVERY" {
		sb.WriteString("当前为【脉冲式修复】，价格快速收复失地。请判断：这是 V 型反转的开始，还是反弹后继续下跌？\n")
	} else if s.RegimeClassification == "CAPITULATION_BOTTOM" {
		sb.WriteString("当前为【恐慌探底】，恐慌盘正在抛售。请判断：这是否是抄底的好时机？还是会继续下跌？\n")
	}

	sb.WriteString("\n")
	return sb.String()
}

// detectAnomalies 检测异常背离
//
// 参数说明：
//   - cvdSlope: CVD 斜率（暂时未使用）
//
// 返回值：
//   - 异常描述字符串列表
func (s *InstitutionalRegimeSignal) detectAnomalies(cvdSlope float64) []string {
	anomalies := []string{}

	// 检查：价格上涨但 OI 下降
	if strings.Contains(s.RegimeClassification, "TRENDING_UP") && s.OiChangePercent5 < 0 {
		anomalies = append(anomalies, "- 价格上涨但持仓量 (OI) 下降，暗示上涨动力来自【空头平仓】，而非新多头入场。")
	}

	if s.RegimeClassification == "BULLISH_SHORT_SQUEEZE" {
		anomalies = append(anomalies, "- 检测到【轧空行情】：价格大涨 + 持仓量 (OI) 下降，空头正在平仓。注意：这种上涨可能快拉快跌，持续性较弱。")
	}
	if s.RegimeClassification == "IMPULSIVE_RECOVERY" {
		anomalies = append(anomalies, "- 检测到【脉冲式修复】：价格快速收复 40% 以上的失地，可能预示 V 型反转，但需等待均线确认。")
	}
	if s.RegimeClassification == "CAPITULATION_BOTTOM" {
		anomalies = append(anomalies, "- 检测到【恐慌探底】：价格创 50 周期新低 + 波动率 (ATR) 异常放大，可能是抄底机会。")
	}

	// 检查 CVD 背离（暂时跳过，预留接口）
	// if strings.Contains(s.RegimeClassification, "TRENDING_UP") && cvdSlope < 0 {
	// 	anomalies = append(anomalies, "- 价格上涨但 CVD 下行，存在【买盘枯竭】或【冰山挂单吸收】风险。")
	// }

	// 检查资金费率极端
	if s.FundingRateExtreme {
		anomalies = append(anomalies, "- 资金费率过高，多头杠杆极度拥挤，注意【多杀多/闪崩】风险。")
	}

	return anomalies
}

// =============================================================================
// 摆动点检测与拓扑分析
// =============================================================================

// GenerateTechnicalFeatures 执行完整的特征提取流程
//
// 参数说明：
//   - klines: K 线数据列表
//   - window: 滑动窗口大小（用于检测摆动点）
//
// 返回值：
//   - 技术特征对象，包含摆动点、趋势和机构市场状态
//
// 执行流程：
//  1. 识别摆动点（局部高点和低点）
//  2. 标记拓扑结构（HH/HL/LH/LL）
//  3. 统计支撑阻力位测试次数
//  4. 分析趋势
func GenerateTechnicalFeatures(klines []market.KlineBar, window int) *TechnicalFeatures {
	// 需要足够的数据来识别摆动点
	// 至少需要 window*2+1 根 K 线
	if len(klines) < window*2+1 {
		return &TechnicalFeatures{}
	}

	// 1. 识别摆动点
	swings := findSwingPoints(klines, window)

	// 2. 标记拓扑结构（HH/HL/LH/LL）
	labelSwingTopology(swings)

	// 3. 统计支撑阻力位测试次数
	// 使用 0.2% (0.002) 的容差
	tolerance := 0.002
	for i := range swings {
		// 从摆动点之后的 K 线开始统计测试次数
		startIndex := swings[i].Index + 1
		swings[i].TestCount = countLevelTests(swings[i].Price, swings[i].Type, klines, startIndex, tolerance)
	}

	return &TechnicalFeatures{
		SwingPoints: swings,
		Trend:       analyzeTrend(swings),
	}
}

// findSwingPoints 使用滑动窗口识别局部高点和低点
//
// 参数说明：
//   - klines: K 线数据列表
//   - window: 滑动窗口大小
//
// 返回值：
//   - 摆动点列表
//
// 算法说明：
//   - 对于每个 K 线，检查其左右 window 根 K 线
//   - 如果当前 K 线的 High 是窗口内的最高点，则为 SwingHigh
//   - 如果当前 K 线的 Low 是窗口内的最低点，则为 SwingLow
//   - 左侧使用严格不等号，右侧使用非严格不等号（优先选择最近的峰值）
func findSwingPoints(klines []market.KlineBar, window int) []SwingPoint {
	var swings []SwingPoint

	// 只能在 [window, len-1-window] 范围内识别摆动点
	// 因为需要 window 根 K 线在两侧
	for i := window; i < len(klines)-window; i++ {
		isHigh := true
		isLow := true

		currentHigh := klines[i].High
		currentLow := klines[i].Low

		// 检查左右邻居
		for j := i - window; j <= i+window; j++ {
			if i == j {
				continue
			}
			// 左侧：严格不等号
			if j < i {
				if klines[j].High > currentHigh {
					isHigh = false
				}
				if klines[j].Low < currentLow {
					isLow = false
				}
			} else {
				// 右侧：非严格不等号 + 相等（优先选择最近的峰值）
				// 如果 currentHigh == rightHigh，则当前失败（让右侧的被选中）
				if klines[j].High >= currentHigh {
					isHigh = false
				}
				if klines[j].Low <= currentLow {
					isLow = false
				}
			}
		}

		// 添加 Swing High
		if isHigh {
			swings = append(swings, SwingPoint{
				Price: currentHigh,
				Time:  klines[i].Time,
				Index: i,
				Type:  SwingHigh,
			})
		}
		// 添加 Swing Low
		if isLow {
			swings = append(swings, SwingPoint{
				Price: currentLow,
				Time:  klines[i].Time,
				Index: i,
				Type:  SwingLow,
			})
		}
	}
	return swings
}

// labelSwingTopology 确定高点的 HH/LH 和低点的 HL/LL
//
// 参数说明：
//   - swings: 摆动点列表（按时间顺序）
//
// 算法说明：
//   - 遍历摆动点列表
//   - 对于高点：如果价格高于上一个高点，则标记为 HH（Higher High）
//   - 对于高点：如果价格低于上一个高点，则标记为 LH（Lower High）
//   - 对于低点：如果价格高于上一个低点，则标记为 HL（Higher Low）
//   - 对于低点：如果价格低于上一个低点，则标记为 LL（Lower Low）
func labelSwingTopology(swings []SwingPoint) {
	// 分别跟踪最后看到的高点和低点
	var lastHigh *SwingPoint
	var lastLow *SwingPoint

	// 按时间顺序遍历摆动点
	for i := 0; i < len(swings); i++ {
		// 使用指针直接修改切片中的元素
		s := &swings[i]

		if s.Type == SwingHigh {
			if lastHigh == nil {
				s.Label = "High" // 第一个高点
			} else {
				if s.Price > lastHigh.Price {
					s.Label = "HH" // Higher High
				} else if s.Price < lastHigh.Price {
					s.Label = "LH" // Lower High
				} else {
					s.Label = "EH" // Equal High
				}
			}
			lastHigh = s
		} else { // SwingLow
			if lastLow == nil {
				s.Label = "Low" // 第一个低点
			} else {
				if s.Price > lastLow.Price {
					s.Label = "HL" // Higher Low
				} else if s.Price < lastLow.Price {
					s.Label = "LL" // Lower Low
				} else {
					s.Label = "EL" // Equal Low
				}
			}
			lastLow = s
		}
	}
}

// countLevelTests 统计价格在容差范围内测试某个价位的次数
//
// 参数说明：
//   - level: 支撑/阻力位价格
//   - sType: 摆动点类型（SwingLow=支撑，SwingHigh=阻力）
//   - klines: K 线数据列表
//   - startIndex: 开始统计的索引（从摆动点之后开始）
//   - tolerancePct: 容差百分比（如 0.002 = 0.2%）
//
// 返回值：
//   - 测试次数
//
// 算法说明（支撑位）：
//  1. 价格进入容差区间 [level-tol, level+tol]
//  2. 如果价格反弹并收盘高于 level+tol，则测试成功
//  3. 如果价格跌破 level-tol，则支撑被破坏
//
// 算法说明（阻力位）：
//  1. 价格进入容差区间 [level-tol, level+tol]
//  2. 如果价格回落并收盘低于 level-tol，则测试成功
//  3. 如果价格突破 level+tol，则阻力被突破
func countLevelTests(level float64, sType SwingType, klines []market.KlineBar, startIndex int, tolerancePct float64) int {
	if startIndex >= len(klines) {
		return 0
	}

	testedCount := 0
	upperBound := level * (1 + tolerancePct)
	lowerBound := level * (1 - tolerancePct)

	inTestZone := false

	for i := startIndex; i < len(klines); i++ {
		c := klines[i]

		if sType == SwingLow { // 支撑逻辑
			// 1. 检查价格是否进入容差区间（Low 在范围内）
			// 用户逻辑：if c.Low >= lowerBound && c.Low <= upperBound
			// 扩展：允许略低于但不收盘低于？
			// 用户逻辑："If Low < lowerBound -> broken"。所以严格在边界内为"测试"。

			// 进入测试区间
			if c.Low >= lowerBound && c.Low <= upperBound {
				inTestZone = true
			}

			// 有效的反弹（测试确认）
			// 用户逻辑：if inTestZone && c.Close > upperBound
			if inTestZone && c.Close > upperBound {
				testedCount++
				inTestZone = false // 重置以进行下一次测试
			}

			// 支撑被破坏
			// 用户逻辑：if c.Low < lowerBound
			if c.Low < lowerBound {
				inTestZone = false // 支撑被破坏，停止计数此序列？
				// 通常如果支撑被破坏，它就不再是支撑。
				// 但我们只是停止当前的"测试"状态。未来测试可能是从下方的回测（阻力）？
				// 为简化起见，我们只是重置。
			}

		} else { // SwingHigh (阻力逻辑) - 对称
			// 进入测试区间（High 在范围内）
			if c.High >= lowerBound && c.High <= upperBound {
				inTestZone = true
			}

			// 有效的反弹（测试确认）
			// 对于阻力，我们希望 Close < lowerBound（向下反弹）
			if inTestZone && c.Close < lowerBound {
				testedCount++
				inTestZone = false
			}

			// 阻力被突破
			// 如果 High > upperBound
			if c.High > upperBound {
				inTestZone = false
			}
		}
	}
	return testedCount
}

// analyzeTrend 分析摆动点序列以确定趋势
//
// 参数说明：
//   - swings: 摆动点列表
//
// 返回值：
//   - 趋势描述字符串
//
// 分析逻辑：
//   - 检查最近 6 个摆动点（或全部，如果少于 6 个）
//   - 统计 HH、HL、LH、LL 的数量
//   - 根据模式识别趋势
func analyzeTrend(swings []SwingPoint) string {
	if len(swings) < 3 {
		return "Insufficient data"
	}

	// 分析标签序列
	// 我们想看到连续的 HH+HL 或 LH+LL

	// 获取最近的摆动点
	startIdx := 0
	if len(swings) > 6 {
		startIdx = len(swings) - 6
	}
	recent := swings[startIdx:]

	hh := 0
	hl := 0
	lh := 0
	ll := 0

	for _, s := range recent {
		switch s.Label {
		case "HH":
			hh++
		case "HL":
			hl++
		case "LH":
			lh++
		case "LL":
			ll++
		}
	}

	if hh >= 2 && hl >= 1 {
		return "Strong Uptrend (Consecutive HHs)"
	}
	if lh >= 2 && ll >= 1 {
		return "Strong Downtrend (Consecutive LHs)"
	}
	if hh >= 1 && hl >= 1 {
		return "Uptrend Structure (HH+HL)"
	}
	if lh >= 1 && ll >= 1 {
		return "Downtrend Structure (LH+LL)"
	}

	// 3 点的早期信号检测（例如 H -> L -> LH）
	if len(swings) == 3 {
		last := swings[len(swings)-1]
		if last.Label == "LH" {
			return "Potential Downtrend (Lower High observed)"
		}
		if last.Label == "LL" {
			return "Potential Downtrend (Lower Low observed)"
		}
		if last.Label == "HH" {
			return "Potential Uptrend (Higher High observed)"
		}
		if last.Label == "HL" {
			return "Potential Uptrend (Higher Low observed)"
		}
	}

	return "Mixed/Consolidation"
}

// FormatFeaturesToText 生成用户请求的字符串块
//
// 参数说明：
//   - maxItems: 要显示的最大项目数
//
// 返回值：
//   - 格式化的物理结构锚点字符串
func (f *TechnicalFeatures) FormatFeaturesToText(maxItems int) string {
	if len(f.SwingPoints) == 0 {
		return "### 物理结构锚点：\n(最近数据中未识别摆动点)\n\n"
	}

	var sb strings.Builder

	// 过滤以仅显示最后 N 个项目
	count := len(f.SwingPoints)
	start := 0
	if maxItems > 0 && count > maxItems {
		start = count - maxItems
	}

	// 标题
	sb.WriteString(fmt.Sprintf("### 物理结构锚点（最近 %d 个摆动点）：\n", count-start))

	for i := start; i < count; i++ {
		s := f.SwingPoints[i]

		// 标签格式化：[Higher High (HH)] 或 [Swing High]
		labelStr := ""
		if s.Label != "" && s.Label != "High" && s.Label != "Low" {
			// 将短代码映射到全名
			fullLabel := ""
			switch s.Label {
			case "HH":
				fullLabel = "Higher High"
			case "HL":
				fullLabel = "Higher Low"
			case "LH":
				fullLabel = "Lower High"
			case "LL":
				fullLabel = "Lower Low"
			case "EH":
				fullLabel = "Equal High"
			case "EL":
				fullLabel = "Equal Low"
			default:
				fullLabel = s.Label
			}
			labelStr = fmt.Sprintf("[%s (%s)]", fullLabel, s.Label)
		} else {
			// 首个点的回退
			if s.Type == SwingHigh {
				labelStr = "[Swing High]"
			} else {
				labelStr = "[Swing Low]"
			}
		}

		timeStr := time.Unix(s.Time/1000, 0).UTC().Format("01-02 15:04")

		// 格式化行
		// - [Higher Low (HL)]: 67763.70 | Tested: 3 times (02-22 00:00)
		sb.WriteString(fmt.Sprintf("- %-20s: %-9s | 测试次数：%d 次 (%s)\n",
			labelStr,
			formatPriceForPrompt(s.Price),
			s.TestCount,
			timeStr,
		))
	}

	if f.Trend != "" {
		sb.WriteString(fmt.Sprintf("(趋势证据：%s)\n", f.Trend))
	}
	sb.WriteString("\n")

	return sb.String()
}
