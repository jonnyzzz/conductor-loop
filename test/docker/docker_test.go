package docker_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jonnyzzz/conductor-loop/internal/messagebus"
	"github.com/jonnyzzz/conductor-loop/internal/storage"
)

const (
	imageName       = "conductor:test"
	healthURL       = "http://localhost:8080/api/v1/health"
	secondaryHealth = "http://localhost:8081/api/v1/health"
)

var buildOnce sync.Once
var buildErr error

func TestDockerBuild(t *testing.T) {
	root := repoRoot(t)
	ensureDocker(t)
	buildDockerImage(t, root)

	cmd := exec.Command("docker", "image", "inspect", imageName, "--format", "{{.Size}}")
	cmd.Dir = root
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("inspect image size: %v (%s)", err, strings.TrimSpace(string(output)))
	}
	sizeStr := strings.TrimSpace(string(output))
	size, err := strconv.ParseInt(sizeStr, 10, 64)
	if err != nil {
		t.Fatalf("parse image size %q: %v", sizeStr, err)
	}
	const maxSize = 500 * 1024 * 1024
	if size > maxSize {
		t.Fatalf("image size too large: %d bytes", size)
	}
}

func TestDockerRun(t *testing.T) {
	root := repoRoot(t)
	ensureDocker(t)
	ensureDockerConfig(t, root)
	ensureRunsDir(t, root)

	project := uniqueProjectName(t)
	compose := composeCommand(t)
	composeUp(t, root, project, compose, "conductor")
	defer composeDown(t, root, project, compose)

	waitForHTTP(t, healthURL)
}

func TestDockerPersistence(t *testing.T) {
	root := repoRoot(t)
	ensureDocker(t)
	ensureDockerConfig(t, root)
	ensureRunsDir(t, root)

	project := uniqueProjectName(t)
	compose := composeCommand(t)
	composeUp(t, root, project, compose, "conductor")
	defer composeDown(t, root, project, compose)

	waitForHTTP(t, healthURL)

	runID := fmt.Sprintf("run-%d", time.Now().UnixNano())
	writeRunInfo(t, root, "proj", "task-001", runID)

	assertRunExists(t, runID, "http://localhost:8080/api/v1/runs/")

	composeRestart(t, root, project, compose, "conductor")
	waitForHTTP(t, healthURL)
	assertRunExists(t, runID, "http://localhost:8080/api/v1/runs/")
}

func TestDockerNetworkIsolation(t *testing.T) {
	root := repoRoot(t)
	ensureDocker(t)
	ensureDockerConfig(t, root)
	ensureRunsDir(t, root)

	project := uniqueProjectName(t)
	compose := composeCommand(t)
	composeUp(t, root, project, compose, "conductor")
	defer composeDown(t, root, project, compose)

	waitForHTTP(t, healthURL)
	composeExec(t, root, project, compose, "conductor", []string{"curl", "-f", "http://conductor:8080/api/v1/health"})
}

func TestDockerVolumes(t *testing.T) {
	root := repoRoot(t)
	ensureDocker(t)
	ensureDockerConfig(t, root)
	ensureRunsDir(t, root)

	project := uniqueProjectName(t)
	compose := composeCommand(t)
	composeUp(t, root, project, compose, "conductor")
	defer composeDown(t, root, project, compose)

	markerDir := filepath.Join(root, "data", "runs", "volume-test")
	markerPath := filepath.Join(markerDir, "marker.txt")
	if err := os.MkdirAll(markerDir, 0o755); err != nil {
		t.Fatalf("mkdir marker dir: %v", err)
	}
	if err := os.WriteFile(markerPath, []byte("ok"), 0o644); err != nil {
		t.Fatalf("write marker file: %v", err)
	}

	composeExec(t, root, project, compose, "conductor", []string{"test", "-f", "/data/runs/volume-test/marker.txt"})
}

func TestDockerLogs(t *testing.T) {
	root := repoRoot(t)
	ensureDocker(t)
	ensureDockerConfig(t, root)
	ensureRunsDir(t, root)

	project := uniqueProjectName(t)
	compose := composeCommand(t)
	composeUp(t, root, project, compose, "conductor")
	defer composeDown(t, root, project, compose)

	waitForHTTP(t, healthURL)

	args := append(compose[1:], "-p", project, "logs", "conductor")
	cmd := exec.Command(compose[0], args...)
	cmd.Dir = root
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("compose logs failed: %v (%s)", err, strings.TrimSpace(string(output)))
	}
}

