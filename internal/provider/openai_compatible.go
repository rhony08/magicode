// Package provider provides OpenAI-compatible provider implementations.
// These providers use the OpenAI API format but with different base URLs.
package provider

import (
	"context"
)

// OpenAICompatibleProvider wraps OpenAI provider with custom configuration
type OpenAICompatibleProvider struct {
	*OpenAIProvider
	providerID ProviderID
	name       string
	envKeys    []string
}

// ID returns the provider ID
func (p *OpenAICompatibleProvider) ID() ProviderID {
	return p.providerID
}

// Info returns provider information
func (p *OpenAICompatibleProvider) Info() ProviderInfo {
	return ProviderInfo{
		ID:          p.providerID,
		Name:        p.name,
		Description: getProviderDescription(p.providerID),
		BaseURL:     p.baseURL,
		EnvKeys:     p.envKeys,
		Models:      p.models,
	}
}

// getProviderDescription returns a description for each provider
func getProviderDescription(id ProviderID) string {
	switch id {
	case ProviderOpenRouter:
		return "OpenRouter - unified API for many LLM providers"
	case ProviderGroq:
		return "Groq - ultra-fast inference with LPU technology"
	case ProviderMistral:
		return "Mistral AI - efficient open-source models"
	case ProviderTogetherAI:
		return "Together AI - open-source model hosting"
	case ProviderPerplexity:
		return "Perplexity - AI-powered search and reasoning"
	case ProviderXAI:
		return "X.AI - Grok models by xAI"
	case ProviderCerebras:
		return "Cerebras - fast inference with wafer-scale engine"
	case ProviderDeepInfra:
		return "DeepInfra - cost-effective inference platform"
	case ProviderAlibaba:
		return "Alibaba Cloud - Qwen models via DashScope"
	case ProviderAlibabaCN:
		return "Alibaba Cloud CN - Qwen models via DashScope"
	default:
		return "OpenAI-compatible provider"
	}
}

// Bundled provider configurations
var bundledProviderConfigs = map[ProviderID]struct {
	baseURL string
	envKeys []string
	name    string
}{
	ProviderOpenRouter: {
		baseURL: "https://openrouter.ai/api/v1",
		envKeys: []string{"OPENROUTER_API_KEY"},
		name:    "OpenRouter",
	},
	ProviderGroq: {
		baseURL: "https://api.groq.com/openai/v1",
		envKeys: []string{"GROQ_API_KEY"},
		name:    "Groq",
	},
	ProviderMistral: {
		baseURL: "https://api.mistral.ai/v1",
		envKeys: []string{"MISTRAL_API_KEY"},
		name:    "Mistral",
	},
	ProviderTogetherAI: {
		baseURL: "https://api.together.xyz/v1",
		envKeys: []string{"TOGETHERAI_API_KEY"},
		name:    "Together AI",
	},
	ProviderPerplexity: {
		baseURL: "https://api.perplexity.ai",
		envKeys: []string{"PERPLEXITY_API_KEY"},
		name:    "Perplexity",
	},
	ProviderXAI: {
		baseURL: "https://api.x.ai/v1",
		envKeys: []string{"XAI_API_KEY"},
		name:    "X.AI",
	},
	ProviderCerebras: {
		baseURL: "https://api.cerebras.ai/v1",
		envKeys: []string{"CEREBRAS_API_KEY"},
		name:    "Cerebras",
	},
	ProviderDeepInfra: {
		baseURL: "https://api.deepinfra.com/v1/openai",
		envKeys: []string{"DEEPINFRA_API_KEY"},
		name:    "DeepInfra",
	},
	// Alibaba/Bailian providers
	ProviderAlibaba: {
		baseURL: "https://dashscope.aliyuncs.com/compatible-mode/v1",
		envKeys: []string{"ALIBABA_API_KEY", "DASHSCOPE_API_KEY"},
		name:    "Alibaba Cloud",
	},
	ProviderAlibabaCN: {
		baseURL: "https://dashscope.aliyuncs.com/compatible-mode/v1",
		envKeys: []string{"ALIBABA_API_KEY", "DASHSCOPE_API_KEY"},
		name:    "Alibaba Cloud (CN)",
	},
}

