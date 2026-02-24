package messagebus

import (
	"bufio"
	"bytes"
	stderrors "errors"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"time"

	"github.com/jonnyzzz/conductor-loop/internal/obslog"
	"github.com/pkg/errors"
	"gopkg.in/yaml.v3"
)

const (
	defaultLockTimeout  = 10 * time.Second
	defaultPollInterval = 200 * time.Millisecond
	defaultMaxRetries   = 3
	defaultRetryBackoff = 100 * time.Millisecond
	messageBusFileMode  = 0o644
)

// Run event types posted to the message bus.
const (
	EventTypeRunStart = "RUN_START"
	EventTypeRunStop  = "RUN_STOP"
	EventTypeRunCrash = "RUN_CRASH"
)

// ErrSinceIDNotFound indicates the requested since ID was not found.
var ErrSinceIDNotFound = stderrors.New("since id not found")

// Parent represents a structured parent reference.
type Parent struct {
	MsgID string            `yaml:"msg_id"`
	Kind  string            `yaml:"kind,omitempty"` // e.g. "depends_on", "blocks", "child_of"
	Meta  map[string]string `yaml:"meta,omitempty"`
}

// Link represents an advisory link.
type Link struct {
	URL   string `yaml:"url"`
	Label string `yaml:"label,omitempty"`
	Kind  string `yaml:"kind,omitempty"`
}

// Message represents a message bus entry.
type Message struct {
	MsgID     string            `yaml:"msg_id"`
	Timestamp time.Time         `yaml:"ts"`
	Type      string            `yaml:"type"`
	ProjectID string            `yaml:"project_id"`
	TaskID    string            `yaml:"task_id"`
	RunID     string            `yaml:"run_id"`
	IssueID   string            `yaml:"issue_id,omitempty"` // alias for msg_id on ISSUE messages
	Parents   []Parent          `yaml:"parents,omitempty"`  // replaces ParentMsgIDs
	Links     []Link            `yaml:"links,omitempty"`
	Meta      map[string]string `yaml:"meta,omitempty"`
	Body      string            `yaml:"-"`
}

// rawMessage is used for custom YAML parsing to handle backward compat.
type rawMessage struct {
	MsgID     string            `yaml:"msg_id"`
	Timestamp time.Time         `yaml:"ts"`
	Type      string            `yaml:"type"`
	ProjectID string            `yaml:"project_id"`
	TaskID    string            `yaml:"task_id"`
	RunID     string            `yaml:"run_id"`
	IssueID   string            `yaml:"issue_id,omitempty"`
	Parents   yaml.Node         `yaml:"parents,omitempty"`
	Links     []Link            `yaml:"links,omitempty"`
	Meta      map[string]string `yaml:"meta,omitempty"`
}

func parseParents(node yaml.Node) []Parent {
	if node.Kind == 0 {
		return nil
	}
	if node.Kind == yaml.SequenceNode && len(node.Content) > 0 {
		// Check if first element is a scalar (string) or mapping (object).
		first := node.Content[0]
		if first.Kind == yaml.ScalarNode {
			// Old format: string list.
			var strs []string
			if err := node.Decode(&strs); err == nil {
				parents := make([]Parent, 0, len(strs))
				for _, s := range strs {
					if s != "" {
						parents = append(parents, Parent{MsgID: s})
					}
				}
				return parents
			}
		}
		// New format: object list.
		var parents []Parent
		if err := node.Decode(&parents); err == nil {
			return parents
		}
	}
	return nil
}

// MessageBus manages append-only message bus files.
type MessageBus struct {
	path            string
	now             func() time.Time
	lockTimeout     time.Duration
	pollInterval    time.Duration
	maxRetries      int
	retryBackoff    time.Duration
	fsync           bool
	autoRotateBytes int64

	attempts int64
	retries  int64
}

// Option configures a MessageBus.
type Option func(*MessageBus)

// WithLockTimeout sets the exclusive lock timeout.
func WithLockTimeout(timeout time.Duration) Option {
	return func(bus *MessageBus) {
		bus.lockTimeout = timeout
	}
}

