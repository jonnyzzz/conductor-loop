package api

import (
	"bytes"
	"errors"
	"log"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestWithLoggingWritesStructuredRequestFields(t *testing.T) {
	root := t.TempDir()
	var logs bytes.Buffer

	server, err := NewServer(Options{
		RootDir:          root,
		DisableTaskStart: true,
		Logger:           log.New(&logs, "", 0),
	})
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}

	h := server.withLogging(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusAccepted)
	}))

	req := httptest.NewRequest(
		http.MethodPost,
		"/api/projects/proj1/tasks/task-a/runs/run-1/stop?project_id=proj1",
		nil,
	)
	req.Header.Set(requestIDHeader, "req-123")
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)

	line := logs.String()
	for _, expected := range []string{
		"event=request_completed",
		"subsystem=api",
		"request_id=req-123",
		"correlation_id=req-123",
		"method=POST",
		"status=202",
		"project_id=proj1",
		"task_id=task-a",
		"run_id=run-1",
	} {
		if !strings.Contains(line, expected) {
			t.Fatalf("expected %q in log line: %q", expected, line)
		}
	}
}

func TestWriteErrorRedactsSensitiveValues(t *testing.T) {
	root := t.TempDir()
	var logs bytes.Buffer

	server, err := NewServer(Options{
		RootDir:          root,
		DisableTaskStart: true,
		Logger:           log.New(&logs, "", 0),
	})
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}

	rec := httptest.NewRecorder()
	server.writeError(rec, &apiError{
		Status:  http.StatusInternalServerError,
		Code:    "INTERNAL",
		Message: "boom",
		Err:     errors.New("authorization=Bearer sk_live_abc123456789"),
	})

	line := logs.String()
	if !strings.Contains(line, "event=request_error") {
		t.Fatalf("expected request_error event in %q", line)
	}
	if strings.Contains(line, "sk_live_abc123456789") {
		t.Fatalf("secret leaked in %q", line)
	}
	if !strings.Contains(line, "[REDACTED]") {
		t.Fatalf("expected redaction marker in %q", line)
	}
}
