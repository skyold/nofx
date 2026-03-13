package chaos

import (
	"context"
	"fmt"
	"nofx/engine"
	"nofx/kernel"
	"nofx/logger"
	"nofx/market"
	"nofx/mcp"
	"nofx/provider/nofxos"
	"nofx/store"
	"nofx/trader/types"
	"time"
)

// 确保 ChaosEngine 实现 engine.Engine 接口
var _ engine.Engine = (*ChaosEngine)(nil)

// =============================================================================
// Engine 接口实现
// =============================================================================

// Name 返回引擎名称
func (e *ChaosEngine) Name() string {
	return "chaos"
}

// BuildContext 构建完整的交易上下文
// 这是 Engine 接口的核心方法
// 引擎负责构建完整的上下文，包括账户、持仓、市场数据等
func (e *ChaosEngine) BuildContext(ctx context.Context, runtime engine.RuntimeInfo) (*engine.Context, error) {
	// 检查是否设置了依赖
	if e.trader == nil {
		// 返回简化版本（向后兼容）
		return &engine.Context{
			CurrentTime:    runtime.CurrentTime,
			RuntimeMinutes: runtime.RuntimeMinutes,
			CallCount:      runtime.CallCount,
			Config:         e.config,
		}, nil
	}

	// 使用 ContextBuilder 构建完整上下文
	chaosConfig := &ChaosConfig{
		ChaosPrompt:         e.config.ChaosConfig.ChaosPrompt,
		RiskControl:         e.config.ChaosConfig.RiskControl,
		SystemPromptVariant: e.config.ChaosConfig.SystemPromptVariant,
		UserPromptVersion:   e.config.ChaosConfig.UserPromptVersion,
		Indicators:          e.config.ChaosConfig.Indicators,
	}

	builder := NewContextBuilder(
		e.trader,
		e.strategyEngine,
		e.nofxosClient,
		chaosConfig,
		e.store,
		e.traderID,
		e.startTime,
		e.callCount,
	)

	builderRuntime := RuntimeInfo{
		CurrentTime:    time.Now(),
		RuntimeMinutes: runtime.RuntimeMinutes,
		CallCount:      runtime.CallCount,
	}

	chaosCtx, err := builder.BuildContext(ctx, builderRuntime)
	if err != nil {
		return nil, fmt.Errorf("failed to build chaos context: %w", err)
	}

	// 将 ChaosContext 转换为 engine.Context
	return &engine.Context{
		CurrentTime:    chaosCtx.CurrentTime,
		RuntimeMinutes: chaosCtx.RuntimeMinutes,
		CallCount:      chaosCtx.CallCount,
		Account:        chaosCtx.Account,
		Positions:      chaosCtx.Positions,
		CandidateCoins: chaosCtx.CandidateCoins,
		MarketDataMap:  chaosCtx.MarketDataMap,
		QuantDataMap:   chaosCtx.QuantDataMap,
		Config:         e.config,
	}, nil
}

// BuildSystemPrompt 构建系统提示词（实现 engine.Engine 接口）
func (e *ChaosEngine) BuildSystemPrompt(ctx *engine.Context) string {
	// 将 engine.Context 转换为 ChaosContext
	chaosCtx := &ChaosContext{
		CurrentTime:    ctx.CurrentTime,
		RuntimeMinutes: ctx.RuntimeMinutes,
		CallCount:      ctx.CallCount,
		Config: &ChaosConfig{
			ChaosPrompt: e.config.ChaosConfig.ChaosPrompt,
			RiskControl: e.config.ChaosConfig.RiskControl,
			Indicators:  e.config.ChaosConfig.Indicators,
		},
	}
	return e.BuildSystemPromptWithContext(chaosCtx)
}

// BuildUserPrompt 构建用户提示词（实现 engine.Engine 接口）
func (e *ChaosEngine) BuildUserPrompt(ctx *engine.Context) string {
	// 将 engine.Context 转换为 ChaosContext
	chaosCtx := &ChaosContext{
		CurrentTime:    ctx.CurrentTime,
		RuntimeMinutes: ctx.RuntimeMinutes,
		CallCount:      ctx.CallCount,
		// TODO: 需要从 ctx.Account, ctx.Positions 等填充数据
	}
	return e.BuildUserPromptLegacy(chaosCtx)
}

