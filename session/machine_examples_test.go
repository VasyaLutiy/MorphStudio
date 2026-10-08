package session

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"morphstudio/internal/testhelp"
	"morphstudio/stream"
)

// probePath is the recorded claude stream the examples read their events from.
const probePath = "../tests/fixtures/stream/probe.jsonl"

// probeEvent reads probe line n (1-based) and parses it into a stream.Event.
func probeEvent(t *testing.T, n int) stream.Event {
	t.Helper()
	_, msg := testhelp.ProbeLine(t, probePath, n)
	ev, err := stream.Parse(msg)
	if err != nil {
		t.Fatalf("probe line %d: %v", n, err)
	}
	return ev
}

// parseEvent parses one literal claude line into a stream.Event.
func parseEvent(t *testing.T, line string) stream.Event {
	t.Helper()
	ev, err := stream.Parse([]byte(line))
	if err != nil {
		t.Fatalf("parse %s: %v", line, err)
	}
	return ev
}

// at returns a time on the probe day, 2026-10-08, in UTC.
func at(hour, min, sec int) time.Time {
	return time.Date(2026, 10, 8, hour, min, sec, 0, time.UTC)
}

// oneAction fails the test unless acts holds exactly one action, and returns it.
func oneAction(t *testing.T, acts []Action) Action {
	t.Helper()
	if len(acts) != 1 {
		t.Fatalf("want exactly 1 action, got %d: %#v", len(acts), acts)
	}
	return acts[0]
}

// TestSessionMachineExample1: init sets the session id, the rate limit event
// stamps the limits and asks for a limit action, and the result closes the turn.
func TestSessionMachineExample1(t *testing.T) {
	m := New()
	t3 := at(13, 9, 3)
	t4 := at(13, 9, 4)

	var noActions []Action

	testhelp.Equal(t, "line 4 actions", m.Apply(probeEvent(t, 4), t3), noActions)
	testhelp.Equal(t, "session id", m.SessionID, "967e8f4d-a2eb-4aa6-968b-b9054a1c3d7e")

	testhelp.Equal(t, "line 5 actions", m.Apply(probeEvent(t, 5), t3), noActions)

	acts := m.Apply(probeEvent(t, 6), t3)
	if m.Limits == nil {
		t.Fatal("limits are nil")
	}
	testhelp.Equal(t, "line 6 five hour", m.Limits.FiveHour, stream.Window{Utilization: 0.22, ResetsAt: 1791469200})
	testhelp.Equal(t, "line 6 limits at", m.LimitsAt, t3)
	testhelp.Equal(t, "line 6 actions", acts, []Action{{Kind: "limit"}})

	acts = m.Apply(probeEvent(t, 7), t4)
	testhelp.Equal(t, "state", m.State, "ready")
	testhelp.Equal(t, "turns", m.Turns, 1)
	testhelp.Equal(t, "cost", m.CostUSD, 0.0021784900000000004)
	if m.LastResult == nil {
		t.Fatal("last result is nil")
	}
	testhelp.Equal(t, "last result subtype", m.LastResult.Subtype, "success")
	testhelp.Equal(t, "actions", acts, []Action{{Kind: "turn", Headline: "PONG", Numbers: "success · turns 1 · $0.0022"}})
	testhelp.Equal(t, "last event at", m.LastEventAt, t4)
}

// TestSessionMachineExample2: a second result replaces the cost instead of
// adding to it, and counts a second turn.
func TestSessionMachineExample2(t *testing.T) {
	m := New()

	m.Apply(probeEvent(t, 7), at(13, 9, 4))
	testhelp.Equal(t, "first cost", m.CostUSD, 0.0021784900000000004)

	acts := m.Apply(probeEvent(t, 31), at(13, 9, 9))
	testhelp.Equal(t, "cost", m.CostUSD, 0.005775230000000001)
	testhelp.Equal(t, "turns", m.Turns, 2)
	got := oneAction(t, acts)
	testhelp.Equal(t, "numbers", got.Numbers, "success · turns 2 · $0.0058")
}

// TestSessionMachineExample3: a can_use_tool AskUserQuestion parks the turn in
// the question state and asks the daemon to put the options to the user.
func TestSessionMachineExample3(t *testing.T) {
	m := New()
	m.State = "busy"
	now := at(13, 9, 8)

	acts := m.Apply(probeEvent(t, 27), now)
	testhelp.Equal(t, "state", m.State, "question")
	if m.Pending == nil {
		t.Fatal("pending is nil")
	}
	testhelp.Equal(t, "request id", m.Pending.RequestID, "9f4ffa22-2676-4391-bfd9-bd7d16c3866c")
	testhelp.Equal(t, "options", m.Pending.Question.Options, []string{"Option A", "Option B"})
	testhelp.Equal(t, "asked at", m.Pending.AskedAt, now)
	testhelp.Equal(t, "actions", acts, []Action{{
		Kind:     "ask",
		Headline: "Which option do you want: A or B?",
		Numbers:  "1 Option A · 2 Option B",
	}})
}

