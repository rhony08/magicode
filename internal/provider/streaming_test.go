// Package provider provides streaming tests for Anthropic and OpenAI providers.
package provider

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// TestAnthropicStreaming tests Anthropic SSE streaming
func TestAnthropicStreaming(t *testing.T) {
	// Create mock Anthropic SSE response
	sseResponse := `data: {"type":"message_start","message":{"id":"msg_123","model":"claude-sonnet-4-5","role":"assistant","usage":{"input_tokens":10,"output_tokens":0}}}

data: {"type":"content_block_start","index":0,"content_block":{"type":"text","text":""}}

data: {"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":"Hello"}}

data: {"type":"content_block_delta","index":0,"delta":{"type":"text_delta","text":" world"}}

data: {"type":"content_block_stop","index":0}

data: {"type":"message_delta","delta":{"stop_reason":"end_turn"},"usage":{"output_tokens":2}}

data: {"type":"message_stop"}

`

	// Create mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request
		if r.Method != "POST" {
			t.Errorf("Expected POST request, got %s", r.Method)
		}
		if r.URL.Path != "/messages" {
			t.Errorf("Expected /messages path, got %s", r.URL.Path)
		}

		// Verify headers
		if r.Header.Get("x-api-key") == "" {
			t.Error("Missing x-api-key header")
		}
		if r.Header.Get("anthropic-version") == "" {
			t.Error("Missing anthropic-version header")
		}

		// Verify streaming enabled in request body
		body, _ := io.ReadAll(r.Body)
		var req map[string]interface{}
		json.Unmarshal(body, &req)
		if stream, ok := req["stream"].(bool); !ok || !stream {
			t.Error("Request should have stream: true")
		}

		// Send SSE response
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.WriteHeader(200)

		// Write SSE events
		for _, line := range strings.Split(sseResponse, "\n\n") {
			if line != "" {
				w.Write([]byte(line + "\n\n"))
			}
		}
	}))
	defer server.Close()

	// Create Anthropic provider with mock server URL
	provider := NewAnthropicProvider("test-api-key")
	provider.SetBaseURL(server.URL)

	// Create chat request
	req := ChatRequest{
		Model:     FormatModelID(ProviderAnthropic, "claude-sonnet-4-5"),
		Messages:  []Message{{Role: RoleUser, Content: "Hello"}},
		MaxTokens: 100,
	}

	// Start streaming
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	events, err := provider.StreamChat(ctx, req)
	if err != nil {
		t.Fatalf("StreamChat failed: %v", err)
	}

	// Collect events
	var collectedEvents []StreamEvent
	for event := range events {
		collectedEvents = append(collectedEvents, event)
	}

	// Verify events
	if len(collectedEvents) < 5 {
		t.Errorf("Expected at least 5 events, got %d", len(collectedEvents))
	}

	// Check message_start event
	var foundMessageStart bool
	var foundContentDelta bool
	var foundMessageStop bool
	var textContent string

	for _, event := range collectedEvents {
		switch e := event.(type) {
		case MessageStartEvent:
			foundMessageStart = true
			if e.Message.ID != "msg_123" {
				t.Errorf("Expected message ID 'msg_123', got '%s'", e.Message.ID)
			}
		case ContentBlockDeltaEvent:
			foundContentDelta = true
			if delta, ok := e.Delta.(TextPart); ok {
				textContent += delta.Text
			}
		case MessageStopEvent:
			foundMessageStop = true
		}
	}

	if !foundMessageStart {
		t.Error("Missing message_start event")
	}
	if !foundContentDelta {
		t.Error("Missing content_block_delta event")
	}
	if !foundMessageStop {
		t.Error("Missing message_stop event")
	}
	if textContent != "Hello world" {
		t.Errorf("Expected text 'Hello world', got '%s'", textContent)
	}
}

