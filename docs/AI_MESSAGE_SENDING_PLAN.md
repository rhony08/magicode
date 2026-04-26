# MagiCode AI Message Sending Implementation Plan

## Executive Summary

Currently, MagiCode's message sending in TUI shows a **placeholder response**. However, the **provider layer already supports streaming** (Anthropic + OpenAI). This plan focuses on:
1. Connecting TUI to existing provider streaming
2. Adding OpenCode-compatible providers from config
3. Building message processing pipeline

---

## Current State Analysis

### MagiCode (Already Has Streaming!)
**Provider Layer:** ✅ Fully implemented
- `anthropic.go` - Full SSE streaming with event parsing
- `openai.go` - Full SSE streaming with tool call support
- `types.go` - StreamEvent types (ContentBlockStart, ContentBlockDelta, etc.)

**Missing:** ❌
- TUI integration (submitInput shows placeholder)
- Message processing service (SessionProcessor)
- Tool execution during streaming
- Event bus connection to TUI

### OpenCode Provider Coverage
| Provider | OpenCode | MagiCode | Base Type |
|----------|----------|----------|-----------|
| anthropic | ✅ | ✅ | Native |
| openai | ✅ | ✅ | Native |
| azure | ✅ | ❌ | OpenAI-compatible |
| google | ✅ | ❌ | Native (Gemini) |
| google-vertex | ✅ | ❌ | Native |
| openrouter | ✅ | ❌ | OpenAI-compatible |
| groq | ✅ | ❌ | OpenAI-compatible |
| mistral | ✅ | ❌ | OpenAI-compatible |
| togetherai | ✅ | ❌ | OpenAI-compatible |
| perplexity | ✅ | ❌ | OpenAI-compatible |
| xai | ✅ | ❌ | OpenAI-compatible |
| cerebras | ✅ | ❌ | OpenAI-compatible |
| deepinfra | ✅ | ❌ | OpenAI-compatible |
| github-copilot | ✅ | ❌ | OpenAI-compatible |
| bedrock | ✅ | ❌ | Native (AWS) |
| cohere | ✅ | ❌ | Native |
| gitlab | ✅ | ❌ | Native (Workflow) |
| opencode | ✅ | ❌ | Native (their API) |

---

## Architecture Comparison

### OpenCode Flow
```
TUI Input
    ↓
POST /:sessionID/message (or prompt_async)
    ↓
SessionPrompt.Service.prompt()
    ↓
LLM.Service.stream()
    ↓
Provider.Service.getLanguage(model)
    ↓
streamText({ model, messages, tools, ... })
    ↓
Stream<Event> → SessionProcessor
    ↓
Tool execution + Bus events
    ↓
TUI receives events via SSE/WebSocket
```

### MagiCode Current Flow
```
TUI Input
    ↓
submitInput() → placeholder response
    ↓
Shows "Processing..." spinner
    ↓
Returns static "This is a placeholder response"
```

### MagiCode Target Flow
```
TUI Input (submitInput)
    ↓
POST /session/:id/prompt_async (HTTP)
    ↓
SessionProcessor.Process()
    ↓
Provider.StreamChat() ← ALREADY IMPLEMENTED!
    ↓
Parse SSE events → execute tools
    ↓
Publish to Bus → TUI subscribes
    ↓
TUI updates messages in real-time
```

---

## Implementation Plan

### Phase 0: Verify Existing Streaming (QUICK)

**Goal:** Confirm Anthropic + OpenAI streaming actually works

**Status:** Providers have `StreamChat()` implemented - verify with test

```bash
# Quick test
cd magicode
go test ./internal/provider/... -v -run TestAnthropicStream
go test ./internal/provider/... -v -run TestOpenAIStream
```

**Tasks:**
- [ ] Add streaming integration test for Anthropic
- [ ] Add streaming integration test for OpenAI
- [ ] Verify SSE parsing works correctly
- [ ] Test tool_use event handling

---

### Phase 1: Add OpenAI-Compatible Providers (MEDIUM)

**Goal:** Add providers from OpenCode config that use OpenAI-compatible API

