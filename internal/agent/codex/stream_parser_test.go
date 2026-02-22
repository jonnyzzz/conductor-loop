package codex

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseStreamJSONResultEvent(t *testing.T) {
	input := `{"type":"item.completed","item":{"type":"agent_message","text":"hello from codex"}}`

	text, ok := ParseStreamJSON([]byte(input))
	if !ok {
		t.Fatalf("expected parsed output")
	}
	if text != "hello from codex" {
		t.Fatalf("unexpected text: %q", text)
	}
}

func TestParseStreamJSONCumulativeChunks(t *testing.T) {
	input := `{"type":"item.completed","item":{"type":"agent_message","text":"hello"}}
{"type":"item.completed","item":{"type":"agent_message","text":"hello world"}}`

	text, ok := ParseStreamJSON([]byte(input))
	if !ok {
		t.Fatalf("expected parsed output")
	}
	if text != "hello world" {
		t.Fatalf("unexpected text: %q", text)
	}
}

func TestParseStreamJSONSkipsInvalidLines(t *testing.T) {
	input := `not valid
{"type":"turn.started"}`

	if _, ok := ParseStreamJSON([]byte(input)); ok {
		t.Fatalf("expected no parsed output")
	}
}

func TestWriteOutputMDFromStreamCreatesFile(t *testing.T) {
	dir := t.TempDir()
	stdoutPath := filepath.Join(dir, "agent-stdout.txt")
	data := `{"type":"item.completed","item":{"type":"agent_message","text":"final codex output"}}`
	if err := os.WriteFile(stdoutPath, []byte(data), 0o644); err != nil {
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
	if string(content) != "final codex output" {
		t.Fatalf("unexpected output content: %q", string(content))
	}
}

func TestWriteOutputMDFromStreamSkipsIfExists(t *testing.T) {
	dir := t.TempDir()
	stdoutPath := filepath.Join(dir, "agent-stdout.txt")
	outputPath := filepath.Join(dir, "output.md")
	if err := os.WriteFile(outputPath, []byte("existing"), 0o644); err != nil {
		t.Fatalf("write output.md: %v", err)
	}
	data := `{"type":"item.completed","item":{"type":"agent_message","text":"new"}}`
	if err := os.WriteFile(stdoutPath, []byte(data), 0o644); err != nil {
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
		t.Fatalf("output.md should not be overwritten, got %q", string(content))
	}
}

func TestWriteOutputMDFromStreamNoResult(t *testing.T) {
	dir := t.TempDir()
	stdoutPath := filepath.Join(dir, "agent-stdout.txt")
	if err := os.WriteFile(stdoutPath, []byte(`{"type":"turn.started"}`), 0o644); err != nil {
		t.Fatalf("write stdout: %v", err)
	}

	err := WriteOutputMDFromStream(dir, stdoutPath)
	if err == nil {
		t.Fatalf("expected parse error")
	}
}
