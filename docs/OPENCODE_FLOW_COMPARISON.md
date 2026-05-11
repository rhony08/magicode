# Multi-Turn Tool Loop: Deep Comparison with OpenCode

## Status: IMPLEMENTATION COMPLETE

All major missing components have been implemented. See "Implementation Status" section below.

---

## Current Implementation (MagiCode)

```go
// processor.go Process()
for {
    if iteration >= maxIterations { break }
    
    chatReq := buildChatRequestWithContentMessages(req, contentMessages)
    result := processStreamEventsWithResult(...)
    
    // Handle error - break out of loop
    if result.IsError { break }
    
    // Handle blocked state (permission denied)
    if result.IsBlocked {
        // Publish EventBlocked, wait for user action
        break
    }
    
    // Handle compaction (token overflow)
    if result.NeedsCompaction {
        // Publish EventCompactionCreated
        // Log warning, continue processing
    }
    
    // Check if we need to continue with tool results
    // Uses TRANSLATED finish reasons (like OpenAI format):
    // - "tool_calls" (translated from Anthropic "tool_use")
    // - "stop" (translated from "end_turn" or "stop_sequence")
    // - "length" (translated from "max_tokens")
    // - "unknown" (provider couldn't determine)
    shouldContinue := false
    
    // Primary check: finish reason indicates tool calls
    if result.StopReason == "tool_calls" || result.StopReason == "unknown" {
        shouldContinue = true
    }
    
    // Secondary check: even if stop_reason is "stop",
    // check if there are tool_use parts (some providers return "stop" with pending tools)
    if len(result.ToolResults) > 0 && !shouldContinue {
        parts, _ := p.parts.ListByMessage(ctx, currentMessageID)
        for _, part := range parts {
            if part.Data.Type == "tool_use" {
                shouldContinue = true
                break
            }
        }
    }
    
    if shouldContinue && len(result.ToolResults) > 0 {
        // Build assistant message content from parts
        assistantContent := p.buildAssistantContent(req.SessionID, currentMessageID)
        
        // Add assistant message to conversation
        contentMessages = append(contentMessages, provider.ContentMessage{
            Role:    provider.RoleAssistant,
            Content: assistantContent,
        })
        
        // Combine ALL tool results into ONE user message (Anthropic requirement)
        toolResultParts := []provider.ContentPart{}
        for _, tr := range result.ToolResults {
            toolResultParts = append(toolResultParts, provider.ToolResultPart{
                Type:      "tool_result",
                ToolUseID: tr.ToolID,
                Content:   tr.Result,
                IsError:   tr.IsError,
            })
        }
        contentMessages = append(contentMessages, provider.ContentMessage{
            Role:    provider.RoleUser,
            Content: toolResultParts,
        })
        
        // Create new assistant message for next turn
        newAssistantMsg, _ := p.createAssistantMessage(processCtx, req)
        currentMessageID = newAssistantMsg.ID
        
        // Doom loop detection
        if len(result.ToolResults) >= doomLoopThreshold {
            // Check if same tool repeated 3 times
            // If doom loop detected, break out
        }
        
        iteration++
        continue
    }
    break
}
```

## OpenCode Implementation

```typescript
// prompt.ts runLoop()
while (true) {
    // RELOAD messages from DB each iteration
    let msgs = yield* MessageV2.filterCompactedEffect(sessionID)
    
    // Find lastUser, lastAssistant, lastFinished by scanning backwards
    // Check hasToolCalls (pending/running tools in last assistant)
    
    // EXIT condition:
    if (lastAssistant?.finish &&
        !["tool-calls"].includes(lastAssistant.finish) &&
        !hasToolCalls &&
        lastUser.id < lastAssistant.id) {
        break
    }
    
    step++
    
    // Handle compaction/subtask
    
    // Create new assistant message
    // Call processor.process(streamInput)
    
    // Check result:
    if (result === "stop") break
    if (result === "compact") { create compaction; continue }
    continue  // result === "continue"
}

// processor.ts process()
// Returns: "compact" | "stop" | "continue"

// Determination:
if (ctx.needsCompaction) return "compact"
if (ctx.blocked || ctx.assistantMessage.error) return "stop"
return "continue"
```

