package chaos

import (
	"nofx/store"
)

// BuildSystemPrompt implements PromptBuilder interface for API compatibility
func (e *ChaosEngine) BuildSystemPrompt(accountEquity float64, variant string) string {
	// Construct a temporary context for preview
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
			ChaosPrompt:   chaosPrompt,
			RiskControl:   riskControl,
			SystemPromptVariant: variant,
			Indicators:    indicators,
		},
	}
	return e.buildSystemPromptWithChaosContext(ctx)
}

// buildSystemPromptWithChaosContext builds the system prompt for Chaos mode using context
func (e *ChaosEngine) buildSystemPromptWithChaosContext(ctx *ChaosContext) string {
	if ctx.Config != nil && ctx.Config.ChaosPrompt != "" {
		// Use manager to build the full prompt including Contract and Footer
		// We pass indicators config for dynamic market data generation
		return e.manager.BuildSystemPrompt(
			ctx.Config.SystemPromptVariant,
			ctx.Config.ChaosPrompt,
			ctx.Config.Indicators,
		)
	}

	// Fallback should ideally not happen in pure Chaos mode, but if config is missing:
	return "# Chaos Mode (Missing Configuration)\n\nPlease provide trading decisions based on market data."
}
