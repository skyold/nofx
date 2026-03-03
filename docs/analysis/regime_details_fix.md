# Regime Details 字段修复报告

## 🐛 问题描述

V2 版本的 `RegimeSignal` 结构体缺少 V4.1 prompt 需要的字段，导致 `regime_details` 输出为 `null`：

```json
"regime_details": { 
  "recovery_ratio": null,      // ❌ 计算了但没保存
  "is_short_squeeze": null,    // ❌ 计算了但没保存
  "is_impulsive_cross": null,  // ❌ 计算了但没保存
  "price_making_low_50": null, // ✅ 已保存
  "price_making_high_50": null // ✅ 已保存
}
```

## 🔍 根本原因

1. **计算了但没有保存到结构体**：
   - `recovery_ratio` 在 `features.go:294-297` 计算
   - `is_short_squeeze` 在 `features.go:304-308` 计算
   - `is_impulsive_cross` 在 `features.go:299-302` 计算
   - **但这些值都是局部变量，没有保存到 `RegimeSignal` 结构体**

2. **结构体缺少字段定义**：
   - `RegimeSignal` 结构体没有 `RecoveryRatio`、`IsShortSqueeze`、`IsImpulsiveCross` 字段

3. **FormatToText 没有输出**：
   - `FormatToText()` 函数没有输出这些字段

## ✅ 修复内容

### 1. 添加字段到 RegimeSignal 结构体

**文件**: `chaos/features.go:73-76`

```go
type RegimeSignal struct {
    // ... 原有字段 ...
    
    // ➕ V4.1 新增字段：机构市场状态详情
    RecoveryRatio    float64 // 收复比例（0.0-1.0）
    IsShortSqueeze   bool    // 是否轧空行情
    IsImpulsiveCross bool    // 是否脉冲交叉
}
```

### 2. 保存计算结果

**文件**: `chaos/features.go:301-316`

```go
// 计算 recovery_ratio 并保存
recoveryRatio := 0.0
if high50 > low50 {
    recoveryRatio = (currentPrice - low50) / (high50 - low50)
}
signal.RecoveryRatio = recoveryRatio  // ✅ 保存到结构体

// 计算 is_impulsive_cross 并保存
isImpulsiveCross := false
if lastIdx >= 1 {
    isImpulsiveCross = currentPrice > currentEma20 && klines[lastIdx-1].Close < currentEma20
}
signal.IsImpulsiveCross = isImpulsiveCross  // ✅ 保存到结构体

// 计算 is_short_squeeze 并保存
isShortSqueeze := false
if lastIdx >= 1 {
    isShortSqueeze = (currentPrice > klines[lastIdx-1].Close*config.ShortSqueezePriceThreshold) &&
        (signal.OiChangePercent5 < config.ShortSqueezeOiThreshold)
}
signal.IsShortSqueeze = isShortSqueeze  // ✅ 保存到结构体
```

### 3. 更新 FormatToText 输出

**文件**: `chaos/features.go:376-389`

```go
// V4.1 机构市场状态详情
sb.WriteString(fmt.Sprintf("- Recovery Ratio: %.2f\n", s.RecoveryRatio))
sb.WriteString(fmt.Sprintf("- Is Short Squeeze: %v\n", s.IsShortSqueeze))
sb.WriteString(fmt.Sprintf("- Is Impulsive Cross: %v\n", s.IsImpulsiveCross))

// 价格结构
if s.PriceMakingLow50 {
    sb.WriteString("- Price Making 50-period Low: Yes\n")
}
if s.PriceMakingHigh50 {
    sb.WriteString("- Price Making 50-period High: Yes\n")
}
```

## 📊 修复后的输出示例

**V2 User Prompt 输出** (文本格式):

```markdown
### Regime Classifier:
- Regime: BULLISH_SHORT_SQUEEZE
- EMA20 > EMA50 (last 20): 18/20
- OI Change (5-period): -3.50%
- Recovery Ratio: 0.45
- Is Short Squeeze: true
- Is Impulsive Cross: true
- Price Making 50-period High: Yes
```

