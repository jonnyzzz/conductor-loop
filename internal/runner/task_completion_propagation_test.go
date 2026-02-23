package runner

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/jonnyzzz/conductor-loop/internal/messagebus"
	"github.com/jonnyzzz/conductor-loop/internal/storage"
)

func TestRunTask_PropagatesCompletionFactOnTransition(t *testing.T) {
	root := t.TempDir()
	projectID := "project"
	taskID := "task"
	taskDir := filepath.Join(root, projectID, taskID)
	if err := os.MkdirAll(taskDir, 0o755); err != nil {
		t.Fatalf("mkdir task: %v", err)
	}
	if err := os.WriteFile(filepath.Join(taskDir, "TASK.md"), []byte("do work"), 0o644); err != nil {
		t.Fatalf("write TASK.md: %v", err)
	}

	binDir := filepath.Join(root, "bin")
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		t.Fatalf("mkdir bin: %v", err)
	}
	createDoneWritingCLI(t, binDir, "codex")
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))

	if err := RunTask(projectID, taskID, TaskOptions{
		RootDir:        root,
		Agent:          "codex",
		MaxRestarts:    3,
		MaxRestartsSet: true,
		RestartDelay:   10 * time.Millisecond,
	}); err != nil {
		t.Fatalf("RunTask: %v", err)
	}

	projectBusPath := filepath.Join(root, projectID, "PROJECT-MESSAGE-BUS.md")
	projectBus, err := messagebus.NewMessageBus(projectBusPath)
	if err != nil {
		t.Fatalf("NewMessageBus(project): %v", err)
	}
	messages, err := projectBus.ReadMessages("")
	if err != nil {
		t.Fatalf("ReadMessages(project): %v", err)
	}
	propagation := findPropagationFactMessage(messages)
	if propagation == nil {
		t.Fatalf("expected propagation FACT in project bus")
	}
	if propagation.TaskID != taskID {
		t.Fatalf("expected task_id=%q, got %q", taskID, propagation.TaskID)
	}
	if strings.TrimSpace(propagation.RunID) == "" {
		t.Fatalf("expected propagated run_id")
	}
	if !strings.Contains(propagation.Body, "source_task_id: "+taskID) {
		t.Fatalf("expected source task id in body, got:\n%s", propagation.Body)
	}
	if !strings.Contains(propagation.Body, "run_outcome_summary:") {
		t.Fatalf("expected run summary in body, got:\n%s", propagation.Body)
	}
	if propagation.Meta == nil || propagation.Meta["source_latest_run_id"] == "" {
		t.Fatalf("expected source_latest_run_id in meta")
	}
}

