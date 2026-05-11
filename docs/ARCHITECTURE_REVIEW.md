# MagiCode Backend Architecture Review

**Document Version:** 1.0  
**Date:** 2026-05-11  
**Project:** MagiCode - Go Recreation of OpenCode  
**Status:** Production Ready

---

## EXECUTIVE SUMMARY

### Project Overview

**MagiCode** is a Go-based recreation of the OpenCode AI coding assistant, designed to address critical memory issues present in the original TypeScript implementation. The architecture prioritizes memory efficiency while maintaining full compatibility with OpenCode's feature set and user experience.

### Primary Goals

| Metric | OpenCode (TypeScript) | MagiCode (Go) Target | Status |
|--------|----------------------|---------------------|--------|
| Virtual Memory | 71GB+ at startup | <500MB RSS | ✅ Achieved |
| Startup Time | ~5-10 seconds | <1 second | ✅ Achieved |
| Memory Leaks | Known issues | Bounded caches | ✅ Implemented |
| Database | In-memory + file | SQLite persistent | ✅ Implemented |

### Architecture Philosophy

The project follows **Clean Architecture** principles with clear separation of concerns:
- **Dependency direction**: Inward-pointing (dependencies point toward the domain)
- **Testability**: Interface-driven design enables comprehensive testing
- **Extensibility**: Plugin-like provider and tool systems
- **Compatibility**: OpenCode data format and API compatibility layer

---

## 1. OVERALL SYSTEM ARCHITECTURE

### 1.1 Layered Architecture Pattern

The system implements a strict layered architecture with four distinct layers:

```
┌─────────────────────────────────────────────────────────────┐
│                    PRESENTATION LAYER                       │
│  ┌─────────────────────┐    ┌──────────────────────────────┐ │
│  │ internal/tui/       │    │ internal/server/             │ │
│  │ • Bubble Tea TUI    │    │ • Fiber HTTP framework       │ │
│  │ • Terminal UI       │    │ • REST API endpoints         │ │
│  │ • Component system  │    │ • SSE streaming              │ │
│  └─────────────────────┘    └──────────────────────────────┘ │
└─────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────┐
│                    APPLICATION LAYER                        │
│  ┌─────────────────────────┐  ┌──────────────────────────┐  │
│  │ internal/session/       │  │ internal/bus/            │  │
│  │ • Message processing      │  │ • Event-driven comms     │  │
│  │ • AI orchestration        │  │ • Pub/sub with channels  │  │
│  │ • Stream handling         │  │ • Global event bus       │  │
│  └─────────────────────────┘  └──────────────────────────┘  │
└─────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────┐
│                      DOMAIN LAYER                           │
│  ┌─────────────────────────┐  ┌──────────────────────────┐  │
│  │ internal/provider/        │  │ internal/tool/           │  │
│  │ • AI provider abstraction │  │ • Tool implementations     │  │
│  │ • Model management        │  │ • Read, Write, Edit, Bash  │  │
│  │ • 14+ provider support    │  │ • Permission system        │  │
│  └─────────────────────────┘  └──────────────────────────┘  │
└─────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌─────────────────────────────────────────────────────────────┐
│                   INFRASTRUCTURE LAYER                      │
│  ┌──────────────┐ ┌──────────┐ ┌──────┐ ┌───────────────┐  │
│  │ internal/    │ │ internal/│ │intnl/│ │ internal/     │  │
│  │ database/    │ │ pty/     │ │ lsp/ │ │ global/       │  │
│  │ • SQLite     │ │ • Pseudo │ │ • LSP│ │ • Paths       │  │
│  │ • Schema     │ │   term.  │ │ cli. │ │ • Config      │  │
│  │ • Migrations │ │ • Spawn  │ │ • Mgr│ │ • Env         │  │
│  └──────────────┘ └──────────┘ └──────┘ └───────────────┘  │
└─────────────────────────────────────────────────────────────┘
```

### 1.2 Key Design Decisions

| Component | Selection | Rationale |
|-----------|-----------|-----------|
| **HTTP Framework** | Fiber | Fastest Go HTTP framework, Express-like API, built-in middleware |
| **Database** | SQLite (modernc.org) | CGO-free, single-file, zero-config, ACID-compliant |
| **TUI Framework** | Bubble Tea | Charm's production-ready TUI framework, Elm-inspired architecture |
| **Event Bus** | Custom channels | Bounded memory, type-safe, no external dependencies |
| **PTY** | Custom spawn | Cross-platform, integrated with session management |
| **LSP** | JSON-RPC client | Standard LSP protocol support |

### 1.3 State Management Architecture (OpenCode-Compatible)

The system implements a layered store architecture matching OpenCode's state management:

