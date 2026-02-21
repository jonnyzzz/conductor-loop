package docker_test

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
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

// testImage is unique per process — no collisions when multiple go test runs share a Docker daemon.
var testImage = fmt.Sprintf("conductor:test-%d", os.Getpid())

var buildOnce sync.Once
var buildErr error

// TestMain runs the test suite and removes the per-process image afterwards.
func TestMain(m *testing.M) {
	code := m.Run()
	// Best-effort cleanup — ignore errors (image may not exist if build was skipped).
	_ = exec.Command("docker", "rmi", "-f", testImage).Run()
	os.Exit(code)
}

// testEnv holds per-test isolation: unique project name, free port, temp dirs.
// Every test gets its own data directory and config file so tests can run in parallel
// on the same host without interfering with each other or dirtying the repository tree.
type testEnv struct {
	root       string
	hostPort   int
	dataDir    string // isolated /data/runs mount
	configPath string // isolated config.yaml copy
	project    string
	compose    []string
}

func newTestEnv(t *testing.T) *testEnv {
	t.Helper()
	root := repoRoot(t)

	// Isolated data directory — cleaned up automatically by t.Cleanup via t.TempDir.
	dataDir := t.TempDir()

	// Copy config.docker.yaml to an isolated temp file.
	configDir := t.TempDir()
	configPath := filepath.Join(configDir, "config.yaml")
	data, err := os.ReadFile(filepath.Join(root, "config.docker.yaml"))
	if err != nil {
		t.Fatalf("read config.docker.yaml: %v", err)
	}
	if err := os.WriteFile(configPath, data, 0o644); err != nil {
		t.Fatalf("write isolated config: %v", err)
	}

	return &testEnv{
		root:       root,
		hostPort:   findFreePort(t),
		dataDir:    dataDir,
		configPath: configPath,
		project:    uniqueProjectName(t),
		compose:    composeCommand(t),
	}
}

// composeEnv returns the environment for all docker compose invocations.
// All variable paths are unique to this testEnv — safe to run in parallel.
func (e *testEnv) composeEnv() []string {
	return append(os.Environ(),
		fmt.Sprintf("CONDUCTOR_HOST_PORT=%d", e.hostPort),
		fmt.Sprintf("CONDUCTOR_DATA_DIR=%s", e.dataDir),
		fmt.Sprintf("CONDUCTOR_CONFIG_PATH=%s", e.configPath),
		fmt.Sprintf("CONDUCTOR_IMAGE=%s", testImage),
	)
}

func (e *testEnv) healthURL() string {
	return fmt.Sprintf("http://localhost:%d/api/v1/health", e.hostPort)
}

func (e *testEnv) runsURL() string {
	return fmt.Sprintf("http://localhost:%d/api/v1/runs/", e.hostPort)
}

// findFreePort asks the OS for an unused TCP port and returns it.
func findFreePort(t *testing.T) int {
	t.Helper()
	ln, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("find free port: %v", err)
	}
	port := ln.Addr().(*net.TCPAddr).Port
	ln.Close()
	return port
}

func TestDockerBuild(t *testing.T) {
	root := repoRoot(t)
	ensureDocker(t)
	buildDockerImage(t, root)

	cmd := exec.Command("docker", "image", "inspect", testImage, "--format", "{{.Size}}")
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
	ensureDocker(t)
	env := newTestEnv(t)
	buildDockerImage(t, env.root)
	composeUp(t, env, "conductor")
	defer composeDown(t, env)
	waitForHTTP(t, env.healthURL())
}

func TestDockerPersistence(t *testing.T) {
	ensureDocker(t)
	env := newTestEnv(t)
	buildDockerImage(t, env.root)
	composeUp(t, env, "conductor")
	defer composeDown(t, env)

	waitForHTTP(t, env.healthURL())

	runID := fmt.Sprintf("run-%d", time.Now().UnixNano())
	writeRunInfo(t, env.dataDir, "proj", "task-001", runID)
	assertRunExists(t, runID, env.runsURL())

	composeRestart(t, env, "conductor")
	waitForHTTP(t, env.healthURL())
	assertRunExists(t, runID, env.runsURL())
}

func TestDockerNetworkIsolation(t *testing.T) {
	ensureDocker(t)
	env := newTestEnv(t)
	buildDockerImage(t, env.root)
	composeUp(t, env, "conductor")
	defer composeDown(t, env)

	waitForHTTP(t, env.healthURL())
	composeExec(t, env, "conductor", []string{"curl", "-f", "http://conductor:8080/api/v1/health"})
}

func TestDockerVolumes(t *testing.T) {
	ensureDocker(t)
	env := newTestEnv(t)
	buildDockerImage(t, env.root)
	composeUp(t, env, "conductor")
	defer composeDown(t, env)

	markerDir := filepath.Join(env.dataDir, "volume-test")
	markerPath := filepath.Join(markerDir, "marker.txt")
	if err := os.MkdirAll(markerDir, 0o755); err != nil {
		t.Fatalf("mkdir marker dir: %v", err)
	}
	if err := os.WriteFile(markerPath, []byte("ok"), 0o644); err != nil {
		t.Fatalf("write marker file: %v", err)
	}

	composeExec(t, env, "conductor", []string{"test", "-f", "/data/runs/volume-test/marker.txt"})
}

