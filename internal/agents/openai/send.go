package openai

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
	defaultModel = "gpt-5-nano"
	chatAPI      = "https://api.openai.com/v1/chat/completions"
)

func (a *Agent) Execute(ctx context.Context, skill *skill.Skill, userInput string, output io.Writer, allowAll bool) error {
	return agents.Execute(ctx, a, a.workDir, skill, userInput, output, allowAll)
}

func (a *Agent) Send(ctx context.Context, messages []agents.Message, tools []model.Tool) (*agents.OpenAIOutput, error) {
	result, _, err := utils.POSTJson[agents.OpenAIOutput](ctx, a.httpClient, chatAPI, map[string]string{
		"Authorization": "Bearer " + a.apiKey,
		"Content-Type":  "application/json",
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
