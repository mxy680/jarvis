package terminal

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
)

// TerminalAppInjector injects text into macOS Terminal.app
type TerminalAppInjector struct{}

// InjectText types text into a Terminal.app tab identified by TTY
func (t *TerminalAppInjector) InjectText(ctx context.Context, tty string, text string) error {
	if tty == "" {
		return fmt.Errorf("TTY is required")
	}
	if text == "" {
		return nil // Nothing to inject
	}

	script := buildTerminalAppScript(tty, text)

	cmd := exec.CommandContext(ctx, "osascript", "-e", script)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return fmt.Errorf("AppleScript execution failed: %w (stderr: %s)", err, stderr.String())
	}

	result := strings.TrimSpace(stdout.String())
	if result == "tty_not_found" {
		return fmt.Errorf("TTY %s not found in Terminal.app", tty)
	}

	return nil
}
