# PR Review: feat/multi-turn-tool-loops

## Summary
The PR implements multi-turn tool loop support, but has several issues compared to OpenCode's implementation.

## Critical Issues

### 1. Tool Results Format is WRONG
**Location**: `processor.go:224-235`

**Current (Wrong)**:
```go
for _, tr := range result.ToolResults {
    contentMessages = append(contentMessages, provider.ContentMessage{
        Role: provider.RoleUser,
        Content: []provider.ContentPart{
            provider.ToolResultPart{...},
        },
    })
}
```

**Expected by Anthropic**: ONE user message with ALL tool results as content parts:
```json
{
  "role": "user",
  "content": [
    {"type": "tool_result", "tool_use_id": "tool_1", "content": "result 1"},
    {"type": "tool_result", "tool_use_id": "tool_2", "content": "result 2"}
  ]
}
```

**Fix**: Combine all tool results into one user message.

### 2. Missing Iteration Limit
**Location**: `processor.go:183` (for loop)

**Issue**: No maximum iteration limit. Could cause infinite loops if AI keeps calling tools.

**OpenCode**: Uses `agent.steps` limit and `DOOM_LOOP_THRESHOLD = 3` to detect repeated same tool calls.

**Fix**: Add max iteration limit (e.g., 20).

### 3. Assistant Content Should Include Tool Use Parts
**Location**: `processor.go:214-220`

**Issue**: `buildAssistantContent` only includes text and tool_use parts, but should also properly format for the provider.

**OpenCode**: The AI SDK handles this via `convertToModelMessages`.

**Potential Issue**: When sending assistant message back with tool_use parts, the format must match what Anthropic expects.

## Medium Issues

### 4. No Per-Tool Timeout
**Location**: `executeToolFromPart`

**Issue**: Tools can run indefinitely.

**Fix**: Add context.WithTimeout for each tool execution.

### 5. Tool Result Size Truncation
**Location**: Not implemented

**Issue**: Large tool outputs could exceed token limits.

**OpenCode**: Has `truncateToolOutput` function.

**Fix**: Add truncation for large tool results.

### 6. Missing "doom_loop" Detection
**Location**: Not implemented

**Issue**: If AI repeatedly calls same tool with same input, it's stuck in a loop.

**OpenCode**: Detects when last 3 tool calls are identical and prompts user for permission.

## Minor Issues

### 7. Finish Reason Check
**Location**: `processor.go:207`

**Current**: `result.StopReason == "tool_use"`

**Issue**: OpenCode checks `!["tool-calls"].includes(lastAssistant.finish)` meaning multiple valid finish reasons for continuing.

**Anthropic values**: "tool_use", "end_turn", "max_tokens", "stop_sequence"

**Fix**: Should continue if finish is "tool_use" OR there are pending tool calls.

### 8. Content Part Index Tracking
**Location**: `processStreamEventsWithResult`

**Issue**: When continuing loop with new assistant message, part indices start fresh. Need to track correctly.

## Comparison with OpenCode Flow

### OpenCode Flow:
1. `runLoop` has `while (true)` loop
2. Gets messages from DB
3. Checks `lastAssistant.finish` - if NOT "tool-calls" AND no pending tools, break
4. Creates new assistant message
5. Calls `processor.process(streamInput)` which:
   - Streams from LLM
   - Handles events (tool-input-start, tool-call, tool-result)
   - Tools executed by AI SDK internally OR manually via tool-result event
6. Returns "continue" or "stop"
7. If "continue", loop continues

### My PR Flow:
1. `Process()` has `for` loop
2. Builds ContentMessages from history
3. Streams from provider
4. `processStreamEventsWithResult` captures stop_reason and tool results
5. If stop_reason == "tool_use", builds new ContentMessages:
   - Adds assistant message with tool_use parts (WRONG format for multiple tool results)
   - Adds separate user messages for each tool result (WRONG - should be ONE)
6. Creates new assistant message
7. Continues loop

## Recommendations

1. **Fix tool result format immediately** - Critical bug
2. **Add iteration limit** - Prevent infinite loops
3. **Add per-tool timeout** - Prevent hanging
4. **Verify stop_reason handling** - Check all Anthropic finish reasons
5. **Test with real provider** - Mock tests don't catch format issues

## Questions

1. Should tool_use parts include full input when sending back to provider?
2. How to handle tool_result parts that were executed by provider (providerExecuted metadata)?
3. Should we track doom loops like OpenCode?
