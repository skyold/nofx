# 接口设计文档 - 阶段 1

## 1. Engine 接口设计

### 1.1 接口定义

```go
// engine/interface.go
package engine

import (
    "context"
)

// Engine 定义交易引擎的标准接口
type Engine interface {
    // Name 返回引擎名称（如 "chaos", "kernel"）
    Name() string
    
    // BuildContext 构建完整的交易上下文
    // 包括：账户信息、持仓、市场数据、候选币种等
    BuildContext(ctx context.Context, runtime RuntimeInfo) (*Context, error)
    
    // BuildSystemPrompt 构建系统提示词
    // 包含：输出格式、规则约束、角色定义等
    BuildSystemPrompt(ctx *Context) string
    
    // BuildUserPrompt 构建用户提示词
    // 包含：账户状态、持仓详情、市场数据等
    BuildUserPrompt(ctx *Context) string
    
    // CallLLM 调用 LLM 服务
    // 封装 MCP Client 的调用逻辑
    CallLLM(ctx context.Context, systemPrompt, userPrompt string) (*AIResponse, error)
    
    // ParseResponse 解析 LLM 响应
    // 从 AI 响应中提取交易决策
    ParseResponse(response *AIResponse) ([]Decision, error)
    
    // ValidateDecisions 验证交易决策
    // 根据风控规则验证决策的合法性
    ValidateDecisions(ctx context.Context, decisions []Decision, context *Context) ([]ValidatedDecision, error)
    
    // ProcessDecision 处理单个决策（可选）
    // 用于决策后处理，如记录日志、发送通知等
    ProcessDecision(ctx context.Context, decision *ValidatedDecision, context *Context) error
}
```

### 1.2 数据结构定义

```go
// engine/types.go
package engine

import (
    "time"
    "github.com/skyold/nofx/kernel"
    "github.com/skyold/nofx/market"
)

// RuntimeInfo 运行时信息
type RuntimeInfo struct {
    CurrentTime   time.Time `json:"current_time"`
    RuntimeMinutes int      `json:"runtime_minutes"` // 运行时长（分钟）
    CallCount     int       `json:"call_count"`      // 调用次数
}

// Context 交易上下文（通用结构）
type Context struct {
    // 基础信息
    CurrentTime   time.Time
    RuntimeMinutes int
    CallCount     int
    
    // 账户信息
    Account *kernel.AccountInfo `json:"account"`
    
    // 持仓信息
    Positions []kernel.PositionInfo `json:"positions"`
    
    // 候选币种
    CandidateCoins []kernel.CandidateCoin `json:"candidate_coins"`
    
    // 市场数据
    MarketDataMap map[string]*market.Data `json:"market_data_map"`
    
    // 量化数据
    QuantDataMap map[string]*kernel.QuantData `json:"quant_data_map"`
    
    // 排名数据
    OIRankingData       *kernel.OIRankingData       `json:"oi_ranking_data"`
    NetFlowRankingData  *kernel.NetFlowRankingData  `json:"netflow_ranking_data"`
    PriceRankingData    *kernel.PriceRankingData    `json:"price_ranking_data"`
    
    // 交易历史
    TradingStats *kernel.TradingStats   `json:"trading_stats"`
    RecentOrders []kernel.TraderOrder   `json:"recent_orders"`
    
    // 配置
    Config interface{} `json:"config"` // 引擎特定配置
}

// Decision 交易决策（通用结构）
type Decision struct {
    Symbol        string  `json:"symbol"`
    Action        string  `json:"action"` // open_long, open_short, close_long, close_short, hold, wait
    Leverage      int     `json:"leverage"`
    PositionSizeUSD float64 `json:"position_size_usd"`
    StopLoss      float64 `json:"stop_loss"`
    TakeProfit    float64 `json:"take_profit"`
    Confidence    int     `json:"confidence"` // 0-100
    Reasoning     string  `json:"reasoning"`
}

// ValidatedDecision 验证后的决策
type ValidatedDecision struct {
    Decision
    ValidatedPositionUSD float64 `json:"validated_position_usd"` // 经过风控验证的仓位
    ValidationErrors     []string `json:"validation_errors"`    // 验证错误（如果有）
    IsApproved           bool     `json:"is_approved"`          // 是否批准执行
}

// AIResponse LLM 响应
type AIResponse struct {
    RawResponse string                 `json:"raw_response"`
    Reasoning   string                 `json:"reasoning"`
    Decisions   map[string]interface{} `json:"decisions"`
    Metadata    map[string]string      `json:"metadata"`
}
```

### 1.3 合理性分析

#### ✅ 设计原则遵循

1. **单一职责原则**
   - `BuildContext()` 只负责数据收集
   - `BuildSystemPrompt()` 和 `BuildUserPrompt()` 只负责提示词构建
   - `CallLLM()` 只负责与 LLM 通信
   - `ParseResponse()` 只负责解析
   - `ValidateDecisions()` 只负责验证
   - **每个方法职责清晰，无重叠**

