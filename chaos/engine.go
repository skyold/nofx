package chaos

import (
	"fmt"
	"nofx/mcp"
	"nofx/store"
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
func GetChaosDecisions(ctx *ChaosContext, mcpClient mcp.AIClient) (*DecisionResult, error) {
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
	userPrompt := e.BuildUserPromptFromChaosContext(ctx)

	// 2. Call AI
	aiCallStart := time.Now()
	aiResponse, err := mcpClient.CallWithMessages(systemPrompt, userPrompt)
	aiCallDuration := time.Since(aiCallStart)
	if err != nil {
		return nil, fmt.Errorf("AI API call failed: %w", err)
	}

	// 3. Parse & Validate
	decisions, decisionJSON, err := extractDecisions(aiResponse)

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
	reasoning, _ := extractReasoningJSON(aiResponse)

	// Validate decisions
	validatedDecisions, err := e.validateDecisions(decisions, reasoning, ctx)
	if err != nil {
		return result, fmt.Errorf("content audit failed: %w", err)
	}

	result.Decisions = validatedDecisions



	

	return result, nil
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
