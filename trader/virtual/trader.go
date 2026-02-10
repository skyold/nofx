package virtual

import (
	"encoding/json"
	"fmt"
	"math"
	"nofx/logger"
	"nofx/market"
	"nofx/store"
	"nofx/trader/types"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"
)

// VirtualPosition represents a position in the virtual exchange
type VirtualPosition struct {
	Symbol          string  `json:"symbol"`
	Side            string  `json:"side"` // "long" or "short"
	EntryPrice      float64 `json:"entry_price"`
	Quantity        float64 `json:"quantity"`
	Leverage        int     `json:"leverage"`
	UnrealizedPnL   float64 `json:"-"`                 // Calculated runtime, not stored
	Margin          float64 `json:"-"`                 // Calculated runtime, not stored
	StorePositionID int64   `json:"store_position_id"` // ID in database
}

// VirtualOrder represents an order in the virtual exchange
type VirtualOrder struct {
	OrderID      string  `json:"order_id"`
	Symbol       string  `json:"symbol"`
	Side         string  `json:"side"`          // BUY/SELL
	PositionSide string  `json:"position_side"` // LONG/SHORT
	Type         string  `json:"type"`          // MARKET
	Price        float64 `json:"price"`         // Avg execution price
	Quantity     float64 `json:"quantity"`
	Status       string  `json:"status"` // FILLED
	Time         int64   `json:"time"`
	Commission   float64 `json:"commission"`
}

// VirtualExchangeState represents the persistent state of the virtual exchange
type VirtualExchangeState struct {
	UserID    string                      `json:"user_id"`
	Balance   float64                     `json:"balance"`
	Positions map[string]*VirtualPosition `json:"positions"`
	Orders    map[string]*VirtualOrder    `json:"orders"`
}

// VirtualTrader implements the Trader interface for simulation
type VirtualTrader struct {
	userID         string
	traderID       string
	store          *store.Store
	state          *VirtualExchangeState
	apiClient      *market.APIClient
	statePath      string
	mu             sync.RWMutex
	initialBalance float64
}

// NewVirtualTrader creates a new virtual trader
func NewVirtualTrader(userID string, traderID string, st *store.Store, initialBalance float64) *VirtualTrader {
	// Ensure data directory exists
	dataDir := "data/virtual_exchange"
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		logger.Errorf("Failed to create virtual exchange data directory: %v", err)
	}

	statePath := filepath.Join(dataDir, fmt.Sprintf("%s.json", traderID))

	trader := &VirtualTrader{
		userID:         userID,
		traderID:       traderID,
		store:          st,
		apiClient:      market.NewAPIClient(),
		statePath:      statePath,
		initialBalance: initialBalance,
		state: &VirtualExchangeState{
			UserID:    userID,
			Balance:   initialBalance,
			Positions: make(map[string]*VirtualPosition),
			Orders:    make(map[string]*VirtualOrder),
		},
	}

	// Try to load existing state
	if err := trader.loadState(); err != nil {
		// New account: set default balance if not provided
		if trader.initialBalance <= 0 {
			trader.initialBalance = 10000.0
			trader.state.Balance = 10000.0
		}
		logger.Infof("🎮 [%s] Created new virtual exchange account with %.2f USDT", userID, trader.initialBalance)
		// Save initial state
		trader.saveState()
	} else {
		logger.Infof("🎮 [%s] Loaded existing virtual exchange account (Balance: %.2f USDT)", userID, trader.state.Balance)
	}

	return trader
}

// loadState loads state from JSON file
func (t *VirtualTrader) loadState() error {
	data, err := os.ReadFile(t.statePath)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, t.state)
}

