package runner

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"time"

	"github.com/jonnyzzz/conductor-loop/internal/storage"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v3"
)

const (
	// WorkflowTemplatePromptV5 is the currently supported staged workflow template.
	WorkflowTemplatePromptV5 = "THE_PROMPT_v5"

	workflowStateVersion = 1

	workflowStageStatusPending   = "pending"
	workflowStageStatusRunning   = "running"
	workflowStageStatusCompleted = "completed"
	workflowStageStatusFailed    = "failed"
)

var workflowStageTitles = map[int]string{
	0:  "Preflight",
	1:  "Context Gathering",
	2:  "Constraint Audit",
	3:  "Decomposition",
	4:  "Test Plan",
	5:  "Implementation",
	6:  "Implementation Refinement",
	7:  "Validation",
	8:  "Independent Review A",
	9:  "Independent Review B",
	10: "Synthesis",
	11: "Finalization",
	12: "Completion",
}

// WorkflowOptions controls staged workflow execution.
type WorkflowOptions struct {
	RootDir    string
	ConfigPath string
	Agent      string
	WorkingDir string
	Template   string
	FromStage      int
	ToStage        int
	Resume         bool
	DryRun         bool
	Timeout        time.Duration
	StatePath      string

	stageExecutor workflowStageExecutor
}

// WorkflowResult describes the final workflow execution state.
type WorkflowResult struct {
	Template       string        `json:"template"`
	FromStage      int           `json:"from_stage"`
	ToStage        int           `json:"to_stage"`
	Resume         bool          `json:"resume"`
	DryRun         bool          `json:"dry_run"`
	StatePath      string        `json:"state_path"`
	PlannedStages  []int         `json:"planned_stages"`
	ExecutedStages []int         `json:"executed_stages"`
	SkippedStages  []int         `json:"skipped_stages"`
	State          WorkflowState `json:"state"`
}

// WorkflowState is the persisted workflow stage state.
type WorkflowState struct {
	Version     int             `json:"version" yaml:"version"`
	Template    string          `json:"template" yaml:"template"`
	ProjectID   string          `json:"project_id" yaml:"project_id"`
	TaskID      string          `json:"task_id" yaml:"task_id"`
	FromStage   int             `json:"from_stage" yaml:"from_stage"`
	ToStage     int             `json:"to_stage" yaml:"to_stage"`
	CreatedAt   time.Time       `json:"created_at" yaml:"created_at"`
	UpdatedAt   time.Time       `json:"updated_at" yaml:"updated_at"`
	CompletedAt time.Time       `json:"completed_at,omitempty" yaml:"completed_at,omitempty"`
	Stages      []WorkflowStage `json:"stages" yaml:"stages"`
}

// WorkflowStage tracks execution status for one template stage.
type WorkflowStage struct {
	Stage     int       `json:"stage" yaml:"stage"`
	Name      string    `json:"name" yaml:"name"`
	Status    string    `json:"status" yaml:"status"`
	Attempts  int       `json:"attempts" yaml:"attempts"`
	RunID     string    `json:"run_id,omitempty" yaml:"run_id,omitempty"`
	StartedAt time.Time `json:"started_at,omitempty" yaml:"started_at,omitempty"`
	EndedAt   time.Time `json:"ended_at,omitempty" yaml:"ended_at,omitempty"`
	Error     string    `json:"error,omitempty" yaml:"error,omitempty"`
}

type workflowStageExecutor func(projectID, taskID string, stage int, prompt string, opts WorkflowOptions) (*storage.RunInfo, error)

