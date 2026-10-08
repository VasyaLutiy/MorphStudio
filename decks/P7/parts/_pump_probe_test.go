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
	"sync"
	"testing"
	"time"

	"morphstudio/claude"
	"morphstudio/control"
	"morphstudio/internal/testhelp"
	"morphstudio/queue"
	"morphstudio/registry"
	"morphstudio/runner"
	"morphstudio/stream"
	"morphstudio/supervisor"
	"morphstudio/telegram"
)

const pPUProbe = "../tests/fixtures/stream/probe.jsonl"

// pPUH is the harness of the Pump examples: the fakes of Daemon Core example 1, guarded by one mutex.
type pPUH struct {
	t        *testing.T
	tmp      string
	state    string
	reg      *registry.Registry
	run      *runner.Fake
	mu       sync.Mutex
	launches []claude.Launch
	procs    []*claude.Fake
	posts    [][4]string
	clock    time.Time
	ids      int
	d        *Daemon
	fp       *claude.Fake
}

func pPUNew(t *testing.T) *pPUH {
	t.Helper()
	h := &pPUH{t: t, tmp: t.TempDir(), clock: time.Date(2026, 10, 8, 15, 0, 0, 0, time.UTC)}
	h.state = filepath.Join(h.tmp, "state")
	reg, err := registry.Open(h.state)
	if err != nil {
		t.Fatal(err)
	}
	h.reg = reg
	dir := filepath.Join(h.tmp, "projects", "demo")
	measure, err := os.ReadFile("../tests/fixtures/git/measure.md")
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(dir, "docs"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "docs", "MEASURE.md"), measure, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := reg.Add(registry.Project{Name: "demo", Language: "go", RepoURL: "https://github.com/acme/demo", Dir: dir,
		Caps: queue.Caps{ClaudeUSD: 30, Hours: 3, ExecutorUSD: 5}, CreatedAt: time.Date(2026, 10, 8, 14, 0, 0, 0, time.UTC)}); err != nil {
		t.Fatal(err)
	}
	h.run = &runner.Fake{Script: map[string]runner.Result{
		"git rev-parse --abbrev-ref HEAD": {Stdout: "main\n"},
		"git rev-parse HEAD":              {Stdout: "aaa111\n"},
		"git rev-parse origin/main":       {Stdout: "aaa111\n"},
		"git status --porcelain":          {Stdout: ""},
	}}
	deps := Deps{
		Registry: h.reg,
		Runner:   h.run,
		Spawn: func(ctx context.Context, l claude.Launch) (claude.Process, error) {
			h.mu.Lock()
			defer h.mu.Unlock()
			f := claude.NewFake()
			h.launches = append(h.launches, l)
			h.procs = append(h.procs, f)
			return f, nil
		},
		Post: func(ctx context.Context, project, kind, headline, numbers string) (telegram.Status, error) {
			h.mu.Lock()
			defer h.mu.Unlock()
			h.posts = append(h.posts, [4]string{project, kind, headline, numbers})
			return telegram.Status{}, nil
		},
		GitHubDo: func(r *http.Request) (*http.Response, error) { return nil, errors.New("no network in tests") },
		Now: func() time.Time {
			h.mu.Lock()
			defer h.mu.Unlock()
			now := h.clock
			h.clock = h.clock.Add(time.Second)
			return now
		},
		NewID: func() string {
			h.mu.Lock()
			defer h.mu.Unlock()
			h.ids++
			return fmt.Sprintf("id-%d", h.ids)
		},
		Config: Config{ClaudeBin: "/opt/claude/bin/claude", MorphBin: "/usr/local/bin/morph", ProjectsDir: filepath.Join(h.tmp, "projects"),
			MCPBaseURL: "http://127.0.0.1:7181", Defaults: queue.Caps{ClaudeUSD: 25, Hours: 2, ExecutorUSD: 4}, MaxParallel: 1,
			Loop: supervisor.Config{MaxResumesPerHour: 3, StallMinutes: 30, StretchUSD: 30, UsageAlertPercent: 50}},
	}
	d, err := Open(context.Background(), deps)
	if err != nil || d == nil {
		t.Fatalf("setup: Open = %v, %v", d, err)
	}
	h.d = d
	b, err := os.ReadFile("../tests/fixtures/plan/queue-3.json")
	if err != nil {
		t.Fatal(err)
	}
	var plan queue.Plan
	if err := json.Unmarshal(b, &plan); err != nil {
		t.Fatal(err)
	}
	if v, err := d.PlanLoad("demo", plan); err != nil || v.State != "running" {
		t.Fatalf("setup: PlanLoad(demo) = %+v, %v", v, err)
	}
	if _, procs := h.spawns(); len(procs) != 1 {
		t.Fatalf("setup: %d spawns after PlanLoad, want 1", len(procs))
	}
	_, procs := h.spawns()
	h.fp = procs[0]
	return h
}

