package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"slices"
	"sort"
	"strings"
	"time"

	"github.com/manifoldco/promptui"
	"github.com/pardnchiu/go-agent-skills/internal/agents/exec"
	"github.com/pardnchiu/go-agent-skills/internal/agents/provider/claude"
	"github.com/pardnchiu/go-agent-skills/internal/agents/provider/compat"
	"github.com/pardnchiu/go-agent-skills/internal/agents/provider/copilot"
	"github.com/pardnchiu/go-agent-skills/internal/agents/provider/gemini"
	"github.com/pardnchiu/go-agent-skills/internal/agents/provider/nvidia"
	"github.com/pardnchiu/go-agent-skills/internal/agents/provider/openai"
	atypes "github.com/pardnchiu/go-agent-skills/internal/agents/types"
	"github.com/pardnchiu/go-agent-skills/internal/skill"

	"github.com/joho/godotenv"
)

func init() {
	if err := godotenv.Load(); err != nil {
		slog.Warn("No .env file found, relying on environment variables")
	}
}

func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage:")
		fmt.Println("  go run cmd/cli/main.go list")
		fmt.Println("  go run cmd/cli/main.go run <skill_name> <input> [--allow]")
		os.Exit(1)
	}

	if os.Args[1] == "list" {
		scanner := skill.NewScanner()

		if len(scanner.Skills.ByName) == 0 {
			fmt.Println("No skills found")
			fmt.Println("\nScanned paths:")
			for _, path := range scanner.Skills.Paths {
				fmt.Printf("  - %s\n", path)
			}
			return
		}

		names := scanner.List()
		sort.Strings(names)

		fmt.Printf("Found %d skill(s):\n\n", len(names))
		for _, name := range names {
			s := scanner.Skills.ByName[name]
			fmt.Printf("• %s\n", name)
			if s.Description != "" {
				fmt.Printf("  %s\n", s.Description)
			}
			fmt.Printf("  Path: %s\n\n", s.Path)
		}
		return
	}

	if os.Args[1] == "run" {
		if len(os.Args) < 3 {
			fmt.Println("Usage: go run cmd/cli/main.go run <input> [--allow]")
			fmt.Println("       go run cmd/cli/main.go run <skill_name> <input> [--allow]")
			os.Exit(1)
		}

		allowAll := slices.Contains(os.Args[3:], "--allow")

		agentRegistry := getAgentRegistry()
		scanner := skill.NewScanner()

		userInput := os.Args[2]
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		selectorBot, err := nvidia.New("compat@qwen3:8b")
		if err != nil {
			slog.Error("failed to initialize", slog.String("error", err.Error()))
			os.Exit(1)
		}

		if err := runWithEvents(ctx, cancel, func(ch chan<- atypes.Event) error {
			return exec.Run(ctx, selectorBot, agentRegistry, scanner, userInput, ch, allowAll)
		}); err != nil && ctx.Err() == nil {
			slog.Error("failed to execute", slog.String("error", err.Error()))
			os.Exit(1)
		}
		return
	}

}

func printTool(ev atypes.Event) {
	var args map[string]any
	json.Unmarshal([]byte(ev.ToolArgs), &args)

	switch ev.ToolName {
	case "read_file":
		fmt.Printf("[*] Read File — \033[36m%s\033[0m\n", args["path"])
	case "list_files":
		fmt.Printf("[*] List Directory — \033[36m%s\033[0m\n", args["path"])
	case "glob_files":
		fmt.Printf("[*] Glob Files — \033[35m%s\033[0m\n", args["pattern"])
	case "write_file":
		fmt.Printf("[*] Write File — \033[33m%s\033[0m\n", args["path"])
	case "search_content":
		fmt.Printf("[*] Search Content — \033[35m%s\033[0m\n", args["pattern"])
	case "patch_edit":
		fmt.Printf("[*] Patch Edit — \033[33m%s\033[0m\n", args["path"])
	case "run_command":
		fmt.Printf("[*] Run Command — \033[32m%s\033[0m\n", args["command"])
	case "fetch_yahoo_finance":
		fmt.Printf("[*] Fetch Ticker — \033[34m%s (%s)\033[0m\n", args["symbol"], args["range"])
	case "fetch_google_rss":
		fmt.Printf("[*] Fetch News — \033[34m%s (%s)\033[0m\n", args["keyword"], args["time"])
	case "fetch_page":
		url := fmt.Sprintf("%v", args["url"])
		if len(url) > 64 {
			url = url[:61] + "..."
		}
		fmt.Printf("[*] Fetch Page — \033[34m%s\033[0m\n", url)
	default:
		fmt.Printf("[*] Tool: %s — \033[90m%s\033[0m\n", ev.ToolName, ev.ToolArgs)
	}
}