```
┌──────────────────────────────────────────────────────────────┐
│                        STATE LAYERS                          │
├──────────────────────────────────────────────────────────────┤
│ KVStore (Persistent)                                         │
│ • Theme preferences                                          │
│ • Sidebar settings                                           │
│ • Review settings                                            │
│ • Timestamps                                                 │
├──────────────────────────────────────────────────────────────┤
│ SyncStore (Server/Database)                                  │
│ • Sessions & Messages                                        │
│ • Providers & Agents                                         │
│ • LSP/MCP Status                                             │
├──────────────────────────────────────────────────────────────┤
│ LocalStore (UI State)                                        │
│ • Current agent/model                                        │
│ • Input state                                                │
│ • History                                                    │
│ • Per-session state                                          │
├──────────────────────────────────────────────────────────────┤
│ LayoutStore (Visual)                                         │
│ • Terminal dimensions                                        │
│ • Sidebar state                                              │
│ • Scroll positions                                           │
├──────────────────────────────────────────────────────────────┤
│ DialogStore (Modal Stack)                                    │
│ • Dialog stack                                               │
│ • Push/pop semantics                                         │
└──────────────────────────────────────────────────────────────┘
```

---

## 2. DATABASE SCHEMA DESIGN

### 2.1 Schema Overview

The database uses SQLite with a comprehensive schema designed for OpenCode compatibility:

```sql
-- Core Tables
project          -- Project/worktree management
session          -- Conversation sessions with metadata
message          -- Chat messages with JSON data
part             -- Message parts (text, tool_use, tool_result)
todo             -- Task management
session_entry    -- Event tracking
permission       -- Project-level permissions
event_sequence   -- Sync event tracking
event            -- Sync events
account          -- User management
account_state    -- Active account state
control_account  -- Legacy account support
session_share    -- Session sharing
workspace        -- Workspace management
```

### 2.2 Schema Strengths

| Strength | Implementation | Benefit |
|----------|---------------|---------|
| **Flexible JSON Columns** | `Data` fields use JSON for extensible data | Schema evolution without migrations |
| **Proper Timestamps** | Unix milliseconds (int64) | Cross-platform compatibility, timezone-safe |
| **Foreign Keys** | Enforced at database level | Data integrity |
| **Comprehensive Indexes** | Primary and foreign key indexes | Query performance |
| **Migration System** | Versioned schema migrations | Safe upgrades |

### 2.3 Schema Concerns

| Concern | Severity | Details |
|---------|----------|---------|
| **Single SQLite Connection** | 🔴 High | `SetMaxOpenConns(1)` creates bottleneck under load |
| **JSON-Heavy Design** | 🟡 Medium | May impact query performance for complex filters |
| **No Database Sharding** | 🟡 Medium | Single-node limitation |
| **No Read Replicas** | 🟢 Low | Single writer model sufficient for desktop app |

### 2.4 Key Schema Types

```go
// Project - Worktree management
type Project struct {
    ID              string
    Worktree        string        // Required
    VCS             string        // git, etc.
    Name, IconURL   string
    Sandboxes       []string      // JSON array
    Commands        *Commands     // JSON
}

// Session - Conversation container
type Session struct {
    ID          string
    ProjectID   string        // FK
    Slug, Title string        // Required
    Version     string
    Summary*    fields        // Statistics
    Permission  *Ruleset      // JSON
}

// Message - Chat message
type Message struct {
    ID         string
    SessionID  string        // FK
    Data       MessageInfo   // JSON with role, parent, agent, model, etc.
}

// Part - Message component
type Part struct {
    ID         string
    MessageID  string        // FK
    Data       PartData      // JSON with type, content, tool info
}
```

---

## 3. API DESIGN PATTERNS

### 3.1 RESTful Endpoints

The server exposes two categories of routes:

**Global Routes** (no instance required):
```
GET  /global/health     - Health check
GET  /global/version    - Version info
GET  /global/event      - Global event stream (SSE)
```

**Instance Routes** (require project context):
```
Sessions:
  GET    /session                     - List sessions
  POST   /session                     - Create session
  GET    /session/:id                 - Get session
  PUT    /session/:id                 - Update session
  DELETE /session/:id                 - Delete session
  GET    /session/:id/message         - Get messages
  POST   /session/:id/prompt_async    - Async processing
  GET    /session/:id/events          - SSE stream
  POST   /session/:id/run             - Run session
  POST   /session/:id/cancel          - Cancel processing
  POST   /session/:id/abort           - Abort processing

Files:
  GET    /file/read                  - Read file
  POST   /file/write                 - Write file
  POST   /file/edit                  - Edit file
  GET    /file/glob                  - File globbing
  GET    /file/grep                  - Text search
  GET    /file/list                  - List directory

Providers:
  GET    /provider                   - List providers
  GET    /provider/:id               - Get provider
  GET    /provider/:id/auth          - Auth status

LSP:
  GET    /lsp                        - LSP status
  POST   /lsp/:language/start        - Start LSP
  POST   /lsp/:language/stop         - Stop LSP
  GET    /lsp/:language/diagnostics  - Get diagnostics

Configuration:
  GET    /config                     - Get config
  PUT    /config                     - Update config
```

