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

// ExecuteDecisions executes a list of decisions
func (e *ChaosExecutor) ExecuteDecisions(decisions []Decision) ([]store.DecisionAction, []string) {
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

	// 5. Record Order and Confirm
	e.recordAndConfirmOrder(order, d.Symbol, d.Action, quantity, price, leverage, 0)

	// Update action record with order ID
	if orderID, ok := order["orderId"]; ok {
		switch v := orderID.(type) {
		case int64:
			record.OrderID = v
		case float64:
			record.OrderID = int64(v)
		case string:
			// Try parse string to int64
			var id int64
			fmt.Sscanf(v, "%d", &id)
			record.OrderID = id
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
	normalizedSymbol := market.Normalize(d.Symbol)
	var quantity float64
	var entryPrice float64

	// Try to get from local database
	if e.store != nil {
		posSide := "LONG"
		if side == "SHORT" {
			posSide = "SHORT"
		}
		if openPos, err := e.store.Position().GetOpenPositionBySymbol(e.traderID, normalizedSymbol, posSide); err == nil && openPos != nil {
			quantity = openPos.Quantity
			entryPrice = openPos.EntryPrice
			logger.Infof("  📊 [Chaos] Using local position data: qty=%.8f, entry=%.2f", quantity, entryPrice)
		}
	}

	// Fallback to exchange API if local data not found
	if quantity == 0 {
		positions, err := e.trader.GetPositions()
		if err == nil {
			for _, pos := range positions {
				if pos["symbol"] == d.Symbol && strings.ToUpper(pos["side"].(string)) == side {
					if ep, ok := pos["entryPrice"].(float64); ok {
						entryPrice = ep
					}
					if amt, ok := pos["positionAmt"].(float64); ok {
						if amt < 0 {
							quantity = -amt
						} else {
							quantity = amt
						}
					}
					break
				}
			}
		}
		logger.Infof("  📊 [Chaos] Using exchange position data: qty=%.8f, entry=%.2f", quantity, entryPrice)
	}

	// 3. Execute Close
	var order map[string]interface{}
	if side == "LONG" {
		order, err = e.trader.CloseLong(d.Symbol, 0) // 0 means close all
	} else {
		order, err = e.trader.CloseShort(d.Symbol, 0)
	}

	if err != nil {
		return err
	}

	// 4. Record Order and Confirm
	e.recordAndConfirmOrder(order, d.Symbol, d.Action, quantity, price, 0, entryPrice)

	// Update action record with order ID
	if orderID, ok := order["orderId"]; ok {
		switch v := orderID.(type) {
		case int64:
			record.OrderID = v
		case float64:
			record.OrderID = int64(v)
		case string:
			var id int64
			fmt.Sscanf(v, "%d", &id)
			record.OrderID = id
		}
	}

	return nil
}

func (e *ChaosExecutor) recordAndConfirmOrder(orderResult map[string]interface{}, symbol, action string, quantity float64, price float64, leverage int, entryPrice float64) {
	if e.store == nil {
		return
	}

	// Format ID
	var orderID string
	switch v := orderResult["orderId"].(type) {
	case string:
		orderID = v
	case int64:
		orderID = fmt.Sprintf("%d", v)
	case float64:
		orderID = fmt.Sprintf("%.0f", v)
	default:
		orderID = fmt.Sprintf("%v", v)
	}

	if orderID == "" || orderID == "0" {
		logger.Infof("  ⚠️ [Chaos] Order ID is empty, skipping record")
		return
	}

	clientOrderID := fmt.Sprintf("chaos_%d", time.Now().UnixNano())

	// Determine positionSide
	var positionSide string
	switch action {
	case "open_long", "close_long":
		positionSide = "LONG"
	case "open_short", "close_short":
		positionSide = "SHORT"
	}

	// Special handling for exchanges with OrderSync
	switch e.exchange {
	case "binance", "lighter", "hyperliquid", "bybit", "okx", "bitget", "aster", "kucoin", "gate":
		orderRecord := e.createOrderRecord(orderID, clientOrderID, symbol, action, positionSide, quantity, price, leverage)
		if err := e.store.Order().CreateOrder(orderRecord); err != nil {
			logger.Infof("  ⚠️ [Chaos] Failed to record order: %v", err)
		} else {
			logger.Infof("  📝 [Chaos] Order recorded: %s [%s] %s (status: NEW)", orderID, action, symbol)
		}
		logger.Infof("  📝 [Chaos] Order submitted (id: %s), will be synced by OrderSync", orderID)
		return
	}

	// For other exchanges, record and poll
	orderRecord := e.createOrderRecord(orderID, clientOrderID, symbol, action, positionSide, quantity, price, leverage)
	if err := e.store.Order().CreateOrder(orderRecord); err != nil {
		logger.Infof("  ⚠️ [Chaos] Failed to record order: %v", err)
	}

	var actualPrice = price
	var actualQty = quantity
	var fee float64

	// Wait for fill
	time.Sleep(500 * time.Millisecond)
	for i := 0; i < 5; i++ {
		status, err := e.trader.GetOrderStatus(symbol, orderID)
		if err == nil {
			statusStr, _ := status["status"].(string)
			if statusStr == "FILLED" {
				if avgPrice, ok := status["avgPrice"].(float64); ok && avgPrice > 0 {
					actualPrice = avgPrice
				}
				if execQty, ok := status["executedQty"].(float64); ok && execQty > 0 {
					actualQty = execQty
				}
				if commission, ok := status["commission"].(float64); ok {
					fee = commission
				}
				logger.Infof("  ✅ [Chaos] Order filled: avgPrice=%.6f, qty=%.6f, fee=%.6f", actualPrice, actualQty, fee)

				e.store.Order().UpdateOrderStatus(orderRecord.ID, "FILLED", actualQty, actualPrice, fee)
				e.recordOrderFill(orderRecord.ID, orderID, symbol, action, actualPrice, actualQty, fee)
				break
			} else if statusStr == "CANCELED" || statusStr == "EXPIRED" || statusStr == "REJECTED" {
				logger.Infof("  ⚠️ [Chaos] Order %s, skipping position record", statusStr)
				e.store.Order().UpdateOrderStatus(orderRecord.ID, statusStr, 0, 0, 0)
				return
			}
		}
		time.Sleep(500 * time.Millisecond)
	}

	normalizedSymbol := market.Normalize(symbol)
	e.recordPositionChange(orderID, normalizedSymbol, positionSide, action, actualQty, actualPrice, leverage, entryPrice, fee)
}

func (e *ChaosExecutor) createOrderRecord(orderID, clientOrderID, symbol, action, positionSide string, quantity, price float64, leverage int) *store.TraderOrder {
	// Determine order type (market for chaos)
	orderType := "MARKET"

	// Determine side (BUY/SELL)
	var side string
	switch action {
	case "open_long", "close_short":
		side = "BUY"
	case "open_short", "close_long":
		side = "SELL"
	default:
		side = "BUY"
	}

	reduceOnly := strings.HasPrefix(action, "close")

	return &store.TraderOrder{
		TraderID:        e.traderID,
		ExchangeID:      e.exchangeID,
		ExchangeType:    e.exchange,
		ExchangeOrderID: orderID,
		ClientOrderID:   clientOrderID,
		Symbol:          market.Normalize(symbol),
		Side:            side,
		PositionSide:    positionSide,
		Type:            orderType,
		TimeInForce:     "GTC",
		Quantity:        quantity,
		Price:           price,
		Status:          "NEW",
		FilledQuantity:  0,
		AvgFillPrice:    0,
		Commission:      0,
		CommissionAsset: "USDT",
		Leverage:        leverage,
		ReduceOnly:      reduceOnly,
		ClosePosition:   reduceOnly,
		OrderAction:     action,
		CreatedAt:       time.Now().UTC().UnixMilli(),
		UpdatedAt:       time.Now().UTC().UnixMilli(),
	}
}

func (e *ChaosExecutor) recordOrderFill(orderRecordID int64, exchangeOrderID, symbol, action string, price, quantity, fee float64) {
	if e.store == nil {
		return
	}

	// Determine side (BUY/SELL)
	var side string
	switch action {
	case "open_long", "close_short":
		side = "BUY"
	case "open_short", "close_long":
		side = "SELL"
	default:
		side = "BUY"
	}

	tradeID := fmt.Sprintf("%s-%d", exchangeOrderID, time.Now().UnixNano())

	fill := &store.TraderFill{
		TraderID:        e.traderID,
		ExchangeID:      e.exchangeID,
		ExchangeType:    e.exchange,
		OrderID:         orderRecordID,
		ExchangeOrderID: exchangeOrderID,
		ExchangeTradeID: tradeID,
		Symbol:          market.Normalize(symbol),
		Side:            side,
		Price:           price,
		Quantity:        quantity,
		QuoteQuantity:   price * quantity,
		Commission:      fee,
		CommissionAsset: "USDT",
		RealizedPnL:     0, // Will be calculated for close orders
		IsMaker:         false,
		CreatedAt:       time.Now().UTC().UnixMilli(),
	}

	// Calculate realized PnL for close orders
	if action == "close_long" || action == "close_short" {
		// Try to get the entry price from the open position
		var positionSide string
		if action == "close_long" {
			positionSide = "LONG"
		} else {
			positionSide = "SHORT"
		}

		if openPos, err := e.store.Position().GetOpenPositionBySymbol(e.traderID, symbol, positionSide); err == nil && openPos != nil {
			if positionSide == "LONG" {
				fill.RealizedPnL = (price - openPos.EntryPrice) * quantity
			} else {
				fill.RealizedPnL = (openPos.EntryPrice - price) * quantity
			}
		}
	}

	e.store.Order().CreateFill(fill)
}

func (e *ChaosExecutor) recordPositionChange(orderID, symbol, side, action string, quantity, price float64, leverage int, entryPrice float64, fee float64) {
	if e.store == nil {
		return
	}

	switch action {
	case "open_long", "open_short":
		nowMs := time.Now().UTC().UnixMilli()
		pos := &store.TraderPosition{
			TraderID:           e.traderID,
			ExchangeID:         e.exchangeID,
			ExchangeType:       e.exchange,
			ExchangePositionID: e.exchangeID + "_" + symbol + "_" + side,
			Symbol:             symbol,
			Side:               side,
			Quantity:           quantity,
			EntryPrice:         price,
			EntryOrderID:       orderID,
			EntryTime:          nowMs,
			Leverage:           leverage,
			Status:             "OPEN",
			CreatedAt:          nowMs,
			UpdatedAt:          nowMs,
		}
		if err := e.store.Position().CreateOpenPosition(pos); err != nil {
			logger.Infof("  ⚠️ [Chaos] Failed to record position: %v", err)
		} else {
			logger.Infof("  📊 [Chaos] Position recorded [%s] %s %s @ %.4f", e.traderID[:8], symbol, side, price)
		}

	case "close_long", "close_short":
		posBuilder := store.NewPositionBuilder(e.store.Position())
		if err := posBuilder.ProcessTrade(
			e.traderID, e.exchangeID, e.exchange,
			symbol, side, action,
			quantity, price, fee, 0,
			time.Now().UTC().UnixMilli(), orderID,
		); err != nil {
			logger.Infof("  ⚠️ [Chaos] Failed to process close position: %v", err)
		} else {
			logger.Infof("  ✅ [Chaos] Position closed [%s] %s %s @ %.4f", e.traderID[:8], symbol, side, price)
		}
	}
}
