package supervisor

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
	"time"

	"morphstudio/control"
	"morphstudio/gitrules"
	"morphstudio/internal/testhelp"
	"morphstudio/queue"
	"morphstudio/stream"
)

func gdAt(h, m, s int) time.Time {
	return time.Date(2026, 10, 8, h, m, s, 0, time.UTC)
}

func gdQueue(t testing.TB) queue.Queue {
	t.Helper()
	data, err := os.ReadFile("../tests/fixtures/plan/queue-3.json")
	if err != nil {
		t.Fatal(err)
	}
	var p queue.Plan
	if err := json.Unmarshal(data, &p); err != nil {
		t.Fatal(err)
	}
	q, err := queue.Load(p, queue.Caps{ClaudeUSD: 30, Hours: 3, ExecutorUSD: 5})
	if err != nil {
		t.Fatal(err)
	}
	return q
}

func gdCfg() Config {
	return Config{MaxResumesPerHour: 3, StallMinutes: 30, StretchUSD: 30, UsageAlertPercent: 50}
}

func gdNewID() string { return "id-1" }

func gdRunning(t testing.TB) *Loop {
	t.Helper()
	l := New("demo", gdQueue(t))
	if _, err := l.Begin(gdAt(15, 0, 0)); err != nil {
		t.Fatal(err)
	}
	l.StartChecked(gitrules.Start{Mode: "fresh"}, gdNewID, gdAt(15, 0, 1))
	return l
}

func TestRuntimeGuardExample1(t *testing.T) {
	cfg := gdCfg()
	l := gdRunning(t)
	testhelp.Equal(t, "state", l.State, "running")
	testhelp.Equal(t, "phase", l.Phase, "P17")
	testhelp.Equal(t, "session id", l.SessionID, "id-1")
	testhelp.Equal(t, "phase started at", l.PhaseStartedAt, gdAt(15, 0, 1))
	testhelp.Equal(t, "last event at", l.LastEventAt, gdAt(15, 0, 1))

	got := l.Exited(4, "", gdAt(15, 20, 0), cfg)
	want := []Action{
		{Kind: "post", Milestone: control.Milestone{Kind: "watchdog", Headline: "P17: session exited (code 4), restarting", Numbers: "restart 1 of 3 this hour"}},
		{Kind: "start_check", Phase: "P17"},
	}
	testhelp.Equal(t, "actions", got, want)
	testhelp.Equal(t, "state", l.State, "starting")
	testhelp.Equal(t, "restarts", l.Restarts, []time.Time{gdAt(15, 20, 0)})
	testhelp.Equal(t, "session id", l.SessionID, "id-1")
	if l.LastExit == nil {
		t.Fatal("last exit nil")
	}
	testhelp.Equal(t, "last exit", *l.LastExit, control.Exit{Code: 4, At: gdAt(15, 20, 0), Stderr: "", Restarts: 1})
}

