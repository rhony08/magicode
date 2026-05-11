// Package session provides message processing for AI conversations.
package session

import (
	"context"
	"fmt"
	"sync"

	"github.com/rhony08/magicode/internal/bus"
	"github.com/rhony08/magicode/internal/database"
	"github.com/rhony08/magicode/internal/provider"
	"github.com/rhony08/magicode/internal/tool"
	"github.com/rhony08/magicode/internal/util/log"
)

// Event definitions for the session processor
var (
	// EventPartCreated is emitted when a new part is created
	EventPartCreated = bus.Definition{Type: "session.part.created"}

	// EventPartUpdated is emitted when a part is updated with new content
	EventPartUpdated = bus.Definition{Type: "session.part.updated"}

	// EventPartComplete is emitted when a part is finished
	EventPartComplete = bus.Definition{Type: "session.part.complete"}

	// EventMessageCreated is emitted when a new message is created
	EventMessageCreated = bus.Definition{Type: "session.message.created"}

	// EventMessageComplete is emitted when a message is finished
	EventMessageComplete = bus.Definition{Type: "session.message.complete"}

	// EventToolCallPending is emitted when a tool call is received
	EventToolCallPending = bus.Definition{Type: "session.tool.pending"}

	// EventToolCallRunning is emitted when a tool starts executing
	EventToolCallRunning = bus.Definition{Type: "session.tool.running"}

	// EventToolCallComplete is emitted when a tool execution finishes
	EventToolCallComplete = bus.Definition{Type: "session.tool.complete"}

	// EventStreamError is emitted when an error occurs during streaming
	EventStreamError = bus.Definition{Type: "session.stream.error"}
)

// StreamResult tracks results from streaming for multi-turn support
type StreamResult struct {
	StopReason   string        // "end_turn", "tool_use", "max_tokens", etc.
	ToolResults  []ToolResult  // Tool execution results
	IsError      bool          // Whether streaming ended with error
	Error        error         // Error if any
}

// ToolResult tracks a single tool execution result
type ToolResult struct {
	ToolID   string
	ToolName string
	Result   string
	IsError  bool
}

// Processor handles AI message processing with streaming support
type Processor struct {
	registry     *provider.ProviderRegistry
	db           *database.Database
	bus          *bus.Service
	messages     *database.MessageStorage
	parts        *database.PartStorage
	toolRegistry *tool.Registry // Tool execution registry
	mu           sync.Mutex
	active       map[string]context.CancelFunc // Active processing contexts by session ID
	logger       *log.Logger
}

// ProcessorConfig contains configuration for the processor
type ProcessorConfig struct {
	Registry     *provider.ProviderRegistry
	DB           *database.Database
	Bus          *bus.Service
	ToolRegistry *tool.Registry // Tool registry for execution
	Logger       *log.Logger
}

// NewProcessor creates a new session processor
func NewProcessor(config ProcessorConfig) *Processor {
	if config.Logger == nil {
		config.Logger = log.Create(map[string]string{"service": "session.processor"})
	}

	if config.ToolRegistry == nil {
		config.ToolRegistry = tool.NewRegistry()
	}

	return &Processor{
		registry:     config.Registry,
		db:           config.DB,
		bus:          config.Bus,
		messages:     database.NewMessageStorage(config.DB),
		parts:        database.NewPartStorage(config.DB),
		toolRegistry: config.ToolRegistry,
		active:       make(map[string]context.CancelFunc),
		logger:       config.Logger,
	}
}

// ProcessRequest contains all info for processing a message
type ProcessRequest struct {
	SessionID    string
	UserMessage  string // The user's input text
	Model        provider.ModelID
	SystemPrompt string
	History      []database.Message // Previous messages for context
	Tools        []provider.ToolDefinition
	Agent        string // Agent name (optional)
}

