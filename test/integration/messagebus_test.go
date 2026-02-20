package integration_test

import (
	"bytes"
	stderrors "errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/jonnyzzz/conductor-loop/internal/messagebus"
	"gopkg.in/yaml.v3"
)

const (
	envHelperMode  = "MESSAGEBUS_HELPER_MODE"
	envBusPath     = "MESSAGEBUS_PATH"
	envCount       = "MESSAGEBUS_COUNT"
	envWorker      = "MESSAGEBUS_WORKER"
	envReady       = "MESSAGEBUS_READY"
	envHoldMs      = "MESSAGEBUS_HOLD_MS"
	helperAppend   = "append"
	helperLock     = "lock"
	helperCrash    = "crash"
	defaultHoldMs  = 500
	defaultPollDur = 2 * time.Second
)

func TestMain(m *testing.M) {
	if os.Getenv(envCodexHelperMode) != "" {
		if err := runCodexHelper(); err != nil {
			_, _ = fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		os.Exit(0)
	}
	mode := os.Getenv(envHelperMode)
	if mode != "" {
		if err := runHelper(mode); err != nil {
			_, _ = fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
		os.Exit(0)
	}
	os.Exit(m.Run())
}

func TestMessageIDUniqueness(t *testing.T) {
	ids := make(map[string]struct{})
	for i := 0; i < 1000; i++ {
		id := messagebus.GenerateMessageID()
		if _, exists := ids[id]; exists {
			t.Fatalf("duplicate msg id %q", id)
		}
		ids[id] = struct{}{}
	}
}

func TestConcurrentAppend(t *testing.T) {
	path := filepath.Join(t.TempDir(), "TASK-MESSAGE-BUS.md")
	const (
		workers    = 10
		perWorker  = 100
		projectID  = "project"
		taskID     = "task-001"
		typeString = "FACT"
	)

	cmds := make([]*exec.Cmd, 0, workers)
	outputs := make([]*bytes.Buffer, 0, workers)
	for i := 0; i < workers; i++ {
		cmd := exec.Command(os.Args[0])
		cmd.Env = append(os.Environ(),
			envHelperMode+"="+helperAppend,
			envBusPath+"="+path,
			envCount+"="+strconv.Itoa(perWorker),
			envWorker+"="+strconv.Itoa(i),
			"MESSAGEBUS_PROJECT="+projectID,
			"MESSAGEBUS_TASK="+taskID,
			"MESSAGEBUS_TYPE="+typeString,
		)
		var out bytes.Buffer
		cmd.Stdout = &out
		cmd.Stderr = &out
		if err := cmd.Start(); err != nil {
			t.Fatalf("start worker %d: %v", i, err)
		}
		cmds = append(cmds, cmd)
		outputs = append(outputs, &out)
	}

	for i, cmd := range cmds {
		if err := cmd.Wait(); err != nil {
			t.Fatalf("worker %d failed: %v\n%s", i, err, outputs[i].String())
		}
	}

	bus, err := messagebus.NewMessageBus(path)
	if err != nil {
		t.Fatalf("new message bus: %v", err)
	}
	messages, err := bus.ReadMessages("")
	if err != nil {
		t.Fatalf("read messages: %v", err)
	}
	if len(messages) != workers*perWorker {
		t.Fatalf("expected %d messages, got %d", workers*perWorker, len(messages))
	}
	ids := make(map[string]struct{})
	for _, msg := range messages {
		if msg == nil {
			t.Fatalf("nil message")
		}
		if _, exists := ids[msg.MsgID]; exists {
			t.Fatalf("duplicate msg id %q", msg.MsgID)
		}
		ids[msg.MsgID] = struct{}{}
	}
}

func TestLockTimeout(t *testing.T) {
	path := filepath.Join(t.TempDir(), "TASK-MESSAGE-BUS.md")
	readyPath := filepath.Join(t.TempDir(), "lock-ready")

	cmd := exec.Command(os.Args[0])
	cmd.Env = append(os.Environ(),
		envHelperMode+"="+helperLock,
		envBusPath+"="+path,
		envReady+"="+readyPath,
		envHoldMs+"="+strconv.Itoa(defaultHoldMs),
	)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	if err := cmd.Start(); err != nil {
		t.Fatalf("start lock helper: %v", err)
	}
	if err := waitForFile(readyPath, defaultPollDur); err != nil {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
		t.Fatalf("wait for lock helper: %v\n%s", err, out.String())
	}

	bus, err := messagebus.NewMessageBus(path, messagebus.WithLockTimeout(100*time.Millisecond), messagebus.WithMaxRetries(1))
	if err != nil {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
		t.Fatalf("new message bus: %v", err)
	}
	_, err = bus.AppendMessage(&messagebus.Message{
		Type:      "FACT",
		ProjectID: "project",
		TaskID:    "task-001",
		Body:      "lock timeout",
	})
	if err == nil {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
		t.Fatalf("expected lock timeout error")
	}
	if !stderrors.Is(err, messagebus.ErrLockTimeout) {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
		t.Fatalf("expected lock timeout error, got %v", err)
	}

	if err := cmd.Wait(); err != nil {
		t.Fatalf("lock helper failed: %v\n%s", err, out.String())
	}
}

func TestCrashRecovery(t *testing.T) {
	path := filepath.Join(t.TempDir(), "TASK-MESSAGE-BUS.md")
	bus, err := messagebus.NewMessageBus(path)
	if err != nil {
		t.Fatalf("new message bus: %v", err)
	}
	if _, err := bus.AppendMessage(&messagebus.Message{
		Type:      "FACT",
		ProjectID: "project",
		TaskID:    "task-001",
		Body:      "before crash",
	}); err != nil {
		t.Fatalf("append message: %v", err)
	}

	readyPath := filepath.Join(t.TempDir(), "crash-ready")
	cmd := exec.Command(os.Args[0])
	cmd.Env = append(os.Environ(),
		envHelperMode+"="+helperCrash,
		envBusPath+"="+path,
		envReady+"="+readyPath,
	)
	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &out
	if err := cmd.Start(); err != nil {
		t.Fatalf("start crash helper: %v", err)
	}
	if err := waitForFile(readyPath, defaultPollDur); err != nil {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
		t.Fatalf("wait for crash helper: %v\n%s", err, out.String())
	}
	_ = cmd.Process.Kill()
	_ = cmd.Wait()

	if _, err := bus.AppendMessage(&messagebus.Message{
		Type:      "FACT",
		ProjectID: "project",
		TaskID:    "task-001",
		Body:      "after crash",
	}); err != nil {
		t.Fatalf("append after crash: %v", err)
	}

	messages, err := bus.ReadMessages("")
	if err != nil {
		t.Fatalf("read messages: %v", err)
	}
	if len(messages) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(messages))
	}
}

