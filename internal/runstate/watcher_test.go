package runstate

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// waitChange waits for a signal on the DirWatcher.Changes() channel within a
// timeout. Returns true if a signal was received, false on timeout.
func waitChange(t *testing.T, dw *DirWatcher, timeout time.Duration) bool {
	t.Helper()
	select {
	case _, ok := <-dw.Changes():
		return ok
	case <-time.After(timeout):
		return false
	}
}

// drainChanges consumes any pending signals without blocking.
func drainChanges(dw *DirWatcher) {
	for {
		select {
		case <-dw.Changes():
		default:
			return
		}
	}
}

// TestDirWatcher_DetectsFileWrite verifies that modifying a file inside a
// watched directory triggers a change notification.
func TestDirWatcher_DetectsFileWrite(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "watched.txt")
	if err := os.WriteFile(path, []byte("initial"), 0o644); err != nil {
		t.Fatalf("write initial file: %v", err)
	}

	dw, err := NewDirWatcher(dir)
	if err != nil {
		t.Fatalf("NewDirWatcher: %v", err)
	}
	defer dw.Close()

	// Let the watcher settle before triggering a change.
	time.Sleep(50 * time.Millisecond)
	drainChanges(dw)

	if err := os.WriteFile(path, []byte("updated"), 0o644); err != nil {
		t.Fatalf("write updated file: %v", err)
	}

	if !waitChange(t, dw, 3*time.Second) {
		t.Fatal("timeout: expected change notification after file write")
	}
}

// TestDirWatcher_AutoWatchesNewSubdirs verifies that a subdirectory created
// inside a watched directory is automatically added to the watch set so that
// subsequent writes inside it also trigger notifications.
func TestDirWatcher_AutoWatchesNewSubdirs(t *testing.T) {
	dir := t.TempDir()
	dw, err := NewDirWatcher(dir)
	if err != nil {
		t.Fatalf("NewDirWatcher: %v", err)
	}
	defer dw.Close()

	time.Sleep(50 * time.Millisecond)
	drainChanges(dw)

	// Create a subdirectory — this fires a Create event and the watcher
	// auto-adds it.
	subdir := filepath.Join(dir, "sub")
	if err := os.Mkdir(subdir, 0o755); err != nil {
		t.Fatalf("mkdir subdir: %v", err)
	}

	// Drain the Create signal for the subdirectory itself.
	if !waitChange(t, dw, 3*time.Second) {
		t.Fatal("timeout: expected change notification after subdir creation")
	}
	drainChanges(dw)

	// Give fsnotify time to add the new directory.
	time.Sleep(50 * time.Millisecond)

	// Write a file inside the subdirectory; the watcher should pick it up.
	if err := os.WriteFile(filepath.Join(subdir, "file.txt"), []byte("hello"), 0o644); err != nil {
		t.Fatalf("write file in subdir: %v", err)
	}

	if !waitChange(t, dw, 3*time.Second) {
		t.Fatal("timeout: expected change notification for file in auto-watched subdir")
	}
}

// TestDirWatcher_CloseStopsNotifications verifies that after Close the changes
// channel is closed and no further notifications are delivered.
func TestDirWatcher_CloseStopsNotifications(t *testing.T) {
	dir := t.TempDir()
	dw, err := NewDirWatcher(dir)
	if err != nil {
		t.Fatalf("NewDirWatcher: %v", err)
	}

	dw.Close()

	// After Close the channel must be closed (read returns zero value, ok=false).
	select {
	case _, ok := <-dw.Changes():
		if ok {
			t.Error("expected changes channel to be closed after Close()")
		}
	case <-time.After(2 * time.Second):
		t.Error("timeout: changes channel was not closed after Close()")
	}
}

// TestDirWatcher_MultipleCloseIsSafe verifies Close is idempotent.
func TestDirWatcher_MultipleCloseIsSafe(t *testing.T) {
	dir := t.TempDir()
	dw, err := NewDirWatcher(dir)
	if err != nil {
		t.Fatalf("NewDirWatcher: %v", err)
	}
	dw.Close()
	dw.Close() // must not panic
}

