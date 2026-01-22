package transcript

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseTranscript(t *testing.T) {
	// Create a temporary test file
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.jsonl")

	// Write test transcript
	testData := `{"type":"user","message":{"role":"user","content":"Hello, can you help me?"}}
{"type":"assistant","message":{"role":"assistant","content":[{"type":"text","text":"Of course! I'd be happy to help you."}]}}
{"type":"user","message":{"role":"user","content":"Can you read a file?"}}
{"type":"assistant","message":{"role":"assistant","content":[{"type":"text","text":"Let me read that file for you."}]}}
{"type":"assistant","message":{"role":"assistant","content":[{"type":"tool_use","id":"toolu_123","name":"Read","input":{}}]}}
{"type":"assistant","message":{"role":"assistant","content":[{"type":"text","text":"The file contains important data."}]}}
`

	if err := os.WriteFile(testFile, []byte(testData), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	parser := NewParser()
	response, err := parser.ParseTranscript(testFile)
	if err != nil {
		t.Fatalf("ParseTranscript failed: %v", err)
	}

	// Should have text from the last conversation turn
	if response.TextContent == "" {
		t.Error("Expected non-empty text content")
	}

	if !response.HasToolCalls {
		t.Error("Expected HasToolCalls to be true")
	}

	if len(response.ToolsUsed) != 1 || response.ToolsUsed[0] != "Read" {
		t.Errorf("Expected ToolsUsed to contain 'Read', got: %v", response.ToolsUsed)
	}
}

func TestParseTranscript_TextOnlyResponse(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.jsonl")

	testData := `{"type":"user","message":{"role":"user","content":"What is 2+2?"}}
{"type":"assistant","message":{"role":"assistant","content":[{"type":"text","text":"2+2 equals 4."}]}}
`

	if err := os.WriteFile(testFile, []byte(testData), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	parser := NewParser()
	response, err := parser.ParseTranscript(testFile)
	if err != nil {
		t.Fatalf("ParseTranscript failed: %v", err)
	}

	if response.TextContent != "2+2 equals 4." {
		t.Errorf("Expected '2+2 equals 4.', got: %s", response.TextContent)
	}

	if response.HasToolCalls {
		t.Error("Expected HasToolCalls to be false")
	}
}

func TestParseTranscript_NoAssistantMessage(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.jsonl")

	testData := `{"type":"user","message":{"role":"user","content":"Hello"}}
`

	if err := os.WriteFile(testFile, []byte(testData), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	parser := NewParser()
	_, err := parser.ParseTranscript(testFile)
	if err == nil {
		t.Error("Expected error for transcript with no assistant messages")
	}
}

func TestGetLastUserMessage(t *testing.T) {
	tmpDir := t.TempDir()
	testFile := filepath.Join(tmpDir, "test.jsonl")

	testData := `{"type":"user","message":{"role":"user","content":"First message"}}
{"type":"assistant","message":{"role":"assistant","content":[{"type":"text","text":"Response 1"}]}}
{"type":"user","message":{"role":"user","content":"Second message"}}
{"type":"assistant","message":{"role":"assistant","content":[{"type":"text","text":"Response 2"}]}}
`

	if err := os.WriteFile(testFile, []byte(testData), 0644); err != nil {
		t.Fatalf("Failed to write test file: %v", err)
	}

	parser := NewParser()
	lastMsg, err := parser.GetLastUserMessage(testFile)
	if err != nil {
		t.Fatalf("GetLastUserMessage failed: %v", err)
	}

	if lastMsg != "Second message" {
		t.Errorf("Expected 'Second message', got: %s", lastMsg)
	}
}
