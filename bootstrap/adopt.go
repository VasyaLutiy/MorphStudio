package bootstrap

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"morphstudio/github"
	"morphstudio/runner"
)

// AdoptSpec describes a request to adopt an existing project directory.
type AdoptSpec struct {
	Name            string
	Language        string
	RepoURL         string
	Dir             string
	CredentialsFile string
}

// ErrRefused marks a refusal to adopt a directory.
var ErrRefused = errors.New("bootstrap: refused")

// refuse returns an error wrapping ErrRefused with the given detail.
func refuse(detail string) error {
	return fmt.Errorf("%w: %s", ErrRefused, detail)
}

// Adopt checks that s.Dir is the named GitHub repository on main with a
// record, and wires only the credential helper into it. It never runs morph
// init, never commits, never pushes and never sets a git identity: the
// repository stays as it is.
func Adopt(ctx context.Context, r runner.Runner, s AdoptSpec) (Report, error) {
	if !nameRE.MatchString(s.Name) {
		return Report{}, refuse(fmt.Sprintf("bad project name %q", s.Name))
	}
	switch s.Language {
	case "typescript", "python", "go":
	default:
		return Report{}, refuse(fmt.Sprintf("unknown language %q", s.Language))
	}
	owner, repo, err := github.Parse(s.RepoURL)
	if err != nil {
		return Report{}, fmt.Errorf("%w: %s", ErrRefused, err.Error())
	}
	if !filepath.IsAbs(s.Dir) {
		return Report{}, refuse("dir must be absolute: " + s.Dir)
	}
	info, err := os.Stat(s.Dir)
	if err != nil || !info.IsDir() {
		return Report{}, refuse("not a directory: " + s.Dir)
	}
	if _, err := os.Stat(filepath.Join(s.Dir, "contour.yaml")); err != nil {
		return Report{}, refuse("no contour.yaml in " + s.Dir)
	}

	env := []string{"GIT_TERMINAL_PROMPT=0"}
	rep := Report{Dir: s.Dir}

	run := func(n int, name string, args ...string) (runner.Result, error) {
		rep.Steps = append(rep.Steps, runner.Key(name, args...))
		res, err := r.Run(ctx, s.Dir, env, name, args...)
		if err != nil {
			return res, fmt.Errorf("bootstrap: step %d: %w", n, err)
		}
		return res, nil
	}

	res, err := run(1, "git", "rev-parse", "--is-inside-work-tree")
	if err != nil {
		return rep, err
	}
	if res.Code != 0 {
		return rep, refuse("not a git repository: " + s.Dir)
	}

	res, err = run(2, "git", "rev-parse", "--abbrev-ref", "HEAD")
	if err != nil {
		return rep, err
	}
	if res.Code != 0 {
		return rep, fmt.Errorf("bootstrap: step 2 failed (exit %d): %s", res.Code, strings.TrimSpace(res.Stderr))
	}
	if branch := strings.TrimSpace(res.Stdout); branch != "main" {
		return rep, refuse("on branch " + branch + ", not main")
	}

	res, err = run(3, "git", "remote", "get-url", "origin")
	if err != nil {
		return rep, err
	}
	if res.Code != 0 {
		return rep, refuse("no origin remote")
	}
	origin := strings.TrimSpace(res.Stdout)
	oOwner, oRepo, oErr := github.Parse(origin)
	if oErr != nil || oOwner != owner || oRepo != repo {
		return rep, refuse("origin " + origin + " is not " + s.RepoURL)
	}

	res, err = run(4, "git", "config", "credential.helper", "store --file="+s.CredentialsFile)
	if err != nil {
		return rep, err
	}
	if res.Code != 0 {
		return rep, fmt.Errorf("bootstrap: step 4 failed (exit %d): %s", res.Code, strings.TrimSpace(res.Stderr))
	}

	return rep, nil
}
