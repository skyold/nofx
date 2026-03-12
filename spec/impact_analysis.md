# 影响分析文档 - 阶段 1

## 1. 概述

本文档分析接口重构对现有代码的影响，识别需要修改的文件、潜在的破坏性变更以及风险缓解措施。

---

## 2. 影响范围评估

### 2.1 受影响的模块

| 模块 | 影响程度 | 说明 |
|------|----------|------|
| `chaos/` | 🔴 高 | 需要实现 Engine 接口，迁移上下文构建 |
| `kernel/` | 🔴 高 | 需要实现 Engine 接口，迁移上下文构建 |
| `trader/` | 🔴 高 | 需要实现 Trader 接口，迁移执行器 |
| `manager/` | 🟡 中 | TraderManager 需要适配新接口 |
| `web/` | 🟢 低 | API 层基本不受影响 |

### 2.2 不受影响的模块

- `store/` - 数据库层，接口不变
- `market/` - 市场数据层，接口不变
- `mcp/` - AI 客户端层，接口不变
- `provider/` - 数据提供商层，接口不变
- `config/` - 配置层，接口不变

---

## 3. 详细影响分析

### 3.1 Chaos 包影响

#### 当前文件结构

```
chaos/
├── engine.go          (需要修改)
├── manager.go         (需要修改)
├── prompt_system.go   (保持不变)
├── prompt_user.go     (保持不变)
├── types.go           (需要重构)
├── parser.go          (需要修改)
├── executor.go        (需要删除)
└── ...
```

#### 需要修改的文件

**1. `chaos/engine.go`**

**当前状态**：
```go
type ChaosEngine struct {
    manager *Manager
    config  *ChaosConfig
}

func (ce *ChaosEngine) BuildSystemPromptWithContext(ctx *ChaosContext) string
func (ce *ChaosEngine) BuildUserPrompt(ctx *ChaosContext) string
func (ce *ChaosEngine) ExtractDecisions(aiResponse string) ([]Decision, error)
func (ce *ChaosEngine) ValidateDecisions(decisions []Decision, ...) error
```

**修改后**：
```go
type ChaosEngine struct {
    builder *ContextBuilder
    config  *ChaosConfig
    mcpClient mcp.Client
}

// 实现 engine.Engine 接口
func (ce *ChaosEngine) Name() string
func (ce *ChaosEngine) BuildContext(ctx context.Context, runtime RuntimeInfo) (*engine.Context, error)
func (ce *ChaosEngine) BuildSystemPrompt(ctx *engine.Context) string
func (ce *ChaosEngine) BuildUserPrompt(ctx *engine.Context) string
func (ce *ChaosEngine) CallLLM(ctx context.Context, systemPrompt, userPrompt string) (*engine.AIResponse, error)
func (ce *ChaosEngine) ParseResponse(response *engine.AIResponse) ([]engine.Decision, error)
func (ce *ChaosEngine) ValidateDecisions(ctx context.Context, decisions []engine.Decision, context *engine.Context) ([]engine.ValidatedDecision, error)
```

**影响**：
- ✅ 新增字段：`builder`, `mcpClient`
- ✅ 新增方法：`BuildContext()`, `CallLLM()`
- ✅ 方法签名变更：`BuildSystemPromptWithContext` → `BuildSystemPrompt`
- ✅ 返回类型变更：使用 `engine.Context` 替代 `ChaosContext`

**风险**：
- 🔴 破坏性变更：外部调用需要修改
- 🟡 需要更新所有测试用例

---

**2. `chaos/manager.go`**

**当前状态**：
```go
type Manager struct {
    // ...
}

func (m *Manager) BuildSystemPrompt() string
func (m *Manager) BuildUserPrompt(ctx *ChaosContext) string
func (m *Manager) ValidateDecision(d *Decision, ...) (float64, error)
```

**修改后**：
```go
// Manager 重命名为 ContextBuilder
type ContextBuilder struct {
    // ...
}

func (cb *ContextBuilder) BuildContext(ctx context.Context, runtime RuntimeInfo) (*engine.Context, error)
func (cb *ContextBuilder) buildSystemPrompt() string
func (cb *ContextBuilder) buildUserPrompt(ctx *engine.Context) string
func (cb *ContextBuilder) validateDecision(d *engine.Decision, ...) (float64, error)
```