2. **开闭原则**
   - 接口稳定，具体实现可变
   - 新增引擎类型（如强化学习）只需实现接口
   - **对扩展开放，对修改关闭**

3. **依赖倒置原则**
   - 高层模块（调度器）依赖抽象（Engine 接口）
   - 低层模块（ChaosEngine, StrategyEngine）实现接口
   - **解耦调度器和具体引擎**

4. **接口隔离原则**
   - 接口方法数量适中（7 个核心方法）
   - 每个方法都有明确的用途
   - **无冗余方法**

#### ✅ 命名规范

- **动词 + 名词** 结构：`BuildContext`, `BuildSystemPrompt`
- **清晰表达意图**：`ValidateDecisions` 比 `CheckDecisions` 更准确
- **一致性**：所有方法都使用现在时态

#### ✅ 参数设计

```go
// 好：使用 context 支持取消和超时
BuildContext(ctx context.Context, runtime RuntimeInfo) (*Context, error)

// 好：上下文作为指针传递，避免拷贝
BuildSystemPrompt(ctx *Context) string

// 好：批量验证，提高效率
ValidateDecisions(ctx context.Context, decisions []Decision, context *Context) ([]ValidatedDecision, error)
```

**理由**：
- `context.Context` 支持超时控制和取消传播
- `*Context` 指针传递避免大对象拷贝
- 批量验证减少重复计算

#### ✅ 错误处理

```go
type ContextError struct {
    Engine  string
    Method  string
    Err     error
    Details map[string]interface{}
}

func (e *ContextError) Error() string {
    return fmt.Sprintf("engine %s method %s failed: %v", e.Engine, e.Method, e.Err)
}
```

**优势**：
- 结构化错误信息
- 便于调试和监控
- 支持错误分类处理

---

### 1.4 与当前实现的差异

#### Chaos Engine 对比

| 方法 | 当前实现 | 新接口 | 差异分析 |
|------|----------|--------|----------|
| BuildContext | ❌ 无 | ✅ `ChaosEngine.BuildContext()` | **新增**：当前在 `AutoTrader.buildChaosContext()` |
| BuildSystemPrompt | ✅ `ChaosEngine.BuildSystemPromptWithContext(ctx)` | ✅ `BuildSystemPrompt(ctx *Context)` | **改进**：移除 `WithContext` 后缀，更简洁 |
| BuildUserPrompt | ✅ `ChaosEngine.BuildUserPrompt(ctx)` | ✅ `BuildUserPrompt(ctx *Context)` | **一致**：无需修改 |
| CallLLM | ❌ 无（在 AutoTrader 中） | ✅ `ChaosEngine.CallLLM()` | **新增**：从 `AutoTrader` 迁移 |
| ParseResponse | ✅ `ChaosEngine.ExtractDecisions()` | ✅ `ParseResponse()` | **改进**：统一命名，增加 `ExtractReasoning()` |
| ValidateDecisions | ✅ `ChaosEngine.ValidateDecisions()` | ✅ `ValidateDecisions()` | **一致**：无需修改 |
| ProcessDecision | ❌ 无 | ✅ `ProcessDecision()` | **新增**：用于决策后处理 |

**关键差异**：

1. **BuildContext 缺失**
   ```go
   // 当前：在 AutoTrader 中
   func (at *AutoTrader) buildChaosContext() (*ChaosContext, error) {
       // 问题：调度器在构建上下文
   }
   
   // 新：在 ChaosEngine 中
   func (ce *ChaosEngine) BuildContext(ctx context.Context, runtime RuntimeInfo) (*Context, error) {
       // 改进：引擎负责完整的上下文构建
   }
   ```

2. **CallLLM 缺失**
   ```go
   // 当前：在 AutoTrader 中
   aiResponse, err := at.mcpClient.CallWithMessages(systemPrompt, userPrompt)
   
   // 新：在 ChaosEngine 中
   func (ce *ChaosEngine) CallLLM(ctx context.Context, systemPrompt, userPrompt string) (*AIResponse, error) {
       // 改进：引擎封装 LLM 调用
   }
   ```

3. **方法命名不统一**
   ```go
   // 当前
   ExtractDecisions()
   ExtractReasoning()
   ExtractCoTTrace()
   
   // 新
   ParseResponse() // 统一解析所有信息
   ```

#### Kernel Engine 对比

| 方法 | 当前实现 | 新接口 | 差异分析 |
|------|----------|--------|----------|
| BuildContext | ❌ 无 | ✅ `StrategyEngine.BuildContext()` | **新增**：当前数据收集分散在多个方法 |
| BuildSystemPrompt | ✅ `StrategyEngine.BuildSystemPrompt()` | ✅ `BuildSystemPrompt(ctx *Context)` | **改进**：参数统一为 `*Context` |
| BuildUserPrompt | ✅ `StrategyEngine.BuildUserPrompt()` | ✅ `BuildUserPrompt(ctx *Context)` | **改进**：参数统一为 `*Context` |
| CallLLM | ❌ 无 | ✅ `StrategyEngine.CallLLM()` | **新增**：封装 LLM 调用 |
| ParseResponse | ✅ 解析逻辑在 `RunChaosCycle()` | ✅ `ParseResponse()` | **新增**：封装解析逻辑 |
| ValidateDecisions | ✅ `ValidateDecision()` (单个) | ✅ `ValidateDecisions()` (批量) | **改进**：支持批量验证 |

