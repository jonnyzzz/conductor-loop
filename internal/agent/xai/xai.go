// Package xai implements the xAI REST backend adapter.
package xai

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"io"
	"net"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/pkg/errors"

	"github.com/jonnyzzz/conductor-loop/internal/agent"
)

const (
	// TypeName identifies the xAI agent type.
	TypeName = "xai"

	// DefaultBaseURL is the default xAI API base URL.
	DefaultBaseURL = "https://api.x.ai"
	// DefaultModel is the default xAI model when none is specified.
	DefaultModel = "grok-4"

	// Environment variable names.
	envAPIKey      = "XAI_API_KEY"
	envBaseURL     = "XAI_BASE_URL"
	envAPIBase     = "XAI_API_BASE"
	envAPIEndpoint = "XAI_API_ENDPOINT"
	envModel       = "XAI_MODEL"
)

const (
	defaultDialTimeout    = 10 * time.Second
	defaultHeaderTimeout  = 60 * time.Second
	defaultIdleTimeout    = 90 * time.Second
	defaultMaxIdleConns   = 100
	defaultMaxLineSize    = 1024 * 1024
	defaultUserAgent      = "conductor-loop/xai"
	streamContentTypeHint = "text/event-stream"
)

// Config configures the xAI agent backend.
type Config struct {
	APIKey     string
	BaseURL    string
	Model      string
	HTTPClient *http.Client
	UserAgent  string
}

// Agent implements the xAI REST backend.
type Agent struct {
	apiKey     string
	baseURL    string
	model      string
	userAgent  string
	httpClient *http.Client
}

// NewAgent builds an xAI agent backend from explicit configuration.
func NewAgent(cfg Config) (*Agent, error) {
	apiKey := strings.TrimSpace(cfg.APIKey)
	baseURL := strings.TrimSpace(cfg.BaseURL)
	model := strings.TrimSpace(cfg.Model)
	userAgent := strings.TrimSpace(cfg.UserAgent)

	if model == "" {
		model = DefaultModel
	}
	if baseURL == "" {
		baseURL = DefaultBaseURL
	}
	if userAgent == "" {
		userAgent = defaultUserAgent
	}

	client := cfg.HTTPClient
	if client == nil {
		client = defaultHTTPClient()
	}

	return &Agent{
		apiKey:     apiKey,
		baseURL:    baseURL,
		model:      model,
		userAgent:  userAgent,
		httpClient: client,
	}, nil
}

// NewAgentFromEnvironment builds an xAI agent backend using environment values.
func NewAgentFromEnvironment(env map[string]string) (*Agent, error) {
	return NewAgent(Config{
		APIKey:  lookupEnv(env, envAPIKey),
		BaseURL: lookupEnv(env, envAPIEndpoint, envBaseURL, envAPIBase),
		Model:   lookupEnv(env, envModel),
	})
}

// Type returns the agent type identifier.
func (a *Agent) Type() string {
	return TypeName
}

// Execute runs the xAI request and streams output to stdout.
func (a *Agent) Execute(ctx context.Context, runCtx *agent.RunContext) error {
	if runCtx == nil {
		return errors.New("run context is nil")
	}
	prompt := runCtx.Prompt
	if strings.TrimSpace(prompt) == "" {
		return errors.New("prompt is empty")
	}

	capture, err := agent.CaptureOutput(os.Stdout, os.Stderr, agent.OutputFiles{
		StdoutPath: runCtx.StdoutPath,
		StderrPath: runCtx.StderrPath,
	})
	if err != nil {
		return errors.Wrap(err, "capture output")
	}
	defer func() {
		_ = capture.Close()
	}()

	resolved, err := a.resolveConfig(runCtx.Environment)
	if err != nil {
		return err
	}

	if err := resolved.streamCompletion(ctx, prompt, capture.Stdout); err != nil {
		return err
	}

	return nil
}

type resolvedConfig struct {
	apiKey     string
	model      string
	endpoint   string
	userAgent  string
	httpClient *http.Client
}

func (a *Agent) resolveConfig(env map[string]string) (*resolvedConfig, error) {
	apiKey := a.apiKey
	if apiKey == "" {
		apiKey = lookupEnv(env, envAPIKey)
	}
	if strings.TrimSpace(apiKey) == "" {
		return nil, errors.New("xai api key is empty")
	}

	baseURL := a.baseURL
	if override := lookupEnv(env, envAPIEndpoint, envBaseURL, envAPIBase); override != "" {
		baseURL = override
	}
	endpoint, err := resolveEndpoint(baseURL)
	if err != nil {
		return nil, err
	}

	model := a.model
	if override := lookupEnv(env, envModel); override != "" {
		model = override
	}
	if strings.TrimSpace(model) == "" {
		model = DefaultModel
	}

	client := a.httpClient
	if client == nil {
		client = defaultHTTPClient()
	}

	userAgent := a.userAgent
	if userAgent == "" {
		userAgent = defaultUserAgent
	}

	return &resolvedConfig{
		apiKey:     apiKey,
		model:      model,
		endpoint:   endpoint,
		userAgent:  userAgent,
		httpClient: client,
	}, nil
}

func (c *resolvedConfig) streamCompletion(ctx context.Context, prompt string, stdout io.Writer) error {
	reqBody := chatCompletionRequest{
		Model: c.model,
		Messages: []chatMessage{
			{
				Role:    "user",
				Content: prompt,
			},
		},
		Stream: true,
	}

	payload, err := json.Marshal(reqBody)
	if err != nil {
		return errors.Wrap(err, "encode request")
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.endpoint, bytes.NewReader(payload))
	if err != nil {
		return errors.Wrap(err, "build request")
	}
	req.Header.Set("Authorization", "Bearer "+c.apiKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", streamContentTypeHint)
	req.Header.Set("User-Agent", c.userAgent)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return errors.Wrap(err, "send request")
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 16*1024))
		return errors.Errorf("xai request failed: status %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	if strings.Contains(strings.ToLower(resp.Header.Get("Content-Type")), streamContentTypeHint) {
		return streamResponse(resp.Body, stdout)
	}

	return decodeSingleResponse(resp.Body, stdout)
}

