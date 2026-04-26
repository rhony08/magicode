# OpenCode vs MagiCode - Keybindings & Viewport Review

This document provides a comprehensive comparison of keybinding and viewport implementations between OpenCode (TypeScript) and MagiCode (Go).

## Executive Summary

| Category | OpenCode Keybinds | MagiCode Keybinds | Match Status |
|----------|------------------|-------------------|--------------|
| Leader Key System | Ctrl+X (2s timeout) | Ctrl+X (2s timeout) | ✅ Match |
| Session Commands | 6 keybinds | 6 keybinds | ✅ Match |
| Navigation | 2 keybinds | 1 keybind | ❌ Missing sidebar_toggle via direct Ctrl+S |
| Message Scrolling | 15 keybinds | 4 keybinds | ❌ Missing 11 scroll keybinds |
| Model/Agent | 8 keybinds | 4 keybinds | ❌ Missing 4 keybinds |
| Input Editing | 35+ keybinds | ~5 keybinds | ❌ Missing 30+ input keybinds |
| System | 10+ keybinds | 6 keybinds | ❌ Missing several system keybinds |

---

## Keybinding Comparison

### Leader Key System

| Feature | OpenCode | MagiCode | Status |
|---------|----------|----------|--------|
| Leader Key | `ctrl+x` | `ctrl+x` | ✅ |
| Timeout Duration | 2000ms | 2000ms | ✅ |
| Timeout Handling | Auto-reset, blur focus restoration | Auto-reset | ✅ |
| Escape Cancel | Yes | Yes (implicit) | ⚠️ Partial |

**Implementation Notes:**
- OpenCode blurs current focusable when leader activates, restores on timeout
- MagiCode doesn't track focus state but functional equivalent for TUI
- Both handle `<leader>` prefix substitution correctly

### Session Commands (Leader Actions)

| Action | OpenCode Key | MagiCode Key | Status |
|--------|-------------|--------------|--------|
| Session List | `<leader>l` | `Ctrl+X l` | ✅ |
| Session New | `<leader>n` | `Ctrl+X n` | ✅ |
| Session Export | `<leader>x` | `Ctrl+X x` | ✅ |
| Session Compact | `<leader>c` | `Ctrl+X c` | ✅ |
| Session Timeline | `<leader>g` | `Ctrl+X g` | ✅ |
| Session Fork | `none` (disabled) | Not implemented | ⚠️ |
| Session Rename | `ctrl+r` | Not implemented | ❌ |
| Session Delete | `ctrl+d` | Not implemented | ❌ |
| Session Share | `none` | Not implemented | ⚠️ |
| Session Unshare | `none` | Not implemented | ⚠️ |
| Session Interrupt | `escape` | `escape` | ✅ |

### Navigation Commands

| Action | OpenCode Key | MagiCode Key | Status |
|--------|-------------|--------------|--------|
| Sidebar Toggle | `<leader>b` | `Ctrl+X b` | ✅ |
| Scrollbar Toggle | `none` (disabled) | Not implemented | ⚠️ |
| Username Toggle | `none` | Not implemented | ⚠️ |
| Editor Open | `<leader>e` | Not implemented | ❌ |
| Go to first child | `<leader>down` | Not implemented | ❌ |
| Go to next child | `right` | Not implemented | ❌ |
| Go to prev child | `left` | Not implemented | ❌ |
| Go to parent | `up` | Not implemented | ❌ |

### Model/Agent Commands

| Action | OpenCode Key | MagiCode Key | Status |
|--------|-------------|--------------|--------|
| Model List | `<leader>m` | `Ctrl+X m` | ✅ |
| Agent List | `<leader>a` | `Ctrl+X a` | ✅ |
| Model Cycle Recent | `f2` | Not implemented | ❌ |
| Model Cycle Recent Reverse | `shift+f2` | Not implemented | ❌ |
| Model Cycle Favorite | `none` | Not implemented | ⚠️ |
| Model Cycle Favorite Reverse | `none` | Not implemented | ⚠️ |
| Agent Cycle | `tab` | Not implemented | ❌ |
| Agent Cycle Reverse | `shift+tab` | Not implemented | ❌ |
| Variant Cycle | `ctrl+t` | Not implemented | ❌ |
| Variant List | `none` | Not implemented | ⚠️ |
| Provider List | `ctrl+a` | Not implemented | ❌ |
| Favorite Toggle | `ctrl+f` | Not implemented | ❌ |
| MCPs Toggle | No keybind (slash command) | Not implemented | ❌ |

