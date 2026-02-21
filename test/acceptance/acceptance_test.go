package acceptance_test

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/jonnyzzz/conductor-loop/internal/api"
	"github.com/jonnyzzz/conductor-loop/internal/config"
	"github.com/jonnyzzz/conductor-loop/internal/messagebus"
	"github.com/jonnyzzz/conductor-loop/internal/runner"
	"github.com/jonnyzzz/conductor-loop/internal/storage"
)

const (
	envAcceptance   = "ACCEPTANCE"
	envAcceptanceUI = "ACCEPTANCE_UI"

	projectID = "acceptance"
	agentType = "codex"

	envStubStdout        = "ORCH_STUB_STDOUT"
	envStubStdoutLines   = "ORCH_STUB_STDOUT_LINES"
	envStubStdoutDelayMs = "ORCH_STUB_STDOUT_DELAY_MS"
	envStubSleepMs       = "ORCH_STUB_SLEEP_MS"
	envStubDoneFile      = "ORCH_STUB_DONE_FILE"
	envStubStartDelayMs  = "ORCH_STUB_START_DELAY_MS"
	envStubStderr        = "ORCH_STUB_STDERR"
)

type conductorHarness struct {
	rootDir string
	baseURL string
	server  *httptest.Server
}

type taskHandle struct {
	ProjectID string
	TaskID    string
	RunID     string
}

type healthSnapshot struct {
	goroutines int
	fdCount    int
	busBytes   int64
	runsBytes  int64
	heapBytes  uint64
}

type scenarioReport struct {
	name     string
	passed   bool
	duration time.Duration
	before   healthSnapshot
	after    healthSnapshot
}

type acceptanceReport struct {
	root      string
	scenarios []scenarioReport
	mu        sync.Mutex
}

func TestAcceptanceScenarios(t *testing.T) {
	if os.Getenv(envAcceptance) == "" {
		t.Skip("set ACCEPTANCE=1 to run acceptance tests")
	}

	h := startConductor(t)
	report := &acceptanceReport{root: h.rootDir}
	t.Cleanup(func() {
		report.write(t)
	})

	scenarios := []struct {
		name string
		fn   func(t *testing.T, h *conductorHarness)
	}{
		{name: "Scenario1_SingleAgentTask", fn: runScenario1},
		{name: "Scenario2_ParentChildRuns", fn: runScenario2},
		{name: "Scenario3_RalphLoopWait", fn: runScenario3},
		{name: "Scenario4_MessageBusRace", fn: runScenario4},
	}
	if os.Getenv(envAcceptanceUI) != "" {
		scenarios = append(scenarios, struct {
			name string
			fn   func(t *testing.T, h *conductorHarness)
		}{name: "Scenario5_UILiveMonitoring", fn: runScenario5})
	} else {
		t.Log("Scenario5_UILiveMonitoring skipped; set ACCEPTANCE_UI=1 to enable")
	}

	for _, scenario := range scenarios {
		before := captureHealth(t, h.rootDir)
		start := time.Now()
		passed := t.Run(scenario.name, func(t *testing.T) {
			scenario.fn(t, h)
		})
		duration := time.Since(start)
		after := captureHealth(t, h.rootDir)
		report.add(scenario.name, passed, duration, before, after)
		checkHealthDelta(t, h.rootDir, scenario.name, before, after)
	}

	if report.allPassed() {
		appendMessageBus(t, []string{
			"FACT: Scenario 1 (single agent) passed",
			"FACT: Scenario 2 (parent-child) passed",
			"FACT: Scenario 3 (Ralph wait) passed",
			"FACT: Scenario 4 (message bus race) passed",
		})
		if os.Getenv(envAcceptanceUI) != "" {
			appendMessageBus(t, []string{
				"FACT: Scenario 5 (UI monitoring) passed",
			})
		}
		appendMessageBus(t, []string{"FACT: All acceptance tests passed"})
	}
}

func runScenario1(t *testing.T, h *conductorHarness) {
	taskID := "single-task"
	donePath := filepath.Join(taskDir(h.rootDir, taskID), "DONE")
	cfg := map[string]string{
		envStubStdout:   "hello",
		envStubDoneFile: donePath,
	}
	task := createTask(t, h, taskID, "echo hello", cfg)

	waitForCompletion(t, h, task.RunID, 2*time.Minute)

	info := getRunInfo(t, h, task.RunID)
	if info.Status != storage.StatusCompleted {
		t.Fatalf("run status: want %q got %q", storage.StatusCompleted, info.Status)
	}
	if info.ExitCode != 0 {
		t.Fatalf("exit code: want 0 got %d", info.ExitCode)
	}

	logs := readLogs(t, h, taskID, task.RunID)
	if !strings.Contains(logs, "hello") {
		t.Fatalf("logs missing 'hello': %q", logs)
	}

	messages := readTaskMessages(t, h, taskID)
	if !containsMessage(messages, "task completed") {
		t.Fatalf("expected task completion message, got %d messages", len(messages))
	}

	appendTaskMessage(t, h, taskID, "FACT", "Task single-task completed")
	messages = readTaskMessages(t, h, taskID)
	if !containsMessage(messages, "Task single-task completed") {
		t.Fatalf("expected FACT completion message, got %d messages", len(messages))
	}
}

func runScenario2(t *testing.T, h *conductorHarness) {
	if runtime.GOOS == "windows" {
		t.Skip("process group checks not reliable on windows")
	}

	taskID := "parent"
	taskDir := taskDir(h.rootDir, taskID)
	cfg := map[string]string{
		envStubStdout:   "parent done",
		envStubSleepMs:  "200",
		envStubStderr:   "",
		envStubDoneFile: "",
	}
	task := createTask(t, h, taskID, "parent", cfg)

	childErr := make(chan error, 3)
	for i := 1; i <= 3; i++ {
		childID := fmt.Sprintf("child-%d", i)
		childEnv := map[string]string{
			envStubStdout:  fmt.Sprintf("child-%d", i),
			envStubSleepMs: "300",
		}
		go func() {
			childErr <- runner.RunJob(projectID, taskID, runner.JobOptions{
				RootDir:     h.rootDir,
				Agent:       agentType,
				Prompt:      childID,
				WorkingDir:  taskDir,
				ParentRunID: task.RunID,
				Environment: childEnv,
			})
		}()
	}

	waitForChildRuns(t, h, taskID, task.RunID, 3, 10*time.Second)

	if err := touchDoneFile(taskDir); err != nil {
		t.Fatalf("touch DONE: %v", err)
	}

	waitForTaskCompletion(t, h, taskID, 5*time.Minute)

	childRuns := findChildRuns(t, h, taskID, task.RunID)
	if len(childRuns) != 3 {
		t.Fatalf("expected 3 child runs, got %d", len(childRuns))
	}
	for _, child := range childRuns {
		if child.Status != storage.StatusCompleted {
			t.Fatalf("child %s status: want %q got %q", child.RunID, storage.StatusCompleted, child.Status)
		}
		if child.ExitCode != 0 {
			t.Fatalf("child %s exit code: want 0 got %d", child.RunID, child.ExitCode)
		}
	}

	parentRunID := latestRunIDForTask(t, h, taskID)
	parentInfo := getRunInfo(t, h, parentRunID)
	if parentInfo.Status != storage.StatusCompleted {
		t.Fatalf("parent status: want %q got %q", storage.StatusCompleted, parentInfo.Status)
	}

	tree := getRunTree(t, h, taskID, task.RunID)
	if tree == nil || len(tree.Children) != 3 {
		count := 0
		if tree != nil {
			count = len(tree.Children)
		}
		t.Fatalf("expected run tree with 3 children, got %d", count)
	}

	for i := 0; i < 3; i++ {
		if err := <-childErr; err != nil {
			t.Fatalf("child run failed: %v", err)
		}
	}
}

