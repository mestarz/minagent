package providers

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"

	"minagent/internal/config"
	"minagent/internal/mcp"
	"minagent/internal/tools/types"
)

// GenericHTTPProvider communicates with any MCP-compatible HTTP endpoint.
type GenericHTTPProvider struct {
	cfg    config.LLMConfig
	client *http.Client
}

// NewGenericHTTPProvider creates a new provider for generic HTTP models.
func NewGenericHTTPProvider(cfg config.LLMConfig) *GenericHTTPProvider {
	return &GenericHTTPProvider{
		cfg:    cfg,
		client: &http.Client{},
	}
}

func (p *GenericHTTPProvider) GetName() string {
	return "generic-http"
}

// GenerateContent sends the request to the configured HTTP endpoint and streams back response parts.
func (p *GenericHTTPProvider) GenerateContent(ctx context.Context, history []*mcp.Message, tools []types.Tool) (chan *mcp.ResponsePart, error) {
	respChan := make(chan *mcp.ResponsePart)

	// Convert internal message history to the format expected by OpenAI/DeepSeek
	deepSeekMessages := make([]map[string]interface{}, 0, len(history))
	for _, msg := range history {
		var contentBuilder strings.Builder
		if len(msg.Parts) > 0 {
			for _, part := range msg.Parts {
				if part.Text != nil {
					contentBuilder.WriteString(*part.Text)
				} else if part.ToolResult != nil && msg.Role == "tool" {
					// DeepSeek/OpenAI expects tool output as part of the assistant's message in their format,
					// or as a 'tool' role message in the history. We're using 'tool' role.
					contentBuilder.WriteString(fmt.Sprintf("\nTool %s output: %s\n", part.ToolResult.Name, part.ToolResult.Output))
				}
			}
		}
		deepSeekMessages = append(deepSeekMessages, map[string]interface{}{
			"role":    msg.Role,
			"content": contentBuilder.String(),
		})
	}

	// Convert internal tool schemas to the format expected by OpenAI/DeepSeek
	apiTools := make([]map[string]interface{}, 0, len(tools))
	for _, t := range tools {
		toolJSONSchema, err := t.Schema()
		if err != nil {
			fmt.Printf("Warning: could not get schema for tool '%s': %v\n", t.Name(), err)
			continue
		}
		var parsedSchema struct {
			Name        string                 
				Description string                 
				Parameters  map[string]interface{} 
		}
		if err := json.Unmarshal(toolJSONSchema, &parsedSchema); err != nil {
			fmt.Printf("Warning: could not parse schema for tool '%s': %v\n", t.Name(), err)
			continue
		}
		if parsedSchema.Parameters == nil {
			parsedSchema.Parameters = map[string]interface{}{"type": "object", "properties": map[string]interface{}{}}
		}
		apiTools = append(apiTools, map[string]interface{}{
			"type": "function",
			"function": map[string]interface{}{
				"name":        parsedSchema.Name,
				"description": parsedSchema.Description,
				"parameters":  parsedSchema.Parameters,
			},
		})
	}

	// Construct the final request body
	reqBody := map[string]interface{}{
		"model":    p.cfg.ModelName,
		"messages": deepSeekMessages,
	}
	if len(apiTools) > 0 {
		reqBody["tools"] = apiTools
	}

	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		close(respChan)
		return nil, fmt.Errorf("failed to marshal request body: %w", err)
	}

	go func() {
		defer close(respChan)

		req, err := http.NewRequestWithContext(ctx, "POST", p.cfg.Endpoint, bytes.NewBuffer(bodyBytes))
		if err != nil {
			respChan <- &mcp.ResponsePart{Error: fmt.Errorf("failed to create http request: %w", err)}
			return
		}

		req.Header.Set("Content-Type", "application/json")
		for key, value := range p.cfg.Headers {
			req.Header.Set(key, value)
		}
		if p.cfg.APIKey != "" && req.Header.Get("Authorization") == "" {
			req.Header.Set("Authorization", "Bearer "+p.cfg.APIKey)
		}

		resp, err := p.client.Do(req)
		if err != nil {
			respChan <- &mcp.ResponsePart{Error: fmt.Errorf("failed to send http request: %w", err)}
			return
		}
		defer resp.Body.Close()

		respBodyBytes, err := io.ReadAll(resp.Body)
		if err != nil {
			respChan <- &mcp.ResponsePart{Error: fmt.Errorf("failed to read response body: %w", err)}
			return
		}

		if resp.StatusCode != http.StatusOK {
			respChan <- &mcp.ResponsePart{Error: fmt.Errorf("http request failed with status: %s, body: %s", resp.Status, string(respBodyBytes))}
			return
		}

		var deepSeekResponse struct {
			Choices []struct {
				Message struct {
					Role      string  
						Content   *string 
						ToolCalls []struct {
							ID       string 
								Type     string 
								Function struct {
										Name      string 
										Arguments string 
										}
							}
				}
			}
		}

		if err := json.Unmarshal(respBodyBytes, &deepSeekResponse); err != nil {
			respChan <- &mcp.ResponsePart{Error: fmt.Errorf("failed to decode http response from provider: %w", err)}
			return
		}

		if len(deepSeekResponse.Choices) == 0 {
			respChan <- &mcp.ResponsePart{Error: fmt.Errorf("provider response contains no choices")}
			return
		}

		choiceMsg := deepSeekResponse.Choices[0].Message

		// Send content part if exists
		if choiceMsg.Content != nil && *choiceMsg.Content != "" {
			respChan <- &mcp.ResponsePart{Text: *choiceMsg.Content}
		}

		// Send tool call parts if exist
		if len(choiceMsg.ToolCalls) > 0 {
			for _, toolCall := range choiceMsg.ToolCalls {
				var args map[string]interface{}
				if err := json.Unmarshal([]byte(toolCall.Function.Arguments), &args); err != nil {
					respChan <- &mcp.ResponsePart{Error: fmt.Errorf("failed to unmarshal tool call arguments from provider: %w", err)}
					return
				}
				respChan <- &mcp.ResponsePart{ToolCall: &mcp.ToolCall{Name: toolCall.Function.Name, Args: args}}
			}
		}
	}()

	return respChan, nil
}
