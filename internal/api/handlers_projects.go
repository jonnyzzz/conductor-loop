package api

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/jonnyzzz/conductor-loop/internal/storage"
	"github.com/pkg/errors"
)

// projectRun is the run summary shape the project API returns.
type projectRun struct {
	ID            string     `json:"id"`
	Agent         string     `json:"agent"`
	Status        string     `json:"status"`
	ExitCode      int        `json:"exit_code"`
	StartTime     time.Time  `json:"start_time"`
	EndTime       *time.Time `json:"end_time,omitempty"`
	ParentRunID   string     `json:"parent_run_id,omitempty"`
	PreviousRunID string     `json:"previous_run_id,omitempty"`
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
	// /api/projects/{id}/tasks[/...]
	if parts[1] == "tasks" {
		if len(parts) == 2 {
			return s.handleProjectTasks(w, r)
		}
		return s.handleProjectTask(w, r)
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
	result := make([]map[string]interface{}, 0, len(tasks))
	for _, t := range tasks {
		result = append(result, map[string]interface{}{
			"id":            t.ID,
			"project_id":    t.ProjectID,
			"status":        t.Status,
			"last_activity": t.LastActivity,
			"run_count":     len(t.Runs),
		})
	}
	sort.Slice(result, func(i, j int) bool {
		ai, _ := result[i]["last_activity"].(time.Time)
		aj, _ := result[j]["last_activity"].(time.Time)
		return ai.After(aj)
	})
	return writeJSON(w, http.StatusOK, map[string]interface{}{"tasks": result})
}

// handleProjectTask serves GET /api/projects/{projectId}/tasks/{taskId}
// and GET /api/projects/{projectId}/tasks/{taskId}/runs/{runId}
// and GET /api/projects/{projectId}/tasks/{taskId}/runs/{runId}/file
func (s *Server) handleProjectTask(w http.ResponseWriter, r *http.Request) *apiError {
	if r.Method != http.MethodGet {
		return apiErrorMethodNotAllowed()
	}
	// /api/projects/{projectId}/tasks/{taskId}[/runs/{runId}[/file]]
	parts := splitPath(r.URL.Path, "/api/projects/")
	// parts[0]=projectId, parts[1]="tasks", parts[2]=taskId, parts[3]="runs", parts[4]=runId, parts[5]="file"
	if len(parts) < 3 || parts[1] != "tasks" {
		return apiErrorNotFound("not found")
	}
	projectID := parts[0]
	taskID := parts[2]

	runs, err := s.allRunInfos()
	if err != nil {
		return apiErrorInternal("scan runs", err)
	}

	if len(parts) >= 5 && parts[3] == "runs" {
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

	// task detail
	tasks := buildTasks(projectID, runs)
	for _, t := range tasks {
		if t.ID == taskID {
			return writeJSON(w, http.StatusOK, t)
		}
	}
	return apiErrorNotFound("task not found")
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
		if run.StdoutPath != "" {
			filePath = filepath.Join(filepath.Dir(run.StdoutPath), "output.md")
		}
	default:
		return apiErrorNotFound("unknown file: " + name)
	}

	if filePath == "" {
		return apiErrorNotFound("file path not set for " + name)
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
		Status:        info.Status,
		ExitCode:      info.ExitCode,
		StartTime:     info.StartTime,
		ParentRunID:   info.ParentRunID,
		PreviousRunID: info.PreviousRunID,
	}
	if !info.EndTime.IsZero() {
		t := info.EndTime
		r.EndTime = &t
	}
	return r
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
