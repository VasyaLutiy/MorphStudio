package mcpserver

import (
	"context"
	"fmt"
	"sort"
	"sync"
	"testing"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"morphstudio/control"
	"morphstudio/internal/testhelp"
	"morphstudio/queue"
	"morphstudio/session"
	"morphstudio/supervisor"
)

// pSTFake implements control.Control; each call is recorded with its arguments, err is returned by every method.
type pSTFake struct {
	mu    sync.Mutex
	calls []string
	err   error
}

func (f *pSTFake) rec(m string, args ...any) error {
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
	return f.err
}

func (f *pSTFake) Projects() []control.ProjectView { f.rec("Projects"); return nil }
func (f *pSTFake) CreateProject(ctx context.Context, name, language, repoURL string) (control.ProjectView, error) {
	return control.ProjectView{}, f.rec("CreateProject", name)
}
func (f *pSTFake) Status(project string) (control.Status, error) {
	return control.Status{}, f.rec("Status", project)
}
func (f *pSTFake) Events(project string, since int64, max int) (control.Events, error) {
	return control.Events{}, f.rec("Events", project)
}
func (f *pSTFake) Order(project, text string) (session.OrderResult, error) {
	return session.OrderResult{}, f.rec("Order", project)
}
func (f *pSTFake) Pending(project string) (*control.Question, error) {
	return nil, f.rec("Pending", project)
}
func (f *pSTFake) Answer(project string, option int) error { return f.rec("Answer", project) }
func (f *pSTFake) Interrupt(project string) error          { return f.rec("Interrupt", project) }
func (f *pSTFake) Usage(project string) (control.Usage, error) {
	return control.Usage{}, f.rec("Usage", project)
}
func (f *pSTFake) Restart(project string, force bool) error { return f.rec("Restart", project) }
func (f *pSTFake) PlanLoad(project string, plan queue.Plan) (control.QueueView, error) {
	return control.QueueView{}, f.rec("PlanLoad", project)
}
func (f *pSTFake) Continue(project string) error { return f.rec("Continue", project) }
func (f *pSTFake) StopCheck(project string) (*control.Stop, error) {
	return nil, f.rec("StopCheck", project)
}
func (f *pSTFake) PutGithubToken(ctx context.Context, project, token string) (control.TokenResult, error) {
	return control.TokenResult{}, f.rec("PutGithubToken", project)
}
func (f *pSTFake) PhaseDone(project, sessionToken, phase, next string) error {
	return f.rec("PhaseDone", project, sessionToken, phase, next)
}
func (f *pSTFake) WaitOperator(project, sessionToken, kind, reason, issueURL string) error {
	return f.rec("WaitOperator", project, sessionToken, kind, reason, issueURL)
}
func (f *pSTFake) Milestone(project, sessionToken string, m control.Milestone) error {
	return f.rec("Milestone", project, sessionToken, m.Kind, m.Headline, m.Numbers)
}

var _ control.Control = (*pSTFake)(nil)

