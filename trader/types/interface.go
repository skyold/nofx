package types

import (
	"context"
	"time"
)

// Action 定义所有可能的交易动作
type Action string

const (
	// 基础动作
	ActionOpenLong   Action = "open_long"
	ActionOpenShort  Action = "open_short"
	ActionCloseLong  Action = "close_long"
	ActionCloseShort Action = "close_short"

	// 风控动作
	ActionSetLeverage   Action = "set_leverage"
	ActionSetStopLoss   Action = "set_stop_loss"
	ActionSetTakeProfit Action = "set_take_profit"

	// 订单查询动作
	ActionOrderStatusQuery Action = "order_status_query"
	ActionOpenOrdersQuery  Action = "open_orders_query"
	ActionCancelOrder      Action = "cancel_order"
	ActionCancelAllOrders  Action = "cancel_all_orders"
)

// Trader 统一交易器接口
// 这是上层应用（Scheduler/Engine）调用的接口
// Trader 负责实现这个接口并适配底层交易所
type Trader interface {
	// === 基础信息 ===

	// GetExchange 返回交易所名称
	GetExchange() string

	// === 能力查询 ===

	// GetCapabilities 获取支持的所有 action 列表
	GetCapabilities() []Action

	// SupportsAction 检查是否支持某个 action
	SupportsAction(action Action) bool

	// === 账户查询 ===

	// GetAccountInfo 获取账户信息
	GetAccountInfo(ctx context.Context) (*AccountInfo, error)

	// GetPositions 获取持仓信息
	GetPositions(ctx context.Context) ([]PositionInfo, error)

	// GetMarketPrice 获取市场价格
	GetMarketPrice(ctx context.Context, symbol string) (float64, error)

	// === 基础交易 ===

	// OpenLong 开多仓
	OpenLong(ctx context.Context, symbol string, quantity float64, leverage int) (*Order, error)

	// OpenShort 开空仓
	OpenShort(ctx context.Context, symbol string, quantity float64, leverage int) (*Order, error)

	// CloseLong 平多仓
	CloseLong(ctx context.Context, symbol string, quantity float64) (*Order, error)

	// CloseShort 平空仓
	CloseShort(ctx context.Context, symbol string, quantity float64) (*Order, error)

	// === 风控（可选） ===

	// SetLeverage 设置杠杆
	SetLeverage(ctx context.Context, symbol string, leverage int) error

	// SetStopLoss 设置止损
	SetStopLoss(ctx context.Context, orderID string, price float64) error

	// SetTakeProfit 设置止盈
	SetTakeProfit(ctx context.Context, orderID string, price float64) error

	// === 订单管理（可选） ===

	// GetOrderStatus 查询订单状态
	GetOrderStatus(ctx context.Context, symbol, orderID string) (*OrderStatus, error)

	// GetOpenOrders 查询未成交订单
	GetOpenOrders(ctx context.Context, symbol string) ([]OpenOrder, error)

	// CancelOrder 取消订单
	CancelOrder(ctx context.Context, symbol string, orderID string) (*Order, error)

	// CancelAllOrders 取消所有订单
	CancelAllOrders(ctx context.Context, symbol string) error

	// === 统一执行 ===

	// ExecuteDecision 执行交易决策
	ExecuteDecision(ctx context.Context, decision interface{}) (*OrderResult, error)
}

