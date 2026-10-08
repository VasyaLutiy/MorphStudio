package eventlog

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"morphstudio/internal/testhelp"
)

func pELAt(s, ms int) time.Time { return time.Date(2026, 10, 8, 13, 9, s, ms*1000000, time.UTC) }

func pELOpen(t *testing.T, what, path string) *Log {
	t.Helper()
	l, err := Open(path)
	testhelp.Equal(t, what+" Open error", err, error(nil))
	if l == nil {
		t.Fatalf("%s: Open(%q) returned a nil *Log", what, path)
	}
	return l
}

func pELFile(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}
	return string(b)
}

func pELLines(t *testing.T, path string) []string {
	t.Helper()
	text := pELFile(t, path)
	if text == "" {
		return nil
	}
	return strings.Split(strings.TrimSuffix(text, "\n"), "\n")
}

func pELSeqs(es []Entry) []int64 {
	if es == nil {
		return nil
	}
	out := []int64{}
	for _, e := range es {
		out = append(out, e.Seq)
	}
	return out
}

func pELErrPrefix(t *testing.T, what string, err error, prefix string) {
	t.Helper()
	if err == nil {
		t.Errorf("%s: error nil, want one beginning %q", what, prefix)
		return
	}
	if !strings.HasPrefix(err.Error(), prefix) {
		t.Errorf("%s: error %q, want one beginning %q", what, err.Error(), prefix)
	}
}

// pELThree is the log of examples 1-3 (seq 1..3) and its path.
func pELThree(t *testing.T, what string) (*Log, string) {
	t.Helper()
	path := filepath.Join(t.TempDir(), "demo", "sessions", "s1.jsonl")
	l := pELOpen(t, what, path)
	l.Append("in", []byte(`{"type":"user"}`), pELAt(1, 189))
	l.Append("out", []byte(`{"a": 1, "b": [1, 2]}`), pELAt(2, 0))
	l.Append("out", []byte(`{"c":3}`), pELAt(3, 0))
	return l, path
}

func TestProbeEventLogExample1(t *testing.T) {
	path := filepath.Join(t.TempDir(), "demo", "sessions", "s1.jsonl")
	l := pELOpen(t, "example 1", path)
	st, err := os.Stat(path)
	if err != nil {
		t.Errorf("example 1: the file does not exist: %v", err)
	} else {
		testhelp.Equal(t, "example 1 file size", st.Size(), int64(0))
	}
	testhelp.Equal(t, "example 1 Last", l.Last(), int64(0))
	got := l.Since(0, 10)
	testhelp.Equal(t, "example 1 Since non-nil", got != nil, true)
	testhelp.Equal(t, "example 1 Since", got, []Entry{})
	testhelp.Equal(t, "example 1 Path", l.Path(), path)
}

func TestProbeEventLogExample2(t *testing.T) {
	path := filepath.Join(t.TempDir(), "demo", "sessions", "s1.jsonl")
	l := pELOpen(t, "example 2", path)
	e, err := l.Append("in", []byte(`{"type":"user"}`), pELAt(1, 189))
	testhelp.Equal(t, "example 2 error", err, error(nil))
	testhelp.Equal(t, "example 2 Entry", e, Entry{Seq: 1, T: 1791464941.189, Dir: "in", Msg: json.RawMessage(`{"type":"user"}`)})
	testhelp.Equal(t, "example 2 file", pELFile(t, path), `{"seq":1,"t":1791464941.189,"dir":"in","msg":{"type":"user"}}`+"\n")
	testhelp.Equal(t, "example 2 Last", l.Last(), int64(1))
}

