package api

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/jonnyzzz/conductor-loop/internal/runner"
	"github.com/jonnyzzz/conductor-loop/internal/storage"
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

// projectRun is the run summary shape the project API returns.
type projectRun struct {
	ID            string     `json:"id"`
	Agent         string     `json:"agent"`
	AgentVersion  string     `json:"agent_version,omitempty"`
	Status        string     `json:"status"`
	ExitCode      int        `json:"exit_code"`
	StartTime     time.Time  `json:"start_time"`
	EndTime       *time.Time `json:"end_time,omitempty"`
	ParentRunID   string     `json:"parent_run_id,omitempty"`
	PreviousRunID string     `json:"previous_run_id,omitempty"`
	ErrorSummary  string     `json:"error_summary,omitempty"`
}

// projectTask is the task shape the project API returns.
type projectTask struct {
	ID           string       `json:"id"`
	ProjectID    string       `json:"project_id"`
	Status       string       `json:"status"`
	LastActivity time.Time    `json:"last_activity"`
	CreatedAt    time.Time    `json:"created_at"`
	Done         bool         `json:"done"`
	State        string       `json:"state"`
	Runs         []projectRun `json:"runs"`
}

// projectSummary is the project list item shape.
type projectSummary struct {
	ID           string    `json:"id"`
	LastActivity time.Time `json:"last_activity"`
	TaskCount    int       `json:"task_count"`
}

// handleProjectsRouter dispatches /api/projects/{...} sub-paths.
func (s *Server) handleProjectsRouter(w http.ResponseWriter, r *http.Request) *apiError {
	parts := splitPath(r.URL.Path, "/api/projects/")
	if len(parts) == 0 {
		return apiErrorNotFound("not found")
	}
	// /api/projects/{id}
	if len(parts) == 1 {
		return s.handleProjectDetail(w, r)
	}
	// /api/projects/{id}/stats
	if parts[1] == "stats" {
		return s.handleProjectStats(w, r)
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
	return apiErrorNotFound("not found")
}

// handleProjectsList serves GET /api/projects
func (s *Server) handleProjectsList(w http.ResponseWriter, r *http.Request) *apiError {
	if r.Method != http.MethodGet {
		return apiErrorMethodNotAllowed()
	}
	runs, err := s.allRunInfos()
	if err != nil {
		return apiErrorInternal("scan runs", err)
	}

	// group by project_id
	type projectData struct {
		lastActivity time.Time
		taskIDs      map[string]struct{}
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
	}

	result := make([]projectSummary, 0, len(projects))
	for id, p := range projects {
		result = append(result, projectSummary{
			ID:           id,
			LastActivity: p.lastActivity,
			TaskCount:    len(p.taskIDs),
		})
	}
	sort.Slice(result, func(i, j int) bool {
		return result[i].LastActivity.After(result[j].LastActivity)
	})
	return writeJSON(w, http.StatusOK, map[string]interface{}{"projects": result})
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

	runs, err := s.allRunInfos()
	if err != nil {
		return apiErrorInternal("scan runs", err)
	}

	var lastActivity time.Time
	taskIDs := make(map[string]struct{})
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
	}
	if len(taskIDs) == 0 {
		return apiErrorNotFound("project not found")
	}
	return writeJSON(w, http.StatusOK, map[string]interface{}{
		"id":            projectID,
		"last_activity": lastActivity,
		"task_count":    len(taskIDs),
	})
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

	runs, err := s.allRunInfos()
	if err != nil {
		return apiErrorInternal("scan runs", err)
	}

	tasks := buildTasks(projectID, runs)
	// Sort by creation time, newest first.
	sort.Slice(tasks, func(i, j int) bool {
		return tasks[i].CreatedAt.After(tasks[j].CreatedAt)
	})

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
		result = append(result, map[string]interface{}{
			"id":            t.ID,
			"project_id":    t.ProjectID,
			"status":        t.Status,
			"last_activity": t.LastActivity,
			"run_count":     len(t.Runs),
			"run_counts":    runCounts,
		})
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
			return s.handleStopRun(w, found)
		}
		// delete endpoint (DELETE) - run itself, no sub-path
		if len(parts) == 5 && r.Method == http.MethodDelete {
			return s.handleRunDelete(w, projectID, taskID, found)
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

	// task delete endpoint (DELETE)
	if r.Method == http.MethodDelete {
		return s.handleTaskDelete(w, projectID, taskID, runs)
	}

	// task detail - GET only
	if r.Method != http.MethodGet {
		return apiErrorMethodNotAllowed()
	}
	tasks := buildTasks(projectID, runs)
	for _, t := range tasks {
		if t.ID == taskID {
			return writeJSON(w, http.StatusOK, t)
		}
	}
	return apiErrorNotFound("task not found")
}

