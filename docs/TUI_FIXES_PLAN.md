# MagiCode TUI Fixes - Implementation Plan

## Overview
Fix three critical TUI issues by implementing OpenCode's proven patterns:
1. Auto-scroll to latest message
2. Tab key functionality
3. Session message loading

---

## Issue 1: Auto-Scroll to Latest Message

### Current Problem
- TUI doesn't auto-scroll when new messages arrive
- User has to manually scroll to see new content
- No tracking of user scroll position

### OpenCode Implementation Reference
**File:** `opencode/packages/ui/src/hooks/create-auto-scroll.tsx`

Key concepts:
- Track `userScrolled` state to detect manual scrolling
- `scrollToBottom(force)` function that respects user scroll
- Distance-from-bottom calculation
- ResizeObserver for content changes

### Implementation Plan

#### 1.1 Add Auto-Scroll State to AppState
**File:** `internal/tui/types/types.go`

```go
type LayoutStore struct {
    // ... existing fields ...
    UserScrolled        bool    // Track if user manually scrolled
    AutoScrollEnabled   bool    // Whether auto-scroll is active
}
```

#### 1.2 Create Auto-Scroll Helper Methods
**File:** `internal/tui/app.go`

```go
// scrollToBottom scrolls the viewport to the bottom
// force: if true, scroll regardless of userScrolled state
func (a *App) scrollToBottom(force bool) {
    if !force && a.state.Layout.UserScrolled {
        return // Respect user's scroll position
    }
    
    // Calculate distance from bottom
    viewportHeight := a.messageViewport.Height
    contentHeight := a.messageViewport.TotalLineCount() // or similar
    
    if contentHeight > viewportHeight {
        a.messageViewport.SetYOffset(contentHeight - viewportHeight)
        a.state.Layout.UserScrolled = false
    }
}

// handleViewportScroll detects if user has scrolled away from bottom
func (a *App) handleViewportScroll() {
    viewportHeight := a.messageViewport.Height
    contentHeight := a.messageViewport.TotalLineCount()
    currentOffset := a.messageViewport.YOffset
    
    // If user scrolled up (not at bottom), mark as userScrolled
    if contentHeight > viewportHeight {
        distanceFromBottom := contentHeight - viewportHeight - currentOffset
        if distanceFromBottom > 2 { // Threshold
            a.state.Layout.UserScrolled = true
        } else {
            a.state.Layout.UserScrolled = false
        }
    }
}
```

#### 1.3 Call scrollToBottom at Key Points
**File:** `internal/tui/app.go`

```go
// When new message arrives (in ResponseMsg handler):
case ResponseMsg:
    // ... existing code ...
    a.addMessage(...)
    a.scrollToBottom(false) // Auto-scroll if user hasn't scrolled up
    
// When streaming message completes (StreamMsg):
case StreamMsg:
    a.appendToLastMessage(msg.Content)
    a.scrollToBottom(false)
```

#### 1.4 Handle User Scroll Events
**File:** `internal/tui/app.go` - Update() method

```go
case tea.MouseMsg:
    // Detect scroll wheel events
    if msg.Action == tea.MouseActionScrollUp || msg.Action == tea.MouseActionScrollDown {
        a.handleViewportScroll()
    }
```

### Acceptance Criteria
- [ ] Auto-scrolls to bottom when new message arrives
- [ ] Stops auto-scrolling if user scrolls up (respects user position)
- [ ] Resumes auto-scrolling if user scrolls back to bottom
- [ ] Works for both streaming and complete messages
- [ ] Smooth scroll behavior

---

## Issue 2: Load Messages When Changing Sessions

### Current Problem
- When switching sessions, messages remain empty
- Session content doesn't load properly
- Need to explicitly load messages from database

### OpenCode Implementation Reference
**File:** `opencode/packages/opencode/src/cli/cmd/tui/routes/session/index.tsx`

Key concepts:
- `sync.session.sync(sessionID)` - Explicitly sync session data
- `scroll.scrollBy(100_000)` - Scroll to bottom after loading
- Clear and reload when workspace changes
- Bootstrap sync on session change

