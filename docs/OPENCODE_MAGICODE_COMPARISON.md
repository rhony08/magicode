# Session/Message Processing Flow Comparison: OpenCode vs MagiCode

## 1. Architecture Overview

### OpenCode Architecture (TypeScript/Effect)

```
┌─────────────────────────────────────────────────────────────────────┐
│                         OpenCode Architecture                         │
├─────────────────────────────────────────────────────────────────────┤
│                                                                       │
│  ┌─────────────┐    ┌─────────────┐    ┌─────────────────────┐       │
│  │ SessionPrompt│───▶│ SessionProcessor│───▶│ LLM Service          │       │
│  │ (prompt.ts) │    │ (processor.ts)│    │ (llm.ts)             │       │
│  └─────────────┘    └─────────────┘    └─────────────────────┘       │
│        │                  │                      │                   │
│        ▼                  ▼                      ▼                   │
│  ┌─────────────┐    ┌─────────────┐    ┌─────────────────────┐       │
│  │ MessageV2   │    │ ToolRegistry│    │ Provider Service    │       │
│  │ (message-v2)│    │ (registry.ts)│    │ (provider.ts)       │       │
│  └─────────────┘    └─────────────┘    └─────────────────────┘       │
│        │                  │                      │                   │
│        ▼                  ▼                      ▼                   │
│  ┌─────────────┐    ┌─────────────┐    ┌─────────────────────┐       │
│  │ Compaction  │    │ Permission  │    │ AI SDK (streamText) │       │
│  │ (compaction)│    │ Service     │    │                      │       │
│  └─────────────┘    └─────────────┘    └─────────────────────┘       │
│                                                                       │
│  Key Patterns:                                                        │
│  - Effect.ts functional programming with typed effects               │
│  - Service-based dependency injection via Context.Service            │
│  - InstanceState for per-directory/project state                     │
│  - Stream-based event processing (effect/Stream)                     │
│  - Deferred for async coordination (tool call completion)            │
│                                                                       │
└─────────────────────────────────────────────────────────────────────┘
```

### MagiCode Architecture (Go)

```
┌─────────────────────────────────────────────────────────────────────┐
│                         MagiCode Architecture                         │
├─────────────────────────────────────────────────────────────────────┤
│                                                                       │
│  ┌─────────────┐    ┌─────────────┐    ┌─────────────────────┐       │
│  │ Processor   │───▶│ AnthropicProvider│───▶│ HTTP Client          │       │
│  │ (processor.go)│    │ (anthropic.go)│    │                      │       │
│  └─────────────┘    └─────────────┘    └─────────────────────┘       │
│        │                  │                      │                   │
│        ▼                  ▼                      ▼                   │
│  ┌─────────────┐    ┌─────────────┐    ┌─────────────────────┐       │
│  │ MessageStorage│    │ ToolRegistry│    │ ProviderRegistry    │       │
│  │ (message.go) │    │ (registry.go)│    │ (provider.go)       │       │
│  └─────────────┘    └─────────────┘    └─────────────────────┘       │
│        │                  │                      │                   │
│        ▼                  ▼                      ▼                   │
│  ┌─────────────┐    ┌─────────────┐    ┌─────────────────────┐       │
│  │ PartStorage │    │ Bus Service │    │ SSE Parser           │       │
│  │ (part.go)   │    │ (bus.go)    │    │                      │       │
│  └─────────────┘    └─────────────┘    └─────────────────────┘       │
│                                                                       │
│  Key Patterns:                                                        │
│  - Go interfaces for abstraction                                     │
│  - Channel-based streaming (<-chan StreamEvent)                      │
│  - Context-based cancellation                                        │
│  - sync.Mutex for thread safety                                      │
│  - Struct-based data models                                          │
│                                                                       │
└─────────────────────────────────────────────────────────────────────┘
```

---

## 2. Flow Comparison (Step by Step)

### 2.1 User Message Creation