func runScenario3(t *testing.T, h *conductorHarness) {
	if runtime.GOOS == "windows" {
		t.Skip("process group checks not reliable on windows")
	}

	taskID := "ralph-wait"
	taskDir := taskDir(h.rootDir, taskID)
	cfg := map[string]string{
		envStubStdout:  "parent DONE created",
		envStubSleepMs: "200",
	}
	task := createTask(t, h, taskID, "ralph wait", cfg)

	childErr := make(chan error, 1)
	go func() {
		childErr <- runner.RunJob(projectID, taskID, runner.JobOptions{
			RootDir:     h.rootDir,
			Agent:       agentType,
			Prompt:      "long child",
			WorkingDir:  taskDir,
			ParentRunID: task.RunID,
			Environment: map[string]string{
				envStubStdout:  "child done",
				envStubSleepMs: "30000",
			},
		})
	}()

	waitForChildRuns(t, h, taskID, task.RunID, 1, 10*time.Second)

	if err := touchDoneFile(taskDir); err != nil {
		t.Fatalf("touch DONE: %v", err)
	}
	waitForDONE(t, taskDir, 1*time.Minute)

	parentInfo := getRunInfo(t, h, task.RunID)
	if parentInfo.Status != storage.StatusRunning && parentInfo.Status != "waiting" {
		t.Fatalf("parent status: expected running/waiting, got %q", parentInfo.Status)
	}

	if err := <-childErr; err != nil {
		t.Fatalf("child run failed: %v", err)
	}
	waitForTaskCompletion(t, h, taskID, 2*time.Minute)

	parentRunID := latestRunIDForTask(t, h, taskID)
	parentInfo = getRunInfo(t, h, parentRunID)
	if parentInfo.Status != storage.StatusCompleted {
		t.Fatalf("parent status: want %q got %q", storage.StatusCompleted, parentInfo.Status)
	}

	messages := readTaskMessages(t, h, taskID)
	if !containsMessage(messages, "waiting for") {
		t.Fatalf("expected wait message in bus, got %d messages", len(messages))
	}
}

func runScenario4(t *testing.T, h *conductorHarness) {
	taskID := "message-bus-race"
	busPath := filepath.Join(taskDir(h.rootDir, taskID), "TASK-MESSAGE-BUS.md")
	if err := os.MkdirAll(filepath.Dir(busPath), 0o755); err != nil {
		t.Fatalf("mkdir task dir: %v", err)
	}
	bus, err := messagebus.NewMessageBus(busPath)
	if err != nil {
		t.Fatalf("new message bus: %v", err)
	}

	var wg sync.WaitGroup
	errCh := make(chan error, 10)
	for i := 0; i < 10; i++ {
		agentID := i
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 1; j <= 100; j++ {
				msg := &messagebus.Message{
					Type:      "FACT",
					ProjectID: projectID,
					TaskID:    taskID,
					RunID:     fmt.Sprintf("agent-%d", agentID),
					Body:      fmt.Sprintf("Agent %d message %d", agentID, j),
				}
				if _, err := bus.AppendMessage(msg); err != nil {
					errCh <- err
					return
				}
			}
		}()
	}
	wg.Wait()
	close(errCh)
	for err := range errCh {
		if err != nil {
			t.Fatalf("append message: %v", err)
		}
	}

	messages, err := bus.ReadMessages("")
	if err != nil {
		t.Fatalf("read message bus: %v", err)
	}

	count := countAgentMessages(messages)
	if count < 1000 {
		t.Fatalf("expected >= 1000 agent messages, got %d", count)
	}

	for i := 0; i < 10; i++ {
		agentMsgs := filterAgentMessages(messages, i)
		verifySequential(t, agentMsgs, i)
	}

	for _, msg := range messages {
		if !isWellFormed(msg) {
			t.Fatalf("malformed message: %#v", msg)
		}
	}
}

func runScenario5(t *testing.T, h *conductorHarness) {
	if os.Getenv(envAcceptanceUI) == "" {
		t.Skip("set ACCEPTANCE_UI=1 to run UI acceptance scenario")
	}
	if _, err := exec.LookPath("npm"); err != nil {
		t.Skip("npm not available")
	}
	if _, err := exec.LookPath("npx"); err != nil {
		t.Skip("npx not available")
	}

	adapter := startUIAdapter(t, h.rootDir, h.baseURL)
	defer adapter.Close()

	frontendURL, stopFrontend := startFrontend(t, adapter.URL)
	defer stopFrontend()

	taskID := "ui-monitor"
	taskDir := taskDir(h.rootDir, taskID)
	progressLines := []string{
		"Progress: 1/10",
		"Progress: 2/10",
		"Progress: 3/10",
		"Progress: 4/10",
		"Progress: 5/10",
		"Progress: 6/10",
		"Progress: 7/10",
		"Progress: 8/10",
		"Progress: 9/10",
		"Progress: 10/10",
		"Complete",
	}
	cfg := map[string]string{
		envStubStdoutLines:   strings.Join(progressLines, "|"),
		envStubStdoutDelayMs: "1000",
		envStubStartDelayMs:  "5000",
		envStubDoneFile:      filepath.Join(taskDir, "DONE"),
	}
	task := createTask(t, h, taskID, "ui-monitor", cfg)

	runPlaywright(t, frontendURL, task.TaskID)

	waitForCompletion(t, h, task.RunID, 2*time.Minute)
}

func startConductor(t *testing.T) *conductorHarness {
	root := t.TempDir()

	stubDir := t.TempDir()
	stubPath := buildCodexStub(t, stubDir)
	t.Setenv("PATH", prependPath(filepath.Dir(stubPath)))

	server, err := api.NewServer(api.Options{
		RootDir:          root,
		DisableTaskStart: false,
		APIConfig: config.APIConfig{
			Host: "127.0.0.1",
			Port: 0,
			SSE: config.SSEConfig{
				PollIntervalMs:      25,
				DiscoveryIntervalMs: 25,
				HeartbeatIntervalS:  1,
				MaxClientsPerRun:    20,
			},
		},
		Version: "acceptance",
	})
	if err != nil {
		t.Fatalf("new server: %v", err)
	}

	ts := httptest.NewServer(server.Handler())
	h := &conductorHarness{rootDir: root, baseURL: ts.URL, server: ts}
	cleanupOnce := sync.Once{}
	t.Cleanup(func() {
		cleanupOnce.Do(func() {
			stopActiveRuns(t, h.rootDir)
			resetMessageBuses(t, h.rootDir)
			cleanupRuns(t, h.rootDir)
			if h.server != nil {
				h.server.Close()
			}
		})
	})
	return h
}

