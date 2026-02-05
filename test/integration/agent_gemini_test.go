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
	"testing"

	"github.com/jonnyzzz/conductor-loop/internal/agent"
	"github.com/jonnyzzz/conductor-loop/internal/agent/gemini"
)

func TestGeminiExecution(t *testing.T) {
	prompt := "hello gemini"
	token := "test-token"

	var (
		gotPath   string
		gotMethod string
		gotToken  string
		gotAccept string
		gotAlt    string
		gotPrompt string
	)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotPath = r.URL.Path
		gotMethod = r.Method
		gotToken = r.Header.Get("x-goog-api-key")
		gotAccept = r.Header.Get("Accept")
		gotAlt = r.URL.Query().Get("alt")

		body, err := io.ReadAll(r.Body)
		if err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		var req struct {
			Contents []struct {
				Parts []struct {
					Text string `json:"text"`
				} `json:"parts"`
			} `json:"contents"`
		}
		if err := json.Unmarshal(body, &req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}
		if len(req.Contents) > 0 && len(req.Contents[0].Parts) > 0 {
			gotPrompt = req.Contents[0].Parts[0].Text
		}

		w.Header().Set("Content-Type", "text/event-stream")
		flusher, _ := w.(http.Flusher)
		fmt.Fprint(w, "data: {\"candidates\":[{\"content\":{\"parts\":[{\"text\":\"Hello \"}]}}]}\n\n")
		if flusher != nil {
			flusher.Flush()
		}
		fmt.Fprint(w, "data: {\"candidates\":[{\"content\":{\"parts\":[{\"text\":\"world\"}]}}]}\n\n")
		if flusher != nil {
			flusher.Flush()
		}
		fmt.Fprint(w, "data: [DONE]\n\n")
	}))
	defer server.Close()

	root := t.TempDir()
	runCtx := &agent.RunContext{
		Prompt:     prompt,
		StdoutPath: filepath.Join(root, "agent-stdout.txt"),
		StderrPath: filepath.Join(root, "agent-stderr.txt"),
		Environment: map[string]string{
			"GEMINI_API_KEY": token,
		},
	}

	gem := &gemini.GeminiAgent{
		BaseURL: server.URL,
		Model:   "gemini-1.5-pro",
	}

	if err := gem.Execute(context.Background(), runCtx); err != nil {
		t.Fatalf("Execute: %v", err)
	}

	if gotMethod != http.MethodPost {
		t.Fatalf("unexpected method: %s", gotMethod)
	}
	if gotPath != "/v1beta/models/gemini-1.5-pro:streamGenerateContent" {
		t.Fatalf("unexpected path: %s", gotPath)
	}
	if gotToken != token {
		t.Fatalf("unexpected token: %s", gotToken)
	}
	if gotAccept != "text/event-stream" {
		t.Fatalf("unexpected Accept header: %s", gotAccept)
	}
	if gotAlt != "sse" {
		t.Fatalf("unexpected alt query: %s", gotAlt)
	}
	if gotPrompt != prompt {
		t.Fatalf("unexpected prompt: %s", gotPrompt)
	}

	stdoutBytes, err := os.ReadFile(runCtx.StdoutPath)
	if err != nil {
		t.Fatalf("read stdout: %v", err)
	}
	if got := string(stdoutBytes); got != "Hello world" {
		t.Fatalf("unexpected stdout: %q", got)
	}

	stderrBytes, err := os.ReadFile(runCtx.StderrPath)
	if err != nil {
		t.Fatalf("read stderr: %v", err)
	}
	if trimmed := strings.TrimSpace(string(stderrBytes)); trimmed != "" {
		t.Fatalf("unexpected stderr: %q", trimmed)
	}
}
