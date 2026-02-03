package api

import (
	"fmt"
	"nofx/logger"
	"nofx/store"
	"nofx/trader"
	"time"
)

// recordClosePositionOrder Record close position order to database (Lighter version - direct FILLED status)
func (s *Server) recordClosePositionOrder(traderID, exchangeID, exchangeType, symbol, side string, quantity, exitPrice float64, result map[string]interface{}) {
	// Skip for exchanges with OrderSync - let the background sync handle it to avoid duplicates
	switch exchangeType {
	case "binance", "lighter", "hyperliquid", "bybit", "okx", "bitget", "aster", "gate":
		logger.Infof("  📝 Close order will be synced by OrderSync, skipping immediate record")
		return
	}

	// Check if order was placed (skip if NO_POSITION)
	status, _ := result["status"].(string)
	if status == "NO_POSITION" {
		logger.Infof("  ⚠️ No position to close, skipping order record")
		return
	}

	// Get order ID from result
	var orderID string
	switch v := result["orderId"].(type) {
	case int64:
		orderID = fmt.Sprintf("%d", v)
	case float64:
		orderID = fmt.Sprintf("%.0f", v)
	case string:
		orderID = v
	default:
		orderID = fmt.Sprintf("%v", v)
	}

	if orderID == "" || orderID == "0" {
		logger.Infof("  ⚠️ Order ID is empty, skipping record")
		return
	}

	// Determine order action based on side
	var orderAction string
	if side == "LONG" {
		orderAction = "close_long"
	} else {
		orderAction = "close_short"
	}

	// Use entry price if exit price not available
	if exitPrice == 0 {
		exitPrice = quantity * 100 // Rough estimate if we don't have price
	}

	// Estimate fee (0.04% for Lighter taker)
	fee := exitPrice * quantity * 0.0004

	// Create order record - DIRECTLY as FILLED (Lighter market orders fill immediately)
	orderRecord := &store.TraderOrder{
		TraderID:        traderID,
		ExchangeID:      exchangeID,
		ExchangeType:    exchangeType,
		ExchangeOrderID: orderID,
		Symbol:          symbol,
		PositionSide:    side,
		OrderAction:     orderAction,
		Type:            "MARKET",
		Side:            getSideFromAction(orderAction),
		Quantity:        quantity,
		Price:           0, // Market order
		Status:          "FILLED",
		FilledQuantity:  quantity,
		AvgFillPrice:    exitPrice,
		Commission:      fee,
		FilledAt:        store.UnixTime(time.Now().UTC().UnixMilli()),
		CreatedAt:       store.UnixTime(time.Now().UTC().UnixMilli()),
		UpdatedAt:       store.UnixTime(time.Now().UTC().UnixMilli()),
	}

	if err := s.store.Order().CreateOrder(orderRecord); err != nil {
		logger.Infof("  ⚠️ Failed to record order: %v", err)
		return
	}

	logger.Infof("  ✅ Order recorded as FILLED: %s [%s] %s qty=%.6f price=%.6f", orderID, orderAction, symbol, quantity, exitPrice)

	// Create fill record immediately
	tradeID := fmt.Sprintf("%s-%d", orderID, time.Now().UnixNano())
	fillRecord := &store.TraderFill{
		TraderID:        traderID,
		ExchangeID:      exchangeID,
		ExchangeType:    exchangeType,
		OrderID:         orderRecord.ID,
		ExchangeOrderID: orderID,
		ExchangeTradeID: tradeID,
		Symbol:          symbol,
		Side:            getSideFromAction(orderAction),
		Price:           exitPrice,
		Quantity:        quantity,
		QuoteQuantity:   exitPrice * quantity,
		Commission:      fee,
		CommissionAsset: "USDT",
		RealizedPnL:     0,
		IsMaker:         false,
		CreatedAt:       store.UnixTime(time.Now().UTC().UnixMilli()),
	}

	if err := s.store.Order().CreateFill(fillRecord); err != nil {
		logger.Infof("  ⚠️ Failed to record fill: %v", err)
	} else {
		logger.Infof("  ✅ Fill record created: price=%.6f qty=%.6f", exitPrice, quantity)
	}
}

