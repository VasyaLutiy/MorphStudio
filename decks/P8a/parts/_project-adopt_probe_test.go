package bootstrap

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"morphstudio/internal/testhelp"
	"morphstudio/runner"
)

// pPARun records the env of every call and delegates to the Fake.
type pPARun struct {
	f    *runner.Fake
	envs [][]string
}

func (r *pPARun) Run(ctx context.Context, dir string, env []string, name string, args ...string) (runner.Result, error) {
	r.envs = append(r.envs, append([]string(nil), env...))
	return r.f.Run(ctx, dir, env, name, args...)
}

func pPAKeys(f *runner.Fake) []string {
	keys := []string{}
	for _, c := range f.Calls {
		keys = append(keys, runner.Key(c.Name, c.Args...))
	}
	return keys
}

// pPAErr checks the whole error text and whether it wraps ErrRefused.
func pPAErr(t *testing.T, what string, err error, want string, refused bool) {
	t.Helper()
	if err == nil {
		t.Errorf("%s: error nil, want %q", what, want)
		return
	}
	testhelp.Equal(t, what+" text", err.Error(), want)
	testhelp.Equal(t, what+" errors.Is ErrRefused", errors.Is(err, ErrRefused), refused)
}

func pPADir(t *testing.T) string {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "contour.yaml"), []byte("version: 1\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	return dir
}

func pPAScript() map[string]runner.Result {
	return map[string]runner.Result{
		"git rev-parse --is-inside-work-tree": {Stdout: "true\n"},
		"git rev-parse --abbrev-ref HEAD":     {Stdout: "main\n"},
		"git remote get-url origin":           {Stdout: "https://github.com/o/demo.git\n"},
	}
}

func pPASpec(dir string) AdoptSpec {
	return AdoptSpec{Name: "demo", Language: "go", RepoURL: "https://github.com/o/demo", Dir: dir, CredentialsFile: "/st/demo/git-credentials"}
}

var pPAFour = []string{"git rev-parse --is-inside-work-tree", "git rev-parse --abbrev-ref HEAD", "git remote get-url origin", "git config credential.helper store --file=/st/demo/git-credentials"}

func TestProbeProjectAdoptExample1(t *testing.T) {
	dir := pPADir(t)
	f := &runner.Fake{Script: pPAScript()}
	r := &pPARun{f: f}
	rep, err := Adopt(context.Background(), r, pPASpec(dir))
	testhelp.Equal(t, "example 1 error", err, nil)
	testhelp.Equal(t, "example 1 report", rep, Report{Dir: dir, Steps: pPAFour})
	testhelp.Equal(t, "example 1 call keys", pPAKeys(f), pPAFour)
	for i, c := range f.Calls {
		testhelp.Equal(t, "example 1 call dir", c.Dir, dir)
		testhelp.Equal(t, "example 1 call env", r.envs[i], []string{"GIT_TERMINAL_PROMPT=0"})
	}
	if len(f.Calls) == 4 {
		testhelp.Equal(t, "example 1 helper args", f.Calls[3].Args, []string{"config", "credential.helper", "store --file=/st/demo/git-credentials"})
	}
	for _, k := range pPAKeys(f) {
		for _, bad := range []string{"morph", "init", "commit", "push", "user.name"} {
			if strings.Contains(k, bad) {
				t.Errorf("example 1: call %q contains %q", k, bad)
			}
		}
	}
	// variant: the Report of an adopted repository is not created anew: nothing written into dir
	ents, _ := os.ReadDir(dir)
	testhelp.Equal(t, "example 1 variant dir untouched", len(ents), 1)
}

