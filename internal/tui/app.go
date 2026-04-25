// Package tui provides the main TUI application.
// This file implements the Bubble Tea Model interface with layered state management.
package tui

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/google/uuid"
	"github.com/rhony08/magicode/internal/database"
	"github.com/rhony08/magicode/internal/opencode"
	"github.com/rhony08/magicode/internal/tui/component"
	"github.com/rhony08/magicode/internal/tui/dialog"
	"github.com/rhony08/magicode/internal/tui/layout"
	"github.com/rhony08/magicode/internal/util/log"
)

// App is the main TUI application implementing tea.Model
type App struct {
	// Core state - layered architecture
	state  AppState
	theme  Theme
	styles ThemeStyles

	// Database path for message loading
	databasePath string

	// Working directory for session creation
	workingDirectory string

	// UI components - layout package
	sidebar       *layout.Sidebar
	mobileSidebar *layout.MobileSidebar
	footer        *layout.Footer
	statusBar     *layout.StatusBar
	prompt        *layout.Prompt
	keybindHints  *layout.KeybindHintBar

	// Active dialog (if any)
	activeDialog dialog.Dialog

	// Leader key handler
	leaderHandler *LeaderKeyHandler

	// Legacy components (kept for compatibility)
	input           textinput.Model
	spinner         spinner.Model
	messageViewport viewport.Model

	// Legacy compatibility
	view    ViewState
	mode    InputMode
	focused bool

	// Help
	showHelp    bool
	helpContent string

	// Keybindings
	keybindings Keybindings

	// Error timer (for auto-dismiss)
	errorTimer *time.Timer
}

// Config represents app configuration for initialization
type Config struct {
	Title           string
	Session         Session
	InitialMessages []Message   // Pre-loaded messages for session continuation
	SessionID       string      // Session ID for loading messages
	Theme           string      // Theme name (optional, defaults to "default")
	Directory       string      // Working directory
	MessageMeta     MessageMeta // Pagination metadata for initial load
	DatabasePath    string      // Database path for loading more messages
}

// workingDirectory returns the working directory from config or current directory
func (c Config) workingDirectory() string {
	if c.Directory != "" {
		return c.Directory
	}
	// Try to get current directory
	wd, err := os.Getwd()
	if err == nil {
		return wd
	}
	return "."
}

// NewApp creates a new TUI application with layered state management
func NewApp(cfg Config) *App {
	// Initialize state
	state := NewAppState()
	state.SessionID = cfg.SessionID
	state.Route = RouteSession
	state.SetStatus("Ready")

	// Apply initial configuration
	if cfg.Session.ID != "" {
		state.SetActiveSession(&cfg.Session)
		state.Sync.Sessions = []Session{cfg.Session}
	}
	if len(cfg.InitialMessages) > 0 {
		state.SetMessages(cfg.InitialMessages)

		// Set pagination metadata from config
		meta := cfg.MessageMeta
		meta.Limit = len(cfg.InitialMessages)
		state.SetMessageMeta(cfg.SessionID, meta)

		state.SetStatus(fmt.Sprintf("Loaded %d messages", len(cfg.InitialMessages)))
	}
	if cfg.Theme != "" {
		state.SetTheme(cfg.Theme)
	}

	// Get theme
	theme := GetTheme(state.KV.Theme)
	styles := ApplyTheme(theme)

	// Create input component
	ti := textinput.New()
	ti.Placeholder = "Type your message..."
	ti.Focus()
	ti.CharLimit = 5000
	ti.Width = 50

	// Create spinner component
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(theme.Spinner)

	// Create message viewport
	vp := viewport.New(80, 20)

	// Default help content
	help := buildHelpContent()

	// Create layout components
	sidebar := layout.NewSidebar(layout.DefaultSidebarConfig(), theme)
	mobileSidebar := layout.NewMobileSidebar(theme)
	footer := layout.NewFooter(layout.DefaultFooterConfig(), theme)
	statusBar := layout.NewStatusBar(layout.DefaultFooterConfig(), theme)
	prompt := layout.NewPrompt(layout.DefaultPromptConfig(), theme)
	keybindHints := layout.NewKeybindHintBar(theme)

	app := &App{
		state:            state,
		theme:            theme,
		styles:           styles,
		databasePath:     cfg.DatabasePath,
		workingDirectory: cfg.workingDirectory(),
		sidebar:          sidebar,
		mobileSidebar:    mobileSidebar,
		footer:           footer,
		statusBar:        statusBar,
		prompt:           prompt,
		keybindHints:     keybindHints,
		leaderHandler:    NewLeaderKeyHandler(),
		input:            ti,
		spinner:          s,
		messageViewport:  vp,
		view:             ViewChat,
		mode:             ModeInput,
		helpContent:      help,
		keybindings:      DefaultKeybindings(),
	}

	return app
}

// Init initializes the app (tea.Model interface)
func (a *App) Init() tea.Cmd {
	// Load theme preference from database if available
	if a.databasePath != "" {
		savedTheme := a.loadThemePreference()
		if savedTheme != "" && savedTheme != a.state.KV.Theme {
			a.state.SetTheme(savedTheme)
			a.theme = GetTheme(savedTheme)
			a.styles = ApplyTheme(a.theme)
			a.spinner.Style = lipgloss.NewStyle().Foreground(a.theme.Spinner)
		}
	}

	return tea.Batch(
		a.spinner.Tick,
		textinput.Blink,
		a.loadSessionsFromDB(), // Load sessions from database on startup
	)
}