// WithPollInterval sets the poll interval for PollForNew.
func WithPollInterval(interval time.Duration) Option {
	return func(bus *MessageBus) {
		bus.pollInterval = interval
	}
}

// WithClock sets the clock used for timestamps.
func WithClock(now func() time.Time) Option {
	return func(bus *MessageBus) {
		bus.now = now
	}
}

// WithMaxRetries sets the maximum number of append attempts. Minimum 1.
func WithMaxRetries(n int) Option {
	return func(bus *MessageBus) {
		if n < 1 {
			n = 1
		}
		bus.maxRetries = n
	}
}

// WithRetryBackoff sets the base backoff duration between retries.
func WithRetryBackoff(d time.Duration) Option {
	return func(bus *MessageBus) {
		bus.retryBackoff = d
	}
}

// WithFsync enables or disables fsync after each message write.
// Default is false (no fsync) for maximum throughput.
// Enable for durability-critical deployments where message loss on OS crash is unacceptable.
// WARNING: fsync significantly reduces throughput (~200 msg/sec vs 37,000+ without).
func WithFsync(enabled bool) Option {
	return func(bus *MessageBus) {
		bus.fsync = enabled
	}
}

// WithAutoRotate configures automatic rotation when the bus file exceeds maxBytes.
// When the file size reaches or exceeds maxBytes, it is renamed to
// <path>.YYYYMMDD-HHMMSS.archived and a fresh bus file is started.
// Rotation occurs atomically inside the write lock.
// Set to 0 to disable (default).
func WithAutoRotate(maxBytes int64) Option {
	return func(bus *MessageBus) {
		bus.autoRotateBytes = maxBytes
	}
}

// ContentionStats returns the total append attempts and lock contention retries.
func (mb *MessageBus) ContentionStats() (attempts, retries int64) {
	return atomic.LoadInt64(&mb.attempts), atomic.LoadInt64(&mb.retries)
}

// NewMessageBus creates a MessageBus for the provided path.
func NewMessageBus(path string, opts ...Option) (*MessageBus, error) {
	clean := filepath.Clean(strings.TrimSpace(path))
	if clean == "." || clean == "" {
		return nil, errors.New("message bus path is empty")
	}
	bus := &MessageBus{
		path:         clean,
		now:          time.Now,
		lockTimeout:  defaultLockTimeout,
		pollInterval: defaultPollInterval,
		maxRetries:   defaultMaxRetries,
		retryBackoff: defaultRetryBackoff,
	}
	for _, opt := range opts {
		if opt != nil {
			opt(bus)
		}
	}
	if bus.now == nil {
		return nil, errors.New("clock is nil")
	}
	if bus.lockTimeout <= 0 {
		return nil, errors.New("lock timeout must be positive")
	}
	if bus.pollInterval <= 0 {
		return nil, errors.New("poll interval must be positive")
	}
	return bus, nil
}

