// Package lsp provides tests for LSP package.
package lsp

import (
	"testing"
)

// TestServerIDConstants tests server ID constants
func TestServerIDConstants(t *testing.T) {
	ids := []ServerID{
		ServerTypeScript,
		ServerGo,
		ServerPython,
		ServerRust,
		ServerJava,
		ServerC,
		ServerCpp,
	}

	expected := []string{
		"typescript",
		"go",
		"python",
		"rust",
		"java",
		"c",
		"cpp",
	}

	for i, id := range ids {
		if string(id) != expected[i] {
			t.Errorf("ServerID %d: expected %s, got %s", i, expected[i], id)
		}
	}
}

// TestLanguageMappings tests language mappings
func TestLanguageMappings(t *testing.T) {
	if len(LanguageMappings) == 0 {
		t.Error("LanguageMappings should not be empty")
	}

	// Check TypeScript mapping
	tsMapping, ok := LanguageMappings[ServerTypeScript]
	if !ok {
		t.Error("TypeScript mapping should exist")
	}
	if tsMapping.LanguageID != "typescript" {
		t.Errorf("Expected languageID 'typescript', got '%s'", tsMapping.LanguageID)
	}
	if len(tsMapping.Extensions) == 0 {
		t.Error("TypeScript mapping should have extensions")
	}

	// Check Go mapping
	goMapping, ok := LanguageMappings[ServerGo]
	if !ok {
		t.Error("Go mapping should exist")
	}
	if goMapping.LanguageID != "go" {
		t.Errorf("Expected languageID 'go', got '%s'", goMapping.LanguageID)
	}
	if len(goMapping.Command) == 0 {
		t.Error("Go mapping should have command")
	}
}

// TestGetLanguageFromPath tests language detection from paths
func TestGetLanguageFromPath(t *testing.T) {
	tests := []struct {
		path     string
		expected string
	}{
		{"test.ts", "typescript"},
		{"test.tsx", "typescript"},
		{"test.js", "typescript"},
		{"test.go", "go"},
		{"test.py", "python"},
		{"test.rs", "rust"},
		{"test.java", "java"},
		{"test.c", "c"},
		{"test.cpp", "cpp"},
		{"test.unknown", ""},
	}

	for _, tt := range tests {
		result := GetLanguageFromPath(tt.path)
		if result != tt.expected {
			t.Errorf("GetLanguageFromPath(%s): expected '%s', got '%s'", tt.path, tt.expected, result)
		}
	}
}

// TestGetServerFromPath tests server detection from paths
func TestGetServerFromPath(t *testing.T) {
	tests := []struct {
		path        string
		expectedID  ServerID
		expectFound bool
	}{
		{"test.ts", ServerTypeScript, true},
		{"test.go", ServerGo, true},
		{"test.py", ServerPython, true},
		{"test.unknown", "", false},
	}

	for _, tt := range tests {
		mapping := GetServerFromPath(tt.path)
		if tt.expectFound {
			if mapping == nil {
				t.Errorf("GetServerFromPath(%s): expected to find server", tt.path)
			} else if mapping.ServerID != tt.expectedID {
				t.Errorf("GetServerFromPath(%s): expected serverID '%s', got '%s'", tt.path, tt.expectedID, mapping.ServerID)
			}
		} else {
			if mapping != nil {
				t.Errorf("GetServerFromPath(%s): expected nil, got serverID '%s'", tt.path, mapping.ServerID)
			}
		}
	}
}

// TestGetExtensions tests extension listing
func TestGetExtensions(t *testing.T) {
	exts := GetExtensions()
	if len(exts) == 0 {
		t.Error("GetExtensions should return non-empty list")
	}

	// Check for known extensions
	found := false
	for _, ext := range exts {
		if ext == ".go" {
			found = true
			break
		}
	}
	if !found {
		t.Error("Extensions should include .go")
	}
}

// TestGetLanguageIDs tests language ID listing
func TestGetLanguageIDs(t *testing.T) {
	langs := GetLanguageIDs()
	if len(langs) == 0 {
		t.Error("GetLanguageIDs should return non-empty list")
	}
}

// TestIsLanguageFile tests language file detection
func TestIsLanguageFile(t *testing.T) {
	tests := []struct {
		path     string
		expected bool
	}{
		{"test.go", true},
		{"test.ts", true},
		{"test.py", true},
		{"test.txt", false},
		{"test.unknown", false},
	}

	for _, tt := range tests {
		result := IsLanguageFile(tt.path)
		if result != tt.expected {
			t.Errorf("IsLanguageFile(%s): expected %v, got %v", tt.path, tt.expected, result)
		}
	}
}

