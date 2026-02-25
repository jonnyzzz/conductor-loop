// Package runner manages agent process execution for the orchestration subsystem.
package runner

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync/atomic"
	"time"

	"github.com/jonnyzzz/conductor-loop/internal/config"
	"github.com/jonnyzzz/conductor-loop/internal/obslog"
	"github.com/pkg/errors"
)

var runCounter uint64

type agentSelection struct {
	Name   string
	Type   string
	Config config.AgentConfig
}

func resolveRootDir(root string) (string, error) {
	trimmed := strings.TrimSpace(root)
	if trimmed == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", errors.Wrap(err, "resolve home dir")
		}
		trimmed = filepath.Join(home, ".run-agent", "runs")
	}
	abs, err := filepath.Abs(trimmed)
	if err != nil {
		return "", errors.Wrap(err, "resolve root dir")
	}
	return abs, nil
}

func resolveTaskDir(rootDir, projectID, taskID string) (string, error) {
	if strings.TrimSpace(projectID) == "" {
		return "", errors.New("project id is empty")
	}
	if strings.TrimSpace(taskID) == "" {
		return "", errors.New("task id is empty")
	}
	if strings.TrimSpace(rootDir) == "" {
		return "", errors.New("root dir is empty")
	}
	return filepath.Join(rootDir, projectID, taskID), nil
}

func loadConfig(path string) (*config.Config, error) {
	trimmed := strings.TrimSpace(path)
	if trimmed == "" {
		return nil, nil
	}
	cfg, err := config.LoadConfig(trimmed)
	if err != nil {
		return nil, errors.Wrap(err, "load config")
	}
	return cfg, nil
}