**关键差异**：

1. **缺少统一的上下文结构**
   ```go
   // 当前：Kernel 的 Context 在 engine.go 中定义
   type Context struct {
       CurrentTime   time.Time
       Account       AccountInfo
       Positions     []PositionInfo
       // ... 但缺少统一构建方法
   }
   
   // 新：统一的 Context 结构
   type Context struct {
       // 标准化字段
       Account *kernel.AccountInfo
       Positions []kernel.PositionInfo
       // ... 由 BuildContext() 统一构建
   }
   ```

2. **验证方法粒度**
   ```go
   // 当前：单个验证
   func (m *Manager) ValidateDecision(d *Decision, ...) (float64, error)
   
   // 新：批量验证
   func (se *StrategyEngine) ValidateDecisions(ctx context.Context, decisions []Decision, context *Context) ([]ValidatedDecision, error)
   ```

---

## 2. Trader 接口设计

### 2.1 接口定义

```go
// trader/interface.go
package trader

import (
    "context"
    "github.com/skyold/nofx/engine"
)

// Trader 定义交易执行的标准接口
type Trader interface {
    // GetExchange 返回交易所名称
    GetExchange() string
    
    // === 数据查询接口 ===
    
    // GetAccountInfo 获取账户信息
    GetAccountInfo(ctx context.Context) (*AccountInfo, error)
    
    // GetPositions 获取持仓信息
    GetPositions(ctx context.Context) ([]PositionInfo, error)
    
    // GetMarketPrice 获取市场价格
    GetMarketPrice(ctx context.Context, symbol string) (float64, error)
    
    // GetBalance 获取余额信息
    GetBalance(ctx context.Context, asset string) (*BalanceInfo, error)
    
    // === 交易执行接口 ===
    
    // OpenLong 开多仓
    OpenLong(ctx context.Context, symbol string, quantity float64, leverage int) (*Order, error)
    
    // OpenShort 开空仓
    OpenShort(ctx context.Context, symbol string, quantity float64, leverage int) (*Order, error)
    
    // CloseLong 平多仓
    CloseLong(ctx context.Context, symbol string, quantity float64) (*Order, error)
    
    // CloseShort 平空仓
    CloseShort(ctx context.Context, symbol string, quantity float64) (*Order, error)
    
    // === 风控接口 ===
    
    // SetLeverage 设置杠杆
    SetLeverage(ctx context.Context, symbol string, leverage int) error
    
    // SetStopLoss 设置止损
    SetStopLoss(ctx context.Context, orderID string, price float64) error
    
    // SetTakeProfit 设置止盈
    SetTakeProfit(ctx context.Context, orderID string, price float64) error
    
    // === 决策执行接口（新增） ===
    
    // ExecuteDecision 执行交易决策
    // 封装完整的交易流程（开仓/平仓 + 止损止盈）
    ExecuteDecision(ctx context.Context, decision *engine.ValidatedDecision) (*OrderResult, error)
}
```

### 2.2 数据结构定义

```go
// trader/types.go
package trader

import (
    "time"
)

// AccountInfo 账户信息
type AccountInfo struct {
    TotalEquity     float64 `json:"total_equity"`
    AvailableBalance float64 `json:"available_balance"`
    TotalPnL        float64 `json:"total_pnl"`
    MarginUsed      float64 `json:"margin_used"`
    MarginRatio     float64 `json:"margin_ratio"`
}

// PositionInfo 持仓信息
type PositionInfo struct {
    Symbol        string    `json:"symbol"`
    Side          string    `json:"side"` // long, short
    Quantity      float64   `json:"quantity"`
    EntryPrice    float64   `json:"entry_price"`
    MarkPrice     float64   `json:"mark_price"`
    Leverage      int       `json:"leverage"`
    UnrealizedPnL float64   `json:"unrealized_pnl"`
    LiquidationPrice float64 `json:"liquidation_price"`
    MarginMode    string    `json:"margin_mode"` // isolated, cross
}

// Order 订单信息
type Order struct {
    ID             string    `json:"id"`
    Symbol         string    `json:"symbol"`
    Side           string    `json:"side"`
    Type           string    `json:"type"` // market, limit
    Quantity       float64   `json:"quantity"`
    Price          float64   `json:"price"`
    AvgFillPrice   float64   `json:"avg_fill_price"`
    Status         string    `json:"status"` // filled, cancelled, rejected
    CreatedAt      time.Time `json:"created_at"`
    FilledAt       time.Time `json:"filled_at"`
}

// OrderResult 订单执行结果
type OrderResult struct {
    Order       *Order   `json:"order"`
    StopLossID  string   `json:"stop_loss_id"`
    TakeProfitID string  `json:"take_profit_id"`
    Success     bool     `json:"success"`
    Error       error    `json:"error"`
    Message     string   `json:"message"`
}

// BalanceInfo 余额信息
type BalanceInfo struct {
    Asset      string  `json:"asset"`
    Free       float64 `json:"free"`
    Locked     float64 `json:"locked"`
    Total      float64 `json:"total"`
}
```

