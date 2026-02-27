package exec

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"os"

	atypes "github.com/pardnchiu/go-agent-skills/internal/agents/types"
	"github.com/pardnchiu/go-agent-skills/internal/skill"
	ttypes "github.com/pardnchiu/go-agent-skills/internal/tools/types"
)

//go:embed prompt/systemPrompt.md
var systemPrompt string

//go:embed prompt/skillExtension.md
var skillExtensionPrompt string

//go:embed prompt/summaryPrompt.md
var summaryPrompt string

var (
	MaxToolIterations = 32
)

type Message struct {
	Role       string           `json:"role"`
	Content    any              `json:"content,omitempty"`
	ToolCalls  []OpenAIToolCall `json:"tool_calls,omitempty"`
	ToolCallID string           `json:"tool_call_id,omitempty"`
}

type OpenAIToolCall struct {
	ID       string `json:"id"`
	Type     string `json:"type"`
	Function struct {
		Name      string `json:"name"`
		Arguments string `json:"arguments"`
	} `json:"function"`
}

type OpenAIOutput struct {
	Choices []OpenAIOutputChoices `json:"choices"`
	Error   *struct {
		Message string      `json:"message"`
		Type    string      `json:"type"`
		Code    json.Number `json:"code"`
	} `json:"error,omitempty"`
}

type OpenAIOutputChoices struct {
	Message      Message `json:"message"`
	Delta        Message `json:"delta"`
	FinishReason string  `json:"finish_reason,omitempty"`
}

type Agent interface {
	Send(ctx context.Context, messages []Message, toolDefs []ttypes.Tool) (*OpenAIOutput, error)
	Execute(ctx context.Context, skill *skill.Skill, userInput string, events chan<- atypes.Event, allowAll bool) error
}

type AgentRegistryData struct {
	Registry map[string]Agent
	Entries  []AgentEntryData
	Fallback Agent
}

func Run(ctx context.Context, bot Agent, registry AgentRegistryData, scanner *skill.Scanner, userInput string, events chan<- atypes.Event, allowAll bool) error {
	workDir, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("os.Getwd: %w", err)
	}

	events <- atypes.Event{Type: atypes.EventText, Text: "Matching Skills"}
	matchedSkill := selectSkill(ctx, bot, scanner, userInput)
	if matchedSkill != nil {
		events <- atypes.Event{Type: atypes.EventText, Text: fmt.Sprintf("Skill: %s", matchedSkill.Name)}
	} else {
		events <- atypes.Event{Type: atypes.EventText, Text: "No matched"}
	}

	agent := registry.Fallback
	events <- atypes.Event{Type: atypes.EventText, Text: "Matching Agent"}
	if chosen := selectAgent(ctx, bot, registry.Entries, userInput); chosen != "" {
		if a, ok := registry.Registry[chosen]; ok {
			agent = a
			events <- atypes.Event{Type: atypes.EventText, Text: chosen}
		}
	}

	return Execute(ctx, agent, workDir, matchedSkill, userInput, events, allowAll)
}
