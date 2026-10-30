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
	dirPath      string
	watchEvents  chan domain.WatcherEvent
	fsw          *fsnotify.Watcher
	lastEvent    map[string]*time.Timer
	debounceTime time.Duration

	stopped bool
	mu      sync.Mutex
	wg      sync.WaitGroup

	ctx    context.Context
	cancel context.CancelFunc
}

func NewWatcher(dirPath string) *Watcher {
	return &Watcher{
		dirPath:      dirPath,
		watchEvents:  make(chan domain.WatcherEvent, DEFAULT_WATCH_EVENTS_CHAN_BUFFER),
		lastEvent:    make(map[string]*time.Timer),
		debounceTime: 2 * time.Second,
	}
}

func (w *Watcher) Watch(ctx context.Context) (<-chan domain.WatcherEvent, error) {
	w.ctx, w.cancel = context.WithCancel(ctx)

	subDirs, err := collectSubDirs(w.dirPath)
	if err != nil {
		return nil, err
	}
	w.fsw, err = fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}
	if err := w.addSubDirs(w.dirPath, subDirs); err != nil {
		_ = w.fsw.Close()
		return nil, err
	}

	go w.listenEvents(w.ctx)

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
			w.resolveEvent(ctx, event)
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

func isMDFile(eventName string) bool {
	return filepath.Ext(eventName) == ".md"
}

func skipEvent(event fsnotify.Event) bool {
	return event.Has(fsnotify.Chmod) || strings.HasPrefix(filepath.Base(event.Name), "~")
}

func (w *Watcher) resolveEvent(ctx context.Context, event fsnotify.Event) {
	isDir, err := isDirectory(event.Name)
	if err != nil {
		slog.Error("failed to determine entity type", slog.Any("error", err))
		return
	}
	if isDir {
		w.resolveDirEvent(event)
		return
	}
	if isMDFile(event.Name) {
		w.debounce(event.Name, func() {
			w.resolveFileEvent(ctx, event)
		})
	}
}

func (w *Watcher) resolveDirEvent(event fsnotify.Event) {
	if event.Has(fsnotify.Create) {
		if err := w.addSubDir(event.Name); err != nil {
			slog.Error("failed to add new dir to watcher list", slog.Any("error", err))
		}
	}
}

func (w *Watcher) resolveFileEvent(ctx context.Context, event fsnotify.Event) {
	switch {
	case event.Has(fsnotify.Create), event.Has(fsnotify.Write):
		w.writeToChannel(ctx, event.Name, domain.UpsertDocActionType)
	case event.Has(fsnotify.Rename), event.Has(fsnotify.Remove):
		_, err := os.Stat(event.Name)
		if os.IsNotExist(err) {
			w.writeToChannel(ctx, event.Name, domain.DeleteDocActionType)
		}
	}
}

func (w *Watcher) writeToChannel(ctx context.Context, path string, entityType domain.DocActionType) {
	select {
	case <-ctx.Done():
		return
	case w.watchEvents <- domain.WatcherEvent{
		Path: path,
		Type: entityType,
	}:
	}
}

func (w *Watcher) addSubDirs(rootDir string, subDirs []string) error {
	if err := w.fsw.Add(rootDir); err != nil {
		return err
	}
	for _, s := range subDirs {
		if err := w.fsw.Add(s); err != nil {
			return err
		}
	}
	return nil
}

func (w *Watcher) addSubDir(p string) error {
	children, err := collectSubDirs(p)
	if err != nil {
		return err
	}
	return w.addSubDirs(p, children)
}

func (w *Watcher) debounce(eventName string, callback func()) {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.stopped {
		return
	}

	if t, ok := w.lastEvent[eventName]; ok {
		if t.Stop() {
			w.wg.Done()
		}
	}

	var timer *time.Timer

	w.wg.Add(1)
	timer = time.AfterFunc(w.debounceTime, func() {
		defer w.wg.Done()
		w.mu.Lock()

		isCurrent := w.lastEvent[eventName] == timer
		if isCurrent {
			delete(w.lastEvent, eventName)
		}

		stopped := w.stopped
		w.mu.Unlock()

		if isCurrent && !stopped {
			callback()
		}
	})

	w.lastEvent[eventName] = timer
}

func (w *Watcher) close() {
	w.cancel()

	w.mu.Lock()
	w.stopped = true
	for name, timer := range w.lastEvent {
		if timer.Stop() {
			w.wg.Done()
		}
		delete(w.lastEvent, name)
	}
	w.mu.Unlock()

	w.wg.Wait()

	close(w.watchEvents)
	if err := w.fsw.Close(); err != nil {
		slog.Error("failed to close file system watcher", slog.Any("error", err))
	}
}

func collectSubDirs(rootDir string) ([]string, error) {
	// TODO: avoid recursion, collect found directories to array and check them for child dirs
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
