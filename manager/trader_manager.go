package manager

import (
	"encoding/json"
	"fmt"
	chaos "nofx/engine/llm"
	"nofx/logger"
	"nofx/scheduler"
	"nofx/store"
	"nofx/trader"
	"sync"
	"time"
)

// CompetitionCache competition data cache
type CompetitionCache struct {
	data      map[string]interface{}
	timestamp time.Time
	mu        sync.RWMutex
}

// TraderManager manages multiple scheduler instances
type TraderManager struct {
	schedulers       map[string]scheduler.Scheduler // key: trader ID
	loadErrors       map[string]error               // key: trader ID, stores last load error
	competitionCache *CompetitionCache
	mu               sync.RWMutex
}

// NewTraderManager creates a trader manager
func NewTraderManager() *TraderManager {
	return &TraderManager{
		schedulers: make(map[string]scheduler.Scheduler),
		loadErrors: make(map[string]error),
		competitionCache: &CompetitionCache{
			data: make(map[string]interface{}),
		},
	}
}

// LoadTradersFromStore loads all traders from the database and creates schedulers
func (tm *TraderManager) LoadTradersFromStore(st *store.Store) error {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	logger.Infof("📦 Loading traders from store...")

	// Get all trader configurations
	traders, err := st.Trader().ListAll()
	if err != nil {
		return fmt.Errorf("failed to load traders: %w", err)
	}

	loadedCount := 0
	for _, trader := range traders {
		if err := tm.loadSingleTrader(st, trader); err != nil {
			logger.Infof("⚠️  Failed to load trader %s: %v", trader.ID, err)
			tm.loadErrors[trader.ID] = err
		} else {
			loadedCount++
		}
	}

	logger.Infof("✓ Loaded %d traders", loadedCount)
	return nil
}

// loadSingleTrader loads a single trader and creates a scheduler
func (tm *TraderManager) loadSingleTrader(st *store.Store, t *store.Trader) error {
	traderID := t.ID

	// Create TraderConfig from Trader
	autoTraderConfig := tm.convertToTraderConfig(t, st)

	// Create Trader (encapsulates exchange creation logic)
	autoTrader, err := trader.NewTrader(autoTraderConfig, st, t.UserID)
	if err != nil {
		return fmt.Errorf("failed to create Trader: %w", err)
	}

	// Create Engine based on strategy type
	var engine scheduler.Engine
	switch t.StrategyID {
	case "chaos":
		chaosEngine := chaos.NewChaosEngine(autoTraderConfig.StrategyConfig)
		// Set dependencies for BuildContext
		chaosEngine.SetDependencies(
			autoTrader,
			nil, // strategyEngine - TODO: pass actual strategy engine
			st,
			traderID,
			time.Now(),
			0, // callCount
		)
		engine = chaosEngine
	case "grid":
		// TODO: Create Grid Engine
		return fmt.Errorf("grid strategy not yet implemented")
	case "nofx":
		// TODO: Create Nofx Engine
		return fmt.Errorf("nofx strategy not yet implemented")
	default:
		logger.Infof("ℹ️  Using default chaos strategy for trader %s", traderID)
		chaosEngine := chaos.NewChaosEngine(autoTraderConfig.StrategyConfig)
		chaosEngine.SetDependencies(
			autoTrader,
			nil,
			st,
			traderID,
			time.Now(),
			0,
		)
		engine = chaosEngine
	}

	// Create Scheduler
	sched := scheduler.NewScheduler(autoTraderConfig.ScanInterval)
	sched.SetEngine(engine)
	sched.SetTrader(autoTrader)
	sched.SetStore(st)

	// Save scheduler
	tm.schedulers[traderID] = sched
	delete(tm.loadErrors, traderID)

	logger.Infof("✓ Loaded trader: %s (%s)", autoTraderConfig.Name, traderID)
	return nil
}

