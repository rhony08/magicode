// Package tool provides permission checking for tools.
package tool

import (
	"path/filepath"
	"strings"
)

// PermissionAction represents the action to take
type PermissionAction string

const (
	PermissionAllow PermissionAction = "allow"
	PermissionDeny  PermissionAction = "deny"
	PermissionAsk   PermissionAction = "ask"
)

// Rule represents a permission rule
type Rule struct {
	ToolPattern string         `json:"tool"`   // Tool pattern (supports wildcards)
	PathPattern  string         `json:"path"`   // Path pattern (supports wildcards)
	Action       PermissionAction `json:"action"`
}

// Ruleset represents a collection of permission rules
type Ruleset struct {
	Allow []Rule `json:"allow,omitempty"`
	Deny  []Rule `json:"deny,omitempty"`
}

// PermissionChecker checks permissions for tool execution
type PermissionChecker struct {
	rulesets []Ruleset
}

// NewPermissionChecker creates a new permission checker
func NewPermissionChecker(rulesets ...Ruleset) *PermissionChecker {
	return &PermissionChecker{
		rulesets: rulesets,
	}
}

// Check checks if a tool action is permitted
func (p *PermissionChecker) Check(toolID ToolID, path string) PermissionAction {
	// Flatten all rules
	rules := p.flattenRules()

	// Find matching rule (last match wins)
	for i := len(rules) - 1; i >= 0; i-- {
		rule := rules[i]
		if wildcardMatch(string(toolID), rule.ToolPattern) {
			// Normalize path for comparison
			normalizedPath := filepath.Clean(path)
			if wildcardMatch(normalizedPath, rule.PathPattern) {
				return rule.Action
			}
		}
	}

	// Default: ask for permission
	return PermissionAsk
}

// flattenRules flattens allow and deny rules into a single list
func (p *PermissionChecker) flattenRules() []Rule {
	rules := []Rule{}

	for _, ruleset := range p.rulesets {
		for _, rule := range ruleset.Allow {
			rule.Action = PermissionAllow
			rules = append(rules, rule)
		}
		for _, rule := range ruleset.Deny {
			rule.Action = PermissionDeny
			rules = append(rules, rule)
		}
	}

	return rules
}

// wildcardMatch checks if a string matches a wildcard pattern
func wildcardMatch(str, pattern string) bool {
	// Handle special patterns
	if pattern == "*" {
		return true
	}
	if pattern == "" {
		return str == ""
	}

	// Simple wildcard matching (supports * but not ?)
	parts := strings.Split(pattern, "*")
	if len(parts) == 1 {
		return str == pattern
	}

	// Check prefix
	if !strings.HasPrefix(str, parts[0]) {
		return false
	}

	// Check suffix
	if !strings.HasSuffix(str, parts[len(parts)-1]) {
		return false
	}

	// Check middle parts
	idx := len(parts[0])
	for i := 1; i < len(parts)-1; i++ {
		part := parts[i]
		found := strings.Index(str[idx:], part)
		if found < 0 {
			return false
		}
		idx += found + len(part)
	}

	return true
}

// DefaultRuleset provides sensible defaults
var DefaultRuleset = Ruleset{
	Allow: []Rule{
		{ToolPattern: "read", PathPattern: "*", Action: PermissionAllow},
		{ToolPattern: "grep", PathPattern: "*", Action: PermissionAllow},
		{ToolPattern: "glob", PathPattern: "*", Action: PermissionAllow},
	},
	Deny: []Rule{
		// Deny dangerous paths
		{ToolPattern: "bash", PathPattern: "/etc/*", Action: PermissionAsk},
		{ToolPattern: "write", PathPattern: "/etc/*", Action: PermissionDeny},
		{ToolPattern: "edit", PathPattern: "/etc/*", Action: PermissionDeny},
	},
}

// PermissionRequest represents a permission request
type PermissionRequest struct {
	ToolID  ToolID         `json:"tool_id"`
	Path    string         `json:"path"`
	Action  PermissionAction `json:"action"`
	Reason  string         `json:"reason,omitempty"`
}

// CheckPermission is a convenience function
func CheckPermission(toolID ToolID, path string) PermissionAction {
	checker := NewPermissionChecker(DefaultRuleset)
	return checker.Check(toolID, path)
}

// IsAllowed checks if the action is permitted
func IsAllowed(action PermissionAction) bool {
	return action == PermissionAllow
}

// IsDenied checks if the action is denied
func IsDenied(action PermissionAction) bool {
	return action == PermissionDeny
}

// NeedsApproval checks if the action needs user approval
func NeedsApproval(action PermissionAction) bool {
	return action == PermissionAsk
}