package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestInstallShellSetupBlock_Idempotent(t *testing.T) {
	block := buildShellSetupBlock("run-agent")
	original := "# existing\n"

	installed, changed, err := installShellSetupBlock(original, block)
	if err != nil {
		t.Fatalf("installShellSetupBlock: %v", err)
	}
	if !changed {
		t.Fatal("expected first install to change content")
	}

	installedAgain, changedAgain, err := installShellSetupBlock(installed, block)
	if err != nil {
		t.Fatalf("installShellSetupBlock second call: %v", err)
	}
	if changedAgain {
		t.Fatal("expected second install to be idempotent")
	}
	if installedAgain != installed {
		t.Fatal("expected second install content to match first install")
	}
	if strings.Count(installedAgain, shellSetupBlockStart) != 1 {
		t.Fatalf("expected one managed block, got %d", strings.Count(installedAgain, shellSetupBlockStart))
	}
}

func TestUninstallShellSetupBlock_Idempotent(t *testing.T) {
	block := buildShellSetupBlock("run-agent")
	installed, _, err := installShellSetupBlock("# existing\n", block)
	if err != nil {
		t.Fatalf("installShellSetupBlock: %v", err)
	}

	removed, changed, err := uninstallShellSetupBlock(installed)
	if err != nil {
		t.Fatalf("uninstallShellSetupBlock: %v", err)
	}
	if !changed {
		t.Fatal("expected uninstall to remove managed block")
	}
	if strings.Contains(removed, shellSetupBlockStart) {
		t.Fatal("managed block still present after uninstall")
	}

	removedAgain, changedAgain, err := uninstallShellSetupBlock(removed)
	if err != nil {
		t.Fatalf("uninstallShellSetupBlock second call: %v", err)
	}
	if changedAgain {
		t.Fatal("expected second uninstall to be idempotent")
	}
	if removedAgain != removed {
		t.Fatal("expected second uninstall content to match first uninstall")
	}
}

func TestShellSetupInstallAndUninstallCommand(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("SHELL", "/bin/zsh")

	rcPath := filepath.Join(home, ".zshrc")
	if err := os.WriteFile(rcPath, []byte("# existing\n"), 0o644); err != nil {
		t.Fatalf("write initial rc file: %v", err)
	}

	run := func(args ...string) error {
		cmd := newRootCmd()
		var stdout bytes.Buffer
		cmd.SetOut(&stdout)
		cmd.SetErr(&stdout)
		cmd.SetArgs(args)
		return cmd.Execute()
	}

	if err := run("shell-setup", "install"); err != nil {
		t.Fatalf("shell-setup install: %v", err)
	}
	rcData, err := os.ReadFile(rcPath)
	if err != nil {
		t.Fatalf("read rc file after install: %v", err)
	}
	text := string(rcData)
	if strings.Count(text, shellSetupBlockStart) != 1 {
		t.Fatalf("expected one managed block after install, got %d", strings.Count(text, shellSetupBlockStart))
	}
	for _, agentName := range shellSetupAgents {
		want := "alias " + agentName + "='run-agent wrap --agent " + agentName + " --'"
		if !strings.Contains(text, want) {
			t.Fatalf("missing alias line: %s", want)
		}
	}

	if err := run("shell-setup", "install"); err != nil {
		t.Fatalf("shell-setup install (idempotent): %v", err)
	}
	rcData, err = os.ReadFile(rcPath)
	if err != nil {
		t.Fatalf("read rc file after second install: %v", err)
	}
	if strings.Count(string(rcData), shellSetupBlockStart) != 1 {
		t.Fatalf("expected one managed block after second install, got %d", strings.Count(string(rcData), shellSetupBlockStart))
	}

	if err := run("shell-setup", "uninstall"); err != nil {
		t.Fatalf("shell-setup uninstall: %v", err)
	}
	rcData, err = os.ReadFile(rcPath)
	if err != nil {
		t.Fatalf("read rc file after uninstall: %v", err)
	}
	text = string(rcData)
	if strings.Contains(text, shellSetupBlockStart) {
		t.Fatal("managed block should be removed after uninstall")
	}
	if !strings.Contains(text, "# existing\n") {
		t.Fatal("existing rc content should be preserved")
	}

	if err := run("shell-setup", "uninstall"); err != nil {
		t.Fatalf("shell-setup uninstall (idempotent): %v", err)
	}
}

func TestShellSetupInstallRejectsInvalidRunAgentBin(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("SHELL", "/bin/zsh")

	cmd := newRootCmd()
	cmd.SetArgs([]string{"shell-setup", "install", "--run-agent-bin", "run agent"})
	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error for invalid run-agent-bin")
	}
	if !strings.Contains(err.Error(), "single shell token") {
		t.Fatalf("unexpected error: %v", err)
	}
}
