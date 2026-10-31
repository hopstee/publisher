package app

import (
	"publisher/internal/adapters/policy"
	"publisher/internal/adapters/reader"
	"publisher/internal/usecases/validation"
)

func runPolicy(args []string) error {
	if len(args) < 2 {
		return ErrNoDirPath
	}
	validator := validation.NewService(
		reader.NewMDReader(""),
		policy.NewDefaultPolicy(true, false), // TODO: arguments need to get from the config
	)
	return validator.Execute(args[1])
}
