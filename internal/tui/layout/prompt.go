// Package layout provides layout components for the TUI.
// Prompt implements the input area matching OpenCode's design.
package layout

import (
	"strings"

	"github.com/charmbracelet/bubbles/textarea"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/rhony08/magicode/internal/tui/types"
)

// PromptConfig holds configuration for the prompt area
type PromptConfig struct {
	Placeholder   string // Placeholder text
	CharLimit     int    // Character limit
	ShowModelInfo bool   // Show agent/model info
	ShowKeybinds  bool   // Show keybind hints
	Multiline     bool   // Enable multiline input
}

// DefaultPromptConfig returns the default prompt configuration
func DefaultPromptConfig() PromptConfig {
	return PromptConfig{
		Placeholder:   "Type your message...",
		CharLimit:     5000,
		ShowModelInfo: true,
		ShowKeybinds:  true,
		Multiline:     true,
	}
}

// Prompt is the input area component
type Prompt struct {
	config PromptConfig
	theme  types.Theme
	styles PromptStyles
	width  int
	height int

	// Input components
	textInput textinput.Model
	textArea  textarea.Model

	// State
	focused    bool
	processing bool
	spinner    string

	// Model/Agent info
	agent   string
	model   string
	variant string

	// Autocomplete
	autocomplete *Autocomplete
}

// PromptStyles holds lipgloss styles for prompt
type PromptStyles struct {
	Container    lipgloss.Style
	Input        lipgloss.Style
	Placeholder  lipgloss.Style
	Border       lipgloss.Style
	BorderActive lipgloss.Style
	InfoBar      lipgloss.Style
	KeybindBar   lipgloss.Style
	Text         lipgloss.Style
	TextMuted    lipgloss.Style
	Spinner      lipgloss.Style
	Processing   lipgloss.Style
}

// NewPrompt creates a new prompt component
func NewPrompt(config PromptConfig, theme types.Theme) *Prompt {
	styles := PromptStyles{
		Container: lipgloss.NewStyle().
			Border(lipgloss.NormalBorder()).
			BorderForeground(theme.Border).
			Background(theme.PanelBg).
			Padding(0, 1),
		Input: lipgloss.NewStyle().
			Foreground(theme.Text),
		Placeholder: lipgloss.NewStyle().
			Foreground(theme.TextMuted),
		Border: lipgloss.NewStyle().
			Border(lipgloss.NormalBorder()).
			BorderForeground(theme.Border),
		BorderActive: lipgloss.NewStyle().
			Border(lipgloss.NormalBorder()).
			BorderForeground(theme.Primary),
		InfoBar: lipgloss.NewStyle().
			Foreground(theme.TextMuted).
			Padding(0, 1),
		KeybindBar: lipgloss.NewStyle().
			Foreground(theme.TextMuted).
			Padding(0, 1),
		Text: lipgloss.NewStyle().
			Foreground(theme.Text),
		TextMuted: lipgloss.NewStyle().
			Foreground(theme.TextMuted),
		Spinner: lipgloss.NewStyle().
			Foreground(theme.Spinner),
		Processing: lipgloss.NewStyle().
			Foreground(theme.Warning),
	}

	prompt := &Prompt{
		config:       config,
		theme:        theme,
		styles:       styles,
		focused:      true,
		autocomplete: NewAutocomplete(theme),
	}

	// Create textarea for multiline input
	if config.Multiline {
		ta := textarea.New()
		ta.SetWidth(80)
		ta.SetHeight(3)
		ta.Placeholder = config.Placeholder
		ta.CharLimit = config.CharLimit
		ta.Focus()
		prompt.textArea = ta
	} else {
		// Create textinput for single line
		ti := textinput.New()
		ti.Placeholder = config.Placeholder
		ti.Focus()
		ti.CharLimit = config.CharLimit
		ti.Width = 80
		prompt.textInput = ti
	}

	return prompt
}

// Init initializes the prompt (tea.Model interface)
func (p *Prompt) Init() tea.Cmd {
	return textarea.Blink
}

