package integration_test

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

type runAgentCmdProbe struct {
	Args       []string `json:"args"`
	Executable string   `json:"executable"`
	Label      string   `json:"label"`
	Launcher   string   `json:"launcher"`
}

type runAgentCmdRun struct {
	ExitCode int
	Output   string
	Probe    *runAgentCmdProbe
}

func TestRunAgentCmdContainsEmbeddedWindowsLauncher(t *testing.T) {
	launcherPath := filepath.Join(runAgentCmdRepoRoot(t), "run-agent.cmd")
	data, err := os.ReadFile(launcherPath)
	if err != nil {
		t.Fatalf("read run-agent.cmd: %v", err)
	}
	text := string(data)

	for _, marker := range []string{
		":CMDSCRIPT",
		"Resolve-RunAgentBinary",
		"Set-StrictMode -Version 3.0",
		"RUN_AGENT_CMD_DISABLE_PATH",
	} {
		if !strings.Contains(text, marker) {
			t.Fatalf("expected run-agent.cmd to contain %q", marker)
		}
	}
	if strings.Contains(text, "run-agent.ps1") {
		t.Fatal("run-agent.cmd must be self-contained and not call an external run-agent.ps1")
	}
}

func TestRunAgentCmdShellPrefersSiblingBinary(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell execution path test")
	}

	fixture := prepareRunAgentCmdFixture(t)
	buildRunAgentProbeBinary(t, filepath.Join(fixture.dir, "run-agent"), "sibling")

	result := runAgentCmdViaBash(t, fixture.launcherPath, []string{"alpha", "two words"}, map[string]string{
		"RUN_AGENT_CMD_DISABLE_PATH": "1",
	})

	if result.ExitCode != 0 {
		t.Fatalf("exit code: want 0 got %d\noutput:\n%s", result.ExitCode, result.Output)
	}
	if result.Probe == nil {
		t.Fatal("expected probe payload")
	}
	if result.Probe.Label != "sibling" {
		t.Fatalf("label: want sibling got %q", result.Probe.Label)
	}
	if got := result.Probe.Args; len(got) != 2 || got[0] != "alpha" || got[1] != "two words" {
		t.Fatalf("unexpected args: %#v", got)
	}
	if result.Probe.Launcher != "run-agent.cmd" {
		t.Fatalf("launcher env: want run-agent.cmd got %q", result.Probe.Launcher)
	}
}

func TestRunAgentCmdShellUsesDistBinaryWhenSiblingMissing(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell execution path test")
	}

	fixture := prepareRunAgentCmdFixture(t)
	assetName := runAgentCmdUnixAssetName(t)
	distDir := filepath.Join(fixture.dir, "dist")
	if err := os.MkdirAll(distDir, 0o755); err != nil {
		t.Fatalf("mkdir dist dir: %v", err)
	}
	buildRunAgentProbeBinary(t, filepath.Join(distDir, assetName), "dist")

	result := runAgentCmdViaBash(t, fixture.launcherPath, []string{"dist-mode"}, map[string]string{
		"RUN_AGENT_CMD_DISABLE_PATH": "1",
	})

	if result.ExitCode != 0 {
		t.Fatalf("exit code: want 0 got %d\noutput:\n%s", result.ExitCode, result.Output)
	}
	if result.Probe == nil {
		t.Fatal("expected probe payload")
	}
	if result.Probe.Label != "dist" {
		t.Fatalf("label: want dist got %q", result.Probe.Label)
	}
}

func TestRunAgentCmdShellRunAgentBinOverride(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell execution path test")
	}

	fixture := prepareRunAgentCmdFixture(t)
	buildRunAgentProbeBinary(t, filepath.Join(fixture.dir, "run-agent"), "sibling")
	overridePath := filepath.Join(t.TempDir(), "custom-run-agent")
	buildRunAgentProbeBinary(t, overridePath, "override")

	result := runAgentCmdViaBash(t, fixture.launcherPath, []string{"override"}, map[string]string{
		"RUN_AGENT_BIN":              overridePath,
		"RUN_AGENT_CMD_DISABLE_PATH": "1",
	})

	if result.ExitCode != 0 {
		t.Fatalf("exit code: want 0 got %d\noutput:\n%s", result.ExitCode, result.Output)
	}
	if result.Probe == nil {
		t.Fatal("expected probe payload")
	}
	if result.Probe.Label != "override" {
		t.Fatalf("label: want override got %q", result.Probe.Label)
	}
}

func TestRunAgentCmdShellMissingBinaryReturnsError(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell execution path test")
	}

	fixture := prepareRunAgentCmdFixture(t)
	result := runAgentCmdViaBash(t, fixture.launcherPath, []string{"--version"}, map[string]string{
		"RUN_AGENT_CMD_DISABLE_PATH": "1",
	})

	if result.ExitCode == 0 {
		t.Fatalf("expected non-zero exit code\noutput:\n%s", result.Output)
	}
	if !strings.Contains(result.Output, "run-agent binary not found") {
		t.Fatalf("expected missing-binary message, got:\n%s", result.Output)
	}
	if !strings.Contains(result.Output, "PATH fallback disabled: 1") {
		t.Fatalf("expected PATH fallback detail, got:\n%s", result.Output)
	}
}

