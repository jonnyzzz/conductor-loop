package performance_test

import (
	"bufio"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jonnyzzz/conductor-loop/internal/api"
	"github.com/jonnyzzz/conductor-loop/internal/config"
	"github.com/jonnyzzz/conductor-loop/internal/messagebus"
	"github.com/jonnyzzz/conductor-loop/internal/runner"
	"github.com/jonnyzzz/conductor-loop/internal/storage"
)

const (
	messageBusThroughputTarget = 1000.0
	messageBusThroughputSecs   = 10
	messageBusWriterCount      = 10

	runCreationTarget = 100.0
	runCreationSecs   = 5

	concurrentAgentsTarget = 5 * time.Minute
	concurrentAgentsCount  = 50

	sseLatencyTarget = 200 * time.Millisecond

	envPerfMessageBusTarget  = "PERF_MESSAGEBUS_TARGET"
	envPerfRunCreationTarget = "PERF_RUN_CREATION_TARGET"
	envPerfSSELatencyMs      = "PERF_SSE_LATENCY_MS"
	envPerfStubDoneFile      = "PERF_STUB_DONE_FILE"
	envPerfStubSleepMs       = "PERF_STUB_SLEEP_MS"
)

func BenchmarkMessageBusWrite(b *testing.B) {
	bus := newMessageBus(b)
	template := messagebus.Message{
		Type:      "FACT",
		ProjectID: "project",
		TaskID:    "task",
		RunID:     "run",
		Body:      "benchmark message",
	}
	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		msg := template
		if _, err := bus.AppendMessage(&msg); err != nil {
			b.Fatalf("append message: %v", err)
		}
	}
}

func BenchmarkMessageBusReadAll(b *testing.B) {
	bus := setupMessageBusWithMessages(b, 1000)
	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		if _, err := bus.ReadMessages(""); err != nil {
			b.Fatalf("read messages: %v", err)
		}
	}
}

func BenchmarkRunCreation(b *testing.B) {
	st := newStorage(b)
	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		if _, err := st.CreateRun("project", fmt.Sprintf("task-%d", i), "codex"); err != nil {
			b.Fatalf("create run: %v", err)
		}
	}
}

func BenchmarkRunInfoWrite(b *testing.B) {
	path := filepath.Join(b.TempDir(), "run-info.yaml")
	info := &storage.RunInfo{
		Version:   1,
		RunID:     "run-1",
		ProjectID: "project",
		TaskID:    "task",
		AgentType: "codex",
		PID:       123,
		PGID:      123,
		StartTime: time.Now().UTC(),
		Status:    storage.StatusRunning,
		ExitCode:  -1,
	}
	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		info.RunID = fmt.Sprintf("run-%d", i)
		if err := storage.WriteRunInfo(path, info); err != nil {
			b.Fatalf("write run-info: %v", err)
		}
	}
}

func BenchmarkRunInfoRead(b *testing.B) {
	path := filepath.Join(b.TempDir(), "run-info.yaml")
	info := &storage.RunInfo{
		Version:   1,
		RunID:     "run-1",
		ProjectID: "project",
		TaskID:    "task",
		AgentType: "codex",
		PID:       123,
		PGID:      123,
		StartTime: time.Now().UTC(),
		Status:    storage.StatusRunning,
		ExitCode:  -1,
	}
	if err := storage.WriteRunInfo(path, info); err != nil {
		b.Fatalf("write run-info: %v", err)
	}
	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		if _, err := storage.ReadRunInfo(path); err != nil {
			b.Fatalf("read run-info: %v", err)
		}
	}
}

