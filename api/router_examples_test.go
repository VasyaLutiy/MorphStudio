package api

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"morphstudio/control"
	"morphstudio/internal/testhelp"
	"morphstudio/queue"
	"morphstudio/session"
)

type rtCall struct {
	Method  string
	Project string
}

type rtFake struct {
	calls []rtCall
}

func (f *rtFake) record(method, project string) {
	f.calls = append(f.calls, rtCall{Method: method, Project: project})
}

func (f *rtFake) Projects() []control.ProjectView {
	f.record("Projects", "")
	return nil
}

func (f *rtFake) CreateProject(ctx context.Context, name, language, repoURL string) (control.ProjectView, error) {
	f.record("CreateProject", name)
	return control.ProjectView{}, nil
}

func (f *rtFake) Status(project string) (control.Status, error) {
	f.record("Status", project)
	if project != "demo" {
		return control.Status{}, control.ErrUnknownProject
	}
	return control.Status{State: "busy", Phase: "P18", Minutes: 37, SessionID: "s-1"}, nil
}

func (f *rtFake) Events(project string, since int64, max int) (control.Events, error) {
	f.record("Events", project)
	return control.Events{}, nil
}

func (f *rtFake) Order(project, text string) (session.OrderResult, error) {
	f.record("Order", project)
	return session.OrderResult{}, nil
}

func (f *rtFake) Pending(project string) (*control.Question, error) {
	f.record("Pending", project)
	return nil, nil
}

func (f *rtFake) Answer(project string, option int) error {
	f.record("Answer", project)
	return nil
}

func (f *rtFake) Interrupt(project string) error {
	f.record("Interrupt", project)
	return nil
}

func (f *rtFake) Usage(project string) (control.Usage, error) {
	f.record("Usage", project)
	return control.Usage{}, nil
}

func (f *rtFake) Restart(project string, force bool) error {
	f.record("Restart", project)
	return nil
}

func (f *rtFake) PlanLoad(project string, plan queue.Plan) (control.QueueView, error) {
	f.record("PlanLoad", project)
	return control.QueueView{}, nil
}

func (f *rtFake) Continue(project string) error {
	f.record("Continue", project)
	return nil
}

func (f *rtFake) StopCheck(project string) (*control.Stop, error) {
	f.record("StopCheck", project)
	return nil, nil
}

func (f *rtFake) PutGithubToken(ctx context.Context, project, token string) (control.TokenResult, error) {
	f.record("PutGithubToken", project)
	return control.TokenResult{}, nil
}

func (f *rtFake) PhaseDone(project, sessionToken, phase, next string) error {
	f.record("PhaseDone", project)
	return nil
}

func (f *rtFake) WaitOperator(project, sessionToken, kind, reason, issueURL string) error {
	f.record("WaitOperator", project)
	return nil
}

func (f *rtFake) Milestone(project, sessionToken string, m control.Milestone) error {
	f.record("Milestone", project)
	return nil
}

func TestRouterExample1(t *testing.T) {
	f := &rtFake{}
	r := NewRouter(Handlers{Control: f}, "tok-1", nil)

	req := httptest.NewRequest("GET", "/healthz", nil)
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	testhelp.Equal(t, "status", rec.Code, 200)
	testhelp.Equal(t, "Content-Type", rec.Header().Get("Content-Type"), "application/json")
	testhelp.Equal(t, "body", rec.Body.String(),
		`{"status":"ok"}`+"\n")
}

func TestRouterExample2(t *testing.T) {
	f := &rtFake{}
	r := NewRouter(Handlers{Control: f}, "tok-1", nil)

	headers := []string{"", "Bearer wrong", "Basic dG9rLTE=", "Bearer tok-1 "}
	for _, h := range headers {
		req := httptest.NewRequest("GET", "/projects/demo/status", nil)
		if h != "" {
			req.Header.Set("Authorization", h)
		}
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)
		testhelp.Equal(t, "status "+h, rec.Code, 401)
		testhelp.Equal(t, "body "+h, rec.Body.String(),
			`{"error":{"code":"unauthorized","message":"bearer token required"}}`+"\n")
	}
}