// TestURIFromFilepath tests URI conversion
func TestURIFromFilepath(t *testing.T) {
	tests := []struct {
		path     string
		expected string
	}{
		{"/home/user/test.go", "file:///home/user/test.go"},
		{"/tmp/file.txt", "file:///tmp/file.txt"},
	}

	for _, tt := range tests {
		result := URIFromFilepath(tt.path)
		if result != tt.expected {
			t.Errorf("URIFromFilepath(%s): expected '%s', got '%s'", tt.path, tt.expected, result)
		}
	}
}

// TestFilepathFromURI tests filepath conversion
func TestFilepathFromURI(t *testing.T) {
	tests := []struct {
		uri      string
		expected string
	}{
		{"file:///home/user/test.go", "/home/user/test.go"},
		{"file:///tmp/file.txt", "/tmp/file.txt"},
		{"http://example.com", ""},
		{"invalid-uri", ""},
	}

	for _, tt := range tests {
		result := FilepathFromURI(tt.uri)
		if result != tt.expected {
			t.Errorf("FilepathFromURI(%s): expected '%s', got '%s'", tt.uri, tt.expected, result)
		}
	}
}

// TestMatchesExtension tests extension matching
func TestMatchesExtension(t *testing.T) {
	tests := []struct {
		path       string
		extensions []string
		expected   bool
	}{
		{"test.go", []string{".go", ".ts"}, true},
		{"test.ts", []string{".go", ".py"}, false},
		{"TEST.GO", []string{".go"}, true}, // case insensitive
	}

	for _, tt := range tests {
		result := MatchesExtension(tt.path, tt.extensions)
		if result != tt.expected {
			t.Errorf("MatchesExtension(%s, %v): expected %v, got %v", tt.path, tt.extensions, tt.expected, result)
		}
	}
}

// TestDiagnosticSeverity tests diagnostic severity constants
func TestDiagnosticSeverity(t *testing.T) {
	if SeverityError != 1 {
		t.Errorf("SeverityError should be 1, got %d", SeverityError)
	}
	if SeverityWarning != 2 {
		t.Errorf("SeverityWarning should be 2, got %d", SeverityWarning)
	}
	if SeverityInformation != 3 {
		t.Errorf("SeverityInformation should be 3, got %d", SeverityInformation)
	}
	if SeverityHint != 4 {
		t.Errorf("SeverityHint should be 4, got %d", SeverityHint)
	}
}

// TestConstants tests LSP constants
func TestConstants(t *testing.T) {
	if MaxTrackedFiles != 50 {
		t.Errorf("MaxTrackedFiles should be 50, got %d", MaxTrackedFiles)
	}
	if InitializeTimeoutMs != 45000 {
		t.Errorf("InitializeTimeoutMs should be 45000, got %d", InitializeTimeoutMs)
	}
	if FileChangeCreated != 1 {
		t.Errorf("FileChangeCreated should be 1, got %d", FileChangeCreated)
	}
	if TextDocumentSyncIncremental != 2 {
		t.Errorf("TextDocumentSyncIncremental should be 2, got %d", TextDocumentSyncIncremental)
	}
}

// TestPosition tests position struct
func TestPosition(t *testing.T) {
	pos := Position{Line: 10, Character: 5}
	if pos.Line != 10 {
		t.Errorf("Position.Line should be 10, got %d", pos.Line)
	}
	if pos.Character != 5 {
		t.Errorf("Position.Character should be 5, got %d", pos.Character)
	}
}

// TestRange tests range struct
func TestRange(t *testing.T) {
	rng := Range{
		Start: Position{Line: 0, Character: 0},
		End:   Position{Line: 10, Character: 5},
	}
	if rng.Start.Line != 0 {
		t.Errorf("Range.Start.Line should be 0, got %d", rng.Start.Line)
	}
	if rng.End.Line != 10 {
		t.Errorf("Range.End.Line should be 10, got %d", rng.End.Line)
	}
}

