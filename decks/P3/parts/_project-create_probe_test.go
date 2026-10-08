package bootstrap

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"morphstudio/internal/testhelp"
	"morphstudio/runner"
)

// pPCRun records the env of every call and delegates to the Fake.
type pPCRun struct {
	f    *runner.Fake
	envs [][]string
}

func (r *pPCRun) Run(ctx context.Context, dir string, env []string, name string, args ...string) (runner.Result, error) {
	r.envs = append(r.envs, append([]string(nil), env...))
	return r.f.Run(ctx, dir, env, name, args...)
}

func pPCKeys(f *runner.Fake) []string {
	keys := []string{}
	for _, c := range f.Calls {
		keys = append(keys, runner.Key(c.Name, c.Args...))
	}
	return keys
}

func pPCErr(t *testing.T, what string, err error, want string) {
	t.Helper()
	if err == nil {
		t.Errorf("%s: error nil, want %q", what, want)
		return
	}
	testhelp.Equal(t, what, err.Error(), want)
}

func pPCEnvs(t *testing.T, what string, r *pPCRun) {
	t.Helper()
	for i, e := range r.envs {
		testhelp.Equal(t, what+" env of call "+string(rune('1'+i)), e, []string{"GIT_TERMINAL_PROMPT=0"})
	}
}

func pPCSteps(morph, dir, name, lang, creds, repo string) []string {
	first := morph + " init --root " + dir + " --name " + name + " --language " + lang
	if lang == "go" {
		first += " --module " + name
	}
	return []string{first, "git init -q -b main", "git config credential.helper store --file=" + creds, "git config user.name morphd",
		"git config user.email morphd@localhost", "git remote add origin " + repo, "git add -A", "git commit -q -m morph init: " + name + " (" + lang + ")"}
}

func TestProbeProjectCreateExample1(t *testing.T) {
	pd := t.TempDir()
	f := &runner.Fake{}
	r := &pPCRun{f: f}
	rep, err := Create(context.Background(), r, Spec{Name: "demo", Language: "go", RepoURL: "https://github.com/o/demo", ProjectsDir: pd, CredentialsFile: "/st/demo/git-credentials", MorphBin: "/usr/local/bin/morph"})
	dir := filepath.Join(pd, "demo")
	testhelp.Equal(t, "example 1 error", err, error(nil))
	testhelp.Equal(t, "example 1 Report.Dir", rep.Dir, dir)
	want := pPCSteps("/usr/local/bin/morph", dir, "demo", "go", "/st/demo/git-credentials", "https://github.com/o/demo")
	testhelp.Equal(t, "example 1 Steps", rep.Steps, want)
	testhelp.Equal(t, "example 1 call keys", pPCKeys(f), want)
	if len(f.Calls) == 8 {
		testhelp.Equal(t, "example 1 Calls[0].Dir", f.Calls[0].Dir, pd)
		for i := 1; i < 8; i++ {
			testhelp.Equal(t, "example 1 Calls[1..7].Dir", f.Calls[i].Dir, dir)
		}
		testhelp.Equal(t, "example 1 Calls[2].Args", f.Calls[2].Args, []string{"config", "credential.helper", "store --file=/st/demo/git-credentials"})
		testhelp.Equal(t, "example 1 Calls[0].Name", f.Calls[0].Name, "/usr/local/bin/morph")
	}
	pPCEnvs(t, "example 1", r)
	// variant: another go project (the module is the name), other credentials and repository
	pd2 := t.TempDir()
	f2 := &runner.Fake{}
	rep, err = Create(context.Background(), f2, Spec{Name: "svc-9", Language: "go", RepoURL: "git@github.com:acme/svc-9.git", ProjectsDir: pd2, CredentialsFile: "/c/x", MorphBin: "m2"})
	testhelp.Equal(t, "example 1 variant error", err, error(nil))
	testhelp.Equal(t, "example 1 variant Steps", rep.Steps, pPCSteps("m2", filepath.Join(pd2, "svc-9"), "svc-9", "go", "/c/x", "git@github.com:acme/svc-9.git"))
}