**影响**：
- ✅ 类型重命名：`Manager` → `ContextBuilder`
- ✅ 方法可见性变更：公共方法改为私有（由 ChaosEngine 封装）
- ✅ 新增方法：`BuildContext()`

**风险**：
- 🔴 破坏性变更：所有引用 `Manager` 的代码需要修改
- 🟡 需要大量重构

---

**3. `chaos/types.go`**

**当前状态**：
```go
type ChaosContext struct {
    Account        kernel.AccountInfo
    Positions      []kernel.PositionInfo
    CandidateCoins []kernel.CandidateCoin
    MarketDataMap  map[string]*market.Data
    // ...
}

type Decision struct {
    Symbol        string
    Action        string
    Leverage      int
    PositionSizeUSD float64
    // ...
}
```

**修改后**：
```go
// ChaosContext 保留，但使用 engine.Context 作为基础
type ChaosContext struct {
    *engine.Context  // 嵌入通用上下文
    // Chaos 特定字段
    ChaosConfig *ChaosConfig
}

// Decision 类型迁移到 engine.Decision
// 在 chaos 包中使用 engine.Decision
```

**影响**：
- ✅ 类型嵌入：`ChaosContext` 嵌入 `engine.Context`
- ✅ 类型复用：使用 `engine.Decision` 替代本地 `Decision`

**风险**：
- 🟡 需要处理类型转换
- 🟡 需要确保字段兼容性

---

**4. `chaos/parser.go`**

**当前状态**：
```go
func ExtractDecisions(aiResponse string) ([]Decision, error)
func ExtractReasoning(aiResponse string) (string, error)
func ExtractCoTTrace(aiResponse string) string
```

**修改后**：
```go
func (ce *ChaosEngine) ParseResponse(response *engine.AIResponse) ([]engine.Decision, error)
// ExtractReasoning 和 ExtractCoTTrace 合并到 ParseResponse
```

**影响**：
- ✅ 方法合并：多个解析方法合并为 `ParseResponse()`
- ✅ 参数变更：接收 `*engine.AIResponse` 而非 `string`

**风险**：
- 🟢 低风险：内部重构，不影响外部调用

---

**5. `chaos/executor.go`**

**当前状态**：
```go
type ChaosExecutor struct {
    trader trader.Trader
    store  *store.Store
}

func (ce *ChaosExecutor) ExecuteDecisions(decisions []Decision, ctx *ChaosContext) []ExecutionResult
func (ce *ChaosExecutor) executeOpen(decision Decision, ctx *ChaosContext) (*Order, error)
func (ce *ChaosExecutor) executeClose(decision Decision, ctx *ChaosContext) (*Order, error)
```

**修改后**：
- ❌ **删除此文件**
- 内容迁移到 `trader/executor.go`

**影响**：
- 🔴 破坏性变更：所有引用需要修改
- 🟡 需要更新导入路径

**风险**：
- 🔴 高风险：需要确保迁移完整性
- 🟡 需要更新所有测试

---

### 3.2 Kernel 包影响

#### 当前文件结构

```
kernel/
├── engine.go          (需要修改)
├── prompt_builder.go  (需要修改)
├── schema.go          (保持不变)
├── formatter.go       (需要修改)
├── grid_engine.go     (需要修改)
└── validate_test.go   (需要修改)
```

#### 需要修改的文件

**1. `kernel/engine.go`**

**当前状态**：
```go
type StrategyEngine struct {
    config     *StrategyConfig
    trader     trader.Trader
    mcpClient  mcp.Client
    store      *store.Store
    // ... 很多字段
}

func (se *StrategyEngine) BuildSystemPrompt() string
func (se *StrategyEngine) BuildUserPrompt() string
func (se *StrategyEngine) FetchMarketData(symbols []string, timeframes []string) (map[string]*market.Data, error)
// ... 很多方法
```