func BenchmarkRunInfoConcurrentReadWrite(b *testing.B) {
	path := filepath.Join(b.TempDir(), "run-info.yaml")
	info := &storage.RunInfo{
		Version:   1,
		RunID:     "run-1",
		ProjectID: "project",
		TaskID:    "task",
		AgentType: "codex",
		PID:       123,
		PGID:      123,
		StartTime: time.Now().UTC(),
		Status:    storage.StatusRunning,
		ExitCode:  -1,
	}
	if err := storage.WriteRunInfo(path, info); err != nil {
		b.Fatalf("write run-info: %v", err)
	}

	var counter atomic.Int64
	b.ReportAllocs()
	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			seq := counter.Add(1)
			if seq%2 == 0 {
				copyInfo := *info
				copyInfo.RunID = fmt.Sprintf("run-%d", seq)
				if err := storage.WriteRunInfo(path, &copyInfo); err != nil {
					b.Fatalf("write run-info: %v", err)
				}
				continue
			}
			if _, err := storage.ReadRunInfo(path); err != nil {
				b.Fatalf("read run-info: %v", err)
			}
		}
	})
}

func TestMessageBusThroughput(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping message bus throughput test in short mode")
	}
	bus := newMessageBus(t)
	start := time.Now()
	deadline := start.Add(messageBusThroughputSecs * time.Second)
	var totalMsgs atomic.Int64
	var wg sync.WaitGroup
	errCh := make(chan error, messageBusWriterCount)

	for i := 0; i < messageBusWriterCount; i++ {
		wg.Add(1)
		go func(worker int) {
			defer wg.Done()
			for time.Now().Before(deadline) {
				msg := messagebus.Message{
					Type:      "FACT",
					ProjectID: "project",
					TaskID:    "task",
					RunID:     fmt.Sprintf("writer-%d", worker),
					Body:      "perf test message",
				}
				if _, err := bus.AppendMessage(&msg); err != nil {
					errCh <- err
					return
				}
				totalMsgs.Add(1)
			}
		}(i)
	}
	wg.Wait()
	close(errCh)
	for err := range errCh {
		if err != nil {
			t.Fatalf("append message: %v", err)
		}
	}

	throughput := float64(totalMsgs.Load()) / messageBusThroughputSecs
	minThroughput := floatEnv(envPerfMessageBusTarget, messageBusThroughputTarget)
	if minThroughput <= 0 {
		minThroughput = messageBusThroughputTarget
	}
	t.Logf("Throughput: %.2f messages/sec", throughput)
	if throughput < minThroughput {
		t.Errorf("throughput too low: %.2f", throughput)
	}
}

func TestRunCreationThroughput(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping run creation throughput test in short mode")
	}
	st := newStorage(t)
	start := time.Now()
	deadline := start.Add(runCreationSecs * time.Second)
	count := 0
	for time.Now().Before(deadline) {
		if _, err := st.CreateRun("project", fmt.Sprintf("task-%d", count), "codex"); err != nil {
			t.Fatalf("create run: %v", err)
		}
		count++
	}

	throughput := float64(count) / runCreationSecs
	minThroughput := floatEnv(envPerfRunCreationTarget, runCreationTarget)
	if minThroughput <= 0 {
		minThroughput = runCreationTarget
	}
	t.Logf("Run creation throughput: %.2f runs/sec", throughput)
	if throughput < minThroughput {
		t.Errorf("run creation throughput too low: %.2f", throughput)
	}
}

func TestConcurrentAgents(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping concurrent agent test in short mode")
	}
	root := t.TempDir()
	stubDir := t.TempDir()
	stubPath := buildCodexStub(t, stubDir)
	t.Setenv("PATH", prependPath(filepath.Dir(stubPath)))

	start := time.Now()
	var wg sync.WaitGroup
	errCh := make(chan error, concurrentAgentsCount)
	for i := 0; i < concurrentAgentsCount; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			if err := runTask(root, "project", fmt.Sprintf("task-%d", id)); err != nil {
				errCh <- err
			}
		}(i)
	}
	wg.Wait()
	close(errCh)
	for err := range errCh {
		if err != nil {
			t.Fatalf("run task: %v", err)
		}
	}

	duration := time.Since(start)
	t.Logf("%d concurrent agents completed in %v", concurrentAgentsCount, duration)
	if duration > concurrentAgentsTarget {
		t.Errorf("too slow: %v", duration)
	}
}

