// Package metrics provides a lightweight Prometheus-compatible metrics registry.
package metrics

import (
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"time"
)

// Registry holds all metrics for the conductor server.
type Registry struct {
	startTime     time.Time
	activeRuns    atomic.Int64
	completedRuns atomic.Int64
	failedRuns    atomic.Int64
	busAppends    atomic.Int64

	mu           sync.Mutex
	apiRequests  map[string]*atomic.Int64 // key: "METHOD:status_code"
}

// New creates a new Registry with the current time as the start time.
func New() *Registry {
	return &Registry{
		startTime:   time.Now(),
		apiRequests: make(map[string]*atomic.Int64),
	}
}

// IncActiveRuns increments the active runs gauge.
func (r *Registry) IncActiveRuns() {
	if r == nil {
		return
	}
	r.activeRuns.Add(1)
}

// DecActiveRuns decrements the active runs gauge.
func (r *Registry) DecActiveRuns() {
	if r == nil {
		return
	}
	r.activeRuns.Add(-1)
}

// IncCompletedRuns increments the completed runs counter.
func (r *Registry) IncCompletedRuns() {
	if r == nil {
		return
	}
	r.completedRuns.Add(1)
}

// IncFailedRuns increments the failed runs counter.
func (r *Registry) IncFailedRuns() {
	if r == nil {
		return
	}
	r.failedRuns.Add(1)
}

// IncBusAppends increments the message bus append counter.
func (r *Registry) IncBusAppends() {
	if r == nil {
		return
	}
	r.busAppends.Add(1)
}

// RecordRequest records an API request by method and HTTP status code.
func (r *Registry) RecordRequest(method string, statusCode int) {
	if r == nil {
		return
	}
	key := fmt.Sprintf("%s:%d", strings.ToUpper(method), statusCode)
	r.mu.Lock()
	ctr, ok := r.apiRequests[key]
	if !ok {
		ctr = &atomic.Int64{}
		r.apiRequests[key] = ctr
	}
	r.mu.Unlock()
	ctr.Add(1)
}

// Render returns the metrics in Prometheus text format (version 0.0.4).
func (r *Registry) Render() string {
	if r == nil {
		return ""
	}
	uptime := time.Since(r.startTime).Seconds()

	var sb strings.Builder

	fmt.Fprintf(&sb, "# HELP conductor_uptime_seconds Server uptime in seconds\n")
	fmt.Fprintf(&sb, "# TYPE conductor_uptime_seconds gauge\n")
	fmt.Fprintf(&sb, "conductor_uptime_seconds %g\n", uptime)
	fmt.Fprintf(&sb, "\n")

	fmt.Fprintf(&sb, "# HELP conductor_active_runs_total Currently active (running) agent runs\n")
	fmt.Fprintf(&sb, "# TYPE conductor_active_runs_total gauge\n")
	fmt.Fprintf(&sb, "conductor_active_runs_total %d\n", r.activeRuns.Load())
	fmt.Fprintf(&sb, "\n")

	fmt.Fprintf(&sb, "# HELP conductor_completed_runs_total Total completed agent runs since startup\n")
	fmt.Fprintf(&sb, "# TYPE conductor_completed_runs_total counter\n")
	fmt.Fprintf(&sb, "conductor_completed_runs_total %d\n", r.completedRuns.Load())
	fmt.Fprintf(&sb, "\n")

	fmt.Fprintf(&sb, "# HELP conductor_failed_runs_total Total failed agent runs since startup\n")
	fmt.Fprintf(&sb, "# TYPE conductor_failed_runs_total counter\n")
	fmt.Fprintf(&sb, "conductor_failed_runs_total %d\n", r.failedRuns.Load())
	fmt.Fprintf(&sb, "\n")

	fmt.Fprintf(&sb, "# HELP conductor_messagebus_appends_total Total message bus append operations\n")
	fmt.Fprintf(&sb, "# TYPE conductor_messagebus_appends_total counter\n")
	fmt.Fprintf(&sb, "conductor_messagebus_appends_total %d\n", r.busAppends.Load())
	fmt.Fprintf(&sb, "\n")

	fmt.Fprintf(&sb, "# HELP conductor_api_requests_total Total API requests by method and status\n")
	fmt.Fprintf(&sb, "# TYPE conductor_api_requests_total counter\n")

	r.mu.Lock()
	keys := make([]string, 0, len(r.apiRequests))
	for k := range r.apiRequests {
		keys = append(keys, k)
	}
	r.mu.Unlock()

	// Sort for deterministic output.
	sortStrings(keys)

	for _, key := range keys {
		r.mu.Lock()
		ctr := r.apiRequests[key]
		r.mu.Unlock()
		parts := strings.SplitN(key, ":", 2)
		if len(parts) != 2 {
			continue
		}
		method, status := parts[0], parts[1]
		fmt.Fprintf(&sb, "conductor_api_requests_total{method=%q,status=%q} %d\n", method, status, ctr.Load())
	}

	return sb.String()
}

// sortStrings sorts a slice of strings in-place (insertion sort for simplicity, no extra import).
func sortStrings(ss []string) {
	for i := 1; i < len(ss); i++ {
		for j := i; j > 0 && ss[j] < ss[j-1]; j-- {
			ss[j], ss[j-1] = ss[j-1], ss[j]
		}
	}
}
