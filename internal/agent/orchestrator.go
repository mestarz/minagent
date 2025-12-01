package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strings"

	"minagent/internal/config"
	"minagent/internal/mcp"
	"minagent/internal/providers"
	"minagent/internal/task"
	"minagent/internal/tools"
	"minagent/internal/tools/types"
)

// Orchestrator is the core of the agent, driving the interaction loop.
type Orchestrator struct {
	cfg          *config.Config
	provider     providers.ModelProvider
	toolRegistry *tools.Registry
	taskManager  *task.Manager
}

// NewOrchestrator creates a new orchestrator.
func NewOrchestrator(cfg *config.Config) (*Orchestrator, error) {
	var provider providers.ModelProvider
	if cfg.LLM.Provider == "generic-http" {
		provider = providers.NewGenericHTTPProvider(cfg.LLM)
	} else if cfg.LLM.Provider == "mock" {
		provider = providers.NewMockProvider()
	} else {
		return nil, fmt.Errorf("unknown model provider: %s", cfg.LLM.Provider)
	}

	toolRegistry, err := tools.NewRegistry(cfg.Tools.PluginDir)
	if err != nil {
		return nil, fmt.Errorf("failed to initialize tool registry: %w", err)
	}

	taskManager, err := task.NewManager("~/.minagent/tasks")
	if err != nil {
		return nil, fmt.Errorf("failed to initialize task manager: %w", err)
	}

	return &Orchestrator{
		cfg:          cfg,
		provider:     provider,
		toolRegistry: toolRegistry,
		taskManager:  taskManager,
	}, nil
}

// Run executes the full Plan-Execute-Verify loop.
func (o *Orchestrator) Run(ctx context.Context, prompt, taskID, specifiedTaskType string) (string, error) {
	expandedPrompt, err := o.processFileMentions(prompt)
	if err != nil {
		return "", err
	}

	taskCtx, err := o.taskManager.Load(taskID)
	if err != nil {
		return "", fmt.Errorf("could not load task context for ID '%s': %w", taskID, err)
	}
	
	if len(taskCtx.History) == 0 {
		userMessage := &mcp.Message{Role: "user", Parts: []mcp.Part{{Text: &expandedPrompt}}}
		taskCtx.History = append(taskCtx.History, userMessage)
	}
	
	if taskCtx.Plan == nil {
		fmt.Println("--- Phase: PLANNING ---")
		plan, err := o.plan(ctx, expandedPrompt)
		if err != nil {
			return "", fmt.Errorf("planning phase failed: %w", err)
		}
		taskCtx.Plan = plan
		if err := o.taskManager.Save(taskCtx); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: failed to save plan: %v\n", err)
		}
	}

	fmt.Println("--- Phase: EXECUTING ---")
	if err := o.execute(ctx, taskCtx); err != nil {
		return "", fmt.Errorf("execution phase failed: %w", err)
	}
	if err := o.taskManager.Save(taskCtx); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: failed to save execution state: %v\n", err)
	}

	fmt.Println("--- Phase: VERIFYING ---")
	finalResult, err := o.verify(ctx, taskCtx, expandedPrompt)
	if err != nil {
		return "", fmt.Errorf("verification phase failed: %w", err)
	}

	if err := o.taskManager.Save(taskCtx); err != nil {
		return "", fmt.Errorf("warning: failed to save final task context: %w", err)
	}

	return finalResult, nil
}

// plan generates a step-by-step plan to achieve the user's goal.
func (o *Orchestrator) plan(ctx context.Context, userGoal string) (*task.Plan, error) {
	toolsList := o.toolRegistry.GetAllTools()
	var toolsSchemaBuilder strings.Builder
	for _, t := range toolsList {
		schema, _ := t.Schema()
		toolsSchemaBuilder.WriteString(string(schema) + "\n")
	}

	promptTemplate := `You are an expert project planner. Your job is to break down a user's goal into a series of concrete, executable steps.
You must respond with only a JSON object containing a "plan" array.
Each step in the plan must have a "step" number, a "thought" explaining your reasoning, an "action" (e.g., "tool_call"), a "tool_name" if applicable, and the "args" for the tool.

Available tools:
%s

User's Goal: "%s"

Generate the JSON plan now.`
	planningPrompt := fmt.Sprintf(promptTemplate, toolsSchemaBuilder.String(), userGoal)
	
	history := []*mcp.Message{{Role: "user", Parts: []mcp.Part{{Text: &planningPrompt}}}} 
	
	resp, err := o.provider.GenerateContent(ctx, history, nil)
	if err != nil { return nil, err }

	if len(resp.Candidate.Parts) == 0 || resp.Candidate.Parts[0].Text == nil {
		return nil, fmt.Errorf("planner returned an empty response")
	}
	
	planJSON := *resp.Candidate.Parts[0].Text
	planJSON = strings.TrimSpace(strings.Trim(strings.TrimSpace(planJSON), "```json"), "```")
	
	var plan task.Plan
	if err := json.Unmarshal([]byte(planJSON), &plan); err != nil {
		return nil, fmt.Errorf("failed to parse plan JSON. Raw response: %s. Error: %w", planJSON, err)
	}

	return &plan, nil
}

