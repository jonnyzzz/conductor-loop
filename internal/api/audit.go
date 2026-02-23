package api

import (
	"encoding/json"
	"net/http"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/jonnyzzz/conductor-loop/internal/obslog"
)

const (
	formSubmissionAuditDir          = "_audit"
	formSubmissionAuditFilename     = "form-submissions.jsonl"
	formSubmissionRedactedValue     = "[REDACTED]"
	formSubmissionMaxStringRuneSize = 4096
)

var formSubmissionSensitiveKeyParts = []string{
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

var formSubmissionTokenPatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)\b(sk|rk|pk|xox[baprs]|ghp|github_pat)_[A-Za-z0-9_\-]{8,}\b`),
	regexp.MustCompile(`\beyJ[A-Za-z0-9_-]{10,}\.[A-Za-z0-9_-]{10,}\.[A-Za-z0-9_-]{10,}\b`),
	regexp.MustCompile(`(?i)\b(?:token|secret|password|passwd|api[_-]?key|authorization)\s*[:=]\s*[^\s,"';]+`),
}

var formSubmissionBearerPattern = regexp.MustCompile(`(?i)\bBearer\s+([A-Za-z0-9\-._~+/]+=*)`)

type formSubmissionAuditRecord struct {
	Timestamp     time.Time      `json:"timestamp"`
	RequestID     string         `json:"request_id"`
	CorrelationID string         `json:"correlation_id"`
	Method        string         `json:"method"`
	Path          string         `json:"path"`
	Endpoint      string         `json:"endpoint"`
	RemoteAddr    string         `json:"remote_addr,omitempty"`
	ProjectID     string         `json:"project_id,omitempty"`
	TaskID        string         `json:"task_id,omitempty"`
	RunID         string         `json:"run_id,omitempty"`
	MessageID     string         `json:"message_id,omitempty"`
	Payload       map[string]any `json:"payload,omitempty"`
}

type formSubmissionAuditArgs struct {
	Endpoint  string
	ProjectID string
	TaskID    string
	RunID     string
	MessageID string
	Payload   any
}

func (s *Server) writeFormSubmissionAudit(r *http.Request, args formSubmissionAuditArgs) {
	if s == nil {
		return
	}

	timestamp := time.Now().UTC()
	if s.now != nil {
		timestamp = s.now().UTC()
	}

	requestID := requestIDFromRequest(r)
	if requestID == "" {
		requestID = newRequestID(timestamp)
	}

	record := formSubmissionAuditRecord{
		Timestamp:     timestamp,
		RequestID:     requestID,
		CorrelationID: requestID,
		Method:        httpMethod(r),
		Path:          httpPath(r),
		Endpoint:      strings.TrimSpace(args.Endpoint),
		RemoteAddr:    httpRemoteAddr(r),
		ProjectID:     strings.TrimSpace(args.ProjectID),
		TaskID:        strings.TrimSpace(args.TaskID),
		RunID:         strings.TrimSpace(args.RunID),
		MessageID:     strings.TrimSpace(args.MessageID),
		Payload:       sanitizeFormSubmissionPayload(args.Payload),
	}
	if record.Endpoint == "" {
		record.Endpoint = strings.TrimSpace(record.Method + " " + record.Path)
	}
	if err := s.appendFormSubmissionAuditRecord(record); err != nil && s.logger != nil {
		obslog.Log(s.logger, "ERROR", "api", "audit_write_failed",
			obslog.F("request_id", record.RequestID),
			obslog.F("correlation_id", record.CorrelationID),
			obslog.F("endpoint", record.Endpoint),
			obslog.F("project_id", record.ProjectID),
			obslog.F("task_id", record.TaskID),
			obslog.F("run_id", record.RunID),
			obslog.F("message_id", record.MessageID),
			obslog.F("error", err),
		)
	}
}

func (s *Server) appendFormSubmissionAuditRecord(record formSubmissionAuditRecord) error {
	path := s.formSubmissionAuditPath()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}

	data, err := json.Marshal(record)
	if err != nil {
		return err
	}

	s.auditMu.Lock()
	defer s.auditMu.Unlock()

	file, err := os.OpenFile(path, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0o640)
	if err != nil {
		return err
	}
	defer file.Close()

	data = append(data, '\n')
	if _, err := file.Write(data); err != nil {
		return err
	}
	return file.Sync()
}

func (s *Server) formSubmissionAuditPath() string {
	if s == nil {
		return ""
	}
	return filepath.Join(s.rootDir, formSubmissionAuditDir, formSubmissionAuditFilename)
}

func sanitizeFormSubmissionPayload(payload any) map[string]any {
	if payload == nil {
		return nil
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return map[string]any{"value": formSubmissionRedactedValue}
	}

	var generic any
	if err := json.Unmarshal(data, &generic); err != nil {
		return map[string]any{"value": formSubmissionRedactedValue}
	}

	if object, ok := generic.(map[string]any); ok {
		return sanitizeFormSubmissionMap(object)
	}
	return map[string]any{"value": sanitizeFormSubmissionValue("", generic)}
}

func sanitizeFormSubmissionMap(input map[string]any) map[string]any {
	output := make(map[string]any, len(input))
	for key, value := range input {
		output[key] = sanitizeFormSubmissionValue(key, value)
	}
	return output
}

func sanitizeFormSubmissionValue(key string, value any) any {
	if isFormSubmissionSensitiveKey(key) {
		return formSubmissionRedactedValue
	}

	switch typed := value.(type) {
	case map[string]any:
		return sanitizeFormSubmissionMap(typed)
	case []any:
		output := make([]any, len(typed))
		for i, item := range typed {
			output[i] = sanitizeFormSubmissionValue("", item)
		}
		return output
	case string:
		return sanitizeFormSubmissionString(typed)
	default:
		return value
	}
}

func sanitizeFormSubmissionString(value string) string {
	sanitized := formSubmissionBearerPattern.ReplaceAllString(value, "Bearer "+formSubmissionRedactedValue)
	for _, pattern := range formSubmissionTokenPatterns {
		sanitized = pattern.ReplaceAllString(sanitized, formSubmissionRedactedValue)
	}

	runes := []rune(sanitized)
	if len(runes) > formSubmissionMaxStringRuneSize {
		return string(runes[:formSubmissionMaxStringRuneSize]) + "...[TRUNCATED]"
	}
	return sanitized
}

func isFormSubmissionSensitiveKey(key string) bool {
	normalized := strings.ToLower(strings.TrimSpace(key))
	if normalized == "" {
		return false
	}
	normalized = strings.ReplaceAll(normalized, "-", "_")
	for _, part := range formSubmissionSensitiveKeyParts {
		if strings.Contains(normalized, part) {
			return true
		}
	}
	return false
}

func httpMethod(r *http.Request) string {
	if r == nil {
		return ""
	}
	return strings.TrimSpace(r.Method)
}

func httpPath(r *http.Request) string {
	if r == nil || r.URL == nil {
		return ""
	}
	return strings.TrimSpace(r.URL.Path)
}

func httpRemoteAddr(r *http.Request) string {
	if r == nil {
		return ""
	}
	return strings.TrimSpace(r.RemoteAddr)
}
