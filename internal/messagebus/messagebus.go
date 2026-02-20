package messagebus

import (
	"bufio"
	"bytes"
	stderrors "errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"time"

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
	path         string
	now          func() time.Time
	lockTimeout  time.Duration
	pollInterval time.Duration
	maxRetries   int
	retryBackoff time.Duration
	fsync        bool

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
			return msg.MsgID, nil
		}
		if !stderrors.Is(lastErr, ErrLockTimeout) {
			return "", lastErr
		}
	}
	return "", fmt.Errorf("append failed after %d attempts: %w", mb.maxRetries, lastErr)
}

func (mb *MessageBus) tryAppend(data []byte) error {
	file, err := os.OpenFile(mb.path, os.O_WRONLY|os.O_APPEND|os.O_CREATE, messageBusFileMode)
	if err != nil {
		return errors.Wrap(err, "open message bus")
	}
	defer file.Close()

	if err := LockExclusive(file, mb.lockTimeout); err != nil {
		return fmt.Errorf("lock message bus: %w", err)
	}
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

// ReadMessages reads messages after sinceID. If sinceID is empty, returns all messages.
func (mb *MessageBus) ReadMessages(sinceID string) ([]*Message, error) {
	if mb == nil {
		return nil, errors.New("message bus is nil")
	}
	if err := validateBusPath(mb.path); err != nil {
		return nil, errors.Wrap(err, "validate message bus path")
	}
	data, err := os.ReadFile(mb.path)
	if err != nil {
		if os.IsNotExist(err) {
			return []*Message{}, nil
		}
		return nil, errors.Wrap(err, "read message bus")
	}
	messages, err := parseMessages(data)
	if err != nil {
		return nil, err
	}
	return filterSince(messages, sinceID)
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
	reader := bufio.NewReader(bytes.NewReader(data))
	const (
		stateSeekHeader = iota
		stateHeader
		stateBody
	)
	state := stateSeekHeader
	var headerBuf bytes.Buffer
	var bodyBuf bytes.Buffer
	var current *Message
	messages := make([]*Message, 0)

	for {
		line, err := reader.ReadString('\n')
		if err != nil && err != io.EOF {
			return nil, errors.Wrap(err, "read message bus")
		}
		if err == io.EOF && line == "" {
			break
		}
		trimmed := strings.TrimRight(line, "\r\n")
		switch state {
		case stateSeekHeader:
			if trimmed == "---" {
				state = stateHeader
				headerBuf.Reset()
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
					messages = append(messages, current)
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
			messages = append(messages, current)
		}
	}
	return messages, nil
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
