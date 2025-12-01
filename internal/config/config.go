package config

import (
	"fmt"
	"os"
	"path/filepath"

	"minagent/internal/policy" // Added import
	"gopkg.in/yaml.v3"
)

// Config is the main configuration structure for the agent.
type Config struct {
	LLM   LLMConfig   `mapstructure:"llm"`
	Tools struct {
		PluginDir string `yaml:"plugin_dir"`
	} `yaml:"tools"`
	Policy PolicyConfig `yaml:"policy"` // New Policy configuration
}

// PolicyConfig holds the policy rules for the agent's Policy Engine.
type PolicyConfig struct {
	Rules []policy.Rule `yaml:"rules"`
}
	Tasks []Task      `mapstructure:"tasks"`
}

// LLMConfig holds configuration for the model provider.
type LLMConfig struct {
	Provider  string            `mapstructure:"provider"`
	ModelName string            `mapstructure:"model_name"` // Added ModelName
	Endpoint  string            `mapstructure:"endpoint"`
	APIKey    string            `mapstructure:"api_key"`
	Headers   map[string]string `mapstructure:"headers"`
}

// ToolsConfig holds configuration for the tool system.
type ToolsConfig struct {
	PluginDir string `mapstructure:"plugin_dir"`
}

// Task defines a configurable task type.
type Task struct {
	Name        string `mapstructure:"name"`
	Description string `mapstructure:"description"`
	SystemPrompt string `mapstructure:"system_prompt"`
}

// Load loads the configuration from a file.
func Load(configPath string) (*Config, error) {
	if configPath != "" {
		viper.SetConfigFile(configPath)
	} else {
		// Default search paths
		viper.AddConfigPath("$HOME/.minagent")
		viper.AddConfigPath(".")
		viper.SetConfigName("config")
		viper.SetConfigType("yaml")
	}

	// Read environment variables
	viper.SetEnvPrefix("MINAGENT")
	viper.AutomaticEnv()
    // Replace dots with underscores in env var names (e.g., llm.api_key -> MINAGENT_LLM_API_KEY)
    viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	if err := viper.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); ok {
			return nil, fmt.Errorf("config file not found")
		}
		return nil, fmt.Errorf("error reading config file: %w", err)
	}

	var cfg Config
	if err := viper.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("error unmarshaling config: %w", err)
	}

    // Expand environment variables in string fields
    cfg.LLM.APIKey = os.ExpandEnv(cfg.LLM.APIKey)
    for k, v := range cfg.LLM.Headers {
        cfg.LLM.Headers[k] = os.ExpandEnv(v)
    }

	return &cfg, nil
}
