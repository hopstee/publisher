package app

import (
	"context"
	"publisher/internal/adapters/policy"
	"publisher/internal/adapters/reader"
	"publisher/internal/adapters/watcher"
	runpolicy "publisher/internal/usecases/policy"
)

func runPolicy(ctx context.Context, args []string) error {
	if len(args) < 2 {
		return ErrNoDirPath
	}
	rp := runpolicy.NewRunPolicy(
		watcher.NewWatcher(args[1]),
		reader.NewReader(""),
		policy.NewPolicy(true, false), // TODO: arguments need to get from the config
	)
	return rp.Execute(ctx)
}
