package api

import (
	"bufio"
	"context"
	"encoding/json"
	stderrors "errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/jonnyzzz/conductor-loop/internal/messagebus"
	"github.com/jonnyzzz/conductor-loop/internal/runstate"
	"github.com/jonnyzzz/conductor-loop/internal/storage"
	"github.com/pkg/errors"
)

const (
	defaultPollInterval      = 500 * time.Millisecond
	defaultDiscoveryInterval = time.Second
	defaultHeartbeatInterval = 30 * time.Second
	defaultMaxClientsPerRun  = 10
)

var ErrMaxClientsReached = stderrors.New("max clients reached for run")

// SSEConfig configures log streaming behavior.
type SSEConfig struct {
	PollInterval      time.Duration
	DiscoveryInterval time.Duration
	HeartbeatInterval time.Duration
	MaxClientsPerRun  int
}

func (c SSEConfig) withDefaults() SSEConfig {
	if c.PollInterval <= 0 {
		c.PollInterval = defaultPollInterval
	}
	if c.DiscoveryInterval <= 0 {
		c.DiscoveryInterval = defaultDiscoveryInterval
	}
	if c.HeartbeatInterval <= 0 {
		c.HeartbeatInterval = defaultHeartbeatInterval
	}
	if c.MaxClientsPerRun <= 0 {
		c.MaxClientsPerRun = defaultMaxClientsPerRun
	}
	return c
}

func (s *Server) handleAllRunsStream(w http.ResponseWriter, r *http.Request) *apiError {
	if r.Method != http.MethodGet {
		return apiErrorMethodNotAllowed()
	}
	return s.streamAllRuns(w, r)
}

func (s *Server) handleMessageStream(w http.ResponseWriter, r *http.Request) *apiError {
	if r.Method != http.MethodGet {
		return apiErrorMethodNotAllowed()
	}
	return s.streamMessages(w, r)
}

func (s *Server) sseConfig() SSEConfig {
	if s == nil {
		return SSEConfig{}.withDefaults()
	}
	cfg := SSEConfig{
		PollInterval:      time.Duration(s.apiConfig.SSE.PollIntervalMs) * time.Millisecond,
		DiscoveryInterval: time.Duration(s.apiConfig.SSE.DiscoveryIntervalMs) * time.Millisecond,
		HeartbeatInterval: time.Duration(s.apiConfig.SSE.HeartbeatIntervalS) * time.Second,
		MaxClientsPerRun:  s.apiConfig.SSE.MaxClientsPerRun,
	}
	return cfg.withDefaults()
}

func (s *Server) sseManager() (*StreamManager, error) {
	if s == nil {
		return nil, errors.New("server is nil")
	}
	s.sseOnce.Do(func() {
		s.sseManagerInst, s.sseErr = NewStreamManager(s.rootDir, s.sseConfig())
	})
	return s.sseManagerInst, s.sseErr
}

func (s *Server) streamRun(w http.ResponseWriter, r *http.Request, runID string) *apiError {
	writer, err := newSSEWriter(w)
	if err != nil {
		return apiErrorBadRequest("sse not supported")
	}
	manager, err := s.sseManager()
	if err != nil {
		return apiErrorInternal("init sse manager", err)
	}
	cfg := s.sseConfig()
	cursor := parseCursor(r.Header.Get("Last-Event-ID"))
	sub, err := manager.SubscribeRun(runID, cursor)
	if err != nil {
		if stderrors.Is(err, ErrMaxClientsReached) {
			return &apiError{Status: http.StatusTooManyRequests, Code: "TOO_MANY_REQUESTS", Message: err.Error()}
		}
		return apiErrorNotFound("run not found")
	}
	defer sub.Close()

	ctx := r.Context()
	heartbeat := time.NewTicker(cfg.HeartbeatInterval)
	defer heartbeat.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-heartbeat.C:
			_ = writer.Send(SSEEvent{Event: "heartbeat", Data: "{}"})
		case ev, ok := <-sub.Events():
			if !ok {
				return nil
			}
			if err := writer.Send(ev); err != nil {
				return nil
			}
		}
	}
}

