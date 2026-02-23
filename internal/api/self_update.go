package api

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/jonnyzzz/conductor-loop/internal/obslog"
	"github.com/pkg/errors"
)

const (
	selfUpdateStateIdle     = "idle"
	selfUpdateStateDeferred = "deferred"
	selfUpdateStateApplying = "applying"
	selfUpdateStateFailed   = "failed"

	defaultSelfUpdatePollInterval = time.Second
	defaultSelfUpdateVerifyTO     = 10 * time.Second
)

type selfUpdateRequestPayload struct {
	BinaryPath string `json:"binary_path"`
}

type selfUpdateStatusResponse struct {
	State               string    `json:"state"`
	BinaryPath          string    `json:"binary_path,omitempty"`
	RequestedAt         time.Time `json:"requested_at,omitempty"`
	StartedAt           time.Time `json:"started_at,omitempty"`
	FinishedAt          time.Time `json:"finished_at,omitempty"`
	ActiveRunsAtRequest int       `json:"active_runs_at_request,omitempty"`
	ActiveRunsNow       int       `json:"active_runs_now"`
	ActiveRunsError     string    `json:"active_runs_error,omitempty"`
	LastError           string    `json:"last_error,omitempty"`
	LastNote            string    `json:"last_note,omitempty"`
}

type selfUpdateState struct {
	State               string
	BinaryPath          string
	RequestedAt         time.Time
	StartedAt           time.Time
	FinishedAt          time.Time
	ActiveRunsAtRequest int
	LastError           string
	LastNote            string
}

type selfUpdateOptions struct {
	Logger              *log.Logger
	Now                 func() time.Time
	PollInterval        time.Duration
	CountActiveRootRuns func() (int, error)
	ResolveExecutable   func() (string, error)
	VerifyBinary        func(path string) error
	InstallBinary       func(candidate, current string, now time.Time) (func() error, error)
	Reexec              func(path string, args []string, env []string) error
	OnDrainReleased     func()
}

type selfUpdateManager struct {
	mu sync.Mutex

	logger *log.Logger
	now    func() time.Time

	pollInterval    time.Duration
	countActive     func() (int, error)
	resolveExe      func() (string, error)
	verify          func(path string) error
	install         func(candidate, current string, now time.Time) (func() error, error)
	reexec          func(path string, args []string, env []string) error
	onDrainReleased func()

	state         selfUpdateState
	workerRunning bool
}

func newSelfUpdateManager(opts selfUpdateOptions) *selfUpdateManager {
	now := opts.Now
	if now == nil {
		now = time.Now
	}
	countActive := opts.CountActiveRootRuns
	if countActive == nil {
		countActive = func() (int, error) { return 0, nil }
	}
	resolveExe := opts.ResolveExecutable
	if resolveExe == nil {
		resolveExe = os.Executable
	}
	verify := opts.VerifyBinary
	if verify == nil {
		verify = func(path string) error {
			return verifyCandidateBinary(path, defaultSelfUpdateVerifyTO)
		}
	}
	install := opts.InstallBinary
	if install == nil {
		install = replaceExecutableWithRollback
	}
	reexec := opts.Reexec
	if reexec == nil {
		reexec = defaultSelfUpdateReexec
	}
	pollInterval := opts.PollInterval
	if pollInterval <= 0 {
		pollInterval = defaultSelfUpdatePollInterval
	}
	return &selfUpdateManager{
		logger:          opts.Logger,
		now:             now,
		pollInterval:    pollInterval,
		countActive:     countActive,
		resolveExe:      resolveExe,
		verify:          verify,
		install:         install,
		reexec:          reexec,
		onDrainReleased: opts.OnDrainReleased,
		state: selfUpdateState{
			State: selfUpdateStateIdle,
		},
	}
}

