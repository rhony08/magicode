package bus

import (
	"context"
	"fmt"
	"sync"
)

// Event represents a bus event
type Event struct {
	Type       string
	Properties interface{}
}

// Definition represents an event definition
type Definition struct {
	Type       string
	Properties interface{} // Schema for validation (if needed)
}

// Payload is the event payload wrapper
type Payload struct {
	Type       string
	Properties interface{}
}

// Logger interface for bus logging
type Logger interface {
	Info(msg string, data map[string]interface{})
}

// noopLogger is a default logger that does nothing
type noopLogger struct{}

func (l *noopLogger) Info(msg string, data map[string]interface{}) {}

const (
	// MaxSubscribersPerEvent is the maximum subscribers per event type
	MaxSubscribersPerEvent = 100
	// SubscriberBufferSize is the buffer size for subscriber channels
	SubscriberBufferSize = 50
)

// Service provides pub/sub functionality with proper cleanup
type Service struct {
	subscribers map[string][]chan Payload
	wildcard    []chan Payload
	maxSubs     int
	ctx         context.Context
	cancel      context.CancelFunc
	registry    map[string]Definition
	logger      Logger
	mu          sync.RWMutex
}

// New creates a new bus service
func New(parentCtx context.Context, logger Logger) *Service {
	if logger == nil {
		logger = &noopLogger{}
	}

	ctx, cancel := context.WithCancel(parentCtx)

	return &Service{
		subscribers: make(map[string][]chan Payload),
		wildcard:    make([]chan Payload, 0),
		maxSubs:     MaxSubscribersPerEvent,
		ctx:         ctx,
		cancel:      cancel,
		registry:    make(map[string]Definition),
		logger:      logger,
	}
}

// NewDefault creates a bus with default settings
func NewDefault() *Service {
	return New(context.Background(), nil)
}

// Define creates a new event definition
func (s *Service) Define(eventType string, properties interface{}) Definition {
	def := Definition{
		Type:       eventType,
		Properties: properties,
	}

	s.mu.Lock()
	s.registry[eventType] = def
	s.mu.Unlock()

	return def
}

// Publish emits an event to all subscribers
func (s *Service) Publish(def Definition, properties interface{}) error {
	payload := Payload{
		Type:       def.Type,
		Properties: properties,
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	// Check if context is done
	if s.ctx.Err() != nil {
		return s.ctx.Err()
	}

	// Send to type-specific subscribers
	if subs, ok := s.subscribers[def.Type]; ok {
		for _, ch := range subs {
			select {
			case ch <- payload:
				// Sent successfully
			case <-s.ctx.Done():
				return s.ctx.Err()
			default:
				// Channel full, skip (backpressure handling)
				s.logger.Info("subscriber channel full, skipping", map[string]interface{}{
					"type": def.Type,
				})
			}
		}
	}

	// Send to wildcard subscribers
	for _, ch := range s.wildcard {
		select {
		case ch <- payload:
		case <-s.ctx.Done():
			return s.ctx.Err()
		default:
			// Channel full
		}
	}

	s.logger.Info("published event", map[string]interface{}{
		"type": def.Type,
	})

	return nil
}

// Subscribe creates a subscription for a specific event type
// Returns a receive-only channel and a cleanup function
func (s *Service) Subscribe(def Definition) (<-chan Payload, func()) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Check if context is done
	if s.ctx.Err() != nil {
		return nil, func() {}
	}

	// Check max subscribers
	if len(s.subscribers[def.Type]) >= s.maxSubs {
		// Evict oldest subscriber safely
		if len(s.subscribers[def.Type]) > 0 {
			oldest := s.subscribers[def.Type][0]
			s.subscribers[def.Type] = s.subscribers[def.Type][1:]
			// Close channel after removing from list to prevent sends to closed channel
			close(oldest)
		}
	}

	ch := make(chan Payload, SubscriberBufferSize)
	s.subscribers[def.Type] = append(s.subscribers[def.Type], ch)

	s.logger.Info("subscribed to event", map[string]interface{}{
		"type": def.Type,
	})

	// Return cleanup function
	return ch, func() {
		s.unsubscribeAndClose(def.Type, ch)
	}
}

// SubscribeAll creates a subscription for all events
// Returns a receive-only channel and a cleanup function
func (s *Service) SubscribeAll() (<-chan Payload, func()) {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Check if context is done
	if s.ctx.Err() != nil {
		return nil, func() {}
	}

	// Check max wildcard subscribers
	if len(s.wildcard) >= s.maxSubs {
		oldest := s.wildcard[0]
		s.wildcard = s.wildcard[1:]
		// Close channel after removing from list to prevent sends to closed channel
		close(oldest)
	}

	ch := make(chan Payload, SubscriberBufferSize)
	s.wildcard = append(s.wildcard, ch)

	s.logger.Info("subscribed to all events", map[string]interface{}{
		"type": "*",
	})

	return ch, func() {
		s.unsubscribeAllAndClose(ch)
	}
}

// SubscribeCallback creates a callback-based subscription
func (s *Service) SubscribeCallback(def Definition, callback func(Payload)) func() {
	ch, cleanup := s.Subscribe(def)

	// Start goroutine to handle events
	done := make(chan struct{})
	go func() {
		defer close(done)
		for {
			select {
			case payload, ok := <-ch:
				if !ok {
					return
				}
				callback(payload)
			case <-s.ctx.Done():
				return
			}
		}
	}()

	// Return cleanup function
	return func() {
		cleanup()
		<-done // Wait for goroutine to finish
	}
}

