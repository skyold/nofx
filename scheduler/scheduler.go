package scheduler

import (
	"context"
	"fmt"
	"nofx/engine"
	"nofx/logger"
	"nofx/store"
	"sync"
	"time"
)

// Scheduler 是 Scheduler 接口的默认实现
// 负责协调引擎和 Trader 完成交易
type SchedulerImpl struct {
	// 配置
	interval time.Duration

	// 组件
	engine Engine
	trader Trader
	store  *store.Store // 添加数据存储

	// 状态
	isRunning bool
	mu        sync.RWMutex

	// 调度控制
	stopCh chan struct{}
	ticker *time.Ticker

	// 统计信息
	stats *SchedulerStats

	// 交易相关信息
	traderID    string
	cycleNumber int
}

// NewScheduler 创建调度器
func NewScheduler(interval time.Duration) Scheduler {
	return &SchedulerImpl{
		interval: interval,
		stats: &SchedulerStats{
			StartTime:         time.Time{},
			LastRunTime:       time.Time{},
			NextRunTime:       time.Time{},
			TotalRuns:         0,
			SuccessfulRuns:    0,
			FailedRuns:        0,
			TotalDecisions:    0,
			ExecutedDecisions: 0,
		},
	}
}

// Start 启动调度器
func (s *SchedulerImpl) Start() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.isRunning {
		return fmt.Errorf("scheduler is already running")
	}

	if s.engine == nil {
		return fmt.Errorf("engine is not set")
	}

	if s.trader == nil {
		return fmt.Errorf("trader is not set")
	}

	s.isRunning = true
	s.stopCh = make(chan struct{})
	s.ticker = time.NewTicker(s.interval)
	s.stats.StartTime = time.Now()
	s.stats.NextRunTime = time.Now().Add(s.interval)

	// 启动调度循环
	go s.runLoop()

	logger.Infof("🚀 Scheduler started with interval: %v", s.interval)
	return nil
}

// Stop 停止调度器
func (s *SchedulerImpl) Stop() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.isRunning {
		return fmt.Errorf("scheduler is not running")
	}

	close(s.stopCh)
	s.ticker.Stop()
	s.isRunning = false

	logger.Infof("🛑 Scheduler stopped")
	return nil
}

// IsRunning 检查是否正在运行
func (s *SchedulerImpl) IsRunning() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.isRunning
}

// SetEngine 设置交易引擎
func (s *SchedulerImpl) SetEngine(engine Engine) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.engine = engine
	logger.Infof("🔧 Scheduler engine set to: %s", engine.Name())
}

// SetTrader 设置交易执行器
func (s *SchedulerImpl) SetTrader(trader Trader) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.trader = trader
	logger.Infof("🔧 Scheduler trader set to: %s", trader.GetExchange())

	// 同时设置 traderID
	if s.traderID == "" {
		s.traderID = "default_trader" // 应该从 trader 或其他地方获取真实的 traderID
	}
}

// SetStore 设置数据存储
func (s *SchedulerImpl) SetStore(store *store.Store) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.store = store
	logger.Infof("🔧 Scheduler store set")
}

// SetInterval 设置调度间隔
func (s *SchedulerImpl) SetInterval(interval time.Duration) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.interval = interval
	if s.ticker != nil && s.isRunning {
		s.ticker.Reset(interval)
	}
	logger.Infof("⏱️ Scheduler interval updated to: %v", interval)
}

// GetInterval 获取调度间隔
func (s *SchedulerImpl) GetInterval() time.Duration {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.interval
}

// GetStatus 获取调度器状态
func (s *SchedulerImpl) GetStatus() *SchedulerStatus {
	s.mu.RLock()
	defer s.mu.RUnlock()

	status := &SchedulerStatus{
		IsRunning: s.isRunning,
		Interval:  s.interval,
	}

	if s.engine != nil {
		status.EngineName = s.engine.Name()
	}

	if s.trader != nil {
		status.TraderExchange = s.trader.GetExchange()
	}

	return status
}

// GetStats 获取统计信息
func (s *SchedulerImpl) GetStats() *SchedulerStats {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.stats
}

// runLoop 运行调度循环
func (s *SchedulerImpl) runLoop() {
	for {
		select {
		case <-s.stopCh:
			return
		case <-s.ticker.C:
			s.runCycle()
		}
	}
}

// runCycle 运行一个完整的交易周期
func (s *SchedulerImpl) runCycle() {
	s.mu.Lock()
	if !s.isRunning {
		s.mu.Unlock()
		return
	}

	s.stats.LastRunTime = time.Now()
	s.stats.NextRunTime = time.Now().Add(s.interval)
	s.stats.TotalRuns++
	s.mu.Unlock()

	logger.Infof("🔄 Starting trading cycle...")

	// 执行交易周期
	err := s.executeTradingCycle()

	s.mu.Lock()
	if err != nil {
		s.stats.FailedRuns++
		logger.Errorf("❌ Trading cycle failed: %v", err)
	} else {
		s.stats.SuccessfulRuns++
		logger.Infof("✅ Trading cycle completed successfully")
	}
	s.mu.Unlock()
}

