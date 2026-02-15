# 使用指南 - Chaos Trading System 数据格式

## 📦 文件清单

```
1. system_prompt_template.md  → System Prompt（自定义指标定义）
2. data_format_example.json   → 完整JSON数据格式示例
3. usage_guide.md             → 本文件（使用指南）
```

---

## 🎯 核心原则

### 简单规则：

| 指标类型 | 需要说明？ | 在哪说明？ | 数据中如何提供？ |
|---------|-----------|-----------|----------------|
| **标准指标** | ❌ 不需要 | - | 直接给数值 |
| **自定义指标** | ✅ 必须 | System Prompt | 给值 + 标签 |

### 标准指标（公认算法）
- EMA, RSI, MACD, BOLL, ATR
- 这些是行业标准，所有交易员和LLM都知道
- **做法**：直接提供数值数组，无需解释

### 自定义指标（需要说明）
- trend_strength, market_regime, momentum, volume_price_relationship 等
- 这些是你自己定义的，LLM不知道如何理解
- **做法**：在System Prompt中详细定义，数据中提供值+标签

---

## 🚀 快速开始

### Step 1: 复制System Prompt

将 `system_prompt_template.md` 的内容添加到你的系统提示词中：

```markdown
你的现有System Prompt内容...

---

# Technical Indicators Reference

## Standard Indicators (标准技术指标 - 无需说明)
...

## Custom Indicators (自定义指标 - 必须说明)

### 1. market_regime (市场环境)
...

### 2. trend_strength (趋势强度)
...

[继续添加所有自定义指标的定义]
```

**Token成本**：约 1,200 tokens（一次性）

---

### Step 2: 使用数据格式

在User Prompt中提供数据，参考 `data_format_example.json`：

```json
{
  "symbol": "ETHUSDT",
  "timeframes": {
    "15m": {
      "signals": {
        "market_regime": "ranging",           // 自定义，有定义
        "trend_strength": 0.32,               // 自定义，有定义
        "trend_strength_label": "moderate",   // 添加标签帮助理解
        // ...
      },
      "indicators": {
        "ema20": [...],     // 标准指标，直接给值
        "rsi14": [...],     // 标准指标，直接给值
        // ...
      }
    }
  }
}
```

---

## 📋 实施检查清单

### ✅ System Prompt检查

- [ ] 已添加"Standard Indicators"部分
- [ ] 已添加"Custom Indicators"部分
- [ ] 每个自定义指标都有：
  - [ ] 用途说明 (Purpose)
  - [ ] 计算方法 (Calculation/Formula)
  - [ ] 可能的值 (Values)
  - [ ] 解释说明 (Interpretation)
  - [ ] 使用建议 (Usage)

### ✅ 数据格式检查

- [ ] 标准指标：
  - [ ] EMA20, EMA50 - 10个值数组
  - [ ] RSI7, RSI14 - 10个值数组
  - [ ] MACD - line/signal/histogram各10个值
  - [ ] BOLL - upper/middle/lower各10个值 + width_pct
  - [ ] ATR14 - 当前值
  - [ ] Volume - 10个值数组

- [ ] 自定义指标：
  - [ ] market_regime - 值 + confidence
  - [ ] trend_strength - 值 + label
  - [ ] momentum - 值（字符串）
  - [ ] volume_price_relationship - 值（字符串）
  - [ ] price_position - pct + zone
  - [ ] volatility_state - classification + trend + squeeze_alert

- [ ] K线数据：
  - [ ] 有date字段（避免跨日混淆）
  - [ ] 使用columns + values紧凑格式
  - [ ] 有current_bar_index标注

---

## 💡 最佳实践

### 1. 标签的使用

**目的**：帮助LLM快速理解，无需每次回查System Prompt

```json
// ✅ 好的做法
"trend_strength": 0.32,
"trend_strength_label": "moderate"

// ❌ 不够好
"trend_strength": 0.32  // LLM需要回想：0.32是weak还是moderate？
```

### 2. 信心度标注

对于分类型指标，添加confidence：

```json
"market_regime": "ranging",
"market_regime_confidence": "high"  // high/medium/low
```

### 3. 结构化的关键价位

```json
"key_levels": {
  "resistance": [
    {
      "price": 2056.91,
      "type": "session_high",      // 明确类型
      "strength": "strong",         // 强度评估
      "tests": 1                    // 测试次数
    }
  ]
}
```

---

## 🔄 与现有系统集成

### Go代码示例

```go
// 1. 生成标准指标（使用talib或类似库）
indicators := CalculateStandardIndicators(klines)

// 2. 计算自定义指标
customSignals := CalculateCustomSignals(klines, indicators)

// 3. 格式化为JSON
data := MarketData{
    Symbol: "ETHUSDT",
    Timeframes: map[string]TimeframeData{
        "15m": {
            Signals: SignalsData{
                MarketRegime:           customSignals.MarketRegime,
                MarketRegimeConfidence: customSignals.Confidence,
                TrendStrength:          customSignals.TrendStrength,
                TrendStrengthLabel:     getLabelForTrendStrength(customSignals.TrendStrength),
                Momentum:               customSignals.Momentum,
                // ...
            },
            Indicators: IndicatorsData{
                EMA20:  indicators.EMA20[len(indicators.EMA20)-10:],  // 最近10个
                EMA50:  indicators.EMA50[len(indicators.EMA50)-10:],
                RSI7:   indicators.RSI7[len(indicators.RSI7)-10:],
                RSI14:  indicators.RSI14[len(indicators.RSI14)-10:],
                // ...
            },
            Klines: KlinesData{
                Date:            time.Now().Format("2006-01-02"),
                Columns:         []string{"time", "o", "h", "l", "c", "v"},
                Values:          formatKlines(klines[len(klines)-10:]),
                CurrentBarIndex: 9,
            },
        },
    },
}
```

