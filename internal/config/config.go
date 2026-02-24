// Package config provides YAML configuration loading for conductor loop.
package config

import (
	"fmt"
	"os"
	"path/filepath"

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
	Agent                  string               `yaml:"agent"`
	Timeout                int                  `yaml:"timeout"`
	MaxConcurrentRuns      int                  `yaml:"max_concurrent_runs"`
	MaxConcurrentRootTasks int                  `yaml:"max_concurrent_root_tasks"`
	Diversification        *DiversificationConfig `yaml:"diversification,omitempty"`
}

// DiversificationConfig controls how agent selection distributes work across
// multiple configured agents instead of always using a single one.
type DiversificationConfig struct {
	// Enabled activates the diversification policy. When false the default
	// agent selection logic is used unchanged.
	Enabled bool `yaml:"enabled"`

	// Strategy determines how the next agent is chosen.
	// "round-robin" (default): cycle through Agents in order.
	// "weighted":              select proportionally by weight (requires Weights).
	Strategy string `yaml:"strategy,omitempty"`

	// Agents is an ordered list of named agents (keys from Config.Agents) to
	// distribute work across. If empty all configured agents are used.
	Agents []string `yaml:"agents,omitempty"`

	// Weights assigns a relative weight to each agent in Agents when strategy
	// is "weighted". Must have the same length as Agents when provided.
	Weights []int `yaml:"weights,omitempty"`

	// FallbackOnFailure retries the job with the next agent in the list when
	// the selected agent fails.
	FallbackOnFailure bool `yaml:"fallback_on_failure,omitempty"`
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
//  3. $HOME/.config/conductor/config.yaml
//  4. $HOME/.config/conductor/config.yml
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
	}
	if home != "" {
		candidates = append(candidates,
			filepath.Join(home, ".config", "conductor", "config.yaml"),
			filepath.Join(home, ".config", "conductor", "config.yml"),
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

// parseConfigFile reads and parses a YAML config file.
func parseConfigFile(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config: %w", err)
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

func applyAgentDefaults(cfg *Config) {
	for name, agent := range cfg.Agents {
		if agent.Type == "" {
			agent.Type = name
		}
		agent.Type = normalizeAgentType(agent.Type)
		cfg.Agents[name] = agent
	}
}
