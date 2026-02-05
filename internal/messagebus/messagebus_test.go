package messagebus

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"testing"
	"time"
)

func TestNewMessageBusValidation(t *testing.T) {
	if _, err := NewMessageBus(""); err == nil {
		t.Fatalf("expected error for empty path")
	}
	if _, err := NewMessageBus("/tmp/bus", WithClock(nil)); err == nil {
		t.Fatalf("expected error for nil clock")
	}
	if _, err := NewMessageBus("/tmp/bus", WithLockTimeout(0)); err == nil {
		t.Fatalf("expected error for invalid lock timeout")
	}
	if _, err := NewMessageBus("/tmp/bus", WithPollInterval(0)); err == nil {
		t.Fatalf("expected error for invalid poll interval")
	}
}

func TestAppendAndReadMessage(t *testing.T) {
	path := filepath.Join(t.TempDir(), "TASK-MESSAGE-BUS.md")
	bus, err := NewMessageBus(path)
	if err != nil {
		t.Fatalf("NewMessageBus: %v", err)
	}
	msgID, err := bus.AppendMessage(&Message{
		Type:      "FACT",
		ProjectID: "project",
		TaskID:    "task",
		RunID:     "run",
		Body:      "hello",
	})
	if err != nil {
		t.Fatalf("AppendMessage: %v", err)
	}
	if msgID == "" {
		t.Fatalf("expected message id")
	}
	messages, err := bus.ReadMessages("")
	if err != nil {
		t.Fatalf("ReadMessages: %v", err)
	}
	if len(messages) != 1 {
		t.Fatalf("expected 1 message, got %d", len(messages))
	}
	if messages[0].Body != "hello" {
		t.Fatalf("unexpected body: %q", messages[0].Body)
	}
}

func TestReadMessagesSinceNotFound(t *testing.T) {
	path := filepath.Join(t.TempDir(), "TASK-MESSAGE-BUS.md")
	bus, err := NewMessageBus(path)
	if err != nil {
		t.Fatalf("NewMessageBus: %v", err)
	}
	_, _ = bus.AppendMessage(&Message{Type: "FACT", ProjectID: "project", Body: "hello"})
	if _, err := bus.ReadMessages("missing"); err == nil {
		t.Fatalf("expected since id error")
	} else if !errors.Is(err, ErrSinceIDNotFound) {
		t.Fatalf("expected ErrSinceIDNotFound, got %v", err)
	}
}

func TestAppendMessageValidation(t *testing.T) {
	var bus *MessageBus
	if _, err := bus.AppendMessage(&Message{}); err == nil {
		t.Fatalf("expected error for nil bus")
	}
	path := filepath.Join(t.TempDir(), "TASK-MESSAGE-BUS.md")
	bus, err := NewMessageBus(path)
	if err != nil {
		t.Fatalf("NewMessageBus: %v", err)
	}
	if _, err := bus.AppendMessage(nil); err == nil {
		t.Fatalf("expected error for nil message")
	}
	if _, err := bus.AppendMessage(&Message{ProjectID: "project"}); err == nil {
		t.Fatalf("expected error for empty type")
	}
	if _, err := bus.AppendMessage(&Message{Type: "FACT"}); err == nil {
		t.Fatalf("expected error for empty project id")
	}
}

func TestPollForNew(t *testing.T) {
	path := filepath.Join(t.TempDir(), "TASK-MESSAGE-BUS.md")
	bus, err := NewMessageBus(path, WithPollInterval(10*time.Millisecond))
	if err != nil {
		t.Fatalf("NewMessageBus: %v", err)
	}

	var wg sync.WaitGroup
	wg.Add(1)
	go func() {
		defer wg.Done()
		time.Sleep(20 * time.Millisecond)
		_, _ = bus.AppendMessage(&Message{Type: "FACT", ProjectID: "project", Body: "hello"})
	}()

	msgs, err := bus.PollForNew("")
	if err != nil {
		t.Fatalf("PollForNew: %v", err)
	}
	if len(msgs) == 0 {
		t.Fatalf("expected messages")
	}
	wg.Wait()
}

