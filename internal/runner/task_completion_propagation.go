package runner

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/jonnyzzz/conductor-loop/internal/messagebus"
	"github.com/jonnyzzz/conductor-loop/internal/storage"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v3"
)

const (
	taskCompletionPropagationStateFile       = "TASK-COMPLETE-FACT-PROPAGATION.yaml"
	taskCompletionPropagationLockFile        = "TASK-COMPLETE-FACT-PROPAGATION.lock"
	taskCompletionPropagationLockTimeout     = 5 * time.Second
	taskCompletionPropagationFactSampleLimit = 5
	taskCompletionPropagationParentLimit     = 20
	taskCompletionPropagationMetaKind        = "task_completion_propagation"
)

type taskCompletionPropagationResult struct {
	Posted           bool
	ProjectMessageID string
	PropagationKey   string
}

type taskCompletionPropagationState struct {
	Version            int       `yaml:"version"`
	LastPropagationKey string    `yaml:"last_propagation_key,omitempty"`
	LastProjectMessage string    `yaml:"last_project_message_id,omitempty"`
	LastPropagatedAt   time.Time `yaml:"last_propagated_at,omitempty"`
}

type taskCompletionRunSummary struct {
	Runs              []*storage.RunInfo
	RunIDs            []string
	MissingRunInfoIDs []string
	Latest            *storage.RunInfo
	LatestRunInfoPath string
	LatestOutputPath  string
	CompletedCount    int
	FailedCount       int
	RunningCount      int
}

type taskFactSignal struct {
	MsgID     string
	Timestamp time.Time
	Body      string
}

func propagateTaskCompletionToProject(rootDir, projectID, taskID, taskDir, taskBusPath string) (taskCompletionPropagationResult, error) {
	result := taskCompletionPropagationResult{}

	cleanRoot := filepath.Clean(strings.TrimSpace(rootDir))
	if cleanRoot == "." || cleanRoot == "" {
		return result, errors.New("root dir is empty")
	}
	cleanProjectID := strings.TrimSpace(projectID)
	if cleanProjectID == "" {
		return result, errors.New("project id is empty")
	}
	cleanTaskID := strings.TrimSpace(taskID)
	if cleanTaskID == "" {
		return result, errors.New("task id is empty")
	}
	cleanTaskDir := filepath.Clean(strings.TrimSpace(taskDir))
	if cleanTaskDir == "." || cleanTaskDir == "" {
		return result, errors.New("task dir is empty")
	}
	cleanTaskBusPath := strings.TrimSpace(taskBusPath)
	if cleanTaskBusPath == "" {
		cleanTaskBusPath = filepath.Join(cleanTaskDir, "TASK-MESSAGE-BUS.md")
	}

	donePath := filepath.Join(cleanTaskDir, "DONE")
	doneInfo, err := os.Stat(donePath)
	if err != nil {
		if os.IsNotExist(err) {
			return result, nil
		}
		return result, errors.Wrap(err, "stat DONE")
	}
	if doneInfo.IsDir() {
		return result, errors.New("DONE is a directory")
	}

	runSummary, err := loadTaskCompletionRunSummary(cleanTaskDir)
	if err != nil {
		return result, err
	}
	latestRunID := ""
	if runSummary.Latest != nil {
		latestRunID = strings.TrimSpace(runSummary.Latest.RunID)
	}

	taskFacts, latestRunEvent, err := readTaskFactSignals(cleanTaskBusPath, latestRunID)
	if err != nil {
		return result, err
	}

	propagationKey := buildTaskCompletionPropagationKey(doneInfo.ModTime(), runSummary, len(taskFacts))
	result.PropagationKey = propagationKey

	projectBusPath := filepath.Join(cleanRoot, cleanProjectID, "PROJECT-MESSAGE-BUS.md")
	if err := ensureDir(filepath.Dir(projectBusPath)); err != nil {
		return result, errors.Wrap(err, "ensure project dir")
	}
	projectBus, err := messagebus.NewMessageBus(projectBusPath)
	if err != nil {
		return result, errors.Wrap(err, "new project message bus")
	}

	statePath := filepath.Join(cleanTaskDir, taskCompletionPropagationStateFile)
	lockPath := filepath.Join(cleanTaskDir, taskCompletionPropagationLockFile)
	var posted bool
	var postedMessageID string
	err = withTaskCompletionPropagationLock(lockPath, func() error {
		state, err := readTaskCompletionPropagationState(statePath)
		if err != nil {
			return err
		}

		if state.LastPropagationKey == propagationKey {
			result.ProjectMessageID = state.LastProjectMessage
			posted = false
			return nil
		}

		existing, err := findExistingPropagationMessage(projectBus, cleanTaskID, propagationKey)
		if err != nil {
			return err
		}
		if existing != nil {
			state.LastPropagationKey = propagationKey
			state.LastProjectMessage = existing.MsgID
			state.LastPropagatedAt = existing.Timestamp.UTC()
			if err := writeTaskCompletionPropagationState(statePath, state); err != nil {
				return err
			}
			result.ProjectMessageID = existing.MsgID
			posted = false
			return nil
		}

		propagatedAt := time.Now().UTC()
		msg := buildTaskCompletionProjectMessage(taskCompletionProjectMessageParams{
			projectID:      cleanProjectID,
			taskID:         cleanTaskID,
			taskDir:        cleanTaskDir,
			taskBusPath:    cleanTaskBusPath,
			donePath:       donePath,
			doneModTime:    doneInfo.ModTime().UTC(),
			propagatedAt:   propagatedAt,
			propagationKey: propagationKey,
			runSummary:     runSummary,
			taskFacts:      taskFacts,
			latestRunEvent: latestRunEvent,
		})

		msgID, err := projectBus.AppendMessage(msg)
		if err != nil {
			return errors.Wrap(err, "append project FACT")
		}
		state.LastPropagationKey = propagationKey
		state.LastProjectMessage = msgID
		state.LastPropagatedAt = propagatedAt
		if err := writeTaskCompletionPropagationState(statePath, state); err != nil {
			return err
		}

		posted = true
		postedMessageID = msgID
		return nil
	})
	if err != nil {
		return result, err
	}

	result.Posted = posted
	if result.ProjectMessageID == "" {
		result.ProjectMessageID = postedMessageID
	}
	return result, nil
}