// AppendMessage appends a message to the bus and returns its msg_id.
func (mb *MessageBus) AppendMessage(msg *Message) (string, error) {
	if mb == nil {
		return "", errors.New("message bus is nil")
	}
	if msg == nil {
		return "", errors.New("message is nil")
	}
	if strings.TrimSpace(msg.Type) == "" {
		return "", errors.New("message type is empty")
	}
	if strings.TrimSpace(msg.ProjectID) == "" {
		return "", errors.New("project id is empty")
	}

	msg.MsgID = GenerateMessageID()
	if msg.Timestamp.IsZero() {
		msg.Timestamp = mb.now().UTC()
	} else {
		msg.Timestamp = msg.Timestamp.UTC()
	}
	if msg.Type == "ISSUE" && msg.IssueID == "" {
		msg.IssueID = msg.MsgID
	}

	data, err := serializeMessage(msg)
	if err != nil {
		return "", errors.Wrap(err, "serialize message")
	}

	if err := validateBusPath(mb.path); err != nil {
		return "", errors.Wrap(err, "validate message bus path")
	}

	var lastErr error
	for attempt := 0; attempt < mb.maxRetries; attempt++ {
		atomic.AddInt64(&mb.attempts, 1)

		if attempt > 0 {
			atomic.AddInt64(&mb.retries, 1)
			backoff := mb.retryBackoff * (1 << (attempt - 1))
			time.Sleep(backoff)
		}

		lastErr = mb.tryAppend(data)
		if lastErr == nil {
			if attempt > 0 {
				obslog.Log(log.Default(), "INFO", "messagebus", "append_recovered_after_retry",
					obslog.F("path", mb.path),
					obslog.F("message_id", msg.MsgID),
					obslog.F("message_type", msg.Type),
					obslog.F("project_id", msg.ProjectID),
					obslog.F("task_id", msg.TaskID),
					obslog.F("run_id", msg.RunID),
					obslog.F("attempt", attempt+1),
					obslog.F("max_retries", mb.maxRetries),
				)
			}
			return msg.MsgID, nil
		}
		if !stderrors.Is(lastErr, ErrLockTimeout) {
			obslog.Log(log.Default(), "ERROR", "messagebus", "append_failed",
				obslog.F("path", mb.path),
				obslog.F("message_id", msg.MsgID),
				obslog.F("message_type", msg.Type),
				obslog.F("project_id", msg.ProjectID),
				obslog.F("task_id", msg.TaskID),
				obslog.F("run_id", msg.RunID),
				obslog.F("attempt", attempt+1),
				obslog.F("error", lastErr),
			)
			return "", lastErr
		}
		obslog.Log(log.Default(), "WARN", "messagebus", "append_lock_timeout_retrying",
			obslog.F("path", mb.path),
			obslog.F("message_id", msg.MsgID),
			obslog.F("message_type", msg.Type),
			obslog.F("project_id", msg.ProjectID),
			obslog.F("task_id", msg.TaskID),
			obslog.F("run_id", msg.RunID),
			obslog.F("attempt", attempt+1),
			obslog.F("max_retries", mb.maxRetries),
			obslog.F("error", lastErr),
		)
	}
	obslog.Log(log.Default(), "ERROR", "messagebus", "append_exhausted_retries",
		obslog.F("path", mb.path),
		obslog.F("message_id", msg.MsgID),
		obslog.F("message_type", msg.Type),
		obslog.F("project_id", msg.ProjectID),
		obslog.F("task_id", msg.TaskID),
		obslog.F("run_id", msg.RunID),
		obslog.F("max_retries", mb.maxRetries),
		obslog.F("error", lastErr),
	)
	return "", fmt.Errorf("append failed after %d attempts: %w", mb.maxRetries, lastErr)
}

func (mb *MessageBus) tryAppend(data []byte) error {
	file, err := os.OpenFile(mb.path, os.O_WRONLY|os.O_APPEND|os.O_CREATE, messageBusFileMode)
	if err != nil {
		return errors.Wrap(err, "open message bus")
	}

	if err := LockExclusive(file, mb.lockTimeout); err != nil {
		file.Close()
		obslog.Log(log.Default(), "WARN", "messagebus", "append_lock_failed",
			obslog.F("path", mb.path),
			obslog.F("lock_timeout", mb.lockTimeout),
			obslog.F("error", err),
		)
		return fmt.Errorf("lock message bus: %w", err)
	}

	// Auto-rotation: if file exceeds threshold, rename to archive and create fresh file.
	// We hold the exclusive lock while renaming, so no other writer can interleave.
	if mb.autoRotateBytes > 0 {
		if fi, statErr := file.Stat(); statErr == nil && fi.Size() >= mb.autoRotateBytes {
			previousSize := fi.Size()
			_ = Unlock(file)
			file.Close()

			archivePath := mb.path + "." + mb.now().UTC().Format("20060102-150405") + ".archived"
			_ = os.Rename(mb.path, archivePath) // best-effort; new file created on next open
			obslog.Log(log.Default(), "INFO", "messagebus", "bus_rotated",
				obslog.F("path", mb.path),
				obslog.F("archive_path", archivePath),
				obslog.F("previous_size_bytes", previousSize),
				obslog.F("rotate_threshold_bytes", mb.autoRotateBytes),
			)

			// Open (or create) the fresh bus file.
			file, err = os.OpenFile(mb.path, os.O_WRONLY|os.O_APPEND|os.O_CREATE, messageBusFileMode)
			if err != nil {
				obslog.Log(log.Default(), "ERROR", "messagebus", "bus_open_after_rotation_failed",
					obslog.F("path", mb.path),
					obslog.F("error", err),
				)
				return errors.Wrap(err, "open new message bus after rotation")
			}
			if err := LockExclusive(file, mb.lockTimeout); err != nil {
				file.Close()
				obslog.Log(log.Default(), "ERROR", "messagebus", "bus_lock_after_rotation_failed",
					obslog.F("path", mb.path),
					obslog.F("lock_timeout", mb.lockTimeout),
					obslog.F("error", err),
				)
				return fmt.Errorf("lock new message bus after rotation: %w", err)
			}
		}
	}

	defer file.Close()
	defer func() {
		_ = Unlock(file)
	}()

	if err := appendEntry(file, data); err != nil {
		return errors.Wrap(err, "write message")
	}
	if mb.fsync {
		if err := file.Sync(); err != nil {
			return errors.Wrap(err, "fsync message bus")
		}
	}
	return nil
}