func TestProbeProjectCreateExample2(t *testing.T) {
	ok := Spec{Name: "demo", Language: "go", RepoURL: "https://github.com/o/demo", ProjectsDir: t.TempDir(), CredentialsFile: "/c", MorphBin: "morph"}
	cases := []struct {
		s    Spec
		want string
	}{
		{Spec{Name: "Demo"}, `bootstrap: bad project name "Demo"`},
		{Spec{Name: "demo", Language: "rust"}, `bootstrap: unknown language "rust"`},
		{Spec{Name: "demo", Language: "go", RepoURL: "https://gitlab.com/o/r"}, "github: not a github repository url: https://gitlab.com/o/r"},
		{Spec{Name: "demo", Language: "go", RepoURL: "https://github.com/o/r"}, "bootstrap: morph binary not configured"},
	}
	// variants: the name rule's bounds; python is a language; an empty language
	long := strings.Repeat("a", 41)
	for _, v := range []struct{ name, want string }{{"-a", `bootstrap: bad project name "-a"`}, {long, `bootstrap: bad project name "` + long + `"`}, {"a_b", `bootstrap: bad project name "a_b"`}, {"", `bootstrap: bad project name ""`}} {
		s := ok
		s.Name = v.name
		cases = append(cases, struct {
			s    Spec
			want string
		}{s, v.want})
	}
	s := ok
	s.Language = ""
	cases = append(cases, struct {
		s    Spec
		want string
	}{s, `bootstrap: unknown language ""`})
	for i, c := range cases {
		f := &runner.Fake{}
		_, err := Create(context.Background(), f, c.s)
		pPCErr(t, "example 2 case "+string(rune('1'+i))+" error", err, c.want)
		testhelp.Equal(t, "example 2 case "+string(rune('1'+i))+" Calls", len(f.Calls), 0)
	}
	// variant: 40 characters and digits first are valid names
	for _, name := range []string{strings.Repeat("b", 40), "9x"} {
		s := ok
		s.Name = name
		s.ProjectsDir = t.TempDir()
		_, err := Create(context.Background(), &runner.Fake{}, s)
		testhelp.Equal(t, "example 2 variant valid name "+name, err, error(nil))
	}
}

