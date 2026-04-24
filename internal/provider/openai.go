// Package provider provides OpenAI provider implementation.
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

// OpenAIProvider implements Provider for OpenAI API
type OpenAIProvider struct {
	apiKey  string
	baseURL string
	client  *http.Client
	models  map[ModelID]ModelInfo
}

// Default OpenAI base URL
const OpenAIDefaultBaseURL = "https://api.openai.com/v1"

// OpenAIModels are the well-known OpenAI models
var OpenAIModels = map[ModelID]ModelInfo{
	FormatModelID(ProviderOpenAI, "gpt-4o"): {
		ID:               FormatModelID(ProviderOpenAI, "gpt-4o"),
		Name:             "GPT-4o",
		Description:      "Most advanced multimodal model",
		MaxInputTokens:   128000,
		MaxOutputTokens:  4096,
		SupportsVision:   true,
		SupportsTools:    true,
		SupportsStreaming: true,
		Cost: Cost{
			Input:  5.00,
			Output: 15.00,
		},
	},
	FormatModelID(ProviderOpenAI, "gpt-4o-mini"): {
		ID:               FormatModelID(ProviderOpenAI, "gpt-4o-mini"),
		Name:             "GPT-4o Mini",
		Description:      "Affordable and intelligent small model",
		MaxInputTokens:   128000,
		MaxOutputTokens:  16384,
		SupportsVision:   true,
		SupportsTools:    true,
		SupportsStreaming: true,
		Cost: Cost{
			Input:  0.15,
			Output: 0.60,
		},
	},
	FormatModelID(ProviderOpenAI, "gpt-4-turbo"): {
		ID:               FormatModelID(ProviderOpenAI, "gpt-4-turbo"),
		Name:             "GPT-4 Turbo",
		Description:      "Previous generation flagship model",
		MaxInputTokens:   128000,
		MaxOutputTokens:  4096,
		SupportsVision:   true,
		SupportsTools:    true,
		SupportsStreaming: true,
		Cost: Cost{
			Input:  10.00,
			Output: 30.00,
		},
	},
	FormatModelID(ProviderOpenAI, "gpt-4"): {
		ID:               FormatModelID(ProviderOpenAI, "gpt-4"),
		Name:             "GPT-4",
		Description:      "Previous generation flagship model",
		MaxInputTokens:   8192,
		MaxOutputTokens:  4096,
		SupportsVision:   false,
		SupportsTools:    true,
		SupportsStreaming: true,
		Cost: Cost{
			Input:  30.00,
			Output: 60.00,
		},
	},
	FormatModelID(ProviderOpenAI, "gpt-3.5-turbo"): {
		ID:               FormatModelID(ProviderOpenAI, "gpt-3.5-turbo"),
		Name:             "GPT-3.5 Turbo",
		Description:      "Fast, affordable model",
		MaxInputTokens:   16385,
		MaxOutputTokens:  4096,
		SupportsVision:   false,
		SupportsTools:    true,
		SupportsStreaming: true,
		Cost: Cost{
			Input:  0.50,
			Output: 1.50,
		},
	},
	FormatModelID(ProviderOpenAI, "o1-preview"): {
		ID:               FormatModelID(ProviderOpenAI, "o1-preview"),
		Name:             "o1 Preview",
		Description:      "Reasoning model for complex problems",
		MaxInputTokens:   128000,
		MaxOutputTokens:  32768,
		SupportsVision:   true,
		SupportsTools:    false,
		SupportsStreaming: false,
		Cost: Cost{
			Input:  15.00,
			Output: 60.00,
		},
	},
	FormatModelID(ProviderOpenAI, "o1-mini"): {
		ID:               FormatModelID(ProviderOpenAI, "o1-mini"),
		Name:             "o1 Mini",
		Description:      "Fast reasoning model",
		MaxInputTokens:   128000,
		MaxOutputTokens:  65536,
		SupportsVision:   true,
		SupportsTools:    false,
		SupportsStreaming: false,
		Cost: Cost{
			Input:  3.00,
			Output: 12.00,
		},
	},
}

// NewOpenAIProvider creates a new OpenAI provider
func NewOpenAIProvider(apiKey string) *OpenAIProvider {
	return &OpenAIProvider{
		apiKey:  apiKey,
		baseURL: OpenAIDefaultBaseURL,
		client: &http.Client{
			Timeout: 120 * time.Second,
			Transport: &http.Transport{
				MaxIdleConns:        10,
				IdleConnTimeout:     30 * time.Second,
				DisableCompression:  false,
				MaxIdleConnsPerHost: 5,
			},
		},
		models: OpenAIModels,
	}
}