func TestRuntimeGuardExample2(t *testing.T) {
	cfg := gdCfg()
	line := "Error: Invalid session ID. Must be a valid UUID.\n"

	l := gdRunning(t)
	l.Exited(4, "", gdAt(15, 20, 0), cfg)
	l.StartChecked(gitrules.Start{Mode: "fresh"}, gdNewID, gdAt(15, 20, 5))
	testhelp.Equal(t, "restarts kept", l.Restarts, []time.Time{gdAt(15, 20, 0)})

	got := l.Exited(1, line, gdAt(15, 30, 0), cfg)
	want := []Action{
		{Kind: "post", Milestone: control.Milestone{Kind: "watchdog", Headline: "P17: session exited (code 1), restarting", Numbers: "restart 2 of 3 this hour · Error: Invalid session ID. Must be a valid UUID."}},
		{Kind: "start_check", Phase: "P17"},
	}
	testhelp.Equal(t, "second exit", got, want)

	l.StartChecked(gitrules.Start{Mode: "resume", Reason: "dirty tree (1 paths)"}, gdNewID, gdAt(15, 30, 5))

	got = l.Exited(137, "", gdAt(15, 40, 0), cfg)
	want = []Action{
		{Kind: "post", Milestone: control.Milestone{Kind: "watchdog", Headline: "P17: session exited (code 137), restarting", Numbers: "restart 3 of 3 this hour"}},
		{Kind: "start_check", Phase: "P17"},
	}
	testhelp.Equal(t, "third exit", got, want)

	l.StartChecked(gitrules.Start{Mode: "fresh"}, gdNewID, gdAt(15, 40, 5))

	got = l.Exited(1, line, gdAt(15, 50, 0), cfg)
	want = []Action{
		{Kind: "post", Milestone: control.Milestone{Kind: "stop", Headline: "P17: session keeps exiting", Numbers: "exited 4 times within an hour (last code 1): Error: Invalid session ID. Must be a valid UUID."}},
	}
	testhelp.Equal(t, "fourth exit", got, want)
	testhelp.Equal(t, "state", l.State, "waiting")
	if l.Stop == nil {
		t.Fatal("stop nil")
	}
	testhelp.Equal(t, "stop", *l.Stop, control.Stop{Kind: "crash", Phase: "P17", Reason: "exited 4 times within an hour (last code 1): Error: Invalid session ID. Must be a valid UUID.", At: gdAt(15, 50, 0)})
	testhelp.Equal(t, "queue state", l.Queue.State, "stopped")
	testhelp.Equal(t, "restarts", l.Restarts, []time.Time{gdAt(15, 20, 0), gdAt(15, 30, 0), gdAt(15, 40, 0)})
	if l.LastExit == nil {
		t.Fatal("last exit nil")
	}
	testhelp.Equal(t, "last exit", *l.LastExit, control.Exit{Code: 1, At: gdAt(15, 50, 0), Stderr: "Error: Invalid session ID. Must be a valid UUID.", Restarts: 3})

	m := gdRunning(t)
	m.Exited(4, "", gdAt(15, 20, 0), cfg)
	m.StartChecked(gitrules.Start{Mode: "fresh"}, gdNewID, gdAt(15, 20, 5))
	m.Exited(1, line, gdAt(15, 30, 0), cfg)
	m.StartChecked(gitrules.Start{Mode: "resume", Reason: "dirty tree (1 paths)"}, gdNewID, gdAt(15, 30, 5))
	m.Exited(137, "", gdAt(15, 40, 0), cfg)
	m.StartChecked(gitrules.Start{Mode: "fresh"}, gdNewID, gdAt(15, 40, 5))

	got = m.Exited(1, line, gdAt(16, 25, 0), cfg)
	want = []Action{
		{Kind: "post", Milestone: control.Milestone{Kind: "watchdog", Headline: "P17: session exited (code 1), restarting", Numbers: "restart 3 of 3 this hour · Error: Invalid session ID. Must be a valid UUID."}},
		{Kind: "start_check", Phase: "P17"},
	}
	testhelp.Equal(t, "exit an hour later", got, want)
	testhelp.Equal(t, "restarts", m.Restarts, []time.Time{gdAt(15, 30, 0), gdAt(15, 40, 0), gdAt(16, 25, 0)})
	testhelp.Equal(t, "state", m.State, "starting")
}

func TestRuntimeGuardExample3(t *testing.T) {
	cfg := gdCfg()
	l := gdRunning(t)
	l.StretchUSD = 28
	l.SessionUSD = 0

	got := l.Turn(stream.Result{TotalCostUSD: 1.5}, gdAt(15, 0, 1), cfg)
	testhelp.Equal(t, "first turn", got, []Action(nil))
	testhelp.Equal(t, "session usd", l.SessionUSD, 1.5)

	got = l.Turn(stream.Result{TotalCostUSD: 2.25}, gdAt(16, 0, 0), cfg)
	want := []Action{
		{Kind: "kill"},
		{Kind: "post", Milestone: control.Milestone{Kind: "stop", Headline: "P17: stretch cap", Numbers: "stretch cap $30.00 reached ($30.2500)"}},
	}
	testhelp.Equal(t, "second turn", got, want)
	testhelp.Equal(t, "state", l.State, "waiting")
	if l.Stop == nil {
		t.Fatal("stop nil")
	}
	testhelp.Equal(t, "stop kind", l.Stop.Kind, "cap")

	l2 := gdRunning(t)
	l2.StretchUSD = 28
	l2.SessionUSD = 0
	cfg0 := gdCfg()
	cfg0.StretchUSD = 0
	l2.Turn(stream.Result{TotalCostUSD: 1.5}, gdAt(15, 0, 1), cfg0)
	got = l2.Turn(stream.Result{TotalCostUSD: 2.25}, gdAt(16, 0, 0), cfg0)
	testhelp.Equal(t, "no stretch cap", got, []Action(nil))
}

