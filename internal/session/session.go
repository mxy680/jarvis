package session

import (
	"time"
)

// Session represents an active Claude Code terminal session
type Session struct {
	Identity     Identity
	LastActivity time.Time // Last notification received
	CreatedAt    time.Time
}

// NewSession creates a new session with the given identity
func NewSession(identity Identity) *Session {
	now := time.Now()
	return &Session{
		Identity:     identity,
		LastActivity: now,
		CreatedAt:    now,
	}
}

// Touch updates the last activity time
func (s *Session) Touch() {
	s.LastActivity = time.Now()
}

// IsStale returns true if the session hasn't had activity within the timeout
func (s *Session) IsStale(timeout time.Duration) bool {
	return time.Since(s.LastActivity) > timeout
}

// Age returns how long since the session was created
func (s *Session) Age() time.Duration {
	return time.Since(s.CreatedAt)
}

// IdleTime returns how long since last activity
func (s *Session) IdleTime() time.Duration {
	return time.Since(s.LastActivity)
}