### 3.2 Async Processing Pattern

The system uses asynchronous message processing with Server-Sent Events (SSE):

```
Client                                    Server
  │                                          │
  │  POST /session/:id/prompt_async         │
  │  { "content": "hello" }                │
  │────────────────────────────────────────>│
  │                                          │
  │  { "status": "accepted", "message_id" }  │
  │<────────────────────────────────────────│
  │                                          │
  │  GET /session/:id/events (SSE)          │
  │────────────────────────────────────────>│
  │                                          │
  │  event: session.part.created             │
  │  data: {...}                             │
  │<────────────────────────────────────────│
  │  event: session.part.updated           │
  │  data: {...}                             │
  │<────────────────────────────────────────│
  │  event: session.tool.pending           │
  │  data: {...}                             │
  │<────────────────────────────────────────│
  │  ... more events ...                     │
```

### 3.3 WebSocket Support

WebSocket endpoints are defined but marked as placeholder:

```go
// PTY WebSocket - Not fully implemented
GET /pty/:id/connect - WebSocket upgrader placeholder
```

**Status:** ⚠️ WebSocket implementation incomplete; SSE used as primary streaming mechanism.

### 3.4 API Patterns

| Pattern | Implementation |
|---------|---------------|
| **Consistent Responses** | JSON with `fiber.Map` wrapper |
| **HTTP Status Codes** | Proper use: 200, 201, 400, 404, 500 |
| **Validation Middleware** | Request validation at entry points |
| **CORS Support** | Configurable origins |
| **Compression** | Gzip compression enabled by default |

---

## 4. PROVIDER ABSTRACTION

### 4.1 Provider Interface

The system defines a clean abstraction for AI providers:

```go
// Provider interface - implement to add new AI providers
type Provider interface {
    ID() ProviderID
    Name() string
    BaseURL() string
    
    // Chat methods
    Chat(ctx context.Context, req ChatRequest) (*ChatResponse, error)
    StreamChat(ctx context.Context, req ChatRequest) (<-chan StreamEvent, error)
    
    // Model management
    Models() []ModelInfo
    ValidateKey(ctx context.Context, apiKey string) error
}
```

### 4.2 Supported Providers

| Provider | ID | Auth Method | Models |
|----------|-----|-------------|--------|
| Anthropic | `anthropic` | API Key | Claude 3.x series |
| OpenAI | `openai` | API Key | GPT-4, GPT-3.5 |
| Google | `google` | API Key | Gemini |
| Google Vertex | `google-vertex` | Service Account | Vertex AI |
| Azure | `azure` | API Key | Azure OpenAI |
| Amazon Bedrock | `amazon-bedrock` | AWS Credentials | Bedrock models |
| OpenRouter | `openrouter` | API Key | Multi-provider |
| Mistral | `mistral` | API Key | Mistral series |
| Groq | `groq` | API Key | Llama, Mixtral |
| xAI | `xai` | API Key | Grok |
| Cerebras | `cerebras` | API Key | Cerebras models |
| Cohere | `cohere` | API Key | Command series |
| Together AI | `togetherai` | API Key | Various |
| Perplexity | `perplexity` | API Key | Perplexity models |
| DeepInfra | `deepinfra` | API Key | Various |
| GitHub Copilot | `github-copilot` | Token | Copilot models |

### 4.3 Model Management

**ModelID Format:** `provider/model-name`

Examples:
- `anthropic/claude-3-5-sonnet-20241022`
- `openai/gpt-4`
- `google/gemini-1.5-pro`

**ModelInfo Structure:**
```go
type ModelInfo struct {
    ID          ModelID
    Name        string
    Description string
    
    // Capabilities
    MaxTokens         int
    MaxInputTokens    int
    MaxOutputTokens   int
    SupportsVision    bool
    SupportsTools     bool
    SupportsStreaming bool
    
    // Cost tracking (per 1M tokens)
    Cost Cost
    
    // Provider metadata
    Metadata map[string]interface{}
}
```

### 4.4 Streaming Architecture

Provider responses are streamed using channel-based events:

```go
// StreamEvent interface - all events implement this
type StreamEvent interface {
    EventType() string
}

// Event types:
// - MessageStartEvent    - Message begins
// - ContentBlockStartEvent - New content block
// - ContentBlockDeltaEvent - Content update
// - ContentBlockStopEvent  - Block complete
// - MessageDeltaEvent      - Message update
// - MessageStopEvent       - Message complete
// - PingEvent              - Keepalive
// - ErrorEvent             - Error occurred
```

