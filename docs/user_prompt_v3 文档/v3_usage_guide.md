# V3 Numeric Engine - 使用指南
## Chaos Trading System - 数值引擎版本

---

## 📦 文档清单

```
1. v3_system_prompt.md        → System Prompt（自定义指标定义）
2. v3_data_format_example.json → 完整JSON数据格式示例
3. v3_usage_guide.md          → 本文件（使用指南）
4. v3_README.md               → 总览文档
```

---

## 🎯 V3的核心理念

### 设计哲学

```
V2: LLM作为规则匹配器 (Rule Matcher)
    → "看到'ranging'和'weak_rally'，所以不做多"

V3: LLM作为分数优化器 (Score Optimizer)  
    → "计算conviction_score=0.18 < 0.25，所以不做多"
```

### 关键特性

| 特性 | V2 | V3 |
|------|----|----|
| 数据类型 | 数值 + 分类标签 | **纯数值** |
| 可微性 | ❌ | ✅ |
| 梯度优化 | ❌ | ✅ |
| 强化学习友好 | ⚠️ 需要转换 | ✅ 原生支持 |
| 人类可读性 | ⭐⭐⭐⭐⭐ | ⭐⭐⭐ |
| 模型优化 | ⭐⭐ | ⭐⭐⭐⭐⭐ |

---

## 🚀 快速开始

### Step 1: 复制System Prompt

将 `v3_system_prompt.md` 的内容添加到你的系统提示词中。

**关键内容**：
- 6个纯数值指标的完整定义
- 每个指标的公式和范围
- 组合决策框架（最重要！）

**Token成本**：约 1,500 tokens（比V2稍多，因为包含数学公式）

---

### Step 2: 理解数值范围

V3的所有指标都是连续数值，必须理解范围：

```python
# 核心指标范围
regime_score:               [-1, +1]  # 市场结构
trend_strength_score:       [0, 1]    # 趋势强度
momentum_score:             [-1, +1]  # 动量方向和强度
volume_confirmation_score:  [-1, +1]  # 量价确认
price_position_score:       [-1, +1]  # 价格位置
volatility_score:           [0, 1]    # 波动率

# 组合指标
directional_bias:           [-1, +1]  # = regime_score
confirmation_bias:          [-1, +1]  # 加权组合
conviction_score:           [0, 1]    # 最终信心度
```

---

### Step 3: 使用组合框架

V3最大的优势是提供了**清晰的数学组合公式**：

```python
# 第一步：计算方向性偏向
directional_bias = regime_score

# 第二步：计算确认性偏向
confirmation_bias = (
    0.4 × momentum_score +
    0.3 × volume_confirmation_score +
    0.3 × price_position_score
)

# 第三步：计算信心度
conviction_score = (
    trend_strength_score ×
    abs(directional_bias) ×
    abs(confirmation_bias)
)

# 第四步：决策
if directional_bias > 0.3 and confirmation_bias > 0.2 and conviction_score > 0.25:
    → Long Setup (做多)
elif directional_bias < -0.3 and confirmation_bias < -0.2 and conviction_score > 0.25:
    → Short Setup (做空)
else:
    → Avoid (避免交易)
```

---

## 📋 实施检查清单

### ✅ System Prompt检查

- [ ] 已添加所有6个数值指标的定义
- [ ] 每个指标都有：
  - [ ] 明确的公式
  - [ ] 数值范围
  - [ ] 解释说明
- [ ] 包含组合决策框架
- [ ] 包含仓位管理规则

### ✅ 数据格式检查

- [ ] 标准指标（与V2相同）：
  - [ ] EMA20, EMA50 - 10个值数组
  - [ ] RSI7, RSI14 - 10个值数组
  - [ ] MACD - line/signal/histogram各10个值
  - [ ] BOLL - upper/middle/lower各10个值
  - [ ] ATR14, ADX14 - 当前值
  - [ ] Volume - 10个值数组

