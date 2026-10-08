package bootstrap

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"morphstudio/internal/testhelp"
	"morphstudio/runner"
)

type envFake struct {
	*runner.Fake
	envs [][]string
}

func (e *envFake) Run(ctx context.Context, dir string, env []string, name string, args ...string) (runner.Result, error) {
	e.envs = append(e.envs, append([]string(nil), env...))
	return e.Fake.Run(ctx, dir, env, name, args...)
}

func TestProjectCreateExample1(t *testing.T) {
	ctx := context.Background()
	projectsDir := t.TempDir()
	f := &runner.Fake{}
	spec := Spec{
		Name:            "demo",
		Language:        "go",
		RepoURL:         "https://github.com/o/demo",
		ProjectsDir:     projectsDir,
		CredentialsFile: "/st/demo/git-credentials",
		MorphBin:        "/usr/local/bin/morph",
	}
	rep, err := Create(ctx, f, spec)
	testhelp.Equal(t, "err", err, nil)
	dir := filepath.Join(projectsDir, "demo")
	testhelp.Equal(t, "dir", rep.Dir, dir)
	wantSteps := []string{
		"/usr/local/bin/morph init --root " + dir + " --name demo --language go --module demo",
		"git init -q -b main",
		"git config credential.helper store --file=/st/demo/git-credentials",
		"git config user.name morphd",
		"git config user.email morphd@localhost",
		"git remote add origin https://github.com/o/demo",
		"git add -A",
		"git commit -q -m morph init: demo (go)",
	}
	testhelp.Equal(t, "steps", rep.Steps, wantSteps)
	if testhelp.Equal(t, "len calls", len(f.Calls), 8) {
		testhelp.Equal(t, "call 0 dir", f.Calls[0].Dir, projectsDir)
		for i := 1; i < len(f.Calls); i++ {
			testhelp.Equal(t, "call dir", f.Calls[i].Dir, dir)
		}
		testhelp.Equal(t, "call 2 args", f.Calls[2].Args, []string{"config", "credential.helper", "store --file=/st/demo/git-credentials"})
	}
}

func TestProjectCreateExample2(t *testing.T) {
	ctx := context.Background()
	cases := []struct {
		name    string
		spec    Spec
		wantErr string
	}{
		{
			name:    "bad name",
			spec:    Spec{Name: "Demo"},
			wantErr: `bootstrap: bad project name "Demo"`,
		},
		{
			name:    "bad language",
			spec:    Spec{Name: "demo", Language: "rust"},
			wantErr: `bootstrap: unknown language "rust"`,
		},
		{
			name:    "bad repo",
			spec:    Spec{Name: "demo", Language: "go", RepoURL: "https://gitlab.com/o/r"},
			wantErr: "github: not a github repository url: https://gitlab.com/o/r",
		},
		{
			name:    "no morph",
			spec:    Spec{Name: "demo", Language: "go", RepoURL: "https://github.com/o/demo", ProjectsDir: t.TempDir()},
			wantErr: "bootstrap: morph binary not configured",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			f := &runner.Fake{}
			_, err := Create(ctx, f, tc.spec)
			if err == nil {
				t.Fatal("expected error")
			}
			testhelp.Equal(t, "error", err.Error(), tc.wantErr)
			testhelp.Equal(t, "calls", len(f.Calls), 0)
		})
	}
}

func TestProjectCreateExample3(t *testing.T) {
	ctx := context.Background()
	projectsDir := t.TempDir()
	dir := filepath.Join(projectsDir, "demo")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		t.Fatal(err)
	}
	f := &runner.Fake{}
	spec := Spec{
		Name:            "demo",
		Language:        "go",
		RepoURL:         "https://github.com/o/demo",
		ProjectsDir:     projectsDir,
		CredentialsFile: "/st/demo/git-credentials",
		MorphBin:        "morph",
	}
	_, err := Create(ctx, f, spec)
	if err == nil {
		t.Fatal("expected error")
	}
	testhelp.Equal(t, "error", err.Error(), "bootstrap: directory exists: "+dir)
	testhelp.Equal(t, "calls", len(f.Calls), 0)
}

func TestProjectCreateExample4(t *testing.T) {
	ctx := context.Background()
	projectsDir := t.TempDir()
	dir := filepath.Join(projectsDir, "demo")
	f := &runner.Fake{Script: map[string]runner.Result{
		"git commit -q -m morph init: demo (go)": {Code: 1, Stderr: "nothing to commit\n"},
	}}
	spec := Spec{
		Name:            "demo",
		Language:        "go",
		RepoURL:         "https://github.com/o/demo",
		ProjectsDir:     projectsDir,
		CredentialsFile: "/st/demo/git-credentials",
		MorphBin:        "morph",
	}
	rep, err := Create(ctx, f, spec)
	if err == nil {
		t.Fatal("expected error")
	}
	testhelp.Equal(t, "error", err.Error(), "bootstrap: step 8 failed (exit 1): nothing to commit")
	testhelp.Equal(t, "steps len", len(rep.Steps), 8)
	testhelp.Equal(t, "dir", rep.Dir, dir)
}

