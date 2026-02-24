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
	if cfg.API.SSE.PollIntervalMs != 500 {
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
	baseDir := t.TempDir()
	// Point HOME at an empty temp dir so ~/.run-agent/conductor-loop.hcl is absent.
	t.Setenv("HOME", t.TempDir())
	path, err := FindDefaultConfigIn(baseDir)
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
	// Verify that ~/.run-agent/conductor-loop.hcl is discovered when it exists.
	dir := t.TempDir()
	runAgentDir := filepath.Join(dir, ".run-agent")
	if err := os.MkdirAll(runAgentDir, 0o700); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	configPath := filepath.Join(runAgentDir, "conductor-loop.hcl")
	if err := os.WriteFile(configPath, []byte("# empty\n"), 0o600); err != nil {
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

func TestFindDefaultConfig_ProjectHCLNotFound(t *testing.T) {
	// Only ~/.run-agent/conductor-loop.hcl is supported as HCL config.
	// A config.hcl in a project directory must NOT be discovered.
	dir := t.TempDir()
	hclPath := filepath.Join(dir, "config.hcl")
	if err := os.WriteFile(hclPath, []byte("# hcl config\n"), 0o600); err != nil {
		t.Fatalf("write hcl: %v", err)
	}
	path, err := FindDefaultConfigIn(dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// The real ~/.run-agent/conductor-loop.hcl may exist on the test host,
	// so we only assert that config.hcl from the project dir was NOT returned.
	if path == hclPath {
		t.Fatalf("project-level config.hcl must not be discovered, got %q", path)
	}
}

func TestEnsureHomeHCLConfig_CreatesFile(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".run-agent", "conductor-loop.hcl")

	got, err := ensureHomeHCLConfigAt(path)
	if err != nil {
		t.Fatalf("ensureHomeHCLConfigAt: %v", err)
	}
	if got != path {
		t.Fatalf("expected path %q, got %q", path, got)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read created file: %v", err)
	}
	if len(data) == 0 {
		t.Fatal("expected non-empty template")
	}
	content := string(data)
	if !strings.Contains(content, "github.com/jonnyzzz/conductor-loop") {
		t.Fatal("expected GitHub URL in template")
	}
	if !strings.Contains(strings.ToLower(content), "documentation") {
		t.Fatal("expected documentation reference in template")
	}

	// Permissions should be 0600.
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat: %v", err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("expected mode 0600, got %04o", info.Mode().Perm())
	}
}

func TestEnsureHomeHCLConfig_ExistingFileUntouched(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "conductor-loop.hcl")
	original := "# my custom config\n"
	if err := os.WriteFile(path, []byte(original), 0o600); err != nil {
		t.Fatalf("write: %v", err)
	}

	if _, err := ensureHomeHCLConfigAt(path); err != nil {
		t.Fatalf("ensureHomeHCLConfigAt: %v", err)
	}

	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if string(data) != original {
		t.Fatalf("existing file was modified: got %q", string(data))
	}
}

func TestFindDefaultConfigIn_HomeHCL(t *testing.T) {
	// FindDefaultConfigIn must discover ~/.run-agent/conductor-loop.hcl when
	// no project-local config is present.
	home, err := os.UserHomeDir()
	if err != nil {
		t.Skip("cannot determine home dir")
	}
	homeHCL := filepath.Join(home, ".run-agent", "conductor-loop.hcl")

	// Create a temp baseDir with no project config so home config is reached.
	baseDir := t.TempDir()
	found, err := FindDefaultConfigIn(baseDir)
	if err != nil {
		t.Fatalf("FindDefaultConfigIn: %v", err)
	}

	// If the home HCL exists it must be the result (no other home candidates).
	if _, statErr := os.Stat(homeHCL); statErr == nil {
		if found != homeHCL {
			t.Fatalf("expected %q, got %q", homeHCL, found)
		}
	}
}

