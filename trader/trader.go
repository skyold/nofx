package trader

import (
	"context"
	"fmt"
	"nofx/logger"
	"nofx/store"
	"nofx/trader/hyperliquid"
	"nofx/trader/types"
	"time"
)

// Trader 简化后的 Trader
// 只负责交易执行，实现 types.Trader 接口
// 不再包含调度、Engine、MCP Client 等职责
type Trader struct {
	// 基本信息
	id         string
	name       string
	exchange   string
	exchangeID string
	userID     string

	// 底层交易所适配器（实现 types.ExchangeAdapter 接口）
	// Trader 负责适配不同交易所的接口
	adapter types.ExchangeAdapter

	// 配置
	config *TraderConfig
}

// Ensure Trader implements types.Trader
var _ types.Trader = (*Trader)(nil)

// NewTrader 创建简化后的 Trader
// 根据配置自动创建对应的交易所 trader
func NewTrader(config TraderConfig, st *store.Store, userID string) (*Trader, error) {
	// 设置默认值
	if config.ID == "" {
		config.ID = "default_trader"
	}
	if config.Name == "" {
		config.Name = "Default Trader"
	}

	// 设置默认交易所
	if config.Exchange == "" {
		config.Exchange = "binance"
	}

	// 记录仓位模式
	marginModeStr := "Cross Margin"
	if !config.IsCrossMargin {
		marginModeStr = "Isolated Margin"
	}
	logger.Infof("📊 [%s] Position mode: %s", config.Name, marginModeStr)

	// 根据配置创建对应的交易所 trader
	// Trader 负责适配不同交易所的接口
	var adapter types.ExchangeAdapter

	switch config.Exchange {
	case "binance":
		logger.Infof("🏦 [%s] Using Binance Futures trading", config.Name)
		if config.APIKey == "" || config.SecretKey == "" {
			return nil, fmt.Errorf("Binance API key and secret are required")
		}
		// TODO: 实际创建
		// adapter = binance.NewFuturesTrader(...)
		return nil, fmt.Errorf("Binance trader creation not yet implemented")

	case "hyperliquid":
		logger.Infof("🔮 [%s] Using Hyperliquid trading", config.Name)
		if config.HyperliquidWalletAddr == "" || config.HyperliquidPrivateKey == "" {
			return nil, fmt.Errorf("Hyperliquid wallet address and private key are required")
		}
		hyperliquidTrader, err := hyperliquid.NewHyperliquidTrader(
			config.HyperliquidPrivateKey,
			config.HyperliquidWalletAddr,
			config.Testnet,
			config.HyperliquidUnifiedAcct,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to create Hyperliquid trader: %w", err)
		}
		adapter = hyperliquidTrader

	case "aster":
		logger.Infof("✨ [%s] Using Aster DEX trading", config.Name)
		if config.AsterUser == "" || config.AsterPrivateKey == "" {
			return nil, fmt.Errorf("Aster user and private key are required")
		}
		// TODO: 实际创建
		// adapter = aster.NewAsterTrader(...)
		return nil, fmt.Errorf("Aster trader creation not yet implemented")

	case "lighter":
		logger.Infof("💡 [%s] Using LIGHTER DEX trading", config.Name)
		if config.LighterWalletAddr == "" || config.LighterPrivateKey == "" {
			return nil, fmt.Errorf("Lighter wallet address and private key are required")
		}
		// TODO: 实际创建
		// adapter = lighter.NewLighterTrader(...)
		return nil, fmt.Errorf("Lighter trader creation not yet implemented")

	case "okx":
		logger.Infof("🟡 [%s] Using OKX trading", config.Name)
		if config.APIKey == "" || config.SecretKey == "" {
			return nil, fmt.Errorf("OKX API key and secret are required")
		}
		// TODO: 实际创建
		return nil, fmt.Errorf("OKX trader creation not yet implemented")

	case "bybit":
		logger.Infof("📘 [%s] Using Bybit trading", config.Name)
		if config.APIKey == "" || config.SecretKey == "" {
			return nil, fmt.Errorf("Bybit API key and secret are required")
		}
		// TODO: 实际创建
		return nil, fmt.Errorf("Bybit trader creation not yet implemented")

	case "bitget":
		logger.Infof("🟣 [%s] Using Bitget trading", config.Name)
		if config.APIKey == "" || config.SecretKey == "" {
			return nil, fmt.Errorf("Bitget API key and secret are required")
		}
		// TODO: 实际创建
		return nil, fmt.Errorf("Bitget trader creation not yet implemented")

	case "gate":
		logger.Infof("🚪 [%s] Using Gate.io trading", config.Name)
		if config.APIKey == "" || config.SecretKey == "" {
			return nil, fmt.Errorf("Gate.io API key and secret are required")
		}
		// TODO: 实际创建
		return nil, fmt.Errorf("Gate trader creation not yet implemented")

	case "virtual":
		logger.Infof("🎮 [%s] Using Virtual Exchange (Simulation)", config.Name)
		// TODO: 实际创建
		// adapter = virtual.NewVirtualTrader(...)
		return nil, fmt.Errorf("Virtual trader creation not yet implemented")

	default:
		return nil, fmt.Errorf("unsupported trading platform: %s", config.Exchange)
	}

	// 验证初始余额配置（通过适配后的接口）
	if config.InitialBalance <= 0 {
		logger.Infof("📊 [%s] Initial balance not set, attempting to fetch current balance from exchange...", config.Name)
		account, err := adapter.GetBalance()
		if err != nil {
			return nil, fmt.Errorf("initial balance not set and unable to fetch balance from exchange: %w", err)
		}
		// 从 map 中提取 TotalEquity
		if totalEquity, ok := account["total_equity"].(float64); ok && totalEquity > 0 {
			config.InitialBalance = totalEquity
			logger.Infof("✓ [%s] Auto-fetched initial balance: %.2f USDT", config.Name, totalEquity)
			// 保存到数据库
			if st != nil {
				if err := st.Trader().UpdateInitialBalance(userID, config.ID, totalEquity); err != nil {
					logger.Infof("⚠️  [%s] Failed to save initial balance to database: %v", config.Name, err)
				} else {
					logger.Infof("✓ [%s] Initial balance saved to database", config.Name)
				}
			}
		} else {
			return nil, fmt.Errorf("initial balance must be greater than 0")
		}
	}

	return &Trader{
		id:         config.ID,
		name:       config.Name,
		exchange:   config.Exchange,
		exchangeID: config.ExchangeID,
		userID:     userID,
		adapter:    adapter,
		config:     &config,
	}, nil
}