// CallLLM 调用 LLM 服务（实现 engine.Engine 接口）
// 注意：这个方法需要 mcp.AIClient 参数，应该由调度器传入
func (e *ChaosEngine) CallLLM(ctx context.Context, systemPrompt, userPrompt string) (*engine.AIResponse, error) {
	// TODO: 需要 mcp.AIClient 参数
	// 这个方法应该在调度器中调用，而不是在引擎内部
	// 调度器应该这样使用：
	// aiResponse, err := mcpClient.CallWithMessages(systemPrompt, userPrompt)
	// return &engine.AIResponse{
	//     RawResponse: aiResponse,
	// }, nil
	return nil, fmt.Errorf("CallLLM requires MCP client from scheduler")
}

// ParseResponse 解析 LLM 响应（实现 engine.Engine 接口）
func (e *ChaosEngine) ParseResponse(response *engine.AIResponse) ([]engine.Decision, error) {
	if response == nil || response.RawResponse == "" {
		return nil, fmt.Errorf("empty response")
	}

	// 使用现有的解析逻辑
	decisions, _, err := e.ExtractDecisions(response.RawResponse)
	if err != nil {
		return nil, err
	}

	// 转换为 engine.Decision 类型
	engineDecisions := make([]engine.Decision, len(decisions))
	for i, d := range decisions {
		engineDecisions[i] = engine.Decision{
			Symbol:          d.Symbol,
			Action:          d.Action,
			Leverage:        d.Leverage,
			EntryPrice:      d.EntryPrice,
			StopLoss:        d.StopLoss,
			TakeProfit:      d.TakeProfit,
			PositionSizeUSD: d.PositionSizeUSD,
			Reasoning:       "", // TODO: 从 response.Reasoning 获取
		}
	}

	return engineDecisions, nil
}

// ValidateDecisions 验证决策（实现 engine.Engine 接口）
func (e *ChaosEngine) ValidateDecisions(ctx context.Context, decisions []engine.Decision, context *engine.Context) ([]engine.ValidatedDecision, error) {
	if len(decisions) == 0 {
		return []engine.ValidatedDecision{}, nil
	}

	// 转换为 chaos.Decision 类型
	chaosDecisions := make([]Decision, len(decisions))
	for i, d := range decisions {
		chaosDecisions[i] = Decision{
			Symbol:          d.Symbol,
			Action:          d.Action,
			Leverage:        d.Leverage,
			EntryPrice:      d.EntryPrice,
			StopLoss:        d.StopLoss,
			TakeProfit:      d.TakeProfit,
			PositionSizeUSD: d.PositionSizeUSD,
		}
	}

	// 将 engine.Context 转换为 ChaosContext
	chaosCtx := &ChaosContext{
		CurrentTime:    context.CurrentTime,
		RuntimeMinutes: context.RuntimeMinutes,
		CallCount:      context.CallCount,
		Config: &ChaosConfig{
			ChaosPrompt: e.config.ChaosConfig.ChaosPrompt,
			RiskControl: e.config.ChaosConfig.RiskControl,
			Indicators:  e.config.ChaosConfig.Indicators,
		},
	}

	// TODO: 需要从 context.Account 获取账户信息
	// 目前使用默认值
	accountEquity := 0.0

	// 使用现有的验证逻辑
	var validated []engine.ValidatedDecision
	riskConfig := chaosCtx.Config.RiskControl

	for _, d := range chaosDecisions {
		// 使用 manager 验证（需要 Reasoning，这里传 nil）
		positionSizeUSD, err := e.manager.ValidateDecision(&d, nil, accountEquity, riskConfig)
		if err != nil {
			return nil, fmt.Errorf("validation failed for %s: %w", d.Symbol, err)
		}

		validated = append(validated, engine.ValidatedDecision{
			Decision: engine.Decision{
				Symbol:          d.Symbol,
				Action:          d.Action,
				Leverage:        d.Leverage,
				EntryPrice:      d.EntryPrice,
				StopLoss:        d.StopLoss,
				TakeProfit:      d.TakeProfit,
				PositionSizeUSD: &positionSizeUSD,
				Reasoning:       "", // chaos.Decision 没有 Reasoning 字段
			},
			ValidatedPositionUSD: positionSizeUSD,
			ValidationErrors:     nil,
			IsApproved:           true,
		})
	}

	return validated, nil
}

