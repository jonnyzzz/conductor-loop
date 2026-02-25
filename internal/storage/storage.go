package storage

import (
	stderrors "errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/jonnyzzz/conductor-loop/internal/obslog"
	"github.com/pkg/errors"
)

// Storage defines run metadata storage operations.
type Storage interface {
	CreateRun(projectID, taskID, agentType string) (*RunInfo, error)
	UpdateRunStatus(runID string, status string, exitCode int) error
	GetRunInfo(runID string) (*RunInfo, error)
	ListRuns(projectID, taskID string) ([]*RunInfo, error)
}

// FileStorage stores run metadata on disk under the storage root.
type FileStorage struct {
	root     string
	now      func() time.Time
	pid      func() int
	runIndex map[string]string
	mu       sync.RWMutex
}

// NewStorage creates a FileStorage rooted at the provided directory.
func NewStorage(root string) (*FileStorage, error) {
	cleanRoot := filepath.Clean(strings.TrimSpace(root))
	if cleanRoot == "." || cleanRoot == "" {
		return nil, errors.New("storage root is empty")
	}
	return &FileStorage{
		root:     cleanRoot,
		now:      time.Now,
		pid:      os.Getpid,
		runIndex: make(map[string]string),
	}, nil
}

// CreateRun creates a new run directory and persists run-info.yaml.
func (s *FileStorage) CreateRun(projectID, taskID, agentType string) (*RunInfo, error) {
	if err := ValidateProjectID(projectID); err != nil {
		return nil, err
	}
	if strings.TrimSpace(taskID) == "" {
		return nil, errors.New("task id is empty")
	}
	if strings.TrimSpace(agentType) == "" {
		return nil, errors.New("agent type is empty")
	}

	runID := s.newRunID()
	runDir := filepath.Join(s.root, projectID, taskID, "runs", runID)
	if err := os.MkdirAll(runDir, 0o755); err != nil {
		return nil, errors.Wrap(err, "create run directory")
	}

	pid := s.pid()
	info := &RunInfo{
		RunID:            runID,
		ProjectID:        projectID,
		TaskID:           taskID,
		AgentType:        agentType,
		ProcessOwnership: ProcessOwnershipManaged,
		PID:              pid,
		PGID:             pid,
		StartTime:        s.now().UTC(),
		ExitCode:         -1,
		Status:           StatusRunning,
	}
	path := filepath.Join(runDir, "run-info.yaml")
	if err := WriteRunInfo(path, info); err != nil {
		return nil, errors.Wrap(err, "write run-info")
	}

	s.trackRun(runID, path)
	return info, nil
}

// UpdateRunStatus updates run status, exit code, and end time.
func (s *FileStorage) UpdateRunStatus(runID string, status string, exitCode int) error {
	if strings.TrimSpace(runID) == "" {
		return errors.New("run id is empty")
	}
	if strings.TrimSpace(status) == "" {
		return errors.New("status is empty")
	}
	path, err := s.runInfoPath(runID)
	if err != nil {
		return err
	}
	return UpdateRunInfo(path, func(info *RunInfo) error {
		info.Status = status
		info.ExitCode = exitCode
		info.EndTime = s.now().UTC()
		return nil
	})
}

// GetRunInfo loads run metadata by run ID.
func (s *FileStorage) GetRunInfo(runID string) (*RunInfo, error) {
	if strings.TrimSpace(runID) == "" {
		return nil, errors.New("run id is empty")
	}
	path, err := s.runInfoPath(runID)
	if err != nil {
		return nil, err
	}
	info, err := ReadRunInfo(path)
	if err != nil {
		return nil, errors.Wrap(err, "read run-info")
	}
	return info, nil
}

// ListRuns lists run metadata for a project task.
func (s *FileStorage) ListRuns(projectID, taskID string) ([]*RunInfo, error) {
	if strings.TrimSpace(projectID) == "" {
		return nil, errors.New("project id is empty")
	}
	if strings.TrimSpace(taskID) == "" {
		return nil, errors.New("task id is empty")
	}
	baseDir := filepath.Join(s.root, projectID, taskID, "runs")
	entries, err := os.ReadDir(baseDir)
	if err != nil {
		return nil, errors.Wrap(err, "read runs directory")
	}

	runs := make([]*RunInfo, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		path := filepath.Join(baseDir, entry.Name(), "run-info.yaml")
		info, err := ReadRunInfo(path)
		if err != nil {
			if stderrors.Is(err, os.ErrNotExist) {
				obslog.Log(log.Default(), "DEBUG", "storage", "run_info_missing_synthesized",
					obslog.F("run_info_path", path),
					obslog.F("run_id", entry.Name()),
					obslog.F("project_id", projectID),
					obslog.F("task_id", taskID),
				)
				// When run-info.yaml is absent, synthesize minimal RunInfo from directory name.
				// This happens for older run directories created before run-info.yaml was introduced.
				info = &RunInfo{
					RunID:     entry.Name(),
					ProjectID: projectID,
					TaskID:    taskID,
					Status:    StatusUnknown,
					Version:   0,
				}
				runs = append(runs, info)
				continue
			}
			return nil, errors.Wrapf(err, "read run-info for run %s", entry.Name())
		}
		runs = append(runs, info)
	}

	sort.Slice(runs, func(i, j int) bool {
		return runs[i].RunID < runs[j].RunID
	})
	return runs, nil
}

func (s *FileStorage) runInfoPath(runID string) (string, error) {
	if path, ok := s.lookupRun(runID); ok {
		return path, nil
	}
	pattern := filepath.Join(s.root, "*", "*", "runs", runID, "run-info.yaml")
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return "", errors.Wrap(err, "glob run-info path")
	}
	if len(matches) == 0 {
		return "", fmt.Errorf("run-info not found for run id %s", runID)
	}
	if len(matches) > 1 {
		return "", fmt.Errorf("multiple run-info files found for run id %s", runID)
	}
	s.trackRun(runID, matches[0])
	return matches[0], nil
}

func (s *FileStorage) trackRun(runID, path string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.runIndex[runID] = path
}

func (s *FileStorage) lookupRun(runID string) (string, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	path, ok := s.runIndex[runID]
	return path, ok
}

// storageRunCounter ensures run IDs are globally unique within a process.
// The time format has only second precision, so two CreateRun calls in the
// same second from the same PID would produce identical IDs without this
// counter.  We append the counter value directly so every call gets a
// distinct ID even under rapid sequential creation.
var storageRunCounter uint64

func (s *FileStorage) newRunID() string {
	now := s.now().UTC()
	seq := atomic.AddUint64(&storageRunCounter, 1)
	stamp := now.Format("20060102-1504050000")
	return fmt.Sprintf("%s-%d-%d", stamp, s.pid(), seq)
}

var _ Storage = (*FileStorage)(nil)
