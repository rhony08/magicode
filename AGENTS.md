# MagiCode Development Guide

## Project Context

**CRITICAL:** This is a Go recreation of OpenCode addressing significant memory issues:
- **Current OpenCode:** 71GB+ virtual memory at startup
- **Target MagiCode:** < 500MB RSS
- **Key fixes:** Bounded caches, proper cleanup, channel-based events

**Reference:** `docs/GOLANG_RECREATION_PLAN.md` for full architecture details

---

## Build & Test Protocol

**CRITICAL: Always follow this workflow when making changes:**

1. **Write/Update Unit Tests** - Every new feature or bug fix must have corresponding tests
2. **Run Tests** - Execute `go test ./...` to validate changes
3. **Build** - Execute `go build ./cmd/magicode/...` to verify compilation
4. **Fix Issues** - Address any test failures or build errors before proceeding

```bash
# Standard validation workflow
cd /usr/local/projects/new_opencode/magicode
go test ./... -short        # Run all tests
go build ./cmd/magicode/... # Build the application
```

## OpenCode Comparison Protocol

When implementing TUI features, **always compare with the existing OpenCode implementation** in `/usr/local/projects/new_opencode/opencode/packages/`:

| Component | OpenCode Path | MagiCode Path |
|-----------|---------------|---------------|
| State Management | `app/src/context/*.tsx` | `internal/tui/state.go` |
| Theme System | `ui/src/theme/*.ts` | `internal/tui/theme.go` |
| Layout/Sidebar | `app/src/context/layout.tsx` | `internal/tui/app.go` |
| Message Sync | `app/src/context/sync.tsx` | `cmd/magicode/commands/root.go` |

**Before implementing each phase:**
1. Read the corresponding OpenCode source files
2. Document key patterns and behaviors
3. Implement equivalent logic in Go
4. Add tests to verify behavior matches

## Style Guide

### Go Conventions

- Use snake_case for JSON field names (matches database schema)
- Prefer explicit error handling over panic
- Use the project's logging framework (`internal/util/log`)
- Follow Go standard library patterns

### Code Organization

```
internal/tui/
├── app.go           # Main Bubble Tea app (tea.Model interface)
├── state.go         # Type aliases for backward compatibility
├── theme.go         # Theme registry and styles
├── types.go         # Local TUI types (ViewState, InputMode, tea.Msg)
├── keybindings.go   # Keybind system
│
├── types/           # Shared types (breaks import cycles)
│   └── types.go     # Theme, AppState, Session, Message, etc.
│
├── layout/          # Layout components
│   ├── sidebar.go   # Sidebar panel (344px width)
│   ├── footer.go    # Footer bar, StatusBar, KeybindHintBar
│   ├── prompt.go    # Multiline input, PromptWithAttachments
│   └── layout_test.go # Tests for layout components
│
├── dialog/          # Dialog system
│   ├── dialog.go    # Dialog interface, BaseDialog, CloseMsg, SelectMsg
│   ├── session_list.go # Session selection dialog with search
│   ├── model_list.go   # Model selection grouped by provider
│   ├── help.go         # Keyboard shortcuts reference
│   └── dialog_test.go  # Tests for dialog components
│
└── component/       # Reusable components (to be created)
    ├── markdown.go
    └── tool_result.go
```

## Testing

- Avoid mocks where possible - test actual implementation
- Use table-driven tests for multiple test cases
- Run tests from package directories: `go test ./internal/tui`
- Benchmark tests use `go test -bench=.`

## Current TUI Rewrite Status

