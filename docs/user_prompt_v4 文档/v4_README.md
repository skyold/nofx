# Chaos Trading System V4 - Structural Facts Edition
## 结构推理时代 - 完整开发文档包

---

## 🎯 V4是什么？

### 一句话总结

```
V4提供纯结构事实，让LLM做真正的结构推理，
而不是规则匹配（V2）或分数组合（V3）
```

### 核心哲学

```
V2: LLM作为规则匹配器 → "看到'bullish'，考虑做多"
V3: LLM作为分数优化器 → "conviction_score > 0.25，做多"
V4: LLM作为结构推理器 → "HH=3 vs LL=1 + RSI=68 → 可能上升结构"
```

---

## 🔑 V4的三大原则

### 原则1：禁止方向词

❌ **禁止**：
- bullish / bearish
- long / short
- uptrend / downtrend
- rising / falling
- strong / weak
- confirmation / divergence

✅ **允许**：
- higher_high_count
- rsi_value
- ema20_above_ema50
- structure_break_high
- volume_percentile

---

### 原则2：必须可验证

每个signal必须：
1. 有明确计算公式
2. 无主观参数
3. 可用代码复现
4. 不依赖解释

---

### 原则3：保持结构粒度

只提供：
- 状态（true/false）
- 次数（count）
- 原始值（rsi_value）
- 百分位（percentile）

❌ **禁止**：
- trend_score
- conviction_score
- market_bias

---

## 🏗️ V4的6层结构

```json
{
  "signals": {
    "structure": {      // 结构层：HH/LL统计、突破检测
      "higher_high_count_5": 3,
      "structure_break_high": true,
      "range_compression_ratio": 1.85
    },
    
    "momentum": {       // 动量层：RSI、EMA、MACD原始值
      "rsi_value": 68.5,
      "rsi_percentile_200": 85,
      "ema20_above_ema50": true,
      "ema20_slope": 42.3
    },
    
    "volatility": {     // 波动层：ATR、BB宽度
      "atr_value": 312.5,
      "atr_percentile_200": 78,
      "body_ratio": 0.64
    },
    
    "liquidity": {      // 流动性层：成交量、扫单检测
      "volume_spike": true,
      "liquidity_sweep_low": true,
      "volume_percentile_200": 82
    },
    
    "positioning": {    // 持仓结构（可选）
      "long_short_ratio": 1.85,
      "oi_change_percent": 3.4
    },
    
    "ranking": {        // 横向排名（可选）
      "volume_rank_24h": 12,
      "volatility_rank_24h": 5
    }
  }
}
```

---

## ⚡ 快速开始（3步）

### 1️⃣ 复制System Prompt

打开 `v4_system_prompt.md`，复制全部内容到系统提示词。

**包含内容**：
- 6层signal的完整定义
- 每个指标的计算公式
- LLM推理示例

---

### 2️⃣ 理解数据格式

查看 `v4_data_format_example.json`：

**关键特点**：
- ✅ 无方向词
- ✅ 无综合评分
- ✅ 完全可验证
- ✅ 判断权在LLM

---

### 3️⃣ 实施开发

参考 `v4_usage_guide.md`：

**内容**：
- Go/Python代码示例
- 每层signal的实现
- 验证清单
- 最佳实践

---

## 📊 V2 vs V3 vs V4 对比

| 特性 | V2 | V3 | **V4** |
|------|----|----|--------|
| **设计哲学** | 规则匹配 | 分数优化 | **结构推理** |
| **方向词** | ✅ 有 | ✅ 有（正负号） | ❌ **无** |
| **综合评分** | ⚠️ 部分 | ✅ 有 | ❌ **无** |
| **信息量** | ⭐⭐⭐ | ⭐⭐ | ⭐⭐⭐⭐⭐ |
| **LLM推理空间** | ⭐⭐ | ⭐⭐ | ⭐⭐⭐⭐⭐ |
| **可验证性** | ⭐⭐⭐ | ⭐⭐ | ⭐⭐⭐⭐⭐ |
| **人类可读** | ⭐⭐⭐⭐⭐ | ⭐⭐⭐ | ⭐⭐⭐⭐ |
| **Token效率** | ⭐⭐⭐⭐ | ⭐⭐⭐⭐⭐ | ⭐⭐⭐ |

---

## 💡 实际案例对比

**场景**：ETHUSDT，最近5根3次创新高，1次创新低，RSI=68

### V2的数据
```json
{
  "market_regime": "trending_bullish",  // 已判断
  "momentum": "rising"                  // 已判断
}
```

**LLM推理**：看到标签 → 规则匹配 → 做多

---

### V3的数据
```json
{
  "regime_score": 0.65,                 // 已评分
  "conviction_score": 0.28              // 已评分
}
```

**LLM推理**：比较数值 → 0.28 > 0.25 → 做多

---

### **V4的数据**
```json
{
  "structure": {
    "higher_high_count_5": 3,           // 事实
    "lower_low_count_5": 1              // 事实
  },
  "momentum": {
    "rsi_value": 68,                    // 原始值
    "rsi_percentile_200": 85,           // 统计位置
    "ema20_above_ema50": true           // 关系
  },
  "liquidity": {
    "volume_spike": true,               // 行为
    "liquidity_sweep_low": true         // 检测
  }
}
```

