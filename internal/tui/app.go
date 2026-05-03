// Package tui provides the main TUI application.
// This file implements the Bubble Tea Model interface with layered state management.
package tui

import (
	"context"
	"encoding/json"
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
	"github.com/rhony08/magicode/internal/bus"
	"github.com/rhony08/magicode/internal/database"
	"github.com/rhony08/magicode/internal/opencode"
	"github.com/rhony08/magicode/internal/provider"
	"github.com/rhony08/magicode/internal/session"
	"github.com/rhony08/magicode/internal/tool"
	"github.com/rhony08/magicode/internal/tui/component"
	"github.com/rhony08/magicode/internal/tui/dialog"
	"github.com/rhony08/magicode/internal/tui/layout"
	"github.com/rhony08/magicode/internal/tui/types"
	"github.com/rhony08/magicode/internal/tui/util"
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

	// OpenCode configuration flag
	useOpenCode bool

	// Default model from config/session (for fallback)
	configDefaultModel ModelKey

	// AI processing
	processor        *session.Processor
	providerRegistry *provider.ProviderRegistry
	toolRegistry     *tool.Registry

	// Event channel for streaming updates (from bus to tea)
	eventChan chan tea.Msg

	// Bus service reference (for cleanup)
	busService *bus.Service

	// Streaming state - tracks current streaming message and parts
	streamingState streamingState

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
	UseOpenCode     bool        // Use OpenCode configuration for agents/providers
	DefaultModel    ModelKey    // Default model from config/session (optional)

	// AI processing components (optional, can be set later with SetProcessor)
	Processor        *session.Processor
	ProviderRegistry *provider.ProviderRegistry
	ToolRegistry     *tool.Registry
	BusService       *bus.Service // Bus service for streaming events
}

