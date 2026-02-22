package claude

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/pardnchiu/go-agent-skills/internal/agents"
	atypes "github.com/pardnchiu/go-agent-skills/internal/agents/types"
	"github.com/pardnchiu/go-agent-skills/internal/skill"
	ttypes "github.com/pardnchiu/go-agent-skills/internal/tools/types"
	"github.com/pardnchiu/go-agent-skills/internal/utils"
)

var (
	// claude-sonnet-4-5 200K/64000
	// claude-opus-4-6   200K/128K
	// claude-opus-4-5   200K/128K
	defaultModel = "claude-sonnet-4-5"
	messagesAPI  = "https://api.anthropic.com/v1/messages"
	maxTokens    = 16384
)

func (a *Agent) Execute(ctx context.Context, skill *skill.Skill, userInput string, events chan<- atypes.Event, allowAll bool) error {
	return agents.Execute(ctx, a, a.workDir, skill, userInput, events, allowAll)
}

func (a *Agent) Send(ctx context.Context, messages []agents.Message, tools []ttypes.Tool) (*agents.OpenAIOutput, error) {
	var systemPrompt string
	var newMessages []map[string]any

	for _, msg := range messages {
		if msg.Role == "system" {
			if content, ok := msg.Content.(string); ok {
				systemPrompt = content
			}
			continue
		}

		message := a.convertToMessage(msg)
		newMessages = append(newMessages, message)
	}

	newTools := a.convertToTools(tools)
	result, _, err := utils.POST[Output](ctx, a.httpClient, messagesAPI, map[string]string{
		"x-api-key":         a.apiKey,
		"anthropic-version": "2023-06-01",
		"Content-Type":      "application/json",
	}, map[string]any{
		"model":      defaultModel,
		"max_tokens": maxTokens,
		"system":     systemPrompt,
		"messages":   newMessages,
		"tools":      newTools,
	}, "json")
	if err != nil {
		return nil, fmt.Errorf("utils.POST: %w", err)
	}

	if result.Error != nil {
		return nil, fmt.Errorf("result.Error: %s", result.Error.Message)
	}

	if result.StopReason == "max_tokens" {
		return nil, fmt.Errorf("exceeded max_tokens (%d)", maxTokens)
	}

	return a.convertToOutput(&result), nil
}

func (a *Agent) convertToMessage(message agents.Message) map[string]any {
	if message.ToolCallID != "" {
		return map[string]any{
			"role": "user",
			"content": []map[string]any{
				{
					"type":        "tool_result",
					"tool_use_id": message.ToolCallID,
					"content":     message.Content,
				},
			},
		}
	}

	if len(message.ToolCalls) > 0 {
		var content []map[string]any
		for _, tool := range message.ToolCalls {
			var input map[string]any
			json.Unmarshal([]byte(tool.Function.Arguments), &input)
			content = append(content, map[string]any{
				"type":  "tool_use",
				"id":    tool.ID,
				"name":  tool.Function.Name,
				"input": input,
			})
		}
		return map[string]any{
			"role":    message.Role,
			"content": content,
		}
	}

	return map[string]any{
		"role":    message.Role,
		"content": message.Content,
	}
}

func (a *Agent) convertToTools(tools []ttypes.Tool) []map[string]any {
	newTools := make([]map[string]any, len(tools))
	for i, tool := range tools {
		newTools[i] = map[string]any{
			"name":         tool.Function.Name,
			"description":  tool.Function.Description,
			"input_schema": json.RawMessage(tool.Function.Parameters),
		}
	}
	return newTools
}

func (a *Agent) convertToOutput(resp *Output) *agents.OpenAIOutput {
	output := &agents.OpenAIOutput{
		Choices: make([]struct {
			Message      agents.Message `json:"message"`
			Delta        agents.Message `json:"delta"`
			FinishReason string         `json:"finish_reason,omitempty"`
		}, 1),
	}

	var toolCalls []agents.OpenAIToolCall
	var textContent string

	for _, item := range resp.Content {
		if item.Type == "text" {
			textContent = item.Text
		} else if item.Type == "tool_use" {
			arg := ""
			if item.Input != nil {
				data, err := json.Marshal(item.Input)
				if err != nil {
					continue
				}
				arg = string(data)
			}

			toolCall := agents.OpenAIToolCall{
				ID:   item.ID,
				Type: "function",
			}
			toolCall.Function.Name = item.Name
			toolCall.Function.Arguments = arg
			toolCalls = append(toolCalls, toolCall)
		}
	}

	output.Choices[0].Message = agents.Message{
		Role:      "assistant",
		Content:   textContent,
		ToolCalls: toolCalls,
	}

	return output
}
