package stream

import (
	"encoding/json"
	"strings"
	"testing"

	"morphstudio/internal/testhelp"
)

const encodeLog = "../tests/fixtures/stream/probe.jsonl"

func encodeDecoded(t *testing.T, what string, b []byte) map[string]any {
	t.Helper()
	var m map[string]any
	if err := json.Unmarshal(b, &m); err != nil {
		t.Errorf("%s: does not decode as a JSON object: %v (bytes %q)", what, err, string(b))
	}
	return m
}

func TestProbeEncodeLinesExample1(t *testing.T) {
	testhelp.Equal(t, "example 1 Initialize(init-1)", string(Initialize("init-1")), `{"type":"control_request","request_id":"init-1","request":{"subtype":"initialize"}}`+"\n")
	testhelp.Equal(t, "example 1 Initialize(init-2)", string(Initialize("init-2")), `{"type":"control_request","request_id":"init-2","request":{"subtype":"initialize"}}`+"\n")
	_, msg := testhelp.ProbeLine(t, encodeLog, 1)
	testhelp.Equal(t, "example 1 Initialize(init-1) decodes equal to probe line 1", encodeDecoded(t, "Initialize", Initialize("init-1")), encodeDecoded(t, "line 1", msg))
}

func TestProbeEncodeLinesExample2(t *testing.T) {
	testhelp.Equal(t, "example 2 Interrupt(int-1)", string(Interrupt("int-1")), `{"type":"control_request","request_id":"int-1","request":{"subtype":"interrupt"}}`+"\n")
	testhelp.Equal(t, "example 2 Interrupt(int-9)", string(Interrupt("int-9")), `{"type":"control_request","request_id":"int-9","request":{"subtype":"interrupt"}}`+"\n")
}

func TestProbeEncodeLinesExample3(t *testing.T) {
	got := User("Reply with exactly one word: PONG")
	testhelp.Equal(t, "example 3 User bytes", string(got), `{"type":"user","message":{"role":"user","content":"Reply with exactly one word: PONG"},"parent_tool_use_id":null}`+"\n")
	_, msg := testhelp.ProbeLine(t, encodeLog, 3)
	testhelp.Equal(t, "example 3 User decodes equal to probe line 3", encodeDecoded(t, "User", got), encodeDecoded(t, "line 3", msg))
	testhelp.Equal(t, "example 3 User(other text)", string(User("hello there")), `{"type":"user","message":{"role":"user","content":"hello there"},"parent_tool_use_id":null}`+"\n")
}

func TestProbeEncodeLinesExample4(t *testing.T) {
	got := string(User("a <b> & \"c\"\nd"))
	want := `{"type":"user","message":{"role":"user","content":"a \u003cb\u003e \u0026 \"c\"\nd"},"parent_tool_use_id":null}` + "\n"
	testhelp.Equal(t, "example 4 User bytes (HTML escapes, \\n escape)", got, want)
}

func TestProbeEncodeLinesExample5(t *testing.T) {
	got, err := Allow("r-7", json.RawMessage(`{"command": "ls"}`))
	testhelp.Equal(t, "example 5 Allow error", err, error(nil))
	testhelp.Equal(t, "example 5 Allow bytes", string(got), `{"type":"control_response","response":{"subtype":"success","request_id":"r-7","response":{"behavior":"allow","updatedInput":{"command":"ls"}}}}`+"\n")
	got, err = Allow("r-2", json.RawMessage(`{"z": 1, "a": [true]}`))
	testhelp.Equal(t, "example 5 Allow(r-2) error", err, error(nil))
	testhelp.Equal(t, "example 5 Allow(r-2) bytes (compacted, key order kept)", string(got), `{"type":"control_response","response":{"subtype":"success","request_id":"r-2","response":{"behavior":"allow","updatedInput":{"z":1,"a":[true]}}}}`+"\n")
}

func TestProbeEncodeLinesExample6(t *testing.T) {
	got, err := Answer("r-1", json.RawMessage(`{"questions":[{"question":"Q?","options":[{"label":"Yes"},{"label":"No"}]}]}`), "Q?", "Yes")
	testhelp.Equal(t, "example 6 Answer error", err, error(nil))
	testhelp.Equal(t, "example 6 Answer bytes", string(got), `{"type":"control_response","response":{"subtype":"success","request_id":"r-1","response":{"behavior":"allow","updatedInput":{"answers":{"Q?":"Yes"},"questions":[{"options":[{"label":"Yes"},{"label":"No"}],"question":"Q?"}]}}}}`+"\n")
	got, err = Answer("r-3", json.RawMessage(`{"answers":{"old":"x"},"k":2}`), "Which?", "No")
	testhelp.Equal(t, "example 6 Answer replaces answers: error", err, error(nil))
	testhelp.Equal(t, "example 6 Answer replaces answers: bytes", string(got), `{"type":"control_response","response":{"subtype":"success","request_id":"r-3","response":{"behavior":"allow","updatedInput":{"answers":{"Which?":"No"},"k":2}}}}`+"\n")
}

func TestProbeEncodeLinesExample7(t *testing.T) {
	_, req := testhelp.ProbeLine(t, encodeLog, 27)
	var line struct {
		Request struct {
			Input json.RawMessage `json:"input"`
		} `json:"request"`
	}
	if err := json.Unmarshal(req, &line); err != nil {
		t.Fatal(err)
	}
	got, err := Answer("9f4ffa22-2676-4391-bfd9-bd7d16c3866c", line.Request.Input, "Which option do you want: A or B?", "Option B")
	testhelp.Equal(t, "example 7 Answer error", err, error(nil))
	testhelp.Equal(t, "example 7 Answer ends with one newline", strings.HasSuffix(string(got), "}\n") && !strings.HasSuffix(string(got), "\n\n"), true)
	dir, msg := testhelp.ProbeLine(t, encodeLog, 28)
	testhelp.Equal(t, "example 7 probe line 28 dir", dir, "in")
	testhelp.Equal(t, "example 7 Answer decodes equal to probe line 28", encodeDecoded(t, "Answer", got), encodeDecoded(t, "line 28", msg))
}

func TestProbeEncodeLinesExample8(t *testing.T) {
	got, err := Answer("r-1", json.RawMessage(`[1]`), "Q?", "Yes")
	if err == nil {
		t.Errorf("example 8 Answer([1]): error nil, want \"stream: input is not an object\"")
	} else {
		testhelp.Equal(t, "example 8 Answer([1]) error", err.Error(), "stream: input is not an object")
	}
	testhelp.Equal(t, "example 8 Answer([1]) line", got, []byte(nil))
	got, err = Allow("r-1", json.RawMessage(`{"a": `))
	if err == nil {
		t.Errorf("example 8 Allow({\"a\": cut): error nil, want one beginning \"stream: bad input\"")
	} else {
		testhelp.Equal(t, "example 8 Allow(cut) error prefix", strings.HasPrefix(err.Error(), "stream: bad input"), true)
	}
	testhelp.Equal(t, "example 8 Allow(cut) line", got, []byte(nil))
}
