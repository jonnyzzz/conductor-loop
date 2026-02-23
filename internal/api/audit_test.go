package api

import (
	"bytes"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestTaskCreateAuditLogWritesSanitizedRecord(t *testing.T) {
	root := t.TempDir()
	projectRoot := t.TempDir()

	server, err := NewServer(Options{
		RootDir:          root,
		DisableTaskStart: true,
		Logger:           log.New(io.Discard, "", 0),
	})
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}

	payload := TaskCreateRequest{
		ProjectID:   "project",
		TaskID:      "task",
		AgentType:   "codex",
		Prompt:      "Deploy with Authorization: Bearer super-secret-token and api_key=raw-secret",
		ProjectRoot: projectRoot,
		Config: map[string]string{
			"mode":      "safe",
			"api_token": "config-secret-token",
		},
	}
	data, _ := json.Marshal(payload)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/tasks", bytes.NewBuffer(data))
	req.Header.Set(requestIDHeader, "req-fixed-123")
	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body.String())
	}
	if got := rec.Header().Get(requestIDHeader); got != "req-fixed-123" {
		t.Fatalf("response request id=%q, want req-fixed-123", got)
	}

	records := readFormSubmissionAuditRecords(t, root)
	if len(records) != 1 {
		t.Fatalf("expected 1 audit record, got %d", len(records))
	}

	record := records[0]
	if got := mustStringField(t, record, "request_id"); got != "req-fixed-123" {
		t.Fatalf("request_id=%q, want req-fixed-123", got)
	}
	if got := mustStringField(t, record, "correlation_id"); got != "req-fixed-123" {
		t.Fatalf("correlation_id=%q, want req-fixed-123", got)
	}
	if got := mustStringField(t, record, "endpoint"); got != "POST /api/v1/tasks" {
		t.Fatalf("endpoint=%q, want POST /api/v1/tasks", got)
	}
	if got := mustStringField(t, record, "project_id"); got != "project" {
		t.Fatalf("project_id=%q, want project", got)
	}
	if got := mustStringField(t, record, "task_id"); got != "task" {
		t.Fatalf("task_id=%q, want task", got)
	}
	if strings.TrimSpace(mustStringField(t, record, "run_id")) == "" {
		t.Fatalf("expected non-empty run_id")
	}

	auditPayload := mustMapField(t, record, "payload")
	prompt := mustStringField(t, auditPayload, "prompt")
	if strings.Contains(prompt, "super-secret-token") || strings.Contains(prompt, "raw-secret") {
		t.Fatalf("prompt was not redacted: %q", prompt)
	}
	if !strings.Contains(prompt, formSubmissionRedactedValue) {
		t.Fatalf("prompt missing redaction marker: %q", prompt)
	}

	config := mustMapField(t, auditPayload, "config")
	if got := mustStringField(t, config, "api_token"); got != formSubmissionRedactedValue {
		t.Fatalf("config.api_token=%q, want %q", got, formSubmissionRedactedValue)
	}
	if got := mustStringField(t, config, "mode"); got != "safe" {
		t.Fatalf("config.mode=%q, want safe", got)
	}

	rawAudit, err := os.ReadFile(filepath.Join(root, formSubmissionAuditDir, formSubmissionAuditFilename))
	if err != nil {
		t.Fatalf("read raw audit file: %v", err)
	}
	if strings.Contains(string(rawAudit), "config-secret-token") {
		t.Fatalf("raw audit file leaked secret token")
	}
}

func TestTaskCreateAuditFailureDoesNotBreakRequest(t *testing.T) {
	root := t.TempDir()
	// Create a file at <root>/_audit so MkdirAll(<root>/_audit) fails.
	if err := os.WriteFile(filepath.Join(root, formSubmissionAuditDir), []byte("block"), 0o644); err != nil {
		t.Fatalf("write blocking file: %v", err)
	}

	var logs bytes.Buffer
	server, err := NewServer(Options{
		RootDir:          root,
		DisableTaskStart: true,
		Logger:           log.New(&logs, "", 0),
	})
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}

	payload := TaskCreateRequest{
		ProjectID: "project",
		TaskID:    "task",
		AgentType: "codex",
		Prompt:    "hello",
	}
	data, _ := json.Marshal(payload)

	req := httptest.NewRequest(http.MethodPost, "/api/v1/tasks", bytes.NewBuffer(data))
	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body.String())
	}
	if _, err := os.Stat(filepath.Join(root, "project", "task", "TASK.md")); err != nil {
		t.Fatalf("expected task creation to succeed despite audit failure: %v", err)
	}
	if !strings.Contains(logs.String(), "event=audit_write_failed") {
		t.Fatalf("expected audit failure warning in logs, got %q", logs.String())
	}
}

