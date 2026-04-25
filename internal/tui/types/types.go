// Package types provides shared types for the TUI and layout packages.
// This package breaks the import cycle between tui and layout.
package types

import (
	"fmt"
	"time"

	"github.com/charmbracelet/lipgloss"
)

// ===========================================
// Theme Types
// ===========================================

// Theme defines a color theme for the TUI
type Theme struct {
	// Identification
	Name   string `json:"name"`
	ID     string `json:"id"`
	IsDark bool   `json:"is_dark"`

	// Core colors
	Primary   lipgloss.Color `json:"primary"`
	Secondary lipgloss.Color `json:"secondary"`
	Accent    lipgloss.Color `json:"accent"`

	// Status colors
	Error   lipgloss.Color `json:"error"`
	Warning lipgloss.Color `json:"warning"`
	Success lipgloss.Color `json:"success"`
	Info    lipgloss.Color `json:"info"`

	// Text colors
	Text      lipgloss.Color `json:"text"`
	TextMuted lipgloss.Color `json:"text_muted"`
	TextBold  lipgloss.Color `json:"text_bold"`

	// Background colors
	Background lipgloss.Color `json:"background"`
	PanelBg    lipgloss.Color `json:"panel_bg"`
	ElementBg  lipgloss.Color `json:"element_bg"`
	MenuBg     lipgloss.Color `json:"menu_bg"`

	// Border colors
	Border       lipgloss.Color `json:"border"`
	BorderActive lipgloss.Color `json:"border_active"`

	// Diff colors
	Added     lipgloss.Color `json:"added"`
	Removed   lipgloss.Color `json:"removed"`
	AddedBg   lipgloss.Color `json:"added_bg"`
	RemovedBg lipgloss.Color `json:"removed_bg"`

	// Markdown colors
	Heading    lipgloss.Color `json:"heading"`
	Link       lipgloss.Color `json:"link"`
	Code       lipgloss.Color `json:"code"`
	CodeBg     lipgloss.Color `json:"code_bg"`
	BlockQuote lipgloss.Color `json:"block_quote"`

	// Syntax highlighting colors
	Comment  lipgloss.Color `json:"comment"`
	Keyword  lipgloss.Color `json:"keyword"`
	Function lipgloss.Color `json:"function"`
	String   lipgloss.Color `json:"string"`
	Number   lipgloss.Color `json:"number"`

	// Special colors
	Spinner  lipgloss.Color `json:"spinner"`
	Progress lipgloss.Color `json:"progress"`
}

// ThemeStyles holds pre-computed styles for a theme
type ThemeStyles struct {
	Title            lipgloss.Style
	Prompt           lipgloss.Style
	UserMessage      lipgloss.Style
	AssistantMessage lipgloss.Style
	SystemMessage    lipgloss.Style
	Error            lipgloss.Style
	Success          lipgloss.Style
	Warning          lipgloss.Style
	Info             lipgloss.Style
	Status           lipgloss.Style
	Text             lipgloss.Style
	TextMuted        lipgloss.Style
	TextAccent       lipgloss.Style
	Bold             lipgloss.Style
	Border           lipgloss.Style
	BorderActive     lipgloss.Style
	Sidebar          lipgloss.Style
	ToolUse          lipgloss.Style
	ToolResult       lipgloss.Style
	Thinking         lipgloss.Style
	Heading          lipgloss.Style
	Code             lipgloss.Style
	Link             lipgloss.Style
	Added            lipgloss.Style
	Removed          lipgloss.Style
	Spinner          lipgloss.Style
}

// ===========================================
// Session Types
// ===========================================

// Session represents a chat session
type Session struct {
	ID        string    `json:"id"`
	Title     string    `json:"title"`
	Directory string    `json:"directory"`
	CreatedAt time.Time `json:"created_at"`
	Active    bool      `json:"active"`
}

// ===========================================
// LSP/MCP Types
// ===========================================

// LSPServer represents an LSP server status
type LSPServer struct {
	Name     string `json:"name"`
	Language string `json:"language"`
	Status   string `json:"status"` // "running", "stopped", "error"
}

// MCPServer represents an MCP server status
type MCPServer struct {
	Name   string `json:"name"`
	Status string `json:"status"` // "running", "stopped", "error"
}

// ===========================================
// Input Types
// ===========================================