**修改后**：
```go
type StrategyEngine struct {
    builder    *ContextBuilder
    config     *StrategyConfig
    mcpClient  mcp.Client
    // 移除 trader, store 字段（由 Scheduler 管理）
}

// 实现 engine.Engine 接口
func (se *StrategyEngine) Name() string
func (se *StrategyEngine) BuildContext(ctx context.Context, runtime RuntimeInfo) (*engine.Context, error)
func (se *StrategyEngine) BuildSystemPrompt(ctx *engine.Context) string
func (se *StrategyEngine) BuildUserPrompt(ctx *engine.Context) string
func (se *StrategyEngine) CallLLM(ctx context.Context, systemPrompt, userPrompt string) (*engine.AIResponse, error)
func (se *StrategyEngine) ParseResponse(response *engine.AIResponse) ([]engine.Decision, error)
func (se *StrategyEngine) ValidateDecisions(ctx context.Context, decisions []engine.Decision, context *engine.Context) ([]engine.ValidatedDecision, error)
```

**影响**：
- ✅ 字段简化：移除 `trader`, `store` 等字段
- ✅ 新增字段：`builder`
- ✅ 方法签名变更：适配 Engine 接口

**风险**：
- 🔴 破坏性变更：大量方法签名修改
- 🟡 需要更新 Grid Engine 的调用

---

**2. `kernel/prompt_builder.go`**

**当前状态**：
```go
func (se *StrategyEngine) BuildSystemPrompt() string
func (se *StrategyEngine) BuildUserPrompt() string
```

**修改后**：
```go
type PromptBuilder struct {
    // ...
}

func (pb *PromptBuilder) BuildSystemPrompt(ctx *engine.Context, config *StrategyConfig) string
func (pb *PromptBuilder) BuildUserPrompt(ctx *engine.Context) string
```

**影响**：
- ✅ 提取为独立组件
- ✅ 方法签名变更

**风险**：
- 🟡 中等风险：需要更新调用方

---

**3. `kernel/formatter.go`**

**当前状态**：
```go
func FormatAccountInfo(account AccountInfo) string
func FormatPositions(positions []PositionInfo) string
// ...
```

**修改后**：
```go
func FormatAccountInfo(account *engine.AccountInfo) string
func FormatPositions(positions []engine.PositionInfo) string
// ...
```

**影响**：
- ✅ 参数类型变更：使用 `engine.*` 类型

**风险**：
- 🟢 低风险：类型适配即可

---

**4. `kernel/grid_engine.go`**

**当前状态**：
```go
type GridEngine struct {
    strategyEngine *StrategyEngine
    // ...
}

func (ge *GridEngine) RunCycle() error
```

**修改后**：
```go
type GridEngine struct {
    engine engine.Engine
    // ...
}

func (ge *GridEngine) RunCycle(ctx context.Context) error
```

**影响**：
- ✅ 依赖类型变更：`*StrategyEngine` → `engine.Engine`
- ✅ 方法签名变更：增加 `context.Context`

**风险**：
- 🟡 中等风险：需要确保接口兼容性

---

### 3.3 Trader 包影响

#### 当前文件结构

```
trader/
├── auto_trader.go          (需要简化)
├── auto_trader_chaos.go    (需要简化)
├── auto_trader_grid.go     (需要简化)
├── interface.go            (需要增强)
├── executor.go             (需要新建)
├── scheduler.go            (需要新建)
└── binance/, bybit/, ...   (需要适配)
```

#### 需要修改的文件

**1. `trader/interface.go`**

**当前状态**：
```go
type Trader interface {
    GetExchangeName() string
    GetBalance() (*Balance, error)
    GetPositions() ([]Position, error)
    OpenLong(symbol string, quantity float64) (*Order, error)
    OpenShort(symbol string, quantity float64) (*Order, error)
    CloseLong(symbol string, quantity float64) (*Order, error)
    CloseShort(symbol string, quantity float64) (*Order, error)
    SetLeverage(symbol string, leverage int) error
    SetStopLoss(orderID string, price float64) error
    SetTakeProfit(orderID string, price float64) error
}
```

