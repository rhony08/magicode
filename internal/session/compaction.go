// Package session provides compaction handling for token overflow.
package session

import (
	"context"

	"github.com/rhony08/magicode/internal/bus"
	"github.com/rhony08/magicode/internal/database"
	"github.com/rhony08/magicode/internal/provider"
	"github.com/rhony08/magicode/internal/util/log"
)

// Compaction constants (matching OpenCode)
const (
	CompactionBuffer    = 20000  // Reserved tokens for response
	MinPreserveTokens   = 2000   // Minimum recent tokens to preserve
	MaxPreserveTokens   = 8000   // Maximum recent tokens to preserve
	PruneMinimum        = 20000  // Minimum tokens before pruning
	PruneProtect        = 40000  // Protected tokens from pruning
	ToolOutputMaxChars  = 2000   // Max chars in tool output for compaction
)

// CompactionConfig holds compaction settings
type CompactionConfig struct {
	Auto     bool // Enable auto-compaction (default: true)
	Reserved int  // Reserved tokens (default: 20000)
}

// CompactionService handles token overflow compaction
type CompactionService struct {
	db      *database.Database
	bus     *bus.Service
	logger  *log.Logger
	config  CompactionConfig
}

// NewCompactionService creates a new compaction service
func NewCompactionService(db *database.Database, bus *bus.Service, config CompactionConfig) *CompactionService {
	if config.Reserved == 0 {
		config.Reserved = CompactionBuffer
	}
	if !config.Auto {
		config.Auto = true // Default to auto
	}

	return &CompactionService{
		db:     db,
		bus:    bus,
		config: config,
		logger: log.Create(map[string]string{"service": "session.compaction"}),
	}
}

// TokenUsage represents token usage for a message
type TokenUsage struct {
	Input      int
	Output     int
	Reasoning  int
	CacheRead  int
	CacheWrite int
	Total      int
}

// IsOverflow checks if token usage exceeds model context limit
// Matches OpenCode's overflow.isOverflow logic
func (s *CompactionService) IsOverflow(tokens TokenUsage, model provider.ModelInfo) bool {
	if !s.config.Auto {
		return false
	}

	// If context limit is 0, can't determine overflow
	// Use MaxTokens as context limit (or MaxInputTokens if available)
	contextLimit := model.MaxTokens
	if model.MaxInputTokens > 0 {
		contextLimit = model.MaxInputTokens
	}
	if contextLimit == 0 {
		return false
	}

	// Calculate usable tokens (context minus reserved)
	usable := s.calculateUsableTokens(model)

	// Count total tokens
	count := tokens.Total
	if count == 0 {
		count = tokens.Input + tokens.Output + tokens.CacheRead + tokens.CacheWrite
	}

	return count >= usable
}

// calculateUsableTokens returns the usable context tokens
// Matches OpenCode's overflow.usable logic
func (s *CompactionService) calculateUsableTokens(model provider.ModelInfo) int {
	// Get context limit from MaxTokens or MaxInputTokens
	contextLimit := model.MaxTokens
	if model.MaxInputTokens > 0 {
		contextLimit = model.MaxInputTokens
	}
	if contextLimit == 0 {
		return 0
	}

	reserved := s.config.Reserved
	if reserved == 0 {
		reserved = CompactionBuffer
	}

	// If model has explicit input limit, use that
	if model.MaxInputTokens > 0 {
		usable := model.MaxInputTokens - reserved
		if usable < 0 {
			return 0
		}
		return usable
	}

	// Otherwise, context minus max output tokens
	maxOutput := model.MaxOutputTokens
	if maxOutput == 0 {
		maxOutput = 4096 // Default max output
	}

	usable := contextLimit - maxOutput
	if usable < 0 {
		return 0
	}
	return usable
}

// CreateCompaction creates a compaction request for a session
// This triggers the AI to summarize old content and prune tokens
func (s *CompactionService) CreateCompaction(ctx context.Context, sessionID string, agent string, model provider.ModelInfo) error {
	s.logger.Info("Creating compaction", map[string]interface{}{
		"session_id": sessionID,
		"agent":      agent,
		"model_id":   model.ID,
	})

	// Publish compaction event
	if s.bus != nil {
		s.bus.Publish(EventCompactionCreated, map[string]interface{}{
			"session_id": sessionID,
			"agent":      agent,
			"model_id":   string(model.ID),
		})
	}

	return nil
}

// NeedsCompaction checks if a session needs compaction based on latest tokens
func (s *CompactionService) NeedsCompaction(sessionID string, tokens TokenUsage, model provider.ModelInfo) bool {
	return s.IsOverflow(tokens, model)
}

// TruncateToolOutput truncates tool output for compaction display
func TruncateToolOutput(output string, maxChars int) string {
	if maxChars == 0 {
		maxChars = ToolOutputMaxChars
	}

	if len(output) <= maxChars {
		return output
	}

	return output[:maxChars] + "\n... (truncated)"
}

// GetCompactionSummaryPrompt returns the prompt template for compaction summaries
// Matches OpenCode's SUMMARY_TEMPLATE
func GetCompactionSummaryPrompt() string {
	return `Output exactly this Markdown structure and keep the section order unchanged:
---
## Goal
- [single-sentence task summary]

## Constraints & Preferences
- [user constraints, preferences, specs, or "(none)"]

## Progress
### Done
- [completed work or "(none)"]

### In Progress
- [current work or "(none)"]

### Blocked
- [blockers or "(none)"]

## Key Decisions
- [decision and why, or "(none)"]

## Next Steps
- [ordered next actions or "(none)"]

## Critical Context
- [important technical facts, errors, open questions, or "(none)"]

## Relevant Files
- [file or directory path: why it matters, or "(none)"]
---

Rules:
- Keep every section, even when empty.
- Use terse bullets, not prose paragraphs.
- Preserve exact file paths, commands, error strings, and identifiers when known.
- Do not mention the summary process or that context was compacted.`
}