package claude

import (
	"fmt"
	"net/http"
	"os"
)

type Agent struct {
	httpClient *http.Client
	apiKey     string
	workDir    string
}

func New() (*Agent, error) {
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