// TestDiagnostic tests diagnostic struct
func TestDiagnostic(t *testing.T) {
	diag := Diagnostic{
		Range: Range{
			Start: Position{Line: 0, Character: 0},
			End:   Position{Line: 1, Character: 0},
		},
		Message:  "Test error",
		Severity: SeverityError,
		Source:   "test",
	}

	if diag.Message != "Test error" {
		t.Errorf("Diagnostic.Message should be 'Test error', got '%s'", diag.Message)
	}
	if diag.Severity != SeverityError {
		t.Errorf("Diagnostic.Severity should be SeverityError, got %d", diag.Severity)
	}
}

// TestTextDocumentItem tests text document item struct
func TestTextDocumentItem(t *testing.T) {
	doc := TextDocumentItem{
		URI:        "file:///test.go",
		LanguageID: "go",
		Version:    1,
		Text:       "package main",
	}

	if doc.URI != "file:///test.go" {
		t.Errorf("TextDocumentItem.URI incorrect")
	}
	if doc.LanguageID != "go" {
		t.Errorf("TextDocumentItem.LanguageID should be 'go', got '%s'", doc.LanguageID)
	}
}

// TestManager tests LSP manager creation
func TestManager(t *testing.T) {
	m := NewManager("/tmp")
	if m == nil {
		t.Error("NewManager should return non-nil manager")
	}
	if m.root != "/tmp" {
		t.Errorf("Manager.root should be '/tmp', got '%s'", m.root)
	}
	if len(m.Running()) != 0 {
		t.Error("New manager should have no running servers")
	}
}

// TestManagerRunning tests running server tracking
func TestManagerRunning(t *testing.T) {
	m := NewManager("/tmp")

	// Initially no servers running
	if m.IsRunning(ServerGo) {
		t.Error("ServerGo should not be running initially")
	}
	if m.ClientCount() != 0 {
		t.Error("ClientCount should be 0 initially")
	}
}

// TestDiagnosticEvent tests diagnostic event struct
func TestDiagnosticEvent(t *testing.T) {
	event := DiagnosticEvent{
		ServerID: ServerGo,
		URI:      "file:///test.go",
		Diagnostics: []Diagnostic{
			{Message: "Test error", Severity: SeverityError},
		},
	}

	if event.ServerID != ServerGo {
		t.Errorf("DiagnosticEvent.ServerID should be ServerGo")
	}
	if len(event.Diagnostics) != 1 {
		t.Errorf("DiagnosticEvent should have 1 diagnostic, got %d", len(event.Diagnostics))
	}
}

// TestJSONRPCRequest tests JSON-RPC request struct
func TestJSONRPCRequest(t *testing.T) {
	req := JSONRPCRequest{
		JSONRPC: "2.0",
		ID:      1,
		Method:  "initialize",
		Params:  nil,
	}

	if req.JSONRPC != "2.0" {
		t.Errorf("JSONRPCRequest.JSONRPC should be '2.0'")
	}
	if req.Method != "initialize" {
		t.Errorf("JSONRPCRequest.Method should be 'initialize'")
	}
}

// TestJSONRPCResponse tests JSON-RPC response struct
func TestJSONRPCResponse(t *testing.T) {
	resp := JSONRPCResponse{
		JSONRPC: "2.0",
		ID:      1,
		Result:  map[string]interface{}{"capabilities": nil},
	}

	if resp.JSONRPC != "2.0" {
		t.Errorf("JSONRPCResponse.JSONRPC should be '2.0'")
	}
	if resp.Error != nil {
		t.Error("JSONRPCResponse.Error should be nil for successful response")
	}
}

// TestJSONRPCError tests JSON-RPC error struct
func TestJSONRPCError(t *testing.T) {
	err := JSONRPCError{
		Code:    -32600,
		Message: "Invalid Request",
	}

	if err.Code != -32600 {
		t.Errorf("JSONRPCError.Code should be -32600, got %d", err.Code)
	}
	if err.Message != "Invalid Request" {
		t.Errorf("JSONRPCError.Message should be 'Invalid Request'")
	}
}

// TestJSONRPCNotification tests JSON-RPC notification struct
func TestJSONRPCNotification(t *testing.T) {
	notif := JSONRPCNotification{
		JSONRPC: "2.0",
		Method:  "textDocument/publishDiagnostics",
		Params:  nil,
	}

	if notif.JSONRPC != "2.0" {
		t.Errorf("JSONRPCNotification.JSONRPC should be '2.0'")
	}
	if notif.Method != "textDocument/publishDiagnostics" {
		t.Errorf("JSONRPCNotification.Method incorrect")
	}
}