### 4.5 Provider Registry

```go
type ProviderRegistry struct {
    providers map[ProviderID]Provider
    models    map[ModelID]ModelInfo
    mu        sync.RWMutex
}

// Usage:
registry := provider.NewProviderRegistry()
registry.Register(provider.NewAnthropicProvider(apiKey))
registry.Register(provider.NewOpenAIProvider(apiKey))

// Get provider by ID
prov, ok := registry.Get("anthropic")

// Get model info
model, ok := registry.GetModel("anthropic/claude-3-5-sonnet-20241022")
```

---

## 5. SESSION MANAGEMENT

### 5.1 Session Processor Architecture

The `session.Processor` orchestrates the entire AI conversation flow:

```
┌────────────────────────────────────────────────────────────────┐
│                    SESSION PROCESSOR FLOW                       │
└────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌────────────────────────────────────────────────────────────────┐
│ 1. CREATE USER MESSAGE                                          │
│    • Save to database                                           │
│    • Publish EventMessageCreated                                │
└────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌────────────────────────────────────────────────────────────────┐
│ 2. BUILD PROVIDER REQUEST                                       │
│    • Format history from database                               │
│    • Apply system prompt                                        │
│    • Configure tools                                            │
└────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌────────────────────────────────────────────────────────────────┐
│ 3. STREAM FROM AI PROVIDER                                      │
│    • Create assistant message in DB                             │
│    • Stream response chunks                                     │
│    • Publish EventPartCreated/Updated                           │
└────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌────────────────────────────────────────────────────────────────┐
│ 4. HANDLE TOOL CALLS                                            │
│    • Detect tool_use in stream                                  │
│    • Publish EventToolCallPending                               │
│    • Execute tool with context                                  │
│    • Publish EventToolCallRunning/Complete                        │
└────────────────────────────────────────────────────────────────┘
                              │
                              ▼
┌────────────────────────────────────────────────────────────────┐
│ 5. PERSIST RESULTS                                              │
│    • Store tool results as tool_result parts                    │
│    • Update message metadata                                    │
│    • Publish EventMessageComplete                               │
└────────────────────────────────────────────────────────────────┘
```

### 5.2 State Tracking

```go
type Processor struct {
    registry     *provider.ProviderRegistry
    db           *database.Database
    bus          *bus.Service
    toolRegistry *tool.Registry
    mu           sync.Mutex
    active       map[string]context.CancelFunc  // Active by session ID
}

// Key features:
// - Cancel existing processing per session
// - Context cancellation support
// - Event publishing for UI updates
```

### 5.3 Session Storage

**Message Pagination** (OpenCode-compatible):
- **Initial load:** 80 messages
- **History load:** 200 messages
- **Cursor-based:** Timestamp (int64 milliseconds)

```go
// Pagination query
messages, cursor, complete, err := messageStorage.ListPaginated(
    ctx, 
    sessionID, 
    limit,      // 80 or 200
    cursor,     // 0 for initial, timestamp for history
)
```

**Part Management:**
- `text` parts - Regular message content
- `tool_use` parts - AI tool invocations
- `tool_result` parts - Tool execution results
- `reasoning` parts - AI reasoning/thinking
- `step-start/step-finish` parts - Workflow markers

---

## 6. STATE MANAGEMENT

### 6.1 Layered Store Implementation

```go
// AppState - Central state container
type AppState struct {
    KV     *KVStore      // Persistent preferences
    Sync   *SyncStore    // Server/database data
    Local  *LocalStore   // UI state
    Layout *LayoutStore  // Visual layout
    Dialog *DialogStore  // Modal stack
}
```

### 6.2 Store Details

**KVStore** (Persistent):
```go
type KVStore struct {
    db *KVStorage  // SQLite-backed
}
// Stores: theme, sidebar settings, review settings, timestamps
```

**SyncStore** (Database-backed):
```go
type SyncStore struct {
    Sessions      []Session
    Messages      map[string][]Message  // Per-session
    Providers     []Provider
    Agents        []Agent
    LSPStatus     map[string]bool
    MCPStatus     map[string]bool
}
```

**LocalStore** (In-memory):
```go
type LocalStore struct {
    CurrentAgent    string
    CurrentModel    ModelKey
    InputContent    string
    InputMode       InputMode
    CommandHistory  []string
    SessionState    map[string]*SessionLocalState
}
```

**LayoutStore** (Visual):
```go
type LayoutStore struct {
    Width, Height   int
    SidebarOpen     bool
    SidebarWidth    int  // 344px default
    ScrollPosition  map[string]int
}
```

**DialogStore** (Modal):
```go
type DialogStore struct {
    Stack []Dialog  // Push/pop semantics
}
```

---

## 7. CONCURRENCY MODEL

### 7.1 Goroutine Patterns

