package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// busSSEBody builds a simple SSE response with bus message events followed by a done event.
func busSSEBody(msgs []busMessage) string {
	var sb strings.Builder
	for _, msg := range msgs {
		data, _ := json.Marshal(msg)
		fmt.Fprintf(&sb, "data: %s\n\n", data)
	}
	sb.WriteString("event: done\ndata: {}\n\n")
	return sb.String()
}

func makeBusMsg(msgType, body string) busMessage {
	return busMessage{
		MsgID:     "MSG-20260221-070000-12345-PID123-0001",
		Timestamp: time.Date(2026, 2, 21, 7, 0, 0, 0, time.UTC),
		Type:      msgType,
		ProjectID: "test-project",
		TaskID:    "task-20260221-070000-test",
		RunID:     "20260221-0700000000-12345-1",
		Body:      body,
	}
}

// TestBusReadProjectLevel verifies project-level bus read (no --task flag).
func TestBusReadProjectLevel(t *testing.T) {
	msgs := []busMessage{
		makeBusMsg("RUN_START", "run started"),
		makeBusMsg("PROGRESS", "working on it"),
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/projects/proj/messages" {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(busMessagesResponse{Messages: msgs})
			return
		}
		t.Errorf("unexpected request: %s", r.URL.Path)
		http.NotFound(w, r)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	err := conductorBusRead(&buf, srv.URL, "proj", "", 0, false, false)
	if err != nil {
		t.Fatalf("conductorBusRead project-level: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "RUN_START") {
		t.Errorf("expected RUN_START in output, got:\n%s", output)
	}
	if !strings.Contains(output, "run started") {
		t.Errorf("expected 'run started' in output, got:\n%s", output)
	}
	if !strings.Contains(output, "PROGRESS") {
		t.Errorf("expected PROGRESS in output, got:\n%s", output)
	}
}

// TestBusReadTaskLevel verifies task-level bus read (with --task flag).
func TestBusReadTaskLevel(t *testing.T) {
	taskID := "task-20260221-070000-test"
	msgs := []busMessage{
		makeBusMsg("FACT", "build passed"),
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		expected := "/api/projects/proj/tasks/" + taskID + "/messages"
		if r.URL.Path == expected {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(busMessagesResponse{Messages: msgs})
			return
		}
		t.Errorf("unexpected request: %s (expected %s)", r.URL.Path, expected)
		http.NotFound(w, r)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	err := conductorBusRead(&buf, srv.URL, "proj", taskID, 0, false, false)
	if err != nil {
		t.Fatalf("conductorBusRead task-level: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "FACT") {
		t.Errorf("expected FACT in output, got:\n%s", output)
	}
	if !strings.Contains(output, "build passed") {
		t.Errorf("expected 'build passed' in output, got:\n%s", output)
	}
}

// TestBusReadTail verifies that --tail limits to last N messages (client-side).
func TestBusReadTail(t *testing.T) {
	msgs := []busMessage{
		makeBusMsg("RUN_START", "msg1"),
		makeBusMsg("PROGRESS", "msg2"),
		makeBusMsg("PROGRESS", "msg3"),
		makeBusMsg("FACT", "msg4"),
		makeBusMsg("RUN_STOP", "msg5"),
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(busMessagesResponse{Messages: msgs})
	}))
	defer srv.Close()

	var buf bytes.Buffer
	err := conductorBusRead(&buf, srv.URL, "proj", "", 3, false, false)
	if err != nil {
		t.Fatalf("conductorBusRead tail: %v", err)
	}

	output := buf.String()
	// First two messages should be excluded.
	if strings.Contains(output, "msg1") || strings.Contains(output, "msg2") {
		t.Errorf("early messages should be excluded with --tail 3, got:\n%s", output)
	}
	// Last three should be present.
	for _, want := range []string{"msg3", "msg4", "msg5"} {
		if !strings.Contains(output, want) {
			t.Errorf("expected %q in tail output, got:\n%s", want, output)
		}
	}
}

// TestBusReadEmptyBus verifies that empty bus prints "(no messages)".
func TestBusReadEmptyBus(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(busMessagesResponse{Messages: []busMessage{}})
	}))
	defer srv.Close()

	var buf bytes.Buffer
	err := conductorBusRead(&buf, srv.URL, "proj", "", 0, false, false)
	if err != nil {
		t.Fatalf("conductorBusRead empty: %v", err)
	}

	if !strings.Contains(buf.String(), "no messages") {
		t.Errorf("expected 'no messages' for empty bus, got:\n%s", buf.String())
	}
}

