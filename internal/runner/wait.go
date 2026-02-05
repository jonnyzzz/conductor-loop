package runner

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/jonnyzzz/conductor-loop/internal/storage"
	"github.com/pkg/errors"
)

const (
	defaultChildPollInterval = time.Second
)

// ErrChildWaitTimeout indicates the child wait loop exceeded the timeout.
var ErrChildWaitTimeout = errors.New("child wait timeout")

// ChildProcess captures minimal run metadata for a child process.
type ChildProcess struct {
	RunID       string
	PID         int
	PGID        int
	RunInfoPath string
}

// FindActiveChildren discovers active children for a task run directory.
func FindActiveChildren(runDir string) ([]ChildProcess, error) {
	clean := filepath.Clean(strings.TrimSpace(runDir))
	if clean == "." || clean == "" {
		return nil, errors.New("run directory is empty")
	}
	runsDir := filepath.Join(clean, "runs")
	entries, err := os.ReadDir(runsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []ChildProcess{}, nil
		}
		return nil, errors.Wrap(err, "read runs directory")
	}

	var (
		children []ChildProcess
		lastErr  error
	)
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		path := filepath.Join(runsDir, entry.Name(), "run-info.yaml")
		info, err := storage.ReadRunInfo(path)
		if err != nil {
			if lastErr == nil {
				lastErr = errors.Wrapf(err, "read run-info for run %s", entry.Name())
			}
			continue
		}
		if !info.EndTime.IsZero() {
			continue
		}
		if strings.TrimSpace(info.ParentRunID) == "" {
			continue
		}
		alive, err := isProcessGroupAlive(info.PGID)
		if err != nil {
			if lastErr == nil {
				lastErr = errors.Wrapf(err, "check process group for run %s", info.RunID)
			}
			continue
		}
		if alive {
			children = append(children, ChildProcess{
				RunID:       info.RunID,
				PID:         info.PID,
				PGID:        info.PGID,
				RunInfoPath: path,
			})
			continue
		}
		markRunInfoFailed(path, time.Now)
	}

	return children, lastErr
}

// WaitForChildren waits for child processes to exit or until timeout.
func WaitForChildren(ctx context.Context, children []ChildProcess, timeout, pollInterval time.Duration) ([]ChildProcess, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if timeout <= 0 {
		return children, errors.New("timeout must be positive")
	}
	if pollInterval <= 0 {
		pollInterval = defaultChildPollInterval
	}

	deadline := time.Now().Add(timeout)
	remaining := children
	for {
		remaining = filterAliveChildren(remaining)
		if len(remaining) == 0 {
			return nil, nil
		}
		if time.Now().After(deadline) {
			return remaining, ErrChildWaitTimeout
		}
		if err := sleepWithContext(ctx, pollInterval); err != nil {
			return remaining, err
		}
	}
}

func filterAliveChildren(children []ChildProcess) []ChildProcess {
	if len(children) == 0 {
		return nil
	}
	still := make([]ChildProcess, 0, len(children))
	for _, child := range children {
		alive, err := isProcessGroupAlive(child.PGID)
		if err != nil {
			still = append(still, child)
			continue
		}
		if alive {
			still = append(still, child)
			continue
		}
		markRunInfoFailed(child.RunInfoPath, time.Now)
	}
	return still
}

func markRunInfoFailed(path string, now func() time.Time) {
	if path == "" || now == nil {
		return
	}
	_ = storage.UpdateRunInfo(path, func(info *storage.RunInfo) error {
		if info == nil {
			return nil
		}
		if !info.EndTime.IsZero() {
			return nil
		}
		info.EndTime = now().UTC()
		if info.ExitCode == 0 {
			info.ExitCode = -1
		}
		info.Status = storage.StatusFailed
		return nil
	})
}
