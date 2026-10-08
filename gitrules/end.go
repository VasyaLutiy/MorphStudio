package gitrules

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"morphstudio/runner"
)

// End is the outcome of the end phase check.
type End struct {
	OK       bool     `json:"ok"`
	Local    string   `json:"local"`
	Remote   string   `json:"remote"`
	Problems []string `json:"problems"`
}

// EndCheck inspects the repository at dir and reports whether the end phase
// for phase is complete.
func EndCheck(ctx context.Context, r runner.Runner, dir, phase string) End {
	end := End{Problems: []string{}}

	step := func(name string, args ...string) (runner.Result, bool) {
		res, err := r.Run(ctx, dir, nil, name, args...)
		if err != nil {
			end.Problems = append(end.Problems, runner.Key(name, args...)+" failed: "+strings.TrimSpace(err.Error()))
			return res, false
		}
		if res.Code != 0 {
			end.Problems = append(end.Problems, runner.Key(name, args...)+" failed: "+strings.TrimSpace(res.Stderr))
			return res, false
		}
		return res, true
	}

	if _, ok := step("git", "fetch", "-q", "origin"); !ok {
		return end
	}

	branchRes, ok := step("git", "rev-parse", "--abbrev-ref", "HEAD")
	if !ok {
		return end
	}

	headRes, ok := step("git", "rev-parse", "HEAD")
	if !ok {
		return end
	}

	remoteRes, ok := step("git", "rev-parse", "origin/main")
	if !ok {
		return end
	}

	statusRes, ok := step("git", "status", "--porcelain")
	if !ok {
		return end
	}

	branchesRes, ok := step("git", "branch", "--list", "morph/*", "--no-merged", "main")
	if !ok {
		return end
	}

	branch := strings.TrimSpace(branchRes.Stdout)
	end.Local = strings.TrimSpace(headRes.Stdout)
	end.Remote = strings.TrimSpace(remoteRes.Stdout)

	if branch != "main" {
		end.Problems = append(end.Problems, "on branch "+branch+", not main")
	}
	if end.Local != end.Remote {
		end.Problems = append(end.Problems, "not pushed: HEAD "+end.Local+", origin/main "+end.Remote)
	}

	dirty := 0
	for _, line := range strings.Split(statusRes.Stdout, "\n") {
		if strings.TrimSpace(line) != "" {
			dirty++
		}
	}
	if dirty > 0 {
		end.Problems = append(end.Problems, fmt.Sprintf("dirty tree (%d paths)", dirty))
	}

	for _, line := range strings.Split(branchesRes.Stdout, "\n") {
		name := strings.TrimSpace(line)
		name = strings.TrimSpace(strings.TrimPrefix(name, "* "))
		if name == "" {
			continue
		}
		end.Problems = append(end.Problems, "unmerged run branch "+name)
	}

	data, err := os.ReadFile(filepath.Join(dir, "docs", "MEASURE.md"))
	if err != nil {
		end.Problems = append(end.Problems, "docs/MEASURE.md missing")
	} else if !measureRow(string(data), phase) {
		end.Problems = append(end.Problems, "no MEASURE row for "+phase)
	}

	end.OK = len(end.Problems) == 0
	return end
}

func measureRow(content, phase string) bool {
	for _, line := range strings.Split(content, "\n") {
		if !strings.HasPrefix(line, "|") {
			continue
		}
		rest := line[1:]
		idx := strings.Index(rest, "|")
		if idx < 0 {
			continue
		}
		cell := strings.TrimSpace(rest[:idx])
		if cell == phase || strings.HasPrefix(cell, phase+" ") {
			return true
		}
	}
	return false
}