// pollAndUpdateOrderStatus Poll order status and update with fill data
func (s *Server) pollAndUpdateOrderStatus(orderRecordID int64, traderID, exchangeID, exchangeType, orderID, symbol, orderAction string, tempTrader trader.Trader) {
	var actualPrice float64
	var actualQty float64
	var fee float64

	// Wait a bit for order to be filled
	time.Sleep(500 * time.Millisecond)

	// For Lighter, use GetTrades instead of GetOrderStatus (market orders are filled immediately)
	if exchangeType == "lighter" {
		s.pollLighterTradeHistory(orderRecordID, traderID, exchangeID, exchangeType, orderID, symbol, orderAction, tempTrader)
		return
	}

	// For other exchanges, poll GetOrderStatus
	for i := 0; i < 5; i++ {
		status, err := tempTrader.GetOrderStatus(symbol, orderID)
		if err != nil {
			logger.Infof("  ⚠️ GetOrderStatus failed (attempt %d/5): %v", i+1, err)
			time.Sleep(500 * time.Millisecond)
			continue
		}
		if err == nil {
			statusStr, _ := status["status"].(string)
			if statusStr == "FILLED" {
				// Get actual fill price
				if avgPrice, ok := status["avgPrice"].(float64); ok && avgPrice > 0 {
					actualPrice = avgPrice
				}
				// Get actual executed quantity
				if execQty, ok := status["executedQty"].(float64); ok && execQty > 0 {
					actualQty = execQty
				}
				// Get commission/fee
				if commission, ok := status["commission"].(float64); ok {
					fee = commission
				}

				logger.Infof("  ✅ Order filled: avgPrice=%.6f, qty=%.6f, fee=%.6f", actualPrice, actualQty, fee)

				// Update order status to FILLED
				if err := s.store.Order().UpdateOrderStatus(orderRecordID, "FILLED", actualQty, actualPrice, fee); err != nil {
					logger.Infof("  ⚠️ Failed to update order status: %v", err)
					return
				}

				// Record fill details
				tradeID := fmt.Sprintf("%s-%d", orderID, time.Now().UnixNano())
				fillRecord := &store.TraderFill{
					TraderID:        traderID,
					ExchangeID:      exchangeID,
					ExchangeType:    exchangeType,
					OrderID:         orderRecordID,
					ExchangeOrderID: orderID,
					ExchangeTradeID: tradeID,
					Symbol:          symbol,
					Side:            getSideFromAction(orderAction),
					Price:           actualPrice,
					Quantity:        actualQty,
					QuoteQuantity:   actualPrice * actualQty,
					Commission:      fee,
					CommissionAsset: "USDT",
					RealizedPnL:     0,
					IsMaker:         false,
					CreatedAt:       store.UnixTime(time.Now().UTC().UnixMilli()),
				}

				if err := s.store.Order().CreateFill(fillRecord); err != nil {
					logger.Infof("  ⚠️ Failed to record fill: %v", err)
				} else {
					logger.Infof("  📝 Fill recorded: price=%.6f, qty=%.6f", actualPrice, actualQty)
				}

				return
			} else if statusStr == "CANCELED" || statusStr == "EXPIRED" || statusStr == "REJECTED" {
				logger.Infof("  ⚠️ Order %s, updating status", statusStr)
				s.store.Order().UpdateOrderStatus(orderRecordID, statusStr, 0, 0, 0)
				return
			}
		}
		time.Sleep(500 * time.Millisecond)
	}

	logger.Infof("  ⚠️ Failed to confirm order fill after polling, order may still be pending")
}

// pollLighterTradeHistory No longer used - Lighter orders are marked as FILLED immediately
// Keeping this function stub for compatibility with other exchanges
func (s *Server) pollLighterTradeHistory(orderRecordID int64, traderID, exchangeID, exchangeType, orderID, symbol, orderAction string, tempTrader trader.Trader) {
	// For Lighter, orders are now recorded as FILLED immediately in recordClosePositionOrder
	// This function is no longer called for Lighter exchange
	logger.Infof("  ℹ️ pollLighterTradeHistory called but not needed (order already marked FILLED)")
}

// getSideFromAction Get order side (BUY/SELL) from order action
func getSideFromAction(action string) string {
	switch action {
	case "open_long", "close_short":
		return "BUY"
	case "open_short", "close_long":
		return "SELL"
	default:
		return "BUY"
	}
}