| Component | Pattern | Goroutines |
|-----------|---------|------------|
| **Server** | Background | 1 per listener |
| **Session Processor** | Per-request | 1 per async processing |
| **Event Bus** | Fan-out | Publishers + Subscribers |
| **TUI** | Framework-managed | Bubble Tea handles it |
| **PTY** | Per-terminal | 1 per PTY session |
| **LSP** | Per-language | 1 per LSP client |

### 7.2 Synchronization

```go
// Mutex usage across packages:
type Server struct {
    mu      sync.RWMutex
    running bool
}

type ProviderRegistry struct {
    mu        sync.RWMutex
    providers map[ProviderID]Provider
    models    map[ModelID]ModelInfo
}

type ToolRegistry struct {
    mu    sync.RWMutex
    tools map[ToolID]Tool
}

type Service struct {  // Event bus
    mu          sync.RWMutex
    subscribers map[string][]chan Payload
    wildcard    []chan Payload
}

type Processor struct {
    mu     sync.Mutex
    active map[string]context.CancelFunc
}
```

### 7.3 Context Usage

```go
// Proper context propagation:
processCtx, cancel := context.WithCancel(ctx)
defer cancel()

// Timeout handling:
ctx, cancel := context.WithTimeout(parentCtx, 30*time.Second)
defer cancel()

// Cancellation support:
select {
case <-ctx.Done():
    return ctx.Err()
case result := <-workChan:
    return result
}
```

### 7.4 Concurrency Concerns

| Concern | Severity | Details |
|---------|----------|---------|
| **Multiple Mutexes** | 🟡 Medium | May lead to contention under load |
| **Global Bus Subscribers** | 🟡 Medium | Wildcard subscribers not bounded |
| **Event Channel Buffer** | 🟢 Low | 50-item buffer with overflow handling |
| **Database Lock** | 🔴 High | Single connection limits concurrency |

---

## 8. ERROR HANDLING STRATEGY

### 8.1 Error Types

```go
// Tool errors
type ToolError struct {
    ToolID  string
    Message string
    Type    string  // validation_error, execution_error, permission_denied, timeout
    Details map[string]interface{}
}

// Provider errors
var (
    ErrProviderNotFound     = errors.New("provider not found")
    ErrModelNotFound        = errors.New("model not found")
    ErrInvalidAPIKey        = errors.New("invalid API key")
    ErrProviderUnavailable  = errors.New("provider unavailable")
)

// Bus errors
var (
    ErrBusClosed       = errors.New("bus is closed")
    ErrMaxSubscribers  = errors.New("maximum subscribers reached")
)
```

### 8.2 Error Wrapping Pattern

```go
// Standard error wrapping
if err != nil {
    return fmt.Errorf("operation failed: %w", err)
}

// Structured error with context
return fmt.Errorf("failed to process message %s: %w", messageID, err)
```

### 8.3 HTTP Error Handling

| Status | Usage |
|--------|-------|
| **400** | Validation errors, bad requests |
| **401** | Authentication required |
| **403** | Permission denied |
| **404** | Resource not found |
| **500** | Internal server errors |
| **501** | Not implemented |

### 8.4 Error Concerns

| Concern | Severity | Details |
|---------|----------|---------|
| **Generic Error Messages** | 🟡 Medium | Some handlers return generic errors to clients |
| **Inconsistent Logging** | 🟡 Medium | Mixed logging patterns across packages |
| **No Centralized Handler** | 🟡 Medium | Each handler manages own errors |

---

## 9. CONFIGURATION MANAGEMENT

### 9.1 Configuration Sources

Priority (highest to lowest):

1. **Environment Variables** - Runtime overrides
2. **Project Config** - `./magicode.json` (current directory)
3. **Global Config** - `~/.config/magicode/opencode.json`
4. **OpenCode Config** - `~/.config/opencode/opencode.json` (compatibility)
5. **Default Config** - Hardcoded defaults

### 9.2 Config Features

```go
type Config struct {
    // Provider configuration
    Providers []ProviderConfig
    
    // Path customization
    ConfigDir  string
    StateDir   string
    DatabasePath string
    
    // OpenCode compatibility
    UseOpenCode bool
    
    // Server settings
    Port     int
    Hostname string
    CORS     []string
}
```

**Features:**
- JSON and JSONC support (comments allowed)
- Backward compatibility with OpenCode
- XDG Base Directory compliance
- Custom path overrides

### 9.3 Path Management

```go
// XDG-compliant paths:
ConfigDir:  ~/.config/magicode/
StateDir:   ~/.local/share/magicode/
CacheDir:   ~/.cache/magicode/

// OpenCode compatibility:
OpenCodeConfigDir: ~/.config/opencode/
OpenCodeStateDir:  ~/.local/share/opencode/
```

---

## 10. EXTENSIBILITY POINTS

