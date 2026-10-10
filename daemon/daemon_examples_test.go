package daemon

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"morphstudio/claude"
	"morphstudio/control"
	"morphstudio/github"
	"morphstudio/internal/testhelp"
	"morphstudio/queue"
	"morphstudio/registry"
	"morphstudio/runner"
	"morphstudio/stream"
	"morphstudio/supervisor"
	"morphstudio/telegram"
)

// dcHarness is the one test harness of this file: a daemon with a fake
// registry, runner, spawn, post, clock and id generator.
type dcHarness struct {
	t   *testing.T
	tmp string
	reg *registry.Registry
	run *runner.Fake

	mu       sync.Mutex
	launches []claude.Launch
	fakes    []*claude.Fake
	posts    [][4]string
	clock    time.Time
	n        int
	do       func(*http.Request) (*http.Response, error)
	spawnErr error
}

func dcNew(t *testing.T) *dcHarness {
	t.Helper()
	tmp := t.TempDir()
	reg, err := registry.Open(filepath.Join(tmp, "state"))
	if err != nil {
		t.Fatal(err)
	}
	h := &dcHarness{
		t:     t,
		tmp:   tmp,
		reg:   reg,
		clock: time.Date(2026, 10, 8, 15, 0, 0, 0, time.UTC),
	}
	h.run = &runner.Fake{Script: map[string]runner.Result{
		"git rev-parse --abbrev-ref HEAD": {Stdout: "main\n"},
		"git rev-parse HEAD":              {Stdout: "aaa111\n"},
		"git rev-parse origin/main":       {Stdout: "aaa111\n"},
		"git status --porcelain":          {Stdout: ""},
	}}
	h.do = dcHTTP(200, `{"permissions":{"push":true}}`)
	return h
}

// dcHTTP returns a GitHubDo fake answering code with body.
func dcHTTP(code int, body string) func(*http.Request) (*http.Response, error) {
	return func(*http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: code,
			Body:       io.NopCloser(strings.NewReader(body)),
			Header:     http.Header{},
		}, nil
	}
}

// dcPlan loads the three-phase plan of the fixtures.
func dcPlan(t *testing.T) queue.Plan {
	t.Helper()
	b, err := os.ReadFile("../tests/fixtures/plan/queue-3.json")
	if err != nil {
		t.Fatal(err)
	}
	var p queue.Plan
	if err := json.Unmarshal(b, &p); err != nil {
		t.Fatal(err)
	}
	return p
}

func (h *dcHarness) addProject(name, language, repoURL string) {
	h.t.Helper()
	dir := filepath.Join(h.tmp, "projects", name)
	err := h.reg.Add(registry.Project{
		Name:      name,
		Language:  language,
		RepoURL:   repoURL,
		Dir:       dir,
		Caps:      queue.Caps{ClaudeUSD: 30, Hours: 3, ExecutorUSD: 5},
		CreatedAt: time.Date(2026, 10, 8, 14, 0, 0, 0, time.UTC),
	})
	if err != nil {
		h.t.Fatal(err)
	}
	h.writeMeasure(dir)
}

func (h *dcHarness) writeMeasure(dir string) {
	h.t.Helper()
	b, err := os.ReadFile("../tests/fixtures/git/measure.md")
	if err != nil {
		h.t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Join(dir, "docs"), 0o755); err != nil {
		h.t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "docs", "MEASURE.md"), b, 0o644); err != nil {
		h.t.Fatal(err)
	}
}

func (h *dcHarness) now() time.Time {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.clock = h.clock.Add(time.Second)
	return h.clock
}

func (h *dcHarness) newID() string {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.n++
	return fmt.Sprintf("id-%d", h.n)
}

func (h *dcHarness) spawn(_ context.Context, l claude.Launch) (claude.Process, error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.launches = append(h.launches, l)
	if h.spawnErr != nil {
		return nil, h.spawnErr
	}
	f := claude.NewFake()
	h.fakes = append(h.fakes, f)
	return f, nil
}

func (h *dcHarness) post(_ context.Context, project, kind, headline, numbers string) (telegram.Status, error) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.posts = append(h.posts, [4]string{project, kind, headline, numbers})
	return telegram.Status{}, nil
}

