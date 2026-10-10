package daemon

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"morphstudio/claude"
	"morphstudio/control"
	"morphstudio/eventlog"
	"morphstudio/internal/testhelp"
	"morphstudio/queue"
	"morphstudio/registry"
	"morphstudio/runner"
	"morphstudio/session"
	"morphstudio/stream"
	"morphstudio/supervisor"
	"morphstudio/telegram"
)

// puProbe is the recorded claude stream every pump example replays.
const puProbe = "../tests/fixtures/stream/probe.jsonl"

// puPlanPath is the three-phase plan the examples walk.
const puPlanPath = "../tests/fixtures/plan/queue-3.json"

// puMeasurePath is the measurement table the git end check reads.
const puMeasurePath = "../tests/fixtures/git/measure.md"

// puCrashScript is the claude stand-in of the crash example: it never writes
// stdout, writes two stderr lines and exits 1 at once.
const puCrashScript = "#!/bin/sh\n" +
	"echo 'Error: Invalid session ID. Must be a valid UUID.' >&2\n" +
	"echo 'second line' >&2\n" +
	"exit 1\n"

// puHarness is the one fixture of this file: one project "demo" with a fake
// runner, recorded spawns, posts and clock. The daemon calls Spawn, Post, Now
// and NewID from its pump goroutine, so every recorder is guarded by one
// mutex.
type puHarness struct {
	tmp     string
	reg     *registry.Registry
	rf      *runner.Fake
	d       *Daemon
	deps    Deps
	spawnFn func(ctx context.Context, l claude.Launch) (claude.Process, error)

	mu       sync.Mutex
	now      time.Time
	nextID   int
	launches []claude.Launch
	procs    []*claude.Fake
	posts    [][4]string
}

// puBare builds the fixture of example 1 without opening the daemon, so a test
// can still choose its Spawn or its Config.
func puBare(t testing.TB) *puHarness {
	t.Helper()
	tmp := t.TempDir()
	reg, err := registry.Open(filepath.Join(tmp, "state"))
	if err != nil {
		t.Fatal(err)
	}
	dir := filepath.Join(tmp, "projects", "demo")
	if err := os.MkdirAll(filepath.Join(dir, "docs"), 0o755); err != nil {
		t.Fatal(err)
	}
	measure, err := os.ReadFile(puMeasurePath)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "docs", "MEASURE.md"), measure, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := reg.Add(registry.Project{
		Name:      "demo",
		Language:  "go",
		RepoURL:   "https://github.com/acme/demo",
		Dir:       dir,
		Caps:      queue.Caps{ClaudeUSD: 30, Hours: 3, ExecutorUSD: 5},
		CreatedAt: time.Date(2026, 10, 8, 14, 0, 0, 0, time.UTC),
	}); err != nil {
		t.Fatal(err)
	}
	h := &puHarness{
		tmp: tmp,
		reg: reg,
		rf: &runner.Fake{Script: map[string]runner.Result{
			"git rev-parse --abbrev-ref HEAD": {Stdout: "main\n"},
			"git rev-parse HEAD":              {Stdout: "aaa111\n"},
			"git rev-parse origin/main":       {Stdout: "aaa111\n"},
			"git status --porcelain":          {Stdout: ""},
		}},
		now: time.Date(2026, 10, 8, 15, 0, 0, 0, time.UTC),
	}
	h.spawnFn = h.fakeSpawn
	return h
}

// puFakeSpawn records the launch and hands back a fresh fake process.
func (h *puHarness) fakeSpawn(ctx context.Context, l claude.Launch) (claude.Process, error) {
	fp := claude.NewFake()
	h.mu.Lock()
	h.launches = append(h.launches, l)
	h.procs = append(h.procs, fp)
	h.mu.Unlock()
	return fp, nil
}

// puPost records one post.
func (h *puHarness) post(ctx context.Context, project, kind, headline, numbers string) (telegram.Status, error) {
	h.mu.Lock()
	h.posts = append(h.posts, [4]string{project, kind, headline, numbers})
	h.mu.Unlock()
	return telegram.Status{}, nil
}

// puNow returns the clock and advances it by one second.
func (h *puHarness) puNow() time.Time {
	h.mu.Lock()
	defer h.mu.Unlock()
	now := h.now
	h.now = h.now.Add(time.Second)
	return now
}

// puNewID returns id-1, id-2, ... in call order.
func (h *puHarness) puNewID() string {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.nextID++
	return fmt.Sprintf("id-%d", h.nextID)
}