- [ ] 数值指标（V3特有）：
  - [ ] regime_score - 单个数值 [-1, +1]
  - [ ] trend_strength_score - 单个数值 [0, 1]
  - [ ] momentum_score - 单个数值 [-1, +1]
  - [ ] volume_confirmation_score - 单个数值 [-1, +1]
  - [ ] price_position_score - 单个数值 [-1, +1]
  - [ ] volatility_score - 单个数值 [0, 1]

- [ ] 组合指标：
  - [ ] directional_bias
  - [ ] confirmation_bias
  - [ ] conviction_score

- [ ] 决策辅助（可选但推荐）：
  - [ ] is_long_setup
  - [ ] is_short_setup
  - [ ] should_avoid
  - [ ] suggested_size

---

## 💡 代码集成

### Go实现示例

```go
package main

import "math"

// ========== 数据结构 ==========

type NumericSignals struct {
    RegimeScore              float64 `json:"regime_score"`
    TrendStrengthScore       float64 `json:"trend_strength_score"`
    MomentumScore            float64 `json:"momentum_score"`
    VolumeConfirmationScore  float64 `json:"volume_confirmation_score"`
    PricePositionScore       float64 `json:"price_position_score"`
    VolatilityScore          float64 `json:"volatility_score"`
    
    // 组合指标
    DirectionalBias    float64 `json:"directional_bias"`
    ConfirmationBias   float64 `json:"confirmation_bias"`
    ConvictionScore    float64 `json:"conviction_score"`
}

type DecisionAnalysis struct {
    IsLongSetup   bool    `json:"is_long_setup"`
    IsShortSetup  bool    `json:"is_short_setup"`
    ShouldAvoid   bool    `json:"should_avoid"`
    SuggestedSize float64 `json:"suggested_size"`
    Reason        string  `json:"reason"`
}

// ========== 1. 计算Regime Score ==========

func CalculateRegimeScore(ema20, ema50, close float64, marketStructure MarketStructure) float64 {
    // Component 1: Trend from EMAs
    emaDistance := math.Abs(ema20-ema50) / close
    trendComponent := math.Copysign(1, ema20-ema50) * math.Min(emaDistance/0.01, 1.0)
    
    // Component 2: Market Structure
    structureComponent := 0.0
    if marketStructure.IsHigherHigh {
        structureComponent += 0.5
    }
    if marketStructure.IsHigherLow {
        structureComponent += 0.5
    }
    if marketStructure.IsLowerHigh {
        structureComponent -= 0.5
    }
    if marketStructure.IsLowerLow {
        structureComponent -= 0.5
    }
    
    // Final score
    regimeScore := trendComponent + structureComponent
    return clamp(regimeScore, -1.0, 1.0)
}

// ========== 2. 计算Trend Strength Score ==========

func CalculateTrendStrengthScore(ema20, ema50, close, adx float64) float64 {
    // Component 1: EMA distance
    emaDistance := math.Abs(ema20-ema50) / close
    emaComponent := math.Min(emaDistance/0.01, 1.0)
    
    // Component 2: ADX
    adxComponent := math.Min(adx/50.0, 1.0)
    
    // Weighted average
    return 0.5*emaComponent + 0.5*adxComponent
}

// ========== 3. 计算Momentum Score ==========

func CalculateMomentumScore(close, close5ago, close10ago, atr float64) float64 {
    // Price changes (percentage)
    change5 := (close - close5ago) / close5ago
    change10 := (close - close10ago) / close10ago
    
    // Normalize by ATR
    momentumRaw := 0.6*(change5/(atr/close)) + 0.4*(change10/(atr/close))
    
    // Scale to [-1, +1]
    return clamp(momentumRaw/3.0, -1.0, 1.0)
}

// ========== 4. 计算Volume Confirmation Score ==========

func CalculateVolumeConfirmationScore(close, closePrev, volume, avgVolume float64) float64 {
    // Price change percentage
    pricePct := (close - closePrev) / closePrev
    
    // Volume change percentage
    volumePct := (volume - avgVolume) / avgVolume
    
    // Interaction term
    score := (pricePct * volumePct) / 0.02
    return clamp(score, -1.0, 1.0)
}

// ========== 5. 计算Price Position Score ==========

func CalculatePricePositionScore(close, rangeHigh, rangeLow float64) float64 {
    if rangeHigh == rangeLow {
        return 0.0
    }
    
    positionPct := (close - rangeLow) / (rangeHigh - rangeLow)
    
    // Center around 0: [-1, +1]
    return (positionPct - 0.5) * 2.0
}

// ========== 6. 计算Volatility Score ==========

func CalculateVolatilityScore(bollUpper, bollLower, close, atr, avgATR float64) float64 {
    // Bollinger Band width
    bbWidth := (bollUpper - bollLower) / close
    bbComponent := normalize(bbWidth, 0.01, 0.08)
    
    // ATR ratio
    atrRatio := atr / avgATR
    atrComponent := normalize(atrRatio, 0.8, 1.8)
    
    // Weighted average
    return 0.5*bbComponent + 0.5*atrComponent
}

// ========== 组合决策框架 ==========

func CalculateCompositeSignals(signals *NumericSignals) {
    // Step 1: Directional bias
    signals.DirectionalBias = signals.RegimeScore
    
    // Step 2: Confirmation bias
    signals.ConfirmationBias = (
        0.4*signals.MomentumScore +
        0.3*signals.VolumeConfirmationScore +
        0.3*signals.PricePositionScore,
    )
    
    // Step 3: Conviction score
    signals.ConvictionScore = (
        signals.TrendStrengthScore *
        math.Abs(signals.DirectionalBias) *
        math.Abs(signals.ConfirmationBias),
    )
}

// ========== 决策分析 ==========

func AnalyzeDecision(signals NumericSignals) DecisionAnalysis {
    decision := DecisionAnalysis{}
    
    // Long setup check
    decision.IsLongSetup = (
        signals.DirectionalBias > 0.3 &&
        signals.ConfirmationBias > 0.2 &&
        signals.ConvictionScore > 0.25,
    )
    
    // Short setup check
    decision.IsShortSetup = (
        signals.DirectionalBias < -0.3 &&
        signals.ConfirmationBias < -0.2 &&
        signals.ConvictionScore > 0.25,
    )
    
    // Avoid conditions
    decision.ShouldAvoid = (
        signals.TrendStrengthScore < 0.2 ||
        signals.VolatilityScore < 0.15 ||
        signals.ConvictionScore < 0.15 ||
        math.Abs(signals.DirectionalBias) < 0.2,
    )
    
    // Position sizing
    volatilityPenalty := signals.VolatilityScore * 0.3
    decision.SuggestedSize = signals.ConvictionScore * (1 - volatilityPenalty)
    
    // Reason
    if decision.ShouldAvoid {
        decision.Reason = fmt.Sprintf("Low conviction (%.3f < 0.15), avoid trading", signals.ConvictionScore)
    } else if decision.IsLongSetup {
        decision.Reason = fmt.Sprintf("Long setup: conviction=%.3f, size=%.1f%%", 
            signals.ConvictionScore, decision.SuggestedSize*100)
    } else if decision.IsShortSetup {
        decision.Reason = fmt.Sprintf("Short setup: conviction=%.3f, size=%.1f%%", 
            signals.ConvictionScore, decision.SuggestedSize*100)
    } else {
        decision.Reason = "Conditions not met"
    }
    
    return decision
}

// ========== 辅助函数 ==========

func clamp(value, min, max float64) float64 {
    if value < min {
        return min
    }
    if value > max {
        return max
    }
    return value
}

func normalize(value, minVal, maxVal float64) float64 {
    if maxVal <= minVal {
        return 0.5
    }
    normalized := (value - minVal) / (maxVal - minVal)
    return clamp(normalized, 0.0, 1.0)
}

// ========== 使用示例 ==========

func main() {
    // 1. 计算标准指标（使用talib或自己实现）
    indicators := CalculateStandardIndicators(klines)
    
    // 2. 计算数值指标
    signals := NumericSignals{
        RegimeScore: CalculateRegimeScore(
            indicators.EMA20[len(indicators.EMA20)-1],
            indicators.EMA50[len(indicators.EMA50)-1],
            klines[len(klines)-1].Close,
            detectMarketStructure(klines),
        ),
        TrendStrengthScore: CalculateTrendStrengthScore(
            indicators.EMA20[len(indicators.EMA20)-1],
            indicators.EMA50[len(indicators.EMA50)-1],
            klines[len(klines)-1].Close,
            indicators.ADX[len(indicators.ADX)-1],
        ),
        MomentumScore: CalculateMomentumScore(
            klines[len(klines)-1].Close,
            klines[len(klines)-6].Close,
            klines[len(klines)-11].Close,
            indicators.ATR[len(indicators.ATR)-1],
        ),
        // ... 其他指标
    }
    
    // 3. 计算组合指标
    CalculateCompositeSignals(&signals)
    
    // 4. 决策分析
    decision := AnalyzeDecision(signals)
    
    // 5. 格式化输出
    data := formatToJSON(signals, indicators, klines, decision)
    
    // 6. 发送给LLM
    sendToLLM(data)
}
```

