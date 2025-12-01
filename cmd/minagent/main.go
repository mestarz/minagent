package main

import (
	"fmt"
	"io"
	"os"

	"minagent/internal/agent"
	"minagent/internal/config"

	"github.com/spf13/cobra"
)

var (
	taskID         string
	taskType       string
	outputFile     string
	configPath     string
)

var rootCmd = &cobra.Command{
	Use:   "minagent [prompt]",
	Short: "MinAgent is a task-oriented AI agent CLI.",
	Long:  `MinAgent is a lightweight, local-first, task-oriented AI agent. 
It intelligently infers the task type from your prompt and uses tools to accomplish the goal.`, 
	Args: cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		prompt := args[0]
		
		cfg, err := config.Load(configPath)
		if err != nil {
			fmt.Printf("Error loading config: %v\n", err)
			os.Exit(1)
		}

		orchestrator, err := agent.NewOrchestrator(cfg)
		if err != nil {
			fmt.Printf("Error initializing agent: %v\n", err)
			os.Exit(1)
		}

		finalResponse, err := orchestrator.Run(cmd.Context(), prompt, taskID, taskType)
		if err != nil {
			fmt.Printf("Agent execution failed: %v\n", err)
			os.Exit(1)
		}

		fmt.Println("\n--- Agent Response ---")
		fmt.Println(finalResponse)
	},
}

var completionCmd = &cobra.Command{
	Use:   "completion [bash|zsh|fish|powershell]",
	Short: "Generate completion script for your shell",
	Long: `To load completions:

Bash:
  $ source <(minagent completion bash)
  # To load completions for every new session, execute once:
  # Linux:
  $ minagent completion bash > /etc/bash_completion.d/minagent
  # macOS:
  $ minagent completion bash > /usr/local/etc/bash_completion.d/minagent

Zsh:
  # If shell completion is not already enabled in your environment,
  # you will need to enable it.  You can execute the following once:
  $ echo "autoload -U compinit; compinit" >> ~/.zshrc
  $ source <(minagent completion zsh)
  # To load completions for every new session, execute once:
  $ minagent completion zsh > "${fpath[1]}/_minagent"
  $ exec zsh

Fish:
  $ minagent completion fish | source
  # To load completions for every new session, execute once:
  $ minagent completion fish > ~/.config/fish/completions/minagent.fish
`,
	DisableFlagsInUseLine: true,
	ValidArgs:             []string{"bash", "zsh", "fish", "powershell"},
	Args:                  cobra.ExactValidArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		switch args[0] {
		case "bash":
			cmd.Root().GenBashCompletion(os.Stdout)
		case "zsh":
			cmd.Root().GenZshCompletion(os.Stdout)
		case "fish":
			cmd.Root().GenFishCompletion(os.Stdout, true)
		case "powershell":
			cmd.Root().GenPowerShellCompletionWithDesc(os.Stdout)
		}
	},
}

// For silent completion output
func sliently(cmd *cobra.Command, args []string) {
	cmd.SetOut(io.Discard)
	cmd.SetErr(io.Discard)
	cmd.Root().GenZshCompletion(io.Discard)
}

func init() {
	// Add the completion command
	rootCmd.AddCommand(completionCmd)

	// Global flags
	rootCmd.PersistentFlags().StringVar(&configPath, "config", "", "Path to config file (default is $HOME/.minagent/config.yaml or ./config.yaml)")

	// Flags for the root command
	rootCmd.Flags().StringVar(&taskID, "task", "", "ID of the task to load or save context for")
	rootCmd.Flags().StringVar(&taskType, "type", "", "Explicitly specify the task type (e.g., 'code-generation')")
	rootCmd.Flags().StringVar(&outputFile, "output", "", "Path to save the output to, instead of printing to stdout")
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}