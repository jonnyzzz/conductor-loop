package facts_test

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/jonnyzzz/conductor-loop/internal/facts"
	"github.com/jonnyzzz/conductor-loop/internal/messagebus"
)

// makeTaskBus creates a task directory with a TASK-MESSAGE-BUS.md and
// returns the path to the bus file.
func makeTaskBus(t *testing.T, root, projectID, taskID string) string {
	t.Helper()
	taskDir := filepath.Join(root, projectID, taskID)
	if err := os.MkdirAll(taskDir, 0o755); err != nil {
		t.Fatalf("mkdir task dir: %v", err)
	}
	busPath := filepath.Join(taskDir, "TASK-MESSAGE-BUS.md")
	return busPath
}

func appendMessage(t *testing.T, busPath, projectID, taskID, msgType, body string) {
	t.Helper()
	bus, err := messagebus.NewMessageBus(busPath)
	if err != nil {
		t.Fatalf("new bus: %v", err)
	}
	msg := &messagebus.Message{
		Type:      msgType,
		ProjectID: projectID,
		TaskID:    taskID,
		Body:      body,
	}
	if _, err := bus.AppendMessage(msg); err != nil {
		t.Fatalf("append message: %v", err)
	}
}

func TestPromoteFacts_Basic(t *testing.T) {
	root := t.TempDir()
	projectID := "proj-basic"
	taskID := "task-20260101-120000-alpha"

	busPath := makeTaskBus(t, root, projectID, taskID)
	appendMessage(t, busPath, projectID, taskID, "FACT", "The sky is blue")
	appendMessage(t, busPath, projectID, taskID, "FACT", "Water is wet")

	cfg := facts.PromoteConfig{
		RootDir:   root,
		ProjectID: projectID,
	}
	promoted, already, err := facts.PromoteFacts(cfg)
	if err != nil {
		t.Fatalf("PromoteFacts error: %v", err)
	}
	if promoted != 2 {
		t.Errorf("promoted = %d; want 2", promoted)
	}
	if already != 0 {
		t.Errorf("already = %d; want 0", already)
	}

	// Verify PROJECT-FACTS.md was written.
	factsPath := filepath.Join(root, projectID, "PROJECT-FACTS.md")
	data, err := os.ReadFile(factsPath)
	if err != nil {
		t.Fatalf("read facts file: %v", err)
	}
	content := string(data)

	if !strings.Contains(content, "# Project Facts: "+projectID) {
		t.Errorf("missing project header in facts file")
	}
	if !strings.Contains(content, "- **Body**: The sky is blue") {
		t.Errorf("missing fact 'The sky is blue' body line in facts file")
	}
	if !strings.Contains(content, "- **Body**: Water is wet") {
		t.Errorf("missing fact 'Water is wet' body line in facts file")
	}
	if !strings.Contains(content, "Source task") {
		t.Errorf("missing Source task field in facts file")
	}
	if !strings.Contains(content, taskID) {
		t.Errorf("missing task ID %q in facts file", taskID)
	}
}

func TestPromoteFacts_Idempotent(t *testing.T) {
	root := t.TempDir()
	projectID := "proj-idempotent"
	taskID := "task-20260101-120000-beta"

	busPath := makeTaskBus(t, root, projectID, taskID)
	appendMessage(t, busPath, projectID, taskID, "FACT", "Idempotency fact one")
	appendMessage(t, busPath, projectID, taskID, "FACT", "Idempotency fact two")

	cfg := facts.PromoteConfig{
		RootDir:   root,
		ProjectID: projectID,
	}

	// First promotion.
	p1, a1, err := facts.PromoteFacts(cfg)
	if err != nil {
		t.Fatalf("first PromoteFacts error: %v", err)
	}
	if p1 != 2 {
		t.Errorf("first run: promoted = %d; want 2", p1)
	}
	if a1 != 0 {
		t.Errorf("first run: already = %d; want 0", a1)
	}

	// Second promotion — should find no new facts.
	p2, a2, err := facts.PromoteFacts(cfg)
	if err != nil {
		t.Fatalf("second PromoteFacts error: %v", err)
	}
	if p2 != 0 {
		t.Errorf("second run: promoted = %d; want 0", p2)
	}
	if a2 != 2 {
		t.Errorf("second run: already = %d; want 2", a2)
	}

	// Verify no duplicates in file: count Body lines specifically.
	factsPath := filepath.Join(root, projectID, "PROJECT-FACTS.md")
	data, err := os.ReadFile(factsPath)
	if err != nil {
		t.Fatalf("read facts file: %v", err)
	}
	content := string(data)
	bodyLine := "- **Body**: Idempotency fact one"
	count := strings.Count(content, bodyLine)
	if count != 1 {
		t.Errorf("body line %q appears %d times in file; want 1", bodyLine, count)
	}
}

func TestPromoteFacts_DryRun(t *testing.T) {
	root := t.TempDir()
	projectID := "proj-dryrun"
	taskID := "task-20260101-120000-gamma"

	busPath := makeTaskBus(t, root, projectID, taskID)
	appendMessage(t, busPath, projectID, taskID, "FACT", "Dry run fact")

	cfg := facts.PromoteConfig{
		RootDir:   root,
		ProjectID: projectID,
		DryRun:    true,
	}
	promoted, already, err := facts.PromoteFacts(cfg)
	if err != nil {
		t.Fatalf("PromoteFacts error: %v", err)
	}
	if promoted != 1 {
		t.Errorf("promoted = %d; want 1", promoted)
	}
	if already != 0 {
		t.Errorf("already = %d; want 0", already)
	}

	// Verify no file was written.
	factsPath := filepath.Join(root, projectID, "PROJECT-FACTS.md")
	if _, err := os.Stat(factsPath); !os.IsNotExist(err) {
		t.Errorf("PROJECT-FACTS.md should not exist on dry-run but stat returned: %v", err)
	}
}

