// Package permission provides permission handling for tool execution.
package permission

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/rhony08/magicode/internal/bus"
	"github.com/rhony08/magicode/internal/util/log"
)

// Action represents the permission action
type Action string

const (
	ActionAllow Action = "allow" // Always allow
	ActionDeny  Action = "deny"  // Always deny
	ActionAsk   Action = "ask"   // Ask user for permission
)

// Reply represents the user's reply to a permission request
type Reply string

const (
	ReplyOnce    Reply = "once"    // Allow just this time
	ReplyAlways  Reply = "always"  // Always allow (adds to approved rules)
	ReplyReject  Reply = "reject"  // Reject the request
)

// Rule represents a permission rule
type Rule struct {
	Permission string `json:"permission"` // Tool name or wildcard pattern
	Pattern    string `json:"pattern"`    // Path pattern (or "*" for all)
	Action     Action `json:"action"`     // allow, deny, ask
}

// Ruleset is a collection of permission rules
type Ruleset []Rule

// Request represents a permission request
type Request struct {
	ID         string                 `json:"id"`
	SessionID  string                 `json:"session_id"`
	Permission string                 `json:"permission"` // Tool name
	Patterns   []string               `json:"patterns"`   // File paths affected
	Metadata   map[string]interface{} `json:"metadata"`   // Additional info
	Always     []string               `json:"always"`     // Patterns to always allow
	Tool       *ToolInfo              `json:"tool,omitempty"` // Tool call info
}

// ToolInfo contains tool-specific information
type ToolInfo struct {
	MessageID string `json:"message_id"`
	CallID    string `json:"call_id"`
}

// ReplyBody represents a reply to a permission request
type ReplyBody struct {
	Reply   Reply  `json:"reply"`
	Message string `json:"message,omitempty"` // Feedback if rejected
}

// Error types for permission
type DeniedError struct {
	Ruleset Ruleset `json:"ruleset"`
}

func (e *DeniedError) Error() string {
	return fmt.Sprintf("The user has specified a rule which prevents this tool call: %v", e.Ruleset)
}

type RejectedError struct{}

func (e *RejectedError) Error() string {
	return "The user rejected permission to use this specific tool call"
}

type CorrectedError struct {
	Feedback string `json:"feedback"`
}

func (e *CorrectedError) Error() string {
	return fmt.Sprintf("The user rejected permission with feedback: %s", e.Feedback)
}

// Event definitions
var (
	EventPermissionAsked   = bus.Definition{Type: "permission.asked"}
	EventPermissionReplied = bus.Definition{Type: "permission.replied"}
)

// PendingEntry tracks a pending permission request
type PendingEntry struct {
	Info     Request
	Response chan ReplyBody
}

// Service handles permission requests
type Service struct {
	bus     *bus.Service
	logger  *log.Logger
	mu      sync.Mutex
	pending map[string]*PendingEntry
	approved Ruleset
}

// NewService creates a new permission service
func NewService(bus *bus.Service, initialRuleset Ruleset) *Service {
	return &Service{
		bus:      bus,
		pending:  make(map[string]*PendingEntry),
		approved: initialRuleset,
		logger:   log.Create(map[string]string{"service": "permission"}),
	}
}

// Ask requests permission for a tool call
// Returns nil if allowed, error if denied/rejected
func (s *Service) Ask(ctx context.Context, input Request, ruleset Ruleset) error {
	s.mu.Lock()
	approved := s.approved
	s.mu.Unlock()

	// Evaluate rules for each pattern
	for _, pattern := range input.Patterns {
		rule := s.Evaluate(input.Permission, pattern, ruleset, approved)
		
		s.logger.Info("Evaluated permission", map[string]interface{}{
			"permission": input.Permission,
			"pattern":    pattern,
			"action":     rule.Action,
		})

		if rule.Action == ActionDeny {
			return &DeniedError{Ruleset: s.filterRuleset(ruleset, input.Permission)}
		}

		if rule.Action == ActionAllow {
			continue
		}

		// Action is "ask" - need to request user permission
		return s.askUser(ctx, input)
	}

	// All patterns allowed by ruleset
	return nil
}

// askUser sends a permission request and waits for reply
func (s *Service) askUser(ctx context.Context, input Request) error {
	// Generate ID if not provided
	if input.ID == "" {
		input.ID = fmt.Sprintf("perm_%d", time.Now().UnixNano())
	}

	// Create response channel
	responseChan := make(chan ReplyBody, 1)

	// Register pending request
	s.mu.Lock()
	s.pending[input.ID] = &PendingEntry{
		Info:     input,
		Response: responseChan,
	}
	s.mu.Unlock()

	// Publish asked event
	if s.bus != nil {
		s.bus.Publish(EventPermissionAsked, input)
	}

	s.logger.Info("Asking permission", map[string]interface{}{
		"id":         input.ID,
		"permission": input.Permission,
		"patterns":   input.Patterns,
	})

	// Wait for response or context cancellation
	select {
	case reply := <-responseChan:
		s.mu.Lock()
		delete(s.pending, input.ID)
		s.mu.Unlock()

		switch reply.Reply {
		case ReplyReject:
			if reply.Message != "" {
				return &CorrectedError{Feedback: reply.Message}
			}
			return &RejectedError{}

		case ReplyOnce:
			return nil

		case ReplyAlways:
			// Add always patterns to approved ruleset
			s.mu.Lock()
			for _, pattern := range input.Always {
				s.approved = append(s.approved, Rule{
					Permission: input.Permission,
					Pattern:    pattern,
					Action:     ActionAllow,
				})
			}
			// Auto-approve other pending requests for same session
			s.autoApproveSession(input.SessionID)
			s.mu.Unlock()
			return nil
		}

	case <-ctx.Done():
		s.mu.Lock()
		delete(s.pending, input.ID)
		s.mu.Unlock()
		return ctx.Err()
	}

	return nil
}

