package api

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
	"time"

	"morphstudio/control"
	"morphstudio/eventlog"
	"morphstudio/github"
	"morphstudio/internal/testhelp"
	"morphstudio/queue"
	"morphstudio/session"
)

// pHHFake implements control.Control; every call is recorded as one line.
type pHHFake struct {
	calls    []string
	projects []control.ProjectView
	view     control.ProjectView
	status   control.Status
	statusOf string
	err      error
	events   control.Events
	order    session.OrderResult
	question *control.Question
	usage    control.Usage
	queue    control.QueueView
	plan     queue.Plan
	stop     *control.Stop
	token    control.TokenResult
}

func (f *pHHFake) rec(format string, a ...any) { f.calls = append(f.calls, fmt.Sprintf(format, a...)) }

func (f *pHHFake) Projects() []control.ProjectView { f.rec("Projects"); return f.projects }
func (f *pHHFake) CreateProject(ctx context.Context, name, language, repoURL string) (control.ProjectView, error) {
	f.rec("CreateProject %s %s %s ctx=%t", name, language, repoURL, ctx != nil)
	return f.view, f.err
}
func (f *pHHFake) Status(project string) (control.Status, error) {
	f.rec("Status %s", project)
	if f.err != nil {
		return control.Status{}, f.err
	}
	if project != f.statusOf {
		return control.Status{}, control.ErrUnknownProject
	}
	return f.status, nil
}
func (f *pHHFake) Events(project string, since int64, max int) (control.Events, error) {
	f.rec("Events %s %d %d", project, since, max)
	return f.events, f.err
}
func (f *pHHFake) Order(project, text string) (session.OrderResult, error) {
	f.rec("Order %s %q", project, text)
	return f.order, f.err
}
func (f *pHHFake) Pending(project string) (*control.Question, error) {
	f.rec("Pending %s", project)
	return f.question, f.err
}
func (f *pHHFake) Answer(project string, option int) error {
	f.rec("Answer %s %d", project, option)
	return f.err
}
func (f *pHHFake) Interrupt(project string) error { f.rec("Interrupt %s", project); return f.err }
func (f *pHHFake) Usage(project string) (control.Usage, error) {
	f.rec("Usage %s", project)
	return f.usage, f.err
}
func (f *pHHFake) Restart(project string, force bool) error {
	f.rec("Restart %s %t", project, force)
	return f.err
}
func (f *pHHFake) PlanLoad(project string, plan queue.Plan) (control.QueueView, error) {
	f.rec("PlanLoad %s", project)
	f.plan = plan
	return f.queue, f.err
}
func (f *pHHFake) Continue(project string) error { f.rec("Continue %s", project); return f.err }
func (f *pHHFake) StopCheck(project string) (*control.Stop, error) {
	f.rec("StopCheck %s", project)
	return f.stop, f.err
}
func (f *pHHFake) PutGithubToken(ctx context.Context, project, token string) (control.TokenResult, error) {
	f.rec("PutGithubToken %s %s", project, token)
	return f.token, f.err
}
func (f *pHHFake) PhaseDone(project, sessionToken, phase, next string) error {
	f.rec("PhaseDone %s", project)
	return f.err
}
func (f *pHHFake) WaitOperator(project, sessionToken, kind, reason, issueURL string) error {
	f.rec("WaitOperator %s", project)
	return f.err
}
func (f *pHHFake) Milestone(project, sessionToken string, m control.Milestone) error {
	f.rec("Milestone %s", project)
	return f.err
}

var _ control.Control = (*pHHFake)(nil)

type pHHResp struct {
	Code        int
	ContentType string
	Body        string
}

// pHHServe runs one handler on a recorder; body "" with ctype "" sends no body and no Content-Type.
func pHHServe(handler func(http.ResponseWriter, *http.Request), method, target, project, ctype, body string) pHHResp {
	var rd io.Reader
	if body != "" {
		rd = strings.NewReader(body)
	}
	req := httptest.NewRequest(method, target, rd)
	if ctype != "" {
		req.Header.Set("Content-Type", ctype)
	}
	req.SetPathValue("project", project)
	rec := httptest.NewRecorder()
	handler(rec, req)
	return pHHResp{rec.Code, rec.Header().Get("Content-Type"), rec.Body.String()}
}

