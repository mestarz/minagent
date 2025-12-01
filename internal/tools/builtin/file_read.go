package builtin

import (
	"encoding/json"
	"fmt"
	"os"

	"minagent/internal/tools/types"
)

// FileReadTool is the implementation for the built-in file_read tool.
type FileReadTool struct{}

func (t *FileReadTool) Name() string {
	return "file_read"
}

func (t *FileReadTool) Schema() (types.Schema, error) {
	schema := `{
		"name": "file_read",
		"description": "Reads the entire content of a specified file.",
		"parameters": {
			"type": "object",
			"properties": {
				"path": {
					"type": "string",
					"description": "The path to the file to read."
				}
			},
			"required": ["path"]
		}
	}`
	return types.Schema(schema), nil
}

func (t *FileReadTool) Execute(arguments json.RawMessage) (string, error) {
	var args struct {
		Path string `json:"path"`
	}
	if err := json.Unmarshal(arguments, &args); err != nil {
		return "", fmt.Errorf("invalid arguments for file_read: %w", err)
	}

	if args.Path == "" {
		return "", fmt.Errorf("path argument is required for file_read")
	}

	content, err := os.ReadFile(args.Path)
	if err != nil {
		return "", fmt.Errorf("failed to read file %s: %w", args.Path, err)
	}

	return string(content), nil
}