// readBusFileShared reads the bus file content after acquiring a shared lock (Windows) or
// directly (Unix, where flock is advisory and readers are always lock-free).
// On failure to acquire the lock within the timeout, returns ErrLockTimeout.
func readBusFileShared(path string) ([]byte, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close() //nolint:errcheck
	if err := LockShared(f, 5*time.Second); err != nil {
		return nil, err
	}
	data, err := io.ReadAll(f)
	if uerr := Unlock(f); uerr != nil && err == nil {
		err = uerr
	}
	return data, err
}

// ReadMessages reads messages after sinceID. If sinceID is empty, returns all messages.
// On parse failure, falls back to returning all partially-parseable messages from the
// beginning of the file rather than returning an empty result (partial recovery).
func (mb *MessageBus) ReadMessages(sinceID string) ([]*Message, error) {
	if mb == nil {
		return nil, errors.New("message bus is nil")
	}
	if err := validateBusPath(mb.path); err != nil {
		return nil, errors.Wrap(err, "validate message bus path")
	}
	// On Windows, acquire a shared lock before reading to avoid reading while
	// a writer holds an exclusive mandatory lock. On Unix, tryFlockShared is a
	// no-op so this adds negligible overhead.
	data, err := readBusFileShared(mb.path)
	if err != nil {
		if os.IsNotExist(err) {
			return []*Message{}, nil
		}
		return nil, errors.Wrap(err, "read message bus")
	}
	messages, parseErr := parseMessages(data)
	if parseErr != nil {
		// Partial recovery: re-parse from scratch with best-effort, ignoring bad entries.
		messages = parseMessagesPartial(data)
		if len(messages) == 0 {
			// Nothing parseable — return empty rather than error to avoid empty display.
			return []*Message{}, nil
		}
		// With partial results, can't reliably do sinceID filtering; return all.
		return messages, nil
	}
	return filterSince(messages, sinceID)
}

// parseMessagesPartial parses as many messages as possible from data, skipping
// any entries that fail to parse. It never returns an error.
func parseMessagesPartial(data []byte) []*Message {
	messages := make([]*Message, 0)
	_ = parseMessagesReader(bufio.NewReader(bytes.NewReader(data)), func(msg *Message) {
		if msg != nil {
			messages = append(messages, msg)
		}
	})
	return messages
}