func loadTaskCompletionRunSummary(taskDir string) (taskCompletionRunSummary, error) {
	summary := taskCompletionRunSummary{
		Runs: make([]*storage.RunInfo, 0),
	}

	runsDir := filepath.Join(taskDir, "runs")
	entries, err := os.ReadDir(runsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return summary, nil
		}
		return summary, errors.Wrap(err, "read runs directory")
	}

	runIDs := make([]string, 0, len(entries))
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		runIDs = append(runIDs, entry.Name())
	}
	sort.Strings(runIDs)
	summary.RunIDs = append(summary.RunIDs, runIDs...)

	latestRunDir := ""
	for _, runID := range runIDs {
		infoPath := filepath.Join(runsDir, runID, "run-info.yaml")
		info, readErr := storage.ReadRunInfo(infoPath)
		if readErr != nil {
			summary.MissingRunInfoIDs = append(summary.MissingRunInfoIDs, runID)
			continue
		}
		if strings.TrimSpace(info.RunID) == "" {
			info.RunID = runID
		}
		summary.Runs = append(summary.Runs, info)

		switch strings.TrimSpace(info.Status) {
		case storage.StatusCompleted:
			summary.CompletedCount++
		case storage.StatusFailed:
			summary.FailedCount++
		case storage.StatusRunning:
			summary.RunningCount++
		}

		summary.Latest = info
		latestRunDir = runID
	}

	if summary.Latest != nil {
		if latestRunDir == "" {
			latestRunDir = strings.TrimSpace(summary.Latest.RunID)
		}
		summary.LatestRunInfoPath = filepath.Join(runsDir, latestRunDir, "run-info.yaml")
		latestOutput := strings.TrimSpace(summary.Latest.OutputPath)
		if latestOutput == "" {
			latestOutput = filepath.Join(runsDir, latestRunDir, "output.md")
		}
		summary.LatestOutputPath = latestOutput
	}

	return summary, nil
}

