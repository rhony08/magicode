# Critical Bugs - Immediate Fix Required

This document lists the bugs that should be fixed immediately due to high severity and impact.

## CRITICAL (Fix Today)

### 1. Race Condition on streamingState.parts (app.go:2599, 2625, 2663, etc.)
**Severity:** CRITICAL
**Impact:** Data corruption, panics, undefined behavior
**Description:** The `streamingState.parts` map is accessed from multiple goroutines without synchronization:
- Main thread reads from it (line 1239, 2625, 2663, etc.)
- Bus subscription goroutines write to it (via event handlers)

**Fix Required:**
```go
type App struct {
    // ...
    streamingState  streamingState
    streamingMu     sync.Mutex  // Add this
}

// In handleStreamPartCreated:
a.streamingMu.Lock()
a.streamingState.parts[msg.Index] = newPart
a.streamingMu.Unlock()

// In handleStreamPartUpdated, handleStreamPartComplete, etc.:
a.streamingMu.Lock()
// access streamingState.parts
a.streamingMu.Unlock()

// In Update method where parts are accessed:
a.streamingMu.RLock()
for _, part := range a.streamingState.parts {
    // ...
}
a.streamingMu.RUnlock()
```

### 2. Goroutine Leak in subscribeToBus (app.go:250-379)
**Severity:** CRITICAL
**Impact:** Memory exhaustion over time
**Description:** subscribeToBus starts 7+ goroutines that loop forever reading from channels. These are never stopped when the app exits.

**Fix Required:**
Add context cancellation to all subscription goroutines:
```go
func (a *App) subscribeToBus() tea.Cmd {
    if a.busService == nil {
        return nil
    }

    ctx, cancel := context.WithCancel(context.Background())
    a.busCancel = cancel  // Store for Cleanup()

    // Pass ctx to all goroutines and check ctx.Done()
    partUpdatedChan, cleanupPart := a.busService.Subscribe(session.EventPartUpdated)
    go func() {
        defer cleanupPart()
        for {
            select {
            case payload := <-partUpdatedChan:
                if payload == nil {
                    return
                }
                // process payload
            case <-ctx.Done():
                return
            }
        }
    }()
    // ... repeat for other subscriptions
}

func (a *App) Cleanup() {
    if a.busCancel != nil {
        a.busCancel()
    }
}
```

### 3. Ignored Error in buildChatRequest (session/processor.go:265)
**Severity:** CRITICAL
**Impact:** Silent failures, incomplete message context
**Description:** Error from parts.ListByMessage is completely ignored, leading to incomplete message history being sent to AI.

**Fix Required:**
```go
parts, err := p.parts.ListByMessage(context.Background(), msg.ID)
if err != nil {
    p.logger.Error("Failed to load message parts", map[string]interface{}{
        "error": err.Error(),
        "message_id": msg.ID,
    })
    // Continue with empty parts or return error
}
content := p.buildContentFromParts(parts)
```

### 4. Unsafe Type Assertions in Bus Handlers (app.go:260-373)
**Severity:** CRITICAL
**Impact:** Panic on malformed events
**Description:** Type assertions like `props["session_id"].(string)` will panic if the type is wrong.

**Fix Required:**
```go
props, ok := payload.Properties.(map[string]interface{})
if !ok {
    log.Error("Invalid properties type in event")
    return
}

sessionID, ok := props["session_id"].(string)
if !ok {
    log.Error("Invalid session_id type in event")
    return
}

index, ok := props["index"].(int)
if !ok {
    // Try float64 for JSON numbers
    if f, ok := props["index"].(float64); ok {
        index = int(f)
    } else {
        log.Error("Invalid index type in event")
        return
    }
}
```

### 5. Channel Close Race (bus/bus.go:159-161)
**Severity:** CRITICAL
**Impact:** Panic when max subscribers reached
**Description:** Channel is closed while another goroutine might be sending.

**Fix Required:**
```go
// Before closing, ensure no one is sending
func (s *Service) evictOldest(eventType string) {
    s.mu.Lock()
    defer s.mu.Unlock()
    
    subs := s.subscribers[eventType]
    if len(subs) > 0 {
        oldest := subs[0]
        // Signal close via context first, then close channel
        s.subscribers[eventType] = subs[1:]
        close(oldest)
    }
}

// Better approach: use a done channel pattern
type subscriber struct {
    ch     chan Payload
    done   chan struct{}
}
```

## HIGH (Fix This Week)

### 6. HTTP Response Body Leak (provider/openai.go)
**Severity:** HIGH
**Impact:** Resource exhaustion
**Description:** Response body is passed to parseStreamResponse but the ownership transfer isn't clear.

**Fix:**
```go
// In StreamChat:
events := make(chan StreamEvent, 100)
go func() {
    defer resp.Body.Close()  // Ensure closure
    p.parseStreamResponse(resp.Body, events)
}()
```

### 7. Nil Pointer in Tool Complete Handler (app.go:354)
**Severity:** HIGH
**Impact:** Panic
**Description:** Type assertion succeeds but value could be nil.

**Fix:**
```go
if resultPart, ok := props["result_part"].(*database.Part); ok && resultPart != nil {
    msg.ResultPartID = resultPart.ID
}
```

### 8. Event Channel Blocking (app.go:215)
**Severity:** HIGH
**Impact:** Deadlock
**Description:** eventChan has buffer of 100 - writes will block if full.

**Fix:**
```go
eventChan: make(chan tea.Msg, 1000), // Increase buffer
// Or add drop logic:
select {
case a.eventChan <- msg:
default:
    log.Warn("Event channel full, dropping event")
}
```

### 9. SQL Error String Comparison (database/kv.go:37, message.go:67, 334)
**Severity:** HIGH
**Impact:** Incorrect error handling if messages change
**Description:** Using string comparison for sql.ErrNoRows is fragile.

**Fix:**
```go
import "errors"
import "database/sql"

// Instead of:
if err.Error() == "sql: no rows in result set" {

// Use:
if errors.Is(err, sql.ErrNoRows) {
```

## Testing Checklist

After fixing these bugs:

1. [ ] Run `go test -race ./...` - should show no races
2. [ ] Run `go vet ./...` - should show no issues
3. [ ] Run with `-count=100` on tests to catch flaky races
4. [ ] Test with malformed bus events
5. [ ] Test rapid session switching
6. [ ] Test with slow subscribers
7. [ ] Monitor goroutine count: `go test -trace trace.out` then `go tool trace trace.out`

## Verification Commands

```bash
# Check for races
go test -race ./internal/tui/... ./internal/session/... ./internal/bus/... ./internal/database/...

# Check for goroutine leaks
go test -count=1 -timeout=30s ./... 2>&1 | grep -i "leak\|goroutine"

# Static analysis
go vet ./...
staticcheck ./...  # if installed

# Build
go build ./cmd/magicode/...
```
