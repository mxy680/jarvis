package daemon

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"os"
	"path/filepath"
	"sync"
)

// NotifyPayload represents the JSON payload sent by jarvis-notify
type NotifyPayload struct {
	TranscriptPath string `json:"transcript_path"`
	SessionID      string `json:"session_id"`      // UUID for this terminal session
	TTY            string `json:"tty"`             // e.g., /dev/ttys001
	TerminalApp    string `json:"terminal_app"`    // "Terminal" or "iTerm"
}

// Handler processes incoming notifications
type Handler interface {
	HandleNotification(ctx context.Context, payload NotifyPayload) error
}

// Server manages the Unix socket server for receiving notifications
type Server struct {
	socketPath string
	listener   net.Listener
	handler    Handler
	wg         sync.WaitGroup
	ctx        context.Context
	cancel     context.CancelFunc
	mu         sync.Mutex
}

// DefaultSocketPath returns the default socket path
func DefaultSocketPath() string {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "/tmp/jarvis.sock"
	}
	return filepath.Join(homeDir, ".jarvis", "jarvis.sock")
}

// NewServer creates a new daemon server
func NewServer(socketPath string, handler Handler) *Server {
	ctx, cancel := context.WithCancel(context.Background())
	return &Server{
		socketPath: socketPath,
		handler:    handler,
		ctx:        ctx,
		cancel:     cancel,
	}
}

// Start begins listening for connections
func (s *Server) Start() error {
	// Ensure the directory exists
	socketDir := filepath.Dir(s.socketPath)
	if err := os.MkdirAll(socketDir, 0700); err != nil {
		return fmt.Errorf("failed to create socket directory: %w", err)
	}

	// Remove existing socket if present
	if err := os.Remove(s.socketPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove existing socket: %w", err)
	}

	listener, err := net.Listen("unix", s.socketPath)
	if err != nil {
		return fmt.Errorf("failed to listen on socket: %w", err)
	}

	// Set socket permissions
	if err := os.Chmod(s.socketPath, 0600); err != nil {
		listener.Close()
		return fmt.Errorf("failed to set socket permissions: %w", err)
	}

	s.mu.Lock()
	s.listener = listener
	s.mu.Unlock()

	log.Printf("Jarvis daemon listening on %s", s.socketPath)

	s.wg.Add(1)
	go s.acceptConnections()

	return nil
}

// acceptConnections handles incoming connections
func (s *Server) acceptConnections() {
	defer s.wg.Done()

	for {
		conn, err := s.listener.Accept()
		if err != nil {
			select {
			case <-s.ctx.Done():
				return
			default:
				log.Printf("Error accepting connection: %v", err)
				continue
			}
		}

		s.wg.Add(1)
		go s.handleConnection(conn)
	}
}

// handleConnection processes a single connection
func (s *Server) handleConnection(conn net.Conn) {
	defer s.wg.Done()
	defer conn.Close()

	decoder := json.NewDecoder(conn)
	var payload NotifyPayload

	if err := decoder.Decode(&payload); err != nil {
		log.Printf("Error decoding payload: %v", err)
		s.sendResponse(conn, false, "invalid JSON payload")
		return
	}

	if payload.TranscriptPath == "" {
		s.sendResponse(conn, false, "transcript_path is required")
		return
	}

	if err := s.handler.HandleNotification(s.ctx, payload); err != nil {
		log.Printf("Error handling notification: %v", err)
		s.sendResponse(conn, false, err.Error())
		return
	}

	s.sendResponse(conn, true, "ok")
}

// sendResponse writes a JSON response to the connection
func (s *Server) sendResponse(conn net.Conn, success bool, message string) {
	response := map[string]interface{}{
		"success": success,
		"message": message,
	}
	json.NewEncoder(conn).Encode(response)
}

// Stop gracefully shuts down the server
func (s *Server) Stop() error {
	s.cancel()

	s.mu.Lock()
	if s.listener != nil {
		s.listener.Close()
	}
	s.mu.Unlock()

	// Wait for all handlers to complete
	s.wg.Wait()

	// Clean up socket file
	os.Remove(s.socketPath)

	log.Println("Jarvis daemon stopped")
	return nil
}

// SocketPath returns the socket path
func (s *Server) SocketPath() string {
	return s.socketPath
}
