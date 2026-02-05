package api

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/pkg/errors"
)

// RunDiscovery polls a root directory for new runs.
type RunDiscovery struct {
	rootDir    string
	knownRuns  map[string]bool
	ticker     *time.Ticker
	newRunChan chan string
}

// NewRunDiscovery creates a RunDiscovery.
func NewRunDiscovery(rootDir string, interval time.Duration) (*RunDiscovery, error) {
	cleanDir := filepath.Clean(strings.TrimSpace(rootDir))
	if cleanDir == "." || cleanDir == "" {
		return nil, errors.New("root directory is empty")
	}
	if interval <= 0 {
		return nil, errors.New("discovery interval must be positive")
	}
	return &RunDiscovery{
		rootDir:    cleanDir,
		knownRuns:  make(map[string]bool),
		newRunChan: make(chan string, 32),
	}, nil
}

// NewRuns returns a channel emitting newly discovered run IDs.
func (d *RunDiscovery) NewRuns() <-chan string {
	if d == nil {
		return nil
	}
	return d.newRunChan
}

// Poll starts polling until context cancellation.
func (d *RunDiscovery) Poll(ctx context.Context, interval time.Duration) {
	if d == nil {
		return
	}
	if interval <= 0 {
		interval = time.Second
	}
	d.ticker = time.NewTicker(interval)
	defer d.ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-d.ticker.C:
			_ = d.scan()
		}
	}
}

func (d *RunDiscovery) scan() error {
	runs, err := listRunIDs(d.rootDir)
	if err != nil {
		return errors.Wrap(err, "list runs")
	}
	for _, name := range runs {
		if name == "" {
			continue
		}
		if d.knownRuns[name] {
			continue
		}
		d.knownRuns[name] = true
		select {
		case d.newRunChan <- name:
		default:
		}
	}
	return nil
}

func listRunIDs(rootDir string) ([]string, error) {
	cleanDir := filepath.Clean(strings.TrimSpace(rootDir))
	if cleanDir == "." || cleanDir == "" {
		return nil, errors.New("root directory is empty")
	}
	var runs []string
	walkErr := filepath.WalkDir(cleanDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		if d.Name() != "run-info.yaml" {
			return nil
		}
		runID := filepath.Base(filepath.Dir(path))
		if runID != "" {
			runs = append(runs, runID)
		}
		return nil
	})
	if walkErr != nil {
		if os.IsNotExist(walkErr) {
			return []string{}, nil
		}
		return nil, errors.Wrap(walkErr, "walk runs")
	}
	return runs, nil
}