func TestParseHCLConfig_AgentInference(t *testing.T) {
	hcl := `
# conductor home config
codex {
  token_file = "~/.tokens/codex"
  model = "codex-davinci-003"
}

claude {
  token_file = "~/.tokens/claude"
}

defaults {
  agent = "codex"
  timeout = 120
  max_concurrent_runs = 4
}

api {
  port = 14355
}
`
	cfg, err := parseHCLConfig([]byte(hcl))
	if err != nil {
		t.Fatalf("parseHCLConfig: %v", err)
	}

	if len(cfg.Agents) != 2 {
		t.Fatalf("expected 2 agents, got %d", len(cfg.Agents))
	}

	codex, ok := cfg.Agents["codex"]
	if !ok {
		t.Fatal("expected codex agent")
	}
	if codex.TokenFile != "~/.tokens/codex" {
		t.Fatalf("expected token_file %q, got %q", "~/.tokens/codex", codex.TokenFile)
	}
	if codex.Model != "codex-davinci-003" {
		t.Fatalf("expected model %q, got %q", "codex-davinci-003", codex.Model)
	}
	// Type not set in HCL — will be inferred from block name by applyAgentDefaults.
	if codex.Type != "" {
		t.Fatalf("expected empty type before inference, got %q", codex.Type)
	}

	if cfg.Defaults.Agent != "codex" {
		t.Fatalf("expected defaults.agent %q, got %q", "codex", cfg.Defaults.Agent)
	}
	if cfg.Defaults.Timeout != 120 {
		t.Fatalf("expected defaults.timeout 120, got %d", cfg.Defaults.Timeout)
	}
	if cfg.Defaults.MaxConcurrentRuns != 4 {
		t.Fatalf("expected max_concurrent_runs 4, got %d", cfg.Defaults.MaxConcurrentRuns)
	}
	if cfg.API.Port != 14355 {
		t.Fatalf("expected api.port 14355, got %d", cfg.API.Port)
	}
}

func TestParseHCLConfig_ExplicitType(t *testing.T) {
	hcl := `
my_openai {
  type = "codex"
  token_file = "~/.tokens/openai"
}
`
	cfg, err := parseHCLConfig([]byte(hcl))
	if err != nil {
		t.Fatalf("parseHCLConfig: %v", err)
	}
	agent, ok := cfg.Agents["my_openai"]
	if !ok {
		t.Fatal("expected my_openai agent")
	}
	if agent.Type != "codex" {
		t.Fatalf("expected explicit type %q, got %q", "codex", agent.Type)
	}
}

func TestParseHCLConfig_UnclosedBlock(t *testing.T) {
	hcl := `
codex {
  token = "abc"
`
	if _, err := parseHCLConfig([]byte(hcl)); err == nil {
		t.Fatal("expected error for unclosed block")
	}
}

func TestParseHCLConfig_InvalidLine(t *testing.T) {
	hcl := `
codex {
  not_a_kv_pair
}
`
	if _, err := parseHCLConfig([]byte(hcl)); err == nil {
		t.Fatal("expected error for invalid key-value line")
	}
}

func TestParseHCLConfig_EmptyFile(t *testing.T) {
	cfg, err := parseHCLConfig([]byte(""))
	if err != nil {
		t.Fatalf("unexpected error for empty file: %v", err)
	}
	if len(cfg.Agents) != 0 {
		t.Fatalf("expected 0 agents, got %d", len(cfg.Agents))
	}
}

func TestLoadConfigYAMLRootTaskLimit(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.yaml")
	content := `agents:
  claude:
    type: claude

defaults:
  agent: claude
  timeout: 10
  max_concurrent_root_tasks: 3
`
	if err := os.WriteFile(configPath, []byte(content), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}
	cfg, err := LoadConfigForServer(configPath)
	if err != nil {
		t.Fatalf("LoadConfigForServer: %v", err)
	}
	if cfg.Defaults.MaxConcurrentRootTasks != 3 {
		t.Fatalf("max_concurrent_root_tasks=%d, want 3", cfg.Defaults.MaxConcurrentRootTasks)
	}
}

