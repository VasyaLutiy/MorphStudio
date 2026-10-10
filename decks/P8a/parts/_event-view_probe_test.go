package control

import (
	"encoding/json"
	"strings"
	"testing"
	"unicode/utf8"

	"morphstudio/eventlog"
	"morphstudio/internal/testhelp"
)

const pEVProbe = "../tests/fixtures/stream/probe.jsonl"

func pEVEnt(t *testing.T, n int) eventlog.Entry {
	t.Helper()
	dir, msg := testhelp.ProbeLine(t, pEVProbe, n)
	return eventlog.Entry{Seq: int64(n), Dir: dir, Msg: msg}
}

func pEVJSON(t *testing.T, v any) string {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	return string(b)
}

func TestProbeEventViewExample1(t *testing.T) {
	var ents []eventlog.Entry
	for n := 3; n <= 7; n++ {
		ents = append(ents, pEVEnt(t, n))
	}
	got := pEVJSON(t, Views(Events{Entries: ents, Last: 7}, 200))
	testhelp.Equal(t, "example 1 Views of lines 3..7", got, `{"events":[{"seq":3,"dir":"in","type":"user","text":"Reply with exactly one word: PONG"},{"seq":4,"dir":"out","type":"system","subtype":"init"},{"seq":5,"dir":"out","type":"assistant","text":"PONG"},{"seq":6,"dir":"out","type":"rate_limit_event","text":"five_hour 22% · seven_day 60%"},{"seq":7,"dir":"out","type":"result","subtype":"success","text":"PONG","cost_usd":0.0021784900000000004,"terminal_reason":"completed"}],"last":7}`)
	// variant: the same entries in reverse order keep that order; Last is the Events' Last, not the last Seq
	rev := []eventlog.Entry{ents[4], ents[0]}
	got = pEVJSON(t, Views(Events{Entries: rev, Last: 99}, 200))
	testhelp.Equal(t, "example 1 variant order and Last", got, `{"events":[{"seq":7,"dir":"out","type":"result","subtype":"success","text":"PONG","cost_usd":0.0021784900000000004,"terminal_reason":"completed"},{"seq":3,"dir":"in","type":"user","text":"Reply with exactly one word: PONG"}],"last":99}`)
}

func TestProbeEventViewExample2(t *testing.T) {
	testhelp.Equal(t, "example 2 line 13", pEVJSON(t, View(pEVEnt(t, 13), 200)), `{"seq":13,"dir":"out","type":"assistant","tools":["Bash"]}`)
	testhelp.Equal(t, "example 2 line 14", pEVJSON(t, View(pEVEnt(t, 14), 200)), `{"seq":14,"dir":"out","type":"user","subtype":"tool_result"}`)
	testhelp.Equal(t, "example 2 line 12", pEVJSON(t, View(pEVEnt(t, 12), 200)), `{"seq":12,"dir":"out","type":"assistant"}`)
	testhelp.Equal(t, "example 2 line 34", pEVJSON(t, View(pEVEnt(t, 34), 200)), `{"seq":34,"dir":"out","type":"assistant","tools":["Bash"]}`)
	// variant: two tool_use parts and a text part, in order
	msg := []byte(`{"type":"assistant","message":{"content":[{"type":"tool_use","name":"Read","input":{}},{"type":"text","text":"ok"},{"type":"tool_use","name":"Edit","input":{}}]}}`)
	testhelp.Equal(t, "example 2 variant two tools", pEVJSON(t, View(eventlog.Entry{Seq: 40, Dir: "out", Msg: msg}, 200)), `{"seq":40,"dir":"out","type":"assistant","text":"ok","tools":["Read","Edit"]}`)
}

func TestProbeEventViewExample3(t *testing.T) {
	e := pEVEnt(t, 17)
	full := View(e, 400).Text
	testhelp.Equal(t, "example 3 rune count at 400", utf8.RuneCountInString(full), 327)
	testhelp.Equal(t, "example 3 begins", strings.HasPrefix(full, "1\n2\n3\n"), true)
	testhelp.Equal(t, "example 3 cut at 20", View(e, 20).Text, "1\n2\n3\n4\n5\n6\n7\n8\n9\n10…")
	d := View(e, 0).Text
	testhelp.Equal(t, "example 3 max 0 rune count", utf8.RuneCountInString(d), 201)
	if utf8.RuneCountInString(full) >= 200 {
		testhelp.Equal(t, "example 3 max 0 prefix", strings.HasPrefix(d, string([]rune(full)[:200])), true)
	}
	testhelp.Equal(t, "example 3 max 0 suffix", strings.HasSuffix(d, "…"), true)
	// variant: max 327 returns the text whole
	testhelp.Equal(t, "example 3 variant max 327 whole", View(e, 327).Text, full)
}

