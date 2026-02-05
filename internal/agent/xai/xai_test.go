package xai

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"

	agentpkg "github.com/jonnyzzz/conductor-loop/internal/agent"
)

func TestResolveEndpoint(t *testing.T) {
	endpoint, err := resolveEndpoint("https://api.x.ai")
	if err != nil {
		t.Fatalf("resolveEndpoint: %v", err)
	}
	if !strings.Contains(endpoint, "/v1/chat/completions") {
		t.Fatalf("unexpected endpoint: %s", endpoint)
	}
	endpoint, err = resolveEndpoint("https://api.x.ai/v1")
	if err != nil {
		t.Fatalf("resolveEndpoint: %v", err)
	}
	if !strings.HasSuffix(endpoint, "/v1/chat/completions") {
		t.Fatalf("unexpected endpoint: %s", endpoint)
	}
	endpoint, err = resolveEndpoint("https://api.x.ai/v1/chat/completions")
	if err != nil {
		t.Fatalf("resolveEndpoint: %v", err)
	}
	if !strings.HasSuffix(endpoint, "/v1/chat/completions") {
		t.Fatalf("unexpected endpoint: %s", endpoint)
	}
	if _, err := resolveEndpoint("://bad"); err == nil {
		t.Fatalf("expected error for invalid url")
	}
}

func TestLookupEnv(t *testing.T) {
	if got := lookupEnv(map[string]string{"XAI_API_KEY": "token"}, envAPIKey); got != "token" {
		t.Fatalf("expected token, got %q", got)
	}
	t.Setenv(envAPIKey, "env-token")
	if got := lookupEnv(nil, envAPIKey); got != "env-token" {
		t.Fatalf("expected env token, got %q", got)
	}
}

func TestWriteChunk(t *testing.T) {
	var stdout bytes.Buffer
	payload := `{"choices":[{"delta":{"content":"hi"}}]}`
	if err := writeChunk(payload, &stdout); err != nil {
		t.Fatalf("writeChunk: %v", err)
	}
	if stdout.String() != "hi" {
		t.Fatalf("unexpected stdout: %q", stdout.String())
	}
	if err := writeChunk(`{"error":{"message":"boom"}}`, &stdout); err == nil {
		t.Fatalf("expected error")
	}
	if err := writeChunk("{bad", &stdout); err == nil {
		t.Fatalf("expected decode error")
	}
}

func TestDecodeSingleResponse(t *testing.T) {
	var stdout bytes.Buffer
	payload := `{"choices":[{"message":{"content":"hello"}}]}`
	if err := decodeSingleResponse(strings.NewReader(payload), &stdout); err != nil {
		t.Fatalf("decodeSingleResponse: %v", err)
	}
	if stdout.String() != "hello" {
		t.Fatalf("unexpected stdout: %q", stdout.String())
	}
	if err := decodeSingleResponse(strings.NewReader(`{"choices":[]}`), &stdout); err == nil {
		t.Fatalf("expected error for missing content")
	}
	if err := decodeSingleResponse(strings.NewReader(`{"error":{"message":"boom"}}`), &stdout); err == nil {
		t.Fatalf("expected error for api error")
	}
	if err := decodeSingleResponse(strings.NewReader("{bad"), &stdout); err == nil {
		t.Fatalf("expected decode error")
	}
}

func TestStreamResponse(t *testing.T) {
	data := "data: {\"choices\":[{\"delta\":{\"content\":\"ok\"}}]}\n" +
		"data: [DONE]\n"
	var stdout bytes.Buffer
	if err := streamResponse(strings.NewReader(data), &stdout); err != nil {
		t.Fatalf("streamResponse: %v", err)
	}
	if stdout.String() != "ok" {
		t.Fatalf("unexpected stdout: %q", stdout.String())
	}
}

func TestStreamResponseError(t *testing.T) {
	data := "data: {bad}\n"
	if err := streamResponse(strings.NewReader(data), io.Discard); err == nil {
		t.Fatalf("expected stream error")
	}
}

func TestResolvedConfig(t *testing.T) {
	agent, err := NewAgent(Config{APIKey: "token", BaseURL: "https://api.x.ai", Model: "model", HTTPClient: &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader("data: [DONE]\n")), Header: http.Header{"Content-Type": []string{"text/event-stream"}}}, nil
	})}})
	if err != nil {
		t.Fatalf("NewAgent: %v", err)
	}
	resolved, err := agent.resolveConfig(map[string]string{})
	if err != nil {
		t.Fatalf("resolveConfig: %v", err)
	}
	if resolved.apiKey != "token" {
		t.Fatalf("expected api key")
	}
	if err := resolved.streamCompletion(context.Background(), "prompt", io.Discard); err != nil {
		t.Fatalf("streamCompletion: %v", err)
	}
}

func TestResolveConfigMissingKey(t *testing.T) {
	agent, err := NewAgent(Config{})
	if err != nil {
		t.Fatalf("NewAgent: %v", err)
	}
	if _, err := agent.resolveConfig(map[string]string{}); err == nil {
		t.Fatalf("expected error for missing api key")
	}
}

func TestStreamCompletionNonStream(t *testing.T) {
	client := &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		body := "{\"choices\":[{\"message\":{\"content\":\"hello\"}}]}"
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader(body)),
			Header:     http.Header{"Content-Type": []string{"application/json"}},
		}, nil
	})}
	agent, err := NewAgent(Config{APIKey: "token", HTTPClient: client})
	if err != nil {
		t.Fatalf("NewAgent: %v", err)
	}
	resolved, err := agent.resolveConfig(map[string]string{})
	if err != nil {
		t.Fatalf("resolveConfig: %v", err)
	}
	var stdout bytes.Buffer
	if err := resolved.streamCompletion(context.Background(), "prompt", &stdout); err != nil {
		t.Fatalf("streamCompletion: %v", err)
	}
	if stdout.String() != "hello" {
		t.Fatalf("unexpected stdout: %q", stdout.String())
	}
}

