package api

import (
	"encoding/json"
	stderrors "errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/jonnyzzz/conductor-loop/internal/messagebus"
	"github.com/jonnyzzz/conductor-loop/internal/obslog"
	"github.com/jonnyzzz/conductor-loop/internal/runner"
	"github.com/jonnyzzz/conductor-loop/internal/runstate"
	"github.com/jonnyzzz/conductor-loop/internal/storage"
	"github.com/jonnyzzz/conductor-loop/internal/taskdeps"
	"github.com/pkg/errors"
)

const maxJSONBodySize = 1 << 20

// TaskCreateRequest defines the payload for task creation.
type TaskCreateRequest struct {
	ProjectID     string                 `json:"project_id"`
	TaskID        string                 `json:"task_id"`
	AgentType     string                 `json:"agent_type"`
	Prompt        string                 `json:"prompt"`
	Config        map[string]string      `json:"config,omitempty"`
	ProjectRoot   string                 `json:"project_root,omitempty"` // working directory for the task
	AttachMode    string                 `json:"attach_mode,omitempty"`  // "create" | "attach" | "resume"
	ProcessImport *ProcessImportRequest  `json:"process_import,omitempty"`
	DependsOn     []string               `json:"depends_on,omitempty"`
	ThreadParent  *ThreadParentReference `json:"thread_parent,omitempty"`
	// ThreadMessageType is validated only when ThreadParent is set.
	// For threaded task creation, only USER_REQUEST is accepted.
	ThreadMessageType string `json:"thread_message_type,omitempty"`
}

// ProcessImportRequest configures adoption of an already-running process into a new run.
type ProcessImportRequest struct {
	PID         int    `json:"pid"`
	PGID        int    `json:"pgid,omitempty"`
	CommandLine string `json:"commandline,omitempty"`
	StdoutPath  string `json:"stdout_path,omitempty"`
	StderrPath  string `json:"stderr_path,omitempty"`
	Ownership   string `json:"ownership,omitempty"` // "managed" | "external" (default)
}

// TaskCreateResponse defines the response for task creation.
type TaskCreateResponse struct {
	ProjectID     string   `json:"project_id"`
	TaskID        string   `json:"task_id"`
	RunID         string   `json:"run_id"` // the run ID that was allocated
	Status        string   `json:"status"`
	QueuePosition int      `json:"queue_position,omitempty"`
	DependsOn     []string `json:"depends_on,omitempty"`
}

// TaskResponse defines the task response payload.
type TaskResponse struct {
	ProjectID     string        `json:"project_id"`
	TaskID        string        `json:"task_id"`
	Status        string        `json:"status"`
	QueuePosition int           `json:"queue_position,omitempty"`
	LastActivity  time.Time     `json:"last_activity"`
	DependsOn     []string      `json:"depends_on,omitempty"`
	BlockedBy     []string      `json:"blocked_by,omitempty"`
	Runs          []RunResponse `json:"runs,omitempty"`
}

// RunResponse defines run metadata returned by the API.
type RunResponse struct {
	RunID            string    `json:"run_id"`
	ProjectID        string    `json:"project_id"`
	TaskID           string    `json:"task_id"`
	Status           string    `json:"status"`
	ProcessOwnership string    `json:"process_ownership,omitempty"`
	StartTime        time.Time `json:"start_time"`
	EndTime          time.Time `json:"end_time,omitempty"`
	ExitCode         int       `json:"exit_code,omitempty"`
	AgentVersion     string    `json:"agent_version,omitempty"`
	ErrorSummary     string    `json:"error_summary,omitempty"`
}

// MessageResponse defines the message bus entry payload.
type MessageResponse struct {
	MsgID     string              `json:"msg_id"`
	Timestamp time.Time           `json:"timestamp"`
	Type      string              `json:"type"`
	ProjectID string              `json:"project_id"`
	TaskID    string              `json:"task_id,omitempty"`
	RunID     string              `json:"run_id,omitempty"`
	IssueID   string              `json:"issue_id,omitempty"`
	Parents   []messagebus.Parent `json:"parents,omitempty"`
	Links     []messagebus.Link   `json:"links,omitempty"`
	Meta      map[string]string   `json:"meta,omitempty"`
	Body      string              `json:"body"`
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
	ActiveRunsCount  int               `json:"active_runs_count"`
	UptimeSeconds    float64           `json:"uptime_seconds"`
	ConfiguredAgents []string          `json:"configured_agents"`
	Version          string            `json:"version"`
	RunningTasks     []runningTaskItem `json:"running_tasks"`
}

// runningTaskItem describes a single currently-running run in the status response.
type runningTaskItem struct {
	ProjectID string    `json:"project_id"`
	TaskID    string    `json:"task_id"`
	RunID     string    `json:"run_id"`
	Agent     string    `json:"agent"`
	Started   time.Time `json:"started"`
}

