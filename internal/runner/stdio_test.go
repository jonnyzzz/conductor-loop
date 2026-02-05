package runner

import (
	"os"
	"path/filepath"
	"testing"
)

func TestOpenStdioValidation(t *testing.T) {
	if _, err := OpenStdio("", nil, nil); err == nil {
		t.Fatalf("expected error for empty run dir")
	}
}

func TestOpenAppendFileErrors(t *testing.T) {
	if _, err := openAppendFile(""); err == nil {
		t.Fatalf("expected error for empty output path")
	}
	root := t.TempDir()
	blocker := filepath.Join(root, "blocker")
	if err := os.WriteFile(blocker, []byte("x"), 0o644); err != nil {
		t.Fatalf("write blocker: %v", err)
	}
	if _, err := openAppendFile(filepath.Join(blocker, "out.txt")); err == nil {
		t.Fatalf("expected error for invalid output dir")
	}
}

func TestStdioCaptureCloseNil(t *testing.T) {
	var capture *StdioCapture
	if err := capture.Close(); err == nil {
		t.Fatalf("expected error for nil stdio capture")
	}
}