func (h *dcHarness) spawnList() []claude.Launch {
	h.mu.Lock()
	defer h.mu.Unlock()
	out := make([]claude.Launch, len(h.launches))
	copy(out, h.launches)
	return out
}

func (h *dcHarness) postList() [][4]string {
	h.mu.Lock()
	defer h.mu.Unlock()
	out := make([][4]string, len(h.posts))
	copy(out, h.posts)
	return out
}

func (h *dcHarness) fake(i int) *claude.Fake {
	h.mu.Lock()
	defer h.mu.Unlock()
	return h.fakes[i]
}

func (h *dcHarness) deps() Deps {
	return Deps{
		Registry: h.reg,
		Runner:   h.run,
		Spawn:    h.spawn,
		Post:     h.post,
		GitHubDo: func(r *http.Request) (*http.Response, error) {
			h.mu.Lock()
			do := h.do
			h.mu.Unlock()
			return do(r)
		},
		Now:   h.now,
		NewID: h.newID,
		Config: Config{
			ClaudeBin:   "/opt/claude/bin/claude",
			MorphBin:    "/usr/local/bin/morph",
			ProjectsDir: filepath.Join(h.tmp, "projects"),
			MCPBaseURL:  "http://127.0.0.1:7181",
			Defaults:    queue.Caps{ClaudeUSD: 25, Hours: 2, ExecutorUSD: 4},
			MaxParallel: 1,
			Loop: supervisor.Config{
				MaxResumesPerHour: 3,
				StallMinutes:      30,
				StretchUSD:        30,
				UsageAlertPercent: 50,
			},
		},
	}
}

func (h *dcHarness) open(ctx context.Context) *Daemon {
	h.t.Helper()
	d, err := Open(ctx, h.deps())
	if err != nil {
		h.t.Fatal(err)
	}
	return d
}

func TestDaemonCoreExample1(t *testing.T) {
	h := dcNew(t)
	h.addProject("demo", "go", "https://github.com/acme/demo")
	d := h.open(context.Background())

	views := d.Projects()
	testhelp.Equal(t, "projects", views, []control.ProjectView{{
		Name:      "demo",
		Language:  "go",
		RepoURL:   "https://github.com/acme/demo",
		State:     "idle",
		CreatedAt: time.Date(2026, 10, 8, 14, 0, 0, 0, time.UTC),
	}})

	st, err := d.Status("demo")
	testhelp.Equal(t, "status err", err, nil)
	testhelp.Equal(t, "status", st, control.Status{State: "idle", LimitsUnknown: true})

	jb, err := json.Marshal(st)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(jb), `"five_hour":null,"seven_day":null`) {
		t.Errorf("status json: %s", jb)
	}

	st0, err := d.Status("nope")
	testhelp.Equal(t, "unknown err", errors.Is(err, control.ErrUnknownProject), true)
	testhelp.Equal(t, "unknown status", st0, control.Status{})

	stop, err := d.StopCheck("demo")
	testhelp.Equal(t, "stopcheck err", err, nil)
	testhelp.Equal(t, "stopcheck nil", stop == nil, true)

	usage, err := d.Usage("demo")
	testhelp.Equal(t, "usage err", err, nil)
	testhelp.Equal(t, "usage", usage, control.Usage{})
	ub, err := json.Marshal(usage)
	if err != nil {
		t.Fatal(err)
	}
	testhelp.Equal(t, "usage json", string(ub), `{"five_hour":null,"seven_day":null,"five_hour_resets_at":null,"seven_day_resets_at":null,"session_cost_usd":0,"stretch_cost_usd":0,"limits_at":null}`)

	testhelp.Equal(t, "no spawns", len(h.spawnList()), 0)
}

