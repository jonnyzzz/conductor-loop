package integration_test

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

type startupScriptRun struct {
	ExitCode int
	Output   string
}

func TestStartupScriptStartConductorHelp(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("startup shell scripts are unix-focused")
	}

	scriptPath := prepareStartupScriptFixture(t, "start-conductor.sh")

	result := runStartupScript(t, scriptPath, []string{"--help"}, nil)
	if result.ExitCode != 0 {
		t.Fatalf("exit code: want 0 got %d\noutput:\n%s", result.ExitCode, result.Output)
	}
	for _, token := range []string{"Usage:", "--background", "CONDUCTOR_CONFIG"} {
		if !strings.Contains(result.Output, token) {
			t.Fatalf("help output missing %q\noutput:\n%s", token, result.Output)
		}
	}
}

func TestStartupScriptStartConductorDryRunUsesConductorBinary(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("startup shell scripts are unix-focused")
	}

	scriptPath := prepareStartupScriptFixture(t, "start-conductor.sh")
	cfgPath := writeStartupConfig(t)
	rootDir := filepath.Join(t.TempDir(), "runs")
	conductorBin := writeStartupProbeBinary(t, "conductor-bin")

	result := runStartupScript(t, scriptPath, []string{"--dry-run", "--config", cfgPath, "--root", rootDir}, map[string]string{
		"CONDUCTOR_BIN": conductorBin,
	})
	if result.ExitCode != 0 {
		t.Fatalf("exit code: want 0 got %d\noutput:\n%s", result.ExitCode, result.Output)
	}
	if !strings.Contains(result.Output, "Backend: conductor") {
		t.Fatalf("expected conductor backend\noutput:\n%s", result.Output)
	}
	if !strings.Contains(result.Output, conductorBin) {
		t.Fatalf("expected conductor binary path in output\noutput:\n%s", result.Output)
	}
}

func TestStartupScriptStartConductorDryRunFallsBackToRunAgent(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("startup shell scripts are unix-focused")
	}

	scriptPath := prepareStartupScriptFixture(t, "start-conductor.sh")
	cfgPath := writeStartupConfig(t)
	rootDir := filepath.Join(t.TempDir(), "runs")
	runAgentBin := writeStartupProbeBinary(t, "run-agent-bin")

	result := runStartupScript(t, scriptPath, []string{"--dry-run", "--config", cfgPath, "--root", rootDir}, map[string]string{
		"RUN_AGENT_BIN": runAgentBin,
		"PATH":          "/usr/bin:/bin",
	})
	if result.ExitCode != 0 {
		t.Fatalf("exit code: want 0 got %d\noutput:\n%s", result.ExitCode, result.Output)
	}
	if !strings.Contains(result.Output, "Backend: run-agent") {
		t.Fatalf("expected run-agent backend\noutput:\n%s", result.Output)
	}
	if !strings.Contains(result.Output, runAgentBin) {
		t.Fatalf("expected run-agent binary path in output\noutput:\n%s", result.Output)
	}
	if !strings.Contains(result.Output, " serve ") {
		t.Fatalf("expected serve subcommand in output\noutput:\n%s", result.Output)
	}
}

func TestStartupScriptRunAgentMonitorDryRunIncludesDisableTaskStart(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("startup shell scripts are unix-focused")
	}

	scriptPath := prepareStartupScriptFixture(t, "start-run-agent-monitor.sh")
	cfgPath := writeStartupConfig(t)
	rootDir := filepath.Join(t.TempDir(), "runs")
	runAgentBin := writeStartupProbeBinary(t, "run-agent-monitor-bin")

	result := runStartupScript(t, scriptPath, []string{"--dry-run", "--config", cfgPath, "--root", rootDir}, map[string]string{
		"RUN_AGENT_BIN": runAgentBin,
		"PATH":          "/usr/bin:/bin",
	})
	if result.ExitCode != 0 {
		t.Fatalf("exit code: want 0 got %d\noutput:\n%s", result.ExitCode, result.Output)
	}
	if !strings.Contains(result.Output, "Mode: monitor-only") {
		t.Fatalf("expected monitor-only mode message\noutput:\n%s", result.Output)
	}
	if !strings.Contains(result.Output, "--disable-task-start") {
		t.Fatalf("expected disable-task-start in command\noutput:\n%s", result.Output)
	}
}

func runStartupScript(t *testing.T, scriptPath string, args []string, env map[string]string) startupScriptRun {
	t.Helper()

	cmdArgs := append([]string{scriptPath}, args...)
	cmd := exec.Command("bash", cmdArgs...)

	homeDir := t.TempDir()
	cmd.Env = append(os.Environ(), "HOME="+homeDir)
	for k, v := range env {
		cmd.Env = append(cmd.Env, k+"="+v)
	}

	out, err := cmd.CombinedOutput()
	result := startupScriptRun{Output: string(out)}
	if err != nil {
		var exitErr *exec.ExitError
		if errors.As(err, &exitErr) {
			result.ExitCode = exitErr.ExitCode()
		} else {
			t.Fatalf("run script: %v\noutput:\n%s", err, out)
		}
	}
	return result
}