// =============================================================================
// ChaosEngine - 对外统一入口
// =============================================================================
// 所有外部调用都通过 ChaosEngine 进行，Manager 等模块作为内部工作模块。
// =============================================================================

// ChaosEngine handles Chaos mode execution
type ChaosEngine struct {
	manager      *Manager
	config       *store.StrategyConfig
	nofxosClient *nofxos.Client

	// 依赖注入（用于 BuildContext）
	trader         types.Trader
	strategyEngine *kernel.StrategyEngine
	store          *store.Store
	traderID       string
	startTime      time.Time
	callCount      int
}

// NewChaosEngine creates a new ChaosEngine
func NewChaosEngine(config *store.StrategyConfig) *ChaosEngine {
	if config == nil {
		defaultConfig := store.GetDefaultStrategyConfig("en")
		config = &defaultConfig
	}

	indicators := config.Indicators
	if config.ChaosConfig != nil {
		indicators = config.ChaosConfig.Indicators
	}

	apiKey := indicators.NofxOSAPIKey
	if apiKey == "" {
		apiKey = nofxos.DefaultAuthKey
	}
	client := nofxos.NewClient(nofxos.DefaultBaseURL, apiKey)

	return &ChaosEngine{
		manager:      NewManager(),
		config:       config,
		nofxosClient: client,
	}
}

// SetDependencies 设置引擎的依赖（用于 BuildContext）
// 这是可选的，如果不设置，BuildContext 将返回简化版本
func (e *ChaosEngine) SetDependencies(
	trader types.Trader,
	strategyEngine *kernel.StrategyEngine,
	store *store.Store,
	traderID string,
	startTime time.Time,
	callCount int,
) {
	e.trader = trader
	e.strategyEngine = strategyEngine
	e.store = store
	e.traderID = traderID
	e.startTime = startTime
	e.callCount = callCount
}

// =============================================================================
// 对外 API - Prompt 构建（已迁移到 Engine 接口）
// =============================================================================

// BuildSystemPromptLegacy 已废弃，使用 Engine 接口的 BuildSystemPrompt
// 保留用于向后兼容
// 注意：这个方法只是为了保持向后兼容，实际应该使用 BuildSystemPromptWithContext
func (e *ChaosEngine) BuildSystemPromptLegacy(accountEquity float64, variant string) string {
	var chaosPrompt string
	var riskControl store.RiskControlConfig
	var indicators store.IndicatorConfig

	if e.config != nil && e.config.ChaosConfig != nil {
		chaosPrompt = e.config.ChaosConfig.ChaosPrompt
		riskControl = e.config.ChaosConfig.RiskControl
		indicators = e.config.ChaosConfig.Indicators
	}

	ctx := &ChaosContext{
		Config: &ChaosConfig{
			ChaosPrompt:         chaosPrompt,
			RiskControl:         riskControl,
			SystemPromptVariant: variant,
			Indicators:          indicators,
		},
	}
	return e.BuildSystemPromptWithContext(ctx)
}

// BuildSystemPromptWithContext builds the system prompt for Chaos mode using context
func (e *ChaosEngine) BuildSystemPromptWithContext(ctx *ChaosContext) string {
	if ctx.Config != nil && ctx.Config.ChaosPrompt != "" {
		return e.manager.BuildSystemPrompt(
			ctx.Config.SystemPromptVariant,
			ctx.Config.ChaosPrompt,
			ctx.Config.Indicators,
		)
	}

	return "# Chaos Mode (Missing Configuration)\n\nPlease provide trading decisions based on market data."
}

// BuildUserPromptLegacy builds User Prompt using ChaosContext (legacy, deprecated)
// Kept for backward compatibility
func (e *ChaosEngine) BuildUserPromptLegacy(ctx *ChaosContext) string {
	return e.manager.BuildUserPrompt(ctx)
}

