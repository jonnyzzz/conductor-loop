package runner

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/jonnyzzz/conductor-loop/internal/agent"
	"github.com/jonnyzzz/conductor-loop/internal/agent/claude"
	"github.com/jonnyzzz/conductor-loop/internal/agent/codex"
	"github.com/jonnyzzz/conductor-loop/internal/agent/gemini"
	"github.com/jonnyzzz/conductor-loop/internal/agent/perplexity"
	"github.com/jonnyzzz/conductor-loop/internal/agent/xai"
	"github.com/jonnyzzz/conductor-loop/internal/messagebus"
	"github.com/jonnyzzz/conductor-loop/internal/storage"
	"github.com/jonnyzzz/conductor-loop/internal/webhook"
	"github.com/pkg/errors"
)

// JobOptions controls execution for a single run-agent job.
type JobOptions struct {
	RootDir            string
	ConfigPath         string
	Agent              string
	Prompt             string
	PromptPath         string
	WorkingDir         string
	MessageBusPath     string
	ParentRunID        string
	PreviousRunID      string
	Environment        map[string]string
	PreallocatedRunDir string        // optional: pre-created run directory; skip createRunDir if set
	Timeout            time.Duration // idle output timeout for CLI agents; 0 means no limit
	ConductorURL       string        // e.g. "http://127.0.0.1:14355"; if empty, derived from config
}

// RunJob starts a single agent run and waits for completion.
func RunJob(projectID, taskID string, opts JobOptions) error {
	_, err := runJob(projectID, taskID, opts)
	return err
}

