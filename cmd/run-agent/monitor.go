package main

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"

	"github.com/jonnyzzz/conductor-loop/internal/config"
	"github.com/jonnyzzz/conductor-loop/internal/runner"
	"github.com/jonnyzzz/conductor-loop/internal/runstate"
	"github.com/jonnyzzz/conductor-loop/internal/storage"
	"github.com/spf13/cobra"
)

// todoTaskIDRegexp matches task IDs embedded anywhere in a line of text.
var todoTaskIDRegexp = regexp.MustCompile(`task-\d{8}-\d{6}-[a-z0-9][a-z0-9-]{1,48}[a-z0-9]`)

// todoEntry holds a parsed unchecked TODO item containing a task ID.
type todoEntry struct {
	TaskID string
	Text   string // full text after the "- [ ] " prefix
}

// parseTodoEntries reads a TODOs.md-style file and returns unchecked items
// that contain a valid task ID.
func parseTodoEntries(path string) ([]todoEntry, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open %s: %w", path, err)
	}
	defer f.Close()

	var entries []todoEntry
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		trimmed := strings.TrimLeft(line, " \t")
		if !strings.HasPrefix(trimmed, "- [ ] ") {
			continue
		}
		text := strings.TrimPrefix(trimmed, "- [ ] ")
		taskID := extractTaskIDFromText(text)
		if taskID == "" {
			continue
		}
		entries = append(entries, todoEntry{TaskID: taskID, Text: text})
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("scan %s: %w", path, err)
	}
	return entries, nil
}

// extractTaskIDFromText returns the first task ID found in text, or "" if none.
func extractTaskIDFromText(text string) string {
	return todoTaskIDRegexp.FindString(text)
}

// monitorTaskState holds the assessed state of a single task for the monitor.
type monitorTaskState struct {
	TaskID    string
	Exists    bool   // task directory exists
	HasRuns   bool   // at least one run directory exists
	Status    string // latest run status, or "-" if no runs
	Done      bool   // DONE marker is present
	PIDAlive  bool   // latest run process is alive
	IsStale   bool   // running and PID alive but output has been silent too long
	LatestRun string // name of the latest run directory
	Info      *storage.RunInfo
}

// assessMonitorTask reads the on-disk state of a task for the monitor.
func assessMonitorTask(root, projectID, taskID string, staleAfter time.Duration, now time.Time) monitorTaskState {
	state := monitorTaskState{TaskID: taskID, Status: "-"}
	taskDir := filepath.Join(root, projectID, taskID)

	if _, err := os.Stat(taskDir); err != nil {
		return state
	}
	state.Exists = true

	if _, err := os.Stat(filepath.Join(taskDir, "DONE")); err == nil {
		state.Done = true
	}

	runsDir := filepath.Join(taskDir, "runs")
	entries, err := os.ReadDir(runsDir)
	if err != nil {
		return state
	}
	var runNames []string
	for _, e := range entries {
		if e.IsDir() {
			runNames = append(runNames, e.Name())
		}
	}
	if len(runNames) == 0 {
		return state
	}
	state.HasRuns = true
	sort.Strings(runNames)
	latest := runNames[len(runNames)-1]
	state.LatestRun = latest

	infoPath := filepath.Join(runsDir, latest, "run-info.yaml")
	info, err := runstate.ReadRunInfo(infoPath)
	if err != nil {
		state.Status = "unknown"
		return state
	}
	state.Info = info
	state.Status = strings.TrimSpace(info.Status)
	state.PIDAlive = isRunPIDAlive(info)

	// Stale detection: running, PID alive, but output has been silent too long.
	if state.Status == storage.StatusRunning && state.PIDAlive && staleAfter > 0 {
		opts := activityOptions{
			Enabled:    true,
			DriftAfter: staleAfter,
			Now:        func() time.Time { return now },
		}
		signals := collectTaskActivitySignals(
			taskDir, latest, info.Status, info.StartTime,
			info.OutputPath, info.StdoutPath, info.StderrPath, opts,
		)
		state.IsStale = signals.AnalysisDriftRisk
	}

	return state
}

// monitorOutputNonEmpty returns true if the task's latest run has a non-empty output.md.
func monitorOutputNonEmpty(root, projectID, taskID, latestRun string, info *storage.RunInfo) bool {
	if info == nil || latestRun == "" {
		return false
	}
	runDir := filepath.Join(root, projectID, taskID, "runs", latestRun)
	// Prefer info.OutputPath; fall back to output.md in the run directory.
	outputPath := filepath.Join(runDir, "output.md")
	if p := strings.TrimSpace(info.OutputPath); p != "" {
		if filepath.IsAbs(p) {
			outputPath = p
		} else {
			outputPath = filepath.Join(runDir, p)
		}
	}
	data, err := os.ReadFile(outputPath)
	return err == nil && len(strings.TrimSpace(string(data))) > 0
}

