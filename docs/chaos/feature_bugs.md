# features.go 代码审查报告

共发现 **6 个 Bug**，其中 3 个致命、2 个中等、1 个轻微。

---

## Bug 1 ⛔ 致命：`GenerateRegimeSignal` 与摆动点拓扑完全隔离

### 问题

文件中存在两套独立的分析系统，但它们**从不互相通信**：

```
GenerateRegimeSignalWithConfig()  ← 只看 EMA 计数
GenerateTechnicalFeatures()       ← 只看摆动点 (LH+LL)
```

`TechnicalFeatures` struct 里有一个 `Regime *RegimeSignal` 字段，但 `GenerateTechnicalFeatures()` 从来不赋值给它：

```go
// features.go:575 — Regime 字段永远是 nil
return &TechnicalFeatures{
    SwingPoints: swings,
    Trend:       analyzeTrend(swings),  // "Downtrend Structure (LH+LL)" 被计算出来...
    // Regime: ???                       // ...但从不传给 RegimeSignal
}
```

`GenerateRegimeSignalWithConfig()` 的 switch/case 里，`TRENDING_UP_STRONG` 判断完全不看价格结构：

```go
// features.go:329 — 只检查 EMA，不检查 swing topology
case countUp >= config.TrendThreshold:
    signal.RegimeClassification = "TRENDING_UP_STRONG"
    // 没有任何 if swingTrend == "Downtrend Structure (LH+LL)" 的检查
```

### 实际影响

数据中 ARCUSDT 的两个信号同时为真：
- **EMA 计数**：EMA20 > EMA50 持续 20/20 → `TRENDING_UP_STRONG`
- **摆动点**：LH (0.0540) + LL (0.0483) → `Downtrend Structure (LH+LL)`

由于两套系统独立运行，Regime 分类器永远输出 `TRENDING_UP_STRONG`，而 LLM 被 System Prompt 明令禁止质疑计算层，所以这个矛盾一直被忽略——这正是 67 个 cycle 全部做多、0 次做空的根本原因。

### 修复方案

在 `GenerateRegimeSignalWithConfig` 的 switch 中加入结构一致性校验：

```go
case countUp >= config.TrendThreshold:
    // 【修复】新增：与价格结构交叉验证
    if swingTrend == "Downtrend Structure (LH+LL)" || 
       swingTrend == "Strong Downtrend (Consecutive LHs)" {
        // EMA 滞后，价格结构已翻转 → 降级
        signal.RegimeClassification = "TRANSITIONAL"
        signal.StructureConflict = true  // 新增字段：通知 LLM 存在矛盾
    } else if fundingRate > config.FundingExtreme {
        signal.RegimeClassification = "TRENDING_UP_OVERHEATED"
    } else {
        signal.RegimeClassification = "TRENDING_UP_STRONG"
    }
```

需要同时修改函数签名，传入 swingTrend：

```go
func GenerateRegimeSignalWithConfig(
    klines []market.KlineBar,
    ema20Values []float64,
    ema50Values []float64,
    atr14Values []float64,
    oiLatest float64,
    oiAverage float64,
    fundingRate float64,
    config *RegimeClassifierConfig,
    swingTrend string,  // 【新增参数】来自 GenerateTechnicalFeatures().Trend
) *RegimeSignal {
```

或者更彻底的做法：让 `GenerateTechnicalFeatures` 内部调用 `GenerateRegimeSignalWithConfig` 并赋值 `Regime` 字段（填充那个从未被使用的字段）：

```go
func GenerateTechnicalFeatures(klines []market.KlineBar, window int, 
    ema20Values, ema50Values, atr14Values []float64,
    oiLatest, oiAverage, fundingRate float64) *TechnicalFeatures {
    
    swings := findSwingPoints(klines, window)
    labelSwingTopology(swings)
    trend := analyzeTrend(swings)
    
    // 【修复】将 swingTrend 传入 Regime 分类器，实现交叉验证
    regime := GenerateRegimeSignalWithConfig(..., trend)
    
    return &TechnicalFeatures{
        SwingPoints: swings,
        Trend:       trend,
        Regime:      regime,  // 现在被正确填充
    }
}
```

---

## Bug 2 ⛔ 致命：`OiChangePercent5` 计算错误，导致多个 Regime 永远无法触发

### 问题

字段命名和注释明确说是"近 5 周期 OI 变化"：

```go
// features.go:58
OiChangePercent5 float64 // 近 5 周期持仓量（OI）变化百分比
```