// Update handles events (tea.Model interface)
func (p *Prompt) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.KeyMsg:
		// Handle autocomplete first if visible
		if p.autocomplete.IsVisible() {
			switch msg.Type {
			case tea.KeyTab:
				// Select current option
				if option, ok := p.autocomplete.Select(); ok {
					p.InsertCompletion(option.Value)
					p.autocomplete.Hide()
				}
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
			case tea.KeyEnter:
				// Also select on Enter
				if option, ok := p.autocomplete.Select(); ok {
					p.InsertCompletion(option.Value)
					p.autocomplete.Hide()
				}
				return p, nil
			}
		}

		// Handle normal key events
		switch msg.Type {
		case tea.KeyEnter:
			if !p.processing && p.GetValue() != "" {
				// Submit would be handled by parent app
			}
		case tea.KeyCtrlC:
			// Clear input
			p.SetValue("")
		case tea.KeyTab:
			// Check if we should trigger autocomplete
			p.checkAutocomplete()
			return p, nil
		}

	case tea.WindowSizeMsg:
		p.SetDimensions(msg.Width, msg.Height)
	}

	// Update textarea/textinput
	if p.config.Multiline {
		p.textArea, cmd = p.textArea.Update(msg)
	} else {
		p.textInput, cmd = p.textInput.Update(msg)
	}

	return p, cmd
}

// View renders the prompt (tea.Model interface)
func (p *Prompt) View() string {
	return p.Render()
}

// Render renders the prompt area
func (p *Prompt) Render() string {
	if p.width == 0 {
		return ""
	}

	var sections []string

	// ===========================================
	// Section 1: Input area
	// ===========================================
	var inputContent string
	if p.processing {
		inputContent = p.styles.Processing.Render("Waiting for response...")
	} else {
		if p.config.Multiline {
			p.textArea.SetWidth(p.width - 4)
			inputContent = p.textArea.View()
		} else {
			p.textInput.Width = p.width - 4
			inputContent = p.textInput.View()
		}
	}

	// Apply border style (active when focused)
	borderStyle := p.styles.Border
	if p.focused {
		borderStyle = p.styles.BorderActive
	}

	inputBox := borderStyle.
		Width(p.width).
		Render(inputContent)

	sections = append(sections, inputBox)

	// ===========================================
	// Section 2: Info bar (agent/model/variant)
	// ===========================================
	if p.config.ShowModelInfo {
		infoParts := []string{}

		if p.agent != "" {
			infoParts = append(infoParts, p.styles.Text.Render(p.agent))
		}
		if p.model != "" {
			infoParts = append(infoParts, p.styles.Text.Render(p.model))
		}
		if p.variant != "" {
			infoParts = append(infoParts, p.styles.TextMuted.Render("[")+
				p.styles.Text.Render(p.variant)+
				p.styles.TextMuted.Render("]"))
		}

		if len(infoParts) > 0 {
			infoLine := strings.Join(infoParts, p.styles.TextMuted.Render(" · "))
			sections = append(sections, p.styles.InfoBar.Render(infoLine))
		}
	}

	// ===========================================
	// Section 3: Keybind hints / Processing indicator
	// ===========================================
	if p.processing {
		spinnerLine := p.styles.Spinner.Render(p.spinner) +
			p.styles.TextMuted.Render(" processing...")
		sections = append(sections, p.styles.KeybindBar.Render(spinnerLine))
	} else if p.config.ShowKeybinds {
		hints := "Enter submit  ·  Ctrl+P palette  ·  Tab cycle"
		sections = append(sections, p.styles.KeybindBar.Render(hints))
	}

	// ===========================================
	// Section 4: Autocomplete dropdown
	// ===========================================
	if p.autocomplete.IsVisible() {
		autocompleteView := p.autocomplete.Render(p.width)
		sections = append(sections, autocompleteView)
	}

	return lipgloss.JoinVertical(lipgloss.Left, sections...)
}

// SetDimensions updates prompt dimensions
func (p *Prompt) SetDimensions(width, height int) {
	p.width = width
	p.height = height

	if p.config.Multiline {
		p.textArea.SetWidth(width - 4)
	} else {
		p.textInput.Width = width - 4
	}
}

// SetValue sets the input value
func (p *Prompt) SetValue(value string) {
	if p.config.Multiline {
		p.textArea.SetValue(value)
	} else {
		p.textInput.SetValue(value)
	}
}

// GetValue returns the input value
func (p *Prompt) GetValue() string {
	if p.config.Multiline {
		return p.textArea.Value()
	}
	return p.textInput.Value()
}

