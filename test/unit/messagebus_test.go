package unit_test

import (
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"testing"
	"time"

	"github.com/jonnyzzz/conductor-loop/internal/messagebus"
)

func TestMsgIDUniqueness(t *testing.T) {
	const total = 10000
	seen := make(map[string]struct{}, total)
	for i := 0; i < total; i++ {
		id := messagebus.GenerateMessageID()
		if _, exists := seen[id]; exists {
			t.Fatalf("duplicate message id: %s", id)
		}
		seen[id] = struct{}{}
	}
}

func TestLockTimeout(t *testing.T) {
	path := filepath.Join(t.TempDir(), "bus.lock")
	file1, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0o644)
	if err != nil {
		t.Fatalf("open file1: %v", err)
	}
	defer file1.Close()
	file2, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0o644)
	if err != nil {
		t.Fatalf("open file2: %v", err)
	}
	defer file2.Close()

	if err := messagebus.LockExclusive(file1, 100*time.Millisecond); err != nil {
		t.Fatalf("lock file1: %v", err)
	}
	defer func() {
		_ = messagebus.Unlock(file1)
	}()

	if err := messagebus.LockExclusive(file2, 30*time.Millisecond); err == nil {
		_ = messagebus.Unlock(file2)
		t.Fatalf("expected lock timeout")
	} else if err != messagebus.ErrLockTimeout {
		t.Fatalf("expected ErrLockTimeout, got %v", err)
	}
}

func TestConcurrentAppend(t *testing.T) {
	path := filepath.Join(t.TempDir(), "TASK-MESSAGE-BUS.md")
	bus, err := messagebus.NewMessageBus(path)
	if err != nil {
		t.Fatalf("NewMessageBus: %v", err)
	}

	const (
		writers    = 10
		iterations = 100
	)

	errCh := make(chan error, writers*iterations)
	var wg sync.WaitGroup
	for w := 0; w < writers; w++ {
		w := w
		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < iterations; i++ {
				msg := &messagebus.Message{
					Type:      "FACT",
					ProjectID: "project",
					TaskID:    "task",
					RunID:     fmt.Sprintf("run-%d", w),
					Body:      fmt.Sprintf("msg %d/%d", w, i),
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
			t.Fatalf("append error: %v", err)
		}
	}

	messages, err := bus.ReadMessages("")
	if err != nil {
		t.Fatalf("ReadMessages: %v", err)
	}
	if len(messages) != writers*iterations {
		t.Fatalf("expected %d messages, got %d", writers*iterations, len(messages))
	}
}
