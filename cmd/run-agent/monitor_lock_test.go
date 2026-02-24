package main

import (
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"
)

// ---- parsePIDFromLock ----

func TestParsePIDFromLock_Valid(t *testing.T) {
	pid, err := parsePIDFromLock("12345\n")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if pid != 12345 {
		t.Errorf("expected 12345, got %d", pid)
	}
}

func TestParsePIDFromLock_WithWhitespace(t *testing.T) {
	pid, err := parsePIDFromLock("  99  \n")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if pid != 99 {
		t.Errorf("expected 99, got %d", pid)
	}
}

func TestParsePIDFromLock_Invalid(t *testing.T) {
	_, err := parsePIDFromLock("not-a-pid")
	if err == nil {
		t.Fatal("expected error for non-numeric content")
	}
}

func TestParsePIDFromLock_Empty(t *testing.T) {
	_, err := parsePIDFromLock("")
	if err == nil {
		t.Fatal("expected error for empty content")
	}
}

// ---- monitorLockPath ----

func TestMonitorLockPath(t *testing.T) {
	path := monitorLockPath("/some/root", "myproj")
	if filepath.Base(path) != ".monitor.pid" {
		t.Errorf("unexpected lock filename: %q", filepath.Base(path))
	}
	if !filepath.IsAbs(path) {
		t.Errorf("expected absolute path, got %q", path)
	}
}

// ---- acquireMonitorLock: basic acquire + release ----

func TestAcquireMonitorLock_AcquireAndRelease(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "proj"), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	release, err := acquireMonitorLock(root, "proj")
	if err != nil {
		t.Fatalf("expected no error acquiring lock, got: %v", err)
	}

	// Lockfile must exist while held.
	lockPath := monitorLockPath(root, "proj")
	data, readErr := os.ReadFile(lockPath)
	if readErr != nil {
		t.Fatalf("expected lockfile to exist: %v", readErr)
	}
	pid, _ := parsePIDFromLock(string(data))
	if pid != os.Getpid() {
		t.Errorf("expected PID %d in lockfile, got %d", os.Getpid(), pid)
	}

	// Release must remove the lockfile.
	release()
	if _, err := os.Stat(lockPath); !os.IsNotExist(err) {
		t.Error("expected lockfile to be removed after release")
	}
}

// ---- acquireMonitorLock: duplicate-start rejection ----

func TestAcquireMonitorLock_DuplicateStartRejected(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "proj"), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	// Write our own PID as if we are an existing live monitor.
	lockPath := monitorLockPath(root, "proj")
	ownPID := os.Getpid()
	if err := os.WriteFile(lockPath, []byte(strconv.Itoa(ownPID)+"\n"), 0o644); err != nil {
		t.Fatalf("write lockfile: %v", err)
	}

	// A second acquire attempt must be rejected because the process (us) is alive.
	_, err := acquireMonitorLock(root, "proj")
	if err == nil {
		t.Fatal("expected error when live monitor already holds the lock")
	}
}

// ---- acquireMonitorLock: stale-lock recovery ----

func TestAcquireMonitorLock_StaleLockRecovered(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "proj"), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	// Write a stale PID (PID 1 is init/launchd and cannot be killed by us,
	// but we need a truly dead PID. Use a PID that should not be alive.
	// We use PID 2 as a near-zero PID unlikely to be a conductor process.)
	// Actually, the safest approach: use a definitely-dead PID by inspecting
	// a process we start and wait for.
	lockPath := monitorLockPath(root, "proj")
	// PID 0 is not a valid process PID in any OS; IsProcessAlive(0) returns false.
	if err := os.WriteFile(lockPath, []byte("0\n"), 0o644); err != nil {
		t.Fatalf("write stale lockfile: %v", err)
	}

	// acquireMonitorLock should auto-clean the stale lock and succeed.
	release, err := acquireMonitorLock(root, "proj")
	if err != nil {
		t.Fatalf("expected stale lock to be recovered, got error: %v", err)
	}
	defer release()

	// Verify lockfile now contains our PID.
	data, _ := os.ReadFile(lockPath)
	pid, _ := parsePIDFromLock(string(data))
	if pid != os.Getpid() {
		t.Errorf("expected own PID %d in lockfile after recovery, got %d", os.Getpid(), pid)
	}
}