func createTask(t *testing.T, h *conductorHarness, taskID, prompt string, env map[string]string) taskHandle {
	payload := api.TaskCreateRequest{
		ProjectID: projectID,
		TaskID:    taskID,
		AgentType: agentType,
		Prompt:    prompt,
		Config:    env,
	}
	data, err := json.Marshal(payload)
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}
	resp, err := http.Post(h.baseURL+"/api/v1/tasks", "application/json", bytes.NewReader(data))
	if err != nil {
		t.Fatalf("post task: %v", err)
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusCreated {
		t.Fatalf("create task: unexpected status %d", resp.StatusCode)
	}

	runID := waitForRunID(t, h, taskID, 30*time.Second)
	return taskHandle{ProjectID: projectID, TaskID: taskID, RunID: runID}
}

func waitForRunID(t *testing.T, h *conductorHarness, taskID string, timeout time.Duration) string {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		resp := getTask(t, h, taskID)
		if len(resp.Runs) > 0 {
			latest := resp.Runs[0]
			for _, run := range resp.Runs[1:] {
				if run.StartTime.After(latest.StartTime) {
					latest = run
				}
			}
			if latest.RunID != "" {
				return latest.RunID
			}
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatalf("timeout waiting for run id for task %s", taskID)
	return ""
}

func waitForCompletion(t *testing.T, h *conductorHarness, runID string, timeout time.Duration) {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		info := getRunInfo(t, h, runID)
		switch info.Status {
		case storage.StatusCompleted:
			return
		case storage.StatusFailed:
			t.Fatalf("run %s failed", runID)
		}
		time.Sleep(200 * time.Millisecond)
	}
	t.Fatalf("timeout waiting for run %s completion", runID)
}

func waitForTaskCompletion(t *testing.T, h *conductorHarness, taskID string, timeout time.Duration) {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		task := getTask(t, h, taskID)
		if task.Status == storage.StatusCompleted {
			return
		}
		time.Sleep(200 * time.Millisecond)
	}
	t.Fatalf("timeout waiting for task %s completion", taskID)
}

func latestRunIDForTask(t *testing.T, h *conductorHarness, taskID string) string {
	resp := getTask(t, h, taskID)
	if len(resp.Runs) == 0 {
		t.Fatalf("no runs for task %s", taskID)
	}
	latest := resp.Runs[0]
	for _, run := range resp.Runs[1:] {
		if run.StartTime.After(latest.StartTime) {
			latest = run
		}
	}
	if latest.RunID == "" {
		t.Fatalf("latest run id empty for task %s", taskID)
	}
	return latest.RunID
}

func getTask(t *testing.T, h *conductorHarness, taskID string) api.TaskResponse {
	url := fmt.Sprintf("%s/api/v1/tasks/%s?project_id=%s", h.baseURL, taskID, projectID)
	resp, err := http.Get(url)
	if err != nil {
		t.Fatalf("get task: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("get task: unexpected status %d", resp.StatusCode)
	}
	var payload api.TaskResponse
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		t.Fatalf("decode task: %v", err)
	}
	return payload
}

func getRunInfo(t *testing.T, h *conductorHarness, runID string) api.RunResponse {
	url := fmt.Sprintf("%s/api/v1/runs/%s", h.baseURL, runID)
	resp, err := http.Get(url)
	if err != nil {
		t.Fatalf("get run: %v", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("get run: unexpected status %d", resp.StatusCode)
	}
	var payload api.RunResponse
	if err := json.NewDecoder(resp.Body).Decode(&payload); err != nil {
		t.Fatalf("decode run: %v", err)
	}
	return payload
}

func readLogs(t *testing.T, h *conductorHarness, taskID, runID string) string {
	path := filepath.Join(taskDir(h.rootDir, taskID), "runs", runID, "agent-stdout.txt")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read logs: %v", err)
	}
	return string(data)
}

func readTaskMessages(t *testing.T, h *conductorHarness, taskID string) []*messagebus.Message {
	busPath := filepath.Join(taskDir(h.rootDir, taskID), "TASK-MESSAGE-BUS.md")
	bus, err := messagebus.NewMessageBus(busPath)
	if err != nil {
		t.Fatalf("new message bus: %v", err)
	}
	messages, err := bus.ReadMessages("")
	if err != nil {
		t.Fatalf("read messages: %v", err)
	}
	return messages
}

func appendTaskMessage(t *testing.T, h *conductorHarness, taskID, msgType, body string) {
	busPath := filepath.Join(taskDir(h.rootDir, taskID), "TASK-MESSAGE-BUS.md")
	bus, err := messagebus.NewMessageBus(busPath)
	if err != nil {
		t.Fatalf("new message bus: %v", err)
	}
	_, err = bus.AppendMessage(&messagebus.Message{
		Type:      msgType,
		ProjectID: projectID,
		TaskID:    taskID,
		Body:      body,
	})
	if err != nil {
		t.Fatalf("append message: %v", err)
	}
}

func containsMessage(messages []*messagebus.Message, substr string) bool {
	for _, msg := range messages {
		if msg != nil && strings.Contains(msg.Body, substr) {
			return true
		}
	}
	return false
}

func findChildRuns(t *testing.T, h *conductorHarness, taskID, parentRunID string) []*storage.RunInfo {
	infos := listRunInfos(t, h, taskID)
	children := make([]*storage.RunInfo, 0)
	for _, info := range infos {
		if info == nil {
			continue
		}
		if strings.TrimSpace(info.ParentRunID) == strings.TrimSpace(parentRunID) {
			children = append(children, info)
		}
	}
	return children
}

func waitForChildRuns(t *testing.T, h *conductorHarness, taskID, parentRunID string, expected int, timeout time.Duration) {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		children := findChildRuns(t, h, taskID, parentRunID)
		if len(children) >= expected {
			return
		}
		time.Sleep(50 * time.Millisecond)
	}
	t.Fatalf("timeout waiting for %d child runs (got fewer)", expected)
}

type runNode struct {
	Info     *storage.RunInfo
	Children []*runNode
}

