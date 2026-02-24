// Package storage provides file-based storage for run metadata.
package storage

import "time"

const (
	// StatusRunning indicates an in-progress run.
	StatusRunning = "running"
	// StatusCompleted indicates a successful run completion.
	StatusCompleted = "completed"
	// StatusFailed indicates a failed run completion.
	StatusFailed = "failed"
	// StatusUnknown is used for synthetic RunInfo when run-info.yaml is absent.
	StatusUnknown = "unknown"

	// Task-level derived statuses (not stored in run-info.yaml).

	// StatusAllFinished indicates a task with the DONE marker and no active runs.
	StatusAllFinished = "all_finished"
	// StatusQueued indicates runs are scheduled but not yet started.
	StatusQueued = "queued"
	// StatusBlocked indicates all pending runs have unfulfilled dependencies.
	StatusBlocked = "blocked"
	// StatusPartialFail indicates some runs have failed while the task still has active runs.
	StatusPartialFail = "partial_failure"
)

// RunInfo defines the persisted run metadata stored in run-info.yaml.
type RunInfo struct {
	Version          int       `yaml:"version"`
	RunID            string    `yaml:"run_id"`
	ParentRunID      string    `yaml:"parent_run_id,omitempty"`
	PreviousRunID    string    `yaml:"previous_run_id,omitempty"`
	ProjectID        string    `yaml:"project_id"`
	TaskID           string    `yaml:"task_id"`
	AgentType        string    `yaml:"agent"`
	ProcessOwnership string    `yaml:"process_ownership,omitempty"` // managed (default) or external
	PID              int       `yaml:"pid"`
	PGID             int       `yaml:"pgid"`
	StartTime        time.Time `yaml:"start_time"`
	EndTime          time.Time `yaml:"end_time"`
	ExitCode         int       `yaml:"exit_code"`
	Status           string    `yaml:"status"` // running, completed, failed
	CWD              string    `yaml:"cwd,omitempty"`
	PromptPath       string    `yaml:"prompt_path,omitempty"`
	OutputPath       string    `yaml:"output_path,omitempty"`
	StdoutPath       string    `yaml:"stdout_path,omitempty"`
	StderrPath       string    `yaml:"stderr_path,omitempty"`
	CommandLine      string    `yaml:"commandline,omitempty"`
	ErrorSummary     string    `yaml:"error_summary,omitempty"`
	AgentVersion     string    `yaml:"agent_version"`
}
