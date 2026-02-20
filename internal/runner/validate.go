package runner

import (
	"context"
	"fmt"
	"log"
	"os/exec"
	"regexp"
	"strconv"
	"strings"

	"github.com/pkg/errors"

	"github.com/jonnyzzz/conductor-loop/internal/agent"
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
		log.Printf("warning: could not detect %s version: %v", command, err)
		return nil
	}

	log.Printf("agent %s: detected version %q at %s", clean, version, path)

	if minVer, ok := minVersions[clean]; ok {
		if !isVersionCompatible(version, minVer) {
			log.Printf("WARNING: agent %s version %q may be below minimum %d.%d.%d",
				clean, version, minVer[0], minVer[1], minVer[2])
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