---

## 🔄 从V2迁移到V3

### 映射关系

```go
// V2 → V3 映射函数
func ConvertV2ToV3(v2Data V2Data) NumericSignals {
    v3 := NumericSignals{}
    
    // 1. market_regime → regime_score
    v3.RegimeScore = convertRegimeToScore(v2Data.MarketRegime)
    
    // 2. trend_strength保持不变（但去掉label）
    v3.TrendStrengthScore = v2Data.TrendStrength
    
    // 3. momentum → momentum_score
    v3.MomentumScore = convertMomentumToScore(v2Data.Momentum)
    
    // 4. volume_price_relationship → volume_confirmation_score
    v3.VolumeConfirmationScore = convertVolumeRelationToScore(v2Data.VolumePriceRelationship)
    
    // 5. price_position → price_position_score
    v3.PricePositionScore = convertPricePositionToScore(v2Data.PricePosition)
    
    // 6. volatility_state → volatility_score
    v3.VolatilityScore = convertVolatilityToScore(v2Data.VolatilityState)
    
    return v3
}

// 转换函数
func convertRegimeToScore(regime string) float64 {
    switch regime {
    case "trending_bullish":
        return 0.7
    case "trending_bearish":
        return -0.7
    case "ranging":
        return 0.0
    case "transition":
        return 0.0
    default:
        return 0.0
    }
}

func convertMomentumToScore(momentum string) float64 {
    switch momentum {
    case "accelerating_up":
        return 0.8
    case "rising":
        return 0.4
    case "choppy":
        return 0.0
    case "falling":
        return -0.4
    case "accelerating_down":
        return -0.8
    default:
        return 0.0
    }
}

func convertVolumeRelationToScore(relation string) float64 {
    switch relation {
    case "bullish_confirmation":
        return 0.8
    case "weak_rally":
        return 0.2
    case "neutral":
        return 0.0
    case "weak_selloff":
        return -0.2
    case "bearish_confirmation":
        return -0.8
    default:
        return 0.0
    }
}

func convertPricePositionToScore(position V2PricePosition) float64 {
    // V2的pct_of_range是0-100，转换为[-1, +1]
    return (position.PctOfRange - 50.0) / 50.0
}

func convertVolatilityToScore(volatility V2VolatilityState) float64 {
    switch volatility.Classification {
    case "very_low":
        return 0.15
    case "low":
        return 0.35
    case "medium":
        return 0.55
    case "high":
        return 0.80
    default:
        return 0.50
    }
}
```

