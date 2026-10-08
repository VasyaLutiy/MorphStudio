package stream

import (
	"encoding/json"
	"fmt"
	"testing"

	"morphstudio/internal/testhelp"
)

const probeLog = "../tests/fixtures/stream/probe.jsonl"
const probeSession = "967e8f4d-a2eb-4aa6-968b-b9054a1c3d7e"

func probeParse(t *testing.T, n int) Event {
	t.Helper()
	_, msg := testhelp.ProbeLine(t, probeLog, n)
	ev, err := Parse(msg)
	testhelp.Equal(t, fmt.Sprintf("Parse(line %d) error", n), err, error(nil))
	return ev
}

func TestProbeParseEventExample1(t *testing.T) {
	_, msg := testhelp.ProbeLine(t, probeLog, 4)
	ev, err := Parse(msg)
	testhelp.Equal(t, "example 1 line 4 error", err, error(nil))
	testhelp.Equal(t, "example 1 line 4 Event", ev, Event{Type: "system", Subtype: "init", SessionID: probeSession, Raw: json.RawMessage(msg)})
	_, msg = testhelp.ProbeLine(t, probeLog, 16)
	ev, err = Parse(append([]byte("  "), append(msg, '\n')...))
	testhelp.Equal(t, "example 1 line 16 error", err, error(nil))
	testhelp.Equal(t, "example 1 line 16 Event (Raw = the trimmed line)", ev, Event{Type: "control_response", Subtype: "success", RequestID: "int-1", Raw: json.RawMessage(msg)})
}

func TestProbeParseEventExample2(t *testing.T) {
	ev := probeParse(t, 5)
	testhelp.Equal(t, "example 2 line 5 Type", ev.Type, "assistant")
	testhelp.Equal(t, "example 2 line 5 Text", ev.Text, "PONG")
	testhelp.Equal(t, "example 2 line 5 SessionID", ev.SessionID, probeSession)
	ev = probeParse(t, 12)
	testhelp.Equal(t, "example 2 line 12 (thinking) Type", ev.Type, "assistant")
	testhelp.Equal(t, "example 2 line 12 (thinking) Text", ev.Text, "")
	ev = probeParse(t, 13)
	testhelp.Equal(t, "example 2 line 13 (tool_use) Type", ev.Type, "assistant")
	testhelp.Equal(t, "example 2 line 13 (tool_use) Text", ev.Text, "")
	ev, err := Parse([]byte(`{"type":"assistant","message":{"content":[{"type":"text","text":"a"},{"type":"thinking","thinking":"x"},{"type":"text","text":"b"}]}}`))
	testhelp.Equal(t, "example 2 two text parts error", err, error(nil))
	testhelp.Equal(t, "example 2 two text parts joined with \\n", ev.Text, "a\nb")
}

func TestProbeParseEventExample3(t *testing.T) {
	ev := probeParse(t, 6)
	testhelp.Equal(t, "example 3 Type", ev.Type, "rate_limit_event")
	testhelp.Equal(t, "example 3 Limits", ev.Limits, &Limits{FiveHour: Window{Utilization: 0.22, ResetsAt: 1791469200}, SevenDay: Window{Utilization: 0.6, ResetsAt: 1791853200}})
	testhelp.Equal(t, "example 3 Result", ev.Result, (*Result)(nil))
	ev, err := Parse([]byte(`{"type":"rate_limit_event","rate_limit_info":{"unifiedWindows":{"seven_day":{"utilization":0.35,"resetsAt":1791000000}}}}`))
	testhelp.Equal(t, "example 3 one window error", err, error(nil))
	testhelp.Equal(t, "example 3 a missing window stays zero", ev.Limits, &Limits{SevenDay: Window{Utilization: 0.35, ResetsAt: 1791000000}})
}

func TestProbeParseEventExample4(t *testing.T) {
	ev := probeParse(t, 7)
	testhelp.Equal(t, "example 4 Type", ev.Type, "result")
	testhelp.Equal(t, "example 4 Subtype", ev.Subtype, "success")
	testhelp.Equal(t, "example 4 Text", ev.Text, "PONG")
	testhelp.Equal(t, "example 4 Result", ev.Result, &Result{Subtype: "success", IsError: false, NumTurns: 1, TotalCostUSD: 0.0021784900000000004, TerminalReason: "completed", DurationMS: 958, Errors: nil})
	testhelp.Equal(t, "example 4 Limits", ev.Limits, (*Limits)(nil))
}

