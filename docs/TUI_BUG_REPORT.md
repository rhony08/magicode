# TUI Bug Report and Fix Plan

**Date:** 2026-04-25  
**Status:** Critical Bugs Found  
**Related:** See `FLOW_COMPARISON_REVIEW.md` for architectural differences

---

## Bug #1: Ctrl+P Shortcut Not Working

### Severity: HIGH (UX Blocking)

### Description
The UI claims `Ctrl+P` opens a "command palette" but the feature is NOT implemented.

### Evidence

1. **UI falsely claims Ctrl+P works** in multiple places:
   - `internal/tui/dialog/help.go` line 121: `{Key: "Ctrl+P", Description: "Command palette"}`
   - `internal/tui/layout/prompt.go` line 245: `"Enter submit  ·  Ctrl+P palette  ·  Tab cycle"`
   - `internal/tui/layout/footer.go` line 292: `{Key: "Ctrl+P", Text: "palette"}`

2. **DialogCommand type exists but unused**:
   - `internal/tui/types/types.go` line 221: `DialogCommand DialogType = "command"`
   - But no `dialog/command.go` file exists!

3. **No Ctrl+P keybinding handler**:
   - `internal/tui/keybindings.go`: No `Palette` or `Command` keybinding defined
   - `internal/tui/leader.go`: No `p` action in `processSecondKey()`
   - `internal/tui/app.go`: No `showCommandDialog()` method, no Ctrl+P handling

### Root Cause Analysis

```
Expected Flow (OpenCode):
Ctrl+P → Open command palette → Type to search → Select action

Current MagiCode Flow:
Ctrl+P → Nothing happens (no handler)
```

The feature was documented/planned but never implemented:
- `DialogCommand` type was defined
- UI hints were added
- But actual implementation missing

### Files Affected
- `internal/tui/dialog/help.go` (false claim)
- `internal/tui/layout/prompt.go` (false claim)
- `internal/tui/layout/footer.go` (false claim)
- `internal/tui/keybindings.go` (missing binding)
- `internal/tui/app.go` (missing handler)

### Fix Options

#### Option A: Remove False Claims (Quick Fix)
Remove Ctrl+P from help, prompt hints, and footer. Update leader help to show actual shortcuts.

**Pros:** Quick, prevents user confusion  
**Cons:** No command palette feature

#### Option B: Implement Command Palette (Proper Fix)
Create `dialog/command.go` with searchable command list, add Ctrl+P keybinding.

**Pros:** Matches OpenCode UX, adds useful feature  
**Cons:** More work, need to implement search/filter

#### Option C: Add Ctrl+P to Leader System (Alternative)
Make `Ctrl+X + p` open a simplified command list (use existing dialog pattern).

**Pros:** Fits current architecture  
**Cons:** Two-key sequence, doesn't match OpenCode directly

### Recommended Fix: Option B (Proper Implementation)

Create command palette matching OpenCode:
1. Create `internal/tui/dialog/command.go` with search
2. Add Ctrl+P to keybindings
3. Add handler in `app.go`
4. Populate with available actions from leader system

---

## Bug #2: Cannot Type After Phase 4 Implementation

### Severity: CRITICAL (App Broken)

### Description
After implementing the Leader Key System (Phase 4), users cannot type anything in the prompt input.

### User Report
"I can't type anything after implementing the Phase 4 when I tried to running the app"

### Root Cause Analysis

#### Key Handling Flow (Current Code)

```
Update() receives tea.KeyMsg
    ↓
Line 205: return a.handleKey(msg)  ← EARLY RETURN, skips input.Update()
    ↓
handleKey() at line 672
    ↓
handleChatKey() at line 796
    ↓
Line 801: leaderHandler.HandleKey(msg)
    ↓
If leader active/handled → return a, cmd  ← NO INPUT UPDATE
    ↓
Line 811-862: Check keybindings (Submit, Sessions, Help, etc.)
    ↓
If none match → 
    ↓
Line 865-868: a.input.Update(msg)  ← ONLY REACHED IF NO KEYBINDING MATCHED
```

#### The Problem

When leader key is active (`Ctrl+X` pressed), `handleChatKey()` returns at line 808:
```go
if handled {
    // If we got a leader action, handle it
    if leaderMsg != nil {
        return a.handleLeaderAction(leaderMsg)
    }
    // Otherwise, just return (e.g., waiting for second key or timeout)
    return a, cmd  // ← INPUT NOT UPDATED HERE!
}
```

**This prevents input.Update() from being called while leader is active.**

But there's another issue - even when leader is NOT active, regular keys may be blocked by the keybinding switch statement if they accidentally match.

#### Additional Issue: Input Update After handleKey()

In `Update()` function (line 315-319):
```go
// Update input if in input mode
if a.mode == ModeInput {
    var cmd tea.Cmd
    a.input, cmd = a.input.Update(msg)
    cmds = append(cmds, cmd)
}
```

This code tries to update input AFTER `handleKey()` returns, BUT:
- Line 206: `return a.handleKey(msg)` exits early for KeyMsg
- So line 315-319 is NEVER reached for KeyMsg!

### The Bug Summary

| Issue | Location | Effect |
|-------|----------|--------|
| Early return in Update() | Line 206 | Input never updated for KeyMsg |
| Leader handler consumes keys | Line 808 | No input when leader active |
| Input update unreachable | Line 315-319 | Dead code for KeyMsg |

