package voice

import (
	"context"
	"testing"
	"time"
)

func TestNewTTS(t *testing.T) {
	tests := []struct {
		name          string
		voice         string
		expectedVoice string
	}{
		{
			name:          "default voice when empty",
			voice:         "",
			expectedVoice: "Samantha",
		},
		{
			name:          "custom voice",
			voice:         "Alex",
			expectedVoice: "Alex",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tts := NewTTS(tt.voice)
			if tts.Voice() != tt.expectedVoice {
				t.Errorf("expected voice %q, got %q", tt.expectedVoice, tts.Voice())
			}
		})
	}
}

func TestTTS_SetVoice(t *testing.T) {
	tts := NewTTS("Samantha")
	tts.SetVoice("Alex")
	if tts.Voice() != "Alex" {
		t.Errorf("expected voice Alex, got %q", tts.Voice())
	}
}

func TestTTS_SpeakEmpty(t *testing.T) {
	tts := NewTTS("Samantha")
	ctx := context.Background()

	// Speaking empty string should succeed immediately
	err := tts.Speak(ctx, "")
	if err != nil {
		t.Errorf("expected no error for empty text, got %v", err)
	}
}

func TestTTS_SpeakContextCanceled(t *testing.T) {
	tts := NewTTS("Samantha")
	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	// With canceled context, speak should fail
	err := tts.Speak(ctx, "This should not be spoken")
	if err == nil {
		t.Error("expected error with canceled context")
	}
}

func TestTTS_SpeakAsync(t *testing.T) {
	tts := NewTTS("Samantha")
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	// SpeakAsync should return immediately
	errCh := tts.SpeakAsync(ctx, "")
	select {
	case err := <-errCh:
		if err != nil {
			t.Errorf("expected no error, got %v", err)
		}
	case <-time.After(1 * time.Second):
		t.Error("SpeakAsync did not complete in time")
	}
}

func TestExtractVoiceName(t *testing.T) {
	tests := []struct {
		line     string
		expected string
	}{
		{
			line:     "Samantha             en_US    # Hi, my name is Samantha.",
			expected: "Samantha",
		},
		{
			line:     "Alex                 en_US    # Most people recognize me by my voice.",
			expected: "Alex",
		},
		{
			line:     "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			got := extractVoiceName(tt.line)
			if got != tt.expected {
				t.Errorf("extractVoiceName(%q) = %q, want %q", tt.line, got, tt.expected)
			}
		})
	}
}

func TestSplitLines(t *testing.T) {
	tests := []struct {
		text     string
		expected []string
	}{
		{
			text:     "line1\nline2\nline3",
			expected: []string{"line1", "line2", "line3"},
		},
		{
			text:     "single",
			expected: []string{"single"},
		},
		{
			text:     "",
			expected: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.text, func(t *testing.T) {
			got := splitLines(tt.text)
			if len(got) != len(tt.expected) {
				t.Errorf("splitLines(%q) = %v, want %v", tt.text, got, tt.expected)
				return
			}
			for i := range got {
				if got[i] != tt.expected[i] {
					t.Errorf("splitLines(%q)[%d] = %q, want %q", tt.text, i, got[i], tt.expected[i])
				}
			}
		})
	}
}