func runJob(projectID, taskID string, opts JobOptions) (*storage.RunInfo, error) {
	rootDir, err := resolveRootDir(opts.RootDir)
	if err != nil {
		return nil, err
	}
	taskDir, err := resolveTaskDir(rootDir, projectID, taskID)
	if err != nil {
		return nil, err
	}
	if err := ensureDir(taskDir); err != nil {
		return nil, errors.Wrap(err, "ensure task dir")
	}

	promptText, err := resolvePrompt(opts)
	if err != nil {
		return nil, err
	}
	taskMDPath := filepath.Join(taskDir, "TASK.md")
	if _, statErr := os.Stat(taskMDPath); statErr != nil {
		if !os.IsNotExist(statErr) {
			return nil, errors.Wrap(statErr, "stat TASK.md")
		}
		if writeErr := os.WriteFile(taskMDPath, []byte(promptText+"\n"), 0o644); writeErr != nil {
			return nil, errors.Wrap(writeErr, "write TASK.md")
		}
	}

	workingDir := strings.TrimSpace(opts.WorkingDir)
	if workingDir == "" {
		workingDir = taskDir
	}
	workingDir, err = absPath(workingDir)
	if err != nil {
		return nil, errors.Wrap(err, "resolve working dir")
	}

	busPath := strings.TrimSpace(opts.MessageBusPath)
	if busPath == "" {
		busPath = filepath.Join(taskDir, "TASK-MESSAGE-BUS.md")
	}

	runsDir := filepath.Join(taskDir, "runs")
	if err := ensureDir(runsDir); err != nil {
		return nil, errors.Wrap(err, "ensure runs dir")
	}

	// Load config to read defaults (MaxConcurrentRuns, agent, etc.).
	cfg, err := loadConfig(opts.ConfigPath)
	if err != nil {
		return nil, err
	}

	selection, err := selectAgent(cfg, opts.Agent)
	if err != nil {
		return nil, err
	}
	agentType := strings.ToLower(strings.TrimSpace(selection.Type))
	if agentType == "" {
		agentType = strings.ToLower(strings.TrimSpace(opts.Agent))
	}
	if agentType == "" {
		return nil, errors.New("agent type is empty")
	}
	if tokenErr := ValidateToken(agentType, selection.Config.Token); tokenErr != nil {
		log.Printf("warning: %v", tokenErr)
	}

	parentRunID := strings.TrimSpace(opts.ParentRunID)

	// Create the run directory and write a sentinel run-info.yaml BEFORE
	// detectAgentVersion. detectAgentVersion spawns a subprocess that can take
	// ~100ms; without the sentinel, FindActiveChildren would not see this child
	// run during that window, causing the parent's RunTask to return early.
	var runID, runDir string
	if preallocated := strings.TrimSpace(opts.PreallocatedRunDir); preallocated != "" {
		runDir = preallocated
		runID = filepath.Base(runDir)
	} else {
		var allocErr error
		runID, runDir, allocErr = createRunDir(runsDir)
		if allocErr != nil {
			return nil, allocErr
		}
	}
	if parentRunID != "" {
		selfPID := os.Getpid()
		selfPGID, pgidErr := ProcessGroupID(selfPID)
		if pgidErr != nil {
			selfPGID = selfPID
		}
		runDirAbsEarly, _ := absPath(runDir)
		sentinel := &storage.RunInfo{
			Version:          1,
			RunID:            runID,
			ParentRunID:      parentRunID,
			PreviousRunID:    strings.TrimSpace(opts.PreviousRunID),
			ProjectID:        projectID,
			TaskID:           taskID,
			AgentType:        agentType,
			ProcessOwnership: storage.ProcessOwnershipManaged,
			StartTime:        time.Now().UTC(),
			ExitCode:         -1,
			Status:           storage.StatusRunning,
			CWD:              workingDir,
			PID:              selfPID,
			PGID:             selfPGID,
			PromptPath:       filepath.Join(runDirAbsEarly, "prompt.md"),
			OutputPath:       filepath.Join(runDirAbsEarly, "output.md"),
			StdoutPath:       filepath.Join(runDirAbsEarly, "agent-stdout.txt"),
			StderrPath:       filepath.Join(runDirAbsEarly, "agent-stderr.txt"),
		}
		_ = storage.WriteRunInfo(filepath.Join(runDir, "run-info.yaml"), sentinel)
	}

	// detectAgentVersion spawns an external process and is best-effort.
	agentVersion := detectAgentVersion(context.Background(), agentType)

	restAgent := isRestAgent(agentType)
	ctx := context.Background()
	if restAgent && opts.Timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(context.Background(), opts.Timeout)
		defer cancel()
	}

	// Initialize the concurrency semaphore from config (no-op after the first
	// call) and acquire a slot. Blocks if the limit is reached.
	maxConcurrent := 0
	if cfg != nil {
		maxConcurrent = cfg.Defaults.MaxConcurrentRuns
	}
	initSemaphore(maxConcurrent)
	if err := acquireSem(ctx); err != nil {
		return nil, fmt.Errorf("acquire run slot: %w", err)
	}
	defer releaseSem()

	// Derive ConductorURL from opts or fall back to config.
	conductorURL := strings.TrimSpace(opts.ConductorURL)
	if conductorURL == "" && cfg != nil && cfg.API.Port > 0 {
		host := strings.TrimSpace(cfg.API.Host)
		if host == "" || host == "0.0.0.0" {
			host = "127.0.0.1"
		}
		conductorURL = fmt.Sprintf("http://%s:%d", host, cfg.API.Port)
	}

	// RepoRoot is the parent of the runs root directory.
	repoRoot := filepath.Dir(rootDir)

	promptPath := filepath.Join(runDir, "prompt.md")
	promptContent := buildPrompt(PromptParams{
		TaskDir:        taskDir,
		RunDir:         runDir,
		ProjectID:      projectID,
		TaskID:         taskID,
		RunID:          runID,
		ParentRunID:    parentRunID,
		MessageBusPath: busPath,
		ConductorURL:   conductorURL,
		RepoRoot:       repoRoot,
	}, promptText)
	if err := os.WriteFile(promptPath, []byte(promptContent), 0o644); err != nil {
		return nil, errors.Wrap(err, "write prompt")
	}

	var notifier *webhook.Notifier
	if cfg != nil {
		notifier = webhook.NewNotifier(cfg.Webhook)
	}

	warnJRunEnvMismatch(projectID, taskID, runID, parentRunID)

	envOverrides := map[string]string{
		"JRUN_PROJECT_ID": projectID,
		"JRUN_TASK_ID":    taskID,
		"JRUN_ID":         runID,
		"JRUN_PARENT_ID":  parentRunID,
		"RUNS_DIR":        runsDir,
		"MESSAGE_BUS":     busPath,
		"TASK_FOLDER":     taskDir,
		"RUN_FOLDER":      runDir,
	}
	if conductorURL != "" {
		envOverrides["CONDUCTOR_URL"] = conductorURL
	}
	if tokenVar := tokenEnvVar(agentType); tokenVar != "" {
		if token := strings.TrimSpace(selection.Config.Token); token != "" {
			envOverrides[tokenVar] = token
		}
	}
	if err := prependPath(envOverrides); err != nil {
		return nil, err
	}
	for key, value := range opts.Environment {
		if strings.TrimSpace(key) == "" {
			continue
		}
		envOverrides[key] = value
	}

	env := mergeEnv(os.Environ(), envOverrides)
	env = removeEnvKeys(env, "CLAUDECODE")

	runDirAbs, err := absPath(runDir)
	if err != nil {
		return nil, errors.Wrap(err, "resolve run dir")
	}
	promptPathAbs := filepath.Join(runDirAbs, "prompt.md")
	outputPathAbs := filepath.Join(runDirAbs, "output.md")
	stdoutPathAbs := filepath.Join(runDirAbs, "agent-stdout.txt")
	stderrPathAbs := filepath.Join(runDirAbs, "agent-stderr.txt")

	info := &storage.RunInfo{
		Version:          1,
		RunID:            runID,
		ParentRunID:      parentRunID,
		PreviousRunID:    strings.TrimSpace(opts.PreviousRunID),
		ProjectID:        projectID,
		TaskID:           taskID,
		AgentType:        agentType,
		ProcessOwnership: storage.ProcessOwnershipManaged,
		AgentVersion:     agentVersion,
		StartTime:        time.Now().UTC(),
		ExitCode:         -1,
		Status:           storage.StatusRunning,
		CWD:              workingDir,
		PromptPath:       promptPathAbs,
		OutputPath:       outputPathAbs,
		StdoutPath:       stdoutPathAbs,
		StderrPath:       stderrPathAbs,
	}

	timedOut := false
	var execErr error
	if restAgent {
		execErr = executeREST(ctx, agentType, selection, promptContent, workingDir, env, runDir, busPath, info)
		timedOut = opts.Timeout > 0 && ctx.Err() == context.DeadlineExceeded
	} else {
		timedOut, execErr = executeCLI(ctx, agentType, promptPathAbs, workingDir, env, runDir, busPath, info, opts.Timeout)
	}

	if timedOut {
		timeoutBody := fmt.Sprintf("agent job timed out after %s", opts.Timeout)
		if !restAgent {
			timeoutBody = fmt.Sprintf("agent job timed out after %s of idle output", opts.Timeout)
		}
		_ = postRunEvent(busPath, info, "WARN", timeoutBody)
	}

	// Send webhook notification asynchronously (non-blocking; failures are logged to message bus).
	if notifier != nil {
		payload := webhook.RunStopPayload{
			Event:           "run_stop",
			ProjectID:       info.ProjectID,
			TaskID:          info.TaskID,
			RunID:           info.RunID,
			AgentType:       info.AgentType,
			Status:          info.Status,
			ExitCode:        info.ExitCode,
			StartedAt:       info.StartTime,
			StoppedAt:       info.EndTime,
			DurationSeconds: info.EndTime.Sub(info.StartTime).Seconds(),
			ErrorSummary:    info.ErrorSummary,
		}
		notifier.SendRunStop(payload, func(err error) {
			_ = postRunEvent(busPath, info, "WARN", fmt.Sprintf("webhook delivery failed: %v", err))
		})
	}

	if execErr != nil {
		return info, execErr
	}
	return info, nil
}

