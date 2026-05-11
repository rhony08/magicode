// Package database provides SQLite database operations with schema definitions
// compatible with the TypeScript version of MagiCode.
package database

import (
	"time"
)

// Timestamps represents common timestamp fields used in all tables
type Timestamps struct {
	TimeCreated int64 `json:"time_created"`
	TimeUpdated int64 `json:"time_updated"`
}

// NewTimestamps creates a new Timestamps with current time
func NewTimestamps() Timestamps {
	now := time.Now().UnixMilli()
	return Timestamps{
		TimeCreated: now,
		TimeUpdated: now,
	}
}

// UpdateTimestamps updates the TimeUpdated field
func (t *Timestamps) UpdateTimestamps() {
	t.TimeUpdated = time.Now().UnixMilli()
}

// ===========================================
// Project Table
// ===========================================

// Project represents a project record
type Project struct {
	ID              string     `json:"id"`
	Worktree        string     `json:"worktree"`   // Required
	VCS             string     `json:"vcs"`        // Optional: git, etc.
	Name            string     `json:"name"`       // Optional
	IconURL         string     `json:"icon_url"`   // Optional
	IconColor       string     `json:"icon_color"` // Optional
	Timestamps      Timestamps `json:"timestamps"`
	TimeInitialized int64      `json:"time_initialized"` // Optional
	Sandboxes       []string   `json:"sandboxes"`        // JSON array
	Commands        *Commands  `json:"commands"`         // Optional JSON
}

// Commands represents project commands configuration
type Commands struct {
	Start string `json:"start,omitempty"`
}

// ===========================================
// Session Table
// ===========================================

// Session represents a conversation session
type Session struct {
	ID               string      `json:"id"`
	ProjectID        string      `json:"project_id"`        // Required, FK to project
	WorkspaceID      string      `json:"workspace_id"`      // Optional
	ParentID         string      `json:"parent_id"`         // Optional, for forked sessions
	Slug             string      `json:"slug"`              // Required
	Directory        string      `json:"directory"`         // Required
	Title            string      `json:"title"`             // Required
	Version          string      `json:"version"`           // Required
	ShareURL         string      `json:"share_url"`         // Optional
	SummaryAdditions int         `json:"summary_additions"` // Optional
	SummaryDeletions int         `json:"summary_deletions"` // Optional
	SummaryFiles     int         `json:"summary_files"`     // Optional
	SummaryDiffs     []FileDiff  `json:"summary_diffs"`     // Optional JSON array
	Revert           *RevertInfo `json:"revert"`            // Optional JSON
	Permission       *Ruleset    `json:"permission"`        // Optional JSON
	Timestamps       Timestamps  `json:"timestamps"`
	TimeCompacting   int64       `json:"time_compacting"` // Optional
	TimeArchived     int64       `json:"time_archived"`   // Optional
}

// FileDiff represents a file diff entry
type FileDiff struct {
	Path      string `json:"path"`
	Additions int    `json:"additions"`
	Deletions int    `json:"deletions"`
}

// RevertInfo represents revert configuration
type RevertInfo struct {
	MessageID string `json:"messageID"`
	PartID    string `json:"partID,omitempty"`
	Snapshot  string `json:"snapshot,omitempty"`
	Diff      string `json:"diff,omitempty"`
}

// Ruleset represents permission ruleset (simplified for storage)
type Ruleset struct {
	Allow []string `json:"allow,omitempty"`
	Deny  []string `json:"deny,omitempty"`
}

// ===========================================
// Message Table
// ===========================================

// Message represents a message in a session
type Message struct {
	ID         string      `json:"id"`
	SessionID  string      `json:"session_id"` // Required, FK to session
	Timestamps Timestamps  `json:"timestamps"`
	Data       MessageInfo `json:"data"` // JSON
}

// MessageInfo represents message metadata (matches actual database schema)
type MessageInfo struct {
	Role       string                 `json:"role"`       // user, assistant
	ParentID   string                 `json:"parentID"`   // Optional
	Agent      string                 `json:"agent"`      // Optional
	Mode       string                 `json:"mode"`       // Optional (build, etc.)
	ModelID    string                 `json:"modelID"`    // Optional
	ProviderID string                 `json:"providerID"` // Optional
	Cost       int64                  `json:"cost"`       // Optional
	Tokens     map[string]interface{} `json:"tokens"`     // Optional object (total, input, output, etc.)
	Time       map[string]interface{} `json:"time"`       // Optional object with created/completed
	Finish     string                 `json:"finish"`     // Optional (tool-calls, end, etc.)
	Path       map[string]interface{} `json:"path"`       // Optional object with cwd/root
	Summary    map[string]interface{} `json:"summary"`    // Optional
	Compacted  bool                   `json:"compacted"`  // Whether message was compacted (summarized)
}

// ===========================================
// Part Table
// ===========================================

// Part represents a part of a message
type Part struct {
	ID         string     `json:"id"`
	MessageID  string     `json:"message_id"` // Required, FK to message
	SessionID  string     `json:"session_id"` // Required
	Timestamps Timestamps `json:"timestamps"`
	Data       PartData   `json:"data"` // JSON
}

