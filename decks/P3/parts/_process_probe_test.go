package claude

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

func pPRScript(t *testing.T, text string) string {
	t.Helper()
	bin := filepath.Join(t.TempDir(), "stub-claude")
	if err := os.WriteFile(bin, []byte(text), 0o755); err != nil {
		t.Fatal(err)
	}
	return bin
}

func pPRStart(t *testing.T, what string, ctx context.Context, bin string) Process {
	t.Helper()
	p, err := Start(ctx, bin, nil, t.TempDir(), nil)
	if err != nil {
		t.Fatalf("%s: Start error %v, want nil", what, err)
	}
	if p == nil {
		t.Fatalf("%s: Start returned a nil Process", what)
	}
	return p
}

// pPRLine reads one line; ok false when the channel is closed; fails the test after d.
func pPRLine(t *testing.T, what string, ch <-chan []byte, d time.Duration) ([]byte, bool) {
	t.Helper()
	select {
	case l, ok := <-ch:
		return l, ok
	case <-time.After(d):
		t.Fatalf("%s: no line and no close within %v", what, d)
	}
	return nil, false
}

func pPRExit(t *testing.T, what string, ch <-chan ExitStatus, d time.Duration) ExitStatus {
	t.Helper()
	select {
	case st := <-ch:
		return st
	case <-time.After(d):
		t.Fatalf("%s: no ExitStatus within %v", what, d)
	}
	return ExitStatus{}
}

func TestProbeProcessExample1(t *testing.T) {
	bin := pPRScript(t, "#!/bin/sh\necho '{\"type\":\"system\",\"subtype\":\"init\"}'\ncat\nexit 4\n")
	p := pPRStart(t, "example 1", context.Background(), bin)
	first, _ := pPRLine(t, "example 1 first line", p.Lines(), 3*time.Second)
	testhelp.Equal(t, "example 1 first line", string(first), `{"type":"system","subtype":"init"}`)
	testhelp.Equal(t, "example 1 Write error", p.Write([]byte("{\"type\":\"user\"}\n")), error(nil))
	next, _ := pPRLine(t, "example 1 next line", p.Lines(), 3*time.Second)
	testhelp.Equal(t, "example 1 next line", string(next), `{"type":"user"}`)
	if p.PID() <= 0 {
		t.Errorf("example 1 PID: %d, want > 0", p.PID())
	}
	p.Stop()
	st := pPRExit(t, "example 1 after Stop", p.Exit(), 3*time.Second)
	testhelp.Equal(t, "example 1 ExitStatus", st, ExitStatus{Code: 4, Stderr: ""})
	_, open := pPRLine(t, "example 1 Lines after the exit", p.Lines(), time.Second)
	testhelp.Equal(t, "example 1 Lines() closed", open, false)
	// variant: empty lines are skipped, several lines in order, the env reaches the child
	bin = pPRScript(t, "#!/bin/sh\necho a1\necho\necho \"$PRB_MARK\"\nexit 0\n")
	p, err := Start(context.Background(), bin, nil, t.TempDir(), []string{"PRB_MARK=m-31"})
	if err != nil || p == nil {
		t.Fatalf("example 1 variant: Start (%v)", err)
	}
	var got []string
	for {
		l, ok := pPRLine(t, "example 1 variant lines", p.Lines(), 3*time.Second)
		if !ok {
			break
		}
		got = append(got, string(l))
	}
	testhelp.Equal(t, "example 1 variant lines (empty skipped, env passed)", got, []string{"a1", "m-31"})
	testhelp.Equal(t, "example 1 variant ExitStatus", pPRExit(t, "example 1 variant", p.Exit(), 3*time.Second), ExitStatus{Code: 0, Stderr: ""})
}

