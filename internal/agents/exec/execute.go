package exec

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	atypes "github.com/pardnchiu/go-agent-skills/internal/agents/types"
	"github.com/pardnchiu/go-agent-skills/internal/skill"
	"github.com/pardnchiu/go-agent-skills/internal/tools"
	"github.com/pardnchiu/go-agent-skills/internal/utils"
)

func Execute(ctx context.Context, agent atypes.Agent, workDir string, skill *skill.Skill, userInput string, events chan<- atypes.Event, allowAll bool) error {
	// if skill is empty, then treat as no skill
	if skill != nil && skill.Content == "" {
		skill = nil
	}

	configDir, err := utils.GetConfigDir("sessions")
	if err != nil {
		return fmt.Errorf("utils.ConfigDir: %w", err)
	}

	prompt := getSystemPrompt(workDir, skill)
	sessionData, sessionID, err := getSession(prompt, userInput)
	if err != nil {
		return fmt.Errorf("getSession: %w", err)
	}

	exec, err := tools.NewExecutor(workDir, sessionID)
	if err != nil {
		return fmt.Errorf("tools.NewExecutor: %w", err)
	}

	limit := MaxToolIterations
	if skill != nil {
		limit = MaxSkillIterations
	}

	alreadyCall := make(map[string]string)
	emptyCount := 0
	const maxEmpty = 3
	for i := 0; i < limit; i++ {
		resp, err := agent.Send(ctx, sessionData.Messages, exec.Tools)
		if err != nil {
			return err
		}

		if len(resp.Choices) == 0 {
			emptyCount++
			if emptyCount >= maxEmpty {
				events <- atypes.Event{Type: atypes.EventText, Text: "工具無法取得資料，請稍後再試或改用其他方式查詢。"}
				events <- atypes.Event{Type: atypes.EventDone}
				return nil
			}
			continue
		}
		emptyCount = 0

		choice := resp.Choices[0]
		if len(choice.Message.ToolCalls) > 0 {
			sessionData, alreadyCall, err = toolCall(ctx, exec, choice, sessionData, events, allowAll, alreadyCall)
			if err != nil {
				return err
			}
			continue
		}

		switch value := choice.Message.Content.(type) {
		case string:
			text := value
			if text == "" {
				text = "工具無法取得資料，請稍後再試或改用其他方式查詢。"
			}
			cleaned := extractSummary(configDir, sessionID, text)

			events <- atypes.Event{Type: atypes.EventText, Text: cleaned}

			choice.Message.Content = fmt.Sprintf("ts:%d\n%s", time.Now().Unix(), cleaned)

			sessionData.Messages = append(sessionData.Messages, choice.Message)

			err := writeHistory(choice, configDir, sessionData, sessionID)
			if err != nil {
				slog.Warn("Failed to write history",
					slog.String("error", err.Error()))
			}
		case nil:
			events <- atypes.Event{Type: atypes.EventText, Text: "工具無法取得資料，請稍後再試或改用其他方式查詢。"}
		default:
			return fmt.Errorf("unexpected content type: %T", choice.Message.Content)
		}

		events <- atypes.Event{Type: atypes.EventDone}

		if len(sessionData.Tools) > 0 {
			now := time.Now()
			date := now.Format("2006-01-02")
			dateWithSec := now.Format("2006-01-02-15-04-05")
			toolActionsDir := filepath.Join(configDir.Work, sessionID, date)
			if err := os.MkdirAll(toolActionsDir, 0755); err == nil {
				filename := dateWithSec + ".json"
				toolActionsPath := filepath.Join(toolActionsDir, filename)
				if data, err := json.Marshal(sessionData.Tools); err == nil {
					os.WriteFile(toolActionsPath, data, 0644)
				}
			}
		}
		return nil
	}

	summaryMessages := append(sessionData.Messages, atypes.Message{
		Role:    "user",
		Content: "請根據以上工具查詢結果，整理並總結回答原始問題。",
	})
	resp, err := agent.Send(ctx, summaryMessages, nil)
	if err == nil && len(resp.Choices) > 0 {
		if text, ok := resp.Choices[0].Message.Content.(string); ok && text != "" {
			cleaned := extractSummary(configDir, sessionID, text)
			events <- atypes.Event{Type: atypes.EventText, Text: cleaned}
			events <- atypes.Event{Type: atypes.EventDone}
			return nil
		}
	}

	events <- atypes.Event{Type: atypes.EventText, Text: "工具無法取得資料，請稍後再試或改用其他方式查詢。"}
	events <- atypes.Event{Type: atypes.EventDone}
	return nil
}

func getSystemPrompt(workDir string, skill *skill.Skill) string {
	if skill == nil {
		return strings.NewReplacer(
			"{{.WorkPath}}", workDir,
			"{{.SkillPath}}", "None",
			"{{.SkillExt}}", "",
			"{{.Content}}", "",
		).Replace(systemPrompt)
	}
	content := skill.Content

	for _, prefix := range []string{"scripts/", "templates/", "assets/"} {
		resolved := filepath.Join(skill.Path, prefix)

		if _, err := os.Stat(resolved); err == nil {
			content = strings.ReplaceAll(content, prefix, resolved+string(filepath.Separator))
		}
	}

	return strings.NewReplacer(
		"{{.WorkPath}}", workDir,
		"{{.SkillPath}}", skill.Path,
		"{{.SkillExt}}", skillExtensionPrompt,
		"{{.Content}}", content,
	).Replace(systemPrompt)
}
