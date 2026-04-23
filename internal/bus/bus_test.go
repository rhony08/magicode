package bus

import (
	"context"
	"sync"
	"testing"
	"time"
)

func TestNew(t *testing.T) {
	bus := NewDefault()

	if bus == nil {
		t.Fatal("Expected bus to be created")
	}

	if bus.maxSubs != MaxSubscribersPerEvent {
		t.Errorf("Expected maxSubs=%d, got %d", MaxSubscribersPerEvent, bus.maxSubs)
	}
}

func TestDefine(t *testing.T) {
	bus := NewDefault()

	def := bus.Define("test.event", nil)

	if def.Type != "test.event" {
		t.Errorf("Expected Type=test.event, got %s", def.Type)
	}

	// Check registry
	registry := bus.Registry()
	if registry["test.event"].Type != "test.event" {
		t.Error("Expected event to be in registry")
	}
}

func TestPublishSubscribe(t *testing.T) {
	bus := NewDefault()

	def := bus.Define("test.event", nil)

	// Subscribe
	ch, cleanup := bus.Subscribe(def)
	defer cleanup()

	// Publish
	err := bus.Publish(def, map[string]string{"key": "value"})
	if err != nil {
		t.Fatalf("Publish failed: %v", err)
	}

	// Receive
	select {
	case payload := <-ch:
		if payload.Type != "test.event" {
			t.Errorf("Expected Type=test.event, got %s", payload.Type)
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("Expected to receive event")
	}
}

func TestSubscribeAll(t *testing.T) {
	bus := NewDefault()

	def1 := bus.Define("event1", nil)
	def2 := bus.Define("event2", nil)

	// Subscribe to all
	ch, cleanup := bus.SubscribeAll()
	defer cleanup()

	// Publish both events
	bus.Publish(def1, nil)
	bus.Publish(def2, nil)

	// Receive both
	count := 0
	for i := 0; i < 2; i++ {
		select {
		case payload := <-ch:
			count++
			if payload.Type != "event1" && payload.Type != "event2" {
				t.Errorf("Unexpected event type: %s", payload.Type)
			}
		case <-time.After(100 * time.Millisecond):
			t.Error("Timeout waiting for event")
		}
	}

	if count != 2 {
		t.Errorf("Expected 2 events, got %d", count)
	}
}

func TestSubscribeCallback(t *testing.T) {
	bus := NewDefault()

	def := bus.Define("test.event", nil)

	var received Payload
	var mu sync.Mutex

	// Subscribe with callback
	unsub := bus.SubscribeCallback(def, func(p Payload) {
		mu.Lock()
		received = p
		mu.Unlock()
	})

	// Publish
	bus.Publish(def, map[string]string{"key": "value"})

	// Wait for callback
	time.Sleep(50 * time.Millisecond)

	mu.Lock()
	if received.Type != "test.event" {
		t.Errorf("Expected Type=test.event, got %s", received.Type)
	}
	mu.Unlock()

	// Unsubscribe
	unsub()

	// Publish again - should not receive
	bus.Publish(def, nil)

	time.Sleep(50 * time.Millisecond)

	// Stats should show 0 subscribers
	stats := bus.Stats()
	if stats["test.event"] != 0 {
		t.Errorf("Expected 0 subscribers after unsubscribe, got %d", stats["test.event"])
	}
}

func TestMaxSubscribers(t *testing.T) {
	bus := New(context.Background(), nil)
	bus.maxSubs = 3 // Low limit for testing

	def := bus.Define("test.event", nil)

	// Subscribe max number
	for i := 0; i < 3; i++ {
		_, cleanup := bus.Subscribe(def)
		defer cleanup()
	}

	// Subscribe 4th - should evict first
	_, cleanup4 := bus.Subscribe(def)
	defer cleanup4()

	// Stats should show 3 subscribers
	stats := bus.Stats()
	if stats["test.event"] != 3 {
		t.Errorf("Expected 3 subscribers, got %d", stats["test.event"])
	}
}

func TestBackpressure(t *testing.T) {
	bus := NewDefault()

	def := bus.Define("test.event", nil)

	// Subscribe
	_, cleanup := bus.Subscribe(def)
	defer cleanup()

	// Publish many events (more than buffer)
	// The bus should handle this gracefully without blocking
	for i := 0; i < SubscriberBufferSize + 10; i++ {
		err := bus.Publish(def, i)
		if err != nil {
			// Publish should not fail - it drops events when channel is full
			t.Errorf("Publish failed: %v", err)
		}
	}

	// Test passes if we successfully published without blocking
	// Backpressure handling is done by dropping events when channels are full
}

func TestClose(t *testing.T) {
	bus := NewDefault()

	def := bus.Define("test.event", nil)
	bus.Subscribe(def) // Don't cleanup - bus.Close() will close channels

	// Close bus
	bus.Close()

	// Publishing after close should fail
	err := bus.Publish(def, nil)
	if err == nil {
		t.Error("Expected error when publishing to closed bus")
	}

	// Stats should show 0
	stats := bus.Stats()
	if stats["wildcard"] != 0 {
		t.Error("Expected 0 wildcard after close")
	}
}

func TestClearRegistry(t *testing.T) {
	bus := NewDefault()

	bus.Define("event1", nil)
	bus.Define("event2", nil)

	registry := bus.Registry()
	if len(registry) != 2 {
		t.Errorf("Expected 2 events in registry, got %d", len(registry))
	}

	bus.ClearRegistry()

	registry = bus.Registry()
	if len(registry) != 0 {
		t.Errorf("Expected 0 events after clear, got %d", len(registry))
	}
}

func TestStats(t *testing.T) {
	bus := NewDefault()

	def1 := bus.Define("event1", nil)
	def2 := bus.Define("event2", nil)

	_, c1 := bus.Subscribe(def1)
	defer c1()
	_, c2 := bus.Subscribe(def1)
	defer c2()
	_, c3 := bus.Subscribe(def2)
	defer c3()
	_, c4 := bus.SubscribeAll()
	defer c4()

	stats := bus.Stats()

	if stats["event1"] != 2 {
		t.Errorf("Expected 2 subscribers for event1, got %d", stats["event1"])
	}
	if stats["event2"] != 1 {
		t.Errorf("Expected 1 subscriber for event2, got %d", stats["event2"])
	}
	if stats["wildcard"] != 1 {
		t.Errorf("Expected 1 wildcard subscriber, got %d", stats["wildcard"])
	}
	if stats["total_events"] != 2 {
		t.Errorf("Expected 2 total events, got %d", stats["total_events"])
	}
}

func TestGlobalBus(t *testing.T) {
	ResetGlobal()

	ch, unsub := SubscribeGlobal()
	defer unsub()

	// Publish global event
	PublishGlobal("/dir", "project", "workspace", Payload{
		Type:       "test",
		Properties: nil,
	})

	// Receive
	select {
	case event := <-ch:
		if event.Directory != "/dir" {
			t.Errorf("Expected Directory=/dir, got %s", event.Directory)
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("Expected to receive global event")
	}

	CloseGlobal()
}

func TestGlobalBusReset(t *testing.T) {
	// Create subscriber
	ch1, _ := SubscribeGlobal()

	// Reset should close channel
	ResetGlobal()

	// Channel should be closed
	select {
	case _, ok := <-ch1:
		if ok {
			t.Error("Expected channel to be closed after reset")
		}
	default:
		// Empty and closed
	}

	// Can subscribe again after reset
	ch2, unsub := SubscribeGlobal()
	defer unsub()
	
	PublishGlobal("/test", "", "", Payload{Type: "test"})
	
	select {
	case <-ch2:
		// Received
	case <-time.After(100 * time.Millisecond):
		t.Error("Expected to receive event after reset")
	}
}

func TestContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	bus := New(ctx, nil)

	def := bus.Define("test.event", nil)
	bus.Subscribe(def) // Subscribe before cancel

	// Cancel context
	cancel()

	// Publishing should fail
	err := bus.Publish(def, nil)
	if err == nil {
		t.Error("Expected error after context cancellation")
	}

	// Subscribing should fail (return nil channel)
	ch2, cleanup2 := bus.Subscribe(def)
	if ch2 != nil {
		cleanup2()
		t.Error("Expected nil channel when subscribing after context cancellation")
	}
}

func TestCleanupFunction(t *testing.T) {
	bus := NewDefault()

	def := bus.Define("test.event", nil)

	// Subscribe
	_, cleanup := bus.Subscribe(def)

	// Check stats
	stats := bus.Stats()
	if stats["test.event"] != 1 {
		t.Errorf("Expected 1 subscriber, got %d", stats["test.event"])
	}

	// Call cleanup
	cleanup()

	// Check stats again
	stats = bus.Stats()
	if stats["test.event"] != 0 {
		t.Errorf("Expected 0 subscribers after cleanup, got %d", stats["test.event"])
	}
}

func TestSubscribeAllCleanup(t *testing.T) {
	bus := NewDefault()

	_, cleanup := bus.SubscribeAll()

	stats := bus.Stats()
	if stats["wildcard"] != 1 {
		t.Errorf("Expected 1 wildcard subscriber, got %d", stats["wildcard"])
	}

	cleanup()

	stats = bus.Stats()
	if stats["wildcard"] != 0 {
		t.Errorf("Expected 0 wildcard subscribers after cleanup, got %d", stats["wildcard"])
	}
}