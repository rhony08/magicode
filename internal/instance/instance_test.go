package instance

import (
	"context"
	"testing"
	"time"
)

// testLogger is a simple logger for testing
type testLogger struct {
	logs []string
}

func (l *testLogger) Info(msg string, data map[string]interface{}) {
	l.logs = append(l.logs, "INFO: "+msg)
}

func (l *testLogger) Error(msg string, data map[string]interface{}) {
	l.logs = append(l.logs, "ERROR: "+msg)
}

func TestNewManager(t *testing.T) {
	logger := &testLogger{}
	manager := NewManager(logger)

	if manager == nil {
		t.Fatal("Expected manager to be created")
	}

	if manager.cache == nil {
		t.Error("Expected cache to be initialized")
	}
}

func TestManagerProvide(t *testing.T) {
	logger := &testLogger{}
	manager := NewManager(logger)

	ctx, err := manager.Provide("/test/dir")
	if err != nil {
		t.Fatalf("Provide failed: %v", err)
	}

	if ctx.Directory != "/test/dir" {
		t.Errorf("Expected Directory=/test/dir, got %s", ctx.Directory)
	}

	// Providing again should return cached instance
	ctx2, err := manager.Provide("/test/dir")
	if err != nil {
		t.Fatalf("Provide (cached) failed: %v", err)
	}

	if ctx != ctx2 {
		t.Error("Expected cached instance to be returned")
	}
}

func TestManagerGet(t *testing.T) {
	logger := &testLogger{}
	manager := NewManager(logger)

	ctx := context.Background()

	// Get without instance in context should fail
	_, err := manager.Get(ctx)
	if err != ErrNotFound {
		t.Errorf("Expected ErrNotFound, got %v", err)
	}

	// With instance in context
	inst := &Context{Directory: "/test"}
	ctxWithInst := WithContext(ctx, inst)

	_, err = manager.Get(ctxWithInst)
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	// Verify context value
	val := ctxWithInst.Value(instanceKey{})
	if val == nil {
		t.Error("Expected instance key to be set in context")
	}
}

func TestManagerDirectory(t *testing.T) {
	logger := &testLogger{}
	manager := NewManager(logger)

	inst := &Context{Directory: "/test/dir"}
	ctx := WithContext(context.Background(), inst)

	dir := manager.Directory(ctx)
	if dir != "/test/dir" {
		t.Errorf("Expected Directory=/test/dir, got %s", dir)
	}
}

func TestManagerDispose(t *testing.T) {
	logger := &testLogger{}
	manager := NewManager(logger)

	manager.Provide("/test/dir")

	err := manager.Dispose("/test/dir")
	if err != nil {
		t.Fatalf("Dispose failed: %v", err)
	}

	// Should not be in cache
	_, err = manager.Provide("/test/dir")
	if err != nil {
		// Provide should create new instance, not fail
		t.Fatalf("Provide after dispose should create new: %v", err)
	}
}

func TestManagerDisposeAll(t *testing.T) {
	logger := &testLogger{}
	manager := NewManager(logger)

	manager.Provide("/dir1")
	manager.Provide("/dir2")
	manager.Provide("/dir3")

	err := manager.DisposeAll()
	if err != nil {
		t.Fatalf("DisposeAll failed: %v", err)
	}

	// Cache should be empty
	if manager.cache.Size() != 0 {
		t.Errorf("Expected cache size=0, got %d", manager.cache.Size())
	}
}

func TestManagerCleanup(t *testing.T) {
	logger := &testLogger{}
	manager := NewManager(logger)
	manager.cache.ttl = 1 * time.Second // Short TTL

	manager.Provide("/dir1")

	// Wait for TTL
	time.Sleep(2 * time.Second)

	count := manager.Cleanup()
	if count != 1 {
		t.Errorf("Expected 1 instance cleaned, got %d", count)
	}

	if manager.cache.Size() != 0 {
		t.Errorf("Expected size=0 after cleanup, got %d", manager.cache.Size())
	}
}

func TestManagerStats(t *testing.T) {
	logger := &testLogger{}
	manager := NewManager(logger)

	manager.Provide("/dir1")
	manager.Provide("/dir2")

	stats := manager.Stats()

	if stats["size"] != 2 {
		t.Errorf("Expected size=2, got %d", stats["size"])
	}
}

func TestWithContext(t *testing.T) {
	inst := &Context{Directory: "/test"}
	ctx := WithContext(context.Background(), inst)

	// Current() uses global storage which is separate from context
	// We mainly test WithContext through context.Value

	// Test that context value is set
	val := ctx.Value(instanceKey{})
	if val == nil {
		t.Error("Expected instance key to be set in context")
	}

	ctxVal, ok := val.(*Context)
	if !ok || ctxVal.Directory != "/test" {
		t.Error("Expected context value to be *Context with Directory=/test")
	}
}

func TestBind(t *testing.T) {
	inst := &Context{Directory: "/test"}

	var result string
	fn := func() string {
		current, err := Current()
		if err != nil {
			return "error"
		}
		return current.Directory
	}

	boundFn := Bind(inst, fn)
	result = boundFn()

	if result != "/test" {
		t.Errorf("Expected result=/test, got %s", result)
	}
}

func TestRestore(t *testing.T) {
	inst := &Context{Directory: "/test"}

	var currentInst *Context
	result := Restore(inst, func() string {
		currentInst, _ = Current()
		return currentInst.Directory
	})

	if result != "/test" {
		t.Errorf("Expected result=/test, got %s", result)
	}
}

func TestSetCurrent(t *testing.T) {
	inst := &Context{Directory: "/test"}

	SetCurrent(inst)

	current, err := Current()
	if err != nil {
		t.Fatalf("Current failed: %v", err)
	}

	if current.Directory != "/test" {
		t.Errorf("Expected Directory=/test, got %s", current.Directory)
	}
}

func TestNoopLogger(t *testing.T) {
	logger := &noopLogger{}

	logger.Info("test", nil)
	logger.Error("test", nil)

	// noopLogger does nothing, so no errors expected
}

func TestNewManagerNilLogger(t *testing.T) {
	manager := NewManager(nil)

	if manager == nil {
		t.Fatal("Expected manager to be created with nil logger")
	}

	if manager.logger == nil {
		t.Error("Expected noopLogger to be set")
	}
}