func (s *Server) streamAllRuns(w http.ResponseWriter, r *http.Request) *apiError {
	writer, err := newSSEWriter(w)
	if err != nil {
		return apiErrorBadRequest("sse not supported")
	}
	manager, err := s.sseManager()
	if err != nil {
		return apiErrorInternal("init sse manager", err)
	}
	cfg := s.sseConfig()
	ctx := r.Context()
	fan := newFanIn(ctx)
	defer fan.Close()

	subsMu := &sync.Mutex{}
	subs := make(map[string]*Subscription)
	addSub := func(runID string, sub *Subscription) {
		subsMu.Lock()
		defer subsMu.Unlock()
		subs[runID] = sub
		fan.Add(sub)
	}
	closeSubs := func() {
		subsMu.Lock()
		defer subsMu.Unlock()
		for _, sub := range subs {
			sub.Close()
		}
	}
	defer closeSubs()

	runs, err := listRunIDs(s.rootDir)
	if err != nil {
		return apiErrorInternal("list runs", err)
	}
	for _, runID := range runs {
		sub, err := manager.SubscribeRun(runID, Cursor{})
		if err != nil {
			continue
		}
		addSub(runID, sub)
	}

	discovery, err := NewRunDiscovery(s.rootDir, cfg.DiscoveryInterval)
	if err != nil {
		return apiErrorInternal("init discovery", err)
	}
	discoveryCtx, cancelDiscovery := context.WithCancel(ctx)
	defer cancelDiscovery()
	go discovery.Poll(discoveryCtx, cfg.DiscoveryInterval)
	go func() {
		for {
			select {
			case <-discoveryCtx.Done():
				return
			case runID := <-discovery.NewRuns():
				subsMu.Lock()
				_, exists := subs[runID]
				subsMu.Unlock()
				if exists {
					continue
				}
				sub, err := manager.SubscribeRun(runID, Cursor{})
				if err != nil {
					continue
				}
				addSub(runID, sub)
			}
		}
	}()

	heartbeat := time.NewTicker(cfg.HeartbeatInterval)
	defer heartbeat.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-heartbeat.C:
			_ = writer.Send(SSEEvent{Event: "heartbeat", Data: "{}"})
		case ev, ok := <-fan.Events():
			if !ok {
				return nil
			}
			if err := writer.Send(ev); err != nil {
				return nil
			}
		}
	}
}

func (s *Server) streamMessages(w http.ResponseWriter, r *http.Request) *apiError {
	projectID := strings.TrimSpace(r.URL.Query().Get("project_id"))
	if projectID == "" {
		return apiErrorBadRequest("project_id is required")
	}
	if err := validateIdentifier(projectID, "project_id"); err != nil {
		return err
	}
	taskID := strings.TrimSpace(r.URL.Query().Get("task_id"))
	if taskID != "" {
		if err := validateIdentifier(taskID, "task_id"); err != nil {
			return err
		}
	}

	busPath, pathErr := joinPathWithinRoot(s.rootDir, projectID, "PROJECT-MESSAGE-BUS.md")
	if pathErr != nil {
		return pathErr
	}
	if taskID != "" {
		busPath, pathErr = joinPathWithinRoot(s.rootDir, projectID, taskID, "TASK-MESSAGE-BUS.md")
		if pathErr != nil {
			return pathErr
		}
	}
	if err := requirePathWithinRoot(s.rootDir, busPath, "message bus path"); err != nil {
		return err
	}
	return s.streamMessageBusPath(w, r, busPath)
}

