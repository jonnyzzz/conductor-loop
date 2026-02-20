package runner

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestValidateAgentEmpty(t *testing.T) {
	if err := ValidateAgent(context.Background(), ""); err == nil {
		t.Fatalf("expected error for empty agent type")
	}
}

func TestValidateAgentRestSkipped(t *testing.T) {
	if err := ValidateAgent(context.Background(), "perplexity"); err != nil {
		t.Fatalf("expected no error for rest agent: %v", err)
	}
	if err := ValidateAgent(context.Background(), "xai"); err != nil {
		t.Fatalf("expected no error for rest agent: %v", err)
	}
}

func TestValidateAgentUnknown(t *testing.T) {
	if err := ValidateAgent(context.Background(), "unknown-agent-xyz"); err == nil {
		t.Fatalf("expected error for unknown agent type")
	}
}

func TestValidateAgentMissingCLI(t *testing.T) {
	t.Setenv("PATH", t.TempDir())
	if err := ValidateAgent(context.Background(), "claude"); err == nil {
		t.Fatalf("expected error for missing claude CLI")
	}
}

func TestValidateAgentWithFakeCLI(t *testing.T) {
	dir := t.TempDir()
	createVersionScript(t, dir, "claude", "claude 1.0.0")
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))

	if err := ValidateAgent(context.Background(), "claude"); err != nil {
		t.Fatalf("ValidateAgent: %v", err)
	}
}

func TestValidateAgentCodex(t *testing.T) {
	dir := t.TempDir()
	createVersionScript(t, dir, "codex", "codex 0.5.0")
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))

	if err := ValidateAgent(context.Background(), "codex"); err != nil {
		t.Fatalf("ValidateAgent: %v", err)
	}
}

func TestValidateAgentGemini(t *testing.T) {
	dir := t.TempDir()
	createVersionScript(t, dir, "gemini", "gemini 2.0.0")
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))

	if err := ValidateAgent(context.Background(), "gemini"); err != nil {
		t.Fatalf("ValidateAgent: %v", err)
	}
}

func TestValidateAgentVersionDetectFails(t *testing.T) {
	dir := t.TempDir()
	createFailingVersionScript(t, dir, "claude")
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))

	// Should NOT error when version detection fails - just warn
	if err := ValidateAgent(context.Background(), "claude"); err != nil {
		t.Fatalf("expected no error when version detection fails, got: %v", err)
	}
}

func TestCliCommand(t *testing.T) {
	tests := []struct {
		agentType string
		expected  string
	}{
		{"claude", "claude"},
		{"codex", "codex"},
		{"gemini", "gemini"},
		{"CLAUDE", "claude"},
		{"unknown", ""},
	}
	for _, tc := range tests {
		got := cliCommand(tc.agentType)
		if got != tc.expected {
			t.Errorf("cliCommand(%q) = %q, want %q", tc.agentType, got, tc.expected)
		}
	}
}

func createVersionScript(t *testing.T, dir, name, output string) {
	t.Helper()
	if runtime.GOOS == "windows" {
		path := filepath.Join(dir, name+".bat")
		content := "@echo off\r\nif \"%1\"==\"--version\" (\r\n  echo " + output + "\r\n) else (\r\n  more >nul\r\n  echo stdout\r\n)\r\n"
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatalf("write bat: %v", err)
		}
		return
	}
	path := filepath.Join(dir, name)
	content := "#!/bin/sh\nif [ \"$1\" = \"--version\" ]; then\n  echo '" + output + "'\nelse\n  cat >/dev/null\n  echo stdout\nfi\n"
	if err := os.WriteFile(path, []byte(content), 0o755); err != nil {
		t.Fatalf("write script: %v", err)
	}
}

func createFailingVersionScript(t *testing.T, dir, name string) {
	t.Helper()
	if runtime.GOOS == "windows" {
		path := filepath.Join(dir, name+".bat")
		content := "@echo off\r\nif \"%1\"==\"--version\" (\r\n  exit /b 1\r\n) else (\r\n  more >nul\r\n  echo stdout\r\n)\r\n"
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatalf("write bat: %v", err)
		}
		return
	}
	path := filepath.Join(dir, name)
	content := "#!/bin/sh\nif [ \"$1\" = \"--version\" ]; then\n  exit 1\nelse\n  cat >/dev/null\n  echo stdout\nfi\n"
	if err := os.WriteFile(path, []byte(content), 0o755); err != nil {
		t.Fatalf("write script: %v", err)
	}
}
