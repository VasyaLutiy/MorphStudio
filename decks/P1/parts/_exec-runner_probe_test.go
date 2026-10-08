package runner

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"morphstudio/internal/testhelp"
)

func TestProbeExecRunnerExample1(t *testing.T) {
	var r Runner = OS{}
	got, err := r.Run(context.Background(), t.TempDir(), nil, "sh", "-c", "printf hi; printf err >&2; exit 3")
	testhelp.Equal(t, "example 1 error (a non-zero exit is not an error)", err, error(nil))
	testhelp.Equal(t, "example 1 Result", got, Result{Stdout: "hi", Stderr: "err", Code: 3})
	got, err = OS{}.Run(context.Background(), t.TempDir(), nil, "sh", "-c", "printf ok; exit 0")
	testhelp.Equal(t, "example 1 exit 0 error", err, error(nil))
	testhelp.Equal(t, "example 1 exit 0 Result", got, Result{Stdout: "ok", Code: 0})
}

func TestProbeExecRunnerExample2(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "f.txt"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := OS{}.Run(context.Background(), dir, nil, "sh", "-c", "ls")
	testhelp.Equal(t, "example 2 error", err, error(nil))
	testhelp.Equal(t, "example 2 Result (runs in dir)", got, Result{Stdout: "f.txt\n", Code: 0})
}

func TestProbeExecRunnerExample3(t *testing.T) {
	got, err := OS{}.Run(context.Background(), t.TempDir(), []string{"MORPH_X=7"}, "sh", "-c", "echo $MORPH_X$HOME")
	testhelp.Equal(t, "example 3 error", err, error(nil))
	testhelp.Equal(t, "example 3 Stdout (parent env kept, entries appended)", got.Stdout, "7"+os.Getenv("HOME")+"\n")
	got, _ = OS{}.Run(context.Background(), t.TempDir(), []string{"MORPH_X=8", "MORPH_Y=y"}, "sh", "-c", "echo $MORPH_X$MORPH_Y")
	testhelp.Equal(t, "example 3 two entries Stdout", got.Stdout, "8y\n")
}

func TestProbeExecRunnerExample4(t *testing.T) {
	for _, c := range []struct {
		what, dir, name string
		args            []string
	}{
		{"missing binary", t.TempDir(), "/nonexistent/bin/x", nil},
		{"missing dir", "/nonexistent/dir", "sh", []string{"-c", "true"}},
	} {
		got, err := OS{}.Run(context.Background(), c.dir, nil, c.name, c.args...)
		testhelp.Equal(t, "example 4 "+c.what+" Code", got.Code, -1)
		if err == nil {
			t.Errorf("example 4 %s: error nil, want one beginning \"runner: \"", c.what)
		} else {
			testhelp.Equal(t, "example 4 "+c.what+" error begins \"runner: \"", strings.HasPrefix(err.Error(), "runner: "), true)
		}
	}
}

func TestProbeExecRunnerExample5(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()
	start := time.Now()
	got, err := OS{}.Run(ctx, t.TempDir(), nil, "sh", "-c", "sleep 5")
	took := time.Since(start)
	testhelp.Equal(t, "example 5 returns within 2 s", took < 2*time.Second, true)
	testhelp.Equal(t, "example 5 Code", got.Code, -1)
	testhelp.Equal(t, "example 5 errors.Is(err, context.DeadlineExceeded)", errors.Is(err, context.DeadlineExceeded), true)
}

func TestProbeExecRunnerExample6(t *testing.T) {
	ctx := context.Background()
	f := &Fake{Script: map[string]Result{"git rev-parse HEAD": {Stdout: "abc\n"}}, Errors: map[string]string{"git fetch -q origin": "runner: fake offline"}, Default: Result{Stdout: "default\n", Code: 4}}
	var r Runner = f
	got, err := r.Run(ctx, "/p", nil, "git", "rev-parse", "HEAD")
	testhelp.Equal(t, "example 6 scripted Result", got, Result{Stdout: "abc\n", Code: 0})
	testhelp.Equal(t, "example 6 scripted error", err, error(nil))
	got, err = f.Run(ctx, "/p", nil, "git", "fetch", "-q", "origin")
	testhelp.Equal(t, "example 6 error Result", got, Result{Code: -1})
	if err == nil {
		t.Errorf("example 6 Errors entry: error nil, want \"runner: fake offline\"")
	} else {
		testhelp.Equal(t, "example 6 error text", err.Error(), "runner: fake offline")
	}
	got, err = f.Run(ctx, "/q", nil, "git", "status")
	testhelp.Equal(t, "example 6 Default Result", got, Result{Stdout: "default\n", Code: 4})
	testhelp.Equal(t, "example 6 Default error", err, error(nil))
	testhelp.Equal(t, "example 6 Calls", f.Calls, []Call{{Dir: "/p", Name: "git", Args: []string{"rev-parse", "HEAD"}}, {Dir: "/p", Name: "git", Args: []string{"fetch", "-q", "origin"}}, {Dir: "/q", Name: "git", Args: []string{"status"}}})
}

func TestProbeExecRunnerExample7(t *testing.T) {
	testhelp.Equal(t, "example 7 Key(git rev-parse HEAD)", Key("git", "rev-parse", "HEAD"), "git rev-parse HEAD")
	testhelp.Equal(t, "example 7 Key(morph)", Key("morph"), "morph")
	testhelp.Equal(t, "example 7 Key(morph plan --judge)", Key("morph", "plan", "--judge"), "morph plan --judge")
}
