package server

import (
	"time"

	"github.com/gofiber/fiber/v2"
)

// setupInstanceRoutes configures instance routes.
func (s *Server) setupInstanceRoutes() {
	// Create instance group with middleware (no prefix - direct routes)
	instance := s.app.Group("", instanceMiddleware())

	// Session routes
	instance.Get("/session", s.handleListSessions)
	instance.Get("/session/status", s.handleSessionStatus)
	instance.Post("/session", s.handleCreateSession)
	instance.Get("/session/:id", s.handleGetSession)
	instance.Put("/session/:id", s.handleUpdateSession)
	instance.Delete("/session/:id", s.handleDeleteSession)
	instance.Get("/session/:id/message", s.handleGetMessages)
	instance.Post("/session/:id/message", s.handleAddMessage)
	instance.Post("/session/:id/prompt_async", s.handlePromptAsync) // Async message processing
	instance.Get("/session/:id/events", s.handleSessionEventStream)  // SSE per session
	instance.Post("/session/:id/run", s.handleRunSession)
	instance.Post("/session/:id/cancel", s.handleCancelSession)
	instance.Post("/session/:id/abort", s.handleAbortSession)
	instance.Post("/session/:id/revert", s.handleRevertSession)
	instance.Post("/session/:id/compact", s.handleCompactSession)

	// Project routes
	instance.Get("/project", s.handleGetProject)
	instance.Get("/project/current", s.handleGetCurrentProject)
	instance.Get("/project/list", s.handleListProjects)

	// PTY routes
	instance.Get("/pty", s.handleListPTY)
	instance.Post("/pty", s.handleCreatePTY)
	instance.Get("/pty/:id", s.handleGetPTY)
	instance.Put("/pty/:id", s.handleUpdatePTY)
	instance.Delete("/pty/:id", s.handleDeletePTY)
	instance.Get("/pty/:id/connect", s.handlePTYConnect)

	// File routes
	instance.Get("/file/read", s.handleFileRead)
	instance.Post("/file/write", s.handleFileWrite)
	instance.Post("/file/edit", s.handleFileEdit)
	instance.Get("/file/glob", s.handleFileGlob)
	instance.Get("/file/grep", s.handleFileGrep)
	instance.Get("/file/list", s.handleFileList)
	instance.Get("/file/info", s.handleFileInfo)

	// Event routes (SSE)
	instance.Get("/event", s.handleInstanceEventStream)

	// Provider routes
	instance.Get("/provider", s.handleListProviders)
	instance.Get("/provider/:id", s.handleGetProvider)
	instance.Get("/provider/:id/auth", s.handleProviderAuth)
	instance.Post("/provider/:id/oauth/authorize", s.handleOAuthAuthorize)
	instance.Post("/provider/:id/oauth/callback", s.handleOAuthCallback)

	// Config routes
	instance.Get("/config", s.handleGetConfig)
	instance.Put("/config", s.handleUpdateConfig)
	instance.Get("/config/providers", s.handleGetConfigProviders)

	// LSP routes
	instance.Get("/lsp", s.handleLSPStatus)
	instance.Post("/lsp/:language/start", s.handleLSPStart)
	instance.Post("/lsp/:language/stop", s.handleLSPStop)
	instance.Get("/lsp/:language/diagnostics", s.handleLSPDiagnostics)

	// Path routes
	instance.Get("/path", s.handlePath)

	// Agent routes
	instance.Get("/agent", s.handleListAgents)

	// Command routes
	instance.Get("/command", s.handleListCommands)

	// Permission routes
	instance.Get("/permission", s.handleListPermissions)
	instance.Get("/permission/:id", s.handleGetPermission)
	instance.Post("/permission/:id/approve", s.handleApprovePermission)
	instance.Post("/permission/:id/deny", s.handleDenyPermission)

	// Sync routes
	instance.Get("/sync/status", s.handleSyncStatus)
	instance.Post("/sync/start", s.handleSyncStart)
	instance.Post("/sync/stop", s.handleSyncStop)
}

// handleInstanceEventStream handles SSE event stream for instance.
func (s *Server) handleInstanceEventStream(c *fiber.Ctx) error {
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

		case <-s.ctx.Done():
			// Server shutting down
			c.Write([]byte("data: "))
			c.Write([]byte(`{"type":"instance.disposed","properties":{}}`))
			c.Write([]byte("\n\n"))
			return nil
		}
	}
}