func TestProbeProcessExample2(t *testing.T) {
	p := pPRStart(t, "example 2", context.Background(), pPRScript(t, "#!/bin/sh\necho oops >&2\nexit 2\n"))
	st := pPRExit(t, "example 2", p.Exit(), 3*time.Second)
	testhelp.Equal(t, "example 2 ExitStatus", st, ExitStatus{Code: 2, Stderr: "oops\n"})
	err := p.Write([]byte("x\n"))
	if !errors.Is(err, ErrExited) {
		t.Errorf("example 2 Write after the exit: %v, want ErrExited", err)
	}
	if ErrExited == nil || ErrExited.Error() != "claude: process exited" {
		t.Errorf("example 2 ErrExited text: %v, want %q", ErrExited, "claude: process exited")
	}
	// variant: only the last 4096 bytes of stderr are kept
	p = pPRStart(t, "example 2 variant", context.Background(), pPRScript(t, "#!/bin/sh\nhead -c 5000 /dev/zero | tr '\\0' e >&2\nprintf 'END' >&2\nexit 3\n"))
	st = pPRExit(t, "example 2 variant", p.Exit(), 3*time.Second)
	testhelp.Equal(t, "example 2 variant Code", st.Code, 3)
	testhelp.Equal(t, "example 2 variant Stderr (last 4096 bytes)", st.Stderr, strings.Repeat("e", 4093)+"END")
}

func TestProbeProcessExample3(t *testing.T) {
	p := pPRStart(t, "example 3", context.Background(), pPRScript(t, "#!/bin/sh\nsleep 30 &\necho started\nwait\n"))
	l, _ := pPRLine(t, "example 3 first line", p.Lines(), 3*time.Second)
	testhelp.Equal(t, "example 3 first line", string(l), "started")
	p.Kill()
	st := pPRExit(t, "example 3 after Kill (the whole group: sh and its background sleep, which holds stdout)", p.Exit(), 2*time.Second)
	testhelp.Equal(t, "example 3 ExitStatus.Code", st.Code, -1)
	// variant: a cancelled context kills the group as Kill does
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	p = pPRStart(t, "example 3 variant", ctx, pPRScript(t, "#!/bin/sh\nsleep 31 &\necho up\nwait\n"))
	l, _ = pPRLine(t, "example 3 variant first line", p.Lines(), 3*time.Second)
	testhelp.Equal(t, "example 3 variant first line", string(l), "up")
	cancel()
	st = pPRExit(t, "example 3 variant after the context is cancelled (the whole group)", p.Exit(), 2*time.Second)
	testhelp.Equal(t, "example 3 variant ExitStatus.Code", st.Code, -1)
}

func TestProbeProcessExample4(t *testing.T) {
	for _, c := range []struct{ bin, dir string }{{"/nonexistent/claude", t.TempDir()}, {"sh", "/nonexistent/dir"}} {
		p, err := Start(context.Background(), c.bin, nil, c.dir, nil)
		if p != nil {
			t.Errorf("example 4 Start(%q, dir %q): a non-nil Process", c.bin, c.dir)
		}
		if err == nil {
			t.Errorf("example 4 Start(%q, dir %q): error nil, want one beginning %q", c.bin, c.dir, "claude: start ")
		} else if !strings.HasPrefix(err.Error(), "claude: start ") {
			t.Errorf("example 4 Start(%q, dir %q): error %q does not begin %q", c.bin, c.dir, err.Error(), "claude: start ")
		}
	}
}

func TestProbeProcessExample5(t *testing.T) {
	p := pPRStart(t, "example 5", context.Background(), pPRScript(t, "#!/bin/sh\nhead -c 70000 /dev/zero | tr '\\0' x; echo\necho short\n"))
	l1, _ := pPRLine(t, "example 5 first line", p.Lines(), 3*time.Second)
	testhelp.Equal(t, "example 5 first line length", len(l1), 70000)
	l2, _ := pPRLine(t, "example 5 second line", p.Lines(), 3*time.Second)
	testhelp.Equal(t, "example 5 second line", string(l2), "short")
	// variant: a 1 MiB line
	p = pPRStart(t, "example 5 variant", context.Background(), pPRScript(t, "#!/bin/sh\nhead -c 1048576 /dev/zero | tr '\\0' y; echo\n"))
	l1, _ = pPRLine(t, "example 5 variant line", p.Lines(), 3*time.Second)
	testhelp.Equal(t, "example 5 variant line length (1 MiB)", len(l1), 1048576)
}