// puOpen wires the dependencies and opens the daemon.
func (h *puHarness) open(t testing.TB) {
	t.Helper()
	h.deps.Registry = h.reg
	h.deps.Runner = h.rf
	h.deps.Spawn = h.spawnFn
	h.deps.Post = h.post
	h.deps.GitHubDo = func(*http.Request) (*http.Response, error) {
		return nil, errors.New("no network")
	}
	h.deps.Now = h.puNow
	h.deps.NewID = h.puNewID
	h.deps.Config.ClaudeBin = "/opt/claude/bin/claude"
	h.deps.Config.MorphBin = "/usr/local/bin/morph"
	h.deps.Config.ProjectsDir = filepath.Join(h.tmp, "projects")
	h.deps.Config.MCPBaseURL = "http://127.0.0.1:7181"
	h.deps.Config.Defaults = queue.Caps{ClaudeUSD: 25, Hours: 2, ExecutorUSD: 4}
	h.deps.Config.MaxParallel = 1
	h.deps.Config.Loop = supervisor.Config{
		MaxResumesPerHour: 3,
		StallMinutes:      30,
		StretchUSD:        30,
		UsageAlertPercent: 50,
	}
	d, err := Open(context.Background(), h.deps)
	if err != nil {
		t.Fatal(err)
	}
	h.d = d
}

// puLoad installs the three-phase fixture plan.
func puLoad(t testing.TB, h *puHarness) {
	t.Helper()
	data, err := os.ReadFile(puPlanPath)
	if err != nil {
		t.Fatal(err)
	}
	var plan queue.Plan
	if err := json.Unmarshal(data, &plan); err != nil {
		t.Fatal(err)
	}
	if _, err := h.d.PlanLoad("demo", plan); err != nil {
		t.Fatal(err)
	}
}

// puEmit delivers one probe line to the fake process.
func puEmit(t testing.TB, fp *claude.Fake, line int) {
	t.Helper()
	_, msg := testhelp.ProbeLine(t, puProbe, line)
	fp.Emit(msg)
}

// puPong replays the four lines of example 1 and waits for the turn's cost.
func puPong(t testing.TB, h *puHarness) *claude.Fake {
	t.Helper()
	fp := h.procAt(0)
	for _, n := range []int{4, 5, 6, 7} {
		puEmit(t, fp, n)
	}
	puPoll(t, "cost", func() float64 {
		s, _ := h.d.Status("demo")
		return s.CostUSD
	}, func(v float64) bool { return v > 0 })
	return fp
}

// puReady1 brings a fresh harness to the state after example 1.
func puReady1(t testing.TB) *puHarness {
	t.Helper()
	h := puBare(t)
	h.open(t)
	puLoad(t, h)
	puPong(t, h)
	return h
}

// puLaunches returns a copy of every recorded launch.
func (h *puHarness) launchesCopy() []claude.Launch {
	h.mu.Lock()
	defer h.mu.Unlock()
	return append([]claude.Launch(nil), h.launches...)
}

// puPostsCopy returns a copy of every recorded post.
func (h *puHarness) postsCopy() [][4]string {
	h.mu.Lock()
	defer h.mu.Unlock()
	return append([][4]string(nil), h.posts...)
}

// puProcAt returns the i-th fake process.
func (h *puHarness) procAt(i int) *claude.Fake {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.procs[i]
}

// puSecret reads one project secret.
func puSecret(t testing.TB, h *puHarness, name string) string {
	t.Helper()
	v, err := h.reg.ReadSecret("demo", name)
	if err != nil {
		t.Fatal(err)
	}
	return v
}

// puState reads the project's persisted state.
func puState(t testing.TB, h *puHarness) State {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(h.tmp, "state", "demo", "state.json"))
	if err != nil {
		t.Fatal(err)
	}
	var st State
	if err := json.Unmarshal(data, &st); err != nil {
		t.Fatal(err)
	}
	return st
}

// puSessionPath is the JSONL log of one session.
func puSessionPath(h *puHarness, id string) string {
	return filepath.Join(h.tmp, "state", "demo", "sessions", id+".jsonl")
}

// puReadLog decodes every line of a session log.
func puReadLog(t testing.TB, path string) []eventlog.Entry {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	text := strings.TrimSuffix(string(data), "\n")
	if text == "" {
		return nil
	}
	lines := strings.Split(text, "\n")
	out := make([]eventlog.Entry, 0, len(lines))
	for _, line := range lines {
		var e eventlog.Entry
		if err := json.Unmarshal([]byte(line), &e); err != nil {
			t.Fatalf("%s: %v", path, err)
		}
		out = append(out, e)
	}
	return out
}