func getRunTree(t *testing.T, h *conductorHarness, taskID, rootRunID string) *runNode {
	infos := listRunInfos(t, h, taskID)
	nodes := make(map[string]*runNode, len(infos))
	for _, info := range infos {
		if info == nil {
			continue
		}
		nodes[info.RunID] = &runNode{Info: info}
	}
	for _, node := range nodes {
		parentID := strings.TrimSpace(node.Info.ParentRunID)
		if parentID == "" {
			continue
		}
		if parent, ok := nodes[parentID]; ok {
			parent.Children = append(parent.Children, node)
		}
	}
	return nodes[rootRunID]
}

func listRunInfos(t *testing.T, h *conductorHarness, taskID string) []*storage.RunInfo {
	runsDir := filepath.Join(taskDir(h.rootDir, taskID), "runs")
	entries, err := os.ReadDir(runsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		t.Fatalf("read runs dir: %v", err)
	}
	infos := make([]*storage.RunInfo, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		path := filepath.Join(runsDir, entry.Name(), "run-info.yaml")
		info, err := storage.ReadRunInfo(path)
		if err != nil {
			continue
		}
		infos = append(infos, info)
	}
	sort.Slice(infos, func(i, j int) bool {
		return infos[i].StartTime.Before(infos[j].StartTime)
	})
	return infos
}

func waitForDONE(t *testing.T, taskDir string, timeout time.Duration) {
	deadline := time.Now().Add(timeout)
	path := filepath.Join(taskDir, "DONE")
	for time.Now().Before(deadline) {
		if _, err := os.Stat(path); err == nil {
			return
		}
		time.Sleep(100 * time.Millisecond)
	}
	t.Fatalf("timeout waiting for DONE at %s", path)
}

func touchDoneFile(taskDir string) error {
	path := filepath.Join(taskDir, "DONE")
	if err := os.MkdirAll(taskDir, 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, []byte(""), 0o644)
}

func countAgentMessages(messages []*messagebus.Message) int {
	count := 0
	for _, msg := range messages {
		if msg == nil {
			continue
		}
		if _, _, ok := parseAgentMessage(msg.Body); ok {
			count++
		}
	}
	return count
}

func filterAgentMessages(messages []*messagebus.Message, agentID int) []int {
	seq := make([]int, 0)
	for _, msg := range messages {
		if msg == nil {
			continue
		}
		agent, num, ok := parseAgentMessage(msg.Body)
		if !ok {
			continue
		}
		if agent == agentID {
			seq = append(seq, num)
		}
	}
	return seq
}

func parseAgentMessage(body string) (int, int, bool) {
	var agent int
	var seq int
	_, err := fmt.Sscanf(body, "Agent %d message %d", &agent, &seq)
	if err != nil {
		return 0, 0, false
	}
	return agent, seq, true
}

func verifySequential(t *testing.T, messages []int, agentID int) {
	if len(messages) == 0 {
		t.Fatalf("no messages for agent %d", agentID)
	}
	for i, seq := range messages {
		expected := i + 1
		if seq != expected {
			t.Fatalf("agent %d message ordering: want %d got %d", agentID, expected, seq)
		}
	}
}

func isWellFormed(msg *messagebus.Message) bool {
	return msg != nil && msg.MsgID != "" && msg.Type != "" && msg.ProjectID != "" && msg.Body != ""
}

func captureHealth(t *testing.T, root string) healthSnapshot {
	runtime.GC()
	var mem runtime.MemStats
	runtime.ReadMemStats(&mem)
	return healthSnapshot{
		goroutines: runtime.NumGoroutine(),
		fdCount:    countOpenFiles(),
		busBytes:   messageBusBytes(root),
		runsBytes:  dirSize(filepath.Join(root, projectID)),
		heapBytes:  mem.HeapAlloc,
	}
}

func checkHealthDelta(t *testing.T, root, label string, before, after healthSnapshot) {
	active, err := findActiveRuns(root)
	if err != nil {
		t.Fatalf("health check (%s): %v", label, err)
	}
	if len(active) > 0 {
		t.Fatalf("health check (%s): %d active runs still running", label, len(active))
	}
	if after.goroutines > before.goroutines+50 {
		t.Fatalf("health check (%s): goroutines increased too much: %d -> %d", label, before.goroutines, after.goroutines)
	}
	if before.fdCount > 0 && after.fdCount > before.fdCount+50 {
		t.Fatalf("health check (%s): fd count increased too much: %d -> %d", label, before.fdCount, after.fdCount)
	}
	if after.busBytes > 10<<20 {
		t.Fatalf("health check (%s): message bus grew too large: %d bytes", label, after.busBytes)
	}
	if after.runsBytes > 200<<20 {
		t.Fatalf("health check (%s): runs directory too large: %d bytes", label, after.runsBytes)
	}
	if after.heapBytes > before.heapBytes+100<<20 {
		t.Fatalf("health check (%s): heap usage increased too much: %d -> %d", label, before.heapBytes, after.heapBytes)
	}
}

func findActiveRuns(root string) ([]*storage.RunInfo, error) {
	var active []*storage.RunInfo
	walkErr := filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		if entry.Name() != "run-info.yaml" {
			return nil
		}
		info, err := storage.ReadRunInfo(path)
		if err != nil {
			return nil
		}
		if info.EndTime.IsZero() {
			active = append(active, info)
		}
		return nil
	})
	return active, walkErr
}

func stopActiveRuns(t *testing.T, root string) {
	active, err := findActiveRuns(root)
	if err != nil {
		return
	}
	for _, info := range active {
		if info == nil || info.PGID == 0 {
			continue
		}
		_ = runner.TerminateProcessGroup(info.PGID)
	}
}

func resetMessageBuses(t *testing.T, root string) {
	_ = filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if entry.IsDir() {
			return nil
		}
		name := entry.Name()
		if name != "TASK-MESSAGE-BUS.md" && name != "PROJECT-MESSAGE-BUS.md" {
			return nil
		}
		_ = os.WriteFile(path, []byte(""), 0o644)
		return nil
	})
}

func cleanupRuns(t *testing.T, root string) {
	projectPath := filepath.Join(root, projectID)
	_ = filepath.WalkDir(projectPath, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if entry.IsDir() && entry.Name() == "runs" {
			_ = os.RemoveAll(path)
			return filepath.SkipDir
		}
		return nil
	})
}

func countOpenFiles() int {
	switch runtime.GOOS {
	case "linux":
		return countDirEntries("/proc/self/fd")
	case "darwin":
		return countDirEntries("/dev/fd")
	default:
		return -1
	}
}

func countDirEntries(path string) int {
	entries, err := os.ReadDir(path)
	if err != nil {
		return -1
	}
	return len(entries)
}

func messageBusBytes(root string) int64 {
	var total int64
	_ = filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if entry.IsDir() {
			return nil
		}
		name := entry.Name()
		if name != "TASK-MESSAGE-BUS.md" && name != "PROJECT-MESSAGE-BUS.md" {
			return nil
		}
		if info, statErr := entry.Info(); statErr == nil {
			total += info.Size()
		}
		return nil
	})
	return total
}

