package api

import (
	"log"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/jonnyzzz/conductor-loop/internal/obslog"
	"github.com/jonnyzzz/conductor-loop/internal/runstate"
	"github.com/jonnyzzz/conductor-loop/internal/storage"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v3"
)

const (
	rootTaskPlannerStateVersion    = 1
	rootTaskPlannerStateDir        = ".conductor"
	rootTaskPlannerStateFileName   = "root-task-planner.yaml"
	rootTaskPlannerEntryQueued     = "queued"
	rootTaskPlannerEntryRunning    = "running"
	rootTaskPlannerFileMode        = 0o644
	rootTaskPlannerRunInfoGraceFor = 5 * time.Second
)

type taskQueueKey struct {
	ProjectID string
	TaskID    string
}

type taskQueueState struct {
	Queued        bool
	QueuePosition int
}

type rootTaskLaunch struct {
	Request   TaskCreateRequest
	RunID     string
	RunDir    string
	RunPrompt string
}

type rootTaskSubmitResult struct {
	Status        string
	QueuePosition int
	Launches      []rootTaskLaunch
}

type rootTaskPlanner struct {
	mu        sync.Mutex
	rootDir   string
	statePath string
	limit     int
	now       func() time.Time
	logger    *log.Logger
}

type rootTaskPlannerState struct {
	Version   int                    `yaml:"version"`
	Limit     int                    `yaml:"limit"`
	NextOrder int64                  `yaml:"next_order"`
	UpdatedAt time.Time              `yaml:"updated_at,omitempty"`
	Entries   []rootTaskPlannerEntry `yaml:"entries"`
}

type rootTaskPlannerEntry struct {
	RunID       string            `yaml:"run_id"`
	ProjectID   string            `yaml:"project_id"`
	TaskID      string            `yaml:"task_id"`
	RunDir      string            `yaml:"run_dir"`
	RunPrompt   string            `yaml:"run_prompt"`
	Request     TaskCreateRequest `yaml:"request"`
	SubmittedAt time.Time         `yaml:"submitted_at"`
	Order       int64             `yaml:"order"`
	State       string            `yaml:"state"`
	StartedAt   time.Time         `yaml:"started_at,omitempty"`
}

func newRootTaskPlanner(rootDir string, limit int, now func() time.Time, logger *log.Logger) *rootTaskPlanner {
	if now == nil {
		now = time.Now
	}
	if logger == nil {
		logger = log.Default()
	}
	return &rootTaskPlanner{
		rootDir:   filepath.Clean(strings.TrimSpace(rootDir)),
		statePath: filepath.Join(rootDir, rootTaskPlannerStateDir, rootTaskPlannerStateFileName),
		limit:     limit,
		now:       now,
		logger:    logger,
	}
}