// Process streams response from AI, executes tools, stores results
// This is the main entry point for processing a user message
// Supports multi-turn tool loops - continues streaming after tool execution
func (p *Processor) Process(ctx context.Context, req ProcessRequest) error {
	p.logger.Info("Processing message", map[string]interface{}{
		"session_id": req.SessionID,
		"model":      req.Model,
	})

	// Cancel any existing processing for this session
	p.cancelExisting(req.SessionID)

	// Create cancellable context for this processing
	processCtx, cancel := context.WithCancel(ctx)
	p.mu.Lock()
	p.active[req.SessionID] = cancel
	p.mu.Unlock()

	defer func() {
		p.mu.Lock()
		delete(p.active, req.SessionID)
		p.mu.Unlock()
		cancel()
	}()

	// 1. Create user message in database
	userMsg, err := p.createUserMessage(processCtx, req)
	if err != nil {
		return fmt.Errorf("failed to create user message: %w", err)
	}

	// Publish user message created event
	if p.bus != nil {
		p.bus.Publish(EventMessageCreated, map[string]interface{}{
			"session_id": req.SessionID,
			"message_id": userMsg.ID,
			"role":       "user",
		})
	}

	// 2. Get provider from registry
	providerID, _ := provider.ParseModelID(req.Model)
	prov, ok := p.registry.Get(providerID)
	if !ok {
		return fmt.Errorf("provider not found: %s", providerID)
	}

	// 3. Build initial messages list (user message + history)
	contentMessages := p.buildContentMessagesForProvider(req.History, userMsg, req.UserMessage)

	// 4. Create initial assistant message in DB
	assistantMsg, err := p.createAssistantMessage(processCtx, req)
	if err != nil {
		return fmt.Errorf("failed to create assistant message: %w", err)
	}

	// Publish assistant message created event
	if p.bus != nil {
		p.bus.Publish(EventMessageCreated, map[string]interface{}{
			"session_id": req.SessionID,
			"message_id": assistantMsg.ID,
			"role":       "assistant",
		})
	}

	// 5. Multi-turn loop: stream, execute tools, continue if needed
	currentMessageID := assistantMsg.ID
	for {
		// Build chat request for this turn
		chatReq := p.buildChatRequestWithContentMessages(req, contentMessages)

		// Start streaming
		events, err := prov.StreamChat(processCtx, chatReq)
		if err != nil {
			p.logger.Error("StreamChat failed", map[string]interface{}{
				"error": err.Error(),
			})
			if p.bus != nil {
				p.bus.Publish(EventStreamError, map[string]interface{}{
					"session_id": req.SessionID,
					"error":      err.Error(),
				})
			}
			return fmt.Errorf("stream failed: %w", err)
		}

		// Process stream events and collect results
		result := p.processStreamEventsWithResult(processCtx, req.SessionID, currentMessageID, events)

		// Check if we need to continue with tool results
		if result.StopReason == "tool_use" && len(result.ToolResults) > 0 {
			p.logger.Info("Continuing with tool results", map[string]interface{}{
				"session_id":   req.SessionID,
				"tool_count":   len(result.ToolResults),
				"stop_reason":  result.StopReason,
			})

			// Build assistant message content from parts
			assistantContent := p.buildAssistantContent(req.SessionID, currentMessageID)

			// Add assistant message to conversation
			contentMessages = append(contentMessages, provider.ContentMessage{
				Role:    provider.RoleAssistant,
				Content: assistantContent,
			})

			// Add tool result messages
			for _, tr := range result.ToolResults {
				contentMessages = append(contentMessages, provider.ContentMessage{
					Role: provider.RoleUser,
					Content: []provider.ContentPart{
						provider.ToolResultPart{
							Type:      "tool_result",
							ToolUseID: tr.ToolID,
							Content:   tr.Result,
							IsError:   tr.IsError,
						},
					},
				})
			}

			// Create new assistant message for next turn
			newAssistantMsg, err := p.createAssistantMessage(processCtx, req)
			if err != nil {
				return fmt.Errorf("failed to create assistant message for tool loop: %w", err)
			}
			currentMessageID = newAssistantMsg.ID

			// Publish new assistant message created
			if p.bus != nil {
				p.bus.Publish(EventMessageCreated, map[string]interface{}{
					"session_id": req.SessionID,
					"message_id": currentMessageID,
					"role":       "assistant",
					"turn":       "tool_loop",
				})
			}

			// Continue loop - will make new streaming request with tool results
			continue
		}

		// No more tool calls - processing complete
		break
	}

	return nil
}

