package chaos

import (
	"fmt"
	"nofx/logger"
	"nofx/market"
	"nofx/store"
	"strings"
	"time"
)

// TraderInterface defines the subset of trader methods needed by ChaosExecutor
// Defined locally to avoid import cycles with trader package
type TraderInterface interface {
	GetBalance() (map[string]interface{}, error)
	GetPositions() ([]map[string]interface{}, error)
	OpenLong(symbol string, quantity float64, leverage int) (map[string]interface{}, error)
	OpenShort(symbol string, quantity float64, leverage int) (map[string]interface{}, error)
	CloseLong(symbol string, quantity float64) (map[string]interface{}, error)
	CloseShort(symbol string, quantity float64) (map[string]interface{}, error)
	SetLeverage(symbol string, leverage int) error
	SetMarginMode(symbol string, isCrossMargin bool) error
	SetStopLoss(symbol string, positionSide string, quantity, stopPrice float64) error
	SetTakeProfit(symbol string, positionSide string, quantity, takeProfitPrice float64) error
	GetOrderStatus(symbol string, orderID string) (map[string]interface{}, error)
}

// ChaosExecutor executes chaos decisions
type ChaosExecutor struct {
	trader     TraderInterface
	store      *store.Store
	traderID   string
	exchange   string
	exchangeID string
	config     *ChaosConfig
}

// NewChaosExecutor creates a new executor
func NewChaosExecutor(trader TraderInterface, store *store.Store, traderID, exchange, exchangeID string, config *ChaosConfig) *ChaosExecutor {
	return &ChaosExecutor{
		trader:     trader,
		store:      store,
		traderID:   traderID,
		exchange:   exchange,
		exchangeID: exchangeID,
		config:     config,
	}
}

// Execute executes a list of decisions
func (e *ChaosExecutor) Execute(decisions []Decision) ([]store.DecisionAction, []string) {
	// Sort decisions (close first)
	sorted := e.sortDecisions(decisions)
	var results []store.DecisionAction
	var logs []string

	msg := fmt.Sprintf("⚡ ChaosExecutor executing %d decisions...", len(sorted))
	logger.Info(msg)
	logs = append(logs, msg)

	for _, d := range sorted {
		// Create action record
		actionRecord := store.DecisionAction{
			Action:    d.Action,
			Symbol:    d.Symbol,
			Timestamp: time.Now().UTC(),
		}
		if d.Leverage != nil {
			actionRecord.Leverage = *d.Leverage
		}
		if d.StopLoss != nil {
			actionRecord.StopLoss = *d.StopLoss
		}
		if d.TakeProfit != nil {
			actionRecord.TakeProfit = *d.TakeProfit
		}
		if d.TotalScore != nil {
			actionRecord.Confidence = *d.TotalScore
		}

		if err := e.executeSingle(&d, &actionRecord); err != nil {
			errMsg := fmt.Sprintf("❌ Chaos execution failed (%s %s): %v", d.Symbol, d.Action, err)
			logger.Error(errMsg)
			logs = append(logs, errMsg)
			actionRecord.Success = false
			actionRecord.Error = err.Error()
		} else {
			successMsg := fmt.Sprintf("✓ Chaos execution succeeded (%s %s)", d.Symbol, d.Action)
			logger.Info(successMsg)
			logs = append(logs, successMsg)
			actionRecord.Success = true
		}

		results = append(results, actionRecord)
	}
	return results, logs
}

func (e *ChaosExecutor) sortDecisions(decisions []Decision) []Decision {
	if len(decisions) <= 1 {
		return decisions
	}

	getActionPriority := func(action string) int {
		switch action {
		case "close_long", "close_short":
			return 1
		case "open_long", "open_short":
			return 2
		default:
			return 3
		}
	}

	sorted := make([]Decision, len(decisions))
	copy(sorted, decisions)

	for i := 0; i < len(sorted)-1; i++ {
		for j := i + 1; j < len(sorted); j++ {
			if getActionPriority(sorted[i].Action) > getActionPriority(sorted[j].Action) {
				sorted[i], sorted[j] = sorted[j], sorted[i]
			}
		}
	}
	return sorted
}

func (e *ChaosExecutor) executeSingle(d *Decision, record *store.DecisionAction) error {
	switch d.Action {
	case "open_long":
		return e.executeOpen(d, "LONG", record)
	case "open_short":
		return e.executeOpen(d, "SHORT", record)
	case "close_long":
		return e.executeClose(d, "LONG", record)
	case "close_short":
		return e.executeClose(d, "SHORT", record)
	case "hold", "wait":
		return nil
	default:
		return fmt.Errorf("unknown action: %s", d.Action)
	}
}

