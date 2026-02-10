package file

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/pardnchiu/go-agent-skills/internal/tools/model"
)

func getFullPath(e *model.Executor, path string) string {
	if filepath.IsAbs(path) {
		return path
	}
	return filepath.Join(e.WorkPath, path)
}

func isExclude(e *model.Executor, path string) bool {
	for _, pattern := range e.Exclude {
		matched, _ := filepath.Match(pattern, filepath.Base(path))
		if matched {
			return true
		}

		if strings.Contains(path, "/"+pattern+"/") ||
			strings.HasPrefix(path, pattern+"/") {
			return true
		}
	}
	return false
}

func ReadFile(e *model.Executor, path string) (string, error) {
	fullPath := getFullPath(e, path)

	if isExclude(e, fullPath) {
		return "", fmt.Errorf("path is excluded: %s", path)
	}

	data, err := os.ReadFile(fullPath)
	if err != nil {
		return "", fmt.Errorf("failed to read file (%s): %w", path, err)
	}
	return string(data), nil
}