**修改后**：
```go
type Trader interface {
    GetExchange() string
    GetAccountInfo(ctx context.Context) (*AccountInfo, error)
    GetPositions(ctx context.Context) ([]PositionInfo, error)
    GetMarketPrice(ctx context.Context, symbol string) (float64, error)
    OpenLong(ctx context.Context, symbol string, quantity float64, leverage int) (*Order, error)
    OpenShort(ctx context.Context, symbol string, quantity float64, leverage int) (*Order, error)
    CloseLong(ctx context.Context, symbol string, quantity float64) (*Order, error)
    CloseShort(ctx context.Context, symbol string, quantity float64) (*Order, error)
    SetLeverage(ctx context.Context, symbol string, leverage int) error
    SetStopLoss(ctx context.Context, orderID string, price float64) error
    SetTakeProfit(ctx context.Context, orderID string, price float64) error
    ExecuteDecision(ctx context.Context, decision *engine.ValidatedDecision) (*OrderResult, error)
}
```

**影响**：
- ✅ 方法签名变更：所有方法增加 `context.Context`
- ✅ 新增方法：`GetMarketPrice()`, `ExecuteDecision()`
- ✅ 方法重命名：`GetExchangeName` → `GetExchange`, `GetBalance` → `GetAccountInfo`

**风险**：
- 🔴 高风险：所有交易所实现都需要修改
- 🟡 需要更新 10+ 个交易所包

---

**2. `trader/auto_trader.go`**

**当前状态**：
```go
type AutoTrader struct {
    trader         Trader
    mcpClient      mcp.Client
    store          *store.Store
    strategyEngine *kernel.StrategyEngine
    chaosEngine    *chaos.ChaosEngine
    // ... 很多字段
}

func (at *AutoTrader) Run() {
    // 包含完整的业务逻辑
}

func (at *AutoTrader) RunChaosCycle() error {
    // 1. 构建上下文
    // 2. 构建提示词
    // 3. 调用 LLM
    // 4. 解析响应
    // 5. 验证决策
    // 6. 执行决策
}
```

**修改后**：
```go
type AutoTrader struct {
    scheduler scheduler.Scheduler
    // 简化为适配器
}

func (at *AutoTrader) Run() error {
    return at.scheduler.Start()
}

func (at *AutoTrader) Stop() error {
    return at.scheduler.Stop()
}

// 移除 RunChaosCycle, RunGridCycle 等方法
```

**影响**：
- ✅ 字段简化：移除业务相关字段
- ✅ 方法简化：委托给 Scheduler
- ✅ 职责变更：从执行者变为适配器

**风险**：
- 🟡 中等风险：需要确保功能完整性

---

**3. `trader/auto_trader_chaos.go`**

**当前状态**：
```go
func (at *AutoTrader) RunChaosCycle() error {
    // 1. 构建上下文（100+ 行代码）
    ctx, err := at.buildChaosContext()
    
    // 2. 构建提示词
    systemPrompt := at.chaosEngine.BuildSystemPromptWithContext(ctx)
    userPrompt := at.chaosEngine.BuildUserPrompt(ctx)
    
    // 3. 调用 LLM
    aiResponse, err := at.mcpClient.CallWithMessages(systemPrompt, userPrompt)
    
    // 4. 解析响应
    decisions, _ := at.chaosEngine.ExtractDecisions(aiResponse)
    
    // 5. 验证决策
    for _, d := range decisions {
        at.chaosEngine.ValidateDecision(&d, ...)
    }
    
    // 6. 执行决策
    at.executeChaosDecision(decision, ctx)
}

func (at *AutoTrader) buildChaosContext() (*ChaosContext, error) {
    // 上下文构建逻辑
}

func (at *AutoTrader) executeChaosDecision(decision *chaos.Decision, ctx *ChaosContext) {
    // 决策执行逻辑
}
```

**修改后**：
```go
// 删除此文件所有内容
// 功能迁移到：
// - chaos/context_builder.go (buildChaosContext)
// - chaos/engine.go (BuildSystemPrompt, BuildUserPrompt, CallLLM, ParseResponse, ValidateDecisions)
// - trader/executor.go (executeChaosDecision)
```