// InputPart represents an input attachment
type InputPart struct {
	Type     string `json:"type"` // "text", "file"
	Text     string `json:"text,omitempty"`
	FilePath string `json:"file_path,omitempty"`
	FileName string `json:"file_name,omitempty"`
}

// ===========================================
// Provider/Agent/Model Types
// ===========================================

// Provider represents an AI provider
type Provider struct {
	ID        string           `json:"id"`
	Name      string           `json:"name"`
	Connected bool             `json:"connected"`
	Models    map[string]Model `json:"models"`
}

// Model represents an AI model
type Model struct {
	ID         string            `json:"id"`
	Name       string            `json:"name"`
	ProviderID string            `json:"provider_id"`
	Variants   map[string]string `json:"variants,omitempty"`
}

// Agent represents an agent type
type Agent struct {
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Model       *ModelKey `json:"model,omitempty"`
	Variant     string    `json:"variant,omitempty"`
	Mode        string    `json:"mode"` // "build", etc.
	Hidden      bool      `json:"hidden"`
}

// ModelKey identifies a model by provider and ID
type ModelKey struct {
	ProviderID string `json:"provider_id"`
	ModelID    string `json:"model_id"`
	Variant    string `json:"variant,omitempty"`
}

// ===========================================
// Route Types
// ===========================================

// Route represents the current navigation state
type Route string

const (
	RouteHome    Route = "home"
	RouteSession Route = "session"
)

// ===========================================
// Sidebar Mode
// ===========================================

// SidebarMode controls sidebar visibility behavior
type SidebarMode string

const (
	SidebarModeAuto SidebarMode = "auto" // Auto-hide on narrow terminals
	SidebarModeShow SidebarMode = "show" // Always visible
	SidebarModeHide SidebarMode = "hide" // Always hidden
)

// ===========================================
// Dialog Types
// ===========================================

// DialogType identifies the type of dialog
type DialogType string

const (
	DialogSessionList DialogType = "session_list"
	DialogModelList   DialogType = "model_list"
	DialogAgentList   DialogType = "agent_list"
	DialogThemeList   DialogType = "theme_list"
	DialogHelp        DialogType = "help"
	DialogCommand     DialogType = "command"
	DialogStatus      DialogType = "status"
	DialogPermission  DialogType = "permission"
	DialogQuestion    DialogType = "question"
	DialogConfirm     DialogType = "confirm"
)

// DialogState holds state for an active dialog
type DialogState struct {
	Type     DialogType `json:"type"`
	Open     bool       `json:"open"`
	Selected int        `json:"selected"` // Selected item index
	Search   string     `json:"search"`   // Search/filter text
	Data     any        `json:"data"`     // Dialog-specific data
}

// ===========================================
// Store Types
// ===========================================

// KVStore holds persistent user preferences stored in localStorage/database
type KVStore struct {
	// Theme settings
	Theme     string `json:"theme"`      // Current theme name
	ThemeDark bool   `json:"theme_dark"` // Dark mode preference

	// Sidebar settings
	SidebarMode       SidebarMode `json:"sidebar_mode"`       // auto, show, hide
	SidebarWidth      int         `json:"sidebar_width"`      // Default: 344 (OpenCode standard)
	SidebarWorkspaces bool        `json:"sidebar_workspaces"` // Show workspace section

	// Timestamp settings
	ShowTimestamps bool `json:"show_timestamps"` // Show message timestamps

	// Review settings
	ReviewDiffStyle string `json:"review_diff_style"` // "unified" or "split"

	// Terminal settings
	TerminalHeight int  `json:"terminal_height"` // Terminal panel height
	TerminalOpened bool `json:"terminal_opened"` // Terminal panel visibility

	// File tree settings
	FileTreeOpened bool   `json:"file_tree_opened"` // File tree visibility
	FileTreeWidth  int    `json:"file_tree_width"`  // File tree width
	FileTreeTab    string `json:"file_tree_tab"`    // "changes" or "all"

	// Session settings
	SessionWidth int `json:"session_width"` // Session content width
}