// streamMessageBusPath streams messages from a bus file as SSE events.
// It supports Last-Event-ID for resumable clients.
func (s *Server) streamMessageBusPath(w http.ResponseWriter, r *http.Request, busPath string) *apiError {
	writer, err := newSSEWriter(w)
	if err != nil {
		return apiErrorBadRequest("sse not supported")
	}
	ctx := r.Context()
	bus, err := messagebus.NewMessageBus(busPath)
	if err != nil {
		return apiErrorInternal("open message bus", err)
	}
	cfg := s.sseConfig()
	lastID := strings.TrimSpace(r.Header.Get("Last-Event-ID"))

	pollTicker := time.NewTicker(cfg.PollInterval)
	defer pollTicker.Stop()
	heartbeat := time.NewTicker(cfg.HeartbeatInterval)
	defer heartbeat.Stop()

	for {
		select {
		case <-ctx.Done():
			return nil
		case <-heartbeat.C:
			_ = writer.Send(SSEEvent{Event: "heartbeat", Data: "{}"})
		case <-pollTicker.C:
			messages, err := bus.ReadMessages(lastID)
			if err != nil {
				if stderrors.Is(err, messagebus.ErrSinceIDNotFound) {
					// sinceID expired (rotation/GC): reset to beginning and re-read
					// immediately so the panel is repopulated without waiting another tick.
					lastID = ""
					if msgs, rerr := bus.ReadMessages(""); rerr == nil {
						messages = msgs
					} else {
						continue
					}
				} else {
					continue
				}
			}
			for _, msg := range messages {
				if msg == nil {
					continue
				}
				ts := msg.Timestamp
				if ts.IsZero() {
					ts = time.Now().UTC()
				}
				// Build parent msg_id list for JSON (extract from Parents slice).
				var parentIDs []string
				for _, p := range msg.Parents {
					if p.MsgID != "" {
						parentIDs = append(parentIDs, p.MsgID)
					}
				}
				payload := messagePayload{
					MsgID:     msg.MsgID,
					Timestamp: ts.Format(time.RFC3339Nano),
					Type:      msg.Type,
					ProjectID: msg.ProjectID,
					TaskID:    msg.TaskID,
					RunID:     msg.RunID,
					IssueID:   msg.IssueID,
					Parents:   parentIDs,
					Meta:      msg.Meta,
					Body:      msg.Body,
				}
				data, err := json.Marshal(payload)
				if err != nil {
					continue
				}
				ev := SSEEvent{
					ID:    msg.MsgID, // set SSE id for resumable clients
					Event: "message",
					Data:  string(data),
				}
				if err := writer.Send(ev); err != nil {
					return nil
				}
				lastID = msg.MsgID
			}
		}
	}
}

// SSEEvent represents a single Server-Sent Event.
type SSEEvent struct {
	ID    string
	Event string
	Data  string
}

type sseWriter struct {
	w       http.ResponseWriter
	flusher http.Flusher
}

func newSSEWriter(w http.ResponseWriter) (*sseWriter, error) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		return nil, errors.New("response writer does not support flushing")
	}
	header := w.Header()
	header.Set("Content-Type", "text/event-stream")
	header.Set("Cache-Control", "no-cache")
	header.Set("Connection", "keep-alive")
	header.Set("X-Accel-Buffering", "no")
	return &sseWriter{w: w, flusher: flusher}, nil
}

func (s *sseWriter) Send(event SSEEvent) error {
	if s == nil {
		return errors.New("sse writer is nil")
	}
	if event.ID != "" {
		if _, err := fmt.Fprintf(s.w, "id: %s\n", event.ID); err != nil {
			return errors.Wrap(err, "write sse id")
		}
	}
	if event.Event != "" {
		if _, err := fmt.Fprintf(s.w, "event: %s\n", event.Event); err != nil {
			return errors.Wrap(err, "write sse event")
		}
	}
	data := event.Data
	if data == "" {
		data = "{}"
	}
	lines := strings.Split(data, "\n")
	for _, line := range lines {
		if _, err := fmt.Fprintf(s.w, "data: %s\n", line); err != nil {
			return errors.Wrap(err, "write sse data")
		}
	}
	if _, err := fmt.Fprint(s.w, "\n"); err != nil {
		return errors.Wrap(err, "write sse terminator")
	}
	s.flusher.Flush()
	return nil
}

type logPayload struct {
	RunID     string `json:"run_id"`
	ProjectID string `json:"project_id,omitempty"`
	TaskID    string `json:"task_id,omitempty"`
	Stream    string `json:"stream,omitempty"`
	Line      string `json:"line"`
	Timestamp string `json:"timestamp"`
}

type statusPayload struct {
	RunID     string `json:"run_id"`
	ProjectID string `json:"project_id,omitempty"`
	TaskID    string `json:"task_id,omitempty"`
	Status    string `json:"status"`
	ExitCode  int    `json:"exit_code"`
}

// messagePayload is the JSON payload for a message SSE event.
type messagePayload struct {
	MsgID     string            `json:"msg_id"`
	Timestamp string            `json:"timestamp"`
	Type      string            `json:"type,omitempty"`
	ProjectID string            `json:"project_id,omitempty"`
	TaskID    string            `json:"task_id,omitempty"`
	RunID     string            `json:"run_id,omitempty"`
	IssueID   string            `json:"issue_id,omitempty"`
	Parents   []string          `json:"parents,omitempty"` // msg_id strings for JSON simplicity
	Meta      map[string]string `json:"meta,omitempty"`
	Body      string            `json:"body"` // Body text
}