func (s *Server) handleStatus(w http.ResponseWriter, r *http.Request) *apiError {
	if r.Method != http.MethodGet {
		return apiErrorMethodNotAllowed()
	}

	infos, err := allRunInfos(s.rootDir)
	if err != nil {
		return apiErrorInternal("list runs", err)
	}

	activeCount := 0
	runningTasks := make([]runningTaskItem, 0)
	for _, info := range infos {
		if info.EndTime.IsZero() {
			activeCount++
			runningTasks = append(runningTasks, runningTaskItem{
				ProjectID: info.ProjectID,
				TaskID:    info.TaskID,
				RunID:     info.RunID,
				Agent:     info.AgentType,
				Started:   info.StartTime,
			})
		}
	}

	for _, extra := range s.extraRoots {
		extraRuns, err := listRunResponsesFlat(extra)
		if err != nil {
			obslog.Log(s.logger, "WARN", "api", "extra_root_scan_failed",
				obslog.F("extra_root", extra),
				obslog.F("error", err),
			)
			continue
		}
		for _, run := range extraRuns {
			if run.EndTime.IsZero() {
				activeCount++
			}
		}
	}

	sort.Slice(runningTasks, func(i, j int) bool {
		return runningTasks[i].Started.Before(runningTasks[j].Started)
	})

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
		RunningTasks:     runningTasks,
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
			obslog.Log(s.logger, "WARN", "api", "extra_root_scan_failed",
				obslog.F("extra_root", extra),
				obslog.F("error", err),
			)
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
	if err := validateIdentifier(projectID, "project_id"); err != nil {
		return err
	}
	after := strings.TrimSpace(r.URL.Query().Get("after"))
	taskID := strings.TrimSpace(r.URL.Query().Get("task_id"))
	if taskID != "" {
		if err := validateIdentifier(taskID, "task_id"); err != nil {
			return err
		}
	}

	var busPath string
	if taskID != "" {
		taskDir, ok := findProjectTaskDir(s.rootDir, projectID, taskID)
		if !ok {
			var pathErr *apiError
			taskDir, pathErr = joinPathWithinRoot(s.rootDir, projectID, taskID)
			if pathErr != nil {
				return pathErr
			}
		}
		if err := requirePathWithinRoot(s.rootDir, taskDir, "task path"); err != nil {
			return err
		}
		busPath = filepath.Join(taskDir, "TASK-MESSAGE-BUS.md")
	} else {
		projectDir, ok := findProjectDir(s.rootDir, projectID)
		if !ok {
			var pathErr *apiError
			projectDir, pathErr = joinPathWithinRoot(s.rootDir, projectID)
			if pathErr != nil {
				return pathErr
			}
		}
		if err := requirePathWithinRoot(s.rootDir, projectDir, "project path"); err != nil {
			return err
		}
		busPath = filepath.Join(projectDir, "PROJECT-MESSAGE-BUS.md")
	}
	if err := requirePathWithinRoot(s.rootDir, busPath, "message bus path"); err != nil {
		return err
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
			MsgID:     msg.MsgID,
			Timestamp: msg.Timestamp,
			Type:      msg.Type,
			ProjectID: msg.ProjectID,
			TaskID:    msg.TaskID,
			RunID:     msg.RunID,
			IssueID:   msg.IssueID,
			Parents:   msg.Parents,
			Links:     msg.Links,
			Meta:      msg.Meta,
			Body:      msg.Body,
		})
	}
	return writeJSON(w, http.StatusOK, map[string][]MessageResponse{"messages": resp})
}

// PostMessageRequest defines the payload for posting a message to the bus.
type PostMessageRequest struct {
	ProjectID string `json:"project_id"`
	TaskID    string `json:"task_id,omitempty"`
	RunID     string `json:"run_id,omitempty"`
	Type      string `json:"type,omitempty"`
	Body      string `json:"body"`
}

// PostMessageResponse defines the response for a posted message.
type PostMessageResponse struct {
	MsgID     string    `json:"msg_id"`
	Timestamp time.Time `json:"timestamp"`
}

