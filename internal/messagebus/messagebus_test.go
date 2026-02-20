package messagebus

import (
	"errors"
	"os"
	"path/filepath"
	"runtime"
	"sync"
	"sync/atomic"
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

func TestAppendRetryOnLockContention(t *testing.T) {
	path := filepath.Join(t.TempDir(), "TASK-MESSAGE-BUS.md")
	bus, err := NewMessageBus(path,
		WithLockTimeout(50*time.Millisecond),
		WithMaxRetries(3),
		WithRetryBackoff(10*time.Millisecond),
	)
	if err != nil {
		t.Fatalf("NewMessageBus: %v", err)
	}

	// Hold an exclusive lock on the file, then release after a short delay.
	lockFile, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE, messageBusFileMode)
	if err != nil {
		t.Fatalf("open lock file: %v", err)
	}
	if err := LockExclusive(lockFile, time.Second); err != nil {
		lockFile.Close()
		t.Fatalf("acquire lock: %v", err)
	}

	var released int32
	go func() {
		// Release after enough time for first attempt to fail but second to succeed.
		time.Sleep(80 * time.Millisecond)
		_ = Unlock(lockFile)
		lockFile.Close()
		atomic.StoreInt32(&released, 1)
	}()

	msgID, err := bus.AppendMessage(&Message{Type: "FACT", ProjectID: "project", Body: "retry-test"})
	if err != nil {
		t.Fatalf("AppendMessage should have succeeded after retry: %v", err)
	}
	if msgID == "" {
		t.Fatalf("expected non-empty message id")
	}

	// Verify the message was written.
	messages, err := bus.ReadMessages("")
	if err != nil {
		t.Fatalf("ReadMessages: %v", err)
	}
	if len(messages) != 1 || messages[0].Body != "retry-test" {
		t.Fatalf("expected message with body 'retry-test', got %v", messages)
	}

	// Verify retries were triggered.
	attempts, retries := bus.ContentionStats()
	if retries == 0 {
		t.Fatalf("expected retries > 0, got attempts=%d retries=%d", attempts, retries)
	}
}

func TestAppendExhaustsRetries(t *testing.T) {
	path := filepath.Join(t.TempDir(), "TASK-MESSAGE-BUS.md")
	bus, err := NewMessageBus(path,
		WithLockTimeout(20*time.Millisecond),
		WithMaxRetries(2),
		WithRetryBackoff(10*time.Millisecond),
	)
	if err != nil {
		t.Fatalf("NewMessageBus: %v", err)
	}

	// Hold lock for the entire duration.
	lockFile, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE, messageBusFileMode)
	if err != nil {
		t.Fatalf("open lock file: %v", err)
	}
	defer func() {
		_ = Unlock(lockFile)
		lockFile.Close()
	}()
	if err := LockExclusive(lockFile, time.Second); err != nil {
		t.Fatalf("acquire lock: %v", err)
	}

	_, err = bus.AppendMessage(&Message{Type: "FACT", ProjectID: "project", Body: "fail"})
	if err == nil {
		t.Fatalf("expected error after exhausting retries")
	}
	if !errors.Is(err, ErrLockTimeout) {
		t.Fatalf("expected ErrLockTimeout in error chain, got: %v", err)
	}

	attempts, retries := bus.ContentionStats()
	if attempts != 2 {
		t.Fatalf("expected 2 attempts, got %d", attempts)
	}
	if retries != 1 {
		t.Fatalf("expected 1 retry, got %d", retries)
	}
}

func TestContentionStats(t *testing.T) {
	path := filepath.Join(t.TempDir(), "TASK-MESSAGE-BUS.md")
	bus, err := NewMessageBus(path)
	if err != nil {
		t.Fatalf("NewMessageBus: %v", err)
	}

	attempts, retries := bus.ContentionStats()
	if attempts != 0 || retries != 0 {
		t.Fatalf("expected zero stats initially, got attempts=%d retries=%d", attempts, retries)
	}

	for i := 0; i < 5; i++ {
		_, err := bus.AppendMessage(&Message{Type: "FACT", ProjectID: "project", Body: "msg"})
		if err != nil {
			t.Fatalf("AppendMessage: %v", err)
		}
	}

	attempts, retries = bus.ContentionStats()
	if attempts != 5 {
		t.Fatalf("expected 5 attempts, got %d", attempts)
	}
	if retries != 0 {
		t.Fatalf("expected 0 retries, got %d", retries)
	}
}