func dirSize(root string) int64 {
	var total int64
	_ = filepath.WalkDir(root, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return nil
		}
		if entry.IsDir() {
			return nil
		}
		info, err := entry.Info()
		if err != nil {
			return nil
		}
		total += info.Size()
		return nil
	})
	return total
}

func (r *acceptanceReport) add(name string, passed bool, duration time.Duration, before, after healthSnapshot) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.scenarios = append(r.scenarios, scenarioReport{
		name:     name,
		passed:   passed,
		duration: duration,
		before:   before,
		after:    after,
	})
}

func (r *acceptanceReport) allPassed() bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	for _, scenario := range r.scenarios {
		if !scenario.passed {
			return false
		}
	}
	return true
}

func (r *acceptanceReport) write(t *testing.T) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if len(r.scenarios) == 0 {
		return
	}

	var buf strings.Builder
	buf.WriteString("Acceptance Test Report\n\n")
	for _, scenario := range r.scenarios {
		status := "FAILED"
		if scenario.passed {
			status = "PASSED"
		}
		buf.WriteString(fmt.Sprintf("- %s: %s (%s)\n", scenario.name, status, scenario.duration))
		buf.WriteString(fmt.Sprintf("  goroutines: %d -> %d\n", scenario.before.goroutines, scenario.after.goroutines))
		buf.WriteString(fmt.Sprintf("  fds: %d -> %d\n", scenario.before.fdCount, scenario.after.fdCount))
		buf.WriteString(fmt.Sprintf("  bus bytes: %d -> %d\n", scenario.before.busBytes, scenario.after.busBytes))
		buf.WriteString(fmt.Sprintf("  runs bytes: %d -> %d\n", scenario.before.runsBytes, scenario.after.runsBytes))
		buf.WriteString(fmt.Sprintf("  heap bytes: %d -> %d\n", scenario.before.heapBytes, scenario.after.heapBytes))
	}

	reportPath := filepath.Join(r.root, "acceptance-report.md")
	if err := os.WriteFile(reportPath, []byte(buf.String()), 0o644); err != nil {
		t.Logf("write acceptance report: %v", err)
		return
	}
	t.Logf("acceptance report written to %s", reportPath)
}

func appendMessageBus(t *testing.T, lines []string) {
	root := repoRoot(t)
	path := filepath.Join(root, "MESSAGE-BUS.md")
	file, err := os.OpenFile(path, os.O_APPEND|os.O_WRONLY, 0o644)
	if err != nil {
		t.Fatalf("open MESSAGE-BUS.md: %v", err)
	}
	defer file.Close()
	timestamp := time.Now().Format("2006-01-02 15:04:05")
	for _, line := range lines {
		if strings.TrimSpace(line) == "" {
			continue
		}
		if _, err := fmt.Fprintf(file, "[%s] %s\n", timestamp, line); err != nil {
			t.Fatalf("append MESSAGE-BUS.md: %v", err)
		}
	}
}

func repoRoot(t *testing.T) string {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("getwd: %v", err)
	}
	dir := wd
	for i := 0; i < 6; i++ {
		if _, err := os.Stat(filepath.Join(dir, "MESSAGE-BUS.md")); err == nil {
			return dir
		}
		next := filepath.Dir(dir)
		if next == dir {
			break
		}
		dir = next
	}
	t.Fatalf("repo root not found from %s", wd)
	return ""
}

func taskDir(root, taskID string) string {
	return filepath.Join(root, projectID, taskID)
}

func prependPath(dir string) string {
	pathEnv := os.Getenv("PATH")
	if pathEnv == "" {
		return dir
	}
	return dir + string(os.PathListSeparator) + pathEnv
}

func buildCodexStub(t *testing.T, dir string) string {
	stubPath := filepath.Join(dir, "codex")
	if runtime.GOOS == "windows" {
		stubPath += ".exe"
	}

	src := `package main

import (
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"
	"time"
)

func main() {
	if delay := os.Getenv("` + envStubStartDelayMs + `"); delay != "" {
		if ms, err := strconv.Atoi(delay); err == nil && ms > 0 {
			time.Sleep(time.Duration(ms) * time.Millisecond)
		}
	}
	if path := os.Getenv("` + envStubDoneFile + `"); path != "" {
		_ = os.WriteFile(path, []byte(""), 0o644)
	}
	if lines := os.Getenv("` + envStubStdoutLines + `"); lines != "" {
		delayMs := 0
		if raw := os.Getenv("` + envStubStdoutDelayMs + `"); raw != "" {
			if ms, err := strconv.Atoi(raw); err == nil {
				delayMs = ms
			}
		}
		for _, line := range strings.Split(lines, "|") {
			if strings.TrimSpace(line) == "" {
				continue
			}
			_, _ = fmt.Fprintln(os.Stdout, line)
			if delayMs > 0 {
				time.Sleep(time.Duration(delayMs) * time.Millisecond)
			}
		}
	} else if out := os.Getenv("` + envStubStdout + `"); out != "" {
		_, _ = fmt.Fprint(os.Stdout, out)
	} else {
		_, _ = fmt.Fprint(os.Stdout, "stub output")
	}
	if errText := os.Getenv("` + envStubStderr + `"); errText != "" {
		_, _ = fmt.Fprint(os.Stderr, errText)
	}
	if sleep := os.Getenv("` + envStubSleepMs + `"); sleep != "" {
		if ms, err := strconv.Atoi(sleep); err == nil && ms > 0 {
			time.Sleep(time.Duration(ms) * time.Millisecond)
		}
	}
	_, _ = io.Copy(io.Discard, os.Stdin)
}
`

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

// UI adapter and Playwright helpers

type uiAdapter struct {
	rootDir    string
	conductor  string
	httpClient *http.Client
	pollDelay  time.Duration
	heartbeat  time.Duration
}

type uiProject struct {
	ID           string `json:"id"`
	LastActivity string `json:"last_activity"`
	TaskCount    int    `json:"task_count"`
}

type uiProjectsResponse struct {
	Projects []uiProject `json:"projects"`
}

type uiTaskSummary struct {
	ID           string `json:"id"`
	Name         string `json:"name,omitempty"`
	Status       string `json:"status"`
	LastActivity string `json:"last_activity"`
	RunCount     int    `json:"run_count,omitempty"`
}

type uiTasksResponse struct {
	Tasks []uiTaskSummary `json:"tasks"`
}

type uiRunSummary struct {
	ID            string `json:"id"`
	Agent         string `json:"agent"`
	Status        string `json:"status"`
	ExitCode      int    `json:"exit_code"`
	StartTime     string `json:"start_time"`
	EndTime       string `json:"end_time,omitempty"`
	ParentRunID   string `json:"parent_run_id,omitempty"`
	PreviousRunID string `json:"previous_run_id,omitempty"`
}

type uiTaskDetail struct {
	ID           string         `json:"id"`
	Name         string         `json:"name,omitempty"`
	ProjectID    string         `json:"project_id"`
	Status       string         `json:"status"`
	LastActivity string         `json:"last_activity"`
	CreatedAt    string         `json:"created_at"`
	Done         bool           `json:"done"`
	State        string         `json:"state"`
	Runs         []uiRunSummary `json:"runs"`
}

type uiRunInfo struct {
	Version       int    `json:"version"`
	RunID         string `json:"run_id"`
	ProjectID     string `json:"project_id"`
	TaskID        string `json:"task_id"`
	ParentRunID   string `json:"parent_run_id"`
	PreviousRunID string `json:"previous_run_id"`
	Agent         string `json:"agent"`
	PID           int    `json:"pid"`
	PGID          int    `json:"pgid"`
	StartTime     string `json:"start_time"`
	EndTime       string `json:"end_time"`
	ExitCode      int    `json:"exit_code"`
	CWD           string `json:"cwd"`
	CommandLine   string `json:"commandline,omitempty"`
}

type uiFileContent struct {
	Name      string `json:"name"`
	Content   string `json:"content"`
	Modified  string `json:"modified"`
	SizeBytes int64  `json:"size_bytes,omitempty"`
}

type uiBusMessage struct {
	MsgID          string   `json:"msg_id"`
	TS             string   `json:"ts"`
	Type           string   `json:"type"`
	Message        string   `json:"message"`
	Parents        []string `json:"parents,omitempty"`
	RunID          string   `json:"run_id,omitempty"`
	AttachmentPath string   `json:"attachment_path,omitempty"`
	Project        string   `json:"project,omitempty"`
	Task           string   `json:"task,omitempty"`
}

func startUIAdapter(t *testing.T, rootDir, conductorURL string) *httptest.Server {
	adapter := &uiAdapter{
		rootDir:    rootDir,
		conductor:  conductorURL,
		httpClient: &http.Client{},
		pollDelay:  200 * time.Millisecond,
		heartbeat:  1 * time.Second,
	}
	mux := http.NewServeMux()
	mux.HandleFunc("/api/projects", adapter.handleProjects)
	mux.HandleFunc("/api/projects/", adapter.handleProject)
	return httptest.NewServer(mux)
}

func (a *uiAdapter) handleProjects(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		return
	}
	projects, err := a.listProjects()
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	writeJSON(w, http.StatusOK, uiProjectsResponse{Projects: projects})
}