func (h *pPUH) spawns() ([]claude.Launch, []*claude.Fake) {
	h.mu.Lock()
	defer h.mu.Unlock()
	return append([]claude.Launch(nil), h.launches...), append([]*claude.Fake(nil), h.procs...)
}

func (h *pPUH) postList() [][4]string {
	h.mu.Lock()
	defer h.mu.Unlock()
	return append([][4]string(nil), h.posts...)
}

func (h *pPUH) line(n int) []byte {
	_, msg := testhelp.ProbeLine(h.t, pPUProbe, n)
	return msg
}

// pPUPoll calls f every 10 ms for up to 2 s until it reports true; it returns the last description.
func pPUPoll(f func() (bool, string)) (bool, string) {
	deadline := time.Now().Add(2 * time.Second)
	for {
		ok, last := f()
		if ok || time.Now().After(deadline) {
			return ok, last
		}
		time.Sleep(10 * time.Millisecond)
	}
}

func pPUStatus(h *pPUH) control.Status {
	s, err := h.d.Status("demo")
	if err != nil {
		h.t.Errorf("Status(demo) error %v", err)
	}
	return s
}

func pPUCompact(b []byte) string {
	var buf bytes.Buffer
	if err := json.Compact(&buf, b); err != nil {
		return "not json: " + string(b)
	}
	return buf.String()
}

func pPUDecode(b []byte) any {
	var v any
	if err := json.Unmarshal(b, &v); err != nil {
		return "not json: " + string(b)
	}
	return v
}

// pPUReady is example 1: the four probe lines emitted, the turn finished.
func pPUReady(t *testing.T) *pPUH {
	h := pPUNew(t)
	for _, n := range []int{4, 5, 6, 7} {
		h.fp.Emit(h.line(n))
	}
	ok, last := pPUPoll(func() (bool, string) {
		s := pPUStatus(h)
		return s.CostUSD > 0, fmt.Sprintf("%+v", s)
	})
	if !ok {
		t.Fatalf("example 1: Status.CostUSD still 0 after 2 s: %s", last)
	}
	return h
}