---

## 📊 参数优化

V3的所有权重都可以优化：

### 可调参数列表

```python
optimizable_params = {
    # Confirmation Bias权重
    'momentum_weight': 0.4,
    'volume_weight': 0.3,
    'price_position_weight': 0.3,
    
    # Position Sizing
    'volatility_penalty': 0.3,
    
    # Decision Thresholds
    'long_directional_threshold': 0.3,
    'long_confirmation_threshold': 0.2,
    'long_conviction_threshold': 0.25,
    
    'short_directional_threshold': -0.3,
    'short_confirmation_threshold': -0.2,
    'short_conviction_threshold': 0.25,
    
    'avoid_trend_threshold': 0.2,
    'avoid_volatility_threshold': 0.15,
    'avoid_conviction_threshold': 0.15,
    
    # Multi-timeframe weights
    '4h_weight': 0.5,
    '1h_weight': 0.3,
    '15m_weight': 0.2
}
```

### 使用贝叶斯优化

```python
from bayes_opt import BayesianOptimization

def objective_function(momentum_weight, volume_weight, conviction_threshold):
    # 标准化权重
    total = momentum_weight + volume_weight
    price_position_weight = 1.0 - total
    
    # 回测
    returns = backtest_with_params({
        'momentum_weight': momentum_weight / total,
        'volume_weight': volume_weight / total,
        'price_position_weight': price_position_weight / total,
        'conviction_threshold': conviction_threshold
    })
    
    # 返回Sharpe Ratio
    return calculate_sharpe(returns)

# 定义参数空间
pbounds = {
    'momentum_weight': (0.2, 0.6),
    'volume_weight': (0.2, 0.5),
    'conviction_threshold': (0.15, 0.35)
}

# 优化
optimizer = BayesianOptimization(
    f=objective_function,
    pbounds=pbounds,
    random_state=1,
)

optimizer.maximize(init_points=10, n_iter=50)

print("最优参数:", optimizer.max['params'])
```

