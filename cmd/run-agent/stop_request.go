package main

import (
	"fmt"
	"os"
	"path/filepath"
	"time"
)

const (
	// StopRequestedFile is the marker file written to the task directory when
	// run-agent stop is issued. Monitor loops check this file before restarting.
	StopRequestedFile = "STOP-REQUESTED"

	// StopRequestedDefaultWindow is the default suppression window: monitor
	// loops will not restart a task for this long after stop is issued.
	StopRequestedDefaultWindow = 60 * time.Second

	// stopRequestedPermanent is the sentinel written by --no-restart to signal
	// permanent suppression (no auto-restart ever).
	stopRequestedPermanent = "permanent"
)

// stopRequestPath returns the path to the STOP-REQUESTED marker for a task.
func stopRequestPath(root, projectID, taskID string) string {
	return filepath.Join(root, projectID, taskID, StopRequestedFile)
}

// writeStopRequest writes a STOP-REQUESTED marker to the task directory.
// If permanent is true, the marker signals indefinite suppression.
// Otherwise it encodes the current time so monitors can check the expiry window.
func writeStopRequest(root, projectID, taskID string, permanent bool) error {
	markerPath := stopRequestPath(root, projectID, taskID)

	// Ensure the task directory exists.
	if err := os.MkdirAll(filepath.Dir(markerPath), 0o755); err != nil {
		return fmt.Errorf("create task directory for stop-request: %w", err)
	}

	var content string
	if permanent {
		content = stopRequestedPermanent
	} else {
		content = time.Now().UTC().Format(time.RFC3339)
	}

	if err := os.WriteFile(markerPath, []byte(content+"\n"), 0o644); err != nil {
		return fmt.Errorf("write stop-request marker: %w", err)
	}
	return nil
}

// removeStopRequest removes the STOP-REQUESTED marker from the task directory.
// Returns nil if the file does not exist.
func removeStopRequest(root, projectID, taskID string) error {
	markerPath := stopRequestPath(root, projectID, taskID)
	if err := os.Remove(markerPath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("remove stop-request marker: %w", err)
	}
	return nil
}

// checkStopRequested returns true if a valid stop suppression is in effect for
// the given task. It respects the suppression window: if the marker is older
// than window, it auto-removes it and returns false.
func checkStopRequested(root, projectID, taskID string, window time.Duration) bool {
	markerPath := stopRequestPath(root, projectID, taskID)
	data, err := os.ReadFile(markerPath)
	if err != nil {
		// File absent → no suppression.
		return false
	}

	content := string(data)
	// Trim whitespace for comparison.
	trimmed := trimNewlines(content)

	if trimmed == stopRequestedPermanent {
		return true
	}

	// Parse timestamp.
	t, parseErr := time.Parse(time.RFC3339, trimmed)
	if parseErr != nil {
		// Corrupt marker; remove and allow restart.
		_ = os.Remove(markerPath)
		return false
	}

	if time.Since(t) < window {
		return true
	}

	// Window expired: clean up the marker.
	_ = os.Remove(markerPath)
	return false
}

func trimNewlines(s string) string {
	result := s
	for len(result) > 0 && (result[len(result)-1] == '\n' || result[len(result)-1] == '\r' || result[len(result)-1] == ' ' || result[len(result)-1] == '\t') {
		result = result[:len(result)-1]
	}
	for len(result) > 0 && (result[0] == '\n' || result[0] == '\r' || result[0] == ' ' || result[0] == '\t') {
		result = result[1:]
	}
	return result
}
