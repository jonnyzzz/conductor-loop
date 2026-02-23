package api

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/jonnyzzz/conductor-loop/internal/obslog"
	"github.com/jonnyzzz/conductor-loop/internal/runner"
	"github.com/jonnyzzz/conductor-loop/internal/runstate"
	"github.com/jonnyzzz/conductor-loop/internal/storage"
	"github.com/jonnyzzz/conductor-loop/internal/taskdeps"
	"github.com/pkg/errors"
)

// paginatedResponse is the JSON envelope for paginated list responses.
type paginatedResponse struct {
	Items   any  `json:"items"`
	Total   int  `json:"total"`
	Limit   int  `json:"limit"`
	Offset  int  `json:"offset"`
	HasMore bool `json:"has_more"`
}

// parsePagination parses `limit` and `offset` query parameters.
// Default: limit=50, offset=0. Max limit: 500.
func parsePagination(r *http.Request) (limit, offset int) {
	limit = 50
	offset = 0
	if v := r.URL.Query().Get("limit"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n > 0 {
			limit = n
		}
	}
	if limit > 500 {
		limit = 500
	}
	if v := r.URL.Query().Get("offset"); v != "" {
		if n, err := strconv.Atoi(v); err == nil && n >= 0 {
			offset = n
		}
	}
	return limit, offset
}

func parseActiveOnlyQuery(raw string) bool {
	switch strings.ToLower(strings.TrimSpace(raw)) {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}

func parseSelectedTaskLimitQuery(raw string) (int, *apiError) {
	trimmed := strings.TrimSpace(raw)
	if trimmed == "" {
		return 0, nil
	}
	n, err := strconv.Atoi(trimmed)
	if err != nil || n < 0 {
		return 0, &apiError{
			Status:  http.StatusBadRequest,
			Code:    "BAD_REQUEST",
			Message: "invalid selected_task_limit",
			Details: map[string]string{"selected_task_limit": "must be a non-negative integer"},
		}
	}
	// Guardrails for high-frequency polling endpoint.
	if n > 200 {
		n = 200
	}
	return n, nil
}

// RunFile describes a file available for a run.
type RunFile struct {
	Name  string `json:"name"`  // logical name for ?name=X
	Label string `json:"label"` // display label
}

// projectRun is the run summary shape the project API returns.
type projectRun struct {
	ID               string     `json:"id"`
	Agent            string     `json:"agent"`
	AgentVersion     string     `json:"agent_version,omitempty"`
	Status           string     `json:"status"`
	ProcessOwnership string     `json:"process_ownership,omitempty"`
	ExitCode         int        `json:"exit_code"`
	StartTime        time.Time  `json:"start_time"`
	EndTime          *time.Time `json:"end_time,omitempty"`
	ParentRunID      string     `json:"parent_run_id,omitempty"`
	PreviousRunID    string     `json:"previous_run_id,omitempty"`
	ErrorSummary     string     `json:"error_summary,omitempty"`
	Files            []RunFile  `json:"files,omitempty"`
}

// projectTask is the task shape the project API returns.
type projectTask struct {
	ID            string                 `json:"id"`
	ProjectID     string                 `json:"project_id"`
	Status        string                 `json:"status"`
	QueuePosition int                    `json:"queue_position,omitempty"`
	LastActivity  time.Time              `json:"last_activity"`
	CreatedAt     time.Time              `json:"created_at"`
	Done          bool                   `json:"done"`
	State         string                 `json:"state"`
	DependsOn     []string               `json:"depends_on,omitempty"`
	BlockedBy     []string               `json:"blocked_by,omitempty"`
	ThreadParent  *ThreadParentReference `json:"thread_parent,omitempty"`
	Runs          []projectRun           `json:"runs"`
}

// projectSummary is the project list item shape.
type projectSummary struct {
	ID           string    `json:"id"`
	LastActivity time.Time `json:"last_activity"`
	TaskCount    int       `json:"task_count"`
	ProjectRoot  string    `json:"project_root,omitempty"`
}

type projectCreateRequest struct {
	ProjectID   string `json:"project_id"`
	ProjectRoot string `json:"project_root"`
}

type projectRootMarker struct {
	ProjectRoot  string
	LastActivity time.Time
}

const projectRootMarkerFile = "PROJECT-ROOT.txt"

// flatRunItem is the shape of a single run in the flat runs endpoint response.
type flatRunItem struct {
	ID            string     `json:"id"`
	TaskID        string     `json:"task_id"`
	Agent         string     `json:"agent"`
	Status        string     `json:"status"`
	ExitCode      int        `json:"exit_code"`
	StartTime     time.Time  `json:"start_time"`
	EndTime       *time.Time `json:"end_time,omitempty"`
	ParentRunID   string     `json:"parent_run_id,omitempty"`
	PreviousRunID string     `json:"previous_run_id,omitempty"`
}

// flatRunsResponse is the JSON response for GET /api/projects/{p}/runs/flat.
type flatRunsResponse struct {
	Runs []flatRunItem `json:"runs"`
}

// handleProjectsRouter dispatches /api/projects/{...} sub-paths.
func (s *Server) handleProjectsRouter(w http.ResponseWriter, r *http.Request) *apiError {
	parts := splitPath(r.URL.Path, "/api/projects/")
	if len(parts) == 0 {
		return apiErrorNotFound("not found")
	}
	projectID := parts[0]
	if err := validateIdentifier(projectID, "project_id"); err != nil {
		return err
	}
	// /api/projects/{id}
	if len(parts) == 1 {
		if r.Method == http.MethodDelete {
			if err := rejectUIDestructiveAction(r, "project deletion"); err != nil {
				return err
			}
			return s.handleProjectDelete(w, r, projectID)
		}
		return s.handleProjectDetail(w, r)
	}
	// /api/projects/{id}/stats
	if parts[1] == "stats" {
		return s.handleProjectStats(w, r)
	}
	// /api/projects/{id}/runs/flat
	if parts[1] == "runs" && len(parts) == 3 && parts[2] == "flat" {
		return s.handleProjectRunsFlat(w, r)
	}
	// /api/projects/{id}/tasks[/...]
	if parts[1] == "tasks" {
		if len(parts) == 2 {
			return s.handleProjectTasks(w, r)
		}
		return s.handleProjectTask(w, r)
	}
	// /api/projects/{id}/messages[/stream]
	if parts[1] == "messages" {
		if len(parts) == 2 {
			return s.handleProjectMessages(w, r)
		}
		if len(parts) == 3 && parts[2] == "stream" {
			return s.handleProjectMessagesStream(w, r)
		}
	}
	// /api/projects/{id}/gc
	if parts[1] == "gc" {
		return s.handleProjectGC(w, r)
	}
	return apiErrorNotFound("not found")
}

// handleProjectRunsFlat serves GET /api/projects/{p}/runs/flat.
// It returns a flat list of all runs across all tasks for the project,
// with task_id, parent_run_id, and previous_run_id for client-side tree building.
func (s *Server) handleProjectRunsFlat(w http.ResponseWriter, r *http.Request) *apiError {
	if r.Method != http.MethodGet {
		return apiErrorMethodNotAllowed()
	}
	parts := splitPath(r.URL.Path, "/api/projects/")
	if len(parts) < 3 {
		return apiErrorNotFound("not found")
	}
	projectID := parts[0]
	if err := validateIdentifier(projectID, "project_id"); err != nil {
		return err
	}
	selectedTaskID := strings.TrimSpace(r.URL.Query().Get("selected_task_id"))
	if selectedTaskID != "" {
		if err := validateIdentifier(selectedTaskID, "selected_task_id"); err != nil {
			return err
		}
	}
	selectedTaskLimit, limitErr := parseSelectedTaskLimitQuery(r.URL.Query().Get("selected_task_limit"))
	if limitErr != nil {
		return limitErr
	}
	activeOnly := parseActiveOnlyQuery(r.URL.Query().Get("active_only"))

	projectRuns, err := s.projectRunInfos(projectID)
	if err != nil {
		return apiErrorInternal("scan runs", err)
	}

	var items []flatRunItem
	if selectedTaskID == "" && !activeOnly {
		items = make([]flatRunItem, 0, len(projectRuns))
		for _, run := range projectRuns {
			items = appendFlatRunItem(items, run)
		}
	} else {
		includeRunIDs := filterProjectRunIDs(projectRuns, selectedTaskID, selectedTaskLimit, activeOnly)
		items = make([]flatRunItem, 0, len(includeRunIDs))
		for _, run := range projectRuns {
			if run == nil {
				continue
			}
			if _, ok := includeRunIDs[run.RunID]; !ok {
				continue
			}
			items = appendFlatRunItem(items, run)
		}
	}
	if items == nil {
		items = []flatRunItem{}
	}
	return writeJSON(w, http.StatusOK, flatRunsResponse{Runs: items})
}

func appendFlatRunItem(items []flatRunItem, run *storage.RunInfo) []flatRunItem {
	if run == nil {
		return items
	}

	item := flatRunItem{
		ID:            run.RunID,
		TaskID:        run.TaskID,
		Agent:         run.AgentType,
		Status:        run.Status,
		ExitCode:      run.ExitCode,
		StartTime:     run.StartTime,
		ParentRunID:   run.ParentRunID,
		PreviousRunID: run.PreviousRunID,
	}
	if !run.EndTime.IsZero() {
		item.EndTime = &run.EndTime
	}

	return append(items, item)
}

func isActiveFlatRunStatus(status string) bool {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "running", "queued":
		return true
	default:
		return false
	}
}

func initialIncludeRunIDCapacity(totalRuns int, selectedTaskID string, selectedTaskLimit int, activeOnly bool) int {
	if totalRuns <= 0 {
		return 0
	}
	if selectedTaskID == "" && !activeOnly {
		return totalRuns
	}

	estimated := 32
	if selectedTaskLimit > 0 {
		estimated += selectedTaskLimit * 4
	} else if selectedTaskID != "" {
		estimated += 64
	}
	if activeOnly {
		estimated += 64
	}
	if estimated > totalRuns {
		return totalRuns
	}
	return estimated
}