func TestDaemonCoreExample2(t *testing.T) {
	h := dcNew(t)
	h.addProject("demo", "go", "https://github.com/acme/demo")
	d := h.open(context.Background())
	plan := dcPlan(t)

	view, err := d.PlanLoad("demo", plan)
	testhelp.Equal(t, "planload err", err, nil)
	testhelp.Equal(t, "planload view", view, control.QueueView{
		State:    "running",
		Approved: "7b31dfe",
		Index:    0,
		Phases:   3,
		Current:  "P17",
	})

	launches := h.spawnList()
	testhelp.Equal(t, "launches len", len(launches), 1)
	testhelp.Equal(t, "launch", launches[0], claude.Launch{
		Bin:           "/opt/claude/bin/claude",
		Dir:           filepath.Join(h.tmp, "projects", "demo"),
		SessionID:     "id-1",
		Resume:        false,
		BudgetUSD:     float64(30),
		MCPConfigPath: filepath.Join(h.tmp, "state", "demo", "morphd-mcp.json"),
	})

	mcp, err := os.ReadFile(filepath.Join(h.tmp, "state", "demo", "morphd-mcp.json"))
	if err != nil {
		t.Fatal(err)
	}
	testhelp.Equal(t, "mcp file", mcp, claude.MCPConfigJSON("http://127.0.0.1:7181/mcp/demo/session", "id-2"))

	tok, err := h.reg.ReadSecret("demo", "session-token")
	testhelp.Equal(t, "token err", err, nil)
	testhelp.Equal(t, "token", tok, "id-2")

	fp := h.fake(0)
	testhelp.Equal(t, "written", fp.Written(), [][]byte{stream.User("/morph-orchestrator P17")})

	testhelp.Equal(t, "posts", h.postList(), [][4]string{
		{"demo", "start", "P17 started", "session id-1 · cap $30 · 3 h"},
	})

	st, err := d.Status("demo")
	testhelp.Equal(t, "status err", err, nil)
	testhelp.Equal(t, "status state", st.State, "busy")
	testhelp.Equal(t, "status phase", st.Phase, "P17")
	testhelp.Equal(t, "status session", st.SessionID, "id-1")

	data, err := os.ReadFile(filepath.Join(h.tmp, "state", "demo", "sessions", "id-1.jsonl"))
	if err != nil {
		t.Fatal(err)
	}
	lines := strings.Split(strings.TrimSuffix(string(data), "\n"), "\n")
	testhelp.Equal(t, "log lines", len(lines), 1)
	var entry map[string]any
	if err := json.Unmarshal([]byte(lines[0]), &entry); err != nil {
		t.Fatal(err)
	}
	testhelp.Equal(t, "log dir", entry["dir"], "in")
}

func TestDaemonCoreExample3(t *testing.T) {
	h := dcNew(t)
	h.addProject("demo", "go", "https://github.com/acme/demo")
	d := h.open(context.Background())
	plan := dcPlan(t)

	if _, err := d.PlanLoad("demo", plan); err != nil {
		t.Fatal(err)
	}
	fp := h.fake(0)

	res, err := d.Order("demo", "smoke checked green")
	testhelp.Equal(t, "order err", err, nil)
	testhelp.Equal(t, "order queued", res.Queued, 1)
	testhelp.Equal(t, "order sent", res.Sent, false)

	q, err := d.Pending("demo")
	testhelp.Equal(t, "pending err", err, nil)
	testhelp.Equal(t, "pending nil", q == nil, true)

	err = d.Answer("demo", 1)
	testhelp.Equal(t, "answer err", errors.Is(err, control.ErrNoQuestion), true)

	err = d.Interrupt("demo")
	testhelp.Equal(t, "interrupt err", err, nil)

	w := fp.Written()
	testhelp.Equal(t, "written len", len(w), 2)
	testhelp.Equal(t, "written 1", w[1], stream.Interrupt("int-id-3"))

	ev, err := d.Events("demo", 0, 10)
	testhelp.Equal(t, "events err", err, nil)
	testhelp.Equal(t, "events last", ev.Last, int64(2))
	testhelp.Equal(t, "events len", len(ev.Entries), 2)
	testhelp.Equal(t, "entry 0 seq", ev.Entries[0].Seq, int64(1))
	testhelp.Equal(t, "entry 0 dir", ev.Entries[0].Dir, "in")
	testhelp.Equal(t, "entry 1 seq", ev.Entries[1].Seq, int64(2))
	testhelp.Equal(t, "entry 1 dir", ev.Entries[1].Dir, "in")
}

