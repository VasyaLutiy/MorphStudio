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
	i := 0
	return func() string {
		i++
		return "id-" + strconv.Itoa(i)
	}
}

func plRunningP17(t *testing.T) *Loop {
	t.Helper()
	l := New("demo", plQueue(t))
	if _, err := l.Begin(time.Date(2026, 10, 8, 15, 0, 0, 0, time.UTC)); err != nil {
		t.Fatal(err)
	}
	l.StartChecked(gitrules.Start{Mode: "fresh", Local: "aaa111", Remote: "aaa111"}, plIDs(), time.Date(2026, 10, 8, 15, 0, 1, 0, time.UTC))
	return l
}

func plStartingP18(t *testing.T) *Loop {
	t.Helper()
	l := New("demo", plQueue(t))
	l.Queue.Index = 1
	l.Queue.State = "running"
	l.State = "starting"
	l.Phase = "P18"
	return l
}

func TestPhaseLoopExample1(t *testing.T) {
	l := New("demo", plQueue(t))
	l.Restarts = []time.Time{time.Date(2026, 10, 8, 14, 0, 0, 0, time.UTC)}
	l.LastExit = &control.Exit{Code: 7, At: time.Date(2026, 10, 8, 14, 0, 0, 0, time.UTC), Stderr: "old", Restarts: 1}
	now := time.Date(2026, 10, 8, 15, 0, 0, 0, time.UTC)

	actions, err := l.Begin(now)
	testhelp.Equal(t, "err", err, nil)
	testhelp.Equal(t, "actions", actions, []Action{{Kind: "start_check", Phase: "P17"}})
	testhelp.Equal(t, "state", l.State, "starting")
	testhelp.Equal(t, "phase", l.Phase, "P17")
	testhelp.Equal(t, "queue state", l.Queue.State, "running")
	testhelp.Equal(t, "restarts", l.Restarts, []time.Time(nil))
	testhelp.Equal(t, "last exit nil", l.LastExit == nil, true)

	actions2, err2 := l.Begin(now)
	testhelp.Equal(t, "actions2", actions2, []Action(nil))
	if err2 == nil {
		t.Fatal("want error, got nil")
	}
	testhelp.Equal(t, "err2", err2.Error(), "queue: cannot start in state \"running\"")
}

func TestPhaseLoopExample2(t *testing.T) {
	l := New("demo", plQueue(t))
	if _, err := l.Begin(time.Date(2026, 10, 8, 15, 0, 0, 0, time.UTC)); err != nil {
		t.Fatal(err)
	}
	exit := control.Exit{Code: 1, At: time.Date(2026, 10, 8, 14, 50, 0, 0, time.UTC), Stderr: "boom", Restarts: 1}
	l.Restarts = []time.Time{time.Date(2026, 10, 8, 14, 50, 0, 0, time.UTC)}
	l.LastExit = &exit
	now := time.Date(2026, 10, 8, 15, 0, 1, 0, time.UTC)

	actions := l.StartChecked(gitrules.Start{Mode: "fresh", Local: "aaa111", Remote: "aaa111"}, plIDs(), now)
	testhelp.Equal(t, "actions", actions, []Action{
		{Kind: "spawn", Phase: "P17", SessionID: "id-1", Resume: false, BudgetUSD: 30},
		{Kind: "first_line", Line: "/morph-orchestrator P17"},
		{Kind: "post", Milestone: control.Milestone{Kind: "start", Headline: "P17 started", Numbers: "session id-1 · cap $30 · 3 h"}},
	})
	testhelp.Equal(t, "state", l.State, "running")
	testhelp.Equal(t, "session", l.SessionID, "id-1")
	testhelp.Equal(t, "started", l.PhaseStartedAt, now)
	testhelp.Equal(t, "restarts", l.Restarts, []time.Time{time.Date(2026, 10, 8, 14, 50, 0, 0, time.UTC)})
	if l.LastExit == nil {
		t.Fatal("last exit nil")
	}
	testhelp.Equal(t, "last exit", *l.LastExit, exit)
}

