package gemini

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
	// gemini-2.5-pro   1m/64k
	// gemini-2.5-flash 1m/64k
	defaultModel = "gemini-2.5-pro"
	prefix       = "gemini@"
)

func New(model ...string) (*Agent, error) {
	if len(model) > 0 && strings.HasPrefix(model[0], prefix) {
		defaultModel = strings.TrimPrefix(model[0], prefix)
	}

	apiKey := os.Getenv("GEMINI_API_KEY")
	if apiKey == "" {
		return nil, fmt.Errorf("os.Getenv: GEMINI_API_KEY is required")
	}

	workDir, _ := os.Getwd()

	return &Agent{
		httpClient: &http.Client{},
		apiKey:     apiKey,
		workDir:    workDir,
	}, nil
}
