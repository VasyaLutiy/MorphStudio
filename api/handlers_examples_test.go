package api

import (
	"bytes"
	"context"
	"errors"
	"fmt"
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

// hhFake is the test's Control: configurable results plus a recorded call list.
type hhFake struct {
	projects   []control.ProjectView
	createView control.ProjectView
	createErr  error

	statuses  map[string]control.Status
	statusErr error

	events    control.Events
	eventsErr error

	orderRes session.OrderResult
	orderErr error

	pending    *control.Question
	pendingErr error

	answerErr    error
	interruptErr error

	usage    control.Usage
	usageErr error

	restartErr error

	planArg  queue.Plan
	planView control.QueueView
	planErr  error

	continueErr error

	stop    *control.Stop
	stopErr error

	tokenRes control.TokenResult
	tokenErr error

	phaseDoneErr error
	waitErr      error
	milestoneErr error

	calls []string
}

func (f *hhFake) Projects() []control.ProjectView {
	f.calls = append(f.calls, "Projects()")
	return f.projects
}

func (f *hhFake) CreateProject(ctx context.Context, name, language, repoURL string) (control.ProjectView, error) {
	f.calls = append(f.calls, fmt.Sprintf("CreateProject(%q, %q, %q)", name, language, repoURL))
	return f.createView, f.createErr
}

func (f *hhFake) Status(project string) (control.Status, error) {
	f.calls = append(f.calls, fmt.Sprintf("Status(%q)", project))
	if f.statusErr != nil {
		return control.Status{}, f.statusErr
	}
	if s, ok := f.statuses[project]; ok {
		return s, nil
	}
	return control.Status{}, control.ErrUnknownProject
}

func (f *hhFake) Events(project string, since int64, max int) (control.Events, error) {
	f.calls = append(f.calls, fmt.Sprintf("Events(%q, %d, %d)", project, since, max))
	return f.events, f.eventsErr
}

func (f *hhFake) Order(project, text string) (session.OrderResult, error) {
	f.calls = append(f.calls, fmt.Sprintf("Order(%q, %q)", project, text))
	return f.orderRes, f.orderErr
}

func (f *hhFake) Pending(project string) (*control.Question, error) {
	f.calls = append(f.calls, fmt.Sprintf("Pending(%q)", project))
	return f.pending, f.pendingErr
}

func (f *hhFake) Answer(project string, option int) error {
	f.calls = append(f.calls, fmt.Sprintf("Answer(%q, %d)", project, option))
	return f.answerErr
}

func (f *hhFake) Interrupt(project string) error {
	f.calls = append(f.calls, fmt.Sprintf("Interrupt(%q)", project))
	return f.interruptErr
}

func (f *hhFake) Usage(project string) (control.Usage, error) {
	f.calls = append(f.calls, fmt.Sprintf("Usage(%q)", project))
	return f.usage, f.usageErr
}

func (f *hhFake) Restart(project string, force bool) error {
	f.calls = append(f.calls, fmt.Sprintf("Restart(%q, %t)", project, force))
	return f.restartErr
}

func (f *hhFake) PlanLoad(project string, plan queue.Plan) (control.QueueView, error) {
	f.calls = append(f.calls, fmt.Sprintf("PlanLoad(%q)", project))
	f.planArg = plan
	return f.planView, f.planErr
}

func (f *hhFake) Continue(project string) error {
	f.calls = append(f.calls, fmt.Sprintf("Continue(%q)", project))
	return f.continueErr
}

func (f *hhFake) StopCheck(project string) (*control.Stop, error) {
	f.calls = append(f.calls, fmt.Sprintf("StopCheck(%q)", project))
	return f.stop, f.stopErr
}

func (f *hhFake) PutGithubToken(ctx context.Context, project, token string) (control.TokenResult, error) {
	f.calls = append(f.calls, fmt.Sprintf("PutGithubToken(%q, %q)", project, token))
	return f.tokenRes, f.tokenErr
}

func (f *hhFake) PhaseDone(project, sessionToken, phase, next string) error {
	f.calls = append(f.calls, fmt.Sprintf("PhaseDone(%q, %q, %q, %q)", project, sessionToken, phase, next))
	return f.phaseDoneErr
}

func (f *hhFake) WaitOperator(project, sessionToken, kind, reason, issueURL string) error {
	f.calls = append(f.calls, fmt.Sprintf("WaitOperator(%q, %q, %q, %q, %q)", project, sessionToken, kind, reason, issueURL))
	return f.waitErr
}

func (f *hhFake) Milestone(project, sessionToken string, m control.Milestone) error {
	f.calls = append(f.calls, fmt.Sprintf("Milestone(%q, %q)", project, sessionToken))
	return f.milestoneErr
}

func TestHTTPHandlersExample1(t *testing.T) {
	fake := &hhFake{statuses: map[string]control.Status{
		"demo": {
			State:     "busy",
			Phase:     "P18",
			Minutes:   37,
			CostUSD:   1.92,
			FiveHour:  22,
			SevenDay:  60,
			SessionID: "s-1",
			Queue:     control.QueueView{State: "running", Approved: "7b31dfe", Index: 1, Phases: 3, Current: "P18"},
			Repo: github.Access{
				Checked:   true,
				Reachable: true,
				Push:      true,
				CheckedAt: time.Date(2026, 10, 8, 15, 0, 0, 0, time.UTC),
			},
		},
	}}
	h := Handlers{Control: fake}

	req := httptest.NewRequest(http.MethodGet, "/projects/demo/status", nil)
	req.SetPathValue("project", "demo")
	rec := httptest.NewRecorder()
	h.Status(rec, req)
	testhelp.Equal(t, "status code", rec.Code, 200)
	testhelp.Equal(t, "content type", rec.Header().Get("Content-Type"), "application/json")
	testhelp.Equal(t, "body", rec.Body.String(), `{"state":"busy","phase":"P18","minutes":37,"cost_usd":1.92,"five_hour":22,"seven_day":60,"session_id":"s-1","queue":{"state":"running","approved":"7b31dfe","index":1,"phases":3,"current":"P18","reason":""},"repo":{"checked":true,"reachable":true,"push":true,"reason":"","checked_at":"2026-10-08T15:00:00Z"}}`+"\n")

	req = httptest.NewRequest(http.MethodGet, "/projects/nope/status", nil)
	req.SetPathValue("project", "nope")
	rec = httptest.NewRecorder()
	h.Status(rec, req)
	testhelp.Equal(t, "unknown status code", rec.Code, 404)
	testhelp.Equal(t, "unknown body", rec.Body.String(), `{"error":{"code":"unknown_project","message":"unknown project"}}`+"\n")
}

func TestHTTPHandlersExample2(t *testing.T) {
	fake := &hhFake{orderRes: session.OrderResult{Queued: 1}}
	h := Handlers{Control: fake}

	req := httptest.NewRequest(http.MethodPost, "/projects/demo/messages", strings.NewReader(`{"text":"smoke checked green. Resume with P12b"}`))
	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	req.SetPathValue("project", "demo")
	rec := httptest.NewRecorder()
	h.Messages(rec, req)
	testhelp.Equal(t, "queued status", rec.Code, 200)
	testhelp.Equal(t, "queued body", rec.Body.String(), `{"queued":1}`+"\n")
	testhelp.Equal(t, "calls", fake.calls, []string{`Order("demo", "smoke checked green. Resume with P12b")`})

	fake.orderRes = session.OrderResult{Sent: true}
	req = httptest.NewRequest(http.MethodPost, "/projects/demo/messages", strings.NewReader(`{"text":"x"}`))
	req.Header.Set("Content-Type", "application/json; charset=utf-8")
	req.SetPathValue("project", "demo")
	rec = httptest.NewRecorder()
	h.Messages(rec, req)
	testhelp.Equal(t, "sent status", rec.Code, 200)
	testhelp.Equal(t, "sent body", rec.Body.String(), `{"sent":true}`+"\n")
}

func TestHTTPHandlersExample3(t *testing.T) {
	fake := &hhFake{}
	h := Handlers{Control: fake}

	cases := []struct {
		name        string
		contentType string
	}{
		{"text plain", "text/plain"},
		{"missing", ""},
		{"xml", "application/xml"},
	}
	for _, tc := range cases {
		req := httptest.NewRequest(http.MethodPost, "/projects/demo/messages", strings.NewReader(`{"text":"x"}`))
		if tc.contentType != "" {
			req.Header.Set("Content-Type", tc.contentType)
		}
		req.SetPathValue("project", "demo")
		rec := httptest.NewRecorder()
		h.Messages(rec, req)
		testhelp.Equal(t, tc.name+" status", rec.Code, 415)
		testhelp.Equal(t, tc.name+" body", rec.Body.String(), `{"error":{"code":"unsupported_media_type","message":"Content-Type must be application/json"}}`+"\n")
	}
	testhelp.Equal(t, "no orders", fake.calls, []string(nil))
}

func TestHTTPHandlersExample4(t *testing.T) {
	fake := &hhFake{}
	h := Handlers{Control: fake}

	badBodies := []string{`{"option":`, `{"option":1,"x":2}`, `{"option":1} {}`, ``}
	for i, body := range badBodies {
		req := httptest.NewRequest(http.MethodPost, "/projects/demo/answer", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.SetPathValue("project", "demo")
		rec := httptest.NewRecorder()
		h.Answer(rec, req)
		testhelp.Equal(t, fmt.Sprintf("bad body %d status", i), rec.Code, 400)
		testhelp.Equal(t, fmt.Sprintf("bad body %d body", i), rec.Body.String(), `{"error":{"code":"bad_json","message":"bad json"}}`+"\n")
	}

	req := httptest.NewRequest(http.MethodPost, "/projects/demo/answer", strings.NewReader(`{"option":2}`))
	req.Header.Set("Content-Type", "application/json")
	req.SetPathValue("project", "demo")
	rec := httptest.NewRecorder()
	h.Answer(rec, req)
	testhelp.Equal(t, "ok status", rec.Code, 200)
	testhelp.Equal(t, "ok body", rec.Body.String(), `{"ok":true}`+"\n")
	testhelp.Equal(t, "calls", fake.calls, []string{`Answer("demo", 2)`})

	fake.calls = nil
	fake.answerErr = fmt.Errorf("%w: %d of 1..%d", control.ErrBadOption, 3, 2)
	req = httptest.NewRequest(http.MethodPost, "/projects/demo/answer", strings.NewReader(`{"option":3}`))
	req.Header.Set("Content-Type", "application/json")
	req.SetPathValue("project", "demo")
	rec = httptest.NewRecorder()
	h.Answer(rec, req)
	testhelp.Equal(t, "bad option status", rec.Code, 400)
	testhelp.Equal(t, "bad option body", rec.Body.String(), `{"error":{"code":"bad_option","message":"option out of range: 3 of 1..2"}}`+"\n")
}

func TestHTTPHandlersExample5(t *testing.T) {
	fake := &hhFake{events: control.Events{Entries: []eventlog.Entry{}, Last: 7}}
	h := Handlers{Control: fake}

	req := httptest.NewRequest(http.MethodGet, "/projects/demo/events?since=2&max=5", nil)
	req.SetPathValue("project", "demo")
	rec := httptest.NewRecorder()
	h.Events(rec, req)
	testhelp.Equal(t, "status", rec.Code, 200)
	testhelp.Equal(t, "body", rec.Body.String(), `{"entries":[],"last":7}`+"\n")
	testhelp.Equal(t, "first calls", fake.calls, []string{`Events("demo", 2, 5)`})

	fake.calls = nil
	req = httptest.NewRequest(http.MethodGet, "/projects/demo/events?since=x", nil)
	req.SetPathValue("project", "demo")
	rec = httptest.NewRecorder()
	h.Events(rec, req)
	testhelp.Equal(t, "bad since status", rec.Code, 400)
	testhelp.Equal(t, "bad since body", rec.Body.String(), `{"error":{"code":"bad_input","message":"since must be an integer"}}`+"\n")
	testhelp.Equal(t, "second calls", fake.calls, []string(nil))

	req = httptest.NewRequest(http.MethodGet, "/projects/demo/events", nil)
	req.SetPathValue("project", "demo")
	rec = httptest.NewRecorder()
	h.Events(rec, req)
	testhelp.Equal(t, "third calls", fake.calls, []string{`Events("demo", 0, 0)`})
}

func TestHTTPHandlersExample6(t *testing.T) {
	fake := &hhFake{}
	h := Handlers{Control: fake}

	req := httptest.NewRequest(http.MethodGet, "/projects/demo/pending", nil)
	req.SetPathValue("project", "demo")
	rec := httptest.NewRecorder()
	h.Pending(rec, req)
	testhelp.Equal(t, "no question status", rec.Code, 200)
	testhelp.Equal(t, "no question body", rec.Body.String(), `{"question":null}`+"\n")

	fake.pending = &control.Question{
		RequestID: "9f4ffa22-2676-4391-bfd9-bd7d16c3866c",
		Text:      "Which option do you want: A or B?",
		Header:    "A or B",
		Options:   []string{"Option A", "Option B"},
		AskedAt:   time.Date(2026, 10, 8, 13, 9, 8, 0, time.UTC),
	}
	req = httptest.NewRequest(http.MethodGet, "/projects/demo/pending", nil)
	req.SetPathValue("project", "demo")
	rec = httptest.NewRecorder()
	h.Pending(rec, req)
	testhelp.Equal(t, "question status", rec.Code, 200)
	testhelp.Equal(t, "question body", rec.Body.String(), `{"question":{"request_id":"9f4ffa22-2676-4391-bfd9-bd7d16c3866c","text":"Which option do you want: A or B?","header":"A or B","options":["Option A","Option B"],"asked_at":"2026-10-08T13:09:08Z"}}`+"\n")
}

func TestHTTPHandlersExample7(t *testing.T) {
	data, err := os.ReadFile("../tests/fixtures/plan/queue-3.json")
	if err != nil {
		t.Fatal(err)
	}
	fake := &hhFake{planView: control.QueueView{State: "running", Approved: "7b31dfe", Index: 0, Phases: 3, Current: "P17"}}
	h := Handlers{Control: fake}

	req := httptest.NewRequest(http.MethodPost, "/projects/demo/plan", bytes.NewReader(data))
	req.Header.Set("Content-Type", "application/json")
	req.SetPathValue("project", "demo")
	rec := httptest.NewRecorder()
	h.Plan(rec, req)
	testhelp.Equal(t, "status", rec.Code, 200)
	testhelp.Equal(t, "body", rec.Body.String(), `{"state":"running","approved":"7b31dfe","index":0,"phases":3,"current":"P17","reason":""}`+"\n")
	testhelp.Equal(t, "plan", fake.planArg, queue.Plan{
		Approved: "7b31dfe",
		Phases: []queue.Phase{
			{ID: "P17", StopAfter: "none"},
			{ID: "P18", StopAfter: "smoke", Caps: queue.Caps{ClaudeUSD: 12}},
			{ID: "P19", StopAfter: "none"},
		},
	})
}

func TestHTTPHandlersExample8(t *testing.T) {
	fake := &hhFake{tokenRes: control.TokenResult{
		Access: github.Access{
			Checked:   true,
			Reachable: false,
			Push:      false,
			Reason:    "401 bad credentials",
			CheckedAt: time.Date(2026, 10, 8, 15, 0, 0, 0, time.UTC),
		},
	}}
	h := Handlers{Control: fake}

	req := httptest.NewRequest(http.MethodPut, "/projects/demo/github-token", strings.NewReader(`{"token":"ghp_abc"}`))
	req.Header.Set("Content-Type", "application/json")
	req.SetPathValue("project", "demo")
	rec := httptest.NewRecorder()
	h.GithubToken(rec, req)
	testhelp.Equal(t, "token status", rec.Code, 200)
	body := rec.Body.String()
	testhelp.Equal(t, "token body", body, `{"access":{"checked":true,"reachable":false,"push":false,"reason":"401 bad credentials","checked_at":"2026-10-08T15:00:00Z"},"pushed":false}`+"\n")
	testhelp.Equal(t, "no token leak", strings.Contains(body, "ghp_abc"), false)
	testhelp.Equal(t, "token calls", fake.calls, []string{`PutGithubToken("demo", "ghp_abc")`})

	fake.calls = nil
	req = httptest.NewRequest(http.MethodPost, "/projects/demo/restart", nil)
	req.SetPathValue("project", "demo")
	rec = httptest.NewRecorder()
	h.Restart(rec, req)
	testhelp.Equal(t, "restart status", rec.Code, 200)
	testhelp.Equal(t, "restart body", rec.Body.String(), `{"ok":true}`+"\n")
	testhelp.Equal(t, "restart calls", fake.calls, []string{`Restart("demo", false)`})

	fake.calls = nil
	req = httptest.NewRequest(http.MethodPost, "/projects/demo/restart", strings.NewReader(`{"force":true}`))
	req.Header.Set("Content-Type", "application/json")
	req.SetPathValue("project", "demo")
	rec = httptest.NewRecorder()
	h.Restart(rec, req)
	testhelp.Equal(t, "force status", rec.Code, 200)
	testhelp.Equal(t, "force body", rec.Body.String(), `{"ok":true}`+"\n")
	testhelp.Equal(t, "force calls", fake.calls, []string{`Restart("demo", true)`})

	req = httptest.NewRequest(http.MethodGet, "/projects", nil)
	rec = httptest.NewRecorder()
	h.Projects(rec, req)
	testhelp.Equal(t, "projects status", rec.Code, 200)
	testhelp.Equal(t, "projects body", rec.Body.String(), `{"projects":[]}`+"\n")

	broken := &hhFake{statusErr: errors.New("disk full")}
	hb := Handlers{Control: broken}
	req = httptest.NewRequest(http.MethodGet, "/projects/demo/status", nil)
	req.SetPathValue("project", "demo")
	rec = httptest.NewRecorder()
	hb.Status(rec, req)
	testhelp.Equal(t, "internal status", rec.Code, 500)
	testhelp.Equal(t, "internal body", rec.Body.String(), `{"error":{"code":"internal","message":"internal"}}`+"\n")
}
