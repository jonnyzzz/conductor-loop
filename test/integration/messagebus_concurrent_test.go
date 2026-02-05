package integration_test

import (
	"bytes"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"sync"
	"testing"
	"time"

	"github.com/jonnyzzz/conductor-loop/internal/messagebus"
)

func TestConcurrentAgentWrites(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping message bus concurrency test in short mode")
	}

	const (
		numAgents = 10
		numMsgs   = 100
	)
	runMessageBusConcurrency(t, numAgents, numMsgs)
}

func TestMessageBusConcurrency(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping message bus stress test in short mode")
	}
	runMessageBusConcurrency(t, 10, 100)
}

func TestMessageBusOrdering(t *testing.T) {
	path := filepath.Join(t.TempDir(), "TASK-MESSAGE-BUS.md")
	bus, err := messagebus.NewMessageBus(path)
	if err != nil {
		t.Fatalf("new message bus: %v", err)
	}

	const (
		writers   = 5
		perWriter = 20
		total     = writers * perWriter
	)

	var (
		mu    sync.Mutex
		order []string
	)
	errCh := make(chan error, writers)

	var wg sync.WaitGroup
	wg.Add(writers)
	for i := 0; i < writers; i++ {
		go func(worker int) {
			defer wg.Done()
			for j := 0; j < perWriter; j++ {
				msgID, err := bus.AppendMessage(&messagebus.Message{
					Type:      "FACT",
					ProjectID: "project",
					TaskID:    "task-ord",
					RunID:     fmt.Sprintf("worker-%d", worker),
					Body:      fmt.Sprintf("worker %d message %d", worker, j),
				})
				if err != nil {
					errCh <- err
					return
				}
				mu.Lock()
				order = append(order, msgID)
				mu.Unlock()
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

	if len(order) != total {
		t.Fatalf("expected %d recorded messages, got %d", total, len(order))
	}
	messages, err := bus.ReadMessages("")
	if err != nil {
		t.Fatalf("read messages: %v", err)
	}
	if len(messages) != total {
		t.Fatalf("expected %d messages, got %d", total, len(messages))
	}
	for i, msg := range messages {
		if order[i] != msg.MsgID {
			t.Fatalf("message order mismatch at %d: got %q want %q", i, msg.MsgID, order[i])
		}
	}
}

func runMessageBusConcurrency(t *testing.T, numAgents, numMsgs int) {
	t.Helper()

	path := filepath.Join(t.TempDir(), "TASK-MESSAGE-BUS.md")
	bus, err := messagebus.NewMessageBus(path)
	if err != nil {
		t.Fatalf("new message bus: %v", err)
	}

	totalMsgs := numAgents * numMsgs
	var wg sync.WaitGroup
	errCh := make(chan error, numAgents)
	for i := 0; i < numAgents; i++ {
		wg.Add(1)
		go func(agentID int) {
			defer wg.Done()
			for j := 0; j < numMsgs; j++ {
				_, err := bus.AppendMessage(&messagebus.Message{
					Type:      "FACT",
					ProjectID: "project",
					TaskID:    "task-001",
					RunID:     fmt.Sprintf("agent-%d", agentID),
					Body:      fmt.Sprintf("agent %d message %d", agentID, j),
				})
				if err != nil {
					errCh <- err
					return
				}
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

	messages, err := bus.ReadMessages("")
	if err != nil {
		t.Fatalf("read messages: %v", err)
	}
	if len(messages) != totalMsgs {
		t.Fatalf("expected %d messages, got %d", totalMsgs, len(messages))
	}
	ids := make(map[string]struct{})
	for _, msg := range messages {
		if msg == nil || msg.MsgID == "" {
			t.Fatalf("missing msg id")
		}
		if _, exists := ids[msg.MsgID]; exists {
			t.Fatalf("duplicate msg id %q", msg.MsgID)
		}
		ids[msg.MsgID] = struct{}{}
	}
}

func TestFlockContention(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("flock contention test is unix-only")
	}

	path := filepath.Join(t.TempDir(), "TASK-MESSAGE-BUS.md")
	const (
		workers = 3
		holdMs  = 50
	)

	cmds := make([]*exec.Cmd, 0, workers)
	outputs := make([]*bytes.Buffer, 0, workers)
	readyPaths := make([]string, 0, workers)
	for i := 0; i < workers; i++ {
		ready := filepath.Join(t.TempDir(), fmt.Sprintf("lock-ready-%d", i))
		cmd := exec.Command(os.Args[0])
		cmd.Env = append(os.Environ(),
			envHelperMode+"="+helperLock,
			envBusPath+"="+path,
			envReady+"="+ready,
			envHoldMs+"="+strconv.Itoa(holdMs),
		)
		var out bytes.Buffer
		cmd.Stdout = &out
		cmd.Stderr = &out
		if err := cmd.Start(); err != nil {
			t.Fatalf("start lock helper %d: %v", i, err)
		}
		cmds = append(cmds, cmd)
		outputs = append(outputs, &out)
		readyPaths = append(readyPaths, ready)
	}

	deadline := time.Now().Add(3 * time.Second)
	for i, ready := range readyPaths {
		remaining := time.Until(deadline)
		if remaining <= 0 {
			t.Fatalf("timed out waiting for lock helper %d", i)
		}
		if err := waitForFile(ready, remaining); err != nil {
			t.Fatalf("wait for lock helper %d: %v", i, err)
		}
	}

	for i, cmd := range cmds {
		if err := cmd.Wait(); err != nil {
			t.Fatalf("lock helper %d failed: %v\n%s", i, err, outputs[i].String())
		}
	}
}