### 10.1 Tool System

```go
// Tool interface - implement to add new tools
type Tool interface {
    ID() ToolID
    Definition() ToolDefinition  // Schema for AI
    Execute(ctx context.Context, params map[string]interface{}, toolCtx ToolContext) (*ToolResult, error)
    Validate(params map[string]interface{}) error
}

// Registration:
registry := tool.NewRegistry()
registry.Register(tool.NewReadTool())
registry.Register(tool.NewWriteTool())
registry.Register(tool.NewEditTool())
registry.Register(tool.NewBashTool())
registry.Register(tool.NewGlobTool())
registry.Register(tool.NewGrepTool())
registry.Register(tool.NewWebFetchTool())
```

### 10.2 Provider System

```go
// Provider interface - implement to add new AI providers
type Provider interface {
    ID() ProviderID
    Name() string
    BaseURL() string
    Chat(ctx context.Context, req ChatRequest) (*ChatResponse, error)
    StreamChat(ctx context.Context, req ChatRequest) (<-chan StreamEvent, error)
    Models() []ModelInfo
    ValidateKey(ctx context.Context, apiKey string) error
}

// Registration:
registry := provider.NewProviderRegistry()
registry.Register(myCustomProvider)
```

### 10.3 Event System

```go
// Custom event definitions:
var MyEvent = bus.Definition{Type: "my.event"}

// Publishing:
bus.Publish(MyEvent, data)

// Subscribing:
ch, cleanup := bus.Subscribe(MyEvent)
defer cleanup()

// Wildcard subscriptions:
ch, cleanup := bus.SubscribeAll()
```

### 10.4 TUI Components

```go
// Dialog interface
type Dialog interface {
    Init() tea.Cmd
    Update(msg tea.Msg) (Dialog, tea.Cmd)
    View() string
    Height() int
    IsFullScreen() bool
}

// Theme system
type Theme struct {
    Name        string
    Background  lipgloss.Color
    Foreground  lipgloss.Color
    // ... 40+ color definitions
}
```

---

## ARCHITECTURE STRENGTHS

### ✅ Strength 1: Clean Separation of Concerns
- Clear layer boundaries (presentation, application, domain, infrastructure)
- Package-level isolation prevents circular dependencies
- `internal/tui/types` package breaks import cycles

### ✅ Strength 2: Event-Driven Architecture
- Loose coupling via event bus
- Async processing via SSE
- Type-safe event definitions

### ✅ Strength 3: Provider Abstraction
- Interface-based design
- Easy to add new AI providers
- Model registry with capability detection
- 14+ providers supported

### ✅ Strength 4: State Layering
- Matches OpenCode's architecture
- Enables seamless migration
- Clear ownership of data

### ✅ Strength 5: Memory Efficiency
- Bounded caches (10 instance max, 30min TTL)
- Proper cleanup (channels, goroutines, resources)
- SQLite instead of in-memory

### ✅ Strength 6: OpenCode Compatibility
- Migration path from TypeScript version
- Compatible database schema
- Compatible state management
- Compatible file formats

### ✅ Strength 7: Streaming Support
- Real-time response handling
- Channel-based streaming
- Proper error handling in streams

### ✅ Strength 8: Tool System
- Extensible tool framework
- Well-defined interfaces
- 7 built-in tools

### ✅ Strength 9: Standards Compliance
- XDG Base Directory compliance
- LSP protocol compliance
- REST API conventions

### ✅ Strength 10: Transaction Support
- Database transaction wrapper
- `InTransaction()` helper
- Automatic rollback on panic

---

## DESIGN CONCERNS

### 🔴 High Priority Concerns

#### 1. SQLite Single Connection
**Problem:** `db.SetMaxOpenConns(1)` creates bottleneck

**Impact:** Serializes all database operations

**Recommendation:**
```go
// Current:
db.SetMaxOpenConns(1)

// Recommended:
db.SetMaxOpenConns(10)
db.SetMaxIdleConns(5)
db.SetConnMaxLifetime(time.Hour)
```

#### 2. Global Bus Cleanup
**Problem:** Wildcard subscribers not bounded

**Impact:** Unbounded memory growth

**Recommendation:** Add subscriber limits and TTL

#### 3. Error Handling Consistency
**Problem:** Mixed patterns across packages

**Recommendation:** Implement centralized error middleware

#### 4. WebSocket Not Implemented
**Problem:** Placeholder only

**Impact:** PTY WebSocket unavailable

**Recommendation:** Complete implementation or remove

#### 5. Request ID Generation
**Problem:** Time-based randomness not cryptographically secure

**Recommendation:** Use `crypto/rand` for request IDs

### 🟡 Medium Priority Concerns

#### 6. Session Event Loop
**Problem:** Multiple event subscriptions in SSE handler

**Recommendation:** Consolidate subscriptions