func TestReadWhileWriting(t *testing.T) {
	path := filepath.Join(t.TempDir(), "TASK-MESSAGE-BUS.md")
	bus, err := messagebus.NewMessageBus(path, messagebus.WithPollInterval(10*time.Millisecond))
	if err != nil {
		t.Fatalf("new message bus: %v", err)
	}

	const total = 50
	errCh := make(chan error, 1)
	go func() {
		for i := 0; i < total; i++ {
			if _, err := bus.AppendMessage(&messagebus.Message{
				Type:      "FACT",
				ProjectID: "project",
				TaskID:    "task-001",
				Body:      fmt.Sprintf("message %d", i),
			}); err != nil {
				errCh <- err
				return
			}
			time.Sleep(5 * time.Millisecond)
		}
		errCh <- nil
	}()

	lastID := ""
	read := 0
	for read < total {
		messages, err := pollWithTimeout(bus, lastID, defaultPollDur)
		if err != nil {
			t.Fatalf("poll for new messages: %v", err)
		}
		if len(messages) == 0 {
			t.Fatalf("poll returned no messages")
		}
		read += len(messages)
		lastID = messages[len(messages)-1].MsgID
	}

	if err := <-errCh; err != nil {
		t.Fatalf("writer error: %v", err)
	}
}

func pollWithTimeout(bus *messagebus.MessageBus, lastID string, timeout time.Duration) ([]*messagebus.Message, error) {
	resultCh := make(chan struct {
		messages []*messagebus.Message
		err      error
	}, 1)
	go func() {
		messages, err := bus.PollForNew(lastID)
		resultCh <- struct {
			messages []*messagebus.Message
			err      error
		}{messages: messages, err: err}
	}()

	select {
	case result := <-resultCh:
		return result.messages, result.err
	case <-time.After(timeout):
		return nil, fmt.Errorf("poll timed out")
	}
}

func waitForFile(path string, timeout time.Duration) error {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		if _, err := os.Stat(path); err == nil {
			return nil
		}
		time.Sleep(10 * time.Millisecond)
	}
	return fmt.Errorf("timed out waiting for %s", path)
}

func runHelper(mode string) error {
	switch mode {
	case helperAppend:
		return runAppendHelper()
	case helperLock:
		return runLockHelper()
	case helperCrash:
		return runCrashHelper()
	default:
		return fmt.Errorf("unknown helper mode %q", mode)
	}
}