// convertToTraderConfig converts Trader to TraderConfig
func (tm *TraderManager) convertToTraderConfig(t *store.Trader, st *store.Store) trader.TraderConfig {
	// Get strategy config
	var strategyConfig store.StrategyConfig
	if t.StrategyID != "" {
		strategy, err := st.Strategy().Get(t.UserID, t.StrategyID)
		if err != nil || strategy == nil {
			logger.Warnf("⚠️  Failed to get strategy %s, using default: %v", t.StrategyID, err)
			strategyConfig = store.GetDefaultStrategyConfig("en")
		} else {
			// Parse strategy config from JSON string
			if err := json.Unmarshal([]byte(strategy.Config), &strategyConfig); err != nil {
				logger.Warnf("⚠️  Failed to parse strategy config, using default: %v", err)
				strategyConfig = store.GetDefaultStrategyConfig("en")
			}
		}
	} else {
		strategyConfig = store.GetDefaultStrategyConfig("en")
	}

	// Get exchange config for API keys
	exchange, err := st.Exchange().GetByID(t.UserID, t.ExchangeID)
	if err != nil {
		logger.Warnf("⚠️  Failed to get exchange config for %s: %v", t.ExchangeID, err)
	}

	// Get exchange type from exchange config
	exchangeType := ""
	if exchange != nil {
		exchangeType = exchange.ExchangeType
	}

	// Build API key config (with decryption)
	apiKey := ""
	secretKey := ""
	passphrase := ""
	hyperliquidWalletAddr := ""
	hyperliquidUnifiedAcct := true
	hyperliquidPrivateKey := ""
	asterUser := ""
	asterSigner := ""
	asterPrivateKey := ""
	lighterWalletAddr := ""
	lighterPrivateKey := ""
	lighterAPIKeyPrivateKey := ""
	lighterAPIKeyIndex := 0

	if exchange != nil {
		apiKey = string(exchange.APIKey)
		secretKey = string(exchange.SecretKey)
		passphrase = string(exchange.Passphrase)
		hyperliquidWalletAddr = exchange.HyperliquidWalletAddr
		hyperliquidUnifiedAcct = exchange.HyperliquidUnifiedAcct
		hyperliquidPrivateKey = string(exchange.HyperliquidPrivateKey)
		asterUser = exchange.AsterUser
		asterSigner = exchange.AsterSigner
		asterPrivateKey = string(exchange.AsterPrivateKey)
		lighterWalletAddr = exchange.LighterWalletAddr
		lighterPrivateKey = string(exchange.LighterPrivateKey)
		lighterAPIKeyPrivateKey = string(exchange.LighterAPIKeyPrivateKey)
		lighterAPIKeyIndex = exchange.LighterAPIKeyIndex
	}

	return trader.TraderConfig{
		ID:                      t.ID,
		Name:                    t.Name,
		Exchange:                exchangeType, // 从 Exchange 获取的交易所类型
		ExchangeID:              t.ExchangeID,
		ScanInterval:            time.Duration(t.ScanIntervalMinutes) * time.Minute,
		InitialBalance:          t.InitialBalance,
		IsCrossMargin:           t.IsCrossMargin,
		ShowInCompetition:       t.ShowInCompetition,
		StrategyConfig:          &strategyConfig,
		APIKey:                  apiKey,
		SecretKey:               secretKey,
		Passphrase:              passphrase,
		Testnet:                 exchange != nil && exchange.Testnet,
		HyperliquidWalletAddr:   hyperliquidWalletAddr,
		HyperliquidUnifiedAcct:  hyperliquidUnifiedAcct,
		HyperliquidPrivateKey:   hyperliquidPrivateKey,
		AsterUser:               asterUser,
		AsterSigner:             asterSigner,
		AsterPrivateKey:         asterPrivateKey,
		LighterWalletAddr:       lighterWalletAddr,
		LighterPrivateKey:       lighterPrivateKey,
		LighterAPIKeyPrivateKey: lighterAPIKeyPrivateKey,
		LighterAPIKeyIndex:      lighterAPIKeyIndex,
	}
}

// AutoStartRunningTraders automatically starts traders marked as running
func (tm *TraderManager) AutoStartRunningTraders(st *store.Store) {
	// Get all traders
	traders, err := st.Trader().ListAll()
	if err != nil {
		logger.Infof("⚠️  Failed to get trader list: %v", err)
		return
	}

	// Build set of running trader IDs
	runningTraderIDs := make(map[string]bool)
	for _, t := range traders {
		if t.IsRunning {
			runningTraderIDs[t.ID] = true
		}
	}

	if len(runningTraderIDs) == 0 {
		logger.Info("📋 No traders to auto-restore")
		return
	}

	tm.mu.RLock()
	defer tm.mu.RUnlock()

	startedCount := 0
	for id, sched := range tm.schedulers {
		if runningTraderIDs[id] {
			go func(traderID string, s scheduler.Scheduler) {
				logger.Infof("▶️  Auto-restoring %s...", traderID)
				if err := s.Start(); err != nil {
					logger.Infof("❌ %s runtime error: %v", traderID, err)
				}
			}(id, sched)
			startedCount++
		}
	}

	if startedCount > 0 {
		logger.Infof("✓ Auto-restored %d traders", startedCount)
	}
}

// StartAll starts all traders
func (tm *TraderManager) StartAll() {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	logger.Info("🚀 Starting all traders...")
	for id, sched := range tm.schedulers {
		go func(traderID string, s scheduler.Scheduler) {
			logger.Infof("▶️  Starting %s...", traderID)
			if err := s.Start(); err != nil {
				logger.Infof("❌ %s runtime error: %v", traderID, err)
			}
		}(id, sched)
	}
}

// StopAll stops all traders
func (tm *TraderManager) StopAll() {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	logger.Info("⏹  Stopping all traders...")
	for _, sched := range tm.schedulers {
		if err := sched.Stop(); err != nil {
			logger.Infof("⚠️  Error stopping scheduler: %v", err)
		}
	}
}

// StartTrader starts a specific trader
func (tm *TraderManager) StartTrader(traderID string) error {
	tm.mu.RLock()
	sched, exists := tm.schedulers[traderID]
	tm.mu.RUnlock()

	if !exists {
		return fmt.Errorf("trader %s not found", traderID)
	}

	logger.Infof("▶️  Starting trader %s...", traderID)
	return sched.Start()
}

