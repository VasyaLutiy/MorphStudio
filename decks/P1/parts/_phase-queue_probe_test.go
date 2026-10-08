package queue

import (
	"encoding/json"
	"os"
	"testing"

	"morphstudio/internal/testhelp"
)

func probeErr(t *testing.T, what string, err error, want string) {
	t.Helper()
	if err == nil {
		t.Errorf("%s: error nil, want %q", what, want)
		return
	}
	testhelp.Equal(t, what, err.Error(), want)
}

func probeLoaded(t *testing.T) Queue {
	t.Helper()
	data, err := os.ReadFile("../tests/fixtures/plan/queue-3.json")
	if err != nil {
		t.Fatal(err)
	}
	var p Plan
	if err := json.Unmarshal(data, &p); err != nil {
		t.Fatal(err)
	}
	q, err := Load(p, Caps{ClaudeUSD: 30, Hours: 3, ExecutorUSD: 5})
	testhelp.Equal(t, "Load(queue-3.json) error", err, error(nil))
	return q
}

// probeAfter3 replays example 3: Start, Done(P17, P18).
func probeAfter3(t *testing.T) Queue {
	t.Helper()
	q := probeLoaded(t)
	q.Start()
	q.Done("P17", "P18")
	return q
}

// probeAfter4 replays example 4 on top: Start, Done(P18, P19) → stopped after smoke.
func probeAfter4(t *testing.T) Queue {
	t.Helper()
	q := probeAfter3(t)
	q.Start()
	q.Done("P18", "P19")
	return q
}

func TestProbePhaseQueueExample1(t *testing.T) {
	q := probeLoaded(t)
	testhelp.Equal(t, "example 1 State", q.State, "queued")
	testhelp.Equal(t, "example 1 Index", q.Index, 0)
	testhelp.Equal(t, "example 1 Approved", q.Approved, "7b31dfe")
	testhelp.Equal(t, "example 1 Phases", q.Phases, []Phase{
		{ID: "P17", StopAfter: "none", Caps: Caps{ClaudeUSD: 30, Hours: 3, ExecutorUSD: 5}},
		{ID: "P18", StopAfter: "smoke", Caps: Caps{ClaudeUSD: 12, Hours: 3, ExecutorUSD: 5}},
		{ID: "P19", StopAfter: "none", Caps: Caps{ClaudeUSD: 30, Hours: 3, ExecutorUSD: 5}},
	})
	cur, ok := q.Current()
	testhelp.Equal(t, "example 1 Current", cur.ID, "P17")
	testhelp.Equal(t, "example 1 Current ok", ok, true)
}

func TestProbePhaseQueueExample2(t *testing.T) {
	d := Caps{ClaudeUSD: 30, Hours: 3, ExecutorUSD: 5}
	for _, c := range []struct {
		plan Plan
		want string
	}{
		{Plan{Approved: " "}, "queue: approved hash required"},
		{Plan{Approved: "h", Phases: nil}, "queue: no phases"},
		{Plan{Approved: "h", Phases: []Phase{{ID: " "}}}, "queue: phase 0: empty id"},
		{Plan{Approved: "h", Phases: []Phase{{ID: "P1"}, {ID: ""}}}, "queue: phase 1: empty id"},
		{Plan{Approved: "h", Phases: []Phase{{ID: "P1"}, {ID: "P1"}}}, `queue: duplicate phase "P1"`},
		{Plan{Approved: "h", Phases: []Phase{{ID: "P1", StopAfter: "pause"}}}, `queue: phase "P1": stop_after "pause"`},
		{Plan{Approved: "h", Phases: []Phase{{ID: "P2", StopAfter: "later"}}}, `queue: phase "P2": stop_after "later"`},
	} {
		_, err := Load(c.plan, d)
		probeErr(t, "example 2 Load error", err, c.want)
	}
}

func TestProbePhaseQueueExample3(t *testing.T) {
	q := probeLoaded(t)
	p, err := q.Start()
	testhelp.Equal(t, "example 3 Start", p.ID, "P17")
	testhelp.Equal(t, "example 3 Start error", err, error(nil))
	_, err = q.Start()
	probeErr(t, "example 3 second Start", err, `queue: cannot start in state "running"`)
	out, err := q.Done("P17", "P18")
	testhelp.Equal(t, "example 3 Done Outcome", out, Outcome{Next: "P18"})
	testhelp.Equal(t, "example 3 Done error", err, error(nil))
	testhelp.Equal(t, "example 3 State", q.State, "queued")
	testhelp.Equal(t, "example 3 Index", q.Index, 1)
	cur, _ := q.Current()
	testhelp.Equal(t, "example 3 Current", cur.ID, "P18")
}

