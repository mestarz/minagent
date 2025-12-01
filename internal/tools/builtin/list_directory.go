package builtin

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"minagent/internal/tools/types"
)

// ListDirectoryTool is the implementation for the built-in list_directory tool.
type ListDirectoryTool struct{}

func (t *ListDirectoryTool) Name() string {
	return "list_directory"
}

func (t *ListDirectoryTool) Schema() (types.Schema, error) {
	schema := `{
		"name": "list_directory",
		"description": "Lists the contents (files and subdirectories) of a specified directory.",
		"parameters": {
			"type": "object",
			"properties": {
				"path": {
					"type": "string",
					"description": "The path to the directory to list. Defaults to the current directory '.'."
				},
				"recursive": {
					"type": "boolean",
					"description": "Whether to list contents recursively. Defaults to false.",
					"default": false
				}
			},
			"required": ["path"]
		}
	}`
	return types.Schema(schema), nil
}

func (t *ListDirectoryTool) Execute(arguments json.RawMessage) (string, error) {
	var args struct {
		Path      string `json:"path"`
		Recursive bool   `json:"recursive"`
	}
	// Set default path
	args.Path = "."
	if err := json.Unmarshal(arguments, &args); err != nil {
		return "", fmt.Errorf("invalid arguments for list_directory: %w", err)
	}

	var builder strings.Builder
	builder.WriteString(fmt.Sprintf("Contents of directory '%s':\n", args.Path))

	if args.Recursive {
		err := filepath.WalkDir(args.Path, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			// Get path relative to the starting path for cleaner output
			relPath, err := filepath.Rel(args.Path, path)
			if err != nil {
				return err
			}
			if relPath == "." { // Skip the root itself
				return nil
			}
			
			entry := relPath
			if d.IsDir() {
				entry += "/"
			}
			builder.WriteString(entry)
			builder.WriteString("\n")
			return nil
		})
		if err != nil {
			return "", fmt.Errorf("failed to walk directory %s: %w", args.Path, err)
		}
	} else {
		entries, err := os.ReadDir(args.Path)
		if err != nil {
			return "", fmt.Errorf("failed to read directory %s: %w", args.Path, err)
		}
		for _, e := range entries {
			entry := e.Name()
			if e.IsDir() {
				entry += "/"
			}
			builder.WriteString(entry)
			builder.WriteString("\n")
		}
	}

	return builder.String(), nil
}