func TestSSELatency(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping SSE latency test in short mode")
	}
	root := t.TempDir()
	runID := "run-latency"
	runDir := writeRunInfoSSE(t, root, "project", "task", runID, storage.StatusRunning)
	stdoutPath := filepath.Join(runDir, "agent-stdout.txt")

	server := newSSEServer(t, root)
	defer server.Close()

	resp, err := http.Get(server.URL + "/api/v1/runs/" + runID + "/stream")
	if err != nil {
		t.Fatalf("stream request: %v", err)
	}
	defer resp.Body.Close()

	events := readSSEEvents(resp)
	writeTime := time.Now()
	if err := appendLine(stdoutPath, "test message"); err != nil {
		t.Fatalf("append line: %v", err)
	}
	_ = waitForEvent(t, events, "log", 2*time.Second)
	receiveTime := time.Now()

	latency := receiveTime.Sub(writeTime)
	maxLatency := durationEnvMs(envPerfSSELatencyMs, sseLatencyTarget)
	if maxLatency <= 0 {
		maxLatency = sseLatencyTarget
	}
	t.Logf("SSE latency: %v", latency)
	if latency > maxLatency {
		t.Errorf("latency too high: %v", latency)
	}
}

func newMessageBus(tb testing.TB) *messagebus.MessageBus {
	tb.Helper()
	path := filepath.Join(tb.TempDir(), "TASK-MESSAGE-BUS.md")
	bus, err := messagebus.NewMessageBus(path)
	if err != nil {
		tb.Fatalf("new message bus: %v", err)
	}
	return bus
}

func setupMessageBusWithMessages(tb testing.TB, count int) *messagebus.MessageBus {
	tb.Helper()
	bus := newMessageBus(tb)
	for i := 0; i < count; i++ {
		msg := messagebus.Message{
			Type:      "FACT",
			ProjectID: "project",
			TaskID:    "task",
			RunID:     fmt.Sprintf("seed-%d", i),
			Body:      fmt.Sprintf("seed message %d", i),
		}
		if _, err := bus.AppendMessage(&msg); err != nil {
			tb.Fatalf("append message: %v", err)
		}
	}
	return bus
}

func newStorage(tb testing.TB) *storage.FileStorage {
	tb.Helper()
	st, err := storage.NewStorage(tb.TempDir())
	if err != nil {
		tb.Fatalf("new storage: %v", err)
	}
	return st
}

type sseEvent struct {
	ID    string
	Event string
	Data  string
}

func readSSEEvents(resp *http.Response) <-chan sseEvent {
	out := make(chan sseEvent, 16)
	go func() {
		defer close(out)
		scanner := bufio.NewScanner(resp.Body)
		scanner.Buffer(make([]byte, 0, 64*1024), 1024*1024)
		var (
			current sseEvent
			data    []string
		)
		for scanner.Scan() {
			line := scanner.Text()
			if line == "" {
				if current.Event != "" || current.ID != "" || len(data) > 0 {
					current.Data = strings.Join(data, "\n")
					out <- current
				}
				current = sseEvent{}
				data = data[:0]
				continue
			}
			if strings.HasPrefix(line, "event:") {
				current.Event = strings.TrimSpace(strings.TrimPrefix(line, "event:"))
				continue
			}
			if strings.HasPrefix(line, "id:") {
				current.ID = strings.TrimSpace(strings.TrimPrefix(line, "id:"))
				continue
			}
			if strings.HasPrefix(line, "data:") {
				data = append(data, strings.TrimSpace(strings.TrimPrefix(line, "data:")))
			}
		}
	}()
	return out
}

func waitForEvent(t *testing.T, events <-chan sseEvent, eventType string, timeout time.Duration) sseEvent {
	t.Helper()
	timer := time.NewTimer(timeout)
	defer timer.Stop()
	for {
		select {
		case ev, ok := <-events:
			if !ok {
				t.Fatalf("event stream closed while waiting for %s", eventType)
			}
			if ev.Event == eventType {
				return ev
			}
		case <-timer.C:
			t.Fatalf("timeout waiting for event %s", eventType)
		}
	}
}

