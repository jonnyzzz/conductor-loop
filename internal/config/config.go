// Package config provides YAML and HCL configuration loading for conductor loop.
package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/hashicorp/hcl"
	"gopkg.in/yaml.v3"
)

// Config defines the YAML configuration structure.
type Config struct {
	Agents   map[string]AgentConfig `yaml:"agents"`
	Defaults DefaultConfig          `yaml:"defaults"`
	API      APIConfig              `yaml:"api"`
	Storage  StorageConfig          `yaml:"storage"`
	Webhook  *WebhookConfig         `yaml:"webhook,omitempty"`
}

// WebhookConfig holds configuration for run completion webhook notifications.
type WebhookConfig struct {
	URL     string   `yaml:"url"`
	Events  []string `yaml:"events,omitempty"`  // if empty, send all events
	Secret  string   `yaml:"secret,omitempty"`  // HMAC-SHA256 signing secret (optional)
	Timeout string   `yaml:"timeout,omitempty"` // HTTP timeout, e.g. "10s" (default: "10s")
}

// AgentConfig describes a single agent backend configuration.
type AgentConfig struct {
	Type      string `yaml:"type"` // claude, codex, gemini, perplexity, xai
	Token     string `yaml:"token,omitempty"`
	TokenFile string `yaml:"token_file,omitempty"`
	BaseURL   string `yaml:"base_url,omitempty"`
	Model     string `yaml:"model,omitempty"`

	tokenFromFile bool `yaml:"-"`
}

// DefaultConfig defines defaults used by the runner.
type DefaultConfig struct {
	Agent                  string `yaml:"agent"`
	Timeout                int    `yaml:"timeout"`
	MaxConcurrentRuns      int    `yaml:"max_concurrent_runs"`
	MaxConcurrentRootTasks int    `yaml:"max_concurrent_root_tasks"`
}

// StorageConfig defines storage-related settings.
type StorageConfig struct {
	RunsDir    string   `yaml:"runs_dir"`
	ExtraRoots []string `yaml:"extra_roots,omitempty"`
}

// FindDefaultConfig searches for a config file in default locations.
// Returns the path if found, empty string if not found.
// Search order:
//  1. ./config.yaml
//  2. ./config.yml
//  3. ./config.hcl
//  4. $HOME/.config/conductor/config.yaml
//  5. $HOME/.config/conductor/config.yml
//  6. $HOME/.config/conductor/config.hcl
func FindDefaultConfig() (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("get working directory: %w", err)
	}
	return FindDefaultConfigIn(cwd)
}

// FindDefaultConfigIn searches for a config file starting from baseDir.
// This variant is provided for testability without os.Chdir.
func FindDefaultConfigIn(baseDir string) (string, error) {
	home, _ := os.UserHomeDir()

	candidates := []string{
		filepath.Join(baseDir, "config.yaml"),
		filepath.Join(baseDir, "config.yml"),
		filepath.Join(baseDir, "config.hcl"),
	}
	if home != "" {
		candidates = append(candidates,
			filepath.Join(home, ".config", "conductor", "config.yaml"),
			filepath.Join(home, ".config", "conductor", "config.yml"),
			filepath.Join(home, ".config", "conductor", "config.hcl"),
		)
	}

	for _, path := range candidates {
		if _, err := os.Stat(path); err != nil {
			continue
		}
		return path, nil
	}

	return "", nil
}

// LoadConfig loads and validates configuration from a YAML or HCL file.
func LoadConfig(path string) (*Config, error) {
	if path == "" {
		return nil, fmt.Errorf("config path is empty")
	}

	cfg, err := parseConfigFile(path)
	if err != nil {
		return nil, err
	}

	applyAgentDefaults(cfg)
	applyAPIDefaults(cfg)
	applyTokenEnvOverrides(cfg)
	applyAPIEnvOverrides(cfg)

	baseDir := filepath.Dir(path)
	if err := resolveTokenFilePaths(cfg, baseDir); err != nil {
		return nil, err
	}
	if err := resolveStoragePaths(cfg, baseDir); err != nil {
		return nil, err
	}

	if err := ValidateConfig(cfg); err != nil {
		return nil, err
	}

	if err := resolveTokens(cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}

// LoadConfigForServer loads configuration without validating agent tokens.
// This is intended for API server startup where agent execution may be disabled.
func LoadConfigForServer(path string) (*Config, error) {
	if path == "" {
		return nil, fmt.Errorf("config path is empty")
	}

	cfg, err := parseConfigFile(path)
	if err != nil {
		return nil, err
	}

	applyAgentDefaults(cfg)
	applyAPIDefaults(cfg)
	applyTokenEnvOverrides(cfg)
	applyAPIEnvOverrides(cfg)

	baseDir := filepath.Dir(path)
	if err := resolveTokenFilePaths(cfg, baseDir); err != nil {
		return nil, err
	}
	if err := resolveStoragePaths(cfg, baseDir); err != nil {
		return nil, err
	}

	return cfg, nil
}

// parseConfigFile reads and parses a config file, auto-detecting format by extension.
func parseConfigFile(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}

	ext := strings.ToLower(filepath.Ext(path))
	if ext == ".hcl" {
		return parseHCLConfig(path, data)
	}
	return parseYAMLConfig(data)
}

