package nvidia

import (
	"context"
	"fmt"

	"github.com/pardnchiu/go-agent-skills/internal/agents/exec"
	atypes "github.com/pardnchiu/go-agent-skills/internal/agents/types"
	"github.com/pardnchiu/go-agent-skills/internal/skill"
	ttypes "github.com/pardnchiu/go-agent-skills/internal/tools/types"
	"github.com/pardnchiu/go-agent-skills/internal/utils"
)

const (
	chatAPI = "https://integrate.api.nvidia.com/v1/chat/completions"
)

func (a *Agent) Execute(ctx context.Context, skill *skill.Skill, userInput string, events chan<- atypes.Event, allowAll bool) error {
	return exec.Execute(ctx, a, a.workDir, skill, userInput, events, allowAll)
}

func (a *Agent) Send(ctx context.Context, messages []atypes.Message, tools []ttypes.Tool) (*atypes.Output, error) {
	result, _, err := utils.POST[atypes.Output](ctx, a.httpClient, chatAPI, map[string]string{
		"Authorization": "Bearer " + a.apiKey,
		"Content-Type":  "application/json",
	}, map[string]any{
		"model":    a.model,
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
