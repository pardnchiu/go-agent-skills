package file

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/pardnchiu/go-agent-skills/internal/utils"
)

type historyEntry struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

func searchHistory(sessionID, keyword string, limit int) (string, error) {
	if keyword == "" {
		return "", fmt.Errorf("keyword is required")
	}
	if sessionID == "" {
		return "", fmt.Errorf("sessionID is required")
	}
	if limit <= 0 {
		limit = 10
	}

	configDir, err := utils.ConfigDir("sessions")
	if err != nil {
		return "", fmt.Errorf("utils.ConfigDir: %w", err)
	}

	historyPath := filepath.Join(configDir.Home, sessionID, "history.json")

	data, err := os.ReadFile(historyPath)
	if err != nil {
		if os.IsNotExist(err) {
			return "No history found for current session", nil
		}
		return "", fmt.Errorf("failed to read history file (%s): %w", historyPath, err)
	}

	var entries []historyEntry
	if err := json.Unmarshal(data, &entries); err != nil {
		return "", fmt.Errorf("failed to parse history file: %w", err)
	}

	lower := strings.ToLower(keyword)
	var matches []historyEntry

	for i := len(entries) - 1; i >= 0; i-- {
		entry := entries[i]
		if strings.Contains(strings.ToLower(entry.Content), lower) {
			matches = append(matches, entry)
			if len(matches) >= limit {
				break
			}
		}
	}

	if len(matches) == 0 {
		return fmt.Sprintf("No matches found for keyword: %s", keyword), nil
	}

	var result strings.Builder
	for _, m := range matches {
		result.WriteString(fmt.Sprintf("[%s] %s\n", m.Role, m.Content))
	}
	return result.String(), nil
}
