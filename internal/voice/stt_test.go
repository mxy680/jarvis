package voice

import (
	"testing"
	"time"
)

func TestNewSTT(t *testing.T) {
	tests := []struct {
		name            string
		timeoutSeconds  int
		binPath         string
		expectedTimeout time.Duration
		expectedBinPath string
	}{
		{
			name:            "default values",
			timeoutSeconds:  0,
			binPath:         "",
			expectedTimeout: 30 * time.Second,
			expectedBinPath: "JarvisListen",
		},
		{
			name:            "custom timeout",
			timeoutSeconds:  60,
			binPath:         "",
			expectedTimeout: 60 * time.Second,
			expectedBinPath: "JarvisListen",
		},
		{
			name:            "custom bin path",
			timeoutSeconds:  30,
			binPath:         "/usr/local/bin/JarvisListen",
			expectedTimeout: 30 * time.Second,
			expectedBinPath: "/usr/local/bin/JarvisListen",
		},
		{
			name:            "negative timeout uses default",
			timeoutSeconds:  -5,
			binPath:         "",
			expectedTimeout: 30 * time.Second,
			expectedBinPath: "JarvisListen",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stt := NewSTT(tt.timeoutSeconds, tt.binPath)
			if stt.Timeout() != tt.expectedTimeout {
				t.Errorf("expected timeout %v, got %v", tt.expectedTimeout, stt.Timeout())
			}
			if stt.BinPath() != tt.expectedBinPath {
				t.Errorf("expected binPath %q, got %q", tt.expectedBinPath, stt.BinPath())
			}
		})
	}
}

func TestSTT_SetTimeout(t *testing.T) {
	stt := NewSTT(30, "")

	// Valid timeout
	stt.SetTimeout(60)
	if stt.Timeout() != 60*time.Second {
		t.Errorf("expected 60s timeout, got %v", stt.Timeout())
	}

	// Invalid timeout should not change
	stt.SetTimeout(0)
	if stt.Timeout() != 60*time.Second {
		t.Errorf("expected 60s timeout after invalid set, got %v", stt.Timeout())
	}

	// Negative timeout should not change
	stt.SetTimeout(-10)
	if stt.Timeout() != 60*time.Second {
		t.Errorf("expected 60s timeout after negative set, got %v", stt.Timeout())
	}
}
