package agent

import (
	"bytes"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestSpawnProcessWithOptionsEmptyCommand(t *testing.T) {
	if _, err := SpawnProcessWithOptions("", nil, nil, nil, nil, ProcessOptions{}); err == nil {
		t.Fatalf("expected error for empty command")
	}
}

func TestSpawnProcess(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell-based test unsupported on windows")
	}
	var stdout bytes.Buffer
	cmd, err := SpawnProcess("sh", []string{"-c", "echo hello"}, nil, &stdout, &stdout)
	if err != nil {
		t.Fatalf("SpawnProcess: %v", err)
	}
	if err := cmd.Wait(); err != nil {
		t.Fatalf("wait: %v", err)
	}
	if strings.TrimSpace(stdout.String()) != "hello" {
		t.Fatalf("unexpected stdout: %q", stdout.String())
	}
}

func TestSpawnProcessWithOptionsWorkingDir(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("shell-based test unsupported on windows")
	}
	workDir := t.TempDir()
	var stdout bytes.Buffer
	cmd, err := SpawnProcessWithOptions("sh", []string{"-c", "pwd"}, nil, &stdout, &stdout, ProcessOptions{Dir: workDir})
	if err != nil {
		t.Fatalf("SpawnProcessWithOptions: %v", err)
	}
	if err := cmd.Wait(); err != nil {
		t.Fatalf("wait: %v", err)
	}
	got := strings.TrimSpace(stdout.String())
	if got != workDir {
		t.Fatalf("expected working dir %q, got %q", workDir, got)
	}
}

func TestCaptureOutputValidation(t *testing.T) {
	if _, err := CaptureOutput(nil, nil, OutputFiles{}); err == nil {
		t.Fatalf("expected error for empty paths")
	}
}

func TestOutputCaptureCloseNil(t *testing.T) {
	var capture *OutputCapture
	if err := capture.Close(); err == nil {
		t.Fatalf("expected error for nil capture")
	}
}

func TestCaptureOutputSuccess(t *testing.T) {
	dir := t.TempDir()
	stdoutPath := filepath.Join(dir, "stdout.txt")
	stderrPath := filepath.Join(dir, "stderr.txt")
	capture, err := CaptureOutput(nil, nil, OutputFiles{StdoutPath: stdoutPath, StderrPath: stderrPath})
	if err != nil {
		t.Fatalf("CaptureOutput: %v", err)
	}
	if _, err := capture.Stdout.Write([]byte("hello\n")); err != nil {
		t.Fatalf("write stdout: %v", err)
	}
	if err := capture.Close(); err != nil {
		t.Fatalf("close: %v", err)
	}
	data, err := os.ReadFile(stdoutPath)
	if err != nil {
		t.Fatalf("read stdout: %v", err)
	}
	if !strings.Contains(string(data), "hello") {
		t.Fatalf("unexpected stdout: %q", string(data))
	}
}

func TestOpenOutputFileEmpty(t *testing.T) {
	if _, err := openOutputFile(""); err == nil {
		t.Fatalf("expected error for empty output path")
	}
}

func TestOpenOutputFileDirError(t *testing.T) {
	root := t.TempDir()
	blocker := filepath.Join(root, "block")
	if err := os.WriteFile(blocker, []byte("x"), 0o644); err != nil {
		t.Fatalf("write blocker: %v", err)
	}
	if _, err := openOutputFile(filepath.Join(blocker, "out.txt")); err == nil {
		t.Fatalf("expected error for bad output dir")
	}
}

func TestCaptureOutputEmptyStderr(t *testing.T) {
	dir := t.TempDir()
	if _, err := CaptureOutput(nil, nil, OutputFiles{StdoutPath: filepath.Join(dir, "stdout.txt")}); err == nil {
		t.Fatalf("expected error for empty stderr path")
	}
}

func TestCreateOutputMDNotDir(t *testing.T) {
	file := filepath.Join(t.TempDir(), "file.txt")
	if err := os.WriteFile(file, []byte("x"), 0o644); err != nil {
		t.Fatalf("write file: %v", err)
	}
	if _, err := CreateOutputMD(file, ""); err == nil {
		t.Fatalf("expected error for non-dir run path")
	}
}