func TestDockerLogs(t *testing.T) {
	ensureDocker(t)
	env := newTestEnv(t)
	buildDockerImage(t, env.root)
	composeUp(t, env, "conductor")
	defer composeDown(t, env)

	waitForHTTP(t, env.healthURL())

	args := append(env.compose[1:], "-p", env.project, "logs", "conductor")
	cmd := exec.Command(env.compose[0], args...)
	cmd.Dir = env.root
	cmd.Env = env.composeEnv()
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("compose logs failed: %v (%s)", err, strings.TrimSpace(string(output)))
	}
}

func TestDockerMultiContainer(t *testing.T) {
	ensureDocker(t)
	env := newTestEnv(t)
	buildDockerImage(t, env.root)
	composeUp(t, env, "conductor")
	defer composeDown(t, env)

	waitForHTTP(t, env.healthURL())

	secondaryPort := findFreePort(t)
	secondaryName := fmt.Sprintf("conductor-secondary-%s", env.project)
	network := fmt.Sprintf("%s_conductor-net", env.project)
	args := []string{
		"run", "-d",
		"--name", secondaryName,
		"--network", network,
		"-p", fmt.Sprintf("%d:8080", secondaryPort),
		"-v", env.dataDir + ":/data/runs",
		"-v", env.configPath + ":/data/config/config.yaml:ro",
		"-e", "CONDUCTOR_CONFIG=/data/config/config.yaml",
		testImage,
	}
	cmd := exec.Command("docker", args...)
	cmd.Dir = env.root
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("start secondary container: %v (%s)", err, strings.TrimSpace(string(output)))
	}
	defer func() {
		_ = exec.Command("docker", "rm", "-f", secondaryName).Run()
	}()

	secondaryRunsURL := fmt.Sprintf("http://localhost:%d/api/v1/runs/", secondaryPort)
	waitForHTTP(t, fmt.Sprintf("http://localhost:%d/api/v1/health", secondaryPort))

	runID := fmt.Sprintf("run-%d", time.Now().UnixNano())
	writeRunInfo(t, env.dataDir, "proj", "task-002", runID)

	assertRunExists(t, runID, env.runsURL())
	assertRunExists(t, runID, secondaryRunsURL)

	msgID := writeProjectMessage(t, env.dataDir, "proj", "multi-container")
	if msgID == "" {
		t.Fatalf("expected msg id")
	}
	assertMessageVisible(t, fmt.Sprintf("http://localhost:%d/api/v1/messages?project_id=proj", env.hostPort), msgID)
	assertMessageVisible(t, fmt.Sprintf("http://localhost:%d/api/v1/messages?project_id=proj", secondaryPort), msgID)
}

// ── helpers ──────────────────────────────────────────────────────────────────

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
		cmd := exec.CommandContext(ctx, "docker", "build", "-t", testImage, ".")
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

func composeUp(t *testing.T, env *testEnv, services ...string) {
	t.Helper()
	args := append(env.compose[1:], "-p", env.project, "up", "-d")
	args = append(args, services...)
	cmd := exec.Command(env.compose[0], args...)
	cmd.Dir = env.root
	cmd.Env = env.composeEnv()
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("compose up failed: %v (%s)", err, strings.TrimSpace(string(output)))
	}
}

func composeDown(t *testing.T, env *testEnv) {
	t.Helper()
	args := append(env.compose[1:], "-p", env.project, "down", "-v", "--remove-orphans")
	cmd := exec.Command(env.compose[0], args...)
	cmd.Dir = env.root
	cmd.Env = env.composeEnv()
	_ = cmd.Run()
}

func composeRestart(t *testing.T, env *testEnv, service string) {
	t.Helper()
	args := append(env.compose[1:], "-p", env.project, "restart")
	if service != "" {
		args = append(args, service)
	}
	cmd := exec.Command(env.compose[0], args...)
	cmd.Dir = env.root
	cmd.Env = env.composeEnv()
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("compose restart failed: %v (%s)", err, strings.TrimSpace(string(output)))
	}
}

func composeExec(t *testing.T, env *testEnv, service string, cmdArgs []string) {
	t.Helper()
	args := append(env.compose[1:], "-p", env.project, "exec", "-T", service)
	args = append(args, cmdArgs...)
	cmd := exec.Command(env.compose[0], args...)
	cmd.Dir = env.root
	cmd.Env = env.composeEnv()
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
	return fmt.Sprintf("ctest-%d", time.Now().UnixNano())
}

func waitForHTTP(t *testing.T, url string) {
	t.Helper()
	deadline := time.Now().Add(45 * time.Second)
	var lastErr error
	client := &http.Client{Timeout: 2 * time.Second}
	for time.Now().Before(deadline) {
		resp, err := client.Get(url)
		if err == nil {
			_, _ = io.Copy(io.Discard, resp.Body)
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

func writeRunInfo(t *testing.T, dataDir, projectID, taskID, runID string) {
	t.Helper()
	runDir := filepath.Join(dataDir, projectID, taskID, "runs", runID)
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

func writeProjectMessage(t *testing.T, dataDir, projectID, body string) string {
	t.Helper()
	busPath := filepath.Join(dataDir, projectID, "PROJECT-MESSAGE-BUS.md")
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