// handleStopRun handles POST /api/projects/{p}/tasks/{t}/runs/{r}/stop.
// It sends SIGTERM to the process group of a running run (best-effort).
func (s *Server) handleStopRun(w http.ResponseWriter, run *storage.RunInfo) *apiError {
	if run.Status != storage.StatusRunning {
		return apiErrorConflict("run is not running", map[string]string{"status": run.Status})
	}

	// Use PGID if available, otherwise fall back to PID.
	pgid := run.PGID
	if pgid <= 0 {
		pgid = run.PID
	}

	// Best-effort SIGTERM — log failures but return 202 regardless.
	if pgid > 0 {
		if err := runner.TerminateProcessGroup(pgid); err != nil && s.logger != nil {
			s.logger.Printf("stop run %s: terminate pgid %d: %v", run.RunID, pgid, err)
		}
	}

	return writeJSON(w, http.StatusAccepted, map[string]interface{}{
		"run_id":  run.RunID,
		"message": "SIGTERM sent",
	})
}

// handleRunDelete handles DELETE /api/projects/{p}/tasks/{t}/runs/{r}.
// It deletes a completed or failed run directory from disk.
// Returns 409 if the run is still running, 404 if the run directory is not found.
func (s *Server) handleRunDelete(w http.ResponseWriter, projectID, taskID string, run *storage.RunInfo) *apiError {
	if run.Status == storage.StatusRunning {
		return apiErrorConflict("run is still running", map[string]string{"status": run.Status})
	}
	taskDir, ok := findProjectTaskDir(s.rootDir, projectID, taskID)
	if !ok {
		return apiErrorNotFound("task directory not found")
	}
	runDir := filepath.Join(taskDir, "runs", run.RunID)
	if err := os.RemoveAll(runDir); err != nil {
		return apiErrorInternal("delete run directory", err)
	}
	w.WriteHeader(http.StatusNoContent)
	return nil
}

