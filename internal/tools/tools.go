package tools

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/pardnchiu/go-agent-skills/internal/tools/model"
)

var (
// * template allow all for testing
// disallowed = regexp.MustCompile(`[;&|` + "`" + `$(){}!<>\\]`)
)

func runCommand(e *model.Executor, command string) (string, error) {
	command = strings.TrimSpace(command)
	if command == "" {
		return "", fmt.Errorf("failed to run command: command is empty")
	}

	// * template allow all for testing
	// if disallowed.MatchString(command) {
	// 	return "", fmt.Errorf("failed to run command: disallowed characters")
	// }

	hasShellOps := strings.ContainsAny(command, "|><&")

	var binary string
	var args []string

	if hasShellOps {
		binary = "sh"
		args = []string{"-c", command}

		firstCmd := strings.Fields(command)[0]
		if !e.AllowedCommand[filepath.Base(firstCmd)] {
			return "", fmt.Errorf("failed to run command: %s is not allowed", firstCmd)
		}
	} else {
		args = strings.Fields(command)
		binary = filepath.Base(args[0])

		if !e.AllowedCommand[binary] {
			return "", fmt.Errorf("failed to run command: %s is not allowed", binary)
		}

		if binary == "rm" {
			return moveToTrash(e, args[1:])
		}
	}

	// TODO: need to change to dynamic timeout based on command complexity
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	var cmd *exec.Cmd
	if hasShellOps {
		cmd = exec.CommandContext(ctx, binary, args...)
	} else {
		cmd = exec.CommandContext(ctx, args[0], args[1:]...)
	}
	cmd.Dir = e.WorkPath

	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Sprintf("%s\nError: %s", string(output), err.Error()), nil
	}

	return string(output), nil
}

func moveToTrash(e *model.Executor, args []string) (string, error) {
	trashPath := filepath.Join(e.WorkPath, ".Trash")
	os.MkdirAll(trashPath, 0755)

	var moved []string
	for _, arg := range args {
		if strings.HasPrefix(arg, "-") {
			continue
		}
		src := filepath.Join(e.WorkPath, filepath.Clean(arg))
		name := filepath.Base(arg)
		dst := filepath.Join(trashPath, name)

		if _, err := os.Stat(dst); err == nil {
			ext := filepath.Ext(name)
			dst = filepath.Join(trashPath, fmt.Sprintf("%s_%s%s",
				strings.TrimSuffix(name, ext),
				time.Now().Format("20060102_150405"),
				ext))
		}

		if err := os.Rename(src, dst); err == nil {
			moved = append(moved, arg)
		}
	}
	return fmt.Sprintf("Successfully moved to .Trash: %s", strings.Join(moved, ", ")), nil
}

func searchContent(e *model.Executor, pattern, filePattern string) (string, error) {
	re, err := regexp.Compile(pattern)
	if err != nil {
		return "", fmt.Errorf("failed to compile regex pattern (%s): %w", pattern, err)
	}

	var result strings.Builder

	err = filepath.Walk(e.WorkPath, func(path string, d os.FileInfo, err error) error {
		if err != nil {
			slog.Warn("failed to access path",
				slog.String("error", err.Error()))
			return nil
		}

		if strings.HasPrefix(filepath.Base(path), ".") {
			return nil
		}

		if d.IsDir() {
			return nil
		}

		ext := filepath.Ext(path)
		exts := map[string]bool{
			".exe":   true,
			".bin":   true,
			".so":    true,
			".dylib": true,
			".dll":   true,
			".o":     true,
			".a":     true,
		}
		if exts[ext] {
			return nil
		}

		if filePattern != "" {
			matched, err := filepath.Match(filePattern, filepath.Base(path))
			if err != nil {
				slog.Warn("failed to match pattern",
					slog.String("error", err.Error()))
			}
			if !matched {
				return nil
			}
		}

		data, err := os.ReadFile(path)
		if err != nil {
			slog.Warn("failed to read file during search",
				slog.String("error", err.Error()))
			return nil
		}

		lines := strings.Split(string(data), "\n")
		relPath, err := filepath.Rel(e.WorkPath, path)
		if err != nil {
			slog.Warn("failed to get relative path",
				slog.String("error", err.Error()))
			return nil
		}

		for i, line := range lines {
			if re.MatchString(line) {
				result.WriteString(fmt.Sprintf("%s:%d: %s\n", relPath, i+1, strings.TrimSpace(line)))
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
