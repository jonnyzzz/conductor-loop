package runner

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/jonnyzzz/conductor-loop/internal/agent"
	"github.com/jonnyzzz/conductor-loop/internal/messagebus"
	"github.com/jonnyzzz/conductor-loop/internal/storage"
	"github.com/pkg/errors"
)

const (
	importedLogPollInterval      = 250 * time.Millisecond
	importedLivenessPollInterval = 500 * time.Millisecond
)

// ImportedProcess describes an already-running external process that should be
// adopted into run tracking.
type ImportedProcess struct {
	PID         int
	PGID        int
	AgentType   string
	CommandLine string
	StdoutPath  string
	StderrPath  string
	Ownership   string // managed or external; defaults to external
}

// ImportOptions controls adoption behavior for RunImportedProcess.
type ImportOptions struct {
	RootDir            string
	WorkingDir         string
	MessageBusPath     string
	PreallocatedRunDir string
	Process            ImportedProcess
}

// RunImportedProcess adopts an already-running process into a tracked run.
// It mirrors external stdout/stderr logs into canonical run files so existing
// streaming endpoints continue to work.
func RunImportedProcess(projectID, taskID string, opts ImportOptions) error {
	_, err := runImportedProcess(projectID, taskID, opts)
	return err
}

func runImportedProcess(projectID, taskID string, opts ImportOptions) (*storage.RunInfo, error) {
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

	process, err := normalizeImportedProcess(opts.Process)
	if err != nil {
		return nil, err
	}
	if !importedProcessAlive(process.PID, process.PGID) {
		return nil, fmt.Errorf("imported process is not alive (pid=%d pgid=%d)", process.PID, process.PGID)
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
		runID = filepath.Base(preallocated)
	} else {
		runID, runDir, err = createRunDir(runsDir)
		if err != nil {
			return nil, err
		}
	}
	runDirAbs, err := absPath(runDir)
	if err != nil {
		return nil, errors.Wrap(err, "resolve run dir")
	}

	promptPath := filepath.Join(runDirAbs, "prompt.md")
	stdoutPath := filepath.Join(runDirAbs, "agent-stdout.txt")
	stderrPath := filepath.Join(runDirAbs, "agent-stderr.txt")
	outputPath := filepath.Join(runDirAbs, "output.md")

	if err := os.WriteFile(promptPath, []byte(buildImportedPrompt(process, workingDir)), 0o644); err != nil {
		return nil, errors.Wrap(err, "write prompt")
	}
	if err := ensureImportLogFile(stdoutPath, process.StdoutPath, "stdout"); err != nil {
		return nil, err
	}
	if err := ensureImportLogFile(stderrPath, process.StderrPath, "stderr"); err != nil {
		return nil, err
	}

	info := &storage.RunInfo{
		Version:          1,
		RunID:            runID,
		ProjectID:        projectID,
		TaskID:           taskID,
		AgentType:        process.AgentType,
		ProcessOwnership: process.Ownership,
		PID:              process.PID,
		PGID:             process.PGID,
		StartTime:        time.Now().UTC(),
		ExitCode:         -1,
		Status:           storage.StatusRunning,
		CWD:              workingDir,
		PromptPath:       promptPath,
		OutputPath:       outputPath,
		StdoutPath:       stdoutPath,
		StderrPath:       stderrPath,
		CommandLine:      process.CommandLine,
	}

	runInfoPath := filepath.Join(runDir, "run-info.yaml")
	if err := storage.WriteRunInfo(runInfoPath, info); err != nil {
		return nil, errors.Wrap(err, "write run-info")
	}

	startBody := fmt.Sprintf(
		"run adopted from external process\nrun_dir: %s\npid: %d\npgid: %d\nownership: %s\nstdout: %s\nstderr: %s\noutput: %s",
		runDir,
		process.PID,
		process.PGID,
		process.Ownership,
		info.StdoutPath,
		info.StderrPath,
		info.OutputPath,
	)
	if err := postRunEvent(busPath, info, messagebus.EventTypeRunStart, startBody); err != nil {
		return info, err
	}

	mirrorCtx, cancelMirrors := context.WithCancel(context.Background())
	defer cancelMirrors()

	var mirrorWG sync.WaitGroup
	mirrorErrs := make(chan error, 4)
	startMirror := func(source, target, streamName string) {
		if strings.TrimSpace(source) == "" {
			return
		}
		sourceAbs := filepath.Clean(source)
		targetAbs := filepath.Clean(target)
		if sourceAbs == targetAbs {
			return
		}
		mirrorWG.Add(1)
		go func() {
			defer mirrorWG.Done()
			if mirrorErr := mirrorLogFile(mirrorCtx, sourceAbs, targetAbs); mirrorErr != nil {
				mirrorErrs <- fmt.Errorf("%s mirror: %w", streamName, mirrorErr)
			}
		}()
	}
	startMirror(process.StdoutPath, stdoutPath, "stdout")
	startMirror(process.StderrPath, stderrPath, "stderr")

	for importedProcessAlive(process.PID, process.PGID) {
		time.Sleep(importedLivenessPollInterval)
	}

	cancelMirrors()
	mirrorWG.Wait()
	close(mirrorErrs)

	var mirrorWarnings []string
	for mirrorErr := range mirrorErrs {
		if mirrorErr != nil {
			mirrorWarnings = append(mirrorWarnings, mirrorErr.Error())
		}
	}

	info.EndTime = time.Now().UTC()
	info.ExitCode = -1
	info.Status = storage.StatusCompleted
	info.ErrorSummary = "adopted process exited; exit code unavailable"
	if len(mirrorWarnings) > 0 {
		info.ErrorSummary = fmt.Sprintf("%s; log mirror warnings: %s", info.ErrorSummary, strings.Join(mirrorWarnings, "; "))
	}

	if err := storage.UpdateRunInfo(runInfoPath, func(update *storage.RunInfo) error {
		update.EndTime = info.EndTime
		update.ExitCode = info.ExitCode
		update.Status = info.Status
		update.ErrorSummary = info.ErrorSummary
		return nil
	}); err != nil {
		return info, errors.Wrap(err, "update run-info")
	}

	if _, err := agent.CreateOutputMD(runDir, ""); err != nil {
		return info, errors.Wrap(err, "ensure output.md")
	}

	stopBody := fmt.Sprintf("adopted process exited\nrun_dir: %s\nstatus: %s\nexit_code: %d\noutput: %s",
		runDir,
		info.Status,
		info.ExitCode,
		info.OutputPath,
	)
	if len(mirrorWarnings) > 0 {
		stopBody += "\n\n## mirror warnings\n" + strings.Join(mirrorWarnings, "\n")
	}
	if err := postRunEvent(busPath, info, messagebus.EventTypeRunStop, stopBody); err != nil {
		return info, err
	}

	return info, nil
}

