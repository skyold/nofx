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

## 📁 文件结构（按功能分组）

### 核心入口与类型
| 文件 | 说明 | 主要结构/函数 |
|------|------|---------------|
| **engine.go** | **核心引擎，统一入口** | `ChaosEngine`, 所有对外方法 |
| **types.go** | 核心数据类型定义 | `Decision`, `DecisionResult`, `ChaosContext`, `ChaosConfig` |
| **utils.go** | 通用工具函数 | `formatPriceForPrompt()`, `formatFloatSlice()` |
| **00_README.md** | 快速导航文档 | - |

### 核心业务逻辑
| 文件 | 说明 | 主要结构/函数 |
|------|------|---------------|
| **manager.go** | 业务逻辑管理器 | `Manager`, `BuildUserPrompt()`, `ValidateDecision()`, `IsChaosMode()` |
| **executor.go** | 交易执行器 | `ChaosExecutor`, `ExecuteDecisions()` |
| **parser.go** | AI 响应解析器 | `ExtractDecisions()`, `ExtractReasoningJSON()`, `ExtractReasoning()` |

### System Prompt（系统提示词）
| 文件 | 说明 | 主要结构/函数 |
|------|------|---------------|
| **prompt_system.go** | 系统提示词构建 | `BuildSystemPrompt()` |
| **prompt_contract.go** | 系统执行协议定义 | `GenerateOutputSchema()`, `SystemExecutionContract` |

### User Prompt（用户提示词）
| 文件 | 说明 | 主要结构/函数 |
|------|------|---------------|
| **prompt_user.go** | 用户提示词入口（版本分发） | `BuildUserPrompt()` - 根据配置选择版本 |
| **prompt_user_v1.go** | V1 版本用户提示词（文本格式） | `buildUserPromptV1()` |
| **prompt_user_v2.go** | V2 版本用户提示词（JSON 格式）- 默认 | `buildUserPromptV2()` |
| **prompt_user_v4.go** | V4 版本用户提示词（结构事实版） | `buildUserPromptV4()` |
| **prompt_user_test.go** | User Prompt 单元测试 | - |
| **prompt_builder.go** | Builder 模式实现 | `MarketDataBuilder`, `BasicBuilder`, `EnhancedBuilder` |
| **prompt_data.go** | Prompt 数据结构定义 | `MarketPromptData`, `AccountPromptData`, `PositionPromptData` |
| **prompt_formatter_text.go** | 文本格式化器 | 格式化市场数据为文本 |
| **prompt_helpers.go** | 通用辅助函数 | 构建头部、账户状态、持仓等辅助函数 |
| **prompt_types_json.go** | JSON 类型定义 | JSON 相关的类型定义 |

### 数据加工与特征工程
| 文件 | 说明 | 主要结构/函数 |
|------|------|---------------|
| **features.go** | 特征工程（技术指标计算） | `GenerateTechnicalFeatures()`, `GenerateInstitutionalRegimeSignal()`, `SwingPoint` |

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

### 按功能查找
- **想了解对外接口？** → 查看 [engine.go](file:///Users/zhengningdai/workspace/skyold/nofx/chaos/engine.go)
- **想修改交易执行逻辑？** → 查看 [executor.go](file:///Users/zhengningdai/workspace/skyold/nofx/chaos/executor.go)
- **想修改决策验证规则？** → 查看 [manager.go](file:///Users/zhengningdai/workspace/skyold/nofx/chaos/manager.go#L114-L339) 中的 `ValidateDecision()`
- **想修改 AI 响应解析？** → 查看 [parser.go](file:///Users/zhengningdai/workspace/skyold/nofx/chaos/parser.go)
- **想修改数据结构？** → 查看 [types.go](file:///Users/zhengningdai/workspace/skyold/nofx/chaos/types.go)
- **想修改通用工具函数？** → 查看 [utils.go](file:///Users/zhengningdai/workspace/skyold/nofx/chaos/utils.go)

### 按 Prompt 查找
- **想修改 System Prompt？** → 查看 [prompt_system.go](file:///Users/zhengningdai/workspace/skyold/nofx/chaos/prompt_system.go)
- **想修改系统执行协议？** → 查看 [prompt_contract.go](file:///Users/zhengningdai/workspace/skyold/nofx/chaos/prompt_contract.go)
- **想修改 User Prompt 版本分发？** → 查看 [prompt_user.go](file:///Users/zhengningdai/workspace/skyold/nofx/chaos/prompt_user.go)
- **想修改 V1 版本（文本）？** → 查看 [prompt_user_v1.go](file:///Users/zhengningdai/workspace/skyold/nofx/chaos/prompt_user_v1.go)
- **想修改 V2 版本（JSON）？** → 查看 [prompt_user_v2.go](file:///Users/zhengningdai/workspace/skyold/nofx/chaos/prompt_user_v2.go)
- **想修改 V4 版本（结构事实）？** → 查看 [prompt_user_v4.go](file:///Users/zhengningdai/workspace/skyold/nofx/chaos/prompt_user_v4.go)
- **想修改 Prompt 构建器？** → 查看 [prompt_builder.go](file:///Users/zhengningdai/workspace/skyold/nofx/chaos/prompt_builder.go)
- **想修改 Prompt 数据结构？** → 查看 [prompt_data.go](file:///Users/zhengningdai/workspace/skyold/nofx/chaos/prompt_data.go)
- **想修改 Prompt 格式化？** → 查看 [prompt_formatter_text.go](file:///Users/zhengningdai/workspace/skyold/nofx/chaos/prompt_formatter_text.go)

### 按数据加工查找
- **想修改特征工程？** → 查看 [features.go](file:///Users/zhengningdai/workspace/skyold/nofx/chaos/features.go)

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

## 📊 文件分组总览

```
chaos/
├── 核心入口与类型
│   ├── engine.go                    # 核心引擎（统一入口）
│   ├── types.go                     # 核心类型定义
│   ├── utils.go                     # 通用工具函数
│   └── 00_README.md                 # 快速导航（本文档）
│
├── 核心业务逻辑
│   ├── manager.go                   # 业务逻辑管理器
│   ├── executor.go                  # 交易执行器
│   └── parser.go                    # AI 响应解析器
│
├── System Prompt（系统提示词）
│   ├── prompt_system.go             # 系统提示词构建
│   └── prompt_contract.go           # 系统执行协议
│
├── User Prompt（用户提示词）
│   ├── prompt_user.go               # 用户提示词入口（版本分发）
│   ├── prompt_user_v1.go            # V1 版本（文本格式）
│   ├── prompt_user_v2.go            # V2 版本（JSON 格式）- 默认
│   ├── prompt_user_v4.go            # V4 版本（结构事实版）
│   ├── prompt_user_test.go          # 用户提示词测试
│   ├── prompt_builder.go            # Builder 模式实现
│   ├── prompt_data.go               # Prompt 数据结构定义
│   ├── prompt_formatter_text.go     # 文本格式化器
│   ├── prompt_helpers.go            # 通用辅助函数
│   └── prompt_types_json.go         # JSON 类型定义
│
└── 数据加工与特征工程
    └── features.go                  # 特征工程（原始数据 → 指标 → 特征）
```

## 🎯 文件命名规范

- **核心文件**：直接使用功能名（如 `engine.go`, `manager.go`）
- **Prompt 相关文件**：使用 `prompt_` 前缀区分（如 `prompt_system.go`, `prompt_user.go`）
- **User Prompt 版本**：使用 `prompt_user_vX.go` 格式（如 `prompt_user_v2.go`）
- **工具函数**：使用 `utils.go`