// Update handles events (tea.Model interface)
func (a *App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	// Handle resize events - update state dimensions
	if wsMsg, ok := msg.(tea.WindowSizeMsg); ok {
		a.state.SetDimensions(wsMsg.Width, wsMsg.Height)
		a.updateViewportSize()
		a.input.Width = a.state.Layout.Width - 20

		// Update layout components
		a.sidebar.SetDimensions(a.state.KV.SidebarWidth/8, a.state.Layout.Height-6)
		a.footer.SetWidth(a.state.Layout.Width)
		a.statusBar.SetWidth(a.state.Layout.Width)
		a.prompt.SetDimensions(a.state.Layout.Width, 5)
		a.keybindHints.SetWidth(a.state.Layout.Width)
	}

	switch msg := msg.(type) {
	case tea.KeyMsg:
		return a.handleKey(msg)

	case spinner.TickMsg:
		var cmd tea.Cmd
		a.spinner, cmd = a.spinner.Update(msg)
		cmds = append(cmds, cmd)

	case StreamMsg:
		a.appendToLastMessage(msg.Content)
		if msg.Done {
			a.state.Processing = false
			a.mode = ModeInput
			a.state.SetStatus("Ready")
		}

	case ResponseMsg:
		if msg.Error != nil {
			a.setError(msg.Error)
		} else {
			a.addMessage(Message{
				Role:      RoleAssistant,
				Content:   msg.Content,
				Timestamp: time.Now(),
			})
		}
		a.state.Processing = false
		a.mode = ModeInput
		a.state.SetStatus("Ready")

	case ToolCallMsg:
		a.addMessage(Message{
			Role:      RoleTool,
			Timestamp: time.Now(),
			ToolCall: &ToolCall{
				Tool:   msg.Tool,
				Input:  msg.Input,
				Result: msg.Result,
				Status: msg.Status,
			},
		})

	case SessionMsg:
		a.handleSessionMsg(msg)

	case ErrorMsg:
		a.setError(msg.Error)

	case LoadMessagesResult:
		if msg.Error != nil {
			a.setError(msg.Error)
		} else {
			a.state.SetMessages(msg.Messages)
			a.messageViewport.SetContent(a.buildMessagesContent())
			a.messageViewport.GotoBottom()
			if len(msg.Messages) > 0 {
				a.state.SetStatus(fmt.Sprintf("Loaded %d messages", len(msg.Messages)))
			}
		}

	case LoadMoreMessagesResult:
		a.state.SetHistoryLoading(a.state.SessionID, false)
		if msg.Error != nil {
			a.setError(msg.Error)
			log.Warn("Failed to load more messages", "error", msg.Error.Error())
		} else if len(msg.Messages) > 0 {
			// Prepend older messages
			a.state.PrependMessages(msg.Messages, msg.Cursor, msg.Complete)
			a.messageViewport.SetContent(a.buildMessagesContent())
			a.state.SetStatus(fmt.Sprintf("Loaded %d more messages", len(msg.Messages)))
			log.Info("Loaded more messages", "count", len(msg.Messages), "cursor", msg.Cursor, "complete", msg.Complete)
		}

	// Dialog close
	case dialog.CloseMsg:
		a.activeDialog = nil
		a.state.PopDialog()

	// Dialog selection
	case dialog.SelectMsg:
		cmd := a.handleDialogSelection(msg)
		if cmd != nil {
			cmds = append(cmds, cmd)
		}

		// Update active dialog
		if a.activeDialog != nil {
			var dialogCmd tea.Cmd
			a.activeDialog, dialogCmd = a.activeDialog.Update(msg)
			if dialogCmd != nil {
				cmds = append(cmds, dialogCmd)
			}
			// Don't process other keys when dialog is open
			return a, tea.Batch(cmds...)
		}

	case TickMsg:
		if a.state.ShowError {
			a.state.ShowError = false
			a.state.LastError = nil
		}

	// Leader timeout - reset leader state
	case LeaderTimeoutMsg:
		// Leader key timeout expired, state is automatically reset

	// Theme change message
	case ThemeChangeMsg:
		a.theme = GetTheme(msg.ThemeID)
		a.styles = ApplyTheme(a.theme)
		a.spinner.Style = lipgloss.NewStyle().Foreground(a.theme.Spinner)
	}

	// Update prompt input if in input mode
	if a.mode == ModeInput {
		_, cmd := a.prompt.Update(msg)
		cmds = append(cmds, cmd)
	}

	// Update viewport
	var cmd tea.Cmd
	a.messageViewport, cmd = a.messageViewport.Update(msg)
	cmds = append(cmds, cmd)

	return a, tea.Batch(cmds...)
}

// ThemeChangeMsg is sent when the theme changes
type ThemeChangeMsg struct {
	ThemeID string
}

// View renders the app (tea.Model interface)
func (a *App) View() string {
	width := a.state.Layout.Width
	height := a.state.Layout.Height

	if width == 0 || height == 0 {
		return "Loading..."
	}

	// Build responsive layout
	var sections []string

	// Title bar
	sections = append(sections, a.renderTitle())

	// Main content area
	contentHeight := height - 6 // Reserve space for title, status, input
	if a.state.ShowError {
		contentHeight -= 3
	}

	// Sidebar (if visible and wide enough)
	if a.state.IsSidebarVisible() && !a.state.Layout.IsResponsive() {
		sections = append(sections, a.renderWithSidebar(contentHeight))
	} else {
		sections = append(sections, a.renderMainContent(contentHeight))
	}

	// Status bar - use layout component
	a.statusBar.SetProcessing(a.state.Processing, a.spinner.View())
	sections = append(sections, a.statusBar.Render(&a.state))

	// Input area - use layout component
	a.prompt.SetProcessing(a.state.Processing, a.spinner.View())
	a.prompt.SetModelInfo(a.state.Local.CurrentAgent, a.state.Local.CurrentModel.ModelID, a.state.Local.ModelVariant)
	sections = append(sections, a.prompt.Render())

	// Error overlay
	if a.state.ShowError && a.state.LastError != nil {
		sections = append(sections, a.renderError())
	}

	// Dialog overlay (if any)
	if a.state.Dialog.HasOpen() {
		sections = append(sections, a.renderDialog())
	}

	// Mobile sidebar overlay (for narrow terminals)
	if a.state.Layout.MobileSidebar.Opened {
		sections = append(sections, a.mobileSidebar.View())
	}

	return lipgloss.JoinVertical(lipgloss.Left, sections...)
}

// renderTitle renders the title bar
func (a *App) renderTitle() string {
	title := "MagiCode"
	if len(a.state.Sync.Sessions) > 0 && a.state.Sync.Sessions[0].Title != "" {
		title = fmt.Sprintf("MagiCode - %s", a.state.Sync.Sessions[0].Title)
	}
	return a.styles.Title.Render(title)
}

// renderWithSidebar renders layout with sidebar
func (a *App) renderWithSidebar(height int) string {
	sidebarWidth := a.state.KV.SidebarWidth
	if sidebarWidth == 0 {
		sidebarWidth = 344 // Default from OpenCode
	}

	// Convert sidebarWidth (pixels) to columns (approximately 8 pixels per column)
	sidebarCols := sidebarWidth / 8
	if sidebarCols < 43 {
		sidebarCols = 43 // Minimum width
	}

	// Calculate content width
	contentWidth := a.state.Layout.Width - sidebarCols

	// Update sidebar dimensions
	a.sidebar.SetDimensions(sidebarCols, height)

	// Render sidebar using layout component
	sidebar := a.sidebar.Render(&a.state)

	// Render main content
	content := a.renderMainContent(height)

	// Join horizontally
	return lipgloss.JoinHorizontal(lipgloss.Top,
		lipgloss.NewStyle().Width(sidebarCols).Render(sidebar),
		lipgloss.NewStyle().Width(contentWidth).Render(content),
	)
}

