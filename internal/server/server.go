package server

import (
	"context"
	"net/url"
	"os"
	"sync"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/rhony08/magicode/internal/bus"
	"github.com/rhony08/magicode/internal/config"
	"github.com/rhony08/magicode/internal/database"
	"github.com/rhony08/magicode/internal/global"
	"github.com/rhony08/magicode/internal/lsp"
	"github.com/rhony08/magicode/internal/pty"
	"github.com/rhony08/magicode/internal/provider"
	"github.com/rhony08/magicode/internal/session"
	"github.com/rhony08/magicode/internal/tool"
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

// Services holds all service references for the server
type Services struct {
	DB        *database.Database
	Config    *config.Service
	Bus       *bus.Service
	Provider  *provider.ProviderRegistry
	PTY       *pty.Manager
	LSP       *lsp.Manager
	Tools     *tool.Registry
	Processor *session.Processor // Session message processor
}

// Server represents the MagiCode HTTP server.
type Server struct {
	app       *fiber.App
	config    Config
	directory string
	ctx       context.Context
	cancel    context.CancelFunc
	mu        sync.RWMutex
	running   bool

	// Services
	services Services
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
		AppName:               "MagiCode",
		ServerHeader:          "MagiCode",
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

// WithServices attaches services to the server
func (s *Server) WithServices(svcs Services) *Server {
	s.services = svcs
	return s
}

// InitServices initializes all services
func (s *Server) InitServices() error {
	// Initialize database
	dbPath := global.DatabasePath()
	db, err := database.New(s.ctx, database.Config{Path: dbPath})
	if err != nil {
		return err
	}
	s.services.DB = db

	// Initialize config
	cfg, err := config.New(s.directory, global.Path.Config)
	if err != nil {
		// Continue with defaults
		cfg = config.NewDefault()
	}
	s.services.Config = cfg

	// Initialize bus
	s.services.Bus = bus.NewDefault()

	// Initialize provider registry
	s.services.Provider = provider.NewProviderRegistry()
	if os.Getenv("ANTHROPIC_API_KEY") != "" {
		s.services.Provider.Register(provider.NewAnthropicProvider(os.Getenv("ANTHROPIC_API_KEY")))
	}
	if os.Getenv("OPENAI_API_KEY") != "" {
		s.services.Provider.Register(provider.NewOpenAIProvider(os.Getenv("OPENAI_API_KEY")))
	}

	// Initialize PTY manager
	s.services.PTY = pty.NewManager(s.directory, s.services.Bus)

	// Initialize LSP manager
	s.services.LSP = lsp.NewManager(s.directory)

	// Initialize tool registry
	s.services.Tools = tool.NewRegistry()
	s.services.Tools.Register(tool.NewReadTool())
	s.services.Tools.Register(tool.NewWriteTool())
	s.services.Tools.Register(tool.NewEditTool())
	s.services.Tools.Register(tool.NewGlobTool())
	s.services.Tools.Register(tool.NewGrepTool())
	s.services.Tools.Register(tool.NewBashTool())
	s.services.Tools.Register(tool.NewWebFetchTool())

	return nil
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

	// Instance routes (require instance context) - defined in routes.go
	s.setupInstanceRoutes()
}

// setupGlobalRoutes configures global routes.
func (s *Server) setupGlobalRoutes() {
	globalGroup := s.app.Group("/global")

	// Health check
	globalGroup.Get("/health", s.handleHealth)

	// Version info
	globalGroup.Get("/version", s.handleVersion)

	// Event stream (SSE)
	globalGroup.Get("/event", s.handleGlobalEventStream)
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

	// Close services
	if s.services.DB != nil {
		s.services.DB.Close()
	}
	if s.services.Bus != nil {
		s.services.Bus.Close()
	}
	if s.services.PTY != nil {
		s.services.PTY.Shutdown()
	}
	if s.services.LSP != nil {
		s.services.LSP.Close()
	}

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

// Services returns the server services.
func (s *Server) Services() Services {
	return s.services
}