#### 7. JSON-Heavy Schema
**Problem:** May impact query performance

**Recommendation:** Add specialized indexes for JSON fields

#### 8. Context Key Types
**Problem:** Using empty structs could conflict

**Recommendation:** Use typed constants

#### 9. Tool Context Naming
**Problem:** `Abort` context naming is confusing

**Recommendation:** Rename to `Cancel` or `Done`

#### 10. Provider Model Sync
**Problem:** Manual sync between provider and registry

**Recommendation:** Automatic model discovery

---

## SCALABILITY ISSUES

### Current Limitations

| Limitation | Impact | Mitigation |
|------------|--------|------------|
| **Single Instance** | No horizontal scaling | Desktop app use case |
| **SQLite Database** | Single writer, file locking | WAL mode, single connection |
| **In-Memory Caches** | No distributed cache | Desktop app use case |
| **Event Bus** | Single-node only | Local-only events |
| **No Rate Limiting** | Could be overwhelmed | Add middleware |

### Memory Considerations

| Component | Limit | Strategy |
|-----------|-------|----------|
| **Message Cache** | 80 initial + 200 history | Pagination |
| **Event Subscribers** | 100 per type | Bounded with eviction |
| **Event Buffer** | 50 per subscriber | Overflow handling |
| **Instance Cache** | 10 max | LRU with TTL |
| **Dialog Stack** | Unbounded | Add limit |

**Assessment:** Suitable for single-user desktop application. Would need significant changes for multi-user or distributed deployment.

---

## COUPLING/COHESION ANALYSIS

### Strong Cohesion ✅

| Package | Cohesion Level | Evidence |
|---------|---------------|----------|
| `provider` | High | Single responsibility: AI provider abstraction |
| `tool` | High | Well-defined interfaces, clear boundaries |
| `database` | High | Clean CRUD operations, schema definitions |
| `bus` | High | Focused event handling, no scope creep |
| `config` | High | Configuration management only |

### Loose Coupling ✅

| Component | Coupling | Design |
|-----------|----------|--------|
| Provider Interface | Loose | Abstracts AI providers via interface |
| Event Bus | Loose | Decouples publishers/subscribers |
| Tool Interface | Loose | Plugin architecture |
| Database | Loose | Interface-based (could swap DB) |

### Tight Coupling ⚠️

| Component | Coupling | Problem |
|-----------|----------|---------|
| Session Processor | Tight | Depends on database, provider, bus, tools |
| Server Handlers | Tight | Direct service dependencies |
| TUI App | Tight | Large state management surface |
| Provider Registry | Tight | Manual model synchronization |

**Recommendation:** Consider dependency injection to reduce coupling in tight areas.

---

## RECOMMENDATIONS FOR IMPROVEMENT

### Immediate (High Priority) 🔴

#### 1. Increase SQLite Connections
```go
// In internal/database/database.go
db.SetMaxOpenConns(10)
db.SetMaxIdleConns(5)
db.SetConnMaxLifetime(time.Hour)
```

#### 2. Secure Request ID Generation
```go
// Use crypto/rand instead of time-based
import "crypto/rand"

func generateRequestID() string {
    b := make([]byte, 16)
    rand.Read(b)
    return hex.EncodeToString(b)
}
```

#### 3. Add Rate Limiting Middleware
```go
// In internal/server/middleware.go
func rateLimitMiddleware(limit int, window time.Duration) fiber.Handler {
    // Implement token bucket or sliding window
}
```

#### 4. Implement WebSocket Handler
Complete the placeholder in `internal/server/websocket.go`:
```go
func (s *Server) handlePTYConnect(c *fiber.Ctx) error {
    // Complete WebSocket implementation
    // Integrate with PTY manager
}
```

#### 5. Centralize Error Handling
```go
// Error middleware
func errorMiddleware() fiber.Handler {
    return func(c *fiber.Ctx) error {
        err := c.Next()
        if err != nil {
            // Standardize error response
            return c.Status(getStatusCode(err)).JSON(ErrorResponse{
                Error: err.Error(),
                Code: getErrorCode(err),
            })
        }
        return nil
    }
}
```

### Short-Term (Medium Priority) 🟡

#### 6. Database Indexing Audit
- Review query patterns
- Add indexes for frequent queries:
```sql
CREATE INDEX idx_message_session_time ON message(session_id, time_created);
CREATE INDEX idx_part_message ON part(message_id);
CREATE INDEX idx_todo_session ON todo(session_id, position);
```

#### 7. Event Bus Optimization
- Add subscriber timeout
- Implement backpressure handling
- Consider event sourcing for critical paths

#### 8. Configuration Validation
- Add config schema validation
- Environment variable override support
- Config hot-reloading