func newSSEServer(t *testing.T, root string) *httptest.Server {
	t.Helper()
	server, err := api.NewServer(api.Options{
		RootDir:          root,
		DisableTaskStart: true,
		APIConfig: config.APIConfig{
			Host: "127.0.0.1",
			Port: 0,
			SSE: config.SSEConfig{
				PollIntervalMs:      10,
				DiscoveryIntervalMs: 50,
				HeartbeatIntervalS:  1,
				MaxClientsPerRun:    10,
			},
		},
		Version: "test",
	})
	if err != nil {
		t.Fatalf("new server: %v", err)
	}
	return httptest.NewServer(server.Handler())
}

func writeRunInfoSSE(t *testing.T, root, projectID, taskID, runID, status string) string {
	t.Helper()
	runDir := filepath.Join(root, projectID, taskID, "runs", runID)
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
		StartTime: time.Now().UTC(),
		Status:    status,
		ExitCode:  -1,
	}
	if status != storage.StatusRunning {
		info.EndTime = time.Now().UTC()
		info.ExitCode = 0
	}
	if err := storage.WriteRunInfo(filepath.Join(runDir, "run-info.yaml"), info); err != nil {
		t.Fatalf("write run info: %v", err)
	}
	return runDir
}

func appendLine(path, line string) error {
	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return err
	}
	defer file.Close()
	if _, err := file.WriteString(line + "\n"); err != nil {
		return err
	}
	return file.Sync()
}

func buildCodexStub(t *testing.T, dir string) string {
	t.Helper()

	stubPath := filepath.Join(dir, "codex")
	if runtime.GOOS == "windows" {
		stubPath += ".exe"
	}

	src := fmt.Sprintf(`package main

import (
	"io"
	"os"
	"strconv"
	"time"
)

func main() {
	if sleep := os.Getenv(%q); sleep != "" {
		if ms, err := strconv.Atoi(sleep); err == nil {
			time.Sleep(time.Duration(ms) * time.Millisecond)
		}
	}
	if path := os.Getenv(%q); path != "" {
		_ = os.WriteFile(path, []byte(""), 0o644)
	}
	_, _ = io.Copy(io.Discard, os.Stdin)
}
`, envPerfStubSleepMs, envPerfStubDoneFile)

	srcPath := filepath.Join(dir, "codex_stub.go")
	if err := os.WriteFile(srcPath, []byte(src), 0o644); err != nil {
		t.Fatalf("write stub: %v", err)
	}

	cmd := exec.Command("go", "build", "-o", stubPath, srcPath)
	cmd.Env = os.Environ()
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("build stub: %v\n%s", err, out)
	}

	return stubPath
}

func runTask(root, projectID, taskID string) error {
	taskDir := filepath.Join(root, projectID, taskID)
	if err := os.MkdirAll(taskDir, 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(taskDir, "TASK.md"), []byte("performance test task"), 0o644); err != nil {
		return err
	}
	donePath := filepath.Join(taskDir, "DONE")
	opts := runner.TaskOptions{
		RootDir:      root,
		Agent:        "codex",
		WorkingDir:   taskDir,
		MaxRestarts:  1,
		WaitTimeout:  30 * time.Second,
		PollInterval: 10 * time.Millisecond,
		RestartDelay: 10 * time.Millisecond,
		Environment: map[string]string{
			envPerfStubDoneFile: donePath,
			envPerfStubSleepMs:  "10",
		},
	}
	return runner.RunTask(projectID, taskID, opts)
}

func prependPath(dir string) string {
	pathEnv := os.Getenv("PATH")
	if pathEnv == "" {
		return dir
	}
	return dir + string(os.PathListSeparator) + pathEnv
}

func floatEnv(key string, fallback float64) float64 {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	parsed, err := strconv.ParseFloat(value, 64)
	if err != nil {
		return fallback
	}
	return parsed
}

func durationEnvMs(key string, fallback time.Duration) time.Duration {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	parsed, err := strconv.Atoi(value)
	if err != nil {
		return fallback
	}
	return time.Duration(parsed) * time.Millisecond
}