| Step | OpenCode (prompt.ts) | MagiCode (processor.go) | Gap Analysis |
|------|---------------------|------------------------|--------------|
| **Input parsing** | `PromptInput` schema with parts (text, file, agent, subtask) | `ProcessRequest` with simple `UserMessage` string | **Major Gap**: MagiCode lacks part type diversity |
| **Part resolution** | Complex resolution: file reading, MCP resources, agent parts | Simple text part creation | **Major Gap**: No file/agent/subtask part handling |
| **File handling** | Uses Read tool for text files, base64 encoding for binary | Not implemented | **Gap**: No file attachment support |
| **MCP resources** | Calls `mcp.readResource()` for MCP-provided files | Not implemented | **Gap**: No MCP integration |
| **Validation** | Zod schema validation with detailed error messages | Basic struct validation | **Minor Gap**: Less robust validation |

**OpenCode Flow:**
```typescript
// prompt.ts createUserMessage()
const info: MessageV2.User = {
  id: input.messageID ?? MessageID.ascending(),
  role: "user",
  sessionID: input.sessionID,
  agent: ag.name,
  model: { providerID, modelID, variant },
}

// Resolve parts (file, agent, subtask types)
const parts = yield* Effect.forEach(input.parts, resolvePart)
yield* sessions.updateMessage(info)
for (const part of parts) yield* sessions.updatePart(part)
```

**MagiCode Flow:**
```go
// processor.go createUserMessage()
msg := database.Message{
  SessionID: req.SessionID,
  Data: database.MessageInfo{
    Role:    "user",
    Agent:   req.Agent,
    ModelID: string(req.Model),
  },
}
part := database.Part{
  MessageID: msg.ID,
  Data: database.PartData{
    Type: "text",
    Text: req.UserMessage, // Only text parts!
  },
}
```

### 2.2 Assistant Message Processing

| Step | OpenCode (processor.ts) | MagiCode (processor.go) | Gap Analysis |
|------|-------------------------|------------------------|--------------|
| **Pre-snapshot** | Captures filesystem snapshot before streaming | Not implemented | **Gap**: No snapshot tracking |
| **Event handling** | Comprehensive handler for all event types | Basic handling (text, tool_use, tool_result) | **Gap**: Missing reasoning/step events |
| **Reasoning blocks** | Full support with `reasoning-start/delta/end` events | Not implemented | **Major Gap**: No thinking/reasoning support |
| **Step tracking** | `step-start/step-finish` for multi-step reasoning | Not implemented | **Gap**: No step tracking |
| **Tool call coordination** | `Deferred` for async completion tracking | Basic synchronous execution | **Minor Gap**: Less sophisticated coordination |
| **Patch tracking** | Filesystem diff patches after tool execution | Not implemented | **Gap**: No file change tracking |
| **Error handling** | Typed errors (ContextOverflowError, APIError, etc.) | Basic error strings | **Gap**: Less structured error handling |

**OpenCode Event Handling:**
```typescript
// processor.ts handleEvent()
case "reasoning-start":
  ctx.reasoningMap[value.id] = {
    type: "reasoning",
    text: "",
    time: { start: Date.now() },
  }
  yield* session.updatePart(ctx.reasoningMap[value.id])

case "step-start":
  yield* session.updatePart({
    type: "step-start",
    snapshot: ctx.snapshot,
  })

case "finish-step":
  // Track usage, tokens, cost
  // Generate patch for file changes
  // Trigger summary generation
```

**MagiCode Event Handling:**
```go
// processor.go processStreamEventsWithResult()
case provider.ContentBlockStartEvent:
  part := p.createPartFromContentBlock(...)
  // Only handles "text" and "tool_use" types

case provider.MessageDeltaEvent:
  result.StopReason = e.Delta.StopReason
  result.Tokens = TokenUsage{...}
  // Basic token tracking only
```

### 2.3 Tool Execution

| Feature | OpenCode | MagiCode | Gap Analysis |
|---------|----------|----------|--------------|
| **Tool types** | 16+ tools (bash, read, write, edit, glob, grep, task, todo, webfetch, websearch, codesearch, skill, question, lsp, plan, patch) | 8 tools (bash, read, write, edit, glob, grep, webfetch, permission) | **Major Gap**: Missing 8+ tools |
| **Tool context** | Rich context (sessionID, messageID, abort, callID, messages, metadata callback, permission ask) | Basic context (sessionID, messageID, abort, callID) | **Gap**: Missing messages, metadata callback, permission |
| **Permission system** | Full permission service with rulesets, ask/deny/allow | Basic permission denied error | **Major Gap**: No permission system |
| **Tool output truncation** | Truncate service for large outputs | Basic truncation | **Minor Gap**: Less sophisticated truncation |
| **Attachments** | Tool results can return file attachments | Basic string output only | **Gap**: No attachment support |
| **Task tool (subagents)** | Full subagent support with TaskTool | Not implemented | **Major Gap**: No subagent support |
| **Plugin tools** | Plugin-defined tools with custom execution | Not implemented | **Gap**: No plugin system |