func TestDaemonCoreExample4(t *testing.T) {
	h := dcNew(t)
	h.addProject("demo", "go", "https://github.com/acme/demo")
	d := h.open(context.Background())
	plan := dcPlan(t)

	if _, err := d.PlanLoad("demo", plan); err != nil {
		t.Fatal(err)
	}
	fp := h.fake(0)

	_, err := d.PlanLoad("demo", plan)
	testhelp.Equal(t, "planload2 busy", errors.Is(err, control.ErrBusy), true)

	err = d.Restart("demo", false)
	testhelp.Equal(t, "restart busy", errors.Is(err, control.ErrBusy), true)

	err = d.Restart("demo", true)
	testhelp.Equal(t, "restart err", err, nil)

	testhelp.Equal(t, "killed", fp.Killed(), true)

	launches := h.spawnList()
	testhelp.Equal(t, "launches len", len(launches), 2)
	testhelp.Equal(t, "launch2 session", launches[1].SessionID, "id-3")
	testhelp.Equal(t, "launch2 resume", launches[1].Resume, false)
	testhelp.Equal(t, "launch2 budget", launches[1].BudgetUSD, float64(30))

	testhelp.Equal(t, "posts", h.postList(), [][4]string{
		{"demo", "start", "P17 started", "session id-1 · cap $30 · 3 h"},
		{"demo", "start", "P17 started", "session id-3 · cap $30 · 3 h"},
	})

	st, err := d.Status("demo")
	testhelp.Equal(t, "status err", err, nil)
	testhelp.Equal(t, "status session", st.SessionID, "id-3")
}

func TestDaemonCoreExample5(t *testing.T) {
	h := dcNew(t)
	h.addProject("demo", "go", "https://github.com/acme/demo")
	d := h.open(context.Background())
	plan := dcPlan(t)

	if _, err := d.PlanLoad("demo", plan); err != nil {
		t.Fatal(err)
	}
	fp := h.fake(0)

	err := d.PhaseDone("demo", "wrong", "P17", "P18")
	testhelp.Equal(t, "phase done bad token", errors.Is(err, control.ErrBadToken), true)

	err = d.PhaseDone("demo", "id-2", "P17", "P18")
	testhelp.Equal(t, "phase done err", err, nil)

	testhelp.Equal(t, "killed", fp.Killed(), true)

	launches := h.spawnList()
	testhelp.Equal(t, "launches len", len(launches), 2)
	testhelp.Equal(t, "launch2 session", launches[1].SessionID, "id-3")
	testhelp.Equal(t, "launch2 resume", launches[1].Resume, false)
	testhelp.Equal(t, "launch2 budget", launches[1].BudgetUSD, float64(12))

	tok, err := h.reg.ReadSecret("demo", "session-token")
	testhelp.Equal(t, "token err", err, nil)
	testhelp.Equal(t, "token", tok, "id-4")

	posts := h.postList()
	testhelp.Equal(t, "last post", posts[len(posts)-1], [4]string{"demo", "start", "P18 started", "session id-3 · cap $12 · 3 h"})

	st, err := d.Status("demo")
	testhelp.Equal(t, "status err", err, nil)
	testhelp.Equal(t, "phase", st.Phase, "P18")
	testhelp.Equal(t, "queue index", st.Queue.Index, 1)
	testhelp.Equal(t, "queue current", st.Queue.Current, "P18")
}

