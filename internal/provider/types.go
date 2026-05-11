// Package provider provides AI provider abstraction for MagiCode.
package provider

// ProviderID is a unique identifier for a provider
type ProviderID string

// Well-known provider IDs
const (
	ProviderAnthropic     ProviderID = "anthropic"
	ProviderOpenAI        ProviderID = "openai"
	ProviderGoogle        ProviderID = "google"
	ProviderGoogleVertex  ProviderID = "google-vertex"
	ProviderAzure         ProviderID = "azure"
	ProviderBedrock       ProviderID = "amazon-bedrock"
	ProviderOpenRouter    ProviderID = "openrouter"
	ProviderMistral       ProviderID = "mistral"
	ProviderGroq          ProviderID = "groq"
	ProviderXAI           ProviderID = "xai"
	ProviderCerebras      ProviderID = "cerebras"
	ProviderCohere        ProviderID = "cohere"
	ProviderTogetherAI    ProviderID = "togetherai"
	ProviderPerplexity    ProviderID = "perplexity"
	ProviderDeepInfra     ProviderID = "deepinfra"
	ProviderGitHubCopilot ProviderID = "github-copilot"
)

// ModelID is a unique identifier for a model (format: provider/model-name)
type ModelID string

// Role represents the role of a message participant
type Role string

const (
	RoleUser      Role = "user"
	RoleAssistant Role = "assistant"
	RoleSystem    Role = "system"
)

// Message represents a chat message
type Message struct {
	Role    Role   `json:"role"`
	Content string `json:"content"`
}

// ToolDefinition represents a tool that can be used by the AI
type ToolDefinition struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	InputSchema map[string]interface{} `json:"input_schema"`
}

// ToolUse represents a tool use request from the AI
type ToolUse struct {
	ID    string                 `json:"id"`
	Name  string                 `json:"name"`
	Input map[string]interface{} `json:"input"`
}

// ToolResult represents the result of a tool execution
type ToolResult struct {
	ToolUseID string `json:"tool_use_id"`
	Content   string `json:"content"`
	IsError   bool   `json:"is_error"`
}

// ContentPart represents a part of a message content
type ContentPart interface {
	ContentType() string
}

// TextPart is a text content part
type TextPart struct {
	Type string `json:"type"` // always "text"
	Text string `json:"text"`
}

func (p TextPart) ContentType() string { return "text" }

// ReasoningPart is a reasoning/thinking content part
// Used by Claude (Anthropic) for extended thinking blocks
type ReasoningPart struct {
	Type     string                 `json:"type"` // always "reasoning"
	Text     string                 `json:"text"`
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

func (p ReasoningPart) ContentType() string { return "reasoning" }

// ToolUsePart is a tool use content part
type ToolUsePart struct {
	Type  string                 `json:"type"` // always "tool_use"
	ID    string                 `json:"id"`
	Name  string                 `json:"name"`
	Input map[string]interface{} `json:"input"`
}

func (p ToolUsePart) ContentType() string { return "tool_use" }

// ToolResultPart is a tool result content part
type ToolResultPart struct {
	Type      string `json:"type"` // always "tool_result"
	ToolUseID string `json:"tool_use_id"`
	Content   string `json:"content"`
	IsError   bool   `json:"is_error"`
}

func (p ToolResultPart) ContentType() string { return "tool_result" }

// ContentMessage represents a message with structured content parts
type ContentMessage struct {
	Role    Role          `json:"role"`
	Content []ContentPart `json:"content"`
}

// ChatRequest represents a chat completion request
type ChatRequest struct {
	Model         ModelID          `json:"model"`
	Messages      []Message        `json:"messages"`       // Simple text messages
	ContentMessages []ContentMessage `json:"content_messages,omitempty"` // Structured content (for multi-turn with tools)
	System        string           `json:"system,omitempty"`
	MaxTokens     int              `json:"max_tokens,omitempty"`
	Temperature   float64          `json:"temperature,omitempty"`
	Tools         []ToolDefinition `json:"tools,omitempty"`
	Stop          []string         `json:"stop,omitempty"`
	Stream        bool             `json:"stream,omitempty"`
}

// ChatResponse represents a chat completion response
type ChatResponse struct {
	ID      string `json:"id"`
	Model   string `json:"model"`
	Content string `json:"content"`
	Role    Role   `json:"role"`

	// For responses with tool uses
	ToolUses []ToolUse `json:"tool_uses,omitempty"`

	// Usage statistics
	Usage Usage `json:"usage"`

	// Stop reason
	StopReason string `json:"stop_reason"`
}

// Usage represents token usage statistics
type Usage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
	TotalTokens  int `json:"total_tokens"`
	
	// Anthropic-specific cache fields (from API response)
	CacheRead      int `json:"cache_read_input_tokens,omitempty"`
	CacheWrite     int `json:"cache_creation_input_tokens,omitempty"`
}

// StreamEvent represents an event in a streaming response
type StreamEvent interface {
	EventType() string
}

// ContentBlockStartEvent marks the start of a content block
type ContentBlockStartEvent struct {
	Type         string      `json:"type"` // "content_block_start"
	Index        int         `json:"index"`
	ContentBlock ContentPart `json:"content_block"`
}

func (e ContentBlockStartEvent) EventType() string { return "content_block_start" }

// ContentBlockDeltaEvent contains a delta for a content block
type ContentBlockDeltaEvent struct {
	Type  string      `json:"type"` // "content_block_delta"
	Index int         `json:"index"`
	Delta ContentPart `json:"delta"`
}

