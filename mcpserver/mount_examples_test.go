package mcpserver

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"morphstudio/control"
	"morphstudio/github"
	"morphstudio/internal/testhelp"
	"morphstudio/queue"
	"morphstudio/session"
)

// mmCall is one recorded call on mmFake: the method name and its arguments.
type mmCall struct {
	Method string
	Args   []any
}

// mmFake is the Control fake of this test file: every method of
// control.Control, recording each call, with fields to steer the answers.
type mmFake struct {
	calls []mmCall

	status    control.Status
	statusErr error
}

func (f *mmFake) rec(method string, args ...any) {
	f.calls = append(f.calls, mmCall{Method: method, Args: args})
}

func (f *mmFake) Projects() []control.ProjectView {
	f.rec("Projects")
	return nil
}

func (f *mmFake) CreateProject(_ context.Context, name, language, repoURL string) (control.ProjectView, error) {
	f.rec("CreateProject", name, language, repoURL)
	return control.ProjectView{}, nil
}

func (f *mmFake) Status(project string) (control.Status, error) {
	f.rec("Status", project)
	return f.status, f.statusErr
}

func (f *mmFake) Events(project string, since int64, max int) (control.Events, error) {
	f.rec("Events", project, since, max)
	return control.Events{}, nil
}

func (f *mmFake) Order(project, text string) (session.OrderResult, error) {
	f.rec("Order", project, text)
	return session.OrderResult{}, nil
}

func (f *mmFake) Pending(project string) (*control.Question, error) {
	f.rec("Pending", project)
	return nil, nil
}

func (f *mmFake) Answer(project string, option int) error {
	f.rec("Answer", project, option)
	return nil
}

func (f *mmFake) Interrupt(project string) error {
	f.rec("Interrupt", project)
	return nil
}

func (f *mmFake) Usage(project string) (control.Usage, error) {
	f.rec("Usage", project)
	return control.Usage{}, nil
}

func (f *mmFake) Restart(project string, force bool) error {
	f.rec("Restart", project, force)
	return nil
}

func (f *mmFake) PlanLoad(project string, plan queue.Plan) (control.QueueView, error) {
	f.rec("PlanLoad", project, plan)
	return control.QueueView{}, nil
}

func (f *mmFake) Continue(project string) error {
	f.rec("Continue", project)
	return nil
}

func (f *mmFake) StopCheck(project string) (*control.Stop, error) {
	f.rec("StopCheck", project)
	return nil, nil
}

func (f *mmFake) PutGithubToken(_ context.Context, project, token string) (control.TokenResult, error) {
	f.rec("PutGithubToken", project, token)
	return control.TokenResult{}, nil
}

func (f *mmFake) PhaseDone(project, sessionToken, phase, next string) error {
	f.rec("PhaseDone", project, sessionToken, phase, next)
	return nil
}

func (f *mmFake) WaitOperator(project, sessionToken, kind, reason, issueURL string) error {
	f.rec("WaitOperator", project, sessionToken, kind, reason, issueURL)
	return nil
}

func (f *mmFake) Milestone(project, sessionToken string, m control.Milestone) error {
	f.rec("Milestone", project, sessionToken, m)
	return nil
}

var _ control.Control = (*mmFake)(nil)

// mmHandler builds the mount of examples 1-7: demo is known, tok-1 is the API
// token, and demo has session token id-2.
func mmHandler(f *mmFake) http.Handler {
	known := func(p string) bool { return p == "demo" }
	tokens := Tokens{
		API: "tok-1",
		Session: func(project string) (string, bool) {
			if project == "demo" {
				return "id-2", true
			}
			return "", false
		},
	}
	return Handler(f, known, tokens)
}