func TestRouterExample3(t *testing.T) {
	f := &rtFake{}
	r := NewRouter(Handlers{Control: f}, "tok-1", nil)

	body := `{"state":"busy","phase":"P18","minutes":37,"cost_usd":0,"five_hour":0,"seven_day":0,"session_id":"s-1","queue":{"state":"","approved":"","index":0,"phases":0,"current":"","reason":""},"repo":{"checked":false,"reachable":false,"push":false,"reason":"","checked_at":"0001-01-01T00:00:00Z"}}` + "\n"

	for _, h := range []string{"Bearer tok-1", "bearer tok-1"} {
		req := httptest.NewRequest("GET", "/projects/demo/status", nil)
		req.Header.Set("Authorization", h)
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)
		testhelp.Equal(t, "status "+h, rec.Code, 200)
		testhelp.Equal(t, "body "+h, rec.Body.String(), body)
	}

	req := httptest.NewRequest("GET", "/projects/nope/status", nil)
	req.Header.Set("Authorization", "Bearer tok-1")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	testhelp.Equal(t, "status nope", rec.Code, 404)
}

func TestRouterExample4(t *testing.T) {
	f := &rtFake{}
	r := NewRouter(Handlers{Control: f}, "", nil)

	req := httptest.NewRequest("GET", "/projects", nil)
	req.Header.Set("Authorization", "Bearer ")
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)

	testhelp.Equal(t, "status", rec.Code, 401)
	testhelp.Equal(t, "body", rec.Body.String(),
		`{"error":{"code":"unauthorized","message":"bearer token required"}}`+"\n")
}

func TestRouterExample5(t *testing.T) {
	f := &rtFake{}
	r := NewRouter(Handlers{Control: f}, "tok-1", nil)

	post := func(target, body string) int {
		req := httptest.NewRequest("POST", target, strings.NewReader(body))
		req.Header.Set("Authorization", "Bearer tok-1")
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)
		return rec.Code
	}
	get := func(target string) int {
		req := httptest.NewRequest("GET", target, nil)
		req.Header.Set("Authorization", "Bearer tok-1")
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)
		return rec.Code
	}
	put := func(target, body string) int {
		req := httptest.NewRequest("PUT", target, strings.NewReader(body))
		req.Header.Set("Authorization", "Bearer tok-1")
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)
		return rec.Code
	}

	got := []int{
		post("/projects/demo/messages", `{"text":"hi"}`),
		get("/projects/demo/pending"),
		post("/projects/demo/interrupt", ""),
		get("/projects/demo/usage"),
		post("/projects/demo/continue", ""),
		get("/projects/demo/stop"),
		put("/projects/demo/github-token", `{"token":"t"}`),
	}
	testhelp.Equal(t, "statuses", got, []int{200, 200, 200, 200, 200, 200, 200})

	want := []rtCall{
		{Method: "Order", Project: "demo"},
		{Method: "Pending", Project: "demo"},
		{Method: "Interrupt", Project: "demo"},
		{Method: "Usage", Project: "demo"},
		{Method: "Continue", Project: "demo"},
		{Method: "StopCheck", Project: "demo"},
		{Method: "PutGithubToken", Project: "demo"},
	}
	testhelp.Equal(t, "calls", f.calls, want)
}

func TestRouterExample6(t *testing.T) {
	f := &rtFake{}
	r := NewRouter(Handlers{Control: f}, "tok-1", nil)

	cases := []struct {
		method string
		target string
		want   int
	}{
		{"DELETE", "/projects/demo/status", 405},
		{"POST", "/projects/demo/status", 405},
		{"GET", "/projects/demo/messages", 405},
		{"GET", "/nowhere", 404},
		{"POST", "/healthz", 405},
	}
	for _, c := range cases {
		req := httptest.NewRequest(c.method, c.target, nil)
		req.Header.Set("Authorization", "Bearer tok-1")
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)
		testhelp.Equal(t, c.method+" "+c.target, rec.Code, c.want)
	}
}

func TestRouterExample7(t *testing.T) {
	f := &rtFake{}
	mcp := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(299)
		_, _ = w.Write([]byte("mcp " + r.URL.Path))
	})
	r := NewRouter(Handlers{Control: f}, "tok-1", mcp)

	cases := []struct {
		method string
		target string
		want   string
	}{
		{"GET", "/mcp", "mcp /mcp"},
		{"POST", "/mcp/demo", "mcp /mcp/demo"},
		{"POST", "/mcp/demo/session", "mcp /mcp/demo/session"},
	}
	for _, c := range cases {
		req := httptest.NewRequest(c.method, c.target, nil)
		rec := httptest.NewRecorder()
		r.ServeHTTP(rec, req)
		testhelp.Equal(t, "status "+c.target, rec.Code, 299)
		testhelp.Equal(t, "body "+c.target, rec.Body.String(), c.want)
	}

	r2 := NewRouter(Handlers{Control: f}, "tok-1", nil)
	for _, c := range cases {
		req := httptest.NewRequest(c.method, c.target, nil)
		rec := httptest.NewRecorder()
		r2.ServeHTTP(rec, req)
		testhelp.Equal(t, "status "+c.target, rec.Code, 404)
	}
}
