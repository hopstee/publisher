package runpolicy

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
	Check(context.Context, <-chan domain.ReadResult) <-chan domain.CheckVerdict
}

type runPolicy struct {
	watcher Watcher
	reader  Reader
	policy  Policy
}

func NewRunPolicy(w Watcher, r Reader, p Policy) runPolicy {
	return runPolicy{watcher: w, reader: r, policy: p}
}

func (r runPolicy) Execute(ctx context.Context) error {
	events, err := r.watcher.Watch(ctx)
	if err != nil {
		return err
	}

	docs := r.reader.Read(ctx, events)
	checked := r.policy.Check(ctx, docs)

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case c := <-checked:
			slog.Debug("new doc", slog.Any("data", c))
		}
	}
}
