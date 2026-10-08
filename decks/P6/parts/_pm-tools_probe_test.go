package mcpserver

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"sort"
	"sync"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"morphstudio/control"
	"morphstudio/github"
	"morphstudio/internal/testhelp"
	"morphstudio/queue"
	"morphstudio/session"
)

// pPMFake implements control.Control; each call is recorded with its arguments, errs[method] is returned.
type pPMFake struct {
	mu       sync.Mutex
	calls    []string
	errs     map[string]error
	status   control.Status
	order    session.OrderResult
	question *control.Question
	stop     *control.Stop
	usage    control.Usage
	queue    control.QueueView
	plan     queue.Plan
	projects []control.ProjectView
	created  control.ProjectView
}

func (f *pPMFake) rec(m string, err string, args ...any) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	s := m + "("
	for i, a := range args {
		if i > 0 {
			s += ", "
		}
		s += fmt.Sprintf("%#v", a)
	}
	f.calls = append(f.calls, s+")")
	return f.errs[m]
}

func (f *pPMFake) Projects() []control.ProjectView { f.rec("Projects", ""); return f.projects }
func (f *pPMFake) CreateProject(ctx context.Context, name, language, repoURL string) (control.ProjectView, error) {
	return f.created, f.rec("CreateProject", "", name, language, repoURL)
}
func (f *pPMFake) Status(project string) (control.Status, error) {
	return f.status, f.rec("Status", "", project)
}
func (f *pPMFake) Events(project string, since int64, max int) (control.Events, error) {
	return control.Events{}, f.rec("Events", "", project)
}
func (f *pPMFake) Order(project, text string) (session.OrderResult, error) {
	return f.order, f.rec("Order", "", project, text)
}
func (f *pPMFake) Pending(project string) (*control.Question, error) {
	return f.question, f.rec("Pending", "", project)
}
func (f *pPMFake) Answer(project string, option int) error { return f.rec("Answer", "", project, option) }
func (f *pPMFake) Interrupt(project string) error          { return f.rec("Interrupt", "", project) }
func (f *pPMFake) Usage(project string) (control.Usage, error) {
	return f.usage, f.rec("Usage", "", project)
}
func (f *pPMFake) Restart(project string, force bool) error {
	return f.rec("Restart", "", project, force)
}
func (f *pPMFake) PlanLoad(project string, plan queue.Plan) (control.QueueView, error) {
	f.plan = plan
	return f.queue, f.rec("PlanLoad", "", project)
}
func (f *pPMFake) Continue(project string) error { return f.rec("Continue", "", project) }
func (f *pPMFake) StopCheck(project string) (*control.Stop, error) {
	return f.stop, f.rec("StopCheck", "", project)
}
func (f *pPMFake) PutGithubToken(ctx context.Context, project, token string) (control.TokenResult, error) {
	return control.TokenResult{}, f.rec("PutGithubToken", "", project)
}
func (f *pPMFake) PhaseDone(project, sessionToken, phase, next string) error {
	return f.rec("PhaseDone", "", project, sessionToken, phase, next)
}
func (f *pPMFake) WaitOperator(project, sessionToken, kind, reason, issueURL string) error {
	return f.rec("WaitOperator", "", project, sessionToken, kind, reason, issueURL)
}
func (f *pPMFake) Milestone(project, sessionToken string, m control.Milestone) error {
	return f.rec("Milestone", "", project, sessionToken, m.Kind, m.Headline, m.Numbers)
}

var _ control.Control = (*pPMFake)(nil)

// pPMConnect connects a client to s over in-memory transports; a nil server is a readable red.
func pPMConnect(t *testing.T, what string, s *mcp.Server) *mcp.ClientSession {
	t.Helper()
	if s == nil {
		t.Fatalf("%s: the server is nil", what)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)
	t.Cleanup(cancel)
	ct, st := mcp.NewInMemoryTransports()
	ss, err := s.Connect(ctx, st, nil)
	if err != nil {
		t.Fatalf("%s: server connect: %v", what, err)
	}
	cs, err := mcp.NewClient(&mcp.Implementation{Name: "t", Version: "0"}, nil).Connect(ctx, ct, nil)
	if err != nil {
		t.Fatalf("%s: client connect: %v", what, err)
	}
	t.Cleanup(func() { cs.Close(); ss.Wait() })
	return cs
}