// renderMainContent renders the main content area
func (a *App) renderMainContent(height int) string {
	switch a.view {
	case ViewChat:
		return a.renderChat(height)
	case ViewSession:
		return a.renderSessionList(height)
	case ViewHelp:
		return a.renderHelp(height)
	default:
		return a.renderChat(height)
	}
}

// renderChat renders the chat view
func (a *App) renderChat(height int) string {
	content := a.buildMessagesContent()
	a.messageViewport.SetContent(content)

	viewportStyle := lipgloss.NewStyle().Height(height)
	return viewportStyle.Render(a.messageViewport.View())
}

// renderSessionList renders the session list
func (a *App) renderSessionList(height int) string {
	var lines []string
	lines = append(lines, a.styles.Bold.Render("Sessions"))

	for _, session := range a.state.Sync.Sessions {
		style := a.styles.Text
		if session.Active {
			style = a.styles.BorderActive
		}
		item := fmt.Sprintf("%s", session.Title)
		lines = append(lines, style.Render(item))
	}

	content := strings.Join(lines, "\n")
	return a.styles.Sidebar.Height(height).Render(content)
}

// renderHelp renders the help overlay
func (a *App) renderHelp(height int) string {
	return a.styles.Border.Height(height).Render(a.helpContent)
}

// renderError renders the error overlay
func (a *App) renderError() string {
	return a.styles.Error.Render(fmt.Sprintf("Error: %v", a.state.LastError))
}

// renderDialog renders the active dialog overlay
func (a *App) renderDialog() string {
	// Use active dialog if available
	if a.activeDialog != nil {
		return a.activeDialog.View()
	}

	// Fallback to DialogState rendering (legacy)
	dialog := a.state.Dialog.Top()
	if dialog == nil {
		return ""
	}

	// Dialog container style
	dialogStyle := lipgloss.NewStyle().
		Foreground(a.theme.Text).
		Background(a.theme.Background).
		Border(lipgloss.RoundedBorder()).
		BorderForeground(a.theme.Primary).
		Padding(1, 2).
		Width(60).
		Height(15)

	var content string
	switch dialog.Type {
	case DialogSessionList:
		content = a.renderDialogSessionList()
	case DialogModelList:
		content = a.renderDialogModelList()
	case DialogHelp:
		content = a.helpContent
	default:
		content = "Dialog: " + string(dialog.Type)
	}

	return dialogStyle.Render(content)
}

// renderDialogSessionList renders session selection dialog content
func (a *App) renderDialogSessionList() string {
	var lines []string
	lines = append(lines, a.styles.Bold.Render("Select Session"))
	lines = append(lines, a.styles.TextMuted.Render("(Esc to close)"))
	lines = append(lines, "")

	for i, session := range a.state.Sync.Sessions {
		style := a.styles.Text
		if i == a.state.Dialog.Top().Selected {
			style = a.styles.BorderActive
		}
		lines = append(lines, style.Render(session.Title))
	}

	return strings.Join(lines, "\n")
}

// renderDialogModelList renders model selection dialog content
func (a *App) renderDialogModelList() string {
	var lines []string
	lines = append(lines, a.styles.Bold.Render("Select Model"))
	lines = append(lines, a.styles.TextMuted.Render("(Esc to close)"))
	lines = append(lines, "")

	// Show recent models
	for _, model := range a.state.Sync.Providers {
		if len(model.Models) > 0 {
			lines = append(lines, a.styles.TextMuted.Render(model.Name))
			for id := range model.Models {
				lines = append(lines, a.styles.Text.Render(id))
			}
		}
	}

	return strings.Join(lines, "\n")
}

// buildMessagesContent builds the messages content for viewport
func (a *App) buildMessagesContent() string {
	var lines []string

	for _, msg := range a.state.Sync.Messages {
		switch msg.Role {
		case RoleUser:
			timeStr := msg.Timestamp.Format("15:04")
			userContent := fmt.Sprintf("[%s] You: %s", timeStr, msg.Content)
			lines = append(lines, a.styles.UserMessage.Render(userContent))

		case RoleAssistant:
			timeStr := msg.Timestamp.Format("15:04")
			if len(msg.Parts) > 0 {
				partLines := a.renderParts(msg.Parts, msg.Model)
				lines = append(lines, partLines)
			} else if msg.Content != "" {
				assistantContent := fmt.Sprintf("[%s] Assistant: %s", timeStr, msg.Content)
				if msg.Model != "" {
					assistantContent = fmt.Sprintf("[%s] Assistant (%s): %s", timeStr, msg.Model, msg.Content)
				}
				lines = append(lines, a.styles.AssistantMessage.Render(assistantContent))
			}

		case RoleSystem:
			lines = append(lines, a.styles.SystemMessage.Render(msg.Content))

		case RoleTool:
			toolLine := a.renderToolCall(msg.ToolCall)
			lines = append(lines, toolLine)
		}
	}

	if len(lines) == 0 {
		return a.styles.TextMuted.Render("No messages. Start a conversation!")
	}

	return strings.Join(lines, "\n\n")
}

// renderParts renders assistant message parts
func (a *App) renderParts(parts []Part, model string) string {
	var lines []string

	header := "Assistant"
	if model != "" {
		header = fmt.Sprintf("Assistant (%s)", model)
	}
	lines = append(lines, a.styles.AssistantMessage.Render(header))

	// Create tool result renderer
	toolRenderer := component.NewToolResultRenderer(a.theme, a.styles)

	for _, part := range parts {
		switch part.Type {
		case "text":
			if part.Text != "" {
				lines = append(lines, a.styles.Text.Render(part.Text))
			}

		case "tool_use":
			// Use the new tool renderer for tool calls
			toolCall := toolRenderer.RenderToolCall(part.ToolName, part.ToolInput)
			lines = append(lines, toolCall)

		case "tool_result":
			// Use the new tool renderer for tool results
			toolResult := toolRenderer.RenderResult(part.ToolName, part.ToolInput, part.ToolResult, part.Status)
			lines = append(lines, toolResult)

		case "thinking":
			if part.Text != "" {
				lines = append(lines, a.styles.Thinking.Render(fmt.Sprintf("💭 %s", part.Text)))
			}
		}
	}

	return strings.Join(lines, "\n")
}