// =============================================================================
// types.Trader 接口实现 - Trader 负责适配和转换
// =============================================================================

// GetExchange 返回交易所名称
func (at *Trader) GetExchange() string {
	return at.exchange
}

// GetCapabilities 获取支持的所有 action 列表
func (at *Trader) GetCapabilities() []types.Action {
	// 返回所有基础交易能力
	return []types.Action{
		types.ActionOpenLong,
		types.ActionOpenShort,
		types.ActionCloseLong,
		types.ActionCloseShort,
	}
}

// SupportsAction 检查是否支持某个 action
func (at *Trader) SupportsAction(action types.Action) bool {
	for _, cap := range at.GetCapabilities() {
		if cap == action {
			return true
		}
	}
	return false
}

// GetAccountInfo 获取账户信息 - 适配返回类型
func (at *Trader) GetAccountInfo(ctx context.Context) (*types.AccountInfo, error) {
	// 调用底层交易所接口（无 context）
	accountMap, err := at.adapter.GetBalance()
	if err != nil {
		return nil, err
	}

	// 适配：从 map 转换为强类型
	return &types.AccountInfo{
		TotalEquity:        getFloat64(accountMap, "total_equity"),
		AvailableBalance:   getFloat64(accountMap, "available_balance"),
		TotalUnrealizedPnL: getFloat64(accountMap, "total_unrealized_pnl"),
		TotalRealizedPnL:   getFloat64(accountMap, "total_realized_pnl"),
		MarginUsed:         getFloat64(accountMap, "margin_used"),
		MarginRatio:        getFloat64(accountMap, "margin_ratio"),
		AccountID:          getString(accountMap, "account_id"),
	}, nil
}