**影响**：
- 🔴 删除整个文件
- ✅ 代码迁移到其他包

**风险**：
- 🔴 高风险：需要确保迁移完整性
- 🟡 需要全面测试

---

**4. `trader/executor.go` (新建)**

```go
package trader

import (
    "context"
    "github.com/skyold/nofx/engine"
)

type DecisionExecutor struct {
    trader Trader
    store  *store.Store
}

func NewDecisionExecutor(trader Trader, store *store.Store) *DecisionExecutor {
    return &DecisionExecutor{trader, store}
}

func (de *DecisionExecutor) ExecuteDecision(ctx context.Context, decision *engine.ValidatedDecision) (*OrderResult, error) {
    switch decision.Action {
    case "open_long":
        return de.executeOpen(ctx, decision, "long")
    case "open_short":
        return de.executeOpen(ctx, decision, "short")
    case "close_long":
        return de.executeClose(ctx, decision, "long")
    case "close_short":
        return de.executeClose(ctx, decision, "short")
    }
}

func (de *DecisionExecutor) executeOpen(ctx context.Context, decision *engine.ValidatedDecision, side string) (*OrderResult, error) {
    // 实现细节
}

func (de *DecisionExecutor) executeClose(ctx context.Context, decision *engine.ValidatedDecision, side string) (*OrderResult, error) {
    // 实现细节
}
```

**影响**：
- ✅ 新建文件
- ✅ 复用现有代码

**风险**：
- 🟢 低风险：代码迁移

---

**5. `trader/scheduler.go` (新建)**

```go
package scheduler

import (
    "context"
    "time"
    "github.com/skyold/nofx/engine"
    "github.com/skyold/nofx/trader"
)

type AutoScheduler struct {
    engine   engine.Engine
    trader   trader.Trader
    executor trader.DecisionExecutor
    interval time.Duration
    // ...
}

func (s *AutoScheduler) Start() error {
    // 启动调度循环
}

func (s *AutoScheduler) Stop() error {
    // 停止调度循环
}

func (s *AutoScheduler) runCycle() {
    // 编排流程
}
```

**影响**：
- ✅ 新建文件
- ✅ 复用现有调度逻辑

**风险**：
- 🟢 低风险：代码迁移

---

### 3.4 交易所实现包影响

#### 受影响的包

- `trader/binance/`
- `trader/bybit/`
- `trader/hyperliquid/`
- `trader/okx/`
- `trader/bitget/`
- `trader/gate/`
- `trader/kucoin/`
- `trader/aster/`
- `trader/lighter/`

#### 需要修改的内容

**每个交易所包需要**：

1. **更新方法签名**
   ```go
   // 之前
   func (t *FuturesTrader) OpenLong(symbol string, quantity float64) (*Order, error)
   
   // 之后
   func (t *FuturesTrader) OpenLong(ctx context.Context, symbol string, quantity float64, leverage int) (*Order, error)
   ```

2. **实现新方法**
   ```go
   func (t *FuturesTrader) GetAccountInfo(ctx context.Context) (*trader.AccountInfo, error)
   func (t *FuturesTrader) GetMarketPrice(ctx context.Context, symbol string) (float64, error)
   func (t *FuturesTrader) ExecuteDecision(ctx context.Context, decision *engine.ValidatedDecision) (*trader.OrderResult, error)
   ```

3. **更新类型引用**
   ```go
   // 之前
   import "github.com/skyold/nofx/trader"
   
   // 之后
   import (
       "github.com/skyold/nofx/trader"
       "github.com/skyold/nofx/engine"
   )
   ```

**影响**：
- 🟡 每个包需要修改 10+ 个方法
- 🟡 需要更新 9 个包

**风险**：
- 🔴 高风险：工作量大
- 🟡 需要回归测试

---

## 4. 调用链影响分析

### 4.1 当前调用链

