package server

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/gofiber/fiber/v2"
	"github.com/rhony08/magicode/internal/database"
	"github.com/rhony08/magicode/internal/global"
	"github.com/rhony08/magicode/internal/lsp"
	"github.com/rhony08/magicode/internal/pty"
	"github.com/rhony08/magicode/internal/provider"
)

// Session handlers

// handleListSessions handles listing sessions.
func (s *Server) handleListSessions(c *fiber.Ctx) error {
	if s.services.DB == nil {
		return c.JSON([]fiber.Map{})
	}

	ctx := context.Background()
	sessionStorage := database.NewSessionStorage(s.services.DB)

	// List sessions by directory
	sessions, err := sessionStorage.ListByDirectory(ctx, s.directory)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{
			"error": err.Error(),
		})
	}

	// Transform to API response
	result := make([]fiber.Map, len(sessions))
	for i, session := range sessions {
		result[i] = fiber.Map{
			"id":         session.ID,
			"title":      session.Title,
			"directory":  session.Directory,
			"slug":       session.Slug,
			"created_at": session.Timestamps.TimeCreated,
			"updated_at": session.Timestamps.TimeUpdated,
		}
	}

	return c.JSON(result)
}

// handleSessionStatus handles getting session status.
func (s *Server) handleSessionStatus(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{
		"status":    "idle",
		"directory": s.directory,
	})
}

// handleCreateSession handles creating a session.
func (s *Server) handleCreateSession(c *fiber.Ctx) error {
	// Parse request body
	type CreateRequest struct {
		Title string `json:"title"`
	}
	var req CreateRequest
	if err := c.BodyParser(&req); err != nil {
		req.Title = "New Session"
	}
	if req.Title == "" {
		req.Title = "New Session"
	}

	if s.services.DB == nil {
		return c.JSON(fiber.Map{
			"id":         randomString(8),
			"title":      req.Title,
			"directory":  s.directory,
			"created_at": time.Now().UnixMilli(),
		})
	}

	ctx := context.Background()
	sessionStorage := database.NewSessionStorage(s.services.DB)

	// Create project if needed
	projectStorage := database.NewProjectStorage(s.services.DB)
	project, err := projectStorage.GetByWorktree(ctx, s.directory)
	if err != nil {
		// Create project
		project = &database.Project{
			Worktree: s.directory,
			Name:     "Project",
		}
		project, err = projectStorage.Create(ctx, *project)
		if err != nil {
			return c.Status(500).JSON(fiber.Map{"error": err.Error()})
		}
	}

	// Create session
	session := database.Session{
		ProjectID: project.ID,
		Directory: s.directory,
		Title:     req.Title,
	}
	created, err := sessionStorage.Create(ctx, session)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{
		"id":         created.ID,
		"title":      created.Title,
		"directory":  created.Directory,
		"slug":       created.Slug,
		"created_at": created.Timestamps.TimeCreated,
	})
}

// handleGetSession handles getting a session by ID.
func (s *Server) handleGetSession(c *fiber.Ctx) error {
	id := c.Params("id")

	if s.services.DB == nil {
		return c.JSON(fiber.Map{
			"id":    id,
			"title": "Session " + id,
		})
	}

	ctx := context.Background()
	sessionStorage := database.NewSessionStorage(s.services.DB)

	session, err := sessionStorage.Get(ctx, id)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{
			"error": "Session not found",
			"id":    id,
		})
	}

	return c.JSON(fiber.Map{
		"id":         session.ID,
		"title":      session.Title,
		"directory":  session.Directory,
		"slug":       session.Slug,
		"created_at": session.Timestamps.TimeCreated,
		"updated_at": session.Timestamps.TimeUpdated,
	})
}

// handleUpdateSession handles updating a session.
func (s *Server) handleUpdateSession(c *fiber.Ctx) error {
	id := c.Params("id")

	// Parse request body
	type UpdateRequest struct {
		Title string `json:"title"`
	}
	var req UpdateRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid request body"})
	}

	if s.services.DB == nil {
		return c.JSON(fiber.Map{
			"id":    id,
			"title": req.Title,
		})
	}

	ctx := context.Background()
	sessionStorage := database.NewSessionStorage(s.services.DB)

	session, err := sessionStorage.Get(ctx, id)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "Session not found"})
	}

	if req.Title != "" {
		session.Title = req.Title
	}

	err = sessionStorage.Update(ctx, *session)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{
		"id":         session.ID,
		"title":      session.Title,
		"updated_at": session.Timestamps.TimeUpdated,
	})
}

