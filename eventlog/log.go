package eventlog

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sync"
	"time"
)

const maxEntries = 2000

// Entry is a single event log entry.
type Entry struct {
	Seq int64           `json:"seq"`
	T   float64         `json:"t"`
	Dir string          `json:"dir"`
	Msg json.RawMessage `json:"msg"`
}

// Log is an append-only JSON lines event log.
type Log struct {
	mu      sync.Mutex
	path    string
	f       *os.File
	last    int64
	entries []Entry
}

// Open opens or creates the log at path, reading any existing entries.
func Open(path string) (*Log, error) {
	if dir := filepath.Dir(path); dir != "" {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			return nil, fmt.Errorf("eventlog: open %s: %w", path, err)
		}
	}
	f, err := os.OpenFile(path, os.O_RDWR|os.O_CREATE, 0o644)
	if err != nil {
		return nil, fmt.Errorf("eventlog: open %s: %w", path, err)
	}
	l := &Log{path: path, f: f}
	if err := l.read(); err != nil {
		f.Close()
		return nil, err
	}
	return l, nil
}

func (l *Log) read() error {
	if _, err := l.f.Seek(0, io.SeekStart); err != nil {
		return fmt.Errorf("eventlog: open %s: %w", l.path, err)
	}
	r := bufio.NewReader(l.f)
	n := 0
	for {
		line, err := r.ReadBytes('\n')
		if len(line) > 0 {
			n++
			trimmed := bytes.TrimRight(line, "\r\n")
			var e Entry
			if derr := json.Unmarshal(trimmed, &e); derr != nil {
				return fmt.Errorf("eventlog: read %s line %d: %w", l.path, n, derr)
			}
			l.entries = append(l.entries, e)
			l.last = e.Seq
			if len(l.entries) > maxEntries {
				l.entries = l.entries[len(l.entries)-maxEntries:]
			}
		}
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("eventlog: read %s line %d: %w", l.path, n+1, err)
		}
	}
	return nil
}

// Append compacts msg, appends an entry and returns it.
func (l *Log) Append(dir string, msg json.RawMessage, now time.Time) (Entry, error) {
	var buf bytes.Buffer
	if err := json.Compact(&buf, msg); err != nil {
		return Entry{}, fmt.Errorf("eventlog: bad msg: %w", err)
	}

	l.mu.Lock()
	defer l.mu.Unlock()

	e := Entry{
		Seq: l.last + 1,
		T:   float64(now.UnixMilli()) / 1000,
		Dir: dir,
		Msg: json.RawMessage(buf.Bytes()),
	}
	data, err := json.Marshal(e)
	if err != nil {
		return Entry{}, fmt.Errorf("eventlog: bad msg: %w", err)
	}
	data = append(data, '\n')
	if _, err := l.f.Write(data); err != nil {
		return Entry{}, fmt.Errorf("eventlog: open %s: %w", l.path, err)
	}
	l.last = e.Seq
	l.entries = append(l.entries, e)
	if len(l.entries) > maxEntries {
		l.entries = l.entries[len(l.entries)-maxEntries:]
	}
	return e, nil
}

// Since returns copies of kept entries with Seq > after, at most max.
func (l *Log) Since(after int64, max int) []Entry {
	l.mu.Lock()
	defer l.mu.Unlock()
	if max <= 0 {
		max = 200
	}
	out := []Entry{}
	for _, e := range l.entries {
		if e.Seq <= after {
			continue
		}
		cp := Entry{Seq: e.Seq, T: e.T, Dir: e.Dir}
		if e.Msg != nil {
			cp.Msg = append(json.RawMessage(nil), e.Msg...)
		}
		out = append(out, cp)
		if len(out) >= max {
			break
		}
	}
	return out
}

// Last returns the Seq of the last entry, or 0 for an empty log.
func (l *Log) Last() int64 {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.last
}

// Path returns the path the log was opened with.
func (l *Log) Path() string {
	return l.path
}
