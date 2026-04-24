// Package lsp provides LSP client implementation.
package lsp

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"sync"
	"time"

	"github.com/rhony08/magicode/internal/util/log"
)

// Client represents an LSP client
type Client struct {
	serverID   ServerID
	conn       *JSONRPCConn
	process    *exec.Cmd
	root       string
	capabilities ServerCapabilities

	// Bounded file tracking (max 50 files)
	files      map[string]*FileState
	filesMu    sync.RWMutex
	maxFiles   int

	// Diagnostics storage (bounded)
	diagnostics map[string][]Diagnostic
	diagsMu     sync.RWMutex

	// Diagnostic event channel
	diagEvents  chan DiagnosticEvent
}

// ClientConfig represents client configuration
type ClientConfig struct {
	ServerID ServerID
	Command  []string
	Args     []string
	Root     string
}

// NewClient creates a new LSP client
func NewClient(cfg ClientConfig) (*Client, error) {
	// Get language mapping
	mapping, ok := LanguageMappings[cfg.ServerID]
	if !ok {
		return nil, fmt.Errorf("unknown server: %s", cfg.ServerID)
	}

	// Build command
	args := mapping.Command
	if len(cfg.Args) > 0 {
		args = append(args, cfg.Args...)
	}

	// Start process
	cmd := exec.Command(args[0], args[1:]...)
	cmd.Dir = cfg.Root

	// Get pipes
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to get stdin: %w", err)
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to get stdout: %w", err)
	}

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, fmt.Errorf("failed to get stderr: %w", err)
	}

	// Start process
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("failed to start server: %w", err)
	}

	// Create JSON-RPC connection
	conn := NewJSONRPCConn(stdin, stdout, stderr)

	client := &Client{
		serverID:    cfg.ServerID,
		conn:        conn,
		process:     cmd,
		root:        cfg.Root,
		files:       make(map[string]*FileState),
		diagnostics: make(map[string][]Diagnostic),
		maxFiles:    MaxTrackedFiles,
		diagEvents:  make(chan DiagnosticEvent, 100),
	}

	// Register diagnostic handler
	conn.OnNotification("textDocument/publishDiagnostics", client.handlePublishDiagnostics)

	return client, nil
}

// Initialize initializes the LSP server
func (c *Client) Initialize(ctx context.Context) error {
	log.Info("Initializing LSP server", "serverID", c.serverID)

	rootURI := URIFromFilepath(c.root)

	params := InitializeParams{
		ProcessID: nil,
		ClientInfo: &ClientInfo{
			Name:    "MagiCode",
			Version: "1.0",
		},
		RootURI:  &rootURI,
		Capabilities: ClientCapabilities{
			TextDocument: &TextDocumentClientCapabilities{
				Synchronization: &TextDocumentSyncClientCapabilities{
					DynamicRegistration: false,
					WillSave:            false,
					WillSaveWaitUntil:   false,
					DidSave:             true,
				},
				PublishDiagnostics: &PublishDiagnosticsClientCapabilities{
					RelatedInformation: true,
					VersionSupport:     false,
					DataSupport:        false,
				},
				Diagnostic: &DiagnosticTextDocumentCapabilities{
					DynamicRegistration: false,
					RelatedDocumentSupport: false,
				},
			},
			Workspace: &WorkspaceClientCapabilities{
				Diagnostic: &DiagnosticWorkspaceCapabilities{
					RefreshSupport: false,
				},
			},
		},
	}

	var result InitializeResult
	err := c.conn.CallWithTimeout("initialize", params, &result, InitializeTimeoutMs)
	if err != nil {
		return fmt.Errorf("initialize failed: %w", err)
	}

	c.capabilities = result.Capabilities

	// Send initialized notification
	if err := c.conn.SendNotification("initialized", nil); err != nil {
		return fmt.Errorf("initialized notification failed: %w", err)
	}

	log.Info("LSP server initialized", "serverName", result.ServerInfo.Name, "serverID", c.serverID)
	return nil
}