// =============================================================================
// 对外 API - 模式检测与参数
// =============================================================================

// IsChaosMode checks if the current prompt indicates Chaos mode
func (e *ChaosEngine) IsChaosMode(customPrompt string) bool {
	return e.manager.IsChaosMode(customPrompt)
}

// GetVariantParams returns the parameters for a specific variant
func (e *ChaosEngine) GetVariantParams(variant string) map[string]string {
	return e.manager.GetVariantParams(variant)
}

// =============================================================================
// 对外 API - 决策验证
// =============================================================================

// ValidateDecision validates a single decision and returns position size
func (e *ChaosEngine) ValidateDecision(d *Decision, reasoning *Reasoning, accountEquity float64, riskConfig store.RiskControlConfig) (float64, error) {
	return e.manager.ValidateDecision(d, reasoning, accountEquity, riskConfig)
}

// =============================================================================
// 对外 API - 响应解析
// =============================================================================

// ExtractDecisions extracts decisions from AI response
func (e *ChaosEngine) ExtractDecisions(aiResponse string) ([]Decision, string, error) {
	return ExtractDecisions(aiResponse)
}

// ExtractReasoningJSON extracts reasoning from AI response
func (e *ChaosEngine) ExtractReasoningJSON(aiResponse string) (*Reasoning, error) {
	return ExtractReasoningJSON(aiResponse)
}

// ExtractCoTTrace extracts the Chain of Thought from the AI response
// 无论是否有 JSON 格式，都只处理最外层的<execution>标签
func (e *ChaosEngine) ExtractCoTTrace(response string) string {
	return ExtractReasoning(response)
}

// =============================================================================
// 对外 API - 候选币种
// =============================================================================

// GetCandidateCoins gets candidate coins based on chaos strategy configuration
func (e *ChaosEngine) GetCandidateCoins() ([]kernel.CandidateCoin, error) {
	if e.config.ChaosConfig == nil {
		return e.getCandidateCoinsFromSource(e.config.CoinSource)
	}
	return e.getCandidateCoinsFromSource(e.config.ChaosConfig.CoinSource)
}

// =============================================================================
// 对外 API - 完整执行流程
// =============================================================================

// Execute runs the Chaos decision process
func (e *ChaosEngine) Execute(ctx *ChaosContext, mcpClient mcp.AIClient) (*DecisionResult, error) {
	systemPrompt := e.BuildSystemPromptWithContext(ctx)
	userPrompt := e.BuildUserPromptLegacy(ctx)

	aiCallStart := time.Now()
	aiResponse, err := mcpClient.CallWithMessages(systemPrompt, userPrompt)
	aiCallDuration := time.Since(aiCallStart)
	if err != nil {
		return nil, fmt.Errorf("AI API call failed: %w", err)
	}

	reasoning, err := e.ExtractReasoningJSON(aiResponse)
	if err != nil {
		return nil, fmt.Errorf("failed to extract reasoning: %w", err)
	}

	decisions, decisionJSON, err := e.ExtractDecisions(aiResponse)
	if err != nil {
		return nil, fmt.Errorf("failed to extract decisions: %w", err)
	}

	result := &DecisionResult{
		SystemPrompt:        systemPrompt,
		UserPrompt:          userPrompt,
		CoTTrace:            ExtractReasoning(aiResponse),
		DecisionJSON:        decisionJSON,
		RawResponse:         aiResponse,
		Timestamp:           time.Now(),
		AIRequestDurationMs: aiCallDuration.Milliseconds(),
	}

	if err != nil {
		return result, fmt.Errorf("format audit failed (JSON parsing error): %w", err)
	}

	result.RawDecisions = decisions

	validatedDecisions, err := e.ValidateDecisionsInternal(decisions, reasoning, ctx)
	if err != nil {
		return result, fmt.Errorf("content audit failed: %w", err)
	}

	result.Decisions = validatedDecisions

	return result, nil
}

