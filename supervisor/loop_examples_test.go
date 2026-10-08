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

func plQueue(t testing.TB) queue.Queue {
	t.Helper()
	data, err := os.ReadFile("../tests/fixtures/plan/queue-3.json")
	if err != nil {
		t.Fatal(err)
	}
	var plan queue.Plan
	if err := json.Unmarshal(data, &plan); err != nil {
		t.Fatal(err)
	}
	q, err := queue.Load(plan, queue.Caps{ClaudeUSD: 30, Hours: 3, ExecutorUSD: 5})
	if err != nil {
		t.Fatal(err)
	}
	return q
}

func plIDs() func() string {
	n := 0
	return func() string {
		n++
		return "id-" + strconv.Itoa(n)
	}
}

func plFixedID(id string) func() string {
	return func() string { return id }
}

func plTime(h, m, s int) time.Time {
	return time.Date(2026, 10, 8, h, m, s, 0, time.UTC)
}

func plError(t testing.TB, what string, err, target error, text string) {
	t.Helper()
	if err == nil {
		t.Errorf("%s: got nil, want %v", what, target)
		return
	}
	if !errors.Is(err, target) {
		t.Errorf("%s: got %v, want %v", what, err, target)
	}
	testhelp.Equal(t, what+" text", err.Error(), text)
}

func plErrorText(t testing.TB, what string, err error, text string) {
	t.Helper()
	if err == nil {
		t.Errorf("%s: got nil, want %q", what, text)
		return
	}
	testhelp.Equal(t, what, err.Error(), text)
}

func plStartingAt(t testing.TB, index int, phase, sessionID string) *Loop {
	t.Helper()
	q := plQueue(t)
	q.Index = index
	q.State = "running"
	l := New("demo", q)
	l.State = "starting"
	l.Phase = phase
	l.SessionID = sessionID
	return l
}

func plRunningAt(t testing.TB, index int, phase string, sessionUSD, stretchUSD float64) *Loop {
	t.Helper()
	q := plQueue(t)
	q.Index = index
	q.State = "running"
	l := New("demo", q)
	l.State = "running"
	l.Phase = phase
	l.SessionID = "id-x"
	l.SessionUSD = sessionUSD
	l.StretchUSD = stretchUSD
	return l
}

func plRunning(t testing.TB) *Loop {
	t.Helper()
	q := plQueue(t)
	l := New("demo", q)
	if _, err := l.Begin(plTime(15, 0, 0)); err != nil {
		t.Fatal(err)
	}
	l.StartChecked(gitrules.Start{Mode: "fresh", Local: "aaa111", Remote: "aaa111"}, plIDs(), plTime(15, 0, 1))
	return l
}

func TestPhaseLoopExample1(t *testing.T) {
	l := New("demo", plQueue(t))

	actions, err := l.Begin(plTime(15, 0, 0))
	if err != nil {
		t.Fatal(err)
	}
	testhelp.Equal(t, "actions", actions, []Action{{Kind: "start_check", Phase: "P17"}})
	testhelp.Equal(t, "state", l.State, "starting")
	testhelp.Equal(t, "phase", l.Phase, "P17")
	testhelp.Equal(t, "queue state", l.Queue.State, "running")

	again, err2 := l.Begin(plTime(15, 0, 0))
	testhelp.Equal(t, "second begin actions", again, []Action(nil))
	plErrorText(t, "second begin", err2, `queue: cannot start in state "running"`)
	testhelp.Equal(t, "state after", l.State, "starting")
	testhelp.Equal(t, "phase after", l.Phase, "P17")
	testhelp.Equal(t, "queue state after", l.Queue.State, "running")
}

func TestPhaseLoopExample2(t *testing.T) {
	l := New("demo", plQueue(t))
	if _, err := l.Begin(plTime(15, 0, 0)); err != nil {
		t.Fatal(err)
	}

	actions := l.StartChecked(gitrules.Start{Mode: "fresh", Local: "aaa111", Remote: "aaa111"}, plIDs(), plTime(15, 0, 1))
	want := []Action{
		{Kind: "spawn", Phase: "P17", SessionID: "id-1", Resume: false, BudgetUSD: 30},
		{Kind: "first_line", Line: "/morph-orchestrator P17"},
		{Kind: "post", Milestone: control.Milestone{Kind: "start", Headline: "P17 started", Numbers: "session id-1 · cap $30 · 3 h"}},
	}
	testhelp.Equal(t, "actions", actions, want)
	testhelp.Equal(t, "state", l.State, "running")
	testhelp.Equal(t, "session id", l.SessionID, "id-1")
	testhelp.Equal(t, "phase started at", l.PhaseStartedAt, plTime(15, 0, 1))
}

