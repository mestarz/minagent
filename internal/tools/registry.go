package tools

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"minagent/internal/tools/builtin"
	"minagent/internal/tools/types"
)

// Registry holds all the available tools that the agent can use.
type Registry struct {

tools map[string]types.Tool
}

// NewRegistry creates and initializes a tool registry.
// It first registers all built-in tools, then discovers external tools
// from the provided plugin directory.
func NewRegistry(pluginDir string) (*Registry, error) {
	r := &Registry{
		tools: make(map[string]types.Tool),
	}

	// 1. Register all built-in tools
	r.registerBuiltInTools()

	// 2. Scan for external executable tools
	if pluginDir != "" {
		if err := r.discoverExternalTools(pluginDir); err != nil {
			// We can treat this as a non-fatal error and just log it
			fmt.Fprintf(os.Stderr, "Warning: failed to discover external tools in %s: %v\n", pluginDir, err)
		}
	}

	return r, nil
}

// registerBuiltInTools initializes and registers all native Go tools.
func (r *Registry) registerBuiltInTools() {
	builtInTools := []types.Tool{
		&builtin.FileReadTool{},
		&builtin.FileWriteTool{},
		&builtin.ListDirectoryTool{},
	}
	for _, tool := range builtInTools {
		r.Register(tool)
	}
}

// discoverExternalTools scans a directory for executable files and registers them as tools.
func (r *Registry) discoverExternalTools(pluginDir string) error {
	// Expand tilde (~) in path
	if pluginDir[0] == '~' {
		home, err := os.UserHomeDir()
		if err != nil {
			return fmt.Errorf("could not get user home directory: %w", err)
		}
		pluginDir = filepath.Join(home, pluginDir[1:])
	}

	err := filepath.WalkDir(pluginDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		// Check if the file is executable
		info, err := os.Stat(path)
		if err != nil {
			return err
		}
		if info.Mode()&0111 != 0 { // Check for execute permission
			toolName := filepath.Base(path)
		
			// Avoid overriding a built-in tool with an external one
			if _, exists := r.tools[toolName]; exists {
				fmt.Fprintf(os.Stderr, "Warning: external tool %s has the same name as a built-in tool and will be ignored.\n", toolName)
				return nil
			}

			tool, err := NewExecutableTool(path, toolName)
			if err != nil {
				fmt.Fprintf(os.Stderr, "Warning: could not load external tool %s: %v\n", toolName, err)
				return nil
			}
			r.Register(tool)
		}
		return nil
	})

	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("error walking plugin directory %s: %w", pluginDir, err)
	}

	return nil
}


// Register adds a tool to the registry.
func (r *Registry) Register(tool types.Tool) {
	fmt.Printf("Registering tool: %s\n", tool.Name())
	r.tools[tool.Name()] = tool
}

// GetTool retrieves a tool by its name.
func (r *Registry) GetTool(name string) (types.Tool, bool) {
	tool, exists := r.tools[name]
	return tool, exists
}

// GetAllTools returns all registered tools.
func (r *Registry) GetAllTools() []types.Tool {
	allTools := make([]types.Tool, 0, len(r.tools))
	for _, tool := range r.tools {
		allTools = append(allTools, tool)
	}
	return allTools
}
