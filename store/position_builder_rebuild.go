package store

import (
	"fmt"
	"nofx/logger"
	"strings"
)

// RebuildFromFills rebuilds all positions from trade history (fills)
// This is a destructive operation that deletes all existing positions for the trader
// and recreates them by replaying the fill history.
func (pb *PositionBuilder) RebuildFromFills(traderID string, orderStore *OrderStore) (int, error) {
	logger.Infof("🔄 Starting position rebuild for trader %s...", traderID)

	// 1. Get all fills with actions
	fills, err := orderStore.GetFillsWithActions(traderID)
	if err != nil {
		return 0, fmt.Errorf("failed to get fills: %w", err)
	}
	
	if len(fills) == 0 {
		logger.Infof("⚠️ No fills found for trader %s, clearing all positions", traderID)
		if err := pb.positionStore.DeleteAllPositions(traderID); err != nil {
			return 0, fmt.Errorf("failed to clear positions: %w", err)
		}
		return 0, nil
	}

	logger.Infof("📊 Found %d fills to process", len(fills))

	// 2. Clear all existing positions
	if err := pb.positionStore.DeleteAllPositions(traderID); err != nil {
		return 0, fmt.Errorf("failed to clear positions: %w", err)
	}

	// 3. Replay fills
	count := 0
	for _, fill := range fills {
		// Use OrderAction if available, otherwise infer from side
		action := fill.OrderAction
		if action == "" {
			// Infer action from side (simple assumption, might be wrong for complex cases)
			if fill.Side == "BUY" {
				action = "open_long" // Default assumption
			} else {
				action = "close_long" // Default assumption
			}
			// Note: This inference is weak. Ideally OrderAction should always be present.
			// If we have position_side in order, we could do better.
			// But GetFillsWithActions only joins order_action.
		}

		// Determine position side from action
		var posSide string
		if strings.Contains(action, "_long") {
			posSide = "LONG"
		} else if strings.Contains(action, "_short") {
			posSide = "SHORT"
		} else {
			posSide = fill.Side // Fallback
		}

		err := pb.ProcessTrade(
			fill.TraderID,
			fill.ExchangeID,
			fill.ExchangeType,
			fill.Symbol,
			posSide,
			action,
			fill.Quantity,
			fill.Price,
			fill.Commission,
			fill.RealizedPnL,
			int64(fill.CreatedAt),
			fmt.Sprintf("%d", fill.OrderID), // Use internal order ID as reference
		)

		if err != nil {
			logger.Errorf("❌ Failed to process fill %d: %v", fill.ID, err)
			// Continue processing other fills? Or stop? 
			// Usually better to continue to recover as much as possible.
		} else {
			count++
		}
	}

	logger.Infof("✅ Successfully rebuilt positions from %d fills", count)
	return count, nil
}
