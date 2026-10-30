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

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM, syscall.SIGINT)
	defer stop()

	return route(ctx)
}
