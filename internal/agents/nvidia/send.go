package nvidia

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
	// openai/gpt-oss-120b
	// z-ai/glm4.7
	// qwen/qwen3-235b-a22b
	// qwen/qwen3-coder-480b-a35b-instruct
	defaultModel = "openai/gpt-oss-120b"
	chatAPI      = "https://integrate.api.nvidia.com/v1/chat/completions"
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
