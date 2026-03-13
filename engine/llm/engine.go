package ai

import (
	"context"
	"fmt"
	"nofx/engine"
	"nofx/market"
	"nofx/mcp"
	"nofx/provider/nofxos"
	"nofx/store"
)

var _ engine.Engine = (*LLMEngine)(nil)

type LLMEngine struct {
	name      string
	config    *store.StrategyConfig
	manager   *Manager
	parser    *Parser
	prompt    *PromptBuilder
	mcpClient mcp.AIClient
	client    *nofxos.Client
}

func NewLLMEngine(config *store.StrategyConfig) *LLMEngine {
	engineName := "ai"
	if config != nil && config.StrategyType != "" {
		engineName = config.StrategyType
	}

	apiKey := ""
	if config != nil && config.Indicators.NofxOSAPIKey != "" {
		apiKey = config.Indicators.NofxOSAPIKey
	} else {
		apiKey = nofxos.DefaultAuthKey
	}
	client := nofxos.NewClient(nofxos.DefaultBaseURL, apiKey)

	return &LLMEngine{
		name:    engineName,
		config:  config,
		manager: NewManager(config),
		parser:  NewParser(),
		prompt:  NewPromptBuilder(config),
		client:  client,
	}
}

func (e *LLMEngine) Name() string {
	return e.name
}

func (e *LLMEngine) BuildContext(ctx context.Context, runtime engine.RuntimeInfo) (*engine.Context, error) {
	aiCtx := &Context{
		CurrentTime:    runtime.CurrentTime,
		RuntimeMinutes: runtime.RuntimeMinutes,
		CallCount:      runtime.CallCount,
		Config:         e.config,
	}

	if e.config == nil {
		return &engine.Context{
			CurrentTime:    runtime.CurrentTime,
			RuntimeMinutes: runtime.RuntimeMinutes,
			CallCount:      runtime.CallCount,
			Config:         e.config,
		}, nil
	}

	aiCtx.Account = engine.AccountInfo{}
	aiCtx.Positions = []engine.PositionInfo{}
	aiCtx.CandidateCoins = []engine.CandidateCoin{}
	aiCtx.MarketDataMap = map[string]interface{}{}
	aiCtx.QuantDataMap = map[string]interface{}{}

	return &engine.Context{
		CurrentTime:    aiCtx.CurrentTime,
		RuntimeMinutes: aiCtx.RuntimeMinutes,
		CallCount:      aiCtx.CallCount,
		Account:        aiCtx.Account,
		Positions:      aiCtx.Positions,
		CandidateCoins: aiCtx.CandidateCoins,
		MarketDataMap:  aiCtx.MarketDataMap,
		QuantDataMap:   aiCtx.QuantDataMap,
		Config:         e.config,
	}, nil
}

func (e *LLMEngine) BuildSystemPrompt(ctx *engine.Context) string {
	if ctx == nil {
		return e.prompt.BuildSystemPrompt(nil)
	}

	aiCtx := &Context{
		CurrentTime:    ctx.CurrentTime,
		RuntimeMinutes: ctx.RuntimeMinutes,
		CallCount:      ctx.CallCount,
		Config:         e.config,
	}
	return e.prompt.BuildSystemPrompt(aiCtx)
}

func (e *LLMEngine) BuildUserPrompt(ctx *engine.Context) string {
	if ctx == nil {
		return ""
	}

	aiCtx := &Context{
		CurrentTime:    ctx.CurrentTime,
		RuntimeMinutes: ctx.RuntimeMinutes,
		CallCount:      ctx.CallCount,
		Account:        ctx.Account,
		Positions:      ctx.Positions,
		CandidateCoins: ctx.CandidateCoins,
		MarketDataMap:  ctx.MarketDataMap,
		QuantDataMap:   ctx.QuantDataMap,
		Config:         e.config,
	}

	return e.prompt.BuildUserPrompt(aiCtx)
}

func (e *LLMEngine) CallLLM(ctx context.Context, systemPrompt, userPrompt string) (*engine.AIResponse, error) {
	if e.mcpClient == nil {
		return nil, fmt.Errorf("MCP client not initialized")
	}

	response, err := e.mcpClient.CallWithMessages(systemPrompt, userPrompt)
	if err != nil {
		return nil, fmt.Errorf("LLM call failed: %w", err)
	}

	return &engine.AIResponse{
		RawResponse: response,
		Metadata:    make(map[string]string),
	}, nil
}

func (e *LLMEngine) ParseResponse(response *engine.AIResponse) ([]engine.Decision, error) {
	if response == nil || response.RawResponse == "" {
		return nil, fmt.Errorf("empty response")
	}

	decisions, err := e.parser.ExtractDecisions(response.RawResponse)
	if err != nil {
		return nil, err
	}

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
			Reasoning:       d.Reasoning,
		}
	}

	return engineDecisions, nil
}

