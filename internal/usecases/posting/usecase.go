package posting

import (
	"context"
	"fmt"
	"publisher/internal/domain"
)

type Reader interface {
	Read(domain.WatcherEvent) (domain.ReadResult, error)
}

type Policy interface {
	Check(domain.ReadResult) domain.CheckVerdict
}

type Publisher interface {
	Publish(context.Context, domain.CheckVerdict) error
}

type Service struct {
	reader    Reader
	policy    Policy
	publisher Publisher
}

func NewService(r Reader, p Policy, pb Publisher) *Service {
	return &Service{reader: r, policy: p, publisher: pb}
}

func (s *Service) Execute(ctx context.Context, filePath string) error {
	doc, err := s.reader.Read(domain.WatcherEvent{
		Path: filePath,
	})
	if err != nil {
		return fmt.Errorf("failed to read provided file: %w", err)
	}
	verdict := s.policy.Check(doc)
	return s.publisher.Publish(ctx, verdict)
}
