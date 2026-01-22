package terminal

import (
	"strings"
)

// AppleScript templates for terminal injection
// These scripts find a terminal tab/session by TTY and type text into it

const terminalAppScript = `
tell application "Terminal"
    repeat with w in windows
        repeat with t in tabs of w
            if tty of t is "%s" then
                -- Bring window to front and select tab
                set frontmost of w to true
                set selected tab of w to t
                -- Type the text by setting contents of selected tab
                do script "%s" in t
                return "success"
            end if
        end repeat
    end repeat
end tell
return "tty_not_found"
`

const iTermScript = `
tell application "iTerm2"
    repeat with w in windows
        repeat with t in tabs of w
            repeat with s in sessions of t
                if tty of s is "%s" then
                    -- Bring window to front
                    select w
                    -- Select the tab
                    select t
                    -- Write text to the session
                    tell s to write text "%s"
                    return "success"
                end if
            end repeat
        end repeat
    end repeat
end tell
return "tty_not_found"
`

// escapeForAppleScript escapes a string for safe inclusion in AppleScript
func escapeForAppleScript(s string) string {
	// Escape backslashes first, then quotes
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, "\"", "\\\"")
	// Escape newlines
	s = strings.ReplaceAll(s, "\n", "\\n")
	s = strings.ReplaceAll(s, "\r", "\\r")
	return s
}

// buildTerminalAppScript creates an AppleScript for Terminal.app
func buildTerminalAppScript(tty, text string) string {
	return strings.Replace(
		strings.Replace(terminalAppScript, "%s", escapeForAppleScript(tty), 1),
		"%s", escapeForAppleScript(text), 1)
}

// buildITermScript creates an AppleScript for iTerm2
func buildITermScript(tty, text string) string {
	return strings.Replace(
		strings.Replace(iTermScript, "%s", escapeForAppleScript(tty), 1),
		"%s", escapeForAppleScript(text), 1)
}
