package providers

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"strings"

	"minagent/internal/mcp"
	"minagent/internal/tools/types"
)

// --- Mock Provider and its Handlers ---

type MockProvider struct {
	handlerChain []mockHandler
}

func NewMockProvider() *MockProvider {
	return &MockProvider{
		handlerChain: []mockHandler{
			&planningHandler{}, // New handler for the planning phase
			&fileToolCallHandler{},
			&inferenceHandler{},
			&toolResultHandler{},
			&genericHandler{},
		},
	}
}

func (p *MockProvider) GetName() string {
	return "mock"
}

func (p *MockProvider) GenerateContent(ctx context.Context, history []*mcp.Message, tools []types.Tool) (*mcp.Response, error) {
	log.Println("MockProvider: Generating content locally.")

	// Construct a mock request body for the handlers to process.
	mockMessages := make([]map[string]interface{}, len(history))
	for i, msg := range history {
		var contentBuilder strings.Builder
		for _, part := range msg.Parts {
			if part.Text != nil {
				contentBuilder.WriteString(*part.Text)
			} else if part.ToolResult != nil {
				contentBuilder.WriteString(fmt.Sprintf("Tool %s output: %s", part.ToolResult.Name, part.ToolResult.Output))
			}
		}
		mockMessages[i] = map[string]interface{}{"role": msg.Role, "content": contentBuilder.String()}
	}
	
	mockTools := make([]map[string]interface{}, 0, len(tools))
	for _, tool := range tools {
		mockTools = append(mockTools, map[string]interface{}{"name": tool.Name()})
	}
	
	reqBody := mockRequestBody{
		Model:    "local-mock-model",
		Messages: mockMessages,
		Tools:    mockTools,
	}

	for _, handler := range p.handlerChain {
		if responseMap, handled := handler.Handle(reqBody); handled {
			return mapToMCPResponse(responseMap)
		}
	}
	return nil, fmt.Errorf("no mock handler could process the request")
}

// --- Mock Handler Infrastructure ---

type mockRequestBody struct {
	Model    string                   `json:"model"`
	Messages []map[string]interface{} `json:"messages"`
	Tools    []map[string]interface{} `json:"tools"`
}

type mockHandler interface {
	Handle(reqBody mockRequestBody) (response map[string]interface{}, handled bool)
}

// planningHandler generates a mock plan.
type planningHandler struct{}
func (h *planningHandler) Handle(reqBody mockRequestBody) (map[string]interface{}, bool) {
	lastPrompt := getLastPrompt(reqBody)
	if strings.Contains(lastPrompt, "Generate the JSON plan now.") {
		log.Println("MockProvider/PlanningHandler: Matched. Responding with a mock plan.")
		// A mock plan to write a file and then read it.
		mockPlanJSON := `{
			"plan": [
				{
					"step": 1,
					"thought": "First, I will create a file with the requested content.",
					"action": "tool_call",
					"tool_name": "file_write",
					"args": {
						"path": "plan_test.txt",
						"content": "Hello from a planned execution!"
					}
				},
				{
					"step": 2,
					"thought": "Next, I will read the file back to verify its content.",
					"action": "tool_call",
					"tool_name": "file_read",
					"args": {
						"path": "plan_test.txt"
					}
				}
			]
		}`
		response := map[string]interface{}{"choices": []map[string]interface{}{{"message": map[string]interface{}{"role": "assistant", "content": mockPlanJSON}}}}
		return response, true
	}
	return nil, false
}


// ... (Other handlers remain the same) ...
type fileToolCallHandler struct{}
func (h *fileToolCallHandler) Handle(reqBody mockRequestBody) (map[string]interface{}, bool) { return nil, false } // Simplified for this refactoring
type inferenceHandler struct{}
func (h *inferenceHandler) Handle(reqBody mockRequestBody) (map[string]interface{}, bool) { return nil, false } // Not used in PEV loop
type toolResultHandler struct{}
func (h *toolResultHandler) Handle(reqBody mockRequestBody) (map[string]interface{}, bool) { return nil, false } // Execution is now handled by orchestrator
type genericHandler struct{}
func (h *genericHandler) Handle(reqBody mockRequestBody) (map[string]interface{}, bool) {
	log.Println("MockProvider/GenericHandler: Matched. This should not happen in a planned execution.")
	response := map[string]interface{}{"choices": []map[string]interface{}{{"message": map[string]interface{}{"role": "assistant", "content": "Generic mock response."}}}}
	return response, true
}


// ... (Helper functions remain the same) ...
func getLastPrompt(reqBody mockRequestBody) string {
	if len(reqBody.Messages) > 0 {
		if content, ok := reqBody.Messages[len(reqBody.Messages)-1]["content"].(string); ok {
			return content
		}
	}
	return ""
}
func mapToMCPResponse(responseMap map[string]interface{}) (*mcp.Response, error) {
	jsonData, err := json.Marshal(responseMap)
	if err != nil { return nil, fmt.Errorf("failed to marshal mock response map: %w", err) }
	var intermediateResponse struct {
		Choices []struct {
			Message struct {
				Role      string  `json:"role"`
				Content   *string `json:"content"`
				ToolCalls []struct {
					ID       string `json:"id"`
					Type     string `json:"type"`
					Function struct {
						Name      string `json:"name"`
						Arguments string `json:"arguments"`
					} `json:"function"`
				} `json:"tool_calls"`
			} `json:"message"`
		} `json:"choices"`
	}
	if err := json.Unmarshal(jsonData, &intermediateResponse); err != nil { return nil, fmt.Errorf("failed to unmarshal mock response json: %w", err) }
	if len(intermediateResponse.Choices) == 0 { return nil, fmt.Errorf("mock response contains no choices") }
	choiceMsg := intermediateResponse.Choices[0].Message
	mcpCandidate := mcp.Message{Role: choiceMsg.Role, Parts: []mcp.Part{}}
	if choiceMsg.Content != nil && *choiceMsg.Content != "" {
		mcpCandidate.Parts = append(mcpCandidate.Parts, mcp.Part{Text: choiceMsg.Content})
	}
	if len(choiceMsg.ToolCalls) > 0 {
		for _, toolCall := range choiceMsg.ToolCalls {
			var args map[string]interface{}
			if err := json.Unmarshal([]byte(toolCall.Function.Arguments), &args); err != nil {
				return nil, fmt.Errorf("failed to unmarshal mock tool call arguments: %w", err)
			}
			mcpCandidate.Parts = append(mcpCandidate.Parts, mcp.Part{ToolCall: &mcp.ToolCall{Name: toolCall.Function.Name, Args: args}})
		}
	}
	return &mcp.Response{Candidate: mcpCandidate}, nil
}
