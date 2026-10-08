// Package daemon launches claude sessions, pumps their stream output through
// the session machine and the supervisor loop, and exposes control.Control.
package daemon

import (
	"context"
	"encoding/json"

	"morphstudio/claude"
	"morphstudio/eventlog"
	"morphstudio/stream"
	"morphstudio/supervisor"
)

func init() {
	startPump = func(d *Daemon, ctx context.Context, project string, p claude.Process, l *eventlog.Log) {
		pr := d.projects[project]
		if pr == nil {
			return
		}
		done := make(chan struct{})
		pr.pumps[p] = done
		go func() {
			defer close(done)
			d.pump(ctx, project, p, l)
		}()
	}
}

// pump reads every line of p, records it, folds it into the session machine
// and the supervisor loop, and runs the actions they return. It closes when
// p.Lines() closes, at which point it consumes the exit status: if p is still
// the project's current process the loop sees the exit, otherwise nothing.
func (d *Daemon) pump(ctx context.Context, project string, p claude.Process, l *eventlog.Log) {
	for line := range p.Lines() {
		d.mu.Lock()
		pr := d.projects[project]
		if pr == nil {
			d.mu.Unlock()
			continue
		}

		now := d.deps.Now()
		msg := json.RawMessage(line)
		if !json.Valid(msg) {
			if b, err := json.Marshal(string(line)); err == nil {
				msg = b
			}
		}
		if l != nil {
			_, _ = l.Append("out", msg, now)
		}

		ev, err := stream.Parse(line)
		if err == nil && pr.proc == p && pr.machine != nil {
			pr.state.Loop.Event(now)
			macts := pr.machine.Apply(ev, now)

			var lacts []supervisor.Action
			switch ev.Type {
			case "result":
				if ev.Result != nil {
					lacts = pr.state.Loop.Turn(*ev.Result, now, d.deps.Config.Loop)
				}
			case "rate_limit_event":
				if ev.Limits != nil {
					lacts = pr.state.Loop.Limits(*ev.Limits, now, d.deps.Config.Loop)
				}
			}

			for _, a := range macts {
				switch a.Kind {
				case "write":
					d.writeLine(project, a.Line)
				case "ask":
					_, _ = d.deps.Post(ctx, project, "ask", a.Headline, a.Numbers)
				case "turn":
					headline := a.Headline
					if r := []rune(headline); len(r) > 200 {
						headline = string(r[:200])
					}
					if headline == "" {
						headline = "(no text)"
					}
					_, _ = d.deps.Post(ctx, project, "idle", pr.state.Loop.Phase+" turn ended: "+headline, a.Numbers)
				case "limit":
				}
			}

			d.execute(ctx, project, lacts)
		}
		d.mu.Unlock()
	}

	st := <-p.Exit()
	d.mu.Lock()
	pr := d.projects[project]
	if pr != nil && pr.proc == p {
		pr.proc = nil
		d.execute(ctx, project, pr.state.Loop.Exited(st.Code, d.deps.Now(), d.deps.Config.Loop))
	}
	d.mu.Unlock()
}

// Wait blocks until every pump of project whose process is not the current
// one has finished. It returns immediately when there is nothing to wait for.
func (d *Daemon) Wait(project string) {
	d.mu.Lock()
	var dones []chan struct{}
	if pr := d.projects[project]; pr != nil {
		for p, done := range pr.pumps {
			if p == pr.proc {
				continue
			}
			dones = append(dones, done)
		}
	}
	d.mu.Unlock()

	for _, done := range dones {
		<-done
	}
}
