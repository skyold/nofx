package tests

import (
	"nofx/store"
	"testing"
	"time"
)

func TestGetTransactions(t *testing.T) {
	db := setupTestDB(t)
	s := store.NewOrderStore(db)

	// Create some fills
	fills := []*store.TraderFill{
		{
			TraderID:        "trader1",
			ExchangeID:      "binance",
			ExchangeType:    "cex",
			ExchangeOrderID: "1",
			ExchangeTradeID: "t1",
			Symbol:          "BTCUSDT",
			Side:            "BUY",
			Price:           50000,
			Quantity:        1,
			Commission:      0.1,
			CommissionAsset: "USDT",
			CreatedAt:       1000,
		},
		{
			TraderID:        "trader2",
			ExchangeID:      "binance",
			ExchangeType:    "cex",
			ExchangeOrderID: "2",
			ExchangeTradeID: "t2",
			Symbol:          "ETHUSDT",
			Side:            "SELL",
			Price:           3000,
			Quantity:        10,
			Commission:      0.1,
			CommissionAsset: "USDT",
			CreatedAt:       2000,
		},
	}

	for _, f := range fills {
		if err := s.CreateFill(f); err != nil {
			t.Fatalf("failed to create fill: %v", err)
		}
	}

	// Test 1: Get all
	results, total, err := s.GetTransactions(1, 10, "", "", "desc")
	if err != nil {
		t.Fatalf("GetTransactions failed: %v", err)
	}
	if total != 2 {
		t.Errorf("expected 2 transactions, got %d", total)
	}
	if len(results) != 2 {
		t.Errorf("expected 2 results, got %d", len(results))
	}

	// Test 2: Filter by TraderID
	results, total, err = s.GetTransactions(1, 10, "", "trader1", "desc")
	if err != nil {
		t.Fatalf("GetTransactions failed: %v", err)
	}
	if total != 1 {
		t.Errorf("expected 1 transaction, got %d", total)
	}
	if results[0].TraderID != "trader1" {
		t.Errorf("expected trader1, got %s", results[0].TraderID)
	}

	// Test 3: Pagination
	results, total, err = s.GetTransactions(1, 1, "", "", "desc")
	if total != 2 {
		t.Errorf("expected 2 total, got %d", total)
	}
	if len(results) != 1 {
		t.Errorf("expected 1 result, got %d", len(results))
	}

	// Test 4: Empty result
	results, total, err = s.GetTransactions(1, 10, "", "trader_non_exist", "desc")
	if total != 0 {
		t.Errorf("expected 0 total, got %d", total)
	}
	if results == nil {
		// Verify if it returns nil or empty slice
		// t.Logf("results is nil")
	} else {
		if len(results) != 0 {
			t.Errorf("expected 0 results, got %d", len(results))
		}
	}
}

func TestClearHistory(t *testing.T) {
	db := setupTestDB(t)
	s := store.NewOrderStore(db)

	traderID := "trader_clear"

	// Create order and fill
	order := &store.TraderOrder{
		TraderID:        traderID,
		ExchangeOrderID: "ord1",
		Symbol:          "BTCUSDT",
		Status:          "FILLED",
		CreatedAt:       store.UnixTime(time.Now().UnixMilli()),
	}
	db.Create(order)

	fill := &store.TraderFill{
		TraderID:        traderID,
		ExchangeOrderID: "ord1",
		ExchangeTradeID: "trade1",
		Symbol:          "BTCUSDT",
		CreatedAt:       store.UnixTime(time.Now().UnixMilli()),
	}
	db.Create(fill)

	// Clear history
	if err := s.ClearHistory(traderID); err != nil {
		t.Fatalf("ClearHistory failed: %v", err)
	}

	// Verify deleted
	var count int64
	db.Model(&store.TraderOrder{}).Where("trader_id = ?", traderID).Count(&count)
	if count != 0 {
		t.Errorf("expected 0 orders, got %d", count)
	}

	db.Model(&store.TraderFill{}).Where("trader_id = ?", traderID).Count(&count)
	if count != 0 {
		t.Errorf("expected 0 fills, got %d", count)
	}
}
