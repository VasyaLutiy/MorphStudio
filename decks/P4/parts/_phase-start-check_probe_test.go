package gitrules

import (
	"context"
	"testing"

	"morphstudio/internal/testhelp"
	"morphstudio/runner"
)

// pSCFake scripts the five reads of example 1 with the given branch, HEAD, origin/main and status.
func pSCFake(branch, head, remote, status string) *runner.Fake {
	return &runner.Fake{Script: map[string]runner.Result{
		"git rev-parse --abbrev-ref HEAD": {Stdout: branch + "\n"},
		"git rev-parse HEAD":              {Stdout: head + "\n"},
		"git rev-parse origin/main":       {Stdout: remote + "\n"},
		"git status --porcelain":          {Stdout: status},
	}}
}

func pSCKeys(f *runner.Fake) []string {
	out := []string{}
	for _, c := range f.Calls {
		out = append(out, runner.Key(c.Name, c.Args...))
	}
	return out
}

func pSCDirs(f *runner.Fake) []string {
	out := []string{}
	for _, c := range f.Calls {
		out = append(out, c.Dir)
	}
	return out
}

var pSCFive = []string{"git fetch -q origin", "git rev-parse --abbrev-ref HEAD", "git rev-parse HEAD", "git rev-parse origin/main", "git status --porcelain"}

func pSCWith(rest ...string) []string {
	return append(append([]string{}, pSCFive...), rest...)
}

func TestProbePhaseStartCheckExample1(t *testing.T) {
	f := pSCFake("main", "aaa111", "aaa111", "")
	got := StartCheck(context.Background(), f, "/p")
	testhelp.Equal(t, "example 1 Start", got, Start{Mode: "fresh", Branch: "main", Local: "aaa111", Remote: "aaa111", Reason: ""})
	testhelp.Equal(t, "example 1 call keys", pSCKeys(f), pSCFive)
	testhelp.Equal(t, "example 1 call dirs", pSCDirs(f), []string{"/p", "/p", "/p", "/p", "/p"})
	// variant: other hashes and dir
	f = pSCFake("main", "f00d42", "f00d42", "\n")
	got = StartCheck(context.Background(), f, "/srv/x")
	testhelp.Equal(t, "example 1 variant Start", got, Start{Mode: "fresh", Branch: "main", Local: "f00d42", Remote: "f00d42"})
	testhelp.Equal(t, "example 1 variant call dirs", pSCDirs(f), []string{"/srv/x", "/srv/x", "/srv/x", "/srv/x", "/srv/x"})
}

func TestProbePhaseStartCheckExample2(t *testing.T) {
	f := pSCFake("main", "bbb222", "aaa111", "")
	f.Script["git merge-base --is-ancestor origin/main HEAD"] = runner.Result{Code: 0}
	got := StartCheck(context.Background(), f, "/p")
	testhelp.Equal(t, "example 2 Start", got, Start{Mode: "resume", Branch: "main", Local: "bbb222", Remote: "aaa111", Reason: "local ahead of origin/main"})
	testhelp.Equal(t, "example 2 call keys", pSCKeys(f), pSCWith("git merge-base --is-ancestor origin/main HEAD"))
	// variant: other hashes
	f = pSCFake("main", "e5e5e5", "d4d4d4", "")
	f.Script["git merge-base --is-ancestor origin/main HEAD"] = runner.Result{Code: 0}
	got = StartCheck(context.Background(), f, "/p")
	testhelp.Equal(t, "example 2 variant Start", got, Start{Mode: "resume", Branch: "main", Local: "e5e5e5", Remote: "d4d4d4", Reason: "local ahead of origin/main"})
}

func TestProbePhaseStartCheckExample3(t *testing.T) {
	f := pSCFake("main", "aaa111", "ccc333", "")
	f.Script["git merge-base --is-ancestor origin/main HEAD"] = runner.Result{Code: 1}
	f.Script["git merge-base --is-ancestor HEAD origin/main"] = runner.Result{Code: 0}
	f.Script["git pull -q --ff-only"] = runner.Result{Code: 0}
	got := StartCheck(context.Background(), f, "/p")
	testhelp.Equal(t, "example 3 Start", got, Start{Mode: "fresh", Branch: "main", Local: "ccc333", Remote: "ccc333", Reason: "pulled to ccc333"})
	testhelp.Equal(t, "example 3 call keys", pSCKeys(f), pSCWith("git merge-base --is-ancestor origin/main HEAD", "git merge-base --is-ancestor HEAD origin/main", "git pull -q --ff-only"))
	// variant: the pull fails (marked (!)): Mode "error" with its stderr
	f = pSCFake("main", "aaa111", "9a9a9a", "")
	f.Script["git merge-base --is-ancestor origin/main HEAD"] = runner.Result{Code: 1}
	f.Script["git merge-base --is-ancestor HEAD origin/main"] = runner.Result{Code: 0}
	f.Script["git pull -q --ff-only"] = runner.Result{Code: 1, Stderr: "fatal: Not possible to fast-forward, aborting.\n"}
	got = StartCheck(context.Background(), f, "/p")
	testhelp.Equal(t, "example 3 variant (pull fails) Mode, Reason", []string{got.Mode, got.Reason}, []string{"error", "git pull -q --ff-only failed: fatal: Not possible to fast-forward, aborting."})
	testhelp.Equal(t, "example 3 variant (pull fails) calls", len(f.Calls), 8)
}