// Cursor tracks last-seen stdout/stderr line counts.
type Cursor struct {
	Stdout int64
	Stderr int64
}

func (c Cursor) isZero() bool {
	return c.Stdout == 0 && c.Stderr == 0
}

func parseCursor(raw string) Cursor {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return Cursor{}
	}
	if n, err := strconv.ParseInt(raw, 10, 64); err == nil {
		return Cursor{Stdout: n, Stderr: n}
	}
	var cursor Cursor
	parts := strings.FieldsFunc(raw, func(r rune) bool {
		return r == ';' || r == ',' || r == '|'
	})
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		key, val, ok := strings.Cut(part, "=")
		if !ok {
			key, val, ok = strings.Cut(part, ":")
		}
		if !ok {
			continue
		}
		key = strings.ToLower(strings.TrimSpace(key))
		val = strings.TrimSpace(val)
		num, err := strconv.ParseInt(val, 10, 64)
		if err != nil {
			continue
		}
		switch key {
		case "s", "stdout", "out":
			cursor.Stdout = num
		case "e", "stderr", "err":
			cursor.Stderr = num
		}
	}
	return cursor
}

func formatCursor(c Cursor) string {
	return fmt.Sprintf("s=%d;e=%d", c.Stdout, c.Stderr)
}

type subscriber struct {
	events  chan SSEEvent
	paused  bool
	pending []SSEEvent
	closed  bool
	mu      sync.Mutex
}

func newSubscriber(buffer int, paused bool) *subscriber {
	if buffer <= 0 {
		buffer = 32
	}
	return &subscriber{
		events: make(chan SSEEvent, buffer),
		paused: paused,
	}
}

func (s *subscriber) enqueue(event SSEEvent) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return false
	}
	if s.paused {
		s.pending = append(s.pending, event)
		return true
	}
	select {
	case s.events <- event:
		return true
	default:
		return false
	}
}

func (s *subscriber) sendDirect(event SSEEvent) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.closed {
		return
	}
	select {
	case s.events <- event:
	default:
	}
}

func (s *subscriber) resume() {
	s.mu.Lock()
	if s.closed {
		s.pending = nil
		s.mu.Unlock()
		return
	}
	s.paused = false
	pending := s.pending
	s.pending = nil
	for _, event := range pending {
		select {
		case s.events <- event:
		default:
		}
	}
	s.mu.Unlock()
}

func (s *subscriber) close() {
	s.mu.Lock()
	if s.closed {
		s.mu.Unlock()
		return
	}
	s.closed = true
	close(s.events)
	s.mu.Unlock()
}

// Subscription wraps a subscriber channel.
type Subscription struct {
	events <-chan SSEEvent
	close  func()
}

// Events returns the event channel.
func (s *Subscription) Events() <-chan SSEEvent {
	if s == nil {
		return nil
	}
	return s.events
}

// Close releases the subscription.
func (s *Subscription) Close() {
	if s == nil || s.close == nil {
		return
	}
	s.close()
}

// StreamManager manages run tailers and subscribers.
type StreamManager struct {
	rootDir       string
	pollInterval  time.Duration
	maxClientsRun int
	mu            sync.Mutex
	runs          map[string]*runStream
}

// NewStreamManager creates a StreamManager.
func NewStreamManager(rootDir string, cfg SSEConfig) (*StreamManager, error) {
	cleanRoot := filepath.Clean(strings.TrimSpace(rootDir))
	if cleanRoot == "." || cleanRoot == "" {
		return nil, errors.New("root directory is empty")
	}
	cfg = cfg.withDefaults()
	return &StreamManager{
		rootDir:       cleanRoot,
		pollInterval:  cfg.PollInterval,
		maxClientsRun: cfg.MaxClientsPerRun,
		runs:          make(map[string]*runStream),
	}, nil
}

// SubscribeRun registers a subscriber for a run.
func (m *StreamManager) SubscribeRun(runID string, cursor Cursor) (*Subscription, error) {
	rs, err := m.ensureRun(runID)
	if err != nil {
		return nil, err
	}
	sub, snapshot, err := rs.subscribe(cursor)
	if err != nil {
		return nil, err
	}
	if !cursor.isZero() && (cursor.Stdout < snapshot.Stdout || cursor.Stderr < snapshot.Stderr) {
		go rs.catchUp(sub, cursor, snapshot)
	}
	return &Subscription{
		events: sub.events,
		close: func() {
			m.unsubscribe(runID, sub)
		},
	}, nil
}

