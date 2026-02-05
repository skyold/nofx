package chaos

import (
	"fmt"
	"nofx/kernel"
	"nofx/mcp"
	"nofx/store"
	"strings"
	"time"
)

// ChaosEngine handles Chaos mode execution
type ChaosEngine struct {
	manager *Manager
	config  *store.StrategyConfig
}

// NewChaosEngine creates a new ChaosEngine
func NewChaosEngine(config *store.StrategyConfig) *ChaosEngine {
	if config == nil {
		defaultConfig := store.GetDefaultStrategyConfig("en")
		config = &defaultConfig
	}
	return &ChaosEngine{
		manager: NewManager(),
		config:  config,
	}
}

// GetDecisions gets the decisions for Chaos mode
func GetDecisions(ctx *ChaosContext, mcpClient mcp.AIClient) (*DecisionResult, error) {
	// Create engine with context config
	// The config is already in the context, but NewChaosEngine expects *store.StrategyConfig
	// We need to refactor NewChaosEngine or create a temporary config adapter
	// For now, let's update Execute to use ChaosContext directly

	// Since ChaosEngine.config is store.StrategyConfig, but we are moving away from it in Phase 2.
	// We will instantiate ChaosEngine with nil or default, but Execute will use ctx.Config.

	// We need a way to pass the full strategy config if we still need it for non-chaos parts (like indicators)
	// But ChaosContext should have everything.

	// Let's create a minimal engine instance
	engine := &ChaosEngine{
		manager: NewManager(),
		// Config is now inside ctx, engine struct config might be deprecated
	}

	return engine.Execute(ctx, mcpClient)
}

// Execute runs the Chaos decision process
func (e *ChaosEngine) Execute(ctx *ChaosContext, mcpClient mcp.AIClient) (*DecisionResult, error) {
	// 1. Build Prompts
	systemPrompt := e.buildSystemPromptWithContext(ctx)
	userPrompt := e.buildUserPromptWithContext(ctx)

	// 2. Call AI
	aiCallStart := time.Now()
	aiResponse, err := mcpClient.CallWithMessages(systemPrompt, userPrompt)
	aiCallDuration := time.Since(aiCallStart)
	if err != nil {
		return nil, fmt.Errorf("AI API call failed: %w", err)
	}

	// 3. Parse & Validate
	decisions, err := extractDecisions(aiResponse)

	// Create result structure
	result := &DecisionResult{
		SystemPrompt:        systemPrompt,
		UserPrompt:          userPrompt,
		CoTTrace:            e.manager.ExtractReasoning(aiResponse),
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
	reasoning, _ := extractReasoningJSON(aiResponse)

	// Validate decisions
	validatedDecisions, err := e.validateDecisions(decisions, reasoning, ctx)
	if err != nil {
		return result, fmt.Errorf("content audit failed: %w", err)
	}

	result.Decisions = validatedDecisions
	return result, nil
}

// BuildSystemPrompt implements PromptBuilder interface for API compatibility
func (e *ChaosEngine) BuildSystemPrompt(accountEquity float64, variant string) string {
	// Construct a temporary context for preview
	var chaosPrompt string
	var riskControl store.RiskControlConfig
	var indicators store.IndicatorConfig

	if e.config != nil {
		// Default to top-level indicators (legacy)
		indicators = e.config.Indicators
		if e.config.ChaosConfig != nil {
			chaosPrompt = e.config.ChaosConfig.ChaosPrompt
			riskControl = e.config.ChaosConfig.RiskControl
			// If ChaosConfig has Indicators populated (check if non-zero), use them
			// Simple check: if PrimaryTimeframe is set, assume populated
			if e.config.ChaosConfig.Indicators.Klines.PrimaryTimeframe != "" {
				indicators = e.config.ChaosConfig.Indicators
			}
		}
	}

	ctx := &ChaosContext{
		Config: &ChaosConfig{
			ChaosPrompt:   chaosPrompt,
			RiskControl:   riskControl,
			PromptVariant: variant,
			Indicators:    indicators,
		},
	}
	return e.buildSystemPromptWithContext(ctx)
}

// buildSystemPromptWithContext builds the system prompt for Chaos mode using context
func (e *ChaosEngine) buildSystemPromptWithContext(ctx *ChaosContext) string {
	if ctx.Config != nil && ctx.Config.ChaosPrompt != "" {
		// Use manager to build the full prompt including Contract and Footer
		// We pass indicators config for dynamic market data generation
		return e.manager.BuildPrompt(
			ctx.Config.PromptVariant,
			ctx.Config.ChaosPrompt,
			ctx.Config.Indicators,
		)
	}

	// Fallback should ideally not happen in pure Chaos mode, but if config is missing:
	return "# Chaos Mode (Missing Configuration)\n\nPlease provide trading decisions based on market data."
}

// BuildUserPrompt implements PromptBuilder interface for API compatibility
func (e *ChaosEngine) BuildUserPrompt(ctx *kernel.Context) string {
	return e.BuildUserPromptFromKernel(ctx)
}

// buildUserPromptWithContext builds the user prompt using ChaosContext
func (e *ChaosEngine) buildUserPromptWithContext(ctx *ChaosContext) string {
	// This should be implemented similar to BuildGridUserPrompt or kernel.BuildUserPrompt
	// But using ChaosContext data
	// For now, let's implement a basic version that includes market data

	var sb strings.Builder
	sb.WriteString(fmt.Sprintf("Time: %s | Cycle: #%d\n\n", ctx.CurrentTime, ctx.CallCount))

	// Account info
	sb.WriteString(fmt.Sprintf("Account: Equity %.2f | Available %.2f | PnL %+.2f%%\n\n",
		ctx.Account.TotalEquity, ctx.Account.AvailableBalance, ctx.Account.TotalPnLPct))

	// Market Data
	sb.WriteString("## Market Data\n\n")
	for _, coin := range ctx.CandidateCoins {
		if data, ok := ctx.MarketDataMap[coin.Symbol]; ok {
			sb.WriteString(fmt.Sprintf("### %s\n", coin.Symbol))
			sb.WriteString(fmt.Sprintf("Price: %.4f\n", data.CurrentPrice))
			// Add more indicators...
			sb.WriteString("\n")
		}
	}

	return sb.String()
}

func (e *ChaosEngine) validateDecisions(decisions []Decision, reasoning *Reasoning, ctx *ChaosContext) ([]Decision, error) {
	// Implement validation using ctx.Config.RiskControl
	// This logic mirrors the kernel validation but uses Chaos structures

	var validated []Decision
	riskConfig := ctx.Config.RiskControl

	for _, d := range decisions {
		// Use Manager for validation which contains the core logic
		// We pass ChaosConfig.RiskControl params

		// Note: Manager.ValidateDecision returns calculated position size USD
		positionSizeUSD, err := e.manager.ValidateDecision(&d, reasoning, ctx.Account.TotalEquity,
			riskConfig.BTCETHMaxLeverage, riskConfig.AltcoinMaxLeverage,
			riskConfig.BTCETHMaxPositionValueRatio, riskConfig.AltcoinMaxPositionValueRatio)

		if err != nil {
			return nil, fmt.Errorf("validation failed for %s: %w", d.Symbol, err)
		}

		// Store validated position size
		d.PositionSizeUSD = &positionSizeUSD
		validated = append(validated, d)
	}
	return validated, nil
}
