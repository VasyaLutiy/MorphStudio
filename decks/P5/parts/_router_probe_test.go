package api

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"morphstudio/control"
	"morphstudio/internal/testhelp"
	"morphstudio/queue"
	"morphstudio/session"
)

// pRTFake implements control.Control; each call is recorded as "<Method> <project>".
type pRTFake struct {
	calls []string
}

func (f *pRTFake) rec(m, p string) { f.calls = append(f.calls, m+" "+p) }

func (f *pRTFake) Projects() []control.ProjectView { f.rec("Projects", ""); return nil }
func (f *pRTFake) CreateProject(ctx context.Context, name, language, repoURL string) (control.ProjectView, error) {
	f.rec("CreateProject", name)
	return control.ProjectView{Name: name}, nil
}
func (f *pRTFake) Status(project string) (control.Status, error) {
	f.rec("Status", project)
	if project != "demo" {
		return control.Status{}, control.ErrUnknownProject
	}
	return control.Status{State: "busy", Phase: "P18", Minutes: 37, SessionID: "s-1"}, nil
}
func (f *pRTFake) Events(project string, since int64, max int) (control.Events, error) {
	f.rec("Events", project)
	return control.Events{}, nil
}
func (f *pRTFake) Order(project, text string) (session.OrderResult, error) {
	f.rec("Order", project)
	return session.OrderResult{Sent: true}, nil
}
func (f *pRTFake) Pending(project string) (*control.Question, error) {
	f.rec("Pending", project)
	return nil, nil
}
func (f *pRTFake) Answer(project string, option int) error { f.rec("Answer", project); return nil }
func (f *pRTFake) Interrupt(project string) error          { f.rec("Interrupt", project); return nil }
func (f *pRTFake) Usage(project string) (control.Usage, error) {
	f.rec("Usage", project)
	return control.Usage{}, nil
}
func (f *pRTFake) Restart(project string, force bool) error { f.rec("Restart", project); return nil }
func (f *pRTFake) PlanLoad(project string, plan queue.Plan) (control.QueueView, error) {
	f.rec("PlanLoad", project)
	return control.QueueView{}, nil
}
func (f *pRTFake) Continue(project string) error { f.rec("Continue", project); return nil }
func (f *pRTFake) StopCheck(project string) (*control.Stop, error) {
	f.rec("StopCheck", project)
	return nil, nil
}
func (f *pRTFake) PutGithubToken(ctx context.Context, project, token string) (control.TokenResult, error) {
	f.rec("PutGithubToken", project)
	return control.TokenResult{}, nil
}
func (f *pRTFake) PhaseDone(project, sessionToken, phase, next string) error {
	f.rec("PhaseDone", project)
	return nil
}
func (f *pRTFake) WaitOperator(project, sessionToken, kind, reason, issueURL string) error {
	f.rec("WaitOperator", project)
	return nil
}
func (f *pRTFake) Milestone(project, sessionToken string, m control.Milestone) error {
	f.rec("Milestone", project)
	return nil
}

var _ control.Control = (*pRTFake)(nil)

type pRTResp struct {
	Code        int
	ContentType string
	Body        string
}

func pRTServe(h http.Handler, method, target, auth, body string) pRTResp {
	var rd io.Reader
	if body != "" {
		rd = strings.NewReader(body)
	}
	req := httptest.NewRequest(method, target, rd)
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	if auth != "" {
		req.Header.Set("Authorization", auth)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return pRTResp{rec.Code, rec.Header().Get("Content-Type"), rec.Body.String()}
}

// pRTRouter is NewRouter, or a 501 handler (a readable red) when it returns nil.
func pRTRouter(h Handlers, token string, mcp http.Handler) http.Handler {
	r := NewRouter(h, token, mcp)
	if r == nil {
		return http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(501) })
	}
	return r
}

const pRT401 = `{"error":{"code":"unauthorized","message":"bearer token required"}}` + "\n"
const pRTStatusJSON = `{"state":"busy","phase":"P18","minutes":37,"cost_usd":0,"five_hour":0,"seven_day":0,"session_id":"s-1","queue":{"state":"","approved":"","index":0,"phases":0,"current":"","reason":""},"repo":{"checked":false,"reachable":false,"push":false,"reason":"","checked_at":"0001-01-01T00:00:00Z"}}` + "\n"

func TestProbeRouterExample1(t *testing.T) {
	r := pRTRouter(Handlers{Control: &pRTFake{}}, "tok-1", nil)
	testhelp.Equal(t, "example 1 GET /healthz", pRTServe(r, "GET", "/healthz", "", ""), pRTResp{200, "application/json", `{"status":"ok"}` + "\n"})
	// variant: another token; healthz with a wrong token is still open
	v := pRTRouter(Handlers{Control: &pRTFake{}}, "other-7", nil)
	testhelp.Equal(t, "example 1 variant GET /healthz with a wrong token", pRTServe(v, "GET", "/healthz", "Bearer nope", ""), pRTResp{200, "application/json", `{"status":"ok"}` + "\n"})
}