func writeStartupProbeBinary(t *testing.T, name string) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), name)
	content := "#!/usr/bin/env bash\nexit 0\n"
	if err := os.WriteFile(path, []byte(content), 0o755); err != nil {
		t.Fatalf("write probe binary: %v", err)
	}
	return path
}

func writeStartupConfig(t *testing.T) string {
	t.Helper()

	path := filepath.Join(t.TempDir(), "config.yaml")
	content := "agents: {}\ndefaults: {}\napi: {}\nstorage: {}\n"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}
	return path
}

func startupScriptsRepoRoot(t *testing.T) string {
	t.Helper()

	dir, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}

	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatalf("could not locate repo root from %s", dir)
		}
		dir = parent
	}
}

func prepareStartupScriptFixture(t *testing.T, scriptName string) string {
	t.Helper()

	repoRoot := startupScriptsRepoRoot(t)
	sourcePath := filepath.Join(repoRoot, "scripts", scriptName)
	data, err := os.ReadFile(sourcePath)
	if err != nil {
		t.Fatalf("read source script %s: %v", scriptName, err)
	}

	fixtureRoot := t.TempDir()
	scriptsDir := filepath.Join(fixtureRoot, "scripts")
	if err := os.MkdirAll(scriptsDir, 0o755); err != nil {
		t.Fatalf("mkdir scripts dir: %v", err)
	}

	destPath := filepath.Join(scriptsDir, scriptName)
	if err := os.WriteFile(destPath, data, 0o755); err != nil {
		t.Fatalf("write fixture script %s: %v", scriptName, err)
	}

	return destPath
}

func TestStartupScriptStartConductorMissingConfig(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("startup shell scripts are unix-focused")
	}

	scriptPath := prepareStartupScriptFixture(t, "start-conductor.sh")
	rootDir := filepath.Join(t.TempDir(), "runs")

	result := runStartupScript(t, scriptPath, []string{"--dry-run", "--root", rootDir}, map[string]string{
		"PATH": "/usr/bin:/bin",
	})
	if result.ExitCode == 0 {
		t.Fatalf("expected non-zero exit code\noutput:\n%s", result.Output)
	}
	if !strings.Contains(result.Output, "config file not found") {
		t.Fatalf("expected config preflight message\noutput:\n%s", result.Output)
	}
}

func TestStartupScriptRunAgentMonitorMissingBinary(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("startup shell scripts are unix-focused")
	}

	scriptPath := prepareStartupScriptFixture(t, "start-run-agent-monitor.sh")
	cfgPath := writeStartupConfig(t)
	rootDir := filepath.Join(t.TempDir(), "runs")

	result := runStartupScript(t, scriptPath, []string{"--dry-run", "--config", cfgPath, "--root", rootDir}, map[string]string{
		"PATH": "/usr/bin:/bin",
	})
	if result.ExitCode == 0 {
		t.Fatalf("expected non-zero exit code\noutput:\n%s", result.Output)
	}
	if !strings.Contains(result.Output, "run-agent binary not found") {
		t.Fatalf("expected missing-binary message\noutput:\n%s", result.Output)
	}
}

func TestStartupScriptStartConductorAllowsExtraArgs(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("startup shell scripts are unix-focused")
	}

	scriptPath := prepareStartupScriptFixture(t, "start-conductor.sh")
	cfgPath := writeStartupConfig(t)
	rootDir := filepath.Join(t.TempDir(), "runs")
	conductorBin := writeStartupProbeBinary(t, "conductor-extra")

	extraKey := "--disable-task-start"
	extraVal := "--api-key=test"
	result := runStartupScript(t, scriptPath, []string{"--dry-run", "--config", cfgPath, "--root", rootDir, "--", extraKey, extraVal}, map[string]string{
		"CONDUCTOR_BIN": conductorBin,
	})
	if result.ExitCode != 0 {
		t.Fatalf("exit code: want 0 got %d\noutput:\n%s", result.ExitCode, result.Output)
	}
	if !strings.Contains(result.Output, fmt.Sprintf(" %s", extraKey)) {
		t.Fatalf("expected extra arg %q\noutput:\n%s", extraKey, result.Output)
	}
	if !strings.Contains(result.Output, fmt.Sprintf(" %s", extraVal)) {
		t.Fatalf("expected extra arg %q\noutput:\n%s", extraVal, result.Output)
	}
}
