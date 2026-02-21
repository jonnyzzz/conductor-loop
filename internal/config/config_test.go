package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestTokenEnvVarName(t *testing.T) {
	got := tokenEnvVarName("claude-1")
	if got != "CONDUCTOR_AGENT_CLAUDE_1_TOKEN" {
		t.Fatalf("unexpected env var: %q", got)
	}
}

func TestResolvePath(t *testing.T) {
	base := t.TempDir()
	resolved, err := resolvePath(base, "relative/path.txt")
	if err != nil {
		t.Fatalf("resolvePath: %v", err)
	}
	if !filepath.IsAbs(resolved) {
		t.Fatalf("expected absolute path, got %q", resolved)
	}
}

func TestExpandHome(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("UserHomeDir: %v", err)
	}
	got, err := expandHome("~")
	if err != nil {
		t.Fatalf("expandHome: %v", err)
	}
	if got != home {
		t.Fatalf("expected %q, got %q", home, got)
	}
	if _, err := expandHome("~user"); err == nil {
		t.Fatalf("expected error for unsupported home expansion")
	}
}

func TestReadTokenFileEmpty(t *testing.T) {
	path := filepath.Join(t.TempDir(), "token.txt")
	if err := os.WriteFile(path, []byte("\n"), 0o600); err != nil {
		t.Fatalf("write token: %v", err)
	}
	if _, err := readTokenFile(path); err == nil {
		t.Fatalf("expected error for empty token file")
	}
}

func TestApplyAPIDefaults(t *testing.T) {
	cfg := &Config{}
	applyAPIDefaults(cfg)
	if cfg.API.Host != "0.0.0.0" {
		t.Fatalf("expected default host, got %q", cfg.API.Host)
	}
	if cfg.API.Port != 14355 {
		t.Fatalf("expected default port, got %d", cfg.API.Port)
	}
	if cfg.API.SSE.PollIntervalMs != 100 {
		t.Fatalf("expected default poll interval, got %d", cfg.API.SSE.PollIntervalMs)
	}
}

func TestValidateConfigErrors(t *testing.T) {
	if err := ValidateConfig(nil); err == nil {
		t.Fatalf("expected error for nil config")
	}
	cfg := &Config{
		Agents: map[string]AgentConfig{
			"bad": {Type: "unknown", Token: "x"},
		},
		Defaults: DefaultConfig{Timeout: 1},
	}
	if err := ValidateConfig(cfg); err == nil {
		t.Fatalf("expected error for invalid agent type")
	}
	cfg = &Config{
		Agents: map[string]AgentConfig{
			"codex": {Type: "codex", Token: "x"},
		},
		Defaults: DefaultConfig{Timeout: 1},
		API:      APIConfig{SSE: SSEConfig{PollIntervalMs: -1}},
	}
	if err := ValidateConfig(cfg); err == nil {
		t.Fatalf("expected error for negative poll interval")
	}
	cfg = &Config{
		Agents: map[string]AgentConfig{
			"codex": {Type: "codex", Token: "x"},
		},
		Defaults: DefaultConfig{Timeout: 1},
		API:      APIConfig{Port: 70000},
	}
	if err := ValidateConfig(cfg); err == nil {
		t.Fatalf("expected error for invalid port")
	}
}

func TestValidateConfigNoToken(t *testing.T) {
	cfg := &Config{
		Agents: map[string]AgentConfig{
			"claude": {Type: "claude"},
		},
		Defaults: DefaultConfig{Timeout: 1},
	}
	if err := ValidateConfig(cfg); err != nil {
		t.Fatalf("expected no error for agent without token, got %v", err)
	}
}

func TestResolveTokensFromFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "token.txt")
	if err := os.WriteFile(path, []byte("secret"), 0o600); err != nil {
		t.Fatalf("write token: %v", err)
	}
	cfg := &Config{Agents: map[string]AgentConfig{"claude": {Type: "claude", TokenFile: path}}}
	if err := resolveTokens(cfg); err != nil {
		t.Fatalf("resolveTokens: %v", err)
	}
	agent := cfg.Agents["claude"]
	if agent.Token != "secret" || !agent.tokenFromFile {
		t.Fatalf("expected token from file, got %+v", agent)
	}
}