**LLM推理**（真正的结构推理）：
```
结构层：HH=3 vs LL=1 → 更多创新高
动量层：RSI=68在85%分位 → 动量强但未超买
流动性：扫了低点 + 成交量放大 → 可能是止损猎杀

→ 综合推理：可能从"假破低"转向"真破高"
→ 考虑在回调时做多，但需确认
```

**关键差异**：V4的LLM在做真正的多维度结构推理！

---

## 🎯 适用场景

### ✅ 应该使用V4

1. **充分发挥LLM能力**
   - Claude/GPT-4o等高级模型
   - 需要复杂推理
   - 追求决策质量

2. **系统工程级项目**
   - 长期维护
   - 团队协作
   - 需要可审计

3. **未来兼容性**
   - 为模型升级做准备
   - 保持最大灵活性
   - 信息无损

---

### ⚠️ 暂不建议V4

1. **简单规则策略**
   - 固定规则
   - 不需要复杂推理

2. **极致Token效率**
   - V4比V2/V3多用1.5-2倍Token
   - 但信息量多5倍

3. **人工Review重度**
   - 需要快速理解决策
   - V2的标签更直观

---

## 🚀 实施路线图

### Phase 1: 学习理解（1-2周）

```
✓ 阅读 v4_README.md
✓ 理解 v4_system_prompt.md
✓ 查看 v4_data_format_example.json
✓ 掌握6层结构含义
```

---

### Phase 2: 原型开发（2-4周）

```
✓ 实现structure层计算
✓ 实现momentum层计算
✓ 实现volatility层计算
✓ 实现liquidity层计算
✓ 单元测试所有signal
```

---

### Phase 3: 并行运行（1-2月）

```
✓ V2/V3与V4同时计算
✓ 对比LLM推理质量
✓ 收集推理案例
✓ 调优signal定义
```

---

### Phase 4: 渐进切换（2-3月）

```
Month 1: 10%流量 → V4
Month 2: 50%流量 → V4
Month 3: 90%流量 → V4
```

---

## 📚 文档导航

### 新手入门
1. **v4_README.md** ← 从这里开始（本文）
2. **v4_data_format_example.json** ← 看数据格式
3. **v4_comparison.md** ← 理解与V2/V3的差异

### 深入学习
4. **v4_system_prompt.md** ← 学习所有signal定义
5. **v4_usage_guide.md** ← 代码实现指南

---

## 💡 核心优势

### 1. 信息无损
```
V2: "trending_bullish" → 丢失了结构细节
V3: 0.65 → 压缩了多维信息
V4: {HH=3, LL=1, RSI=68, ...} → 完整保留
```

### 2. 判断权在LLM
```
V2/V3: 已经预判了方向
V4: 只提供事实，LLM自己推理
```

### 3. 完全可验证
```
V2: "bullish" - 如何验证？
V3: 0.65 - 如何复现？
V4: HH=3 - 数K线即可验证
```

### 4. 未来兼容
```
随着LLM推理能力提升，V4的价值会越来越大
```

---

## ⚠️ 重要提醒

### 1. Token消耗
```
V2: 80 tokens
V3: 60 tokens
V4: 180 tokens（多3倍）

但信息量多5倍！
信息密度：V4 >> V2 > V3
```

### 2. 需要强LLM
```
V4需要LLM有强推理能力
适合Claude Sonnet 4及以上
不适合简单模型
```

### 3. 开发成本
```
V4的signal计算更复杂
需要更多测试
但长期维护更容易
```

---

## ✅ 检查清单

开始使用V4前：

- [ ] 理解V4的三大原则
- [ ] 理解6层结构设计
- [ ] 能解释"为什么禁止方向词"
- [ ] 能解释"为什么禁止综合评分"
- [ ] 知道V4与V2/V3的根本差异
- [ ] 有强推理能力的LLM
- [ ] 有完整的测试计划

---

## 🎓 学习路径

### 初学者（1周）
1. 阅读本文（v4_README.md）
2. 查看数据示例（v4_data_format_example.json）
3. 理解基本概念

### 中级（2-4周）
4. 学习signal定义（v4_system_prompt.md）
5. 理解每层的作用
6. 对比V2/V3差异（v4_comparison.md）

### 高级（1-2月）
7. 实现所有signal计算
8. 集成到系统
9. 优化和调试

---

## 🎯 总结

### V4的本质

```
把"判断"变成"事实"
把"评分"变成"证据"
把"压缩"变成"保留"

让LLM做它最擅长的事：结构推理
```

### 何时使用V4

```
当前：V2（稳定、成熟）
未来：V4（潜力、长期价值）
极致：V3（性能优化）
```

### 核心价值

```
V4是为"结构推理时代"准备的架构

随着LLM推理能力提升，
V4的价值会越来越大
```

---

**准备好开始V4之旅了吗？打开 `v4_system_prompt.md` 深入学习！** 🚀

---

## 📝 版本信息

- **版本**：V4.0
- **状态**：实验性/未来架构
- **适合**：高级LLM + 系统工程级项目
- **创建日期**：2026-02-17
