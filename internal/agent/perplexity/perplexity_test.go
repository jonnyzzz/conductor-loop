package perplexity

import (
	"bytes"
	"context"
	"errors"
	"io"
	"math/rand"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/jonnyzzz/conductor-loop/internal/agent"
)

type roundTripFunc func(*http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) { return f(r) }

func TestResolveToken(t *testing.T) {
	if got := resolveToken("", map[string]string{perplexityTokenEnv: "token"}); got != "token" {
		t.Fatalf("expected token, got %q", got)
	}
	if got := resolveToken("fallback", map[string]string{}); got != "fallback" {
		t.Fatalf("expected fallback token, got %q", got)
	}
}

func TestResolveTokenFromEnv(t *testing.T) {
	t.Setenv(perplexityTokenEnv, "env-token")
	if got := resolveToken("", nil); got != "env-token" {
		t.Fatalf("expected env token, got %q", got)
	}
}

func TestParseRetryAfter(t *testing.T) {
	if d, ok := parseRetryAfter("2"); !ok || d != 2*time.Second {
		t.Fatalf("expected 2s retry, got %v (%v)", d, ok)
	}
	if _, ok := parseRetryAfter("invalid"); ok {
		t.Fatalf("expected invalid retry-after")
	}
}

func TestParseRetryAfterHTTPDate(t *testing.T) {
	future := time.Now().Add(2 * time.Second).UTC().Format(http.TimeFormat)
	if d, ok := parseRetryAfter(future); !ok || d <= 0 {
		t.Fatalf("expected retry-after date, got %v (%v)", d, ok)
	}
	if _, ok := parseRetryAfter("-1"); ok {
		t.Fatalf("expected negative retry-after to be invalid")
	}
}

func TestRetryDelayWithHeader(t *testing.T) {
	resp := &http.Response{Header: http.Header{"Retry-After": []string{"1"}}}
	delay := retryDelay(resp, 0, nil)
	if delay != time.Second {
		t.Fatalf("expected 1s delay, got %v", delay)
	}
}

func TestAppendContent(t *testing.T) {
	if err := appendContent(nil, "hi", nil); err == nil {
		t.Fatalf("expected error for nil stdout")
	}
	var stdout bytes.Buffer
	last := "hello"
	if err := appendContent(&stdout, "hello world", &last); err != nil {
		t.Fatalf("appendContent: %v", err)
	}
	if stdout.String() != " world" {
		t.Fatalf("unexpected content: %q", stdout.String())
	}
}

func TestAppendCitations(t *testing.T) {
	var stdout bytes.Buffer
	err := appendCitations(&stdout, []string{"http://a", "http://a", "http://b"}, nil)
	if err != nil {
		t.Fatalf("appendCitations: %v", err)
	}
	out := stdout.String()
	if !strings.Contains(out, "Sources") || !strings.Contains(out, "http://a") {
		t.Fatalf("unexpected citations output: %q", out)
	}
}

func TestHandleEvent(t *testing.T) {
	payload := `{"choices":[{"delta":{"content":"hi"}}],"citations":["http://a"]}`
	state := &streamState{}
	var stdout bytes.Buffer
	done, err := handleEvent(payload, &stdout, state)
	if err != nil {
		t.Fatalf("handleEvent: %v", err)
	}
	if done {
		t.Fatalf("did not expect done")
	}
	if stdout.String() != "hi" {
		t.Fatalf("unexpected stdout: %q", stdout.String())
	}
	if len(state.citations) != 1 {
		t.Fatalf("expected citations")
	}
}

func TestHandleEventErrors(t *testing.T) {
	var stdout bytes.Buffer
	if _, err := handleEvent("{bad", &stdout, &streamState{}); err == nil {
		t.Fatalf("expected parse error")
	}
	if _, err := handleEvent(`{"error":{"message":""}}`, &stdout, &streamState{}); err == nil {
		t.Fatalf("expected stream error")
	}
	if _, err := handleEvent(`{"choices":[{"finish_reason":"error"}]}`, &stdout, &streamState{}); err == nil {
		t.Fatalf("expected finish_reason error")
	}
}