func TestRunAgentCmdShellPropagatesExitCode(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell execution path test")
	}

	fixture := prepareRunAgentCmdFixture(t)
	buildRunAgentProbeBinary(t, filepath.Join(fixture.dir, "run-agent"), "exit-code")

	result := runAgentCmdViaBash(t, fixture.launcherPath, []string{"code"}, map[string]string{
		"RUN_AGENT_CMD_DISABLE_PATH": "1",
		"RUN_AGENT_TEST_EXIT":        "37",
	})

	if result.ExitCode != 37 {
		t.Fatalf("exit code: want 37 got %d\noutput:\n%s", result.ExitCode, result.Output)
	}
	if result.Probe == nil {
		t.Fatal("expected probe payload")
	}
	if result.Probe.Label != "exit-code" {
		t.Fatalf("label: want exit-code got %q", result.Probe.Label)
	}
}

func prepareRunAgentCmdFixture(t *testing.T) struct {
	dir          string
	launcherPath string
} {
	t.Helper()

	dir := t.TempDir()
	src := filepath.Join(runAgentCmdRepoRoot(t), "run-agent.cmd")
	dst := filepath.Join(dir, "run-agent.cmd")
	data, err := os.ReadFile(src)
	if err != nil {
		t.Fatalf("read launcher: %v", err)
	}
	if err := os.WriteFile(dst, data, 0o755); err != nil {
		t.Fatalf("write launcher fixture: %v", err)
	}

	return struct {
		dir          string
		launcherPath string
	}{
		dir:          dir,
		launcherPath: dst,
	}
}

func runAgentCmdViaBash(t *testing.T, launcherPath string, args []string, env map[string]string) runAgentCmdRun {
	t.Helper()

	probeOut := filepath.Join(t.TempDir(), "probe.json")
	cmdArgs := append([]string{launcherPath}, args...)
	cmd := exec.Command("bash", cmdArgs...)
	cmd.Dir = filepath.Dir(launcherPath)
	cmd.Env = append(os.Environ(), "RUN_AGENT_TEST_OUTPUT="+probeOut)
	for k, v := range env {
		cmd.Env = append(cmd.Env, k+"="+v)
	}

	out, err := cmd.CombinedOutput()
	result := runAgentCmdRun{
		Output: string(out),
	}
	if err != nil {
		var exitErr *exec.ExitError
		if !strings.Contains(err.Error(), "exit status") {
			t.Fatalf("run launcher: %v\noutput:\n%s", err, out)
		}
		if ok := errors.As(err, &exitErr); ok {
			result.ExitCode = exitErr.ExitCode()
		} else {
			result.ExitCode = 1
		}
	}

	data, readErr := os.ReadFile(probeOut)
	if readErr == nil {
		var probe runAgentCmdProbe
		if err := json.Unmarshal(data, &probe); err != nil {
			t.Fatalf("decode probe payload: %v\nraw: %s", err, data)
		}
		result.Probe = &probe
	}

	return result
}

func buildRunAgentProbeBinary(t *testing.T, outputPath string, label string) {
	t.Helper()

	src := fmt.Sprintf(`package main

import (
	"encoding/json"
	"os"
	"strconv"
)

const label = %q

type payload struct {
	Args       []string `+"`json:\"args\"`"+`
	Executable string   `+"`json:\"executable\"`"+`
	Label      string   `+"`json:\"label\"`"+`
	Launcher   string   `+"`json:\"launcher\"`"+`
}

func main() {
	p := payload{
		Args:       os.Args[1:],
		Executable: os.Args[0],
		Label:      label,
		Launcher:   os.Getenv("RUN_AGENT_LAUNCHER"),
	}
	if outPath := os.Getenv("RUN_AGENT_TEST_OUTPUT"); outPath != "" {
		f, err := os.Create(outPath)
		if err == nil {
			_ = json.NewEncoder(f).Encode(&p)
			_ = f.Close()
		}
	}
	if raw := os.Getenv("RUN_AGENT_TEST_EXIT"); raw != "" {
		if code, err := strconv.Atoi(raw); err == nil {
			os.Exit(code)
		}
	}
}
`, label)

	dir := filepath.Dir(outputPath)
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir probe dir: %v", err)
	}
	srcPath := filepath.Join(dir, "run_agent_probe_main.go")
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		t.Fatalf("write probe source: %v", err)
	}

	cmd := exec.Command("go", "build", "-o", outputPath, srcPath)
	cmd.Env = os.Environ()
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("build probe: %v\n%s", err, out)
	}
}

func runAgentCmdUnixAssetName(t *testing.T) string {
	t.Helper()

	var goos string
	switch runtime.GOOS {
	case "linux", "darwin":
		goos = runtime.GOOS
	default:
		t.Skipf("unsupported test OS: %s", runtime.GOOS)
	}

	var goarch string
	switch runtime.GOARCH {
	case "amd64", "arm64":
		goarch = runtime.GOARCH
	default:
		t.Skipf("unsupported test architecture: %s", runtime.GOARCH)
	}

	return "run-agent-" + goos + "-" + goarch
}

func runAgentCmdRepoRoot(t *testing.T) string {
	t.Helper()

	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
}
