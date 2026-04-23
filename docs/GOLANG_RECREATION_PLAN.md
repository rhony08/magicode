# OpenCode CLI - Go Recreation Plan

**Date:** 2026-04-23
**Last Updated:** 2026-04-23
**Goal:** Recreate the OpenCode CLI in Go for faster performance and lower memory usage
**Reference:** MEMORY_LEAK_REVIEW.md
**Project Location:** `/usr/local/personal-project/opencode/opencode-go/`

---

## 🚀 Current Progress

### Phase 1: Core Infrastructure ✅ COMPLETED

| Component | Status | Tests | Notes |
|-----------|--------|-------|-------|
| Project structure | ✅ Done | - | Created full folder structure |
| go.mod | ✅ Done | - | Dependencies: cobra, viper, xdg, jsonc, uuid |
| CLI entry point | ✅ Done | - | Cobra-based with subcommands |
| Global paths | ✅ Done | ✅ 7 tests | XDG paths with reset for testing |
| Logging system | ✅ Done | ✅ 10 tests | Structured logging with levels |
| Configuration | ✅ Done | ✅ 12 tests | JSON/JSONC parsing, merging |
| Instance cache | ✅ Done | ✅ 14 tests | **Bounded LRU cache (max 10, TTL)** |
| Instance manager | ✅ Done | ✅ 12 tests | Manager with cleanup |
| Bus system | ✅ Done | ✅ 15 tests | **Channel-based pub/sub with cleanup** |
| Makefile | ✅ Done | - | build, test, lint, etc. |
| README | ✅ Done | - | Documentation |

**Total Tests: 60+ tests passing**

### Phase 2: Storage & Session 🚧 NEXT

| Component | Status | Notes |
|-----------|--------|-------|
| SQLite setup | 📋 Pending | Use modernc.org/sqlite (pure Go) |
| Schema definitions | 📋 Pending | Must be compatible with TS version |
| Session manager | 📋 Pending | CRUD operations |
| Message storage | 📋 Pending | Message/part tables |
| Sync system | 📋 Pending | Event synchronization |

---

## 📝 Notes for Next AI

### How to Continue

1. **Read this plan first** - Understand the architecture and memory leak solutions
2. **Run existing tests** - Verify Phase 1 is working:
   ```bash
   export GOPATH=/tmp/go-path
   cd /usr/local/personal-project/opencode/opencode-go
   GO111MODULE=on go test ./internal/... -v
   ```
3. **Continue Phase 2** - Start with SQLite database setup

### Key Files to Review

| File | Purpose | Why Review |
|------|---------|------------|
| `internal/instance/cache.go` | Bounded LRU cache | Core memory leak fix |
| `internal/bus/bus.go` | Channel-based pub/sub | Core memory leak fix |
| `internal/config/config.go` | Configuration loading | Pattern for other modules |
| `opencode/packages/opencode/src/storage/schema.sql.ts` | TS schema | Must match for compatibility |
| `opencode/packages/opencode/src/session/session.sql.ts` | Session tables | Must match for compatibility |

### TypeScript Schema Reference

The Go schema must be compatible with these TS tables:
- `session` - Session metadata
- `message` - Messages in sessions
- `part` - Message parts
- `project` - Project info
- `event` - Sync events
- `event_sequence` - Event ordering
- `account` - User accounts
- `share` - Shared sessions

### Important Patterns Established

1. **Bounded Caches** - Always use `NewCache(maxSize, ttl)` pattern
2. **Cleanup Functions** - Return cleanup func from Subscribe/SubscribeAll
3. **Testing** - Each module has `_test.go` with comprehensive tests
4. **Context Usage** - Use `context.Context` for cancellation
5. **Module Structure** - Each module exports a `Service` with interface

### Testing Requirements

Before proceeding to Phase 3, ensure:
- All Phase 2 tests pass
- SQLite database works with existing TS data
- Sessions can be created/read/updated/deleted

---

## User Decisions (Confirmed)

| Decision | Choice | Notes |
|----------|--------|-------|
| Feature Scope | **Full Parity** | Implement all features before release |
| Providers | **Anthropic + OpenAI** | Minimum viable set for initial version |
| Compatibility | **Compatible** | Must work with existing sessions/config schema |
| Priority | **TUI First** | Focus on terminal UI before HTTP server |
| MCP Integration | **Skip** | Focus on native tools only |
| LSP Integration | **Include** | LSP diagnostics important for coding |
| PTY Integration | **Include** | Terminal sessions are important feature |
| DB Compatibility | **All Tables** | Core, Sync, Account, Share tables must be compatible |

---

## Executive Summary

The current OpenCode CLI (TypeScript/Bun) suffers from significant memory issues:
- **71GB+ virtual memory** at startup
- Multiple unbounded caches growing indefinitely
- Global event listeners accumulating without cleanup
- LSP file storage storing full contents indefinitely
- Provider SDK connection pools without limits

This plan outlines a complete recreation of the CLI in Go, focusing on:
1. **Memory efficiency** - bounded caches, proper cleanup, explicit resource management
2. **Performance** - compiled binary, no runtime overhead
3. **Simplicity** - cleaner architecture without Effect.ts complexity
4. **Maintainability** - explicit resource lifecycle management

---

## Part 1: Current Architecture Analysis

### Core Components (TypeScript)