// TestAnthropicToolUseStreaming tests Anthropic tool_use streaming
func TestAnthropicToolUseStreaming(t *testing.T) {
	// Create mock Anthropic SSE response with tool_use
	sseResponse := `data: {"type":"message_start","message":{"id":"msg_456","model":"claude-sonnet-4-5","role":"assistant","usage":{"input_tokens":10,"output_tokens":0}}}

data: {"type":"content_block_start","index":0,"content_block":{"type":"tool_use","id":"toolu_123","name":"bash"}}

data: {"type":"content_block_delta","index":0,"delta":{"type":"input_json_delta","partial_json":"{\"command\":\"ls\"}"}}

data: {"type":"content_block_stop","index":0}

data: {"type":"message_delta","delta":{"stop_reason":"tool_use"},"usage":{"output_tokens":5}}

data: {"type":"message_stop"}

`

	// Create mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(200)
		w.Write([]byte(sseResponse))
	}))
	defer server.Close()

	provider := NewAnthropicProvider("test-api-key")
	provider.SetBaseURL(server.URL)

	req := ChatRequest{
		Model:    FormatModelID(ProviderAnthropic, "claude-sonnet-4-5"),
		Messages: []Message{{Role: RoleUser, Content: "List files"}},
		Tools: []ToolDefinition{
			{Name: "bash", Description: "Run bash command", InputSchema: map[string]interface{}{"type": "object"}},
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	events, err := provider.StreamChat(ctx, req)
	if err != nil {
		t.Fatalf("StreamChat failed: %v", err)
	}

	// Check for tool_use events
	var foundToolUseStart bool
	for event := range events {
		switch e := event.(type) {
		case ContentBlockStartEvent:
			if toolUse, ok := e.ContentBlock.(ToolUsePart); ok {
				foundToolUseStart = true
				if toolUse.Name != "bash" {
					t.Errorf("Expected tool name 'bash', got '%s'", toolUse.Name)
				}
				if toolUse.ID != "toolu_123" {
					t.Errorf("Expected tool ID 'toolu_123', got '%s'", toolUse.ID)
				}
			}
		}
	}

	if !foundToolUseStart {
		t.Error("Missing tool_use content_block_start event")
	}
}

// TestOpenAIStreaming tests OpenAI SSE streaming
func TestOpenAIStreaming(t *testing.T) {
	// Create mock OpenAI SSE response
	sseResponse := `data: {"id":"chatcmpl-123","object":"chat.completion.chunk","model":"gpt-4o","choices":[{"index":0,"delta":{"role":"assistant","content":""},"finish_reason":null}]}

data: {"id":"chatcmpl-123","object":"chat.completion.chunk","model":"gpt-4o","choices":[{"index":0,"delta":{"content":"Hello"},"finish_reason":null}]}

data: {"id":"chatcmpl-123","object":"chat.completion.chunk","model":"gpt-4o","choices":[{"index":0,"delta":{"content":" there"},"finish_reason":null}]}

data: {"id":"chatcmpl-123","object":"chat.completion.chunk","model":"gpt-4o","choices":[{"index":0,"delta":{},"finish_reason":"stop"}]}

data: [DONE]

`

	// Create mock server
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Verify request
		if r.Method != "POST" {
			t.Errorf("Expected POST request, got %s", r.Method)
		}
		if r.URL.Path != "/chat/completions" {
			t.Errorf("Expected /chat/completions path, got %s", r.URL.Path)
		}

		// Verify Authorization header
		if r.Header.Get("Authorization") == "" {
			t.Error("Missing Authorization header")
		}

		// Send SSE response
		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.WriteHeader(200)
		w.Write([]byte(sseResponse))
	}))
	defer server.Close()

	// Create OpenAI provider with mock server URL
	provider := NewOpenAIProvider("test-api-key")
	provider.SetBaseURL(server.URL)

	// Create chat request
	req := ChatRequest{
		Model:    FormatModelID(ProviderOpenAI, "gpt-4o"),
		Messages: []Message{{Role: RoleUser, Content: "Hello"}},
	}

	// Start streaming
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	events, err := provider.StreamChat(ctx, req)
	if err != nil {
		t.Fatalf("StreamChat failed: %v", err)
	}

	// Collect events and verify
	var foundContentDelta bool
	var foundMessageStop bool
	var textContent string

	for event := range events {
		switch e := event.(type) {
		case ContentBlockDeltaEvent:
			foundContentDelta = true
			if delta, ok := e.Delta.(TextPart); ok {
				textContent += delta.Text
			}
		case MessageStopEvent:
			foundMessageStop = true
		}
	}

	if !foundContentDelta {
		t.Error("Missing content_block_delta event")
	}
	if !foundMessageStop {
		t.Error("Missing message_stop event")
	}
	if textContent != "Hello there" {
		t.Errorf("Expected text 'Hello there', got '%s'", textContent)
	}
}