func (a *uiAdapter) handleProject(w http.ResponseWriter, r *http.Request) {
	segments := splitPath(strings.TrimPrefix(r.URL.Path, "/api/projects/"))
	if len(segments) == 0 {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "project not found"})
		return
	}
	project := segments[0]
	if len(segments) == 1 {
		if r.Method != http.MethodGet {
			writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
			return
		}
		tasks, err := a.listTasks(project)
		if err != nil {
			writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
			return
		}
		projectInfo := uiProject{ID: project, TaskCount: len(tasks)}
		if len(tasks) > 0 {
			projectInfo.LastActivity = tasks[0].LastActivity
		} else {
			projectInfo.LastActivity = time.Now().UTC().Format(time.RFC3339Nano)
		}
		writeJSON(w, http.StatusOK, map[string]interface{}{
			"id":            projectInfo.ID,
			"last_activity": projectInfo.LastActivity,
			"task_count":    projectInfo.TaskCount,
			"tasks":         tasks,
		})
		return
	}
	if len(segments) == 2 && segments[1] == "tasks" {
		switch r.Method {
		case http.MethodGet:
			tasks, err := a.listTasks(project)
			if err != nil {
				writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
				return
			}
			writeJSON(w, http.StatusOK, uiTasksResponse{Tasks: tasks})
		case http.MethodPost:
			var payload struct {
				TaskID      string `json:"task_id"`
				Prompt      string `json:"prompt"`
				ProjectRoot string `json:"project_root"`
				AttachMode  string `json:"attach_mode"`
			}
			if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
				writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid payload"})
				return
			}
			req := api.TaskCreateRequest{
				ProjectID: project,
				TaskID:    payload.TaskID,
				AgentType: agentType,
				Prompt:    payload.Prompt,
			}
			buf, _ := json.Marshal(req)
			resp, err := a.httpClient.Post(a.conductor+"/api/v1/tasks", "application/json", bytes.NewReader(buf))
			if err != nil {
				writeJSON(w, http.StatusBadGateway, map[string]string{"error": err.Error()})
				return
			}
			_ = resp.Body.Close()
			writeJSON(w, http.StatusAccepted, map[string]string{"status": "started"})
		default:
			writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
		}
		return
	}
	if len(segments) == 3 && segments[1] == "tasks" {
		if r.Method != http.MethodGet {
			writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
			return
		}
		taskID := segments[2]
		task, err := a.getTaskDetail(project, taskID)
		if err != nil {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, task)
		return
	}
	if len(segments) == 3 && segments[1] == "bus" && segments[2] == "stream" {
		a.serveBusStream(w, r, project, "")
		return
	}
	if len(segments) == 4 && segments[1] == "tasks" && segments[3] == "file" {
		if r.Method != http.MethodGet {
			writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
			return
		}
		taskID := segments[2]
		a.serveFile(w, r, filepath.Join(a.rootDir, project, taskID))
		return
	}
	if len(segments) == 5 && segments[1] == "tasks" && segments[3] == "runs" {
		if r.Method != http.MethodGet {
			writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
			return
		}
		taskID := segments[2]
		runID := segments[4]
		run, err := a.getRunInfo(project, taskID, runID)
		if err != nil {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": err.Error()})
			return
		}
		writeJSON(w, http.StatusOK, run)
		return
	}
	if len(segments) == 5 && segments[1] == "tasks" && segments[3] == "logs" && segments[4] == "stream" {
		taskID := segments[2]
		runID := a.latestRunID(project, taskID)
		if runID == "" {
			writeJSON(w, http.StatusNotFound, map[string]string{"error": "run not found"})
			return
		}
		proxySSE(w, r, a.conductor+"/api/v1/runs/"+runID+"/stream")
		return
	}
	if len(segments) == 5 && segments[1] == "tasks" && segments[3] == "bus" && segments[4] == "stream" {
		a.serveBusStream(w, r, project, segments[2])
		return
	}
	if len(segments) == 6 && segments[1] == "tasks" && segments[3] == "runs" && segments[5] == "file" {
		if r.Method != http.MethodGet {
			writeJSON(w, http.StatusMethodNotAllowed, map[string]string{"error": "method not allowed"})
			return
		}
		taskID := segments[2]
		runID := segments[4]
		base := filepath.Join(a.rootDir, project, taskID, "runs", runID)
		a.serveFile(w, r, base)
		return
	}
	writeJSON(w, http.StatusNotFound, map[string]string{"error": "not found"})
}

