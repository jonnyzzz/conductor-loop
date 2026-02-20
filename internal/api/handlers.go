package api

import (
	"encoding/json"
	stderrors "errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/jonnyzzz/conductor-loop/internal/messagebus"
	"github.com/jonnyzzz/conductor-loop/internal/runner"
	"github.com/jonnyzzz/conductor-loop/internal/storage"
	"github.com/pkg/errors"
)

const maxJSONBodySize = 1 << 20

// TaskCreateRequest defines the payload for task creation.
type TaskCreateRequest struct {
	ProjectID string            `json:"project_id"`
	TaskID    string            `json:"task_id"`
	AgentType string            `json:"agent_type"`
	Prompt    string            `json:"prompt"`
	Config    map[string]string `json:"config,omitempty"`
}

// TaskCreateResponse defines the response for task creation.
type TaskCreateResponse struct {
	ProjectID string `json:"project_id"`
	TaskID    string `json:"task_id"`
	Status    string `json:"status"`
}

// TaskResponse defines the task response payload.
type TaskResponse struct {
	ProjectID    string        `json:"project_id"`
	TaskID       string        `json:"task_id"`
	Status       string        `json:"status"`
	LastActivity time.Time     `json:"last_activity"`
	Runs         []RunResponse `json:"runs,omitempty"`
}

// RunResponse defines run metadata returned by the API.
type RunResponse struct {
	RunID     string    `json:"run_id"`
	ProjectID string    `json:"project_id"`
	TaskID    string    `json:"task_id"`
	Status    string    `json:"status"`
	StartTime time.Time `json:"start_time"`
	EndTime   time.Time `json:"end_time,omitempty"`
	ExitCode  int       `json:"exit_code,omitempty"`
}

// MessageResponse defines the message bus entry payload.
type MessageResponse struct {
	MsgID        string    `json:"msg_id"`
	Timestamp    time.Time `json:"timestamp"`
	Type         string    `json:"type"`
	ProjectID    string    `json:"project_id"`
	TaskID       string    `json:"task_id,omitempty"`
	RunID        string    `json:"run_id,omitempty"`
	ParentMsgIDs []string  `json:"parents,omitempty"`
	Attachment   string    `json:"attachment_path,omitempty"`
	Body         string    `json:"body"`
}

type handlerFunc func(http.ResponseWriter, *http.Request) *apiError

func (s *Server) wrap(handler handlerFunc) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if handler == nil {
			s.writeError(w, apiErrorInternal("handler is nil", nil))
			return
		}
		if err := handler(w, r); err != nil {
			s.writeError(w, err)
		}
	})
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) *apiError {
	if r.Method != http.MethodGet {
		return apiErrorMethodNotAllowed()
	}
	return writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) handleVersion(w http.ResponseWriter, r *http.Request) *apiError {
	if r.Method != http.MethodGet {
		return apiErrorMethodNotAllowed()
	}
	return writeJSON(w, http.StatusOK, map[string]string{"version": s.version})
}

// StatusResponse defines the payload for the /api/v1/status endpoint.
type StatusResponse struct {
	ActiveRunsCount  int      `json:"active_runs_count"`
	UptimeSeconds    float64  `json:"uptime_seconds"`
	ConfiguredAgents []string `json:"configured_agents"`
	Version          string   `json:"version"`
}

