package instance

import (
	"fmt"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

// Context represents an instance context (similar to InstanceContext in TS)
type Context struct {
	Directory string
	Worktree  string
	ProjectID string
	CreatedAt time.Time
	LastAccess time.Time
	element   *Element // LRU tracking
}

// Cache is a bounded LRU cache for instances
type Cache struct {
	instances map[string]*Context
	lru       *List
	maxSize   int
	ttl       time.Duration
	mu        sync.RWMutex
}

// NewCache creates a bounded instance cache
func NewCache(maxSize int, ttl time.Duration) *Cache {
	return &Cache{
		instances: make(map[string]*Context),
		lru:       &List{},
		maxSize:   maxSize,
		ttl:       ttl,
	}
}

// Get retrieves an instance from the cache
func (c *Cache) Get(directory string) (*Context, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if ctx, ok := c.instances[directory]; ok {
		// Move to front of LRU
		c.lru.moveToFront(ctx.element)
		ctx.LastAccess = time.Now()
		return ctx, nil
	}

	return nil, ErrNotFound
}

// Put adds an instance to the cache
func (c *Cache) Put(ctx *Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Check if already exists
	if existing, ok := c.instances[ctx.Directory]; ok {
		// Update existing
		c.lru.moveToFront(existing.element)
		existing.LastAccess = time.Now()
		return nil
	}

	// Evict if over capacity
	if len(c.instances) >= c.maxSize {
		if err := c.evictOldest(); err != nil {
			return err
		}
	}

	// Add to cache
	ctx.element = c.lru.pushFront(ctx)
	ctx.CreatedAt = time.Now()
	ctx.LastAccess = time.Now()
	c.instances[ctx.Directory] = ctx

	return nil
}

// evictOldest removes the oldest (least recently used) instance
func (c *Cache) evictOldest() error {
	oldest := c.lru.back()
	if oldest == nil {
		return nil
	}

	ctx := oldest.Value.(*Context)

	// Cleanup instance resources
	if err := ctx.Close(); err != nil {
		// Log but continue eviction
		fmt.Printf("error closing instance %s: %v\n", ctx.Directory, err)
	}

	// Remove from cache
	c.lru.remove(oldest)
	delete(c.instances, ctx.Directory)

	return nil
}

// Remove explicitly removes an instance
func (c *Cache) Remove(directory string) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	ctx, ok := c.instances[directory]
	if !ok {
		return ErrNotFound
	}

	// Cleanup
	if err := ctx.Close(); err != nil {
		return err
	}

	// Remove from cache
	c.lru.remove(ctx.element)
	delete(c.instances, directory)

	return nil
}

// CleanupStale removes instances that haven't been accessed recently
func (c *Cache) CleanupStale() int {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now()
	count := 0

	for dir, ctx := range c.instances {
		if now.Sub(ctx.LastAccess) > c.ttl {
			ctx.Close()
			c.lru.remove(ctx.element)
			delete(c.instances, dir)
			count++
		}
	}

	return count
}

// Close implements cleanup for the context
func (ctx *Context) Close() error {
	// TODO: Cleanup resources (LSP clients, PTY sessions, etc.)
	// This will be implemented as we add more components
	return nil
}

// Size returns the current cache size
func (c *Cache) Size() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.instances)
}

// List returns all cached directories
func (c *Cache) List() []string {
	c.mu.RLock()
	defer c.mu.RUnlock()

	result := make([]string, 0, len(c.instances))
	for dir := range c.instances {
		result = append(result, dir)
	}
	return result
}

// Contains checks if a path is within the instance boundary
func (ctx *Context) ContainsPath(filepath string) bool {
	// Check if path is inside Instance.Directory
	if contains(ctx.Directory, filepath) {
		return true
	}
	
	// Check worktree (skip if "/" to avoid matching any path)
	if ctx.Worktree == "/" {
		return false
	}
	
	return contains(ctx.Worktree, filepath)
}

// contains checks if target is inside base directory
func contains(base, target string) bool {
	rel, err := filepath.Rel(base, target)
	if err != nil {
		return false
	}
	return !filepath.IsAbs(rel) && !strings.HasPrefix(rel, "..")
}

// Errors
var (
	ErrNotFound      = fmt.Errorf("instance not found")
	ErrCacheFull     = fmt.Errorf("instance cache is full")
)

// --- Simple doubly-linked list for LRU tracking ---

type Element struct {
	Value interface{}
	next  *Element
	prev  *Element
}

type List struct {
	head *Element
	tail *Element
	len  int
}

func (l *List) pushFront(value interface{}) *Element {
	e := &Element{Value: value}
	e.next = l.head
	e.prev = nil

	if l.head != nil {
		l.head.prev = e
	}
	l.head = e

	if l.tail == nil {
		l.tail = e
	}

	l.len++
	return e
}

func (l *List) moveToFront(e *Element) {
	if e == l.head {
		return
	}

	// Remove from current position
	if e.prev != nil {
		e.prev.next = e.next
	}
	if e.next != nil {
		e.next.prev = e.prev
	}
	if e == l.tail {
		l.tail = e.prev
	}

	// Move to front
	e.next = l.head
	e.prev = nil
	if l.head != nil {
		l.head.prev = e
	}
	l.head = e
}

func (l *List) remove(e *Element) {
	if e.prev != nil {
		e.prev.next = e.next
	} else {
		l.head = e.next
	}

	if e.next != nil {
		e.next.prev = e.prev
	} else {
		l.tail = e.prev
	}

	l.len--
}

func (l *List) back() *Element {
	return l.tail
}

func (l *List) Len() int {
	return l.len
}