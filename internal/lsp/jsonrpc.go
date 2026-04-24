// Package lsp provides JSON-RPC 2.0 client implementation.
package lsp

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/rhony08/magicode/internal/util/log"
)

// JSONRPCRequest represents a JSON-RPC request
type JSONRPCRequest struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      interface{} `json:"id"` // int or string
	Method  string      `json:"method"`
	Params  interface{} `json:"params,omitempty"`
}

// JSONRPCResponse represents a JSON-RPC response
type JSONRPCResponse struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      interface{} `json:"id"`
	Result  interface{} `json:"result,omitempty"`
	Error   *JSONRPCError `json:"error,omitempty"`
}

// JSONRPCError represents a JSON-RPC error
type JSONRPCError struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

// JSONRPCNotification represents a JSON-RPC notification
type JSONRPCNotification struct {
	JSONRPC string      `json:"jsonrpc"`
	Method  string      `json:"method"`
	Params  interface{} `json:"params,omitempty"`
}

// JSONRPCConn represents a JSON-RPC connection
type JSONRPCConn struct {
	stdin     io.Writer
	stdout    io.Reader
	stderr    io.Reader

	requests  map[interface{}]chan *JSONRPCResponse
	nextID    atomic.Int64
	mu        sync.RWMutex
	ctx       context.Context
	cancel    context.CancelFunc

	// Notification handlers
	notificationHandlers map[string]func(interface{})
	notificationMu       sync.RWMutex

	// Logger
	logger log.Logger
}

// NewJSONRPCConn creates a new JSON-RPC connection
func NewJSONRPCConn(stdin io.Writer, stdout, stderr io.Reader) *JSONRPCConn {
	ctx, cancel := context.WithCancel(context.Background())

	conn := &JSONRPCConn{
		stdin:               stdin,
		stdout:              stdout,
		stderr:              stderr,
		requests:            make(map[interface{}]chan *JSONRPCResponse),
		notificationHandlers: make(map[string]func(interface{})),
		ctx:                 ctx,
		cancel:              cancel,
	}

	// Start reading responses
	go conn.readLoop()
	go conn.stderrLoop()

	return conn
}

// Close closes the connection
func (c *JSONRPCConn) Close() {
	c.cancel()
	c.mu.Lock()
	for id, ch := range c.requests {
		close(ch)
		delete(c.requests, id)
	}
	c.mu.Unlock()
}

// nextRequestID generates a new request ID
func (c *JSONRPCConn) nextRequestID() int {
	return int(c.nextID.Add(1))
}

// SendRequest sends a request and waits for a response
func (c *JSONRPCConn) SendRequest(ctx context.Context, method string, params interface{}) (*JSONRPCResponse, error) {
	id := c.nextRequestID()

	req := JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      id,
		Method:  method,
		Params:  params,
	}

	// Register response channel before sending
	ch := make(chan *JSONRPCResponse, 1)
	c.mu.Lock()
	c.requests[id] = ch
	c.mu.Unlock()

	// Send request
	if err := c.writeRequest(req); err != nil {
		c.mu.Lock()
		delete(c.requests, id)
		c.mu.Unlock()
		return nil, fmt.Errorf("failed to send request: %w", err)
	}

	// Wait for response with timeout
	select {
	case resp := <-ch:
		if resp == nil {
			return nil, fmt.Errorf("connection closed")
		}
		if resp.Error != nil {
			return nil, fmt.Errorf("JSON-RPC error %d: %s", resp.Error.Code, resp.Error.Message)
		}
		return resp, nil
	case <-ctx.Done():
		c.mu.Lock()
		delete(c.requests, id)
		c.mu.Unlock()
		return nil, ctx.Err()
	case <-c.ctx.Done():
		c.mu.Lock()
		delete(c.requests, id)
		c.mu.Unlock()
		return nil, fmt.Errorf("connection closed")
	}
}

// SendNotification sends a notification (no response expected)
func (c *JSONRPCConn) SendNotification(method string, params interface{}) error {
	req := JSONRPCRequest{
		JSONRPC: "2.0",
		Method:  method,
		Params:  params,
	}
	// No ID for notifications
	req.ID = nil

	return c.writeRequest(req)
}

// writeRequest writes a request to stdin
func (c *JSONRPCConn) writeRequest(req JSONRPCRequest) error {
	data, err := json.Marshal(req)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %w", err)
	}

	// Write Content-Length header + body
	header := fmt.Sprintf("Content-Length: %d\r\n\r\n", len(data))
	if _, err := c.stdin.Write([]byte(header)); err != nil {
		return fmt.Errorf("failed to write header: %w", err)
	}
	if _, err := c.stdin.Write(data); err != nil {
		return fmt.Errorf("failed to write body: %w", err)
	}

	return nil
}

