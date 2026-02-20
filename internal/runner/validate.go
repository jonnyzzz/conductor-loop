package runner

import (
	"context"
	"log"
	"os/exec"
	"strings"

	"github.com/pkg/errors"

	"github.com/jonnyzzz/conductor-loop/internal/agent"
)

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
	return nil
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
