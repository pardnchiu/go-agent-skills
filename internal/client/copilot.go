package client

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"time"
)

type CopilotToken struct {
	AccessToken string    `json:"access_token"`
	TokenType   string    `json:"token_type"`
	Scope       string    `json:"scope"`
	ExpiresAt   time.Time `json:"expires_at"`
}

type CopilotAgent struct {
	httpClient *http.Client
	Token      *CopilotToken
	Refresh    *RefreshToken
	workPath   string
	tokenPath  string
}

func NewCopilot() (*CopilotAgent, error) {
	workDir, _ := os.Getwd()

	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}

	agent := &CopilotAgent{
		httpClient: &http.Client{},
		workPath:   workDir,
		tokenPath:  filepath.Join(home, ".config", "go-agent-skills", "copilot_token.json"),
	}

	var token *CopilotToken

	data, err := os.ReadFile(agent.tokenPath)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			// * if is not exist, then login
			ctx, cancel := context.WithTimeout(context.Background(), 15*time.Minute)
			defer cancel()

			token, err = CopilotLogin(ctx, agent.tokenPath)
			if err != nil {
				return nil, err
			}
			agent.Token = token
			return agent, nil
		}
		return nil, err
	}

	if err := json.Unmarshal(data, &token); err != nil {
		return nil, err
	}
	agent.Token = token

	agent.checkAndRefresnToken(context.Background())

	return agent, nil
}