func TestDaemonCoreExample6(t *testing.T) {
	h := dcNew(t)
	h.addProject("demo", "go", "https://github.com/acme/demo")
	d := h.open(context.Background())
	plan := dcPlan(t)

	if _, err := d.PlanLoad("demo", plan); err != nil {
		t.Fatal(err)
	}
	fp := h.fake(0)

	h.run.Script["git rev-parse HEAD"] = runner.Result{Stdout: "bbb222\n"}

	err := d.PhaseDone("demo", "id-2", "P17", "P18")
	testhelp.Equal(t, "phase done err", err, nil)

	h.run.Script["git rev-parse HEAD"] = runner.Result{Stdout: "aaa111\n"}

	st, err := d.Status("demo")
	testhelp.Equal(t, "status err", err, nil)
	testhelp.Equal(t, "state", st.State, "waiting")

	testhelp.Equal(t, "killed", fp.Killed(), false)

	posts := h.postList()
	testhelp.Equal(t, "last post", posts[len(posts)-1], [4]string{"demo", "stop", "P17 not pushed", "not pushed: HEAD bbb222, origin/main aaa111"})

	stop, err := d.StopCheck("demo")
	testhelp.Equal(t, "stopcheck err", err, nil)
	if stop == nil {
		t.Fatalf("stop nil")
	}
	cp := *stop
	if cp.At.IsZero() {
		t.Fatalf("stop at zero")
	}
	cp.At = time.Time{}
	testhelp.Equal(t, "stop", cp, control.Stop{Kind: "not_pushed", Phase: "P17", Reason: "not pushed: HEAD bbb222, origin/main aaa111"})

	err = d.Continue("demo")
	testhelp.Equal(t, "continue err", err, nil)

	testhelp.Equal(t, "killed after continue", fp.Killed(), true)

	launches := h.spawnList()
	testhelp.Equal(t, "launches len", len(launches), 2)
	testhelp.Equal(t, "launch2 session", launches[1].SessionID, "id-3")
	testhelp.Equal(t, "launch2 budget", launches[1].BudgetUSD, float64(12))

	st, err = d.Status("demo")
	testhelp.Equal(t, "status2 err", err, nil)
	testhelp.Equal(t, "phase2", st.Phase, "P18")
}

func TestDaemonCoreExample7(t *testing.T) {
	h := dcNew(t)
	h.addProject("demo", "go", "https://github.com/acme/demo")
	d := h.open(context.Background())
	plan := dcPlan(t)

	if _, err := d.PlanLoad("demo", plan); err != nil {
		t.Fatal(err)
	}
	fp := h.fake(0)

	err := d.WaitOperator("demo", "id-2", "smoke", "P17 smoke: run morphd status", "")
	testhelp.Equal(t, "waitop err", err, nil)

	err = d.Milestone("demo", "id-2", control.Milestone{Kind: "gate", Headline: "P17 gate passed", Numbers: "10 cards · $0.12"})
	testhelp.Equal(t, "milestone err", err, nil)

	err = d.Milestone("demo", "bad", control.Milestone{Kind: "gate", Headline: "x", Numbers: "y"})
	testhelp.Equal(t, "milestone bad err", errors.Is(err, control.ErrBadToken), true)

	posts := h.postList()
	testhelp.Equal(t, "posts", posts[1:], [][4]string{
		{"demo", "stop", "P17: wait_operator smoke", "P17 smoke: run morphd status"},
		{"demo", "gate", "P17 gate passed", "10 cards · $0.12"},
	})

	st, err := d.Status("demo")
	testhelp.Equal(t, "status err", err, nil)
	testhelp.Equal(t, "state", st.State, "waiting")
	testhelp.Equal(t, "killed", fp.Killed(), false)

	err = d.Continue("demo")
	testhelp.Equal(t, "continue err", err, nil)

	testhelp.Equal(t, "killed after continue", fp.Killed(), true)

	launches := h.spawnList()
	testhelp.Equal(t, "launches len", len(launches), 2)
	testhelp.Equal(t, "launch2 session", launches[1].SessionID, "id-3")

	st, err = d.Status("demo")
	testhelp.Equal(t, "status2 err", err, nil)
	testhelp.Equal(t, "state2", st.State, "busy")
}