#### 1.1 Bundled OpenAI-Compatible Providers

**Easy wins:** These providers use OpenAI-compatible endpoints, so we can reuse `openai.go`:

| Provider | Base URL | Environment Key |
|----------|----------|-----------------|
| openrouter | `https://openrouter.ai/api/v1` | `OPENROUTER_API_KEY` |
| groq | `https://api.groq.com/openai/v1` | `GROQ_API_KEY` |
| mistral | `https://api.mistral.ai/v1` | `MISTRAL_API_KEY` |
| togetherai | `https://api.together.xyz/v1` | `TOGETHERAI_API_KEY` |
| perplexity | `https://api.perplexity.ai` | `PERPLEXITY_API_KEY` |
| xai | `https://api.x.ai/v1` | `XAI_API_KEY` |
| cerebras | `https://api.cerebras.ai/v1` | `CEREBRAS_API_KEY` |
| deepinfra | `https://api.deepinfra.com/v1/openai` | `DEEPINFRA_API_KEY` |

#### 1.2 Custom Providers from OpenCode Config

**OpenCode allows users to define custom providers in `opencode.json`:**

```json
{
  "provider": {
    "my-custom-provider": {
      "name": "My Custom API",
      "baseURL": "https://my-api.example.com/v1",
      "apiKey": "xxx",
      "models": {
        "custom-model": {
          "id": "custom-model",
          "name": "Custom Model",
          "limit": { "context": 128000, "output": 4096 }
        }
      }
    }
  }
}
```

**Implementation:** Dynamic provider registration from config

```go
// File: internal/provider/dynamic.go

// DynamicProvider creates a provider from OpenCode config
type DynamicProvider struct {
    *OpenAIProvider
    providerID ProviderID
    name       string
    models     map[ModelID]ModelInfo
}

// RegisterFromConfig reads OpenCode config and registers all providers
func RegisterFromConfig(registry *ProviderRegistry, configReader *opencode.ConfigReader) error {
    providers, err := configReader.ReadProviders()
    if err != nil {
        return err
    }
    
    for _, p := range providers {
        // Skip bundled providers (already registered)
        if isBundledProvider(p.ID) {
            // Just add models to existing provider
            registry.AddModels(ProviderID(p.ID), convertModels(p.Models))
            continue
        }
        
        // Create dynamic provider for custom providers
        dp := NewDynamicProvider(p.ID, p.Name, p.BaseURL, p.APIKey, p.Models)
        registry.Register(dp)
    }
    
    return nil
}

func NewDynamicProvider(id, name, baseURL, apiKey string, models map[string]opencode.Model) *DynamicProvider {
    p := NewOpenAIProvider(apiKey)
    if baseURL != "" {
        p.SetBaseURL(baseURL)
    }
    
    // Convert models
    modelInfo := convertModels(models)
    
    return &DynamicProvider{
        OpenAIProvider: p,
        providerID:     ProviderID(id),
        name:           name,
        models:         modelInfo,
    }
}
```

**Tasks:**
- [ ] Create `openai_compatible.go` base for bundled providers
- [ ] Create `dynamic.go` for custom providers from config
- [ ] Add OpenRouter provider (bundled)
- [ ] Add Groq provider (bundled)
- [ ] Add Mistral provider (bundled)
- [ ] Add TogetherAI provider (bundled)
- [ ] Add Perplexity provider (bundled)
- [ ] Add XAI (Grok) provider (bundled)
- [ ] Add Cerebras provider (bundled)
- [ ] Add DeepInfra provider (bundled)
- [ ] Implement `RegisterFromConfig()` for dynamic registration
- [ ] Read models from OpenCode config for each provider
- [ ] Test with custom provider defined in config

---

### Phase 2: SessionProcessor Service (HIGH)

**Goal:** Create service that connects TUI → Provider → Database

**New File:** `internal/session/processor.go`

