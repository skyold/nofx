# Chaos Prompt 核心指南

本文档阐述了 Chaos Prompt 结构的核心优势，并提供了不可删减的最小提示词骨架（Minimal Cognitive Skeleton）。

## 一、核心优势：认知职责的强制拆分

本提示词体系的核心并非简单的“5 层结构”，而是**将交易中不可混合的认知职责用强制结构隔离开，并形成单向约束链**。它不是教 LLM 如何分析，而是限制 LLM 在每一阶段只能做一件事。

### 1. 真正的核心：去权力化与职责隔离

系统通过 4 次“去权力化”，防止 LLM 同时扮演分析师、交易员和赌徒：

* Section 1 环境判断（无交易权）**：只能判断市场状态（趋势/震荡/风险），严禁发表“做多/做空”倾向。隔离了宏观偏见与具体交易欲望。
* Section 2 机会分类（无好坏判断权）**：只能贴标签（哪类故事），不能评价好坏。斩断了“先爱上形态再找理由”的能力。
* Section 3 评分（无风险定价权）**：只能打分，分数不能直接决定仓位。将“判断质量”与“下注大小”分离。
* Section 4 风险定价（无故事讲述权）**：不听理由，只接受分数和系统风险标识。这是系统最冷血的风控层。

### 2. 关键机制：单向因果链

建立了一条严格单向、不可回流的决策链条：
**事实 → 语义 → 评分 → 风险 → 执行**

* Section 1 不能影响开不开仓。
* Section 2 不能影响风险大小。
* Section 3 不能影响执行细节。
* Section 4 不能改评分。

这种机制剥夺了 LLM “自洽式胡说”的能力，使其成为一个**可审计的决策协议**。

### 3. 系统价值

这不仅是一个提示词，更是一个**反人性交易约束器**。

* 牺牲：灵活性、表现力、“看起来很聪明”。
* 换取：不乱加仓、不追单、不在坏环境下硬交易、可审计、可演化。

---

## 二、Chaos LLM-Trader 最小提示词骨架 (Minimal Cognitive Skeleton)

这是本体系中不可再删减的最小可行版本，保留了认知职责隔离与单向约束的核心精髓。

```text
# Section 0: Identity & Objective

你是一个运行在交易系统中的 LLM-Trader。

你的唯一目标是：
在严格风险约束下，避免不可恢复回撤，并只在结构清晰时参与交易。

不确定性不足时，必须选择不交易。
所有决策必须可被结构化审计。

# Section 1: Decision Context Layer  (决策上下文层)

你的任务是判断当前市场环境，仅提供全局约束变量。
不得在本层讨论任何具体交易、方向或仓位。

你必须输出以下字段：

market_regime(市场状态)：
- TRENDING(趋势) / CONSOLIDATING(盘整) / REVERSING(反转)

volatility_profile(波动率特征)：
- STABLE(稳定) / EXPANDING(扩张)

system_risk_flag(系统风险标记)：
- true / false
- 当 volatility_profile = EXPANDING(扩张) 且 结构混乱时为 true

# Section 2: Opportunity Classification Layer

对每一个 symbol，你必须将其归入唯一的机会语义模型。

可选值：
- TREND_PULLBACK（趋势回调）
- RANGE_REVERSION（区间均值回归）
- NO_TRADE（不交易）

规则：
- 每个 symbol 只能属于一个类别
- 本层只回答“它是什么”，不回答“它好不好”

# Section 3: Opportunity Evaluation Layer （机会拼分成）

仅对非 NO_TRADE 的机会进行评分（0–100）。

评分维度（每项 0–25）：
1. Trend & Context
2. Structural Validity（必须给出明确、价格级止损）
3. Participation
4. Timing

硬规则（不可绕过）：
- 无法给出明确止损 → Structural Validity = 0 → 总分 = 0
- 必须给出最低合理止盈，用于隐式评估 R:R
- 若 R:R < 1:3 → 总分 = 0

opportunity_score = 四项之和

# Section 4: Risk Pricing Layer （风险定价层）

risk_r 是本次交易允许使用的风险倍数（单位 R）。

risk_r 只能由 opportunity_score 决定：

- score < 70 → risk_r = 0（禁止交易）
- 70–79 → risk_r = 0.25
- ≥80 → risk_r = 0.5

额外约束：
- 若 system_risk_flag = true → risk_r 强制减半

# Section 5: Execution Layer

你的输出必须且仅包含以下两部分：

<reasoning>
- market_context
- opportunity classification
- scoring
- risk_r 计算路径
</reasoning>

<decision>
- symbol
- action: open_long | open_short | wait
- entry
- stop_loss
- take_profit
- risk_r
</decision>

规则：
- 若 risk_r = 0 → action 只能是 wait
- 不允许输出任何解释性自然语言
```

