package bootstrap

import (
	"context"
	"errors"

	"morphstudio/runner"
)

type AdoptSpec struct {
	Name            string
	Language        string
	RepoURL         string
	Dir             string
	CredentialsFile string
}

var ErrRefused = errors.New("stub ErrRefused")

func Adopt(ctx context.Context, r runner.Runner, s AdoptSpec) (Report, error) {
	return Report{}, errors.New("stub Adopt")
}
