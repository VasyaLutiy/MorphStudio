package session

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
	"time"

	"morphstudio/internal/testhelp"
	"morphstudio/stream"
)

const pSMLog = "../tests/fixtures/stream/probe.jsonl"

type pSMAct struct{ Kind, Line, Headline, Numbers string }

func pSMActs(acts []Action) []pSMAct {
	if acts == nil {
		return nil
	}
	out := []pSMAct{}
	for _, a := range acts {
		out = append(out, pSMAct{a.Kind, string(a.Line), a.Headline, a.Numbers})
	}
	return out
}

func pSMLine(t *testing.T, n int) stream.Event {
	t.Helper()
	_, msg := testhelp.ProbeLine(t, pSMLog, n)
	ev, err := stream.Parse(msg)
	if err != nil {
		t.Fatalf("probe line %d: %v", n, err)
	}
	return ev
}

func pSMLit(t *testing.T, line string) stream.Event {
	t.Helper()
	ev, err := stream.Parse([]byte(line))
	if err != nil {
		t.Fatalf("literal line: %v", err)
	}
	return ev
}

func pSMNew(t *testing.T, what string) *Machine {
	t.Helper()
	m := New()
	if m == nil {
		t.Fatalf("%s: New() returned nil", what)
	}
	return m
}

func pSMAt(h, m, s int) time.Time { return time.Date(2026, 10, 8, h, m, s, 0, time.UTC) }

func pSMUser(text string) string { return string(stream.User(text)) }

func TestProbeSessionMachineExample1(t *testing.T) {
	m := pSMNew(t, "example 1")
	acts := m.Apply(pSMLine(t, 4), pSMAt(13, 9, 3))
	testhelp.Equal(t, "example 1 after line 4 SessionID", m.SessionID, "967e8f4d-a2eb-4aa6-968b-b9054a1c3d7e")
	testhelp.Equal(t, "example 1 after line 4 actions", pSMActs(acts), []pSMAct(nil))
	acts = m.Apply(pSMLine(t, 5), pSMAt(13, 9, 3))
	testhelp.Equal(t, "example 1 after line 5 actions", pSMActs(acts), []pSMAct(nil))
	acts = m.Apply(pSMLine(t, 6), pSMAt(13, 9, 3))
	if m.Limits == nil {
		t.Errorf("example 1 after line 6: Limits nil, want FiveHour {0.22, 1791469200}")
	} else {
		testhelp.Equal(t, "example 1 after line 6 Limits.FiveHour", m.Limits.FiveHour, stream.Window{Utilization: 0.22, ResetsAt: 1791469200})
	}
	testhelp.Equal(t, "example 1 after line 6 LimitsAt", m.LimitsAt, pSMAt(13, 9, 3))
	testhelp.Equal(t, "example 1 after line 6 actions", pSMActs(acts), []pSMAct{{Kind: "limit"}})
	acts = m.Apply(pSMLine(t, 7), pSMAt(13, 9, 4))
	testhelp.Equal(t, "example 1 after line 7 State", m.State, "ready")
	testhelp.Equal(t, "example 1 after line 7 Turns", m.Turns, 1)
	testhelp.Equal(t, "example 1 after line 7 CostUSD", m.CostUSD, 0.0021784900000000004)
	if m.LastResult == nil {
		t.Errorf("example 1 after line 7: LastResult nil, want Subtype \"success\"")
	} else {
		testhelp.Equal(t, "example 1 after line 7 LastResult.Subtype", m.LastResult.Subtype, "success")
	}
	testhelp.Equal(t, "example 1 after line 7 actions", pSMActs(acts), []pSMAct{{Kind: "turn", Headline: "PONG", Numbers: "success · turns 1 · $0.0022"}})
	testhelp.Equal(t, "example 1 LastEventAt", m.LastEventAt, pSMAt(13, 9, 4))
}

func TestProbeSessionMachineExample2(t *testing.T) {
	m := pSMNew(t, "example 2")
	m.Apply(pSMLine(t, 7), pSMAt(13, 9, 4))
	testhelp.Equal(t, "example 2 CostUSD after line 7", m.CostUSD, 0.0021784900000000004)
	acts := m.Apply(pSMLine(t, 31), pSMAt(13, 9, 20))
	testhelp.Equal(t, "example 2 CostUSD after line 31 (replaced, not added)", m.CostUSD, 0.005775230000000001)
	testhelp.Equal(t, "example 2 Turns", m.Turns, 2)
	testhelp.Equal(t, "example 2 actions", pSMActs(acts), []pSMAct{{Kind: "turn", Headline: "PICKED=Option B", Numbers: "success · turns 2 · $0.0058"}})
}