func runAppendHelper() error {
	path := os.Getenv(envBusPath)
	if path == "" {
		return fmt.Errorf("missing %s", envBusPath)
	}
	count, err := strconv.Atoi(os.Getenv(envCount))
	if err != nil || count <= 0 {
		return fmt.Errorf("invalid %s", envCount)
	}
	worker, err := strconv.Atoi(os.Getenv(envWorker))
	if err != nil {
		return fmt.Errorf("invalid %s", envWorker)
	}
	projectID := os.Getenv("MESSAGEBUS_PROJECT")
	if projectID == "" {
		projectID = "project"
	}
	taskID := os.Getenv("MESSAGEBUS_TASK")
	if taskID == "" {
		taskID = "task-001"
	}
	typeString := os.Getenv("MESSAGEBUS_TYPE")
	if typeString == "" {
		typeString = "FACT"
	}

	bus, err := messagebus.NewMessageBus(path)
	if err != nil {
		return err
	}
	for i := 0; i < count; i++ {
		if _, err := bus.AppendMessage(&messagebus.Message{
			Type:      typeString,
			ProjectID: projectID,
			TaskID:    taskID,
			RunID:     fmt.Sprintf("run-%d", worker),
			Body:      fmt.Sprintf("worker %d message %d", worker, i),
		}); err != nil {
			return err
		}
	}
	return nil
}

func runLockHelper() error {
	path := os.Getenv(envBusPath)
	if path == "" {
		return fmt.Errorf("missing %s", envBusPath)
	}
	ready := os.Getenv(envReady)
	if ready == "" {
		return fmt.Errorf("missing %s", envReady)
	}
	holdMs := defaultHoldMs
	if value := os.Getenv(envHoldMs); value != "" {
		parsed, err := strconv.Atoi(value)
		if err != nil {
			return fmt.Errorf("invalid %s", envHoldMs)
		}
		holdMs = parsed
	}
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0o644)
	if err != nil {
		return err
	}
	defer file.Close()
	if err := messagebus.LockExclusive(file, 5*time.Second); err != nil {
		return err
	}
	defer messagebus.Unlock(file)
	if err := os.WriteFile(ready, []byte("ready"), 0o644); err != nil {
		return err
	}
	time.Sleep(time.Duration(holdMs) * time.Millisecond)
	return nil
}

func runCrashHelper() error {
	path := os.Getenv(envBusPath)
	if path == "" {
		return fmt.Errorf("missing %s", envBusPath)
	}
	ready := os.Getenv(envReady)
	if ready == "" {
		return fmt.Errorf("missing %s", envReady)
	}
	file, err := os.OpenFile(path, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0o644)
	if err != nil {
		return err
	}
	if err := messagebus.LockExclusive(file, 5*time.Second); err != nil {
		_ = file.Close()
		return err
	}
	msg := &messagebus.Message{
		MsgID:     messagebus.GenerateMessageID(),
		Timestamp: time.Now().UTC(),
		Type:      "FACT",
		ProjectID: "project",
		TaskID:    "task-001",
		RunID:     "run-crash",
		Body:      "partial write",
	}
	data, err := serializeMessageForTest(msg)
	if err != nil {
		_ = file.Close()
		return err
	}
	cut := len(data) / 2
	delimiter := []byte("---\n")
	if idx := bytes.Index(data[len(delimiter):], delimiter); idx > -1 {
		second := idx + len(delimiter)
		if cut > second {
			cut = second - 1
		}
	}
	for cut > 0 && data[cut-1] == '\n' {
		cut--
	}
	if cut > 0 {
		if _, err := file.Write(data[:cut]); err != nil {
			_ = file.Close()
			return err
		}
	}
	if err := os.WriteFile(ready, []byte("ready"), 0o644); err != nil {
		_ = file.Close()
		return err
	}
	time.Sleep(10 * time.Second)
	return nil
}

func serializeMessageForTest(msg *messagebus.Message) ([]byte, error) {
	header, err := yaml.Marshal(msg)
	if err != nil {
		return nil, err
	}
	var buf bytes.Buffer
	buf.WriteString("---\n")
	buf.Write(header)
	if len(header) == 0 || header[len(header)-1] != '\n' {
		buf.WriteByte('\n')
	}
	buf.WriteString("---\n")
	if msg.Body != "" {
		buf.WriteString(msg.Body)
	}
	if !strings.HasSuffix(msg.Body, "\n") {
		buf.WriteByte('\n')
	}
	return buf.Bytes(), nil
}