func (m *selfUpdateManager) status() selfUpdateStatusResponse {
	if m == nil {
		return selfUpdateStatusResponse{
			State:         selfUpdateStateFailed,
			LastError:     "self-update manager is nil",
			ActiveRunsNow: -1,
		}
	}
	m.mu.Lock()
	current := m.state
	m.mu.Unlock()

	resp := selfUpdateStatusResponse{
		State:               current.State,
		BinaryPath:          current.BinaryPath,
		RequestedAt:         current.RequestedAt,
		StartedAt:           current.StartedAt,
		FinishedAt:          current.FinishedAt,
		ActiveRunsAtRequest: current.ActiveRunsAtRequest,
		LastError:           current.LastError,
		LastNote:            current.LastNote,
		ActiveRunsNow:       -1,
	}
	active, err := m.countActive()
	if err != nil {
		resp.ActiveRunsError = err.Error()
		return resp
	}
	resp.ActiveRunsNow = active
	return resp
}

func (m *selfUpdateManager) blocksNewRootRuns() bool {
	if m == nil {
		return false
	}
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.state.State == selfUpdateStateDeferred || m.state.State == selfUpdateStateApplying
}

func (m *selfUpdateManager) request(binaryPath string) (selfUpdateStatusResponse, int, error) {
	if m == nil {
		return selfUpdateStatusResponse{}, http.StatusInternalServerError, errors.New("self-update manager is nil")
	}
	candidate, err := normalizeBinaryPath(binaryPath)
	if err != nil {
		return selfUpdateStatusResponse{}, http.StatusBadRequest, err
	}
	active, err := m.countActive()
	if err != nil {
		return selfUpdateStatusResponse{}, http.StatusInternalServerError, errors.Wrap(err, "count active root runs")
	}

	now := m.now().UTC()
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.state.State == selfUpdateStateApplying {
		snapshot := m.snapshotLocked(active, "")
		return snapshot, http.StatusConflict, errors.New("self-update is already in progress")
	}

	m.state.State = selfUpdateStateDeferred
	m.state.BinaryPath = candidate
	m.state.RequestedAt = now
	m.state.StartedAt = time.Time{}
	m.state.FinishedAt = time.Time{}
	m.state.ActiveRunsAtRequest = active
	m.state.LastError = ""
	m.state.LastNote = ""

	if active > 0 {
		if !m.workerRunning {
			m.workerRunning = true
			go m.runDeferredWorker()
		}
		m.log("INFO", "self_update_deferred", map[string]any{
			"binary_path": candidate,
			"active_runs": active,
		})
		return m.snapshotLocked(active, ""), http.StatusAccepted, nil
	}

	m.state.State = selfUpdateStateApplying
	m.state.StartedAt = now
	m.log("INFO", "self_update_started_immediately", map[string]any{
		"binary_path": candidate,
	})
	go m.apply(candidate)
	return m.snapshotLocked(active, ""), http.StatusAccepted, nil
}

func (m *selfUpdateManager) runDeferredWorker() {
	if m == nil {
		return
	}
	ticker := time.NewTicker(m.pollInterval)
	defer ticker.Stop()

	for {
		m.mu.Lock()
		if m.state.State != selfUpdateStateDeferred {
			m.workerRunning = false
			m.mu.Unlock()
			return
		}
		candidate := m.state.BinaryPath
		m.mu.Unlock()

		active, err := m.countActive()
		if err != nil {
			m.setFailure("count active root runs", err)
			return
		}
		if active > 0 {
			<-ticker.C
			continue
		}

		m.mu.Lock()
		if m.state.State != selfUpdateStateDeferred {
			m.workerRunning = false
			m.mu.Unlock()
			return
		}
		candidate = m.state.BinaryPath
		m.state.State = selfUpdateStateApplying
		m.state.StartedAt = m.now().UTC()
		m.state.LastError = ""
		m.state.LastNote = ""
		m.mu.Unlock()

		m.log("INFO", "self_update_promoted_from_deferred", map[string]any{
			"binary_path": candidate,
		})
		m.apply(candidate)
		return
	}
}

