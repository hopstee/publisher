package app

import (
	"context"
	"log/slog"
	"os"
	"os/signal"
	"syscall"
)

func Run() error {
	slog.SetLogLoggerLevel(slog.LevelDebug)

	ctx, cancel := ctxWithSysNotify()
	defer cancel()

	return route(ctx)
}

func ctxWithSysNotify() (context.Context, context.CancelFunc) {
	return signal.NotifyContext(
		context.Background(),
		os.Interrupt,
		syscall.SIGTERM,
		syscall.SIGKILL,
		syscall.SIGINT,
		syscall.SIGQUIT,
	)
}
