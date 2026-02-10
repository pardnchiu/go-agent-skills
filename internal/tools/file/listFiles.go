package file

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/pardnchiu/go-agent-skills/internal/tools/model"
)

func ListFiles(e *model.Executor, path string, recursive bool) (string, error) {
	fullPath := getFullPath(e, path)

	var result strings.Builder
	if recursive {
		err := filepath.Walk(fullPath, func(p string, info os.FileInfo, err error) error {
			if err != nil {
				slog.Warn("failed to access path",
					slog.String("error", err.Error()))
				return nil
			}

			if isExclude(e, p) {
				if info.IsDir() {
					return filepath.SkipDir
				}
				return nil
			}

			relPath, err := filepath.Rel(fullPath, p)
			if err != nil {
				slog.Warn("failed to get relative path",
					slog.String("error", err.Error()))
				return nil
			}
			if relPath == "." {
				return nil
			}
			if strings.HasPrefix(filepath.Base(p), ".") && info.IsDir() {
				return filepath.SkipDir
			}
			if info.IsDir() {
				result.WriteString(relPath + "/\n")
			} else {
				result.WriteString(relPath + "\n")
			}
			return nil
		})
		if err != nil {
			return "", fmt.Errorf("failed to walk directory (%s): %w", path, err)
		}
	} else {
		entries, err := os.ReadDir(fullPath)
		if err != nil {
			return "", fmt.Errorf("failed to read directory (%s): %w", path, err)
		}
		for _, entry := range entries {
			entryPath := filepath.Join(fullPath, entry.Name())
			if isExclude(e, entryPath) {
				continue
			}

			if entry.IsDir() {
				result.WriteString(entry.Name() + "/\n")
			} else {
				result.WriteString(entry.Name() + "\n")
			}
		}
	}

	return result.String(), nil
}