// TestSessionMachineExample4: an ordinary tool request is allowed with its input
// echoed back, and a request with no input is allowed with an empty object.
func TestSessionMachineExample4(t *testing.T) {
	m := New()
	m.State = "busy"
	now := at(13, 9, 5)

	first := parseEvent(t, `{"type":"control_request","request_id":"r-7","request":{"subtype":"can_use_tool","tool_name":"Bash","input":{"command":"ls"},"tool_use_id":"toolu_x"}}`)
	acts := m.Apply(first, now)
	testhelp.Equal(t, "state", m.State, "busy")
	testhelp.Equal(t, "pending", m.Pending, (*Pending)(nil))
	testhelp.Equal(t, "allowed", m.Allowed, 1)
	got := oneAction(t, acts)
	testhelp.Equal(t, "kind", got.Kind, "write")
	testhelp.Equal(t, "line", string(got.Line),
		`{"type":"control_response","response":{"subtype":"success","request_id":"r-7","response":{"behavior":"allow","updatedInput":{"command":"ls"}}}}`+"\n")

	second := parseEvent(t, `{"type":"control_request","request_id":"r-8","request":{"subtype":"can_use_tool","tool_name":"Read"}}`)
	acts = m.Apply(second, now)
	testhelp.Equal(t, "state", m.State, "busy")
	testhelp.Equal(t, "pending", m.Pending, (*Pending)(nil))
	testhelp.Equal(t, "allowed", m.Allowed, 2)
	got = oneAction(t, acts)
	testhelp.Equal(t, "kind", got.Kind, "write")
	testhelp.Equal(t, "line", string(got.Line),
		`{"type":"control_response","response":{"subtype":"success","request_id":"r-8","response":{"behavior":"allow","updatedInput":{}}}}`+"\n")
}

// TestSessionMachineExample5: a result drains the queue one message at a time,
// and the machine is ready only once the queue runs out.
func TestSessionMachineExample5(t *testing.T) {
	m := New()
	m.State = "busy"
	m.Queue = []string{"second", "third"}
	start := at(13, 10, 0)

	acts := m.Apply(probeEvent(t, 7), start)
	if len(acts) != 2 {
		t.Fatalf("want 2 actions, got %d: %#v", len(acts), acts)
	}
	testhelp.Equal(t, "first kind", acts[0].Kind, "turn")
	testhelp.Equal(t, "first headline", acts[0].Headline, "PONG")
	testhelp.Equal(t, "first write", string(acts[1].Line), string(stream.User("second")))
	testhelp.Equal(t, "state", m.State, "busy")
	testhelp.Equal(t, "turn started at", m.TurnStartedAt, start)
	testhelp.Equal(t, "queue", m.Queue, []string{"third"})

	acts = m.Apply(probeEvent(t, 31), at(13, 10, 5))
	if len(acts) != 2 {
		t.Fatalf("want 2 actions, got %d: %#v", len(acts), acts)
	}
	testhelp.Equal(t, "second kind", acts[0].Kind, "turn")
	testhelp.Equal(t, "second write", string(acts[1].Line), string(stream.User("third")))
	testhelp.Equal(t, "state", m.State, "busy")
	testhelp.Equal(t, "queue", m.Queue, []string{})

	acts = m.Apply(probeEvent(t, 7), at(13, 10, 6))
	testhelp.Equal(t, "third kind", oneAction(t, acts).Kind, "turn")
	testhelp.Equal(t, "state", m.State, "ready")
}

// TestSessionMachineExample6: an aborted result clears the pending question and
// ends the turn without a headline.
func TestSessionMachineExample6(t *testing.T) {
	m := New()
	m.State = "question"
	m.Pending = &Pending{RequestID: "q-1", AskedAt: at(13, 9, 8)}

	acts := m.Apply(probeEvent(t, 19), at(13, 9, 6))
	testhelp.Equal(t, "state", m.State, "ready")
	testhelp.Equal(t, "pending", m.Pending, (*Pending)(nil))
	if m.LastResult == nil {
		t.Fatal("last result is nil")
	}
	testhelp.Equal(t, "terminal reason", m.LastResult.TerminalReason, "aborted_streaming")
	testhelp.Equal(t, "actions", acts, []Action{{
		Kind:     "turn",
		Headline: "",
		Numbers:  "error_during_execution · turns 3 · $0.0029",
	}})
}

// TestSessionMachineExample7: events that say nothing about the turn leave the
// machine untouched, and a fresh machine marshals an empty queue and no pending.
func TestSessionMachineExample7(t *testing.T) {
	m := New()
	now := at(13, 9, 5)

	var noActions []Action
	for _, n := range []int{12, 13, 14, 16} {
		testhelp.Equal(t, "actions", m.Apply(probeEvent(t, n), now), noActions)
	}

	testhelp.Equal(t, "state", m.State, "ready")
	testhelp.Equal(t, "turns", m.Turns, 0)
	testhelp.Equal(t, "last event at", m.LastEventAt, now)

	fresh := New()
	testhelp.Equal(t, "queue is non-nil", fresh.Queue != nil, true)
	testhelp.Equal(t, "queue", fresh.Queue, []string{})

	b, err := json.Marshal(fresh)
	if err != nil {
		t.Fatal(err)
	}
	s := string(b)
	testhelp.Equal(t, `contains "queue":[]`, strings.Contains(s, `"queue":[]`), true)
	testhelp.Equal(t, `contains "pending"`, strings.Contains(s, `"pending"`), false)
}