const pHHJSON = "application/json"

var pHHStatus1 = control.Status{State: "busy", Phase: "P18", Minutes: 37, CostUSD: 1.92, FiveHour: 22, SevenDay: 60, SessionID: "s-1",
	Queue: control.QueueView{State: "running", Approved: "7b31dfe", Index: 1, Phases: 3, Current: "P18"},
	Repo:  github.Access{Checked: true, Reachable: true, Push: true, CheckedAt: time.Date(2026, 10, 8, 15, 0, 0, 0, time.UTC)}}

const pHHStatus1JSON = `{"state":"busy","phase":"P18","minutes":37,"cost_usd":1.92,"five_hour":22,"seven_day":60,"session_id":"s-1","queue":{"state":"running","approved":"7b31dfe","index":1,"phases":3,"current":"P18","reason":""},"repo":{"checked":true,"reachable":true,"push":true,"reason":"","checked_at":"2026-10-08T15:00:00Z"}}` + "\n"

const pHH415 = `{"error":{"code":"unsupported_media_type","message":"Content-Type must be application/json"}}` + "\n"
const pHHBadJSON = `{"error":{"code":"bad_json","message":"bad json"}}` + "\n"
const pHHOK = `{"ok":true}` + "\n"

func TestProbeHTTPHandlersExample1(t *testing.T) {
	f := &pHHFake{status: pHHStatus1, statusOf: "demo"}
	h := Handlers{Control: f}
	testhelp.Equal(t, "example 1 GET status demo", pHHServe(h.Status, "GET", "/projects/demo/status", "demo", "", ""), pHHResp{200, "application/json", pHHStatus1JSON})
	testhelp.Equal(t, "example 1 GET status nope", pHHServe(h.Status, "GET", "/projects/nope/status", "nope", "", ""), pHHResp{404, "application/json", `{"error":{"code":"unknown_project","message":"unknown project"}}` + "\n"})
	testhelp.Equal(t, "example 1 calls", f.calls, []string{"Status demo", "Status nope"})
	// variant: WriteJSON and WriteError directly; a Status with a Stop
	rec := httptest.NewRecorder()
	WriteJSON(rec, 202, map[string]int{"n": 1})
	testhelp.Equal(t, "example 1 variant WriteJSON", pHHResp{rec.Code, rec.Header().Get("Content-Type"), rec.Body.String()}, pHHResp{202, "application/json", `{"n":1}` + "\n"})
	rec = httptest.NewRecorder()
	WriteError(rec, fmt.Errorf("restart: %w", control.ErrBusy))
	testhelp.Equal(t, "example 1 variant WriteError(wrapped ErrBusy)", pHHResp{rec.Code, rec.Header().Get("Content-Type"), rec.Body.String()}, pHHResp{409, "application/json", `{"error":{"code":"busy","message":"restart: session is busy"}}` + "\n"})
	g := &pHHFake{statusOf: "p2", status: control.Status{State: "waiting", Stop: &control.Stop{Kind: "gate", Phase: "P3", Reason: "r", At: time.Date(2027, 1, 2, 3, 4, 5, 0, time.UTC)}}}
	got := pHHServe(Handlers{Control: g}.Status, "GET", "/projects/p2/status", "p2", "", "")
	testhelp.Equal(t, "example 1 variant status with a Stop (body end)", strings.HasSuffix(got.Body, `"stop":{"kind":"gate","phase":"P3","reason":"r","at":"2027-01-02T03:04:05Z"}}`+"\n"), true)
}