但实际计算是：

```go
// features.go:252
signal.OiChangePercent5 = (oiLatest - oiAverage) / oiAverage
```

这里 `oiAverage` 是调用方传入的"均值"，从实际数据来看是一个**长期平均值**（如 20 或 100 周期均值）：

```
OI latest:  530,168,464
OI average: 529,638,295   ← 非常接近，说明是长期均值

OiChangePercent5 = (530168464 - 529638295) / 529638295 = +0.001 (0.1%)
```

而且这个值在 67 个 cycle 中**几乎完全固定在 0.001**，证明调用方传入的是长期均值。

### 实际影响

依赖 `OiChangePercent5` 的三个 Regime 条件完全失效：

| Regime | OI 触发条件 | 实际 OI 值 | 能否触发 |
|--------|-----------|-----------|---------|
| `REVERSING_POTENTIAL_TOP` | `< -0.05` (-5%) | +0.001 | ❌ 永不触发 |
| `BULLISH_SHORT_SQUEEZE` | `< -0.02` (-2%) | +0.001 | ❌ 永不触发 |
| `RANGE_BOUND` | `abs < 0.05` | 0.001 | ✅ 但偶然正确 |

结果：`BULLISH_SHORT_SQUEEZE` 在 67 个 cycle 中出现 0 次，`REVERSING_POTENTIAL_TOP` 出现 0 次，即使市场真的发生了短暂的轧空或顶部反转也无法检测到。

### 修复方案

调用方需传入真正的"5 周期前 OI 值"，而非均值：

```go
// 调用方（正确做法）
oiHistory := getOIHistory(symbol, 5)  // 获取最近 5 根 K 线的 OI
oiBefore5 := oiHistory[0]             // 5 周期前的 OI
oiLatest := oiHistory[len(oiHistory)-1]

signal := GenerateRegimeSignal(klines, ema20, ema50, atr14, oiLatest, oiBefore5, fundingRate)
// 含义变为：(现在 - 5根前) / 5根前，即真正的 5 周期变化率
```

或者在 `GenerateRegimeSignalWithConfig` 内部计算时明确区分：

```go
// features.go 修复
// 参数重命名：oiAverage → oiBefore5Period，语义更清晰
func GenerateRegimeSignalWithConfig(
    ...
    oiLatest float64,
    oiBefore5Period float64,  // 【重命名】5 周期前的 OI，而非均值
    ...
)
```

---

## Bug 3 ⛔ 致命：`TRENDING_UP_OVERHEATED` 判断使用了错误的方向比较

### 问题

```go
// features.go:329-333
case countUp >= config.TrendThreshold:
    if fundingRate > config.FundingExtreme {   // ← 只检查正向资金费率
        signal.RegimeClassification = "TRENDING_UP_OVERHEATED"
    } else {
        signal.RegimeClassification = "TRENDING_UP_STRONG"
    }
```

但 `FundingRateExtreme` 的计算使用了绝对值：

```go
// features.go:254
signal.FundingRateExtreme = math.Abs(fundingRate) > config.FundingExtreme
```

这导致**不一致**：
- `FundingRateExtreme` 字段（用于 REVERSING_POTENTIAL_TOP 判断）会被极端负资金费率触发
- 但 `TRENDING_UP_OVERHEATED` 判断只检查正向（`fundingRate > threshold`），不检查负向

更严重的是 `REVERSING_POTENTIAL_TOP` 条件：

```go
// features.go:319
case signal.PriceMakingHigh50 && (... || signal.FundingRateExtreme):
```

如果价格在 50 周期高点，资金费率极端负值（空头付费给多头），会错误地触发 `REVERSING_POTENTIAL_TOP`。极端负资金费率 + 价格创新高，恰恰是**多头强势、空头被挤**的信号，不应该判定为"顶部反转预警"。

### 修复方案

```go
// 修复 FundingRateExtreme 区分正负
signal.FundingRateExtreme = fundingRate > config.FundingExtreme        // 多头过热
signal.FundingRateExtremeBearish = fundingRate < -config.FundingExtreme // 空头过热（潜在轧空）

// 修复 REVERSING_POTENTIAL_TOP：只在多头资金过热时触发
case signal.PriceMakingHigh50 && (signal.OiChangePercent5 < -config.OiGrowthThreshold || signal.FundingRateExtreme):
    signal.RegimeClassification = "REVERSING_POTENTIAL_TOP"

// 修复 TRENDING_UP_OVERHEATED：与 FundingRateExtreme 保持一致
case countUp >= config.TrendThreshold:
    if signal.FundingRateExtreme {  // 使用同一个字段，而非重复判断
        signal.RegimeClassification = "TRENDING_UP_OVERHEATED"
    } else {
        signal.RegimeClassification = "TRENDING_UP_STRONG"
    }
```

