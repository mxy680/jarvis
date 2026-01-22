package voice

import (
	"bytes"
	"context"
	"fmt"
	"os/exec"
	"strings"
	"time"
)

// STT provides speech-to-text functionality using macOS SFSpeechRecognizer
type STT struct {
	timeout time.Duration
	binPath string // Path to the JarvisListen binary
}

// NewSTT creates a new STT instance
func NewSTT(timeoutSeconds int, binPath string) *STT {
	if timeoutSeconds <= 0 {
		timeoutSeconds = 30
	}
	if binPath == "" {
		binPath = "JarvisListen" // Assume it's in PATH
	}
	return &STT{
		timeout: time.Duration(timeoutSeconds) * time.Second,
		binPath: binPath,
	}
}

// Listen starts listening for speech and returns the transcribed text
func (s *STT) Listen(ctx context.Context) (string, error) {
	// Create timeout context
	ctx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, s.binPath)
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return "", fmt.Errorf("speech recognition timed out after %v", s.timeout)
		}
		if ctx.Err() == context.Canceled {
			return "", fmt.Errorf("speech recognition canceled")
		}
		return "", fmt.Errorf("speech recognition failed: %w (stderr: %s)", err, stderr.String())
	}

	// Trim whitespace from output
	text := strings.TrimSpace(stdout.String())
	return text, nil
}

// ListenWithCallback starts listening and calls the callback with partial results
// This requires the Swift binary to support streaming output
func (s *STT) ListenWithCallback(ctx context.Context, onPartial func(string)) (string, error) {
	ctx, cancel := context.WithTimeout(ctx, s.timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, s.binPath, "--stream")
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return "", fmt.Errorf("failed to create stdout pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return "", fmt.Errorf("failed to start speech recognition: %w", err)
	}

	// Read partial results
	var finalText string
	buf := make([]byte, 1024)
	for {
		n, readErr := stdout.Read(buf)
		if n > 0 {
			text := strings.TrimSpace(string(buf[:n]))
			if text != "" {
				finalText = text
				if onPartial != nil {
					onPartial(text)
				}
			}
		}
		if readErr != nil {
			break
		}
	}

	if err := cmd.Wait(); err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return finalText, fmt.Errorf("speech recognition timed out after %v", s.timeout)
		}
		if ctx.Err() == context.Canceled {
			return finalText, nil // Canceled is normal when we have final text
		}
		return finalText, fmt.Errorf("speech recognition failed: %w (stderr: %s)", err, stderr.String())
	}

	return finalText, nil
}

// Timeout returns the configured timeout duration
func (s *STT) Timeout() time.Duration {
	return s.timeout
}

// SetTimeout updates the timeout duration
func (s *STT) SetTimeout(timeoutSeconds int) {
	if timeoutSeconds > 0 {
		s.timeout = time.Duration(timeoutSeconds) * time.Second
	}
}

// BinPath returns the path to the speech recognition binary
func (s *STT) BinPath() string {
	return s.binPath
}