func TestProbePumpExample1(t *testing.T) {
	h := pPUNew(t)
	for _, n := range []int{4, 5, 6, 7} {
		h.fp.Emit(h.line(n))
	}
	ok, last := pPUPoll(func() (bool, string) {
		s := pPUStatus(h)
		return s.CostUSD > 0, fmt.Sprintf("%+v", s)
	})
	if !ok {
		t.Errorf("example 1 poll Status until CostUSD > 0: still %s", last)
	}
	s := pPUStatus(h)
	testhelp.Equal(t, "example 1 Status", []any{s.State, s.Phase, s.CostUSD, s.FiveHour, s.SevenDay, s.SessionID},
		[]any{"ready", "P17", 0.0021784900000000004, 22, 60, "967e8f4d-a2eb-4aa6-968b-b9054a1c3d7e"})
	ev, err := h.d.Events("demo", 0, 10)
	got := []string{}
	for _, e := range ev.Entries {
		got = append(got, fmt.Sprintf("%d %s %s", e.Seq, e.Dir, e.Msg))
	}
	want := []string{"1 in " + pPUCompact(stream.User("/morph-orchestrator P17"))}
	for i, n := range []int{4, 5, 6, 7} {
		want = append(want, fmt.Sprintf("%d out %s", i+2, pPUCompact(h.line(n))))
	}
	testhelp.Equal(t, "example 1 Events(demo, 0, 10)", []any{got, err == nil}, []any{want, true})
	testhelp.Equal(t, "example 1 the posts after the start", h.postList()[1:], [][4]string{
		{"demo", "info", "P17: weekly usage 60%", "resets 2026-10-13T01:00:00Z"},
		{"demo", "idle", "P17 turn ended: PONG", "success · turns 1 · $0.0022"}})
	u, err := h.d.Usage("demo")
	limitsAt := u.LimitsAt
	u.LimitsAt = time.Time{}
	testhelp.Equal(t, "example 1 Usage(demo)", []any{u, err == nil, limitsAt.IsZero()}, []any{control.Usage{FiveHour: 22, SevenDay: 60,
		FiveHourResetsAt: 1791469200, SevenDayResetsAt: 1791853200, SessionCostUSD: 0.0021784900000000004, StretchCostUSD: 0.0021784900000000004}, true, false})
	// variant: the state file holds the loop's session cost
	var st State
	err = h.reg.LoadState("demo", &st)
	testhelp.Equal(t, "example 1 variant the saved loop", []any{err == nil, st.Loop.SessionUSD, st.Loop.State}, []any{true, 0.0021784900000000004, "running"})
}

func TestProbePumpExample2(t *testing.T) {
	h := pPUReady(t)
	h.fp.Emit(h.line(27))
	var q *control.Question
	ok, last := pPUPoll(func() (bool, string) {
		var err error
		q, err = h.d.Pending("demo")
		return q != nil, fmt.Sprintf("%v %v", q, err)
	})
	if !ok {
		t.Errorf("example 2 poll Pending: still %s", last)
	}
	if q != nil {
		testhelp.Equal(t, "example 2 Pending", []any{q.RequestID, q.Text, q.Header, q.Options, q.AskedAt.IsZero()},
			[]any{"9f4ffa22-2676-4391-bfd9-bd7d16c3866c", "Which option do you want: A or B?", "A or B", []string{"Option A", "Option B"}, false})
	}
	testhelp.Equal(t, "example 2 Status.State", pPUStatus(h).State, "question")
	posts := h.postList()
	testhelp.Equal(t, "example 2 the last post", posts[len(posts)-1], [4]string{"demo", "ask", "Which option do you want: A or B?", "1 Option A · 2 Option B"})
	err := h.d.Answer("demo", 2)
	testhelp.Equal(t, "example 2 Answer(demo, 2)", err == nil, true)
	w := h.fp.Written()
	testhelp.Equal(t, "example 2 fp.Written() last decodes to line 28", pPUDecode(w[len(w)-1]), pPUDecode(h.line(28)))
	ev, _ := h.d.Events("demo", 0, 100)
	lastDir := ""
	if len(ev.Entries) > 0 {
		lastDir = ev.Entries[len(ev.Entries)-1].Dir
	}
	testhelp.Equal(t, "example 2 the log's last entry dir", lastDir, "in")
	testhelp.Equal(t, "example 2 Status.State after the answer", pPUStatus(h).State, "busy")
}