func (m *StreamManager) ensureRun(runID string) (*runStream, error) {
	cleanID := strings.TrimSpace(runID)
	if cleanID == "" {
		return nil, errors.New("run id is empty")
	}
	path, err := findRunInfoPath(m.rootDir, cleanID)
	if err != nil {
		return nil, err
	}
	runDir := filepath.Dir(path)
	projectID := ""
	taskID := ""
	if info, readErr := runstate.ReadRunInfo(path); readErr == nil {
		projectID = strings.TrimSpace(info.ProjectID)
		taskID = strings.TrimSpace(info.TaskID)
	}
	if projectID == "" || taskID == "" {
		inferredProjectID, inferredTaskID := inferRunScopeFromDir(runDir)
		if projectID == "" {
			projectID = inferredProjectID
		}
		if taskID == "" {
			taskID = inferredTaskID
		}
	}

	m.mu.Lock()
	defer m.mu.Unlock()
	if existing, ok := m.runs[cleanID]; ok {
		return existing, nil
	}
	rs := newRunStream(cleanID, runDir, projectID, taskID, m.pollInterval, m.maxClientsRun)
	m.runs[cleanID] = rs
	return rs, nil
}

func inferRunScopeFromDir(runDir string) (projectID string, taskID string) {
	cleanRunDir := filepath.Clean(strings.TrimSpace(runDir))
	if cleanRunDir == "." || cleanRunDir == "" {
		return "", ""
	}
	runsDir := filepath.Dir(cleanRunDir)
	if filepath.Base(runsDir) != "runs" {
		return "", ""
	}
	taskDir := filepath.Dir(runsDir)
	projectDir := filepath.Dir(taskDir)
	taskID = strings.TrimSpace(filepath.Base(taskDir))
	projectID = strings.TrimSpace(filepath.Base(projectDir))
	if taskID == "." || taskID == string(filepath.Separator) {
		taskID = ""
	}
	if projectID == "." || projectID == string(filepath.Separator) {
		projectID = ""
	}
	return projectID, taskID
}

func (m *StreamManager) unsubscribe(runID string, sub *subscriber) {
	m.mu.Lock()
	rs, ok := m.runs[runID]
	m.mu.Unlock()
	if !ok {
		return
	}
	rs.unsubscribe(sub)
	if rs.isEmpty() {
		m.mu.Lock()
		delete(m.runs, runID)
		m.mu.Unlock()
	}
}

type runStream struct {
	runID        string
	runDir       string
	projectID    string
	taskID       string
	pollInterval time.Duration
	maxClients   int

	mu          sync.Mutex
	subscribers map[*subscriber]struct{}
	logCh       chan LogLine
	stopCh      chan struct{}
	started     bool

	stdoutLines int64
	stderrLines int64
	lastStatus  string
	lastExit    int
}

func newRunStream(runID, runDir, projectID, taskID string, pollInterval time.Duration, maxClients int) *runStream {
	return &runStream{
		runID:        runID,
		runDir:       runDir,
		projectID:    strings.TrimSpace(projectID),
		taskID:       strings.TrimSpace(taskID),
		pollInterval: pollInterval,
		maxClients:   maxClients,
		subscribers:  make(map[*subscriber]struct{}),
		lastExit:     -1,
	}
}

func (rs *runStream) subscribe(cursor Cursor) (*subscriber, Cursor, error) {
	rs.mu.Lock()
	defer rs.mu.Unlock()
	if rs.maxClients > 0 && len(rs.subscribers) >= rs.maxClients {
		return nil, Cursor{}, ErrMaxClientsReached
	}
	if !rs.started {
		rs.startLocked()
	}
	snapshot := Cursor{Stdout: rs.stdoutLines, Stderr: rs.stderrLines}
	paused := !cursor.isZero() && (cursor.Stdout < snapshot.Stdout || cursor.Stderr < snapshot.Stderr)
	sub := newSubscriber(128, paused)
	rs.subscribers[sub] = struct{}{}
	return sub, snapshot, nil
}

