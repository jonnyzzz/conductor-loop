// Package api provides the REST API server for Conductor Loop.
package api

import (
	"context"
	stderrors "errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/jonnyzzz/conductor-loop/internal/config"
	"github.com/jonnyzzz/conductor-loop/internal/metrics"
	"github.com/jonnyzzz/conductor-loop/internal/runner"
	"github.com/pkg/errors"
)

// Options configures the REST API server.
type Options struct {
	RootDir          string
	ExtraRoots       []string
	ConfigPath       string
	APIConfig        config.APIConfig
	Version          string
	AgentNames       []string
	Logger           *log.Logger
	DisableTaskStart bool
	Now              func() time.Time
	Metrics          *metrics.Registry
}

// Server serves REST API endpoints for tasks and runs.
type Server struct {
	apiConfig  config.APIConfig
	rootDir    string
	extraRoots []string
	configPath string
	version    string
	agentNames []string
	startTime  time.Time
	logger     *log.Logger
	now        func() time.Time
	startTasks bool
	handler    http.Handler
	server     *http.Server
	metrics    *metrics.Registry

	actualPort int

	mu     sync.Mutex
	taskWg sync.WaitGroup

	sseOnce        sync.Once
	sseManagerInst *StreamManager
	sseErr         error
}

// WaitForTasks waits for all background task goroutines to finish.
// Call this in tests before temp directory cleanup to avoid races.
func (s *Server) WaitForTasks() {
	s.taskWg.Wait()
}

// NewServer constructs a REST API server with defaults applied.
func NewServer(opts Options) (*Server, error) {
	rootDir, err := resolveRootDir(opts.RootDir)
	if err != nil {
		return nil, err
	}

	cfg := opts.APIConfig
	if strings.TrimSpace(cfg.Host) == "" {
		cfg.Host = "0.0.0.0"
	}
	if cfg.Port == 0 {
		cfg.Port = 14355
	}

	logger := opts.Logger
	if logger == nil {
		logger = log.New(os.Stdout, "api ", log.LstdFlags)
	}

	now := opts.Now
	if now == nil {
		now = time.Now
	}

	version := strings.TrimSpace(opts.Version)
	if version == "" {
		version = "dev"
	}

	m := opts.Metrics
	if m == nil {
		m = metrics.New()
	}

	// Forward queued-run count changes from the runner semaphore to the metrics registry.
	runner.SetWaitingRunHook(func(delta int64) {
		m.RecordWaitingRun(delta)
	})

	s := &Server{
		apiConfig:  cfg,
		rootDir:    rootDir,
		extraRoots: opts.ExtraRoots,
		configPath: strings.TrimSpace(opts.ConfigPath),
		version:    version,
		agentNames: opts.AgentNames,
		startTime:  now(),
		logger:     logger,
		now:        now,
		startTasks: !opts.DisableTaskStart,
		metrics:    m,
	}
	s.handler = s.routes()
	return s, nil
}

// Handler returns the http.Handler for the server.
func (s *Server) Handler() http.Handler {
	if s == nil {
		return nil
	}
	return s.handler
}

// findFreeListener tries to bind on host:basePort, incrementing up to maxAttempts times.
// Returns the listener and the port it bound to.
func findFreeListener(host string, basePort, maxAttempts int) (net.Listener, int, error) {
	for i := 0; i < maxAttempts; i++ {
		port := basePort + i
		addr := net.JoinHostPort(host, strconv.Itoa(port))
		ln, err := net.Listen("tcp", addr)
		if err == nil {
			return ln, port, nil
		}
		var opErr *net.OpError
		if stderrors.As(err, &opErr) && stderrors.Is(opErr.Err, syscall.EADDRINUSE) {
			continue
		}
		return nil, 0, err
	}
	return nil, 0, fmt.Errorf("no free port in range %d-%d", basePort, basePort+maxAttempts-1)
}

// ListenAndServe starts the HTTP server.
// When explicit is true the configured port must be free or an error is returned.
// When explicit is false (default port) up to 100 consecutive ports are tried.
func (s *Server) ListenAndServe(explicit bool) error {
	if s == nil {
		return errors.New("server is nil")
	}
	maxAttempts := 1
	if !explicit {
		maxAttempts = 100
	}
	ln, actualPort, err := findFreeListener(s.apiConfig.Host, s.apiConfig.Port, maxAttempts)
	if err != nil {
		return err
	}
	s.mu.Lock()
	s.actualPort = actualPort
	if s.server == nil {
		s.server = &http.Server{Handler: s.handler}
	}
	srv := s.server
	s.mu.Unlock()
	apiURL, uiURL := startupURLs(s.apiConfig.Host, actualPort)
	s.logger.Printf("API listening on %s", apiURL)
	s.logger.Printf("Web UI available at %s", uiURL)
	return srv.Serve(ln)
}

// ActualPort returns the port the server bound to after ListenAndServe was called.
func (s *Server) ActualPort() int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.actualPort
}

// Shutdown gracefully stops the HTTP server.
func (s *Server) Shutdown(ctx context.Context) error {
	if s == nil {
		return nil
	}
	s.mu.Lock()
	srv := s.server
	s.mu.Unlock()
	if srv == nil {
		return nil
	}
	return srv.Shutdown(ctx)
}

func resolveRootDir(root string) (string, error) {
	trimmed := strings.TrimSpace(root)
	if trimmed == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", errors.Wrap(err, "resolve home dir")
		}
		trimmed = filepath.Join(home, "run-agent")
	}
	abs, err := filepath.Abs(trimmed)
	if err != nil {
		return "", errors.Wrap(err, "resolve root dir")
	}
	return abs, nil
}

func intToString(value int) string {
	return strconv.Itoa(value)
}

func startupURLs(host string, port int) (string, string) {
	listenURL := httpBaseURL(resolveListenHost(host), port) + "/"
	uiURL := httpBaseURL(resolveNavigationHost(host), port) + "/ui/"
	return listenURL, uiURL
}

func resolveListenHost(host string) string {
	host = trimBrackets(strings.TrimSpace(host))
	if host == "" {
		return "0.0.0.0"
	}
	return host
}

func resolveNavigationHost(host string) string {
	host = resolveListenHost(host)
	if isUnspecifiedHost(host) {
		return "localhost"
	}
	return host
}

func resolveLoopbackHost(host string) string {
	host = resolveListenHost(host)
	if isUnspecifiedHost(host) {
		return "127.0.0.1"
	}
	return host
}

func isUnspecifiedHost(host string) bool {
	ip := net.ParseIP(host)
	return ip != nil && ip.IsUnspecified()
}

func trimBrackets(host string) string {
	if strings.HasPrefix(host, "[") && strings.HasSuffix(host, "]") {
		return strings.TrimSuffix(strings.TrimPrefix(host, "["), "]")
	}
	return host
}

func httpBaseURL(host string, port int) string {
	return "http://" + net.JoinHostPort(host, strconv.Itoa(port))
}
