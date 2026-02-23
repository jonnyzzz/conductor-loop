package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/jonnyzzz/conductor-loop/internal/storage"
)

func TestProjectRunInfosCacheHitAndExpiry(t *testing.T) {
	now := time.Date(2026, time.February, 23, 2, 0, 0, 0, time.UTC)
	cache := newProjectRunInfosCache(200*time.Millisecond, func() time.Time {
		return now
	})

	loadCount := 0
	loader := func() ([]*storage.RunInfo, error) {
		loadCount++
		return []*storage.RunInfo{{RunID: fmt.Sprintf("run-%d", loadCount), ProjectID: "project"}}, nil
	}

	first, err := cache.get("project", loader)
	if err != nil {
		t.Fatalf("first load: %v", err)
	}
	if loadCount != 1 {
		t.Fatalf("expected first load count=1, got %d", loadCount)
	}
	if len(first) != 1 || first[0].RunID != "run-1" {
		t.Fatalf("unexpected first result: %+v", first)
	}

	second, err := cache.get("project", loader)
	if err != nil {
		t.Fatalf("second load: %v", err)
	}
	if loadCount != 1 {
		t.Fatalf("cache hit should not reload, got load count=%d", loadCount)
	}
	if len(second) != 1 || second[0].RunID != "run-1" {
		t.Fatalf("unexpected second result: %+v", second)
	}

	now = now.Add(201 * time.Millisecond)
	third, err := cache.get("project", loader)
	if err != nil {
		t.Fatalf("third load after ttl: %v", err)
	}
	if loadCount != 2 {
		t.Fatalf("expected refresh after ttl, got load count=%d", loadCount)
	}
	if len(third) != 1 || third[0].RunID != "run-2" {
		t.Fatalf("unexpected third result: %+v", third)
	}
}

func TestProjectRunInfosCacheConcurrentMissSharesRefresh(t *testing.T) {
	cache := newProjectRunInfosCache(time.Second, time.Now)

	const goroutineCount = 24
	var loadCount atomic.Int32
	loader := func() ([]*storage.RunInfo, error) {
		loadCount.Add(1)
		time.Sleep(20 * time.Millisecond)
		return []*storage.RunInfo{{RunID: "run-1", ProjectID: "project"}}, nil
	}

	errCh := make(chan error, goroutineCount)
	var wg sync.WaitGroup
	wg.Add(goroutineCount)
	for i := 0; i < goroutineCount; i++ {
		go func() {
			defer wg.Done()
			runs, err := cache.get("project", loader)
			if err != nil {
				errCh <- err
				return
			}
			if len(runs) != 1 || runs[0].RunID != "run-1" {
				errCh <- fmt.Errorf("unexpected runs: %+v", runs)
			}
		}()
	}
	wg.Wait()
	close(errCh)

	for err := range errCh {
		if err != nil {
			t.Fatalf("concurrent get failed: %v", err)
		}
	}

	if got := loadCount.Load(); got != 1 {
		t.Fatalf("expected single refresh for concurrent miss, got %d", got)
	}
}

func TestHandleProjectRunsFlatUsesCacheUntilTTLExpiry(t *testing.T) {
	root := t.TempDir()
	now := time.Date(2026, time.February, 23, 2, 30, 0, 0, time.UTC)
	nowFn := func() time.Time {
		return now
	}

	server, err := NewServer(Options{
		RootDir:                 root,
		DisableTaskStart:        true,
		Now:                     nowFn,
		ProjectRunsFlatCacheTTL: time.Second,
	})
	if err != nil {
		t.Fatalf("NewServer: %v", err)
	}

	makeProjectRun(t, root, "project", "task", "run-1", storage.StatusCompleted, "first\n")
	first := fetchProjectRunsFlat(t, server, "/api/projects/project/runs/flat")
	if len(first) != 1 {
		t.Fatalf("expected 1 run on first fetch, got %d", len(first))
	}

	makeProjectRun(t, root, "project", "task", "run-2", storage.StatusCompleted, "second\n")
	now = now.Add(500 * time.Millisecond)
	second := fetchProjectRunsFlat(t, server, "/api/projects/project/runs/flat")
	if len(second) != 1 {
		t.Fatalf("expected cached 1 run before ttl expiry, got %d", len(second))
	}

	now = now.Add(600 * time.Millisecond)
	third := fetchProjectRunsFlat(t, server, "/api/projects/project/runs/flat")
	if len(third) != 2 {
		t.Fatalf("expected refreshed 2 runs after ttl expiry, got %d", len(third))
	}
}

func fetchProjectRunsFlat(t *testing.T, server *Server, path string) []flatRunItem {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, path, nil)
	rec := httptest.NewRecorder()
	server.Handler().ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("GET %s status=%d body=%s", path, rec.Code, rec.Body.String())
	}
	var resp flatRunsResponse
	if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
		t.Fatalf("decode %s response: %v", path, err)
	}
	return resp.Runs
}
