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
	Agent   string `yaml:"agent"`
	Timeout int    `yaml:"timeout"`
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
//  3. ./config.hcl (returns error — HCL not yet supported)
//  4. $HOME/.config/conductor/config.yaml
//  5. $HOME/.config/conductor/config.yml
//  6. $HOME/.config/conductor/config.hcl (returns error — HCL not yet supported)
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

	type candidate struct {
		path  string
		isHCL bool
	}

	candidates := []candidate{
		{filepath.Join(baseDir, "config.yaml"), false},
		{filepath.Join(baseDir, "config.yml"), false},
		{filepath.Join(baseDir, "config.hcl"), true},
	}
	if home != "" {
		candidates = append(candidates,
			candidate{filepath.Join(home, ".config", "conductor", "config.yaml"), false},
			candidate{filepath.Join(home, ".config", "conductor", "config.yml"), false},
			candidate{filepath.Join(home, ".config", "conductor", "config.hcl"), true},
		)
	}

	for _, c := range candidates {
		if _, err := os.Stat(c.path); err != nil {
			continue
		}
		if c.isHCL {
			return "", fmt.Errorf("HCL config format not yet supported: %s", c.path)
		}
		return c.path, nil
	}

	return "", nil
}

// LoadConfig loads and validates configuration from a YAML file.
func LoadConfig(path string) (*Config, error) {
	if path == "" {
		return nil, fmt.Errorf("config path is empty")
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}

	if cfg.Agents == nil {
		cfg.Agents = make(map[string]AgentConfig)
	}

	applyAgentDefaults(&cfg)
	applyAPIDefaults(&cfg)
	applyTokenEnvOverrides(&cfg)

	baseDir := filepath.Dir(path)
	if err := resolveTokenFilePaths(&cfg, baseDir); err != nil {
		return nil, err
	}
	if err := resolveStoragePaths(&cfg, baseDir); err != nil {
		return nil, err
	}

	if err := ValidateConfig(&cfg); err != nil {
		return nil, err
	}

	if err := resolveTokens(&cfg); err != nil {
		return nil, err
	}

	return &cfg, nil
}

// LoadConfigForServer loads configuration without validating agent tokens.
// This is intended for API server startup where agent execution may be disabled.
func LoadConfigForServer(path string) (*Config, error) {
	if path == "" {
		return nil, fmt.Errorf("config path is empty")
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}

	if cfg.Agents == nil {
		cfg.Agents = make(map[string]AgentConfig)
	}

	applyAgentDefaults(&cfg)
	applyAPIDefaults(&cfg)
	applyTokenEnvOverrides(&cfg)

	baseDir := filepath.Dir(path)
	if err := resolveTokenFilePaths(&cfg, baseDir); err != nil {
		return nil, err
	}
	if err := resolveStoragePaths(&cfg, baseDir); err != nil {
		return nil, err
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
