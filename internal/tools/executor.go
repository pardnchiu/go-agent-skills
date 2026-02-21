package tools

import (
	_ "embed"
	"encoding/json"
	"fmt"

	"github.com/pardnchiu/go-agent-skills/internal/tools/apis/googleRSS"
	yahoofinance "github.com/pardnchiu/go-agent-skills/internal/tools/apis/yahooFinance"
	"github.com/pardnchiu/go-agent-skills/internal/tools/file"
	"github.com/pardnchiu/go-agent-skills/internal/tools/types"
)

//go:embed embed/tools.json
var toolsMap []byte

//go:embed embed/commands.json
var allowCommand []byte

func NewExecutor(workPath string) (*types.Executor, error) {
	var tools []types.Tool
	if err := json.Unmarshal(toolsMap, &tools); err != nil {
		return nil, fmt.Errorf("failed to unmarshal tools: %w", err)
	}

	var commands []string
	if err := json.Unmarshal(allowCommand, &commands); err != nil {
		return nil, fmt.Errorf("failed to unmarshal commands: %w", err)
	}

	allowedCommand := make(map[string]bool, len(commands))
	for _, cmd := range commands {
		allowedCommand[cmd] = true
	}

	return &types.Executor{
		WorkPath:       workPath,
		AllowedCommand: allowedCommand,
		Exclude:        file.ListExcludes(workPath),
		Tools:          tools,
	}, nil
}

func Execute(e *types.Executor, name string, args json.RawMessage) (string, error) {
	switch name {
	case "read_file":
		var params struct {
			Path string `json:"path"`
		}
		if err := json.Unmarshal(args, &params); err != nil {
			return "", fmt.Errorf("failed to unmarshal json (%s): %w", name, err)
		}
		return file.ReadFile(e, params.Path)

	case "list_files":
		var params struct {
			Path      string `json:"path"`
			Recursive bool   `json:"recursive"`
		}
		if err := json.Unmarshal(args, &params); err != nil {
			return "", fmt.Errorf("failed to unmarshal json (%s): %w", name, err)
		}
		return file.ListFiles(e, params.Path, params.Recursive)

	case "glob_files":
		var params struct {
			Pattern string `json:"pattern"`
		}
		if err := json.Unmarshal(args, &params); err != nil {
			return "", fmt.Errorf("failed to unmarshal json (%s): %w", name, err)
		}
		return file.GlobFiles(e, params.Pattern)

	case "write_file":
		var params struct {
			Path    string `json:"path"`
			Content string `json:"content"`
		}
		if err := json.Unmarshal(args, &params); err != nil {
			return "", fmt.Errorf("failed to unmarshal json (%s): %w", name, err)
		}
		return file.WriteFile(e, params.Path, params.Content)

	case "search_content":
		var params struct {
			Pattern     string `json:"pattern"`
			FilePattern string `json:"file_pattern"`
		}
		if err := json.Unmarshal(args, &params); err != nil {
			return "", err
		}
		return searchContent(e, params.Pattern, params.FilePattern)

	case "patch_edit":
		var params struct {
			Path      string `json:"path"`
			OldString string `json:"old_string"`
			NewString string `json:"new_string"`
		}
		if err := json.Unmarshal(args, &params); err != nil {
			return "", fmt.Errorf("failed to unmarshal json (%s): %w", name, err)
		}
		return file.PatchEdit(e, params.Path, params.OldString, params.NewString)

	case "run_command":
		var params struct {
			Command string `json:"command"`
		}
		if err := json.Unmarshal(args, &params); err != nil {
			return "", err
		}
		return runCommand(e, params.Command)

	case "fetch_yahoo_finance":
		var params struct {
			Symbol   string `json:"symbol"`
			Interval string `json:"interval"`
			Range    string `json:"range"`
		}
		if err := json.Unmarshal(args, &params); err != nil {
			return "", fmt.Errorf("failed to unmarshal json (%s): %w", name, err)
		}
		return yahoofinance.Fetch(params.Symbol, params.Interval, params.Range)

	case "fetch_google_rss":
		var params struct {
			Keyword string `json:"keyword"`
			Time    string `json:"time"`
			Lang    string `json:"lang"`
		}
		if err := json.Unmarshal(args, &params); err != nil {
			return "", fmt.Errorf("failed to unmarshal json (%s): %w", name, err)
		}
		return googleRSS.Fetch(params.Keyword, params.Time, params.Lang)

	default:
		return "", fmt.Errorf("unknown tool: %s", name)
	}
}
