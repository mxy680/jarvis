package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"syscall"

	"github.com/markshteyn/jarvis/internal/daemon"
	"github.com/markshteyn/jarvis/internal/session"
	"github.com/markshteyn/jarvis/internal/summarizer"
	"github.com/markshteyn/jarvis/internal/terminal"
	"github.com/markshteyn/jarvis/internal/transcript"
	"github.com/markshteyn/jarvis/internal/voice"
)

func main() {
	configPath := flag.String("config", "", "Path to config file (default: ~/.jarvis/config.yaml)")
	flag.Parse()

	// Load configuration
	if *configPath == "" {
		*configPath = daemon.DefaultConfigPath()
	}

	config, err := daemon.LoadConfig(*configPath)
	if err != nil {
		log.Fatalf("Failed to load config: %v", err)
	}

	if err := config.Validate(); err != nil {
		log.Fatalf("Invalid configuration: %v", err)
	}

	// Initialize components
	jarvis, err := NewJarvis(config)
	if err != nil {
		log.Fatalf("Failed to initialize Jarvis: %v", err)
	}
	defer jarvis.Close()

	// Set up signal handling
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigCh
		log.Println("Shutting down...")
		cancel()
	}()

	// Start the daemon
	if err := jarvis.Run(ctx); err != nil {
		log.Fatalf("Daemon error: %v", err)
	}
}

// Jarvis is the main daemon that coordinates all components
type Jarvis struct {
	config         *daemon.Config
	server         *daemon.Server
	sessionManager *session.Manager
	summarizer     *summarizer.Summarizer
	tts            *voice.TTS
	stt            *voice.STT
	parser         *transcript.Parser
}

// NewJarvis creates and initializes the Jarvis daemon
func NewJarvis(config *daemon.Config) (*Jarvis, error) {
	// Create summarizer
	sum, err := summarizer.New(summarizer.Config{
		APIKey:   config.ClaudeAPIKey,
		MaxWords: config.SummaryMaxWords,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create summarizer: %w", err)
	}

	// Determine STT binary path
	sttBinPath := getSTTBinPath()

	// Get session timeout from config
	sessionTimeout := config.MultiTerminal.SessionTimeoutSeconds
	if sessionTimeout <= 0 {
		sessionTimeout = 3600 // Default: 1 hour
	}

	j := &Jarvis{
		config:         config,
		sessionManager: session.NewManager(sessionTimeout),
		summarizer:     sum,
		tts:            voice.NewTTS(config.Voice),
		stt:            voice.NewSTT(config.STTTimeoutSeconds, sttBinPath),
		parser:         transcript.NewParser(),
	}

	// Get socket path
	socketPath := config.SocketPath
	if socketPath == "" {
		socketPath = daemon.DefaultSocketPath()
	}

	// Create server with handler
	j.server = daemon.NewServer(socketPath, j)

	return j, nil
}

// Run starts the daemon and blocks until context is canceled
func (j *Jarvis) Run(ctx context.Context) error {
	if err := j.server.Start(); err != nil {
		return fmt.Errorf("failed to start server: %w", err)
	}

	log.Println("Jarvis daemon is running")
	log.Printf("Voice: %s, STT timeout: %ds", j.config.Voice, j.config.STTTimeoutSeconds)

	// Wait for shutdown
	<-ctx.Done()

	return j.server.Stop()
}

// Close releases all resources
func (j *Jarvis) Close() {
	j.sessionManager.Close()
}

// HandleNotification processes incoming notifications from jarvis-notify
// This implements the daemon.Handler interface
func (j *Jarvis) HandleNotification(ctx context.Context, payload daemon.NotifyPayload) error {
	log.Printf("Received notification: session=%s, tty=%s, terminal=%s",
		payload.SessionID, payload.TTY, payload.TerminalApp)

	// Register/update session
	identity := session.Identity{
		SessionID:   payload.SessionID,
		TTY:         payload.TTY,
		TerminalApp: payload.TerminalApp,
	}
	if _, err := j.sessionManager.Register(identity); err != nil {
		log.Printf("Warning: failed to register session: %v", err)
	}

	// Parse the transcript
	response, err := j.parser.ParseTranscript(payload.TranscriptPath)
	if err != nil {
		errMsg := j.summarizer.SummarizeError(err)
		j.speak(ctx, errMsg)
		return err
	}

	// Generate summary
	summary := j.generateSummary(ctx, response)

	// Speak the summary
	j.speak(ctx, summary)

	// Listen for follow-up voice command
	j.listenForCommand(ctx)

	return nil
}

// generateSummary creates a verbal summary of the response
func (j *Jarvis) generateSummary(ctx context.Context, response *transcript.ParsedResponse) string {
	// Try quick summary first (no API call)
	if quick := j.summarizer.QuickSummary(response.TextContent, response.ToolsUsed); quick != "" {
		return quick
	}

	// Use Claude API for longer responses
	summary, err := j.summarizer.SummarizeResponse(ctx, response.TextContent, response.ToolsUsed)
	if err != nil {
		log.Printf("Failed to summarize: %v", err)
		return "Claude completed the task."
	}

	return summary
}

// speak uses TTS to speak text
func (j *Jarvis) speak(ctx context.Context, text string) {
	if text == "" {
		return
	}

	log.Printf("Speaking: %s", text)
	if err := j.tts.Speak(ctx, text); err != nil {
		log.Printf("TTS error: %v", err)
	}
}

// listenForCommand listens for a voice command and injects it into the active terminal
func (j *Jarvis) listenForCommand(ctx context.Context) {
	// Get active session
	activeSession := j.sessionManager.Active()
	if activeSession == nil {
		log.Println("No active session for voice command")
		return
	}

	// Prompt for input
	j.speak(ctx, "Listening...")

	// Listen for speech
	text, err := j.stt.Listen(ctx)
	if err != nil {
		log.Printf("STT error: %v", err)
		return
	}

	if text == "" {
		log.Println("No speech detected")
		return
	}

	log.Printf("Heard: %s", text)

	// Inject into terminal
	if err := terminal.InjectToSession(ctx, activeSession, text); err != nil {
		log.Printf("Failed to inject text: %v", err)
		j.speak(ctx, "Sorry, I couldn't send that to the terminal.")
		return
	}

	log.Printf("Injected text into terminal %s", activeSession.Identity.TTY)
}

// getSTTBinPath returns the path to the JarvisListen binary
func getSTTBinPath() string {
	// Check if JARVIS_STT_BIN is set
	if bin := os.Getenv("JARVIS_STT_BIN"); bin != "" {
		return bin
	}

	// Check common locations
	candidates := []string{
		"/usr/local/bin/JarvisListen",
		filepath.Join(os.Getenv("HOME"), ".jarvis", "bin", "JarvisListen"),
		"JarvisListen", // Assume in PATH
	}

	for _, path := range candidates {
		if path == "JarvisListen" {
			return path // Will be found via PATH
		}
		if _, err := os.Stat(path); err == nil {
			return path
		}
	}

	return "JarvisListen"
}