func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request) *apiError {
	if r.Method != http.MethodGet {
		return apiErrorMethodNotAllowed()
	}

	runs, err := listRunResponses(s.rootDir)
	if err != nil {
		return apiErrorInternal("list runs", err)
	}
	for _, extra := range s.extraRoots {
		extraRuns, err := listRunResponsesFlat(extra)
		if err != nil {
			s.logger.Printf("warn: scan extra root %s: %v", extra, err)
			continue
		}
		runs = append(runs, extraRuns...)
	}

	activeCount := 0
	for _, run := range runs {
		if run.EndTime.IsZero() {
			activeCount++
		}
	}

	agents := s.agentNames
	if agents == nil {
		agents = []string{}
	}

	uptime := s.now().Sub(s.startTime).Seconds()

	resp := StatusResponse{
		ActiveRunsCount:  activeCount,
		UptimeSeconds:    uptime,
		ConfiguredAgents: agents,
		Version:          s.version,
	}
	return writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handleTasks(w http.ResponseWriter, r *http.Request) *apiError {
	switch r.Method {
	case http.MethodGet:
		return s.handleTaskList(w, r)
	case http.MethodPost:
		return s.handleTaskCreate(w, r)
	default:
		return apiErrorMethodNotAllowed()
	}
}

func (s *Server) handleTaskByID(w http.ResponseWriter, r *http.Request) *apiError {
	segments := pathSegments(r.URL.Path, "/api/v1/tasks/")
	if len(segments) != 1 {
		return apiErrorNotFound("task not found")
	}
	taskID := segments[0]
	projectID := strings.TrimSpace(r.URL.Query().Get("project_id"))

	switch r.Method {
	case http.MethodGet:
		return s.handleTaskGet(w, r, projectID, taskID)
	case http.MethodDelete:
		return s.handleTaskCancel(w, r, projectID, taskID)
	default:
		return apiErrorMethodNotAllowed()
	}
}

func (s *Server) handleRuns(w http.ResponseWriter, r *http.Request) *apiError {
	if r.Method != http.MethodGet {
		return apiErrorMethodNotAllowed()
	}
	runs, err := listRunResponses(s.rootDir)
	if err != nil {
		return apiErrorInternal("list runs", err)
	}
	for _, extra := range s.extraRoots {
		extraRuns, err := listRunResponsesFlat(extra)
		if err != nil {
			s.logger.Printf("warn: scan extra root %s: %v", extra, err)
			continue
		}
		runs = append(runs, extraRuns...)
	}
	sort.Slice(runs, func(i, j int) bool {
		return runs[i].RunID < runs[j].RunID
	})
	return writeJSON(w, http.StatusOK, map[string][]RunResponse{"runs": runs})
}

func (s *Server) handleRunByID(w http.ResponseWriter, r *http.Request) *apiError {
	segments := pathSegments(r.URL.Path, "/api/v1/runs/")
	if len(segments) == 0 {
		return apiErrorNotFound("run not found")
	}
	runID := segments[0]
	if len(segments) == 1 {
		if r.Method != http.MethodGet {
			return apiErrorMethodNotAllowed()
		}
		return s.handleRunGet(w, r, runID)
	}
	if len(segments) == 2 {
		switch segments[1] {
		case "info":
			if r.Method != http.MethodGet {
				return apiErrorMethodNotAllowed()
			}
			return s.handleRunInfo(w, r, runID)
		case "stop":
			if r.Method != http.MethodPost {
				return apiErrorMethodNotAllowed()
			}
			return s.handleRunStop(w, r, runID)
		case "stream":
			if r.Method != http.MethodGet {
				return apiErrorMethodNotAllowed()
			}
			return s.streamRun(w, r, runID)
		default:
			return apiErrorNotFound("run not found")
		}
	}
	return apiErrorNotFound("run not found")
}

func (s *Server) handleMessages(w http.ResponseWriter, r *http.Request) *apiError {
	if r.Method != http.MethodGet {
		return apiErrorMethodNotAllowed()
	}
	projectID := strings.TrimSpace(r.URL.Query().Get("project_id"))
	if projectID == "" {
		return apiErrorBadRequest("project_id is required")
	}
	after := strings.TrimSpace(r.URL.Query().Get("after"))
	taskID := strings.TrimSpace(r.URL.Query().Get("task_id"))

	busPath := filepath.Join(s.rootDir, projectID, "PROJECT-MESSAGE-BUS.md")
	if taskID != "" {
		busPath = filepath.Join(s.rootDir, projectID, taskID, "TASK-MESSAGE-BUS.md")
	}
	bus, err := messagebus.NewMessageBus(busPath)
	if err != nil {
		return apiErrorInternal("open message bus", err)
	}
	messages, err := bus.ReadMessages(after)
	if err != nil {
		if stderrors.Is(err, messagebus.ErrSinceIDNotFound) {
			return apiErrorNotFound("message id not found")
		}
		return apiErrorInternal("read message bus", err)
	}
	resp := make([]MessageResponse, 0, len(messages))
	for _, msg := range messages {
		if msg == nil {
			continue
		}
		resp = append(resp, MessageResponse{
			MsgID:        msg.MsgID,
			Timestamp:    msg.Timestamp,
			Type:         msg.Type,
			ProjectID:    msg.ProjectID,
			TaskID:       msg.TaskID,
			RunID:        msg.RunID,
			ParentMsgIDs: msg.ParentMsgIDs,
			Attachment:   msg.Attachment,
			Body:         msg.Body,
		})
	}
	return writeJSON(w, http.StatusOK, map[string][]MessageResponse{"messages": resp})
}

func (s *Server) handleTaskCreate(w http.ResponseWriter, r *http.Request) *apiError {
	var req TaskCreateRequest
	if err := decodeJSON(r, &req); err != nil {
		return err
	}
	if err := validateIdentifier(req.ProjectID, "project_id"); err != nil {
		return err
	}
	if err := validateIdentifier(req.TaskID, "task_id"); err != nil {
		return err
	}
	if strings.TrimSpace(req.AgentType) == "" {
		return apiErrorBadRequest("agent_type is required")
	}
	if strings.TrimSpace(req.Prompt) == "" {
		return apiErrorBadRequest("prompt is required")
	}

	taskDir := filepath.Join(s.rootDir, req.ProjectID, req.TaskID)
	if err := os.MkdirAll(taskDir, 0o755); err != nil {
		return apiErrorInternal("create task directory", err)
	}
	prompt := strings.TrimSpace(req.Prompt)
	if !strings.HasSuffix(prompt, "\n") {
		prompt += "\n"
	}
	if err := os.WriteFile(filepath.Join(taskDir, "TASK.md"), []byte(prompt), 0o644); err != nil {
		return apiErrorInternal("write TASK.md", err)
	}

	if s.startTasks {
		go s.startTask(req)
	}

	resp := TaskCreateResponse{
		ProjectID: req.ProjectID,
		TaskID:    req.TaskID,
		Status:    "started",
	}
	return writeJSON(w, http.StatusCreated, resp)
}

func (s *Server) handleTaskList(w http.ResponseWriter, r *http.Request) *apiError {
	tasks, err := listTasks(s.rootDir)
	if err != nil {
		return apiErrorInternal("list tasks", err)
	}
	resp := make([]TaskResponse, 0, len(tasks))
	for _, task := range tasks {
		resp = append(resp, TaskResponse{
			ProjectID:    task.ProjectID,
			TaskID:       task.TaskID,
			Status:       task.Status,
			LastActivity: task.LastActivity,
		})
	}
	return writeJSON(w, http.StatusOK, map[string][]TaskResponse{"tasks": resp})
}

func (s *Server) handleTaskGet(w http.ResponseWriter, r *http.Request, projectID, taskID string) *apiError {
	if err := validateIdentifier(taskID, "task_id"); err != nil {
		return err
	}
	var task taskInfo
	var err error
	if projectID != "" {
		if err := validateIdentifier(projectID, "project_id"); err != nil {
			return err
		}
		task, err = getTask(s.rootDir, projectID, taskID)
	} else {
		task, err = findTask(s.rootDir, taskID)
	}
	if err != nil {
		if stderrors.Is(err, errNotFound) {
			return apiErrorNotFound("task not found")
		}
		if stderrors.Is(err, errAmbiguous) {
			return apiErrorConflict("multiple tasks found", map[string]string{"task_id": taskID})
		}
		return apiErrorInternal("get task", err)
	}

	runs, err := listTaskRuns(task.Path)
	if err != nil {
		return apiErrorInternal("list task runs", err)
	}
	resp := TaskResponse{
		ProjectID:    task.ProjectID,
		TaskID:       task.TaskID,
		Status:       task.Status,
		LastActivity: task.LastActivity,
		Runs:         runs,
	}
	return writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handleTaskCancel(w http.ResponseWriter, r *http.Request, projectID, taskID string) *apiError {
	if err := validateIdentifier(taskID, "task_id"); err != nil {
		return err
	}
	var task taskInfo
	var err error
	if projectID != "" {
		if err := validateIdentifier(projectID, "project_id"); err != nil {
			return err
		}
		task, err = getTask(s.rootDir, projectID, taskID)
	} else {
		task, err = findTask(s.rootDir, taskID)
	}
	if err != nil {
		if stderrors.Is(err, errNotFound) {
			return apiErrorNotFound("task not found")
		}
		if stderrors.Is(err, errAmbiguous) {
			return apiErrorConflict("multiple tasks found", map[string]string{"task_id": taskID})
		}
		return apiErrorInternal("get task", err)
	}

	if err := os.WriteFile(filepath.Join(task.Path, "DONE"), []byte(""), 0o644); err != nil {
		return apiErrorInternal("write DONE", err)
	}

	stopped, err := stopTaskRuns(task.Path)
	if err != nil {
		return apiErrorInternal("stop task", err)
	}
	return writeJSON(w, http.StatusAccepted, map[string]int{"stopped_runs": stopped})
}

func (s *Server) handleRunGet(w http.ResponseWriter, r *http.Request, runID string) *apiError {
	info, err := getRunInfo(s.rootDir, runID)
	if err != nil {
		if stderrors.Is(err, errNotFound) {
			return apiErrorNotFound("run not found")
		}
		return apiErrorInternal("get run", err)
	}
	resp := runInfoToResponse(info)
	return writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handleRunInfo(w http.ResponseWriter, r *http.Request, runID string) *apiError {
	path, err := findRunInfoPath(s.rootDir, runID)
	if err != nil {
		if stderrors.Is(err, errNotFound) {
			return apiErrorNotFound("run not found")
		}
		if stderrors.Is(err, errAmbiguous) {
			return apiErrorConflict("multiple runs found", map[string]string{"run_id": runID})
		}
		return apiErrorInternal("get run info", err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return apiErrorInternal("read run-info", err)
	}
	w.Header().Set("Content-Type", "application/x-yaml")
	w.WriteHeader(http.StatusOK)
	if _, err := w.Write(data); err != nil {
		return apiErrorInternal("write response", err)
	}
	return nil
}

func (s *Server) handleRunStop(w http.ResponseWriter, r *http.Request, runID string) *apiError {
	info, err := getRunInfo(s.rootDir, runID)
	if err != nil {
		if stderrors.Is(err, errNotFound) {
			return apiErrorNotFound("run not found")
		}
		return apiErrorInternal("get run", err)
	}
	if !info.EndTime.IsZero() {
		return apiErrorConflict("run already finished", map[string]string{"run_id": runID})
	}
	if err := runner.TerminateProcessGroup(info.PGID); err != nil {
		return apiErrorInternal("stop run", err)
	}
	return writeJSON(w, http.StatusAccepted, map[string]string{"status": "stopping"})
}

func (s *Server) startTask(req TaskCreateRequest) {
	opts := runner.TaskOptions{
		RootDir:     s.rootDir,
		ConfigPath:  s.configPath,
		Agent:       req.AgentType,
		Prompt:      req.Prompt,
		Environment: req.Config,
	}
	if err := runner.RunTask(req.ProjectID, req.TaskID, opts); err != nil {
		s.logger.Printf("task %s/%s failed: %v", req.ProjectID, req.TaskID, err)
	}
}

func runInfoToResponse(info *storage.RunInfo) RunResponse {
	if info == nil {
		return RunResponse{}
	}
	return RunResponse{
		RunID:     info.RunID,
		ProjectID: info.ProjectID,
		TaskID:    info.TaskID,
		Status:    info.Status,
		StartTime: info.StartTime,
		EndTime:   info.EndTime,
		ExitCode:  info.ExitCode,
	}
}

func decodeJSON(r *http.Request, dest interface{}) *apiError {
	if r == nil || r.Body == nil {
		return apiErrorBadRequest("request body is required")
	}
	defer r.Body.Close()
	dec := json.NewDecoder(io.LimitReader(r.Body, maxJSONBodySize))
	dec.DisallowUnknownFields()
	if err := dec.Decode(dest); err != nil {
		return apiErrorBadRequest(fmt.Sprintf("invalid json: %v", err))
	}
	if err := dec.Decode(&struct{}{}); err != io.EOF {
		return apiErrorBadRequest("unexpected trailing data")
	}
	return nil
}

func writeJSON(w http.ResponseWriter, status int, payload interface{}) *apiError {
	if w == nil {
		return apiErrorInternal("response writer is nil", nil)
	}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	if payload == nil {
		return nil
	}
	if err := json.NewEncoder(w).Encode(payload); err != nil {
		return apiErrorInternal("encode response", err)
	}
	return nil
}

func validateIdentifier(value, name string) *apiError {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return apiErrorBadRequest(fmt.Sprintf("%s is required", name))
	}
	if strings.Contains(trimmed, "/") || strings.Contains(trimmed, "\\") {
		return apiErrorBadRequest(fmt.Sprintf("%s must not contain path separators", name))
	}
	if strings.Contains(trimmed, "..") {
		return apiErrorBadRequest(fmt.Sprintf("%s must not contain ..", name))
	}
	return nil
}

func pathSegments(path, prefix string) []string {
	if !strings.HasPrefix(path, prefix) {
		return nil
	}
	trimmed := strings.Trim(strings.TrimPrefix(path, prefix), "/")
	if trimmed == "" {
		return nil
	}
	parts := strings.Split(trimmed, "/")
	filtered := parts[:0]
	for _, part := range parts {
		if part == "" {
			continue
		}
		filtered = append(filtered, part)
	}
	return filtered
}

// storage helpers

type taskInfo struct {
	ProjectID    string
	TaskID       string
	Path         string
	Status       string
	LastActivity time.Time
}

var (
	errNotFound  = stderrors.New("not found")
	errAmbiguous = stderrors.New("ambiguous")
)

func listTasks(root string) ([]taskInfo, error) {
	root = filepath.Clean(strings.TrimSpace(root))
	if root == "." || root == "" {
		return nil, stderrors.New("root dir is empty")
	}
	entries, err := os.ReadDir(root)
	if err != nil {
		if os.IsNotExist(err) {
			return []taskInfo{}, nil
		}
		return nil, errors.Wrap(err, "read root dir")
	}

	var tasks []taskInfo
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		projectID := entry.Name()
		projectDir := filepath.Join(root, projectID)
		projectTasks, err := listProjectTasks(projectID, projectDir)
		if err != nil {
			return nil, err
		}
		tasks = append(tasks, projectTasks...)
	}
	return tasks, nil
}

func listProjectTasks(projectID, projectDir string) ([]taskInfo, error) {
	taskEntries, err := os.ReadDir(projectDir)
	if err != nil {
		return nil, errors.Wrapf(err, "read project dir %s", projectID)
	}
	var tasks []taskInfo
	for _, entry := range taskEntries {
		if !entry.IsDir() {
			continue
		}
		taskID := entry.Name()
		taskPath := filepath.Join(projectDir, taskID)
		if _, err := os.Stat(filepath.Join(taskPath, "TASK.md")); err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, errors.Wrapf(err, "stat TASK.md for %s/%s", projectID, taskID)
		}
		info, err := buildTaskInfo(projectID, taskID, taskPath)
		if err != nil {
			return nil, err
		}
		tasks = append(tasks, info)
	}
	return tasks, nil
}

func getTask(root, projectID, taskID string) (taskInfo, error) {
	root = filepath.Clean(strings.TrimSpace(root))
	if root == "." || root == "" {
		return taskInfo{}, stderrors.New("root dir is empty")
	}
	taskPath := filepath.Join(root, projectID, taskID)
	if _, err := os.Stat(filepath.Join(taskPath, "TASK.md")); err != nil {
		if os.IsNotExist(err) {
			return taskInfo{}, errNotFound
		}
		return taskInfo{}, errors.Wrap(err, "stat TASK.md")
	}
	info, err := buildTaskInfo(projectID, taskID, taskPath)
	if err != nil {
		return taskInfo{}, err
	}
	return info, nil
}

func findTask(root, taskID string) (taskInfo, error) {
	matches := make([]taskInfo, 0, 1)
	tasks, err := listTasks(root)
	if err != nil {
		return taskInfo{}, err
	}
	for _, task := range tasks {
		if task.TaskID == taskID {
			matches = append(matches, task)
		}
	}
	if len(matches) == 0 {
		return taskInfo{}, errNotFound
	}
	if len(matches) > 1 {
		return taskInfo{}, errAmbiguous
	}
	return matches[0], nil
}

func buildTaskInfo(projectID, taskID, taskPath string) (taskInfo, error) {
	status := "idle"
	if _, err := os.Stat(filepath.Join(taskPath, "DONE")); err == nil {
		status = "completed"
	}
	runs, err := listTaskRuns(taskPath)
	if err != nil {
		return taskInfo{}, err
	}
	lastActivity := time.Time{}
	for _, run := range runs {
		candidate := run.EndTime
		if candidate.IsZero() {
			candidate = run.StartTime
		}
		if candidate.After(lastActivity) {
			lastActivity = candidate
		}
		if run.EndTime.IsZero() {
			status = "running"
		}
	}
	if lastActivity.IsZero() {
		if info, err := os.Stat(filepath.Join(taskPath, "TASK.md")); err == nil {
			lastActivity = info.ModTime()
		}
	}
	return taskInfo{
		ProjectID:    projectID,
		TaskID:       taskID,
		Path:         taskPath,
		Status:       status,
		LastActivity: lastActivity,
	}, nil
}

func listTaskRuns(taskPath string) ([]RunResponse, error) {
	runsDir := filepath.Join(taskPath, "runs")
	entries, err := os.ReadDir(runsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []RunResponse{}, nil
		}
		return nil, errors.Wrap(err, "read runs directory")
	}
	responses := make([]RunResponse, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		path := filepath.Join(runsDir, entry.Name(), "run-info.yaml")
		info, err := storage.ReadRunInfo(path)
		if err != nil {
			return nil, errors.Wrapf(err, "read run-info for run %s", entry.Name())
		}
		responses = append(responses, runInfoToResponse(info))
	}
	sort.Slice(responses, func(i, j int) bool {
		return responses[i].RunID < responses[j].RunID
	})
	return responses, nil
}

