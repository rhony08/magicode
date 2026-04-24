package server

import (
	"encoding/json"
	"fmt"
	"log"
	"sync"

	"github.com/fasthttp/websocket"
	"github.com/gofiber/fiber/v2"
	"github.com/opencode-ai/opencode-go/internal/pty"
	"github.com/valyala/fasthttp"
)

// WebSocketUpgrader is the WebSocket upgrader.
var WebSocketUpgrader = websocket.FastHTTPUpgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(ctx *fasthttp.RequestCtx) bool {
		// Allow all origins for development
		// In production, this should check the origin
		return true
	},
}

// WSConnection represents a WebSocket connection wrapper.
type WSConnection struct {
	conn      *websocket.Conn
	mu        sync.Mutex
	sessionID string
	closed    bool
}

// NewWSConnection creates a new WebSocket connection wrapper.
func NewWSConnection(conn *websocket.Conn, sessionID string) *WSConnection {
	return &WSConnection{
		conn:      conn,
		sessionID: sessionID,
	}
}

// Send sends data to the WebSocket connection.
func (ws *WSConnection) Send(data []byte) error {
	ws.mu.Lock()
	defer ws.mu.Unlock()

	if ws.closed {
		return fmt.Errorf("connection closed")
	}

	return ws.conn.WriteMessage(websocket.TextMessage, data)
}

// Close closes the WebSocket connection.
func (ws *WSConnection) Close(code int, reason string) error {
	ws.mu.Lock()
	defer ws.mu.Unlock()

	if ws.closed {
		return nil
	}

	ws.closed = true
	return ws.conn.Close()
}

// ReadyState returns the WebSocket ready state (simulated).
func (ws *WSConnection) ReadyState() int {
	ws.mu.Lock()
	defer ws.mu.Unlock()

	if ws.closed {
		return pty.WebSocketClosed
	}
	return pty.WebSocketOpen
}

// ReadMessage reads a message from the WebSocket.
func (ws *WSConnection) ReadMessage() (int, []byte, error) {
	return ws.conn.ReadMessage()
}

// WebSocketMessage represents a WebSocket message.
type WebSocketMessage struct {
	Type    string          `json:"type"`
	Data    json.RawMessage `json:"data,omitempty"`
	Message string          `json:"message,omitempty"`
}

// WebSocket close codes.
const (
	CloseNormalClosure   = 1000
	CloseGoingAway       = 1001
	CloseProtocolError   = 1002
	CloseUnsupportedData = 1003
	CloseAbnormalClosure = 1006
	CloseInternalError   = 1011
)

// Ensure WSConnection implements Socket interface
var _ pty.Socket = (*WSConnection)(nil)

// handlePTYWebSocketUpgrade handles WebSocket upgrade for PTY.
// This is a placeholder implementation that returns 501.
// Full WebSocket implementation requires integrating with the PTY manager.
func (s *Server) handlePTYWebSocketUpgrade(c *fiber.Ctx) error {
	sessionID := c.Params("id")
	
	// Log the WebSocket attempt for debugging
	log.Printf("WebSocket upgrade requested for PTY session: %s", sessionID)

	// For now, return 501 - WebSocket implementation pending
	// Full implementation would:
	// 1. Use websocket.New() handler
	// 2. Connect to PTY session via manager
	// 3. Stream output to WebSocket
	// 4. Write WebSocket input to PTY
	return c.Status(501).JSON(fiber.Map{
		"error":    true,
		"message":  "WebSocket for PTY not yet implemented",
		"session_id": sessionID,
		"hint":     "Use direct terminal access or wait for WebSocket support",
	})
}

// WebSocketHandler creates a WebSocket handler function.
// This helper can be used when WebSocket is fully implemented.
func WebSocketHandler(onConnect func(*WSConnection)) fiber.Handler {
	return func(c *fiber.Ctx) error {
		// WebSocket upgrade logic placeholder
		return c.Status(501).JSON(fiber.Map{
			"error":   true,
			"message": "WebSocket handler placeholder",
		})
	}
}