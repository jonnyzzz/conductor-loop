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
	if cfg.API.SSE.PollIntervalMs < 0 {
		return fmt.Errorf("api.sse.poll_interval_ms must be non-negative")
	}
	if cfg.API.SSE.DiscoveryIntervalMs < 0 {
		return fmt.Errorf("api.sse.discovery_interval_ms must be non-negative")
	}
	if cfg.API.SSE.HeartbeatIntervalS < 0 {
		return fmt.Errorf("api.sse.heartbeat_interval_s must be non-negative")
	}
	if cfg.API.SSE.MaxClientsPerRun < 0 {
		return fmt.Errorf("api.sse.max_clients_per_run must be non-negative")
	}

	for name, agent := range cfg.Agents {
		if agent.Type == "" {
			return fmt.Errorf("agent %q has empty type", name)
		}

		if _, ok := validAgentTypes[agent.Type]; !ok {
			return fmt.Errorf("agent %q has invalid type %q", name, agent.Type)
		}

		// token/token_file are optional — CLI agents (claude, codex, gemini)
		// can authenticate via their own mechanisms.

		if agent.Token != "" && agent.TokenFile != "" && !agent.tokenFromFile {
			return fmt.Errorf("agent %q cannot set both token and token_file", name)
		}

		if agent.TokenFile != "" {
			if err := validateTokenFile(agent.TokenFile); err != nil {
				return fmt.Errorf("agent %q token_file %q: %w", name, agent.TokenFile, err)
			}
		}
	}

	if cfg.API.Port < 0 || cfg.API.Port > 65535 {
		return fmt.Errorf("api.port must be between 0 and 65535")
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

	if _, err := os.ReadFile(path); err != nil {
		return err
	}

	return nil
}