// streamingState tracks current streaming message state for real-time updates
type streamingState struct {
	sessionID string        // Session being streamed
	messageID string        // Message being streamed
	parts     map[int]*Part // Parts by index (for delta appending)
	isActive  bool          // Whether streaming is active
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
	} else if cfg.SessionID != "" {
		// If only SessionID is provided, set it
		state.SessionID = cfg.SessionID
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
		useOpenCode:      cfg.UseOpenCode,
		configDefaultModel: cfg.DefaultModel,
		processor:        cfg.Processor,
		providerRegistry: cfg.ProviderRegistry,
		toolRegistry:     cfg.ToolRegistry,
		busService:       cfg.BusService,
		eventChan:        make(chan tea.Msg, 100), // Buffer for streaming events
		streamingState:   streamingState{parts: make(map[int]*Part)},
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

// SetProcessor sets the processor and related components after initialization
// This is useful when the database is opened after the app is created
func (a *App) SetProcessor(processor *session.Processor, providerRegistry *provider.ProviderRegistry, toolRegistry *tool.Registry) {
	a.processor = processor
	a.providerRegistry = providerRegistry
	a.toolRegistry = toolRegistry
}

// SetBusService sets the bus service for streaming events
func (a *App) SetBusService(busService *bus.Service) {
	a.busService = busService
}

// subscribeToBus subscribes to bus events and converts them to tea.Msg
func (a *App) subscribeToBus() tea.Cmd {
	if a.busService == nil {
		return nil
	}

	// Subscribe to part updates
	partUpdatedChan, cleanupPart := a.busService.Subscribe(session.EventPartUpdated)
	go func() {
		defer cleanupPart()
		for payload := range partUpdatedChan {
			props := payload.Properties.(map[string]interface{})
			msg := StreamPartUpdatedMsg{
				SessionID: props["session_id"].(string),
				MessageID: props["message_id"].(string),
				PartID:    props["part_id"].(string),
				Index:     props["index"].(int),
				Delta:     props["delta"].(string),
				DeltaType: props["delta_type"].(string),
			}
			a.eventChan <- msg
		}
	}()

	// Subscribe to part created
	partCreatedChan, cleanupCreated := a.busService.Subscribe(session.EventPartCreated)
	go func() {
		defer cleanupCreated()
		for payload := range partCreatedChan {
			props := payload.Properties.(map[string]interface{})
			msg := StreamPartCreatedMsg{
				SessionID: props["session_id"].(string),
				MessageID: props["message_id"].(string),
				PartID:    props["part_id"].(string),
				Index:     props["index"].(int),
				Type:      props["type"].(string),
			}
			a.eventChan <- msg
		}
	}()

	// Subscribe to message complete
	msgCompleteChan, cleanupComplete := a.busService.Subscribe(session.EventMessageComplete)
	go func() {
		defer cleanupComplete()
		for payload := range msgCompleteChan {
			props := payload.Properties.(map[string]interface{})
			msg := StreamMessageCompleteMsg{
				SessionID: props["session_id"].(string),
				MessageID: props["message_id"].(string),
			}
			a.eventChan <- msg
		}
	}()

	// Subscribe to tool events
	toolPendingChan, cleanupToolPending := a.busService.Subscribe(session.EventToolCallPending)
	go func() {
		defer cleanupToolPending()
		for payload := range toolPendingChan {
			props := payload.Properties.(map[string]interface{})
			msg := StreamToolPendingMsg{
				SessionID: props["session_id"].(string),
				MessageID: props["message_id"].(string),
				PartID:    props["part_id"].(string),
				ToolName:  props["tool_name"].(string),
				ToolID:    props["tool_id"].(string),
			}
			if input, ok := props["input"].(map[string]interface{}); ok {
				// Convert input to JSON string for display
				inputJSON, _ := json.Marshal(input)
				msg.Input = string(inputJSON)
			}
			a.eventChan <- msg
		}
	}()

	// Subscribe to tool running events
	toolRunningChan, cleanupToolRunning := a.busService.Subscribe(session.EventToolCallRunning)
	go func() {
		defer cleanupToolRunning()
		for payload := range toolRunningChan {
			props := payload.Properties.(map[string]interface{})
			msg := StreamToolRunningMsg{
				SessionID: props["session_id"].(string),
				ToolName:  props["tool_name"].(string),
				ToolID:    props["tool_id"].(string),
			}
			a.eventChan <- msg
		}
	}()

	// Subscribe to tool complete events
	toolCompleteChan, cleanupToolComplete := a.busService.Subscribe(session.EventToolCallComplete)
	go func() {
		defer cleanupToolComplete()
		for payload := range toolCompleteChan {
			props := payload.Properties.(map[string]interface{})
			msg := StreamToolCompleteMsg{
				SessionID: props["session_id"].(string),
				ToolName:  props["tool_name"].(string),
				ToolID:    props["tool_id"].(string),
				Result:    props["result"].(string),
				IsError:   props["is_error"].(bool),
			}
			if resultPart, ok := props["result_part"].(*database.Part); ok {
				msg.ResultPartID = resultPart.ID
			}
			a.eventChan <- msg
		}
	}()

	// Subscribe to error events
	errorChan, cleanupError := a.busService.Subscribe(session.EventStreamError)
	go func() {
		defer cleanupError()
		for payload := range errorChan {
			props := payload.Properties.(map[string]interface{})
			msg := StreamErrorMsg{
				SessionID: props["session_id"].(string),
				Error:     props["error"].(string),
				ErrorType: props["error_type"].(string),
			}
			a.eventChan <- msg
		}
	}()

	// Return a command that waits for events from the channel
	return tea.Batch(
		a.waitForEvent(),
	)
}

// waitForEvent returns a command that waits for the next event
func (a *App) waitForEvent() tea.Cmd {
	return func() tea.Msg {
		return <-a.eventChan
	}
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

	// Initialize providers based on config source
	if a.useOpenCode {
		// Use OpenCode config - sync will load minimal providers
		a.syncOpenCodeConfig()
	} else if a.providerRegistry != nil {
		// Use provider registry (from env vars) when not using OpenCode config
		a.populateProvidersFromRegistry()
	}

	// Set default model if provided in config (from session or config file)
	if a.state.Local.CurrentModel.ProviderID == "" {
		// First check if DefaultModel was passed in config
		if cfgDefaultModel, ok := a.getConfigDefaultModel(); ok && cfgDefaultModel.ProviderID != "" {
			a.state.SetCurrentModel(cfgDefaultModel)
			log.Info("Set model from config default", "provider", cfgDefaultModel.ProviderID, "model", cfgDefaultModel.ModelID)
		} else if len(a.state.Local.Providers) > 0 {
			// Pick first available model from loaded providers
			for _, prov := range a.state.Local.Providers {
				for modelID := range prov.Models {
					a.state.Local.CurrentModel = ModelKey{
						ProviderID: prov.ID,
						ModelID:    modelID,
					}
					log.Info("Set default model from providers", "provider", prov.ID, "model", modelID)
					break
				}
				break
			}
		}
	}

	// Initialize footer with current agent and model
	a.updateFooterAgent()
	a.updateFooterModel(a.state.Local.CurrentModel)

	// Set initial viewport content if there are messages
	if len(a.state.Sync.Messages) > 0 {
		a.messageViewport.SetContent(a.buildMessagesContent())
		a.messageViewport.GotoBottom()
	}

	return tea.Batch(
		a.spinner.Tick,
		textinput.Blink,
		a.loadSessionsFromDB(), // Load sessions from database on startup
		a.subscribeToBus(),     // Subscribe to bus events for streaming
	)
}

// syncOpenCodeConfig reads OpenCode configuration and syncs it to the app state
// This is memory-efficient: only loads the current model for existing sessions,
// or the default/recent model for new sessions. Full config is lazy-loaded when needed.
func (a *App) syncOpenCodeConfig() {
	// Create config reader with default paths
	configReader := opencode.DefaultConfigReader()

	// Check if OpenCode config exists
	if !configReader.ConfigExists() {
		log.Info("OpenCode config not found, skipping sync")
		return
	}

	// Determine if this is an existing session (has messages) or new session
	isExistingSession := len(a.state.Sync.Messages) > 0

	if isExistingSession {
		// For existing sessions: only load the model that was actually used
		a.syncExistingSessionModel(configReader)
	} else {
		// For new sessions: load minimal defaults
		a.syncNewSessionDefaults(configReader)
	}

	log.Info("OpenCode config sync completed",
		"session_type", map[bool]string{true: "existing", false: "new"}[isExistingSession],
		"current_model", a.state.Local.CurrentModel.ModelID,
		"current_agent", a.state.Local.CurrentAgent)
}

// syncExistingSessionModel loads only the model used in an existing session
func (a *App) syncExistingSessionModel(configReader *opencode.ConfigReader) {
	// Find the model used in the last assistant message
	var lastModelKey ModelKey
	for i := len(a.state.Sync.Messages) - 1; i >= 0; i-- {
		msg := a.state.Sync.Messages[i]
		if msg.Role == RoleAssistant && msg.Model != "" && msg.Provider != "" {
			lastModelKey = ModelKey{
				ProviderID: msg.Provider,
				ModelID:    msg.Model,
			}
			log.Info("Found model from existing session", "provider", msg.Provider, "model", msg.Model)
			break
		}
	}

	// If we found a model used in this session, use it
	if lastModelKey.ModelID != "" {
		a.state.SetCurrentModel(lastModelKey)
	}

	// Load minimal agent metadata for cycling (names only, no full content)
	agents := a.loadMinimalAgents(configReader)
	if len(agents) > 0 {
		a.state.Local.Agents = agents

		// Set default agent if none selected
		if a.state.Local.CurrentAgent == "" {
			for _, agent := range agents {
				if !agent.Hidden {
					a.state.SetCurrentAgent(agent.Name)
					break
				}
			}
		}
	}

	// Load minimal provider metadata for the current model only
	a.loadMinimalProviderForModel(configReader, lastModelKey)

	// Update footer with loaded agent and model
	a.updateFooterAgent()
	a.updateFooterModel(lastModelKey)
}

// syncNewSessionDefaults loads default model/agent for new sessions
func (a *App) syncNewSessionDefaults(configReader *opencode.ConfigReader) {
	// Read model state to get recent model
	modelState, err := configReader.ReadModelState()
	if err != nil {
		log.Warn("Failed to read OpenCode model state", "error", err)
	}

	// Use recent model if available
	if modelState != nil && len(modelState.Recent) > 0 {
		recent := modelState.Recent[0]
		a.state.SetCurrentModel(ModelKey{
			ProviderID: recent.ProviderID,
			ModelID:    recent.ModelID,
		})
		log.Info("Set model from OpenCode recent", "provider", recent.ProviderID, "model", recent.ModelID)
	} else {
		// Fall back to first valid model from config
		modelRef, err := configReader.GetFirstValidModel()
		if err == nil && modelRef != nil {
			a.state.SetCurrentModel(ModelKey{
				ProviderID: modelRef.ProviderID,
				ModelID:    modelRef.ModelID,
			})
			log.Info("Set default model from OpenCode config", "provider", modelRef.ProviderID, "model", modelRef.ModelID)
		}
	}

	// Load minimal agent metadata
	agents := a.loadMinimalAgents(configReader)
	if len(agents) > 0 {
		a.state.Local.Agents = agents

		// Set default agent
		for _, agent := range agents {
			if !agent.Hidden {
				a.state.SetCurrentAgent(agent.Name)
				break
			}
		}
	}

	// Load minimal provider metadata for current model
	a.loadMinimalProviderForModel(configReader, a.state.Local.CurrentModel)

	// Update footer with loaded agent and model
	a.updateFooterAgent()
	a.updateFooterModel(a.state.Local.CurrentModel)
}

// loadMinimalAgents loads only agent metadata (no full content/prompts)
func (a *App) loadMinimalAgents(configReader *opencode.ConfigReader) []Agent {
	opencodeAgents, err := configReader.ReadAgents()
	if err != nil {
		log.Warn("Failed to read OpenCode agents", "error", err)
		return nil
	}

	var agents []Agent
	for _, opencodeAgent := range opencodeAgents {
		// Only store metadata, not the full content (saves memory)
		agents = append(agents, Agent{
			Name:        opencodeAgent.Name,
			Description: opencodeAgent.Description,
			Mode:        opencodeAgent.Mode,
			Color:       opencodeAgent.Color,
			Temperature: opencodeAgent.Temperature,
			Tools:       opencodeAgent.Tools,
			Hidden:      opencodeAgent.Hidden,
			// Content is empty - will be lazy-loaded when needed
		})
	}

	log.Info("Loaded minimal agent metadata", "count", len(agents))
	return agents
}

// loadMinimalProviderForModel loads only the provider metadata needed for a specific model
func (a *App) loadMinimalProviderForModel(configReader *opencode.ConfigReader, modelKey ModelKey) {
	if modelKey.ProviderID == "" || modelKey.ModelID == "" {
		return
	}

	// Read config to get provider details
	config, err := configReader.ReadConfig()
	if err != nil || config == nil {
		return
	}

	// Find the specific provider
	opencodeProvider, exists := config.Providers[modelKey.ProviderID]
	if !exists {
		log.Warn("Provider not found in OpenCode config", "provider", modelKey.ProviderID)
		return
	}

	// Find the specific model
	opencodeModel, exists := opencodeProvider.Models[modelKey.ModelID]
	if !exists {
		log.Warn("Model not found in OpenCode config", "provider", modelKey.ProviderID, "model", modelKey.ModelID)
		return
	}

	// Create minimal provider with just this model
	models := map[string]Model{
		modelKey.ModelID: {
			ID:         opencodeModel.ID,
			Name:       opencodeModel.Name,
			ProviderID: modelKey.ProviderID,
			// Modalities and limits - only store what's needed for UI
			Modalities: types.ModelModalities{
				Input:  opencodeModel.Modalities.Input,
				Output: opencodeModel.Modalities.Output,
			},
			Limit: types.ModelLimit{
				Context: opencodeModel.Limit.Context,
				Output:  opencodeModel.Limit.Output,
			},
			// Options omitted - only needed when making API calls
		},
	}

	provider := Provider{
		ID:        opencodeProvider.ID,
		Name:      opencodeProvider.Name,
		Models:    models,
		Connected: false,
		// APIKey and BaseURL omitted - only load when making API calls
	}

	// Store only this minimal provider
	a.state.Local.Providers = []Provider{provider}
	log.Info("Loaded minimal provider metadata", "provider", provider.ID, "model_count", 1)
}

// LoadFullAgent loads full agent details including content (lazy loading)
func (a *App) LoadFullAgent(agentName string) (*Agent, error) {
	configReader := opencode.DefaultConfigReader()

	// Find agent file
	opencodeAgents, err := configReader.ReadAgents()
	if err != nil {
		return nil, fmt.Errorf("failed to read agents: %w", err)
	}

	for _, opencodeAgent := range opencodeAgents {
		if opencodeAgent.Name == agentName {
			// Convert permission
			permission := make(map[string]interface{})
			for k, v := range opencodeAgent.Permission {
				permission[k] = v
			}

			return &Agent{
				Name:        opencodeAgent.Name,
				Description: opencodeAgent.Description,
				Mode:        opencodeAgent.Mode,
				Color:       opencodeAgent.Color,
				Temperature: opencodeAgent.Temperature,
				Tools:       opencodeAgent.Tools,
				Permission:  permission,
				Content:     opencodeAgent.Content, // Full content loaded
				Hidden:      opencodeAgent.Hidden,
			}, nil
		}
	}

	return nil, fmt.Errorf("agent not found: %s", agentName)
}

// LoadFullProvider loads full provider details including API keys (lazy loading)
func (a *App) LoadFullProvider(providerID string) (*Provider, error) {
	configReader := opencode.DefaultConfigReader()

	providers, err := configReader.ReadProviders()
	if err != nil {
		return nil, fmt.Errorf("failed to read providers: %w", err)
	}

	for _, opencodeProvider := range providers {
		if opencodeProvider.ID == providerID {
			// Convert all models
			models := make(map[string]Model)
			for modelID, opencodeModel := range opencodeProvider.Models {
				var thinkingOptions *types.ModelThinkingOptions
				if opencodeModel.Options.Thinking != nil {
					thinkingOptions = &types.ModelThinkingOptions{
						Type:         opencodeModel.Options.Thinking.Type,
						BudgetTokens: opencodeModel.Options.Thinking.BudgetTokens,
					}
				}

				models[modelID] = Model{
					ID:         opencodeModel.ID,
					Name:       opencodeModel.Name,
					ProviderID: providerID,
					Modalities: types.ModelModalities{
						Input:  opencodeModel.Modalities.Input,
						Output: opencodeModel.Modalities.Output,
					},
					Options: types.ModelOptions{
						Thinking: thinkingOptions,
					},
					Limit: types.ModelLimit{
						Context: opencodeModel.Limit.Context,
						Output:  opencodeModel.Limit.Output,
					},
				}
			}

			return &Provider{
				ID:        opencodeProvider.ID,
				Name:      opencodeProvider.Name,
				Models:    models,
				NPM:       opencodeProvider.NPM,
				BaseURL:   opencodeProvider.BaseURL,
				APIKey:    opencodeProvider.APIKey, // API key loaded
				Connected: false,
			}, nil
		}
	}

	return nil, fmt.Errorf("provider not found: %s", providerID)
}

// EnsureModelLoaded checks if a model is loaded and loads it if necessary
// Returns true if the model is now available, false if it couldn't be loaded
func (a *App) EnsureModelLoaded(modelKey ModelKey) bool {
	// Check if model is already loaded
	for _, provider := range a.state.Local.Providers {
		if provider.ID == modelKey.ProviderID {
			if _, exists := provider.Models[modelKey.ModelID]; exists {
				return true // Model already loaded
			}
		}
	}

	// Model not loaded, need to load it
	log.Info("Lazy-loading model", "provider", modelKey.ProviderID, "model", modelKey.ModelID)
	configReader := opencode.DefaultConfigReader()
	a.loadMinimalProviderForModel(configReader, modelKey)

	// Verify it was loaded
	for _, provider := range a.state.Local.Providers {
		if provider.ID == modelKey.ProviderID {
			if _, exists := provider.Models[modelKey.ModelID]; exists {
				return true
			}
		}
	}

	log.Warn("Failed to load model", "provider", modelKey.ProviderID, "model", modelKey.ModelID)
	return false
}

// SwitchAgent switches to a new agent and ensures its preferred model is loaded
// This is called when subagents are invoked or when cycling agents
func (a *App) SwitchAgent(agentName string) error {
	// Get the agent
	agent := a.state.GetAgent(agentName)
	if agent == nil {
		return fmt.Errorf("agent not found: %s", agentName)
	}

	// Set the agent
	a.state.SetCurrentAgent(agentName)

	// Update footer with new agent
	a.footer.SetAgent(agentName, agent.Color)

	// If agent has a preferred model, ensure it's loaded and switch to it
	if agent.Model != nil {
		if a.EnsureModelLoaded(*agent.Model) {
			a.state.SetCurrentModel(*agent.Model)
			// Update footer with agent's preferred model
			a.updateFooterModel(*agent.Model)
			log.Info("Switched to agent's preferred model", "agent", agentName, "model", agent.Model.ModelID)
		} else {
			log.Warn("Agent's preferred model not available, keeping current model", "agent", agentName, "model", agent.Model.ModelID)
		}
	}

	return nil
}

// SafeCycleAgent cycles to the next/previous agent with lazy-loading support
func (a *App) SafeCycleAgent(direction int) {
	if len(a.state.Local.Agents) == 0 {
		return
	}

	// Find current agent index
	currentIdx := -1
	for i, agent := range a.state.Local.Agents {
		if agent.Name == a.state.Local.CurrentAgent {
			currentIdx = i
			break
		}
	}

	// Calculate next index with wrapping
	nextIdx := currentIdx + direction
	if nextIdx < 0 {
		nextIdx = len(a.state.Local.Agents) - 1
	} else if nextIdx >= len(a.state.Local.Agents) {
		nextIdx = 0
	}

	// Switch to the agent (handles model loading)
	nextAgent := a.state.Local.Agents[nextIdx]
	if err := a.SwitchAgent(nextAgent.Name); err != nil {
		log.Warn("Failed to switch agent", "agent", nextAgent.Name, "error", err)
	}
}

// SafeCycleModel cycles to the next/previous model with lazy-loading support
func (a *App) SafeCycleModel(direction int) {
	// Get all available models by reading full config
	configReader := opencode.DefaultConfigReader()
	config, err := configReader.ReadConfig()
	if err != nil || config == nil {
		log.Warn("Cannot cycle models: config not available")
		return
	}

	// Build list of all available models
	var allModels []ModelKey
	for providerID, provider := range config.Providers {
		for modelID := range provider.Models {
			allModels = append(allModels, ModelKey{
				ProviderID: providerID,
				ModelID:    modelID,
			})
		}
	}

	if len(allModels) == 0 {
		return
	}

	// Find current model index
	currentIdx := -1
	for i, model := range allModels {
		if model.ProviderID == a.state.Local.CurrentModel.ProviderID &&
			model.ModelID == a.state.Local.CurrentModel.ModelID {
			currentIdx = i
			break
		}
	}

	// Calculate next index with wrapping
	nextIdx := currentIdx + direction
	if nextIdx < 0 {
		nextIdx = len(allModels) - 1
	} else if nextIdx >= len(allModels) {
		nextIdx = 0
	}

	nextModel := allModels[nextIdx]

	// Ensure the model is loaded before switching
	if a.EnsureModelLoaded(nextModel) {
		a.state.SetCurrentModel(nextModel)
		// Update footer with new model
		a.updateFooterModel(nextModel)
		log.Info("Cycled to model", "provider", nextModel.ProviderID, "model", nextModel.ModelID)
	} else {
		log.Warn("Failed to cycle to model", "provider", nextModel.ProviderID, "model", nextModel.ModelID)
	}
}

// ValidateCurrentModel checks if the current model is valid and loaded
// If not, attempts to load a fallback model
func (a *App) ValidateCurrentModel() bool {
	// Check if current model is valid
	if a.state.IsModelValid(a.state.Local.CurrentModel) {
		return true
	}

	log.Warn("Current model not valid, attempting to load fallback")

	// Try to load the current model
	if a.EnsureModelLoaded(a.state.Local.CurrentModel) {
		return true
	}

	// If that fails, try to get first valid model from config
	configReader := opencode.DefaultConfigReader()
	modelRef, err := configReader.GetFirstValidModel()
	if err == nil && modelRef != nil {
		fallbackModel := ModelKey{
			ProviderID: modelRef.ProviderID,
			ModelID:    modelRef.ModelID,
		}
		if a.EnsureModelLoaded(fallbackModel) {
			a.state.SetCurrentModel(fallbackModel)
			log.Info("Set fallback model", "provider", fallbackModel.ProviderID, "model", fallbackModel.ModelID)
			return true
		}
	}

	log.Error("No valid model available")
	return false
}

// updateFooterModel updates the footer with the current model information
func (a *App) updateFooterModel(modelKey ModelKey) {
	// Find model name from providers
	modelName := modelKey.ModelID
	for _, provider := range a.state.Local.Providers {
		if provider.ID == modelKey.ProviderID {
			if model, exists := provider.Models[modelKey.ModelID]; exists {
				modelName = model.Name
				break
			}
		}
	}
	a.footer.SetModel(modelName, modelKey.ModelID)
}

// updateFooterAgent updates the footer with the current agent information
func (a *App) updateFooterAgent() {
	agent := a.state.GetCurrentAgent()
	if agent != nil {
		a.footer.SetAgent(agent.Name, agent.Color)
	} else {
		a.footer.SetAgent("default", "")
	}
}

// populateProvidersFromRegistry populates state providers from the provider registry
func (a *App) populateProvidersFromRegistry() {
	if a.providerRegistry == nil {
		return
	}

	providers := a.providerRegistry.ListProviders()
	for _, prov := range providers {
		info := prov.Info()

		// Convert to TUI Provider type
		tuiProvider := Provider{
			ID:        string(info.ID),
			Name:      info.Name,
			Connected: true, // Assume connected if registered
			Models:    make(map[string]Model),
		}

		// Add models
		for modelID, modelInfo := range info.Models {
			_, modelName := provider.ParseModelID(modelID)
			tuiProvider.Models[modelName] = Model{
				ID:   modelName,
				Name: modelInfo.Name,
			}
		}

		a.state.Local.Providers = append(a.state.Local.Providers, tuiProvider)
	}

	log.Info("Populated providers from registry", "count", len(a.state.Local.Providers))
}

// getConfigDefaultModel returns the default model passed in config
func (a *App) getConfigDefaultModel() (ModelKey, bool) {
	if a.configDefaultModel.ProviderID != "" && a.configDefaultModel.ModelID != "" {
		return a.configDefaultModel, true
	}
	return ModelKey{}, false
}

// triggerAutocomplete triggers autocomplete based on current input
func (a *App) triggerAutocomplete() {
	value := a.prompt.GetValue()
	cursorPos := a.prompt.GetCursorPos()

	// Get word at cursor
	word := a.getWordAtCursor(value, cursorPos)

	// Check for trigger characters
	if strings.HasPrefix(word, "@") {
		query := word[1:] // Remove @
		options := a.getAgentCompletions(query)
		a.prompt.GetAutocomplete().Show(options, "@", query)
	} else if strings.HasPrefix(word, "/") {
		// Only trigger / commands at start of line or after newline
		beforeCursor := value[:cursorPos]
		lastNewline := strings.LastIndex(beforeCursor, "\n")
		if lastNewline == -1 || cursorPos-lastNewline <= len(word)+1 {
			query := word[1:] // Remove /
			options := a.getCommandCompletions(query)
			a.prompt.GetAutocomplete().Show(options, "/", query)
		}
	}
}

// checkAutocomplete checks if autocomplete should be triggered or updated
func (a *App) checkAutocomplete() {
	value := a.prompt.GetValue()
	cursorPos := a.prompt.GetCursorPos()

	// Get word at cursor
	word := a.getWordAtCursor(value, cursorPos)

	// Check if we should trigger or update autocomplete
	if strings.HasPrefix(word, "@") {
		query := word[1:] // Remove @
		if !a.prompt.IsAutocompleteVisible() || a.prompt.GetAutocomplete().GetTrigger() != "@" {
			options := a.getAgentCompletions(query)
			a.prompt.GetAutocomplete().Show(options, "@", query)
		} else {
			// Update existing autocomplete with new query
			options := a.getAgentCompletions(query)
			a.prompt.SetAutocompleteOptions(options)
		}
	} else if strings.HasPrefix(word, "/") {
		// Only trigger / commands at start of line or after newline
		beforeCursor := value[:cursorPos]
		lastNewline := strings.LastIndex(beforeCursor, "\n")
		if lastNewline == -1 || cursorPos-lastNewline <= len(word)+1 {
			query := word[1:] // Remove /
			if !a.prompt.IsAutocompleteVisible() || a.prompt.GetAutocomplete().GetTrigger() != "/" {
				options := a.getCommandCompletions(query)
				a.prompt.GetAutocomplete().Show(options, "/", query)
			} else {
				// Update existing autocomplete with new query
				options := a.getCommandCompletions(query)
				a.prompt.SetAutocompleteOptions(options)
			}
		}
	} else if a.prompt.IsAutocompleteVisible() {
		// Hide autocomplete if word doesn't start with trigger
		a.prompt.HideAutocomplete()
	}
}

// getWordAtCursor extracts the word at cursor position
func (a *App) getWordAtCursor(text string, pos int) string {
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

// isWordSeparator checks if a character is a word separator
func isWordSeparator(c byte) bool {
	return c == ' ' || c == '\t' || c == '\n' || c == '\r'
}

// getAgentCompletions returns agent completions for @ trigger
func (a *App) getAgentCompletions(query string) []layout.AutocompleteOption {
	var options []layout.AutocompleteOption

	// Add agents from state
	for _, agent := range a.state.Local.Agents {
		if query == "" || strings.Contains(strings.ToLower(agent.Name), strings.ToLower(query)) {
			options = append(options, layout.AutocompleteOption{
				Value:       "@" + agent.Name,
				Display:     agent.Name,
				Description: agent.Description,
				Icon:        "🤖",
			})
		}
	}

	return options
}

// getCommandCompletions returns slash command completions
func (a *App) getCommandCompletions(query string) []layout.AutocompleteOption {
	commands := []struct {
		Name        string
		Description string
		Icon        string
	}{
		{"status", "Show system status", "📊"},
		{"compact", "Compact session history", "🗜"},
		{"export", "Export session", "📤"},
		{"help", "Show help", "❓"},
		{"theme", "Change theme", "🎨"},
		{"model", "Change model", "🤖"},
		{"agent", "Change agent", "👤"},
	}

	var options []layout.AutocompleteOption
	for _, cmd := range commands {
		if query == "" || strings.HasPrefix(cmd.Name, query) {
			options = append(options, layout.AutocompleteOption{
				Value:       "/" + cmd.Name,
				Display:     cmd.Name,
				Description: cmd.Description,
				Icon:        cmd.Icon,
			})
		}
	}
	return options
}

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
		// appendToLastMessage now handles SetContent and GotoBottom
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
			// Auto-scroll to show new message
			a.scrollToBottom(false)
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

	// Streaming events from bus
	case StreamPartCreatedMsg:
		// New part created - start tracking it for streaming updates
		if msg.SessionID == a.state.SessionID {
			a.handleStreamPartCreated(msg)
			cmds = append(cmds, a.waitForEvent()) // Continue waiting for events
		}

	case StreamPartUpdatedMsg:
		// Part received new content - append to display
		if msg.SessionID == a.state.SessionID {
			a.handleStreamPartUpdated(msg)
			cmds = append(cmds, a.waitForEvent()) // Continue waiting for events
		}

	case StreamPartCompleteMsg:
		// Part finished - mark as complete
		if msg.SessionID == a.state.SessionID {
			a.handleStreamPartComplete(msg)
			cmds = append(cmds, a.waitForEvent()) // Continue waiting for events
		}

	case StreamMessageCompleteMsg:
		// Whole message finished
		if msg.SessionID == a.state.SessionID {
			a.handleStreamMessageComplete(msg)
			cmds = append(cmds, a.waitForEvent()) // Continue waiting for events
		}

	case StreamToolPendingMsg:
		// Tool call started
		if msg.SessionID == a.state.SessionID {
a.handleStreamToolPending(msg)
			cmds = append(cmds, a.waitForEvent()) // Continue waiting for events
		}

	case StreamToolRunningMsg:
		// Tool execution started
		if msg.SessionID == a.state.SessionID {
			// Find the part for this tool and update status to running
			for _, part := range a.streamingState.parts {
				if part.ToolID == msg.ToolID {
					part.Status = "running"

					// Update in message's Parts array
					messages := a.state.Sync.Messages
					if len(messages) > 0 {
						last := &messages[len(messages)-1]
						if last.ID == a.streamingState.messageID {
							for i := range last.Parts {
								if last.Parts[i].ToolID == msg.ToolID {
									last.Parts[i].Status = "running"
								}
							}
						}
					}
					break
				}
			}

			a.state.SetStatus(fmt.Sprintf("Running: %s", msg.ToolName))
			cmds = append(cmds, a.waitForEvent())
		}

	case StreamToolCompleteMsg:
		// Tool execution finished
		if msg.SessionID == a.state.SessionID {
			a.handleStreamToolComplete(msg)
			cmds = append(cmds, a.waitForEvent()) // Continue waiting for events
		}

	case StreamErrorMsg:
		// Error during streaming
		if msg.SessionID == a.state.SessionID {
			a.setError(fmt.Errorf("%s: %s", msg.ErrorType, msg.Error))
			a.state.Processing = false
			a.mode = ModeInput
			a.state.SetStatus("Error")
			cmds = append(cmds, a.waitForEvent()) // Continue waiting for events
		}

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
			a.state.SetUserScrolled(false) // Reset scroll state
			if len(msg.Messages) > 0 {
				a.state.SetStatus(fmt.Sprintf("Loaded %d messages", len(msg.Messages)))
			}
		}

	case MessagesLoadedMsg:
		// Messages loaded from session switch
		log.Info("MessagesLoadedMsg received",
			"msgSessionID", msg.SessionID,
			"currentSessionID", a.state.SessionID,
			"messageCount", len(msg.Messages),
			"match", msg.SessionID == a.state.SessionID)

		if msg.SessionID == a.state.SessionID {
			log.Info("Setting messages in state", "count", len(msg.Messages))
			a.state.SetMessages(msg.Messages)

			content := a.buildMessagesContent()
			log.Info("Built content", "contentLen", len(content), "lines", strings.Count(content, "\n"))

			a.messageViewport.SetContent(content)
			log.Info("Set viewport content, total lines", "lines", a.messageViewport.TotalLineCount())

			a.messageViewport.GotoBottom()
			log.Info("GotoBottom called, YOffset", "offset", a.messageViewport.YOffset)

			a.state.SetUserScrolled(false)
			if len(msg.Messages) > 0 {
				a.state.SetStatus(fmt.Sprintf("Loaded %d messages", len(msg.Messages)))
			}
		} else {
			log.Warn("Session ID mismatch, not loading messages",
				"msgSessionID", msg.SessionID,
				"currentSessionID", a.state.SessionID)
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

	// Detect if user scrolled away from bottom
	a.handleViewportScroll()

	return a, tea.Batch(cmds...)
}

// ThemeChangeMsg is sent when the theme changes
type ThemeChangeMsg struct {
	ThemeID string
}

// MessagesLoadedMsg is sent when messages are loaded for a session
type MessagesLoadedMsg struct {
	SessionID string
	Messages  []Message
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
	// Debug logging
	msgCount := len(a.state.Sync.Messages)
	vpLineCount := a.messageViewport.TotalLineCount()

	if msgCount > 0 {
		log.Info("renderChat",
			"msgCount", msgCount,
			"vpLineCount", vpLineCount,
			"vpYOffset", a.messageViewport.YOffset,
			"vpHeight", a.messageViewport.Height)
	}

	// Ensure content is set (in case it wasn't set elsewhere)
	if vpLineCount == 0 && msgCount > 0 {
		log.Info("renderChat: setting content because viewport is empty but messages exist")
		a.messageViewport.SetContent(a.buildMessagesContent())
	}

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

	// Calculate available width for text wrapping
	// Subtract padding, timestamps, and other formatting
	availableWidth := a.state.Layout.Width - 8
	if availableWidth < 40 {
		availableWidth = 40 // Minimum width
	}

	for _, msg := range a.state.Sync.Messages {
		switch msg.Role {
		case RoleUser:
			timeStr := msg.Timestamp.Format("15:04")
			prefix := fmt.Sprintf("[%s] You: ", timeStr)
			prefixWidth := util.StringWidth(prefix)
			contentWidth := availableWidth - prefixWidth
			if contentWidth < 20 {
				contentWidth = 20
			}

			// Wrap the content
			wrappedContent := util.WrapPreserveNewlines(msg.Content, contentWidth)
			for i, line := range wrappedContent {
				if i == 0 {
					lines = append(lines, a.styles.UserMessage.Render(prefix+line))
				} else {
					// Continuation lines get padding to align with content
					padding := strings.Repeat(" ", prefixWidth)
					lines = append(lines, a.styles.UserMessage.Render(padding+line))
				}
			}

		case RoleAssistant:
			timeStr := msg.Timestamp.Format("15:04")
			if len(msg.Parts) > 0 {
				partLines := a.renderParts(msg.Parts, msg.Model, availableWidth)
				lines = append(lines, partLines)
			} else if msg.Content != "" {
				var prefix string
				if msg.Model != "" {
					prefix = fmt.Sprintf("[%s] Assistant (%s): ", timeStr, msg.Model)
				} else {
					prefix = fmt.Sprintf("[%s] Assistant: ", timeStr)
				}
				prefixWidth := util.StringWidth(prefix)
				contentWidth := availableWidth - prefixWidth
				if contentWidth < 20 {
					contentWidth = 20
				}

				// Wrap the content
				wrappedContent := util.WrapPreserveNewlines(msg.Content, contentWidth)
				for i, line := range wrappedContent {
					if i == 0 {
						lines = append(lines, a.styles.AssistantMessage.Render(prefix+line))
					} else {
						// Continuation lines get padding to align with content
						padding := strings.Repeat(" ", prefixWidth)
						lines = append(lines, a.styles.AssistantMessage.Render(padding+line))
					}
				}
			}

		case RoleSystem:
			// System messages are typically short, but wrap just in case
			wrappedContent := util.WrapPreserveNewlines(msg.Content, availableWidth)
			for _, line := range wrappedContent {
				lines = append(lines, a.styles.SystemMessage.Render(line))
			}

		case RoleTool:
			toolLine := a.renderToolCall(msg.ToolCall)
			lines = append(lines, toolLine)
		}
	}

	if len(lines) == 0 {
		return a.styles.TextMuted.Render("No messages. Start a conversation!")
	}

	return strings.Join(lines, "\n")
}

// renderParts renders assistant message parts
func (a *App) renderParts(parts []Part, model string, width int) string {
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
				// Wrap text content
				wrappedContent := util.WrapPreserveNewlines(part.Text, width)
				for _, line := range wrappedContent {
					lines = append(lines, a.styles.Text.Render(line))
				}
			}

		case "tool_use":
			// Use the new tool renderer for tool calls
			toolCall := toolRenderer.RenderToolCall(part.ToolName, part.ToolInput)
			lines = append(lines, toolCall)

		case "tool_result":
			// Use the new tool renderer for tool results
			toolResult := toolRenderer.RenderResult(part.ToolName, part.ToolInput, part.ToolResult, part.Status)
			lines = append(lines, toolResult)

		case "reasoning":
			// OpenCode uses "reasoning" as the part type for thinking blocks
			if part.Text != "" {
				// Wrap reasoning content
				wrappedContent := util.WrapPreserveNewlines(part.Text, width)
				for _, line := range wrappedContent {
					lines = append(lines, a.styles.Thinking.Render(fmt.Sprintf("💭 %s", line)))
				}
			}

		case "thinking":
			// Also handle "thinking" for backwards compatibility
			if part.Text != "" {
				wrappedContent := util.WrapPreserveNewlines(part.Text, width)
				for _, line := range wrappedContent {
					lines = append(lines, a.styles.Thinking.Render(fmt.Sprintf("💭 %s", line)))
				}
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
				log.Info("handleDialogSelection: switching session",
					"sessionID", session.ID,
					"title", session.Title,
					"currentSessionID", a.state.SessionID)

				// Clear viewport content immediately so user doesn't see stale messages
				a.messageViewport.SetContent("")
				log.Info("handleDialogSelection: cleared viewport content")

				a.state.SetActiveSession(&session)
				log.Info("handleDialogSelection: SetActiveSession called, new SessionID", "sessionID", a.state.SessionID)

				a.activeDialog = nil
				a.state.PopDialog()
				log.Info("handleDialogSelection: dialog closed, loading messages...")

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
		// Mark that user has scrolled
		a.state.Layout.UserScrolled = true
		// Check if at top and need to load more
		if a.atTopOfMessages() && a.state.HistoryMore(a.state.SessionID) && !a.state.HistoryLoading(a.state.SessionID) {
			return a, a.loadMoreMessages()
		}
		return a, nil

	case kb.Down.Match(msg):
		a.messageViewport.LineDown(1)
		// Check if scrolled back to bottom
		if a.isAtBottom() {
			a.state.Layout.UserScrolled = false
		}
		return a, nil

	case kb.PageUp.Match(msg):
		a.messageViewport.HalfViewUp()
		// Mark that user has scrolled
		a.state.Layout.UserScrolled = true
		// Check if at top and need to load more
		if a.atTopOfMessages() && a.state.HistoryMore(a.state.SessionID) && !a.state.HistoryLoading(a.state.SessionID) {
			return a, a.loadMoreMessages()
		}
		return a, nil

	case kb.PageDown.Match(msg):
		a.messageViewport.HalfViewDown()
		// Check if scrolled back to bottom
		if a.isAtBottom() {
			a.state.Layout.UserScrolled = false
		}
		return a, nil

	// New navigation keybindings
	case kb.HalfPageUp.Match(msg):
		// Scroll half page up
		a.messageViewport.HalfViewUp()
		a.state.Layout.UserScrolled = true
		if a.atTopOfMessages() && a.state.HistoryMore(a.state.SessionID) && !a.state.HistoryLoading(a.state.SessionID) {
			return a, a.loadMoreMessages()
		}
		return a, nil

	case kb.HalfPageDown.Match(msg):
		// Scroll half page down
		a.messageViewport.HalfViewDown()
		if a.isAtBottom() {
			a.state.Layout.UserScrolled = false
		}
		return a, nil

	case kb.FirstMessage.Match(msg):
		// Jump to first message (Home)
		a.messageViewport.GotoTop()
		a.state.Layout.UserScrolled = true
		// Check if we need to load more history
		if a.state.HistoryMore(a.state.SessionID) && !a.state.HistoryLoading(a.state.SessionID) {
			return a, a.loadMoreMessages()
		}
		return a, nil

	case kb.LastMessage.Match(msg):
		// Jump to last message (End)
		a.messageViewport.GotoBottom()
		a.state.Layout.UserScrolled = false
		return a, nil

	// Also handle Ctrl+G for first message (OpenCode style)
	case msg.String() == "ctrl+g":
		a.messageViewport.GotoTop()
		a.state.Layout.UserScrolled = true
		if a.state.HistoryMore(a.state.SessionID) && !a.state.HistoryLoading(a.state.SessionID) {
			return a, a.loadMoreMessages()
		}
		return a, nil

	// Also handle Ctrl+Alt+G for last message (OpenCode style)
	case msg.String() == "ctrl+alt+g":
		a.messageViewport.GotoBottom()
		a.state.Layout.UserScrolled = false
		return a, nil

	// Also handle Ctrl+Alt+B for page up (OpenCode style)
	case msg.String() == "ctrl+alt+b":
		a.messageViewport.HalfViewUp()
		a.state.Layout.UserScrolled = true
		if a.atTopOfMessages() && a.state.HistoryMore(a.state.SessionID) && !a.state.HistoryLoading(a.state.SessionID) {
			return a, a.loadMoreMessages()
		}
		return a, nil

	// Also handle Ctrl+Alt+F for page down (OpenCode style)
	case msg.String() == "ctrl+alt+f":
		a.messageViewport.HalfViewDown()
		if a.isAtBottom() {
			a.state.Layout.UserScrolled = false
		}
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
		// Handle Tab key for autocomplete
		if msg.Type == tea.KeyTab {
			// Check if autocomplete is visible
			if a.prompt.IsAutocompleteVisible() {
				// Select current option
				if option, ok := a.prompt.GetAutocomplete().Select(); ok {
					a.prompt.InsertCompletion(option.Value)
					a.prompt.HideAutocomplete()
				}
			} else {
				// Trigger autocomplete
				a.triggerAutocomplete()
			}
			return a, nil
		}

		// Handle Shift+Tab for navigating autocomplete
		if msg.Type == tea.KeyShiftTab {
			if a.prompt.IsAutocompleteVisible() {
				a.prompt.GetAutocomplete().Previous()
				return a, nil
			}
		}

		// Handle Escape to hide autocomplete
		if msg.Type == tea.KeyEsc {
			if a.prompt.IsAutocompleteVisible() {
				a.prompt.HideAutocomplete()
				return a, nil
			}
		}

		// Only pass printable characters and essential editing keys to input
		switch msg.Type {
		case tea.KeyRunes, tea.KeySpace, tea.KeyBackspace, tea.KeyDelete, tea.KeyLeft, tea.KeyRight:
			// Update the prompt's internal input by passing the message to it
			_, cmd := a.prompt.Update(msg)
			// Check if we should trigger autocomplete after input update
			a.checkAutocomplete()
			return a, cmd
		}
		// Also handle keys with runes
		if msg.Runes != nil && len(msg.Runes) > 0 {
			_, cmd := a.prompt.Update(msg)
			// Check if we should trigger autocomplete after input update
			a.checkAutocomplete()
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

// sendMessage sends a message using the AI processor
func (a *App) sendMessage(content string) tea.Cmd {
	return func() tea.Msg {
		// Check if processor is available
		if a.processor == nil {
			// Fallback to placeholder if processor not configured
			return ResponseMsg{
				Content: "AI processing not configured. Please initialize the processor.",
			}
		}

		// Get current model from state
		modelKey := a.state.Local.CurrentModel
		if modelKey.ProviderID == "" {
			// Default model
			modelKey = types.ModelKey{
				ProviderID: "anthropic",
				ModelID:    "claude-sonnet-4-5",
			}
		}

		// Build model ID string
		fullModelID := provider.FormatModelID(provider.ProviderID(modelKey.ProviderID), modelKey.ModelID)

		// Get system prompt from current agent
		var systemPrompt string
		if agent := a.state.GetCurrentAgent(); agent != nil {
			systemPrompt = agent.Content
		}

		// Get tool definitions from processor
		var tools []provider.ToolDefinition
		if a.toolRegistry != nil {
			tools = a.processor.GetToolDefinitions()
		}

		// Build process request
		req := session.ProcessRequest{
			SessionID:    a.state.SessionID,
			UserMessage:  content,
			Model:        fullModelID,
			SystemPrompt: systemPrompt,
			Tools:        tools,
			Agent:        a.state.Local.CurrentAgent,
		}

		// Start processing asynchronously
		ctx := context.Background()
		err := a.processor.Process(ctx, req)
		if err != nil {
			return ResponseMsg{
				Content: fmt.Sprintf("Error: %v", err),
				Error:   err,
			}
		}

		// Return empty response - actual content will come via bus events
		// For now, we show a message indicating streaming is happening
		return ResponseMsg{
			Content: "Processing... (streaming response)",
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
				// Scroll to bottom when switching sessions
				a.scrollToBottom(true)
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
		log.Info("loadMessagesForSession: START", "sessionID", sessionID, "databasePath", a.databasePath)

		if sessionID == "" || a.databasePath == "" {
			log.Warn("loadMessagesForSession: missing sessionID or databasePath")
			return nil
		}

		ctx := context.Background()
		db, err := database.New(ctx, database.Config{Path: a.databasePath})
		if err != nil {
			log.Error("Failed to open database for loading messages", "error", err.Error())
			return ErrorMsg{Error: fmt.Errorf("failed to open database: %w", err)}
		}
		defer db.Close()

		messageStorage := database.NewMessageStorage(db)
		messages, _, _, err := messageStorage.ListPaginated(ctx, sessionID, 80, 0)
		if err != nil {
			log.Error("Failed to load messages from database", "error", err.Error())
			return ErrorMsg{Error: fmt.Errorf("failed to load messages: %w", err)}
		}

		log.Info("loadMessagesForSession: loaded from DB", "sessionID", sessionID, "count", len(messages))

		// Convert database messages to UI messages
		uiMessages := make([]Message, 0, len(messages))
		for i, dbMsg := range messages {
			uiMsg := convertDBMessageToTUI(dbMsg)
			uiMessages = append(uiMessages, uiMsg)
			if i < 3 { // Log first 3 messages for debugging
				log.Info("loadMessagesForSession: message",
					"index", i,
					"role", uiMsg.Role,
					"contentPreview", uiMsg.Content[:min(50, len(uiMsg.Content))])
			}
		}

		log.Info("loadMessagesForSession: returning MessagesLoadedMsg", "sessionID", sessionID, "uiMessageCount", len(uiMessages))

		// Return a message to trigger update
		return MessagesLoadedMsg{
			SessionID: sessionID,
			Messages:  uiMessages,
		}
	}
}

// addMessage adds a message to the state
func (a *App) addMessage(msg Message) {
	msg.ID = fmt.Sprintf("msg-%d", time.Now().UnixNano())
	a.state.AddMessage(msg)
	a.messageViewport.SetContent(a.buildMessagesContent())
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
			a.messageViewport.GotoBottom()
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

// ===========================================
// Streaming Event Handlers
// ===========================================

// handleStreamPartCreated handles when a new part is created during streaming
func (a *App) handleStreamPartCreated(msg StreamPartCreatedMsg) {
	// Initialize streaming state for new message if needed
	if !a.streamingState.isActive || a.streamingState.messageID != msg.MessageID {
		a.streamingState = streamingState{
			sessionID: msg.SessionID,
			messageID: msg.MessageID,
			parts:     make(map[int]*Part),
			isActive:  true,
		}

		// Create a new assistant message placeholder
		a.addMessage(Message{
			ID:        msg.MessageID,
			Role:      RoleAssistant,
			Timestamp: time.Now(),
			Parts:     []Part{},
			Model:     a.state.Local.CurrentModel.ModelID,
			Provider:  a.state.Local.CurrentModel.ProviderID,
		})
	}

	// Create the new part
	newPart := &Part{
		ID:    msg.PartID,
		Index: msg.Index,
		Type:  msg.Type,
		Text:  "",
		Status: "streaming",
	}

	// Track it in streaming state
	a.streamingState.parts[msg.Index] = newPart

	// Add it to the message's Parts array
	messages := a.state.Sync.Messages
	if len(messages) > 0 {
		last := &messages[len(messages)-1]
		if last.ID == msg.MessageID {
			// Ensure Parts array has space for this index
			if len(last.Parts) <= msg.Index {
				// Extend array
				extended := make([]Part, msg.Index+1)
				copy(extended, last.Parts)
				last.Parts = extended
			}
			last.Parts[msg.Index] = *newPart
		}
	}

	// Update viewport
	a.messageViewport.SetContent(a.buildMessagesContent())
	a.messageViewport.GotoBottom()
}

// handleStreamPartUpdated handles when a part receives new content
func (a *App) handleStreamPartUpdated(msg StreamPartUpdatedMsg) {
	// Find the part in streaming state
	part, exists := a.streamingState.parts[msg.Index]
	if !exists {
		return
	}

	// Append delta based on type
	switch msg.DeltaType {
	case "text":
		part.Text += msg.Delta
	case "reasoning":
		part.Text += msg.Delta
	case "tool_input":
		// Tool input is being streamed (JSON fragments)
		part.ToolInput += msg.Delta
	}

	// Update the message's Parts array
	messages := a.state.Sync.Messages
	if len(messages) > 0 {
		last := &messages[len(messages)-1]
		if last.ID == a.streamingState.messageID && msg.Index < len(last.Parts) {
			// Copy updated part to message
			last.Parts[msg.Index] = *part
		}
	}

	// Update viewport to show streaming content
	a.messageViewport.SetContent(a.buildMessagesContent())

	// Auto-scroll if not at bottom (only if user hasn't scrolled away)
	if !a.state.Layout.UserScrolled {
		a.messageViewport.GotoBottom()
	}
}

// handleStreamPartComplete handles when a part is finished
func (a *App) handleStreamPartComplete(msg StreamPartCompleteMsg) {
	// Find the part and mark as complete
	part, exists := a.streamingState.parts[msg.Index]
	if exists {
		part.Status = "complete"

		// Update in message's Parts array
		messages := a.state.Sync.Messages
		if len(messages) > 0 {
			last := &messages[len(messages)-1]
			if last.ID == a.streamingState.messageID && msg.Index < len(last.Parts) {
				last.Parts[msg.Index].Status = "complete"
			}
		}
	}

	// Update viewport
	a.messageViewport.SetContent(a.buildMessagesContent())
}

// handleStreamMessageComplete handles when the whole message is finished
func (a *App) handleStreamMessageComplete(msg StreamMessageCompleteMsg) {
	// Clear streaming state
	a.streamingState.isActive = false
	a.streamingState.parts = make(map[int]*Part)

	// Mark processing as done
	a.state.Processing = false
	a.mode = ModeInput
	a.state.SetStatus("Ready")

	// Final viewport update
	a.messageViewport.SetContent(a.buildMessagesContent())
	a.messageViewport.GotoBottom()
}

// handleStreamToolPending handles when a tool call starts
func (a *App) handleStreamToolPending(msg StreamToolPendingMsg) {
	// Find the part for this tool
	part, exists := a.streamingState.parts[msg.PartIndex]
	if exists {
		// Update tool part with tool details
		part.Type = "tool_use"
		part.ToolID = msg.ToolID
		part.ToolName = msg.ToolName
		part.Status = "pending"
		if msg.Input != "" {
			part.ToolInput = msg.Input
		}

		// Update in message's Parts array
		messages := a.state.Sync.Messages
		if len(messages) > 0 {
			last := &messages[len(messages)-1]
			if last.ID == a.streamingState.messageID && msg.PartIndex < len(last.Parts) {
				last.Parts[msg.PartIndex] = *part
			}
		}
	}

	// Update status and viewport
	a.state.SetStatus(fmt.Sprintf("Tool: %s...", msg.ToolName))
	a.messageViewport.SetContent(a.buildMessagesContent())
}

// handleStreamToolComplete handles when tool execution finishes
func (a *App) handleStreamToolComplete(msg StreamToolCompleteMsg) {
	// Find the tool_use part and create corresponding tool_result
	for idx, part := range a.streamingState.parts {
		if part.ToolID == msg.ToolID {
			// Mark tool_use as complete
			part.Status = "complete"

			// Update in message's Parts array
			messages := a.state.Sync.Messages
			if len(messages) > 0 {
				last := &messages[len(messages)-1]
				if last.ID == a.streamingState.messageID {
					// Update tool_use part
					if idx < len(last.Parts) {
						last.Parts[idx].Status = "complete"
						if msg.IsError {
							last.Parts[idx].Status = "error"
						}
					}

					// Add tool_result part after tool_use
					resultPart := Part{
						ID:         uuid.New().String(),
						Index:      idx + 1,
						Type:       "tool_result",
						ToolID:     msg.ToolID,
						ToolName:   msg.ToolName,
						ToolResult: msg.Result,
						Status:     "complete",
					}
					if msg.IsError {
						resultPart.Status = "error"
					}

					// Insert after tool_use
					if idx+1 < len(last.Parts) {
						last.Parts[idx+1] = resultPart
					} else {
						last.Parts = append(last.Parts, resultPart)
					}

					// Also track in streaming state
					a.streamingState.parts[idx+1] = &resultPart
				}
			}
			break
		}
	}

	// Update viewport
	a.messageViewport.SetContent(a.buildMessagesContent())

	if msg.IsError {
		a.state.SetStatus(fmt.Sprintf("Tool error: %s", msg.ToolName))
	} else {
		a.state.SetStatus(fmt.Sprintf("Tool done: %s", msg.ToolName))
	}
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

// scrollToBottom scrolls the viewport to the bottom
// force: if true, scroll regardless of userScrolled state
func (a *App) scrollToBottom(force bool) {
	// Don't auto-scroll if user has scrolled up (unless forced)
	if !force && a.state.IsUserScrolled() {
		return
	}

	// Get content height (total lines)
	content := a.buildMessagesContent()
	lines := strings.Split(content, "\n")
	contentHeight := len(lines)

	// Calculate if we need to scroll
	viewportHeight := a.messageViewport.Height
	if contentHeight > viewportHeight {
		// Scroll to show the last lines
		offset := contentHeight - viewportHeight
		if offset < 0 {
			offset = 0
		}
		a.messageViewport.SetYOffset(offset)
		a.state.SetUserScrolled(false)
	}
}

// handleViewportScroll detects if user has scrolled away from bottom
// Call this whenever the viewport is scrolled
func (a *App) handleViewportScroll() {
	content := a.buildMessagesContent()
	lines := strings.Split(content, "\n")
	contentHeight := len(lines)
	viewportHeight := a.messageViewport.Height
	currentOffset := a.messageViewport.YOffset

	// Check if there's content to scroll
	if contentHeight <= viewportHeight {
		a.state.SetUserScrolled(false)
		return
	}

	// Calculate distance from bottom
	maxOffset := contentHeight - viewportHeight
	distanceFromBottom := maxOffset - currentOffset

	// If user scrolled up (more than 2 lines from bottom), mark as userScrolled
	if distanceFromBottom > 2 {
		a.state.SetUserScrolled(true)
	} else {
		a.state.SetUserScrolled(false)
	}
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

// isAtBottom returns true if viewport is at the bottom of messages
func (a *App) isAtBottom() bool {
	contentHeight := a.messageViewport.TotalLineCount()
	viewportHeight := a.messageViewport.Height
	currentOffset := a.messageViewport.YOffset

	// Calculate distance from bottom
	distanceFromBottom := contentHeight - viewportHeight - currentOffset

	// Consider "at bottom" if within 2 lines of the bottom
	return distanceFromBottom <= 2
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
// OpenCode part types: text, reasoning, file, tool_use, tool_result
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

		// Handle specific part types
		if p.Data.Type == "file" {
			tuiPart.Text = fmt.Sprintf("[File: %s]", p.Data.Filename)
		}

		// Reasoning/thinking parts - text is copied directly
		// OpenCode uses "reasoning" type for thinking blocks
		if p.Data.Type == "reasoning" || p.Data.Type == "thinking" {
			// Text field already contains the reasoning content
		}

		if p.Data.Type == "tool_use" || p.Data.Type == "tool_result" {
			tuiPart.ToolName = p.Data.ToolName
			tuiPart.ToolID = p.Data.ToolID
			// Convert tool input to string if it's a map
			if p.Data.ToolInput != nil {
				tuiPart.ToolInput = fmt.Sprintf("%v", p.Data.ToolInput)
			}
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
