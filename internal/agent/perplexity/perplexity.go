// Package perplexity provides a Perplexity REST/SSE agent backend.
package perplexity

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"math/rand"
	"net"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/pkg/errors"

	"github.com/jonnyzzz/conductor-loop/internal/agent"
)

const (
	defaultPerplexityEndpoint = "https://api.perplexity.ai/chat/completions"
	defaultPerplexityModel    = "sonar-reasoning"
	perplexityTokenEnv        = "PERPLEXITY_API_KEY"
	maxRetryAttempts          = 5
)

const (
	connectTimeout        = 10 * time.Second
	tlsHandshakeTimeout   = 10 * time.Second
	responseHeaderTimeout = 10 * time.Second
	idleStreamTimeout     = 60 * time.Second
	totalRequestTimeout   = 2 * time.Minute
	maxScannerTokenSize   = 1024 * 1024
)

// Options defines configuration for a Perplexity agent backend.
type Options struct {
	Token       string
	Model       string
	APIEndpoint string
	HTTPClient  *http.Client
}

// PerplexityAgent implements the agent.Agent interface for Perplexity.
type PerplexityAgent struct {
	token       string
	model       string
	apiEndpoint string
	client      *http.Client
	idleTimeout time.Duration
}

// NewPerplexityAgent constructs a PerplexityAgent with the provided options.
func NewPerplexityAgent(options Options) *PerplexityAgent {
	endpoint := strings.TrimSpace(options.APIEndpoint)
	if endpoint == "" {
		endpoint = defaultPerplexityEndpoint
	}
	model := strings.TrimSpace(options.Model)
	if model == "" {
		model = defaultPerplexityModel
	}
	return &PerplexityAgent{
		token:       strings.TrimSpace(options.Token),
		model:       model,
		apiEndpoint: endpoint,
		client:      options.HTTPClient,
		idleTimeout: idleStreamTimeout,
	}
}

// Execute runs the Perplexity API request and streams tokens to stdout.
func (a *PerplexityAgent) Execute(ctx context.Context, runCtx *agent.RunContext) (err error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if runCtx == nil {
		return errors.New("run context is nil")
	}

	capture, err := agent.CaptureOutput(nil, nil, agent.OutputFiles{
		StdoutPath: runCtx.StdoutPath,
		StderrPath: runCtx.StderrPath,
	})
	if err != nil {
		return errors.Wrap(err, "capture output")
	}
	defer func() {
		if cerr := capture.Close(); cerr != nil && err == nil {
			err = errors.Wrap(cerr, "close output capture")
		}
	}()

	token := resolveToken(a.token, runCtx.Environment)
	if token == "" {
		return errors.New("perplexity api token is empty")
	}

	model := strings.TrimSpace(a.model)
	if model == "" {
		model = defaultPerplexityModel
	}
	endpoint := strings.TrimSpace(a.apiEndpoint)
	if endpoint == "" {
		endpoint = defaultPerplexityEndpoint
	}

	payload, err := json.Marshal(perplexityRequest{
		Model: model,
		Messages: []perplexityMessage{{
			Role:    "user",
			Content: runCtx.Prompt,
		}},
		Stream: true,
	})
	if err != nil {
		return errors.Wrap(err, "marshal request")
	}

	client := a.client
	if client == nil {
		client = defaultHTTPClient()
	}
	return executeWithRetry(ctx, client, endpoint, token, payload, capture.Stdout, capture.Stderr, a.idleTimeout)
}

// Type returns the agent type.
func (a *PerplexityAgent) Type() string {
	return "perplexity"
}

type perplexityRequest struct {
	Model    string              `json:"model"`
	Messages []perplexityMessage `json:"messages"`
	Stream   bool                `json:"stream"`
}

type perplexityMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type perplexityStreamResponse struct {
	Choices       []perplexityChoice       `json:"choices"`
	Citations     []string                 `json:"citations"`
	SearchResults []perplexitySearchResult `json:"search_results"`
	Error         *perplexityError         `json:"error"`
}

