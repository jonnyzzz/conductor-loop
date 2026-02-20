package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"regexp"
	"sort"
	"strings"

	"github.com/jonnyzzz/conductor-loop/internal/agent"
	"github.com/jonnyzzz/conductor-loop/internal/config"
	"github.com/jonnyzzz/conductor-loop/internal/runner"
	"github.com/spf13/cobra"
)

// validateVersionRe matches a semantic version number in CLI output.
var validateVersionRe = regexp.MustCompile(`v?(\d+\.\d+\.\d+)`)

func newValidateCmd() *cobra.Command {
	var (
		configPath   string
		rootDir      string
		agentFilter  string
		checkNetwork bool
	)

	cmd := &cobra.Command{
		Use:   "validate",
		Short: "Validate conductor configuration and agent availability",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runValidate(configPath, rootDir, agentFilter, checkNetwork)
		},
	}

	cmd.Flags().StringVar(&configPath, "config", "", "config file path")
	cmd.Flags().StringVar(&rootDir, "root", "", "root directory to validate")
	cmd.Flags().StringVar(&agentFilter, "agent", "", "validate only this agent (default: all)")
	cmd.Flags().BoolVar(&checkNetwork, "check-network", false, "run network connectivity test for REST agents")

	return cmd
}

// agentCheckResult holds the validation outcome for a single agent.
type agentCheckResult struct {
	name        string
	ok          bool
	cliStatus   string
	version     string
	tokenStatus string
}

func runValidate(configPath, rootDir, agentFilter string, checkNetwork bool) error {
	fmt.Println("Conductor Loop Configuration Validator")
	fmt.Println()

	hasFailure := false

	// Auto-discover config if not specified.
	if configPath == "" {
		found, err := config.FindDefaultConfig()
		if err != nil {
			fmt.Printf("Config: error: %v\n", err)
			hasFailure = true
		} else {
			configPath = found
		}
	}

	if configPath == "" {
		fmt.Println("Config: (not found)")
	} else {
		fmt.Printf("Config: %s\n", configPath)
	}

	if rootDir != "" {
		fmt.Printf("Root:   %s\n", rootDir)
	}
	fmt.Println()

	// Validate root directory if provided.
	if rootDir != "" {
		if err := checkRootDir(rootDir); err != nil {
			fmt.Printf("Root: FAIL (%v)\n\n", err)
			hasFailure = true
		} else {
			fmt.Println("Root: OK")
			fmt.Println()
		}
	}

	// Load config if a path was found or specified.
	var cfg *config.Config
	if configPath != "" {
		var err error
		cfg, err = config.LoadConfig(configPath)
		if err != nil {
			fmt.Printf("Config: FAIL\n  %v\n\n", err)
			hasFailure = true
		}
	}

	// Validate agents.
	if cfg != nil {
		names := sortedAgentNames(cfg.Agents, agentFilter)
		if len(names) == 0 {
			if agentFilter != "" {
				fmt.Printf("Agents: (agent %q not found in config)\n", agentFilter)
				hasFailure = true
			} else {
				fmt.Println("Agents: (none configured)")
			}
		} else {
			fmt.Println("Agents:")
			ctx := context.Background()
			okCount, warnCount := 0, 0
			for _, name := range names {
				result := validateSingleAgent(ctx, name, cfg.Agents[name])
				printAgentResult(result)
				if result.ok {
					okCount++
				} else {
					warnCount++
					hasFailure = true
				}
			}
			fmt.Println()
			fmt.Printf("Validation: %d OK, %d WARNING\n", okCount, warnCount)
		}
	} else if agentFilter != "" {
		fmt.Printf("Agents: (no config loaded, cannot validate agent %q)\n", agentFilter)
	} else {
		fmt.Println("Agents: (no config loaded)")
	}

	if checkNetwork {
		fmt.Println()
		fmt.Println("Note: --check-network is not yet implemented")
	}

	if hasFailure {
		return fmt.Errorf("validation completed with warnings")
	}
	return nil
}