func listRunResponses(root string) ([]RunResponse, error) {
	root = filepath.Clean(strings.TrimSpace(root))
	if root == "." || root == "" {
		return nil, stderrors.New("root dir is empty")
	}
	var runs []RunResponse
	walkErr := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if d.Name() != "run-info.yaml" {
			return nil
		}
		info, err := storage.ReadRunInfo(path)
		if err != nil {
			return err
		}
		runs = append(runs, runInfoToResponse(info))
		return nil
	})
	if walkErr != nil {
		if os.IsNotExist(walkErr) {
			return []RunResponse{}, nil
		}
		return nil, errors.Wrap(walkErr, "walk run-info")
	}
	sort.Slice(runs, func(i, j int) bool {
		return runs[i].RunID < runs[j].RunID
	})
	return runs, nil
}

func getRunInfo(root, runID string) (*storage.RunInfo, error) {
	path, err := findRunInfoPath(root, runID)
	if err != nil {
		return nil, err
	}
	info, err := storage.ReadRunInfo(path)
	if err != nil {
		return nil, errors.Wrap(err, "read run-info")
	}
	return info, nil
}

func findRunInfoPath(root, runID string) (string, error) {
	trimmed := strings.TrimSpace(runID)
	if trimmed == "" {
		return "", stderrors.New("run id is empty")
	}
	pattern := filepath.Join(root, "*", "*", "runs", trimmed, "run-info.yaml")
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return "", errors.Wrap(err, "glob run-info path")
	}
	if len(matches) == 0 {
		return "", errNotFound
	}
	if len(matches) > 1 {
		return "", errAmbiguous
	}
	return matches[0], nil
}