func TestPropagateTaskCompletionToProject_ContentAndTraceability(t *testing.T) {
	root := t.TempDir()
	projectID := "project"
	taskID := "task"
	taskDir := filepath.Join(root, projectID, taskID)
	if err := os.MkdirAll(taskDir, 0o755); err != nil {
		t.Fatalf("mkdir task: %v", err)
	}
	if err := os.WriteFile(filepath.Join(taskDir, "DONE"), []byte(""), 0o644); err != nil {
		t.Fatalf("write DONE: %v", err)
	}

	startedAt := time.Date(2026, 2, 22, 10, 0, 0, 0, time.UTC)
	run1Dir := filepath.Join(taskDir, "runs", "run-001")
	run2Dir := filepath.Join(taskDir, "runs", "run-002")
	if err := os.MkdirAll(run1Dir, 0o755); err != nil {
		t.Fatalf("mkdir run-001: %v", err)
	}
	if err := os.MkdirAll(run2Dir, 0o755); err != nil {
		t.Fatalf("mkdir run-002: %v", err)
	}
	run1Output := filepath.Join(run1Dir, "output.md")
	run2Output := filepath.Join(run2Dir, "output.md")
	if err := os.WriteFile(run1Output, []byte("run1 output"), 0o644); err != nil {
		t.Fatalf("write run1 output: %v", err)
	}
	if err := os.WriteFile(run2Output, []byte("run2 output"), 0o644); err != nil {
		t.Fatalf("write run2 output: %v", err)
	}
	if err := storage.WriteRunInfo(filepath.Join(run1Dir, "run-info.yaml"), &storage.RunInfo{
		Version:      1,
		RunID:        "run-001",
		ProjectID:    projectID,
		TaskID:       taskID,
		AgentType:    "codex",
		StartTime:    startedAt,
		EndTime:      startedAt.Add(2 * time.Minute),
		ExitCode:     1,
		Status:       storage.StatusFailed,
		OutputPath:   run1Output,
		AgentVersion: "1.0.0",
	}); err != nil {
		t.Fatalf("write run-001 run-info: %v", err)
	}
	if err := storage.WriteRunInfo(filepath.Join(run2Dir, "run-info.yaml"), &storage.RunInfo{
		Version:      1,
		RunID:        "run-002",
		ProjectID:    projectID,
		TaskID:       taskID,
		AgentType:    "codex",
		StartTime:    startedAt.Add(3 * time.Minute),
		EndTime:      startedAt.Add(5 * time.Minute),
		ExitCode:     0,
		Status:       storage.StatusCompleted,
		OutputPath:   run2Output,
		AgentVersion: "1.0.0",
	}); err != nil {
		t.Fatalf("write run-002 run-info: %v", err)
	}

	taskBusPath := filepath.Join(taskDir, "TASK-MESSAGE-BUS.md")
	taskBus, err := messagebus.NewMessageBus(taskBusPath)
	if err != nil {
		t.Fatalf("NewMessageBus(task): %v", err)
	}
	factID1, err := taskBus.AppendMessage(&messagebus.Message{
		Type:      "FACT",
		ProjectID: projectID,
		TaskID:    taskID,
		RunID:     "run-001",
		Body:      "first fact from run 1",
	})
	if err != nil {
		t.Fatalf("append fact1: %v", err)
	}
	factID2, err := taskBus.AppendMessage(&messagebus.Message{
		Type:      "FACT",
		ProjectID: projectID,
		TaskID:    taskID,
		RunID:     "run-002",
		Body:      "second fact from run 2",
	})
	if err != nil {
		t.Fatalf("append fact2: %v", err)
	}
	runStopID, err := taskBus.AppendMessage(&messagebus.Message{
		Type:      messagebus.EventTypeRunStop,
		ProjectID: projectID,
		TaskID:    taskID,
		RunID:     "run-002",
		Body:      "run stopped with code 0",
	})
	if err != nil {
		t.Fatalf("append run stop: %v", err)
	}

	result, err := propagateTaskCompletionToProject(root, projectID, taskID, taskDir, taskBusPath)
	if err != nil {
		t.Fatalf("propagateTaskCompletionToProject: %v", err)
	}
	if !result.Posted {
		t.Fatalf("expected propagation to post")
	}
	if result.ProjectMessageID == "" {
		t.Fatalf("expected project message id")
	}

	projectBusPath := filepath.Join(root, projectID, "PROJECT-MESSAGE-BUS.md")
	projectBus, err := messagebus.NewMessageBus(projectBusPath)
	if err != nil {
		t.Fatalf("NewMessageBus(project): %v", err)
	}
	projectMessages, err := projectBus.ReadMessages("")
	if err != nil {
		t.Fatalf("ReadMessages(project): %v", err)
	}
	if len(projectMessages) != 1 {
		t.Fatalf("expected 1 project message, got %d", len(projectMessages))
	}
	msg := projectMessages[0]
	if msg.Type != "FACT" {
		t.Fatalf("expected FACT, got %q", msg.Type)
	}
	if msg.TaskID != taskID {
		t.Fatalf("expected task_id=%q, got %q", taskID, msg.TaskID)
	}
	if msg.Meta == nil {
		t.Fatalf("expected metadata")
	}
	if got := msg.Meta["kind"]; got != taskCompletionPropagationMetaKind {
		t.Fatalf("expected kind=%q, got %q", taskCompletionPropagationMetaKind, got)
	}
	if got := msg.Meta["source_task_id"]; got != taskID {
		t.Fatalf("expected source_task_id=%q, got %q", taskID, got)
	}
	if got := msg.Meta["source_latest_run_id"]; got != "run-002" {
		t.Fatalf("expected source_latest_run_id=run-002, got %q", got)
	}
	if got := msg.Meta["source_output_path"]; got != run2Output {
		t.Fatalf("expected source_output_path=%q, got %q", run2Output, got)
	}
	if !strings.Contains(msg.Body, "run_ids: run-001, run-002") {
		t.Fatalf("expected run ids in body, got:\n%s", msg.Body)
	}
	if !strings.Contains(msg.Body, "task_fact_signals:") {
		t.Fatalf("expected task fact section in body, got:\n%s", msg.Body)
	}
	if !strings.Contains(msg.Body, run2Output) {
		t.Fatalf("expected output path in body, got:\n%s", msg.Body)
	}

	if !hasParent(msg.Parents, factID1) {
		t.Fatalf("expected parent for fact1 id %s", factID1)
	}
	if !hasParent(msg.Parents, factID2) {
		t.Fatalf("expected parent for fact2 id %s", factID2)
	}
	if !hasParent(msg.Parents, runStopID) {
		t.Fatalf("expected parent for run stop id %s", runStopID)
	}

	if !hasLink(msg.Links, taskBusPath, "task_bus") {
		t.Fatalf("expected task bus link")
	}
	if !hasLink(msg.Links, run2Output, "output") {
		t.Fatalf("expected output link")
	}
}

