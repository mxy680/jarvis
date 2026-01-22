package summarizer

import (
	"testing"
)

func TestNew_RequiresAPIKey(t *testing.T) {
	_, err := New(Config{})
	if err == nil {
		t.Error("Expected error when API key is missing")
	}
}

func TestNew_SetsDefaultMaxWords(t *testing.T) {
	s, err := New(Config{APIKey: "test-key"})
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if s.maxWords != 100 {
		t.Errorf("Expected default maxWords to be 100, got %d", s.maxWords)
	}
}

func TestNew_UsesProvidedMaxWords(t *testing.T) {
	s, err := New(Config{APIKey: "test-key", MaxWords: 50})
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if s.maxWords != 50 {
		t.Errorf("Expected maxWords to be 50, got %d", s.maxWords)
	}
}

func TestQuickSummary_ShortContent(t *testing.T) {
	s := &Summarizer{maxWords: 100}

	// Short content with no tools should be returned as-is
	content := "The file has been created successfully."
	result := s.QuickSummary(content, nil)
	if result != content {
		t.Errorf("Expected short content to be returned as-is, got: %s", result)
	}
}

func TestQuickSummary_ToolsOnly(t *testing.T) {
	s := &Summarizer{maxWords: 100}

	result := s.QuickSummary("", []string{"Read", "Write"})
	expected := "I used Read, Write to complete the task."
	if result != expected {
		t.Errorf("Expected '%s', got '%s'", expected, result)
	}
}

func TestQuickSummary_LongContent(t *testing.T) {
	s := &Summarizer{maxWords: 100}

	// Long content should return empty string (needs full summarization)
	longContent := ""
	for i := 0; i < 100; i++ {
		longContent += "word "
	}
	result := s.QuickSummary(longContent, nil)
	if result != "" {
		t.Error("Expected empty string for long content")
	}
}

func TestSummarizeError(t *testing.T) {
	s := &Summarizer{maxWords: 100}

	err := &testError{msg: "connection timeout"}
	result := s.SummarizeError(err)
	expected := "I encountered an error: connection timeout"
	if result != expected {
		t.Errorf("Expected '%s', got '%s'", expected, result)
	}
}

type testError struct {
	msg string
}

func (e *testError) Error() string {
	return e.msg
}
