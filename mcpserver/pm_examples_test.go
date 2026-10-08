package mcpserver

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"sort"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"morphstudio/control"
	"morphstudio/github"
	"morphstudio/internal/testhelp"
	"morphstudio/queue"
	"morphstudio/session"
)

// ptCall is one recorded Control call: the method name and its arguments.
type ptCall struct {
	Method string
	Args   []any
}

// ptFake is the Control fake of this test file: every method of
// control.Control with its exact signature, each call recorded in order,
// with a result and an error field per method to steer the answers.
type ptFake struct {
	calls []ptCall

	projects     []control.ProjectView
	created      control.ProjectView
	createErr    error
	status       control.Status
	statusErr    error
	events       control.Events
	eventsErr    error
	order        session.OrderResult
	orderErr     error
	question     *control.Question
	pendingErr   error
	answerErr    error
	interruptErr error
	usage        control.Usage
	usageErr     error
	restartErr   error
	queueView    control.QueueView
	planLoadErr  error
	continueErr  error
	stop         *control.Stop
	stopCheckErr error
	token        control.TokenResult
	tokenErr     error
	phaseDoneErr error
	waitErr      error
	milestoneErr error
}

var _ control.Control = (*ptFake)(nil)

func (f *ptFake) rec(method string, args ...any) {
	f.calls = append(f.calls, ptCall{Method: method, Args: args})
}

func (f *ptFake) Projects() []control.ProjectView {
	f.rec("Projects")
	return f.projects
}

func (f *ptFake) CreateProject(_ context.Context, name, language, repoURL string) (control.ProjectView, error) {
	f.rec("CreateProject", name, language, repoURL)
	return f.created, f.createErr
}

func (f *ptFake) Status(project string) (control.Status, error) {
	f.rec("Status", project)
	return f.status, f.statusErr
}

func (f *ptFake) Events(project string, since int64, max int) (control.Events, error) {
	f.rec("Events", project, since, max)
	return f.events, f.eventsErr
}

func (f *ptFake) Order(project, text string) (session.OrderResult, error) {
	f.rec("Order", project, text)
	return f.order, f.orderErr
}

func (f *ptFake) Pending(project string) (*control.Question, error) {
	f.rec("Pending", project)
	return f.question, f.pendingErr
}

func (f *ptFake) Answer(project string, option int) error {
	f.rec("Answer", project, option)
	return f.answerErr
}

func (f *ptFake) Interrupt(project string) error {
	f.rec("Interrupt", project)
	return f.interruptErr
}

func (f *ptFake) Usage(project string) (control.Usage, error) {
	f.rec("Usage", project)
	return f.usage, f.usageErr
}

func (f *ptFake) Restart(project string, force bool) error {
	f.rec("Restart", project, force)
	return f.restartErr
}

func (f *ptFake) PlanLoad(project string, plan queue.Plan) (control.QueueView, error) {
	f.rec("PlanLoad", project, plan)
	return f.queueView, f.planLoadErr
}

func (f *ptFake) Continue(project string) error {
	f.rec("Continue", project)
	return f.continueErr
}

func (f *ptFake) StopCheck(project string) (*control.Stop, error) {
	f.rec("StopCheck", project)
	return f.stop, f.stopCheckErr
}

func (f *ptFake) PutGithubToken(_ context.Context, project, token string) (control.TokenResult, error) {
	f.rec("PutGithubToken", project, token)
	return f.token, f.tokenErr
}

func (f *ptFake) PhaseDone(project, sessionToken, phase, next string) error {
	f.rec("PhaseDone", project, sessionToken, phase, next)
	return f.phaseDoneErr
}

func (f *ptFake) WaitOperator(project, sessionToken, kind, reason, issueURL string) error {
	f.rec("WaitOperator", project, sessionToken, kind, reason, issueURL)
	return f.waitErr
}

func (f *ptFake) Milestone(project, sessionToken string, m control.Milestone) error {
	f.rec("Milestone", project, sessionToken, m)
	return f.milestoneErr
}

// ptConnect wires s to a client over the in-memory transports and closes
// the pair (client first, then the server session) when the test ends.
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