// handleDeleteSession handles deleting a session.
func (s *Server) handleDeleteSession(c *fiber.Ctx) error {
	id := c.Params("id")

	if s.services.DB == nil {
		return c.JSON(fiber.Map{
			"success": true,
			"id":      id,
		})
	}

	ctx := context.Background()
	sessionStorage := database.NewSessionStorage(s.services.DB)

	err := sessionStorage.Delete(ctx, id)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"id":      id,
	})
}

// handleGetMessages handles getting messages for a session.
func (s *Server) handleGetMessages(c *fiber.Ctx) error {
	id := c.Params("id")

	if s.services.DB == nil {
		return c.JSON([]fiber.Map{})
	}

	ctx := context.Background()
	msgStorage := database.NewMessageStorage(s.services.DB)

	messages, err := msgStorage.List(ctx, id)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	// Transform to API response
	result := make([]fiber.Map, len(messages))
	for i, msg := range messages {
		result[i] = fiber.Map{
			"id":         msg.ID,
			"session_id": msg.SessionID,
			"role":       msg.Data.Role,
			"model":      msg.Data.ModelID,
			"provider":   msg.Data.ProviderID,
			"created_at": msg.Timestamps.TimeCreated,
		}
	}

	return c.JSON(result)
}

// handleAddMessage handles adding a message to a session.
func (s *Server) handleAddMessage(c *fiber.Ctx) error {
	id := c.Params("id")

	// Parse request body
	type AddMessageRequest struct {
		Role    string `json:"role"`
		Content string `json:"content"`
	}
	var req AddMessageRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid request body"})
	}

	if s.services.DB == nil {
		return c.JSON(fiber.Map{
			"id":         randomString(8),
			"session_id": id,
			"role":       req.Role,
			"content":    req.Content,
			"created_at": time.Now().UnixMilli(),
		})
	}

	ctx := context.Background()
	msgStorage := database.NewMessageStorage(s.services.DB)
	partStorage := database.NewPartStorage(s.services.DB)

	msg := database.Message{
		SessionID: id,
		Data: database.MessageInfo{
			Role: req.Role,
		},
	}

	created, err := msgStorage.Create(ctx, msg)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	// Create a part for the message content
	if req.Content != "" {
		part := database.Part{
			MessageID: created.ID,
			SessionID: id,
			Data: database.PartData{
				Type: "text",
				Text: req.Content,
			},
		}
		_, err = partStorage.Create(ctx, part)
		if err != nil {
			// Log warning but don't fail
		}
	}

	return c.JSON(fiber.Map{
		"id":         created.ID,
		"session_id": created.SessionID,
		"role":       created.Data.Role,
		"model":      created.Data.ModelID,
		"provider":   created.Data.ProviderID,
		"created_at": created.Timestamps.TimeCreated,
	})
}

// handleRunSession handles running a session.
func (s *Server) handleRunSession(c *fiber.Ctx) error {
	id := c.Params("id")
	return c.JSON(fiber.Map{
		"id":     id,
		"status": "running",
	})
}

// handleCancelSession handles canceling a session.
func (s *Server) handleCancelSession(c *fiber.Ctx) error {
	id := c.Params("id")
	return c.JSON(fiber.Map{
		"id":     id,
		"status": "cancelled",
	})
}

// handleAbortSession handles aborting a session.
func (s *Server) handleAbortSession(c *fiber.Ctx) error {
	id := c.Params("id")
	return c.JSON(fiber.Map{
		"id":     id,
		"status": "aborted",
	})
}

// handleRevertSession handles reverting a session.
func (s *Server) handleRevertSession(c *fiber.Ctx) error {
	id := c.Params("id")
	return c.JSON(fiber.Map{
		"id":     id,
		"status": "reverted",
	})
}

