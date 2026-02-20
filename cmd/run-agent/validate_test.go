package main

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// testValidateConfig is a minimal valid config with claude agent.
const testValidateConfig = `
agents:
  claude:
    type: claude
defaults:
  timeout: 30
`

// writeValidateTestConfig writes content to a temp config.yaml and returns its path.
func writeValidateTestConfig(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}
	return path
}

// createValidateCLIScript creates a fake CLI script in dir that responds to
// --version with the given versionOutput string.
func createValidateCLIScript(t *testing.T, dir, name, versionOutput string) {
	t.Helper()
	if runtime.GOOS == "windows" {
		path := filepath.Join(dir, name+".bat")
		content := "@echo off\r\nif \"%1\"==\"--version\" (\r\n  echo " + versionOutput + "\r\n) else (\r\n  more >nul\r\n  echo stdout\r\n)\r\n"
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatalf("write bat: %v", err)
		}
		return
	}
	path := filepath.Join(dir, name)
	content := "#!/bin/sh\nif [ \"$1\" = \"--version\" ]; then\n  echo '" + versionOutput + "'\nelse\n  cat >/dev/null\n  echo stdout\nfi\n"
	if err := os.WriteFile(path, []byte(content), 0o755); err != nil {
		t.Fatalf("write script: %v", err)
	}
}

func TestValidateCmd_NoConfig(t *testing.T) {
	// Use an empty temp dir so FindDefaultConfig finds nothing.
	t.Chdir(t.TempDir())
	err := runValidate("", "", "", false)
	if err != nil {
		t.Errorf("expected no error with no config found: %v", err)
	}
}

func TestValidateCmd_RootDirValid(t *testing.T) {
	dir := t.TempDir()
	err := runValidate("", dir, "", false)
	if err != nil {
		t.Errorf("expected no error with valid root dir: %v", err)
	}
}

func TestValidateCmd_RootDirNotExist(t *testing.T) {
	err := runValidate("", "/nonexistent/path/validate-test-xyz123", "", false)
	if err == nil {
		t.Error("expected error for nonexistent root dir")
	}
}

func TestValidateCmd_RootDirIsFile(t *testing.T) {
	dir := t.TempDir()
	filePath := filepath.Join(dir, "notadir.txt")
	if err := os.WriteFile(filePath, []byte("x"), 0o644); err != nil {
		t.Fatalf("create file: %v", err)
	}
	err := runValidate("", filePath, "", false)
	if err == nil {
		t.Error("expected error when root is a file not a directory")
	}
}

func TestValidateCmd_MissingCLI(t *testing.T) {
	cfgPath := writeValidateTestConfig(t, testValidateConfig)
	// Use an isolated PATH so claude CLI is not found.
	t.Setenv("PATH", t.TempDir())
	t.Setenv("ANTHROPIC_API_KEY", "")
	err := runValidate(cfgPath, "", "", false)
	if err == nil {
		t.Error("expected error when CLI not found")
	}
}

func TestValidateCmd_ValidConfigWithMockCLI(t *testing.T) {
	cfgPath := writeValidateTestConfig(t, testValidateConfig)

	dir := t.TempDir()
	createValidateCLIScript(t, dir, "claude", "claude 2.1.49")
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
	t.Setenv("ANTHROPIC_API_KEY", "sk-ant-test-key")

	err := runValidate(cfgPath, "", "", false)
	if err != nil {
		t.Errorf("expected no error with mock CLI and token set: %v", err)
	}
}

func TestValidateCmd_MissingToken(t *testing.T) {
	cfgPath := writeValidateTestConfig(t, testValidateConfig)

	dir := t.TempDir()
	createValidateCLIScript(t, dir, "claude", "claude 2.1.49")
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
	t.Setenv("ANTHROPIC_API_KEY", "")

	err := runValidate(cfgPath, "", "", false)
	if err == nil {
		t.Error("expected error when token is missing")
	}
}