func (e *ChaosExecutor) executeOpen(d *Decision, side string, record *store.DecisionAction) error {
	// 1. Get Market Price
	marketData, err := market.GetWithExchange(d.Symbol, e.exchange)
	if err != nil {
		return fmt.Errorf("failed to get market data: %w", err)
	}
	price := marketData.CurrentPrice
	record.Price = price

	// 2. Calculate Quantity
	positionSizeUSD := 12.0 // Default minimum
	if d.PositionSizeUSD != nil && *d.PositionSizeUSD > 0 {
		positionSizeUSD = *d.PositionSizeUSD
	}

	quantity := positionSizeUSD / price
	record.Quantity = quantity

	// 3. Leverage
	leverage := 10
	if d.Leverage != nil {
		leverage = *d.Leverage
	}

	// 4. Execute Open
	var order map[string]interface{}
	if side == "LONG" {
		order, err = e.trader.OpenLong(d.Symbol, quantity, leverage)
	} else {
		order, err = e.trader.OpenShort(d.Symbol, quantity, leverage)
	}

	if err != nil {
		return err
	}

	// 5. Record Order
	if orderID, ok := order["orderId"]; ok {
		e.recordOrder(orderID, d.Symbol, d.Action, quantity, price, leverage)

		// Save orderID to record
		switch v := orderID.(type) {
		case int64:
			record.OrderID = v
		case float64:
			record.OrderID = int64(v)
		}
	}

	// 6. Set TPSL
	if d.StopLoss != nil {
		e.trader.SetStopLoss(d.Symbol, side, quantity, *d.StopLoss)
	}
	if d.TakeProfit != nil {
		e.trader.SetTakeProfit(d.Symbol, side, quantity, *d.TakeProfit)
	}

	return nil
}

func (e *ChaosExecutor) executeClose(d *Decision, side string, record *store.DecisionAction) error {
	// 1. Get Market Price
	marketData, err := market.GetWithExchange(d.Symbol, e.exchange)
	if err != nil {
		return fmt.Errorf("failed to get market data: %w", err)
	}
	price := marketData.CurrentPrice
	record.Price = price

	// 2. Determine Quantity (Close All)
	// In Chaos, we might want to support partial closes, but for now Close All is safe
	quantity := 0.0 // 0 means close all

	// 3. Execute Close
	var order map[string]interface{}
	if side == "LONG" {
		order, err = e.trader.CloseLong(d.Symbol, quantity)
	} else {
		order, err = e.trader.CloseShort(d.Symbol, quantity)
	}

	if err != nil {
		return err
	}

	// 4. Record Order
	if orderID, ok := order["orderId"]; ok {
		// Try to find quantity if 0 passed
		actualQty := quantity
		if quantity == 0 {
			// Try to find from open position to record accurate qty
			// Or wait for fill. For simplicity here:
			actualQty = 0 // Will be updated by sync
		}
		e.recordOrder(orderID, d.Symbol, d.Action, actualQty, price, 0)

		switch v := orderID.(type) {
		case int64:
			record.OrderID = v
		case float64:
			record.OrderID = int64(v)
		}
	}

	return nil
}

func (e *ChaosExecutor) recordOrder(orderID interface{}, symbol, action string, quantity, price float64, leverage int) {
	if e.store == nil {
		return
	}

	// Format ID
	var oidStr string
	switch v := orderID.(type) {
	case string:
		oidStr = v
	case int64:
		oidStr = fmt.Sprintf("%d", v)
	case float64:
		oidStr = fmt.Sprintf("%.0f", v)
	default:
		oidStr = fmt.Sprintf("%v", v)
	}

	clientOrderID := fmt.Sprintf("chaos_%d", time.Now().UnixNano())

	// Map action to side
	side := "BUY"
	if action == "open_short" || action == "close_long" {
		side = "SELL"
	}

	positionSide := "LONG"
	if action == "open_short" || action == "close_short" {
		positionSide = "SHORT"
	}

	reduceOnly := strings.HasPrefix(action, "close")

	order := &store.TraderOrder{
		TraderID:        e.traderID,
		ExchangeID:      e.exchangeID,
		ExchangeType:    e.exchange,
		ExchangeOrderID: oidStr,
		ClientOrderID:   clientOrderID,
		Symbol:          market.Normalize(symbol),
		Side:            side,
		PositionSide:    positionSide,
		Type:            "MARKET",
		TimeInForce:     "GTC",
		Quantity:        quantity,
		Price:           price,
		Status:          "NEW",
		Leverage:        leverage,
		ReduceOnly:      reduceOnly,
		ClosePosition:   reduceOnly,
		OrderAction:     action,
		CreatedAt:       store.UnixTime(time.Now().UTC().UnixMilli()),
		UpdatedAt:       store.UnixTime(time.Now().UTC().UnixMilli()),
	}

	e.store.Order().CreateOrder(order)
}
