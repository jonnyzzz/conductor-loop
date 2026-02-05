package api

import (
	"context"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestOffsetForLineNegative(t *testing.T) {
	path := filepath.Join(t.TempDir(), "log.txt")
	data := []byte("a\n" + "b\n")
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatalf("write log: %v", err)
	}
	offset, err := offsetForLine(path, -1)
	if err != nil {
		t.Fatalf("offsetForLine: %v", err)
	}
	if offset != int64(len(data)) {
		t.Fatalf("expected offset %d, got %d", len(data), offset)
	}
}

func TestTailerPollReadsLines(t *testing.T) {
	path := filepath.Join(t.TempDir(), "log.txt")
	if err := os.WriteFile(path, []byte(""), 0o644); err != nil {
		t.Fatalf("write log: %v", err)
	}
	events := make(chan LogLine, 2)
	tailer, err := NewTailer(path, "run-1", "stdout", 10*time.Millisecond, 0, events)
	if err != nil {
		t.Fatalf("NewTailer: %v", err)
	}
	if err := os.WriteFile(path, []byte("line1\nline2\n"), 0o644); err != nil {
		t.Fatalf("write log: %v", err)
	}
	if err := tailer.poll(); err != nil {
		t.Fatalf("poll: %v", err)
	}
	if len(events) != 2 {
		t.Fatalf("expected 2 events, got %d", len(events))
	}
}

func TestTailerStartStop(t *testing.T) {
	path := filepath.Join(t.TempDir(), "log.txt")
	if err := os.WriteFile(path, []byte("line1\n"), 0o644); err != nil {
		t.Fatalf("write log: %v", err)
	}
	events := make(chan LogLine, 1)
	tailer, err := NewTailer(path, "run-1", "stdout", 5*time.Millisecond, 0, events)
	if err != nil {
		t.Fatalf("NewTailer: %v", err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	tailer.Start(ctx)
	tailer.Stop()
	tailer.Stop()
	cancel()
}
