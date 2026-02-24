package main

import (
	"strings"
	"testing"
)

func TestDetectIterateVerdict_Pass(t *testing.T) {
	for _, tok := range []string{"APPROVED", "approved", "This looks great, APPROVED!", "All PASS"} {
		v := detectIterateVerdict(tok)
		if v != "pass" {
			t.Errorf("detectIterateVerdict(%q) = %q, want pass", tok, v)
		}
	}
}

func TestDetectIterateVerdict_Fail(t *testing.T) {
	for _, tok := range []string{"REJECTED", "FAIL", "CHANGES_REQUESTED", "I say REJECTED!"} {
		v := detectIterateVerdict(tok)
		if v != "fail" {
			t.Errorf("detectIterateVerdict(%q) = %q, want fail", tok, v)
		}
	}
}

func TestDetectIterateVerdict_Unknown(t *testing.T) {
	v := detectIterateVerdict("Looks good to me, but no verdict token.")
	if v != "unknown" {
		t.Errorf("detectIterateVerdict(no-token) = %q, want unknown", v)
	}
}

func TestBuildIteratePrompt_Inline(t *testing.T) {
	result := buildIteratePrompt("Do the thing", "", "")
	if result != "Do the thing" {
		t.Errorf("unexpected prompt: %q", result)
	}
}

func TestBuildIteratePrompt_WithFeedback(t *testing.T) {
	result := buildIteratePrompt("Do the thing", "", "Fix this issue")
	if !strings.Contains(result, "Do the thing") {
		t.Errorf("missing original prompt")
	}
	if !strings.Contains(result, "Fix this issue") {
		t.Errorf("missing feedback")
	}
	if !strings.Contains(result, "Previous review feedback") {
		t.Errorf("missing feedback section header")
	}
}

func TestIterateCmdHelp(t *testing.T) {
	cmd := newIterateCmd()
	if !strings.Contains(cmd.Long, "APPROVED") {
		t.Errorf("iterate help should mention pass tokens")
	}
}

func TestIterateCmdRequiredFlags(t *testing.T) {
	cmd := newIterateCmd()
	err := cmd.RunE(cmd, nil)
	if err == nil || !strings.Contains(err.Error(), "--root") {
		t.Errorf("expected --root required error, got: %v", err)
	}
}