| Component | Files | Purpose | Memory Impact |
|-----------|-------|---------|---------------|
| **CLI Entry** | `src/index.ts`, `src/cli/cmd/*.ts` | Command parsing with yargs | Low |
| **Instance Manager** | `src/project/instance.ts` | Project/directory context | HIGH - unbounded cache |
| **Bus System** | `src/bus/*.ts` | Event pub/sub | CRITICAL - listener accumulation |
| **Effect Runtime** | `src/effect/*.ts` | Service layer composition | HIGH - memoization |
| **LSP Client** | `src/lsp/*.ts` | Language server protocol | CRITICAL - file text storage |
| **Provider System** | `src/provider/*.ts` | AI SDK integration | HIGH - SDK caches |
| **MCP Server** | `src/mcp/*.ts` | Model Context Protocol | HIGH - OAuth transports |
| **Session Manager** | `src/session/*.ts` | Conversation sessions | MEDIUM |
| **PTY Manager** | `src/pty/*.ts` | Terminal sessions | HIGH - 2MB buffers |
| **Storage** | `src/storage/*.ts` | SQLite + JSON storage | MEDIUM |
| **Config** | `src/config/*.ts` | Configuration loading | MEDIUM |
| **Server** | `src/server/*.ts` | HTTP/WebSocket server | MEDIUM |
| **Sync System** | `src/sync/*.ts` | Event synchronization | HIGH - registry maps |
| **Agent System** | `src/agent/*.ts` | AI agent definitions | Low |
| **Tool System** | `src/tool/*.ts` | Tool implementations | Low |
| **Snapshot** | `src/snapshot/*.ts` | Git snapshots | MEDIUM - semaphore locks |

### Key Memory Leak Sources (from MEMORY_LEAK_REVIEW.md)

| Issue | Location | Severity | Root Cause |
|-------|----------|----------|------------|
| GlobalBus EventEmitter | `src/bus/global.ts` | CRITICAL | No cleanup mechanism for listeners |
| BusEvent Registry | `src/bus/bus-event.ts` | CRITICAL | Unbounded Map for event definitions |
| InstanceState Cache | `src/effect/instance-state.ts` | CRITICAL | `capacity: Number.POSITIVE_INFINITY` |
| Instance Cache | `src/project/instance.ts` | CRITICAL | Unbounded Map by directory |
| LSP File Maps | `src/lsp/client.ts` | CRITICAL | Stores full file text + diagnostics |
| Provider SDK Cache | `src/provider/provider.ts` | HIGH | 20+ AI SDKs with internal caches |
| PTY Buffers | `src/pty/index.ts` | HIGH | 2MB buffer per session |
| OAuth Transports | `src/mcp/index.ts` | HIGH | Pending OAuth flows persist |
| Sync Registry | `src/sync/index.ts` | HIGH | Global registry never cleared |

---

## Part 2: Go Architecture Design

### Design Principles

1. **Explicit Resource Management**
   - Every resource has an owner
   - Cleanup is deterministic (defer patterns)
   - Context-based cancellation

2. **Bounded Caches**
   - All caches have maximum capacity
   - LRU eviction policies
   - TTL for stale entries

3. **No Global Singletons**
   - All state is instance-scoped
   - Dependency injection via interfaces
   - Factory patterns for stateful components

4. **Channel-Based Event System**
   - Replace EventEmitter with Go channels
   - Subscriber goroutines with graceful shutdown
   - Backpressure handling

5. **Zero-Copy Where Possible**
   - File reading without storing full content
   - Streaming diagnostics
   - Buffer pooling

### Proposed Folder Structure