func (p *rootTaskPlanner) Submit(req TaskCreateRequest, runDir, runPrompt string) (rootTaskSubmitResult, error) {
	if p == nil || p.limit <= 0 {
		runID := sanitizeRunID(filepath.Base(strings.TrimSpace(runDir)))
		return rootTaskSubmitResult{
			Status: "started",
			Launches: []rootTaskLaunch{{
				Request:   req,
				RunID:     runID,
				RunDir:    runDir,
				RunPrompt: runPrompt,
			}},
		}, nil
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	state, err := p.loadStateLocked()
	if err != nil {
		return rootTaskSubmitResult{}, err
	}
	if _, err := p.reconcileLocked(state); err != nil {
		return rootTaskSubmitResult{}, err
	}

	runID := sanitizeRunID(filepath.Base(strings.TrimSpace(runDir)))
	if runID == "" {
		return rootTaskSubmitResult{}, errors.New("planner run id is empty")
	}

	now := p.now().UTC()
	idx := -1
	for i := range state.Entries {
		if state.Entries[i].RunID == runID {
			idx = i
			break
		}
	}

	if idx < 0 {
		entry := rootTaskPlannerEntry{
			RunID:       runID,
			ProjectID:   strings.TrimSpace(req.ProjectID),
			TaskID:      strings.TrimSpace(req.TaskID),
			RunDir:      strings.TrimSpace(runDir),
			RunPrompt:   runPrompt,
			Request:     req,
			SubmittedAt: now,
			Order:       state.NextOrder,
			State:       rootTaskPlannerEntryQueued,
		}
		state.NextOrder++
		state.Entries = append(state.Entries, entry)
	} else {
		entry := &state.Entries[idx]
		entry.ProjectID = strings.TrimSpace(req.ProjectID)
		entry.TaskID = strings.TrimSpace(req.TaskID)
		entry.RunDir = strings.TrimSpace(runDir)
		entry.RunPrompt = runPrompt
		entry.Request = req
		if entry.SubmittedAt.IsZero() {
			entry.SubmittedAt = now
		}
		if entry.Order <= 0 {
			entry.Order = state.NextOrder
			state.NextOrder++
		}
		entry.State = rootTaskPlannerEntryQueued
		entry.StartedAt = time.Time{}
	}

	launchEntries := p.scheduleLocked(state)
	if err := p.saveStateLocked(state); err != nil {
		return rootTaskSubmitResult{}, err
	}

	result := rootTaskSubmitResult{
		Status:   "queued",
		Launches: plannerEntriesToLaunches(launchEntries),
	}

	positions := queuedRunPositions(state.Entries)
	for _, entry := range state.Entries {
		if entry.RunID != runID {
			continue
		}
		if entry.State == rootTaskPlannerEntryRunning {
			result.Status = "started"
			result.QueuePosition = 0
			return result, nil
		}
		if pos, ok := positions[runID]; ok {
			result.QueuePosition = pos
		}
		return result, nil
	}

	return result, nil
}

func (p *rootTaskPlanner) OnRunFinished(projectID, taskID, runID string) ([]rootTaskLaunch, error) {
	return p.OnRunFinishedWithScheduling(projectID, taskID, runID, true)
}

func (p *rootTaskPlanner) OnRunFinishedWithScheduling(projectID, taskID, runID string, allowSchedule bool) ([]rootTaskLaunch, error) {
	if p == nil || p.limit <= 0 {
		return nil, nil
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	state, err := p.loadStateLocked()
	if err != nil {
		return nil, err
	}
	if _, err := p.reconcileLocked(state); err != nil {
		return nil, err
	}

	runID = sanitizeRunID(runID)
	projectID = strings.TrimSpace(projectID)
	taskID = strings.TrimSpace(taskID)
	filtered := state.Entries[:0]
	removed := false
	for _, entry := range state.Entries {
		matchRun := runID != "" && entry.RunID == runID
		matchTaskFallback := runID == "" && entry.ProjectID == projectID && entry.TaskID == taskID && entry.State == rootTaskPlannerEntryRunning
		if matchRun || matchTaskFallback {
			removed = true
			continue
		}
		filtered = append(filtered, entry)
	}
	state.Entries = filtered
	if !removed {
		obslog.Log(p.logger, "INFO", "api", "root_task_planner_finish_noop",
			obslog.F("project_id", projectID),
			obslog.F("task_id", taskID),
			obslog.F("run_id", runID),
		)
	}

	var launchEntries []rootTaskPlannerEntry
	if allowSchedule {
		launchEntries = p.scheduleLocked(state)
	}
	if err := p.saveStateLocked(state); err != nil {
		return nil, err
	}
	return plannerEntriesToLaunches(launchEntries), nil
}

func (p *rootTaskPlanner) DropQueuedForTask(projectID, taskID string) ([]rootTaskLaunch, error) {
	if p == nil || p.limit <= 0 {
		return nil, nil
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	state, err := p.loadStateLocked()
	if err != nil {
		return nil, err
	}
	if _, err := p.reconcileLocked(state); err != nil {
		return nil, err
	}

	projectID = strings.TrimSpace(projectID)
	taskID = strings.TrimSpace(taskID)
	filtered := state.Entries[:0]
	for _, entry := range state.Entries {
		if entry.ProjectID == projectID && entry.TaskID == taskID && entry.State == rootTaskPlannerEntryQueued {
			continue
		}
		filtered = append(filtered, entry)
	}
	state.Entries = filtered

	launchEntries := p.scheduleLocked(state)
	if err := p.saveStateLocked(state); err != nil {
		return nil, err
	}
	return plannerEntriesToLaunches(launchEntries), nil
}

func (p *rootTaskPlanner) Snapshot() (map[taskQueueKey]taskQueueState, error) {
	if p == nil || p.limit <= 0 {
		return map[taskQueueKey]taskQueueState{}, nil
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	state, err := p.loadStateLocked()
	if err != nil {
		return nil, err
	}
	changed, err := p.reconcileLocked(state)
	if err != nil {
		return nil, err
	}
	if changed {
		if err := p.saveStateLocked(state); err != nil {
			return nil, err
		}
	}
	return queueSnapshotFromEntries(state.Entries), nil
}

func (p *rootTaskPlanner) Recover() ([]rootTaskLaunch, error) {
	if p == nil || p.limit <= 0 {
		return nil, nil
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	state, err := p.loadStateLocked()
	if err != nil {
		return nil, err
	}
	if _, err := p.reconcileLocked(state); err != nil {
		return nil, err
	}
	launchEntries := p.scheduleLocked(state)
	if err := p.saveStateLocked(state); err != nil {
		return nil, err
	}
	return plannerEntriesToLaunches(launchEntries), nil
}

func (p *rootTaskPlanner) scheduleLocked(state *rootTaskPlannerState) []rootTaskPlannerEntry {
	if p == nil || state == nil || p.limit <= 0 {
		return nil
	}

	runningByRunID := make(map[string]struct{})
	runningCount := 0
	for _, entry := range state.Entries {
		if entry.State != rootTaskPlannerEntryRunning {
			continue
		}
		runningByRunID[entry.RunID] = struct{}{}
		runningCount++
	}

	runningCount += p.externalRunningRootCountLocked(runningByRunID)
	available := p.limit - runningCount
	if available <= 0 {
		return nil
	}

	queuedIndexes := make([]int, 0)
	for idx, entry := range state.Entries {
		if entry.State == rootTaskPlannerEntryQueued {
			queuedIndexes = append(queuedIndexes, idx)
		}
	}
	sort.Slice(queuedIndexes, func(i, j int) bool {
		return plannerEntryLess(state.Entries[queuedIndexes[i]], state.Entries[queuedIndexes[j]])
	})

	now := p.now().UTC()
	launches := make([]rootTaskPlannerEntry, 0, available)
	for _, idx := range queuedIndexes {
		if available == 0 {
			break
		}
		state.Entries[idx].State = rootTaskPlannerEntryRunning
		if state.Entries[idx].StartedAt.IsZero() {
			state.Entries[idx].StartedAt = now
		}
		launches = append(launches, state.Entries[idx])
		available--
	}
	return launches
}

func (p *rootTaskPlanner) reconcileLocked(state *rootTaskPlannerState) (bool, error) {
	if state == nil {
		return false, errors.New("planner state is nil")
	}

	changed := false
	if state.Version != rootTaskPlannerStateVersion {
		state.Version = rootTaskPlannerStateVersion
		changed = true
	}
	if state.Limit != p.limit {
		state.Limit = p.limit
		changed = true
	}
	if state.NextOrder <= 0 {
		state.NextOrder = 1
		changed = true
	}

	sort.Slice(state.Entries, func(i, j int) bool {
		return plannerEntryLess(state.Entries[i], state.Entries[j])
	})

	now := p.now().UTC()
	normalized := make([]rootTaskPlannerEntry, 0, len(state.Entries))
	seenRunIDs := make(map[string]struct{})
	for _, entry := range state.Entries {
		entry.RunID = sanitizeRunID(entry.RunID)
		entry.ProjectID = strings.TrimSpace(entry.ProjectID)
		entry.TaskID = strings.TrimSpace(entry.TaskID)
		entry.RunDir = strings.TrimSpace(entry.RunDir)
		if entry.RunID == "" || entry.ProjectID == "" || entry.TaskID == "" || entry.RunDir == "" {
			changed = true
			continue
		}

		if _, seen := seenRunIDs[entry.RunID]; seen {
			changed = true
			continue
		}
		seenRunIDs[entry.RunID] = struct{}{}

		if entry.SubmittedAt.IsZero() {
			entry.SubmittedAt = now
			changed = true
		}
		if entry.Order <= 0 {
			entry.Order = state.NextOrder
			state.NextOrder++
			changed = true
		}
		if entry.State != rootTaskPlannerEntryQueued && entry.State != rootTaskPlannerEntryRunning {
			entry.State = rootTaskPlannerEntryQueued
			entry.StartedAt = time.Time{}
			changed = true
		}

		done := taskDoneMarkerExists(p.rootDir, entry.ProjectID, entry.TaskID)
		if done && entry.State == rootTaskPlannerEntryQueued {
			changed = true
			continue
		}

		runInfoPath := filepath.Join(entry.RunDir, "run-info.yaml")
		runInfo, err := runstate.ReadRunInfo(runInfoPath)
		if err != nil {
			cause := errors.Cause(err)
			if os.IsNotExist(cause) || os.IsNotExist(err) {
				if entry.State == rootTaskPlannerEntryRunning {
					if entry.StartedAt.IsZero() || now.Sub(entry.StartedAt) > rootTaskPlannerRunInfoGraceFor {
						entry.State = rootTaskPlannerEntryQueued
						entry.StartedAt = time.Time{}
						changed = true
						if done {
							changed = true
							continue
						}
					}
				}
				normalized = append(normalized, entry)
				continue
			}
			return changed, errors.Wrapf(err, "read planner run-info for %s/%s", entry.ProjectID, entry.TaskID)
		}
		if strings.TrimSpace(runInfo.Status) == storage.StatusUnknown {
			if entry.State == rootTaskPlannerEntryRunning {
				if entry.StartedAt.IsZero() || now.Sub(entry.StartedAt) > rootTaskPlannerRunInfoGraceFor {
					entry.State = rootTaskPlannerEntryQueued
					entry.StartedAt = time.Time{}
					changed = true
					if done {
						changed = true
						continue
					}
				}
			}
			normalized = append(normalized, entry)
			continue
		}

		if isRunningRootRun(runInfo) {
			if entry.State != rootTaskPlannerEntryRunning {
				entry.State = rootTaskPlannerEntryRunning
				changed = true
			}
			if entry.StartedAt.IsZero() && !runInfo.StartTime.IsZero() {
				entry.StartedAt = runInfo.StartTime.UTC()
				changed = true
			}
			normalized = append(normalized, entry)
			continue
		}

		// Completed/failed root runs are terminal for planner slots.
		changed = true
	}

	state.Entries = normalized
	maxOrder := int64(0)
	for _, entry := range state.Entries {
		if entry.Order > maxOrder {
			maxOrder = entry.Order
		}
	}
	if state.NextOrder <= maxOrder {
		state.NextOrder = maxOrder + 1
		changed = true
	}
	if state.NextOrder <= 0 {
		state.NextOrder = 1
		changed = true
	}

	return changed, nil
}

func (p *rootTaskPlanner) externalRunningRootCountLocked(excluding map[string]struct{}) int {
	infos, err := allRunInfos(p.rootDir)
	if err != nil {
		obslog.Log(p.logger, "WARN", "api", "root_task_planner_scan_failed",
			obslog.F("root_dir", p.rootDir),
			obslog.F("error", err),
		)
		return 0
	}

	count := 0
	for _, info := range infos {
		if !isRunningRootRun(info) {
			continue
		}
		if _, skip := excluding[info.RunID]; skip {
			continue
		}
		count++
	}
	return count
}

func (p *rootTaskPlanner) loadStateLocked() (*rootTaskPlannerState, error) {
	data, err := os.ReadFile(p.statePath)
	if err != nil {
		if os.IsNotExist(err) {
			return defaultRootTaskPlannerState(p.limit), nil
		}
		return nil, errors.Wrap(err, "read root task planner state")
	}
	if len(strings.TrimSpace(string(data))) == 0 {
		return defaultRootTaskPlannerState(p.limit), nil
	}
	var state rootTaskPlannerState
	if err := yaml.Unmarshal(data, &state); err != nil {
		return nil, errors.Wrap(err, "unmarshal root task planner state")
	}
	if state.Entries == nil {
		state.Entries = []rootTaskPlannerEntry{}
	}
	return &state, nil
}

func (p *rootTaskPlanner) saveStateLocked(state *rootTaskPlannerState) error {
	if state == nil {
		return errors.New("planner state is nil")
	}

	state.Version = rootTaskPlannerStateVersion
	state.Limit = p.limit
	state.UpdatedAt = p.now().UTC()
	if state.NextOrder <= 0 {
		state.NextOrder = 1
	}
	if state.Entries == nil {
		state.Entries = []rootTaskPlannerEntry{}
	}

	data, err := yaml.Marshal(state)
	if err != nil {
		return errors.Wrap(err, "marshal root task planner state")
	}

	if err := os.MkdirAll(filepath.Dir(p.statePath), 0o755); err != nil {
		return errors.Wrap(err, "create root task planner state directory")
	}
	if err := writePlannerFileAtomic(p.statePath, data); err != nil {
		return errors.Wrap(err, "write root task planner state")
	}
	return nil
}

func writePlannerFileAtomic(path string, data []byte) error {
	dir := filepath.Dir(path)
	tmpFile, err := os.CreateTemp(dir, "root-task-planner.*.yaml.tmp")
	if err != nil {
		return errors.Wrap(err, "create temp planner file")
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
		return errors.Wrap(err, "write temp planner file")
	}
	if err := tmpFile.Sync(); err != nil {
		return errors.Wrap(err, "fsync temp planner file")
	}
	if err := tmpFile.Chmod(rootTaskPlannerFileMode); err != nil {
		return errors.Wrap(err, "chmod temp planner file")
	}
	if err := tmpFile.Close(); err != nil {
		return errors.Wrap(err, "close temp planner file")
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
		return errors.Wrap(err, "rename temp planner file")
	}
	success = true
	return nil
}

func defaultRootTaskPlannerState(limit int) *rootTaskPlannerState {
	return &rootTaskPlannerState{
		Version:   rootTaskPlannerStateVersion,
		Limit:     limit,
		NextOrder: 1,
		Entries:   []rootTaskPlannerEntry{},
	}
}

func queueSnapshotFromEntries(entries []rootTaskPlannerEntry) map[taskQueueKey]taskQueueState {
	snapshot := make(map[taskQueueKey]taskQueueState)
	queued := make([]rootTaskPlannerEntry, 0)
	for _, entry := range entries {
		if entry.State == rootTaskPlannerEntryQueued {
			queued = append(queued, entry)
		}
	}
	sort.Slice(queued, func(i, j int) bool {
		return plannerEntryLess(queued[i], queued[j])
	})
	for idx, entry := range queued {
		key := taskQueueKey{ProjectID: entry.ProjectID, TaskID: entry.TaskID}
		if _, exists := snapshot[key]; exists {
			continue
		}
		snapshot[key] = taskQueueState{
			Queued:        true,
			QueuePosition: idx + 1,
		}
	}
	return snapshot
}

func queuedRunPositions(entries []rootTaskPlannerEntry) map[string]int {
	positions := make(map[string]int)
	queued := make([]rootTaskPlannerEntry, 0)
	for _, entry := range entries {
		if entry.State == rootTaskPlannerEntryQueued {
			queued = append(queued, entry)
		}
	}
	sort.Slice(queued, func(i, j int) bool {
		return plannerEntryLess(queued[i], queued[j])
	})
	for idx, entry := range queued {
		positions[entry.RunID] = idx + 1
	}
	return positions
}

func plannerEntriesToLaunches(entries []rootTaskPlannerEntry) []rootTaskLaunch {
	if len(entries) == 0 {
		return nil
	}
	launches := make([]rootTaskLaunch, 0, len(entries))
	for _, entry := range entries {
		launches = append(launches, rootTaskLaunch{
			Request:   entry.Request,
			RunID:     entry.RunID,
			RunDir:    entry.RunDir,
			RunPrompt: entry.RunPrompt,
		})
	}
	return launches
}

func plannerEntryLess(a, b rootTaskPlannerEntry) bool {
	if a.Order != b.Order {
		return a.Order < b.Order
	}
	if !a.SubmittedAt.Equal(b.SubmittedAt) {
		return a.SubmittedAt.Before(b.SubmittedAt)
	}
	if a.ProjectID != b.ProjectID {
		return a.ProjectID < b.ProjectID
	}
	if a.TaskID != b.TaskID {
		return a.TaskID < b.TaskID
	}
	return a.RunID < b.RunID
}

func sanitizeRunID(value string) string {
	trimmed := strings.TrimSpace(value)
	switch trimmed {
	case "", ".", string(filepath.Separator):
		return ""
	default:
		return trimmed
	}
}

func taskDoneMarkerExists(rootDir, projectID, taskID string) bool {
	donePath := filepath.Join(rootDir, projectID, taskID, "DONE")
	if _, err := os.Stat(donePath); err == nil {
		return true
	}
	return false
}

func isRunningRootRun(info *storage.RunInfo) bool {
	if info == nil {
		return false
	}
	if strings.TrimSpace(info.ParentRunID) != "" {
		return false
	}
	if info.Status != storage.StatusRunning {
		return false
	}
	if !info.EndTime.IsZero() {
		return false
	}
	return true
}
