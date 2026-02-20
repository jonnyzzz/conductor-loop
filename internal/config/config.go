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