// GetPositions 获取持仓信息 - 适配返回类型
func (at *Trader) GetPositions(ctx context.Context) ([]types.PositionInfo, error) {
	// 调用底层交易所接口（无 context）
	positionsMap, err := at.adapter.GetPositions()
	if err != nil {
		return nil, err
	}

	// 适配：从 []map 转换为 []PositionInfo
	positions := make([]types.PositionInfo, 0, len(positionsMap))
	for _, posMap := range positionsMap {
		positions = append(positions, types.PositionInfo{
			Symbol:           getString(posMap, "symbol"),
			Side:             getString(posMap, "side"),
			Quantity:         getFloat64(posMap, "quantity"),
			EntryPrice:       getFloat64(posMap, "entry_price"),
			MarkPrice:        getFloat64(posMap, "mark_price"),
			Leverage:         getInt(posMap, "leverage"),
			UnrealizedPnL:    getFloat64(posMap, "unrealized_pnl"),
			LiquidationPrice: getFloat64(posMap, "liquidation_price"),
			MarginMode:       getString(posMap, "margin_mode"),
			PositionID:       getString(posMap, "position_id"),
		})
	}

	return positions, nil
}

// GetMarketPrice 获取市场价格
func (at *Trader) GetMarketPrice(ctx context.Context, symbol string) (float64, error) {
	// 直接调用底层接口（无 context）
	return at.adapter.GetMarketPrice(symbol)
}

// OpenLong 开多仓 - 适配参数和返回类型
func (at *Trader) OpenLong(ctx context.Context, symbol string, quantity float64, leverage int) (*types.Order, error) {
	// 调用底层交易所接口（无 context，返回 map）
	orderMap, err := at.adapter.OpenLong(symbol, quantity, leverage)
	if err != nil {
		return nil, err
	}

	// 适配：从 map 转换为 *Order
	return at.mapToOrder(orderMap), nil
}

// OpenShort 开空仓 - 适配参数和返回类型
func (at *Trader) OpenShort(ctx context.Context, symbol string, quantity float64, leverage int) (*types.Order, error) {
	orderMap, err := at.adapter.OpenShort(symbol, quantity, leverage)
	if err != nil {
		return nil, err
	}

	return at.mapToOrder(orderMap), nil
}

// CloseLong 平多仓 - 适配参数和返回类型
func (at *Trader) CloseLong(ctx context.Context, symbol string, quantity float64) (*types.Order, error) {
	orderMap, err := at.adapter.CloseLong(symbol, quantity)
	if err != nil {
		return nil, err
	}

	return at.mapToOrder(orderMap), nil
}

// CloseShort 平空仓 - 适配参数和返回类型
func (at *Trader) CloseShort(ctx context.Context, symbol string, quantity float64) (*types.Order, error) {
	orderMap, err := at.adapter.CloseShort(symbol, quantity)
	if err != nil {
		return nil, err
	}

	return at.mapToOrder(orderMap), nil
}

// SetLeverage 设置杠杆
func (at *Trader) SetLeverage(ctx context.Context, symbol string, leverage int) error {
	// 直接调用底层接口
	return at.adapter.SetLeverage(symbol, leverage)
}

// SetStopLoss 设置止损 - TODO: 根据交易所实现
func (at *Trader) SetStopLoss(ctx context.Context, orderID string, price float64) error {
	return fmt.Errorf("SetStopLoss not yet implemented")
}

// SetTakeProfit 设置止盈 - TODO: 根据交易所实现
func (at *Trader) SetTakeProfit(ctx context.Context, orderID string, price float64) error {
	return fmt.Errorf("SetTakeProfit not yet implemented")
}

