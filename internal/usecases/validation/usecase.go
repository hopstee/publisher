package validation

import (
	"fmt"
	"log/slog"
	"publisher/internal/domain"
)

type Reader interface {
	Read(domain.WatcherEvent) (domain.ReadResult, error)
}

type Policy interface {
	Check(domain.ReadResult) domain.CheckVerdict
}

type Service struct {
	reader Reader
	policy Policy
}

func NewService(r Reader, p Policy) *Service {
	return &Service{reader: r, policy: p}
}

func (s *Service) Execute(filePath string) error {
	doc, err := s.reader.Read(domain.WatcherEvent{
		Path: filePath,
	})
	if err != nil {
		return fmt.Errorf("failed to read provided file: %w", err)
	}
	verdict := s.policy.Check(doc)

	slog.Info("policy check done", slog.Any("verdict", verdict))

	return nil
}
