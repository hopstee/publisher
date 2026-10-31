package processing

import (
	"context"
	"fmt"
	"publisher/internal/domain"
)

type Watcher interface {
	Watch(context.Context) (<-chan domain.WatcherEvent, error)
}

type Reader interface {
	ReadAll(context.Context, <-chan domain.WatcherEvent) <-chan domain.ReadResult
}

type Policy interface {
	CheckAll(context.Context, <-chan domain.ReadResult) <-chan domain.CheckVerdict
}

type Publisher interface {
	PublishAll(context.Context, <-chan domain.CheckVerdict)
}

type Service struct {
	watcher   Watcher
	reader    Reader
	policy    Policy
	publisher Publisher
}

func NewService(w Watcher, r Reader, p Policy, pb Publisher) *Service {
	return &Service{watcher: w, reader: r, policy: p, publisher: pb}
}

func (s *Service) Execute(ctx context.Context) error {
	events, err := s.watcher.Watch(ctx)
	if err != nil {
		return fmt.Errorf("failed to start fs watcher: %w", err)
	}

	docs := s.reader.ReadAll(ctx, events)
	checked := s.policy.CheckAll(ctx, docs)
	s.publisher.PublishAll(ctx, checked)

	<-ctx.Done()
	return ctx.Err()
}