// handleCompactSession handles compacting a session.
func (s *Server) handleCompactSession(c *fiber.Ctx) error {
	id := c.Params("id")
	return c.JSON(fiber.Map{
		"id":      id,
		"status":  "compacted",
		"compact": true,
	})
}

// Project handlers

// handlePath handles path requests.
func (s *Server) handlePath(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{
		"home":      s.directory,
		"state":     global.Path.State,
		"config":    global.Path.Config,
		"data":      global.Path.Data,
		"worktree":  s.directory,
		"directory": s.directory,
	})
}

// handleGetProject handles getting project info.
func (s *Server) handleGetProject(c *fiber.Ctx) error {
	if s.services.DB == nil {
		return c.JSON(fiber.Map{
			"id":        randomString(8),
			"directory": s.directory,
			"name":      "My Project",
		})
	}

	ctx := context.Background()
	projectStorage := database.NewProjectStorage(s.services.DB)

	project, err := projectStorage.GetByWorktree(ctx, s.directory)
	if err != nil {
		return c.JSON(fiber.Map{
			"directory": s.directory,
			"name":      "New Project",
		})
	}

	return c.JSON(fiber.Map{
		"id":        project.ID,
		"worktree":  project.Worktree,
		"name":      project.Name,
		"created_at": project.Timestamps.TimeCreated,
	})
}

// handleGetCurrentProject handles getting current project.
func (s *Server) handleGetCurrentProject(c *fiber.Ctx) error {
	return s.handleGetProject(c)
}

// handleListProjects handles listing projects.
func (s *Server) handleListProjects(c *fiber.Ctx) error {
	if s.services.DB == nil {
		return c.JSON([]fiber.Map{})
	}

	ctx := context.Background()
	projectStorage := database.NewProjectStorage(s.services.DB)

	projects, err := projectStorage.List(ctx)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	result := make([]fiber.Map, len(projects))
	for i, project := range projects {
		result[i] = fiber.Map{
			"id":        project.ID,
			"worktree":  project.Worktree,
			"name":      project.Name,
			"created_at": project.Timestamps.TimeCreated,
		}
	}

	return c.JSON(result)
}

// PTY handlers

// handleListPTY handles listing PTY sessions.
func (s *Server) handleListPTY(c *fiber.Ctx) error {
	if s.services.PTY == nil {
		return c.JSON([]fiber.Map{})
	}

	sessions := s.services.PTY.List()

	result := make([]fiber.Map, len(sessions))
	for i, session := range sessions {
		result[i] = fiber.Map{
			"id":     session.ID,
			"title":  session.Title,
			"status": session.Status,
			"pid":    session.PID,
		}
	}

	return c.JSON(result)
}

// handleCreatePTY handles creating a PTY session.
func (s *Server) handleCreatePTY(c *fiber.Ctx) error {
	// Parse request body
	type CreatePTYRequest struct {
		Command string `json:"command"`
		Title   string `json:"title"`
		Cols    int    `json:"cols"`
		Rows    int    `json:"rows"`
	}
	var req CreatePTYRequest
	if err := c.BodyParser(&req); err != nil {
		req.Command = "/bin/bash"
		req.Title = "Terminal"
	}
	if req.Command == "" {
		req.Command = "/bin/bash"
	}
	if req.Cols == 0 {
		req.Cols = 80
	}
	if req.Rows == 0 {
		req.Rows = 24
	}

	if s.services.PTY == nil {
		return c.JSON(fiber.Map{
			"id":     randomString(8),
			"title":  req.Title,
			"status": "running",
		})
	}

	info, err := s.services.PTY.Create(pty.CreateInput{
		Command: req.Command,
		Title:   req.Title,
		Cols:    req.Cols,
		Rows:    req.Rows,
		CWD:     s.directory,
	})
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{
		"id":     info.ID,
		"title":  info.Title,
		"status": info.Status,
		"pid":    info.PID,
	})
}

// handleGetPTY handles getting a PTY session.
func (s *Server) handleGetPTY(c *fiber.Ctx) error {
	id := c.Params("id")

	if s.services.PTY == nil {
		return c.JSON(fiber.Map{
			"id":     id,
			"title":  "Terminal",
			"status": "running",
		})
	}

	info, err := s.services.PTY.Get(id)
	if err != nil {
		return c.Status(404).JSON(fiber.Map{"error": "PTY session not found"})
	}

	return c.JSON(fiber.Map{
		"id":     info.ID,
		"title":  info.Title,
		"status": info.Status,
		"pid":    info.PID,
	})
}

