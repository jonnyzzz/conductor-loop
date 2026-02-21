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
	PreallocatedRunDir string // optional: pre-created run directory; skip createRunDir if set
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

	parentRunID := strings.TrimSpace(opts.ParentRunID)

	promptPath := filepath.Join(runDir, "prompt.md")
	promptContent := buildPrompt(PromptParams{
		TaskDir:     taskDir,
		RunDir:      runDir,
		ProjectID:   projectID,
		TaskID:      taskID,
		RunID:       runID,
		ParentRunID: parentRunID,
	}, promptText)
	if err := os.WriteFile(promptPath, []byte(promptContent), 0o644); err != nil {
		return nil, errors.Wrap(err, "write prompt")
	}

	cfg, err := loadConfig(opts.ConfigPath)
	if err != nil {
		return nil, err
	}
	var notifier *webhook.Notifier
	if cfg != nil {
		notifier = webhook.NewNotifier(cfg.Webhook)
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

	// Warn-only token validation
	if tokenErr := ValidateToken(agentType, selection.Config.Token); tokenErr != nil {
		log.Printf("warning: %v", tokenErr)
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
		Version:       1,
		RunID:         runID,
		ParentRunID:   parentRunID,
		PreviousRunID: strings.TrimSpace(opts.PreviousRunID),
		ProjectID:     projectID,
		TaskID:        taskID,
		AgentType:     agentType,
		AgentVersion:  detectAgentVersion(context.Background(), agentType),
		StartTime:     time.Now().UTC(),
		ExitCode:      -1,
		Status:        storage.StatusRunning,
		CWD:           workingDir,
		PromptPath:    promptPathAbs,
		OutputPath:    outputPathAbs,
		StdoutPath:    stdoutPathAbs,
		StderrPath:    stderrPathAbs,
	}

	var execErr error
	if isRestAgent(agentType) {
		execErr = executeREST(ctxOrBackground(), agentType, selection, promptContent, workingDir, env, runDir, busPath, info)
	} else {
		execErr = executeCLI(ctxOrBackground(), agentType, promptPathAbs, workingDir, env, runDir, busPath, info)
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

func executeCLI(ctx context.Context, agentType, promptPath, workingDir string, env []string, runDir, busPath string, info *storage.RunInfo) error {
	command, args, err := commandForAgent(agentType)
	if err != nil {
		return err
	}
	promptFile, err := os.Open(promptPath)
	if err != nil {
		return errors.Wrap(err, "open prompt")
	}
	pm, err := NewProcessManager(runDir)
	if err != nil {
		_ = promptFile.Close()
		return err
	}
	proc, err := pm.SpawnAgent(ctx, agentType, SpawnOptions{
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
			return errors.Wrap(writeErr, "write run-info")
		}
		return errors.Wrap(err, "spawn agent")
	}
	info.PID = proc.PID
	info.PGID = proc.PGID
	info.CommandLine = fmt.Sprintf("%s %s < %s", command, strings.Join(args, " "), promptPath)
	if err := storage.WriteRunInfo(filepath.Join(runDir, "run-info.yaml"), info); err != nil {
		_ = proc.Cmd.Process.Kill()
		_ = proc.Wait()
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
		_ = proc.Cmd.Process.Kill()
		_ = proc.Wait()
		return err
	}

	waitErr := proc.Wait()
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
		info.ErrorSummary = classifyExitCode(exitCode)
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
	if exitCode != 0 {
		stopEvent = messagebus.EventTypeRunCrash
	}
	if err := postRunEvent(busPath, info, stopEvent, stopBody); err != nil {
		return err
	}
	if waitErr != nil || exitCode != 0 {
		return errors.Wrap(waitErr, "agent execution failed")
	}
	return nil
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
		args := []string{"exec", "--dangerously-bypass-approvals-and-sandbox", "-"}
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
		args := []string{"--screen-reader", "true", "--approval-mode", "yolo"}
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