// ID returns the provider ID
func (p *OpenAIProvider) ID() ProviderID {
	return ProviderOpenAI
}

// Info returns provider information
func (p *OpenAIProvider) Info() ProviderInfo {
	return ProviderInfo{
		ID:          ProviderOpenAI,
		Name:        "OpenAI",
		Description: "GPT models by OpenAI",
		BaseURL:     p.baseURL,
		EnvKeys:     []string{"OPENAI_API_KEY"},
		Models:      p.models,
	}
}

// SetAPIKey sets the API key
func (p *OpenAIProvider) SetAPIKey(key string) {
	p.apiKey = key
}

// SetBaseURL sets a custom base URL
func (p *OpenAIProvider) SetBaseURL(url string) {
	p.baseURL = url
}

// Close cleans up resources
func (p *OpenAIProvider) Close() error {
	p.client.CloseIdleConnections()
	return nil
}

// ValidateKey checks if the API key is valid
func (p *OpenAIProvider) ValidateKey(ctx context.Context) error {
	if p.apiKey == "" {
		return fmt.Errorf("openai API key is not set")
	}
	
	// Make a minimal request to validate the key
	req := ChatRequest{
		Model:     FormatModelID(ProviderOpenAI, "gpt-3.5-turbo"),
		MaxTokens: 1,
		Messages:  []Message{{Role: RoleUser, Content: "Hi"}},
	}
	
	_, err := p.Chat(ctx, req)
	if err != nil {
		if strings.Contains(err.Error(), "invalid api_key") || strings.Contains(err.Error(), "Incorrect API key") {
			return fmt.Errorf("invalid OpenAI API key")
		}
		// Rate limits mean key is valid
		if strings.Contains(err.Error(), "rate limit") {
			return nil
		}
	}
	return err
}

// openaiRequest is the OpenAI API request format
type openaiRequest struct {
	Model       string           `json:"model"`
	Messages    []openaiMessage  `json:"messages"`
	MaxTokens   int              `json:"max_tokens,omitempty"`
	Temperature float64          `json:"temperature,omitempty"`
	Tools       []openaiTool     `json:"tools,omitempty"`
	Stop        []string         `json:"stop,omitempty"`
	Stream      bool             `json:"stream,omitempty"`
}

type openaiMessage struct {
	Role    string      `json:"role"`
	Content interface{} `json:"content"`
}

type openaiTool struct {
	Type     string                 `json:"type"`
	Function openaiToolFunction     `json:"function"`
}

type openaiToolFunction struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Parameters  map[string]interface{} `json:"parameters"`
}

// Chat sends a non-streaming chat request
func (p *OpenAIProvider) Chat(ctx context.Context, req ChatRequest) (*ChatResponse, error) {
	providerID, modelName := ParseModelID(req.Model)
	if providerID != ProviderOpenAI {
		return nil, fmt.Errorf("invalid provider for openai: %s", providerID)
	}

	oReq := openaiRequest{
		Model:       modelName,
		MaxTokens:   req.MaxTokens,
		Temperature: req.Temperature,
		Stream:      false,
	}

	// Add system message if present
	if req.System != "" {
		oReq.Messages = append(oReq.Messages, openaiMessage{
			Role:    "system",
			Content: req.System,
		})
	}

	// Convert messages
	for _, msg := range req.Messages {
		oReq.Messages = append(oReq.Messages, openaiMessage{
			Role:    string(msg.Role),
			Content: msg.Content,
		})
	}

	// Convert tools
	for _, tool := range req.Tools {
		oReq.Tools = append(oReq.Tools, openaiTool{
			Type: "function",
			Function: openaiToolFunction{
				Name:        tool.Name,
				Description: tool.Description,
				Parameters:  tool.InputSchema,
			},
		})
	}

	// Set default max tokens for models that need it
	if oReq.MaxTokens == 0 {
		oReq.MaxTokens = 4096
	}

	body, err := json.Marshal(oReq)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", p.baseURL+"/chat/completions", strings.NewReader(string(body)))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+p.apiKey)

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("openai API error: %s - %s", resp.Status, string(bodyBytes))
	}

	// Parse response
	var oResp openaiResponse
	if err := json.NewDecoder(resp.Body).Decode(&oResp); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return convertOpenAIResponse(&oResp), nil
}

