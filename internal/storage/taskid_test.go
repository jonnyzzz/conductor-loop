package storage

import (
	"strings"
	"testing"
	"time"
)

func TestValidateProjectID(t *testing.T) {
	valid := []string{
		"my-project",
		"myproject",
		"a",
		"project-123",
		"my-long-project-name-with-many-hyphens",
	}
	for _, id := range valid {
		if err := ValidateProjectID(id); err != nil {
			t.Errorf("expected valid project ID %q, got error: %v", id, err)
		}
	}

	invalid := []string{
		"",
		"   ",
		"project/nested",         // path separator — would nest under another project
		"project\\nested",        // Windows path separator
		"../escape",              // path traversal
		".",                      // relative reference
		"..",                     // relative reference
		"Project",                // uppercase not allowed
		"my_project",             // underscore not allowed
		"my project",             // space not allowed
		"-leading-hyphen",        // leading hyphen
		"trailing-hyphen-",       // trailing hyphen
	}
	for _, id := range invalid {
		if err := ValidateProjectID(id); err == nil {
			t.Errorf("expected invalid project ID %q to fail validation", id)
		}
	}
}

func TestValidateTaskID(t *testing.T) {
	valid := []string{
		"task-20260220-153045-my-feature",
		"task-20260220-153045-a3f9bc",
		"task-20260220-000000-bug-fix",
		"task-99991231-235959-abc",
		"task-20260220-153045-abc",
		"task-20260220-153045-a-b-c-d-e-f-g",
		"task-20260220-153045-" + strings.Repeat("a", 50), // max slug length
	}
	for _, id := range valid {
		if err := ValidateTaskID(id); err != nil {
			t.Errorf("expected valid task ID %q, got error: %v", id, err)
		}
	}

	invalid := []string{
		"",
		"my-task",
		"task-foo",
		"random-string",
		"task-2026022-153045-my-feature",   // 7-digit date
		"task-202602200-153045-my-feature",  // 9-digit date
		"task-20260220-15304-my-feature",    // 5-digit time
		"task-20260220-1530450-my-feature",  // 7-digit time
		"task-20260220-153045-My-Feature",   // uppercase letters
		"task-20260220-153045-my_feature",   // underscore
		"task-20260220-153045--my-feature",  // leading hyphen in slug
		"task-20260220-153045-my-feature-",  // trailing hyphen in slug
		"task-20260220-153045-a",            // slug too short (1 char)
		"task-20260220-153045-" + strings.Repeat("a", 51), // slug too long (51 chars)
		"TASK-20260220-153045-my-feature",   // uppercase prefix
	}
	for _, id := range invalid {
		if err := ValidateTaskID(id); err == nil {
			t.Errorf("expected invalid task ID %q to fail validation", id)
		}
	}
}

func TestGenerateTaskID(t *testing.T) {
	id := GenerateTaskID("")
	if err := ValidateTaskID(id); err != nil {
		t.Errorf("generated task ID %q failed validation: %v", id, err)
	}
	if !strings.HasPrefix(id, "task-") {
		t.Errorf("expected task ID to start with 'task-', got %q", id)
	}
	parts := strings.SplitN(id, "-", 4)
	if len(parts) != 4 {
		t.Fatalf("expected 4 parts in task ID %q, got %d", id, len(parts))
	}
	if len(parts[1]) != 8 {
		t.Errorf("expected 8-digit date in task ID %q, got %q", id, parts[1])
	}
	if len(parts[2]) != 6 {
		t.Errorf("expected 6-digit time in task ID %q, got %q", id, parts[2])
	}
	if len(parts[3]) != 6 {
		t.Errorf("expected 6-char random slug in task ID %q, got %q", id, parts[3])
	}
}

func TestGenerateTaskIDWithSlug(t *testing.T) {
	id := GenerateTaskID("my-feature")
	if err := ValidateTaskID(id); err != nil {
		t.Errorf("generated task ID %q failed validation: %v", id, err)
	}
	if !strings.HasSuffix(id, "-my-feature") {
		t.Errorf("expected task ID to end with '-my-feature', got %q", id)
	}
}

func TestGenerateTaskIDUsesCurrentTime(t *testing.T) {
	now := time.Date(2026, 2, 20, 15, 30, 45, 0, time.UTC)
	id := generateTaskIDAt(now, "test-slug")
	if !strings.HasPrefix(id, "task-20260220-153045-") {
		t.Errorf("expected task ID to start with 'task-20260220-153045-', got %q", id)
	}
	if id != "task-20260220-153045-test-slug" {
		t.Errorf("unexpected task ID %q", id)
	}
}

func TestGenerateTaskIDMultipleUnique(t *testing.T) {
	seen := make(map[string]bool)
	for i := 0; i < 20; i++ {
		id := GenerateTaskID("")
		if seen[id] {
			t.Errorf("duplicate task ID generated: %q", id)
		}
		seen[id] = true
	}
}
