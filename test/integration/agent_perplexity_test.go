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
		if r.URL.Path != "/chat/completions" {
			captureError(&mu, &capturedErr, "unexpected path: %s", r.URL.Path)
			w.WriteHeader(http.StatusNotFound)
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
		APIEndpoint: server.URL + "/chat/completions",
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

func TestPerplexityExecutionUsesRunContextToken(t *testing.T) {
	t.Setenv("PERPLEXITY_API_KEY", "token-from-process-env")

	var (
		mu      sync.Mutex
		gotAuth string
	)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		gotAuth = r.Header.Get("Authorization")
		mu.Unlock()

		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		_, _ = io.WriteString(w, "data: {\"choices\":[{\"delta\":{\"content\":\"ok\"}}]}\n\n")
		_, _ = io.WriteString(w, "data: [DONE]\n\n")
	}))
	defer server.Close()

	root := t.TempDir()
	runCtx := &agent.RunContext{
		Prompt:     "Say hello",
		StdoutPath: filepath.Join(root, "agent-stdout.txt"),
		StderrPath: filepath.Join(root, "agent-stderr.txt"),
		Environment: map[string]string{
			"PERPLEXITY_API_KEY": "token-from-run-context",
		},
	}

	agentImpl := perplexity.NewPerplexityAgent(perplexity.Options{
		Token:       "token-from-options",
		APIEndpoint: server.URL,
		HTTPClient:  server.Client(),
	})
	if err := agentImpl.Execute(context.Background(), runCtx); err != nil {
		t.Fatalf("Execute: %v", err)
	}

	mu.Lock()
	auth := gotAuth
	mu.Unlock()
	if auth != "Bearer token-from-run-context" {
		t.Fatalf("unexpected authorization header: %q", auth)
	}
}

func TestPerplexityExecutionSourcesFallbackToSearchResults(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		events := []string{
			`{"choices":[{"delta":{"content":"Hello world"}}],"search_results":[{"url":"https://search.example/doc"}]}`,
			`{"choices":[{"finish_reason":"stop"}]}`,
		}
		for _, event := range events {
			_, _ = io.WriteString(w, "data: "+event+"\n\n")
		}
		_, _ = io.WriteString(w, "data: [DONE]\n\n")
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

	stdoutBytes, err := os.ReadFile(runCtx.StdoutPath)
	if err != nil {
		t.Fatalf("read stdout: %v", err)
	}
	output := string(stdoutBytes)
	if !strings.Contains(output, "Hello world") {
		t.Fatalf("stdout missing content: %q", output)
	}
	if !strings.Contains(output, "Sources:") {
		t.Fatalf("stdout missing sources section: %q", output)
	}
	if !strings.Contains(output, "https://search.example/doc") {
		t.Fatalf("stdout missing search result source: %q", output)
	}
}

func TestPerplexityExecutionStatusErrorPropagation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, "invalid token", http.StatusUnauthorized)
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
	err := agentImpl.Execute(context.Background(), runCtx)
	if err == nil {
		t.Fatalf("expected Execute to fail")
	}
	if !strings.Contains(err.Error(), "status 401") {
		t.Fatalf("expected status code in error, got %v", err)
	}
	if !strings.Contains(err.Error(), "invalid token") {
		t.Fatalf("expected response body in error, got %v", err)
	}
}

func TestPerplexityExecutionStreamErrorPropagation(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		_, _ = io.WriteString(w, "data: {\"error\":{\"message\":\"stream exploded\"}}\n\n")
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
	err := agentImpl.Execute(context.Background(), runCtx)
	if err == nil {
		t.Fatalf("expected Execute to fail")
	}
	if !strings.Contains(err.Error(), "stream exploded") {
		t.Fatalf("unexpected error: %v", err)
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