```
opencode-go/
├── cmd/
│   └── opencode/
│       └── main.go                 # CLI entry point
│       └── commands/
│           ├── root.go             # Root command (cobra)
│           ├── run.go              # Main TUI command
│           ├── serve.go            # HTTP server command
│           ├── session.go          # Session management
│           ├── providers.go        # Provider listing
│           ├── models.go           # Model listing
│           ├── mcp.go              # MCP management
│           ├── auth.go             # Authentication
│           ├── config.go           # Configuration
│           ├── debug.go            # Debug commands
│           ├── export.go           # Export sessions
│           ├── import.go           # Import sessions
│           ├── upgrade.go          # Self-upgrade
│           ├── pr.go               # PR creation
│           └── github.go           # GitHub integration
│
├── internal/
│   ├── instance/
│   │   ├── instance.go             # Instance manager (bounded cache)
│   │   ├── context.go              # Instance context (ALS equivalent)
│   │   └── registry.go             # Instance registry with cleanup
│   │
│   ├── bus/
│   │   ├── bus.go                  # Pub/sub with channels
│   │   ├── event.go                # Event definitions
│   │   ├── subscriber.go           # Subscriber management
│   │   └── global.go               # Global bus (with cleanup)
│   │
│   ├── lsp/
│   │   ├── client.go               # LSP client with bounded file tracking
│   │   ├── server.go               # LSP server spawning
│   │   ├── diagnostics.go          # Diagnostic streaming
│   │   ├── language.go             # Language mappings
│   │   └── process.go              # Process management
│   │
│   ├── provider/
│   │   ├── provider.go             # Provider interface
│   │   ├── registry.go             # Provider registry
│   │   ├── model.go                # Model definitions
│   │   ├── auth.go                 # Provider authentication
│   │   ├── client.go               # HTTP client with connection pooling
│   │   └── implementations/
│   │       ├── anthropic.go
│   │       ├── openai.go
│   │       ├── bedrock.go
│   │       ├── azure.go
│   │       ├── google.go
│   │       ├── groq.go
│   │       ├── openrouter.go
│   │       └── ... (other providers)
│   │
│   ├── mcp/
│   │   ├── client.go               # MCP client with cleanup
│   │   ├── transport.go            # Transport implementations
│   │   ├── oauth.go                # OAuth with timeout cleanup
│   │   ├── tools.go                # Tool conversion
│   │   └── prompts.go              # Prompt handling
│   │
│   ├── session/
│   │   ├── session.go              # Session manager
│   │   ├── message.go              # Message handling
│   │   ├── part.go                 # Message parts
│   │   ├── llm.go                  # LLM integration
│   │   ├── processor.go            # Message processing
│   │   ├── compaction.go           # Context compaction
│   │   └── prompt.go               # Prompt construction
│   │
│   ├── pty/
│   │   ├── pty.go                  # PTY manager with session limits
│   │   ├── session.go              # PTY session with bounded buffer
│   │   ├── process.go              # Process spawning
│   │   └── websocket.go            # WebSocket handling
│   │
│   ├── storage/
│   │   ├── database.go             # SQLite database
│   │   ├── schema.go               # Schema definitions
│   │   ├── session.go              # Session storage
│   │   ├── message.go              # Message storage
│   │   ├── project.go              # Project storage
│   │   └── migration.go            # Schema migrations
│   │
│   ├── config/
│   │   ├── config.go               # Configuration loading
│   │   ├── loader.go               # File loading
│   │   ├── parser.go               # JSON/JSONC parsing
│   │   ├── paths.go                # Config paths
│   │   ├── provider.go             # Provider config
│   │   ├── agent.go                # Agent config
│   │   ├── mcp.go                  # MCP config
│   │   └── permission.go           # Permission config
│   │
│   ├── server/
│   │   ├── server.go               # HTTP server (net/http or fiber)
│   │   ├── router.go               # Route definitions
│   │   ├── middleware.go           # Middleware
│   │   ├── websocket.go            # WebSocket handling
│   │   ├── sse.go                  # Server-sent events
│   │   └── routes/
│   │       ├── instance.go
│   │       ├── session.go
│   │       ├── global.go
│   │       └── workspace.go
│   │
│   ├── sync/
│   │   ├── sync.go                 # Event sync system
│   │   ├── projector.go            # Projector functions
│   │   ├── event.go                # Event types
│   │   └ registry.go               # Bounded registry
│   │
│   ├── agent/
│   │   ├── agent.go                # Agent definitions
│   │   ├── build.go                # Build agent
│   │   ├── plan.go                 # Plan agent
│   │   ├── explore.go              # Explore agent
│   │   ├── generate.go             # Agent generation
│   │
│   ├── tool/
│   │   ├── tool.go                 # Tool interface
│   │   ├── registry.go             # Tool registry
│   │   ├── bash.go
│   │   ├── read.go
│   │   ├── write.go
│   │   ├── edit.go
│   │   ├── grep.go
│   │   ├── glob.go
│   │   ├── webfetch.go
│   │   ├── websearch.go
│   │   ├── lsp.go
│   │   └── ... (other tools)
│   │
│   ├── snapshot/
│   │   ├── snapshot.go             # Git snapshot
│   │   ├── diff.go                 # Diff handling
│   │   └── lock.go                 # Bounded lock map
│   │
│   ├── file/
│   │   ├── filesystem.go           # File operations
│   │   ├── watcher.go              # File watching
│   │   ├── ignore.go               # Ignore patterns
│   │   ├── ripgrep.go              # Ripgrep integration
│   │
│   ├── project/
│   │   ├── project.go              # Project info
│   │   ├── vcs.go                  # VCS detection
│   │   ├── bootstrap.go            # Project bootstrap
│   │
│   ├── permission/
│   │   ├── permission.go           # Permission system
│   │   ├── ruleset.go              # Ruleset definitions
│   │   ├── check.go                # Permission checking
│   │
│   ├── auth/
│   │   ├── auth.go                 # Authentication storage
│   │   ├── provider.go             # Provider auth
│   │
│   ├── plugin/
│   │   ├── plugin.go               # Plugin system
│   │   ├── loader.go               # Plugin loading
│   │
│   ├── skill/
│   │   ├── skill.go                # Skill system
│   │   ├── discovery.go            # Skill discovery
│   │
│   ├── util/
│   │   ├── log.go                  # Logging
│   │   ├── error.go                # Error handling
│   │   ├── queue.go                # Async queue with timeout
│   │   ├── timeout.go              # Timeout helpers
│   │   ├── process.go              # Process utilities
│   │   ├── hash.go                 # Hash utilities
│   │   ├── path.go                 # Path utilities
│   │   ├── lock.go                 # Lock utilities
│   │   └── network.go              # Network utilities
│   │
│   ├── global/
│   │   ├── global.go               # Global paths and constants
│   │
│   ├── installation/
│   │   ├── version.go              # Version info
│   │   ├── install.go              # Installation detection
│   │
│   └── tui/
│       ├── tui.go                  # TUI entry point
│       ├── app.go                  # TUI application
│       ├── components/
│       │   ├── prompt.go
│       │   ├── message.go
│       │   ├── sidebar.go
│       │   ├── status.go
│       │   └── terminal.go
│       └── events/
│           └── events.go
│
├── pkg/
│   ├── api/
│   │   ├── client.go               # Public API client
│   │   └── types.go                # Public API types
│   │
│   └ models/
│   │   ├── session.go              # Session models
│   │   ├── message.go              # Message models
│   │   ├── provider.go             # Provider models
│   │   └ config.go                 # Config models
│   │
│   └── sdk/
│       └── opencode.go             # SDK for external use
│
├── api/
│   └ openapi.json                  # OpenAPI specification
│
├── configs/
│   └ default.json                  # Default configuration
│
├── scripts/
│   ├── build.sh                    # Build script
│   └ release.sh                    # Release script
│
├── go.mod
├── go.sum
├── Makefile
├── README.md
└ CLAUDE.md                         # Claude Code instructions
```