func TestProbeProjectAdoptExample2(t *testing.T) {
	dir := pPADir(t)
	missing := filepath.Join(t.TempDir(), "missing")
	empty := t.TempDir()
	cases := []struct {
		spec AdoptSpec
		want string
	}{
		{AdoptSpec{Name: "Demo", Language: "go", RepoURL: "https://github.com/o/demo", Dir: dir}, "bootstrap: refused: bad project name \"Demo\""},
		{AdoptSpec{Name: "demo", Language: "rust", RepoURL: "https://github.com/o/demo", Dir: dir}, "bootstrap: refused: unknown language \"rust\""},
		{AdoptSpec{Name: "demo", Language: "go", RepoURL: "https://gitlab.com/o/r", Dir: dir}, "bootstrap: refused: github: not a github repository url: https://gitlab.com/o/r"},
		{AdoptSpec{Name: "demo", Language: "go", RepoURL: "https://github.com/o/demo", Dir: "relative/demo"}, "bootstrap: refused: dir must be absolute: relative/demo"},
		{AdoptSpec{Name: "demo", Language: "go", RepoURL: "https://github.com/o/demo", Dir: missing}, "bootstrap: refused: not a directory: " + missing},
		{AdoptSpec{Name: "demo", Language: "go", RepoURL: "https://github.com/o/demo", Dir: empty}, "bootstrap: refused: no contour.yaml in " + empty},
	}
	for i, c := range cases {
		f := &runner.Fake{Script: pPAScript()}
		rep, err := Adopt(context.Background(), f, c.spec)
		what := "example 2 case " + string(rune('1'+i))
		pPAErr(t, what, err, c.want, true)
		testhelp.Equal(t, what+" report empty", rep, Report{})
		testhelp.Equal(t, what+" no calls", len(f.Calls), 0)
	}
	// variant: a regular file as Dir is not a directory
	file := filepath.Join(dir, "contour.yaml")
	f := &runner.Fake{Script: pPAScript()}
	_, err := Adopt(context.Background(), f, AdoptSpec{Name: "demo", Language: "go", RepoURL: "https://github.com/o/demo", Dir: file})
	pPAErr(t, "example 2 variant a file as Dir", err, "bootstrap: refused: not a directory: "+file, true)
	testhelp.Equal(t, "example 2 variant no calls", len(f.Calls), 0)
}

func TestProbeProjectAdoptExample3(t *testing.T) {
	dir := pPADir(t)
	f := &runner.Fake{Script: map[string]runner.Result{"git rev-parse --is-inside-work-tree": {Code: 128, Stderr: "fatal: not a git repository (or any of the parent directories): .git\n"}}}
	rep, err := Adopt(context.Background(), f, pPASpec(dir))
	pPAErr(t, "example 3", err, "bootstrap: refused: not a git repository: "+dir, true)
	testhelp.Equal(t, "example 3 steps", rep.Steps, []string{"git rev-parse --is-inside-work-tree"})
	testhelp.Equal(t, "example 3 calls", len(f.Calls), 1)
}

func TestProbeProjectAdoptExample4(t *testing.T) {
	dir := pPADir(t)
	s := pPAScript()
	s["git rev-parse --abbrev-ref HEAD"] = runner.Result{Stdout: "morph/20261010-084526\n"}
	f := &runner.Fake{Script: s}
	rep, err := Adopt(context.Background(), f, pPASpec(dir))
	pPAErr(t, "example 4 branch", err, "bootstrap: refused: on branch morph/20261010-084526, not main", true)
	testhelp.Equal(t, "example 4 branch steps", len(rep.Steps), 2)
	s["git rev-parse --abbrev-ref HEAD"] = runner.Result{Code: 128, Stderr: "fatal: ambiguous argument\n"}
	f = &runner.Fake{Script: s}
	rep, err = Adopt(context.Background(), f, pPASpec(dir))
	pPAErr(t, "example 4 exit 128", err, "bootstrap: step 2 failed (exit 128): fatal: ambiguous argument", false)
	testhelp.Equal(t, "example 4 exit 128 steps", len(rep.Steps), 2)
}

