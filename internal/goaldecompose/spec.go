// Package goaldecompose builds deterministic workflow specs from project goals.
package goaldecompose

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

const (
	// SchemaVersion is the stable schema identifier for goal decomposition output.
	SchemaVersion = "goal-workflow/v1"
	// DefaultStrategy is the only supported decomposition strategy in this skeleton.
	DefaultStrategy = "rlm"
	// DefaultTemplate is the orchestration prompt template used by generated tasks.
	DefaultTemplate = "THE_PROMPT_v5"
	// DefaultMaxParallel limits parallel fan-out in generated workflow specs.
	DefaultMaxParallel = 6
	// MaxParallelLimit reflects THE_PROMPT_v5 guidance for max parallel agents.
	MaxParallelLimit = 16

	defaultRLMSemanticsDocument = "artifacts/context/RLM.md"
	rlmSemanticsDocumentEnvVar  = "CONDUCTOR_RLM_SEMANTICS_DOC"
	thePromptSemanticsDocument  = "docs/workflow/THE_PROMPT_v5.md"
)

// BuildOptions controls deterministic workflow-spec generation from a goal.
type BuildOptions struct {
	ProjectID   string
	GoalText    string
	GoalMode    string
	GoalSource  string
	RootDir     string
	Strategy    string
	Template    string
	MaxParallel int
}

// WorkflowSpec is the deterministic workflow output for goal decomposition.
type WorkflowSpec struct {
	SchemaVersion string        `json:"schema_version" yaml:"schema_version"`
	WorkflowID    string        `json:"workflow_id" yaml:"workflow_id"`
	ProjectID     string        `json:"project_id" yaml:"project_id"`
	RootDir       string        `json:"root_dir,omitempty" yaml:"root_dir,omitempty"`
	Strategy      string        `json:"strategy" yaml:"strategy"`
	Template      string        `json:"template" yaml:"template"`
	MaxParallel   int           `json:"max_parallel" yaml:"max_parallel"`
	Semantics     SemanticsSpec `json:"semantics" yaml:"semantics"`
	Goal          GoalSpec      `json:"goal" yaml:"goal"`
	Tasks         []TaskSpec    `json:"tasks" yaml:"tasks"`
}

// SemanticsSpec records the source documents used for orchestration semantics.
type SemanticsSpec struct {
	RLMDocument       string `json:"rlm_document" yaml:"rlm_document"`
	ThePromptDocument string `json:"the_prompt_document" yaml:"the_prompt_document"`
}

// GoalSpec describes the source goal used to build the workflow.
type GoalSpec struct {
	Mode         string `json:"mode" yaml:"mode"`
	Source       string `json:"source,omitempty" yaml:"source,omitempty"`
	DigestSHA256 string `json:"digest_sha256" yaml:"digest_sha256"`
	Bytes        int    `json:"bytes" yaml:"bytes"`
	Preview      string `json:"preview" yaml:"preview"`
}

// TaskSpec is one deterministic task in the generated workflow DAG.
type TaskSpec struct {
	TaskID      string   `json:"task_id" yaml:"task_id"`
	Title       string   `json:"title" yaml:"title"`
	Agent       string   `json:"agent" yaml:"agent"`
	RolePrompt  string   `json:"role_prompt" yaml:"role_prompt"`
	PromptFile  string   `json:"prompt_file" yaml:"prompt_file"`
	DependsOn   []string `json:"depends_on,omitempty" yaml:"depends_on,omitempty"`
	RLMStep     string   `json:"rlm_step" yaml:"rlm_step"`
	PromptStage []int    `json:"the_prompt_v5_stages" yaml:"the_prompt_v5_stages"`
}

type taskBlueprint struct {
	slug       string
	title      string
	agent      string
	rolePrompt string
	rlmStep    string
	stages     []int
	dependsOn  []string
}

func resolveRLMSemanticsDocument() string {
	if override := strings.TrimSpace(os.Getenv(rlmSemanticsDocumentEnvVar)); override != "" {
		return filepath.ToSlash(filepath.Clean(override))
	}
	return defaultRLMSemanticsDocument
}

