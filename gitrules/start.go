package gitrules

import (
	"context"
	"fmt"
	"strings"

	"morphstudio/runner"
)

// Start is the outcome of the start phase check.
type Start struct {
	Mode   string `json:"mode"`
	Branch string `json:"branch"`
	Local  string `json:"local"`
	Remote string `json:"remote"`
	Reason string `json:"reason"`
}

const (
	fetchKey          = "git fetch -q origin"
	branchKey         = "git rev-parse --abbrev-ref HEAD"
	headKey           = "git rev-parse HEAD"
	remoteKey         = "git rev-parse origin/main"
	statusKey         = "git status --porcelain"
	ancestorRemoteKey = "git merge-base --is-ancestor origin/main HEAD"
	ancestorHeadKey   = "git merge-base --is-ancestor HEAD origin/main"
	pullKey           = "git pull -q --ff-only"
)

func startError(key string, err error, stderr string) Start {
	msg := ""
	if err != nil {
		msg = err.Error()
	} else {
		msg = strings.TrimSpace(stderr)
	}
	return Start{Mode: "error", Reason: key + " failed: " + msg}
}

// StartCheck inspects the repository at dir and reports how the start phase
// should proceed.
func StartCheck(ctx context.Context, r runner.Runner, dir string) Start {
	res, err := r.Run(ctx, dir, nil, "git", "fetch", "-q", "origin")
	if err != nil {
		return startError(fetchKey, err, "")
	}
	if res.Code != 0 {
		return startError(fetchKey, nil, res.Stderr)
	}

	res, err = r.Run(ctx, dir, nil, "git", "rev-parse", "--abbrev-ref", "HEAD")
	if err != nil {
		return startError(branchKey, err, "")
	}
	if res.Code != 0 {
		return startError(branchKey, nil, res.Stderr)
	}
	branch := strings.TrimSpace(res.Stdout)

	res, err = r.Run(ctx, dir, nil, "git", "rev-parse", "HEAD")
	if err != nil {
		return startError(headKey, err, "")
	}
	if res.Code != 0 {
		return startError(headKey, nil, res.Stderr)
	}
	local := strings.TrimSpace(res.Stdout)

	res, err = r.Run(ctx, dir, nil, "git", "rev-parse", "origin/main")
	if err != nil {
		return startError(remoteKey, err, "")
	}
	if res.Code != 0 {
		return startError(remoteKey, nil, res.Stderr)
	}
	remote := strings.TrimSpace(res.Stdout)

	res, err = r.Run(ctx, dir, nil, "git", "status", "--porcelain")
	if err != nil {
		return startError(statusKey, err, "")
	}
	if res.Code != 0 {
		return startError(statusKey, nil, res.Stderr)
	}
	dirty := 0
	for _, line := range strings.Split(res.Stdout, "\n") {
		if strings.TrimSpace(line) != "" {
			dirty++
		}
	}

	start := Start{Branch: branch, Local: local, Remote: remote}

	if branch != "main" {
		start.Mode = "resume"
		start.Reason = "on branch " + branch
		return start
	}
	if dirty > 0 {
		start.Mode = "resume"
		start.Reason = fmt.Sprintf("dirty tree (%d paths)", dirty)
		return start
	}
	if local == remote {
		start.Mode = "fresh"
		return start
	}

	res, err = r.Run(ctx, dir, nil, "git", "merge-base", "--is-ancestor", "origin/main", "HEAD")
	if err != nil {
		return startError(ancestorRemoteKey, err, "")
	}
	if res.Code == 0 {
		start.Mode = "resume"
		start.Reason = "local ahead of origin/main"
		return start
	}

	res, err = r.Run(ctx, dir, nil, "git", "merge-base", "--is-ancestor", "HEAD", "origin/main")
	if err != nil {
		return startError(ancestorHeadKey, err, "")
	}
	if res.Code == 0 {
		res, err = r.Run(ctx, dir, nil, "git", "pull", "-q", "--ff-only")
		if err != nil {
			return startError(pullKey, err, "")
		}
		if res.Code != 0 {
			return startError(pullKey, nil, res.Stderr)
		}
		start.Mode = "fresh"
		start.Local = remote
		start.Remote = remote
		start.Reason = "pulled to " + remote
		return start
	}

	start.Mode = "diverged"
	start.Reason = "local " + local + " and origin/main " + remote + " diverged"
	return start
}