// cancelExisting cancels any active processing for a session
func (p *Processor) cancelExisting(sessionID string) {
	p.mu.Lock()
	cancel, exists := p.active[sessionID]
	p.mu.Unlock()

	if exists {
		p.logger.Info("Cancelling existing processing", map[string]interface{}{
			"session_id": sessionID,
		})
		cancel()
	}
}

// createUserMessage creates a user message in the database
func (p *Processor) createUserMessage(ctx context.Context, req ProcessRequest) (*database.Message, error) {
	msg := database.Message{
		SessionID: req.SessionID,
		Data: database.MessageInfo{
			Role:    "user",
			Agent:   req.Agent,
			ModelID: string(req.Model),
		},
	}

	// Create text part for user message
	part := database.Part{
		MessageID: msg.ID,
		SessionID: req.SessionID,
		Data: database.PartData{
			Type: "text",
			Text: req.UserMessage,
		},
	}

	// Create message first
	createdMsg, err := p.messages.Create(ctx, msg)
	if err != nil {
		return nil, err
	}

	// Create part
	part.MessageID = createdMsg.ID
	_, err = p.parts.Create(ctx, part)
	if err != nil {
		return nil, err
	}

	return createdMsg, nil
}

// createAssistantMessage creates an assistant message placeholder
func (p *Processor) createAssistantMessage(ctx context.Context, req ProcessRequest) (*database.Message, error) {
	msg := database.Message{
		SessionID: req.SessionID,
		Data: database.MessageInfo{
			Role:       "assistant",
			Agent:      req.Agent,
			ModelID:    string(req.Model),
			ProviderID: string(provider.ProviderID("")),
		},
	}

	// Set provider ID from model
	providerID, _ := provider.ParseModelID(req.Model)
	msg.Data.ProviderID = string(providerID)

	return p.messages.Create(ctx, msg)
}

// buildChatRequest builds a provider chat request from the process request
func (p *Processor) buildChatRequest(req ProcessRequest) provider.ChatRequest {
	// Convert history messages to provider format
	providerMessages := make([]provider.Message, 0, len(req.History)+1)

	// Add history messages
	for _, msg := range req.History {
		// Get parts for this message
		parts, _ := p.parts.ListByMessage(context.Background(), msg.ID)
		content := p.buildContentFromParts(parts)

		role := provider.Role(msg.Data.Role)
		providerMessages = append(providerMessages, provider.Message{
			Role:    role,
			Content: content,
		})
	}

	// Add current user message
	providerMessages = append(providerMessages, provider.Message{
		Role:    provider.RoleUser,
		Content: req.UserMessage,
	})

	return provider.ChatRequest{
		Model:    req.Model,
		Messages: providerMessages,
		System:   req.SystemPrompt,
		Tools:    req.Tools,
		Stream:   true,
	}
}

// buildContentFromParts builds a content string from parts
func (p *Processor) buildContentFromParts(parts []database.Part) string {
	var content string
	for _, part := range parts {
		switch part.Data.Type {
		case "text", "reasoning":
			content += part.Data.Text
		case "tool_result":
			content += part.Data.ToolResult
		}
	}
	return content
}

