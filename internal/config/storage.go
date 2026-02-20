package config

import "fmt"

func resolveStoragePaths(cfg *Config, baseDir string) error {
	if cfg == nil {
		return nil
	}
	if cfg.Storage.RunsDir != "" {
		resolved, err := resolvePath(baseDir, cfg.Storage.RunsDir)
		if err != nil {
			return fmt.Errorf("resolve storage.runs_dir: %w", err)
		}
		cfg.Storage.RunsDir = resolved
	}
	for i, extra := range cfg.Storage.ExtraRoots {
		resolved, err := resolvePath(baseDir, extra)
		if err != nil {
			return fmt.Errorf("resolve storage.extra_roots[%d]: %w", i, err)
		}
		cfg.Storage.ExtraRoots[i] = resolved
	}
	return nil
}
