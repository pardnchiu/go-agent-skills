package main

import (
	"context"
	"fmt"
	"log/slog"
	"os"

	c "github.com/pardnchiu/go-agent-skills/internal/client"
)

func main() {
	client, err := c.NewCopilot()
	if err != nil {
		slog.Error("failed to load Copilot token",
			slog.String("error", err.Error()))
		os.Exit(1)
	}

	slog.Info("successfully loaded Copilot token",
		slog.String("access_token", client.Token.AccessToken),
		slog.String("token_type", client.Token.TokenType),
		slog.String("scope", client.Token.Scope),
		slog.Time("expires_at", client.Token.ExpiresAt))

	if len(os.Args) < 3 || os.Args[1] != "input" {
		slog.Error("usage: go run cmd/cli/main.go input \"your message\"")
		os.Exit(1)
	}

	userInput := os.Args[2]
	ctx := context.Background()
	messages := []c.Message{
		{
			Role:    "system",
			Content: "You are a helpful assistant.",
		},
		{
			Role:    "user",
			Content: userInput,
		},
	}

	resp, err := client.SendChat(ctx, messages, nil)
	if err != nil {
		slog.Error("failed to send chat",
			slog.String("error", err.Error()))
		os.Exit(1)
	}

	if len(resp.Choices) > 0 {
		choice := resp.Choices[0]
		if content, ok := choice.Message.Content.(string); ok {
			fmt.Println("Response:", content)
		}
	}
}