type openaiResponse struct {
	ID      string         `json:"id"`
	Object  string         `json:"object"`
	Created int64          `json:"created"`
	Model   string         `json:"model"`
	Choices []openaiChoice `json:"choices"`
	Usage   openaiUsage    `json:"usage"`
}

type openaiChoice struct {
	Index        int             `json:"index"`
	Message      openaiChoiceMsg `json:"message"`
	FinishReason string          `json:"finish_reason"`
}

type openaiChoiceMsg struct {
	Role      string              `json:"role"`
	Content   string              `json:"content"`
	ToolCalls []openaiToolCall    `json:"tool_calls,omitempty"`
}

type openaiToolCall struct {
	ID       string                 `json:"id"`
	Type     string                 `json:"type"`
	Function openaiToolCallFunction `json:"function"`
}

type openaiToolCallFunction struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

type openaiUsage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

func convertOpenAIResponse(oResp *openaiResponse) *ChatResponse {
	if len(oResp.Choices) == 0 {
		return &ChatResponse{
			ID:    oResp.ID,
			Model: oResp.Model,
		}
	}

	choice := oResp.Choices[0]
	resp := &ChatResponse{
		ID:         oResp.ID,
		Model:      oResp.Model,
		Role:       Role(choice.Message.Role),
		Content:    choice.Message.Content,
		StopReason: choice.FinishReason,
		Usage: Usage{
			InputTokens:  oResp.Usage.PromptTokens,
			OutputTokens: oResp.Usage.CompletionTokens,
			TotalTokens:  oResp.Usage.TotalTokens,
		},
	}

	// Convert tool calls
	for _, tc := range choice.Message.ToolCalls {
		var input map[string]interface{}
		if err := json.Unmarshal([]byte(tc.Function.Arguments), &input); err == nil {
			resp.ToolUses = append(resp.ToolUses, ToolUse{
				ID:    tc.ID,
				Name:  tc.Function.Name,
				Input: input,
			})
		}
	}

	return resp
}

// StreamChat sends a streaming chat request
func (p *OpenAIProvider) StreamChat(ctx context.Context, req ChatRequest) (<-chan StreamEvent, error) {
	body, err := p.buildStreamRequestBody(req)
	if err != nil {
		return nil, err
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", p.baseURL+"/chat/completions", strings.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+p.apiKey)
	httpReq.Header.Set("Accept", "text/event-stream")

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		return nil, fmt.Errorf("openai API error: %s - %s", resp.Status, string(bodyBytes))
	}

	events := make(chan StreamEvent, 100)
	go p.parseStreamResponse(resp.Body, events)

	return events, nil
}

// StreamChatRaw returns the raw response body
func (p *OpenAIProvider) StreamChatRaw(ctx context.Context, req ChatRequest) (io.ReadCloser, error) {
	body, err := p.buildStreamRequestBody(req)
	if err != nil {
		return nil, err
	}

	httpReq, err := http.NewRequestWithContext(ctx, "POST", p.baseURL+"/chat/completions", strings.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("Authorization", "Bearer "+p.apiKey)
	httpReq.Header.Set("Accept", "text/event-stream")

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		resp.Body.Close()
		return nil, fmt.Errorf("openai API error: %s - %s", resp.Status, string(bodyBytes))
	}

	return resp.Body, nil
}

func (p *OpenAIProvider) buildStreamRequestBody(req ChatRequest) (string, error) {
	providerID, modelName := ParseModelID(req.Model)
	if providerID != ProviderOpenAI {
		return "", fmt.Errorf("invalid provider for openai: %s", providerID)
	}

	oReq := openaiRequest{
		Model:       modelName,
		MaxTokens:   req.MaxTokens,
		Temperature: req.Temperature,
		Stream:      true,
	}

	// Add system message if present
	if req.System != "" {
		oReq.Messages = append(oReq.Messages, openaiMessage{
			Role:    "system",
			Content: req.System,
		})
	}

	// Convert messages
	for _, msg := range req.Messages {
		oReq.Messages = append(oReq.Messages, openaiMessage{
			Role:    string(msg.Role),
			Content: msg.Content,
		})
	}

	// Convert tools
	for _, tool := range req.Tools {
		oReq.Tools = append(oReq.Tools, openaiTool{
			Type: "function",
			Function: openaiToolFunction{
				Name:        tool.Name,
				Description: tool.Description,
				Parameters:  tool.InputSchema,
			},
		})
	}

	if oReq.MaxTokens == 0 {
		oReq.MaxTokens = 4096
	}

	body, err := json.Marshal(oReq)
	if err != nil {
		return "", fmt.Errorf("failed to marshal request: %w", err)
	}

	return string(body), nil
}

