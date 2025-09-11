package filewatcher

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/fsnotify/fsnotify"
)

// EventType represents the type of file system event
type EventType string

const (
	Add    EventType = "add"
	Change EventType = "change"
	Remove EventType = "remove"
	Error  EventType = "error"
)

// Event represents a file system event
type Event struct {
	Type EventType
	Path string
	Info os.FileInfo
	Err  error
}

// EventHandler is a function that handles file system events
type EventHandler func(event Event)

// Options for configuring the file watcher
type Options struct {
	Recursive     bool
	IgnoreInitial bool
	Ignored       []string // patterns to ignore
	Whitelist     []string // patterns to whitelist (if provided, only these will be processed)
}

// Watcher represents a file system watcher
type Watcher struct {
	fsWatcher    *fsnotify.Watcher
	options      Options
	handlers     map[EventType][]EventHandler
	watchedPaths map[string]bool
	mutex        sync.RWMutex
	done         chan bool
	closed       bool
}

// NewWatcher creates a new file watcher with the given options
func NewWatcher(options Options) (*Watcher, error) {
	fsWatcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}

	watcher := &Watcher{
		fsWatcher:    fsWatcher,
		options:      options,
		handlers:     make(map[EventType][]EventHandler),
		watchedPaths: make(map[string]bool),
		done:         make(chan bool),
		closed:       false,
	}

	go watcher.eventLoop()

	return watcher, nil
}

// On registers an event handler for the specified event type
func (w *Watcher) On(eventType EventType, handler EventHandler) {
	w.mutex.Lock()
	defer w.mutex.Unlock()

	if w.handlers[eventType] == nil {
		w.handlers[eventType] = make([]EventHandler, 0)
	}
	w.handlers[eventType] = append(w.handlers[eventType], handler)
}

// Add adds files or directories to watch
func (w *Watcher) Add(paths ...string) error {
	w.mutex.Lock()
	defer w.mutex.Unlock()

	if w.closed {
		return fmt.Errorf("watcher is closed")
	}

	for _, path := range paths {
		if err := w.addPath(path); err != nil {
			return err
		}
	}

	return nil
}

// addPath adds a single path to watch
func (w *Watcher) addPath(path string) error {
	// Clean and resolve the path
	cleanPath, err := filepath.Abs(path)
	if err != nil {
		return err
	}

	// Check if path exists
	info, err := os.Stat(cleanPath)
	if err != nil {
		return err
	}

	// Add the path to fsnotify watcher
	if err := w.fsWatcher.Add(cleanPath); err != nil {
		return err
	}

	w.watchedPaths[cleanPath] = true

	// If it's a directory and recursive is enabled, add subdirectories
	if info.IsDir() && w.options.Recursive {
		if err := w.addRecursive(cleanPath); err != nil {
			return err
		}
	}

	// Emit initial events if ignoreInitial is false
	if !w.options.IgnoreInitial {
		w.emitInitialEvents(cleanPath, info)
	}

	return nil
}

// addRecursive recursively adds all subdirectories
func (w *Watcher) addRecursive(dir string) error {
	return filepath.Walk(dir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() && path != dir {
			if w.shouldIgnore(path) {
				return filepath.SkipDir
			}

			if err := w.fsWatcher.Add(path); err != nil {
				return err
			}
			w.watchedPaths[path] = true
		}

		return nil
	})
}

// emitInitialEvents emits initial add events for existing files
func (w *Watcher) emitInitialEvents(path string, info os.FileInfo) {
	if info.IsDir() {
		filepath.Walk(path, func(walkPath string, walkInfo os.FileInfo, err error) error {
			if err != nil {
				w.emitEvent(Event{Type: Error, Path: walkPath, Err: err})
				return nil
			}

			if !w.shouldIgnore(walkPath) {
				w.emitEvent(Event{Type: Add, Path: walkPath, Info: walkInfo})
			}

			return nil
		})
	} else {
		if !w.shouldIgnore(path) {
			w.emitEvent(Event{Type: Add, Path: path, Info: info})
		}
	}
}

// shouldIgnore checks if a path should be ignored based on patterns
func (w *Watcher) shouldIgnore(path string) bool {
	// If whitelist is provided, only allow files that match whitelist patterns
	if len(w.options.Whitelist) > 0 {
		whitelisted := false
		for _, pattern := range w.options.Whitelist {
			if matched, _ := filepath.Match(pattern, filepath.Base(path)); matched {
				whitelisted = true
				break
			}
			// Also check if the pattern matches anywhere in the path
			if strings.Contains(path, pattern) {
				whitelisted = true
				break
			}
		}
		if !whitelisted {
			return true
		}
	}

	// Check ignored patterns
	for _, pattern := range w.options.Ignored {
		if matched, _ := filepath.Match(pattern, filepath.Base(path)); matched {
			return true
		}
		// Also check if the pattern matches anywhere in the path
		if strings.Contains(path, pattern) {
			return true
		}
	}
	return false
}

// eventLoop processes file system events
func (w *Watcher) eventLoop() {
	for {
		select {
		case event, ok := <-w.fsWatcher.Events:
			if !ok {
				return
			}
			w.handleFsEvent(event)

		case err, ok := <-w.fsWatcher.Errors:
			if !ok {
				return
			}
			w.emitEvent(Event{Type: Error, Path: "", Err: err})

		case <-w.done:
			return
		}
	}
}

// handleFsEvent converts fsnotify events to our event format
func (w *Watcher) handleFsEvent(event fsnotify.Event) {
	if w.shouldIgnore(event.Name) {
		return
	}

	var eventType EventType
	var info os.FileInfo

	switch {
	case event.Op&fsnotify.Create == fsnotify.Create:
		eventType = Add
		// If recursive and it's a new directory, start watching it
		if stat, err := os.Stat(event.Name); err == nil {
			info = stat
			if stat.IsDir() && w.options.Recursive {
				w.mutex.Lock()
				w.addRecursive(event.Name)
				w.mutex.Unlock()
			}
		}

	case event.Op&fsnotify.Write == fsnotify.Write:
		eventType = Change
		if stat, err := os.Stat(event.Name); err == nil {
			info = stat
		}

	case event.Op&fsnotify.Remove == fsnotify.Remove:
		eventType = Remove
		// Remove from watched paths if it was a directory
		w.mutex.Lock()
		delete(w.watchedPaths, event.Name)
		w.mutex.Unlock()

	case event.Op&fsnotify.Rename == fsnotify.Rename:
		eventType = Remove
		w.mutex.Lock()
		delete(w.watchedPaths, event.Name)
		w.mutex.Unlock()

	default:
		return
	}

	w.emitEvent(Event{
		Type: eventType,
		Path: event.Name,
		Info: info,
	})
}

// emitEvent sends an event to all registered handlers
func (w *Watcher) emitEvent(event Event) {
	w.mutex.RLock()
	handlers := w.handlers[event.Type]
	w.mutex.RUnlock()

	for _, handler := range handlers {
		go handler(event)
	}
}

// Close stops the watcher and releases resources
func (w *Watcher) Close() error {
	w.mutex.Lock()
	defer w.mutex.Unlock()

	if w.closed {
		return nil
	}

	w.closed = true
	close(w.done)
	return w.fsWatcher.Close()
}

// GetWatchedPaths returns a slice of currently watched paths
func (w *Watcher) GetWatchedPaths() []string {
	w.mutex.RLock()
	defer w.mutex.RUnlock()

	paths := make([]string, 0, len(w.watchedPaths))
	for path := range w.watchedPaths {
		paths = append(paths, path)
	}
	return paths
}