func TestProbeProcessExample6(t *testing.T) {
	f := NewFake()
	if f == nil {
		t.Fatalf("example 6: NewFake returned nil")
	}
	done := make(chan struct{})
	go func() { f.Emit([]byte(`{"type":"x"}`)); close(done) }()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Fatalf("example 6: Emit blocks with no reader (the Fake's Lines() must be buffered)")
	}
	testhelp.Equal(t, "example 6 Write a", f.Write([]byte("a\n")), error(nil))
	testhelp.Equal(t, "example 6 Write b", f.Write([]byte("b\n")), error(nil))
	f.Finish(0)
	var lines []string
	for {
		l, ok := pPRLine(t, "example 6 Lines", f.Lines(), 2*time.Second)
		if !ok {
			break
		}
		lines = append(lines, string(l))
	}
	testhelp.Equal(t, "example 6 lines", lines, []string{`{"type":"x"}`})
	testhelp.Equal(t, "example 6 Written", f.Written(), [][]byte{[]byte("a\n"), []byte("b\n")})
	testhelp.Equal(t, "example 6 ExitStatus", pPRExit(t, "example 6", f.Exit(), 2*time.Second), ExitStatus{Code: 0})
	if err := f.Write([]byte("c\n")); !errors.Is(err, ErrExited) {
		t.Errorf("example 6 Write after Finish: %v, want ErrExited", err)
	}
	testhelp.Equal(t, "example 6 Killed", f.Killed(), false)
	// variant: Written is a copy; Finish twice, Emit after Finish and Kill after Finish never panic
	if w := f.Written(); len(w) > 0 {
		w[0] = []byte("zz")
	}
	testhelp.Equal(t, "example 6 Written is a copy", f.Written(), [][]byte{[]byte("a\n"), []byte("b\n")})
	func() {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("example 6 variant: panic after Finish: %v", r)
			}
		}()
		f.Finish(5)
		f.Emit([]byte("late"))
		f.Kill()
		f.Stop()
	}()
	// variant: a Fake used as a Process
	var p Process = NewFake()
	testhelp.Equal(t, "example 6 Process PID", p.PID(), 4242)
}

func TestProbeProcessExample7(t *testing.T) {
	f := NewFake()
	if f == nil {
		t.Fatalf("example 7: NewFake returned nil")
	}
	f.Kill()
	testhelp.Equal(t, "example 7 ExitStatus", pPRExit(t, "example 7", f.Exit(), 2*time.Second), ExitStatus{Code: -1})
	testhelp.Equal(t, "example 7 Killed", f.Killed(), true)
	testhelp.Equal(t, "example 7 PID", f.PID(), 4242)
	_, open := pPRLine(t, "example 7 Lines after Kill", f.Lines(), time.Second)
	testhelp.Equal(t, "example 7 Lines() closed", open, false)
	// variant: Stop finishes with 0; Finish(3) delivers 3
	g := NewFake()
	g.Stop()
	testhelp.Equal(t, "example 7 Stop → ExitStatus", pPRExit(t, "example 7 Stop", g.Exit(), 2*time.Second), ExitStatus{Code: 0})
	testhelp.Equal(t, "example 7 Stop → Killed", g.Killed(), false)
	h := NewFake()
	h.Finish(3)
	testhelp.Equal(t, "example 7 Finish(3) → ExitStatus", pPRExit(t, "example 7 Finish(3)", h.Exit(), 2*time.Second), ExitStatus{Code: 3})
	h.Kill()
	testhelp.Equal(t, "example 7 Kill after Finish → Killed", h.Killed(), true)
}
