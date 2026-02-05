// Package runner manages agent process execution for the orchestration subsystem.
package runner

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"sync/atomic"
	"time"

	"github.com/jonnyzzz/conductor-loop/internal/config"
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
		trimmed = filepath.Join(home, "run-agent")
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
	offset := atomic.AddUint64(&runCounter, 1)
	if offset != 0 {
		now = now.Add(time.Duration(offset%10000) * 100 * time.Microsecond)
	}
	stamp := now.UTC().Format("20060102-1504050000")
	return fmt.Sprintf("%s-%d", stamp, pid)
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

func buildPrompt(taskDir, runDir, prompt string) string {
	trimmed := strings.TrimSpace(prompt)
	preamble := fmt.Sprintf("TASK_FOLDER=%s\nRUN_FOLDER=%s\nWrite output.md to %s\n\n", taskDir, runDir, filepath.Join(runDir, "output.md"))
	if trimmed == "" {
		return preamble
	}
	if !strings.HasSuffix(trimmed, "\n") {
		trimmed += "\n"
	}
	return preamble + trimmed
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
	env["PATH"] = execDir + string(os.PathListSeparator) + existing
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