func TestDockerMultiContainer(t *testing.T) {
	root := repoRoot(t)
	ensureDocker(t)
	ensureDockerConfig(t, root)
	ensureRunsDir(t, root)
	buildDockerImage(t, root)

	project := uniqueProjectName(t)
	compose := composeCommand(t)
	composeUp(t, root, project, compose, "conductor")
	defer composeDown(t, root, project, compose)

	waitForHTTP(t, healthURL)

	secondaryName := fmt.Sprintf("conductor-secondary-%s", project)
	secondaryPort := "8081:8080"
	network := fmt.Sprintf("%s_conductor-net", project)
	args := []string{
		"run", "-d",
		"--name", secondaryName,
		"--network", network,
		"-p", secondaryPort,
		"-v", filepath.Join(root, "data", "runs") + ":/data/runs",
		"-v", filepath.Join(root, "config.yaml") + ":/data/config/config.yaml:ro",
		"-e", "CONDUCTOR_CONFIG=/data/config/config.yaml",
		imageName,
	}
	cmd := exec.Command("docker", args...)
	cmd.Dir = root
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("start secondary container: %v (%s)", err, strings.TrimSpace(string(output)))
	}
	defer func() {
		_ = exec.Command("docker", "rm", "-f", secondaryName).Run()
	}()

	waitForHTTP(t, secondaryHealth)

	runID := fmt.Sprintf("run-%d", time.Now().UnixNano())
	writeRunInfo(t, root, "proj", "task-002", runID)

	assertRunExists(t, runID, "http://localhost:8080/api/v1/runs/")
	assertRunExists(t, runID, "http://localhost:8081/api/v1/runs/")

	msgID := writeProjectMessage(t, root, "proj", "multi-container")
	if msgID == "" {
		t.Fatalf("expected msg id")
	}
	assertMessageVisible(t, "http://localhost:8080/api/v1/messages?project_id=proj", msgID)
	assertMessageVisible(t, "http://localhost:8081/api/v1/messages?project_id=proj", msgID)
}

func ensureDocker(t *testing.T) {
	if _, err := exec.LookPath("docker"); err != nil {
		t.Skip("docker not available")
	}
	cmd := exec.Command("docker", "info")
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Skipf("docker not running: %v (%s)", err, strings.TrimSpace(string(output)))
	}
}

func buildDockerImage(t *testing.T, root string) {
	t.Helper()
	buildOnce.Do(func() {
		ctx, cancel := context.WithTimeout(context.Background(), 10*time.Minute)
		defer cancel()
		cmd := exec.CommandContext(ctx, "docker", "build", "-t", imageName, ".")
		cmd.Dir = root
		output, err := cmd.CombinedOutput()
		if ctx.Err() != nil {
			buildErr = ctx.Err()
			return
		}
		if err != nil {
			buildErr = fmt.Errorf("docker build failed: %w (%s)", err, strings.TrimSpace(string(output)))
		}
	})
	if buildErr != nil {
		t.Fatalf("docker build failed: %v", buildErr)
	}
}

func composeCommand(t *testing.T) []string {
	t.Helper()
	if _, err := exec.LookPath("docker-compose"); err == nil {
		return []string{"docker-compose"}
	}
	if _, err := exec.LookPath("docker"); err != nil {
		t.Skip("docker compose not available")
	}
	cmd := exec.Command("docker", "compose", "version")
	if output, err := cmd.CombinedOutput(); err == nil {
		_ = output
		return []string{"docker", "compose"}
	}
	t.Skip("docker compose not available")
	return nil
}

func composeUp(t *testing.T, root, project string, compose []string, services ...string) {
	t.Helper()
	args := append(compose[1:], "-p", project, "up", "-d")
	args = append(args, services...)
	cmd := exec.Command(compose[0], args...)
	cmd.Dir = root
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("compose up failed: %v (%s)", err, strings.TrimSpace(string(output)))
	}
}

func composeDown(t *testing.T, root, project string, compose []string) {
	t.Helper()
	args := append(compose[1:], "-p", project, "down", "-v", "--remove-orphans")
	cmd := exec.Command(compose[0], args...)
	cmd.Dir = root
	_ = cmd.Run()
}

func composeRestart(t *testing.T, root, project string, compose []string, service string) {
	t.Helper()
	args := append(compose[1:], "-p", project, "restart")
	if service != "" {
		args = append(args, service)
	}
	cmd := exec.Command(compose[0], args...)
	cmd.Dir = root
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("compose restart failed: %v (%s)", err, strings.TrimSpace(string(output)))
	}
}

func composeExec(t *testing.T, root, project string, compose []string, service string, cmdArgs []string) {
	t.Helper()
	args := append(compose[1:], "-p", project, "exec", "-T", service)
	args = append(args, cmdArgs...)
	cmd := exec.Command(compose[0], args...)
	cmd.Dir = root
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("compose exec failed: %v (%s)", err, strings.TrimSpace(string(output)))
	}
}

func repoRoot(t *testing.T) string {
	t.Helper()
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	for i := 0; i < 6; i++ {
		if _, err := os.Stat(filepath.Join(wd, "go.mod")); err == nil {
			return wd
		}
		parent := filepath.Dir(wd)
		if parent == wd {
			break
		}
		wd = parent
	}
	t.Fatalf("repo root not found")
	return ""
}

