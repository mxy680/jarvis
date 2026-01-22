package session

import (
	"testing"
	"time"
)

func TestIdentity_Valid(t *testing.T) {
	tests := []struct {
		name     string
		identity Identity
		want     bool
	}{
		{
			name:     "valid identity",
			identity: Identity{SessionID: "abc123", TTY: "/dev/ttys001", TerminalApp: "Terminal"},
			want:     true,
		},
		{
			name:     "missing session ID",
			identity: Identity{SessionID: "", TTY: "/dev/ttys001"},
			want:     false,
		},
		{
			name:     "missing TTY",
			identity: Identity{SessionID: "abc123", TTY: ""},
			want:     false,
		},
		{
			name:     "minimal valid",
			identity: Identity{SessionID: "x", TTY: "y"},
			want:     true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.identity.Valid(); got != tt.want {
				t.Errorf("Identity.Valid() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestIdentity_TerminalType(t *testing.T) {
	tests := []struct {
		terminalApp string
		want        TerminalType
	}{
		{"iTerm.app", TerminalITerm},
		{"iTerm", TerminalITerm},
		{"iTerm2", TerminalITerm},
		{"iterm", TerminalITerm},
		{"Apple_Terminal", TerminalApple},
		{"Terminal", TerminalApple},
		{"", TerminalApple},
		{"unknown", TerminalApple},
	}

	for _, tt := range tests {
		t.Run(tt.terminalApp, func(t *testing.T) {
			identity := Identity{TerminalApp: tt.terminalApp}
			if got := identity.TerminalType(); got != tt.want {
				t.Errorf("TerminalType() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestNewManager(t *testing.T) {
	m := NewManager(3600)
	defer m.Close()

	if m == nil {
		t.Fatal("NewManager returned nil")
	}
	if m.Count() != 0 {
		t.Errorf("expected 0 sessions, got %d", m.Count())
	}
	if m.Active() != nil {
		t.Error("expected no active session")
	}
}

func TestManager_Register(t *testing.T) {
	m := NewManager(3600)
	defer m.Close()

	identity := Identity{
		SessionID:   "test-session-1",
		TTY:         "/dev/ttys001",
		TerminalApp: "iTerm.app",
	}

	session, err := m.Register(identity)
	if err != nil {
		t.Fatalf("Register failed: %v", err)
	}

	if session == nil {
		t.Fatal("Register returned nil session")
	}

	if m.Count() != 1 {
		t.Errorf("expected 1 session, got %d", m.Count())
	}

	// Should be active
	active := m.Active()
	if active == nil {
		t.Fatal("expected active session")
	}
	if active.Identity.SessionID != identity.SessionID {
		t.Errorf("expected active session ID %q, got %q",
			identity.SessionID, active.Identity.SessionID)
	}
}

func TestManager_RegisterInvalid(t *testing.T) {
	m := NewManager(3600)
	defer m.Close()

	_, err := m.Register(Identity{})
	if err == nil {
		t.Error("expected error for invalid identity")
	}
}

func TestManager_MultipleSessionsActiveSwitch(t *testing.T) {
	m := NewManager(3600)
	defer m.Close()

	// Register first session
	id1 := Identity{SessionID: "session-1", TTY: "/dev/ttys001", TerminalApp: "Terminal"}
	_, err := m.Register(id1)
	if err != nil {
		t.Fatalf("Register session 1 failed: %v", err)
	}

	// First session should be active
	if m.Active().Identity.SessionID != "session-1" {
		t.Error("session-1 should be active")
	}

	// Register second session
	id2 := Identity{SessionID: "session-2", TTY: "/dev/ttys002", TerminalApp: "iTerm.app"}
	_, err = m.Register(id2)
	if err != nil {
		t.Fatalf("Register session 2 failed: %v", err)
	}

	// Second session should now be active (most recent)
	if m.Active().Identity.SessionID != "session-2" {
		t.Error("session-2 should be active after registration")
	}

	if m.Count() != 2 {
		t.Errorf("expected 2 sessions, got %d", m.Count())
	}

	// Re-register first session (simulates new notification)
	_, err = m.Register(id1)
	if err != nil {
		t.Fatalf("Re-register session 1 failed: %v", err)
	}

	// First session should be active again
	if m.Active().Identity.SessionID != "session-1" {
		t.Error("session-1 should be active after re-registration")
	}

	// Still only 2 sessions (not duplicated)
	if m.Count() != 2 {
		t.Errorf("expected 2 sessions after re-register, got %d", m.Count())
	}
}

func TestManager_GetByTTY(t *testing.T) {
	m := NewManager(3600)
	defer m.Close()

	id := Identity{SessionID: "session-1", TTY: "/dev/ttys005", TerminalApp: "Terminal"}
	m.Register(id)

	// Find by TTY
	session := m.GetByTTY("/dev/ttys005")
	if session == nil {
		t.Fatal("GetByTTY returned nil")
	}
	if session.Identity.SessionID != "session-1" {
		t.Errorf("expected session-1, got %s", session.Identity.SessionID)
	}

	// Not found
	session = m.GetByTTY("/dev/ttys999")
	if session != nil {
		t.Error("expected nil for unknown TTY")
	}
}

func TestManager_Remove(t *testing.T) {
	m := NewManager(3600)
	defer m.Close()

	id := Identity{SessionID: "to-remove", TTY: "/dev/ttys001"}
	m.Register(id)

	if m.Count() != 1 {
		t.Fatal("expected 1 session")
	}

	m.Remove("to-remove")

	if m.Count() != 0 {
		t.Errorf("expected 0 sessions after remove, got %d", m.Count())
	}
	if m.Active() != nil {
		t.Error("expected no active session after remove")
	}
}

func TestSession_IsStale(t *testing.T) {
	session := NewSession(Identity{SessionID: "test", TTY: "/dev/ttys001"})

	// Fresh session should not be stale
	if session.IsStale(time.Hour) {
		t.Error("fresh session should not be stale")
	}

	// Manually set old activity time
	session.LastActivity = time.Now().Add(-2 * time.Hour)

	if !session.IsStale(time.Hour) {
		t.Error("old session should be stale")
	}
}

func TestSession_Touch(t *testing.T) {
	session := NewSession(Identity{SessionID: "test", TTY: "/dev/ttys001"})
	originalTime := session.LastActivity

	time.Sleep(10 * time.Millisecond)
	session.Touch()

	if !session.LastActivity.After(originalTime) {
		t.Error("Touch should update LastActivity")
	}
}
