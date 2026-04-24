# MagiCode TUI Rewrite - Todo List

**Generated:** 2026-04-24  
**Total Tasks:** 25  
**Timeline:** ~10-11 days

---

## Summary

- **HIGH Priority:** 11 tasks (Phases 1-3)
- **MEDIUM Priority:** 10 tasks (Phases 4-6)
- **LOW Priority:** 4 tasks (Phase 7)

---

## Phase 1: Foundation (HIGH - 1 day)

### ✅ Phase 1.1: Implement Message Loading
**Priority:** HIGH | **Status:** Pending

**Description:**  
Load session messages from database on continue. Fix critical issue where `--session <id>` flag shows empty conversation.

**Tasks:**
- [ ] Create `messageStorage.List()` call in `runTUI()` function
- [ ] Load messages from database for given session ID
- [ ] Convert database message format to TUI message format
- [ ] Populate `App.messages` array with converted messages
- [ ] Implement scroll to last message after loading
- [ ] Test with existing sessions

**Acceptance Criteria:**
- Messages load correctly when continuing a session
- All previous user/assistant/tool messages render
- Cursor positioned at last message
- No performance issues with large conversations

**Files:**
- `internal/tui/app.go` (message loading logic)
- `internal/database/message.go` (storage interface)

---

### ✅ Phase 1.2: Refactor State Management Architecture
**Priority:** HIGH | **Status:** Pending

**Description:**  
Refactor App struct with proper state management. Separate concerns into distinct store layers.

**Tasks:**
- [ ] Design `AppState` struct with all required fields
- [ ] Implement state layers:
  - **KVStore:** Persistent preferences (theme, sidebar, timestamps)
  - **SyncStore:** Server-synced data (sessions, messages, providers)
  - **LocalStore:** UI state (agent/model selection)
  - **RouteStore:** Navigation state (home/session)
  - **DialogStore:** Modal stack
  - **ThemeStore:** Current theme
- [ ] Create `internal/tui/state.go` with state helpers
- [ ] Migrate existing App struct to use AppState
- [ ] Add state update methods

**AppState Structure:**
```go
type AppState struct {
    // Route
    Route       Route      // "home" or "session"
    SessionID   string
    
    // Sync (from server/database)
    Sessions    []Session
    Messages    []Message
    Providers   []Provider
    Agents      []Agent
    
    // Local
    CurrentAgent string
    CurrentModel string
    ModelVariant string
    
    // KV (persistent)
    Theme       string
    SidebarMode string  // "auto", "show", "hide"
    
    // Dialog
    DialogStack []Dialog
    
    // Input
    InputText   string
    InputParts  []InputPart  // attachments
    
    // Status
    Processing  bool
    StatusText  string
    Toast       *ToastMsg
    
    // Dimensions
    Width       int
    Height      int
}
```

**Files:**
- `internal/tui/types.go` (type definitions)
- `internal/tui/state.go` (state management)

---

### ✅ Phase 1.3: Implement Responsive Layout
**Priority:** HIGH | **Status:** Pending

**Description:**  
Auto-hide sidebar on narrow terminals. Match OpenCode's responsive behavior.

**Tasks:**
- [ ] Add `Width` and `Height` to AppState
- [ ] Create layout calculation logic in root view
- [ ] Set sidebar hide threshold at width < 120
- [ ] Implement conditional sidebar rendering
- [ ] Handle terminal resize events (Bubble Tea WindowSizeMsg)
- [ ] Update layout on dimension changes

**Acceptance Criteria:**
- Sidebar visible when width >= 120
- Sidebar hidden when width < 120
- Smooth transition on resize
- Layout recalculates correctly

**Files:**
- `internal/tui/layout/root.go`

---

## Phase 2: Core Components (HIGH - 2 days)

### ✅ Phase 2.1: Create Sidebar Component
**Priority:** HIGH | **Status:** Pending

**Description:**  
Create sidebar panel (42 cols) displaying session info and status.

**Tasks:**
- [ ] Create `internal/tui/layout/sidebar.go`
- [ ] Implement `View()` method with lipgloss styling
- [ ] Display components:
  - Session title
  - Session ID
  - Workspace status (● indicator)
  - MagiCode version (v1.x.x)