func TestProbePhaseStartCheckExample4(t *testing.T) {
	f := pSCFake("main", "aaa111", "ccc333", "")
	f.Script["git merge-base --is-ancestor origin/main HEAD"] = runner.Result{Code: 1}
	f.Script["git merge-base --is-ancestor HEAD origin/main"] = runner.Result{Code: 1}
	got := StartCheck(context.Background(), f, "/p")
	testhelp.Equal(t, "example 4 Start", got, Start{Mode: "diverged", Branch: "main", Local: "aaa111", Remote: "ccc333", Reason: "local aaa111 and origin/main ccc333 diverged"})
	testhelp.Equal(t, "example 4 call keys", pSCKeys(f), pSCWith("git merge-base --is-ancestor origin/main HEAD", "git merge-base --is-ancestor HEAD origin/main"))
	// variant: other hashes
	f = pSCFake("main", "1234567", "89abcde", "")
	f.Script["git merge-base --is-ancestor origin/main HEAD"] = runner.Result{Code: 1}
	f.Script["git merge-base --is-ancestor HEAD origin/main"] = runner.Result{Code: 1}
	got = StartCheck(context.Background(), f, "/p")
	testhelp.Equal(t, "example 4 variant Reason", got.Reason, "local 1234567 and origin/main 89abcde diverged")
	// variant: a Run error of an unmarked command (6) is Mode "error" too
	f = pSCFake("main", "aaa111", "ccc333", "")
	f.Errors = map[string]string{"git merge-base --is-ancestor origin/main HEAD": "runner: signal: killed"}
	got = StartCheck(context.Background(), f, "/p")
	testhelp.Equal(t, "example 4 variant (Run error of command 6) Mode, Reason", []string{got.Mode, got.Reason}, []string{"error", "git merge-base --is-ancestor origin/main HEAD failed: runner: signal: killed"})
	testhelp.Equal(t, "example 4 variant (Run error of command 6) calls", len(f.Calls), 6)
}

func TestProbePhaseStartCheckExample5(t *testing.T) {
	f := pSCFake("main", "aaa111", "aaa111", " M docs/TASK_P18.md\n?? decks/p18/\n")
	got := StartCheck(context.Background(), f, "/p")
	testhelp.Equal(t, "example 5 Mode, Reason", []string{got.Mode, got.Reason}, []string{"resume", "dirty tree (2 paths)"})
	testhelp.Equal(t, "example 5 calls", pSCKeys(f), pSCFive)
	// variant: three paths and HEAD behind (dirty wins before the hash comparison)
	f = pSCFake("main", "aaa111", "bbb222", "?? a\n?? b\n M c\n")
	got = StartCheck(context.Background(), f, "/p")
	testhelp.Equal(t, "example 5 variant Mode, Reason", []string{got.Mode, got.Reason}, []string{"resume", "dirty tree (3 paths)"})
	testhelp.Equal(t, "example 5 variant calls", len(f.Calls), 5)
}

func TestProbePhaseStartCheckExample6(t *testing.T) {
	f := pSCFake("morph/20261008-083312", "aaa111", "aaa111", "")
	got := StartCheck(context.Background(), f, "/p")
	testhelp.Equal(t, "example 6 Start", got, Start{Mode: "resume", Branch: "morph/20261008-083312", Local: "aaa111", Remote: "aaa111", Reason: "on branch morph/20261008-083312"})
	testhelp.Equal(t, "example 6 calls (the status is still read)", pSCKeys(f), pSCFive)
	// variant: a branch and a dirty tree: the branch reason wins
	f = pSCFake("feature-x", "aaa111", "bbb222", "?? z\n")
	got = StartCheck(context.Background(), f, "/p")
	testhelp.Equal(t, "example 6 variant Mode, Branch, Reason", []string{got.Mode, got.Branch, got.Reason}, []string{"resume", "feature-x", "on branch feature-x"})
}

func TestProbePhaseStartCheckExample7(t *testing.T) {
	f := &runner.Fake{Script: map[string]runner.Result{"git fetch -q origin": {Code: 128, Stderr: "fatal: unable to access 'https://github.com/o/r/': Could not resolve host\n"}}}
	got := StartCheck(context.Background(), f, "/p")
	testhelp.Equal(t, "example 7 Mode, Reason", []string{got.Mode, got.Reason}, []string{"error", "git fetch -q origin failed: fatal: unable to access 'https://github.com/o/r/': Could not resolve host"})
	testhelp.Equal(t, "example 7 calls", pSCKeys(f), []string{"git fetch -q origin"})
	// variant: the third read fails; nothing runs after it
	f = pSCFake("main", "aaa111", "aaa111", "")
	f.Script["git rev-parse origin/main"] = runner.Result{Code: 128, Stderr: "fatal: ambiguous argument 'origin/main'\n"}
	got = StartCheck(context.Background(), f, "/p")
	testhelp.Equal(t, "example 7 variant Mode, Reason", []string{got.Mode, got.Reason}, []string{"error", "git rev-parse origin/main failed: fatal: ambiguous argument 'origin/main'"})
	testhelp.Equal(t, "example 7 variant calls", len(f.Calls), 4)
}

func TestProbePhaseStartCheckExample8(t *testing.T) {
	f := &runner.Fake{Errors: map[string]string{"git fetch -q origin": "runner: exec: \"git\": executable file not found"}}
	got := StartCheck(context.Background(), f, "/p")
	testhelp.Equal(t, "example 8 Mode, Reason", []string{got.Mode, got.Reason}, []string{"error", "git fetch -q origin failed: runner: exec: \"git\": executable file not found"})
	testhelp.Equal(t, "example 8 calls", len(f.Calls), 1)
	// variant: a Run error of the status read
	f = pSCFake("main", "aaa111", "aaa111", "")
	f.Errors = map[string]string{"git status --porcelain": "context canceled"}
	got = StartCheck(context.Background(), f, "/p")
	testhelp.Equal(t, "example 8 variant Mode, Reason", []string{got.Mode, got.Reason}, []string{"error", "git status --porcelain failed: context canceled"})
}
