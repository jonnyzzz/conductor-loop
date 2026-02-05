package runner

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/jonnyzzz/conductor-loop/internal/config"
)

func TestResolveRootDirDefault(t *testing.T) {
	root, err := resolveRootDir("")
	if err != nil {
		t.Fatalf("resolveRootDir: %v", err)
	}
	if !strings.Contains(root, "run-agent") {
		t.Fatalf("expected run-agent in path, got %q", root)
	}
}

func TestResolveTaskDirValidation(t *testing.T) {
	if _, err := resolveTaskDir("/tmp", "", "task"); err == nil {
		t.Fatalf("expected error for empty project id")
	}
	if _, err := resolveTaskDir("/tmp", "project", ""); err == nil {
		t.Fatalf("expected error for empty task id")
	}
	if _, err := resolveTaskDir("", "project", "task"); err == nil {
		t.Fatalf("expected error for empty root dir")
	}
}

func TestSelectAgent(t *testing.T) {
	cfg := &config.Config{
		Agents: map[string]config.AgentConfig{
			"claude": {Type: "claude", Token: "x"},
			"codex":  {Type: "codex", Token: "y"},
		},
		Defaults: config.DefaultConfig{Agent: "codex", Timeout: 1},
	}
	selection, err := selectAgent(cfg, "")
	if err != nil {
		t.Fatalf("selectAgent: %v", err)
	}
	if selection.Name != "codex" {
		t.Fatalf("expected codex, got %q", selection.Name)
	}
	selection, err = selectAgent(cfg, "claude")
	if err != nil || selection.Type != "claude" {
		t.Fatalf("expected claude selection")
	}
	selection, err = selectAgent(cfg, "codex")
	if err != nil || selection.Type != "codex" {
		t.Fatalf("expected codex selection")
	}
	if _, err := selectAgent(cfg, "unknown"); err == nil {
		t.Fatalf("expected error for unknown agent")
	}
}

func TestSelectAgentNilConfig(t *testing.T) {
	if _, err := selectAgent(nil, ""); err == nil {
		t.Fatalf("expected error for empty agent")
	}
	selection, err := selectAgent(nil, "Codex")
	if err != nil {
		t.Fatalf("selectAgent: %v", err)
	}
	if selection.Name != "Codex" || selection.Type != "codex" {
		t.Fatalf("unexpected selection: %+v", selection)
	}
}

func TestSelectAgentByType(t *testing.T) {
	cfg := &config.Config{
		Agents: map[string]config.AgentConfig{
			"primary": {Type: "codex"},
			"backup":  {Type: "claude"},
		},
	}
	selection, err := selectAgent(cfg, "codex")
	if err != nil {
		t.Fatalf("selectAgent: %v", err)
	}
	if selection.Name != "primary" {
		t.Fatalf("expected primary agent, got %q", selection.Name)
	}
}

func TestSelectAgentNoAgents(t *testing.T) {
	cfg := &config.Config{}
	if _, err := selectAgent(cfg, ""); err == nil {
		t.Fatalf("expected error for no agents configured")
	}
	cfg.Defaults.Agent = "missing"
	if _, err := selectAgent(cfg, ""); err == nil {
		t.Fatalf("expected error for default agent not found")
	}
}

func TestBuildPrompt(t *testing.T) {
	prompt := buildPrompt("/task", "/run", "do something")
	if !strings.Contains(prompt, "TASK_FOLDER=/task") || !strings.Contains(prompt, "RUN_FOLDER=/run") {
		t.Fatalf("missing preamble: %q", prompt)
	}
	if !strings.HasSuffix(prompt, "\n") {
		t.Fatalf("expected trailing newline")
	}
}

func TestEnsureDirValidation(t *testing.T) {
	if err := ensureDir(""); err == nil {
		t.Fatalf("expected error for empty dir")
	}
}

