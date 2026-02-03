package store

import (
	"testing"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupOrderTestDB(t *testing.T) *OrderStore {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to connect database: %v", err)
	}

	orderStore := NewOrderStore(db)
	if err := orderStore.InitTables(); err != nil {
		t.Fatalf("failed to init tables: %v", err)
	}

	return orderStore
}

func TestGetTransactions(t *testing.T) {
	s := setupOrderTestDB(t)

	// Create some fills
	fills := []*TraderFill{
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
	// GORM Find returns initialized slice if using var slice []Type
	// But here GetTransactions uses var fills []*TraderFill which is nil
	// So results should be empty slice (GORM initializes it) or nil?
	// My previous test showed [] but that was using Find(&items) where items was initialized?
	// No, var items []*Item -> nil.
	// So results should be [] (non-nil empty slice).
	if results == nil {
		// If it is nil, it will be marshaled as null in JSON
		t.Logf("results is nil (will be null in JSON)")
	} else {
		t.Logf("results is not nil, len=%d", len(results))
	}
}
