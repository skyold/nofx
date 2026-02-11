package tests

import (
	"nofx/store"
	"testing"
	"time"

	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

func setupTestDB(t *testing.T) *gorm.DB {
	db, err := gorm.Open(sqlite.Open(":memory:"), &gorm.Config{})
	if err != nil {
		t.Fatalf("failed to connect database: %v", err)
	}

	if err := db.AutoMigrate(&store.TraderPosition{}, &store.TraderFill{}, &store.TraderOrder{}); err != nil {
		t.Fatalf("failed to migrate database: %v", err)
	}

	return db
}

func TestRebuildFromFills_PhantomPosition(t *testing.T) {
	db := setupTestDB(t)
	posStore := store.NewPositionStore(db)
	orderStore := store.NewOrderStore(db)
	pb := store.NewPositionBuilder(posStore)

	traderID := "trader1"
	symbol := "BTCUSDT"

	// 1. Create a Phantom Open Position (exists in DB but no fills supporting it)
	phantomPos := &store.TraderPosition{
		TraderID:   traderID,
		Symbol:     symbol,
		Side:       "LONG",
		Quantity:   1.0,
		EntryPrice: 50000,
		Status:     "OPEN",
		CreatedAt:  store.UnixTime(time.Now().UnixMilli()),
	}
	if err := db.Create(phantomPos).Error; err != nil {
		t.Fatalf("failed to create phantom position: %v", err)
	}

	// Verify it exists
	var count int64
	db.Model(&store.TraderPosition{}).Where("trader_id = ? AND status = ?", traderID, "OPEN").Count(&count)
	if count != 1 {
		t.Fatalf("expected 1 open position, got %d", count)
	}

	// 2. Run RebuildFromFills (with NO fills)
	// We pass orderStore
	rebuiltCount, err := pb.RebuildFromFills(traderID, orderStore)
	if err != nil {
		t.Fatalf("RebuildFromFills failed: %v", err)
	}

	// 3. Verify results
	// Should have 0 closed positions created (rebuiltCount)
	if rebuiltCount != 0 {
		t.Errorf("expected 0 rebuilt closed positions, got %d", rebuiltCount)
	}

	// Crucially: The Phantom Open Position should be DELETED
	db.Model(&store.TraderPosition{}).Where("trader_id = ? AND status = ?", traderID, "OPEN").Count(&count)
	if count != 0 {
		t.Errorf("expected 0 open positions after rebuild (phantom deletion), got %d", count)
	}
}

func TestRebuildFromFills_ValidOpenPosition(t *testing.T) {
	db := setupTestDB(t)
	posStore := store.NewPositionStore(db)
	orderStore := store.NewOrderStore(db)
	pb := store.NewPositionBuilder(posStore)

	traderID := "trader1"
	symbol := "ETHUSDT"

	// 1. Create a Fill (Open Long)
	fill := &store.TraderFill{
		TraderID:        traderID,
		Symbol:          symbol,
		Side:            "BUY",
		Price:           3000,
		Quantity:        10,
		Commission:      1,
		CommissionAsset: "USDT",
		RealizedPnL:     0, // 0 implies Open
		ExchangeID:      "binance",
		CreatedAt:       store.UnixTime(time.Now().UnixMilli()),
	}
	if err := db.Create(fill).Error; err != nil {
		t.Fatalf("failed to create fill: %v", err)
	}

	// 2. Run RebuildFromFills
	_, err := pb.RebuildFromFills(traderID, orderStore)
	if err != nil {
		t.Fatalf("RebuildFromFills failed: %v", err)
	}

	// 3. Verify Open Position Created
	var pos store.TraderPosition
	err = db.Where("trader_id = ? AND status = ?", traderID, "OPEN").First(&pos).Error
	if err != nil {
		t.Fatalf("expected open position to be created, got error: %v", err)
	}

	if pos.Symbol != symbol {
		t.Errorf("expected symbol %s, got %s", symbol, pos.Symbol)
	}
	if pos.Quantity != 10 {
		t.Errorf("expected quantity 10, got %f", pos.Quantity)
	}
}

func TestRebuildFromFills_ClosedPosition_ClearsOpen(t *testing.T) {
	db := setupTestDB(t)
	posStore := store.NewPositionStore(db)
	orderStore := store.NewOrderStore(db)
	pb := store.NewPositionBuilder(posStore)

	traderID := "trader1"
	symbol := "SOLUSDT"

	// 1. Create an existing Open Position (maybe with wrong qty)
	existingPos := &store.TraderPosition{
		TraderID: traderID,
		Symbol:   symbol,
		Side:     "LONG",
		Quantity: 50.0, // Wrong quantity
		Status:   "OPEN",
	}
	db.Create(existingPos)

	// 2. Create Fills (Open + Close)
	t1 := time.Now().Add(-1 * time.Hour).UnixMilli()
	t2 := time.Now().UnixMilli()

	fills := []store.TraderFill{
		{
			TraderID:        traderID,
			Symbol:          symbol,
			Side:            "BUY",
			Price:           100,
			Quantity:        10,
			RealizedPnL:     0,
			ExchangeTradeID: "trade1",
			CreatedAt:       store.UnixTime(t1),
		},
		{
			TraderID:        traderID,
			Symbol:          symbol,
			Side:            "SELL",
			Price:           110,
			Quantity:        10,
			RealizedPnL:     100, // Closed
			ExchangeOrderID: "ord1",
			ExchangeTradeID: "trade2",
			CreatedAt:       store.UnixTime(t2),
		},
	}
	for _, f := range fills {
		db.Create(&f)
	}

	// 3. Run RebuildFromFills
	rebuiltCount, err := pb.RebuildFromFills(traderID, orderStore)
	if err != nil {
		t.Fatalf("RebuildFromFills failed: %v", err)
	}

	// 4. Verify
	// Should process 2 fills
	if rebuiltCount != 2 {
		t.Errorf("expected 2 rebuilt fills, got %d", rebuiltCount)
	}

	// Should have NO open positions (because fills fully closed it)
	var count int64
	db.Model(&store.TraderPosition{}).Where("trader_id = ? AND status = ?", traderID, "OPEN").Count(&count)
	if count != 0 {
		t.Errorf("expected 0 open positions, got %d. Phantom/Existing position was not cleared!", count)
	}
}

func TestRebuildFromFills_ZeroPnL_Close(t *testing.T) {
	db := setupTestDB(t)
	posStore := store.NewPositionStore(db)
	orderStore := store.NewOrderStore(db)
	pb := store.NewPositionBuilder(posStore)

	traderID := "trader1"
	symbol := "BTCUSDT"

	t1 := time.Now().Add(-1 * time.Hour).UnixMilli()
	t2 := time.Now().UnixMilli()

	// 1. Create Orders (Open and Close)
	// We need orders to disambiguate 0 PnL close
	orders := []store.TraderOrder{
		{
			ID:              1,
			TraderID:        traderID,
			Symbol:          symbol,
			ExchangeOrderID: "ord_open",
			Side:            "BUY",
			OrderAction:     "open_long",
			PositionSide:    "LONG",
			Status:          "FILLED",
			CreatedAt:       t1,
		},
		{
			ID:              2,
			TraderID:        traderID,
			Symbol:          symbol,
			ExchangeOrderID: "ord_close",
			Side:            "SELL",
			OrderAction:     "close_long",
			PositionSide:    "LONG",
			Status:          "FILLED",
			CreatedAt:       t2,
		},
	}
	for _, o := range orders {
		db.Create(&o)
	}

	// 2. Create Fills (Open and Close with 0 PnL)
	fills := []store.TraderFill{
		{
			TraderID:        traderID,
			Symbol:          symbol,
			Side:            "BUY",
			Price:           50000,
			Quantity:        1,
			RealizedPnL:     0,
			OrderID:         1, // Links to Open Order
			ExchangeOrderID: "ord_open",
			ExchangeTradeID: "trade1",
			CreatedAt:       t1,
		},
		{
			TraderID:        traderID,
			Symbol:          symbol,
			Side:            "SELL",
			Price:           50000, // Break even
			Quantity:        1,
			RealizedPnL:     0, // 0 PnL! This caused the bug
			OrderID:         2, // Links to Close Order
			ExchangeOrderID: "ord_close",
			ExchangeTradeID: "trade2",
			CreatedAt:       t2,
		},
	}
	for _, f := range fills {
		db.Create(&f)
	}

	// 3. Run RebuildFromFills
	rebuiltCount, err := pb.RebuildFromFills(traderID, orderStore)
	if err != nil {
		t.Fatalf("RebuildFromFills failed: %v", err)
	}

	// 4. Verify
	// Should process 2 fills
	if rebuiltCount != 2 {
		t.Errorf("expected 2 rebuilt fills, got %d", rebuiltCount)
	}

	// Should have NO open positions
	var count int64
	db.Model(&store.TraderPosition{}).Where("trader_id = ? AND status = ?", traderID, "OPEN").Count(&count)
	if count != 0 {
		t.Errorf("expected 0 open positions, got %d. Zero PnL Close was treated as Open!", count)
	}
}
