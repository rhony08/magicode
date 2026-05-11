// Package provider provides Anthropic AI provider implementation.
package provider

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/rhony08/magicode/internal/util/log"
)

// AnthropicProvider implements Provider for Anthropic API
type AnthropicProvider struct {
	apiKey  string
	baseURL string
	client  *http.Client
	models  map[ModelID]ModelInfo
}

// Default Anthropic base URL
const AnthropicDefaultBaseURL = "https://api.anthropic.com/v1"

// AnthropicModels are the well-known Anthropic models
var AnthropicModels = map[ModelID]ModelInfo{
	FormatModelID(ProviderAnthropic, "claude-sonnet-4-5"): {
		ID:               FormatModelID(ProviderAnthropic, "claude-sonnet-4-5"),
		Name:             "Claude Sonnet 4.5",
		Description:      "Most intelligent model with exceptional performance",
		MaxInputTokens:   200000,
		MaxOutputTokens:  8192,
		SupportsVision:   true,
		SupportsTools:    true,
		SupportsStreaming: true,
		Cost: Cost{
			Input:  3.00,
			Output: 15.00,
			Cache:  0.30,
		},
	},
	FormatModelID(ProviderAnthropic, "claude-sonnet-4"): {
		ID:               FormatModelID(ProviderAnthropic, "claude-sonnet-4"),
		Name:             "Claude Sonnet 4",
		Description:      "Balanced model for everyday tasks",
		MaxInputTokens:   200000,
		MaxOutputTokens:  8192,
		SupportsVision:   true,
		SupportsTools:    true,
		SupportsStreaming: true,
		Cost: Cost{
			Input:  3.00,
			Output: 15.00,
		},
	},
	FormatModelID(ProviderAnthropic, "claude-3-5-haiku"): {
		ID:               FormatModelID(ProviderAnthropic, "claude-3-5-haiku"),
		Name:             "Claude 3.5 Haiku",
		Description:      "Fast, compact model for simple tasks",
		MaxInputTokens:   200000,
		MaxOutputTokens:  8192,
		SupportsVision:   true,
		SupportsTools:    true,
		SupportsStreaming: true,
		Cost: Cost{
			Input:  0.80,
			Output: 4.00,
		},
	},
	FormatModelID(ProviderAnthropic, "claude-3-opus"): {
		ID:               FormatModelID(ProviderAnthropic, "claude-3-opus"),
		Name:             "Claude 3 Opus",
		Description:      "Powerful model for complex tasks",
		MaxInputTokens:   200000,
		MaxOutputTokens:  4096,
		SupportsVision:   true,
		SupportsTools:    true,
		SupportsStreaming: true,
		Cost: Cost{
			Input:  15.00,
			Output: 75.00,
		},
	},
	FormatModelID(ProviderAnthropic, "claude-3-sonnet"): {
		ID:               FormatModelID(ProviderAnthropic, "claude-3-sonnet"),
		Name:             "Claude 3 Sonnet",
		Description:      "Balanced model",
		MaxInputTokens:   200000,
		MaxOutputTokens:  4096,
		SupportsVision:   true,
		SupportsTools:    true,
		SupportsStreaming: true,
		Cost: Cost{
			Input:  3.00,
			Output: 15.00,
		},
	},
	FormatModelID(ProviderAnthropic, "claude-3-haiku"): {
		ID:               FormatModelID(ProviderAnthropic, "claude-3-haiku"),
		Name:             "Claude 3 Haiku",
		Description:      "Fast, lightweight model",
		MaxInputTokens:   200000,
		MaxOutputTokens:  4096,
		SupportsVision:   true,
		SupportsTools:    true,
		SupportsStreaming: true,
		Cost: Cost{
			Input:  0.25,
			Output: 1.25,
		},
	},
}

// NewAnthropicProvider creates a new Anthropic provider
func NewAnthropicProvider(apiKey string) *AnthropicProvider {
	return &AnthropicProvider{
		apiKey:  apiKey,
		baseURL: AnthropicDefaultBaseURL,
		client: &http.Client{
			Timeout: 120 * time.Second,
			Transport: &http.Transport{
				MaxIdleConns:        10,
				IdleConnTimeout:     30 * time.Second,
				DisableCompression:  false,
				MaxIdleConnsPerHost: 5,
			},
		},
		models: AnthropicModels,
	}
}

// ID returns the provider ID
func (p *AnthropicProvider) ID() ProviderID {
	return ProviderAnthropic
}

