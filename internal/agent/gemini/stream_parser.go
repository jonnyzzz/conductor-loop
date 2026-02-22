package gemini

import (
	"bufio"
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"

	"github.com/pkg/errors"
)

// ParseStreamJSON extracts normalized assistant text from Gemini stream-json NDJSON output.
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

		eventType := geminiTextField(event["type"])
		if eventType == "result" {
			if text := geminiTextField(event["result"]); text != "" {
				return text, true
			}
		}

		if text := parseGeminiMessageText(eventType, event); text != "" {
			delta := diffText(emitted, text)
			if delta == "" {
				continue
			}
			builder.WriteString(delta)
			emitted += delta
		}
	}

	text := strings.TrimSpace(builder.String())
	if text == "" {
		return "", false
	}
	return text, true
}

func parseGeminiMessageText(eventType string, event map[string]json.RawMessage) string {
	switch eventType {
	case "message":
		role := geminiTextField(event["role"])
		if role != "" && role != "assistant" {
			return ""
		}
		if text := geminiTextField(event["content"]); text != "" {
			return text
		}
		if text := parseGeminiMessageObject(event["message"]); text != "" {
			return text
		}
	case "assistant":
		if text := parseGeminiMessageObject(event["message"]); text != "" {
			return text
		}
	}
	return ""
}

func parseGeminiMessageObject(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	var message map[string]json.RawMessage
	if err := json.Unmarshal(raw, &message); err != nil {
		return ""
	}
	if role := geminiTextField(message["role"]); role != "" && role != "assistant" {
		return ""
	}
	return geminiTextField(message["content"])
}

func geminiTextField(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}

	var asString string
	if err := json.Unmarshal(raw, &asString); err == nil {
		return strings.TrimSpace(asString)
	}

	var asObject map[string]json.RawMessage
	if err := json.Unmarshal(raw, &asObject); err == nil {
		for _, key := range []string{"text", "content", "result", "message"} {
			if text := geminiTextField(asObject[key]); text != "" {
				return text
			}
		}
		return ""
	}

	var asArray []json.RawMessage
	if err := json.Unmarshal(raw, &asArray); err == nil {
		parts := make([]string, 0, len(asArray))
		for _, part := range asArray {
			if text := geminiTextField(part); text != "" {
				parts = append(parts, text)
			}
		}
		return strings.TrimSpace(strings.Join(parts, "\n"))
	}

	return ""
}

// WriteOutputMDFromStream parses Gemini stream-json stdout and writes output.md.
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
		return errors.New("no result found in gemini stream-json output")
	}

	if err := os.WriteFile(outputPath, []byte(text), 0o644); err != nil {
		return errors.Wrap(err, "write output.md from stream")
	}
	return nil
}
