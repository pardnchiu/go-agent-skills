package tools

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/pardnchiu/go-agent-skills/internal/tools/apiAdapter"
	"github.com/pardnchiu/go-agent-skills/internal/tools/apis/googleRSS"
	"github.com/pardnchiu/go-agent-skills/internal/tools/apis/weatherReport"
	"github.com/pardnchiu/go-agent-skills/internal/tools/apis/yahooFinance"
	"github.com/pardnchiu/go-agent-skills/internal/tools/browser"
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
	apiToolbox.Load(filepath.Join(workPath, ".config", "apis"))

	home, err := os.UserHomeDir()
	if err != nil {
		apiToolbox.Load(filepath.Join(home, ".config", "go-agent-skills", "apis"))
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
		AllowedCommand: allowedCommand,
		Exclude:        file.ListExcludes(workPath),
		Tools:          tools,
		APIToolbox:     apiToolbox,
	}, nil
}

func Execute(e *types.Executor, name string, args json.RawMessage) (string, error) {
	// * get all api tools
	if strings.HasPrefix(name, "api_") && e.APIToolbox != nil && e.APIToolbox.IsExist(name) {
		var params map[string]any
		if err := json.Unmarshal(args, &params); err != nil {
			return "", fmt.Errorf("json.Unmarshal: %w", err)
		}
		return e.APIToolbox.Execute(name, params)
	}

	switch name {
	case "read_file", "list_files", "glob_files", "write_file", "patch_edit":
		return file.Routes(e, name, args)

	case "search_content":
		var params struct {
			Pattern     string `json:"pattern"`
			FilePattern string `json:"file_pattern"`
		}
		if err := json.Unmarshal(args, &params); err != nil {
			return "", fmt.Errorf("json.Unmarshal: %w", err)
		}
		return searchContent(e, params.Pattern, params.FilePattern)

	case "run_command":
		var params struct {
			Command string `json:"command"`
		}
		if err := json.Unmarshal(args, &params); err != nil {
			return "", fmt.Errorf("json.Unmarshal: %w", err)
		}
		return runCommand(e, params.Command)

	case "fetch_yahoo_finance":
		var params struct {
			Symbol   string `json:"symbol"`
			Interval string `json:"interval"`
			Range    string `json:"range"`
		}
		if err := json.Unmarshal(args, &params); err != nil {
			return "", fmt.Errorf("json.Unmarshal: %w", err)
		}
		return yahooFinance.Fetch(params.Symbol, params.Interval, params.Range)

	case "fetch_google_rss":
		var params struct {
			Keyword string `json:"keyword"`
			Time    string `json:"time"`
			Lang    string `json:"lang"`
		}
		if err := json.Unmarshal(args, &params); err != nil {
			return "", fmt.Errorf("json.Unmarshal: %w", err)
		}
		return googleRSS.Fetch(params.Keyword, params.Time, params.Lang)

	case "send_http_request":
		var params struct {
			URL         string            `json:"url"`
			Method      string            `json:"method"`
			Headers     map[string]string `json:"headers"`
			Body        map[string]any    `json:"body"`
			ContentType string            `json:"content_type"`
			Timeout     int               `json:"timeout"`
		}
		if err := json.Unmarshal(args, &params); err != nil {
			return "", fmt.Errorf("json.Unmarshal: %w", err)
		}
		return apiAdapter.Send(params.URL, params.Method, params.Headers, params.Body, params.ContentType, params.Timeout)

	case "fetch_weather":
		var params struct {
			City           string      `json:"city"`
			Days           int         `json:"days"`
			HourlyInterval json.Number `json:"hourly_interval"`
		}
		if err := json.Unmarshal(args, &params); err != nil {
			return "", fmt.Errorf("failed to unmarshal json (%s): %w", name, err)
		}
		hourlyInterval, _ := params.HourlyInterval.Int64()
		return weatherReport.Fetch(params.City, params.Days, int(hourlyInterval))

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

	default:
		return "", fmt.Errorf("unknown tool: %s", name)
	}
}