### 2.3 合理性分析

#### ✅ 接口分层设计

```go
// 第一层：数据查询（只读）
GetAccountInfo()
GetPositions()
GetMarketPrice()
GetBalance()

// 第二层：基础交易操作（写操作）
OpenLong/OpenShort()
CloseLong/CloseShort()
SetLeverage()
SetStopLoss/SetTakeProfit()

// 第三层：高级抽象（组合操作）
ExecuteDecision() // 封装完整流程
```

**优势**：
- **灵活性**：支持细粒度操作和粗粒度操作
- **便利性**：`ExecuteDecision()` 简化常用场景
- **可扩展性**：新增操作可在对应层级添加

#### ✅ Context 参数设计

```go
// 所有方法都接收 context.Context
GetAccountInfo(ctx context.Context) (*AccountInfo, error)
OpenLong(ctx context.Context, symbol string, quantity float64, leverage int) (*Order, error)
```

**理由**：
- 支持超时控制（防止 API 调用卡住）
- 支持取消操作（紧急情况下取消交易）
- 支持请求追踪（便于调试和监控）

#### ✅ ExecuteDecision 设计

```go
// ExecuteDecision 封装完整流程
func (t *BaseTrader) ExecuteDecision(ctx context.Context, decision *engine.ValidatedDecision) (*OrderResult, error) {
    switch decision.Action {
    case "open_long":
        return t.executeOpen(ctx, decision, "long")
    case "open_short":
        return t.executeOpen(ctx, decision, "short")
    case "close_long":
        return t.executeClose(ctx, decision, "long")
    case "close_short":
        return t.executeClose(ctx, decision, "short")
    }
}

func (t *BaseTrader) executeOpen(ctx context.Context, decision *engine.ValidatedDecision, side string) (*OrderResult, error) {
    // 1. 获取市场价格
    price, err := t.GetMarketPrice(ctx, decision.Symbol)
    
    // 2. 计算数量
    quantity := decision.ValidatedPositionUSD / price
    
    // 3. 设置杠杆
    if err := t.SetLeverage(ctx, decision.Symbol, decision.Leverage); err != nil {
        return nil, err
    }
    
    // 4. 开仓
    var order *Order
    if side == "long" {
        order, err = t.OpenLong(ctx, decision.Symbol, quantity, decision.Leverage)
    } else {
        order, err = t.OpenShort(ctx, decision.Symbol, quantity, decision.Leverage)
    }
    if err != nil {
        return nil, err
    }
    
    // 5. 设置止损止盈
    var stopLossID, takeProfitID string
    if decision.StopLoss > 0 {
        if err := t.SetStopLoss(ctx, order.ID, decision.StopLoss); err == nil {
            stopLossID = order.ID + "_sl"
        }
    }
    if decision.TakeProfit > 0 {
        if err := t.SetTakeProfit(ctx, order.ID, decision.TakeProfit); err == nil {
            takeProfitID = order.ID + "_tp"
        }
    }
    
    return &OrderResult{
        Order:        order,
        StopLossID:   stopLossID,
        TakeProfitID: takeProfitID,
        Success:      true,
    }, nil
}
```

**优势**：
- **封装复杂性**：调用者无需关心内部流程
- **原子性**：要么全部成功，要么回滚
- **可复用**：所有交易所共享逻辑

---

### 2.4 与当前实现的差异

#### 当前 Trader 接口对比

| 方法 | 当前实现 | 新接口 | 差异分析 |
|------|----------|--------|----------|
| GetExchange | ✅ `GetExchangeName()` | ✅ `GetExchange()` | **简化**：移除冗余后缀 |
| GetAccountInfo | ✅ `GetBalance()` | ✅ `GetAccountInfo()` | **改进**：命名更准确，返回更完整信息 |
| GetPositions | ✅ `GetPositions()` | ✅ `GetPositions()` | **一致**：无需修改 |
| GetMarketPrice | ✅ `GetMarkPrice()` | ✅ `GetMarketPrice()` | **改进**：修正命名（Mark → Market） |
| OpenLong | ✅ `OpenLong()` | ✅ `OpenLong(ctx, symbol, quantity, leverage)` | **改进**：增加 context 参数 |
| OpenShort | ✅ `OpenShort()` | ✅ `OpenShort(ctx, symbol, quantity, leverage)` | **改进**：增加 context 参数 |
| CloseLong | ✅ `CloseLong()` | ✅ `CloseLong(ctx, symbol, quantity)` | **改进**：增加 context 参数 |
| CloseShort | ✅ `CloseShort()` | ✅ `CloseShort(ctx, symbol, quantity)` | **改进**：增加 context 参数 |
| SetLeverage | ✅ `SetLeverage()` | ✅ `SetLeverage(ctx, symbol, leverage)` | **改进**：增加 context 参数 |
| SetStopLoss | ✅ `SetStopLoss()` | ✅ `SetStopLoss(ctx, orderID, price)` | **改进**：增加 context 参数 |
| SetTakeProfit | ✅ `SetTakeProfit()` | ✅ `SetTakeProfit(ctx, orderID, price)` | **改进**：增加 context 参数 |
| ExecuteDecision | ❌ 无 | ✅ `ExecuteDecision()` | **新增**：封装完整流程 |