// PartData represents part content (matches actual database schema)
// Part types: text, reasoning (thinking), file, tool_use, tool_result, step-start, step-finish, patch
// Note: OpenCode uses "reasoning" type for thinking blocks (not "thinking")
type PartData struct {
	Type       string                 `json:"type"`       // text, reasoning, file, tool_use, tool_result, step-start, etc.
	Text       string                 `json:"text"`       // For text/reasoning parts
	Synthetic  bool                   `json:"synthetic"`  // Optional flag for synthetic parts
	
	// Reasoning-specific fields (for thinking blocks)
	ReasoningMetadata map[string]interface{} `json:"reasoning_metadata,omitempty"` // Provider metadata for reasoning
	ReasoningTime     *ReasoningTime         `json:"reasoning_time,omitempty"`     // Time tracking for reasoning
	
	// File-specific fields
	Mime       string                 `json:"mime"`       // For file parts
	Filename   string                 `json:"filename"`   // For file parts
	URL        string                 `json:"url"`        // For file parts
	Source     map[string]interface{} `json:"source"`     // For file parts
	
	// Tool-specific fields
	ToolID     string                 `json:"toolID"`     // For tool parts
	ToolName   string                 `json:"toolName"`   // For tool parts
	ToolInput  map[string]any         `json:"toolInput"`  // For tool_use
	ToolResult string                 `json:"toolResult"` // For tool_result
	Status     string                 `json:"status"`     // pending, running, success, error, blocked
	Error      string                 `json:"error"`      // For error status
	
	// Step-specific fields
	SnapshotID string                 `json:"snapshot_id,omitempty"` // For step-start/step-finish
	StepReason string                 `json:"step_reason,omitempty"` // For step-finish
	
	// Compaction-specific fields
	CompactionAuto    bool   `json:"compaction_auto,omitempty"`    // Whether auto-compaction
	CompactionTailID  string `json:"compaction_tail_id,omitempty"` // Tail start ID
}

// ReasoningTime tracks timing for reasoning/thinking blocks
type ReasoningTime struct {
	Start int64 `json:"start"`           // Start timestamp (ms)
	End   int64 `json:"end,omitempty"`   // End timestamp (ms)
}

// ===========================================
// Todo Table
// ===========================================

// Todo represents a todo item in a session
type Todo struct {
	SessionID  string     `json:"session_id"` // Required, FK to session
	Content    string     `json:"content"`    // Required
	Status     string     `json:"status"`     // pending, in_progress, completed, cancelled
	Priority   string     `json:"priority"`   // high, medium, low
	Position   int        `json:"position"`   // Required for ordering
	Timestamps Timestamps `json:"timestamps"`
}

// ===========================================
// Session Entry Table
// ===========================================

// SessionEntry represents an entry in a session (for event tracking)
type SessionEntry struct {
	ID         string     `json:"id"`
	SessionID  string     `json:"session_id"` // Required, FK to session
	Type       string     `json:"type"`       // Entry type
	Timestamps Timestamps `json:"timestamps"`
	Data       EntryData  `json:"data"` // JSON
}

// EntryData represents session entry data (generic)
type EntryData map[string]any

// ===========================================
// Permission Table
// ===========================================

// Permission represents project-level permission settings
type Permission struct {
	ProjectID  string     `json:"project_id"` // PK, FK to project
	Timestamps Timestamps `json:"timestamps"`
	Data       Ruleset    `json:"data"` // JSON
}

// ===========================================
// Event Tables (for sync)
// ===========================================

// EventSequence represents event sequence tracking
type EventSequence struct {
	AggregateID string `json:"aggregate_id"` // PK
	Seq         int    `json:"seq"`          // Sequence number
}

// Event represents a sync event
type Event struct {
	ID          string         `json:"id"`           // PK
	AggregateID string         `json:"aggregate_id"` // FK to event_sequence
	Seq         int            `json:"seq"`          // Sequence within aggregate
	Type        string         `json:"type"`         // Event type
	Data        map[string]any `json:"data"`         // JSON event data
}

// ===========================================
// Account Tables
// ===========================================

// Account represents a user account
type Account struct {
	ID           string     `json:"id"`            // PK
	Email        string     `json:"email"`         // Required
	URL          string     `json:"url"`           // Required (server URL)
	AccessToken  string     `json:"access_token"`  // Required
	RefreshToken string     `json:"refresh_token"` // Required
	TokenExpiry  int64      `json:"token_expiry"`  // Optional
	Timestamps   Timestamps `json:"timestamps"`
}

// AccountState represents active account state (single row)
type AccountState struct {
	ID              int    `json:"id"`                // PK (always 1)
	ActiveAccountID string `json:"active_account_id"` // FK to account
	ActiveOrgID     string `json:"active_org_id"`     // Optional
}

// ControlAccount represents legacy account (multi-account by email+url)
type ControlAccount struct {
	Email        string     `json:"email"`         // Part of PK
	URL          string     `json:"url"`           // Part of PK
	AccessToken  string     `json:"access_token"`  // Required
	RefreshToken string     `json:"refresh_token"` // Required
	TokenExpiry  int64      `json:"token_expiry"`  // Optional
	Active       bool       `json:"active"`        // Default false
	Timestamps   Timestamps `json:"timestamps"`
}

// ===========================================
// Share Table
// ===========================================

// SessionShare represents a shared session
type SessionShare struct {
	SessionID  string     `json:"session_id"` // PK, FK to session
	ID         string     `json:"id"`         // Share ID
	Secret     string     `json:"secret"`     // Share secret
	URL        string     `json:"url"`        // Share URL
	Timestamps Timestamps `json:"timestamps"`
}

// ===========================================
// Workspace Table
// ===========================================

// Workspace represents a workspace linked to a project
type Workspace struct {
	ID        string         `json:"id"`         // PK
	Type      string         `json:"type"`       // Required
	Name      string         `json:"name"`       // Default ""
	Branch    string         `json:"branch"`     // Optional
	Directory string         `json:"directory"`  // Optional
	Extra     map[string]any `json:"extra"`      // Optional JSON
	ProjectID string         `json:"project_id"` // Required, FK to project
}