// renderToolCall renders a tool call message
func (a *App) renderToolCall(tc *ToolCall) string {
	if tc == nil {
		return ""
	}

	var style lipgloss.Style
	switch tc.Status {
	case "success":
		style = a.styles.Success
	case "error":
		style = a.styles.Error
	default:
		style = a.styles.Status
	}

	return style.Render(fmt.Sprintf("[%s] %s", tc.Tool, tc.Status))
}

// handleKey handles keyboard input
func (a *App) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	// If active dialog is open, route keys to it
	if a.activeDialog != nil {
		var cmd tea.Cmd
		a.activeDialog, cmd = a.activeDialog.Update(msg)
		return a, cmd
	}

	// Check for quit
	if msg.Type == tea.KeyCtrlC || msg.String() == "q" {
		return a, tea.Quit
	}

	// Handle by view
	switch a.view {
	case ViewChat:
		return a.handleChatKey(msg)
	case ViewSession:
		return a.handleSessionKey(msg)
	case ViewHelp:
		return a.handleHelpKey(msg)
	}

	return a, nil
}

// handleDialogKey handles keyboard input when a dialog is open
func (a *App) handleDialogKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	kb := a.keybindings

	switch {
	case kb.Back.Match(msg) || kb.Cancel.Match(msg):
		a.state.PopDialog()
		return a, nil

	case kb.Up.Match(msg):
		dialog := a.state.Dialog.Top()
		if dialog != nil && dialog.Selected > 0 {
			dialog.Selected--
		}
		return a, nil

	case kb.Down.Match(msg):
		dialog := a.state.Dialog.Top()
		if dialog != nil {
			dialog.Selected++
		}
		return a, nil

	case kb.Select.Match(msg):
		// Handle selection based on dialog type
		dialog := a.state.Dialog.Top()
		if dialog == nil {
			return a, nil
		}
		return a.handleDialogSelect(dialog)
	}

	return a, nil
}

// handleDialogSelection handles selection from a dialog
func (a *App) handleDialogSelection(msg dialog.SelectMsg) tea.Cmd {
	switch msg.Type {
	case DialogSessionList:
		if msg.Data != nil {
			// Check if "new" was selected
			if msg.Data == "new" {
				// Create new session - handled by command
				return nil
			}
			// Otherwise it's a session selection
			if session, ok := msg.Data.(Session); ok {
				a.state.SetActiveSession(&session)
				a.activeDialog = nil
				a.state.PopDialog()
				// Load messages for the selected session
				return a.loadMessagesForSession(session.ID)
			}
		}

	case DialogModelList:
		if msg.Data != nil {
			if modelKey, ok := msg.Data.(ModelKey); ok {
				a.state.SetCurrentModel(modelKey)
				a.activeDialog = nil
				a.state.PopDialog()
			}
		}

	case DialogThemeList:
		if msg.Data != nil {
			if themeID, ok := msg.Data.(string); ok {
				a.state.SetTheme(themeID)
				// Update the app's theme
				a.theme = GetTheme(themeID)
				a.styles = ApplyTheme(a.theme)
				a.spinner.Style = lipgloss.NewStyle().Foreground(a.theme.Spinner)
				a.state.SetStatus(fmt.Sprintf("Theme changed to %s", themeID))
				a.activeDialog = nil
				a.state.PopDialog()
				// Save theme preference to database
				if err := a.saveThemePreference(themeID); err != nil {
					log.Warn("Failed to save theme preference", "error", err, "theme", themeID)
				}
			}
		}

	case DialogCommand:
		if msg.Data != nil {
			if action, ok := msg.Data.(string); ok {
				a.state.SetStatus(fmt.Sprintf("Command: %s", action))
				a.activeDialog = nil
				a.state.PopDialog()
				// Execute the action via leader action handler
				_, cmd := a.handleLeaderAction(&LeaderKeyMsg{Action: action})
				return cmd
			}
		}

	default:
		a.activeDialog = nil
		a.state.PopDialog()
	}
	return nil
}

// handleDialogSelect handles the old DialogState format (for compatibility)
func (a *App) handleDialogSelect(dialog *DialogState) (tea.Model, tea.Cmd) {
	switch dialog.Type {
	case DialogSessionList:
		if dialog.Selected < len(a.state.Sync.Sessions) {
			session := a.state.Sync.Sessions[dialog.Selected]
			a.state.SetActiveSession(&session)
			a.state.PopDialog()
		}

	case DialogModelList:
		// TODO: Implement model selection
		a.state.PopDialog()

	default:
		a.state.PopDialog()
	}

	return a, nil
}

// handleChatKey handles chat view keys
func (a *App) handleChatKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	kb := a.keybindings

	// Check for leader key first (Ctrl+X)
	// This must be checked before any other keybindings
	handled, cmd, leaderMsg := a.leaderHandler.HandleKey(msg)
	if handled {
		// If we got a leader action, handle it
		if leaderMsg != nil {
			return a.handleLeaderAction(leaderMsg)
		}
		// Otherwise, just return (e.g., waiting for second key or timeout)
		return a, cmd
	}

	switch {
	case kb.Submit.Match(msg):
		return a.submitInput()

	case kb.Sessions.Match(msg):
		a.view = ViewSession
		return a, nil

	case kb.Help.Match(msg):
		return a, a.showHelpDialog()

	case kb.NewSession.Match(msg):
		return a, a.createSession()

	case kb.Up.Match(msg):
		a.messageViewport.LineUp(1)
		// Check if at top and need to load more
		if a.atTopOfMessages() && a.state.HistoryMore(a.state.SessionID) && !a.state.HistoryLoading(a.state.SessionID) {
			return a, a.loadMoreMessages()
		}
		return a, nil

	case kb.Down.Match(msg):
		a.messageViewport.LineDown(1)
		return a, nil

	case kb.PageUp.Match(msg):
		a.messageViewport.HalfViewUp()
		// Check if at top and need to load more
		if a.atTopOfMessages() && a.state.HistoryMore(a.state.SessionID) && !a.state.HistoryLoading(a.state.SessionID) {
			return a, a.loadMoreMessages()
		}
		return a, nil

	case kb.PageDown.Match(msg):
		a.messageViewport.HalfViewDown()
		return a, nil

	case kb.HistoryUp.Match(msg):
		return a.navigateHistoryUp(), nil

	case kb.HistoryDown.Match(msg):
		return a.navigateHistoryDown(), nil

	case kb.Cancel.Match(msg):
		if a.state.Processing {
			a.state.Processing = false
			a.mode = ModeInput
			a.state.SetStatus("Cancelled")
		}
		return a, nil

	case kb.CommandPalette.Match(msg):
		return a, a.showCommandPaletteDialog()
	}

	// Pass to prompt's input if in input mode and not a control key
	// This handles regular typing
	if a.mode == ModeInput {
		// Only pass printable characters and essential editing keys to input
		switch msg.Type {
		case tea.KeyRunes, tea.KeySpace, tea.KeyBackspace, tea.KeyDelete, tea.KeyLeft, tea.KeyRight:
			// Update the prompt's internal input by passing the message to it
			_, cmd := a.prompt.Update(msg)
			return a, cmd
		}
		// Also handle keys with runes
		if msg.Runes != nil && len(msg.Runes) > 0 {
			_, cmd := a.prompt.Update(msg)
			return a, cmd
		}
	}

	return a, nil
}

