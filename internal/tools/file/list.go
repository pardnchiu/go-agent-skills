package file

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/pardnchiu/go-agent-skills/internal/tools/types"
)

func list(e *types.Executor, path string, recursive bool) (string, error) {
	fullPath := getFullPath(e, path)

	var files []string
	var err error
	if recursive {
		files, err = walkFiles(e, fullPath)
	} else {
		files, err = listDir(e, fullPath)
	}
	if err != nil {
		return "", fmt.Errorf("list files — %w", err)
	}
	return strings.Join(files, "\n") + "\n", nil
}

func walkFiles(e *types.Executor, root string) ([]string, error) {
	var files []string
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			slog.Warn("failed to access path, just skipping",
				slog.String("error", err.Error()))
			return nil
		}

		if isExclude(e, path) {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		rel, err := filepath.Rel(root, path)
		if err != nil {
			slog.Warn("failed to get relative path, just skipping",
				slog.String("error", err.Error()))
			return nil
		}
		if rel == "." {
			return nil
		}

		if d.IsDir() {
			if strings.HasPrefix(d.Name(), ".") {
				return filepath.SkipDir
			}
			return nil
		}

		files = append(files, filepath.ToSlash(rel))
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("walk files — %w", err)
	}
	return files, nil
}

func listDir(e *types.Executor, path string) ([]string, error) {
	entries, err := os.ReadDir(path)
	if err != nil {
		return nil, fmt.Errorf("list directory — %w", err)
	}

	var files []string
	for _, entry := range entries {
		newPath := filepath.Join(path, entry.Name())
		if isExclude(e, newPath) {
			continue
		}

		if entry.IsDir() {
			files = append(files, entry.Name()+"/")
		} else {
			files = append(files, entry.Name())
		}
	}
	return files, nil
}
