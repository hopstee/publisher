package usecases

import (
	"context"
	"log/slog"
	"publisher/internal/domain"
)

type Watcher interface {
	Watch(context.Context) (<-chan domain.WatcherEvent, error)
}

type Reader interface {
	Read(string) ([]byte, error)
}

type Policy interface {
	Check(string) ([]byte, error)
}

type Publisher interface {
	Publish(string) ([]byte, error)
}

type runDaemon struct {
	watcher   Watcher
	reader    Reader
	policy    Policy
	publisher Publisher
}

func NewRunDaemon(w Watcher) runDaemon {
	return runDaemon{watcher: w}
}

func (r runDaemon) Execute(ctx context.Context) error {
	events, err := r.watcher.Watch(ctx)
	if err != nil {
		return err
	}

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case e := <-events:
			slog.Debug("new event", slog.Any("data", e))
		}
	}
}