// processStreamEventsWithResult handles streaming events and returns results for multi-turn support
func (p *Processor) processStreamEventsWithResult(ctx context.Context, sessionID, messageID string, events <-chan provider.StreamEvent) StreamResult {
	result := StreamResult{
		StopReason:  "end_turn", // Default
		ToolResults: []ToolResult{},
	}

	// Track active parts by index
	activeParts := make(map[int]*database.Part)
	// Track tool_use parts that need execution
	toolUseParts := make(map[int]*database.Part)
	var mu sync.Mutex

	for event := range events {
		// Check if context is cancelled
		if ctx.Err() != nil {
			p.logger.Info("Processing cancelled", map[string]interface{}{
				"session_id": sessionID,
			})
			result.IsError = true
			result.Error = ctx.Err()
			return result
		}

		switch e := event.(type) {
		case provider.MessageStartEvent:
			// Message started - nothing to do, already created

		case provider.ContentBlockStartEvent:
			mu.Lock()
			part := p.createPartFromContentBlock(sessionID, messageID, e)
			activeParts[e.Index] = part

			// Track tool_use parts for later execution
			if part.Data.Type == "tool_use" {
				toolUseParts[e.Index] = part
			}
			mu.Unlock()

			// Publish part created event
			if p.bus != nil {
				p.bus.Publish(EventPartCreated, map[string]interface{}{
					"session_id": sessionID,
					"message_id": messageID,
					"part_id":    part.ID,
					"index":      e.Index,
					"type":       part.Data.Type,
				})
			}

		case provider.ContentBlockDeltaEvent:
			mu.Lock()
			part, exists := activeParts[e.Index]
			mu.Unlock()

			if exists {
				// Update part with delta
				p.updatePartWithDelta(ctx, part, e)

				// Publish part updated event
				if p.bus != nil {
					deltaText := ""
					if textPart, ok := e.Delta.(provider.TextPart); ok {
						deltaText = textPart.Text
					}
					p.bus.Publish(EventPartUpdated, map[string]interface{}{
						"session_id": sessionID,
						"message_id": messageID,
						"part_id":    part.ID,
						"index":      e.Index,
						"delta":      deltaText,
						"delta_type": e.Delta.ContentType(),
					})
				}
			}

		case provider.ContentBlockStopEvent:
			mu.Lock()
			part, exists := activeParts[e.Index]
			delete(activeParts, e.Index)
			mu.Unlock()

			if exists {
				// Publish part complete event
				if p.bus != nil {
					p.bus.Publish(EventPartComplete, map[string]interface{}{
						"session_id": sessionID,
						"message_id": messageID,
						"part_id":    part.ID,
						"index":      e.Index,
						"type":       part.Data.Type,
					})
				}

				// If this is a tool_use part, execute the tool
				if part.Data.Type == "tool_use" {
					toolResult := p.executeToolFromPart(ctx, sessionID, messageID, part)
					result.ToolResults = append(result.ToolResults, toolResult)
				}
			}

		case provider.MessageDeltaEvent:
			// Track stop reason for multi-turn support
			result.StopReason = e.Delta.StopReason
			p.logger.Info("Message delta received", map[string]interface{}{
				"stop_reason": e.Delta.StopReason,
				"usage":       e.Usage,
			})

		case provider.MessageStopEvent:
			// Mark message complete
			if p.bus != nil {
				p.bus.Publish(EventMessageComplete, map[string]interface{}{
					"session_id": sessionID,
					"message_id": messageID,
					"stop_reason": result.StopReason,
				})
			}

		case provider.ErrorEvent:
			p.logger.Error("Stream error", map[string]interface{}{
				"error": e.Error.Message,
			})
			result.IsError = true
			result.Error = fmt.Errorf("%s: %s", e.Error.Type, e.Error.Message)
			if p.bus != nil {
				p.bus.Publish(EventStreamError, map[string]interface{}{
					"session_id": sessionID,
					"message_id": messageID,
					"error":      e.Error.Message,
					"error_type": e.Error.Type,
				})
			}
		}
	}

	return result
}

