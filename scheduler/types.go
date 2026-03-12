package scheduler

import (
	"time"
)

// SchedulerStatus 调度器状态
type SchedulerStatus struct {
	// 是否正在运行
	IsRunning bool `json:"is_running"`
	// 调度间隔
	Interval time.Duration `json:"interval"`
	// 引擎名称
	EngineName string `json:"engine_name"`
	// Trader 交易所
	TraderExchange string `json:"trader_exchange"`
}

// SchedulerStats 调度器统计
type SchedulerStats struct {
	// 启动时间
	StartTime time.Time `json:"start_time"`
	// 上次运行时间
	LastRunTime time.Time `json:"last_run_time"`
	// 下次运行时间
	NextRunTime time.Time `json:"next_run_time"`
	// 总运行次数
	TotalRuns int64 `json:"total_runs"`
	// 成功运行次数
	SuccessfulRuns int64 `json:"successful_runs"`
	// 失败运行次数
	FailedRuns int64 `json:"failed_runs"`
	// 总决策数
	TotalDecisions int64 `json:"total_decisions"`
	// 已执行决策数
	ExecutedDecisions int64 `json:"executed_decisions"`
}