func TestPhaseLoopExample3(t *testing.T) {
	now := time.Date(2026, 10, 8, 15, 0, 0, 0, time.UTC)
	resume := gitrules.Start{Mode: "resume", Reason: "local ahead of origin/main"}

	l := plStartingP18(t)
	actions := l.StartChecked(resume, func() string { return "id-7" }, now)
	testhelp.Equal(t, "actions", actions, []Action{
		{Kind: "spawn", Phase: "P18", SessionID: "id-7", Resume: true, BudgetUSD: 12},
		{Kind: "post", Milestone: control.Milestone{Kind: "start", Headline: "P18 resumed", Numbers: "session id-7 · local ahead of origin/main"}},
	})

	l2 := plStartingP18(t)
	l2.SessionID = "id-3"
	actions2 := l2.StartChecked(resume, func() string { return "id-7" }, now)
	testhelp.Equal(t, "actions2", actions2, []Action{
		{Kind: "spawn", Phase: "P18", SessionID: "id-3", Resume: true, BudgetUSD: 12},
		{Kind: "post", Milestone: control.Milestone{Kind: "start", Headline: "P18 resumed", Numbers: "session id-3 · local ahead of origin/main"}},
	})
}

func TestPhaseLoopExample4(t *testing.T) {
	l := New("demo", plQueue(t))
	if _, err := l.Begin(time.Date(2026, 10, 8, 15, 0, 0, 0, time.UTC)); err != nil {
		t.Fatal(err)
	}
	now := time.Date(2026, 10, 8, 15, 2, 0, 0, time.UTC)
	reason := "local aaa111 and origin/main ccc333 diverged"

	actions := l.StartChecked(gitrules.Start{Mode: "diverged", Reason: reason}, plIDs(), now)
	testhelp.Equal(t, "actions", actions, []Action{
		{Kind: "post", Milestone: control.Milestone{Kind: "stop", Headline: "P17 not started: git", Numbers: reason}},
	})
	testhelp.Equal(t, "state", l.State, "waiting")
	if l.Stop == nil {
		t.Fatal("stop nil")
	}
	testhelp.Equal(t, "stop", *l.Stop, control.Stop{Kind: "git", Phase: "P17", Reason: reason, At: now})
	testhelp.Equal(t, "queue state", l.Queue.State, "stopped")
}

func TestPhaseLoopExample5(t *testing.T) {
	l := plRunningP17(t)

	actions, err := l.PhaseDone("P17", "P18")
	testhelp.Equal(t, "err", err, nil)
	testhelp.Equal(t, "actions", actions, []Action{{Kind: "end_check", Phase: "P17"}})

	end := l.EndChecked(gitrules.End{OK: true, Local: "ddd", Remote: "ddd", Problems: []string{}}, time.Date(2026, 10, 8, 15, 40, 0, 0, time.UTC))
	testhelp.Equal(t, "end", end, []Action{
		{Kind: "kill"},
		{Kind: "start_check", Phase: "P18"},
	})
	testhelp.Equal(t, "state", l.State, "starting")
	testhelp.Equal(t, "phase", l.Phase, "P18")
	testhelp.Equal(t, "session", l.SessionID, "")
	testhelp.Equal(t, "index", l.Queue.Index, 1)
	testhelp.Equal(t, "queue state", l.Queue.State, "running")
}

func TestPhaseLoopExample6(t *testing.T) {
	l := plRunningP17(t)

	actions, err := l.PhaseDone("P16", "P17")
	testhelp.Equal(t, "actions", actions, []Action(nil))
	if !errors.Is(err, ErrPhaseMismatch) {
		t.Fatalf("want ErrPhaseMismatch, got %v", err)
	}
	testhelp.Equal(t, "text", err.Error(), "phase mismatch: phase_done \"P16\" while running \"P17\"")

	l2 := plRunningP17(t)
	l2.State = "waiting"
	actions2, err2 := l2.PhaseDone("P17", "P18")
	testhelp.Equal(t, "actions2", actions2, []Action(nil))
	if !errors.Is(err2, ErrPhaseMismatch) {
		t.Fatalf("want ErrPhaseMismatch, got %v", err2)
	}
	testhelp.Equal(t, "text2", err2.Error(), "phase mismatch: phase_done \"P17\" while waiting \"P17\"")
}

