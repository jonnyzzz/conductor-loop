// Package codex implements the Codex CLI agent backend.
package codex

import (
	"context"
	"io"
	"os"
	"os/exec"
	"sort"
	"strings"

	"github.com/pkg/errors"

	"github.com/jonnyzzz/conductor-loop/internal/agent"
)

const (
	codexCommand = "codex"
	tokenEnvVar  = "OPENAI_API_KEY"
)

// CodexAgent implements the Codex CLI backend.
type CodexAgent struct {
	token string
}

// Execute runs the Codex CLI for the provided run context.
func (a *CodexAgent) Execute(ctx context.Context, runCtx *agent.RunContext) (err error) {
	if ctx == nil {
		ctx = context.Background()
	}
	if runCtx == nil {
		return errors.New("run context is nil")
	}
	if ctx.Err() != nil {
		return errors.Wrap(ctx.Err(), "context canceled")
	}

	workingDir := strings.TrimSpace(runCtx.WorkingDir)
	if workingDir == "" {
		return errors.New("working dir is empty")
	}

	promptReader, closePrompt, err := openPrompt(runCtx.Prompt)
	if err != nil {
		return err
	}
	defer func() {
		if closeErr := closePrompt(); closeErr != nil && err == nil {
			err = errors.Wrap(closeErr, "close prompt")
		}
	}()

	capture, err := agent.CaptureOutput(nil, nil, agent.OutputFiles{
		StdoutPath: runCtx.StdoutPath,
		StderrPath: runCtx.StderrPath,
	})
	if err != nil {
		return errors.Wrap(err, "capture output")
	}
	defer func() {
		if closeErr := capture.Close(); closeErr != nil && err == nil {
			err = errors.Wrap(closeErr, "close output capture")
		}
	}()

	args := codexArgs(workingDir)
	env := buildEnvironment(runCtx.Environment, a.token)

	cmd, err := agent.SpawnProcessWithOptions(codexCommand, args, promptReader, capture.Stdout, capture.Stderr, agent.ProcessOptions{
		Dir: workingDir,
		Env: env,
	})
	if err != nil {
		return errors.Wrap(err, "spawn codex")
	}

	if err := waitForProcess(ctx, cmd); err != nil {
		return err
	}

	return nil
}

// Type returns the backend type identifier.
func (a *CodexAgent) Type() string {
	return "codex"
}

func openPrompt(prompt string) (io.Reader, func() error, error) {
	trimmed := strings.TrimSpace(prompt)
	if trimmed == "" {
		return nil, nil, errors.New("prompt is empty")
	}

	info, err := os.Stat(trimmed)
	if err == nil {
		if info.IsDir() {
			return nil, nil, errors.New("prompt path is a directory")
		}
		file, err := os.Open(trimmed)
		if err != nil {
			return nil, nil, errors.Wrap(err, "open prompt file")
		}
		return file, file.Close, nil
	}
	if !os.IsNotExist(err) {
		return nil, nil, errors.Wrap(err, "stat prompt")
	}

	return strings.NewReader(prompt), func() error { return nil }, nil
}

func codexArgs(workingDir string) []string {
	args := []string{
		"exec",
		"--dangerously-bypass-approvals-and-sandbox",
		"-",
	}
	if strings.TrimSpace(workingDir) == "" {
		return args
	}
	return []string{
		"exec",
		"--dangerously-bypass-approvals-and-sandbox",
		"-C",
		workingDir,
		"-",
	}
}

func buildEnvironment(overrides map[string]string, token string) []string {
	merged := make(map[string]string)
	for key, value := range overrides {
		cleanKey := strings.TrimSpace(key)
		if cleanKey == "" {
			continue
		}
		merged[cleanKey] = value
	}

	trimmedToken := strings.TrimSpace(token)
	if trimmedToken != "" {
		if _, exists := merged[tokenEnvVar]; !exists {
			merged[tokenEnvVar] = trimmedToken
		}
	}

	return mergeEnvironment(os.Environ(), merged)
}

func mergeEnvironment(base []string, overrides map[string]string) []string {
	merged := make(map[string]string, len(base)+len(overrides))
	for _, entry := range base {
		if entry == "" {
			continue
		}
		parts := strings.SplitN(entry, "=", 2)
		key := strings.TrimSpace(parts[0])
		if key == "" {
			continue
		}
		value := ""
		if len(parts) > 1 {
			value = parts[1]
		}
		merged[key] = value
	}
	for key, value := range overrides {
		cleanKey := strings.TrimSpace(key)
		if cleanKey == "" {
			continue
		}
		merged[cleanKey] = value
	}

	keys := make([]string, 0, len(merged))
	for key := range merged {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	result := make([]string, 0, len(keys))
	for _, key := range keys {
		result = append(result, key+"="+merged[key])
	}
	return result
}

func waitForProcess(ctx context.Context, cmd *exec.Cmd) error {
	if cmd == nil {
		return errors.New("command is nil")
	}
	waitCh := make(chan error, 1)
	go func() {
		waitCh <- cmd.Wait()
	}()

	select {
	case <-ctx.Done():
		if cmd.Process != nil {
			_ = cmd.Process.Kill()
		}
		waitErr := <-waitCh
		if waitErr != nil {
			return errors.Wrap(ctx.Err(), "codex execution canceled")
		}
		return errors.Wrap(ctx.Err(), "codex execution canceled")
	case waitErr := <-waitCh:
		if waitErr != nil {
			return errors.Wrap(waitErr, "codex execution failed")
		}
		return nil
	}
}

var _ agent.Agent = (*CodexAgent)(nil)