// TestBusReadJSONOutput verifies that --json outputs a valid JSON array.
func TestBusReadJSONOutput(t *testing.T) {
	msgs := []busMessage{
		makeBusMsg("RUN_START", "run started"),
		makeBusMsg("PROGRESS", "working"),
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(busMessagesResponse{Messages: msgs})
	}))
	defer srv.Close()

	var buf bytes.Buffer
	err := conductorBusRead(&buf, srv.URL, "proj", "", 0, false, true)
	if err != nil {
		t.Fatalf("conductorBusRead JSON output: %v", err)
	}

	// Verify it's valid JSON array.
	var result []busMessage
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("expected valid JSON array, got parse error: %v\nOutput:\n%s", err, buf.String())
	}
	if len(result) != 2 {
		t.Errorf("expected 2 messages in JSON output, got %d", len(result))
	}
	if result[0].Type != "RUN_START" {
		t.Errorf("expected first message type RUN_START, got %s", result[0].Type)
	}
}

// TestBusReadHTTP404 verifies that a 404 from the server returns an error.
func TestBusReadHTTP404(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, `{"error":"not found"}`, http.StatusNotFound)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	err := conductorBusRead(&buf, srv.URL, "proj", "", 0, false, false)
	if err == nil {
		t.Fatal("expected error for 404, got nil")
	}
	if !strings.Contains(err.Error(), "not found") {
		t.Errorf("expected 'not found' in error, got: %v", err)
	}
}

// TestBusReadHTTP500 verifies that a 500 from the server returns an error.
func TestBusReadHTTP500(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "internal server error", http.StatusInternalServerError)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	err := conductorBusRead(&buf, srv.URL, "proj", "", 0, false, false)
	if err == nil {
		t.Fatal("expected error for 500, got nil")
	}
	if !strings.Contains(err.Error(), "500") {
		t.Errorf("expected '500' in error, got: %v", err)
	}
}

// TestBusReadFollowSSE verifies that --follow streams messages then terminates on "event: done".
func TestBusReadFollowSSE(t *testing.T) {
	initialMsgs := []busMessage{
		makeBusMsg("RUN_START", "run started"),
	}
	streamMsgs := []busMessage{
		makeBusMsg("PROGRESS", "streaming progress"),
		makeBusMsg("FACT", "streaming fact"),
	}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/stream") {
			w.Header().Set("Content-Type", "text/event-stream")
			w.WriteHeader(http.StatusOK)
			fmt.Fprint(w, busSSEBody(streamMsgs))
			return
		}
		// Non-stream: return initial messages.
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(busMessagesResponse{Messages: initialMsgs})
	}))
	defer srv.Close()

	var buf bytes.Buffer
	err := conductorBusRead(&buf, srv.URL, "proj", "", 0, true, false)
	if err != nil {
		t.Fatalf("conductorBusRead follow: %v", err)
	}

	output := buf.String()
	// Should contain initial messages.
	if !strings.Contains(output, "RUN_START") {
		t.Errorf("expected RUN_START in output, got:\n%s", output)
	}
	// Should contain streamed messages.
	if !strings.Contains(output, "streaming progress") {
		t.Errorf("expected 'streaming progress' in output, got:\n%s", output)
	}
	if !strings.Contains(output, "streaming fact") {
		t.Errorf("expected 'streaming fact' in output, got:\n%s", output)
	}
}

