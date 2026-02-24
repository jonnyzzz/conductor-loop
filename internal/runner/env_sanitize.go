package runner

import (
	"strings"
)

// allAgentAPIKeys lists all known per-agent API key environment variable names.
// When launching an agent, only the key for the target agent type is allowed through.
var allAgentAPIKeys = []string{
	"OPENAI_API_KEY",
	"ANTHROPIC_API_KEY",
	"GEMINI_API_KEY",
	"PERPLEXITY_API_KEY",
	"XAI_API_KEY",
	"OPENAI_ORG_ID",
}

// sanitizeEnvForAgent strips foreign API keys from the environment before
// passing it to an agent subprocess. Only the key relevant to agentType is
// kept; all other agent API keys are removed.
//
// System path vars, JRUN_* vars, and the conductor-loop internal vars are
// always preserved. Keys and their names are logged at debug level by the
// caller; values are never logged.
func sanitizeEnvForAgent(agentType string, env []string) []string {
	allowedAgentKey := tokenEnvVar(agentType)
	// also allow codex org key when running codex
	allowOrgKey := strings.EqualFold(agentType, "codex")

	result := make([]string, 0, len(env))
	for _, entry := range env {
		parts := strings.SplitN(entry, "=", 2)
		if len(parts) < 1 || parts[0] == "" {
			continue
		}
		key := parts[0]
		if isForeignAgentAPIKey(key, allowedAgentKey, allowOrgKey) {
			continue
		}
		result = append(result, entry)
	}
	return result
}

// isForeignAgentAPIKey returns true if key is a known agent API key that
// should NOT be passed to the current agent (i.e., it belongs to a different
// agent type).
func isForeignAgentAPIKey(key, allowedKey string, allowOrgKey bool) bool {
	for _, k := range allAgentAPIKeys {
		if !strings.EqualFold(key, k) {
			continue
		}
		// It's a known agent key — allow it only if it's the designated key.
		if strings.EqualFold(key, allowedKey) {
			return false
		}
		if allowOrgKey && strings.EqualFold(key, "OPENAI_ORG_ID") {
			return false
		}
		return true
	}
	return false
}
