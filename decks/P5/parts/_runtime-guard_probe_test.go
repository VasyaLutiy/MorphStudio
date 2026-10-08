package supervisor

import (
	"encoding/json"
	"os"
	"testing"
	"time"

	"morphstudio/control"
	"morphstudio/gitrules"
	"morphstudio/internal/testhelp"
	"morphstudio/queue"
	"morphstudio/stream"
)

func pRGAt(h, m, s int) time.Time { return time.Date(2026, 10, 8, h, m, s, 0, time.UTC) }

var pRGCfg = Config{MaxResumesPerHour: 3, StallMinutes: 30, StretchUSD: 30, UsageAlertPercent: 50}

func pRGPost(kind, headline, numbers string) Action {
	return Action{Kind: "post", Milestone: control.Milestone{Kind: kind, Headline: headline, Numbers: numbers}}
}

// pRGRunning is the running loop of example 1 (P17, "id-1", started 15:00:01Z) with the given defaults.
func pRGRunning(t *testing.T, what string, defaults queue.Caps) *Loop {
	t.Helper()
	b, err := os.ReadFile("../tests/fixtures/plan/queue-3.json")
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	var p queue.Plan
	if err := json.Unmarshal(b, &p); err != nil {
		t.Fatalf("decode fixture: %v", err)
	}
	q, err := queue.Load(p, defaults)
	if err != nil {
		t.Fatalf("queue.Load: %v", err)
	}
	l := New("demo", q)
	if l == nil {
		t.Fatalf("%s: New returned nil", what)
	}
	l.Begin(pRGAt(15, 0, 0))
	l.StartChecked(gitrules.Start{Mode: "fresh"}, func() string { return "id-1" }, pRGAt(15, 0, 1))
	if l.State != "running" || l.Phase != "P17" {
		t.Fatalf("%s: the running loop has State %q Phase %q, want running P17", what, l.State, l.Phase)
	}
	return l
}

func pRGDefault(t *testing.T, what string) *Loop {
	t.Helper()
	return pRGRunning(t, what, queue.Caps{ClaudeUSD: 30, Hours: 3, ExecutorUSD: 5})
}

func pRGStopKind(l *Loop) string {
	if l.Stop == nil {
		return "<nil Stop>"
	}
	return l.Stop.Kind
}

func TestProbeRuntimeGuardExample1(t *testing.T) {
	l := pRGDefault(t, "example 1")
	acts := l.Exited(4, pRGAt(15, 20, 0), pRGCfg)
	testhelp.Equal(t, "example 1 Exited(4)", acts, []Action{
		pRGPost("watchdog", "P17: session exited (code 4), resuming", "resume 1 of 3 this hour"),
		{Kind: "start_check", Phase: "P17"},
	})
	testhelp.Equal(t, "example 1 State, SessionID", []string{l.State, l.SessionID}, []string{"starting", "id-1"})
	testhelp.Equal(t, "example 1 Resumes", l.Resumes, []time.Time{pRGAt(15, 20, 0)})
	// variant: another max and code; a starting loop follows the same rule
	v := pRGDefault(t, "example 1 variant")
	acts = v.Exited(137, pRGAt(15, 21, 0), Config{MaxResumesPerHour: 5})
	testhelp.Equal(t, "example 1 variant Exited(137) max 5", acts, []Action{
		pRGPost("watchdog", "P17: session exited (code 137), resuming", "resume 1 of 5 this hour"),
		{Kind: "start_check", Phase: "P17"},
	})
	acts = v.Exited(2, pRGAt(15, 22, 0), Config{MaxResumesPerHour: 5})
	testhelp.Equal(t, "example 1 variant Exited while starting", acts, []Action{
		pRGPost("watchdog", "P17: session exited (code 2), resuming", "resume 2 of 5 this hour"),
		{Kind: "start_check", Phase: "P17"},
	})
}