func (e ContentBlockDeltaEvent) EventType() string { return "content_block_delta" }

// ContentBlockStopEvent marks the end of a content block
type ContentBlockStopEvent struct {
	Type  string `json:"type"` // "content_block_stop"
	Index int    `json:"index"`
}

func (e ContentBlockStopEvent) EventType() string { return "content_block_stop" }

// MessageStartEvent marks the start of a message
type MessageStartEvent struct {
	Type    string `json:"type"` // "message_start"
	Message struct {
		ID    string `json:"id"`
		Model string `json:"model"`
		Role  Role   `json:"role"`
		Usage Usage  `json:"usage"`
	} `json:"message"`
}

func (e MessageStartEvent) EventType() string { return "message_start" }

// MessageDeltaEvent contains a delta for the message
type MessageDeltaEvent struct {
	Type  string `json:"type"` // "message_delta"
	Delta struct {
		StopReason string `json:"stop_reason"`
	} `json:"delta"`
	Usage Usage `json:"usage"`
}

func (e MessageDeltaEvent) EventType() string { return "message_delta" }

// MessageStopEvent marks the end of a message
type MessageStopEvent struct {
	Type string `json:"type"` // "message_stop"
}

func (e MessageStopEvent) EventType() string { return "message_stop" }

// PingEvent is a keepalive ping
type PingEvent struct {
	Type string `json:"type"` // "ping"
}

func (e PingEvent) EventType() string { return "ping" }

// ErrorEvent represents an error in streaming
type ErrorEvent struct {
	Type  string `json:"type"` // "error"
	Error struct {
		Type    string `json:"type"`
		Message string `json:"message"`
	} `json:"error"`
}

func (e ErrorEvent) EventType() string { return "error" }

// ReasoningStartEvent marks the start of a reasoning/thinking block
// Used by Claude (Anthropic) for extended thinking
type ReasoningStartEvent struct {
	Type     string                 `json:"type"` // "reasoning_start"
	ID       string                 `json:"id"`   // Unique ID for this reasoning block
	Metadata map[string]interface{} `json:"metadata,omitempty"` // Provider metadata
}

func (e ReasoningStartEvent) EventType() string { return "reasoning_start" }

// ReasoningDeltaEvent contains a delta for reasoning text
type ReasoningDeltaEvent struct {
	Type     string                 `json:"type"` // "reasoning_delta"
	ID       string                 `json:"id"`   // ID of the reasoning block
	Text     string                 `json:"text"` // Delta text
	Metadata map[string]interface{} `json:"metadata,omitempty"` // Provider metadata
}

func (e ReasoningDeltaEvent) EventType() string { return "reasoning_delta" }

// ReasoningEndEvent marks the end of a reasoning/thinking block
type ReasoningEndEvent struct {
	Type     string                 `json:"type"` // "reasoning_end"
	ID       string                 `json:"id"`   // ID of the reasoning block
	Metadata map[string]interface{} `json:"metadata,omitempty"` // Provider metadata
}

func (e ReasoningEndEvent) EventType() string { return "reasoning_end" }

// StepStartEvent marks the start of a processing step
// Used for multi-step reasoning and progress tracking
type StepStartEvent struct {
	Type       string `json:"type"` // "step_start"
	SnapshotID string `json:"snapshot_id,omitempty"` // File system snapshot ID
}

func (e StepStartEvent) EventType() string { return "step_start" }

// StepFinishEvent marks the end of a processing step
type StepFinishEvent struct {
	Type     string    `json:"type"` // "step_finish"
	Tokens   Usage     `json:"tokens,omitempty"`
	Cost     float64   `json:"cost,omitempty"`
	Reason   string    `json:"reason,omitempty"` // Reason for step completion
}

func (e StepFinishEvent) EventType() string { return "step_finish" }

// Cost represents the cost of using a model
type Cost struct {
	Input  float64 `json:"input"`  // Cost per 1M input tokens
	Output float64 `json:"output"` // Cost per 1M output tokens
	Cache  float64 `json:"cache"`  // Cost per 1M cached tokens (optional)
}

// ModelInfo represents information about a model
type ModelInfo struct {
	ID          ModelID `json:"id"`
	Name        string  `json:"name"`
	Description string  `json:"description,omitempty"`

	// Capabilities
	MaxTokens         int  `json:"max_tokens,omitempty"`
	MaxInputTokens    int  `json:"max_input_tokens,omitempty"`
	MaxOutputTokens   int  `json:"max_output_tokens,omitempty"`
	SupportsVision    bool `json:"supports_vision,omitempty"`
	SupportsTools     bool `json:"supports_tools,omitempty"`
	SupportsStreaming bool `json:"supports_streaming,omitempty"`

	// Cost per 1M tokens
	Cost Cost `json:"cost"`

	// Provider-specific metadata
	Metadata map[string]interface{} `json:"metadata,omitempty"`
}

// ProviderInfo represents information about a provider
type ProviderInfo struct {
	ID          ProviderID `json:"id"`
	Name        string     `json:"name"`
	Description string     `json:"description,omitempty"`
	BaseURL     string     `json:"base_url"`

	// Environment variables for API keys
	EnvKeys []string `json:"env"`

	// Models available from this provider
	Models map[ModelID]ModelInfo `json:"models"`

	// Provider-specific options
	Options map[string]interface{} `json:"options,omitempty"`
}