type perplexityChoice struct {
	Delta        perplexityDelta   `json:"delta"`
	Message      perplexityMessage `json:"message"`
	FinishReason string            `json:"finish_reason"`
	Index        int               `json:"index"`
	Logprobs     *json.RawMessage  `json:"logprobs"`
}

type perplexityDelta struct {
	Content string `json:"content"`
}

type perplexitySearchResult struct {
	Title string `json:"title"`
	URL   string `json:"url"`
}

type perplexityError struct {
	Message string `json:"message"`
	Type    string `json:"type"`
}

type streamState struct {
	lastContent  string
	citations    []string
	searchResult []perplexitySearchResult
}

func resolveToken(fallback string, env map[string]string) string {
	if env != nil {
		if value := strings.TrimSpace(env[perplexityTokenEnv]); value != "" {
			return value
		}
	}
	if strings.TrimSpace(fallback) != "" {
		return strings.TrimSpace(fallback)
	}
	if value, ok := os.LookupEnv(perplexityTokenEnv); ok {
		return strings.TrimSpace(value)
	}
	return ""
}

func defaultHTTPClient() *http.Client {
	transport := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   connectTimeout,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		TLSHandshakeTimeout:   tlsHandshakeTimeout,
		ResponseHeaderTimeout: responseHeaderTimeout,
		IdleConnTimeout:       90 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}
	return &http.Client{
		Transport: transport,
		Timeout:   totalRequestTimeout,
	}
}

func executeWithRetry(ctx context.Context, client *http.Client, endpoint, token string, payload []byte, stdout, stderr io.Writer, idleTimeout time.Duration) error {
	if client == nil {
		return errors.New("http client is nil")
	}
	if strings.TrimSpace(endpoint) == "" {
		return errors.New("perplexity api endpoint is empty")
	}
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))

	for attempt := 0; attempt < maxRetryAttempts; attempt++ {
		reqCtx, cancel := context.WithCancel(ctx)
		req, err := http.NewRequestWithContext(reqCtx, http.MethodPost, endpoint, bytes.NewReader(payload))
		if err != nil {
			cancel()
			return errors.Wrap(err, "create request")
		}
		req.Header.Set("Authorization", "Bearer "+token)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Accept", "text/event-stream")

		resp, err := client.Do(req)
		if err != nil {
			cancel()
			if ctx.Err() != nil {
				return errors.Wrap(ctx.Err(), "request canceled")
			}
			if attempt < maxRetryAttempts-1 {
				delay := retryDelay(nil, attempt, rng)
				logRetry(stderr, delay, 0, err)
				if err := sleepWithContext(ctx, delay); err != nil {
					return err
				}
				continue
			}
			return errors.Wrap(err, "send request")
		}

		if resp.StatusCode == http.StatusOK {
			state := &streamState{}
			err = consumeStream(reqCtx, cancel, resp.Body, stdout, idleTimeout, state)
			cancel()
			if err != nil {
				return err
			}
			return appendCitations(stdout, state.citations, state.searchResult)
		}

		status := resp.StatusCode
		bodyText := readErrorBody(resp.Body)
		_ = resp.Body.Close()
		cancel()

		retryable := status == http.StatusTooManyRequests || status >= http.StatusInternalServerError
		if retryable && attempt < maxRetryAttempts-1 {
			delay := retryDelay(resp, attempt, rng)
			logRetry(stderr, delay, status, nil)
			if err := sleepWithContext(ctx, delay); err != nil {
				return err
			}
			continue
		}

		if bodyText != "" {
			return errors.Errorf("perplexity api error: status %d: %s", status, bodyText)
		}
		return errors.Errorf("perplexity api error: status %d", status)
	}
	return errors.New("perplexity request failed after retries")
}

