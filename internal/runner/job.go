package runner

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/jonnyzzz/conductor-loop/internal/agent"
	"github.com/jonnyzzz/conductor-loop/internal/agent/perplexity"
	"github.com/jonnyzzz/conductor-loop/internal/agent/xai"
	"github.com/jonnyzzz/conductor-loop/internal/messagebus"
	"github.com/jonnyzzz/conductor-loop/internal/storage"
	"github.com/pkg/errors"
)

// JobOptions controls execution for a single run-agent job.
type JobOptions struct {
	RootDir        string
	ConfigPath     string
	Agent          string
	Prompt         string
	PromptPath     string
	WorkingDir     string
	MessageBusPath string
	ParentRunID    string
	PreviousRunID  string
	Environment    map[string]string
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
	runID, runDir, err := createRunDir(runsDir)
	if err != nil {
		return nil, err
	}

	promptPath := filepath.Join(runDir, "prompt.md")
	promptContent := buildPrompt(taskDir, runDir, promptText)
	if err := os.WriteFile(promptPath, []byte(promptContent), 0o644); err != nil {
		return nil, errors.Wrap(err, "write prompt")
	}

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

	envOverrides := map[string]string{
		"JRUN_PROJECT_ID": projectID,
		"JRUN_TASK_ID":    taskID,
		"JRUN_ID":         runID,
		"JRUN_PARENT_ID":  strings.TrimSpace(opts.ParentRunID),
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
		ParentRunID:   strings.TrimSpace(opts.ParentRunID),
		PreviousRunID: strings.TrimSpace(opts.PreviousRunID),
		ProjectID:     projectID,
		TaskID:        taskID,
		AgentType:     agentType,
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
	command, args, err := commandForAgent(agentType, workingDir)
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
	if err := postRunEvent(busPath, info, "RUN_START", "run started"); err != nil {
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
	if err := storage.UpdateRunInfo(filepath.Join(runDir, "run-info.yaml"), func(update *storage.RunInfo) error {
		update.ExitCode = info.ExitCode
		update.EndTime = info.EndTime
		update.Status = info.Status
		return nil
	}); err != nil {
		return errors.Wrap(err, "update run-info")
	}
	if _, err := agent.CreateOutputMD(runDir, ""); err != nil {
		return errors.Wrap(err, "ensure output.md")
	}
	if err := postRunEvent(busPath, info, "RUN_STOP", fmt.Sprintf("run stopped with code %d", info.ExitCode)); err != nil {
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
	if err := postRunEvent(busPath, info, "RUN_START", "run started"); err != nil {
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
	} else {
		info.ExitCode = 0
		info.Status = storage.StatusCompleted
	}
	if err := storage.UpdateRunInfo(filepath.Join(runDir, "run-info.yaml"), func(update *storage.RunInfo) error {
		update.ExitCode = info.ExitCode
		update.EndTime = info.EndTime
		update.Status = info.Status
		return nil
	}); err != nil {
		return errors.Wrap(err, "update run-info")
	}
	if _, err := agent.CreateOutputMD(runDir, ""); err != nil {
		return errors.Wrap(err, "ensure output.md")
	}
	if err := postRunEvent(busPath, info, "RUN_STOP", fmt.Sprintf("run stopped with code %d", info.ExitCode)); err != nil {
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

func commandForAgent(agentType, workingDir string) (string, []string, error) {
	switch strings.ToLower(agentType) {
	case "codex":
		args := []string{"exec", "--dangerously-bypass-approvals-and-sandbox", "-"}
		if strings.TrimSpace(workingDir) != "" {
			args = []string{"exec", "--dangerously-bypass-approvals-and-sandbox", "-C", workingDir, "-"}
		}
		return "codex", args, nil
	case "claude":
		args := []string{
			"-p",
			"--input-format",
			"text",
			"--output-format",
			"text",
			"--tools",
			"default",
			"--permission-mode",
			"bypassPermissions",
		}
		if strings.TrimSpace(workingDir) != "" {
			args = append([]string{"-C", workingDir}, args...)
		}
		return "claude", args, nil
	case "gemini":
		args := []string{"--screen-reader", "true", "--approval-mode", "yolo"}
		return "gemini", args, nil
	default:
		return "", nil, fmt.Errorf("unsupported agent type %q", agentType)
	}
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

func ctxOrBackground() context.Context {
	return context.Background()
}
