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
- ✅ Phase 6.1: Tool Result Rendering - Bash, Read, Write, Edit, Glob, Grep, TodoWrite with diff view
- ✅ Phase 6.2: Markdown Rendering - Full markdown support with syntax highlighting
- ✅ Phase 6.3: Command Palette - Ctrl+P searchable command list
- ✅ Phase 6.4: Undo/Redo - Message history undo/redo with stack management
- ✅ Phase 7.1: Animations - Dialog open/close, toast, and scroll animations
- ✅ Phase 7.2: Sound Notifications - Terminal bell support for completion
- ✅ Phase 7.3: Clipboard Support - Cross-platform clipboard operations (Ctrl+X y)
- ✅ Phase 7.4: Help Dialog - Comprehensive keyboard shortcuts reference
- ✅ File Storage Tests - Comprehensive tests for OpenCode v1.2 file-based storage adapter
- ✅ KV Storage Tests - Tests for KV storage with theme preferences
- ✅ Animation Tests - Tests for animation system
- ✅ Clipboard Tests - Tests for clipboard operations

### In Progress
- None (All phases complete!)

### Pending
- None (All phases complete!)

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

## OpenCode Configuration Integration (Phase 1 Complete)

**Status**: ✅ All subtasks complete

### Overview
MagiCode can now read and use OpenCode's agent and provider configuration when the `--use-opencode` flag is used. This enables seamless migration from OpenCode while maintaining memory efficiency.

### Implementation Details

#### Phase 1.1: Config Reader (`internal/opencode/config.go`)

**Files created:**
- `internal/opencode/config.go` - Main config reader
- `internal/opencode/config_test.go` - Comprehensive tests

**Key types:**
```go
type Agent struct {
    Name, Description, Mode, Color string
    Temperature float64
    Tools map[string]bool
    Permission map[string]map[string]string
    Content string  // Full prompt content
    Hidden bool
}

type Provider struct {
    ID, NPM, Name, BaseURL, APIKey string
    Models map[string]Model
}

type Model struct {
    ID, Name string
    Modalities ModelModalities  // input/output types
    Options ModelOptions        // thinking configuration
    Limit ModelLimit           // context/output limits
}

type ConfigReader struct {
    configDir string  // ~/.config/opencode
    stateDir  string  // ~/.local/share/opencode/state
}
```

**Methods:**
- `ReadAgents()` - Reads all agents from `~/.config/opencode/agents/*.md`
- `ReadConfig()` - Reads provider config from `~/.config/opencode/opencode.json`
- `ReadProviders()` - Returns providers as slice
- `ReadModelState()` - Reads recent/favorite models from state
- `GetFirstValidModel()` - Returns first available model
- `ConfigExists()` - Checks if OpenCode config exists

#### Phase 1.2: Type Extensions (`internal/tui/types/types.go`)

**Extended Agent type:**
```go
type Agent struct {
    Name, Description, Mode, Color string
    Temperature float64
    Tools map[string]bool
    Permission map[string]interface{}  // For OpenCode permissions
    Content string                     // Full prompt content
    Hidden bool
}
```

**Extended Provider type:**
```go
type Provider struct {
    ID, Name string
    Models map[string]Model
    Connected bool
    NPM, BaseURL, APIKey string  // OpenCode fields
}
```

**New Model subtypes:**
```go
type ModelModalities struct { Input, Output []string }
type ModelOptions struct { Thinking *ModelThinkingOptions }
type ModelThinkingOptions struct { Type string; BudgetTokens int }
type ModelLimit struct { Context, Output int }
```

**Added methods to AppState:**
- `CycleAgent(direction int)` - Cycles through agents with wrapping
- `CycleModel(direction int)` - Cycles through available models
- `IsModelValid(modelKey) bool` - Checks if model exists in providers
- `GetAgent(name) *Agent` - Gets agent by name
- `GetCurrentAgent() *Agent` - Gets current agent (with fallback)
- `GetAgentColor(name) string` - Returns agent color or default

#### Phase 1.3: Memory-Efficient Config Sync (`internal/tui/app.go`)

**Problem solved:** Loading entire OpenCode config (~500KB-2MB) is wasteful

**Solution:** Minimal initial load with lazy-loading

**Memory-efficient sync flow:**
```
Startup
  ↓
Check session type
  ↓
Existing Session? → Load ONLY model used in last assistant message
                    Load agent names only (no full content)
                    Load single provider for that model
  ↓
New Session?      → Load recent model from state.json
                    Load agent names only
                    Load single provider for default model
```