func readTaskFactSignals(taskBusPath, latestRunID string) ([]taskFactSignal, *messagebus.Message, error) {
	bus, err := messagebus.NewMessageBus(taskBusPath)
	if err != nil {
		return nil, nil, errors.Wrap(err, "new task message bus")
	}
	messages, err := bus.ReadMessages("")
	if err != nil {
		return nil, nil, errors.Wrap(err, "read task message bus")
	}

	facts := make([]taskFactSignal, 0)
	var latestRunEvent *messagebus.Message
	for _, msg := range messages {
		if msg == nil {
			continue
		}
		msgType := strings.TrimSpace(strings.ToUpper(msg.Type))
		if msgType == "FACT" {
			facts = append(facts, taskFactSignal{
				MsgID:     msg.MsgID,
				Timestamp: msg.Timestamp.UTC(),
				Body:      msg.Body,
			})
		}
		if latestRunID == "" || strings.TrimSpace(msg.RunID) != latestRunID {
			continue
		}
		if msgType == messagebus.EventTypeRunStop || msgType == messagebus.EventTypeRunCrash {
			latestRunEvent = msg
		}
	}
	return facts, latestRunEvent, nil
}

func buildTaskCompletionPropagationKey(doneMod time.Time, summary taskCompletionRunSummary, factCount int) string {
	var b strings.Builder
	b.WriteString(doneMod.UTC().Format(time.RFC3339Nano))
	b.WriteString("|fact_count=")
	b.WriteString(strconv.Itoa(factCount))

	for _, runID := range summary.RunIDs {
		b.WriteString("|run_id=")
		b.WriteString(runID)
	}
	for _, runID := range summary.MissingRunInfoIDs {
		b.WriteString("|missing_run_info=")
		b.WriteString(runID)
	}
	for _, info := range summary.Runs {
		if info == nil {
			continue
		}
		b.WriteString("|run=")
		b.WriteString(strings.TrimSpace(info.RunID))
		b.WriteString(",status=")
		b.WriteString(strings.TrimSpace(info.Status))
		b.WriteString(",exit=")
		b.WriteString(strconv.Itoa(info.ExitCode))
		b.WriteString(",start=")
		if !info.StartTime.IsZero() {
			b.WriteString(info.StartTime.UTC().Format(time.RFC3339Nano))
		}
		b.WriteString(",end=")
		if !info.EndTime.IsZero() {
			b.WriteString(info.EndTime.UTC().Format(time.RFC3339Nano))
		}
	}

	sum := sha256.Sum256([]byte(b.String()))
	return hex.EncodeToString(sum[:])
}

func withTaskCompletionPropagationLock(lockPath string, action func() error) error {
	if action == nil {
		return errors.New("propagation action is nil")
	}
	lockFile, err := os.OpenFile(lockPath, os.O_CREATE|os.O_RDWR, 0o644)
	if err != nil {
		return errors.Wrap(err, "open propagation lock")
	}
	defer lockFile.Close()

	if err := messagebus.LockExclusive(lockFile, taskCompletionPropagationLockTimeout); err != nil {
		return errors.Wrap(err, "acquire propagation lock")
	}
	defer func() {
		_ = messagebus.Unlock(lockFile)
	}()

	return action()
}

func readTaskCompletionPropagationState(path string) (*taskCompletionPropagationState, error) {
	state := &taskCompletionPropagationState{
		Version: 1,
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return state, nil
		}
		return nil, errors.Wrap(err, "read propagation state")
	}
	if len(strings.TrimSpace(string(data))) == 0 {
		return state, nil
	}
	if err := yaml.Unmarshal(data, state); err != nil {
		return nil, errors.Wrap(err, "unmarshal propagation state")
	}
	if state.Version == 0 {
		state.Version = 1
	}
	return state, nil
}

func writeTaskCompletionPropagationState(path string, state *taskCompletionPropagationState) error {
	if state == nil {
		return errors.New("propagation state is nil")
	}
	state.Version = 1
	data, err := yaml.Marshal(state)
	if err != nil {
		return errors.Wrap(err, "marshal propagation state")
	}
	if err := writeAtomicFile(path, data, 0o644); err != nil {
		return errors.Wrap(err, "write propagation state")
	}
	return nil
}