// handleSessionKey handles session view keys
func (a *App) handleSessionKey(msg tea.Msg) (tea.Model, tea.Cmd) {
	kb := a.keybindings

	switch msg := msg.(type) {
	case tea.KeyMsg:
		switch {
		case kb.Back.Match(msg) || kb.Sessions.Match(msg):
			a.view = ViewChat
			return a, nil

		case kb.Up.Match(msg):
			return a, nil

		case kb.Down.Match(msg):
			return a, nil

		case kb.Select.Match(msg):
			return a, nil

		case kb.NewSession.Match(msg):
			return a, a.createSession()
		}
	}

	return a, nil
}

// handleHelpKey handles help view keys
func (a *App) handleHelpKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	kb := a.keybindings

	if kb.Help.Match(msg) || kb.Back.Match(msg) || kb.Quit.Match(msg) {
		a.showHelp = false
		a.view = ViewChat
		return a, nil
	}

	return a, nil
}

// submitInput submits the current input
func (a *App) submitInput() (tea.Model, tea.Cmd) {
	content := strings.TrimSpace(a.prompt.GetValue())
	if content == "" {
		return a, nil
	}

	// Add user message
	a.addMessage(Message{
		Role:      RoleUser,
		Content:   content,
		Timestamp: time.Now(),
	})

	// Add to history
	a.state.Local.InputHistory = append(a.state.Local.InputHistory, content)
	a.state.Local.HistoryIndex = len(a.state.Local.InputHistory)

	// Clear input
	a.prompt.Clear()

	// Set waiting state
	a.state.Processing = true
	a.mode = ModeWait
	a.state.SetStatus("Processing...")

	return a, a.sendMessage(content)
}

// sendMessage sends a message (placeholder for integration)
func (a *App) sendMessage(content string) tea.Cmd {
	return func() tea.Msg {
		// Placeholder - would integrate with provider
		return ResponseMsg{
			Content: "This is a placeholder response. Integration needed.",
		}
	}
}

// createSession creates a new session and persists it to the database
func (a *App) createSession() tea.Cmd {
	return func() tea.Msg {
		log.Info("Creating session", "databasePath", a.databasePath, "workingDirectory", a.workingDirectory)

		// Create session in database
		ctx := context.Background()
		db, err := database.New(ctx, database.Config{Path: a.databasePath})
		if err != nil {
			log.Error("Failed to open database for session creation", "error", err.Error(), "path", a.databasePath)
			return SessionMsg{
				ID:     "",
				Title:  "",
				Action: "error",
				Error:  fmt.Sprintf("Failed to open database: %v", err),
			}
		}
		defer db.Close()

		sessionStorage := database.NewSessionStorage(db)

		// Create the database session
		dbSession := database.Session{
			ID:        uuid.New().String(),
			ProjectID: "default-project", // TODO: Get actual project ID
			Slug:      fmt.Sprintf("session-%d", time.Now().Unix()),
			Directory: a.workingDirectory,
			Title:     "New Session",
			Version:   "1",
		}

		createdSession, err := sessionStorage.Create(ctx, dbSession)
		if err != nil {
			log.Error("Failed to create session", "error", err.Error())
			return SessionMsg{
				ID:     "",
				Title:  "",
				Action: "error",
				Error:  fmt.Sprintf("Failed to create session: %v", err),
			}
		}

		log.Info("Created session in database", "id", createdSession.ID, "directory", createdSession.Directory)

		return SessionMsg{
			ID:        createdSession.ID,
			Title:     createdSession.Title,
			Directory: createdSession.Directory,
			Action:    "create",
		}
	}
}

// handleSessionMsg handles session events
func (a *App) handleSessionMsg(msg SessionMsg) {
	switch msg.Action {
	case "create":
		session := Session{
			ID:        msg.ID,
			Title:     msg.Title,
			Directory: msg.Directory,
			CreatedAt: time.Now(),
			Active:    true,
		}
		a.state.Sync.Sessions = append(a.state.Sync.Sessions, session)
		a.state.SetActiveSession(&session)
		a.state.SetStatus(fmt.Sprintf("Created session: %s", msg.Title))

	case "error":
		a.state.SetStatus(fmt.Sprintf("Session error: %s", msg.Error))

	case "delete":
		for i, s := range a.state.Sync.Sessions {
			if s.ID == msg.ID {
				a.state.Sync.Sessions = append(a.state.Sync.Sessions[:i], a.state.Sync.Sessions[i+1:]...)
				break
			}
		}

	case "switch":
		for i, s := range a.state.Sync.Sessions {
			if s.ID == msg.ID {
				a.state.SetActiveSession(&a.state.Sync.Sessions[i])
				break
			}
		}
	}
}

