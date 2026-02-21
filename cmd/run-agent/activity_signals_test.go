package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/jonnyzzz/conductor-loop/internal/messagebus"
)

func writeTaskBusMessage(t *testing.T, busPath, projectID, taskID, msgType, body string, ts time.Time) {
	t.Helper()
	bus, err := messagebus.NewMessageBus(busPath)
	if err != nil {
		t.Fatalf("new message bus: %v", err)
	}
	_, err = bus.AppendMessage(&messagebus.Message{
		Timestamp: ts,
		Type:      msgType,
		ProjectID: projectID,
		TaskID:    taskID,
		Body:      body,
	})
	if err != nil {
		t.Fatalf("append bus message: %v", err)
	}
}

func TestCollectTaskActivitySignals_ReportsLatestBusOutputAndDrift(t *testing.T) {
	root := t.TempDir()
	projectID := "proj"
	taskID := "task-20260221-activity-aa"
	taskDir := filepath.Join(root, projectID, taskID)
	runID := "run-001"
	runDir := filepath.Join(taskDir, "runs", runID)
	if err := os.MkdirAll(runDir, 0o755); err != nil {
		t.Fatalf("mkdir run dir: %v", err)
	}

	busPath := filepath.Join(taskDir, "TASK-MESSAGE-BUS.md")
	meaningfulAt := time.Date(2026, 2, 21, 20, 0, 0, 0, time.UTC)
	latestBusAt := meaningfulAt.Add(10 * time.Minute)
	outputAt := meaningfulAt.Add(12 * time.Minute)
	now := meaningfulAt.Add(35 * time.Minute)
	runStart := meaningfulAt.Add(-5 * time.Minute)

	writeTaskBusMessage(t, busPath, projectID, taskID, "FACT", "completed API contract and shipped tests", meaningfulAt)
	writeTaskBusMessage(t, busPath, projectID, taskID, "PROGRESS", "still evaluating edge cases with broad analysis text to test preview truncation behavior in status output", latestBusAt)

	outputPath := filepath.Join(runDir, "output.md")
	if err := os.WriteFile(outputPath, []byte("latest output"), 0o644); err != nil {
		t.Fatalf("write output.md: %v", err)
	}
	if err := os.Chtimes(outputPath, outputAt, outputAt); err != nil {
		t.Fatalf("chtimes output.md: %v", err)
	}

	signals := collectTaskActivitySignals(
		taskDir,
		runID,
		"running",
		runStart,
		outputPath,
		"",
		"",
		activityOptions{
			Enabled:    true,
			DriftAfter: 20 * time.Minute,
			Now: func() time.Time {
				return now
			},
		},
	)

	if signals.LatestBusMessage == nil {
		t.Fatalf("expected latest_bus_message")
	}
	if signals.LatestBusMessage.Type != "PROGRESS" {
		t.Fatalf("latest bus type=%q, want PROGRESS", signals.LatestBusMessage.Type)
	}
	if signals.LatestBusMessage.Timestamp != latestBusAt.Format(time.RFC3339) {
		t.Fatalf("latest bus timestamp=%q, want %q", signals.LatestBusMessage.Timestamp, latestBusAt.Format(time.RFC3339))
	}
	if !strings.Contains(signals.LatestBusMessage.BodyPreview, "still evaluating edge cases") {
		t.Fatalf("unexpected bus preview=%q", signals.LatestBusMessage.BodyPreview)
	}
	if signals.LastMeaningfulSignalAt == nil || *signals.LastMeaningfulSignalAt != meaningfulAt.Format(time.RFC3339) {
		t.Fatalf("last meaningful signal=%v, want %s", signals.LastMeaningfulSignalAt, meaningfulAt.Format(time.RFC3339))
	}
	if signals.LatestOutputActivityAt == nil || *signals.LatestOutputActivityAt != outputAt.Format(time.RFC3339) {
		t.Fatalf("latest output activity=%v, want %s", signals.LatestOutputActivityAt, outputAt.Format(time.RFC3339))
	}
	if signals.MeaningfulSignalAgeSeconds == nil {
		t.Fatalf("expected meaningful_signal_age_seconds")
	}
	if got, want := *signals.MeaningfulSignalAgeSeconds, int64(35*60); got != want {
		t.Fatalf("meaningful_signal_age_seconds=%d, want %d", got, want)
	}
	if !signals.AnalysisDriftRisk {
		t.Fatalf("expected analysis_drift_risk=true")
	}
	if !strings.Contains(signals.DriftReason, "no meaningful bus signal") {
		t.Fatalf("unexpected drift reason=%q", signals.DriftReason)
	}
}

func TestCollectTaskActivitySignals_NoMeaningfulSignalUsesRunStart(t *testing.T) {
	root := t.TempDir()
	projectID := "proj"
	taskID := "task-20260221-activity-bb"
	taskDir := filepath.Join(root, projectID, taskID)
	if err := os.MkdirAll(taskDir, 0o755); err != nil {
		t.Fatalf("mkdir task dir: %v", err)
	}

	busPath := filepath.Join(taskDir, "TASK-MESSAGE-BUS.md")
	start := time.Date(2026, 2, 21, 21, 0, 0, 0, time.UTC)
	now := start.Add(45 * time.Minute)
	writeTaskBusMessage(t, busPath, projectID, taskID, "PROGRESS", "analyzing more ideas", start.Add(10*time.Minute))

	signals := collectTaskActivitySignals(
		taskDir,
		"",
		"running",
		start,
		"",
		"",
		"",
		activityOptions{
			Enabled:    true,
			DriftAfter: 20 * time.Minute,
			Now: func() time.Time {
				return now
			},
		},
	)

	if signals.LastMeaningfulSignalAt != nil {
		t.Fatalf("expected no meaningful signal timestamp, got %v", *signals.LastMeaningfulSignalAt)
	}
	if signals.MeaningfulSignalAgeSeconds != nil {
		t.Fatalf("expected meaningful_signal_age_seconds=nil, got %d", *signals.MeaningfulSignalAgeSeconds)
	}
	if !signals.AnalysisDriftRisk {
		t.Fatalf("expected analysis_drift_risk=true")
	}
	if !strings.Contains(signals.DriftReason, "without meaningful bus signal") {
		t.Fatalf("unexpected drift reason=%q", signals.DriftReason)
	}
}

func TestCollectTaskActivitySignals_NotRunningNeverMarksDriftRisk(t *testing.T) {
	root := t.TempDir()
	taskDir := filepath.Join(root, "proj", "task-20260221-activity-cc")
	if err := os.MkdirAll(taskDir, 0o755); err != nil {
		t.Fatalf("mkdir task dir: %v", err)
	}

	now := time.Date(2026, 2, 21, 22, 0, 0, 0, time.UTC)
	signals := collectTaskActivitySignals(
		taskDir,
		"",
		"completed",
		time.Time{},
		"",
		"",
		"",
		activityOptions{
			Enabled:    true,
			DriftAfter: 5 * time.Minute,
			Now: func() time.Time {
				return now
			},
		},
	)

	if signals.AnalysisDriftRisk {
		t.Fatalf("expected analysis_drift_risk=false")
	}
	if !strings.Contains(signals.DriftReason, "not running") {
		t.Fatalf("unexpected drift reason=%q", signals.DriftReason)
	}
}
