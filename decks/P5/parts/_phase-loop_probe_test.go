package supervisor

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"testing"
	"time"

	"morphstudio/control"
	"morphstudio/gitrules"
	"morphstudio/internal/testhelp"
	"morphstudio/queue"
)

func pPLAt(h, m, s int) time.Time { return time.Date(2026, 10, 8, h, m, s, 0, time.UTC) }

// pPLQueue is queue.Load of the fixture with the given defaults.
func pPLQueue(t *testing.T, defaults queue.Caps) queue.Queue {
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
	return q
}

func pPLIDs(ids ...string) func() string {
	i := 0
	return func() string {
		if i >= len(ids) {
			return "id-extra"
		}
		i++
		return ids[i-1]
	}
}

func pPLPost(kind, headline, numbers string) Action {
	return Action{Kind: "post", Milestone: control.Milestone{Kind: kind, Headline: headline, Numbers: numbers}}
}

// pPLLoop returns a non-nil loop or stops the test with a readable line.
func pPLLoop(t *testing.T, what string, l *Loop) *Loop {
	t.Helper()
	if l == nil {
		t.Fatalf("%s: New returned nil", what)
	}
	return l
}

// pPLRunning is the running loop of example 2: Begin at 15:00:00Z, StartChecked fresh with "id-1" at 15:00:01Z.
func pPLRunning(t *testing.T, what string) *Loop {
	t.Helper()
	l := pPLLoop(t, what, New("demo", pPLQueue(t, queue.Caps{ClaudeUSD: 30, Hours: 3, ExecutorUSD: 5})))
	if _, err := l.Begin(pPLAt(15, 0, 0)); err != nil {
		t.Fatalf("%s: Begin: %v", what, err)
	}
	l.StartChecked(gitrules.Start{Mode: "fresh", Local: "aaa111", Remote: "aaa111"}, pPLIDs("id-1"), pPLAt(15, 0, 1))
	if l.State != "running" {
		t.Fatalf("%s: the running loop of example 2 has State %q, want \"running\"", what, l.State)
	}
	return l
}

func pPLErr(err error) string {
	if err == nil {
		return "<nil>"
	}
	return err.Error()
}

func TestProbePhaseLoopExample1(t *testing.T) {
	l := pPLLoop(t, "example 1", New("demo", pPLQueue(t, queue.Caps{ClaudeUSD: 30, Hours: 3, ExecutorUSD: 5})))
	testhelp.Equal(t, "example 1 New State", l.State, "idle")
	testhelp.Equal(t, "example 1 New Project", l.Project, "demo")
	acts, err := l.Begin(pPLAt(15, 0, 0))
	testhelp.Equal(t, "example 1 Begin actions", acts, []Action{{Kind: "start_check", Phase: "P17"}})
	testhelp.Equal(t, "example 1 Begin error", pPLErr(err), "<nil>")
	testhelp.Equal(t, "example 1 State, Phase, Queue.State", []string{l.State, l.Phase, l.Queue.State}, []string{"starting", "P17", "running"})
	acts, err = l.Begin(pPLAt(15, 0, 5))
	testhelp.Equal(t, "example 1 second Begin error", pPLErr(err), `queue: cannot start in state "running"`)
	testhelp.Equal(t, "example 1 second Begin actions", acts, []Action(nil))
	testhelp.Equal(t, "example 1 after second Begin: State, Phase, Queue.State", []string{l.State, l.Phase, l.Queue.State}, []string{"starting", "P17", "running"})
	// variant: the JSON of a new loop (the record's tags, Stop omitted when nil)
	n := pPLLoop(t, "example 1 variant", New("x", queue.Queue{Approved: "h", State: "queued"}))
	b, _ := json.Marshal(n)
	testhelp.Equal(t, "example 1 variant json.Marshal(New)", string(b), `{"project":"x","queue":{"approved":"h","phases":null,"index":0,"state":"queued","reason":""},"state":"idle","phase":"","session_id":"","phase_started_at":"0001-01-01T00:00:00Z","resumes":null,"paused_until":"0001-01-01T00:00:00Z","stretch_usd":0,"session_usd":0,"last_event_at":"0001-01-01T00:00:00Z","nudged":false,"alerted":false}`)
	// variant: Begin resets the per-phase fields
	w := pPLLoop(t, "example 1 variant", New("demo", pPLQueue(t, queue.Caps{ClaudeUSD: 30, Hours: 3, ExecutorUSD: 5})))
	w.State, w.Stop, w.Nudged, w.Alerted, w.SessionUSD = "waiting", &control.Stop{Kind: "gate"}, true, true, 4.5
	acts, err = w.Begin(pPLAt(9, 0, 0))
	testhelp.Equal(t, "example 1 variant Begin from waiting", acts, []Action{{Kind: "start_check", Phase: "P17"}})
	testhelp.Equal(t, "example 1 variant Begin from waiting error", pPLErr(err), "<nil>")
	testhelp.Equal(t, "example 1 variant Begin resets Stop", w.Stop == nil, true)
	testhelp.Equal(t, "example 1 variant Begin resets Nudged, Alerted, SessionUSD", []any{w.Nudged, w.Alerted, w.SessionUSD}, []any{false, false, 0.0})
}

