package virtual

import (
	"fmt"
	"nofx/logger"
	"nofx/market"
	"nofx/store"
	"sort"
	"strings"
	"time"
)

// SyncOrdersFromVirtual syncs Virtual exchange order history to local database
// Also creates/updates position records to ensure orders/fills/positions data consistency
func (t *VirtualTrader) SyncOrdersFromVirtual(traderID string, exchangeID string, exchangeType string, st *store.Store) error {
	if st == nil {
		return fmt.Errorf("store is nil")
	}

	// Get recent trades (last 24 hours)
	// For virtual exchange, we can just sync last 24h as it's fast
	startTime := time.Now().Add(-24 * time.Hour)

	// Use GetTrades method to fetch trade records
	trades, err := t.GetTrades(startTime, 1000) // Higher limit for virtual
	if err != nil {
		return fmt.Errorf("failed to get trades: %w", err)
	}

	// Sort trades by time ASC (oldest first) for proper position building
	sort.Slice(trades, func(i, j int) bool {
		return trades[i].Time.UnixMilli() < trades[j].Time.UnixMilli()
	})

	// Process trades one by one
	orderStore := st.Order()
	positionStore := st.Position()
	posBuilder := store.NewPositionBuilder(positionStore)
	syncedCount := 0
	exchangePositions, err := t.GetPositions()
	if err != nil {
		logger.Infof("⚠️  Failed to get virtual positions for normalization: %v", err)
	} else {
		realPositions := make(map[string]struct {
			Quantity   float64
			EntryPrice float64
		})
		for _, pos := range exchangePositions {
			symbol, _ := pos["symbol"].(string)
			side, _ := pos["side"].(string)
			qty, _ := pos["positionAmt"].(float64)
			entryPrice, _ := pos["entryPrice"].(float64)
			if symbol == "" || qty == 0 {
				continue
			}
			if qty < 0 {
				qty = -qty
				if side == "" {
					side = "short"
				}
			}
			key := fmt.Sprintf("%s_%s", market.Normalize(symbol), strings.ToLower(side))
			realPositions[key] = struct {
				Quantity   float64
				EntryPrice float64
			}{
				Quantity:   qty,
				EntryPrice: entryPrice,
			}
		}
		if localPositions, err := positionStore.GetOpenPositions(traderID); err == nil {
			for _, localPos := range localPositions {
				if localPos.ExchangeID != exchangeID {
					continue
				}
				sideUpper := strings.ToUpper(localPos.Side)
				if sideUpper != "BUY" && sideUpper != "SELL" {
					continue
				}
				mappedSide := "LONG"
				if sideUpper == "SELL" {
					mappedSide = "SHORT"
				}
				key := fmt.Sprintf("%s_%s", market.Normalize(localPos.Symbol), strings.ToLower(mappedSide))
				nowMs := time.Now().UTC().UnixMilli()
				if realPos, ok := realPositions[key]; ok {
					if correctPos, _ := positionStore.GetOpenPositionBySymbol(traderID, localPos.Symbol, mappedSide); correctPos == nil {
						if err := positionStore.ClosePositionFully(localPos.ID, localPos.EntryPrice, "normalize_side", nowMs, 0, 0, "normalize_side"); err != nil {
							logger.Infof("⚠️  Failed to close legacy position %d: %v", localPos.ID, err)
							continue
						}
						entryPrice := realPos.EntryPrice
						if entryPrice == 0 {
							entryPrice = localPos.EntryPrice
						}
						newPos := &store.TraderPosition{
							TraderID:           traderID,
							ExchangeID:         exchangeID,
							ExchangeType:       exchangeType,
							ExchangePositionID: fmt.Sprintf("normalize_%s_%s_%d", market.Normalize(localPos.Symbol), mappedSide, nowMs),
							Symbol:             market.Normalize(localPos.Symbol),
							Side:               mappedSide,
							Quantity:           realPos.Quantity,
							EntryPrice:         entryPrice,
							EntryOrderID:       "normalize_side",
							EntryTime:          nowMs,
							Leverage:           localPos.Leverage,
							Status:             "OPEN",
							Source:             "sync",
							Fee:                localPos.Fee,
							CreatedAt:          nowMs,
							UpdatedAt:          nowMs,
						}
						if err := positionStore.CreateOpenPosition(newPos); err != nil {
							logger.Infof("⚠️  Failed to create normalized position for %s: %v", localPos.Symbol, err)
						}
					} else {
						if err := positionStore.ClosePositionFully(localPos.ID, localPos.EntryPrice, "normalize_side", nowMs, 0, 0, "normalize_side"); err != nil {
							logger.Infof("⚠️  Failed to close legacy position %d: %v", localPos.ID, err)
						}
					}
				} else {
					if err := positionStore.ClosePositionFully(localPos.ID, localPos.EntryPrice, "normalize_side", nowMs, 0, 0, "normalize_side"); err != nil {
						logger.Infof("⚠️  Failed to close legacy position %d: %v", localPos.ID, err)
					}
				}
			}
		}
	}

	for _, trade := range trades {
		// Normalize symbol
		symbol := market.Normalize(trade.Symbol)

		// Check if trade already exists (use exchangeID which is UUID, not exchange type)
		// Virtual exchange uses OrderID as TradeID
		existing, err := orderStore.GetOrderByExchangeID(exchangeID, trade.TradeID)
		if err == nil && existing != nil {
			positionSide := strings.ToUpper(trade.PositionSide)
			if positionSide == "" {
				positionSide = inferPositionSide(trade.Side, trade.OrderAction)
			}
			orderAction := trade.OrderAction
			if orderAction == "" {
				orderAction = getOrderAction(strings.ToUpper(trade.Side), positionSide)
			}
			if strings.HasPrefix(orderAction, "close_") && trade.Price > 0 {
				if err := posBuilder.ProcessTrade(
					traderID, exchangeID, exchangeType,
					symbol, positionSide, orderAction,
					trade.Quantity, trade.Price, trade.Fee, trade.RealizedPnL,
					trade.Time.UTC().UnixMilli(), trade.TradeID,
				); err != nil {
					logger.Infof("  ⚠️ Retry position update for existing trade %s failed: %v", trade.TradeID, err)
				}
			}
			continue
		}

		positionSide := strings.ToUpper(trade.PositionSide)
		if positionSide == "" {
			positionSide = inferPositionSide(trade.Side, trade.OrderAction)
		}
		orderAction := trade.OrderAction
		if orderAction == "" {
			orderAction = getOrderAction(strings.ToUpper(trade.Side), positionSide)
		}
		tradeTimeMs := trade.Time.UTC().UnixMilli()
		orderRecord := &store.TraderOrder{
			TraderID:        traderID,
			ExchangeID:      exchangeID,   // UUID
			ExchangeType:    exchangeType, // Exchange type
			ExchangeOrderID: trade.TradeID,
			ClientOrderID:   fmt.Sprintf("sync_%s", trade.TradeID),
			Symbol:          symbol,
			Side:            strings.ToUpper(trade.Side),
			PositionSide:    positionSide,
			Type:            "MARKET",
			OrderAction:     orderAction,
			Quantity:        trade.Quantity,
			Price:           trade.Price,
			Status:          "FILLED",
			FilledQuantity:  trade.Quantity,
			AvgFillPrice:    trade.Price,
			Commission:      trade.Fee,
			FilledAt:        tradeTimeMs,
			CreatedAt:       tradeTimeMs,
			UpdatedAt:       tradeTimeMs,
		}

		// Save order to DB
		if err := orderStore.CreateOrder(orderRecord); err != nil {
			logger.Errorf("  ❌ Failed to save order %s: %v", trade.TradeID, err)
			continue
		}

		fillRecord := &store.TraderFill{
			TraderID:        traderID,
			ExchangeID:      exchangeID,
			ExchangeType:    exchangeType,
			OrderID:         orderRecord.ID,
			ExchangeOrderID: trade.TradeID,
			ExchangeTradeID: trade.TradeID,
			Symbol:          symbol,
			Side:            strings.ToUpper(trade.Side),
			Price:           trade.Price,
			Quantity:        trade.Quantity,
			QuoteQuantity:   trade.Price * trade.Quantity,
			Commission:      trade.Fee,
			CommissionAsset: "USDT",
			RealizedPnL:     trade.RealizedPnL,
			IsMaker:         false,
			CreatedAt:       tradeTimeMs,
		}

		if err := orderStore.CreateFill(fillRecord); err != nil {
			logger.Infof("  ⚠️ Failed to sync fill for trade %s: %v", trade.TradeID, err)
		}

		if err := posBuilder.ProcessTrade(
			traderID, exchangeID, exchangeType,
			symbol, positionSide, orderAction,
			trade.Quantity, trade.Price, trade.Fee, trade.RealizedPnL,
			tradeTimeMs, trade.TradeID,
		); err != nil {
			logger.Errorf("  ❌ Failed to update position for %s: %v", trade.TradeID, err)
		} else {
			syncedCount++
		}
	}

	if syncedCount > 0 {
		logger.Infof("✅ Synced %d new trades from Virtual exchange", syncedCount)
	}

	return nil
}

// StartOrderSync starts background order sync task for Virtual exchange
func (t *VirtualTrader) StartOrderSync(traderID string, exchangeID string, exchangeType string, st *store.Store, interval time.Duration) {
	ticker := time.NewTicker(interval)
	go func() {
		for range ticker.C {
			if err := t.SyncOrdersFromVirtual(traderID, exchangeID, exchangeType, st); err != nil {
				logger.Infof("⚠️  Virtual order sync failed: %v", err)
			}
		}
	}()
	logger.Infof("🔄 Virtual order sync started (interval: %v)", interval)
}

func inferPositionSide(side, orderAction string) string {
	action := strings.ToLower(orderAction)
	if strings.Contains(action, "long") {
		return "LONG"
	}
	if strings.Contains(action, "short") {
		return "SHORT"
	}
	switch strings.ToUpper(side) {
	case "BUY":
		return "LONG"
	case "SELL":
		return "SHORT"
	default:
		return ""
	}
}