// TestBusReadHeartbeatIgnored verifies that SSE heartbeat events are not printed.
func TestBusReadHeartbeatIgnored(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.HasSuffix(r.URL.Path, "/stream") {
			w.Header().Set("Content-Type", "text/event-stream")
			w.WriteHeader(http.StatusOK)
			// Heartbeat then a real message then done.
			fmt.Fprint(w, "event: heartbeat\ndata: {}\n\n")
			msg := makeBusMsg("DECISION", "chose option A")
			data, _ := json.Marshal(msg)
			fmt.Fprintf(w, "data: %s\n\n", data)
			fmt.Fprint(w, "event: done\ndata: {}\n\n")
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(busMessagesResponse{Messages: []busMessage{}})
	}))
	defer srv.Close()

	var buf bytes.Buffer
	err := conductorBusRead(&buf, srv.URL, "proj", "", 0, true, false)
	if err != nil {
		t.Fatalf("conductorBusRead heartbeat: %v", err)
	}

	output := buf.String()
	if strings.Contains(output, "{}") {
		t.Errorf("heartbeat data should not appear in output, got:\n%s", output)
	}
	if !strings.Contains(output, "chose option A") {
		t.Errorf("expected 'chose option A' in output, got:\n%s", output)
	}
}

// TestBusReadMessageFormat verifies the formatted text output format.
func TestBusReadMessageFormat(t *testing.T) {
	msg := busMessage{
		MsgID:     "MSG-001",
		Timestamp: time.Date(2026, 2, 21, 7, 30, 45, 0, time.UTC),
		Type:      "RUN_START",
		Body:      "run started",
	}

	formatted := formatBusMessage(msg)
	if !strings.Contains(formatted, "[2026-02-21 07:30:45]") {
		t.Errorf("expected timestamp in formatted output, got: %s", formatted)
	}
	if !strings.Contains(formatted, "RUN_START") {
		t.Errorf("expected type in formatted output, got: %s", formatted)
	}
	if !strings.Contains(formatted, "run started") {
		t.Errorf("expected body in formatted output, got: %s", formatted)
	}
}

// TestBusReadMessageFormatMultilineTruncated verifies multiline bodies are truncated.
func TestBusReadMessageFormatMultilineTruncated(t *testing.T) {
	msg := busMessage{
		MsgID:     "MSG-002",
		Timestamp: time.Date(2026, 2, 21, 7, 30, 45, 0, time.UTC),
		Type:      "FACT",
		Body:      "first line\nsecond line\nthird line",
	}

	formatted := formatBusMessage(msg)
	if !strings.Contains(formatted, "first line...") {
		t.Errorf("expected truncated multiline body, got: %s", formatted)
	}
	if strings.Contains(formatted, "second line") {
		t.Errorf("expected second line to be truncated, got: %s", formatted)
	}
}

// TestBusPostSuccess verifies posting a message to the project bus returns msg_id.
func TestBusPostSuccess(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("expected POST, got %s", r.Method)
		}
		if r.URL.Path != "/api/projects/proj/messages" {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		var req busPostRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			t.Errorf("decode request: %v", err)
		}
		if req.Type != "PROGRESS" {
			t.Errorf("expected type PROGRESS, got %s", req.Type)
		}
		if req.Body != "Build started" {
			t.Errorf("expected body 'Build started', got %s", req.Body)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(busPostResponse{MsgID: "MSG-20260221-110000-abc123"})
	}))
	defer srv.Close()

	var buf bytes.Buffer
	err := conductorBusPost(&buf, srv.URL, "proj", "", "PROGRESS", "Build started")
	if err != nil {
		t.Fatalf("conductorBusPost: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "msg_id: MSG-20260221-110000-abc123") {
		t.Errorf("expected msg_id in output, got:\n%s", output)
	}
}

