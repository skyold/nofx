package trader

import (
	"fmt"
	"nofx/logger"
	"nofx/market"
	"nofx/store"
	"sort"
	"strings"
	"sync"
	"time"
)

// syncState stores the last sync time (Unix ms) for incremental sync
var (
	binanceSyncState      = make(map[string]int64) // exchangeID -> lastSyncTimeMs (Unix ms)
	binanceSyncStateMutex sync.RWMutex
)

// SyncOrdersFromBinance syncs Binance Futures trade history to local database
// Uses COMMISSION detection + fromId for efficient incremental sync
// Also creates/updates position records to ensure orders/fills/positions data consistency
func (t *FuturesTrader) SyncOrdersFromBinance(traderID string, exchangeID string, exchangeType string, st *store.Store) error {
	if st == nil {
		return fmt.Errorf("store is nil")
	}

	orderStore := st.Order()

	// Get last sync time (Unix ms) - first try memory, then database, then default
	binanceSyncStateMutex.RLock()
	lastSyncTimeMs, exists := binanceSyncState[exchangeID]
	binanceSyncStateMutex.RUnlock()

	nowMs := time.Now().UTC().UnixMilli()
	if !exists {
		// Try to get last fill time from database (persist across restarts)
		lastFillTimeMs, err := orderStore.GetLastFillTimeByExchange(exchangeID)
		if err == nil && lastFillTimeMs > 0 {
			// If recovered time is in the future, it's clearly wrong - use default
			if lastFillTimeMs > nowMs {
				logger.Infof("⚠️ DB sync time %d is in the future (now: %d), using default",
					lastFillTimeMs, nowMs)
				lastSyncTimeMs = nowMs - 24*60*60*1000 // 24 hours ago
			} else {
				// Add 1 second buffer to avoid re-fetching the same fill
				lastSyncTimeMs = lastFillTimeMs + 1000
				logger.Infof("📅 Recovered last sync time from DB: %s (UTC)",
					time.UnixMilli(lastSyncTimeMs).UTC().Format("2006-01-02 15:04:05"))
			}
		} else {
			// First sync: go back 24 hours
			lastSyncTimeMs = nowMs - 24*60*60*1000
			logger.Infof("📅 First sync, starting from 24 hours ago: %s (UTC)",
				time.UnixMilli(lastSyncTimeMs).UTC().Format("2006-01-02 15:04:05"))
		}
	}

	logger.Infof("🔄 Syncing Binance trades from: %s (UTC) [ms: %d, now: %d]",
		time.UnixMilli(lastSyncTimeMs).UTC().Format("2006-01-02 15:04:05"), lastSyncTimeMs, nowMs)

	// Step 1: Get max trade IDs from local DB for incremental sync
	maxTradeIDs, err := orderStore.GetMaxTradeIDsByExchange(exchangeID)
	if err != nil {
		logger.Infof("  ⚠️ Failed to get max trade IDs: %v, will use time-based query", err)
		maxTradeIDs = make(map[string]int64)
	}

	// Step 2: Detect symbols to sync using multiple methods
	// COMMISSION detection may miss trades (VIP users, BNB discount, 0-fee trades)
	symbolMap := make(map[string]bool)
	lastSyncTime := time.UnixMilli(lastSyncTimeMs) // Convert to time.Time for API calls

	// Method 1: COMMISSION income detection
	commissionSymbols, err := t.GetCommissionSymbols(lastSyncTime)
	if err != nil {
		logger.Infof("  ⚠️ Failed to get commission symbols: %v", err)
	} else {
		logger.Infof("  📋 COMMISSION symbols found: %d - %v", len(commissionSymbols), commissionSymbols)
		for _, s := range commissionSymbols {
			symbolMap[s] = true
		}
	}

	// Method 2: Always include active positions (catches trades that COMMISSION missed)
	positionSymbols := t.getPositionSymbols()
	logger.Infof("  📋 Position symbols found: %d - %v", len(positionSymbols), positionSymbols)
	for _, s := range positionSymbols {
		symbolMap[s] = true
	}

	// Method 3: Include symbols from local DB open positions (to detect if they were closed externally)
	// This fixes the issue where a position closed on exchange (but open in DB) is missed by sync because it has no active position/commission
	localPositions, err := st.Position().GetOpenPositions(traderID)
	if err != nil {
		logger.Infof("  ⚠️ Failed to get local open positions: %v", err)
	} else {
		localSymbols := make([]string, 0)
		for _, p := range localPositions {
			symbolMap[p.Symbol] = true
			localSymbols = append(localSymbols, p.Symbol)
		}
		logger.Infof("  📋 Local open position symbols found: %d - %v", len(localPositions), localSymbols)
	}

	// Method 4: Include symbols from recent fills in DB (in case some were partially synced)
	recentSymbols, _ := orderStore.GetRecentFillSymbolsByExchange(exchangeID, lastSyncTimeMs)
	logger.Infof("  📋 Recent fill symbols found: %d - %v", len(recentSymbols), recentSymbols)
	for _, s := range recentSymbols {
		symbolMap[s] = true
	}

	// Method 4: ALWAYS query REALIZED_PNL income to find symbols with closed trades
	// This catches trades that COMMISSION missed (VIP users, BNB fee discount)
	// IMPORTANT: Must run always, not just when symbolMap is empty,
	// because a position might be fully closed (no active position) but have PnL
	pnlSymbols, err := t.GetPnLSymbols(lastSyncTime)
	if err != nil {
		logger.Infof("  ⚠️ Failed to get PnL symbols: %v", err)
	} else {
		logger.Infof("  📋 REALIZED_PNL symbols found: %d - %v", len(pnlSymbols), pnlSymbols)
		for _, s := range pnlSymbols {
			symbolMap[s] = true
		}
	}

	var changedSymbols []string
	for s := range symbolMap {
		changedSymbols = append(changedSymbols, s)
	}

	if len(changedSymbols) == 0 {
		logger.Infof("📭 No symbols with new trades to sync")
		// DON'T update lastSyncTime to current time here!
		// Keep using the last actual trade time from DB to avoid creating gaps
		// The lastSyncTimeMs from DB already has +1000ms buffer added
		return nil
	}

	logger.Infof("📊 Found %d symbols with new trades: %v", len(changedSymbols), changedSymbols)

	// Step 3: Query trades for changed symbols using fromId (incremental) or time-based (new symbols)
	var allTrades []TradeRecord
	var failedSymbols []string
	apiCalls := 0
	for _, symbol := range changedSymbols {
		var trades []TradeRecord
		var queryErr error

		if lastID, ok := maxTradeIDs[symbol]; ok && lastID > 0 {
			// Incremental sync: query from last known trade ID
			trades, queryErr = t.GetTradesForSymbolFromID(symbol, lastID+1, 500)
		} else {
			// New symbol or first sync: query by time
			trades, queryErr = t.GetTradesForSymbol(symbol, lastSyncTime, 500)
		}
		apiCalls++

		if queryErr != nil {
			logger.Infof("  ⚠️ Failed to get trades for %s: %v", symbol, queryErr)
			failedSymbols = append(failedSymbols, symbol)
			continue
		}
		allTrades = append(allTrades, trades...)
	}

	logger.Infof("📥 Received %d trades from Binance (%d API calls)", len(allTrades), apiCalls)

	if len(allTrades) == 0 {
		// No trades returned, but symbols were detected - might be false positive from COMMISSION/PnL detection
		// Don't update lastSyncTime, keep using DB value
		if len(failedSymbols) > 0 {
			logger.Infof("  ⚠️ %d symbols failed: %v", len(failedSymbols), failedSymbols)
		}
		return nil
	}

	// Sort trades by time ASC (oldest first) for proper position building
	sort.Slice(allTrades, func(i, j int) bool {
		return allTrades[i].Time.UnixMilli() < allTrades[j].Time.UnixMilli()
	})

	// Process trades one by one
	positionStore := st.Position()
	posBuilder := store.NewPositionBuilder(positionStore)
	syncedCount := 0

	skippedCount := 0
	for _, trade := range allTrades {
		// 1. Check if this FILL (Trade) already exists
		existingFill, err := orderStore.GetFillByExchangeTradeID(exchangeID, trade.TradeID)
		if err == nil && existingFill != nil {
			continue // Fill already processed, skip
		}

		// Normalize symbol
		symbol := market.Normalize(trade.Symbol)
		side := strings.ToUpper(trade.Side)
		orderAction := t.determineOrderAction(trade.Side, trade.PositionSide, trade.RealizedPnL)

		// Determine position side for position builder
		positionSide := trade.PositionSide
		if positionSide == "" || positionSide == "BOTH" {
			// Infer from order action
			if strings.Contains(orderAction, "long") {
				positionSide = "LONG"
			} else {
				positionSide = "SHORT"
			}
		}

		// Create order record - use Unix milliseconds UTC
		tradeTimeMs := trade.Time.UTC().UnixMilli()

		// Use OrderID from trade if available, otherwise fallback to TradeID
		exchangeOrderID := trade.OrderID
		if exchangeOrderID == "" || exchangeOrderID == "0" {
			exchangeOrderID = trade.TradeID
		}

		// [CRITICAL] Determine the correct TraderID
		// 1. Try to find existing order in local DB
		// 2. If found, use its TraderID (this handles cases where multiple traders share an account)
		// 3. If not found, fallback to the current sync task's TraderID
		finalTraderID := traderID
		existingOrder, err := orderStore.GetOrderByExchangeID(exchangeID, exchangeOrderID)
		if err == nil && existingOrder != nil && existingOrder.TraderID != "" {
			finalTraderID = existingOrder.TraderID
			// Log if we're syncing a trade that belongs to a different trader than the sync context
			if finalTraderID != traderID {
				logger.Infof("  ℹ️  Trade %s belongs to trader %s (not current sync context %s), respecting local DB record",
					trade.TradeID, finalTraderID, traderID)
			}
		}

		orderRecord := &store.TraderOrder{
			TraderID:        finalTraderID,
			ExchangeID:      exchangeID,
			ExchangeType:    exchangeType,
			ExchangeOrderID: exchangeOrderID,
			Symbol:          symbol,
			Side:            side,
			PositionSide:    positionSide,
			Type:            "MARKET",
			OrderAction:     orderAction,
			Quantity:        trade.Quantity,
			Price:           trade.Price,
			Status:          "FILLED",
			FilledQuantity:  trade.Quantity,
			AvgFillPrice:    trade.Price,
			Commission:      trade.Fee,
			FilledAt:        store.UnixTime(tradeTimeMs),
			CreatedAt:       store.UnixTime(tradeTimeMs),
			UpdatedAt:       store.UnixTime(tradeTimeMs),
		}

		// Insert order record (or find existing)
		if err := orderStore.CreateOrder(orderRecord); err != nil {
			logger.Infof("  ⚠️ Failed to sync order %s: %v", exchangeOrderID, err)
			continue
		}

		// Ensure status is updated to FILLED (if it was created as NEW by AutoTrader)
		if orderRecord.ID > 0 {
			orderStore.UpdateOrderStatus(orderRecord.ID, "FILLED", trade.Quantity, trade.Price, trade.Fee)
		}

		// Create fill record - use Unix milliseconds UTC
		fillRecord := &store.TraderFill{
			TraderID:        finalTraderID,
			ExchangeID:      exchangeID,
			ExchangeType:    exchangeType,
			OrderID:         orderRecord.ID,
			ExchangeOrderID: exchangeOrderID,
			ExchangeTradeID: trade.TradeID,
			Symbol:          symbol,
			Side:            side,
			Price:           trade.Price,
			Quantity:        trade.Quantity,
			QuoteQuantity:   trade.Price * trade.Quantity,
			Commission:      trade.Fee,
			CommissionAsset: "USDT",
			RealizedPnL:     trade.RealizedPnL,
			IsMaker:         false,
			CreatedAt:       store.UnixTime(tradeTimeMs),
		}

		if err := orderStore.CreateFill(fillRecord); err != nil {
			logger.Infof("  ⚠️ Failed to sync fill for trade %s: %v", trade.TradeID, err)
		}

		// Create/update position record using PositionBuilder
		if err := posBuilder.ProcessTrade(
			finalTraderID, exchangeID, exchangeType,
			symbol, positionSide, orderAction,
			trade.Quantity, trade.Price, trade.Fee, trade.RealizedPnL,
			tradeTimeMs, exchangeOrderID,
		); err != nil {
			logger.Infof("  ⚠️ Failed to sync position for trade %s: %v", trade.TradeID, err)
		} else {
			logger.Infof("  📍 Position updated for trade: %s (action: %s, qty: %.6f)", trade.TradeID, orderAction, trade.Quantity)
		}

		syncedCount++
		logger.Infof("  ✅ Synced trade: %s %s %s qty=%.6f price=%.6f pnl=%.2f fee=%.6f action=%s time=%s(UTC)",
			trade.TradeID, symbol, side, trade.Quantity, trade.Price, trade.RealizedPnL, trade.Fee, orderAction,
			trade.Time.UTC().Format("01-02 15:04:05"))
	}

	// Update lastSyncTime to the LATEST trade time (not current time!)
	// This ensures next sync starts from where we left off, not from "now"
	// allTrades is already sorted by time ASC, so last element is the latest
	if len(allTrades) > 0 && len(failedSymbols) == 0 {
		latestTradeTimeMs := allTrades[len(allTrades)-1].Time.UTC().UnixMilli()
		binanceSyncStateMutex.Lock()
		binanceSyncState[exchangeID] = latestTradeTimeMs
		binanceSyncStateMutex.Unlock()
		logger.Infof("📅 Updated lastSyncTime to latest trade: %s (UTC)",
			time.UnixMilli(latestTradeTimeMs).UTC().Format("2006-01-02 15:04:05"))
	} else if len(failedSymbols) > 0 {
		logger.Infof("  ⚠️ %d symbols failed, not updating lastSyncTime to retry next time: %v", len(failedSymbols), failedSymbols)
	}

	// Step 5: [NEW] Reconcile positions with snapshot
	// After processing all incremental trades, perform a snapshot check for all active symbols
	// This fixes "Phantom Positions" where a close was missed or happened externally
	// We only do this if we successfully synced trades (or if no new trades were found but we have active symbols)
	if len(failedSymbols) == 0 {
		t.reconcilePositions(traderID, exchangeID, exchangeType, st, symbolMap)
	}

	logger.Infof("✅ Binance order sync completed: %d new trades synced, %d skipped (already exist)", syncedCount, skippedCount)
	return nil
}