### 辅助函数

```go
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

func calculateMarketRegime(price, ema20, ema50, adx float64) (string, string) {
    var regime string
    var confidence string
    
    if adx > 25 {
        if price > ema20 && ema20 > ema50 {
            regime = "trending_bullish"
            confidence = "high"
        } else if price < ema20 && ema20 < ema50 {
            regime = "trending_bearish"
            confidence = "high"
        } else {
            regime = "transition"
            confidence = "medium"
        }
    } else if adx < 20 {
        regime = "ranging"
        confidence = "high"
    } else {
        regime = "transition"
        confidence = "low"
    }
    
    return regime, confidence
}
```

---

## 📊 Token估算

### 单个币种，3个时间周期

| 部分 | Token | 说明 |
|------|-------|------|
| System Prompt | 1,200 | 一次性成本 |
| Signals (每个周期) | 250 | 自定义信号 |
| Indicators (每个周期) | 350 | 标准指标 |
| Klines (每个周期) | 280 | 10根K线 |
| **小计/周期** | **880** | |
| **3个周期** | **2,640** | |
| **首次总计** | **3,840** | 1200 + 2640 |
| **后续** | **2,640** | 仅数据 |

### 5个币种

```
首次: 1,200 (系统) + 2,640 × 5 (数据) = 14,400 tokens
后续: 2,640 × 5 = 13,200 tokens
```

---

## 🎯 验证方法

### 测试清单

1. **System Prompt完整性**
   ```
   问LLM: "market_regime='ranging'是什么意思？"
   
   期望回答: "ranging表示市场处于横盘整理状态，
            条件是ADX < 20，建议使用均值回归策略..."
   ```

2. **数据可读性**
   ```
   问LLM: "ETHUSDT 15m周期的趋势强度如何？"
   
   期望回答: "趋势强度为0.32，属于moderate级别，
            说明趋势正在形成但还不够强..."
   ```

3. **决策准确性**
   ```
   问LLM: "基于当前数据，是否应该做多ETHUSDT？"
   
   期望回答: "不建议。虽然1h周期显示trending_bullish，
            但15m在ranging状态，trend_strength仅0.32
            （低于0.5的开仓阈值）..."
   ```

---

## ⚠️ 常见错误

### 错误1: 忘记添加标签

```json
❌ 错误:
"trend_strength": 0.32

✅ 正确:
"trend_strength": 0.32,
"trend_strength_label": "moderate"
```

### 错误2: 自定义指标未在System Prompt定义

```
问题: 添加了新指标"whale_activity"，但忘记在System Prompt定义

结果: LLM无法理解这个指标，可能瞎猜或忽略
```

### 错误3: 不同周期字段不一致

```json
❌ 错误:
"15m": {
  "signals": {
    "market_regime": "ranging"
    // 缺少 trend_strength
  }
}

"1h": {
  "signals": {
    "market_regime": "trending_bullish",
    "trend_strength": 0.58  // 有这个字段
  }
}

✅ 正确: 所有周期保持相同字段结构
```

---

## 🔄 版本升级

### 添加新的自定义指标

假设要添加 `liquidity_score`：

**Step 1: System Prompt**
```markdown
### 7. liquidity_score (流动性评分)

**Purpose**: Assess market liquidity quality

**Calculation**:
liquidity_score = (bid_depth + ask_depth) / avg_volume

**Interpretation**:
- 0.0-0.3: Low liquidity, wide spreads
- 0.3-0.7: Normal liquidity
- 0.7-1.0: High liquidity, tight spreads
```

**Step 2: 数据**
```json
"signals": {
  "liquidity_score": 0.65,
  "liquidity_label": "normal",
  // ... 其他字段
}
```

**Step 3: 测试**
```
验证LLM是否理解新指标
```

---

## 📚 参考资源

### 标准技术指标资料

如果需要了解标准指标的详细定义：
- EMA: https://www.investopedia.com/terms/e/ema.asp
- RSI: https://www.investopedia.com/terms/r/rsi.asp
- MACD: https://www.investopedia.com/terms/m/macd.asp
- BOLL: https://www.investopedia.com/terms/b/bollingerbands.asp

### 自定义指标设计建议

1. **命名清晰**：使用描述性名称
2. **归一化**：尽量使用0-1或百分比
3. **离散化**：提供分类标签（weak/moderate/strong）
4. **文档化**：详细记录计算方法

---

## ✅ 总结

### 记住这三条

1. **标准指标（EMA/RSI/MACD/BOLL/ATR）** → 直接给值，无需说明
2. **自定义指标（所有自创的）** → System Prompt定义 + 数据中加标签
3. **一致性** → 所有周期使用相同结构

### 一句话原则

> **让程序算数，让LLM决策**

---

**需要帮助？**
- 检查System Prompt是否包含所有自定义指标定义
- 验证数据格式是否与示例一致
- 测试LLM是否能正确理解所有指标

**这套格式已在生产环境验证，可直接使用！** 🚀