// puCompact strips the whitespace of a JSON document.
func puCompact(t testing.TB, b []byte) []byte {
	t.Helper()
	var buf bytes.Buffer
	if err := json.Compact(&buf, b); err != nil {
		t.Fatalf("compact: %v", err)
	}
	return buf.Bytes()
}

// puPoll calls get every 10 ms until ok holds, and fails after five seconds.
func puPoll[T any](t testing.TB, what string, get func() T, ok func(T) bool) T {
	t.Helper()
	deadline := time.Now().Add(5 * time.Second)
	for {
		v := get()
		if ok(v) {
			return v
		}
		if !time.Now().Before(deadline) {
			t.Fatalf("%s: timeout, last %#v", what, v)
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func TestPumpExample1(t *testing.T) {
	h := puBare(t)
	h.open(t)
	puLoad(t, h)

	ls := h.launchesCopy()
	if !testhelp.Equal(t, "launches", len(ls), 1) {
		t.FailNow()
	}
	testhelp.Equal(t, "session", ls[0].SessionID, "id-1")
	testhelp.Equal(t, "token", puSecret(t, h, "session-token"), "id-2")
	testhelp.Equal(t, "start post", h.postsCopy()[0], [4]string{
		"demo", "start", "P17 started", "session id-1 · cap $30 · 3 h",
	})

	fp := puPong(t, h)

	s, err := h.d.Status("demo")
	testhelp.Equal(t, "status err", err, nil)
	testhelp.Equal(t, "state", s.State, "ready")
	testhelp.Equal(t, "phase", s.Phase, "P17")
	testhelp.Equal(t, "cost", s.CostUSD, 0.0021784900000000004)
	testhelp.Equal(t, "five hour", s.FiveHour, 22)
	testhelp.Equal(t, "seven day", s.SevenDay, 60)
	testhelp.Equal(t, "session", s.SessionID, "967e8f4d-a2eb-4aa6-968b-b9054a1c3d7e")

	ev, err := h.d.Events("demo", 0, 10)
	testhelp.Equal(t, "events err", err, nil)
	if !testhelp.Equal(t, "entries", len(ev.Entries), 5) {
		t.FailNow()
	}
	testhelp.Equal(t, "last", ev.Last, int64(5))
	testhelp.Equal(t, "seq 1", ev.Entries[0].Seq, int64(1))
	testhelp.Equal(t, "dir 1", ev.Entries[0].Dir, "in")
	testhelp.Equal(t, "msg 1", puCompact(t, []byte(ev.Entries[0].Msg)),
		puCompact(t, stream.User("/morph-orchestrator P17")))
	for i, n := range []int{4, 5, 6, 7} {
		_, msg := testhelp.ProbeLine(t, puProbe, n)
		e := ev.Entries[i+1]
		testhelp.Equal(t, fmt.Sprintf("seq %d", i+2), e.Seq, int64(i+2))
		testhelp.Equal(t, fmt.Sprintf("dir %d", i+2), e.Dir, "out")
		testhelp.Equal(t, fmt.Sprintf("msg %d", i+2), puCompact(t, []byte(e.Msg)), puCompact(t, msg))
	}

	testhelp.Equal(t, "posts", h.postsCopy()[1:], [][4]string{
		{"demo", "info", "P17: weekly usage 60%", "resets 2026-10-13T01:00:00Z"},
		{"demo", "idle", "P17 turn ended: PONG", "success · turns 1 · $0.0022"},
	})

	u, err := h.d.Usage("demo")
	testhelp.Equal(t, "usage err", err, nil)
	testhelp.Equal(t, "five hour", u.FiveHour, 22)
	testhelp.Equal(t, "seven day", u.SevenDay, 60)
	testhelp.Equal(t, "five hour reset", u.FiveHourResetsAt, int64(1791469200))
	testhelp.Equal(t, "seven day reset", u.SevenDayResetsAt, int64(1791853200))
	testhelp.Equal(t, "session cost", u.SessionCostUSD, 0.0021784900000000004)
	testhelp.Equal(t, "stretch cost", u.StretchCostUSD, 0.0021784900000000004)
	testhelp.Equal(t, "limits at zero", u.LimitsAt.IsZero(), false)

	_ = fp
}

func TestPumpExample2(t *testing.T) {
	h := puReady1(t)
	fp := h.procAt(0)
	puEmit(t, fp, 27)

	puPoll(t, "pending", func() *control.Question {
		p, _ := h.d.Pending("demo")
		return p
	}, func(p *control.Question) bool { return p != nil })

	p, err := h.d.Pending("demo")
	testhelp.Equal(t, "pending err", err, nil)
	if !testhelp.Equal(t, "pending set", p != nil, true) {
		t.FailNow()
	}
	testhelp.Equal(t, "request", p.RequestID, "9f4ffa22-2676-4391-bfd9-bd7d16c3866c")
	testhelp.Equal(t, "text", p.Text, "Which option do you want: A or B?")
	testhelp.Equal(t, "header", p.Header, "A or B")
	testhelp.Equal(t, "options", p.Options, []string{"Option A", "Option B"})
	testhelp.Equal(t, "asked at zero", p.AskedAt.IsZero(), false)

	s, err := h.d.Status("demo")
	testhelp.Equal(t, "status err", err, nil)
	testhelp.Equal(t, "state", s.State, "question")

	posts := h.postsCopy()
	testhelp.Equal(t, "ask post", posts[len(posts)-1], [4]string{
		"demo", "ask", "Which option do you want: A or B?", "1 Option A · 2 Option B",
	})

	testhelp.Equal(t, "answer err", h.d.Answer("demo", 2), nil)

	written := fp.Written()
	if !testhelp.Equal(t, "written", len(written) > 0, true) {
		t.FailNow()
	}
	var got, want any
	if err := json.Unmarshal(written[len(written)-1], &got); err != nil {
		t.Fatal(err)
	}
	_, msg28 := testhelp.ProbeLine(t, puProbe, 28)
	if err := json.Unmarshal(msg28, &want); err != nil {
		t.Fatal(err)
	}
	testhelp.Equal(t, "answer line", got, want)

	ev, err := h.d.Events("demo", 0, 100)
	testhelp.Equal(t, "events err", err, nil)
	if !testhelp.Equal(t, "entries", len(ev.Entries) > 0, true) {
		t.FailNow()
	}
	testhelp.Equal(t, "last dir", ev.Entries[len(ev.Entries)-1].Dir, "in")

	s, err = h.d.Status("demo")
	testhelp.Equal(t, "status err", err, nil)
	testhelp.Equal(t, "state", s.State, "busy")
}

func TestPumpExample3(t *testing.T) {
	h := puReady1(t)
	fp := h.procAt(0)

	res, err := h.d.Order("demo", "next step")
	testhelp.Equal(t, "order err", err, nil)
	testhelp.Equal(t, "order", res, session.OrderResult{Sent: true})

	fp.Emit([]byte(`{"type":"control_request","request_id":"r-7","request":{"subtype":"can_use_tool","tool_name":"Bash","input":{"command":"ls"},"tool_use_id":"toolu_x"}}`))

	puPoll(t, "last", func() int64 {
		ev, _ := h.d.Events("demo", 0, 100)
		return ev.Last
	}, func(v int64) bool { return v == 8 })

	want, err := stream.Allow("r-7", json.RawMessage(`{"command":"ls"}`))
	if err != nil {
		t.Fatal(err)
	}
	written := fp.Written()
	testhelp.Equal(t, "allow line", written[len(written)-1], want)

	s, err := h.d.Status("demo")
	testhelp.Equal(t, "status err", err, nil)
	testhelp.Equal(t, "state", s.State, "busy")
}

func TestPumpExample4(t *testing.T) {
	h := puReady1(t)
	fp := h.procAt(0)

	fp.Emit([]byte("not json"))
	puEmit(t, fp, 10)

	puPoll(t, "last", func() int64 {
		ev, _ := h.d.Events("demo", 5, 100)
		return ev.Last
	}, func(v int64) bool { return v == 7 })

	ev, err := h.d.Events("demo", 5, 100)
	testhelp.Equal(t, "events err", err, nil)
	if !testhelp.Equal(t, "entries", len(ev.Entries), 2) {
		t.FailNow()
	}
	testhelp.Equal(t, "seq 6", ev.Entries[0].Seq, int64(6))
	testhelp.Equal(t, "dir 6", ev.Entries[0].Dir, "out")
	testhelp.Equal(t, "msg 6", string(ev.Entries[0].Msg), `"not json"`)

	_, msg10 := testhelp.ProbeLine(t, puProbe, 10)
	testhelp.Equal(t, "seq 7", ev.Entries[1].Seq, int64(7))
	testhelp.Equal(t, "dir 7", ev.Entries[1].Dir, "out")
	testhelp.Equal(t, "msg 7", puCompact(t, []byte(ev.Entries[1].Msg)), puCompact(t, msg10))

	s, err := h.d.Status("demo")
	testhelp.Equal(t, "status err", err, nil)
	testhelp.Equal(t, "state", s.State, "ready")

	testhelp.Equal(t, "posts", len(h.postsCopy()), 3)
}

func TestPumpExample5(t *testing.T) {
	h := puReady1(t)
	fp := h.procAt(0)

	h.rf.Script["git status --porcelain"] = runner.Result{Stdout: " M notes.md\n"}

	fp.Finish(4)

	puPoll(t, "spawns", func() int { return len(h.launchesCopy()) }, func(n int) bool { return n == 2 })
	h.d.Wait("demo")

	testhelp.Equal(t, "posts", h.postsCopy()[3:], [][4]string{
		{"demo", "watchdog", "P17: session exited (code 4), restarting", "restart 1 of 3 this hour"},
		{"demo", "start", "P17 resumed", "session id-1 · dirty tree (1 paths)"},
	})

	ls := h.launchesCopy()
	testhelp.Equal(t, "session", ls[1].SessionID, "id-1")
	testhelp.Equal(t, "resume", ls[1].Resume, true)
	testhelp.Equal(t, "budget", ls[1].BudgetUSD, 30.0)
	testhelp.Equal(t, "mcp config", ls[1].MCPConfigPath, filepath.Join(h.tmp, "state", "demo", "morphd-mcp.json"))

	testhelp.Equal(t, "token", puSecret(t, h, "session-token"), "id-3")

	st := puState(t, h)
	testhelp.Equal(t, "restarts", len(st.Loop.Restarts), 1)
	testhelp.Equal(t, "loop state", st.Loop.State, "running")
	if !testhelp.Equal(t, "last exit set", st.Loop.LastExit != nil, true) {
		t.FailNow()
	}
	le := st.Loop.LastExit
	testhelp.Equal(t, "exit at zero", le.At.IsZero(), false)
	testhelp.Equal(t, "last exit", *le, control.Exit{Code: 4, At: le.At, Stderr: "", Restarts: 1})

	log := puReadLog(t, puSessionPath(h, "id-1"))
	testhelp.Equal(t, "log lines", len(log), 6)
	last := log[len(log)-1]
	testhelp.Equal(t, "last seq", last.Seq, int64(6))
	testhelp.Equal(t, "last dir", last.Dir, "exit")
	testhelp.Equal(t, "last msg", string(last.Msg), `{"code":4,"stderr":""}`)

	s, err := h.d.Status("demo")
	testhelp.Equal(t, "status err", err, nil)
	testhelp.Equal(t, "state", s.State, "ready")
	testhelp.Equal(t, "session", s.SessionID, "id-1")
	if !testhelp.Equal(t, "status exit set", s.LastExit != nil, true) {
		t.FailNow()
	}
	testhelp.Equal(t, "status exit at zero", s.LastExit.At.IsZero(), false)
	testhelp.Equal(t, "status exit", *s.LastExit, control.Exit{Code: 4, At: s.LastExit.At, Stderr: "", Restarts: 1})
}

func TestPumpExample6(t *testing.T) {
	h := puReady1(t)
	fp := h.procAt(0)

	testhelp.Equal(t, "restart err", h.d.Restart("demo", true), nil)

	fp.Finish(0)
	h.d.Wait("demo")

	ls := h.launchesCopy()
	testhelp.Equal(t, "launches", len(ls), 2)
	testhelp.Equal(t, "session", ls[1].SessionID, "id-3")
	testhelp.Equal(t, "resume", ls[1].Resume, false)

	testhelp.Equal(t, "posts", h.postsCopy()[3:], [][4]string{
		{"demo", "start", "P17 started", "session id-3 · cap $30 · 3 h"},
	})

	time.Sleep(200 * time.Millisecond)
	testhelp.Equal(t, "still two launches", len(h.launchesCopy()), 2)

	s, err := h.d.Status("demo")
	testhelp.Equal(t, "status err", err, nil)
	testhelp.Equal(t, "session", s.SessionID, "id-3")
	testhelp.Equal(t, "state", s.State, "busy")
	testhelp.Equal(t, "last exit", s.LastExit == nil, true)

	log := puReadLog(t, puSessionPath(h, "id-1"))
	testhelp.Equal(t, "log lines", len(log), 6)
	last := log[len(log)-1]
	testhelp.Equal(t, "last seq", last.Seq, int64(6))
	testhelp.Equal(t, "last dir", last.Dir, "exit")
	testhelp.Equal(t, "last msg", string(last.Msg), `{"code":-1,"stderr":""}`)

	ev, err := h.d.Events("demo", 0, 10)
	testhelp.Equal(t, "events err", err, nil)
	if !testhelp.Equal(t, "entries", len(ev.Entries), 1) {
		t.FailNow()
	}
	testhelp.Equal(t, "dir", ev.Entries[0].Dir, "in")
}

func TestPumpExample7(t *testing.T) {
	h := puBare(t)
	script := filepath.Join(t.TempDir(), "claude.sh")
	if err := os.WriteFile(script, []byte(puCrashScript), 0o755); err != nil {
		t.Fatal(err)
	}
	h.spawnFn = func(ctx context.Context, l claude.Launch) (claude.Process, error) {
		h.mu.Lock()
		h.launches = append(h.launches, l)
		h.mu.Unlock()
		return claude.Start(ctx, script, claude.Args(l), l.Dir, nil)
	}
	h.open(t)
	puLoad(t, h)

	puPoll(t, "waiting", func() string {
		s, _ := h.d.Status("demo")
		return s.State
	}, func(v string) bool { return v == "waiting" })
	h.d.Wait("demo")

	ls := h.launchesCopy()
	if !testhelp.Equal(t, "launches", len(ls), 4) {
		t.FailNow()
	}
	for i, id := range []string{"id-1", "id-3", "id-5", "id-7"} {
		testhelp.Equal(t, fmt.Sprintf("session %d", i), ls[i].SessionID, id)
		testhelp.Equal(t, fmt.Sprintf("resume %d", i), ls[i].Resume, false)
	}

	time.Sleep(200 * time.Millisecond)
	testhelp.Equal(t, "no storm", len(h.launchesCopy()), 4)

	testhelp.Equal(t, "posts", h.postsCopy(), [][4]string{
		{"demo", "start", "P17 started", "session id-1 · cap $30 · 3 h"},
		{"demo", "watchdog", "P17: session exited (code 1), restarting", "restart 1 of 3 this hour · Error: Invalid session ID. Must be a valid UUID."},
		{"demo", "start", "P17 started", "session id-3 · cap $30 · 3 h"},
		{"demo", "watchdog", "P17: session exited (code 1), restarting", "restart 2 of 3 this hour · Error: Invalid session ID. Must be a valid UUID."},
		{"demo", "start", "P17 started", "session id-5 · cap $30 · 3 h"},
		{"demo", "watchdog", "P17: session exited (code 1), restarting", "restart 3 of 3 this hour · Error: Invalid session ID. Must be a valid UUID."},
		{"demo", "start", "P17 started", "session id-7 · cap $30 · 3 h"},
		{"demo", "stop", "P17: session keeps exiting", "exited 4 times within an hour (last code 1): Error: Invalid session ID. Must be a valid UUID."},
	})

	s, err := h.d.Status("demo")
	testhelp.Equal(t, "status err", err, nil)
	testhelp.Equal(t, "state", s.State, "waiting")
	testhelp.Equal(t, "phase", s.Phase, "P17")
	testhelp.Equal(t, "session", s.SessionID, "id-7")
	testhelp.Equal(t, "limits unknown", s.LimitsUnknown, true)
	if !testhelp.Equal(t, "stop set", s.Stop != nil, true) {
		t.FailNow()
	}
	testhelp.Equal(t, "stop kind", s.Stop.Kind, "crash")
	testhelp.Equal(t, "stop reason", s.Stop.Reason, "exited 4 times within an hour (last code 1): Error: Invalid session ID. Must be a valid UUID.")
	if !testhelp.Equal(t, "last exit set", s.LastExit != nil, true) {
		t.FailNow()
	}
	testhelp.Equal(t, "last exit at zero", s.LastExit.At.IsZero(), false)
	testhelp.Equal(t, "last exit", *s.LastExit, control.Exit{
		Code:     1,
		At:       s.LastExit.At,
		Stderr:   "Error: Invalid session ID. Must be a valid UUID.",
		Restarts: 3,
	})

	ev, err := h.d.Events("demo", 0, 10)
	testhelp.Equal(t, "events err", err, nil)
	if !testhelp.Equal(t, "entries", len(ev.Entries), 2) {
		t.FailNow()
	}
	testhelp.Equal(t, "seq 1", ev.Entries[0].Seq, int64(1))
	testhelp.Equal(t, "dir 1", ev.Entries[0].Dir, "in")
	testhelp.Equal(t, "seq 2", ev.Entries[1].Seq, int64(2))
	testhelp.Equal(t, "dir 2", ev.Entries[1].Dir, "exit")
	testhelp.Equal(t, "msg 2", string(ev.Entries[1].Msg),
		`{"code":1,"stderr":"Error: Invalid session ID. Must be a valid UUID.\nsecond line"}`)

	for _, id := range []string{"id-1", "id-3", "id-5"} {
		testhelp.Equal(t, "log "+id, len(puReadLog(t, puSessionPath(h, id))), 2)
	}

	_, err = h.d.Order("demo", "x")
	testhelp.Equal(t, "order err", errors.Is(err, control.ErrNoSession), true)
}

func TestPumpExample8(t *testing.T) {
	h := puReady1(t)
	fp := h.procAt(0)

	testhelp.Equal(t, "wait err", h.d.WaitOperator("demo", "id-2", "smoke", "P17 smoke: run morphd status", ""), nil)

	fp.Finish(0)
	puPoll(t, "last", func() int64 {
		ev, _ := h.d.Events("demo", 0, 100)
		return ev.Last
	}, func(v int64) bool { return v == 6 })
	h.d.Wait("demo")

	testhelp.Equal(t, "posts", h.postsCopy()[3:], [][4]string{
		{"demo", "stop", "P17: wait_operator smoke", "P17 smoke: run morphd status"},
	})

	testhelp.Equal(t, "launches", len(h.launchesCopy()), 1)

	ev, err := h.d.Events("demo", 0, 100)
	testhelp.Equal(t, "events err", err, nil)
	if !testhelp.Equal(t, "entries", len(ev.Entries) > 0, true) {
		t.FailNow()
	}
	last := ev.Entries[len(ev.Entries)-1]
	testhelp.Equal(t, "seq", last.Seq, int64(6))
	testhelp.Equal(t, "dir", last.Dir, "exit")
	testhelp.Equal(t, "msg", string(last.Msg), `{"code":0,"stderr":""}`)

	s, err := h.d.Status("demo")
	testhelp.Equal(t, "status err", err, nil)
	testhelp.Equal(t, "state", s.State, "waiting")
	if !testhelp.Equal(t, "stop set", s.Stop != nil, true) {
		t.FailNow()
	}
	testhelp.Equal(t, "stop kind", s.Stop.Kind, "smoke")
	testhelp.Equal(t, "last exit", s.LastExit == nil, true)

	st := puState(t, h)
	testhelp.Equal(t, "restarts", len(st.Loop.Restarts), 0)

	_, err = h.d.Order("demo", "x")
	testhelp.Equal(t, "order err", errors.Is(err, control.ErrNoSession), true)

	testhelp.Equal(t, "continue err", h.d.Continue("demo"), nil)
	ls := h.launchesCopy()
	if !testhelp.Equal(t, "launches", len(ls), 2) {
		t.FailNow()
	}
	testhelp.Equal(t, "session", ls[1].SessionID, "id-3")

	s, err = h.d.Status("demo")
	testhelp.Equal(t, "status err", err, nil)
	testhelp.Equal(t, "state", s.State, "busy")
}

func TestPumpExample9(t *testing.T) {
	h := puReady1(t)
	fp := h.procAt(0)

	fp.Emit([]byte(`{"type":"rate_limit_event","rate_limit_info":{"unifiedWindows":{"five_hour":{"utilization":1.0,"resetsAt":1791469200},"seven_day":{"utilization":0.6,"resetsAt":1791853200}}}}`))

	puPoll(t, "paused", func() string {
		s, _ := h.d.Status("demo")
		return s.State
	}, func(v string) bool { return v == "paused" })

	posts := h.postsCopy()
	testhelp.Equal(t, "pause post", posts[len(posts)-1], [4]string{
		"demo", "watchdog", "P17: usage limit, paused", "until 2026-10-08T14:20:00Z",
	})

	h.d.Tick(context.Background())

	written := fp.Written()
	if !testhelp.Equal(t, "written", len(written) > 0, true) {
		t.FailNow()
	}
	testhelp.Equal(t, "resume line", written[len(written)-1],
		stream.User("Continue by docs/AUTONOMY.md from where you stopped; the usage window has reset."))

	s, err := h.d.Status("demo")
	testhelp.Equal(t, "status err", err, nil)
	testhelp.Equal(t, "state", s.State, "busy")

	posts = h.postsCopy()
	testhelp.Equal(t, "resume post", posts[len(posts)-1], [4]string{
		"demo", "watchdog", "P17: resumed after the limit", "",
	})
}

func TestPumpExample10(t *testing.T) {
	h := puBare(t)
	h.deps.Config.DoneWait = 45 * time.Second
	h.open(t)
	puLoad(t, h)
	fp := puPong(t, h)

	res, err := h.d.Order("demo", "finish the phase")
	testhelp.Equal(t, "order err", err, nil)
	testhelp.Equal(t, "order", res, session.OrderResult{Sent: true})

	testhelp.Equal(t, "phase done", h.d.PhaseDone("demo", "id-2", "P17", "P18"), nil)

	testhelp.Equal(t, "killed before", fp.Killed(), false)
	testhelp.Equal(t, "launches before", len(h.launchesCopy()), 1)
	testhelp.Equal(t, "posts before", len(h.postsCopy()), 3)

	puEmit(t, fp, 31)
	puPoll(t, "spawns", func() int { return len(h.launchesCopy()) }, func(n int) bool { return n == 2 })
	h.d.Wait("demo")

	testhelp.Equal(t, "killed", fp.Killed(), true)

	ls := h.launchesCopy()
	testhelp.Equal(t, "session", ls[1].SessionID, "id-3")
	testhelp.Equal(t, "resume", ls[1].Resume, false)
	testhelp.Equal(t, "budget", ls[1].BudgetUSD, 12.0)

	testhelp.Equal(t, "posts", h.postsCopy()[3:], [][4]string{
		{"demo", "idle", "P17 turn ended: PICKED=Option B", "success · turns 2 · $0.0058"},
		{"demo", "start", "P18 started", "session id-3 · cap $12 · 3 h"},
	})

	st := puState(t, h)
	testhelp.Equal(t, "phase", st.Loop.Phase, "P18")
	testhelp.Equal(t, "stretch", st.Loop.StretchUSD, 0.005775230000000001)
	testhelp.Equal(t, "session usd", st.Loop.SessionUSD, 0.0)

	u, err := h.d.Usage("demo")
	testhelp.Equal(t, "usage err", err, nil)
	testhelp.Equal(t, "stretch cost", u.StretchCostUSD, 0.005775230000000001)

	s, err := h.d.Status("demo")
	testhelp.Equal(t, "status err", err, nil)
	testhelp.Equal(t, "phase", s.Phase, "P18")
	testhelp.Equal(t, "state", s.State, "busy")
	testhelp.Equal(t, "session", s.SessionID, "id-3")
}

func TestPumpExample11(t *testing.T) {
	h := puBare(t)
	h.deps.Config.DoneWait = 45 * time.Second
	h.open(t)
	puLoad(t, h)
	fp := puPong(t, h)

	_, _ = h.d.Order("demo", "finish the phase")
	testhelp.Equal(t, "phase done", h.d.PhaseDone("demo", "id-2", "P17", "P18"), nil)

	fp.Finish(0)
	puPoll(t, "spawns", func() int { return len(h.launchesCopy()) }, func(n int) bool { return n == 2 })
	h.d.Wait("demo")

	testhelp.Equal(t, "posts", h.postsCopy()[3:], [][4]string{
		{"demo", "start", "P18 started", "session id-3 · cap $12 · 3 h"},
	})

	ls := h.launchesCopy()
	testhelp.Equal(t, "session", ls[1].SessionID, "id-3")
	testhelp.Equal(t, "resume", ls[1].Resume, false)
	testhelp.Equal(t, "budget", ls[1].BudgetUSD, 12.0)

	st := puState(t, h)
	testhelp.Equal(t, "phase", st.Loop.Phase, "P18")
	testhelp.Equal(t, "restarts", len(st.Loop.Restarts), 0)
	testhelp.Equal(t, "last exit", st.Loop.LastExit == nil, true)
	testhelp.Equal(t, "stretch", st.Loop.StretchUSD, 0.0021784900000000004)

	log := puReadLog(t, puSessionPath(h, "id-1"))
	if !testhelp.Equal(t, "log lines", len(log) > 0, true) {
		t.FailNow()
	}
	last := log[len(log)-1]
	testhelp.Equal(t, "last dir", last.Dir, "exit")
	testhelp.Equal(t, "last msg", string(last.Msg), `{"code":0,"stderr":""}`)

	s, err := h.d.Status("demo")
	testhelp.Equal(t, "status err", err, nil)
	testhelp.Equal(t, "phase", s.Phase, "P18")
	testhelp.Equal(t, "session", s.SessionID, "id-3")
	testhelp.Equal(t, "last exit", s.LastExit == nil, true)
}
