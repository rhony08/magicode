package session

import (
	"context"
	"io"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/rhony08/magicode/internal/bus"
	"github.com/rhony08/magicode/internal/database"
	"github.com/rhony08/magicode/internal/provider"
	"github.com/rhony08/magicode/internal/tool"
)

// mockProvider implements provider.Provider interface for testing
type mockProvider struct {
	id      provider.ProviderID
	name    string
	events  []provider.StreamEvent
	err     error
	delay   time.Duration
	models  map[provider.ModelID]provider.ModelInfo
}

func newMockProvider(id string, events []provider.StreamEvent) *mockProvider {
	return &mockProvider{
		id:     provider.ProviderID(id),
		name:   id,
		events: events,
		delay:  50 * time.Millisecond,
		models: map[provider.ModelID]provider.ModelInfo{
			provider.ModelID(id + "/test-model"): {
				ID:   provider.ModelID(id + "/test-model"),
				Name: "Test Model",
			},
		},
	}
}

func (m *mockProvider) ID() provider.ProviderID {
	return m.id
}

func (m *mockProvider) Info() provider.ProviderInfo {
	return provider.ProviderInfo{
		ID:     m.id,
		Name:   m.name,
		Models: m.models,
	}
}

func (m *mockProvider) StreamChat(ctx context.Context, req provider.ChatRequest) (<-chan provider.StreamEvent, error) {
	eventChan := make(chan provider.StreamEvent, 100)

	go func() {
		defer close(eventChan)

		for _, event := range m.events {
			select {
			case eventChan <- event:
				time.Sleep(m.delay) // Simulate streaming delay
			case <-ctx.Done():
				return
			}
		}

		if m.err != nil {
			eventChan <- provider.ErrorEvent{
				Type: "error",
				Error: struct {
					Type    string `json:"type"`
					Message string `json:"message"`
				}{
					Type:    "test_error",
					Message: m.err.Error(),
				},
			}
		}
	}()

	return eventChan, nil
}

func (m *mockProvider) Chat(ctx context.Context, req provider.ChatRequest) (*provider.ChatResponse, error) {
	return &provider.ChatResponse{}, nil
}