func validateSingleAgent(ctx context.Context, name string, agentCfg config.AgentConfig) agentCheckResult {
	result := agentCheckResult{
		name: name,
		ok:   true,
	}

	agType := strings.ToLower(agentCfg.Type)

	// Check CLI availability (REST agents have no local CLI).
	if isValidateRestAgent(agType) {
		result.cliStatus = "REST agent"
	} else {
		cliName := validateCLIName(agType)
		if cliName == "" {
			result.cliStatus = fmt.Sprintf("unknown type %q", agType)
			result.ok = false
		} else {
			path, err := exec.LookPath(cliName)
			if err != nil {
				result.cliStatus = fmt.Sprintf("CLI %q not found in PATH", cliName)
				result.ok = false
			} else {
				result.cliStatus = "CLI found"
				ver, verErr := agent.DetectCLIVersion(ctx, path)
				if verErr == nil {
					result.version = extractValidateVersion(ver)
				}
			}
		}
	}

	// Check token availability.
	tokenErr := runner.ValidateToken(agentCfg.Type, agentCfg.Token)
	if tokenErr != nil {
		result.ok = false
	}
	result.tokenStatus = computeTokenStatus(agType, agentCfg.Token, tokenErr == nil)

	return result
}

func printAgentResult(r agentCheckResult) {
	symbol := "✓"
	if !r.ok {
		symbol = "✗"
	}
	if r.version != "" {
		fmt.Printf("  %s %-12s %-10s (%s, token: %s)\n", symbol, r.name, r.version, r.cliStatus, r.tokenStatus)
	} else {
		fmt.Printf("  %s %-12s (%s, token: %s)\n", symbol, r.name, r.cliStatus, r.tokenStatus)
	}
}

// computeTokenStatus returns a short human-readable description of the token
// availability for display in the validate output.
func computeTokenStatus(agType, configToken string, ok bool) string {
	if isValidateRestAgent(agType) {
		if ok {
			return "token set"
		}
		return "not set"
	}
	envVar := validateTokenEnvVar(agType)
	if ok {
		if envVar != "" && os.Getenv(envVar) != "" {
			return envVar + " set"
		}
		if configToken != "" {
			return "config token"
		}
		return "OK"
	}
	if envVar != "" {
		return envVar + " not set"
	}
	return "not configured"
}

// checkRootDir checks that dir exists, is a directory, and is writable.
func checkRootDir(dir string) error {
	info, err := os.Stat(dir)
	if err != nil {
		return fmt.Errorf("directory not found: %w", err)
	}
	if !info.IsDir() {
		return fmt.Errorf("not a directory")
	}
	tmpPath := fmt.Sprintf("%s/.validate-%d", dir, os.Getpid())
	f, err := os.Create(tmpPath)
	if err != nil {
		return fmt.Errorf("not writable: %w", err)
	}
	_ = f.Close()
	_ = os.Remove(tmpPath)
	return nil
}

// sortedAgentNames returns the agent names from the map, optionally filtered,
// sorted alphabetically for deterministic output.
func sortedAgentNames(agents map[string]config.AgentConfig, filter string) []string {
	var names []string
	for name := range agents {
		if filter == "" || name == filter {
			names = append(names, name)
		}
	}
	sort.Strings(names)
	return names
}

// extractValidateVersion extracts the first semver number from raw CLI output,
// stripping any leading "v" prefix.
func extractValidateVersion(raw string) string {
	m := validateVersionRe.FindStringSubmatch(raw)
	if len(m) < 2 {
		return ""
	}
	return m[1]
}

func isValidateRestAgent(agType string) bool {
	switch agType {
	case "perplexity", "xai":
		return true
	default:
		return false
	}
}

func validateCLIName(agType string) string {
	switch agType {
	case "claude":
		return "claude"
	case "codex":
		return "codex"
	case "gemini":
		return "gemini"
	default:
		return ""
	}
}

func validateTokenEnvVar(agType string) string {
	switch agType {
	case "claude":
		return "ANTHROPIC_API_KEY"
	case "codex":
		return "OPENAI_API_KEY"
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
