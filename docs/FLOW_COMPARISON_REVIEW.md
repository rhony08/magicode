# MagiCode vs OpenCode Flow Comparison Review

**Date:** 2026-04-25  
**Status:** Completed  
**Purpose:** Ensure MagiCode matches OpenCode flow exactly (except improvement plan)

---

## Summary

After comparing the OpenCode TypeScript implementation with the MagiCode Go implementation, I've identified several significant differences in flow and architecture.

---

## Phase-by-Phase Review

### Phase 1.1: Message Loading

**OpenCode Flow (sync.tsx lines 18, 294-368):**
```typescript
const SKIP_PARTS = new Set(["patch", "step-start", "step-finish"])

// Fetch with retry, cursor-based pagination
const fetchMessages = async (input) => {
  const messages = await retry(() =>
    input.client.session.messages({ sessionID, limit, before })
  )
  // Filter parts
  const filtered = p.part.filter((x) => !SKIP_PARTS.has(x.type))
}
```

**MagiCode Flow:**
- ✅ `SKIP_PARTS` pattern documented in AGENTS.md
- ❓ Need to verify actual implementation in Go message loading code

**📝 Note:** The `SKIP_PARTS` pattern is documented but implementation location needs verification.

---

### Phase 1.2: State Management Architecture

**OpenCode Flow (5 layered stores):**
| Layer | OpenCode File | Purpose |
|-------|---------------|---------|
| GlobalSyncStore | `global-sync.tsx` | Global data (projects, providers, session_todo) |
| SyncStore | `sync.tsx` | Per-directory (sessions, messages) |
| LayoutStore | `layout.tsx` | UI dimensions, sidebar state |
| LocalStore | `local.tsx` | Agent/model selection per session |
| KVStore | Persisted via `persisted()` | Persistent preferences |

**MagiCode Flow (types/types.go lines 241-509):**
| Layer | Go Location | Match Status |
|-------|-------------|--------------|
| KVStore | lines 242-269 | ✅ Match |
| SyncStore | lines 271-301 | ✅ Match |
| LocalStore | lines 346-381 | ✅ Match |
| LayoutStore | lines 383-441 | ✅ Match |
| DialogStore | lines 443-456 | ✅ Match (New in MagiCode) |

**🚨 Difference #1: GlobalSyncStore vs SyncStore**

OpenCode separates:
- `GlobalSyncStore` - Global project/provider data (line 31-43 in global-sync.tsx)
- `SyncStore` - Per-directory session/message data (sync.tsx)

MagiCode combines both into single `SyncStore`:
```go
type SyncStore struct {
  Sessions     []Session
  Messages     []Message
  Providers    []Provider  // <-- This should be in GlobalSyncStore
  Agents       []Agent     // <-- This should be in GlobalSyncStore
  ProjectID    string
  ...
}
```

**📝 Note:** MagiCode lacks the `GlobalSyncStore` separation. OpenCode has:
- `session_todo` at global level (global-sync.tsx line 36-38)
- Separate `child()` method for per-directory stores

---

### Phase 1.3: Responsive Layout

**OpenCode Flow (layout.tsx lines 17-19, 601-626):**
```typescript
const DEFAULT_SIDEBAR_WIDTH = 344
const ResponsiveThreshold = 120 // cols

sidebar: {
  opened: false,
  width: DEFAULT_SIDEBAR_WIDTH,
  workspaces: {} as Record<string, boolean>,
  workspacesDefault: false,
}
```

**MagiCode Flow (types/types.go lines 511-531):**
```go
const ResponsiveThreshold = 120  // ✅ Match

func (l *LayoutStore) IsResponsive() bool {
  return l.Width < ResponsiveThreshold  // ✅ Match
}
```

**✅ Match:** Sidebar width 344px, threshold 120 cols - all correct.

**✅ Match:** Sidebar opened default is `false` in both implementations.

---

### Phase 2: Layout Components

**OpenCode Sidebar (layout.tsx):**
- Width: 344px (line 17)
- Workspace toggle per directory (line 616-625)
- Projects list with enrich/project metadata (lines 387-425)

