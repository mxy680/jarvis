package transcript

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

// Message represents a parsed message from the transcript
type Message struct {
	Type    string `json:"type"`
	Message struct {
		Role    string          `json:"role"`
		Content json.RawMessage `json:"content"`
	} `json:"message"`
}

// ContentBlock represents a content block in an assistant message
type ContentBlock struct {
	Type string `json:"type"`
	Text string `json:"text,omitempty"`
	// Tool use fields
	ID    string          `json:"id,omitempty"`
	Name  string          `json:"name,omitempty"`
	Input json.RawMessage `json:"input,omitempty"`
}

// ParsedResponse holds the extracted response from a transcript
type ParsedResponse struct {
	TextContent  string   // Combined text content
	ToolsUsed    []string // Names of tools that were used
	HasToolCalls bool     // Whether any tool calls were made
}

// Parser handles transcript parsing
type Parser struct{}

// NewParser creates a new transcript parser
func NewParser() *Parser {
	return &Parser{}
}

// ParseTranscript reads a transcript file and extracts the last assistant response
func (p *Parser) ParseTranscript(path string) (*ParsedResponse, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("failed to open transcript: %w", err)
	}
	defer file.Close()

	var lastAssistantMessages []Message
	scanner := bufio.NewScanner(file)

	// Increase buffer size for large lines
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 1024*1024)

	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		var msg Message
		if err := json.Unmarshal(line, &msg); err != nil {
			continue // Skip malformed lines
		}

		// Track assistant messages
		if msg.Message.Role == "assistant" {
			lastAssistantMessages = append(lastAssistantMessages, msg)
		} else if msg.Message.Role == "user" {
			// Reset on new user message - we want the last conversation turn
			lastAssistantMessages = nil
		}
	}

	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("error reading transcript: %w", err)
	}

	if len(lastAssistantMessages) == 0 {
		return nil, fmt.Errorf("no assistant messages found in transcript")
	}

	return p.extractResponse(lastAssistantMessages)
}

// extractResponse extracts text and tool information from assistant messages
func (p *Parser) extractResponse(messages []Message) (*ParsedResponse, error) {
	response := &ParsedResponse{
		ToolsUsed: []string{},
	}

	var textParts []string
	toolSet := make(map[string]bool)

	for _, msg := range messages {
		// Handle content as either string or array of blocks
		content := msg.Message.Content

		// Try parsing as string first
		var textContent string
		if err := json.Unmarshal(content, &textContent); err == nil {
			if textContent != "" {
				textParts = append(textParts, textContent)
			}
			continue
		}

		// Parse as array of content blocks
		var blocks []ContentBlock
		if err := json.Unmarshal(content, &blocks); err != nil {
			continue // Skip if neither format works
		}

		for _, block := range blocks {
			switch block.Type {
			case "text":
				if block.Text != "" {
					textParts = append(textParts, block.Text)
				}
			case "tool_use":
				response.HasToolCalls = true
				if block.Name != "" && !toolSet[block.Name] {
					toolSet[block.Name] = true
					response.ToolsUsed = append(response.ToolsUsed, block.Name)
				}
			}
		}
	}

	response.TextContent = strings.TrimSpace(strings.Join(textParts, "\n"))

	return response, nil
}

// GetLastUserMessage extracts the last user message from a transcript
func (p *Parser) GetLastUserMessage(path string) (string, error) {
	file, err := os.Open(path)
	if err != nil {
		return "", fmt.Errorf("failed to open transcript: %w", err)
	}
	defer file.Close()

	var lastUserContent string
	scanner := bufio.NewScanner(file)

	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 1024*1024)

	for scanner.Scan() {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}

		var msg Message
		if err := json.Unmarshal(line, &msg); err != nil {
			continue
		}

		if msg.Message.Role == "user" {
			// Try to extract text content
			var textContent string
			if err := json.Unmarshal(msg.Message.Content, &textContent); err == nil {
				lastUserContent = textContent
				continue
			}

			// Try array format
			var blocks []ContentBlock
			if err := json.Unmarshal(msg.Message.Content, &blocks); err == nil {
				for _, block := range blocks {
					if block.Type == "text" {
						lastUserContent = block.Text
						break
					}
				}
			}
		}
	}

	return lastUserContent, nil
}