func TestProbeEventLogExample3(t *testing.T) {
	path := filepath.Join(t.TempDir(), "demo", "sessions", "s1.jsonl")
	l := pELOpen(t, "example 3", path)
	l.Append("in", []byte(`{"type":"user"}`), pELAt(1, 189))
	e, err := l.Append("out", []byte(`{"a": 1, "b": [1, 2]}`), pELAt(2, 0))
	testhelp.Equal(t, "example 3 error", err, error(nil))
	testhelp.Equal(t, "example 3 Entry 2", e, Entry{Seq: 2, T: 1791464942, Dir: "out", Msg: json.RawMessage(`{"a":1,"b":[1,2]}`)})
	e, _ = l.Append("out", []byte(`{"c":3}`), pELAt(3, 0))
	testhelp.Equal(t, "example 3 Entry 3 Seq", e.Seq, int64(3))
	lines := pELLines(t, path)
	testhelp.Equal(t, "example 3 file lines", len(lines), 3)
	if len(lines) >= 2 {
		testhelp.Equal(t, "example 3 second line", lines[1], `{"seq":2,"t":1791464942,"dir":"out","msg":{"a":1,"b":[1,2]}}`)
	}
	testhelp.Equal(t, "example 3 Last", l.Last(), int64(3))
}

func TestProbeEventLogExample4(t *testing.T) {
	l, _ := pELThree(t, "example 4")
	testhelp.Equal(t, "example 4 Since(1, 0)", pELSeqs(l.Since(1, 0)), []int64{2, 3})
	testhelp.Equal(t, "example 4 Since(0, 2)", pELSeqs(l.Since(0, 2)), []int64{1, 2})
	testhelp.Equal(t, "example 4 Since(3, 10)", l.Since(3, 10), []Entry{})
	testhelp.Equal(t, "example 4 Since(5, 10)", l.Since(5, 10), []Entry{})
	got := l.Since(1, 1)
	if len(got) == 1 {
		testhelp.Equal(t, "example 4 Since(1, 1) entry", got[0], Entry{Seq: 2, T: 1791464942, Dir: "out", Msg: json.RawMessage(`{"a":1,"b":[1,2]}`)})
	} else {
		t.Errorf("example 4 Since(1, 1): %d entries, want 1", len(got))
	}
}

func TestProbeEventLogExample5(t *testing.T) {
	first, path := pELThree(t, "example 5")
	want := first.Since(0, 10)
	l := pELOpen(t, "example 5 reopen", path)
	testhelp.Equal(t, "example 5 Last", l.Last(), int64(3))
	testhelp.Equal(t, "example 5 Since(0, 10)", l.Since(0, 10), want)
	testhelp.Equal(t, "example 5 Since(0, 10) as written", l.Since(0, 10), []Entry{
		{Seq: 1, T: 1791464941.189, Dir: "in", Msg: json.RawMessage(`{"type":"user"}`)},
		{Seq: 2, T: 1791464942, Dir: "out", Msg: json.RawMessage(`{"a":1,"b":[1,2]}`)},
		{Seq: 3, T: 1791464943, Dir: "out", Msg: json.RawMessage(`{"c":3}`)},
	})
	e, err := l.Append("in", []byte(`{}`), pELAt(4, 0))
	testhelp.Equal(t, "example 5 Append error", err, error(nil))
	testhelp.Equal(t, "example 5 new Seq", e.Seq, int64(4))
	testhelp.Equal(t, "example 5 file lines", len(pELLines(t, path)), 4)
}

func TestProbeEventLogExample6(t *testing.T) {
	path := testhelp.WriteFile(t, "bad.jsonl", `{"seq":1,"t":1,"dir":"in","msg":{}}`+"\n"+"not json\n")
	l, err := Open(path)
	testhelp.Equal(t, "example 6 Log is nil", l == nil, true)
	pELErrPrefix(t, "example 6", err, "eventlog: read "+path+" line 2: ")
	// variant: the line number is counted, not a constant
	path3 := testhelp.WriteFile(t, "bad3.jsonl", `{"seq":1,"t":1,"dir":"in","msg":{}}`+"\n"+`{"seq":2,"t":2,"dir":"out","msg":{}}`+"\n"+"[1\n")
	l, err = Open(path3)
	testhelp.Equal(t, "example 6 third line Log is nil", l == nil, true)
	pELErrPrefix(t, "example 6 third line", err, "eventlog: read "+path3+" line 3: ")
}

