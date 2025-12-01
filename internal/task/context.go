package task

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"minagent/internal/mcp"
)

// --- Task Context ---

// Context holds all the information for a single, stateful task.
type Context struct {
	ID           string                 `json:"id"`
	History      []*mcp.Message         `json:"history"`
	Metadata     map[string]interface{} `json:"metadata"`
	CreatedAt    time.Time              `json:"created_at"`
	LastModified time.Time              `json:"last_modified_at"`
}

// NewContext creates a new, empty task context.
func NewContext(id string) *Context {
	return &Context{
		ID:          id,
		History:     make([]*mcp.Message, 0),
		Metadata:    make(map[string]interface{}),
		CreatedAt:   time.Now(),
		LastModified:time.Now(),
	}
}

// Manager handles the loading and saving of task contexts to the filesystem.
type Manager struct {
	basePath string
}

// NewManager creates a new context manager.
func NewManager(basePath string) (*Manager, error) {
	if basePath[0] == '~' {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, fmt.Errorf("could not get user home directory: %w", err)
		}
		basePath = filepath.Join(home, basePath[1:])
	}

	if err := os.MkdirAll(basePath, 0755); err != nil {
		return nil, fmt.Errorf("could not create task store directory at %s: %w", basePath, err)
	}

	return &Manager{basePath: basePath}, nil
}

func (m *Manager) getTaskPath(id string) string {
	return filepath.Join(m.basePath, fmt.Sprintf("%s.json", id))
}

// Load loads a task context from a file. If the file doesn't exist, it creates a new context.
func (m *Manager) Load(id string) (*Context, error) {
	if id == "" {
		return NewContext("temp"), nil
	}

	path := m.getTaskPath(id)
	file, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return NewContext(id), nil
		}
		return nil, fmt.Errorf("failed to read task file %s: %w", path, err)
	}

	var ctx Context
	if err := json.Unmarshal(file, &ctx); err != nil {
		return nil, fmt.Errorf("failed to unmarshal task context from %s: %w", path, err)
	}

	return &ctx, nil
}

// Save saves a task context to a file.
func (m *Manager) Save(ctx *Context) error {
	if ctx.ID == "" || ctx.ID == "temp" {
		return nil
	}

	ctx.LastModified = time.Now()
	path := m.getTaskPath(ctx.ID)

	data, err := json.MarshalIndent(ctx, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal task context: %w", err)
	}

	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write task file to %s: %w", path, err)
	}

	return nil
}