// reconcilePositions compares local OPEN positions with exchange active positions and fixes discrepancies
func (t *FuturesTrader) reconcilePositions(traderID, exchangeID, exchangeType string, st *store.Store, relevantSymbols map[string]bool) {
	// 1. Get ALL active positions from exchange (Snapshot)
	exchangePositions, err := t.GetPositions()
	if err != nil {
		logger.Infof("  ⚠️ Failed to get exchange positions for reconciliation: %v", err)
		return
	}

	// Map: Symbol -> Side -> Quantity
	// Normalize symbol and side
	type PosKey struct {
		Symbol string
		Side   string
	}
	exchangePosMap := make(map[PosKey]float64)

	for _, pos := range exchangePositions {
		symbol, _ := pos["symbol"].(string)
		if symbol == "" {
			continue
		}

		// Binance Futures API returns positions as list.
		// For One-Way Mode: Side is usually "BOTH" or empty, but positionAmt can be + or -
		// For Hedge Mode: Side is "LONG" or "SHORT"
		// We need to normalize to our internal "LONG"/"SHORT" representation

		positionAmtStr, _ := pos["positionAmt"].(string)
		positionSide, _ := pos["positionSide"].(string) // "BOTH", "LONG", "SHORT"

		qty, _ := market.ParseFloat(positionAmtStr)
		if qty == 0 {
			continue
		}

		symbol = market.Normalize(symbol)

		var side string
		var absQty float64

		if positionSide == "LONG" {
			side = "LONG"
			absQty = qty
		} else if positionSide == "SHORT" {
			side = "SHORT"
			absQty = -qty // API returns negative for short? Usually positive for Hedge Short, but check API.
			// Actually Binance Hedge Mode: Short positionAmt is negative? Let's assume absolute.
			// Wait, documentation says: "positionAmt": "-0.001" for SHORT usually.
			// Let's take absolute value for quantity storage.
			if absQty < 0 {
				absQty = -absQty
			}
		} else {
			// One-way mode "BOTH"
			if qty > 0 {
				side = "LONG"
				absQty = qty
			} else {
				side = "SHORT"
				absQty = -qty
			}
		}

		exchangePosMap[PosKey{Symbol: symbol, Side: side}] = absQty
	}

	// 2. Get ALL local OPEN positions
	localPositions, err := st.Position().GetOpenPositions(traderID)
	if err != nil {
		logger.Infof("  ⚠️ Failed to get local positions for reconciliation: %v", err)
		return
	}

	// 3. Compare and Fix
	positionStore := st.Position()
	nowMs := time.Now().UTC().UnixMilli()

	// A. Check for Phantom Positions (Local exists, Exchange does not or quantity mismatch)
	for _, localPos := range localPositions {
		// Only check symbols that we are interested in (or should we check all?)
		// Checking all is safer to catch ghosts.
		// But if we are in a partial sync (e.g. only syncing specific symbols), we should be careful?
		// SyncOrdersFromBinance targets specific symbols? No, it targets "changedSymbols" but logic above discovers ALL relevant symbols.
		// However, relevantSymbols might not cover everything if detection failed.
		// Safest approach: Only reconcile symbols that appear in exchange snapshot OR local DB.
		// Since we have full snapshot of exchange positions, we can trust it for ANY symbol.

		key := PosKey{Symbol: localPos.Symbol, Side: localPos.Side}
		exchangeQty, existsOnExchange := exchangePosMap[key]

		if !existsOnExchange {
			// Phantom Position! Exchange has 0, Local has > 0
			logger.Infof("  👻 Found PHANTOM position: %s %s (Local: %.6f, Exchange: 0). Closing it.",
				localPos.Symbol, localPos.Side, localPos.Quantity)

			// Force close
			// We don't have exit price/time from a specific trade, so we use current info or 0?
			// Ideally we should have found the trade. If not, this is a "force sync" close.
			err := positionStore.ClosePositionFully(
				localPos.ID,
				0, // Exit Price unknown
				"force_sync_reconcile",
				nowMs,
				0, // PnL unknown
				0, // Fee unknown
				"reconcile",
			)
			if err != nil {
				logger.Errorf("    Failed to close phantom position: %v", err)
			}
		} else {
			// Exists on both. Check quantity.
			diff := localPos.Quantity - exchangeQty
			if diff < -0.000001 || diff > 0.000001 {
				logger.Infof("  ⚖️ Quantity Mismatch for %s %s: Local %.6f != Exchange %.6f. Adjusting.",
					localPos.Symbol, localPos.Side, localPos.Quantity, exchangeQty)

				// Adjust quantity
				// We use UpdatePositionQuantityAndPrice but with 0 price/fee difference to just set quantity?
				// Actually UpdatePositionQuantityAndPrice adds/subtracts.
				// Let's calculate delta.
				// delta := exchangeQty - localPos.Quantity

				// We don't want to affect Entry Price if possible, or do we?
				// If we assume partial close happened, Entry Price stays same.
				// If we assume partial open happened, Entry Price might change.
				// Without trade details, safer to keep Entry Price same and just fix Quantity.
				// But UpdatePositionQuantityAndPrice recalculates Entry Price.

				// Let's use a direct update for reconciliation to avoid messing up Entry Price with 0
				err := st.GormDB().Model(&store.TraderPosition{}).Where("id = ?", localPos.ID).
					Updates(map[string]interface{}{
						"quantity":   exchangeQty,
						"updated_at": nowMs,
						"source":     "reconcile",
					}).Error

				if err != nil {
					logger.Errorf("    Failed to update position quantity: %v", err)
				}
			}
		}
	}

	// B. Check for Missing Positions (Exchange exists, Local does not)
	// (Optional: If we missed the OPEN trade entirely)
	for key, qty := range exchangePosMap {
		// Check if we already have it locally
		found := false
		for _, localPos := range localPositions {
			if localPos.Symbol == key.Symbol && localPos.Side == key.Side {
				found = true
				break
			}
		}

		if !found {
			logger.Infof("  👻 Found MISSING position: %s %s (Exchange: %.6f, Local: 0). Creating it.",
				key.Symbol, key.Side, qty)

			// Create new position
			// We don't know Entry Price/Time. Use current market price? Or 0?
			// Ideally fetch ticker price. For now, 0 to indicate unknown.
			newPos := &store.TraderPosition{
				TraderID:     traderID,
				ExchangeID:   exchangeID,
				ExchangeType: exchangeType,
				Symbol:       key.Symbol,
				Side:         key.Side,
				Quantity:     qty,
				EntryPrice:   0, // Unknown
				Status:       "OPEN",
				Source:       "reconcile",
				CreatedAt:    store.UnixTime(nowMs),
				UpdatedAt:    store.UnixTime(nowMs),
			}
			if err := positionStore.CreateOpenPosition(newPos); err != nil {
				logger.Errorf("    Failed to create missing position: %v", err)
			}
		}
	}
}