// NewOpenAICompatibleProvider creates a provider for a bundled OpenAI-compatible service
func NewOpenAICompatibleProvider(providerID ProviderID, apiKey string) *OpenAICompatibleProvider {
	config, ok := bundledProviderConfigs[providerID]
	if !ok {
		return nil
	}

	p := NewOpenAIProvider(apiKey)
	p.SetBaseURL(config.baseURL)

	return &OpenAICompatibleProvider{
		OpenAIProvider: p,
		providerID:     providerID,
		name:           config.name,
		envKeys:        config.envKeys,
	}
}

// NewOpenRouterProvider creates an OpenRouter provider
func NewOpenRouterProvider(apiKey string) *OpenAICompatibleProvider {
	return NewOpenAICompatibleProvider(ProviderOpenRouter, apiKey)
}

// NewGroqProvider creates a Groq provider
func NewGroqProvider(apiKey string) *OpenAICompatibleProvider {
	return NewOpenAICompatibleProvider(ProviderGroq, apiKey)
}

// NewMistralProvider creates a Mistral provider
func NewMistralProvider(apiKey string) *OpenAICompatibleProvider {
	return NewOpenAICompatibleProvider(ProviderMistral, apiKey)
}

// NewTogetherAIProvider creates a Together AI provider
func NewTogetherAIProvider(apiKey string) *OpenAICompatibleProvider {
	return NewOpenAICompatibleProvider(ProviderTogetherAI, apiKey)
}

// NewPerplexityProvider creates a Perplexity provider
func NewPerplexityProvider(apiKey string) *OpenAICompatibleProvider {
	return NewOpenAICompatibleProvider(ProviderPerplexity, apiKey)
}

// NewXAIProvider creates an X.AI provider
func NewXAIProvider(apiKey string) *OpenAICompatibleProvider {
	return NewOpenAICompatibleProvider(ProviderXAI, apiKey)
}

// NewCerebrasProvider creates a Cerebras provider
func NewCerebrasProvider(apiKey string) *OpenAICompatibleProvider {
	return NewOpenAICompatibleProvider(ProviderCerebras, apiKey)
}

// NewDeepInfraProvider creates a DeepInfra provider
func NewDeepInfraProvider(apiKey string) *OpenAICompatibleProvider {
	return NewOpenAICompatibleProvider(ProviderDeepInfra, apiKey)
}

// NewAlibabaProvider creates a new Alibaba Cloud provider
func NewAlibabaProvider(apiKey string) *OpenAICompatibleProvider {
	return NewOpenAICompatibleProvider(ProviderAlibaba, apiKey)
}

// NewAlibabaCNProvider creates a new Alibaba Cloud CN provider
func NewAlibabaCNProvider(apiKey string) *OpenAICompatibleProvider {
	return NewOpenAICompatibleProvider(ProviderAlibabaCN, apiKey)
}

// CustomOpenAICompatibleProvider creates a provider with custom configuration
// This is used for dynamic providers loaded from OpenCode config
func CustomOpenAICompatibleProvider(providerID ProviderID, name, baseURL, apiKey string, models map[ModelID]ModelInfo) *OpenAICompatibleProvider {
	p := NewOpenAIProvider(apiKey)
	p.SetBaseURL(baseURL)
	p.models = models

	return &OpenAICompatibleProvider{
		OpenAIProvider: p,
		providerID:     providerID,
		name:           name,
		envKeys:        []string{}, // Custom providers don't use env keys
	}
}

// ValidateKey checks if the API key is valid
func (p *OpenAICompatibleProvider) ValidateKey(ctx context.Context) error {
	if p.apiKey == "" {
		return providerKeyError(p.providerID, "API key is not set")
	}

	// Make a minimal request to validate the key
	req := ChatRequest{
		Model:     FormatModelID(p.providerID, getTestModel(p.providerID)),
		MaxTokens: 1,
		Messages:  []Message{{Role: RoleUser, Content: "Hi"}},
	}

	_, err := p.Chat(ctx, req)
	if err != nil {
		if isInvalidKeyError(err) {
			return providerKeyError(p.providerID, "invalid API key")
		}
		// Rate limits mean key is valid
		if isRateLimitError(err) {
			return nil
		}
	}
	return err
}

