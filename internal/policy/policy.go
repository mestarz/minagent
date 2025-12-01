package policy

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"minagent/internal/mcp"
)

// DecisionType represents the outcome of a policy evaluation.
type DecisionType string

const (
	DecisionAllow    DecisionType = "allow"
	DecisionDeny     DecisionType = "deny"
	DecisionAskUser  DecisionType = "ask" // Future enhancement
	DecisionContinue DecisionType = "continue" // Special: process next rule
)

// Rule defines a single policy rule.
type Rule struct {
	Decision   DecisionType           `json:"decision"`
	Tool       string                 `json:"tool"`        // Tool name or "*" for all tools
	ToolPattern string                `json:"tool_pattern"` // Regex pattern for tool name
	ArgsMatch  map[string]interface{} `json:"args_match"`  // Match specific arguments
	Reason     string                 `json:"reason,omitempty"`
}

// PolicyEngine defines the interface for evaluating tool calls against policies.
type PolicyEngine interface {
	Evaluate(toolCall *mcp.ToolCall) PolicyDecision
}

// PolicyDecision represents the result of a policy evaluation.
type PolicyDecision struct {
	Type   DecisionType
	Reason string
}

// ConfigPolicyEngine implements PolicyEngine based on a list of configurable rules.
type ConfigPolicyEngine struct {

rules []Rule
}

// NewConfigPolicyEngine creates a new ConfigPolicyEngine with the given rules.
func NewConfigPolicyEngine(rules []Rule) *ConfigPolicyEngine {
	// Pre-compile regex patterns for efficiency
	for i := range rules {
		if rules[i].ToolPattern != "" {
			// For future: Store compiled regex
		}
	}
	return &ConfigPolicyEngine{rules: rules}
}

// Evaluate evaluates a tool call against the configured rules.
// Rules are evaluated in order. The first matching rule determines the decision.
func (e *ConfigPolicyEngine) Evaluate(toolCall *mcp.ToolCall) PolicyDecision {
	for _, rule := range e.rules {
		if e.matchTool(rule, toolCall) && e.matchArgs(rule, toolCall) {
			if rule.Decision == DecisionContinue {
				// Special decision to continue to the next rule
				continue
			}
			return PolicyDecision{Type: rule.Decision, Reason: rule.Reason}
		}
	}
	// Default to deny if no rule matches (or implement a default allow rule as the last rule)
	return PolicyDecision{Type: DecisionDeny, Reason: "No policy rule matched, defaulting to deny."}
}

func (e *ConfigPolicyEngine) matchTool(rule Rule, toolCall *mcp.ToolCall) bool {
	if rule.Tool == "*" {
		return true
	}
	if rule.Tool != "" && rule.Tool == toolCall.Name {
		return true
	}
	if rule.ToolPattern != "" {
		match, err := regexp.MatchString(rule.ToolPattern, toolCall.Name)
		if err != nil {
			fmt.Fprintf(os.Stderr, "Warning: invalid regex pattern in policy rule '%s': %v\n", rule.ToolPattern, err)
			return false
		}
		return match
	}
	return false
}

func (e *ConfigPolicyEngine) matchArgs(rule Rule, toolCall *mcp.ToolCall) bool {
	if len(rule.ArgsMatch) == 0 {
		return true // No arg matching specified for the rule
	}
	
	// Convert toolCall.Args to JSON string for comparison or iterate over fields
	// For simplicity, let's assume direct key-value matching for now.
	for key, expectedValue := range rule.ArgsMatch {
		if actualValue, ok := toolCall.Args[key]; !ok || fmt.Sprintf("%v", actualValue) != fmt.Sprintf("%v", expectedValue) {
			return false // Argument mismatch
		}
	}
	return true
}
