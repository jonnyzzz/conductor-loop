package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/jonnyzzz/conductor-loop/internal/runner"
)

// monitorLockPath returns the path to the PID lockfile for the given monitor scope.
// Scope is currently {root, project}.
func monitorLockPath(root, projectID string) string {
	return filepath.Join(root, projectID, ".monitor.pid")
}

// acquireMonitorLock attempts to acquire a PID lockfile for the monitor.
// Returns a cleanup function on success that removes the lockfile.
// Returns an error if a live monitor already holds the lock.
func acquireMonitorLock(root, projectID string) (release func(), err error) {
	lockPath := monitorLockPath(root, projectID)

	// Ensure the project directory exists.
	if err := os.MkdirAll(filepath.Dir(lockPath), 0o755); err != nil {
		return nil, fmt.Errorf("create monitor lock directory: %w", err)
	}

	// Check for an existing lockfile.
	if existing, readErr := os.ReadFile(lockPath); readErr == nil {
		existingPID, parseErr := parsePIDFromLock(string(existing))
		if parseErr == nil && existingPID > 0 {
			if runner.IsProcessAlive(existingPID) {
				return nil, fmt.Errorf(
					"monitor already running for project %q (PID %d); lockfile: %s\n"+
						"If the process is stale, remove the lockfile manually: rm %s",
					projectID, existingPID, lockPath, lockPath,
				)
			}
			// Stale lock: process is dead. Remove it and continue.
			_ = os.Remove(lockPath)
		} else {
			// Corrupt/unreadable lockfile; remove and continue.
			_ = os.Remove(lockPath)
		}
	}

	// Write our PID atomically: write to a temp file, then rename.
	pid := os.Getpid()
	content := strconv.Itoa(pid) + "\n"
	tmpPath := lockPath + ".tmp"
	if err := os.WriteFile(tmpPath, []byte(content), 0o644); err != nil {
		return nil, fmt.Errorf("write monitor lockfile: %w", err)
	}
	if err := os.Rename(tmpPath, lockPath); err != nil {
		_ = os.Remove(tmpPath)
		return nil, fmt.Errorf("install monitor lockfile: %w", err)
	}

	// Verify we actually own the lock (re-read and confirm our PID).
	if written, readErr := os.ReadFile(lockPath); readErr == nil {
		if got, _ := parsePIDFromLock(string(written)); got != pid {
			return nil, fmt.Errorf("monitor lock race: another process took the lock")
		}
	}

	release = func() {
		// Only remove if we still own it.
		if data, err := os.ReadFile(lockPath); err == nil {
			if got, _ := parsePIDFromLock(string(data)); got == pid {
				_ = os.Remove(lockPath)
			}
		}
	}
	return release, nil
}

// parsePIDFromLock extracts the integer PID from a lockfile's content.
func parsePIDFromLock(content string) (int, error) {
	s := strings.TrimSpace(content)
	pid, err := strconv.Atoi(s)
	if err != nil {
		return 0, fmt.Errorf("parse PID %q: %w", s, err)
	}
	return pid, nil
}