// ptCallTool calls one tool and fails the test on a transport error.
func ptCallTool(t *testing.T, cs *mcp.ClientSession, name string, args map[string]any) *mcp.CallToolResult {
	t.Helper()
	res, err := cs.CallTool(context.Background(), &mcp.CallToolParams{Name: name, Arguments: args})
	if err != nil {
		t.Fatalf("CallTool %s: %v", name, err)
	}
	return res
}

// ptText returns the text of the one TextContent of a result.
func ptText(t *testing.T, res *mcp.CallToolResult) string {
	t.Helper()
	if len(res.Content) != 1 {
		t.Fatalf("Content: got %d entries, want 1", len(res.Content))
	}
	tc, ok := res.Content[0].(*mcp.TextContent)
	if !ok {
		t.Fatalf("Content[0]: got %T, want *mcp.TextContent", res.Content[0])
	}
	return tc.Text
}

// ptNames returns the tool names of a list result in the order listed.
func ptNames(list *mcp.ListToolsResult) []string {
	names := make([]string, 0, len(list.Tools))
	for _, tool := range list.Tools {
		names = append(names, tool.Name)
	}
	return names
}

// ptDemoStatus is the Status of example 2.
func ptDemoStatus() control.Status {
	return control.Status{
		State:     "busy",
		Phase:     "P18",
		Minutes:   37,
		CostUSD:   1.92,
		FiveHour:  22,
		SevenDay:  60,
		SessionID: "s-1",
		Queue:     control.QueueView{State: "running", Approved: "7b31dfe", Index: 1, Phases: 3, Current: "P18"},
		Repo:      github.Access{Checked: true, Reachable: true, Push: true, CheckedAt: time.Date(2026, 10, 8, 15, 0, 0, 0, time.UTC)},
	}
}

// ptPlanArgs reads the plan fixture and decodes it into the arguments of plan_load.
func ptPlanArgs(t *testing.T) map[string]any {
	t.Helper()
	data, err := os.ReadFile("../tests/fixtures/plan/queue-3.json")
	if err != nil {
		t.Fatal(err)
	}
	var args map[string]any
	if err := json.Unmarshal(data, &args); err != nil {
		t.Fatal(err)
	}
	return args
}

