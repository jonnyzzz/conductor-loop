package storage

import (
	"bufio"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

// ParseCwdTxt synthesizes a RunInfo from a cwd.txt file produced by run-agent.sh.
// The format is KEY=VALUE lines, e.g.:
//
//	RUN_ID=run_20260128-163814-2127
//	CWD=/Users/jonnyzzz/Work/...
//	AGENT=codex
//	CMD=codex exec ...
//	STDOUT=.../agent-stdout.txt
//	STDERR=.../agent-stderr.txt
//	PID=1234
//	EXIT_CODE=0   (only present when run has completed)
func ParseCwdTxt(path string) (*RunInfo, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	kv := make(map[string]string)
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}
		idx := strings.IndexByte(line, '=')
		if idx < 0 {
			continue
		}
		kv[line[:idx]] = line[idx+1:]
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}

	runID := kv["RUN_ID"]
	if runID == "" {
		// fall back to parent directory name
		runID = filepath.Base(filepath.Dir(path))
	}

	// Derive project/task from run folder name and CWD
	cwd := kv["CWD"]
	projectID := filepath.Base(cwd)
	if projectID == "" || projectID == "." {
		projectID = "unknown"
	}
	taskID := runID

	pid, _ := strconv.Atoi(kv["PID"])

	info := &RunInfo{
		RunID:       runID,
		ProjectID:   projectID,
		TaskID:      taskID,
		AgentType:   kv["AGENT"],
		PID:         pid,
		PGID:        pid,
		CWD:         cwd,
		CommandLine: kv["CMD"],
		StdoutPath:  kv["STDOUT"],
		StderrPath:  kv["STDERR"],
		PromptPath:  kv["PROMPT"],
		StartTime:   startTimeFromRunID(runID),
		ExitCode:    -1,
		Status:      StatusRunning,
	}

	if exitStr, ok := kv["EXIT_CODE"]; ok {
		code, err := strconv.Atoi(exitStr)
		if err == nil {
			info.ExitCode = code
			if code == 0 {
				info.Status = StatusCompleted
			} else {
				info.Status = StatusFailed
			}
			// use file mtime as end time
			if fi, err := os.Stat(path); err == nil {
				info.EndTime = fi.ModTime().UTC()
			}
		}
	}

	return info, nil
}

// startTimeFromRunID extracts a time from run IDs of the form
// YYYYMMDD-HHMMSSffff-PID or run_YYYYMMDD-HHMMSS-PID (legacy).
func startTimeFromRunID(runID string) time.Time {
	// Strip leading "run_"
	s := strings.TrimPrefix(runID, "run_")
	// Try 4-digit fractional seconds first (YYYYMMDD-HHMMSSffff)
	if len(s) >= 19 {
		t, err := time.ParseInLocation("20060102-1504050000", s[:19], time.UTC)
		if err == nil {
			return t
		}
	}
	// Fall back to 3-digit fractional seconds (YYYYMMDD-HHMMSSfff)
	if len(s) >= 18 {
		t, err := time.ParseInLocation("20060102-150405000", s[:18], time.UTC)
		if err == nil {
			return t
		}
	}
	// Fall back to seconds only (YYYYMMDD-HHMMSS)
	if len(s) >= 15 {
		t, err := time.ParseInLocation("20060102-150405", s[:15], time.UTC)
		if err == nil {
			return t
		}
	}
	return time.Time{}
}
