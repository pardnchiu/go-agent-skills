package tools

import (
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/pardnchiu/go-agent-skills/internal/tools/apiAdapter"
	"github.com/pardnchiu/go-agent-skills/internal/tools/apis"
	"github.com/pardnchiu/go-agent-skills/internal/tools/apis/searchWeb"
	"github.com/pardnchiu/go-agent-skills/internal/tools/browser"
	"github.com/pardnchiu/go-agent-skills/internal/tools/calculator"
	"github.com/pardnchiu/go-agent-skills/internal/tools/file"
	"github.com/pardnchiu/go-agent-skills/internal/tools/types"
	"github.com/pardnchiu/go-agent-skills/internal/utils"
)

//go:embed embed/tools.json
var toolsMap []byte

//go:embed embed/commands.json
var allowCommand []byte

func NewExecutor(workPath, sessionID string) (*types.Executor, error) {
	var tools []types.Tool
	if err := json.Unmarshal(toolsMap, &tools); err != nil {
		return nil, fmt.Errorf("json.Unmarshal: %w", err)
	}

	var commands []string
	if err := json.Unmarshal(allowCommand, &commands); err != nil {
		return nil, fmt.Errorf("json.Unmarshal: %w", err)
	}

	allowedCommand := make(map[string]bool, len(commands))
	for _, cmd := range commands {
		allowedCommand[cmd] = true
	}

	apiToolbox := apiAdapter.New()

	if configDir, err := utils.ConfigDir("apis"); err == nil {
		apiToolbox.Load(configDir.Home)
		apiToolbox.Load(configDir.Work)
	}

	for _, tool := range apiToolbox.GetTools() {
		data, err := json.Marshal(tool)
		if err != nil {
			continue
		}
		var t types.Tool
		if err := json.Unmarshal(data, &t); err != nil {
			continue
		}
		tools = append(tools, t)
	}

	return &types.Executor{
		WorkPath:       workPath,
		SessionID:      sessionID,
		AllowedCommand: allowedCommand,
		Exclude:        file.ListExcludes(workPath),
		Tools:          tools,
		APIToolbox:     apiToolbox,
	}, nil
}

func Execute(ctx context.Context, e *types.Executor, name string, args json.RawMessage) (string, error) {
	// * get all api tools
	if strings.HasPrefix(name, "api_") && e.APIToolbox != nil && e.APIToolbox.IsExist(name) {
		var params map[string]any
		if err := json.Unmarshal(args, &params); err != nil {
			return "", fmt.Errorf("json.Unmarshal: %w", err)
		}
		return e.APIToolbox.Execute(name, params)
	}

	switch name {
	case "read_file", "list_files", "glob_files", "search_content", "search_history", "write_file", "patch_edit":
		return file.Routes(e, name, args)

	case "send_http_request", "fetch_yahoo_finance", "fetch_google_rss", "fetch_weather":
		return apis.Routes(e, name, args)

	case "run_command":
		var params struct {
			Command string `json:"command"`
		}
		if err := json.Unmarshal(args, &params); err != nil {
			return "", fmt.Errorf("json.Unmarshal: %w", err)
		}
		return runCommand(e, params.Command)

	case "fetch_page":
		var params struct {
			URL string `json:"url"`
		}
		if err := json.Unmarshal(args, &params); err != nil {
			return "", fmt.Errorf("failed to unmarshal json (%s): %w", name, err)
		}

		result, err := browser.Load(params.URL)
		if err != nil {
			return "", err
		}
		return result, nil

	case "search_web":
		var params struct {
			Query string `json:"query"`
			Range string `json:"range"`
			Limit int    `json:"limit"`
		}
		if err := json.Unmarshal(args, &params); err != nil {
			return "", fmt.Errorf("failed to unmarshal json (%s): %w", name, err)
		}
		return searchWeb.Search(ctx, params.Query, searchWeb.TimeRange(params.Range), params.Limit)

	case "calculate":
		var params struct {
			Expression string `json:"expression"`
		}
		if err := json.Unmarshal(args, &params); err != nil {
			return "", fmt.Errorf("json.Unmarshal: %w", err)
		}
		return calculator.Calc(params.Expression)

	default:
		return "", fmt.Errorf("unknown tool: %s", name)
	}
}