- [ ] Add toggle functionality (Ctrl+X b)
- [ ] Handle sidebar mode: auto, show, hide
- [ ] Width: 42 columns (matches OpenCode)

**Layout:**
```
┌─────────────────────────────┐
│        SIDEBAR (42 cols)    │
│  Session Title              │
│  Session ID                 │
│  ● workspace status         │
│                             │
│  • MagiCode v1.x.x          │
└─────────────────────────────┘
```

**Files:**
- `internal/tui/layout/sidebar.go`

---

### ✅ Phase 2.2: Create Footer Component
**Priority:** HIGH | **Status:** Pending

**Description:**  
Create footer bar showing directory and status indicators.

**Tasks:**
- [ ] Create `internal/tui/layout/footer.go`
- [ ] Display components:
  - Current directory path
  - LSP server count (e.g., "3 LSP")
  - MCP server count (e.g., "2 MCP")
  - Status command hint (/status)
- [ ] Update on workspace changes
- [ ] Style with lipgloss

**Layout:**
```
┌─────────────────────────────────────────┐
│  /path/to/dir    3 LSP · 2 MCP   /status│
└─────────────────────────────────────────┘
```

**Files:**
- `internal/tui/layout/footer.go`

---

### ✅ Phase 2.3: Create Prompt Component
**Priority:** HIGH | **Status:** Pending

**Description:**  
Create multiline textarea for user input with attachment support.

**Tasks:**
- [ ] Create `internal/tui/layout/prompt.go`
- [ ] Use Bubble Tea textarea component
- [ ] Support file attachments (InputParts)
- [ ] Display current agent/model/variant
- [ ] Show keybind hints (tab, ctrl+p)
- [ ] Show processing spinner during AI response
- [ ] Handle keybindings:
  - `Enter`: Submit prompt
  - `Shift+Enter` / `Ctrl+J`: New line
  - `Ctrl+C`: Clear input
  - `Ctrl+V`: Paste
  - `Ctrl+-`: Undo input
  - `Ctrl+.`: Redo input

**Layout:**
```
┌──────────────────────────────────────────┐
│  [multiline textarea]                     │
│  Agent · Model · [variant]                │
│  [spinner] processing...  [tab, ctrl+p]  │
└──────────────────────────────────────────┘
```

**Files:**
- `internal/tui/layout/prompt.go`

---

### ✅ Phase 2.4: Implement Message Rendering
**Priority:** HIGH | **Status:** Pending

**Description:**  
Create MessageView component with proper styling for user/assistant messages.

**Tasks:**
- [ ] Create `internal/tui/view/message.go`
- [ ] Implement user message styling:
  - Message text
  - File attachments
- [ ] Implement assistant message styling:
  - Thinking blocks (muted color)
  - Markdown content
  - Tool results
  - Model name and duration (▣ indicator)
- [ ] Support scrolling navigation:
  - `PgUp` / `Ctrl+Alt+B`: Scroll up
  - `PgDn` / `Ctrl+Alt+F`: Scroll down
  - `Ctrl+G` / `Home`: First message
  - `Ctrl+Alt+G` / `End`: Last message
- [ ] Style with theme colors

**Layout:**
```
[User Message]
│ <message text>
│ [file attachments]

[Assistant Message]
    <thinking block> (muted)
    <markdown content>
    <tool results>
    ▣ model-name · duration
```

**Files:**
- `internal/tui/view/message.go`

---

## Phase 3: Dialog System (HIGH - 2 days)

### ✅ Phase 3.1: Build Dialog Framework
**Priority:** HIGH | **Status:** Pending

**Description:**  
Create stack-based modal overlay system for dialogs.

**Tasks:**
- [ ] Create Dialog interface:
  ```go
  type Dialog interface {
      Init() tea.Cmd
      Update(msg tea.Msg) (Dialog, tea.Cmd)
      View() string
  }
  ```
- [ ] Implement dialog stack in AppState
- [ ] Create dialog manager:
  - Push/Pop dialogs
  - Focus management
  - Render on top of main content
- [ ] Create base dialog component with common functionality
- [ ] Handle Escape key to close dialogs

