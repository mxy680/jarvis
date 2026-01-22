package main

import (
	"encoding/json"
	"fmt"
	"net"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/google/uuid"
	"github.com/markshteyn/jarvis/internal/daemon"
)

func main() {
	if len(os.Args) < 2 {
		fmt.Fprintln(os.Stderr, "Usage: jarvis-notify <transcript_path>")
		os.Exit(1)
	}

	transcriptPath := os.Args[1]

	// Validate transcript path exists
	if _, err := os.Stat(transcriptPath); os.IsNotExist(err) {
		fmt.Fprintf(os.Stderr, "Transcript file not found: %s\n", transcriptPath)
		os.Exit(1)
	}

	// Get or generate session ID
	sessionID := getOrCreateSessionID()

	// Detect TTY
	tty := detectTTY()

	// Detect terminal app
	terminalApp := detectTerminalApp()

	// Build payload
	payload := daemon.NotifyPayload{
		TranscriptPath: transcriptPath,
		SessionID:      sessionID,
		TTY:            tty,
		TerminalApp:    terminalApp,
	}

	// Send to daemon
	if err := sendNotification(payload); err != nil {
		fmt.Fprintf(os.Stderr, "Failed to notify daemon: %v\n", err)
		os.Exit(1)
	}
}

// getOrCreateSessionID returns a persistent session ID for this terminal
// Stores it in an environment variable via shell if possible
func getOrCreateSessionID() string {
	// Check if we already have a session ID in the environment
	if id := os.Getenv("JARVIS_SESSION_ID"); id != "" {
		return id
	}

	// Generate a new UUID
	return uuid.New().String()
}

// detectTTY returns the TTY path for the current terminal
func detectTTY() string {
	// Try the `tty` command first
	cmd := exec.Command("tty")
	output, err := cmd.Output()
	if err == nil {
		tty := strings.TrimSpace(string(output))
		if tty != "" && tty != "not a tty" {
			return tty
		}
	}

	// Fallback: check common TTY environment variables
	if tty := os.Getenv("TTY"); tty != "" {
		return tty
	}

	// Fallback: try to read from /dev/tty
	if fi, err := os.Stat("/dev/tty"); err == nil && fi.Mode()&os.ModeCharDevice != 0 {
		// Try to resolve the actual TTY
		link, err := os.Readlink("/dev/fd/0")
		if err == nil && strings.HasPrefix(link, "/dev/ttys") {
			return link
		}
	}

	return ""
}

// detectTerminalApp returns the terminal application name
func detectTerminalApp() string {
	// TERM_PROGRAM is set by most terminal emulators
	if termProgram := os.Getenv("TERM_PROGRAM"); termProgram != "" {
		return termProgram
	}

	// LC_TERMINAL is set by iTerm2
	if lcTerminal := os.Getenv("LC_TERMINAL"); lcTerminal != "" {
		return lcTerminal
	}

	// Default to Terminal.app
	return "Apple_Terminal"
}

// sendNotification sends the payload to the jarvis daemon
func sendNotification(payload daemon.NotifyPayload) error {
	socketPath := getSocketPath()

	conn, err := net.Dial("unix", socketPath)
	if err != nil {
		return fmt.Errorf("failed to connect to daemon at %s: %w", socketPath, err)
	}
	defer conn.Close()

	// Send payload
	if err := json.NewEncoder(conn).Encode(payload); err != nil {
		return fmt.Errorf("failed to send payload: %w", err)
	}

	// Read response
	var response struct {
		Success bool   `json:"success"`
		Message string `json:"message"`
	}
	if err := json.NewDecoder(conn).Decode(&response); err != nil {
		return fmt.Errorf("failed to read response: %w", err)
	}

	if !response.Success {
		return fmt.Errorf("daemon error: %s", response.Message)
	}

	return nil
}

// getSocketPath returns the path to the daemon socket
func getSocketPath() string {
	// Check environment variable first
	if path := os.Getenv("JARVIS_SOCKET"); path != "" {
		return path
	}

	// Default path
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "/tmp/jarvis.sock"
	}
	return filepath.Join(homeDir, ".jarvis", "jarvis.sock")
}