// ExchangeAdapter 底层交易所适配器接口
// 这是 Trader 持有的接口，用于适配各个交易所的原始实现
// 所有交易所的原始方法签名都类似这样（无 context，返回 map）
type ExchangeAdapter interface {
	// === 账户查询（无 context，返回 map） ===

	// GetBalance 获取账户余额
	GetBalance() (map[string]interface{}, error)

	// GetPositions 获取持仓信息
	GetPositions() ([]map[string]interface{}, error)

	// GetMarketPrice 获取市场价格
	GetMarketPrice(symbol string) (float64, error)

	// === 基础交易（无 context，返回 map） ===

	// OpenLong 开多仓
	OpenLong(symbol string, quantity float64, leverage int) (map[string]interface{}, error)

	// OpenShort 开空仓
	OpenShort(symbol string, quantity float64, leverage int) (map[string]interface{}, error)

	// CloseLong 平多仓
	CloseLong(symbol string, quantity float64) (map[string]interface{}, error)

	// CloseShort 平空仓
	CloseShort(symbol string, quantity float64) (map[string]interface{}, error)

	// === 风控 ===

	// SetLeverage 设置杠杆
	SetLeverage(symbol string, leverage int) error

	// === 订单管理 ===

	// CancelOrder 取消订单
	CancelOrder(symbol string, orderID string) error

	// CancelAllOrders 取消所有订单（可选 symbol 参数）
	CancelAllOrders(symbol string) error
}

// === 数据类型定义 ===

// AccountInfo 账户信息
type AccountInfo struct {
	TotalEquity        float64 `json:"total_equity"`         // 总权益
	AvailableBalance   float64 `json:"available_balance"`    // 可用余额
	TotalUnrealizedPnL float64 `json:"total_unrealized_pnl"` // 总未实现盈亏
	TotalRealizedPnL   float64 `json:"total_realized_pnl"`   // 总已实现盈亏
	MarginUsed         float64 `json:"margin_used"`          // 已使用保证金
	MarginRatio        float64 `json:"margin_ratio"`         // 保证金率
	AccountID          string  `json:"account_id"`           // 账户 ID
}

// PositionInfo 持仓信息
type PositionInfo struct {
	Symbol           string  `json:"symbol"`            // 交易对
	Side             string  `json:"side"`              // 方向：LONG/SHORT
	Quantity         float64 `json:"quantity"`          // 数量
	EntryPrice       float64 `json:"entry_price"`       // 入场价
	MarkPrice        float64 `json:"mark_price"`        // 标记价格
	Leverage         int     `json:"leverage"`          // 杠杆
	UnrealizedPnL    float64 `json:"unrealized_pnl"`    // 未实现盈亏
	LiquidationPrice float64 `json:"liquidation_price"` // 强平价格
	MarginMode       string  `json:"margin_mode"`       // 保证金模式：cross/isolated
	PositionID       string  `json:"position_id"`       // 持仓 ID
}

// Order 订单信息
type Order struct {
	OrderID        string    `json:"order_id"`        // 订单 ID
	Symbol         string    `json:"symbol"`          // 交易对
	Side           string    `json:"side"`            // 方向：BUY/SELL
	Type           string    `json:"type"`            // 类型：MARKET/LIMIT
	Quantity       float64   `json:"quantity"`        // 数量
	Price          float64   `json:"price"`           // 价格
	AvgFillPrice   float64   `json:"avg_fill_price"`  // 平均成交价
	FilledQuantity float64   `json:"filled_quantity"` // 已成交数量
	Status         string    `json:"status"`          // 状态：NEW/FILLED/CANCELED
	CreatedAt      time.Time `json:"created_at"`      // 创建时间
	FilledAt       time.Time `json:"filled_at"`       // 成交时间
	Fee            float64   `json:"fee"`             // 手续费
	FeeAsset       string    `json:"fee_asset"`       // 手续费币种
}

// OrderStatus 订单状态
type OrderStatus struct {
	OrderID      string  `json:"order_id"`      // 订单 ID
	Status       string  `json:"status"`        // 状态
	AvgPrice     float64 `json:"avg_price"`     // 平均价格
	ExecutedQty  float64 `json:"executed_qty"`  // 已执行数量
	RemainingQty float64 `json:"remaining_qty"` // 剩余数量
	Fee          float64 `json:"fee"`           // 手续费
}

