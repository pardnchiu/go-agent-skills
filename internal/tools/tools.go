package tools

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/pardnchiu/go-agent-skills/internal/tools/types"
)

var (
// * template allow all for testing
// disallowed = regexp.MustCompile(`[;&|` + "`" + `$(){}!<>\\]`)
)

func runCommand(ctx context.Context, e *types.Executor, command string) (string, error) {
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
			return moveToTrash(ctx, e, args[1:])
		}
	}

	// TODO: need to change to dynamic timeout based on command complexity
	ctx, cancel := context.WithTimeout(ctx, 300*time.Second)
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

func moveToTrash(ctx context.Context, e *types.Executor, args []string) (string, error) {
	trashPath := filepath.Join(e.WorkPath, ".Trash")
	if err := os.MkdirAll(trashPath, 0755); err != nil {
		return "", fmt.Errorf("os.MkdirAll .Trash: %w", err)
	}

	var moved []string
	for _, arg := range args {
		if err := ctx.Err(); err != nil {
			return "", fmt.Errorf("moveToTrash cancelled: %w", err)
		}
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