// parseYAMLConfig decodes YAML bytes into a Config.
func parseYAMLConfig(data []byte) (*Config, error) {
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}
	if cfg.Agents == nil {
		cfg.Agents = make(map[string]AgentConfig)
	}
	return &cfg, nil
}

// parseHCLConfig decodes HCL bytes into a Config.
func parseHCLConfig(path string, data []byte) (*Config, error) {
	var raw map[string]interface{}
	if err := hcl.Decode(&raw, string(data)); err != nil {
		return nil, fmt.Errorf("parse HCL config %s: %w", path, err)
	}

	cfg := &Config{
		Agents: make(map[string]AgentConfig),
	}

	// agents block
	if v, ok := raw["agents"]; ok {
		for name, agentRaw := range hclFirstBlock(v) {
			agentBlocks := hclFirstBlock(agentRaw)
			if agentBlocks == nil {
				continue
			}
			agent := AgentConfig{}
			if s, ok := agentBlocks["type"].(string); ok {
				agent.Type = s
			}
			if s, ok := agentBlocks["token"].(string); ok {
				agent.Token = s
			}
			if s, ok := agentBlocks["token_file"].(string); ok {
				agent.TokenFile = s
			}
			if s, ok := agentBlocks["base_url"].(string); ok {
				agent.BaseURL = s
			}
			if s, ok := agentBlocks["model"].(string); ok {
				agent.Model = s
			}
			cfg.Agents[name] = agent
		}
	}

	// defaults block
	if m := hclFirstBlock(raw["defaults"]); m != nil {
		if s, ok := m["agent"].(string); ok {
			cfg.Defaults.Agent = s
		}
		if n, ok := m["timeout"].(int); ok {
			cfg.Defaults.Timeout = n
		}
		if n, ok := m["max_concurrent_runs"].(int); ok {
			cfg.Defaults.MaxConcurrentRuns = n
		}
		if n, ok := m["max_concurrent_root_tasks"].(int); ok {
			cfg.Defaults.MaxConcurrentRootTasks = n
		}
	}

	// api block
	if m := hclFirstBlock(raw["api"]); m != nil {
		if s, ok := m["host"].(string); ok {
			cfg.API.Host = s
		}
		if n, ok := m["port"].(int); ok {
			cfg.API.Port = n
		}
		if b, ok := m["auth_enabled"].(bool); ok {
			cfg.API.AuthEnabled = b
		}
		if s, ok := m["api_key"].(string); ok {
			cfg.API.APIKey = s
		}
		if list, ok := m["cors_origins"].([]interface{}); ok {
			for _, item := range list {
				if s, ok := item.(string); ok {
					cfg.API.CORSOrigins = append(cfg.API.CORSOrigins, s)
				}
			}
		}
		if sm := hclFirstBlock(m["sse"]); sm != nil {
			if n, ok := sm["poll_interval_ms"].(int); ok {
				cfg.API.SSE.PollIntervalMs = n
			}
			if n, ok := sm["discovery_interval_ms"].(int); ok {
				cfg.API.SSE.DiscoveryIntervalMs = n
			}
			if n, ok := sm["heartbeat_interval_s"].(int); ok {
				cfg.API.SSE.HeartbeatIntervalS = n
			}
			if n, ok := sm["max_clients_per_run"].(int); ok {
				cfg.API.SSE.MaxClientsPerRun = n
			}
		}
	}

	// storage block
	if m := hclFirstBlock(raw["storage"]); m != nil {
		if s, ok := m["runs_dir"].(string); ok {
			cfg.Storage.RunsDir = s
		}
		if list, ok := m["extra_roots"].([]interface{}); ok {
			for _, item := range list {
				if s, ok := item.(string); ok {
					cfg.Storage.ExtraRoots = append(cfg.Storage.ExtraRoots, s)
				}
			}
		}
	}

	return cfg, nil
}

// hclFirstBlock returns the first element of an HCL v1 block value ([]map[string]interface{}).
// Returns nil if the value is not a block or is empty.
func hclFirstBlock(v interface{}) map[string]interface{} {
	if v == nil {
		return nil
	}
	blocks, ok := v.([]map[string]interface{})
	if !ok || len(blocks) == 0 {
		return nil
	}
	return blocks[0]
}

func applyAgentDefaults(cfg *Config) {
	for name, agent := range cfg.Agents {
		if agent.Type == "" {
			agent.Type = name
		}
		agent.Type = normalizeAgentType(agent.Type)
		cfg.Agents[name] = agent
	}
}
