package filewatcher

import (
	"fmt"
	"math"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
)

type FileWatcher struct {
	watcher *fsnotify.Watcher
	mu      sync.Mutex
	handler func(event EventType, path string)
}

type EventType int

func (et EventType) String() string {
	switch et {
	case FileCreated:
		return "Created"
	case FileModified:
		return "Modified"
	case FileDeleted:
		return "Deleted"
	}
	return "Unknown"
}

const (
	FileCreated EventType = iota
	FileModified
	FileDeleted
)

func New() (*FileWatcher, error) {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}
	return &FileWatcher{
		watcher: watcher,
		mu:      sync.Mutex{},
	}, nil
}

func (fw *FileWatcher) Add(path string) error {
	fw.mu.Lock()
	defer fw.mu.Unlock()

	return filepath.WalkDir(path, func(p string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			if err := fw.watcher.Add(p); err != nil {
				return err
			}
		}
		return nil
	})
}

func (fw *FileWatcher) SetHandler(handler func(event EventType, path string)) {
	fw.mu.Lock()
	defer fw.mu.Unlock()
	fw.handler = handler
}

func (fw *FileWatcher) Close() error {
	fw.mu.Lock()
	defer fw.mu.Unlock()
	return fw.watcher.Close()
}

func (fw *FileWatcher) Start() {
	go fw.eventLoop()
}

func (fw *FileWatcher) eventLoop() {
	timerMu := sync.Mutex{}
	timers := make(map[string]*time.Timer)
	for {
		select {
		case event, ok := <-fw.watcher.Events:
			if !ok {
				return
			}
			// fmt.Println("event:", event)
			if event.Has(fsnotify.Write) || event.Has(fsnotify.Create) {
				// fmt.Println("modified file:", event.Name)
				timerMu.Lock()
				t, ok := timers[event.Name]
				timerMu.Unlock()

				if !ok {
					t = time.AfterFunc(math.MaxInt64, func() {
						fw.mu.Lock()
						tempHandler := fw.handler
						fw.mu.Unlock()
						if tempHandler != nil {
							info, err := os.Stat(event.Name)
							if err != nil {
								fmt.Println("error stating file:", err)
								return
							}
							if event.Has(fsnotify.Create) {
								if info.IsDir() {
									if err := fw.Add(event.Name); err != nil {
										fmt.Println("error adding directory:", err)
									}
								}
								if info.Mode().IsRegular() {
									go tempHandler(FileCreated, event.Name)
								}
							} else if event.Has(fsnotify.Write) {
								if info.Mode().IsRegular() {
									go tempHandler(FileModified, event.Name)
								}
							}
						}
					})
					t.Stop()

					timerMu.Lock()
					timers[event.Name] = t
					timerMu.Unlock()
				}
				t.Reset(100 * time.Millisecond)
			}
			if event.Has(fsnotify.Remove) || event.Has(fsnotify.Rename) {
				// fmt.Println("deleted or renamed file:", event.Name)
				timerMu.Lock()
				if t, ok := timers[event.Name]; ok {
					t.Stop()
					delete(timers, event.Name)
				}
			}
		case err, ok := <-fw.watcher.Errors:
			if !ok {
				return
			}
			fmt.Println("error:", err)
		}
	}
}