**Key methods:**
```go
// Main entry point - chooses appropriate sync strategy
func (a *App) syncOpenCodeConfig()

// For existing sessions - preserves context
func (a *App) syncExistingSessionModel(configReader)

// For new sessions - uses defaults
func (a *App) syncNewSessionDefaults(configReader)

// Minimal metadata only (no full prompts)
func (a *App) loadMinimalAgents(configReader) []Agent

// Single provider for specific model
func (a *App) loadMinimalProviderForModel(configReader, modelKey)
```

**Memory comparison:**
- Before: ~500KB-2MB (all agents + all providers + all models)
- After: ~5-10KB (current model + agent names only)

#### Phase 1.4: Safe Model/Agent Switching (`internal/tui/app.go`)

**Problem solved:** Subagents may require different models, cycling needs to handle missing models

**Solution:** Lazy-loading with validation

**Key safety methods:**
```go
// Validates model is loaded, loads if needed
func (a *App) EnsureModelLoaded(modelKey ModelKey) bool

// Switches agent AND ensures its preferred model
func (a *App) SwitchAgent(agentName string) error

// Cycles agents with automatic model loading
func (a *App) SafeCycleAgent(direction int)

// Cycles models with lazy-loading from config
func (a *App) SafeCycleModel(direction int)

// Validates current model, falls back if invalid
func (a *App) ValidateCurrentModel() bool
```

**Usage examples:**
```go
// Safe agent switching (for subagents)
if err := a.SwitchAgent("code-review"); err != nil {
    log.Warn("Subagent switch failed", "error", err)
}

// Safe model cycling
a.SafeCycleModel(+1) // Next model
a.SafeCycleModel(-1) // Previous model

// Validation before operations
if !a.ValidateCurrentModel() {
    a.state.SetStatus("No valid model available")
    return
}
```

**Lazy-loading flow:**
```
User switches agent/subagent invoked
  ↓
SwitchAgent("agent-name")
  ↓
Sets current agent
  ↓
Agent has preferred model?
  ↓ YES
EnsureModelLoaded(agent.Model)
  ↓
Check if model in memory
  ↓ NO
Read from OpenCode config
  ↓
Load provider + model
  ↓
Add to Local.Providers
  ↓
Switch to new model
```

### Configuration

**Enable OpenCode integration:**
```go
app := tui.NewApp(tui.Config{
    UseOpenCode: true,  // Enable config sync
    // ... other config
})
```

**CLI flag (when implemented):**
```bash
magicode --use-opencode
```

### File Locations

**OpenCode config files read:**
- Agents: `~/.config/opencode/agents/*.md`
- Providers: `~/.config/opencode/opencode.json`
- State: `~/.local/share/opencode/state/model.json`

### Testing

**Run tests:**
```bash
go test ./internal/opencode/... -v
go test ./internal/tui/... -short
```

**All tests pass:**
- Config reader tests (11 tests)
- TUI tests (all packages)
- Build successful

---

## Footer Redesign (Phase 2 Complete)

**Status**: ✅ All subtasks complete

### Overview
Footer now displays current agent and model with color-coded agent indicator, matching OpenCode's design.

### Implementation

#### Phase 2.1: Footer Component (`internal/tui/layout/footer.go`)

**Added fields:**
```go
type Footer struct {
    // ... existing fields ...
    agentName  string
    agentColor string
    modelName  string
    modelID    string
}
```

**Added config options:**
```go
type FooterConfig struct {
    ShowDirectory bool
    ShowAgent     bool  // NEW
    ShowModel     bool  // NEW
    ShowLSPMCP    bool
    ShowHint      bool
}
```

**Added methods:**
```go
func (f *Footer) SetAgent(name string, color string)
func (f *Footer) SetModel(name string, id string)
func (f *Footer) GetAgent() string
func (f *Footer) GetModel() string
```

**Responsive Layout:**
- **Narrow (< 60 cols)**: Directory only
- **Medium (60-79 cols)**: Directory + Agent/Model
- **Wide (>= 80 cols)**: Directory + Agent/Model + LSP/MCP + Status hint

#### Phase 2.2: App Integration (`internal/tui/app.go`)

**Added helper methods:**
```go
func (a *App) updateFooterModel(modelKey ModelKey)
func (a *App) updateFooterAgent()
```

**Updated methods:**
- `SwitchAgent()` - Updates footer with new agent/model
- `SafeCycleModel()` - Updates footer after cycling
- `syncExistingSessionModel()` - Initializes footer
- `syncNewSessionDefaults()` - Initializes footer with defaults
- `Init()` - Initializes footer on startup

#### Phase 2.3: Footer Display Format

**Wide (>= 80 cols):**
```
┌────────────────────────────────────────────────────────────────┐
│ ~/projects/myapp    🤖 CodeReview    ⚡ Claude 3.5 Sonnet   3 LSP · 2 MCP │
└────────────────────────────────────────────────────────────────┘
```