// parseStreamResponse parses SSE stream from OpenAI
func (p *OpenAIProvider) parseStreamResponse(body io.ReadCloser, events chan StreamEvent) {
	defer close(events)
	defer body.Close()

	scanner := bufio.NewScanner(body)
	scanner.Buffer(make([]byte, 64*1024), 1024*1024)

	var accumulatedContent = make(map[int]string)

	for scanner.Scan() {
		line := scanner.Text()
		
		if line == "" {
			continue
		}
		
		if !strings.HasPrefix(line, "data: ") {
			continue
		}
		
		data := strings.TrimPrefix(line, "data: ")
		
		// Check for stream end
		if data == "[DONE]" {
			// Send final content block stop
			for idx := range accumulatedContent {
				events <- ContentBlockStopEvent{Type: "content_block_stop", Index: idx}
			}
			events <- MessageStopEvent{Type: "message_stop"}
			return
		}

		var chunk openaiStreamChunk
		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			log.Error("Failed to parse stream chunk", "data", data, "error", err)
			continue
		}

		// Convert to our event format
		for _, choice := range chunk.Choices {
			if choice.Delta.Content != "" {
				// Content delta
				if accumulatedContent[choice.Index] == "" {
					// First content for this index - send start event
					events <- ContentBlockStartEvent{
						Type:        "content_block_start",
						Index:        choice.Index,
						ContentBlock: TextPart{Type: "text"},
					}
				}
				
				events <- ContentBlockDeltaEvent{
					Type:  "content_block_delta",
					Index: choice.Index,
					Delta: TextPart{Type: "text", Text: choice.Delta.Content},
				}
				accumulatedContent[choice.Index] += choice.Delta.Content
			}

			// Handle tool call deltas
			for _, tc := range choice.Delta.ToolCalls {
				if accumulatedContent[choice.Index] == "" {
					// First content - send start event
					events <- ContentBlockStartEvent{
						Type:        "content_block_start",
						Index:        choice.Index,
						ContentBlock: ToolUsePart{Type: "tool_use", ID: tc.ID, Name: tc.Function.Name},
					}
				}

				// Parse arguments delta
				var input map[string]interface{}
				if tc.Function.Arguments != "" {
					if err := json.Unmarshal([]byte(tc.Function.Arguments), &input); err == nil {
						events <- ContentBlockDeltaEvent{
							Type:  "content_block_delta",
							Index: choice.Index,
							Delta: ToolUsePart{Type: "tool_use", ID: tc.ID, Name: tc.Function.Name, Input: input},
						}
					}
				}
			}

			// Handle finish reason
			if choice.FinishReason != "" && choice.FinishReason != "null" {
				events <- MessageDeltaEvent{
					Type: "message_delta",
					Delta: struct {
						StopReason string `json:"stop_reason"`
					}{
						StopReason: choice.FinishReason,
					},
				}
			}
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

type openaiStreamChunk struct {
	ID      string            `json:"id"`
	Object  string            `json:"object"`
	Created int64             `json:"created"`
	Model   string            `json:"model"`
	Choices []openaiStreamChoice `json:"choices"`
}

type openaiStreamChoice struct {
	Index        int               `json:"index"`
	Delta        openaiStreamDelta `json:"delta"`
	FinishReason string            `json:"finish_reason"`
}

type openaiStreamDelta struct {
	Role      string               `json:"role,omitempty"`
	Content   string               `json:"content,omitempty"`
	ToolCalls []openaiStreamToolCall `json:"tool_calls,omitempty"`
}

type openaiStreamToolCall struct {
	ID       string                     `json:"id,omitempty"`
	Type     string                     `json:"type,omitempty"`
	Function openaiStreamToolCallFunction `json:"function"`
}

type openaiStreamToolCallFunction struct {
	Name      string `json:"name,omitempty"`
	Arguments string `json:"arguments,omitempty"`
}