func (a *uiAdapter) listProjects() ([]uiProject, error) {
	entries, err := os.ReadDir(a.rootDir)
	if err != nil {
		return nil, err
	}
	projects := make([]uiProject, 0)
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		project := entry.Name()
		tasks, err := a.listTasks(project)
		if err != nil {
			continue
		}
		lastActivity := time.Now().UTC().Format(time.RFC3339Nano)
		if len(tasks) > 0 {
			lastActivity = tasks[0].LastActivity
		}
		projects = append(projects, uiProject{
			ID:           project,
			LastActivity: lastActivity,
			TaskCount:    len(tasks),
		})
	}
	sort.Slice(projects, func(i, j int) bool {
		return projects[i].LastActivity > projects[j].LastActivity
	})
	return projects, nil
}

func (a *uiAdapter) listTasks(project string) ([]uiTaskSummary, error) {
	projectDir := filepath.Join(a.rootDir, project)
	entries, err := os.ReadDir(projectDir)
	if err != nil {
		if os.IsNotExist(err) {
			return []uiTaskSummary{}, nil
		}
		return nil, err
	}
	tasks := make([]uiTaskSummary, 0)
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		taskID := entry.Name()
		info, err := a.getTaskDetail(project, taskID)
		if err != nil {
			continue
		}
		tasks = append(tasks, uiTaskSummary{
			ID:           taskID,
			Name:         info.Name,
			Status:       info.Status,
			LastActivity: info.LastActivity,
			RunCount:     len(info.Runs),
		})
	}
	sort.Slice(tasks, func(i, j int) bool {
		return tasks[i].LastActivity > tasks[j].LastActivity
	})
	return tasks, nil
}

func (a *uiAdapter) getTaskDetail(project, taskID string) (uiTaskDetail, error) {
	taskDir := filepath.Join(a.rootDir, project, taskID)
	stat, err := os.Stat(taskDir)
	if err != nil {
		return uiTaskDetail{}, err
	}
	runs, err := a.listRunSummaries(project, taskID)
	if err != nil {
		return uiTaskDetail{}, err
	}
	status := "unknown"
	lastActivity := stat.ModTime().UTC().Format(time.RFC3339Nano)
	if len(runs) > 0 {
		status = runs[len(runs)-1].Status
		lastActivity = runs[len(runs)-1].StartTime
		if runs[len(runs)-1].EndTime != "" {
			lastActivity = runs[len(runs)-1].EndTime
		}
	}
	state := ""
	if data, err := os.ReadFile(filepath.Join(taskDir, "TASK_STATE.md")); err == nil {
		state = string(data)
	}
	_, doneErr := os.Stat(filepath.Join(taskDir, "DONE"))
	return uiTaskDetail{
		ID:           taskID,
		Name:         taskID,
		ProjectID:    project,
		Status:       status,
		LastActivity: lastActivity,
		CreatedAt:    stat.ModTime().UTC().Format(time.RFC3339Nano),
		Done:         doneErr == nil,
		State:        state,
		Runs:         runs,
	}, nil
}

func (a *uiAdapter) listRunSummaries(project, taskID string) ([]uiRunSummary, error) {
	runs := make([]uiRunSummary, 0)
	infos, err := a.readRunInfos(project, taskID)
	if err != nil {
		return runs, err
	}
	for _, info := range infos {
		runs = append(runs, uiRunSummary{
			ID:            info.RunID,
			Agent:         info.AgentType,
			Status:        mapStatus(info.Status),
			ExitCode:      info.ExitCode,
			StartTime:     info.StartTime.UTC().Format(time.RFC3339Nano),
			EndTime:       formatTime(info.EndTime),
			ParentRunID:   info.ParentRunID,
			PreviousRunID: info.PreviousRunID,
		})
	}
	return runs, nil
}

func (a *uiAdapter) getRunInfo(project, taskID, runID string) (uiRunInfo, error) {
	path := filepath.Join(a.rootDir, project, taskID, "runs", runID, "run-info.yaml")
	info, err := storage.ReadRunInfo(path)
	if err != nil {
		return uiRunInfo{}, err
	}
	return uiRunInfo{
		Version:       info.Version,
		RunID:         info.RunID,
		ProjectID:     info.ProjectID,
		TaskID:        info.TaskID,
		ParentRunID:   info.ParentRunID,
		PreviousRunID: info.PreviousRunID,
		Agent:         info.AgentType,
		PID:           info.PID,
		PGID:          info.PGID,
		StartTime:     info.StartTime.UTC().Format(time.RFC3339Nano),
		EndTime:       formatTime(info.EndTime),
		ExitCode:      info.ExitCode,
		CWD:           info.CWD,
		CommandLine:   info.CommandLine,
	}, nil
}

func (a *uiAdapter) readRunInfos(project, taskID string) ([]*storage.RunInfo, error) {
	runsDir := filepath.Join(a.rootDir, project, taskID, "runs")
	entries, err := os.ReadDir(runsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	infos := make([]*storage.RunInfo, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		path := filepath.Join(runsDir, entry.Name(), "run-info.yaml")
		info, err := storage.ReadRunInfo(path)
		if err != nil {
			continue
		}
		infos = append(infos, info)
	}
	sort.Slice(infos, func(i, j int) bool {
		return infos[i].StartTime.Before(infos[j].StartTime)
	})
	return infos, nil
}

func (a *uiAdapter) latestRunID(project, taskID string) string {
	infos, err := a.readRunInfos(project, taskID)
	if err != nil || len(infos) == 0 {
		return ""
	}
	return infos[len(infos)-1].RunID
}

func (a *uiAdapter) serveFile(w http.ResponseWriter, r *http.Request, base string) {
	name := r.URL.Query().Get("name")
	if strings.TrimSpace(name) == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "name required"})
		return
	}
	path := filepath.Join(base, filepath.Clean(name))
	data, err := os.ReadFile(path)
	if err != nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "file not found"})
		return
	}
	info, err := os.Stat(path)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "stat failed"})
		return
	}
	writeJSON(w, http.StatusOK, uiFileContent{
		Name:      name,
		Content:   string(data),
		Modified:  info.ModTime().UTC().Format(time.RFC3339Nano),
		SizeBytes: info.Size(),
	})
}