func TestProbeRuntimeGuardExample2(t *testing.T) {
	l := pRGDefault(t, "example 2")
	resume := func(at time.Time) {
		l.StartChecked(gitrules.Start{Mode: "resume", Reason: "local ahead of origin/main"}, func() string { return "id-x" }, at)
	}
	l.Exited(4, pRGAt(15, 20, 0), pRGCfg)
	resume(pRGAt(15, 20, 5))
	l.Exited(4, pRGAt(15, 30, 0), pRGCfg)
	resume(pRGAt(15, 30, 5))
	l.Exited(4, pRGAt(15, 40, 0), pRGCfg)
	resume(pRGAt(15, 40, 5))
	testhelp.Equal(t, "example 2 three resumes kept", len(l.Resumes), 3)
	acts := l.Exited(1, pRGAt(15, 50, 0), pRGCfg)
	testhelp.Equal(t, "example 2 fourth exit", acts, []Action{pRGPost("stop", "P17: session keeps exiting", "exited 3 times within an hour (last code 1)")})
	testhelp.Equal(t, "example 2 State, Stop.Kind, Queue.State", []string{l.State, pRGStopKind(l), l.Queue.State}, []string{"waiting", "crash", "stopped"})
	if l.Stop != nil {
		testhelp.Equal(t, "example 2 Stop", *l.Stop, control.Stop{Kind: "crash", Phase: "P17", Reason: "exited 3 times within an hour (last code 1)", At: pRGAt(15, 50, 0)})
	}
	testhelp.Equal(t, "example 2 Queue.Reason", l.Queue.Reason, "crash")
	o := pRGDefault(t, "example 2 (older resume)")
	o.Resumes = []time.Time{pRGAt(15, 20, 0), pRGAt(15, 30, 0), pRGAt(15, 40, 0)}
	acts = o.Exited(1, pRGAt(16, 25, 0), pRGCfg)
	testhelp.Equal(t, "example 2 fourth exit at 16:25", acts, []Action{
		pRGPost("watchdog", "P17: session exited (code 1), resuming", "resume 3 of 3 this hour"),
		{Kind: "start_check", Phase: "P17"},
	})
	// variant: exactly one hour old does not count; max 1
	e := pRGDefault(t, "example 2 variant")
	e.Resumes = []time.Time{pRGAt(15, 0, 0)}
	acts = e.Exited(9, pRGAt(16, 0, 0), Config{MaxResumesPerHour: 1})
	testhelp.Equal(t, "example 2 variant a resume exactly 1 h old", acts, []Action{
		pRGPost("watchdog", "P17: session exited (code 9), resuming", "resume 1 of 1 this hour"),
		{Kind: "start_check", Phase: "P17"},
	})
	e2 := pRGDefault(t, "example 2 variant")
	e2.Resumes = []time.Time{pRGAt(15, 0, 1)}
	acts = e2.Exited(9, pRGAt(16, 0, 0), Config{MaxResumesPerHour: 1})
	testhelp.Equal(t, "example 2 variant max 1 reached", acts, []Action{pRGPost("stop", "P17: session keeps exiting", "exited 1 times within an hour (last code 9)")})
}

func TestProbeRuntimeGuardExample3(t *testing.T) {
	l := pRGDefault(t, "example 3")
	l.StretchUSD, l.SessionUSD = 28, 0
	acts := l.Turn(stream.Result{TotalCostUSD: 1.5}, pRGAt(15, 50, 0), pRGCfg)
	testhelp.Equal(t, "example 3 first Turn", acts, []Action(nil))
	testhelp.Equal(t, "example 3 SessionUSD", l.SessionUSD, 1.5)
	acts = l.Turn(stream.Result{TotalCostUSD: 2.25}, pRGAt(16, 0, 0), pRGCfg)
	testhelp.Equal(t, "example 3 second Turn", acts, []Action{{Kind: "kill"}, pRGPost("stop", "P17: stretch cap", "stretch cap $30.00 reached ($30.2500)")})
	testhelp.Equal(t, "example 3 State, Stop.Kind, Queue.State", []string{l.State, pRGStopKind(l), l.Queue.State}, []string{"waiting", "cap", "stopped"})
	if l.Stop != nil {
		testhelp.Equal(t, "example 3 Stop", *l.Stop, control.Stop{Kind: "cap", Phase: "P17", Reason: "stretch cap $30.00 reached ($30.2500)", At: pRGAt(16, 0, 0)})
	}
	z := pRGDefault(t, "example 3")
	z.StretchUSD = 28
	z.Turn(stream.Result{TotalCostUSD: 1.5}, pRGAt(15, 50, 0), Config{MaxResumesPerHour: 3})
	testhelp.Equal(t, "example 3 cfg.StretchUSD 0", z.Turn(stream.Result{TotalCostUSD: 2.25}, pRGAt(16, 0, 0), Config{MaxResumesPerHour: 3}), []Action(nil))
	testhelp.Equal(t, "example 3 cfg.StretchUSD 0 State", z.State, "running")
	// variant: another cap, the sum exactly at the cap, a paused loop
	v := pRGDefault(t, "example 3 variant")
	v.StretchUSD, v.State = 7.5, "paused"
	acts = v.Turn(stream.Result{TotalCostUSD: 2.5}, pRGAt(16, 1, 0), Config{StretchUSD: 10})
	testhelp.Equal(t, "example 3 variant cap 10 reached exactly", acts, []Action{{Kind: "kill"}, pRGPost("stop", "P17: stretch cap", "stretch cap $10.00 reached ($10.0000)")})
	u := pRGDefault(t, "example 3 variant")
	u.StretchUSD = 7.5
	testhelp.Equal(t, "example 3 variant below cap 10", u.Turn(stream.Result{TotalCostUSD: 2.4375}, pRGAt(16, 1, 0), Config{StretchUSD: 10}), []Action(nil))
}

