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

// TestExecRunnerExample1 covers example 1: a non-zero exit is a Result, not an error.
func TestExecRunnerExample1(t *testing.T) {
	res, err := OS{}.Run(context.Background(), t.TempDir(), nil, "sh", "-c", "printf hi; printf err >&2; exit 3")
	if !testhelp.Equal(t, "error", err, nil) {
		return
	}
	testhelp.Equal(t, "result", res, Result{Stdout: "hi", Stderr: "err", Code: 3})
}

// TestExecRunnerExample2 covers example 2: the command runs in dir.
func TestExecRunnerExample2(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "f.txt"), []byte("x\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	res, err := OS{}.Run(context.Background(), dir, nil, "sh", "-c", "ls")
	if !testhelp.Equal(t, "error", err, nil) {
		return
	}
	testhelp.Equal(t, "result", res, Result{Stdout: "f.txt\n", Code: 0})
}

// TestExecRunnerExample3 covers example 3: env entries are appended to the parent environment.
func TestExecRunnerExample3(t *testing.T) {
	res, err := OS{}.Run(context.Background(), t.TempDir(), []string{"MORPH_X=7"}, "sh", "-c", "echo $MORPH_X$HOME")
	if !testhelp.Equal(t, "error", err, nil) {
		return
	}
	testhelp.Equal(t, "result", res, Result{Stdout: "7" + os.Getenv("HOME") + "\n", Code: 0})
}

// TestExecRunnerExample4 covers example 4: commands that cannot start return Code -1 and a "runner: " error.
func TestExecRunnerExample4(t *testing.T) {
	res1, err1 := OS{}.Run(context.Background(), t.TempDir(), nil, "/nonexistent/bin/x")
	testhelp.Equal(t, "missing program code", res1.Code, -1)
	hasPrefix := err1 != nil && strings.HasPrefix(err1.Error(), "runner: ")
	testhelp.Equal(t, "missing program error begins runner: ", hasPrefix, true)

	res2, err2 := OS{}.Run(context.Background(), "/nonexistent/dir", nil, "sh", "-c", "true")
	testhelp.Equal(t, "missing directory code", res2.Code, -1)
	hasPrefix = err2 != nil && strings.HasPrefix(err2.Error(), "runner: ")
	testhelp.Equal(t, "missing directory error begins runner: ", hasPrefix, true)
}

// TestExecRunnerExample5 covers example 5: a cancelled context kills the process and returns the context error.
func TestExecRunnerExample5(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 200*time.Millisecond)
	defer cancel()

	start := time.Now()
	res, err := OS{}.Run(ctx, t.TempDir(), nil, "sh", "-c", "sleep 5")
	elapsed := time.Since(start)

	testhelp.Equal(t, "code", res.Code, -1)
	testhelp.Equal(t, "deadline exceeded", errors.Is(err, context.DeadlineExceeded), true)
	testhelp.Equal(t, "returns within 2s", elapsed < 2*time.Second, true)
}

// TestExecRunnerExample6 covers example 6: Fake returns scripted results, scripted errors, then the default, recording every call.
func TestExecRunnerExample6(t *testing.T) {
	f := &Fake{
		Script:  map[string]Result{"git rev-parse HEAD": {Stdout: "abc\n"}},
		Errors:  map[string]string{"git fetch -q origin": "runner: fake offline"},
		Default: Result{Stdout: "default\n", Code: 4},
	}
	ctx := context.Background()

	res1, err1 := f.Run(ctx, "/p", nil, "git", "rev-parse", "HEAD")
	testhelp.Equal(t, "scripted error", err1, nil)
	testhelp.Equal(t, "scripted result", res1, Result{Stdout: "abc\n", Code: 0})

	res2, err2 := f.Run(ctx, "/p", nil, "git", "fetch", "-q", "origin")
	testhelp.Equal(t, "scripted error code", res2.Code, -1)
	if err2 == nil {
		t.Fatal("scripted error: got nil, want non-nil")
	}
	testhelp.Equal(t, "scripted error text", err2.Error(), "runner: fake offline")

	res3, err3 := f.Run(ctx, "/q", nil, "git", "status")
	testhelp.Equal(t, "default error", err3, nil)
	testhelp.Equal(t, "default result", res3, Result{Stdout: "default\n", Code: 4})

	testhelp.Equal(t, "calls", f.Calls, []Call{
		{Dir: "/p", Name: "git", Args: []string{"rev-parse", "HEAD"}},
		{Dir: "/p", Name: "git", Args: []string{"fetch", "-q", "origin"}},
		{Dir: "/q", Name: "git", Args: []string{"status"}},
	})
}

// TestExecRunnerExample7 covers example 7: Key joins name and args with single spaces.
func TestExecRunnerExample7(t *testing.T) {
	testhelp.Equal(t, "key with args", Key("git", "rev-parse", "HEAD"), "git rev-parse HEAD")
	testhelp.Equal(t, "key without args", Key("morph"), "morph")
}
