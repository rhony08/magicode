package instance

import (
	"testing"
	"time"
)

func TestNewCache(t *testing.T) {
	cache := NewCache(10, 30*time.Minute)

	if cache == nil {
		t.Fatal("Expected cache to be created")
	}

	if cache.maxSize != 10 {
		t.Errorf("Expected maxSize=10, got %d", cache.maxSize)
	}

	if cache.Size() != 0 {
		t.Errorf("Expected initial size=0, got %d", cache.Size())
	}
}

func TestCachePutAndGet(t *testing.T) {
	cache := NewCache(10, 30*time.Minute)

	ctx := &Context{
		Directory: "/test/dir",
		Worktree:  "/test/worktree",
		ProjectID: "test-project",
	}

	err := cache.Put(ctx)
	if err != nil {
		t.Fatalf("Put failed: %v", err)
	}

	if cache.Size() != 1 {
		t.Errorf("Expected size=1, got %d", cache.Size())
	}

	// Get the instance
	retrieved, err := cache.Get("/test/dir")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	if retrieved.Directory != ctx.Directory {
		t.Errorf("Expected Directory=%s, got %s", ctx.Directory, retrieved.Directory)
	}

	if retrieved.Worktree != ctx.Worktree {
		t.Errorf("Expected Worktree=%s, got %s", ctx.Worktree, retrieved.Worktree)
	}
}

func TestCacheNotFound(t *testing.T) {
	cache := NewCache(10, 30*time.Minute)

	_, err := cache.Get("/nonexistent")
	if err != ErrNotFound {
		t.Errorf("Expected ErrNotFound, got %v", err)
	}
}

func TestCacheEviction(t *testing.T) {
	cache := NewCache(3, 30*time.Minute)

	// Add 3 instances
	for i := 0; i < 3; i++ {
		ctx := &Context{
			Directory: "/test/dir" + string(rune('a'+i)),
		}
		cache.Put(ctx)
	}

	if cache.Size() != 3 {
		t.Errorf("Expected size=3, got %d", cache.Size())
	}

	// Add 4th instance - should evict oldest
	ctx := &Context{
		Directory: "/test/dir4",
	}
	cache.Put(ctx)

	if cache.Size() != 3 {
		t.Errorf("Expected size=3 after eviction, got %d", cache.Size())
	}

	// First instance should have been evicted
	_, err := cache.Get("/test/dira")
	if err != ErrNotFound {
		t.Error("Expected first instance to be evicted")
	}

	// New instance should exist
	_, err = cache.Get("/test/dir4")
	if err != nil {
		t.Error("Expected new instance to exist")
	}
}

func TestCacheLRU(t *testing.T) {
	cache := NewCache(3, 30*time.Minute)

	// Add instances a, b, c
	ctxA := &Context{Directory: "/a"}
	ctxB := &Context{Directory: "/b"}
	ctxC := &Context{Directory: "/c"}

	cache.Put(ctxA)
	cache.Put(ctxB)
	cache.Put(ctxC)

	// Access instance a (makes it more recently used)
	cache.Get("/a")

	// Add instance d - should evict b (not a)
	ctxD := &Context{Directory: "/d"}
	cache.Put(ctxD)

	// a should still exist (was accessed recently)
	_, err := cache.Get("/a")
	if err != nil {
		t.Error("Expected a to exist (was accessed)")
	}

	// b should be evicted
	_, err = cache.Get("/b")
	if err != ErrNotFound {
		t.Error("Expected b to be evicted")
	}
}

func TestCacheRemove(t *testing.T) {
	cache := NewCache(10, 30*time.Minute)

	ctx := &Context{Directory: "/test/dir"}
	cache.Put(ctx)

	err := cache.Remove("/test/dir")
	if err != nil {
		t.Fatalf("Remove failed: %v", err)
	}

	if cache.Size() != 0 {
		t.Errorf("Expected size=0, got %d", cache.Size())
	}

	// Should not exist
	_, err = cache.Get("/test/dir")
	if err != ErrNotFound {
		t.Error("Expected ErrNotFound after remove")
	}
}

func TestCacheRemoveNonexistent(t *testing.T) {
	cache := NewCache(10, 30*time.Minute)

	err := cache.Remove("/nonexistent")
	if err != ErrNotFound {
		t.Errorf("Expected ErrNotFound, got %v", err)
	}
}

