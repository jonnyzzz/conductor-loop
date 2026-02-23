package config

import (
	"fmt"
	"net/url"
	"os"
	"strings"
	"time"
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
	if cfg.Defaults.MaxConcurrentRootTasks < 0 {
		return fmt.Errorf("defaults.max_concurrent_root_tasks must be non-negative")
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

	if cfg.Webhook != nil {
		if err := validateWebhookConfig(cfg.Webhook); err != nil {
			return err
		}
	}

	if cfg.Defaults.Diversification != nil {
		if err := validateDiversificationConfig(cfg.Defaults.Diversification, cfg); err != nil {
			return err
		}
	}

	return nil
}

var validDiversificationStrategies = map[string]struct{}{
	"round-robin": {},
	"weighted":    {},
}

func validateDiversificationConfig(d *DiversificationConfig, cfg *Config) error {
	if d == nil {
		return nil
	}
	if strategy := strings.TrimSpace(d.Strategy); strategy != "" {
		if _, ok := validDiversificationStrategies[strategy]; !ok {
			return fmt.Errorf("defaults.diversification.strategy %q is invalid; valid values: round-robin, weighted", strategy)
		}
	}
	for i, name := range d.Agents {
		if strings.TrimSpace(name) == "" {
			return fmt.Errorf("defaults.diversification.agents[%d] is empty", i)
		}
		if _, ok := cfg.Agents[name]; !ok {
			return fmt.Errorf("defaults.diversification.agents[%d] %q not found in agents map", i, name)
		}
	}
	if len(d.Weights) > 0 {
		if len(d.Weights) != len(d.Agents) {
			return fmt.Errorf("defaults.diversification.weights length (%d) must match agents length (%d)",
				len(d.Weights), len(d.Agents))
		}
		for i, w := range d.Weights {
			if w <= 0 {
				return fmt.Errorf("defaults.diversification.weights[%d] must be positive, got %d", i, w)
			}
		}
	}
	return nil
}

func validateWebhookConfig(wh *WebhookConfig) error {
	if wh.URL != "" {
		if _, err := url.ParseRequestURI(wh.URL); err != nil {
			return fmt.Errorf("webhook.url is invalid: %w", err)
		}
	}
	if wh.Timeout != "" {
		if _, err := time.ParseDuration(wh.Timeout); err != nil {
			return fmt.Errorf("webhook.timeout is invalid: %w", err)
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

	if _, err := os.ReadFile(path); err != nil {
		return err
	}

	return nil
}
