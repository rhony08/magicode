package server

import (
	"net/http/httptest"
	"testing"

	"github.com/gofiber/fiber/v2"
)

// TestNewServer tests server creation.
func TestNewServer(t *testing.T) {
	cfg := DefaultConfig()
	s := New("/tmp/test", cfg)

	if s == nil {
		t.Error("Server should not be nil")
	}
	if s.directory != "/tmp/test" {
		t.Error("Directory should be '/tmp/test'")
	}
	if s.config.Port != DefaultConfig().Port {
		t.Error("Port should be default")
	}
	if s.config.Hostname != DefaultConfig().Hostname {
		t.Error("Hostname should be default")
	}
}

// TestDefaultConfig tests default configuration.
func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if cfg.Port != 3000 {
		t.Errorf("Port should be 3000, got %d", cfg.Port)
	}
	if cfg.Hostname != "localhost" {
		t.Errorf("Hostname should be 'localhost', got %s", cfg.Hostname)
	}
	if !cfg.Compression {
		t.Error("Compression should be true")
	}
}

// TestServerConfig tests server configuration.
func TestServerConfig(t *testing.T) {
	cfg := Config{
		Port:        4000,
		Hostname:    "example.com",
		CORS:        []string{"http://localhost:3000"},
		Compression: false,
	}

	s := New("/tmp/test", cfg)
	if s.config.Port != 4000 {
		t.Errorf("Port should be 4000, got %d", s.config.Port)
	}
	if s.config.Hostname != "example.com" {
		t.Errorf("Hostname should be 'example.com', got %s", s.config.Hostname)
	}
}

// TestHealthEndpoint tests health endpoint.
func TestHealthEndpoint(t *testing.T) {
	s := New("/tmp/test", DefaultConfig())

	req := httptest.NewRequest("GET", "/global/health", nil)
	resp, err := s.app.Test(req, 1000)
	if err != nil {
		t.Errorf("Health request failed: %v", err)
	}

	if resp.StatusCode != 200 {
		t.Errorf("Health should return 200, got %d", resp.StatusCode)
	}
}

// TestVersionEndpoint tests version endpoint.
func TestVersionEndpoint(t *testing.T) {
	s := New("/tmp/test", DefaultConfig())

	req := httptest.NewRequest("GET", "/global/version", nil)
	resp, err := s.app.Test(req, 1000)
	if err != nil {
		t.Errorf("Version request failed: %v", err)
	}

	if resp.StatusCode != 200 {
		t.Errorf("Version should return 200, got %d", resp.StatusCode)
	}
}

// TestSessionRoutes tests session routes.
func TestSessionRoutes(t *testing.T) {
	s := New("/tmp/test", DefaultConfig())

	// Test list sessions
	req := httptest.NewRequest("GET", "/session", nil)
	resp, err := s.app.Test(req, 1000)
	if err != nil {
		t.Errorf("List sessions request failed: %v", err)
	}
	if resp.StatusCode != 200 {
		t.Errorf("List sessions should return 200, got %d", resp.StatusCode)
	}

	// Test create session
	req = httptest.NewRequest("POST", "/session", nil)
	resp, err = s.app.Test(req, 1000)
	if err != nil {
		t.Errorf("Create session request failed: %v", err)
	}
	if resp.StatusCode != 200 {
		t.Errorf("Create session should return 200, got %d", resp.StatusCode)
	}

	// Test get session
	req = httptest.NewRequest("GET", "/session/test-id", nil)
	resp, err = s.app.Test(req, 1000)
	if err != nil {
		t.Errorf("Get session request failed: %v", err)
	}
	if resp.StatusCode != 200 {
		t.Errorf("Get session should return 200, got %d", resp.StatusCode)
	}

	// Test delete session
	req = httptest.NewRequest("DELETE", "/session/test-id", nil)
	resp, err = s.app.Test(req, 1000)
	if err != nil {
		t.Errorf("Delete session request failed: %v", err)
	}
	if resp.StatusCode != 200 {
		t.Errorf("Delete session should return 200, got %d", resp.StatusCode)
	}
}