func TestPhaseLoopExample3(t *testing.T) {
	l := plStartingAt(t, 1, "P18", "")
	actions := l.StartChecked(gitrules.Start{Mode: "resume", Reason: "local ahead of origin/main"}, plFixedID("id-7"), plTime(15, 0, 0))
	want := []Action{
		{Kind: "spawn", Phase: "P18", SessionID: "id-7", Resume: true, BudgetUSD: 12},
		{Kind: "post", Milestone: control.Milestone{Kind: "start", Headline: "P18 resumed", Numbers: "session id-7 · local ahead of origin/main"}},
	}
	testhelp.Equal(t, "actions", actions, want)
	testhelp.Equal(t, "state", l.State, "running")
	testhelp.Equal(t, "session id", l.SessionID, "id-7")

	l2 := plStartingAt(t, 1, "P18", "id-3")
	actions2 := l2.StartChecked(gitrules.Start{Mode: "resume", Reason: "local ahead of origin/main"}, plFixedID("id-7"), plTime(15, 0, 0))
	want2 := []Action{
		{Kind: "spawn", Phase: "P18", SessionID: "id-3", Resume: true, BudgetUSD: 12},
		{Kind: "post", Milestone: control.Milestone{Kind: "start", Headline: "P18 resumed", Numbers: "session id-3 · local ahead of origin/main"}},
	}
	testhelp.Equal(t, "actions kept", actions2, want2)
	testhelp.Equal(t, "session id kept", l2.SessionID, "id-3")
}

func TestPhaseLoopExample4(t *testing.T) {
	l := New("demo", plQueue(t))
	if _, err := l.Begin(plTime(15, 0, 0)); err != nil {
		t.Fatal(err)
	}

	reason := "local aaa111 and origin/main ccc333 diverged"
	actions := l.StartChecked(gitrules.Start{Mode: "diverged", Reason: reason}, plIDs(), plTime(15, 2, 0))
	want := []Action{
		{Kind: "post", Milestone: control.Milestone{Kind: "stop", Headline: "P17 not started: git", Numbers: reason}},
	}
	testhelp.Equal(t, "actions", actions, want)
	testhelp.Equal(t, "state", l.State, "waiting")
	testhelp.Equal(t, "stop", *l.Stop, control.Stop{Kind: "git", Phase: "P17", Reason: reason, At: plTime(15, 2, 0)})
	testhelp.Equal(t, "queue state", l.Queue.State, "stopped")
}

func TestPhaseLoopExample5(t *testing.T) {
	l := plRunning(t)

	doneActions, err := l.PhaseDone("P17", "P18")
	if err != nil {
		t.Fatal(err)
	}
	testhelp.Equal(t, "phase done actions", doneActions, []Action{{Kind: "end_check", Phase: "P17"}})

	endActions := l.EndChecked(gitrules.End{OK: true, Local: "ddd", Remote: "ddd", Problems: []string{}}, plTime(15, 40, 0))
	want := []Action{
		{Kind: "kill"},
		{Kind: "start_check", Phase: "P18"},
	}
	testhelp.Equal(t, "end checked actions", endActions, want)
	testhelp.Equal(t, "state", l.State, "starting")
	testhelp.Equal(t, "phase", l.Phase, "P18")
	testhelp.Equal(t, "session id", l.SessionID, "")
	testhelp.Equal(t, "queue index", l.Queue.Index, 1)
	testhelp.Equal(t, "queue state", l.Queue.State, "running")
}

func TestPhaseLoopExample6(t *testing.T) {
	l := plRunning(t)

	actions, err := l.PhaseDone("P16", "P17")
	testhelp.Equal(t, "actions", actions, []Action(nil))
	plError(t, "PhaseDone running", err, ErrPhaseMismatch, `phase mismatch: phase_done "P16" while running "P17"`)

	l.State = "waiting"
	actions2, err2 := l.PhaseDone("P17", "P18")
	testhelp.Equal(t, "actions waiting", actions2, []Action(nil))
	plError(t, "PhaseDone waiting", err2, ErrPhaseMismatch, `phase mismatch: phase_done "P17" while waiting "P17"`)
}

func TestPhaseLoopExample7(t *testing.T) {
	l := plRunning(t)
	l.SessionUSD = 1.92

	reason := "not pushed: HEAD bbb222, origin/main aaa111"
	actions := l.EndChecked(gitrules.End{OK: false, Problems: []string{reason}}, plTime(16, 0, 0))
	want := []Action{
		{Kind: "post", Milestone: control.Milestone{Kind: "stop", Headline: "P17 not pushed", Numbers: reason}},
	}
	testhelp.Equal(t, "actions", actions, want)
	testhelp.Equal(t, "state", l.State, "waiting")
	testhelp.Equal(t, "stop", *l.Stop, control.Stop{Kind: "not_pushed", Phase: "P17", Reason: reason, At: plTime(16, 0, 0)})
	testhelp.Equal(t, "stretch", l.StretchUSD, 0.0)
	testhelp.Equal(t, "queue state", l.Queue.State, "stopped")

	contActions, err := l.Continue(plTime(16, 5, 0))
	if err != nil {
		t.Fatal(err)
	}
	testhelp.Equal(t, "continue actions", contActions, []Action{{Kind: "end_check", Phase: "P17"}})
	testhelp.Equal(t, "state after continue", l.State, "running")
	testhelp.Equal(t, "stop after continue", l.Stop, (*control.Stop)(nil))
	testhelp.Equal(t, "queue state after continue", l.Queue.State, "running")
	testhelp.Equal(t, "queue index after continue", l.Queue.Index, 0)

	endActions := l.EndChecked(gitrules.End{OK: true, Problems: []string{}}, plTime(16, 10, 0))
	wantEnd := []Action{
		{Kind: "kill"},
		{Kind: "start_check", Phase: "P18"},
	}
	testhelp.Equal(t, "end actions", endActions, wantEnd)
	testhelp.Equal(t, "stretch after end", l.StretchUSD, 1.92)
	testhelp.Equal(t, "session usd after end", l.SessionUSD, 0.0)
}

