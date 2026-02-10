package file

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/pardnchiu/go-agent-skills/internal/tools/model"
)

func WriteFile(e *model.Executor, path, content string) (string, error) {
	fullPath := getFullPath(e, path)

	dir := filepath.Dir(fullPath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return "", fmt.Errorf("failed to create directory (%s): %w", path, err)
	}

	if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
		return "", fmt.Errorf("failed to write file (%s): %w", path, err)
	}

	return fmt.Sprintf("Successfully wrote file: %s", path), nil
}