// ReadMessagesSinceLimited reads messages after sinceID and keeps only the latest limit entries.
// For limit <= 0 it behaves like ReadMessages(sinceID).
func (mb *MessageBus) ReadMessagesSinceLimited(sinceID string, limit int) ([]*Message, error) {
	if mb == nil {
		return nil, errors.New("message bus is nil")
	}
	if err := validateBusPath(mb.path); err != nil {
		return nil, errors.Wrap(err, "validate message bus path")
	}
	if limit <= 0 {
		return mb.ReadMessages(sinceID)
	}

	normalizedSinceID := strings.TrimSpace(sinceID)
	file, err := os.Open(mb.path)
	if err != nil {
		if os.IsNotExist(err) {
			return []*Message{}, nil
		}
		return nil, errors.Wrap(err, "open message bus")
	}
	defer file.Close()

	foundSince := normalizedSinceID == ""
	tail := make([]*Message, 0, min(limit, 16))
	tailStart := 0

	pushTail := func(msg *Message) {
		if len(tail) < limit {
			tail = append(tail, msg)
			return
		}
		tail[tailStart] = msg
		tailStart = (tailStart + 1) % limit
	}

	parseErr := parseMessagesReader(bufio.NewReader(file), func(msg *Message) {
		if msg == nil {
			return
		}
		if !foundSince {
			if msg.MsgID == normalizedSinceID {
				foundSince = true
			}
			return
		}
		pushTail(msg)
	})
	if parseErr != nil {
		return nil, parseErr
	}

	if !foundSince {
		return nil, fmt.Errorf("since id %q not found: %w", normalizedSinceID, ErrSinceIDNotFound)
	}
	if len(tail) < limit || tailStart == 0 {
		return tail, nil
	}

	ordered := make([]*Message, 0, len(tail))
	ordered = append(ordered, tail[tailStart:]...)
	ordered = append(ordered, tail[:tailStart]...)
	return ordered, nil
}

const readLastNChunkSize = 64 * 1024 // 64KB initial seek window

// ReadLastN returns the last n messages without loading the entire file into memory.
// For n <= 0, returns all messages (same as ReadMessages("")).
// For small files (≤64KB), falls back to a full read then trims to n.
// For large files, uses a seek-based approach: reads a chunk from near the end,
// parsing messages. Doubles the window up to 3 times if needed, then falls back.
func (mb *MessageBus) ReadLastN(n int) ([]*Message, error) {
	if mb == nil {
		return nil, errors.New("message bus is nil")
	}
	if err := validateBusPath(mb.path); err != nil {
		return nil, errors.Wrap(err, "validate message bus path")
	}
	if n <= 0 {
		return mb.ReadMessages("")
	}

	f, err := os.Open(mb.path)
	if err != nil {
		if os.IsNotExist(err) {
			return []*Message{}, nil
		}
		return nil, errors.Wrap(err, "open message bus")
	}
	defer f.Close()

	fileSize, err := f.Seek(0, io.SeekEnd)
	if err != nil {
		return nil, errors.Wrap(err, "seek end of message bus")
	}

	if fileSize <= readLastNChunkSize {
		// Small file: full read then trim.
		return mb.readAllLastN(n)
	}

	// Estimate bytes by message count, then grow exponentially when needed.
	// This avoids eagerly reading the whole file for moderate tail sizes.
	const estimatedBytesPerMessage int64 = 384
	chunkSize := int64(n) * estimatedBytesPerMessage
	if chunkSize < readLastNChunkSize {
		chunkSize = readLastNChunkSize
	}

	for attempt := 0; attempt < 4; attempt++ {
		if chunkSize >= fileSize {
			break
		}

		offset := fileSize - chunkSize
		if _, err := f.Seek(offset, io.SeekStart); err != nil {
			return nil, errors.Wrap(err, "seek message bus")
		}

		data := make([]byte, chunkSize)
		nRead, readErr := io.ReadFull(f, data)
		if readErr != nil && readErr != io.ErrUnexpectedEOF {
			return nil, errors.Wrap(readErr, "read message bus chunk")
		}
		data = data[:nRead]

		messages, err := parseMessages(data)
		if err != nil {
			return nil, err
		}

		if len(messages) >= n {
			return messages[len(messages)-n:], nil
		}

		// Not enough messages: double the window and retry.
		chunkSize *= 2
	}

	// Fall back to full read then trim.
	return mb.readAllLastN(n)
}

// readAllLastN reads all messages and returns the last n.
func (mb *MessageBus) readAllLastN(n int) ([]*Message, error) {
	messages, err := mb.ReadMessages("")
	if err != nil {
		return nil, err
	}
	if len(messages) > n {
		return messages[len(messages)-n:], nil
	}
	return messages, nil
}

