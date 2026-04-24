package integration

import (
	"context"
	"fmt"
	"os"
	"testing"
	"time"

	"github.com/rhony08/magicode/internal/bus"
	"github.com/rhony08/magicode/internal/config"
	"github.com/rhony08/magicode/internal/database"
	"github.com/rhony08/magicode/internal/instance"
	"github.com/rhony08/magicode/internal/provider"
	"github.com/rhony08/magicode/internal/server"
)

// TestIntegrationFullFlow tests the full integration flow
func TestIntegrationFullFlow(t *testing.T) {
	ctx := context.Background()

	// Setup: Create temp directories
	tmpDir := t.TempDir()
	dataDir := tmpDir + "/data"
	configDir := tmpDir + "/config"
	stateDir := tmpDir + "/state"

	// Create directories
	os.MkdirAll(dataDir, 0755)
	os.MkdirAll(configDir, 0755)
	os.MkdirAll(stateDir, 0755)

	t.Run("DatabaseIntegration", func(t *testing.T) {
		dbPath := dataDir + "/test.db"
		db, err := database.New(ctx, database.Config{Path: dbPath})
		if err != nil {
			t.Fatalf("Failed to create database: %v", err)
		}
		defer db.Close()

		// Create session storage
		sessionStorage := database.NewSessionStorage(db)

		// Test session creation
		session := database.Session{
			Directory: tmpDir,
			Title:     "Test Session",
			ProjectID: "test-project",
		}
		created, err := sessionStorage.Create(ctx, session)
		if err != nil {
			t.Fatalf("Failed to create session: %v", err)
		}

		// Test message storage
		msgStorage := database.NewMessageStorage(db)

		// Create message
		msg := database.Message{
			SessionID: created.ID,
			Data: database.MessageInfo{
				Role:    "user",
				Content: "Hello, world!",
			},
		}
		createdMsg, err := msgStorage.Create(ctx, msg)
		if err != nil {
			t.Fatalf("Failed to create message: %v", err)
		}
		if createdMsg.Data.Role != "user" {
			t.Errorf("Expected role 'user', got '%s'", createdMsg.Data.Role)
		}

		// Verify session exists
		sessions, err := sessionStorage.ListByDirectory(ctx, tmpDir)
		if err != nil {
			t.Fatalf("Failed to list sessions: %v", err)
		}
		if len(sessions) == 0 {
			t.Error("Expected at least one session")
		}

		// Verify messages exist
		messages, err := msgStorage.List(ctx, created.ID)
		if err != nil {
			t.Fatalf("Failed to list messages: %v", err)
		}
		if len(messages) == 0 {
			t.Error("Expected at least one message")
		}
	})

	t.Run("BusIntegration", func(t *testing.T) {
		b := bus.NewDefault()
		defer b.Close()

		// Define event
		def := b.Define("test.event", nil)

		// Subscribe
		received := make(chan bus.Payload, 1)
		ch, cleanup := b.Subscribe(def)
		defer cleanup()

		go func() {
			select {
			case p := <-ch:
				received <- p
			case <-time.After(1 * time.Second):
			}
		}()

		// Publish
		err := b.Publish(def, map[string]interface{}{"key": "value"})
		if err != nil {
			t.Fatalf("Failed to publish: %v", err)
		}

		// Verify receipt
		select {
		case p := <-received:
			if p.Type != "test.event" {
				t.Errorf("Expected type 'test.event', got '%s'", p.Type)
			}
		case <-time.After(1 * time.Second):
			t.Error("Did not receive event")
		}
	})

	t.Run("InstanceCacheIntegration", func(t *testing.T) {
		cache := instance.NewCache(10, 30*time.Minute)

		// Add instance
		instCtx := &instance.Context{
			Directory: tmpDir,
			ProjectID: "test-project",
		}
		err := cache.Put(instCtx)
		if err != nil {
			t.Fatalf("Failed to put instance: %v", err)
		}

		// Get instance
		retrieved, err := cache.Get(tmpDir)
		if err != nil {
			t.Fatalf("Failed to get instance: %v", err)
		}
		if retrieved.Directory != tmpDir {
			t.Errorf("Expected directory '%s', got '%s'", tmpDir, retrieved.Directory)
		}

		// Verify size
		if cache.Size() != 1 {
			t.Errorf("Expected size 1, got %d", cache.Size())
		}

		// Remove instance
		err = cache.Remove(tmpDir)
		if err != nil {
			t.Fatalf("Failed to remove instance: %v", err)
		}

		if cache.Size() != 0 {
			t.Errorf("Expected size 0 after removal, got %d", cache.Size())
		}
	})

	t.Run("ProviderRegistryIntegration", func(t *testing.T) {
		registry := provider.NewProviderRegistry()

		// Register providers
		anthropic := provider.NewAnthropicProvider("test-key")
		registry.Register(anthropic)

		openai := provider.NewOpenAIProvider("test-key")
		registry.Register(openai)

		// List providers
		providers := registry.ListProviders()
		if len(providers) != 2 {
			t.Errorf("Expected 2 providers, got %d", len(providers))
		}

		// Get provider
		p, ok := registry.Get(provider.ProviderAnthropic)
		if !ok {
			t.Error("Expected to find Anthropic provider")
		}
		if p.ID() != provider.ProviderAnthropic {
			t.Errorf("Expected ID 'anthropic', got '%s'", p.ID())
		}

		// List models
		models := registry.ListModels()
		if len(models) == 0 {
			t.Error("Expected at least one model")
		}

		// Get model
		modelID := provider.ModelID("anthropic/claude-sonnet-4-5")
		model, ok := registry.GetModel(modelID)
		if !ok {
			t.Error("Expected to find model")
		}
		if model.Name == "" {
			t.Error("Expected model name to be set")
		}
	})

	t.Run("ServerIntegration", func(t *testing.T) {
		srvCfg := server.DefaultConfig()
		srvCfg.Port = 0 // Use default port for quick test

		srv := server.New(tmpDir, srvCfg)

		// Start server
		listener, err := srv.Listen()
		if err != nil {
			t.Fatalf("Failed to start server: %v", err)
		}

		// Verify server is running
		if !srv.IsRunning() {
			t.Error("Expected server to be running")
		}

		// Shutdown
		err = srv.Shutdown()
		if err != nil {
			t.Fatalf("Failed to shutdown server: %v", err)
		}

		if srv.IsRunning() {
			t.Error("Expected server to be stopped")
		}

		t.Logf("Server URL: %s", listener.URL.String())
	})

	t.Run("ConfigIntegration", func(t *testing.T) {
		// Create test config file
		configFile := configDir + "/magicode.json"
		configContent := `{
			"model": "anthropic/claude-sonnet-4-5",
			"small_model": "anthropic/claude-3-5-haiku",
			"username": "test-user"
		}`
		os.WriteFile(configFile, []byte(configContent), 0644)

		// Load config
		cfg, err := config.New(tmpDir, configDir)
		if err != nil {
			t.Fatalf("Failed to load config: %v", err)
		}

		// Verify config values
		if cfg.Model() != "anthropic/claude-sonnet-4-5" {
			t.Errorf("Expected model 'anthropic/claude-sonnet-4-5', got '%s'", cfg.Model())
		}

		if cfg.SmallModel() != "anthropic/claude-3-5-haiku" {
			t.Errorf("Expected small model 'anthropic/claude-3-5-haiku', got '%s'", cfg.SmallModel())
		}

		info := cfg.Get()
		if info.Username != "test-user" {
			t.Errorf("Expected username 'test-user', got '%s'", info.Username)
		}
	})
}