// SubscribeAllCallback creates a callback-based subscription for all events
func (s *Service) SubscribeAllCallback(callback func(Payload)) func() {
	ch, cleanup := s.SubscribeAll()

	done := make(chan struct{})
	go func() {
		defer close(done)
		for {
			select {
			case payload, ok := <-ch:
				if !ok {
					return
				}
				callback(payload)
			case <-s.ctx.Done():
				return
			}
		}
	}()

	return func() {
		cleanup()
		<-done
	}
}

// unsubscribeAndClose removes a subscriber and closes the channel
func (s *Service) unsubscribeAndClose(eventType string, ch chan Payload) {
	s.mu.Lock()
	defer s.mu.Unlock()

	subscribers := s.subscribers[eventType]
	for i, sub := range subscribers {
		if sub == ch {
			s.subscribers[eventType] = append(subscribers[:i], subscribers[i+1:]...)
			close(ch)
			break
		}
	}

	s.logger.Info("unsubscribed from event", map[string]interface{}{
		"type": eventType,
	})
}

// unsubscribeAllAndClose removes a wildcard subscriber and closes the channel
func (s *Service) unsubscribeAllAndClose(ch chan Payload) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for i, sub := range s.wildcard {
		if sub == ch {
			s.wildcard = append(s.wildcard[:i], s.wildcard[i+1:]...)
			close(ch)
			break
		}
	}

	s.logger.Info("unsubscribed from all events", map[string]interface{}{
		"type": "*",
	})
}

// Close shuts down the bus and closes all channels
func (s *Service) Close() error {
	s.cancel()

	s.mu.Lock()
	defer s.mu.Unlock()

	// Close all subscriber channels
	for eventType, subs := range s.subscribers {
		for _, ch := range subs {
			close(ch)
		}
		s.subscribers[eventType] = nil
	}
	s.subscribers = nil

	for _, ch := range s.wildcard {
		close(ch)
	}
	s.wildcard = nil

	// Clear registry
	s.registry = nil

	s.logger.Info("bus closed", nil)

	return nil
}

// Registry returns the event registry (bounded)
func (s *Service) Registry() map[string]Definition {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Return copy to prevent modification
	result := make(map[string]Definition, len(s.registry))
	for k, v := range s.registry {
		result[k] = v
	}
	return result
}

// ClearRegistry clears the event registry
func (s *Service) ClearRegistry() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.registry = make(map[string]Definition)
}

// Stats returns subscriber statistics
func (s *Service) Stats() map[string]int {
	s.mu.RLock()
	defer s.mu.RUnlock()

	stats := make(map[string]int)
	stats["wildcard"] = len(s.wildcard)
	for eventType, subs := range s.subscribers {
		stats[eventType] = len(subs)
	}
	stats["total_events"] = len(s.registry)
	return stats
}

// Context returns the bus context
func (s *Service) Context() context.Context {
	return s.ctx
}

// --- Global Bus for cross-instance communication ---

// GlobalBus is a singleton for cross-instance events (similar to GlobalBus in TS)
type GlobalBus struct {
	subscribers []chan GlobalEvent
	mu          sync.RWMutex
}

// GlobalEvent represents a global event
type GlobalEvent struct {
	Directory string
	Project   string
	Workspace string
	Payload   Payload
}

var globalBus = &GlobalBus{
	subscribers: make([]chan GlobalEvent, 0),
}

// SubscribeGlobal subscribes to global events
func SubscribeGlobal() (<-chan GlobalEvent, func()) {
	globalBus.mu.Lock()
	defer globalBus.mu.Unlock()

	ch := make(chan GlobalEvent, 100)
	globalBus.subscribers = append(globalBus.subscribers, ch)

	return ch, func() {
		globalBus.unsubscribeAndCloseGlobal(ch)
	}
}

func (gb *GlobalBus) unsubscribeAndCloseGlobal(ch chan GlobalEvent) {
	gb.mu.Lock()
	defer gb.mu.Unlock()

	for i, sub := range gb.subscribers {
		if sub == ch {
			gb.subscribers = append(gb.subscribers[:i], gb.subscribers[i+1:]...)
			close(ch)
			break
		}
	}
}

// PublishGlobal publishes a global event
func PublishGlobal(directory, project, workspace string, payload Payload) {
	event := GlobalEvent{
		Directory: directory,
		Project:   project,
		Workspace: workspace,
		Payload:   payload,
	}

	globalBus.mu.RLock()
	defer globalBus.mu.RUnlock()

	for _, ch := range globalBus.subscribers {
		select {
		case ch <- event:
		default:
			// Channel full, skip
		}
	}
}

// CloseGlobal closes all global subscribers
func CloseGlobal() {
	globalBus.mu.Lock()
	defer globalBus.mu.Unlock()

	for _, ch := range globalBus.subscribers {
		close(ch)
	}
	globalBus.subscribers = nil
}

// ResetGlobal resets the global bus for testing
func ResetGlobal() {
	globalBus.mu.Lock()
	defer globalBus.mu.Unlock()
	
	for _, ch := range globalBus.subscribers {
		close(ch)
	}
	globalBus.subscribers = make([]chan GlobalEvent, 0)
}

// Errors
var (
	ErrBusClosed      = fmt.Errorf("bus is closed")
	ErrMaxSubscribers = fmt.Errorf("maximum subscribers reached")
)