**关键差异**：

1. **缺少 ExecuteDecision 方法**
   ```go
   // 当前：在 AutoTrader 中
   func (at *AutoTrader) executeChaosDecision(decision *chaos.Decision, ctx *ChaosContext) {
       // 问题：调度器在执行决策
       // 1. 获取价格
       // 2. 计算数量
       // 3. 调用 trader.OpenLong()
       // 4. 设置止损止盈
   }
   
   // 新：在 Trader 中
   func (t *BaseTrader) ExecuteDecision(ctx context.Context, decision *engine.ValidatedDecision) (*OrderResult, error) {
       // 改进：Trader 负责执行决策
       // 封装完整流程，返回执行结果
   }
   ```

2. **GetBalance 语义不清晰**
   ```go
   // 当前
   func (t *FuturesTrader) GetBalance() (*Balance, error)
   // 问题：返回的是账户信息，不只是余额
   
   // 新
   func (t *BaseTrader) GetAccountInfo(ctx context.Context) (*AccountInfo, error)
   // 改进：命名准确，返回完整账户信息
   ```

3. **缺少 context 支持**
   ```go
   // 当前
   func (t *FuturesTrader) OpenLong(symbol string, quantity float64) (*Order, error)
   // 问题：无法超时控制，无法取消
   
   // 新
   func (t *BaseTrader) OpenLong(ctx context.Context, symbol string, quantity float64, leverage int) (*Order, error)
   // 改进：支持 context，可控制超时和取消
   ```

---

## 3. Scheduler 接口设计

### 3.1 接口定义

```go
// scheduler/interface.go
package scheduler

import (
    "context"
    "time"
    "github.com/skyold/nofx/engine"
    "github.com/skyold/nofx/trader"
)

// Scheduler 定义调度器的标准接口
type Scheduler interface {
    // === 生命周期管理 ===
    
    // Start 启动调度器
    Start() error
    
    // Stop 停止调度器
    Stop() error
    
    // IsRunning 检查是否正在运行
    IsRunning() bool
    
    // === 组件管理 ===
    
    // SetEngine 设置交易引擎
    SetEngine(engine engine.Engine)
    
    // SetTrader 设置交易执行器
    SetTrader(trader trader.Trader)
    
    // SetExecutor 设置决策执行器（可选）
    SetExecutor(executor DecisionExecutor)
    
    // === 配置管理 ===
    
    // SetInterval 设置调度间隔
    SetInterval(interval time.Duration)
    
    // GetInterval 获取调度间隔
    GetInterval() time.Duration
    
    // === 状态查询 ===
    
    // GetStatus 获取调度器状态
    GetStatus() *SchedulerStatus
    
    // GetStats 获取统计信息
    GetStats() *SchedulerStats
}

// DecisionExecutor 决策执行器接口（可选，用于复杂场景）
type DecisionExecutor interface {
    Execute(ctx context.Context, decision *engine.ValidatedDecision) (*trader.OrderResult, error)
}
```

### 3.2 数据结构定义

```go
// scheduler/types.go
package scheduler

import (
    "time"
    "sync/atomic"
)

// SchedulerStatus 调度器状态
type SchedulerStatus struct {
    IsRunning   bool      `json:"is_running"`
    StartedAt   time.Time `json:"started_at"`
    StoppedAt   time.Time `json:"stopped_at"`
    CurrentCycle int64    `json:"current_cycle"` // 当前周期数
    LastCycleAt time.Time `json:"last_cycle_at"` // 上次执行时间
    NextCycleAt time.Time `json:"next_cycle_at"` // 下次执行时间
}

// SchedulerStats 调度器统计
type SchedulerStats struct {
    TotalCycles      int64         `json:"total_cycles"`      // 总周期数
    SuccessfulCycles int64         `json:"successful_cycles"` // 成功周期数
    FailedCycles     int64         `json:"failed_cycles"`     // 失败周期数
    TotalDecisions   int64         `json:"total_decisions"`   // 总决策数
    ExecutedDecisions int64        `json:"executed_decisions"` // 已执行决策数
    AvgCycleDuration time.Duration `json:"avg_cycle_duration"` // 平均周期耗时
    LastCycleDuration time.Duration `json:"last_cycle_duration"` // 上次周期耗时
    LastError        error         `json:"last_error"`        // 上次错误
}

// CycleResult 周期执行结果
type CycleResult struct {
    CycleID       int64                    `json:"cycle_id"`
    StartTime     time.Time                `json:"start_time"`
    EndTime       time.Time                `json:"end_time"`
    Duration      time.Duration            `json:"duration"`
    Decisions     []engine.ValidatedDecision `json:"decisions"`
    Results       []*trader.OrderResult    `json:"results"`
    Error         error                    `json:"error"`
}
```