### Message Scrolling Commands

| Action | OpenCode Key | MagiCode Key | Status |
|--------|-------------|--------------|--------|
| Page Up | `pageup,ctrl+alt+b` | `pgup` | ⚠️ Partial (missing ctrl+alt+b) |
| Page Down | `pagedown,ctrl+alt+f` | `pgdown` | ⚠️ Partial (missing ctrl+alt+f) |
| Line Up | `ctrl+alt+y` | `up` | ❌ Different (up is scroll by line) |
| Line Down | `ctrl+alt+e` | `down` | ❌ Different |
| Half Page Up | `ctrl+alt+u` | `pgup` (half view) | ⚠️ Different behavior |
| Half Page Down | `ctrl+alt+d` | `pgdown` (half view) | ⚠️ Different behavior |
| First Message | `ctrl+g,home` | Not implemented | ❌ |
| Last Message | `ctrl+alt+g,end` | Not implemented | ❌ |
| Next Message | `none` | Not implemented | ⚠️ |
| Previous Message | `none` | Not implemented | ⚠️ |
| Last User Message | `none` | Not implemented | ⚠️ |
| Copy Message | `<leader>y` | `Ctrl+X y` | ✅ |
| Undo Message | `<leader>u` | `Ctrl+X u` | ✅ (placeholder) |
| Redo Message | `<leader>r` | `Ctrl+X r` | ✅ (placeholder) |
| Toggle Conceal | `<leader>h` | Not implemented | ❌ |

### Input Editing Commands

OpenCode has extensive input editing keybinds. MagiCode uses Bubble Tea's textarea which has different defaults.

| Action | OpenCode Key | MagiCode Key | Status |
|--------|-------------|--------------|--------|
| Submit Input | `return` | `enter` | ✅ |
| Newline | `shift+return,ctrl+return,alt+return,ctrl+j` | Not explicit | ⚠️ |
| Clear Input | `ctrl+c` | Not implemented | ❌ |
| Paste | `ctrl+v` | Not implemented | ❌ |
| Move Left | `left,ctrl+b` | `left` | ⚠️ Partial |
| Move Right | `right,ctrl+f` | `right` | ⚠️ Partial |
| Move Up | `up` | `up` (scroll viewport) | ❌ Conflict |
| Move Down | `down` | `down` (scroll viewport) | ❌ Conflict |
| Select Left | `shift+left` | Not implemented | ❌ |
| Select Right | `shift+right` | Not implemented | ❌ |
| Select Up | `shift+up` | Not implemented | ❌ |
| Select Down | `shift+down` | Not implemented | ❌ |
| Line Home | `ctrl+a` | `home` | ⚠️ Different |
| Line End | `ctrl+e` | `end` | ⚠️ Different |
| Buffer Home | `home` | `home` | ✅ |
| Buffer End | `end` | `end` | ✅ |
| Delete Line | `ctrl+shift+d` | Not implemented | ❌ |
| Delete to End | `ctrl+k` | Not implemented | ❌ |
| Delete to Start | `ctrl+u` | Not implemented | ❌ |
| Backspace | `backspace,shift+backspace` | `backspace` | ⚠️ Partial |
| Delete | `ctrl+d,delete,shift+delete` | `delete` | ⚠️ Partial |
| Undo Input | `ctrl+-,super+z` (Win: `ctrl+z`) | Not implemented | ❌ |
| Redo Input | `ctrl+.,super+shift+z` | Not implemented | ❌ |
| Word Forward | `alt+f,alt+right,ctrl+right` | Not implemented | ❌ |
| Word Backward | `alt+b,alt+left,ctrl+left` | Not implemented | ❌ |
| Select Word Forward | `alt+shift+f,alt+shift+right` | Not implemented | ❌ |
| Select Word Backward | `alt+shift+b,alt+shift+left` | Not implemented | ❌ |
| Delete Word Forward | `alt+d,alt+delete,ctrl+delete` | Not implemented | ❌ |
| Delete Word Backward | `ctrl+w,ctrl+backspace,alt+backspace` | Not implemented | ❌ |
| History Previous | `up` (in input context) | `ctrl+up` | ❌ Different |
| History Next | `down` (in input context) | `ctrl+down` | ❌ Different |

### System Commands