// loadSessionsFromDB loads sessions from the database and optionally from file-based storage
func (a *App) loadSessionsFromDB() tea.Cmd {
	return func() tea.Msg {
		var allSessions []Session

		// Log the database path being used
		log.Info("Loading sessions from database", "path", a.databasePath)

		// Check if database file exists
		if _, err := os.Stat(a.databasePath); err == nil {
			// Load from SQLite
			ctx := context.Background()
			db, err := database.New(ctx, database.Config{Path: a.databasePath})
			if err != nil {
				log.Error("Failed to open database for loading sessions", "error", err.Error(), "path", a.databasePath)
			} else {
				defer db.Close()

				sessionStorage := database.NewSessionStorage(db)
				dbSessions, err := sessionStorage.ListAll(ctx)
				if err != nil {
					log.Error("Failed to load sessions from database", "error", err.Error())
				} else {
					log.Info("Retrieved sessions from SQLite database", "count", len(dbSessions))

					for _, dbSession := range dbSessions {
						allSessions = append(allSessions, Session{
							ID:        dbSession.ID,
							Title:     dbSession.Title,
							Directory: dbSession.Directory,
							CreatedAt: time.UnixMilli(dbSession.Timestamps.TimeCreated),
							Active:    false,
						})
					}
				}
			}
		} else {
			log.Info("Database file does not exist, skipping SQLite load", "path", a.databasePath)
		}

		// Also check for file-based storage (OpenCode v1.2)
		dataDir := filepath.Dir(a.databasePath)
		log.Info("Checking for file-based storage", "dataDir", dataDir)
		fileStorage := opencode.NewFileStorage(dataDir)
		if fileStorage.HasFileStorage() {
			log.Info("File-based storage detected, loading sessions")
			fileSessions, err := fileStorage.ListSessions()
			if err != nil {
				log.Warn("Failed to load file-based sessions", "error", err.Error())
			} else {
				log.Info("Retrieved sessions from file-based storage", "count", len(fileSessions))

				for _, fs := range fileSessions {
					id, title, directory, createdAt := fs.ToTUISession()
					allSessions = append(allSessions, Session{
						ID:        id,
						Title:     "[Old] " + title,
						Directory: directory,
						CreatedAt: createdAt,
						Active:    false,
					})
				}
			}
		}

		log.Info("Total sessions loaded", "count", len(allSessions))

		// Update state with loaded sessions
		a.state.Sync.Sessions = allSessions
		return nil
	}
}

// loadMessagesForSession loads messages for a specific session from the database
func (a *App) loadMessagesForSession(sessionID string) tea.Cmd {
	return func() tea.Msg {
		if sessionID == "" || a.databasePath == "" {
			return nil
		}

		log.Info("Loading messages for session", "sessionID", sessionID)

		ctx := context.Background()
		db, err := database.New(ctx, database.Config{Path: a.databasePath})
		if err != nil {
			log.Error("Failed to open database for loading messages", "error", err.Error())
			return nil
		}
		defer db.Close()

		messageStorage := database.NewMessageStorage(db)
		messages, _, _, err := messageStorage.ListPaginated(ctx, sessionID, 80, 0)
		if err != nil {
			log.Error("Failed to load messages from database", "error", err.Error())
			return nil
		}

		log.Info("Loaded messages for session", "sessionID", sessionID, "count", len(messages))

		// Convert database messages to UI messages
		uiMessages := make([]Message, 0, len(messages))
		for _, dbMsg := range messages {
			uiMsg := convertDBMessageToTUI(dbMsg)
			uiMessages = append(uiMessages, uiMsg)
		}

		// Update state with loaded messages
		a.state.Sync.Messages = uiMessages
		a.state.SetStatus(fmt.Sprintf("Loaded %d messages", len(uiMessages)))

		return nil
	}
}

// addMessage adds a message to the state
func (a *App) addMessage(msg Message) {
	msg.ID = fmt.Sprintf("msg-%d", time.Now().UnixNano())
	a.state.AddMessage(msg)
	a.messageViewport.GotoBottom()
}

// appendToLastMessage appends to the last assistant message
func (a *App) appendToLastMessage(content string) {
	messages := a.state.Sync.Messages
	if len(messages) > 0 {
		last := &messages[len(messages)-1]
		if last.Role == RoleAssistant {
			last.Content += content
			a.messageViewport.SetContent(a.buildMessagesContent())
		}
	}
}

// setError sets an error
func (a *App) setError(err error) {
	a.state.LastError = err
	a.state.ShowError = true
	a.state.SetStatus(fmt.Sprintf("Error: %v", err))

	// Auto-clear after 3 seconds
	if a.errorTimer != nil {
		a.errorTimer.Stop()
	}
	a.errorTimer = time.AfterFunc(3*time.Second, func() {
		a.state.ShowError = false
	})
}

// navigateHistoryUp navigates input history up
func (a *App) navigateHistoryUp() tea.Model {
	if a.state.Local.HistoryIndex > 0 {
		a.state.Local.HistoryIndex--
		a.prompt.SetValue(a.state.Local.InputHistory[a.state.Local.HistoryIndex])
	}
	return a
}

// navigateHistoryDown navigates input history down
func (a *App) navigateHistoryDown() tea.Model {
	if a.state.Local.HistoryIndex < len(a.state.Local.InputHistory) {
		a.state.Local.HistoryIndex++
		if a.state.Local.HistoryIndex < len(a.state.Local.InputHistory) {
			a.prompt.SetValue(a.state.Local.InputHistory[a.state.Local.HistoryIndex])
		} else {
			a.prompt.Clear()
		}
	}
	return a
}

// updateViewportSize updates viewport dimensions
func (a *App) updateViewportSize() {
	width := a.state.Layout.Width
	height := a.state.Layout.Height

	contentWidth := width - 4
	contentHeight := height - 8

	a.messageViewport.Width = contentWidth
	a.messageViewport.Height = contentHeight
}

// buildHelpContent builds the help content
func buildHelpContent() string {
	return lipgloss.JoinVertical(lipgloss.Left,
		lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("#7C3AED")).Render("Keyboard Shortcuts"),
		"",
		lipgloss.NewStyle().Foreground(lipgloss.Color("#E5E7EB")).Render("Enter       Submit message"),
		lipgloss.NewStyle().Foreground(lipgloss.Color("#E5E7EB")).Render("Ctrl+C/q    Quit"),
		lipgloss.NewStyle().Foreground(lipgloss.Color("#E5E7EB")).Render("Ctrl+S      Sessions list"),
		lipgloss.NewStyle().Foreground(lipgloss.Color("#E5E7EB")).Render("Ctrl+H      Toggle help"),
		lipgloss.NewStyle().Foreground(lipgloss.Color("#E5E7EB")).Render("Ctrl+N      New session"),
		lipgloss.NewStyle().Foreground(lipgloss.Color("#E5E7EB")).Render("↑/↓         Scroll messages"),
		lipgloss.NewStyle().Foreground(lipgloss.Color("#E5E7EB")).Render("PgUp/PgDn   Half-page scroll"),
		lipgloss.NewStyle().Foreground(lipgloss.Color("#E5E7EB")).Render("Ctrl+↑      History up"),
		lipgloss.NewStyle().Foreground(lipgloss.Color("#E5E7EB")).Render("Ctrl+↓      History down"),
		lipgloss.NewStyle().Foreground(lipgloss.Color("#E5E7EB")).Render("Esc         Cancel/back"),
	)
}