// handleTaskDelete handles DELETE /api/projects/{p}/tasks/{t}.
// It deletes the task directory (and all its runs) from disk.
// Returns 409 if any run is still running, 404 if the task does not exist.
func (s *Server) handleTaskDelete(w http.ResponseWriter, projectID, taskID string, runs []*storage.RunInfo) *apiError {
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

	if err := os.RemoveAll(taskDir); err != nil {
		return apiErrorInternal("delete task directory", err)
	}

	w.WriteHeader(http.StatusNoContent)
	return nil
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
	name := r.URL.Query().Get("name")
	if name != "TASK.md" {
		return apiErrorNotFound("unknown task file: " + name)
	}
	taskDir, ok := findProjectTaskDir(s.rootDir, projectID, taskID)
	if !ok {
		return apiErrorNotFound("TASK.md not found")
	}
	filePath := filepath.Join(taskDir, "TASK.md")
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
		// try output.md next to stdout
		if run.StdoutPath != "" {
			filePath = filepath.Join(filepath.Dir(run.StdoutPath), "output.md")
		}
	default:
		return apiErrorNotFound("unknown file: " + name)
	}

	if filePath == "" {
		return apiErrorNotFound("file path not set for " + name)
	}

	var fallbackName string
	actualPath := filePath
	data, err := os.ReadFile(filePath)
	if err != nil && os.IsNotExist(err) && name == "output.md" {
		// Try agent-stdout.txt as fallback
		if run.StdoutPath != "" {
			if fb, ferr := os.ReadFile(run.StdoutPath); ferr == nil {
				data = fb
				err = nil
				fallbackName = "agent-stdout.txt"
				actualPath = run.StdoutPath
			}
		}
	}
	if err != nil {
		if os.IsNotExist(err) {
			return apiErrorNotFound("file not found: " + name)
		}
		return apiErrorInternal("read file", err)
	}

	fi, _ := os.Stat(actualPath)
	var modified time.Time
	if fi != nil {
		modified = fi.ModTime().UTC()
	}
	resp := map[string]interface{}{
		"name":       name,
		"content":    string(data),
		"modified":   modified,
		"size_bytes": len(data),
	}
	if fallbackName != "" {
		resp["fallback"] = fallbackName
	}
	return writeJSON(w, http.StatusOK, resp)
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
		if run.StdoutPath != "" {
			filePath = filepath.Join(filepath.Dir(run.StdoutPath), "output.md")
		}
	default:
		return apiErrorNotFound("unknown file: " + name)
	}

	if filePath == "" {
		return apiErrorNotFound("file path not set for " + name)
	}

	// Fallback: if output.md doesn't exist, stream agent-stdout.txt instead.
	if name == "output.md" {
		if _, err := os.Stat(filePath); os.IsNotExist(err) && run.StdoutPath != "" {
			if _, err2 := os.Stat(run.StdoutPath); err2 == nil {
				filePath = run.StdoutPath
			}
		}
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
				if current, err := storage.ReadRunInfo(runInfoPath); err == nil {
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
		info, err := storage.ReadRunInfo(path)
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
			s.logger.Printf("warn: read extra root %s: %v", runsDir, err)
			continue
		}
		for _, entry := range entries {
			if !entry.IsDir() {
				continue
			}
			dir := filepath.Join(runsDir, entry.Name())
			if info, err := storage.ReadRunInfo(filepath.Join(dir, "run-info.yaml")); err == nil {
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
func buildTasks(projectID string, runs []*storage.RunInfo) []projectTask {
	taskMap := make(map[string][]projectRun)
	for _, run := range runs {
		if run.ProjectID != projectID {
			continue
		}
		taskMap[run.TaskID] = append(taskMap[run.TaskID], runInfoToProjectRun(run))
	}

	tasks := make([]projectTask, 0, len(taskMap))
	for taskID, taskRuns := range taskMap {
		sort.Slice(taskRuns, func(i, j int) bool {
			return taskRuns[i].StartTime.Before(taskRuns[j].StartTime)
		})

		var lastActivity time.Time
		status := "completed"
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
		if !running && len(taskRuns) > 0 {
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

		tasks = append(tasks, projectTask{
			ID:           taskID,
			ProjectID:    projectID,
			Status:       status,
			LastActivity: lastActivity,
			CreatedAt:    createdAt,
			Done:         !running,
			State:        state,
			Runs:         taskRuns,
		})
	}
	return tasks
}

func runInfoToProjectRun(info *storage.RunInfo) projectRun {
	r := projectRun{
		ID:            info.RunID,
		Agent:         info.AgentType,
		AgentVersion:  info.AgentVersion,
		Status:        info.Status,
		ExitCode:      info.ExitCode,
		StartTime:     info.StartTime,
		ParentRunID:   info.ParentRunID,
		PreviousRunID: info.PreviousRunID,
		ErrorSummary:  info.ErrorSummary,
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

	projectDir, ok := findProjectDir(s.rootDir, projectID)
	if !ok {
		return apiErrorNotFound("project not found")
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
			runInfo, err := storage.ReadRunInfo(runInfoPath)
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
	if dir := filepath.Join(rootDir, projectID); dirExists(dir) {
		return dir, true
	}
	if dir := filepath.Join(rootDir, "runs", projectID); dirExists(dir) {
		return dir, true
	}
	return "", false
}

// findProjectTaskDir locates the task directory for (projectID, taskID) under rootDir.
// It checks the most common layouts in order:
//  1. Direct: <rootDir>/<projectID>/<taskID>
//  2. Runs subdirectory: <rootDir>/runs/<projectID>/<taskID>
//  3. Walk: any directory matching <anything>/<projectID>/<taskID>, up to 3 levels deep
func findProjectTaskDir(rootDir, projectID, taskID string) (string, bool) {
	if dir := filepath.Join(rootDir, projectID, taskID); dirExists(dir) {
		return dir, true
	}
	if dir := filepath.Join(rootDir, "runs", projectID, taskID); dirExists(dir) {
		return dir, true
	}
	// Walk rootDir looking for <anything>/<projectID>/<taskID>, pruning at depth >= 3.
	var found string
	_ = filepath.WalkDir(rootDir, func(path string, d os.DirEntry, err error) error {
		if err != nil || !d.IsDir() {
			return nil
		}
		if filepath.Base(path) == taskID && filepath.Base(filepath.Dir(path)) == projectID {
			found = path
			return filepath.SkipAll
		}
		rel, relErr := filepath.Rel(rootDir, path)
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