### 3.3 实现示例

```go
// scheduler/auto_scheduler.go
package scheduler

import (
    "context"
    "log"
    "sync"
    "time"
    "github.com/skyold/nofx/engine"
    "github.com/skyold/nofx/trader"
)

type AutoScheduler struct {
    mu       sync.RWMutex
    engine   engine.Engine
    trader   trader.Trader
    executor DecisionExecutor
    
    interval time.Duration
    isRunning atomic.Bool
    stopChan  chan struct{}
    
    status *SchedulerStatus
    stats  *SchedulerStats
}

func NewAutoScheduler(interval time.Duration) *AutoScheduler {
    return &AutoScheduler{
        interval: interval,
        stopChan: make(chan struct{}),
        status:   &SchedulerStatus{},
        stats:    &SchedulerStats{},
    }
}

func (s *AutoScheduler) Start() error {
    s.mu.Lock()
    defer s.mu.Unlock()
    
    if s.isRunning.Load() {
        return ErrAlreadyRunning
    }
    
    if s.engine == nil {
        return ErrEngineNotSet
    }
    
    if s.trader == nil {
        return ErrTraderNotSet
    }
    
    s.isRunning.Store(true)
    s.status.StartedAt = time.Now()
    s.status.IsRunning = true
    
    go s.runLoop()
    
    return nil
}

func (s *AutoScheduler) Stop() error {
    s.mu.Lock()
    defer s.mu.Unlock()
    
    if !s.isRunning.Load() {
        return ErrNotRunning
    }
    
    s.isRunning.Store(false)
    s.status.StoppedAt = time.Now()
    s.status.IsRunning = false
    
    close(s.stopChan)
    
    return nil
}

func (s *AutoScheduler) runLoop() {
    ticker := time.NewTicker(s.interval)
    defer ticker.Stop()
    
    for {
        select {
        case <-ticker.C:
            s.runCycle()
        case <-s.stopChan:
            return
        }
    }
}

func (s *AutoScheduler) runCycle() {
    ctx := context.Background()
    cycleStart := time.Now()
    
    // 更新状态
    s.mu.Lock()
    s.status.CurrentCycle++
    s.status.LastCycleAt = cycleStart
    s.status.NextCycleAt = cycleStart.Add(s.interval)
    s.mu.Unlock()
    
    // 1. 构建上下文
    runtime := engine.RuntimeInfo{
        CurrentTime:   cycleStart,
        RuntimeMinutes: int(time.Since(s.status.StartedAt).Minutes()),
        CallCount:     int(s.status.CurrentCycle),
    }
    
    context, err := s.engine.BuildContext(ctx, runtime)
    if err != nil {
        s.recordCycleError(err)
        log.Printf("[Scheduler] BuildContext failed: %v", err)
        return
    }
    
    // 2. 构建提示词
    systemPrompt := s.engine.BuildSystemPrompt(context)
    userPrompt := s.engine.BuildUserPrompt(context)
    
    // 3. 调用 LLM
    aiResponse, err := s.engine.CallLLM(ctx, systemPrompt, userPrompt)
    if err != nil {
        s.recordCycleError(err)
        log.Printf("[Scheduler] CallLLM failed: %v", err)
        return
    }
    
    // 4. 解析响应
    decisions, err := s.engine.ParseResponse(aiResponse)
    if err != nil {
        s.recordCycleError(err)
        log.Printf("[Scheduler] ParseResponse failed: %v", err)
        return
    }
    
    // 5. 验证决策
    validatedDecisions, err := s.engine.ValidateDecisions(ctx, decisions, context)
    if err != nil {
        s.recordCycleError(err)
        log.Printf("[Scheduler] ValidateDecisions failed: %v", err)
        return
    }
    
    // 6. 执行决策
    var results []*trader.OrderResult
    for _, decision := range validatedDecisions {
        if !decision.IsApproved {
            log.Printf("[Scheduler] Decision rejected: %v", decision.ValidationErrors)
            continue
        }
        
        var result *trader.OrderResult
        if s.executor != nil {
            result, err = s.executor.Execute(ctx, &decision)
        } else {
            result, err = s.trader.ExecuteDecision(ctx, &decision)
        }
        
        if err != nil {
            log.Printf("[Scheduler] ExecuteDecision failed: %v", err)
        } else {
            results = append(results, result)
        }
    }
    
    // 7. 记录统计
    s.recordCycleSuccess(len(validatedDecisions), len(results), time.Since(cycleStart))
}

func (s *AutoScheduler) recordCycleError(err error) {
    s.mu.Lock()
    defer s.mu.Unlock()
    
    s.stats.FailedCycles++
    s.stats.LastError = err
}

func (s *AutoScheduler) recordCycleSuccess(decisionCount, executedCount int, duration time.Duration) {
    s.mu.Lock()
    defer s.mu.Unlock()
    
    s.stats.TotalCycles++
    s.stats.SuccessfulCycles++
    s.stats.TotalDecisions += int64(decisionCount)
    s.stats.ExecutedDecisions += int64(executedCount)
    s.stats.LastCycleDuration = duration
    s.stats.AvgCycleDuration = time.Duration(
        int64(s.stats.AvgCycleDuration)*(s.stats.TotalCycles-1) + int64(duration),
    ) / s.stats.TotalCycles
}
```