func filterProjectRunIDs(
	runs []*storage.RunInfo,
	selectedTaskID string,
	selectedTaskLimit int,
	activeOnly bool,
) map[string]struct{} {
	includeRunIDs := make(map[string]struct{}, initialIncludeRunIDCapacity(len(runs), selectedTaskID, selectedTaskLimit, activeOnly))
	if len(runs) == 0 {
		return includeRunIDs
	}

	runByID := make(map[string]*storage.RunInfo, len(runs))
	childrenByParent := make(map[string][]string, len(runs))
	selectedTaskRunsCap := 16
	if selectedTaskLimit > 0 && selectedTaskLimit < selectedTaskRunsCap {
		selectedTaskRunsCap = selectedTaskLimit
	}
	selectedTaskRuns := make([]*storage.RunInfo, 0, selectedTaskRunsCap)
	includeCrossTaskDescendantsOnly := false

	for _, run := range runs {
		if run == nil {
			continue
		}
		runByID[run.RunID] = run
		if run.ParentRunID != "" {
			childrenByParent[run.ParentRunID] = append(childrenByParent[run.ParentRunID], run.RunID)
		}
		if selectedTaskID != "" && run.TaskID == selectedTaskID {
			if selectedTaskLimit > 0 {
				selectedTaskRuns = append(selectedTaskRuns, run)
			} else {
				includeRunIDs[run.RunID] = struct{}{}
			}
		}
		if activeOnly && isActiveFlatRunStatus(run.Status) {
			includeRunIDs[run.RunID] = struct{}{}
		}
	}

	if selectedTaskID != "" && selectedTaskLimit > 0 {
		seedLatestRunInfos(selectedTaskRuns, selectedTaskLimit, includeRunIDs)
		// Selected-task filtering can otherwise drop bridge runs from ancestor
		// tasks when the selected branch only keeps detached/latest runs. Seed
		// anchors using selected-run ancestry context so unrelated newer anchors
		// in the same task do not override the selected branch hierarchy.
		seedSelectedTaskParentAnchorRunInfos(
			selectedTaskRuns,
			selectedTaskLimit,
			runs,
			runByID,
			includeRunIDs,
		)
	}
	if activeOnly && selectedTaskID == "" && len(includeRunIDs) > 0 {
		// Preserve cross-task parent anchors across terminal branches, even when
		// the active seed comes from an unrelated task. Without these anchors,
		// clients lose parent_run_id edges needed to keep nested tasks attached.
		seedLatestParentAnchorRunInfos(runs, runByID, includeRunIDs)
	}
	if len(includeRunIDs) == 0 && activeOnly && selectedTaskID == "" {
		// Idle project fallback: keep only the latest run per task and its
		// ancestry to preserve cross-task nesting edges. Returning full run
		// history here caused large payloads and slow tree refreshes.
		seedLatestTaskRunInfos(runs, includeRunIDs)
		// Preserve cross-task parent anchors when the latest task run is a local
		// restart without parent/previous pointers.
		seedLatestParentAnchorRunInfos(runs, runByID, includeRunIDs)
		// Keep descendant expansion in fallback mode, but only for cross-task
		// edges so we preserve nesting links without rehydrating full history.
		includeCrossTaskDescendantsOnly = true
	}
	if len(includeRunIDs) == 0 {
		return includeRunIDs
	}

	seedRunIDs := make([]string, 0, len(includeRunIDs))
	for runID := range includeRunIDs {
		seedRunIDs = append(seedRunIDs, runID)
	}
	descendantSeedRunIDs := append([]string(nil), seedRunIDs...)
	descendantSeedRunSet := make(map[string]struct{}, len(descendantSeedRunIDs))
	for _, runID := range descendantSeedRunIDs {
		descendantSeedRunSet[runID] = struct{}{}
	}

	// Expand lineage transitively across both parent_run_id and previous_run_id
	// for every included run. This preserves cross-task bridge runs when parent
	// tasks restart and only older runs carry the parent linkage.
	lineageQueue := append([]string(nil), seedRunIDs...)
	lineageQueued := make(map[string]struct{}, len(lineageQueue))
	for _, runID := range lineageQueue {
		lineageQueued[runID] = struct{}{}
	}
	for index := 0; index < len(lineageQueue); index++ {
		runID := lineageQueue[index]
		run, ok := runByID[runID]
		if !ok || run == nil {
			continue
		}

		parentCursor := run
		parentGuard := 0
		for parentCursor.ParentRunID != "" {
			parentRunID := parentCursor.ParentRunID
			includeRunIDs[parentRunID] = struct{}{}
			if _, seen := lineageQueued[parentRunID]; !seen {
				lineageQueued[parentRunID] = struct{}{}
				lineageQueue = append(lineageQueue, parentRunID)
			}

			next, exists := runByID[parentRunID]
			parentGuard += 1
			if !exists || next == nil || parentGuard > len(runs) {
				break
			}
			parentCursor = next
		}

		previousCursor := run
		previousGuard := 0
		for previousCursor.PreviousRunID != "" {
			previousRunID := previousCursor.PreviousRunID
			includeRunIDs[previousRunID] = struct{}{}
			if _, seen := descendantSeedRunSet[previousRunID]; !seen {
				descendantSeedRunSet[previousRunID] = struct{}{}
				descendantSeedRunIDs = append(descendantSeedRunIDs, previousRunID)
			}
			if _, seen := lineageQueued[previousRunID]; !seen {
				lineageQueued[previousRunID] = struct{}{}
				lineageQueue = append(lineageQueue, previousRunID)
			}

			next, exists := runByID[previousRunID]
			previousGuard += 1
			if !exists || next == nil || previousGuard > len(runs) {
				break
			}
			previousCursor = next
		}
	}

	descendantQueue := append([]string(nil), descendantSeedRunIDs...)
	for len(descendantQueue) > 0 {
		runID := descendantQueue[0]
		descendantQueue = descendantQueue[1:]

		parentRun, parentOK := runByID[runID]
		for _, childRunID := range childrenByParent[runID] {
			if _, seen := includeRunIDs[childRunID]; seen {
				continue
			}
			childRun, childOK := runByID[childRunID]
			if includeCrossTaskDescendantsOnly &&
				parentOK && parentRun != nil &&
				childOK && childRun != nil &&
				childRun.TaskID == parentRun.TaskID &&
				len(childrenByParent[childRunID]) == 0 {
				continue
			}
			includeRunIDs[childRunID] = struct{}{}
			descendantQueue = append(descendantQueue, childRunID)
		}
	}

	return includeRunIDs
}

func seedLatestTaskRunInfos(runs []*storage.RunInfo, includeRunIDs map[string]struct{}) {
	latestByTask := make(map[string]*storage.RunInfo, len(runs))
	for _, run := range runs {
		if run == nil {
			continue
		}
		current, ok := latestByTask[run.TaskID]
		if !ok || current == nil {
			latestByTask[run.TaskID] = run
			continue
		}
		if runInfoLater(run, current) {
			latestByTask[run.TaskID] = run
		}
	}
	for _, run := range latestByTask {
		if run != nil {
			includeRunIDs[run.RunID] = struct{}{}
		}
	}
}

func seedLatestRunInfos(runs []*storage.RunInfo, limit int, includeRunIDs map[string]struct{}) {
	for _, run := range latestRunInfos(runs, limit) {
		includeRunIDs[run.RunID] = struct{}{}
	}
}

func latestRunInfos(runs []*storage.RunInfo, limit int) []*storage.RunInfo {
	if limit <= 0 || len(runs) == 0 {
		return nil
	}
	latestRuns := make([]*storage.RunInfo, 0, limit)
	for _, run := range runs {
		latestRuns = insertLatestRunInfo(latestRuns, run, limit)
	}
	return latestRuns
}

func seedSelectedTaskParentAnchorRunInfos(
	selectedTaskRuns []*storage.RunInfo,
	selectedTaskLimit int,
	runs []*storage.RunInfo,
	runByID map[string]*storage.RunInfo,
	includeRunIDs map[string]struct{},
) {
	if len(selectedTaskRuns) == 0 || selectedTaskLimit <= 0 || len(runs) == 0 {
		return
	}

	latestSelectedRuns := latestRunInfos(selectedTaskRuns, selectedTaskLimit)
	if len(latestSelectedRuns) == 0 {
		return
	}

	taskReferenceActivity := make(map[string]time.Time, len(latestSelectedRuns))
	ancestryQueue := make([]*storage.RunInfo, 0, len(latestSelectedRuns)*2)
	visitedRuns := make(map[string]struct{}, len(latestSelectedRuns))
	queueRun := func(run *storage.RunInfo) {
		if run == nil || strings.TrimSpace(run.RunID) == "" {
			return
		}
		if _, seen := visitedRuns[run.RunID]; seen {
			return
		}
		visitedRuns[run.RunID] = struct{}{}
		ancestryQueue = append(ancestryQueue, run)
	}
	for _, run := range latestSelectedRuns {
		queueRun(run)
	}

	for index := 0; index < len(ancestryQueue); index++ {
		run := ancestryQueue[index]
		if run == nil {
			continue
		}

		activity := runInfoActivityTime(run)
		if current, ok := taskReferenceActivity[run.TaskID]; !ok || activity.After(current) {
			taskReferenceActivity[run.TaskID] = activity
		}

		if run.ParentRunID != "" {
			queueRun(runByID[run.ParentRunID])
		}
		if run.PreviousRunID != "" {
			queueRun(runByID[run.PreviousRunID])
		}
	}

	if len(taskReferenceActivity) == 0 {
		return
	}

	hasIncludedCrossTaskParent := make(map[string]bool, len(taskReferenceActivity))
	for runID := range includeRunIDs {
		run, ok := runByID[runID]
		if !ok || run == nil || run.ParentRunID == "" {
			continue
		}
		parentRun, parentOK := runByID[run.ParentRunID]
		if !parentOK || parentRun == nil || parentRun.TaskID == run.TaskID {
			continue
		}
		hasIncludedCrossTaskParent[run.TaskID] = true
	}

	anchorRunsByTask := make(map[string][]*storage.RunInfo, len(taskReferenceActivity))
	for _, run := range runs {
		if run == nil || run.ParentRunID == "" {
			continue
		}
		parentRun, ok := runByID[run.ParentRunID]
		if !ok || parentRun == nil || parentRun.TaskID == run.TaskID {
			continue
		}
		anchorRunsByTask[run.TaskID] = append(anchorRunsByTask[run.TaskID], run)
	}

	for taskID, referenceTime := range taskReferenceActivity {
		if hasIncludedCrossTaskParent[taskID] {
			continue
		}
		candidates := anchorRunsByTask[taskID]
		if len(candidates) == 0 {
			continue
		}

		var bestAtOrBeforeReference *storage.RunInfo
		var fallbackLatest *storage.RunInfo
		for _, candidate := range candidates {
			if candidate == nil {
				continue
			}
			if fallbackLatest == nil || runInfoLater(candidate, fallbackLatest) {
				fallbackLatest = candidate
			}
			if runInfoActivityTime(candidate).After(referenceTime) {
				continue
			}
			if bestAtOrBeforeReference == nil || runInfoLater(candidate, bestAtOrBeforeReference) {
				bestAtOrBeforeReference = candidate
			}
		}
		if bestAtOrBeforeReference == nil {
			bestAtOrBeforeReference = fallbackLatest
		}
		if bestAtOrBeforeReference != nil {
			includeRunIDs[bestAtOrBeforeReference.RunID] = struct{}{}
		}
	}
}

func insertLatestRunInfo(top []*storage.RunInfo, run *storage.RunInfo, limit int) []*storage.RunInfo {
	if run == nil || limit <= 0 {
		return top
	}

	insertAt := len(top)
	for index, current := range top {
		if runInfoLater(run, current) {
			insertAt = index
			break
		}
	}
	if insertAt >= limit {
		return top
	}

	if len(top) < limit {
		top = append(top, nil)
	}
	for index := len(top) - 1; index > insertAt; index-- {
		top[index] = top[index-1]
	}
	top[insertAt] = run
	return top
}

