// Package taskdeps manages task dependency metadata and dependency graph checks.
package taskdeps

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/jonnyzzz/conductor-loop/internal/storage"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v3"
)

const (
	// ConfigFileName is the task-local file that stores dependency metadata.
	ConfigFileName = "TASK-CONFIG.yaml"
)

// Config stores task-level metadata.
type Config struct {
	DependsOn []string `yaml:"depends_on,omitempty" json:"depends_on,omitempty"`
}

// Normalize cleans and validates depends_on values.
// Each item may be a task ID or a comma-separated list of task IDs.
func Normalize(taskID string, dependsOn []string) ([]string, error) {
	taskID = strings.TrimSpace(taskID)
	seen := make(map[string]struct{}, len(dependsOn))
	out := make([]string, 0, len(dependsOn))

	for _, raw := range dependsOn {
		for _, part := range strings.Split(raw, ",") {
			dep := strings.TrimSpace(part)
			if dep == "" {
				continue
			}
			if err := validateTaskIdentifier(dep); err != nil {
				return nil, err
			}
			if taskID != "" && dep == taskID {
				return nil, fmt.Errorf("task %q cannot depend on itself", taskID)
			}
			if _, ok := seen[dep]; ok {
				continue
			}
			seen[dep] = struct{}{}
			out = append(out, dep)
		}
	}
	return out, nil
}

// ConfigPath returns the absolute path of TASK-CONFIG.yaml for a task directory.
func ConfigPath(taskDir string) string {
	return filepath.Join(taskDir, ConfigFileName)
}

// ReadDependsOn reads depends_on from TASK-CONFIG.yaml.
// If the file does not exist, it returns an empty list and nil error.
func ReadDependsOn(taskDir string) ([]string, error) {
	path := ConfigPath(taskDir)
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, errors.Wrap(err, "read task config")
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, errors.Wrap(err, "unmarshal task config")
	}

	dependsOn, err := Normalize("", cfg.DependsOn)
	if err != nil {
		return nil, err
	}
	return dependsOn, nil
}

// WriteDependsOn writes depends_on to TASK-CONFIG.yaml.
// When depends_on is empty, TASK-CONFIG.yaml is removed if present.
func WriteDependsOn(taskDir string, dependsOn []string) error {
	path := ConfigPath(taskDir)
	if len(dependsOn) == 0 {
		if err := os.Remove(path); err != nil && !os.IsNotExist(err) {
			return errors.Wrap(err, "remove task config")
		}
		return nil
	}

	cfg := Config{DependsOn: dependsOn}
	data, err := yaml.Marshal(&cfg)
	if err != nil {
		return errors.Wrap(err, "marshal task config")
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		return errors.Wrap(err, "write task config")
	}
	return nil
}

// ValidateNoCycle checks that setting depends_on for taskID in projectID does not
// create a dependency cycle.
func ValidateNoCycle(rootDir, projectID, taskID string, dependsOn []string) error {
	graph, err := loadProjectGraph(rootDir, projectID)
	if err != nil {
		return err
	}
	graph[taskID] = dependsOn

	cycle := findCycle(graph)
	if len(cycle) == 0 {
		return nil
	}
	return fmt.Errorf("dependency cycle detected: %s", strings.Join(cycle, " -> "))
}

// BlockedBy resolves unresolved dependencies for a task.
// A dependency is considered resolved when:
// - dependency task directory exists, and
// - DONE file is present, OR latest run status is completed and no run is running.
func BlockedBy(rootDir, projectID string, dependsOn []string) ([]string, error) {
	blocked := make([]string, 0, len(dependsOn))
	for _, depID := range dependsOn {
		ready, err := isDependencyReady(rootDir, projectID, depID)
		if err != nil {
			return nil, err
		}
		if !ready {
			blocked = append(blocked, depID)
		}
	}
	return blocked, nil
}

// FindProjectDir locates the project directory for projectID under rootDir.
func FindProjectDir(rootDir, projectID string) (string, bool) {
	if dir := filepath.Join(rootDir, projectID); isDir(dir) {
		return dir, true
	}
	if dir := filepath.Join(rootDir, "runs", projectID); isDir(dir) {
		return dir, true
	}
	return "", false
}

