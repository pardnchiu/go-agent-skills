package agents

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/manifoldco/promptui"
	"github.com/pardnchiu/go-agent-skills/internal/skill"
	"github.com/pardnchiu/go-agent-skills/internal/tools"
	"github.com/pardnchiu/go-agent-skills/internal/tools/model"
)

//go:embed sysprompt.md
var sysPrompt string

//go:embed sysPromptBase.md
var sysPromptBase string

//go:embed prompt/skillSelector.md
var promptSkillSelectpr string

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
	Choices []struct {
		Message      Message `json:"message"`
		Delta        Message `json:"delta"`
		FinishReason string  `json:"finish_reason,omitempty"`
	} `json:"choices"`
	Error *struct {
		Message string `json:"message"`
		Type    string `json:"type"`
		Code    string `json:"code"`
	} `json:"error,omitempty"`
}

type Agent interface {
	Send(ctx context.Context, messages []Message, toolDefs []model.Tool) (*OpenAIOutput, error)
	Execute(ctx context.Context, skill *skill.Skill, userInput string, output io.Writer, allowAll bool) error
}

func ExecuteAuto(ctx context.Context, agent Agent, scanner *skill.Scanner, userInput string, output io.Writer, allowAll bool) error {
	workDir, _ := os.Getwd()

	matched := selectSkill(ctx, agent, scanner, userInput)
	if matched != nil {
		fmt.Fprintf(output, "[*] Auto-selected skill: %s\n", matched.Name)
		return Execute(ctx, agent, workDir, matched, userInput, output, allowAll)
	}

	fmt.Fprintln(output, "[*] No matching skill found, using tools directly")
	return Execute(ctx, agent, workDir, nil, userInput, output, allowAll)
}

func selectSkill(ctx context.Context, agent Agent, scanner *skill.Scanner, userInput string) *skill.Skill {
	skills := scanner.List()
	if len(skills) == 0 {
		return nil
	}

	var sb strings.Builder
	for _, skill := range skills {
		s := scanner.Skills.ByName[skill]
		sb.WriteString(fmt.Sprintf("- %s: %s\n", skill, s.Description))
	}

	messages := []Message{
		{
			Role:    "system",
			Content: promptSkillSelectpr,
		},
		{
			Role:    "user",
			Content: fmt.Sprintf("Available skills:\n%s\nUser request: %s", sb.String(), userInput),
		},
	}

	resp, err := agent.Send(ctx, messages, nil)
	if err != nil || len(resp.Choices) == 0 {
		return nil
	}

	answer := ""
	if content, ok := resp.Choices[0].Message.Content.(string); ok {
		answer = strings.TrimSpace(content)
	}

	if answer == "NONE" || answer == "" {
		return nil
	}

	if s, ok := scanner.Skills.ByName[answer]; ok {
		return s
	}

	cleaned := strings.Trim(answer, "\"'` \n")
	if s, ok := scanner.Skills.ByName[cleaned]; ok {
		return s
	}

	return nil
}

func Execute(ctx context.Context, agent Agent, workDir string, skill *skill.Skill, userInput string, output io.Writer, allowAll bool) error {
	if skill != nil && skill.Content == "" {
		return fmt.Errorf("SKILL.md is empty: %s", skill.Path)
	}

	exec, err := tools.NewExecutor(workDir)
	if err != nil {
		return fmt.Errorf("failed to create executor: %w", err)
	}

	systemPrompt := systemPrompt(workDir, skill)
	messages := []Message{
		{
			Role:    "system",
			Content: systemPrompt,
		},
		{
			Role:    "user",
			Content: userInput,
		},
	}

	for i := 0; i < MaxToolIterations; i++ {
		if i > 0 {
			time.Sleep(1 * time.Second)
		}

		resp, err := agent.Send(ctx, messages, exec.Tools)
		if err != nil {
			return err
		}

		if len(resp.Choices) == 0 {
			return fmt.Errorf("no choices in response")
		}

		choice := resp.Choices[0]

		if len(choice.Message.ToolCalls) > 0 {
			messages = append(messages, choice.Message)

			for _, e := range choice.Message.ToolCalls {
				fmt.Printf("[*] Tool: %s — \033[90m%s\033[0m\n", e.Function.Name, e.Function.Arguments)

				if !allowAll {
					var args map[string]any
					if err := json.Unmarshal([]byte(e.Function.Arguments), &args); err == nil {
						fmt.Printf("\033[90m──────────────────────────────────────────────────\n")
						for k, v := range args {
							fmt.Printf("- %s: %v\n", k, v)
						}
						fmt.Printf("──────────────────────────────────────────────────\033[0m\n")
					} else {
						fmt.Printf("\033[90m──────────────────────────────────────────────────\n")
						fmt.Printf("- %s\n", e.Function.Arguments)
						fmt.Printf("──────────────────────────────────────────────────\033[0m\n")
					}
					prompt := promptui.Select{
						Label: "Continue?",
						Items: []string{
							"Yes",
							"Skip",
							"Stop",
						},
						Size:         3,
						HideSelected: true,
					}

					idx, _, err := prompt.Run()
					if err != nil || idx == 1 {
						fmt.Printf("[x] User skipped\n")
						messages = append(messages, Message{
							Role:       "tool",
							Content:    "User skipped",
							ToolCallID: e.ID,
						})
						continue
					} else if idx == 2 {
						fmt.Printf("[x] User stopped\n")
						return nil
					}
				}

				result, err := tools.Execute(exec, e.Function.Name, json.RawMessage(e.Function.Arguments))
				if err != nil {
					result = "Error: " + err.Error()
				}

				if e.Function.Name == "write_file" {
					fmt.Printf("\033[90m──────────────────────────────────────────────────\n")
					fmt.Printf("%s\n", strings.TrimSpace(result))
					fmt.Printf("──────────────────────────────────────────────────\033[0m\n")
				}

				messages = append(messages, Message{
					Role:       "tool",
					Content:    result,
					ToolCallID: e.ID,
				})
			}
			continue
		}

		switch v := choice.Message.Content.(type) {
		case string:
			if v != "" {
				output.Write([]byte(v))
				output.Write([]byte("\n"))
			}
		case nil:
		default:
			return fmt.Errorf("unexpected content type: %T", choice.Message.Content)
		}
		return nil
	}

	return fmt.Errorf("exceeded max iterations (%d)", MaxToolIterations)
}

func systemPrompt(workPath string, skill *skill.Skill) string {
	if skill == nil {
		return strings.ReplaceAll(sysPromptBase, "{{.WorkPath}}", workPath)
	}
	content := skill.Content

	for _, prefix := range []string{"scripts/", "templates/", "assets/"} {
		resolved := filepath.Join(skill.Path, prefix)

		if _, err := os.Stat(resolved); err == nil {
			content = strings.ReplaceAll(content, prefix, resolved+string(filepath.Separator))
		}
	}

	return strings.NewReplacer(
		"{{.WorkPath}}", workPath,
		"{{.SkillPath}}", skill.Path,
		"{{.Content}}", content,
	).Replace(sysPrompt)
}
