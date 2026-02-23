package obslog

import (
	"fmt"
	"log"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode"
)

const (
	redactedValue = "[REDACTED]"
	maxValueRunes = 2048
)

var sensitiveKeyParts = []string{
	"token",
	"secret",
	"password",
	"passwd",
	"api_key",
	"apikey",
	"authorization",
	"auth",
	"cookie",
	"session",
	"private_key",
	"ssh_key",
	"access_key",
	"refresh_token",
}

var bearerPattern = regexp.MustCompile(`(?i)\bBearer\s+([A-Za-z0-9\-._~+/]+=*)`)

var secretValuePatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)\b(sk|rk|pk|xox[baprs]|ghp|github_pat)_[A-Za-z0-9_\-]{8,}\b`),
	regexp.MustCompile(`\beyJ[A-Za-z0-9_-]{10,}\.[A-Za-z0-9_-]{10,}\.[A-Za-z0-9_-]{10,}\b`),
	regexp.MustCompile(`(?i)\b(?:token|secret|password|passwd|api[_-]?key|authorization)\s*[:=]\s*[^\s,"';]+`),
}

// Field is a single structured log field.
type Field struct {
	Key   string
	Value any
}

// F constructs a structured log field.
func F(key string, value any) Field {
	return Field{Key: key, Value: value}
}

// Log writes a redacted structured log line.
func Log(logger *log.Logger, level, subsystem, event string, fields ...Field) {
	if logger == nil {
		return
	}

	ts := time.Now().UTC().Format(time.RFC3339Nano)
	level = normalizeLevel(level)
	subsystem = normalizeIdentifier(subsystem, "unknown")
	event = normalizeIdentifier(event, "unknown")

	parts := make([]string, 0, len(fields)+4)
	parts = append(parts,
		encodePair("ts", ts),
		encodePair("level", level),
		encodePair("subsystem", subsystem),
		encodePair("event", event),
	)

	for _, field := range fields {
		key := normalizeKey(field.Key)
		if key == "" {
			continue
		}
		value := valueToString(field.Value)
		value = sanitizeValueForKey(key, value)
		parts = append(parts, encodePair(key, value))
	}

	logger.Print(strings.Join(parts, " "))
}

func normalizeLevel(level string) string {
	clean := strings.ToUpper(strings.TrimSpace(level))
	switch clean {
	case "DEBUG", "INFO", "WARN", "ERROR":
		return clean
	default:
		return "INFO"
	}
}

func normalizeIdentifier(value, fallback string) string {
	clean := normalizeKey(value)
	if clean == "" {
		return fallback
	}
	return clean
}

func normalizeKey(key string) string {
	trimmed := strings.TrimSpace(strings.ToLower(key))
	if trimmed == "" {
		return ""
	}
	var b strings.Builder
	b.Grow(len(trimmed))
	lastUnderscore := false
	for _, r := range trimmed {
		if unicode.IsLetter(r) || unicode.IsDigit(r) {
			b.WriteRune(r)
			lastUnderscore = false
			continue
		}
		if !lastUnderscore {
			b.WriteByte('_')
			lastUnderscore = true
		}
	}
	out := strings.Trim(b.String(), "_")
	if out == "" {
		return ""
	}
	return out
}

func valueToString(value any) string {
	switch typed := value.(type) {
	case nil:
		return ""
	case string:
		return typed
	case time.Time:
		if typed.IsZero() {
			return ""
		}
		return typed.UTC().Format(time.RFC3339Nano)
	case time.Duration:
		return typed.String()
	case error:
		return typed.Error()
	default:
		return fmt.Sprint(typed)
	}
}

func sanitizeValueForKey(key, value string) string {
	if isSensitiveKey(key) {
		return redactedValue
	}
	return sanitizeValue(value)
}

func sanitizeValue(value string) string {
	sanitized := strings.TrimSpace(value)
	if sanitized == "" {
		return ""
	}
	sanitized = bearerPattern.ReplaceAllString(sanitized, "Bearer "+redactedValue)
	for _, pattern := range secretValuePatterns {
		sanitized = pattern.ReplaceAllString(sanitized, redactedValue)
	}
	runes := []rune(sanitized)
	if len(runes) > maxValueRunes {
		sanitized = string(runes[:maxValueRunes]) + "...[TRUNCATED]"
	}
	return sanitized
}

func isSensitiveKey(key string) bool {
	normalized := strings.ToLower(strings.TrimSpace(key))
	if normalized == "" {
		return false
	}
	normalized = strings.ReplaceAll(normalized, "-", "_")
	for _, part := range sensitiveKeyParts {
		if strings.Contains(normalized, part) {
			return true
		}
	}
	return false
}

func encodePair(key, value string) string {
	if value == "" {
		return key + "=" + strconv.Quote("")
	}
	if needsQuotes(value) {
		return key + "=" + strconv.Quote(value)
	}
	return key + "=" + value
}

func needsQuotes(value string) bool {
	for _, r := range value {
		switch r {
		case ' ', '\t', '\n', '\r', '=', '"':
			return true
		}
	}
	return false
}