// TestPTYRoutes tests PTY routes.
func TestPTYRoutes(t *testing.T) {
	s := New("/tmp/test", DefaultConfig())

	// Test list PTY
	req := httptest.NewRequest("GET", "/pty", nil)
	resp, err := s.app.Test(req, 1000)
	if err != nil {
		t.Errorf("List PTY request failed: %v", err)
	}
	if resp.StatusCode != 200 {
		t.Errorf("List PTY should return 200, got %d", resp.StatusCode)
	}

	// Test create PTY
	req = httptest.NewRequest("POST", "/pty", nil)
	resp, err = s.app.Test(req, 1000)
	if err != nil {
		t.Errorf("Create PTY request failed: %v", err)
	}
	if resp.StatusCode != 200 {
		t.Errorf("Create PTY should return 200, got %d", resp.StatusCode)
	}

	// Test get PTY
	req = httptest.NewRequest("GET", "/pty/test-id", nil)
	resp, err = s.app.Test(req, 1000)
	if err != nil {
		t.Errorf("Get PTY request failed: %v", err)
	}
	if resp.StatusCode != 200 {
		t.Errorf("Get PTY should return 200, got %d", resp.StatusCode)
	}

	// Test PTY connect (WebSocket - should return 501)
	req = httptest.NewRequest("GET", "/pty/test-id/connect", nil)
	resp, err = s.app.Test(req, 1000)
	if err != nil {
		t.Errorf("PTY connect request failed: %v", err)
	}
	if resp.StatusCode != 501 {
		t.Errorf("PTY connect should return 501 (not implemented), got %d", resp.StatusCode)
	}
}

// TestProviderRoutes tests provider routes.
func TestProviderRoutes(t *testing.T) {
	s := New("/tmp/test", DefaultConfig())

	// Test list providers
	req := httptest.NewRequest("GET", "/provider", nil)
	resp, err := s.app.Test(req, 1000)
	if err != nil {
		t.Errorf("List providers request failed: %v", err)
	}
	if resp.StatusCode != 200 {
		t.Errorf("List providers should return 200, got %d", resp.StatusCode)
	}

	// Test get provider
	req = httptest.NewRequest("GET", "/provider/anthropic", nil)
	resp, err = s.app.Test(req, 1000)
	if err != nil {
		t.Errorf("Get provider request failed: %v", err)
	}
	if resp.StatusCode != 200 {
		t.Errorf("Get provider should return 200, got %d", resp.StatusCode)
	}
}

// TestConfigRoutes tests config routes.
func TestConfigRoutes(t *testing.T) {
	s := New("/tmp/test", DefaultConfig())

	// Test get config
	req := httptest.NewRequest("GET", "/config", nil)
	resp, err := s.app.Test(req, 1000)
	if err != nil {
		t.Errorf("Get config request failed: %v", err)
	}
	if resp.StatusCode != 200 {
		t.Errorf("Get config should return 200, got %d", resp.StatusCode)
	}
}

// TestProjectRoutes tests project routes.
func TestProjectRoutes(t *testing.T) {
	s := New("/tmp/test", DefaultConfig())

	// Test get project
	req := httptest.NewRequest("GET", "/project", nil)
	resp, err := s.app.Test(req, 1000)
	if err != nil {
		t.Errorf("Get project request failed: %v", err)
	}
	if resp.StatusCode != 200 {
		t.Errorf("Get project should return 200, got %d", resp.StatusCode)
	}

	// Test get current project
	req = httptest.NewRequest("GET", "/project/current", nil)
	resp, err = s.app.Test(req, 1000)
	if err != nil {
		t.Errorf("Get current project request failed: %v", err)
	}
	if resp.StatusCode != 200 {
		t.Errorf("Get current project should return 200, got %d", resp.StatusCode)
	}
}

// TestPathEndpoint tests path endpoint.
func TestPathEndpoint(t *testing.T) {
	s := New("/tmp/test", DefaultConfig())

	req := httptest.NewRequest("GET", "/path", nil)
	resp, err := s.app.Test(req, 1000)
	if err != nil {
		t.Errorf("Path request failed: %v", err)
	}
	if resp.StatusCode != 200 {
		t.Errorf("Path should return 200, got %d", resp.StatusCode)
	}
}

// TestAgentRoutes tests agent routes.
func TestAgentRoutes(t *testing.T) {
	s := New("/tmp/test", DefaultConfig())

	req := httptest.NewRequest("GET", "/agent", nil)
	resp, err := s.app.Test(req, 1000)
	if err != nil {
		t.Errorf("Agent request failed: %v", err)
	}
	if resp.StatusCode != 200 {
		t.Errorf("Agent should return 200, got %d", resp.StatusCode)
	}
}

