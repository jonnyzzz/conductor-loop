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

func TestParseVersion(t *testing.T) {
	tests := []struct {
		name    string
		raw     string
		major   int
		minor   int
		patch   int
		wantErr bool
	}{
		{"simple", "1.2.3", 1, 2, 3, false},
		{"v prefix", "v1.2.3", 1, 2, 3, false},
		{"claude format", "claude 1.0.0", 1, 0, 0, false},
		{"codex format", "codex 0.5.3", 0, 5, 3, false},
		{"gemini format", "gemini 2.1.0-beta", 2, 1, 0, false},
		{"with extra text", "Some CLI Tool v0.3.7 (build 1234)", 0, 3, 7, false},
		{"large numbers", "10.20.30", 10, 20, 30, false},
		{"no version", "no version here", 0, 0, 0, true},
		{"empty string", "", 0, 0, 0, true},
		{"partial version", "1.2", 0, 0, 0, true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			major, minor, patch, err := parseVersion(tc.raw)
			if tc.wantErr {
				if err == nil {
					t.Fatalf("parseVersion(%q): expected error, got nil", tc.raw)
				}
				return
			}
			if err != nil {
				t.Fatalf("parseVersion(%q): unexpected error: %v", tc.raw, err)
			}
			if major != tc.major || minor != tc.minor || patch != tc.patch {
				t.Errorf("parseVersion(%q) = (%d, %d, %d), want (%d, %d, %d)",
					tc.raw, major, minor, patch, tc.major, tc.minor, tc.patch)
			}
		})
	}
}

func TestIsVersionCompatible(t *testing.T) {
	tests := []struct {
		name       string
		detected   string
		minVersion [3]int
		want       bool
	}{
		{"exact match", "claude 1.0.0", [3]int{1, 0, 0}, true},
		{"above minimum", "claude 2.0.0", [3]int{1, 0, 0}, true},
		{"below minimum major", "claude 0.9.0", [3]int{1, 0, 0}, false},
		{"below minimum minor", "codex 0.0.9", [3]int{0, 1, 0}, false},
		{"above minimum minor", "codex 0.2.0", [3]int{0, 1, 0}, true},
		{"below minimum patch", "gemini 0.1.0", [3]int{0, 1, 1}, false},
		{"above minimum patch", "gemini 0.1.2", [3]int{0, 1, 1}, true},
		{"unparseable returns true", "unknown output", [3]int{1, 0, 0}, true},
		{"empty returns true", "", [3]int{1, 0, 0}, true},
		{"pre-release suffix", "claude 1.0.0-beta", [3]int{1, 0, 0}, true},
		{"higher major lower minor", "claude 2.0.0", [3]int{1, 5, 0}, true},
		{"same major lower minor", "claude 1.4.9", [3]int{1, 5, 0}, false},
		{"zero version", "tool 0.0.0", [3]int{0, 0, 0}, true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := isVersionCompatible(tc.detected, tc.minVersion)
			if got != tc.want {
				t.Errorf("isVersionCompatible(%q, %v) = %v, want %v",
					tc.detected, tc.minVersion, got, tc.want)
			}
		})
	}
}

func TestValidateAgentWarnsOnOldVersion(t *testing.T) {
	dir := t.TempDir()
	createVersionScript(t, dir, "claude", "claude 0.0.1")
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))

	// Should return nil (warn-only, no hard failure)
	if err := ValidateAgent(context.Background(), "claude"); err != nil {
		t.Fatalf("expected no error for old version (warn-only), got: %v", err)
	}
}

func TestValidateAgentPassesOnGoodVersion(t *testing.T) {
	dir := t.TempDir()
	createVersionScript(t, dir, "codex", "codex 0.5.0")
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))

	if err := ValidateAgent(context.Background(), "codex"); err != nil {
		t.Fatalf("expected no error for compatible version, got: %v", err)
	}
}

func TestValidateToken(t *testing.T) {
	tests := []struct {
		name      string
		agentType string
		token     string
		envKey    string
		envValue  string
		wantErr   bool
	}{
		{
			name:      "REST agent missing token",
			agentType: "perplexity",
			token:     "",
			wantErr:   true,
		},
		{
			name:      "REST agent present token",
			agentType: "perplexity",
			token:     "pplx-abc123",
			wantErr:   false,
		},
		{
			name:      "REST agent xai missing token",
			agentType: "xai",
			token:     "",
			wantErr:   true,
		},
		{
			name:      "REST agent xai present token",
			agentType: "xai",
			token:     "xai-token",
			wantErr:   false,
		},
		{
			name:      "CLI agent missing env var no config token",
			agentType: "claude",
			token:     "",
			envKey:    "ANTHROPIC_API_KEY",
			envValue:  "",
			wantErr:   true,
		},
		{
			name:      "CLI agent with env var set",
			agentType: "claude",
			token:     "",
			envKey:    "ANTHROPIC_API_KEY",
			envValue:  "sk-ant-key",
			wantErr:   false,
		},
		{
			name:      "CLI agent with config token but no env var",
			agentType: "claude",
			token:     "sk-ant-from-config",
			envKey:    "ANTHROPIC_API_KEY",
			envValue:  "",
			wantErr:   false,
		},
		{
			name:      "codex missing env var no config token",
			agentType: "codex",
			token:     "",
			envKey:    "OPENAI_API_KEY",
			envValue:  "",
			wantErr:   true,
		},
		{
			name:      "gemini with config token",
			agentType: "gemini",
			token:     "gemini-token-from-config",
			envKey:    "GEMINI_API_KEY",
			envValue:  "",
			wantErr:   false,
		},
		{
			name:      "unknown agent type",
			agentType: "unknown-agent",
			token:     "",
			wantErr:   false,
		},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if tc.envKey != "" {
				t.Setenv(tc.envKey, tc.envValue)
			}
			err := ValidateToken(tc.agentType, tc.token)
			if tc.wantErr && err == nil {
				t.Errorf("ValidateToken(%q, %q): expected error, got nil", tc.agentType, tc.token)
			}
			if !tc.wantErr && err != nil {
				t.Errorf("ValidateToken(%q, %q): unexpected error: %v", tc.agentType, tc.token, err)
			}
		})
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