func TestProbeEventViewExample4(t *testing.T) {
	want := map[int]string{
		27: `{"seq":27,"dir":"out","type":"control_request","subtype":"can_use_tool","text":"Which option do you want: A or B?","tools":["AskUserQuestion"]}`,
		28: `{"seq":28,"dir":"in","type":"control_response","subtype":"success"}`,
		15: `{"seq":15,"dir":"in","type":"control_request","subtype":"interrupt"}`,
		16: `{"seq":16,"dir":"out","type":"control_response","subtype":"success"}`,
		19: `{"seq":19,"dir":"out","type":"result","subtype":"error_during_execution","cost_usd":0.0028906000000000005,"terminal_reason":"aborted_streaming"}`,
		18: `{"seq":18,"dir":"out","type":"user","subtype":"text","text":"[Request interrupted by user]"}`,
	}
	for _, n := range []int{27, 28, 15, 16, 19, 18} {
		testhelp.Equal(t, "example 4 line "+string(rune('0'+n/10))+string(rune('0'+n%10)), pEVJSON(t, View(pEVEnt(t, n), 200)), want[n])
	}
	// variant: a can_use_tool request without a question names its tool only; the question text is cut
	msg := []byte(`{"type":"control_request","request_id":"r-1","request":{"subtype":"can_use_tool","tool_name":"Bash","input":{"command":"ls"}}}`)
	testhelp.Equal(t, "example 4 variant can_use_tool Bash", pEVJSON(t, View(eventlog.Entry{Seq: 41, Dir: "out", Msg: msg}, 200)), `{"seq":41,"dir":"out","type":"control_request","subtype":"can_use_tool","tools":["Bash"]}`)
	testhelp.Equal(t, "example 4 variant question cut at 5", View(pEVEnt(t, 27), 5).Text, "Which…")
}

func TestProbeEventViewExample5(t *testing.T) {
	exit := eventlog.Entry{Seq: 9, Dir: "exit", Msg: []byte(`{"code":1,"stderr":"Error: Invalid session ID. Must be a valid UUID.\nsecond line"}`)}
	testhelp.Equal(t, "example 5 exit", pEVJSON(t, View(exit, 200)), `{"seq":9,"dir":"exit","type":"exit","exit":{"code":1,"stderr":"Error: Invalid session ID. Must be a valid UUID.\nsecond line"}}`)
	var m map[string]any
	_ = json.Unmarshal([]byte(pEVJSON(t, View(exit, 10))), &m)
	ex, _ := m["exit"].(map[string]any)
	testhelp.Equal(t, "example 5 exit.stderr at 10", ex["stderr"], "Error: Inv…")
	testhelp.Equal(t, "example 5 raw string", pEVJSON(t, View(eventlog.Entry{Seq: 6, Dir: "out", Msg: []byte(`"not json"`)}, 200)), `{"seq":6,"dir":"out","type":"raw","text":"not json"}`)
	testhelp.Equal(t, "example 5 raw array", pEVJSON(t, View(eventlog.Entry{Seq: 7, Dir: "out", Msg: []byte(`[1,2]`)}, 200)), `{"seq":7,"dir":"out","type":"raw","text":"[1,2]"}`)
	testhelp.Equal(t, "example 5 raw no type", pEVJSON(t, View(eventlog.Entry{Seq: 8, Dir: "out", Msg: []byte(`{"a":1}`)}, 200)), `{"seq":8,"dir":"out","type":"raw","text":"{\"a\":1}"}`)
	testhelp.Equal(t, "example 5 exit not an object", pEVJSON(t, View(eventlog.Entry{Seq: 2, Dir: "exit", Msg: []byte(`"x"`)}, 200)), `{"seq":2,"dir":"exit","type":"exit"}`)
	// variant: an exit with code 0 and no stderr keeps both keys
	testhelp.Equal(t, "example 5 variant exit 0", pEVJSON(t, View(eventlog.Entry{Seq: 3, Dir: "exit", Msg: []byte(`{"code":0,"stderr":""}`)}, 200)), `{"seq":3,"dir":"exit","type":"exit","exit":{"code":0,"stderr":""}}`)
}

func TestProbeEventViewExample6(t *testing.T) {
	v := Views(Events{Entries: []eventlog.Entry{}, Last: 3}, 200)
	testhelp.Equal(t, "example 6 events non-nil", v.Events != nil, true)
	testhelp.Equal(t, "example 6 events empty", len(v.Events), 0)
	testhelp.Equal(t, "example 6 last", v.Last, int64(3))
	testhelp.Equal(t, "example 6 json", pEVJSON(t, v), `{"events":[],"last":3}`)
	testhelp.Equal(t, "example 6 zero Events", pEVJSON(t, Views(Events{}, 50)), `{"events":[],"last":0}`)
	testhelp.Equal(t, "example 6 Cut empty", Cut("", 5), "")
	testhelp.Equal(t, "example 6 Cut runes", Cut("héllo wörld", 5), "héllo…")
	testhelp.Equal(t, "example 6 Cut 0 means 200", Cut("abc", 0), "abc")
	testhelp.Equal(t, "example 6 Cut exactly max", Cut("abcde", 5), "abcde")
	// variant: one rune over; a negative max means 200
	testhelp.Equal(t, "example 6 variant one over", Cut("abcdef", 5), "abcde…")
	testhelp.Equal(t, "example 6 variant negative max", Cut(strings.Repeat("x", 201), -1), strings.Repeat("x", 200)+"…")
}
