package main

import (
	"log/slog"
	"os"
	"strings"

	"github.com/pardnchiu/agenvoy/internal/agents/exec"
	"github.com/pardnchiu/agenvoy/internal/agents/provider/claude"
	"github.com/pardnchiu/agenvoy/internal/agents/provider/compat"
	"github.com/pardnchiu/agenvoy/internal/agents/provider/copilot"
	"github.com/pardnchiu/agenvoy/internal/agents/provider/gemini"
	"github.com/pardnchiu/agenvoy/internal/agents/provider/nvidia"
	"github.com/pardnchiu/agenvoy/internal/agents/provider/openai"
	agentTypes "github.com/pardnchiu/agenvoy/internal/agents/types"
)

func getAgentRegistry() agentTypes.AgentRegistry {
	newFn := map[string]func(string) (agentTypes.Agent, error){
		"copilot": func(m string) (agentTypes.Agent, error) { return copilot.New(m) },
		"openai":  func(m string) (agentTypes.Agent, error) { return openai.New(m) },
		"compat":  func(m string) (agentTypes.Agent, error) { return compat.New(m) },
		"claude":  func(m string) (agentTypes.Agent, error) { return claude.New(m) },
		"gemini":  func(m string) (agentTypes.Agent, error) { return gemini.New(m) },
		"nvidia":  func(m string) (agentTypes.Agent, error) { return nvidia.New(m) },
	}

	agentEntries := exec.GetAgentEntries()
	// var fallback exec.Agent
	// registry := make(map[string]exec.Agent, len(agentEntries))
	// entries := make([]exec.AgentEntryData, 0, len(agentEntries))

	agentRegistry := agentTypes.AgentRegistry{
		Registry: make(map[string]agentTypes.Agent, len(agentEntries)),
		Entries:  make([]agentTypes.AgentEntry, 0, len(agentEntries)),
	}
	for _, e := range agentEntries {
		providerFull := strings.SplitN(e.Name, "@", 2)[0]
		provider := providerFull
		if idx := strings.Index(providerFull, "["); idx != -1 {
			provider = providerFull[:idx]
		}
		fn, ok := newFn[provider]
		if !ok {
			continue
		}
		a, err := fn(e.Name)
		if err != nil {
			slog.Warn("failed to initialize agent", slog.String("name", e.Name), slog.String("error", err.Error()))
			continue
		}
		agentRegistry.Registry[e.Name] = a
		agentRegistry.Entries = append(agentRegistry.Entries, e)
		if agentRegistry.Fallback == nil {
			agentRegistry.Fallback = a
		}
	}

	if agentRegistry.Fallback == nil {
		slog.Error("no agent available; check API keys")
		os.Exit(1)
	}

	return agentRegistry
}