func TestProbeParseEventExample5(t *testing.T) {
	ev := probeParse(t, 19)
	testhelp.Equal(t, "example 5 Result", ev.Result, &Result{Subtype: "error_during_execution", IsError: true, NumTurns: 3, TotalCostUSD: 0.0028906000000000005, TerminalReason: "aborted_streaming", DurationMS: 3032, Errors: []string{"[ede_diagnostic] result_type=user last_content_type=n/a stop_reason=tool_use"}})
	testhelp.Equal(t, "example 5 Text", ev.Text, "")
	testhelp.Equal(t, "example 5 Subtype", ev.Subtype, "error_during_execution")
}

func TestProbeParseEventExample6(t *testing.T) {
	_, msg := testhelp.ProbeLine(t, probeLog, 27)
	ev := probeParse(t, 27)
	testhelp.Equal(t, "example 6 Type", ev.Type, "control_request")
	testhelp.Equal(t, "example 6 Subtype", ev.Subtype, "can_use_tool")
	testhelp.Equal(t, "example 6 RequestID", ev.RequestID, "9f4ffa22-2676-4391-bfd9-bd7d16c3866c")
	testhelp.Equal(t, "example 6 Tool", ev.Tool, "AskUserQuestion")
	testhelp.Equal(t, "example 6 Question", ev.Question, &Question{Text: "Which option do you want: A or B?", Header: "A or B", Options: []string{"Option A", "Option B"}, MultiSelect: false, ToolUseID: "toolu_01FUdZWoHspVQKBCdwkjgj1R"})
	var line struct {
		Request struct {
			Input map[string]any `json:"input"`
		} `json:"request"`
	}
	if err := json.Unmarshal(msg, &line); err != nil {
		t.Fatal(err)
	}
	var got map[string]any
	if err := json.Unmarshal(ev.Input, &got); err != nil {
		t.Errorf("example 6 Input does not decode: %v (Input %q)", err, string(ev.Input))
	}
	testhelp.Equal(t, "example 6 Input decoded", got, line.Request.Input)
	ev, err := Parse([]byte(`{"type":"control_request","request_id":"r-8","request":{"subtype":"can_use_tool","tool_name":"AskUserQuestion","input":{"questions":[{"question":"Pick?","header":"H","options":[{"label":"x"},{"label":"y"},{"label":"z"}],"multiSelect":true}]},"tool_use_id":"toolu_y"}}`))
	testhelp.Equal(t, "example 6 variant error", err, error(nil))
	testhelp.Equal(t, "example 6 variant Question", ev.Question, &Question{Text: "Pick?", Header: "H", Options: []string{"x", "y", "z"}, MultiSelect: true, ToolUseID: "toolu_y"})
}

func TestProbeParseEventExample7(t *testing.T) {
	ev, err := Parse([]byte(`{"type":"control_request","request_id":"r-7","request":{"subtype":"can_use_tool","tool_name":"Bash","input":{"command":"ls"},"tool_use_id":"toolu_x"}}`))
	testhelp.Equal(t, "example 7 error", err, error(nil))
	testhelp.Equal(t, "example 7 RequestID", ev.RequestID, "r-7")
	testhelp.Equal(t, "example 7 Tool", ev.Tool, "Bash")
	testhelp.Equal(t, "example 7 Question", ev.Question, (*Question)(nil))
	testhelp.Equal(t, "example 7 Input", string(ev.Input), `{"command":"ls"}`)
	testhelp.Equal(t, "example 7 SessionID", ev.SessionID, "")
}

func TestProbeParseEventExample8(t *testing.T) {
	cases := []struct {
		line, want string
		prefix     bool
	}{
		{"", "stream: empty line", false},
		{"  \n", "stream: empty line", false},
		{"[1,2]", "stream: not an object", false},
		{`{"a":1}`, "stream: no type", false},
		{`{"type":"result"`, "stream: bad json", true},
		{`{"type":7}`, "stream: no type", false},
	}
	for _, c := range cases {
		ev, err := Parse([]byte(c.line))
		if err == nil {
			t.Errorf("example 8 Parse(%q): error nil, want %q", c.line, c.want)
			continue
		}
		got := err.Error()
		if c.prefix && len(got) >= len(c.want) {
			got = got[:len(c.want)]
		}
		testhelp.Equal(t, "example 8 Parse("+c.line+") error text", got, c.want)
		testhelp.Equal(t, "example 8 Parse("+c.line+") Event", ev, Event{})
	}
}