var defaultBlueprint = []taskBlueprint{
	{
		slug:       "assess-context",
		title:      "Assess context and constraints",
		agent:      "claude",
		rolePrompt: "docs/workflow/THE_PROMPT_v5_research.md",
		rlmStep:    "ASSESS",
		stages:     []int{0, 1, 2},
	},
	{
		slug:       "decompose-plan",
		title:      "Decide strategy and decompose into executable tasks",
		agent:      "codex",
		rolePrompt: "docs/workflow/THE_PROMPT_v5_orchestrator.md",
		rlmStep:    "DECIDE_DECOMPOSE",
		stages:     []int{3},
		dependsOn:  []string{"assess-context"},
	},
	{
		slug:       "implement-changes",
		title:      "Implement code changes and update tests",
		agent:      "codex",
		rolePrompt: "docs/workflow/THE_PROMPT_v5_implementation.md",
		rlmStep:    "EXECUTE",
		stages:     []int{5, 6},
		dependsOn:  []string{"decompose-plan"},
	},
	{
		slug:       "run-tests",
		title:      "Run and validate focused test/build gates",
		agent:      "codex",
		rolePrompt: "docs/workflow/THE_PROMPT_v5_test.md",
		rlmStep:    "EXECUTE",
		stages:     []int{4, 7},
		dependsOn:  []string{"implement-changes"},
	},
	{
		slug:       "review-claude",
		title:      "Independent review pass (claude)",
		agent:      "claude",
		rolePrompt: "docs/workflow/THE_PROMPT_v5_review.md",
		rlmStep:    "EXECUTE",
		stages:     []int{8, 9},
		dependsOn:  []string{"implement-changes"},
	},
	{
		slug:       "review-gemini",
		title:      "Independent review pass (gemini)",
		agent:      "gemini",
		rolePrompt: "docs/workflow/THE_PROMPT_v5_review.md",
		rlmStep:    "EXECUTE",
		stages:     []int{8, 9},
		dependsOn:  []string{"implement-changes"},
	},
	{
		slug:       "synthesize-results",
		title:      "Synthesize implementation, test, and review outcomes",
		agent:      "codex",
		rolePrompt: "docs/workflow/THE_PROMPT_v5_orchestrator.md",
		rlmStep:    "SYNTHESIZE",
		stages:     []int{10, 11},
		dependsOn:  []string{"run-tests", "review-claude", "review-gemini"},
	},
	{
		slug:       "verify-completeness",
		title:      "Verify coverage, completeness, and publish summary",
		agent:      "claude",
		rolePrompt: "docs/workflow/THE_PROMPT_v5_monitor.md",
		rlmStep:    "VERIFY",
		stages:     []int{12},
		dependsOn:  []string{"synthesize-results"},
	},
}

// BuildSpec converts a project goal into a deterministic workflow spec.
func BuildSpec(opts BuildOptions) (WorkflowSpec, error) {
	projectID := strings.TrimSpace(opts.ProjectID)
	if projectID == "" {
		return WorkflowSpec{}, fmt.Errorf("project is required")
	}

	normalizedGoal := normalizeGoalText(opts.GoalText)
	if normalizedGoal == "" {
		return WorkflowSpec{}, fmt.Errorf("goal is required")
	}

	strategy := strings.ToLower(strings.TrimSpace(opts.Strategy))
	if strategy == "" {
		strategy = DefaultStrategy
	}
	if strategy != DefaultStrategy {
		return WorkflowSpec{}, fmt.Errorf("unsupported strategy %q", strategy)
	}

	template := strings.TrimSpace(opts.Template)
	if template == "" {
		template = DefaultTemplate
	}

	maxParallel := opts.MaxParallel
	if maxParallel <= 0 {
		maxParallel = DefaultMaxParallel
	}
	if maxParallel > MaxParallelLimit {
		return WorkflowSpec{}, fmt.Errorf("max parallel %d exceeds limit %d", maxParallel, MaxParallelLimit)
	}

	goalMode := strings.TrimSpace(opts.GoalMode)
	if goalMode == "" {
		goalMode = "inline"
	}

	goalSource := strings.TrimSpace(opts.GoalSource)
	if goalSource != "" {
		goalSource = filepath.ToSlash(filepath.Clean(goalSource))
	}

	hash := digestGoal(projectID, strategy, template, normalizedGoal)
	workflowID := fmt.Sprintf("workflow-%s-%s", sanitizeID(projectID), hash[:12])
	taskDate, taskTime := deriveTaskClock(hash)

	slugToTaskID := make(map[string]string, len(defaultBlueprint))
	tasks := make([]TaskSpec, 0, len(defaultBlueprint))
	for idx, blueprint := range defaultBlueprint {
		taskID := fmt.Sprintf("task-%s-%s-%02d-%s", taskDate, taskTime, idx+1, blueprint.slug)
		slugToTaskID[blueprint.slug] = taskID
		tasks = append(tasks, TaskSpec{
			TaskID:      taskID,
			Title:       blueprint.title,
			Agent:       blueprint.agent,
			RolePrompt:  blueprint.rolePrompt,
			PromptFile:  filepath.ToSlash(filepath.Join("workflow", workflowID, "prompts", fmt.Sprintf("%02d-%s.md", idx+1, blueprint.slug))),
			RLMStep:     blueprint.rlmStep,
			PromptStage: append([]int(nil), blueprint.stages...),
		})
	}

	for idx, blueprint := range defaultBlueprint {
		if len(blueprint.dependsOn) == 0 {
			continue
		}
		dependsOn := make([]string, 0, len(blueprint.dependsOn))
		for _, depSlug := range blueprint.dependsOn {
			depTaskID, ok := slugToTaskID[depSlug]
			if !ok {
				return WorkflowSpec{}, fmt.Errorf("unknown dependency %q for %q", depSlug, blueprint.slug)
			}
			dependsOn = append(dependsOn, depTaskID)
		}
		tasks[idx].DependsOn = dependsOn
	}

	spec := WorkflowSpec{
		SchemaVersion: SchemaVersion,
		WorkflowID:    workflowID,
		ProjectID:     projectID,
		RootDir:       strings.TrimSpace(opts.RootDir),
		Strategy:      strategy,
		Template:      template,
		MaxParallel:   maxParallel,
		Semantics: SemanticsSpec{
			RLMDocument:       resolveRLMSemanticsDocument(),
			ThePromptDocument: thePromptSemanticsDocument,
		},
		Goal: GoalSpec{
			Mode:         goalMode,
			Source:       goalSource,
			DigestSHA256: hash,
			Bytes:        len(normalizedGoal),
			Preview:      goalPreview(normalizedGoal),
		},
		Tasks: tasks,
	}

	return spec, nil
}