// TestPMToolsExample1: the PM server lists the ten PM tools.
func TestPMToolsExample1(t *testing.T) {
	ctx := context.Background()
	c := &ptFake{}
	cs := ptConnect(t, PMServer(c, "demo"))

	list, err := cs.ListTools(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	names := ptNames(list)
	sort.Strings(names)
	testhelp.Equal(t, "tool names", names, []string{"answer", "continue", "interrupt", "order", "pending", "plan_load", "restart", "status", "stop_check", "usage"})
	testhelp.Equal(t, "calls", len(c.calls), 0)
}

// TestPMToolsExample2: status renders the Status as one text content.
func TestPMToolsExample2(t *testing.T) {
	c := &ptFake{status: ptDemoStatus()}
	cs := ptConnect(t, PMServer(c, "demo"))

	res := ptCallTool(t, cs, "status", nil)
	testhelp.Equal(t, "IsError", res.IsError, false)
	testhelp.Equal(t, "StructuredContent", res.StructuredContent, nil)
	testhelp.Equal(t, "Text", ptText(t, res),
		`{"state":"busy","phase":"P18","minutes":37,"cost_usd":1.92,"five_hour":22,"seven_day":60,"session_id":"s-1","queue":{"state":"running","approved":"7b31dfe","index":1,"phases":3,"current":"P18","reason":""},"repo":{"checked":true,"reachable":true,"push":true,"reason":"","checked_at":"2026-10-08T15:00:00Z"}}`)
	testhelp.Equal(t, "calls", c.calls, []ptCall{{Method: "Status", Args: []any{"demo"}}})
}

// TestPMToolsExample3: order, answer and plan_load reach the Control with their arguments.
func TestPMToolsExample3(t *testing.T) {
	c := &ptFake{
		order:     session.OrderResult{Queued: 1},
		queueView: control.QueueView{State: "queued", Approved: "7b31dfe", Index: 0, Phases: 3, Current: "P17"},
	}
	cs := ptConnect(t, PMServer(c, "demo"))

	res := ptCallTool(t, cs, "order", map[string]any{"text": "smoke checked green. Resume with P12b"})
	testhelp.Equal(t, "order IsError", res.IsError, false)
	testhelp.Equal(t, "order Text", ptText(t, res), `{"queued":1}`)
	testhelp.Equal(t, "order call", c.calls, []ptCall{
		{Method: "Order", Args: []any{"demo", "smoke checked green. Resume with P12b"}},
	})

	res = ptCallTool(t, cs, "answer", map[string]any{"option": 1})
	testhelp.Equal(t, "answer IsError", res.IsError, false)
	testhelp.Equal(t, "answer Text", ptText(t, res), `{"ok":true}`)
	testhelp.Equal(t, "answer call", c.calls, []ptCall{
		{Method: "Order", Args: []any{"demo", "smoke checked green. Resume with P12b"}},
		{Method: "Answer", Args: []any{"demo", 1}},
	})

	res = ptCallTool(t, cs, "plan_load", ptPlanArgs(t))
	testhelp.Equal(t, "plan_load IsError", res.IsError, false)
	testhelp.Equal(t, "plan_load Text", ptText(t, res), `{"state":"queued","approved":"7b31dfe","index":0,"phases":3,"current":"P17","reason":""}`)
	testhelp.Equal(t, "plan_load call", c.calls, []ptCall{
		{Method: "Order", Args: []any{"demo", "smoke checked green. Resume with P12b"}},
		{Method: "Answer", Args: []any{"demo", 1}},
		{Method: "PlanLoad", Args: []any{"demo", queue.Plan{
			Approved: "7b31dfe",
			Phases: []queue.Phase{
				{ID: "P17", StopAfter: "none"},
				{ID: "P18", StopAfter: "smoke", Caps: queue.Caps{ClaudeUSD: 12}},
				{ID: "P19", StopAfter: "none"},
			},
		}}},
	})
}

// TestPMToolsExample4: a Control error is an error result "code: message".
func TestPMToolsExample4(t *testing.T) {
	c := &ptFake{answerErr: control.ErrNoQuestion}
	cs := ptConnect(t, PMServer(c, "beta"))

	res := ptCallTool(t, cs, "answer", map[string]any{"option": 1})
	testhelp.Equal(t, "answer IsError", res.IsError, true)
	testhelp.Equal(t, "answer Text", ptText(t, res), "no_question: no pending question")
	testhelp.Equal(t, "answer call", c.calls, []ptCall{
		{Method: "Answer", Args: []any{"beta", 1}},
	})

	c.statusErr = errors.New("disk full")
	res = ptCallTool(t, cs, "status", nil)
	testhelp.Equal(t, "status IsError", res.IsError, true)
	testhelp.Equal(t, "status Text", ptText(t, res), "internal: disk full")
	testhelp.Equal(t, "status call", c.calls, []ptCall{
		{Method: "Answer", Args: []any{"beta", 1}},
		{Method: "Status", Args: []any{"beta"}},
	})
}

// TestPMToolsExample5: the no-input tools, restart and usage render their views.
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

	res := ptCallTool(t, cs, "pending", nil)
	testhelp.Equal(t, "pending IsError", res.IsError, false)
	testhelp.Equal(t, "pending Text", ptText(t, res), `{"question":null}`)

	res = ptCallTool(t, cs, "stop_check", nil)
	testhelp.Equal(t, "stop_check IsError", res.IsError, false)
	testhelp.Equal(t, "stop_check Text", ptText(t, res), `{"stop":null}`)

	res = ptCallTool(t, cs, "interrupt", nil)
	testhelp.Equal(t, "interrupt IsError", res.IsError, false)
	testhelp.Equal(t, "interrupt Text", ptText(t, res), `{"ok":true}`)

	res = ptCallTool(t, cs, "continue", nil)
	testhelp.Equal(t, "continue IsError", res.IsError, false)
	testhelp.Equal(t, "continue Text", ptText(t, res), `{"ok":true}`)

	res = ptCallTool(t, cs, "restart", map[string]any{"force": true})
	testhelp.Equal(t, "restart IsError", res.IsError, false)
	testhelp.Equal(t, "restart Text", ptText(t, res), `{"ok":true}`)

	res = ptCallTool(t, cs, "usage", nil)
	testhelp.Equal(t, "usage IsError", res.IsError, false)
	testhelp.Equal(t, "usage Text", ptText(t, res),
		`{"five_hour":22,"seven_day":60,"five_hour_resets_at":1791469200,"seven_day_resets_at":1791853200,"session_cost_usd":0.0064,"stretch_cost_usd":5.2609,"limits_at":"2026-10-08T13:09:03Z"}`)

	testhelp.Equal(t, "calls", c.calls, []ptCall{
		{Method: "Pending", Args: []any{"demo"}},
		{Method: "StopCheck", Args: []any{"demo"}},
		{Method: "Interrupt", Args: []any{"demo"}},
		{Method: "Continue", Args: []any{"demo"}},
		{Method: "Restart", Args: []any{"demo", true}},
		{Method: "Usage", Args: []any{"demo"}},
	})
}