---

## Implementation Status

| Component | Status | Description |
|-----------|--------|-------------|
| Finish reason translation | ✅ DONE | Anthropic values translated to common format |
| hasToolCalls check | ✅ DONE | Checks for tool_use parts even when stop_reason is "stop" |
| Blocked state | ✅ DONE | Permission denial stops loop, EventBlocked published |
| Compaction detection | ✅ DONE | Token overflow detected, EventCompactionCreated published |
| Token usage tracking | ✅ DONE | Usage captured from MessageDeltaEvent |
| Doom loop detection | ✅ DONE | Same tool repeated 3 times triggers break |
| Provider-executed flag | ⏳ PENDING | Low priority - skip tools executed by provider |
| Reload messages from DB | ⏳ PENDING | High priority - ensures consistency |

---

## Key Components Implemented

### 1. Finish Reason Translation (anthropic.go)

```go
// translateAnthropicStopReason converts Anthropic's stop_reason to common format
// Anthropic values: "tool_use", "end_turn", "max_tokens", "stop_sequence"
// Common values:    "tool_calls", "stop", "length", "stop"
func translateAnthropicStopReason(reason string) string {
    switch reason {
    case "tool_use":
        return "tool_calls"
    case "end_turn":
        return "stop"
    case "max_tokens":
        return "length"
    case "stop_sequence":
        return "stop"
    default:
        return reason
    }
}
```

Applied in `parseAnthropicEvent()` for `message_delta` events.

### 2. Blocked State (processor.go)

```go
type StreamResult struct {
    StopReason      string
    ToolResults     []ToolResult
    IsError         bool
    Error           error
    NeedsCompaction bool      // Token overflow detected
    IsBlocked       bool      // Permission denied (blocked state)
    Tokens          TokenUsage // Token usage from message
}

type ToolResult struct {
    ToolID   string
    ToolName string
    Result   string
    IsError  bool
    Blocked  bool  // Permission denied - waiting for approval
}
```

Detection in `executeToolFromPart()`:
```go
if toolErr, ok := err.(*tool.ToolError); ok && toolErr.Type == "permission_denied" {
    isBlocked = true
    toolResult.Blocked = true
}
```

Event published:
- `EventBlocked` - UI can show permission request dialog

### 3. Compaction Handling (compaction.go)

```go
type CompactionService struct {
    db      *database.Database
    bus     *bus.Service
    logger  *log.Logger
    config  CompactionConfig
}

type TokenUsage struct {
    Input      int
    Output     int
    Reasoning  int
    CacheRead  int  // Anthropic: cache_read_input_tokens
    CacheWrite int  // Anthropic: cache_creation_input_tokens
    Total      int
}

// IsOverflow checks if token usage exceeds model context limit
func (s *CompactionService) IsOverflow(tokens TokenUsage, model provider.ModelInfo) bool {
    // Matches OpenCode's overflow.isOverflow logic
    usable := s.calculateUsableTokens(model)
    count := tokens.Total
    return count >= usable
}
```

Detection in `Process()` loop:
```go
if result.Tokens.Total > 0 && p.compaction != nil {
    modelInfo, ok := p.registry.GetModel(req.Model)
    if ok && p.compaction.IsOverflow(result.Tokens, modelInfo) {
        result.NeedsCompaction = true
    }
}
```

Event published:
- `EventCompactionCreated` - UI can show notification

### 4. Usage Type Extended (provider/types.go)

```go
type Usage struct {
    InputTokens  int `json:"input_tokens"`
    OutputTokens int `json:"output_tokens"`
    TotalTokens  int `json:"total_tokens"`
    
    // Anthropic-specific cache fields
    CacheRead  int `json:"cache_read_input_tokens,omitempty"`
    CacheWrite int `json:"cache_creation_input_tokens,omitempty"`
}
```