func seedLatestParentAnchorRunInfos(
	runs []*storage.RunInfo,
	runByID map[string]*storage.RunInfo,
	includeRunIDs map[string]struct{},
) {
	if len(runs) == 0 {
		return
	}
	hasIncludedParentRun := make(map[string]bool, len(runs))
	anchorByTask := make(map[string]*storage.RunInfo, len(runs))
	for _, run := range runs {
		if run == nil || run.ParentRunID == "" {
			continue
		}
		parentRun, ok := runByID[run.ParentRunID]
		if !ok || parentRun == nil || parentRun.TaskID == run.TaskID {
			continue
		}
		if _, included := includeRunIDs[run.RunID]; included {
			hasIncludedParentRun[run.TaskID] = true
		}
		current, ok := anchorByTask[run.TaskID]
		if !ok {
			anchorByTask[run.TaskID] = run
			continue
		}
		if runInfoLater(run, current) {
			anchorByTask[run.TaskID] = run
		}
	}
	for taskID, run := range anchorByTask {
		if hasIncludedParentRun[taskID] {
			continue
		}
		if run != nil {
			includeRunIDs[run.RunID] = struct{}{}
		}
	}
}

func runInfoActivityTime(run *storage.RunInfo) time.Time {
	if run == nil {
		return time.Time{}
	}
	if !run.EndTime.IsZero() && run.EndTime.After(run.StartTime) {
		return run.EndTime
	}
	return run.StartTime
}

func runInfoLater(candidate, current *storage.RunInfo) bool {
	if candidate == nil {
		return false
	}
	if current == nil {
		return true
	}
	candidateTime := runInfoActivityTime(candidate)
	currentTime := runInfoActivityTime(current)
	if candidateTime.After(currentTime) {
		return true
	}
	if candidateTime.Before(currentTime) {
		return false
	}
	if candidate.StartTime.After(current.StartTime) {
		return true
	}
	if candidate.StartTime.Before(current.StartTime) {
		return false
	}
	return candidate.RunID > current.RunID
}

func sortRunInfosByStartTime(runs []*storage.RunInfo) {
	sort.SliceStable(runs, func(i, j int) bool {
		if runs[i] == nil {
			return false
		}
		if runs[j] == nil {
			return true
		}
		if runs[i].StartTime.Equal(runs[j].StartTime) {
			return runs[i].RunID < runs[j].RunID
		}
		return runs[i].StartTime.Before(runs[j].StartTime)
	})
}

func readRunInfoFiles(paths []string) ([]*storage.RunInfo, error) {
	if len(paths) == 0 {
		return nil, nil
	}

	// Small scans are faster sequentially due goroutine/setup overhead.
	if len(paths) < 32 {
		out := make([]*storage.RunInfo, 0, len(paths))
		for _, path := range paths {
			info, err := runstate.ReadRunInfo(path)
			if err != nil {
				return nil, errors.Wrapf(err, "read %s", path)
			}
			out = append(out, info)
		}
		return out, nil
	}

	workerCount := runtime.GOMAXPROCS(0)
	if workerCount < 1 {
		workerCount = 1
	}
	if workerCount > len(paths) {
		workerCount = len(paths)
	}

	results := make([]*storage.RunInfo, len(paths))
	indexCh := make(chan int, len(paths))
	var (
		wg      sync.WaitGroup
		errOnce sync.Once
		readErr error
	)

	worker := func() {
		defer wg.Done()
		for idx := range indexCh {
			info, err := runstate.ReadRunInfo(paths[idx])
			if err != nil {
				errOnce.Do(func() {
					readErr = errors.Wrapf(err, "read %s", paths[idx])
				})
				continue
			}
			results[idx] = info
		}
	}

	for i := 0; i < workerCount; i++ {
		wg.Add(1)
		go worker()
	}
	for idx := range paths {
		indexCh <- idx
	}
	close(indexCh)
	wg.Wait()

	if readErr != nil {
		return nil, readErr
	}

	out := make([]*storage.RunInfo, 0, len(results))
	for _, info := range results {
		if info != nil {
			out = append(out, info)
		}
	}
	return out, nil
}

// projectRunInfos collects RunInfo records scoped to a single project.
// This avoids scanning unrelated project trees for the high-frequency runs/flat endpoint.
func (s *Server) projectRunInfos(projectID string) ([]*storage.RunInfo, error) {
	if s.projectRunsCache == nil {
		return s.scanProjectRunInfos(projectID)
	}
	return s.projectRunsCache.get(projectID, func() ([]*storage.RunInfo, error) {
		return s.scanProjectRunInfos(projectID)
	})
}

// scanProjectRunInfos collects RunInfo records scoped to a single project.
// This method always scans storage and bypasses the runs/flat cache.
func (s *Server) scanProjectRunInfos(projectID string) ([]*storage.RunInfo, error) {
	runs := make([]*storage.RunInfo, 0, 128)

	if projectDir, ok := findProjectDir(s.rootDir, projectID); ok {
		runInfoPaths := make([]string, 0, 128)
		if err := filepath.WalkDir(projectDir, func(path string, d os.DirEntry, err error) error {
			if err != nil || d.IsDir() || d.Name() != "run-info.yaml" {
				return err
			}
			runInfoPaths = append(runInfoPaths, path)
			return nil
		}); err != nil && !os.IsNotExist(err) {
			return nil, errors.Wrap(err, "walk project root")
		}
		if parsedRuns, err := readRunInfoFiles(runInfoPaths); err != nil {
			return nil, err
		} else {
			for _, info := range parsedRuns {
				if info.ProjectID == projectID {
					runs = append(runs, info)
				}
			}
		}
	}

	// Extra roots use a flat runs/ layout; filter loaded run-info entries by project.
	for _, extra := range s.extraRoots {
		runsDir := filepath.Join(extra, "runs")
		entries, err := os.ReadDir(runsDir)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			obslog.Log(s.logger, "WARN", "api", "extra_root_read_failed",
				obslog.F("extra_root", runsDir),
				obslog.F("error", err),
			)
			continue
		}
		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}
			dir := filepath.Join(runsDir, entry.Name())
			if info, err := runstate.ReadRunInfo(filepath.Join(dir, "run-info.yaml")); err == nil {
				if info.ProjectID == projectID {
					runs = append(runs, info)
				}
				continue
			}
			if info, err := storage.ParseCwdTxt(filepath.Join(dir, "cwd.txt")); err == nil && info.ProjectID == projectID {
				runs = append(runs, info)
			}
		}
	}

	sortRunInfosByStartTime(runs)
	return runs, nil
}

// handleProjectsList serves GET and POST /api/projects.
func (s *Server) handleProjectsList(w http.ResponseWriter, r *http.Request) *apiError {
	switch r.Method {
	case http.MethodGet:
		return s.handleProjectsListGet(w, r)
	case http.MethodPost:
		return s.handleProjectCreate(w, r)
	default:
		return apiErrorMethodNotAllowed()
	}
}

func (s *Server) handleProjectsListGet(w http.ResponseWriter, _ *http.Request) *apiError {
	runs, err := s.allRunInfos()
	if err != nil {
		return apiErrorInternal("scan runs", err)
	}

	// group by project_id
	type projectData struct {
		lastActivity time.Time
		taskIDs      map[string]struct{}
		projectRoot  string
	}
	projects := make(map[string]*projectData)
	for _, run := range runs {
		pid := run.ProjectID
		if pid == "" {
			pid = "unknown"
		}
		p, ok := projects[pid]
		if !ok {
			p = &projectData{taskIDs: make(map[string]struct{})}
			projects[pid] = p
		}
		p.taskIDs[run.TaskID] = struct{}{}
		t := run.StartTime
		if !run.EndTime.IsZero() && run.EndTime.After(t) {
			t = run.EndTime
		}
		if t.After(p.lastActivity) {
			p.lastActivity = t
		}
		cwd := strings.TrimSpace(run.CWD)
		if p.projectRoot == "" && cwd != "" && !pathWithinRoot(cwd, s.rootDir) {
			p.projectRoot = cwd
		}
	}

	for projectID, marker := range s.collectProjectRootMarkers() {
		p, ok := projects[projectID]
		if !ok {
			p = &projectData{taskIDs: make(map[string]struct{})}
			projects[projectID] = p
		}
		if p.projectRoot == "" {
			p.projectRoot = marker.ProjectRoot
		}
		if marker.LastActivity.After(p.lastActivity) {
			p.lastActivity = marker.LastActivity
		}
	}

	result := make([]projectSummary, 0, len(projects))
	for id, p := range projects {
		result = append(result, projectSummary{
			ID:           id,
			LastActivity: p.lastActivity,
			TaskCount:    len(p.taskIDs),
			ProjectRoot:  p.projectRoot,
		})
	}
	sort.Slice(result, func(i, j int) bool {
		if result[i].LastActivity.Equal(result[j].LastActivity) {
			return result[i].ID < result[j].ID
		}
		return result[i].LastActivity.After(result[j].LastActivity)
	})
	return writeJSON(w, http.StatusOK, map[string]interface{}{"projects": result})
}

func (s *Server) handleProjectCreate(w http.ResponseWriter, r *http.Request) *apiError {
	var req projectCreateRequest
	if err := decodeJSON(r, &req); err != nil {
		return err
	}

	req.ProjectID = strings.TrimSpace(req.ProjectID)
	if err := validateIdentifier(req.ProjectID, "project_id"); err != nil {
		return err
	}

	projectRoot, apiErr := normalizeProjectRoot(req.ProjectRoot, s.rootDir)
	if apiErr != nil {
		return apiErr
	}
	req.ProjectRoot = projectRoot

	if _, exists := findProjectDir(s.rootDir, req.ProjectID); exists {
		return apiErrorConflict("project already exists", map[string]string{"project_id": req.ProjectID})
	}
	runs, err := s.allRunInfos()
	if err != nil {
		return apiErrorInternal("scan runs", err)
	}
	for _, run := range runs {
		if strings.TrimSpace(run.ProjectID) == req.ProjectID {
			return apiErrorConflict("project already exists", map[string]string{"project_id": req.ProjectID})
		}
	}

	projectDir := filepath.Join(s.rootDir, req.ProjectID)
	if err := os.MkdirAll(projectDir, 0o755); err != nil {
		return apiErrorInternal("create project directory", err)
	}

	markerPath := filepath.Join(projectDir, projectRootMarkerFile)
	if err := os.WriteFile(markerPath, []byte(req.ProjectRoot+"\n"), 0o644); err != nil {
		return apiErrorInternal("write project root marker", err)
	}
	markerInfo, err := os.Stat(markerPath)
	if err != nil {
		return apiErrorInternal("stat project root marker", err)
	}

	s.writeFormSubmissionAudit(r, formSubmissionAuditArgs{
		Endpoint:  "POST /api/projects",
		ProjectID: req.ProjectID,
		Payload:   req,
	})

	return writeJSON(w, http.StatusCreated, projectSummary{
		ID:           req.ProjectID,
		LastActivity: markerInfo.ModTime().UTC(),
		TaskCount:    0,
		ProjectRoot:  req.ProjectRoot,
	})
}