func stopTaskRuns(taskPath string) (int, error) {
	runs, err := listTaskRunInfos(taskPath)
	if err != nil {
		return 0, err
	}
	stopped := 0
	for _, info := range runs {
		if info.EndTime.IsZero() {
			if err := runner.TerminateProcessGroup(info.PGID); err != nil {
				return stopped, err
			}
			stopped++
		}
	}
	return stopped, nil
}

// listRunResponsesFlat scans a flat runs/ directory produced by run-agent.sh.
// Each subdirectory may contain either run-info.yaml (conductor format) or
// cwd.txt (run-agent.sh format); both are accepted.
func listRunResponsesFlat(root string) ([]RunResponse, error) {
	runsDir := filepath.Join(root, "runs")
	entries, err := os.ReadDir(runsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []RunResponse{}, nil
		}
		return nil, errors.Wrap(err, "read runs dir")
	}
	var runs []RunResponse
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		dir := filepath.Join(runsDir, entry.Name())
		// prefer run-info.yaml
		if info, err := storage.ReadRunInfo(filepath.Join(dir, "run-info.yaml")); err == nil {
			runs = append(runs, runInfoToResponse(info))
			continue
		}
		// fall back to cwd.txt
		if info, err := storage.ParseCwdTxt(filepath.Join(dir, "cwd.txt")); err == nil {
			runs = append(runs, runInfoToResponse(info))
		}
	}
	sort.Slice(runs, func(i, j int) bool {
		return runs[i].RunID < runs[j].RunID
	})
	return runs, nil
}

func listTaskRunInfos(taskPath string) ([]*storage.RunInfo, error) {
	runsDir := filepath.Join(taskPath, "runs")
	entries, err := os.ReadDir(runsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []*storage.RunInfo{}, nil
		}
		return nil, errors.Wrap(err, "read runs directory")
	}
	infos := make([]*storage.RunInfo, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		path := filepath.Join(runsDir, entry.Name(), "run-info.yaml")
		info, err := storage.ReadRunInfo(path)
		if err != nil {
			return nil, errors.Wrapf(err, "read run-info for run %s", entry.Name())
		}
		infos = append(infos, info)
	}
	return infos, nil
}
