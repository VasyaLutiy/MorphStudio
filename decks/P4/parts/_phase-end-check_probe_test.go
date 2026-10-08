package gitrules

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"morphstudio/internal/testhelp"
	"morphstudio/runner"
)

// pECFake is example 1's fake with the given branch, HEAD, status and run-branch list (origin/main "aaa111").
func pECFake(branch, head, status, branches string) *runner.Fake {
	return &runner.Fake{Script: map[string]runner.Result{
		"git rev-parse --abbrev-ref HEAD":            {Stdout: branch + "\n"},
		"git rev-parse HEAD":                         {Stdout: head + "\n"},
		"git rev-parse origin/main":                  {Stdout: "aaa111\n"},
		"git status --porcelain":                     {Stdout: status},
		"git branch --list morph/* --no-merged main": {Stdout: branches},
	}}
}

// pECDir is a project dir; measure "" means no docs/MEASURE.md, "fixture" the copy of tests/fixtures/git/measure.md.
func pECDir(t *testing.T, measure string) string {
	t.Helper()
	dir := t.TempDir()
	if measure == "" {
		return dir
	}
	if measure == "fixture" {
		b, err := os.ReadFile("../tests/fixtures/git/measure.md")
		if err != nil {
			t.Fatalf("read the fixture: %v", err)
		}
		measure = string(b)
	}
	if err := os.MkdirAll(filepath.Join(dir, "docs"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "docs", "MEASURE.md"), []byte(measure), 0o644); err != nil {
		t.Fatal(err)
	}
	return dir
}

func pECKeys(f *runner.Fake) []string {
	out := []string{}
	for _, c := range f.Calls {
		out = append(out, runner.Key(c.Name, c.Args...))
	}
	return out
}

var pECSix = []string{"git fetch -q origin", "git rev-parse --abbrev-ref HEAD", "git rev-parse HEAD", "git rev-parse origin/main", "git status --porcelain", "git branch --list morph/* --no-merged main"}

func TestProbePhaseEndCheckExample1(t *testing.T) {
	dir := pECDir(t, "fixture")
	f := pECFake("main", "aaa111", "", "")
	got := EndCheck(context.Background(), f, dir, "P17")
	testhelp.Equal(t, "example 1 End", got, End{OK: true, Local: "aaa111", Remote: "aaa111", Problems: []string{}})
	testhelp.Equal(t, "example 1 Problems non-nil", got.Problems != nil, true)
	testhelp.Equal(t, "example 1 call keys", pECKeys(f), pECSix)
	if len(f.Calls) == 6 {
		testhelp.Equal(t, "example 1 Calls[5].Args", f.Calls[5].Args, []string{"branch", "--list", "morph/*", "--no-merged", "main"})
		testhelp.Equal(t, "example 1 Calls[5].Dir", f.Calls[5].Dir, dir)
	}
	// variant: the other row of the fixture
	got = EndCheck(context.Background(), pECFake("main", "aaa111", "", ""), dir, "P16")
	testhelp.Equal(t, "example 1 variant (P16) End", got, End{OK: true, Local: "aaa111", Remote: "aaa111", Problems: []string{}})
}

func TestProbePhaseEndCheckExample2(t *testing.T) {
	dir := pECDir(t, "fixture")
	got := EndCheck(context.Background(), pECFake("main", "bbb222", "", ""), dir, "P17")
	testhelp.Equal(t, "example 2 End", got, End{OK: false, Local: "bbb222", Remote: "aaa111", Problems: []string{"not pushed: HEAD bbb222, origin/main aaa111"}})
	// variant: another HEAD
	got = EndCheck(context.Background(), pECFake("main", "77c0ffee", "", ""), dir, "P17")
	testhelp.Equal(t, "example 2 variant Problems", got.Problems, []string{"not pushed: HEAD 77c0ffee, origin/main aaa111"})
}

func TestProbePhaseEndCheckExample3(t *testing.T) {
	dir := pECDir(t, "fixture")
	got := EndCheck(context.Background(), pECFake("main", "aaa111", "", ""), dir, "P18")
	testhelp.Equal(t, "example 3 End", got, End{OK: false, Local: "aaa111", Remote: "aaa111", Problems: []string{"no MEASURE row for P18"}})
	// variant: a part of a cell never matches
	for _, ph := range []string{"P1", "P7", "primer-go", "debt"} {
		got = EndCheck(context.Background(), pECFake("main", "aaa111", "", ""), dir, ph)
		testhelp.Equal(t, "example 3 variant Problems for "+ph, got.Problems, []string{"no MEASURE row for " + ph})
	}
}