### 3.4 合理性分析

#### ✅ 职责清晰

```go
// Scheduler 只负责编排流程
func (s *AutoScheduler) runCycle() {
    // 1. 调用引擎构建上下文
    context, _ := s.engine.BuildContext()
    
    // 2. 调用引擎构建提示词
    systemPrompt := s.engine.BuildSystemPrompt(context)
    
    // 3. 调用引擎调用 LLM
    aiResponse, _ := s.engine.CallLLM()
    
    // 4. 调用引擎解析响应
    decisions, _ := s.engine.ParseResponse(aiResponse)
    
    // 5. 调用引擎验证决策
    validated, _ := s.engine.ValidateDecisions(decisions)
    
    // 6. 调用 Trader 执行决策
    s.trader.ExecuteDecision(validated)
    
    // 注意：Scheduler 不包含任何业务逻辑
}
```

#### ✅ 依赖注入

```go
// 通过接口依赖，易于测试和替换
func (s *AutoScheduler) SetEngine(engine engine.Engine) {
    s.engine = engine
}

func (s *AutoScheduler) SetTrader(trader trader.Trader) {
    s.trader = trader
}
```

#### ✅ 状态管理

```go
// 提供完整的状态查询
type SchedulerStatus struct {
    IsRunning   bool
    StartedAt   time.Time
    CurrentCycle int64
    // ...
}

type SchedulerStats struct {
    TotalCycles      int64
    SuccessfulCycles int64
    AvgCycleDuration time.Duration
    // ...
}
```

---

### 3.5 与当前实现的差异

#### 当前 AutoTrader 对比

| 功能 | 当前实现 | 新接口 | 差异分析 |
|------|----------|--------|----------|
| Start | ✅ `AutoTrader.Run()` | ✅ `Scheduler.Start()` | **改进**：拆分为 Start/Stop，更清晰 |
| Stop | ✅ `AutoTrader.Stop()` | ✅ `Scheduler.Stop()` | **一致**：无需修改 |
| SetEngine | ❌ 无 | ✅ `SetEngine()` | **新增**：支持动态设置引擎 |
| SetTrader | ❌ 无 | ✅ `SetTrader()` | **新增**：支持动态设置 Trader |
| SetInterval | ❌ 硬编码 | ✅ `SetInterval()` | **改进**：支持动态调整间隔 |
| GetStatus | ❌ 无 | ✅ `GetStatus()` | **新增**：提供状态查询 |
| GetStats | ❌ 无 | ✅ `GetStats()` | **新增**：提供统计信息 |
| runCycle | ✅ `RunChaosCycle()` | ✅ `runCycle()` | **简化**：移除业务逻辑，只保留编排 |

**关键差异**：

1. **Run 方法职责过重**
   ```go
   // 当前：AutoTrader.Run() 包含太多逻辑
   func (at *AutoTrader) Run() {
       // 1. 启动订单同步 goroutine
       // 2. 创建 ticker
       // 3. 主循环：
       //    - 构建上下文（业务逻辑）
       //    - 构建提示词（业务逻辑）
       //    - 调用 LLM（业务逻辑）
       //    - 解析响应（业务逻辑）
       //    - 验证决策（业务逻辑）
       //    - 执行决策（业务逻辑）
   }
   
   // 新：Scheduler.Run() 只负责编排
   func (s *Scheduler) Run() {
       // 1. 启动订单同步 goroutine
       // 2. 创建 ticker
       // 3. 主循环：
       //    - 调用 engine.BuildContext()
       //    - 调用 engine.BuildSystemPrompt()
       //    - 调用 engine.CallLLM()
       //    - 调用 engine.ParseResponse()
       //    - 调用 engine.ValidateDecisions()
       //    - 调用 trader.ExecuteDecision()
       // 注意：只调用接口方法，不包含业务逻辑
   }
   ```

2. **缺少状态管理**
   ```go
   // 当前：无状态查询
   // 问题：无法知道调度器运行状态
   
   // 新：提供完整状态
   func (s *Scheduler) GetStatus() *SchedulerStatus {
       return s.status
   }
   
   func (s *Scheduler) GetStats() *SchedulerStats {
       return s.stats
   }
   ```

