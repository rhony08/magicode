// Package provider provides the provider interface.
package provider

import (
	"context"
	"io"
)

// Provider is the interface for AI providers
type Provider interface {
	// ID returns the provider's unique identifier
	ID() ProviderID
	
	// Info returns provider information
	Info() ProviderInfo
	
	// Chat sends a non-streaming chat request
	Chat(ctx context.Context, req ChatRequest) (*ChatResponse, error)
	
	// StreamChat sends a streaming chat request and returns a channel of events
	StreamChat(ctx context.Context, req ChatRequest) (<-chan StreamEvent, error)
	
	// StreamChatRaw sends a streaming request and returns the raw response body
	// for custom parsing
	StreamChatRaw(ctx context.Context, req ChatRequest) (io.ReadCloser, error)
	
	// ValidateKey checks if the API key is valid
	ValidateKey(ctx context.Context) error
	
	// SetAPIKey sets the API key for this provider
	SetAPIKey(key string)
	
	// SetBaseURL sets a custom base URL (for proxies)
	SetBaseURL(url string)
	
	// Close cleans up resources (connection pools, etc.)
	Close() error
}

// Streamer handles streaming responses
type Streamer interface {
	// Stream returns a channel that receives stream events
	Stream(ctx context.Context, req ChatRequest) (<-chan StreamEvent, <-chan error)
}

// ProviderRegistry manages available providers
type ProviderRegistry struct {
	providers map[ProviderID]Provider
	models    map[ModelID]ModelInfo
}

// NewProviderRegistry creates a new provider registry
func NewProviderRegistry() *ProviderRegistry {
	return &ProviderRegistry{
		providers: make(map[ProviderID]Provider),
		models:    make(map[ModelID]ModelInfo),
	}
}

// Register adds a provider to the registry
func (r *ProviderRegistry) Register(p Provider) {
	r.providers[p.ID()] = p
	
	// Register all models from this provider
	for modelID, modelInfo := range p.Info().Models {
		r.models[modelID] = modelInfo
	}
}

// Get retrieves a provider by ID
func (r *ProviderRegistry) Get(id ProviderID) (Provider, bool) {
	p, ok := r.providers[id]
	return p, ok
}

// GetModel retrieves model info by ID
func (r *ProviderRegistry) GetModel(id ModelID) (ModelInfo, bool) {
	m, ok := r.models[id]
	return m, ok
}

// ListProviders returns all registered providers
func (r *ProviderRegistry) ListProviders() []Provider {
	result := make([]Provider, 0, len(r.providers))
	for _, p := range r.providers {
		result = append(result, p)
	}
	return result
}

// ListModels returns all registered models
func (r *ProviderRegistry) ListModels() []ModelInfo {
	result := make([]ModelInfo, 0, len(r.models))
	for _, m := range r.models {
		result = append(result, m)
	}
	return result
}

// ListModelsByProvider returns models for a specific provider
func (r *ProviderRegistry) ListModelsByProvider(providerID ProviderID) []ModelInfo {
	result := []ModelInfo{}
	for modelID, modelInfo := range r.models {
		// ModelID format is provider/model-name
		if extractProvider(modelID) == providerID {
			result = append(result, modelInfo)
		}
	}
	return result
}

// extractProvider extracts the provider ID from a model ID
func extractProvider(modelID ModelID) ProviderID {
	id := string(modelID)
	for i := 0; i < len(id); i++ {
		if id[i] == '/' {
			return ProviderID(id[:i])
		}
	}
	return ProviderID(id)
}

// ParseModelID parses a model ID into provider and model name
func ParseModelID(modelID ModelID) (ProviderID, string) {
	id := string(modelID)
	for i := 0; i < len(id); i++ {
		if id[i] == '/' {
			return ProviderID(id[:i]), id[i+1:]
		}
	}
	return ProviderID(id), id
}

// FormatModelID formats a provider and model name into a model ID
func FormatModelID(provider ProviderID, model string) ModelID {
	return ModelID(string(provider) + "/" + model)
}