func TestPromoteFacts_FilterType(t *testing.T) {
	root := t.TempDir()
	projectID := "proj-filter"
	taskID := "task-20260101-120000-delta"

	busPath := makeTaskBus(t, root, projectID, taskID)
	appendMessage(t, busPath, projectID, taskID, "FACT", "This is a real fact")
	appendMessage(t, busPath, projectID, taskID, "PROGRESS", "Step 1 complete")
	appendMessage(t, busPath, projectID, taskID, "INFO", "Informational message")
	appendMessage(t, busPath, projectID, taskID, "FACT", "Another real fact")

	cfg := facts.PromoteConfig{
		RootDir:   root,
		ProjectID: projectID,
		// Default FilterType="FACT"
	}
	promoted, already, err := facts.PromoteFacts(cfg)
	if err != nil {
		t.Fatalf("PromoteFacts error: %v", err)
	}
	if promoted != 2 {
		t.Errorf("promoted = %d; want 2 (only FACT messages)", promoted)
	}
	if already != 0 {
		t.Errorf("already = %d; want 0", already)
	}

	factsPath := filepath.Join(root, projectID, "PROJECT-FACTS.md")
	data, err := os.ReadFile(factsPath)
	if err != nil {
		t.Fatalf("read facts file: %v", err)
	}
	content := string(data)
	if strings.Contains(content, "Step 1 complete") {
		t.Errorf("PROGRESS message should not be in PROJECT-FACTS.md")
	}
	if strings.Contains(content, "Informational message") {
		t.Errorf("INFO message should not be in PROJECT-FACTS.md")
	}
	if !strings.Contains(content, "- **Body**: This is a real fact") {
		t.Errorf("FACT message 'This is a real fact' body line should be in PROJECT-FACTS.md")
	}
	if !strings.Contains(content, "- **Body**: Another real fact") {
		t.Errorf("FACT message 'Another real fact' body line should be in PROJECT-FACTS.md")
	}
}

func TestPromoteFacts_Since(t *testing.T) {
	root := t.TempDir()
	projectID := "proj-since"
	taskID := "task-20260101-120000-epsilon"

	busPath := makeTaskBus(t, root, projectID, taskID)
	appendMessage(t, busPath, projectID, taskID, "FACT", "Old fact before cutoff")

	// Cutoff is set after the first message.
	cutoff := time.Now()

	// New message after cutoff.
	appendMessage(t, busPath, projectID, taskID, "FACT", "New fact after cutoff")

	cfg := facts.PromoteConfig{
		RootDir:   root,
		ProjectID: projectID,
		Since:     cutoff,
	}
	promoted, already, err := facts.PromoteFacts(cfg)
	if err != nil {
		t.Fatalf("PromoteFacts error: %v", err)
	}
	// The old fact should be filtered; at most 1 promoted.
	if promoted > 1 {
		t.Errorf("promoted = %d; want at most 1 (since filter should exclude old fact)", promoted)
	}
	_ = already
}

func TestPromoteFacts_EmptyProject(t *testing.T) {
	root := t.TempDir()
	projectID := "proj-empty"

	// Create the project dir but no tasks.
	if err := os.MkdirAll(filepath.Join(root, projectID), 0o755); err != nil {
		t.Fatalf("mkdir: %v", err)
	}

	cfg := facts.PromoteConfig{
		RootDir:   root,
		ProjectID: projectID,
	}
	promoted, already, err := facts.PromoteFacts(cfg)
	if err != nil {
		t.Fatalf("PromoteFacts error: %v", err)
	}
	if promoted != 0 {
		t.Errorf("promoted = %d; want 0", promoted)
	}
	if already != 0 {
		t.Errorf("already = %d; want 0", already)
	}
}

func TestPromoteFacts_MultipleTaskDeduplication(t *testing.T) {
	root := t.TempDir()
	projectID := "proj-multidup"
	task1 := "task-20260101-120000-zeta"
	task2 := "task-20260101-130000-eta"

	bus1 := makeTaskBus(t, root, projectID, task1)
	bus2 := makeTaskBus(t, root, projectID, task2)

	// Same fact in both tasks.
	appendMessage(t, bus1, projectID, task1, "FACT", "Shared fact across tasks")
	appendMessage(t, bus2, projectID, task2, "FACT", "Shared fact across tasks")
	appendMessage(t, bus2, projectID, task2, "FACT", "Unique to task2")

	cfg := facts.PromoteConfig{
		RootDir:   root,
		ProjectID: projectID,
	}
	promoted, already, err := facts.PromoteFacts(cfg)
	if err != nil {
		t.Fatalf("PromoteFacts error: %v", err)
	}
	// Should promote 2 unique facts: "Shared fact across tasks" once + "Unique to task2"
	if promoted != 2 {
		t.Errorf("promoted = %d; want 2 (duplicate cross-task fact deduplicated)", promoted)
	}
	_ = already

	factsPath := filepath.Join(root, projectID, "PROJECT-FACTS.md")
	data, err := os.ReadFile(factsPath)
	if err != nil {
		t.Fatalf("read facts file: %v", err)
	}
	content := string(data)
	// Count only the body lines to detect true duplicates.
	bodyLine := "- **Body**: Shared fact across tasks"
	count := strings.Count(content, bodyLine)
	if count != 1 {
		t.Errorf("body line %q appears %d times; want 1", bodyLine, count)
	}
}
