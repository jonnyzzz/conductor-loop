// Package api provides the REST API server for Conductor Loop.
package api

import (
	"context"
	"log"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/jonnyzzz/conductor-loop/internal/config"
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
		cfg.Port = 8080
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

// ListenAndServe starts the HTTP server.
func (s *Server) ListenAndServe() error {
	if s == nil {
		return errors.New("server is nil")
	}
	s.mu.Lock()
	if s.server == nil {
		addr := net.JoinHostPort(s.apiConfig.Host, intToString(s.apiConfig.Port))
		s.server = &http.Server{
			Addr:    addr,
			Handler: s.handler,
		}
	}
	srv := s.server
	s.mu.Unlock()
	return srv.ListenAndServe()
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