func TestProbeEventLogExample7(t *testing.T) {
	path := filepath.Join(testhelp.WriteFile(t, "f", "x"), "s.jsonl")
	l, err := Open(path)
	testhelp.Equal(t, "example 7 Log is nil", l == nil, true)
	pELErrPrefix(t, "example 7", err, "eventlog: open ")
}

func TestProbeEventLogExample8(t *testing.T) {
	path := filepath.Join(t.TempDir(), "s.jsonl")
	l := pELOpen(t, "example 8", path)
	now := time.Date(2026, 10, 8, 13, 0, 0, 0, time.UTC)
	for n := 1; n <= 2500; n++ {
		if _, err := l.Append("out", []byte(fmt.Sprintf(`{"i":%d}`, n)), now); err != nil {
			t.Fatalf("example 8 append %d: %v", n, err)
		}
	}
	want := []int64{}
	for s := int64(501); s <= 700; s++ {
		want = append(want, s)
	}
	testhelp.Equal(t, "example 8 Since(0, 0) seqs", pELSeqs(l.Since(0, 0)), want)
	testhelp.Equal(t, "example 8 Since(2498, 0) seqs", pELSeqs(l.Since(2498, 0)), []int64{2499, 2500})
	testhelp.Equal(t, "example 8 Last", l.Last(), int64(2500))
	testhelp.Equal(t, "example 8 file lines", len(pELLines(t, path)), 2500)
	got := l.Since(2499, 0)
	if len(got) == 1 {
		testhelp.Equal(t, "example 8 entry 2500 Msg", string(got[0].Msg), `{"i":2500}`)
	}
	// variant: Open keeps the last 2000 of the file too
	r := pELOpen(t, "example 8 reopen", path)
	testhelp.Equal(t, "example 8 reopen Since(0, 0) seqs", pELSeqs(r.Since(0, 0)), want)
	testhelp.Equal(t, "example 8 reopen Last", r.Last(), int64(2500))
}

func TestProbeEventLogExample9(t *testing.T) {
	path := filepath.Join(t.TempDir(), "demo", "sessions", "s1.jsonl")
	l := pELOpen(t, "example 9", path)
	e, err := l.Append("in", []byte(`{"a":`), pELAt(5, 0))
	testhelp.Equal(t, "example 9 Entry", e, Entry{})
	pELErrPrefix(t, "example 9", err, "eventlog: bad msg")
	testhelp.Equal(t, "example 9 Last", l.Last(), int64(0))
	testhelp.Equal(t, "example 9 file", pELFile(t, path), "")
}

func TestProbeEventLogExample10(t *testing.T) {
	path := filepath.Join(t.TempDir(), "demo", "sessions", "s1.jsonl")
	l := pELOpen(t, "example 10", path)
	msg := `{"s":"` + strings.Repeat("a", 100000) + `"}`
	_, err := l.Append("out", []byte(msg), pELAt(5, 0))
	testhelp.Equal(t, "example 10 Append error", err, error(nil))
	testhelp.Equal(t, "example 10 file bytes", len(pELFile(t, path)), 100052)
	r, err := Open(path)
	if err != nil || r == nil {
		t.Fatalf("example 10: reopen of a 100 052-byte line: (%v, %v), want a log and nil", r, err)
	}
	testhelp.Equal(t, "example 10 Last", r.Last(), int64(1))
	got := r.Since(0, 10)
	if len(got) != 1 {
		t.Fatalf("example 10: Since(0, 10) %d entries, want 1", len(got))
	}
	testhelp.Equal(t, "example 10 Seq", got[0].Seq, int64(1))
	testhelp.Equal(t, "example 10 Dir", got[0].Dir, "out")
	testhelp.Equal(t, "example 10 Msg length", len(got[0].Msg), 100008)
	testhelp.Equal(t, "example 10 Msg whole", string(got[0].Msg) == msg, true)
}
