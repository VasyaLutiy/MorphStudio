package mcpserver

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sort"
	"strings"
	"sync"
	"testing"
	"time"

	"morphstudio/control"
	"morphstudio/github"
	"morphstudio/internal/testhelp"
	"morphstudio/queue"
	"morphstudio/session"
)

// pMMFake implements control.Control; each call is recorded with its arguments.
type pMMFake struct {
	mu     sync.Mutex
	calls  []string
	status control.Status
}

func (f *pMMFake) rec(m string, args ...any) {
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
}

func (f *pMMFake) Projects() []control.ProjectView { f.rec("Projects"); return nil }
func (f *pMMFake) CreateProject(ctx context.Context, name, language, repoURL string) (control.ProjectView, error) {
	f.rec("CreateProject", name)
	return control.ProjectView{}, nil
}
func (f *pMMFake) Status(project string) (control.Status, error) {
	f.rec("Status", project)
	return f.status, nil
}
func (f *pMMFake) Events(project string, since int64, max int) (control.Events, error) {
	f.rec("Events", project)
	return control.Events{}, nil
}
func (f *pMMFake) Order(project, text string) (session.OrderResult, error) {
	f.rec("Order", project, text)
	return session.OrderResult{Sent: true}, nil
}
func (f *pMMFake) Pending(project string) (*control.Question, error) {
	f.rec("Pending", project)
	return nil, nil
}
func (f *pMMFake) Answer(project string, option int) error { f.rec("Answer", project); return nil }
func (f *pMMFake) Interrupt(project string) error          { f.rec("Interrupt", project); return nil }
func (f *pMMFake) Usage(project string) (control.Usage, error) {
	f.rec("Usage", project)
	return control.Usage{}, nil
}
func (f *pMMFake) Restart(project string, force bool) error { f.rec("Restart", project); return nil }
func (f *pMMFake) PlanLoad(project string, plan queue.Plan) (control.QueueView, error) {
	f.rec("PlanLoad", project)
	return control.QueueView{}, nil
}
func (f *pMMFake) Continue(project string) error { f.rec("Continue", project); return nil }
func (f *pMMFake) StopCheck(project string) (*control.Stop, error) {
	f.rec("StopCheck", project)
	return nil, nil
}
func (f *pMMFake) PutGithubToken(ctx context.Context, project, token string) (control.TokenResult, error) {
	f.rec("PutGithubToken", project)
	return control.TokenResult{}, nil
}
func (f *pMMFake) PhaseDone(project, sessionToken, phase, next string) error {
	f.rec("PhaseDone", project, sessionToken, phase, next)
	return nil
}
func (f *pMMFake) WaitOperator(project, sessionToken, kind, reason, issueURL string) error {
	f.rec("WaitOperator", project, sessionToken, kind)
	return nil
}
func (f *pMMFake) Milestone(project, sessionToken string, m control.Milestone) error {
	f.rec("Milestone", project, sessionToken, m.Kind, m.Headline, m.Numbers)
	return nil
}

var _ control.Control = (*pMMFake)(nil)

var pMMStatus = control.Status{State: "busy", Phase: "P18", Minutes: 37, CostUSD: 1.92, FiveHour: 22, SevenDay: 60, SessionID: "s-1",
	Queue: control.QueueView{State: "running", Approved: "7b31dfe", Index: 1, Phases: 3, Current: "P18"},
	Repo:  github.Access{Checked: true, Reachable: true, Push: true, CheckedAt: time.Date(2026, 10, 8, 15, 0, 0, 0, time.UTC)}}

const pMMStatusJSON = `{"state":"busy","phase":"P18","minutes":37,"cost_usd":1.92,"five_hour":22,"seven_day":60,"session_id":"s-1","queue":{"state":"running","approved":"7b31dfe","index":1,"phases":3,"current":"P18","reason":""},"repo":{"checked":true,"reachable":true,"push":true,"reason":"","checked_at":"2026-10-08T15:00:00Z"}}`

