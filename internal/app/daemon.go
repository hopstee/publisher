package app

import (
	"context"
	"errors"
	"publisher/internal/adapters/watcher"
	"publisher/internal/usecases"
)

var (
	ErrNoDirPath = errors.New("dir path is not specified")
)

func runDaemon(ctx context.Context, args []string) error {
	if len(args) < 2 {
		return ErrNoDirPath
	}
	rd := usecases.NewRunDaemon(
		watcher.NewWatcher(args[1]),
	)
	return rd.Execute(ctx)
}