func TestWithMaxRetriesOption(t *testing.T) {
	path := filepath.Join(t.TempDir(), "TASK-MESSAGE-BUS.md")

	// Verify custom max retries is applied.
	bus, err := NewMessageBus(path, WithMaxRetries(5))
	if err != nil {
		t.Fatalf("NewMessageBus: %v", err)
	}
	if bus.maxRetries != 5 {
		t.Fatalf("expected maxRetries=5, got %d", bus.maxRetries)
	}

	// Verify minimum of 1 is enforced.
	bus2, err := NewMessageBus(path, WithMaxRetries(0))
	if err != nil {
		t.Fatalf("NewMessageBus: %v", err)
	}
	if bus2.maxRetries != 1 {
		t.Fatalf("expected maxRetries=1 (minimum), got %d", bus2.maxRetries)
	}

	bus3, err := NewMessageBus(path, WithMaxRetries(-1))
	if err != nil {
		t.Fatalf("NewMessageBus: %v", err)
	}
	if bus3.maxRetries != 1 {
		t.Fatalf("expected maxRetries=1 (minimum), got %d", bus3.maxRetries)
	}
}

func TestWithRetryBackoffOption(t *testing.T) {
	path := filepath.Join(t.TempDir(), "TASK-MESSAGE-BUS.md")
	bus, err := NewMessageBus(path, WithRetryBackoff(500*time.Millisecond))
	if err != nil {
		t.Fatalf("NewMessageBus: %v", err)
	}
	if bus.retryBackoff != 500*time.Millisecond {
		t.Fatalf("expected retryBackoff=500ms, got %v", bus.retryBackoff)
	}
}

func TestWithFsyncFalseDefault(t *testing.T) {
	path := filepath.Join(t.TempDir(), "TASK-MESSAGE-BUS.md")
	bus, err := NewMessageBus(path)
	if err != nil {
		t.Fatalf("NewMessageBus: %v", err)
	}
	if bus.fsync {
		t.Fatalf("expected fsync=false by default")
	}
}

func TestWithFsyncOption(t *testing.T) {
	path := filepath.Join(t.TempDir(), "TASK-MESSAGE-BUS.md")
	bus, err := NewMessageBus(path, WithFsync(true))
	if err != nil {
		t.Fatalf("NewMessageBus: %v", err)
	}
	if !bus.fsync {
		t.Fatalf("expected fsync=true after WithFsync(true)")
	}

	bus2, err := NewMessageBus(path, WithFsync(false))
	if err != nil {
		t.Fatalf("NewMessageBus: %v", err)
	}
	if bus2.fsync {
		t.Fatalf("expected fsync=false after WithFsync(false)")
	}
}

func TestFsyncWritesComplete(t *testing.T) {
	// WithFsync(true) should write and read back correctly.
	// Note: fsync reduces throughput (~200 msg/sec vs 37,000+ without).
	path := filepath.Join(t.TempDir(), "TASK-MESSAGE-BUS.md")
	bus, err := NewMessageBus(path, WithFsync(true))
	if err != nil {
		t.Fatalf("NewMessageBus: %v", err)
	}

	const count = 10
	for i := 0; i < count; i++ {
		_, err := bus.AppendMessage(&Message{
			Type:      "FACT",
			ProjectID: "project",
			Body:      "fsync-msg",
		})
		if err != nil {
			t.Fatalf("AppendMessage %d: %v", i, err)
		}
	}

	messages, err := bus.ReadMessages("")
	if err != nil {
		t.Fatalf("ReadMessages: %v", err)
	}
	if len(messages) != count {
		t.Fatalf("expected %d messages, got %d", count, len(messages))
	}
}

func TestParentsObjectFormRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "TASK-MESSAGE-BUS.md")
	bus, err := NewMessageBus(path)
	if err != nil {
		t.Fatalf("NewMessageBus: %v", err)
	}
	_, err = bus.AppendMessage(&Message{
		Type:      "FACT",
		ProjectID: "project",
		Parents: []Parent{
			{MsgID: "x", Kind: "depends_on"},
		},
		Body: "parent test",
	})
	if err != nil {
		t.Fatalf("AppendMessage: %v", err)
	}
	msgs, err := bus.ReadMessages("")
	if err != nil {
		t.Fatalf("ReadMessages: %v", err)
	}
	if len(msgs) != 1 {
		t.Fatalf("expected 1 message, got %d", len(msgs))
	}
	if len(msgs[0].Parents) != 1 {
		t.Fatalf("expected 1 parent, got %d", len(msgs[0].Parents))
	}
	if msgs[0].Parents[0].MsgID != "x" {
		t.Fatalf("expected parent MsgID %q, got %q", "x", msgs[0].Parents[0].MsgID)
	}
	if msgs[0].Parents[0].Kind != "depends_on" {
		t.Fatalf("expected parent Kind %q, got %q", "depends_on", msgs[0].Parents[0].Kind)
	}
}