func TestRuntimeGuardExample4(t *testing.T) {
	cfg := gdCfg()
	l := gdRunning(t)
	lim := stream.Limits{
		FiveHour: stream.Window{Utilization: 0.22, ResetsAt: 1791469200},
		SevenDay: stream.Window{Utilization: 0.6, ResetsAt: 1791853200},
	}

	got := l.Limits(lim, gdAt(15, 0, 1), cfg)
	want := []Action{
		{Kind: "post", Milestone: control.Milestone{Kind: "info", Headline: "P17: weekly usage 60%", Numbers: "resets 2026-10-13T01:00:00Z"}},
	}
	testhelp.Equal(t, "usage alert", got, want)
	testhelp.Equal(t, "alerted", l.Alerted, true)

	got = l.Limits(lim, gdAt(15, 0, 2), cfg)
	testhelp.Equal(t, "second alert", got, []Action(nil))

	l2 := gdRunning(t)
	cfg0 := gdCfg()
	cfg0.UsageAlertPercent = 0
	got = l2.Limits(lim, gdAt(15, 0, 1), cfg0)
	testhelp.Equal(t, "no alert percent", got, []Action(nil))
	testhelp.Equal(t, "alerted stays false", l2.Alerted, false)
}

func TestRuntimeGuardExample5(t *testing.T) {
	cfg := gdCfg()
	l := gdRunning(t)
	lim := stream.Limits{
		FiveHour: stream.Window{Utilization: 1, ResetsAt: 1791469200},
		SevenDay: stream.Window{Utilization: 0.6, ResetsAt: 1791853200},
	}

	got := l.Limits(lim, gdAt(13, 9, 3), cfg)
	want := []Action{
		{Kind: "post", Milestone: control.Milestone{Kind: "watchdog", Headline: "P17: usage limit, paused", Numbers: "until 2026-10-08T14:20:00Z"}},
	}
	testhelp.Equal(t, "paused", got, want)
	testhelp.Equal(t, "state", l.State, "paused")
	testhelp.Equal(t, "paused until", l.PausedUntil, gdAt(14, 20, 0))

	l2 := gdRunning(t)
	both := stream.Limits{
		FiveHour: stream.Window{Utilization: 1, ResetsAt: 1791469200},
		SevenDay: stream.Window{Utilization: 1, ResetsAt: 1791853200},
	}
	got = l2.Limits(both, gdAt(13, 9, 3), cfg)
	want = []Action{
		{Kind: "post", Milestone: control.Milestone{Kind: "watchdog", Headline: "P17: usage limit, paused", Numbers: "until 2026-10-13T01:00:00Z"}},
	}
	testhelp.Equal(t, "both windows", got, want)
	testhelp.Equal(t, "state", l2.State, "paused")
	testhelp.Equal(t, "paused until", l2.PausedUntil, time.Date(2026, 10, 13, 1, 0, 0, 0, time.UTC))
}

func TestRuntimeGuardExample6(t *testing.T) {
	cfg := gdCfg()
	l := gdRunning(t)
	lim := stream.Limits{
		FiveHour: stream.Window{Utilization: 1, ResetsAt: 1791469200},
		SevenDay: stream.Window{Utilization: 0.6, ResetsAt: 1791853200},
	}
	l.Limits(lim, gdAt(13, 9, 3), cfg)

	got := l.Tick(gdAt(14, 19, 59), cfg)
	testhelp.Equal(t, "before reset", got, []Action(nil))

	got = l.Tick(gdAt(14, 20, 0), cfg)
	want := []Action{
		{Kind: "write", Line: "Continue by docs/AUTONOMY.md from where you stopped; the usage window has reset."},
		{Kind: "post", Milestone: control.Milestone{Kind: "watchdog", Headline: "P17: resumed after the limit", Numbers: ""}},
	}
	testhelp.Equal(t, "after reset", got, want)
	testhelp.Equal(t, "state", l.State, "running")
	testhelp.Equal(t, "last event at", l.LastEventAt, gdAt(14, 20, 0))
}

func TestRuntimeGuardExample7(t *testing.T) {
	cfg := gdCfg()
	l := gdRunning(t)
	testhelp.Equal(t, "phase started at", l.PhaseStartedAt, gdAt(15, 0, 1))
	testhelp.Equal(t, "last event at", l.LastEventAt, gdAt(15, 0, 1))

	got := l.Tick(gdAt(15, 29, 0), cfg)
	testhelp.Equal(t, "before the stall", got, []Action(nil))

	got = l.Tick(gdAt(15, 31, 0), cfg)
	want := []Action{
		{Kind: "post", Milestone: control.Milestone{Kind: "watchdog", Headline: "P17: no events for 30 min", Numbers: "one nudge sent"}},
		{Kind: "write", Line: "Nothing has happened for 30 minutes. Check the state of your agents and the run branch, then continue by docs/AUTONOMY.md; post the current state first."},
	}
	testhelp.Equal(t, "nudge", got, want)
	testhelp.Equal(t, "nudged", l.Nudged, true)

	got = l.Tick(gdAt(16, 10, 0), cfg)
	testhelp.Equal(t, "only one nudge", got, []Action(nil))

	l.Event(gdAt(16, 20, 0))
	testhelp.Equal(t, "last event at", l.LastEventAt, gdAt(16, 20, 0))

	got = l.Tick(gdAt(18, 0, 1), cfg)
	want = []Action{
		{Kind: "kill"},
		{Kind: "post", Milestone: control.Milestone{Kind: "stop", Headline: "P17: wall clock cap", Numbers: "wall clock cap 3 h reached"}},
	}
	testhelp.Equal(t, "wall clock cap", got, want)
	testhelp.Equal(t, "state", l.State, "waiting")
	if l.Stop == nil {
		t.Fatal("stop nil")
	}
	testhelp.Equal(t, "stop kind", l.Stop.Kind, "cap")
}

