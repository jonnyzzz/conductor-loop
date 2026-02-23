package main

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestServerUpdateStatusCmd(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/api/v1/admin/self-update" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"state":"idle","active_runs_now":0}`))
	}))
	defer ts.Close()

	cmd := newServerUpdateStatusCmd()
	cmd.SetArgs([]string{"--server", ts.URL})

	var runErr error
	output := captureStdout(t, func() {
		runErr = cmd.Execute()
	})
	if runErr != nil {
		t.Fatalf("Execute: %v", runErr)
	}
	if !strings.Contains(output, "State:") {
		t.Fatalf("unexpected output: %q", output)
	}
	if !strings.Contains(output, "idle") {
		t.Fatalf("expected idle state in output: %q", output)
	}
}

func TestServerUpdateStartCmd(t *testing.T) {
	wantBinary := "/tmp/new-run-agent"
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/api/v1/admin/self-update" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		body, _ := io.ReadAll(r.Body)
		var req map[string]string
		if err := json.Unmarshal(body, &req); err != nil {
			t.Fatalf("decode request: %v", err)
		}
		if req["binary_path"] != wantBinary {
			t.Fatalf("binary_path = %q, want %q", req["binary_path"], wantBinary)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusAccepted)
		_, _ = w.Write([]byte(`{"state":"deferred","active_runs_now":2}`))
	}))
	defer ts.Close()

	cmd := newServerUpdateStartCmd()
	cmd.SetArgs([]string{"--server", ts.URL, "--binary", wantBinary})

	var runErr error
	output := captureStdout(t, func() {
		runErr = cmd.Execute()
	})
	if runErr != nil {
		t.Fatalf("Execute: %v", runErr)
	}
	if !strings.Contains(output, "Self-update state: deferred") {
		t.Fatalf("unexpected output: %q", output)
	}
}
