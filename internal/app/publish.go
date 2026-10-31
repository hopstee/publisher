package app

import (
	"context"
	"publisher/internal/adapters/policy"
	"publisher/internal/adapters/publisher"
	"publisher/internal/adapters/reader"
	"publisher/internal/usecases/posting"
)

func runPublish(ctx context.Context, args []string) error {
	if len(args) < 2 {
		return ErrNoDirPath
	}
	poster := posting.NewService(
		reader.NewMDReader(""),
		policy.NewDefaultPolicy(true, false),     // TODO: arguments need to get from the config
		publisher.NewGitHubPublisher("", "", ""), // TODO: arguments need to get from the config
	)
	return poster.Execute(ctx, args[1])
}