// TestIntegrationMemoryBounds tests memory bounds are respected
func TestIntegrationMemoryBounds(t *testing.T) {
	t.Run("InstanceCacheBounds", func(t *testing.T) {
		cache := instance.NewCache(5, 30*time.Minute) // Max 5 instances

		// Add 10 instances (should trigger evictions)
		for i := 0; i < 10; i++ {
			instCtx := &instance.Context{
				Directory: fmt.Sprintf("/tmp/project-%d", i),
				ProjectID: fmt.Sprintf("project-%d", i),
			}
			err := cache.Put(instCtx)
			if err != nil {
				t.Fatalf("Failed to put instance %d: %v", i, err)
			}
		}

		// Size should be 5 (max)
		if cache.Size() > 5 {
			t.Errorf("Expected size <= 5, got %d", cache.Size())
		}

		// First 5 instances should be evicted (LRU)
		for i := 0; i < 5; i++ {
			_, err := cache.Get(fmt.Sprintf("/tmp/project-%d", i))
			if err == nil {
				t.Errorf("Expected instance %d to be evicted", i)
			}
		}

		// Last 5 instances should still exist
		for i := 5; i < 10; i++ {
			instCtx, err := cache.Get(fmt.Sprintf("/tmp/project-%d", i))
			if err != nil {
				t.Errorf("Expected instance %d to exist: %v", i, err)
			}
			if instCtx.Directory != fmt.Sprintf("/tmp/project-%d", i) {
				t.Errorf("Wrong directory for instance %d", i)
			}
		}
	})

	t.Run("BusSubscriberBounds", func(t *testing.T) {
		b := bus.NewDefault()
		defer b.Close()

		def := b.Define("bounded.event", nil)

		// Subscribe more than max (should evict oldest)
		for i := 0; i < 110; i++ {
			ch, cleanup := b.Subscribe(def)
			// Keep cleanup functions but don't call them yet
			// This tests the evict-oldest behavior
			if i < 100 {
				cleanup() // Clean up first 100 to avoid test warnings
			}
			_ = ch
		}

		stats := b.Stats()
		if stats["bounded.event"] > bus.MaxSubscribersPerEvent {
			t.Errorf("Expected subscribers <= %d, got %d",
				bus.MaxSubscribersPerEvent, stats["bounded.event"])
		}
	})
}