func TestValidateConfigRejectsNegativeRootTaskLimit(t *testing.T) {
	cfg := &Config{
		Agents: map[string]AgentConfig{
			"claude": {Type: "claude"},
		},
		Defaults: DefaultConfig{
			Timeout:                1,
			MaxConcurrentRootTasks: -1,
		},
	}
	if err := ValidateConfig(cfg); err == nil {
		t.Fatalf("expected validation error for negative max_concurrent_root_tasks")
	}
}

func TestLoadConfigYAMLFormat(t *testing.T) {
	dir := t.TempDir()

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

	yamlCfg, err := LoadConfig(yamlPath)
	if err != nil {
		t.Fatalf("LoadConfig YAML: %v", err)
	}
	if yamlCfg.Agents["codex"].Token != "yaml-token" {
		t.Fatalf("expected yaml-token, got %q", yamlCfg.Agents["codex"].Token)
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

func TestFindDefaultConfig(t *testing.T) {
	// FindDefaultConfig is a wrapper around os.Getwd + FindDefaultConfigIn.
	// It should not error in a normal environment.
	_, err := FindDefaultConfig()
	if err != nil {
		t.Fatalf("FindDefaultConfig: %v", err)
	}
}

func TestValidateWebhookConfigValid(t *testing.T) {
	wh := &WebhookConfig{
		URL:     "https://example.com/webhook",
		Timeout: "10s",
	}
	if err := validateWebhookConfig(wh); err != nil {
		t.Fatalf("validateWebhookConfig: %v", err)
	}
}

func TestValidateWebhookConfigInvalidURL(t *testing.T) {
	wh := &WebhookConfig{
		URL: "not-a-valid-url",
	}
	if err := validateWebhookConfig(wh); err == nil {
		t.Fatalf("expected error for invalid URL")
	}
}

func TestValidateWebhookConfigInvalidTimeout(t *testing.T) {
	wh := &WebhookConfig{
		URL:     "https://example.com/hook",
		Timeout: "not-a-duration",
	}
	if err := validateWebhookConfig(wh); err == nil {
		t.Fatalf("expected error for invalid timeout")
	}
}

func TestValidateWebhookConfigEmptyURL(t *testing.T) {
	// Empty URL is allowed — webhook config with no URL is valid.
	wh := &WebhookConfig{
		Timeout: "5s",
	}
	if err := validateWebhookConfig(wh); err != nil {
		t.Fatalf("validateWebhookConfig with empty URL: %v", err)
	}
}

func TestValidateConfigWithWebhook(t *testing.T) {
	cfg := &Config{
		Agents: map[string]AgentConfig{
			"claude": {Type: "claude"},
		},
		Defaults: DefaultConfig{Timeout: 10},
		Webhook:  &WebhookConfig{URL: "https://example.com/hook", Timeout: "5s"},
	}
	if err := ValidateConfig(cfg); err != nil {
		t.Fatalf("ValidateConfig with valid webhook: %v", err)
	}
}

func TestValidateConfigWithInvalidWebhook(t *testing.T) {
	cfg := &Config{
		Agents: map[string]AgentConfig{
			"claude": {Type: "claude"},
		},
		Defaults: DefaultConfig{Timeout: 10},
		Webhook:  &WebhookConfig{URL: "bad-url"},
	}
	if err := ValidateConfig(cfg); err == nil {
		t.Fatalf("expected error for invalid webhook URL")
	}
}

func TestResolveStoragePathsWithExtraRoots(t *testing.T) {
	base := t.TempDir()
	cfg := &Config{
		Storage: StorageConfig{
			RunsDir:    "runs",
			ExtraRoots: []string{"extra1", "extra2"},
		},
	}
	if err := resolveStoragePaths(cfg, base); err != nil {
		t.Fatalf("resolveStoragePaths: %v", err)
	}
	if !filepath.IsAbs(cfg.Storage.RunsDir) {
		t.Fatalf("runs_dir not absolute: %q", cfg.Storage.RunsDir)
	}
	for i, root := range cfg.Storage.ExtraRoots {
		if !filepath.IsAbs(root) {
			t.Fatalf("extra_root[%d] not absolute: %q", i, root)
		}
	}
}

func TestResolveStoragePathsNilConfig(t *testing.T) {
	if err := resolveStoragePaths(nil, "/tmp"); err != nil {
		t.Fatalf("resolveStoragePaths nil: %v", err)
	}
}

// --- DiversificationConfig validation ---

func makeMinimalConfig(agentNames ...string) *Config {
	agents := make(map[string]AgentConfig, len(agentNames))
	for _, name := range agentNames {
		agents[name] = AgentConfig{Type: name}
	}
	return &Config{
		Agents: agents,
		Defaults: DefaultConfig{
			Timeout: 60,
		},
	}
}

func TestValidateDiversificationConfig_Valid(t *testing.T) {
	cfg := makeMinimalConfig("claude", "gemini")
	d := &DiversificationConfig{
		Enabled:  true,
		Strategy: "round-robin",
		Agents:   []string{"claude", "gemini"},
	}
	if err := validateDiversificationConfig(d, cfg); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateDiversificationConfig_ValidWeighted(t *testing.T) {
	cfg := makeMinimalConfig("claude", "gemini")
	d := &DiversificationConfig{
		Enabled:  true,
		Strategy: "weighted",
		Agents:   []string{"claude", "gemini"},
		Weights:  []int{3, 1},
	}
	if err := validateDiversificationConfig(d, cfg); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestValidateDiversificationConfig_InvalidStrategy(t *testing.T) {
	cfg := makeMinimalConfig("claude")
	d := &DiversificationConfig{
		Enabled:  true,
		Strategy: "banana",
	}
	if err := validateDiversificationConfig(d, cfg); err == nil {
		t.Fatal("expected error for invalid strategy")
	}
}

func TestValidateDiversificationConfig_UnknownAgent(t *testing.T) {
	cfg := makeMinimalConfig("claude")
	d := &DiversificationConfig{
		Enabled: true,
		Agents:  []string{"notexist"},
	}
	if err := validateDiversificationConfig(d, cfg); err == nil {
		t.Fatal("expected error for unknown agent name")
	}
}

func TestValidateDiversificationConfig_WeightsLengthMismatch(t *testing.T) {
	cfg := makeMinimalConfig("claude", "gemini")
	d := &DiversificationConfig{
		Enabled: true,
		Agents:  []string{"claude", "gemini"},
		Weights: []int{1}, // wrong length
	}
	if err := validateDiversificationConfig(d, cfg); err == nil {
		t.Fatal("expected error for weights length mismatch")
	}
}

func TestValidateDiversificationConfig_ZeroWeight(t *testing.T) {
	cfg := makeMinimalConfig("claude", "gemini")
	d := &DiversificationConfig{
		Enabled: true,
		Agents:  []string{"claude", "gemini"},
		Weights: []int{1, 0},
	}
	if err := validateDiversificationConfig(d, cfg); err == nil {
		t.Fatal("expected error for zero weight")
	}
}

func TestValidateDiversificationConfig_EmptyAgentName(t *testing.T) {
	cfg := makeMinimalConfig("claude")
	d := &DiversificationConfig{
		Enabled: true,
		Agents:  []string{"claude", ""},
	}
	if err := validateDiversificationConfig(d, cfg); err == nil {
		t.Fatal("expected error for empty agent name in list")
	}
}

func TestValidateDiversificationConfig_NilIsNoOp(t *testing.T) {
	cfg := makeMinimalConfig("claude")
	if err := validateDiversificationConfig(nil, cfg); err != nil {
		t.Fatalf("unexpected error for nil: %v", err)
	}
}
