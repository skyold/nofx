package main

import (
	"fmt"
	"log"
	"math"
	"nofx/config"
	"nofx/store"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

// Rebuild positions from orders
func main() {
	// Initialize config
	config.Init()
	// cfg := config.Get()

	// Connect to database
	dbPath := "data/data.db"
	if len(os.Args) > 1 {
		dbPath = os.Args[1]
	}

	fmt.Printf("Opening database: %s\n", dbPath)
	db, err := gorm.Open(sqlite.Open(dbPath), &gorm.Config{
		Logger: logger.Default.LogMode(logger.Info),
	})
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	// 1. Get all filled orders
	// Use a raw struct to handle potential time format mismatches
	type RawOrder struct {
		ID              int64
		TraderID        string
		ExchangeID      string
		ExchangeType    string
		ExchangeOrderID string
		Symbol          string
		Side            string
		PositionSide    string
		Type            string
		Status          string
		OrderAction     string
		Quantity        float64
		FilledQuantity  float64
		Price           float64
		AvgFillPrice    float64
		Commission      float64
		Leverage        int
		CreatedAt       interface{} `gorm:"column:created_at"`
	}

	var rawOrders []RawOrder
	if err := db.Table("trader_orders").Where("status = ?", "FILLED").Order("created_at ASC").Find(&rawOrders).Error; err != nil {
		log.Fatalf("Failed to fetch orders: %v", err)
	}

	fmt.Printf("Found %d filled orders. Processing...\n", len(rawOrders))

	// Group orders by TraderID + Symbol + PositionSide
	// Key: traderID_symbol_positionSide
	orderGroups := make(map[string][]store.TraderOrder)
	for _, raw := range rawOrders {
		// Convert RawOrder to store.TraderOrder
		var createdAt int64
		switch v := raw.CreatedAt.(type) {
		case time.Time:
			createdAt = v.UnixMilli()
		case int64:
			createdAt = v
		case float64:
			createdAt = int64(v)
		case []uint8:
			// Bytes, usually string
			s := string(v)
			if t, err := time.Parse(time.RFC3339, s); err == nil {
				createdAt = t.UnixMilli()
			} else if i, err := strconv.ParseInt(s, 10, 64); err == nil {
				createdAt = i
			} else {
				// Try other formats
				if t, err := time.Parse("2006-01-02 15:04:05.999-07:00", s); err == nil {
					createdAt = t.UnixMilli()
				} else {
					createdAt = time.Now().UnixMilli()
				}
			}
		case string:
			// Try numeric string first
			if i, err := strconv.ParseInt(v, 10, 64); err == nil {
				createdAt = i
			} else {
				// Try parsing various time formats
				formats := []string{
					time.RFC3339,
					"2006-01-02 15:04:05.999-07:00",
					"2006-01-02 15:04:05-07:00",
					"2006-01-02 15:04:05",
				}
				parsed := false
				for _, f := range formats {
					if t, err := time.Parse(f, v); err == nil {
						createdAt = t.UnixMilli()
						parsed = true
						break
					}
				}
				if !parsed {
					// Fallback to now if all fail
					fmt.Printf("Warning: Failed to parse time '%v', using Now\n", v)
					createdAt = time.Now().UnixMilli()
				}
			}
		default:
			createdAt = time.Now().UnixMilli()
		}

		order := store.TraderOrder{
			ID:              raw.ID,
			TraderID:        raw.TraderID,
			ExchangeID:      raw.ExchangeID,
			ExchangeType:    raw.ExchangeType,
			ExchangeOrderID: raw.ExchangeOrderID,
			Symbol:          raw.Symbol,
			Side:            raw.Side,
			PositionSide:    raw.PositionSide,
			Type:            raw.Type,
			Status:          raw.Status,
			OrderAction:     raw.OrderAction,
			Quantity:        raw.Quantity,
			FilledQuantity:  raw.FilledQuantity,
			Price:           raw.Price,
			AvgFillPrice:    raw.AvgFillPrice,
			Commission:      raw.Commission,
			Leverage:        raw.Leverage,
			CreatedAt:       createdAt,
		}

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

		key := fmt.Sprintf("%s_%s_%s", order.TraderID, symbol, posSide)
		orderGroups[key] = append(orderGroups[key], order)
	}

	rebuiltCount := 0

	// Process each group
	for key, groupOrders := range orderGroups {
		if len(groupOrders) == 0 {
			continue
		}

		// Get traderID directly from the first order (safer than parsing key)
		traderID := groupOrders[0].TraderID

		// Parse symbol and side from key parts (these don't contain underscores usually, but better to use order data too)
		// Actually, let's just use order data for everything to be safe
		symbol := groupOrders[0].Symbol

		// Determine side from the group logic (since we grouped by it)
		parts := strings.Split(key, "_")
		// The last part is the PositionSide
		side := parts[len(parts)-1]

		// Override symbol from parts if needed, but order.Symbol is better.
		// However, we grouped by Uppercase Symbol.
		// Let's rely on the order data, but we need the SIDE that was used for grouping.
		// The key format is: fmt.Sprintf("%s_%s_%s", order.TraderID, symbol, posSide)
		// Since TraderID can contain underscores, we can't easily split from the front.
		// But we know the LAST part is the Side.
		// And we know the TraderID.
		// The symbol is everything in between?
		// Actually, we have the orders! We can just use `groupOrders[0].Symbol` and `groupOrders[0].PositionSide` (or inferred side).

		// Re-infer side logic to be consistent with grouping
		// We can just use the 'side' variable extracted from the last part of the key
		// Or better, since we have the orders, just re-evaluate the common side?
		// No, the group might contain mixed BUY/SELL orders, so we need the grouping key's side context.
		// The key was constructed as: key := fmt.Sprintf("%s_%s_%s", order.TraderID, symbol, posSide)

		// Let's just trust the orders in the group. They are all for the same TraderID, Symbol, and PositionSide.
		// So we can take them from the first order.
		// Exception: PositionSide might be inferred in the loop above.
		// Let's grab it from the order if possible, or re-infer it.
		// But wait, in the grouping loop:
		/*
			posSide := strings.ToUpper(order.PositionSide)
			if posSide == "" { ... inferred ... }
			key := fmt.Sprintf("%s_%s_%s", order.TraderID, symbol, posSide)
		*/
		// So all orders in this group share the same (inferred) posSide.
		// We can't easily get the *inferred* posSide from the order struct again without repeating logic.
		// But we can extract it from the key safely if we know TraderID and Symbol?
		// TraderID might have underscores. Symbol might have underscores (unlikely for pairs like BNBUSDT).

		// Safer approach: Extract side from the *last* component of the key.
		// Then Symbol is everything between TraderID and Side?
		// No, that's messy.

		// Simplest Fix:
		// We already have the list of orders `groupOrders`.
		// We can just use `groupOrders[0]` to get TraderID and Symbol.
		// For `side`, we can take `parts[len(parts)-1]`.

		// Wait, `parts` was `strings.Split(key, "_")`.
		// If key is `ID_SYMBOL_SIDE`, and ID has `_`, then parts is `[ID, part2, ..., SYMBOL, SIDE]`.
		// So `side` is indeed `parts[len(parts)-1]`.
		// `symbol` is `groupOrders[0].Symbol` (normalized to upper case in loop, but order struct has original).
		// Let's use `strings.ToUpper(groupOrders[0].Symbol)`.

		// Fix the variable assignment:
		// traderID := parts[0]  <-- DELETE THIS
		// symbol := parts[1]    <-- DELETE THIS
		// side := parts[2]      <-- DELETE THIS

		// New logic:

		// Sort orders by time
		sort.Slice(groupOrders, func(i, j int) bool {
			return groupOrders[i].CreatedAt < groupOrders[j].CreatedAt
		})

		// Track open positions in memory
		// We use a simple FIFO queue for matching open/close
		type OpenPos struct {
			Order        store.TraderOrder
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
					db.Model(&store.TraderPosition{}).Where(
						"trader_id = ? AND symbol = ? AND side = ? AND status = ? AND ABS(entry_time - ?) < 5000 AND ABS(exit_time - ?) < 5000",
						traderID, symbol, side, "CLOSED", openPos.Order.CreatedAt, order.CreatedAt,
					).Count(&existingCount)

					if existingCount == 0 {
						// Not found! Create it.
						fmt.Printf("  [REBUILD] Missing position found: %s %s (Open: %d, Close: %d)\n",
							symbol, side, openPos.Order.ID, order.ID)

						newPos := &store.TraderPosition{
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
							CloseReason:        "rebuild_script",
							Source:             "rebuild",
							CreatedAt:          order.CreatedAt, // Use close time as creation time
							UpdatedAt:          time.Now().UnixMilli(),
						}

						if err := db.Create(newPos).Error; err != nil {
							fmt.Printf("    Error creating position: %v\n", err)
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

	fmt.Printf("Done! Rebuilt %d missing positions.\n", rebuiltCount)
}
