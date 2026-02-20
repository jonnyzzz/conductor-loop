package claude

import (
	"bufio"
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
)

// streamEvent represents a single JSON event from Claude's stream-json output.
type streamEvent struct {
	Type    string          `json:"type"`
	Subtype string          `json:"subtype,omitempty"`
	Result  string          `json:"result,omitempty"`
	IsError bool            `json:"is_error"`
	Message json.RawMessage `json:"message,omitempty"`
}

// assistantMessage is the nested message within a stream event.
type assistantMessage struct {
	Content []messageContent `json:"content"`
}

// messageContent is a content block within an assistant message.
type messageContent struct {
	Type string `json:"type"`
	Text string `json:"text,omitempty"`
}

// ParseStreamJSON extracts the final human-readable text from Claude's
// --output-format stream-json output. It returns the text from the "result"
// event if found, otherwise concatenates text blocks from assistant messages.
// Returns ("", false) if no useful text is found.
func ParseStreamJSON(data []byte) (string, bool) {
	scanner := bufio.NewScanner(bytes.NewReader(data))
	// Allow up to 1 MB per line to handle large tool call outputs.
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024)

	var textParts []string

	for scanner.Scan() {
		line := bytes.TrimSpace(scanner.Bytes())
		if len(line) == 0 {
			continue
		}

		var event streamEvent
		if err := json.Unmarshal(line, &event); err != nil {
			continue
		}

		switch event.Type {
		case "result":
			if !event.IsError && event.Result != "" {
				return event.Result, true
			}
		case "assistant":
			if len(event.Message) == 0 {
				continue
			}
			var msg assistantMessage
			if err := json.Unmarshal(event.Message, &msg); err != nil {
				continue
			}
			for _, content := range msg.Content {
				if content.Type == "text" && content.Text != "" {
					textParts = append(textParts, content.Text)
				}
			}
		}
	}

	if len(textParts) == 0 {
		return "", false
	}
	return strings.Join(textParts, "\n"), true
}

// writeOutputMDFromStream parses the Claude stream-json stdout file and writes
// the extracted final text to output.md in runDir. It is a no-op if output.md
// already exists. Returns an error if the stdout cannot be parsed.
func writeOutputMDFromStream(runDir, stdoutPath string) error {
	outputPath := filepath.Join(runDir, "output.md")
	if _, err := os.Stat(outputPath); err == nil {
		return nil // already exists, don't overwrite
	}

	data, err := os.ReadFile(stdoutPath)
	if err != nil {
		return errors.Wrap(err, "read stdout for stream parsing")
	}

	text, ok := ParseStreamJSON(data)
	if !ok {
		return errors.New("no result found in claude stream-json output")
	}

	if err := os.WriteFile(outputPath, []byte(text), 0o644); err != nil {
		return errors.Wrap(err, "write output.md from stream")
	}
	return nil
}