func normalizeImportedProcess(spec ImportedProcess) (ImportedProcess, error) {
	pid := spec.PID
	if pid <= 0 {
		return ImportedProcess{}, errors.New("process_import.pid must be > 0")
	}

	pgid := spec.PGID
	if pgid <= 0 {
		if resolved, err := ProcessGroupID(pid); err == nil && resolved > 0 {
			pgid = resolved
		} else {
			pgid = pid
		}
	}

	agentType := strings.ToLower(strings.TrimSpace(spec.AgentType))
	if agentType == "" {
		return ImportedProcess{}, errors.New("process_import.agent_type is required")
	}

	ownership := strings.ToLower(strings.TrimSpace(spec.Ownership))
	if ownership == "" {
		ownership = storage.ProcessOwnershipExternal
	}
	ownership = storage.NormalizeProcessOwnership(ownership)

	commandLine := strings.TrimSpace(spec.CommandLine)
	if commandLine == "" {
		commandLine = fmt.Sprintf("%s (adopted pid %d)", agentType, pid)
	}

	stdoutPath, err := normalizeOptionalAbsFile(spec.StdoutPath, "process_import.stdout_path")
	if err != nil {
		return ImportedProcess{}, err
	}
	stderrPath, err := normalizeOptionalAbsFile(spec.StderrPath, "process_import.stderr_path")
	if err != nil {
		return ImportedProcess{}, err
	}
	if stdoutPath == "" && stderrPath == "" {
		return ImportedProcess{}, errors.New("process_import requires stdout_path and/or stderr_path")
	}

	return ImportedProcess{
		PID:         pid,
		PGID:        pgid,
		AgentType:   agentType,
		CommandLine: commandLine,
		StdoutPath:  stdoutPath,
		StderrPath:  stderrPath,
		Ownership:   ownership,
	}, nil
}

