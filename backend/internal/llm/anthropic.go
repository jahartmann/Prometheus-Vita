package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type AnthropicProvider struct {
	apiKey string
	client *http.Client
}

func NewAnthropicProvider(apiKey string) *AnthropicProvider {
	return &AnthropicProvider{
		apiKey: apiKey,
		client: &http.Client{
			Timeout: 120 * time.Second,
		},
	}
}

func (p *AnthropicProvider) Name() string {
	return "anthropic"
}

func (p *AnthropicProvider) Models() []string {
	return []string{"claude-sonnet-4-20250514", "claude-haiku-4-20250414"}
}

type anthropicRequest struct {
	Model     string             `json:"model"`
	MaxTokens int                `json:"max_tokens"`
	System    string             `json:"system,omitempty"`
	Messages  []anthropicMessage `json:"messages"`
	Tools     []anthropicTool    `json:"tools,omitempty"`
}

type anthropicMessage struct {
	Role    string      `json:"role"`
	Content interface{} `json:"content"`
}

type anthropicTextBlock struct {
	Type string `json:"type"`
	Text string `json:"text"`
}

type anthropicToolUseBlock struct {
	Type  string          `json:"type"`
	ID    string          `json:"id"`
	Name  string          `json:"name"`
	Input json.RawMessage `json:"input"`
}

type anthropicToolResultBlock struct {
	Type       string `json:"type"`
	ToolUseID  string `json:"tool_use_id"`
	Content    string `json:"content"`
}

type anthropicTool struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	InputSchema json.RawMessage `json:"input_schema"`
}

type anthropicResponse struct {
	Content    []json.RawMessage `json:"content"`
	StopReason string            `json:"stop_reason"`
	Usage      anthropicUsage    `json:"usage"`
}

type anthropicUsage struct {
	InputTokens  int `json:"input_tokens"`
	OutputTokens int `json:"output_tokens"`
}

type anthropicContentBlock struct {
	Type string `json:"type"`
}

func (p *AnthropicProvider) Complete(ctx context.Context, req CompletionRequest) (*CompletionResponse, error) {
	var systemMsg string
	var messages []anthropicMessage

	for _, m := range req.Messages {
		if m.Role == "system" {
			systemMsg = m.Content
			continue
		}

		if m.Role == "tool" {
			messages = append(messages, anthropicMessage{
				Role: "user",
				Content: []anthropicToolResultBlock{
					{
						Type:      "tool_result",
						ToolUseID: m.ToolCallID,
						Content:   m.Content,
					},
				},
			})
			continue
		}

		if m.Role == "assistant" && len(m.ToolCalls) > 0 {
			var content []interface{}
			if m.Content != "" {
				content = append(content, anthropicTextBlock{Type: "text", Text: m.Content})
			}
			for _, tc := range m.ToolCalls {
				var input json.RawMessage
				if err := json.Unmarshal([]byte(tc.Function.Arguments), &input); err != nil {
					input = json.RawMessage(tc.Function.Arguments)
				}
				content = append(content, anthropicToolUseBlock{
					Type:  "tool_use",
					ID:    tc.ID,
					Name:  tc.Function.Name,
					Input: input,
				})
			}
			messages = append(messages, anthropicMessage{
				Role:    "assistant",
				Content: content,
			})
			continue
		}

		messages = append(messages, anthropicMessage{
			Role:    m.Role,
			Content: m.Content,
		})
	}

	var tools []anthropicTool
	for _, t := range req.Tools {
		tools = append(tools, anthropicTool{
			Name:        t.Function.Name,
			Description: t.Function.Description,
			InputSchema: t.Function.Parameters,
		})
	}

	maxTokens := req.MaxTokens
	if maxTokens == 0 {
		maxTokens = 4096
	}

	anthropicReq := anthropicRequest{
		Model:     req.Model,
		MaxTokens: maxTokens,
		System:    systemMsg,
		Messages:  messages,
		Tools:     tools,
	}

	body, err := json.Marshal(anthropicReq)
	if err != nil {
		return nil, fmt.Errorf("marshal anthropic request: %w", err)
	}

	httpReq, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.anthropic.com/v1/messages", bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("create anthropic request: %w", err)
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("x-api-key", p.apiKey)
	httpReq.Header.Set("anthropic-version", "2023-06-01")

	resp, err := p.client.Do(httpReq)
	if err != nil {
		return nil, fmt.Errorf("anthropic request failed: %w", err)
	}
	defer resp.Body.Close()

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("read anthropic response: %w", err)
	}

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("anthropic returned status %d: %s", resp.StatusCode, string(respBody))
	}

	var anthropicResp anthropicResponse
	if err := json.Unmarshal(respBody, &anthropicResp); err != nil {
		return nil, fmt.Errorf("unmarshal anthropic response: %w", err)
	}

	result := &CompletionResponse{
		FinishReason: anthropicResp.StopReason,
		Usage: Usage{
			PromptTokens:     anthropicResp.Usage.InputTokens,
			CompletionTokens: anthropicResp.Usage.OutputTokens,
			TotalTokens:      anthropicResp.Usage.InputTokens + anthropicResp.Usage.OutputTokens,
		},
	}

	for _, rawBlock := range anthropicResp.Content {
		var block anthropicContentBlock
		if err := json.Unmarshal(rawBlock, &block); err != nil {
			continue
		}

		switch block.Type {
		case "text":
			var textBlock anthropicTextBlock
			if err := json.Unmarshal(rawBlock, &textBlock); err == nil {
				result.Content += textBlock.Text
			}
		case "tool_use":
			var toolBlock anthropicToolUseBlock
			if err := json.Unmarshal(rawBlock, &toolBlock); err == nil {
				args, _ := json.Marshal(toolBlock.Input)
				result.ToolCalls = append(result.ToolCalls, ToolCall{
					ID:   toolBlock.ID,
					Type: "function",
					Function: ToolCallFunction{
						Name:      toolBlock.Name,
						Arguments: string(args),
					},
				})
			}
		}
	}

	if anthropicResp.StopReason == "tool_use" {
		result.FinishReason = "tool_calls"
	}
	if anthropicResp.StopReason == "end_turn" {
		result.FinishReason = "stop"
	}

	return result, nil
}
