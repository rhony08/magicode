// Package tui provides the main TUI application.
package tui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/spinner"
	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// App is the main TUI application
type App struct {
	// State
	view     ViewState
	mode     InputMode
	width    int
	height   int
	focused  bool

	// Sessions
	sessions    []Session
	activeSession *Session

	// Messages
	messages    []Message
	messageViewport viewport.Model

	// Input
	input       textinput.Model
	inputHistory []string
	historyIndex int

	// Status
	status      string
	spinner     spinner.Model
	processing  bool

	// Error
	lastError   error
	showError   bool
	errorTimer  *time.Timer

	// Help
	showHelp    bool
	helpContent string

	// Keybindings
	keybindings Keybindings
}

// Config represents app configuration
type Config struct {
	Title   string
	Session Session
}

// NewApp creates a new TUI application
func NewApp(cfg Config) *App {
	// Create input
	ti := textinput.New()
	ti.Placeholder = "Type your message..."
	ti.Focus()
	ti.CharLimit = 5000
	ti.Width = 50

	// Create spinner
	s := spinner.New()
	s.Spinner = spinner.Dot
	s.Style = lipgloss.NewStyle().Foreground(colorPrimary)

	// Create message viewport
	vp := viewport.New(80, 20)

	// Default help content
	help := buildHelpContent()

	app := &App{
		view:          ViewChat,
		mode:          ModeInput,
		input:         ti,
		spinner:       s,
		messageViewport: vp,
		status:        "Ready",
		helpContent:   help,
		keybindings:   DefaultKeybindings(),
		inputHistory:  []string{},
	}

	// Add initial session if provided
	if cfg.Session.ID != "" {
		app.sessions = []Session{cfg.Session}
		app.activeSession = &cfg.Session
	}

	return app
}

// Init initializes the app
func (a *App) Init() tea.Cmd {
	return tea.Batch(
		a.spinner.Tick,
		textinput.Blink,
	)
}

