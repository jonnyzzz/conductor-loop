package main

import (
	"bytes"
	"fmt"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"syscall"
	"testing"
	"time"

	"github.com/jonnyzzz/conductor-loop/internal/messagebus"
)

// captureStdout redirects os.Stdout for the duration of fn and returns the captured output.
func captureStdout(t *testing.T, fn func()) string {
	t.Helper()
	r, w, err := os.Pipe()
	if err != nil {
		t.Fatalf("pipe: %v", err)
	}
	old := os.Stdout
	os.Stdout = w
	fn()
	w.Close()
	os.Stdout = old
	var buf bytes.Buffer
	if _, err := buf.ReadFrom(r); err != nil {
		t.Fatalf("read pipe: %v", err)
	}
	return buf.String()
}

// findFreePort returns an available TCP port on localhost.
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

func TestServeCmd_StartsServer(t *testing.T) {
	port := findFreePort(t)
	root := t.TempDir()

	errCh := make(chan error, 1)
	go func() {
		errCh <- runServe("127.0.0.1", port, root, "")
	}()

	addr := fmt.Sprintf("http://127.0.0.1:%d/api/v1/health", port)
	var (
		resp *http.Response
		err  error
	)
	for i := 0; i < 30; i++ {
		time.Sleep(100 * time.Millisecond)
		resp, err = http.Get(addr) //nolint:noctx
		if err == nil {
			break
		}
	}
	if err != nil {
		t.Fatalf("server did not start: %v", err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 from health endpoint, got %d", resp.StatusCode)
	}

	// Stop the server via SIGTERM
	p, _ := os.FindProcess(os.Getpid())
	_ = p.Signal(syscall.SIGTERM)

	select {
	case e := <-errCh:
		if e != nil {
			t.Fatalf("serve error after shutdown: %v", e)
		}
	case <-time.After(5 * time.Second):
		t.Fatal("server did not stop within timeout")
	}
}

func TestBusPostCmd_PostsMessage(t *testing.T) {
	busPath := filepath.Join(t.TempDir(), "bus.yaml")

	cmd := newRootCmd()
	cmd.SetArgs([]string{
		"bus", "post",
		"--bus", busPath,
		"--project", "my-project",
		"--task", "my-task",
		"--type", "INFO",
		"--body", "hello world",
	})

	var stdout string
	var runErr error
	stdout = captureStdout(t, func() {
		runErr = cmd.Execute()
	})
	if runErr != nil {
		t.Fatalf("bus post failed: %v", runErr)
	}
	if !strings.Contains(stdout, "msg_id:") {
		t.Errorf("expected stdout to contain 'msg_id:', got: %q", stdout)
	}

	// Verify message was written to the bus
	bus, err := messagebus.NewMessageBus(busPath)
	if err != nil {
		t.Fatalf("open bus: %v", err)
	}
	messages, err := bus.ReadMessages("")
	if err != nil {
		t.Fatalf("read messages: %v", err)
	}
	if len(messages) != 1 {
		t.Fatalf("expected 1 message, got %d", len(messages))
	}
	msg := messages[0]
	if msg.ProjectID != "my-project" {
		t.Errorf("expected project my-project, got %q", msg.ProjectID)
	}
	if msg.TaskID != "my-task" {
		t.Errorf("expected task my-task, got %q", msg.TaskID)
	}
	if msg.Body != "hello world" {
		t.Errorf("expected body 'hello world', got %q", msg.Body)
	}
	if msg.Type != "INFO" {
		t.Errorf("expected type INFO, got %q", msg.Type)
	}
}

func TestBusReadCmd_ReadMessages(t *testing.T) {
	busPath := filepath.Join(t.TempDir(), "bus.yaml")

	bus, err := messagebus.NewMessageBus(busPath)
	if err != nil {
		t.Fatalf("create bus: %v", err)
	}

	for i := 1; i <= 7; i++ {
		_, err := bus.AppendMessage(&messagebus.Message{
			Type:      "INFO",
			ProjectID: "proj",
			Body:      fmt.Sprintf("message %d", i),
		})
		if err != nil {
			t.Fatalf("append message %d: %v", i, err)
		}
	}

	cmd := newRootCmd()
	cmd.SetArgs([]string{
		"bus", "read",
		"--bus", busPath,
		"--tail", "5",
	})

	var output string
	var runErr error
	output = captureStdout(t, func() {
		runErr = cmd.Execute()
	})
	if runErr != nil {
		t.Fatalf("bus read failed: %v", runErr)
	}
	// Should contain the last 5 messages (3–7), not the first two
	for i := 3; i <= 7; i++ {
		want := fmt.Sprintf("message %d", i)
		if !strings.Contains(output, want) {
			t.Errorf("expected output to contain %q, got:\n%s", want, output)
		}
	}
	for i := 1; i <= 2; i++ {
		notWant := fmt.Sprintf("message %d", i)
		if strings.Contains(output, notWant) {
			t.Errorf("expected output NOT to contain %q, got:\n%s", notWant, output)
		}
	}
}