**MagiCode Sidebar (layout/sidebar.go):**
- Width: 43 cols (≈344px) ✅
- ❌ **Missing:** `workspaces` toggle per directory
- ❌ **Missing:** Project `enrich()` with metadata/color handling

**🚨 Difference #2:** OpenCode sidebar has:
1. Per-directory `workspaces[directory]` toggle (layout.tsx line 616)
2. Project color picker/avatar colors (lines 378-543)
3. `SessionTabs` state per session (lines 875-947)

MagiCode sidebar is simplified - missing:
- Per-directory workspace toggle
- Project avatar colors (pink, mint, orange, purple, cyan, lime)
- Session tabs state (All tabs, Active tab per session)

---

### Phase 3: Dialog System

**OpenCode Dialog (ui/context/dialog.tsx):**
- Single active dialog (line 31: `[active, setActive]`)
- Auto-close on new dialog open (line 76-81)
- Escape key handling (line 65-72)
- Close animation (line 54-59: 100ms delay)

**MagiCode Dialog (dialog/dialog.go):**
- Stack-based (DialogStore.Stack)
- Multiple dialogs can be stacked
- CloseMsg, SelectMsg patterns

**🚨 Difference #3: Single vs Stack Dialogs**

OpenCode: Only **ONE** active dialog at a time
```typescript
// Immediately dispose any existing dialog when showing a new one (line 76-81)
const current = active()
if (current) {
  current.dispose()
  setActive(undefined)
}
```

MagiCode: **STACK** of dialogs (types/types.go line 443-446)
```go
type DialogStore struct {
  Stack []DialogState `json:"stack"` // Active dialogs (top is last)
}
```

**📝 Note:** This is a **significant flow difference**. OpenCode explicitly replaces existing dialogs when opening new ones, while MagiCode allows stacking.

---

### Phase 4: Keybinding System

**OpenCode Keybinds (command.tsx):**

OpenCode uses a **command palette** approach:
- `mod+shift+p` opens command palette (line 14)
- Single keybinds per action (line 108-150: parseKeybind)
- Per-action registration (line 385-400: register())
- Customizable keybinds via settings (line 249-267)

Key format: `"mod+b", "ctrl+l", "mod+shift+s"` etc.

**MagiCode Keybinds (leader.go):**

MagiCode uses a **leader key system**:
- `Ctrl+X` as prefix (line 14: LeaderTimeout = 2000ms)
- Two-key sequences: `Ctrl+X + l` for session list
- Hardcoded actions (lines 72-149)

**🚨 Difference #4: Keybind Architecture**

| Aspect | OpenCode | MagiCode |
|--------|----------|----------|
| System | Command palette | Leader key prefix |
| Primary shortcut | `mod+shift+p` (palette) | `Ctrl+X` (leader) |
| Session list | Direct keybind | `Ctrl+X + l` (2-key) |
| Sidebar toggle | `mod+b` | `Ctrl+X + b` (2-key) |
| Customizable | Yes (settings.keybinds) | No (hardcoded) |
| Multiple shortcuts | Yes (comma-separated) | No |

**📝 Note:** OpenCode's command.tsx has:
```typescript
keybind: "mod+k,mod+p"  // Two ways to trigger same action
```

MagiCode's leader.go only supports single sequence per action.

---

### Phase 4.2: Keybind Registry

**OpenCode (command.tsx lines 76-97, 255-338):**
- `CommandOption` with `keybind?: KeybindConfig`
- `CommandRegistration` with dynamic options
- `keymap()` builds signature -> option map
- `matchKeybind()` checks keyboard event

**MagiCode (keybindings.go):**
- Static `Keybindings` struct (lines 44-61)
- `DefaultKeybindings()` returns hardcoded values
- No registration system

**🚨 Difference #5: No Dynamic Registration**

OpenCode allows dynamic command registration:
```typescript
function register(key: string | (() => CommandOption[]), cb?: () => CommandOption[])
```

MagiCode has static keybindings - no registration API.

