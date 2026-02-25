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

		agent := selectAgent()
		scanner := skill.NewScanner()

		// 嘗試第二個參數是否為已知 skill name
		if len(os.Args) >= 4 {
			if targetSkill, ok := scanner.Skills.ByName[os.Args[2]]; ok {
				// 明確指定 skill：run <skill_name> <input>
				userInput := os.Args[3]
				ctx, cancel := context.WithCancel(context.Background())
				defer cancel()

				if err := runWithEvents(ctx, cancel, func(ch chan<- atypes.Event) error {
					return agent.Execute(ctx, targetSkill, userInput, ch, allowAll)
				}); err != nil && ctx.Err() == nil {
					slog.Error("failed to execute skill", slog.String("error", err.Error()))
					os.Exit(1)
				}
				return
			}
		}

		userInput := os.Args[2]
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		if err := runWithEvents(ctx, cancel, func(ch chan<- atypes.Event) error {
			return exec.Run(ctx, agent, scanner, userInput, ch, allowAll)
		}); err != nil && ctx.Err() == nil {
			slog.Error("failed to execute", slog.String("error", err.Error()))
			os.Exit(1)
		}

		// agent := selectAgent()
		// scanner := skill.NewScanner()
		// targetSkill, ok := scanner.Skills.ByName[skillName]
		// if !ok {
		// 	slog.Error("skill not found", slog.String("name", skillName))
		// 	os.Exit(1)
		// }

		// ctx := context.Background()
		// if err := agent.Execute(ctx, targetSkill, userInput, os.Stdout, allowAll); err != nil {
		// 	slog.Error("failed to execute skill", slog.String("error", err.Error()))
		// 	os.Exit(1)
		// }
		return
	}

}

func printTool(ev atypes.Event) {
	var args map[string]any
	json.Unmarshal([]byte(ev.ToolArgs), &args)

	switch ev.ToolName {
	case "read_file":
		fmt.Printf("\r\033[K[*] Read File — \033[36m%s\033[0m", args["path"])
	case "list_files":
		fmt.Printf("\r\033[K[*] List Directory — \033[36m%s\033[0m", args["path"])
	case "glob_files":
		fmt.Printf("\r\033[K[*] Glob Files — \033[35m%s\033[0m", args["pattern"])
	case "write_file":
		fmt.Printf("\r\033[K[*] Write File — \033[33m%s\033[0m", args["path"])
	case "search_content":
		fmt.Printf("[\r\033[K*] Search Content — \033[35m%s\033[0m", args["pattern"])
	case "patch_edit":
		fmt.Printf("\r\033[K[*] Patch Edit — \033[33m%s\033[0m", args["path"])
	case "run_command":
		fmt.Printf("\r\033[K[*] Run Command — \033[32m%s\033[0m", args["command"])
	case "fetch_yahoo_finance":
		fmt.Printf("\r\033[K[*] Fetch Ticker — \033[34m%s (%s)\033[0m", args["symbol"], args["range"])
	case "fetch_google_rss":
		fmt.Printf("\r\033[K[*] Fetch News — \033[34m%s (%s)\033[0m", args["keyword"], args["time"])
	case "fetch_page":
		url := fmt.Sprintf("%v", args["url"])
		if len(url) > 64 {
			url = url[:61] + "..."
		}
		fmt.Printf("\r\033[K[*] Fetch Page — \033[34m%s\033[0m", url)
	default:
		fmt.Printf("\r\033[K[*] Tool: %s — \033[90m%s\033[0m", ev.ToolName, ev.ToolArgs)
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
			fmt.Printf("\r\033[K[*] %s", ev.Text)

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

func selectAgent() exec.Agent {
	prompt := promptui.Select{
		Label: "Select Agent",
		Items: []string{
			"GitHub Copilot",
			"OpenAI",
			"Claude",
			"Gemini",
			"Nvidia",
		},
		HideSelected: true,
	}

	idx, _, err := prompt.Run()
	if err != nil {
		slog.Error("agent selection failed", slog.String("error", err.Error()))
		os.Exit(1)
	}

	switch idx {
	case 0:
		agent, err := copilot.New()
		if err != nil {
			slog.Error("failed to initialize Copilot", slog.String("error", err.Error()))
			os.Exit(1)
		}
		return agent

	case 1:
		agent, err := openai.New()
		if err != nil {
			slog.Error("failed to initialize OpenAI", slog.String("error", err.Error()))
			os.Exit(1)
		}
		return agent

	case 2:
		agent, err := claude.New()
		if err != nil {
			slog.Error("failed to initialize Anthropic", slog.String("error", err.Error()))
			os.Exit(1)
		}
		return agent

	case 3:
		agent, err := gemini.New()
		if err != nil {
			slog.Error("failed to initialize Anthropic", slog.String("error", err.Error()))
			os.Exit(1)
		}
		return agent

	case 4:
		agent, err := nvidia.New()
		if err != nil {
			slog.Error("failed to initialize Anthropic", slog.String("error", err.Error()))
			os.Exit(1)
		}
		return agent

	default:
		os.Exit(1)
		return nil
	}
}