func TestPhaseLoopExample7(t *testing.T) {
	l := plRunningP17(t)
	l.SessionUSD = 1.92

	actions := l.EndChecked(gitrules.End{OK: false, Problems: []string{"not pushed: HEAD bbb222, origin/main aaa111"}}, time.Date(2026, 10, 8, 16, 0, 0, 0, time.UTC))
	testhelp.Equal(t, "actions", actions, []Action{
		{Kind: "post", Milestone: control.Milestone{Kind: "stop", Headline: "P17 not pushed", Numbers: "not pushed: HEAD bbb222, origin/main aaa111"}},
	})
	testhelp.Equal(t, "state", l.State, "waiting")
	if l.Stop == nil {
		t.Fatal("stop nil")
	}
	testhelp.Equal(t, "stop", *l.Stop, control.Stop{
		Kind:   "not_pushed",
		Phase:  "P17",
		Reason: "not pushed: HEAD bbb222, origin/main aaa111",
		At:     time.Date(2026, 10, 8, 16, 0, 0, 0, time.UTC),
	})
	testhelp.Equal(t, "stretch", l.StretchUSD, 0.0)
	testhelp.Equal(t, "queue state", l.Queue.State, "stopped")

	cont, err := l.Continue(time.Date(2026, 10, 8, 16, 5, 0, 0, time.UTC))
	testhelp.Equal(t, "cont err", err, nil)
	testhelp.Equal(t, "cont", cont, []Action{{Kind: "end_check", Phase: "P17"}})
	testhelp.Equal(t, "state2", l.State, "running")
	testhelp.Equal(t, "stop2 nil", l.Stop == nil, true)
	testhelp.Equal(t, "queue state2", l.Queue.State, "running")
	testhelp.Equal(t, "index2", l.Queue.Index, 0)

	actions2 := l.EndChecked(gitrules.End{OK: true, Problems: []string{}}, time.Date(2026, 10, 8, 16, 10, 0, 0, time.UTC))
	testhelp.Equal(t, "actions2", actions2, []Action{
		{Kind: "kill"},
		{Kind: "start_check", Phase: "P18"},
	})
	testhelp.Equal(t, "stretch2", l.StretchUSD, 1.92)
	testhelp.Equal(t, "session2", l.SessionUSD, 0.0)
}

func TestPhaseLoopExample8(t *testing.T) {
	l := New("demo", plQueue(t))
	l.Queue.Index = 1
	l.Queue.State = "running"
	l.State = "running"
	l.Phase = "P18"
	l.SessionUSD = 0.5
	l.StretchUSD = 1.0

	actions := l.EndChecked(gitrules.End{OK: true}, time.Date(2026, 10, 8, 17, 0, 0, 0, time.UTC))
	testhelp.Equal(t, "actions", actions, []Action{
		{Kind: "kill"},
		{Kind: "post", Milestone: control.Milestone{Kind: "stop", Headline: "P18 done: stop after smoke", Numbers: "next P19 · waiting for continue"}},
	})
	testhelp.Equal(t, "state", l.State, "waiting")
	if l.Stop == nil {
		t.Fatal("stop nil")
	}
	testhelp.Equal(t, "kind", l.Stop.Kind, "smoke")
	testhelp.Equal(t, "stretch", l.StretchUSD, 1.5)

	cont, err := l.Continue(time.Date(2026, 10, 8, 17, 30, 0, 0, time.UTC))
	testhelp.Equal(t, "cont err", err, nil)
	testhelp.Equal(t, "cont", cont, []Action{{Kind: "start_check", Phase: "P19"}})
	testhelp.Equal(t, "state2", l.State, "starting")
	testhelp.Equal(t, "phase2", l.Phase, "P19")

	l.State = "running"

	end := l.EndChecked(gitrules.End{OK: true}, time.Date(2026, 10, 8, 18, 0, 0, 0, time.UTC))
	testhelp.Equal(t, "end", end, []Action{
		{Kind: "kill"},
		{Kind: "post", Milestone: control.Milestone{Kind: "end", Headline: "P19 done: plan complete", Numbers: "stretch $1.5000"}},
	})
	testhelp.Equal(t, "state3", l.State, "waiting")
	if l.Stop == nil {
		t.Fatal("stop3 nil")
	}
	testhelp.Equal(t, "kind3", l.Stop.Kind, "end")
	testhelp.Equal(t, "queue state3", l.Queue.State, "done")

	_, err2 := l.Continue(time.Date(2026, 10, 8, 18, 30, 0, 0, time.UTC))
	if err2 == nil {
		t.Fatal("want error, got nil")
	}
	testhelp.Equal(t, "err2", err2.Error(), "queue: nothing to continue (state \"done\")")
}

