package exec

import (
	"context"
	_ "embed"
	"fmt"
	"os"

	atypes "github.com/pardnchiu/go-agent-skills/internal/agents/types"
	"github.com/pardnchiu/go-agent-skills/internal/skill"
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

func Run(ctx context.Context, bot atypes.Agent, registry atypes.AgentRegistry, scanner *skill.Scanner, userInput string, events chan<- atypes.Event, allowAll bool) error {
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
	if chosen := selectAgent(ctx, bot, registry.Entries, userInput); chosen != "" {
		if a, ok := registry.Registry[chosen]; ok {
			agent = a
			events <- atypes.Event{Type: atypes.EventText, Text: chosen}
		} else {
			events <- atypes.Event{Type: atypes.EventText, Text: fmt.Sprintf("Agent %s not found, use fallback", chosen)}
		}
	} else {
		events <- atypes.Event{Type: atypes.EventText, Text: fmt.Sprintf("Agent %s not found, use fallback", chosen)}
	}

	return Execute(ctx, agent, workDir, matchedSkill, userInput, events, allowAll)
}
