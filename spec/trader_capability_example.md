# Trader 能力设计实现示例

本文档展示如何实现新的 Trader 能力查询接口。

## 1. Binance 实现示例

```go
// trader/binance/futures.go
package binance

import (
    "context"
    "github.com/skyold/nofx/trader"
)

type FuturesTrader struct {
    // ... 现有字段
}

// GetCapabilities 返回 Binance 支持的所有 action
func (t *FuturesTrader) GetCapabilities() []trader.Action {
    return []trader.Action{
        // 基础交易
        trader.ActionOpenLong,
        trader.ActionOpenShort,
        trader.ActionCloseLong,
        trader.ActionCloseShort,
        
        // 风控
        trader.ActionSetLeverage,
        trader.ActionSetStopLoss,
        trader.ActionSetTakeProfit,
        
        // 订单查询
        trader.ActionOrderStatusQuery,
        trader.ActionOpenOrdersQuery,
        
        // 注意：Binance Futures 不支持现货
        // trader.ActionBuySpot, trader.ActionSellSpot 不在列表中
    }
}

// SupportsAction 检查是否支持某个 action
func (t *FuturesTrader) SupportsAction(action trader.Action) bool {
    capabilities := t.GetCapabilities()
    for _, cap := range capabilities {
        if cap == action {
            return true
        }
    }
    return false
}

// SetStopLoss 设置止损（带能力检查）
func (t *FuturesTrader) SetStopLoss(ctx context.Context, orderID string, price float64) error {
    // 检查是否支持
    if !t.SupportsAction(trader.ActionSetStopLoss) {
        return trader.ErrCapabilityNotSupported
    }
    
    // 正常实现
    // ...
    return nil
}

// GetOrderStatus 查询订单状态
func (t *FuturesTrader) GetOrderStatus(ctx context.Context, symbol, orderID string) (*trader.OrderStatus, error) {
    // 检查是否支持
    if !t.SupportsAction(trader.ActionOrderStatusQuery) {
        return nil, trader.ErrCapabilityNotSupported
    }
    
    // 正常实现
    order, err := t.client.GetOrder()
    if err != nil {
        return nil, err
    }
    
    return &trader.OrderStatus{
        Status: order.Status,
        AvgPrice: order.Price,
        ExecutedQty: order.Quantity,
    }, nil
}
```

## 2. Lighter 实现示例（不支持订单查询）

```go
// trader/lighter/trader.go
package lighter

import (
    "context"
    "github.com/skyold/nofx/trader"
)

type Trader struct {
    // ...
}

// GetCapabilities Lighter 支持的能力（假设不支持订单查询）
func (t *Trader) GetCapabilities() []trader.Action {
    return []trader.Action{
        // 基础交易
        trader.ActionOpenLong,
        trader.ActionOpenShort,
        trader.ActionCloseLong,
        trader.ActionCloseShort,
        
        // 风控
        trader.ActionSetLeverage,
        
        // 注意：Lighter 不支持订单查询和止损止盈
        // trader.ActionSetStopLoss, trader.ActionSetTakeProfit 不在列表中
        // trader.ActionOrderStatusQuery, trader.ActionOpenOrdersQuery 不在列表中
    }
}

// SupportsAction 检查是否支持
func (t *Trader) SupportsAction(action trader.Action) bool {
    capabilities := t.GetCapabilities()
    for _, cap := range capabilities {
        if cap == action {
            return true
        }
    }
    return false
}

// SetStopLoss 不支持，返回错误
func (t *Trader) SetStopLoss(ctx context.Context, orderID string, price float64) error {
    return trader.ErrCapabilityNotSupported
}

// GetOrderStatus 不支持，返回错误
func (t *Trader) GetOrderStatus(ctx context.Context, symbol, orderID string) (*trader.OrderStatus, error) {
    return nil, trader.ErrCapabilityNotSupported
}

// GetOpenOrders 不支持，返回错误
func (t *Trader) GetOpenOrders(ctx context.Context, symbol string) ([]trader.OpenOrder, error) {
    return nil, trader.ErrCapabilityNotSupported
}
```

## 3. 调度器使用示例

```go
// trader/auto_trader.go
func (at *AutoTrader) RunChaosCycle() error {
    // 1. 获取交易所能力列表
    capabilities := at.trader.GetCapabilities()
    
    // 2. 构建系统提示词（包含能力列表）
    systemPrompt := at.buildSystemPromptWithCapabilities(capabilities)
    
    // 3. LLM 返回决策
    decisions := callLLM(systemPrompt, userPrompt)
    
    // 4. 验证决策是否在能力范围内
    for _, decision := range decisions {
        action := trader.Action(decision.Action)
        
        if !at.trader.SupportsAction(action) {
            logger.Warnf("Decision action %s not supported by exchange %s, skipping",
                action, at.trader.GetExchange())
            continue
        }
        
        // 5. 执行决策
        result, err := at.trader.ExecuteDecision(ctx, &decision)
        if err != nil {
            if trader.IsCapabilityNotSupported(err) {
                // 能力不支持，降级处理
                logger.Warnf("Capability not supported, using fallback for %s", action)
                at.handleFallback(action, decision)
            } else {
                logger.Errorf("Execute decision failed: %v", err)
            }
        }
    }
    
    return nil
}

// buildSystemPromptWithCapabilities 构建包含能力列表的系统提示词
func (at *AutoTrader) buildSystemPromptWithCapabilities(caps []trader.Action) string {
    return trader.ToLLMPrompt(at.trader.GetExchange(), caps)
}

// handleFallback 降级处理
func (at *AutoTrader) handleFallback(action trader.Action, decision interface{}) {
    switch action {
    case trader.ActionSetStopLoss:
        // 不支持止损，使用市价单模拟紧急平仓
        at.emergencyClose(decision)
    case trader.ActionOrderStatusQuery:
        // 不支持订单查询，等待固定时间后查询持仓
        time.Sleep(2 * time.Second)
        at.getFillFromPosition(decision)
    }
}
```