// handleProjectDetail serves GET /api/projects/{projectId}
func (s *Server) handleProjectDetail(w http.ResponseWriter, r *http.Request) *apiError {
	if r.Method != http.MethodGet {
		return apiErrorMethodNotAllowed()
	}
	projectID := strings.TrimPrefix(r.URL.Path, "/api/projects/")
	projectID = strings.TrimSuffix(projectID, "/")
	if projectID == "" {
		return apiErrorNotFound("project not found")
	}
	if err := validateIdentifier(projectID, "project_id"); err != nil {
		return err
	}

	runs, err := s.allRunInfos()
	if err != nil {
		return apiErrorInternal("scan runs", err)
	}

	var lastActivity time.Time
	taskIDs := make(map[string]struct{})
	projectRoot := ""
	for _, run := range runs {
		if run.ProjectID != projectID {
			continue
		}
		taskIDs[run.TaskID] = struct{}{}
		t := run.StartTime
		if !run.EndTime.IsZero() && run.EndTime.After(t) {
			t = run.EndTime
		}
		if t.After(lastActivity) {
			lastActivity = t
		}
		cwd := strings.TrimSpace(run.CWD)
		if projectRoot == "" && cwd != "" && !pathWithinRoot(cwd, s.rootDir) {
			projectRoot = cwd
		}
	}

	if marker, ok := s.collectProjectRootMarkers()[projectID]; ok {
		if projectRoot == "" {
			projectRoot = marker.ProjectRoot
		}
		if marker.LastActivity.After(lastActivity) {
			lastActivity = marker.LastActivity
		}
	}
	if len(taskIDs) == 0 && projectRoot == "" {
		return apiErrorNotFound("project not found")
	}

	resp := map[string]interface{}{
		"id":            projectID,
		"last_activity": lastActivity,
		"task_count":    len(taskIDs),
	}
	if projectRoot != "" {
		resp["project_root"] = projectRoot
	}
	return writeJSON(w, http.StatusOK, resp)
}

// handleProjectTasks serves GET /api/projects/{projectId}/tasks
func (s *Server) handleProjectTasks(w http.ResponseWriter, r *http.Request) *apiError {
	if r.Method != http.MethodGet {
		return apiErrorMethodNotAllowed()
	}
	// path: /api/projects/{projectId}/tasks
	parts := splitPath(r.URL.Path, "/api/projects/")
	if len(parts) < 2 || parts[1] != "tasks" {
		return apiErrorNotFound("not found")
	}
	projectID := parts[0]
	if err := validateIdentifier(projectID, "project_id"); err != nil {
		return err
	}

	runs, err := s.allRunInfos()
	if err != nil {
		return apiErrorInternal("scan runs", err)
	}

	tasks := buildTasksWithQueue(s.rootDir, projectID, runs, s.taskQueueSnapshot())
	// Sort by creation time, newest first.
	sort.Slice(tasks, func(i, j int) bool {
		return tasks[i].CreatedAt.After(tasks[j].CreatedAt)
	})

	// Apply optional status filter.
	if statusFilter := strings.ToLower(r.URL.Query().Get("status")); statusFilter != "" {
		tasks = filterTasksByStatus(tasks, statusFilter, s.rootDir, projectID)
	}

	limit, offset := parsePagination(r)
	total := len(tasks)

	// Determine the page slice.
	pageEnd := offset + limit
	if pageEnd > total {
		pageEnd = total
	}
	var pageTasks []projectTask
	if offset < total {
		pageTasks = tasks[offset:pageEnd]
	}

	result := make([]map[string]interface{}, 0, len(pageTasks))
	for _, t := range pageTasks {
		runCounts := make(map[string]int)
		for _, r := range t.Runs {
			runCounts[r.Status]++
		}
		item := map[string]interface{}{
			"id":            t.ID,
			"project_id":    t.ProjectID,
			"status":        t.Status,
			"last_activity": t.LastActivity,
			"run_count":     len(t.Runs),
			"run_counts":    runCounts,
			"depends_on":    t.DependsOn,
			"blocked_by":    t.BlockedBy,
			"thread_parent": t.ThreadParent,
		}
		if t.QueuePosition > 0 {
			item["queue_position"] = t.QueuePosition
		}
		result = append(result, item)
	}
	return writeJSON(w, http.StatusOK, paginatedResponse{
		Items:   result,
		Total:   total,
		Limit:   limit,
		Offset:  offset,
		HasMore: pageEnd < total,
	})
}

// handleProjectTask serves GET /api/projects/{projectId}/tasks/{taskId}
// and GET /api/projects/{projectId}/tasks/{taskId}/runs/{runId}
// and GET /api/projects/{projectId}/tasks/{taskId}/runs/{runId}/file
// and POST /api/projects/{projectId}/tasks/{taskId}/runs/{runId}/stop
func (s *Server) handleProjectTask(w http.ResponseWriter, r *http.Request) *apiError {
	// /api/projects/{projectId}/tasks/{taskId}[/runs/{runId}[/file|stream|stop]]
	parts := splitPath(r.URL.Path, "/api/projects/")
	// parts[0]=projectId, parts[1]="tasks", parts[2]=taskId, parts[3]="runs", parts[4]=runId, parts[5]="file|stream|stop"
	if len(parts) < 3 || parts[1] != "tasks" {
		return apiErrorNotFound("not found")
	}
	projectID := parts[0]
	taskID := parts[2]
	if err := validateIdentifier(projectID, "project_id"); err != nil {
		return err
	}
	if err := validateIdentifier(taskID, "task_id"); err != nil {
		return err
	}

	runs, err := s.allRunInfos()
	if err != nil {
		return apiErrorInternal("scan runs", err)
	}

	// task-scoped file endpoint: GET /api/projects/{p}/tasks/{t}/file?name=TASK.md
	if len(parts) == 4 && parts[3] == "file" {
		if r.Method != http.MethodGet {
			return apiErrorMethodNotAllowed()
		}
		return s.serveTaskFile(w, r, projectID, taskID)
	}

	// task-scoped message bus: /api/projects/{p}/tasks/{t}/messages[/stream]
	if len(parts) >= 4 && parts[3] == "messages" {
		if len(parts) == 4 {
			return s.handleTaskMessages(w, r, projectID, taskID)
		}
		if len(parts) == 5 && parts[4] == "stream" {
			return s.handleTaskMessagesStream(w, r, projectID, taskID)
		}
		return apiErrorNotFound("not found")
	}

	if len(parts) >= 4 && parts[3] == "runs" {
		// Paginated runs list: GET /api/projects/{p}/tasks/{t}/runs
		if len(parts) == 4 {
			if r.Method != http.MethodGet {
				return apiErrorMethodNotAllowed()
			}
			return s.handleProjectTaskRunsList(w, r, projectID, taskID, runs)
		}

		// Task-level log stream: GET /api/projects/{p}/tasks/{t}/runs/stream
		if len(parts) == 5 && parts[4] == "stream" {
			if r.Method != http.MethodGet {
				return apiErrorMethodNotAllowed()
			}
			return s.streamTaskRuns(w, r, projectID, taskID)
		}

		runID := parts[4]
		// find the specific run
		var found *storage.RunInfo
		for _, run := range runs {
			if run.ProjectID == projectID && run.TaskID == taskID && run.RunID == runID {
				found = run
				break
			}
		}
		if found == nil {
			return apiErrorNotFound("run not found")
		}
		// stop endpoint (POST)
		if len(parts) >= 6 && parts[5] == "stop" {
			if r.Method != http.MethodPost {
				return apiErrorMethodNotAllowed()
			}
			return s.handleStopRun(w, r, found)
		}
		// delete endpoint (DELETE) - run itself, no sub-path
		if len(parts) == 5 && r.Method == http.MethodDelete {
			if err := rejectUIDestructiveAction(r, "run deletion"); err != nil {
				return err
			}
			return s.handleRunDelete(w, r, projectID, taskID, found)
		}
		// remaining run endpoints are GET-only
		if r.Method != http.MethodGet {
			return apiErrorMethodNotAllowed()
		}
		// file endpoint
		if len(parts) >= 6 && parts[5] == "file" {
			return s.serveRunFile(w, r, found)
		}
		// stream endpoint
		if len(parts) >= 6 && parts[5] == "stream" {
			return s.serveRunFileStream(w, r, found)
		}
		// run info
		return writeJSON(w, http.StatusOK, runInfoToProjectRun(found))
	}

	// task resume endpoint (POST /api/projects/{p}/tasks/{t}/resume)
	if len(parts) == 4 && parts[3] == "resume" {
		if r.Method != http.MethodPost {
			return apiErrorMethodNotAllowed()
		}
		return s.handleTaskResume(w, r, projectID, taskID)
	}

	// task delete endpoint (DELETE)
	if r.Method == http.MethodDelete {
		if err := rejectUIDestructiveAction(r, "task deletion"); err != nil {
			return err
		}
		return s.handleTaskDelete(w, r, projectID, taskID, runs)
	}

	// task detail - GET only
	if r.Method != http.MethodGet {
		return apiErrorMethodNotAllowed()
	}
	tasks := buildTasksWithQueue(s.rootDir, projectID, runs, s.taskQueueSnapshot())
	for _, t := range tasks {
		if t.ID == taskID {
			return writeJSON(w, http.StatusOK, t)
		}
	}
	return apiErrorNotFound("task not found")
}

// handleStopRun handles POST /api/projects/{p}/tasks/{t}/runs/{r}/stop.
// It sends SIGTERM to the process group of a running run (best-effort).
func (s *Server) handleStopRun(w http.ResponseWriter, r *http.Request, run *storage.RunInfo) *apiError {
	if run.Status != storage.StatusRunning {
		return apiErrorConflict("run is not running", map[string]string{"status": run.Status})
	}
	if !storage.CanTerminateProcess(run) {
		return apiErrorConflict("run is externally owned and cannot be stopped by conductor", map[string]string{"run_id": run.RunID})
	}

	// Use PGID if available, otherwise fall back to PID.
	pgid := run.PGID
	if pgid <= 0 {
		pgid = run.PID
	}

	// Best-effort SIGTERM — log failures but return 202 regardless.
	if pgid > 0 {
		if err := runner.TerminateProcessGroup(pgid); err != nil {
			obslog.Log(s.logger, "ERROR", "api", "run_stop_signal_failed",
				obslog.F("request_id", requestIDFromRequest(r)),
				obslog.F("correlation_id", requestIDFromRequest(r)),
				obslog.F("project_id", run.ProjectID),
				obslog.F("task_id", run.TaskID),
				obslog.F("run_id", run.RunID),
				obslog.F("pgid", pgid),
				obslog.F("error", err),
			)
		}
	}
	s.writeFormSubmissionAudit(r, formSubmissionAuditArgs{
		Endpoint:  "POST /api/projects/{project_id}/tasks/{task_id}/runs/{run_id}/stop",
		ProjectID: run.ProjectID,
		TaskID:    run.TaskID,
		RunID:     run.RunID,
		Payload: map[string]any{
			"pgid": pgid,
		},
	})
	obslog.Log(s.logger, "WARN", "api", "run_stop_requested",
		obslog.F("request_id", requestIDFromRequest(r)),
		obslog.F("correlation_id", requestIDFromRequest(r)),
		obslog.F("project_id", run.ProjectID),
		obslog.F("task_id", run.TaskID),
		obslog.F("run_id", run.RunID),
		obslog.F("pgid", pgid),
	)

	return writeJSON(w, http.StatusAccepted, map[string]interface{}{
		"run_id":  run.RunID,
		"message": "SIGTERM sent",
	})
}