| Action | OpenCode Key | MagiCode Key | Status |
|--------|-------------|--------------|--------|
| Exit App | `ctrl+c,ctrl+d,<leader>q` | `ctrl+c,q` | ✅ |
| Theme List | `<leader>t` | `Ctrl+X t` | ✅ |
| Status View | `<leader>s` | `Ctrl+X s` | ✅ |
| Help Dialog | `<leader>h` | `Ctrl+X h` | ✅ |
| Command List | `ctrl+p` | `ctrl+p` | ✅ |
| Terminal Suspend | `ctrl+z` (disabled on Win32) | Not implemented | ❌ |
| Terminal Title Toggle | `none` | Not implemented | ⚠️ |
| Toggle Animations | No keybind (KV setting) | Not implemented | ⚠️ |
| Toggle Diff Wrap | No keybind (KV setting) | Not implemented | ⚠️ |
| Toggle Thinking | `none` | Not implemented | ⚠️ |
| Toggle Tool Details | `none` | Not implemented | ⚠️ |
| Toggle Tips | `<leader>h` (home only) | Not implemented | ⚠️ |
| Plugin Manager | `none` | Not implemented | ⚠️ |
| Open Docs | No keybind (slash: /docs) | Not implemented | ❌ |

---

## Missing Keybindings in MagiCode

### Critical Missing (Core Functionality)

1. **Scroll Navigation**
   - `home` / `ctrl+g` - Jump to first message
   - `end` / `ctrl+alt+g` - Jump to last message
   - `ctrl+alt+y` / `ctrl+alt+e` - Line-by-line scrolling with separate viewport navigation
   - `ctrl+alt+u` / `ctrl+alt+d` - Half-page scrolling

2. **Input Context Separation**
   - OpenCode: `up/down` scroll viewport, `ctrl+up/ctrl+down` navigate history
   - MagiCode: `up/down` scroll viewport, `ctrl+up/ctrl+down` navigate history (matches!)
   - BUT: MagiCode lacks input movement keybinds (word navigation, selection, etc.)

3. **Session Tree Navigation**
   - `<leader>down` - Go to first child session
   - `right` - Go to next child session
   - `left` - Go to previous child session
   - `up` - Go to parent session

4. **Model/Agent Cycling**
   - `f2` / `shift+f2` - Cycle recent models
   - `tab` / `shift+tab` - Cycle agents (not implemented as agent cycling)

5. **Input Editing**
   - `ctrl+k` - Delete to end of line
   - `ctrl+u` - Delete to start of line
   - `ctrl+w` / `ctrl+backspace` - Delete word backward
   - `alt+d` - Delete word forward
   - `ctrl+shift+d` - Delete entire line
   - Undo/redo for input (ctrl+z / ctrl+y on Windows)

### Medium Priority Missing

1. **Session Management**
   - `ctrl+r` - Rename session
   - `ctrl+d` - Delete session

2. **Model Management**
   - `ctrl+a` - Open provider list
   - `ctrl+f` - Toggle favorite model
   - `ctrl+t` - Cycle model variants

3. **Display Controls**
   - `<leader>h` - Toggle code block concealment
   - Scrollbar toggle (disabled by default but has keybind)

### Low Priority Missing (Optional/Disabled by Default)

1. **System Features**
   - `ctrl+z` - Terminal suspend (disabled on Windows)
   - Plugin manager dialog
   - Tips toggle on home screen

---

## Viewport/Scroll Differences

### Scroll Architecture

| Aspect | OpenCode | MagiCode | Status |
|--------|----------|----------|--------|
| Component | `<scrollbox>` (opentui) | `viewport.Model` (bubbletea) | Different libs |
| Auto-scroll | Yes, on new messages | Yes, via `GotoBottom()` | ✅ |
| Sticky scroll | `stickyScroll={true}` | Not implemented | ❌ |
| Sticky position | `stickyStart="bottom"` | Not implemented | ❌ |
| Scroll acceleration | Optional (MacOSScrollAccel) | Not implemented | ❌ |
| Scroll speed config | `scroll_speed` option | Hardcoded | ❌ |
| Scrollbar options | Configurable visibility | Basic viewport scrollbar | ⚠️ |

### Auto-Scroll Behavior

**OpenCode Implementation:**
```tsx
// From session/index.tsx
// When message stream completes:
if (route.sessionID === sessionID && scroll) scroll.scrollBy(100_000)

// Sticky scroll keeps viewport at bottom when new messages arrive
<scrollbox stickyScroll={true} stickyStart="bottom">
```