func resolvePrompt(opts JobOptions) (string, error) {
	if path := strings.TrimSpace(opts.PromptPath); path != "" {
		return readFileTrimmed(path)
	}
	if prompt := strings.TrimSpace(opts.Prompt); prompt != "" {
		return prompt, nil
	}
	return "", errors.New("prompt is empty")
}

func isRestAgent(agentType string) bool {
	switch strings.ToLower(agentType) {
	case "perplexity", "xai":
		return true
	default:
		return false
	}
}

func executeCLI(ctx context.Context, agentType, promptPath, workingDir string, env []string, runDir, busPath string, info *storage.RunInfo, idleOutputTimeout time.Duration) (bool, error) {
	command, args, err := commandForAgent(agentType)
	if err != nil {
		return false, err
	}
	promptFile, err := os.Open(promptPath)
	if err != nil {
		return false, errors.Wrap(err, "open prompt")
	}
	pm, err := NewProcessManager(runDir)
	if err != nil {
		_ = promptFile.Close()
		return false, err
	}
	processCtx, processCancel := context.WithCancel(ctx)
	defer processCancel()

	proc, err := pm.SpawnAgent(processCtx, agentType, SpawnOptions{
		Command: command,
		Args:    args,
		Dir:     workingDir,
		Env:     env,
		Stdin:   promptFile,
	})
	_ = promptFile.Close()
	if err != nil {
		pid := os.Getpid()
		pgid := pid
		if resolved, resolveErr := ProcessGroupID(pid); resolveErr == nil {
			pgid = resolved
		}
		info.PID = pid
		info.PGID = pgid
		info.EndTime = time.Now().UTC()
		info.ExitCode = -1
		info.Status = storage.StatusFailed
		if writeErr := storage.WriteRunInfo(filepath.Join(runDir, "run-info.yaml"), info); writeErr != nil {
			return false, errors.Wrap(writeErr, "write run-info")
		}
		return false, errors.Wrap(err, "spawn agent")
	}
	info.PID = proc.PID
	info.PGID = proc.PGID
	info.CommandLine = fmt.Sprintf("%s %s < %s", command, strings.Join(args, " "), promptPath)
	if err := storage.WriteRunInfo(filepath.Join(runDir, "run-info.yaml"), info); err != nil {
		_ = proc.Cmd.Process.Kill()
		_ = proc.Wait()
		return false, errors.Wrap(err, "write run-info")
	}
	startBody := fmt.Sprintf("run started\nrun_dir: %s\nprompt: %s\nstdout: %s\nstderr: %s\noutput: %s",
		runDir,
		info.PromptPath,
		info.StdoutPath,
		info.StderrPath,
		info.OutputPath,
	)
	if err := postRunEvent(busPath, info, messagebus.EventTypeRunStart, startBody); err != nil {
		_ = proc.Cmd.Process.Kill()
		_ = proc.Wait()
		return false, err
	}

	waitErr, idleTimedOut := waitForProcessWithIdleOutputTimeout(processCtx, processCancel, proc, idleOutputTimeout)
	exitCode := 0
	if proc.Cmd.ProcessState != nil {
		exitCode = proc.Cmd.ProcessState.ExitCode()
	}
	info.ExitCode = exitCode
	info.EndTime = time.Now().UTC()
	if exitCode == 0 && waitErr == nil {
		info.Status = storage.StatusCompleted
	} else {
		info.Status = storage.StatusFailed
	}
	if info.Status == storage.StatusFailed {
		if idleTimedOut {
			info.ErrorSummary = "timed out"
		} else {
			info.ErrorSummary = classifyExitCode(exitCode)
		}
	}
	if err := storage.UpdateRunInfo(filepath.Join(runDir, "run-info.yaml"), func(update *storage.RunInfo) error {
		update.ExitCode = info.ExitCode
		update.EndTime = info.EndTime
		update.Status = info.Status
		update.ErrorSummary = info.ErrorSummary
		return nil
	}); err != nil {
		return idleTimedOut, errors.Wrap(err, "update run-info")
	}
	// For stream-json CLI agents: extract clean text from JSON stream before
	// falling back to raw copy.
	switch strings.ToLower(agentType) {
	case "claude":
		if parseErr := claude.WriteOutputMDFromStream(runDir, info.StdoutPath); parseErr != nil {
			log.Printf("JSONL parse for output.md failed (writing placeholder): %v", parseErr)
			placeholder := "# Agent Output\n\n*The agent did not write output.md. Raw output is available in the stdout tab.*\n"
			_ = os.WriteFile(filepath.Join(runDir, "output.md"), []byte(placeholder), 0o644)
		}
	case "codex":
		_ = codex.WriteOutputMDFromStream(runDir, info.StdoutPath)
	case "gemini":
		_ = gemini.WriteOutputMDFromStream(runDir, info.StdoutPath)
	}
	if _, err := agent.CreateOutputMD(runDir, ""); err != nil {
		return idleTimedOut, errors.Wrap(err, "ensure output.md")
	}
	stopBody := fmt.Sprintf("run stopped with code %d\nrun_dir: %s\noutput: %s",
		info.ExitCode,
		runDir,
		info.OutputPath,
	)
	if info.Status == storage.StatusFailed {
		if excerpt := tailFile(info.StderrPath, 50); excerpt != "" {
			stopBody += "\n\n## stderr (last 50 lines)\n" + excerpt
		}
	}
	stopEvent := messagebus.EventTypeRunStop
	if exitCode != 0 {
		stopEvent = messagebus.EventTypeRunCrash
	}
	if err := postRunEvent(busPath, info, stopEvent, stopBody); err != nil {
		return idleTimedOut, err
	}
	if waitErr != nil || exitCode != 0 {
		return idleTimedOut, errors.Wrap(waitErr, "agent execution failed")
	}
	return idleTimedOut, nil
}