// handleRunDelete handles DELETE /api/projects/{p}/tasks/{t}/runs/{r}.
// It deletes a completed or failed run directory from disk.
// Returns 409 if the run is still running, 404 if the run directory is not found.
func (s *Server) handleRunDelete(w http.ResponseWriter, r *http.Request, projectID, taskID string, run *storage.RunInfo) *apiError {
	if err := validateIdentifier(projectID, "project_id"); err != nil {
		return err
	}
	if err := validateIdentifier(taskID, "task_id"); err != nil {
		return err
	}
	if run.Status == storage.StatusRunning {
		return apiErrorConflict("run is still running", map[string]string{"status": run.Status})
	}
	taskDir, ok := findProjectTaskDir(s.rootDir, projectID, taskID)
	if !ok {
		return apiErrorNotFound("task directory not found")
	}
	if err := requirePathWithinRoot(s.rootDir, taskDir, "task path"); err != nil {
		return err
	}
	runDir := filepath.Join(taskDir, "runs", run.RunID)
	if err := requirePathWithinRoot(s.rootDir, runDir, "run path"); err != nil {
		return err
	}
	if err := os.RemoveAll(runDir); err != nil {
		return apiErrorInternal("delete run directory", err)
	}
	s.writeFormSubmissionAudit(r, formSubmissionAuditArgs{
		Endpoint:  "DELETE /api/projects/{project_id}/tasks/{task_id}/runs/{run_id}",
		ProjectID: projectID,
		TaskID:    taskID,
		RunID:     run.RunID,
		Payload: map[string]any{
			"status": run.Status,
		},
	})
	obslog.Log(s.logger, "WARN", "api", "run_deleted",
		obslog.F("request_id", requestIDFromRequest(r)),
		obslog.F("correlation_id", requestIDFromRequest(r)),
		obslog.F("project_id", projectID),
		obslog.F("task_id", taskID),
		obslog.F("run_id", run.RunID),
		obslog.F("status", run.Status),
	)
	w.WriteHeader(http.StatusNoContent)
	return nil
}

// handleTaskResume handles POST /api/projects/{p}/tasks/{t}/resume.
// It removes the DONE file for an exhausted task so the Ralph loop can run again.
func (s *Server) handleTaskResume(w http.ResponseWriter, r *http.Request, projectID, taskID string) *apiError {
	if err := validateIdentifier(projectID, "project_id"); err != nil {
		return err
	}
	if err := validateIdentifier(taskID, "task_id"); err != nil {
		return err
	}
	taskDir, ok := findProjectTaskDir(s.rootDir, projectID, taskID)
	if !ok {
		return apiErrorNotFound("task not found")
	}
	if err := requirePathWithinRoot(s.rootDir, taskDir, "task path"); err != nil {
		return err
	}

	doneFile := filepath.Join(taskDir, "DONE")
	if err := os.Remove(doneFile); err != nil {
		if os.IsNotExist(err) {
			return apiErrorBadRequest("task has no DONE file; nothing to resume")
		}
		return apiErrorInternal("remove DONE file", err)
	}
	s.writeFormSubmissionAudit(r, formSubmissionAuditArgs{
		Endpoint:  "POST /api/projects/{project_id}/tasks/{task_id}/resume",
		ProjectID: projectID,
		TaskID:    taskID,
	})
	obslog.Log(s.logger, "INFO", "api", "task_resumed",
		obslog.F("request_id", requestIDFromRequest(r)),
		obslog.F("correlation_id", requestIDFromRequest(r)),
		obslog.F("project_id", projectID),
		obslog.F("task_id", taskID),
	)

	return writeJSON(w, http.StatusOK, map[string]interface{}{
		"project_id": projectID,
		"task_id":    taskID,
		"resumed":    true,
	})
}

// handleTaskDelete handles DELETE /api/projects/{p}/tasks/{t}.
// It deletes the task directory (and all its runs) from disk.
// Returns 409 if any run is still running, 404 if the task does not exist.
func (s *Server) handleTaskDelete(w http.ResponseWriter, r *http.Request, projectID, taskID string, runs []*storage.RunInfo) *apiError {
	if err := validateIdentifier(projectID, "project_id"); err != nil {
		return err
	}
	if err := validateIdentifier(taskID, "task_id"); err != nil {
		return err
	}
	// Check for any running runs belonging to this task.
	for _, run := range runs {
		if run.ProjectID == projectID && run.TaskID == taskID && run.Status == storage.StatusRunning {
			return apiErrorConflict("task has running runs", map[string]string{"status": "running"})
		}
	}

	taskDir, ok := findProjectTaskDir(s.rootDir, projectID, taskID)
	if !ok {
		return apiErrorNotFound("task not found")
	}
	if err := requirePathWithinRoot(s.rootDir, taskDir, "task path"); err != nil {
		return err
	}

	if err := os.RemoveAll(taskDir); err != nil {
		return apiErrorInternal("delete task directory", err)
	}
	if s.rootTaskPlanner != nil {
		launches, planErr := s.rootTaskPlanner.DropQueuedForTask(projectID, taskID)
		if planErr != nil {
			obslog.Log(s.logger, "ERROR", "api", "task_queue_drop_failed",
				obslog.F("project_id", projectID),
				obslog.F("task_id", taskID),
				obslog.F("error", planErr),
			)
		} else {
			s.launchPlannedTasks(launches)
		}
	}
	s.writeFormSubmissionAudit(r, formSubmissionAuditArgs{
		Endpoint:  "DELETE /api/projects/{project_id}/tasks/{task_id}",
		ProjectID: projectID,
		TaskID:    taskID,
	})
	obslog.Log(s.logger, "WARN", "api", "task_deleted",
		obslog.F("request_id", requestIDFromRequest(r)),
		obslog.F("correlation_id", requestIDFromRequest(r)),
		obslog.F("project_id", projectID),
		obslog.F("task_id", taskID),
	)

	w.WriteHeader(http.StatusNoContent)
	return nil
}

// handleProjectDelete handles DELETE /api/projects/{id}.
// With ?force=true it stops all running tasks first, then deletes the project.
// Without force, returns 409 if any task has running runs.
func (s *Server) handleProjectDelete(w http.ResponseWriter, r *http.Request, projectID string) *apiError {
	if err := validateIdentifier(projectID, "project_id"); err != nil {
		return err
	}
	force := r.URL.Query().Get("force") == "true"

	projectDir, ok := findProjectDir(s.rootDir, projectID)
	if !ok {
		return apiErrorNotFound("project not found")
	}
	if err := requirePathWithinRoot(s.rootDir, projectDir, "project path"); err != nil {
		return err
	}

	runs, err := s.allRunInfos()
	if err != nil {
		return apiErrorInternal("scan runs", err)
	}

	// Collect running runs for this project.
	var runningRuns []*storage.RunInfo
	for _, run := range runs {
		if run.ProjectID == projectID && run.Status == storage.StatusRunning {
			runningRuns = append(runningRuns, run)
		}
	}

	if len(runningRuns) > 0 && !force {
		return apiErrorConflict("project has running tasks; stop them first or use --force", nil)
	}

	// If force=true, send SIGTERM to all running runs (best-effort).
	if force {
		for _, run := range runningRuns {
			if !storage.CanTerminateProcess(run) {
				continue
			}
			pgid := run.PGID
			if pgid <= 0 {
				pgid = run.PID
			}
			if pgid > 0 {
				if err := runner.TerminateProcessGroup(pgid); err != nil {
					obslog.Log(s.logger, "ERROR", "api", "project_force_stop_signal_failed",
						obslog.F("request_id", requestIDFromRequest(r)),
						obslog.F("correlation_id", requestIDFromRequest(r)),
						obslog.F("project_id", projectID),
						obslog.F("task_id", run.TaskID),
						obslog.F("run_id", run.RunID),
						obslog.F("pgid", pgid),
						obslog.F("error", err),
					)
				}
			}
		}
	}

	// Count task directories and compute freed bytes before deletion.
	entries, err := os.ReadDir(projectDir)
	if err != nil {
		return apiErrorInternal("read project dir", err)
	}
	var deletedTasks int
	for _, entry := range entries {
		if entry.IsDir() && isTaskID(entry.Name()) {
			deletedTasks++
		}
	}

	freedBytes := gcDirSize(projectDir)

	if err := os.RemoveAll(projectDir); err != nil {
		return apiErrorInternal("delete project directory", err)
	}
	s.writeFormSubmissionAudit(r, formSubmissionAuditArgs{
		Endpoint:  "DELETE /api/projects/{project_id}",
		ProjectID: projectID,
		Payload: map[string]any{
			"force":         force,
			"deleted_tasks": deletedTasks,
			"freed_bytes":   freedBytes,
		},
	})
	obslog.Log(s.logger, "WARN", "api", "project_deleted",
		obslog.F("request_id", requestIDFromRequest(r)),
		obslog.F("correlation_id", requestIDFromRequest(r)),
		obslog.F("project_id", projectID),
		obslog.F("force", force),
		obslog.F("deleted_tasks", deletedTasks),
		obslog.F("freed_bytes", freedBytes),
	)

	return writeJSON(w, http.StatusOK, map[string]interface{}{
		"project_id":    projectID,
		"deleted_tasks": deletedTasks,
		"freed_bytes":   freedBytes,
	})
}

// handleProjectTaskRunsList serves GET /api/projects/{p}/tasks/{t}/runs (paginated run list).
func (s *Server) handleProjectTaskRunsList(w http.ResponseWriter, r *http.Request, projectID, taskID string, allRuns []*storage.RunInfo) *apiError {
	var taskRuns []projectRun
	for _, run := range allRuns {
		if run.ProjectID == projectID && run.TaskID == taskID {
			taskRuns = append(taskRuns, runInfoToProjectRun(run))
		}
	}
	if len(taskRuns) == 0 {
		return apiErrorNotFound("task not found")
	}
	// Sort by start time, newest first.
	sort.Slice(taskRuns, func(i, j int) bool {
		return taskRuns[i].StartTime.After(taskRuns[j].StartTime)
	})

	limit, offset := parsePagination(r)
	total := len(taskRuns)

	pageEnd := offset + limit
	if pageEnd > total {
		pageEnd = total
	}
	page := make([]projectRun, 0)
	if offset < total {
		page = taskRuns[offset:pageEnd]
	}
	return writeJSON(w, http.StatusOK, paginatedResponse{
		Items:   page,
		Total:   total,
		Limit:   limit,
		Offset:  offset,
		HasMore: pageEnd < total,
	})
}

// serveTaskFile serves a named file from a task directory (task-scoped, not run-scoped).
// Only TASK.md is allowed; returns 404 for any other name.
func (s *Server) serveTaskFile(w http.ResponseWriter, r *http.Request, projectID, taskID string) *apiError {
	if err := validateIdentifier(projectID, "project_id"); err != nil {
		return err
	}
	if err := validateIdentifier(taskID, "task_id"); err != nil {
		return err
	}
	name := r.URL.Query().Get("name")
	if name != "TASK.md" {
		return apiErrorNotFound("unknown task file: " + name)
	}
	taskDir, ok := findProjectTaskDir(s.rootDir, projectID, taskID)
	if !ok {
		return apiErrorNotFound("TASK.md not found")
	}
	if err := requirePathWithinRoot(s.rootDir, taskDir, "task path"); err != nil {
		return err
	}
	filePath := filepath.Join(taskDir, "TASK.md")
	if err := requirePathWithinRoot(s.rootDir, filePath, "task file path"); err != nil {
		return err
	}
	data, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return apiErrorNotFound("TASK.md not found")
		}
		return apiErrorInternal("read task file", err)
	}
	fi, _ := os.Stat(filePath)
	var modified time.Time
	if fi != nil {
		modified = fi.ModTime().UTC()
	}
	return writeJSON(w, http.StatusOK, map[string]interface{}{
		"name":       "TASK.md",
		"content":    string(data),
		"modified":   modified,
		"size_bytes": len(data),
	})
}