// TestFileReader_ReadAll verifies basic read-all functionality.
func TestFileReader_ReadAll(t *testing.T) {
	path := filepath.Join(t.TempDir(), "test.txt")
	if err := os.WriteFile(path, []byte("hello"), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	fr, err := NewFileReader(path)
	if err != nil {
		t.Fatalf("NewFileReader: %v", err)
	}
	defer fr.Close()

	data, err := fr.ReadAll()
	if err != nil {
		t.Fatalf("ReadAll: %v", err)
	}
	if string(data) != "hello" {
		t.Errorf("ReadAll = %q, want %q", string(data), "hello")
	}
}

// TestFileReader_ReadAllAfterContentUpdate verifies that ReadAll returns the
// new content after the file is overwritten in place (truncate + write).
func TestFileReader_ReadAllAfterContentUpdate(t *testing.T) {
	path := filepath.Join(t.TempDir(), "test.txt")
	if err := os.WriteFile(path, []byte("initial"), 0o644); err != nil {
		t.Fatalf("write initial: %v", err)
	}

	fr, err := NewFileReader(path)
	if err != nil {
		t.Fatalf("NewFileReader: %v", err)
	}
	defer fr.Close()

	// First read.
	data, err := fr.ReadAll()
	if err != nil {
		t.Fatalf("ReadAll (first): %v", err)
	}
	if string(data) != "initial" {
		t.Errorf("first ReadAll = %q, want %q", string(data), "initial")
	}

	// Overwrite file content in place.
	if err := os.WriteFile(path, []byte("updated"), 0o644); err != nil {
		t.Fatalf("overwrite file: %v", err)
	}

	// Second read should seek to start and return updated content.
	data, err = fr.ReadAll()
	if err != nil {
		t.Fatalf("ReadAll (second): %v", err)
	}
	if string(data) != "updated" {
		t.Errorf("second ReadAll = %q, want %q", string(data), "updated")
	}
}

// TestFileReader_ReadAllAfterAtomicReplace verifies that ReadAll transparently
// re-opens the file after it is replaced via atomic rename.
func TestFileReader_ReadAllAfterAtomicReplace(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "test.txt")
	if err := os.WriteFile(path, []byte("original"), 0o644); err != nil {
		t.Fatalf("write original: %v", err)
	}

	fr, err := NewFileReader(path)
	if err != nil {
		t.Fatalf("NewFileReader: %v", err)
	}
	defer fr.Close()

	// First read.
	data, err := fr.ReadAll()
	if err != nil {
		t.Fatalf("ReadAll (first): %v", err)
	}
	if string(data) != "original" {
		t.Errorf("first ReadAll = %q, want %q", string(data), "original")
	}

	// Atomic replace: write to temp file then rename.
	tmp := filepath.Join(dir, "test.tmp")
	if err := os.WriteFile(tmp, []byte("replaced"), 0o644); err != nil {
		t.Fatalf("write tmp: %v", err)
	}
	if err := os.Rename(tmp, path); err != nil {
		t.Fatalf("rename: %v", err)
	}

	// ReadAll should re-open and return the replaced content.
	data, err = fr.ReadAll()
	if err != nil {
		t.Fatalf("ReadAll after rename: %v", err)
	}
	if string(data) != "replaced" {
		t.Errorf("ReadAll after rename = %q, want %q", string(data), "replaced")
	}
}

// TestFileReader_CloseReleasesHandle verifies that Close is safe to call.
func TestFileReader_CloseReleasesHandle(t *testing.T) {
	path := filepath.Join(t.TempDir(), "test.txt")
	if err := os.WriteFile(path, []byte("data"), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}

	fr, err := NewFileReader(path)
	if err != nil {
		t.Fatalf("NewFileReader: %v", err)
	}
	if err := fr.Close(); err != nil {
		t.Errorf("Close: %v", err)
	}
}

// TestNewFileReader_MissingFileReturnsError verifies that NewFileReader fails
// gracefully when the target file does not exist.
func TestNewFileReader_MissingFileReturnsError(t *testing.T) {
	_, err := NewFileReader("/nonexistent/path/to/file.txt")
	if err == nil {
		t.Fatal("expected error for non-existent file")
	}
}