---

## Part 3: Memory Leak Solutions (Go Implementation)

### 1. GlobalBus EventEmitter → Channel-Based Bus

**Problem:** EventEmitter listeners accumulate without cleanup

**Go Solution:**
```go
// internal/bus/bus.go
type Bus struct {
    subscribers map[string][]chan Event
    wildcard    []chan Event
    mu          sync.RWMutex
    maxSubs     int // Bounded subscriber count
    ctx         context.Context
    cancel      context.CancelFunc
}

func (b *Bus) Subscribe(eventType string) <-chan Event {
    b.mu.Lock()
    defer b.mu.Unlock()
    
    // Check max subscribers
    if len(b.subscribers[eventType]) >= b.maxSubs {
        // Evict oldest subscriber
        oldest := b.subscribers[eventType][0]
        close(oldest)
        b.subscribers[eventType] = b.subscribers[eventType][1:]
    }
    
    ch := make(chan Event, 100) // Buffered channel
    b.subscribers[eventType] = append(b.subscribers[eventType], ch)
    return ch
}

func (b *Bus) Close() {
    b.cancel()
    b.mu.Lock()
    defer b.mu.Unlock()
    
    // Close all subscriber channels
    for _, subs := range b.subscribers {
        for _, ch := range subs {
            close(ch)
        }
    }
    for _, ch := range b.wildcard {
        close(ch)
    }
    b.subscribers = nil
    b.wildcard = nil
}
```

### 2. BusEvent Registry → Bounded Registry

**Problem:** Registry Map grows unbounded

**Go Solution:**
```go
// internal/bus/registry.go
type Registry struct {
    definitions map[string]Definition
    mu          sync.RWMutex
    maxSize     int // Maximum definitions
}

func (r *Registry) Register(def Definition) error {
    r.mu.Lock()
    defer r.mu.Unlock()
    
    if len(r.definitions) >= r.maxSize {
        return ErrRegistryFull
    }
    
    r.definitions[def.Type] = def
    return nil
}

func (r *Registry) Clear() {
    r.mu.Lock()
    defer r.mu.Unlock()
    r.definitions = make(map[string]Definition)
}
```

### 3. InstanceState Cache → Bounded Cache with LRU

**Problem:** `capacity: Number.POSITIVE_INFINITY`

**Go Solution:**
```go
// internal/instance/cache.go
type InstanceCache struct {
    cache    map[string]*Instance
    lru      *list.List // LRU tracking
    maxSize  int        // Maximum instances (e.g., 10)
    ttl      time.Duration
    mu       sync.RWMutex
}

func (c *InstanceCache) Get(directory string) (*Instance, error) {
    c.mu.Lock()
    defer c.mu.Unlock()
    
    if inst, ok := c.cache[directory]; ok {
        // Move to front of LRU
        c.lru.MoveToFront(inst.element)
        inst.lastAccess = time.Now()
        return inst, nil
    }
    
    return nil, ErrNotFound
}

func (c *InstanceCache) Put(directory string, inst *Instance) error {
    c.mu.Lock()
    defer c.mu.Unlock()
    
    // Evict if over capacity
    if len(c.cache) >= c.maxSize {
        oldest := c.lru.Back()
        if oldest != nil {
            oldInst := oldest.Value.(*Instance)
            oldInst.Close() // Cleanup resources
            delete(c.cache, oldInst.directory)
            c.lru.Remove(oldest)
        }
    }
    
    c.cache[directory] = inst
    inst.element = c.lru.PushFront(inst)
    return nil
}

func (c *InstanceCache) CleanupStale() {
    c.mu.Lock()
    defer c.mu.Unlock()
    
    now := time.Now()
    for dir, inst := range c.cache {
        if now.Sub(inst.lastAccess) > c.ttl {
            inst.Close()
            c.lru.Remove(inst.element)
            delete(c.cache, dir)
        }
    }
}
```

### 4. LSP Client → Streaming with File Limits

**Problem:** Stores full file text indefinitely

**Go Solution:**
```go
// internal/lsp/client.go
type LSPClient struct {
    // Bounded file tracking
    files      *BoundedMap[string, *FileState] // Max 50 files
    diagnostics *BoundedMap[string, []Diagnostic]
    
    // Streaming - don't store full text
    streamDiagnostics bool
}

type FileState struct {
    version  int
    size     int64
    // Don't store text - only track version/size
    // Text is streamed on demand
}

type BoundedMap[K, V] struct {
    data    map[K]V
    maxSize int
    mu      sync.RWMutex
}

func (bm *BoundedMap[K, V]) Set(key K, value V) {
    bm.mu.Lock()
    defer bm.mu.Unlock()
    
    if len(bm.data) >= bm.maxSize {
        // Evict oldest entry
        // (use separate LRU tracking)
    }
    bm.data[key] = value
}
```

### 5. Provider SDK → Lazy Loading + Connection Limits

**Problem:** 20+ SDKs loaded at startup with internal caches