func TestStreamCompletionStatusError(t *testing.T) {
	client := &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusInternalServerError,
			Body:       io.NopCloser(strings.NewReader("oops")),
			Header:     http.Header{"Content-Type": []string{"application/json"}},
		}, nil
	})}
	agent, err := NewAgent(Config{APIKey: "token", HTTPClient: client})
	if err != nil {
		t.Fatalf("NewAgent: %v", err)
	}
	resolved, err := agent.resolveConfig(map[string]string{})
	if err != nil {
		t.Fatalf("resolveConfig: %v", err)
	}
	if err := resolved.streamCompletion(context.Background(), "prompt", io.Discard); err == nil {
		t.Fatalf("expected status error")
	}
}

func TestNewAgentFromEnvironment(t *testing.T) {
	agent, err := NewAgentFromEnvironment(map[string]string{
		envAPIKey:      "token",
		envAPIEndpoint: "http://example",
		envModel:       "model",
	})
	if err != nil {
		t.Fatalf("NewAgentFromEnvironment: %v", err)
	}
	if agent.apiKey != "token" {
		t.Fatalf("expected api key")
	}
}

func TestAgentType(t *testing.T) {
	agent, err := NewAgent(Config{APIKey: "token"})
	if err != nil {
		t.Fatalf("NewAgent: %v", err)
	}
	if agent.Type() != TypeName {
		t.Fatalf("unexpected type: %q", agent.Type())
	}
}

func TestExecuteMissingAPIKey(t *testing.T) {
	agent := &Agent{}
	runCtx := &agentpkg.RunContext{
		Prompt:     "prompt",
		StdoutPath: filepath.Join(t.TempDir(), "stdout.txt"),
		StderrPath: filepath.Join(t.TempDir(), "stderr.txt"),
	}
	if err := agent.Execute(context.Background(), runCtx); err == nil {
		t.Fatalf("expected error for missing api key")
	}
}

func TestExecuteValidation(t *testing.T) {
	agent, err := NewAgent(Config{APIKey: "token"})
	if err != nil {
		t.Fatalf("NewAgent: %v", err)
	}
	if err := agent.Execute(context.Background(), nil); err == nil {
		t.Fatalf("expected error for nil run context")
	}
	runCtx := &agentpkg.RunContext{Prompt: "   "}
	if err := agent.Execute(context.Background(), runCtx); err == nil {
		t.Fatalf("expected error for empty prompt")
	}
}

func TestChunkContentHelpers(t *testing.T) {
	chunk := chatCompletionChunk{Choices: []chatChoice{{Delta: chatDelta{Content: "delta"}}}}
	if chunk.Content() != "delta" {
		t.Fatalf("unexpected chunk content")
	}
	chunk = chatCompletionChunk{Choices: []chatChoice{{Message: chatMessage{Content: "msg"}}}}
	if chunk.Content() != "msg" {
		t.Fatalf("unexpected message content")
	}
	resp := chatCompletionResponse{Choices: []chatChoice{{Text: "text"}}}
	if resp.Content() != "text" {
		t.Fatalf("unexpected response content")
	}
}

func TestExecuteSuccess(t *testing.T) {
	client := &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		body := "{\"choices\":[{\"message\":{\"content\":\"hello\"}}]}"
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader(body)),
			Header:     http.Header{"Content-Type": []string{"application/json"}},
		}, nil
	})}
	agent, err := NewAgent(Config{APIKey: "token", BaseURL: "http://example", HTTPClient: client})
	if err != nil {
		t.Fatalf("NewAgent: %v", err)
	}
	dir := t.TempDir()
	runCtx := &agentpkg.RunContext{
		Prompt:     "prompt",
		StdoutPath: filepath.Join(dir, "stdout.txt"),
		StderrPath: filepath.Join(dir, "stderr.txt"),
	}
	if err := agent.Execute(context.Background(), runCtx); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	data, err := os.ReadFile(runCtx.StdoutPath)
	if err != nil {
		t.Fatalf("read stdout: %v", err)
	}
	if string(data) != "hello" {
		t.Fatalf("unexpected stdout: %q", string(data))
	}
}

func TestResolveConfigEnvOverrides(t *testing.T) {
	agent, err := NewAgent(Config{APIKey: "token", BaseURL: "https://api.x.ai"})
	if err != nil {
		t.Fatalf("NewAgent: %v", err)
	}
	cfg, err := agent.resolveConfig(map[string]string{
		envAPIEndpoint: "http://example",
		envModel:       "override",
	})
	if err != nil {
		t.Fatalf("resolveConfig: %v", err)
	}
	if !strings.Contains(cfg.endpoint, "example") {
		t.Fatalf("expected endpoint override")
	}
	if cfg.model != "override" {
		t.Fatalf("expected model override")
	}
}

func TestRoundTripErrorPropagates(t *testing.T) {
	client := &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		return nil, errors.New("boom")
	})}
	agent, err := NewAgent(Config{APIKey: "token", HTTPClient: client})
	if err != nil {
		t.Fatalf("NewAgent: %v", err)
	}
	cfg, err := agent.resolveConfig(map[string]string{})
	if err != nil {
		t.Fatalf("resolveConfig: %v", err)
	}
	if err := cfg.streamCompletion(context.Background(), "prompt", io.Discard); err == nil {
		t.Fatalf("expected request error")
	}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }
