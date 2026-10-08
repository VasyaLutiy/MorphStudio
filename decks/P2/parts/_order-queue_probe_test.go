package session

import (
	"encoding/json"
	"errors"
	"testing"
	"time"

	"morphstudio/internal/testhelp"
	"morphstudio/stream"
)

type pOQAct struct{ Kind, Line, Headline, Numbers string }

func pOQActs(acts []Action) []pOQAct {
	if acts == nil {
		return nil
	}
	out := []pOQAct{}
	for _, a := range acts {
		out = append(out, pOQAct{a.Kind, string(a.Line), a.Headline, a.Numbers})
	}
	return out
}

func pOQAt(h, m, s int) time.Time { return time.Date(2026, 10, 8, h, m, s, 0, time.UTC) }

func pOQNew(t *testing.T, what string) *Machine {
	t.Helper()
	m := New()
	if m == nil {
		t.Fatalf("%s: New() returned nil", what)
	}
	return m
}

func pOQJSON(t *testing.T, v OrderResult) string {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}

func pOQErr(t *testing.T, what string, err, target error, text string) {
	t.Helper()
	if err == nil {
		t.Errorf("%s: error nil, want %q", what, text)
		return
	}
	testhelp.Equal(t, what+" errors.Is", errors.Is(err, target), true)
	testhelp.Equal(t, what+" text", err.Error(), text)
}

// pOQAsking is a machine in State "question" whose Pending is the question of probe line 27.
func pOQAsking(t *testing.T, what string) *Machine {
	t.Helper()
	m := pOQNew(t, what)
	_, msg := testhelp.ProbeLine(t, "../tests/fixtures/stream/probe.jsonl", 27)
	ev, err := stream.Parse(msg)
	if err != nil || ev.Question == nil {
		t.Fatalf("%s: probe line 27 did not parse into a question: %v", what, err)
	}
	m.State = "question"
	m.Pending = &Pending{RequestID: ev.RequestID, Question: *ev.Question, Input: ev.Input, AskedAt: pOQAt(13, 9, 8)}
	return m
}

func pOQBusy(t *testing.T, what string) *Machine {
	t.Helper()
	m := pOQNew(t, what)
	m.Order("smoke checked green. Resume with P12b", pOQAt(14, 0, 0))
	return m
}

func TestProbeOrderQueueExample1(t *testing.T) {
	m := pOQNew(t, "example 1")
	res, acts := m.Order("smoke checked green. Resume with P12b", pOQAt(14, 0, 0))
	testhelp.Equal(t, "example 1 OrderResult", res, OrderResult{Sent: true})
	testhelp.Equal(t, "example 1 json", pOQJSON(t, res), `{"sent":true}`)
	testhelp.Equal(t, "example 1 actions", pOQActs(acts), []pOQAct{{Kind: "write", Line: string(stream.User("smoke checked green. Resume with P12b"))}})
	testhelp.Equal(t, "example 1 State", m.State, "busy")
	testhelp.Equal(t, "example 1 TurnStartedAt", m.TurnStartedAt, pOQAt(14, 0, 0))
	testhelp.Equal(t, "example 1 Queue empty", len(m.Queue), 0)
}

func TestProbeOrderQueueExample2(t *testing.T) {
	m := pOQBusy(t, "example 2")
	res, acts := m.Order("second", pOQAt(14, 0, 5))
	testhelp.Equal(t, "example 2 first OrderResult", res, OrderResult{Queued: 1})
	testhelp.Equal(t, "example 2 first json", pOQJSON(t, res), `{"queued":1}`)
	testhelp.Equal(t, "example 2 first actions", pOQActs(acts), []pOQAct(nil))
	res, acts = m.Order("  third  ", pOQAt(14, 0, 6))
	testhelp.Equal(t, "example 2 second OrderResult", res, OrderResult{Queued: 2})
	testhelp.Equal(t, "example 2 second actions", pOQActs(acts), []pOQAct(nil))
	testhelp.Equal(t, "example 2 Queue (trimmed)", m.Queue, []string{"second", "third"})
	testhelp.Equal(t, "example 2 TurnStartedAt unchanged", m.TurnStartedAt, pOQAt(14, 0, 0))
	testhelp.Equal(t, "example 2 State", m.State, "busy")
	// variant: a machine in State "question" queues too
	q := pOQAsking(t, "example 2 question")
	q.Queue = []string{"a", "b"}
	res, acts = q.Order("c", pOQAt(14, 0, 7))
	testhelp.Equal(t, "example 2 question OrderResult", res, OrderResult{Queued: 3})
	testhelp.Equal(t, "example 2 question actions", pOQActs(acts), []pOQAct(nil))
	testhelp.Equal(t, "example 2 question State", q.State, "question")
}

func TestProbeOrderQueueExample3(t *testing.T) {
	m := pOQNew(t, "example 3")
	for _, text := range []string{"   ", ""} {
		res, acts := m.Order(text, pOQAt(14, 1, 0))
		testhelp.Equal(t, "example 3 OrderResult of "+`"`+text+`"`, res, OrderResult{})
		testhelp.Equal(t, "example 3 json of "+`"`+text+`"`, pOQJSON(t, res), `{}`)
		testhelp.Equal(t, "example 3 actions of "+`"`+text+`"`, pOQActs(acts), []pOQAct(nil))
	}
	testhelp.Equal(t, "example 3 State", m.State, "ready")
	testhelp.Equal(t, "example 3 Queue empty", len(m.Queue), 0)
	testhelp.Equal(t, "example 3 TurnStartedAt untouched", m.TurnStartedAt, time.Time{})
	// variant: the empty orders queued nothing, so the next order is sent at once
	res, acts := m.Order("go", pOQAt(14, 1, 1))
	testhelp.Equal(t, "example 3 next order after the empty ones", res, OrderResult{Sent: true})
	testhelp.Equal(t, "example 3 next order actions", pOQActs(acts), []pOQAct{{Kind: "write", Line: string(stream.User("go"))}})
}

func TestProbeOrderQueueExample4(t *testing.T) {
	m := pOQAsking(t, "example 4")
	acts, err := m.Answer(2, pOQAt(13, 9, 8))
	testhelp.Equal(t, "example 4 error", err, error(nil))
	if len(acts) != 1 {
		t.Fatalf("example 4: %d actions, want 1 write", len(acts))
	}
	testhelp.Equal(t, "example 4 action Kind", acts[0].Kind, "write")
	_, want := testhelp.ProbeLine(t, "../tests/fixtures/stream/probe.jsonl", 28)
	var got, exp map[string]any
	if err := json.Unmarshal(acts[0].Line, &got); err != nil {
		t.Errorf("example 4: Line is not JSON: %v (%q)", err, acts[0].Line)
	}
	if err := json.Unmarshal(want, &exp); err != nil {
		t.Fatal(err)
	}
	testhelp.Equal(t, "example 4 Line decodes equal to line 28", got, exp)
	testhelp.Equal(t, "example 4 Pending is nil", m.Pending == nil, true)
	testhelp.Equal(t, "example 4 State", m.State, "busy")
}

func TestProbeOrderQueueExample5(t *testing.T) {
	m := pOQAsking(t, "example 5")
	before := *m.Pending
	acts, err := m.Answer(0, pOQAt(13, 9, 9))
	pOQErr(t, "example 5 Answer(0)", err, ErrBadOption, "option out of range: 0 of 1..2")
	testhelp.Equal(t, "example 5 Answer(0) actions", pOQActs(acts), []pOQAct(nil))
	_, err = m.Answer(3, pOQAt(13, 9, 9))
	pOQErr(t, "example 5 Answer(3)", err, ErrBadOption, "option out of range: 3 of 1..2")
	if m.Pending == nil {
		t.Fatalf("example 5: Pending cleared by a failed Answer, want it unchanged")
	}
	testhelp.Equal(t, "example 5 Pending unchanged", *m.Pending, before)
	testhelp.Equal(t, "example 5 State unchanged", m.State, "question")
	_, err = m.Answer(1, pOQAt(13, 9, 10))
	testhelp.Equal(t, "example 5 Answer(1) error", err, error(nil))
	_, err = m.Answer(1, pOQAt(13, 9, 11))
	pOQErr(t, "example 5 Answer(1) again", err, ErrNoQuestion, "no pending question")
	// variant: the range follows the number of options
	q := pOQNew(t, "example 5 three options")
	q.State = "question"
	q.Pending = &Pending{RequestID: "q-3", Question: stream.Question{Text: "Pick", Options: []string{"Red", "Green", "Blue"}}, Input: json.RawMessage(`{"questions":[]}`)}
	_, err = q.Answer(4, pOQAt(13, 9, 12))
	pOQErr(t, "example 5 Answer(4) of three", err, ErrBadOption, "option out of range: 4 of 1..3")
	acts, err = q.Answer(3, pOQAt(13, 9, 12))
	testhelp.Equal(t, "example 5 Answer(3) of three error", err, error(nil))
	if len(acts) == 1 {
		var got map[string]any
		_ = json.Unmarshal(acts[0].Line, &got)
		resp, _ := got["response"].(map[string]any)
		inner, _ := resp["response"].(map[string]any)
		upd, _ := inner["updatedInput"].(map[string]any)
		testhelp.Equal(t, "example 5 Answer(3) of three answers", upd["answers"], map[string]any{"Pick": "Blue"})
	} else {
		t.Errorf("example 5 Answer(3) of three: %d actions, want 1", len(acts))
	}
}

func TestProbeOrderQueueExample6(t *testing.T) {
	m := pOQNew(t, "example 6")
	acts, err := m.Interrupt("int-1")
	testhelp.Equal(t, "example 6 ready actions", pOQActs(acts), []pOQAct(nil))
	if err == nil {
		t.Errorf("example 6 ready: error nil, want ErrNotBusy")
	} else {
		testhelp.Equal(t, "example 6 ready errors.Is ErrNotBusy", errors.Is(err, ErrNotBusy), true)
	}
	b := pOQBusy(t, "example 6 busy")
	acts, err = b.Interrupt("int-1")
	testhelp.Equal(t, "example 6 busy error", err, error(nil))
	testhelp.Equal(t, "example 6 busy actions", pOQActs(acts), []pOQAct{{Kind: "write", Line: `{"type":"control_request","request_id":"int-1","request":{"subtype":"interrupt"}}` + "\n"}})
	testhelp.Equal(t, "example 6 busy State", b.State, "busy")
}

func TestProbeOrderQueueExample7(t *testing.T) {
	m := pOQAsking(t, "example 7")
	before := *m.Pending
	acts, err := m.Interrupt("int-2")
	testhelp.Equal(t, "example 7 error", err, error(nil))
	testhelp.Equal(t, "example 7 actions", pOQActs(acts), []pOQAct{{Kind: "write", Line: `{"type":"control_request","request_id":"int-2","request":{"subtype":"interrupt"}}` + "\n"}})
	testhelp.Equal(t, "example 7 State", m.State, "question")
	if m.Pending == nil {
		t.Errorf("example 7: Pending cleared by Interrupt, want it kept")
	} else {
		testhelp.Equal(t, "example 7 Pending kept", *m.Pending, before)
	}
}
