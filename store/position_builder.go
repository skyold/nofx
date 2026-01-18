package store

import (
	"fmt"
	"math"
	"nofx/logger"
	"sort"
	"strings"
	"time"
)

// PositionBuilder handles position creation and updates with support for:
// - Position averaging (merging multiple opens)
// - Partial closes (reducing quantity)
// - FIFO matching
// - Time-ordered processing
type PositionBuilder struct {
	positionStore *PositionStore
}

// NewPositionBuilder creates a new PositionBuilder
func NewPositionBuilder(positionStore *PositionStore) *PositionBuilder {
	return &PositionBuilder{
		positionStore: positionStore,
	}
}

// ProcessTrade processes a single trade and updates position accordingly
// tradeTimeMs is Unix milliseconds UTC
func (pb *PositionBuilder) ProcessTrade(
	traderID, exchangeID, exchangeType, symbol, side, action string,
	quantity, price, fee, realizedPnL float64,
	tradeTimeMs int64,
	orderID string,
) error {
	if strings.HasPrefix(action, "open_") {
		return pb.handleOpen(traderID, exchangeID, exchangeType, symbol, side, quantity, price, fee, tradeTimeMs, orderID)
	} else if strings.HasPrefix(action, "close_") {
		return pb.handleClose(traderID, exchangeID, exchangeType, symbol, side, quantity, price, fee, realizedPnL, tradeTimeMs, orderID)
	}
	return nil
}

// handleOpen handles opening positions (create new or average into existing)
// tradeTimeMs is Unix milliseconds UTC
func (pb *PositionBuilder) handleOpen(
	traderID, exchangeID, exchangeType, symbol, side string,
	quantity, price, fee float64,
	tradeTimeMs int64,
	orderID string,
) error {
	// Get existing OPEN position for (symbol, side)
	// Normalize side to ensure consistency
	existing, err := pb.positionStore.GetOpenPositionBySymbol(traderID, symbol, strings.ToUpper(side))
	if err != nil {
		return fmt.Errorf("failed to get open position: %w", err)
	}

	nowMs := time.Now().UTC().UnixMilli()
	if existing == nil {
		// Create new position
		position := &TraderPosition{
			TraderID:           traderID,
			ExchangeID:         exchangeID,
			ExchangeType:       exchangeType,
			ExchangePositionID: fmt.Sprintf("sync_%s_%s_%d", symbol, side, tradeTimeMs),
			Symbol:             symbol,
			Side:               side,
			Quantity:           quantity,
			EntryPrice:         price,
			EntryOrderID:       orderID,
			EntryTime:          UnixTime(tradeTimeMs),
			Leverage:           1,
			Status:             "OPEN",
			Source:             "sync",
			Fee:                fee,
			CreatedAt:          UnixTime(nowMs),
			UpdatedAt:          UnixTime(nowMs),
		}
		return pb.positionStore.CreateOpenPosition(position)
	}

	// Merge: Calculate weighted average entry price and update position
	logger.Infof("  📊 Averaging position: %s %s %.6f @ %.2f + %.6f @ %.2f",
		symbol, side, existing.Quantity, existing.EntryPrice, quantity, price)

	// Also update exchange_id and exchange_type if they were empty
	if existing.ExchangeID == "" || existing.ExchangeType == "" {
		if err := pb.positionStore.UpdatePositionExchangeInfo(existing.ID, exchangeID, exchangeType); err != nil {
			logger.Infof("  ⚠️  Failed to update exchange info: %v", err)
		}
	}

	return pb.positionStore.UpdatePositionQuantityAndPrice(existing.ID, quantity, price, fee)
}

