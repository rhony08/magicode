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
		config:  config,
		theme:   theme,
		styles:  styles,
		focused: true,
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
		// Handle key events
		switch msg.Type {
		case tea.KeyEnter:
			if !p.processing && p.GetValue() != "" {
				// Submit would be handled by parent app
			}
		case tea.KeyCtrlC:
			// Clear input
			p.SetValue("")
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