```
main.go
  ↓
manager/trader_manager.go:CreateAutoTrader()
  ↓
trader/auto_trader.go:NewAutoTrader()
  ↓
trader/auto_trader.go:Run()
  ↓
trader/auto_trader_chaos.go:RunChaosCycle()
  ├─→ buildChaosContext()
  ├─→ chaosEngine.BuildSystemPromptWithContext()
  ├─→ chaosEngine.BuildUserPrompt()
  ├─→ mcpClient.CallWithMessages()
  ├─→ chaosEngine.ExtractDecisions()
  ├─→ chaosEngine.ValidateDecision()
  └─→ executeChaosDecision()
       ├─→ trader.OpenLong()
       ├─→ trader.SetStopLoss()
       └─→ trader.SetTakeProfit()
```

### 4.2 重构后调用链

```
main.go
  ↓
manager/trader_manager.go:CreateAutoTrader()
  ↓
trader/auto_trader.go:NewAutoTrader()
  ↓
scheduler/auto_scheduler.go:SetEngine()
scheduler/auto_scheduler.go:SetTrader()
  ↓
scheduler/auto_scheduler.go:Start()
  ↓
scheduler/auto_scheduler.go:runCycle()
  ├─→ engine.BuildContext()
  ├─→ engine.BuildSystemPrompt()
  ├─→ engine.BuildUserPrompt()
  ├─→ engine.CallLLM()
  ├─→ engine.ParseResponse()
  ├─→ engine.ValidateDecisions()
  └─→ trader.ExecuteDecision()
       ├─→ trader.OpenLong()
       ├─→ trader.SetStopLoss()
       └─→ trader.SetTakeProfit()
```

### 4.3 调用链变化总结

| 调用 | 当前调用方 | 新调用方 | 变化 |
|------|------------|----------|------|
| `BuildContext()` | ❌ 无 | `Scheduler.runCycle()` | 新增 |
| `BuildSystemPrompt()` | `AutoTrader.RunChaosCycle()` | `Scheduler.runCycle()` | 调用方变更 |
| `BuildUserPrompt()` | `AutoTrader.RunChaosCycle()` | `Scheduler.runCycle()` | 调用方变更 |
| `CallLLM()` | `AutoTrader.RunChaosCycle()` | `Scheduler.runCycle()` | 调用方变更 |
| `ParseResponse()` | `AutoTrader.RunChaosCycle()` | `Scheduler.runCycle()` | 调用方变更 |
| `ValidateDecisions()` | `AutoTrader.RunChaosCycle()` | `Scheduler.runCycle()` | 调用方变更 |
| `ExecuteDecision()` | ❌ 无 | `Scheduler.runCycle()` | 新增 |
| `buildChaosContext()` | `AutoTrader.RunChaosCycle()` | ❌ 删除 | 移除 |
| `executeChaosDecision()` | `AutoTrader.RunChaosCycle()` | ❌ 删除 | 移除 |

---

## 5. 数据库影响分析

### 5.1 受影响的表

| 表名 | 影响 | 说明 |
|------|------|------|
| `decision_records` | 🟢 无 | 结构不变 |
| `trader_orders` | 🟢 无 | 结构不变 |
| `trader_positions` | 🟢 无 | 结构不变 |
| `equity_snapshots` | 🟢 无 | 结构不变 |

### 5.2 受影响的存储操作

**当前**：
```go
// auto_trader_chaos.go
func (at *AutoTrader) RunChaosCycle() {
    // ...
    at.store.SaveOrder(order)
    at.store.SaveDecision(record)
}
```

**重构后**：
```go
// trader/executor.go
func (de *DecisionExecutor) ExecuteDecision() {
    // ...
    de.store.SaveOrder(order)
}

// scheduler/auto_scheduler.go
func (s *AutoScheduler) runCycle() {
    // ...
    s.saveDecisionRecord(decision, result)
}
```

**影响**：
- 🟢 无数据库 schema 变更
- 🟡 保存逻辑位置变更

---

## 6. 配置影响分析

### 6.1 受影响的配置项

**当前配置**：
```yaml
trader:
  scan_interval: 3m
  exchanges:
    - binance
    - bybit
    
chaos:
  enabled: true
  risk_config:
    max_position_ratio: 0.2
```