func TestProbeRouterExample2(t *testing.T) {
	f := &pRTFake{}
	r := pRTRouter(Handlers{Control: f}, "tok-1", nil)
	for _, auth := range []string{"", "Bearer wrong", "Basic dG9rLTE=", "Bearer tok-1 "} {
		testhelp.Equal(t, fmt.Sprintf("example 2 Authorization %q", auth), pRTServe(r, "GET", "/projects/demo/status", auth, ""), pRTResp{401, "application/json", pRT401})
	}
	testhelp.Equal(t, "example 2 the Control saw nothing", len(f.calls), 0)
	// variant: a prefix of the token, the token without a scheme, two spaces
	for _, auth := range []string{"Bearer tok-", "tok-1", "Bearer  tok-1", "Bearertok-1", "Token tok-1"} {
		testhelp.Equal(t, fmt.Sprintf("example 2 variant Authorization %q", auth), pRTServe(r, "GET", "/projects/demo/status", auth, ""), pRTResp{401, "application/json", pRT401})
	}
	// variant: Bearer alone, in front of any handler
	next := http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(204) })
	testhelp.Equal(t, "example 2 variant Bearer(\"s3\") with \"Bearer s3\"", pRTServe(Bearer("s3", next), "GET", "/x", "Bearer s3", "").Code, 204)
	testhelp.Equal(t, "example 2 variant Bearer(\"s3\") with \"Bearer s4\"", pRTServe(Bearer("s3", next), "GET", "/x", "Bearer s4", ""), pRTResp{401, "application/json", pRT401})
}

func TestProbeRouterExample3(t *testing.T) {
	f := &pRTFake{}
	r := pRTRouter(Handlers{Control: f}, "tok-1", nil)
	testhelp.Equal(t, "example 3 \"Bearer tok-1\"", pRTServe(r, "GET", "/projects/demo/status", "Bearer tok-1", ""), pRTResp{200, "application/json", pRTStatusJSON})
	testhelp.Equal(t, "example 3 \"bearer tok-1\"", pRTServe(r, "GET", "/projects/demo/status", "bearer tok-1", ""), pRTResp{200, "application/json", pRTStatusJSON})
	testhelp.Equal(t, "example 3 GET /projects/nope/status", pRTServe(r, "GET", "/projects/nope/status", "Bearer tok-1", "").Code, 404)
	testhelp.Equal(t, "example 3 calls", f.calls, []string{"Status demo", "Status demo", "Status nope"})
	// variant: "BEARER" scheme
	testhelp.Equal(t, "example 3 variant \"BEARER tok-1\"", pRTServe(r, "GET", "/projects/demo/status", "BEARER tok-1", "").Code, 200)
}

func TestProbeRouterExample4(t *testing.T) {
	f := &pRTFake{}
	r := pRTRouter(Handlers{Control: f}, "", nil)
	testhelp.Equal(t, "example 4 empty token, \"Bearer \"", pRTServe(r, "GET", "/projects", "Bearer ", ""), pRTResp{401, "application/json", pRT401})
	testhelp.Equal(t, "example 4 empty token, no header", pRTServe(r, "GET", "/projects", "", ""), pRTResp{401, "application/json", pRT401})
	testhelp.Equal(t, "example 4 the Control saw nothing", len(f.calls), 0)
	testhelp.Equal(t, "example 4 variant Bearer(\"\") alone", pRTServe(Bearer("", http.NotFoundHandler()), "GET", "/x", "Bearer ", "").Code, 401)
}