// Clear clears the input
func (p *Prompt) Clear() {
	p.SetValue("")
}

// Focus focuses the input
func (p *Prompt) Focus() tea.Cmd {
	p.focused = true
	if p.config.Multiline {
		return p.textArea.Focus()
	}
	return nil
}

// Blur unfocuses the input
func (p *Prompt) Blur() {
	p.focused = false
	if p.config.Multiline {
		p.textArea.Blur()
	}
}

// SetProcessing sets the processing state
func (p *Prompt) SetProcessing(processing bool, spinner string) {
	p.processing = processing
	p.spinner = spinner
}

// SetModelInfo sets the agent/model/variant info
func (p *Prompt) SetModelInfo(agent, model, variant string) {
	p.agent = agent
	p.model = model
	p.variant = variant
}

// InsertCompletion inserts an autocomplete completion at cursor position
func (p *Prompt) InsertCompletion(value string) {
	currentValue := p.GetValue()
	cursorPos := p.GetCursorPos()

	// Find the word at cursor position
	start, end := p.getWordBounds(currentValue, cursorPos)

	// Replace the word with the completion
	newValue := currentValue[:start] + value + currentValue[end:]
	p.SetValue(newValue)

	// Move cursor to end of inserted text
	newCursorPos := start + len(value)
	p.SetCursorPos(newCursorPos)
}

// checkAutocomplete checks if autocomplete should be triggered
func (p *Prompt) checkAutocomplete() {
	value := p.GetValue()
	cursorPos := p.GetCursorPos()

	// Get word at cursor
	word := p.getWordAtCursor(value, cursorPos)

	// Check for trigger characters
	if strings.HasPrefix(word, "@") {
		query := word[1:] // Remove @
		p.ShowAgentCompletions(query)
	} else if strings.HasPrefix(word, "/") && strings.Count(value[:cursorPos], "\n") == 0 {
		// Only trigger / commands at start of line
		query := word[1:] // Remove /
		p.ShowCommandCompletions(query)
	}
}

// getWordAtCursor extracts the word at cursor position
func (p *Prompt) getWordAtCursor(text string, pos int) string {
	if pos > len(text) {
		pos = len(text)
	}

	// Find word boundaries
	start := pos
	for start > 0 && !isWordSeparator(text[start-1]) {
		start--
	}

	end := pos
	for end < len(text) && !isWordSeparator(text[end]) {
		end++
	}

	return text[start:end]
}

// getWordBounds returns the start and end positions of the word at cursor
func (p *Prompt) getWordBounds(text string, pos int) (int, int) {
	if pos > len(text) {
		pos = len(text)
	}

	start := pos
	for start > 0 && !isWordSeparator(text[start-1]) {
		start--
	}

	end := pos
	for end < len(text) && !isWordSeparator(text[end]) {
		end++
	}

	return start, end
}

// isWordSeparator checks if a character is a word separator
func isWordSeparator(c byte) bool {
	return c == ' ' || c == '\t' || c == '\n' || c == '\r'
}

// GetCursorPos returns the current cursor position
func (p *Prompt) GetCursorPos() int {
	if p.config.Multiline {
		// textarea doesn't expose cursor position directly
		// For now, return end of value
		return len(p.textArea.Value())
	}
	// For textinput, we need to access the position differently
	return len(p.textInput.Value())
}

// SetCursorPos sets the cursor position
func (p *Prompt) SetCursorPos(pos int) {
	// Both textarea and textinput don't support setting cursor position directly
	// This is a limitation of the bubbles library
}

// ShowAgentCompletions shows agent/file completions
func (p *Prompt) ShowAgentCompletions(query string) {
	// This will be populated by the app with actual agents
	// For now, just show a placeholder
	options := []AutocompleteOption{
		{Value: "@default", Display: "default", Description: "Default agent", Icon: "🤖"},
		{Value: "@code", Display: "code", Description: "Code review agent", Icon: "💻"},
	}
	p.autocomplete.Show(options, "@", query)
}