**重构后配置**：
```yaml
scheduler:
  scan_interval: 3m
  engine: chaos  # 新增：指定引擎类型
  
engine:
  type: chaos
  risk_config:
    max_position_ratio: 0.2

trader:
  exchanges:
    - binance
    - bybit
```

**影响**：
- 🟡 配置结构调整
- 🟡 需要迁移脚本

---

## 7. 测试影响分析

### 7.1 需要更新的测试

| 测试文件 | 影响程度 | 修改内容 |
|----------|----------|----------|
| `chaos/*_test.go` | 🔴 高 | 更新所有测试用例 |
| `kernel/*_test.go` | 🔴 高 | 更新所有测试用例 |
| `trader/*_test.go` | 🔴 高 | 更新所有测试用例 |
| `manager/*_test.go` | 🟡 中 | 更新集成测试 |

### 7.2 需要新增的测试

| 测试文件 | 测试内容 |
|----------|----------|
| `engine/interface_test.go` | 接口一致性测试 |
| `scheduler/scheduler_test.go` | 调度器单元测试 |
| `trader/executor_test.go` | 执行器单元测试 |
| `integration/integration_test.go` | 端到端集成测试 |

---

## 8. 性能影响分析

### 8.1 性能基准

**当前性能**：
- 单次循环耗时：~2-3 秒
- 内存占用：~50MB
- Goroutine 数量：~10

**预期影响**：
- ⚠️ 可能增加 5-10% 的耗时（接口调用开销）
- ⚠️ 可能增加 5-10% 的内存（新对象创建）
- 🟢 Goroutine 数量基本不变

### 8.2 性能优化措施

1. **减少对象创建**
   ```go
   // 优化前
   func (s *Scheduler) runCycle() {
       context := &engine.Context{} // 每次创建
   }
   
   // 优化后
   func (s *Scheduler) runCycle() {
       context := s.pool.Get().(*engine.Context) // 对象池
       defer s.pool.Put(context)
   }
   ```

2. **批量处理**
   ```go
   // 优化前
   for _, d := range decisions {
       engine.ValidateDecisions(ctx, []Decision{d}, context)
   }
   
   // 优化后
   engine.ValidateDecisions(ctx, decisions, context) // 批量验证
   ```

---

## 9. 风险评估

### 9.1 高风险项

| 风险 | 影响 | 概率 | 缓解措施 |
|------|------|------|----------|
| 破坏现有功能 | 🔴 高 | 🟡 中 | 全面回归测试 |
| 交易所包修改遗漏 | 🔴 高 | 🟡 中 | 自动化检查脚本 |
| 数据不一致 | 🔴 高 | 🟢 低 | 保持数据库 schema 不变 |
| 性能回退 | 🟡 中 | 🟢 低 | 性能基准测试 |

### 9.2 中风险项

| 风险 | 影响 | 概率 | 缓解措施 |
|------|------|------|----------|
| 测试覆盖不足 | 🟡 中 | 🟡 中 | 强制覆盖率 > 80% |
| 文档更新延迟 | 🟡 中 | 🟡 中 | 文档作为验收标准 |
| 配置迁移错误 | 🟡 中 | 🟢 低 | 提供迁移脚本 |

### 9.3 低风险项

| 风险 | 影响 | 概率 | 缓解措施 |
|------|------|------|----------|
| 代码风格不一致 | 🟢 低 | 🟡 中 | Code Review |
| 注释不完整 | 🟢 低 | 🟡 中 | 强制注释规范 |

---

## 10. 迁移策略

### 10.1 阶段性迁移

**阶段 1**（当前）：
- ✅ 创建接口文件
- ✅ 保持现有代码不变
- ✅ 并行开发

**阶段 2**：
- 🔄 迁移 Chaos 上下文构建
- 🔄 迁移 Kernel 上下文构建
- 🔄 更新测试

**阶段 3**：
- ⏳ 迁移执行器到 Trader
- ⏳ 删除 chaos/executor.go
- ⏳ 更新测试

**阶段 4**：
- ⏳ 创建 Scheduler
- ⏳ 简化 AutoTrader
- ⏳ 更新测试