// Used as fallback when COMMISSION detection fails
func (t *FuturesTrader) getPositionSymbols() []string {
	positions, err := t.GetPositions()
	if err != nil {
		return nil
	}

	var symbols []string
	for _, pos := range positions {
		if symbol, ok := pos["symbol"].(string); ok && symbol != "" {
			symbols = append(symbols, symbol)
		}
	}
	return symbols
}

// determineOrderAction determines the order action based on trade data
func (t *FuturesTrader) determineOrderAction(side, positionSide string, realizedPnL float64) string {
	side = strings.ToUpper(side)
	positionSide = strings.ToUpper(positionSide)

	// If there's realized PnL, it's likely a close trade
	isClose := realizedPnL != 0

	if positionSide == "LONG" || positionSide == "" {
		if side == "BUY" {
			if isClose {
				return "close_short" // Buying to close short
			}
			return "open_long"
		} else {
			if isClose {
				return "close_long" // Selling to close long
			}
			return "open_short"
		}
	} else if positionSide == "SHORT" {
		if side == "SELL" {
			if isClose {
				return "close_long"
			}
			return "open_short"
		} else {
			if isClose {
				return "close_short"
			}
			return "open_long"
		}
	}

	// Default fallback
	if side == "BUY" {
		return "open_long"
	}
	return "open_short"
}

// StartOrderSync starts background order sync task for Binance
func (t *FuturesTrader) StartOrderSync(traderID string, exchangeID string, exchangeType string, st *store.Store, interval time.Duration) {
	// Run first sync immediately
	go func() {
		logger.Infof("🔄 Running initial Binance order sync...")
		if err := t.SyncOrdersFromBinance(traderID, exchangeID, exchangeType, st); err != nil {
			logger.Infof("⚠️  Initial Binance order sync failed: %v", err)
		}
	}()

	// Then run periodically
	ticker := time.NewTicker(interval)
	go func() {
		for range ticker.C {
			if err := t.SyncOrdersFromBinance(traderID, exchangeID, exchangeType, st); err != nil {
				logger.Infof("⚠️  Binance order sync failed: %v", err)
			}
		}
	}()
	logger.Infof("🔄 Binance order sync started (interval: %v)", interval)
}
