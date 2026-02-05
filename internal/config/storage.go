package config

import "fmt"

func resolveStoragePaths(cfg *Config, baseDir string) error {
	if cfg == nil {
		return nil
	}
	if cfg.Storage.RunsDir == "" {
		return nil
	}
	resolved, err := resolvePath(baseDir, cfg.Storage.RunsDir)
	if err != nil {
		return fmt.Errorf("resolve storage.runs_dir: %w", err)
	}
	cfg.Storage.RunsDir = resolved
	return nil
}
