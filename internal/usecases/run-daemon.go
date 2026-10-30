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
	Read(context.Context, <-chan domain.WatcherEvent) <-chan domain.ReadResult
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

func NewRunDaemon(w Watcher, r Reader) runDaemon {
	return runDaemon{watcher: w, reader: r}
}

func (r runDaemon) Execute(ctx context.Context) error {
	events, err := r.watcher.Watch(ctx)
	if err != nil {
		return err
	}

	docs := r.reader.Read(ctx, events)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case d := <-docs:
			slog.Debug("new doc", slog.Any("data", d))
		}
	}
}
