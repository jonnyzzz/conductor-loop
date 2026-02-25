package runstate

import (
	"fmt"
	"io"
	"os"
	"sync"

	"github.com/fsnotify/fsnotify"
)

// DirWatcher watches a set of paths (files or directories) for filesystem
// changes using OS-native notifications (inotify/kqueue/FSEvents). New
// subdirectories created inside watched directories are automatically added
// to the watch set so the entire tree is covered as it grows.
//
// Rapid-fire events are coalesced into a single buffered notification so the
// consumer is never flooded: at most one pending signal is queued at a time.
type DirWatcher struct {
	fsw       *fsnotify.Watcher
	changes   chan struct{}
	closeOnce sync.Once
}

// NewDirWatcher creates a watcher for the given paths and starts the internal
// event loop. Both files and directories are accepted. Close must be called
// when the watcher is no longer needed to release OS resources.
func NewDirWatcher(paths ...string) (*DirWatcher, error) {
	fsw, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, fmt.Errorf("create fsnotify watcher: %w", err)
	}
	for _, p := range paths {
		if err := fsw.Add(p); err != nil {
			_ = fsw.Close()
			return nil, fmt.Errorf("watch %s: %w", p, err)
		}
	}
	dw := &DirWatcher{
		fsw:     fsw,
		changes: make(chan struct{}, 1),
	}
	go dw.loop()
	return dw, nil
}

func (dw *DirWatcher) loop() {
	defer close(dw.changes)
	for {
		select {
		case event, ok := <-dw.fsw.Events:
			if !ok {
				return
			}
			// Automatically watch newly-created subdirectories so the watch
			// set grows with the directory tree without requiring re-creation.
			// fsnotify is concurrency-safe; Add after Close returns an error
			// that we intentionally discard here.
			if event.Has(fsnotify.Create) {
				if fi, statErr := os.Stat(event.Name); statErr == nil && fi.IsDir() {
					_ = dw.fsw.Add(event.Name)
				}
			}
			// Non-blocking send: coalesce rapid events into a single signal.
			select {
			case dw.changes <- struct{}{}:
			default:
			}
		case _, ok := <-dw.fsw.Errors:
			if !ok {
				return
			}
			// Watcher errors are swallowed; the caller falls back to
			// ticker-based polling when the changes channel closes.
		}
	}
}

// Changes returns a channel that emits one value whenever any watched path
// changes. At most one notification is buffered; the consumer will not be
// called more than once per event batch.
//
// The channel is closed when the watcher is stopped via Close.
func (dw *DirWatcher) Changes() <-chan struct{} {
	return dw.changes
}

// Close stops the watcher and releases all OS resources.
// It is safe to call Close multiple times.
func (dw *DirWatcher) Close() error {
	var err error
	dw.closeOnce.Do(func() {
		// Closing fsw closes both fsw.Events and fsw.Errors, which causes
		// the loop goroutine to exit via the !ok branch.
		err = dw.fsw.Close()
	})
	return err
}

// FileReader holds a persistent open file handle and re-reads from the
// beginning on each call to ReadAll. This avoids repeated open/close cycles
// and keeps the file descriptor valid across monitoring ticks.
//
// If the file is replaced via an atomic rename (a common write pattern),
// ReadAll transparently re-opens the path so fresh content is returned.
type FileReader struct {
	path string
	f    *os.File
	mu   sync.Mutex
}

// NewFileReader opens the file at path and returns a FileReader ready for use.
func NewFileReader(path string) (*FileReader, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("open %s: %w", path, err)
	}
	return &FileReader{path: path, f: f}, nil
}

// ReadAll seeks to the start of the file and reads all its contents.
//
// If the file was replaced via atomic rename (a new inode is now at the path),
// the handle is transparently re-opened before reading so fresh content is
// returned. If the seek itself fails, a reopen is also attempted.
func (fr *FileReader) ReadAll() ([]byte, error) {
	fr.mu.Lock()
	defer fr.mu.Unlock()

	if _, err := fr.f.Seek(0, io.SeekStart); err != nil {
		// Seek failure — try re-opening regardless of the cause.
		if err2 := fr.reopen(); err2 != nil {
			return nil, fmt.Errorf("seek %s: %w", fr.path, err)
		}
	} else if replaced, _ := fr.isReplaced(); replaced {
		// The path now points to a different inode (atomic rename). Reopen
		// so subsequent reads reflect the new file content.
		if err := fr.reopen(); err != nil {
			return nil, err
		}
	}

	return io.ReadAll(fr.f)
}

// isReplaced reports whether the file at fr.path is a different inode from
// the currently-open file descriptor. Returns false on any stat error so
// the caller can proceed with the existing handle rather than failing.
// Caller must hold fr.mu.
func (fr *FileReader) isReplaced() (bool, error) {
	pathInfo, err := os.Stat(fr.path)
	if err != nil {
		return false, err
	}
	fdInfo, err := fr.f.Stat()
	if err != nil {
		return false, err
	}
	return !os.SameFile(pathInfo, fdInfo), nil
}

// reopen opens a fresh handle for fr.path, replacing the existing one.
// The new handle is opened before the old one is closed to ensure fr.f
// always points to a valid (or newly-opened) file.
// Caller must hold fr.mu.
func (fr *FileReader) reopen() error {
	f, err := os.Open(fr.path)
	if err != nil {
		return fmt.Errorf("reopen %s: %w", fr.path, err)
	}
	_ = fr.f.Close()
	fr.f = f
	return nil
}

// Close releases the underlying file handle.
func (fr *FileReader) Close() error {
	fr.mu.Lock()
	defer fr.mu.Unlock()
	return fr.f.Close()
}