func TestReadFileTrimmed(t *testing.T) {
	path := filepath.Join(t.TempDir(), "file.txt")
	if err := os.WriteFile(path, []byte("hello\n"), 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}
	text, err := readFileTrimmed(path)
	if err != nil {
		t.Fatalf("readFileTrimmed: %v", err)
	}
	if text != "hello" {
		t.Fatalf("unexpected content: %q", text)
	}
	if err := os.WriteFile(path, []byte("\n"), 0o644); err != nil {
		t.Fatalf("write empty: %v", err)
	}
	if _, err := readFileTrimmed(path); err == nil {
		t.Fatalf("expected error for empty file")
	}
}

func TestTokenEnvVar(t *testing.T) {
	cases := map[string]string{
		"codex":      "OPENAI_API_KEY",
		"claude":     "ANTHROPIC_API_KEY",
		"gemini":     "GEMINI_API_KEY",
		"perplexity": "PERPLEXITY_API_KEY",
		"xai":        "XAI_API_KEY",
	}
	for agentType, env := range cases {
		if got := tokenEnvVar(agentType); got != env {
			t.Fatalf("expected %s for %s, got %s", env, agentType, got)
		}
	}
	if got := tokenEnvVar("unknown"); got != "" {
		t.Fatalf("expected empty for unknown, got %q", got)
	}
}

func TestMergeEnv(t *testing.T) {
	base := []string{"A=1", "B=2"}
	merged := mergeEnv(base, map[string]string{"B": "3", "C": "4"})
	joined := strings.Join(merged, ";")
	if !strings.Contains(joined, "B=3") || !strings.Contains(joined, "C=4") {
		t.Fatalf("unexpected merged env: %v", merged)
	}
}

func TestPrependPath(t *testing.T) {
	env := map[string]string{}
	if err := prependPath(env); err != nil {
		t.Fatalf("prependPath: %v", err)
	}
	if env["PATH"] == "" {
		t.Fatalf("expected PATH set")
	}
}

func TestPrependPathNilEnv(t *testing.T) {
	if err := prependPath(nil); err == nil {
		t.Fatalf("expected error for nil env")
	}
}

func TestAbsPathValidation(t *testing.T) {
	if _, err := absPath(""); err == nil {
		t.Fatalf("expected error for empty path")
	}
	abs, err := absPath(".")
	if err != nil {
		t.Fatalf("absPath: %v", err)
	}
	if abs == "." {
		t.Fatalf("expected absolute path")
	}
}

func TestNewRunIDFormat(t *testing.T) {
	now := time.Now().UTC()
	id := newRunID(now, 123)
	if !strings.Contains(id, "123") {
		t.Fatalf("expected pid in run id, got %q", id)
	}
}

func TestLoadConfig(t *testing.T) {
	path := filepath.Join(t.TempDir(), "config.yaml")
	content := `agents:
  codex:
    type: codex
    token: token

defaults:
  agent: codex
  timeout: 10
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}
	cfg, err := loadConfig(path)
	if err != nil {
		t.Fatalf("loadConfig: %v", err)
	}
	if cfg == nil || cfg.Defaults.Agent != "codex" {
		t.Fatalf("unexpected config")
	}
}

func TestLoadConfigEmpty(t *testing.T) {
	cfg, err := loadConfig("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg != nil {
		t.Fatalf("expected nil config for empty path")
	}
}

func TestCreateRunDir(t *testing.T) {
	if _, _, err := createRunDir(""); err == nil {
		t.Fatalf("expected error for empty runs dir")
	}
	runsDir := filepath.Join(t.TempDir(), "runs")
	if err := os.MkdirAll(runsDir, 0o755); err != nil {
		t.Fatalf("mkdir runs: %v", err)
	}
	runID, runDir, err := createRunDir(runsDir)
	if err != nil {
		t.Fatalf("createRunDir: %v", err)
	}
	if runID == "" || runDir == "" {
		t.Fatalf("expected run id/dir")
	}
	if _, err := os.Stat(runDir); err != nil {
		t.Fatalf("expected run dir: %v", err)
	}
}

func TestCreateRunDirError(t *testing.T) {
	root := t.TempDir()
	blocker := filepath.Join(root, "runs")
	if err := os.WriteFile(blocker, []byte("x"), 0o644); err != nil {
		t.Fatalf("write blocker: %v", err)
	}
	if _, _, err := createRunDir(blocker); err == nil {
		t.Fatalf("expected error for invalid runs dir")
	}
}
