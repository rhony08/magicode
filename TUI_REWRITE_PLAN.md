# MagiCode TUI Rewrite Plan

## Goal
Recreate OpenCode's TUI experience in Go using Bubble Tea framework, maintaining full feature parity for seamless user experience.

---

## Current State Analysis

### Existing Go TUI (Simple)
- Basic chat view with message list
- Simple keybindings (Ctrl+S, Ctrl+H, Ctrl+N)
- No sidebar, no dialogs
- No theme system
- Limited state management
- Placeholder message loading

### OpenCode TUI (Complex)
- Leader-key system (Ctrl+X prefix)
- Multiple dialogs (session list, model selection, etc.)
- Sidebar with session/workspace info
- 30+ themes with dark/light support
- Rich markdown/code rendering
- Event-driven real-time updates
- Plugin slots for extensibility
- Responsive layout (width < 120 = sidebar hidden)

---

## Layout Structure (Target)

```
┌──────────────────────────────────────────────────────────────────────────┐
│                              TERMINAL WINDOW                              │
│  ┌────────────────────────────────────────────────────────────────────┐  │
│  │                        MAIN CONTENT AREA                           │  │
│  │  ┌──────────────────────────────────────────────────────────────┐ │  │
│  │  │                  SCROLLABLE MESSAGE AREA                      │ │  │
│  │  │                                                                │ │  │
│  │  │   [User Message]                                              │ │  │
│  │  │   │ <message text>                                            │ │  │
│  │  │   │ [file attachments]                                        │ │  │
│  │  │                                                                │ │  │
│  │  │   [Assistant Message]                                         │ │  │
│  │  │       <thinking block> (muted)                                │ │  │
│  │  │       <markdown content>                                      │ │  │
│  │  │       <tool results>                                          │ │  │
│  │  │       ▣ model-name · duration                                 │ │  │
│  │  │                                                                │ │  │
│  │  └──────────────────────────────────────────────────────────────┘ │  │
│  │                                                                    │  │
│  │  ┌──────────────────────────────────────────────────────────────┐ │  │
│  │  │                        PROMPT AREA                             │ │  │
│  │  │  ┌──────────────────────────────────────────────────────────┐ │ │  │
│  │  │  │ [multiline textarea]                                     │ │ │  │
│  │  │  │ Agent · Model · [variant]                                 │ │ │  │
│  │  │  └──────────────────────────────────────────────────────────┘ │ │  │
│  │  │  [spinner] processing...    [keybind hints: tab, ctrl+p]      │ │  │
│  │  └──────────────────────────────────────────────────────────────┘ │  │
│  └────────────────────────────────────────────────────────────────────┘  │
│                                                                          │
│  ┌─────────────────────────────┐ ┌─────────────────────────────────────┐ │
│  │        SIDEBAR (42 cols)    │ │            FOOTER BAR              │ │
│  │  Session Title              │ │  /path/to/dir    3 LSP · 2 MCP     │ │
│  │  Session ID                 │ │                     /status        │ │
│  │  ● workspace status         │ │                                    │ │
│  │                             │ │                                    │ │
│  │  • MagiCode v1.x.x          │ │                                    │ │
│  └─────────────────────────────┘ └─────────────────────────────────────┘ │
└──────────────────────────────────────────────────────────────────────────┘
```

---

## Keybindings (Target)

### Leader Key System
- Leader: `Ctrl+X` (2000ms timeout)
- Format: `<leader> + key` (e.g., `Ctrl+X` then `l` for session list)

### Keybindings Table

