package bootstrap

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"morphstudio/github"
	"morphstudio/runner"
)

// Spec describes a project creation request.
type Spec struct {
	Name            string
	Language        string
	RepoURL         string
	ProjectsDir     string
	CredentialsFile string
	MorphBin        string
}

// Report is the outcome of Create.
type Report struct {
	Dir   string   `json:"dir"`
	Steps []string `json:"steps"`
}

var nameRE = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{0,39}$`)

// Create validates s and runs the project bootstrap sequence through r.
func Create(ctx context.Context, r runner.Runner, s Spec) (Report, error) {
	if !nameRE.MatchString(s.Name) {
		return Report{}, fmt.Errorf("bootstrap: bad project name %q", s.Name)
	}
	switch s.Language {
	case "typescript", "python", "go":
	default:
		return Report{}, fmt.Errorf("bootstrap: unknown language %q", s.Language)
	}
	if _, _, err := github.Parse(s.RepoURL); err != nil {
		return Report{}, err
	}
	if s.MorphBin == "" {
		return Report{}, fmt.Errorf("bootstrap: morph binary not configured")
	}
	dir := filepath.Join(s.ProjectsDir, s.Name)
	if _, err := os.Stat(dir); err == nil {
		return Report{}, fmt.Errorf("bootstrap: directory exists: %s", dir)
	}

	env := []string{"GIT_TERMINAL_PROMPT=0"}
	rep := Report{Dir: dir}

	run := func(n int, stepDir, name string, args ...string) error {
		rep.Steps = append(rep.Steps, runner.Key(name, args...))
		res, err := r.Run(ctx, stepDir, env, name, args...)
		if err != nil {
			return fmt.Errorf("bootstrap: step %d: %w", n, err)
		}
		if res.Code != 0 {
			return fmt.Errorf("bootstrap: step %d failed (exit %d): %s", n, res.Code, strings.TrimSpace(res.Stderr))
		}
		return nil
	}

	initArgs := []string{"init", "--root", dir, "--name", s.Name, "--language", s.Language}
	if s.Language == "go" {
		initArgs = append(initArgs, "--module", s.Name)
	}
	if err := run(1, s.ProjectsDir, s.MorphBin, initArgs...); err != nil {
		return rep, err
	}
	if err := run(2, dir, "git", "init", "-q", "-b", "main"); err != nil {
		return rep, err
	}
	if err := run(3, dir, "git", "config", "credential.helper", "store --file="+s.CredentialsFile); err != nil {
		return rep, err
	}
	if err := run(4, dir, "git", "config", "user.name", "morphd"); err != nil {
		return rep, err
	}
	if err := run(5, dir, "git", "config", "user.email", "morphd@localhost"); err != nil {
		return rep, err
	}
	if err := run(6, dir, "git", "remote", "add", "origin", s.RepoURL); err != nil {
		return rep, err
	}
	if err := run(7, dir, "git", "add", "-A"); err != nil {
		return rep, err
	}
	if err := run(8, dir, "git", "commit", "-q", "-m", fmt.Sprintf("morph init: %s (%s)", s.Name, s.Language)); err != nil {
		return rep, err
	}
	return rep, nil
}

// FirstPush pushes main to origin unless it is already pushed.
func FirstPush(ctx context.Context, r runner.Runner, dir string) (bool, error) {
	env := []string{"GIT_TERMINAL_PROMPT=0"}
	res, err := r.Run(ctx, dir, env, "git", "rev-parse", "--verify", "-q", "origin/main")
	if err != nil {
		return false, fmt.Errorf("bootstrap: push: %w", err)
	}
	if res.Code == 0 {
		return false, nil
	}
	res, err = r.Run(ctx, dir, env, "git", "push", "-q", "-u", "origin", "main")
	if err != nil {
		return false, fmt.Errorf("bootstrap: push: %w", err)
	}
	if res.Code != 0 {
		return false, fmt.Errorf("bootstrap: push failed (exit %d): %s", res.Code, strings.TrimSpace(res.Stderr))
	}
	return true, nil
}
