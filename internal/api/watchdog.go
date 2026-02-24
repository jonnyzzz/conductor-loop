package api

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"time"
)

// probeClient is the HTTP client used by the watchdog. It has a short timeout
// so that a hung listener does not block the probe indefinitely.
var probeClient = &http.Client{Timeout: 5 * time.Second}

// Watchdog periodically probes the server's /healthz endpoint and exits if it
// fails too many consecutive times. It is designed to be run as a goroutine
// alongside the HTTP server started by ListenAndServe.
type Watchdog struct {
	// Server is the server instance whose ActualPort is polled after startup.
	Server *Server
	// Host is the loopback address used to reach the server (e.g. "127.0.0.1").
	Host string
	// Interval is the time between health probe attempts.
	Interval time.Duration
	// MaxFailures is the number of consecutive failures after which the process exits.
	MaxFailures int
	// Logger receives WARN/ERROR messages from the watchdog.
	Logger *log.Logger
	// ExitFunc is called when the failure threshold is reached (default: os.Exit(1)).
	// Override in tests to avoid terminating the test process.
	ExitFunc func(code int)
	// ProbeFunc overrides the default HTTP-based health probe. Used in tests.
	// It receives the target URL and returns nil on success, an error on failure.
	ProbeFunc func(url string) error
}

// Run starts the watchdog loop. It blocks until the process is asked to exit.
func (w *Watchdog) Run() {
	exitFn := w.ExitFunc
	if exitFn == nil {
		exitFn = os.Exit
	}
	logger := w.Logger
	if logger == nil {
		logger = log.New(os.Stderr, "watchdog ", log.LstdFlags)
	}
	interval := w.Interval
	if interval <= 0 {
		interval = 30 * time.Second
	}
	maxFailures := w.MaxFailures
	if maxFailures <= 0 {
		maxFailures = 3
	}
	probe := w.ProbeFunc
	if probe == nil {
		probe = defaultProbe
	}

	failures := 0
	for {
		time.Sleep(interval)

		port := w.Server.ActualPort()
		if port == 0 {
			// Server not yet listening; skip this cycle.
			continue
		}

		url := fmt.Sprintf("http://%s:%d/healthz", w.Host, port)
		if err := probe(url); err != nil {
			failures++
			logger.Printf("WARN: server health probe failed %d times: %v", failures, err)
			if failures >= maxFailures {
				logger.Printf("ERROR: server health probe failed %d consecutive times (max %d); exiting", failures, maxFailures)
				exitFn(1)
				return
			}
		} else {
			failures = 0
		}
	}
}

// defaultProbe performs an HTTP GET to the given URL and returns an error if the
// response status is not 200 OK or if the request fails.
func defaultProbe(url string) error {
	resp, err := probeClient.Get(url) //nolint:noctx
	if err != nil {
		return err
	}
	_ = resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("unexpected status %d", resp.StatusCode)
	}
	return nil
}