func TestProbePhaseLoopExample2(t *testing.T) {
	l := pPLLoop(t, "example 2", New("demo", pPLQueue(t, queue.Caps{ClaudeUSD: 30, Hours: 3, ExecutorUSD: 5})))
	l.Begin(pPLAt(15, 0, 0))
	acts := l.StartChecked(gitrules.Start{Mode: "fresh", Local: "aaa111", Remote: "aaa111"}, pPLIDs("id-1", "id-2"), pPLAt(15, 0, 1))
	testhelp.Equal(t, "example 2 StartChecked actions", acts, []Action{
		{Kind: "spawn", Phase: "P17", SessionID: "id-1", Resume: false, BudgetUSD: 30},
		{Kind: "first_line", Line: "/morph-orchestrator P17"},
		pPLPost("start", "P17 started", "session id-1 · cap $30 · 3 h"),
	})
	testhelp.Equal(t, "example 2 State, SessionID", []string{l.State, l.SessionID}, []string{"running", "id-1"})
	testhelp.Equal(t, "example 2 PhaseStartedAt", l.PhaseStartedAt, pPLAt(15, 0, 1))
	testhelp.Equal(t, "example 2 LastEventAt", l.LastEventAt, pPLAt(15, 0, 1))
	// variant: other defaults and a pull reason; a second StartChecked on a running loop is nil
	v := pPLLoop(t, "example 2 variant", New("demo", pPLQueue(t, queue.Caps{ClaudeUSD: 12.5, Hours: 2.5, ExecutorUSD: 5})))
	v.Begin(pPLAt(15, 0, 0))
	v.Resumes = []time.Time{pPLAt(14, 0, 0)}
	acts = v.StartChecked(gitrules.Start{Mode: "fresh", Reason: "pulled to ccc333"}, pPLIDs("s-9"), pPLAt(16, 0, 0))
	testhelp.Equal(t, "example 2 variant StartChecked (caps 12.5 / 2.5, pulled)", acts, []Action{
		{Kind: "spawn", Phase: "P17", SessionID: "s-9", Resume: false, BudgetUSD: 12.5},
		{Kind: "first_line", Line: "/morph-orchestrator P17"},
		pPLPost("start", "P17 started", "session s-9 · cap $12.5 · 2.5 h · pulled to ccc333"),
	})
	testhelp.Equal(t, "example 2 variant fresh clears Resumes", len(v.Resumes), 0)
	testhelp.Equal(t, "example 2 variant StartChecked while running", v.StartChecked(gitrules.Start{Mode: "fresh"}, pPLIDs("s-10"), pPLAt(16, 1, 0)), []Action(nil))
	testhelp.Equal(t, "example 2 variant SessionID kept", v.SessionID, "s-9")
}

