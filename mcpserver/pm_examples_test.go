package mcpserver

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"sort"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"morphstudio/control"
	"morphstudio/eventlog"
	"morphstudio/github"
	"morphstudio/internal/testhelp"
	"morphstudio/queue"
	"morphstudio/session"
)

// ptCall is one recorded call on a fake Control: the method's name and its arguments.
type ptCall struct {
	Method string
	Args   []any
}

// ptFake is a control.Control and control.Adopter with every result and error
// settable, which records every call it receives.
type ptFake struct {
	calls []ptCall

	projects    []control.ProjectView
	projectView control.ProjectView
	createErr   error

	status    control.Status
	statusErr error

	events    control.Events
	eventsErr error

	order    session.OrderResult
	orderErr error

	pending    *control.Question
	pendingErr error

	answerErr    error
	interruptErr error

	usage    control.Usage
	usageErr error

	restartErr error

	planView control.QueueView
	planErr  error

	continueErr error

	stop    *control.Stop
	stopErr error

	adoptView control.ProjectView
	adoptErr  error
}

func (f *ptFake) record(method string, args ...any) {
	f.calls = append(f.calls, ptCall{Method: method, Args: args})
}

func (f *ptFake) Projects() []control.ProjectView {
	f.record("Projects")
	return f.projects
}

func (f *ptFake) CreateProject(ctx context.Context, name, language, repoURL string) (control.ProjectView, error) {
	f.record("CreateProject", name, language, repoURL)
	return f.projectView, f.createErr
}

func (f *ptFake) Status(project string) (control.Status, error) {
	f.record("Status", project)
	return f.status, f.statusErr
}

func (f *ptFake) Events(project string, since int64, max int) (control.Events, error) {
	f.record("Events", project, since, max)
	return f.events, f.eventsErr
}

func (f *ptFake) Order(project, text string) (session.OrderResult, error) {
	f.record("Order", project, text)
	return f.order, f.orderErr
}

func (f *ptFake) Pending(project string) (*control.Question, error) {
	f.record("Pending", project)
	return f.pending, f.pendingErr
}

func (f *ptFake) Answer(project string, option int) error {
	f.record("Answer", project, option)
	return f.answerErr
}

func (f *ptFake) Interrupt(project string) error {
	f.record("Interrupt", project)
	return f.interruptErr
}

func (f *ptFake) Usage(project string) (control.Usage, error) {
	f.record("Usage", project)
	return f.usage, f.usageErr
}

func (f *ptFake) Restart(project string, force bool) error {
	f.record("Restart", project, force)
	return f.restartErr
}

func (f *ptFake) PlanLoad(project string, plan queue.Plan) (control.QueueView, error) {
	f.record("PlanLoad", project, plan)
	return f.planView, f.planErr
}

func (f *ptFake) Continue(project string) error {
	f.record("Continue", project)
	return f.continueErr
}

func (f *ptFake) StopCheck(project string) (*control.Stop, error) {
	f.record("StopCheck", project)
	return f.stop, f.stopErr
}

func (f *ptFake) PutGithubToken(ctx context.Context, project, token string) (control.TokenResult, error) {
	f.record("PutGithubToken", project, token)
	return control.TokenResult{}, nil
}

func (f *ptFake) PhaseDone(project, sessionToken, phase, next string) error {
	f.record("PhaseDone", project, sessionToken, phase, next)
	return nil
}

func (f *ptFake) WaitOperator(project, sessionToken, kind, reason, issueURL string) error {
	f.record("WaitOperator", project, sessionToken, kind, reason, issueURL)
	return nil
}

func (f *ptFake) Milestone(project, sessionToken string, m control.Milestone) error {
	f.record("Milestone", project, sessionToken, m)
	return nil
}

func (f *ptFake) AdoptProject(ctx context.Context, name, dir, repoURL, language string) (control.ProjectView, error) {
	f.record("AdoptProject", name, dir, repoURL, language)
	return f.adoptView, f.adoptErr
}

// ptPlain is a control.Control that is deliberately not a control.Adopter: it
// implements the 17 methods of the interface and nothing else.
type ptPlain struct {
	calls []ptCall
}

func (p *ptPlain) record(method string, args ...any) {
	p.calls = append(p.calls, ptCall{Method: method, Args: args})
}