// ===========================================
// Accessor Methods (for integration layer)
// ===========================================

// SetTitle sets the app title
func (a *App) SetTitle(title string) {
	if len(a.state.Sync.Sessions) > 0 {
		a.state.Sync.Sessions[0].Title = title
	}
}

// SetStatus sets the status text
func (a *App) SetStatus(status string) {
	a.state.SetStatus(status)
}

// SetSessions sets the session list
func (a *App) SetSessions(sessions []Session) {
	a.state.SetSessions(sessions)
}

// SetMessages sets the message list
func (a *App) SetMessages(messages []Message) {
	a.state.SetMessages(messages)
	a.messageViewport.GotoBottom()
}

// ActiveSession returns the active session
func (a *App) ActiveSession() *Session {
	if len(a.state.Sync.Sessions) == 0 {
		return nil
	}
	// Return first session as active (simplified)
	return &a.state.Sync.Sessions[0]
}

// Messages returns the messages
func (a *App) Messages() []Message {
	return a.state.Sync.Messages
}

// Width returns the terminal width
func (a *App) Width() int {
	return a.state.Layout.Width
}

// Height returns the terminal height
func (a *App) Height() int {
	return a.state.Layout.Height
}

// Processing returns if processing
func (a *App) Processing() bool {
	return a.state.Processing
}

// State returns the app state (for direct access)
func (a *App) State() *AppState {
	return &a.state
}

// Theme returns the current theme
func (a *App) Theme() Theme {
	return a.theme
}

// Styles returns the current styles
func (a *App) Styles() ThemeStyles {
	return a.styles
}

// ===========================================
// Pagination Methods
// ===========================================

// atTopOfMessages returns true if viewport is at the top of messages
func (a *App) atTopOfMessages() bool {
	return a.messageViewport.YOffset <= 1
}

// loadMoreMessages returns a command to load more messages from history
func (a *App) loadMoreMessages() tea.Cmd {
	// Check if we can load more
	if !a.state.HistoryMore(a.state.SessionID) {
		return nil
	}

	// Mark as loading
	a.state.SetHistoryLoading(a.state.SessionID, true)
	a.state.SetStatus("Loading more messages...")

	return func() tea.Msg {
		// Open database
		ctx := context.Background()
		db, err := database.New(ctx, database.Config{Path: a.databasePath})
		if err != nil {
			log.Warn("Failed to open database for loadMore", "error", err.Error())
			return LoadMoreMessagesResult{Error: err}
		}
		defer db.Close()

		// Get cursor from metadata
		meta := a.state.GetMessageMeta(a.state.SessionID)
		cursor := meta.Cursor

		// Load more messages (HistoryMessagePageSize = 200)
		messageStorage := database.NewMessageStorage(db)
		partStorage := database.NewPartStorage(db)

		dbMessages, nextCursor, complete, err := messageStorage.ListPaginated(ctx, a.state.SessionID, HistoryMessagePageSize, cursor)
		if err != nil {
			log.Warn("Failed to load more messages", "error", err.Error())
			return LoadMoreMessagesResult{Error: err}
		}

		// Convert database messages to TUI messages
		var messages []Message
		for _, dbMsg := range dbMessages {
			tuiMsg := convertDBMessageToTUI(dbMsg)

			// Load parts for all messages
			parts, err := partStorage.ListByMessage(ctx, dbMsg.ID)
			if err == nil && len(parts) > 0 {
				tuiMsg.Parts = convertPartsToTUI(parts)
				// For user messages, extract text from parts if available
				if dbMsg.Data.Role == "user" {
					for _, part := range parts {
						if part.Data.Type == "text" && part.Data.Text != "" {
							tuiMsg.Content = part.Data.Text
							break
						}
					}
				}
			}

			messages = append(messages, tuiMsg)
		}

		log.Info("Loaded more messages from database", "count", len(messages), "cursor", nextCursor, "complete", complete)

		return LoadMoreMessagesResult{
			Messages: messages,
			Cursor:   nextCursor,
			Complete: complete,
		}
	}
}

// convertDBMessageToTUI converts a database message to a TUI message
func convertDBMessageToTUI(dbMsg database.Message) Message {
	role := RoleUser
	if dbMsg.Data.Role == "assistant" {
		role = RoleAssistant
	} else if dbMsg.Data.Role == "system" {
		role = RoleSystem
	}

	// Extract timestamp from data.time.created if available
	var timestamp time.Time
	if dbMsg.Data.Time != nil {
		if created, ok := dbMsg.Data.Time["created"]; ok {
			switch v := created.(type) {
			case float64:
				timestamp = time.UnixMilli(int64(v))
			case int64:
				timestamp = time.UnixMilli(v)
			case int:
				timestamp = time.UnixMilli(int64(v))
			default:
				timestamp = time.UnixMilli(dbMsg.Timestamps.TimeCreated)
			}
		} else {
			timestamp = time.UnixMilli(dbMsg.Timestamps.TimeCreated)
		}
	} else {
		timestamp = time.UnixMilli(dbMsg.Timestamps.TimeCreated)
	}

	return Message{
		ID:        dbMsg.ID,
		Role:      role,
		Content:   "",
		Timestamp: timestamp,
		Model:     dbMsg.Data.ModelID,
		Provider:  dbMsg.Data.ProviderID,
	}
}

// convertPartsToTUI converts database parts to TUI parts
// Filters out internal parts (patch, step-start, step-finish)
func convertPartsToTUI(parts []database.Part) []Part {
	skipParts := map[string]bool{
		"patch":       true,
		"step-start":  true,
		"step-finish": true,
	}

	var result []Part
	for _, p := range parts {
		if skipParts[p.Data.Type] {
			continue
		}

		tuiPart := Part{
			ID:     p.ID,
			Type:   p.Data.Type,
			Text:   p.Data.Text,
			Status: p.Data.Status,
		}

		if p.Data.Type == "file" {
			tuiPart.Text = fmt.Sprintf("[File: %s]", p.Data.Filename)
		}

		if p.Data.Type == "tool_use" || p.Data.Type == "tool_result" {
			tuiPart.ToolName = p.Data.ToolName
			tuiPart.ToolID = p.Data.ToolID
			if p.Data.Type == "tool_result" {
				tuiPart.ToolResult = p.Data.ToolResult
			}
		}

		result = append(result, tuiPart)
	}
	return result
}

