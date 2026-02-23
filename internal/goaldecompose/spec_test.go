package goaldecompose

import (
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"
)

func TestBuildSpec_DeterministicAndValid(t *testing.T) {
	opts := BuildOptions{
		ProjectID:   "my-project",
		GoalText:    "Implement deterministic goal decomposition\nwith reproducible output.",
		GoalMode:    "inline",
		Strategy:    DefaultStrategy,
		Template:    DefaultTemplate,
		MaxParallel: 4,
	}

	a, err := BuildSpec(opts)
	if err != nil {
		t.Fatalf("build spec A: %v", err)
	}
	b, err := BuildSpec(opts)
	if err != nil {
		t.Fatalf("build spec B: %v", err)
	}

	if a.WorkflowID != b.WorkflowID {
		t.Fatalf("workflow_id mismatch: %q != %q", a.WorkflowID, b.WorkflowID)
	}
	if a.Goal.DigestSHA256 != b.Goal.DigestSHA256 {
		t.Fatalf("goal digest mismatch: %q != %q", a.Goal.DigestSHA256, b.Goal.DigestSHA256)
	}
	if len(a.Tasks) != len(defaultBlueprint) {
		t.Fatalf("task count = %d, want %d", len(a.Tasks), len(defaultBlueprint))
	}

	seen := map[string]struct{}{}
	for _, task := range a.Tasks {
		if _, ok := seen[task.TaskID]; ok {
			t.Fatalf("duplicate task id: %s", task.TaskID)
		}
		seen[task.TaskID] = struct{}{}
	}

	for _, task := range a.Tasks {
		for _, dep := range task.DependsOn {
			if _, ok := seen[dep]; !ok {
				t.Fatalf("task %s depends on unknown task %s", task.TaskID, dep)
			}
		}
	}
}

func TestBuildSpec_ValidatesInput(t *testing.T) {
	if _, err := BuildSpec(BuildOptions{GoalText: "x"}); err == nil || !strings.Contains(err.Error(), "project is required") {
		t.Fatalf("expected project validation error, got %v", err)
	}

	if _, err := BuildSpec(BuildOptions{ProjectID: "p", GoalText: " "}); err == nil || !strings.Contains(err.Error(), "goal is required") {
		t.Fatalf("expected goal validation error, got %v", err)
	}

	if _, err := BuildSpec(BuildOptions{ProjectID: "p", GoalText: "x", Strategy: "manual"}); err == nil || !strings.Contains(err.Error(), "unsupported strategy") {
		t.Fatalf("expected strategy validation error, got %v", err)
	}

	if _, err := BuildSpec(BuildOptions{ProjectID: "p", GoalText: "x", MaxParallel: MaxParallelLimit + 1}); err == nil || !strings.Contains(err.Error(), "exceeds limit") {
		t.Fatalf("expected max-parallel validation error, got %v", err)
	}
}

func TestBuildSpec_Defaults(t *testing.T) {
	spec, err := BuildSpec(BuildOptions{ProjectID: "p", GoalText: "x"})
	if err != nil {
		t.Fatalf("build spec: %v", err)
	}
	if spec.Strategy != DefaultStrategy {
		t.Fatalf("strategy = %q, want %q", spec.Strategy, DefaultStrategy)
	}
	if spec.Template != DefaultTemplate {
		t.Fatalf("template = %q, want %q", spec.Template, DefaultTemplate)
	}
	if spec.MaxParallel != DefaultMaxParallel {
		t.Fatalf("max_parallel = %d, want %d", spec.MaxParallel, DefaultMaxParallel)
	}
	if spec.Semantics.RLMDocument != defaultRLMSemanticsDocument {
		t.Fatalf("rlm_document = %q, want %q", spec.Semantics.RLMDocument, defaultRLMSemanticsDocument)
	}
	if spec.Semantics.ThePromptDocument != thePromptSemanticsDocument {
		t.Fatalf("the_prompt_document = %q, want %q", spec.Semantics.ThePromptDocument, thePromptSemanticsDocument)
	}
}

func TestBuildSpec_RLMSemanticsOverrideFromEnv(t *testing.T) {
	t.Setenv(rlmSemanticsDocumentEnvVar, "/tmp/example/../RLM.md")

	spec, err := BuildSpec(BuildOptions{ProjectID: "p", GoalText: "x"})
	if err != nil {
		t.Fatalf("build spec: %v", err)
	}

	want := filepath.ToSlash(filepath.Clean("/tmp/example/../RLM.md"))
	if spec.Semantics.RLMDocument != want {
		t.Fatalf("rlm_document = %q, want %q", spec.Semantics.RLMDocument, want)
	}
}

func TestEncodeSpec_Formats(t *testing.T) {
	spec, err := BuildSpec(BuildOptions{ProjectID: "p", GoalText: "x"})
	if err != nil {
		t.Fatalf("build spec: %v", err)
	}

	jsonData, err := EncodeSpec(spec, OutputFormatJSON)
	if err != nil {
		t.Fatalf("encode json: %v", err)
	}
	var decoded WorkflowSpec
	if err := json.Unmarshal(jsonData, &decoded); err != nil {
		t.Fatalf("decode json: %v", err)
	}
	if decoded.WorkflowID != spec.WorkflowID {
		t.Fatalf("decoded workflow_id = %q, want %q", decoded.WorkflowID, spec.WorkflowID)
	}

	yamlData, err := EncodeSpec(spec, OutputFormatYAML)
	if err != nil {
		t.Fatalf("encode yaml: %v", err)
	}
	if !strings.Contains(string(yamlData), "workflow_id:") {
		t.Fatalf("yaml output missing workflow_id: %q", string(yamlData))
	}
}

func TestOutputFormatFromPath(t *testing.T) {
	if got := OutputFormatFromPath("workflow.json"); got != OutputFormatJSON {
		t.Fatalf("format from .json = %q, want %q", got, OutputFormatJSON)
	}
	if got := OutputFormatFromPath("workflow.yaml"); got != OutputFormatYAML {
		t.Fatalf("format from .yaml = %q, want %q", got, OutputFormatYAML)
	}
	if got := OutputFormatFromPath("workflow"); got != OutputFormatYAML {
		t.Fatalf("format from no extension = %q, want %q", got, OutputFormatYAML)
	}
}