func TestApplyTokenEnvOverrides(t *testing.T) {
	cfg := &Config{Agents: map[string]AgentConfig{"claude": {Type: "claude", TokenFile: "file.txt"}}}
	t.Setenv("CONDUCTOR_AGENT_CLAUDE_TOKEN", " env-token ")
	applyTokenEnvOverrides(cfg)
	agent := cfg.Agents["claude"]
	if agent.Token != "env-token" {
		t.Fatalf("expected env token, got %q", agent.Token)
	}
	if agent.TokenFile != "" {
		t.Fatalf("expected token_file cleared, got %q", agent.TokenFile)
	}
}

func TestNormalizeAgentType(t *testing.T) {
	if got := normalizeAgentType(" Claude "); got != "claude" {
		t.Fatalf("unexpected normalization: %q", got)
	}
}

func TestLoadConfigSuccess(t *testing.T) {
	dir := t.TempDir()
	tokenPath := filepath.Join(dir, "token.txt")
	if err := os.WriteFile(tokenPath, []byte("secret"), 0o600); err != nil {
		t.Fatalf("write token: %v", err)
	}
	configPath := filepath.Join(dir, "config.yaml")
	content := `agents:
  claude:
    type: claude
    token_file: token.txt

defaults:
  agent: claude
  timeout: 10
`
	if err := os.WriteFile(configPath, []byte(content), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	cfg, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	if cfg.Agents["claude"].Token != "secret" {
		t.Fatalf("expected token from file")
	}
}

func TestValidateTokenFileNotRegular(t *testing.T) {
	dir := t.TempDir()
	cfg := &Config{
		Agents: map[string]AgentConfig{
			"claude": {Type: "claude", TokenFile: dir},
		},
		Defaults: DefaultConfig{Timeout: 1},
	}
	if err := ValidateConfig(cfg); err == nil {
		t.Fatalf("expected error for non-regular token file")
	}
}

func TestResolvePathHome(t *testing.T) {
	home, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("UserHomeDir: %v", err)
	}
	resolved, err := resolvePath("/tmp", "~/token.txt")
	if err != nil {
		t.Fatalf("resolvePath: %v", err)
	}
	if !strings.HasPrefix(resolved, home) {
		t.Fatalf("expected home prefix, got %q", resolved)
	}
}

