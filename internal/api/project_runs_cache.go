package api

import (
	"sync"
	"time"

	"github.com/jonnyzzz/conductor-loop/internal/storage"
)

const defaultProjectRunsFlatCacheTTL = 500 * time.Millisecond

type projectRunInfosCache struct {
	ttl time.Duration
	now func() time.Time

	mu      sync.Mutex
	entries map[string]*projectRunInfosCacheEntry
}

type projectRunInfosCacheEntry struct {
	expiresAt   time.Time
	runs        []*storage.RunInfo
	refreshing  bool
	refreshDone chan struct{}
}

func newProjectRunInfosCache(ttl time.Duration, now func() time.Time) *projectRunInfosCache {
	if ttl <= 0 {
		return nil
	}
	if now == nil {
		now = time.Now
	}
	return &projectRunInfosCache{
		ttl:     ttl,
		now:     now,
		entries: make(map[string]*projectRunInfosCacheEntry),
	}
}

func (c *projectRunInfosCache) get(
	projectID string,
	loader func() ([]*storage.RunInfo, error),
) ([]*storage.RunInfo, error) {
	if c == nil {
		return loader()
	}

	for {
		now := c.now()
		c.mu.Lock()
		entry, ok := c.entries[projectID]
		if ok && now.Before(entry.expiresAt) {
			cached := entry.runs
			c.mu.Unlock()
			return cached, nil
		}
		if ok && entry.refreshing {
			done := entry.refreshDone
			c.mu.Unlock()
			<-done
			continue
		}
		if !ok {
			entry = &projectRunInfosCacheEntry{}
			c.entries[projectID] = entry
		}
		entry.refreshing = true
		entry.refreshDone = make(chan struct{})
		done := entry.refreshDone
		c.mu.Unlock()

		runs, err := loader()

		c.mu.Lock()
		entry.refreshing = false
		close(done)
		entry.refreshDone = nil
		if err != nil {
			c.mu.Unlock()
			return nil, err
		}
		entry.runs = freezeRunInfoSlice(runs)
		entry.expiresAt = c.now().Add(c.ttl)
		cached := entry.runs
		c.mu.Unlock()
		return cached, nil
	}
}

func freezeRunInfoSlice(runs []*storage.RunInfo) []*storage.RunInfo {
	if len(runs) == 0 {
		if runs == nil {
			return nil
		}
		return []*storage.RunInfo{}
	}
	out := make([]*storage.RunInfo, len(runs))
	copy(out, runs)
	return out
}
