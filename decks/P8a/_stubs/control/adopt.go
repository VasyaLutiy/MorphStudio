package control

import (
	"context"
	"errors"
)

type Adopter interface {
	AdoptProject(ctx context.Context, name, dir, repoURL, language string) (ProjectView, error)
}

var ErrUnsupported = errors.New("stub ErrUnsupported")

func Adopt(ctx context.Context, c Control, name, dir, repoURL, language string) (ProjectView, error) {
	return ProjectView{}, errors.New("stub Adopt")
}