const (
	monitorActionSkip     = "skip"
	monitorActionStart    = "start"
	monitorActionResume   = "resume"
	monitorActionRecover  = "recover"
	monitorActionFinalize = "finalize"
)

type monitorDecision struct {
	Action string
	Reason string
}

// decideMonitorAction determines what to do with a task based on its assessed state.
func decideMonitorAction(state monitorTaskState, root, projectID string) monitorDecision {
	return decideMonitorActionWithWindow(state, root, projectID, StopRequestedDefaultWindow)
}

// decideMonitorActionWithWindow is the testable variant that accepts an explicit
// stop-suppression window duration.
func decideMonitorActionWithWindow(state monitorTaskState, root, projectID string, stopWindow time.Duration) monitorDecision {
	switch {
	case state.Done:
		return monitorDecision{monitorActionSkip, "already DONE"}

	case state.Status == storage.StatusRunning && state.PIDAlive && state.IsStale:
		// Stale tasks need recovery regardless of stop-request (the process is alive
		// but unresponsive; recovering it is not a "restart" in the user-visible sense).
		return monitorDecision{monitorActionRecover, "running but output inactive beyond stale threshold"}

	case state.Status == storage.StatusRunning && state.PIDAlive:
		return monitorDecision{monitorActionSkip, "actively running"}

	case state.Status == storage.StatusRunning && !state.PIDAlive:
		// Recorded as running but the process is gone.
		if checkStopRequested(root, projectID, state.TaskID, stopWindow) {
			return monitorDecision{monitorActionSkip, "skip restart: stop-requested within window"}
		}
		return monitorDecision{monitorActionResume, "marked running but process is dead"}

	case state.Status == storage.StatusCompleted:
		if monitorOutputNonEmpty(root, projectID, state.TaskID, state.LatestRun, state.Info) {
			return monitorDecision{monitorActionFinalize, "completed with non-empty output"}
		}
		return monitorDecision{monitorActionSkip, "completed (output empty, not finalizing)"}

	case state.Status == storage.StatusFailed:
		if checkStopRequested(root, projectID, state.TaskID, stopWindow) {
			return monitorDecision{monitorActionSkip, "skip restart: stop-requested within window"}
		}
		return monitorDecision{monitorActionResume, "latest run failed"}

	case !state.Exists || !state.HasRuns:
		if checkStopRequested(root, projectID, state.TaskID, stopWindow) {
			return monitorDecision{monitorActionSkip, "skip restart: stop-requested within window"}
		}
		return monitorDecision{monitorActionStart, "task has no runs yet"}

	default:
		if checkStopRequested(root, projectID, state.TaskID, stopWindow) {
			return monitorDecision{monitorActionSkip, "skip restart: stop-requested within window"}
		}
		return monitorDecision{monitorActionStart, fmt.Sprintf("unexpected status %q", state.Status)}
	}
}

// monitorOpts holds all options for the monitor command.
type monitorOpts struct {
	RootDir    string
	ProjectID  string
	TODOFile   string
	Agent      string
	ConfigPath string
	WorkingDir string
	Interval   time.Duration
	StaleAfter time.Duration
	RateLimit  time.Duration
	DryRun     bool
	Once       bool
}

