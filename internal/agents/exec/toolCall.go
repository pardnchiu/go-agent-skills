package exec

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	atypes "github.com/pardnchiu/go-agent-skills/internal/agents/types"
	"github.com/pardnchiu/go-agent-skills/internal/tools"
	"github.com/pardnchiu/go-agent-skills/internal/tools/types"
)

func toolCall(ctx context.Context, exec *types.Executor, choice OpenAIOutputChoices, sessionData *SessionData, events chan<- atypes.Event, allowAll bool, alreadyCall map[string]string) (*SessionData, map[string]string, error) {
	sessionData.messages = append(sessionData.messages, choice.Message)

	for _, tool := range choice.Message.ToolCalls {
		toolName := tool.Function.Name
		toolArg := tool.Function.Arguments

		hash := fmt.Sprintf("%v|%v", toolName, toolArg)
		if cached, ok := alreadyCall[hash]; ok && cached != "" {
			sessionData.messages = append(sessionData.messages, Message{
				Role:       "tool",
				Content:    cached,
				ToolCallID: tool.ID,
			})
			continue
		}

		if idx := strings.Index(toolName, "<|"); idx != -1 {
			toolName = toolName[:idx]
		}

		events <- atypes.Event{
			Type:     atypes.EventToolCall,
			ToolName: toolName,
			ToolArgs: tool.Function.Arguments,
			ToolID:   tool.ID,
		}

		if !allowAll {
			replyCh := make(chan bool, 1)
			events <- atypes.Event{
				Type:     atypes.EventToolConfirm,
				ToolName: toolName,
				ToolArgs: tool.Function.Arguments,
				ToolID:   tool.ID,
				ReplyCh:  replyCh,
			}
			proceed := <-replyCh
			if !proceed {
				events <- atypes.Event{
					Type:     atypes.EventToolSkipped,
					ToolName: toolName,
					ToolID:   tool.ID,
				}
				sessionData.tools = append(sessionData.tools, Message{
					Role:       "tool",
					Content:    "Skipped by user",
					ToolCallID: tool.ID,
				})
				sessionData.messages = append(sessionData.messages, Message{
					Role:       "tool",
					Content:    "Skipped by user",
					ToolCallID: tool.ID,
				})
				continue
			}
		}

		result, err := tools.Execute(ctx, exec, toolName, json.RawMessage(tool.Function.Arguments))
		if err != nil {
			result = fmt.Sprintf("Error '%s': %v", toolName, err)
		}

		alreadyCall[hash] = fmt.Sprintf("Tool '%s'\nresult: %s", toolName, result)

		events <- atypes.Event{
			Type:     atypes.EventToolResult,
			ToolName: toolName,
			ToolID:   tool.ID,
			Result:   result,
		}
		sessionData.tools = append(sessionData.tools, Message{
			Role:       "tool",
			Content:    fmt.Sprintf("Tool '%s'\nresult: %s", toolName, result),
			ToolCallID: tool.ID,
		})
		sessionData.messages = append(sessionData.messages, Message{
			Role:       "tool",
			Content:    fmt.Sprintf("Tool '%s'\nresult: %s", toolName, result),
			ToolCallID: tool.ID,
		})
	}
	return sessionData, alreadyCall, nil
}