// handleUpdatePTY handles updating a PTY session.
func (s *Server) handleUpdatePTY(c *fiber.Ctx) error {
	id := c.Params("id")

	// Parse request body
	type UpdatePTYRequest struct {
		Title string `json:"title"`
		Cols  int    `json:"cols"`
		Rows  int    `json:"rows"`
	}
	var req UpdatePTYRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid request"})
	}

	if s.services.PTY == nil {
		return c.JSON(fiber.Map{
			"id":     id,
			"title":  req.Title,
			"status": "running",
		})
	}

	updateInput := pty.UpdateInput{}
	if req.Title != "" {
		updateInput.Title = req.Title
	}
	if req.Cols > 0 && req.Rows > 0 {
		updateInput.Size = &pty.Size{Cols: req.Cols, Rows: req.Rows}
	}

	info, err := s.services.PTY.Update(id, updateInput)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{
		"id":     info.ID,
		"title":  info.Title,
		"status": info.Status,
	})
}

// handleDeletePTY handles deleting a PTY session.
func (s *Server) handleDeletePTY(c *fiber.Ctx) error {
	id := c.Params("id")

	if s.services.PTY == nil {
		return c.JSON(fiber.Map{
			"success": true,
			"id":      id,
		})
	}

	err := s.services.PTY.Remove(id)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"id":      id,
	})
}

// handlePTYConnect handles WebSocket connection to PTY.
func (s *Server) handlePTYConnect(c *fiber.Ctx) error {
	// WebSocket implementation requires additional middleware
	return c.Status(501).JSON(fiber.Map{
		"error":   true,
		"message": "WebSocket not implemented - use native terminal",
	})
}

// Provider handlers

// handleListProviders handles listing providers.
func (s *Server) handleListProviders(c *fiber.Ctx) error {
	if s.services.Provider == nil {
		// Return default providers
		return c.JSON([]fiber.Map{
			{
				"id":    "anthropic",
				"name":  "Anthropic",
				"models": []fiber.Map{
					{"id": "claude-sonnet-4-5", "name": "Claude Sonnet 4.5"},
					{"id": "claude-3-5-haiku", "name": "Claude 3.5 Haiku"},
				},
			},
			{
				"id":    "openai",
				"name":  "OpenAI",
				"models": []fiber.Map{
					{"id": "gpt-4o", "name": "GPT-4o"},
					{"id": "gpt-4o-mini", "name": "GPT-4o Mini"},
				},
			},
		})
	}

	providers := s.services.Provider.ListProviders()

	result := make([]fiber.Map, len(providers))
	for i, p := range providers {
		info := p.Info()
		models := s.services.Provider.ListModelsByProvider(info.ID)

		modelList := make([]fiber.Map, len(models))
		for j, m := range models {
			modelList[j] = fiber.Map{
				"id":   string(m.ID),
				"name": m.Name,
			}
		}

		// Check auth status
		authStatus := "not_authenticated"
		for _, envKey := range info.EnvKeys {
			if os.Getenv(envKey) != "" {
				authStatus = "authenticated"
				break
			}
		}

		result[i] = fiber.Map{
			"id":           string(info.ID),
			"name":         info.Name,
			"models":       modelList,
			"auth_status":  authStatus,
		}
	}

	return c.JSON(result)
}

// handleGetProvider handles getting a provider.
func (s *Server) handleGetProvider(c *fiber.Ctx) error {
	id := c.Params("id")

	if s.services.Provider == nil {
		return c.JSON(fiber.Map{
			"id":   id,
			"name": "Provider " + id,
		})
	}

	p, ok := s.services.Provider.Get(provider.ProviderID(id))
	if !ok {
		return c.Status(404).JSON(fiber.Map{"error": "Provider not found"})
	}

	info := p.Info()
	models := s.services.Provider.ListModelsByProvider(info.ID)

	modelList := make([]fiber.Map, len(models))
	for i, m := range models {
		modelList[i] = fiber.Map{
			"id":   string(m.ID),
			"name": m.Name,
		}
	}

	return c.JSON(fiber.Map{
		"id":     string(info.ID),
		"name":   info.Name,
		"models": modelList,
	})
}