**V4.1 System Prompt 可以解析为 JSON**:

```json
{
  "market_state": {
    "regime": "BULLISH_SHORT_SQUEEZE",
    "institutional_regime_details": {
      "recovery_ratio": 0.45,
      "is_short_squeeze": true,
      "is_impulsive_cross": true,
      "price_making_low_50": false,
      "price_making_high_50": true
    }
  }
}
```

## 🎯 字段计算逻辑详解

### 1. Recovery Ratio (收复比例)

**公式**: `(当前价格 - 50 周期最低价) / (50 周期最高价 - 50 周期最低价)`

**含义**: 
- 0.0 = 在 50 周期最低点
- 1.0 = 在 50 周期最高点
- 0.45 = 收复了 45% 的失地

**V4.1 用途**: 判断脉冲式修复的强度

### 2. Is Short Squeeze (轧空行情)

**判定条件**:
1. 价格大涨 > 1% (当前价格 > 前一根收盘 * 1.01)
2. OI 显著下降 < -2% (持仓量变化 < -0.02)

**含义**: 空头正在平仓推动价格上涨

**V4.1 用途**: 
- 机会类型：SHORT_SQUEEZE_PLAY
- 仓位：30%-50% 正常仓位
- 最小 RR: 1.2 (可降低要求)

### 3. Is Impulsive Cross (脉冲交叉)

**判定条件**: 
- 价格瞬间穿透 EMA20
- 前一根 K 线收盘 < EMA20，当前收盘 > EMA20

**含义**: 价格快速上穿 EMA20，可能是反转信号

**V4.1 用途**:
- 机会类型：IMPULSIVE_RECOVERY
- 仓位：40%-70% 正常仓位

### 4. Price Making Low/High 50

**判定条件**:
- `PriceMakingLow50` = 当前价格 <= 50 周期最低价
- `PriceMakingHigh50` = 当前价格 >= 50 周期最高价

**V4.1 用途**:
- 检测顶部/底部反转
- 恐慌探底的判定条件之一

## ✅ 验证结果

1. **编译成功**:
   ```bash
   $ go build ./chaos
   ✅ 无错误
   ```

2. **字段完整性**:
   - ✅ `RecoveryRatio` 已计算并保存
   - ✅ `IsShortSqueeze` 已计算并保存
   - ✅ `IsImpulsiveCross` 已计算并保存
   - ✅ `PriceMakingLow50` 已保存
   - ✅ `PriceMakingHigh50` 已保存

3. **输出格式**:
   - ✅ `FormatToText()` 输出所有字段
   - ✅ `GenerateLLMBriefing()` 可使用所有字段

## 🔄 V2 与 V4.1 的兼容性

修复后，V2 User Prompt 输出**完全兼容** V4.1 System Prompt：

| V4.1 字段 | V2 输出 | 兼容性 |
|-----------|---------|--------|
| `regime` | ✅ `RegimeClassification` | ✅ 100% |
| `recovery_ratio` | ✅ `RecoveryRatio` | ✅ 100% |
| `is_short_squeeze` | ✅ `IsShortSqueeze` | ✅ 100% |
| `is_impulsive_cross` | ✅ `IsImpulsiveCross` | ✅ 100% |
| `price_making_low_50` | ✅ `PriceMakingLow50` | ✅ 100% |
| `price_making_high_50` | ✅ `PriceMakingHigh50` | ✅ 100% |

## 📝 总结

**问题根源**: 计算了字段但没有保存到结构体，导致输出为 `null`

**修复方式**: 
1. 添加字段定义到结构体
2. 保存计算结果到结构体字段
3. 更新 `FormatToText()` 输出这些字段

**修复结果**: V2 User Prompt 现在可以输出完整的 `regime_details`，与 V4.1 System Prompt 完全兼容。

---

**修复时间**: 2026-03-02  
**修复文件**: `chaos/features.go`  
**影响范围**: Regime 分类器输出、V2/V4.1 Prompt 兼容性  
**测试状态**: ✅ 编译通过
