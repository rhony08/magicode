// Package provider provides tests for provider package.
package provider

import (
	"context"
	"io"
	"testing"
)

// TestProviderIDConstants tests provider ID constants
func TestProviderIDConstants(t *testing.T) {
	ids := []ProviderID{
		ProviderAnthropic,
		ProviderOpenAI,
		ProviderGoogle,
		ProviderAzure,
		ProviderBedrock,
		ProviderOpenRouter,
		ProviderMistral,
		ProviderGroq,
	}

	expected := []string{
		"anthropic",
		"openai",
		"google",
		"azure",
		"amazon-bedrock",
		"openrouter",
		"mistral",
		"groq",
	}

	for i, id := range ids {
		if string(id) != expected[i] {
			t.Errorf("ProviderID %d: expected %s, got %s", i, expected[i], id)
		}
	}
}

// TestParseModelID tests parsing model IDs
func TestParseModelID(t *testing.T) {
	tests := []struct {
		modelID      ModelID
		expectedProv ProviderID
		expectedName string
	}{
		{"anthropic/claude-sonnet-4-5", ProviderAnthropic, "claude-sonnet-4-5"},
		{"openai/gpt-4o", ProviderOpenAI, "gpt-4o"},
		{"google/gemini-pro", ProviderGoogle, "gemini-pro"},
		{"simple", ProviderID("simple"), "simple"}, // no slash
	}

	for _, tt := range tests {
		prov, name := ParseModelID(tt.modelID)
		if prov != tt.expectedProv {
			t.Errorf("ParseModelID(%s): expected provider %s, got %s", tt.modelID, tt.expectedProv, prov)
		}
		if name != tt.expectedName {
			t.Errorf("ParseModelID(%s): expected name %s, got %s", tt.modelID, tt.expectedName, name)
		}
	}
}

// TestFormatModelID tests formatting model IDs
func TestFormatModelID(t *testing.T) {
	tests := []struct {
		provider ProviderID
		model    string
		expected ModelID
	}{
		{ProviderAnthropic, "claude-sonnet-4-5", "anthropic/claude-sonnet-4-5"},
		{ProviderOpenAI, "gpt-4o", "openai/gpt-4o"},
		{ProviderID("custom"), "model-v1", "custom/model-v1"},
	}

	for _, tt := range tests {
		result := FormatModelID(tt.provider, tt.model)
		if result != tt.expected {
			t.Errorf("FormatModelID(%s, %s): expected %s, got %s", tt.provider, tt.model, tt.expected, result)
		}
	}
}

// TestProviderRegistry tests provider registry
func TestProviderRegistry(t *testing.T) {
	registry := NewProviderRegistry()

	// Check initial state
	if len(registry.ListProviders()) != 0 {
		t.Error("New registry should have no providers")
	}
	if len(registry.ListModels()) != 0 {
		t.Error("New registry should have no models")
	}

	// Create mock provider
	mockProv := &mockProvider{
		id: ProviderAnthropic,
		info: ProviderInfo{
			ID:   ProviderAnthropic,
			Name: "Anthropic",
			Models: map[ModelID]ModelInfo{
				FormatModelID(ProviderAnthropic, "claude-sonnet-4"): {
					ID:   FormatModelID(ProviderAnthropic, "claude-sonnet-4"),
					Name: "Claude Sonnet 4",
				},
			},
		},
	}

	// Register provider
	registry.Register(mockProv)

	// Check provider registered
	providers := registry.ListProviders()
	if len(providers) != 1 {
		t.Errorf("Expected 1 provider, got %d", len(providers))
	}

	// Get provider
	p, ok := registry.Get(ProviderAnthropic)
	if !ok {
		t.Error("Provider should be registered")
	}
	if p.ID() != ProviderAnthropic {
		t.Errorf("Expected provider ID %s, got %s", ProviderAnthropic, p.ID())
	}

	// Check model registered
	models := registry.ListModels()
	if len(models) != 1 {
		t.Errorf("Expected 1 model, got %d", len(models))
	}

	// Get model
	m, ok := registry.GetModel(FormatModelID(ProviderAnthropic, "claude-sonnet-4"))
	if !ok {
		t.Error("Model should be registered")
	}
	if m.Name != "Claude Sonnet 4" {
		t.Errorf("Expected model name 'Claude Sonnet 4', got '%s'", m.Name)
	}

	// List models by provider
	anthropicModels := registry.ListModelsByProvider(ProviderAnthropic)
	if len(anthropicModels) != 1 {
		t.Errorf("Expected 1 Anthropic model, got %d", len(anthropicModels))
	}

	// Check non-existent provider models
	openaiModels := registry.ListModelsByProvider(ProviderOpenAI)
	if len(openaiModels) != 0 {
		t.Errorf("Expected 0 OpenAI models, got %d", len(openaiModels))
	}
}

// TestRoleConstants tests role constants
func TestRoleConstants(t *testing.T) {
	roles := []Role{RoleUser, RoleAssistant, RoleSystem}
	expected := []string{"user", "assistant", "system"}

	for i, role := range roles {
		if string(role) != expected[i] {
			t.Errorf("Role %d: expected %s, got %s", i, expected[i], role)
		}
	}
}