func pPMNames(t *testing.T, cs *mcp.ClientSession) []string {
	t.Helper()
	r, err := cs.ListTools(context.Background(), nil)
	if err != nil {
		return []string{"list error: " + err.Error()}
	}
	names := []string{}
	for _, tl := range r.Tools {
		names = append(names, tl.Name)
		if tl.Description == "" {
			names = append(names, "(no description: "+tl.Name+")")
		}
	}
	sort.Strings(names)
	return names
}

// pPMRes is what a tool call answered: the error flag, the texts of the content and whether anything structured came.
type pPMRes struct {
	IsError    bool
	Texts      []string
	Structured bool
}

func pPMCall(cs *mcp.ClientSession, name string, args any) pPMRes {
	r, err := cs.CallTool(context.Background(), &mcp.CallToolParams{Name: name, Arguments: args})
	if err != nil {
		return pPMRes{IsError: true, Texts: []string{"call error: " + err.Error()}}
	}
	out := pPMRes{IsError: r.IsError, Texts: []string{}, Structured: r.StructuredContent != nil}
	for _, c := range r.Content {
		if tc, ok := c.(*mcp.TextContent); ok {
			out.Texts = append(out.Texts, tc.Text)
		} else {
			out.Texts = append(out.Texts, fmt.Sprintf("not text: %T", c))
		}
	}
	return out
}

func pPMOK(text string) pPMRes { return pPMRes{Texts: []string{text}} }
func pPMErr(text string) pPMRes { return pPMRes{IsError: true, Texts: []string{text}} }

var pPMPMNames = []string{"answer", "continue", "interrupt", "order", "pending", "plan_load", "restart", "status", "stop_check", "usage"}

var pPMStatus = control.Status{State: "busy", Phase: "P18", Minutes: 37, CostUSD: 1.92, FiveHour: 22, SevenDay: 60, SessionID: "s-1",
	Queue: control.QueueView{State: "running", Approved: "7b31dfe", Index: 1, Phases: 3, Current: "P18"},
	Repo:  github.Access{Checked: true, Reachable: true, Push: true, CheckedAt: time.Date(2026, 10, 8, 15, 0, 0, 0, time.UTC)}}

const pPMStatusJSON = `{"state":"busy","phase":"P18","minutes":37,"cost_usd":1.92,"five_hour":22,"seven_day":60,"session_id":"s-1","queue":{"state":"running","approved":"7b31dfe","index":1,"phases":3,"current":"P18","reason":""},"repo":{"checked":true,"reachable":true,"push":true,"reason":"","checked_at":"2026-10-08T15:00:00Z"}}`

func TestProbePMToolsExample1(t *testing.T) {
	cs := pPMConnect(t, "example 1", PMServer(&pPMFake{}, "demo"))
	testhelp.Equal(t, "example 1 PMServer tool names", pPMNames(t, cs), pPMPMNames)
	// variant: the server's name and version
	info := cs.InitializeResult().ServerInfo
	testhelp.Equal(t, "example 1 variant server info", []string{info.Name, info.Version, ServerName, ServerVersion}, []string{"morphd", "0.1.0", "morphd", "0.1.0"})
}

func TestProbePMToolsExample2(t *testing.T) {
	f := &pPMFake{status: pPMStatus}
	cs := pPMConnect(t, "example 2", PMServer(f, "demo"))
	testhelp.Equal(t, "example 2 status", pPMCall(cs, "status", nil), pPMOK(pPMStatusJSON))
	testhelp.Equal(t, "example 2 calls", f.calls, []string{`Status("demo")`})
	// variant: another project, arguments {}
	g := &pPMFake{status: control.Status{State: "idle"}}
	cs = pPMConnect(t, "example 2 variant", PMServer(g, "beta"))
	got := pPMCall(cs, "status", map[string]any{})
	testhelp.Equal(t, "example 2 variant status of beta", got, pPMOK(`{"state":"idle","phase":"","minutes":0,"cost_usd":0,"five_hour":0,"seven_day":0,"session_id":"","queue":{"state":"","approved":"","index":0,"phases":0,"current":"","reason":""},"repo":{"checked":false,"reachable":false,"push":false,"reason":"","checked_at":"0001-01-01T00:00:00Z"}}`))
	testhelp.Equal(t, "example 2 variant calls", g.calls, []string{`Status("beta")`})
}

