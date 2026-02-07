package client

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/pardnchiu/go-agent-skills/internal/utils"
)

var (
	CopilotChatAPI = "https://api.githubcopilot.com/chat/completions"
)

type Tool struct {
	Type     string       `json:"type"`
	Function ToolFunction `json:"function"`
}

type ToolFunction struct {
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Parameters  json.RawMessage `json:"parameters"`
}

type Message struct {
	Role      string `json:"role"`
	Content   any    `json:"content,omitempty"`
	ToolCalls []struct {
		ID       string `json:"id"`
		Type     string `json:"type"`
		Function struct {
			Name      string `json:"name"`
			Arguments string `json:"arguments"`
		} `json:"function"`
	} `json:"tool_calls,omitempty"`
	ToolCallID string `json:"tool_call_id,omitempty"`
}

type OpenAIOutput struct {
	Choices []struct {
		Message      Message `json:"message"`
		Delta        Message `json:"delta"`
		FinishReason string  `json:"finish_reason,omitempty"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
		Type    string `json:"type"`
		Code    string `json:"code"`
	} `json:"error,omitempty"`
}

func (c *CopilotAgent) SendChat(ctx context.Context, messages []Message, toolDefs []Tool) (*OpenAIOutput, error) {
	result, _, error := utils.POSTJson[OpenAIOutput](ctx, c.httpClient, CopilotChatAPI, map[string]string{
		"Authorization":         "Bearer " + c.Refresh.Token,
		"Editor-Version":        "vscode/1.95.0",
		"Editor-Plugin-Version": "copilot/1.245.0",
		"Openai-Organization":   "github-copilot",
	}, map[string]any{
		"model":    CopilotDefaultModel,
		"messages": messages,
		"tools":    toolDefs,
	})
	if error != nil {
		return nil, fmt.Errorf("API request: %w", error)
	}
	if result.Error != nil {
		return nil, fmt.Errorf("API error: %s", result.Error.Message)
	}

	return &result, nil
}
