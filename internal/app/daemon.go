package app

import (
	"context"
	"publisher/internal/adapters/policy"
	"publisher/internal/adapters/reader"
	"publisher/internal/adapters/watcher"
	rundaemon "publisher/internal/usecases/daemon"
)

func runDaemon(ctx context.Context, args []string) error {
	if len(args) < 2 {
		return ErrNoDirPath
	}
	rd := rundaemon.NewRunDaemon(
		watcher.NewWatcher(args[1]),
		reader.NewReader(""),
		policy.NewPolicy(false, true), // TODO: arguments need to get from the config
	)
	return rd.Execute(ctx)
}