func selectAgent(cfg *config.Config, preferred string) (agentSelection, error) {
	preferred = strings.TrimSpace(preferred)
	if cfg == nil {
		if preferred == "" {
			return agentSelection{}, errors.New("agent is empty")
		}
		return agentSelection{Name: preferred, Type: strings.ToLower(preferred)}, nil
	}
	if preferred != "" {
		if agentCfg, ok := cfg.Agents[preferred]; ok {
			return agentSelection{Name: preferred, Type: agentCfg.Type, Config: agentCfg}, nil
		}
		for name, agentCfg := range cfg.Agents {
			if agentCfg.Type == strings.ToLower(preferred) {
				return agentSelection{Name: name, Type: agentCfg.Type, Config: agentCfg}, nil
			}
		}
		return agentSelection{}, fmt.Errorf("unknown agent %q", preferred)
	}

	if cfg.Defaults.Agent != "" {
		return selectAgent(cfg, cfg.Defaults.Agent)
	}
	if len(cfg.Agents) == 0 {
		return agentSelection{}, errors.New("no agents configured")
	}
	keys := make([]string, 0, len(cfg.Agents))
	for key := range cfg.Agents {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	name := keys[0]
	agentCfg := cfg.Agents[name]
	return agentSelection{Name: name, Type: agentCfg.Type, Config: agentCfg}, nil
}

func newRunID(now time.Time, pid int) string {
	seq := atomic.AddUint64(&runCounter, 1)
	stamp := now.UTC().Format("20060102-1504050000")
	return fmt.Sprintf("%s-%d-%d", stamp, pid, seq)
}

func createRunDir(runsDir string) (string, string, error) {
	if strings.TrimSpace(runsDir) == "" {
		return "", "", errors.New("runs directory is empty")
	}
	pid := os.Getpid()
	runID := newRunID(time.Now().UTC(), pid)
	runDir := filepath.Join(runsDir, runID)
	if err := os.Mkdir(runDir, 0o755); err == nil {
		return runID, runDir, nil
	} else if !os.IsExist(err) {
		return "", "", errors.Wrap(err, "create run directory")
	}
	fallback, err := os.MkdirTemp(runsDir, runID+"-")
	if err != nil {
		return "", "", errors.Wrap(err, "create run directory")
	}
	return filepath.Base(fallback), fallback, nil
}

// AllocateRunDir pre-creates a new run directory under runsDir and returns its ID and path.
// This allows callers to obtain a run ID before the agent process starts.
func AllocateRunDir(runsDir string) (runID, runDir string, err error) {
	return createRunDir(runsDir)
}

// PromptParams holds the values used to build the agent prompt preamble.
type PromptParams struct {
	TaskDir        string
	RunDir         string
	ProjectID      string
	TaskID         string
	RunID          string
	ParentRunID    string
	MessageBusPath string // absolute path to TASK-MESSAGE-BUS.md
	ConductorURL   string // e.g. "http://127.0.0.1:14355"
	RepoRoot       string // absolute path to conductor-loop repo root
}

func buildPrompt(params PromptParams, prompt string) string {
	var b strings.Builder

	// --- Environment variables ---
	fmt.Fprintf(&b, "TASK_FOLDER=%s\n", params.TaskDir)
	fmt.Fprintf(&b, "RUN_FOLDER=%s\n", params.RunDir)
	fmt.Fprintf(&b, "JRUN_PROJECT_ID=%s\n", params.ProjectID)
	fmt.Fprintf(&b, "JRUN_TASK_ID=%s\n", params.TaskID)
	fmt.Fprintf(&b, "JRUN_ID=%s\n", params.RunID)
	if strings.TrimSpace(params.ParentRunID) != "" {
		fmt.Fprintf(&b, "JRUN_PARENT_ID=%s\n", params.ParentRunID)
	}
	if params.MessageBusPath != "" {
		fmt.Fprintf(&b, "MESSAGE_BUS=%s\n", params.MessageBusPath)
	}
	if params.ConductorURL != "" {
		fmt.Fprintf(&b, "CONDUCTOR_URL=%s\n", params.ConductorURL)
	}
	fmt.Fprintf(&b, "Write output.md to %s\n", filepath.Join(params.RunDir, "output.md"))

	// --- Message Bus usage ---
	b.WriteString("\n## Message Bus\n")
	b.WriteString("Report progress using:\n")
	b.WriteString("  run-agent bus post --type PROGRESS --body \"your message\"\n")
	b.WriteString("  run-agent bus post --type FACT --body \"key result\"\n")
	b.WriteString("  run-agent bus post --type ERROR --body \"what failed\"\n")
	b.WriteString("  run-agent bus post --type DECISION --body \"what was decided\"\n")
	b.WriteString("Types: FACT, PROGRESS, DECISION, ERROR, QUESTION, INFO\n")
	b.WriteString("The MESSAGE_BUS env var is set; run-agent bus post uses it automatically.\n")

	// --- Sub-agent spawning ---
	b.WriteString("\n## Sub-Agent Spawning (RLM Pattern)\n")
	b.WriteString("For complex tasks, decompose using run-agent:\n")
	b.WriteString("  run-agent job --project $JRUN_PROJECT_ID --task $JRUN_TASK_ID \\\n")
	b.WriteString("    --parent-run-id $JRUN_ID --agent claude --prompt \"sub-task description\"\n")
	b.WriteString("Run multiple sub-agents in PARALLEL for independent sub-tasks.\n")
	b.WriteString("Use RLM (Recursive Language Model) decomposition:\n")
	b.WriteString("  1. ASSESS context size and task complexity\n")
	b.WriteString("  2. DECOMPOSE into independent sub-tasks at natural boundaries\n")
	b.WriteString("  3. EXECUTE sub-agents in parallel\n")
	b.WriteString("  4. SYNTHESIZE results\n")

	// --- Reference docs ---
	if params.RepoRoot != "" {
		promptV5 := filepath.Join(params.RepoRoot, "docs", "workflow", "THE_PROMPT_v5_conductor.md")
		if _, err := os.Stat(promptV5); err == nil {
			fmt.Fprintf(&b, "\n## Methodology\nRead %s for the full orchestration methodology.\n", promptV5)
		}
		agentsMd := filepath.Join(params.RepoRoot, "AGENTS.md")
		if _, err := os.Stat(agentsMd); err == nil {
			fmt.Fprintf(&b, "Read %s for project conventions.\n", agentsMd)
		}
	}

	// --- DONE file ---
	b.WriteString("\n## Completion\n")
	fmt.Fprintf(&b, "When done, create: %s/DONE\n", params.TaskDir)
	b.WriteString("This signals the conductor loop to stop restarting this task.\n")

	b.WriteString("\n---\n\n")
	trimmed := strings.TrimSpace(prompt)
	if trimmed != "" {
		b.WriteString(trimmed)
		b.WriteString("\n")
	}
	return b.String()
}

// warnJRunEnvMismatch logs a warning if JRUN_* environment variables from the
// current process don't match the job's values. This helps catch misconfigured
// nested invocations where a parent runner's env leaks incorrect values.
func warnJRunEnvMismatch(projectID, taskID, runID, parentRunID string) {
	checks := []struct {
		envKey   string
		jobValue string
	}{
		{"JRUN_PROJECT_ID", projectID},
		{"JRUN_TASK_ID", taskID},
		{"JRUN_ID", runID},
		{"JRUN_PARENT_ID", parentRunID},
	}
	for _, c := range checks {
		envVal := os.Getenv(c.envKey)
		if envVal == "" {
			continue
		}
		if envVal != c.jobValue {
			obslog.Log(log.Default(), "WARN", "runner", "jrun_env_mismatch",
				obslog.F("env_key", c.envKey),
				obslog.F("env_value", envVal),
				obslog.F("expected_value", c.jobValue),
				obslog.F("project_id", projectID),
				obslog.F("task_id", taskID),
				obslog.F("run_id", runID),
				obslog.F("parent_run_id", parentRunID),
			)
		}
	}
}

func ensureDir(path string) error {
	clean := filepath.Clean(strings.TrimSpace(path))
	if clean == "." || clean == "" {
		return errors.New("directory is empty")
	}
	if err := os.MkdirAll(clean, 0o755); err != nil {
		return errors.Wrap(err, "create directory")
	}
	return nil
}

func readFileTrimmed(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", errors.Wrap(err, "read file")
	}
	trimmed := strings.TrimSpace(string(data))
	if trimmed == "" {
		return "", errors.New("file is empty")
	}
	return trimmed, nil
}

