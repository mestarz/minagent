package tools

import (
	"bytes"
	"encoding/json"
	"fmt"
	"os/exec"

	"minagent/internal/tools/types"
)

// ExecutableTool represents a tool that is an executable file on the filesystem.
type ExecutableTool struct {
	name       string
	path       string
	schema     types.Schema
}

// NewExecutableTool creates a new tool instance from an executable file path.
// It fetches the schema by running the executable with the "--schema" flag.
func NewExecutableTool(path string, name string) (*ExecutableTool, error) {
	tool := &ExecutableTool{
		name: name,
		path: path,
	}

	if err := tool.fetchSchema(); err != nil {
		return nil, fmt.Errorf("failed to fetch schema for tool '%s': %w", name, err)
	}
	
	return tool, nil
}

func (t *ExecutableTool) Name() string {
	return t.name
}

func (t *ExecutableTool) Schema() (types.Schema, error) {
	return t.schema, nil
}

// Execute runs the executable file, passing the arguments via stdin.
func (t *ExecutableTool) Execute(arguments json.RawMessage) (string, error) {
	cmd := exec.Command(t.path)

	// Pipe arguments to stdin
	cmd.Stdin = bytes.NewReader(arguments)

	var out bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr

	err := cmd.Run()
	if err != nil {
		return "", fmt.Errorf("tool '%s' execution failed: %s", t.name, stderr.String())
	}

	return out.String(), nil
}

// fetchSchema runs the executable with "--schema" to get its definition.
func (t *ExecutableTool) fetchSchema() error {
	cmd := exec.Command(t.path, "--schema")

	var out bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr
	
	err := cmd.Run()
	if err != nil {
		// We can log this as a warning instead of a fatal error to allow for simple executables that don't have schemas.
		return fmt.Errorf("could not get schema, command failed: %s", stderr.String())
	}

	// The output must be a valid JSON raw message.
	t.schema = types.Schema(out.Bytes())
	return nil
}