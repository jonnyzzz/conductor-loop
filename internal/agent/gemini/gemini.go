// Package gemini implements the Gemini agent backend using the REST API.
package gemini

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"strings"

	"github.com/pkg/errors"

	"github.com/jonnyzzz/conductor-loop/internal/agent"
)

const (
	defaultBaseURL = "https://generativelanguage.googleapis.com"
	defaultModel   = "gemini-1.5-pro"
	tokenEnvVar    = "GEMINI_API_KEY"
)

// GeminiAgent invokes the Gemini REST API with streaming responses.
type GeminiAgent struct {
	Token      string
	BaseURL    string
	Model      string
	HTTPClient *http.Client
}

// Type returns the agent type identifier.
func (a *GeminiAgent) Type() string {
	return "gemini"
}

// Execute sends the prompt to Gemini and streams the response to stdout.
func (a *GeminiAgent) Execute(ctx context.Context, runCtx *agent.RunContext) (retErr error) {
	if runCtx == nil {
		return errors.New("run context is nil")
	}
	token := strings.TrimSpace(a.Token)
	if token == "" {
		token = strings.TrimSpace(runCtx.Environment[tokenEnvVar])
	}
	if token == "" {
		token = strings.TrimSpace(os.Getenv(tokenEnvVar))
	}
	if token == "" {
		return errors.New("gemini token is empty")
	}

	baseURL := strings.TrimSpace(a.BaseURL)
	if baseURL == "" {
		baseURL = defaultBaseURL
	}

	model := strings.TrimSpace(a.Model)
	if model == "" {
		model = defaultModel
	}

	capture, err := agent.CaptureOutput(nil, nil, agent.OutputFiles{
		StdoutPath: runCtx.StdoutPath,
		StderrPath: runCtx.StderrPath,
	})
	if err != nil {
		return errors.Wrap(err, "capture output")
	}
	defer func() {
		if err := capture.Close(); err != nil && retErr == nil {
			retErr = errors.Wrap(err, "close output")
		}
	}()

	stdout := bufio.NewWriter(capture.Stdout)
	stderr := bufio.NewWriter(capture.Stderr)
	defer func() {
		_ = stdout.Flush()
		_ = stderr.Flush()
	}()

	if err := a.stream(ctx, token, baseURL, model, runCtx.Prompt, stdout, stderr); err != nil {
		_, _ = fmt.Fprintf(stderr, "gemini request failed: %v\n", err)
		_ = stderr.Flush()
		return err
	}

	return nil
}

func (a *GeminiAgent) stream(ctx context.Context, token, baseURL, model, prompt string, stdout, stderr io.Writer) error {
	endpoint, err := buildEndpoint(baseURL, model)
	if err != nil {
		return err
	}

	payload, err := json.Marshal(geminiRequest{
		Contents: []geminiContent{{
			Role:  "user",
			Parts: []geminiPart{{Text: prompt}},
		}},
	})
	if err != nil {
		return errors.Wrap(err, "marshal request")
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, endpoint, bytes.NewReader(payload))
	if err != nil {
		return errors.Wrap(err, "build request")
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "text/event-stream")
	req.Header.Set("x-goog-api-key", token)

	client := a.HTTPClient
	if client == nil {
		client = &http.Client{
			Timeout: 0,
		}
	}

	resp, err := client.Do(req)
	if err != nil {
		return errors.Wrap(err, "call gemini api")
	}
	defer func() {
		_ = resp.Body.Close()
	}()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4*1024))
		return errors.Errorf("gemini api request failed: status %s: %s", resp.Status, strings.TrimSpace(string(body)))
	}

	return readStream(resp.Body, stdout, stderr)
}

func buildEndpoint(baseURL, model string) (string, error) {
	parsed, err := url.Parse(baseURL)
	if err != nil {
		return "", errors.Wrap(err, "parse base url")
	}
	if parsed.Scheme == "" {
		return "", errors.New("base url scheme is empty")
	}
	if parsed.Host == "" {
		return "", errors.New("base url host is empty")
	}

	parsed.Path = path.Join(parsed.Path, "v1beta", "models", model+":streamGenerateContent")
	query := parsed.Query()
	query.Set("alt", "sse")
	parsed.RawQuery = query.Encode()
	return parsed.String(), nil
}

func readStream(reader io.Reader, stdout, stderr io.Writer) error {
	scanner := bufio.NewScanner(reader)
	buf := make([]byte, 0, 64*1024)
	scanner.Buffer(buf, 1024*1024)

	var emitted string
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		if strings.HasPrefix(line, ":") || strings.HasPrefix(line, "event:") {
			continue
		}

		if strings.HasPrefix(line, "data:") {
			line = strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		}
		if line == "" {
			continue
		}
		if line == "[DONE]" {
			break
		}

		text, err := extractText(line)
		if err != nil {
			if stderr != nil {
				_, _ = fmt.Fprintf(stderr, "gemini stream parse error: %v\n", err)
				flushIfPossible(stderr)
			}
			return err
		}

		if text == "" {
			continue
		}

		delta := diffText(emitted, text)
		if delta == "" {
			continue
		}

		if _, err := io.WriteString(stdout, delta); err != nil {
			return errors.Wrap(err, "write stdout")
		}
		flushIfPossible(stdout)
		emitted += delta
	}

	if err := scanner.Err(); err != nil {
		return errors.Wrap(err, "read stream")
	}
	return nil
}

func flushIfPossible(writer io.Writer) {
	if flusher, ok := writer.(interface{ Flush() error }); ok {
		_ = flusher.Flush()
		return
	}
	if flusher, ok := writer.(interface{ Flush() }); ok {
		flusher.Flush()
	}
}

func diffText(emitted, next string) string {
	if strings.HasPrefix(next, emitted) {
		return next[len(emitted):]
	}
	if strings.HasPrefix(emitted, next) {
		return ""
	}
	return next
}

type geminiRequest struct {
	Contents []geminiContent `json:"contents"`
}

type geminiContent struct {
	Role  string       `json:"role,omitempty"`
	Parts []geminiPart `json:"parts"`
}

type geminiPart struct {
	Text string `json:"text,omitempty"`
}

type geminiResponse struct {
	Candidates []geminiCandidate `json:"candidates"`
	Error      *geminiError      `json:"error,omitempty"`
}

type geminiCandidate struct {
	Content geminiContent `json:"content"`
}

type geminiError struct {
	Message string `json:"message"`
}

func extractText(payload string) (string, error) {
	var resp geminiResponse
	if err := json.Unmarshal([]byte(payload), &resp); err != nil {
		return "", errors.Wrap(err, "decode gemini response")
	}
	if resp.Error != nil && resp.Error.Message != "" {
		return "", errors.Errorf("gemini api error: %s", strings.TrimSpace(resp.Error.Message))
	}

	var builder strings.Builder
	for _, candidate := range resp.Candidates {
		for _, part := range candidate.Content.Parts {
			if part.Text != "" {
				builder.WriteString(part.Text)
			}
		}
	}
	return builder.String(), nil
}
