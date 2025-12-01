package mcp

// Message represents a single message in the conversation history.
type Message struct {
	Role  string `json:"role"` // "user", "model", or "tool"
	Parts []Part `json:"parts"`
}

// Part is a segment of a message, which can be text or a tool call/result.
type Part struct {
	Text      *string    `json:"text,omitempty"`
	ToolCall  *ToolCall  `json:"tool_call,omitempty"`
	ToolResult *ToolResult `json:"tool_result,omitempty"`
}

// ToolCall represents a request from the model to call a specific tool.
type ToolCall struct {
	Name string            `json:"name"`
	Args map[string]interface{} `json:"args"`
}

// ToolResult represents the output of a tool execution.
type ToolResult struct {
	Name   string `json:"name"`
	Output string `json:"output"`
}

// Response is the standardized response structure from a ModelProvider.
type Response struct {
	// For now, we assume the response contains a single message from the model.
	// This can be extended to support multiple choices or candidates later.
	Candidate Message `json:"candidate"`
}