**Go Solution:**
```go
// internal/provider/registry.go
type ProviderRegistry struct {
    providers  map[string]*Provider
    clients    *ConnectionPool // Shared HTTP client pool
    lazyLoad   bool            // Load providers on demand
    mu         sync.RWMutex
}

type ConnectionPool struct {
    maxConnsPerHost int
    idleTimeout     time.Duration
    client          *http.Client
}

func NewConnectionPool() *ConnectionPool {
    return &ConnectionPool{
        maxConnsPerHost: 10,
        idleTimeout:     30 * time.Second,
        client: &http.Client{
            Transport: &http.Transport{
                MaxIdleConnsPerHost: 10,
                IdleConnTimeout:     30 * time.Second,
            },
        },
    }
}

func (r *ProviderRegistry) GetProvider(id string) (*Provider, error) {
    r.mu.RLock()
    if p, ok := r.providers[id]; ok {
        r.mu.RUnlock()
        return p, nil
    }
    r.mu.RUnlock()
    
    // Lazy load
    r.mu.Lock()
    defer r.mu.Unlock()
    
    p, err := r.loadProvider(id)
    if err != nil {
        return nil, err
    }
    r.providers[id] = p
    return p, nil
}
```

### 6. PTY Sessions → Bounded Sessions + Buffer Pooling

**Problem:** 2MB buffer per session, unlimited sessions

**Go Solution:**
```go
// internal/pty/manager.go
type PTYManager struct {
    sessions   map[string]*PTYSession
    maxSessions int              // e.g., 5 concurrent
    bufferPool  *sync.Pool       // Reuse buffers
    mu          sync.RWMutex
}

func NewPTYManager() *PTYManager {
    return &PTYManager{
        maxSessions: 5,
        bufferPool: &sync.Pool{
            New: func() interface{} {
                return make([]byte, 64*1024) // 64KB chunks, not 2MB
            },
        },
    }
}

func (m *PTYManager) Create(input CreateInput) (*PTYSession, error) {
    m.mu.Lock()
    defer m.mu.Unlock()
    
    // Check max sessions
    if len(m.sessions) >= m.maxSessions {
        // Evict oldest idle session
        for id, s := range m.sessions {
            if s.IsIdle() {
                s.Close()
                delete(m.sessions, id)
                break
            }
        }
    }
    
    session := &PTYSession{
        buffer: m.bufferPool.Get().([]byte),
    }
    m.sessions[input.ID] = session
    return session, nil
}

func (s *PTYSession) Close() {
    // Return buffer to pool
    s.manager.bufferPool.Put(s.buffer)
    // ... cleanup process
}
```

### 7. MCP OAuth → Timeout Cleanup

**Problem:** OAuth transports persist indefinitely

**Go Solution:**
```go
// internal/mcp/oauth.go
type OAuthManager struct {
    pending    map[string]*OAuthFlow
    timeout    time.Duration // 10 minute timeout
    mu         sync.RWMutex
}

type OAuthFlow struct {
    transport  Transport
    startTime  time.Time
    timer      *time.Timer
}

func (m *OAuthManager) StartFlow(name string) (*OAuthFlow, error) {
    m.mu.Lock()
    defer m.mu.Unlock()
    
    flow := &OAuthFlow{
        startTime: time.Now(),
    }
    
    // Auto cleanup after timeout
    flow.timer = time.AfterFunc(m.timeout, func() {
        m.CancelFlow(name)
    })
    
    m.pending[name] = flow
    return flow, nil
}

func (m *OAuthManager) CancelFlow(name string) {
    m.mu.Lock()
    defer m.mu.Unlock()
    
    if flow, ok := m.pending[name]; ok {
        flow.timer.Stop()
        flow.transport.Close()
        delete(m.pending, name)
    }
}
```

### 8. Sync Registry → Instance-Scoped Registry

**Problem:** Global registry persists across instances

**Go Solution:**
```go
// internal/sync/registry.go
// Registry is per-instance, not global
type SyncRegistry struct {
    definitions map[string]Definition
    projectors  map[string]ProjectorFunc
    versions    map[string]int
    frozen      bool
    
    // Instance-scoped, cleared on dispose
}

func (r *SyncRegistry) Reset() {
    r.definitions = make(map[string]Definition)
    r.projectors = make(map[string]ProjectorFunc)
    r.versions = make(map[string]int)
    r.frozen = false
}
```

---

## Part 4: Key Design Decisions

### CLI Framework: Cobra vs Custom

**Recommendation:** Use Cobra

- Mature, well-documented
- Built-in help generation
- Shell completion support
- Subcommand nesting

```go
// cmd/opencode/main.go
func main() {
    rootCmd := commands.NewRootCommand()
    if err := rootCmd.Execute(); err != nil {
        fmt.Fprintln(os.Stderr, err)
        os.Exit(1)
    }
}
```

### HTTP Server: net/http vs Fiber vs Gin

**Recommendation:** Use Fiber

- Fast (similar to Hono in TypeScript)
- Express-like API
- WebSocket support
- SSE support

```go
// internal/server/server.go
func NewServer() *fiber.App {
    app := fiber.New(fiber.Config{
        BodyLimit: 10 * 1024 * 1024,
    })
    
    app.Use(middleware.Logger())
    app.Use(middleware.Compress())
    app.Use(middleware.CORS())
    
    // Routes...
    return app
}
```

### Database: SQLite via modernc.org/sqlite

**Recommendation:** Use modernc.org/sqlite

- Pure Go (no CGO)
- Cross-platform
- Compatible with SQLite format