func TestProbePhaseEndCheckExample4(t *testing.T) {
	dir := pECDir(t, "")
	f := pECFake("main", "bbb222", " M a.go\n", "  morph/20261008-141116\n* morph/20261008-152814\n")
	got := EndCheck(context.Background(), f, dir, "P17")
	testhelp.Equal(t, "example 4 Problems", got.Problems, []string{"not pushed: HEAD bbb222, origin/main aaa111", "dirty tree (1 paths)", "unmerged run branch morph/20261008-141116", "unmerged run branch morph/20261008-152814", "docs/MEASURE.md missing"})
	testhelp.Equal(t, "example 4 OK", got.OK, false)
	// variant: every problem at once, the branch first; three dirty paths, one run branch
	f = pECFake("morph/r-1", "ccc333", "?? x\n?? y\n M z\n", "  morph/r-1\n")
	got = EndCheck(context.Background(), f, pECDir(t, "fixture"), "P17")
	testhelp.Equal(t, "example 4 variant Problems", got.Problems, []string{"on branch morph/r-1, not main", "not pushed: HEAD ccc333, origin/main aaa111", "dirty tree (3 paths)", "unmerged run branch morph/r-1"})
}

func TestProbePhaseEndCheckExample5(t *testing.T) {
	dir := pECDir(t, "fixture")
	got := EndCheck(context.Background(), pECFake("morph/x", "aaa111", "", ""), dir, "P17")
	testhelp.Equal(t, "example 5 Problems", got.Problems, []string{"on branch morph/x, not main"})
	testhelp.Equal(t, "example 5 OK", got.OK, false)
}

func TestProbePhaseEndCheckExample6(t *testing.T) {
	f := &runner.Fake{Script: map[string]runner.Result{"git fetch -q origin": {Code: 1, Stderr: "fatal: not a git repository\n"}}}
	got := EndCheck(context.Background(), f, pECDir(t, "fixture"), "P17")
	testhelp.Equal(t, "example 6 End", got, End{OK: false, Local: "", Remote: "", Problems: []string{"git fetch -q origin failed: fatal: not a git repository"}})
	testhelp.Equal(t, "example 6 calls", len(f.Calls), 1)
	// variant: the run-branch list fails (command 6) by a Run error
	f = pECFake("main", "aaa111", "", "")
	f.Errors = map[string]string{"git branch --list morph/* --no-merged main": "runner: signal: killed"}
	got = EndCheck(context.Background(), f, pECDir(t, "fixture"), "P17")
	testhelp.Equal(t, "example 6 variant Problems", got.Problems, []string{"git branch --list morph/* --no-merged main failed: runner: signal: killed"})
	testhelp.Equal(t, "example 6 variant OK", got.OK, false)
	// variant: the status read exits 128; nothing after it runs
	f = pECFake("main", "aaa111", "", "")
	f.Script["git status --porcelain"] = runner.Result{Code: 128, Stderr: "fatal: index locked\n"}
	got = EndCheck(context.Background(), f, pECDir(t, "fixture"), "P17")
	testhelp.Equal(t, "example 6 variant (status) Problems", got.Problems, []string{"git status --porcelain failed: fatal: index locked"})
	testhelp.Equal(t, "example 6 variant (status) calls", len(f.Calls), 5)
}

func TestProbePhaseEndCheckExample7(t *testing.T) {
	dir := pECDir(t, "| phase | x |\n|---|---|\n| P17 | 8 / 8 |\n| P170 debt | 1 / 1 |\n")
	got := EndCheck(context.Background(), pECFake("main", "aaa111", "", ""), dir, "P17")
	testhelp.Equal(t, "example 7 End (exact cell)", got, End{OK: true, Local: "aaa111", Remote: "aaa111", Problems: []string{}})
	dir = pECDir(t, "| phase | x |\n|---|---|\n| P170 debt | 1 / 1 |\n")
	got = EndCheck(context.Background(), pECFake("main", "aaa111", "", ""), dir, "P17")
	testhelp.Equal(t, "example 7 only the P170 row", got.Problems, []string{"no MEASURE row for P17"})
	// variant: the row id in a padded cell, and a phase id given as P170
	dir = pECDir(t, "text\n|   P23   |\n| P170 debt | 1 |\n")
	got = EndCheck(context.Background(), pECFake("main", "aaa111", "", ""), dir, "P23")
	testhelp.Equal(t, "example 7 variant (padded cell) Problems", got.Problems, []string{})
	got = EndCheck(context.Background(), pECFake("main", "aaa111", "", ""), dir, "P170")
	testhelp.Equal(t, "example 7 variant (P170) Problems", got.Problems, []string{})
}