// OpenDocument opens a document in the LSP server
func (c *Client) OpenDocument(ctx context.Context, filePath string) error {
	// Check file tracking limit
	if err := c.checkFileLimit(filePath); err != nil {
		return err
	}

	// Read file content
	content, err := os.ReadFile(filePath)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	// Get language
	lang := GetLanguageFromPath(filePath)
	if lang == "" {
		return fmt.Errorf("unknown language for file: %s", filePath)
	}

	// Create document
	uri := URIFromFilepath(filePath)
	doc := TextDocumentItem{
		URI:        uri,
		LanguageID: lang,
		Version:    1,
		Text:       string(content),
	}

	// Track file (bounded)
	c.trackFile(uri, 1)

	// Send didOpen notification
	params := map[string]interface{}{
		"textDocument": doc,
	}

	if err := c.conn.SendNotification("textDocument/didOpen", params); err != nil {
		return fmt.Errorf("didOpen failed: %w", err)
	}

	log.Debug("Opened document", "uri", uri, "serverID", c.serverID)
	return nil
}

// CloseDocument closes a document in the LSP server
func (c *Client) CloseDocument(ctx context.Context, filePath string) error {
	uri := URIFromFilepath(filePath)

	params := map[string]interface{}{
		"textDocument": TextDocumentIdentifier{
			URI: uri,
		},
	}

	if err := c.conn.SendNotification("textDocument/didClose", params); err != nil {
		return fmt.Errorf("didClose failed: %w", err)
	}

	// Untrack file
	c.untrackFile(uri)

	// Clear diagnostics for this file
	c.diagsMu.Lock()
	delete(c.diagnostics, filePath)
	c.diagsMu.Unlock()

	log.Debug("Closed document", "uri", uri, "serverID", c.serverID)
	return nil
}

// ChangeDocument notifies the server of document changes
func (c *Client) ChangeDocument(ctx context.Context, filePath string, content string, version int) error {
	uri := URIFromFilepath(filePath)

	// Get sync kind from capabilities
	syncKind := c.getSyncKind()

	params := map[string]interface{}{
		"textDocument": VersionedTextDocumentIdentifier{
			URI:     uri,
			Version: version,
		},
	}

	switch syncKind {
	case TextDocumentSyncFull:
		params["contentChanges"] = []TextDocumentContentChangeEvent{
			{Text: content},
		}
	case TextDocumentSyncIncremental:
		// For incremental, we need proper ranges
		// This is simplified - full sync for now
		params["contentChanges"] = []TextDocumentContentChangeEvent{
			{Text: content},
		}
	default:
		// None - don't send changes
		return nil
	}

	// Update file tracking
	c.trackFile(uri, version)

	if err := c.conn.SendNotification("textDocument/didChange", params); err != nil {
		return fmt.Errorf("didChange failed: %w", err)
	}

	log.Debug("Changed document", "uri", uri, "version", version, "serverID", c.serverID)
	return nil
}

// GetDiagnostics returns diagnostics for a file
func (c *Client) GetDiagnostics(filePath string) []Diagnostic {
	c.diagsMu.RLock()
	diags := c.diagnostics[filePath]
	c.diagsMu.RUnlock()
	return diags
}

// GetAllDiagnostics returns all diagnostics
func (c *Client) GetAllDiagnostics() map[string][]Diagnostic {
	c.diagsMu.RLock()
	result := make(map[string][]Diagnostic)
	for k, v := range c.diagnostics {
		result[k] = v
	}
	c.diagsMu.RUnlock()
	return result
}

// DiagnosticEvents returns the diagnostic event channel
func (c *Client) DiagnosticEvents() <-chan DiagnosticEvent {
	return c.diagEvents
}

// handlePublishDiagnostics handles publishDiagnostics notifications
func (c *Client) handlePublishDiagnostics(params interface{}) {
	p, ok := params.(map[string]interface{})
	if !ok {
		log.Error("Invalid diagnostics params", "serverID", c.serverID)
		return
	}

	// Parse params
	uri, _ := p["uri"].(string)

	// Parse diagnostics
	diags := []Diagnostic{}
	if d, ok := p["diagnostics"].([]interface{}); ok {
		for _, item := range d {
			if diag, err := parseDiagnostic(item); err == nil {
				diags = append(diags, diag)
			}
		}
	}

	// Convert URI to filepath
	filePath := FilepathFromURI(uri)

	// Store diagnostics
	c.diagsMu.Lock()
	c.diagnostics[filePath] = diags
	c.diagsMu.Unlock()

	// Emit event
	event := DiagnosticEvent{
		ServerID:    c.serverID,
		URI:         uri,
		Diagnostics: diags,
	}

	select {
	case c.diagEvents <- event:
	default:
		// Channel full, drop event (bounded)
		log.Warn("Dropped diagnostic event (channel full)", "serverID", c.serverID)
	}

	log.Debug("Published diagnostics", "uri", uri, "count", len(diags), "serverID", c.serverID)
}