### Completed
- ✅ Phase 1.1: Message Loading (with SKIP_PARTS filtering)
- ✅ Phase 1.2: State Management Architecture (layered stores)
- ✅ Phase 1.3: Responsive Layout (sidebar auto-hide)
- ✅ Bonus: Theme System (8 built-in themes)
- ✅ Phase 2.1: Sidebar component (344px width, session info, workspace status)
- ✅ Phase 2.2: Footer component (directory, LSP/MCP indicators)
- ✅ Phase 2.3: Prompt component (multiline textarea, attachments support)
- ✅ Phase 2.4: Layout integration into main App (sidebar, footer, prompt components)
- ✅ Import cycle resolution: Created `internal/tui/types` package for shared types
- ✅ Message Pagination: Cursor-based pagination matching OpenCode (80 initial, 200 history)
- ✅ Phase 3.1: Dialog Framework - Dialog interface, stack management, base dialog component
- ✅ Phase 3.2: DialogSessionList - Searchable session selection with fuzzy filter
- ✅ Phase 3.3: DialogModelList - Model selection grouped by provider
- ✅ Phase 3.4: DialogHelp - Keyboard shortcuts organized by category
- ✅ Phase 4.1: Leader Key System - Ctrl+X prefix with 2s timeout
- ✅ Phase 4.2: Keybind Registry - All leader actions mapped (l, n, m, a, etc.)
- ✅ Phase 5.1: Theme Registry - 8 built-in themes (default, catppuccin, dracula, tokyonight, nord, gruvbox, onehalf, solarized)
- ✅ Phase 5.2: Theme File Loading - Load custom themes from ~/.config/magicode/themes/*.json
- ✅ Phase 5.3: Theme Switching Dialog - Ctrl+X t to open theme list dialog
- ✅ Phase 5.4: Theme Persistence - Save/load theme preference to KVStore (SQLite)
- ✅ File Storage Tests - Comprehensive tests for OpenCode v1.2 file-based storage adapter
- ✅ KV Storage Tests - Tests for KV storage with theme preferences

### In Progress
- None (Phase 5 complete)

### Pending
- ⏳ Phase 6: Advanced Features (tool result rendering, markdown, undo/redo)
- ⏳ Phase 7: Polish (animations, sound, clipboard)

## Key Patterns from OpenCode

### State Layers (from OpenCode's context providers)

| Layer | OpenCode | MagiCode Equivalent |
|-------|----------|---------------------|
| GlobalSyncStore | Global data (projects, providers) | SyncStore |
| SyncStore | Per-directory (sessions, messages) | SyncStore.Messages |
| LayoutStore | UI dimensions, sidebar state | LayoutStore |
| LocalStore | Agent/model selection | LocalStore |
| KVStore | Persistent preferences | KVStore |

### SKIP_PARTS Pattern

OpenCode filters internal parts that shouldn't be displayed:
```typescript
const SKIP_PARTS = new Set(["patch", "step-start", "step-finish"])
```

MagiCode equivalent in `convertPartsToTUI()`:
```go
skipParts := map[string]bool{
    "patch":       true,
    "step-start":  true,
    "step-finish": true,
}
```

### Sidebar Dimensions

- OpenCode default: **344px width** (not 42 cols)
- Responsive threshold: Hide when terminal width < 120 cols
- Mobile sidebar: Separate overlay for narrow terminals

### Message Pagination (matches OpenCode)

- **Initial load:** 80 messages (`InitialMessagePageSize`)
- **History load:** 200 messages (`HistoryMessagePageSize`)
- **Cursor:** Timestamp-based (int64 milliseconds) for proper ordering
- **Detection:** Scroll to top triggers `loadMoreMessages()` when `HistoryMore()` returns true
- **State tracking:** `MessageMeta` struct with `Cursor`, `Complete`, `Limit`, `Loading` fields

Key implementation files:
- `internal/database/message.go`: `ListPaginated(ctx, sessionID, limit, cursor)`
- `internal/tui/types/types.go`: `MessageMeta`, `HistoryMore()`, `PrependMessages()`
- `internal/tui/app.go`: `loadMoreMessages()` tea.Cmd, `atTopOfMessages()` detection

```go
// Pagination flow
messages, cursor, complete, err := messageStorage.ListPaginated(ctx, sessionID, 80, 0)
// ... user scrolls to top ...
if a.atTopOfMessages() && a.state.HistoryMore(sessionID) {
    return a, a.loadMoreMessages() // loads 200 more with cursor
}
```

### Leader Key System

OpenCode uses **Ctrl+X** as the leader key prefix for most commands:

```
Ctrl+X + Key = Action
Ctrl+X + l    Session list
Ctrl+X + n    New session
Ctrl+X + m    Model list
Ctrl+X + b    Toggle sidebar
Ctrl+X + h    Help dialog
Ctrl+X + q    Exit app
```

**Implementation:** `internal/tui/leader.go`

- `LeaderKeyHandler` manages state (None, Active, Complete)
- `LeaderTimeout = 2000ms` (2 second timeout)
- `HandleKey()` checks for Ctrl+X first, then buffers second key
- `processSecondKey()` maps keys to LeaderKeyMsg actions
- `handleLeaderAction()` in app.go processes actions

**Timeout handling:**
- `LeaderTimeoutMsg` sent when timeout expires
- Automatically resets leader state
- User can press Escape to cancel leader sequence

**Actions implemented:**
- Session: l, n, x, c, g (list, new, export, compact, timeline)
- Navigation: b (sidebar toggle)
- Model/Agent: m, a (model list, agent list)
- Messages: y, u, r (copy, undo, redo)
- System: t, s, h (theme, status, help)
- Exit: q (quit app)

## Debugging

- Use `log.Info`, `log.Warn`, `log.Error` for logging
- Check LSP diagnostics in-editor after edits
- Run `go test -v ./internal/tui` for verbose test output