func (p *ptPlain) Projects() []control.ProjectView {
	p.record("Projects")
	return nil
}

func (p *ptPlain) CreateProject(ctx context.Context, name, language, repoURL string) (control.ProjectView, error) {
	p.record("CreateProject", name, language, repoURL)
	return control.ProjectView{}, nil
}

func (p *ptPlain) Status(project string) (control.Status, error) {
	p.record("Status", project)
	return control.Status{}, nil
}

func (p *ptPlain) Events(project string, since int64, max int) (control.Events, error) {
	p.record("Events", project, since, max)
	return control.Events{}, nil
}

func (p *ptPlain) Order(project, text string) (session.OrderResult, error) {
	p.record("Order", project, text)
	return session.OrderResult{}, nil
}

func (p *ptPlain) Pending(project string) (*control.Question, error) {
	p.record("Pending", project)
	return nil, nil
}

func (p *ptPlain) Answer(project string, option int) error {
	p.record("Answer", project, option)
	return nil
}

func (p *ptPlain) Interrupt(project string) error {
	p.record("Interrupt", project)
	return nil
}

func (p *ptPlain) Usage(project string) (control.Usage, error) {
	p.record("Usage", project)
	return control.Usage{}, nil
}

func (p *ptPlain) Restart(project string, force bool) error {
	p.record("Restart", project, force)
	return nil
}

func (p *ptPlain) PlanLoad(project string, plan queue.Plan) (control.QueueView, error) {
	p.record("PlanLoad", project, plan)
	return control.QueueView{}, nil
}

func (p *ptPlain) Continue(project string) error {
	p.record("Continue", project)
	return nil
}

func (p *ptPlain) StopCheck(project string) (*control.Stop, error) {
	p.record("StopCheck", project)
	return nil, nil
}

func (p *ptPlain) PutGithubToken(ctx context.Context, project, token string) (control.TokenResult, error) {
	p.record("PutGithubToken", project, token)
	return control.TokenResult{}, nil
}

func (p *ptPlain) PhaseDone(project, sessionToken, phase, next string) error {
	p.record("PhaseDone", project, sessionToken, phase, next)
	return nil
}

func (p *ptPlain) WaitOperator(project, sessionToken, kind, reason, issueURL string) error {
	p.record("WaitOperator", project, sessionToken, kind, reason, issueURL)
	return nil
}

func (p *ptPlain) Milestone(project, sessionToken string, m control.Milestone) error {
	p.record("Milestone", project, sessionToken, m)
	return nil
}

// ptConnect connects an in-memory client to s and closes both when the test ends.
func ptConnect(t *testing.T, s *mcp.Server) *mcp.ClientSession {
	t.Helper()
	ctx := context.Background()
	ct, st := mcp.NewInMemoryTransports()
	ss, err := s.Connect(ctx, st, nil)
	if err != nil {
		t.Fatal(err)
	}
	cs, err := mcp.NewClient(&mcp.Implementation{Name: "t", Version: "0"}, nil).Connect(ctx, ct, nil)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		cs.Close()
		ss.Wait()
	})
	return cs
}

// ptTools returns the names of the tools ListTools reports, sorted.
func ptTools(t *testing.T, cs *mcp.ClientSession) []string {
	t.Helper()
	res, err := cs.ListTools(context.Background(), nil)
	if err != nil {
		t.Fatal(err)
	}
	names := make([]string, 0, len(res.Tools))
	for _, tool := range res.Tools {
		names = append(names, tool.Name)
	}
	sort.Strings(names)
	return names
}

// ptText returns the text of a tool result's single content element.
func ptText(t testing.TB, res *mcp.CallToolResult) string {
	t.Helper()
	if len(res.Content) != 1 {
		t.Fatalf("content: got %d elements, want 1", len(res.Content))
	}
	tc, ok := res.Content[0].(*mcp.TextContent)
	if !ok {
		t.Fatalf("content[0]: got %T, want *mcp.TextContent", res.Content[0])
	}
	return tc.Text
}

// TestPMToolsExample1 lists the per-project tools.
func TestPMToolsExample1(t *testing.T) {
	c := &ptFake{}
	cs := ptConnect(t, PMServer(c, "demo"))
	testhelp.Equal(t, "tool names", ptTools(t, cs), []string{"answer", "continue", "events", "interrupt", "order", "pending", "plan_load", "restart", "status", "stop_check", "usage"})
}

