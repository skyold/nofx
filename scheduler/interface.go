package scheduler

import (
	"time"

	"nofx/engine"
	"nofx/store"
	"nofx/trader/types"
)

// Scheduler 调度器接口
// 负责协调引擎和 Trader 完成交易
type Scheduler interface {
	// === 生命周期管理 ===

	// Start 启动调度器
	Start() error

	// Stop 停止调度器
	Stop() error

	// IsRunning 检查是否正在运行
	IsRunning() bool

	// === 组件管理 ===

	// SetEngine 设置交易引擎
	SetEngine(engine Engine)

	// SetTrader 设置交易执行器
	SetTrader(trader Trader)

	// SetStore 设置数据存储
	SetStore(store *store.Store)

	// === 配置管理 ===

	// SetInterval 设置调度间隔
	SetInterval(interval time.Duration)

	// GetInterval 获取调度间隔
	GetInterval() time.Duration

	// === 状态查询 ===

	// GetStatus 获取调度器状态
	GetStatus() *SchedulerStatus

	// GetStats 获取统计信息
	GetStats() *SchedulerStats
}

// Engine 引擎接口（使用 engine 包的定义）
type Engine = engine.Engine

// Trader 交易器接口（使用 trader/types 包的定义）
type Trader = types.Trader