// RunWorkflow executes a staged workflow and persists resumable stage state.
func RunWorkflow(projectID, taskID string, opts WorkflowOptions) (*WorkflowResult, error) {
	projectID = strings.TrimSpace(projectID)
	taskID = strings.TrimSpace(taskID)
	if projectID == "" {
		return nil, errors.New("project is required")
	}
	if taskID == "" {
		return nil, errors.New("task is required")
	}
	if err := storage.ValidateTaskID(taskID); err != nil {
		return nil, err
	}

	template, err := normalizeWorkflowTemplate(opts.Template)
	if err != nil {
		return nil, err
	}
	fromStage, toStage, err := normalizeWorkflowStageRange(template, opts.FromStage, opts.ToStage)
	if err != nil {
		return nil, err
	}

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

	busPath := filepath.Join(taskDir, "TASK-MESSAGE-BUS.md")

	statePath := strings.TrimSpace(opts.StatePath)
	if statePath == "" {
		statePath = defaultWorkflowStatePath(taskDir, template)
	}
	statePath = filepath.Clean(statePath)
	if err := ensureDir(filepath.Dir(statePath)); err != nil {
		return nil, errors.Wrap(err, "ensure workflow state directory")
	}

	state, err := loadOrInitWorkflowState(statePath, projectID, taskID, template, fromStage, toStage, opts.Resume)
	if err != nil {
		return nil, err
	}

	result := &WorkflowResult{
		Template:      template,
		FromStage:     fromStage,
		ToStage:       toStage,
		Resume:        opts.Resume,
		DryRun:        opts.DryRun,
		StatePath:     statePath,
		PlannedStages: workflowStageNumbers(fromStage, toStage),
		State:         *state,
	}

	mode := "fresh"
	if opts.Resume {
		mode = "resume"
	}
	_ = postWorkflowBusMessage(busPath, projectID, taskID, "", "DECISION",
		fmt.Sprintf("workflow run started (%s) template=%s stages=%d..%d", mode, template, fromStage, toStage))

	if opts.DryRun {
		_ = postWorkflowBusMessage(busPath, projectID, taskID, "", "INFO",
			fmt.Sprintf("workflow dry-run planned stages=%v", result.PlannedStages))
		result.State = *state
		return result, nil
	}

	execStage := opts.stageExecutor
	if execStage == nil {
		execStage = runWorkflowStage
	}

	for _, stageNum := range result.PlannedStages {
		entry := findOrCreateWorkflowStage(state, stageNum)
		if opts.Resume && entry.Status == workflowStageStatusCompleted {
			result.SkippedStages = append(result.SkippedStages, stageNum)
			_ = postWorkflowBusMessage(busPath, projectID, taskID, entry.RunID, "DECISION",
				fmt.Sprintf("workflow stage %d skipped (already completed)", stageNum))
			continue
		}

		result.ExecutedStages = append(result.ExecutedStages, stageNum)
		now := time.Now().UTC()
		entry.Status = workflowStageStatusRunning
		entry.Attempts++
		entry.StartedAt = now
		entry.EndedAt = time.Time{}
		entry.Error = ""
		state.UpdatedAt = now
		state.CompletedAt = time.Time{}
		if err := saveWorkflowState(statePath, state); err != nil {
			return nil, err
		}

		_ = postWorkflowBusMessage(busPath, projectID, taskID, "", "PROGRESS",
			fmt.Sprintf("workflow stage %d started", stageNum))

		prompt := workflowStagePrompt(template, stageNum, taskDir)
		info, stageErr := execStage(projectID, taskID, stageNum, prompt, opts)
		if info != nil {
			entry.RunID = info.RunID
		}

		entry.EndedAt = time.Now().UTC()
		state.UpdatedAt = entry.EndedAt
		if stageErr != nil {
			entry.Status = workflowStageStatusFailed
			entry.Error = stageErr.Error()
			if err := saveWorkflowState(statePath, state); err != nil {
				return nil, err
			}
			_ = postWorkflowBusMessage(busPath, projectID, taskID, entry.RunID, "ERROR",
				fmt.Sprintf("workflow stage %d failed: %v", stageNum, stageErr))
			result.State = *state
			return result, fmt.Errorf("workflow stage %d failed: %w", stageNum, stageErr)
		}

		entry.Status = workflowStageStatusCompleted
		entry.Error = ""
		if err := saveWorkflowState(statePath, state); err != nil {
			return nil, err
		}
		_ = postWorkflowBusMessage(busPath, projectID, taskID, entry.RunID, "FACT",
			fmt.Sprintf("workflow stage %d completed", stageNum))
	}

	now := time.Now().UTC()
	state.CompletedAt = now
	state.UpdatedAt = now
	if err := saveWorkflowState(statePath, state); err != nil {
		return nil, err
	}

	_ = postWorkflowBusMessage(busPath, projectID, taskID, "", "FACT",
		fmt.Sprintf("workflow completed template=%s stages=%d..%d", template, fromStage, toStage))

	result.State = *state
	return result, nil
}

// WorkflowResultJSON returns a stable JSON rendering of the result.
func WorkflowResultJSON(result *WorkflowResult) ([]byte, error) {
	if result == nil {
		return nil, errors.New("workflow result is nil")
	}
	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return nil, errors.Wrap(err, "marshal workflow result")
	}
	return append(data, '\n'), nil
}

func runWorkflowStage(projectID, taskID string, stage int, prompt string, opts WorkflowOptions) (*storage.RunInfo, error) {
	jobOpts := JobOptions{
		RootDir:    opts.RootDir,
		ConfigPath: opts.ConfigPath,
		Agent:      opts.Agent,
		Prompt:     prompt,
		WorkingDir: opts.WorkingDir,
		Timeout:    opts.Timeout,
	}
	return runJob(projectID, taskID, jobOpts)
}