func defaultHTTPClient() *http.Client {
	transport := &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		DialContext: (&net.Dialer{
			Timeout:   defaultDialTimeout,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		ForceAttemptHTTP2:     true,
		MaxIdleConns:          defaultMaxIdleConns,
		IdleConnTimeout:       defaultIdleTimeout,
		TLSHandshakeTimeout:   defaultDialTimeout,
		ResponseHeaderTimeout: defaultHeaderTimeout,
		ExpectContinueTimeout: 1 * time.Second,
	}

	return &http.Client{
		Transport: transport,
	}
}

func resolveEndpoint(baseURL string) (string, error) {
	trimmed := strings.TrimSpace(baseURL)
	if trimmed == "" {
		trimmed = DefaultBaseURL
	}
	u, err := url.Parse(trimmed)
	if err != nil {
		return "", errors.Wrap(err, "parse base url")
	}

	path := strings.TrimRight(u.Path, "/")
	switch {
	case strings.HasSuffix(path, "/v1/chat/completions"):
		return u.String(), nil
	case strings.HasSuffix(path, "/v1"):
		u.Path = path + "/chat/completions"
	default:
		if path == "" {
			u.Path = "/v1/chat/completions"
		} else {
			u.Path = path + "/v1/chat/completions"
		}
	}
	return u.String(), nil
}

func lookupEnv(env map[string]string, keys ...string) string {
	for _, key := range keys {
		if env != nil {
			if value, ok := env[key]; ok {
				if strings.TrimSpace(value) != "" {
					return strings.TrimSpace(value)
				}
			}
		}
		if value, ok := os.LookupEnv(key); ok {
			if strings.TrimSpace(value) != "" {
				return strings.TrimSpace(value)
			}
		}
	}
	return ""
}

func streamResponse(reader io.Reader, stdout io.Writer) error {
	scanner := bufio.NewScanner(reader)
	scanner.Buffer(make([]byte, 0, 64*1024), defaultMaxLineSize)

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, ":") {
			continue
		}
		if !strings.HasPrefix(line, "data:") {
			continue
		}
		payload := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		if payload == "" {
			continue
		}
		if payload == "[DONE]" {
			return nil
		}
		if err := writeChunk(payload, stdout); err != nil {
			return err
		}
	}
	if err := scanner.Err(); err != nil {
		return errors.Wrap(err, "read stream")
	}
	return nil
}

func writeChunk(payload string, stdout io.Writer) error {
	var chunk chatCompletionChunk
	if err := json.Unmarshal([]byte(payload), &chunk); err != nil {
		return errors.Wrap(err, "decode stream chunk")
	}
	if chunk.Error != nil {
		return errors.New(strings.TrimSpace(chunk.Error.Message))
	}
	content := chunk.Content()
	if content == "" {
		return nil
	}
	if _, err := io.WriteString(stdout, content); err != nil {
		return errors.Wrap(err, "write stream content")
	}
	return nil
}

func decodeSingleResponse(reader io.Reader, stdout io.Writer) error {
	body, err := io.ReadAll(reader)
	if err != nil {
		return errors.Wrap(err, "read response")
	}
	var resp chatCompletionResponse
	if err := json.Unmarshal(body, &resp); err != nil {
		return errors.Wrap(err, "decode response")
	}
	if resp.Error != nil {
		return errors.New(strings.TrimSpace(resp.Error.Message))
	}
	content := resp.Content()
	if strings.TrimSpace(content) == "" {
		return errors.New("xai response missing content")
	}
	if _, err := io.WriteString(stdout, content); err != nil {
		return errors.Wrap(err, "write response content")
	}
	return nil
}

type chatCompletionRequest struct {
	Model    string        `json:"model"`
	Messages []chatMessage `json:"messages"`
	Stream   bool          `json:"stream,omitempty"`
}

type chatMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type chatCompletionChunk struct {
	Choices []chatChoice `json:"choices"`
	Error   *apiError    `json:"error,omitempty"`
}

func (c chatCompletionChunk) Content() string {
	if len(c.Choices) == 0 {
		return ""
	}
	choice := c.Choices[0]
	if choice.Delta.Content != "" {
		return choice.Delta.Content
	}
	if choice.Message.Content != "" {
		return choice.Message.Content
	}
	if choice.Text != "" {
		return choice.Text
	}
	return ""
}

type chatCompletionResponse struct {
	Choices []chatChoice `json:"choices"`
	Error   *apiError    `json:"error,omitempty"`
}

func (c chatCompletionResponse) Content() string {
	if len(c.Choices) == 0 {
		return ""
	}
	choice := c.Choices[0]
	if choice.Message.Content != "" {
		return choice.Message.Content
	}
	if choice.Text != "" {
		return choice.Text
	}
	if choice.Delta.Content != "" {
		return choice.Delta.Content
	}
	return ""
}

type chatChoice struct {
	Delta   chatDelta   `json:"delta,omitempty"`
	Message chatMessage `json:"message,omitempty"`
	Text    string      `json:"text,omitempty"`
}

type chatDelta struct {
	Content string `json:"content,omitempty"`
	Role    string `json:"role,omitempty"`
}

type apiError struct {
	Message string `json:"message"`
}
