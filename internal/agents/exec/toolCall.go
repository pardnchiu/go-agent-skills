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

func toolCall(ctx context.Context, exec *types.Executor, choice atypes.OutputChoices, sessionData *atypes.AgentSession, events chan<- atypes.Event, allowAll bool, alreadyCall map[string]string) (*atypes.AgentSession, map[string]string, error) {
	sessionData.Messages = append(sessionData.Messages, choice.Message)

	for _, tool := range choice.Message.ToolCalls {
		toolName := tool.Function.Name
		toolArg := tool.Function.Arguments

		hash := fmt.Sprintf("%v|%v", toolName, toolArg)
		if cached, ok := alreadyCall[hash]; ok && cached != "" {
			sessionData.Messages = append(sessionData.Messages, atypes.Message{
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
				sessionData.Tools = append(sessionData.Tools, atypes.Message{
					Role:       "tool",
					Content:    "Skipped by user",
					ToolCallID: tool.ID,
				})
				sessionData.Messages = append(sessionData.Messages, atypes.Message{
					Role:       "tool",
					Content:    "Skipped by user",
					ToolCallID: tool.ID,
				})
				continue
			}
		}

		result, err := tools.Execute(ctx, exec, toolName, json.RawMessage(tool.Function.Arguments))
		if err != nil {
			result = "no data"
		}

		content := fmt.Sprintf("[%s] %s", toolName, result)
		alreadyCall[hash] = content

		events <- atypes.Event{
			Type:     atypes.EventToolResult,
			ToolName: toolName,
			ToolID:   tool.ID,
			Result:   result,
		}
		sessionData.Tools = append(sessionData.Tools, atypes.Message{
			Role:       "tool",
			Content:    content,
			ToolCallID: tool.ID,
		})
		sessionData.Messages = append(sessionData.Messages, atypes.Message{
			Role:       "tool",
			Content:    content,
			ToolCallID: tool.ID,
		})
	}
	return sessionData, alreadyCall, nil
}