func workflowStagePrompt(template string, stage int, taskDir string) string {
	return strings.TrimSpace(fmt.Sprintf(`Workflow stage execution request.

Template: %s
Stage: %d - %s
Task folder: %s

Instructions:
1. Read TASK.md in the task folder for baseline context.
2. Execute only this template stage and produce concrete, file-backed outcomes.
3. Preserve prior stage outputs unless this stage explicitly requires changes.
4. Summarize this stage in output.md, including decisions, facts, and unresolved risks.
`, template, stage, workflowStageTitle(stage), taskDir))
}

func normalizeWorkflowTemplate(template string) (string, error) {
	trimmed := strings.TrimSpace(template)
	if trimmed == "" {
		return WorkflowTemplatePromptV5, nil
	}
	if strings.EqualFold(trimmed, WorkflowTemplatePromptV5) {
		return WorkflowTemplatePromptV5, nil
	}
	if strings.EqualFold(trimmed, "the_prompt_v5") {
		return WorkflowTemplatePromptV5, nil
	}
	return "", fmt.Errorf("unsupported template %q", trimmed)
}

func normalizeWorkflowStageRange(template string, fromStage, toStage int) (int, int, error) {
	minStage, maxStage, err := templateStageBounds(template)
	if err != nil {
		return 0, 0, err
	}
	if fromStage < minStage || fromStage > maxStage {
		return 0, 0, fmt.Errorf("from-stage %d out of range [%d..%d]", fromStage, minStage, maxStage)
	}
	if toStage < minStage || toStage > maxStage {
		return 0, 0, fmt.Errorf("to-stage %d out of range [%d..%d]", toStage, minStage, maxStage)
	}
	if fromStage > toStage {
		return 0, 0, fmt.Errorf("from-stage %d must be <= to-stage %d", fromStage, toStage)
	}
	return fromStage, toStage, nil
}

func templateStageBounds(template string) (int, int, error) {
	if template != WorkflowTemplatePromptV5 {
		return 0, 0, fmt.Errorf("unsupported template %q", template)
	}
	return 0, 12, nil
}

func workflowStageTitle(stage int) string {
	if title, ok := workflowStageTitles[stage]; ok {
		return title
	}
	return fmt.Sprintf("Stage %d", stage)
}

func workflowStageNumbers(fromStage, toStage int) []int {
	if fromStage > toStage {
		return nil
	}
	result := make([]int, 0, toStage-fromStage+1)
	for stage := fromStage; stage <= toStage; stage++ {
		result = append(result, stage)
	}
	return result
}

func defaultWorkflowStatePath(taskDir, template string) string {
	cleanTemplate := sanitizeWorkflowTemplate(template)
	return filepath.Join(taskDir, "workflow", cleanTemplate, "stage-state.yaml")
}

