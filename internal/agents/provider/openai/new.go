package openai

import (
	"fmt"
	"net/http"
	"os"
	"strings"
)

type Agent struct {
	httpClient *http.Client
	model      string
	apiKey     string
	workDir    string
}

const (
	defaultModel = "gpt-5-mini"
	prefix       = "openai@"
)

func New(model ...string) (*Agent, error) {
	usedModel := defaultModel
	if len(model) > 0 && strings.HasPrefix(model[0], prefix) {
		usedModel = strings.TrimPrefix(model[0], prefix)
	}
	apiKey := os.Getenv("OPENAI_API_KEY")
	if apiKey == "" {
		return nil, fmt.Errorf("os.Getenv: OPENAI_API_KEY is required")
	}

	workDir, _ := os.Getwd()

	return &Agent{
		httpClient: &http.Client{},
		model:      usedModel,
		apiKey:     apiKey,
		workDir:    workDir,
	}, nil
}