// createPartFromContentBlock creates a Part from a ContentBlockStartEvent
func (p *Processor) createPartFromContentBlock(sessionID, messageID string, event provider.ContentBlockStartEvent) *database.Part {
	part := database.Part{
		MessageID: messageID,
		SessionID: sessionID,
		Data: database.PartData{
			Type: event.ContentBlock.ContentType(),
		},
	}

	switch content := event.ContentBlock.(type) {
	case provider.TextPart:
		part.Data.Type = "text"
		part.Data.Text = content.Text

	case provider.ToolUsePart:
		part.Data.Type = "tool_use"
		part.Data.ToolID = content.ID
		part.Data.ToolName = content.Name
		part.Data.ToolInput = content.Input
		part.Data.Status = "pending"
	}

	// Create in database
	created, err := p.parts.Create(context.Background(), part)
	if err != nil {
		p.logger.Error("Failed to create part", map[string]interface{}{
			"error": err.Error(),
		})
		return &part
	}

	return created
}

// updatePartWithDelta updates a part with delta content
func (p *Processor) updatePartWithDelta(ctx context.Context, part *database.Part, event provider.ContentBlockDeltaEvent) {
	switch delta := event.Delta.(type) {
	case provider.TextPart:
		// Append text to existing content
		part.Data.Text += delta.Text

		// Update in database
		err := p.parts.Update(ctx, *part)
		if err != nil {
			p.logger.Error("Failed to update part", map[string]interface{}{
				"error": err.Error(),
			})
		}

	case provider.ToolUsePart:
		// Update tool input (for streaming tool inputs)
		if delta.Input != nil {
			part.Data.ToolInput = delta.Input
			err := p.parts.Update(ctx, *part)
			if err != nil {
				p.logger.Error("Failed to update tool part", map[string]interface{}{
					"error": err.Error(),
				})
			}
		}
	}
}

// Stop stops all active processing
func (p *Processor) Stop() {
	p.mu.Lock()
	for sessionID, cancel := range p.active {
		p.logger.Info("Stopping processing", map[string]interface{}{
			"session_id": sessionID,
		})
		cancel()
	}
	p.active = make(map[string]context.CancelFunc)
	p.mu.Unlock()
}

// IsProcessing checks if a session is actively processing
func (p *Processor) IsProcessing(sessionID string) bool {
	p.mu.Lock()
	_, exists := p.active[sessionID]
	p.mu.Unlock()
	return exists
}

// GetToolDefinitions returns tool definitions in provider format
func (p *Processor) GetToolDefinitions() []provider.ToolDefinition {
	if p.toolRegistry == nil {
		return nil
	}

	toolDefs := p.toolRegistry.ListDefinitions()
	return ConvertToolDefinitions(toolDefs)
}

// ConvertToolDefinitions converts tool.ToolDefinition to provider.ToolDefinition
func ConvertToolDefinitions(toolDefs []tool.ToolDefinition) []provider.ToolDefinition {
	result := make([]provider.ToolDefinition, 0, len(toolDefs))

	for _, td := range toolDefs {
		// Convert parameter schema
		inputSchema := make(map[string]interface{})
		inputSchema["type"] = "object"

		properties := make(map[string]interface{})
		required := make([]string, 0)

		for name, param := range td.Parameters {
			prop := convertParameterSchema(param)
			properties[name] = prop
			if param.Required {
				required = append(required, name)
			}
		}

		inputSchema["properties"] = properties
		if len(required) > 0 {
			inputSchema["required"] = required
		}

		result = append(result, provider.ToolDefinition{
			Name:        td.ID,
			Description: td.Description,
			InputSchema: inputSchema,
		})
	}

	return result
}

// convertParameterSchema converts tool.ParameterSchema to JSON schema format
func convertParameterSchema(param tool.ParameterSchema) map[string]interface{} {
	result := map[string]interface{}{
		"type":        param.Type,
		"description": param.Description,
	}

	if param.Default != nil {
		result["default"] = param.Default
	}

	if len(param.Enum) > 0 {
		result["enum"] = param.Enum
	}

	if param.Type == "object" && len(param.Properties) > 0 {
		props := make(map[string]interface{})
		for name, p := range param.Properties {
			props[name] = convertParameterSchema(p)
		}
		result["properties"] = props
	}

	if param.Type == "array" && param.Items != nil {
		result["items"] = convertParameterSchema(*param.Items)
	}

	return result
}

