package engine

import (
	"context"
)

// Engine 定义交易引擎的标准接口
// 引擎负责构建上下文、与 LLM 对话、解析和验证决策
// 注意：引擎不负责执行决策！
type Engine interface {
	// Name 返回引擎名称（如 "chaos", "kernel"）
	Name() string

	// BuildContext 构建完整的交易上下文
	// 包括：账户信息、持仓、市场数据、候选币种等
	// 这是引擎的核心职责之一
	BuildContext(ctx context.Context, runtime RuntimeInfo) (*Context, error)

	// BuildSystemPrompt 构建系统提示词
	// 包含：输出格式、规则约束、角色定义等
	BuildSystemPrompt(ctx *Context) string

	// BuildUserPrompt 构建用户提示词
	// 包含：账户状态、持仓详情、市场数据等
	BuildUserPrompt(ctx *Context) string

	// CallLLM 调用 LLM 服务
	// 封装 MCP Client 的调用逻辑
	CallLLM(ctx context.Context, systemPrompt, userPrompt string) (*AIResponse, error)

	// ParseResponse 解析 LLM 响应
	// 从 AI 响应中提取交易决策
	ParseResponse(response *AIResponse) ([]Decision, error)

	// ValidateDecisions 验证交易决策
	// 根据风控规则验证决策的合法性
	ValidateDecisions(ctx context.Context, decisions []Decision, context *Context) ([]ValidatedDecision, error)
}

// RuntimeInfo 运行时信息
type RuntimeInfo struct {
	CurrentTime    string `json:"current_time"`
	RuntimeMinutes int    `json:"runtime_minutes"` // 运行时长（分钟）
	CallCount      int    `json:"call_count"`      // 调用次数
}

// Context 交易上下文（通用结构）
// 包含引擎需要的所有数据
type Context struct {
	// 基础信息
	CurrentTime    string `json:"current_time"`
	RuntimeMinutes int    `json:"runtime_minutes"`
	CallCount      int    `json:"call_count"`

	// 账户信息
	Account interface{} `json:"account"`

	// 持仓信息
	Positions interface{} `json:"positions"`

	// 候选币种
	CandidateCoins interface{} `json:"candidate_coins"`

	// 市场数据
	MarketDataMap interface{} `json:"market_data_map"`

	// 量化数据
	QuantDataMap interface{} `json:"quant_data_map"`

	// 排名数据
	OIRankingData      interface{} `json:"oi_ranking_data"`
	NetFlowRankingData interface{} `json:"netflow_ranking_data"`
	PriceRankingData   interface{} `json:"price_ranking_data"`

	// 交易历史
	TradingStats interface{} `json:"trading_stats"`
	RecentOrders interface{} `json:"recent_orders"`

	// 配置（引擎特定）
	Config interface{} `json:"config"`
}

// Decision 交易决策（通用结构）
type Decision struct {
	Symbol          string   `json:"symbol"`
	Action          string   `json:"action"` // open_long, open_short, close_long, close_short, hold, wait
	Leverage        *int     `json:"leverage,omitempty"`
	EntryPrice      *float64 `json:"entry,omitempty"`
	StopLoss        *float64 `json:"stop_loss,omitempty"`
	TakeProfit      *float64 `json:"take_profit,omitempty"`
	PositionSizeUSD *float64 `json:"position_size_usd,omitempty"`
	Confidence      *int     `json:"confidence,omitempty"`
	Reasoning       string   `json:"reasoning"`
}

// ValidatedDecision 验证后的决策
type ValidatedDecision struct {
	Decision
	ValidatedPositionUSD float64  `json:"validated_position_usd"` // 经过风控验证的仓位
	ValidationErrors     []string `json:"validation_errors"`      // 验证错误（如果有）
	IsApproved           bool     `json:"is_approved"`            // 是否批准执行
}

// AIResponse LLM 响应
type AIResponse struct {
	RawResponse string                 `json:"raw_response"`
	Reasoning   string                 `json:"reasoning"`
	Decisions   interface{}            `json:"decisions"`
	Metadata    map[string]string      `json:"metadata"`
}