func TestProbeHTTPHandlersExample2(t *testing.T) {
	f := &pHHFake{order: session.OrderResult{Queued: 1}}
	h := Handlers{Control: f}
	got := pHHServe(h.Messages, "POST", "/projects/demo/messages", "demo", "application/json; charset=utf-8", `{"text":"smoke checked green. Resume with P12b"}`)
	testhelp.Equal(t, "example 2 POST messages (queued)", got, pHHResp{200, pHHJSON, `{"queued":1}` + "\n"})
	testhelp.Equal(t, "example 2 calls", f.calls, []string{`Order demo "smoke checked green. Resume with P12b"`})
	f.order = session.OrderResult{Sent: true}
	got = pHHServe(h.Messages, "POST", "/projects/demo/messages", "demo", pHHJSON, `{"text":"go"}`)
	testhelp.Equal(t, "example 2 POST messages (sent)", got, pHHResp{200, pHHJSON, `{"sent":true}` + "\n"})
	// variant: another project, a Control error, a trailing newline after the object
	f.calls = nil
	f.order = session.OrderResult{Queued: 4}
	got = pHHServe(h.Messages, "POST", "/projects/beta/messages", "beta", "Application/JSON", "{\"text\":\"x y\"}\n")
	testhelp.Equal(t, "example 2 variant beta, trailing newline", got, pHHResp{200, pHHJSON, `{"queued":4}` + "\n"})
	testhelp.Equal(t, "example 2 variant calls", f.calls, []string{`Order beta "x y"`})
	e := &pHHFake{err: control.ErrNoSession}
	got = pHHServe(Handlers{Control: e}.Messages, "POST", "/projects/demo/messages", "demo", pHHJSON, `{"text":"x"}`)
	testhelp.Equal(t, "example 2 variant ErrNoSession", got, pHHResp{409, pHHJSON, `{"error":{"code":"no_session","message":"no session running"}}` + "\n"})
}

func TestProbeHTTPHandlersExample3(t *testing.T) {
	f := &pHHFake{}
	h := Handlers{Control: f}
	for _, ct := range []string{"text/plain", "", "application/xml"} {
		got := pHHServe(h.Messages, "POST", "/projects/demo/messages", "demo", ct, `{"text":"x"}`)
		testhelp.Equal(t, fmt.Sprintf("example 3 Content-Type %q", ct), got, pHHResp{415, pHHJSON, pHH415})
	}
	testhelp.Equal(t, "example 3 the fake saw no Order", len(f.calls), 0)
	// variant: every other body-reading handler refuses a wrong Content-Type the same way
	for name, hf := range map[string]func(http.ResponseWriter, *http.Request){"CreateProject": h.CreateProject, "Answer": h.Answer, "Plan": h.Plan, "GithubToken": h.GithubToken, "Restart": h.Restart} {
		got := pHHServe(hf, "POST", "/x", "demo", "text/plain", `{}`)
		testhelp.Equal(t, "example 3 variant "+name+" text/plain", got, pHHResp{415, pHHJSON, pHH415})
	}
	testhelp.Equal(t, "example 3 variant no calls", len(f.calls), 0)
}