func (s *Server) handlePostMessage(w http.ResponseWriter, r *http.Request) *apiError {
	var req PostMessageRequest
	if err := decodeJSON(r, &req); err != nil {
		return err
	}
	if err := validateIdentifier(req.ProjectID, "project_id"); err != nil {
		return err
	}
	if strings.TrimSpace(req.Body) == "" {
		return apiErrorBadRequest("body is required")
	}
	taskID := strings.TrimSpace(req.TaskID)
	if taskID != "" {
		if err := validateIdentifier(taskID, "task_id"); err != nil {
			return err
		}
	}

	var busPath string
	if taskID != "" {
		taskDir, ok := findProjectTaskDir(s.rootDir, req.ProjectID, taskID)
		if !ok {
			var pathErr *apiError
			taskDir, pathErr = joinPathWithinRoot(s.rootDir, req.ProjectID, taskID)
			if pathErr != nil {
				return pathErr
			}
		}
		if err := requirePathWithinRoot(s.rootDir, taskDir, "task path"); err != nil {
			return err
		}
		busPath = filepath.Join(taskDir, "TASK-MESSAGE-BUS.md")
	} else {
		projectDir, ok := findProjectDir(s.rootDir, req.ProjectID)
		if !ok {
			var pathErr *apiError
			projectDir, pathErr = joinPathWithinRoot(s.rootDir, req.ProjectID)
			if pathErr != nil {
				return pathErr
			}
		}
		if err := requirePathWithinRoot(s.rootDir, projectDir, "project path"); err != nil {
			return err
		}
		busPath = filepath.Join(projectDir, "PROJECT-MESSAGE-BUS.md")
	}
	if err := requirePathWithinRoot(s.rootDir, busPath, "message bus path"); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(busPath), 0o755); err != nil {
		return apiErrorInternal("create message bus directory", err)
	}

	bus, err := messagebus.NewMessageBus(busPath)
	if err != nil {
		return apiErrorInternal("open message bus", err)
	}

	msgType := strings.TrimSpace(req.Type)
	if msgType == "" {
		msgType = "USER"
	}

	msg := &messagebus.Message{
		Type:      msgType,
		ProjectID: req.ProjectID,
		TaskID:    taskID,
		RunID:     strings.TrimSpace(req.RunID),
		Body:      req.Body,
	}
	msgID, err := bus.AppendMessage(msg)
	if err != nil {
		return apiErrorInternal("append message", err)
	}
	req.TaskID = taskID
	s.writeFormSubmissionAudit(r, formSubmissionAuditArgs{
		Endpoint:  "POST /api/v1/messages",
		ProjectID: req.ProjectID,
		TaskID:    taskID,
		MessageID: msgID,
		Payload:   req,
	})
	obslog.Log(s.logger, "INFO", "api", "bus_message_posted",
		obslog.F("request_id", requestIDFromRequest(r)),
		obslog.F("correlation_id", requestIDFromRequest(r)),
		obslog.F("project_id", req.ProjectID),
		obslog.F("task_id", taskID),
		obslog.F("run_id", msg.RunID),
		obslog.F("message_id", msgID),
		obslog.F("message_type", msgType),
	)

	return writeJSON(w, http.StatusCreated, PostMessageResponse{
		MsgID:     msgID,
		Timestamp: msg.Timestamp,
	})
}