// ExecuteTool executes a tool and returns the result
func (p *Processor) ExecuteTool(ctx context.Context, toolName string, input map[string]interface{}, toolCtx tool.ToolContext) (*tool.ToolResult, error) {
	if p.toolRegistry == nil {
		return nil, fmt.Errorf("tool registry not configured")
	}

	return p.toolRegistry.Execute(tool.ToolID(toolName), toolCtx, input)
}

// executeToolFromPart executes a tool from a tool_use part and creates a tool_result part
// Returns the ToolResult for multi-turn support
func (p *Processor) executeToolFromPart(ctx context.Context, sessionID, messageID string, toolUsePart *database.Part) ToolResult {
	toolName := toolUsePart.Data.ToolName
	toolID := toolUsePart.Data.ToolID
	toolResult := ToolResult{
		ToolID:   toolID,
		ToolName: toolName,
	}

	// Publish tool pending event
	if p.bus != nil {
		p.bus.Publish(EventToolCallPending, map[string]interface{}{
			"session_id": sessionID,
			"message_id": messageID,
			"part_id":    toolUsePart.ID,
			"tool_name":  toolName,
			"tool_id":    toolID,
			"input":      toolUsePart.Data.ToolInput,
		})
	}

	// Build tool context
	toolCtx := tool.ToolContext{
		SessionID: sessionID,
		MessageID: messageID,
		CallID:    toolID,
		Abort:     ctx,
	}

	// Convert input to proper format
	input := toolUsePart.Data.ToolInput
	if input == nil {
		input = make(map[string]interface{})
	}

	// Publish tool running event
	if p.bus != nil {
		p.bus.Publish(EventToolCallRunning, map[string]interface{}{
			"session_id": sessionID,
			"message_id": messageID,
			"tool_name":  toolName,
			"tool_id":    toolID,
		})
	}

	// Execute the tool
	result, err := p.ExecuteTool(ctx, toolName, input, toolCtx)

	// Create tool_result part
	resultPart := database.Part{
		MessageID: messageID,
		SessionID: sessionID,
		Data: database.PartData{
			Type:     "tool_result",
			ToolID:   toolID,
			ToolName: toolName,
		},
	}

	if err != nil {
		// Tool execution failed
		resultPart.Data.Status = "error"
		resultPart.Data.ToolResult = err.Error()
		resultPart.Data.Error = err.Error()

		toolResult.Result = err.Error()
		toolResult.IsError = true

		p.logger.Error("Tool execution failed", map[string]interface{}{
			"tool_name": toolName,
			"error":     err.Error(),
		})
	} else {
		// Tool execution succeeded
		resultPart.Data.Status = "complete"
		resultPart.Data.ToolResult = result.Output

		toolResult.Result = result.Output
		toolResult.IsError = false
	}

	// Save tool_result part to database
	createdResultPart, dbErr := p.parts.Create(ctx, resultPart)
	if dbErr != nil {
		p.logger.Error("Failed to create tool_result part", map[string]interface{}{
			"error": dbErr.Error(),
		})
		createdResultPart = &resultPart
	}

	// Update tool_use part status
	if err != nil {
		toolUsePart.Data.Status = "error"
	} else {
		toolUsePart.Data.Status = "complete"
	}
	p.parts.Update(ctx, *toolUsePart)

	// Publish tool complete event
	if p.bus != nil {
		p.bus.Publish(EventToolCallComplete, map[string]interface{}{
			"session_id":  sessionID,
			"message_id":  messageID,
			"part_id":     createdResultPart.ID,
			"tool_name":   toolName,
			"tool_id":     toolID,
			"result":      resultPart.Data.ToolResult,
			"status":      resultPart.Data.Status,
			"is_error":    err != nil,
			"result_part": createdResultPart,
		})
	}

	return toolResult
}