// StopTrader stops a specific trader
func (tm *TraderManager) StopTrader(traderID string) error {
	tm.mu.RLock()
	sched, exists := tm.schedulers[traderID]
	tm.mu.RUnlock()

	if !exists {
		return fmt.Errorf("trader %s not found", traderID)
	}

	logger.Infof("⏹  Stopping trader %s...", traderID)
	return sched.Stop()
}

// GetScheduler retrieves a scheduler by ID
func (tm *TraderManager) GetScheduler(id string) (scheduler.Scheduler, error) {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	sched, exists := tm.schedulers[id]
	if !exists {
		return nil, fmt.Errorf("scheduler ID '%s' does not exist", id)
	}
	return sched, nil
}

// GetAllSchedulers retrieves all schedulers
func (tm *TraderManager) GetAllSchedulers() map[string]scheduler.Scheduler {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	result := make(map[string]scheduler.Scheduler)
	for id, sched := range tm.schedulers {
		result[id] = sched
	}
	return result
}

// GetTraderIDs retrieves all trader IDs
func (tm *TraderManager) GetTraderIDs() []string {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	ids := make([]string, 0, len(tm.schedulers))
	for id := range tm.schedulers {
		ids = append(ids, id)
	}
	return ids
}

// GetLoadError returns the last load error for a trader
func (tm *TraderManager) GetLoadError(traderID string) error {
	tm.mu.RLock()
	defer tm.mu.RUnlock()
	return tm.loadErrors[traderID]
}

// GetComparisonData retrieves comparison data
func (tm *TraderManager) GetComparisonData() (map[string]interface{}, error) {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	comparison := make(map[string]interface{})
	traders := make([]map[string]interface{}, 0, len(tm.schedulers))

	for _, sched := range tm.schedulers {
		// TODO: Get trader info from scheduler
		// This requires accessing the trader through scheduler
		// For now, skip implementation
		_ = sched
	}

	comparison["traders"] = traders
	comparison["count"] = len(traders)

	return comparison, nil
}

// GetCompetitionData retrieves competition data
func (tm *TraderManager) GetCompetitionData() (map[string]interface{}, error) {
	// Check cache
	tm.competitionCache.mu.RLock()
	if time.Since(tm.competitionCache.timestamp) < 30*time.Second && len(tm.competitionCache.data) > 0 {
		cachedData := make(map[string]interface{})
		for k, v := range tm.competitionCache.data {
			cachedData[k] = v
		}
		tm.competitionCache.mu.RUnlock()
		logger.Infof("📋 Returning competition data cache (cache age: %.1fs)", time.Since(tm.competitionCache.timestamp).Seconds())
		return cachedData, nil
	}
	tm.competitionCache.mu.RUnlock()

	// TODO: Implement competition data fetching
	return tm.GetComparisonData()
}

// getConcurrentTraderData fetches trader data concurrently
func (tm *TraderManager) getConcurrentTraderData(schedulers []scheduler.Scheduler) []map[string]interface{} {
	// TODO: Implement concurrent data fetching
	return make([]map[string]interface{}, 0)
}

// UpdateTraderConfig updates a trader's configuration
func (tm *TraderManager) UpdateTraderConfig(traderID string, trader *store.Trader) error {
	// TODO: Stop existing scheduler, reload with new config
	return fmt.Errorf("not implemented")
}

// RemoveTrader removes a trader
func (tm *TraderManager) RemoveTrader(traderID string) error {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	sched, exists := tm.schedulers[traderID]
	if !exists {
		return fmt.Errorf("trader %s not found", traderID)
	}

	// Stop the scheduler
	if err := sched.Stop(); err != nil {
		logger.Infof("⚠️  Error stopping scheduler: %v", err)
	}

	// Remove from map
	delete(tm.schedulers, traderID)
	delete(tm.loadErrors, traderID)

	logger.Infof("✓ Removed trader: %s", traderID)
	return nil
}

// LoadUserTradersFromStore loads traders for a specific user
func (tm *TraderManager) LoadUserTradersFromStore(st *store.Store, userID string) error {
	tm.mu.Lock()
	defer tm.mu.Unlock()

	logger.Infof("📦 Loading traders for user %s from store...", userID)

	traders, err := st.Trader().List(userID)
	if err != nil {
		return fmt.Errorf("failed to list traders: %w", err)
	}

	for _, t := range traders {
		if err := tm.loadSingleTrader(st, t); err != nil {
			logger.Warnf("⚠️ Failed to load trader %s: %v", t.ID, err)
		}
	}

	logger.Infof("✓ Loaded %d traders for user %s", len(traders), userID)
	return nil
}

// GetTrader returns a trader by ID
func (tm *TraderManager) GetTrader(traderID string) (scheduler.Trader, error) {
	tm.mu.RLock()
	defer tm.mu.RUnlock()

	sched, exists := tm.schedulers[traderID]
	if !exists {
		return nil, fmt.Errorf("trader %s not found", traderID)
	}

	t := sched.GetTrader()
	if t == nil {
		return nil, fmt.Errorf("trader %s not initialized", traderID)
	}

	return t, nil
}