func sanitizeWorkflowTemplate(template string) string {
	template = strings.ToLower(strings.TrimSpace(template))
	if template == "" {
		return "template"
	}
	var b strings.Builder
	lastDash := false
	for _, r := range template {
		switch {
		case (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9'):
			b.WriteRune(r)
			lastDash = false
		case r == '-' || r == '_' || r == ' ' || r == '.':
			if !lastDash {
				b.WriteRune('-')
				lastDash = true
			}
		}
	}
	out := strings.Trim(b.String(), "-")
	if out == "" {
		return "template"
	}
	return out
}

func loadOrInitWorkflowState(statePath, projectID, taskID, template string, fromStage, toStage int, resume bool) (*WorkflowState, error) {
	state, err := loadWorkflowState(statePath)
	if err != nil {
		if !os.IsNotExist(err) {
			return nil, err
		}
		state = &WorkflowState{
			Version:   workflowStateVersion,
			Template:  template,
			ProjectID: projectID,
			TaskID:    taskID,
			FromStage: fromStage,
			ToStage:   toStage,
			CreatedAt: time.Now().UTC(),
			UpdatedAt: time.Now().UTC(),
			Stages:    make([]WorkflowStage, 0),
		}
	}

	if err := validateWorkflowStateIdentity(state, projectID, taskID, template); err != nil {
		return nil, err
	}

	state.Version = workflowStateVersion
	state.Template = template
	state.ProjectID = projectID
	state.TaskID = taskID
	state.FromStage = fromStage
	state.ToStage = toStage
	state.UpdatedAt = time.Now().UTC()
	if state.CreatedAt.IsZero() {
		state.CreatedAt = state.UpdatedAt
	}

	for _, stageNum := range workflowStageNumbers(fromStage, toStage) {
		entry := findOrCreateWorkflowStage(state, stageNum)
		if resume {
			if entry.Status == workflowStageStatusRunning {
				entry.Status = workflowStageStatusFailed
				if strings.TrimSpace(entry.Error) == "" {
					entry.Error = "previous run interrupted"
				}
				entry.EndedAt = state.UpdatedAt
			}
			continue
		}

		entry.Status = workflowStageStatusPending
		entry.Attempts = 0
		entry.RunID = ""
		entry.StartedAt = time.Time{}
		entry.EndedAt = time.Time{}
		entry.Error = ""
	}
	state.CompletedAt = time.Time{}
	sort.Slice(state.Stages, func(i, j int) bool {
		return state.Stages[i].Stage < state.Stages[j].Stage
	})

	if err := saveWorkflowState(statePath, state); err != nil {
		return nil, err
	}
	return state, nil
}

func findOrCreateWorkflowStage(state *WorkflowState, stage int) *WorkflowStage {
	for i := range state.Stages {
		if state.Stages[i].Stage == stage {
			if strings.TrimSpace(state.Stages[i].Name) == "" {
				state.Stages[i].Name = workflowStageTitle(stage)
			}
			if strings.TrimSpace(state.Stages[i].Status) == "" {
				state.Stages[i].Status = workflowStageStatusPending
			}
			return &state.Stages[i]
		}
	}
	state.Stages = append(state.Stages, WorkflowStage{
		Stage:  stage,
		Name:   workflowStageTitle(stage),
		Status: workflowStageStatusPending,
	})
	return &state.Stages[len(state.Stages)-1]
}

func validateWorkflowStateIdentity(state *WorkflowState, projectID, taskID, template string) error {
	if state == nil {
		return errors.New("workflow state is nil")
	}
	if existingTemplate := strings.TrimSpace(state.Template); existingTemplate != "" && !strings.EqualFold(existingTemplate, template) {
		return fmt.Errorf("workflow state template mismatch: %q != %q", existingTemplate, template)
	}
	if existingProject := strings.TrimSpace(state.ProjectID); existingProject != "" && existingProject != projectID {
		return fmt.Errorf("workflow state project mismatch: %q != %q", existingProject, projectID)
	}
	if existingTask := strings.TrimSpace(state.TaskID); existingTask != "" && existingTask != taskID {
		return fmt.Errorf("workflow state task mismatch: %q != %q", existingTask, taskID)
	}
	return nil
}

func loadWorkflowState(path string) (*WorkflowState, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var state WorkflowState
	if err := yaml.Unmarshal(data, &state); err != nil {
		return nil, fmt.Errorf("decode workflow state: %w", err)
	}
	return &state, nil
}

func saveWorkflowState(path string, state *WorkflowState) error {
	if state == nil {
		return errors.New("workflow state is nil")
	}
	data, err := yaml.Marshal(state)
	if err != nil {
		return fmt.Errorf("encode workflow state: %w", err)
	}
	if err := writeWorkflowFileAtomic(path, data); err != nil {
		return fmt.Errorf("write workflow state: %w", err)
	}
	return nil
}

func writeWorkflowFileAtomic(path string, data []byte) error {
	dir := filepath.Dir(path)
	tmpFile, err := os.CreateTemp(dir, "workflow-state.*.tmp")
	if err != nil {
		return err
	}
	tmpName := tmpFile.Name()
	success := false
	defer func() {
		if success {
			return
		}
		_ = tmpFile.Close()
		_ = os.Remove(tmpName)
	}()

	if _, err := tmpFile.Write(data); err != nil {
		return err
	}
	if err := tmpFile.Sync(); err != nil {
		return err
	}
	if err := tmpFile.Chmod(0o644); err != nil {
		return err
	}
	if err := tmpFile.Close(); err != nil {
		return err
	}
	if err := os.Rename(tmpName, path); err != nil {
		if runtime.GOOS == "windows" {
			if removeErr := os.Remove(path); removeErr == nil {
				if renameErr := os.Rename(tmpName, path); renameErr == nil {
					success = true
					return nil
				}
			}
		}
		return err
	}
	success = true
	return nil
}

func postWorkflowBusMessage(busPath, projectID, taskID, runID, msgType, body string) error {
	info := &storage.RunInfo{
		ProjectID: projectID,
		TaskID:    taskID,
		RunID:     runID,
	}
	return postRunEvent(busPath, info, msgType, body)
}