// TestPMToolsExample2 reads the project's status.
func TestPMToolsExample2(t *testing.T) {
	c := &ptFake{status: control.Status{
		State:     "busy",
		Phase:     "P18",
		Minutes:   37,
		CostUSD:   1.92,
		FiveHour:  22,
		SevenDay:  60,
		SessionID: "s-1",
		Queue:     control.QueueView{State: "running", Approved: "7b31dfe", Index: 1, Phases: 3, Current: "P18"},
		Repo:      github.Access{Checked: true, Reachable: true, Push: true, CheckedAt: time.Date(2026, 10, 8, 15, 0, 0, 0, time.UTC)},
	}}
	cs := ptConnect(t, PMServer(c, "demo"))

	res, err := cs.CallTool(context.Background(), &mcp.CallToolParams{Name: "status", Arguments: map[string]any{}})
	if err != nil {
		t.Fatal(err)
	}
	testhelp.Equal(t, "is_error", res.IsError, false)
	testhelp.Equal(t, "structured_content", res.StructuredContent, nil)
	testhelp.Equal(t, "text", ptText(t, res), `{"state":"busy","phase":"P18","minutes":37,"cost_usd":1.92,"five_hour":22,"seven_day":60,"session_id":"s-1","queue":{"state":"running","approved":"7b31dfe","index":1,"phases":3,"current":"P18","reason":""},"repo":{"checked":true,"reachable":true,"push":true,"reason":"","checked_at":"2026-10-08T15:00:00Z"}}`)
	testhelp.Equal(t, "calls", c.calls, []ptCall{{Method: "Status", Args: []any{"demo"}}})
}

// TestPMToolsExample3 orders, answers and loads a plan.
func TestPMToolsExample3(t *testing.T) {
	c := &ptFake{
		order:    session.OrderResult{Queued: 1},
		planView: control.QueueView{State: "queued", Approved: "7b31dfe", Index: 0, Phases: 3, Current: "P17"},
	}
	cs := ptConnect(t, PMServer(c, "demo"))
	ctx := context.Background()

	res, err := cs.CallTool(ctx, &mcp.CallToolParams{Name: "order", Arguments: map[string]any{"text": "smoke checked green. Resume with P12b"}})
	if err != nil {
		t.Fatal(err)
	}
	testhelp.Equal(t, "order text", ptText(t, res), `{"queued":1}`)
	testhelp.Equal(t, "order calls", c.calls, []ptCall{{Method: "Order", Args: []any{"demo", "smoke checked green. Resume with P12b"}}})

	res, err = cs.CallTool(ctx, &mcp.CallToolParams{Name: "answer", Arguments: map[string]any{"option": 1}})
	if err != nil {
		t.Fatal(err)
	}
	testhelp.Equal(t, "answer text", ptText(t, res), `{"ok":true}`)
	testhelp.Equal(t, "answer calls", c.calls, []ptCall{
		{Method: "Order", Args: []any{"demo", "smoke checked green. Resume with P12b"}},
		{Method: "Answer", Args: []any{"demo", 1}},
	})

	data, err := os.ReadFile("../tests/fixtures/plan/queue-3.json")
	if err != nil {
		t.Fatal(err)
	}
	plan := map[string]any{}
	if err := json.Unmarshal(data, &plan); err != nil {
		t.Fatal(err)
	}
	res, err = cs.CallTool(ctx, &mcp.CallToolParams{Name: "plan_load", Arguments: plan})
	if err != nil {
		t.Fatal(err)
	}
	testhelp.Equal(t, "plan_load text", ptText(t, res), `{"state":"queued","approved":"7b31dfe","index":0,"phases":3,"current":"P17","reason":""}`)
	testhelp.Equal(t, "calls", c.calls, []ptCall{
		{Method: "Order", Args: []any{"demo", "smoke checked green. Resume with P12b"}},
		{Method: "Answer", Args: []any{"demo", 1}},
		{Method: "PlanLoad", Args: []any{"demo", queue.Plan{Approved: "7b31dfe", Phases: []queue.Phase{
			{ID: "P17", StopAfter: "none"},
			{ID: "P18", StopAfter: "smoke", Caps: queue.Caps{ClaudeUSD: 12}},
			{ID: "P19", StopAfter: "none"},
		}}}},
	})
}