func consumeStream(ctx context.Context, cancel context.CancelFunc, body io.ReadCloser, stdout io.Writer, idleTimeout time.Duration, state *streamState) error {
	if body == nil {
		return errors.New("response body is nil")
	}
	defer body.Close()

	activity, idleTriggered := startIdleMonitor(ctx, cancel, idleTimeout)

	scanner := bufio.NewScanner(body)
	scanner.Buffer(make([]byte, 0, 64*1024), maxScannerTokenSize)

	dataLines := make([]string, 0, 4)
	for scanner.Scan() {
		signalActivity(activity)
		line := scanner.Text()
		if line == "" {
			if len(dataLines) > 0 {
				done, err := handleEvent(strings.Join(dataLines, "\n"), stdout, state)
				if err != nil {
					return err
				}
				if done {
					return nil
				}
				dataLines = dataLines[:0]
			}
			continue
		}
		if strings.HasPrefix(line, "data:") {
			dataLines = append(dataLines, strings.TrimSpace(line[len("data:"):]))
		}
	}
	if len(dataLines) > 0 {
		done, err := handleEvent(strings.Join(dataLines, "\n"), stdout, state)
		if err != nil {
			return err
		}
		if done {
			return nil
		}
	}

	if err := scanner.Err(); err != nil {
		if ctx.Err() != nil {
			return streamContextError(ctx, idleTriggered)
		}
		return errors.Wrap(err, "read stream")
	}
	if ctx.Err() != nil {
		return streamContextError(ctx, idleTriggered)
	}
	return nil
}

func handleEvent(payload string, stdout io.Writer, state *streamState) (bool, error) {
	trimmed := strings.TrimSpace(payload)
	if trimmed == "" {
		return false, nil
	}
	if trimmed == "[DONE]" {
		return true, nil
	}

	var chunk perplexityStreamResponse
	if err := json.Unmarshal([]byte(trimmed), &chunk); err != nil {
		return false, errors.Wrap(err, "parse stream chunk")
	}
	if chunk.Error != nil {
		message := strings.TrimSpace(chunk.Error.Message)
		if message == "" {
			message = "perplexity stream error"
		}
		return false, errors.New(message)
	}

	if len(chunk.Choices) > 0 {
		choice := chunk.Choices[0]
		if choice.FinishReason == "error" {
			return false, errors.New("perplexity stream finished with error")
		}
		content := extractContent(choice)
		if err := appendContent(stdout, content, &state.lastContent); err != nil {
			return false, err
		}
	}

	if len(chunk.Citations) > 0 {
		state.citations = append(state.citations, chunk.Citations...)
	}
	if len(chunk.SearchResults) > 0 {
		state.searchResult = append(state.searchResult, chunk.SearchResults...)
	}

	return false, nil
}

func extractContent(choice perplexityChoice) string {
	if choice.Delta.Content != "" {
		return choice.Delta.Content
	}
	if choice.Message.Content != "" {
		return choice.Message.Content
	}
	return ""
}

func appendContent(stdout io.Writer, content string, last *string) error {
	if stdout == nil {
		return errors.New("stdout writer is nil")
	}
	if content == "" {
		return nil
	}
	if last != nil && *last != "" && strings.HasPrefix(content, *last) {
		content = strings.TrimPrefix(content, *last)
	}
	if content == "" {
		return nil
	}
	if _, err := io.WriteString(stdout, content); err != nil {
		return errors.Wrap(err, "write stdout")
	}
	if last != nil {
		*last += content
	}
	return nil
}

