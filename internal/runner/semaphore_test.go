package runner

import (
	"context"
	"sync"
	"testing"
	"time"
)

// withTestSemaphore installs ch as the package-level semaphore for the duration
// of the test and restores the previous state via t.Cleanup.
func withTestSemaphore(t *testing.T, ch chan struct{}) {
	t.Helper()
	semMu.Lock()
	origChan := semChan
	origSet := semSet
	semChan = ch
	semSet = true
	semMu.Unlock()
	origGauge := queuedRunsGauge.Swap(0)
	t.Cleanup(func() {
		semMu.Lock()
		semChan = origChan
		semSet = origSet
		semMu.Unlock()
		queuedRunsGauge.Store(origGauge)
	})
}

// TestSemaphoreUnlimited verifies that nil semaphore (max=0) never blocks.
func TestSemaphoreUnlimited(t *testing.T) {
	withTestSemaphore(t, nil)

	var wg sync.WaitGroup
	errs := make(chan error, 2)
	for i := 0; i < 2; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			errs <- acquireSem(context.Background())
		}()
	}
	wg.Wait()
	close(errs)
	for err := range errs {
		if err != nil {
			t.Fatalf("acquireSem unlimited: %v", err)
		}
	}
	// No slots to release (unlimited), so no releaseSem calls needed.
}

// TestSemaphoreLimit1 verifies that max=1 serialises runs: the second caller
// blocks until the first releases the slot.
func TestSemaphoreLimit1(t *testing.T) {
	sem := make(chan struct{}, 1)
	withTestSemaphore(t, sem)

	// Acquire the sole slot.
	if err := acquireSem(context.Background()); err != nil {
		t.Fatalf("first acquireSem: %v", err)
	}

	// Second acquire must block.
	done := make(chan error, 1)
	go func() {
		done <- acquireSem(context.Background())
	}()

	select {
	case err := <-done:
		t.Fatalf("expected blocking but got result: %v", err)
	case <-time.After(60 * time.Millisecond):
		// Good: still blocking.
	}

	// Release the slot; second goroutine should now proceed.
	releaseSem()

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("second acquireSem after release: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("second acquireSem did not unblock after release")
	}

	// Release the slot held by the second goroutine.
	releaseSem()
}

// TestSemaphoreContextCancellationWhileWaiting verifies that a run waiting for
// a slot returns ctx.Err() immediately when the context is cancelled.
func TestSemaphoreContextCancellationWhileWaiting(t *testing.T) {
	sem := make(chan struct{}, 1)
	// Fill the slot so the next acquire will block.
	sem <- struct{}{}
	withTestSemaphore(t, sem)

	ctx, cancel := context.WithCancel(context.Background())

	done := make(chan error, 1)
	go func() {
		done <- acquireSem(ctx)
	}()

	// Give the goroutine time to enter the select.
	time.Sleep(20 * time.Millisecond)
	cancel()

	select {
	case err := <-done:
		if err == nil {
			t.Fatal("expected context error, got nil")
		}
	case <-time.After(2 * time.Second):
		t.Fatal("acquireSem did not return after context cancellation")
	}
}

// TestSemaphoreAlreadyCancelledContext verifies that an already-cancelled
// context causes acquireSem to return immediately without acquiring the slot.
func TestSemaphoreAlreadyCancelledContext(t *testing.T) {
	sem := make(chan struct{}, 1)
	// Fill the slot.
	sem <- struct{}{}
	withTestSemaphore(t, sem)

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // cancel immediately

	if err := acquireSem(ctx); err == nil {
		t.Fatal("expected context error for already-cancelled context")
	}
	// Slot must still be full (not consumed).
	if len(sem) != 1 {
		t.Fatalf("expected semaphore still full, len=%d", len(sem))
	}
}

// TestSemaphoreReleasedAfterError verifies that releaseSem frees the slot so
// a subsequent acquire can proceed.
func TestSemaphoreReleasedAfterError(t *testing.T) {
	sem := make(chan struct{}, 1)
	withTestSemaphore(t, sem)

	if err := acquireSem(context.Background()); err != nil {
		t.Fatalf("first acquireSem: %v", err)
	}
	// Simulate an error path by releasing immediately.
	releaseSem()

	// Slot must be free again.
	if len(sem) != 0 {
		t.Fatalf("expected empty semaphore after release, len=%d", len(sem))
	}

	// A second acquire should succeed.
	if err := acquireSem(context.Background()); err != nil {
		t.Fatalf("second acquireSem: %v", err)
	}
	releaseSem()
}

