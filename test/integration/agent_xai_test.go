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
	"time"

	"github.com/jonnyzzz/conductor-loop/internal/agent"
	"github.com/jonnyzzz/conductor-loop/internal/agent/xai"
)

func TestXAIExecution(t *testing.T) {
	t.Parallel()

	const (
		token = "test-token"
		model = "grok-4"
	)

	var (
		reqErr error
		mu     sync.Mutex
	)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		setErr := func(err error) {
			mu.Lock()
			defer mu.Unlock()
			if reqErr == nil {
				reqErr = err
			}
		}

		if r.Method != http.MethodPost {
			setErr(fmt.Errorf("expected POST, got %s", r.Method))
		}
		if r.URL.Path != "/v1/chat/completions" {
			setErr(fmt.Errorf("unexpected path: %s", r.URL.Path))
		}
		if got := r.Header.Get("Authorization"); got != "Bearer "+token {
			setErr(fmt.Errorf("authorization header mismatch: %q", got))
		}
		if ct := r.Header.Get("Content-Type"); !strings.Contains(ct, "application/json") {
			setErr(fmt.Errorf("content-type missing: %q", ct))
		}
		if accept := r.Header.Get("Accept"); !strings.Contains(accept, "text/event-stream") {
			setErr(fmt.Errorf("accept header missing: %q", accept))
		}

		body, err := io.ReadAll(r.Body)
		if err != nil {
			setErr(fmt.Errorf("read body: %w", err))
		}
		var req xaiRequest
		if err := json.Unmarshal(body, &req); err != nil {
			setErr(fmt.Errorf("decode body: %w", err))
		}
		if req.Model != model {
			setErr(fmt.Errorf("model mismatch: %q", req.Model))
		}
		if !req.Stream {
			setErr(fmt.Errorf("stream not enabled"))
		}
		if len(req.Messages) != 1 || req.Messages[0].Role != "user" || req.Messages[0].Content != "Hello" {
			setErr(fmt.Errorf("messages mismatch: %#v", req.Messages))
		}

		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(http.StatusOK)
		flusher, ok := w.(http.Flusher)
		if !ok {
			setErr(fmt.Errorf("response does not support flushing"))
			return
		}

		chunks := []string{
			`{"choices":[{"delta":{"content":"Hello"}}]}`,
			`{"choices":[{"delta":{"content":" world"}}]}`,
		}
		for _, chunk := range chunks {
			_, _ = fmt.Fprintf(w, "data: %s\n\n", chunk)
			flusher.Flush()
		}
		_, _ = fmt.Fprint(w, "data: [DONE]\n\n")
		flusher.Flush()
	}))
	defer server.Close()

	agentImpl, err := xai.NewAgent(xai.Config{
		APIKey:     token,
		BaseURL:    server.URL,
		Model:      model,
		HTTPClient: server.Client(),
	})
	if err != nil {
		t.Fatalf("NewAgent: %v", err)
	}

	root := t.TempDir()
	stdoutPath := filepath.Join(root, "agent-stdout.txt")
	stderrPath := filepath.Join(root, "agent-stderr.txt")
	runCtx := &agent.RunContext{
		RunID:      "run-123",
		ProjectID:  "project",
		TaskID:     "task",
		Prompt:     "Hello",
		WorkingDir: root,
		StdoutPath: stdoutPath,
		StderrPath: stderrPath,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	if err := agentImpl.Execute(ctx, runCtx); err != nil {
		t.Fatalf("Execute: %v", err)
	}

	mu.Lock()
	defer mu.Unlock()
	if reqErr != nil {
		t.Fatalf("request validation failed: %v", reqErr)
	}

	stdoutBytes, err := os.ReadFile(stdoutPath)
	if err != nil {
		t.Fatalf("read stdout: %v", err)
	}
	if string(stdoutBytes) != "Hello world" {
		t.Fatalf("stdout mismatch: %q", string(stdoutBytes))
	}
}

type xaiRequest struct {
	Model    string       `json:"model"`
	Messages []xaiMessage `json:"messages"`
	Stream   bool         `json:"stream"`
}

type xaiMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}