func TestPropagateTaskCompletionToProject_Idempotent(t *testing.T) {
	root := t.TempDir()
	projectID := "project"
	taskID := "task"
	taskDir := filepath.Join(root, projectID, taskID)
	if err := os.MkdirAll(taskDir, 0o755); err != nil {
		t.Fatalf("mkdir task: %v", err)
	}
	if err := os.WriteFile(filepath.Join(taskDir, "DONE"), []byte(""), 0o644); err != nil {
		t.Fatalf("write DONE: %v", err)
	}

	runDir := filepath.Join(taskDir, "runs", "run-001")
	if err := os.MkdirAll(runDir, 0o755); err != nil {
		t.Fatalf("mkdir run-001: %v", err)
	}
	runOutput := filepath.Join(runDir, "output.md")
	if err := os.WriteFile(runOutput, []byte("output"), 0o644); err != nil {
		t.Fatalf("write output: %v", err)
	}
	if err := storage.WriteRunInfo(filepath.Join(runDir, "run-info.yaml"), &storage.RunInfo{
		Version:      1,
		RunID:        "run-001",
		ProjectID:    projectID,
		TaskID:       taskID,
		AgentType:    "codex",
		StartTime:    time.Now().UTC().Add(-2 * time.Minute),
		EndTime:      time.Now().UTC().Add(-time.Minute),
		ExitCode:     0,
		Status:       storage.StatusCompleted,
		OutputPath:   runOutput,
		AgentVersion: "1.0.0",
	}); err != nil {
		t.Fatalf("write run-info: %v", err)
	}

	taskBusPath := filepath.Join(taskDir, "TASK-MESSAGE-BUS.md")
	taskBus, err := messagebus.NewMessageBus(taskBusPath)
	if err != nil {
		t.Fatalf("NewMessageBus(task): %v", err)
	}
	if _, err := taskBus.AppendMessage(&messagebus.Message{
		Type:      "FACT",
		ProjectID: projectID,
		TaskID:    taskID,
		RunID:     "run-001",
		Body:      "fact",
	}); err != nil {
		t.Fatalf("append fact: %v", err)
	}

	first, err := propagateTaskCompletionToProject(root, projectID, taskID, taskDir, taskBusPath)
	if err != nil {
		t.Fatalf("first propagate: %v", err)
	}
	if !first.Posted {
		t.Fatalf("expected first propagation to post")
	}

	second, err := propagateTaskCompletionToProject(root, projectID, taskID, taskDir, taskBusPath)
	if err != nil {
		t.Fatalf("second propagate: %v", err)
	}
	if second.Posted {
		t.Fatalf("expected second propagation to skip")
	}
	if second.PropagationKey != first.PropagationKey {
		t.Fatalf("expected same propagation key")
	}

	projectBusPath := filepath.Join(root, projectID, "PROJECT-MESSAGE-BUS.md")
	projectBus, err := messagebus.NewMessageBus(projectBusPath)
	if err != nil {
		t.Fatalf("NewMessageBus(project): %v", err)
	}
	projectMessages, err := projectBus.ReadMessages("")
	if err != nil {
		t.Fatalf("ReadMessages(project): %v", err)
	}
	if countPropagationFacts(projectMessages) != 1 {
		t.Fatalf("expected exactly one propagated FACT after second call")
	}

	statePath := filepath.Join(taskDir, taskCompletionPropagationStateFile)
	if err := os.Remove(statePath); err != nil {
		t.Fatalf("remove propagation state: %v", err)
	}
	third, err := propagateTaskCompletionToProject(root, projectID, taskID, taskDir, taskBusPath)
	if err != nil {
		t.Fatalf("third propagate: %v", err)
	}
	if third.Posted {
		t.Fatalf("expected third propagation to skip due existing project FACT")
	}

	projectMessages, err = projectBus.ReadMessages("")
	if err != nil {
		t.Fatalf("ReadMessages(project) after third call: %v", err)
	}
	if countPropagationFacts(projectMessages) != 1 {
		t.Fatalf("expected exactly one propagated FACT after third call")
	}
}