// handleProviderAuth handles provider authentication.
func (s *Server) handleProviderAuth(c *fiber.Ctx) error {
	id := c.Params("id")

	// Check environment variable
	envVars := map[string][]string{
		"anthropic": []string{"ANTHROPIC_API_KEY"},
		"openai":    []string{"OPENAI_API_KEY"},
	}

	vars, ok := envVars[id]
	if !ok {
		return c.Status(400).JSON(fiber.Map{"error": "Unknown provider"})
	}

	status := "not_authenticated"
	for _, envVar := range vars {
		if os.Getenv(envVar) != "" {
			status = "authenticated"
			break
		}
	}

	return c.JSON(fiber.Map{
		"provider_id": id,
		"auth_status": status,
	})
}

// handleOAuthAuthorize handles OAuth authorization.
func (s *Server) handleOAuthAuthorize(c *fiber.Ctx) error {
	id := c.Params("id")
	return c.JSON(fiber.Map{
		"provider_id":   id,
		"authorize_url": fmt.Sprintf("https://%s.com/oauth/authorize", id),
		"message":       "OAuth not supported - use environment variables",
	})
}

// handleOAuthCallback handles OAuth callback.
func (s *Server) handleOAuthCallback(c *fiber.Ctx) error {
	id := c.Params("id")
	return c.JSON(fiber.Map{
		"provider_id": id,
		"status":      "authorized",
	})
}

// Config handlers

// handleGetConfig handles getting config.
func (s *Server) handleGetConfig(c *fiber.Ctx) error {
	if s.services.Config == nil {
		return c.JSON(fiber.Map{
			"model":       "anthropic/claude-sonnet-4-5",
			"small_model": "anthropic/claude-3-5-haiku",
			"log_level":   "INFO",
		})
	}

	info := s.services.Config.Get()

	return c.JSON(fiber.Map{
		"model":        s.services.Config.Model(),
		"small_model":  s.services.Config.SmallModel(),
		"log_level":    s.services.Config.LogLevel(),
		"username":     info.Username,
		"default_agent": info.DefaultAgent,
		"config_file":  s.services.Config.ConfigPath(),
	})
}

// handleUpdateConfig handles updating config.
func (s *Server) handleUpdateConfig(c *fiber.Ctx) error {
	// TODO: Implement config file writing
	return c.JSON(fiber.Map{
		"success": true,
		"message": "Config update not implemented - edit config file directly",
	})
}

// handleGetConfigProviders handles getting providers from config.
func (s *Server) handleGetConfigProviders(c *fiber.Ctx) error {
	if s.services.Config == nil {
		return c.JSON([]fiber.Map{})
	}

	info := s.services.Config.Get()

	result := make([]fiber.Map, 0)
	for id, p := range info.Providers {
		result = append(result, fiber.Map{
			"id":   id,
			"env":  p.Env,
		})
	}

	return c.JSON(result)
}

// LSP handlers

// handleLSPStatus handles getting LSP status.
func (s *Server) handleLSPStatus(c *fiber.Ctx) error {
	if s.services.LSP == nil {
		return c.JSON([]fiber.Map{})
	}

	running := s.services.LSP.Running()

	result := make([]fiber.Map, len(running))
	for i, serverID := range running {
		isRunning := s.services.LSP.IsRunning(serverID)
		status := "stopped"
		if isRunning {
			status = "running"
		}
		result[i] = fiber.Map{
			"language":  string(serverID),
			"status":    status,
			"server_id": string(serverID),
		}
	}

	return c.JSON(result)
}

// handleLSPStart handles starting LSP for a language.
func (s *Server) handleLSPStart(c *fiber.Ctx) error {
	language := c.Params("language")

	if s.services.LSP == nil {
		return c.JSON(fiber.Map{
			"language": language,
			"status":   "started",
		})
	}

	ctx := context.Background()
	err := s.services.LSP.Start(ctx, lsp.ServerID(language))
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{
		"language": language,
		"status":   "started",
	})
}

