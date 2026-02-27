package exec

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/pardnchiu/go-agent-skills/internal/utils"
)

//go:embed prompt/agentSelector.md
var agentSelectorPrompt string

type AgentEntryData struct {
	Name        string `json:"name"`
	Description string `json:"description"`
}

func GetAgentEntries() []AgentEntryData {
	configDir, err := utils.GetConfigDir()
	if err != nil {
		return []AgentEntryData{}
	}

	for _, dir := range configDir.Dirs {
		data, err := os.ReadFile(filepath.Join(dir, "config.json"))
		if err != nil {
			continue
		}
		var cfg struct {
			Models []AgentEntryData `json:"models"`
		}
		if json.Unmarshal(data, &cfg) == nil && len(cfg.Models) > 0 {
			return cfg.Models
		}
	}
	return []AgentEntryData{}
}

func selectAgent(ctx context.Context, bot Agent, agentEntries []AgentEntryData, userInput string) string {
	if len(agentEntries) == 0 {
		return ""
	}

	newMap := make(map[string]struct{}, len(agentEntries))
	for _, a := range agentEntries {
		newMap[a.Name] = struct{}{}
	}

	agentJson, err := json.Marshal(agentEntries)
	if err != nil {
		return ""
	}

	messages := []Message{
		{Role: "system", Content: agentSelectorPrompt},
		{
			Role:    "user",
			Content: fmt.Sprintf("Available agents:\n%s\nUser request: %s", agentJson, userInput),
		},
	}

	resp, err := bot.Send(ctx, messages, nil)
	if err != nil || len(resp.Choices) == 0 {
		return ""
	}

	answer := ""
	if content, ok := resp.Choices[0].Message.Content.(string); ok {
		answer = strings.Trim(strings.TrimSpace(content), "\"'` \n")
	}

	if answer == "NONE" || answer == "" {
		return ""
	}

	if _, ok := newMap[answer]; ok {
		return answer
	}

	return ""
}