func TestProjectCreateExample5(t *testing.T) {
	ctx := context.Background()
	revParseKey := "git rev-parse --verify -q origin/main"
	pushKey := "git push -q -u origin main"

	f1 := &runner.Fake{Script: map[string]runner.Result{
		revParseKey: {Code: 1},
		pushKey:     {Code: 0},
	}}
	w1 := &envFake{Fake: f1}
	pushed, err := FirstPush(ctx, w1, "/p/demo")
	testhelp.Equal(t, "pushed", pushed, true)
	testhelp.Equal(t, "err", err, nil)
	testhelp.Equal(t, "calls len", len(f1.Calls), 2)
	if len(f1.Calls) == 2 {
		testhelp.Equal(t, "call 0 key", runner.Key(f1.Calls[0].Name, f1.Calls[0].Args...), revParseKey)
		testhelp.Equal(t, "call 1 key", runner.Key(f1.Calls[1].Name, f1.Calls[1].Args...), pushKey)
		testhelp.Equal(t, "call 0 dir", f1.Calls[0].Dir, "/p/demo")
		testhelp.Equal(t, "call 1 dir", f1.Calls[1].Dir, "/p/demo")
	}
	testhelp.Equal(t, "envs len", len(w1.envs), 2)
	for i := range w1.envs {
		testhelp.Equal(t, "env", w1.envs[i], []string{"GIT_TERMINAL_PROMPT=0"})
	}

	f2 := &runner.Fake{Script: map[string]runner.Result{
		revParseKey: {Stdout: "aaa111\n", Code: 0},
	}}
	w2 := &envFake{Fake: f2}
	pushed, err = FirstPush(ctx, w2, "/p/demo")
	testhelp.Equal(t, "pushed", pushed, false)
	testhelp.Equal(t, "err", err, nil)
	testhelp.Equal(t, "calls len", len(f2.Calls), 1)
	if len(f2.Calls) == 1 {
		testhelp.Equal(t, "call key", runner.Key(f2.Calls[0].Name, f2.Calls[0].Args...), revParseKey)
		testhelp.Equal(t, "call dir", f2.Calls[0].Dir, "/p/demo")
	}

	f3 := &runner.Fake{Script: map[string]runner.Result{
		revParseKey: {Code: 1},
		pushKey:     {Code: 1, Stderr: "error: failed to push some refs\n"},
	}}
	w3 := &envFake{Fake: f3}
	pushed, err = FirstPush(ctx, w3, "/p/demo")
	testhelp.Equal(t, "pushed", pushed, false)
	if err == nil {
		t.Fatal("expected error")
	}
	testhelp.Equal(t, "error", err.Error(), "bootstrap: push failed (exit 1): error: failed to push some refs")
}

func TestProjectCreateExample6(t *testing.T) {
	ctx := context.Background()
	projectsDir := t.TempDir()
	dir := filepath.Join(projectsDir, "demo")
	initKey := "morph init --root " + dir + " --name demo --language python"
	f := &runner.Fake{Errors: map[string]string{
		initKey: `runner: exec: "morph": executable file not found`,
	}}
	spec := Spec{
		Name:            "demo",
		Language:        "python",
		RepoURL:         "https://github.com/o/demo",
		ProjectsDir:     projectsDir,
		CredentialsFile: "/st/demo/git-credentials",
		MorphBin:        "morph",
	}
	rep, err := Create(ctx, f, spec)
	if err == nil {
		t.Fatal("expected error")
	}
	testhelp.Equal(t, "error", err.Error(), `bootstrap: step 1: runner: exec: "morph": executable file not found`)
	testhelp.Equal(t, "steps len", len(rep.Steps), 1)
	if len(rep.Steps) == 1 {
		testhelp.Equal(t, "step", rep.Steps[0], initKey)
	}
	testhelp.Equal(t, "calls len", len(f.Calls), 1)
	if len(f.Calls) == 1 {
		testhelp.Equal(t, "call name", f.Calls[0].Name, "morph")
	}
}

func TestProjectCreateExample7(t *testing.T) {
	ctx := context.Background()
	projectsDir := t.TempDir()
	dir := filepath.Join(projectsDir, "a-1-b")
	f := &runner.Fake{}
	w := &envFake{Fake: f}
	spec := Spec{
		Name:            "a-1-b",
		Language:        "typescript",
		RepoURL:         "https://github.com/o/a-1-b",
		ProjectsDir:     projectsDir,
		CredentialsFile: "/st/a-1-b/git-credentials",
		MorphBin:        "morph",
	}
	rep, err := Create(ctx, w, spec)
	testhelp.Equal(t, "err", err, nil)
	testhelp.Equal(t, "steps len", len(rep.Steps), 8)
	if len(rep.Steps) == 8 {
		testhelp.Equal(t, "step 0", rep.Steps[0], "morph init --root "+dir+" --name a-1-b --language typescript")
		testhelp.Equal(t, "step 7", rep.Steps[7], "git commit -q -m morph init: a-1-b (typescript)")
	}
	testhelp.Equal(t, "envs len", len(w.envs), 8)
	for i := range w.envs {
		testhelp.Equal(t, "env", w.envs[i], []string{"GIT_TERMINAL_PROMPT=0"})
	}
}