**Medium (60-79 cols):**
```
┌─────────────────────────────────────────┐
│ ~/projects/myapp    🤖 CodeReview    ⚡ Claude 3.5...  │
└─────────────────────────────────────────┘
```

**Narrow (< 60 cols):**
```
┌─────────────────────────┐
│ ~/projects/myapp        │
└─────────────────────────┘
```

**Features:**
- Color-coded agent name (uses agent.Color hex value)
- Emoji indicators (🤖 for agent, ⚡ for model)
- Smart truncation for long names
- Responsive layout based on terminal width

### Next Steps (Phase 3)

**Phase 3: Text Wrapping**
- Implement text wrapping for message display
- Handle wide characters and terminal resizing
- Preserve markdown formatting while wrapping

---

## Text Wrapping (Phase 3 Complete)

**Status**: ✅ All subtasks complete

### Overview
Message display now properly wraps text at word boundaries, respecting viewport width and handling wide characters correctly.

### Implementation

#### Phase 3.1: Text Wrapper Utility (`internal/tui/util/wrap.go`)

**Functions:**
```go
// Wrap wraps text at specified width, preserving word boundaries
func Wrap(text string, width int) []string

// WrapPreserveNewlines wraps text but preserves explicit newlines
func WrapPreserveNewlines(text string, width int) []string

// WrapPreserveIndent wraps text while preserving leading indentation
func WrapPreserveIndent(text string, width int) []string

// StringWidth returns display width (handles CJK, emoji, ANSI codes)
func StringWidth(s string) int

// Truncate truncates text with ellipsis
func Truncate(text string, width int, ellipsis string) string

// StripANSI removes ANSI escape codes
func StripANSI(s string) string
```

**Features:**
- Word boundary preservation (doesn't split words mid-word)
- Wide character support (CJK characters have width 2)
- ANSI escape code handling (colors don't count toward width)
- Empty line preservation
- Long word breaking (words longer than width are broken)
- Indentation preservation for code blocks

**Comprehensive Tests:** 10 test cases covering:
- Basic wrapping, long words, empty text
- Newline preservation, indentation preservation
- ANSI handling, truncation, string width calculation

#### Phase 3.2: Message Rendering Updates (`internal/tui/app.go`)

**Updated `buildMessagesContent()`:**
- Calculates available width for text (viewport width - 8 for padding)
- Wraps user messages with continuation line alignment
- Wraps assistant messages with prefix handling
- Wraps system messages
- Maintains proper indentation for multi-line messages

**Example wrapping:**
```
Before:
[14:32] You: This is a very long message that would overflow the viewport and look bad

After:
[14:32] You: This is a very long message that would
          overflow the viewport and look bad
```

**Updated `renderParts()`:**
- Added `width` parameter
- Wraps text parts with `WrapPreserveNewlines()`
- Wraps thinking parts with emoji prefix
- Tool results already handled by component renderer

#### Phase 3.3: Responsive Width Calculation

**Width calculation:**
```go
availableWidth := a.state.Layout.Width - 8  // Subtract padding
if availableWidth < 40 {
    availableWidth = 40  // Minimum width
}
```

**Content width per message type:**
- User messages: `availableWidth - prefixWidth`
- Assistant messages: `availableWidth - prefixWidth`
- System messages: `availableWidth`
- Text parts: `availableWidth` (after header)

#### Phase 3.4: Continuation Line Handling

**Proper alignment:**
```go
prefix := fmt.Sprintf("[%s] You: ", timeStr)
prefixWidth := util.StringWidth(prefix)

wrappedContent := util.WrapPreserveNewlines(msg.Content, contentWidth)
for i, line := range wrappedContent {
    if i == 0 {
        lines = append(lines, prefix+line)
    } else {
        // Align continuation lines with content
        padding := strings.Repeat(" ", prefixWidth)
        lines = append(lines, padding+line)
    }
}
```

### Usage Example

```go
// In message rendering:
contentWidth := availableWidth - prefixWidth
wrappedLines := util.WrapPreserveNewlines(msg.Content, contentWidth)

// For code blocks with indentation:
wrappedCode := util.WrapPreserveIndent(codeBlock, contentWidth)

// For truncation:
shortText := util.Truncate(longText, 50, "...")

// For calculating display width:
width := util.StringWidth(textWithEmojiAndColors)
```

### Responsive to Terminal Resize

Text wrapping automatically adjusts when terminal is resized:
- WindowSizeMsg triggers layout update
- `buildMessagesContent()` recalculates width
- Content re-wraps on next render