package file

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/pardnchiu/go-agent-skills/internal/utils"
)

type historyEntry struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

var historyTimeRanges = map[string]time.Duration{
	"1d": 24 * time.Hour,
	"7d": 7 * 24 * time.Hour,
	"1m": 30 * 24 * time.Hour,
	"1y": 365 * 24 * time.Hour,
}

func extractSec(content string) (int64, string) {
	if !strings.HasPrefix(content, "ts:") {
		return 0, content
	}
	rest := content[3:]
	idx := strings.IndexByte(rest, '\n')
	if idx < 0 {
		return 0, content
	}
	ts, err := strconv.ParseInt(rest[:idx], 10, 64)
	if err != nil {
		return 0, content
	}
	return ts, rest[idx+1:]
}

func searchHistory(sessionID, keyword, timeRange string) (string, error) {
	const limit = 10
	if keyword == "" {
		return "", fmt.Errorf("keyword is required")
	}
	if sessionID == "" {
		return "", fmt.Errorf("sessionID is required")
	}

	configDir, err := utils.GetConfigDir("sessions")
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

	var after int64
	if d, ok := historyTimeRanges[timeRange]; ok {
		after = time.Now().Add(-d).Unix()
	}

	lower := strings.ToLower(keyword)
	var matches []historyEntry

	for i := len(entries) - 5; i >= 0; i-- {
		entry := entries[i]
		ts, body := extractSec(entry.Content)
		if after > 0 && ts > 0 && ts < after {
			continue
		}
		if strings.Contains(strings.ToLower(body), lower) {
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