func waitForProcessWithIdleOutputTimeout(ctx context.Context, cancel context.CancelFunc, proc *Process, idleOutputTimeout time.Duration) (error, bool) {
	waitDone := make(chan error, 1)
	go func() {
		waitDone <- proc.Wait()
	}()

	if idleOutputTimeout <= 0 {
		return <-waitDone, false
	}

	pollInterval := idleOutputTimeout / 4
	if pollInterval < 50*time.Millisecond {
		pollInterval = 50 * time.Millisecond
	}
	if pollInterval > 250*time.Millisecond {
		pollInterval = 250 * time.Millisecond
	}
	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()

	lastStdoutSize := fileSize(proc.StdoutPath)
	lastStderrSize := fileSize(proc.StderrPath)
	lastOutputAt := time.Now()
	for {
		select {
		case err := <-waitDone:
			return err, false
		case <-ticker.C:
			currentStdoutSize := fileSize(proc.StdoutPath)
			currentStderrSize := fileSize(proc.StderrPath)
			if currentStdoutSize > lastStdoutSize || currentStderrSize > lastStderrSize {
				lastOutputAt = time.Now()
				if currentStdoutSize > lastStdoutSize {
					lastStdoutSize = currentStdoutSize
				}
				if currentStderrSize > lastStderrSize {
					lastStderrSize = currentStderrSize
				}
				continue
			}
			if time.Since(lastOutputAt) >= idleOutputTimeout {
				cancel()
				return <-waitDone, true
			}
		case <-ctx.Done():
			return <-waitDone, false
		}
	}
}