// TestPMToolsExample4 reports the control layer's errors as isError results.
func TestPMToolsExample4(t *testing.T) {
	c := &ptFake{answerErr: control.ErrNoQuestion}
	cs := ptConnect(t, PMServer(c, "beta"))
	ctx := context.Background()

	res, err := cs.CallTool(ctx, &mcp.CallToolParams{Name: "answer", Arguments: map[string]any{"option": 1}})
	if err != nil {
		t.Fatal(err)
	}
	testhelp.Equal(t, "is_error", res.IsError, true)
	testhelp.Equal(t, "text", ptText(t, res), `no_question: no pending question`)
	testhelp.Equal(t, "calls", c.calls, []ptCall{{Method: "Answer", Args: []any{"beta", 1}}})

	c.statusErr = errors.New("disk full")
	res, err = cs.CallTool(ctx, &mcp.CallToolParams{Name: "status", Arguments: map[string]any{}})
	if err != nil {
		t.Fatal(err)
	}
	testhelp.Equal(t, "is_error", res.IsError, true)
	testhelp.Equal(t, "text", ptText(t, res), `internal: disk full`)
	testhelp.Equal(t, "calls", c.calls, []ptCall{
		{Method: "Answer", Args: []any{"beta", 1}},
		{Method: "Status", Args: []any{"beta"}},
	})
}

// TestPMToolsExample5 reads the null views and the mutating tools' ok views.
func TestPMToolsExample5(t *testing.T) {
	c := &ptFake{usage: control.Usage{
		FiveHour:         22,
		SevenDay:         60,
		FiveHourResetsAt: 1791469200,
		SevenDayResetsAt: 1791853200,
		SessionCostUSD:   0.0064,
		StretchCostUSD:   5.2609,
		LimitsAt:         time.Date(2026, 10, 8, 13, 9, 3, 0, time.UTC),
	}}
	cs := ptConnect(t, PMServer(c, "demo"))
	ctx := context.Background()

	want := []struct {
		name string
		args map[string]any
		text string
	}{
		{"pending", map[string]any{}, `{"question":null}`},
		{"stop_check", map[string]any{}, `{"stop":null}`},
		{"interrupt", map[string]any{}, `{"ok":true}`},
		{"continue", map[string]any{}, `{"ok":true}`},
		{"restart", map[string]any{"force": true}, `{"ok":true}`},
		{"usage", map[string]any{}, `{"five_hour":22,"seven_day":60,"five_hour_resets_at":1791469200,"seven_day_resets_at":1791853200,"session_cost_usd":0.0064,"stretch_cost_usd":5.2609,"limits_at":"2026-10-08T13:09:03Z"}`},
	}
	for _, w := range want {
		res, err := cs.CallTool(ctx, &mcp.CallToolParams{Name: w.name, Arguments: w.args})
		if err != nil {
			t.Fatal(err)
		}
		testhelp.Equal(t, w.name+" text", ptText(t, res), w.text)
	}
	testhelp.Equal(t, "calls", c.calls, []ptCall{
		{Method: "Pending", Args: []any{"demo"}},
		{Method: "StopCheck", Args: []any{"demo"}},
		{Method: "Interrupt", Args: []any{"demo"}},
		{Method: "Continue", Args: []any{"demo"}},
		{Method: "Restart", Args: []any{"demo", true}},
		{Method: "Usage", Args: []any{"demo"}},
	})
}

