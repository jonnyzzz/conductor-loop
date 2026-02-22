package codex

import (
	"bufio"
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
)

// ParseStreamJSON extracts normalized assistant text from Codex NDJSON output.
// It tolerates noisy non-JSON lines and duplicate/cumulative message chunks.
func ParseStreamJSON(data []byte) (string, bool) {
	scanner := bufio.NewScanner(bytes.NewReader(data))
	scanner.Buffer(make([]byte, 1024*1024), 1024*1024)

	var (
		builder strings.Builder
		emitted string
	)

	for scanner.Scan() {
		line := bytes.TrimSpace(scanner.Bytes())
		if len(line) == 0 {
			continue
		}

		var event map[string]json.RawMessage
		if err := json.Unmarshal(line, &event); err != nil {
			continue
		}

		eventType := textField(event["type"])
		if text := parseCodexTopLevelText(eventType, event); text != "" {
			appendNormalized(&builder, &emitted, text)
		}
		if text := parseCodexItemText(event["item"]); text != "" {
			appendNormalized(&builder, &emitted, text)
		}
	}

	text := strings.TrimSpace(builder.String())
	if text == "" {
		return "", false
	}
	return text, true
}

func parseCodexTopLevelText(eventType string, event map[string]json.RawMessage) string {
	switch eventType {
	case "result", "response.completed", "turn.completed", "message":
		// Ignore transport/progress event types and only parse likely final/message
		// events to avoid pulling tool output into output.md.
	default:
		return ""
	}
	for _, key := range []string{"result", "output_text", "text", "content", "message"} {
		if text := textField(event[key]); text != "" {
			return text
		}
	}
	return ""
}

func parseCodexItemText(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	var item struct {
		Type       string          `json:"type"`
		Text       string          `json:"text"`
		OutputText string          `json:"output_text"`
		Content    json.RawMessage `json:"content"`
		Message    json.RawMessage `json:"message"`
	}
	if err := json.Unmarshal(raw, &item); err != nil {
		return ""
	}
	itemType := strings.TrimSpace(item.Type)
	if itemType != "" && itemType != "agent_message" && itemType != "message" {
		return ""
	}
	for _, value := range []string{item.Text, item.OutputText} {
		if trimmed := strings.TrimSpace(value); trimmed != "" {
			return trimmed
		}
	}
	if text := textField(item.Content); text != "" {
		return text
	}
	if text := textField(item.Message); text != "" {
		return text
	}
	return ""
}

func textField(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}

	var asString string
	if err := json.Unmarshal(raw, &asString); err == nil {
		return strings.TrimSpace(asString)
	}

	var asObject map[string]json.RawMessage
	if err := json.Unmarshal(raw, &asObject); err == nil {
		for _, key := range []string{"text", "content", "result", "message", "output", "output_text"} {
			if text := textField(asObject[key]); text != "" {
				return text
			}
		}
		return ""
	}

	var asArray []json.RawMessage
	if err := json.Unmarshal(raw, &asArray); err == nil {
		parts := make([]string, 0, len(asArray))
		for _, item := range asArray {
			if text := textField(item); text != "" {
				parts = append(parts, text)
			}
		}
		return strings.TrimSpace(strings.Join(parts, "\n"))
	}

	return ""
}

func appendNormalized(builder *strings.Builder, emitted *string, text string) {
	trimmed := strings.TrimSpace(text)
	if trimmed == "" {
		return
	}
	if strings.HasPrefix(*emitted, trimmed) {
		return
	}

	if strings.HasPrefix(trimmed, *emitted) {
		delta := trimmed[len(*emitted):]
		if delta == "" {
			return
		}
		builder.WriteString(delta)
		*emitted += delta
		return
	}

	if builder.Len() > 0 && !strings.HasSuffix(builder.String(), "\n") {
		builder.WriteString("\n")
	}
	builder.WriteString(trimmed)
	*emitted += trimmed
}

// WriteOutputMDFromStream parses Codex NDJSON stdout and writes output.md.
// It is a no-op when output.md already exists.
func WriteOutputMDFromStream(runDir, stdoutPath string) error {
	outputPath := filepath.Join(runDir, "output.md")
	if _, err := os.Stat(outputPath); err == nil {
		return nil
	}

	data, err := os.ReadFile(stdoutPath)
	if err != nil {
		return errors.Wrap(err, "read stdout for stream parsing")
	}

	text, ok := ParseStreamJSON(data)
	if !ok {
		return errors.New("no result found in codex json output")
	}

	if err := os.WriteFile(outputPath, []byte(text), 0o644); err != nil {
		return errors.Wrap(err, "write output.md from stream")
	}
	return nil
}