### Fix Required

The input must be updated EVEN when leader is active (for typing). The leader should only consume specific key sequences, not all keys.

#### Fix Approach

1. **Move input update before key handling**:
```go
// Update input FIRST (before checking keybindings)
if a.mode == ModeInput && !a.state.Dialog.HasOpen() {
    a.input, cmd = a.input.Update(msg)
    cmds = append(cmds, cmd)
}

// Then handle special keys
if handled, cmd, leaderMsg := a.leaderHandler.HandleKey(msg); handled {
    // Handle leader, but input was already updated
    ...
}
```

OR

2. **Pass non-leader keys to input in handleChatKey()**:
```go
// In handleChatKey()
handled, cmd, leaderMsg := a.leaderHandler.HandleKey(msg)
if handled {
    if leaderMsg != nil {
        return a.handleLeaderAction(leaderMsg)
    }
    // Leader waiting for second key - but still allow typing!
    // Pass through to input if it's a regular character
    if a.mode == ModeInput && isRegularChar(msg) {
        a.input, cmd = a.input.Update(msg)
        return a, cmd
    }
    return a, cmd
}
```

OR

3. **Check leader BEFORE calling handleKey() in Update()**:

This requires restructuring the flow to:
- Always update input for regular characters
- Only intercept specific leader sequences (Ctrl+X)

### Files to Fix
- `internal/tui/app.go` lines 204-206, 315-319, 796-872

### Recommended Fix: Restructure Key Handling

```go
// Update() function - proposed fix
func (a *App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    var cmds []tea.Cmd

    // Handle resize
    if wsMsg, ok := msg.(tea.WindowSizeMsg); ok {
        // ... resize handling
    }

    switch msg := msg.(type) {
    case tea.KeyMsg:
        // DON'T return early - process input then handle special keys
        
        // Update input FIRST for all keys (unless dialog open)
        if a.mode == ModeInput && !a.state.Dialog.HasOpen() {
            var cmd tea.Cmd
            a.input, cmd = a.input.Update(msg)
            cmds = append(cmds, cmd)
        }
        
        // Then check for special key handling (leader, quit, etc.)
        if a.state.Dialog.HasOpen() {
            return a.handleDialogKey(msg)
        }
        
        // Check quit
        if msg.Type == tea.KeyCtrlC || msg.String() == "q" {
            return a, tea.Quit
        }
        
        // Check leader - but input was already updated above
        handled, cmd, leaderMsg := a.leaderHandler.HandleKey(msg)
        if handled && leaderMsg != nil {
            return a.handleLeaderAction(leaderMsg)
        }
        
        // Check other keybindings...
        return a.handleChatKeyBindings(msg)
        
    // ... other cases
    }
    
    return a, tea.Batch(cmds...)
}
```

---

## Fix Implementation Plan

### Phase 1: Fix Critical Bug (Cannot Type) - IMMEDIATE

**Files to modify:**
- `internal/tui/app.go`

**Changes:**
1. Remove early return at line 206
2. Move input.Update() BEFORE key handling
3. Ensure input always gets updated for regular typing
4. Only intercept specific leader/quit sequences

**Testing:**
- Run app, type characters - should appear in input
- Press Ctrl+X, then type - should still work (leader waiting)
- Press Ctrl+X + l - should open session list (not type 'l')

### Phase 2: Fix Ctrl+P False Claims - SAME SESSION

**Files to modify:**
- `internal/tui/dialog/help.go` - Remove Ctrl+P claim OR add handler
- `internal/tui/layout/prompt.go` - Remove Ctrl+P claim OR add handler
- `internal/tui/layout/footer.go` - Remove Ctrl+P claim OR add handler

**Changes:**
Either remove the false claims OR implement the feature.

### Phase 3: Implement Command Palette - FUTURE

**Files to create:**
- `internal/tui/dialog/command.go`

**Files to modify:**
- `internal/tui/keybindings.go` - Add Palette binding
- `internal/tui/app.go` - Add showCommandDialog()

---

## Test Cases for Fixes

### Bug #2 Fix Verification

| Test | Expected Result |
|------|-----------------|
| Type "hello" | "hello" appears in input |
| Press Ctrl+X | Leader activates, status shows "Ctrl+X" |
| After Ctrl+X, type "test" | "test" should appear in input |
| Ctrl+X + l | Session dialog opens (not type 'l') |
| Ctrl+X timeout (2s) | Leader resets, typing works |
| Escape during leader | Leader cancels, typing works |

### Bug #1 Fix Verification

| Test | Expected Result |
|------|-----------------|
| Press Ctrl+P | Command palette opens OR nothing (if removed) |
| Help dialog shows Ctrl+P | Either shows working palette OR removed |
| Footer shows Ctrl+P | Either shows working palette OR removed |

---

## Related Files

- `FLOW_COMPARISON_REVIEW.md` - Architectural differences between OpenCode and MagiCode
- `AGENTS.md` - Development guide and current status
- `internal/tui/app.go` - Main TUI application
- `internal/tui/leader.go` - Leader key handler
- `internal/tui/keybindings.go` - Keybinding definitions
- `internal/tui/dialog/help.go` - Help dialog (false Ctrl+P claim)
- `internal/tui/layout/prompt.go` - Prompt component (false Ctrl+P claim)
- `internal/tui/layout/footer.go` - Footer component (false Ctrl+P claim)