// TestPMToolsExample6 lists, creates and (never) adopts through the user server.
func TestPMToolsExample6(t *testing.T) {
	c := &ptFake{
		projects: []control.ProjectView{{
			Name:      "alpha",
			Language:  "ts",
			RepoURL:   "https://github.com/o/alpha",
			State:     "idle",
			CreatedAt: time.Date(2026, 10, 7, 9, 0, 0, 0, time.UTC),
		}},
		projectView: control.ProjectView{
			Name:      "demo",
			Language:  "go",
			RepoURL:   "https://github.com/o/demo",
			State:     "idle",
			CreatedAt: time.Date(2026, 10, 8, 15, 0, 0, 0, time.UTC),
		},
	}
	cs := ptConnect(t, UserServer(c))
	ctx := context.Background()

	testhelp.Equal(t, "tool names", ptTools(t, cs), []string{"project_adopt", "project_create", "projects_list"})

	res, err := cs.CallTool(ctx, &mcp.CallToolParams{Name: "projects_list", Arguments: map[string]any{}})
	if err != nil {
		t.Fatal(err)
	}
	testhelp.Equal(t, "projects_list text", ptText(t, res), `{"projects":[{"name":"alpha","language":"ts","repo_url":"https://github.com/o/alpha","state":"idle","created_at":"2026-10-07T09:00:00Z"}]}`)

	res, err = cs.CallTool(ctx, &mcp.CallToolParams{Name: "project_create", Arguments: map[string]any{
		"name":     "demo",
		"language": "go",
		"repo_url": "https://github.com/o/demo",
	}})
	if err != nil {
		t.Fatal(err)
	}
	testhelp.Equal(t, "project_create text", ptText(t, res), `{"name":"demo","language":"go","repo_url":"https://github.com/o/demo","state":"idle","created_at":"2026-10-08T15:00:00Z"}`)

	testhelp.Equal(t, "calls", c.calls, []ptCall{
		{Method: "Projects"},
		{Method: "CreateProject", Args: []any{"demo", "go", "https://github.com/o/demo"}},
	})
}

// TestPMToolsExample7 refuses an option of the wrong JSON type.
func TestPMToolsExample7(t *testing.T) {
	c := &ptFake{}
	cs := ptConnect(t, PMServer(c, "demo"))
	ctx := context.Background()

	res, err := cs.CallTool(ctx, &mcp.CallToolParams{Name: "answer", Arguments: map[string]any{"option": "one"}})
	if err == nil && !res.IsError {
		t.Errorf("answer with a string option: got no error and isError false")
	}
	testhelp.Equal(t, "calls", c.calls, []ptCall(nil))

	res, err = cs.CallTool(ctx, &mcp.CallToolParams{Name: "answer", Arguments: map[string]any{"option": 2}})
	if err != nil {
		t.Fatal(err)
	}
	testhelp.Equal(t, "is_error", res.IsError, false)
	testhelp.Equal(t, "text", ptText(t, res), `{"ok":true}`)
	testhelp.Equal(t, "calls", c.calls, []ptCall{{Method: "Answer", Args: []any{"demo", 2}}})
}

// TestPMToolsExample8 renders windows of the project's event log.
func TestPMToolsExample8(t *testing.T) {
	ent := func(n int) eventlog.Entry {
		dir, msg := testhelp.ProbeLine(t, "../tests/fixtures/stream/probe.jsonl", n)
		return eventlog.Entry{Seq: int64(n), Dir: dir, Msg: msg}
	}
	c := &ptFake{events: control.Events{Entries: []eventlog.Entry{ent(3), ent(5), ent(7)}, Last: 7}}
	cs := ptConnect(t, PMServer(c, "demo"))
	ctx := context.Background()

	res, err := cs.CallTool(ctx, &mcp.CallToolParams{Name: "events", Arguments: map[string]any{}})
	if err != nil {
		t.Fatal(err)
	}
	testhelp.Equal(t, "is_error", res.IsError, false)
	testhelp.Equal(t, "structured_content", res.StructuredContent, nil)
	testhelp.Equal(t, "text", ptText(t, res), `{"events":[{"seq":3,"dir":"in","type":"user","text":"Reply with exactly one word: PONG"},{"seq":5,"dir":"out","type":"assistant","text":"PONG"},{"seq":7,"dir":"out","type":"result","subtype":"success","text":"PONG","cost_usd":0.0021784900000000004,"terminal_reason":"completed"}],"last":7}`)
	testhelp.Equal(t, "calls", c.calls, []ptCall{{Method: "Events", Args: []any{"demo", int64(0), 50}}})

	c.events = control.Events{Entries: []eventlog.Entry{}, Last: 7}
	res, err = cs.CallTool(ctx, &mcp.CallToolParams{Name: "events", Arguments: map[string]any{"since": 7, "limit": 10}})
	if err != nil {
		t.Fatal(err)
	}
	testhelp.Equal(t, "text", ptText(t, res), `{"events":[],"last":7}`)
	testhelp.Equal(t, "calls", c.calls[1:], []ptCall{{Method: "Events", Args: []any{"demo", int64(7), 10}}})

	c.eventsErr = errors.New("disk full")
	res, err = cs.CallTool(ctx, &mcp.CallToolParams{Name: "events", Arguments: map[string]any{"since": 2}})
	if err != nil {
		t.Fatal(err)
	}
	testhelp.Equal(t, "is_error", res.IsError, true)
	testhelp.Equal(t, "text", ptText(t, res), `internal: disk full`)
	testhelp.Equal(t, "calls", c.calls, []ptCall{
		{Method: "Events", Args: []any{"demo", int64(0), 50}},
		{Method: "Events", Args: []any{"demo", int64(7), 10}},
		{Method: "Events", Args: []any{"demo", int64(2), 50}},
	})
}