func TestProbeHTTPHandlersExample4(t *testing.T) {
	f := &pHHFake{}
	h := Handlers{Control: f}
	for _, body := range []string{`{"option": `, `{"option":1,"x":2}`, `{"option":1} {}`, ``} {
		got := pHHServe(h.Answer, "POST", "/projects/demo/answer", "demo", pHHJSON, body)
		testhelp.Equal(t, fmt.Sprintf("example 4 body %q", body), got, pHHResp{400, pHHJSON, pHHBadJSON})
	}
	testhelp.Equal(t, "example 4 no Answer on bad json", len(f.calls), 0)
	got := pHHServe(h.Answer, "POST", "/projects/demo/answer", "demo", pHHJSON, `{"option":2}`)
	testhelp.Equal(t, "example 4 option 2", got, pHHResp{200, pHHJSON, pHHOK})
	testhelp.Equal(t, "example 4 calls", f.calls, []string{"Answer demo 2"})
	e := &pHHFake{err: fmt.Errorf("%w: 3 of 1..2", control.ErrBadOption)}
	got = pHHServe(Handlers{Control: e}.Answer, "POST", "/projects/demo/answer", "demo", pHHJSON, `{"option":3}`)
	testhelp.Equal(t, "example 4 ErrBadOption", got, pHHResp{400, pHHJSON, `{"error":{"code":"bad_option","message":"option out of range: 3 of 1..2"}}` + "\n"})
	// variant: a body over 1 MiB; a wrong type
	big := `{"text":"` + strings.Repeat("x", 1<<20) + `"}`
	m := &pHHFake{order: session.OrderResult{Sent: true}}
	got = pHHServe(Handlers{Control: m}.Messages, "POST", "/projects/demo/messages", "demo", pHHJSON, big)
	testhelp.Equal(t, "example 4 variant a text over 1 MiB", got, pHHResp{400, pHHJSON, pHHBadJSON})
	testhelp.Equal(t, "example 4 variant no Order for a body over 1 MiB", len(m.calls), 0)
	got = pHHServe(h.Answer, "POST", "/projects/demo/answer", "demo", pHHJSON, `{"option":"1"}`)
	testhelp.Equal(t, "example 4 variant option as a string", got, pHHResp{400, pHHJSON, pHHBadJSON})
	ok := `{"text":"` + strings.Repeat("y", 1<<19) + `"}`
	got = pHHServe(Handlers{Control: &pHHFake{order: session.OrderResult{Sent: true}}}.Messages, "POST", "/projects/demo/messages", "demo", pHHJSON, ok)
	testhelp.Equal(t, "example 4 variant a 512 KiB text is accepted", got, pHHResp{200, pHHJSON, `{"sent":true}` + "\n"})
	testhelp.Equal(t, "example 4 variant calls", f.calls, []string{"Answer demo 2"})
}

func TestProbeHTTPHandlersExample5(t *testing.T) {
	f := &pHHFake{events: control.Events{Entries: []eventlog.Entry{}, Last: 7}}
	h := Handlers{Control: f}
	got := pHHServe(h.Events, "GET", "/projects/demo/events?since=2&max=5", "demo", "", "")
	testhelp.Equal(t, "example 5 since=2&max=5", got, pHHResp{200, pHHJSON, `{"entries":[],"last":7}` + "\n"})
	got = pHHServe(h.Events, "GET", "/projects/demo/events?since=x", "demo", "", "")
	testhelp.Equal(t, "example 5 since=x", got, pHHResp{400, pHHJSON, `{"error":{"code":"bad_input","message":"since must be an integer"}}` + "\n"})
	got = pHHServe(h.Events, "GET", "/projects/demo/events", "demo", "", "")
	testhelp.Equal(t, "example 5 no query", got.Code, 200)
	testhelp.Equal(t, "example 5 calls", f.calls, []string{"Events demo 2 5", "Events demo 0 0"})
	// variant: max not an integer; a large since; entries passed through
	f.calls = nil
	got = pHHServe(h.Events, "GET", "/projects/demo/events?since=3&max=ten", "demo", "", "")
	testhelp.Equal(t, "example 5 variant max=ten", got, pHHResp{400, pHHJSON, `{"error":{"code":"bad_input","message":"max must be an integer"}}` + "\n"})
	f.events = control.Events{Entries: []eventlog.Entry{{Seq: 9000000001, T: 2.5, Dir: "in", Msg: []byte(`{"a":1}`)}}, Last: 9000000001}
	got = pHHServe(h.Events, "GET", "/projects/zeta/events?since=9000000000", "zeta", "", "")
	testhelp.Equal(t, "example 5 variant since=9000000000", got, pHHResp{200, pHHJSON, `{"entries":[{"seq":9000000001,"t":2.5,"dir":"in","msg":{"a":1}}],"last":9000000001}` + "\n"})
	testhelp.Equal(t, "example 5 variant calls", f.calls, []string{"Events zeta 9000000000 0"})
	e := &pHHFake{err: control.ErrUnknownProject}
	got = pHHServe(Handlers{Control: e}.Events, "GET", "/projects/q/events", "q", "", "")
	testhelp.Equal(t, "example 5 variant unknown project", got.Code, 404)
}