func TestParentsBackwardCompat(t *testing.T) {
	// Old format: parents as a YAML string list.
	raw := []byte("---\nmsg_id: msg-001\nts: 2024-01-01T00:00:00Z\ntype: FACT\nproject_id: project\nparents:\n  - msg-001\n  - msg-002\n---\nbody\n")
	msgs, err := parseMessages(raw)
	if err != nil {
		t.Fatalf("parseMessages: %v", err)
	}
	if len(msgs) != 1 {
		t.Fatalf("expected 1 message, got %d", len(msgs))
	}
	if len(msgs[0].Parents) != 2 {
		t.Fatalf("expected 2 parents, got %d", len(msgs[0].Parents))
	}
	if msgs[0].Parents[0].MsgID != "msg-001" {
		t.Fatalf("expected Parents[0].MsgID %q, got %q", "msg-001", msgs[0].Parents[0].MsgID)
	}
	if msgs[0].Parents[1].MsgID != "msg-002" {
		t.Fatalf("expected Parents[1].MsgID %q, got %q", "msg-002", msgs[0].Parents[1].MsgID)
	}
}

func TestIssueIDAutoSet(t *testing.T) {
	path := filepath.Join(t.TempDir(), "TASK-MESSAGE-BUS.md")
	bus, err := NewMessageBus(path)
	if err != nil {
		t.Fatalf("NewMessageBus: %v", err)
	}
	msgID, err := bus.AppendMessage(&Message{
		Type:      "ISSUE",
		ProjectID: "project",
		Body:      "issue body",
	})
	if err != nil {
		t.Fatalf("AppendMessage: %v", err)
	}
	msgs, err := bus.ReadMessages("")
	if err != nil {
		t.Fatalf("ReadMessages: %v", err)
	}
	if len(msgs) != 1 {
		t.Fatalf("expected 1 message, got %d", len(msgs))
	}
	if msgs[0].IssueID != msgID {
		t.Fatalf("expected IssueID %q to equal MsgID %q", msgs[0].IssueID, msgID)
	}
}

func TestMetaRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "TASK-MESSAGE-BUS.md")
	bus, err := NewMessageBus(path)
	if err != nil {
		t.Fatalf("NewMessageBus: %v", err)
	}
	_, err = bus.AppendMessage(&Message{
		Type:      "FACT",
		ProjectID: "project",
		Meta:      map[string]string{"key1": "value1", "key2": "value2"},
		Body:      "meta test",
	})
	if err != nil {
		t.Fatalf("AppendMessage: %v", err)
	}
	msgs, err := bus.ReadMessages("")
	if err != nil {
		t.Fatalf("ReadMessages: %v", err)
	}
	if len(msgs) != 1 {
		t.Fatalf("expected 1 message, got %d", len(msgs))
	}
	if msgs[0].Meta["key1"] != "value1" {
		t.Fatalf("expected Meta[key1]=%q, got %q", "value1", msgs[0].Meta["key1"])
	}
	if msgs[0].Meta["key2"] != "value2" {
		t.Fatalf("expected Meta[key2]=%q, got %q", "value2", msgs[0].Meta["key2"])
	}
}

func TestLinksRoundTrip(t *testing.T) {
	path := filepath.Join(t.TempDir(), "TASK-MESSAGE-BUS.md")
	bus, err := NewMessageBus(path)
	if err != nil {
		t.Fatalf("NewMessageBus: %v", err)
	}
	_, err = bus.AppendMessage(&Message{
		Type:      "FACT",
		ProjectID: "project",
		Links: []Link{
			{URL: "https://example.com", Label: "example", Kind: "reference"},
		},
		Body: "links test",
	})
	if err != nil {
		t.Fatalf("AppendMessage: %v", err)
	}
	msgs, err := bus.ReadMessages("")
	if err != nil {
		t.Fatalf("ReadMessages: %v", err)
	}
	if len(msgs) != 1 {
		t.Fatalf("expected 1 message, got %d", len(msgs))
	}
	if len(msgs[0].Links) != 1 {
		t.Fatalf("expected 1 link, got %d", len(msgs[0].Links))
	}
	if msgs[0].Links[0].URL != "https://example.com" {
		t.Fatalf("expected Link.URL %q, got %q", "https://example.com", msgs[0].Links[0].URL)
	}
	if msgs[0].Links[0].Label != "example" {
		t.Fatalf("expected Link.Label %q, got %q", "example", msgs[0].Links[0].Label)
	}
	if msgs[0].Links[0].Kind != "reference" {
		t.Fatalf("expected Link.Kind %q, got %q", "reference", msgs[0].Links[0].Kind)
	}
}

type shortWriter struct{}

func (w *shortWriter) Write(p []byte) (int, error) { return 0, nil }

type errorWriter struct{}

func (w *errorWriter) Write(p []byte) (int, error) { return 0, errors.New("write failed") }