func fileSize(path string) int64 {
	if strings.TrimSpace(path) == "" {
		return -1
	}
	stat, err := os.Stat(path)
	if err != nil {
		return -1
	}
	return stat.Size()
}

func executeREST(ctx context.Context, agentType string, selection agentSelection, promptContent, workingDir string, env []string, runDir, busPath string, info *storage.RunInfo) error {
	pid := os.Getpid()
	pgid := pid
	if resolved, err := ProcessGroupID(pid); err == nil {
		pgid = resolved
	}
	info.PID = pid
	info.PGID = pgid
	if err := storage.WriteRunInfo(filepath.Join(runDir, "run-info.yaml"), info); err != nil {
		return errors.Wrap(err, "write run-info")
	}
	startBody := fmt.Sprintf("run started\nrun_dir: %s\nprompt: %s\nstdout: %s\nstderr: %s\noutput: %s",
		runDir,
		info.PromptPath,
		info.StdoutPath,
		info.StderrPath,
		info.OutputPath,
	)
	if err := postRunEvent(busPath, info, messagebus.EventTypeRunStart, startBody); err != nil {
		return err
	}

	var execErr error
	runCtx := &agent.RunContext{
		RunID:       info.RunID,
		ProjectID:   info.ProjectID,
		TaskID:      info.TaskID,
		Prompt:      promptContent,
		WorkingDir:  workingDir,
		StdoutPath:  info.StdoutPath,
		StderrPath:  info.StderrPath,
		Environment: envMap(env),
	}
	switch strings.ToLower(agentType) {
	case "perplexity":
		agentImpl := perplexity.NewPerplexityAgent(perplexity.Options{
			Token:       selection.Config.Token,
			Model:       selection.Config.Model,
			APIEndpoint: selection.Config.BaseURL,
		})
		execErr = agentImpl.Execute(ctx, runCtx)
	case "xai":
		agentImpl, err := xai.NewAgent(xai.Config{
			APIKey:  selection.Config.Token,
			BaseURL: selection.Config.BaseURL,
			Model:   selection.Config.Model,
		})
		if err != nil {
			return err
		}
		execErr = agentImpl.Execute(ctx, runCtx)
	default:
		return fmt.Errorf("unsupported rest agent %q", agentType)
	}
	return finalizeRun(runDir, busPath, info, execErr)
}