// GetOrderStatus 查询订单状态 - TODO: 根据交易所实现
func (at *Trader) GetOrderStatus(ctx context.Context, symbol, orderID string) (*types.OrderStatus, error) {
	return nil, fmt.Errorf("GetOrderStatus not yet implemented")
}

// GetOpenOrders 查询未成交订单 - TODO: 根据交易所实现
func (at *Trader) GetOpenOrders(ctx context.Context, symbol string) ([]types.OpenOrder, error) {
	return nil, fmt.Errorf("GetOpenOrders not yet implemented")
}

// CancelOrder 取消订单 - TODO
func (at *Trader) CancelOrder(ctx context.Context, symbol string, orderID string) (*types.Order, error) {
	return nil, fmt.Errorf("CancelOrder not yet implemented")
}

// CancelAllOrders 取消所有订单 - TODO
func (at *Trader) CancelAllOrders(ctx context.Context, symbol string) error {
	return fmt.Errorf("CancelAllOrders not yet implemented")
}

// ExecuteDecision 执行交易决策
func (at *Trader) ExecuteDecision(ctx context.Context, decision interface{}) (*types.OrderResult, error) {
	// TODO: 实现决策执行逻辑
	return &types.OrderResult{
		Success: false,
		Message: "ExecuteDecision not yet implemented for Trader",
	}, nil
}

// =============================================================================
// 辅助方法 - 类型转换
// =============================================================================

// mapToOrder 将 map 转换为 *types.Order
func (at *Trader) mapToOrder(orderMap map[string]interface{}) *types.Order {
	if orderMap == nil {
		return nil
	}

	return &types.Order{
		OrderID:        getString(orderMap, "order_id"),
		Symbol:         getString(orderMap, "symbol"),
		Side:           getString(orderMap, "side"),
		Type:           getString(orderMap, "type"),
		Quantity:       getFloat64(orderMap, "quantity"),
		Price:          getFloat64(orderMap, "price"),
		AvgFillPrice:   getFloat64(orderMap, "avg_fill_price"),
		FilledQuantity: getFloat64(orderMap, "filled_quantity"),
		Status:         getString(orderMap, "status"),
		CreatedAt:      getTime(orderMap, "created_at"),
		FilledAt:       getTime(orderMap, "filled_at"),
		Fee:            getFloat64(orderMap, "fee"),
		FeeAsset:       getString(orderMap, "fee_asset"),
	}
}

// getFloat64 从 map 中安全获取 float64 值
func getFloat64(m map[string]interface{}, key string) float64 {
	if m == nil {
		return 0
	}
	if v, ok := m[key].(float64); ok {
		return v
	}
	return 0
}

// getInt 从 map 中安全获取 int 值
func getInt(m map[string]interface{}, key string) int {
	if m == nil {
		return 0
	}
	if v, ok := m[key].(int); ok {
		return v
	}
	if v, ok := m[key].(float64); ok {
		return int(v)
	}
	return 0
}

// getString 从 map 中安全获取 string 值
func getString(m map[string]interface{}, key string) string {
	if m == nil {
		return ""
	}
	if v, ok := m[key].(string); ok {
		return v
	}
	return ""
}

// getTime 从 map 中安全获取 time.Time 值
func getTime(m map[string]interface{}, key string) time.Time {
	if m == nil {
		return time.Time{}
	}
	if v, ok := m[key].(time.Time); ok {
		return v
	}
	return time.Time{}
}

// =============================================================================
// 辅助方法
// =============================================================================

// GetID 返回 Trader ID
func (at *Trader) GetID() string {
	return at.id
}

// GetName 返回 Trader 名称
func (at *Trader) GetName() string {
	return at.name
}

// GetConfig 返回配置
func (at *Trader) GetConfig() *TraderConfig {
	return at.config
}

// Close 关闭 Trader
func (at *Trader) Close() error {
	return nil
}
