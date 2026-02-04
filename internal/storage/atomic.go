package storage

import (
	"os"
	"path/filepath"
	"runtime"

	"github.com/pkg/errors"
	"gopkg.in/yaml.v3"
)

const runInfoFileMode = 0o644

// WriteRunInfo atomically writes run metadata to the specified path.
func WriteRunInfo(path string, info *RunInfo) error {
	if path == "" {
		return errors.New("run-info path is empty")
	}
	if info == nil {
		return errors.New("run-info is nil")
	}
	data, err := yaml.Marshal(info)
	if err != nil {
		return errors.Wrap(err, "marshal run-info")
	}
	if err := writeFileAtomic(path, data); err != nil {
		return errors.Wrap(err, "write run-info")
	}
	return nil
}

// ReadRunInfo reads run metadata from the specified path.
func ReadRunInfo(path string) (*RunInfo, error) {
	if path == "" {
		return nil, errors.New("run-info path is empty")
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, errors.Wrap(err, "read run-info")
	}
	var info RunInfo
	if err := yaml.Unmarshal(data, &info); err != nil {
		return nil, errors.Wrap(err, "unmarshal run-info")
	}
	return &info, nil
}

// UpdateRunInfo applies updates to run-info.yaml and rewrites it atomically.
func UpdateRunInfo(path string, update func(*RunInfo) error) error {
	if update == nil {
		return errors.New("update function is nil")
	}
	info, err := ReadRunInfo(path)
	if err != nil {
		return errors.Wrap(err, "read run-info for update")
	}
	if err := update(info); err != nil {
		return errors.Wrap(err, "apply run-info update")
	}
	if err := WriteRunInfo(path, info); err != nil {
		return errors.Wrap(err, "rewrite run-info")
	}
	return nil
}

func writeFileAtomic(path string, data []byte) error {
	dir := filepath.Dir(path)
	tmpFile, err := os.CreateTemp(dir, "run-info.*.yaml.tmp")
	if err != nil {
		return errors.Wrap(err, "create temp file")
	}
	tmpName := tmpFile.Name()
	success := false
	defer func() {
		if success {
			return
		}
		_ = tmpFile.Close()
		_ = os.Remove(tmpName)
	}()

	if _, err := tmpFile.Write(data); err != nil {
		return errors.Wrap(err, "write temp file")
	}
	if err := tmpFile.Sync(); err != nil {
		return errors.Wrap(err, "fsync temp file")
	}
	if err := tmpFile.Chmod(runInfoFileMode); err != nil {
		return errors.Wrap(err, "chmod temp file")
	}
	if err := tmpFile.Close(); err != nil {
		return errors.Wrap(err, "close temp file")
	}
	if err := os.Rename(tmpName, path); err != nil {
		if runtime.GOOS == "windows" {
			if removeErr := os.Remove(path); removeErr == nil {
				if renameErr := os.Rename(tmpName, path); renameErr == nil {
					success = true
					return nil
				}
			}
		}
		return errors.Wrap(err, "rename temp file")
	}
	success = true
	return nil
}
