package testhelp

import (
	"bytes"
	"encoding/json"
	"os"
	"testing"
)

// probeLine is one line of a probe log: the time, the direction ("in" to claude, "out" from claude) and the message.
type probeLine struct {
	T   any             `json:"t"`
	Dir string          `json:"dir"`
	Msg json.RawMessage `json:"msg"`
}

// probeLines reads a JSONL probe log and decodes every line; a missing file or a bad line fails the test.
func probeLines(t testing.TB, path string) []probeLine {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var lines []probeLine
	for i, raw := range bytes.Split(bytes.TrimRight(data, "\n"), []byte("\n")) {
		var l probeLine
		if err := json.Unmarshal(raw, &l); err != nil {
			t.Fatalf("%s line %d: %v", path, i+1, err)
		}
		lines = append(lines, l)
	}
	return lines
}

// ProbeLine reads line n (1-based) of a JSONL probe log and returns its direction and the raw bytes of its msg.
func ProbeLine(t testing.TB, path string, n int) (dir string, msg []byte) {
	t.Helper()
	lines := probeLines(t, path)
	if n < 1 || n > len(lines) {
		t.Fatalf("%s: line %d out of 1..%d", path, n, len(lines))
	}
	return lines[n-1].Dir, []byte(lines[n-1].Msg)
}

// ProbeOut returns the msg bytes of every "out" line from line from to line to (1-based, inclusive), in order.
func ProbeOut(t testing.TB, path string, from, to int) [][]byte {
	t.Helper()
	lines := probeLines(t, path)
	if from < 1 || to > len(lines) || from > to {
		t.Fatalf("%s: lines %d..%d out of 1..%d", path, from, to, len(lines))
	}
	var out [][]byte
	for _, l := range lines[from-1 : to] {
		if l.Dir == "out" {
			out = append(out, []byte(l.Msg))
		}
	}
	return out
}