// TestPMToolsExample6: the user server lists and creates projects.
func TestPMToolsExample6(t *testing.T) {
	ctx := context.Background()
	c := &ptFake{
		projects: []control.ProjectView{{
			Name:      "alpha",
			Language:  "ts",
			RepoURL:   "https://github.com/o/alpha",
			State:     "idle",
			CreatedAt: time.Date(2026, 10, 7, 9, 0, 0, 0, time.UTC),
		}},
		created: control.ProjectView{
			Name:      "demo",
			Language:  "go",
			RepoURL:   "https://github.com/o/demo",
			State:     "idle",
			CreatedAt: time.Date(2026, 10, 8, 15, 0, 0, 0, time.UTC),
		},
	}
	cs := ptConnect(t, UserServer(c))

	list, err := cs.ListTools(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	testhelp.Equal(t, "tool names", ptNames(list), []string{"project_create", "projects_list"})

	res := ptCallTool(t, cs, "projects_list", nil)
	testhelp.Equal(t, "projects_list IsError", res.IsError, false)
	testhelp.Equal(t, "projects_list Text", ptText(t, res),
		`{"projects":[{"name":"alpha","language":"ts","repo_url":"https://github.com/o/alpha","state":"idle","created_at":"2026-10-07T09:00:00Z"}]}`)

	res = ptCallTool(t, cs, "project_create", map[string]any{"name": "demo", "language": "go", "repo_url": "https://github.com/o/demo"})
	testhelp.Equal(t, "project_create IsError", res.IsError, false)
	testhelp.Equal(t, "project_create Text", ptText(t, res),
		`{"name":"demo","language":"go","repo_url":"https://github.com/o/demo","state":"idle","created_at":"2026-10-08T15:00:00Z"}`)

	testhelp.Equal(t, "calls", c.calls, []ptCall{
		{Method: "Projects"},
		{Method: "CreateProject", Args: []any{"demo", "go", "https://github.com/o/demo"}},
	})
}

// TestPMToolsExample7: an argument of the wrong type is refused by the schema
// before the Control sees it; the next well-typed call goes through.
func TestPMToolsExample7(t *testing.T) {
	ctx := context.Background()
	c := &ptFake{}
	cs := ptConnect(t, PMServer(c, "demo"))

	res, err := cs.CallTool(ctx, &mcp.CallToolParams{Name: "answer", Arguments: map[string]any{"option": "one"}})
	if err == nil && (res == nil || !res.IsError) {
		t.Errorf("answer with a string option: got a non-error result %#v, want a CallTool error or IsError true", res)
	}
	testhelp.Equal(t, "calls after the refused answer", c.calls, []ptCall(nil))

	res = ptCallTool(t, cs, "answer", map[string]any{"option": 2})
	testhelp.Equal(t, "answer IsError", res.IsError, false)
	testhelp.Equal(t, "answer Text", ptText(t, res), `{"ok":true}`)
	testhelp.Equal(t, "calls", c.calls, []ptCall{
		{Method: "Answer", Args: []any{"demo", 2}},
	})
}