func TestProbePMToolsExample3(t *testing.T) {
	f := &pPMFake{order: session.OrderResult{Queued: 1}, queue: control.QueueView{State: "queued", Approved: "7b31dfe", Index: 0, Phases: 3, Current: "P17"}}
	cs := pPMConnect(t, "example 3", PMServer(f, "demo"))
	testhelp.Equal(t, "example 3 order", pPMCall(cs, "order", map[string]any{"text": "smoke checked green. Resume with P12b"}), pPMOK(`{"queued":1}`))
	testhelp.Equal(t, "example 3 answer", pPMCall(cs, "answer", map[string]any{"option": 1}), pPMOK(`{"ok":true}`))
	raw, err := os.ReadFile("../tests/fixtures/plan/queue-3.json")
	if err != nil {
		t.Fatal(err)
	}
	var args map[string]any
	if err := json.Unmarshal(raw, &args); err != nil {
		t.Fatal(err)
	}
	testhelp.Equal(t, "example 3 plan_load", pPMCall(cs, "plan_load", args), pPMOK(`{"state":"queued","approved":"7b31dfe","index":0,"phases":3,"current":"P17","reason":""}`))
	testhelp.Equal(t, "example 3 calls", f.calls, []string{`Order("demo", "smoke checked green. Resume with P12b")`, `Answer("demo", 1)`, `PlanLoad("demo")`})
	testhelp.Equal(t, "example 3 the plan given to PlanLoad", f.plan, queue.Plan{Approved: "7b31dfe", Phases: []queue.Phase{
		{ID: "P17", StopAfter: "none"}, {ID: "P18", StopAfter: "smoke", Caps: queue.Caps{ClaudeUSD: 12}}, {ID: "P19", StopAfter: "none"}}})
	// variant: every caps field, a phase with no stop_after, the {"sent":true} answer
	g := &pPMFake{order: session.OrderResult{Sent: true}}
	cs = pPMConnect(t, "example 3 variant", PMServer(g, "beta"))
	testhelp.Equal(t, "example 3 variant order", pPMCall(cs, "order", map[string]any{"text": "go"}), pPMOK(`{"sent":true}`))
	pPMCall(cs, "plan_load", map[string]any{"approved": "abc1234", "phases": []any{
		map[string]any{"id": "P1", "caps": map[string]any{"claude_usd": 2.5, "hours": 1.5, "executor_usd": 0.75}}, map[string]any{"id": "P2", "stop_after": "operator"}}})
	testhelp.Equal(t, "example 3 variant plan", g.plan, queue.Plan{Approved: "abc1234", Phases: []queue.Phase{
		{ID: "P1", Caps: queue.Caps{ClaudeUSD: 2.5, Hours: 1.5, ExecutorUSD: 0.75}}, {ID: "P2", StopAfter: "operator"}}})
	testhelp.Equal(t, "example 3 variant calls", g.calls, []string{`Order("beta", "go")`, `PlanLoad("beta")`})
}

