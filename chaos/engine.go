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

	// Create NofxOS client with API key from config
	// Prefer ChaosConfig indicators if available
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

// GetDecisions gets the decisions for Chaos mode
func GetChaosDecisions(ctx *ChaosContext, mcpClient mcp.AIClient, engine *ChaosEngine) (*DecisionResult, error) {
	// Create engine with context config
	// The config is already in the context, but NewChaosEngine expects *store.StrategyConfig
	// We need to refactor NewChaosEngine or create a temporary config adapter
	// For now, let's update Execute to use ChaosContext directly

	// Since ChaosEngine.config is store.StrategyConfig, but we are moving away from it in Phase 2.
	// We will instantiate ChaosEngine with nil or default, but Execute will use ctx.Config.

	// We need a way to pass the full strategy config if we still need it for non-chaos parts (like indicators)
	// But ChaosContext should have everything.

	// Let's create a minimal engine instance
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

// GetCandidateCoins gets candidate coins based on chaos strategy configuration
func (e *ChaosEngine) GetCandidateCoins() ([]kernel.CandidateCoin, error) {
	if e.config.ChaosConfig == nil {
		// Fallback to standard coins if chaos config is missing
		return e.getCandidateCoinsFromSource(e.config.CoinSource)
	}
	return e.getCandidateCoinsFromSource(e.config.ChaosConfig.CoinSource)
}

// getCandidateCoinsFromSource gets candidate coins based on a specific coin source configuration
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
		// 检查 use_ai500 标志，如果为 false 则回退到静态币种
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
		// 空列表是正常情况，直接返回
		return e.filterExcludedCoinsFromList(coins, coinSource.ExcludedCoins), nil

	case "oi_top":
		// 检查 use_oi_top 标志，如果为 false 则回退到静态币种
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
		// 空列表是正常情况，直接返回
		return e.filterExcludedCoinsFromList(coins, coinSource.ExcludedCoins), nil

	case "oi_low":
		// 持仓减少榜，适合做空
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
		// 空列表是正常情况，直接返回
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

// filterExcludedCoinsFromList removes excluded coins from the candidates list using a provided exclusion list
func (e *ChaosEngine) filterExcludedCoinsFromList(candidates []kernel.CandidateCoin, excludedCoins []string) []kernel.CandidateCoin {
	if len(excludedCoins) == 0 {
		return candidates
	}

	// Build excluded set for O(1) lookup
	excluded := make(map[string]bool)
	for _, coin := range excludedCoins {
		normalized := market.Normalize(coin)
		excluded[normalized] = true
	}

	// Filter out excluded coins
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

// ExtractCoTTrace extracts the Chain of Thought from the AI response
func (e *ChaosEngine) ExtractCoTTrace(response string) string {
	return e.manager.ExtractReasoning(response)
}

// Execute runs the Chaos decision process
func (e *ChaosEngine) Execute(ctx *ChaosContext, mcpClient mcp.AIClient) (*DecisionResult, error) {
	// 1. Build Prompts
	systemPrompt := e.BuildSystemPromptWithContext(ctx)
	userPrompt := e.BuildUserPrompt(ctx)

	// 2. Call AI
	aiCallStart := time.Now()
	aiResponse, err := mcpClient.CallWithMessages(systemPrompt, userPrompt)
	aiCallDuration := time.Since(aiCallStart)
	if err != nil {
		return nil, fmt.Errorf("AI API call failed: %w", err)
	}

	// 3. Parse & Validate
	decisions, decisionJSON, err := ExtractDecisions(aiResponse)

	// Create result structure
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
		// If extraction failed, return result with empty decisions but error
		return result, fmt.Errorf("format audit failed (JSON parsing error): %w", err)
	}

	// Set raw decisions for audit
	// We need to convert []Decision to interface{}
	// Since extractDecisions returns []chaos.Decision, we can just use it if we want,
	// but DecisionResult.RawDecisions is interface{}.
	// To match the structure, we can just assign it if the type matches what we want to store.
	// Let's just store the slice.
	result.RawDecisions = decisions

	// Extract Reasoning for validation
	reasoning, _ := ExtractReasoningJSON(aiResponse)

	// Validate decisions
	validatedDecisions, err := e.ValidateDecisions(decisions, reasoning, ctx)
	if err != nil {
		return result, fmt.Errorf("content audit failed: %w", err)
	}

	result.Decisions = validatedDecisions

	return result, nil
}

func (e *ChaosEngine) ValidateDecisions(decisions []Decision, reasoning *Reasoning, ctx *ChaosContext) ([]Decision, error) {
	// Implement validation using ctx.Config.RiskControl
	// This logic mirrors the kernel validation but uses Chaos structures

	var validated []Decision
	riskConfig := ctx.Config.RiskControl

	for _, d := range decisions {
		// Use Manager for validation which contains the core logic
		// We pass ChaosConfig.RiskControl params

		// Note: Manager.ValidateDecision returns calculated position size USD
		positionSizeUSD, err := e.manager.ValidateDecision(&d, reasoning, ctx.Account.TotalEquity, riskConfig)

		if err != nil {
			return nil, fmt.Errorf("validation failed for %s: %w", d.Symbol, err)
		}

		// Store validated position size
		d.PositionSizeUSD = &positionSizeUSD
		validated = append(validated, d)
	}
	return validated, nil
}