func TestProbeRuntimeGuardExample4(t *testing.T) {
	lim := stream.Limits{FiveHour: stream.Window{Utilization: 0.22, ResetsAt: 1791469200}, SevenDay: stream.Window{Utilization: 0.6, ResetsAt: 1791853200}}
	l := pRGDefault(t, "example 4")
	acts := l.Limits(lim, pRGAt(13, 9, 3), pRGCfg)
	testhelp.Equal(t, "example 4 Limits", acts, []Action{pRGPost("info", "P17: weekly usage 60%", "resets 2026-10-13T01:00:00Z")})
	testhelp.Equal(t, "example 4 Alerted", l.Alerted, true)
	testhelp.Equal(t, "example 4 second Limits", l.Limits(lim, pRGAt(13, 9, 4), pRGCfg), []Action(nil))
	testhelp.Equal(t, "example 4 State unchanged", l.State, "running")
	o := pRGDefault(t, "example 4")
	testhelp.Equal(t, "example 4 UsageAlertPercent 0", o.Limits(lim, pRGAt(13, 9, 3), Config{MaxResumesPerHour: 3, StallMinutes: 30, StretchUSD: 30}), []Action(nil))
	testhelp.Equal(t, "example 4 UsageAlertPercent 0 Alerted", o.Alerted, false)
	// variant: another threshold, below and at it; other reset
	v := pRGDefault(t, "example 4 variant")
	below := stream.Limits{SevenDay: stream.Window{Utilization: 0.74, ResetsAt: 1800000000}}
	testhelp.Equal(t, "example 4 variant 74% under an alert at 75", v.Limits(below, pRGAt(13, 0, 0), Config{UsageAlertPercent: 75}), []Action(nil))
	at := stream.Limits{SevenDay: stream.Window{Utilization: 0.75, ResetsAt: 1800000000}}
	testhelp.Equal(t, "example 4 variant 75% at an alert at 75", v.Limits(at, pRGAt(13, 0, 0), Config{UsageAlertPercent: 75}), []Action{pRGPost("info", "P17: weekly usage 75%", "resets 2027-01-15T08:00:00Z")})
}