func TestCacheCleanupStale(t *testing.T) {
	cache := NewCache(10, 1*time.Second) // Short TTL for testing

	// Add instance
	ctx := &Context{Directory: "/test/dir"}
	cache.Put(ctx)

	// Should not be stale immediately
	count := cache.CleanupStale()
	if count != 0 {
		t.Errorf("Expected 0 stale instances, got %d", count)
	}

	// Wait for TTL
	time.Sleep(2 * time.Second)

	// Should cleanup stale instance
	count = cache.CleanupStale()
	if count != 1 {
		t.Errorf("Expected 1 stale instance cleaned, got %d", count)
	}

	if cache.Size() != 0 {
		t.Errorf("Expected size=0 after cleanup, got %d", cache.Size())
	}
}

func TestCacheList(t *testing.T) {
	cache := NewCache(10, 30*time.Minute)

	// Add instances
	cache.Put(&Context{Directory: "/dir1"})
	cache.Put(&Context{Directory: "/dir2"})
	cache.Put(&Context{Directory: "/dir3"})

	list := cache.List()
	if len(list) != 3 {
		t.Errorf("Expected 3 items in list, got %d", len(list))
	}

	// Check all directories are in list
	found := map[string]bool{}
	for _, dir := range list {
		found[dir] = true
	}

	if !found["/dir1"] || !found["/dir2"] || !found["/dir3"] {
		t.Error("Expected all directories in list")
	}
}

func TestContainsPath(t *testing.T) {
	ctx := &Context{
		Directory: "/project",
		Worktree:  "/worktree",
	}

	// Inside directory
	if !ctx.ContainsPath("/project/subdir/file.txt") {
		t.Error("Expected path inside directory to be contained")
	}

	// Inside worktree
	if !ctx.ContainsPath("/worktree/subdir/file.txt") {
		t.Error("Expected path inside worktree to be contained")
	}

	// Outside both
	if ctx.ContainsPath("/other/file.txt") {
		t.Error("Expected path outside both to not be contained")
	}
}

func TestContainsPathWithRootWorktree(t *testing.T) {
	ctx := &Context{
		Directory: "/project",
		Worktree:  "/", // Root worktree should not match anything
	}

	// Inside directory
	if !ctx.ContainsPath("/project/subdir/file.txt") {
		t.Error("Expected path inside directory to be contained")
	}

	// Root worktree should NOT match any path
	if ctx.ContainsPath("/other/file.txt") {
		t.Error("Root worktree should not match any path")
	}
}

func TestListOperations(t *testing.T) {
	l := &List{}

	// Test pushFront
	e1 := l.pushFront("first")
	if l.Len() != 1 {
		t.Errorf("Expected len=1, got %d", l.Len())
	}

	e2 := l.pushFront("second")
	if l.Len() != 2 {
		t.Errorf("Expected len=2, got %d", l.Len())
	}
	if e2.Value != "second" {
		t.Errorf("Expected e2.Value=second, got %s", e2.Value)
	}

	// Test back
	back := l.back()
	if back == nil || back.Value != "first" {
		t.Error("Expected back to be 'first'")
	}

	// Test moveToFront
	l.moveToFront(e1)
	back = l.back()
	if back == nil || back.Value != "second" {
		t.Error("Expected back to be 'second' after moveToFront")
	}

	// Test remove
	l.remove(e1)
	if l.Len() != 1 {
		t.Errorf("Expected len=1 after remove, got %d", l.Len())
	}
}

func TestListRemoveHead(t *testing.T) {
	l := &List{}
	e := l.pushFront("value")

	l.remove(e)

	if l.head != nil {
		t.Error("Expected head to be nil after removing only element")
	}
	if l.tail != nil {
		t.Error("Expected tail to be nil after removing only element")
	}
}

func TestListRemoveTail(t *testing.T) {
	l := &List{}
	_ = l.pushFront("first")  // e1 - will be tail
	e2 := l.pushFront("second")

	// Remove tail (e1)
	l.remove(l.back())

	if l.tail != e2 {
		t.Error("Expected tail to be e2 after removing e1")
	}
}