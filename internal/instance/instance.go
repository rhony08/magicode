package instance

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

const (
	// MaxInstances is the maximum number of cached instances
	MaxInstances = 10
	// InstanceTTL is the time-to-live for idle instances
	InstanceTTL = 30 * time.Minute
)

// instanceKey is the context key for instances
type instanceKey struct{}

// Manager manages project instances
type Manager struct {
	cache   *Cache
	mu      sync.RWMutex
	logger  Logger
}

// Logger interface for instance logging
type Logger interface {
	Info(msg string, data map[string]interface{})
	Error(msg string, data map[string]interface{})
}

// noopLogger is a default logger that does nothing
type noopLogger struct{}

func (l *noopLogger) Info(msg string, data map[string]interface{}) {}
func (l *noopLogger) Error(msg string, data map[string]interface{}) {}

// NewManager creates a new instance manager
func NewManager(logger Logger) *Manager {
	if logger == nil {
		logger = &noopLogger{}
	}
	return &Manager{
		cache:  NewCache(MaxInstances, InstanceTTL),
		logger: logger,
	}
}

// Provide creates or retrieves an instance for a directory
func (m *Manager) Provide(directory string) (*Context, error) {
	// Resolve directory path
	dir, err := filepath.Abs(directory)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve directory: %w", err)
	}

	// Check cache
	ctx, err := m.cache.Get(dir)
	if err == nil {
		m.logger.Info("using cached instance", map[string]interface{}{"directory": dir})
		return ctx, nil
	}

	// Create new instance
	m.logger.Info("creating new instance", map[string]interface{}{"directory": dir})

	ctx = &Context{
		Directory: dir,
		CreatedAt: time.Now(),
	}

	// Determine project root (git worktree or directory)
	ctx.Worktree = m.findWorktree(dir)
	ctx.ProjectID = m.findProjectID(dir)

	// Add to cache
	if err := m.cache.Put(ctx); err != nil {
		return nil, err
	}

	return ctx, nil
}

// Get retrieves the current instance from context
func (m *Manager) Get(ctx context.Context) (*Context, error) {
	// Check context value
	if inst, ok := ctx.Value(instanceKey{}).(*Context); ok {
		return inst, nil
	}
	return nil, ErrNotFound
}

// Directory returns the directory from the current instance
func (m *Manager) Directory(ctx context.Context) string {
	inst, err := m.Get(ctx)
	if err != nil {
		return ""
	}
	return inst.Directory
}

// Dispose removes an instance from the cache
func (m *Manager) Dispose(directory string) error {
	m.logger.Info("disposing instance", map[string]interface{}{"directory": directory})
	return m.cache.Remove(directory)
}

// DisposeAll removes all instances
func (m *Manager) DisposeAll() error {
	m.logger.Info("disposing all instances", nil)

	dirs := m.cache.List()
	for _, dir := range dirs {
		if err := m.cache.Remove(dir); err != nil {
			m.logger.Error("failed to dispose instance", map[string]interface{}{
				"directory": dir,
				"error":     err.Error(),
			})
		}
	}

	return nil
}

// Cleanup runs periodic cleanup of stale instances
func (m *Manager) Cleanup() int {
	count := m.cache.CleanupStale()
	if count > 0 {
		m.logger.Info("cleaned up stale instances", map[string]interface{}{"count": count})
	}
	return count
}

// findWorktree finds the git worktree root
func (m *Manager) findWorktree(dir string) string {
	// Walk up directory tree looking for .git
	current := dir
	for {
		gitDir := filepath.Join(current, ".git")
		if _, err := os.Stat(gitDir); err == nil {
			return current
		}

		parent := filepath.Dir(current)
		if parent == current {
			// Reached root
			break
		}
		current = parent
	}

	// No git found, use "/" as worktree (per TS implementation)
	return "/"
}

// findProjectID generates a project ID
func (m *Manager) findProjectID(dir string) string {
	// For git projects, use first commit hash as project ID
	// For non-git projects, use directory path hash
	// TODO: Implement actual git logic
	return filepath.Base(dir)
}

// WithContext returns a new context with the instance attached
func WithContext(parent context.Context, inst *Context) context.Context {
	return context.WithValue(parent, instanceKey{}, inst)
}

// CurrentInstance holds the current instance for sync code
var currentInstance struct {
	ctx *Context
	mu  sync.RWMutex
}

// Bind captures the instance context and returns a function that restores it
// Similar to Instance.bind in TS for native callbacks
func Bind[T any](ctx *Context, fn func() T) func() T {
	return func() T {
		currentInstance.mu.Lock()
		old := currentInstance.ctx
		currentInstance.ctx = ctx
		currentInstance.mu.Unlock()

		defer func() {
			currentInstance.mu.Lock()
			currentInstance.ctx = old
			currentInstance.mu.Unlock()
		}()

		return fn()
	}
}

// Restore runs a function within a specific instance context
func Restore[T any](inst *Context, fn func() T) T {
	return Bind(inst, fn)()
}

// Current returns the current instance from global storage
func Current() (*Context, error) {
	currentInstance.mu.RLock()
	defer currentInstance.mu.RUnlock()

	if currentInstance.ctx == nil {
		return nil, ErrNotFound
	}
	return currentInstance.ctx, nil
}

// SetCurrent sets the current instance
func SetCurrent(ctx *Context) {
	currentInstance.mu.Lock()
	defer currentInstance.mu.Unlock()
	currentInstance.ctx = ctx
}

// Stats returns cache statistics
func (m *Manager) Stats() map[string]int {
	return map[string]int{
		"size":     m.cache.Size(),
		"max_size": MaxInstances,
	}
}