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

// pDCH is the harness of the Daemon Core examples: every fake the record names, guarded by one mutex.
type pDCH struct {
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
	do       func(*http.Request) (*http.Response, error)
	d        *Daemon
}

func pDCAdd(t *testing.T, reg *registry.Registry, tmp, name string) {
	t.Helper()
	dir := filepath.Join(tmp, "projects", name)
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
	if err := reg.Add(registry.Project{Name: name, Language: "go", RepoURL: "https://github.com/acme/" + name, Dir: dir,
		Caps: queue.Caps{ClaudeUSD: 30, Hours: 3, ExecutorUSD: 5}, CreatedAt: time.Date(2026, 10, 8, 14, 0, 0, 0, time.UTC)}); err != nil {
		t.Fatal(err)
	}
}

func pDCNew(t *testing.T, names ...string) *pDCH {
	t.Helper()
	h := &pDCH{t: t, tmp: t.TempDir(), clock: time.Date(2026, 10, 8, 15, 0, 0, 0, time.UTC)}
	h.state = filepath.Join(h.tmp, "state")
	reg, err := registry.Open(h.state)
	if err != nil {
		t.Fatal(err)
	}
	h.reg = reg
	for _, n := range names {
		pDCAdd(t, reg, h.tmp, n)
	}
	h.run = &runner.Fake{Script: map[string]runner.Result{
		"git rev-parse --abbrev-ref HEAD": {Stdout: "main\n"},
		"git rev-parse HEAD":              {Stdout: "aaa111\n"},
		"git rev-parse origin/main":       {Stdout: "aaa111\n"},
		"git status --porcelain":          {Stdout: ""},
	}}
	return h
}