// getTestModel returns a model to use for key validation
func getTestModel(providerID ProviderID) string {
	// Use small/fast models for validation
	switch providerID {
	case ProviderOpenRouter:
		return "openai/gpt-3.5-turbo" // OpenRouter uses provider/model format
	case ProviderGroq:
		return "llama-3.1-8b-instant"
	case ProviderMistral:
		return "mistral-small-latest"
	case ProviderTogetherAI:
		return "meta-llama/Llama-3-8b-chat-hf"
	case ProviderPerplexity:
		return "llama-3.1-sonar-small-128k-online"
	case ProviderXAI:
		return "grok-beta"
	case ProviderCerebras:
		return "llama3.1-8b"
	case ProviderDeepInfra:
		return "meta-llama/Llama-3-8b-chat-hf"
	default:
		return "gpt-3.5-turbo"
	}
}

// Helper error detection functions
func isInvalidKeyError(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return containsAny(msg, []string{
		"invalid api_key",
		"Incorrect API key",
		"invalid_api_key",
		"Unauthorized",
		"authentication failed",
	})
}

func isRateLimitError(err error) bool {
	if err == nil {
		return false
	}
	msg := err.Error()
	return containsAny(msg, []string{
		"rate limit",
		"rate_limit",
		"429",
		"Too Many Requests",
	})
}

func containsAny(s string, substrs []string) bool {
	for _, substr := range substrs {
		if contains(s, substr) {
			return true
		}
	}
	return false
}

func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func providerKeyError(providerID ProviderID, msg string) error {
	return &ProviderKeyError{
		ProviderID: providerID,
		Message:    msg,
	}
}

// ProviderKeyError represents an API key validation error
type ProviderKeyError struct {
	ProviderID ProviderID
	Message    string
}

func (e *ProviderKeyError) Error() string {
	return string(e.ProviderID) + ": " + e.Message
}

