# Chaos 模块架构与功能说明

本文档详细说明了 Chaos 交易模块（`nofx/chaos/`）的代码结构、各模块职责以及核心执行流程。该架构经过重构，旨在实现**职责分离（Separation of Concerns）**，特别是明确了 `Manager` 作为“Prompt 总管”的角色，以及 `Engine` 和 `Executor` 在执行层面的分工。

## 📂 模块结构总览

Chaos 模块位于 `nofx/chaos/` 目录下，主要包含以下四个功能层：

### 1. 核心引擎层 (Core Engine)
负责协调整个策略执行流程，是外部调用的唯一入口。

| 文件 | 主要职责 | 关键方法/核心逻辑 |
| :--- | :--- | :--- |
| **`engine.go`** | **指挥官**。协调数据获取、Prompt 构建、LLM 调用、结果解析和验证。 | - `Execute(ctx, mcpClient)`: 执行全流程。<br>- **设计变更**: `BuildUserPrompt` 现在只是一个代理方法，直接委托给 `Manager` 实现。 |

### 2. 管理与 Prompt 构建层 (Management & Prompts)
负责策略配置管理、模式识别以及核心的 Prompt (System & User) 构建。这是重构的重点区域，实现了 Prompt 构建逻辑的统一管理。

| 文件 | 主要职责 | 关键方法/核心逻辑 |
| :--- | :--- | :--- |
| **`manager.go`** | **大管家**。统一管理所有 Prompt 构建逻辑和配置参数。 | - `BuildUserPrompt(ctx)`: 统一入口，根据配置分发到不同版本实现。<br>- `BuildSystemPrompt`: 构建系统提示词。<br>- `IsChaosMode`: 判断策略模式。 |
| **`user_prompt.go`** | **Prompt 分发器**。定义 User Prompt 的统一接口和版本路由逻辑。 | - 接收者: `(m *Manager)`。<br>- 根据 `prompt_version` 路由到 v1/v2/v4 实现。 |
| **`user_prompt_legacy.go`** | **V1 实现**。旧版文本格式 Prompt 构建逻辑。 | - 接收者: `(m *Manager)`。 |
| **`user_prompt_v2.go`** | **V2 实现**。JSON 格式 Prompt 构建逻辑 (当前稳定版)。 | - 接收者: `(m *Manager)`。<br>- 包含 `buildMarketRankings`, `buildIndicators` 等核心数据格式化方法。 |
| **`user_prompt_v4.go`** | **V4 实现**。结构化事实版 Prompt (实验性)。 | - 接收者: `(m *Manager)`。 |
| **`system_prompt.go`** | **System Prompt 实现**。构建系统级指令和约束。 | - 接收者: `(m *Manager)`。 |

### 3. 执行层 (Execution)
负责将抽象的交易决策转化为具体的交易所 API 调用。

| 文件 | 主要职责 | 关键方法/核心逻辑 |
| :--- | :--- | :--- |
| **`executor.go`** | **执行者**。负责具体的下单、平仓操作。 | - **方法重命名**: `ExecuteDecisions` (原 `Execute`)，以消除与 `Engine.Execute` 的歧义。<br>- 只关注 `[]Decision` 的执行，不关心决策来源。 |

### 4. 数据与基础设施层 (Data & Infrastructure)
提供数据结构定义、解析、校验和特征工程支持。

| 文件 | 主要职责 | 关键方法 |
| :--- | :--- | :--- |
| **`parser.go`** | **解析器**。处理 LLM 返回的非结构化文本，提取 JSON 决策。 | `ExtractDecisions`, `ExtractReasoningJSON` |
| **`features.go`** | **特征工程**。计算 Swing Points 等技术特征。 | `GenerateTechnicalFeatures` |
| **`contract.go`** | **协议定义**。定义 System Execution Contract 文本。 | `GenerateOutputSchema` |
| **`types.go`** | **类型定义**。定义 `ChaosContext`, `Decision`, `DecisionResult` 等核心数据结构。 | - |

---

## 🔄 核心流程流转 (Refactored Flow)

以下是 Chaos 周期执行的核心调用链：

1.  **启动**: `AutoTrader` 调用 `ChaosEngine.Execute(ctx)`.
2.  **构建 Prompt**:
    *   `Engine` 调用 `m.manager.BuildSystemPrompt(ctx)`.
    *   `Engine` 调用 `m.manager.BuildUserPrompt(ctx)`.
    *   `Manager` 根据配置 (`prompt_version`) 路由到 `user_prompt_v2.go` 等具体实现来构建 User Prompt。
3.  **LLM 交互**: `Engine` 将 Prompts 发送给 LLM 并获取响应。
4.  **解析与验证**: `Engine` 调用 `parser.go` 解析结果，并使用 `Manager` 进行逻辑验证 (Risk Control)。
5.  **执行**: `AutoTrader` (或上层调用者) 拿到结果后，实例化 `ChaosExecutor` 并调用 **`ExecuteDecisions`** 执行交易。

---

## ✨ 架构优化亮点

*   **Manager 权责统一**: `Manager` 现在完全掌控 Prompt 的生成（System + User），不再是一个空壳，符合 "Manager" 的命名含义。
*   **消除歧义**: `Engine.Execute` (引擎流程) vs `Executor.ExecuteDecisions` (交易动作) 区分开来，代码可读性显著提升。
*   **Engine 瘦身**: `ChaosEngine` 回归到流程编排者的角色，不再包含具体的 Prompt 拼接逻辑，降低了模块间的耦合度。