// executeTradingCycle 执行一个完整的交易周期
// 这是调度器的核心逻辑，协调引擎和 Trader 完成交易
func (s *SchedulerImpl) executeTradingCycle() error {
	ctx := context.Background()

	// 1. 构建运行时信息
	runtime := engine.RuntimeInfo{
		CurrentTime:    time.Now().Format(time.RFC3339),
		RuntimeMinutes: int(time.Since(s.stats.StartTime).Minutes()),
		CallCount:      int(s.stats.TotalRuns + 1),
	}

	// 2. 构建上下文（引擎负责）
	logger.Infof("📦 Building context...")
	context, err := s.engine.BuildContext(ctx, runtime)
	if err != nil {
		// 保存错误决策记录
		s.saveErrorDecision(runtime, "", "", "", fmt.Sprintf("failed to build context: %v", err))
		return fmt.Errorf("failed to build context: %w", err)
	}
	logger.Infof("✅ Context built successfully")

	// 3. 构建系统提示词（引擎负责）
	logger.Infof("📝 Building system prompt...")
	systemPrompt := s.engine.BuildSystemPrompt(context)

	// 4. 构建用户提示词（引擎负责）
	logger.Infof("📝 Building user prompt...")
	userPrompt := s.engine.BuildUserPrompt(context)

	// 5. 调用 LLM（引擎负责）
	logger.Infof("🤖 Calling LLM...")
	response, err := s.engine.CallLLM(ctx, systemPrompt, userPrompt)
	if err != nil {
		// 保存错误决策记录
		s.saveErrorDecision(runtime, systemPrompt, userPrompt, "", fmt.Sprintf("failed to call LLM: %v", err))
		return fmt.Errorf("failed to call LLM: %w", err)
	}
	logger.Infof("✅ LLM response received")

	// 保存 AI 决策记录（成功）
	s.saveDecisionRecord(runtime, systemPrompt, userPrompt, response.RawResponse, "success", "")

	// 6. 解析响应（引擎负责）
	logger.Infof("🔍 Parsing response...")
	decisions, err := s.engine.ParseResponse(response)
	if err != nil {
		// 保存错误决策记录
		s.saveErrorDecision(runtime, systemPrompt, userPrompt, response.RawResponse, fmt.Sprintf("failed to parse response: %v", err))
		return fmt.Errorf("failed to parse response: %w", err)
	}
	logger.Infof("✅ Parsed %d decisions", len(decisions))

	s.mu.Lock()
	s.stats.TotalDecisions += int64(len(decisions))
	s.mu.Unlock()

	// 7. 验证决策（引擎负责）
	logger.Infof("✅ Validating decisions...")
	validatedDecisions, err := s.engine.ValidateDecisions(ctx, decisions, context)
	if err != nil {
		// 保存错误决策记录
		s.saveErrorDecision(runtime, systemPrompt, userPrompt, response.RawResponse, fmt.Sprintf("failed to validate decisions: %v", err))
		return fmt.Errorf("failed to validate decisions: %w", err)
	}
	logger.Infof("✅ Validated %d decisions", len(validatedDecisions))

	// 8. 执行决策（Trader 负责）
	for i, decision := range validatedDecisions {
		if !decision.IsApproved {
			logger.Warnf("⚠️  Decision %d not approved, skipping", i+1)
			continue
		}

		logger.Infof("⚡ Executing decision %d: %s %s", i+1, decision.Symbol, decision.Action)
		result, err := s.trader.ExecuteDecision(ctx, decision)
		if err != nil {
			logger.Errorf("❌ Failed to execute decision %d: %v", i+1, err)
			continue
		}

		logger.Infof("✅ Decision %d executed successfully: %+v", i+1, result)

		s.mu.Lock()
		s.stats.ExecutedDecisions++
		s.mu.Unlock()
	}

	return nil
}

// saveDecisionRecord 保存决策记录到数据库
func (s *SchedulerImpl) saveDecisionRecord(runtime engine.RuntimeInfo, systemPrompt, userPrompt, rawResponse, status, errorMessage string) {
	if s.store == nil {
		logger.Warnf("⚠️  Store not set, skipping decision record save")
		return
	}

	s.cycleNumber++
	record := &store.DecisionRecord{
		TraderID:     s.traderID,
		CycleNumber:  s.cycleNumber,
		Timestamp:    time.Now().UTC(),
		SystemPrompt: systemPrompt,
		InputPrompt:  userPrompt,
		RawResponse:  rawResponse,
		Success:      status == "success",
	}

	if errorMessage != "" {
		record.ErrorMessage = errorMessage
	}

	if err := s.store.Decision().LogDecision(record); err != nil {
		logger.Errorf("❌ Failed to save decision record: %v", err)
	} else {
		logger.Infof("📝 Decision record saved: trader=%s, cycle=%d", s.traderID, s.cycleNumber)
	}
}

// saveErrorDecision 保存错误决策记录
func (s *SchedulerImpl) saveErrorDecision(runtime engine.RuntimeInfo, systemPrompt, userPrompt, rawResponse, errorMessage string) {
	s.saveDecisionRecord(runtime, systemPrompt, userPrompt, rawResponse, "error", errorMessage)
}
