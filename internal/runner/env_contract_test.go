package runner

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// createEnvDumpCLI creates a fake agent CLI that dumps its environment to stdout.
func createEnvDumpCLI(t *testing.T, dir, name string) {
	t.Helper()
	if runtime.GOOS == "windows" {
		path := filepath.Join(dir, name+".bat")
		content := "@echo off\r\nif \"%1\"==\"--version\" (\r\n  echo " + name + " 1.0.0\r\n  exit /b 0\r\n)\r\nmore >nul\r\nset\r\n"
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatalf("write bat: %v", err)
		}
		return
	}
	path := filepath.Join(dir, name)
	content := "#!/bin/sh\nif [ \"$1\" = \"--version\" ]; then echo '" + name + " 1.0.0'; exit 0; fi\ncat >/dev/null\nenv\n"
	if err := os.WriteFile(path, []byte(content), 0o755); err != nil {
		t.Fatalf("write script: %v", err)
	}
}

// parseEnvOutput parses env command output into a map.
func parseEnvOutput(output string) map[string]string {
	result := make(map[string]string)
	for _, line := range strings.Split(output, "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) == 2 {
			result[parts[0]] = parts[1]
		}
	}
	return result
}

func TestEnvContractTokenInjection(t *testing.T) {
	root := t.TempDir()
	binDir := filepath.Join(root, "bin")
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		t.Fatalf("mkdir bin: %v", err)
	}
	createEnvDumpCLI(t, binDir, "codex")
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))

	configPath := filepath.Join(root, "config.yaml")
	configContent := `agents:
  codex:
    type: codex
    token: test-secret-token

defaults:
  agent: codex
  timeout: 10
`
	if err := os.WriteFile(configPath, []byte(configContent), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	info, err := runJob("project", "task", JobOptions{
		RootDir:    root,
		ConfigPath: configPath,
		Agent:      "codex",
		Prompt:     "hello",
	})
	if err != nil {
		t.Fatalf("runJob: %v", err)
	}

	stdout, err := os.ReadFile(info.StdoutPath)
	if err != nil {
		t.Fatalf("read stdout: %v", err)
	}
	envVars := parseEnvOutput(string(stdout))

	if got := envVars["OPENAI_API_KEY"]; got != "test-secret-token" {
		t.Errorf("expected OPENAI_API_KEY=test-secret-token, got %q", got)
	}
}

func TestEnvContractTokenPassthrough(t *testing.T) {
	root := t.TempDir()
	binDir := filepath.Join(root, "bin")
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		t.Fatalf("mkdir bin: %v", err)
	}
	createEnvDumpCLI(t, binDir, "codex")
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))
	t.Setenv("OPENAI_API_KEY", "env-passthrough-token")

	// No config — agent selected by name, no token override.
	info, err := runJob("project", "task", JobOptions{
		RootDir: root,
		Agent:   "codex",
		Prompt:  "hello",
	})
	if err != nil {
		t.Fatalf("runJob: %v", err)
	}

	stdout, err := os.ReadFile(info.StdoutPath)
	if err != nil {
		t.Fatalf("read stdout: %v", err)
	}
	envVars := parseEnvOutput(string(stdout))

	if got := envVars["OPENAI_API_KEY"]; got != "env-passthrough-token" {
		t.Errorf("expected OPENAI_API_KEY=env-passthrough-token (passthrough), got %q", got)
	}
}

func TestEnvContractTokenFile(t *testing.T) {
	root := t.TempDir()
	binDir := filepath.Join(root, "bin")
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		t.Fatalf("mkdir bin: %v", err)
	}
	createEnvDumpCLI(t, binDir, "codex")
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))

	tokenFile := filepath.Join(root, "token.txt")
	if err := os.WriteFile(tokenFile, []byte("file-loaded-token\n"), 0o644); err != nil {
		t.Fatalf("write token file: %v", err)
	}

	configPath := filepath.Join(root, "config.yaml")
	configContent := "agents:\n  codex:\n    type: codex\n    token_file: " + tokenFile + "\n\ndefaults:\n  agent: codex\n  timeout: 10\n"
	if err := os.WriteFile(configPath, []byte(configContent), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}

	info, err := runJob("project", "task", JobOptions{
		RootDir:    root,
		ConfigPath: configPath,
		Agent:      "codex",
		Prompt:     "hello",
	})
	if err != nil {
		t.Fatalf("runJob: %v", err)
	}

	stdout, err := os.ReadFile(info.StdoutPath)
	if err != nil {
		t.Fatalf("read stdout: %v", err)
	}
	envVars := parseEnvOutput(string(stdout))

	if got := envVars["OPENAI_API_KEY"]; got != "file-loaded-token" {
		t.Errorf("expected OPENAI_API_KEY=file-loaded-token (from token_file), got %q", got)
	}
}