func pSTConnect(t *testing.T, what string, s *mcp.Server) *mcp.ClientSession {
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

type pSTRes struct {
	IsError bool
	Texts   []string
}

func pSTCall(cs *mcp.ClientSession, name string, args any) pSTRes {
	r, err := cs.CallTool(context.Background(), &mcp.CallToolParams{Name: name, Arguments: args})
	if err != nil {
		return pSTRes{IsError: true, Texts: []string{"call error: " + err.Error()}}
	}
	out := pSTRes{IsError: r.IsError, Texts: []string{}}
	for _, c := range r.Content {
		if tc, ok := c.(*mcp.TextContent); ok {
			out.Texts = append(out.Texts, tc.Text)
		} else {
			out.Texts = append(out.Texts, fmt.Sprintf("not text: %T", c))
		}
	}
	if r.StructuredContent != nil {
		out.Texts = append(out.Texts, "(structured content)")
	}
	return out
}

var pSTOK = pSTRes{Texts: []string{`{"ok":true}`}}

func pSTErr(text string) pSTRes { return pSTRes{IsError: true, Texts: []string{text}} }

func TestProbeSessionToolsExample1(t *testing.T) {
	cs := pSTConnect(t, "example 1", SessionServer(&pSTFake{}, "demo", "id-2"))
	r, err := cs.ListTools(context.Background(), nil)
	names := []string{}
	if err != nil {
		names = append(names, "list error: "+err.Error())
	} else {
		for _, tl := range r.Tools {
			names = append(names, tl.Name)
		}
	}
	sort.Strings(names)
	testhelp.Equal(t, "example 1 SessionServer tool names", names, []string{"milestone", "phase_done", "wait_operator"})
	info := cs.InitializeResult().ServerInfo
	testhelp.Equal(t, "example 1 variant server info", []string{info.Name, info.Version}, []string{"morphd", "0.1.0"})
}

func TestProbeSessionToolsExample2(t *testing.T) {
	f := &pSTFake{}
	cs := pSTConnect(t, "example 2", SessionServer(f, "demo", "id-2"))
	testhelp.Equal(t, "example 2 phase_done P17 → P18", pSTCall(cs, "phase_done", map[string]any{"phase": "P17", "next": "P18"}), pSTOK)
	testhelp.Equal(t, "example 2 phase_done P19 (next omitted)", pSTCall(cs, "phase_done", map[string]any{"phase": "P19"}), pSTOK)
	testhelp.Equal(t, "example 2 calls", f.calls, []string{`PhaseDone("demo", "id-2", "P17", "P18")`, `PhaseDone("demo", "id-2", "P19", "")`})
	// variant: a missing phase fails the schema, the Control not called
	g := &pSTFake{}
	cs = pSTConnect(t, "example 2 variant", SessionServer(g, "demo", "id-2"))
	testhelp.Equal(t, "example 2 variant phase_done without phase is an error", pSTCall(cs, "phase_done", map[string]any{"next": "P18"}).IsError, true)
	testhelp.Equal(t, "example 2 variant calls", len(g.calls), 0)
}

func TestProbeSessionToolsExample3(t *testing.T) {
	f := &pSTFake{err: control.ErrBadToken}
	cs := pSTConnect(t, "example 3", SessionServer(f, "beta", "id-9"))
	testhelp.Equal(t, "example 3 bad token", pSTCall(cs, "phase_done", map[string]any{"phase": "P17", "next": "P18"}), pSTErr("bad_session_token: session token mismatch"))
	testhelp.Equal(t, "example 3 calls", f.calls, []string{`PhaseDone("beta", "id-9", "P17", "P18")`})
	f.err = fmt.Errorf("%w: phase_done \"P16\" while running \"P17\"", supervisor.ErrPhaseMismatch)
	testhelp.Equal(t, "example 3 phase mismatch", pSTCall(cs, "phase_done", map[string]any{"phase": "P16", "next": "P17"}), pSTErr(`internal: phase mismatch: phase_done "P16" while running "P17"`))
	// variant: the other tools map errors the same way
	f.err = control.ErrNoSession
	testhelp.Equal(t, "example 3 variant wait_operator error", pSTCall(cs, "wait_operator", map[string]any{"kind": "gate", "reason": "r"}), pSTErr("no_session: no session running"))
	f.err = fmt.Errorf("post: %w", control.ErrUnknownProject)
	testhelp.Equal(t, "example 3 variant milestone error", pSTCall(cs, "milestone", map[string]any{"kind": "run", "headline": "h"}), pSTErr("unknown_project: post: unknown project"))
}

func TestProbeSessionToolsExample4(t *testing.T) {
	f := &pSTFake{}
	cs := pSTConnect(t, "example 4", SessionServer(f, "demo", "id-2"))
	testhelp.Equal(t, "example 4 wait_operator smoke", pSTCall(cs, "wait_operator", map[string]any{"kind": "smoke", "reason": "P17 smoke: curl status", "issue_url": ""}), pSTOK)
	testhelp.Equal(t, "example 4 wait_operator pause", pSTCall(cs, "wait_operator", map[string]any{"kind": "pause", "reason": "x"}), pSTErr("bad_input: kind must be one of smoke, gate, emergency, no-next"))
	testhelp.Equal(t, "example 4 calls", f.calls, []string{`WaitOperator("demo", "id-2", "smoke", "P17 smoke: curl status", "")`})
	// variant: every other kind; issue_url passed through or omitted
	g := &pSTFake{}
	cs = pSTConnect(t, "example 4 variant", SessionServer(g, "zeta", "tk-5"))
	for _, k := range []string{"gate", "emergency", "no-next"} {
		testhelp.Equal(t, "example 4 variant wait_operator "+k, pSTCall(cs, "wait_operator", map[string]any{"kind": k, "reason": "r " + k, "issue_url": "https://github.com/o/r/issues/3"}), pSTOK)
	}
	testhelp.Equal(t, "example 4 variant wait_operator without issue_url", pSTCall(cs, "wait_operator", map[string]any{"kind": "gate", "reason": "g"}), pSTOK)
	testhelp.Equal(t, "example 4 variant wait_operator Smoke (case)", pSTCall(cs, "wait_operator", map[string]any{"kind": "Smoke", "reason": "r"}).IsError, true)
	testhelp.Equal(t, "example 4 variant calls", g.calls, []string{
		`WaitOperator("zeta", "tk-5", "gate", "r gate", "https://github.com/o/r/issues/3")`,
		`WaitOperator("zeta", "tk-5", "emergency", "r emergency", "https://github.com/o/r/issues/3")`,
		`WaitOperator("zeta", "tk-5", "no-next", "r no-next", "https://github.com/o/r/issues/3")`,
		`WaitOperator("zeta", "tk-5", "gate", "g", "")`})
}

func TestProbeSessionToolsExample5(t *testing.T) {
	f := &pSTFake{}
	cs := pSTConnect(t, "example 5", SessionServer(f, "demo", "id-2"))
	testhelp.Equal(t, "example 5 milestone gate", pSTCall(cs, "milestone", map[string]any{"kind": "gate", "headline": "P17 gate passed", "numbers": "10 cards · $0.12"}), pSTOK)
	testhelp.Equal(t, "example 5 milestone party", pSTCall(cs, "milestone", map[string]any{"kind": "party", "headline": "h", "numbers": ""}), pSTErr(`bad_input: unknown milestone kind "party"`))
	testhelp.Equal(t, "example 5 calls", f.calls, []string{`Milestone("demo", "id-2", "gate", "P17 gate passed", "10 cards · $0.12")`})
	// variant: the exact list; every kind of it accepted
	testhelp.Equal(t, "example 5 variant MilestoneKinds", MilestoneKinds, []string{"start", "gate", "run", "fail", "merge", "smoke", "stop", "debt", "end", "info", "watchdog", "idle", "ask"})
	g := &pSTFake{}
	cs = pSTConnect(t, "example 5 variant", SessionServer(g, "demo", "id-2"))
	bad := []string{}
	for _, k := range []string{"start", "gate", "run", "fail", "merge", "smoke", "stop", "debt", "end", "info", "watchdog", "idle", "ask"} {
		if r := pSTCall(cs, "milestone", map[string]any{"kind": k, "headline": "h"}); r.IsError {
			bad = append(bad, k)
		}
	}
	testhelp.Equal(t, "example 5 variant kinds refused", bad, []string{})
	testhelp.Equal(t, "example 5 variant kind with a quote", pSTCall(cs, "milestone", map[string]any{"kind": `a"b`, "headline": "h"}), pSTErr(`bad_input: unknown milestone kind "a\"b"`))
	testhelp.Equal(t, "example 5 variant calls", len(g.calls), 13)
}

func TestProbeSessionToolsExample6(t *testing.T) {
	f := &pSTFake{}
	cs := pSTConnect(t, "example 6", SessionServer(f, "demo", "id-2"))
	testhelp.Equal(t, "example 6 milestone without numbers", pSTCall(cs, "milestone", map[string]any{"kind": "info", "headline": "h"}), pSTOK)
	testhelp.Equal(t, "example 6 calls", f.calls, []string{`Milestone("demo", "id-2", "info", "h", "")`})
	// variant: a missing headline fails the schema
	testhelp.Equal(t, "example 6 variant milestone without headline is an error", pSTCall(cs, "milestone", map[string]any{"kind": "info"}).IsError, true)
	testhelp.Equal(t, "example 6 variant calls", len(f.calls), 1)
}