func TestProbePhaseLoopExample3(t *testing.T) {
	q := pPLQueue(t, queue.Caps{ClaudeUSD: 30, Hours: 3, ExecutorUSD: 5})
	q.Index, q.State = 1, "running"
	l := &Loop{Project: "demo", Queue: q, State: "starting", Phase: "P18"}
	acts := l.StartChecked(gitrules.Start{Mode: "resume", Reason: "local ahead of origin/main"}, pPLIDs("id-7"), pPLAt(15, 30, 0))
	testhelp.Equal(t, "example 3 StartChecked resume with SessionID \"\"", acts, []Action{
		{Kind: "spawn", Phase: "P18", SessionID: "id-7", Resume: true, BudgetUSD: 12},
		pPLPost("start", "P18 resumed", "session id-7 · local ahead of origin/main"),
	})
	testhelp.Equal(t, "example 3 State", l.State, "running")
	testhelp.Equal(t, "example 3 PhaseStartedAt", l.PhaseStartedAt, pPLAt(15, 30, 0))
	l2 := &Loop{Project: "demo", Queue: q, State: "starting", Phase: "P18", SessionID: "id-3", Resumes: []time.Time{pPLAt(15, 20, 0)}}
	acts = l2.StartChecked(gitrules.Start{Mode: "resume", Reason: "local ahead of origin/main"}, pPLIDs("id-7"), pPLAt(15, 30, 0))
	testhelp.Equal(t, "example 3 StartChecked resume with SessionID \"id-3\"", acts, []Action{
		{Kind: "spawn", Phase: "P18", SessionID: "id-3", Resume: true, BudgetUSD: 12},
		pPLPost("start", "P18 resumed", "session id-3 · local ahead of origin/main"),
	})
	testhelp.Equal(t, "example 3 resume keeps Resumes", l2.Resumes, []time.Time{pPLAt(15, 20, 0)})
}

func TestProbePhaseLoopExample4(t *testing.T) {
	l := pPLLoop(t, "example 4", New("demo", pPLQueue(t, queue.Caps{ClaudeUSD: 30, Hours: 3, ExecutorUSD: 5})))
	l.Begin(pPLAt(15, 0, 0))
	acts := l.StartChecked(gitrules.Start{Mode: "diverged", Reason: "local aaa111 and origin/main ccc333 diverged"}, pPLIDs("id-1"), pPLAt(15, 2, 0))
	testhelp.Equal(t, "example 4 StartChecked diverged", acts, []Action{pPLPost("stop", "P17 not started: git", "local aaa111 and origin/main ccc333 diverged")})
	testhelp.Equal(t, "example 4 State, Queue.State", []string{l.State, l.Queue.State}, []string{"waiting", "stopped"})
	if l.Stop == nil {
		t.Fatalf("example 4 Stop: nil, want {git P17 ...}")
	}
	testhelp.Equal(t, "example 4 Stop", *l.Stop, control.Stop{Kind: "git", Phase: "P17", Reason: "local aaa111 and origin/main ccc333 diverged", At: pPLAt(15, 2, 0)})
	// variant: Mode "error" the same way, with its own reason
	v := pPLLoop(t, "example 4 variant", New("demo", pPLQueue(t, queue.Caps{ClaudeUSD: 30, Hours: 3, ExecutorUSD: 5})))
	v.Begin(pPLAt(15, 0, 0))
	acts = v.StartChecked(gitrules.Start{Mode: "error", Reason: "git fetch -q origin failed: offline"}, pPLIDs("id-1"), pPLAt(15, 3, 0))
	testhelp.Equal(t, "example 4 variant StartChecked error", acts, []Action{pPLPost("stop", "P17 not started: git", "git fetch -q origin failed: offline")})
	testhelp.Equal(t, "example 4 variant Queue.Reason", v.Queue.Reason, "git fetch -q origin failed: offline")
}

func TestProbePhaseLoopExample5(t *testing.T) {
	l := pPLRunning(t, "example 5")
	acts, err := l.PhaseDone("P17", "P18")
	testhelp.Equal(t, "example 5 PhaseDone", acts, []Action{{Kind: "end_check", Phase: "P17"}})
	testhelp.Equal(t, "example 5 PhaseDone error", pPLErr(err), "<nil>")
	testhelp.Equal(t, "example 5 PhaseDone changes nothing", []string{l.State, l.Phase, l.SessionID}, []string{"running", "P17", "id-1"})
	l.SessionUSD, l.Nudged, l.Alerted = 1.92, true, true
	acts = l.EndChecked(gitrules.End{OK: true, Local: "ddd", Remote: "ddd", Problems: []string{}}, pPLAt(15, 40, 0))
	testhelp.Equal(t, "example 5 EndChecked", acts, []Action{{Kind: "kill"}, {Kind: "start_check", Phase: "P18"}})
	testhelp.Equal(t, "example 5 State, Phase, SessionID, Queue.State", []string{l.State, l.Phase, l.SessionID, l.Queue.State}, []string{"starting", "P18", "", "running"})
	testhelp.Equal(t, "example 5 Queue.Index", l.Queue.Index, 1)
	// variant: the next phase begins as Begin does (SessionUSD added to the stretch once, then 0)
	testhelp.Equal(t, "example 5 variant StretchUSD, SessionUSD", []float64{l.StretchUSD, l.SessionUSD}, []float64{1.92, 0})
	testhelp.Equal(t, "example 5 variant Nudged, Alerted reset", []bool{l.Nudged, l.Alerted}, []bool{false, false})
	testhelp.Equal(t, "example 5 variant Stop", l.Stop == nil, true)
}