func (a *uiAdapter) serveBusStream(w http.ResponseWriter, r *http.Request, project, taskID string) {
	busPath := filepath.Join(a.rootDir, project, "PROJECT-MESSAGE-BUS.md")
	if taskID != "" {
		busPath = filepath.Join(a.rootDir, project, taskID, "TASK-MESSAGE-BUS.md")
	}
	bus, err := messagebus.NewMessageBus(busPath)
	if err != nil {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": err.Error()})
		return
	}
	flusher, ok := w.(http.Flusher)
	if !ok {
		writeJSON(w, http.StatusInternalServerError, map[string]string{"error": "streaming unsupported"})
		return
	}
	header := w.Header()
	header.Set("Content-Type", "text/event-stream")
	header.Set("Cache-Control", "no-cache")
	header.Set("Connection", "keep-alive")

	ctx := r.Context()
	lastID := strings.TrimSpace(r.Header.Get("Last-Event-ID"))
	pollTicker := time.NewTicker(a.pollDelay)
	defer pollTicker.Stop()
	heartbeat := time.NewTicker(a.heartbeat)
	defer heartbeat.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-heartbeat.C:
			_, _ = fmt.Fprint(w, "event: heartbeat\ndata: {}\n\n")
			flusher.Flush()
		case <-pollTicker.C:
			messages, err := bus.ReadMessages(lastID)
			if err != nil {
				if errors.Is(err, messagebus.ErrSinceIDNotFound) {
					lastID = ""
				}
				continue
			}
			for _, msg := range messages {
				if msg == nil {
					continue
				}
				parentIDs := make([]string, 0, len(msg.Parents))
				for _, p := range msg.Parents {
					if p.MsgID != "" {
						parentIDs = append(parentIDs, p.MsgID)
					}
				}
				payload := uiBusMessage{
					MsgID:   msg.MsgID,
					TS:      msg.Timestamp.UTC().Format(time.RFC3339Nano),
					Type:    msg.Type,
					Message: msg.Body,
					Parents: parentIDs,
					RunID:   msg.RunID,
					Project: project,
					Task:    taskID,
				}
				data, err := json.Marshal(payload)
				if err != nil {
					continue
				}
				_, _ = fmt.Fprintf(w, "event: message\ndata: %s\n\n", data)
				flusher.Flush()
				lastID = msg.MsgID
			}
		}
	}
}

func proxySSE(w http.ResponseWriter, r *http.Request, upstream string) {
	ctx := r.Context()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, upstream, nil)
	if err != nil {
		writeJSON(w, http.StatusBadGateway, map[string]string{"error": err.Error()})
		return
	}
	if lastID := strings.TrimSpace(r.Header.Get("Last-Event-ID")); lastID != "" {
		req.Header.Set("Last-Event-ID", lastID)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		writeJSON(w, http.StatusBadGateway, map[string]string{"error": err.Error()})
		return
	}
	defer resp.Body.Close()

	for key, values := range resp.Header {
		for _, value := range values {
			w.Header().Add(key, value)
		}
	}
	w.WriteHeader(resp.StatusCode)

	flusher, ok := w.(http.Flusher)
	if !ok {
		_, _ = io.Copy(w, resp.Body)
		return
	}

	reader := bufio.NewReader(resp.Body)
	for {
		line, err := reader.ReadBytes('\n')
		if len(line) > 0 {
			_, _ = w.Write(line)
			flusher.Flush()
		}
		if err != nil {
			return
		}
	}
}

func splitPath(path string) []string {
	trimmed := strings.Trim(path, "/")
	if trimmed == "" {
		return nil
	}
	parts := strings.Split(trimmed, "/")
	out := parts[:0]
	for _, part := range parts {
		if part == "" {
			continue
		}
		out = append(out, part)
	}
	return out
}

func mapStatus(status string) string {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case storage.StatusRunning:
		return storage.StatusRunning
	case storage.StatusCompleted:
		return storage.StatusCompleted
	case storage.StatusFailed:
		return storage.StatusFailed
	default:
		return "unknown"
	}
}

func formatTime(t time.Time) string {
	if t.IsZero() {
		return ""
	}
	return t.UTC().Format(time.RFC3339Nano)
}

func writeJSON(w http.ResponseWriter, status int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}

func startFrontend(t *testing.T, apiBaseURL string) (string, func()) {
	root := repoRoot(t)
	frontendDir := filepath.Join(root, "frontend")
	if _, err := os.Stat(frontendDir); err != nil {
		t.Fatalf("frontend directory missing: %v", err)
	}

	port := pickPort(t)
	baseURL := fmt.Sprintf("http://127.0.0.1:%d", port)

	cmd := exec.Command("npm", "run", "dev", "--", "--host", "127.0.0.1", "--port", strconv.Itoa(port))
	cmd.Dir = frontendDir
	cmd.Env = append(os.Environ(), "VITE_API_BASE_URL="+apiBaseURL)
	var output bytes.Buffer
	cmd.Stdout = &output
	cmd.Stderr = &output
	if err := cmd.Start(); err != nil {
		t.Fatalf("start frontend: %v", err)
	}

	ready := waitForHTTP(baseURL, 45*time.Second)
	if !ready {
		_ = cmd.Process.Kill()
		t.Fatalf("frontend did not start: %s", output.String())
	}

	stop := func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = cmd.Process.Signal(os.Interrupt)
		done := make(chan error, 1)
		go func() { done <- cmd.Wait() }()
		select {
		case <-ctx.Done():
			_ = cmd.Process.Kill()
		case <-done:
		}
	}

	return baseURL, stop
}

func runPlaywright(t *testing.T, baseURL, taskID string) {
	spec := fmt.Sprintf(`const { test, expect } = require('@playwright/test');

test('ui live monitoring', async ({ page }) => {
  const baseURL = process.env.PLAYWRIGHT_TEST_BASE_URL || '%s';
  await page.goto(baseURL);
  await expect(page.getByText('Conductor Loop Monitor')).toBeVisible();

  await page.getByRole('button', { name: '%s' }).click();
  await page.getByTestId('task-item-%s').click();

  const logStream = page.locator('.log-stream');
  await expect(logStream).toContainText('Progress: 5/10', { timeout: 20000 });

  const status = page.locator('.status-pill');
  await expect(status).toContainText('running', { timeout: 10000 });

  await expect(logStream).toContainText('Complete', { timeout: 40000 });

  const logText = await logStream.textContent();
  expect(logText).toContain('Progress: 10/10');
  expect(logText).toContain('Complete');

  await page.reload();
  await page.getByRole('button', { name: '%s' }).click();
  await page.getByTestId('task-item-%s').click();
  await expect(page.locator('.status-pill')).toContainText('completed', { timeout: 20000 });
});`, baseURL, projectID, taskID, projectID, taskID)

	tmpDir := t.TempDir()
	specPath := filepath.Join(tmpDir, "acceptance-ui.spec.js")
	if err := os.WriteFile(specPath, []byte(spec), 0o600); err != nil {
		t.Fatalf("write playwright spec: %v", err)
	}

	cmd := exec.Command("npx", "playwright", "test", specPath)
	cmd.Dir = repoRoot(t)
	cmd.Env = append(os.Environ(), "PLAYWRIGHT_TEST_BASE_URL="+baseURL)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("playwright test failed: %v\n%s", err, string(output))
	}
}

func pickPort(t *testing.T) int {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("listen: %v", err)
	}
	defer listener.Close()
	addr := listener.Addr().(*net.TCPAddr)
	return addr.Port
}

func waitForHTTP(url string, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		resp, err := http.Get(url)
		if err == nil {
			_ = resp.Body.Close()
			if resp.StatusCode >= 200 && resp.StatusCode < 500 {
				return true
			}
		}
		time.Sleep(200 * time.Millisecond)
	}
	return false
}
