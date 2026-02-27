package compat

import (
	"fmt"
	"net/http"
	"os"
	"strings"
)

type Agent struct {
	httpClient *http.Client
	model      string
	baseURL    string
	apiKey     string
	workDir    string
}

const (
	defaultModel = "qwen3:8b"
	prefix       = "compat@"
)

func New(model ...string) (*Agent, error) {
	usedModel := defaultModel
	if len(model) > 0 && strings.HasPrefix(model[0], prefix) {
		usedModel = strings.TrimPrefix(model[0], prefix)
	}

	baseURL := os.Getenv("COMPAT_URL")
	if baseURL == "" {
		baseURL = "http://localhost:11434"
	}
	baseURL = strings.TrimRight(baseURL, "/")

	apiKey := os.Getenv("COMPAT_API_KEY")

	workDir, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("os.Getwd: %w", err)
	}

	return &Agent{
		httpClient: &http.Client{},
		model:      usedModel,
		baseURL:    baseURL,
		apiKey:     apiKey,
		workDir:    workDir,
	}, nil
}
