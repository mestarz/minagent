package providers

import (
	"context"

	"minagent/internal/mcp"
	"minagent/internal/tools/types" // Import types.Tool
)

// ModelProvider defines the contract for any module that can call a large language model.
// Its responsibility is to communicate with a specific model backend and return a standardized response.
type ModelProvider interface {
	// GetName returns the unique name of the provider (e.g., "generic-http").
	GetName() string

	// GenerateContent sends the conversation history and available tools to the model
	// and returns a channel of streaming response parts.
	GenerateContent(ctx context.Context, history []*mcp.Message, tools []types.Tool) (chan *mcp.ResponsePart, error)
}