func TestPhaseLoopExample8(t *testing.T) {
	l := plRunningAt(t, 1, "P18", 0.5, 1.0)

	endActions := l.EndChecked(gitrules.End{OK: true, Problems: []string{}}, plTime(17, 0, 0))
	want := []Action{
		{Kind: "kill"},
		{Kind: "post", Milestone: control.Milestone{Kind: "stop", Headline: "P18 done: stop after smoke", Numbers: "next P19 · waiting for continue"}},
	}
	testhelp.Equal(t, "end actions", endActions, want)
	testhelp.Equal(t, "state", l.State, "waiting")
	testhelp.Equal(t, "stop kind", l.Stop.Kind, "smoke")
	testhelp.Equal(t, "stretch", l.StretchUSD, 1.5)

	contActions, err := l.Continue(plTime(17, 30, 0))
	if err != nil {
		t.Fatal(err)
	}
	testhelp.Equal(t, "continue actions", contActions, []Action{{Kind: "start_check", Phase: "P19"}})
	testhelp.Equal(t, "state after continue", l.State, "starting")
	testhelp.Equal(t, "phase after continue", l.Phase, "P19")
	testhelp.Equal(t, "stop after continue", l.Stop, (*control.Stop)(nil))

	l.State = "running"
	endActions2 := l.EndChecked(gitrules.End{OK: true, Problems: []string{}}, plTime(18, 0, 0))
	wantEnd := []Action{
		{Kind: "kill"},
		{Kind: "post", Milestone: control.Milestone{Kind: "end", Headline: "P19 done: plan complete", Numbers: "stretch $1.5000"}},
	}
	testhelp.Equal(t, "end actions plan", endActions2, wantEnd)
	testhelp.Equal(t, "state plan", l.State, "waiting")
	testhelp.Equal(t, "stop kind plan", l.Stop.Kind, "end")
	testhelp.Equal(t, "queue state plan", l.Queue.State, "done")

	contActions2, err2 := l.Continue(plTime(18, 5, 0))
	testhelp.Equal(t, "continue done actions", contActions2, []Action(nil))
	plErrorText(t, "continue done", err2, `queue: nothing to continue (state "done")`)
}

func TestPhaseLoopExample9(t *testing.T) {
	l := plRunning(t)
	actions := l.WaitOperator("gate", "forecast $1.4 over the cap", "", plTime(15, 10, 0))
	want := []Action{
		{Kind: "post", Milestone: control.Milestone{Kind: "stop", Headline: "P17: wait_operator gate", Numbers: "forecast $1.4 over the cap"}},
	}
	testhelp.Equal(t, "actions", actions, want)
	testhelp.Equal(t, "state", l.State, "waiting")
	testhelp.Equal(t, "stop", *l.Stop, control.Stop{Kind: "gate", Phase: "P17", Reason: "forecast $1.4 over the cap", At: plTime(15, 10, 0)})

	l2 := plRunning(t)
	actions2 := l2.WaitOperator("emergency", "card x red after fix", "https://github.com/o/r/issues/12", plTime(15, 20, 0))
	want2 := []Action{
		{Kind: "post", Milestone: control.Milestone{Kind: "stop", Headline: "P17: wait_operator emergency", Numbers: "card x red after fix · https://github.com/o/r/issues/12"}},
	}
	testhelp.Equal(t, "actions emergency", actions2, want2)
	testhelp.Equal(t, "issue url", l2.Stop.IssueURL, "https://github.com/o/r/issues/12")

	restartActions, err := l.Restart(plTime(15, 30, 0))
	testhelp.Equal(t, "restart waiting actions", restartActions, []Action(nil))
	if !errors.Is(err, control.ErrNoSession) {
		t.Errorf("restart waiting: got %v, want ErrNoSession", err)
	}

	l3 := plRunning(t)
	restartActions2, err2 := l3.Restart(plTime(15, 30, 0))
	if err2 != nil {
		t.Fatal(err2)
	}
	wantRestart := []Action{
		{Kind: "kill"},
		{Kind: "start_check", Phase: "P17"},
	}
	testhelp.Equal(t, "restart running actions", restartActions2, wantRestart)
	testhelp.Equal(t, "restart state", l3.State, "starting")
	testhelp.Equal(t, "restart session id", l3.SessionID, "")
}
