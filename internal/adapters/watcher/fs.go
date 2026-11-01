package watcher

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"publisher/internal/domain"
	"strings"
	"sync"
	"time"

	"github.com/fsnotify/fsnotify"
)

const defaultWatcherEventsChanBuffer = 10

type FS struct {
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

func NewFSWatcher(dirPath string) *FS {
	return &FS{
		dirPath:      dirPath,
		watchEvents:  make(chan domain.WatcherEvent, defaultWatcherEventsChanBuffer),
		lastEvent:    make(map[string]*time.Timer),
		debounceTime: 2 * time.Second,
	}
}

func (fs *FS) Watch(ctx context.Context) (<-chan domain.WatcherEvent, error) {
	fs.ctx, fs.cancel = context.WithCancel(ctx)

	subDirs, err := collectSubDirs(fs.dirPath)
	if err != nil {
		return nil, err
	}
	fs.fsw, err = fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}
	if err := fs.addSubDirs(fs.dirPath, subDirs); err != nil {
		_ = fs.fsw.Close()
		return nil, err
	}

	go fs.listenEvents(fs.ctx)

	return fs.watchEvents, nil
}

func (fs *FS) listenEvents(ctx context.Context) {
	defer fs.close()
	for {
		select {
		case event, ok := <-fs.fsw.Events:
			if !ok {
				return
			}
			if skipEvent(event) {
				continue
			}
			fs.resolveEvent(ctx, event)
		case err, ok := <-fs.fsw.Errors:
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

func (fs *FS) resolveEvent(ctx context.Context, event fsnotify.Event) {
	// TODO: if we delete directory or file function isDerictory will throw error.
	// by the way if we delete directory we need to handle this event too
	// because direcctory can contain child entities, and if directory deleted
	// need to delete all the children
	isDir, err := isDirectory(event.Name)
	if err != nil {
		slog.Error("failed to determine entity type", slog.Any("error", err))
		return
	}
	if isDir {
		fs.resolveDirEvent(event)
		return
	}
	if isMDFile(event.Name) {
		fs.debounce(event.Name, func() {
			fs.resolveFileEvent(ctx, event)
		})
	}
}

func (fs *FS) resolveDirEvent(event fsnotify.Event) {
	if event.Has(fsnotify.Create) {
		if err := fs.addSubDir(event.Name); err != nil {
			slog.Error("failed to add new dir to watcher list", slog.Any("error", err))
		}
	}
}

func (fs *FS) resolveFileEvent(ctx context.Context, event fsnotify.Event) {
	switch {
	case event.Has(fsnotify.Create), event.Has(fsnotify.Write):
		fs.writeToChannel(ctx, event.Name, domain.UpsertDocActionType)
	case event.Has(fsnotify.Rename), event.Has(fsnotify.Remove):
		_, err := os.Stat(event.Name)
		if os.IsNotExist(err) {
			fs.writeToChannel(ctx, event.Name, domain.DeleteDocActionType)
		}
	}
}

func (fs *FS) writeToChannel(ctx context.Context, path string, entityType domain.DocActionType) {
	select {
	case <-ctx.Done():
		return
	case fs.watchEvents <- domain.WatcherEvent{
		Path: path,
		Type: entityType,
	}:
	}
}

func (fs *FS) addSubDirs(rootDir string, subDirs []string) error {
	if err := fs.fsw.Add(rootDir); err != nil {
		return fmt.Errorf("failed to add root directory to fs watcher: %w", err)
	}
	for _, s := range subDirs {
		if err := fs.fsw.Add(s); err != nil {
			return fmt.Errorf("failed to add sub directory %s to fs watcher: %w", s, err)
		}
	}
	return nil
}

func (fs *FS) addSubDir(p string) error {
	children, err := collectSubDirs(p)
	if err != nil {
		return fmt.Errorf("failed to collect sub directories: %w", err)
	}
	return fs.addSubDirs(p, children)
}

func (fs *FS) debounce(eventName string, callback func()) {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	if fs.stopped {
		return
	}

	if t, ok := fs.lastEvent[eventName]; ok {
		if t.Stop() {
			fs.wg.Done()
		}
	}

	var timer *time.Timer

	fs.wg.Add(1)
	timer = time.AfterFunc(fs.debounceTime, func() {
		defer fs.wg.Done()
		fs.mu.Lock()

		isCurrent := fs.lastEvent[eventName] == timer
		if isCurrent {
			delete(fs.lastEvent, eventName)
		}

		stopped := fs.stopped
		fs.mu.Unlock()

		if isCurrent && !stopped {
			callback()
		}
	})

	fs.lastEvent[eventName] = timer
}

func (fs *FS) close() {
	fs.cancel()

	fs.mu.Lock()
	fs.stopped = true
	for name, timer := range fs.lastEvent {
		if timer.Stop() {
			fs.wg.Done()
		}
		delete(fs.lastEvent, name)
	}
	fs.mu.Unlock()

	fs.wg.Wait()

	close(fs.watchEvents)
	if err := fs.fsw.Close(); err != nil {
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
