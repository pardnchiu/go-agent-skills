package exec

import (
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"time"

	atypes "github.com/pardnchiu/go-agent-skills/internal/agents/types"
	"github.com/pardnchiu/go-agent-skills/internal/utils"
)

type IndexData struct {
	SessionID string `json:"session_id"`
}

func getSession(prompt string, userInput string) (*atypes.AgentSession, string, error) {
	now := fmt.Sprintf("%d", time.Now().Unix())
	input := atypes.AgentSession{
		Tools: []atypes.Message{},
		Messages: []atypes.Message{
			{Role: "system", Content: prompt},
		},
		Histories: []atypes.Message{},
	}

	configDir, err := utils.GetConfigDir("sessions")
	if err != nil {
		fmt.Printf("utils.ConfigDir: %v\n", err)
		return nil, "", err
	}

	indexJSONPath := filepath.Join(configDir.Home, "..", "config.json")

	unlock, err := lockConfig(filepath.Dir(indexJSONPath))
	if err != nil {
		return nil, "", fmt.Errorf("lockConfig: %w", err)
	}
	defer unlock()

	var sessionID string
	data, readErr := os.ReadFile(indexJSONPath)
	switch {
	case readErr == nil:
		var indexData IndexData
		if err := json.Unmarshal(data, &indexData); err != nil || indexData.SessionID == "" {
			return nil, "", fmt.Errorf("config.json corrupted: %w", err)
		}
		sessionID = indexData.SessionID

		var summary string
		if summaryData, err := os.ReadFile(filepath.Join(configDir.Home, sessionID, "summary.json")); err == nil {
			summary = strings.NewReplacer(
				"{{.Summary}}", string(summaryData),
			).Replace(summaryPrompt)
		}

		if histData, err := os.ReadFile(filepath.Join(configDir.Home, sessionID, "history.json")); err == nil {
			var oldHistory []atypes.Message
			if err := json.Unmarshal(histData, &oldHistory); err == nil {
				input.Histories = oldHistory
			}
			input.Histories = append(input.Histories, atypes.Message{Role: "user", Content: fmt.Sprintf("ts:%s\n%s", now, userInput)})

			input.Messages = append(input.Messages, atypes.Message{Role: "system", Content: summary})
			recentHistory := oldHistory
			if len(recentHistory) > 4 {
				recentHistory = recentHistory[len(recentHistory)-4:]
			}
			input.Messages = append(input.Messages, recentHistory...)
			input.Messages = append(input.Messages, atypes.Message{Role: "user", Content: fmt.Sprintf("ts:%s\n%s", now, userInput)})
		}

	case os.IsNotExist(readErr):
		var genErr error
		sessionID, genErr = newSessionID()
		if genErr != nil {
			return nil, "", fmt.Errorf("newSessionID: %w", genErr)
		}

		input.Histories = append(input.Histories, atypes.Message{Role: "user", Content: fmt.Sprintf("ts:%s\n%s", now, userInput)})
		input.Messages = append(input.Messages, atypes.Message{Role: "user", Content: fmt.Sprintf("ts:%s\n%s", now, userInput)})

		indexDataBytes, err := json.Marshal(IndexData{SessionID: sessionID})
		if err != nil {
			return nil, "", fmt.Errorf("json.Marshal: %w", err)
		}
		// O_EXCL: flock 외 추가 방어층 — 파일이 생성된 경우 원자적 실패
		f, err := os.OpenFile(indexJSONPath, os.O_CREATE|os.O_EXCL|os.O_WRONLY, 0644)
		if err != nil {
			return nil, "", fmt.Errorf("os.OpenFile config.json: %w", err)
		}
		_, writeErr := f.Write(indexDataBytes)
		closeErr := f.Close()
		if writeErr != nil {
			return nil, "", fmt.Errorf("write config.json: %w", writeErr)
		}
		if closeErr != nil {
			return nil, "", fmt.Errorf("close config.json: %w", closeErr)
		}

	default:
		return nil, "", fmt.Errorf("os.ReadFile config.json: %w", readErr)
	}

	err = os.MkdirAll(filepath.Join(configDir.Home, sessionID), 0755)
	if err != nil {
		return nil, "", fmt.Errorf("os.MkdirAll: %w", err)
	}

	return &input, sessionID, nil
}

func lockConfig(dir string) (func(), error) {
	lockPath := filepath.Join(dir, "config.json.lock")
	f, err := os.OpenFile(lockPath, os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		return nil, fmt.Errorf("os.OpenFile: %w", err)
	}
	if err := syscall.Flock(int(f.Fd()), syscall.LOCK_EX); err != nil {
		f.Close()
		return nil, fmt.Errorf("syscall.Flock: %w", err)
	}
	return func() {
		_ = syscall.Flock(int(f.Fd()), syscall.LOCK_UN)
		f.Close()
	}, nil
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