// execute iterates through the plan and executes each step.
func (o *Orchestrator) execute(ctx context.Context, taskCtx *task.Context) error {
	for i := taskCtx.CurrentStep; i < len(taskCtx.Plan.Steps); i++ {
		step := taskCtx.Plan.Steps[i]
		taskCtx.CurrentStep = i
		fmt.Printf("Executing Step %d: %s\n", step.Step, step.Thought)

		if step.Action == "tool_call" {
			toolToRun, exists := o.toolRegistry.GetTool(step.ToolName)
			if !exists {
				return fmt.Errorf("step %d: planned to use unknown tool '%s'", step.Step, step.ToolName)
			}
			
			argsJSON, err := json.Marshal(step.Args)
			if err != nil {
				return fmt.Errorf("step %d: failed to marshal args for tool '%s': %w", step.Step, step.ToolName, err)
			}
			
			toolOutput, err := toolToRun.Execute(argsJSON)
			if err != nil {
				return fmt.Errorf("step %d: tool '%s' execution failed: %w", step.Step, step.ToolName, err)
			}

			resultMsg := fmt.Sprintf("Step %d completed. Tool '%s' returned: %s", step.Step, step.ToolName, toolOutput)
			taskCtx.StepResults = append(taskCtx.StepResults, resultMsg)
			
			taskCtx.History = append(taskCtx.History, &mcp.Message{
				Role: "user",
				Parts: []mcp.Part{{Text: &resultMsg}},
			})
		}
	}
	taskCtx.CurrentStep = len(taskCtx.Plan.Steps)
	return nil
}

// verify checks if the final state meets the user's goal.
func (o *Orchestrator) verify(ctx context.Context, taskCtx *task.Context, userGoal string) (string, error) {
	fmt.Println("Verification: All planned steps executed.")
	var resultBuilder strings.Builder
	resultBuilder.WriteString("--- Execution Summary ---\n")
	resultBuilder.WriteString(fmt.Sprintf("User Goal: %s\n", userGoal))
	resultBuilder.WriteString("Plan Executed:\n")
	for _, step := range taskCtx.Plan.Steps {
		resultBuilder.WriteString(fmt.Sprintf("  - Step %d: %s\n", step.Step, step.Thought))
	}
	resultBuilder.WriteString("\n--- Step Results ---\n")
	for _, res := range taskCtx.StepResults {
		resultBuilder.WriteString(res + "\n")
	}
	resultBuilder.WriteString("\n--- Task Complete ---")
	return resultBuilder.String(), nil
}

// processFileMentions and other helpers
func (o *Orchestrator) processFileMentions(prompt string) (string, error) {
	re := regexp.MustCompile(`@([\w./-]+)`)
	matches := re.FindAllStringSubmatch(prompt, -1)
	if len(matches) == 0 { return prompt, nil }
	var contentBuilder strings.Builder
	contentBuilder.WriteString("The user has provided content from the following files:\n\n")
	fileContents := make(map[string]string)
	for _, match := range matches {
		path := match[1]
		if _, ok := fileContents[path]; ok { continue }
		content, err := os.ReadFile(path)
		if err != nil { return "", fmt.Errorf("failed to read file @%s: %w", path, err) }
		fileContents[path] = string(content)
	}
	for path, content := range fileContents {
		contentBuilder.WriteString(fmt.Sprintf("--- Start of file: %s ---\n%s\n--- End of file: %s ---\n\n", path, content, path))
	}
	finalPrompt := re.ReplaceAllString(prompt, "")
	contentBuilder.WriteString("Based on the file(s) above, please respond to the following request:\n\n" + finalPrompt)
	return contentBuilder.String(), nil
}
