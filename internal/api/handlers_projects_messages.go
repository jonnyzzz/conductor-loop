package api

import (
	stderrors "errors"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/jonnyzzz/conductor-loop/internal/messagebus"
	"github.com/jonnyzzz/conductor-loop/internal/obslog"
)

// projectPostRequest is the request body for posting a message via project/task APIs.
type projectPostRequest struct {
	Type string `json:"type"`
	Body string `json:"body"`
}

const maxMessageListLimit = 5000
const sinceTailWindowMultiplier = 2
const minSinceTailWindow = 256

func parseMessageListLimit(raw string) int {
	value := strings.TrimSpace(raw)
	if value == "" {
		return 0
	}
	limit, err := strconv.Atoi(value)
	if err != nil || limit <= 0 {
		return 0
	}
	if limit > maxMessageListLimit {
		return maxMessageListLimit
	}
	return limit
}

func sliceMessagesAfterSince(messages []*messagebus.Message, sinceID string) ([]*messagebus.Message, bool) {
	for i, msg := range messages {
		if msg == nil || msg.MsgID != sinceID {
			continue
		}
		if i+1 >= len(messages) {
			return []*messagebus.Message{}, true
		}
		return messages[i+1:], true
	}
	return nil, false
}

// handleProjectMessages handles GET and POST for /api/projects/{p}/messages.
func (s *Server) handleProjectMessages(w http.ResponseWriter, r *http.Request) *apiError {
	parts := splitPath(r.URL.Path, "/api/projects/")
	if len(parts) < 2 || parts[1] != "messages" {
		return apiErrorNotFound("not found")
	}
	projectID := parts[0]
	if err := validateIdentifier(projectID, "project_id"); err != nil {
		return err
	}
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
	busPath := filepath.Join(projectDir, "PROJECT-MESSAGE-BUS.md")
	if err := requirePathWithinRoot(s.rootDir, busPath, "message bus path"); err != nil {
		return err
	}
	switch r.Method {
	case http.MethodGet:
		return s.listBusMessages(w, r, busPath)
	case http.MethodPost:
		return s.postBusMessage(w, r, projectID, "", busPath)
	default:
		return apiErrorMethodNotAllowed()
	}
}

// handleProjectMessagesStream handles GET /api/projects/{p}/messages/stream (SSE).
func (s *Server) handleProjectMessagesStream(w http.ResponseWriter, r *http.Request) *apiError {
	if r.Method != http.MethodGet {
		return apiErrorMethodNotAllowed()
	}
	parts := splitPath(r.URL.Path, "/api/projects/")
	if len(parts) < 3 || parts[1] != "messages" || parts[2] != "stream" {
		return apiErrorNotFound("not found")
	}
	projectID := parts[0]
	if err := validateIdentifier(projectID, "project_id"); err != nil {
		return err
	}
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
	busPath := filepath.Join(projectDir, "PROJECT-MESSAGE-BUS.md")
	if err := requirePathWithinRoot(s.rootDir, busPath, "message bus path"); err != nil {
		return err
	}
	return s.streamMessageBusPath(w, r, busPath)
}

// handleTaskMessages handles GET and POST for /api/projects/{p}/tasks/{t}/messages.
func (s *Server) handleTaskMessages(w http.ResponseWriter, r *http.Request, projectID, taskID string) *apiError {
	if err := validateIdentifier(projectID, "project_id"); err != nil {
		return err
	}
	if err := validateIdentifier(taskID, "task_id"); err != nil {
		return err
	}
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
	busPath := filepath.Join(taskDir, "TASK-MESSAGE-BUS.md")
	if err := requirePathWithinRoot(s.rootDir, busPath, "message bus path"); err != nil {
		return err
	}
	switch r.Method {
	case http.MethodGet:
		return s.listBusMessages(w, r, busPath)
	case http.MethodPost:
		return s.postBusMessage(w, r, projectID, taskID, busPath)
	default:
		return apiErrorMethodNotAllowed()
	}
}

