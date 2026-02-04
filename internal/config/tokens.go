package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func applyTokenEnvOverrides(cfg *Config) {
	for name, agent := range cfg.Agents {
		envName := tokenEnvVarName(name)
		if value, ok := os.LookupEnv(envName); ok {
			value = strings.TrimSpace(value)
			if value != "" {
				agent.Token = value
				agent.TokenFile = ""
				agent.tokenFromFile = false
				cfg.Agents[name] = agent
			}
		}
	}
}

func tokenEnvVarName(name string) string {
	upper := strings.ToUpper(name)
	var b strings.Builder
	b.Grow(len(upper))
	for _, r := range upper {
		if (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') {
			b.WriteRune(r)
			continue
		}
		b.WriteByte('_')
	}

	return "CONDUCTOR_AGENT_" + b.String() + "_TOKEN"
}

func resolveTokenFilePaths(cfg *Config, baseDir string) error {
	for name, agent := range cfg.Agents {
		if agent.TokenFile == "" {
			continue
		}

		resolved, err := resolvePath(baseDir, agent.TokenFile)
		if err != nil {
			return fmt.Errorf("resolve token_file for agent %q: %w", name, err)
		}

		agent.TokenFile = resolved
		cfg.Agents[name] = agent
	}

	return nil
}

func resolvePath(baseDir, target string) (string, error) {
	trimmed := strings.TrimSpace(target)
	if trimmed == "" {
		return "", nil
	}

	if strings.HasPrefix(trimmed, "~") {
		resolved, err := expandHome(trimmed)
		if err != nil {
			return "", err
		}
		trimmed = resolved
	}

	if !filepath.IsAbs(trimmed) {
		trimmed = filepath.Join(baseDir, trimmed)
	}

	abs, err := filepath.Abs(trimmed)
	if err != nil {
		return "", fmt.Errorf("resolve path %q: %w", trimmed, err)
	}

	return abs, nil
}

func expandHome(path string) (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve home dir: %w", err)
	}

	if path == "~" {
		return home, nil
	}

	if strings.HasPrefix(path, "~/") {
		return filepath.Join(home, path[2:]), nil
	}

	return "", fmt.Errorf("unsupported home expansion: %q", path)
}

func resolveTokens(cfg *Config) error {
	for name, agent := range cfg.Agents {
		if agent.TokenFile == "" {
			continue
		}

		token, err := readTokenFile(agent.TokenFile)
		if err != nil {
			return fmt.Errorf("read token_file for agent %q: %w", name, err)
		}

		agent.Token = token
		agent.tokenFromFile = true
		cfg.Agents[name] = agent
	}

	return nil
}

func readTokenFile(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}

	trimmed := strings.TrimSpace(string(data))
	if trimmed == "" {
		return "", fmt.Errorf("token file is empty: %s", path)
	}

	return trimmed, nil
}

func normalizeAgentType(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}
