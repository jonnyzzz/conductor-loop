package runner

import (
	"context"
	"sync"
	"sync/atomic"
)

var (
	semMu   sync.Mutex
	semChan chan struct{}
	semSet  bool // true once initSemaphore has been called

	queuedRunsGauge atomic.Int64

	hookMu         sync.Mutex
	waitingRunHook func(int64)
)

// initSemaphore configures the package-level concurrency semaphore.
// Only the first call takes effect; subsequent calls are no-ops.
// n <= 0 means unlimited (no semaphore).
func initSemaphore(n int) {
	semMu.Lock()
	defer semMu.Unlock()
	if semSet {
		return
	}
	semSet = true
	if n > 0 {
		semChan = make(chan struct{}, n)
	}
}

// SetWaitingRunHook registers a function called with +1 when a run starts
// waiting for a concurrency slot and -1 when it acquires or gives up.
// Used by the API server to propagate queued-run counts to the metrics registry.
func SetWaitingRunHook(fn func(int64)) {
	hookMu.Lock()
	defer hookMu.Unlock()
	waitingRunHook = fn
}

// QueuedRunCount returns the number of runs currently waiting for a semaphore slot.
func QueuedRunCount() int64 {
	return queuedRunsGauge.Load()
}

// acquireSem blocks until a run slot is available or ctx is cancelled.
// Returns nil if the semaphore is unlimited (max_concurrent_runs == 0).
func acquireSem(ctx context.Context) error {
	semMu.Lock()
	ch := semChan
	semMu.Unlock()
	if ch == nil {
		return nil // unlimited
	}
	queuedRunsGauge.Add(1)
	notifySemHook(1)
	defer func() {
		queuedRunsGauge.Add(-1)
		notifySemHook(-1)
	}()
	select {
	case ch <- struct{}{}:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

// releaseSem releases a previously acquired run slot.
func releaseSem() {
	semMu.Lock()
	ch := semChan
	semMu.Unlock()
	if ch != nil {
		<-ch
	}
}

func notifySemHook(delta int64) {
	hookMu.Lock()
	fn := waitingRunHook
	hookMu.Unlock()
	if fn != nil {
		fn(delta)
	}
}
