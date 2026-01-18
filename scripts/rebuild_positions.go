package main

import (
	"fmt"
	"log"
	"math"
	"nofx/config"
	"nofx/store"
	"os"
	"sort"
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
		CreatedAt       interface{} // Handle both int64 and string
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
		case int64:
			createdAt = v
		case float64:
			createdAt = int64(v)
		case string:
			// Try parsing RFC3339
			t, err := time.Parse(time.RFC3339, v)
			if err == nil {
				createdAt = t.UnixMilli()
			} else {
				// Try other formats or default to now
				createdAt = time.Now().UnixMilli()
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
		parts := strings.Split(key, "_")
		if len(parts) < 3 {
			continue
		}
		traderID := parts[0]
		symbol := parts[1]
		side := parts[2]

		// Sort orders by time
		sort.Slice(groupOrders, func(i, j int) bool {
			return groupOrders[i].CreatedAt < groupOrders[j].CreatedAt
		})

		// Track open positions in memory
		// We use a simple FIFO queue for matching open/close
		type OpenPos struct {
			Order store.TraderOrder
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
					Order: order,
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