func TestProbePhaseLoopExample6(t *testing.T) {
	l := pPLRunning(t, "example 6")
	acts, err := l.PhaseDone("P16", "P17")
	testhelp.Equal(t, "example 6 PhaseDone(P16) actions", acts, []Action(nil))
	testhelp.Equal(t, "example 6 errors.Is ErrPhaseMismatch", errors.Is(err, ErrPhaseMismatch), true)
	testhelp.Equal(t, "example 6 error text", pPLErr(err), `phase mismatch: phase_done "P16" while running "P17"`)
	w := pPLRunning(t, "example 6")
	w.State = "waiting"
	_, err = w.PhaseDone("P17", "P18")
	testhelp.Equal(t, "example 6 waiting error text", pPLErr(err), `phase mismatch: phase_done "P17" while waiting "P17"`)
	testhelp.Equal(t, "example 6 ErrPhaseMismatch text", ErrPhaseMismatch.Error(), "phase mismatch")
	// variant: paused accepts phase_done; idle refuses
	p := pPLRunning(t, "example 6 variant")
	p.State = "paused"
	acts, err = p.PhaseDone("P17", "P18")
	testhelp.Equal(t, "example 6 variant paused PhaseDone", acts, []Action{{Kind: "end_check", Phase: "P17"}})
	testhelp.Equal(t, "example 6 variant paused error", pPLErr(err), "<nil>")
	i := pPLLoop(t, "example 6 variant", New("demo", queue.Queue{}))
	_, err = i.PhaseDone("P1", "")
	testhelp.Equal(t, "example 6 variant idle error text", pPLErr(err), `phase mismatch: phase_done "P1" while idle ""`)
}

func TestProbePhaseLoopExample7(t *testing.T) {
	l := pPLRunning(t, "example 7")
	l.SessionUSD = 1.92
	acts := l.EndChecked(gitrules.End{OK: false, Problems: []string{"not pushed: HEAD bbb222, origin/main aaa111"}}, pPLAt(16, 0, 0))
	testhelp.Equal(t, "example 7 EndChecked not OK", acts, []Action{pPLPost("stop", "P17 not pushed", "not pushed: HEAD bbb222, origin/main aaa111")})
	testhelp.Equal(t, "example 7 State, Queue.State", []string{l.State, l.Queue.State}, []string{"waiting", "stopped"})
	if l.Stop == nil {
		t.Fatalf("example 7 Stop: nil, want {not_pushed P17 ...}")
	}
	testhelp.Equal(t, "example 7 Stop", *l.Stop, control.Stop{Kind: "not_pushed", Phase: "P17", Reason: "not pushed: HEAD bbb222, origin/main aaa111", At: pPLAt(16, 0, 0)})
	testhelp.Equal(t, "example 7 StretchUSD", l.StretchUSD, 0.0)
	acts, err := l.Continue(pPLAt(16, 5, 0))
	testhelp.Equal(t, "example 7 Continue", acts, []Action{{Kind: "end_check", Phase: "P17"}})
	testhelp.Equal(t, "example 7 Continue error", pPLErr(err), "<nil>")
	testhelp.Equal(t, "example 7 after Continue: State, Queue.State", []string{l.State, l.Queue.State}, []string{"running", "running"})
	testhelp.Equal(t, "example 7 after Continue: Queue.Index", l.Queue.Index, 0)
	testhelp.Equal(t, "example 7 after Continue: Stop", l.Stop == nil, true)
	acts = l.EndChecked(gitrules.End{OK: true, Problems: []string{}}, pPLAt(16, 10, 0))
	testhelp.Equal(t, "example 7 EndChecked OK after Continue", acts, []Action{{Kind: "kill"}, {Kind: "start_check", Phase: "P18"}})
	testhelp.Equal(t, "example 7 StretchUSD, SessionUSD", []float64{l.StretchUSD, l.SessionUSD}, []float64{1.92, 0})
	// variant: two problems joined with "; "
	v := pPLRunning(t, "example 7 variant")
	acts = v.EndChecked(gitrules.End{OK: false, Problems: []string{"dirty tree (1 paths)", "no MEASURE row for P17"}}, pPLAt(16, 0, 0))
	testhelp.Equal(t, "example 7 variant two problems", acts, []Action{pPLPost("stop", "P17 not pushed", "dirty tree (1 paths); no MEASURE row for P17")})
	testhelp.Equal(t, "example 7 variant Queue.Reason", v.Queue.Reason, "dirty tree (1 paths); no MEASURE row for P17")
	// variant: EndChecked on a starting loop is nil
	s := pPLLoop(t, "example 7 variant", New("demo", pPLQueue(t, queue.Caps{ClaudeUSD: 30, Hours: 3, ExecutorUSD: 5})))
	s.Begin(pPLAt(15, 0, 0))
	testhelp.Equal(t, "example 7 variant EndChecked while starting", s.EndChecked(gitrules.End{OK: true}, pPLAt(15, 1, 0)), []Action(nil))
}

