package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/jonnyzzz/conductor-loop/internal/config"
	"github.com/jonnyzzz/conductor-loop/internal/runner"
	"github.com/jonnyzzz/conductor-loop/internal/storage"
)

// resolveOrInitProject attempts to find an existing project for the current
// working directory, interactively offering to create one when none is found
// and stdin is a terminal.
//
// Lookup order:
//  1. Scan all home-folders.md files under runsRoot for an exact project_root match.
//  2. Interactive TTY prompt — propose a project ID derived from the directory
//     base name and let the user accept, edit, or abort.
//
// Returns the project ID or an error.
func resolveOrInitProject(rootDir string) (string, error) {
	runsRoot, err := config.ResolveRunsDir(rootDir)
	if err != nil {
		return "", fmt.Errorf("resolve runs dir: %w", err)
	}

	// Locate the project root directory by scanning upward from CWD.
	projectRootDir := runner.FindProjectRoot("")
	if projectRootDir == "" {
		// No marker found — fall back to CWD.
		cwd, err := os.Getwd()
		if err != nil {
			return "", fmt.Errorf("get working directory: %w", err)
		}
		projectRootDir = cwd
	}

	// Check whether any existing project already maps to this directory.
	if projectID := storage.FindProjectByDir(runsRoot, projectRootDir); projectID != "" {
		return projectID, nil
	}

	// Not found in runs storage. Prompt if stdin is a terminal.
	if !isTTY(os.Stdin) {
		return "", fmt.Errorf("cannot infer project: run from inside a task or project directory")
	}

	suggested := runner.SuggestProjectID(filepath.Base(projectRootDir))

	fmt.Fprintf(os.Stderr, "No project found for: %s\n", projectRootDir)
	if suggested != "" {
		fmt.Fprintf(os.Stderr, "Project ID [%s]: ", suggested)
	} else {
		fmt.Fprintf(os.Stderr, "Project ID: ")
	}

	scanner := bufio.NewScanner(os.Stdin)
	if !scanner.Scan() {
		return "", fmt.Errorf("project init cancelled")
	}
	input := strings.TrimSpace(scanner.Text())
	if input == "" {
		input = suggested
	}
	if input == "" {
		return "", fmt.Errorf("project ID is required")
	}
	if err := storage.ValidateProjectID(input); err != nil {
		return "", err
	}

	hf := &storage.HomeFolders{
		ProjectRoot: projectRootDir,
	}
	if _, err := storage.InitProject(runsRoot, input, hf); err != nil {
		return "", fmt.Errorf("init project %q: %w", input, err)
	}
	fmt.Fprintf(os.Stderr, "Project %q created at %s\n", input, filepath.Join(runsRoot, input))
	return input, nil
}

// isTTY reports whether f is connected to a terminal.
func isTTY(f *os.File) bool {
	info, err := f.Stat()
	if err != nil {
		return false
	}
	return (info.Mode() & os.ModeCharDevice) != 0
}
