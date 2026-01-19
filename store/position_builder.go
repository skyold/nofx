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

// RebuildFromFills rebuilds positions from fills (trades) for a specific trader
// This is more accurate than orders because it uses actual execution data
func (pb *PositionBuilder) RebuildFromFills(traderID string, orderStore *OrderStore) (int, error) {
	// 1. Get all fills
	var fills []TraderFill
	if err := pb.positionStore.db.Where("trader_id = ?", traderID).Order("created_at ASC").Find(&fills).Error; err != nil {
		return 0, fmt.Errorf("failed to fetch fills: %w", err)
	}

	logger.Infof("Found %d fills for trader %s. Processing...", len(fills), traderID)

	// 2. Delete existing CLOSED positions for this trader to avoid duplicates
	// We only delete CLOSED positions because OPEN positions are managed by live trading
	if err := pb.positionStore.db.Where("trader_id = ? AND status = ?", traderID, "CLOSED").Delete(&TraderPosition{}).Error; err != nil {
		return 0, fmt.Errorf("failed to delete existing closed positions: %w", err)
	}

	// 3. Group fills by Symbol + PositionSide
	// Since TraderFill doesn't always have PositionSide (it might be inferred), we need to be careful.
	// However, for most exchanges we support, we can infer it or it's present in Order if we joined.
	// For simplicity and robustness, we'll infer based on Side and "Open/Close" intent if available,
	// or try to match purely FIFO per symbol if we assume one-way mode or can detect hedge mode.
	// To strictly follow "transaction history", we should probably use the same logic as the unified algorithm.

	// Group by Symbol
	fillGroups := make(map[string][]TraderFill)
	for _, fill := range fills {
		fillGroups[fill.Symbol] = append(fillGroups[fill.Symbol], fill)
	}

	rebuiltCount := 0

	// Process each symbol
	for symbol, groupFills := range fillGroups {
		// Sort by time
		sort.Slice(groupFills, func(i, j int) bool {
			return groupFills[i].CreatedAt < groupFills[j].CreatedAt
		})

		// Track open positions (long and short separately)
		// Key: "LONG" or "SHORT"
		type OpenPos struct {
			Fill         TraderFill
			RemainingQty float64
		}
		openPositions := make(map[string][]OpenPos)
		openPositions["LONG"] = []OpenPos{}
		openPositions["SHORT"] = []OpenPos{}

		for _, fill := range groupFills {
			// Determine side and action
			// We need to know if this fill is OPENING or CLOSING.
			// In many cases, realized_pnl != 0 implies closing.
			// But for opening trades, realized_pnl is usually 0.
			// Side: BUY or SELL.

			isBuy := strings.EqualFold(fill.Side, "BUY")
			// isSell := strings.EqualFold(fill.Side, "SELL")

			// Try to determine Position Side (LONG/SHORT)
			// If we have access to the Order, we could check PositionSide.
			// But here we only have Fill.
			// Heuristic:
			// 1. If RealizedPnL != 0, it's a CLOSE.
			//    If BUY & PnL!=0 -> Closing SHORT.
			//    If SELL & PnL!=0 -> Closing LONG.
			// 2. If RealizedPnL == 0, it's likely OPEN (or closing a losing trade with 0 PnL? Unlikely exactly 0).
			//    Actually, PnL is computed by exchange. If it's 0, it's usually Open.
			//    If BUY & PnL==0 -> Opening LONG.
			//    If SELL & PnL==0 -> Opening SHORT.

			// Note: This heuristic works for One-Way mode and Hedge Mode if strictly separated.
			// But in Hedge Mode, you can Open Long (Buy) and Close Short (Buy).
			// If PnL is reliable, we use it.

			var positionSide string
			var isOpen bool

			// Check if we can rely on RealizedPnL
			// Most exchanges (Binance, Bybit) provide PnL on close.
			if fill.RealizedPnL != 0 {
				// Definitely closing
				isOpen = false
				if isBuy {
					positionSide = "SHORT" // Buy to close Short
				} else {
					positionSide = "LONG" // Sell to close Long
				}
			} else {
				// PnL is 0. Likely Opening.
				// Exception: Closing a trade at breakeven.
				// To handle this, we can look at current open positions.
				// If we have open SHORTs and we BUY, is it opening LONG or closing SHORT?
				// Without explicit "ReduceOnly" or "PositionSide" flag, it's ambiguous.
				// However, the unified algorithm (trader/position_rebuild.go) assumes PnL!=0 means close.
				// Let's stick to that for now as it's the "standard" we want to sync with.
				isOpen = true
				if isBuy {
					positionSide = "LONG"
				} else {
					positionSide = "SHORT"
				}
			}

			if isOpen {
				// Add to open queue
				openPositions[positionSide] = append(openPositions[positionSide], OpenPos{
					Fill:         fill,
					RemainingQty: fill.Quantity,
				})
			} else {
				// Close logic
				closeQty := fill.Quantity
				queue := openPositions[positionSide]

				matchedQtyTotal := 0.0
				var avgEntryPrice float64
				var firstEntryTime UnixTime
				var totalEntryFee float64

				for closeQty > 0.00000001 && len(queue) > 0 {
					openPos := &queue[0]
					matchQty := math.Min(closeQty, openPos.RemainingQty)

					// Accumulate weighted entry price
					avgEntryPrice += openPos.Fill.Price * matchQty
					totalEntryFee += openPos.Fill.Commission * (matchQty / openPos.Fill.Quantity)
					if matchedQtyTotal == 0 {
						firstEntryTime = openPos.Fill.CreatedAt
					}

					matchedQtyTotal += matchQty
					closeQty -= matchQty
					openPos.RemainingQty -= matchQty

					if openPos.RemainingQty <= 0.00000001 {
						queue = queue[1:]
					}
				}

				// Update the queue in map
				openPositions[positionSide] = queue

				// If we matched something, create a closed position record
				if matchedQtyTotal > 0.00000001 {
					avgEntryPrice /= matchedQtyTotal

					// Create Position Record
					newPos := &TraderPosition{
						TraderID:           traderID,
						ExchangeID:         fill.ExchangeID,
						ExchangeType:       fill.ExchangeType,
						ExchangePositionID: fmt.Sprintf("rebuild_%s_%d", fill.ExchangeTradeID, time.Now().UnixNano()), // Unique ID
						Symbol:             symbol,
						Side:               positionSide,
						Quantity:           matchedQtyTotal,
						EntryPrice:         avgEntryPrice,
						EntryQuantity:      matchedQtyTotal,
						EntryOrderID:       "", // Unknown if multiple
						EntryTime:          firstEntryTime,
						ExitPrice:          fill.Price,
						ExitOrderID:        fill.ExchangeOrderID,
						ExitTime:           fill.CreatedAt,
						RealizedPnL:        fill.RealizedPnL * (matchedQtyTotal / fill.Quantity), // Pro-rate PnL if partial match (though usually 1:1)
						Fee:                fill.Commission*(matchedQtyTotal/fill.Quantity) + totalEntryFee,
						Leverage:           1, // Unknown from fills usually
						Status:             "CLOSED",
						CloseReason:        "rebuild_manual",
						Source:             "rebuild",
						CreatedAt:          fill.CreatedAt,
						UpdatedAt:          UnixTime(time.Now().UTC().UnixMilli()),
					}

					if err := pb.positionStore.db.Create(newPos).Error; err != nil {
						logger.Errorf("    Error creating position: %v", err)
					} else {
						rebuiltCount++
					}
				}
			}
		}
	}

	return rebuiltCount, nil
}
