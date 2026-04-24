package bus

import (
	"testing"
)

// BenchmarkPublish benchmarks publishing events to the bus
func BenchmarkPublish(b *testing.B) {
	bus := NewDefault()
	defer bus.Close()

	def := bus.Define("test.event", nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		bus.Publish(def, map[string]interface{}{"index": i})
	}
}

// BenchmarkPublishWithSubscribers benchmarks publishing with multiple subscribers
func BenchmarkPublishWithSubscribers(b *testing.B) {
	bus := NewDefault()
	defer bus.Close()

	def := bus.Define("test.event", nil)

	// Subscribe multiple listeners
	for i := 0; i < 10; i++ {
		bus.SubscribeCallback(def, func(p Payload) {})
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		bus.Publish(def, map[string]interface{}{"index": i})
	}
}

// BenchmarkSubscribe benchmarks subscribing to the bus
func BenchmarkSubscribe(b *testing.B) {
	bus := NewDefault()
	defer bus.Close()

	def := bus.Define("test.event", nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ch, cleanup := bus.Subscribe(def)
		cleanup()
		_ = ch
	}
}

// BenchmarkSubscribeCallback benchmarks callback-based subscription
func BenchmarkSubscribeCallback(b *testing.B) {
	bus := NewDefault()
	defer bus.Close()

	def := bus.Define("test.event", nil)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cleanup := bus.SubscribeCallback(def, func(p Payload) {})
		cleanup()
	}
}

// BenchmarkDefine benchmarks event definition
func BenchmarkDefine(b *testing.B) {
	bus := NewDefault()
	defer bus.Close()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		bus.Define("test.event", nil)
	}
}

// BenchmarkMemoryBus benchmarks memory usage per operation
func BenchmarkMemoryBus(b *testing.B) {
	bus := NewDefault()
	defer bus.Close()

	def := bus.Define("memory.test", nil)

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		bus.Publish(def, map[string]interface{}{
			"key":   "value",
			"index": i,
		})
	}
}

// BenchmarkGlobalBus benchmarks global bus operations
func BenchmarkGlobalBus(b *testing.B) {
	ResetGlobal()
	defer CloseGlobal()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		PublishGlobal("/tmp/test", "test-project", "test-workspace",
			Payload{Type: "test", Properties: nil})
	}
}

// BenchmarkSubscribeGlobal benchmarks subscribing to global events
func BenchmarkSubscribeGlobal(b *testing.B) {
	ResetGlobal()
	defer CloseGlobal()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ch, cleanup := SubscribeGlobal()
		cleanup()
		_ = ch
	}
}