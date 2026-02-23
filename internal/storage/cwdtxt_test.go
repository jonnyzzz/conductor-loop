package storage

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestParseCwdTxtBasic(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "cwd.txt")
	content := `RUN_ID=20260221-1507120000-12345-1
CWD=/workspace/myproject
AGENT=claude
CMD=claude --permission-mode bypassPermissions
STDOUT=/tmp/run/agent-stdout.txt
STDERR=/tmp/run/agent-stderr.txt
PID=5678
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write cwd.txt: %v", err)
	}

	info, err := ParseCwdTxt(path)
	if err != nil {
		t.Fatalf("ParseCwdTxt: %v", err)
	}
	if info.RunID != "20260221-1507120000-12345-1" {
		t.Errorf("RunID: got %q, want %q", info.RunID, "20260221-1507120000-12345-1")
	}
	if info.AgentType != "claude" {
		t.Errorf("AgentType: got %q, want %q", info.AgentType, "claude")
	}
	if info.PID != 5678 {
		t.Errorf("PID: got %d, want 5678", info.PID)
	}
	if info.CWD != "/workspace/myproject" {
		t.Errorf("CWD: got %q", info.CWD)
	}
	if info.Status != StatusRunning {
		t.Errorf("Status: got %q, want running (no EXIT_CODE)", info.Status)
	}
	if info.ExitCode != -1 {
		t.Errorf("ExitCode: got %d, want -1", info.ExitCode)
	}
	// ProjectID derived from CWD base
	if info.ProjectID != "myproject" {
		t.Errorf("ProjectID: got %q, want %q", info.ProjectID, "myproject")
	}
}

func TestParseCwdTxtWithExitCodeZero(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "cwd.txt")
	content := `RUN_ID=20260221-1507120000-99-2
CWD=/home/user/proj
AGENT=codex
PID=100
EXIT_CODE=0
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write cwd.txt: %v", err)
	}

	info, err := ParseCwdTxt(path)
	if err != nil {
		t.Fatalf("ParseCwdTxt: %v", err)
	}
	if info.Status != StatusCompleted {
		t.Errorf("Status: got %q, want completed", info.Status)
	}
	if info.ExitCode != 0 {
		t.Errorf("ExitCode: got %d, want 0", info.ExitCode)
	}
}

func TestParseCwdTxtWithExitCodeNonZero(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "cwd.txt")
	content := `RUN_ID=20260221-1507120000-99-3
CWD=/home/user/proj
AGENT=codex
PID=200
EXIT_CODE=1
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write cwd.txt: %v", err)
	}

	info, err := ParseCwdTxt(path)
	if err != nil {
		t.Fatalf("ParseCwdTxt: %v", err)
	}
	if info.Status != StatusFailed {
		t.Errorf("Status: got %q, want failed", info.Status)
	}
	if info.ExitCode != 1 {
		t.Errorf("ExitCode: got %d, want 1", info.ExitCode)
	}
}

func TestParseCwdTxtMissingRunID(t *testing.T) {
	dir := t.TempDir()
	runDir := filepath.Join(dir, "my-run-id")
	if err := os.MkdirAll(runDir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	path := filepath.Join(runDir, "cwd.txt")
	content := `CWD=/home/user/proj
AGENT=codex
PID=300
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write cwd.txt: %v", err)
	}

	info, err := ParseCwdTxt(path)
	if err != nil {
		t.Fatalf("ParseCwdTxt: %v", err)
	}
	// Falls back to parent directory name as run ID
	if info.RunID != "my-run-id" {
		t.Errorf("RunID fallback: got %q, want %q", info.RunID, "my-run-id")
	}
}

func TestParseCwdTxtNotFound(t *testing.T) {
	_, err := ParseCwdTxt("/nonexistent/path/cwd.txt")
	if err == nil {
		t.Fatal("expected error for nonexistent file")
	}
}

func TestParseCwdTxtEmptyCWD(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "cwd.txt")
	content := `RUN_ID=some-run
AGENT=gemini
PID=1
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write cwd.txt: %v", err)
	}

	info, err := ParseCwdTxt(path)
	if err != nil {
		t.Fatalf("ParseCwdTxt: %v", err)
	}
	// CWD is empty, ProjectID should be "unknown"
	if info.ProjectID != "unknown" {
		t.Errorf("ProjectID: got %q, want %q", info.ProjectID, "unknown")
	}
}

func TestParseCwdTxtSkipsBlankAndMalformedLines(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "cwd.txt")
	content := `
RUN_ID=run-abc
# not a key=value line
AGENT=xai

PID=42
CWD=/some/path
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write cwd.txt: %v", err)
	}

	info, err := ParseCwdTxt(path)
	if err != nil {
		t.Fatalf("ParseCwdTxt: %v", err)
	}
	if info.RunID != "run-abc" {
		t.Errorf("RunID: got %q", info.RunID)
	}
	if info.AgentType != "xai" {
		t.Errorf("AgentType: got %q", info.AgentType)
	}
	if info.PID != 42 {
		t.Errorf("PID: got %d", info.PID)
	}
}

func TestParseCwdTxtInvalidPID(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "cwd.txt")
	content := `RUN_ID=run-x
CWD=/tmp/p
AGENT=claude
PID=notanumber
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write cwd.txt: %v", err)
	}

	info, err := ParseCwdTxt(path)
	if err != nil {
		t.Fatalf("ParseCwdTxt: %v", err)
	}
	// Invalid PID parses as 0
	if info.PID != 0 {
		t.Errorf("PID: got %d, want 0", info.PID)
	}
}

func TestStartTimeFromRunID(t *testing.T) {
	tests := []struct {
		name     string
		runID    string
		wantUTC  time.Time
		wantZero bool
	}{
		{
			name:    "4-digit fractional seconds",
			runID:   "20260221-15071200001234-99",
			wantUTC: time.Date(2026, 2, 21, 15, 7, 12, 0, time.UTC),
		},
		{
			name:    "3-digit fractional seconds",
			runID:   "20260221-150712000-999",
			wantUTC: time.Date(2026, 2, 21, 15, 7, 12, 0, time.UTC),
		},
		{
			name:    "seconds only",
			runID:   "20260221-150712-5678",
			wantUTC: time.Date(2026, 2, 21, 15, 7, 12, 0, time.UTC),
		},
		{
			name:    "run_ prefix stripped",
			runID:   "run_20260221-150712-5678",
			wantUTC: time.Date(2026, 2, 21, 15, 7, 12, 0, time.UTC),
		},
		{
			name:     "invalid format returns zero",
			runID:    "not-a-valid-run-id",
			wantZero: true,
		},
		{
			name:     "empty string returns zero",
			runID:    "",
			wantZero: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := startTimeFromRunID(tc.runID)
			if tc.wantZero {
				if !got.IsZero() {
					t.Errorf("expected zero time, got %v", got)
				}
				return
			}
			if got.IsZero() {
				t.Errorf("expected non-zero time for %q, got zero", tc.runID)
				return
			}
			// Compare date/time components
			if got.Year() != tc.wantUTC.Year() || got.Month() != tc.wantUTC.Month() ||
				got.Day() != tc.wantUTC.Day() || got.Hour() != tc.wantUTC.Hour() ||
				got.Minute() != tc.wantUTC.Minute() || got.Second() != tc.wantUTC.Second() {
				t.Errorf("time mismatch: got %v, want %v", got, tc.wantUTC)
			}
		})
	}
}