// TestOpenAIToolCallStreaming tests OpenAI tool_calls streaming
func TestOpenAIToolCallStreaming(t *testing.T) {
	// Create mock OpenAI SSE response with tool_calls
	sseResponse := `data: {"id":"chatcmpl-456","object":"chat.completion.chunk","model":"gpt-4o","choices":[{"index":0,"delta":{"role":"assistant","content":null,"tool_calls":[{"index":0,"id":"call_123","type":"function","function":{"name":"bash","arguments":""}}]},"finish_reason":null}]}

data: {"id":"chatcmpl-456","object":"chat.completion.chunk","model":"gpt-4o","choices":[{"index":0,"delta":{"tool_calls":[{"index":0,"function":{"arguments":"{\"command\":\"ls\"}"}}]},"finish_reason":null}]}

data: {"id":"chatcmpl-456","object":"chat.completion.chunk","model":"gpt-4o","choices":[{"index":0,"delta":{},"finish_reason":"tool_calls"}]}

data: [DONE]

`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(200)
		w.Write([]byte(sseResponse))
	}))
	defer server.Close()

	provider := NewOpenAIProvider("test-api-key")
	provider.SetBaseURL(server.URL)

	req := ChatRequest{
		Model:    FormatModelID(ProviderOpenAI, "gpt-4o"),
		Messages: []Message{{Role: RoleUser, Content: "List files"}},
		Tools: []ToolDefinition{
			{Name: "bash", Description: "Run bash", InputSchema: map[string]interface{}{"type": "object"}},
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	events, err := provider.StreamChat(ctx, req)
	if err != nil {
		t.Fatalf("StreamChat failed: %v", err)
	}

	// Check for tool_use events
	var foundToolUse bool
	for event := range events {
		switch e := event.(type) {
		case ContentBlockStartEvent:
			if toolUse, ok := e.ContentBlock.(ToolUsePart); ok {
				foundToolUse = true
				if toolUse.Name != "bash" {
					t.Errorf("Expected tool name 'bash', got '%s'", toolUse.Name)
				}
			}
		}
	}

	if !foundToolUse {
		t.Error("Missing tool_use content_block_start event")
	}
}

// TestAnthropicErrorStreaming tests error handling
func TestAnthropicErrorStreaming(t *testing.T) {
	// Create mock server that returns error
	sseResponse := `data: {"type":"error","error":{"type":"overloaded_error","message":"Overloaded"}}

`

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(200)
		w.Write([]byte(sseResponse))
	}))
	defer server.Close()

	provider := NewAnthropicProvider("test-api-key")
	provider.SetBaseURL(server.URL)

	req := ChatRequest{
		Model:    FormatModelID(ProviderAnthropic, "claude-sonnet-4-5"),
		Messages: []Message{{Role: RoleUser, Content: "Hello"}},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	events, err := provider.StreamChat(ctx, req)
	if err != nil {
		t.Fatalf("StreamChat failed: %v", err)
	}

	// Check for error event
	var foundError bool
	for event := range events {
		switch e := event.(type) {
		case ErrorEvent:
			foundError = true
			if e.Error.Type != "overloaded_error" {
				t.Errorf("Expected error type 'overloaded_error', got '%s'", e.Error.Type)
			}
		}
	}

	if !foundError {
		t.Error("Missing error event")
	}
}