func TestLoadConfigForServer(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.yaml")
	content := `agents:
  claude:
    type: claude

defaults:
  agent: claude
  timeout: 10
`
	if err := os.WriteFile(configPath, []byte(content), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	cfg, err := LoadConfigForServer(configPath)
	if err != nil {
		t.Fatalf("LoadConfigForServer: %v", err)
	}
	if cfg == nil || cfg.Agents["claude"].Type != "claude" {
		t.Fatalf("unexpected config: %+v", cfg)
	}
}

func TestResolveStoragePaths(t *testing.T) {
	cfg := &Config{Storage: StorageConfig{RunsDir: "runs"}}
	if err := resolveStoragePaths(cfg, t.TempDir()); err != nil {
		t.Fatalf("resolveStoragePaths: %v", err)
	}
	if !filepath.IsAbs(cfg.Storage.RunsDir) {
		t.Fatalf("expected absolute runs dir, got %q", cfg.Storage.RunsDir)
	}
}

func TestFindDefaultConfig_NotFound(t *testing.T) {
	dir := t.TempDir()
	path, err := FindDefaultConfigIn(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if path != "" {
		t.Fatalf("expected empty path, got %q", path)
	}
}

func TestFindDefaultConfig_FoundYaml(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.yaml")
	if err := os.WriteFile(configPath, []byte("agents: {}\n"), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	path, err := FindDefaultConfigIn(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if path != configPath {
		t.Fatalf("expected %q, got %q", configPath, path)
	}
}

func TestFindDefaultConfig_FoundHome(t *testing.T) {
	dir := t.TempDir()
	homeConfigDir := filepath.Join(dir, ".config", "conductor")
	if err := os.MkdirAll(homeConfigDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	configPath := filepath.Join(homeConfigDir, "config.yaml")
	if err := os.WriteFile(configPath, []byte("agents: {}\n"), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	// Use a separate baseDir with no local config so home path is reached.
	baseDir := t.TempDir()

	// Temporarily override HOME so FindDefaultConfigIn picks up our fake home.
	t.Setenv("HOME", dir)

	path, err := FindDefaultConfigIn(baseDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if path != configPath {
		t.Fatalf("expected %q, got %q", configPath, path)
	}
}

func TestFindDefaultConfig_FoundHCL(t *testing.T) {
	dir := t.TempDir()
	hclPath := filepath.Join(dir, "config.hcl")
	if err := os.WriteFile(hclPath, []byte("# hcl config\n"), 0o600); err != nil {
		t.Fatalf("write hcl: %v", err)
	}
	path, err := FindDefaultConfigIn(dir)
	if err != nil {
		t.Fatalf("unexpected error for HCL config: %v", err)
	}
	if path != hclPath {
		t.Fatalf("expected %q, got %q", hclPath, path)
	}
}

func TestLoadHCLConfig(t *testing.T) {
	dir := t.TempDir()
	tokenPath := filepath.Join(dir, "token.txt")
	if err := os.WriteFile(tokenPath, []byte("hcl-secret"), 0o600); err != nil {
		t.Fatalf("write token: %v", err)
	}
	content := `
agents {
  claude {
    type       = "claude"
    token_file = "token.txt"
  }
}

defaults {
  agent   = "claude"
  timeout = 10
}

storage {
  runs_dir = "./runs"
}
`
	configPath := filepath.Join(dir, "config.hcl")
	if err := os.WriteFile(configPath, []byte(content), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	cfg, err := LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig HCL: %v", err)
	}
	agent, ok := cfg.Agents["claude"]
	if !ok {
		t.Fatalf("expected claude agent in config")
	}
	if agent.Type != "claude" {
		t.Fatalf("expected type claude, got %q", agent.Type)
	}
	if agent.Token != "hcl-secret" {
		t.Fatalf("expected token from file, got %q", agent.Token)
	}
	if cfg.Defaults.Agent != "claude" {
		t.Fatalf("expected default agent claude, got %q", cfg.Defaults.Agent)
	}
	if cfg.Defaults.Timeout != 10 {
		t.Fatalf("expected timeout 10, got %d", cfg.Defaults.Timeout)
	}
}

func TestLoadConfigAutoDetectsFormat(t *testing.T) {
	dir := t.TempDir()

	// Write YAML config
	yamlPath := filepath.Join(dir, "config.yaml")
	yamlContent := `agents:
  codex:
    type: codex
    token: yaml-token
defaults:
  agent: codex
  timeout: 5
`
	if err := os.WriteFile(yamlPath, []byte(yamlContent), 0o600); err != nil {
		t.Fatalf("write yaml: %v", err)
	}

	// Write HCL config
	hclPath := filepath.Join(dir, "config.hcl")
	hclContent := `
agents {
  codex {
    type  = "codex"
    token = "hcl-token"
  }
}
defaults {
  agent   = "codex"
  timeout = 5
}
`
	if err := os.WriteFile(hclPath, []byte(hclContent), 0o600); err != nil {
		t.Fatalf("write hcl: %v", err)
	}

	yamlCfg, err := LoadConfig(yamlPath)
	if err != nil {
		t.Fatalf("LoadConfig YAML: %v", err)
	}
	if yamlCfg.Agents["codex"].Token != "yaml-token" {
		t.Fatalf("expected yaml-token, got %q", yamlCfg.Agents["codex"].Token)
	}

	hclCfg, err := LoadConfig(hclPath)
	if err != nil {
		t.Fatalf("LoadConfig HCL: %v", err)
	}
	if hclCfg.Agents["codex"].Token != "hcl-token" {
		t.Fatalf("expected hcl-token, got %q", hclCfg.Agents["codex"].Token)
	}
}

func TestLoadHCLConfigInvalidSyntax(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.hcl")
	if err := os.WriteFile(configPath, []byte(":::invalid hcl:::"), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	_, err := LoadConfig(configPath)
	if err == nil {
		t.Fatalf("expected error for invalid HCL, got nil")
	}
	if !strings.Contains(err.Error(), "config.hcl") {
		t.Fatalf("expected error to mention file path, got: %v", err)
	}
}

func TestLoadConfigErrors(t *testing.T) {
	if _, err := LoadConfig(""); err == nil {
		t.Fatalf("expected error for empty path")
	}
	path := filepath.Join(t.TempDir(), "bad.yaml")
	if err := os.WriteFile(path, []byte("::bad"), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	if _, err := LoadConfig(path); err == nil {
		t.Fatalf("expected error for invalid yaml")
	}
}