func TestProbePhaseLoopExample8(t *testing.T) {
	q := pPLQueue(t, queue.Caps{ClaudeUSD: 30, Hours: 3, ExecutorUSD: 5})
	q.Index, q.State = 1, "running"
	l := &Loop{Project: "demo", Queue: q, State: "running", Phase: "P18", SessionID: "id-2", SessionUSD: 0.5, StretchUSD: 1.0}
	acts := l.EndChecked(gitrules.End{OK: true, Problems: []string{}}, pPLAt(17, 0, 0))
	testhelp.Equal(t, "example 8 EndChecked P18 (stop after smoke)", acts, []Action{{Kind: "kill"}, pPLPost("stop", "P18 done: stop after smoke", "next P19 · waiting for continue")})
	testhelp.Equal(t, "example 8 State", l.State, "waiting")
	testhelp.Equal(t, "example 8 StretchUSD", l.StretchUSD, 1.5)
	if l.Stop == nil {
		t.Fatalf("example 8 Stop: nil, want {smoke P18 ...}")
	}
	testhelp.Equal(t, "example 8 Stop", *l.Stop, control.Stop{Kind: "smoke", Phase: "P18", Reason: "stop after smoke", At: pPLAt(17, 0, 0)})
	acts, err := l.Continue(pPLAt(17, 30, 0))
	testhelp.Equal(t, "example 8 Continue", acts, []Action{{Kind: "start_check", Phase: "P19"}})
	testhelp.Equal(t, "example 8 Continue error", pPLErr(err), "<nil>")
	testhelp.Equal(t, "example 8 after Continue: State, Phase", []string{l.State, l.Phase}, []string{"starting", "P19"})
	l.StartChecked(gitrules.Start{Mode: "fresh"}, pPLIDs("id-3"), pPLAt(17, 31, 0))
	acts = l.EndChecked(gitrules.End{OK: true, Problems: []string{}}, pPLAt(18, 0, 0))
	testhelp.Equal(t, "example 8 EndChecked P19 (plan complete)", acts, []Action{{Kind: "kill"}, pPLPost("end", "P19 done: plan complete", "stretch $1.5000")})
	testhelp.Equal(t, "example 8 State, Queue.State", []string{l.State, l.Queue.State}, []string{"waiting", "done"})
	if l.Stop == nil {
		t.Fatalf("example 8 Stop after the end: nil, want {end P19 ...}")
	}
	testhelp.Equal(t, "example 8 Stop after the end", *l.Stop, control.Stop{Kind: "end", Phase: "P19", Reason: "plan complete", At: pPLAt(18, 0, 0)})
	acts, err = l.Continue(pPLAt(18, 5, 0))
	testhelp.Equal(t, "example 8 Continue after the end error", pPLErr(err), `queue: nothing to continue (state "done")`)
	testhelp.Equal(t, "example 8 Continue after the end actions", acts, []Action(nil))
	// variant: Continue on a running loop; other StretchUSD for the end line
	r := pPLRunning(t, "example 8 variant")
	_, err = r.Continue(pPLAt(15, 1, 0))
	testhelp.Equal(t, "example 8 variant Continue while running is ErrNotWaiting", errors.Is(err, control.ErrNotWaiting), true)
	q3 := pPLQueue(t, queue.Caps{ClaudeUSD: 30, Hours: 3, ExecutorUSD: 5})
	q3.Index, q3.State = 2, "running"
	e := &Loop{Project: "demo", Queue: q3, State: "paused", Phase: "P19", SessionUSD: 0.0625, StretchUSD: 2}
	acts = e.EndChecked(gitrules.End{OK: true}, pPLAt(19, 0, 0))
	testhelp.Equal(t, "example 8 variant end line", acts, []Action{{Kind: "kill"}, pPLPost("end", "P19 done: plan complete", "stretch $2.0625")})
}