// readLoop reads responses and notifications from stdout
func (c *JSONRPCConn) readLoop() {
	scanner := bufio.NewScanner(c.stdout)
	scanner.Buffer(make([]byte, 64*1024), 1024*1024) // 1MB max message

	for scanner.Scan() {
		// Check for context cancellation
		if c.ctx.Err() != nil {
			return
		}

		line := scanner.Text()

		// Parse Content-Length header
		if !strings.HasPrefix(line, "Content-Length:") {
			continue
		}

		// Extract length
		lengthStr := strings.TrimSpace(strings.TrimPrefix(line, "Content-Length:"))
		length := 0
		for _, ch := range lengthStr {
			if ch >= '0' && ch <= '9' {
				length = length * 10 + int(ch - '0')
			} else {
				break
			}
		}

		// Read empty line after header
		if !scanner.Scan() {
			break
		}

		// Read body
		if !scanner.Scan() {
			break
		}

		body := []byte(scanner.Text())
		if len(body) < length {
			// Need to read more
			remaining := length - len(body)
			buf := make([]byte, remaining)
			n, err := io.ReadFull(c.stdout, buf)
			if err != nil {
				log.Error("Failed to read message body", "error", err.Error())
				continue
			}
			body = append(body, buf[:n]...)
		}

		// Parse message
		c.handleMessage(body)
	}

	if err := scanner.Err(); err != nil && c.ctx.Err() == nil {
		log.Error("Scanner error", "error", err.Error())
	}
}

// handleMessage handles an incoming JSON-RPC message
func (c *JSONRPCConn) handleMessage(data []byte) {
	// Try to parse as response first
	var resp JSONRPCResponse
	if err := json.Unmarshal(data, &resp); err == nil && resp.ID != nil {
		c.handleResponse(&resp)
		return
	}

	// Try to parse as notification
	var notif JSONRPCNotification
	if err := json.Unmarshal(data, &notif); err == nil {
		c.handleNotification(&notif)
		return
	}

	log.Error("Failed to parse message", "data", string(data))
}

// handleResponse handles an incoming response
func (c *JSONRPCConn) handleResponse(resp *JSONRPCResponse) {
	c.mu.RLock()
	ch, ok := c.requests[resp.ID]
	c.mu.RUnlock()

	if !ok {
		log.Error("Received response for unknown request", "id", resp.ID)
		return
	}

	ch <- resp

	// Cleanup
	c.mu.Lock()
	delete(c.requests, resp.ID)
	c.mu.Unlock()
}

// handleNotification handles an incoming notification
func (c *JSONRPCConn) handleNotification(notif *JSONRPCNotification) {
	c.notificationMu.RLock()
	handler, ok := c.notificationHandlers[notif.Method]
	c.notificationMu.RUnlock()

	if ok {
		handler(notif.Params)
	} else {
		log.Debug("Unhandled notification", "method", notif.Method)
	}
}

// stderrLoop reads stderr from the LSP server
func (c *JSONRPCConn) stderrLoop() {
	scanner := bufio.NewScanner(c.stderr)
	for scanner.Scan() {
		line := scanner.Text()
		log.Debug("Server stderr", "message", line)
	}
}

// OnNotification registers a handler for a notification method
func (c *JSONRPCConn) OnNotification(method string, handler func(interface{})) {
	c.notificationMu.Lock()
	c.notificationHandlers[method] = handler
	c.notificationMu.Unlock()
}

// Call is a convenience method for typed requests
func (c *JSONRPCConn) Call(ctx context.Context, method string, params interface{}, result interface{}) error {
	resp, err := c.SendRequest(ctx, method, params)
	if err != nil {
		return err
	}

	if result != nil && resp.Result != nil {
		data, err := json.Marshal(resp.Result)
		if err != nil {
			return fmt.Errorf("failed to marshal result: %w", err)
		}
		if err := json.Unmarshal(data, result); err != nil {
			return fmt.Errorf("failed to unmarshal result: %w", err)
		}
	}

	return nil
}

// CallWithTimeout calls a method with a timeout
func (c *JSONRPCConn) CallWithTimeout(method string, params interface{}, result interface{}, timeoutMs int) error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Duration(timeoutMs)*time.Millisecond)
	defer cancel()
	return c.Call(ctx, method, params, result)
}