**Files:**
- `internal/tui/dialog/manager.go`
- `internal/tui/dialog/base.go`

---

### ✅ Phase 3.2: Create DialogSessionList
**Priority:** HIGH | **Status:** Pending

**Description:**  
Create fuzzy searchable session selection dialog.

**Tasks:**
- [ ] Create `internal/tui/dialog/session_list.go`
- [ ] List all sessions with:
  - Title
  - Directory
  - Date
- [ ] Implement fuzzy search filter
- [ ] Keybindings:
  - `Enter`: Select session
  - `Esc`: Close dialog
  - Arrow keys: Navigate
- [ ] Show keybind hints
- [ ] Style with theme

**Files:**
- `internal/tui/dialog/session_list.go`

---

### ✅ Phase 3.3: Create DialogModelList
**Priority:** HIGH | **Status:** Pending

**Description:**  
Create model selection dialog.

**Tasks:**
- [ ] Create `internal/tui/dialog/model_list.go`
- [ ] List all available models from providers
- [ ] Show current selection indicator
- [ ] Support cycling models (`F2`)
- [ ] Implement search/filter functionality
- [ ] Keybindings:
  - `Enter`: Select model
  - `Esc`: Close
  - Arrow keys: Navigate

**Files:**
- `internal/tui/dialog/model_list.go`

---

### ✅ Phase 3.4: Create DialogHelp
**Priority:** HIGH | **Status:** Pending

**Description:**  
Create keyboard shortcuts reference dialog.

**Tasks:**
- [ ] Create `internal/tui/dialog/help.go`
- [ ] Display all keybindings organized by category:
  - App (exit, suspend)
  - Navigation (session list, sidebar, timeline)
  - Messages (scroll, copy, undo, redo)
  - Model/Agent (model list, agent list, cycling)
  - Input (submit, newline, clear, paste)
  - Session (interrupt, compact, export)
  - System (command palette, theme, status)
- [ ] Show leader key notation (Ctrl+X)
- [ ] Support scrolling for long lists
- [ ] Keybindings:
  - `Esc`: Close
  - Arrow keys: Navigate

**Files:**
- `internal/tui/dialog/help.go`

---

## Phase 4: Keybindings (MEDIUM - 1 day)

### ✅ Phase 4.1: Implement Leader Key System
**Priority:** MEDIUM | **Status:** Pending

**Description:**  
Implement Ctrl+X prefix handling with timeout.

**Tasks:**
- [ ] Track leader key state (waiting/not-waiting)
- [ ] Buffer keys after leader press
- [ ] Execute keybind on second key press
- [ ] Implement 2000ms timeout (resets state)
- [ ] Support keybindings:
  - `Ctrl+X l`: Session list
  - `Ctrl+X n`: New session
  - `Ctrl+X b`: Toggle sidebar
  - `Ctrl+X g`: Message timeline
  - `Ctrl+X y`: Copy last message
  - `Ctrl+X u`: Undo message
  - `Ctrl+X r`: Redo message
  - `Ctrl+X m`: Model list
  - `Ctrl+X a`: Agent list
  - `Ctrl+X c`: Compact session
  - `Ctrl+X x`: Export
  - `Ctrl+X t`: Themes
  - `Ctrl+X s`: Status
  - `Ctrl+X h`: Help
  - `Ctrl+X q`: Exit

**Files:**
- `internal/tui/keybindings.go`

---

### ✅ Phase 4.2: Create Keybind Registry
**Priority:** MEDIUM | **Status:** Pending

**Description:**  
Create configurable keybind registry.

**Tasks:**
- [ ] Define KeyBind struct:
  ```go
  type KeyBind struct {
      Category    string
      Name        string
      Keys        []string
      Description string
  }
  ```
- [ ] Create registry with all keybindings from plan table
- [ ] Support multiple key sequences per action:
  - `Ctrl+C`, `Ctrl+D`, `Ctrl+X q`: Exit
- [ ] Enable runtime lookup
- [ ] Generate help from registry

**Files:**
- `internal/tui/keybindings.go`

---

### ✅ Phase 4.3: Handle Platform Differences
**Priority:** MEDIUM | **Status:** Pending

**Description:**  
Handle Windows vs Unix differences for suspend.