func writeAtomicFile(path string, data []byte, mode os.FileMode) error {
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, ".task-propagation-*.tmp")
	if err != nil {
		return errors.Wrap(err, "create temp state file")
	}
	tmpPath := tmp.Name()
	success := false
	defer func() {
		if success {
			return
		}
		_ = tmp.Close()
		_ = os.Remove(tmpPath)
	}()

	if _, err := tmp.Write(data); err != nil {
		return errors.Wrap(err, "write temp state file")
	}
	if err := tmp.Sync(); err != nil {
		return errors.Wrap(err, "sync temp state file")
	}
	if err := tmp.Chmod(mode); err != nil {
		return errors.Wrap(err, "chmod temp state file")
	}
	if err := tmp.Close(); err != nil {
		return errors.Wrap(err, "close temp state file")
	}
	if err := os.Rename(tmpPath, path); err != nil {
		return errors.Wrap(err, "rename temp state file")
	}
	success = true
	return nil
}

func findExistingPropagationMessage(projectBus *messagebus.MessageBus, taskID, propagationKey string) (*messagebus.Message, error) {
	if projectBus == nil {
		return nil, errors.New("project message bus is nil")
	}
	messages, err := projectBus.ReadMessages("")
	if err != nil {
		return nil, errors.Wrap(err, "read project message bus")
	}
	for i := len(messages) - 1; i >= 0; i-- {
		msg := messages[i]
		if msg == nil {
			continue
		}
		if strings.TrimSpace(strings.ToUpper(msg.Type)) != "FACT" {
			continue
		}
		if msg.Meta == nil {
			continue
		}
		if strings.TrimSpace(msg.Meta["kind"]) != taskCompletionPropagationMetaKind {
			continue
		}
		if strings.TrimSpace(msg.Meta["propagation_key"]) != propagationKey {
			continue
		}
		sourceTask := strings.TrimSpace(msg.Meta["source_task_id"])
		if sourceTask != "" && sourceTask != taskID {
			continue
		}
		return msg, nil
	}
	return nil, nil
}

type taskCompletionProjectMessageParams struct {
	projectID      string
	taskID         string
	taskDir        string
	taskBusPath    string
	donePath       string
	doneModTime    time.Time
	propagatedAt   time.Time
	propagationKey string
	runSummary     taskCompletionRunSummary
	taskFacts      []taskFactSignal
	latestRunEvent *messagebus.Message
}

