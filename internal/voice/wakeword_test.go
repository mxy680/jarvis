package voice

import (
	"testing"
)

func TestParseKeyword(t *testing.T) {
	tests := []struct {
		keyword     string
		expectError bool
	}{
		{"jarvis", false},
		{"Jarvis", false},
		{"JARVIS", false},
		{"alexa", false},
		{"computer", false},
		{"hey siri", false},
		{"ok google", false},
		{"unknown_keyword", true}, // Should error but default to jarvis
	}

	for _, tt := range tests {
		t.Run(tt.keyword, func(t *testing.T) {
			_, err := parseKeyword(tt.keyword)
			if tt.expectError && err == nil {
				t.Errorf("expected error for keyword %q", tt.keyword)
			}
			if !tt.expectError && err != nil {
				t.Errorf("unexpected error for keyword %q: %v", tt.keyword, err)
			}
		})
	}
}

func TestNewWakeWordDetector_NoAccessKey(t *testing.T) {
	_, err := NewWakeWordDetector("", "jarvis")
	if err == nil {
		t.Error("expected error for empty access key")
	}
}

func TestNewWakeWordDetector_ValidKeyword(t *testing.T) {
	// Use a fake access key - actual initialization will fail,
	// but construction should succeed
	detector, err := NewWakeWordDetector("fake_access_key", "jarvis")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if detector == nil {
		t.Error("expected non-nil detector")
	}
}

func TestWakeWordDetector_NotInitialized(t *testing.T) {
	detector, _ := NewWakeWordDetector("fake_key", "jarvis")

	// Process should fail when not initialized
	_, err := detector.Process(make([]int16, 512))
	if err == nil {
		t.Error("expected error when not initialized")
	}
}

func TestWakeWordDetector_FrameLength(t *testing.T) {
	detector, _ := NewWakeWordDetector("fake_key", "jarvis")
	fl := detector.FrameLength()
	if fl != PorcupineFrameLength {
		t.Errorf("expected frame length %d, got %d", PorcupineFrameLength, fl)
	}
}

func TestWakeWordDetector_SampleRate(t *testing.T) {
	detector, _ := NewWakeWordDetector("fake_key", "jarvis")
	sr := detector.SampleRate()
	if sr != PorcupineSampleRate {
		t.Errorf("expected sample rate %d, got %d", PorcupineSampleRate, sr)
	}
}

func TestWakeWordListener_NotRunning(t *testing.T) {
	detector, _ := NewWakeWordDetector("fake_key", "jarvis")
	listener := NewWakeWordListener(detector, func() {})

	if listener.IsRunning() {
		t.Error("listener should not be running initially")
	}

	// ProcessAudio should fail when not running
	err := listener.ProcessAudio(make([]int16, 512))
	if err == nil {
		t.Error("expected error when processing audio on stopped listener")
	}
}

func TestWakeWordListener_StopWhenNotRunning(t *testing.T) {
	detector, _ := NewWakeWordDetector("fake_key", "jarvis")
	listener := NewWakeWordListener(detector, func() {})

	// Stopping when not running should be fine
	err := listener.Stop()
	if err != nil {
		t.Errorf("unexpected error stopping non-running listener: %v", err)
	}
}