// Popular models for each provider (used when no models defined)
var PopularModels = map[ProviderID][]ModelInfo{
	ProviderOpenRouter: {
		{ID: FormatModelID(ProviderOpenRouter, "anthropic/claude-3.5-sonnet"), Name: "Claude 3.5 Sonnet (via OpenRouter)", SupportsTools: true, SupportsStreaming: true},
		{ID: FormatModelID(ProviderOpenRouter, "openai/gpt-4o"), Name: "GPT-4o (via OpenRouter)", SupportsTools: true, SupportsStreaming: true},
		{ID: FormatModelID(ProviderOpenRouter, "google/gemini-pro-1.5"), Name: "Gemini Pro 1.5 (via OpenRouter)", SupportsTools: true, SupportsStreaming: true},
	},
	ProviderGroq: {
		{ID: FormatModelID(ProviderGroq, "llama-3.3-70b-versatile"), Name: "Llama 3.3 70B", SupportsTools: true, SupportsStreaming: true},
		{ID: FormatModelID(ProviderGroq, "llama-3.1-8b-instant"), Name: "Llama 3.1 8B", SupportsTools: true, SupportsStreaming: true},
		{ID: FormatModelID(ProviderGroq, "mixtral-8x7b-32768"), Name: "Mixtral 8x7B", SupportsTools: true, SupportsStreaming: true},
	},
	ProviderMistral: {
		{ID: FormatModelID(ProviderMistral, "mistral-large-latest"), Name: "Mistral Large", SupportsTools: true, SupportsStreaming: true},
		{ID: FormatModelID(ProviderMistral, "mistral-small-latest"), Name: "Mistral Small", SupportsTools: true, SupportsStreaming: true},
		{ID: FormatModelID(ProviderMistral, "codestral-latest"), Name: "Codestral", SupportsTools: true, SupportsStreaming: true},
	},
	ProviderTogetherAI: {
		{ID: FormatModelID(ProviderTogetherAI, "meta-llama/Llama-3.3-70B-Instruct-Turbo"), Name: "Llama 3.3 70B", SupportsTools: true, SupportsStreaming: true},
		{ID: FormatModelID(ProviderTogetherAI, "mistralai/Mixtral-8x7B-Instruct-v0.1"), Name: "Mixtral 8x7B", SupportsTools: true, SupportsStreaming: true},
	},
	ProviderPerplexity: {
		{ID: FormatModelID(ProviderPerplexity, "llama-3.1-sonar-large-128k-online"), Name: "Sonar Large Online", SupportsTools: false, SupportsStreaming: true},
		{ID: FormatModelID(ProviderPerplexity, "llama-3.1-sonar-small-128k-online"), Name: "Sonar Small Online", SupportsTools: false, SupportsStreaming: true},
	},
	ProviderXAI: {
		{ID: FormatModelID(ProviderXAI, "grok-beta"), Name: "Grok Beta", SupportsTools: true, SupportsStreaming: true},
	},
	ProviderCerebras: {
		{ID: FormatModelID(ProviderCerebras, "llama3.1-8b"), Name: "Llama 3.1 8B", SupportsTools: true, SupportsStreaming: true},
		{ID: FormatModelID(ProviderCerebras, "llama3.1-70b"), Name: "Llama 3.1 70B", SupportsTools: true, SupportsStreaming: true},
	},
	ProviderDeepInfra: {
		{ID: FormatModelID(ProviderDeepInfra, "meta-llama/Llama-3.3-70B-Instruct-Turbo"), Name: "Llama 3.3 70B", SupportsTools: true, SupportsStreaming: true},
		{ID: FormatModelID(ProviderDeepInfra, "mistralai/Mistral-Small-24B-Instruct-2501"), Name: "Mistral Small 24B", SupportsTools: true, SupportsStreaming: true},
	},
	// Alibaba/Bailian Qwen models
	ProviderAlibaba: {
		{ID: FormatModelID(ProviderAlibaba, "qwen-max"), Name: "Qwen Max", SupportsTools: true, SupportsStreaming: true, MaxInputTokens: 32768, MaxOutputTokens: 8192},
		{ID: FormatModelID(ProviderAlibaba, "qwen-plus"), Name: "Qwen Plus", SupportsTools: true, SupportsStreaming: true, MaxInputTokens: 131072, MaxOutputTokens: 8192},
		{ID: FormatModelID(ProviderAlibaba, "qwen-turbo"), Name: "Qwen Turbo", SupportsTools: true, SupportsStreaming: true, MaxInputTokens: 131072, MaxOutputTokens: 8192},
		{ID: FormatModelID(ProviderAlibaba, "qwen-coder-plus"), Name: "Qwen Coder Plus", SupportsTools: true, SupportsStreaming: true, MaxInputTokens: 131072, MaxOutputTokens: 8192},
	},
	ProviderAlibabaCN: {
		{ID: FormatModelID(ProviderAlibabaCN, "qwen-max"), Name: "Qwen Max (CN)", SupportsTools: true, SupportsStreaming: true, MaxInputTokens: 32768, MaxOutputTokens: 8192},
		{ID: FormatModelID(ProviderAlibabaCN, "qwen-plus"), Name: "Qwen Plus (CN)", SupportsTools: true, SupportsStreaming: true, MaxInputTokens: 131072, MaxOutputTokens: 8192},
		{ID: FormatModelID(ProviderAlibabaCN, "qwen-turbo"), Name: "Qwen Turbo (CN)", SupportsTools: true, SupportsStreaming: true, MaxInputTokens: 131072, MaxOutputTokens: 8192},
	},
}

// GetPopularModels returns popular models for a provider
func GetPopularModels(providerID ProviderID) map[ModelID]ModelInfo {
	models := PopularModels[providerID]
	if models == nil {
		return nil
	}

	result := make(map[ModelID]ModelInfo)
	for _, m := range models {
		result[m.ID] = m
	}
	return result
}
