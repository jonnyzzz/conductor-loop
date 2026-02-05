// Package agent defines shared interfaces and run context for agent backends.
package agent

import "context"

// Agent defines the common behavior implemented by all agent backends.
type Agent interface {
	// Execute runs the agent with the given context.
	Execute(ctx context.Context, runCtx *RunContext) error

	// Type returns the agent type (claude, codex, etc.).
	Type() string
}

// RunContext carries the runtime data needed to execute an agent backend.
type RunContext struct {
	RunID       string
	ProjectID   string
	TaskID      string
	Prompt      string
	WorkingDir  string
	StdoutPath  string
	StderrPath  string
	Environment map[string]string
}
