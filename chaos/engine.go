package chaos

import (
	"fmt"
	kernel "nofx/kernel"
	"nofx/logger"
	"nofx/market"
	"nofx/mcp"
	"nofx/provider/nofxos"
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

// BuildSystemPrompt builds the system prompt for Chaos mode
func (e *ChaosEngine) BuildSystemPrompt(accountEquity float64, variant string) string {
	var sb strings.Builder

	// Use ChaosConfig if available (Phase 2 refactoring will fully migrate to this)
	// For now, if config.ChaosConfig exists, use it. Otherwise fallback to old logic.
	if e.config.ChaosConfig != nil && e.config.ChaosConfig.ChaosPrompt != "" {
		sb.WriteString(e.config.ChaosConfig.ChaosPrompt)
		sb.WriteString("\n\n")
		sb.WriteString("# Available Indicators\n")
		e.writeAvailableIndicators(&sb)
		return sb.String()
	}

	// Fallback to old Manager logic (will be deprecated in Phase 2)
	return e.manager.BuildPrompt(variant, e.config.CustomPrompt, func(sb *strings.Builder) {
		e.writeAvailableIndicators(sb)
	})
}

// GetFullDecision gets the full decision for Chaos mode
func GetFullDecision(ctx *kernel.Context, mcpClient mcp.AIClient) (*kernel.FullDecision, error) {
	// Use default config if not provided in context (similar to kernel.GetFullDecision)
	// In a real integration, we might inject config differently.
	defaultConfig := store.GetDefaultStrategyConfig("en")
	engine := NewChaosEngine(&defaultConfig)
	return engine.Execute(ctx, mcpClient)
}

// Execute runs the Chaos decision process
func (e *ChaosEngine) Execute(ctx *kernel.Context, mcpClient mcp.AIClient) (*kernel.FullDecision, error) {
	// 1. Fetch Market Data (if missing)
	if len(ctx.MarketDataMap) == 0 {
		if err := e.fetchMarketData(ctx); err != nil {
			return nil, fmt.Errorf("failed to fetch market data: %w", err)
		}
	}

	// Ensure OITopDataMap is initialized
	if ctx.OITopDataMap == nil {
		ctx.OITopDataMap = make(map[string]*kernel.OITopData)
		apiKey := e.config.Indicators.NofxOSAPIKey
		if apiKey == "" {
			apiKey = nofxos.DefaultAuthKey
		}
		client := nofxos.NewClient(nofxos.DefaultBaseURL, apiKey)

		oiPositions, err := client.GetOITopPositions()
		if err == nil {
			for _, pos := range oiPositions {
				ctx.OITopDataMap[pos.Symbol] = &kernel.OITopData{
					Rank:              pos.Rank,
					OIDeltaPercent:    pos.OIDeltaPercent,
					OIDeltaValue:      pos.OIDeltaValue,
					PriceDeltaPercent: pos.PriceDeltaPercent,
				}
			}
		}
	}

	// 2. Build Prompts
	systemPrompt := e.manager.BuildPrompt(ctx.PromptVariant, e.config.CustomPrompt, func(sb *strings.Builder) {
		e.writeAvailableIndicators(sb)
	})

	// Use local BuildUserPrompt instead of reusing kernel logic
	userPrompt := e.BuildUserPrompt(ctx)

	// 3. Call AI
	aiCallStart := time.Now()
	aiResponse, err := mcpClient.CallWithMessages(systemPrompt, userPrompt)
	aiCallDuration := time.Since(aiCallStart)
	if err != nil {
		return nil, fmt.Errorf("AI API call failed: %w", err)
	}

	// 4. Parse & Validate
	// Step 4.1: Format Audit (Is the JSON valid? Does it follow the schema?)
	// This step verifies if the LLM followed the communication protocol (JSON format, field types, etc.)
	decisions, err := extractDecisions(aiResponse)
	if err != nil {
		fullDecision := &kernel.FullDecision{
			SystemPrompt:        systemPrompt,
			UserPrompt:          userPrompt,
			CoTTrace:            e.manager.ExtractReasoning(aiResponse),
			RawResponse:         aiResponse,
			Timestamp:           time.Now(),
			AIRequestDurationMs: aiCallDuration.Milliseconds(),
			Decisions:           []kernel.Decision{},
		}
		// If extraction failed completely, return error
		return fullDecision, fmt.Errorf("format audit failed (JSON parsing error): %w", err)
	}

	// Prepare result with initial parsed decisions
	// Note: We need to convert chaos.Decision (pointers) to a structure suitable for RawDecisions if it expects values,
	// but RawDecisions is []chaos.Decision, so it's fine.
	fullDecision := &kernel.FullDecision{
		SystemPrompt:        systemPrompt,
		UserPrompt:          userPrompt,
		CoTTrace:            e.manager.ExtractReasoning(aiResponse),
		RawResponse:         aiResponse,
		RawDecisions:        decisions, // Store raw decisions for audit visibility
		Timestamp:           time.Now(),
		AIRequestDurationMs: aiCallDuration.Milliseconds(),
	}

	// Extract Reasoning for strict validation
	reasoning, errReasoning := extractReasoningJSON(aiResponse)
	if errReasoning != nil {
		logger.Warnf("⚠️  [Format Audit] Failed to parse reasoning JSON: %v. Strict validation might be skipped or limited.", errReasoning)
	}

	// Check 1: Unique symbol assertion
	seenSymbols := make(map[string]bool)
	for _, d := range decisions {
		if seenSymbols[d.Symbol] {
			return fullDecision, fmt.Errorf("Strict Constraint Violation: Duplicate symbol %s found in decisions", d.Symbol)
		}
		seenSymbols[d.Symbol] = true
	}

	// Step 4.2: Content Audit (Risk Control & Business Logic)
	// This step validates the business logic (Action validity, R:R ratio) and enforces risk controls (RiskR limits)
	var kernelDecisions []kernel.Decision
	riskConfig := e.config.RiskControl

	for i, d := range decisions {
		// Validate using Chaos Manager and get calculated PositionSizeUSD
		positionSizeUSD, err := e.manager.ValidateDecision(&d, reasoning, ctx.Account.TotalEquity,
			riskConfig.BTCETHMaxLeverage, riskConfig.AltcoinMaxLeverage,
			riskConfig.BTCETHMaxPositionValueRatio, riskConfig.AltcoinMaxPositionValueRatio)

		if err != nil {
			// If content audit fails, we should log it but we might still want to return the raw response
			// for debugging in Chaos Studio. However, for AutoTrader, this is a failure.
			return fullDecision, fmt.Errorf("content audit failed for decision #%d: %w", i+1, err)
		}

		// Safe dereference for kernel.Decision
		var entry, sl, tp, riskR float64
		var leverage, score int

		if d.EntryPrice != nil {
			entry = *d.EntryPrice
		}
		if d.StopLoss != nil {
			sl = *d.StopLoss
		}
		if d.TakeProfit != nil {
			tp = *d.TakeProfit
		}
		if d.RiskR != nil {
			riskR = *d.RiskR
		}
		if d.Leverage != nil {
			leverage = *d.Leverage
		}
		if d.TotalScore != nil {
			score = *d.TotalScore
		}

		// Map to kernel.Decision (Processed Result)
		kd := kernel.Decision{
			Symbol:          d.Symbol,
			Action:          d.Action,
			Leverage:        leverage,
			PositionSizeUSD: positionSizeUSD, // Use the calculated value returned by ValidateDecision
			StopLoss:        sl,
			TakeProfit:      tp,
			Confidence:      score,
			// Reasoning removed from chaos.Decision as per new design
			// AI Reasoning is now captured at the top level via CoTTrace
			Reasoning:  "",
			EntryPrice: entry,
			RiskR:      riskR,
		}
		kernelDecisions = append(kernelDecisions, kd)
	}

	fullDecision.Decisions = kernelDecisions
	return fullDecision, nil
}

func (e *ChaosEngine) fetchMarketData(ctx *kernel.Context) error {
	ctx.MarketDataMap = make(map[string]*market.Data)

	timeframes := e.config.Indicators.Klines.SelectedTimeframes
	primaryTimeframe := e.config.Indicators.Klines.PrimaryTimeframe
	klineCount := e.config.Indicators.Klines.PrimaryCount

	// Compatible with old configuration
	if len(timeframes) == 0 {
		if primaryTimeframe != "" {
			timeframes = append(timeframes, primaryTimeframe)
		} else {
			timeframes = append(timeframes, "3m")
		}
		if e.config.Indicators.Klines.LongerTimeframe != "" {
			timeframes = append(timeframes, e.config.Indicators.Klines.LongerTimeframe)
		}
	}
	if primaryTimeframe == "" {
		primaryTimeframe = timeframes[0]
	}
	if klineCount <= 0 {
		klineCount = 30
	}

	logger.Infof("📊 Chaos Strategy timeframes: %v, Primary: %s, Kline count: %d", timeframes, primaryTimeframe, klineCount)

	// 1. First fetch data for position coins
	for _, pos := range ctx.Positions {
		data, err := market.GetWithTimeframes(pos.Symbol, timeframes, primaryTimeframe, klineCount)
		if err != nil {
			logger.Infof("⚠️  Failed to fetch market data for position %s: %v", pos.Symbol, err)
			continue
		}
		ctx.MarketDataMap[pos.Symbol] = data
	}

	// 2. Fetch data for all candidate coins
	positionSymbols := make(map[string]bool)
	for _, pos := range ctx.Positions {
		positionSymbols[pos.Symbol] = true
	}

	const minOIThresholdMillions = 15.0

	for _, coin := range ctx.CandidateCoins {
		if _, exists := ctx.MarketDataMap[coin.Symbol]; exists {
			continue
		}

		data, err := market.GetWithTimeframes(coin.Symbol, timeframes, primaryTimeframe, klineCount)
		if err != nil {
			logger.Infof("⚠️  Failed to fetch market data for %s: %v", coin.Symbol, err)
			continue
		}

		// Liquidity filter
		isExistingPosition := positionSymbols[coin.Symbol]
		isXyzAsset := market.IsXyzDexAsset(coin.Symbol)
		if !isExistingPosition && !isXyzAsset && data.OpenInterest != nil && data.CurrentPrice > 0 {
			oiValue := data.OpenInterest.Latest * data.CurrentPrice
			oiValueInMillions := oiValue / 1_000_000
			if oiValueInMillions < minOIThresholdMillions {
				logger.Infof("⚠️  %s OI value too low (%.2fM USD < %.1fM), skipping coin",
					coin.Symbol, oiValueInMillions, minOIThresholdMillions)
				continue
			}
		}

		ctx.MarketDataMap[coin.Symbol] = data
	}

	return nil
}

func (e *ChaosEngine) writeAvailableIndicators(sb *strings.Builder) {
	indicators := e.config.Indicators
	kline := indicators.Klines

	primaryTimeframe := strings.TrimSpace(kline.PrimaryTimeframe)
	selectedTimeframes := kline.SelectedTimeframes
	if len(selectedTimeframes) == 0 {
		if primaryTimeframe != "" {
			selectedTimeframes = append(selectedTimeframes, primaryTimeframe)
		}
		if kline.EnableMultiTimeframe && strings.TrimSpace(kline.LongerTimeframe) != "" {
			selectedTimeframes = append(selectedTimeframes, strings.TrimSpace(kline.LongerTimeframe))
		}
	}
	if primaryTimeframe == "" && len(selectedTimeframes) > 0 {
		primaryTimeframe = selectedTimeframes[0]
	}

	ordered := make([]string, 0, len(selectedTimeframes)+1)
	seen := map[string]struct{}{}
	if primaryTimeframe != "" {
		ordered = append(ordered, primaryTimeframe)
		seen[primaryTimeframe] = struct{}{}
	}
	for _, tf := range selectedTimeframes {
		tf = strings.TrimSpace(tf)
		if tf == "" {
			continue
		}
		if _, ok := seen[tf]; ok {
			continue
		}
		ordered = append(ordered, tf)
		seen[tf] = struct{}{}
	}

	if len(ordered) == 0 {
		sb.WriteString("- price series\n")
	} else {
		sb.WriteString(fmt.Sprintf("- %s price series", ordered[0]))
		if len(ordered) > 1 {
			sb.WriteString(fmt.Sprintf(" + %s K-line series\n", strings.Join(ordered[1:], " + ")))
		} else {
			sb.WriteString("\n")
		}
	}

	if indicators.EnableEMA {
		sb.WriteString("- EMA indicators")
		if len(indicators.EMAPeriods) > 0 {
			sb.WriteString(fmt.Sprintf(" (periods: %v)", indicators.EMAPeriods))
		}
		sb.WriteString("\n")
	}

	if indicators.EnableMACD {
		sb.WriteString("- MACD indicators\n")
	}

	if indicators.EnableRSI {
		sb.WriteString("- RSI indicators")
		if len(indicators.RSIPeriods) > 0 {
			sb.WriteString(fmt.Sprintf(" (periods: %v)", indicators.RSIPeriods))
		}
		sb.WriteString("\n")
	}

	if indicators.EnableATR {
		sb.WriteString("- ATR indicators")
		if len(indicators.ATRPeriods) > 0 {
			sb.WriteString(fmt.Sprintf(" (periods: %v)", indicators.ATRPeriods))
		}
		sb.WriteString("\n")
	}

	if indicators.EnableBOLL {
		sb.WriteString("- Bollinger Bands (BOLL) - Upper/Middle/Lower bands")
		if len(indicators.BOLLPeriods) > 0 {
			sb.WriteString(fmt.Sprintf(" (periods: %v)", indicators.BOLLPeriods))
		}
		sb.WriteString("\n")
	}

	if indicators.EnableVolume {
		sb.WriteString("- Volume data\n")
	}

	if indicators.EnableOI {
		sb.WriteString("- Open Interest (OI) data\n")
	}

	if indicators.EnableFundingRate {
		sb.WriteString("- Funding rate\n")
	}

	if len(e.config.CoinSource.StaticCoins) > 0 || e.config.CoinSource.UseAI500 || e.config.CoinSource.UseOITop {
		sb.WriteString("- AI500 / OI_Top filter tags (if available)\n")
	}

	if indicators.EnableQuantData {
		sb.WriteString("- Quantitative data (institutional/retail fund flow, position changes, multi-period price changes)\n")
	}
}