func (s *Server) handleTaskCreate(w http.ResponseWriter, r *http.Request) *apiError {
	var req TaskCreateRequest
	if err := decodeJSON(r, &req); err != nil {
		return err
	}
	if s.startTasks {
		s.rootRunGateMu.Lock()
		blockedErr := s.taskCreateBlockedBySelfUpdateLocked()
		s.rootRunGateMu.Unlock()
		if blockedErr != nil {
			return blockedErr
		}
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
	threadParentCtx, threadErr := s.validateThreadedParent(&req)
	if threadErr != nil {
		return threadErr
	}

	// Validate project_root if provided.
	projectRoot := strings.TrimSpace(req.ProjectRoot)
	if strings.HasPrefix(projectRoot, "~/") {
		if home, homeErr := os.UserHomeDir(); homeErr == nil {
			projectRoot = filepath.Join(home, projectRoot[2:])
		}
	}
	if projectRoot != "" {
		if _, err := os.Stat(projectRoot); err != nil {
			if os.IsNotExist(err) {
				return apiErrorBadRequest(fmt.Sprintf("project_root does not exist: %s", projectRoot))
			}
			return apiErrorInternal("stat project_root", err)
		}
	}
	req.ProjectRoot = projectRoot

	var (
		dependsOn      []string
		dependsUpdated bool
	)
	if req.DependsOn != nil {
		var err error
		dependsOn, err = taskdeps.Normalize(req.TaskID, req.DependsOn)
		if err != nil {
			return apiErrorBadRequest(err.Error())
		}
		dependsUpdated = true
	}

	// Validate and normalise attach_mode.
	attachMode := strings.TrimSpace(req.AttachMode)
	if attachMode == "" {
		attachMode = "create"
	}
	switch attachMode {
	case "create", "attach", "resume":
	default:
		return apiErrorBadRequest(fmt.Sprintf("invalid attach_mode %q: must be create, attach, or resume", attachMode))
	}
	req.AttachMode = attachMode

	if req.ProcessImport != nil {
		if req.ProcessImport.PID <= 0 {
			return apiErrorBadRequest("process_import.pid must be > 0")
		}
		if strings.TrimSpace(req.ProcessImport.StdoutPath) == "" && strings.TrimSpace(req.ProcessImport.StderrPath) == "" {
			return apiErrorBadRequest("process_import requires stdout_path and/or stderr_path")
		}
		if ownership := strings.TrimSpace(req.ProcessImport.Ownership); ownership != "" {
			normalized := storage.NormalizeProcessOwnership(ownership)
			if normalized != strings.ToLower(ownership) {
				return apiErrorBadRequest("process_import.ownership must be managed or external")
			}
		}
	}

	// Use the existing task directory if found, otherwise create at the default location.
	taskDir, ok := findProjectTaskDir(s.rootDir, req.ProjectID, req.TaskID)
	if !ok {
		taskDir = filepath.Join(s.rootDir, req.ProjectID, req.TaskID)
	}
	if err := os.MkdirAll(taskDir, 0o755); err != nil {
		return apiErrorInternal("create task directory", err)
	}

	if !dependsUpdated {
		var err error
		dependsOn, err = taskdeps.ReadDependsOn(taskDir)
		if err != nil {
			return apiErrorInternal("read task dependencies", err)
		}
	}
	if err := taskdeps.ValidateNoCycle(s.rootDir, req.ProjectID, req.TaskID, dependsOn); err != nil {
		return apiErrorConflict(err.Error(), map[string]string{"task_id": req.TaskID})
	}
	if dependsUpdated {
		if err := taskdeps.WriteDependsOn(taskDir, dependsOn); err != nil {
			return apiErrorInternal("write task dependencies", err)
		}
	}
	req.DependsOn = dependsOn

	// Preserve prompt bytes as provided by the client. Validation above already
	// ensures the prompt contains non-whitespace content.
	prompt := req.Prompt
	if !strings.HasSuffix(prompt, "\n") {
		prompt += "\n"
	}

	// Write TASK.md only when absent. Never overwrite an existing task prompt.
	taskMDPath := filepath.Join(taskDir, "TASK.md")
	writeTaskMD := false
	if _, err := os.Stat(taskMDPath); err != nil {
		if !os.IsNotExist(err) {
			return apiErrorInternal("stat TASK.md", err)
		}
		writeTaskMD = true
	}
	if writeTaskMD {
		if err := os.WriteFile(taskMDPath, []byte(prompt), 0o644); err != nil {
			return apiErrorInternal("write TASK.md", err)
		}
	}

	// For "resume" mode, prepend the restart prefix so the first run continues the task.
	runPrompt := prompt
	if attachMode == "resume" {
		runPrompt = runner.RestartPrefix + prompt
	}

	// Pre-allocate a run directory so we can return the run_id immediately.
	runsDir := filepath.Join(taskDir, "runs")
	if err := os.MkdirAll(runsDir, 0o755); err != nil {
		return apiErrorInternal("create runs directory", err)
	}
	runID, runDir, err := runner.AllocateRunDir(runsDir)
	if err != nil {
		return apiErrorInternal("allocate run directory", err)
	}
	if threadParentCtx != nil {
		if err := s.persistThreadedTaskLinkage(taskDir, req, runID, threadParentCtx); err != nil {
			return apiErrorInternal("persist threaded linkage", err)
		}
	}
	s.writeFormSubmissionAudit(r, formSubmissionAuditArgs{
		Endpoint:  "POST /api/v1/tasks",
		ProjectID: req.ProjectID,
		TaskID:    req.TaskID,
		RunID:     runID,
		Payload:   req,
	})

	responseStatus := "started"
	queuePosition := 0
	if s.startTasks {
		s.rootRunGateMu.Lock()
		if blockedErr := s.taskCreateBlockedBySelfUpdateLocked(); blockedErr != nil {
			s.rootRunGateMu.Unlock()
			return blockedErr
		}
		if s.rootTaskPlanner != nil {
			planResult, planErr := s.rootTaskPlanner.Submit(req, runDir, runPrompt)
			if planErr != nil {
				s.rootRunGateMu.Unlock()
				return apiErrorInternal("plan root task start", planErr)
			}
			responseStatus = planResult.Status
			queuePosition = planResult.QueuePosition
			s.launchPlannedTasksLocked(planResult.Launches)
		} else {
			s.launchPlannedTasksLocked([]rootTaskLaunch{{
				Request:   req,
				RunID:     runID,
				RunDir:    runDir,
				RunPrompt: runPrompt,
			}})
		}
		s.rootRunGateMu.Unlock()
	}
	obslog.Log(s.logger, "INFO", "api", "task_create_accepted",
		obslog.F("request_id", requestIDFromRequest(r)),
		obslog.F("correlation_id", requestIDFromRequest(r)),
		obslog.F("project_id", req.ProjectID),
		obslog.F("task_id", req.TaskID),
		obslog.F("run_id", runID),
		obslog.F("agent_type", req.AgentType),
		obslog.F("attach_mode", attachMode),
		obslog.F("planner_status", responseStatus),
		obslog.F("queue_position", queuePosition),
		obslog.F("task_start_enabled", s.startTasks),
	)

	resp := TaskCreateResponse{
		ProjectID:     req.ProjectID,
		TaskID:        req.TaskID,
		RunID:         runID,
		Status:        responseStatus,
		QueuePosition: queuePosition,
		DependsOn:     dependsOn,
	}
	return writeJSON(w, http.StatusCreated, resp)
}

// taskCreateBlockedBySelfUpdateLocked checks whether root-run admission is blocked.
// Caller must hold s.rootRunGateMu.
func (s *Server) taskCreateBlockedBySelfUpdateLocked() *apiError {
	if s == nil || s.selfUpdate == nil || !s.selfUpdate.blocksNewRootRuns() {
		return nil
	}
	state := s.selfUpdate.status().State
	return apiErrorConflict(
		"self-update drain is in progress; task creation is temporarily unavailable",
		map[string]string{"self_update_state": state},
	)
}

func (s *Server) handleTaskList(w http.ResponseWriter, r *http.Request) *apiError {
	tasks, err := listTasksWithQueue(s.rootDir, s.taskQueueSnapshot())
	if err != nil {
		return apiErrorInternal("list tasks", err)
	}
	resp := make([]TaskResponse, 0, len(tasks))
	for _, task := range tasks {
		resp = append(resp, TaskResponse{
			ProjectID:     task.ProjectID,
			TaskID:        task.TaskID,
			Status:        task.Status,
			QueuePosition: task.QueuePosition,
			LastActivity:  task.LastActivity,
			DependsOn:     task.DependsOn,
			BlockedBy:     task.BlockedBy,
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
	queueSnapshot := s.taskQueueSnapshot()
	if projectID != "" {
		if err := validateIdentifier(projectID, "project_id"); err != nil {
			return err
		}
		task, err = getTaskWithQueue(s.rootDir, projectID, taskID, queueSnapshot)
	} else {
		task, err = findTaskWithQueue(s.rootDir, taskID, queueSnapshot)
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
		ProjectID:     task.ProjectID,
		TaskID:        task.TaskID,
		Status:        task.Status,
		QueuePosition: task.QueuePosition,
		LastActivity:  task.LastActivity,
		DependsOn:     task.DependsOn,
		BlockedBy:     task.BlockedBy,
		Runs:          runs,
	}
	return writeJSON(w, http.StatusOK, resp)
}

func (s *Server) handleTaskCancel(w http.ResponseWriter, r *http.Request, projectID, taskID string) *apiError {
	if err := validateIdentifier(taskID, "task_id"); err != nil {
		return err
	}
	var task taskInfo
	var err error
	queueSnapshot := s.taskQueueSnapshot()
	if projectID != "" {
		if err := validateIdentifier(projectID, "project_id"); err != nil {
			return err
		}
		task, err = getTaskWithQueue(s.rootDir, projectID, taskID, queueSnapshot)
	} else {
		task, err = findTaskWithQueue(s.rootDir, taskID, queueSnapshot)
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
	if s.rootTaskPlanner != nil {
		launches, planErr := s.rootTaskPlanner.DropQueuedForTask(task.ProjectID, task.TaskID)
		if planErr != nil {
			obslog.Log(s.logger, "ERROR", "api", "task_queue_drop_failed",
				obslog.F("project_id", task.ProjectID),
				obslog.F("task_id", task.TaskID),
				obslog.F("error", planErr),
			)
		} else {
			s.launchPlannedTasks(launches)
		}
	}
	s.writeFormSubmissionAudit(r, formSubmissionAuditArgs{
		Endpoint:  "DELETE /api/v1/tasks/{task_id}",
		ProjectID: task.ProjectID,
		TaskID:    task.TaskID,
		Payload: map[string]any{
			"stopped_runs": stopped,
		},
	})
	obslog.Log(s.logger, "WARN", "api", "task_cancel_requested",
		obslog.F("request_id", requestIDFromRequest(r)),
		obslog.F("correlation_id", requestIDFromRequest(r)),
		obslog.F("project_id", task.ProjectID),
		obslog.F("task_id", task.TaskID),
		obslog.F("stopped_runs", stopped),
	)
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
	if !storage.CanTerminateProcess(info) {
		return apiErrorConflict("run is externally owned and cannot be stopped by conductor", map[string]string{"run_id": runID})
	}
	if err := runner.TerminateProcessGroup(info.PGID); err != nil {
		return apiErrorInternal("stop run", err)
	}
	s.writeFormSubmissionAudit(r, formSubmissionAuditArgs{
		Endpoint:  "POST /api/v1/runs/{run_id}/stop",
		ProjectID: info.ProjectID,
		TaskID:    info.TaskID,
		RunID:     info.RunID,
		Payload: map[string]any{
			"pgid": info.PGID,
		},
	})
	obslog.Log(s.logger, "WARN", "api", "run_stop_requested",
		obslog.F("request_id", requestIDFromRequest(r)),
		obslog.F("correlation_id", requestIDFromRequest(r)),
		obslog.F("project_id", info.ProjectID),
		obslog.F("task_id", info.TaskID),
		obslog.F("run_id", info.RunID),
		obslog.F("pgid", info.PGID),
	)
	return writeJSON(w, http.StatusAccepted, map[string]string{"status": "stopping"})
}

func (s *Server) conductorURL() string {
	host := resolveLoopbackHost(s.apiConfig.Host)
	port := s.ActualPort()
	if port == 0 {
		port = s.apiConfig.Port
	}
	if port == 0 {
		port = 14355
	}
	return httpBaseURL(host, port)
}

func (s *Server) startTask(req TaskCreateRequest, firstRunDir, prompt string) {
	runID := strings.TrimSpace(filepath.Base(firstRunDir))
	parentRunID := ""
	if req.ThreadParent != nil {
		parentRunID = strings.TrimSpace(req.ThreadParent.RunID)
	}
	if req.ProcessImport != nil {
		opts := runner.ImportOptions{
			RootDir:            s.rootDir,
			WorkingDir:         strings.TrimSpace(req.ProjectRoot),
			PreallocatedRunDir: firstRunDir,
			ParentRunID:        parentRunID,
			Process: runner.ImportedProcess{
				PID:         req.ProcessImport.PID,
				PGID:        req.ProcessImport.PGID,
				AgentType:   req.AgentType,
				CommandLine: req.ProcessImport.CommandLine,
				StdoutPath:  req.ProcessImport.StdoutPath,
				StderrPath:  req.ProcessImport.StderrPath,
				Ownership:   req.ProcessImport.Ownership,
			},
		}
		obslog.Log(s.logger, "INFO", "api", "task_import_started",
			obslog.F("project_id", req.ProjectID),
			obslog.F("task_id", req.TaskID),
			obslog.F("run_id", runID),
			obslog.F("agent_type", req.AgentType),
			obslog.F("import_pid", req.ProcessImport.PID),
		)
		s.metrics.IncActiveRuns()
		if err := runner.RunImportedProcess(req.ProjectID, req.TaskID, opts); err != nil {
			obslog.Log(s.logger, "ERROR", "api", "task_import_failed",
				obslog.F("project_id", req.ProjectID),
				obslog.F("task_id", req.TaskID),
				obslog.F("run_id", runID),
				obslog.F("agent_type", req.AgentType),
				obslog.F("error", err),
			)
			s.metrics.DecActiveRuns()
			s.metrics.IncFailedRuns()
		} else {
			obslog.Log(s.logger, "INFO", "api", "task_import_completed",
				obslog.F("project_id", req.ProjectID),
				obslog.F("task_id", req.TaskID),
				obslog.F("run_id", runID),
				obslog.F("agent_type", req.AgentType),
			)
			s.metrics.DecActiveRuns()
			s.metrics.IncCompletedRuns()
		}
		return
	}

	opts := runner.TaskOptions{
		RootDir:      s.rootDir,
		ConfigPath:   s.configPath,
		Agent:        req.AgentType,
		Prompt:       prompt,
		WorkingDir:   strings.TrimSpace(req.ProjectRoot),
		Environment:  req.Config,
		FirstRunDir:  firstRunDir,
		ConductorURL: s.conductorURL(),
		ParentRunID:  parentRunID,
		DependsOn:    req.DependsOn,
	}
	obslog.Log(s.logger, "INFO", "api", "task_run_started",
		obslog.F("project_id", req.ProjectID),
		obslog.F("task_id", req.TaskID),
		obslog.F("run_id", runID),
		obslog.F("agent_type", req.AgentType),
	)
	s.metrics.IncActiveRuns()
	if err := runner.RunTask(req.ProjectID, req.TaskID, opts); err != nil {
		obslog.Log(s.logger, "ERROR", "api", "task_run_failed",
			obslog.F("project_id", req.ProjectID),
			obslog.F("task_id", req.TaskID),
			obslog.F("run_id", runID),
			obslog.F("agent_type", req.AgentType),
			obslog.F("error", err),
		)
		s.metrics.DecActiveRuns()
		s.metrics.IncFailedRuns()
	} else {
		obslog.Log(s.logger, "INFO", "api", "task_run_completed",
			obslog.F("project_id", req.ProjectID),
			obslog.F("task_id", req.TaskID),
			obslog.F("run_id", runID),
			obslog.F("agent_type", req.AgentType),
		)
		s.metrics.DecActiveRuns()
		s.metrics.IncCompletedRuns()
	}
}

func (s *Server) launchPlannedTasks(launches []rootTaskLaunch) {
	if s == nil || len(launches) == 0 {
		return
	}
	s.rootRunGateMu.Lock()
	defer s.rootRunGateMu.Unlock()
	s.launchPlannedTasksLocked(launches)
}

func (s *Server) onSelfUpdateDrainReleased() {
	if s == nil || !s.startTasks || s.rootTaskPlanner == nil {
		return
	}
	s.rootRunGateMu.Lock()
	defer s.rootRunGateMu.Unlock()
	if s.selfUpdate != nil && s.selfUpdate.blocksNewRootRuns() {
		return
	}
	launches, err := s.rootTaskPlanner.Recover()
	if err != nil {
		obslog.Log(s.logger, "ERROR", "api", "root_task_planner_recover_after_self_update_failed",
			obslog.F("error", err),
		)
		return
	}
	s.launchPlannedTasksLocked(launches)
}

// launchPlannedTasksLocked starts planned root runs while holding s.rootRunGateMu.
func (s *Server) launchPlannedTasksLocked(launches []rootTaskLaunch) {
	if s == nil || len(launches) == 0 {
		return
	}
	for _, launch := range launches {
		launch := launch
		if strings.TrimSpace(launch.RunDir) == "" {
			continue
		}
		s.activeRootRuns.Add(1)
		s.taskWg.Add(1)
		go func() {
			defer s.taskWg.Done()
			defer s.activeRootRuns.Add(-1)
			s.startTask(launch.Request, launch.RunDir, launch.RunPrompt)
			if s.rootTaskPlanner == nil {
				return
			}
			s.rootRunGateMu.Lock()
			defer s.rootRunGateMu.Unlock()
			allowSchedule := true
			if s.selfUpdate != nil && s.selfUpdate.blocksNewRootRuns() {
				allowSchedule = false
			}
			next, err := s.rootTaskPlanner.OnRunFinishedWithScheduling(
				launch.Request.ProjectID,
				launch.Request.TaskID,
				launch.RunID,
				allowSchedule,
			)
			if err != nil {
				obslog.Log(s.logger, "ERROR", "api", "root_task_planner_finish_failed",
					obslog.F("project_id", launch.Request.ProjectID),
					obslog.F("task_id", launch.Request.TaskID),
					obslog.F("run_id", launch.RunID),
					obslog.F("allow_schedule", allowSchedule),
					obslog.F("error", err),
				)
				return
			}
			s.launchPlannedTasksLocked(next)
		}()
	}
}

func (s *Server) taskQueueSnapshot() map[taskQueueKey]taskQueueState {
	if s == nil || s.rootTaskPlanner == nil {
		return nil
	}
	snapshot, err := s.rootTaskPlanner.Snapshot()
	if err != nil {
		obslog.Log(s.logger, "ERROR", "api", "root_task_queue_snapshot_failed",
			obslog.F("error", err),
		)
		return nil
	}
	return snapshot
}

func runInfoToResponse(info *storage.RunInfo) RunResponse {
	if info == nil {
		return RunResponse{}
	}
	return RunResponse{
		RunID:            info.RunID,
		ProjectID:        info.ProjectID,
		TaskID:           info.TaskID,
		Status:           info.Status,
		ProcessOwnership: storage.EffectiveProcessOwnership(info),
		StartTime:        info.StartTime,
		EndTime:          info.EndTime,
		ExitCode:         info.ExitCode,
		AgentVersion:     info.AgentVersion,
		ErrorSummary:     info.ErrorSummary,
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
	decoded := trimmed
	if unescaped, err := url.PathUnescape(trimmed); err == nil {
		decoded = unescaped
	}
	if strings.Contains(decoded, "/") || strings.Contains(decoded, "\\") {
		return apiErrorBadRequest(fmt.Sprintf("%s must not contain path separators", name))
	}
	if strings.Contains(decoded, "..") {
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
	ProjectID     string
	TaskID        string
	Path          string
	Status        string
	QueuePosition int
	LastActivity  time.Time
	DependsOn     []string
	BlockedBy     []string
}

var (
	errNotFound  = stderrors.New("not found")
	errAmbiguous = stderrors.New("ambiguous")
)

func listTasks(root string) ([]taskInfo, error) {
	return listTasksWithQueue(root, nil)
}

func listTasksWithQueue(root string, queue map[taskQueueKey]taskQueueState) ([]taskInfo, error) {
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
		projectTasks, err := listProjectTasksWithQueue(projectID, projectDir, queue)
		if err != nil {
			return nil, err
		}
		tasks = append(tasks, projectTasks...)
	}
	return tasks, nil
}

func listProjectTasks(projectID, projectDir string) ([]taskInfo, error) {
	return listProjectTasksWithQueue(projectID, projectDir, nil)
}

func listProjectTasksWithQueue(projectID, projectDir string, queue map[taskQueueKey]taskQueueState) ([]taskInfo, error) {
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
		info, err := buildTaskInfoWithQueue(filepath.Dir(projectDir), projectID, taskID, taskPath, queue)
		if err != nil {
			return nil, err
		}
		tasks = append(tasks, info)
	}
	return tasks, nil
}

func getTask(root, projectID, taskID string) (taskInfo, error) {
	return getTaskWithQueue(root, projectID, taskID, nil)
}

func getTaskWithQueue(root, projectID, taskID string, queue map[taskQueueKey]taskQueueState) (taskInfo, error) {
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
	info, err := buildTaskInfoWithQueue(root, projectID, taskID, taskPath, queue)
	if err != nil {
		return taskInfo{}, err
	}
	return info, nil
}

func findTask(root, taskID string) (taskInfo, error) {
	return findTaskWithQueue(root, taskID, nil)
}

func findTaskWithQueue(root, taskID string, queue map[taskQueueKey]taskQueueState) (taskInfo, error) {
	matches := make([]taskInfo, 0, 1)
	tasks, err := listTasksWithQueue(root, queue)
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

func buildTaskInfo(rootDir, projectID, taskID, taskPath string) (taskInfo, error) {
	return buildTaskInfoWithQueue(rootDir, projectID, taskID, taskPath, nil)
}

func buildTaskInfoWithQueue(rootDir, projectID, taskID, taskPath string, queue map[taskQueueKey]taskQueueState) (taskInfo, error) {
	status := "idle"
	done := false
	if _, err := os.Stat(filepath.Join(taskPath, "DONE")); err == nil {
		status = storage.StatusAllFinished
		done = true
	}
	dependsOn, err := taskdeps.ReadDependsOn(taskPath)
	if err != nil {
		return taskInfo{}, errors.Wrapf(err, "read task dependencies for %s/%s", projectID, taskID)
	}
	runs, err := listTaskRuns(taskPath)
	if err != nil {
		return taskInfo{}, err
	}
	lastActivity := time.Time{}
	hasRunning := false
	hasFailed := false
	for _, run := range runs {
		candidate := run.EndTime
		if candidate.IsZero() {
			candidate = run.StartTime
		}
		if candidate.After(lastActivity) {
			lastActivity = candidate
		}
		if run.EndTime.IsZero() {
			hasRunning = true
		} else if run.Status == storage.StatusFailed {
			hasFailed = true
		}
	}
	if hasRunning {
		if hasFailed {
			status = storage.StatusPartialFail
		} else {
			status = storage.StatusRunning
		}
	}
	blockedBy := []string(nil)
	if !done && status == "idle" && len(dependsOn) > 0 {
		blockedBy, err = taskdeps.BlockedBy(rootDir, projectID, dependsOn)
		if err != nil {
			return taskInfo{}, errors.Wrapf(err, "resolve blocked dependencies for %s/%s", projectID, taskID)
		}
		if len(blockedBy) > 0 {
			status = storage.StatusBlocked
		}
	}
	queuePosition := 0
	if !done && status != storage.StatusRunning && status != storage.StatusPartialFail {
		if state, ok := queue[taskQueueKey{ProjectID: projectID, TaskID: taskID}]; ok && state.Queued {
			status = storage.StatusQueued
			queuePosition = state.QueuePosition
		}
	}
	if lastActivity.IsZero() {
		if info, err := os.Stat(filepath.Join(taskPath, "TASK.md")); err == nil {
			lastActivity = info.ModTime()
		}
	}
	return taskInfo{
		ProjectID:     projectID,
		TaskID:        taskID,
		Path:          taskPath,
		Status:        status,
		QueuePosition: queuePosition,
		LastActivity:  lastActivity,
		DependsOn:     dependsOn,
		BlockedBy:     blockedBy,
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
		info, err := runstate.ReadRunInfo(path)
		if err != nil {
			if os.IsNotExist(errors.Cause(err)) || os.IsNotExist(err) {
				// Pre-allocated run directory without run-info.yaml yet; skip it.
				continue
			}
			return nil, errors.Wrapf(err, "read run-info for run %s", entry.Name())
		}
		responses = append(responses, runInfoToResponse(info))
	}
	sort.Slice(responses, func(i, j int) bool {
		return responses[i].RunID < responses[j].RunID
	})
	return responses, nil
}

func allRunInfos(root string) ([]*storage.RunInfo, error) {
	root = filepath.Clean(strings.TrimSpace(root))
	if root == "." || root == "" {
		return nil, stderrors.New("root dir is empty")
	}
	var infos []*storage.RunInfo
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
		info, err := runstate.ReadRunInfo(path)
		if err != nil {
			return err
		}
		infos = append(infos, info)
		return nil
	})
	if walkErr != nil {
		if os.IsNotExist(walkErr) {
			return []*storage.RunInfo{}, nil
		}
		return nil, errors.Wrap(walkErr, "walk run-info")
	}
	return infos, nil
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
		info, err := runstate.ReadRunInfo(path)
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
	info, err := runstate.ReadRunInfo(path)
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
			if !storage.CanTerminateProcess(info) {
				continue
			}
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
		if info, err := runstate.ReadRunInfo(filepath.Join(dir, "run-info.yaml")); err == nil {
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
		info, err := runstate.ReadRunInfo(path)
		if err != nil {
			if os.IsNotExist(errors.Cause(err)) || os.IsNotExist(err) {
				// Pre-allocated run directory without run-info.yaml yet; skip it.
				continue
			}
			return nil, errors.Wrapf(err, "read run-info for run %s", entry.Name())
		}
		infos = append(infos, info)
	}
	return infos, nil
}
