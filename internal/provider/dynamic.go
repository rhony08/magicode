// Package provider provides dynamic provider registration from OpenCode config.
package provider

import (
	"fmt"
	"strings"

	"github.com/rhony08/magicode/internal/opencode"
)

// DynamicProvider wraps an OpenAI-compatible provider with custom configuration
type DynamicProvider struct {
	*OpenAICompatibleProvider
	customModels map[string]opencode.Model
}

// RegisterBundledProviders registers all bundled OpenAI-compatible providers
// This should be called at startup to enable OpenRouter, Groq, etc.
func RegisterBundledProviders(registry *ProviderRegistry) {
	bundledProviders := []ProviderID{
		ProviderOpenRouter,
		ProviderGroq,
		ProviderMistral,
		ProviderTogetherAI,
		ProviderPerplexity,
		ProviderXAI,
		ProviderCerebras,
		ProviderDeepInfra,
		ProviderAlibaba,
		ProviderAlibabaCN,
	}

	for _, providerID := range bundledProviders {
		// Register with empty API key initially
		// API key will be set when provider is used
		provider := NewOpenAICompatibleProvider(providerID, "")
		if provider != nil {
			// Add popular models
			popularModels := GetPopularModels(providerID)
			for modelID, modelInfo := range popularModels {
				provider.OpenAIProvider.models[modelID] = modelInfo
			}
			registry.Register(provider)
		}
	}
}

// RegisterFromConfig reads OpenCode config and registers all providers
// This includes both bundled providers (with models from config) and custom providers
func RegisterFromConfig(registry *ProviderRegistry, configReader *opencode.ConfigReader) error {
	providers, err := configReader.ReadProviders()
	if err != nil {
		return fmt.Errorf("failed to read providers from config: %w", err)
	}

	for _, p := range providers {
		providerID := ProviderID(p.ID)

		// Check if this is a bundled provider
		if isBundledProvider(providerID) {
			// Add models from config to existing provider
			if len(p.Models) > 0 {
				modelInfos := convertOpenCodeModels(p.Models, providerID)
				registry.AddModels(providerID, modelInfos)
			}
			continue
		}

		// Create dynamic provider for custom providers
		dp := NewDynamicProvider(p.ID, p.Name, p.BaseURL, p.APIKey, p.Models)
		if dp != nil {
			registry.Register(dp)
		}
	}

	return nil
}

// isBundledProvider checks if a provider is bundled (pre-registered)
func isBundledProvider(providerID ProviderID) bool {
	bundled := []ProviderID{
		ProviderAnthropic,
		ProviderOpenAI,
		ProviderOpenRouter,
		ProviderGroq,
		ProviderMistral,
		ProviderTogetherAI,
		ProviderPerplexity,
		ProviderXAI,
		ProviderCerebras,
		ProviderDeepInfra,
	}

	for _, id := range bundled {
		if id == providerID {
			return true
		}
	}
	return false
}

// NewDynamicProvider creates a provider from OpenCode config
func NewDynamicProvider(id, name, baseURL, apiKey string, models map[string]opencode.Model) *DynamicProvider {
	providerID := ProviderID(id)

	// Convert OpenCode models to provider ModelInfo
	modelInfos := convertOpenCodeModels(models, providerID)

	// Create custom OpenAI-compatible provider
	compatible := CustomOpenAICompatibleProvider(providerID, name, baseURL, apiKey, modelInfos)

	return &DynamicProvider{
		OpenAICompatibleProvider: compatible,
		customModels:             models,
	}
}

// convertOpenCodeModels converts OpenCode models to provider ModelInfo
func convertOpenCodeModels(models map[string]opencode.Model, providerID ProviderID) map[ModelID]ModelInfo {
	result := make(map[ModelID]ModelInfo)

	for _, model := range models {
		modelID := FormatModelID(providerID, model.ID)

		info := ModelInfo{
			ID:                modelID,
			Name:              model.Name,
			Description:       model.ID,
			SupportsTools:     true, // Default to true for OpenAI-compatible
			SupportsStreaming: true, // Default to true for OpenAI-compatible
		}

		// Set limits from config
		if model.Limit.Context > 0 {
			info.MaxInputTokens = model.Limit.Context
		}
		if model.Limit.Output > 0 {
			info.MaxOutputTokens = model.Limit.Output
		}

		// Check modalities for vision support
		for _, modality := range model.Modalities.Input {
			if modality == "image" {
				info.SupportsVision = true
			}
		}

		result[modelID] = info
	}

	return result
}

// GetEnvKeyForProvider returns the environment variable key for a provider
func GetEnvKeyForProvider(providerID ProviderID) string {
	config, ok := bundledProviderConfigs[providerID]
	if ok && len(config.envKeys) > 0 {
		return config.envKeys[0]
	}

	// Convert provider ID to env key format
	// e.g., "openrouter" -> "OPENROUTER_API_KEY"
	return strings.ToUpper(string(providerID)) + "_API_KEY"
}

// SetProviderAPIKey sets the API key for a provider in the registry
func SetProviderAPIKey(registry *ProviderRegistry, providerID ProviderID, apiKey string) error {
	prov, ok := registry.Get(providerID)
	if !ok {
		return fmt.Errorf("provider not found: %s", providerID)
	}

	switch p := prov.(type) {
	case *OpenAICompatibleProvider:
		p.SetAPIKey(apiKey)
	case *OpenAIProvider:
		p.SetAPIKey(apiKey)
	case *AnthropicProvider:
		p.SetAPIKey(apiKey)
	case *DynamicProvider:
		p.SetAPIKey(apiKey)
	default:
		return fmt.Errorf("unsupported provider type: %T", prov)
	}

	return nil
}