// serveRunFile reads a named file from a run directory.
func (s *Server) serveRunFile(w http.ResponseWriter, r *http.Request, run *storage.RunInfo) *apiError {
	name := r.URL.Query().Get("name")
	if name == "" {
		name = "stdout"
	}

	var filePath string
	switch name {
	case "stdout":
		filePath = run.StdoutPath
	case "stderr":
		filePath = run.StderrPath
	case "prompt":
		filePath = run.PromptPath
	case "output.md":
		if run.OutputPath != "" {
			filePath = run.OutputPath
		} else if run.StdoutPath != "" {
			filePath = filepath.Join(filepath.Dir(run.StdoutPath), "output.md")
		}
	default:
		return apiErrorNotFound("unknown file: " + name)
	}

	if filePath == "" {
		return apiErrorNotFound("file path not set for " + name)
	}
	if err := requirePathWithinRoot(s.rootDir, filePath, "run file path"); err != nil {
		return err
	}

	data, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return apiErrorNotFound("file not found: " + name)
		}
		return apiErrorInternal("read file", err)
	}

	fi, _ := os.Stat(filePath)
	var modified time.Time
	if fi != nil {
		modified = fi.ModTime().UTC()
	}
	return writeJSON(w, http.StatusOK, map[string]interface{}{
		"name":       name,
		"content":    string(data),
		"modified":   modified,
		"size_bytes": len(data),
	})
}

// serveRunFileStream streams a growing file using SSE (text/event-stream).
// It tails the file from the beginning, sending new content every 500ms.
func (s *Server) serveRunFileStream(w http.ResponseWriter, r *http.Request, run *storage.RunInfo) *apiError {
	name := r.URL.Query().Get("name")
	if name == "" {
		name = "stdout"
	}

	var filePath string
	switch name {
	case "stdout":
		filePath = run.StdoutPath
	case "stderr":
		filePath = run.StderrPath
	case "prompt":
		filePath = run.PromptPath
	case "output.md":
		if run.OutputPath != "" {
			filePath = run.OutputPath
		} else if run.StdoutPath != "" {
			filePath = filepath.Join(filepath.Dir(run.StdoutPath), "output.md")
		}
	default:
		return apiErrorNotFound("unknown file: " + name)
	}

	if filePath == "" {
		return apiErrorNotFound("file path not set for " + name)
	}
	if err := requirePathWithinRoot(s.rootDir, filePath, "run file path"); err != nil {
		return err
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming not supported", http.StatusInternalServerError)
		return nil
	}

	// Derive run-info.yaml path for live status re-checks.
	runInfoPath := ""
	if run.StdoutPath != "" {
		runInfoPath = filepath.Join(filepath.Dir(run.StdoutPath), "run-info.yaml")
	}

	var offset int64
	ticker := time.NewTicker(500 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-r.Context().Done():
			return nil
		case <-ticker.C:
			var fileSize int64
			f, err := os.Open(filePath)
			if err != nil {
				if os.IsNotExist(err) {
					continue // file not yet created
				}
				fmt.Fprintf(w, "event: error\ndata: %s\n\n", err.Error())
				flusher.Flush()
				return nil
			}

			fi, _ := f.Stat()
			if fi != nil {
				fileSize = fi.Size()
			}
			if fi != nil && fileSize > offset {
				if _, err := f.Seek(offset, io.SeekStart); err == nil {
					buf := make([]byte, fileSize-offset)
					n, _ := f.Read(buf)
					if n > 0 {
						chunk := string(buf[:n])
						offset += int64(n)
						for _, line := range strings.Split(chunk, "\n") {
							fmt.Fprintf(w, "data: %s\n", line)
						}
						fmt.Fprintf(w, "\n")
						flusher.Flush()
					}
				}
			}
			f.Close()

			// Re-read run-info.yaml to detect completion.
			currentStatus := run.Status
			if runInfoPath != "" {
				if current, err := runstate.ReadRunInfo(runInfoPath); err == nil {
					currentStatus = current.Status
				}
			}
			if currentStatus == storage.StatusCompleted || currentStatus == storage.StatusFailed {
				if fileSize <= offset {
					fmt.Fprintf(w, "event: done\ndata: run %s\n\n", currentStatus)
					flusher.Flush()
					return nil
				}
			}
		}
	}
}

// allRunInfos collects RunInfo from the primary root and all extra roots.
func (s *Server) allRunInfos() ([]*storage.RunInfo, error) {
	var all []*storage.RunInfo

	// scan primary root (run-info.yaml based)
	if err := filepath.WalkDir(s.rootDir, func(path string, d os.DirEntry, err error) error {
		if err != nil || d.IsDir() || d.Name() != "run-info.yaml" {
			return err
		}
		info, err := runstate.ReadRunInfo(path)
		if err != nil {
			return errors.Wrapf(err, "read %s", path)
		}
		all = append(all, info)
		return nil
	}); err != nil && !os.IsNotExist(err) {
		return nil, errors.Wrap(err, "walk primary root")
	}

	// scan extra roots (cwd.txt or run-info.yaml)
	for _, extra := range s.extraRoots {
		runsDir := filepath.Join(extra, "runs")
		entries, err := os.ReadDir(runsDir)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			obslog.Log(s.logger, "WARN", "api", "extra_root_read_failed",
				obslog.F("extra_root", runsDir),
				obslog.F("error", err),
			)
			continue
		}
		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}
			dir := filepath.Join(runsDir, entry.Name())
			if info, err := runstate.ReadRunInfo(filepath.Join(dir, "run-info.yaml")); err == nil {
				all = append(all, info)
				continue
			}
			if info, err := storage.ParseCwdTxt(filepath.Join(dir, "cwd.txt")); err == nil {
				all = append(all, info)
			}
		}
	}

	return all, nil
}

// buildTasks groups runs into tasks for a given project.
func buildTasks(rootDir, projectID string, runs []*storage.RunInfo) []projectTask {
	return buildTasksWithQueue(rootDir, projectID, runs, nil)
}

// buildTasksWithQueue groups runs into tasks and overlays queued planner state.
func buildTasksWithQueue(rootDir, projectID string, runs []*storage.RunInfo, queue map[taskQueueKey]taskQueueState) []projectTask {
	taskMap := make(map[string][]projectRun)
	for _, run := range runs {
		if run.ProjectID != projectID {
			continue
		}
		taskMap[run.TaskID] = append(taskMap[run.TaskID], runInfoToProjectRun(run))
	}

	if projectDir, ok := findProjectDir(rootDir, projectID); ok {
		entries, err := os.ReadDir(projectDir)
		if err == nil {
			for _, entry := range entries {
				if !entry.IsDir() {
					continue
				}
				taskID := entry.Name()
				taskDir := filepath.Join(projectDir, taskID)
				if !isTaskDirectory(taskDir) {
					continue
				}
				if _, exists := taskMap[taskID]; !exists {
					taskMap[taskID] = nil
				}
			}
		}
	}

	tasks := make([]projectTask, 0, len(taskMap))
	for taskID, taskRuns := range taskMap {
		sort.Slice(taskRuns, func(i, j int) bool {
			return taskRuns[i].StartTime.Before(taskRuns[j].StartTime)
		})

		taskDir, hasTaskDir := findProjectTaskDir(rootDir, projectID, taskID)
		var lastActivity time.Time
		status := "-"
		running := false
		for _, run := range taskRuns {
			t := run.StartTime
			if run.EndTime != nil && run.EndTime.After(t) {
				t = *run.EndTime
			}
			if t.After(lastActivity) {
				lastActivity = t
			}
			if run.Status == "running" {
				running = true
				status = "running"
			}
		}
		if len(taskRuns) > 0 && !running {
			status = taskRuns[len(taskRuns)-1].Status
		}

		createdAt := time.Time{}
		if len(taskRuns) > 0 {
			createdAt = taskRuns[0].StartTime
		}

		state := "idle"
		if running {
			state = "running"
		}

		done := false
		dependsOn := []string(nil)
		blockedBy := []string(nil)
		threadParent := (*ThreadParentReference)(nil)
		if hasTaskDir {
			if cfgDepends, err := taskdeps.ReadDependsOn(taskDir); err == nil {
				dependsOn = cfgDepends
			}
			if parentRef, err := readTaskThreadLink(taskDir); err == nil {
				threadParent = parentRef
			}
			if _, err := os.Stat(filepath.Join(taskDir, "DONE")); err == nil {
				done = true
			}
			if createdAt.IsZero() {
				if info, err := os.Stat(filepath.Join(taskDir, "TASK.md")); err == nil {
					createdAt = info.ModTime().UTC()
				}
			}
			if lastActivity.IsZero() {
				if info, err := os.Stat(filepath.Join(taskDir, "TASK.md")); err == nil {
					lastActivity = info.ModTime().UTC()
				}
			}
		}
		if done && len(taskRuns) == 0 {
			status = "done"
		}
		if len(taskRuns) == 0 && !done && len(dependsOn) > 0 {
			if unresolved, err := taskdeps.BlockedBy(rootDir, projectID, dependsOn); err == nil && len(unresolved) > 0 {
				blockedBy = unresolved
				status = "blocked"
				state = "blocked"
			}
		}
		queuePosition := 0
		if !done && !running {
			if queued, ok := queue[taskQueueKey{ProjectID: projectID, TaskID: taskID}]; ok && queued.Queued {
				status = "queued"
				state = "queued"
				queuePosition = queued.QueuePosition
			}
		}

		tasks = append(tasks, projectTask{
			ID:            taskID,
			ProjectID:     projectID,
			Status:        status,
			QueuePosition: queuePosition,
			LastActivity:  lastActivity,
			CreatedAt:     createdAt,
			Done:          done,
			State:         state,
			DependsOn:     dependsOn,
			BlockedBy:     blockedBy,
			ThreadParent:  threadParent,
			Runs:          taskRuns,
		})
	}
	return tasks
}

// filterTasksByStatus filters tasks by the given status string (case-insensitive).
// Supported values: "running"/"active", "done", "failed".
// Unknown values return all tasks unchanged (graceful degradation).
func filterTasksByStatus(tasks []projectTask, filter, rootDir, projectID string) []projectTask {
	switch filter {
	case "running", "active":
		var out []projectTask
		for _, t := range tasks {
			if t.Status == "running" {
				out = append(out, t)
			}
		}
		return out
	case "done":
		var out []projectTask
		for _, t := range tasks {
			taskDir, ok := findProjectTaskDir(rootDir, projectID, t.ID)
			if ok {
				if _, err := os.Stat(filepath.Join(taskDir, "DONE")); err == nil {
					out = append(out, t)
				}
			}
		}
		return out
	case "failed":
		var out []projectTask
		for _, t := range tasks {
			if t.Status == "failed" {
				out = append(out, t)
			}
		}
		return out
	case "blocked":
		var out []projectTask
		for _, t := range tasks {
			if t.Status == "blocked" {
				out = append(out, t)
			}
		}
		return out
	case "queued", "postponed":
		var out []projectTask
		for _, t := range tasks {
			if t.Status == "queued" {
				out = append(out, t)
			}
		}
		return out
	default:
		// Unknown status: graceful degradation — return all tasks.
		return tasks
	}
}