// PollForNew blocks until new messages appear after lastID.
func (mb *MessageBus) PollForNew(lastID string) ([]*Message, error) {
	if mb == nil {
		return nil, errors.New("message bus is nil")
	}
	for {
		messages, err := mb.ReadMessages(lastID)
		if err != nil {
			return nil, err
		}
		if len(messages) > 0 {
			return messages, nil
		}
		time.Sleep(mb.pollInterval)
	}
}

func appendEntry(file *os.File, data []byte) error {
	if file == nil {
		return errors.New("message bus file is nil")
	}
	if len(data) == 0 {
		return errors.New("message data is empty")
	}
	info, err := file.Stat()
	if err != nil {
		return errors.Wrap(err, "stat message bus")
	}
	if info.Size() > 0 {
		if _, err := file.Write([]byte("\n")); err != nil {
			return errors.Wrap(err, "write separator")
		}
	}
	if err := writeAll(file, data); err != nil {
		return err
	}
	return nil
}

func writeAll(w io.Writer, data []byte) error {
	for len(data) > 0 {
		n, err := w.Write(data)
		if err != nil {
			return errors.Wrap(err, "write message data")
		}
		if n == 0 {
			return errors.New("short write")
		}
		data = data[n:]
	}
	return nil
}

func serializeMessage(msg *Message) ([]byte, error) {
	if msg == nil {
		return nil, errors.New("message is nil")
	}
	header, err := yaml.Marshal(msg)
	if err != nil {
		return nil, errors.Wrap(err, "marshal message")
	}
	var buf bytes.Buffer
	buf.WriteString("---\n")
	buf.Write(header)
	if len(header) == 0 || header[len(header)-1] != '\n' {
		buf.WriteByte('\n')
	}
	buf.WriteString("---\n")
	if msg.Body != "" {
		buf.WriteString(msg.Body)
	}
	if !strings.HasSuffix(msg.Body, "\n") {
		buf.WriteByte('\n')
	}
	return buf.Bytes(), nil
}

func parseMessages(data []byte) ([]*Message, error) {
	messages := make([]*Message, 0)
	if err := parseMessagesReader(bufio.NewReader(bytes.NewReader(data)), func(msg *Message) {
		messages = append(messages, msg)
	}); err != nil {
		return nil, err
	}
	return messages, nil
}

func parseMessagesReader(reader *bufio.Reader, onMessage func(*Message)) error {
	if reader == nil {
		return errors.New("message reader is nil")
	}
	if onMessage == nil {
		return errors.New("message callback is nil")
	}

	const (
		stateSeekHeader = iota
		stateHeader
		stateBody
	)
	state := stateSeekHeader
	var headerBuf bytes.Buffer
	var bodyBuf bytes.Buffer
	var current *Message
	lineNo := 0

	for {
		line, err := reader.ReadString('\n')
		if err != nil && err != io.EOF {
			return errors.Wrap(err, "read message bus")
		}
		if err == io.EOF && line == "" {
			break
		}
		lineNo++
		trimmed := strings.TrimRight(line, "\r\n")
		switch state {
		case stateSeekHeader:
			if trimmed == "---" {
				state = stateHeader
				headerBuf.Reset()
			} else if msg, ok := parseLegacyMessageLine(trimmed, lineNo); ok {
				onMessage(msg)
			}
		case stateHeader:
			if trimmed == "---" {
				var raw rawMessage
				if err := yaml.Unmarshal(headerBuf.Bytes(), &raw); err != nil {
					headerBuf.Reset()
					current = nil
					state = stateHeader
					break
				}
				msg := &Message{
					MsgID:     raw.MsgID,
					Timestamp: raw.Timestamp,
					Type:      raw.Type,
					ProjectID: raw.ProjectID,
					TaskID:    raw.TaskID,
					RunID:     raw.RunID,
					IssueID:   raw.IssueID,
					Parents:   parseParents(raw.Parents),
					Links:     raw.Links,
					Meta:      raw.Meta,
				}
				current = msg
				bodyBuf.Reset()
				state = stateBody
			} else {
				headerBuf.WriteString(line)
			}
		case stateBody:
			if trimmed == "---" {
				if current != nil {
					current.Body = finalizeBody(bodyBuf.Bytes())
					onMessage(current)
				}
				headerBuf.Reset()
				state = stateHeader
			} else {
				bodyBuf.WriteString(line)
			}
		}
		if err == io.EOF {
			break
		}
	}

	if state == stateBody && current != nil {
		bodyBytes := bodyBuf.Bytes()
		if len(bodyBytes) > 0 && bodyBytes[len(bodyBytes)-1] == '\n' {
			current.Body = finalizeBody(bodyBytes)
			onMessage(current)
		}
	}
	return nil
}