func TestProbeProjectCreateExample3(t *testing.T) {
	pd := t.TempDir()
	dir := filepath.Join(pd, "demo")
	if err := os.Mkdir(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	f := &runner.Fake{}
	_, err := Create(context.Background(), f, Spec{Name: "demo", Language: "go", RepoURL: "https://github.com/o/demo", ProjectsDir: pd, CredentialsFile: "/c", MorphBin: "morph"})
	pPCErr(t, "example 3 error", err, "bootstrap: directory exists: "+dir)
	testhelp.Equal(t, "example 3 Calls", len(f.Calls), 0)
	// variant: a regular file of that name counts as existing too
	pd2 := t.TempDir()
	if err := os.WriteFile(filepath.Join(pd2, "p2"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err = Create(context.Background(), f, Spec{Name: "p2", Language: "python", RepoURL: "https://github.com/o/p2", ProjectsDir: pd2, CredentialsFile: "/c", MorphBin: "morph"})
	pPCErr(t, "example 3 variant error", err, "bootstrap: directory exists: "+filepath.Join(pd2, "p2"))
}

func TestProbeProjectCreateExample4(t *testing.T) {
	pd := t.TempDir()
	f := &runner.Fake{Script: map[string]runner.Result{"git commit -q -m morph init: demo (go)": {Code: 1, Stderr: "nothing to commit\n"}}}
	rep, err := Create(context.Background(), f, Spec{Name: "demo", Language: "go", RepoURL: "https://github.com/o/demo", ProjectsDir: pd, CredentialsFile: "/c", MorphBin: "morph"})
	pPCErr(t, "example 4 error", err, "bootstrap: step 8 failed (exit 1): nothing to commit")
	testhelp.Equal(t, "example 4 Steps count", len(rep.Steps), 8)
	testhelp.Equal(t, "example 4 Report.Dir", rep.Dir, filepath.Join(pd, "demo"))
	// variant: step 3 fails with exit 128; the steps after it never run
	pd2 := t.TempDir()
	f2 := &runner.Fake{Script: map[string]runner.Result{"git config credential.helper store --file=/k/c": {Code: 128, Stderr: "  fatal: bad config  \n"}}}
	rep, err = Create(context.Background(), f2, Spec{Name: "q", Language: "typescript", RepoURL: "https://github.com/o/q", ProjectsDir: pd2, CredentialsFile: "/k/c", MorphBin: "morph"})
	pPCErr(t, "example 4 variant error (step 3)", err, "bootstrap: step 3 failed (exit 128): fatal: bad config")
	testhelp.Equal(t, "example 4 variant Steps (step 3)", rep.Steps, pPCSteps("morph", filepath.Join(pd2, "q"), "q", "typescript", "/k/c", "https://github.com/o/q")[:3])
	testhelp.Equal(t, "example 4 variant Calls (step 3)", len(f2.Calls), 3)
}

func TestProbeProjectCreateExample5(t *testing.T) {
	f := &runner.Fake{Script: map[string]runner.Result{"git rev-parse --verify -q origin/main": {Code: 1}, "git push -q -u origin main": {Code: 0}}}
	r := &pPCRun{f: f}
	pushed, err := FirstPush(context.Background(), r, "/p/demo")
	testhelp.Equal(t, "example 5 FirstPush", pushed, true)
	testhelp.Equal(t, "example 5 error", err, error(nil))
	testhelp.Equal(t, "example 5 call keys", pPCKeys(f), []string{"git rev-parse --verify -q origin/main", "git push -q -u origin main"})
	for _, c := range f.Calls {
		testhelp.Equal(t, "example 5 Dir", c.Dir, "/p/demo")
	}
	pPCEnvs(t, "example 5", r)
	f2 := &runner.Fake{Script: map[string]runner.Result{"git rev-parse --verify -q origin/main": {Stdout: "aaa111\n", Code: 0}}}
	pushed, err = FirstPush(context.Background(), f2, "/p/demo")
	testhelp.Equal(t, "example 5 already pushed", pushed, false)
	testhelp.Equal(t, "example 5 already pushed error", err, error(nil))
	testhelp.Equal(t, "example 5 already pushed calls", len(f2.Calls), 1)
	f3 := &runner.Fake{Script: map[string]runner.Result{"git rev-parse --verify -q origin/main": {Code: 1}, "git push -q -u origin main": {Code: 1, Stderr: "error: failed to push some refs\n"}}}
	pushed, err = FirstPush(context.Background(), f3, "/p/demo")
	testhelp.Equal(t, "example 5 push failed", pushed, false)
	pPCErr(t, "example 5 push failed error", err, "bootstrap: push failed (exit 1): error: failed to push some refs")
	// variants: exit 128 in another dir; a Run error of the push
	f4 := &runner.Fake{Script: map[string]runner.Result{"git rev-parse --verify -q origin/main": {Code: 1}, "git push -q -u origin main": {Code: 128, Stderr: "\nfatal: no access\n"}}}
	pushed, err = FirstPush(context.Background(), f4, "/q/x")
	testhelp.Equal(t, "example 5 variant pushed", pushed, false)
	pPCErr(t, "example 5 variant error (exit 128)", err, "bootstrap: push failed (exit 128): fatal: no access")
	if len(f4.Calls) > 0 {
		testhelp.Equal(t, "example 5 variant Dir", f4.Calls[0].Dir, "/q/x")
	}
	f5 := &runner.Fake{Script: map[string]runner.Result{"git rev-parse --verify -q origin/main": {Code: 1}}, Errors: map[string]string{"git push -q -u origin main": "runner: exec: \"git\": not found"}}
	pushed, err = FirstPush(context.Background(), f5, "/q/x")
	testhelp.Equal(t, "example 5 variant pushed (run error)", pushed, false)
	pPCErr(t, "example 5 variant error (run error)", err, "bootstrap: push: runner: exec: \"git\": not found")
}

func TestProbeProjectCreateExample6(t *testing.T) {
	pd := t.TempDir()
	dir := filepath.Join(pd, "demo")
	f := &runner.Fake{Errors: map[string]string{"morph init --root " + dir + " --name demo --language python": "runner: exec: \"morph\": executable file not found"}}
	rep, err := Create(context.Background(), f, Spec{Name: "demo", Language: "python", RepoURL: "https://github.com/o/demo", ProjectsDir: pd, CredentialsFile: "/c", MorphBin: "morph"})
	pPCErr(t, "example 6 error", err, "bootstrap: step 1: runner: exec: \"morph\": executable file not found")
	testhelp.Equal(t, "example 6 Steps count", len(rep.Steps), 1)
	testhelp.Equal(t, "example 6 calls", len(f.Calls), 1)
	// variant: a Run error at step 6
	pd2 := t.TempDir()
	f2 := &runner.Fake{Errors: map[string]string{"git remote add origin https://github.com/o/w": "boom"}}
	rep, err = Create(context.Background(), f2, Spec{Name: "w", Language: "go", RepoURL: "https://github.com/o/w", ProjectsDir: pd2, CredentialsFile: "/c", MorphBin: "morph"})
	pPCErr(t, "example 6 variant error (step 6)", err, "bootstrap: step 6: boom")
	testhelp.Equal(t, "example 6 variant Steps count (step 6)", len(rep.Steps), 6)
	testhelp.Equal(t, "example 6 variant Report.Dir", rep.Dir, filepath.Join(pd2, "w"))
}

func TestProbeProjectCreateExample7(t *testing.T) {
	pd := t.TempDir()
	f := &runner.Fake{}
	r := &pPCRun{f: f}
	rep, err := Create(context.Background(), r, Spec{Name: "a-1-b", Language: "typescript", RepoURL: "https://github.com/o/demo", ProjectsDir: pd, CredentialsFile: "/st/demo/git-credentials", MorphBin: "morph"})
	dir := filepath.Join(pd, "a-1-b")
	testhelp.Equal(t, "example 7 error", err, error(nil))
	if len(rep.Steps) == 8 {
		testhelp.Equal(t, "example 7 Steps[0]", rep.Steps[0], "morph init --root "+dir+" --name a-1-b --language typescript")
		testhelp.Equal(t, "example 7 Steps[7]", rep.Steps[7], "git commit -q -m morph init: a-1-b (typescript)")
	} else {
		t.Errorf("example 7 Steps: %d entries, want 8", len(rep.Steps))
	}
	testhelp.Equal(t, "example 7 env calls", len(r.envs), 8)
	pPCEnvs(t, "example 7", r)
	// variant: python has no --module either
	pd2 := t.TempDir()
	rep, err = Create(context.Background(), &runner.Fake{}, Spec{Name: "py", Language: "python", RepoURL: "https://github.com/o/py", ProjectsDir: pd2, CredentialsFile: "/c", MorphBin: "morph"})
	testhelp.Equal(t, "example 7 variant error", err, error(nil))
	if len(rep.Steps) > 0 {
		testhelp.Equal(t, "example 7 variant Steps[0] (python)", rep.Steps[0], "morph init --root "+filepath.Join(pd2, "py")+" --name py --language python")
	}
}
