package provider

import (
	"testing"
)

// BenchmarkRegistry benchmarks registry operations
func BenchmarkRegistry(b *testing.B) {
	registry := NewProviderRegistry()

	// Pre-register providers
	registry.Register(NewAnthropicProvider("test-key"))
	registry.Register(NewOpenAIProvider("test-key"))

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		registry.ListProviders()
		registry.ListModels()
	}
}

// BenchmarkRegistryGet benchmarks getting a provider
func BenchmarkRegistryGet(b *testing.B) {
	registry := NewProviderRegistry()
	registry.Register(NewAnthropicProvider("test-key"))

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		registry.Get(ProviderAnthropic)
	}
}

// BenchmarkRegistryGetModel benchmarks getting model info
func BenchmarkRegistryGetModel(b *testing.B) {
	registry := NewProviderRegistry()
	registry.Register(NewAnthropicProvider("test-key"))

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		registry.GetModel(ModelID("anthropic/claude-sonnet-4-5"))
	}
}

// BenchmarkParseModelID benchmarks parsing model ID
func BenchmarkParseModelID(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ParseModelID(ModelID("anthropic/claude-sonnet-4-5"))
	}
}

// BenchmarkFormatModelID benchmarks formatting model ID
func BenchmarkFormatModelID(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		FormatModelID(ProviderAnthropic, "claude-sonnet-4-5")
	}
}

// BenchmarkListModelsByProvider benchmarks listing models by provider
func BenchmarkListModelsByProvider(b *testing.B) {
	registry := NewProviderRegistry()
	registry.Register(NewAnthropicProvider("test-key"))
	registry.Register(NewOpenAIProvider("test-key"))

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		registry.ListModelsByProvider(ProviderAnthropic)
	}
}

// BenchmarkMemoryRegistry benchmarks memory usage for registry
func BenchmarkMemoryRegistry(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		registry := NewProviderRegistry()
		registry.Register(NewAnthropicProvider("test-key"))
		registry.Register(NewOpenAIProvider("test-key"))
		registry.ListProviders()
	}
}

// BenchmarkMessage benchmarks message creation
func BenchmarkMessage(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = Message{
			Role:    RoleUser,
			Content: "Test message content",
		}
	}
}

// BenchmarkChatRequest benchmarks chat request creation
func BenchmarkChatRequest(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = ChatRequest{
			Model: ModelID("anthropic/claude-sonnet-4-5"),
			Messages: []Message{
				{Role: RoleUser, Content: "Hello"},
			},
			MaxTokens: 4096,
		}
	}
}

// BenchmarkModelInfo benchmarks model info creation
func BenchmarkModelInfo(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = ModelInfo{
			ID:          ModelID("anthropic/claude-sonnet-4-5"),
			Name:        "Claude Sonnet 4.5",
			MaxTokens:   200000,
			MaxOutputTokens: 8192,
			SupportsTools: true,
			SupportsStreaming: true,
		}
	}
}

// BenchmarkProviderInfo benchmarks provider info creation
func BenchmarkProviderInfo(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = ProviderInfo{
			ID:          ProviderAnthropic,
			Name:        "Anthropic",
			BaseURL:     "https://api.anthropic.com",
			EnvKeys:     []string{"ANTHROPIC_API_KEY"},
			Models:      make(map[ModelID]ModelInfo),
		}
	}
}

// BenchmarkAnthropicProvider benchmarks creating Anthropic provider
func BenchmarkAnthropicProvider(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = NewAnthropicProvider("test-api-key")
	}
}

// BenchmarkOpenAIProvider benchmarks creating OpenAI provider
func BenchmarkOpenAIProvider(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = NewOpenAIProvider("test-api-key")
	}
}