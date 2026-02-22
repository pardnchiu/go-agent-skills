package tools

import (
	_ "embed"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/pardnchiu/go-agent-skills/internal/tools/apiAdapter"
	"github.com/pardnchiu/go-agent-skills/internal/tools/apis/googleRSS"
	"github.com/pardnchiu/go-agent-skills/internal/tools/apis/yahooFinance"
	"github.com/pardnchiu/go-agent-skills/internal/tools/file"
	"github.com/pardnchiu/go-agent-skills/internal/tools/types"
)

//go:embed embed/tools.json
var toolsMap []byte

//go:embed embed/commands.json
var allowCommand []byte

const cacheExpiry = 1 * time.Hour

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
	case "read_file":
		var params struct {
			Path string `json:"path"`
		}
		if err := json.Unmarshal(args, &params); err != nil {
			return "", fmt.Errorf("json.Unmarshal: %w", err)
		}
		return file.ReadFile(e, params.Path)

	case "list_files":
		var params struct {
			Path      string `json:"path"`
			Recursive bool   `json:"recursive"`
		}
		if err := json.Unmarshal(args, &params); err != nil {
			return "", fmt.Errorf("json.Unmarshal: %w", err)
		}
		return file.ListFiles(e, params.Path, params.Recursive)

	case "glob_files":
		var params struct {
			Pattern string `json:"pattern"`
		}
		if err := json.Unmarshal(args, &params); err != nil {
			return "", fmt.Errorf("json.Unmarshal: %w", err)
		}
		return file.GlobFiles(e, params.Pattern)

	case "write_file":
		var params struct {
			Path    string `json:"path"`
			Content string `json:"content"`
		}
		if err := json.Unmarshal(args, &params); err != nil {
			return "", fmt.Errorf("json.Unmarshal: %w", err)
		}
		return file.WriteFile(e, params.Path, params.Content)

	case "search_content":
		var params struct {
			Pattern     string `json:"pattern"`
			FilePattern string `json:"file_pattern"`
		}
		if err := json.Unmarshal(args, &params); err != nil {
			return "", fmt.Errorf("json.Unmarshal: %w", err)
		}
		return searchContent(e, params.Pattern, params.FilePattern)

	case "patch_edit":
		var params struct {
			Path      string `json:"path"`
			OldString string `json:"old_string"`
			NewString string `json:"new_string"`
		}
		if err := json.Unmarshal(args, &params); err != nil {
			return "", fmt.Errorf("json.Unmarshal: %w", err)
		}
		return file.PatchEdit(e, params.Path, params.OldString, params.NewString)

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

	default:
		return "", fmt.Errorf("unknown tool: %s", name)
	}
}
