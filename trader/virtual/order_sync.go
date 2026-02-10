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

	for _, trade := range trades {
		// Check if trade already exists (use exchangeID which is UUID, not exchange type)
		// Virtual exchange uses OrderID as TradeID
		existing, err := orderStore.GetOrderByExchangeID(exchangeID, trade.TradeID)
		if err == nil && existing != nil {
			continue // Order already exists, skip
		}

		// Normalize symbol
		symbol := market.Normalize(trade.Symbol)

		// Create order record - use Unix milliseconds UTC
		tradeTimeMs := trade.Time.UTC().UnixMilli()
		orderRecord := &store.TraderOrder{
			TraderID:        traderID,
			ExchangeID:      exchangeID,   // UUID
			ExchangeType:    exchangeType, // Exchange type
			ExchangeOrderID: trade.TradeID,
			ClientOrderID:   fmt.Sprintf("sync_%s", trade.TradeID),
			Symbol:          symbol,
			Side:            strings.ToUpper(trade.Side),
			PositionSide:    trade.PositionSide,
			Type:            "MARKET",
			OrderAction:     trade.OrderAction,
			Quantity:        trade.Quantity,
			Price:           trade.Price,
			Status:          "FILLED",
			FilledQuantity:  trade.Quantity,
			AvgFillPrice:    trade.Price,
			Commission:      trade.Fee,
			FilledAt:        store.UnixTime(tradeTimeMs),
			CreatedAt:       store.UnixTime(tradeTimeMs),
		}

		// Save order to DB
		if err := orderStore.CreateOrder(orderRecord); err != nil {
			logger.Errorf("  ❌ Failed to save order %s: %v", trade.TradeID, err)
			continue
		}

		// Update position from trade
		// This handles both opening and closing positions, including PnL calculation
		if err := posBuilder.ProcessTrade(
			traderID, exchangeID, exchangeType,
			symbol, strings.ToUpper(trade.Side), trade.OrderAction,
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
