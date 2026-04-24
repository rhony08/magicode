package server

import (
	"fmt"
	"runtime"
	"time"

	"github.com/gofiber/fiber/v2"
)

// handleHealth handles health check requests.
func (s *Server) handleHealth(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{
		"status":    "healthy",
		"timestamp": time.Now().UnixMilli(),
		"version":   Version,
		"uptime":    time.Since(startTime).Milliseconds(),
	})
}

// handleVersion handles version requests.
func (s *Server) handleVersion(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{
		"version":    Version,
		"go_version": runtime.Version(),
		"platform":   runtime.GOOS,
		"arch":       runtime.GOARCH,
		"build_time": BuildTime,
	})
}

// handleGlobalEventStream handles SSE event stream.
func (s *Server) handleGlobalEventStream(c *fiber.Ctx) error {
	// Set SSE headers
	c.Set("Content-Type", "text/event-stream")
	c.Set("Cache-Control", "no-cache, no-transform")
	c.Set("X-Accel-Buffering", "no")
	c.Set("X-Content-Type-Options", "nosniff")
	c.Set("Connection", "keep-alive")

	// Stream events
	ctx := c.Context()

	// Send connected event
	c.Write([]byte("data: "))
	c.Write([]byte(`{"type":"server.connected","properties":{}}`))
	c.Write([]byte("\n\n"))

	// Heartbeat ticker
	heartbeat := time.NewTicker(10 * time.Second)
	defer heartbeat.Stop()

	// Event ticker (for demo)
	eventTicker := time.NewTicker(5 * time.Second)
	defer eventTicker.Stop()

	for {
		select {
		case <-ctx.Done():
			// Client disconnected
			return nil

		case <-heartbeat.C:
			// Send heartbeat
			c.Write([]byte("data: "))
			c.Write([]byte(`{"type":"server.heartbeat","properties":{}}`))
			c.Write([]byte("\n\n"))

		case <-eventTicker.C:
			// Send sample event (in production, this would be real events)
			data := fmt.Sprintf(`{"type":"server.tick","properties":{"time":%d}}`, time.Now().UnixMilli())
			c.Write([]byte("data: "))
			c.Write([]byte(data))
			c.Write([]byte("\n\n"))

		case <-s.ctx.Done():
			// Server shutting down
			c.Write([]byte("data: "))
			c.Write([]byte(`{"type":"server.disposed","properties":{}}`))
			c.Write([]byte("\n\n"))
			return nil
		}
	}
}

// handleNotFound handles 404 requests.
func handleNotFound(c *fiber.Ctx) error {
	return c.Status(404).JSON(fiber.Map{
		"error":   true,
		"message": "Not found",
		"path":    c.Path(),
	})
}

// handleMethodNotAllowed handles 405 requests.
func handleMethodNotAllowed(c *fiber.Ctx) error {
	return c.Status(405).JSON(fiber.Map{
		"error":   true,
		"message": "Method not allowed",
		"method":  c.Method(),
	})
}

// Version is set at build time.
var Version = "dev"

// BuildTime is set at build time.
var BuildTime = ""

// startTime tracks server start time.
var startTime = time.Now()

// SSEEvent represents a Server-Sent Event.
type SSEEvent struct {
	Type       string
	Properties map[string]interface{}
}

// WriteSSE writes an SSE event to the response.
func WriteSSE(c *fiber.Ctx, event SSEEvent) error {
	data := fmt.Sprintf(`{"type":"%s","properties":%s}`,
		event.Type,
		jsonMarshal(event.Properties))
	c.Write([]byte("data: "))
	c.Write([]byte(data))
	c.Write([]byte("\n\n"))
	return nil
}

// jsonMarshal is a simple JSON marshaler for map.
func jsonMarshal(m map[string]interface{}) string {
	if len(m) == 0 {
		return "{}"
	}

	result := "{"
	first := true
	for k, v := range m {
		if !first {
			result += ","
		}
		first = false

		result += fmt.Sprintf(`"%s":`, k)

		switch val := v.(type) {
		case string:
			result += fmt.Sprintf(`"%s"`, val)
		case int:
			result += fmt.Sprintf(`%d`, val)
		case int64:
			result += fmt.Sprintf(`%d`, val)
		case int32:
			result += fmt.Sprintf(`%d`, val)
		case float64:
			result += fmt.Sprintf(`%f`, val)
		case float32:
			result += fmt.Sprintf(`%f`, val)
		case bool:
			if val {
				result += "true"
			} else {
				result += "false"
			}
		case nil:
			result += "null"
		default:
			result += fmt.Sprintf(`"%v"`, val)
		}
	}
	result += "}"

	return result
}