// TestPMToolsExample9 adopts an existing repository, or reports why it cannot.
func TestPMToolsExample9(t *testing.T) {
	c := &ptFake{adoptView: control.ProjectView{
		Name:      "studio",
		Language:  "go",
		RepoURL:   "https://github.com/VasyaLutiy/MorphStudio",
		State:     "idle",
		CreatedAt: time.Date(2026, 10, 10, 18, 0, 0, 0, time.UTC),
	}}
	cs := ptConnect(t, UserServer(c))
	ctx := context.Background()

	args := map[string]any{
		"name":     "studio",
		"dir":      "/home/morph/MorphStudio",
		"repo_url": "https://github.com/VasyaLutiy/MorphStudio",
		"language": "go",
	}
	res, err := cs.CallTool(ctx, &mcp.CallToolParams{Name: "project_adopt", Arguments: args})
	if err != nil {
		t.Fatal(err)
	}
	testhelp.Equal(t, "text", ptText(t, res), `{"name":"studio","language":"go","repo_url":"https://github.com/VasyaLutiy/MorphStudio","state":"idle","created_at":"2026-10-10T18:00:00Z"}`)
	testhelp.Equal(t, "calls", c.calls, []ptCall{{Method: "AdoptProject", Args: []any{"studio", "/home/morph/MorphStudio", "https://github.com/VasyaLutiy/MorphStudio", "go"}}})

	c.adoptErr = fmt.Errorf("%w: bootstrap: refused: not a git repository: /home/morph/nowhere", control.ErrBadInput)
	res, err = cs.CallTool(ctx, &mcp.CallToolParams{Name: "project_adopt", Arguments: args})
	if err != nil {
		t.Fatal(err)
	}
	testhelp.Equal(t, "is_error", res.IsError, true)
	testhelp.Equal(t, "text", ptText(t, res), `bad_input: bad input: bootstrap: refused: not a git repository: /home/morph/nowhere`)

	c.adoptErr = control.ErrExists
	res, err = cs.CallTool(ctx, &mcp.CallToolParams{Name: "project_adopt", Arguments: args})
	if err != nil {
		t.Fatal(err)
	}
	testhelp.Equal(t, "is_error", res.IsError, true)
	testhelp.Equal(t, "text", ptText(t, res), `exists: project exists`)

	res, err = cs.CallTool(ctx, &mcp.CallToolParams{Name: "project_adopt", Arguments: map[string]any{"name": "studio"}})
	if err == nil && !res.IsError {
		t.Errorf("project_adopt with missing fields: got no error and isError false")
	}
	testhelp.Equal(t, "calls", c.calls, []ptCall{
		{Method: "AdoptProject", Args: []any{"studio", "/home/morph/MorphStudio", "https://github.com/VasyaLutiy/MorphStudio", "go"}},
		{Method: "AdoptProject", Args: []any{"studio", "/home/morph/MorphStudio", "https://github.com/VasyaLutiy/MorphStudio", "go"}},
		{Method: "AdoptProject", Args: []any{"studio", "/home/morph/MorphStudio", "https://github.com/VasyaLutiy/MorphStudio", "go"}},
	})

	p := &ptPlain{}
	cs2 := ptConnect(t, UserServer(p))
	res, err = cs2.CallTool(ctx, &mcp.CallToolParams{Name: "project_adopt", Arguments: args})
	if err != nil {
		t.Fatal(err)
	}
	testhelp.Equal(t, "is_error", res.IsError, true)
	testhelp.Equal(t, "text", ptText(t, res), `internal: adopt not supported`)
	testhelp.Equal(t, "calls", p.calls, []ptCall(nil))
}