func TestProbePMToolsExample4(t *testing.T) {
	f := &pPMFake{errs: map[string]error{"Answer": control.ErrNoQuestion, "Status": errors.New("disk full")}}
	cs := pPMConnect(t, "example 4", PMServer(f, "beta"))
	testhelp.Equal(t, "example 4 answer", pPMCall(cs, "answer", map[string]any{"option": 1}), pPMErr("no_question: no pending question"))
	testhelp.Equal(t, "example 4 status", pPMCall(cs, "status", nil), pPMErr("internal: disk full"))
	testhelp.Equal(t, "example 4 calls", f.calls, []string{`Answer("beta", 1)`, `Status("beta")`})
	// variant: a wrapped sentinel keeps its whole text; other tools map errors the same way
	g := &pPMFake{errs: map[string]error{"Order": fmt.Errorf("wrap: %w", control.ErrBusy), "Interrupt": control.ErrNotBusy, "Continue": control.ErrNotWaiting,
		"Restart": control.ErrNoSession, "Usage": control.ErrUnknownProject, "PlanLoad": control.ErrBadInput, "Pending": control.ErrBadToken, "StopCheck": errors.New("x")}}
	cs = pPMConnect(t, "example 4 variant", PMServer(g, "demo"))
	testhelp.Equal(t, "example 4 variant order", pPMCall(cs, "order", map[string]any{"text": "t"}), pPMErr("busy: wrap: session is busy"))
	testhelp.Equal(t, "example 4 variant interrupt", pPMCall(cs, "interrupt", nil), pPMErr("not_busy: session is not busy"))
	testhelp.Equal(t, "example 4 variant continue", pPMCall(cs, "continue", nil), pPMErr("not_waiting: not waiting"))
	testhelp.Equal(t, "example 4 variant restart", pPMCall(cs, "restart", map[string]any{}), pPMErr("no_session: no session running"))
	testhelp.Equal(t, "example 4 variant usage", pPMCall(cs, "usage", nil), pPMErr("unknown_project: unknown project"))
	testhelp.Equal(t, "example 4 variant plan_load", pPMCall(cs, "plan_load", map[string]any{"approved": "a", "phases": []any{}}), pPMErr("bad_input: bad input"))
	testhelp.Equal(t, "example 4 variant pending", pPMCall(cs, "pending", nil), pPMErr("bad_session_token: session token mismatch"))
	testhelp.Equal(t, "example 4 variant stop_check", pPMCall(cs, "stop_check", nil), pPMErr("internal: x"))
	h := &pPMFake{errs: map[string]error{"CreateProject": control.ErrExists}}
	cu := pPMConnect(t, "example 4 variant user", UserServer(h))
	testhelp.Equal(t, "example 4 variant project_create", pPMCall(cu, "project_create", map[string]any{"name": "a", "language": "go", "repo_url": "u"}), pPMErr("exists: project exists"))
}

func TestProbePMToolsExample5(t *testing.T) {
	f := &pPMFake{usage: control.Usage{FiveHour: 22, SevenDay: 60, FiveHourResetsAt: 1791469200, SevenDayResetsAt: 1791853200, SessionCostUSD: 0.0064, StretchCostUSD: 5.2609,
		LimitsAt: time.Date(2026, 10, 8, 13, 9, 3, 0, time.UTC)}}
	cs := pPMConnect(t, "example 5", PMServer(f, "demo"))
	got := []pPMRes{pPMCall(cs, "pending", nil), pPMCall(cs, "stop_check", nil), pPMCall(cs, "interrupt", nil), pPMCall(cs, "continue", nil),
		pPMCall(cs, "restart", map[string]any{"force": true}), pPMCall(cs, "usage", nil)}
	testhelp.Equal(t, "example 5 texts", got, []pPMRes{pPMOK(`{"question":null}`), pPMOK(`{"stop":null}`), pPMOK(`{"ok":true}`), pPMOK(`{"ok":true}`), pPMOK(`{"ok":true}`),
		pPMOK(`{"five_hour":22,"seven_day":60,"five_hour_resets_at":1791469200,"seven_day_resets_at":1791853200,"session_cost_usd":0.0064,"stretch_cost_usd":5.2609,"limits_at":"2026-10-08T13:09:03Z"}`)})
	testhelp.Equal(t, "example 5 calls", f.calls, []string{`Pending("demo")`, `StopCheck("demo")`, `Interrupt("demo")`, `Continue("demo")`, `Restart("demo", true)`, `Usage("demo")`})
	// variant: a question and a stop; restart with force absent is false
	g := &pPMFake{question: &control.Question{RequestID: "r-2", Text: "Pick", Header: "H", Options: []string{"x", "y"}, AskedAt: time.Date(2026, 10, 8, 13, 9, 8, 0, time.UTC)},
		stop: &control.Stop{Kind: "smoke", Phase: "P18", Reason: "stop after smoke", At: time.Date(2026, 10, 8, 16, 0, 0, 0, time.UTC)}}
	cs = pPMConnect(t, "example 5 variant", PMServer(g, "gamma"))
	testhelp.Equal(t, "example 5 variant pending", pPMCall(cs, "pending", nil), pPMOK(`{"question":{"request_id":"r-2","text":"Pick","header":"H","options":["x","y"],"asked_at":"2026-10-08T13:09:08Z"}}`))
	testhelp.Equal(t, "example 5 variant stop_check", pPMCall(cs, "stop_check", nil), pPMOK(`{"stop":{"kind":"smoke","phase":"P18","reason":"stop after smoke","at":"2026-10-08T16:00:00Z"}}`))
	testhelp.Equal(t, "example 5 variant restart {}", pPMCall(cs, "restart", map[string]any{}), pPMOK(`{"ok":true}`))
	testhelp.Equal(t, "example 5 variant calls", g.calls, []string{`Pending("gamma")`, `StopCheck("gamma")`, `Restart("gamma", false)`})
}