func TestProbeHTTPHandlersExample6(t *testing.T) {
	f := &pHHFake{}
	h := Handlers{Control: f}
	testhelp.Equal(t, "example 6 pending nil", pHHServe(h.Pending, "GET", "/projects/demo/pending", "demo", "", ""), pHHResp{200, pHHJSON, `{"question":null}` + "\n"})
	f.question = &control.Question{RequestID: "9f4ffa22-2676-4391-bfd9-bd7d16c3866c", Text: "Which option do you want: A or B?", Header: "A or B", Options: []string{"Option A", "Option B"}, AskedAt: time.Date(2026, 10, 8, 13, 9, 8, 0, time.UTC)}
	testhelp.Equal(t, "example 6 pending question", pHHServe(h.Pending, "GET", "/projects/demo/pending", "demo", "", ""), pHHResp{200, pHHJSON, `{"question":{"request_id":"9f4ffa22-2676-4391-bfd9-bd7d16c3866c","text":"Which option do you want: A or B?","header":"A or B","options":["Option A","Option B"],"asked_at":"2026-10-08T13:09:08Z"}}` + "\n"})
	// variant: StopCheck null and a Stop; Interrupt, Continue and Usage; Pending's error
	testhelp.Equal(t, "example 6 variant stop null", pHHServe(h.StopCheck, "GET", "/projects/demo/stop", "demo", "", ""), pHHResp{200, pHHJSON, `{"stop":null}` + "\n"})
	f.stop = &control.Stop{Kind: "smoke", Phase: "P3", Reason: "stop after smoke", IssueURL: "https://github.com/o/r/issues/1", At: time.Date(2026, 10, 8, 17, 0, 0, 0, time.UTC)}
	testhelp.Equal(t, "example 6 variant stop", pHHServe(h.StopCheck, "GET", "/projects/demo/stop", "demo", "", ""), pHHResp{200, pHHJSON, `{"stop":{"kind":"smoke","phase":"P3","reason":"stop after smoke","issue_url":"https://github.com/o/r/issues/1","at":"2026-10-08T17:00:00Z"}}` + "\n"})
	testhelp.Equal(t, "example 6 variant interrupt", pHHServe(h.Interrupt, "POST", "/projects/demo/interrupt", "demo", "", ""), pHHResp{200, pHHJSON, pHHOK})
	testhelp.Equal(t, "example 6 variant continue", pHHServe(h.Continue, "POST", "/projects/demo/continue", "demo", "", ""), pHHResp{200, pHHJSON, pHHOK})
	f.usage = control.Usage{FiveHour: 22, SevenDay: 60, FiveHourResetsAt: 1791469200, SevenDayResetsAt: 1791853200, SessionCostUSD: 0.0064, StretchCostUSD: 5.2609, LimitsAt: time.Date(2026, 10, 8, 13, 9, 3, 0, time.UTC)}
	testhelp.Equal(t, "example 6 variant usage", pHHServe(h.Usage, "GET", "/projects/demo/usage", "demo", "", ""), pHHResp{200, pHHJSON, `{"five_hour":22,"seven_day":60,"five_hour_resets_at":1791469200,"seven_day_resets_at":1791853200,"session_cost_usd":0.0064,"stretch_cost_usd":5.2609,"limits_at":"2026-10-08T13:09:03Z"}` + "\n"})
	testhelp.Equal(t, "example 6 variant calls", f.calls, []string{"Pending demo", "Pending demo", "StopCheck demo", "StopCheck demo", "Interrupt demo", "Continue demo", "Usage demo"})
	e := &pHHFake{err: control.ErrNoQuestion}
	testhelp.Equal(t, "example 6 variant pending error", pHHServe(Handlers{Control: e}.Pending, "GET", "/projects/demo/pending", "demo", "", ""), pHHResp{409, pHHJSON, `{"error":{"code":"no_question","message":"no pending question"}}` + "\n"})
	e.err = control.ErrNotWaiting
	testhelp.Equal(t, "example 6 variant continue error", pHHServe(Handlers{Control: e}.Continue, "POST", "/projects/demo/continue", "demo", "", ""), pHHResp{409, pHHJSON, `{"error":{"code":"not_waiting","message":"not waiting"}}` + "\n"})
	e.err = control.ErrNotBusy
	testhelp.Equal(t, "example 6 variant interrupt error", pHHServe(Handlers{Control: e}.Interrupt, "POST", "/projects/demo/interrupt", "demo", "", "").Code, 409)
}