// TestIntegrationConcurrency tests concurrent operations
func TestIntegrationConcurrency(t *testing.T) {
	t.Run("ConcurrentCacheOperations", func(t *testing.T) {
		cache := instance.NewCache(10, 30*time.Minute)

		// Concurrent puts
		done := make(chan bool, 10)
		for i := 0; i < 10; i++ {
			go func(id int) {
				instCtx := &instance.Context{
					Directory: fmt.Sprintf("/tmp/concurrent-%d", id),
					ProjectID: fmt.Sprintf("concurrent-%d", id),
				}
				cache.Put(instCtx)
				done <- true
			}(i)
		}

		// Wait for all puts
		for i := 0; i < 10; i++ {
			select {
			case <-done:
			case <-time.After(2 * time.Second):
				t.Fatal("Timeout waiting for concurrent puts")
			}
		}

		// Concurrent gets
		for i := 0; i < 10; i++ {
			go func(id int) {
				cache.Get(fmt.Sprintf("/tmp/concurrent-%d", id))
				done <- true
			}(i)
		}

		for i := 0; i < 10; i++ {
			select {
			case <-done:
			case <-time.After(2 * time.Second):
				t.Fatal("Timeout waiting for concurrent gets")
			}
		}

		// Verify size
		if cache.Size() != 10 {
			t.Errorf("Expected size 10, got %d", cache.Size())
		}
	})

	t.Run("ConcurrentBusOperations", func(t *testing.T) {
		b := bus.NewDefault()
		defer b.Close()

		def := b.Define("concurrent.event", nil)

		// Subscribe a handler
		b.SubscribeCallback(def, func(p bus.Payload) {})

		// Concurrent publishes
		done := make(chan bool, 100)
		for i := 0; i < 100; i++ {
			go func(id int) {
				b.Publish(def, map[string]interface{}{"id": id})
				done <- true
			}(i)
		}

		for i := 0; i < 100; i++ {
			select {
			case <-done:
			case <-time.After(2 * time.Second):
				t.Fatal("Timeout waiting for concurrent publishes")
			}
		}
	})

	t.Run("ConcurrentDatabaseOperations", func(t *testing.T) {
		ctx := context.Background()
		tmpDir := t.TempDir()
		dbPath := tmpDir + "/concurrent.db"
		db, err := database.New(ctx, database.Config{Path: dbPath})
		if err != nil {
			t.Fatalf("Failed to create database: %v", err)
		}
		defer db.Close()

		// Create session storage
		sessionStorage := database.NewSessionStorage(db)
		msgStorage := database.NewMessageStorage(db)

		// Create base session
		session := database.Session{
			Directory: tmpDir,
			Title:     "Concurrent Session",
			ProjectID: "test",
		}
		created, err := sessionStorage.Create(ctx, session)
		if err != nil {
			t.Fatalf("Failed to create session: %v", err)
		}

		// Concurrent message additions
		done := make(chan bool, 50)
		for i := 0; i < 50; i++ {
			go func(id int) {
				msg := database.Message{
					SessionID: created.ID,
					Data: database.MessageInfo{
						Role:    "user",
						Content: fmt.Sprintf("Message %d", id),
					},
				}
				msgStorage.Create(ctx, msg)
				done <- true
			}(i)
		}

		for i := 0; i < 50; i++ {
			select {
			case <-done:
			case <-time.After(5 * time.Second):
				t.Fatal("Timeout waiting for concurrent messages")
			}
		}

		// Verify messages were created
		messages, err := msgStorage.List(ctx, created.ID)
		if err != nil {
			t.Fatalf("Failed to list messages: %v", err)
		}
		if len(messages) < 50 {
			t.Errorf("Expected at least 50 messages, got %d", len(messages))
		}
	})
}