// handleTaskMessagesStream handles GET /api/projects/{p}/tasks/{t}/messages/stream (SSE).
func (s *Server) handleTaskMessagesStream(w http.ResponseWriter, r *http.Request, projectID, taskID string) *apiError {
	if r.Method != http.MethodGet {
		return apiErrorMethodNotAllowed()
	}
	if err := validateIdentifier(projectID, "project_id"); err != nil {
		return err
	}
	if err := validateIdentifier(taskID, "task_id"); err != nil {
		return err
	}
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
	busPath := filepath.Join(taskDir, "TASK-MESSAGE-BUS.md")
	if err := requirePathWithinRoot(s.rootDir, busPath, "message bus path"); err != nil {
		return err
	}
	return s.streamMessageBusPath(w, r, busPath)
}

// listBusMessages reads messages from a message bus file and writes them as JSON.
func (s *Server) listBusMessages(w http.ResponseWriter, r *http.Request, busPath string) *apiError {
	since := strings.TrimSpace(r.URL.Query().Get("since"))
	limit := parseMessageListLimit(r.URL.Query().Get("limit"))
	bus, err := messagebus.NewMessageBus(busPath)
	if err != nil {
		return apiErrorInternal("open message bus", err)
	}
	var messages []*messagebus.Message
	if since == "" && limit > 0 {
		messages, err = bus.ReadLastN(limit)
	} else if since != "" && limit > 0 {
		tailWindow := limit * sinceTailWindowMultiplier
		if tailWindow < minSinceTailWindow {
			tailWindow = minSinceTailWindow
		}
		tailMessages, tailErr := bus.ReadLastN(tailWindow)
		if tailErr != nil {
			err = tailErr
		} else if sliced, ok := sliceMessagesAfterSince(tailMessages, since); ok {
			messages = sliced
		} else {
			messages, err = bus.ReadMessagesSinceLimited(since, limit)
		}
	} else {
		messages, err = bus.ReadMessages(since)
	}
	if err == nil && limit > 0 && len(messages) > limit {
		messages = messages[len(messages)-limit:]
	}
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
	return writeJSON(w, http.StatusOK, map[string]interface{}{"messages": resp})
}

// postBusMessage appends a message to a bus file.
func (s *Server) postBusMessage(w http.ResponseWriter, r *http.Request, projectID, taskID, busPath string) *apiError {
	var req projectPostRequest
	if err := decodeJSON(r, &req); err != nil {
		return err
	}
	if strings.TrimSpace(req.Body) == "" {
		return apiErrorBadRequest("body is required")
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
		ProjectID: projectID,
		TaskID:    taskID,
		Body:      req.Body,
	}
	msgID, err := bus.AppendMessage(msg)
	if err != nil {
		return apiErrorInternal("append message", err)
	}
	endpoint := "POST /api/projects/{project_id}/messages"
	if strings.TrimSpace(taskID) != "" {
		endpoint = "POST /api/projects/{project_id}/tasks/{task_id}/messages"
	}
	s.writeFormSubmissionAudit(r, formSubmissionAuditArgs{
		Endpoint:  endpoint,
		ProjectID: projectID,
		TaskID:    taskID,
		MessageID: msgID,
		Payload:   req,
	})
	obslog.Log(s.logger, "INFO", "api", "bus_message_posted",
		obslog.F("request_id", requestIDFromRequest(r)),
		obslog.F("correlation_id", requestIDFromRequest(r)),
		obslog.F("project_id", projectID),
		obslog.F("task_id", taskID),
		obslog.F("message_id", msgID),
		obslog.F("message_type", msgType),
		obslog.F("endpoint", endpoint),
	)
	return writeJSON(w, http.StatusCreated, PostMessageResponse{
		MsgID:     msgID,
		Timestamp: msg.Timestamp,
	})
}