// mmDo serves one request against h and returns the recorder.
func mmDo(h http.Handler, method, target, token, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, target, strings.NewReader(body))
	req.Header.Set("Accept", "application/json, text/event-stream")
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	if token != "" {
		req.Header.Set("Authorization", token)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

// mmRPC is the decoded JSON-RPC response the tests inspect.
type mmRPC struct {
	ID     int `json:"id"`
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
}

// mmDecode decodes a 200 body into mmRPC.
func mmDecode(t *testing.T, rec *httptest.ResponseRecorder) mmRPC {
	t.Helper()
	var rpc mmRPC
	if err := json.Unmarshal(rec.Body.Bytes(), &rpc); err != nil {
		t.Fatalf("decode body: %v\nbody: %s", err, rec.Body.String())
	}
	return rpc
}

const (
	mmListBody      = `{"jsonrpc":"2.0","id":1,"method":"tools/list","params":{}}`
	mmStatusBody    = `{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"status","arguments":{}}}`
	mmMilestoneBody = `{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"milestone","arguments":{"kind":"gate","headline":"P17 gate passed","numbers":"10 cards · $0.12"}}}`
)

const (
	mmUnauthorizedBody   = "{\"error\":{\"code\":\"unauthorized\",\"message\":\"bearer token required\"}}\n"
	mmUnknownProjectBody = "{\"error\":{\"code\":\"unknown_project\",\"message\":\"unknown project\"}}\n"
)

// mmDemoFake is the fake of example 1 with the status example 2 reads back.
func mmDemoFake() *mmFake {
	return &mmFake{status: control.Status{
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
}

// TestMCPMountExample1: tools/list on /mcp/demo names the ten PM tools.
func TestMCPMountExample1(t *testing.T) {
	f := mmDemoFake()
	h := mmHandler(f)

	rec := mmDo(h, http.MethodPost, "/mcp/demo", "Bearer tok-1", mmListBody)
	testhelp.Equal(t, "status", rec.Code, 200)
	if got := rec.Header().Get("Content-Type"); !strings.HasPrefix(got, "application/json") {
		t.Errorf("Content-Type: got %q, want prefix application/json", got)
	}

	rpc := mmDecode(t, rec)
	testhelp.Equal(t, "id", rpc.ID, 1)
	names := make([]string, 0, len(rpc.Result.Tools))
	for _, tool := range rpc.Result.Tools {
		names = append(names, tool.Name)
	}
	testhelp.Equal(t, "tools", names, []string{"answer", "continue", "interrupt", "order", "pending", "plan_load", "restart", "status", "stop_check", "usage"})
}

// TestMCPMountExample2: tools/call status renders the Status as one text content.
func TestMCPMountExample2(t *testing.T) {
	f := mmDemoFake()
	h := mmHandler(f)

	rec := mmDo(h, http.MethodPost, "/mcp/demo", "Bearer tok-1", mmStatusBody)
	testhelp.Equal(t, "status", rec.Code, 200)

	rpc := mmDecode(t, rec)
	if len(rpc.Result.Content) != 1 {
		t.Fatalf("content: got %d entries, want 1", len(rpc.Result.Content))
	}
	testhelp.Equal(t, "content[0].type", rpc.Result.Content[0].Type, "text")
	testhelp.Equal(t, "content[0].text", rpc.Result.Content[0].Text,
		`{"state":"busy","phase":"P18","minutes":37,"cost_usd":1.92,"five_hour":22,"seven_day":60,"session_id":"s-1","queue":{"state":"running","approved":"7b31dfe","index":1,"phases":3,"current":"P18","reason":""},"repo":{"checked":true,"reachable":true,"push":true,"reason":"","checked_at":"2026-10-08T15:00:00Z"}}`)
	testhelp.Equal(t, "isError", rpc.Result.IsError, false)
	testhelp.Equal(t, "calls", f.calls, []mmCall{{Method: "Status", Args: []any{"demo"}}})
}

// TestMCPMountExample3: the session mount routes milestone to the Control.
func TestMCPMountExample3(t *testing.T) {
	f := mmDemoFake()
	h := mmHandler(f)

	rec := mmDo(h, http.MethodPost, "/mcp/demo/session", "Bearer id-2", mmMilestoneBody)
	testhelp.Equal(t, "status", rec.Code, 200)

	rpc := mmDecode(t, rec)
	if len(rpc.Result.Content) != 1 {
		t.Fatalf("content: got %d entries, want 1", len(rpc.Result.Content))
	}
	testhelp.Equal(t, "content[0].text", rpc.Result.Content[0].Text, `{"ok":true}`)
	testhelp.Equal(t, "calls", f.calls, []mmCall{{
		Method: "Milestone",
		Args:   []any{"demo", "id-2", control.Milestone{Kind: "gate", Headline: "P17 gate passed", Numbers: "10 cards · $0.12"}},
	}})
}

// TestMCPMountExample4: a missing or mismatched bearer token is a 401.
func TestMCPMountExample4(t *testing.T) {
	f := mmDemoFake()
	h := mmHandler(f)

	cases := []struct {
		name   string
		target string
		token  string
	}{
		{"no token", "/mcp/demo", ""},
		{"wrong token", "/mcp/demo", "Bearer wrong"},
		{"api token at session", "/mcp/demo/session", "Bearer tok-1"},
		{"unknown session", "/mcp/other/session", "Bearer id-2"},
	}
	for _, tc := range cases {
		rec := mmDo(h, http.MethodPost, tc.target, tc.token, mmListBody)
		testhelp.Equal(t, tc.name+" status", rec.Code, 401)
		testhelp.Equal(t, tc.name+" content type", rec.Header().Get("Content-Type"), "application/json")
		testhelp.Equal(t, tc.name+" body", rec.Body.String(), mmUnauthorizedBody)
	}
	testhelp.Equal(t, "calls", len(f.calls), 0)
}

// TestMCPMountExample5: an unknown project is a 404, checked after the token.
func TestMCPMountExample5(t *testing.T) {
	f := mmDemoFake()
	h := mmHandler(f)

	rec := mmDo(h, http.MethodPost, "/mcp/nope", "Bearer tok-1", mmListBody)
	testhelp.Equal(t, "status with token", rec.Code, 404)
	testhelp.Equal(t, "content type with token", rec.Header().Get("Content-Type"), "application/json")
	testhelp.Equal(t, "body with token", rec.Body.String(), mmUnknownProjectBody)

	rec = mmDo(h, http.MethodPost, "/mcp/nope", "", mmListBody)
	testhelp.Equal(t, "status without token", rec.Code, 401)
	testhelp.Equal(t, "content type without token", rec.Header().Get("Content-Type"), "application/json")
	testhelp.Equal(t, "body without token", rec.Body.String(), mmUnauthorizedBody)

	testhelp.Equal(t, "calls", len(f.calls), 0)
}

// TestMCPMountExample6: /mcp exposes the user-level tools.
func TestMCPMountExample6(t *testing.T) {
	f := mmDemoFake()
	h := mmHandler(f)

	rec := mmDo(h, http.MethodPost, "/mcp", "Bearer tok-1", mmListBody)
	testhelp.Equal(t, "status", rec.Code, 200)

	rpc := mmDecode(t, rec)
	names := make([]string, 0, len(rpc.Result.Tools))
	for _, tool := range rpc.Result.Tools {
		names = append(names, tool.Name)
	}
	testhelp.Equal(t, "tools", names, []string{"project_create", "projects_list"})
}

// TestMCPMountExample7: GET and DELETE are not how the stateless mount is used.
func TestMCPMountExample7(t *testing.T) {
	f := mmDemoFake()
	h := mmHandler(f)

	rec := mmDo(h, http.MethodGet, "/mcp/demo", "Bearer tok-1", "")
	testhelp.Equal(t, "GET status", rec.Code, 405)

	rec = mmDo(h, http.MethodDelete, "/mcp", "", "")
	testhelp.Equal(t, "DELETE status", rec.Code, 405)
}
