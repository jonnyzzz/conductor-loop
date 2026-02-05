// Package runner manages agent process execution for the orchestration subsystem.
package runner

import (
	"context"
	stderrors "errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/jonnyzzz/conductor-loop/internal/messagebus"
	"github.com/pkg/errors"
)

const (
	defaultRalphWaitTimeout  = 300 * time.Second
	defaultRalphPollInterval = time.Second
	defaultRalphMaxRestarts  = 100
	defaultRalphRestartDelay = time.Second
)

// RootRunner executes one root agent run.
type RootRunner func(ctx context.Context, attempt int) error

// RalphLoop implements the root agent restart loop.
type RalphLoop struct {
	runDir      string
	messagebus  *messagebus.MessageBus
	maxRestarts int
	waitTimeout time.Duration

	pollInterval time.Duration
	restartDelay time.Duration
	projectID    string
	taskID       string
	runRoot      RootRunner
}

// RalphOption configures the Ralph loop.
type RalphOption func(*RalphLoop)

// NewRalphLoop constructs a RalphLoop with defaults and validation.
func NewRalphLoop(runDir string, bus *messagebus.MessageBus, opts ...RalphOption) (*RalphLoop, error) {
	clean := filepath.Clean(strings.TrimSpace(runDir))
	if clean == "." || clean == "" {
		return nil, errors.New("run directory is empty")
	}
	if bus == nil {
		return nil, errors.New("message bus is nil")
	}
	info, err := os.Stat(clean)
	if err != nil {
		return nil, errors.Wrap(err, "stat run directory")
	}
	if !info.IsDir() {
		return nil, errors.New("run directory is not a directory")
	}

	rl := &RalphLoop{
		runDir:       clean,
		messagebus:   bus,
		maxRestarts:  defaultRalphMaxRestarts,
		waitTimeout:  defaultRalphWaitTimeout,
		pollInterval: defaultRalphPollInterval,
		restartDelay: defaultRalphRestartDelay,
	}
	for _, opt := range opts {
		if opt != nil {
			opt(rl)
		}
	}
	if rl.waitTimeout <= 0 {
		return nil, errors.New("wait timeout must be positive")
	}
	if rl.pollInterval <= 0 {
		return nil, errors.New("poll interval must be positive")
	}
	if rl.restartDelay < 0 {
		return nil, errors.New("restart delay must not be negative")
	}
	if rl.maxRestarts < 0 {
		return nil, errors.New("max restarts must not be negative")
	}
	if rl.runRoot == nil {
		return nil, errors.New("root runner is nil")
	}
	if rl.projectID == "" || rl.taskID == "" {
		projectID, taskID := inferProjectTaskID(clean)
		if rl.projectID == "" {
			rl.projectID = projectID
		}
		if rl.taskID == "" {
			rl.taskID = taskID
		}
	}
	if rl.projectID == "" {
		return nil, errors.New("project id is empty")
	}

	return rl, nil
}

// WithMaxRestarts overrides the maximum restarts.
func WithMaxRestarts(max int) RalphOption {
	return func(rl *RalphLoop) {
		rl.maxRestarts = max
	}
}

// WithWaitTimeout overrides the child wait timeout.
func WithWaitTimeout(timeout time.Duration) RalphOption {
	return func(rl *RalphLoop) {
		rl.waitTimeout = timeout
	}
}

// WithPollInterval overrides the child polling interval.
func WithPollInterval(interval time.Duration) RalphOption {
	return func(rl *RalphLoop) {
		rl.pollInterval = interval
	}
}

// WithRestartDelay overrides the pause between restarts.
func WithRestartDelay(delay time.Duration) RalphOption {
	return func(rl *RalphLoop) {
		rl.restartDelay = delay
	}
}

// WithProjectTask sets project/task identifiers for message bus entries.
func WithProjectTask(projectID, taskID string) RalphOption {
	return func(rl *RalphLoop) {
		rl.projectID = strings.TrimSpace(projectID)
		rl.taskID = strings.TrimSpace(taskID)
	}
}

// WithRootRunner sets the root runner callback.
func WithRootRunner(run RootRunner) RalphOption {
	return func(rl *RalphLoop) {
		rl.runRoot = run
	}
}