**OpenCode Tool Registry:**
```typescript
// registry.ts - 16+ built-in tools
builtin: [
  tool.question, tool.bash, tool.read, tool.glob, tool.grep,
  tool.edit, tool.write, tool.task, tool.fetch, tool.todo,
  tool.search, tool.code, tool.skill, tool.patch, tool.lsp, tool.plan
]

// Dynamic plugin tools
const custom: Tool.Def[] = []
for (const p of plugins) {
  for (const [id, def] of Object.entries(p.tool ?? {})) {
    custom.push(fromPlugin(id, def))
  }
}
```

**MagiCode Tool Registry:**
```go
// types.go - 8 defined tool IDs
const (
  ToolBash, ToolRead, ToolWrite, ToolEdit, ToolGrep,
  ToolGlob, ToolWebFetch, ToolWebSearch, ToolLSP, ToolTask,
  ToolQuestion, ToolTodo, ToolPlan
)

// But only 8 actually implemented in separate files
```

### 2.4 Streaming Loop (Multi-turn)

| Feature | OpenCode | MagiCode | Gap Analysis |
|---------|----------|----------|--------------|
| **Loop detection** | `DOOM_LOOP_THRESHOLD = 3` with exact tool call comparison | `doomLoopThreshold = 3` but less sophisticated | **Minor Gap**: Similar but OpenCode more robust |
| **Finish reason handling** | Checks `["tool-calls", "unknown"]` to continue | Checks `"tool_calls"` and `"unknown"` | **Match**: Both handle correctly |
| **Tool result message** | Creates user message for tool results (Anthropic API requirement) | Creates user message with tool_result parts | **Match**: Both implement correctly |
| **Message reload** | Reloads messages from DB each iteration | Reloads messages from DB each iteration | **Match**: Both follow OpenCode pattern |
| **Blocked state** | Permission denied causes blocked state, waits for approval | Permission denied marks blocked but no approval flow | **Gap**: No approval flow |
| **Compaction trigger** | Checks `isOverflow()` after each step, triggers compaction | Checks `IsOverflow()` and publishes event | **Minor Gap**: Similar but less integration |

**OpenCode Loop Logic:**
```typescript
// prompt.ts runLoop()
while (true) {
  // Check if should exit
  if (lastAssistant?.finish && !["tool-calls"].includes(lastAssistant.finish) && !hasToolCalls) {
    break
  }
  
  // Check for overflow
  if (yield* compaction.isOverflow({ tokens: lastFinished.tokens, model })) {
    yield* compaction.create({ sessionID, auto: true })
    continue
  }
  
  // Handle subtasks
  if (task?.type === "subtask") {
    yield* handleSubtask({ task, model, lastUser, sessionID, session, msgs })
    continue
  }
}
```

**MagiCode Loop Logic:**
```go
// processor.go Process()
for {
  if iteration >= maxIterations { break }
  
  // Reload messages from DB
  if iteration > 0 {
    contentMessages = p.buildContentMessagesFromDB(processCtx, req.SessionID)
  }
  
  // Stream and process
  result := p.processStreamEventsWithResult(...)
  
  // Check compaction
  if result.NeedsCompaction { ... }
  
  // Check if should continue
  if result.StopReason == "tool_calls" { shouldContinue = true }
  
  // Create tool result message
  toolResultMsg, err := p.createToolResultUserMessage(...)
}
```

---

## 3. Data Structures Comparison

### 3.1 Message Types

| Type | OpenCode (message-v2.ts) | MagiCode (schema.go) | Gap |
|------|--------------------------|---------------------|-----|
| **User message** | `User` with role, time, agent, model, system, tools, format, summary | `MessageInfo` with role, agent, modelID | Missing: system, tools, format, summary |
| **Assistant message** | `Assistant` with error, tokens, cost, path, summary, structured, finish, variant | `MessageInfo` with role, modelID, cost, tokens, finish | Missing: error types, path, summary, variant |

