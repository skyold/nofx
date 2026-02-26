package chaos

import (
	"fmt"
	"nofx/kernel"
	"nofx/logger"
	"nofx/market"
	"nofx/mcp"
	"nofx/provider/nofxos"
	"nofx/store"
	"time"
)

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

// =============================================================================
// 对外 API - Prompt 构建
// =============================================================================

// BuildSystemPrompt implements PromptBuilder interface for API compatibility
func (e *ChaosEngine) BuildSystemPrompt(accountEquity float64, variant string) string {
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

// BuildUserPrompt builds User Prompt using ChaosContext
func (e *ChaosEngine) BuildUserPrompt(ctx *ChaosContext) string {
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
func (e *ChaosEngine) ExtractCoTTrace(response string) string {
	return e.manager.ExtractReasoning(response)
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
	userPrompt := e.BuildUserPrompt(ctx)

	aiCallStart := time.Now()
	aiResponse, err := mcpClient.CallWithMessages(systemPrompt, userPrompt)
	aiCallDuration := time.Since(aiCallStart)
	if err != nil {
		return nil, fmt.Errorf("AI API call failed: %w", err)
	}

	decisions, decisionJSON, err := ExtractDecisions(aiResponse)

	result := &DecisionResult{
		SystemPrompt:        systemPrompt,
		UserPrompt:          userPrompt,
		CoTTrace:            e.manager.ExtractReasoning(aiResponse),
		DecisionJSON:        decisionJSON,
		RawResponse:         aiResponse,
		Timestamp:           time.Now(),
		AIRequestDurationMs: aiCallDuration.Milliseconds(),
	}

	if err != nil {
		return result, fmt.Errorf("format audit failed (JSON parsing error): %w", err)
	}

	result.RawDecisions = decisions

	reasoning, _ := ExtractReasoningJSON(aiResponse)

	validatedDecisions, err := e.ValidateDecisions(decisions, reasoning, ctx)
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

func (e *ChaosEngine) ValidateDecisions(decisions []Decision, reasoning *Reasoning, ctx *ChaosContext) ([]Decision, error) {
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
