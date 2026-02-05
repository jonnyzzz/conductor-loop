package api

import (
	"bufio"
	"context"
	"io"
	"os"
	"strings"
	"time"

	"github.com/pkg/errors"
)

// LogLine represents a tailed log line.
type LogLine struct {
	RunID     string
	Stream    string
	Line      string
	Timestamp time.Time
}

// Tailer polls a log file and emits new lines.
type Tailer struct {
	filePath     string
	runID        string
	stream       string
	pollInterval time.Duration
	offset       int64
	pending      string
	events       chan<- LogLine
	done         chan struct{}
	ticker       *time.Ticker
}

// NewTailer creates a Tailer starting after the provided line number.
func NewTailer(filePath, runID, stream string, pollInterval time.Duration, startLine int64, events chan<- LogLine) (*Tailer, error) {
	cleanPath := strings.TrimSpace(filePath)
	if cleanPath == "" {
		return nil, errors.New("file path is empty")
	}
	if pollInterval <= 0 {
		return nil, errors.New("poll interval must be positive")
	}
	offset, err := offsetForLine(cleanPath, startLine)
	if err != nil {
		return nil, errors.Wrap(err, "compute start offset")
	}
	return &Tailer{
		filePath:     cleanPath,
		runID:        strings.TrimSpace(runID),
		stream:       strings.TrimSpace(stream),
		pollInterval: pollInterval,
		offset:       offset,
		events:       events,
		done:         make(chan struct{}),
	}, nil
}

// Start begins polling in a goroutine.
func (t *Tailer) Start(ctx context.Context) {
	if t == nil {
		return
	}
	t.ticker = time.NewTicker(t.pollInterval)
	go func() {
		defer t.ticker.Stop()
		for {
			select {
			case <-ctx.Done():
				return
			case <-t.done:
				return
			case <-t.ticker.C:
				_ = t.poll()
			}
		}
	}()
}

// Stop stops polling.
func (t *Tailer) Stop() {
	if t == nil {
		return
	}
	select {
	case <-t.done:
		return
	default:
		close(t.done)
	}
}

func (t *Tailer) poll() error {
	file, err := os.Open(t.filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return errors.Wrap(err, "open log file")
	}
	defer file.Close()

	info, err := file.Stat()
	if err != nil {
		return errors.Wrap(err, "stat log file")
	}
	if info.Size() < t.offset {
		t.offset = 0
		t.pending = ""
	}
	if _, err := file.Seek(t.offset, io.SeekStart); err != nil {
		return errors.Wrap(err, "seek log file")
	}

	reader := bufio.NewReader(file)
	for {
		part, err := reader.ReadString('\n')
		if part != "" {
			t.offset += int64(len(part))
			if strings.HasSuffix(part, "\n") {
				line := t.pending + strings.TrimSuffix(part, "\n")
				t.pending = ""
				line = strings.TrimSuffix(line, "\r")
				t.emit(line)
			} else {
				t.pending += part
			}
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			return errors.Wrap(err, "read log file")
		}
	}
	return nil
}

func (t *Tailer) emit(line string) {
	if t.events == nil {
		return
	}
	event := LogLine{
		RunID:     t.runID,
		Stream:    t.stream,
		Line:      line,
		Timestamp: time.Now().UTC(),
	}
	select {
	case t.events <- event:
	default:
	}
}

func offsetForLine(path string, line int64) (int64, error) {
	if line < 0 {
		return offsetForEnd(path)
	}
	if line == 0 {
		return 0, nil
	}
	file, err := os.Open(path)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, nil
		}
		return 0, errors.Wrap(err, "open log file")
	}
	defer file.Close()

	reader := bufio.NewReader(file)
	var (
		offset int64
		count  int64
	)
	for count < line {
		part, err := reader.ReadString('\n')
		if part != "" {
			offset += int64(len(part))
			count++
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			return 0, errors.Wrap(err, "read log file")
		}
	}
	return offset, nil
}

func offsetForEnd(path string) (int64, error) {
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return 0, nil
		}
		return 0, errors.Wrap(err, "stat log file")
	}
	return info.Size(), nil
}