// Info returns provider information
func (p *AnthropicProvider) Info() ProviderInfo {
	return ProviderInfo{
		ID:          ProviderAnthropic,
		Name:        "Anthropic",
		Description: "Claude AI models by Anthropic",
		BaseURL:     p.baseURL,
		EnvKeys:     []string{"ANTHROPIC_API_KEY"},
		Models:      p.models,
	}
}

// SetAPIKey sets the API key
func (p *AnthropicProvider) SetAPIKey(key string) {
	p.apiKey = key
}

// SetBaseURL sets a custom base URL
func (p *AnthropicProvider) SetBaseURL(url string) {
	p.baseURL = url
}

// Close cleans up resources
func (p *AnthropicProvider) Close() error {
	p.client.CloseIdleConnections()
	return nil
}

// ValidateKey checks if the API key is valid
func (p *AnthropicProvider) ValidateKey(ctx context.Context) error {
	if p.apiKey == "" {
		return fmt.Errorf("anthropic API key is not set")
	}
	
	// Make a minimal request to validate the key
	req := ChatRequest{
		Model:     FormatModelID(ProviderAnthropic, "claude-3-haiku"),
		MaxTokens: 1,
		Messages:  []Message{{Role: RoleUser, Content: "Hi"}},
	}
	
	_, err := p.Chat(ctx, req)
	if err != nil {
		if strings.Contains(err.Error(), "invalid api_key") {
			return fmt.Errorf("invalid Anthropic API key")
		}
		// Other errors might be rate limits, which mean the key is valid
		if strings.Contains(err.Error(), "rate limit") {
			return nil
		}
	}
	return err
}

// anthropicRequest is the Anthropic API request format
type anthropicRequest struct {
	Model     string               `json:"model"`
	MaxTokens int                  `json:"max_tokens"`
	Messages  []anthropicMessage   `json:"messages"`
	System    string               `json:"system,omitempty"`
	Tools     []anthropicTool      `json:"tools,omitempty"`
	Stream    bool                 `json:"stream,omitempty"`
}

type anthropicMessage struct {
	Role    string          `json:"role"`
	Content anthropicContent `json:"content"`
}

type anthropicContent interface{}

// anthropicTextContent is text content
type anthropicTextContent struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

// anthropicToolUseContent is tool use content
type anthropicToolUseContent struct {
	Type  string                 `json:"type"`
	ID    string                 `json:"id"`
	Name  string                 `json:"name"`
	Input map[string]interface{} `json:"input"`
}

// anthropicToolResultContent is tool result content
type anthropicToolResultContent struct {
	Type      string `json:"type"`
	ToolUseID string `json:"tool_use_id"`
	Content   string `json:"content"`
	IsError   bool   `json:"is_error,omitempty"`
}

type anthropicTool struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	InputSchema map[string]interface{} `json:"input_schema"`
}

// Chat sends a non-streaming chat request
func (p *AnthropicProvider) Chat(ctx context.Context, req ChatRequest) (*ChatResponse, error) {
	// Convert request to Anthropic format
	providerID, modelName := ParseModelID(req.Model)
	if providerID != ProviderAnthropic {
		return nil, fmt.Errorf("invalid provider for anthropic: %s", providerID)
	}

	aReq := anthropicRequest{
		Model:     modelName,
		MaxTokens: req.MaxTokens,
		System:    req.System,
		Stream:    false,
	}

	// Convert messages
	for _, msg := range req.Messages {
		content := []anthropicTextContent{{Type: "text", Text: msg.Content}}
		aReq.Messages = append(aReq.Messages, anthropicMessage{
			Role:    string(msg.Role),
			Content: content,
		})
	}

	// Convert tools
	for _, tool := range req.Tools {
		aReq.Tools = append(aReq.Tools, anthropicTool{
			Name:        tool.Name,
			Description: tool.Description,
			InputSchema: tool.InputSchema,
		})
	}

	// Set default max tokens
	if aReq.MaxTokens == 0 {
		aReq.MaxTokens = 4096
	}

	body, err := json.Marshal(aReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", p.baseURL+"/messages", strings.NewReader(string(body)))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-api-key", p.apiKey)
	httpReq.Header.Set("anthropic-version", "2023-06-01")
	httpReq.Header.Set("anthropic-beta", "interleaved-thinking-2025-05-14,fine-grained-tool-streaming-2025-05-14")

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("anthropic API error: %s - %s", resp.Status, string(bodyBytes))
	}

	// Parse response
	var aResp anthropicResponse
	if err := json.NewDecoder(resp.Body).Decode(&aResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return convertAnthropicResponse(&aResp), nil
}

