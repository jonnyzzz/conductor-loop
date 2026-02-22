package gemini

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseStreamJSONMessageEvent(t *testing.T) {
	input := `{"type":"message","role":"assistant","content":"hello gemini"}`

	text, ok := ParseStreamJSON([]byte(input))
	if !ok {
		t.Fatalf("expected parsed output")
	}
	if text != "hello gemini" {
		t.Fatalf("unexpected text: %q", text)
	}
}

func TestParseStreamJSONCumulativeMessageChunks(t *testing.T) {
	input := `{"type":"message","role":"assistant","content":"hello"}
{"type":"message","role":"assistant","content":"hello world"}`

	text, ok := ParseStreamJSON([]byte(input))
	if !ok {
		t.Fatalf("expected parsed output")
	}
	if text != "hello world" {
		t.Fatalf("unexpected text: %q", text)
	}
}

func TestParseStreamJSONResultField(t *testing.T) {
	input := `{"type":"result","result":"final summary"}`

	text, ok := ParseStreamJSON([]byte(input))
	if !ok {
		t.Fatalf("expected parsed output")
	}
	if text != "final summary" {
		t.Fatalf("unexpected text: %q", text)
	}
}

func TestParseStreamJSONSkipsInvalid(t *testing.T) {
	input := `not json
{"type":"tool_use","tool_name":"read_file"}`

	if _, ok := ParseStreamJSON([]byte(input)); ok {
		t.Fatalf("expected no parsed output")
	}
}

func TestWriteOutputMDFromStreamCreatesFile(t *testing.T) {
	dir := t.TempDir()
	stdoutPath := filepath.Join(dir, "agent-stdout.txt")
	data := `{"type":"message","role":"assistant","content":"final gemini output"}`
	if err := os.WriteFile(stdoutPath, []byte(data), 0o644); err != nil {
		t.Fatalf("write stdout: %v", err)
	}

	if err := WriteOutputMDFromStream(dir, stdoutPath); err != nil {
		t.Fatalf("WriteOutputMDFromStream: %v", err)
	}

	content, err := os.ReadFile(filepath.Join(dir, "output.md"))
	if err != nil {
		t.Fatalf("read output.md: %v", err)
	}
	if string(content) != "final gemini output" {
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
	data := `{"type":"message","role":"assistant","content":"new"}`
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
	if err := os.WriteFile(stdoutPath, []byte(`{"type":"tool_use"}`), 0o644); err != nil {
		t.Fatalf("write stdout: %v", err)
	}

	err := WriteOutputMDFromStream(dir, stdoutPath)
	if err == nil {
		t.Fatalf("expected parse error")
	}
}