func TestValidateBusPathSymlink(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink behavior differs on windows")
	}
	dir := t.TempDir()
	path := filepath.Join(dir, "TASK-MESSAGE-BUS.md")
	if err := os.WriteFile(path, []byte(""), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}
	symlink := filepath.Join(dir, "link.md")
	if err := os.Symlink(path, symlink); err != nil {
		t.Fatalf("symlink: %v", err)
	}
	if err := validateBusPath(symlink); err == nil {
		t.Fatalf("expected symlink error")
	}
}

func TestGenerateMessageIDUniqueness(t *testing.T) {
	ids := make(map[string]struct{})
	for i := 0; i < 1000; i++ {
		id := GenerateMessageID()
		if _, exists := ids[id]; exists {
			t.Fatalf("duplicate id: %s", id)
		}
		ids[id] = struct{}{}
	}
}

func TestAppendEntryValidation(t *testing.T) {
	if err := appendEntry(nil, []byte("x")); err == nil {
		t.Fatalf("expected error for nil file")
	}
}

func TestWriteAllShortWrite(t *testing.T) {
	w := &shortWriter{}
	if err := writeAll(w, []byte("data")); err == nil {
		t.Fatalf("expected short write error")
	}
}

func TestWriteAllError(t *testing.T) {
	w := &errorWriter{}
	if err := writeAll(w, []byte("data")); err == nil {
		t.Fatalf("expected error")
	}
}

func TestUnlockNil(t *testing.T) {
	if err := Unlock(nil); err == nil {
		t.Fatalf("expected error for nil file")
	}
}

func TestLockExclusiveInvalidTimeout(t *testing.T) {
	path := filepath.Join(t.TempDir(), "bus.lock")
	file, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0o644)
	if err != nil {
		t.Fatalf("open file: %v", err)
	}
	defer file.Close()
	if err := LockExclusive(file, 0); err == nil {
		t.Fatalf("expected error for invalid timeout")
	}
}

func TestReadMessagesEmpty(t *testing.T) {
	path := filepath.Join(t.TempDir(), "TASK-MESSAGE-BUS.md")
	bus, err := NewMessageBus(path)
	if err != nil {
		t.Fatalf("NewMessageBus: %v", err)
	}
	messages, err := bus.ReadMessages("")
	if err != nil {
		t.Fatalf("ReadMessages: %v", err)
	}
	if len(messages) != 0 {
		t.Fatalf("expected empty messages")
	}
}

func TestFilterSinceLastID(t *testing.T) {
	path := filepath.Join(t.TempDir(), "TASK-MESSAGE-BUS.md")
	bus, err := NewMessageBus(path)
	if err != nil {
		t.Fatalf("NewMessageBus: %v", err)
	}
	id1, _ := bus.AppendMessage(&Message{Type: "FACT", ProjectID: "project", Body: "one"})
	_, _ = bus.AppendMessage(&Message{Type: "FACT", ProjectID: "project", Body: "two"})
	messages, err := bus.ReadMessages(id1)
	if err != nil {
		t.Fatalf("ReadMessages: %v", err)
	}
	if len(messages) != 1 {
		t.Fatalf("expected one message after since id")
	}
	lastID := messages[0].MsgID
	messages, err = bus.ReadMessages(lastID)
	if err != nil {
		t.Fatalf("ReadMessages: %v", err)
	}
	if len(messages) != 0 {
		t.Fatalf("expected no messages after last id")
	}
}

func TestValidateBusPathDirectory(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "busdir")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}
	if err := validateBusPath(dir); err == nil {
		t.Fatalf("expected error for directory path")
	}
}

type shortWriter struct{}

func (w *shortWriter) Write(p []byte) (int, error) { return 0, nil }

type errorWriter struct{}

func (w *errorWriter) Write(p []byte) (int, error) { return 0, errors.New("write failed") }
