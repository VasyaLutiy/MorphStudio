package control

import (
	"context"
	"errors"
)

// Adopter is the optional surface a Control may implement to register a
// repository that already exists at dir: no init, no commit, no push.
type Adopter interface {
	AdoptProject(ctx context.Context, name, dir, repoURL, language string) (ProjectView, error)
}

// ErrUnsupported reports that a Control does not implement Adopter.
var ErrUnsupported = errors.New("adopt not supported")

// Adopt registers a repository that already exists at dir through c when c
// implements Adopter, and returns ErrUnsupported otherwise.
func Adopt(ctx context.Context, c Control, name, dir, repoURL, language string) (ProjectView, error) {
	if a, ok := c.(Adopter); ok {
		return a.AdoptProject(ctx, name, dir, repoURL, language)
	}
	return ProjectView{}, ErrUnsupported
}