### Implementation Plan

#### 2.1 Add Session Switching Logic
**File:** `internal/tui/app.go`

```go
// SwitchSession switches to a different session and loads its messages
func (a *App) SwitchSession(sessionID string) error {
    log.Info("Switching session", "from", a.state.SessionID, "to", sessionID)
    
    // Clear current messages
    a.state.SetMessages([]Message{})
    a.state.SessionID = sessionID
    
    // Load session info from database
    session, err := a.loadSessionFromDB(sessionID)
    if err != nil {
        return fmt.Errorf("failed to load session: %w", err)
    }
    
    a.state.SetActiveSession(session)
    
    // Load messages for this session
    messages, err := a.loadMessagesFromDB(sessionID)
    if err != nil {
        return fmt.Errorf("failed to load messages: %w", err)
    }
    
    a.state.SetMessages(messages)
    a.state.SetStatus(fmt.Sprintf("Loaded %d messages", len(messages)))
    
    // Scroll to bottom
    a.messageViewport.GotoBottom()
    
    // Reset auto-scroll
    a.state.Layout.UserScrolled = false
    
    return nil
}

// loadSessionFromDB loads session info from database
func (a *App) loadSessionFromDB(sessionID string) (*Session, error) {
    // Query database for session
    // Return session struct
}

// loadMessagesFromDB loads messages for a session from database
func (a *App) loadMessagesFromDB(sessionID string) ([]Message, error) {
    // Query database for messages
    // Convert to TUI Message format
    // Return messages
}
```

#### 2.2 Update Dialog Selection Handler
**File:** `internal/tui/app.go` - handleDialogSelection()

```go
func (a *App) handleDialogSelection(msg dialog.SelectMsg) tea.Cmd {
    switch msg.Type {
    case dialog.SelectSession:
        sessionID := msg.Value.(string)
        if err := a.SwitchSession(sessionID); err != nil {
            a.setError(err)
        }
        return nil
    // ... other cases ...
    }
}
```

#### 2.3 Update Initial Load in Init()
**File:** `internal/tui/app.go`

```go
func (a *App) Init() tea.Cmd {
    // ... existing init code ...
    
    // If we have a session ID, load its messages
    if a.state.SessionID != "" {
        return tea.Batch(
            a.spinner.Tick,
            textinput.Blink,
            a.loadSessionsFromDB(),
            func() tea.Msg {
                // Load messages for current session
                messages, err := a.loadMessagesFromDB(a.state.SessionID)
                if err != nil {
                    return ErrorMsg{Error: err}
                }
                return MessagesLoadedMsg{
                    SessionID: sessionID,
                    Messages:  messages,
                }
            },
        )
    }
    
    return tea.Batch(...)
}

// MessagesLoadedMsg is sent when messages are loaded
type MessagesLoadedMsg struct {
    SessionID string
    Messages  []Message
}
```

### Acceptance Criteria
- [ ] When switching sessions, messages from the new session load
- [ ] Previous session messages are cleared before loading new ones
- [ ] Viewport scrolls to bottom after loading
- [ ] Session info (title, directory) updates correctly
- [ ] Works from both dialog selection and initial load
- [ ] Shows loading state while fetching
- [ ] Displays error if loading fails

---

## Issue 3: Tab Key Functionality

### Current Problem
- Tab key does nothing in the prompt
- Should trigger autocomplete or cycle through options

### OpenCode Implementation Reference
**File:** `opencode/packages/opencode/src/cli/cmd/tui/component/prompt/autocomplete.tsx`

Key concepts:
- Tab in autocomplete: expands directories or selects items
- Autocomplete triggers on `@` (files/agents) and `/` (commands)
- Configurable keybindings system
- Tab cycles through completions

### Implementation Plan

#### 3.1 Create Autocomplete State
**File:** `internal/tui/types/types.go`

