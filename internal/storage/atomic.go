package storage

import (
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/jonnyzzz/conductor-loop/internal/messagebus"
	"github.com/jonnyzzz/conductor-loop/internal/obslog"
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
		obslog.Log(log.Default(), "ERROR", "storage", "run_info_marshal_failed",
			obslog.F("run_info_path", path),
			obslog.F("run_id", info.RunID),
			obslog.F("error", err),
		)
		return errors.Wrap(err, "marshal run-info")
	}
	if err := writeFileAtomic(path, data); err != nil {
		obslog.Log(log.Default(), "ERROR", "storage", "run_info_write_failed",
			obslog.F("run_info_path", path),
			obslog.F("run_id", info.RunID),
			obslog.F("error", err),
		)
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
		if os.IsNotExist(err) {
			obslog.Log(log.Default(), "DEBUG", "storage", "run_info_read_missing",
				obslog.F("run_info_path", path),
				obslog.F("run_id", runIDFromRunInfoPath(path)),
			)
			return nil, errors.Wrap(err, "read run-info")
		}
		obslog.Log(log.Default(), "ERROR", "storage", "run_info_read_failed",
			obslog.F("run_info_path", path),
			obslog.F("run_id", runIDFromRunInfoPath(path)),
			obslog.F("error", err),
		)
		return nil, errors.Wrap(err, "read run-info")
	}
	var info RunInfo
	if err := yaml.Unmarshal(data, &info); err != nil {
		obslog.Log(log.Default(), "WARN", "storage", "run_info_unmarshal_failed",
			obslog.F("run_info_path", path),
			obslog.F("run_id", runIDFromRunInfoPath(path)),
			obslog.F("error", err),
		)
		return nil, errors.Wrap(err, "unmarshal run-info")
	}
	hydrateRunInfoDefaults(&info, path)
	return &info, nil
}

const updateRunInfoLockTimeout = 5 * time.Second

// UpdateRunInfo applies updates to run-info.yaml and rewrites it atomically.
// A file lock is held for the duration of the read-modify-write cycle to
// prevent concurrent updates from losing data (ISSUE-019).
func UpdateRunInfo(path string, update func(*RunInfo) error) error {
	if update == nil {
		return errors.New("update function is nil")
	}

	lockPath := path + ".lock"
	lockFile, err := os.OpenFile(lockPath, os.O_CREATE|os.O_RDWR, 0o644)
	if err != nil {
		obslog.Log(log.Default(), "ERROR", "storage", "run_info_lock_open_failed",
			obslog.F("run_info_path", path),
			obslog.F("error", err),
		)
		return errors.Wrap(err, "open run-info lock file")
	}
	defer func() {
		_ = messagebus.Unlock(lockFile)
		_ = lockFile.Close()
		_ = os.Remove(lockPath)
	}()

	lockStart := time.Now()
	if err := messagebus.LockExclusive(lockFile, updateRunInfoLockTimeout); err != nil {
		obslog.Log(log.Default(), "ERROR", "storage", "run_info_lock_acquire_failed",
			obslog.F("run_info_path", path),
			obslog.F("run_id", runIDFromRunInfoPath(path)),
			obslog.F("lock_timeout", updateRunInfoLockTimeout),
			obslog.F("error", err),
		)
		return errors.Wrap(err, "acquire run-info lock")
	}
	lockWait := time.Since(lockStart)
	if lockWait >= 200*time.Millisecond {
		obslog.Log(log.Default(), "WARN", "storage", "run_info_lock_wait_slow",
			obslog.F("run_info_path", path),
			obslog.F("run_id", runIDFromRunInfoPath(path)),
			obslog.F("wait_ms", lockWait.Milliseconds()),
		)
	}

	info, err := ReadRunInfo(path)
	if err != nil {
		obslog.Log(log.Default(), "ERROR", "storage", "run_info_read_for_update_failed",
			obslog.F("run_info_path", path),
			obslog.F("run_id", runIDFromRunInfoPath(path)),
			obslog.F("error", err),
		)
		return errors.Wrap(err, "read run-info for update")
	}
	if err := update(info); err != nil {
		obslog.Log(log.Default(), "ERROR", "storage", "run_info_update_callback_failed",
			obslog.F("run_info_path", path),
			obslog.F("run_id", info.RunID),
			obslog.F("error", err),
		)
		return errors.Wrap(err, "apply run-info update")
	}
	if err := WriteRunInfo(path, info); err != nil {
		obslog.Log(log.Default(), "ERROR", "storage", "run_info_rewrite_failed",
			obslog.F("run_info_path", path),
			obslog.F("run_id", info.RunID),
			obslog.F("error", err),
		)
		return errors.Wrap(err, "rewrite run-info")
	}
	return nil
}

func runIDFromRunInfoPath(path string) string {
	clean := filepath.Clean(strings.TrimSpace(path))
	if clean == "." || clean == "" {
		return ""
	}
	parent := filepath.Base(filepath.Dir(clean))
	if parent == "." || parent == string(filepath.Separator) {
		return ""
	}
	return parent
}

func hydrateRunInfoDefaults(info *RunInfo, path string) {
	if info == nil {
		return
	}
	info.RunID = strings.TrimSpace(info.RunID)
	if info.RunID == "" {
		info.RunID = runIDFromRunInfoPath(path)
	}

	info.ProjectID = strings.TrimSpace(info.ProjectID)
	info.TaskID = strings.TrimSpace(info.TaskID)
	inferredProjectID, inferredTaskID := runScopeFromRunInfoPath(path)
	if info.ProjectID == "" {
		info.ProjectID = inferredProjectID
	}
	if info.TaskID == "" {
		info.TaskID = inferredTaskID
	}

	if strings.TrimSpace(info.Status) == "" {
		info.Status = StatusUnknown
	}
}

// RunScopeFromRunInfoPath extracts projectID and taskID from a canonical
// run-info.yaml path (<root>/<project>/<task>/runs/<runID>/run-info.yaml).
// Returns empty strings if the path does not match the expected layout.
func RunScopeFromRunInfoPath(path string) (projectID, taskID string) {
	return runScopeFromRunInfoPath(path)
}

func runScopeFromRunInfoPath(path string) (projectID, taskID string) {
	clean := filepath.Clean(strings.TrimSpace(path))
	if clean == "." || clean == "" {
		return "", ""
	}
	runDir := filepath.Dir(clean)
	if runDir == "." || runDir == "" {
		return "", ""
	}
	runsDir := filepath.Dir(runDir)
	if filepath.Base(runsDir) != "runs" {
		return "", ""
	}
	taskDir := filepath.Dir(runsDir)
	projectDir := filepath.Dir(taskDir)
	taskID = strings.TrimSpace(filepath.Base(taskDir))
	projectID = strings.TrimSpace(filepath.Base(projectDir))
	if taskID == "." || taskID == string(filepath.Separator) {
		taskID = ""
	}
	if projectID == "." || projectID == string(filepath.Separator) {
		projectID = ""
	}
	return projectID, taskID
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