func TestValidateCmd_ConfigTokenNoEnvVar(t *testing.T) {
	// Explicit config token should satisfy validation even without env var.
	cfgContent := `
agents:
  claude:
    type: claude
    token: sk-ant-from-config
defaults:
  timeout: 30
`
	cfgPath := writeValidateTestConfig(t, cfgContent)

	dir := t.TempDir()
	createValidateCLIScript(t, dir, "claude", "claude 2.1.49")
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
	t.Setenv("ANTHROPIC_API_KEY", "")

	err := runValidate(cfgPath, "", "", false)
	if err != nil {
		t.Errorf("expected no error when config token is set: %v", err)
	}
}

func TestValidateCmd_AgentFilter(t *testing.T) {
	cfgContent := `
agents:
  claude:
    type: claude
    token: sk-ant-test
  codex:
    type: codex
defaults:
  timeout: 30
`
	cfgPath := writeValidateTestConfig(t, cfgContent)

	dir := t.TempDir()
	// Only create mock for claude; codex CLI is absent.
	createValidateCLIScript(t, dir, "claude", "claude 2.1.49")
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
	t.Setenv("ANTHROPIC_API_KEY", "sk-ant-test")
	t.Setenv("OPENAI_API_KEY", "")

	// Filter to only claude — codex is excluded so its missing CLI/token is ignored.
	err := runValidate(cfgPath, "", "claude", false)
	if err != nil {
		t.Errorf("expected no error when filtering to valid agent: %v", err)
	}
}

func TestValidateCmd_AgentFilterNotFound(t *testing.T) {
	cfgPath := writeValidateTestConfig(t, testValidateConfig)
	err := runValidate(cfgPath, "", "nonexistent-agent", false)
	if err == nil {
		t.Error("expected error when agent filter doesn't match any configured agent")
	}
}

func TestValidateCmd_OutputContainsExpectedStrings(t *testing.T) {
	cfgPath := writeValidateTestConfig(t, testValidateConfig)

	dir := t.TempDir()
	createValidateCLIScript(t, dir, "claude", "claude 2.1.49")
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
	t.Setenv("ANTHROPIC_API_KEY", "sk-ant-test-key")

	output := captureStdout(t, func() {
		_ = runValidate(cfgPath, "", "", false)
	})

	checks := []string{
		"Conductor Loop Configuration Validator",
		"claude",
		"CLI found",
		"Validation:",
	}
	for _, want := range checks {
		if !strings.Contains(output, want) {
			t.Errorf("expected output to contain %q, got:\n%s", want, output)
		}
	}
}

func TestValidateCmd_OutputVersionDisplayed(t *testing.T) {
	cfgPath := writeValidateTestConfig(t, testValidateConfig)

	dir := t.TempDir()
	createValidateCLIScript(t, dir, "claude", "claude 2.1.49")
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
	t.Setenv("ANTHROPIC_API_KEY", "sk-ant-test-key")

	output := captureStdout(t, func() {
		_ = runValidate(cfgPath, "", "", false)
	})

	if !strings.Contains(output, "2.1.49") {
		t.Errorf("expected version '2.1.49' in output, got:\n%s", output)
	}
}

func TestValidateCmd_RESTAgentWithToken(t *testing.T) {
	cfgContent := `
agents:
  perplexity:
    type: perplexity
    token: pplx-test-token
defaults:
  timeout: 30
`
	cfgPath := writeValidateTestConfig(t, cfgContent)

	err := runValidate(cfgPath, "", "", false)
	if err != nil {
		t.Errorf("expected no error with REST agent and token: %v", err)
	}
}

func TestValidateCmd_RESTAgentMissingToken(t *testing.T) {
	cfgContent := `
agents:
  perplexity:
    type: perplexity
defaults:
  timeout: 30
`
	cfgPath := writeValidateTestConfig(t, cfgContent)
	// Ensure neither the conductor-specific nor the agent-specific env vars are set.
	t.Setenv("CONDUCTOR_AGENT_PERPLEXITY_TOKEN", "")

	err := runValidate(cfgPath, "", "", false)
	if err == nil {
		t.Error("expected error when REST agent has no token")
	}
}