func newMonitorCmd() *cobra.Command {
	var opts monitorOpts

	cmd := &cobra.Command{
		Use:   "monitor",
		Short: "Monitor TODOs.md tasks: start, resume, recover stale, finalize completed",
		Long: `Monitor reads pending task IDs from unchecked items in a TODOs.md file and
manages their lifecycle automatically:

  start    — task has no runs yet; launches a new job
  resume   — latest run failed or process is dead; launches a new job
  recover  — running task whose output has been silent too long; stops then resumes
  finalize — completed task with non-empty output.md; creates the DONE marker
  skip     — task is already DONE or actively running

Use --once for a single monitoring pass and exit.
Omit --once (default) to run as a daemon polling every --interval.
Use --dry-run to see planned actions without executing them.`,

		RunE: func(cmd *cobra.Command, args []string) error {
			var rootErr error
			opts.RootDir, rootErr = config.ResolveRunsDir(opts.RootDir)
			if rootErr != nil {
				return fmt.Errorf("resolve runs dir: %w", rootErr)
			}
			if opts.TODOFile == "" {
				opts.TODOFile = "TODOs.md"
			}
			if opts.StaleAfter == 0 {
				opts.StaleAfter = defaultAnalysisDriftAfter
			}
			if opts.RateLimit == 0 {
				opts.RateLimit = 2 * time.Second
			}
			// Try to find a default config when no agent or config is specified.
			if opts.Agent == "" && opts.ConfigPath == "" && !opts.DryRun {
				if found, err := config.FindDefaultConfig(); err == nil && found != "" {
					opts.ConfigPath = found
				}
			}
			return runMonitor(cmd.OutOrStdout(), opts)
		},
	}

	cmd.Flags().StringVar(&opts.RootDir, "root", "", "run-agent root directory (default: ~/.run-agent/runs)")
	cmd.Flags().StringVar(&opts.ProjectID, "project", "", "project id (required)")
	cmd.Flags().StringVar(&opts.TODOFile, "todo", "TODOs.md", "path to TODOs.md file")
	cmd.Flags().StringVar(&opts.Agent, "agent", "", "agent type for starting/resuming tasks (e.g. claude)")
	cmd.Flags().StringVar(&opts.ConfigPath, "config", "", "config file path")
	cmd.Flags().StringVar(&opts.WorkingDir, "cwd", "", "working directory for agent jobs")
	cmd.Flags().DurationVar(&opts.Interval, "interval", 30*time.Second, "polling interval for daemon mode")
	cmd.Flags().DurationVar(&opts.StaleAfter, "stale-after", defaultAnalysisDriftAfter, "mark running task stale when output silent for this duration")
	cmd.Flags().DurationVar(&opts.RateLimit, "rate-limit", 2*time.Second, "minimum delay between successive start/resume actions")
	cmd.Flags().BoolVar(&opts.DryRun, "dry-run", false, "report planned actions without executing them")
	cmd.Flags().BoolVar(&opts.Once, "once", false, "run a single monitoring pass and exit (default is daemon mode)")
	_ = cmd.MarkFlagRequired("project")

	return cmd
}

// runMonitor is the entry point for the monitor command.
// In --once mode it performs one pass, waits for any started jobs, then returns.
// In daemon mode it loops on --interval until the process is killed.
func runMonitor(out io.Writer, opts monitorOpts) error {
	// Acquire a PID lockfile to enforce single monitor ownership per scope.
	// --once mode is exempt: single-pass callers don't need a long-lived lock.
	if !opts.Once && opts.Interval != 0 {
		release, err := acquireMonitorLock(opts.RootDir, opts.ProjectID)
		if err != nil {
			return err
		}
		defer release()
	}

	if opts.Once || opts.Interval == 0 {
		var wg sync.WaitGroup
		err := monitorPass(out, opts, &wg, time.Now())
		wg.Wait()
		return err
	}

	// Daemon: first pass immediately, then on every tick.
	var wg sync.WaitGroup
	ticker := time.NewTicker(opts.Interval)
	defer ticker.Stop()

	_ = monitorPass(out, opts, &wg, time.Now())
	for range ticker.C {
		_ = monitorPass(out, opts, &wg, time.Now())
	}
	return nil
}