---

## 🤖 强化学习集成

### State Vector定义

```python
import numpy as np

class TradingEnvironment:
    def get_state(self):
        """返回RL agent的状态向量"""
        return np.array([
            # 数值指标
            self.regime_score,
            self.trend_strength_score,
            self.momentum_score,
            self.volume_confirmation_score,
            self.price_position_score,
            self.volatility_score,
            
            # 组合指标
            self.directional_bias,
            self.confirmation_bias,
            self.conviction_score,
            
            # 标准指标（归一化）
            self.normalize_rsi(self.rsi7),
            self.normalize_rsi(self.rsi14),
            self.normalize_ema_distance(self.ema20, self.ema50),
            
            # 仓位状态
            self.current_position,  # -1, 0, +1
            self.unrealized_pnl,
            self.bars_in_position,
            
            # 风险指标
            self.current_drawdown,
            self.volatility_score
        ])
    
    def normalize_rsi(self, rsi):
        """RSI归一化到[-1, +1]"""
        return (rsi - 50) / 50
    
    def normalize_ema_distance(self, ema20, ema50):
        """EMA距离归一化"""
        distance = (ema20 - ema50) / ema50
        return np.clip(distance / 0.05, -1, 1)
```

### Action Space

```python
# 连续动作空间
class ContinuousAction:
    def __init__(self, direction, size):
        self.direction = direction  # [-1, +1]
        self.size = size            # [0, 1]

# 或离散动作空间
ACTION_SPACE = [
    'do_nothing',
    'long_25%',
    'long_50%',
    'long_75%',
    'long_100%',
    'short_25%',
    'short_50%',
    'short_75%',
    'short_100%',
    'close_position'
]
```

### Reward Function

