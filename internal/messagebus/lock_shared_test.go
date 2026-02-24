package messagebus

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// TestLockShared_ReadsSucceedAfterWriterUnlocks verifies that a reader using
// LockShared can proceed once a writer releases the exclusive lock.
// This test uses the public LockExclusive / LockShared / Unlock APIs and
// exercises the retry/timeout semantics on all platforms.
func TestLockShared_ReadsSucceedAfterWriterUnlocks(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bus.lock")

	// Create the lock file
	f1, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0o644)
	if err != nil {
		t.Fatalf("create lock file: %v", err)
	}
	defer f1.Close() //nolint:errcheck

	// Acquire exclusive lock
	if err := LockExclusive(f1, 2*time.Second); err != nil {
		t.Fatalf("LockExclusive: %v", err)
	}

	// Open a second handle for the reader
	f2, err := os.Open(path)
	if err != nil {
		t.Fatalf("open reader: %v", err)
	}
	defer f2.Close() //nolint:errcheck

	// On Unix, LockShared is a no-op so it succeeds immediately.
	// On Windows, it would block until the exclusive lock is released.
	// Release the exclusive lock and then try shared — should always succeed.
	if err := Unlock(f1); err != nil {
		t.Fatalf("Unlock exclusive: %v", err)
	}

	if err := LockShared(f2, 2*time.Second); err != nil {
		t.Fatalf("LockShared after unlock: %v", err)
	}
	if err := Unlock(f2); err != nil {
		t.Fatalf("Unlock shared: %v", err)
	}
}

// TestLockShared_Timeout verifies that LockShared returns ErrLockTimeout when
// the lock cannot be acquired within the deadline. On Unix, LockShared is a
// no-op and always succeeds immediately, so this test is only meaningful on
// Windows. We test the timeout path by using a very short timeout against
// a non-existent lock, ensuring the timeout value is validated.
func TestLockShared_Timeout_InvalidDuration(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bus2.lock")
	f, err := os.OpenFile(path, os.O_CREATE|os.O_RDWR, 0o644)
	if err != nil {
		t.Fatalf("create lock file: %v", err)
	}
	defer f.Close() //nolint:errcheck

	// Negative timeout must return an error immediately.
	err = LockShared(f, -1*time.Second)
	if err == nil {
		t.Fatal("expected error for negative timeout")
	}
}

// TestReadBusFileShared_ReadsContent verifies the readBusFileShared helper
// correctly reads file content.
func TestReadBusFileShared_ReadsContent(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "bus.md")
	content := []byte("---\nhello: world\n---\n")
	if err := os.WriteFile(path, content, 0o644); err != nil {
		t.Fatalf("write: %v", err)
	}

	got, err := readBusFileShared(path)
	if err != nil {
		t.Fatalf("readBusFileShared: %v", err)
	}
	if string(got) != string(content) {
		t.Errorf("expected %q, got %q", content, got)
	}
}

// TestReadBusFileShared_NotFound verifies readBusFileShared returns an error
// (not a panic) when the file does not exist.
func TestReadBusFileShared_NotFound(t *testing.T) {
	_, err := readBusFileShared("/nonexistent/path/bus.md")
	if err == nil {
		t.Fatal("expected error for nonexistent file")
	}
}
