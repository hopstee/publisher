package watcher

import (
	"context"
	"log/slog"
	"os"
	"path/filepath"
	"publisher/internal/domain"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
)

const DEFAULT_WATCH_EVENTS_CHAN_BUFFER = 10

type Watcher struct {
	dirPath     string
	subDirs     []string
	watchEvents chan domain.WatcherEvent
	fsw         *fsnotify.Watcher
	lastEvent   map[string]time.Time
	mu          sync.Mutex
}

func NewWatcher(dirPath string) *Watcher {
	return &Watcher{
		dirPath:     dirPath,
		watchEvents: make(chan domain.WatcherEvent, DEFAULT_WATCH_EVENTS_CHAN_BUFFER),
		lastEvent:   make(map[string]time.Time),
	}
}

func (w *Watcher) Watch(ctx context.Context) (<-chan domain.WatcherEvent, error) {
	var err error

	w.subDirs, err = collectSubDirs(w.dirPath)
	if err != nil {
		return w.watchEvents, err
	}
	w.fsw, err = fsnotify.NewWatcher()
	if err != nil {
		_ = w.close()
		return w.watchEvents, err
	}
	go w.listenEvents(ctx)
	if err := w.addSubDirs(); err != nil {
		_ = w.close()
		return w.watchEvents, err
	}
	return w.watchEvents, nil
}

func (w *Watcher) listenEvents(ctx context.Context) {
	defer w.close()
	for {
		select {
		case event, ok := <-w.fsw.Events:
			if !ok {
				return
			}
			if skipEvent(event) {
				continue
			}
			if w.shouldDebounce(event) {
				continue
			}
			w.resolveEvent(event)
		case err, ok := <-w.fsw.Errors:
			if !ok {
				return
			}
			slog.Error("file system notification", slog.Any("error", err))
		case <-ctx.Done():
			if ctx.Err() != nil {
				slog.Error("context occured", slog.Any("error", ctx.Err()))
			}
			return
		}
	}
}

func skipEvent(event fsnotify.Event) bool {
	return event.Has(fsnotify.Chmod) || filepath.Ext(event.Name) != ".md" || strings.HasPrefix(event.Name, "~")
}

func (w *Watcher) shouldDebounce(event fsnotify.Event) bool {
	now := time.Now()
	if t, exists := w.lastEvent[event.Name]; exists {
		if now.Sub(t) < 2*time.Second {
			return true
		}
	}
	w.mu.Lock()
	defer w.mu.Unlock()
	w.lastEvent[event.Name] = now
	return false
}

func (w *Watcher) resolveEvent(event fsnotify.Event) {
	switch {
	case event.Has(fsnotify.Create):
		isDir, err := isDirectory(event.Name)
		if err != nil {
			slog.Error("failed to determine entity type", slog.Any("error", err))
			return
		}

		if isDir {
			slog.Debug("directody created, add it to watch dirs", slog.String("info", event.String()))
			if err := w.addSubDir(event.Name); err != nil {
				slog.Error("failed to add new dir to watcher list", slog.Any("error", err))
			}
		} else {
			w.watchEvents <- domain.WatcherEvent{
				Path: event.Name,
				Type: domain.UpsertWatcherEventType,
			}
		}
	case event.Has(fsnotify.Write):
		w.watchEvents <- domain.WatcherEvent{
			Path: event.Name,
			Type: domain.UpsertWatcherEventType,
		}
	case event.Has(fsnotify.Rename), event.Has(fsnotify.Remove):
		_, err := os.Stat(event.Name)
		if os.IsNotExist(err) {
			w.watchEvents <- domain.WatcherEvent{
				Path: event.Name,
				Type: domain.DeleteWatcherEventType,
			}
		}
	}
}

func (w *Watcher) addSubDirs() error {
	if err := w.fsw.Add(w.dirPath); err != nil {
		return err
	}
	for _, s := range w.subDirs {
		if err := w.fsw.Add(s); err != nil {
			return err
		}
	}
	return nil
}

func (w *Watcher) addSubDir(p string) error {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.subDirs = append(w.subDirs, p)
	if err := w.fsw.Add(p); err != nil {
		return err
	}
	return nil
}

func (w *Watcher) close() error {
	return w.fsw.Close()
}

func collectSubDirs(rootDir string) ([]string, error) {
	var subDirs = []string{}
	entries, err := os.ReadDir(rootDir)
	if err != nil {
		return subDirs, err
	}
	for _, e := range entries {
		if e.IsDir() {
			childDir := filepath.Join(rootDir, e.Name())
			childSubDirs, err := collectSubDirs(childDir)
			if err != nil {
				return subDirs, err
			}
			subDirs = append(subDirs, append([]string{childDir}, childSubDirs...)...)
		}
	}
	return subDirs, nil
}

func isDirectory(filePath string) (bool, error) {
	info, err := os.Stat(filePath)
	if err != nil {
		return false, err
	}
	return info.IsDir(), nil
}