// SyncStore holds data synced from server/database
type SyncStore struct {
	// Sessions
	Sessions     []Session `json:"sessions"`
	SessionTotal int       `json:"session_total"` // Total count (may have more)
	SessionLimit int       `json:"session_limit"` // Current limit for pagination

	// Current session messages
	// Note: Message type is defined in tui package due to Part dependency
	Messages     []Message `json:"messages"`
	MessageTotal int       `json:"message_total"`

	// Providers
	Providers []Provider `json:"providers"`

	// Agents
	Agents []Agent `json:"agents"`

	// Project info
	ProjectID   string `json:"project_id"`
	ProjectName string `json:"project_name"`

	// LSP/MCP status
	LSPServers []LSPServer `json:"lsp_servers"`
	LSPReady   bool        `json:"lsp_ready"`
	MCPServers []MCPServer `json:"mcp_servers"`

	// Status
	Ready   bool `json:"ready"`   // Data loaded
	Loading bool `json:"loading"` // Currently loading
}

// Message represents a chat message (defined here to break cycle)
type Message struct {
	ID        string    `json:"id"`
	Role      Role      `json:"role"`
	Content   string    `json:"content"`
	Timestamp time.Time `json:"timestamp"`
	ToolCall  *ToolCall `json:"tool_call,omitempty"`
	// Additional fields for loaded messages
	Model    string `json:"model,omitempty"`
	Provider string `json:"provider,omitempty"`
	Parts    []Part `json:"parts,omitempty"` // For assistant messages with multiple parts
}

// Role represents message role
type Role string

const (
	RoleUser      Role = "user"
	RoleAssistant Role = "assistant"
	RoleSystem    Role = "system"
	RoleTool      Role = "tool"
)

// Part represents a part of a message (for assistant responses)
type Part struct {
	ID         string `json:"id"`
	Type       string `json:"type"` // text, tool_use, tool_result
	Text       string `json:"text,omitempty"`
	ToolID     string `json:"tool_id,omitempty"`
	ToolName   string `json:"tool_name,omitempty"`
	ToolInput  string `json:"tool_input,omitempty"`
	ToolResult string `json:"tool_result,omitempty"`
	Status     string `json:"status,omitempty"` // pending, running, success, error
}

// ToolCall represents a tool call
type ToolCall struct {
	Tool   string `json:"tool"`
	Input  string `json:"input"`
	Result string `json:"result"`
	Status string `json:"status"`
}

// LocalStore holds local UI state (agent/model selection, input)
type LocalStore struct {
	// Agent/Model selection
	CurrentAgent string   `json:"current_agent"`
	CurrentModel ModelKey `json:"current_model"`
	ModelVariant string   `json:"model_variant"` // Optional model variant

	// Per-session selections (sessionID -> State)
	SessionState map[string]SessionLocalState `json:"session_state"`

	// Input state
	InputText  string      `json:"input_text"`
	InputParts []InputPart `json:"input_parts"` // Attachments

	// History
	InputHistory []string `json:"input_history"`
	HistoryIndex int      `json:"history_index"`

	// Last selection for undo
	LastSelection *LastSelection `json:"last_selection,omitempty"`
}

// SessionLocalState holds per-session agent/model selection
type SessionLocalState struct {
	Agent   string   `json:"agent"`
	Model   ModelKey `json:"model"`
	Variant string   `json:"variant"`
}

// LastSelection tracks the last selection for undo functionality
type LastSelection struct {
	Type    string    `json:"type"` // "agent", "model", or "variant"
	Agent   string    `json:"agent,omitempty"`
	Model   *ModelKey `json:"model,omitempty"`
	Variant string    `json:"variant,omitempty"`
}

// LayoutStore holds layout dimensions and panel states
type LayoutStore struct {
	// Terminal dimensions
	Width  int `json:"width"`
	Height int `json:"height"`

	// Sidebar (for wide terminals)
	Sidebar SidebarLayout `json:"sidebar"`

	// Mobile sidebar (for narrow terminals)
	MobileSidebar MobileSidebarLayout `json:"mobile_sidebar"`

	// Panel states per session (sessionKey -> state)
	SessionTabs map[string]SessionTabsState `json:"session_tabs"`
	SessionView map[string]SessionViewState `json:"session_view"`

	// Handoff state for session transitions
	Handoff *TabHandoff `json:"handoff,omitempty"`
}

// SidebarLayout holds sidebar layout state
type SidebarLayout struct {
	Opened            bool            `json:"opened"`             // Sidebar visibility
	Width             int             `json:"width"`              // Sidebar width (default 344)
	Workspaces        map[string]bool `json:"workspaces"`         // Per-directory workspace toggle
	WorkspacesDefault bool            `json:"workspaces_default"` // Default workspace state
}

