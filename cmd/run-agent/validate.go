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
		checkTokens  bool
	)

	cmd := &cobra.Command{
		Use:   "validate",
		Short: "Validate conductor configuration and agent availability",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runValidate(configPath, rootDir, agentFilter, checkNetwork, checkTokens)
		},
	}

	cmd.Flags().StringVar(&configPath, "config", "", "config file path")
	cmd.Flags().StringVar(&rootDir, "root", "", "root directory to validate")
	cmd.Flags().StringVar(&agentFilter, "agent", "", "validate only this agent (default: all)")
	cmd.Flags().BoolVar(&checkNetwork, "check-network", false, "run network connectivity test for REST agents")
	cmd.Flags().BoolVar(&checkTokens, "check-tokens", false, "verify token files are readable and non-empty")

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

func runValidate(configPath, rootDir, agentFilter string, checkNetwork, checkTokens bool) error {
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
	// Always use LoadConfigForServer here — the validate command performs its own
	// agent-level checks and reports them user-visibly. Using the strict LoadConfig
	// would cause validate to hard-fail on a fresh template home config that has no
	// agents yet (which is the expected default state on first run).
	var cfg *config.Config
	if configPath != "" {
		var err error
		cfg, err = config.LoadConfigForServer(configPath)
		if err != nil {
			fmt.Printf("Config: FAIL\n  %v\n\n", err)
			hasFailure = true
		}
	}

	// Validate agents.
	var names []string
	if cfg != nil {
		names = sortedAgentNames(cfg.Agents, agentFilter)
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
				result := validateSingleAgent(ctx, name, cfg.Agents[name], checkTokens)
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

	// Deep token checks.
	if checkTokens && cfg != nil && len(names) > 0 {
		fmt.Println()
		fmt.Println("Token checks:")
		for _, name := range names {
			desc, ok := checkToken(name, cfg.Agents[name])
			fmt.Printf("  Agent %-14s %s\n", name+":", desc)
			if !ok {
				hasFailure = true
			}
		}
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

// checkToken performs a deep accessibility check on the token for the given agent.
// It returns a human-readable description and whether the token is accessible.
func checkToken(agentName string, agentCfg config.AgentConfig) (string, bool) {
	agType := strings.ToLower(agentCfg.Type)

	// Token set directly in config (also catches CONDUCTOR_AGENT_<NAME>_TOKEN overrides).
	if agentCfg.Token != "" {
		return "token [OK]", true
	}

	// Token file configured — check existence, readability, and non-empty content.
	if agentCfg.TokenFile != "" {
		data, err := os.ReadFile(agentCfg.TokenFile)
		if err != nil {
			if os.IsNotExist(err) {
				return fmt.Sprintf("token_file %s [MISSING - file not found]", agentCfg.TokenFile), false
			}
			return fmt.Sprintf("token_file %s [ERROR - %v]", agentCfg.TokenFile, err), false
		}
		if strings.TrimSpace(string(data)) == "" {
			return fmt.Sprintf("token_file %s [EMPTY]", agentCfg.TokenFile), false
		}
		return fmt.Sprintf("token_file %s [OK]", agentCfg.TokenFile), true
	}

	// No explicit token — check the well-known env var for this agent type.
	envVar := validateTokenEnvVar(agType)
	if envVar != "" {
		if os.Getenv(envVar) != "" {
			return fmt.Sprintf("env %s [OK]", envVar), true
		}
		return fmt.Sprintf("env %s [NOT SET]", envVar), false
	}

	return "token [NOT SET]", false
}

func validateSingleAgent(ctx context.Context, name string, agentCfg config.AgentConfig, skipTokenCheck bool) agentCheckResult {
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

	// Check token availability — skipped when --check-tokens is set because the
	// deep token check section handles this separately.
	if !skipTokenCheck {
		tokenErr := runner.ValidateToken(agentCfg.Type, agentCfg.Token)
		if tokenErr != nil {
			result.ok = false
		}
		result.tokenStatus = computeTokenStatus(agType, agentCfg.Token, tokenErr == nil)
	}

	return result
}

func printAgentResult(r agentCheckResult) {
	symbol := "✓"
	if !r.ok {
		symbol = "✗"
	}
	if r.tokenStatus != "" {
		if r.version != "" {
			fmt.Printf("  %s %-12s %-10s (%s, token: %s)\n", symbol, r.name, r.version, r.cliStatus, r.tokenStatus)
		} else {
			fmt.Printf("  %s %-12s (%s, token: %s)\n", symbol, r.name, r.cliStatus, r.tokenStatus)
		}
	} else {
		if r.version != "" {
			fmt.Printf("  %s %-12s %-10s (%s)\n", symbol, r.name, r.version, r.cliStatus)
		} else {
			fmt.Printf("  %s %-12s (%s)\n", symbol, r.name, r.cliStatus)
		}
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