func (rs *runStream) unsubscribe(sub *subscriber) {
	rs.mu.Lock()
	defer rs.mu.Unlock()
	delete(rs.subscribers, sub)
	sub.close()
	if len(rs.subscribers) == 0 {
		rs.stopLocked()
	}
}

func (rs *runStream) isEmpty() bool {
	rs.mu.Lock()
	defer rs.mu.Unlock()
	return len(rs.subscribers) == 0
}

func (rs *runStream) startLocked() {
	if rs.started {
		return
	}
	rs.logCh = make(chan LogLine, 256)
	rs.stopCh = make(chan struct{})
	rs.started = true

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		<-rs.stopCh
		cancel()
	}()

	stdoutPath := filepath.Join(rs.runDir, "agent-stdout.txt")
	stderrPath := filepath.Join(rs.runDir, "agent-stderr.txt")
	if count, err := countLines(stdoutPath); err == nil {
		rs.stdoutLines = count
	}
	if count, err := countLines(stderrPath); err == nil {
		rs.stderrLines = count
	}
	stdoutTailer, _ := NewTailer(stdoutPath, rs.runID, "stdout", rs.pollInterval, -1, rs.logCh)
	stderrTailer, _ := NewTailer(stderrPath, rs.runID, "stderr", rs.pollInterval, -1, rs.logCh)
	if stdoutTailer != nil {
		stdoutTailer.Start(ctx)
	}
	if stderrTailer != nil {
		stderrTailer.Start(ctx)
	}
	go rs.loop(ctx, stdoutTailer, stderrTailer)
}

func (rs *runStream) stopLocked() {
	if !rs.started {
		return
	}
	close(rs.stopCh)
	rs.started = false
}

func (rs *runStream) loop(ctx context.Context, stdoutTailer, stderrTailer *Tailer) {
	statusTicker := time.NewTicker(rs.pollInterval)
	defer statusTicker.Stop()
	for {
		select {
		case <-ctx.Done():
			if stdoutTailer != nil {
				stdoutTailer.Stop()
			}
			if stderrTailer != nil {
				stderrTailer.Stop()
			}
			return
		case line := <-rs.logCh:
			rs.handleLogLine(line)
		case <-statusTicker.C:
			rs.checkStatus()
		}
	}
}

func (rs *runStream) handleLogLine(line LogLine) {
	var cursor Cursor
	var projectID string
	var taskID string
	rs.mu.Lock()
	switch line.Stream {
	case "stdout":
		rs.stdoutLines++
	case "stderr":
		rs.stderrLines++
	}
	cursor = Cursor{Stdout: rs.stdoutLines, Stderr: rs.stderrLines}
	subs := make([]*subscriber, 0, len(rs.subscribers))
	for sub := range rs.subscribers {
		subs = append(subs, sub)
	}
	projectID = rs.projectID
	taskID = rs.taskID
	rs.mu.Unlock()

	payload := logPayload{
		RunID:     line.RunID,
		ProjectID: projectID,
		TaskID:    taskID,
		Stream:    line.Stream,
		Line:      line.Line,
		Timestamp: line.Timestamp.Format(time.RFC3339Nano),
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return
	}
	event := SSEEvent{
		ID:    formatCursor(cursor),
		Event: "log",
		Data:  string(data),
	}
	for _, sub := range subs {
		if !sub.enqueue(event) {
			rs.unsubscribe(sub)
		}
	}
}

func (rs *runStream) checkStatus() {
	path := filepath.Join(rs.runDir, "run-info.yaml")
	info, err := runstate.ReadRunInfo(path)
	if err != nil {
		return
	}
	status := strings.TrimSpace(info.Status)
	projectID := strings.TrimSpace(info.ProjectID)
	taskID := strings.TrimSpace(info.TaskID)
	exitCode := info.ExitCode
	if status == "" && exitCode >= 0 {
		status = storage.StatusCompleted
	}
	// Emit the first observed non-empty status (including running/queued) so
	// the UI can react immediately to newly discovered runs.
	shouldReport := status != ""
	rs.mu.Lock()
	if projectID != "" {
		rs.projectID = projectID
	}
	if taskID != "" {
		rs.taskID = taskID
	}
	changed := shouldReport && (status != rs.lastStatus || exitCode != rs.lastExit)
	if changed {
		rs.lastStatus = status
		rs.lastExit = exitCode
	}
	projectID = rs.projectID
	taskID = rs.taskID
	subs := make([]*subscriber, 0, len(rs.subscribers))
	for sub := range rs.subscribers {
		subs = append(subs, sub)
	}
	rs.mu.Unlock()
	if !changed {
		return
	}
	payload := statusPayload{
		RunID:     rs.runID,
		ProjectID: projectID,
		TaskID:    taskID,
		Status:    status,
		ExitCode:  exitCode,
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return
	}
	event := SSEEvent{
		Event: "status",
		Data:  string(data),
	}
	for _, sub := range subs {
		if !sub.enqueue(event) {
			rs.unsubscribe(sub)
		}
	}
}

