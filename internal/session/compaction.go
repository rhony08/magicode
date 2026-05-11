// Package session provides compaction handling for token overflow.
package session

import (
	"context"
	"strings"

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

// CompactionResult represents the result of a compaction operation
type CompactionResult struct {
	Continue    bool   // Whether to continue processing
	Stop        bool   // Whether to stop processing
	TailStartID string // ID of the message where tail starts
	Summary     string // Generated summary text
}

// SelectMessages selects messages to be compacted
// Returns head (messages to summarize) and tail_start_id (ID where recent messages start)
func (s *CompactionService) SelectMessages(messages []database.Message, tailTurns int) ([]database.Message, string) {
	if len(messages) <= 2 {
		return messages, ""
	}

	// Find user messages (turn boundaries)
	turns := []struct {
		start int
		id    string
	}{}

	for i, msg := range messages {
		if msg.Data.Role == "user" {
			turns = append(turns, struct {
				start int
				id    string
			}{start: i, id: msg.ID})
		}
	}

	if len(turns) <= tailTurns {
		return messages, ""
	}

	// Keep the last tailTurns turns as "tail"
	keepTurn := turns[len(turns)-tailTurns]

	return messages[:keepTurn.start], keepTurn.id
}

// PruneToolOutputs truncates large tool outputs to free context space
// Goes backwards through parts until PRUNE_PROTECT tokens worth of tool calls
func (s *CompactionService) PruneToolOutputs(ctx context.Context, sessionID string) error {
	if s.db == nil {
		return nil
	}

	msgStorage := database.NewMessageStorage(s.db)
	partStorage := database.NewPartStorage(s.db)

	messages, err := msgStorage.List(ctx, sessionID)
	if err != nil {
		return err
	}

	pruned := 0
	for i := len(messages) - 1; i >= 0; i-- {
		msg := messages[i]
		if msg.Data.Role != "assistant" {
			continue
		}

		parts, err := partStorage.ListByMessage(ctx, msg.ID)
		if err != nil {
			continue
		}

		for j := len(parts) - 1; j >= 0; j-- {
			part := parts[j]
			if part.Data.Type == "tool_result" && len(part.Data.ToolResult) > ToolOutputMaxChars {
				// Truncate tool output
				part.Data.ToolResult = TruncateToolOutput(part.Data.ToolResult, ToolOutputMaxChars)
				partStorage.Update(ctx, part)
				pruned++
			}
		}

		// Stop after pruning enough tokens
		if pruned >= 10 {
			break
		}
	}

	s.logger.Info("Pruned tool outputs", map[string]interface{}{
		"session_id": sessionID,
		"pruned":     pruned,
	})

	return nil
}

// BuildCompactionPrompt builds the prompt for generating a summary
func (s *CompactionService) BuildCompactionPrompt(previousSummary string, context []string) string {
	anchor := ""
	if previousSummary != "" {
		anchor = `Update the anchored summary below using the conversation history above.
Preserve still-true details, remove stale details, and merge in the new facts.
<previous-summary>
` + previousSummary + `
</previous-summary>`
	} else {
		anchor = "Create a new anchored summary from the conversation history above."
	}

	parts := []string{anchor, GetCompactionSummaryPrompt()}
	parts = append(parts, context...)

	return strings.Join(parts, "\n\n")
}

// CreateCompactionMessage creates a compaction user message in the database
// This marks the start of a compacted section
func (s *CompactionService) CreateCompactionMessage(ctx context.Context, sessionID, tailStartID string, auto bool) (*database.Message, error) {
	if s.db == nil {
		return nil, nil
	}

	msgStorage := database.NewMessageStorage(s.db)
	partStorage := database.NewPartStorage(s.db)

	// Create user message for compaction marker
	msg := database.Message{
		SessionID: sessionID,
		Data: database.MessageInfo{
			Role: "user",
		},
	}

	createdMsg, err := msgStorage.Create(ctx, msg)
	if err != nil {
		return nil, err
	}

	// Create compaction part
	part := database.Part{
		MessageID: createdMsg.ID,
		SessionID: sessionID,
		Data: database.PartData{
			Type:            "compaction",
			CompactionAuto:  auto,
			CompactionTailID: tailStartID,
		},
	}

	_, err = partStorage.Create(ctx, part)
	if err != nil {
		return nil, err
	}

	s.logger.Info("Created compaction message", map[string]interface{}{
		"session_id":    sessionID,
		"message_id":    createdMsg.ID,
		"tail_start_id": tailStartID,
		"auto":          auto,
	})

	return createdMsg, nil
}

// CreateSummaryMessage creates an assistant message with the summary
func (s *CompactionService) CreateSummaryMessage(ctx context.Context, sessionID, parentID, summary string) (*database.Message, error) {
	if s.db == nil {
		return nil, nil
	}

	msgStorage := database.NewMessageStorage(s.db)
	partStorage := database.NewPartStorage(s.db)

	// Create assistant message
	msg := database.Message{
		SessionID: sessionID,
		Data: database.MessageInfo{
			Role:       "assistant",
			ParentID:   parentID,
			Finish:     "stop",
		},
	}

	createdMsg, err := msgStorage.Create(ctx, msg)
	if err != nil {
		return nil, err
	}

	// Create text part with summary
	part := database.Part{
		MessageID: createdMsg.ID,
		SessionID: sessionID,
		Data: database.PartData{
			Type: "text",
			Text: summary,
		},
	}

	_, err = partStorage.Create(ctx, part)
	if err != nil {
		return nil, err
	}

	return createdMsg, nil
}

// MarkCompacted marks old messages as compacted (excluded from future requests)
func (s *CompactionService) MarkCompacted(ctx context.Context, sessionID string, upToMessageID string) error {
	if s.db == nil {
		return nil
	}

	msgStorage := database.NewMessageStorage(s.db)

	messages, err := msgStorage.List(ctx, sessionID)
	if err != nil {
		return err
	}

	marked := 0
	for _, msg := range messages {
		if msg.ID == upToMessageID {
			break
		}
		msg.Data.Compacted = true
		msgStorage.Update(ctx, msg)
		marked++
	}

	s.logger.Info("Marked messages as compacted", map[string]interface{}{
		"session_id":     sessionID,
		"marked_count":   marked,
		"up_to_message":  upToMessageID,
	})

	return nil
}

// GetCompletedCompactions returns completed compaction summaries
func (s *CompactionService) GetCompletedCompactions(ctx context.Context, sessionID string) ([]string, error) {
	if s.db == nil {
		return nil, nil
	}

	msgStorage := database.NewMessageStorage(s.db)
	partStorage := database.NewPartStorage(s.db)

	messages, err := msgStorage.List(ctx, sessionID)
	if err != nil {
		return nil, err
	}

	summaries := []string{}
	for _, msg := range messages {
		if msg.Data.Role != "assistant" {
			continue
		}

		parts, err := partStorage.ListByMessage(ctx, msg.ID)
		if err != nil {
			continue
		}

		// Find compaction marker in parent user message
		for _, part := range parts {
			if part.Data.Type == "compaction" {
				// Get the previous assistant message's summary
				summaries = append(summaries, s.extractSummary(ctx, msg.ID))
			}
		}
	}

	return summaries, nil
}

// extractSummary extracts summary text from an assistant message
func (s *CompactionService) extractSummary(ctx context.Context, messageID string) string {
	if s.db == nil {
		return ""
	}

	partStorage := database.NewPartStorage(s.db)
	parts, err := partStorage.ListByMessage(ctx, messageID)
	if err != nil {
		return ""
	}

	for _, part := range parts {
		if part.Data.Type == "text" {
			return part.Data.Text
		}
	}

	return ""
}