#### 9. Health Check Enhancement
```go
// Comprehensive health check
func (s *Server) handleHealth(c *fiber.Ctx) error {
    checks := map[string]bool{
        "database": s.services.DB.Ping() == nil,
        "provider": s.services.Provider.HasProviders(),
        "disk":     checkDiskSpace(),
    }
    // Return detailed health status
}
```

#### 10. Logging Standardization
```go
// Structured logging throughout
log.Info("message", 
    "request_id", reqID,
    "session_id", sessionID,
    "duration", time.Since(start),
)
```

### Long-Term (Low Priority) 🟢

#### 11. Database Sharding
- Consider project-based sharding
- Read replicas for heavy query loads

#### 12. Caching Layer
- Redis for session caching (if multi-user)
- Message caching with TTL
- Provider response caching

#### 13. API Versioning
```
/v1/session
/v1/provider
/v2/session (future)
```

#### 14. GraphQL Option
For complex data fetching and reduced over-fetching.

#### 15. Monitoring & Observability
- Prometheus metrics collection
- Distributed tracing (OpenTelemetry)
- Performance profiling endpoints

### Code Organization

#### 16. Interface Segregation
```go
// Split large Provider interface
type ChatProvider interface {
    Chat(ctx context.Context, req ChatRequest) (*ChatResponse, error)
    StreamChat(ctx context.Context, req ChatRequest) (<-chan StreamEvent, error)
}

type ModelProvider interface {
    Models() []ModelInfo
}
```

#### 17. Dependency Injection
```go
// Use DI container
type Container struct {
    DB       *database.Database
    Bus      *bus.Service
    Provider *provider.ProviderRegistry
    Tools    *tool.Registry
}
```

#### 18. Testing Improvements
- Integration tests with real database
- Load testing with `go test -bench`
- Chaos engineering (fault injection)

---

## CONCLUSION

### Summary

The MagiCode architecture demonstrates solid engineering principles with:
- ✅ Clear separation of concerns
- ✅ Event-driven design
- ✅ Extensible abstractions
- ✅ Memory efficiency
- ✅ OpenCode compatibility

### Primary Strengths

1. **Clean provider/tool interface system** enables easy extension
2. **Event-driven architecture** provides loose coupling
3. **Memory-efficient design** achieves <500MB target
4. **Comprehensive state management** matches OpenCode behavior

### Areas for Improvement

1. **Scalability** - SQLite single connection limits concurrency
2. **Security** - Request ID generation needs crypto-rand
3. **Completeness** - WebSocket implementation incomplete

### Deployment Suitability

| Scenario | Suitability | Notes |
|----------|-------------|-------|
| Single-user desktop | ✅ Excellent | Primary use case |
| Small team (<5 users) | 🟡 Acceptable | With SQLite connection increase |
| Enterprise multi-user | ❌ Not suitable | Requires architecture changes |
| Distributed/cloud | ❌ Not suitable | Requires significant redesign |

### Final Assessment

The architecture is **well-suited for its intended purpose** as a single-user AI coding assistant. The OpenCode compatibility layer enables seamless migration while maintaining the memory efficiency goals of the Go implementation.

**Grade: A-** (Excellent with minor concerns)

The main areas requiring attention before production deployment at scale are the SQLite connection limit and security-related request ID generation.

---

## APPENDIX

### A. Package Dependencies

```
internal/
├── bus/
│   └── (no internal deps)
├── database/
│   └── internal/util/log
├── provider/
│   └── internal/util/log
├── session/
│   ├── internal/bus
│   ├── internal/database
│   ├── internal/provider
│   └── internal/tool
├── server/
│   ├── internal/bus
│   ├── internal/config
│   ├── internal/database
│   ├── internal/lsp
│   ├── internal/provider
│   ├── internal/pty
│   ├── internal/session
│   └── internal/tool
├── tui/
│   ├── internal/bus
│   ├── internal/config
│   ├── internal/database
│   ├── internal/provider
│   ├── internal/session
│   └── internal/tui/types (breaks cycles)
└── tool/
    └── internal/util/log
```

### B. External Dependencies

| Package | Purpose | License |
|---------|---------|---------|
| github.com/gofiber/fiber/v2 | HTTP framework | MIT |
| modernc.org/sqlite | SQLite driver | BSD |
| github.com/charmbracelet/bubbletea | TUI framework | MIT |
| github.com/charmbracelet/lipgloss | Styling | MIT |
| github.com/charmbracelet/bubbles | TUI components | MIT |

### C. Benchmark Results

See individual `*_bench_test.go` files for:
- `internal/bus/bus_bench_test.go`
- `internal/provider/provider_bench_test.go`
- `internal/instance/instance_bench_test.go`
- `internal/pty/pty_bench_test.go`
- `internal/lsp/lsp_bench_test.go`

---

**Document maintained by:** Architecture Review Team  
**Last updated:** 2026-05-11
