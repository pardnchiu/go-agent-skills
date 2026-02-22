package nvidia

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