func TestProbePumpExample3(t *testing.T) {
	h := pPUReady(t)
	r, err := h.d.Order("demo", "next step")
	testhelp.Equal(t, "example 3 Order(demo, next step)", []any{r.Sent, r.Queued, err == nil}, []any{true, 0, true})
	h.fp.Emit([]byte(`{"type":"control_request","request_id":"r-7","request":{"subtype":"can_use_tool","tool_name":"Bash","input":{"command":"ls"},"tool_use_id":"toolu_x"}}`))
	ok, last := pPUPoll(func() (bool, string) {
		ev, _ := h.d.Events("demo", 0, 100)
		return ev.Last == 8, fmt.Sprintf("Last %d", ev.Last)
	})
	if !ok {
		t.Errorf("example 3 poll Events until Last is 8: %s", last)
	}
	allow, _ := stream.Allow("r-7", json.RawMessage(`{"command":"ls"}`))
	w := h.fp.Written()
	testhelp.Equal(t, "example 3 fp.Written() last", string(w[len(w)-1]), string(allow))
	testhelp.Equal(t, "example 3 Status.State", pPUStatus(h).State, "busy")
}

func TestProbePumpExample4(t *testing.T) {
	h := pPUReady(t)
	before := len(h.postList())
	h.fp.Emit([]byte("not json"))
	h.fp.Emit(h.line(10))
	var ev control.Events
	ok, last := pPUPoll(func() (bool, string) {
		ev, _ = h.d.Events("demo", 5, 100)
		return ev.Last == 7, fmt.Sprintf("Last %d", ev.Last)
	})
	if !ok {
		t.Errorf("example 4 poll Events until Last is 7: %s", last)
	}
	got := []string{}
	for _, e := range ev.Entries {
		got = append(got, fmt.Sprintf("%d %s %s", e.Seq, e.Dir, e.Msg))
	}
	testhelp.Equal(t, "example 4 the two new entries", got, []string{`6 out "not json"`, "7 out " + pPUCompact(h.line(10))})
	testhelp.Equal(t, "example 4 Status.State", pPUStatus(h).State, "ready")
	testhelp.Equal(t, "example 4 no post", len(h.postList()), before)
	// variant: a turn without text, then a result over the stretch cap kills the process (no new spawn)
	_, _ = h.d.Order("demo", "again")
	h.fp.Emit([]byte(`{"type":"result","subtype":"success","is_error":false,"num_turns":2,"result":"","total_cost_usd":0.003}`))
	pPUPoll(func() (bool, string) { ev, _ := h.d.Events("demo", 0, 100); return ev.Last == 9, "" })
	posts := h.postList()
	testhelp.Equal(t, "example 4 variant the turn without text", posts[len(posts)-1], [4]string{"demo", "idle", "P17 turn ended: (no text)", "success · turns 2 · $0.0030"})
	_, _ = h.d.Order("demo", "more")
	h.fp.Emit([]byte(`{"type":"result","subtype":"success","is_error":false,"num_turns":3,"result":"x","total_cost_usd":31}`))
	ok, last = pPUPoll(func() (bool, string) { s := pPUStatus(h); return s.State == "waiting", s.State })
	testhelp.Equal(t, "example 4 variant the stretch cap stops and kills", []any{ok, last, h.fp.Killed()}, []any{true, "waiting", true})
	h.d.Wait("demo")
	pPUPoll(func() (bool, string) {
		_, err := h.d.Order("demo", "x")
		return errors.Is(err, control.ErrNoSession), ""
	})
}

