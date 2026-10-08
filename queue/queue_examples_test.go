package queue

import (
	"encoding/json"
	"os"
	"testing"

	"morphstudio/internal/testhelp"
)

const fixturePath = "../tests/fixtures/plan/queue-3.json"

func loadFixturePlan(t *testing.T) Plan {
	t.Helper()
	data, err := os.ReadFile(fixturePath)
	if err != nil {
		t.Fatal(err)
	}
	var p Plan
	if err := json.Unmarshal(data, &p); err != nil {
		t.Fatal(err)
	}
	return p
}

func loadFixtureQueue(t *testing.T) *Queue {
	t.Helper()
	q, err := Load(loadFixturePlan(t), Caps{ClaudeUSD: 30, Hours: 3, ExecutorUSD: 5})
	if err != nil {
		t.Fatal(err)
	}
	return &q
}

func stoppedQueueAfterExample4(t *testing.T) *Queue {
	t.Helper()
	q := loadFixtureQueue(t)
	if _, err := q.Start(); err != nil {
		t.Fatal(err)
	}
	if _, err := q.Done("P17", "P18"); err != nil {
		t.Fatal(err)
	}
	if _, err := q.Start(); err != nil {
		t.Fatal(err)
	}
	if _, err := q.Done("P18", "P19"); err != nil {
		t.Fatal(err)
	}
	return q
}

func TestPhaseQueueExample1(t *testing.T) {
	q, err := Load(loadFixturePlan(t), Caps{ClaudeUSD: 30, Hours: 3, ExecutorUSD: 5})
	testhelp.Equal(t, "Load error", err, nil)
	testhelp.Equal(t, "State", q.State, "queued")
	testhelp.Equal(t, "Index", q.Index, 0)
	testhelp.Equal(t, "Approved", q.Approved, "7b31dfe")
	testhelp.Equal(t, "Phases[0]", q.Phases[0], Phase{ID: "P17", StopAfter: "none", Caps: Caps{ClaudeUSD: 30, Hours: 3, ExecutorUSD: 5}})
	testhelp.Equal(t, "Phases[1]", q.Phases[1], Phase{ID: "P18", StopAfter: "smoke", Caps: Caps{ClaudeUSD: 12, Hours: 3, ExecutorUSD: 5}})
	testhelp.Equal(t, "Phases[2].StopAfter", q.Phases[2].StopAfter, "none")
	cur, ok := q.Current()
	testhelp.Equal(t, "Current", cur, Phase{ID: "P17", StopAfter: "none", Caps: Caps{ClaudeUSD: 30, Hours: 3, ExecutorUSD: 5}})
	testhelp.Equal(t, "Current ok", ok, true)
}

func TestPhaseQueueExample2(t *testing.T) {
	defaults := Caps{ClaudeUSD: 30, Hours: 3, ExecutorUSD: 5}
	cases := []struct {
		name string
		p    Plan
		want string
	}{
		{"blank approved", Plan{Approved: " "}, "queue: approved hash required"},
		{"no phases", Plan{Approved: "h", Phases: nil}, "queue: no phases"},
		{"empty id", Plan{Approved: "h", Phases: []Phase{{ID: " "}}}, "queue: phase 0: empty id"},
		{"duplicate", Plan{Approved: "h", Phases: []Phase{{ID: "P1"}, {ID: "P1"}}}, `queue: duplicate phase "P1"`},
		{"bad stop_after", Plan{Approved: "h", Phases: []Phase{{ID: "P1", StopAfter: "pause"}}}, `queue: phase "P1": stop_after "pause"`},
	}
	for _, c := range cases {
		_, err := Load(c.p, defaults)
		if err == nil {
			t.Errorf("%s: Load(%#v): got nil error, want %q", c.name, c.p, c.want)
			continue
		}
		testhelp.Equal(t, c.name, err.Error(), c.want)
	}
}

func TestPhaseQueueExample3(t *testing.T) {
	q := loadFixtureQueue(t)

	cur, err := q.Start()
	testhelp.Equal(t, "Start error", err, nil)
	testhelp.Equal(t, "Start phase", cur, Phase{ID: "P17", StopAfter: "none", Caps: Caps{ClaudeUSD: 30, Hours: 3, ExecutorUSD: 5}})

	_, err = q.Start()
	if err == nil {
		t.Fatal("second Start: got nil error")
	}
	testhelp.Equal(t, "second Start error", err.Error(), `queue: cannot start in state "running"`)

	out, err := q.Done("P17", "P18")
	testhelp.Equal(t, "Done error", err, nil)
	testhelp.Equal(t, "Done outcome", out, Outcome{Next: "P18"})
	testhelp.Equal(t, "State", q.State, "queued")
	testhelp.Equal(t, "Index", q.Index, 1)

	cur, ok := q.Current()
	testhelp.Equal(t, "Current", cur, Phase{ID: "P18", StopAfter: "smoke", Caps: Caps{ClaudeUSD: 12, Hours: 3, ExecutorUSD: 5}})
	testhelp.Equal(t, "Current ok", ok, true)
}

