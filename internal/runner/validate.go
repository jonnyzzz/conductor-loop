package runner

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"regexp"
	"strconv"
	"strings"

	"github.com/pkg/errors"

	"github.com/jonnyzzz/conductor-loop/internal/agent"
	"github.com/jonnyzzz/conductor-loop/internal/obslog"
)

// minVersions defines the minimum supported CLI version per agent type.
var minVersions = map[string][3]int{
	"claude": {1, 0, 0},
	"codex":  {0, 1, 0},
	"gemini": {0, 1, 0},
}

// semverRe matches a semantic version pattern (with optional v prefix).
var semverRe = regexp.MustCompile(`v?(\d+)\.(\d+)\.(\d+)`)

// ValidateAgent checks that the CLI binary for the given agent type exists in
// PATH and attempts to detect its version. REST-based agents (perplexity, xai)
// are skipped since they do not require a local CLI binary.
func ValidateAgent(ctx context.Context, agentType string) error {
	clean := strings.ToLower(strings.TrimSpace(agentType))
	if clean == "" {
		return errors.New("agent type is empty")
	}

	if isRestAgent(clean) {
		return nil
	}

	command := cliCommand(clean)
	if command == "" {
		return errors.Errorf("unknown cli agent type %q", clean)
	}

	path, err := exec.LookPath(command)
	if err != nil {
		return errors.Errorf("agent cli %q not found in PATH: %v", command, err)
	}

	version, err := agent.DetectCLIVersion(ctx, path)
	if err != nil {
		obslog.Log(log.Default(), "WARN", "runner", "agent_version_detect_failed",
			obslog.F("agent_type", clean),
			obslog.F("command", command),
			obslog.F("path", path),
			obslog.F("error", err),
		)
		return nil
	}

	obslog.Log(log.Default(), "INFO", "runner", "agent_version_detected",
		obslog.F("agent_type", clean),
		obslog.F("version", version),
		obslog.F("path", path),
	)

	if minVer, ok := minVersions[clean]; ok {
		if !isVersionCompatible(version, minVer) {
			obslog.Log(log.Default(), "WARN", "runner", "agent_version_below_minimum",
				obslog.F("agent_type", clean),
				obslog.F("version", version),
				obslog.F("min_version", fmt.Sprintf("%d.%d.%d", minVer[0], minVer[1], minVer[2])),
			)
		}
	}

	return nil
}

// parseVersion extracts major, minor, patch integers from a version string.
// It handles formats like "claude 1.0.0", "v1.2.3", "codex 0.5.3-beta", etc.
func parseVersion(raw string) (major, minor, patch int, err error) {
	m := semverRe.FindStringSubmatch(raw)
	if m == nil {
		return 0, 0, 0, fmt.Errorf("no semantic version found in %q", raw)
	}
	major, _ = strconv.Atoi(m[1])
	minor, _ = strconv.Atoi(m[2])
	patch, _ = strconv.Atoi(m[3])
	return major, minor, patch, nil
}

// isVersionCompatible returns true if the detected version string meets or
// exceeds the given minimum version. Returns true (compatible) if the version
// cannot be parsed, since version detection is best-effort.
func isVersionCompatible(detected string, minVersion [3]int) bool {
	major, minor, patch, err := parseVersion(detected)
	if err != nil {
		return true
	}
	if major != minVersion[0] {
		return major > minVersion[0]
	}
	if minor != minVersion[1] {
		return minor > minVersion[1]
	}
	return patch >= minVersion[2]
}

// cliCommand returns the CLI binary name for a given agent type.
func cliCommand(agentType string) string {
	switch strings.ToLower(agentType) {
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

// ValidateToken checks if a token is configured for the given agent type.
// It returns a warning (non-nil error) if the token appears to be missing,
// but callers should treat this as advisory only.
func ValidateToken(agentType string, token string) error {
	agentType = strings.ToLower(strings.TrimSpace(agentType))

	// For REST agents, check the provided token
	if isRestAgent(agentType) {
		if strings.TrimSpace(token) == "" {
			return fmt.Errorf("agent %q: no token configured; set token in config or via environment", agentType)
		}
		return nil
	}

	// For CLI agents, check environment variable
	envVar := tokenEnvVar(agentType)
	if envVar != "" && os.Getenv(envVar) == "" {
		// Check if there's a token in config that will be injected
		if strings.TrimSpace(token) == "" {
			return fmt.Errorf("agent %q: %s not set and no token in config", agentType, envVar)
		}
	}
	return nil
}
