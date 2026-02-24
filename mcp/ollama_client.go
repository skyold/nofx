package mcp

import (
	"net/http"
)

const (
	ProviderOllama       = "ollama"
	DefaultOllamaBaseURL = "http://localhost:11434/v1"
	DefaultOllamaModel   = "llama3"
)

type OllamaClient struct {
	*Client
}

// NewOllamaClient creates Ollama client (backward compatible)
func NewOllamaClient() AIClient {
	return NewOllamaClientWithOptions()
}

// NewOllamaClientWithOptions creates Ollama client (supports options pattern)
func NewOllamaClientWithOptions(opts ...ClientOption) AIClient {
	// 1. Create Ollama preset options
	ollamaOpts := []ClientOption{
		WithProvider(ProviderOllama),
		WithModel(DefaultOllamaModel),
		WithBaseURL(DefaultOllamaBaseURL),
	}

	// 2. Merge user options (user options have higher priority)
	allOpts := append(ollamaOpts, opts...)

	// 3. Create base client
	baseClient := NewClient(allOpts...).(*Client)

	// 4. Create Ollama client
	ollamaClient := &OllamaClient{
		Client: baseClient,
	}

	// 5. Set hooks to point to OllamaClient (implement dynamic dispatch)
	baseClient.hooks = ollamaClient

	return ollamaClient
}

func (c *OllamaClient) SetAPIKey(apiKey string, customURL string, customModel string) {
	// Ollama usually doesn't require an API key, but we keep the field for consistency
	c.APIKey = apiKey

	if len(apiKey) > 8 {
		c.logger.Infof("🔧 [MCP] Ollama API Key: %s...%s", apiKey[:4], apiKey[len(apiKey)-4:])
	}
	if customURL != "" {
		c.BaseURL = customURL
		c.logger.Infof("🔧 [MCP] Ollama using custom BaseURL: %s", customURL)
	} else {
		c.logger.Infof("🔧 [MCP] Ollama using default BaseURL: %s", c.BaseURL)
	}
	if customModel != "" {
		c.Model = customModel
		c.logger.Infof("🔧 [MCP] Ollama using custom Model: %s", customModel)
	} else {
		c.logger.Infof("🔧 [MCP] Ollama using default Model: %s", c.Model)
	}
}

// Ollama uses standard Bearer auth if provided, though often optional for local instances
func (c *OllamaClient) setAuthHeader(reqHeaders http.Header) {
	if c.APIKey != "" {
		c.Client.setAuthHeader(reqHeaders)
	}
}