type anthropicResponse struct {
	ID           string              `json:"id"`
	Type         string              `json:"type"`
	Role         string              `json:"role"`
	Model        string              `json:"model"`
	Content      []anthropicContentBlock `json:"content"`
	StopReason   string              `json:"stop_reason"`
	StopSequence string              `json:"stop_sequence"`
	Usage        anthropicUsage      `json:"usage"`
}

type anthropicContentBlock struct {
	Type string `json:"type"`
	Text string `json:"text,omitempty"`
	
	// For tool_use
	ID    string                 `json:"id,omitempty"`
	Name  string                 `json:"name,omitempty"`
	Input map[string]interface{} `json:"input,omitempty"`
}

type anthropicUsage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
}

func convertAnthropicResponse(aResp *anthropicResponse) *ChatResponse {
	resp := &ChatResponse{
		ID:         aResp.ID,
		Model:      aResp.Model,
		Role:       RoleAssistant,
		StopReason: aResp.StopReason,
		Usage: Usage{
			InputTokens:  aResp.Usage.InputTokens,
			OutputTokens: aResp.Usage.OutputTokens,
			TotalTokens:  aResp.Usage.InputTokens + aResp.Usage.OutputTokens,
		},
	}

	for _, block := range aResp.Content {
		switch block.Type {
		case "text":
			resp.Content += block.Text
		case "tool_use":
			resp.ToolUses = append(resp.ToolUses, ToolUse{
				ID:    block.ID,
				Name:  block.Name,
				Input: block.Input,
			})
		}
	}

	return resp
}

// StreamChat sends a streaming chat request
func (p *AnthropicProvider) StreamChat(ctx context.Context, req ChatRequest) (<-chan StreamEvent, error) {
	body, err := p.buildStreamRequestBody(req)
	if err != nil {
		return nil, err
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", p.baseURL+"/messages", strings.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-api-key", p.apiKey)
	httpReq.Header.Set("anthropic-version", "2023-06-01")
	httpReq.Header.Set("anthropic-beta", "interleaved-thinking-2025-05-14,fine-grained-tool-streaming-2025-05-14")
	httpReq.Header.Set("Accept", "text/event-stream")

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		return nil, fmt.Errorf("anthropic API error: %s - %s", resp.Status, string(bodyBytes))
	}

	events := make(chan StreamEvent, 100)
	go p.parseStreamResponse(resp.Body, events)

	return events, nil
}

// StreamChatRaw returns the raw response body for custom parsing
func (p *AnthropicProvider) StreamChatRaw(ctx context.Context, req ChatRequest) (io.ReadCloser, error) {
	body, err := p.buildStreamRequestBody(req)
	if err != nil {
		return nil, err
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", p.baseURL+"/messages", strings.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-api-key", p.apiKey)
	httpReq.Header.Set("anthropic-version", "2023-06-01")
	httpReq.Header.Set("anthropic-beta", "interleaved-thinking-2025-05-14,fine-grained-tool-streaming-2025-05-14")
	httpReq.Header.Set("Accept", "text/event-stream")

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		return nil, fmt.Errorf("anthropic API error: %s - %s", resp.Status, string(bodyBytes))
	}

	return resp.Body, nil
}

func (p *AnthropicProvider) buildStreamRequestBody(req ChatRequest) (string, error) {
	providerID, modelName := ParseModelID(req.Model)
	if providerID != ProviderAnthropic {
		return "", fmt.Errorf("invalid provider for anthropic: %s", providerID)
	}

	aReq := anthropicRequest{
		Model:     modelName,
		MaxTokens: req.MaxTokens,
		System:    req.System,
		Stream:    true,
	}

	// Convert ContentMessages if present (for multi-turn with tools)
	if len(req.ContentMessages) > 0 {
		for _, msg := range req.ContentMessages {
			content := p.convertContentPartsToAnthropic(msg.Content)
			aReq.Messages = append(aReq.Messages, anthropicMessage{
				Role:    string(msg.Role),
				Content: content,
			})
		}
	} else {
		// Convert simple Messages
		for _, msg := range req.Messages {
			content := []anthropicTextContent{{Type: "text", Text: msg.Content}}
			aReq.Messages = append(aReq.Messages, anthropicMessage{
				Role:    string(msg.Role),
				Content: content,
			})
		}
	}

	// Convert tools
	for _, tool := range req.Tools {
		aReq.Tools = append(aReq.Tools, anthropicTool{
			Name:        tool.Name,
			Description: tool.Description,
			InputSchema: tool.InputSchema,
		})
	}

	// Set default max tokens
	if aReq.MaxTokens == 0 {
		aReq.MaxTokens = 4096
	}

	body, err := json.Marshal(aReq)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	return string(body), nil
}