// handleClose handles closing positions (partial or full)
// tradeTimeMs is Unix milliseconds UTC
func (pb *PositionBuilder) handleClose(
	traderID, exchangeID, exchangeType, symbol, side string,
	quantity, price, fee, realizedPnL float64,
	tradeTimeMs int64,
	orderID string,
) error {
	// Get OPEN position
	// Normalize side to ensure consistency
	position, err := pb.positionStore.GetOpenPositionBySymbol(traderID, symbol, strings.ToUpper(side))
	if err != nil {
		return fmt.Errorf("failed to get open position: %w", err)
	}

	if position == nil {
		// No open position found - just skip
		// This can happen if trades are processed out of order or database was cleared
		logger.Infof("  ⚠️  No matching open position for %s %s (orderID: %s), skipping", symbol, side, orderID)
		return nil
	}

	const QUANTITY_TOLERANCE = 0.0001

	// Calculate realized PnL if not provided (some exchanges like Lighter don't return it)
	if realizedPnL == 0 && position.EntryPrice > 0 {
		if side == "LONG" {
			realizedPnL = (price - position.EntryPrice) * quantity
		} else {
			realizedPnL = (position.EntryPrice - price) * quantity
		}
		// Round to 2 decimal places
		realizedPnL = math.Round(realizedPnL*100) / 100
	}

	if quantity < position.Quantity-QUANTITY_TOLERANCE {
		// Partial close: reduce quantity and update weighted average exit price
		logger.Infof("  📉 Partial close: %s %s %.6f → %.6f (closed %.6f @ %.2f, PnL: %.2f)",
			symbol, side, position.Quantity, position.Quantity-quantity, quantity, price, realizedPnL)
		return pb.positionStore.ReducePositionQuantity(position.ID, quantity, price, fee, realizedPnL)
	} else {
		// Full close (or close with tolerance): mark as CLOSED
		closeQty := quantity
		if quantity > position.Quantity {
			logger.Infof("  ⚠️  Over-close detected: %s %s trying to close %.6f but only %.6f open, closing full position",
				symbol, side, quantity, position.Quantity)
			closeQty = position.Quantity
		}

		// Calculate final weighted average exit price
		// Include previously accumulated partial close prices + this final close
		closedBefore := position.EntryQuantity - position.Quantity
		totalClosed := closedBefore + closeQty
		var finalExitPrice float64
		if totalClosed > 0 {
			finalExitPrice = (position.ExitPrice*closedBefore + price*closeQty) / totalClosed
			finalExitPrice = math.Round(finalExitPrice*100) / 100
		} else {
			finalExitPrice = price
		}

		// Calculate total PnL (existing + new)
		totalPnL := position.RealizedPnL + realizedPnL

		// Calculate total fee (existing + new)
		totalFee := position.Fee + fee

		logger.Infof("  ✅ Full close: %s %s %.6f @ %.2f (avg exit: %.2f, entry: %.2f, PnL: %.2f)",
			symbol, side, closeQty, price, finalExitPrice, position.EntryPrice, totalPnL)

		return pb.positionStore.ClosePositionFully(
			position.ID,
			finalExitPrice,
			orderID,
			tradeTimeMs,
			totalPnL,
			totalFee,
			"sync",
		)
	}
}

// quantitiesMatch checks if two quantities are close enough (within tolerance)
func quantitiesMatch(a, b float64) bool {
	const QUANTITY_TOLERANCE = 0.0001
	return math.Abs(a-b) < QUANTITY_TOLERANCE
}