### 3.2 Part Types

| Part Type | OpenCode | MagiCode | Status |
|-----------|----------|----------|--------|
| `text` | Full support with synthetic, ignored, metadata | Basic text field | **Partial** |
| `reasoning` | Full support with metadata, time | Not implemented | **Missing** |
| `file` | Full support with mime, url, source, filename | Basic file fields | **Partial** |
| `tool_use` | Full support via ToolPart | Basic tool_use type | **Partial** |
| `tool_result` | Implicit (stored in ToolPart state) | Explicit tool_result type | **Different approach** |
| `step-start` | Full support with snapshot | Not implemented | **Missing** |
| `step-finish` | Full support with tokens, cost, reason | Not implemented | **Missing** |
| `patch` | Full support with hash, files | Not implemented | **Missing** |
| `compaction` | Full support with auto, overflow, tail_start_id | Not implemented | **Missing** |
| `subtask` | Full support for subagent invocation | Not implemented | **Missing** |
| `agent` | Full support for agent references | Not implemented | **Missing** |
| `retry` | Full support with attempt, error | Not implemented | **Missing** |
| `snapshot` | Full support | Not implemented | **Missing** |

### 3.3 Tool State

| State | OpenCode | MagiCode | Status |
|-------|----------|----------|--------|
| `pending` | ToolStatePending with input, raw | Basic pending status | **Match** |
| `running` | ToolStateRunning with title, metadata, time | Basic running status | **Partial** - missing title/metadata |
| `completed` | ToolStateCompleted with output, title, metadata, time, attachments | Basic complete status | **Partial** - missing attachments |
| `error` | ToolStateError with error message | Basic error status | **Match** |

---

## 4. Missing Features in MagiCode

### 4.1 Critical Missing Features

1. **Reasoning/Thinking Support**
   - OpenCode handles `reasoning-start/delta/end` events
   - MagiCode has no reasoning part type
   - Impact: Cannot display Claude thinking blocks

2. **Step Tracking**
   - OpenCode tracks multi-step reasoning with `step-start/step-finish`
   - MagiCode has no step events
   - Impact: No step-by-step progress visualization

3. **Subagent/Task Tool**
   - OpenCode has full TaskTool for spawning subagents
   - MagiCode defines ToolTask but doesn't implement
   - Impact: Cannot delegate tasks to specialized agents

4. **Permission System**
   - OpenCode has full Permission service with ask/deny/allow rulesets
   - MagiCode has basic permission denied error only
   - Impact: No user approval flow for dangerous operations

5. **Compaction System**
   - OpenCode has full CompactionService with prune, process, create
   - MagiCode has basic IsOverflow check only
   - Impact: No automatic context summarization

6. **File Patch Tracking**
   - OpenCode tracks file changes via snapshot/patch
   - MagiCode has no filesystem change tracking
   - Impact: No undo/revert capabilities

### 4.2 Missing Tools

| Tool | OpenCode | MagiCode Status |
|------|----------|-----------------|
| `task` | Full implementation for subagents | Defined but not implemented |
| `todo` | TodoWriteTool for task tracking | Defined but not implemented |
| `question` | QuestionTool for user questions | Defined but not implemented |
| `codesearch` | CodeSearchTool (Exa integration) | Not defined |
| `websearch` | WebSearchTool (Exa integration) | Defined but not implemented |
| `skill` | SkillTool for loading skill instructions | Not defined |
| `lsp` | LspTool for LSP operations | Defined but not implemented |
| `plan` | PlanExitTool for plan mode | Defined but not implemented |
| `apply_patch` | ApplyPatchTool for git patches | Not defined |

### 4.3 Missing Provider Features

| Feature | OpenCode | MagiCode | Gap |
|---------|----------|----------|-----|
| **Multi-provider** | 18+ providers (Anthropic, OpenAI, Bedrock, Google, Azure, etc.) | Only Anthropic + OpenAI-compatible | **Major** |
| **Provider-specific transforms** | ProviderTransform for schema/output adaptation | Basic translation | **Major** |
| **Custom model loaders** | Per-provider model loading customization | Not implemented | **Major** |
| **Model variants** | Support for thinking/reasoning variants | Not implemented | **Gap** |
| **Plugin providers** | Plugin-defined providers | Not implemented | **Gap** |

