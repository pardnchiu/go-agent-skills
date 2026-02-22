package copilot

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

type Token struct {
	AccessToken string    `json:"access_token"`
	TokenType   string    `json:"token_type"`
	Scope       string    `json:"scope"`
	ExpiresAt   time.Time `json:"expires_at"`
}

type Agent struct {
	httpClient *http.Client
	Token      *Token
	Refresh    *RefreshToken
	workDir    string
	tokenDir   string
}

func New() (*Agent, error) {
	workDir, err := os.Getwd()
	if err != nil {
		return nil, fmt.Errorf("os.Getwd: %w", err)
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		return nil, fmt.Errorf("os.UserHomeDir: %w", err)
	}

	agent := &Agent{
		httpClient: &http.Client{},
		workDir:    workDir,
		tokenDir:   filepath.Join(homeDir, ".config", "go-agent-skills", "copilot_token.json"),
	}

	var token *Token

	data, err := os.ReadFile(agent.tokenDir)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			// * if is not exist, then login, github copilot code expire in 900s
			ctx, cancel := context.WithTimeout(context.Background(), 15*time.Minute)
			defer cancel()

			token, err = agent.Login(ctx)
			if err != nil {
				return nil, fmt.Errorf("agent.Login: %w", err)
			}
			agent.Token = token
			return agent, nil
		}
		return nil, fmt.Errorf("os.ReadFile: %w", err)
	}

	if err := json.Unmarshal(data, &token); err != nil {
		return nil, fmt.Errorf("json.Unmarshal: %w", err)
	}
	agent.Token = token

	return agent, nil
}