```go
// internal/storage/database.go
import "modernc.org/sqlite"

func NewDatabase(path string) (*sql.DB, error) {
    db, err := sqlite.Open(path)
    // ...
    return db, nil
}
```

### LSP Communication: JSON-RPC 2.0

**Recommendation:** Custom implementation

```go
// internal/lsp/jsonrpc.go
type JSONRPCConn struct {
    stdin  io.Writer
    stdout io.Reader
    stderr io.Reader
    
    requests  map[int]chan Response
    mu        sync.RWMutex
}

func (c *JSONRPCConn) SendRequest(method string, params any) (Response, error) {
    id := c.nextID()
    req := Request{
        JSONRPC: "2.0",
        ID:      id,
        Method:  method,
        Params:  params,
    }
    
    ch := make(chan Response, 1)
    c.mu.Lock()
    c.requests[id] = ch
    c.mu.Unlock()
    
    // Send and wait for response with timeout
    // ...
}
```

### AI Provider Integration

**Recommendation:** Direct HTTP API calls

- No SDK dependencies (reduces memory)
- Shared HTTP client pool
- Streaming response handling

```go
// internal/provider/anthropic.go
func (p *AnthropicProvider) StreamChat(ctx context.Context, req ChatRequest) (<-chan ChatChunk, error) {
    // Direct HTTP call to Anthropic API
    // Stream response via channel
}
```

### Configuration: JSON/JSONC

**Recommendation:** Use github.com/tidwall/jsonc

```go
// internal/config/parser.go
import "github.com/tidwall/jsonc"

func ParseConfig(path string) (*Config, error) {
    data, err := os.ReadFile(path)
    if err != nil {
        return nil, err
    }
    
    // Convert JSONC to standard JSON
    json := jsonc.ToJSON(data)
    
    var cfg Config
    if err := json.Unmarshal(json, &cfg); err != nil {
        return nil, err
    }
    
    return &cfg, nil
}
```

### TUI Framework

**Recommendation:** Use Bubble Tea (charmbracelet/bubbletea)

- Composable components
- Event-based architecture
- Cross-platform terminal support

```go
// internal/tui/app.go
import "github.com/charmbracelet/bubbletea"

type App struct {
    session    *Session
    prompt     PromptComponent
    messages   MessagesComponent
    sidebar    SidebarComponent
}

func (a *App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case KeyMsg:
        // Handle key input
    case StreamMsg:
        // Handle streaming response
    }
    return a, nil
}

func (a *App) View() string {
    return lipgloss.JoinVertical(
        a.sidebar.View(),
        a.messages.View(),
        a.prompt.View(),
    )
}
```

---

## Part 5: Implementation Phases

### Phase 1: Core Infrastructure (Week 1-2)

**Priority:** Foundation for everything else

| Task | Description | Estimated Time |
|------|-------------|----------------|
| Project setup | go.mod, folder structure, Makefile | 1 day |
| CLI entry point | Cobra root command, argument parsing | 1 day |
| Logging system | Structured logging with levels | 1 day |
| Global paths | XDG paths, installation detection | 1 day |
| Configuration | JSON/JSONC parsing, merging | 3 days |
| Instance manager | Bounded cache, context propagation | 3 days |
| Bus system | Channel-based pub/sub | 3 days |

**Key Deliverables:**
- `opencode --help` works
- `opencode config` commands work
- Instance manager with bounded cache
- Bus with proper cleanup

### Phase 2: Storage & Session (Week 3-4)

**Priority:** Data persistence layer

| Task | Description | Estimated Time |
|------|-------------|----------------|
| SQLite setup | modernc.org/sqlite, schema | 2 days |
| Schema definitions | Tables for sessions, messages, projects | 2 days |
| Migration system | Schema migrations | 2 days |
| Session manager | CRUD operations | 3 days |
| Message handling | Message/part storage | 3 days |
| Sync system | Event synchronization | 3 days |

**Key Deliverables:**
- Database works with SQLite
- Sessions can be created/listed/deleted
- Messages persist properly
- Sync events work with bounded registry

### Phase 3: AI Integration (Week 5-6)

**Priority:** Core functionality

| Task | Description | Estimated Time |
|------|-------------|----------------|
| Provider interface | Generic provider abstraction | 2 days |
| Anthropic implementation | Streaming chat API | 3 days |
| OpenAI implementation | Chat/completions API | 3 days |
| Model registry | models.dev integration | 2 days |
| Authentication | API key storage | 2 days |

**Key Deliverables:**
- Can chat with AI providers
- Streaming responses work
- Anthropic + OpenAI providers available
- Authentication storage

### Phase 4: Tools & Agent (Week 7-8)

**Priority:** Agent execution

| Task | Description | Estimated Time |
|------|-------------|----------------|
| Tool interface | Generic tool definition | 1 day |
| Bash tool | Command execution | 2 days |
| Read tool | File reading | 1 day |
| Write/Edit tools | File modification | 2 days |
| Grep/Glob tools | Search tools | 2 days |
| Web tools | WebFetch, WebSearch | 2 days |
| LSP tool | Diagnostics integration | 3 days |
| Agent system | Agent definitions, permissions | 3 days |
| Permission system | Ruleset checking | 2 days |

**Key Deliverables:**
- Tools execute properly
- Permission system works
- Agents configured correctly

### Phase 5: LSP Integration (Week 9-10)

**Priority:** External integrations (MCP skipped per user decision)

