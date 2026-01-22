package voice

import (
	"context"
	"fmt"
	"sync"

	porcupine "github.com/Picovoice/porcupine/binding/go/v3"
)

// Porcupine audio constants (standard values from Porcupine documentation)
const (
	PorcupineFrameLength = 512   // Number of samples per frame
	PorcupineSampleRate  = 16000 // Required sample rate in Hz
)

// WakeWordDetector listens for wake word activation using Porcupine
type WakeWordDetector struct {
	accessKey string
	keyword   porcupine.BuiltInKeyword
	porcupine *porcupine.Porcupine
	mu        sync.Mutex
}

// NewWakeWordDetector creates a new wake word detector
func NewWakeWordDetector(accessKey string, keyword string) (*WakeWordDetector, error) {
	if accessKey == "" {
		return nil, fmt.Errorf("porcupine access key is required")
	}

	// Map keyword string to built-in keyword
	builtInKeyword, err := parseKeyword(keyword)
	if err != nil {
		return nil, err
	}

	return &WakeWordDetector{
		accessKey: accessKey,
		keyword:   builtInKeyword,
	}, nil
}

// parseKeyword converts a keyword string to a Porcupine built-in keyword
func parseKeyword(keyword string) (porcupine.BuiltInKeyword, error) {
	switch keyword {
	case "jarvis", "Jarvis", "JARVIS":
		return porcupine.JARVIS, nil
	case "alexa", "Alexa", "ALEXA":
		return porcupine.ALEXA, nil
	case "americano", "Americano", "AMERICANO":
		return porcupine.AMERICANO, nil
	case "blueberry", "Blueberry", "BLUEBERRY":
		return porcupine.BLUEBERRY, nil
	case "bumblebee", "Bumblebee", "BUMBLEBEE":
		return porcupine.BUMBLEBEE, nil
	case "computer", "Computer", "COMPUTER":
		return porcupine.COMPUTER, nil
	case "grapefruit", "Grapefruit", "GRAPEFRUIT":
		return porcupine.GRAPEFRUIT, nil
	case "grasshopper", "Grasshopper", "GRASSHOPPER":
		return porcupine.GRASSHOPPER, nil
	case "hey google", "Hey Google", "HEY GOOGLE":
		return porcupine.HEY_GOOGLE, nil
	case "hey siri", "Hey Siri", "HEY SIRI":
		return porcupine.HEY_SIRI, nil
	case "ok google", "Ok Google", "OK GOOGLE":
		return porcupine.OK_GOOGLE, nil
	case "picovoice", "Picovoice", "PICOVOICE":
		return porcupine.PICOVOICE, nil
	case "porcupine", "Porcupine", "PORCUPINE":
		return porcupine.PORCUPINE, nil
	case "terminator", "Terminator", "TERMINATOR":
		return porcupine.TERMINATOR, nil
	default:
		return porcupine.JARVIS, fmt.Errorf("unknown keyword %q, defaulting to jarvis", keyword)
	}
}

// Initialize sets up the Porcupine engine
func (w *WakeWordDetector) Initialize() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.porcupine != nil {
		return nil // Already initialized
	}

	p := &porcupine.Porcupine{
		AccessKey:       w.accessKey,
		BuiltInKeywords: []porcupine.BuiltInKeyword{w.keyword},
	}

	if err := p.Init(); err != nil {
		return fmt.Errorf("failed to initialize Porcupine: %w", err)
	}

	w.porcupine = p
	return nil
}

// Process processes a frame of audio and returns the keyword index if detected
// Returns -1 if no keyword detected, or the keyword index (0) if detected
func (w *WakeWordDetector) Process(pcm []int16) (int, error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.porcupine == nil {
		return -1, fmt.Errorf("wake word detector not initialized")
	}

	return w.porcupine.Process(pcm)
}

// FrameLength returns the number of audio samples per frame required by Porcupine
func (w *WakeWordDetector) FrameLength() int {
	return PorcupineFrameLength
}

// SampleRate returns the required sample rate for audio input
func (w *WakeWordDetector) SampleRate() int {
	return PorcupineSampleRate
}

// Close releases the Porcupine resources
func (w *WakeWordDetector) Close() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.porcupine != nil {
		if err := w.porcupine.Delete(); err != nil {
			return fmt.Errorf("failed to delete Porcupine: %w", err)
		}
		w.porcupine = nil
	}
	return nil
}

// WakeWordListener wraps the detector with audio capture for continuous listening
type WakeWordListener struct {
	detector  *WakeWordDetector
	onDetect  func()
	isRunning bool
	mu        sync.Mutex
	ctx       context.Context
	cancel    context.CancelFunc
}

// NewWakeWordListener creates a listener that invokes callback when wake word detected
func NewWakeWordListener(detector *WakeWordDetector, onDetect func()) *WakeWordListener {
	return &WakeWordListener{
		detector: detector,
		onDetect: onDetect,
	}
}

// Start begins listening for the wake word
// Note: Actual audio capture requires platform-specific code or external audio source
func (l *WakeWordListener) Start(ctx context.Context) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	if l.isRunning {
		return nil
	}

	if err := l.detector.Initialize(); err != nil {
		return err
	}

	l.ctx, l.cancel = context.WithCancel(ctx)
	l.isRunning = true

	return nil
}

// ProcessAudio processes audio samples and triggers callback on detection
// This should be called continuously with audio frames
func (l *WakeWordListener) ProcessAudio(pcm []int16) error {
	l.mu.Lock()
	if !l.isRunning {
		l.mu.Unlock()
		return fmt.Errorf("listener not running")
	}
	l.mu.Unlock()

	keywordIndex, err := l.detector.Process(pcm)
	if err != nil {
		return err
	}

	if keywordIndex >= 0 && l.onDetect != nil {
		l.onDetect()
	}

	return nil
}

// Stop stops listening for the wake word
func (l *WakeWordListener) Stop() error {
	l.mu.Lock()
	defer l.mu.Unlock()

	if !l.isRunning {
		return nil
	}

	if l.cancel != nil {
		l.cancel()
	}

	l.isRunning = false
	return l.detector.Close()
}

// IsRunning returns whether the listener is currently active
func (l *WakeWordListener) IsRunning() bool {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.isRunning
}