// MobileSidebarLayout holds mobile sidebar state (for narrow terminals)
type MobileSidebarLayout struct {
	Opened bool `json:"opened"` // Mobile sidebar overlay visibility
}

// SessionTabsState holds tab state for a session
type SessionTabsState struct {
	All    []string `json:"all"`    // All open tabs
	Active string   `json:"active"` // Currently active tab
}

// SessionViewState holds view state for a session (scroll, etc.)
type SessionViewState struct {
	Scroll           map[string]ScrollState `json:"scroll"`             // Per-tab scroll position
	ReviewOpen       []string               `json:"review_open"`        // Open review paths
	PendingMessage   string                 `json:"pending_message"`    // Pending message ID
	PendingMessageAt int64                  `json:"pending_message_at"` // Timestamp
}

// ScrollState holds scroll position for a view
type ScrollState struct {
	Offset int `json:"offset"` // Scroll offset
	Height int `json:"height"` // Content height
}

// TabHandoff holds state for session transitions
type TabHandoff struct {
	Dir string `json:"dir"`
	ID  string `json:"id"`
	At  int64  `json:"at"`
}

// DialogStore holds the dialog stack
type DialogStore struct {
	Stack []DialogState `json:"stack"` // Active dialogs (top is last)
}

// ToastMsg represents a toast notification
type ToastMsg struct {
	Type        string    `json:"type"` // "success", "error", "warning", "info"
	Title       string    `json:"title"`
	Description string    `json:"description"`
	Duration    int       `json:"duration"` // milliseconds
	ShownAt     time.Time `json:"shown_at"`
}

// ===========================================
// AppState - Combined State
// ===========================================

// Pagination constants (matching OpenCode)
const (
	InitialMessagePageSize = 80  // Initial message load
	HistoryMessagePageSize = 200 // History load (loadMore)
)

// MessageMeta holds pagination metadata for a session's messages
type MessageMeta struct {
	Cursor   int64 `json:"cursor"`   // Next cursor for loading more (timestamp in milliseconds)
	Complete bool  `json:"complete"` // True when all messages loaded
	Limit    int   `json:"limit"`    // Current message count
	Loading  bool  `json:"loading"`  // Currently loading more messages
}

// AppState combines all state layers into a unified structure
type AppState struct {
	// Route - Navigation state
	Route     Route  `json:"route"`
	SessionID string `json:"session_id"`

	// KV - Persistent preferences
	KV KVStore `json:"kv"`

	// Sync - Server/database synced data
	Sync SyncStore `json:"sync"`

	// Local - UI state
	Local LocalStore `json:"local"`

	// Layout - Responsive layout
	Layout LayoutStore `json:"layout"`

	// Dialog - Modal stack
	Dialog DialogStore `json:"dialog"`

	// Message - Pagination metadata (per session, keyed by sessionID)
	MessageMeta map[string]MessageMeta `json:"message_meta"`

	// Status - Processing state
	Processing bool   `json:"processing"`
	StatusText string `json:"status_text"`

	// Toast - Notification
	Toast *ToastMsg `json:"toast,omitempty"`

	// Error state (not serialized)
	LastError error
	ShowError bool
}

// Responsive threshold - sidebar auto-hides below this width
const ResponsiveThreshold = 120

// IsResponsive returns true if terminal is narrow (sidebar should auto-hide)
func (l *LayoutStore) IsResponsive() bool {
	return l.Width < ResponsiveThreshold
}

// ShouldShowSidebar returns true if sidebar should be visible based on mode and width
func (l *LayoutStore) ShouldShowSidebar(mode SidebarMode) bool {
	switch mode {
	case SidebarModeShow:
		return true
	case SidebarModeHide:
		return false
	case SidebarModeAuto:
		return !l.IsResponsive()
	default:
		return !l.IsResponsive()
	}
}

// ===========================================
// Default Store Functions
// ===========================================

// DefaultKVStore returns the default persistent preferences
func DefaultKVStore() KVStore {
	return KVStore{
		Theme:             "default",
		ThemeDark:         true,
		SidebarMode:       SidebarModeAuto,
		SidebarWidth:      344, // OpenCode standard width
		SidebarWorkspaces: false,
		ShowTimestamps:    true,
		ReviewDiffStyle:   "split",
		TerminalHeight:    280,
		TerminalOpened:    false,
		FileTreeOpened:    false,
		FileTreeWidth:     200,
		FileTreeTab:       "changes",
		SessionWidth:      600,
	}
}