func TestProbeSessionMachineExample3(t *testing.T) {
	m := pSMNew(t, "example 3")
	m.State = "busy"
	ev := pSMLine(t, 27)
	acts := m.Apply(ev, pSMAt(13, 9, 8))
	testhelp.Equal(t, "example 3 State", m.State, "question")
	if m.Pending == nil {
		t.Errorf("example 3: Pending nil, want RequestID \"9f4ffa22-2676-4391-bfd9-bd7d16c3866c\"")
	} else {
		testhelp.Equal(t, "example 3 Pending.RequestID", m.Pending.RequestID, "9f4ffa22-2676-4391-bfd9-bd7d16c3866c")
		testhelp.Equal(t, "example 3 Pending.Question.Options", m.Pending.Question.Options, []string{"Option A", "Option B"})
		testhelp.Equal(t, "example 3 Pending.AskedAt", m.Pending.AskedAt, pSMAt(13, 9, 8))
		testhelp.Equal(t, "example 3 Pending.Input", string(m.Pending.Input), string(ev.Input))
	}
	testhelp.Equal(t, "example 3 actions", pSMActs(acts), []pSMAct{{Kind: "ask", Headline: "Which option do you want: A or B?", Numbers: "1 Option A · 2 Option B"}})
	// variant: three options are numbered 1..3 (the behaviour's rule, not a constant of two)
	m2 := pSMNew(t, "example 3 three options")
	m2.State = "busy"
	acts = m2.Apply(pSMLit(t, `{"type":"control_request","request_id":"q-3","request":{"subtype":"can_use_tool","tool_name":"AskUserQuestion","input":{"questions":[{"question":"Pick a colour","header":"Colour","options":[{"label":"Red"},{"label":"Green"},{"label":"Blue"}],"multiSelect":false}]},"tool_use_id":"toolu_q"}}`), pSMAt(13, 9, 9))
	testhelp.Equal(t, "example 3 three options actions", pSMActs(acts), []pSMAct{{Kind: "ask", Headline: "Pick a colour", Numbers: "1 Red · 2 Green · 3 Blue"}})
	if m2.Pending != nil {
		testhelp.Equal(t, "example 3 three options Pending.RequestID", m2.Pending.RequestID, "q-3")
	}
}

func TestProbeSessionMachineExample4(t *testing.T) {
	m := pSMNew(t, "example 4")
	m.State = "busy"
	acts := m.Apply(pSMLit(t, `{"type":"control_request","request_id":"r-7","request":{"subtype":"can_use_tool","tool_name":"Bash","input":{"command":"ls"},"tool_use_id":"toolu_x"}}`), pSMAt(13, 9, 10))
	testhelp.Equal(t, "example 4 State", m.State, "busy")
	testhelp.Equal(t, "example 4 Pending is nil", m.Pending == nil, true)
	testhelp.Equal(t, "example 4 Allowed", m.Allowed, 1)
	testhelp.Equal(t, "example 4 actions", pSMActs(acts), []pSMAct{{Kind: "write", Line: `{"type":"control_response","response":{"subtype":"success","request_id":"r-7","response":{"behavior":"allow","updatedInput":{"command":"ls"}}}}` + "\n"}})
	acts = m.Apply(pSMLit(t, `{"type":"control_request","request_id":"r-8","request":{"subtype":"can_use_tool","tool_name":"Read"}}`), pSMAt(13, 9, 11))
	testhelp.Equal(t, "example 4 no input Allowed", m.Allowed, 2)
	testhelp.Equal(t, "example 4 no input actions (sent as {})", pSMActs(acts), []pSMAct{{Kind: "write", Line: `{"type":"control_response","response":{"subtype":"success","request_id":"r-8","response":{"behavior":"allow","updatedInput":{}}}}` + "\n"}})
	testhelp.Equal(t, "example 4 no input State", m.State, "busy")
}