func TestDaemonCoreExample8(t *testing.T) {
	h := dcNew(t)
	h.run.Script["git rev-parse --verify -q origin/main"] = runner.Result{Code: 1}
	h.run.Script["git push -q -u origin main"] = runner.Result{Code: 0}
	h.addProject("demo", "go", "https://github.com/acme/demo")
	d := h.open(context.Background())
	ctx := context.Background()

	h.do = dcHTTP(200, `{"permissions":{"push":true}}`)

	res, err := d.PutGithubToken(ctx, "demo", "ghp_abc")
	testhelp.Equal(t, "put err", err, nil)
	acc := res.Access
	if acc.CheckedAt.IsZero() {
		t.Fatalf("checked at zero")
	}
	acc.CheckedAt = time.Time{}
	testhelp.Equal(t, "access", acc, github.Access{Checked: true, Reachable: true, Push: true})
	testhelp.Equal(t, "pushed", res.Pushed, true)
	testhelp.Equal(t, "push error", res.PushError, "")

	tok, err := h.reg.ReadSecret("demo", "github-token")
	testhelp.Equal(t, "token err", err, nil)
	testhelp.Equal(t, "token", tok, "ghp_abc")
	creds, err := h.reg.ReadSecret("demo", "git-credentials")
	testhelp.Equal(t, "creds err", err, nil)
	testhelp.Equal(t, "creds", creds, "https://x-access-token:ghp_abc@github.com\n")

	for _, kind := range []string{"github-token", "git-credentials"} {
		fi, err := os.Stat(h.reg.SecretPath("demo", kind))
		testhelp.Equal(t, "stat "+kind, err == nil, true)
		testhelp.Equal(t, "mode "+kind, fi.Mode().Perm(), os.FileMode(0o600))
	}

	repo, err := d.Status("demo")
	testhelp.Equal(t, "status err", err, nil)
	racc := repo.Repo
	if racc.CheckedAt.IsZero() {
		t.Fatalf("repo checked at zero")
	}
	racc.CheckedAt = time.Time{}
	testhelp.Equal(t, "status repo", racc, github.Access{Checked: true, Reachable: true, Push: true})

	testhelp.Equal(t, "runner calls", h.run.Calls, []runner.Call{
		{Dir: filepath.Join(h.tmp, "projects", "demo"), Name: "git", Args: []string{"rev-parse", "--verify", "-q", "origin/main"}},
		{Dir: filepath.Join(h.tmp, "projects", "demo"), Name: "git", Args: []string{"push", "-q", "-u", "origin", "main"}},
	})

	h.do = dcHTTP(401, `{"message":"Bad credentials"}`)

	res2, err := d.PutGithubToken(ctx, "demo", "ghp_abc")
	testhelp.Equal(t, "put2 err", err, nil)
	acc2 := res2.Access
	if acc2.CheckedAt.IsZero() {
		t.Fatalf("checked at zero 2")
	}
	acc2.CheckedAt = time.Time{}
	testhelp.Equal(t, "access2", acc2, github.Access{Checked: true, Reason: "401 bad credentials"})
	testhelp.Equal(t, "pushed2", res2.Pushed, false)
	testhelp.Equal(t, "runner calls2", len(h.run.Calls), 2)

	res3, err := d.PutGithubToken(ctx, "demo", "")
	testhelp.Equal(t, "put3 err", errors.Is(err, control.ErrBadInput), true)
	testhelp.Equal(t, "put3 res", res3, control.TokenResult{})

	res4, err := d.PutGithubToken(ctx, "nope", "x")
	testhelp.Equal(t, "put4 err", errors.Is(err, control.ErrUnknownProject), true)
	testhelp.Equal(t, "put4 res", res4, control.TokenResult{})
}

func TestDaemonCoreExample9(t *testing.T) {
	h := dcNew(t)
	h.addProject("demo", "go", "https://github.com/acme/demo")
	h.addProject("beta", "go", "https://github.com/acme/beta")
	d := h.open(context.Background())
	plan := dcPlan(t)

	v1, err := d.PlanLoad("demo", plan)
	testhelp.Equal(t, "demo plan err", err, nil)
	testhelp.Equal(t, "demo plan state", v1.State, "running")

	v2, err := d.PlanLoad("beta", plan)
	testhelp.Equal(t, "beta plan err", err, nil)
	testhelp.Equal(t, "beta view", v2, control.QueueView{
		State:    "queued",
		Approved: "7b31dfe",
		Index:    0,
		Phases:   3,
		Current:  "P17",
	})

	testhelp.Equal(t, "spawns len", len(h.spawnList()), 1)

	st, err := d.Status("beta")
	testhelp.Equal(t, "beta status err", err, nil)
	testhelp.Equal(t, "beta state", st.State, "idle")
	testhelp.Equal(t, "beta queue state", st.Queue.State, "queued")

	err = d.WaitOperator("demo", "id-2", "smoke", "stop", "")
	testhelp.Equal(t, "waitop err", err, nil)

	d.Tick(context.Background())

	spawns := h.spawnList()
	testhelp.Equal(t, "spawns len2", len(spawns), 2)
	testhelp.Equal(t, "spawn dir", spawns[1].Dir, filepath.Join(h.tmp, "projects", "beta"))
	testhelp.Equal(t, "spawn sid", spawns[1].SessionID, "id-3")

	st, err = d.Status("beta")
	testhelp.Equal(t, "beta status2 err", err, nil)
	testhelp.Equal(t, "beta state2", st.State, "busy")
	testhelp.Equal(t, "beta phase2", st.Phase, "P17")
}