---

### Theme System Comparison

**OpenCode (ui/src/theme):**
- **35+ themes** (default-themes.ts lines 78-116)
- JSON file loading from themes/*.json
- `DesktopTheme` with light/dark variants (types.ts lines 43-49)
- Seed colors → palette generation (resolve.ts)
- CSS variables output (loader.ts: themeToCss)

**MagiCode (theme.go):**
- **8 themes** (lines 83-380)
- Inline hardcoded colors
- Flat `Theme` struct (no light/dark variants)
- Direct lipgloss.Color usage

**🚨 Difference #6: Theme Scale**

| Aspect | OpenCode | MagiCode |
|--------|----------|----------|
| Theme count | 35+ themes | 8 themes |
| Theme format | JSON files + seeds/palette | Inline hardcoded |
| Light/dark variants | Yes (DesktopTheme) | No (single Theme) |
| Seed color generation | Yes (resolve.ts) | No |
| CSS variable output | Yes | No (lipgloss only) |

---

### Message Pagination Comparison

**OpenCode (sync.tsx lines 184-186, 545-579):**
```typescript
const initialMessagePageSize = 80
const historyMessagePageSize = 200

history: {
  more(sessionID) { return !!meta.cursor[key] && !meta.complete[key] }
  loading(sessionID) { return meta.loading[key] }
  loadMore(sessionID, count) { ... }
}
```

**MagiCode (types/types.go lines 461-465, 802-839):**
```go
const InitialMessagePageSize = 80   // ✅ Match
const HistoryMessagePageSize = 200  // ✅ Match

func (s *AppState) HistoryMore(sessionID string) bool  // ✅ Match
func (s *AppState) HistoryLoading(sessionID string) bool  // ✅ Match
func (s *AppState) PrependMessages(messages, cursor, complete) // ✅ Match
```

**✅ Match:** Pagination constants and flow match OpenCode exactly.

---

## Summary of All Differences

| # | Category | Difference | Impact | Priority |
|---|----------|------------|--------|----------|
| 1 | State | Missing GlobalSyncStore separation | Medium - affects multi-directory | HIGH |
| 2 | Layout | Missing workspace toggle per dir | Low | LOW |
| 3 | Sidebar | Missing project colors/avatars | Low | LOW |
| 4 | Sidebar | Missing SessionTabs state | Medium - affects tab persistence | MEDIUM |
| 5 | Dialog | Stack vs Single dialog | Medium - UX difference | HIGH |
| 6 | Keybind | Leader key vs Command palette | **High** - major UX change | HIGH |
| 7 | Keybind | No customizable keybinds | Medium | MEDIUM |
| 8 | Keybind | No dynamic registration | Low | LOW |
| 9 | Theme | 8 vs 35+ themes | Medium | LOW |
| 10 | Theme | No light/dark variants | Low | LOW |
| 11 | Theme | No seed color generation | Low | LOW |

---

## Critical Flow Differences (Must Address)

### 1. GlobalSyncStore Missing
OpenCode separates global data (providers, agents, session_todo) from per-directory data. MagiCode combines them into SyncStore, which could cause issues with:
- Multi-directory projects
- Global session_todo tracking
- Provider connection state sharing

**Location:** `internal/tui/types/types.go` line 271-301

---

### 2. Dialog Stack vs Single
OpenCode explicitly disallows stacking dialogs. MagiCode's stack-based approach may:
- Cause UX confusion
- Break keyboard focus handling
- Not match user expectations from OpenCode

**Location:** `internal/tui/types/types.go` line 443-446, `internal/tui/dialog/dialog.go`

---

### 3. Leader Key vs Command Palette
This is the **most significant UX difference**:
- OpenCode: `mod+shift+p` → palette → select action
- MagiCode: `Ctrl+X` → wait → press key → action

Users expecting OpenCode's command palette will be confused by leader key system.

**Location:** `internal/tui/leader.go`, `internal/tui/keybindings.go`

**Issue:** `Ctrl+P` shortcut not working - need to investigate

---

## Recommendations

### HIGH Priority (Blocking UX)
1. **Fix keybinding system** - Ensure Ctrl+P and other shortcuts work correctly
2. **Add Command Palette** - Implement palette option alongside leader keys (or replace leader)
3. **Change Dialog to Single** - Match OpenCode's single-dialog model

### MEDIUM Priority
4. **Add GlobalSyncStore** - Separate global data from directory-specific data
5. **Add SessionTabs State** - Per-session tab tracking for file views
6. **Add Customizable Keybinds** - Allow user keybind customization

### LOW Priority
7. **Add workspace toggle per directory**
8. **Add project avatar colors**
9. **Expand Theme Support** - Load JSON themes with light/dark variants
10. **Add seed color generation**

---

## Keybinding Investigation

### Current Issue
`Ctrl+P` shortcut not working in MagiCode.

### Expected Behavior (OpenCode)
- `Ctrl+P` (or `mod+shift+p`) opens command palette
- Command palette shows searchable list of all actions

### Current MagiCode Implementation
- No `Ctrl+P` binding exists
- Leader key system: `Ctrl+X` → second key
- Static `Keybindings` struct in `keybindings.go`

### Files to Investigate
- `internal/tui/keybindings.go` - Check if Ctrl+P is defined
- `internal/tui/leader.go` - Check if Ctrl+P is blocked by leader handler
- `internal/tui/app.go` - Check `handleKey()` and `handleChatKey()` flow

---

## Files Referenced

### OpenCode Files
- `packages/app/src/context/sync.tsx`
- `packages/app/src/context/layout.tsx`
- `packages/app/src/context/local.tsx`
- `packages/app/src/context/global-sync.tsx`
- `packages/app/src/context/command.tsx`
- `packages/ui/src/context/dialog.tsx`
- `packages/ui/src/theme/index.ts`
- `packages/ui/src/theme/default-themes.ts`
- `packages/ui/src/theme/types.ts`

### MagiCode Files
- `internal/tui/types/types.go`
- `internal/tui/theme.go`
- `internal/tui/leader.go`
- `internal/tui/keybindings.go`
- `internal/tui/app.go`
- `internal/tui/dialog/dialog.go`
- `internal/tui/layout/sidebar.go`

---

## Critical Bugs Found

**See detailed bug report and fix plan:** [`TUI_BUG_REPORT.md`](TUI_BUG_REPORT.md)

### Bug Summary

| Bug | Severity | Status |
|-----|----------|--------|
| Ctrl+P not working | HIGH | Missing implementation |
| Cannot type after Phase 4 | **CRITICAL** | Leader handler blocking input |

### Bug #1: Ctrl+P Shortcut Not Working

The UI claims `Ctrl+P` opens a command palette, but:
- No `dialog/command.go` file exists
- No Ctrl+P keybinding defined
- False claims in help.go, prompt.go, footer.go

### Bug #2: Cannot Type After Phase 4

**Root cause:** Input never updated for KeyMsg due to:
1. Early return at line 206 (`return a.handleKey(msg)`)
2. Leader handler consuming keys at line 808
3. Input update code (lines 315-319) unreachable for KeyMsg

**Impact:** Users cannot type anything in the prompt.

---

## Fix Priority Order

1. **IMMEDIATE:** Fix Bug #2 (cannot type) - Restructure key handling flow
2. **SAME SESSION:** Fix Bug #1 (Ctrl+P claims) - Remove false claims or implement
3. **FUTURE:** Address HIGH priority differences from comparison

---

## References

- **Bug Report:** [`TUI_BUG_REPORT.md`](TUI_BUG_REPORT.md) - Detailed analysis and fix plan
- **Development Guide:** [`../AGENTS.md`](../AGENTS.md) - Current status and patterns

---

**Next Steps:**
1. Fix Bug #2 (critical - cannot type) per TUI_BUG_REPORT.md
2. Fix Bug #1 (Ctrl+P false claims) per TUI_BUG_REPORT.md
3. Address HIGH priority architectural differences
4. Update AGENTS.md with corrected status after fixes