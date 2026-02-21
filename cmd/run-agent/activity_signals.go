package main

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/jonnyzzz/conductor-loop/internal/messagebus"
	"github.com/jonnyzzz/conductor-loop/internal/storage"
)

const (
	defaultAnalysisDriftAfter = 20 * time.Minute
	busBodyPreviewLimit       = 96
)

type activityOptions struct {
	Enabled    bool
	DriftAfter time.Duration
	Now        func() time.Time
}

type activityBusMessage struct {
	Timestamp   string `json:"timestamp"`
	Type        string `json:"type"`
	BodyPreview string `json:"body_preview"`
}

type taskActivitySignals struct {
	LatestBusMessage           *activityBusMessage `json:"latest_bus_message"`
	LatestOutputActivityAt     *string             `json:"latest_output_activity_at"`
	LastMeaningfulSignalAt     *string             `json:"last_meaningful_signal_at"`
	MeaningfulSignalAgeSeconds *int64              `json:"meaningful_signal_age_seconds"`
	AnalysisDriftRisk          bool                `json:"analysis_drift_risk"`
	DriftThresholdSeconds      int64               `json:"drift_threshold_seconds"`
	DriftReason                string              `json:"drift_reason"`
}

var meaningfulBusTypes = map[string]struct{}{
	"FACT":     {},
	"DECISION": {},
	"ERROR":    {},
	"REVIEW":   {},
}

func (opts activityOptions) normalized() activityOptions {
	if opts.DriftAfter <= 0 {
		opts.DriftAfter = defaultAnalysisDriftAfter
	}
	if opts.Now == nil {
		opts.Now = time.Now
	}
	return opts
}

func collectTaskActivitySignals(taskDir, latestRunID, status string, latestRunStart time.Time, outputPath, stdoutPath, stderrPath string, opts activityOptions) taskActivitySignals {
	opts = opts.normalized()
	now := opts.Now().UTC()
	signals := taskActivitySignals{
		DriftThresholdSeconds: int64(opts.DriftAfter / time.Second),
		DriftReason:           "task is not running",
	}

	latestBus, lastMeaningfulSignal, busErr := readTaskBusSignals(taskDir)
	signals.LatestBusMessage = latestBus
	if lastMeaningfulSignal != nil {
		ts := lastMeaningfulSignal.UTC().Format(time.RFC3339)
		signals.LastMeaningfulSignalAt = stringPtr(ts)
		ageSec := int64(safeAge(now, *lastMeaningfulSignal) / time.Second)
		signals.MeaningfulSignalAgeSeconds = int64Ptr(ageSec)
	}

	latestOutputAt := latestOutputActivity(taskDir, latestRunID, outputPath, stdoutPath, stderrPath)
	if latestOutputAt != nil {
		ts := latestOutputAt.UTC().Format(time.RFC3339)
		signals.LatestOutputActivityAt = stringPtr(ts)
	}

	running := strings.EqualFold(strings.TrimSpace(status), storage.StatusRunning)
	if !running {
		if busErr != nil {
			signals.DriftReason = fmt.Sprintf("task is not running (message bus read failed: %v)", busErr)
		}
		return signals
	}

	signals.DriftReason = "running within drift threshold"

	if lastMeaningfulSignal != nil {
		age := safeAge(now, *lastMeaningfulSignal)
		if age > opts.DriftAfter {
			signals.AnalysisDriftRisk = true
			signals.DriftReason = fmt.Sprintf("no meaningful bus signal for %s", age.Round(time.Second))
		} else {
			signals.DriftReason = fmt.Sprintf("last meaningful bus signal %s ago", age.Round(time.Second))
		}
	} else if !latestRunStart.IsZero() {
		sinceStart := safeAge(now, latestRunStart)
		if sinceStart > opts.DriftAfter {
			signals.AnalysisDriftRisk = true
			signals.DriftReason = fmt.Sprintf("running for %s without meaningful bus signal", sinceStart.Round(time.Second))
		} else {
			signals.DriftReason = fmt.Sprintf("no meaningful bus signal yet (running for %s)", sinceStart.Round(time.Second))
		}
	} else {
		signals.DriftReason = "running with no meaningful bus signal"
	}

	if busErr != nil {
		signals.DriftReason = fmt.Sprintf("%s (message bus read failed: %v)", signals.DriftReason, busErr)
	}

	return signals
}

func readTaskBusSignals(taskDir string) (*activityBusMessage, *time.Time, error) {
	busPath := filepath.Join(taskDir, "TASK-MESSAGE-BUS.md")
	bus, err := messagebus.NewMessageBus(busPath)
	if err != nil {
		return nil, nil, err
	}

	msgs, err := bus.ReadMessages("")
	if err != nil {
		return nil, nil, err
	}
	if len(msgs) == 0 {
		return nil, nil, nil
	}

	var latest *messagebus.Message
	for i := len(msgs) - 1; i >= 0; i-- {
		if msgs[i] != nil {
			latest = msgs[i]
			break
		}
	}

	var latestSignal *activityBusMessage
	if latest != nil {
		latestSignal = &activityBusMessage{
			Timestamp:   activityTimestamp(latest.Timestamp),
			Type:        strings.TrimSpace(latest.Type),
			BodyPreview: compactBodyPreview(latest.Body, busBodyPreviewLimit),
		}
	}

	for i := len(msgs) - 1; i >= 0; i-- {
		msg := msgs[i]
		if msg == nil || !isMeaningfulBusType(msg.Type) || msg.Timestamp.IsZero() {
			continue
		}
		ts := msg.Timestamp.UTC()
		return latestSignal, &ts, nil
	}

	return latestSignal, nil, nil
}

