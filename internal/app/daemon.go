package app

import (
	"context"
	"publisher/internal/adapters/policy"
	"publisher/internal/adapters/publisher"
	"publisher/internal/adapters/reader"
	"publisher/internal/adapters/watcher"
	"publisher/internal/usecases/processing"
)

func runDaemon(ctx context.Context, args []string) error {
	if len(args) < 2 {
		return ErrNoDirPath
	}
	process := processing.NewService(
		watcher.NewFSWatcher(args[1]),
		reader.NewMDReader(""),                   // TODO: argument need to get from the config
		policy.NewDefaultPolicy(false, true),     // TODO: arguments need to get from the config
		publisher.NewGitHubPublisher("", "", ""), // TODO: arguments need to get from the config
	)
	return process.Execute(ctx)
}
