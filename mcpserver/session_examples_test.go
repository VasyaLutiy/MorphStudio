package mcpserver

import (
	"context"
	"fmt"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"morphstudio/control"
	"morphstudio/internal/testhelp"
	"morphstudio/queue"
	"morphstudio/session"
	"morphstudio/supervisor"
)

// stCall is one recorded Control call: the method name and its arguments.
type stCall struct {
	Method string
	Args   []any
}

// stFake implements control.Control, records each call and returns err.
type stFake struct {
	Calls []stCall
	err   error
}

func (f *stFake) record(method string, args ...any) {
	f.Calls = append(f.Calls, stCall{Method: method, Args: args})
}

func (f *stFake) Projects() []control.ProjectView {
	f.record("Projects")
	return nil
}

func (f *stFake) CreateProject(ctx context.Context, name, language, repoURL string) (control.ProjectView, error) {
	f.record("CreateProject", ctx, name, language, repoURL)
	return control.ProjectView{}, f.err
}

func (f *stFake) Status(project string) (control.Status, error) {
	f.record("Status", project)
	return control.Status{}, f.err
}

func (f *stFake) Events(project string, since int64, max int) (control.Events, error) {
	f.record("Events", project, since, max)
	return control.Events{}, f.err
}

func (f *stFake) Order(project, text string) (session.OrderResult, error) {
	f.record("Order", project, text)
	return session.OrderResult{}, f.err
}

func (f *stFake) Pending(project string) (*control.Question, error) {
	f.record("Pending", project)
	return nil, f.err
}

func (f *stFake) Answer(project string, option int) error {
	f.record("Answer", project, option)
	return f.err
}

func (f *stFake) Interrupt(project string) error {
	f.record("Interrupt", project)
	return f.err
}

func (f *stFake) Usage(project string) (control.Usage, error) {
	f.record("Usage", project)
	return control.Usage{}, f.err
}

func (f *stFake) Restart(project string, force bool) error {
	f.record("Restart", project, force)
	return f.err
}

func (f *stFake) PlanLoad(project string, plan queue.Plan) (control.QueueView, error) {
	f.record("PlanLoad", project, plan)
	return control.QueueView{}, f.err
}

func (f *stFake) Continue(project string) error {
	f.record("Continue", project)
	return f.err
}

func (f *stFake) StopCheck(project string) (*control.Stop, error) {
	f.record("StopCheck", project)
	return nil, f.err
}

func (f *stFake) PutGithubToken(ctx context.Context, project, token string) (control.TokenResult, error) {
	f.record("PutGithubToken", ctx, project, token)
	return control.TokenResult{}, f.err
}

func (f *stFake) PhaseDone(project, sessionToken, phase, next string) error {
	f.record("PhaseDone", project, sessionToken, phase, next)
	return f.err
}

func (f *stFake) WaitOperator(project, sessionToken, kind, reason, issueURL string) error {
	f.record("WaitOperator", project, sessionToken, kind, reason, issueURL)
	return f.err
}

func (f *stFake) Milestone(project, sessionToken string, m control.Milestone) error {
	f.record("Milestone", project, sessionToken, m)
	return f.err
}