func TestDaemonCoreExample10(t *testing.T) {
	h := dcNew(t)
	h.addProject("demo", "go", "https://github.com/acme/demo")
	d := h.open(context.Background())
	ctx := context.Background()

	_, err := d.CreateProject(ctx, "demo", "go", "https://github.com/acme/demo")
	testhelp.Equal(t, "dup err", errors.Is(err, control.ErrExists), true)
	testhelp.Equal(t, "no runner calls", len(h.run.Calls), 0)

	pv, err := d.CreateProject(ctx, "gamma", "python", "https://github.com/acme/gamma")
	testhelp.Equal(t, "gamma err", err, nil)
	if pv.CreatedAt.IsZero() {
		t.Fatalf("gamma created at zero")
	}
	pv2 := pv
	pv2.CreatedAt = time.Time{}
	testhelp.Equal(t, "gamma view", pv2, control.ProjectView{
		Name:     "gamma",
		Language: "python",
		RepoURL:  "https://github.com/acme/gamma",
		State:    "idle",
	})

	got, err := h.reg.Get("gamma")
	testhelp.Equal(t, "get err", err, nil)
	testhelp.Equal(t, "get dir", got.Dir, filepath.Join(h.tmp, "projects", "gamma"))
	testhelp.Equal(t, "get caps", got.Caps, queue.Caps{ClaudeUSD: 25, Hours: 2, ExecutorUSD: 4})

	testhelp.Equal(t, "call0", h.run.Calls[0], runner.Call{
		Dir:  filepath.Join(h.tmp, "projects"),
		Name: "/usr/local/bin/morph",
		Args: []string{"init", "--root", filepath.Join(h.tmp, "projects", "gamma"), "--name", "gamma", "--language", "python"},
	})

	views := d.Projects()
	var names, states []string
	for _, v := range views {
		names = append(names, v.Name)
		states = append(states, v.State)
	}
	testhelp.Equal(t, "names", names, []string{"demo", "gamma"})
	testhelp.Equal(t, "states", states, []string{"idle", "idle"})

	_, err = d.CreateProject(ctx, "delta", "cobol", "https://github.com/acme/delta")
	testhelp.Equal(t, "delta err", errors.Is(err, control.ErrBadInput), true)
	testhelp.Equal(t, "delta text", err.Error(), `bad input: bootstrap: unknown language "cobol"`)
}