// GetChaosDecisions gets the decisions for Chaos mode (legacy function)
func GetChaosDecisions(ctx *ChaosContext, mcpClient mcp.AIClient, engine *ChaosEngine) (*DecisionResult, error) {
	if engine == nil {
		engine = NewChaosEngine(nil)
		logger.Warnf("⚠️  ChaosEngine instantiated with nil config. This is expected if not using custom indicators.")
	} else {
		if engine.config.ChaosConfig != nil {
			logger.Infof("ChaosEngine config: %+v", *engine.config.ChaosConfig)
		} else {
			logger.Infof("ChaosEngine config: <nil>")
		}
	}

	return engine.Execute(ctx, mcpClient)
}

// =============================================================================
// 内部方法 - 候选币种获取
// =============================================================================

func (e *ChaosEngine) getCandidateCoinsFromSource(coinSource store.CoinSourceConfig) ([]kernel.CandidateCoin, error) {
	var candidates []kernel.CandidateCoin
	symbolSources := make(map[string][]string)

	switch coinSource.SourceType {
	case "static":
		for _, symbol := range coinSource.StaticCoins {
			symbol = market.Normalize(symbol)
			candidates = append(candidates, kernel.CandidateCoin{
				Symbol:  symbol,
				Sources: []string{"static"},
			})
		}

		return e.filterExcludedCoinsFromList(candidates, coinSource.ExcludedCoins), nil

	case "ai500":
		if !coinSource.UseAI500 {
			logger.Infof("⚠️  source_type is 'ai500' but use_ai500 is false, falling back to static coins")
			for _, symbol := range coinSource.StaticCoins {
				symbol = market.Normalize(symbol)
				candidates = append(candidates, kernel.CandidateCoin{
					Symbol:  symbol,
					Sources: []string{"static"},
				})
			}
			return e.filterExcludedCoinsFromList(candidates, coinSource.ExcludedCoins), nil
		}
		coins, err := e.getAI500Coins(coinSource.AI500Limit)
		if err != nil {
			return nil, err
		}
		return e.filterExcludedCoinsFromList(coins, coinSource.ExcludedCoins), nil

	case "oi_top":
		if !coinSource.UseOITop {
			logger.Infof("⚠️  source_type is 'oi_top' but use_oi_top is false, falling back to static coins")
			for _, symbol := range coinSource.StaticCoins {
				symbol = market.Normalize(symbol)
				candidates = append(candidates, kernel.CandidateCoin{
					Symbol:  symbol,
					Sources: []string{"static"},
				})
			}
			return e.filterExcludedCoinsFromList(candidates, coinSource.ExcludedCoins), nil
		}
		coins, err := e.getOITopCoins(coinSource.OITopLimit)
		if err != nil {
			return nil, err
		}
		return e.filterExcludedCoinsFromList(coins, coinSource.ExcludedCoins), nil

	case "oi_low":
		if !coinSource.UseOILow {
			logger.Infof("⚠️  source_type is 'oi_low' but use_oi_low is false, falling back to static coins")
			for _, symbol := range coinSource.StaticCoins {
				symbol = market.Normalize(symbol)
				candidates = append(candidates, kernel.CandidateCoin{
					Symbol:  symbol,
					Sources: []string{"static"},
				})
			}
			return e.filterExcludedCoinsFromList(candidates, coinSource.ExcludedCoins), nil
		}
		coins, err := e.getOILowCoins(coinSource.OILowLimit)
		if err != nil {
			return nil, err
		}
		return e.filterExcludedCoinsFromList(coins, coinSource.ExcludedCoins), nil

	case "mixed":
		if coinSource.UseAI500 {
			poolCoins, err := e.getAI500Coins(coinSource.AI500Limit)
			if err != nil {
				logger.Infof("⚠️  Failed to get AI500 coins: %v", err)
			} else {
				for _, coin := range poolCoins {
					symbolSources[coin.Symbol] = append(symbolSources[coin.Symbol], "ai500")
				}
			}
		}

		if coinSource.UseOITop {
			oiCoins, err := e.getOITopCoins(coinSource.OITopLimit)
			if err != nil {
				logger.Infof("⚠️  Failed to get OI Top: %v", err)
			} else {
				for _, coin := range oiCoins {
					symbolSources[coin.Symbol] = append(symbolSources[coin.Symbol], "oi_top")
				}
			}
		}

		if coinSource.UseOILow {
			oiLowCoins, err := e.getOILowCoins(coinSource.OILowLimit)
			if err != nil {
				logger.Infof("⚠️  Failed to get OI Low: %v", err)
			} else {
				for _, coin := range oiLowCoins {
					symbolSources[coin.Symbol] = append(symbolSources[coin.Symbol], "oi_low")
				}
			}
		}

		for _, symbol := range coinSource.StaticCoins {
			symbol = market.Normalize(symbol)
			if _, exists := symbolSources[symbol]; !exists {
				symbolSources[symbol] = []string{"static"}
			} else {
				symbolSources[symbol] = append(symbolSources[symbol], "static")
			}
		}

		for symbol, sources := range symbolSources {
			candidates = append(candidates, kernel.CandidateCoin{
				Symbol:  symbol,
				Sources: sources,
			})
		}
		return e.filterExcludedCoinsFromList(candidates, coinSource.ExcludedCoins), nil

	default:
		return nil, fmt.Errorf("unknown coin source type: %s", coinSource.SourceType)
	}
}

