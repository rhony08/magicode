// Package server provides session handlers for async message processing.
package server

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/rhony08/magicode/internal/provider"
	"github.com/rhony08/magicode/internal/session"
	"github.com/rhony08/magicode/internal/util/log"
)

// PromptAsyncRequest is the request body for async prompt
type PromptAsyncRequest struct {
	Content     string `json:"content"`     // User message content
	Model       string `json:"model"`       // Model ID (optional, uses default if empty)
	Agent       string `json:"agent"`       // Agent name (optional)
	SystemPrompt string `json:"system"`     // System prompt override (optional)
}

// PromptAsyncResponse is the response for async prompt
type PromptAsyncResponse struct {
	MessageID string `json:"message_id"` // ID of the created user message
	Status    string `json:"status"`     // "processing", "queued"
}

// handlePromptAsync handles async message processing
// POST /session/:id/prompt_async
func (s *Server) handlePromptAsync(c *fiber.Ctx) error {
	sessionID := c.Params("id")
	if sessionID == "" {
		return c.Status(400).JSON(fiber.Map{
			"error": "session_id required",
		})
	}

	// Parse request body
	var req PromptAsyncRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{
			"error": "invalid request body",
		})
	}

	if req.Content == "" {
		return c.Status(400).JSON(fiber.Map{
			"error": "content required",
		})
	}

	// Get processor from services
	processor := s.services.Processor
	if processor == nil {
		return c.Status(500).JSON(fiber.Map{
			"error": "processor not initialized",
		})
	}

	// Get model - use from request or default
	modelID := req.Model
	if modelID == "" {
		// Get default model from config/state
		modelID = s.getDefaultModel()
	}

	// Get system prompt
	systemPrompt := req.SystemPrompt
	if systemPrompt == "" {
		systemPrompt = s.getDefaultSystemPrompt(req.Agent)
	}

	// Get tool definitions
	tools := s.getToolDefinitions()

	// Build process request
	processReq := session.ProcessRequest{
		SessionID:    sessionID,
		UserMessage:  req.Content,
		Model:        provider.ModelID(modelID),
		SystemPrompt: systemPrompt,
		Agent:        req.Agent,
		Tools:        tools,
	}

	// Start processing asynchronously
	go func() {
		ctx := context.Background()
		err := processor.Process(ctx, processReq)
		if err != nil {
			log.Error("Async processing failed", "session_id", sessionID, "error", err)
			// Publish error event
			if s.services.Bus != nil {
				s.services.Bus.Publish(session.EventStreamError, map[string]interface{}{
					"session_id": sessionID,
					"error":      err.Error(),
					"error_type": "processing_error",
				})
			}
		}
	}()

	// Return immediately (async)
	return c.Status(200).JSON(PromptAsyncResponse{
		MessageID: "processing", // Will be set by processor
		Status:    "processing",
	})
}