func TestProbePhaseQueueExample4(t *testing.T) {
	q := probeAfter3(t)
	q.Start()
	_, err := q.Done("P17", "P19")
	probeErr(t, "example 4 Done(P17, P19)", err, `queue: phase_done for "P17" but running "P18"`)
	_, err = q.Done("P18", "P17")
	probeErr(t, "example 4 Done(P18, P17)", err, `queue: next "P17", plan says "P19"`)
	out, err := q.Done("P18", "P19")
	testhelp.Equal(t, "example 4 Done(P18, P19) Outcome", out, Outcome{Next: "P19", Stop: "smoke"})
	testhelp.Equal(t, "example 4 Done(P18, P19) error", err, error(nil))
	testhelp.Equal(t, "example 4 State", q.State, "stopped")
	testhelp.Equal(t, "example 4 Reason", q.Reason, "smoke")
	testhelp.Equal(t, "example 4 Index", q.Index, 2)
}

func TestProbePhaseQueueExample5(t *testing.T) {
	q := probeAfter4(t)
	_, err := q.Done("P19", "")
	probeErr(t, "example 5 Done while stopped", err, `queue: phase_done for "P19" but running ""`)
	p, err := q.Continue()
	testhelp.Equal(t, "example 5 Continue", p.ID, "P19")
	testhelp.Equal(t, "example 5 Continue error", err, error(nil))
	testhelp.Equal(t, "example 5 State after Continue", q.State, "queued")
	testhelp.Equal(t, "example 5 Reason after Continue", q.Reason, "")
	p, err = q.Start()
	testhelp.Equal(t, "example 5 Start", p.ID, "P19")
	testhelp.Equal(t, "example 5 Start error", err, error(nil))
	out, err := q.Done("P19", "")
	testhelp.Equal(t, "example 5 last Done Outcome", out, Outcome{Stop: "end"})
	testhelp.Equal(t, "example 5 last Done error", err, error(nil))
	testhelp.Equal(t, "example 5 State", q.State, "done")
	testhelp.Equal(t, "example 5 Index", q.Index, 3)
	_, ok := q.Current()
	testhelp.Equal(t, "example 5 Current ok", ok, false)
}

func TestProbePhaseQueueExample6(t *testing.T) {
	q := probeLoaded(t)
	q.Start()
	q.Stop("not pushed")
	testhelp.Equal(t, "example 6 State", q.State, "stopped")
	testhelp.Equal(t, "example 6 Reason", q.Reason, "not pushed")
	testhelp.Equal(t, "example 6 Index", q.Index, 0)
	p, err := q.Continue()
	testhelp.Equal(t, "example 6 Continue", p.ID, "P17")
	testhelp.Equal(t, "example 6 Continue error", err, error(nil))
	testhelp.Equal(t, "example 6 State after Continue", q.State, "queued")
	_, err = q.Done("P17", "P18")
	probeErr(t, "example 6 Done without Start", err, `queue: phase_done for "P17" but running ""`)
	q2 := probeLoaded(t)
	q2.Stop("queued stop")
	testhelp.Equal(t, "example 6 Stop while queued State", q2.State, "stopped")
	testhelp.Equal(t, "example 6 Stop while queued Reason", q2.Reason, "queued stop")
}

func TestProbePhaseQueueExample7(t *testing.T) {
	q := probeAfter4(t)
	q.Continue()
	q.Start()
	q.Done("P19", "")
	before := q
	before.Phases = append([]Phase(nil), q.Phases...)
	_, err := q.Continue()
	probeErr(t, "example 7 Continue on done", err, `queue: nothing to continue (state "done")`)
	q.Stop("x")
	testhelp.Equal(t, "example 7 done queue unchanged by Stop", q, before)
	var z Queue
	_, ok := z.Current()
	testhelp.Equal(t, "example 7 zero Current ok", ok, false)
	_, err = z.Continue()
	probeErr(t, "example 7 zero Continue", err, `queue: nothing to continue (state "")`)
}

func TestProbePhaseQueueExample8(t *testing.T) {
	q, err := Load(Plan{Approved: "abc", Phases: []Phase{{ID: "P1", StopAfter: "operator", Caps: Caps{Hours: 1}}}}, Caps{ClaudeUSD: 20, Hours: 2, ExecutorUSD: 4})
	testhelp.Equal(t, "example 8 Load error", err, error(nil))
	if len(q.Phases) != 1 {
		t.Fatalf("example 8: %d phases loaded, want 1", len(q.Phases))
	}
	testhelp.Equal(t, "example 8 Caps", q.Phases[0].Caps, Caps{ClaudeUSD: 20, Hours: 1, ExecutorUSD: 4})
	q.Start()
	out, err := q.Done("P1", "")
	testhelp.Equal(t, "example 8 Done Outcome (no next phase wins over stop_after)", out, Outcome{Stop: "end"})
	testhelp.Equal(t, "example 8 Done error", err, error(nil))
	testhelp.Equal(t, "example 8 State", q.State, "done")
	q2, err := Load(Plan{Approved: "abc", Phases: []Phase{{ID: "P2"}}}, Caps{ClaudeUSD: 20, Hours: 2, ExecutorUSD: 4})
	testhelp.Equal(t, "example 8 zero Caps Load error", err, error(nil))
	if len(q2.Phases) == 1 {
		testhelp.Equal(t, "example 8 zero Caps take every default", q2.Phases[0].Caps, Caps{ClaudeUSD: 20, Hours: 2, ExecutorUSD: 4})
	}
}
