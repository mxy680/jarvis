package voice

import (
	"context"
	"fmt"
	"os/exec"
)

// TTS provides text-to-speech functionality using macOS `say` command
type TTS struct {
	voice string
}

// NewTTS creates a new TTS instance with the specified voice
func NewTTS(voice string) *TTS {
	if voice == "" {
		voice = "Samantha"
	}
	return &TTS{voice: voice}
}

// Speak converts text to speech using the macOS say command
func (t *TTS) Speak(ctx context.Context, text string) error {
	if text == "" {
		return nil
	}

	cmd := exec.CommandContext(ctx, "say", "-v", t.voice, text)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("TTS failed: %w", err)
	}
	return nil
}

// SpeakAsync starts speaking in a goroutine and returns immediately
// Returns a channel that receives nil on success or an error on failure
func (t *TTS) SpeakAsync(ctx context.Context, text string) <-chan error {
	errCh := make(chan error, 1)
	go func() {
		errCh <- t.Speak(ctx, text)
		close(errCh)
	}()
	return errCh
}

// Voice returns the current voice name
func (t *TTS) Voice() string {
	return t.voice
}

// SetVoice changes the voice
func (t *TTS) SetVoice(voice string) {
	t.voice = voice
}

// ListVoices returns available voices on the system
func ListVoices(ctx context.Context) ([]string, error) {
	cmd := exec.CommandContext(ctx, "say", "-v", "?")
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to list voices: %w", err)
	}
	// Parse output - each line starts with voice name followed by locale
	// e.g., "Samantha             en_US    # Hi, my name is Samantha."
	var voices []string
	lines := splitLines(string(output))
	for _, line := range lines {
		if len(line) > 0 {
			// Extract voice name (first field, space-delimited)
			name := extractVoiceName(line)
			if name != "" {
				voices = append(voices, name)
			}
		}
	}
	return voices, nil
}

// splitLines splits text into lines
func splitLines(text string) []string {
	var lines []string
	start := 0
	for i, c := range text {
		if c == '\n' {
			lines = append(lines, text[start:i])
			start = i + 1
		}
	}
	if start < len(text) {
		lines = append(lines, text[start:])
	}
	return lines
}

// extractVoiceName extracts the voice name from a say -v ? output line
func extractVoiceName(line string) string {
	// Voice name ends at first group of spaces
	var name []byte
	inName := true
	for i := 0; i < len(line); i++ {
		if line[i] == ' ' {
			if inName && len(name) > 0 {
				break
			}
		} else {
			inName = true
			name = append(name, line[i])
		}
	}
	return string(name)
}
