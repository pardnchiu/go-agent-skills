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

func Execute(ctx context.Context, agent Agent, workDir string, skill *skill.Skill, userInput string, events chan<- atypes.Event, allowAll bool) error {
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

	alreadyCall := make(map[string]string)
	for i := 0; i < MaxToolIterations; i++ {
		resp, err := agent.Send(ctx, sessionData.messages, exec.Tools)
		if err != nil {
			return err
		}

		if len(resp.Choices) == 0 {
			events <- atypes.Event{Type: atypes.EventDone}
			return nil
		}

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

			choice.Message.Content = fmt.Sprintf("當前時間：%s\n%s", time.Now().Format("2006-01-02T15:04:05 MST (UTC-07:00)"), cleaned)

			sessionData.messages = append(sessionData.messages, choice.Message)

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

		if len(sessionData.tools) > 0 {
			date := time.Now().Format("2006-01-02")
			dateWithSec := time.Now().Format("2006-01-02-15-04-05")
			toolActionsDir := filepath.Join(configDir.Work, sessionID, date)
			if err := os.MkdirAll(toolActionsDir, 0755); err == nil {
				filename := dateWithSec + ".json"
				toolActionsPath := filepath.Join(toolActionsDir, filename)
				if data, err := json.Marshal(sessionData.tools); err == nil {
					os.WriteFile(toolActionsPath, data, 0644)
				}
			}
		}
		return nil
	}

	return fmt.Errorf("exceeded max iterations (%d)", MaxToolIterations)
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
