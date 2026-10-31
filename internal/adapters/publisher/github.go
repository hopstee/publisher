package publisher

import (
	"context"
	"log/slog"
	"publisher/internal/domain"
)

type GitHub struct {
	repoPath   string
	contentDir string
	branchName string
}

func NewGitHubPublisher(rp, cd, bn string) *GitHub {
	return &GitHub{repoPath: rp, contentDir: cd, branchName: bn}
}

func (gh *GitHub) PublishAll(ctx context.Context, verdicts <-chan domain.CheckVerdict) {
	go gh.listenVerdicts(ctx, verdicts)
}

func (gh *GitHub) Publish(ctx context.Context, v domain.CheckVerdict) error {
	return gh.handleVerdict(ctx, v)
}

func (gh *GitHub) listenVerdicts(ctx context.Context, verdicts <-chan domain.CheckVerdict) {
	for {
		select {
		case <-ctx.Done():
			return
		case v, ok := <-verdicts:
			if !ok {
				return
			}
			slog.Debug("new publish event", slog.Any("verdict", v))
			gh.handleVerdict(ctx, v)
		}
	}
}

func (gh *GitHub) handleVerdict(ctx context.Context, v domain.CheckVerdict) error {
	if v.CanPublish {
		// if publishing is immediate, then do it
		// if not - add to outbox
	}
	return nil
}