func normalizeOptionalAbsFile(path, label string) (string, error) {
	clean := strings.TrimSpace(path)
	if clean == "" {
		return "", nil
	}
	abs, err := filepath.Abs(clean)
	if err != nil {
		return "", errors.Wrapf(err, "resolve %s", label)
	}
	if stat, statErr := os.Stat(abs); statErr == nil {
		if stat.IsDir() {
			return "", fmt.Errorf("%s must be a file path, got directory: %s", label, abs)
		}
		return abs, nil
	} else if !os.IsNotExist(statErr) {
		return "", errors.Wrapf(statErr, "stat %s", label)
	}
	parent := filepath.Dir(abs)
	if stat, statErr := os.Stat(parent); statErr != nil || !stat.IsDir() {
		if statErr != nil {
			return "", errors.Wrapf(statErr, "stat parent for %s", label)
		}
		return "", fmt.Errorf("parent for %s is not a directory: %s", label, parent)
	}
	return abs, nil
}

func buildImportedPrompt(process ImportedProcess, workingDir string) string {
	return strings.TrimSpace(fmt.Sprintf(
		"# Imported External Process\n\nagent: %s\npid: %d\npgid: %d\nownership: %s\ncommand: %s\ncwd: %s\nstdout_source: %s\nstderr_source: %s\n",
		process.AgentType,
		process.PID,
		process.PGID,
		process.Ownership,
		process.CommandLine,
		workingDir,
		emptyIfMissing(process.StdoutPath),
		emptyIfMissing(process.StderrPath),
	)) + "\n"
}

func emptyIfMissing(path string) string {
	clean := strings.TrimSpace(path)
	if clean == "" {
		return "(not provided)"
	}
	return clean
}

func ensureImportLogFile(targetPath, sourcePath, streamName string) error {
	target := filepath.Clean(strings.TrimSpace(targetPath))
	if target == "." || target == "" {
		return fmt.Errorf("invalid %s target path", streamName)
	}
	initial := ""
	if strings.TrimSpace(sourcePath) == "" {
		initial = fmt.Sprintf("no %s source path provided for imported process; live transcript capture is unavailable\n", streamName)
	}
	if err := os.WriteFile(target, []byte(initial), 0o644); err != nil {
		return errors.Wrapf(err, "write %s placeholder", streamName)
	}
	return nil
}

func mirrorLogFile(ctx context.Context, sourcePath, targetPath string) error {
	target, err := os.OpenFile(targetPath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0o644)
	if err != nil {
		return errors.Wrap(err, "open target log file")
	}
	defer target.Close()

	var offset int64
	for {
		if err := copyFileDelta(sourcePath, target, &offset); err != nil {
			return err
		}
		select {
		case <-ctx.Done():
			_ = copyFileDelta(sourcePath, target, &offset)
			return nil
		case <-time.After(importedLogPollInterval):
		}
	}
}

func copyFileDelta(sourcePath string, target io.Writer, offset *int64) error {
	if offset == nil {
		return errors.New("offset is nil")
	}
	source, err := os.Open(sourcePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return errors.Wrap(err, "open source log file")
	}
	defer source.Close()

	stat, err := source.Stat()
	if err != nil {
		return errors.Wrap(err, "stat source log file")
	}
	if stat.IsDir() {
		return fmt.Errorf("source log file is a directory: %s", sourcePath)
	}

	size := stat.Size()
	if size < *offset {
		*offset = 0
	}
	if size == *offset {
		return nil
	}
	if _, err := source.Seek(*offset, io.SeekStart); err != nil {
		return errors.Wrap(err, "seek source log file")
	}
	copied, err := io.CopyN(target, source, size-*offset)
	*offset += copied
	if err != nil && !errors.Is(err, io.EOF) {
		return errors.Wrap(err, "copy source log delta")
	}
	return nil
}

func importedProcessAlive(pid, pgid int) bool {
	// Prefer PID liveness when available; PGID can outlive a specific process.
	if pid > 0 {
		return IsProcessAlive(pid)
	}
	if pgid > 0 {
		if alive, err := IsProcessGroupAlive(pgid); err == nil && alive {
			return true
		}
	}
	return false
}
