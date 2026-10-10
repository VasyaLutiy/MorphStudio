package supervisor

import (
	"encoding/json"
	"errors"
	"os"
	"strconv"
	"testing"
	"time"

	"morphstudio/control"
	"morphstudio/gitrules"
	"morphstudio/internal/testhelp"
	"morphstudio/queue"
)

func plQueue(t *testing.T) queue.Queue {
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

func plStarting(t *testing.T) *Loop {
	t.Helper()
	l := New("demo", plQueue(t))
	if _, err := l.Begin(time.Date(2026, 10, 8, 15, 0, 0, 0, time.UTC)); err != nil {
		t.Fatal(err)
	}
	return l
}

func plRunning(t *testing.T) *Loop {
	t.Helper()
	l := plStarting(t)
	stamp := time.Date(2026, 10, 8, 14, 50, 0, 0, time.UTC)
	l.Restarts = []time.Time{stamp}
	l.LastExit = &control.Exit{Code: 1, At: stamp, Stderr: "boom", Restarts: 1}
	calls := 0
	ids := func() string {
		calls++
		return "id-" + strconv.Itoa(calls)
	}
	l.StartChecked(gitrules.Start{Mode: "fresh", Local: "aaa111", Remote: "aaa111"}, ids, time.Date(2026, 10, 8, 15, 0, 1, 0, time.UTC))
	return l
}

func TestPhaseLoopExample1(t *testing.T) {
	q := plQueue(t)
	testhelp.Equal(t, "queue state", q.State, "queued")
	l := New("demo", q)
	stamp := time.Date(2026, 10, 8, 14, 0, 0, 0, time.UTC)
	l.Restarts = []time.Time{stamp}
	l.LastExit = &control.Exit{Code: 7, At: stamp, Stderr: "old", Restarts: 1}
	calls := 0
	ids := func() string {
		calls++
		return "id-" + strconv.Itoa(calls)
	}
	_ = ids

	got, err := l.Begin(time.Date(2026, 10, 8, 15, 0, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("Begin: %v", err)
	}
	testhelp.Equal(t, "actions", got, []Action{{Kind: "start_check", Phase: "P17"}})
	testhelp.Equal(t, "state", l.State, "starting")
	testhelp.Equal(t, "phase", l.Phase, "P17")
	testhelp.Equal(t, "queue state", l.Queue.State, "running")
	testhelp.Equal(t, "restarts", l.Restarts, []time.Time(nil))
	if l.LastExit != nil {
		t.Errorf("LastExit: got %#v, want nil", *l.LastExit)
	}

	got2, err2 := l.Begin(time.Date(2026, 10, 8, 15, 0, 1, 0, time.UTC))
	testhelp.Equal(t, "second begin actions", got2, []Action(nil))
	if err2 == nil {
		t.Fatal("second Begin: expected error")
	}
	testhelp.Equal(t, "second begin error", err2.Error(), `queue: cannot start in state "running"`)
	testhelp.Equal(t, "state after second begin", l.State, "starting")
	testhelp.Equal(t, "phase after second begin", l.Phase, "P17")
	testhelp.Equal(t, "queue state after second begin", l.Queue.State, "running")
}

func TestPhaseLoopExample2(t *testing.T) {
	l := plStarting(t)
	stamp := time.Date(2026, 10, 8, 14, 50, 0, 0, time.UTC)
	exit := control.Exit{Code: 1, At: stamp, Stderr: "boom", Restarts: 1}
	l.Restarts = []time.Time{stamp}
	l.LastExit = &exit
	now := time.Date(2026, 10, 8, 15, 0, 1, 0, time.UTC)
	calls := 0
	ids := func() string {
		calls++
		return "id-" + strconv.Itoa(calls)
	}
	got := l.StartChecked(gitrules.Start{Mode: "fresh", Local: "aaa111", Remote: "aaa111"}, ids, now)
	want := []Action{
		{Kind: "spawn", Phase: "P17", SessionID: "id-1", Resume: false, BudgetUSD: 30},
		{Kind: "first_line", Line: "/morph-orchestrator P17"},
		{Kind: "post", Milestone: control.Milestone{Kind: "start", Headline: "P17 started", Numbers: "session id-1 · cap $30 · 3 h"}},
	}
	testhelp.Equal(t, "actions", got, want)
	testhelp.Equal(t, "state", l.State, "running")
	testhelp.Equal(t, "session id", l.SessionID, "id-1")
	testhelp.Equal(t, "phase started at", l.PhaseStartedAt, now)
	testhelp.Equal(t, "restarts", l.Restarts, []time.Time{stamp})
	if l.LastExit == nil {
		t.Fatal("LastExit is nil")
	}
	testhelp.Equal(t, "last exit", *l.LastExit, exit)
}

func TestPhaseLoopExample3(t *testing.T) {
	q := plQueue(t)
	q.Index = 1
	q.State = "running"
	l := New("demo", q)
	l.State = "starting"
	l.Phase = "P18"
	now := time.Date(2026, 10, 8, 15, 45, 0, 0, time.UTC)
	ids := func() string { return "id-7" }
	got := l.StartChecked(gitrules.Start{Mode: "resume", Reason: "local ahead of origin/main", Local: "bbb222", Remote: "aaa111"}, ids, now)
	want := []Action{
		{Kind: "spawn", Phase: "P18", SessionID: "id-7", Resume: false, BudgetUSD: 12},
		{Kind: "first_line", Line: "/morph-orchestrator P18"},
		{Kind: "post", Milestone: control.Milestone{Kind: "start", Headline: "P18 resumed on a fresh session", Numbers: "session id-7 · cap $12 · 3 h · local ahead of origin/main"}},
	}
	testhelp.Equal(t, "actions", got, want)
	testhelp.Equal(t, "state", l.State, "running")
	testhelp.Equal(t, "session id", l.SessionID, "id-7")
	testhelp.Equal(t, "phase started at", l.PhaseStartedAt, now)
	testhelp.Equal(t, "last event at", l.LastEventAt, now)
}

func TestPhaseLoopExample4(t *testing.T) {
	l := plStarting(t)
	now := time.Date(2026, 10, 8, 15, 2, 0, 0, time.UTC)
	ids := func() string { return "id-x" }
	got := l.StartChecked(gitrules.Start{Mode: "diverged", Reason: "local aaa111 and origin/main ccc333 diverged"}, ids, now)
	want := []Action{
		{Kind: "post", Milestone: control.Milestone{Kind: "stop", Headline: "P17 not started: git", Numbers: "local aaa111 and origin/main ccc333 diverged"}},
	}
	testhelp.Equal(t, "actions", got, want)
	testhelp.Equal(t, "state", l.State, "waiting")
	if l.Stop == nil {
		t.Fatal("Stop is nil")
	}
	testhelp.Equal(t, "stop", *l.Stop, control.Stop{Kind: "git", Phase: "P17", Reason: "local aaa111 and origin/main ccc333 diverged", At: now})
	testhelp.Equal(t, "queue state", l.Queue.State, "stopped")
}

func TestPhaseLoopExample5(t *testing.T) {
	l := plRunning(t)
	got, err := l.PhaseDone("P17", "P18")
	if err != nil {
		t.Fatalf("PhaseDone: %v", err)
	}
	testhelp.Equal(t, "phase done actions", got, []Action{{Kind: "end_check", Phase: "P17"}})

	now := time.Date(2026, 10, 8, 15, 40, 0, 0, time.UTC)
	got2 := l.EndChecked(gitrules.End{OK: true, Local: "ddd", Remote: "ddd", Problems: nil}, now)
	want := []Action{
		{Kind: "kill"},
		{Kind: "start_check", Phase: "P18"},
	}
	testhelp.Equal(t, "end checked actions", got2, want)
	testhelp.Equal(t, "state", l.State, "starting")
	testhelp.Equal(t, "phase", l.Phase, "P18")
	testhelp.Equal(t, "session id", l.SessionID, "")
	testhelp.Equal(t, "queue index", l.Queue.Index, 1)
	testhelp.Equal(t, "queue state", l.Queue.State, "running")
}

func TestPhaseLoopExample6(t *testing.T) {
	l := plRunning(t)
	got, err := l.PhaseDone("P16", "P17")
	testhelp.Equal(t, "actions", got, []Action(nil))
	if err == nil {
		t.Fatal("expected error")
	}
	if !errors.Is(err, ErrPhaseMismatch) {
		t.Errorf("errors.Is: got false, want true")
	}
	testhelp.Equal(t, "error text", err.Error(), `phase mismatch: phase_done "P16" while running "P17"`)

	l2 := plRunning(t)
	l2.State = "waiting"
	got2, err2 := l2.PhaseDone("P17", "P18")
	testhelp.Equal(t, "actions 2", got2, []Action(nil))
	if err2 == nil {
		t.Fatal("expected error")
	}
	if !errors.Is(err2, ErrPhaseMismatch) {
		t.Errorf("errors.Is: got false, want true")
	}
	testhelp.Equal(t, "error text 2", err2.Error(), `phase mismatch: phase_done "P17" while waiting "P17"`)
}

func TestPhaseLoopExample7(t *testing.T) {
	l := plRunning(t)
	l.SessionUSD = 1.92
	now := time.Date(2026, 10, 8, 16, 0, 0, 0, time.UTC)
	got := l.EndChecked(gitrules.End{OK: false, Problems: []string{"not pushed: HEAD bbb222, origin/main aaa111"}}, now)
	want := []Action{
		{Kind: "post", Milestone: control.Milestone{Kind: "stop", Headline: "P17 not pushed", Numbers: "not pushed: HEAD bbb222, origin/main aaa111"}},
	}
	testhelp.Equal(t, "actions", got, want)
	testhelp.Equal(t, "state", l.State, "waiting")
	if l.Stop == nil {
		t.Fatal("Stop is nil")
	}
	testhelp.Equal(t, "stop", *l.Stop, control.Stop{Kind: "not_pushed", Phase: "P17", Reason: "not pushed: HEAD bbb222, origin/main aaa111", At: now})
	testhelp.Equal(t, "stretch usd", l.StretchUSD, 0.0)
	testhelp.Equal(t, "queue state", l.Queue.State, "stopped")

	got2, err := l.Continue(time.Date(2026, 10, 8, 16, 5, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("Continue: %v", err)
	}
	testhelp.Equal(t, "continue actions", got2, []Action{{Kind: "end_check", Phase: "P17"}})
	testhelp.Equal(t, "state after continue", l.State, "running")
	if l.Stop != nil {
		t.Errorf("Stop: got %#v, want nil", *l.Stop)
	}
	testhelp.Equal(t, "queue state after continue", l.Queue.State, "running")
	testhelp.Equal(t, "queue index after continue", l.Queue.Index, 0)

	got3 := l.EndChecked(gitrules.End{OK: true, Problems: nil}, time.Date(2026, 10, 8, 16, 10, 0, 0, time.UTC))
	want3 := []Action{
		{Kind: "kill"},
		{Kind: "start_check", Phase: "P18"},
	}
	testhelp.Equal(t, "end actions", got3, want3)
	testhelp.Equal(t, "stretch usd after end", l.StretchUSD, 1.92)
	testhelp.Equal(t, "session usd after end", l.SessionUSD, 0.0)
}

func TestPhaseLoopExample8(t *testing.T) {
	q := plQueue(t)
	q.Index = 1
	q.State = "running"
	l := New("demo", q)
	l.State = "running"
	l.Phase = "P18"
	l.SessionID = "id-18"
	l.SessionUSD = 0.5
	l.StretchUSD = 1.0

	got := l.EndChecked(gitrules.End{OK: true, Problems: nil}, time.Date(2026, 10, 8, 17, 0, 0, 0, time.UTC))
	want := []Action{
		{Kind: "kill"},
		{Kind: "post", Milestone: control.Milestone{Kind: "stop", Headline: "P18 done: stop after smoke", Numbers: "next P19 · waiting for continue"}},
	}
	testhelp.Equal(t, "end actions", got, want)
	testhelp.Equal(t, "state", l.State, "waiting")
	if l.Stop == nil {
		t.Fatal("Stop is nil")
	}
	testhelp.Equal(t, "stop kind", l.Stop.Kind, "smoke")
	testhelp.Equal(t, "stretch usd", l.StretchUSD, 1.5)

	got2, err := l.Continue(time.Date(2026, 10, 8, 17, 30, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("Continue: %v", err)
	}
	testhelp.Equal(t, "continue actions", got2, []Action{{Kind: "start_check", Phase: "P19"}})
	testhelp.Equal(t, "state after continue", l.State, "starting")
	testhelp.Equal(t, "phase after continue", l.Phase, "P19")

	l.State = "running"
	got3 := l.EndChecked(gitrules.End{OK: true, Problems: nil}, time.Date(2026, 10, 8, 18, 0, 0, 0, time.UTC))
	want3 := []Action{
		{Kind: "kill"},
		{Kind: "post", Milestone: control.Milestone{Kind: "end", Headline: "P19 done: plan complete", Numbers: "stretch $1.5000"}},
	}
	testhelp.Equal(t, "p19 end actions", got3, want3)
	testhelp.Equal(t, "state after p19 end", l.State, "waiting")
	if l.Stop == nil {
		t.Fatal("Stop is nil")
	}
	testhelp.Equal(t, "stop kind after p19 end", l.Stop.Kind, "end")
	testhelp.Equal(t, "queue state after p19 end", l.Queue.State, "done")

	_, cerr := l.Continue(time.Date(2026, 10, 8, 18, 30, 0, 0, time.UTC))
	if cerr == nil {
		t.Fatal("expected error")
	}
	testhelp.Equal(t, "continue error", cerr.Error(), `queue: nothing to continue (state "done")`)
}

func TestPhaseLoopExample9(t *testing.T) {
	l := plRunning(t)
	now := time.Date(2026, 10, 8, 15, 10, 0, 0, time.UTC)
	got := l.WaitOperator("gate", "forecast $1.4 over the cap", "", now)
	want := []Action{
		{Kind: "post", Milestone: control.Milestone{Kind: "stop", Headline: "P17: wait_operator gate", Numbers: "forecast $1.4 over the cap"}},
	}
	testhelp.Equal(t, "actions", got, want)
	testhelp.Equal(t, "state", l.State, "waiting")
	if l.Stop == nil {
		t.Fatal("Stop is nil")
	}
	testhelp.Equal(t, "stop", *l.Stop, control.Stop{Kind: "gate", Phase: "P17", Reason: "forecast $1.4 over the cap", At: now})

	l2 := plRunning(t)
	now2 := time.Date(2026, 10, 8, 15, 20, 0, 0, time.UTC)
	got2 := l2.WaitOperator("emergency", "card x red after fix", "https://github.com/o/r/issues/12", now2)
	want2 := []Action{
		{Kind: "post", Milestone: control.Milestone{Kind: "stop", Headline: "P17: wait_operator emergency", Numbers: "card x red after fix · https://github.com/o/r/issues/12"}},
	}
	testhelp.Equal(t, "emergency actions", got2, want2)
	if l2.Stop == nil {
		t.Fatal("Stop is nil")
	}
	testhelp.Equal(t, "stop issue url", l2.Stop.IssueURL, "https://github.com/o/r/issues/12")
	testhelp.Equal(t, "emergency stop", *l2.Stop, control.Stop{Kind: "emergency", Phase: "P17", Reason: "card x red after fix", IssueURL: "https://github.com/o/r/issues/12", At: now2})

	_, rerr := l.Restart(now)
	if rerr == nil {
		t.Fatal("expected Restart error")
	}
	if !errors.Is(rerr, control.ErrNoSession) {
		t.Errorf("errors.Is: got false, want true")
	}

	l3 := plRunning(t)
	stamp := time.Date(2026, 10, 8, 15, 5, 0, 0, time.UTC)
	exit := control.Exit{Code: 3, At: stamp, Stderr: "", Restarts: 1}
	l3.Restarts = []time.Time{stamp}
	l3.LastExit = &exit
	got3, err3 := l3.Restart(now)
	if err3 != nil {
		t.Fatalf("Restart: %v", err3)
	}
	testhelp.Equal(t, "restart actions", got3, []Action{
		{Kind: "kill"},
		{Kind: "start_check", Phase: "P17"},
	})
	testhelp.Equal(t, "state after restart", l3.State, "starting")
	testhelp.Equal(t, "session id after restart", l3.SessionID, "")
	testhelp.Equal(t, "restarts after restart", l3.Restarts, []time.Time(nil))
	if l3.LastExit == nil {
		t.Fatal("LastExit is nil")
	}
	testhelp.Equal(t, "last exit after restart", *l3.LastExit, exit)
}

func TestPhaseLoopExample10(t *testing.T) {
	l := plRunning(t)
	l.State = "starting"
	now := time.Date(2026, 10, 8, 15, 21, 0, 0, time.UTC)
	calls := 0
	ids := func() string {
		calls++
		return "id-9"
	}
	got := l.StartChecked(gitrules.Start{Mode: "resume", Reason: "dirty tree (1 paths)"}, ids, now)
	want := []Action{
		{Kind: "spawn", Phase: "P17", SessionID: "id-1", Resume: true, BudgetUSD: 30},
		{Kind: "post", Milestone: control.Milestone{Kind: "start", Headline: "P17 resumed", Numbers: "session id-1 · dirty tree (1 paths)"}},
	}
	testhelp.Equal(t, "actions", got, want)
	testhelp.Equal(t, "session id", l.SessionID, "id-1")
	testhelp.Equal(t, "ids calls", calls, 0)
	testhelp.Equal(t, "state", l.State, "running")
	testhelp.Equal(t, "phase started at", l.PhaseStartedAt, now)
	testhelp.Equal(t, "last event at", l.LastEventAt, now)
}

func TestPhaseLoopExample11(t *testing.T) {
	l := plRunning(t)
	got, err := l.Restart(time.Date(2026, 10, 8, 15, 30, 0, 0, time.UTC))
	if err != nil {
		t.Fatalf("Restart: %v", err)
	}
	testhelp.Equal(t, "restart actions", got, []Action{
		{Kind: "kill"},
		{Kind: "start_check", Phase: "P17"},
	})
	testhelp.Equal(t, "session id after restart", l.SessionID, "")

	now := time.Date(2026, 10, 8, 15, 30, 5, 0, time.UTC)
	calls := 0
	ids := func() string {
		calls++
		return "id-2"
	}
	got2 := l.StartChecked(gitrules.Start{Mode: "resume", Reason: "on branch morph/20261008-083312", Branch: "morph/20261008-083312"}, ids, now)
	want2 := []Action{
		{Kind: "spawn", Phase: "P17", SessionID: "id-2", Resume: false, BudgetUSD: 30},
		{Kind: "first_line", Line: "/morph-orchestrator P17"},
		{Kind: "post", Milestone: control.Milestone{Kind: "start", Headline: "P17 resumed on a fresh session", Numbers: "session id-2 · cap $30 · 3 h · on branch morph/20261008-083312"}},
	}
	testhelp.Equal(t, "actions", got2, want2)
	testhelp.Equal(t, "state", l.State, "running")
	testhelp.Equal(t, "session id after start", l.SessionID, "id-2")
	testhelp.Equal(t, "ids calls", calls, 1)
	testhelp.Equal(t, "phase started at", l.PhaseStartedAt, now)
}
