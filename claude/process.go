// Package claude launches the claude CLI and describes the MCP configuration
// file handed to it.
package claude

import (
	"bufio"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"sync"
	"syscall"
)

// ExitStatus is the outcome of one claude process: its exit code and the tail
// of its stderr.
type ExitStatus struct {
	Code   int
	Stderr string
}

// Process is a running claude process with line-delimited stdin and stdout,
// its exit status, and the fake.
type Process interface {
	Write(line []byte) error
	Lines() <-chan []byte
	Exit() <-chan ExitStatus
	Stop() error
	Kill() error
	PID() int
}

// ErrExited is returned by Write once the process has exited.
var ErrExited = errors.New("claude: process exited")

const (
	stderrKeep = 4096
	lineBufMax = 16 * 1024 * 1024
)

type stderrBuffer struct {
	mu  sync.Mutex
	buf []byte
}

func (b *stderrBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	b.buf = append(b.buf, p...)
	if len(b.buf) > stderrKeep {
		b.buf = b.buf[len(b.buf)-stderrKeep:]
	}
	return len(p), nil
}

func (b *stderrBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return string(b.buf)
}

type process struct {
	cmd    *exec.Cmd
	stdin  io.WriteCloser
	lines  chan []byte
	exit   chan ExitStatus
	pid    int
	stderr stderrBuffer
	mu     sync.Mutex
	exited bool
}

func killGroup(cmd *exec.Cmd) error {
	if cmd.Process == nil {
		return os.ErrProcessDone
	}
	if err := syscall.Kill(-cmd.Process.Pid, syscall.SIGKILL); err != nil {
		if errors.Is(err, syscall.ESRCH) {
			return os.ErrProcessDone
		}
		return err
	}
	return nil
}

// Start launches bin with args in dir, adding env to the environment, and
// returns the running Process.
func Start(ctx context.Context, bin string, args []string, dir string, env []string) (Process, error) {
	cmd := exec.CommandContext(ctx, bin, args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), env...)
	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("claude: start %s: %w", bin, err)
	}
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("claude: start %s: %w", bin, err)
	}
	stderr, err := cmd.StderrPipe()
	if err != nil {
		return nil, fmt.Errorf("claude: start %s: %w", bin, err)
	}
	cmd.SysProcAttr = &syscall.SysProcAttr{Setpgid: true}
	cmd.Cancel = func() error { return killGroup(cmd) }
	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("claude: start %s: %w", bin, err)
	}
	p := &process{
		cmd:   cmd,
		stdin: stdin,
		lines: make(chan []byte),
		exit:  make(chan ExitStatus, 1),
		pid:   cmd.Process.Pid,
	}
	stdoutDone := make(chan struct{})
	go func() {
		defer close(stdoutDone)
		defer close(p.lines)
		sc := bufio.NewScanner(stdout)
		sc.Buffer(make([]byte, 0, 64*1024), lineBufMax)
		for sc.Scan() {
			line := sc.Bytes()
			if len(line) == 0 {
				continue
			}
			b := make([]byte, len(line))
			copy(b, line)
			p.lines <- b
		}
	}()
	stderrDone := make(chan struct{})
	go func() {
		defer close(stderrDone)
		buf := make([]byte, 32*1024)
		for {
			n, err := stderr.Read(buf)
			if n > 0 {
				_, _ = p.stderr.Write(buf[:n])
			}
			if err != nil {
				return
			}
		}
	}()
	go func() {
		<-stdoutDone
		<-stderrDone
		_ = cmd.Wait()
		code := -1
		if cmd.ProcessState != nil {
			code = cmd.ProcessState.ExitCode()
		}
		p.mu.Lock()
		p.exited = true
		p.mu.Unlock()
		p.exit <- ExitStatus{Code: code, Stderr: p.stderr.String()}
		close(p.exit)
	}()
	return p, nil
}

func (p *process) Write(line []byte) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.exited {
		return ErrExited
	}
	_, err := p.stdin.Write(line)
	return err
}

func (p *process) Lines() <-chan []byte { return p.lines }

func (p *process) Exit() <-chan ExitStatus { return p.exit }

func (p *process) Stop() error { return p.stdin.Close() }

func (p *process) Kill() error {
	if err := killGroup(p.cmd); err != nil && !errors.Is(err, os.ErrProcessDone) {
		return err
	}
	return nil
}

func (p *process) PID() int { return p.pid }

// Fake is an in-memory Process for tests.
type Fake struct {
	mu       sync.Mutex
	lines    chan []byte
	exit     chan ExitStatus
	written  [][]byte
	finished bool
	killed   bool
}

// NewFake returns a ready Fake that implements Process.
func NewFake() *Fake {
	return &Fake{
		lines: make(chan []byte, 1024),
		exit:  make(chan ExitStatus, 1),
	}
}

// Emit delivers a line on Lines; after Finish it is dropped.
func (f *Fake) Emit(line []byte) {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.finished {
		return
	}
	b := make([]byte, len(line))
	copy(b, line)
	f.lines <- b
}

// Written returns a copy of every Write, in order.
func (f *Fake) Written() [][]byte {
	f.mu.Lock()
	defer f.mu.Unlock()
	out := make([][]byte, len(f.written))
	for i, w := range f.written {
		out[i] = make([]byte, len(w))
		copy(out[i], w)
	}
	return out
}

// Finish closes Lines and delivers ExitStatus{Code: code}.
func (f *Fake) Finish(code int) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.finishLocked(code)
}

func (f *Fake) finishLocked(code int) {
	if f.finished {
		return
	}
	f.finished = true
	close(f.lines)
	f.exit <- ExitStatus{Code: code}
	close(f.exit)
}

func (f *Fake) Write(line []byte) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	if f.finished {
		return ErrExited
	}
	b := make([]byte, len(line))
	copy(b, line)
	f.written = append(f.written, b)
	return nil
}

func (f *Fake) Lines() <-chan []byte { return f.lines }

func (f *Fake) Exit() <-chan ExitStatus { return f.exit }

// Stop finishes with 0 if not already finished.
func (f *Fake) Stop() error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.finishLocked(0)
	return nil
}

// Kill records the kill and finishes with -1 if not already finished.
func (f *Fake) Kill() error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.killed = true
	f.finishLocked(-1)
	return nil
}

// Killed reports whether Kill was called.
func (f *Fake) Killed() bool {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.killed
}

func (f *Fake) PID() int { return 4242 }