func isTaskDirectory(taskDir string) bool {
	if _, err := os.Stat(filepath.Join(taskDir, "TASK.md")); err == nil {
		return true
	}
	if _, err := os.Stat(filepath.Join(taskDir, taskdeps.ConfigFileName)); err == nil {
		return true
	}
	return dirExists(filepath.Join(taskDir, "runs"))
}

// buildRunFiles returns the list of files available for a run, checking existence on disk.
func buildRunFiles(info *storage.RunInfo) []RunFile {
	var files []RunFile
	if info.OutputPath != "" {
		if _, err := os.Stat(info.OutputPath); err == nil {
			files = append(files, RunFile{Name: "output.md", Label: "output"})
		}
	}
	if info.StdoutPath != "" {
		if _, err := os.Stat(info.StdoutPath); err == nil {
			files = append(files, RunFile{Name: "stdout", Label: "stdout"})
		}
	}
	if info.StderrPath != "" {
		if _, err := os.Stat(info.StderrPath); err == nil {
			files = append(files, RunFile{Name: "stderr", Label: "stderr"})
		}
	}
	if info.PromptPath != "" {
		if _, err := os.Stat(info.PromptPath); err == nil {
			files = append(files, RunFile{Name: "prompt", Label: "prompt"})
		}
	}
	return files
}

func runInfoToProjectRun(info *storage.RunInfo) projectRun {
	r := projectRun{
		ID:               info.RunID,
		Agent:            info.AgentType,
		AgentVersion:     info.AgentVersion,
		Status:           info.Status,
		ProcessOwnership: storage.EffectiveProcessOwnership(info),
		ExitCode:         info.ExitCode,
		StartTime:        info.StartTime,
		ParentRunID:      info.ParentRunID,
		PreviousRunID:    info.PreviousRunID,
		ErrorSummary:     info.ErrorSummary,
		Files:            buildRunFiles(info),
	}
	if !info.EndTime.IsZero() {
		t := info.EndTime
		r.EndTime = &t
	}
	return r
}

// streamTaskRuns fans in SSE log streams for all runs belonging to a project+task.
// It subscribes to existing runs and discovers new ones while the client is connected.
func (s *Server) streamTaskRuns(w http.ResponseWriter, r *http.Request, projectID, taskID string) *apiError {
	// Collect initial run IDs for this project+task; also validate that the task exists.
	allRuns, err := s.allRunInfos()
	if err != nil {
		return apiErrorInternal("list runs", err)
	}
	var initialRunIDs []string
	taskKnown := false
	for _, run := range allRuns {
		if run.ProjectID == projectID && run.TaskID == taskID {
			taskKnown = true
			initialRunIDs = append(initialRunIDs, run.RunID)
		}
	}
	if !taskKnown {
		return apiErrorNotFound("project or task not found")
	}

	writer, err := newSSEWriter(w)
	if err != nil {
		return apiErrorBadRequest("sse not supported")
	}
	manager, err := s.sseManager()
	if err != nil {
		return apiErrorInternal("init sse manager", err)
	}
	cfg := s.sseConfig()
	ctx := r.Context()
	fan := newFanIn(ctx)
	defer fan.Close()

	subsMu := &sync.Mutex{}
	subs := make(map[string]*Subscription)
	addSub := func(runID string, sub *Subscription) {
		subsMu.Lock()
		defer subsMu.Unlock()
		subs[runID] = sub
		fan.Add(sub)
	}
	closeSubs := func() {
		subsMu.Lock()
		defer subsMu.Unlock()
		for _, sub := range subs {
			sub.Close()
		}
	}
	defer closeSubs()

	for _, runID := range initialRunIDs {
		sub, subErr := manager.SubscribeRun(runID, Cursor{})
		if subErr != nil {
			continue
		}
		addSub(runID, sub)
	}

	discovery, discoveryErr := NewRunDiscovery(s.rootDir, cfg.DiscoveryInterval)
	if discoveryErr != nil {
		return apiErrorInternal("init discovery", discoveryErr)
	}
	discoveryCtx, cancelDiscovery := context.WithCancel(ctx)
	defer cancelDiscovery()
	go discovery.Poll(discoveryCtx, cfg.DiscoveryInterval)
	go func() {
		for {
			select {
			case <-discoveryCtx.Done():
				return
			case runID := <-discovery.NewRuns():
				subsMu.Lock()
				_, exists := subs[runID]
				subsMu.Unlock()
				if exists {
					continue
				}
				// Only subscribe if this run belongs to our project+task.
				latestRuns, latestErr := s.allRunInfos()
				if latestErr != nil {
					continue
				}
				match := false
				for _, run := range latestRuns {
					if run.RunID == runID && run.ProjectID == projectID && run.TaskID == taskID {
						match = true
						break
					}
				}
				if !match {
					continue
				}
				sub, subErr := manager.SubscribeRun(runID, Cursor{})
				if subErr != nil {
					continue
				}
				addSub(runID, sub)
			}
		}
	}()

	heartbeat := time.NewTicker(cfg.HeartbeatInterval)
	defer heartbeat.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-heartbeat.C:
			_ = writer.Send(SSEEvent{Event: "heartbeat", Data: "{}"})
		case ev, ok := <-fan.Events():
			if !ok {
				return nil
			}
			if err := writer.Send(ev); err != nil {
				return nil
			}
		}
	}
}

// gcResult is the JSON response for POST /api/projects/{id}/gc.
type gcResult struct {
	DeletedRuns int64 `json:"deleted_runs"`
	FreedBytes  int64 `json:"freed_bytes"`
	DryRun      bool  `json:"dry_run"`
}

// handleProjectGC handles POST /api/projects/{id}/gc.
// It garbage-collects old completed/failed runs for a project.
// Query params:
//   - older_than: duration string (default "168h")
//   - dry_run: "true"/"false" (default false)
//   - keep_failed: "true"/"false" (default false)
func (s *Server) handleProjectGC(w http.ResponseWriter, r *http.Request) *apiError {
	if r.Method != http.MethodPost {
		return apiErrorMethodNotAllowed()
	}
	if err := rejectUIDestructiveAction(r, "project garbage collection"); err != nil {
		return err
	}
	parts := splitPath(r.URL.Path, "/api/projects/")
	if len(parts) < 2 || parts[1] != "gc" {
		return apiErrorNotFound("not found")
	}
	projectID := parts[0]
	if err := validateIdentifier(projectID, "project_id"); err != nil {
		return err
	}

	q := r.URL.Query()
	olderThanStr := q.Get("older_than")
	if olderThanStr == "" {
		olderThanStr = "168h"
	}
	olderThan, err := time.ParseDuration(olderThanStr)
	if err != nil {
		return apiErrorBadRequest("invalid older_than: " + err.Error())
	}
	dryRun := q.Get("dry_run") == "true"
	keepFailed := q.Get("keep_failed") == "true"

	cutoff := time.Now().Add(-olderThan)

	_, ok := findProjectDir(s.rootDir, projectID)
	if !ok {
		return apiErrorNotFound("project not found")
	}

	runs, err := s.allRunInfos()
	if err != nil {
		return apiErrorInternal("scan runs", err)
	}

	// Cache task directory lookups to avoid repeated walks.
	taskDirCache := make(map[string]string)
	getTaskDir := func(taskID string) (string, bool) {
		key := projectID + "/" + taskID
		if d, cached := taskDirCache[key]; cached {
			return d, d != ""
		}
		d, found := findProjectTaskDir(s.rootDir, projectID, taskID)
		if !found {
			taskDirCache[key] = ""
		} else {
			taskDirCache[key] = d
		}
		return d, found
	}

	var deletedRuns int64
	var freedBytes int64
	for _, run := range runs {
		if run.ProjectID != projectID {
			continue
		}
		// Never delete running runs.
		if run.Status == storage.StatusRunning {
			continue
		}
		// Only delete completed or failed runs.
		if run.Status != storage.StatusCompleted && run.Status != storage.StatusFailed {
			continue
		}
		// Honour keep_failed.
		if keepFailed && run.Status == storage.StatusFailed {
			continue
		}
		// Check age.
		runTime := run.StartTime
		if runTime.IsZero() {
			runTime = run.EndTime
		}
		if runTime.IsZero() || !runTime.Before(cutoff) {
			continue
		}
		// Locate run directory.
		taskDir, ok := getTaskDir(run.TaskID)
		if !ok {
			continue
		}
		if err := requirePathWithinRoot(s.rootDir, taskDir, "task path"); err != nil {
			return err
		}
		runDir := filepath.Join(taskDir, "runs", run.RunID)
		if err := requirePathWithinRoot(s.rootDir, runDir, "run path"); err != nil {
			return err
		}
		if _, statErr := os.Stat(runDir); os.IsNotExist(statErr) {
			continue
		}
		size := gcDirSize(runDir)
		if !dryRun {
			if removeErr := os.RemoveAll(runDir); removeErr != nil {
				continue // best effort
			}
		}
		deletedRuns++
		freedBytes += size
	}
	s.writeFormSubmissionAudit(r, formSubmissionAuditArgs{
		Endpoint:  "POST /api/projects/{project_id}/gc",
		ProjectID: projectID,
		Payload: map[string]any{
			"older_than":   olderThan.String(),
			"dry_run":      dryRun,
			"keep_failed":  keepFailed,
			"deleted_runs": deletedRuns,
			"freed_bytes":  freedBytes,
		},
	})
	obslog.Log(s.logger, "WARN", "api", "project_gc_completed",
		obslog.F("request_id", requestIDFromRequest(r)),
		obslog.F("correlation_id", requestIDFromRequest(r)),
		obslog.F("project_id", projectID),
		obslog.F("older_than", olderThan),
		obslog.F("dry_run", dryRun),
		obslog.F("keep_failed", keepFailed),
		obslog.F("deleted_runs", deletedRuns),
		obslog.F("freed_bytes", freedBytes),
	)

	return writeJSON(w, http.StatusOK, gcResult{
		DeletedRuns: deletedRuns,
		FreedBytes:  freedBytes,
		DryRun:      dryRun,
	})
}

// gcDirSize returns the total size in bytes of all files under path.
func gcDirSize(path string) int64 {
	var size int64
	_ = filepath.Walk(path, func(_ string, info os.FileInfo, err error) error {
		if err != nil || info.IsDir() {
			return nil
		}
		size += info.Size()
		return nil
	})
	return size
}

// projectStats holds operational statistics for a project.
type projectStats struct {
	ProjectID            string `json:"project_id"`
	TotalTasks           int    `json:"total_tasks"`
	TotalRuns            int    `json:"total_runs"`
	RunningRuns          int    `json:"running_runs"`
	CompletedRuns        int    `json:"completed_runs"`
	FailedRuns           int    `json:"failed_runs"`
	CrashedRuns          int    `json:"crashed_runs"`
	MessageBusFiles      int    `json:"message_bus_files"`
	MessageBusTotalBytes int64  `json:"message_bus_total_bytes"`
}