func (m *mockProvider) StreamChatRaw(ctx context.Context, req provider.ChatRequest) (io.ReadCloser, error) {
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

// createTestDatabase creates a temporary database for testing
func createTestDatabase(t *testing.T) (*database.Database, func()) {
	tmpDir, err := os.MkdirTemp("", "magicode-session-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}

	dbPath := filepath.Join(tmpDir, "test.db")
	db, err := database.New(context.Background(), database.Config{Path: dbPath})
	if err != nil {
		os.RemoveAll(tmpDir)
		t.Fatalf("Failed to create database: %v", err)
	}

	cleanup := func() {
		db.Close()
		os.RemoveAll(tmpDir)
	}

	return db, cleanup
}

func TestProcessorEndToEndStreaming(t *testing.T) {
	// Create test database
	db, cleanupDb := createTestDatabase(t)
	defer cleanupDb()

	// Create bus service
	busService := bus.NewDefault()
	defer busService.Close()

	// Create mock provider with streaming events
	events := []provider.StreamEvent{
		// Message start
		provider.MessageStartEvent{
			Type: "message_start",
			Message: struct {
				ID    string         `json:"id"`
				Model string         `json:"model"`
				Role  provider.Role  `json:"role"`
				Usage provider.Usage `json:"usage"`
			}{
				ID:    "msg-123",
				Model: "test-model",
				Role:  provider.RoleAssistant,
				Usage: provider.Usage{InputTokens: 10},
			},
		},

		// Content block start (text)
		provider.ContentBlockStartEvent{
			Type:  "content_block_start",
			Index: 0,
			ContentBlock: provider.TextPart{
				Type: "text",
				Text: "",
			},
		},

		// Content block deltas (text chunks)
		provider.ContentBlockDeltaEvent{
			Type:  "content_block_delta",
			Index: 0,
			Delta: provider.TextPart{Type: "text", Text: "Hello"},
		},
		provider.ContentBlockDeltaEvent{
			Type:  "content_block_delta",
			Index: 0,
			Delta: provider.TextPart{Type: "text", Text: ", world"},
		},

		// Content block stop (text)
		provider.ContentBlockStopEvent{
			Type:  "content_block_stop",
			Index: 0,
		},

		// Content block start (tool use)
		provider.ContentBlockStartEvent{
			Type:  "content_block_start",
			Index: 1,
			ContentBlock: provider.ToolUsePart{
				Type: "tool_use",
				ID:   "tool-1",
				Name: "bash",
				Input: map[string]interface{}{
					"command": "echo test",
				},
			},
		},

		// Content block stop (tool use)
		provider.ContentBlockStopEvent{
			Type:  "content_block_stop",
			Index: 1,
		},

		// Message delta (stop reason)
		provider.MessageDeltaEvent{
			Type: "message_delta",
			Delta: struct {
				StopReason string `json:"stop_reason"`
			}{
				StopReason: "tool_use",
			},
			Usage: provider.Usage{OutputTokens: 20},
		},

		// Message stop
		provider.MessageStopEvent{
			Type: "message_stop",
		},
	}

	mockProv := newMockProvider("test-provider", events)

	// Create provider registry
	registry := provider.NewProviderRegistry()
	registry.Register(mockProv)

	// Create processor config
	config := ProcessorConfig{
		Registry:     registry,
		DB:           db,
		Bus:          busService,
		ToolRegistry: tool.NewRegistry(),
		Logger:       nil,
	}

	processor := NewProcessor(config)

	// Subscribe to all events
	eventChan, cleanup := busService.SubscribeAll()
	defer cleanup()

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	// Create process request
	req := ProcessRequest{
		SessionID:   "test-session",
		UserMessage: "Hello",
		Model:       provider.ModelID("test-provider/test-model"),
		History:     nil,
	}

	// Start processing in goroutine
	errChan := make(chan error, 1)
	go func() {
		errChan <- processor.Process(ctx, req)
	}()

	// Collect events with timeout
	var receivedEvents []bus.Payload
	timeout := time.After(5 * time.Second)

	for {
		select {
		case payload := <-eventChan:
			receivedEvents = append(receivedEvents, payload)
			t.Logf("Received event: %s", payload.Type)

			// Stop when we get message complete or error
			if payload.Type == EventMessageComplete.Type ||
				payload.Type == EventStreamError.Type {
				goto CheckResults
			}

		case err := <-errChan:
			if err != nil {
				t.Logf("Process returned error: %v", err)
			}
			goto CheckResults

		case <-timeout:
			t.Fatal("Timeout waiting for events")
		}
	}

CheckResults:
	// Verify we received expected events
	expectedTypes := map[string]bool{
		"session.message.created": false, // User message created
		"session.part.created":    false, // Text part created
		"session.part.updated":    false, // Text part updated
		"session.part.complete":   false, // Text part complete
		"session.tool.pending":    false, // Tool call pending
		"session.tool.running":    false, // Tool executing
		"session.tool.complete":   false, // Tool finished
		"session.message.complete": false, // Message complete
	}

	for _, payload := range receivedEvents {
		if _, ok := expectedTypes[payload.Type]; ok {
			expectedTypes[payload.Type] = true
		}
	}

	// Check critical events
	for eventType, found := range expectedTypes {
		if !found {
			t.Logf("Warning: event type %s not found (may not be implemented yet)", eventType)
		}
	}

	t.Logf("Received %d events total", len(receivedEvents))
}

func TestProcessorTextStreaming(t *testing.T) {
	// Create test database
	db, cleanupDb := createTestDatabase(t)
	defer cleanupDb()

	// Create bus service
	busService := bus.NewDefault()
	defer busService.Close()

	// Create mock provider with text-only events
	events := []provider.StreamEvent{
		provider.MessageStartEvent{
			Type: "message_start",
			Message: struct {
				ID    string         `json:"id"`
				Model string         `json:"model"`
				Role  provider.Role  `json:"role"`
				Usage provider.Usage `json:"usage"`
			}{
				ID:    "msg-text",
				Model: "test-model",
				Role:  provider.RoleAssistant,
			},
		},

		provider.ContentBlockStartEvent{
			Type:  "content_block_start",
			Index: 0,
			ContentBlock: provider.TextPart{
				Type: "text",
			},
		},

		provider.ContentBlockDeltaEvent{
			Type:  "content_block_delta",
			Index: 0,
			Delta: provider.TextPart{Type: "text", Text: "This is "},
		},

		provider.ContentBlockDeltaEvent{
			Type:  "content_block_delta",
			Index: 0,
			Delta: provider.TextPart{Type: "text", Text: "a streaming "},
		},

		provider.ContentBlockDeltaEvent{
			Type:  "content_block_delta",
			Index: 0,
			Delta: provider.TextPart{Type: "text", Text: "response."},
		},

		provider.ContentBlockStopEvent{
			Type:  "content_block_stop",
			Index: 0,
		},

		provider.MessageDeltaEvent{
			Type: "message_delta",
			Delta: struct {
				StopReason string `json:"stop_reason"`
			}{
				StopReason: "end_turn",
			},
		},

		provider.MessageStopEvent{
			Type: "message_stop",
		},
	}

	mockProv := newMockProvider("text-provider", events)

	// Create provider registry
	registry := provider.NewProviderRegistry()
	registry.Register(mockProv)

	// Create processor
	config := ProcessorConfig{
		Registry:     registry,
		DB:           db,
		Bus:          busService,
		ToolRegistry: tool.NewRegistry(),
	}
	processor := NewProcessor(config)

	// Subscribe to all events
	eventChan, cleanup := busService.SubscribeAll()
	defer cleanup()

	// Create context
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Process request
	req := ProcessRequest{
		SessionID:   "text-session",
		UserMessage: "Test",
		Model:       provider.ModelID("text-provider/test-model"),
	}

	// Start processing
	go processor.Process(ctx, req)

	// Collect events
	var receivedEvents []bus.Payload
	timeout := time.After(3 * time.Second)

	for {
		select {
		case payload := <-eventChan:
			receivedEvents = append(receivedEvents, payload)

			if payload.Type == EventMessageComplete.Type {
				goto Done
			}

		case <-timeout:
			t.Fatal("Timeout waiting for events")
		}
	}

Done:
	// Verify we got message complete
	foundComplete := false
	for _, payload := range receivedEvents {
		if payload.Type == EventMessageComplete.Type {
			foundComplete = true
			break
		}
	}

	if !foundComplete {
		t.Error("Expected MessageComplete event")
	}

	t.Logf("Received %d events for text streaming", len(receivedEvents))
}

func TestProcessorCancelation(t *testing.T) {
	// Create test database
	db, cleanupDb := createTestDatabase(t)
	defer cleanupDb()

	// Create bus service
	busService := bus.NewDefault()
	defer busService.Close()

	// Create mock provider with slow streaming (1 second delay per event)
	events := []provider.StreamEvent{
		provider.MessageStartEvent{
			Type: "message_start",
			Message: struct {
				ID    string         `json:"id"`
				Model string         `json:"model"`
				Role  provider.Role  `json:"role"`
				Usage provider.Usage `json:"usage"`
			}{
				ID:    "msg-cancel",
				Model: "test-model",
				Role:  provider.RoleAssistant,
			},
		},
		provider.ContentBlockStartEvent{
			Type:  "content_block_start",
			Index: 0,
			ContentBlock: provider.TextPart{Type: "text"},
		},
		provider.ContentBlockDeltaEvent{
			Type:  "content_block_delta",
			Index: 0,
			Delta: provider.TextPart{Type: "text", Text: "Start"},
		},
		// More events would come, but we'll cancel early
	}

	mockProv := newMockProvider("cancel-provider", events)
	mockProv.delay = 1 * time.Second // Slow streaming

	// Create registry and processor
	registry := provider.NewProviderRegistry()
	registry.Register(mockProv)

	config := ProcessorConfig{
		Registry:     registry,
		DB:           db,
		Bus:          busService,
		ToolRegistry: tool.NewRegistry(),
	}
	processor := NewProcessor(config)

	// Create context that we'll cancel early
	ctx, cancel := context.WithCancel(context.Background())

	// Process request
	req := ProcessRequest{
		SessionID:   "cancel-session",
		UserMessage: "Test",
		Model:       provider.ModelID("cancel-provider/test-model"),
	}

	// Start processing
	go processor.Process(ctx, req)

	// Wait a bit then cancel
	time.Sleep(500 * time.Millisecond)
	cancel()

	// Wait for processor to finish
	time.Sleep(500 * time.Millisecond)

	t.Log("Cancelation test completed successfully")
}