// FindTaskDir locates the task directory for (projectID, taskID) under rootDir.
func FindTaskDir(rootDir, projectID, taskID string) (string, bool) {
	if dir := filepath.Join(rootDir, projectID, taskID); isDir(dir) {
		return dir, true
	}
	if dir := filepath.Join(rootDir, "runs", projectID, taskID); isDir(dir) {
		return dir, true
	}

	var found string
	_ = filepath.WalkDir(rootDir, func(path string, d os.DirEntry, err error) error {
		if err != nil || !d.IsDir() {
			return nil
		}
		if filepath.Base(path) == taskID && filepath.Base(filepath.Dir(path)) == projectID {
			found = path
			return filepath.SkipAll
		}

		rel, relErr := filepath.Rel(rootDir, path)
		if relErr != nil || rel == "." {
			return nil
		}
		depth := strings.Count(filepath.ToSlash(rel), "/") + 1
		if depth >= 3 {
			return filepath.SkipDir
		}
		return nil
	})
	return found, found != ""
}

func loadProjectGraph(rootDir, projectID string) (map[string][]string, error) {
	graph := make(map[string][]string)
	projectDir, ok := FindProjectDir(rootDir, projectID)
	if !ok {
		return graph, nil
	}

	entries, err := os.ReadDir(projectDir)
	if err != nil {
		return nil, errors.Wrap(err, "read project directory")
	}
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		taskID := entry.Name()
		taskDir := filepath.Join(projectDir, taskID)
		hasTaskMD := isFile(filepath.Join(taskDir, "TASK.md"))
		hasTaskConfig := isFile(filepath.Join(taskDir, ConfigFileName))
		if !hasTaskMD && !hasTaskConfig {
			continue
		}

		dependsOn, err := ReadDependsOn(taskDir)
		if err != nil {
			return nil, errors.Wrapf(err, "read depends_on for task %s", taskID)
		}
		graph[taskID] = dependsOn
	}

	return graph, nil
}

func findCycle(graph map[string][]string) []string {
	nodes := make(map[string]struct{}, len(graph))
	for node, deps := range graph {
		nodes[node] = struct{}{}
		for _, dep := range deps {
			nodes[dep] = struct{}{}
		}
	}

	keys := make([]string, 0, len(nodes))
	for k := range nodes {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	state := make(map[string]int, len(keys)) // 0=unseen,1=visiting,2=done
	stack := make([]string, 0, len(keys))
	var cycle []string

	var visit func(string) bool
	visit = func(node string) bool {
		state[node] = 1
		stack = append(stack, node)

		for _, dep := range graph[node] {
			switch state[dep] {
			case 0:
				if visit(dep) {
					return true
				}
			case 1:
				idx := -1
				for i, value := range stack {
					if value == dep {
						idx = i
						break
					}
				}
				if idx >= 0 {
					cycle = append(cycle[:0], stack[idx:]...)
					cycle = append(cycle, dep)
				}
				return true
			}
		}

		stack = stack[:len(stack)-1]
		state[node] = 2
		return false
	}

	for _, node := range keys {
		if state[node] != 0 {
			continue
		}
		if visit(node) {
			return cycle
		}
	}
	return nil
}

func isDependencyReady(rootDir, projectID, depID string) (bool, error) {
	taskDir, ok := FindTaskDir(rootDir, projectID, depID)
	if !ok {
		return false, nil
	}
	if isFile(filepath.Join(taskDir, "DONE")) {
		return true, nil
	}

	runsDir := filepath.Join(taskDir, "runs")
	entries, err := os.ReadDir(runsDir)
	if err != nil {
		if os.IsNotExist(err) {
			return false, nil
		}
		return false, errors.Wrapf(err, "read runs for dependency %s", depID)
	}

	latestRunID := ""
	latestStatus := ""
	hasRunning := false

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		runInfoPath := filepath.Join(runsDir, entry.Name(), "run-info.yaml")
		info, err := storage.ReadRunInfo(runInfoPath)
		if err != nil {
			if os.IsNotExist(errors.Cause(err)) || os.IsNotExist(err) {
				continue
			}
			return false, errors.Wrapf(err, "read run-info for dependency %s", depID)
		}
		if strings.TrimSpace(info.Status) == storage.StatusRunning {
			hasRunning = true
		}
		if entry.Name() > latestRunID {
			latestRunID = entry.Name()
			latestStatus = strings.TrimSpace(info.Status)
		}
	}

	if hasRunning {
		return false, nil
	}
	if latestRunID == "" {
		return false, nil
	}
	return latestStatus == storage.StatusCompleted, nil
}

func isDir(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

func isFile(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

func validateTaskIdentifier(taskID string) error {
	trimmed := strings.TrimSpace(taskID)
	if trimmed == "" {
		return errors.New("dependency task id is empty")
	}
	if strings.Contains(trimmed, "/") || strings.Contains(trimmed, "\\") {
		return fmt.Errorf("dependency task id %q must not contain path separators", trimmed)
	}
	if strings.Contains(trimmed, "..") {
		return fmt.Errorf("dependency task id %q must not contain ..", trimmed)
	}
	return nil
}
