package gemini

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
	// gemini-2.5-pro   1m/64k
	// gemini-2.5-flash 1m/64k
	defaultModel = "gemini-2.5-pro"
	baseAPI      = "https://generativelanguage.googleapis.com/v1beta/models/"
)

func (a *Agent) Execute(ctx context.Context, skill *skill.Skill, userInput string, events chan<- atypes.Event, allowAll bool) error {
	return agents.Execute(ctx, a, a.workDir, skill, userInput, events, allowAll)
}

func (a *Agent) Send(ctx context.Context, messages []agents.Message, tools []ttypes.Tool) (*agents.OpenAIOutput, error) {
	var systemPrompt string
	var newMessages []Content

	for _, msg := range messages {
		if msg.Role == "system" {
			if content, ok := msg.Content.(string); ok {
				systemPrompt = content
			}
			continue
		}

		message := a.convertToContent(msg)
		newMessages = append(newMessages, message)
	}

	newTools := a.convertToTools(tools)
	apiURL := fmt.Sprintf("%s%s:generateContent?key=%s", baseAPI, defaultModel, a.apiKey)
	requestBody := a.generateRequestBody(newMessages, systemPrompt, newTools)

	result, _, err := utils.POST[Output](ctx, a.httpClient, apiURL, map[string]string{
		"Content-Type": "application/json",
	}, requestBody, "json")
	if err != nil {
		return nil, fmt.Errorf("utils.POST: %w", err)
	}

	return a.convertToOutput(&result), nil
}

func (a *Agent) convertToContent(message agents.Message) Content {
	content := Content{}
	if message.ToolCallID != "" {
		content.Role = "function"
		data := map[string]any{}
		if contentStr, ok := message.Content.(string); ok {
			data["result"] = contentStr
		}
		content.Parts = []Part{
			{
				FunctionResponse: &FunctionResponse{
					Name:     message.ToolCallID,
					Response: data,
				},
			},
		}
		return content
	}

	role := message.Role
	if role == "assistant" {
		role = "model"
	}
	content.Role = role

	if len(message.ToolCalls) > 0 {
		for _, tool := range message.ToolCalls {
			var args map[string]any
			json.Unmarshal([]byte(tool.Function.Arguments), &args)
			content.Parts = append(content.Parts, Part{
				FunctionCall: &FunctionCall{
					Name: tool.Function.Name,
					Args: args,
				},
			})
		}
		return content
	}

	if contentStr, ok := message.Content.(string); ok {
		content.Parts = []Part{
			{Text: contentStr},
		}
	}

	return content
}

func (a *Agent) convertToTools(tools []ttypes.Tool) []map[string]any {
	newTools := make([]map[string]any, len(tools))
	for i, tool := range tools {
		var params map[string]any
		json.Unmarshal(tool.Function.Parameters, &params)

		newTools[i] = map[string]any{
			"name":        tool.Function.Name,
			"description": tool.Function.Description,
			"parameters":  params,
		}
	}
	return newTools
}

func (a *Agent) generateRequestBody(messages []Content, prompt string, newTools []map[string]any) map[string]any {
	body := map[string]any{
		"contents": messages,
	}

	if prompt != "" {
		body["systemInstruction"] = map[string]any{
			"parts": []map[string]any{
				{"text": prompt},
			},
		}
	}

	if len(newTools) > 0 {
		body["tools"] = []map[string]any{
			{"functionDeclarations": newTools},
		}
	}
	return body
}

func (a *Agent) convertToOutput(resp *Output) *agents.OpenAIOutput {
	output := &agents.OpenAIOutput{
		Choices: make([]struct {
			Message      agents.Message `json:"message"`
			Delta        agents.Message `json:"delta"`
			FinishReason string         `json:"finish_reason,omitempty"`
		}, 1),
	}

	if len(resp.Candidates) == 0 {
		return output
	}

	candidate := resp.Candidates[0]
	var toolCalls []agents.OpenAIToolCall
	var textContent string

	for _, part := range candidate.Content.Parts {
		if part.Text != "" {
			textContent = part.Text
		} else if part.FunctionCall != nil {
			args := "{}"
			if part.FunctionCall.Args != nil {
				data, err := json.Marshal(part.FunctionCall.Args)
				if err != nil {
					continue
				}
				args = string(data)
			}

			toolCall := agents.OpenAIToolCall{
				ID:   part.FunctionCall.Name,
				Type: "function",
			}
			toolCall.Function.Name = part.FunctionCall.Name
			toolCall.Function.Arguments = args
			toolCalls = append(toolCalls, toolCall)
		}
	}

	output.Choices[0].Message = agents.Message{
		Role:      "assistant",
		Content:   textContent,
		ToolCalls: toolCalls,
	}
	output.Choices[0].FinishReason = candidate.FinishReason

	return output
}