func tokenEnvVar(agentType string) string {
	switch strings.ToLower(agentType) {
	case "codex":
		return "OPENAI_API_KEY"
	case "claude":
		return "ANTHROPIC_API_KEY"
	case "gemini":
		return "GEMINI_API_KEY"
	case "perplexity":
		return "PERPLEXITY_API_KEY"
	case "xai":
		return "XAI_API_KEY"
	default:
		return ""
	}
}

func mergeEnv(base []string, overrides map[string]string) []string {
	merged := make(map[string]string, len(base)+len(overrides))
	for _, entry := range base {
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
		merged[key] = value
	}
	for key, value := range overrides {
		cleanKey := strings.TrimSpace(key)
		if cleanKey == "" {
			continue
		}
		merged[cleanKey] = value
	}
	keys := make([]string, 0, len(merged))
	for key := range merged {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	result := make([]string, 0, len(keys))
	for _, key := range keys {
		result = append(result, key+"="+merged[key])
	}
	return result
}

func prependPath(env map[string]string) error {
	if env == nil {
		return errors.New("env is nil")
	}
	execPath, err := os.Executable()
	if err != nil {
		return errors.Wrap(err, "resolve executable")
	}
	execDir := filepath.Dir(execPath)
	existing := env["PATH"]
	if existing == "" {
		existing = os.Getenv("PATH")
	}
	if existing == "" {
		env["PATH"] = execDir
		return nil
	}
	// Check if execDir is already at the front of PATH to avoid duplicates.
	sep := string(os.PathListSeparator)
	for _, dir := range strings.Split(existing, sep) {
		if dir == execDir {
			// Already present; no-op to avoid duplicate entries.
			env["PATH"] = existing
			return nil
		}
	}
	env["PATH"] = execDir + sep + existing
	return nil
}

func absPath(path string) (string, error) {
	trimmed := strings.TrimSpace(path)
	if trimmed == "" {
		return "", errors.New("path is empty")
	}
	abs, err := filepath.Abs(trimmed)
	if err != nil {
		return "", errors.Wrap(err, "resolve absolute path")
	}
	return abs, nil
}