func TestCreateOutputMDErrors(t *testing.T) {
	if _, err := CreateOutputMD("", ""); err == nil {
		t.Fatalf("expected error for empty run dir")
	}
	runDir := t.TempDir()
	outputPath := filepath.Join(runDir, "output.md")
	if err := os.MkdirAll(outputPath, 0o755); err != nil {
		t.Fatalf("mkdir output.md: %v", err)
	}
	if _, err := CreateOutputMD(runDir, ""); err == nil {
		t.Fatalf("expected error when output.md is a directory")
	}
}

func TestCreateOutputMDFallback(t *testing.T) {
	runDir := t.TempDir()
	fallback := filepath.Join(runDir, "agent-stdout.txt")
	if err := os.WriteFile(fallback, []byte("hello"), 0o644); err != nil {
		t.Fatalf("write fallback: %v", err)
	}
	path, err := CreateOutputMD(runDir, "")
	if err != nil {
		t.Fatalf("CreateOutputMD: %v", err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read output: %v", err)
	}
	if string(data) != "hello" {
		t.Fatalf("unexpected output content: %q", string(data))
	}
}

func TestCreateOutputMDExisting(t *testing.T) {
	runDir := t.TempDir()
	outputPath := filepath.Join(runDir, "output.md")
	if err := os.WriteFile(outputPath, []byte("existing"), 0o644); err != nil {
		t.Fatalf("write output: %v", err)
	}
	path, err := CreateOutputMD(runDir, "")
	if err != nil {
		t.Fatalf("CreateOutputMD: %v", err)
	}
	if path != outputPath {
		t.Fatalf("unexpected output path: %q", path)
	}
}

func TestCreateOutputMDFallbackRelative(t *testing.T) {
	runDir := t.TempDir()
	fallback := "custom.txt"
	if err := os.WriteFile(filepath.Join(runDir, fallback), []byte("custom"), 0o644); err != nil {
		t.Fatalf("write fallback: %v", err)
	}
	path, err := CreateOutputMD(runDir, fallback)
	if err != nil {
		t.Fatalf("CreateOutputMD: %v", err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read output: %v", err)
	}
	if string(data) != "custom" {
		t.Fatalf("unexpected output content: %q", string(data))
	}
}

func TestCreateOutputMDFallbackAbsolute(t *testing.T) {
	runDir := t.TempDir()
	fallback := filepath.Join(t.TempDir(), "abs.txt")
	if err := os.WriteFile(fallback, []byte("abs"), 0o644); err != nil {
		t.Fatalf("write fallback: %v", err)
	}
	path, err := CreateOutputMD(runDir, fallback)
	if err != nil {
		t.Fatalf("CreateOutputMD: %v", err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read output: %v", err)
	}
	if string(data) != "abs" {
		t.Fatalf("unexpected output content: %q", string(data))
	}
}

func TestCaptureOutputStderrOpenError(t *testing.T) {
	dir := t.TempDir()
	stdoutPath := filepath.Join(dir, "stdout.txt")
	blocker := filepath.Join(dir, "blocker")
	if err := os.WriteFile(blocker, []byte("x"), 0o644); err != nil {
		t.Fatalf("write blocker: %v", err)
	}
	stderrPath := filepath.Join(blocker, "stderr.txt")
	if _, err := CaptureOutput(nil, nil, OutputFiles{StdoutPath: stdoutPath, StderrPath: stderrPath}); err == nil {
		t.Fatalf("expected error for stderr open failure")
	}
}

func TestCreateOutputMDFallbackMissing(t *testing.T) {
	runDir := t.TempDir()
	if _, err := CreateOutputMD(runDir, "missing.txt"); err == nil {
		t.Fatalf("expected error for missing fallback")
	}
}

func TestCreateOutputMDCannotCreateOutput(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("chmod based test unsupported on windows")
	}
	runDir := t.TempDir()
	fallback := filepath.Join(runDir, "agent-stdout.txt")
	if err := os.WriteFile(fallback, []byte("output"), 0o644); err != nil {
		t.Fatalf("write fallback: %v", err)
	}
	if err := os.Chmod(runDir, 0o500); err != nil {
		t.Fatalf("chmod run dir: %v", err)
	}
	defer func() {
		_ = os.Chmod(runDir, 0o755)
	}()
	if _, err := CreateOutputMD(runDir, ""); err == nil {
		t.Fatalf("expected error for unwritable output dir")
	}
}