**MagiCode Implementation:**
```go
// From app.go
func (a *App) scrollToBottom(force bool) {
    if !force && a.state.IsUserScrolled() {
        return  // Don't auto-scroll if user scrolled up
    }
    a.messageViewport.GotoBottom()
}

// Called when:
// - New message added (addMessage)
// - StreamMsg received (appendToLastMessage)
// - Session switched (handleSessionMsg "switch")
```

**Key Differences:**
1. OpenCode uses sticky scroll - viewport auto-sticks to bottom as content grows
2. MagiCode manually calls `GotoBottom()` on message events
3. Both track user scroll state to prevent unwanted auto-scroll

### User Scroll Detection

**OpenCode:**
- Uses scrollbox's internal position tracking
- No explicit "userScrolled" state visible in code

**MagiCode:**
```go
func (a *App) handleViewportScroll() {
    contentHeight := len(strings.Split(content, "\n"))
    viewportHeight := a.messageViewport.Height
    currentOffset := a.messageViewport.YOffset
    
    maxOffset := contentHeight - viewportHeight
    distanceFromBottom := maxOffset - currentOffset
    
    // Mark as user scrolled if > 2 lines from bottom
    if distanceFromBottom > 2 {
        a.state.SetUserScrolled(true)
    } else {
        a.state.SetUserScrolled(false)
    }
}
```

**Comparison:**
- MagiCode has explicit user scroll state tracking
- Both prevent auto-scroll when user is reading history
- Threshold differs (OpenCode may use different threshold)

### Scroll Pagination

Both implementations support cursor-based pagination:

| Feature | OpenCode | MagiCode | Status |
|---------|----------|----------|--------|
| Initial Load | 80 messages | 80 messages | ✅ |
| History Load | 200 messages | 200 messages | ✅ |
| Trigger | Scroll to top | `atTopOfMessages()` | ✅ |
| Cursor | Timestamp-based | Timestamp-based | ✅ |

---

## Implementation Gaps

### Gap 1: Sticky Scroll

**OpenCode Feature:** `stickyScroll={true}` keeps viewport at bottom when new content arrives without explicit scroll calls.

**MagiCode:** Uses explicit `GotoBottom()` calls in message handlers.

**Fix Required:** Consider implementing sticky scroll behavior in viewport update loop:
```go
// In Update() after StreamMsg handling:
if !a.state.IsUserScrolled() {
    a.messageViewport.GotoBottom()
}
```

### Gap 2: Scroll Acceleration

**OpenCode Feature:** Optional scroll acceleration via `MacOSScrollAccel` or custom speed.