var legacyTimestampLayouts = []string{
	"2006-01-02 15:04:05",
	"2006-01-02T15:04:05",
	time.RFC3339,
}

// parseLegacyMessageLine parses legacy markdown log lines:
// [YYYY-MM-DD HH:MM:SS] TYPE: body
// Lines that do not match this format are ignored by design.
func parseLegacyMessageLine(line string, lineNo int) (*Message, bool) {
	trimmed := strings.TrimSpace(line)
	if !strings.HasPrefix(trimmed, "[") {
		return nil, false
	}

	closing := strings.Index(trimmed, "]")
	if closing <= 1 {
		return nil, false
	}

	tsRaw := strings.TrimSpace(trimmed[1:closing])
	ts, ok := parseLegacyTimestamp(tsRaw)
	if !ok {
		return nil, false
	}

	content := strings.TrimSpace(trimmed[closing+1:])
	if content == "" {
		return nil, false
	}

	msgType := "INFO"
	body := content
	if colon := strings.Index(content, ":"); colon > 0 {
		candidateType := strings.TrimSpace(content[:colon])
		candidateBody := strings.TrimSpace(content[colon+1:])
		if isLegacyTypeToken(candidateType) {
			msgType = strings.ToUpper(candidateType)
			body = candidateBody
		}
	}
	if body == "" {
		body = content
	}

	return &Message{
		MsgID:     fmt.Sprintf("LEGACY-LINE-%09d", lineNo),
		Timestamp: ts.UTC(),
		Type:      msgType,
		Body:      body,
	}, true
}

func parseLegacyTimestamp(raw string) (time.Time, bool) {
	for _, layout := range legacyTimestampLayouts {
		ts, err := time.Parse(layout, raw)
		if err == nil {
			return ts.UTC(), true
		}
	}
	return time.Time{}, false
}

func isLegacyTypeToken(token string) bool {
	if token == "" {
		return false
	}
	for _, ch := range token {
		if ch >= 'a' && ch <= 'z' {
			continue
		}
		if ch >= 'A' && ch <= 'Z' {
			continue
		}
		if ch >= '0' && ch <= '9' {
			continue
		}
		switch ch {
		case '_', '-', '/':
			continue
		default:
			return false
		}
	}
	return true
}

func finalizeBody(body []byte) string {
	if len(body) == 0 {
		return ""
	}
	text := string(body)
	text = strings.TrimSuffix(text, "\n")
	text = strings.TrimSuffix(text, "\r")
	return text
}

func filterSince(messages []*Message, sinceID string) ([]*Message, error) {
	if strings.TrimSpace(sinceID) == "" {
		return messages, nil
	}
	for i, msg := range messages {
		if msg != nil && msg.MsgID == sinceID {
			if i+1 >= len(messages) {
				return []*Message{}, nil
			}
			return messages[i+1:], nil
		}
	}
	return nil, fmt.Errorf("since id %q not found: %w", sinceID, ErrSinceIDNotFound)
}

func validateBusPath(path string) error {
	if strings.TrimSpace(path) == "" {
		return errors.New("message bus path is empty")
	}
	info, err := os.Lstat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return errors.Wrap(err, "stat message bus path")
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return errors.Errorf("message bus path %q is a symlink", path)
	}
	if !info.Mode().IsRegular() {
		return errors.Errorf("message bus path %q is not a regular file", path)
	}
	return nil
}
