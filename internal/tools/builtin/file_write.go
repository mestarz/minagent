package builtin

import (
	"encoding/json"
	"fmt"
	"os"

	"minagent/internal/tools/types"
)

// FileWriteTool is the implementation for the built-in file_write tool.
type FileWriteTool struct{}

func (t *FileWriteTool) Name() string {
	return "file_write"
}

func (t *FileWriteTool) Schema() (types.Schema, error) {
	schema := `{
		"name": "file_write",
		"description": "Writes content to a specified file, creating it if it doesn't exist, or overwriting it if it does.",
		"parameters": {
			"type": "object",
			"properties": {
				"path": {
					"type": "string",
					"description": "The path to the file to write to."
				},
				"content": {
					"type": "string",
					"description": "The content to write into the file."
				}
			},
			"required": ["path", "content"]
		}
	}`
	return types.Schema(schema), nil
}

func (t *FileWriteTool) Execute(arguments json.RawMessage) (string, error) {
	var args struct {
		Path    string `json:"path"`
		Content string `json:"content"`
	}
	if err := json.Unmarshal(arguments, &args); err != nil {
		return "", fmt.Errorf("invalid arguments for file_write: %w", err)
	}

	if args.Path == "" {
		return "", fmt.Errorf("path argument is required for file_write")
	}

	err := os.WriteFile(args.Path, []byte(args.Content), 0644)
	if err != nil {
		return "", fmt.Errorf("failed to write to file %s: %w", args.Path, err)
	}

	return fmt.Sprintf("Successfully wrote %d bytes to %s", len(args.Content), args.Path), nil
}