**MagiCode:** Hardcoded scroll speed (uses viewport's default behavior).

**Fix Required:** Add scroll speed configuration option:
```go
type Config struct {
    ScrollSpeed float64 // Pixels per tick
    ScrollAcceleration bool
}
```

### Gap 3: Comprehensive Input Keybinds

**OpenCode:** 35+ input editing keybinds including:
- Word navigation (ctrl+left/right, alt+b/f)
- Selection (shift+arrows, ctrl+shift+a/e)
- Line/buffer navigation (ctrl+a/e, home/end)
- Undo/redo for input
- Multiple delete variants

**MagiCode:** Basic textarea with limited keybinds.

**Fix Required:** Either:
1. Implement custom textarea keybind handler
2. Extend bubbletea textarea with additional keybinds
3. Document that textarea uses bubbletea defaults

### Gap 4: Session Tree Navigation

**OpenCode:** Parent/child session navigation with arrow keys.

**MagiCode:** No session hierarchy support.

**Fix Required:** If session hierarchy is planned:
1. Add `<leader>down` for first child
2. Add `right/left` for sibling navigation
3. Add `up` for parent navigation

### Gap 5: Model/Agent Quick Cycling

**OpenCode:** `f2/shift+f2` for recent models, `tab/shift+tab` for agents.

**MagiCode:** Uses dialogs for model/agent selection only.

**Fix Required:** Add quick cycling:
```go
case kb.ModelCycle.Match(msg):  // f2
    a.SafeCycleModel(1)
    
case kb.AgentCycle.Match(msg):  // tab (when not in autocomplete)
    a.SafeCycleAgent(1)
```

---

## Platform-Specific Differences

### Windows Terminal

**OpenCode:**
- `terminal_suspend = "none"` on Windows (no POSIX suspend)
- `input_undo` includes `ctrl+z` on Windows
- Ctrl+C guard for Windows processed input

**MagiCode:**
- No platform-specific handling visible
- Should add Windows-specific keybind adjustments

### macOS

**OpenCode:**
- Uses `super` key (cmd) for some keybinds
- `input_undo: "ctrl+-,super+z"` on macOS

**MagiCode:**
- No `super` key handling (bubbletea limitation)

---

## Recommendations

### Immediate Fixes (P0)

1. **Add Home/End message navigation**
   ```go
   case kb.First.Match(msg):  // home
       a.messageViewport.GotoTop()
   case kb.Last.Match(msg):    // end
       a.messageViewport.GotoBottom()
   ```

2. **Fix input/viewport conflict**
   - When input focused: `up/down` should move cursor
   - When not focused: `up/down` should scroll viewport
   - Current: `up/down` always scrolls viewport

3. **Add sticky scroll behavior**
   - Keep viewport at bottom when processing unless user scrolled

### Short-term Fixes (P1)

1. Implement word navigation in input (ctrl+left/right)
2. Add half-page scrolling (ctrl+alt+u/d)
3. Add model cycling (f2/shift+f2)
4. Add agent cycling (tab/shift+tab)

### Medium-term Fixes (P2)

1. Full input editing keybinds
2. Session rename/delete keybinds
3. Scroll acceleration configuration
4. Scrollbar toggle functionality

### Documentation Needed

1. Document that MagiCode uses Bubble Tea defaults for textarea
2. Document keybind differences for users migrating from OpenCode
3. Document missing features with "not yet implemented" status

---

## Testing Checklist

- [ ] Leader key timeout works (2 seconds)
- [ ] Escape cancels leader sequence
- [ ] All leader actions implemented work correctly
- [ ] Auto-scroll respects user scroll state
- [ ] History pagination triggers at top
- [ ] Input doesn't conflict with viewport scrolling
- [ ] Model/agent dialogs work with leader keys
- [ ] Theme switching works and persists
- [ ] Help dialog shows correct keybinds
- [ ] Command palette accessible with Ctrl+P

---

## Source Files Reference

### OpenCode
- `/packages/opencode/src/config/keybinds.ts` - All keybind definitions
- `/packages/opencode/src/cli/cmd/tui/context/keybind.tsx` - Leader key handler
- `/packages/opencode/src/cli/cmd/tui/routes/session/index.tsx` - Scroll handling
- `/packages/opencode/src/cli/cmd/tui/util/scroll.ts` - Scroll acceleration

### MagiCode
- `/internal/tui/keybindings.go` - Basic keybind definitions
- `/internal/tui/leader.go` - Leader key system
- `/internal/tui/app.go` - Viewport handling, scroll behavior
- `/internal/tui/layout/prompt.go` - Input handling

---

## Appendix: Full Keybind List from OpenCode

From `config/keybinds.ts`:

| Keybind Name | Default Value | Description |
|--------------|---------------|-------------|
| `leader` | `ctrl+x` | Leader key |
| `app_exit` | `ctrl+c,ctrl+d,<leader>q` | Exit application |
| `editor_open` | `<leader>e` | Open external editor |
| `theme_list` | `<leader>t` | List themes |
| `sidebar_toggle` | `<leader>b` | Toggle sidebar |
| `scrollbar_toggle` | `none` | Toggle scrollbar |
| `username_toggle` | `none` | Toggle username |
| `status_view` | `<leader>s` | View status |
| `session_export` | `<leader>x` | Export session |
| `session_new` | `<leader>n` | New session |
| `session_list` | `<leader>l` | List sessions |
| `session_timeline` | `<leader>g` | Session timeline |
| `session_fork` | `none` | Fork session |
| `session_rename` | `ctrl+r` | Rename session |
| `session_delete` | `ctrl+d` | Delete session |
| `session_share` | `none` | Share session |
| `session_unshare` | `none` | Unshare session |
| `session_interrupt` | `escape` | Interrupt session |
| `session_compact` | `<leader>c` | Compact session |
| `messages_page_up` | `pageup,ctrl+alt+b` | Page up |
| `messages_page_down` | `pagedown,ctrl+alt+f` | Page down |
| `messages_line_up` | `ctrl+alt+y` | Line up |
| `messages_line_down` | `ctrl+alt+e` | Line down |
| `messages_half_page_up` | `ctrl+alt+u` | Half page up |
| `messages_half_page_down` | `ctrl+alt+d` | Half page down |
| `messages_first` | `ctrl+g,home` | First message |
| `messages_last` | `ctrl+alt+g,end` | Last message |
| `messages_next` | `none` | Next message |
| `messages_previous` | `none` | Previous message |
| `messages_last_user` | `none` | Last user message |
| `messages_copy` | `<leader>y` | Copy message |
| `messages_undo` | `<leader>u` | Undo |
| `messages_redo` | `<leader>r` | Redo |
| `messages_toggle_conceal` | `<leader>h` | Toggle conceal |
| `tool_details` | `none` | Toggle tool details |
| `model_list` | `<leader>m` | List models |
| `model_cycle_recent` | `f2` | Cycle recent |
| `model_cycle_recent_reverse` | `shift+f2` | Cycle recent reverse |
| `model_cycle_favorite` | `none` | Cycle favorite |
| `model_cycle_favorite_reverse` | `none` | Cycle favorite reverse |
| `command_list` | `ctrl+p` | Command palette |
| `agent_list` | `<leader>a` | List agents |
| `agent_cycle` | `tab` | Cycle agent |
| `agent_cycle_reverse` | `shift+tab` | Cycle agent reverse |
| `variant_cycle` | `ctrl+t` | Cycle variant |
| `variant_list` | `none` | List variants |
| `input_clear` | `ctrl+c` | Clear input |
| `input_paste` | `ctrl+v` | Paste |
| `input_submit` | `return` | Submit |
| `input_newline` | `shift+return,ctrl+return,alt+return,ctrl+j` | Newline |
| `input_move_left` | `left,ctrl+b` | Move left |
| `input_move_right` | `right,ctrl+f` | Move right |
| `input_move_up` | `up` | Move up |
| `input_move_down` | `down` | Move down |
| `input_select_left` | `shift+left` | Select left |
| `input_select_right` | `shift+right` | Select right |
| `input_select_up` | `shift+up` | Select up |
| `input_select_down` | `shift+down` | Select down |
| `input_line_home` | `ctrl+a` | Line home |
| `input_line_end` | `ctrl+e` | Line end |
| `input_select_line_home` | `ctrl+shift+a` | Select to line home |
| `input_select_line_end` | `ctrl+shift+e` | Select to line end |
| `input_visual_line_home` | `alt+a` | Visual line home |
| `input_visual_line_end` | `alt+e` | Visual line end |
| `input_select_visual_line_home` | `alt+shift+a` | Select visual home |
| `input_select_visual_line_end` | `alt+shift+e` | Select visual end |
| `input_buffer_home` | `home` | Buffer home |
| `input_buffer_end` | `end` | Buffer end |
| `input_select_buffer_home` | `shift+home` | Select buffer home |
| `input_select_buffer_end` | `shift+end` | Select buffer end |
| `input_delete_line` | `ctrl+shift+d` | Delete line |
| `input_delete_to_line_end` | `ctrl+k` | Delete to end |
| `input_delete_to_line_start` | `ctrl+u` | Delete to start |
| `input_backspace` | `backspace,shift+backspace` | Backspace |
| `input_delete` | `ctrl+d,delete,shift+delete` | Delete |
| `input_undo` | `ctrl+-,super+z` (Win: `ctrl+z`) | Undo |
| `input_redo` | `ctrl+.,super+shift+z` | Redo |
| `input_word_forward` | `alt+f,alt+right,ctrl+right` | Word forward |
| `input_word_backward` | `alt+b,alt+left,ctrl+left` | Word backward |
| `input_select_word_forward` | `alt+shift+f,alt+shift+right` | Select word forward |
| `input_select_word_backward` | `alt+shift+b,alt+shift+left` | Select word backward |
| `input_delete_word_forward` | `alt+d,alt+delete,ctrl+delete` | Delete word forward |
| `input_delete_word_backward` | `ctrl+w,ctrl+backspace,alt+backspace` | Delete word backward |
| `history_previous` | `up` | History previous |
| `history_next` | `down` | History next |
| `session_child_first` | `<leader>down` | First child |
| `session_child_cycle` | `right` | Next child |
| `session_child_cycle_reverse` | `left` | Previous child |
| `session_parent` | `up` | Parent session |
| `terminal_suspend` | `ctrl+z` (Win: `none`) | Suspend terminal |
| `terminal_title_toggle` | `none` | Toggle title |
| `tips_toggle` | `<leader>h` | Toggle tips |
| `plugin_manager` | `none` | Plugin manager |
| `display_thinking` | `none` | Toggle thinking |