// ===========================================
// Leader Key Methods
// ===========================================

// handleLeaderAction handles leader key actions
func (a *App) handleLeaderAction(msg *LeaderKeyMsg) (tea.Model, tea.Cmd) {
	switch msg.Action {
	case "session_list":
		return a, a.showSessionListDialog()

	case "session_new":
		return a, a.createSession()

	case "session_export":
		// Show session info for now
		if len(a.state.Sync.Sessions) > 0 {
			session := a.state.Sync.Sessions[0]
			a.state.SetStatus(fmt.Sprintf("Session: %s (%d messages)", session.Title, len(a.state.Sync.Messages)))
		} else {
			a.state.SetStatus("No active session")
		}
		return a, nil

	case "session_compact":
		a.state.SetStatus("Session compact: Feature not yet implemented")
		return a, nil

	case "session_timeline":
		// Show message timeline info
		if len(a.state.Sync.Messages) > 0 {
			a.state.SetStatus(fmt.Sprintf("Timeline: %d messages loaded", len(a.state.Sync.Messages)))
		} else {
			a.state.SetStatus("No messages in timeline")
		}
		return a, nil

	case "sidebar_toggle":
		a.state.ToggleSidebar()
		return a, nil

	case "model_list":
		return a, a.showModelListDialog()

	case "agent_list":
		// Show current agent info in status
		agentInfo := a.state.Local.CurrentAgent
		if agentInfo == "" {
			agentInfo = "default"
		}
		a.state.SetStatus(fmt.Sprintf("Current agent: %s", agentInfo))
		return a, nil

	case "message_copy":
		// Copy last assistant message content
		if len(a.state.Sync.Messages) > 0 {
			lastMsg := a.state.Sync.Messages[len(a.state.Sync.Messages)-1]
			if lastMsg.Role == RoleAssistant && lastMsg.Content != "" {
				a.state.SetStatus(fmt.Sprintf("Copied: %.50s...", lastMsg.Content))
			} else {
				a.state.SetStatus("No assistant message to copy")
			}
		} else {
			a.state.SetStatus("No messages to copy")
		}
		return a, nil

	case "message_undo":
		a.state.SetStatus("Undo: Feature not yet implemented")
		return a, nil

	case "message_redo":
		a.state.SetStatus("Redo: Feature not yet implemented")
		return a, nil

	case "theme_list":
		return a, a.showThemeListDialog()

	case "status_view":
		return a, a.showStatusDialog()

	case "help_dialog":
		return a, a.showHelpDialog()

	case "exit_app":
		return a, tea.Quit

	default:
		return a, nil
	}
}

// ===========================================
// Dialog Methods
// ===========================================

// showSessionListDialog opens the session list dialog
func (a *App) showSessionListDialog() tea.Cmd {
	return tea.Batch(
		// First load sessions from database
		a.loadSessionsFromDB(),
		// Then show the dialog
		func() tea.Msg {
			a.activeDialog = dialog.NewSessionListDialog(a.theme, &a.state)
			a.activeDialog.SetDimensions(a.state.Layout.Width, a.state.Layout.Height)
			a.state.PushDialog(DialogSessionList)
			return nil
		},
	)
}

// showModelListDialog opens the model list dialog
func (a *App) showModelListDialog() tea.Cmd {
	a.activeDialog = dialog.NewModelListDialog(a.theme, &a.state)
	a.activeDialog.SetDimensions(a.state.Layout.Width, a.state.Layout.Height)
	a.state.PushDialog(DialogModelList)
	return a.activeDialog.Init()
}

// showHelpDialog opens the help dialog
func (a *App) showHelpDialog() tea.Cmd {
	a.activeDialog = dialog.NewHelpDialog(a.theme)
	a.activeDialog.SetDimensions(a.state.Layout.Width, a.state.Layout.Height)
	a.state.PushDialog(DialogHelp)
	return a.activeDialog.Init()
}

// closeActiveDialog closes the current dialog
func (a *App) closeActiveDialog() {
	a.activeDialog = nil
	a.state.PopDialog()
}

// showThemeListDialog opens the theme list dialog
func (a *App) showThemeListDialog() tea.Cmd {
	// Get all themes from registry
	registry := NewThemeRegistry()
	themes := registry.ListThemes()

	a.activeDialog = dialog.NewThemeListDialog(a.theme, &a.state, themes)
	a.activeDialog.SetDimensions(a.state.Layout.Width, a.state.Layout.Height)
	a.state.PushDialog(DialogThemeList)
	return a.activeDialog.Init()
}

// showStatusDialog opens the status dialog
func (a *App) showStatusDialog() tea.Cmd {
	// For now, just show status in the status bar
	a.state.SetStatus(fmt.Sprintf("Session: %s | Messages: %d | Theme: %s",
		a.state.SessionID,
		len(a.state.Sync.Messages),
		a.state.KV.Theme))
	return nil
}

// showCommandPaletteDialog opens the command palette dialog
func (a *App) showCommandPaletteDialog() tea.Cmd {
	a.activeDialog = dialog.NewCommandPaletteDialog(a.theme, nil)
	a.activeDialog.SetDimensions(a.state.Layout.Width, a.state.Layout.Height)
	a.state.PushDialog(DialogCommand)
	return a.activeDialog.Init()
}

// saveThemePreference saves the theme preference to the database
func (a *App) saveThemePreference(themeID string) error {
	if a.databasePath == "" {
		return nil // No database to save to
	}

	ctx := context.Background()
	db, err := database.New(ctx, database.Config{Path: a.databasePath})
	if err != nil {
		return fmt.Errorf("failed to open database: %w", err)
	}
	defer db.Close()

	kvStorage := database.NewKVStorage(db)

	if err := kvStorage.SetTheme(ctx, themeID); err != nil {
		return fmt.Errorf("failed to save theme: %w", err)
	}

	log.Info("Theme preference saved", "theme", themeID)
	return nil
}

// loadThemePreference loads the theme preference from the database
func (a *App) loadThemePreference() string {
	if a.databasePath == "" {
		return a.state.KV.Theme // Return current theme if no database
	}

	ctx := context.Background()
	db, err := database.New(ctx, database.Config{Path: a.databasePath})
	if err != nil {
		log.Warn("Failed to open database for theme loading", "error", err)
		return a.state.KV.Theme
	}
	defer db.Close()

	kvStorage := database.NewKVStorage(db)

	theme := kvStorage.GetTheme(ctx)
	log.Info("Theme preference loaded", "theme", theme)
	return theme
}