func printContent(ev atypes.Event) {
	fmt.Print("\033[90m──────────────────────────────────────────────────\n")
	fmt.Printf("%s\n", strings.TrimSpace(ev.Result))
	fmt.Print("──────────────────────────────────────────────────\033[0m\n")
}

func runWithEvents(_ context.Context, cancel context.CancelFunc, fn func(chan<- atypes.Event) error) error {
	start := time.Now()
	ch := make(chan atypes.Event, 16)
	var execErr error

	go func() {
		defer close(ch)
		execErr = fn(ch)
	}()

	for ev := range ch {
		switch ev.Type {
		case atypes.EventText:
			fmt.Printf("[*] %s\n", ev.Text)

		case atypes.EventToolCall:
			printTool(ev)

		case atypes.EventToolConfirm:
			prompt := promptui.Select{
				Label:        fmt.Sprintf("Run %s?", ev.ToolName),
				Items:        []string{"Yes", "Skip", "Stop"},
				Size:         3,
				HideSelected: true,
			}
			idx, _, err := prompt.Run()
			if err != nil || idx == 2 {
				fmt.Printf("[x] User stopped\n")
				cancel()
				ev.ReplyCh <- false
			} else if idx == 1 {
				fmt.Printf("[x] User skipped: %s\n", ev.ToolName)
				ev.ReplyCh <- false
			} else {
				ev.ReplyCh <- true
			}

		case atypes.EventToolSkipped:
			fmt.Printf("[x] Skipped: %s\n", ev.ToolName)

		case atypes.EventToolResult:
			if ev.ToolName == "write_file" {
				printContent(ev)
			}

		case atypes.EventError:
			if ev.Err != nil {
				fmt.Fprintf(os.Stderr, "[!] Error: %v\n", ev.Err)
			}

		case atypes.EventDone:
			fmt.Printf(" (%s)", time.Since(start).Round(time.Millisecond))
			fmt.Println()
		}
	}

	return execErr
}

func getAgentRegistry() atypes.AgentRegistry {
	newFn := map[string]func(string) (atypes.Agent, error){
		"copilot": func(m string) (atypes.Agent, error) { return copilot.New(m) },
		"openai":  func(m string) (atypes.Agent, error) { return openai.New(m) },
		"compat":  func(m string) (atypes.Agent, error) { return compat.New(m) },
		"claude":  func(m string) (atypes.Agent, error) { return claude.New(m) },
		"gemini":  func(m string) (atypes.Agent, error) { return gemini.New(m) },
		"nvidia":  func(m string) (atypes.Agent, error) { return nvidia.New(m) },
	}

	agentEntries := exec.GetAgentEntries()
	// var fallback exec.Agent
	// registry := make(map[string]exec.Agent, len(agentEntries))
	// entries := make([]exec.AgentEntryData, 0, len(agentEntries))

	agentRegistry := atypes.AgentRegistry{
		Registry: make(map[string]atypes.Agent, len(agentEntries)),
		Entries:  make([]atypes.AgentEntry, 0, len(agentEntries)),
	}
	for _, e := range agentEntries {
		provider := strings.SplitN(e.Name, "@", 2)[0]
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