func (e *ChaosEngine) filterExcludedCoinsFromList(candidates []kernel.CandidateCoin, excludedCoins []string) []kernel.CandidateCoin {
	if len(excludedCoins) == 0 {
		return candidates
	}

	excluded := make(map[string]bool)
	for _, coin := range excludedCoins {
		normalized := market.Normalize(coin)
		excluded[normalized] = true
	}

	filtered := make([]kernel.CandidateCoin, 0, len(candidates))
	for _, c := range candidates {
		if !excluded[c.Symbol] {
			filtered = append(filtered, c)
		} else {
			logger.Infof("🚫 Excluded coin: %s", c.Symbol)
		}
	}

	return filtered
}

func (e *ChaosEngine) getAI500Coins(limit int) ([]kernel.CandidateCoin, error) {
	if limit <= 0 {
		limit = 30
	}

	symbols, err := e.nofxosClient.GetTopRatedCoins(limit)
	if err != nil {
		return nil, err
	}

	var candidates []kernel.CandidateCoin
	for _, symbol := range symbols {
		candidates = append(candidates, kernel.CandidateCoin{
			Symbol:  symbol,
			Sources: []string{"ai500"},
		})
	}
	return candidates, nil
}

func (e *ChaosEngine) getOITopCoins(limit int) ([]kernel.CandidateCoin, error) {
	if limit <= 0 {
		limit = 10
	}

	positions, err := e.nofxosClient.GetOITopPositions()
	if err != nil {
		return nil, err
	}

	var candidates []kernel.CandidateCoin
	for i, pos := range positions {
		if i >= limit {
			break
		}
		symbol := market.Normalize(pos.Symbol)
		candidates = append(candidates, kernel.CandidateCoin{
			Symbol:  symbol,
			Sources: []string{"oi_top"},
		})
	}
	return candidates, nil
}

func (e *ChaosEngine) getOILowCoins(limit int) ([]kernel.CandidateCoin, error) {
	if limit <= 0 {
		limit = 10
	}

	positions, err := e.nofxosClient.GetOILowPositions()
	if err != nil {
		return nil, err
	}

	var candidates []kernel.CandidateCoin
	for i, pos := range positions {
		if i >= limit {
			break
		}
		symbol := market.Normalize(pos.Symbol)
		candidates = append(candidates, kernel.CandidateCoin{
			Symbol:  symbol,
			Sources: []string{"oi_low"},
		})
	}
	return candidates, nil
}

// =============================================================================
// 内部方法 - 决策验证
// =============================================================================

func (e *ChaosEngine) ValidateDecisionsInternal(decisions []Decision, reasoning *Reasoning, ctx *ChaosContext) ([]Decision, error) {
	var validated []Decision
	riskConfig := ctx.Config.RiskControl

	for _, d := range decisions {
		positionSizeUSD, err := e.manager.ValidateDecision(&d, reasoning, ctx.Account.TotalEquity, riskConfig)

		if err != nil {
			return nil, fmt.Errorf("validation failed for %s: %w", d.Symbol, err)
		}

		d.PositionSizeUSD = &positionSizeUSD
		validated = append(validated, d)
	}
	return validated, nil
}
