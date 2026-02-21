package claude

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseStreamJSONResultEvent(t *testing.T) {
	input := `{"type":"system","subtype":"init","session_id":"xxx"}
{"type":"assistant","message":{"content":[{"type":"text","text":"I'll start by reading the file."}]}}
{"type":"result","subtype":"success","is_error":false,"result":"Final answer text","session_id":"xxx","total_cost_usd":0.001}`

	text, ok := ParseStreamJSON([]byte(input))
	if !ok {
		t.Fatal("expected ok=true")
	}
	if text != "Final answer text" {
		t.Fatalf("expected 'Final answer text', got %q", text)
	}
}

func TestParseStreamJSONFallbackToAssistant(t *testing.T) {
	input := `{"type":"system","subtype":"init","session_id":"xxx"}
{"type":"assistant","message":{"content":[{"type":"text","text":"Hello"},{"type":"text","text":"World"}]}}`

	text, ok := ParseStreamJSON([]byte(input))
	if !ok {
		t.Fatal("expected ok=true")
	}
	if text != "Hello\nWorld" {
		t.Fatalf("unexpected text: %q", text)
	}
}

func TestParseStreamJSONEmptyInput(t *testing.T) {
	_, ok := ParseStreamJSON([]byte(""))
	if ok {
		t.Fatal("expected ok=false for empty input")
	}
}

func TestParseStreamJSONInvalidLines(t *testing.T) {
	input := `not valid json
{"type":"result","is_error":false,"result":"text","subtype":"success"}
also not valid`

	text, ok := ParseStreamJSON([]byte(input))
	if !ok {
		t.Fatal("expected ok=true despite invalid lines")
	}
	if text != "text" {
		t.Fatalf("expected 'text', got %q", text)
	}
}

func TestParseStreamJSONErrorResult(t *testing.T) {
	input := `{"type":"result","subtype":"error","is_error":true,"result":"error occurred"}`

	_, ok := ParseStreamJSON([]byte(input))
	if ok {
		t.Fatal("expected ok=false for error result")
	}
}

func TestParseStreamJSONMultipleAssistantMessages(t *testing.T) {
	input := `{"type":"assistant","message":{"content":[{"type":"text","text":"Part 1"}]}}
{"type":"tool_result","tool_use_id":"toolu_xxx","is_error":false}
{"type":"assistant","message":{"content":[{"type":"text","text":"Part 2"}]}}`

	text, ok := ParseStreamJSON([]byte(input))
	if !ok {
		t.Fatal("expected ok=true")
	}
	if text != "Part 1\nPart 2" {
		t.Fatalf("unexpected text: %q", text)
	}
}

func TestParseStreamJSONToolUseContentSkipped(t *testing.T) {
	// tool_use content blocks have no text, should be skipped
	input := `{"type":"assistant","message":{"content":[{"type":"tool_use","id":"toolu_xxx","name":"Write","input":{}}]}}
{"type":"result","is_error":false,"result":"done","subtype":"success"}`

	text, ok := ParseStreamJSON([]byte(input))
	if !ok {
		t.Fatal("expected ok=true")
	}
	if text != "done" {
		t.Fatalf("expected 'done', got %q", text)
	}
}

func TestParseStreamJSONEmptyResultField(t *testing.T) {
	// result event with empty result field and no assistant text → false
	input := `{"type":"result","is_error":false,"result":"","subtype":"success"}`

	_, ok := ParseStreamJSON([]byte(input))
	if ok {
		t.Fatal("expected ok=false for empty result field")
	}
}

func TestWriteOutputMDFromStreamCreatesFile(t *testing.T) {
	dir := t.TempDir()
	stdoutPath := filepath.Join(dir, "agent-stdout.txt")

	jsonStream := `{"type":"result","subtype":"success","is_error":false,"result":"The answer"}`
	if err := os.WriteFile(stdoutPath, []byte(jsonStream), 0o644); err != nil {
		t.Fatalf("write stdout: %v", err)
	}

	if err := WriteOutputMDFromStream(dir, stdoutPath); err != nil {
		t.Fatalf("WriteOutputMDFromStream: %v", err)
	}

	outputPath := filepath.Join(dir, "output.md")
	content, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("read output.md: %v", err)
	}
	if string(content) != "The answer" {
		t.Fatalf("unexpected content: %q", string(content))
	}
}

func TestWriteOutputMDFromStreamSkipsIfExists(t *testing.T) {
	dir := t.TempDir()
	stdoutPath := filepath.Join(dir, "agent-stdout.txt")
	outputPath := filepath.Join(dir, "output.md")

	// Pre-create output.md with existing content.
	if err := os.WriteFile(outputPath, []byte("existing"), 0o644); err != nil {
		t.Fatalf("write output.md: %v", err)
	}

	// Even with valid JSON, should not overwrite.
	jsonStream := `{"type":"result","is_error":false,"result":"new content","subtype":"success"}`
	if err := os.WriteFile(stdoutPath, []byte(jsonStream), 0o644); err != nil {
		t.Fatalf("write stdout: %v", err)
	}

	if err := WriteOutputMDFromStream(dir, stdoutPath); err != nil {
		t.Fatalf("WriteOutputMDFromStream: %v", err)
	}

	content, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("read output.md: %v", err)
	}
	if string(content) != "existing" {
		t.Fatalf("expected existing content preserved, got %q", string(content))
	}
}

func TestWriteOutputMDFromStreamNoResult(t *testing.T) {
	dir := t.TempDir()
	stdoutPath := filepath.Join(dir, "agent-stdout.txt")

	if err := os.WriteFile(stdoutPath, []byte("not json at all"), 0o644); err != nil {
		t.Fatalf("write stdout: %v", err)
	}

	err := WriteOutputMDFromStream(dir, stdoutPath)
	if err == nil {
		t.Fatal("expected error for non-JSON stdout")
	}

	// output.md should not have been created.
	outputPath := filepath.Join(dir, "output.md")
	if _, statErr := os.Stat(outputPath); !os.IsNotExist(statErr) {
		t.Fatal("output.md should not exist after failed parsing")
	}
}

func TestWriteOutputMDFromStreamMissingStdout(t *testing.T) {
	dir := t.TempDir()
	stdoutPath := filepath.Join(dir, "nonexistent.txt")

	err := WriteOutputMDFromStream(dir, stdoutPath)
	if err == nil {
		t.Fatal("expected error for missing stdout file")
	}
}