**Tasks:**
- [ ] Unix: `Ctrl+Z` sends SIGTSTP (suspend process)
- [ ] Windows: Skip or show "not available" message
- [ ] Add build tags or runtime OS checks
- [ ] Test on both platforms
- [ ] Update documentation

**Files:**
- `internal/tui/keybindings.go`

---

## Phase 5: Theme System (MEDIUM - 1 day)

### ✅ Phase 5.1: Implement Theme Registry
**Priority:** MEDIUM | **Status:** Pending

**Description:**  
Create theme registry with 8 priority built-in themes.

**Tasks:**
- [ ] Create Theme struct with all color fields:
  - Primary, Secondary, Accent
  - Error, Warning, Success, Info
  - Text, TextMuted
  - Background, PanelBg, ElementBg, MenuBg
  - Border, BorderActive
  - Added, Removed, AddedBg, RemovedBg (diff)
  - Heading, Link, Code, BlockQuote (markdown)
  - Comment, Keyword, Function, String, Number (syntax)
- [ ] Implement 8 themes:
  1. `default` - Purple primary
  2. `catppuccin` - Soft pastels
  3. `dracula` - Dark purple/green
  4. `tokyonight` - Blue/orange
  5. `nord` - Arctic blue
  6. `gruvbox` - Retro warm
  7. `onehalf` - Clean dark/light
  8. `solarized` - Classic
- [ ] Create theme registry map

**Files:**
- `internal/tui/theme.go`

---

### ✅ Phase 5.2: Add Theme File Loading
**Priority:** MEDIUM | **Status:** Pending

**Description:**  
Support loading custom themes from JSON files.

**Tasks:**
- [ ] Support loading from `~/.config/magicode/themes/*.json`
- [ ] Parse JSON into Theme struct
- [ ] Validate required fields
- [ ] Merge with defaults for missing fields
- [ ] Handle errors gracefully
- [ ] Allow users to create custom themes

**Files:**
- `internal/tui/theme.go`

---

### ✅ Phase 5.3: Create Theme Switching Dialog
**Priority:** MEDIUM | **Status:** Pending

**Description:**  
Create theme list dialog with persistence.

**Tasks:**
- [ ] Create `internal/tui/dialog/theme_list.go`
- [ ] Show theme list (`Ctrl+X t`)
- [ ] Preview theme names
- [ ] Select to apply theme
- [ ] Save preference to KVStore
- [ ] Apply theme immediately to all components
- [ ] Keybindings:
  - `Enter`: Select theme
  - `Esc`: Close

**Files:**
- `internal/tui/dialog/theme_list.go`

---

## Phase 6: Advanced Features (MEDIUM - 3 days)

### ✅ Phase 6.1: Implement Tool Result Rendering
**Priority:** MEDIUM | **Status:** Pending

**Description:**  
Create renderers for tool call results.

**Tasks:**
- [ ] Create `internal/tui/component/tool_result.go`
- [ ] Create `internal/tui/component/diff.go`
- [ ] Implement renderers:
  - **Bash**: Show command, output, truncate if long
  - **Read**: File path, line numbers
  - **Write**: File path, content preview
  - **Edit**: Show diff with added/removed lines
  - **Glob**: File list
  - **Grep**: Matches
  - **Task**: Description, status
  - **TodoWrite**: Todo list display
- [ ] Use theme colors for diff (Added, Removed)

**Files:**
- `internal/tui/component/tool_result.go`
- `internal/tui/component/diff.go`

---

### ✅ Phase 6.2: Add Markdown Rendering
**Priority:** MEDIUM | **Status:** Pending

**Description:**  
Add markdown parsing and syntax highlighting.

**Tasks:**
- [ ] Create `internal/tui/component/markdown.go`
- [ ] Parse markdown elements:
  - Headers (h1-h6)
  - Lists (ordered, unordered)
  - Code blocks (fenced)
  - Inline code
  - Links
  - Blockquotes
  - Bold, italic
- [ ] Apply theme colors:
  - Heading
  - Link
  - Code
  - BlockQuote
- [ ] Syntax highlighting for code blocks (use chroma or similar)
- [ ] Support inline code styling

**Files:**
- `internal/tui/component/markdown.go`

