package types

import (
	"encoding/json"
)

// Schema represents the MCP-compatible schema of a tool.
// Using json.RawMessage allows for flexibility without deep struct dependencies.
type Schema json.RawMessage

// Tool defines the contract for any executable tool.
type Tool interface {
	// Name returns the unique name of the tool.
	Name() string
	// Schema returns the tool's MCP-compatible JSON schema.
	Schema() (Schema, error)
	// Execute runs the tool with the given arguments.
	// The arguments are a raw JSON message from the model.
	Execute(arguments json.RawMessage) (output string, err error)
}
