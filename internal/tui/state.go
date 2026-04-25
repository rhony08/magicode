// Package tui provides state management for the terminal user interface.
// This file provides backward compatibility by re-exporting types from types package.
package tui

import (
	"github.com/rhony08/magicode/internal/tui/types"
)

// ===========================================
// Type re-exports for backward compatibility
// These allow the tui package to use types directly
// ===========================================

// Re-export types for use within tui package
type (
	Route             = types.Route
	SidebarMode       = types.SidebarMode
	KVStore           = types.KVStore
	SyncStore         = types.SyncStore
	LocalStore        = types.LocalStore
	LayoutStore       = types.LayoutStore
	DialogStore       = types.DialogStore
	AppState          = types.AppState
	DialogType        = types.DialogType
	DialogState       = types.DialogState
	Session           = types.Session
	Provider          = types.Provider
	Model             = types.Model
	Agent             = types.Agent
	ModelKey          = types.ModelKey
	LSPServer         = types.LSPServer
	MCPServer         = types.MCPServer
	InputPart         = types.InputPart
	ToastMsg          = types.ToastMsg
	Role              = types.Role
	Message           = types.Message
	Part              = types.Part
	ToolCall          = types.ToolCall
	SessionLocalState = types.SessionLocalState
	MessageMeta       = types.MessageMeta
)

// Re-export constants
const (
	RouteHome              = types.RouteHome
	RouteSession           = types.RouteSession
	SidebarModeAuto        = types.SidebarModeAuto
	SidebarModeShow        = types.SidebarModeShow
	SidebarModeHide        = types.SidebarModeHide
	ResponsiveThreshold    = types.ResponsiveThreshold
	RoleUser               = types.RoleUser
	RoleAssistant          = types.RoleAssistant
	RoleSystem             = types.RoleSystem
	RoleTool               = types.RoleTool
	InitialMessagePageSize = types.InitialMessagePageSize
	HistoryMessagePageSize = types.HistoryMessagePageSize
)

// Re-export dialog types
const (
	DialogSessionList = types.DialogSessionList
	DialogModelList   = types.DialogModelList
	DialogAgentList   = types.DialogAgentList
	DialogThemeList   = types.DialogThemeList
	DialogHelp        = types.DialogHelp
	DialogCommand     = types.DialogCommand
	DialogStatus      = types.DialogStatus
	DialogPermission  = types.DialogPermission
	DialogQuestion    = types.DialogQuestion
	DialogConfirm     = types.DialogConfirm
)

// Factory function wrappers for backward compatibility
// These call the underlying types package functions
var (
	NewAppState        = types.NewAppState
	DefaultKVStore     = types.DefaultKVStore
	DefaultSyncStore   = types.DefaultSyncStore
	DefaultLocalStore  = types.DefaultLocalStore
	DefaultLayoutStore = types.DefaultLayoutStore
	DefaultDialogStore = types.DefaultDialogStore
)