### 4.4 Missing Event Types

| Event | OpenCode | MagiCode |
|-------|----------|----------|
| `message.part.delta` | Real-time text delta updates | Not emitted |
| `session.compacted` | Compaction completion | Not emitted |
| `experimental.text.complete` | Text processing hook | Not supported |

---

## 5. Implementation Recommendations

### 5.1 High Priority (Critical for functionality)

#### 1. Implement Reasoning Support

```go
// Add to PartData in schema.go
type PartData struct {
  // ... existing fields
  ReasoningText string `json:"reasoning_text,omitempty"` // For reasoning type
  ReasoningTime *ReasoningTime `json:"reasoning_time,omitempty"`
}

type ReasoningTime struct {
  Start int64 `json:"start"`
  End   int64 `json:"end,omitempty"`
}

// Add event handling in processor.go
case "reasoning_start":
  part := database.Part{
    Data: database.PartData{
      Type: "reasoning",
      ReasoningText: "",
      ReasoningTime: &ReasoningTime{Start: time.Now().UnixMilli()},
    },
  }
  // Store in reasoningMap

case "reasoning_delta":
  // Append text to reasoning part
  // Update delta in DB

case "reasoning_end":
  // Finalize reasoning part
```

#### 2. Implement Permission System

```go
// Add PermissionService in internal/permission/service.go
type PermissionService struct {
  bus    *bus.Service
  pending map[string]*PermissionRequest
}

type PermissionRequest struct {
  ID         string
  SessionID  string
  Permission string
  Patterns   []string
  Metadata   map[string]interface{}
  Ruleset    []PermissionRule
  Response   chan PermissionResponse
}

type PermissionRule struct {
  Permission string `json:"permission"`
  Action     string `json:"action"` // "allow", "deny", "ask"
  Pattern    string `json:"pattern"`
}

// Ask method sends request and waits for approval
func (s *PermissionService) Ask(ctx context.Context, req PermissionRequest) (PermissionResponse, error) {
  req.Response = make(chan PermissionResponse, 1)
  s.pending[req.ID] = &req
  s.bus.Publish(EventPermissionRequest, req)
  
  select {
  case resp := <-req.Response:
    return resp, nil
  case <-ctx.Done():
    return PermissionResponse{Approved: false}, ctx.Err()
  }
}
```

#### 3. Implement TaskTool (Subagents)

```go
// Add to internal/tool/task.go
type TaskTool struct {
  processor *Processor
  agents    *AgentRegistry
}

func (t *TaskTool) Execute(ctx context.Context, params map[string]interface{}, toolCtx ToolContext) (*ToolResult, error) {
  agentName := params["subagent_type"].(string)
  prompt := params["prompt"].(string)
  
  // Get agent configuration
  agent, ok := t.agents.Get(agentName)
  if !ok {
    return nil, NewValidationError(ToolTask, "agent not found: "+agentName)
  }
  
  // Create sub-session
  subSession := CreateSubSession(toolCtx.SessionID, agentName)
  
  // Process with agent's model
  result, err := t.processor.Process(ctx, ProcessRequest{
    SessionID:   subSession.ID,
    UserMessage: prompt,
    Model:       agent.Model,
    Agent:       agentName,
  })
  
  return &ToolResult{
    Output: result.Output,
    Title:  agentName + " completed",
  }, err
}
```

### 5.2 Medium Priority (Enhanced functionality)

#### 4. Implement Compaction Process

```go
// Extend CompactionService in compaction.go
func (s *CompactionService) Process(ctx context.Context, sessionID string, messages []database.Message) error {
  // 1. Select messages to summarize
  selected := s.selectMessages(messages)
  
  // 2. Build summary prompt
  prompt := GetCompactionSummaryPrompt()
  
  // 3. Call AI to generate summary
  summary, err := s.generateSummary(ctx, selected, prompt)
  if err != nil {
    return err
  }
  
  // 4. Create compaction message
  compactionMsg := database.Message{
    SessionID: sessionID,
    Data: database.MessageInfo{Role: "user"},
  }
  compactionPart := database.Part{
    Data: database.PartData{
      Type:         "compaction",
      Auto:         true,
      TailStartID:  selected.TailStartID,
    },
  }
  
  // 5. Store in database
  s.db.CreateMessage(ctx, compactionMsg)
  s.db.CreatePart(ctx, compactionPart)
  
  return nil
}
```

