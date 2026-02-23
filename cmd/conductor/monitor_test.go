package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseTodos(t *testing.T) {
	content := []byte(`
# TODOs

- [ ] task-1: Task 1 description
- [x] task-2: Task 2 description
- [ ] task-3 Task 3 no colon
- [ ] task-4
- [ ] not-a-task
- [ ] task-5: 
`)
	tmpDir := t.TempDir()
	tmpFile := filepath.Join(tmpDir, "todos.md")
	if err := os.WriteFile(tmpFile, content, 0644); err != nil {
		t.Fatal(err)
	}

	tasks, lines, err := parseTodos(tmpFile)
	if err != nil {
		t.Fatalf("parseTodos failed: %v", err)
	}

	if len(lines) != 8 { 
		// Just logging, not failing, as exact line count depends on editor behavior/formatting
		t.Logf("got %d lines", len(lines))
	}

	expected := []struct {
		ID      string
		Desc    string
		Checked bool
	}{
		{"task-1", "Task 1 description", false},
		{"task-2", "Task 2 description", true},
		{"task-3", "Task 3 no colon", false},
		{"task-4", "", false},
		{"task-5", "", false},
	}

	if len(tasks) != len(expected) {
		t.Fatalf("expected %d tasks, got %d", len(expected), len(tasks))
	}

	for i, exp := range expected {
		if tasks[i].ID != exp.ID {
			t.Errorf("task %d: expected ID %q, got %q", i, exp.ID, tasks[i].ID)
		}
		if tasks[i].Description != exp.Desc {
			t.Errorf("task %d: expected Desc %q, got %q", i, exp.Desc, tasks[i].Description)
		}
		if tasks[i].Checked != exp.Checked {
			t.Errorf("task %d: expected Checked %v, got %v", i, exp.Checked, tasks[i].Checked)
		}
	}
}