func TestValidateCmd_CheckNetworkFlagNoConfig(t *testing.T) {
	t.Chdir(t.TempDir())
	// --check-network with no config should not crash.
	err := runValidate("", "", "", true)
	if err != nil {
		t.Errorf("expected no error with --check-network and no config: %v", err)
	}
}

func TestValidateCmd_MultipleAgents(t *testing.T) {
	cfgContent := `
agents:
  claude:
    type: claude
    token: sk-ant-test
  codex:
    type: codex
    token: sk-openai-test
defaults:
  timeout: 30
`
	cfgPath := writeValidateTestConfig(t, cfgContent)

	dir := t.TempDir()
	createValidateCLIScript(t, dir, "claude", "claude 2.1.49")
	createValidateCLIScript(t, dir, "codex", "codex 0.104.0")
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
	t.Setenv("ANTHROPIC_API_KEY", "sk-ant-test")
	t.Setenv("OPENAI_API_KEY", "sk-openai-test")

	output := captureStdout(t, func() {
		_ = runValidate(cfgPath, "", "", false)
	})

	if !strings.Contains(output, "claude") {
		t.Errorf("expected 'claude' in output, got:\n%s", output)
	}
	if !strings.Contains(output, "codex") {
		t.Errorf("expected 'codex' in output, got:\n%s", output)
	}
	if !strings.Contains(output, "Validation: 2 OK") {
		t.Errorf("expected 'Validation: 2 OK' in output, got:\n%s", output)
	}
}

func TestValidateCmd_SubcommandRegistered(t *testing.T) {
	cmd := newRootCmd()
	cmd.SetArgs([]string{"validate", "--help"})
	// Should not return an error (just prints help).
	_ = cmd.Execute()
}

func TestSortedAgentNames(t *testing.T) {
	cfgContent := `
agents:
  zebra:
    type: claude
  alpha:
    type: codex
  mango:
    type: gemini
defaults:
  timeout: 30
`
	cfgPath := writeValidateTestConfig(t, cfgContent)

	dir := t.TempDir()
	createValidateCLIScript(t, dir, "claude", "claude 2.0.0")
	createValidateCLIScript(t, dir, "codex", "codex 0.1.0")
	createValidateCLIScript(t, dir, "gemini", "gemini 1.0.0")
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
	t.Setenv("ANTHROPIC_API_KEY", "sk-ant-test")
	t.Setenv("OPENAI_API_KEY", "sk-openai-test")
	t.Setenv("GEMINI_API_KEY", "gemini-test")

	output := captureStdout(t, func() {
		_ = runValidate(cfgPath, "", "", false)
	})

	// Agents should appear in alphabetical order: alpha, mango, zebra.
	alphaIdx := strings.Index(output, "alpha")
	mangoIdx := strings.Index(output, "mango")
	zebraIdx := strings.Index(output, "zebra")

	if alphaIdx < 0 || mangoIdx < 0 || zebraIdx < 0 {
		t.Fatalf("expected all agent names in output, got:\n%s", output)
	}
	if !(alphaIdx < mangoIdx && mangoIdx < zebraIdx) {
		t.Errorf("expected alphabetical order (alpha < mango < zebra), got positions: alpha=%d mango=%d zebra=%d\noutput:\n%s",
			alphaIdx, mangoIdx, zebraIdx, output)
	}
}

func TestExtractValidateVersion(t *testing.T) {
	tests := []struct {
		raw  string
		want string
	}{
		{"claude 2.1.49", "2.1.49"},
		{"codex v0.104.0", "0.104.0"},
		{"gemini 0.28.2-beta", "0.28.2"},
		{"no version here", ""},
		{"", ""},
	}
	for _, tc := range tests {
		got := extractValidateVersion(tc.raw)
		if got != tc.want {
			t.Errorf("extractValidateVersion(%q) = %q, want %q", tc.raw, got, tc.want)
		}
	}
}
