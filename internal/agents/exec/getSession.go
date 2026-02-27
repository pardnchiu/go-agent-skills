package exec

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/pardnchiu/go-agent-skills/internal/utils"
)

type SessionData struct {
	tools     []Message
	messages  []Message
	histories []Message
}

type IndexData struct {
	SessionID string `json:"session_id"`
}

func getSession(prompt string, userInput string) (*SessionData, string, error) {
	now := time.Now().Format("2006-01-02T15:04:05 MST (UTC-07:00)")
	input := SessionData{
		tools: []Message{},
		messages: []Message{
			{Role: "system", Content: prompt},
		},
		histories: []Message{},
	}

	configDir, err := utils.GetConfigDir("sessions")
	if err != nil {
		fmt.Printf("utils.ConfigDir: %v\n", err)
		return nil, "", err
	}

	indexJSONPath := filepath.Join(configDir.Home, "..", "config.json")
	var sessionID string
	if data, err := os.ReadFile(indexJSONPath); err == nil {
		var indexData IndexData
		if err := json.Unmarshal(data, &indexData); err == nil {
			sessionID = indexData.SessionID

			var summary string
			data, err := os.ReadFile(filepath.Join(configDir.Home, sessionID, "summary.json"))
			if err == nil {
				summary = strings.NewReplacer(
					"{{.Summary}}", string(data),
				).Replace(summaryPrompt)
			}

			data, err = os.ReadFile(filepath.Join(configDir.Home, sessionID, "history.json"))
			if err == nil {
				var oldHistory []Message
				if err := json.Unmarshal(data, &oldHistory); err == nil {
					input.histories = oldHistory
				}
				input.histories = append(input.histories, Message{Role: "user", Content: fmt.Sprintf("當前時間：%s\n%s", now, userInput)})

				input.messages = append(input.messages, Message{Role: "system", Content: summary})
				recentHistory := oldHistory
				if len(recentHistory) > 4 {
					recentHistory = recentHistory[len(recentHistory)-4:]
				}
				input.messages = append(input.messages, recentHistory...)
				input.messages = append(input.messages, Message{Role: "user", Content: fmt.Sprintf("當前時間：%s\n%s", now, userInput)})
			}
		}
	} else {
		sessionID, err = newSessionID()
		if err != nil {
			return nil, "", fmt.Errorf("newSessionID: %w", err)
		}
		indexData := IndexData{SessionID: sessionID}

		input.histories = append(input.histories, Message{Role: "user", Content: fmt.Sprintf("當前時間：%s\n%s", now, userInput)})
		input.messages = append(input.messages, Message{Role: "user", Content: fmt.Sprintf("當前時間：%s\n%s", now, userInput)})

		indexDataBytes, err := json.Marshal(indexData)
		if err != nil {
			return nil, "", fmt.Errorf("json.Marshal: %w", err)
		}
		if err := os.WriteFile(indexJSONPath, indexDataBytes, 0644); err != nil {
			return nil, "", fmt.Errorf("os.WriteFile: %w", err)
		}
	}

	err = os.MkdirAll(filepath.Join(configDir.Home, sessionID), 0755)
	if err != nil {
		return nil, "", fmt.Errorf("os.MkdirAll: %w", err)
	}

	return &input, sessionID, nil
}

func newSessionID() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", fmt.Errorf("rand.Read: %w", err)
	}
	b[6] = (b[6] & 0x0f) | 0x40
	b[8] = (b[8] & 0x3f) | 0x80
	h := hex.EncodeToString(b)
	return h[0:8] + "-" + h[8:12] + "-" + h[12:16] + "-" + h[16:20] + "-" + h[20:], nil
}