func (e *LLMEngine) ValidateDecisions(ctx context.Context, decisions []engine.Decision, engCtx *engine.Context) ([]engine.ValidatedDecision, error) {
	if len(decisions) == 0 {
		return []engine.ValidatedDecision{}, nil
	}

	var accountEquity float64
	if engCtx.Account != nil {
		if acc, ok := engCtx.Account.(engine.AccountInfo); ok {
			accountEquity = acc.TotalEquity
		}
	}

	var riskConfig store.RiskControlConfig
	if e.config != nil && e.config.ChaosConfig != nil {
		riskConfig = e.config.ChaosConfig.RiskControl
	} else if e.config != nil {
		riskConfig = e.config.RiskControl
	}

	validatedDecisions := make([]engine.ValidatedDecision, 0, len(decisions))

	for _, d := range decisions {
		decision := &Decision{
			Symbol:          d.Symbol,
			Action:          d.Action,
			Leverage:        d.Leverage,
			EntryPrice:      d.EntryPrice,
			StopLoss:        d.StopLoss,
			TakeProfit:      d.TakeProfit,
			PositionSizeUSD: d.PositionSizeUSD,
		}

		positionSizeUSD, err := e.manager.ValidateDecision(decision, nil, accountEquity, riskConfig)
		if err != nil {
			continue
		}

		validatedDecisions = append(validatedDecisions, engine.ValidatedDecision{
			Decision: engine.Decision{
				Symbol:          d.Symbol,
				Action:          d.Action,
				Leverage:        d.Leverage,
				EntryPrice:      d.EntryPrice,
				StopLoss:        d.StopLoss,
				TakeProfit:      d.TakeProfit,
				PositionSizeUSD: &positionSizeUSD,
				Reasoning:       d.Reasoning,
			},
			ValidatedPositionUSD: positionSizeUSD,
			ValidationErrors:     nil,
			IsApproved:           true,
		})
	}

	return validatedDecisions, nil
}

func (e *LLMEngine) GetCandidateCoins() ([]engine.CandidateCoin, error) {
	if e.config == nil {
		return []engine.CandidateCoin{}, nil
	}

	coinSource := e.config.CoinSource
	if e.config.ChaosConfig != nil {
		coinSource = e.config.ChaosConfig.CoinSource
	}

	if coinSource.SourceType == "" {
		coinSource.SourceType = "static"
	}

	var candidates []engine.CandidateCoin
	symbolSources := make(map[string][]string)

	switch coinSource.SourceType {
	case "static":
		for _, symbol := range coinSource.StaticCoins {
			symbol = market.Normalize(symbol)
			candidates = append(candidates, engine.CandidateCoin{
				Symbol:  symbol,
				Sources: []string{"static"},
			})
		}
		return e.filterExcludedCoins(candidates, coinSource.ExcludedCoins), nil

	case "ai500":
		if !coinSource.UseAI500 {
			return []engine.CandidateCoin{}, nil
		}
		coins, err := e.getAI500Coins(coinSource.AI500Limit)
		if err != nil {
			return nil, err
		}
		return e.filterExcludedCoins(coins, coinSource.ExcludedCoins), nil

	case "oi_top":
		if !coinSource.UseOITop {
			return []engine.CandidateCoin{}, nil
		}
		coins, err := e.getOITopCoins(coinSource.OITopLimit)
		if err != nil {
			return nil, err
		}
		return e.filterExcludedCoins(coins, coinSource.ExcludedCoins), nil

	case "mixed":
		if coinSource.UseAI500 {
			poolCoins, err := e.getAI500Coins(coinSource.AI500Limit)
			if err == nil {
				for _, coin := range poolCoins {
					symbolSources[coin.Symbol] = append(symbolSources[coin.Symbol], "ai500")
				}
			}
		}

		if coinSource.UseOITop {
			oiCoins, err := e.getOITopCoins(coinSource.OITopLimit)
			if err == nil {
				for _, coin := range oiCoins {
					symbolSources[coin.Symbol] = append(symbolSources[coin.Symbol], "oi_top")
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
			candidates = append(candidates, engine.CandidateCoin{
				Symbol:  symbol,
				Sources: sources,
			})
		}
		return e.filterExcludedCoins(candidates, coinSource.ExcludedCoins), nil

	default:
		return nil, fmt.Errorf("unknown coin source type: %s", coinSource.SourceType)
	}
}

func (e *LLMEngine) filterExcludedCoins(candidates []engine.CandidateCoin, excludedCoins []string) []engine.CandidateCoin {
	if len(excludedCoins) == 0 {
		return candidates
	}

	excluded := make(map[string]bool)
	for _, coin := range excludedCoins {
		excluded[market.Normalize(coin)] = true
	}

	filtered := make([]engine.CandidateCoin, 0, len(candidates))
	for _, c := range candidates {
		if !excluded[c.Symbol] {
			filtered = append(filtered, c)
		}
	}

	return filtered
}

func (e *LLMEngine) getAI500Coins(limit int) ([]engine.CandidateCoin, error) {
	if limit <= 0 {
		limit = 30
	}

	symbols, err := e.client.GetTopRatedCoins(limit)
	if err != nil {
		return nil, err
	}

	var candidates []engine.CandidateCoin
	for _, symbol := range symbols {
		candidates = append(candidates, engine.CandidateCoin{
			Symbol:  symbol,
			Sources: []string{"ai500"},
		})
	}
	return candidates, nil
}

func (e *LLMEngine) getOITopCoins(limit int) ([]engine.CandidateCoin, error) {
	if limit <= 0 {
		limit = 10
	}

	positions, err := e.client.GetOITopPositions()
	if err != nil {
		return nil, err
	}

	var candidates []engine.CandidateCoin
	for i, pos := range positions {
		if i >= limit {
			break
		}
		symbol := market.Normalize(pos.Symbol)
		candidates = append(candidates, engine.CandidateCoin{
			Symbol:  symbol,
			Sources: []string{"oi_top"},
		})
	}
	return candidates, nil
}

func (e *LLMEngine) ExtractReasoning(response string) string {
	return e.parser.ExtractReasoning(response)
}

func (e *LLMEngine) SetClient(client *nofxos.Client) {
	e.client = client
}
