package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNewRootCmd(t *testing.T) {
	cmd := newRootCmd()
	if cmd == nil {
		t.Fatalf("expected command")
	}
	if cmd.Use != "conductor" {
		t.Fatalf("unexpected use: %q", cmd.Use)
	}
	cmd.SetArgs([]string{"task"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute: %v", err)
	}
}

func TestNewRootCmdJob(t *testing.T) {
	cmd := newRootCmd()
	cmd.SetArgs([]string{"job"})
	if err := cmd.Execute(); err != nil {
		t.Fatalf("execute: %v", err)
	}
}

func TestParseBool(t *testing.T) {
	if !parseBool("true") {
		t.Fatalf("expected true")
	}
	if parseBool("no") {
		t.Fatalf("expected false")
	}
}

func TestRunServerInvalidPort(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.yaml")
	content := "api:\n  port: -1\n"
	if err := os.WriteFile(configPath, []byte(content), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}
	if err := runServer(configPath, dir, true, "", 0, ""); err == nil {
		t.Fatalf("expected error for invalid port")
	}
}

func TestMainHelp(t *testing.T) {
	origArgs := os.Args
	defer func() { os.Args = origArgs }()

	os.Args = []string{"conductor", "--help"}
	main()
}

func TestRunServerConfigFromEnv(t *testing.T) {
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.yaml")
	content := "api:\n  port: -1\nstorage:\n  runs_dir: " + dir + "\n"
	if err := os.WriteFile(configPath, []byte(content), 0o644); err != nil {
		t.Fatalf("write config: %v", err)
	}
	t.Setenv("CONDUCTOR_CONFIG", configPath)
	t.Setenv("CONDUCTOR_DISABLE_TASK_START", "yes")

	if err := runServer("", "", false, "", 0, ""); err == nil {
		t.Fatalf("expected error for invalid port from env config")
	}
}