| Task | Description | Estimated Time |
|------|-------------|----------------|
| LSP client | JSON-RPC, bounded file tracking | 4 days |
| LSP server spawning | TypeScript, Python, Go, etc. | 3 days |
| Diagnostic streaming | Streaming diagnostics | 2 days |
| LSP tool | Diagnostics in agent tools | 2 days |

**Key Deliverables:**
- LSP diagnostics work
- Multiple language servers
- Bounded file tracking (max 50 files)
- Diagnostics available in tools

*(MCP integration skipped for initial version - focus on native tools)*

### Phase 6: TUI (Week 11-13) ← Moved earlier (TUI First priority)

**Priority:** User interface (Primary focus per user decision)

| Task | Description | Estimated Time |
|------|-------------|----------------|
| Bubble Tea setup | App structure | 2 days |
| Prompt component | Input handling | 3 days |
| Messages component | Streaming display | 3 days |
| Sidebar component | Session list | 2 days |
| Terminal component | PTY display | 2 days |
| Status component | Progress, errors | 2 days |
| Keybinds | Keyboard handling | 2 days |

**Key Deliverables:**
- Full TUI works
- Streaming display
- Keyboard navigation
- Session management via TUI

### Phase 7: PTY & Terminal (Week 14)

**Priority:** Terminal sessions

| Task | Description | Estimated Time |
|------|-------------|----------------|
| PTY spawning | Terminal process creation | 3 days |
| Session limits | Bounded sessions (max 5), buffer pooling | 2 days |
| WebSocket integration | Terminal over WebSocket | 2 days |

**Key Deliverables:**
- PTY sessions work
- Memory bounded
- WebSocket terminal

### Phase 8: Server & API (Week 15-16) ← Deferred after TUI

**Priority:** HTTP/WebSocket server

| Task | Description | Estimated Time |
|------|-------------|----------------|
| HTTP server | Fiber setup, middleware | 2 days |
| Instance routes | Session, message endpoints | 3 days |
| WebSocket handling | PTY WebSocket, streaming | 3 days |
| SSE handling | Server-sent events | 2 days |
| Global routes | Provider, model listing | 2 days |
| OpenAPI spec | API documentation | 1 day |

**Key Deliverables:**
- HTTP server works
- WebSocket for PTY
- SSE for streaming
- API documented

### Phase 9: Polish & Testing (Week 17-18)

**Priority:** Quality assurance

| Task | Description | Estimated Time |
|------|-------------|----------------|
| Memory testing | Leak detection, benchmarks | 3 days |
| Integration tests | End-to-end tests | 3 days |
| Documentation | README, help text | 2 days |
| Build optimization | Binary size, cross-compilation | 2 days |

**Key Deliverables:**
- Memory usage < 500MB
- All tests pass
- Documentation complete
- Release binaries

---

## Revised Implementation Timeline (Based on User Decisions)

| Phase | Focus | Duration | Adjustments |
|-------|-------|----------|-------------|
| 1 | Core Infrastructure | Week 1-2 | No change |
| 2 | Storage & Session | Week 3-4 | All tables compatible |
| 3 | AI Integration | Week 5-6 | Anthropic + OpenAI only |
| 4 | Tools & Agent | Week 7-8 | No change |
| 5 | LSP Integration | Week 9-10 | MCP skipped |
| 6 | TUI | Week 11-13 | Moved earlier (TUI First) |
| 7 | PTY & Terminal | Week 14 | No change |
| 8 | Server & API | Week 15-16 | Deferred after TUI |
| 9 | Polish & Testing | Week 17-18 | No change |

**Total Duration: 18 weeks**

---

## Part 6: Improvements Over Existing Code

### Architecture Improvements

| Area | Current (TS) | Improved (Go) |
|------|--------------|---------------|
| Runtime | Bun/V8 runtime overhead | Compiled binary, no runtime |
| Memory | 71GB+ virtual memory | Target < 500MB RSS |
| Caches | Unbounded, infinite capacity | Bounded with LRU eviction |
| Cleanup | Effect.Scope complexity | defer patterns, context cancellation |
| Events | EventEmitter global | Instance-scoped channels |
| State | AsyncLocalStorage | Context.Value or explicit passing |

### Performance Improvements

| Area | Current (TS) | Improved (Go) |
|------|--------------|---------------|
| Startup | Bun module loading | Instant binary startup |
| File I/O | Bun APIs | Native Go I/O, zero-copy where possible |
| HTTP | Hono + adapter | Fiber, optimized routing |
| JSON | V8 JSON parsing | encoding/json or json-iterator |
| Streaming | Effect.Stream | Go channels, native streaming |

### Code Quality Improvements

| Area | Current (TS) | Improved (Go) |
|------|--------------|---------------|
| Type safety | TypeScript inference | Go explicit types |
| Error handling | Effect error channels | Go error patterns |
| Dependency management | npm/bun packages | Go modules |
| Testing | Bun test | Go test, coverage built-in |
| Documentation | JSDoc | Go doc comments |

### Memory Leak Fixes

| Issue | Current Fix Needed | Go Prevention |
|-------|--------------------|---------------|
| GlobalBus listeners | Track and cleanup | Channel-based, close on shutdown |
| BusEvent registry | clearRegistry() | Instance-scoped, cleared on dispose |
| InstanceState cache | Set capacity limit | Bounded cache with LRU |
| LSP file storage | File close mechanism | BoundedMap, don't store text |
| Provider SDKs | Explicit cleanup | Lazy load, shared connection pool |
| PTY buffers | Session limits | Pool buffers, limit sessions |
| OAuth transports | Timeout cleanup | Timer-based auto cleanup |
| Sync registry | reset() function | Instance-scoped by default |