func TestProbeHTTPHandlersExample7(t *testing.T) {
	plan, err := os.ReadFile("../tests/fixtures/plan/queue-3.json")
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	f := &pHHFake{queue: control.QueueView{State: "running", Approved: "7b31dfe", Index: 0, Phases: 3, Current: "P17"}}
	h := Handlers{Control: f}
	got := pHHServe(h.Plan, "POST", "/projects/demo/plan", "demo", pHHJSON, string(plan))
	testhelp.Equal(t, "example 7 POST plan", got, pHHResp{200, pHHJSON, `{"state":"running","approved":"7b31dfe","index":0,"phases":3,"current":"P17","reason":""}` + "\n"})
	testhelp.Equal(t, "example 7 calls", f.calls, []string{"PlanLoad demo"})
	testhelp.Equal(t, "example 7 recorded plan", f.plan, queue.Plan{Approved: "7b31dfe", Phases: []queue.Phase{{ID: "P17", StopAfter: "none"}, {ID: "P18", StopAfter: "smoke", Caps: queue.Caps{ClaudeUSD: 12}}, {ID: "P19", StopAfter: "none"}}})
	// variant: an unknown field in a phase; the Control's error
	got = pHHServe(h.Plan, "POST", "/projects/demo/plan", "demo", pHHJSON, `{"approved":"a","phases":[{"id":"P1","stop":"x"}]}`)
	testhelp.Equal(t, "example 7 variant unknown phase field", got, pHHResp{400, pHHJSON, pHHBadJSON})
	e := &pHHFake{err: fmt.Errorf("%w: queue: no phases", control.ErrBadInput)}
	got = pHHServe(Handlers{Control: e}.Plan, "POST", "/projects/demo/plan", "demo", pHHJSON, `{"approved":"a","phases":[]}`)
	testhelp.Equal(t, "example 7 variant bad input", got, pHHResp{400, pHHJSON, `{"error":{"code":"bad_input","message":"bad input: queue: no phases"}}` + "\n"})
}