// stConnect wires an in-memory server/client pair and closes them at test end.
func stConnect(t *testing.T, s *mcp.Server) *mcp.ClientSession {
	t.Helper()
	ctx := context.Background()
	ct, ts := mcp.NewInMemoryTransports()
	ss, err := s.Connect(ctx, ts, nil)
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

// stText returns the one text content of a result.
func stText(res *mcp.CallToolResult) string {
	return res.Content[0].(*mcp.TextContent).Text
}

func TestSessionToolsExample1(t *testing.T) {
	ctx := context.Background()
	c := &stFake{}
	cs := stConnect(t, SessionServer(c, "demo", "id-2"))

	res, err := cs.ListTools(ctx, nil)
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, tool := range res.Tools {
		names = append(names, tool.Name)
	}
	testhelp.Equal(t, "tool names", names, []string{"milestone", "phase_done", "wait_operator"})
}

func TestSessionToolsExample2(t *testing.T) {
	ctx := context.Background()
	c := &stFake{}
	cs := stConnect(t, SessionServer(c, "demo", "id-2"))

	res, err := cs.CallTool(ctx, &mcp.CallToolParams{
		Name:      "phase_done",
		Arguments: map[string]any{"phase": "P17", "next": "P18"},
	})
	if err != nil {
		t.Fatal(err)
	}
	testhelp.Equal(t, "first Text", stText(res), `{"ok":true}`)
	testhelp.Equal(t, "first IsError", res.IsError, false)
	testhelp.Equal(t, "first call", c.Calls[0], stCall{Method: "PhaseDone", Args: []any{"demo", "id-2", "P17", "P18"}})

	res, err = cs.CallTool(ctx, &mcp.CallToolParams{
		Name:      "phase_done",
		Arguments: map[string]any{"phase": "P19"},
	})
	if err != nil {
		t.Fatal(err)
	}
	testhelp.Equal(t, "second Text", stText(res), `{"ok":true}`)
	testhelp.Equal(t, "second IsError", res.IsError, false)
	testhelp.Equal(t, "second call", c.Calls[1], stCall{Method: "PhaseDone", Args: []any{"demo", "id-2", "P19", ""}})
}

func TestSessionToolsExample3(t *testing.T) {
	ctx := context.Background()
	c := &stFake{err: control.ErrBadToken}
	cs := stConnect(t, SessionServer(c, "beta", "id-9"))

	res, err := cs.CallTool(ctx, &mcp.CallToolParams{
		Name:      "phase_done",
		Arguments: map[string]any{"phase": "P17", "next": "P18"},
	})
	if err != nil {
		t.Fatal(err)
	}
	testhelp.Equal(t, "first Text", stText(res), "bad_session_token: session token mismatch")
	testhelp.Equal(t, "first IsError", res.IsError, true)
	testhelp.Equal(t, "first call", c.Calls[0], stCall{Method: "PhaseDone", Args: []any{"beta", "id-9", "P17", "P18"}})

	c.err = fmt.Errorf("%w: phase_done \"P16\" while running \"P17\"", supervisor.ErrPhaseMismatch)
	res, err = cs.CallTool(ctx, &mcp.CallToolParams{
		Name:      "phase_done",
		Arguments: map[string]any{"phase": "P17", "next": "P18"},
	})
	if err != nil {
		t.Fatal(err)
	}
	testhelp.Equal(t, "second Text", stText(res), "internal: phase mismatch: phase_done \"P16\" while running \"P17\"")
	testhelp.Equal(t, "second IsError", res.IsError, true)
}

func TestSessionToolsExample4(t *testing.T) {
	ctx := context.Background()
	c := &stFake{}
	cs := stConnect(t, SessionServer(c, "demo", "id-2"))

	res, err := cs.CallTool(ctx, &mcp.CallToolParams{
		Name:      "wait_operator",
		Arguments: map[string]any{"kind": "smoke", "reason": "P17 smoke: curl status", "issue_url": ""},
	})
	if err != nil {
		t.Fatal(err)
	}
	testhelp.Equal(t, "first Text", stText(res), `{"ok":true}`)
	testhelp.Equal(t, "first IsError", res.IsError, false)
	testhelp.Equal(t, "first call", c.Calls[0], stCall{Method: "WaitOperator", Args: []any{"demo", "id-2", "smoke", "P17 smoke: curl status", ""}})

	before := len(c.Calls)
	res, err = cs.CallTool(ctx, &mcp.CallToolParams{
		Name:      "wait_operator",
		Arguments: map[string]any{"kind": "pause", "reason": "x"},
	})
	if err != nil {
		t.Fatal(err)
	}
	testhelp.Equal(t, "second Text", stText(res), "bad_input: kind must be one of smoke, gate, emergency, no-next")
	testhelp.Equal(t, "second IsError", res.IsError, true)
	testhelp.Equal(t, "no new calls", len(c.Calls), before)
}

func TestSessionToolsExample5(t *testing.T) {
	ctx := context.Background()
	c := &stFake{}
	cs := stConnect(t, SessionServer(c, "demo", "id-2"))

	res, err := cs.CallTool(ctx, &mcp.CallToolParams{
		Name:      "milestone",
		Arguments: map[string]any{"kind": "gate", "headline": "P17 gate passed", "numbers": "10 cards · $0.12"},
	})
	if err != nil {
		t.Fatal(err)
	}
	testhelp.Equal(t, "first Text", stText(res), `{"ok":true}`)
	testhelp.Equal(t, "first IsError", res.IsError, false)
	testhelp.Equal(t, "first call", c.Calls[0], stCall{Method: "Milestone", Args: []any{"demo", "id-2", control.Milestone{Kind: "gate", Headline: "P17 gate passed", Numbers: "10 cards · $0.12"}}})

	before := len(c.Calls)
	res, err = cs.CallTool(ctx, &mcp.CallToolParams{
		Name:      "milestone",
		Arguments: map[string]any{"kind": "party", "headline": "h", "numbers": ""},
	})
	if err != nil {
		t.Fatal(err)
	}
	testhelp.Equal(t, "second Text", stText(res), `bad_input: unknown milestone kind "party"`)
	testhelp.Equal(t, "second IsError", res.IsError, true)
	testhelp.Equal(t, "no new calls", len(c.Calls), before)
}

func TestSessionToolsExample6(t *testing.T) {
	ctx := context.Background()
	c := &stFake{}
	cs := stConnect(t, SessionServer(c, "demo", "id-2"))

	res, err := cs.CallTool(ctx, &mcp.CallToolParams{
		Name:      "milestone",
		Arguments: map[string]any{"kind": "info", "headline": "h"},
	})
	if err != nil {
		t.Fatal(err)
	}
	testhelp.Equal(t, "Text", stText(res), `{"ok":true}`)
	testhelp.Equal(t, "IsError", res.IsError, false)
	testhelp.Equal(t, "call", c.Calls[0], stCall{Method: "Milestone", Args: []any{"demo", "id-2", control.Milestone{Kind: "info", Headline: "h", Numbers: ""}}})
}
