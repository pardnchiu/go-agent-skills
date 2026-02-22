package nvidia

import (
	"context"
	"fmt"

	"github.com/pardnchiu/go-agent-skills/internal/agents"
	atypes "github.com/pardnchiu/go-agent-skills/internal/agents/types"
	"github.com/pardnchiu/go-agent-skills/internal/skill"
	ttypes "github.com/pardnchiu/go-agent-skills/internal/tools/types"
	"github.com/pardnchiu/go-agent-skills/internal/utils"
)

var (
	// openai/gpt-oss-120b
	// z-ai/glm4.7
	// qwen/qwen3-235b-a22b
	// qwen/qwen3-coder-480b-a35b-instruct
	defaultModel = "openai/gpt-oss-20b"
	chatAPI      = "https://integrate.api.nvidia.com/v1/chat/completions"
)

func (a *Agent) Execute(ctx context.Context, skill *skill.Skill, userInput string, events chan<- atypes.Event, allowAll bool) error {
	return agents.Execute(ctx, a, a.workDir, skill, userInput, events, allowAll)
}

func (a *Agent) Send(ctx context.Context, messages []agents.Message, tools []ttypes.Tool) (*agents.OpenAIOutput, error) {
	result, _, err := utils.POST[agents.OpenAIOutput](ctx, a.httpClient, chatAPI, map[string]string{
		"Authorization": "Bearer " + a.apiKey,
		"Content-Type":  "application/json",
	}, map[string]any{
		"model":    defaultModel,
		"messages": messages,
		"tools":    tools,
	}, "json")
	if err != nil {
		return nil, fmt.Errorf("utils.POST: %w", err)
	}
	if result.Error != nil {
		return nil, fmt.Errorf("utils.POST: %s", result.Error.Message)
	}

	return &result, nil
}