func TestRunTask_PropagationFailureDoesNotFailTask(t *testing.T) {
	root := t.TempDir()
	projectID := "project"
	taskID := "task"
	taskDir := filepath.Join(root, projectID, taskID)
	if err := os.MkdirAll(taskDir, 0o755); err != nil {
		t.Fatalf("mkdir task: %v", err)
	}
	if err := os.WriteFile(filepath.Join(taskDir, "TASK.md"), []byte("done"), 0o644); err != nil {
		t.Fatalf("write TASK.md: %v", err)
	}
	if err := os.WriteFile(filepath.Join(taskDir, "DONE"), []byte(""), 0o644); err != nil {
		t.Fatalf("write DONE: %v", err)
	}

	projectBusAsDir := filepath.Join(root, projectID, "PROJECT-MESSAGE-BUS.md")
	if err := os.MkdirAll(projectBusAsDir, 0o755); err != nil {
		t.Fatalf("mkdir project bus directory: %v", err)
	}

	binDir := filepath.Join(root, "bin")
	if err := os.MkdirAll(binDir, 0o755); err != nil {
		t.Fatalf("mkdir bin: %v", err)
	}
	createFakeCLI(t, binDir, "codex")
	t.Setenv("PATH", binDir+string(os.PathListSeparator)+os.Getenv("PATH"))

	if err := RunTask(projectID, taskID, TaskOptions{
		RootDir: root,
		Agent:   "codex",
	}); err != nil {
		t.Fatalf("RunTask should succeed despite propagation failure: %v", err)
	}

	taskBus, err := messagebus.NewMessageBus(filepath.Join(taskDir, "TASK-MESSAGE-BUS.md"))
	if err != nil {
		t.Fatalf("NewMessageBus(task): %v", err)
	}
	taskMessages, err := taskBus.ReadMessages("")
	if err != nil {
		t.Fatalf("ReadMessages(task): %v", err)
	}
	foundError := false
	for _, msg := range taskMessages {
		if msg == nil {
			continue
		}
		if strings.TrimSpace(strings.ToUpper(msg.Type)) != "ERROR" {
			continue
		}
		if strings.Contains(msg.Body, "task completion fact propagation failed") {
			foundError = true
			break
		}
	}
	if !foundError {
		t.Fatalf("expected propagation failure error in task bus")
	}
}

func createDoneWritingCLI(t *testing.T, dir, name string) {
	t.Helper()
	if runtime.GOOS == "windows" {
		path := filepath.Join(dir, name+".bat")
		content := "@echo off\r\n" +
			"if \"%1\"==\"--version\" (\r\n" +
			"  echo " + name + " 1.0.0\r\n" +
			"  exit /b 0\r\n" +
			")\r\n" +
			"more >nul\r\n" +
			"type nul > \"%TASK_FOLDER%\\DONE\"\r\n" +
			"echo stdout\r\n"
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatalf("write bat: %v", err)
		}
		return
	}
	path := filepath.Join(dir, name)
	content := "#!/bin/sh\n" +
		"if [ \"$1\" = \"--version\" ]; then echo '" + name + " 1.0.0'; exit 0; fi\n" +
		"cat >/dev/null\n" +
		": > \"$TASK_FOLDER/DONE\"\n" +
		"echo stdout\n"
	if err := os.WriteFile(path, []byte(content), 0o755); err != nil {
		t.Fatalf("write script: %v", err)
	}
}

func findPropagationFactMessage(messages []*messagebus.Message) *messagebus.Message {
	for _, msg := range messages {
		if msg == nil || msg.Meta == nil {
			continue
		}
		if strings.TrimSpace(msg.Meta["kind"]) == taskCompletionPropagationMetaKind {
			return msg
		}
	}
	return nil
}

func countPropagationFacts(messages []*messagebus.Message) int {
	count := 0
	for _, msg := range messages {
		if msg == nil || msg.Meta == nil {
			continue
		}
		if strings.TrimSpace(msg.Meta["kind"]) == taskCompletionPropagationMetaKind {
			count++
		}
	}
	return count
}

func hasParent(parents []messagebus.Parent, msgID string) bool {
	for _, parent := range parents {
		if strings.TrimSpace(parent.MsgID) == msgID {
			return true
		}
	}
	return false
}

func hasLink(links []messagebus.Link, url, kind string) bool {
	for _, link := range links {
		if strings.TrimSpace(link.URL) == url && strings.TrimSpace(link.Kind) == kind {
			return true
		}
	}
	return false
}
