package file

import (
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"

	"github.com/pardnchiu/go-agent-skills/internal/tools/model"
)

// * just fit one level of glob pattern
// TODO: need to do more work to support complex glob patterns
func GlobFiles(e *model.Executor, pattern string) (string, error) {
	var result strings.Builder
	err := filepath.WalkDir(e.WorkPath, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			slog.Warn("failed to access path",
				slog.String("error", err.Error()))
			return nil
		}

		if isExclude(e, path) {
			if d.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		if d.IsDir() && strings.HasPrefix(d.Name(), ".") && path != e.WorkPath {
			return filepath.SkipDir
		}

		if d.IsDir() {
			return nil
		}

		relPath, err := filepath.Rel(e.WorkPath, path)
		if err != nil {
			slog.Warn("failed to get relative path",
				slog.String("error", err.Error()))
			return nil
		}

		matched, err := filepath.Match(pattern, relPath)
		if err != nil {
			slog.Warn("failed to match pattern",
				slog.String("error", err.Error()))
			return nil
		}
		if matched {
			result.WriteString(relPath + "\n")
			return nil
		}

		if strings.Contains(pattern, "**") {
			parts := strings.SplitN(pattern, "**", 2)
			prefix := parts[0]
			suffix := strings.TrimPrefix(parts[1], "/")
			if strings.HasPrefix(relPath, prefix) {
				rest := relPath[len(prefix):]
				if suffix == "" {
					result.WriteString(relPath + "\n")
				} else if matched, _ := filepath.Match(suffix, filepath.Base(rest)); matched {
					result.WriteString(relPath + "\n")
				}
			}
		}
		return nil
	})
	if err != nil {
		return "", fmt.Errorf("failed to walk directory (%s): %w", pattern, err)
	}

	if result.Len() == 0 {
		return fmt.Sprintf("No fils found: %s", pattern), nil
	}
	return result.String(), nil
}