func TestProbePMToolsExample6(t *testing.T) {
	f := &pPMFake{projects: []control.ProjectView{{Name: "alpha", Language: "ts", RepoURL: "https://github.com/o/alpha", State: "idle", CreatedAt: time.Date(2026, 10, 7, 9, 0, 0, 0, time.UTC)}},
		created: control.ProjectView{Name: "demo", Language: "go", RepoURL: "https://github.com/o/demo", State: "idle", CreatedAt: time.Date(2026, 10, 8, 15, 0, 0, 0, time.UTC)}}
	cs := pPMConnect(t, "example 6", UserServer(f))
	testhelp.Equal(t, "example 6 UserServer tool names", pPMNames(t, cs), []string{"project_create", "projects_list"})
	testhelp.Equal(t, "example 6 projects_list", pPMCall(cs, "projects_list", nil), pPMOK(`{"projects":[{"name":"alpha","language":"ts","repo_url":"https://github.com/o/alpha","state":"idle","created_at":"2026-10-07T09:00:00Z"}]}`))
	testhelp.Equal(t, "example 6 project_create", pPMCall(cs, "project_create", map[string]any{"name": "demo", "language": "go", "repo_url": "https://github.com/o/demo"}),
		pPMOK(`{"name":"demo","language":"go","repo_url":"https://github.com/o/demo","state":"idle","created_at":"2026-10-08T15:00:00Z"}`))
	testhelp.Equal(t, "example 6 calls", f.calls, []string{"Projects()", `CreateProject("demo", "go", "https://github.com/o/demo")`})
	// variant: a nil list is [], a missing repo_url fails the schema
	g := &pPMFake{}
	cs = pPMConnect(t, "example 6 variant", UserServer(g))
	testhelp.Equal(t, "example 6 variant projects_list of nil", pPMCall(cs, "projects_list", nil), pPMOK(`{"projects":[]}`))
	testhelp.Equal(t, "example 6 variant project_create without repo_url is an error", pPMCall(cs, "project_create", map[string]any{"name": "x", "language": "go"}).IsError, true)
	testhelp.Equal(t, "example 6 variant calls", g.calls, []string{"Projects()"})
}

func TestProbePMToolsExample7(t *testing.T) {
	f := &pPMFake{}
	cs := pPMConnect(t, "example 7", PMServer(f, "demo"))
	testhelp.Equal(t, "example 7 answer {\"option\": \"one\"} is an error", pPMCall(cs, "answer", map[string]any{"option": "one"}).IsError, true)
	// variant: a missing required field
	testhelp.Equal(t, "example 7 variant order {} is an error", pPMCall(cs, "order", map[string]any{}).IsError, true)
	testhelp.Equal(t, "example 7 variant answer {} is an error", pPMCall(cs, "answer", map[string]any{}).IsError, true)
	testhelp.Equal(t, "example 7 the fake saw nothing", len(f.calls), 0)
	testhelp.Equal(t, "example 7 then answer {\"option\": 2}", pPMCall(cs, "answer", map[string]any{"option": 2}), pPMOK(`{"ok":true}`))
	testhelp.Equal(t, "example 7 calls", f.calls, []string{`Answer("demo", 2)`})
}