// RebuildFromOrders rebuilds positions from orders for a specific trader
func (pb *PositionBuilder) RebuildFromOrders(traderID string, orderStore *OrderStore) (int, error) {
	// 1. Get all filled orders
	var orders []TraderOrder
	if err := pb.positionStore.db.Where("trader_id = ? AND status = ?", traderID, "FILLED").Order("created_at ASC").Find(&orders).Error; err != nil {
		return 0, fmt.Errorf("failed to fetch orders: %w", err)
	}

	logger.Infof("Found %d filled orders for trader %s. Processing...", len(orders), traderID)

	// Group orders by Symbol + PositionSide
	// Key: symbol_positionSide
	orderGroups := make(map[string][]TraderOrder)
	for _, order := range orders {
		// Normalize
		symbol := strings.ToUpper(order.Symbol)
		// Try to determine PositionSide if missing
		posSide := strings.ToUpper(order.PositionSide)
		if posSide == "" {
			if strings.Contains(order.OrderAction, "long") {
				posSide = "LONG"
			} else if strings.Contains(order.OrderAction, "short") {
				posSide = "SHORT"
			} else {
				// Fallback based on side
				if order.Side == "BUY" {
					posSide = "LONG" // Assume long for buy
				} else {
					posSide = "SHORT" // Assume short for sell
				}
			}
		}

		key := fmt.Sprintf("%s_%s", symbol, posSide)
		orderGroups[key] = append(orderGroups[key], order)
	}

	rebuiltCount := 0

	// Process each group
	for key, groupOrders := range orderGroups {
		parts := strings.Split(key, "_")
		if len(parts) < 2 {
			continue
		}
		symbol := parts[0]
		side := parts[1]

		// Sort orders by time
		sort.Slice(groupOrders, func(i, j int) bool {
			return groupOrders[i].CreatedAt < groupOrders[j].CreatedAt
		})

		// Track open positions in memory
		// We use a simple FIFO queue for matching open/close
		type OpenPos struct {
			Order        TraderOrder
			RemainingQty float64
		}
		var openPositions []OpenPos

		for _, order := range groupOrders {
			action := strings.ToLower(order.OrderAction)
			isOpen := strings.Contains(action, "open") ||
				(side == "LONG" && order.Side == "BUY") ||
				(side == "SHORT" && order.Side == "SELL")

			if isOpen {
				// Add to open queue
				openPositions = append(openPositions, OpenPos{
					Order:        order,
					RemainingQty: order.FilledQuantity,
				})
			} else {
				// Close order: match with open positions
				closeQty := order.FilledQuantity

				for closeQty > 0 && len(openPositions) > 0 {
					openPos := &openPositions[0]
					matchQty := math.Min(closeQty, openPos.RemainingQty)

					// Calculate PnL
					var pnl float64
					if side == "LONG" {
						pnl = (order.AvgFillPrice - openPos.Order.AvgFillPrice) * matchQty
					} else {
						pnl = (openPos.Order.AvgFillPrice - order.AvgFillPrice) * matchQty
					}

					// Check if this position record exists in DB
					// We check by rough timestamp match (since we don't have exact link)
					var existingCount int64
					pb.positionStore.db.Model(&TraderPosition{}).Where(
						"trader_id = ? AND symbol = ? AND side = ? AND status = ? AND ABS(entry_time - ?) < 5000 AND ABS(exit_time - ?) < 5000",
						traderID, symbol, side, "CLOSED", int64(openPos.Order.CreatedAt), int64(order.CreatedAt),
					).Count(&existingCount)

					if existingCount == 0 {
						// Not found! Create it.
						logger.Infof("  [REBUILD] Missing position found: %s %s (Open: %d, Close: %d)",
							symbol, side, openPos.Order.ID, order.ID)

						newPos := &TraderPosition{
							TraderID:           traderID,
							ExchangeID:         order.ExchangeID,
							ExchangeType:       order.ExchangeType,
							ExchangePositionID: fmt.Sprintf("rebuild_%d_%d", openPos.Order.ID, order.ID),
							Symbol:             symbol,
							Side:               side,
							Quantity:           matchQty,
							EntryPrice:         openPos.Order.AvgFillPrice,
							EntryQuantity:      matchQty,
							EntryOrderID:       openPos.Order.ExchangeOrderID,
							EntryTime:          openPos.Order.CreatedAt,
							ExitPrice:          order.AvgFillPrice,
							ExitOrderID:        order.ExchangeOrderID,
							ExitTime:           order.CreatedAt,
							RealizedPnL:        pnl,
							Fee:                order.Commission + openPos.Order.Commission*(matchQty/openPos.Order.FilledQuantity), // Pro-rate fee
							Leverage:           order.Leverage,
							Status:             "CLOSED",
							CloseReason:        "rebuild_manual",
							Source:             "rebuild",
							CreatedAt:          order.CreatedAt, // Use close time as creation time
							UpdatedAt:          UnixTime(time.Now().UnixMilli()),
						}

						if err := pb.positionStore.db.Create(newPos).Error; err != nil {
							logger.Errorf("    Error creating position: %v", err)
						} else {
							rebuiltCount++
						}
					}

					// Update remaining quantities
					closeQty -= matchQty
					openPos.RemainingQty -= matchQty

					if openPos.RemainingQty <= 0.00000001 {
						// Remove fully closed position from queue
						openPositions = openPositions[1:]
					}
				}
			}
		}
	}

	return rebuiltCount, nil
}
