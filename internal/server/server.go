// Package server provides HTTP/WebSocket server for OpenCode.
package server

import (
	"context"
	"net/url"
	"sync"
	"time"

	"github.com/gofiber/fiber/v2"
)

// Listener represents a server listener.
type Listener struct {
	Hostname string
	Port     int
	URL      *url.URL
	server   *fiber.App
	mu       sync.Mutex
	closed   bool
}

// Stop stops the server.
func (l *Listener) Stop(close bool) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.closed {
		return nil
	}

	if close {
		l.closed = true
		return l.server.Shutdown()
	}
	return nil
}

// Config represents server configuration.
type Config struct {
	// Port is the server port (default: 3000)
	Port int

	// Hostname is the server hostname (default: "localhost")
	Hostname string

	// CORS origins
	CORS []string

	// Enable compression
	Compression bool

	// Enable mDNS discovery
	MDNS bool

	// mDNS domain
	MDNSDomain string
}

// DefaultConfig returns default server configuration.
func DefaultConfig() Config {
	return Config{
		Port:        3000,
		Hostname:    "localhost",
		CORS:        []string{},
		Compression: true,
		MDNS:        false,
		MDNSDomain:  "",
	}
}

// Server represents the OpenCode HTTP server.
type Server struct {
	app       *fiber.App
	config    Config
	directory string
	ctx       context.Context
	cancel    context.CancelFunc
	mu        sync.RWMutex
	running   bool

	// Services
	bus      interface{} // Bus service (will be typed later)
	database interface{} // Database service
	provider interface{} // Provider service
	lsp      interface{} // LSP service
	pty      interface{} // PTY manager
}

// New creates a new server.
func New(directory string, cfg Config) *Server {
	ctx, cancel := context.WithCancel(context.Background())

	// Use default config if not specified
	if cfg.Port == 0 {
		cfg.Port = DefaultConfig().Port
	}
	if cfg.Hostname == "" {
		cfg.Hostname = DefaultConfig().Hostname
	}

	s := &Server{
		config:    cfg,
		directory: directory,
		ctx:       ctx,
		cancel:    cancel,
	}

	// Create Fiber app
	s.app = fiber.New(fiber.Config{
		AppName:               "OpenCode",
		ServerHeader:          "OpenCode",
		DisableDefaultDate:    true,
		DisableDefaultContentType: true,
		BodyLimit:             50 * 1024 * 1024,
		StrictRouting:         true,
		CaseSensitive:         true,
	})

	// Setup middleware
	s.setupMiddleware()

	// Setup routes
	s.setupRoutes()

	return s
}

// setupMiddleware configures middleware.
func (s *Server) setupMiddleware() {
	// Recovery middleware
	s.app.Use(recoveryMiddleware())

	// Logger middleware
	s.app.Use(loggerMiddleware())

	// CORS middleware
	if len(s.config.CORS) > 0 {
		s.app.Use(corsMiddleware(s.config.CORS))
	}

	// Compression middleware
	if s.config.Compression {
		s.app.Use(compressMiddleware())
	}
}

// setupRoutes configures routes.
func (s *Server) setupRoutes() {
	// Global routes (no instance required)
	s.setupGlobalRoutes()

	// Instance routes (require instance context)
	s.setupInstanceRoutes()
}

// setupGlobalRoutes configures global routes.
func (s *Server) setupGlobalRoutes() {
	global := s.app.Group("/global")

	// Health check
	global.Get("/health", s.handleHealth)

	// Version info
	global.Get("/version", s.handleVersion)

	// Event stream (SSE)
	global.Get("/event", s.handleGlobalEventStream)
}

// setupInstanceRoutes configures instance routes.
func (s *Server) setupRoutesOnGroup() {
	// These will be setup when instance middleware is applied
}

// App returns the Fiber app.
func (s *Server) App() *fiber.App {
	return s.app
}

// Listen starts the server.
func (s *Server) Listen() (*Listener, error) {
	s.mu.Lock()
	s.running = true
	s.mu.Unlock()

	addr := s.config.Hostname
	if s.config.Port != 0 {
		addr = addr + ":" + portString(s.config.Port)
	}

	// Build URL
	u := &url.URL{
		Scheme: "http",
		Host:   s.config.Hostname + ":" + portString(s.config.Port),
	}

	listener := &Listener{
		Hostname: s.config.Hostname,
		Port:     s.config.Port,
		URL:      u,
		server:   s.app,
	}

	// Start server in background
	go func() {
		if err := s.app.Listen(addr); err != nil {
			// Log error but don't panic
			s.mu.Lock()
			s.running = false
			s.mu.Unlock()
		}
	}()

	// Wait a bit for server to start
	time.Sleep(100 * time.Millisecond)

	return listener, nil
}

// Shutdown shuts down the server.
func (s *Server) Shutdown() error {
	s.cancel()
	s.mu.Lock()
	s.running = false
	s.mu.Unlock()
	return s.app.Shutdown()
}

// IsRunning checks if server is running.
func (s *Server) IsRunning() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.running
}

// portString converts port to string.
func portString(port int) string {
	if port == 0 {
		return "3000"
	}
	return itoa(port)
}

// itoa converts int to string (simple version for small numbers).
func itoa(n int) string {
	if n == 0 {
		return "0"
	}

	var digits []byte
	for n > 0 {
		digits = append([]byte{byte('0' + n%10)}, digits...)
		n /= 10
	}
	return string(digits)
}

// Context returns the server context.
func (s *Server) Context() context.Context {
	return s.ctx
}

// Directory returns the server directory.
func (s *Server) Directory() string {
	return s.directory
}

// Config returns the server config.
func (s *Server) Config() Config {
	return s.config
}