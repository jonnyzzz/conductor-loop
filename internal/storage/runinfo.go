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
)

// RunInfo defines the persisted run metadata stored in run-info.yaml.
type RunInfo struct {
	RunID       string    `yaml:"run_id"`
	ParentRunID string    `yaml:"parent_run_id,omitempty"`
	ProjectID   string    `yaml:"project_id"`
	TaskID      string    `yaml:"task_id"`
	AgentType   string    `yaml:"agent_type"`
	PID         int       `yaml:"pid"`
	PGID        int       `yaml:"pgid"`
	StartTime   time.Time `yaml:"start_time"`
	EndTime     time.Time `yaml:"end_time,omitempty"`
	ExitCode    int       `yaml:"exit_code,omitempty"`
	Status      string    `yaml:"status"` // running, completed, failed
}