// parseDiagnostic parses a diagnostic from JSON
func parseDiagnostic(item interface{}) (Diagnostic, error) {
	data, err := json.Marshal(item)
	if err != nil {
		return Diagnostic{}, err
	}

	var diag Diagnostic
	if err := json.Unmarshal(data, &diag); err != nil {
		return Diagnostic{}, err
	}

	return diag, nil
}

// Shutdown shuts down the LSP server
func (c *Client) Shutdown(ctx context.Context) error {
	log.Info("Shutting down LSP server", "serverID", c.serverID)

	// Send shutdown request
	_, err := c.conn.SendRequest(ctx, "shutdown", nil)
	if err != nil {
		log.Error("Shutdown request failed", "error", err, "serverID", c.serverID)
	}

	// Send exit notification
	if err := c.conn.SendNotification("exit", nil); err != nil {
		log.Error("Exit notification failed", "error", err, "serverID", c.serverID)
	}

	// Close connection
	c.conn.Close()

	// Wait for process to exit
	if c.process != nil && c.process.Process != nil {
		done := make(chan error, 1)
		go func() {
			done <- c.process.Wait()
		}()

		select {
		case err := <-done:
			log.Debug("Server process exited", "error", err, "serverID", c.serverID)
		case <-time.After(5 * time.Second):
			log.Warn("Server process didn't exit gracefully, killing", "serverID", c.serverID)
			c.process.Process.Kill()
		}
	}

	// Close event channel
	close(c.diagEvents)

	return nil
}

// ServerID returns the server ID
func (c *Client) ServerID() ServerID {
	return c.serverID
}

// Root returns the root directory
func (c *Client) Root() string {
	return c.root
}

// Capabilities returns server capabilities
func (c *Client) Capabilities() ServerCapabilities {
	return c.capabilities
}

// TrackFileCount returns the number of tracked files
func (c *Client) TrackFileCount() int {
	c.filesMu.RLock()
	count := len(c.files)
	c.filesMu.RUnlock()
	return count
}

// checkFileLimit checks if we can track another file
func (c *Client) checkFileLimit(filePath string) error {
	c.filesMu.RLock()
	count := len(c.files)
	c.filesMu.RUnlock()

	if count >= c.maxFiles {
		return fmt.Errorf("max tracked files limit reached (%d)", c.maxFiles)
	}
	return nil
}

// trackFile tracks a file (bounded with eviction)
func (c *Client) trackFile(uri string, version int) {
	c.filesMu.Lock()
	defer c.filesMu.Unlock()

	// Check if we need to evict
	if len(c.files) >= c.maxFiles && c.files[uri] == nil {
		// Evict oldest file
		var oldest string
		var oldestTime time.Time
		for u, state := range c.files {
			if oldest == "" || state.LastAccess.Before(oldestTime) {
				oldest = u
				oldestTime = state.LastAccess
			}
		}
		if oldest != "" {
			delete(c.files, oldest)
			log.Debug("Evicted oldest tracked file", "uri", oldest, "serverID", c.serverID)
		}
	}

	// Add/update file
	c.files[uri] = &FileState{
		URI:        uri,
		Version:    version,
		LastAccess: time.Now(),
	}
}

// untrackFile removes a file from tracking
func (c *Client) untrackFile(uri string) {
	c.filesMu.Lock()
	delete(c.files, uri)
	c.filesMu.Unlock()
}

// getSyncKind returns the text document sync kind
func (c *Client) getSyncKind() int {
	sync := c.capabilities.TextDocumentSync
	if sync == nil {
		return TextDocumentSyncNone
	}

	// Can be int or object
	switch v := sync.(type) {
	case int:
		return v
	case float64:
		return int(v)
	case map[string]interface{}:
		if change, ok := v["change"]; ok {
			switch c := change.(type) {
			case int:
				return c
			case float64:
				return int(c)
			}
		}
	}

	return TextDocumentSyncFull
}