// convertContentPartsToAnthropic converts ContentParts to anthropic content format
func (p *AnthropicProvider) convertContentPartsToAnthropic(parts []ContentPart) []anthropicContent {
	content := []anthropicContent{}

	for _, part := range parts {
		switch cp := part.(type) {
		case TextPart:
			content = append(content, anthropicTextContent{
				Type: "text",
				Text: cp.Text,
			})

		case ToolUsePart:
			content = append(content, anthropicToolUseContent{
				Type:  "tool_use",
				ID:    cp.ID,
				Name:  cp.Name,
				Input: cp.Input,
			})

		case ToolResultPart:
			content = append(content, anthropicToolResultContent{
				Type:      "tool_result",
				ToolUseID: cp.ToolUseID,
				Content:   cp.Content,
				IsError:   cp.IsError,
			})
		}
	}

	return content
}

// parseStreamResponse parses SSE stream from Anthropic
func (p *AnthropicProvider) parseStreamResponse(body io.ReadCloser, events chan StreamEvent) {
	defer close(events)
	defer body.Close()

	scanner := bufio.NewScanner(body)
	scanner.Buffer(make([]byte, 64*1024), 1024*1024) // 1MB max line size

	for scanner.Scan() {
		line := scanner.Text()
		
		// Skip empty lines
		if line == "" {
			continue
		}
		
		// Parse SSE format: data: {...}
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		
		data := strings.TrimPrefix(line, "data: ")
		if data == "" {
			continue
		}

		// Parse the event JSON
		var raw map[string]interface{}
		if err := json.Unmarshal([]byte(data), &raw); err != nil {
			log.Error("Failed to parse stream event", "data", data, "error", err)
			continue
		}

		eventType, ok := raw["type"].(string)
		if !ok {
			continue
		}

		event := parseAnthropicEvent(eventType, data)
		if event != nil {
			events <- event
		}
	}

	if err := scanner.Err(); err != nil {
		log.Error("Stream scanner error", "error", err)
		events <- ErrorEvent{
			Type: "error",
			Error: struct {
				Type    string `json:"type"`
				Message string `json:"message"`
			}{
				Type:    "stream_error",
				Message: err.Error(),
			},
		}
	}
}

func parseAnthropicEvent(eventType, data string) StreamEvent {
	switch eventType {
	case "message_start":
		var e MessageStartEvent
		if err := json.Unmarshal([]byte(data), &e); err == nil {
			return e
		}
	case "content_block_start":
		var raw struct {
			Type         string `json:"type"`
			Index        int    `json:"index"`
			ContentBlock struct {
				Type string `json:"type"`
				Text string `json:"text,omitempty"`
				ID   string `json:"id,omitempty"`
				Name string `json:"name,omitempty"`
			} `json:"content_block"`
		}
		if err := json.Unmarshal([]byte(data), &raw); err == nil {
			var block ContentPart
			switch raw.ContentBlock.Type {
			case "text":
				block = TextPart{Type: "text", Text: raw.ContentBlock.Text}
			case "tool_use":
				block = ToolUsePart{Type: "tool_use", ID: raw.ContentBlock.ID, Name: raw.ContentBlock.Name}
			}
			return ContentBlockStartEvent{Type: eventType, Index: raw.Index, ContentBlock: block}
		}
	case "content_block_delta":
		var raw struct {
			Type  string `json:"type"`
			Index int    `json:"index"`
			Delta struct {
				Type string `json:"type"`
				Text string `json:"text,omitempty"`
			} `json:"delta"`
		}
		if err := json.Unmarshal([]byte(data), &raw); err == nil {
			return ContentBlockDeltaEvent{
				Type:  eventType,
				Index: raw.Index,
				Delta: TextPart{Type: "text", Text: raw.Delta.Text},
			}
		}
	case "content_block_stop":
		var e ContentBlockStopEvent
		if err := json.Unmarshal([]byte(data), &e); err == nil {
			return e
		}
	case "message_delta":
		var e MessageDeltaEvent
		if err := json.Unmarshal([]byte(data), &e); err == nil {
			return e
		}
	case "message_stop":
		var e MessageStopEvent
		if err := json.Unmarshal([]byte(data), &e); err == nil {
			return e
		}
	case "error":
		var e ErrorEvent
		if err := json.Unmarshal([]byte(data), &e); err == nil {
			return e
		}
	}
	return nil
}