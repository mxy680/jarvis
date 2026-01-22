package summarizer

import (
	"context"
	"fmt"
	"strings"

	"github.com/anthropics/anthropic-sdk-go"
	"github.com/anthropics/anthropic-sdk-go/option"
)

// Summarizer handles summarization of Claude Code responses
type Summarizer struct {
	client   anthropic.Client
	maxWords int
}

// Config holds summarizer configuration
type Config struct {
	APIKey   string
	MaxWords int
}

// New creates a new Summarizer
func New(cfg Config) (*Summarizer, error) {
	if cfg.APIKey == "" {
		return nil, fmt.Errorf("API key is required")
	}

	client := anthropic.NewClient(
		option.WithAPIKey(cfg.APIKey),
	)

	maxWords := cfg.MaxWords
	if maxWords <= 0 {
		maxWords = 100
	}

	return &Summarizer{
		client:   client,
		maxWords: maxWords,
	}, nil
}

// SummarizeResponse summarizes a Claude Code response for verbal delivery
func (s *Summarizer) SummarizeResponse(ctx context.Context, content string, toolsUsed []string) (string, error) {
	if content == "" && len(toolsUsed) == 0 {
		return "Claude completed the task without any text output.", nil
	}

	// Build context about tools used
	var toolContext string
	if len(toolsUsed) > 0 {
		toolContext = fmt.Sprintf("\n\nTools used: %s", strings.Join(toolsUsed, ", "))
	}

	prompt := fmt.Sprintf(`Summarize the following Claude Code response in 2-3 sentences for verbal delivery.
Keep it concise (under %d words), conversational, and focus on what was accomplished.
Do not use markdown formatting, code blocks, or bullet points - this will be spoken aloud.
If there were errors or issues, mention them briefly.

Response to summarize:
%s%s`, s.maxWords, content, toolContext)

	message, err := s.client.Messages.New(ctx, anthropic.MessageNewParams{
		Model:     anthropic.ModelClaude3_5Haiku20241022,
		MaxTokens: 256,
		Messages: []anthropic.MessageParam{
			anthropic.NewUserMessage(anthropic.NewTextBlock(prompt)),
		},
	})
	if err != nil {
		return "", fmt.Errorf("failed to call Claude API: %w", err)
	}

	// Extract text from response
	var result strings.Builder
	for _, block := range message.Content {
		if block.Type == "text" {
			result.WriteString(block.Text)
		}
	}

	summary := strings.TrimSpace(result.String())
	if summary == "" {
		return "Claude completed the task.", nil
	}

	return summary, nil
}

// SummarizeError creates a verbal summary of an error
func (s *Summarizer) SummarizeError(err error) string {
	return fmt.Sprintf("I encountered an error: %s", err.Error())
}

// QuickSummary provides a fast summary without API call for simple cases
func (s *Summarizer) QuickSummary(content string, toolsUsed []string) string {
	// For very short responses, just return them directly
	words := strings.Fields(content)
	if len(words) <= 50 && len(toolsUsed) == 0 {
		return content
	}

	// For tool-only responses
	if content == "" && len(toolsUsed) > 0 {
		return fmt.Sprintf("I used %s to complete the task.", strings.Join(toolsUsed, ", "))
	}

	// Otherwise, need full summarization
	return ""
}
