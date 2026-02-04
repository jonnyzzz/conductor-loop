package unit_test

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/jonnyzzz/conductor-loop/internal/config"
)

func TestLoadConfig(t *testing.T) {
	dir := t.TempDir()
	tokenPath := filepath.Join(dir, "claude.token")
	if err := os.WriteFile(tokenPath, []byte("secret-token\n"), 0o600); err != nil {
		t.Fatalf("write token file: %v", err)
	}

	configPath := filepath.Join(dir, "config.yaml")
	content := fmt.Sprintf(`agents:
  claude:
    type: claude
    token_file: %s
  codex:
    type: codex
    token: inline-token

defaults:
  agent: claude
  timeout: 120
`, tokenPath)
	if err := os.WriteFile(configPath, []byte(content), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}

	claude := cfg.Agents["claude"]
	if claude.Token != "secret-token" {
		t.Fatalf("claude token mismatch: %q", claude.Token)
	}
	if claude.TokenFile != tokenPath {
		t.Fatalf("claude token_file mismatch: %q", claude.TokenFile)
	}

	codex := cfg.Agents["codex"]
	if codex.Token != "inline-token" {
		t.Fatalf("codex token mismatch: %q", codex.Token)
	}

	if cfg.Defaults.Agent != "claude" {
		t.Fatalf("defaults agent mismatch: %q", cfg.Defaults.Agent)
	}
	if cfg.Defaults.Timeout != 120 {
		t.Fatalf("defaults timeout mismatch: %d", cfg.Defaults.Timeout)
	}
}

func TestTokenFileResolution(t *testing.T) {
	dir := t.TempDir()
	tokenDir := filepath.Join(dir, "secrets")
	if err := os.MkdirAll(tokenDir, 0o700); err != nil {
		t.Fatalf("mkdir secrets: %v", err)
	}

	tokenPath := filepath.Join(tokenDir, "token.txt")
	if err := os.WriteFile(tokenPath, []byte("file-token"), 0o600); err != nil {
		t.Fatalf("write token file: %v", err)
	}

	configPath := filepath.Join(dir, "config.yaml")
	content := `agents:
  claude:
    type: claude
    token_file: secrets/token.txt

defaults:
  agent: claude
  timeout: 10
`
	if err := os.WriteFile(configPath, []byte(content), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}

	claude := cfg.Agents["claude"]
	if claude.Token != "file-token" {
		t.Fatalf("token mismatch: %q", claude.Token)
	}
	if claude.TokenFile != tokenPath {
		t.Fatalf("token_file mismatch: %q", claude.TokenFile)
	}
	if !filepath.IsAbs(claude.TokenFile) {
		t.Fatalf("token_file should be absolute: %q", claude.TokenFile)
	}
}

func TestTokenFromEnv(t *testing.T) {
	dir := t.TempDir()
	tokenPath := filepath.Join(dir, "token.txt")
	if err := os.WriteFile(tokenPath, []byte("file-token"), 0o600); err != nil {
		t.Fatalf("write token file: %v", err)
	}

	configPath := filepath.Join(dir, "config.yaml")
	content := fmt.Sprintf(`agents:
  claude:
    type: claude
    token_file: %s

defaults:
  agent: claude
  timeout: 10
`, tokenPath)
	if err := os.WriteFile(configPath, []byte(content), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	t.Setenv("CONDUCTOR_AGENT_CLAUDE_TOKEN", " env-token ")

	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}

	claude := cfg.Agents["claude"]
	if claude.Token != "env-token" {
		t.Fatalf("env token mismatch: %q", claude.Token)
	}
	if claude.TokenFile != "" {
		t.Fatalf("expected token_file to be empty, got %q", claude.TokenFile)
	}
}

func TestConfigValidation(t *testing.T) {
	dir := t.TempDir()
	goodTokenPath := filepath.Join(dir, "token.txt")
	if err := os.WriteFile(goodTokenPath, []byte("token"), 0o600); err != nil {
		t.Fatalf("write token file: %v", err)
	}

	cases := []struct {
		name string
		cfg  config.Config
	}{
		{
			name: "no agents",
			cfg: config.Config{
				Agents:   map[string]config.AgentConfig{},
				Defaults: config.DefaultConfig{Timeout: 1},
			},
		},
		{
			name: "invalid type",
			cfg: config.Config{
				Agents: map[string]config.AgentConfig{
					"bad": {Type: "invalid", Token: "x"},
				},
				Defaults: config.DefaultConfig{Timeout: 1},
			},
		},
		{
			name: "missing token",
			cfg: config.Config{
				Agents: map[string]config.AgentConfig{
					"claude": {Type: "claude"},
				},
				Defaults: config.DefaultConfig{Timeout: 1},
			},
		},
		{
			name: "both token and token_file",
			cfg: config.Config{
				Agents: map[string]config.AgentConfig{
					"claude": {Type: "claude", Token: "x", TokenFile: goodTokenPath},
				},
				Defaults: config.DefaultConfig{Timeout: 1},
			},
		},
		{
			name: "token_file missing",
			cfg: config.Config{
				Agents: map[string]config.AgentConfig{
					"claude": {Type: "claude", TokenFile: filepath.Join(dir, "missing.txt")},
				},
				Defaults: config.DefaultConfig{Timeout: 1},
			},
		},
		{
			name: "timeout not positive",
			cfg: config.Config{
				Agents: map[string]config.AgentConfig{
					"claude": {Type: "claude", Token: "x"},
				},
				Defaults: config.DefaultConfig{Timeout: 0},
			},
		},
	}

	for _, tc := range cases {
		tc := tc
		t.Run(tc.name, func(t *testing.T) {
			if err := config.ValidateConfig(&tc.cfg); err == nil {
				t.Fatalf("expected error")
			}
		})
	}
}

func TestAgentDefaults(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.yaml")
	content := `agents:
  claude:
    token: inline-token

defaults:
  agent: claude
  timeout: 10
`
	if err := os.WriteFile(configPath, []byte(content), 0o600); err != nil {
		t.Fatalf("write config: %v", err)
	}

	cfg, err := config.LoadConfig(configPath)
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}

	claude := cfg.Agents["claude"]
	if claude.Type != "claude" {
		t.Fatalf("expected type default to claude, got %q", claude.Type)
	}
}