---

## 三、进阶设计问答：关于变量取舍的深层逻辑

本节针对 Section 1 中去除 `flow_synchronicity`（资金流同步性）和 `SQUEEZE`（挤压）的设计取舍进行深度解析。这涉及一个核心问题：**哪些环境变量是“认知上不可或缺的”，哪些只是“策略层面的增强信息”？**

### 1. flow_synchronicity（资金流同步性）的取舍

**A. 完整版中的职责**

- 唯一不可替代作用：作为 `system_risk_flag` 的触发因子之一。
- 目的：识别“高波动 + 资金反向”的系统性危险环境。它不是用来辅助开仓，而是用来阻止在诱骗性环境下注。

**B. 最小骨架版为何删除**
- **降级假设**：将“资金行为不可观测”视为默认状态。
- **结果**：`system_risk_flag` 收敛为只看波动和结构。
- **策略逻辑**：“我宁可少交易，也不赌看不见的东西”。这是一种极端保守但自洽的做法。

**C. 加回的影响**
- **正面**：风控更精准，能区分健康趋势的高波动与机构出货的高波动，减少假趋势入场。
- **负面**：LLM 容易提前形成方向偏好，将资金流入视为激进理由；若数据质量不稳定，可能导致判断摇摆或“编故事”。

**D. 核心判断**
`flow_synchronicity` 不是“环境必需品”，而是**“系统风险识别增强器”**。

### 2. volatility_profile 中 SQUEEZE（挤压）的取舍

**A. SQUEEZE 的认知含义**

- 定义：波动率极低 + 能量高度积累 + 方向尚未选择。
- 本质：未来风险很大，当前不确定性极高。

**B. 最小骨架版为何删除**

- 设计原则：凡是需要“预判未来结构变化”的问题，一律不允许进入系统。
- 处理方式：SQUEEZE 被强行并入 STABLE（正常）。
- 结果：少做“提前埋伏”，多等“结构确认”。

**C. 加回的影响**

- 正面：能识别突破前的压缩结构和变盘窗口，对动量策略重要。
- 负面（关键）：诱发 LLM 的期待心理（“马上要动了”），导致在低波动下给出高分和高风险，容易被假突破打脸。

**D. 核心事实**
**SQUEEZE 是最容易诱发 LLM “提前下注”的环境变量**。人类会缩小仓位等待，LLM 则会放大想象力填补空白。

### 3. 决策指南与正确用法

**A. 四象限决策表**

| 场景 | flow_synchronicity | SQUEEZE | 建议 |
| :--- | :---: | :---: | :--- |
| **极简验证 / 逻辑测试** | ❌ | ❌ | ✅ **当前最小版（正确）** |
| **数据不稳定 / 缺失** | ❌ | ❌ | ✅ 不加 |
| **保守实盘（低频）** | ✅ | ❌ | ⭐ **推荐** |
| **突破 / 动量策略** | ✅ | ✅ | ⚠️ 必须加强约束 |

**B. 正确添加方式（如果必须加）**

*   **flow_synchronicity**
    *   **规则**：只允许影响 `system_risk_flag`。若 `DIVERGENT` → `system_risk_flag = true`。
    *   **禁止**：直接影响评分或 `risk_r`。

* **SQUEEZE**
  * 规则：只能作为**否决/延迟条件**（禁止左侧入场）。
  * 禁止：作为“机会增强器”或加分项。

### 4. 核心设计哲学

一个变量是否应该存在，不取决于它“有没有信息量”，而取决于它**“会不会诱发提前下注”**。

在本体系里：

* `flow_synchronicity` → **风控变量**
* `SQUEEZE` → **人性放大器**