func (h *pDCH) deps() Deps {
	return Deps{
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
		GitHubDo: func(r *http.Request) (*http.Response, error) {
			h.mu.Lock()
			do := h.do
			h.mu.Unlock()
			if do == nil {
				return nil, errors.New("no network in tests")
			}
			return do(r)
		},
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
}

func (h *pDCH) open() *Daemon {
	h.t.Helper()
	d, err := Open(context.Background(), h.deps())
	if err != nil {
		h.t.Fatalf("Open: %v", err)
	}
	if d == nil {
		h.t.Fatalf("Open returned a nil daemon")
	}
	h.d = d
	return d
}

func (h *pDCH) spawns() ([]claude.Launch, []*claude.Fake) {
	h.mu.Lock()
	defer h.mu.Unlock()
	return append([]claude.Launch(nil), h.launches...), append([]*claude.Fake(nil), h.procs...)
}

func (h *pDCH) postList() [][4]string {
	h.mu.Lock()
	defer h.mu.Unlock()
	return append([][4]string(nil), h.posts...)
}

func (h *pDCH) setScript(key string, r runner.Result) {
	h.run.Script[key] = r
}

func pDCPlan(t *testing.T) queue.Plan {
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

func pDCErr(err error) string {
	if err == nil {
		return "<nil>"
	}
	return err.Error()
}

// pDCRunning builds example 2's state: demo with P17 running.
func pDCRunning(t *testing.T) (*pDCH, *Daemon) {
	h := pDCNew(t, "demo")
	d := h.open()
	v, err := d.PlanLoad("demo", pDCPlan(t))
	if err != nil || v.State != "running" {
		t.Fatalf("setup: PlanLoad(demo) = %+v, %v; want State running", v, err)
	}
	if _, procs := h.spawns(); len(procs) != 1 {
		t.Fatalf("setup: PlanLoad(demo) spawned %d processes, want 1", len(procs))
	}
	return h, d
}

func pDCStatus(t *testing.T, d *Daemon, project string) control.Status {
	t.Helper()
	s, err := d.Status(project)
	if err != nil {
		t.Errorf("Status(%q) error %v", project, err)
	}
	return s
}

func pDCLines(t *testing.T, path string) []string {
	b, err := os.ReadFile(path)
	if err != nil {
		return []string{"read error: " + err.Error()}
	}
	return strings.Split(strings.TrimRight(string(b), "\n"), "\n")
}

func TestProbeDaemonCoreExample1(t *testing.T) {
	h := pDCNew(t, "demo")
	d := h.open()
	testhelp.Equal(t, "example 1 Projects()", d.Projects(), []control.ProjectView{{Name: "demo", Language: "go",
		RepoURL: "https://github.com/acme/demo", State: "idle", CreatedAt: time.Date(2026, 10, 8, 14, 0, 0, 0, time.UTC)}})
	s, err := d.Status("demo")
	testhelp.Equal(t, "example 1 Status(demo)", []any{s, pDCErr(err)}, []any{control.Status{State: "idle"}, "<nil>"})
	s, err = d.Status("nope")
	testhelp.Equal(t, "example 1 Status(nope)", []any{s, errors.Is(err, control.ErrUnknownProject)}, []any{control.Status{}, true})
	st, err := d.StopCheck("demo")
	testhelp.Equal(t, "example 1 StopCheck(demo)", []any{st == nil, pDCErr(err)}, []any{true, "<nil>"})
	_, procs := h.spawns()
	testhelp.Equal(t, "example 1 no spawn at Open", len(procs), 0)
	// variant: an empty registry lists [] (not nil)
	h2 := pDCNew(t)
	testhelp.Equal(t, "example 1 variant Projects() of an empty registry", h2.open().Projects(), []control.ProjectView{})
}

func TestProbeDaemonCoreExample2(t *testing.T) {
	h := pDCNew(t, "demo")
	d := h.open()
	v, err := d.PlanLoad("demo", pDCPlan(t))
	testhelp.Equal(t, "example 2 PlanLoad(demo)", []any{v, pDCErr(err)},
		[]any{control.QueueView{State: "running", Approved: "7b31dfe", Index: 0, Phases: 3, Current: "P17"}, "<nil>"})
	launches, procs := h.spawns()
	mcp := filepath.Join(h.state, "demo", "morphd-mcp.json")
	testhelp.Equal(t, "example 2 the Spawn launches", launches, []claude.Launch{{Bin: "/opt/claude/bin/claude",
		Dir: filepath.Join(h.tmp, "projects", "demo"), SessionID: "id-1", Resume: false, BudgetUSD: 30, MCPConfigPath: mcp}})
	b, err := os.ReadFile(mcp)
	testhelp.Equal(t, "example 2 the MCP config file", string(b)+pDCErr(err),
		string(claude.MCPConfigJSON("http://127.0.0.1:7181/mcp/demo/session", "id-2"))+"<nil>")
	tok, err := h.reg.ReadSecret("demo", "session-token")
	testhelp.Equal(t, "example 2 the secret session-token", tok+" "+pDCErr(err), "id-2 <nil>")
	if len(procs) == 1 {
		testhelp.Equal(t, "example 2 fp.Written()", procs[0].Written(), [][]byte{stream.User("/morph-orchestrator P17")})
	}
	testhelp.Equal(t, "example 2 the posts", h.postList(), [][4]string{{"demo", "start", "P17 started", "session id-1 · cap $30 · 3 h"}})
	s := pDCStatus(t, d, "demo")
	testhelp.Equal(t, "example 2 Status State, Phase, SessionID", []string{s.State, s.Phase, s.SessionID}, []string{"busy", "P17", "id-1"})
	lines := pDCLines(t, filepath.Join(h.state, "demo", "sessions", "id-1.jsonl"))
	dir := ""
	if len(lines) == 1 {
		var e struct {
			Dir string `json:"dir"`
		}
		_ = json.Unmarshal([]byte(lines[0]), &e)
		dir = e.Dir
	}
	testhelp.Equal(t, "example 2 the session log lines and the first dir", []any{len(lines), dir}, []any{1, "in"})
	// variant: a project the registry does not hold
	_, err = d.PlanLoad("nope", pDCPlan(t))
	testhelp.Equal(t, "example 2 variant PlanLoad(nope) is ErrUnknownProject", errors.Is(err, control.ErrUnknownProject), true)
	// variant: a plan queue.Load refuses is ErrBadInput wrapping the queue's text
	h2 := pDCNew(t, "demo")
	_, err = h2.open().PlanLoad("demo", queue.Plan{Approved: "abc", Phases: nil})
	testhelp.Equal(t, "example 2 variant PlanLoad of a plan without phases", []any{errors.Is(err, control.ErrBadInput), pDCErr(err)},
		[]any{true, "bad input: queue: no phases"})
}

func TestProbeDaemonCoreExample3(t *testing.T) {
	h, d := pDCRunning(t)
	r, err := d.Order("demo", "smoke checked green")
	testhelp.Equal(t, "example 3 Order while the first line is in flight", []any{r.Queued, r.Sent, pDCErr(err)}, []any{1, false, "<nil>"})
	q, err := d.Pending("demo")
	testhelp.Equal(t, "example 3 Pending", []any{q == nil, pDCErr(err)}, []any{true, "<nil>"})
	err = d.Answer("demo", 1)
	testhelp.Equal(t, "example 3 Answer without a question is ErrNoQuestion", errors.Is(err, control.ErrNoQuestion), true)
	err = d.Interrupt("demo")
	testhelp.Equal(t, "example 3 Interrupt", pDCErr(err), "<nil>")
	_, procs := h.spawns()
	w := procs[0].Written()
	if testhelp.Equal(t, "example 3 the number of lines written", len(w), 2) {
		testhelp.Equal(t, "example 3 Written()[1]", string(w[1]), string(stream.Interrupt("int-id-3")))
	}
	ev, err := d.Events("demo", 0, 10)
	dirs := []string{}
	for _, e := range ev.Entries {
		dirs = append(dirs, fmt.Sprintf("%d %s", e.Seq, e.Dir))
	}
	testhelp.Equal(t, "example 3 Events", []any{dirs, ev.Last, pDCErr(err)}, []any{[]string{"1 in", "2 in"}, int64(2), "<nil>"})
	// variant: an idle project has no session
	h2 := pDCNew(t, "demo")
	d2 := h2.open()
	_, err = d2.Order("demo", "x")
	testhelp.Equal(t, "example 3 variant Order on an idle project is ErrNoSession", errors.Is(err, control.ErrNoSession), true)
	ev, err = d2.Events("demo", 0, 10)
	testhelp.Equal(t, "example 3 variant Events of a project without a log", []any{ev.Entries != nil, len(ev.Entries), ev.Last, pDCErr(err)},
		[]any{true, 0, int64(0), "<nil>"})
}

func TestProbeDaemonCoreExample4(t *testing.T) {
	h, d := pDCRunning(t)
	_, err := d.PlanLoad("demo", pDCPlan(t))
	testhelp.Equal(t, "example 4 PlanLoad while running is ErrBusy", errors.Is(err, control.ErrBusy), true)
	err = d.Restart("demo", false)
	testhelp.Equal(t, "example 4 Restart(false) on a busy machine is ErrBusy", errors.Is(err, control.ErrBusy), true)
	err = d.Restart("demo", true)
	testhelp.Equal(t, "example 4 Restart(true)", pDCErr(err), "<nil>")
	launches, procs := h.spawns()
	if testhelp.Equal(t, "example 4 the number of spawns", len(launches), 2) {
		testhelp.Equal(t, "example 4 the first process Killed()", procs[0].Killed(), true)
		testhelp.Equal(t, "example 4 the second launch SessionID, Resume, BudgetUSD", []any{launches[1].SessionID, launches[1].Resume, launches[1].BudgetUSD},
			[]any{"id-3", false, 30.0})
	}
	testhelp.Equal(t, "example 4 the posts", h.postList(), [][4]string{
		{"demo", "start", "P17 started", "session id-1 · cap $30 · 3 h"},
		{"demo", "start", "P17 started", "session id-3 · cap $30 · 3 h"}})
	testhelp.Equal(t, "example 4 Status.SessionID", pDCStatus(t, d, "demo").SessionID, "id-3")
}

func TestProbeDaemonCoreExample5(t *testing.T) {
	h, d := pDCRunning(t)
	err := d.PhaseDone("demo", "wrong", "P17", "P18")
	testhelp.Equal(t, "example 5 PhaseDone with a wrong token is ErrBadToken", errors.Is(err, control.ErrBadToken), true)
	err = d.PhaseDone("demo", "id-2", "P17", "P18")
	testhelp.Equal(t, "example 5 PhaseDone(id-2)", pDCErr(err), "<nil>")
	launches, procs := h.spawns()
	if testhelp.Equal(t, "example 5 the number of spawns", len(launches), 2) {
		testhelp.Equal(t, "example 5 the first process Killed()", procs[0].Killed(), true)
		testhelp.Equal(t, "example 5 the P18 launch SessionID, Resume, BudgetUSD", []any{launches[1].SessionID, launches[1].Resume, launches[1].BudgetUSD},
			[]any{"id-3", false, 12.0})
	}
	tok, _ := h.reg.ReadSecret("demo", "session-token")
	testhelp.Equal(t, "example 5 the new session token", tok, "id-4")
	posts := h.postList()
	testhelp.Equal(t, "example 5 the last post", posts[len(posts)-1], [4]string{"demo", "start", "P18 started", "session id-3 · cap $12 · 3 h"})
	s := pDCStatus(t, d, "demo")
	testhelp.Equal(t, "example 5 Status Phase, Queue.Index, Queue.Current", []any{s.Phase, s.Queue.Index, s.Queue.Current}, []any{"P18", 1, "P18"})
	// variant: the old token no longer reports
	err = d.PhaseDone("demo", "id-2", "P18", "P19")
	testhelp.Equal(t, "example 5 variant the old token after the spawn is ErrBadToken", errors.Is(err, control.ErrBadToken), true)
	// variant: an unknown project
	err = d.PhaseDone("nope", "id-4", "P18", "P19")
	testhelp.Equal(t, "example 5 variant PhaseDone(nope) is not nil", err != nil, true)
}

func TestProbeDaemonCoreExample6(t *testing.T) {
	h, d := pDCRunning(t)
	h.setScript("git rev-parse HEAD", runner.Result{Stdout: "bbb222\n"})
	err := d.PhaseDone("demo", "id-2", "P17", "P18")
	testhelp.Equal(t, "example 6 PhaseDone(id-2) not pushed", pDCErr(err), "<nil>")
	launches, procs := h.spawns()
	testhelp.Equal(t, "example 6 no second spawn, the process not killed", []any{len(launches), procs[0].Killed()}, []any{1, false})
	posts := h.postList()
	testhelp.Equal(t, "example 6 the last post", posts[len(posts)-1], [4]string{"demo", "stop", "P17 not pushed", "not pushed: HEAD bbb222, origin/main aaa111"})
	testhelp.Equal(t, "example 6 Status.State", pDCStatus(t, d, "demo").State, "waiting")
	st, err := d.StopCheck("demo")
	got := []string{pDCErr(err)}
	if st != nil {
		got = append(got, st.Kind, st.Phase, st.Reason)
	}
	testhelp.Equal(t, "example 6 StopCheck", got, []string{"<nil>", "not_pushed", "P17", "not pushed: HEAD bbb222, origin/main aaa111"})
	h.setScript("git rev-parse HEAD", runner.Result{Stdout: "aaa111\n"})
	err = d.Continue("demo")
	testhelp.Equal(t, "example 6 Continue", pDCErr(err), "<nil>")
	launches, procs = h.spawns()
	if testhelp.Equal(t, "example 6 the number of spawns after Continue", len(launches), 2) {
		testhelp.Equal(t, "example 6 the process killed, P18 spawned", []any{procs[0].Killed(), launches[1].SessionID, launches[1].BudgetUSD},
			[]any{true, "id-3", 12.0})
	}
	testhelp.Equal(t, "example 6 Status.Phase", pDCStatus(t, d, "demo").Phase, "P18")
	// variant: Continue on a running loop is ErrNotWaiting
	err = d.Continue("demo")
	testhelp.Equal(t, "example 6 variant Continue while running is ErrNotWaiting", errors.Is(err, control.ErrNotWaiting), true)
}

func TestProbeDaemonCoreExample7(t *testing.T) {
	h, d := pDCRunning(t)
	err := d.WaitOperator("demo", "id-2", "smoke", "P17 smoke: run morphd status", "")
	testhelp.Equal(t, "example 7 WaitOperator", pDCErr(err), "<nil>")
	err2 := d.Milestone("demo", "id-2", control.Milestone{Kind: "gate", Headline: "P17 gate passed", Numbers: "10 cards · $0.12"})
	testhelp.Equal(t, "example 7 Milestone", pDCErr(err2), "<nil>")
	err3 := d.Milestone("demo", "bad", control.Milestone{Kind: "gate", Headline: "x", Numbers: "y"})
	testhelp.Equal(t, "example 7 Milestone with a bad token is ErrBadToken", errors.Is(err3, control.ErrBadToken), true)
	testhelp.Equal(t, "example 7 the posts after the start", h.postList()[1:], [][4]string{
		{"demo", "stop", "P17: wait_operator smoke", "P17 smoke: run morphd status"},
		{"demo", "gate", "P17 gate passed", "10 cards · $0.12"}})
	testhelp.Equal(t, "example 7 Status.State", pDCStatus(t, d, "demo").State, "waiting")
	_, procs := h.spawns()
	testhelp.Equal(t, "example 7 a wait keeps the process", procs[0].Killed(), false)
	err = d.Continue("demo")
	testhelp.Equal(t, "example 7 Continue", pDCErr(err), "<nil>")
	launches, procs := h.spawns()
	if testhelp.Equal(t, "example 7 the number of spawns after Continue", len(launches), 2) {
		testhelp.Equal(t, "example 7 the held process killed, P17 spawned again", []any{procs[0].Killed(), launches[1].SessionID}, []any{true, "id-3"})
	}
	testhelp.Equal(t, "example 7 Status.State after Continue", pDCStatus(t, d, "demo").State, "busy")
	// variant: WaitOperator with a bad token posts nothing
	n := len(h.postList())
	err = d.WaitOperator("demo", "id-2", "smoke", "x", "")
	testhelp.Equal(t, "example 7 variant WaitOperator(id-2) after the respawn is ErrBadToken", errors.Is(err, control.ErrBadToken), true)
	testhelp.Equal(t, "example 7 variant no further post", len(h.postList()), n)
}

func pDCResp(code int, body string) func(*http.Request) (*http.Response, error) {
	return func(r *http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: code, Body: io.NopCloser(strings.NewReader(body)), Header: http.Header{}}, nil
	}
}

func TestProbeDaemonCoreExample8(t *testing.T) {
	h := pDCNew(t, "demo")
	h.setScript("git rev-parse --verify -q origin/main", runner.Result{Code: 1})
	h.setScript("git push -q -u origin main", runner.Result{Code: 0})
	h.do = pDCResp(200, `{"permissions":{"push":true}}`)
	d := h.open()
	res, err := d.PutGithubToken(context.Background(), "demo", "ghp_abc")
	at := res.Access.CheckedAt
	res.Access.CheckedAt = time.Time{}
	testhelp.Equal(t, "example 8 PutGithubToken 200", []any{res, pDCErr(err), at.IsZero()},
		[]any{control.TokenResult{Access: github.Access{Checked: true, Reachable: true, Push: true}, Pushed: true}, "<nil>", false})
	for _, s := range [][2]string{{"github-token", "ghp_abc"}, {"git-credentials", "https://x-access-token:ghp_abc@github.com\n"}} {
		v, err := h.reg.ReadSecret("demo", s[0])
		mode := ""
		if fi, e := os.Stat(h.reg.SecretPath("demo", s[0])); e == nil {
			mode = fi.Mode().Perm().String()
		}
		testhelp.Equal(t, "example 8 the secret "+s[0], []string{v, pDCErr(err), mode}, []string{s[1], "<nil>", "-rw-------"})
	}
	repo := pDCStatus(t, d, "demo").Repo
	testhelp.Equal(t, "example 8 Status.Repo", []any{repo.Checked, repo.Reachable, repo.Push, repo.Reason, repo.CheckedAt.Equal(at)}, []any{true, true, true, "", true})
	dir := filepath.Join(h.tmp, "projects", "demo")
	testhelp.Equal(t, "example 8 the runner calls", h.run.Calls, []runner.Call{
		{Dir: dir, Name: "git", Args: []string{"rev-parse", "--verify", "-q", "origin/main"}},
		{Dir: dir, Name: "git", Args: []string{"push", "-q", "-u", "origin", "main"}}})
	h.mu.Lock()
	h.do = pDCResp(401, `{"message":"Bad credentials"}`)
	h.mu.Unlock()
	res, err = d.PutGithubToken(context.Background(), "demo", "ghp_abc")
	at = res.Access.CheckedAt
	res.Access.CheckedAt = time.Time{}
	testhelp.Equal(t, "example 8 PutGithubToken 401", []any{res, pDCErr(err), at.IsZero(), len(h.run.Calls)},
		[]any{control.TokenResult{Access: github.Access{Checked: true, Reason: "401 bad credentials"}}, "<nil>", false, 2})
	_, err = d.PutGithubToken(context.Background(), "demo", "")
	testhelp.Equal(t, "example 8 an empty token is ErrBadInput", errors.Is(err, control.ErrBadInput), true)
	_, err = d.PutGithubToken(context.Background(), "nope", "x")
	testhelp.Equal(t, "example 8 PutGithubToken(nope) is ErrUnknownProject", errors.Is(err, control.ErrUnknownProject), true)
	// variant: a failed push is reported, never an error
	h.mu.Lock()
	h.do = pDCResp(200, `{"permissions":{"push":true}}`)
	h.mu.Unlock()
	h.setScript("git push -q -u origin main", runner.Result{Code: 1, Stderr: "rejected"})
	res, err = d.PutGithubToken(context.Background(), "demo", "ghp_def")
	testhelp.Equal(t, "example 8 variant a refused push", []any{res.Pushed, res.PushError, pDCErr(err)},
		[]any{false, "bootstrap: push failed (exit 1): rejected", "<nil>"})
}

func TestProbeDaemonCoreExample9(t *testing.T) {
	h := pDCNew(t, "demo", "beta")
	d := h.open()
	v, err := d.PlanLoad("demo", pDCPlan(t))
	testhelp.Equal(t, "example 9 PlanLoad(demo).State", []any{v.State, pDCErr(err)}, []any{"running", "<nil>"})
	v, err = d.PlanLoad("beta", pDCPlan(t))
	testhelp.Equal(t, "example 9 PlanLoad(beta)", []any{v.State, v.Index, v.Phases, pDCErr(err)}, []any{"queued", 0, 3, "<nil>"})
	launches, _ := h.spawns()
	testhelp.Equal(t, "example 9 one spawn", len(launches), 1)
	s := pDCStatus(t, d, "beta")
	testhelp.Equal(t, "example 9 Status(beta) State, Queue.State", []string{s.State, s.Queue.State}, []string{"idle", "queued"})
	err = d.WaitOperator("demo", "id-2", "smoke", "stop", "")
	testhelp.Equal(t, "example 9 WaitOperator(demo)", pDCErr(err), "<nil>")
	d.Tick(context.Background())
	launches, _ = h.spawns()
	if testhelp.Equal(t, "example 9 two spawns after Tick", len(launches), 2) {
		testhelp.Equal(t, "example 9 the second spawn is beta's", []string{launches[1].Dir, launches[1].SessionID},
			[]string{filepath.Join(h.tmp, "projects", "beta"), "id-3"})
	}
	s = pDCStatus(t, d, "beta")
	testhelp.Equal(t, "example 9 Status(beta) after Tick", []string{s.State, s.Phase}, []string{"busy", "P17"})
	// variant: the oldest queued project begins first, not the first by name
	h3 := pDCNew(t, "demo", "beta", "gamma")
	d3 := h3.open()
	_, _ = d3.PlanLoad("demo", pDCPlan(t))
	_, _ = d3.PlanLoad("gamma", pDCPlan(t))
	_, _ = d3.PlanLoad("beta", pDCPlan(t))
	_ = d3.WaitOperator("demo", "id-2", "smoke", "stop", "")
	d3.Tick(context.Background())
	l3, _ := h3.spawns()
	dir3 := ""
	if len(l3) == 2 {
		dir3 = l3[1].Dir
	}
	testhelp.Equal(t, "example 9 variant the oldest queued (gamma) begins", dir3, filepath.Join(h3.tmp, "projects", "gamma"))
	// variant: a second Tick begins nothing more
	d.Tick(context.Background())
	launches, _ = h.spawns()
	testhelp.Equal(t, "example 9 variant no third spawn", len(launches), 2)
}

func TestProbeDaemonCoreExample10(t *testing.T) {
	h := pDCNew(t, "demo")
	d := h.open()
	_, err := d.CreateProject(context.Background(), "demo", "go", "https://github.com/acme/demo")
	testhelp.Equal(t, "example 10 CreateProject(demo) is control.ErrExists, no command", []any{errors.Is(err, control.ErrExists), len(h.run.Calls)}, []any{true, 0})
	v, err := d.CreateProject(context.Background(), "gamma", "python", "https://github.com/acme/gamma")
	created := v.CreatedAt
	v.CreatedAt = time.Time{}
	testhelp.Equal(t, "example 10 CreateProject(gamma)", []any{v, pDCErr(err), created.IsZero()},
		[]any{control.ProjectView{Name: "gamma", Language: "python", RepoURL: "https://github.com/acme/gamma", State: "idle"}, "<nil>", false})
	p, err := h.reg.Get("gamma")
	testhelp.Equal(t, "example 10 the registry's gamma", []any{p.Dir, p.Caps, p.RepoURL, pDCErr(err)},
		[]any{filepath.Join(h.tmp, "projects", "gamma"), queue.Caps{ClaudeUSD: 25, Hours: 2, ExecutorUSD: 4}, "https://github.com/acme/gamma", "<nil>"})
	first := runner.Call{}
	if len(h.run.Calls) > 0 {
		first = h.run.Calls[0]
	}
	testhelp.Equal(t, "example 10 the first command", first, runner.Call{Dir: filepath.Join(h.tmp, "projects"), Name: "/usr/local/bin/morph",
		Args: []string{"init", "--root", filepath.Join(h.tmp, "projects", "gamma"), "--name", "gamma", "--language", "python"}})
	names := []string{}
	for _, pv := range d.Projects() {
		names = append(names, pv.Name+" "+pv.State)
	}
	testhelp.Equal(t, "example 10 Projects()", names, []string{"demo idle", "gamma idle"})
	_, err = d.CreateProject(context.Background(), "delta", "cobol", "https://github.com/acme/delta")
	testhelp.Equal(t, "example 10 an unknown language is ErrBadInput", []any{errors.Is(err, control.ErrBadInput), pDCErr(err)},
		[]any{true, `bad input: bootstrap: unknown language "cobol"`})
	// variant: a failing command is returned as is (not bad input)
	h.run.Script["git init -q -b main"] = runner.Result{Code: 128, Stderr: "fatal"}
	_, err = d.CreateProject(context.Background(), "omega", "go", "https://github.com/acme/omega")
	testhelp.Equal(t, "example 10 variant a failed step", []any{errors.Is(err, control.ErrBadInput), pDCErr(err)},
		[]any{false, "bootstrap: step 2 failed (exit 128): fatal"})
}

// Variants of the behaviour the examples do not pin: Open over a saved running loop, Continue's queue error.
func TestProbeDaemonCoreBehaviour(t *testing.T) {
	h, d := pDCRunning(t)
	_ = d
	// a second daemon over the same registry finds P17 running: Exited(-2) at once, the start check, a fresh spawn
	h2 := &pDCH{t: t, tmp: h.tmp, state: h.state, reg: h.reg, run: h.run, clock: time.Date(2026, 10, 8, 16, 0, 0, 0, time.UTC), ids: 10}
	h2.open()
	launches, _ := h2.spawns()
	posts := h2.postList()
	testhelp.Equal(t, "behaviour Open over a running loop: spawns and posts", []any{len(launches), posts}, []any{1, [][4]string{
		{"demo", "watchdog", "P17: session exited (code -2), resuming", "resume 1 of 3 this hour"},
		{"demo", "start", "P17 started", "session id-11 · cap $30 · 3 h"}}})
	// Continue after the plan's end: the queue's error is wrapped as ErrNotWaiting
	h3 := pDCNew(t, "demo")
	d3 := h3.open()
	if _, err := d3.PlanLoad("demo", queue.Plan{Approved: "abc1234", Phases: []queue.Phase{{ID: "P17"}}}); err != nil {
		t.Fatalf("setup: %v", err)
	}
	err := d3.PhaseDone("demo", "id-2", "P17", "")
	testhelp.Equal(t, "behaviour PhaseDone of the last phase", pDCErr(err), "<nil>")
	st, _ := d3.StopCheck("demo")
	kind := ""
	if st != nil {
		kind = st.Kind
	}
	testhelp.Equal(t, "behaviour the stop after the last phase", kind, "end")
	if startPump != nil {
		// the killed process stays current until its pump handles the exit (and saves): wait for that before the end
		deadline := time.Now().Add(2 * time.Second)
		for time.Now().Before(deadline) {
			if _, err := d3.Order("demo", "x"); errors.Is(err, control.ErrNoSession) {
				break
			}
			time.Sleep(10 * time.Millisecond)
		}
	}
	err = d3.Continue("demo")
	testhelp.Equal(t, "behaviour Continue after the end", []any{errors.Is(err, control.ErrNotWaiting), pDCErr(err)},
		[]any{true, `not waiting: queue: nothing to continue (state "done")`})
	// the state file is saved after every executed batch
	var s State
	err = h3.reg.LoadState("demo", &s)
	testhelp.Equal(t, "behaviour the saved state", []any{pDCErr(err), s.Loop.State, s.Loop.Stop != nil}, []any{"<nil>", "waiting", true})
}