func buildTaskCompletionProjectMessage(params taskCompletionProjectMessageParams) *messagebus.Message {
	meta := map[string]string{
		"kind":                  taskCompletionPropagationMetaKind,
		"propagation_key":       params.propagationKey,
		"source_project_id":     params.projectID,
		"source_task_id":        params.taskID,
		"source_task_dir":       params.taskDir,
		"source_task_bus_path":  params.taskBusPath,
		"source_done_path":      params.donePath,
		"source_done_mtime":     params.doneModTime.UTC().Format(time.RFC3339Nano),
		"propagated_at":         params.propagatedAt.UTC().Format(time.RFC3339Nano),
		"run_count":             strconv.Itoa(len(params.runSummary.RunIDs)),
		"completed_run_count":   strconv.Itoa(params.runSummary.CompletedCount),
		"failed_run_count":      strconv.Itoa(params.runSummary.FailedCount),
		"running_run_count":     strconv.Itoa(params.runSummary.RunningCount),
		"task_fact_count":       strconv.Itoa(len(params.taskFacts)),
		"source_run_ids":        strings.Join(params.runSummary.RunIDs, ","),
		"missing_run_info_ids":  strings.Join(params.runSummary.MissingRunInfoIDs, ","),
		"source_latest_run_id":  "",
		"source_latest_status":  "",
		"source_output_path":    "",
		"source_run_info_path":  "",
		"latest_run_event_id":   "",
		"latest_run_event_type": "",
	}

	runID := ""
	if params.runSummary.Latest != nil {
		runID = strings.TrimSpace(params.runSummary.Latest.RunID)
		meta["source_latest_run_id"] = runID
		meta["source_latest_status"] = strings.TrimSpace(params.runSummary.Latest.Status)
		meta["source_latest_exit_code"] = strconv.Itoa(params.runSummary.Latest.ExitCode)
		if !params.runSummary.Latest.StartTime.IsZero() {
			meta["source_latest_start_time"] = params.runSummary.Latest.StartTime.UTC().Format(time.RFC3339Nano)
		}
		if !params.runSummary.Latest.EndTime.IsZero() {
			meta["source_latest_end_time"] = params.runSummary.Latest.EndTime.UTC().Format(time.RFC3339Nano)
		}
	}
	if params.runSummary.LatestOutputPath != "" {
		meta["source_output_path"] = params.runSummary.LatestOutputPath
	}
	if params.runSummary.LatestRunInfoPath != "" {
		meta["source_run_info_path"] = params.runSummary.LatestRunInfoPath
	}
	if params.latestRunEvent != nil {
		meta["latest_run_event_id"] = params.latestRunEvent.MsgID
		meta["latest_run_event_type"] = strings.TrimSpace(params.latestRunEvent.Type)
		if !params.latestRunEvent.Timestamp.IsZero() {
			meta["latest_run_event_ts"] = params.latestRunEvent.Timestamp.UTC().Format(time.RFC3339Nano)
		}
	}

	links := []messagebus.Link{
		{
			URL:   params.taskBusPath,
			Label: "task message bus",
			Kind:  "task_bus",
		},
		{
			URL:   params.donePath,
			Label: "task DONE marker",
			Kind:  "task_done",
		},
	}
	if params.runSummary.LatestRunInfoPath != "" {
		links = append(links, messagebus.Link{
			URL:   params.runSummary.LatestRunInfoPath,
			Label: "latest run-info.yaml",
			Kind:  "run_info",
		})
	}
	if params.runSummary.LatestOutputPath != "" {
		links = append(links, messagebus.Link{
			URL:   params.runSummary.LatestOutputPath,
			Label: "latest output.md",
			Kind:  "output",
		})
	}

	parents := make([]messagebus.Parent, 0)
	seenParents := map[string]struct{}{}
	appendParent := func(msgID, kind string) {
		cleanMsgID := strings.TrimSpace(msgID)
		if cleanMsgID == "" {
			return
		}
		if _, exists := seenParents[cleanMsgID]; exists {
			return
		}
		seenParents[cleanMsgID] = struct{}{}
		parents = append(parents, messagebus.Parent{
			MsgID: cleanMsgID,
			Kind:  kind,
		})
	}
	if params.latestRunEvent != nil {
		appendParent(params.latestRunEvent.MsgID, "source_run_event")
	}
	start := 0
	if len(params.taskFacts) > taskCompletionPropagationParentLimit {
		start = len(params.taskFacts) - taskCompletionPropagationParentLimit
	}
	for _, fact := range params.taskFacts[start:] {
		appendParent(fact.MsgID, "source_fact")
	}

	return &messagebus.Message{
		Type:      "FACT",
		ProjectID: params.projectID,
		TaskID:    params.taskID,
		RunID:     runID,
		Parents:   parents,
		Links:     links,
		Meta:      meta,
		Body:      buildTaskCompletionBody(params),
	}
}