func TestProbeProjectAdoptExample5(t *testing.T) {
	dir := pPADir(t)
	cases := []struct {
		res   runner.Result
		want  string
		steps int
	}{
		{runner.Result{Code: 2, Stderr: "error: No such remote 'origin'\n"}, "bootstrap: refused: no origin remote", 3},
		{runner.Result{Stdout: "git@github.com:o/other.git\n"}, "bootstrap: refused: origin git@github.com:o/other.git is not https://github.com/o/demo", 3},
		{runner.Result{Stdout: "git@github.com:o/demo.git\n"}, "", 4},
		{runner.Result{Stdout: "https://github.com/o/demo/\n"}, "", 4},
		{runner.Result{Stdout: "https://example.com/x\n"}, "bootstrap: refused: origin https://example.com/x is not https://github.com/o/demo", 3},
	}
	for i, c := range cases {
		s := pPAScript()
		s["git remote get-url origin"] = c.res
		f := &runner.Fake{Script: s}
		rep, err := Adopt(context.Background(), f, pPASpec(dir))
		what := "example 5 case " + string(rune('1'+i))
		if c.want == "" {
			testhelp.Equal(t, what+" error", err, nil)
		} else {
			pPAErr(t, what, err, c.want, true)
		}
		testhelp.Equal(t, what+" steps", len(rep.Steps), c.steps)
	}
}

func TestProbeProjectAdoptExample6(t *testing.T) {
	dir := pPADir(t)
	f := &runner.Fake{Errors: map[string]string{"git rev-parse --is-inside-work-tree": "runner: exec: \"git\": executable file not found"}}
	rep, err := Adopt(context.Background(), f, pPASpec(dir))
	pPAErr(t, "example 6 run error", err, "bootstrap: step 1: runner: exec: \"git\": executable file not found", false)
	testhelp.Equal(t, "example 6 run error steps", len(rep.Steps), 1)
	s := pPAScript()
	s["git config credential.helper store --file=/st/demo/git-credentials"] = runner.Result{Code: 1, Stderr: "error: could not lock config file .git/config\n"}
	f = &runner.Fake{Script: s}
	rep, err = Adopt(context.Background(), f, pPASpec(dir))
	pPAErr(t, "example 6 step 4", err, "bootstrap: step 4 failed (exit 1): error: could not lock config file .git/config", false)
	testhelp.Equal(t, "example 6 step 4 steps", len(rep.Steps), 4)
	testhelp.Equal(t, "example 6 step 4 dir", rep.Dir, dir)
	// variant: a Run error of the origin read is a step error, not a refusal
	s = pPAScript()
	f = &runner.Fake{Script: s, Errors: map[string]string{"git remote get-url origin": "boom"}}
	_, err = Adopt(context.Background(), f, pPASpec(dir))
	pPAErr(t, "example 6 variant step 3 run error", err, "bootstrap: step 3: boom", false)
}

func TestProbeProjectAdoptExample7(t *testing.T) {
	dir := pPADir(t)
	s := pPAScript()
	s["git remote get-url origin"] = runner.Result{Stdout: "https://github.com/o/a-1-b/\n"}
	f := &runner.Fake{Script: s}
	r := &pPARun{f: f}
	rep, err := Adopt(context.Background(), r, AdoptSpec{Name: "a-1-b", Language: "typescript", RepoURL: "git@github.com:o/a-1-b.git", Dir: dir, CredentialsFile: "/var/morphd/a-1-b/git-credentials"})
	testhelp.Equal(t, "example 7 error", err, nil)
	if len(rep.Steps) == 4 {
		testhelp.Equal(t, "example 7 step 4", rep.Steps[3], "git config credential.helper store --file=/var/morphd/a-1-b/git-credentials")
	} else {
		t.Errorf("example 7: %d steps, want 4", len(rep.Steps))
	}
	testhelp.Equal(t, "example 7 dir", rep.Dir, dir)
	testhelp.Equal(t, "example 7 calls", len(f.Calls), 4)
	for _, e := range r.envs {
		testhelp.Equal(t, "example 7 env", e, []string{"GIT_TERMINAL_PROMPT=0"})
	}
}