func TestProbeRuntimeGuardExample5(t *testing.T) {
	l := pRGDefault(t, "example 5")
	acts := l.Limits(stream.Limits{FiveHour: stream.Window{Utilization: 1, ResetsAt: 1791469200}, SevenDay: stream.Window{Utilization: 0.6, ResetsAt: 1791853200}}, pRGAt(13, 9, 3), pRGCfg)
	testhelp.Equal(t, "example 5 Limits (five hour at 1)", acts, []Action{pRGPost("watchdog", "P17: usage limit, paused", "until 2026-10-08T14:20:00Z")})
	testhelp.Equal(t, "example 5 State", l.State, "paused")
	testhelp.Equal(t, "example 5 PausedUntil", l.PausedUntil, time.Date(2026, 10, 8, 14, 20, 0, 0, time.UTC))
	b := pRGDefault(t, "example 5")
	b.Limits(stream.Limits{FiveHour: stream.Window{Utilization: 1, ResetsAt: 1791469200}, SevenDay: stream.Window{Utilization: 1, ResetsAt: 1791853200}}, pRGAt(13, 9, 3), pRGCfg)
	testhelp.Equal(t, "example 5 both windows at 1: PausedUntil", b.PausedUntil, time.Date(2026, 10, 13, 1, 0, 0, 0, time.UTC))
	// variant: only the seven-day window exhausted, over 1
	s := pRGDefault(t, "example 5 variant")
	acts = s.Limits(stream.Limits{FiveHour: stream.Window{Utilization: 0.3, ResetsAt: 1791469200}, SevenDay: stream.Window{Utilization: 1.2, ResetsAt: 1800000000}}, pRGAt(13, 9, 3), Config{})
	testhelp.Equal(t, "example 5 variant seven day at 1.2", acts, []Action{pRGPost("watchdog", "P17: usage limit, paused", "until 2027-01-15T08:00:00Z")})
	testhelp.Equal(t, "example 5 variant PausedUntil location", s.PausedUntil.Location(), time.UTC)
}

func TestProbeRuntimeGuardExample6(t *testing.T) {
	l := pRGDefault(t, "example 6")
	l.Limits(stream.Limits{FiveHour: stream.Window{Utilization: 1, ResetsAt: 1791469200}, SevenDay: stream.Window{Utilization: 0.6, ResetsAt: 1791853200}}, pRGAt(13, 9, 3), pRGCfg)
	testhelp.Equal(t, "example 6 Tick 14:19:59", l.Tick(pRGAt(14, 19, 59), pRGCfg), []Action(nil))
	testhelp.Equal(t, "example 6 still paused", l.State, "paused")
	acts := l.Tick(pRGAt(14, 20, 0), pRGCfg)
	testhelp.Equal(t, "example 6 Tick 14:20:00", acts, []Action{
		{Kind: "write", Line: "Continue by docs/AUTONOMY.md from where you stopped; the usage window has reset."},
		pRGPost("watchdog", "P17: resumed after the limit", ""),
	})
	testhelp.Equal(t, "example 6 State", l.State, "running")
	testhelp.Equal(t, "example 6 LastEventAt", l.LastEventAt, pRGAt(14, 20, 0))
}

func TestProbeRuntimeGuardExample7(t *testing.T) {
	l := pRGDefault(t, "example 7")
	testhelp.Equal(t, "example 7 Tick 15:29:00", l.Tick(pRGAt(15, 29, 0), pRGCfg), []Action(nil))
	acts := l.Tick(pRGAt(15, 31, 0), pRGCfg)
	testhelp.Equal(t, "example 7 Tick 15:31:00 (stall)", acts, []Action{
		pRGPost("watchdog", "P17: no events for 30 min", "one nudge sent"),
		{Kind: "write", Line: "Nothing has happened for 30 minutes. Check the state of your agents and the run branch, then continue by docs/AUTONOMY.md; post the current state first."},
	})
	testhelp.Equal(t, "example 7 Nudged", l.Nudged, true)
	testhelp.Equal(t, "example 7 Tick 16:10:00 (one nudge only)", l.Tick(pRGAt(16, 10, 0), pRGCfg), []Action(nil))
	l.Event(pRGAt(16, 20, 0))
	testhelp.Equal(t, "example 7 Event sets LastEventAt", l.LastEventAt, pRGAt(16, 20, 0))
	acts = l.Tick(pRGAt(18, 0, 1), pRGCfg)
	testhelp.Equal(t, "example 7 Tick 18:00:01 (wall clock)", acts, []Action{{Kind: "kill"}, pRGPost("stop", "P17: wall clock cap", "wall clock cap 3 h reached")})
	testhelp.Equal(t, "example 7 State, Stop.Kind, Queue.State", []string{l.State, pRGStopKind(l), l.Queue.State}, []string{"waiting", "cap", "stopped"})
	// variant: other stall minutes and hours; one second short of each
	v := pRGRunning(t, "example 7 variant", queue.Caps{ClaudeUSD: 30, Hours: 2.5, ExecutorUSD: 5})
	cfg := Config{StallMinutes: 45}
	testhelp.Equal(t, "example 7 variant 44:59 of 45", v.Tick(pRGAt(15, 45, 0), cfg), []Action(nil))
	acts = v.Tick(pRGAt(15, 45, 1), cfg)
	testhelp.Equal(t, "example 7 variant stall 45", acts, []Action{
		pRGPost("watchdog", "P17: no events for 45 min", "one nudge sent"),
		{Kind: "write", Line: "Nothing has happened for 45 minutes. Check the state of your agents and the run branch, then continue by docs/AUTONOMY.md; post the current state first."},
	})
	testhelp.Equal(t, "example 7 variant 2.5 h less a second", v.Tick(pRGAt(17, 30, 0), cfg), []Action(nil))
	acts = v.Tick(pRGAt(17, 30, 1), cfg)
	testhelp.Equal(t, "example 7 variant wall clock 2.5 h", acts, []Action{{Kind: "kill"}, pRGPost("stop", "P17: wall clock cap", "wall clock cap 2.5 h reached")})
	n := pRGDefault(t, "example 7 variant")
	testhelp.Equal(t, "example 7 variant StallMinutes 0 never nudges", n.Tick(pRGAt(17, 0, 0), Config{}), []Action(nil))
}

