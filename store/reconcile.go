package store

import (
	"fmt"
	"math"
	"nofx/logger"
	"nofx/market"
	"nofx/trader/types"
	"strings"
	"time"
)

// ReconcilePositions reconciles local positions with exchange positions
// It detects and fixes:
// 1. Ghost positions (exist locally but closed on exchange)
// 2. Quantity mismatches (partial external fills/closes)
func (s *PositionStore) ReconcilePositions(trader types.Trader, traderID, exchangeID string) error {
	// 1. Get real positions from exchange
	exchangePositions, err := trader.GetPositions()
	if err != nil {
		return fmt.Errorf("failed to get exchange positions: %w", err)
	}

	// Map: Symbol_Side -> Position Info
	realPositions := make(map[string]struct {
		Quantity   float64
		EntryPrice float64
	})

	for _, pos := range exchangePositions {
		symbol := pos.Symbol
		side := pos.Side
		qty := pos.Quantity
		entryPrice := pos.EntryPrice

		if symbol == "" || qty == 0 {
			continue
		}

		// Normalize: ensure side is "long" or "short", quantity is positive
		side = strings.ToLower(side)
		if qty < 0 {
			qty = -qty
			if side == "" { // Some exchanges might not set side if qty is negative implies short
				side = "short"
			}
		}

		normalizedSymbol := market.Normalize(symbol)
		key := fmt.Sprintf("%s_%s", normalizedSymbol, side)
		realPositions[key] = struct {
			Quantity   float64
			EntryPrice float64
		}{
			Quantity:   qty,
			EntryPrice: entryPrice,
		}
	}

	// 2. Get local OPEN positions
	localPositions, err := s.GetOpenPositions(traderID)
	if err != nil {
		return fmt.Errorf("failed to get local positions: %w", err)
	}

	// 3. Compare and Fix
	reconciledCount := 0
	for _, localPos := range localPositions {
		// Only check positions for this exchange account
		if localPos.ExchangeID != exchangeID {
			continue
		}

		key := fmt.Sprintf("%s_%s", localPos.Symbol, strings.ToLower(localPos.Side))
		realPos, exists := realPositions[key]

		nowMs := time.Now().UTC().UnixMilli()

		if !exists {
			// Case 1: Ghost position (exists locally but not on exchange)
			logger.Infof("👻 Found ghost position: %s %s (qty: %.4f), closing...", localPos.Symbol, localPos.Side, localPos.Quantity)
			
			// Close it locally
			// Use current market price as exit price if possible, or entry price as fallback
			exitPrice := localPos.EntryPrice
			if marketData, err := market.Get(localPos.Symbol); err == nil {
				exitPrice = marketData.CurrentPrice
			}

			// Calculate PnL roughly
			var pnl float64
			if localPos.Side == "LONG" {
				pnl = (exitPrice - localPos.EntryPrice) * localPos.Quantity
			} else {
				pnl = (localPos.EntryPrice - exitPrice) * localPos.Quantity
			}

			err := s.ClosePositionFully(
				localPos.ID,
				exitPrice,
				"reconcile_auto_close",
				nowMs,
				pnl,
				0, // Unknown fee
				"reconcile_ghost",
			)
			if err != nil {
				logger.Errorf("❌ Failed to close ghost position %s: %v", localPos.Symbol, err)
			} else {
				reconciledCount++
			}

		} else {
			// Case 2: Quantity mismatch
			// Allow small tolerance for floating point errors
			diff := math.Abs(localPos.Quantity - realPos.Quantity)
			if diff > 0.00001 {
				logger.Infof("⚠️ Quantity mismatch for %s %s: local=%.4f, real=%.4f, fixing...", 
					localPos.Symbol, localPos.Side, localPos.Quantity, realPos.Quantity)
				
				// Update quantity directly
				// Calculate delta to add (can be negative)
				delta := realPos.Quantity - localPos.Quantity
				
				err := s.UpdatePositionQuantityAndPrice(
					localPos.ID, 
					delta,
					realPos.EntryPrice, // Use real entry price
					0, // No extra fee info
				)
				if err != nil {
					logger.Errorf("❌ Failed to update position quantity %s: %v", localPos.Symbol, err)
				} else {
					reconciledCount++
				}
			}
		}
	}

	if reconciledCount > 0 {
		logger.Infof("✅ Reconciled %d positions", reconciledCount)
	}

	return nil
}