// handleLSPStop handles stopping LSP for a language.
func (s *Server) handleLSPStop(c *fiber.Ctx) error {
	language := c.Params("language")

	if s.services.LSP == nil {
		return c.JSON(fiber.Map{
			"language": language,
			"status":   "stopped",
		})
	}

	ctx := context.Background()
	s.services.LSP.Stop(ctx, lsp.ServerID(language))

	return c.JSON(fiber.Map{
		"language": language,
		"status":   "stopped",
	})
}

// handleLSPDiagnostics handles getting LSP diagnostics.
func (s *Server) handleLSPDiagnostics(c *fiber.Ctx) error {
	file := c.Query("file")

	if s.services.LSP == nil {
		return c.JSON([]fiber.Map{})
	}

	diagnostics := s.services.LSP.GetDiagnostics(file)

	result := make([]fiber.Map, len(diagnostics))
	for i, d := range diagnostics {
		result[i] = fiber.Map{
			"message":    d.Message,
			"severity":   d.Severity,
			"source":     d.Source,
			"range": fiber.Map{
				"start": fiber.Map{
					"line":      d.Range.Start.Line,
					"character": d.Range.Start.Character,
				},
				"end": fiber.Map{
					"line":      d.Range.End.Line,
					"character": d.Range.End.Character,
				},
			},
		}
	}

	return c.JSON(result)
}

// Agent handlers

// handleListAgents handles listing agents.
func (s *Server) handleListAgents(c *fiber.Ctx) error {
	return c.JSON([]fiber.Map{
		{
			"id":          "build",
			"name":        "Build Agent",
			"description": "Developer agent for writing code",
		},
		{
			"id":          "plan",
			"name":        "Plan Agent",
			"description": "Planning agent for project management",
		},
		{
			"id":          "architect",
			"name":        "Architect Agent",
			"description": "System design and architecture",
		},
	})
}

// Command handlers

// handleListCommands handles listing commands.
func (s *Server) handleListCommands(c *fiber.Ctx) error {
	return c.JSON([]fiber.Map{
		{"name": "run", "description": "Start interactive TUI"},
		{"name": "serve", "description": "Start HTTP server"},
		{"name": "session", "description": "Manage sessions"},
		{"name": "config", "description": "Manage configuration"},
		{"name": "providers", "description": "List AI providers"},
		{"name": "models", "description": "List AI models"},
	})
}

// Permission handlers

// handleListPermissions handles listing permissions.
func (s *Server) handleListPermissions(c *fiber.Ctx) error {
	// Return default permission rules
	return c.JSON([]fiber.Map{
		{
			"id":     "read-*",
			"tool":   "read",
			"pattern": "*",
			"action": "allow",
		},
		{
			"id":     "write-ask",
			"tool":   "write",
			"pattern": "*",
			"action": "ask",
		},
		{
			"id":     "bash-ask",
			"tool":   "bash",
			"pattern": "*",
			"action": "ask",
		},
	})
}

// handleGetPermission handles getting a permission.
func (s *Server) handleGetPermission(c *fiber.Ctx) error {
	id := c.Params("id")
	return c.JSON(fiber.Map{
		"id":     id,
		"status": "pending",
	})
}

// handleApprovePermission handles approving a permission.
func (s *Server) handleApprovePermission(c *fiber.Ctx) error {
	id := c.Params("id")
	return c.JSON(fiber.Map{
		"id":     id,
		"status": "approved",
	})
}

// handleDenyPermission handles denying a permission.
func (s *Server) handleDenyPermission(c *fiber.Ctx) error {
	id := c.Params("id")
	return c.JSON(fiber.Map{
		"id":     id,
		"status": "denied",
	})
}

// Sync handlers

// handleSyncStatus handles getting sync status.
func (s *Server) handleSyncStatus(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{
		"status": "idle",
	})
}

// handleSyncStart handles starting sync.
func (s *Server) handleSyncStart(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{
		"status": "syncing",
	})
}

// handleSyncStop handles stopping sync.
func (s *Server) handleSyncStop(c *fiber.Ctx) error {
	return c.JSON(fiber.Map{
		"status": "stopped",
	})
}

// File handlers