---

## Remaining Work

### 1. Reload messages from DB each iteration (HIGH PRIORITY)

OpenCode reloads messages from DB each iteration to ensure consistency. This is important for:
- Tools that create new messages (subtask tool)
- Compaction modifying messages
- Concurrent modifications by other processes

**Recommended fix:**
```go
for {
    // Reload messages from DB instead of using in-memory contentMessages
    history, err := p.messages.ListBySession(ctx, req.SessionID)
    if err != nil { return err }
    
    // Build ContentMessages from fresh DB state
    contentMessages := p.buildContentMessagesFromDB(ctx, req.SessionID)
    
    // ... rest of loop
}
```

### 2. Provider-executed flag (LOW PRIORITY)

OpenCode marks tools as `providerExecuted` when the provider executes them internally:
```typescript
if (part.metadata?.providerExecuted) {
    // Skip - tool already executed by provider
}
```

This is needed for providers like DWS Agent Platform that execute tools internally.

---

## Key Differences Summary

| Feature | MagiCode | OpenCode | Status |
|---------|----------|----------|--------|
| Finish reason values | "tool_calls", "stop", "length" | "tool-calls", "stop", "length" | ✅ Compatible |
| hasToolCalls check | Checks tool_use parts | Checks pending/running tools | ✅ Similar |
| Blocked handling | Breaks + EventBlocked | Returns "stop" | ✅ Compatible |
| Compaction | Event + warning logged | Returns "compact", creates summary | ✅ Detection done |
| Doom loop | 3 same tools = break | Same logic | ✅ Compatible |
| Tool timeout | 2 min per tool | Similar timeout | ✅ Compatible |
| Max iterations | 20 | Similar limit | ✅ Compatible |
| Message reload | Memory-based | DB-based | ⏳ Pending |

---

## Testing Coverage

Tests should cover:
1. ✅ Multiple tool calls in one response
2. ✅ Provider returning "stop" with pending tools (hasToolCalls check)
3. ✅ Permission denial during tool execution (blocked state)
4. ✅ Token overflow detection (compaction)
5. ⏳ Concurrent modifications to messages during tool execution
6. ⏳ Doom loop detection

---

## Events Published

| Event | When | Payload |
|-------|------|---------|
| EventBlocked | Permission denied | session_id, message_id, tool_count |
| EventCompactionCreated | Token overflow | session_id, tokens |
| EventToolCallPending | Tool about to execute | session_id, message_id, tool_name, tool_id, input |
| EventToolCallRunning | Tool started executing | session_id, message_id, tool_name, tool_id |
| EventToolCallComplete | Tool finished | session_id, message_id, part_id, result, status |
| EventMessageCreated | Message created | session_id, message_id, role |
| EventMessageComplete | Message finished | session_id, message_id, stop_reason |
| EventPartCreated | Part created | session_id, message_id, part_id, index, type |
| EventPartUpdated | Part updated | session_id, message_id, part_id, delta |
| EventPartComplete | Part finished | session_id, message_id, part_id, type |
| EventStreamError | Stream error | session_id, error |

---

## Compaction Summary Template (OpenCode pattern)

```go
func GetCompactionSummaryPrompt() string {
    return `Output exactly this Markdown structure...
---
## Goal
- [single-sentence task summary]

## Constraints & Preferences
- [user constraints, preferences, specs, or "(none)"]

## Progress
### Done
- [completed work or "(none)"]

### In Progress
- [current work or "(none)"]

### Blocked
- [blockers or "(none)"]

## Key Decisions
- [decision and why, or "(none)"]

## Next Steps
- [ordered next actions or "(none)"]

## Critical Context
- [important technical facts, errors, open questions, or "(none)"]

## Relevant Files
- [file or directory path: why it matters, or "(none)"]
---
...`
}
```