// Reply handles a user's reply to a permission request
func (s *Service) Reply(input ReplyInput) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	existing, ok := s.pending[input.RequestID]
	if !ok {
		return fmt.Errorf("request not found: %s", input.RequestID)
	}

	// Remove from pending
	delete(s.pending, input.RequestID)

	// Publish replied event
	if s.bus != nil {
		s.bus.Publish(EventPermissionReplied, map[string]interface{}{
			"session_id": existing.Info.SessionID,
			"request_id": input.RequestID,
			"reply":      input.Reply,
		})
	}

	// Handle rejection - reject all pending requests for this session
	if input.Reply == ReplyReject {
		for id, entry := range s.pending {
			if entry.Info.SessionID == existing.Info.SessionID {
				delete(s.pending, id)
				entry.Response <- ReplyBody{Reply: ReplyReject}
			}
		}
		existing.Response <- ReplyBody{Reply: ReplyReject, Message: input.Message}
		return nil
	}

	// Allow once or always
	existing.Response <- ReplyBody{Reply: input.Reply}

	// If always, add to approved ruleset
	if input.Reply == ReplyAlways {
		for _, pattern := range existing.Info.Always {
			s.approved = append(s.approved, Rule{
				Permission: existing.Info.Permission,
				Pattern:    pattern,
				Action:     ActionAllow,
			})
		}

		// Auto-approve other pending requests for same session that match
		s.autoApproveSession(existing.Info.SessionID)
	}

	return nil
}

// ReplyInput represents a reply input
type ReplyInput struct {
	RequestID string `json:"request_id"`
	Reply     Reply  `json:"reply"`
	Message   string `json:"message,omitempty"`
}

// autoApproveSession auto-approves pending requests for a session if rules allow
func (s *Service) autoApproveSession(sessionID string) {
	for id, entry := range s.pending {
		if entry.Info.SessionID != sessionID {
			continue
		}

		// Check if all patterns are now allowed
		allAllowed := true
		for _, pattern := range entry.Info.Patterns {
			rule := s.Evaluate(entry.Info.Permission, pattern, s.approved)
			if rule.Action != ActionAllow {
				allAllowed = false
				break
			}
		}

		if allAllowed {
			delete(s.pending, id)
			if s.bus != nil {
				s.bus.Publish(EventPermissionReplied, map[string]interface{}{
					"session_id": sessionID,
					"request_id": id,
					"reply":      ReplyAlways,
				})
			}
			entry.Response <- ReplyBody{Reply: ReplyAlways}
		}
	}
}

// List returns all pending permission requests
func (s *Service) List() []Request {
	s.mu.Lock()
	defer s.mu.Unlock()

	result := make([]Request, 0, len(s.pending))
	for _, entry := range s.pending {
		result = append(result, entry.Info)
	}
	return result
}

// Evaluate evaluates permission rules to find matching action
// Uses last matching rule (specific overrides wildcard)
func (s *Service) Evaluate(permission, pattern string, rulesets ...Ruleset) Rule {
	// Flatten all rulesets
	allRules := make(Ruleset, 0)
	for _, rs := range rulesets {
		allRules = append(allRules, rs...)
	}

	// Find last matching rule (specific rules should override wildcards)
	var lastMatch Rule
	for _, rule := range allRules {
		if s.matchPermission(permission, rule.Permission) && s.matchPattern(pattern, rule.Pattern) {
			lastMatch = rule
		}
	}

	if lastMatch.Action != "" {
		return lastMatch
	}

	// Default to ask if no matching rule
	return Rule{Action: ActionAsk}
}

// matchPermission checks if permission matches rule permission (with wildcard support)
func (s *Service) matchPermission(permission, rulePermission string) bool {
	if rulePermission == "*" {
		return true
	}
	if rulePermission == permission {
		return true
	}
	// Simple wildcard matching (e.g., "mcp_*" matches "mcp_file_read")
	if len(rulePermission) > 0 && rulePermission[len(rulePermission)-1] == '*' {
		prefix := rulePermission[:len(rulePermission)-1]
		return len(permission) >= len(prefix) && permission[:len(prefix)] == prefix
	}
	return false
}

// matchPattern checks if pattern matches rule pattern (with wildcard support)
func (s *Service) matchPattern(pattern, rulePattern string) bool {
	if rulePattern == "*" {
		return true
	}
	return pattern == rulePattern
}

// filterRuleset filters ruleset to only include rules matching permission
func (s *Service) filterRuleset(ruleset Ruleset, permission string) Ruleset {
	result := make(Ruleset, 0)
	for _, rule := range ruleset {
		if s.matchPermission(permission, rule.Permission) {
			result = append(result, rule)
		}
	}
	return result
}

// SetApprovedRuleset sets the approved ruleset (from DB)
func (s *Service) SetApprovedRuleset(ruleset Ruleset) {
	s.mu.Lock()
	s.approved = ruleset
	s.mu.Unlock()
}

// GetApprovedRuleset returns the current approved ruleset
func (s *Service) GetApprovedRuleset() Ruleset {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.approved
}

// ClearPending clears all pending requests (for cleanup)
func (s *Service) ClearPending() {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, entry := range s.pending {
		entry.Response <- ReplyBody{Reply: ReplyReject}
	}
	s.pending = make(map[string]*PendingEntry)
}