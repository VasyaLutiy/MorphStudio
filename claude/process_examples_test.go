package claude

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"morphstudio/internal/testhelp"
)

const prTimeout = 5 * time.Second

// prScript writes body to name under a fresh temporary directory and makes it
// executable, returning the script's path.
func prScript(t *testing.T, name, body string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(path, []byte(body), 0o755); err != nil {
		t.Fatal(err)
	}
	return path
}

// prNext reads one line from ch, failing the test after prTimeout.
func prNext(t *testing.T, ch <-chan []byte) []byte {
	t.Helper()
	select {
	case line := <-ch:
		return line
	case <-time.After(prTimeout):
		t.Fatal("timeout waiting for line")
		return nil
	}
}

// prExit reads one ExitStatus from p, failing the test after prTimeout.
func prExit(t *testing.T, p Process) ExitStatus {
	t.Helper()
	select {
	case st := <-p.Exit():
		return st
	case <-time.After(prTimeout):
		t.Fatal("timeout waiting for exit")
		return ExitStatus{}
	}
}

// Process example 1: a script that prints an init line, echoes stdin, and exits
// 4; Start, one Write, the echoed line, Stop and the exit status.
func TestProcessExample1(t *testing.T) {
	bin := prScript(t, "claude",
		"#!/bin/sh\necho '{\"type\":\"system\",\"subtype\":\"init\"}'\ncat\nexit 4\n")
	p, err := Start(context.Background(), bin, nil, t.TempDir(), nil)
	if !testhelp.Equal(t, "err", err, nil) {
		return
	}
	first := prNext(t, p.Lines())
	testhelp.Equal(t, "first line", string(first), `{"type":"system","subtype":"init"}`)
	if err := p.Write([]byte("{\"type\":\"user\"}\n")); err != nil {
		t.Fatalf("write: %v", err)
	}
	next := prNext(t, p.Lines())
	testhelp.Equal(t, "next line", string(next), `{"type":"user"}`)
	if err := p.Stop(); err != nil {
		t.Fatalf("stop: %v", err)
	}
	st := prExit(t, p)
	testhelp.Equal(t, "exit", st, ExitStatus{Code: 4, Stderr: ""})
	select {
	case _, ok := <-p.Lines():
		if ok {
			t.Error("lines channel still open")
		}
	case <-time.After(prTimeout):
		t.Fatal("timeout waiting for lines to close")
	}
	if p.PID() <= 0 {
		t.Errorf("pid:\ngot:  %d\nwant: > 0", p.PID())
	}
}

// Process example 2: a script writing to stderr and exiting 2; the exit status
// carries the stderr tail and Write afterwards reports ErrExited.
func TestProcessExample2(t *testing.T) {
	bin := prScript(t, "claude", "#!/bin/sh\necho oops >&2\nexit 2\n")
	p, err := Start(context.Background(), bin, nil, t.TempDir(), nil)
	if !testhelp.Equal(t, "err", err, nil) {
		return
	}
	st := prExit(t, p)
	testhelp.Equal(t, "exit", st, ExitStatus{Code: 2, Stderr: "oops\n"})
	testhelp.Equal(t, "write after exit", p.Write([]byte("x\n")), ErrExited)
}

// Process example 3: a script holding stdout with a background sleep; Kill
// takes the whole process group down within 2 s.
func TestProcessExample3(t *testing.T) {
	bin := prScript(t, "claude", "#!/bin/sh\nsleep 30 &\necho started\nwait\n")
	p, err := Start(context.Background(), bin, nil, t.TempDir(), nil)
	if !testhelp.Equal(t, "err", err, nil) {
		return
	}
	first := prNext(t, p.Lines())
	testhelp.Equal(t, "first line", string(first), "started")
	if err := p.Kill(); err != nil {
		t.Fatalf("kill: %v", err)
	}
	select {
	case st := <-p.Exit():
		testhelp.Equal(t, "exit code", st.Code, -1)
	case <-time.After(2 * time.Second):
		t.Fatal("timeout waiting for exit after kill")
	}
}

// Process example 4: Start fails for a missing binary and for a missing
// working directory, each with an error beginning "claude: start ".
func TestProcessExample4(t *testing.T) {
	dir := t.TempDir()
	cases := []struct {
		bin string
		dir string
	}{
		{"/nonexistent/claude", dir},
		{"sh", "/nonexistent/dir"},
	}
	for _, tc := range cases {
		p, err := Start(context.Background(), tc.bin, nil, tc.dir, nil)
		testhelp.Equal(t, "process nil", p == nil, true)
		if err == nil || !strings.HasPrefix(err.Error(), "claude: start ") {
			t.Errorf("err:\ngot:  %#v\nwant: prefix %q", err, "claude: start ")
		}
	}
}

// Process example 5: a 70000-byte line arrives whole, longer than the default
// bufio buffer.
func TestProcessExample5(t *testing.T) {
	bin := prScript(t, "claude",
		"#!/bin/sh\nhead -c 70000 /dev/zero | tr '\\0' x; echo\necho short\n")
	p, err := Start(context.Background(), bin, nil, t.TempDir(), nil)
	if !testhelp.Equal(t, "err", err, nil) {
		return
	}
	first := prNext(t, p.Lines())
	testhelp.Equal(t, "first length", len(first), 70000)
	second := prNext(t, p.Lines())
	testhelp.Equal(t, "second line", string(second), "short")
}

// Process example 6: the Fake records writes, replays an emitted line, and
// reports ErrExited after Finish.
func TestProcessExample6(t *testing.T) {
	f := NewFake()
	f.Emit([]byte(`{"type":"x"}`))
	if err := f.Write([]byte("a\n")); err != nil {
		t.Fatalf("write a: %v", err)
	}
	if err := f.Write([]byte("b\n")); err != nil {
		t.Fatalf("write b: %v", err)
	}
	f.Finish(0)
	line := prNext(t, f.Lines())
	testhelp.Equal(t, "line", string(line), `{"type":"x"}`)
	select {
	case _, ok := <-f.Lines():
		if ok {
			t.Error("lines channel still open")
		}
	case <-time.After(prTimeout):
		t.Fatal("timeout waiting for lines to close")
	}
	testhelp.Equal(t, "written", f.Written(), [][]byte{[]byte("a\n"), []byte("b\n")})
	testhelp.Equal(t, "exit", prExit(t, f), ExitStatus{Code: 0})
	testhelp.Equal(t, "write after exit", f.Write([]byte("c\n")), ErrExited)
	testhelp.Equal(t, "killed", f.Killed(), false)
}

// Process example 7: Fake.Kill finishes with -1, records the kill, and keeps
// its fixed pid.
func TestProcessExample7(t *testing.T) {
	f := NewFake()
	if err := f.Kill(); err != nil {
		t.Fatalf("kill: %v", err)
	}
	testhelp.Equal(t, "exit", prExit(t, f), ExitStatus{Code: -1})
	testhelp.Equal(t, "killed", f.Killed(), true)
	testhelp.Equal(t, "pid", f.PID(), 4242)
}