// handleProjectStats serves GET /api/projects/{p}/stats.
// It walks the project directory and returns run/task/bus counts.
func (s *Server) handleProjectStats(w http.ResponseWriter, r *http.Request) *apiError {
	if r.Method != http.MethodGet {
		return apiErrorMethodNotAllowed()
	}
	parts := splitPath(r.URL.Path, "/api/projects/")
	if len(parts) < 2 || parts[1] != "stats" {
		return apiErrorNotFound("not found")
	}
	projectID := parts[0]
	if err := validateIdentifier(projectID, "project_id"); err != nil {
		return err
	}

	projectDir, ok := findProjectDir(s.rootDir, projectID)
	if !ok {
		return apiErrorNotFound("project not found")
	}
	if err := requirePathWithinRoot(s.rootDir, projectDir, "project path"); err != nil {
		return err
	}
	entries, err := os.ReadDir(projectDir)
	if err != nil {
		return apiErrorInternal("read project dir", err)
	}

	stats := projectStats{ProjectID: projectID}
	for _, entry := range entries {
		name := entry.Name()
		if !entry.IsDir() {
			// Count project-level message bus files (e.g. PROJECT-MESSAGE-BUS.md)
			if strings.HasSuffix(name, "MESSAGE-BUS.md") {
				stats.MessageBusFiles++
				if fi, infoErr := entry.Info(); infoErr == nil {
					stats.MessageBusTotalBytes += fi.Size()
				}
			}
			continue
		}

		// Subdirectory: check if it's a valid task ID
		if isTaskID(name) {
			stats.TotalTasks++
		}

		taskDir := filepath.Join(projectDir, name)

		// Count task-level message bus files (e.g. TASK-MESSAGE-BUS.md)
		taskEntries, readErr := os.ReadDir(taskDir)
		if readErr == nil {
			for _, te := range taskEntries {
				if !te.IsDir() && strings.HasSuffix(te.Name(), "MESSAGE-BUS.md") {
					stats.MessageBusFiles++
					if fi, infoErr := te.Info(); infoErr == nil {
						stats.MessageBusTotalBytes += fi.Size()
					}
				}
			}
		}

		// Count runs under <taskDir>/runs/
		runsDir := filepath.Join(taskDir, "runs")
		runEntries, err := os.ReadDir(runsDir)
		if err != nil {
			continue // no runs directory or unreadable
		}
		for _, runEntry := range runEntries {
			if !runEntry.IsDir() {
				continue
			}
			stats.TotalRuns++
			runInfoPath := filepath.Join(runsDir, runEntry.Name(), "run-info.yaml")
			runInfo, err := runstate.ReadRunInfo(runInfoPath)
			if err != nil {
				continue
			}
			switch runInfo.Status {
			case storage.StatusRunning:
				stats.RunningRuns++
			case storage.StatusCompleted:
				stats.CompletedRuns++
			case storage.StatusFailed:
				stats.FailedRuns++
			default:
				stats.CrashedRuns++
			}
		}
	}

	return writeJSON(w, http.StatusOK, stats)
}

// dirExists reports whether path exists and is a directory.
func dirExists(path string) bool {
	fi, err := os.Stat(path)
	return err == nil && fi.IsDir()
}

// findProjectDir locates the project directory for projectID under rootDir.
// It checks the most common layouts in order:
//  1. Direct: <rootDir>/<projectID>
//  2. Runs subdirectory: <rootDir>/runs/<projectID>
func findProjectDir(rootDir, projectID string) (string, bool) {
	cleanRoot := filepath.Clean(strings.TrimSpace(rootDir))
	if cleanRoot == "" || cleanRoot == "." {
		return "", false
	}
	candidates := []string{
		filepath.Join(cleanRoot, projectID),
		filepath.Join(cleanRoot, "runs", projectID),
	}
	for _, dir := range candidates {
		dir = filepath.Clean(dir)
		if !pathWithinRoot(dir, cleanRoot) {
			continue
		}
		if dirExists(dir) {
			return dir, true
		}
	}
	return "", false
}

// findProjectTaskDir locates the task directory for (projectID, taskID) under rootDir.
// It checks the most common layouts in order:
//  1. Direct: <rootDir>/<projectID>/<taskID>
//  2. Runs subdirectory: <rootDir>/runs/<projectID>/<taskID>
//  3. Walk: any directory matching <anything>/<projectID>/<taskID>, up to 3 levels deep
func findProjectTaskDir(rootDir, projectID, taskID string) (string, bool) {
	cleanRoot := filepath.Clean(strings.TrimSpace(rootDir))
	if cleanRoot == "" || cleanRoot == "." {
		return "", false
	}
	candidates := []string{
		filepath.Join(cleanRoot, projectID, taskID),
		filepath.Join(cleanRoot, "runs", projectID, taskID),
	}
	for _, dir := range candidates {
		dir = filepath.Clean(dir)
		if !pathWithinRoot(dir, cleanRoot) {
			continue
		}
		if dirExists(dir) {
			return dir, true
		}
	}
	// Walk rootDir looking for <anything>/<projectID>/<taskID>, pruning at depth >= 3.
	var found string
	_ = filepath.WalkDir(cleanRoot, func(path string, d os.DirEntry, err error) error {
		if err != nil || !d.IsDir() {
			return nil
		}
		if !pathWithinRoot(path, cleanRoot) {
			return filepath.SkipDir
		}
		if filepath.Base(path) == taskID && filepath.Base(filepath.Dir(path)) == projectID {
			candidate := filepath.Clean(path)
			if pathWithinRoot(candidate, cleanRoot) {
				found = candidate
			}
			return filepath.SkipAll
		}
		rel, relErr := filepath.Rel(cleanRoot, path)
		if relErr == nil && rel != "." {
			depth := strings.Count(filepath.ToSlash(rel), "/") + 1
			if depth >= 3 {
				return filepath.SkipDir
			}
		}
		return nil
	})
	return found, found != ""
}

// isTaskID reports whether name matches the task ID format: task-YYYYMMDD-HHMMSS-slug.
func isTaskID(name string) bool {
	if !strings.HasPrefix(name, "task-") {
		return false
	}
	rest := name[5:]
	if len(rest) < 17 { // 8 + 1 + 6 + 1 + at least 1 for slug
		return false
	}
	for i := 0; i < 8; i++ {
		if rest[i] < '0' || rest[i] > '9' {
			return false
		}
	}
	if rest[8] != '-' {
		return false
	}
	for i := 9; i < 15; i++ {
		if rest[i] < '0' || rest[i] > '9' {
			return false
		}
	}
	return rest[15] == '-' && len(rest) > 16
}

// splitPath splits a URL path after trimming the given prefix, returning path segments.
func splitPath(urlPath, prefix string) []string {
	trimmed := strings.TrimPrefix(urlPath, prefix)
	trimmed = strings.Trim(trimmed, "/")
	if trimmed == "" {
		return nil
	}
	return strings.Split(trimmed, "/")
}

// homeDirsResponse is the JSON response for GET /api/projects/home-dirs.
type homeDirsResponse struct {
	Dirs []string `json:"dirs"`
}

// handleProjectHomeDirs serves GET /api/projects/home-dirs.
// It returns a deduplicated list of recently used project home directories (CWD values)
// collected from run-info.yaml files across all tasks.
func (s *Server) handleProjectHomeDirs(w http.ResponseWriter, r *http.Request) *apiError {
	if r.Method != http.MethodGet {
		return apiErrorMethodNotAllowed()
	}

	runs, err := s.allRunInfos()
	if err != nil {
		return apiErrorInternal("scan runs", err)
	}

	seen := make(map[string]struct{})
	var dirs []string
	for _, run := range runs {
		cwd := strings.TrimSpace(run.CWD)
		if cwd == "" {
			continue
		}
		// Exclude paths that are under the conductor rootDir (task/run folders).
		if pathWithinRoot(cwd, s.rootDir) {
			continue
		}
		if _, dup := seen[cwd]; dup {
			continue
		}
		seen[cwd] = struct{}{}
		dirs = append(dirs, cwd)
		if len(dirs) >= 20 {
			break
		}
	}

	if len(dirs) < 20 {
		for _, marker := range s.collectProjectRootMarkers() {
			projectRoot := strings.TrimSpace(marker.ProjectRoot)
			if projectRoot == "" || pathWithinRoot(projectRoot, s.rootDir) {
				continue
			}
			if _, dup := seen[projectRoot]; dup {
				continue
			}
			seen[projectRoot] = struct{}{}
			dirs = append(dirs, projectRoot)
			if len(dirs) >= 20 {
				break
			}
		}
	}

	sort.Strings(dirs)
	if dirs == nil {
		dirs = []string{}
	}
	return writeJSON(w, http.StatusOK, homeDirsResponse{Dirs: dirs})
}

func (s *Server) collectProjectRootMarkers() map[string]projectRootMarker {
	result := make(map[string]projectRootMarker)
	for _, baseDir := range []string{s.rootDir, filepath.Join(s.rootDir, "runs")} {
		entries, err := os.ReadDir(baseDir)
		if err != nil {
			continue
		}
		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}
			projectID := entry.Name()
			projectDir := filepath.Join(baseDir, projectID)
			markerPath := filepath.Join(projectDir, projectRootMarkerFile)
			rootData, err := os.ReadFile(markerPath)
			if err != nil {
				continue
			}
			projectRoot := strings.TrimSpace(string(rootData))
			if projectRoot == "" {
				continue
			}
			markerInfo, err := os.Stat(markerPath)
			if err != nil {
				continue
			}
			marker := projectRootMarker{
				ProjectRoot:  projectRoot,
				LastActivity: markerInfo.ModTime().UTC(),
			}
			if existing, ok := result[projectID]; ok && existing.LastActivity.After(marker.LastActivity) {
				continue
			}
			result[projectID] = marker
		}
	}
	return result
}

func normalizeProjectRoot(projectRoot string, rootDir string) (string, *apiError) {
	projectRoot = strings.TrimSpace(projectRoot)
	if projectRoot == "" {
		return "", apiErrorBadRequest("project_root is required")
	}
	if strings.HasPrefix(projectRoot, "~/") {
		home, err := os.UserHomeDir()
		if err == nil {
			projectRoot = filepath.Join(home, projectRoot[2:])
		}
	}
	projectRoot = filepath.Clean(projectRoot)
	if !filepath.IsAbs(projectRoot) {
		return "", apiErrorBadRequest("project_root must be an absolute path or use ~/...")
	}
	if pathWithinRoot(projectRoot, rootDir) {
		return "", apiErrorBadRequest("project_root must be outside conductor storage root")
	}

	info, err := os.Stat(projectRoot)
	if err != nil {
		if os.IsNotExist(err) {
			return "", apiErrorBadRequest(fmt.Sprintf("project_root does not exist: %s", projectRoot))
		}
		return "", apiErrorInternal("stat project_root", err)
	}
	if !info.IsDir() {
		return "", apiErrorBadRequest(fmt.Sprintf("project_root is not a directory: %s", projectRoot))
	}
	return projectRoot, nil
}

func pathWithinRoot(path string, rootDir string) bool {
	cleanPath := filepath.Clean(strings.TrimSpace(path))
	cleanRoot := filepath.Clean(strings.TrimSpace(rootDir))
	if cleanPath == "" || cleanRoot == "" {
		return false
	}
	rel, err := filepath.Rel(cleanRoot, cleanPath)
	if err != nil {
		return false
	}
	if rel == "." {
		return true
	}
	return rel != ".." && !strings.HasPrefix(rel, ".."+string(os.PathSeparator))
}