3. **缺少依赖注入**
   ```go
   // 当前：引擎和 Trader 在构造时固定
   // 问题：难以测试和替换
   
   // 新：支持动态设置
   func (s *Scheduler) SetEngine(engine engine.Engine)
   func (s *Scheduler) SetTrader(trader trader.Trader)
   ```

---

## 4. 接口依赖关系图

```
┌─────────────────────────────────────────────────────────┐
│                    Scheduler                            │
│  - depends on: Engine interface                         │
│  - depends on: Trader interface                         │
│  - depends on: DecisionExecutor interface (optional)    │
└────────────────────┬────────────────────┬───────────────┘
                     │                    │
                     │ calls              │ calls
                     ↓                    ↓
┌──────────────────────────────┐  ┌──────────────────────┐
│         Engine               │  │       Trader         │
│  - BuildContext()            │  │  - GetAccountInfo()  │
│  - BuildSystemPrompt()       │  │  - GetPositions()    │
│  - BuildUserPrompt()         │  │  - OpenLong()        │
│  - CallLLM()                 │  │  - CloseLong()       │
│  - ParseResponse()           │  │  - ExecuteDecision() │
│  - ValidateDecisions()       │  │                      │
└──────────────┬───────────────┘  └──────────┬───────────┘
               │                             │
               │ uses                        │ uses
               ↓                             ↓
┌──────────────────────────────┐  ┌──────────────────────┐
│      MCP Client              │  │   Exchange API       │
│  - CallWithMessages()        │  │  - CreateOrder()     │
│  - Stream()                  │  │  - CancelOrder()     │
└──────────────────────────────┘  │  - GetBalance()      │
                                  └──────────────────────┘
```

**依赖规则**：
- ✅ Scheduler → Engine (接口依赖)
- ✅ Scheduler → Trader (接口依赖)
- ✅ Engine → MCP Client (实现依赖)
- ✅ Trader → Exchange API (实现依赖)
- ❌ Engine → Trader (不允许)
- ❌ Trader → Engine (不允许)
- ❌ Scheduler → MCP Client (不允许)
- ❌ Scheduler → Exchange API (不允许)

---

## 5. 接口设计总结

### 5.1 设计原则验证

| 原则 | 验证结果 | 说明 |
|------|----------|------|
| 单一职责 | ✅ | 每个接口职责清晰 |
| 开闭原则 | ✅ | 对扩展开放，对修改关闭 |
| 里氏替换 | ✅ | 实现类可互换 |
| 依赖倒置 | ✅ | 依赖抽象而非具体实现 |
| 接口隔离 | ✅ | 接口粒度适中 |

### 5.2 与当前实现的核心差异

| 方面 | 当前 | 新设计 | 改进点 |
|------|------|--------|--------|
| 上下文构建 | AutoTrader | Engine | 职责回归 |
| LLM 调用 | AutoTrader | Engine | 职责回归 |
| 决策执行 | AutoTrader | Trader | 职责回归 |
| 调度逻辑 | 包含业务逻辑 | 纯编排 | 职责单一 |
| Context 支持 | 无 | 全面支持 | 可超时控制 |
| 错误处理 | 不统一 | 结构化 | 易于调试 |
| 状态管理 | 无 | 完整 | 可观测性 |

### 5.3 需要修改的文件清单

#### 阶段 1（接口设计）需要创建的文件：

1. `engine/interface.go` - 新建
2. `engine/types.go` - 新建（从现有类型迁移）
3. `trader/interface.go` - 增强（添加 ExecuteDecision）
4. `trader/types.go` - 新建（统一类型定义）
5. `scheduler/interface.go` - 新建
6. `scheduler/types.go` - 新建

#### 后续阶段需要修改的文件：

**阶段 2**：
- `chaos/context_builder.go` - 新建
- `kernel/context_builder.go` - 新建
- `chaos/engine.go` - 增强
- `kernel/engine.go` - 增强
- `trader/auto_trader_chaos.go` - 简化

**阶段 3**：
- `trader/executor.go` - 新建
- `chaos/executor.go` - 删除
- `trader/auto_trader_chaos.go` - 简化

**阶段 4**：
- `scheduler/auto_scheduler.go` - 新建
- `trader/auto_trader.go` - 简化

**阶段 5**：
- 全面测试和优化

---

## 6. 下一步行动

### 阶段 1 任务清单

- [ ] 创建 `engine/interface.go`
- [ ] 创建 `engine/types.go`
- [ ] 创建 `trader/interface.go`（增强版）
- [ ] 创建 `trader/types.go`
- [ ] 创建 `scheduler/interface.go`
- [ ] 创建 `scheduler/types.go`
- [ ] 编写接口文档注释
- [ ] 评审接口设计
- [ ] 创建影响分析文档
- [ ] 创建迁移计划文档

### 验收标准

- [ ] 所有接口文件创建完成
- [ ] 接口注释完整
- [ ] 通过代码审查
- [ ] 影响分析文档完成
- [ ] 迁移计划文档完成
- [ ] 团队评审通过