func TestRuntimeGuardExample8(t *testing.T) {
	cfg := gdCfg()
	l := gdRunning(t)
	l.State = "waiting"

	got := l.Exited(0, "bye\n", gdAt(15, 5, 0), cfg)
	testhelp.Equal(t, "exit while waiting", got, []Action(nil))

	got = l.Tick(gdAt(15, 5, 0), cfg)
	testhelp.Equal(t, "tick while waiting", got, []Action(nil))

	got = l.Turn(stream.Result{TotalCostUSD: 99}, gdAt(15, 5, 0), cfg)
	testhelp.Equal(t, "turn while waiting", got, []Action(nil))

	idle := New("demo", gdQueue(t))
	got = idle.Limits(stream.Limits{FiveHour: stream.Window{Utilization: 1, ResetsAt: 1791469200}}, gdAt(15, 5, 0), cfg)
	testhelp.Equal(t, "limits while idle", got, []Action(nil))

	testhelp.Equal(t, "state", l.State, "waiting")
	testhelp.Equal(t, "restarts", l.Restarts, []time.Time(nil))
	testhelp.Equal(t, "last exit", l.LastExit, (*control.Exit)(nil))
	testhelp.Equal(t, "state", idle.State, "idle")
}

func TestRuntimeGuardExample9(t *testing.T) {
	cfg := gdCfg()

	l := gdRunning(t)
	got := l.Exited(2, "\n\n node: bad option --foo \n(use node --help for usage)\n", gdAt(15, 5, 0), cfg)
	want := []Action{
		{Kind: "post", Milestone: control.Milestone{Kind: "watchdog", Headline: "P17: session exited (code 2), restarting", Numbers: "restart 1 of 3 this hour · node: bad option --foo"}},
		{Kind: "start_check", Phase: "P17"},
	}
	testhelp.Equal(t, "first stderr", got, want)
	if l.LastExit == nil {
		t.Fatal("last exit nil")
	}
	testhelp.Equal(t, "first stderr line", l.LastExit.Stderr, "node: bad option --foo")

	l2 := gdRunning(t)
	got = l2.Exited(3, strings.Repeat("x", 250)+"\nsecond\n", gdAt(15, 6, 0), cfg)
	want = []Action{
		{Kind: "post", Milestone: control.Milestone{Kind: "watchdog", Headline: "P17: session exited (code 3), restarting", Numbers: "restart 1 of 3 this hour · " + strings.Repeat("x", 200)}},
		{Kind: "start_check", Phase: "P17"},
	}
	testhelp.Equal(t, "long stderr", got, want)

	l3 := gdRunning(t)
	cfg0 := Config{MaxResumesPerHour: 0, StallMinutes: 30, StretchUSD: 30, UsageAlertPercent: 50}
	got = l3.Exited(-3, "claude: start fork/exec /opt/claude/bin/claude: no such file or directory", gdAt(15, 7, 0), cfg0)
	want = []Action{
		{Kind: "post", Milestone: control.Milestone{Kind: "stop", Headline: "P17: session keeps exiting", Numbers: "exited 1 times within an hour (last code -3): claude: start fork/exec /opt/claude/bin/claude: no such file or directory"}},
	}
	testhelp.Equal(t, "no resumes", got, want)
	testhelp.Equal(t, "state", l3.State, "waiting")
	if l3.Stop == nil {
		t.Fatal("stop nil")
	}
	testhelp.Equal(t, "stop kind", l3.Stop.Kind, "crash")
	testhelp.Equal(t, "restarts", l3.Restarts, []time.Time(nil))
	if l3.LastExit == nil {
		t.Fatal("last exit nil")
	}
	testhelp.Equal(t, "last exit", *l3.LastExit, control.Exit{Code: -3, At: gdAt(15, 7, 0), Stderr: "claude: start fork/exec /opt/claude/bin/claude: no such file or directory", Restarts: 0})
}
