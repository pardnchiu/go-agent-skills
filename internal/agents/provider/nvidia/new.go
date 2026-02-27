package nvidia

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
	// openai/gpt-oss-120b
	// z-ai/glm4.7
	// qwen/qwen3-235b-a22b
	// qwen/qwen3-coder-480b-a35b-instruct
	defaultModel = "openai/gpt-oss-120b"
	prefix       = "nvidia@"
)

func New(model ...string) (*Agent, error) {
	if len(model) > 0 && strings.HasPrefix(model[0], prefix) {
		defaultModel = strings.TrimPrefix(model[0], prefix)
	}
	apiKey := os.Getenv("NVIDIA_API_KEY")
	if apiKey == "" {
		return nil, fmt.Errorf("os.Getenv: NVIDIA_API_KEY is required")
	}

	workDir, _ := os.Getwd()

	return &Agent{
		httpClient: &http.Client{},
		apiKey:     apiKey,
		workDir:    workDir,
	}, nil
}