const pMM401 = `{"error":{"code":"unauthorized","message":"bearer token required"}}` + "\n"
const pMM404 = `{"error":{"code":"unknown_project","message":"unknown project"}}` + "\n"
const pMMList = `{"jsonrpc":"2.0","id":1,"method":"tools/list","params":{}}`

func pMMHandler(f *pMMFake, api string, known []string, sessions map[string]string) http.Handler {
	h := Handler(f, func(p string) bool {
		for _, k := range known {
			if k == p {
				return true
			}
		}
		return false
	}, Tokens{API: api, Session: func(p string) (string, bool) { s, ok := sessions[p]; return s, ok }})
	if h == nil {
		return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(501) })
	}
	return h
}

func pMMExample1(f *pMMFake) http.Handler {
	return pMMHandler(f, "tok-1", []string{"demo"}, map[string]string{"demo": "id-2"})
}

type pMMResp struct {
	Code        int
	ContentType string
	Body        string
}

func pMMServe(h http.Handler, method, target, auth, body string) pMMResp {
	req := httptest.NewRequest(method, target, strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")
	if auth != "" {
		req.Header.Set("Authorization", auth)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return pMMResp{rec.Code, rec.Header().Get("Content-Type"), rec.Body.String()}
}

// pMMRPC reads a JSON-RPC answer: its id, the sorted tool names of a tools/list, the content texts and isError of a call.
type pMMRPC struct {
	Status  int
	JSON    bool
	ID      any
	Tools   []string
	Texts   []string
	IsError bool
}

func pMMRead(r pMMResp) pMMRPC {
	out := pMMRPC{Status: r.Code, JSON: strings.HasPrefix(r.ContentType, "application/json")}
	var m struct {
		ID     any `json:"id"`
		Result struct {
			Tools []struct {
				Name string `json:"name"`
			} `json:"tools"`
			Content []struct {
				Type string `json:"type"`
				Text string `json:"text"`
			} `json:"content"`
			IsError bool `json:"isError"`
		} `json:"result"`
		Error any `json:"error"`
	}
	if err := json.Unmarshal([]byte(r.Body), &m); err != nil {
		out.Texts = []string{"body not JSON: " + r.Body}
		return out
	}
	if m.Error != nil {
		out.Texts = []string{fmt.Sprintf("json-rpc error: %v", m.Error)}
	}
	out.ID = m.ID
	for _, tl := range m.Result.Tools {
		out.Tools = append(out.Tools, tl.Name)
	}
	sort.Strings(out.Tools)
	for _, c := range m.Result.Content {
		out.Texts = append(out.Texts, c.Type+" "+c.Text)
	}
	out.IsError = m.Result.IsError
	return out
}

func pMMCallBody(id int, name, args string) string {
	return fmt.Sprintf(`{"jsonrpc":"2.0","id":%d,"method":"tools/call","params":{"name":%q,"arguments":%s}}`, id, name, args)
}

func TestProbeMCPMountExample1(t *testing.T) {
	f := &pMMFake{status: pMMStatus}
	got := pMMRead(pMMServe(pMMExample1(f), "POST", "/mcp/demo", "Bearer tok-1", pMMList))
	testhelp.Equal(t, "example 1 POST /mcp/demo tools/list", got, pMMRPC{Status: 200, JSON: true, ID: float64(1),
		Tools: []string{"answer", "continue", "interrupt", "order", "pending", "plan_load", "restart", "status", "stop_check", "usage"}})
	// variant: the scheme in lower case
	got = pMMRead(pMMServe(pMMExample1(f), "POST", "/mcp/demo", "bearer tok-1", pMMList))
	testhelp.Equal(t, "example 1 variant \"bearer tok-1\" status", got.Status, 200)
}

func TestProbeMCPMountExample2(t *testing.T) {
	f := &pMMFake{status: pMMStatus}
	got := pMMRead(pMMServe(pMMExample1(f), "POST", "/mcp/demo", "Bearer tok-1", pMMCallBody(2, "status", "{}")))
	testhelp.Equal(t, "example 2 tools/call status", got, pMMRPC{Status: 200, JSON: true, ID: float64(2), Texts: []string{"text " + pMMStatusJSON}})
	testhelp.Equal(t, "example 2 calls", f.calls, []string{`Status("demo")`})
	// variant: another known project and another API token
	g := &pMMFake{status: control.Status{State: "idle"}}
	h := pMMHandler(g, "k-77", []string{"demo", "beta"}, nil)
	got = pMMRead(pMMServe(h, "POST", "/mcp/beta", "Bearer k-77", pMMCallBody(3, "order", `{"text":"hello"}`)))
	testhelp.Equal(t, "example 2 variant order on beta", got, pMMRPC{Status: 200, JSON: true, ID: float64(3), Texts: []string{`text {"sent":true}`}})
	testhelp.Equal(t, "example 2 variant calls", g.calls, []string{`Order("beta", "hello")`})
	testhelp.Equal(t, "example 2 variant tok-1 refused", pMMServe(h, "POST", "/mcp/beta", "Bearer tok-1", pMMList), pMMResp{401, "application/json", pMM401})
}

func TestProbeMCPMountExample3(t *testing.T) {
	f := &pMMFake{}
	got := pMMRead(pMMServe(pMMExample1(f), "POST", "/mcp/demo/session", "Bearer id-2", pMMCallBody(4, "milestone", `{"kind":"gate","headline":"P17 gate passed","numbers":"10 cards · $0.12"}`)))
	testhelp.Equal(t, "example 3 session milestone", got, pMMRPC{Status: 200, JSON: true, ID: float64(4), Texts: []string{`text {"ok":true}`}})
	testhelp.Equal(t, "example 3 calls", f.calls, []string{`Milestone("demo", "id-2", "gate", "P17 gate passed", "10 cards · $0.12")`})
	// variant: another project's session with its own token; its tool list
	g := &pMMFake{}
	h := pMMHandler(g, "tok-1", []string{"demo", "beta"}, map[string]string{"demo": "id-2", "beta": "s-b"})
	got = pMMRead(pMMServe(h, "POST", "/mcp/beta/session", "Bearer s-b", pMMCallBody(5, "phase_done", `{"phase":"P3","next":"P4"}`)))
	testhelp.Equal(t, "example 3 variant beta phase_done", got, pMMRPC{Status: 200, JSON: true, ID: float64(5), Texts: []string{`text {"ok":true}`}})
	testhelp.Equal(t, "example 3 variant calls", g.calls, []string{`PhaseDone("beta", "s-b", "P3", "P4")`})
	got = pMMRead(pMMServe(h, "POST", "/mcp/beta/session", "Bearer s-b", pMMList))
	testhelp.Equal(t, "example 3 variant session tools", got.Tools, []string{"milestone", "phase_done", "wait_operator"})
	testhelp.Equal(t, "example 3 variant beta with demo's token", pMMServe(h, "POST", "/mcp/beta/session", "Bearer id-2", pMMList), pMMResp{401, "application/json", pMM401})
}

func TestProbeMCPMountExample4(t *testing.T) {
	f := &pMMFake{}
	h := pMMExample1(f)
	reqs := []struct{ target, auth string }{
		{"/mcp/demo", ""}, {"/mcp/demo", "Bearer wrong"}, {"/mcp/demo/session", "Bearer tok-1"}, {"/mcp/other/session", "Bearer id-2"},
	}
	for _, q := range reqs {
		testhelp.Equal(t, fmt.Sprintf("example 4 POST %s with %q", q.target, q.auth), pMMServe(h, "POST", q.target, q.auth, pMMList), pMMResp{401, "application/json", pMM401})
	}
	testhelp.Equal(t, "example 4 the fake saw nothing", len(f.calls), 0)
	// variant: /mcp without a token, a trailing space, the Basic scheme, an empty API token, an empty session token
	more := []struct{ target, auth string }{
		{"/mcp", ""}, {"/mcp", "Bearer wrong"}, {"/mcp/demo", "Bearer tok-1 "}, {"/mcp/demo", "Basic dG9rLTE="}, {"/mcp/demo", "tok-1"}, {"/mcp/demo/session", "Bearer id-"},
	}
	for _, q := range more {
		testhelp.Equal(t, fmt.Sprintf("example 4 variant POST %s with %q", q.target, q.auth), pMMServe(h, "POST", q.target, q.auth, pMMList), pMMResp{401, "application/json", pMM401})
	}
	e := pMMHandler(f, "", []string{"demo"}, map[string]string{"demo": ""})
	testhelp.Equal(t, "example 4 variant empty API token, \"Bearer \"", pMMServe(e, "POST", "/mcp/demo", "Bearer ", pMMList), pMMResp{401, "application/json", pMM401})
	testhelp.Equal(t, "example 4 variant empty session token, \"Bearer \"", pMMServe(e, "POST", "/mcp/demo/session", "Bearer ", pMMList), pMMResp{401, "application/json", pMM401})
	testhelp.Equal(t, "example 4 variant the fake saw nothing", len(f.calls), 0)
}

func TestProbeMCPMountExample5(t *testing.T) {
	f := &pMMFake{}
	h := pMMExample1(f)
	testhelp.Equal(t, "example 5 POST /mcp/nope", pMMServe(h, "POST", "/mcp/nope", "Bearer tok-1", pMMList), pMMResp{404, "application/json", pMM404})
	testhelp.Equal(t, "example 5 POST /mcp/nope without a token", pMMServe(h, "POST", "/mcp/nope", "", pMMList), pMMResp{401, "application/json", pMM401})
	// variant: a session token known for an unknown project
	g := pMMHandler(f, "tok-1", []string{"demo"}, map[string]string{"demo": "id-2", "ghost": "g-1"})
	testhelp.Equal(t, "example 5 variant POST /mcp/ghost/session", pMMServe(g, "POST", "/mcp/ghost/session", "Bearer g-1", pMMList), pMMResp{404, "application/json", pMM404})
	testhelp.Equal(t, "example 5 variant the fake saw nothing", len(f.calls), 0)
}

func TestProbeMCPMountExample6(t *testing.T) {
	f := &pMMFake{}
	got := pMMRead(pMMServe(pMMExample1(f), "POST", "/mcp", "Bearer tok-1", pMMList))
	testhelp.Equal(t, "example 6 POST /mcp tools/list", got, pMMRPC{Status: 200, JSON: true, ID: float64(1), Tools: []string{"project_create", "projects_list"}})
	// variant: projects_list through /mcp
	got = pMMRead(pMMServe(pMMExample1(f), "POST", "/mcp", "Bearer tok-1", pMMCallBody(6, "projects_list", "{}")))
	testhelp.Equal(t, "example 6 variant projects_list", got, pMMRPC{Status: 200, JSON: true, ID: float64(6), Texts: []string{`text {"projects":[]}`}})
}

func TestProbeMCPMountExample7(t *testing.T) {
	h := pMMExample1(&pMMFake{})
	testhelp.Equal(t, "example 7 GET /mcp/demo, DELETE /mcp", []int{pMMServe(h, "GET", "/mcp/demo", "Bearer tok-1", "").Code, pMMServe(h, "DELETE", "/mcp", "Bearer tok-1", "").Code}, []int{405, 405})
	// variant: GET /mcp, DELETE /mcp/demo/session
	testhelp.Equal(t, "example 7 variant GET /mcp, DELETE /mcp/demo/session", []int{pMMServe(h, "GET", "/mcp", "", "").Code, pMMServe(h, "DELETE", "/mcp/demo/session", "Bearer id-2", "").Code}, []int{405, 405})
}