func TestConsumeStream(t *testing.T) {
	data := "data: {\"choices\":[{\"delta\":{\"content\":\"hello\"}}]}\n\n" +
		"data: [DONE]\n\n"
	var stdout bytes.Buffer
	ctx := context.Background()
	err := consumeStream(ctx, func() {}, io.NopCloser(strings.NewReader(data)), &stdout, 0, &streamState{})
	if err != nil {
		t.Fatalf("consumeStream: %v", err)
	}
	if stdout.String() != "hello" {
		t.Fatalf("unexpected stdout: %q", stdout.String())
	}
}

func TestConsumeStreamNilBody(t *testing.T) {
	if err := consumeStream(context.Background(), func() {}, nil, io.Discard, 0, &streamState{}); err == nil {
		t.Fatalf("expected error for nil body")
	}
}

func TestConsumeStreamScannerError(t *testing.T) {
	longLine := "data: " + strings.Repeat("a", maxScannerTokenSize+10) + "\n\n"
	err := consumeStream(context.Background(), func() {}, io.NopCloser(strings.NewReader(longLine)), io.Discard, 0, &streamState{})
	if err == nil {
		t.Fatalf("expected scanner error")
	}
}

func TestExecuteWithRetrySuccess(t *testing.T) {
	body := "data: {\"choices\":[{\"delta\":{\"content\":\"ok\"}}]}\n\n" +
		"data: [DONE]\n\n"
	client := &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader(body)),
			Header:     http.Header{},
		}, nil
	})}
	var stdout bytes.Buffer
	if err := executeWithRetry(context.Background(), client, "http://example", "token", []byte("{}"), &stdout, io.Discard, 0); err != nil {
		t.Fatalf("executeWithRetry: %v", err)
	}
	if stdout.String() != "ok" {
		t.Fatalf("unexpected stdout: %q", stdout.String())
	}
}

func TestStartIdleMonitor(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	activity, triggered := startIdleMonitor(ctx, cancel, 20*time.Millisecond)
	if activity == nil || triggered == nil {
		t.Fatalf("expected channels")
	}
	select {
	case <-triggered:
	case <-time.After(200 * time.Millisecond):
		t.Fatalf("expected idle trigger")
	}
}

func TestExecuteValidationErrors(t *testing.T) {
	if err := executeWithRetry(context.Background(), nil, "http://example", "token", []byte("{}"), io.Discard, io.Discard, 0); err == nil {
		t.Fatalf("expected error for nil client")
	}
	client := &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: http.StatusOK, Body: io.NopCloser(strings.NewReader("data: [DONE]\n\n"))}, nil
	})}
	if err := executeWithRetry(context.Background(), client, "", "token", []byte("{}"), io.Discard, io.Discard, 0); err == nil {
		t.Fatalf("expected error for empty endpoint")
	}
}

func TestPerplexityAgentExecute(t *testing.T) {
	body := "data: {\"choices\":[{\"delta\":{\"content\":\"ok\"}}]}\n\n" +
		"data: [DONE]\n\n"
	client := &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusOK,
			Body:       io.NopCloser(strings.NewReader(body)),
			Header:     http.Header{},
		}, nil
	})}
	dir := t.TempDir()
	agentImpl := NewPerplexityAgent(Options{Token: "token", HTTPClient: client, Model: "model", APIEndpoint: "http://example"})
	runCtx := &agent.RunContext{
		Prompt:     "prompt",
		StdoutPath: filepath.Join(dir, "stdout.txt"),
		StderrPath: filepath.Join(dir, "stderr.txt"),
	}
	if err := agentImpl.Execute(context.Background(), runCtx); err != nil {
		t.Fatalf("Execute: %v", err)
	}
	stdout, err := os.ReadFile(runCtx.StdoutPath)
	if err != nil {
		t.Fatalf("read stdout: %v", err)
	}
	if string(stdout) != "ok" {
		t.Fatalf("unexpected stdout: %q", string(stdout))
	}
}

func TestStreamContextError(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := streamContextError(ctx, nil); err == nil {
		t.Fatalf("expected context error")
	}
	triggered := make(chan struct{}, 1)
	triggered <- struct{}{}
	if err := streamContextError(context.Background(), triggered); err == nil {
		t.Fatalf("expected idle timeout error")
	}
}