```python
def calculate_reward(self, action, next_state):
    """计算奖励"""
    # 1. PnL component
    pnl = (next_state.equity - self.equity) / self.equity
    
    # 2. Risk-adjusted reward
    sharpe_reward = pnl / self.volatility_score if self.volatility_score > 0 else 0
    
    # 3. Penalty for high volatility exposure
    volatility_penalty = -0.1 * self.volatility_score * abs(action.size)
    
    # 4. Penalty for overtrading
    trading_penalty = -0.01 if action != 'do_nothing' else 0
    
    # 5. Penalty for drawdown
    drawdown_penalty = -0.5 * max(0, self.current_drawdown - 0.1)
    
    # Total reward
    total_reward = (
        sharpe_reward + 
        volatility_penalty + 
        trading_penalty + 
        drawdown_penalty
    )
    
    return total_reward
```

---

## ⚠️ 常见错误

### 错误1: 忘记归一化

```python
❌ 错误:
regime_score = (ema20 - ema50) / close  # 可能超出[-1, +1]

✅ 正确:
distance = abs(ema20 - ema50) / close
regime_score = clamp(sign(ema20 - ema50) * min(distance / 0.01, 1.0), -1, 1)
```

### 错误2: 权重不归一

```python
❌ 错误:
confirmation_bias = 0.5 * momentum + 0.5 * volume + 0.5 * position  # 总和1.5

✅ 正确:
confirmation_bias = 0.4 * momentum + 0.3 * volume + 0.3 * position  # 总和1.0
```

### 错误3: 忘记取绝对值

```python
❌ 错误:
conviction_score = trend_strength * directional_bias * confirmation_bias
# 如果directional_bias和confirmation_bias符号相反，结果为负

✅ 正确:
conviction_score = trend_strength * abs(directional_bias) * abs(confirmation_bias)
```

### 错误4: 范围检查不足

```python
❌ 危险:
regime_score = calculate_regime()  # 可能因数据错误超出范围

✅ 安全:
regime_score = calculate_regime()
if regime_score < -1 or regime_score > 1:
    log.Error("regime_score out of range:", regime_score)
    return error
```

---

## 🎯 最佳实践

### DO ✅

1. **使用数学库**
   ```go
   import "math"
   // 使用math.Abs, math.Min, math.Max等
   ```

2. **验证范围**
   ```go
   if regime_score < -1 || regime_score > 1 {
       return errors.New("score out of valid range")
   }
   ```

3. **记录所有计算步骤**
   ```go
   log.Debug("Calculating conviction_score:")
   log.Debug("  trend_strength =", trend_strength)
   log.Debug("  directional_bias =", directional_bias)
   log.Debug("  confirmation_bias =", confirmation_bias)
   log.Debug("  conviction_score =", conviction_score)
   ```

4. **A/B测试**
   ```python
   # 同时运行V2和V3，比较结果
   v2_decision = decide_with_v2(data)
   v3_decision = decide_with_v3(data)
   log_comparison(v2_decision, v3_decision)
   ```

### DON'T ❌

1. **不要手动设置标签**
   ```json
   ❌ {
       "regime_score": 0.65,
       "regime_label": "trending_bullish"  // V3不需要标签
   }
   ```

2. **不要混用V2和V3**
   ```go
   ❌ signals := NumericSignals{
       RegimeScore: 0.65,
       MarketRegime: "trending_bullish"  // 不要混用
   }
   ```

3. **不要忽略数据质量检查**
   ```go
   ❌ regime_score := (ema20 - ema50) / close  // close可能为0
   
   ✅ if close == 0 {
       return nil, errors.New("invalid close price")
   }
   regime_score := (ema20 - ema50) / close
   ```

---

## 📈 性能监控

### 关键指标

```python
metrics = {
    # 信号质量
    'avg_conviction_long': 0.35,
    'avg_conviction_short': 0.28,
    'signal_accuracy': 0.62,
    
    # 参数稳定性
    'regime_score_std': 0.15,
    'conviction_score_std': 0.08,
    
    # 决策分布
    'pct_long_setups': 0.15,
    'pct_short_setups': 0.12,
    'pct_avoid': 0.73,
    
    # 性能指标
    'sharpe_ratio': 2.1,
    'win_rate': 0.58,
    'profit_factor': 1.85
}
```

