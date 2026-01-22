package terminal

import (
	"context"
	"strings"
	"testing"

	"github.com/markshteyn/jarvis/internal/session"
)

func TestNewInjector(t *testing.T) {
	tests := []struct {
		terminalType session.TerminalType
		expectType   string
	}{
		{session.TerminalApple, "*terminal.TerminalAppInjector"},
		{session.TerminalITerm, "*terminal.ITermInjector"},
	}

	for _, tt := range tests {
		t.Run(tt.terminalType.String(), func(t *testing.T) {
			injector := NewInjector(tt.terminalType)
			typeName := strings.Replace(
				strings.Replace(
					strings.TrimPrefix(
						strings.TrimPrefix(
							typeNameOf(injector),
							"*"),
						"terminal."),
					"terminal.", "", -1),
				"*", "", -1)

			if !strings.Contains(tt.expectType, typeName) {
				t.Errorf("expected %s, got %T", tt.expectType, injector)
			}
		})
	}
}

func typeNameOf(v interface{}) string {
	return strings.Replace(
		strings.Replace(
			strings.TrimPrefix(
				strings.TrimPrefix(
					stringOf(v),
					"&{"),
				"*"),
			"}", "", -1),
		"{}", "", -1)
}

func stringOf(v interface{}) string {
	return strings.TrimPrefix(
		strings.TrimSuffix(
			strings.Replace(
				strings.Replace(
					strings.Replace(
						strings.Replace(
							strings.Replace(
								strings.Replace(
									typeString(v),
									"&", "", -1),
								"{", "", -1),
							"}", "", -1),
						"*", "", -1),
					"terminal.", "", -1),
				" ", "", -1),
			"Injector"),
		"terminal.")
}

func typeString(v interface{}) string {
	switch v.(type) {
	case *TerminalAppInjector:
		return "TerminalAppInjector"
	case *ITermInjector:
		return "ITermInjector"
	default:
		return "unknown"
	}
}

func TestEscapeForAppleScript(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"hello", "hello"},
		{`hello "world"`, `hello \"world\"`},
		{"line1\nline2", "line1\\nline2"},
		{`path\to\file`, `path\\to\\file`},
		{`"quoted"`, `\"quoted\"`},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := escapeForAppleScript(tt.input)
			if result != tt.expected {
				t.Errorf("escapeForAppleScript(%q) = %q, want %q",
					tt.input, result, tt.expected)
			}
		})
	}
}

func TestBuildTerminalAppScript(t *testing.T) {
	script := buildTerminalAppScript("/dev/ttys001", "hello world")

	if !strings.Contains(script, "/dev/ttys001") {
		t.Error("script should contain TTY")
	}
	if !strings.Contains(script, "hello world") {
		t.Error("script should contain text")
	}
	if !strings.Contains(script, "Terminal") {
		t.Error("script should reference Terminal app")
	}
}

func TestBuildITermScript(t *testing.T) {
	script := buildITermScript("/dev/ttys002", "test input")

	if !strings.Contains(script, "/dev/ttys002") {
		t.Error("script should contain TTY")
	}
	if !strings.Contains(script, "test input") {
		t.Error("script should contain text")
	}
	if !strings.Contains(script, "iTerm2") {
		t.Error("script should reference iTerm2 app")
	}
}

func TestTerminalAppInjector_EmptyTTY(t *testing.T) {
	injector := &TerminalAppInjector{}
	err := injector.InjectText(context.Background(), "", "test")
	if err == nil {
		t.Error("expected error for empty TTY")
	}
}

func TestTerminalAppInjector_EmptyText(t *testing.T) {
	injector := &TerminalAppInjector{}
	err := injector.InjectText(context.Background(), "/dev/ttys001", "")
	if err != nil {
		t.Errorf("expected no error for empty text, got %v", err)
	}
}

func TestITermInjector_EmptyTTY(t *testing.T) {
	injector := &ITermInjector{}
	err := injector.InjectText(context.Background(), "", "test")
	if err == nil {
		t.Error("expected error for empty TTY")
	}
}

func TestITermInjector_EmptyText(t *testing.T) {
	injector := &ITermInjector{}
	err := injector.InjectText(context.Background(), "/dev/ttys001", "")
	if err != nil {
		t.Errorf("expected no error for empty text, got %v", err)
	}
}

func TestInjectToSession_NilSession(t *testing.T) {
	err := InjectToSession(context.Background(), nil, "test")
	if err == nil {
		t.Error("expected error for nil session")
	}
}