**阶段 5**：
- ⏳ 全面测试
- ⏳ 性能优化
- ⏳ 文档完善

### 10.2 回滚策略

如果重构失败，可以：

1. **Git 回滚**
   ```bash
   git revert <commit-hash>
   ```

2. **分支切换**
   ```bash
   git checkout main  # 切换到主分支
   ```

3. **配置回滚**
   ```yaml
   # 使用旧配置
   trader:
     scan_interval: 3m
   ```

---

## 11. 总结

### 11.1 修改文件统计

| 类型 | 数量 | 说明 |
|------|------|------|
| 新建文件 | 8 | 接口、类型、调度器等 |
| 修改文件 | 25+ | 引擎、Trader、交易所等 |
| 删除文件 | 1 | chaos/executor.go |
| 测试文件 | 15+ | 单元测试、集成测试 |

### 11.2 代码量统计

| 指标 | 估算 |
|------|------|
| 新增代码行数 | ~2000 行 |
| 修改代码行数 | ~3000 行 |
| 删除代码行数 | ~1000 行 |
| 净增代码行数 | ~4000 行 |

### 11.3 工作量估算

| 阶段 | 前端开发 | 后端开发 | 测试 | 总计 |
|------|----------|----------|------|------|
| 阶段 1 | 0 天 | 2 天 | 0 天 | 2 天 |
| 阶段 2 | 0 天 | 3 天 | 1 天 | 4 天 |
| 阶段 3 | 0 天 | 2 天 | 1 天 | 3 天 |
| 阶段 4 | 0 天 | 3 天 | 1 天 | 4 天 |
| 阶段 5 | 0 天 | 2 天 | 2 天 | 4 天 |
| **总计** | **0 天** | **12 天** | **5 天** | **17 天** |

### 11.4 建议

1. **优先保证功能完整性**
   - 每个阶段都要有完整测试
   - 不要为了进度牺牲质量

2. **渐进式重构**
   - 不要一次性修改所有代码
   - 小步快跑，及时验证

3. **充分沟通**
   - 每日站会同步进度
   - 遇到问题及时讨论

4. **文档先行**
   - 先写文档再写代码
   - 文档作为验收标准

---

## 附录：检查清单

### 阶段 1 检查清单

- [ ] 创建 `engine/interface.go`
- [ ] 创建 `engine/types.go`
- [ ] 创建 `trader/interface.go`（增强版）
- [ ] 创建 `trader/types.go`
- [ ] 创建 `scheduler/interface.go`
- [ ] 创建 `scheduler/types.go`
- [ ] 编写接口文档注释
- [ ] 完成影响分析文档
- [ ] 完成迁移计划文档
- [ ] 团队评审通过

### 阶段 2 检查清单

- [ ] 创建 `chaos/context_builder.go`
- [ ] 创建 `kernel/context_builder.go`
- [ ] 修改 `ChaosEngine.BuildContext()`
- [ ] 修改 `StrategyEngine.BuildContext()`
- [ ] 更新 `AutoTrader.RunChaosCycle()`
- [ ] 编写单元测试
- [ ] 编写集成测试
- [ ] 所有测试通过

### 阶段 3 检查清单

- [ ] 创建 `trader/executor.go`
- [ ] 迁移 `chaos/executor.go` 内容
- [ ] 删除 `chaos/executor.go`
- [ ] 更新 `AutoTrader.executeChaosDecision()`
- [ ] 编写单元测试
- [ ] 所有测试通过

### 阶段 4 检查清单

- [ ] 创建 `trader/scheduler.go`
- [ ] 迁移调度逻辑
- [ ] 简化 `AutoTrader`
- [ ] 编写调度器测试
- [ ] 所有测试通过

### 阶段 5 检查清单

- [ ] 创建 `engine/interface_test.go`
- [ ] 创建 `trader/interface_test.go`
- [ ] 创建 `scheduler/interface_test.go`
- [ ] 编写集成测试
- [ ] 性能基准测试
- [ ] 文档完善
- [ ] 团队评审通过