// buildContentMessagesForProvider builds ContentMessages from history and user message
func (p *Processor) buildContentMessagesForProvider(history []database.Message, userMsg *database.Message, userText string) []provider.ContentMessage {
	messages := []provider.ContentMessage{}

	// Add history messages
	for _, h := range history {
		msg := p.convertDatabaseMessageToContentMessage(h)
		messages = append(messages, msg)
	}

	// Add user message
	if userMsg != nil {
		// Get parts for user message to extract text (use userText as fallback)
		parts, err := p.parts.ListByMessage(context.Background(), userMsg.ID)
		text := userText
		if err == nil && len(parts) > 0 {
			for _, part := range parts {
				if part.Data.Type == "text" {
					text = part.Data.Text
					break
				}
			}
		}

		messages = append(messages, provider.ContentMessage{
			Role: provider.RoleUser,
			Content: []provider.ContentPart{
				provider.TextPart{
					Type: "text",
					Text: text,
				},
			},
		})
	}

	return messages
}

// buildChatRequestWithContentMessages builds ChatRequest with ContentMessages
func (p *Processor) buildChatRequestWithContentMessages(req ProcessRequest, messages []provider.ContentMessage) provider.ChatRequest {
	return provider.ChatRequest{
		Model:           req.Model,
		ContentMessages:  messages,
		System:           req.SystemPrompt,
		MaxTokens:        4096,
		Tools:            p.GetToolDefinitions(),
		Stream:           true,
	}
}

// buildAssistantContent builds assistant content from message parts
func (p *Processor) buildAssistantContent(sessionID, messageID string) []provider.ContentPart {
	// Get parts for this message
	parts, err := p.parts.ListByMessage(context.Background(), messageID)
	if err != nil {
		p.logger.Error("Failed to list parts for assistant content", map[string]interface{}{
			"error": err.Error(),
		})
		return []provider.ContentPart{}
	}

	content := []provider.ContentPart{}
	for _, part := range parts {
		switch part.Data.Type {
		case "text":
			content = append(content, provider.TextPart{
				Type: "text",
				Text: part.Data.Text,
			})

		case "tool_use":
			content = append(content, provider.ToolUsePart{
				Type:  "tool_use",
				ID:    part.Data.ToolID,
				Name:  part.Data.ToolName,
				Input: part.Data.ToolInput,
			})
		}
	}

	return content
}

// convertDatabaseMessageToContentMessage converts a database message to ContentMessage format
func (p *Processor) convertDatabaseMessageToContentMessage(dbMsg database.Message) provider.ContentMessage {
	msg := provider.ContentMessage{
		Role: provider.Role(dbMsg.Data.Role),
	}

	content := []provider.ContentPart{}

	// Get parts from database
	parts, err := p.parts.ListByMessage(context.Background(), dbMsg.ID)
	if err == nil && len(parts) > 0 {
		for _, part := range parts {
			switch part.Data.Type {
			case "text":
				content = append(content, provider.TextPart{
					Type: "text",
					Text: part.Data.Text,
				})
			case "tool_use":
				content = append(content, provider.ToolUsePart{
					Type:  "tool_use",
					ID:    part.Data.ToolID,
					Name:  part.Data.ToolName,
					Input: part.Data.ToolInput,
				})
			case "tool_result":
				content = append(content, provider.ToolResultPart{
					Type:      "tool_result",
					ToolUseID: part.Data.ToolID,
					Content:   part.Data.ToolResult,
					IsError:   part.Data.Status == "error",
				})
			}
		}
	}

	// If no parts found, create empty text part for user role
	if len(content) == 0 && dbMsg.Data.Role == "user" {
		content = append(content, provider.TextPart{
			Type: "text",
			Text: "",
		})
	}

	msg.Content = content
	return msg
}