// handleFileRead handles reading a file.
func (s *Server) handleFileRead(c *fiber.Ctx) error {
	path := c.Query("path")
	if path == "" {
		return c.Status(400).JSON(fiber.Map{"error": "path parameter required"})
	}

	content, err := os.ReadFile(path)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{
		"path":    path,
		"content": string(content),
	})
}

// handleFileWrite handles writing a file.
func (s *Server) handleFileWrite(c *fiber.Ctx) error {
	type WriteRequest struct {
		Path    string `json:"path"`
		Content string `json:"content"`
	}
	var req WriteRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid request"})
	}

	if req.Path == "" {
		return c.Status(400).JSON(fiber.Map{"error": "path required"})
	}

	err := os.WriteFile(req.Path, []byte(req.Content), 0644)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"path":    req.Path,
	})
}

// handleFileEdit handles editing a file.
func (s *Server) handleFileEdit(c *fiber.Ctx) error {
	type EditRequest struct {
		Path      string `json:"path"`
		OldString string `json:"old_string"`
		NewString string `json:"new_string"`
	}
	var req EditRequest
	if err := c.BodyParser(&req); err != nil {
		return c.Status(400).JSON(fiber.Map{"error": "Invalid request"})
	}

	// Read file
	content, err := os.ReadFile(req.Path)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	// Replace (simple string replacement)
	strContent := string(content)
	if !contains(strContent, req.OldString) {
		return c.Status(400).JSON(fiber.Map{"error": "old_string not found"})
	}

	newContent := replaceAll(strContent, req.OldString, req.NewString)

	err = os.WriteFile(req.Path, []byte(newContent), 0644)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{
		"success": true,
		"path":    req.Path,
	})
}

// handleFileGlob handles globbing files.
func (s *Server) handleFileGlob(c *fiber.Ctx) error {
	pattern := c.Query("pattern")
	if pattern == "" {
		pattern = "**/*"
	}

	// Simple glob implementation
	files := globFiles(s.directory, pattern)

	return c.JSON(files)
}

// handleFileGrep handles grepping files.
func (s *Server) handleFileGrep(c *fiber.Ctx) error {
	pattern := c.Query("pattern")
	path := c.Query("path")

	if pattern == "" {
		return c.Status(400).JSON(fiber.Map{"error": "pattern required"})
	}

	if path == "" {
		path = s.directory
	}

	// Simple grep implementation
	results := grepFiles(path, pattern)

	return c.JSON(results)
}

// handleFileList handles listing files.
func (s *Server) handleFileList(c *fiber.Ctx) error {
	path := c.Query("path")
	if path == "" {
		path = s.directory
	}

	entries, err := os.ReadDir(path)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	result := make([]fiber.Map, len(entries))
	for i, entry := range entries {
		info, _ := entry.Info()
		result[i] = fiber.Map{
			"name":  entry.Name(),
			"is_dir": entry.IsDir(),
			"size":  info.Size(),
		}
	}

	return c.JSON(result)
}

// handleFileInfo handles getting file info.
func (s *Server) handleFileInfo(c *fiber.Ctx) error {
	path := c.Query("path")
	if path == "" {
		return c.Status(400).JSON(fiber.Map{"error": "path required"})
	}

	info, err := os.Stat(path)
	if err != nil {
		return c.Status(500).JSON(fiber.Map{"error": err.Error()})
	}

	return c.JSON(fiber.Map{
		"name":     info.Name(),
		"size":     info.Size(),
		"is_dir":   info.IsDir(),
		"mod_time": info.ModTime().UnixMilli(),
	})
}

// Helper functions

func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func replaceAll(s, old, new string) string {
	result := ""
	for i := 0; i < len(s); {
		if i <= len(s)-len(old) && s[i:i+len(old)] == old {
			result += new
			i += len(old)
		} else {
			result += string(s[i])
			i++
		}
	}
	return result
}

func globFiles(dir, pattern string) []string {
	// Simple glob - just return directory contents for now
	entries, err := os.ReadDir(dir)
	if err != nil {
		return []string{}
	}

	files := make([]string, 0)
	for _, entry := range entries {
		files = append(files, entry.Name())
	}
	return files
}

func grepFiles(path, pattern string) []fiber.Map {
	// Simple grep - placeholder
	return []fiber.Map{}
}