// TestQueuedRunCountWhileWaiting verifies that QueuedRunCount reflects the
// number of goroutines blocked in acquireSem.
func TestQueuedRunCountWhileWaiting(t *testing.T) {
	sem := make(chan struct{}, 1)
	// Fill the slot so any acquire will block.
	sem <- struct{}{}
	withTestSemaphore(t, sem)

	ready := make(chan struct{})
	done := make(chan error, 1)
	go func() {
		close(ready) // signal: about to call acquireSem
		done <- acquireSem(context.Background())
	}()

	<-ready
	// Wait until the queued count increments (acquireSem runs Add(1) before select).
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		if QueuedRunCount() == 1 {
			break
		}
		time.Sleep(time.Millisecond)
	}
	if count := QueuedRunCount(); count != 1 {
		t.Errorf("expected 1 queued run, got %d", count)
	}

	// Free the slot so the goroutine can proceed.
	<-sem

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("acquireSem: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("acquireSem did not complete after slot was freed")
	}

	// After acquisition the queued count must drop back to 0.
	if count := QueuedRunCount(); count != 0 {
		t.Errorf("expected 0 queued runs after acquire, got %d", count)
	}

	releaseSem()
}

// TestInitSemaphoreOnlyFirstCallTakesEffect verifies that initSemaphore is a
// one-shot operation: subsequent calls with a different capacity are no-ops.
func TestInitSemaphoreOnlyFirstCallTakesEffect(t *testing.T) {
	semMu.Lock()
	origChan := semChan
	origSet := semSet
	semChan = nil
	semSet = false
	semMu.Unlock()
	t.Cleanup(func() {
		semMu.Lock()
		semChan = origChan
		semSet = origSet
		semMu.Unlock()
	})

	initSemaphore(2)

	semMu.Lock()
	ch := semChan
	semMu.Unlock()
	if cap(ch) != 2 {
		t.Fatalf("expected capacity 2 after first initSemaphore, got %d", cap(ch))
	}

	// Second call with a different value must be a no-op.
	initSemaphore(5)
	semMu.Lock()
	ch = semChan
	semMu.Unlock()
	if cap(ch) != 2 {
		t.Fatalf("expected capacity still 2 after second initSemaphore, got %d", cap(ch))
	}
}

// TestInitSemaphoreZeroMeansUnlimited verifies that max_concurrent_runs=0
// leaves the semaphore nil (unlimited).
func TestInitSemaphoreZeroMeansUnlimited(t *testing.T) {
	semMu.Lock()
	origChan := semChan
	origSet := semSet
	semChan = nil
	semSet = false
	semMu.Unlock()
	t.Cleanup(func() {
		semMu.Lock()
		semChan = origChan
		semSet = origSet
		semMu.Unlock()
	})

	initSemaphore(0)

	semMu.Lock()
	ch := semChan
	semMu.Unlock()
	if ch != nil {
		t.Fatal("expected nil semaphore for unlimited (max=0)")
	}
}

// TestWaitingRunHook verifies that SetWaitingRunHook receives +1/-1 deltas
// while a run is waiting for a slot.
func TestWaitingRunHook(t *testing.T) {
	sem := make(chan struct{}, 1)
	// Fill the slot.
	sem <- struct{}{}
	withTestSemaphore(t, sem)

	var mu sync.Mutex
	var deltas []int64
	SetWaitingRunHook(func(delta int64) {
		mu.Lock()
		deltas = append(deltas, delta)
		mu.Unlock()
	})
	t.Cleanup(func() { SetWaitingRunHook(nil) })

	ready := make(chan struct{})
	done := make(chan error, 1)
	go func() {
		close(ready)
		done <- acquireSem(context.Background())
	}()

	<-ready
	// Wait until +1 is recorded.
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		mu.Lock()
		n := len(deltas)
		mu.Unlock()
		if n >= 1 {
			break
		}
		time.Sleep(time.Millisecond)
	}

	mu.Lock()
	if len(deltas) == 0 || deltas[0] != 1 {
		t.Errorf("expected first hook call with +1, got %v", deltas)
	}
	mu.Unlock()

	// Release the slot.
	<-sem

	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("acquireSem: %v", err)
		}
	case <-time.After(2 * time.Second):
		t.Fatal("acquireSem did not complete")
	}

	mu.Lock()
	if len(deltas) < 2 || deltas[1] != -1 {
		t.Errorf("expected second hook call with -1, got %v", deltas)
	}
	mu.Unlock()

	releaseSem()
}
