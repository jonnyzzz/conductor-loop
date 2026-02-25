package runner

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/jonnyzzz/conductor-loop/internal/agent"
	"github.com/jonnyzzz/conductor-loop/internal/messagebus"
	"github.com/jonnyzzz/conductor-loop/internal/storage"
	"github.com/pkg/errors"
)

const wrapStdioNotice = "run-agent wrap used terminal passthrough stdio to preserve CLI behavior; console transcript was not captured.\n"

// WrapOptions controls execution for a wrapped CLI invocation.
type WrapOptions struct {
	RootDir       string
	WorkingDir    string
	ParentRunID   string
	PreviousRunID  string
	Environment    map[string]string
	Timeout        time.Duration // hard timeout for wrapped process; 0 means no limit
	TaskPrompt     string        // optional TASK.md content when task is created
}

// RunWrap executes a wrapped agent command while recording task/run metadata.
func RunWrap(projectID, taskID, agentType string, forwardedArgs []string, opts WrapOptions) error {
	_, err := runWrap(projectID, taskID, agentType, forwardedArgs, opts)
	return err
}

func runWrap(projectID, taskID, agentType string, forwardedArgs []string, opts WrapOptions) (*storage.RunInfo, error) {
	commandName := strings.TrimSpace(agentType)
	if commandName == "" {
		return nil, errors.New("agent type is empty")
	}
	agentLabel := strings.ToLower(commandName)

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

	workingDir := strings.TrimSpace(opts.WorkingDir)
	if workingDir == "" {
		workingDir, err = os.Getwd()
		if err != nil {
			return nil, errors.Wrap(err, "resolve working dir")
		}
	}
	workingDir, err = absPath(workingDir)
	if err != nil {
		return nil, errors.Wrap(err, "resolve working dir")
	}

	if err := ensureWrapTaskPrompt(filepath.Join(taskDir, "TASK.md"), opts.TaskPrompt, agentLabel, forwardedArgs, workingDir); err != nil {
		return nil, err
	}

	busPath := filepath.Join(taskDir, "TASK-MESSAGE-BUS.md")

	runsDir := filepath.Join(taskDir, "runs")
	if err := ensureDir(runsDir); err != nil {
		return nil, errors.Wrap(err, "ensure runs dir")
	}

	runID, runDir, err := createRunDir(runsDir)
	if err != nil {
		return nil, err
	}
	runDirAbs, err := absPath(runDir)
	if err != nil {
		return nil, errors.Wrap(err, "resolve run dir")
	}

	promptPath := filepath.Join(runDirAbs, "prompt.md")
	stdoutPath := filepath.Join(runDirAbs, "agent-stdout.txt")
	stderrPath := filepath.Join(runDirAbs, "agent-stderr.txt")
	outputPath := filepath.Join(runDirAbs, "output.md")
	runInfoPath := filepath.Join(runDir, "run-info.yaml")

	commandLine := renderWrappedCommandLine(commandName, forwardedArgs)
	promptContent := buildWrapPrompt(commandLine, workingDir)
	if err := os.WriteFile(promptPath, []byte(promptContent), 0o644); err != nil {
		return nil, errors.Wrap(err, "write prompt")
	}
	if err := os.WriteFile(stdoutPath, []byte(wrapStdioNotice), 0o644); err != nil {
		return nil, errors.Wrap(err, "write stdout placeholder")
	}
	if err := os.WriteFile(stderrPath, []byte(wrapStdioNotice), 0o644); err != nil {
		return nil, errors.Wrap(err, "write stderr placeholder")
	}

	parentRunID := strings.TrimSpace(opts.ParentRunID)
	previousRunID := strings.TrimSpace(opts.PreviousRunID)
	agentVersion := detectAgentVersion(context.Background(), agentLabel)

	info := &storage.RunInfo{
		Version:          1,
		RunID:            runID,
		ParentRunID:      parentRunID,
		PreviousRunID:    previousRunID,
		ProjectID:        projectID,
		TaskID:           taskID,
		AgentType:        agentLabel,
		ProcessOwnership: storage.ProcessOwnershipManaged,
		AgentVersion:     agentVersion,
		StartTime:        time.Now().UTC(),
		ExitCode:         -1,
		Status:           storage.StatusRunning,
		CWD:              workingDir,
		PromptPath:       promptPath,
		OutputPath:       outputPath,
		StdoutPath:       stdoutPath,
		StderrPath:       stderrPath,
		CommandLine:      commandLine,
	}

	envOverrides := map[string]string{
		"JRUN_PROJECT_ID": projectID,
		"JRUN_TASK_ID":    taskID,
		"JRUN_ID":         runID,
		"JRUN_PARENT_ID":  parentRunID,
		"JRUN_RUNS_DIR":        runsDir,
		"JRUN_MESSAGE_BUS":     busPath,
		"JRUN_TASK_FOLDER":     taskDir,
		"JRUN_RUN_FOLDER":      runDir,
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

	ctx := context.Background()
	cancel := func() {}
	if opts.Timeout > 0 {
		ctx, cancel = context.WithTimeout(ctx, opts.Timeout)
	}
	defer cancel()

	cmd := exec.CommandContext(ctx, commandName, forwardedArgs...)
	cmd.Dir = workingDir
	cmd.Env = env
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Start(); err != nil {
		pid := os.Getpid()
		pgid, pgidErr := ProcessGroupID(pid)
		if pgidErr != nil {
			pgid = pid
		}
		info.PID = pid
		info.PGID = pgid
		info.EndTime = time.Now().UTC()
		info.Status = storage.StatusFailed
		errMsg := err.Error()
		if len(errMsg) > 200 {
			errMsg = errMsg[:200]
		}
		info.ErrorSummary = errMsg
		if writeErr := storage.WriteRunInfo(runInfoPath, info); writeErr != nil {
			return info, errors.Wrap(writeErr, "write run-info")
		}
		crashBody := fmt.Sprintf("run failed to start\nrun_dir: %s\nerror: %v", runDir, err)
		_ = postRunEvent(busPath, info, messagebus.EventTypeRunCrash, crashBody)
		_, _ = agent.CreateOutputMD(runDir, "")
		return info, errors.Wrap(err, "start wrapped command")
	}

	info.PID = cmd.Process.Pid
	pgid, pgidErr := ProcessGroupID(info.PID)
	if pgidErr != nil {
		info.PGID = info.PID
	} else {
		info.PGID = pgid
	}
	if err := storage.WriteRunInfo(runInfoPath, info); err != nil {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
		return info, errors.Wrap(err, "write run-info")
	}

	startBody := fmt.Sprintf("run started\nrun_dir: %s\nprompt: %s\nstdout: %s\nstderr: %s\noutput: %s",
		runDir,
		info.PromptPath,
		info.StdoutPath,
		info.StderrPath,
		info.OutputPath,
	)
	if err := postRunEvent(busPath, info, messagebus.EventTypeRunStart, startBody); err != nil {
		_ = cmd.Process.Kill()
		_ = cmd.Wait()
		return info, err
	}

	waitErr := cmd.Wait()
	exitCode := 0
	if cmd.ProcessState != nil {
		exitCode = cmd.ProcessState.ExitCode()
	}
	info.ExitCode = exitCode
	info.EndTime = time.Now().UTC()
	timedOut := opts.Timeout > 0 && ctx.Err() == context.DeadlineExceeded
	if waitErr == nil && exitCode == 0 {
		info.Status = storage.StatusCompleted
	} else {
		info.Status = storage.StatusFailed
		if timedOut {
			info.ErrorSummary = "timed out"
		} else if exitCode != 0 {
			info.ErrorSummary = classifyExitCode(exitCode)
		} else if waitErr != nil {
			errMsg := waitErr.Error()
			if len(errMsg) > 200 {
				errMsg = errMsg[:200]
			}
			info.ErrorSummary = errMsg
		}
	}
	if err := storage.UpdateRunInfo(runInfoPath, func(update *storage.RunInfo) error {
		update.ExitCode = info.ExitCode
		update.EndTime = info.EndTime
		update.Status = info.Status
		update.ErrorSummary = info.ErrorSummary
		return nil
	}); err != nil {
		return info, errors.Wrap(err, "update run-info")
	}
	if _, err := agent.CreateOutputMD(runDir, ""); err != nil {
		return info, errors.Wrap(err, "ensure output.md")
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
	if info.Status == storage.StatusFailed {
		stopEvent = messagebus.EventTypeRunCrash
	}
	if err := postRunEvent(busPath, info, stopEvent, stopBody); err != nil {
		return info, err
	}

	if waitErr != nil {
		return info, errors.Wrap(waitErr, "wrapped command failed")
	}
	if exitCode != 0 {
		return info, fmt.Errorf("wrapped command exited with code %d", exitCode)
	}
	return info, nil
}

func ensureWrapTaskPrompt(taskMDPath, taskPrompt, agentType string, args []string, workingDir string) error {
	if _, err := os.Stat(taskMDPath); err == nil {
		return nil
	} else if !os.IsNotExist(err) {
		return errors.Wrap(err, "stat TASK.md")
	}

	prompt := strings.TrimSpace(taskPrompt)
	if prompt == "" {
		prompt = defaultWrapTaskPrompt(agentType, args, workingDir)
	}
	if err := os.WriteFile(taskMDPath, []byte(prompt+"\n"), 0o644); err != nil {
		return errors.Wrap(err, "write TASK.md")
	}
	return nil
}

func defaultWrapTaskPrompt(agentType string, args []string, workingDir string) string {
	return strings.TrimSpace(fmt.Sprintf(
		"Wrapped CLI invocation.\n\nagent: %s\ncommand: %s\ncwd: %s",
		agentType,
		renderWrappedCommandLine(agentType, args),
		workingDir,
	))
}

func buildWrapPrompt(commandLine, workingDir string) string {
	return strings.TrimSpace(fmt.Sprintf(
		"# Wrapped Invocation\n\ncommand: %s\ncwd: %s\n\n%s",
		commandLine,
		workingDir,
		wrapStdioNotice,
	)) + "\n"
}

func renderWrappedCommandLine(command string, args []string) string {
	parts := make([]string, 0, len(args)+1)
	parts = append(parts, maybeQuoteShellWord(strings.TrimSpace(command)))
	for _, arg := range args {
		parts = append(parts, maybeQuoteShellWord(arg))
	}
	return strings.TrimSpace(strings.Join(parts, " "))
}

func maybeQuoteShellWord(value string) string {
	if value == "" {
		return `""`
	}
	if strings.ContainsAny(value, " \t\n\r\"'`\\$") {
		return strconv.Quote(value)
	}
	return value
}