// ShowCommandCompletions shows slash command completions
func (p *Prompt) ShowCommandCompletions(query string) {
	options := []AutocompleteOption{
		{Value: "/status", Display: "status", Description: "Show system status", Icon: "📊"},
		{Value: "/compact", Display: "compact", Description: "Compact session history", Icon: "🗜"},
		{Value: "/export", Display: "export", Description: "Export session", Icon: "📤"},
		{Value: "/help", Display: "help", Description: "Show help", Icon: "❓"},
	}

	// Filter by query
	if query != "" {
		var filtered []AutocompleteOption
		for _, opt := range options {
			if strings.HasPrefix(opt.Display, query) {
				filtered = append(filtered, opt)
			}
		}
		options = filtered
	}

	p.autocomplete.Show(options, "/", query)
}

// HideAutocomplete hides the autocomplete dropdown
func (p *Prompt) HideAutocomplete() {
	p.autocomplete.Hide()
}

// IsAutocompleteVisible returns true if autocomplete is visible
func (p *Prompt) IsAutocompleteVisible() bool {
	return p.autocomplete.IsVisible()
}

// GetAutocomplete returns the autocomplete component
func (p *Prompt) GetAutocomplete() *Autocomplete {
	return p.autocomplete
}

// SetAutocompleteOptions sets the autocomplete options for agents
func (p *Prompt) SetAutocompleteOptions(options []AutocompleteOption) {
	p.autocomplete.SetOptions(options)
}

// Width returns the prompt width
func (p *Prompt) Width() int {
	return p.width
}

// Height returns the prompt height
func (p *Prompt) Height() int {
	return p.height
}

// IsFocused returns whether the prompt is focused
func (p *Prompt) IsFocused() bool {
	return p.focused
}

// IsProcessing returns whether the prompt is processing
func (p *Prompt) IsProcessing() bool {
	return p.processing
}

// ===========================================
// Prompt with Attachments
// ===========================================

// PromptWithAttachments extends Prompt with file attachment support
type PromptWithAttachments struct {
	Prompt
	attachments []types.InputPart
	styles      AttachmentStyles
}

// AttachmentStyles holds styles for attachments
type AttachmentStyles struct {
	Container  lipgloss.Style
	File       lipgloss.Style
	FileIcon   lipgloss.Style
	RemoveIcon lipgloss.Style
}

// NewPromptWithAttachments creates a prompt with attachment support
func NewPromptWithAttachments(config PromptConfig, theme types.Theme) *PromptWithAttachments {
	attachmentStyles := AttachmentStyles{
		Container: lipgloss.NewStyle().
			Foreground(theme.Text).
			Background(theme.ElementBg).
			Padding(0, 1),
		File: lipgloss.NewStyle().
			Foreground(theme.Info),
		FileIcon: lipgloss.NewStyle().
			Foreground(theme.Info),
		RemoveIcon: lipgloss.NewStyle().
			Foreground(theme.Error),
	}

	return &PromptWithAttachments{
		Prompt:      *NewPrompt(config, theme),
		styles:      attachmentStyles,
		attachments: []types.InputPart{},
	}
}

// AddAttachment adds a file attachment
func (p *PromptWithAttachments) AddAttachment(path string) {
	p.attachments = append(p.attachments, types.InputPart{
		Type:     "file",
		FilePath: path,
	})
}

// RemoveAttachment removes an attachment by index
func (p *PromptWithAttachments) RemoveAttachment(index int) {
	if index >= 0 && index < len(p.attachments) {
		p.attachments = append(p.attachments[:index], p.attachments[index+1:]...)
	}
}

// ClearAttachments clears all attachments
func (p *PromptWithAttachments) ClearAttachments() {
	p.attachments = []types.InputPart{}
}

// GetAttachments returns the attachments
func (p *PromptWithAttachments) GetAttachments() []types.InputPart {
	return p.attachments
}

// Render renders the prompt with attachments
func (p *PromptWithAttachments) Render() string {
	base := p.Prompt.Render()

	// Add attachments section if any
	if len(p.attachments) > 0 {
		var attachLines []string
		for _, att := range p.attachments {
			if att.Type == "file" {
				icon := p.styles.FileIcon.Render("📄")
				name := att.FileName
				if name == "" {
					name = att.FilePath
					// Truncate long paths
					if len(name) > 30 {
						name = "..." + name[len(name)-25:]
					}
				}
				attachLines = append(attachLines,
					p.styles.Container.Render(icon+" "+name))
			}
		}

		attachSection := strings.Join(attachLines, " ")
		return lipgloss.JoinVertical(lipgloss.Left, attachSection, base)
	}

	return base
}