func finalizeRun(runDir, busPath string, info *storage.RunInfo, execErr error) error {
	if info == nil {
		return errors.New("run info is nil")
	}
	info.EndTime = time.Now().UTC()
	if execErr != nil {
		info.ExitCode = 1
		info.Status = storage.StatusFailed
		errMsg := execErr.Error()
		if len(errMsg) > 200 {
			errMsg = errMsg[:200]
		}
		info.ErrorSummary = errMsg
	} else {
		info.ExitCode = 0
		info.Status = storage.StatusCompleted
	}
	if err := storage.UpdateRunInfo(filepath.Join(runDir, "run-info.yaml"), func(update *storage.RunInfo) error {
		update.ExitCode = info.ExitCode
		update.EndTime = info.EndTime
		update.Status = info.Status
		update.ErrorSummary = info.ErrorSummary
		return nil
	}); err != nil {
		return errors.Wrap(err, "update run-info")
	}
	if _, err := agent.CreateOutputMD(runDir, ""); err != nil {
		return errors.Wrap(err, "ensure output.md")
	}
	stopBody := fmt.Sprintf("run stopped with code %d\nrun_dir: %s\noutput: %s",
		info.ExitCode,
		runDir,
		info.OutputPath,
	)
	if info.Status == storage.StatusFailed {
		if excerpt := tailFile(info.StderrPath, 50); excerpt != "" {
			stopBody += "\n\n## stderr (last 50 lines)\n" + excerpt
		}
	}
	stopEvent := messagebus.EventTypeRunStop
	if execErr != nil {
		stopEvent = messagebus.EventTypeRunCrash
	}
	if err := postRunEvent(busPath, info, stopEvent, stopBody); err != nil {
		return err
	}
	if execErr != nil {
		return errors.Wrap(execErr, "agent execution failed")
	}
	return nil
}

