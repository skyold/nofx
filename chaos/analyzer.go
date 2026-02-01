package chaos

import (
	"encoding/json"
	"fmt"
	"nofx/kernel"
	"nofx/logger"
	"nofx/market"
	"nofx/mcp"
	"nofx/provider/nofxos"
	"nofx/store"
	"strings"
	"time"

	"github.com/google/uuid"
)

// AnalystProfile defines the configuration for a specific analyst
type AnalystProfile struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	SystemPrompt string `json:"system_prompt"` // Content of the system prompt
	ModelID      string `json:"model_id"`      // Optional override
}

// AnalysisResponse defines the expected JSON output from the LLM
type AnalysisResponse struct {
	Sentiment string   `json:"sentiment"`
	Reasoning string   `json:"reasoning"`
	Tags      []string `json:"tags"`
}

// AnalyzerEngine handles Analyzer mode execution
type AnalyzerEngine struct {
	store       *store.Store
	config      *store.StrategyConfig
	chaosEngine *ChaosEngine // Reuse ChaosEngine for shared logic
}

// NewAnalyzerEngine creates a new AnalyzerEngine
func NewAnalyzerEngine(s *store.Store, config *store.StrategyConfig) *AnalyzerEngine {
	if config == nil {
		defaultConfig := store.GetDefaultStrategyConfig("en")
		config = &defaultConfig
	}
	return &AnalyzerEngine{
		store:       s,
		config:      config,
		chaosEngine: NewChaosEngine(config),
	}
}

// RunSentinel runs a single analyst in sentinel mode
func (e *AnalyzerEngine) RunSentinel(ctx *kernel.Context, mcpClient mcp.AIClient, profile AnalystProfile) (*store.AnalysisSession, error) {
	// 1. Fetch Market Data (if missing)
	if len(ctx.MarketDataMap) == 0 {
		if err := e.fetchMarketData(ctx); err != nil {
			return nil, fmt.Errorf("failed to fetch market data: %w", err)
		}
	}

	// Ensure OITopDataMap is initialized (reusing logic from ChaosEngine)
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

	// 2. Build User Prompt (Shared)
	userPrompt := e.chaosEngine.BuildUserPrompt(ctx)

	// 3. Create Session in DB
	sessionID := uuid.New().String()
	session := &store.AnalysisSession{
		SessionID:             sessionID,
		TriggerType:           "sentinel",
		MarketContextSnapshot: userPrompt, // Snapshot the prompt as context
		CreatedAt:             time.Now(),
	}

	if err := e.store.Analysis().CreateSession(session); err != nil {
		return nil, fmt.Errorf("failed to create analysis session: %w", err)
	}

	// 4. Run Analysis
	// Use profile.SystemPrompt
	systemPrompt := profile.SystemPrompt
	if systemPrompt == "" {
		systemPrompt = "You are a crypto market analyst. Analyze the provided market data and output JSON with sentiment, reasoning, and tags."
	}

	// Call AI
	// Note: We ignore profile.ModelID for now as mcpClient is passed in.
	// In the future, we might need a factory to get client by ModelID.
	aiResponse, err := mcpClient.CallWithMessages(systemPrompt, userPrompt)
	// aiCallDuration := time.Since(aiCallStart) // Unused for now

	if err != nil {
		return nil, fmt.Errorf("AI API call failed: %w", err)
	}

	// 5. Parse Response
	var analysisResp AnalysisResponse
	// Basic cleaning of markdown code blocks if present
	cleanResp := strings.TrimSpace(aiResponse)
	cleanResp = strings.TrimPrefix(cleanResp, "```json")
	cleanResp = strings.TrimPrefix(cleanResp, "```")
	cleanResp = strings.TrimSuffix(cleanResp, "```")
	cleanResp = strings.TrimSpace(cleanResp)

	if err := json.Unmarshal([]byte(cleanResp), &analysisResp); err != nil {
		// Fallback for non-JSON response or partial failure
		// We still record it but with RawResponse
		logger.Warnf("Failed to parse analysis JSON: %v", err)
		analysisResp.Reasoning = aiResponse // Treat whole response as reasoning
		analysisResp.Sentiment = "unknown"
	}

	// 6. Save Record
	record := &store.AnalysisRecord{
		SessionID:   sessionID,
		AnalystID:   profile.ID,
		Sentiment:   analysisResp.Sentiment,
		Reasoning:   analysisResp.Reasoning,
		Tags:        analysisResp.Tags,
		RawResponse: aiResponse,
		CreatedAt:   time.Now(),
	}

	if err := e.store.Analysis().CreateRecord(record); err != nil {
		return nil, fmt.Errorf("failed to create analysis record: %w", err)
	}

	// Return the session with the record
	session.Records = []store.AnalysisRecord{*record}
	return session, nil
}

// fetchMarketData fetches market data for the context (Duplicated from ChaosEngine for now)
func (e *AnalyzerEngine) fetchMarketData(ctx *kernel.Context) error {
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

	logger.Infof("📊 Analyzer Strategy timeframes: %v, Primary: %s, Kline count: %d", timeframes, primaryTimeframe, klineCount)

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