func TestProbeRuntimeGuardExample8(t *testing.T) {
	w := pRGDefault(t, "example 8")
	w.WaitOperator("gate", "x", "", pRGAt(15, 5, 0))
	testhelp.Equal(t, "example 8 waiting Exited", w.Exited(0, pRGAt(15, 6, 0), pRGCfg), []Action(nil))
	testhelp.Equal(t, "example 8 waiting Tick", w.Tick(pRGAt(23, 0, 0), pRGCfg), []Action(nil))
	testhelp.Equal(t, "example 8 waiting Turn", w.Turn(stream.Result{TotalCostUSD: 99}, pRGAt(15, 7, 0), pRGCfg), []Action(nil))
	testhelp.Equal(t, "example 8 waiting State", w.State, "waiting")
	i := New("demo", queue.Queue{})
	if i == nil {
		t.Fatalf("example 8: New returned nil")
	}
	testhelp.Equal(t, "example 8 idle Limits", i.Limits(stream.Limits{FiveHour: stream.Window{Utilization: 1, ResetsAt: 1791469200}}, pRGAt(13, 0, 0), pRGCfg), []Action(nil))
	testhelp.Equal(t, "example 8 idle State", i.State, "idle")
	// variant: an idle loop ignores exits and ticks; a paused loop does not pause again
	testhelp.Equal(t, "example 8 variant idle Exited", i.Exited(1, pRGAt(13, 0, 0), pRGCfg), []Action(nil))
	testhelp.Equal(t, "example 8 variant idle Tick", i.Tick(pRGAt(13, 0, 0), pRGCfg), []Action(nil))
	p := pRGDefault(t, "example 8 variant")
	p.State, p.PausedUntil = "paused", pRGAt(14, 20, 0)
	testhelp.Equal(t, "example 8 variant paused Limits at 1", p.Limits(stream.Limits{FiveHour: stream.Window{Utilization: 1, ResetsAt: 1800000000}}, pRGAt(14, 0, 0), pRGCfg), []Action(nil))
	testhelp.Equal(t, "example 8 variant paused PausedUntil kept", p.PausedUntil, pRGAt(14, 20, 0))
	// variant: the same calls on a running loop do act (the state is what made them nil)
	r := pRGDefault(t, "example 8 variant")
	testhelp.Equal(t, "example 8 variant running Turn(99)", r.Turn(stream.Result{TotalCostUSD: 99}, pRGAt(15, 7, 0), pRGCfg), []Action{{Kind: "kill"}, pRGPost("stop", "P17: stretch cap", "stretch cap $30.00 reached ($99.0000)")})
	r2 := pRGDefault(t, "example 8 variant")
	testhelp.Equal(t, "example 8 variant running Exited(0)", r2.Exited(0, pRGAt(15, 6, 0), pRGCfg), []Action{
		pRGPost("watchdog", "P17: session exited (code 0), resuming", "resume 1 of 3 this hour"),
		{Kind: "start_check", Phase: "P17"},
	})
}