func buildTaskCompletionBody(params taskCompletionProjectMessageParams) string {
	var b strings.Builder
	fmt.Fprintf(&b, "task completion propagated to project scope\n\n")
	fmt.Fprintf(&b, "source_project_id: %s\n", params.projectID)
	fmt.Fprintf(&b, "source_task_id: %s\n", params.taskID)
	fmt.Fprintf(&b, "source_task_dir: %s\n", params.taskDir)
	fmt.Fprintf(&b, "source_task_bus: %s\n", params.taskBusPath)
	fmt.Fprintf(&b, "source_done_file: %s\n", params.donePath)
	fmt.Fprintf(&b, "source_done_mtime: %s\n", params.doneModTime.UTC().Format(time.RFC3339Nano))
	fmt.Fprintf(&b, "propagated_at: %s\n", params.propagatedAt.UTC().Format(time.RFC3339Nano))
	fmt.Fprintf(&b, "propagation_key: %s\n", params.propagationKey)

	fmt.Fprintf(&b, "\nrun_outcome_summary:\n")
	fmt.Fprintf(&b, "- run_count: %d\n", len(params.runSummary.RunIDs))
	fmt.Fprintf(&b, "- completed_runs: %d\n", params.runSummary.CompletedCount)
	fmt.Fprintf(&b, "- failed_runs: %d\n", params.runSummary.FailedCount)
	fmt.Fprintf(&b, "- running_runs: %d\n", params.runSummary.RunningCount)
	if len(params.runSummary.RunIDs) == 0 {
		fmt.Fprintf(&b, "- run_ids: (none)\n")
	} else {
		fmt.Fprintf(&b, "- run_ids: %s\n", strings.Join(params.runSummary.RunIDs, ", "))
	}
	if len(params.runSummary.MissingRunInfoIDs) > 0 {
		fmt.Fprintf(&b, "- missing_run_info_ids: %s\n", strings.Join(params.runSummary.MissingRunInfoIDs, ", "))
	}
	if params.runSummary.Latest != nil {
		fmt.Fprintf(&b, "- latest_run_id: %s\n", strings.TrimSpace(params.runSummary.Latest.RunID))
		fmt.Fprintf(&b, "- latest_status: %s\n", strings.TrimSpace(params.runSummary.Latest.Status))
		fmt.Fprintf(&b, "- latest_exit_code: %d\n", params.runSummary.Latest.ExitCode)
		if !params.runSummary.Latest.StartTime.IsZero() {
			fmt.Fprintf(&b, "- latest_start_time: %s\n", params.runSummary.Latest.StartTime.UTC().Format(time.RFC3339Nano))
		}
		if !params.runSummary.Latest.EndTime.IsZero() {
			fmt.Fprintf(&b, "- latest_end_time: %s\n", params.runSummary.Latest.EndTime.UTC().Format(time.RFC3339Nano))
		}
	}
	if params.runSummary.LatestRunInfoPath != "" {
		fmt.Fprintf(&b, "- latest_run_info_path: %s\n", params.runSummary.LatestRunInfoPath)
	}
	if params.runSummary.LatestOutputPath != "" {
		fmt.Fprintf(&b, "- latest_output_path: %s\n", params.runSummary.LatestOutputPath)
	}
	if params.latestRunEvent != nil {
		fmt.Fprintf(&b, "- latest_run_event_msg_id: %s\n", params.latestRunEvent.MsgID)
		fmt.Fprintf(&b, "- latest_run_event_type: %s\n", params.latestRunEvent.Type)
		if !params.latestRunEvent.Timestamp.IsZero() {
			fmt.Fprintf(&b, "- latest_run_event_ts: %s\n", params.latestRunEvent.Timestamp.UTC().Format(time.RFC3339Nano))
		}
	}

	fmt.Fprintf(&b, "\ntask_fact_signals:\n")
	fmt.Fprintf(&b, "- count: %d\n", len(params.taskFacts))
	if len(params.taskFacts) == 0 {
		fmt.Fprintf(&b, "- sample: (none)\n")
		return b.String()
	}
	latestFact := params.taskFacts[len(params.taskFacts)-1]
	fmt.Fprintf(&b, "- latest_fact_msg_id: %s\n", latestFact.MsgID)
	if !latestFact.Timestamp.IsZero() {
		fmt.Fprintf(&b, "- latest_fact_ts: %s\n", latestFact.Timestamp.UTC().Format(time.RFC3339Nano))
	}
	fmt.Fprintf(&b, "- sample:\n")
	for _, fact := range taskFactSamples(params.taskFacts, taskCompletionPropagationFactSampleLimit) {
		ts := ""
		if !fact.Timestamp.IsZero() {
			ts = fact.Timestamp.UTC().Format(time.RFC3339Nano)
		}
		fmt.Fprintf(&b, "  - [%s] %s %s\n", ts, fact.MsgID, singleLineSnippet(fact.Body, 160))
	}

	return b.String()
}

func taskFactSamples(facts []taskFactSignal, limit int) []taskFactSignal {
	if limit <= 0 || len(facts) <= limit {
		return facts
	}
	return facts[len(facts)-limit:]
}

func singleLineSnippet(body string, maxLen int) string {
	clean := strings.ReplaceAll(body, "\r\n", "\n")
	lines := strings.Split(clean, "\n")
	firstLine := ""
	if len(lines) > 0 {
		firstLine = strings.TrimSpace(lines[0])
	}
	firstLine = strings.Join(strings.Fields(firstLine), " ")
	if firstLine == "" {
		return "(empty)"
	}
	if maxLen <= 3 || len(firstLine) <= maxLen {
		return firstLine
	}
	return firstLine[:maxLen-3] + "..."
}