func TestProbeHTTPHandlersExample8(t *testing.T) {
	f := &pHHFake{token: control.TokenResult{Access: github.Access{Checked: true, Reachable: false, Push: false, Reason: "401 bad credentials", CheckedAt: time.Date(2026, 10, 8, 15, 0, 0, 0, time.UTC)}}}
	h := Handlers{Control: f}
	got := pHHServe(h.GithubToken, "PUT", "/projects/demo/github-token", "demo", pHHJSON, `{"token":"ghp_abc"}`)
	testhelp.Equal(t, "example 8 PUT github-token", got, pHHResp{200, pHHJSON, `{"access":{"checked":true,"reachable":false,"push":false,"reason":"401 bad credentials","checked_at":"2026-10-08T15:00:00Z"},"pushed":false}` + "\n"})
	testhelp.Equal(t, "example 8 token nowhere in the body", strings.Contains(got.Body, "ghp_abc"), false)
	got = pHHServe(h.Restart, "POST", "/projects/demo/restart", "demo", "", "")
	testhelp.Equal(t, "example 8 restart, no body", got, pHHResp{200, pHHJSON, pHHOK})
	got = pHHServe(h.Restart, "POST", "/projects/demo/restart", "demo", pHHJSON, `{"force":true}`)
	testhelp.Equal(t, "example 8 restart force", got, pHHResp{200, pHHJSON, pHHOK})
	got = pHHServe(h.Projects, "GET", "/projects", "", "", "")
	testhelp.Equal(t, "example 8 projects (nil from the Control)", got, pHHResp{200, pHHJSON, `{"projects":[]}` + "\n"})
	testhelp.Equal(t, "example 8 calls", f.calls, []string{"PutGithubToken demo ghp_abc", "Restart demo false", "Restart demo true", "Projects"})
	d := &pHHFake{err: errors.New("disk full")}
	got = pHHServe(Handlers{Control: d}.Status, "GET", "/projects/demo/status", "demo", "", "")
	testhelp.Equal(t, "example 8 status disk full", got, pHHResp{500, pHHJSON, `{"error":{"code":"internal","message":"internal"}}` + "\n"})
	// variant: CreateProject 201; a project list; restart with a bad body; restart's error
	c := &pHHFake{view: control.ProjectView{Name: "zeta", Language: "go", RepoURL: "https://github.com/o/zeta", State: "idle", CreatedAt: time.Date(2026, 10, 8, 15, 0, 0, 0, time.UTC)}}
	got = pHHServe(Handlers{Control: c}.CreateProject, "POST", "/projects", "", pHHJSON, `{"name":"zeta","language":"go","repo_url":"https://github.com/o/zeta"}`)
	testhelp.Equal(t, "example 8 variant create project", got, pHHResp{201, pHHJSON, `{"name":"zeta","language":"go","repo_url":"https://github.com/o/zeta","state":"idle","created_at":"2026-10-08T15:00:00Z"}` + "\n"})
	testhelp.Equal(t, "example 8 variant create calls", c.calls, []string{"CreateProject zeta go https://github.com/o/zeta ctx=true"})
	c.err = control.ErrExists
	got = pHHServe(Handlers{Control: c}.CreateProject, "POST", "/projects", "", pHHJSON, `{"name":"zeta","language":"go","repo_url":"u"}`)
	testhelp.Equal(t, "example 8 variant create exists", got, pHHResp{409, pHHJSON, `{"error":{"code":"exists","message":"project exists"}}` + "\n"})
	c.err = nil
	c.projects = []control.ProjectView{c.view}
	got = pHHServe(Handlers{Control: c}.Projects, "GET", "/projects", "", "", "")
	testhelp.Equal(t, "example 8 variant one project", got, pHHResp{200, pHHJSON, `{"projects":[{"name":"zeta","language":"go","repo_url":"https://github.com/o/zeta","state":"idle","created_at":"2026-10-08T15:00:00Z"}]}` + "\n"})
	f.calls = nil
	got = pHHServe(h.Restart, "POST", "/projects/demo/restart", "demo", pHHJSON, `{"force":1}`)
	testhelp.Equal(t, "example 8 variant restart bad body", got, pHHResp{400, pHHJSON, pHHBadJSON})
	testhelp.Equal(t, "example 8 variant no Restart on a bad body", len(f.calls), 0)
	b := &pHHFake{err: fmt.Errorf("restart: %w", control.ErrBusy)}
	got = pHHServe(Handlers{Control: b}.Restart, "POST", "/projects/demo/restart", "demo", "", "")
	testhelp.Equal(t, "example 8 variant restart busy", got, pHHResp{409, pHHJSON, `{"error":{"code":"busy","message":"restart: session is busy"}}` + "\n"})
	tk := &pHHFake{err: errors.New("write ghp_zzz failed")}
	got = pHHServe(Handlers{Control: tk}.GithubToken, "PUT", "/projects/demo/github-token", "demo", pHHJSON, `{"token":"ghp_zzz"}`)
	testhelp.Equal(t, "example 8 variant token error is internal", got, pHHResp{500, pHHJSON, `{"error":{"code":"internal","message":"internal"}}` + "\n"})
}