```go
package session

import (
    "context"
    "github.com/rhony08/magicode/internal/provider"
    "github.com/rhony08/magicode/internal/database"
    "github.com/rhony08/magicode/internal/bus"
    "github.com/rhony08/magicode/internal/tool"
)

// Processor handles AI message processing
type Processor struct {
    registry   *provider.ProviderRegistry
    db         *database.DB
    bus        *bus.Bus
    toolRunner *tool.Runner
}

// ProcessRequest contains all info for processing a message
type ProcessRequest struct {
    SessionID    string
    UserMessage  database.Message
    Model        provider.ModelID
    SystemPrompt string
    History      []database.Message  // Previous messages for context
    Tools        []provider.ToolDefinition
}

// Process streams response from AI, executes tools, stores results
func (p *Processor) Process(ctx context.Context, req ProcessRequest) error {
    // 1. Build provider request
    chatReq := provider.ChatRequest{
        Model:    req.Model,
        Messages: p.convertMessages(req.History),
        System:   req.SystemPrompt,
        Tools:    req.Tools,
        Stream:   true,
    }
    
    // 2. Get provider from registry
    providerID, _ := provider.ParseModelID(req.Model)
    prov, ok := p.registry.Get(providerID)
    if !ok {
        return fmt.Errorf("provider not found: %s", providerID)
    }
    
    // 3. Create assistant message in DB (for storing parts)
    assistantMsg := database.Message{
        SessionID: req.SessionID,
        Data: database.MessageInfo{
            Role:       "assistant",
            ModelID:    string(req.Model),
            ProviderID: string(providerID),
        },
    }
    assistantMsg, _ = p.db.Messages.Create(ctx, assistantMsg)
    
    // 4. Start streaming
    events, err := prov.StreamChat(ctx, chatReq)
    if err != nil {
        return err
    }
    
    // 5. Process stream events
    for event := range events {
        p.handleStreamEvent(ctx, req.SessionID, assistantMsg.ID, event)
    }
    
    return nil
}

// handleStreamEvent processes each stream event
func (p *Processor) handleStreamEvent(ctx context.Context, sessionID, msgID string, event provider.StreamEvent) {
    switch e := event.(type) {
    case provider.ContentBlockStartEvent:
        // Create new part in database
        part := p.createPartFromContentBlock(sessionID, msgID, e.ContentBlock)
        p.db.Parts.Create(ctx, part)
        p.bus.Publish(bus.EventPartCreated, part)
        
    case provider.ContentBlockDeltaEvent:
        // Update existing part with delta
        p.db.Parts.AppendContent(ctx, sessionID, msgID, e.Index, e.Delta)
        p.bus.Publish(bus.EventPartUpdated, map[string]interface{}{
            "session_id": sessionID,
            "index":      e.Index,
            "delta":      e.Delta,
        })
        
    case provider.MessageStopEvent:
        // Mark message complete
        p.db.Messages.MarkComplete(ctx, msgID)
        p.bus.Publish(bus.EventMessageComplete, msgID)
    }
}
```

**Tasks:**
- [ ] Create `internal/session/processor.go`
- [ ] Create `internal/session/converter.go` (message format conversion)
- [ ] Implement `Process()` method
- [ ] Implement `handleStreamEvent()` method
- [ ] Add bus event publishing
- [ ] Write unit tests

---

### Phase 3: Tool Runner (HIGH)

**Goal:** Execute tools during AI processing

**New File:** `internal/tool/runner.go`