func TestPhaseLoopExample9(t *testing.T) {
	l := plRunningP17(t)
	actions := l.WaitOperator("gate", "forecast $1.4 over the cap", "", time.Date(2026, 10, 8, 15, 10, 0, 0, time.UTC))
	testhelp.Equal(t, "actions", actions, []Action{
		{Kind: "post", Milestone: control.Milestone{Kind: "stop", Headline: "P17: wait_operator gate", Numbers: "forecast $1.4 over the cap"}},
	})
	testhelp.Equal(t, "state", l.State, "waiting")
	if l.Stop == nil {
		t.Fatal("stop nil")
	}
	testhelp.Equal(t, "stop", *l.Stop, control.Stop{
		Kind:   "gate",
		Phase:  "P17",
		Reason: "forecast $1.4 over the cap",
		At:     time.Date(2026, 10, 8, 15, 10, 0, 0, time.UTC),
	})

	l2 := plRunningP17(t)
	now := time.Date(2026, 10, 8, 15, 20, 0, 0, time.UTC)
	actions2 := l2.WaitOperator("emergency", "card x red after fix", "https://github.com/o/r/issues/12", now)
	testhelp.Equal(t, "actions2", actions2, []Action{
		{Kind: "post", Milestone: control.Milestone{Kind: "stop", Headline: "P17: wait_operator emergency", Numbers: "card x red after fix · https://github.com/o/r/issues/12"}},
	})
	if l2.Stop == nil {
		t.Fatal("stop2 nil")
	}
	testhelp.Equal(t, "issue", l2.Stop.IssueURL, "https://github.com/o/r/issues/12")

	restart, err := l.Restart(now)
	testhelp.Equal(t, "restart", restart, []Action(nil))
	if !errors.Is(err, control.ErrNoSession) {
		t.Fatalf("want ErrNoSession, got %v", err)
	}

	l3 := plRunningP17(t)
	old := control.Exit{Code: 3, At: time.Date(2026, 10, 8, 15, 5, 0, 0, time.UTC), Stderr: "", Restarts: 1}
	l3.Restarts = []time.Time{time.Date(2026, 10, 8, 15, 5, 0, 0, time.UTC)}
	l3.LastExit = &old
	restart3, err3 := l3.Restart(now)
	testhelp.Equal(t, "err3", err3, nil)
	testhelp.Equal(t, "restart3", restart3, []Action{
		{Kind: "kill"},
		{Kind: "start_check", Phase: "P17"},
	})
	testhelp.Equal(t, "state3", l3.State, "starting")
	testhelp.Equal(t, "session3", l3.SessionID, "")
	testhelp.Equal(t, "restarts3", l3.Restarts, []time.Time(nil))
	if l3.LastExit == nil {
		t.Fatal("last exit3 nil")
	}
	testhelp.Equal(t, "last exit3", *l3.LastExit, old)
}
