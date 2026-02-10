package virtual

import (
	"encoding/json"
	"fmt"
	"nofx/logger"
	"nofx/market"
	"nofx/trader/types"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// VirtualPosition represents a position in the virtual exchange
type VirtualPosition struct {
	Symbol        string  `json:"symbol"`
	Side          string  `json:"side"` // "long" or "short"
	EntryPrice    float64 `json:"entry_price"`
	Quantity      float64 `json:"quantity"`
	Leverage      int     `json:"leverage"`
	UnrealizedPnL float64 `json:"-"` // Calculated runtime, not stored
	Margin        float64 `json:"-"` // Calculated runtime, not stored
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
	state          *VirtualExchangeState
	apiClient      *market.APIClient
	statePath      string
	mu             sync.RWMutex
	initialBalance float64
}

// NewVirtualTrader creates a new virtual trader
func NewVirtualTrader(userID string, initialBalance float64) *VirtualTrader {
	// Ensure data directory exists
	dataDir := "data/virtual_exchange"
	if err := os.MkdirAll(dataDir, 0755); err != nil {
		logger.Errorf("Failed to create virtual exchange data directory: %v", err)
	}

	statePath := filepath.Join(dataDir, fmt.Sprintf("%s.json", userID))

	trader := &VirtualTrader{
		userID:         userID,
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
	availableBalance := t.state.Balance - totalMarginUsed // Simplified margin logic

	return map[string]interface{}{
		"totalWalletBalance":    t.state.Balance,
		"totalUnrealizedProfit": totalUnrealizedPnL,
		"availableBalance":      availableBalance,
		"totalEquity":           totalEquity,
	}, nil
}

// GetPositions gets all positions
func (t *VirtualTrader) GetPositions() ([]map[string]interface{}, error) {
	t.mu.RLock()
	defer t.mu.RUnlock()

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

	// 2. Generate Order ID
	orderID := fmt.Sprintf("%d", time.Now().UnixNano())

	// 3. Calculate Fee (0.05% taker fee)
	feeRate := 0.0005
	commission := quantity * price * feeRate
	t.state.Balance -= commission // Deduct fee immediately

	// 4. Update Position Logic
	posKey := symbol + "_" + stringsToLowerCase(positionSide)

	// Handle Open Position
	if (side == "BUY" && positionSide == "LONG") || (side == "SELL" && positionSide == "SHORT") {
		// Opening/Adding to position
		pos, exists := t.state.Positions[posKey]
		if !exists {
			t.state.Positions[posKey] = &VirtualPosition{
				Symbol:     symbol,
				Side:       stringsToLowerCase(positionSide),
				EntryPrice: price,
				Quantity:   quantity,
				Leverage:   leverage,
			}
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
			// Trying to close non-existent position, treat as error or ignore
			// For simulation, let's error to be safe
			return nil, fmt.Errorf("no position found to close for %s %s", symbol, positionSide)
		}

		// Calculate PnL
		var pnl float64
		if positionSide == "LONG" {
			pnl = (price - pos.EntryPrice) * quantity
		} else {
			pnl = (pos.EntryPrice - price) * quantity
		}
		t.state.Balance += pnl // Add realized PnL

		// Update position
		pos.Quantity -= quantity
		if pos.Quantity <= 0.00000001 { // Float epsilon
			delete(t.state.Positions, posKey)
		}
	}

	// 5. Create Order Record
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

	// 6. Save State
	if err := t.saveState(); err != nil {
		logger.Errorf("Failed to save virtual exchange state: %v", err)
	}

	logger.Infof("🎮 [%s] Virtual Order Executed: %s %s %s %.4f @ %.4f (Fee: %.4f)",
		t.userID, side, positionSide, symbol, quantity, price, commission)

	return map[string]interface{}{
		"orderId": orderID,
		"symbol":  symbol,
		"status":  "FILLED",
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