func uniqueProjectName(t *testing.T) string {
	t.Helper()
	stamp := time.Now().UnixNano()
	return fmt.Sprintf("conductor-test-%d", stamp)
}

func ensureDockerConfig(t *testing.T, root string) {
	t.Helper()
	src := filepath.Join(root, "config.docker.yaml")
	dst := filepath.Join(root, "config.yaml")
	data, err := os.ReadFile(src)
	if err != nil {
		t.Fatalf("read config.docker.yaml: %v", err)
	}
	if existing, err := os.ReadFile(dst); err == nil {
		// Preserve original config.yaml
		if err := os.WriteFile(dst, data, 0o644); err != nil {
			t.Fatalf("write config.yaml: %v", err)
		}
		t.Cleanup(func() {
			_ = os.WriteFile(dst, existing, 0o644)
		})
		return
	}
	if err := os.WriteFile(dst, data, 0o644); err != nil {
		t.Fatalf("write config.yaml: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Remove(dst)
	})
}

func ensureRunsDir(t *testing.T, root string) {
	t.Helper()
	path := filepath.Join(root, "data", "runs")
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatalf("mkdir runs dir: %v", err)
	}
}

func waitForHTTP(t *testing.T, url string) {
	t.Helper()
	deadline := time.Now().Add(45 * time.Second)
	var lastErr error
	client := &http.Client{Timeout: 2 * time.Second}
	for time.Now().Before(deadline) {
		resp, err := client.Get(url)
		if err == nil {
			io.Copy(io.Discard, resp.Body)
			resp.Body.Close()
			if resp.StatusCode == http.StatusOK {
				return
			}
			lastErr = fmt.Errorf("status %d", resp.StatusCode)
		} else {
			lastErr = err
		}
		time.Sleep(500 * time.Millisecond)
	}
	t.Fatalf("timeout waiting for %s: %v", url, lastErr)
}

func writeRunInfo(t *testing.T, root, projectID, taskID, runID string) {
	t.Helper()
	runDir := filepath.Join(root, "data", "runs", projectID, taskID, "runs", runID)
	if err := os.MkdirAll(runDir, 0o755); err != nil {
		t.Fatalf("mkdir run dir: %v", err)
	}
	info := &storage.RunInfo{
		Version:   1,
		RunID:     runID,
		ProjectID: projectID,
		TaskID:    taskID,
		AgentType: "codex",
		PID:       123,
		PGID:      123,
		StartTime: time.Now().Add(-time.Minute).UTC(),
		EndTime:   time.Now().UTC(),
		ExitCode:  0,
		Status:    storage.StatusCompleted,
	}
	if err := storage.WriteRunInfo(filepath.Join(runDir, "run-info.yaml"), info); err != nil {
		t.Fatalf("write run-info: %v", err)
	}
}

func assertRunExists(t *testing.T, runID, baseURL string) {
	t.Helper()
	url := baseURL + runID
	resp, err := http.Get(url)
	if err != nil {
		t.Fatalf("get run: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		t.Fatalf("unexpected status %d (%s)", resp.StatusCode, strings.TrimSpace(string(body)))
	}
}

func writeProjectMessage(t *testing.T, root, projectID, body string) string {
	t.Helper()
	busPath := filepath.Join(root, "data", "runs", projectID, "PROJECT-MESSAGE-BUS.md")
	if err := os.MkdirAll(filepath.Dir(busPath), 0o755); err != nil {
		t.Fatalf("mkdir bus dir: %v", err)
	}
	bus, err := messagebus.NewMessageBus(busPath)
	if err != nil {
		t.Fatalf("new message bus: %v", err)
	}
	msg := &messagebus.Message{
		Type:      "FACT",
		ProjectID: projectID,
		Body:      body,
	}
	msgID, err := bus.AppendMessage(msg)
	if err != nil {
		t.Fatalf("append message: %v", err)
	}
	return msgID
}

func assertMessageVisible(t *testing.T, url, msgID string) {
	t.Helper()
	deadline := time.Now().Add(10 * time.Second)
	var lastPayload []byte
	for time.Now().Before(deadline) {
		resp, err := http.Get(url)
		if err != nil {
			time.Sleep(300 * time.Millisecond)
			continue
		}
		if resp.StatusCode != http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			resp.Body.Close()
			lastPayload = body
			time.Sleep(300 * time.Millisecond)
			continue
		}
		var payload struct {
			Messages []struct {
				MsgID string `json:"msg_id"`
			} `json:"messages"`
		}
		if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
			resp.Body.Close()
			time.Sleep(300 * time.Millisecond)
			continue
		}
		resp.Body.Close()
		for _, msg := range payload.Messages {
			if msg.MsgID == msgID {
				return
			}
		}
		lastPayload, _ = json.Marshal(payload)
		time.Sleep(300 * time.Millisecond)
	}
	if len(lastPayload) == 0 {
		lastPayload = []byte("<empty>")
	}
	t.Fatalf("message %s not found (%s)", msgID, string(bytes.TrimSpace(lastPayload)))
}