---

### ✅ Phase 6.3: Create Command Palette
**Priority:** MEDIUM | **Status:** Pending

**Description:**  
Create searchable commands dialog.

**Tasks:**
- [ ] Create `internal/tui/dialog/command.go`
- [ ] Trigger with `Ctrl+P`
- [ ] List all available commands/actions
- [ ] Implement fuzzy search by name
- [ ] Show keybind hint for each command
- [ ] Keybindings:
  - `Enter`: Execute command
  - `Esc`: Close
  - Arrow keys: Navigate

**Files:**
- `internal/tui/dialog/command.go`

---

### ✅ Phase 6.4: Implement Undo/Redo
**Priority:** MEDIUM | **Status:** Pending

**Description:**  
Implement message history navigation.

**Tasks:**
- [ ] Track message edit history
- [ ] Support keybindings:
  - `Ctrl+X u`: Undo message
  - `Ctrl+X r`: Redo message
- [ ] Store undo stack per session
- [ ] Revert last user message edit
- [ ] Useful for correcting typos before resubmit

**Files:**
- `internal/tui/state.go`

---

## Phase 7: Polish (LOW - 1 day)

### ✅ Phase 7.1: Add Smooth Animations
**Priority:** LOW | **Status:** Pending

**Description:**  
Add smooth animations and transitions.

**Tasks:**
- [ ] Animate dialog open/close (fade in/out or slide)
- [ ] Animate toast notifications
- [ ] Smooth scroll to new messages
- [ ] Use Bubble Tea frame-based animation
- [ ] Not critical for MVP but improves UX

**Files:**
- `internal/tui/dialog/*.go`
- `internal/tui/component/toast.go`

---

### ✅ Phase 7.2: Add Completion Notification Sounds
**Priority:** LOW | **Status:** Pending

**Description:**  
Play sound when AI finishes response.

**Tasks:**
- [ ] Play sound when AI finishes (optional)
- [ ] Use terminal bell or system sound
- [ ] Make configurable in settings
- [ ] Useful when user is multitasking
- [ ] Not critical for MVP

**Files:**
- `internal/tui/app.go`

---

### ✅ Phase 7.3: Implement Clipboard Support
**Priority:** LOW | **Status:** Pending

**Description:**  
Support copying message content to clipboard.

**Tasks:**
- [ ] Create `internal/tui/util/clipboard.go`
- [ ] Support keybindings:
  - `Ctrl+X y`: Copy last message
- [ ] Copy code blocks
- [ ] Copy tool results
- [ ] Use platform clipboard:
  - Linux: `xclip`/`xsel`
  - macOS: `pbcopy`
  - Windows: `clip`

**Files:**
- `internal/tui/util/clipboard.go`

---

### ✅ Phase 7.4: Create Comprehensive Help Dialog
**Priority:** LOW | **Status:** Pending

**Description:**  
Expand help dialog with full reference.

**Tasks:**
- [ ] Expand DialogHelp to show all keybindings
- [ ] Add descriptions for each keybind
- [ ] Organize by category
- [ ] Support pagination for long lists
- [ ] Show current theme
- [ ] Show version info
- [ ] Nice-to-have for user onboarding

**Files:**
- `internal/tui/dialog/help.go`

---

## Notes

- **Phase 1 is highest priority** (message loading is blocking UX)
- **Phases 2-3 are critical** for feature parity with OpenCode
- **Phase 4-6 are important** for user experience
- **Phase 7 is nice-to-have** for polish
- Keybindings should match OpenCode exactly for easy transition
- Theme system can be simpler initially (5-6 themes)
- Plugin slots are NOT needed for initial version
- SolidJS patterns translate to Bubble Tea Cmd/Msg pattern

---

## Success Criteria

1. ✅ Same layout as OpenCode (sidebar, footer, prompt)
2. ✅ Same keybindings (leader key system)
3. ✅ Same dialogs (session list, model, help)
4. ✅ Same themes (catppuccin, dracula, etc.)
5. ✅ Message history loads on session continue
6. ✅ Responsive layout (sidebar auto-hide)
7. ✅ Smooth UX for existing OpenCode users

---

**Total Estimated Time:** 10-11 days for full parity