func TestProbeRouterExample5(t *testing.T) {
	f := &pRTFake{}
	r := pRTRouter(Handlers{Control: f}, "tok-1", nil)
	reqs := []struct{ method, target, body string }{
		{"POST", "/projects/demo/messages", `{"text":"hi"}`},
		{"GET", "/projects/demo/pending", ""},
		{"POST", "/projects/demo/interrupt", ""},
		{"GET", "/projects/demo/usage", ""},
		{"POST", "/projects/demo/continue", ""},
		{"GET", "/projects/demo/stop", ""},
		{"PUT", "/projects/demo/github-token", `{"token":"t"}`},
	}
	codes := []int{}
	for _, q := range reqs {
		codes = append(codes, pRTServe(r, q.method, q.target, "Bearer tok-1", q.body).Code)
	}
	testhelp.Equal(t, "example 5 statuses", codes, []int{200, 200, 200, 200, 200, 200, 200})
	testhelp.Equal(t, "example 5 calls", f.calls, []string{"Order demo", "Pending demo", "Interrupt demo", "Usage demo", "Continue demo", "StopCheck demo", "PutGithubToken demo"})
	// variant: the other routes, another project, and each one behind the token
	g := &pRTFake{}
	r = pRTRouter(Handlers{Control: g}, "tok-1", nil)
	more := []struct{ method, target, body string }{
		{"GET", "/projects", ""},
		{"POST", "/projects", `{"name":"zeta","language":"go","repo_url":"u"}`},
		{"GET", "/projects/beta/events?since=1", ""},
		{"POST", "/projects/beta/answer", `{"option":1}`},
		{"POST", "/projects/beta/restart", ""},
		{"POST", "/projects/beta/plan", `{"approved":"a","phases":[{"id":"P1"}]}`},
		{"GET", "/projects/beta/status", ""},
	}
	codes = []int{}
	for _, q := range more {
		codes = append(codes, pRTServe(r, q.method, q.target, "Bearer tok-1", q.body).Code)
	}
	testhelp.Equal(t, "example 5 variant statuses", codes, []int{200, 201, 200, 200, 200, 200, 404})
	testhelp.Equal(t, "example 5 variant calls", g.calls, []string{"Projects ", "CreateProject zeta", "Events beta", "Answer beta", "Restart beta", "PlanLoad beta", "Status beta"})
	g.calls = nil
	all := append(append([]struct{ method, target, body string }{}, reqs...), more...)
	codes = []int{}
	for _, q := range all {
		codes = append(codes, pRTServe(r, q.method, q.target, "", q.body).Code)
	}
	testhelp.Equal(t, "example 5 variant every route without a token", codes, []int{401, 401, 401, 401, 401, 401, 401, 401, 401, 401, 401, 401, 401, 401})
	testhelp.Equal(t, "example 5 variant no call without a token", len(g.calls), 0)
}

func TestProbeRouterExample6(t *testing.T) {
	r := pRTRouter(Handlers{Control: &pRTFake{}}, "tok-1", nil)
	reqs := [][2]string{{"DELETE", "/projects/demo/status"}, {"POST", "/projects/demo/status"}, {"GET", "/projects/demo/messages"}, {"GET", "/nowhere"}, {"POST", "/healthz"}}
	codes := []int{}
	for _, q := range reqs {
		codes = append(codes, pRTServe(r, q[0], q[1], "Bearer tok-1", "").Code)
	}
	testhelp.Equal(t, "example 6 statuses", codes, []int{405, 405, 405, 404, 405})
	// variant: other wrong methods and paths
	reqs = [][2]string{{"PUT", "/projects"}, {"GET", "/projects/demo/github-token"}, {"GET", "/projects/demo/plan"}, {"GET", "/projects/demo"}, {"GET", "/mcp"}}
	codes = []int{}
	for _, q := range reqs {
		codes = append(codes, pRTServe(r, q[0], q[1], "Bearer tok-1", "").Code)
	}
	testhelp.Equal(t, "example 6 variant statuses", codes, []int{405, 405, 405, 404, 404})
}

func TestProbeRouterExample7(t *testing.T) {
	mcp := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(299)
		io.WriteString(w, "mcp "+r.URL.Path)
	})
	r := pRTRouter(Handlers{Control: &pRTFake{}}, "tok-1", mcp)
	got := []pRTResp{pRTServe(r, "GET", "/mcp", "", ""), pRTServe(r, "POST", "/mcp/demo", "", ""), pRTServe(r, "POST", "/mcp/demo/session", "", "")}
	bodies := []string{}
	codes := []int{}
	for _, g := range got {
		codes = append(codes, g.Code)
		bodies = append(bodies, g.Body)
	}
	testhelp.Equal(t, "example 7 mcp statuses", codes, []int{299, 299, 299})
	testhelp.Equal(t, "example 7 mcp bodies", bodies, []string{"mcp /mcp", "mcp /mcp/demo", "mcp /mcp/demo/session"})
	n := pRTRouter(Handlers{Control: &pRTFake{}}, "tok-1", nil)
	codes = []int{pRTServe(n, "GET", "/mcp", "", "").Code, pRTServe(n, "POST", "/mcp/demo", "", "").Code, pRTServe(n, "POST", "/mcp/demo/session", "", "").Code}
	testhelp.Equal(t, "example 7 mcp nil statuses", codes, []int{404, 404, 404})
	// variant: the mount takes any method and a token header is passed through untouched
	req := httptest.NewRequest("DELETE", "/mcp/x", nil)
	req.Header.Set("Authorization", "Bearer session-tok")
	seen := ""
	r = pRTRouter(Handlers{Control: &pRTFake{}}, "tok-1", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seen = r.Method + " " + r.Header.Get("Authorization")
		w.WriteHeader(298)
	}))
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	testhelp.Equal(t, "example 7 variant DELETE /mcp/x status", rec.Code, 298)
	testhelp.Equal(t, "example 7 variant the mount saw", seen, "DELETE Bearer session-tok")
}