## 4. 等待订单成交（带降级）

```go
// trader/auto_trader.go
func (at *AutoTrader) waitForOrderFill(symbol, orderID string) (*trader.OrderFill, error) {
    // 检查是否支持订单查询
    if !at.trader.SupportsAction(trader.ActionOrderStatusQuery) {
        // 降级处理：等待固定时间后直接查询持仓
        logger.Infof("Order status query not supported, using fallback")
        time.Sleep(2 * time.Second)
        return at.getFillFromPosition(symbol)
    }
    
    // 正常轮询
    time.Sleep(500 * time.Millisecond)
    for i := 0; i < 5; i++ {
        status, err := at.trader.GetOrderStatus(ctx, symbol, orderID)
        if err != nil {
            if trader.IsCapabilityNotSupported(err) {
                // 能力不支持，降级
                return at.getFillFromPosition(symbol)
            }
            logger.Warnf("GetOrderStatus failed: %v", err)
            time.Sleep(500 * time.Millisecond)
            continue
        }
        
        if status.Status == "FILLED" {
            return &trader.OrderFill{
                Price: status.AvgPrice,
                Quantity: status.ExecutedQty,
            }, nil
        }
        
        time.Sleep(500 * time.Millisecond)
    }
    
    return nil, fmt.Errorf("order not filled after 5 attempts")
}
```

## 5. 能力列表示例

### Binance Futures

```json
{
  "exchange": "binance",
  "capabilities": [
    "open_long",
    "open_short",
    "close_long",
    "close_short",
    "set_leverage",
    "set_stop_loss",
    "set_take_profit",
    "order_status_query",
    "open_orders_query"
  ]
}
```

### Lighter（简化版）

```json
{
  "exchange": "lighter",
  "capabilities": [
    "open_long",
    "open_short",
    "close_long",
    "close_short",
    "set_leverage"
  ]
}
```

## 6. LLM 提示词示例

```
Exchange: Binance Futures

Supported Actions:
- open_long: Open a long position with specified leverage
  Parameters:
  - symbol (string, required): Trading pair (e.g., "BTCUSDT")
  - quantity (number, required): Position size in base currency
  - leverage (number, required): Leverage multiplier (e.g., 10, 20)

- open_short: Open a short position with specified leverage
  Parameters:
  - symbol (string, required): Trading pair
  - quantity (number, required): Position size
  - leverage (number, required): Leverage multiplier

- set_stop_loss: Set a stop-loss order to limit potential loss
  Parameters:
  - orderID (string, required): Order ID to attach stop-loss
  - price (number, required): Stop-loss trigger price

- order_status_query: Query the status of an order by ID
  Parameters:
  - symbol (string, required): Trading pair
  - orderID (string, required): Order ID to query

IMPORTANT: You can ONLY use the actions listed above.
Do NOT suggest actions that are not in the list.
For example, you cannot use "buy_spot" because this is a futures exchange.
```

## 7. 迁移指南

### 从旧接口迁移到新接口

**旧代码**：
```go
type Trader interface {
    SetStopLoss(symbol string, positionSide string, quantity, stopPrice float64) error
    GetOrderStatus(symbol string, orderID string) (map[string]interface{}, error)
}
```

**新代码**：
```go
type CapableTrader interface {
    GetCapabilities() []Action
    SupportsAction(action Action) bool
    SetStopLoss(ctx context.Context, orderID string, price float64) error
    GetOrderStatus(ctx context.Context, symbol, orderID string) (*OrderStatus, error)
}
```

**迁移步骤**：

1. 实现 `GetCapabilities()` 和 `SupportsAction()`
2. 修改方法签名，添加 `context.Context`
3. 对于不支持的能力，返回 `ErrCapabilityNotSupported`
4. 更新调用方，添加能力检查

```go
// 调用方更新
if capableTrader, ok := trader.(CapableTrader); ok {
    if capableTrader.SupportsAction(trader.ActionSetStopLoss) {
        capableTrader.SetStopLoss(ctx, orderID, price)
    } else {
        // 降级处理
    }
}
```

## 8. 总结

### 优势

1. **清晰的能力列表** - 一眼看出交易所支持什么
2. **LLM 友好** - 直接传递给 LLM，避免无效建议
3. **优雅降级** - 不支持时返回特定错误，调用方可以处理
4. **向后兼容** - 现有代码可以逐步迁移
5. **类型安全** - 使用 Action 类型而非字符串

### 注意事项

1. 所有交易所必须实现基础 action（open_long, open_short, etc.）
2. 可选 action 通过 `SupportsAction()` 检查
3. 不支持的 action 返回 `ErrCapabilityNotSupported`
4. 调用方应该检查能力或处理错误