func TestProbePumpExample5(t *testing.T) {
	h := pPUReady(t)
	h.run.Script["git status --porcelain"] = runner.Result{Stdout: " M notes.md\n"}
	before := len(h.postList())
	h.fp.Finish(4)
	ok, last := pPUPoll(func() (bool, string) {
		l, _ := h.spawns()
		return len(l) == 2, fmt.Sprintf("%d spawns", len(l))
	})
	if !ok {
		t.Errorf("example 5 poll until a second spawn: %s", last)
	}
	h.d.Wait("demo")
	posts := h.postList()
	testhelp.Equal(t, "example 5 the posts after the exit", posts[before:], [][4]string{
		{"demo", "watchdog", "P17: session exited (code 4), resuming", "resume 1 of 3 this hour"},
		{"demo", "start", "P17 resumed", "session id-1 · dirty tree (1 paths)"}})
	launches, _ := h.spawns()
	if len(launches) == 2 {
		testhelp.Equal(t, "example 5 the second launch", []any{launches[1].SessionID, launches[1].Resume, launches[1].BudgetUSD, launches[1].MCPConfigPath},
			[]any{"id-1", true, 30.0, filepath.Join(h.state, "demo", "morphd-mcp.json")})
	}
	tok, _ := h.reg.ReadSecret("demo", "session-token")
	testhelp.Equal(t, "example 5 the new session token", tok, "id-3")
	var st State
	b, err := os.ReadFile(filepath.Join(h.state, "demo", "state.json"))
	if err == nil {
		err = json.Unmarshal(b, &st)
	}
	testhelp.Equal(t, "example 5 the state file", []any{err == nil, len(st.Loop.Resumes), st.Loop.State}, []any{true, 1, "running"})
	s := pPUStatus(h)
	testhelp.Equal(t, "example 5 Status State, SessionID", []string{s.State, s.SessionID}, []string{"ready", "id-1"})
}

func TestProbePumpExample6(t *testing.T) {
	h := pPUReady(t)
	before := len(h.postList())
	err := h.d.Restart("demo", true)
	testhelp.Equal(t, "example 6 Restart(demo, true)", err == nil, true)
	launches, _ := h.spawns()
	if testhelp.Equal(t, "example 6 two spawns", len(launches), 2) {
		testhelp.Equal(t, "example 6 the second launch", []any{launches[1].SessionID, launches[1].Resume}, []any{"id-3", false})
	}
	h.fp.Finish(0)
	done := make(chan struct{})
	go func() { h.d.Wait("demo"); close(done) }()
	select {
	case <-done:
	case <-time.After(2 * time.Second):
		t.Errorf("example 6 d.Wait(demo) did not return within 2 s")
	}
	testhelp.Equal(t, "example 6 the posts after the restart", h.postList()[before:], [][4]string{
		{"demo", "start", "P17 started", "session id-3 · cap $30 · 3 h"}})
	launches, _ = h.spawns()
	testhelp.Equal(t, "example 6 no third spawn", len(launches), 2)
	s := pPUStatus(h)
	testhelp.Equal(t, "example 6 Status SessionID, State", []string{s.SessionID, s.State}, []string{"id-3", "busy"})
	// variant: a line of the old process is logged in its own log but changes nothing
	ev, _ := h.d.Events("demo", 0, 100)
	testhelp.Equal(t, "example 6 variant the new log holds the new first line only", ev.Last, int64(1))
}

func TestProbePumpExample7(t *testing.T) {
	h := pPUReady(t)
	h.fp.Emit([]byte(`{"type":"rate_limit_event","rate_limit_info":{"unifiedWindows":{"five_hour":{"utilization":1.0,"resetsAt":1791469200},"seven_day":{"utilization":0.6,"resetsAt":1791853200}}}}`))
	ok, last := pPUPoll(func() (bool, string) {
		s := pPUStatus(h)
		return s.State == "paused", s.State
	})
	if !ok {
		t.Errorf("example 7 poll Status until paused: still %q", last)
	}
	posts := h.postList()
	testhelp.Equal(t, "example 7 the pause post", posts[len(posts)-1], [4]string{"demo", "watchdog", "P17: usage limit, paused", "until 2026-10-08T14:20:00Z"})
	h.d.Tick(context.Background())
	w := h.fp.Written()
	testhelp.Equal(t, "example 7 fp.Written() last after Tick", string(w[len(w)-1]),
		string(stream.User("Continue by docs/AUTONOMY.md from where you stopped; the usage window has reset.")))
	testhelp.Equal(t, "example 7 Status.State after Tick", pPUStatus(h).State, "busy")
	posts = h.postList()
	testhelp.Equal(t, "example 7 the resume post", posts[len(posts)-1], [4]string{"demo", "watchdog", "P17: resumed after the limit", ""})
}
