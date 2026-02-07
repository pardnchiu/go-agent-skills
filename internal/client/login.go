package client

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"time"

	"github.com/pardnchiu/go-agent-skills/internal/utils"
)

var (
	CopilotDefaultModel       = "gpt-4.1"
	CopilotDefaultTokens      = 16384
	CopilotDefaultTimeout     = 5 * time.Minute
	GitHubDeviceCodeAPI       = "https://github.com/login/device/code"
	GitHubOauthAccessTokenAPI = "https://github.com/login/oauth/access_token"
	CopilotTokenURL           = "https://api.github.com/copilot_internal/v2/token"
	CopilotClientID           = "Iv1.b507a08c87ecfe98"
	MaxToolIterations         = 20
)

var ErrAuthorizationPending = fmt.Errorf("authorization pending") // pre declare error for ensuring padding wont cause login exit

type GopilotDeviceCode struct {
	DeviceCode      string `json:"device_code"`
	UserCode        string `json:"user_code"`
	VerificationURI string `json:"verification_uri"`
	ExpiresIn       int    `json:"expires_in"`
	Interval        int    `json:"interval"`
}

func CopilotLogin(ctx context.Context, tokenPath string) (*CopilotToken, error) {
	code, _, err := utils.POSTForm[GopilotDeviceCode](ctx, nil, GitHubDeviceCodeAPI,
		map[string]string{},
		url.Values{
			"client_id": {CopilotClientID},
		})

	expires := time.Now().Add(time.Duration(code.ExpiresIn) * time.Second)
	fmt.Printf("[*] url:      %-42s\n", code.VerificationURI)
	fmt.Printf("[*] code:     %-42s\n", code.UserCode)
	fmt.Printf("[*] expires:  %-36s\n", expires.Format("2006-01-02 15:04:05"))
	fmt.Print("[*] press Enter to open browser")

	go func() {
		var input string
		fmt.Scanln(&input)

		url := code.VerificationURI

		var cmd *exec.Cmd
		switch runtime.GOOS {
		case "darwin":
			cmd = exec.Command("open", url)
		case "windows":
			cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
		case "linux":
			cmd = exec.Command("xdg-open", url)
		default:
			fmt.Printf("[!] can not open browser, please open: %-48s\n", url)
		}

		if cmd != nil {
			cmd.Start()
		}
	}()

	interval := time.Duration(code.Interval) * time.Second
	deadline := time.Now().Add(time.Duration(code.ExpiresIn) * time.Second)

	var token *CopilotToken
	client := &http.Client{} // * use the same http client for reuse connection
	for time.Now().Before(deadline) {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(interval):
		}

		token, err = getAccessToken(ctx, client, code.DeviceCode)
		if err != nil {
			if errors.Is(err, ErrAuthorizationPending) {
				continue
			}
			return nil, err
		}

		path := filepath.Dir(tokenPath)
		if err := os.MkdirAll(path, 0700); err != nil {
			return nil, err
		}

		data, err := json.MarshalIndent(token, "", "  ")
		if err != nil {
			return nil, err
		}

		os.WriteFile(tokenPath, data, 0600)

		return token, nil
	}
	return nil, fmt.Errorf("device code expired")
}

type GopilotAccessToken struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	Scope       string `json:"scope"`
	Error       string `json:"error"`
}

func getAccessToken(ctx context.Context, client *http.Client, deviceCode string) (*CopilotToken, error) {
	accessToken, _, err := utils.POSTForm[GopilotAccessToken](ctx, client, GitHubOauthAccessTokenAPI,
		map[string]string{},
		url.Values{
			"client_id":   {CopilotClientID},
			"device_code": {deviceCode},
			"grant_type":  {"urn:ietf:params:oauth:grant-type:device_code"},
		})
	if err != nil {
		return nil, err
	}

	switch accessToken.Error {
	case "":
		return &CopilotToken{
			AccessToken: accessToken.AccessToken,
			TokenType:   accessToken.TokenType,
			Scope:       accessToken.Scope,
		}, nil
	case "authorization_pending":
		return nil, ErrAuthorizationPending
	default:
		return nil, fmt.Errorf("oauth error: %s", accessToken.Error)
	}
}
