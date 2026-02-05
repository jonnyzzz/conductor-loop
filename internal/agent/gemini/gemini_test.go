package gemini

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"

	"github.com/jonnyzzz/conductor-loop/internal/agent"
)

func TestBuildEndpointValidation(t *testing.T) {
	if _, err := buildEndpoint("http://", "model"); err == nil {
		t.Fatalf("expected error for invalid base url")
	}
	endpoint, err := buildEndpoint("https://example.com/base", "model")
	if err != nil {
		t.Fatalf("buildEndpoint: %v", err)
	}
	if !strings.Contains(endpoint, "v1beta/models/model:streamGenerateContent") {
		t.Fatalf("unexpected endpoint: %s", endpoint)
	}
	if !strings.Contains(endpoint, "alt=sse") {
		t.Fatalf("expected alt=sse, got %s", endpoint)
	}
}

func TestExtractText(t *testing.T) {
	payload := `{"candidates":[{"content":{"parts":[{"text":"hello"}]}}]}`
	text, err := extractText(payload)
	if err != nil {
		t.Fatalf("extractText: %v", err)
	}
	if text != "hello" {
		t.Fatalf("unexpected text: %q", text)
	}
	bad := `{"error":{"message":"boom"}}`
	if _, err := extractText(bad); err == nil {
		t.Fatalf("expected error")
	}
}

func TestDiffText(t *testing.T) {
	if got := diffText("hel", "hello"); got != "lo" {
		t.Fatalf("unexpected diff: %q", got)
	}
	if got := diffText("hello", "hel"); got != "" {
		t.Fatalf("unexpected diff: %q", got)
	}
}

func TestReadStream(t *testing.T) {
	data := "data: {\"candidates\":[{\"content\":{\"parts\":[{\"text\":\"hi\"}]}}]}\n\n" +
		"data: [DONE]\n\n"
	var stdout bytes.Buffer
	if err := readStream(strings.NewReader(data), &stdout, io.Discard); err != nil {
		t.Fatalf("readStream: %v", err)
	}
	if stdout.String() != "hi" {
		t.Fatalf("unexpected stdout: %q", stdout.String())
	}
}

func TestReadStreamInvalidJSON(t *testing.T) {
	data := "data: {bad}\n\n"
	if err := readStream(strings.NewReader(data), io.Discard, io.Discard); err == nil {
		t.Fatalf("expected error")
	}
}

func TestStreamRequest(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Header.Get("x-goog-api-key") != "token" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		w.Header().Set("Content-Type", "text/event-stream")
		_, _ = w.Write([]byte("data: {\"candidates\":[{\"content\":{\"parts\":[{\"text\":\"ok\"}]}}]}\n\n"))
		_, _ = w.Write([]byte("data: [DONE]\n\n"))
	}))
	defer srv.Close()

	agentImpl := &GeminiAgent{HTTPClient: srv.Client()}
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	if err := agentImpl.stream(context.Background(), "token", srv.URL, "model", "prompt", &stdout, &stderr); err != nil {
		t.Fatalf("stream: %v", err)
	}
	if stdout.String() != "ok" {
		t.Fatalf("unexpected stdout: %q", stdout.String())
	}
}

func TestExecuteMissingToken(t *testing.T) {
	agentImpl := &GeminiAgent{}
	dir := t.TempDir()
	runCtx := &agent.RunContext{
		Prompt:     "hi",
		StdoutPath: filepath.Join(dir, "stdout.txt"),
		StderrPath: filepath.Join(dir, "stderr.txt"),
	}
	if err := agentImpl.Execute(context.Background(), runCtx); err == nil {
		t.Fatalf("expected error for missing token")
	}
}