func postRunEvent(busPath string, info *storage.RunInfo, msgType, body string) error {
	if info == nil {
		return errors.New("run info is nil")
	}
	if strings.TrimSpace(busPath) == "" {
		return nil
	}
	bus, err := messagebus.NewMessageBus(busPath)
	if err != nil {
		return errors.Wrap(err, "new message bus")
	}
	_, err = bus.AppendMessage(&messagebus.Message{
		Type:      msgType,
		ProjectID: info.ProjectID,
		TaskID:    info.TaskID,
		RunID:     info.RunID,
		Body:      body,
	})
	if err != nil {
		return errors.Wrap(err, "append message")
	}
	return nil
}

// commandForAgent returns the CLI command and arguments for the given agent type.
// Working directory is handled by SpawnOptions.Dir, not by CLI flags.
func commandForAgent(agentType string) (string, []string, error) {
	switch strings.ToLower(agentType) {
	case "codex":
		args := []string{"exec", "--dangerously-bypass-approvals-and-sandbox", "--json", "-"}
		return "codex", args, nil
	case "claude":
		args := []string{
			"-p",
			"--input-format", "text",
			"--output-format", "stream-json",
			"--verbose",
			"--tools", "default",
			"--permission-mode", "bypassPermissions",
		}
		return "claude", args, nil
	case "gemini":
		args := []string{"--screen-reader", "true", "--approval-mode", "yolo", "--output-format", "stream-json"}
		return "gemini", args, nil
	default:
		return "", nil, fmt.Errorf("unsupported agent type %q", agentType)
	}
}

// removeEnvKeys returns a copy of env with the given keys removed.
func removeEnvKeys(env []string, keys ...string) []string {
	remove := make(map[string]struct{}, len(keys))
	for _, k := range keys {
		remove[k] = struct{}{}
	}
	result := make([]string, 0, len(env))
	for _, entry := range env {
		parts := strings.SplitN(entry, "=", 2)
		if _, ok := remove[parts[0]]; ok {
			continue
		}
		result = append(result, entry)
	}
	return result
}

func envMap(env []string) map[string]string {
	values := make(map[string]string)
	for _, entry := range env {
		if entry == "" {
			continue
		}
		parts := strings.SplitN(entry, "=", 2)
		key := strings.TrimSpace(parts[0])
		if key == "" {
			continue
		}
		value := ""
		if len(parts) > 1 {
			value = parts[1]
		}
		values[key] = value
	}
	return values
}

// tailFile reads the last N lines from a file. Returns empty string if file doesn't exist or is empty.
func tailFile(path string, maxLines int) string {
	if maxLines <= 0 {
		return ""
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return ""
	}
	content := string(data)
	if content == "" {
		return ""
	}
	lines := strings.Split(content, "\n")
	// Remove trailing empty line from final newline
	if len(lines) > 0 && lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	if len(lines) == 0 {
		return ""
	}
	if len(lines) > maxLines {
		lines = lines[len(lines)-maxLines:]
	}
	return strings.Join(lines, "\n")
}

// classifyExitCode returns a one-line error summary for a non-zero exit code.
func classifyExitCode(exitCode int) string {
	switch exitCode {
	case 1:
		return "agent reported failure"
	case 2:
		return "agent usage error"
	case 137:
		return "agent killed (OOM or signal)"
	case 143:
		return "agent terminated (SIGTERM)"
	default:
		return fmt.Sprintf("agent exited with code %d", exitCode)
	}
}

func ctxOrBackground() context.Context {
	return context.Background()
}

// detectAgentVersion returns the CLI version string for CLI agents (best-effort).
// Returns empty string for REST agents or if detection fails.
func detectAgentVersion(ctx context.Context, agentType string) string {
	if isRestAgent(agentType) {
		return ""
	}
	command := cliCommand(agentType)
	if command == "" {
		return ""
	}
	version, err := agent.DetectCLIVersion(ctx, command)
	if err != nil {
		return ""
	}
	return version
}