// OutputFormat selects encoding for workflow spec serialization.
type OutputFormat string

const (
	// OutputFormatYAML encodes output as YAML.
	OutputFormatYAML OutputFormat = "yaml"
	// OutputFormatJSON encodes output as JSON.
	OutputFormatJSON OutputFormat = "json"
)

// OutputFormatFromPath infers output format from file extension.
func OutputFormatFromPath(path string) OutputFormat {
	switch strings.ToLower(strings.TrimSpace(filepath.Ext(path))) {
	case ".json":
		return OutputFormatJSON
	default:
		return OutputFormatYAML
	}
}

// EncodeSpec serializes spec using the requested format.
func EncodeSpec(spec WorkflowSpec, format OutputFormat) ([]byte, error) {
	switch format {
	case OutputFormatJSON:
		data, err := json.MarshalIndent(spec, "", "  ")
		if err != nil {
			return nil, err
		}
		return append(data, '\n'), nil
	case OutputFormatYAML:
		return yaml.Marshal(spec)
	default:
		return nil, fmt.Errorf("unsupported output format %q", format)
	}
}

func normalizeGoalText(text string) string {
	normalized := strings.ReplaceAll(text, "\r\n", "\n")
	normalized = strings.ReplaceAll(normalized, "\r", "\n")
	return strings.TrimSpace(normalized)
}

func digestGoal(projectID, strategy, template, goal string) string {
	material := strings.Join([]string{SchemaVersion, projectID, strategy, template, goal}, "\n")
	sum := sha256.Sum256([]byte(material))
	return hex.EncodeToString(sum[:])
}

func deriveTaskClock(hash string) (string, string) {
	digits := hashDigits(hash, 14)
	return digits[:8], digits[8:14]
}

func hashDigits(hash string, n int) string {
	if n <= 0 {
		return ""
	}
	out := make([]byte, n)
	for i := 0; i < n; i++ {
		value := hexNibbleValue(hash[i%len(hash)])
		out[i] = byte('0' + (value % 10))
	}
	return string(out)
}

func hexNibbleValue(b byte) int {
	switch {
	case b >= '0' && b <= '9':
		return int(b - '0')
	case b >= 'a' && b <= 'f':
		return int(b-'a') + 10
	case b >= 'A' && b <= 'F':
		return int(b-'A') + 10
	default:
		return 0
	}
}

func sanitizeID(value string) string {
	value = strings.ToLower(strings.TrimSpace(value))
	if value == "" {
		return "project"
	}
	var b strings.Builder
	lastDash := false
	for _, r := range value {
		switch {
		case r >= 'a' && r <= 'z':
			b.WriteRune(r)
			lastDash = false
		case r >= '0' && r <= '9':
			b.WriteRune(r)
			lastDash = false
		default:
			if b.Len() > 0 && !lastDash {
				b.WriteByte('-')
				lastDash = true
			}
		}
	}
	clean := strings.Trim(b.String(), "-")
	if clean == "" {
		return "project"
	}
	return clean
}

func goalPreview(goal string) string {
	if goal == "" {
		return ""
	}
	lines := strings.Split(goal, "\n")
	preview := strings.TrimSpace(lines[0])
	if preview == "" {
		preview = strings.TrimSpace(goal)
	}
	if len(preview) > 120 {
		return preview[:117] + "..."
	}
	return preview
}