```go
package tool

// Runner executes tools and returns results
type Runner struct {
    db        *database.DB
    workspace string
}

// Execute runs a tool and returns the result
func (r *Runner) Execute(ctx context.Context, toolName string, input map[string]any) (string, error) {
    switch toolName {
    case "bash":
        return r.executeBash(ctx, input)
    case "read":
        return r.executeRead(ctx, input)
    case "write":
        return r.executeWrite(ctx, input)
    case "edit":
        return r.executeEdit(ctx, input)
    case "glob":
        return r.executeGlob(ctx, input)
    case "grep":
        return r.executeGrep(ctx, input)
    case "task":
        return r.executeTask(ctx, input)
    case "todowrite":
        return r.executeTodoWrite(ctx, input)
    default:
        return "", fmt.Errorf("unknown tool: %s", toolName)
    }
}

// Tool implementations (matching OpenCode)
func (r *Runner) executeBash(ctx context.Context, input map[string]any) (string, error) {
    command := input["command"].(string)
    // Execute command, capture output
    // Return output
}

func (r *Runner) executeRead(ctx context.Context, input map[string]any) (string, error) {
    filePath := input["filePath"].(string)
    // Read file contents
    // Return formatted content with line numbers
}

func (r *Runner) executeEdit(ctx context.Context, input map[string]any) (string, error) {
    filePath := input["filePath"].(string)
    oldString := input["oldString"].(string)
    newString := input["newString"].(string)
    // Apply edit, return diff
}
```

**Tool Definitions for Provider:**
```go
// File: internal/tool/definitions.go

func GetToolDefinitions() []provider.ToolDefinition {
    return []provider.ToolDefinition{
        {
            Name:        "bash",
            Description: "Execute a bash command",
            InputSchema: bashSchema(),
        },
        {
            Name:        "read",
            Description: "Read a file from the filesystem",
            InputSchema: readSchema(),
        },
        // ... all tools
    }
}
```

**Tasks:**
- [ ] Create `internal/tool/runner.go`
- [ ] Implement bash tool
- [ ] Implement read tool
- [ ] Implement write tool
- [ ] Implement edit tool (with diff output)
- [ ] Implement glob tool
- [ ] Implement grep tool
- [ ] Create tool input schemas
- [ ] Write unit tests

---

### Phase 4: TUI Integration (HIGH)

**Goal:** Connect TUI to processor and display streaming responses

**File:** `internal/tui/app.go`

**4.1 Replace Placeholder with Real Processing**

```go
// Current placeholder:
func (a *App) submitInput() (tea.Model, tea.Cmd) {
    return a, tea.Cmd(func() tea.Msg {
        return ResponseMsg{Content: "placeholder"}
    })
}

// Replace with:
func (a *App) submitInput() (tea.Model, tea.Cmd) {
    a.state.Processing = true
    a.state.SetStatus("Sending message...")
    
    return a, tea.Cmd(func() tea.Msg {
        // Get processor from app
        processor := a.processor
        
        // Build request
        req := session.ProcessRequest{
            SessionID:    a.state.SessionID,
            UserMessage:  a.createUserMessage(),
            Model:        a.state.Local.CurrentModel,
            SystemPrompt: a.getSystemPrompt(),
            History:      a.state.Sync.Messages,
            Tools:        tool.GetToolDefinitions(),
        }
        
        // Process asynchronously
        go processor.Process(context.Background(), req)
        
        // Return initial message created event
        return MessageCreatedMsg{...}
    })
}
```

**4.2 Handle Stream Events**

```go
// Add new message types for stream events
type StreamTextMsg struct {
    SessionID string
    Index     int
    Text      string
}

type StreamToolCallMsg struct {
    SessionID string
    Index     int
    ToolName  string
    ToolID    string
    Arguments string
}

type StreamCompleteMsg struct {
    SessionID string
    MessageID string
}

// Handle in Update()
func (a *App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case StreamTextMsg:
        // Append text to message part
        a.appendToPart(msg.SessionID, msg.Index, msg.Text)
        a.messageViewport.SetContent(a.buildMessagesContent())
        a.scrollToBottom(false)
        
    case StreamToolCallMsg:
        // Add tool call part
        a.addToolCallPart(msg)
        
    case StreamCompleteMsg:
        // Mark message complete
        a.state.Processing = false
        a.scrollToBottom(true)
    }
}
```

**4.3 Subscribe to Bus Events**

```go
// In Init()
func (a *App) Init() tea.Cmd {
    // Subscribe to stream events
    a.bus.Subscribe(bus.EventPartUpdated, func(event bus.Event) {
        // Send tea.Msg through channel
        a.eventChan <- StreamTextMsg{
            SessionID: event.Properties["session_id"],
            Index:     event.Properties["index"],
            Text:      event.Properties["delta"].(provider.TextPart).Text,
        }
    })
    
    return tea.Batch(...)
}
```