func TestProbePhaseLoopExample9(t *testing.T) {
	l := pPLRunning(t, "example 9")
	acts := l.WaitOperator("gate", "forecast $1.4 over the cap", "", pPLAt(15, 10, 0))
	testhelp.Equal(t, "example 9 WaitOperator gate", acts, []Action{pPLPost("stop", "P17: wait_operator gate", "forecast $1.4 over the cap")})
	testhelp.Equal(t, "example 9 State, Queue.State, Queue.Reason", []string{l.State, l.Queue.State, l.Queue.Reason}, []string{"waiting", "stopped", "gate"})
	if l.Stop == nil {
		t.Fatalf("example 9 Stop: nil, want {gate P17 ...}")
	}
	testhelp.Equal(t, "example 9 Stop", *l.Stop, control.Stop{Kind: "gate", Phase: "P17", Reason: "forecast $1.4 over the cap", At: pPLAt(15, 10, 0)})
	f := pPLRunning(t, "example 9")
	acts = f.WaitOperator("emergency", "card x red after fix", "https://github.com/o/r/issues/12", pPLAt(15, 11, 0))
	testhelp.Equal(t, "example 9 WaitOperator emergency", acts, []Action{pPLPost("stop", "P17: wait_operator emergency", "card x red after fix · https://github.com/o/r/issues/12")})
	if f.Stop == nil {
		t.Fatalf("example 9 emergency Stop: nil")
	}
	testhelp.Equal(t, "example 9 Stop.IssueURL", f.Stop.IssueURL, "https://github.com/o/r/issues/12")
	acts, err := l.Restart(pPLAt(15, 12, 0))
	testhelp.Equal(t, "example 9 Restart while waiting actions", acts, []Action(nil))
	testhelp.Equal(t, "example 9 Restart while waiting is control.ErrNoSession", err == control.ErrNoSession, true)
	r := pPLRunning(t, "example 9")
	r.Resumes = []time.Time{pPLAt(15, 5, 0)}
	acts, err = r.Restart(pPLAt(15, 13, 0))
	testhelp.Equal(t, "example 9 Restart while running", acts, []Action{{Kind: "kill"}, {Kind: "start_check", Phase: "P17"}})
	testhelp.Equal(t, "example 9 Restart error", pPLErr(err), "<nil>")
	testhelp.Equal(t, "example 9 after Restart: State, SessionID", []string{r.State, r.SessionID}, []string{"starting", ""})
	testhelp.Equal(t, "example 9 after Restart: Resumes", len(r.Resumes), 0)
	// variant: WaitOperator on an idle loop is nil; Restart on a paused loop works, on idle/starting it does not
	i := pPLLoop(t, "example 9 variant", New("demo", pPLQueue(t, queue.Caps{ClaudeUSD: 30, Hours: 3, ExecutorUSD: 5})))
	testhelp.Equal(t, "example 9 variant WaitOperator while idle", i.WaitOperator("gate", "x", "", pPLAt(15, 0, 0)), []Action(nil))
	testhelp.Equal(t, "example 9 variant idle State kept", i.State, "idle")
	_, err = i.Restart(pPLAt(15, 0, 0))
	testhelp.Equal(t, "example 9 variant Restart while idle", fmt.Sprint(err), control.ErrNoSession.Error())
	p := pPLRunning(t, "example 9 variant")
	p.State = "paused"
	acts, _ = p.Restart(pPLAt(15, 14, 0))
	testhelp.Equal(t, "example 9 variant Restart while paused", acts, []Action{{Kind: "kill"}, {Kind: "start_check", Phase: "P17"}})
	s := pPLLoop(t, "example 9 variant", New("demo", pPLQueue(t, queue.Caps{ClaudeUSD: 30, Hours: 3, ExecutorUSD: 5})))
	s.Begin(pPLAt(15, 0, 0))
	_, err = s.Restart(pPLAt(15, 0, 1))
	testhelp.Equal(t, "example 9 variant Restart while starting", fmt.Sprint(err), control.ErrNoSession.Error())
}