// TestFileRoutes tests file routes.
func TestFileRoutes(t *testing.T) {
	s := New("/tmp/test", DefaultConfig())

	// Test file read
	req := httptest.NewRequest("GET", "/file/read", nil)
	resp, err := s.app.Test(req, 1000)
	if err != nil {
		t.Errorf("File read request failed: %v", err)
	}
	if resp.StatusCode != 200 {
		t.Errorf("File read should return 200, got %d", resp.StatusCode)
	}

	// Test file glob
	req = httptest.NewRequest("GET", "/file/glob", nil)
	resp, err = s.app.Test(req, 1000)
	if err != nil {
		t.Errorf("File glob request failed: %v", err)
	}
	if resp.StatusCode != 200 {
		t.Errorf("File glob should return 200, got %d", resp.StatusCode)
	}
}

// TestLSPRoutes tests LSP routes.
func TestLSPRoutes(t *testing.T) {
	s := New("/tmp/test", DefaultConfig())

	// Test LSP status
	req := httptest.NewRequest("GET", "/lsp", nil)
	resp, err := s.app.Test(req, 1000)
	if err != nil {
		t.Errorf("LSP status request failed: %v", err)
	}
	if resp.StatusCode != 200 {
		t.Errorf("LSP status should return 200, got %d", resp.StatusCode)
	}
}

// TestMiddleware tests middleware functions.
func TestMiddleware(t *testing.T) {
	// Test recoveryMiddleware
	recovery := recoveryMiddleware()
	if recovery == nil {
		t.Error("recoveryMiddleware should not be nil")
	}

	// Test loggerMiddleware
	logger := loggerMiddleware()
	if logger == nil {
		t.Error("loggerMiddleware should not be nil")
	}

	// Test corsMiddleware
	cors := corsMiddleware([]string{"http://localhost:3000"})
	if cors == nil {
		t.Error("corsMiddleware should not be nil")
	}

	// Test compressMiddleware
	compress := compressMiddleware()
	if compress == nil {
		t.Error("compressMiddleware should not be nil")
	}

	// Test instanceMiddleware
	instance := instanceMiddleware()
	if instance == nil {
		t.Error("instanceMiddleware should not be nil")
	}

	// Test noCacheMiddleware
	noCache := noCacheMiddleware()
	if noCache == nil {
		t.Error("noCacheMiddleware should not be nil")
	}
}

// TestRandomString tests random string generation.
func TestRandomString(t *testing.T) {
	s1 := randomString(8)
	s2 := randomString(8)

	if len(s1) != 8 {
		t.Errorf("String length should be 8, got %d", len(s1))
	}
	if len(s2) != 8 {
		t.Errorf("String length should be 8, got %d", len(s2))
	}
}

// TestGenerateRequestID tests request ID generation.
func TestGenerateRequestID(t *testing.T) {
	id := generateRequestID()

	if len(id) < 10 {
		t.Errorf("Request ID should be at least 10 characters, got %d", len(id))
	}
}

// TestSSEEvent tests SSE event.
func TestSSEEvent(t *testing.T) {
	event := SSEEvent{
		Type:       "test.event",
		Properties: map[string]interface{}{"key": "value"},
	}

	if event.Type != "test.event" {
		t.Error("Event type should be 'test.event'")
	}
	if len(event.Properties) != 1 {
		t.Error("Event should have 1 property")
	}
}

// TestJSONMarshal tests JSON marshaler.
func TestJSONMarshal(t *testing.T) {
	// Empty map
	result := jsonMarshal(map[string]interface{}{})
	if result != "{}" {
		t.Errorf("Empty map should be {}, got %s", result)
	}

	// Map with string
	result = jsonMarshal(map[string]interface{}{"key": "value"})
	expected := "{\"key\":\"value\"}"
	if result != expected {
		t.Errorf("String map should be %s, got %s", expected, result)
	}

	// Map with int
	result = jsonMarshal(map[string]interface{}{"num": 42})
	expected = "{\"num\":42}"
	if result != expected {
		t.Errorf("Int map should be %s, got %s", expected, result)
	}

	// Map with bool true
	result = jsonMarshal(map[string]interface{}{"bool": true})
	expected = "{\"bool\":true}"
	if result != expected {
		t.Errorf("Bool true map should be %s, got %s", expected, result)
	}

	// Map with bool false
	result = jsonMarshal(map[string]interface{}{"bool": false})
	expected = "{\"bool\":false}"
	if result != expected {
		t.Errorf("Bool false map should be %s, got %s", expected, result)
	}

	// Map with nil
	result = jsonMarshal(map[string]interface{}{"null": nil})
	expected = "{\"null\":null}"
	if result != expected {
		t.Errorf("Null map should be %s, got %s", expected, result)
	}
}