func appendCitations(stdout io.Writer, citations []string, searchResults []perplexitySearchResult) error {
	items := make([]string, 0, len(citations)+len(searchResults))
	seen := make(map[string]struct{})
	for _, cite := range citations {
		trimmed := strings.TrimSpace(cite)
		if trimmed == "" {
			continue
		}
		if _, exists := seen[trimmed]; exists {
			continue
		}
		seen[trimmed] = struct{}{}
		items = append(items, trimmed)
	}
	if len(items) == 0 {
		for _, result := range searchResults {
			trimmed := strings.TrimSpace(result.URL)
			if trimmed == "" {
				continue
			}
			if _, exists := seen[trimmed]; exists {
				continue
			}
			seen[trimmed] = struct{}{}
			items = append(items, trimmed)
		}
	}
	if len(items) == 0 {
		return nil
	}
	if _, err := io.WriteString(stdout, "\n\nSources:\n"); err != nil {
		return errors.Wrap(err, "write citations header")
	}
	for i, item := range items {
		line := fmt.Sprintf("[%d] %s\n", i+1, item)
		if _, err := io.WriteString(stdout, line); err != nil {
			return errors.Wrap(err, "write citations")
		}
	}
	return nil
}

func startIdleMonitor(ctx context.Context, cancel context.CancelFunc, timeout time.Duration) (chan<- struct{}, <-chan struct{}) {
	if timeout <= 0 {
		return nil, nil
	}
	activity := make(chan struct{}, 1)
	triggered := make(chan struct{}, 1)
	go func() {
		timer := time.NewTimer(timeout)
		defer timer.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-timer.C:
				select {
				case triggered <- struct{}{}:
				default:
				}
				cancel()
				return
			case <-activity:
				if !timer.Stop() {
					select {
					case <-timer.C:
					default:
					}
				}
				timer.Reset(timeout)
			}
		}
	}()
	return activity, triggered
}

func signalActivity(activity chan<- struct{}) {
	if activity == nil {
		return
	}
	select {
	case activity <- struct{}{}:
	default:
	}
}

func streamContextError(ctx context.Context, idleTriggered <-chan struct{}) error {
	if idleTriggered != nil {
		select {
		case <-idleTriggered:
			return errors.New("perplexity stream idle timeout")
		default:
		}
	}
	if ctx.Err() != nil {
		return errors.Wrap(ctx.Err(), "stream context canceled")
	}
	return errors.New("stream context canceled")
}

func retryDelay(resp *http.Response, attempt int, rng *rand.Rand) time.Duration {
	if resp != nil {
		if delay, ok := parseRetryAfter(resp.Header.Get("Retry-After")); ok {
			return delay
		}
	}
	backoff := time.Second << attempt
	if backoff > 32*time.Second {
		backoff = 32 * time.Second
	}
	if rng == nil {
		return backoff
	}
	jitterMax := backoff / 2
	if jitterMax <= 0 {
		return backoff
	}
	return backoff + time.Duration(rng.Int63n(int64(jitterMax)))
}

func parseRetryAfter(value string) (time.Duration, bool) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return 0, false
	}
	if seconds, err := strconv.Atoi(trimmed); err == nil {
		if seconds <= 0 {
			return 0, false
		}
		return time.Duration(seconds) * time.Second, true
	}
	if parsed, err := http.ParseTime(trimmed); err == nil {
		delta := time.Until(parsed)
		if delta > 0 {
			return delta, true
		}
	}
	return 0, false
}

func sleepWithContext(ctx context.Context, delay time.Duration) error {
	if delay <= 0 {
		return nil
	}
	timer := time.NewTimer(delay)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return errors.Wrap(ctx.Err(), "request canceled")
	case <-timer.C:
		return nil
	}
}

func logRetry(stderr io.Writer, delay time.Duration, status int, err error) {
	if stderr == nil {
		return
	}
	if status > 0 {
		_, _ = fmt.Fprintf(stderr, "perplexity retry in %s after status %d\n", delay, status)
		return
	}
	if err != nil {
		_, _ = fmt.Fprintf(stderr, "perplexity retry in %s after error: %v\n", delay, err)
		return
	}
	_, _ = fmt.Fprintf(stderr, "perplexity retry in %s\n", delay)
}

func readErrorBody(body io.ReadCloser) string {
	if body == nil {
		return ""
	}
	defer body.Close()
	data, err := io.ReadAll(io.LimitReader(body, 8*1024))
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(data))
}