func TestProjectMessageAuditLogRedactsBodyAndGeneratesRequestID(t *testing.T) {
	root := t.TempDir()

	server, err := NewServer(Options{
		RootDir:          root,
		DisableTaskStart: true,
		Logger:           log.New(io.Discard, "", 0),
	})
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/projects/proj1/messages",
		strings.NewReader(`{"type":"USER","body":"token=plain-secret Authorization: Bearer msg-secret-token"}`),
	)
	req.Header.Set("Content-Type", "application/json")
	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusCreated {
		t.Fatalf("expected 201, got %d: %s", rec.Code, rec.Body.String())
	}

	requestID := strings.TrimSpace(rec.Header().Get(requestIDHeader))
	if requestID == "" {
		t.Fatalf("expected generated request id in response header")
	}

	records := readFormSubmissionAuditRecords(t, root)
	if len(records) != 1 {
		t.Fatalf("expected 1 audit record, got %d", len(records))
	}

	record := records[0]
	if got := mustStringField(t, record, "request_id"); got != requestID {
		t.Fatalf("request_id=%q, want %q", got, requestID)
	}
	if got := mustStringField(t, record, "correlation_id"); got != requestID {
		t.Fatalf("correlation_id=%q, want %q", got, requestID)
	}
	if got := mustStringField(t, record, "endpoint"); got != "POST /api/projects/{project_id}/messages" {
		t.Fatalf("endpoint=%q, want POST /api/projects/{project_id}/messages", got)
	}
	if got := mustStringField(t, record, "project_id"); got != "proj1" {
		t.Fatalf("project_id=%q, want proj1", got)
	}

	auditPayload := mustMapField(t, record, "payload")
	body := mustStringField(t, auditPayload, "body")
	if strings.Contains(body, "plain-secret") || strings.Contains(body, "msg-secret-token") {
		t.Fatalf("message body was not redacted: %q", body)
	}
	if !strings.Contains(body, formSubmissionRedactedValue) {
		t.Fatalf("message body missing redaction marker: %q", body)
	}
}

func TestTaskResumeWritesAuditRecord(t *testing.T) {
	root := t.TempDir()
	taskDir := filepath.Join(root, "proj1", "task1")
	if err := os.MkdirAll(taskDir, 0o755); err != nil {
		t.Fatalf("mkdir task dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(taskDir, "TASK.md"), []byte("task\n"), 0o644); err != nil {
		t.Fatalf("write TASK.md: %v", err)
	}
	if err := os.WriteFile(filepath.Join(taskDir, "DONE"), []byte(""), 0o644); err != nil {
		t.Fatalf("write DONE: %v", err)
	}

	server, err := NewServer(Options{
		RootDir:          root,
		DisableTaskStart: true,
		Logger:           log.New(io.Discard, "", 0),
	})
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}

	req := httptest.NewRequest(http.MethodPost, "/api/projects/proj1/tasks/task1/resume", nil)
	req.Header.Set(requestIDHeader, "req-resume-1")
	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d: %s", rec.Code, rec.Body.String())
	}

	records := readFormSubmissionAuditRecords(t, root)
	if len(records) != 1 {
		t.Fatalf("expected 1 audit record, got %d", len(records))
	}
	record := records[0]
	if got := mustStringField(t, record, "endpoint"); got != "POST /api/projects/{project_id}/tasks/{task_id}/resume" {
		t.Fatalf("endpoint=%q", got)
	}
	if got := mustStringField(t, record, "project_id"); got != "proj1" {
		t.Fatalf("project_id=%q", got)
	}
	if got := mustStringField(t, record, "task_id"); got != "task1" {
		t.Fatalf("task_id=%q", got)
	}
}

func readFormSubmissionAuditRecords(t *testing.T, root string) []map[string]any {
	t.Helper()

	path := filepath.Join(root, formSubmissionAuditDir, formSubmissionAuditFilename)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read audit file: %v", err)
	}
	lines := strings.Split(strings.TrimSpace(string(data)), "\n")
	records := make([]map[string]any, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		var record map[string]any
		if err := json.Unmarshal([]byte(line), &record); err != nil {
			t.Fatalf("unmarshal audit line %q: %v", line, err)
		}
		records = append(records, record)
	}
	return records
}

func mustStringField(t *testing.T, object map[string]any, key string) string {
	t.Helper()
	value, ok := object[key]
	if !ok {
		t.Fatalf("missing key %q", key)
	}
	text, ok := value.(string)
	if !ok {
		t.Fatalf("key %q type=%T, want string", key, value)
	}
	return text
}

func mustMapField(t *testing.T, object map[string]any, key string) map[string]any {
	t.Helper()
	value, ok := object[key]
	if !ok {
		t.Fatalf("missing key %q", key)
	}
	nested, ok := value.(map[string]any)
	if !ok {
		t.Fatalf("key %q type=%T, want map[string]any", key, value)
	}
	return nested
}