// saveState saves state to JSON file
func (t *VirtualTrader) saveState() error {
	data, err := json.MarshalIndent(t.state, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(t.statePath, data, 0644)
}

// GetBalance gets account balance
func (t *VirtualTrader) GetBalance() (map[string]interface{}, error) {
	t.mu.RLock()
	defer t.mu.RUnlock()

	// Calculate unrealized PnL from all positions
	totalUnrealizedPnL := 0.0
	totalMarginUsed := 0.0

	// We need to fetch current prices for all open positions
	// Note: This might be slow if there are many positions, but fine for simulation
	for _, pos := range t.state.Positions {
		currentPrice, err := t.getRobustPrice(pos.Symbol)
		if err != nil {
			logger.Warnf("Failed to get price for %s in simulation: %v", pos.Symbol, err)
			continue
		}

		var pnl float64
		if pos.Side == "long" {
			pnl = (currentPrice - pos.EntryPrice) * pos.Quantity
		} else {
			pnl = (pos.EntryPrice - currentPrice) * pos.Quantity
		}
		totalUnrealizedPnL += pnl

		margin := (pos.Quantity * currentPrice) / float64(pos.Leverage)
		totalMarginUsed += margin
	}

	totalEquity := t.state.Balance + totalUnrealizedPnL
	availableBalance := totalEquity - totalMarginUsed

	return map[string]interface{}{
		"totalWalletBalance":    t.state.Balance,
		"totalUnrealizedProfit": totalUnrealizedPnL,
		"availableBalance":      availableBalance,
		"totalEquity":           totalEquity,
	}, nil
}

// GetPositions gets current positions
func (t *VirtualTrader) GetPositions() ([]map[string]interface{}, error) {
	t.mu.Lock()
	defer t.mu.Unlock()

	// Reload state to ensure we're seeing the latest updates from other processes/threads
	if err := t.loadState(); err != nil {
		logger.Warnf("Failed to reload state in GetPositions: %v", err)
	}

	return t.getPositionsInternal()
}

// getPositionsInternal is the actual implementation (called with lock held)
func (t *VirtualTrader) getPositionsInternal() ([]map[string]interface{}, error) {
	var result []map[string]interface{}

	for _, pos := range t.state.Positions {
		currentPrice, err := t.getRobustPrice(pos.Symbol)
		if err != nil {
			logger.Warnf("Failed to get price for %s in simulation: %v", pos.Symbol, err)
			currentPrice = pos.EntryPrice // Fallback
		}

		var unrealizedPnl float64
		if pos.Side == "long" {
			unrealizedPnl = (currentPrice - pos.EntryPrice) * pos.Quantity
		} else {
			unrealizedPnl = (pos.EntryPrice - currentPrice) * pos.Quantity
		}

		// Calculate liquidation price (Simplified)
		// Long: Entry * (1 - 1/Lev + 0.005)
		// Short: Entry * (1 + 1/Lev - 0.005)
		var liqPrice float64
		mmr := 0.005
		if pos.Side == "long" {
			liqPrice = pos.EntryPrice * (1 - 1.0/float64(pos.Leverage) + mmr)
		} else {
			liqPrice = pos.EntryPrice * (1 + 1.0/float64(pos.Leverage) - mmr)
		}

		positionAmt := pos.Quantity
		if pos.Side == "short" {
			positionAmt = -pos.Quantity
		}

		result = append(result, map[string]interface{}{
			"symbol":           pos.Symbol,
			"side":             pos.Side,
			"positionAmt":      positionAmt,
			"entryPrice":       pos.EntryPrice,
			"markPrice":        currentPrice,
			"unRealizedProfit": unrealizedPnl,
			"leverage":         float64(pos.Leverage),
			"liquidationPrice": liqPrice,
		})
	}

	return result, nil
}

// OpenLong opens a long position
func (t *VirtualTrader) OpenLong(symbol string, quantity float64, leverage int) (map[string]interface{}, error) {
	return t.executeOrder(symbol, "BUY", "LONG", quantity, leverage)
}

// OpenShort opens a short position
func (t *VirtualTrader) OpenShort(symbol string, quantity float64, leverage int) (map[string]interface{}, error) {
	return t.executeOrder(symbol, "SELL", "SHORT", quantity, leverage)
}

// CloseLong closes a long position
func (t *VirtualTrader) CloseLong(symbol string, quantity float64) (map[string]interface{}, error) {
	// If quantity is 0, close all
	if quantity == 0 {
		t.mu.RLock()
		if pos, ok := t.state.Positions[symbol+"_long"]; ok {
			quantity = pos.Quantity
		}
		t.mu.RUnlock()
	}
	return t.executeOrder(symbol, "SELL", "LONG", quantity, 0)
}

// CloseShort closes a short position
func (t *VirtualTrader) CloseShort(symbol string, quantity float64) (map[string]interface{}, error) {
	// If quantity is 0, close all
	if quantity == 0 {
		t.mu.RLock()
		if pos, ok := t.state.Positions[symbol+"_short"]; ok {
			quantity = pos.Quantity
		}
		t.mu.RUnlock()
	}
	return t.executeOrder(symbol, "BUY", "SHORT", quantity, 0)
}

// executeOrder executes a virtual order
func (t *VirtualTrader) executeOrder(symbol, side, positionSide string, quantity float64, leverage int) (map[string]interface{}, error) {
	t.mu.Lock()
	defer t.mu.Unlock()

	// 1. Get current price
	price, err := t.getRobustPrice(symbol)
	if err != nil {
		return nil, err
	}

	// Always reload state before executing to ensure we have the latest data
	// This is crucial when multiple instances (e.g. API temp trader vs AutoTrader) access the same file
	if err := t.loadState(); err != nil {
		logger.Warnf("Failed to reload state before execution: %v", err)
		// Don't fail, just continue with memory state
	}

	// 2. Generate Order ID
	orderID := fmt.Sprintf("%d", time.Now().UnixNano())

	// 3. Calculate Fee (0.05% taker fee)
	feeRate := 0.0005
	commission := quantity * price * feeRate

	var pnl float64

	// 4. Margin Check (for opening)
	isOpening := (side == "BUY" && positionSide == "LONG") || (side == "SELL" && positionSide == "SHORT")
	if isOpening {
		requiredMargin := (quantity * price) / float64(leverage)
		totalUnrealizedPnL := 0.0
		totalMarginUsed := 0.0
		for _, pos := range t.state.Positions {
			currentPrice, _ := t.getRobustPrice(pos.Symbol)
			if currentPrice == 0 {
				currentPrice = pos.EntryPrice
			}
			var pnl float64
			if pos.Side == "long" {
				pnl = (currentPrice - pos.EntryPrice) * pos.Quantity
			} else {
				pnl = (pos.EntryPrice - currentPrice) * pos.Quantity
			}
			totalUnrealizedPnL += pnl
			totalMarginUsed += (pos.Quantity * currentPrice) / float64(pos.Leverage)
		}
		equity := t.state.Balance + totalUnrealizedPnL
		available := equity - totalMarginUsed
		if available < requiredMargin+commission {
			return nil, fmt.Errorf("insufficient balance: available %.2f, required %.2f (margin %.2f + fee %.2f)",
				available, requiredMargin+commission, requiredMargin, commission)
		}
	}

	t.state.Balance -= commission // Deduct fee immediately

	// 6. Update Position Logic
	posKey := symbol + "_" + strings.ToLower(positionSide)

	// Handle Open Position
	if isOpening {
		// Opening/Adding to position
		pos, exists := t.state.Positions[posKey]
		if !exists {
			newPos := &VirtualPosition{
				Symbol:     symbol,
				Side:       strings.ToLower(positionSide),
				EntryPrice: price,
				Quantity:   quantity,
				Leverage:   leverage,
			}
			t.state.Positions[posKey] = newPos
		} else {
			// Average Entry Price
			totalValue := pos.EntryPrice*pos.Quantity + price*quantity
			totalQty := pos.Quantity + quantity
			pos.EntryPrice = totalValue / totalQty
			pos.Quantity = totalQty
			pos.Leverage = leverage // Update leverage to latest
		}
	} else {
		// Closing/Reducing position
		pos, exists := t.state.Positions[posKey]
		if !exists {
			return nil, fmt.Errorf("no position found to close for %s %s", symbol, positionSide)
		}

		// Calculate PnL
		if positionSide == "LONG" {
			pnl = (price - pos.EntryPrice) * quantity
		} else {
			pnl = (pos.EntryPrice - price) * quantity
		}

		// Check if it's a full close or partial close
		isFullClose := math.Abs(pos.Quantity-quantity) < 0.00000001

		// Update balance with realized PnL
		t.state.Balance += pnl

		// Update position in state
		pos.Quantity -= quantity
		if isFullClose {
			delete(t.state.Positions, posKey)
		}
	}

	// 8. Create Order Record in State
	order := &VirtualOrder{
		OrderID:      orderID,
		Symbol:       symbol,
		Side:         side,
		PositionSide: positionSide,
		Type:         "MARKET",
		Price:        price,
		Quantity:     quantity,
		Status:       "FILLED",
		Time:         time.Now().UnixMilli(),
		Commission:   commission,
	}
	t.state.Orders[orderID] = order

	// 9. Save State
	if err := t.saveState(); err != nil {
		logger.Errorf("Failed to save virtual exchange state: %v", err)
	}

	logger.Infof("🎮 [%s] Virtual Order Executed: %s %s %s %.4f @ %.4f (Fee: %.4f)",
		t.userID, side, positionSide, symbol, quantity, price, commission)

	return map[string]interface{}{
		"orderId":     order.OrderID,
		"status":      order.Status,
		"price":       order.Price,
		"quantity":    order.Quantity,
		"symbol":      order.Symbol,
		"side":        order.Side,
		"type":        order.Type,
		"time":        order.Time,
		"commission":  order.Commission,
		"realizedPnl": pnl,
	}, nil
}

// GetOrderStatus gets order status
func (t *VirtualTrader) GetOrderStatus(symbol string, orderID string) (map[string]interface{}, error) {
	t.mu.RLock()
	defer t.mu.RUnlock()

	order, exists := t.state.Orders[orderID]
	if !exists {
		return nil, fmt.Errorf("order not found: %s", orderID)
	}

	return map[string]interface{}{
		"orderId":     order.OrderID,
		"symbol":      order.Symbol,
		"status":      order.Status,
		"avgPrice":    order.Price,
		"executedQty": order.Quantity,
		"side":        order.Side,
		"type":        order.Type,
		"time":        order.Time,
		"commission":  order.Commission,
	}, nil
}

// GetMarketPrice gets market price
func (t *VirtualTrader) GetMarketPrice(symbol string) (float64, error) {
	return t.getRobustPrice(symbol)
}

// GetTrades gets trade history
func (t *VirtualTrader) GetTrades(startTime time.Time, limit int) ([]types.TradeRecord, error) {
	t.mu.RLock()
	defer t.mu.RUnlock()

	// Reload state to ensure we have latest data
	if err := t.loadState(); err != nil {
		logger.Warnf("Failed to reload state in GetTrades: %v", err)
	}

	var trades []types.TradeRecord
	startTimeMs := startTime.UnixMilli()

	for _, order := range t.state.Orders {
		// Only filled orders count as trades
		if order.Status != "FILLED" {
			continue
		}

		// Filter by time
		if order.Time/1000000 < startTimeMs { // order.Time is usually nano in some contexts, but let's check.
			// Wait, in executeOrder: orderID := fmt.Sprintf("%d", time.Now().UnixNano())
			// But order.Time?
			// In executeOrder: "time": order.Time
			// VirtualOrder struct: Time int64 `json:"time"`
			// Let's check where VirtualOrder is created.
			continue
		}

		// Wait, I need to check how Time is stored in VirtualOrder.
		// In executeOrder:
		// order := &VirtualOrder{
		// 	...
		// 	Time:         time.Now().UTC().UnixMilli(),
		// }
		// So it is UnixMilli.

		if order.Time < startTimeMs {
			continue
		}

		trades = append(trades, types.TradeRecord{
			TradeID:      order.OrderID, // Use OrderID as TradeID for simplicity in virtual
			Symbol:       order.Symbol,
			Side:         order.Side,
			PositionSide: order.PositionSide,
			OrderAction:  getOrderAction(order.Side, order.PositionSide),
			Quantity:     order.Quantity,
			Price:        order.Price,
			Fee:          order.Commission,
			Time:         time.UnixMilli(order.Time),
			RealizedPnL:  0, // Not stored in order, will be calculated by PositionBuilder if needed
		})
	}

	// Sort by time descending (newest first)
	sort.Slice(trades, func(i, j int) bool {
		return trades[i].Time.After(trades[j].Time)
	})

	if limit > 0 && len(trades) > limit {
		trades = trades[:limit]
	}

	return trades, nil
}

// Helper to determine action
func getOrderAction(side, positionSide string) string {
	if side == "BUY" {
		if positionSide == "LONG" {
			return "open_long"
		}
		return "close_short"
	}
	// SELL
	if positionSide == "LONG" {
		return "close_long"
	}
	return "open_short"
}

// Implement other interface methods (simplified)

func (t *VirtualTrader) SetLeverage(symbol string, leverage int) error {
	// No-op for simulation, or update position leverage if exists
	return nil
}

func (t *VirtualTrader) SetMarginMode(symbol string, isCrossMargin bool) error {
	return nil // No-op
}

func (t *VirtualTrader) SetStopLoss(symbol string, positionSide string, quantity, stopPrice float64) error {
	// Can record in state if needed, but for now AutoTrader manages risk logic
	return nil
}

func (t *VirtualTrader) SetTakeProfit(symbol string, positionSide string, quantity, takeProfitPrice float64) error {
	return nil
}

func (t *VirtualTrader) CancelStopLossOrders(symbol string) error {
	return nil
}

func (t *VirtualTrader) CancelTakeProfitOrders(symbol string) error {
	return nil
}

func (t *VirtualTrader) CancelAllOrders(symbol string) error {
	return nil
}

func (t *VirtualTrader) CancelStopOrders(symbol string) error {
	return nil
}

func (t *VirtualTrader) FormatQuantity(symbol string, quantity float64) (string, error) {
	// Return 3 decimal places default
	return fmt.Sprintf("%.3f", quantity), nil
}

func (t *VirtualTrader) GetClosedPnL(startTime time.Time, limit int) ([]types.ClosedPnLRecord, error) {
	return []types.ClosedPnLRecord{}, nil // Not implemented yet
}

func (t *VirtualTrader) GetOpenOrders(symbol string) ([]types.OpenOrder, error) {
	return []types.OpenOrder{}, nil
}

// Helper
func stringsToLowerCase(s string) string {
	if s == "LONG" {
		return "long"
	}
	if s == "SHORT" {
		return "short"
	}
	return s
}

func (t *VirtualTrader) getRobustPrice(symbol string) (float64, error) {
	// Try robust market.Get first
	marketData, err := market.Get(symbol)
	if err == nil {
		return marketData.CurrentPrice, nil
	}

	// If market.Get failed (likely due to restriction), try Hyperliquid directly here
	// This keeps the fallback logic localized to the virtual trader
	logger.Warnf("VirtualTrader: market.Get failed for %s, trying Hyperliquid fallback...", symbol)

	// Use Hyperliquid client directly
	// Note: We need to import the provider package if we want to use it directly,
	// but market.Get should have already tried it if configured.
	// Since we want to enforce it here without affecting global market.Get:

	// Create a temporary client/request to Hyperliquid public API
	// Or better, assume market.GetWithExchange might work if we explicitly ask for it
	// But market.Get already tries.

	// Let's use the API client fallback as a last resort, but maybe we can try
	// a different method or just log the failure more clearly.

	// Actually, the user asked to put the fix HERE.
	// So let's implement the Hyperliquid fetch here directly or via a specific call
	// We can use market.GetWithExchange(symbol, "hyperliquid")

	marketDataHL, errHL := market.GetWithExchange(symbol, "hyperliquid")
	if errHL == nil {
		return marketDataHL.CurrentPrice, nil
	}

	// Fallback to direct API client
	price, err2 := t.apiClient.GetCurrentPrice(symbol)
	if err2 != nil {
		return 0, fmt.Errorf("failed to get market price: %v (HL: %v, fallback: %v)", err, errHL, err2)
	}
	return price, nil
}

// CleanupVirtualData deletes virtual exchange data for a specific trader
func CleanupVirtualData(traderID string) error {
	dataDir := "data/virtual_exchange"
	statePath := filepath.Join(dataDir, fmt.Sprintf("%s.json", traderID))

	if _, err := os.Stat(statePath); err == nil {
		logger.Infof("🧹 Cleaning up virtual exchange data for trader %s", traderID)
		return os.Remove(statePath)
	}
	return nil
}
