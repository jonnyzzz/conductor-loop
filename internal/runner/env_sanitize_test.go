package runner

import (
	"strings"
	"testing"
)

func TestSanitizeEnvForAgent_Claude(t *testing.T) {
	fullEnv := []string{
		"PATH=/usr/bin:/bin",
		"HOME=/home/user",
		"ANTHROPIC_API_KEY=claude-secret",
		"OPENAI_API_KEY=openai-secret",
		"GEMINI_API_KEY=gemini-secret",
		"PERPLEXITY_API_KEY=perplexity-secret",
		"XAI_API_KEY=xai-secret",
		"JRUN_PROJECT_ID=myproject",
		"MY_CUSTOM_VAR=foo",
	}

	result := sanitizeEnvForAgent("claude", fullEnv)

	envMap := envSliceToMap(result)

	// Should keep its own key
	if _, ok := envMap["ANTHROPIC_API_KEY"]; !ok {
		t.Errorf("ANTHROPIC_API_KEY should be present for claude agent")
	}
	// Should strip foreign keys
	for _, foreign := range []string{"OPENAI_API_KEY", "GEMINI_API_KEY", "PERPLEXITY_API_KEY", "XAI_API_KEY"} {
		if _, ok := envMap[foreign]; ok {
			t.Errorf("foreign key %s should be stripped from claude env", foreign)
		}
	}
	// Should keep system and JRUN vars
	for _, keep := range []string{"PATH", "HOME", "JRUN_PROJECT_ID", "MY_CUSTOM_VAR"} {
		if _, ok := envMap[keep]; !ok {
			t.Errorf("expected key %s to be present", keep)
		}
	}
}

func TestSanitizeEnvForAgent_Codex(t *testing.T) {
	fullEnv := []string{
		"PATH=/usr/bin",
		"OPENAI_API_KEY=openai-secret",
		"OPENAI_ORG_ID=org-123",
		"ANTHROPIC_API_KEY=claude-secret",
		"GEMINI_API_KEY=gemini-secret",
	}

	result := sanitizeEnvForAgent("codex", fullEnv)
	envMap := envSliceToMap(result)

	if _, ok := envMap["OPENAI_API_KEY"]; !ok {
		t.Errorf("OPENAI_API_KEY should be present for codex agent")
	}
	if _, ok := envMap["OPENAI_ORG_ID"]; !ok {
		t.Errorf("OPENAI_ORG_ID should be present for codex agent")
	}
	if _, ok := envMap["ANTHROPIC_API_KEY"]; ok {
		t.Errorf("ANTHROPIC_API_KEY should be stripped from codex env")
	}
	if _, ok := envMap["GEMINI_API_KEY"]; ok {
		t.Errorf("GEMINI_API_KEY should be stripped from codex env")
	}
}

func TestSanitizeEnvForAgent_UnknownAgent(t *testing.T) {
	fullEnv := []string{
		"PATH=/usr/bin",
		"OPENAI_API_KEY=openai-secret",
		"ANTHROPIC_API_KEY=claude-secret",
		"MY_VAR=value",
	}

	result := sanitizeEnvForAgent("unknown-agent", fullEnv)
	envMap := envSliceToMap(result)

	// Unknown agent has no allowed key, so all known API keys are stripped.
	if _, ok := envMap["OPENAI_API_KEY"]; ok {
		t.Errorf("OPENAI_API_KEY should be stripped from unknown agent env")
	}
	if _, ok := envMap["ANTHROPIC_API_KEY"]; ok {
		t.Errorf("ANTHROPIC_API_KEY should be stripped from unknown agent env")
	}
	// Non-API-key vars should remain
	if _, ok := envMap["PATH"]; !ok {
		t.Errorf("PATH should remain")
	}
	if _, ok := envMap["MY_VAR"]; !ok {
		t.Errorf("MY_VAR should remain")
	}
}

func envSliceToMap(env []string) map[string]string {
	m := make(map[string]string, len(env))
	for _, entry := range env {
		parts := strings.SplitN(entry, "=", 2)
		key := parts[0]
		val := ""
		if len(parts) == 2 {
			val = parts[1]
		}
		m[key] = val
	}
	return m
}