func (rs *runStream) catchUp(sub *subscriber, cursor, snapshot Cursor) {
	stdoutPath := filepath.Join(rs.runDir, "agent-stdout.txt")
	stderrPath := filepath.Join(rs.runDir, "agent-stderr.txt")
	current := cursor
	if snapshot.Stdout > cursor.Stdout {
		lines, err := readLinesRange(stdoutPath, cursor.Stdout, snapshot.Stdout)
		if err == nil {
			for _, line := range lines {
				current.Stdout++
				rs.sendCatchup(sub, current, "stdout", line)
			}
		}
	}
	if snapshot.Stderr > cursor.Stderr {
		lines, err := readLinesRange(stderrPath, cursor.Stderr, snapshot.Stderr)
		if err == nil {
			for _, line := range lines {
				current.Stderr++
				rs.sendCatchup(sub, current, "stderr", line)
			}
		}
	}
	sub.resume()
}

func (rs *runStream) sendCatchup(sub *subscriber, cursor Cursor, stream, line string) {
	rs.mu.Lock()
	projectID := rs.projectID
	taskID := rs.taskID
	rs.mu.Unlock()
	payload := logPayload{
		RunID:     rs.runID,
		ProjectID: projectID,
		TaskID:    taskID,
		Stream:    stream,
		Line:      line,
		Timestamp: time.Now().UTC().Format(time.RFC3339Nano),
	}
	data, err := json.Marshal(payload)
	if err != nil {
		return
	}
	event := SSEEvent{
		ID:    formatCursor(cursor),
		Event: "log",
		Data:  string(data),
	}
	sub.sendDirect(event)
}

func readLinesRange(path string, startLine, endLine int64) ([]string, error) {
	if endLine <= startLine {
		return nil, nil
	}
	file, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, errors.Wrap(err, "open log file")
	}
	defer file.Close()

	reader := bufio.NewReader(file)
	var (
		lines []string
		count int64
	)
	for count < endLine {
		part, err := reader.ReadString('\n')
		if part != "" {
			count++
			if count > startLine {
				line := strings.TrimSuffix(part, "\n")
				line = strings.TrimSuffix(line, "\r")
				lines = append(lines, line)
			}
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, errors.Wrap(err, "read log file")
		}
	}
	return lines, nil
}

func countLines(path string) (int64, error) {
	file, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, nil
		}
		return 0, errors.Wrap(err, "open log file")
	}
	defer file.Close()

	reader := bufio.NewReader(file)
	var count int64
	for {
		part, err := reader.ReadString('\n')
		if part != "" {
			count++
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			return 0, errors.Wrap(err, "read log file")
		}
	}
	return count, nil
}

type fanIn struct {
	ctx    context.Context
	events chan SSEEvent
	wg     sync.WaitGroup
}

func newFanIn(ctx context.Context) *fanIn {
	return &fanIn{
		ctx:    ctx,
		events: make(chan SSEEvent, 256),
	}
}

func (f *fanIn) Add(sub *Subscription) {
	if f == nil || sub == nil {
		return
	}
	f.wg.Add(1)
	go func() {
		defer f.wg.Done()
		for {
			select {
			case <-f.ctx.Done():
				return
			case ev, ok := <-sub.Events():
				if !ok {
					return
				}
				select {
				case f.events <- ev:
				case <-f.ctx.Done():
					return
				}
			}
		}
	}()
}

func (f *fanIn) Events() <-chan SSEEvent {
	if f == nil {
		return nil
	}
	return f.events
}

func (f *fanIn) Close() {
	if f == nil {
		return
	}
	f.wg.Wait()
	close(f.events)
}