func TestDaemonCoreExample11(t *testing.T) {
	h := dcNew(t)
	h.spawnErr = errors.New("claude: start fork/exec /opt/claude/bin/claude: no such file or directory")
	h.addProject("demo", "go", "https://github.com/acme/demo")
	d := h.open(context.Background())
	plan := dcPlan(t)

	view, err := d.PlanLoad("demo", plan)
	testhelp.Equal(t, "planload err", err, nil)
	testhelp.Equal(t, "planload view", view, control.QueueView{
		State:    "stopped",
		Approved: "7b31dfe",
		Index:    0,
		Phases:   3,
		Current:  "P17",
		Reason:   "crash",
	})

	launches := h.spawnList()
	testhelp.Equal(t, "launches len", len(launches), 4)
	var sids []string
	for _, l := range launches {
		sids = append(sids, l.SessionID)
	}
	testhelp.Equal(t, "session ids", sids, []string{"id-1", "id-3", "id-5", "id-7"})
	for i, l := range launches {
		testhelp.Equal(t, fmt.Sprintf("resume %d", i), l.Resume, false)
	}

	tok, err := h.reg.ReadSecret("demo", "session-token")
	testhelp.Equal(t, "token err", err, nil)
	testhelp.Equal(t, "token", tok, "id-8")

	testhelp.Equal(t, "posts", h.postList(), [][4]string{
		{"demo", "watchdog", "P17: session exited (code -3), restarting", "restart 1 of 3 this hour · claude: start fork/exec /opt/claude/bin/claude: no such file or directory"},
		{"demo", "watchdog", "P17: session exited (code -3), restarting", "restart 2 of 3 this hour · claude: start fork/exec /opt/claude/bin/claude: no such file or directory"},
		{"demo", "watchdog", "P17: session exited (code -3), restarting", "restart 3 of 3 this hour · claude: start fork/exec /opt/claude/bin/claude: no such file or directory"},
		{"demo", "stop", "P17: session keeps exiting", "exited 4 times within an hour (last code -3): claude: start fork/exec /opt/claude/bin/claude: no such file or directory"},
	})

	st, err := d.Status("demo")
	testhelp.Equal(t, "status err", err, nil)
	testhelp.Equal(t, "state", st.State, "waiting")
	testhelp.Equal(t, "phase", st.Phase, "P17")
	testhelp.Equal(t, "session id", st.SessionID, "id-7")
	testhelp.Equal(t, "limits unknown", st.LimitsUnknown, true)
	if st.Stop == nil {
		t.Fatalf("stop nil")
	}
	testhelp.Equal(t, "stop kind", st.Stop.Kind, "crash")
	testhelp.Equal(t, "stop reason", st.Stop.Reason, "exited 4 times within an hour (last code -3): claude: start fork/exec /opt/claude/bin/claude: no such file or directory")

	if st.LastExit == nil {
		t.Fatalf("last exit nil")
	}
	le := *st.LastExit
	if le.At.IsZero() {
		t.Fatalf("last exit at zero")
	}
	le.At = time.Time{}
	testhelp.Equal(t, "last exit", le, control.Exit{
		Code:     -3,
		Stderr:   "claude: start fork/exec /opt/claude/bin/claude: no such file or directory",
		Restarts: 3,
	})

	stop, err := d.StopCheck("demo")
	testhelp.Equal(t, "stopcheck err", err, nil)
	if stop == nil {
		t.Fatalf("stopcheck nil")
	}
	sc := *stop
	if sc.At.IsZero() {
		t.Fatalf("stop at zero")
	}
	sc.At = time.Time{}
	testhelp.Equal(t, "stopcheck", sc, control.Stop{
		Kind:   "crash",
		Phase:  "P17",
		Reason: "exited 4 times within an hour (last code -3): claude: start fork/exec /opt/claude/bin/claude: no such file or directory",
	})

	ev, err := d.Events("demo", 0, 10)
	testhelp.Equal(t, "events err", err, nil)
	testhelp.Equal(t, "events len", len(ev.Entries), 0)
	testhelp.Equal(t, "events last", ev.Last, int64(0))

	data, err := os.ReadFile(filepath.Join(h.tmp, "state", "demo", "state.json"))
	if err != nil {
		t.Fatal(err)
	}
	var s State
	if err := json.Unmarshal(data, &s); err != nil {
		t.Fatal(err)
	}
	testhelp.Equal(t, "state loop state", s.Loop.State, "waiting")
	testhelp.Equal(t, "state restarts len", len(s.Loop.Restarts), 3)

	err = d.Continue("demo")
	testhelp.Equal(t, "continue err", err, nil)

	launches = h.spawnList()
	testhelp.Equal(t, "launches len2", len(launches), 8)
	var sids2 []string
	for _, l := range launches[4:] {
		sids2 = append(sids2, l.SessionID)
	}
	testhelp.Equal(t, "session ids2", sids2, []string{"id-9", "id-11", "id-13", "id-15"})

	st, err = d.Status("demo")
	testhelp.Equal(t, "status2 err", err, nil)
	testhelp.Equal(t, "state2", st.State, "waiting")
	if st.Stop == nil {
		t.Fatalf("stop2 nil")
	}
	testhelp.Equal(t, "stop2 kind", st.Stop.Kind, "crash")

	testhelp.Equal(t, "posts total", len(h.postList()), 8)
}