```go
type DialogStore struct {
    // ... existing fields ...
    Autocomplete AutocompleteState
}

type AutocompleteState struct {
    Visible   bool
    Options   []AutocompleteOption
    Selected  int
    Trigger   string // "@" or "/"
    Query     string
}

type AutocompleteOption struct {
    Value       string
    Display     string
    Description string
    IsDirectory bool
    Icon        string
}
```

#### 3.2 Create Autocomplete Component
**File:** `internal/tui/component/autocomplete.go`

```go
package component

// Autocomplete handles tab completion in the prompt
type Autocomplete struct {
    state  AutocompleteState
    theme  types.Theme
    width  int
    height int
}

// NewAutocomplete creates a new autocomplete component
func NewAutocomplete(theme types.Theme) *Autocomplete {
    return &Autocomplete{
        state: AutocompleteState{
            Visible: false,
            Options: []AutocompleteOption{},
        },
        theme: theme,
    }
}

// Show shows autocomplete with options
func (a *Autocomplete) Show(options []AutocompleteOption, trigger string, query string) {
    a.state.Visible = true
    a.state.Options = options
    a.state.Selected = 0
    a.state.Trigger = trigger
    a.state.Query = query
}

// Hide hides autocomplete
func (a *Autocomplete) Hide() {
    a.state.Visible = false
}

// Next selects next option
func (a *Autocomplete) Next() {
    if len(a.state.Options) > 0 {
        a.state.Selected = (a.state.Selected + 1) % len(a.state.Options)
    }
}

// Previous selects previous option
func (a *Autocomplete) Previous() {
    if len(a.state.Options) > 0 {
        a.state.Selected = (a.state.Selected - 1 + len(a.state.Options)) % len(a.state.Options)
    }
}

// Select returns the currently selected option
func (a *Autocomplete) Select() (AutocompleteOption, bool) {
    if !a.state.Visible || len(a.state.Options) == 0 {
        return AutocompleteOption{}, false
    }
    return a.state.Options[a.state.Selected], true
}

// View renders the autocomplete dropdown
func (a *Autocomplete) View() string {
    if !a.state.Visible {
        return ""
    }
    // Render dropdown with options
}

// Height returns the height needed for the component
func (a *Autocomplete) Height() int {
    if !a.state.Visible {
        return 0
    }
    return min(len(a.state.Options), 10) + 2 // Max 10 items + borders
}
```

#### 3.3 Handle Tab Key in Prompt
**File:** `internal/tui/layout/prompt.go`

```go
// Update handles key events
func (p *Prompt) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
    switch msg := msg.(type) {
    case tea.KeyMsg:
        // If autocomplete is visible, handle Tab
        if p.autocomplete.state.Visible {
            switch msg.Type {
            case tea.KeyTab:
                // Select current option
                if option, ok := p.autocomplete.Select(); ok {
                    p.insertCompletion(option.Value)
                    p.autocomplete.Hide()
                }
                return p, nil
                
            case tea.KeyShiftTab:
                p.autocomplete.Previous()
                return p, nil
                
            case tea.KeyDown:
                p.autocomplete.Next()
                return p, nil
                
            case tea.KeyUp:
                p.autocomplete.Previous()
                return p, nil
                
            case tea.KeyEsc:
                p.autocomplete.Hide()
                return p, nil
            }
        }
        
        // Handle normal Tab (trigger autocomplete)
        if msg.Type == tea.KeyTab {
            p.triggerAutocomplete()
            return p, nil
        }
        
        // ... rest of key handling ...
    }
}

// triggerAutocomplete checks for @ or / and shows completions
func (p *Prompt) triggerAutocomplete() {
    text := p.textarea.Value()
    cursorPos := p.textarea.Position()
    
    // Get word at cursor
    word := p.getWordAtCursor(text, cursorPos)
    
    // Check for trigger characters
    if strings.HasPrefix(word, "@") {
        query := word[1:] // Remove @
        options := p.getFileCompletions(query)
        p.autocomplete.Show(options, "@", query)
    } else if strings.HasPrefix(word, "/") && cursorPos == len(word) {
        query := word[1:] // Remove /
        options := p.getCommandCompletions(query)
        p.autocomplete.Show(options, "/", query)
    }
}

// getWordAtCursor extracts the word at cursor position
func (p *Prompt) getWordAtCursor(text string, pos int) string {
    // Find word boundaries
    start := pos
    for start > 0 && !unicode.IsSpace(rune(text[start-1])) {
        start--
    }
    end := pos
    for end < len(text) && !unicode.IsSpace(rune(text[end])) {
        end++
    }
    return text[start:end]
}

// insertCompletion inserts the completion at cursor
func (p *Prompt) insertCompletion(value string) {
    // Replace word at cursor with completion
}
```

