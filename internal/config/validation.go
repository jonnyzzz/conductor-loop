package config

import (
	"fmt"
	"os"
)

var validAgentTypes = map[string]struct{}{
	"claude":     {},
	"codex":      {},
	"gemini":     {},
	"perplexity": {},
	"xai":        {},
}

// ValidateConfig validates the configuration for required fields and constraints.
func ValidateConfig(cfg *Config) error {
	if cfg == nil {
		return fmt.Errorf("config is nil")
	}

	if len(cfg.Agents) == 0 {
		return fmt.Errorf("no agents configured")
	}

	if cfg.Defaults.Timeout <= 0 {
		return fmt.Errorf("defaults.timeout must be positive")
	}

	for name, agent := range cfg.Agents {
		if agent.Type == "" {
			return fmt.Errorf("agent %q has empty type", name)
		}

		if _, ok := validAgentTypes[agent.Type]; !ok {
			return fmt.Errorf("agent %q has invalid type %q", name, agent.Type)
		}

		if agent.Token == "" && agent.TokenFile == "" {
			return fmt.Errorf("agent %q must set token or token_file", name)
		}

		if agent.Token != "" && agent.TokenFile != "" && !agent.tokenFromFile {
			return fmt.Errorf("agent %q cannot set both token and token_file", name)
		}

		if agent.TokenFile != "" {
			if err := validateTokenFile(agent.TokenFile); err != nil {
				return fmt.Errorf("agent %q token_file %q: %w", name, agent.TokenFile, err)
			}
		}
	}

	return nil
}

func validateTokenFile(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return err
	}

	if !info.Mode().IsRegular() {
		return fmt.Errorf("not a regular file")
	}

	file, err := os.Open(path)
	if err != nil {
		return err
	}
	defer file.Close()

	return nil
}
