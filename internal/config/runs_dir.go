package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// ResolveRunsDir returns the runs root directory.
//
// Resolution order:
//  1. If explicit is non-empty it is returned as-is (supports --root flag for tests).
//  2. The default config is loaded; if it specifies storage.runs_dir that is used.
//  3. Falls back to ~/.run-agent/runs.
//
// Returns an error if the config file exists but cannot be parsed, or if the
// user home directory cannot be resolved.
func ResolveRunsDir(explicit string) (string, error) {
	if strings.TrimSpace(explicit) != "" {
		return explicit, nil
	}
	cfgPath, err := FindDefaultConfig()
	if err != nil {
		return "", fmt.Errorf("find config: %w", err)
	}
	if cfgPath != "" {
		cfg, loadErr := LoadConfigForServer(cfgPath)
		if loadErr != nil {
			return "", fmt.Errorf("load config %s: %w", cfgPath, loadErr)
		}
		if strings.TrimSpace(cfg.Storage.RunsDir) != "" {
			return cfg.Storage.RunsDir, nil
		}
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("resolve home dir: %w", err)
	}
	return filepath.Join(home, ".run-agent", "runs"), nil
}