func isMeaningfulBusType(msgType string) bool {
	_, ok := meaningfulBusTypes[strings.ToUpper(strings.TrimSpace(msgType))]
	return ok
}

func latestOutputActivity(taskDir, latestRunID, outputPath, stdoutPath, stderrPath string) *time.Time {
	candidates := outputSignalPaths(taskDir, latestRunID, outputPath, stdoutPath, stderrPath)
	var latest time.Time
	found := false
	for _, p := range candidates {
		info, err := os.Stat(p)
		if err != nil || info.IsDir() {
			continue
		}
		mod := info.ModTime().UTC()
		if !found || mod.After(latest) {
			latest = mod
			found = true
		}
	}
	if !found {
		return nil
	}
	return &latest
}

func outputSignalPaths(taskDir, latestRunID, outputPath, stdoutPath, stderrPath string) []string {
	runDir := ""
	if strings.TrimSpace(latestRunID) != "" {
		runDir = filepath.Join(taskDir, "runs", latestRunID)
	}
	candidates := []string{
		resolveOutputPath(outputPath, runDir),
		resolveOutputPath(stdoutPath, runDir),
		resolveOutputPath(stderrPath, runDir),
	}
	if runDir != "" {
		candidates = append(
			candidates,
			filepath.Join(runDir, "output.md"),
			filepath.Join(runDir, "agent-stdout.txt"),
			filepath.Join(runDir, "agent-stderr.txt"),
		)
	}
	seen := make(map[string]struct{}, len(candidates))
	paths := make([]string, 0, len(candidates))
	for _, p := range candidates {
		trimmed := strings.TrimSpace(p)
		if trimmed == "" {
			continue
		}
		clean := filepath.Clean(trimmed)
		if _, exists := seen[clean]; exists {
			continue
		}
		seen[clean] = struct{}{}
		paths = append(paths, clean)
	}
	return paths
}

func resolveOutputPath(path, runDir string) string {
	trimmed := strings.TrimSpace(path)
	if trimmed == "" {
		return ""
	}
	if filepath.IsAbs(trimmed) || strings.TrimSpace(runDir) == "" {
		return trimmed
	}
	return filepath.Join(runDir, trimmed)
}

func compactBodyPreview(body string, maxRunes int) string {
	compact := strings.Join(strings.Fields(strings.TrimSpace(body)), " ")
	if compact == "" || maxRunes <= 0 {
		return compact
	}
	runes := []rune(compact)
	if len(runes) <= maxRunes {
		return compact
	}
	if maxRunes <= 3 {
		return string(runes[:maxRunes])
	}
	return string(runes[:maxRunes-3]) + "..."
}

func activityTimestamp(ts time.Time) string {
	if ts.IsZero() {
		return ""
	}
	return ts.UTC().Format(time.RFC3339)
}

func formatActivityTimestamp(ts *string) string {
	if ts == nil {
		return "-"
	}
	value := strings.TrimSpace(*ts)
	if value == "" {
		return "-"
	}
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		return value
	}
	return parsed.Local().Format("2006-01-02 15:04")
}

func formatActivityAge(age *int64) string {
	if age == nil {
		return "-"
	}
	seconds := *age
	if seconds < 0 {
		seconds = 0
	}
	return (time.Duration(seconds) * time.Second).String()
}

func formatActivityBusSummary(msg *activityBusMessage) string {
	if msg == nil {
		return "-"
	}
	parts := make([]string, 0, 3)
	if ts := strings.TrimSpace(msg.Timestamp); ts != "" {
		parts = append(parts, ts)
	}
	if typ := strings.TrimSpace(msg.Type); typ != "" {
		parts = append(parts, typ)
	}
	if preview := strings.TrimSpace(msg.BodyPreview); preview != "" {
		parts = append(parts, preview)
	}
	if len(parts) == 0 {
		return "-"
	}
	return strings.Join(parts, " ")
}

func formatActivityRiskFlag(signals *taskActivitySignals) string {
	if signals == nil {
		return "-"
	}
	if signals.AnalysisDriftRisk {
		return "true"
	}
	return "false"
}

func formatActivityRiskText(signals *taskActivitySignals) string {
	if signals == nil {
		return "-"
	}
	if signals.AnalysisDriftRisk {
		return "risk"
	}
	return "ok"
}

func safeField(value string) string {
	clean := strings.Join(strings.Fields(strings.TrimSpace(value)), " ")
	if clean == "" {
		return "-"
	}
	return strings.ReplaceAll(clean, "\t", " ")
}

func safeAge(now, then time.Time) time.Duration {
	age := now.Sub(then)
	if age < 0 {
		return 0
	}
	return age
}

func stringPtr(value string) *string {
	v := value
	return &v
}

func int64Ptr(value int64) *int64 {
	v := value
	return &v
}
