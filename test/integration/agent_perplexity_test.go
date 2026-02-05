package integration_test

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	"github.com/jonnyzzz/conductor-loop/internal/agent"
	"github.com/jonnyzzz/conductor-loop/internal/agent/perplexity"
)

type perplexityRequest struct {
	Model    string `json:"model"`
	Messages []struct {
		Role    string `json:"role"`
		Content string `json:"content"`
	} `json:"messages"`
	Stream bool `json:"stream"`
}

func TestPerplexityExecution(t *testing.T) {
	var mu sync.Mutex
	var capturedErr error
	var capturedReq perplexityRequest
	var requestCount int

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		requestCount++
		mu.Unlock()

		if r.Method != http.MethodPost {
			captureError(&mu, &capturedErr, "unexpected method: %s", r.Method)
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		if auth := r.Header.Get("Authorization"); auth != "Bearer test-token" {
			captureError(&mu, &capturedErr, "unexpected authorization header: %q", auth)
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		if accept := r.Header.Get("Accept"); accept != "text/event-stream" {
			captureError(&mu, &capturedErr, "unexpected accept header: %q", accept)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		if contentType := r.Header.Get("Content-Type"); contentType != "application/json" {
			captureError(&mu, &capturedErr, "unexpected content-type header: %q", contentType)
			w.WriteHeader(http.StatusBadRequest)
			return
		}

		defer r.Body.Close()
		body, err := io.ReadAll(r.Body)
		if err != nil {
			captureError(&mu, &capturedErr, "read body: %v", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		var req perplexityRequest
		if err := json.Unmarshal(body, &req); err != nil {
			captureError(&mu, &capturedErr, "decode body: %v", err)
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		mu.Lock()
		capturedReq = req
		mu.Unlock()

		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		flusher, ok := w.(http.Flusher)
		if !ok {
			captureError(&mu, &capturedErr, "missing flusher")
			return
		}

		events := []string{
			`{"choices":[{"delta":{"content":"Hello"}}]}`,
			`{"choices":[{"delta":{"content":" world"}}],"citations":["https://example.com"]}`,
		}
		for _, event := range events {
			_, _ = io.WriteString(w, "data: "+event+"\n\n")
			flusher.Flush()
		}
		_, _ = io.WriteString(w, "data: [DONE]\n\n")
		flusher.Flush()
	}))
	defer server.Close()

	root := t.TempDir()
	runCtx := &agent.RunContext{
		Prompt:     "Say hello",
		StdoutPath: filepath.Join(root, "agent-stdout.txt"),
		StderrPath: filepath.Join(root, "agent-stderr.txt"),
	}

	agentImpl := perplexity.NewPerplexityAgent(perplexity.Options{
		Token:       "test-token",
		APIEndpoint: server.URL,
		HTTPClient:  server.Client(),
	})

	if err := agentImpl.Execute(context.Background(), runCtx); err != nil {
		t.Fatalf("Execute: %v", err)
	}

	mu.Lock()
	err := capturedErr
	req := capturedReq
	count := requestCount
	mu.Unlock()

	if err != nil {
		t.Fatalf("server assertion failed: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected 1 request, got %d", count)
	}
	if req.Model != "sonar-reasoning" {
		t.Fatalf("unexpected model: %q", req.Model)
	}
	if !req.Stream {
		t.Fatalf("expected stream=true")
	}
	if len(req.Messages) != 1 {
		t.Fatalf("expected 1 message, got %d", len(req.Messages))
	}
	if req.Messages[0].Role != "user" {
		t.Fatalf("unexpected role: %q", req.Messages[0].Role)
	}
	if req.Messages[0].Content != runCtx.Prompt {
		t.Fatalf("unexpected content: %q", req.Messages[0].Content)
	}

	stdoutBytes, err := os.ReadFile(runCtx.StdoutPath)
	if err != nil {
		t.Fatalf("read stdout: %v", err)
	}
	output := string(stdoutBytes)
	if !strings.Contains(output, "Hello world") {
		t.Fatalf("stdout missing content: %q", output)
	}
	if !strings.Contains(output, "Sources:") {
		t.Fatalf("stdout missing sources: %q", output)
	}
	if !strings.Contains(output, "https://example.com") {
		t.Fatalf("stdout missing citation: %q", output)
	}
}

func captureError(mu *sync.Mutex, target *error, format string, args ...interface{}) {
	mu.Lock()
	defer mu.Unlock()
	if *target != nil {
		return
	}
	*target = fmt.Errorf(format, args...)
}