// TestContentPartTypes tests content part type methods
func TestContentPartTypes(t *testing.T) {
	textPart := TextPart{Type: "text", Text: "Hello"}
	if textPart.ContentType() != "text" {
		t.Errorf("TextPart.ContentType(): expected 'text', got '%s'", textPart.ContentType())
	}

	toolUsePart := ToolUsePart{Type: "tool_use", ID: "123", Name: "bash"}
	if toolUsePart.ContentType() != "tool_use" {
		t.Errorf("ToolUsePart.ContentType(): expected 'tool_use', got '%s'", toolUsePart.ContentType())
	}

	toolResultPart := ToolResultPart{Type: "tool_result", ToolUseID: "123", Content: "output"}
	if toolResultPart.ContentType() != "tool_result" {
		t.Errorf("ToolResultPart.ContentType(): expected 'tool_result', got '%s'", toolResultPart.ContentType())
	}
}

// TestStreamEventTypes tests stream event type methods
func TestStreamEventTypes(t *testing.T) {
	events := []StreamEvent{
		ContentBlockStartEvent{Type: "content_block_start"},
		ContentBlockDeltaEvent{Type: "content_block_delta"},
		ContentBlockStopEvent{Type: "content_block_stop"},
		MessageStartEvent{Type: "message_start"},
		MessageDeltaEvent{Type: "message_delta"},
		MessageStopEvent{Type: "message_stop"},
		PingEvent{Type: "ping"},
		ErrorEvent{Type: "error"},
	}

	expected := []string{
		"content_block_start",
		"content_block_delta",
		"content_block_stop",
		"message_start",
		"message_delta",
		"message_stop",
		"ping",
		"error",
	}

	for i, event := range events {
		if event.EventType() != expected[i] {
			t.Errorf("Event %d EventType(): expected '%s', got '%s'", i, expected[i], event.EventType())
		}
	}
}

// TestCostStruct tests cost structure
func TestCostStruct(t *testing.T) {
	cost := Cost{
		Input:  3.00,
		Output: 15.00,
		Cache:  0.30,
	}

	if cost.Input != 3.00 {
		t.Errorf("Expected input cost 3.00, got %f", cost.Input)
	}
	if cost.Output != 15.00 {
		t.Errorf("Expected output cost 15.00, got %f", cost.Output)
	}
	if cost.Cache != 0.30 {
		t.Errorf("Expected cache cost 0.30, got %f", cost.Cache)
	}
}

// TestModelInfoStruct tests model info structure
func TestModelInfoStruct(t *testing.T) {
	model := ModelInfo{
		ID:               FormatModelID(ProviderAnthropic, "claude-sonnet-4"),
		Name:             "Claude Sonnet 4",
		Description:      "Test model",
		MaxInputTokens:   200000,
		MaxOutputTokens:  8192,
		SupportsVision:   true,
		SupportsTools:    true,
		SupportsStreaming: true,
		Cost: Cost{
			Input:  3.00,
			Output: 15.00,
		},
	}

	if model.ID != "anthropic/claude-sonnet-4" {
		t.Errorf("Expected ID 'anthropic/claude-sonnet-4', got '%s'", model.ID)
	}
	if !model.SupportsVision {
		t.Error("Model should support vision")
	}
	if !model.SupportsTools {
		t.Error("Model should support tools")
	}
	if !model.SupportsStreaming {
		t.Error("Model should support streaming")
	}
}

// TestAnthropicModels tests Anthropic model definitions
func TestAnthropicModels(t *testing.T) {
	if len(AnthropicModels) == 0 {
		t.Error("AnthropicModels should not be empty")
	}

	// Check specific model
	modelID := FormatModelID(ProviderAnthropic, "claude-sonnet-4-5")
	model, ok := AnthropicModels[modelID]
	if !ok {
		t.Error("claude-sonnet-4-5 should be in AnthropicModels")
	}

	if model.Name != "Claude Sonnet 4.5" {
		t.Errorf("Expected name 'Claude Sonnet 4.5', got '%s'", model.Name)
	}
	if model.MaxInputTokens != 200000 {
		t.Errorf("Expected MaxInputTokens 200000, got %d", model.MaxInputTokens)
	}
}

// TestOpenAIModels tests OpenAI model definitions
func TestOpenAIModels(t *testing.T) {
	if len(OpenAIModels) == 0 {
		t.Error("OpenAIModels should not be empty")
	}

	// Check specific model
	modelID := FormatModelID(ProviderOpenAI, "gpt-4o")
	model, ok := OpenAIModels[modelID]
	if !ok {
		t.Error("gpt-4o should be in OpenAIModels")
	}

	if model.Name != "GPT-4o" {
		t.Errorf("Expected name 'GPT-4o', got '%s'", model.Name)
	}
	if model.MaxInputTokens != 128000 {
		t.Errorf("Expected MaxInputTokens 128000, got %d", model.MaxInputTokens)
	}
}

// Mock provider for testing
type mockProvider struct {
	id   ProviderID
	info ProviderInfo
}

func (m *mockProvider) ID() ProviderID {
	return m.id
}

func (m *mockProvider) Info() ProviderInfo {
	return m.info
}

func (m *mockProvider) Chat(ctx context.Context, req ChatRequest) (*ChatResponse, error) {
	return nil, nil
}

func (m *mockProvider) StreamChat(ctx context.Context, req ChatRequest) (<-chan StreamEvent, error) {
	return nil, nil
}

func (m *mockProvider) StreamChatRaw(ctx context.Context, req ChatRequest) (io.ReadCloser, error) {
	return nil, nil
}

func (m *mockProvider) ValidateKey(ctx context.Context) error {
	return nil
}

func (m *mockProvider) SetAPIKey(key string) {}

func (m *mockProvider) SetBaseURL(url string) {}

func (m *mockProvider) Close() error {
	return nil
}