#### 3.4 Provide Completion Sources
**File:** `internal/tui/app.go`

```go
// getFileCompletions returns file/agent completions
func (a *App) getFileCompletions(query string) []component.AutocompleteOption {
    var options []component.AutocompleteOption
    
    // Add agent completions
    for _, agent := range a.state.Local.Agents {
        if strings.Contains(strings.ToLower(agent.Name), strings.ToLower(query)) {
            options = append(options, component.AutocompleteOption{
                Value:       "@" + agent.Name,
                Display:     agent.Name,
                Description: agent.Description,
                Icon:        "🤖",
            })
        }
    }
    
    // Add file completions (from working directory)
    files := a.getFilesInDirectory(query)
    for _, file := range files {
        options = append(options, component.AutocompleteOption{
            Value:       "@" + file.Path,
            Display:     file.Name,
            Description: file.Type,
            IsDirectory: file.IsDir,
            Icon:        getFileIcon(file),
        })
    }
    
    return options
}

// getCommandCompletions returns slash command completions
func (a *App) getCommandCompletions(query string) []component.AutocompleteOption {
    commands := []struct {
        Name        string
        Description string
    }{
        {"status", "Show system status"},
        {"compact", "Compact session history"},
        {"export", "Export session"},
        {"help", "Show help"},
        {"theme", "Change theme"},
        {"model", "Change model"},
        {"agent", "Change agent"},
    }
    
    var options []component.AutocompleteOption
    for _, cmd := range commands {
        if strings.HasPrefix(cmd.Name, query) {
            options = append(options, component.AutocompleteOption{
                Value:       "/" + cmd.Name,
                Display:     cmd.Name,
                Description: cmd.Description,
                Icon:        "⌘",
            })
        }
    }
    return options
}
```

### Acceptance Criteria
- [ ] Tab key triggers autocomplete when typing @ or /
- [ ] Tab selects current autocomplete option
- [ ] Shift+Tab cycles backward through options
- [ ] Up/Down arrows navigate options
- [ ] Escape closes autocomplete
- [ ] @ shows agents and files
- [ ] / shows slash commands
- [ ] Selection inserts text at cursor
- [ ] Autocomplete UI styled with theme
- [ ] Max 10 visible options with scroll

---

## Implementation Order

1. **Fix Auto-Scroll** (Issue 1)
   - Add state tracking
   - Implement scrollToBottom
   - Call at message arrival points

2. **Fix Session Loading** (Issue 2)
   - Create SwitchSession method
   - Load messages from DB
   - Update dialog handlers

3. **Fix Tab Key** (Issue 3)
   - Create autocomplete component
   - Handle Tab key in prompt
   - Provide completion sources

---

## Testing Strategy

### Auto-Scroll Tests
- Send message → should auto-scroll
- Scroll up → should stop auto-scroll
- Scroll to bottom → should resume auto-scroll
- Resize window → should maintain scroll position

### Session Loading Tests
- Switch session → messages should load
- Switch to invalid session → error shown
- Initial load with session ID → messages load
- Clear previous messages before loading new

### Tab Key Tests
- Type @ → autocomplete appears
- Type / → autocomplete appears
- Press Tab → selects option
- Press Escape → closes autocomplete
- Select option → inserts text

---

## Open Questions

1. **Auto-Scroll:** Should we smooth scroll or instant scroll?
2. **Session Loading:** Should we show loading spinner while fetching?
3. **Tab Key:** Should we support fuzzy matching for completions?
4. **Files:** How deep should we search for file completions?
