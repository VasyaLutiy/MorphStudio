package daemon

import (
	"context"
	"encoding/json"
	"strings"

	"morphstudio/claude"
	"morphstudio/eventlog"
	"morphstudio/stream"
	"morphstudio/supervisor"
)

// startPump is installed here: daemon.go only declares the hook, the pump
// machinery lives in this file.
func init() {
	startPump = func(d *Daemon, ctx context.Context, project string, p claude.Process, l *eventlog.Log) {
		pr := d.projects[project]
		if pr == nil {
			return
		}
		if pr.pumps == nil {
			pr.pumps = map[claude.Process]chan struct{}{}
		}
		done := make(chan struct{})
		pr.pumps[p] = done
		go func() {
			d.pump(ctx, project, p, l)
			close(done)
			d.mu.Lock()
			delete(pr.pumps, p)
			d.mu.Unlock()
		}()
	}
}

// pumpExit is the payload of an "exit" log entry.
type pumpExit struct {
	Code   int    `json:"code"`
	Stderr string `json:"stderr"`
}

// pumpTail keeps the last n lines of s, with one trailing newline dropped
// first, and returns "" for an empty s.
func pumpTail(s string, n int) string {
	s = strings.TrimSuffix(s, "\n")
	if s == "" {
		return ""
	}
	lines := strings.Split(s, "\n")
	if len(lines) > n {
		lines = lines[len(lines)-n:]
	}
	return strings.Join(lines, "\n")
}

// pumpTurnHeadline is the headline of a "turn" post for a phase.
func pumpTurnHeadline(phase, text string) string {
	if text == "" {
		text = "(no text)"
	}
	r := []rune(text)
	if len(r) > 200 {
		r = r[:200]
	}
	return phase + " turn ended: " + string(r)
}

// pump drains one process: every line is logged, parsed, folded into the
// machine and the loop, and whatever they return is executed; the exit is
// logged and, when the process is still the project's current one, handed to
// the loop.
func (d *Daemon) pump(ctx context.Context, project string, p claude.Process, l *eventlog.Log) {
	for line := range p.Lines() {
		d.pumpLine(ctx, project, p, l, line)
	}
	st := <-p.Exit()
	d.mu.Lock()
	defer d.mu.Unlock()
	if b, err := json.Marshal(pumpExit{Code: st.Code, Stderr: pumpTail(st.Stderr, 20)}); err == nil {
		_, _ = l.Append("exit", json.RawMessage(b), d.deps.Now())
	}
	pr := d.projects[project]
	if pr == nil || pr.proc != p {
		return
	}
	pr.proc = nil
	if pr.done != nil && pr.done.proc == p {
		d.finishDone(ctx, project, p)
		return
	}
	d.execute(ctx, project, pr.state.Loop.Exited(st.Code, st.Stderr, d.deps.Now(), d.deps.Config.Loop))
}

// pumpLine handles one line of a process. It is called with d.mu free and
// holds it for the whole handling, so a caller that sees the line in the
// event log sees its effects too.
func (d *Daemon) pumpLine(ctx context.Context, project string, p claude.Process, l *eventlog.Log, line []byte) {
	d.mu.Lock()
	defer d.mu.Unlock()

	now := d.deps.Now()
	msg := json.RawMessage(line)
	if !json.Valid(line) {
		if b, err := json.Marshal(string(line)); err == nil {
			msg = json.RawMessage(b)
		}
	}
	_, _ = l.Append("out", msg, now)

	ev, err := stream.Parse(line)
	if err != nil {
		return
	}
	pr := d.projects[project]
	if pr == nil || pr.proc != p {
		return
	}
	loop := &pr.state.Loop
	loop.Event(now)

	macts := pr.machine.Apply(ev, now)
	var lacts []supervisor.Action
	switch {
	case ev.Type == "result" && ev.Result != nil:
		lacts = loop.Turn(*ev.Result, now, d.deps.Config.Loop)
	case ev.Type == "rate_limit_event" && ev.Limits != nil:
		lacts = loop.Limits(*ev.Limits, now, d.deps.Config.Loop)
	}
	for _, a := range macts {
		switch a.Kind {
		case "write":
			d.writeLine(project, a.Line)
		case "ask":
			_, _ = d.deps.Post(ctx, project, "ask", a.Headline, a.Numbers)
		case "turn":
			_, _ = d.deps.Post(ctx, project, "idle", pumpTurnHeadline(loop.Phase, a.Headline), a.Numbers)
		}
	}
	d.execute(ctx, project, lacts)
	if ev.Type == "result" {
		d.finishDone(ctx, project, p)
	}
}

// Wait blocks until every pump of the project whose process is no longer the
// current one has returned. It returns at once when there is none.
func (d *Daemon) Wait(project string) {
	d.mu.Lock()
	pr := d.projects[project]
	var done []chan struct{}
	if pr != nil {
		for p, ch := range pr.pumps {
			if p != pr.proc {
				done = append(done, ch)
			}
		}
	}
	d.mu.Unlock()
	for _, ch := range done {
		<-ch
	}
}
