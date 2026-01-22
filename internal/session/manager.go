package session

import (
	"fmt"
	"log"
	"sync"
	"time"
)

// Manager tracks multiple Claude Code terminal sessions
type Manager struct {
	sessions       map[string]*Session // SessionID -> Session
	activeID       string              // Most recently notified session
	sessionTimeout time.Duration       // How long before sessions are cleaned up
	mu             sync.RWMutex
	cleanupTicker  *time.Ticker
	cleanupDone    chan struct{}
}

// NewManager creates a new session manager
func NewManager(sessionTimeoutSeconds int) *Manager {
	if sessionTimeoutSeconds <= 0 {
		sessionTimeoutSeconds = 3600 // Default: 1 hour
	}

	m := &Manager{
		sessions:       make(map[string]*Session),
		sessionTimeout: time.Duration(sessionTimeoutSeconds) * time.Second,
		cleanupDone:    make(chan struct{}),
	}

	// Start background cleanup
	m.startCleanup()

	return m
}

// Register adds or updates a session and makes it active
func (m *Manager) Register(identity Identity) (*Session, error) {
	if !identity.Valid() {
		return nil, fmt.Errorf("invalid session identity: missing required fields")
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	session, exists := m.sessions[identity.SessionID]
	if exists {
		// Update existing session
		session.Identity = identity
		session.Touch()
	} else {
		// Create new session
		session = NewSession(identity)
		m.sessions[identity.SessionID] = session
		log.Printf("Registered new session: %s (TTY: %s, Terminal: %s)",
			identity.SessionID, identity.TTY, identity.TerminalType())
	}

	// Make this session active
	m.activeID = identity.SessionID

	return session, nil
}

// Activate sets a session as the active session
func (m *Manager) Activate(sessionID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	session, exists := m.sessions[sessionID]
	if !exists {
		return fmt.Errorf("session not found: %s", sessionID)
	}

	session.Touch()
	m.activeID = sessionID
	return nil
}

// Active returns the currently active session, or nil if none
func (m *Manager) Active() *Session {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.activeID == "" {
		return nil
	}

	return m.sessions[m.activeID]
}

// Get returns a session by ID
func (m *Manager) Get(sessionID string) *Session {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return m.sessions[sessionID]
}

// GetByTTY returns a session by TTY path
func (m *Manager) GetByTTY(tty string) *Session {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, session := range m.sessions {
		if session.Identity.TTY == tty {
			return session
		}
	}
	return nil
}

// Remove removes a session by ID
func (m *Manager) Remove(sessionID string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	delete(m.sessions, sessionID)

	// Clear active if it was this session
	if m.activeID == sessionID {
		m.activeID = ""
	}
}

// Count returns the number of active sessions
func (m *Manager) Count() int {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return len(m.sessions)
}

// List returns all active sessions
func (m *Manager) List() []*Session {
	m.mu.RLock()
	defer m.mu.RUnlock()

	sessions := make([]*Session, 0, len(m.sessions))
	for _, s := range m.sessions {
		sessions = append(sessions, s)
	}
	return sessions
}

// startCleanup starts the background cleanup goroutine
func (m *Manager) startCleanup() {
	// Run cleanup every 5 minutes
	m.cleanupTicker = time.NewTicker(5 * time.Minute)

	go func() {
		for {
			select {
			case <-m.cleanupTicker.C:
				m.cleanupStale()
			case <-m.cleanupDone:
				return
			}
		}
	}()
}

// cleanupStale removes sessions that haven't had activity within the timeout
func (m *Manager) cleanupStale() {
	m.mu.Lock()
	defer m.mu.Unlock()

	for id, session := range m.sessions {
		if session.IsStale(m.sessionTimeout) {
			log.Printf("Removing stale session: %s (idle for %v)",
				id, session.IdleTime().Round(time.Second))
			delete(m.sessions, id)

			if m.activeID == id {
				m.activeID = ""
			}
		}
	}
}

// Close stops the manager and cleans up resources
func (m *Manager) Close() {
	if m.cleanupTicker != nil {
		m.cleanupTicker.Stop()
	}
	close(m.cleanupDone)
}
