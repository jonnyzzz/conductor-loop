package api

import (
	stderrors "errors"
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/jonnyzzz/conductor-loop/internal/messagebus"
)

// projectPostRequest is the request body for posting a message via project/task APIs.
type projectPostRequest struct {
	Type string `json:"type"`
	Body string `json:"body"`
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
	busPath := filepath.Join(s.rootDir, projectID, "PROJECT-MESSAGE-BUS.md")
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
	busPath := filepath.Join(s.rootDir, projectID, "PROJECT-MESSAGE-BUS.md")
	return s.streamMessageBusPath(w, r, busPath)
}

// handleTaskMessages handles GET and POST for /api/projects/{p}/tasks/{t}/messages.
func (s *Server) handleTaskMessages(w http.ResponseWriter, r *http.Request, projectID, taskID string) *apiError {
	busPath := filepath.Join(s.rootDir, projectID, taskID, "TASK-MESSAGE-BUS.md")
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
	busPath := filepath.Join(s.rootDir, projectID, taskID, "TASK-MESSAGE-BUS.md")
	return s.streamMessageBusPath(w, r, busPath)
}

// listBusMessages reads messages from a message bus file and writes them as JSON.
func (s *Server) listBusMessages(w http.ResponseWriter, r *http.Request, busPath string) *apiError {
	since := strings.TrimSpace(r.URL.Query().Get("since"))
	bus, err := messagebus.NewMessageBus(busPath)
	if err != nil {
		return apiErrorInternal("open message bus", err)
	}
	messages, err := bus.ReadMessages(since)
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
	return writeJSON(w, http.StatusCreated, PostMessageResponse{
		MsgID:     msgID,
		Timestamp: msg.Timestamp,
	})
}