// TestListener tests listener.
func TestListener(t *testing.T) {
	s := New("/tmp/test", DefaultConfig())
	s.app = fiber.New(fiber.Config{
		DisableStartupMessage: true,
	})

	listener := &Listener{
		Hostname: "localhost",
		Port:     3000,
		server:   s.app,
	}

	if listener.Hostname != "localhost" {
		t.Error("Hostname should be 'localhost'")
	}
	if listener.Port != 3000 {
		t.Error("Port should be 3000")
	}

	// Test stop (should not error)
	err := listener.Stop(false)
	if err != nil {
		t.Errorf("Stop should not error: %v", err)
	}
}

// TestServerShutdown tests server shutdown.
func TestServerShutdown(t *testing.T) {
	s := New("/tmp/test", DefaultConfig())
	s.app = fiber.New(fiber.Config{
		DisableStartupMessage: true,
	})

	err := s.Shutdown()
	if err != nil {
		t.Errorf("Shutdown should not error: %v", err)
	}
}

// TestServerContext tests server context.
func TestServerContext(t *testing.T) {
	s := New("/tmp/test", DefaultConfig())

	ctx := s.Context()
	if ctx == nil {
		t.Error("Context should not be nil")
	}
}

// TestServerDirectory tests server directory.
func TestServerDirectory(t *testing.T) {
	s := New("/tmp/test", DefaultConfig())

	dir := s.Directory()
	if dir != "/tmp/test" {
		t.Errorf("Directory should be '/tmp/test', got '%s'", dir)
	}
}

// TestPortString tests port string conversion.
func TestPortString(t *testing.T) {
	result := portString(3000)
	if result != "3000" {
		t.Errorf("Port string should be '3000', got '%s'", result)
	}

	result = portString(0)
	if result != "3000" {
		t.Errorf("Default port string should be '3000', got '%s'", result)
	}
}

// TestITOA tests integer to string conversion.
func TestITOA(t *testing.T) {
	tests := []struct {
		input    int
		expected string
	}{
		{0, "0"},
		{1, "1"},
		{123, "123"},
		{3000, "3000"},
	}

	for _, tt := range tests {
		result := itoa(tt.input)
		if result != tt.expected {
			t.Errorf("itoa(%d) = '%s', expected '%s'", tt.input, result, tt.expected)
		}
	}
}

// TestVersionVariables tests version variables.
func TestVersionVariables(t *testing.T) {
	if Version != "dev" {
		t.Errorf("Default version should be 'dev', got '%s'", Version)
	}
	if BuildTime != "" {
		t.Error("BuildTime should be empty by default")
	}
}

// TestServerConfigStruct tests server config struct.
func TestServerConfigStruct(t *testing.T) {
	cfg := Config{
		Port:        8080,
		Hostname:    "0.0.0.0",
		CORS:        []string{"*"},
		Compression: true,
		MDNS:        false,
		MDNSDomain:  "local",
	}

	if cfg.Port != 8080 {
		t.Error("Port should be 8080")
	}
	if cfg.Hostname != "0.0.0.0" {
		t.Error("Hostname should be '0.0.0.0'")
	}
	if len(cfg.CORS) != 1 {
		t.Error("CORS should have 1 entry")
	}
	if !cfg.Compression {
		t.Error("Compression should be true")
	}
}

// TestServerIsRunning tests IsRunning.
func TestServerIsRunning(t *testing.T) {
	s := New("/tmp/test", DefaultConfig())

	// Initially not running
	if s.IsRunning() {
		t.Error("Server should not be running initially")
	}
}

// TestServeCommand tests serve command structure.
func TestServeCommand(t *testing.T) {
	s := New("/tmp/test", DefaultConfig())

	// Test that App returns non-nil
	app := s.App()
	if app == nil {
		t.Error("App should not be nil")
	}
}