func TestPhaseQueueExample4(t *testing.T) {
	q := loadFixtureQueue(t)
	if _, err := q.Start(); err != nil {
		t.Fatal(err)
	}
	if _, err := q.Done("P17", "P18"); err != nil {
		t.Fatal(err)
	}
	if _, err := q.Start(); err != nil {
		t.Fatal(err)
	}

	_, err := q.Done("P17", "P19")
	if err == nil {
		t.Fatal("wrong phase Done: got nil error")
	}
	testhelp.Equal(t, "wrong phase error", err.Error(), `queue: phase_done for "P17" but running "P18"`)

	_, err = q.Done("P18", "P17")
	if err == nil {
		t.Fatal("wrong next Done: got nil error")
	}
	testhelp.Equal(t, "wrong next error", err.Error(), `queue: next "P17", plan says "P19"`)

	out, err := q.Done("P18", "P19")
	testhelp.Equal(t, "Done error", err, nil)
	testhelp.Equal(t, "Done outcome", out, Outcome{Next: "P19", Stop: "smoke"})
	testhelp.Equal(t, "State", q.State, "stopped")
	testhelp.Equal(t, "Reason", q.Reason, "smoke")
	testhelp.Equal(t, "Index", q.Index, 2)
}

func TestPhaseQueueExample5(t *testing.T) {
	q := stoppedQueueAfterExample4(t)

	_, err := q.Done("P19", "")
	if err == nil {
		t.Fatal("Done while stopped: got nil error")
	}
	testhelp.Equal(t, "Done while stopped error", err.Error(), `queue: phase_done for "P19" but running ""`)

	cur, err := q.Continue()
	testhelp.Equal(t, "Continue error", err, nil)
	testhelp.Equal(t, "Continue phase", cur, Phase{ID: "P19", StopAfter: "none", Caps: Caps{ClaudeUSD: 30, Hours: 3, ExecutorUSD: 5}})
	testhelp.Equal(t, "State after Continue", q.State, "queued")
	testhelp.Equal(t, "Reason after Continue", q.Reason, "")

	cur, err = q.Start()
	testhelp.Equal(t, "Start error", err, nil)
	testhelp.Equal(t, "Start phase", cur, Phase{ID: "P19", StopAfter: "none", Caps: Caps{ClaudeUSD: 30, Hours: 3, ExecutorUSD: 5}})

	out, err := q.Done("P19", "")
	testhelp.Equal(t, "Done error", err, nil)
	testhelp.Equal(t, "Done outcome", out, Outcome{Stop: "end"})
	testhelp.Equal(t, "State", q.State, "done")
	testhelp.Equal(t, "Index", q.Index, 3)

	cur, ok := q.Current()
	testhelp.Equal(t, "Current", cur, Phase{})
	testhelp.Equal(t, "Current ok", ok, false)
}

func TestPhaseQueueExample6(t *testing.T) {
	q := loadFixtureQueue(t)
	if _, err := q.Start(); err != nil {
		t.Fatal(err)
	}
	q.Stop("not pushed")
	testhelp.Equal(t, "State", q.State, "stopped")
	testhelp.Equal(t, "Reason", q.Reason, "not pushed")
	testhelp.Equal(t, "Index", q.Index, 0)

	cur, err := q.Continue()
	testhelp.Equal(t, "Continue error", err, nil)
	testhelp.Equal(t, "Continue phase", cur, Phase{ID: "P17", StopAfter: "none", Caps: Caps{ClaudeUSD: 30, Hours: 3, ExecutorUSD: 5}})
	testhelp.Equal(t, "State after Continue", q.State, "queued")

	_, err = q.Done("P17", "P18")
	if err == nil {
		t.Fatal("Done without Start: got nil error")
	}
	testhelp.Equal(t, "Done without Start error", err.Error(), `queue: phase_done for "P17" but running ""`)
}

func TestPhaseQueueExample7(t *testing.T) {
	p := Plan{Approved: "abc", Phases: []Phase{{ID: "P1"}}}
	q, err := Load(p, Caps{})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := q.Start(); err != nil {
		t.Fatal(err)
	}
	if _, err := q.Done("P1", ""); err != nil {
		t.Fatal(err)
	}
	testhelp.Equal(t, "State", q.State, "done")

	_, err = q.Continue()
	if err == nil {
		t.Fatal("Continue on done queue: got nil error")
	}
	testhelp.Equal(t, "Continue error", err.Error(), `queue: nothing to continue (state "done")`)

	before := q
	q.Stop("x")
	testhelp.Equal(t, "done queue unchanged", q, before)

	var zero Queue
	cur, ok := zero.Current()
	testhelp.Equal(t, "zero Current", cur, Phase{})
	testhelp.Equal(t, "zero Current ok", ok, false)

	_, err = zero.Continue()
	if err == nil {
		t.Fatal("Continue on zero queue: got nil error")
	}
	testhelp.Equal(t, "zero Continue error", err.Error(), `queue: nothing to continue (state "")`)
}

func TestPhaseQueueExample8(t *testing.T) {
	p := Plan{Approved: "abc", Phases: []Phase{{ID: "P1", StopAfter: "operator", Caps: Caps{Hours: 1}}}}
	q, err := Load(p, Caps{ClaudeUSD: 20, Hours: 2, ExecutorUSD: 4})
	testhelp.Equal(t, "Load error", err, nil)
	testhelp.Equal(t, "Caps", q.Phases[0].Caps, Caps{ClaudeUSD: 20, Hours: 1, ExecutorUSD: 4})

	cur, err := q.Start()
	testhelp.Equal(t, "Start error", err, nil)
	testhelp.Equal(t, "Start phase", cur, Phase{ID: "P1", StopAfter: "operator", Caps: Caps{ClaudeUSD: 20, Hours: 1, ExecutorUSD: 4}})

	out, err := q.Done("P1", "")
	testhelp.Equal(t, "Done error", err, nil)
	testhelp.Equal(t, "Done outcome", out, Outcome{Stop: "end"})
	testhelp.Equal(t, "State", q.State, "done")
}