// Run executes the Ralph loop until completion or error.
func (rl *RalphLoop) Run(ctx context.Context) error {
	if rl == nil {
		return errors.New("ralph loop is nil")
	}
	if ctx == nil {
		ctx = context.Background()
	}

	restarts := 0
	for {
		if err := ctx.Err(); err != nil {
			return err
		}

		done, err := rl.doneExists()
		if err != nil {
			return err
		}
		if done {
			return rl.handleDone(ctx)
		}
		if restarts >= rl.maxRestarts {
			if logErr := rl.appendMessage("ERROR", fmt.Sprintf("task failed: max restarts (%d) exceeded", rl.maxRestarts)); logErr != nil {
				return logErr
			}
			return errors.New("max restarts exceeded")
		}

		if err := rl.appendMessage("INFO", fmt.Sprintf("starting root agent (restart #%d)", restarts)); err != nil {
			return err
		}
		if err := rl.runRoot(ctx, restarts); err != nil {
			if logErr := rl.appendMessage("WARNING", fmt.Sprintf("root agent failed on restart #%d: %v", restarts, err)); logErr != nil {
				return logErr
			}
		}
		restarts++

		done, err = rl.doneExists()
		if err != nil {
			return err
		}
		if done {
			return rl.handleDone(ctx)
		}

		if err := sleepWithContext(ctx, rl.restartDelay); err != nil {
			return err
		}
	}
}

func (rl *RalphLoop) handleDone(ctx context.Context) error {
	children, err := FindActiveChildren(rl.runDir)
	if err != nil {
		if logErr := rl.appendMessage("WARNING", fmt.Sprintf("failed to enumerate children: %v", err)); logErr != nil {
			return logErr
		}
	}
	if len(children) == 0 {
		return rl.appendMessage("INFO", "task completed (DONE marker present, no active children)")
	}

	childIDs := childRunIDs(children)
	if err := rl.appendMessage("INFO", fmt.Sprintf("waiting for %d children to complete: %s", len(children), childIDs)); err != nil {
		return err
	}

	remaining, waitErr := WaitForChildren(ctx, children, rl.waitTimeout, rl.pollInterval)
	if waitErr != nil {
		if stderrors.Is(waitErr, ErrChildWaitTimeout) {
			return rl.appendMessage("WARNING", fmt.Sprintf("timeout waiting for children after %s: %s", rl.waitTimeout, childRunIDs(remaining)))
		}
		return waitErr
	}
	return rl.appendMessage("INFO", "task completed (all children finished)")
}

func (rl *RalphLoop) doneExists() (bool, error) {
	path := filepath.Join(rl.runDir, "DONE")
	info, err := os.Stat(path)
	if err == nil {
		if info.IsDir() {
			return false, errors.New("DONE is a directory")
		}
		return true, nil
	}
	if os.IsNotExist(err) {
		return false, nil
	}
	return false, errors.Wrap(err, "stat DONE")
}

func (rl *RalphLoop) appendMessage(msgType, body string) error {
	if rl.messagebus == nil {
		return nil
	}
	if strings.TrimSpace(msgType) == "" {
		return errors.New("message type is empty")
	}
	if strings.TrimSpace(body) == "" {
		return errors.New("message body is empty")
	}
	msg := &messagebus.Message{
		Type:      msgType,
		ProjectID: rl.projectID,
		TaskID:    rl.taskID,
		Body:      body,
	}
	if _, err := rl.messagebus.AppendMessage(msg); err != nil {
		return errors.Wrap(err, "append message")
	}
	return nil
}

func inferProjectTaskID(runDir string) (string, string) {
	clean := filepath.Clean(strings.TrimSpace(runDir))
	if clean == "." || clean == "" {
		return "", ""
	}
	taskID := filepath.Base(clean)
	if taskID == "." || taskID == string(filepath.Separator) {
		return "", ""
	}
	projectID := filepath.Base(filepath.Dir(clean))
	if projectID == "." || projectID == string(filepath.Separator) {
		return "", taskID
	}
	return projectID, taskID
}

func childRunIDs(children []ChildProcess) string {
	if len(children) == 0 {
		return "[]"
	}
	ids := make([]string, 0, len(children))
	for _, child := range children {
		if strings.TrimSpace(child.RunID) == "" {
			continue
		}
		ids = append(ids, child.RunID)
	}
	return fmt.Sprintf("%v", ids)
}