// monitorPass runs one monitoring cycle: read todos, assess each task, take actions.
// Start/resume/recover actions dispatch goroutines tracked by wg.
func monitorPass(out io.Writer, opts monitorOpts, wg *sync.WaitGroup, now time.Time) error {
	todos, err := parseTodoEntries(opts.TODOFile)
	if err != nil {
		return fmt.Errorf("parse TODOs: %w", err)
	}
	if len(todos) == 0 {
		fmt.Fprintf(out, "monitor: no pending task IDs found in %s\n", opts.TODOFile)
		return nil
	}

	fmt.Fprintf(out, "monitor: checking %d pending task(s) from %s\n", len(todos), opts.TODOFile)

	firstAction := true
	for _, todo := range todos {
		state := assessMonitorTask(opts.RootDir, opts.ProjectID, todo.TaskID, opts.StaleAfter, now)
		decision := decideMonitorAction(state, opts.RootDir, opts.ProjectID)

		fmt.Fprintf(out, "  [%-8s] %s — %s\n", decision.Action, todo.TaskID, decision.Reason)

		if decision.Action == monitorActionSkip {
			continue
		}

		// Rate-limit successive task-triggering actions (applies in dry-run too,
		// so the output timing reflects what live mode would experience).
		if !firstAction && opts.RateLimit > 0 {
			time.Sleep(opts.RateLimit)
		}
		firstAction = false

		if opts.DryRun {
			continue
		}

		taskDir := filepath.Join(opts.RootDir, opts.ProjectID, todo.TaskID)

		switch decision.Action {
		case monitorActionFinalize:
			doneFile := filepath.Join(taskDir, "DONE")
			if err := os.WriteFile(doneFile, []byte(""), 0o644); err != nil {
				fmt.Fprintf(out, "  [ERROR] finalize %s: %v\n", todo.TaskID, err)
			} else {
				if err := updateTodoFile(opts.TODOFile, todo.TaskID); err != nil {
					fmt.Fprintf(out, "  [WARN]  update TODOs.md for %s: %v\n", todo.TaskID, err)
				}
				fmt.Fprintf(out, "  [OK]    finalized %s (DONE created)\n", todo.TaskID)
			}

		case monitorActionRecover:
			// Stop the stale/dead task, then start a new run.
			if stopErr := runStop("", opts.RootDir, opts.ProjectID, todo.TaskID, "", true); stopErr != nil {
				fmt.Fprintf(out, "  [WARN]  stop %s: %v\n", todo.TaskID, stopErr)
			} else {
				fmt.Fprintf(out, "  [OK]    stopped stale task %s\n", todo.TaskID)
			}
			monitorLaunchJob(out, opts, todo, wg)

		case monitorActionStart, monitorActionResume:
			// Ensure the task directory and TASK.md exist before launching.
			if mkErr := os.MkdirAll(taskDir, 0o755); mkErr != nil {
				fmt.Fprintf(out, "  [ERROR] mkdir %s: %v\n", todo.TaskID, mkErr)
				continue
			}
			taskMD := filepath.Join(taskDir, "TASK.md")
			if _, statErr := os.Stat(taskMD); os.IsNotExist(statErr) {
				if writeErr := os.WriteFile(taskMD, []byte(todo.Text+"\n"), 0o644); writeErr != nil {
					fmt.Fprintf(out, "  [ERROR] write TASK.md for %s: %v\n", todo.TaskID, writeErr)
					continue
				}
			}
			monitorLaunchJob(out, opts, todo, wg)
		}
	}

	return nil
}

// monitorLaunchJob removes any DONE marker then starts a new job in a goroutine.
func monitorLaunchJob(out io.Writer, opts monitorOpts, todo todoEntry, wg *sync.WaitGroup) {
	if opts.Agent == "" && opts.ConfigPath == "" {
		fmt.Fprintf(out, "  [SKIP]  %s: --agent required to start/resume tasks\n", todo.TaskID)
		return
	}

	taskID := todo.TaskID
	jobOpts := runner.JobOptions{
		RootDir:    opts.RootDir,
		Agent:      opts.Agent,
		ConfigPath: opts.ConfigPath,
		WorkingDir: opts.WorkingDir,
	}

	// Remove DONE marker so the run can proceed.
	doneFile := filepath.Join(opts.RootDir, opts.ProjectID, taskID, "DONE")
	_ = os.Remove(doneFile)
	// Remove any STOP-REQUESTED marker: the monitor has decided to (re)start
	// this task, so the suppression window is no longer relevant.
	_ = removeStopRequest(opts.RootDir, opts.ProjectID, taskID)

	wg.Add(1)
	go func() {
		defer wg.Done()
		if err := runner.RunJob(opts.ProjectID, taskID, jobOpts); err != nil {
			fmt.Fprintf(out, "  [ERROR] run %s: %v\n", taskID, err)
		} else {
			fmt.Fprintf(out, "  [OK]    run completed for %s\n", taskID)
		}
	}()
	fmt.Fprintf(out, "  [OK]    launched job for %s\n", taskID)
}

// updateTodoFile marks a task as done in the TODOs file.
func updateTodoFile(path, taskID string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	lines := strings.Split(string(data), "\n")
	found := false
	for i, line := range lines {
		if strings.Contains(line, taskID) && strings.Contains(line, "- [ ] ") {
			// Careful not to replace arbitrary text, ensure we replace the checkbox prefix.
			// But the line might be indented or have other content.
			// Simple replace of "- [ ] " with "- [x] " is safest given the context.
			lines[i] = strings.Replace(line, "- [ ] ", "- [x] ", 1)
			found = true
			break
		}
	}
	if !found {
		return nil
	}
	output := strings.Join(lines, "\n")
	// Use 0644 or preserve original mode if possible, but 0644 is standard.
	return os.WriteFile(path, []byte(output), 0o644)
}
