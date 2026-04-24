package instance

import (
	"fmt"
	"testing"
	"time"
)

// BenchmarkPut benchmarks adding instance to cache
func BenchmarkPut(b *testing.B) {
	cache := NewCache(10, 30*time.Minute)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ctx := &Context{
			Directory: fmt.Sprintf("/tmp/project-%d", i),
		}
		cache.Put(ctx)
	}
}

// BenchmarkPutMultiple benchmarks cache with multiple directories
func BenchmarkPutMultiple(b *testing.B) {
	cache := NewCache(10, 30*time.Minute)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Cycle through 20 directories (cache max is 10)
		ctx := &Context{
			Directory: fmt.Sprintf("/tmp/project-%d", i%20),
		}
		cache.Put(ctx)
	}
}

// BenchmarkGet benchmarks getting an existing instance
func BenchmarkGet(b *testing.B) {
	cache := NewCache(10, 30*time.Minute)

	// Pre-populate cache
	for i := 0; i < 10; i++ {
		cache.Put(&Context{
			Directory: fmt.Sprintf("/tmp/project-%d", i),
		})
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cache.Get("/tmp/project-0")
	}
}

// BenchmarkEviction benchmarks LRU eviction behavior
func BenchmarkEviction(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cache := NewCache(10, 30*time.Minute)

		// Add 20 items to trigger evictions
		for j := 0; j < 20; j++ {
			cache.Put(&Context{
				Directory: fmt.Sprintf("/tmp/project-%d", j),
			})
		}
	}
}

// BenchmarkRemove benchmarks removing an instance
func BenchmarkRemove(b *testing.B) {
	cache := NewCache(10, 30*time.Minute)

	// Pre-populate cache
	for i := 0; i < 10; i++ {
		cache.Put(&Context{
			Directory: fmt.Sprintf("/tmp/project-%d", i),
		})
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cache.Remove(fmt.Sprintf("/tmp/project-%d", i%10))
		// Re-add for next iteration
		cache.Put(&Context{
			Directory: fmt.Sprintf("/tmp/project-%d", i%10),
		})
	}
}

// BenchmarkCleanupStale benchmarks cleanup of stale instances
func BenchmarkCleanupStale(b *testing.B) {
	cache := NewCache(10, 100*time.Millisecond) // Short TTL for testing

	// Pre-populate cache
	for i := 0; i < 10; i++ {
		cache.Put(&Context{
			Directory: fmt.Sprintf("/tmp/project-%d", i),
		})
	}

	// Wait for TTL to expire
	time.Sleep(150 * time.Millisecond)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cache.CleanupStale()
	}
}

// BenchmarkMemoryCache benchmarks memory usage for cache operations
func BenchmarkMemoryCache(b *testing.B) {
	cache := NewCache(10, 30*time.Minute)

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		cache.Put(&Context{
			Directory: "/tmp/test-project",
		})
	}
}

// BenchmarkList benchmarks listing cached directories
func BenchmarkList(b *testing.B) {
	cache := NewCache(10, 30*time.Minute)

	// Pre-populate cache
	for i := 0; i < 10; i++ {
		cache.Put(&Context{
			Directory: fmt.Sprintf("/tmp/project-%d", i),
		})
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cache.List()
	}
}

// BenchmarkSize benchmarks getting cache size
func BenchmarkSize(b *testing.B) {
	cache := NewCache(10, 30*time.Minute)

	// Pre-populate cache
	for i := 0; i < 10; i++ {
		cache.Put(&Context{
			Directory: fmt.Sprintf("/tmp/project-%d", i),
		})
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cache.Size()
	}
}