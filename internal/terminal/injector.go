package terminal

import (
	"context"
	"fmt"

	"github.com/markshteyn/jarvis/internal/session"
)

// Injector injects text into a terminal session
type Injector interface {
	// InjectText types text into the terminal identified by TTY
	InjectText(ctx context.Context, tty string, text string) error
}

// NewInjector creates an injector for the specified terminal type
func NewInjector(terminalType session.TerminalType) Injector {
	switch terminalType {
	case session.TerminalITerm:
		return &ITermInjector{}
	default:
		return &TerminalAppInjector{}
	}
}

// InjectToSession injects text into the active session
func InjectToSession(ctx context.Context, sess *session.Session, text string) error {
	if sess == nil {
		return fmt.Errorf("no active session")
	}

	injector := NewInjector(sess.Identity.TerminalType())
	return injector.InjectText(ctx, sess.Identity.TTY, text)
}