// Update handles events
func (a *App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		return a.handleKey(msg)

	case tea.WindowSizeMsg:
		a.width = msg.Width
		a.height = msg.Height
		a.updateViewportSize()
		a.input.Width = a.width - 20

	case spinner.TickMsg:
		var cmd tea.Cmd
		a.spinner, cmd = a.spinner.Update(msg)
		cmds = append(cmds, cmd)

	case StreamMsg:
		a.appendToLastMessage(msg.Content)
		if msg.Done {
			a.processing = false
			a.mode = ModeInput
			a.status = "Ready"
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
		a.processing = false
		a.mode = ModeInput
		a.status = "Ready"

	case ToolCallMsg:
		a.addMessage(Message{
			Role:      RoleTool,
			Timestamp: time.Now(),
			ToolCall:  &ToolCall{
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

	case TickMsg:
		if a.showError {
			a.showError = false
			a.lastError = nil
		}
	}

	// Update input
	if a.mode == ModeInput {
		var cmd tea.Cmd
		a.input, cmd = a.input.Update(msg)
		cmds = append(cmds, cmd)
	}

	// Update viewport
	var cmd tea.Cmd
	a.messageViewport, cmd = a.messageViewport.Update(msg)
	cmds = append(cmds, cmd)

	return a, tea.Batch(cmds...)
}

// View renders the app
func (a *App) View() string {
	if a.width == 0 || a.height == 0 {
		return "Loading..."
	}

	// Build layout
	var sections []string

	// Title bar
	sections = append(sections, a.renderTitle())

	// Main content
	contentHeight := a.height - 6 // Reserve space for title, status, input
	if a.showError {
		contentHeight -= 3
	}

	switch a.view {
	case ViewChat:
		sections = append(sections, a.renderChat(contentHeight))
	case ViewSession:
		sections = append(sections, a.renderSessionList(contentHeight))
	case ViewHelp:
		sections = append(sections, a.renderHelp(contentHeight))
	}

	// Status bar
	sections = append(sections, a.renderStatus())

	// Input
	sections = append(sections, a.renderInput())

	// Error overlay
	if a.showError && a.lastError != nil {
		sections = append(sections, a.renderError())
	}

	return lipgloss.JoinVertical(lipgloss.Left, sections...)
}

// renderTitle renders the title bar
func (a *App) renderTitle() string {
	title := "MagiCode"
	if a.activeSession != nil {
		title = fmt.Sprintf("MagiCode - %s", a.activeSession.Title)
	}
	return styleTitle.Render(title)
}

// renderChat renders the chat view
func (a *App) renderChat(height int) string {
	// Build messages content
	content := a.buildMessagesContent()
	a.messageViewport.SetContent(content)

	// Set viewport height
	viewportStyle := lipgloss.NewStyle().Height(height)
	return viewportStyle.Render(a.messageViewport.View())
}

// renderSessionList renders the session list
func (a *App) renderSessionList(height int) string {
	var lines []string
	lines = append(lines, styleBold.Render("Sessions"))

	for _, session := range a.sessions {
		style := styleSessionItem
		if session.Active {
			style = styleSessionActive
		}
		item := fmt.Sprintf("%s", session.Title)
		lines = append(lines, style.Render(item))
	}

	content := strings.Join(lines, "\n")
	return styleSidebar.Height(height).Render(content)
}

// renderHelp renders the help overlay
func (a *App) renderHelp(height int) string {
	return styleBorder.Height(height).Render(a.helpContent)
}

// renderStatus renders the status bar
func (a *App) renderStatus() string {
	status := a.status
	if a.processing {
		status = a.spinner.View() + " Processing..."
	}
	return styleStatus.Width(a.width).Render(status)
}

// renderInput renders the input field
func (a *App) renderInput() string {
	if a.mode == ModeWait {
		return stylePrompt.Render("Waiting for response...")
	}

	prompt := stylePrompt.Render("> ")
	inputField := a.input.View()
	return lipgloss.NewStyle().Padding(0, 1).Render(prompt + inputField)
}

// renderError renders the error overlay
func (a *App) renderError() string {
	return styleError.Render(fmt.Sprintf("Error: %v", a.lastError))
}

// buildMessagesContent builds the messages content
func (a *App) buildMessagesContent() string {
	var lines []string

	for _, msg := range a.messages {
		switch msg.Role {
		case RoleUser:
			lines = append(lines, styleUserMessage.Render(fmt.Sprintf("You: %s", msg.Content)))
		case RoleAssistant:
			lines = append(lines, styleAssistantMessage.Render(fmt.Sprintf("Assistant: %s", msg.Content)))
		case RoleSystem:
			lines = append(lines, styleSystemMessage.Render(msg.Content))
		case RoleTool:
			toolLine := a.renderToolCall(msg.ToolCall)
			lines = append(lines, toolLine)
		}
	}

	if len(lines) == 0 {
		return styleTextMuted.Render("No messages. Start a conversation!")
	}

	return strings.Join(lines, "\n\n")
}

// renderToolCall renders a tool call message
func (a *App) renderToolCall(tc *ToolCall) string {
	if tc == nil {
		return ""
	}

	var style lipgloss.Style
	switch tc.Status {
	case "success":
		style = styleSuccess
	case "error":
		style = styleError
	default:
		style = styleStatus
	}

	return style.Render(fmt.Sprintf("[%s] %s", tc.Tool, tc.Status))
}

// handleKey handles keyboard input
func (a *App) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
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

// handleChatKey handles chat view keys
func (a *App) handleChatKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	kb := a.keybindings

	switch {
	case kb.Submit.Match(msg):
		return a.submitInput()

	case kb.Sessions.Match(msg):
		a.view = ViewSession
		return a, nil

	case kb.Help.Match(msg):
		a.showHelp = !a.showHelp
		if a.showHelp {
			a.view = ViewHelp
		}
		return a, nil

	case kb.NewSession.Match(msg):
		return a, a.createSession()

	case kb.Up.Match(msg):
		a.messageViewport.LineUp(1)
		return a, nil

	case kb.Down.Match(msg):
		a.messageViewport.LineDown(1)
		return a, nil

	case kb.PageUp.Match(msg):
		a.messageViewport.HalfViewUp()
		return a, nil

	case kb.PageDown.Match(msg):
		a.messageViewport.HalfViewDown()
		return a, nil

	case kb.HistoryUp.Match(msg):
		return a.navigateHistoryUp(), nil

	case kb.HistoryDown.Match(msg):
		return a.navigateHistoryDown(), nil

	case kb.Cancel.Match(msg):
		if a.processing {
			a.processing = false
			a.mode = ModeInput
			a.status = "Cancelled"
		}
		return a, nil
	}

	// Pass to input if in input mode
	if a.mode == ModeInput {
		var cmd tea.Cmd
		a.input, cmd = a.input.Update(msg)
		return a, cmd
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
			// Navigate up in session list
			return a, nil

		case kb.Down.Match(msg):
			// Navigate down
			return a, nil

		case kb.Select.Match(msg):
			// Select session
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
	content := strings.TrimSpace(a.input.Value())
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
	a.inputHistory = append(a.inputHistory, content)
	a.historyIndex = len(a.inputHistory)

	// Clear input
	a.input.Reset()

	// Set waiting state
	a.processing = true
	a.mode = ModeWait
	a.status = "Processing..."

	// Return command to send message (would be implemented by integration layer)
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

// createSession creates a new session (placeholder)
func (a *App) createSession() tea.Cmd {
	return func() tea.Msg {
		session := Session{
			ID:        fmt.Sprintf("session-%d", time.Now().Unix()),
			Title:     "New Session",
			CreatedAt: time.Now(),
			Active:    true,
		}
		return SessionMsg{
			ID:     session.ID,
			Title:  session.Title,
			Action: "create",
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
			CreatedAt: time.Now(),
			Active:    true,
		}
		a.sessions = append(a.sessions, session)
		a.activeSession = &session

	case "delete":
		for i, s := range a.sessions {
			if s.ID == msg.ID {
				a.sessions = append(a.sessions[:i], a.sessions[i+1:]...)
				break
			}
		}

	case "switch":
		for i, s := range a.sessions {
			if s.ID == msg.ID {
				a.activeSession = &a.sessions[i]
				break
			}
		}
	}
}

// addMessage adds a message
func (a *App) addMessage(msg Message) {
	msg.ID = fmt.Sprintf("msg-%d", time.Now().UnixNano())
	a.messages = append(a.messages, msg)
	a.messageViewport.GotoBottom()
}

// appendToLastMessage appends to the last assistant message
func (a *App) appendToLastMessage(content string) {
	if len(a.messages) > 0 {
		last := &a.messages[len(a.messages)-1]
		if last.Role == RoleAssistant {
			last.Content += content
			a.messageViewport.SetContent(a.buildMessagesContent())
		}
	}
}

// setError sets an error
func (a *App) setError(err error) {
	a.lastError = err
	a.showError = true
	a.status = fmt.Sprintf("Error: %v", err)

	// Auto-clear after 3 seconds
	if a.errorTimer != nil {
		a.errorTimer.Stop()
	}
	a.errorTimer = time.AfterFunc(3*time.Second, func() {
		a.showError = false
	})
}

// navigateHistoryUp navigates input history up
func (a App) navigateHistoryUp() tea.Model {
	if a.historyIndex > 0 {
		a.historyIndex--
		a.input.SetValue(a.inputHistory[a.historyIndex])
	}
	return &a
}

// navigateHistoryDown navigates input history down
func (a App) navigateHistoryDown() tea.Model {
	if a.historyIndex < len(a.inputHistory) {
		a.historyIndex++
		if a.historyIndex < len(a.inputHistory) {
			a.input.SetValue(a.inputHistory[a.historyIndex])
		} else {
			a.input.Reset()
		}
	}
	return &a
}

// updateViewportSize updates viewport dimensions
func (a *App) updateViewportSize() {
	contentWidth := a.width - 4
	contentHeight := a.height - 8
	a.messageViewport.Width = contentWidth
	a.messageViewport.Height = contentHeight
}

// SetTitle sets the app title
func (a *App) SetTitle(title string) {
	if a.activeSession != nil {
		a.activeSession.Title = title
	}
}

// SetStatus sets the status text
func (a *App) SetStatus(status string) {
	a.status = status
}

// SetSessions sets the session list
func (a *App) SetSessions(sessions []Session) {
	a.sessions = sessions
}

// SetMessages sets the message list
func (a *App) SetMessages(messages []Message) {
	a.messages = messages
	a.messageViewport.GotoBottom()
}

// ActiveSession returns the active session
func (a *App) ActiveSession() *Session {
	return a.activeSession
}

// Messages returns the messages
func (a *App) Messages() []Message {
	return a.messages
}

// Width returns the terminal width
func (a *App) Width() int {
	return a.width
}

// Height returns the terminal height
func (a *App) Height() int {
	return a.height
}

// Processing returns if processing
func (a *App) Processing() bool {
	return a.processing
}

// buildHelpContent builds the help content
func buildHelpContent() string {
	return lipgloss.JoinVertical(lipgloss.Left,
		styleBold.Render("Keyboard Shortcuts"),
		"",
		styleText.Render("Enter       Submit message"),
		styleText.Render("Ctrl+C/q    Quit"),
		styleText.Render("Ctrl+S      Sessions list"),
		styleText.Render("Ctrl+H      Toggle help"),
		styleText.Render("Ctrl+N      New session"),
		styleText.Render("↑/↓         Scroll messages"),
		styleText.Render("PgUp/PgDn   Half-page scroll"),
		styleText.Render("Ctrl+↑      History up"),
		styleText.Render("Ctrl+↓      History down"),
		styleText.Render("Esc         Cancel/back"),
	)
}