// OrderResult 订单执行结果
type OrderResult struct {
	Success      bool   `json:"success"`                  // 是否成功
	Order        *Order `json:"order,omitempty"`          // 订单信息
	OrderID      string `json:"order_id,omitempty"`       // 订单 ID
	StopLossID   string `json:"stop_loss_id,omitempty"`   // 止损订单 ID
	TakeProfitID string `json:"take_profit_id,omitempty"` // 止盈订单 ID
	Error        error  `json:"error,omitempty"`          // 错误
	Message      string `json:"message,omitempty"`        // 消息
}

// OpenOrder 未成交订单
type OpenOrder struct {
	OrderID      string  `json:"order_id"`      // 订单 ID
	Symbol       string  `json:"symbol"`        // 交易对
	Side         string  `json:"side"`          // 方向：BUY/SELL
	PositionSide string  `json:"position_side"` // 持仓方向：LONG/SHORT
	Type         string  `json:"type"`          // 类型
	Price        float64 `json:"price"`         // 价格
	StopPrice    float64 `json:"stop_price"`    // 触发价
	Quantity     float64 `json:"quantity"`      // 数量
	Status       string  `json:"status"`        // 状态
}

// ClosedPnLRecord 已实现盈亏记录
type ClosedPnLRecord struct {
	Symbol      string    `json:"symbol"`       // 交易对
	Side        string    `json:"side"`         // 方向
	EntryPrice  float64   `json:"entry_price"`  // 入场价
	ExitPrice   float64   `json:"exit_price"`   // 出场价
	Quantity    float64   `json:"quantity"`     // 数量
	RealizedPnL float64   `json:"realized_pnl"` // 已实现盈亏
	Fee         float64   `json:"fee"`          // 手续费
	Leverage    int       `json:"leverage"`     // 杠杆
	EntryTime   time.Time `json:"entry_time"`   // 入场时间
	ExitTime    time.Time `json:"exit_time"`    // 出场时间
	OrderID     string    `json:"order_id"`     // 订单 ID
	CloseType   string    `json:"close_type"`   // 平仓类型
	ExchangeID  string    `json:"exchange_id"`  // 交易所 ID
}

// TradeRecord 交易记录
type TradeRecord struct {
	TradeID      string    `json:"trade_id"`      // 交易 ID
	Symbol       string    `json:"symbol"`        // 交易对
	Side         string    `json:"side"`          // 方向
	PositionSide string    `json:"position_side"` // 持仓方向
	OrderAction  string    `json:"order_action"`  // 订单动作
	Price        float64   `json:"price"`         // 价格
	Quantity     float64   `json:"quantity"`      // 数量
	RealizedPnL  float64   `json:"realized_pnl"`  // 已实现盈亏
	Fee          float64   `json:"fee"`           // 手续费
	Time         time.Time `json:"time"`          // 时间
}

// LimitOrderRequest 限价单请求（用于 Grid 交易）
type LimitOrderRequest struct {
	Symbol       string  `json:"symbol"`        // 交易对
	Side         string  `json:"side"`          // 方向
	PositionSide string  `json:"position_side"` // 持仓方向
	Price        float64 `json:"price"`         // 限价
	Quantity     float64 `json:"quantity"`      // 数量
	Leverage     int     `json:"leverage"`      // 杠杆
	PostOnly     bool    `json:"post_only"`     // 是否 PostOnly
	ReduceOnly   bool    `json:"reduce_only"`   // 是否 ReduceOnly
	ClientID     string  `json:"client_id"`     // 客户端订单 ID
}

// LimitOrderResult 限价单结果
type LimitOrderResult struct {
	OrderID      string  `json:"order_id"`      // 订单 ID
	ClientID     string  `json:"client_id"`     // 客户端订单 ID
	Symbol       string  `json:"symbol"`        // 交易对
	Side         string  `json:"side"`          // 方向
	PositionSide string  `json:"position_side"` // 持仓方向
	Price        float64 `json:"price"`         // 价格
	Quantity     float64 `json:"quantity"`      // 数量
	Status       string  `json:"status"`        // 状态
}
