package agent

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestDetectCLIVersionEmptyCommand(t *testing.T) {
	if _, err := DetectCLIVersion(context.Background(), ""); err == nil {
		t.Fatalf("expected error for empty command")
	}
}

func TestDetectCLIVersionMissing(t *testing.T) {
	if _, err := DetectCLIVersion(context.Background(), "nonexistent-binary-12345"); err == nil {
		t.Fatalf("expected error for missing binary")
	}
}

func TestDetectCLIVersionNilContext(t *testing.T) {
	if _, err := DetectCLIVersion(nil, "nonexistent-binary-12345"); err == nil {
		t.Fatalf("expected error for missing binary with nil context")
	}
}

func TestDetectCLIVersionFakeCommand(t *testing.T) {
	dir := t.TempDir()
	createVersionCLI(t, dir, "fake-agent", "fake-agent 1.2.3")
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))

	version, err := DetectCLIVersion(context.Background(), "fake-agent")
	if err != nil {
		t.Fatalf("DetectCLIVersion: %v", err)
	}
	if !strings.Contains(version, "1.2.3") {
		t.Fatalf("expected version to contain 1.2.3, got %q", version)
	}
}

func TestDetectCLIVersionCanceled(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	_, err := DetectCLIVersion(ctx, "sh")
	if err == nil {
		t.Fatalf("expected error for canceled context")
	}
}

func createVersionCLI(t *testing.T, dir, name, output string) {
	t.Helper()
	if runtime.GOOS == "windows" {
		path := filepath.Join(dir, name+".bat")
		content := "@echo off\r\necho " + output + "\r\n"
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatalf("write bat: %v", err)
		}
		return
	}
	path := filepath.Join(dir, name)
	content := "#!/bin/sh\necho '" + output + "'\n"
	if err := os.WriteFile(path, []byte(content), 0o755); err != nil {
		t.Fatalf("write script: %v", err)
	}
}