// TestBusPostWithTask verifies posting to a task-level bus uses the correct URL.
func TestBusPostWithTask(t *testing.T) {
	taskID := "task-20260221-120000-feat"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		expected := "/api/projects/proj/tasks/" + taskID + "/messages"
		if r.URL.Path != expected {
			t.Errorf("expected path %s, got %s", expected, r.URL.Path)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(busPostResponse{MsgID: "MSG-20260221-120000-task01"})
	}))
	defer srv.Close()

	var buf bytes.Buffer
	err := conductorBusPost(&buf, srv.URL, "proj", taskID, "FACT", "Tests passed")
	if err != nil {
		t.Fatalf("conductorBusPost with task: %v", err)
	}

	if !strings.Contains(buf.String(), "MSG-20260221-120000-task01") {
		t.Errorf("expected task msg_id in output, got:\n%s", buf.String())
	}
}

// TestBusPostFromStdin verifies that body is read from stdin when not provided via --body.
func TestBusPostFromStdin(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req busPostRequest
		json.NewDecoder(r.Body).Decode(&req) //nolint:errcheck
		if req.Body != "Deployment complete" {
			t.Errorf("expected body 'Deployment complete', got %q", req.Body)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusCreated)
		json.NewEncoder(w).Encode(busPostResponse{MsgID: "MSG-stdin-001"})
	}))
	defer srv.Close()

	// conductorBusPost itself takes body as parameter; simulate stdin read by passing body directly.
	var buf bytes.Buffer
	err := conductorBusPost(&buf, srv.URL, "proj", "", "FACT", "Deployment complete")
	if err != nil {
		t.Fatalf("conductorBusPost stdin simulation: %v", err)
	}

	if !strings.Contains(buf.String(), "MSG-stdin-001") {
		t.Errorf("expected msg_id in output, got:\n%s", buf.String())
	}
}

// TestBusPostServerError verifies that a 500 response propagates as an error.
func TestBusPostServerError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "internal server error", http.StatusInternalServerError)
	}))
	defer srv.Close()

	var buf bytes.Buffer
	err := conductorBusPost(&buf, srv.URL, "proj", "", "INFO", "hello")
	if err == nil {
		t.Fatal("expected error for 500, got nil")
	}
	if !strings.Contains(err.Error(), "500") {
		t.Errorf("expected '500' in error, got: %v", err)
	}
}

// TestBusPostMissingProject verifies that missing --project returns an error.
func TestBusPostMissingProject(t *testing.T) {
	cmd := newRootCmd()
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	cmd.SetArgs([]string{"bus", "post", "--body", "hello"})
	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error when --project is missing, got nil")
	}
}

// TestBusPostAppearsInBusHelp verifies the post subcommand is listed in bus help.
func TestBusPostAppearsInBusHelp(t *testing.T) {
	cmd := newRootCmd()
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	cmd.SetArgs([]string{"bus", "--help"})
	_ = cmd.Execute()

	output := buf.String()
	if !strings.Contains(output, "post") {
		t.Errorf("expected 'post' in bus --help output, got:\n%s", output)
	}
}

// TestBusCmdHelp verifies that the bus subcommand is registered and shows help.
func TestBusCmdHelp(t *testing.T) {
	cmd := newRootCmd()
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	cmd.SetArgs([]string{"bus", "--help"})
	_ = cmd.Execute()

	output := buf.String()
	if !strings.Contains(output, "read") {
		t.Errorf("expected 'read' in bus --help output, got:\n%s", output)
	}
}

// TestBusReadCmdHelp verifies that bus read shows correct usage.
func TestBusReadCmdHelp(t *testing.T) {
	cmd := newRootCmd()
	var buf bytes.Buffer
	cmd.SetOut(&buf)
	cmd.SetErr(&buf)
	cmd.SetArgs([]string{"bus", "read", "--help"})
	_ = cmd.Execute()

	output := buf.String()
	for _, want := range []string{"project", "task", "tail", "follow", "json"} {
		if !strings.Contains(output, want) {
			t.Errorf("expected %q in bus read --help output, got:\n%s", want, output)
		}
	}
}