func TestReadErrorBody(t *testing.T) {
	body := io.NopCloser(strings.NewReader("oops"))
	if text := readErrorBody(body); text != "oops" {
		t.Fatalf("unexpected body: %q", text)
	}
	if text := readErrorBody(nil); text != "" {
		t.Fatalf("expected empty text")
	}
}

func TestType(t *testing.T) {
	agent := NewPerplexityAgent(Options{})
	if agent.Type() != "perplexity" {
		t.Fatalf("unexpected type: %q", agent.Type())
	}
}

func TestDefaultHTTPClient(t *testing.T) {
	if defaultHTTPClient() == nil {
		t.Fatalf("expected http client")
	}
}

func TestSleepWithContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := sleepWithContext(ctx, time.Second); err == nil {
		t.Fatalf("expected canceled error")
	}
	if err := sleepWithContext(context.Background(), 0); err != nil {
		t.Fatalf("unexpected error for zero delay: %v", err)
	}
}

func TestLogRetry(t *testing.T) {
	var stderr bytes.Buffer
	logRetry(&stderr, time.Second, http.StatusTooManyRequests, nil)
	if !strings.Contains(stderr.String(), "status") {
		t.Fatalf("expected status log, got %q", stderr.String())
	}
	stderr.Reset()
	logRetry(&stderr, time.Second, 0, errors.New("boom"))
	if !strings.Contains(stderr.String(), "error") {
		t.Fatalf("expected error log, got %q", stderr.String())
	}
	stderr.Reset()
	logRetry(&stderr, time.Second, 0, nil)
	if !strings.Contains(stderr.String(), "retry") {
		t.Fatalf("expected retry log, got %q", stderr.String())
	}
}

func TestExtractContent(t *testing.T) {
	choice := perplexityChoice{Delta: perplexityDelta{Content: "delta"}}
	if got := extractContent(choice); got != "delta" {
		t.Fatalf("unexpected content: %q", got)
	}
	choice = perplexityChoice{Message: perplexityMessage{Content: "msg"}}
	if got := extractContent(choice); got != "msg" {
		t.Fatalf("unexpected message content: %q", got)
	}
}

func TestExecuteWithRetryStatusError(t *testing.T) {
	client := &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusBadRequest,
			Body:       io.NopCloser(strings.NewReader("bad request")),
			Header:     http.Header{},
		}, nil
	})}
	err := executeWithRetry(context.Background(), client, "http://example", "token", []byte("{}"), io.Discard, io.Discard, 0)
	if err == nil {
		t.Fatalf("expected error for bad status")
	}
}

func TestExecuteWithRetryRetryable(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	client := &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusTooManyRequests,
			Body:       io.NopCloser(strings.NewReader("retry")),
			Header:     http.Header{"Retry-After": []string{"1"}},
		}, nil
	})}
	if err := executeWithRetry(ctx, client, "http://example", "token", []byte("{}"), io.Discard, io.Discard, time.Second); err == nil {
		t.Fatalf("expected error for canceled retry")
	}
}

func TestExecuteWithRetryRequestError(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	client := &http.Client{Transport: roundTripFunc(func(r *http.Request) (*http.Response, error) {
		return nil, errors.New("network down")
	})}
	if err := executeWithRetry(ctx, client, "http://example", "token", []byte("{}"), io.Discard, io.Discard, 0); err == nil {
		t.Fatalf("expected request canceled error")
	}
}

func TestRetryDelayWithJitter(t *testing.T) {
	rng := rand.New(rand.NewSource(1))
	delay := retryDelay(nil, 1, rng)
	if delay <= time.Second {
		t.Fatalf("expected jittered delay, got %v", delay)
	}
}

func TestAppendCitationsFromSearchResults(t *testing.T) {
	var stdout bytes.Buffer
	err := appendCitations(&stdout, nil, []perplexitySearchResult{{URL: "http://example"}})
	if err != nil {
		t.Fatalf("appendCitations: %v", err)
	}
	if !strings.Contains(stdout.String(), "http://example") {
		t.Fatalf("expected search result citation")
	}
}