// DefaultSyncStore returns an empty sync store
func DefaultSyncStore() SyncStore {
	return SyncStore{
		Sessions:     []Session{},
		Messages:     []Message{},
		Providers:    []Provider{},
		Agents:       []Agent{},
		LSPServers:   []LSPServer{},
		MCPServers:   []MCPServer{},
		SessionLimit: 25, // Default session limit
	}
}

// DefaultLocalStore returns the default local state
func DefaultLocalStore() LocalStore {
	return LocalStore{
		SessionState: map[string]SessionLocalState{},
		InputParts:   []InputPart{},
		InputHistory: []string{},
		HistoryIndex: -1,
	}
}

// DefaultLayoutStore returns the default layout state
func DefaultLayoutStore() LayoutStore {
	return LayoutStore{
		Width:  80, // Default terminal width
		Height: 24, // Default terminal height
		Sidebar: SidebarLayout{
			Opened:            false,
			Width:             344,
			Workspaces:        map[string]bool{},
			WorkspacesDefault: false,
		},
		MobileSidebar: MobileSidebarLayout{
			Opened: false,
		},
		SessionTabs: map[string]SessionTabsState{},
		SessionView: map[string]SessionViewState{},
	}
}

// DefaultDialogStore returns an empty dialog store
func DefaultDialogStore() DialogStore {
	return DialogStore{
		Stack: []DialogState{},
	}
}

// NewAppState creates a new AppState with defaults
func NewAppState() AppState {
	return AppState{
		Route:       RouteSession,
		KV:          DefaultKVStore(),
		Sync:        DefaultSyncStore(),
		Local:       DefaultLocalStore(),
		Layout:      DefaultLayoutStore(),
		Dialog:      DefaultDialogStore(),
		MessageMeta: map[string]MessageMeta{},
		Processing:  false,
		StatusText:  "Ready",
	}
}

// ===========================================
// Store Methods
// ===========================================

// Push adds a dialog to the stack
func (d *DialogStore) Push(dialog DialogState) {
	d.Stack = append(d.Stack, dialog)
}

// Pop removes and returns the top dialog
func (d *DialogStore) Pop() *DialogState {
	if len(d.Stack) == 0 {
		return nil
	}
	top := d.Stack[len(d.Stack)-1]
	d.Stack = d.Stack[:len(d.Stack)-1]
	return &top
}

// Top returns the top dialog without removing it
func (d *DialogStore) Top() *DialogState {
	if len(d.Stack) == 0 {
		return nil
	}
	return &d.Stack[len(d.Stack)-1]
}

// HasOpen returns true if any dialog is open
func (d *DialogStore) HasOpen() bool {
	return len(d.Stack) > 0
}

// ===========================================
// AppState Methods
// ===========================================

// SetDimensions updates terminal dimensions
func (s *AppState) SetDimensions(width, height int) {
	s.Layout.Width = width
	s.Layout.Height = height
}

// ToggleSidebar toggles sidebar visibility
func (s *AppState) ToggleSidebar() {
	if s.Layout.IsResponsive() {
		s.Layout.MobileSidebar.Opened = !s.Layout.MobileSidebar.Opened
	} else {
		s.Layout.Sidebar.Opened = !s.Layout.Sidebar.Opened
	}
}

// OpenSidebar shows the sidebar
func (s *AppState) OpenSidebar() {
	if s.Layout.IsResponsive() {
		s.Layout.MobileSidebar.Opened = true
	} else {
		s.Layout.Sidebar.Opened = true
	}
}

// CloseSidebar hides the sidebar
func (s *AppState) CloseSidebar() {
	s.Layout.Sidebar.Opened = false
	s.Layout.MobileSidebar.Opened = false
}

// IsSidebarVisible returns true if sidebar is currently visible
func (s *AppState) IsSidebarVisible() bool {
	return s.Layout.ShouldShowSidebar(s.KV.SidebarMode) &&
		(s.Layout.Sidebar.Opened || s.Layout.MobileSidebar.Opened)
}

// PushDialog opens a new dialog
func (s *AppState) PushDialog(dialogType DialogType) {
	s.Dialog.Push(DialogState{
		Type: dialogType,
		Open: true,
	})
}