---

## Bug 4 ⚠️ 中等：`countLevelTests` 支撑被破坏后不停止统计

### 问题

```go
// features.go:760-764
if c.Low < lowerBound {
    inTestZone = false // 支撑被破坏
    // 注意：这里只 reset inTestZone，循环继续！
}
```

当支撑位被有效跌破后（`c.Low < lowerBound`），函数只重置了 `inTestZone = false`，但 `for` 循环继续向后扫描所有 K 线。这意味着如果价格随后反弹回来"测试"这个已经被破坏的支撑位，这些测试也会被计入 `testedCount`，导致支撑强度被高估。

对阻力位也有同样问题（`c.High > upperBound` 时不 break）。

### 修复方案

```go
if c.Low < lowerBound {
    break  // 【修复】支撑已破坏，停止统计，返回当前计数
}
```

同理，阻力位：
```go
if c.High > upperBound {
    break  // 【修复】阻力已突破，停止统计
}
```

---

## Bug 5 ⚠️ 中等：`isImpulsiveCross` 只回溯 1 根 K 线，信号时效性极差

### 问题

```go
// features.go:306-308
isImpulsiveCross = currentPrice > currentEma20 && klines[lastIdx-1].Close < currentEma20
```

脉冲交叉信号要求：当前收盘 > EMA20 **且** 上一根收盘 < EMA20。这意味着：
- 信号只在交叉发生的**那根 K 线**有效
- 如果 LLM 的下一个周期（+15 分钟后）才运行，信号已经消失
- 如果交叉发生在 2 根 K 线前，永远检测不到

### 修复方案

扩展回溯窗口到 3 根 K 线：

```go
// 检查最近 3 根 K 线是否有脉冲交叉
lookbackCross := 3
if lookbackCross > lastIdx {
    lookbackCross = lastIdx
}
isImpulsiveCross := false
if currentPrice > currentEma20 {  // 当前必须在 EMA20 上方
    for k := 1; k <= lookbackCross; k++ {
        if klines[lastIdx-k].Close < currentEma20 {
            isImpulsiveCross = true
            break
        }
    }
}
```

---

## Bug 6 🔵 轻微：`RegimeClassifierConfig_Aggressive` 缺少 `New` 前缀

### 问题

```go
// features.go:132 — 缺少 New 前缀
func RegimeClassifierConfig_Aggressive() *RegimeClassifierConfig {

// 对比正确命名
func NewRegimeClassifierConfig_Balanced() *RegimeClassifierConfig {
func NewRegimeClassifierConfig_Conservative() *RegimeClassifierConfig {
```

命名不一致，调用方如果按照 `NewRegimeClassifierConfig_Aggressive()` 调用会编译失败，且 IDE 自动补全也会不一致。

### 修复方案

```go
// 修复：添加 New 前缀
func NewRegimeClassifierConfig_Aggressive() *RegimeClassifierConfig {
```

---

## 修复优先级汇总

| # | 问题 | 严重性 | 修复难度 | 对交易结果的影响 |
|---|------|--------|---------|---------------|
| 1 | Regime 与 swing topology 完全隔离 | ⛔ 致命 | 中等（需重构调用链） | **极大**：直接导致方向性错误 |
| 2 | `OiChangePercent5` 使用长期均值 | ⛔ 致命 | 小（修改调用方传参） | **极大**：ShortSqueeze/ReversalTop 永远失效 |
| 3 | `FundingRateExtreme` abs值逻辑不一致 | ⛔ 致命 | 小（加一个字段） | **中**：极端资金费率判断错误 |
| 4 | `countLevelTests` 不停止 | ⚠️ 中等 | 极小（加 break） | **小**：支撑强度被高估 |
| 5 | `isImpulsiveCross` 只看 1 根 | ⚠️ 中等 | 小（扩展循环） | **中**：脉冲信号大量漏报 |
| 6 | `Aggressive` 缺 `New` 前缀 | 🔵 轻微 | 极小（重命名） | 无（编译错误，不影响运行逻辑） |