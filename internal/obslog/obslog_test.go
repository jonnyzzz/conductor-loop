package obslog

import (
	"bytes"
	"errors"
	"log"
	"strings"
	"testing"
)

func TestLogFormatsStructuredFields(t *testing.T) {
	var buf bytes.Buffer
	logger := log.New(&buf, "", 0)

	Log(logger, "info", "api", "request_completed",
		F("request_id", "req-123"),
		F("project_id", "proj"),
		F("status", 200),
	)

	line := buf.String()
	if !strings.Contains(line, "level=INFO") {
		t.Fatalf("expected INFO level in %q", line)
	}
	if !strings.Contains(line, "subsystem=api") {
		t.Fatalf("expected subsystem in %q", line)
	}
	if !strings.Contains(line, "event=request_completed") {
		t.Fatalf("expected event in %q", line)
	}
	if !strings.Contains(line, "request_id=req-123") {
		t.Fatalf("expected request_id in %q", line)
	}
	if !strings.Contains(line, "status=200") {
		t.Fatalf("expected status in %q", line)
	}
	if !strings.Contains(line, "ts=") {
		t.Fatalf("expected timestamp in %q", line)
	}
}

func TestLogRedactsSensitiveKeyValues(t *testing.T) {
	var buf bytes.Buffer
	logger := log.New(&buf, "", 0)

	Log(logger, "info", "runner", "token_loaded",
		F("api_key", "abc123"),
		F("auth_token", "xyz"),
	)

	line := buf.String()
	if strings.Contains(line, "abc123") || strings.Contains(line, "xyz") {
		t.Fatalf("expected sensitive values to be redacted: %q", line)
	}
	if strings.Count(line, redactedValue) < 2 {
		t.Fatalf("expected redacted placeholders in %q", line)
	}
}

func TestLogRedactsSecretPatternsInValues(t *testing.T) {
	var buf bytes.Buffer
	logger := log.New(&buf, "", 0)

	Log(logger, "warn", "api", "request_error",
		F("error", errors.New("authorization=Bearer sk_live_abc123456789")),
	)

	line := buf.String()
	if strings.Contains(line, "sk_live_abc123456789") {
		t.Fatalf("token leaked in %q", line)
	}
	if !strings.Contains(line, redactedValue) {
		t.Fatalf("expected redaction marker in %q", line)
	}
}