// PopDialog closes the current dialog
func (s *AppState) PopDialog() {
	s.Dialog.Pop()
}

// SetCurrentModel updates the current model selection
func (s *AppState) SetCurrentModel(model ModelKey) {
	s.Local.CurrentModel = model
	if s.SessionID != "" {
		s.Local.SessionState[s.SessionID] = SessionLocalState{
			Agent:   s.Local.CurrentAgent,
			Model:   model,
			Variant: s.Local.ModelVariant,
		}
	}
}

// SetCurrentAgent updates the current agent selection
func (s *AppState) SetCurrentAgent(agent string) {
	s.Local.CurrentAgent = agent
	if s.SessionID != "" {
		state := s.Local.SessionState[s.SessionID]
		state.Agent = agent
		s.Local.SessionState[s.SessionID] = state
	}
}

// AddMessage appends a message to the message list
func (s *AppState) AddMessage(msg Message) {
	msg.ID = fmt.Sprintf("msg-%d", time.Now().UnixNano())
	s.Sync.Messages = append(s.Sync.Messages, msg)
}

// SetMessages replaces the message list
func (s *AppState) SetMessages(messages []Message) {
	s.Sync.Messages = messages
	s.Sync.MessageTotal = len(messages)
}

// SetSessions updates the session list
func (s *AppState) SetSessions(sessions []Session) {
	s.Sync.Sessions = sessions
}

// SetActiveSession sets the current active session
func (s *AppState) SetActiveSession(session *Session) {
	if session != nil {
		s.SessionID = session.ID
		s.Sync.Messages = []Message{} // Clear messages for new session
	}
}

// SetTheme updates the current theme
func (s *AppState) SetTheme(theme string) {
	s.KV.Theme = theme
}

// SetStatus updates the status text
func (s *AppState) SetStatus(status string) {
	s.StatusText = status
}

// ShowToast displays a toast notification
func (s *AppState) ShowToast(typ, title, description string) {
	s.Toast = &ToastMsg{
		Type:        typ,
		Title:       title,
		Description: description,
		Duration:    3000, // 3 seconds
		ShownAt:     time.Now(),
	}
}

// ClearToast removes the toast notification
func (s *AppState) ClearToast() {
	s.Toast = nil
}

// ===========================================
// Message Pagination Methods
// ===========================================

// GetMessageMeta returns the message pagination metadata for a session
func (s *AppState) GetMessageMeta(sessionID string) MessageMeta {
	if s.MessageMeta == nil {
		s.MessageMeta = map[string]MessageMeta{}
	}
	meta, ok := s.MessageMeta[sessionID]
	if !ok {
		return MessageMeta{}
	}
	return meta
}

// SetMessageMeta updates the message pagination metadata for a session
func (s *AppState) SetMessageMeta(sessionID string, meta MessageMeta) {
	if s.MessageMeta == nil {
		s.MessageMeta = map[string]MessageMeta{}
	}
	s.MessageMeta[sessionID] = meta
}

// HistoryMore returns true if there are more messages to load
func (s *AppState) HistoryMore(sessionID string) bool {
	meta := s.GetMessageMeta(sessionID)
	// Need messages loaded, not complete, and have a cursor
	if len(s.Sync.Messages) == 0 {
		return false
	}
	if meta.Complete {
		return false
	}
	return meta.Cursor > 0
}

// HistoryLoading returns true if currently loading more messages
func (s *AppState) HistoryLoading(sessionID string) bool {
	meta := s.GetMessageMeta(sessionID)
	return meta.Loading
}

// SetHistoryLoading sets the loading state for history
func (s *AppState) SetHistoryLoading(sessionID string, loading bool) {
	meta := s.GetMessageMeta(sessionID)
	meta.Loading = loading
	s.SetMessageMeta(sessionID, meta)
}

// PrependMessages adds older messages at the beginning of the list (for loadMore)
func (s *AppState) PrependMessages(messages []Message, cursor int64, complete bool) {
	// Prepend older messages
	s.Sync.Messages = append(messages, s.Sync.Messages...)

	// Update metadata
	meta := s.GetMessageMeta(s.SessionID)
	meta.Cursor = cursor
	meta.Complete = complete
	meta.Limit = len(s.Sync.Messages)
	s.SetMessageMeta(s.SessionID, meta)
}