### 异常检测

```python
def detect_anomalies(signals):
    """检测异常信号"""
    anomalies = []
    
    # 检查1: 分数超出范围
    if abs(signals.regime_score) > 1:
        anomalies.append("regime_score out of range")
    
    # 检查2: 矛盾信号
    if signals.directional_bias > 0.5 and signals.momentum_score < -0.5:
        anomalies.append("Strong divergence: positive regime, negative momentum")
    
    # 检查3: 极端信心度
    if signals.conviction_score > 0.9:
        anomalies.append("Extremely high conviction, check data quality")
    
    # 检查4: NaN或Inf
    if math.isnan(signals.conviction_score) or math.isinf(signals.conviction_score):
        anomalies.append("Invalid conviction_score")
    
    return anomalies
```

---

## 🔄 版本迁移路线图

### Phase 1: 并行运行（1个月）

```
Week 1-2: 实现V3计算逻辑
Week 3-4: V2和V3同时运行，记录对比数据
```

### Phase 2: 渐进切换（2个月）

```
Month 2: 10%流量切换到V3
Month 3: 50%流量切换到V3
```

### Phase 3: 完全切换（1个月）

```
Month 4: 90%流量切换到V3，保留V2作为后备
Month 5: 完全切换到V3
```

---

## 🎓 学习资源

### 强化学习

- **书籍**: "Reinforcement Learning" by Sutton & Barto
- **课程**: DeepMind x UCL RL Course
- **库**: Stable-Baselines3, RLlib

### 参数优化

- **贝叶斯优化**: `scikit-optimize`, `hyperopt`
- **遗传算法**: DEAP, PyGAD
- **网格搜索**: scikit-learn GridSearchCV

### 量化交易

- **回测框架**: Backtrader, VectorBT, QuantConnect
- **数据**: CCXT, yfinance, Alpha Vantage

---

## 💬 FAQ

### Q1: V3比V2更好吗？

**A**: 取决于目标
- 追求可解释性 → V2更好
- 追求性能优化 → V3更好
- 建议：中期用混合方案

### Q2: conviction_score多少算高？

**A**: 经验值
- < 0.15: 很低，避免交易
- 0.15-0.25: 低，谨慎交易
- 0.25-0.40: 中等，标准仓位
- 0.40-0.60: 高，可加仓
- > 0.60: 很高，检查是否过拟合

### Q3: 如何调试数值计算错误？

**A**: 逐步验证
```python
# 1. 打印所有中间结果
print("ema20:", ema20)
print("ema50:", ema50)
print("close:", close)
print("regime_score:", regime_score)

# 2. 验证范围
assert -1 <= regime_score <= 1

# 3. 对比V2结果
v2_regime = "ranging"
expected_v3 = 0.0  # ranging应该接近0
assert abs(regime_score - expected_v3) < 0.2
```

### Q4: 能否只用V3的conviction_score？

**A**: 不建议
```python
# 不够细致
if conviction_score > 0.25:
    trade()

# 更好的方式
if (directional_bias > 0.3 and 
    confirmation_bias > 0.2 and 
    conviction_score > 0.25):
    trade_long()
```

---

## ✅ 总结

V3 Numeric Engine是为未来准备的架构：

**适合场景**：
- ✅ 机器学习驱动
- ✅ 强化学习训练
- ✅ 参数自动优化
- ✅ 追求极致性能

**不适合场景**：
- ❌ 需要快速理解决策
- ❌ 合规要求高可解释性
- ❌ 团队不熟悉数值优化

**建议路径**：
1. 短期：使用V2（稳定、可读）
2. 中期：并行运行，积累经验
3. 长期：逐步迁移到V3

**记住**：V3不是替代V2，而是为了不同目标的进化版本！

---

需要帮助？查看 `v3_README.md` 获取快速入门指南。