#### 5. Implement Step Tracking

```go
// Add step events in processor.go
type StepInfo struct {
  Snapshot string
  Tokens   TokenUsage
  Cost     float64
  Reason   string
}

// Track in ProcessorContext
type ProcessorContext struct {
  // ... existing
  currentStep *StepInfo
  stepCount   int
}

case "step_start":
  ctx.currentStep = &StepInfo{
    Snapshot: captureSnapshot(),
  }
  
case "step_finish":
  ctx.currentStep.Tokens = usage
  ctx.currentStep.Cost = cost
  ctx.currentStep.Reason = reason
  // Create step-finish part
```

#### 6. Add Missing Tools

```go
// Implement TodoWriteTool in internal/tool/todo.go
// Implement QuestionTool in internal/tool/question.go
// Implement WebSearchTool in internal/tool/websearch.go
// Implement CodeSearchTool in internal/tool/codesearch.go
```

### 5.3 Lower Priority (Nice to have)

#### 7. Plugin System

```go
// Add PluginService in internal/plugin/service.go
type Plugin struct {
  ID       string
  Tool     map[string]ToolDefinition
  Provider map[string]ProviderDefinition
  Auth     *PluginAuth
}

type PluginService struct {
  plugins []Plugin
  loader  PluginLoader
}

func (s *PluginService) Load(dir string) error {
  // Scan for plugin files (JS/TS/WASM)
  // Load tool definitions
  // Register with ToolRegistry
}
```

#### 8. Multi-provider Support

```go
// Add providers in internal/provider/
// amazon-bedrock.go
// google.go
// azure.go
// openrouter.go
// xai.go
// mistral.go
// groq.go
```

#### 9. Snapshot/Patch Tracking

```go
// Add SnapshotService in internal/snapshot/service.go
type SnapshotService struct {
  watcher fsnotify.Watcher
}

func (s *SnapshotService) Track() (string, error) {
  // Capture current file state
  // Return snapshot ID
}

func (s *SnapshotService) Patch(snapshotID string) (Patch, error) {
  // Compare current to snapshot
  // Return file diffs
}
```

---

## 6. Summary of Critical Gaps

| Area | Gap Severity | Implementation Effort | Priority |
|------|-------------|----------------------|----------|
| Reasoning support | **Critical** | Medium | P0 |
| Permission system | **Critical** | High | P0 |
| Task/subagent tool | **Critical** | High | P0 |
| Compaction process | **High** | Medium | P1 |
| Step tracking | **High** | Low | P1 |
| Todo/Question tools | **Medium** | Low | P2 |
| Multi-provider | **High** | Very High | P2 |
| Plugin system | **Medium** | High | P3 |
| Snapshot tracking | **Medium** | Medium | P3 |

---

## 7. Code Mapping Reference

### Key Files Comparison

| OpenCode File | MagiCode Equivalent | Notes |
|---------------|--------------------|-------|
| `session/prompt.ts` | `internal/session/processor.go` | Different architecture (prompt.ts creates messages, processor.go does both) |
| `session/processor.ts` | `internal/session/processor.go` | Similar role but processor.go simpler |
| `session/message-v2.ts` | `internal/database/schema.go` | OpenCode has 12+ part types, MagiCode has 5 |
| `session/schema.ts` | `internal/database/schema.go` | ID generation differs |
| `session/compaction.ts` | `internal/session/compaction.go` | OpenCode full implementation, MagiCode basic |
| `session/overflow.ts` | `internal/session/compaction.go` | IsOverflow logic similar |
| `session/llm.ts` | `internal/provider/anthropic.go` | OpenCode uses AI SDK, MagiCode uses HTTP |
| `provider/provider.ts` | `internal/provider/provider.go` | OpenCode 18+ providers, MagiCode 2 |
| `tool/registry.ts` | `internal/tool/registry.go` | Similar structure |
| `tool/tool.ts` | `internal/tool/types.go` | Similar interface patterns |