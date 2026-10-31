package runpolicy

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

type runPolicy struct {
	reader Reader
	policy Policy
}

func NewRunPolicy(r Reader, p Policy) runPolicy {
	return runPolicy{reader: r, policy: p}
}

func (r runPolicy) Execute(filePath string) error {
	docs, err := r.reader.Read(domain.WatcherEvent{
		Path: filePath,
	})
	if err != nil {
		return fmt.Errorf("failed to read provided file: %w", err)
	}
	checked := r.policy.Check(docs)

	slog.Info("policy check done", slog.Any("verdict", checked))

	return nil
}
