package local_test

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/jonnyzzz/conductor-loop/internal/storage"
)

func TestLocalAgents(t *testing.T) {
	if os.Getenv("RUN_LOCAL_AGENT_TESTS") != "1" {
		t.Skip("set RUN_LOCAL_AGENT_TESTS=1 to run real local agent tests")
	}

	_, file, _, _ := runtime.Caller(0)
	projectRoot := filepath.Join(filepath.Dir(file), "../..")

	binaryPath := filepath.Join(projectRoot, "bin", "run-agent")
	if _, err := os.Stat(binaryPath); err != nil {
		t.Skipf("bin/run-agent not found or not accessible: %v", err)
	}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		t.Fatalf("get home dir: %v", err)
	}
	configPath := filepath.Join(homeDir, ".run-agent", "conductor-loop.hcl")
	if _, err := os.Stat(configPath); err != nil {
		t.Skipf("config not found at %s: %v", configPath, err)
	}

	root := rootDir(t)

	agents := []struct {
		name    string
		timeout time.Duration
	}{
		{"claude", 3 * time.Minute},
		{"codex", 3 * time.Minute},
		{"gemini", 3 * time.Minute},
		{"perplexity", 2 * time.Minute},
	}

	for _, agent := range agents {
		agent := agent
		t.Run(agent.name, func(t *testing.T) {
			t.Parallel()

			projectID := "test-local"
			now := time.Now()
			taskID := fmt.Sprintf("task-%s-%s-%s", now.Format("20060102"), now.Format("150405"), agent.name)
			taskDir := filepath.Join(root, projectID, taskID)
			if err := os.MkdirAll(taskDir, 0o755); err != nil {
				t.Fatalf("mkdir task dir: %v", err)
			}

			ctx, cancel := context.WithTimeout(context.Background(), agent.timeout)
			defer cancel()

			cmd := exec.CommandContext(ctx,
				binaryPath,
				"job",
				"--config", configPath,
				"--root", root,
				"--project", projectID,
				"--task", taskID,
				"--agent", agent.name,
				"--cwd", taskDir,
				"--prompt", "Respond with the word OK.",
			)
			cmd.Stdout = os.Stdout
			cmd.Stderr = os.Stderr

			if err := cmd.Run(); err != nil {
				logRunFolder(t, taskDir)
				t.Fatalf("run-agent exited with error: %v", err)
			}

			runDir, err := findSingleRunDir(taskDir)
			if err != nil {
				logRunFolder(t, taskDir)
				t.Fatalf("find run dir: %v", err)
			}

			infoPath := filepath.Join(runDir, "run-info.yaml")
			info, err := storage.ReadRunInfo(infoPath)
			if err != nil {
				logRunFolder(t, taskDir)
				t.Fatalf("read run-info.yaml: %v", err)
			}

			if info.Status != storage.StatusCompleted {
				logRunFolder(t, taskDir)
				t.Fatalf("expected status=%s, got %s", storage.StatusCompleted, info.Status)
			}
			if info.ExitCode != 0 {
				logRunFolder(t, taskDir)
				t.Fatalf("expected exit_code=0, got %d", info.ExitCode)
			}

			stdoutPath := filepath.Join(runDir, "agent-stdout.txt")
			data, err := os.ReadFile(stdoutPath)
			if err != nil {
				logRunFolder(t, taskDir)
				t.Fatalf("read agent-stdout.txt: %v", err)
			}
			if len(data) == 0 {
				logRunFolder(t, taskDir)
				t.Fatalf("agent-stdout.txt is empty")
			}

			logRunFolder(t, taskDir)
		})
	}
}

// rootDir returns the test root directory. If RUN_LOCAL_ROOT is set, it uses
// that fixed path (useful for manual post-run inspection). Otherwise it uses
// t.TempDir() which is auto-cleaned.
func rootDir(t *testing.T) string {
	t.Helper()
	if fixed := os.Getenv("RUN_LOCAL_ROOT"); fixed != "" {
		if err := os.MkdirAll(fixed, 0o755); err != nil {
			t.Fatalf("mkdir fixed root %s: %v", fixed, err)
		}
		return fixed
	}
	return t.TempDir()
}

func findSingleRunDir(taskDir string) (string, error) {
	runsDir := filepath.Join(taskDir, "runs")
	entries, err := os.ReadDir(runsDir)
	if err != nil {
		return "", fmt.Errorf("read runs dir %s: %w", runsDir, err)
	}
	var dirs []string
	for _, e := range entries {
		if e.IsDir() {
			dirs = append(dirs, filepath.Join(runsDir, e.Name()))
		}
	}
	if len(dirs) == 0 {
		return "", fmt.Errorf("no run directories found under %s", runsDir)
	}
	return dirs[0], nil
}

func logRunFolder(t *testing.T, taskDir string) {
	t.Helper()
	runsDir := filepath.Join(taskDir, "runs")
	t.Logf("=== run folder: %s ===", runsDir)

	_ = filepath.WalkDir(runsDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		rel, _ := filepath.Rel(runsDir, path)
		if d.IsDir() {
			t.Logf("  dir:  %s/", rel)
			return nil
		}
		info, statErr := d.Info()
		size := int64(-1)
		if statErr == nil {
			size = info.Size()
		}
		t.Logf("  file: %s (%d bytes)", rel, size)
		return nil
	})

	_ = filepath.WalkDir(runsDir, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		switch filepath.Base(path) {
		case "agent-stdout.txt":
			data, readErr := os.ReadFile(path)
			if readErr == nil {
				preview := string(data)
				if len(preview) > 500 {
					preview = preview[:500] + "...(truncated)"
				}
				t.Logf("=== agent-stdout.txt ===\n%s", preview)
			}
		case "run-info.yaml":
			data, readErr := os.ReadFile(path)
			if readErr == nil {
				t.Logf("=== run-info.yaml ===\n%s", string(data))
			}
		}
		return nil
	})
}
