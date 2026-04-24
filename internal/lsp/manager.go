// Package lsp provides LSP manager for handling multiple clients.
package lsp

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// Manager manages multiple LSP clients
type Manager struct {
	clients map[ServerID]*Client
	mu      sync.RWMutex
	root    string

	// Diagnostic aggregation
	allDiags chan DiagnosticEvent
}

// NewManager creates a new LSP manager
func NewManager(root string) *Manager {
	m := &Manager{
		clients: make(map[ServerID]*Client),
		root:    root,
		allDiags: make(chan DiagnosticEvent, 200),
	}
	return m
}

// Start starts an LSP server for a language
func (m *Manager) Start(ctx context.Context, serverID ServerID) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	// Check if already running
	if _, ok := m.clients[serverID]; ok {
		return nil // Already running
	}

	// Get language mapping
	mapping, ok := LanguageMappings[serverID]
	if !ok {
		return fmt.Errorf("unknown server: %s", serverID)
	}

	// Create client
	client, err := NewClient(ClientConfig{
		ServerID: serverID,
		Command:  mapping.Command,
		Root:     m.root,
	})
	if err != nil {
		return fmt.Errorf("failed to create client: %w", err)
	}

	// Initialize
	if err := client.Initialize(ctx); err != nil {
		client.Shutdown(ctx)
		return fmt.Errorf("failed to initialize: %w", err)
	}

	// Forward diagnostic events
	go m.forwardDiagnostics(client)

	m.clients[serverID] = client
	return nil
}

// Stop stops an LSP server
func (m *Manager) Stop(ctx context.Context, serverID ServerID) error {
	m.mu.Lock()
	client, ok := m.clients[serverID]
	if ok {
		delete(m.clients, serverID)
	}
	m.mu.Unlock()

	if !ok {
		return nil
	}

	return client.Shutdown(ctx)
}

// StopAll stops all LSP servers
func (m *Manager) StopAll(ctx context.Context) error {
	m.mu.RLock()
	ids := make([]ServerID, 0, len(m.clients))
	for id := range m.clients {
		ids = append(ids, id)
	}
	m.mu.RUnlock()

	for _, id := range ids {
		if err := m.Stop(ctx, id); err != nil {
			return err
		}
	}
	return nil
}

// Get returns a client by server ID
func (m *Manager) Get(serverID ServerID) (*Client, bool) {
	m.mu.RLock()
	client, ok := m.clients[serverID]
	m.mu.RUnlock()
	return client, ok
}

// OpenFile opens a file in the appropriate LSP server
func (m *Manager) OpenFile(ctx context.Context, filePath string) error {
	mapping := GetServerFromPath(filePath)
	if mapping == nil {
		return fmt.Errorf("no LSP server for file: %s", filePath)
	}

	// Ensure server is running
	if err := m.Start(ctx, mapping.ServerID); err != nil {
		return err
	}

	// Open document
	client, ok := m.Get(mapping.ServerID)
	if !ok {
		return fmt.Errorf("server not running: %s", mapping.ServerID)
	}

	return client.OpenDocument(ctx, filePath)
}

// CloseFile closes a file in its LSP server
func (m *Manager) CloseFile(ctx context.Context, filePath string) error {
	mapping := GetServerFromPath(filePath)
	if mapping == nil {
		return nil
	}

	client, ok := m.Get(mapping.ServerID)
	if !ok {
		return nil
	}

	return client.CloseDocument(ctx, filePath)
}

// ChangeFile notifies servers of file changes
func (m *Manager) ChangeFile(ctx context.Context, filePath string, content string, version int) error {
	mapping := GetServerFromPath(filePath)
	if mapping == nil {
		return nil
	}

	client, ok := m.Get(mapping.ServerID)
	if !ok {
		return nil
	}

	return client.ChangeDocument(ctx, filePath, content, version)
}

// GetDiagnostics returns diagnostics for a file
func (m *Manager) GetDiagnostics(filePath string) []Diagnostic {
	mapping := GetServerFromPath(filePath)
	if mapping == nil {
		return nil
	}

	client, ok := m.Get(mapping.ServerID)
	if !ok {
		return nil
	}

	return client.GetDiagnostics(filePath)
}

// GetAllDiagnostics returns all diagnostics from all servers
func (m *Manager) GetAllDiagnostics() map[string][]Diagnostic {
	m.mu.RLock()
	defer m.mu.RUnlock()

	result := make(map[string][]Diagnostic)
	for _, client := range m.clients {
		for file, diags := range client.GetAllDiagnostics() {
			result[file] = diags
		}
	}
	return result
}

// DiagnosticEvents returns aggregated diagnostic events from all servers
func (m *Manager) DiagnosticEvents() <-chan DiagnosticEvent {
	return m.allDiags
}

// forwardDiagnostics forwards diagnostics from a client
func (m *Manager) forwardDiagnostics(client *Client) {
	for event := range client.DiagnosticEvents() {
		select {
		case m.allDiags <- event:
		default:
			// Channel full, drop event
		}
	}
}

// Running returns the list of running servers
func (m *Manager) Running() []ServerID {
	m.mu.RLock()
	defer m.mu.RUnlock()

	ids := make([]ServerID, 0, len(m.clients))
	for id := range m.clients {
		ids = append(ids, id)
	}
	return ids
}

// IsRunning checks if a server is running
func (m *Manager) IsRunning(serverID ServerID) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.clients[serverID] != nil
}

// ClientCount returns the number of running clients
func (m *Manager) ClientCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.clients)
}

// TrackedFileCount returns the total number of tracked files
func (m *Manager) TrackedFileCount() int {
	m.mu.RLock()
	defer m.mu.RUnlock()

	total := 0
	for _, client := range m.clients {
		total += client.TrackFileCount()
	}
	return total
}

// CleanupStaleFiles removes files not accessed recently
func (m *Manager) CleanupStaleFiles(maxAge time.Duration) int {
	m.mu.RLock()
	defer m.mu.RUnlock()

	total := 0
	for _, client := range m.clients {
		// This would need to be implemented in Client
		// For now, just count
		total += client.TrackFileCount()
	}
	return total
}

// Close closes all clients and the diagnostic channel
func (m *Manager) Close() error {
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	
	err := m.StopAll(ctx)
	close(m.allDiags)
	return err
}