**Tasks:**
- [ ] Replace placeholder submitInput
- [ ] Add stream event message types
- [ ] Handle stream events in Update()
- [ ] Subscribe to bus events
- [ ] Update message viewport on text deltas
- [ ] Auto-scroll during streaming
- [ ] Show tool calls in UI
- [ ] Mark complete when done

---

### Phase 5: HTTP Endpoint (MEDIUM)

**Goal:** Add endpoint for async message processing

**File:** `internal/server/routes.go`

```go
// Add route:
POST /session/:id/prompt_async

// Handler:
func (s *Server) handlePromptAsync(c *fiber.Ctx) error {
    sessionID := c.Params("id")
    
    var req PromptRequest
    c.BodyParser(&req)
    
    // Get processor
    processor := s.services.Processor
    
    // Build process request
    processReq := session.ProcessRequest{
        SessionID:    sessionID,
        UserMessage:  buildUserMessage(req),
        Model:        getModelFromReq(req),
        SystemPrompt: getSystemPrompt(),
        History:      getHistory(sessionID),
        Tools:        tool.GetToolDefinitions(),
    }
    
    // Process asynchronously
    go processor.Process(context.Background(), processReq)
    
    // Return immediately (async)
    return c.Status(204).Send()
}
```

**Tasks:**
- [ ] Add `/session/:id/prompt_async` route
- [ ] Implement handler
- [ ] Add SSE endpoint `/session/:id/events`
- [ ] Connect SSE to bus events
            return nil
        case <-s.ctx.Done():
            return nil
        }
    }
}
```

---

## Implementation Order

### Week 1 (HIGH Priority)
1. **Day 1:** Verify Anthropic + OpenAI streaming (already implemented)
2. **Day 2:** Add OpenAI-compatible providers (OpenRouter, Groq, etc.)
3. **Day 3-4:** SessionProcessor service
4. **Day 5:** Tool runner basic implementation
5. **Day 6-7:** TUI integration (submitInput + stream events)

### Week 2 (MEDIUM Priority)
1. **Day 1-3:** Full tool implementations (bash, read, write, edit, glob, grep)
2. **Day 4:** HTTP endpoints (prompt_async, SSE)
3. **Day 5:** Permission system
4. **Day 6-7:** Message format conversion, edge cases, polish

---

## API Endpoints Needed

| Endpoint | Purpose | Priority |
|----------|---------|----------|
| `POST /session/:id/prompt_async` | Send message (async) | HIGH |
| `GET /session/:id/events` | SSE event stream | HIGH |
| `POST /session/:id/message` | Send message (sync) | MEDIUM |
| `POST /permission/:id/respond` | Respond to permission request | LOW |

---

## Testing Strategy

### Unit Tests
- Provider streaming tests (verify Anthropic/OpenAI SSE parsing)
- OpenAI-compatible provider tests
- SessionProcessor tests
- Tool runner tests (bash, read, write, edit, glob, grep)
- Message conversion tests

### Integration Tests
- End-to-end message flow: TUI → Processor → Provider → DB
- Tool execution flow with real commands
- SSE event delivery to TUI

### Manual Testing
```bash
# 1. Set up API key
export ANTHROPIC_API_KEY=your-key

# 2. Start TUI
cd magicode && go run ./cmd/magicode

