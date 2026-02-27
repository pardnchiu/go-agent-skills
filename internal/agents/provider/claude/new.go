package claude

import (
	"fmt"
	"net/http"
	"os"
	"strings"
)

type Agent struct {
	httpClient *http.Client
	apiKey     string
	workDir    string
}

var (
	// claude-sonnet-4-5 200K/64000
	// claude-opus-4-6   200K/128K
	// claude-opus-4-5   200K/128K
	defaultModel = "claude-sonnet-4-5"
	prefix       = "claude@"
)

func New(model ...string) (*Agent, error) {
	if len(model) > 0 && strings.HasPrefix(model[0], prefix) {
		defaultModel = strings.TrimPrefix(model[0], prefix)
	}
	apiKey := os.Getenv("ANTHROPIC_API_KEY")
	if apiKey == "" {
		return nil, fmt.Errorf("os.Getenv: ANTHROPIC_API_KEY is required")
	}

	workDir, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("os.Getwd: %w", err)
	}

	return &Agent{
		httpClient: &http.Client{},
		apiKey:     apiKey,
		workDir:    workDir,
	}, nil
}
