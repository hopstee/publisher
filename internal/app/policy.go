package app

import (
	"publisher/internal/adapters/policy"
	"publisher/internal/adapters/reader"
	runpolicy "publisher/internal/usecases/policy"
)

func runPolicy(args []string) error {
	if len(args) < 2 {
		return ErrNoDirPath
	}
	rp := runpolicy.NewRunPolicy(
		reader.NewReader(""),
		policy.NewPolicy(true, false), // TODO: arguments need to get from the config
	)
	return rp.Execute(args[1])
}