| Category | Keybind | Keys | Description |
|----------|---------|------|-------------|
| **App** | `exit` | `Ctrl+C, Ctrl+D, Ctrl+X q` | Exit |
| | `suspend` | `Ctrl+Z` | Suspend (Unix only) |
| **Navigation** | `session_list` | `Ctrl+X l` | List sessions |
| | `session_new` | `Ctrl+X n` | New session |
| | `sidebar_toggle` | `Ctrl+X b` | Toggle sidebar |
| | `timeline` | `Ctrl+X g` | Message timeline |
| **Messages** | `page_up` | `PgUp, Ctrl+Alt+B` | Scroll up |
| | `page_down` | `PgDn, Ctrl+Alt+F` | Scroll down |
| | `first` | `Ctrl+G, Home` | First message |
| | `last` | `Ctrl+Alt+G, End` | Last message |
| | `copy` | `Ctrl+X y` | Copy last message |
| | `undo` | `Ctrl+X u` | Undo message |
| | `redo` | `Ctrl+X r` | Redo message |
| **Model/Agent** | `model_list` | `Ctrl+X m` | List models |
| | `model_cycle` | `F2` | Cycle model |
| | `agent_list` | `Ctrl+X a` | List agents |
| | `agent_cycle` | `Tab` | Cycle agent |
| | `variant_cycle` | `Ctrl+T` | Cycle variant |
| **Input** | `submit` | `Enter` | Submit prompt |
| | `newline` | `Shift+Enter, Ctrl+J` | New line |
| | `clear` | `Ctrl+C` (in input) | Clear input |
| | `paste` | `Ctrl+V` | Paste |
| | `undo` | `Ctrl+-` | Undo input |
| | `redo` | `Ctrl+.` | Redo input |
| **Session** | `interrupt` | `Esc` | Interrupt |
| | `compact` | `Ctrl+X c` | Compact session |
| | `export` | `Ctrl+X x` | Export |
| **System** | `command_palette` | `Ctrl+P` | Command palette |
| | `theme_list` | `Ctrl+X t` | Themes |
| | `status` | `Ctrl+X s` | Status |
| | `help` | `Ctrl+X h` | Help |

---

## Components (Target)

### Dialogs (Modal Overlays)
1. `DialogSessionList` - List/search sessions
2. `DialogModelList` - Select model
3. `DialogAgentList` - Select agent
4. `DialogThemeList` - Select theme
5. `DialogCommandPalette` - Command search
6. `DialogHelp` - Keyboard shortcuts
7. `DialogStatus` - System status
8. `DialogPermission` - Tool permission prompt
9. `DialogQuestion` - AI question prompt

### Core Components
1. `Sidebar` - Session info, workspace status, LSP/MCP indicators
2. `Footer` - Directory path, status indicators
3. `Prompt` - Multiline textarea with attachments
4. `MessageView` - User/assistant messages with parts
5. `ToolResult` - Bash, Read, Write, Edit, Glob, Grep, Task, TodoWrite
6. `Spinner` - Loading animation
7. `Toast` - Notification popup
8. `MarkdownRenderer` - Markdown with syntax highlighting

---

## State Management (Target)

### State Layers
1. **KVStore** - Persistent preferences (theme, sidebar, timestamps)
2. **SyncStore** - Server-synced data (sessions, messages, providers)
3. **LocalStore** - Agent/model selection
4. **RouteStore** - Navigation (home/session)
5. **DialogStore** - Modal stack
6. **ThemeStore** - Current theme