// ---- acquireMonitorLock: corrupt lock recovery ----

func TestAcquireMonitorLock_CorruptLockRecovered(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "proj"), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	lockPath := monitorLockPath(root, "proj")
	if err := os.WriteFile(lockPath, []byte("not-a-pid\n"), 0o644); err != nil {
		t.Fatalf("write corrupt lockfile: %v", err)
	}

	release, err := acquireMonitorLock(root, "proj")
	if err != nil {
		t.Fatalf("expected corrupt lock to be recovered, got error: %v", err)
	}
	defer release()
}

// ---- acquireMonitorLock: distinct scopes coexist ----

func TestAcquireMonitorLock_DistinctScopesCoexist(t *testing.T) {
	root := t.TempDir()
	for _, proj := range []string{"proj-alpha", "proj-beta"} {
		if err := os.MkdirAll(filepath.Join(root, proj), 0o755); err != nil {
			t.Fatalf("mkdir %s: %v", proj, err)
		}
	}

	release1, err := acquireMonitorLock(root, "proj-alpha")
	if err != nil {
		t.Fatalf("acquire lock for proj-alpha: %v", err)
	}
	defer release1()

	release2, err := acquireMonitorLock(root, "proj-beta")
	if err != nil {
		t.Fatalf("acquire lock for proj-beta should succeed independently, got: %v", err)
	}
	defer release2()

	// Both lockfiles must exist.
	for _, proj := range []string{"proj-alpha", "proj-beta"} {
		lockPath := monitorLockPath(root, proj)
		if _, err := os.Stat(lockPath); err != nil {
			t.Errorf("expected lockfile for %s to exist: %v", proj, err)
		}
	}
}

// ---- runMonitor daemon: lock held, second start fails ----

func TestRunMonitor_DaemonLockPreventsSecondStart(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "proj"), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	// Simulate an existing monitor by writing our own PID as a live lock.
	lockPath := monitorLockPath(root, "proj")
	ownPID := os.Getpid()
	if err := os.WriteFile(lockPath, []byte(strconv.Itoa(ownPID)+"\n"), 0o644); err != nil {
		t.Fatalf("write lockfile: %v", err)
	}
	defer os.Remove(lockPath)

	dir := t.TempDir()
	todoPath := writeTODO(t, dir, "# empty\n")

	opts := monitorOpts{
		RootDir:   root,
		ProjectID: "proj",
		TODOFile:  todoPath,
		Interval:  10 * time.Second,
		Once:      false, // daemon mode — lock is enforced
	}

	// runMonitor must fail immediately with a lock error.
	err := runMonitor(os.Stdout, opts)
	if err == nil {
		t.Fatal("expected error when lock is held by another (live) monitor")
	}
}

func TestRunMonitor_OnceSkipsLockEnforcement(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "proj"), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	// Simulate an existing monitor PID lock held by "us" (a live process).
	lockPath := monitorLockPath(root, "proj")
	ownPID := os.Getpid()
	if err := os.WriteFile(lockPath, []byte(strconv.Itoa(ownPID)+"\n"), 0o644); err != nil {
		t.Fatalf("write lockfile: %v", err)
	}
	defer os.Remove(lockPath)

	dir := t.TempDir()
	todoPath := writeTODO(t, dir, "# empty\n")

	opts := monitorOpts{
		RootDir:   root,
		ProjectID: "proj",
		TODOFile:  todoPath,
		Once:      true, // single-pass mode skips daemon lock
	}

	// Must succeed even with a lock present (--once is exempt).
	if err := runMonitor(os.Stdout, opts); err != nil {
		t.Fatalf("expected --once to bypass lock enforcement, got: %v", err)
	}
}
