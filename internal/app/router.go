package app

import (
	"context"
	"errors"
	"fmt"
	"os"
)

var (
	ErrNotEnoughArguments = errors.New("not enough arguments, min 2")
)

func route(ctx context.Context) error {
	args := os.Args
	if len(args) < 2 {
		return ErrNotEnoughArguments
	}

	switch args[1] {
	case "daemon":
		return runDaemon(ctx, args[1:])
	case "publish":
		runPublish(ctx, args[1:])
	case "policy":
		runPolicy(args[1:])
	case "help":
		runUsage()
	default:
		fmt.Printf("unknown command: %s\n\n", args[1])
		runUsage()
	}

	return nil
}