### Store Structure
```go
type AppState struct {
    // Route
    Route       Route       // "home" or "session"
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

---

## Theme System (Target)

### Built-in Themes (Priority)
1. `default` - Purple primary (current)
2. `catppuccin` - Soft pastels
3. `dracula` - Dark purple/green
4. `tokyonight` - Blue/orange
5. `nord` - Arctic blue
6. `gruvbox` - Retro warm
7. `onehalf` - Clean dark/light
8. `solarized` - Classic

### Theme Structure
```go
type Theme struct {
    Name        string
    Dark        bool
    
    // Primary
    Primary     Color
    Secondary   Color
    Accent      Color
    
    // Status
    Error       Color
    Warning     Color
    Success     Color
    Info        Color
    
    // Text
    Text        Color
    TextMuted   Color
    
    // Background
    Background  Color
    PanelBg     Color
    ElementBg   Color
    MenuBg      Color
    
    // Border
    Border      Color
    BorderActive Color
    
    // Diff
    Added       Color
    Removed     Color
    AddedBg     Color
    RemovedBg   Color
    
    // Markdown
    Heading     Color
    Link        Color
    Code        Color
    BlockQuote  Color
    
    // Syntax
    Comment     Color
    Keyword     Color
    Function    Color
    String      Color
    Number      Color
}
```

---

## Implementation Phases

### Phase 1: Foundation (Priority)
1. **Message Loading** - Load session messages from database on continue
2. **State Architecture** - Refactor App struct with proper stores
3. **Responsive Layout** - Auto-hide sidebar on narrow terminals

### Phase 2: Core Components
1. **Sidebar** - Session info panel with toggle
2. **Footer** - Directory + LSP/MCP status
3. **Prompt** - Multiline textarea
4. **Message Rendering** - Proper user/assistant styling

### Phase 3: Dialog System
1. **Dialog Framework** - Stack-based modal overlay
2. **DialogSessionList** - Fuzzy searchable
3. **DialogModelList** - Model selection
4. **DialogHelp** - Keyboard shortcuts reference

### Phase 4: Keybindings
1. **Leader Key System** - Ctrl+X prefix handling
2. **Keybind Registry** - Configurable shortcuts
3. **Platform Differences** - Windows suspend handling

### Phase 5: Theme System
1. **Theme Registry** - Built-in themes
2. **Theme File Loading** - Custom JSON themes
3. **Theme Switching** - Dialog + persistence

### Phase 6: Advanced Features
1. **Tool Result Rendering** - Bash, Edit diffs, etc.
2. **Markdown Rendering** - Syntax highlighting
3. **Command Palette** - Searchable commands
4. **Undo/Redo** - Message history navigation

### Phase 7: Polish
1. **Animations** - Smooth transitions
2. **Sound** - Completion notification
3. **Clipboard** - Copy message content
4. **Help Dialog** - Full keybind reference

---

## File Structure (Target)

```
internal/tui/
├── app.go              # Main Bubble Tea app
├── types.go            # All type definitions
├── keybindings.go      # Keybind system with leader
├── theme.go            # Theme registry and loading
├── state.go            # State management helpers
│
├── layout/
│   ├── root.go         # Root layout manager
│   ├── sidebar.go      # Sidebar panel
│   ├── footer.go       # Footer bar
│   └── prompt.go       # Input area
│
├── view/
│   ├── home.go         # Home/landing view
│   ├── session.go      # Session/conversation view
│   └── message.go      # Message rendering
│
├── dialog/
│   ├── manager.go      # Dialog stack management
│   ├── base.go         # Base dialog component
│   ├── session_list.go # Session list dialog
│   ├── model_list.go   # Model selection
│   ├── agent_list.go   # Agent selection
│   ├── theme_list.go   # Theme selection
│   ├── help.go         # Help reference
│   ├── command.go      # Command palette
│   └── status.go       # System status
│
├── component/
│   ├── spinner.go      # Loading spinner
│   ├── toast.go        # Notification popup
│   ├── markdown.go     # Markdown renderer
│   ├── tool_result.go  # Tool call display
│   ├── diff.go         # Edit diff viewer
│   └── border.go       # Custom border chars
│
└── util/
│   ├── clipboard.go    # Clipboard operations
│   ├── scroll.go       # Scroll calculations
│   └── fuzzy.go        # Fuzzy search
```

---

## Immediate Priority: Message Loading

### Current Issue
When continuing a session with `--session <id>`:
- Session metadata is loaded (directory, title)
- Messages are NOT loaded into TUI chat history
- User sees empty conversation

### Fix Required
1. Load messages from database for session ID
2. Populate TUI messages array
3. Render all previous user/assistant/tool messages
4. Scroll to last message

### Implementation
```go
// In root.go runTUI()
if sessionID != "" {
    // Load messages
    messageStorage := database.NewMessageStorage(db)
    messages, err := messageStorage.List(ctx, sessionID)
    if err != nil {
        return fmt.Errorf("failed to load messages: %w", err)
    }
    
    // Convert to TUI messages
    tuiMessages := convertMessages(messages)
    
    // Pass to TUI config
    tuiConfig.Messages = tuiMessages
}
```

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

## Estimated Timeline

| Phase | Duration | Priority |
|-------|----------|----------|
| Phase 1: Foundation | 1 day | HIGH |
| Phase 2: Core Components | 2 days | HIGH |
| Phase 3: Dialog System | 2 days | HIGH |
| Phase 4: Keybindings | 1 day | MEDIUM |
| Phase 5: Theme System | 1 day | MEDIUM |
| Phase 6: Advanced Features | 3 days | MEDIUM |
| Phase 7: Polish | 1 day | LOW |

**Total: ~10-11 days for full parity**

---

## Notes

- Focus on Phase 1 first (message loading is blocking UX)
- Keybindings should match OpenCode exactly for easy transition
- Theme system can be simpler initially (5-6 themes)
- Plugin slots are NOT needed for initial version
- SolidJS patterns translate to Bubble Tea Cmd/Msg pattern