# 3. Send message and verify:
# - Streaming response appears in real-time
# - Tool calls execute and show results
# - Auto-scroll works correctly
```

---

## Key Differences from OpenCode

| Aspect | OpenCode | MagiCode Approach |
|--------|----------|-------------------|
| **Framework** | Effect + Vercel AI SDK | Go channels + custom streaming |
| **Streaming** | `Stream.Stream<Event>` | `<-chan StreamEvent` (already implemented) |
| **Tools** | AI SDK `tool()` function | Custom ToolRunner with schemas |
| **Events** | Bus + SyncEvent | Bus + SSE (same pattern) |
| **State** | Context services | Struct-based state (simpler) |
| **Providers** | 20+ bundled SDKs | Native + OpenAI-compatible (covers 80%+) |

---

## Files to Create/Modify

### New Files (Need to Create)
| File | Purpose | Phase |
|------|---------|-------|
| `internal/provider/openai_compatible.go` | Bundled OpenAI-compatible providers | 1 |
| `internal/provider/dynamic.go` | Custom providers from OpenCode config | 1 |
| `internal/session/processor.go` | Message processing core | 2 |
| `internal/session/converter.go` | Message format conversion | 2 |
| `internal/tool/runner.go` | Tool execution | 3 |
| `internal/tool/definitions.go` | Tool input schemas | 3 |
| `internal/tool/bash.go` | Bash tool implementation | 3 |
| `internal/tool/read.go` | Read tool implementation | 3 |
| `internal/tool/write.go` | Write tool implementation | 3 |
| `internal/tool/edit.go` | Edit tool implementation | 3 |
| `internal/tool/glob.go` | Glob tool implementation | 3 |
| `internal/tool/grep.go` | Grep tool implementation | 3 |

### Modified Files (Need Updates)
| File | Changes | Phase |
|------|---------|-------|
| `internal/tui/app.go` | Replace submitInput placeholder | 4 |
| `internal/server/routes.go` | Add prompt_async, SSE endpoints | 5 |
| `internal/bus/bus.go` | Add stream event types | 2 |
| `internal/provider/types.go` | May need minor additions | 0 |
| `internal/opencode/config.go` | Add provider registry integration | 1 |

---

## Success Criteria

1. ✅ User sends message in TUI
2. ✅ Message sent to AI provider (Anthropic/OpenAI/OpenRouter/etc.)
3. ✅ Streaming response appears in real-time in TUI
4. ✅ Tool calls execute and show results inline
5. ✅ Assistant message stored in database
6. ✅ Conversation continues with proper history

---

## Estimated Timeline (Updated)

| Phase | Duration | Dependencies | Notes |
|-------|----------|--------------|-------|
| Phase 0 | 0.5 day | None | Verify existing (already done) |
| Phase 1 | 1 day | None | OpenAI-compatible providers |
| Phase 2 | 2 days | None | SessionProcessor |
| Phase 3 | 2 days | Phase 2 | Tool runner |
| Phase 4 | 2 days | Phase 2,3 | TUI integration |
| Phase 5 | 1 day | Phase 2 | HTTP endpoints |

**Total: ~8 days (1.5 weeks)** (reduced from 2 weeks since streaming already exists)

---

## Next Steps

1. **Verify streaming works** - Run provider tests
   ```bash
   go test ./internal/provider/... -v
   ```

2. **Add OpenAI-compatible providers** - Start with OpenRouter (most popular)
   ```bash
   # Create openai_compatible.go
   # Add OpenRouter, Groq, Mistral
   ```

3. **Create SessionProcessor** - Core message processing
   ```bash
   # Create internal/session/processor.go
   # Connect to provider.StreamChat()
   ```

4. **Build tool runner** - Start with bash tool
   ```bash
   # Create internal/tool/runner.go
   # Implement executeBash first
   ```

5. **Replace TUI placeholder** - Connect everything
   ```bash
   # Update submitInput in app.go
   # Add stream event handling
   ```

---

## Questions

1. **Q: Should we support non-OpenAI-compatible providers (Google Gemini, Bedrock)?**
   - A: Defer for now - OpenAI-compatible covers 80%+ of use cases
   - Google/Azure/Bedrock can be added later if needed

2. **Q: How to handle tool results in streaming?**
   - A: When tool_use complete, execute tool, send tool_result back to provider
   - May require multi-turn streaming (OpenCode handles this)

3. **Q: Should SSE be the only transport?**
   - A: Yes for now - SSE is simpler and matches OpenCode's pattern
   - WebSocket could be added later for bidirectional

4. **Q: Permission system - inline vs dialog?**
   - A: Start with inline (simpler) - add dialog for "ask" permissions later