func TestProbeSessionMachineExample5(t *testing.T) {
	m := pSMNew(t, "example 5")
	m.State = "busy"
	m.Queue = []string{"second", "third"}
	acts := m.Apply(pSMLine(t, 7), pSMAt(13, 10, 0))
	testhelp.Equal(t, "example 5 first result actions", pSMActs(acts), []pSMAct{{Kind: "turn", Headline: "PONG", Numbers: "success · turns 1 · $0.0022"}, {Kind: "write", Line: pSMUser("second")}})
	testhelp.Equal(t, "example 5 State", m.State, "busy")
	testhelp.Equal(t, "example 5 TurnStartedAt", m.TurnStartedAt, pSMAt(13, 10, 0))
	testhelp.Equal(t, "example 5 Queue", m.Queue, []string{"third"})
	acts = m.Apply(pSMLine(t, 7), pSMAt(13, 10, 30))
	testhelp.Equal(t, "example 5 second result actions", pSMActs(acts), []pSMAct{{Kind: "turn", Headline: "PONG", Numbers: "success · turns 1 · $0.0022"}, {Kind: "write", Line: pSMUser("third")}})
	testhelp.Equal(t, "example 5 Queue empty", len(m.Queue), 0)
	testhelp.Equal(t, "example 5 second TurnStartedAt", m.TurnStartedAt, pSMAt(13, 10, 30))
	acts = m.Apply(pSMLine(t, 7), pSMAt(13, 11, 0))
	testhelp.Equal(t, "example 5 third result actions (turn only)", pSMActs(acts), []pSMAct{{Kind: "turn", Headline: "PONG", Numbers: "success · turns 1 · $0.0022"}})
	testhelp.Equal(t, "example 5 third State", m.State, "ready")
	testhelp.Equal(t, "example 5 Turns", m.Turns, 3)
}

func TestProbeSessionMachineExample6(t *testing.T) {
	m := pSMNew(t, "example 6")
	q := pSMLine(t, 27)
	m.State = "question"
	if q.Question != nil {
		m.Pending = &Pending{RequestID: q.RequestID, Question: *q.Question, Input: q.Input, AskedAt: pSMAt(13, 9, 8)}
	}
	acts := m.Apply(pSMLine(t, 19), pSMAt(13, 9, 12))
	testhelp.Equal(t, "example 6 State", m.State, "ready")
	testhelp.Equal(t, "example 6 Pending is nil", m.Pending == nil, true)
	if m.LastResult == nil {
		t.Errorf("example 6: LastResult nil, want TerminalReason \"aborted_streaming\"")
	} else {
		testhelp.Equal(t, "example 6 LastResult.TerminalReason", m.LastResult.TerminalReason, "aborted_streaming")
	}
	testhelp.Equal(t, "example 6 actions (Numbers from the result's num_turns)", pSMActs(acts), []pSMAct{{Kind: "turn", Headline: "", Numbers: "error_during_execution · turns 3 · $0.0029"}})
	testhelp.Equal(t, "example 6 Turns (the machine's own count)", m.Turns, 1)
}

func TestProbeSessionMachineExample7(t *testing.T) {
	m := pSMNew(t, "example 7")
	for _, n := range []int{12, 13, 14, 16} {
		acts := m.Apply(pSMLine(t, n), pSMAt(13, 9, 5))
		testhelp.Equal(t, fmt.Sprintf("example 7 line %d actions", n), pSMActs(acts), []pSMAct(nil))
	}
	testhelp.Equal(t, "example 7 State", m.State, "ready")
	testhelp.Equal(t, "example 7 Turns", m.Turns, 0)
	testhelp.Equal(t, "example 7 LastEventAt", m.LastEventAt, pSMAt(13, 9, 5))
	acts := m.Apply(pSMLit(t, `{"type":"system","subtype":"status","session_id":"other-id"}`), pSMAt(13, 9, 6))
	testhelp.Equal(t, "example 7 system non-init actions", pSMActs(acts), []pSMAct(nil))
	testhelp.Equal(t, "example 7 system non-init keeps SessionID", m.SessionID, "")
	testhelp.Equal(t, "example 7 system non-init LastEventAt", m.LastEventAt, pSMAt(13, 9, 6))
	n := pSMNew(t, "example 7 New")
	testhelp.Equal(t, "example 7 New().Queue non-nil", n.Queue != nil, true)
	testhelp.Equal(t, "example 7 New().Queue empty", len(n.Queue), 0)
	testhelp.Equal(t, "example 7 New().State", n.State, "ready")
	b, err := json.Marshal(n)
	testhelp.Equal(t, "example 7 json.Marshal(New()) error", err, error(nil))
	testhelp.Equal(t, "example 7 json holds \"queue\":[]", strings.Contains(string(b), `"queue":[]`), true)
	testhelp.Equal(t, "example 7 json has no \"pending\" key", strings.Contains(string(b), `"pending"`), false)
}
