package copilot

import (
	"context"
	"fmt"
	"io"

	"github.com/pardnchiu/go-agent-skills/internal/agents"
	"github.com/pardnchiu/go-agent-skills/internal/skill"
	"github.com/pardnchiu/go-agent-skills/internal/tools/model"
	"github.com/pardnchiu/go-agent-skills/internal/utils"
)

var (
	// gpt-4.1    1m/32k
	// gpt-5-mini 400k/128k
	// gpt-4o     128k/4k
	defaultModel = "gpt-4.1"
	chatAPI      = "https://api.githubcopilot.com/chat/completions"
)

func (a *Agent) Execute(ctx context.Context, skill *skill.Skill, userInput string, output io.Writer, allowAll bool) error {
	if err := a.checkExpires(ctx); err != nil {
		return err
	}
	return agents.Execute(ctx, a, a.workDir, skill, userInput, output, allowAll)
}

func (a *Agent) Send(ctx context.Context, messages []agents.Message, tools []model.Tool) (*agents.OpenAIOutput, error) {
	result, _, err := utils.POSTJson[agents.OpenAIOutput](ctx, a.httpClient, chatAPI, map[string]string{
		"Authorization":  "Bearer " + a.Refresh.Token,
		"Editor-Version": "vscode/1.95.0",
	}, map[string]any{
		"model":    defaultModel,
		"messages": messages,
		"tools":    tools,
	})
	if err != nil {
		return nil, fmt.Errorf("API request: %w", err)
	}
	if result.Error != nil {
		return nil, fmt.Errorf("API error: %s", result.Error.Message)
	}

	return &result, nil
}