func (m *selfUpdateManager) apply(candidate string) {
	if m == nil {
		return
	}
	if err := m.verify(candidate); err != nil {
		m.setFailure("verify candidate binary", err)
		return
	}
	currentExe, err := m.resolveExe()
	if err != nil {
		m.setFailure("resolve current executable", err)
		return
	}
	rollback, err := m.install(candidate, currentExe, m.now().UTC())
	if err != nil {
		m.setFailure("install candidate binary", err)
		return
	}

	args := append([]string(nil), os.Args...)
	env := append([]string(nil), os.Environ()...)
	if err := m.reexec(currentExe, args, env); err != nil {
		if rollback != nil {
			if rollbackErr := rollback(); rollbackErr != nil {
				err = fmt.Errorf("%w; rollback failed: %v", err, rollbackErr)
			}
		}
		m.setFailure("handoff to updated binary", err)
		return
	}

	// The production reexec implementation does not return on success.
	m.mu.Lock()
	m.state.State = selfUpdateStateIdle
	m.state.FinishedAt = m.now().UTC()
	m.state.LastError = ""
	m.state.LastNote = "handoff completed"
	m.workerRunning = false
	m.mu.Unlock()
}

func (m *selfUpdateManager) setFailure(context string, err error) {
	if m == nil {
		return
	}
	message := context
	if err != nil {
		message = fmt.Sprintf("%s: %v", context, err)
	}
	m.mu.Lock()
	m.state.State = selfUpdateStateFailed
	m.state.FinishedAt = m.now().UTC()
	m.state.LastError = message
	m.state.LastNote = ""
	m.workerRunning = false
	m.mu.Unlock()
	m.log("ERROR", "self_update_failed", map[string]any{
		"context": context,
		"error":   err,
	})
	if m.onDrainReleased != nil {
		m.onDrainReleased()
	}
}

func (m *selfUpdateManager) snapshotLocked(active int, activeErr string) selfUpdateStatusResponse {
	return selfUpdateStatusResponse{
		State:               m.state.State,
		BinaryPath:          m.state.BinaryPath,
		RequestedAt:         m.state.RequestedAt,
		StartedAt:           m.state.StartedAt,
		FinishedAt:          m.state.FinishedAt,
		ActiveRunsAtRequest: m.state.ActiveRunsAtRequest,
		ActiveRunsNow:       active,
		ActiveRunsError:     activeErr,
		LastError:           m.state.LastError,
		LastNote:            m.state.LastNote,
	}
}

func (m *selfUpdateManager) log(level, event string, fields map[string]any) {
	if m == nil || m.logger == nil {
		return
	}
	payload := make([]obslog.Field, 0, len(fields))
	for key, value := range fields {
		payload = append(payload, obslog.F(key, value))
	}
	obslog.Log(m.logger, level, "api", event, payload...)
}

func normalizeBinaryPath(path string) (string, error) {
	trimmed := strings.TrimSpace(path)
	if trimmed == "" {
		return "", errors.New("binary_path is required")
	}
	abs, err := filepath.Abs(trimmed)
	if err != nil {
		return "", errors.Wrap(err, "resolve binary_path")
	}
	info, err := os.Stat(abs)
	if err != nil {
		return "", errors.Wrap(err, "stat binary_path")
	}
	if info.IsDir() {
		return "", errors.New("binary_path must point to a file")
	}
	if !isExecutable(info.Mode()) {
		return "", errors.New("binary_path is not executable")
	}
	return abs, nil
}

func verifyCandidateBinary(path string, timeout time.Duration) error {
	if timeout <= 0 {
		timeout = defaultSelfUpdateVerifyTO
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, path, "--version")
	output, err := cmd.CombinedOutput()
	if err != nil {
		text := strings.TrimSpace(string(output))
		if text == "" {
			return errors.Wrap(err, "run --version")
		}
		return fmt.Errorf("run --version: %w: %s", err, text)
	}
	return nil
}