---

## Part 7: Key Questions to Clarify

Before proceeding, I need clarification on:

1. **Provider Priority**
   - Which providers are most critical?
   - Should we start with Anthropic/OpenAI only?

2. **TUI vs Server Priority**
   - Is TUI the primary interface?
   - Or should we prioritize HTTP API first?

3. **Database Compatibility**
   - Must we support the existing SQLite schema?
   - Or can we design a new schema?

4. **Backward Compatibility**
   - Must sessions/config be compatible with TS version?
   - Or can we migrate?

5. **Feature Scope**
   - Full feature parity or MVP first?
   - Which features can be deferred?

6. **Release Strategy**
   - Single binary or packages?
   - Cross-platform targets (Linux, macOS, Windows)?

---

## Part 8: Dependencies (Go)

### Core Dependencies

| Package | Purpose | Version |
|---------|---------|---------|
| `github.com/spf13/cobra` | CLI framework | v1.8.x |
| `github.com/spf13/viper` | Configuration | v1.18.x |
| `modernc.org/sqlite` | SQLite (pure Go) | v1.29.x |
| `github.com/tidwall/jsonc` | JSONC parsing | v0.3.x |
| `github.com/charmbracelet/bubbletea` | TUI framework | v0.25.x |
| `github.com/charmbracelet/lipgloss` | TUI styling | v0.9.x |
| `github.com/gofiber/fiber/v3` | HTTP server | v3.x |
| `github.com/gofiber/websocket/v3` | WebSocket | v3.x |
| `github.com/go-playground/validator` | Schema validation | v10.x |
| `github.com/charmbracelet/bubbles` | TUI components | v0.18.x |
| `github.com/creack/pty` | PTY support | v1.1.x |
| `github.com/google/uuid` | UUID generation | v1.6.x |
| `github.com/mattn/go-isatty` | Terminal detection | v0.0.x |
| `github.com/rivo/uniseg` | Unicode text | v0.4.x |

### Optional Dependencies

| Package | Purpose | Notes |
|---------|---------|-------|
| `github.com/aws/aws-sdk-go-v2` | AWS Bedrock | If Bedrock priority |
| `github.com/Azure/azure-sdk-for-go` | Azure OpenAI | If Azure priority |
| `github.com/googleapis/google-cloud-go` | Google Vertex | If Google priority |
| `golang.org/x/crypto` | OAuth, hashing | For OAuth flows |
| `golang.org/x/term` | Terminal handling | Alternative to pty |

---

## Part 9: Success Metrics

### Memory Targets

| Metric | Current | Target |
|--------|---------|--------|
| Virtual memory (startup) | 71GB+ | < 100MB |
| RSS (steady state) | Unknown | < 500MB |
| RSS (active session) | Growing | < 1GB |
| Instance cache size | Unlimited | Max 10 instances |
| LSP file tracking | Unlimited | Max 50 files |
| PTY sessions | Unlimited | Max 5 concurrent |

### Performance Targets

| Metric | Current | Target |
|--------|---------|--------|
| Startup time | Bun loading | < 100ms |
| Session creation | Unknown | < 50ms |
| Message streaming | Effect overhead | Direct channel streaming |
| HTTP latency | Hono + adapter | < 10ms overhead |

---

## Appendix A: File-by-File Review Summary

### Critical Files (Memory Leak Focus)

| File | Issues | Go Solution |
|------|--------|-------------|
| `src/bus/global.ts` | EventEmitter no cleanup | Channel-based bus |
| `src/bus/bus-event.ts` | Unbounded registry | BoundedMap with Clear() |
| `src/effect/instance-state.ts` | Infinite capacity | LRU cache with TTL |
| `src/project/instance.ts` | Unbounded cache | InstanceCache with limits |
| `src/lsp/client.ts` | Full text storage | BoundedMap, streaming |
| `src/provider/provider.ts` | SDK accumulation | Lazy load, connection pool |
| `src/pty/index.ts` | 2MB buffers | Buffer pool, session limits |
| `src/mcp/index.ts` | OAuth persist | Timer-based cleanup |
| `src/sync/index.ts` | Global registry | Instance-scoped |

### Supporting Files

| File | Purpose | Go Equivalent |
|------|---------|---------------|
| `src/util/log.ts` | Logging | `internal/util/log.go` |
| `src/util/queue.ts` | Async queue | Channel with timeout |
| `src/util/process.ts` | Process utils | `os/exec` package |
| `src/util/filesystem.ts` | File ops | `io/fs` package |
| `src/util/error.ts` | Error handling | Go error types |
| `src/config/paths.ts` | Config paths | `internal/config/paths.go` |
| `src/config/parse.ts` | JSON parsing | `encoding/json` + jsonc |
| `src/storage/db.ts` | SQLite | `modernc.org/sqlite` |
| `src/storage/schema.ts` | Schema | SQL schema definitions |

---

## Appendix B: Questions for User

Please answer these before I proceed with implementation:

1. Should I start with a minimal CLI that just handles config/session management, then add features incrementally?

2. Which providers are essential for initial version? (Anthropic + OpenAI as minimum?)

3. Is backward compatibility with existing sessions/config required?

4. Should TUI use Bubble Tea or should we prioritize HTTP server first?

5. Any specific features to exclude from initial Go version?

6. Preferred release format: single binary, or separate packages?

---

**Next Steps:**
- Answer questions above
- Begin Phase 1 implementation
- Create initial Go project structure