// handleSessionEventStream handles SSE event stream for a specific session
// GET /session/:id/events
func (s *Server) handleSessionEventStream(c *fiber.Ctx) error {
	sessionID := c.Params("id")
	if sessionID == "" {
		return c.Status(400).JSON(fiber.Map{
			"error": "session_id required",
		})
	}

	// Set SSE headers
	c.Set("Content-Type", "text/event-stream")
	c.Set("Cache-Control", "no-cache, no-transform")
	c.Set("X-Accel-Buffering", "no")
	c.Set("X-Content-Type-Options", "nosniff")
	c.Set("Connection", "keep-alive")

	ctx := c.Context()

	// Send connected event
	c.Write([]byte("data: "))
	c.Write([]byte(fmt.Sprintf(`{"type":"session.connected","properties":{"session_id":"%s"}}`, sessionID)))
	c.Write([]byte("\n\n"))

	// Subscribe to session events if bus is available
	if s.services.Bus != nil {
		// Subscribe to all session-related events
		partCreatedChan, cleanup1 := s.services.Bus.Subscribe(session.EventPartCreated)
		partUpdatedChan, cleanup2 := s.services.Bus.Subscribe(session.EventPartUpdated)
		partCompleteChan, cleanup3 := s.services.Bus.Subscribe(session.EventPartComplete)
		msgCompleteChan, cleanup4 := s.services.Bus.Subscribe(session.EventMessageComplete)
		toolPendingChan, cleanup5 := s.services.Bus.Subscribe(session.EventToolCallPending)
		toolRunningChan, cleanup6 := s.services.Bus.Subscribe(session.EventToolCallRunning)
		toolCompleteChan, cleanup7 := s.services.Bus.Subscribe(session.EventToolCallComplete)
		errorChan, cleanup8 := s.services.Bus.Subscribe(session.EventStreamError)

		defer func() {
			cleanup1()
			cleanup2()
			cleanup3()
			cleanup4()
			cleanup5()
			cleanup6()
			cleanup7()
			cleanup8()
		}()

		// Heartbeat ticker
		heartbeat := time.NewTicker(10 * time.Second)
		defer heartbeat.Stop()

		// Event loop
		for {
			select {
			case <-ctx.Done():
				return nil

			case <-s.ctx.Done():
				c.Write([]byte("data: "))
				c.Write([]byte(`{"type":"session.disposed","properties":{}}`))
				c.Write([]byte("\n\n"))
				return nil

			case <-heartbeat.C:
				c.Write([]byte("data: "))
				c.Write([]byte(`{"type":"session.heartbeat","properties":{}}`))
				c.Write([]byte("\n\n"))

			case payload := <-partCreatedChan:
				props := payload.Properties.(map[string]interface{})
				if props["session_id"] == sessionID {
					data, _ := json.Marshal(map[string]interface{}{
						"type":       "session.part.created",
						"properties": props,
					})
					c.Write([]byte("data: "))
					c.Write(data)
					c.Write([]byte("\n\n"))
				}

			case payload := <-partUpdatedChan:
				props := payload.Properties.(map[string]interface{})
				if props["session_id"] == sessionID {
					data, _ := json.Marshal(map[string]interface{}{
						"type":       "session.part.updated",
						"properties": props,
					})
					c.Write([]byte("data: "))
					c.Write(data)
					c.Write([]byte("\n\n"))
				}

			case payload := <-partCompleteChan:
				props := payload.Properties.(map[string]interface{})
				if props["session_id"] == sessionID {
					data, _ := json.Marshal(map[string]interface{}{
						"type":       "session.part.complete",
						"properties": props,
					})
					c.Write([]byte("data: "))
					c.Write(data)
					c.Write([]byte("\n\n"))
				}

			case payload := <-msgCompleteChan:
				props := payload.Properties.(map[string]interface{})
				if props["session_id"] == sessionID {
					data, _ := json.Marshal(map[string]interface{}{
						"type":       "session.message.complete",
						"properties": props,
					})
					c.Write([]byte("data: "))
					c.Write(data)
					c.Write([]byte("\n\n"))
				}

			case payload := <-toolPendingChan:
				props := payload.Properties.(map[string]interface{})
				if props["session_id"] == sessionID {
					data, _ := json.Marshal(map[string]interface{}{
						"type":       "session.tool.pending",
						"properties": props,
					})
					c.Write([]byte("data: "))
					c.Write(data)
					c.Write([]byte("\n\n"))
				}

			case payload := <-toolRunningChan:
				props := payload.Properties.(map[string]interface{})
				if props["session_id"] == sessionID {
					data, _ := json.Marshal(map[string]interface{}{
						"type":       "session.tool.running",
						"properties": props,
					})
					c.Write([]byte("data: "))
					c.Write(data)
					c.Write([]byte("\n\n"))
				}

			case payload := <-toolCompleteChan:
				props := payload.Properties.(map[string]interface{})
				if props["session_id"] == sessionID {
					data, _ := json.Marshal(map[string]interface{}{
						"type":       "session.tool.complete",
						"properties": props,
					})
					c.Write([]byte("data: "))
					c.Write(data)
					c.Write([]byte("\n\n"))
				}

			case payload := <-errorChan:
				props := payload.Properties.(map[string]interface{})
				if props["session_id"] == sessionID {
					data, _ := json.Marshal(map[string]interface{}{
						"type":       "session.error",
						"properties": props,
					})
					c.Write([]byte("data: "))
					c.Write(data)
					c.Write([]byte("\n\n"))
				}
			}
		}
	}

	// No bus - just heartbeat
	heartbeat := time.NewTicker(10 * time.Second)
	defer heartbeat.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-s.ctx.Done():
			return nil
		case <-heartbeat.C:
			c.Write([]byte("data: "))
			c.Write([]byte(`{"type":"session.heartbeat","properties":{}}`))
			c.Write([]byte("\n\n"))
		}
	}
}

// getDefaultModel returns the default model for this server
func (s *Server) getDefaultModel() string {
	// TODO: Get from config or state
	// For now, return a sensible default
	if s.services.Provider != nil {
		providers := s.services.Provider.ListProviders()
		if len(providers) > 0 {
			info := providers[0].Info()
			for modelID := range info.Models {
				return string(modelID)
			}
		}
	}
	return "anthropic/claude-sonnet-4-5"
}

// getDefaultSystemPrompt returns the default system prompt for an agent
func (s *Server) getDefaultSystemPrompt(agentName string) string {
	// TODO: Get from agent config
	// For now, return empty (agent will provide it)
	return ""
}

// getToolDefinitions returns tool definitions for the provider
func (s *Server) getToolDefinitions() []provider.ToolDefinition {
	if s.services.Tools == nil || s.services.Processor == nil {
		return nil
	}
	return s.services.Processor.GetToolDefinitions()
}