func replaceExecutableWithRollback(candidate, current string, now time.Time) (func() error, error) {
	source := filepath.Clean(strings.TrimSpace(candidate))
	target := filepath.Clean(strings.TrimSpace(current))
	if source == "" || source == "." {
		return nil, errors.New("candidate path is empty")
	}
	if target == "" || target == "." {
		return nil, errors.New("target path is empty")
	}
	if source == target {
		return func() error { return nil }, nil
	}

	stamp := timestampSuffix(now)
	backupPath := target + ".rollback-" + stamp
	if err := copyFile(target, backupPath); err != nil {
		return nil, errors.Wrap(err, "create rollback backup")
	}
	tmpPath := target + ".update-" + stamp + ".tmp"
	if err := copyFile(source, tmpPath); err != nil {
		return nil, errors.Wrap(err, "stage candidate binary")
	}
	if err := renameReplace(tmpPath, target); err != nil {
		return nil, errors.Wrap(err, "activate candidate binary")
	}

	rollback := func() error {
		rbTmp := target + ".rollback-restore-" + timestampSuffix(time.Now().UTC()) + ".tmp"
		if err := copyFile(backupPath, rbTmp); err != nil {
			return errors.Wrap(err, "stage rollback binary")
		}
		if err := renameReplace(rbTmp, target); err != nil {
			return errors.Wrap(err, "restore rollback binary")
		}
		return nil
	}
	return rollback, nil
}

func copyFile(source, target string) error {
	src, err := os.Open(source)
	if err != nil {
		return errors.Wrap(err, "open source")
	}
	defer src.Close()

	info, err := src.Stat()
	if err != nil {
		return errors.Wrap(err, "stat source")
	}

	dst, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, info.Mode()&0o777)
	if err != nil {
		return errors.Wrap(err, "open target")
	}

	success := false
	defer func() {
		if success {
			return
		}
		_ = dst.Close()
		_ = os.Remove(target)
	}()

	if _, err := io.Copy(dst, src); err != nil {
		return errors.Wrap(err, "copy file bytes")
	}
	if err := dst.Sync(); err != nil {
		return errors.Wrap(err, "sync target")
	}
	if err := dst.Close(); err != nil {
		return errors.Wrap(err, "close target")
	}
	success = true
	return nil
}

func renameReplace(source, target string) error {
	if err := os.Rename(source, target); err == nil {
		return nil
	}
	if removeErr := os.Remove(target); removeErr != nil && !os.IsNotExist(removeErr) {
		return removeErr
	}
	return os.Rename(source, target)
}

func timestampSuffix(now time.Time) string {
	if now.IsZero() {
		now = time.Now().UTC()
	}
	return now.UTC().Format("20060102150405")
}

func isExecutable(mode os.FileMode) bool {
	return mode&0o111 != 0
}

func (s *Server) countActiveRootRuns() (int, error) {
	if s == nil {
		return 0, nil
	}
	infos, err := allRunInfos(s.rootDir)
	if err != nil {
		return 0, err
	}
	count := 0
	for _, info := range infos {
		if isRunningRootRun(info) {
			count++
		}
	}
	if inMemory := int(s.activeRootRuns.Load()); inMemory > count {
		count = inMemory
	}
	return count, nil
}

func (s *Server) handleSelfUpdate(w http.ResponseWriter, r *http.Request) *apiError {
	if s == nil || s.selfUpdate == nil {
		return apiErrorInternal("self-update is unavailable", nil)
	}

	switch r.Method {
	case http.MethodGet:
		return writeJSON(w, http.StatusOK, s.selfUpdate.status())
	case http.MethodPost:
		var req selfUpdateRequestPayload
		if err := decodeJSON(r, &req); err != nil {
			return err
		}
		s.rootRunGateMu.Lock()
		status, code, err := s.selfUpdate.request(req.BinaryPath)
		s.rootRunGateMu.Unlock()
		if err != nil {
			switch code {
			case http.StatusBadRequest:
				return apiErrorBadRequest(err.Error())
			case http.StatusConflict:
				return apiErrorConflict(err.Error(), nil)
			default:
				return apiErrorInternal("request self-update", err)
			}
		}
		return writeJSON(w, code, status)
	default:
		return apiErrorMethodNotAllowed()
	}
}