func TestEnvContractRunsDirAndMessageBus(t *testing.T) {
	root := t.TempDir()
	binDir := filepath.Join(root, "bin")
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		t.Fatalf("mkdir bin: %v", err)
	}
	createEnvDumpCLI(t, binDir, "codex")
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))

	info, err := runJob("project", "task", JobOptions{
		RootDir: root,
		Agent:   "codex",
		Prompt:  "hello",
	})
	if err != nil {
		t.Fatalf("runJob: %v", err)
	}

	stdout, err := os.ReadFile(info.StdoutPath)
	if err != nil {
		t.Fatalf("read stdout: %v", err)
	}
	envVars := parseEnvOutput(string(stdout))

	expectedRunsDir := filepath.Join(root, "project", "task", "runs")
	if got := envVars["RUNS_DIR"]; got != expectedRunsDir {
		t.Errorf("expected RUNS_DIR=%q, got %q", expectedRunsDir, got)
	}

	expectedBusPath := filepath.Join(root, "project", "task", "TASK-MESSAGE-BUS.md")
	if got := envVars["MESSAGE_BUS"]; got != expectedBusPath {
		t.Errorf("expected MESSAGE_BUS=%q, got %q", expectedBusPath, got)
	}
}

func TestEnvContractJRunVars(t *testing.T) {
	root := t.TempDir()
	binDir := filepath.Join(root, "bin")
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		t.Fatalf("mkdir bin: %v", err)
	}
	createEnvDumpCLI(t, binDir, "codex")
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))

	info, err := runJob("my-project", "my-task", JobOptions{
		RootDir:     root,
		Agent:       "codex",
		Prompt:      "hello",
		ParentRunID: "parent-run-1",
	})
	if err != nil {
		t.Fatalf("runJob: %v", err)
	}

	stdout, err := os.ReadFile(info.StdoutPath)
	if err != nil {
		t.Fatalf("read stdout: %v", err)
	}
	envVars := parseEnvOutput(string(stdout))

	if got := envVars["JRUN_PROJECT_ID"]; got != "my-project" {
		t.Errorf("expected JRUN_PROJECT_ID=my-project, got %q", got)
	}
	if got := envVars["JRUN_TASK_ID"]; got != "my-task" {
		t.Errorf("expected JRUN_TASK_ID=my-task, got %q", got)
	}
	if got := envVars["JRUN_ID"]; got != info.RunID {
		t.Errorf("expected JRUN_ID=%q, got %q", info.RunID, got)
	}
	if got := envVars["JRUN_PARENT_ID"]; got != "parent-run-1" {
		t.Errorf("expected JRUN_PARENT_ID=parent-run-1, got %q", got)
	}
}

func TestEnvContractCLAUDECODERemoved(t *testing.T) {
	root := t.TempDir()
	binDir := filepath.Join(root, "bin")
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		t.Fatalf("mkdir bin: %v", err)
	}
	createEnvDumpCLI(t, binDir, "codex")
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))
	t.Setenv("CLAUDECODE", "should-be-removed")

	info, err := runJob("project", "task", JobOptions{
		RootDir: root,
		Agent:   "codex",
		Prompt:  "hello",
	})
	if err != nil {
		t.Fatalf("runJob: %v", err)
	}

	stdout, err := os.ReadFile(info.StdoutPath)
	if err != nil {
		t.Fatalf("read stdout: %v", err)
	}
	envVars := parseEnvOutput(string(stdout))

	if _, ok := envVars["CLAUDECODE"]; ok {
		t.Errorf("CLAUDECODE should have been removed from agent env")
	}
}
