# Chaos 模块快速导航

本目录包含 Chaos 交易策略的核心实现代码。

## 🏗️ 架构设计

**核心原则**：所有外部调用都通过 `ChaosEngine` 进行，`Manager` 等模块作为内部工作模块。

```
┌─────────────────────────────────────────────────────────────┐
│                     外部调用者                                │
│   (trader, api, backtest)                                   │
└─────────────────────┬───────────────────────────────────────┘
                      │
                      ▼
┌─────────────────────────────────────────────────────────────┐
│                   ChaosEngine (统一入口)                     │
│  ┌─────────────────────────────────────────────────────────┐│
│  │ 对外暴露的方法:                                          ││
│  │ • NewChaosEngine()        - 构造函数                    ││
│  │ • BuildSystemPrompt()     - 构建系统提示词              ││
│  │ • BuildUserPrompt()       - 构建用户提示词              ││
│  │ • GetCandidateCoins()     - 获取候选币种                ││
│  │ • IsChaosMode()           - 检测 Chaos 模式             ││
│  │ • ValidateDecision()      - 验证决策                    ││
│  │ • GetVariantParams()      - 获取变体参数                ││
│  │ • ExtractDecisions()      - 提取决策                    ││
│  │ • ExtractReasoningJSON()  - 提取推理过程                ││
│  │ • Execute()               - 完整执行流程                ││
│  └─────────────────────────────────────────────────────────┘│
└─────────────────────┬───────────────────────────────────────┘
                      │
                      ▼
┌─────────────────────────────────────────────────────────────┐
│               内部工作模块 (不直接对外暴露)                   │
│  ┌──────────────┐  ┌──────────────┐  ┌──────────────┐      │
│  │   Manager    │  │    Parser    │  │   Executor   │      │
│  │  (业务逻辑)  │  │  (响应解析)  │  │  (交易执行)  │      │
│  └──────────────┘  └──────────────┘  └──────────────┘      │
└─────────────────────────────────────────────────────────────┘
```

## 📁 文件结构

### 核心入口层 (对外暴露)
| 文件 | 说明 | 主要结构/函数 |
|------|------|---------------|
| **engine.go** | **核心引擎，统一入口** | `ChaosEngine`, 所有对外方法 |

### 内部工作层
| 文件 | 说明 | 主要结构/函数 |
|------|------|---------------|
| **manager.go** | 业务逻辑管理器 | `Manager`, `BuildUserPrompt()`, `ValidateDecision()` |
| **executor.go** | 交易执行器 | `ChaosExecutor`, `ExecuteDecisions()` |
| **parser.go** | AI 响应解析器 | `ExtractDecisions()`, `ExtractReasoningJSON()` |

### Prompt 构建层
| 文件 | 说明 | 主要结构/函数 |
|------|------|---------------|
| **prompt_system.go** | 系统提示词构建 | `BuildSystemPrompt()` |
| **prompt_user.go** | 用户提示词入口（版本分发） | `BuildUserPrompt()` |
| **prompt_user_legacy.go** | V1 版本用户提示词（文本格式） | `buildUserPromptLegacy()` |
| **prompt_user_v2.go** | V2 版本用户提示词（JSON 格式）- 默认 | `buildUserPromptV2()` |
| **prompt_user_v4.go** | V4 版本用户提示词（结构事实版） | `buildUserPromptV4()` |

### 工具与类型层
| 文件 | 说明 | 主要结构/函数 |
|------|------|---------------|
| **types.go** | 核心数据类型定义 | `Decision`, `DecisionResult`, `ChaosContext`, `ChaosConfig` |
| **features.go** | 特征工程（技术指标计算） | `GenerateTechnicalFeatures()`, `SwingPoint` |
| **contract.go** | 系统执行协议定义 | `GenerateOutputSchema()`, `SystemExecutionContract` |

## 🔄 核心流程

```
1. ChaosEngine.Execute() 启动
   ↓
2. Engine 调用 Manager 构建 System Prompt + User Prompt
   ↓
3. 调用 LLM 获取 AI 响应
   ↓
4. Engine 调用 Parser 解析响应，提取 Decisions
   ↓
5. Engine 调用 Manager.ValidateDecision() 验证决策
   ↓
6. Engine 调用 Executor.ExecuteDecisions() 执行交易
```

## 📌 快速查找

- **想了解对外接口？** → 查看 `engine.go`
- **想修改 Prompt 构建逻辑？** → 查看 `manager.go` 和 `prompt_*.go`
- **想修改交易执行逻辑？** → 查看 `executor.go`
- **想修改决策验证规则？** → 查看 `manager.go` 中的 `ValidateDecision()`
- **想修改数据结构？** → 查看 `types.go`
- **想修改 AI 响应解析？** → 查看 `parser.go`

## 🔗 外部调用示例

```go
// 创建引擎
engine := chaos.NewChaosEngine(config)

// 检测 Chaos 模式
if engine.IsChaosMode(customPrompt) {
    // 构建 Prompt
    systemPrompt := engine.BuildSystemPrompt(equity, variant)
    userPrompt := engine.BuildUserPrompt(ctx)
    
    // 获取候选币种
    coins, _ := engine.GetCandidateCoins()
    
    // 验证决策
    size, err := engine.ValidateDecision(decision, reasoning, equity, riskConfig)
}
```
