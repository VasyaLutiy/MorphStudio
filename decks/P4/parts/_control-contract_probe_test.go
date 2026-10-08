package control

import (
	"encoding/json"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"testing"
	"time"

	"morphstudio/eventlog"
	"morphstudio/github"
	"morphstudio/internal/testhelp"
	"morphstudio/session"
)

func pCCJSON(t *testing.T, v any) string {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Errorf("json.Marshal: %v", err)
	}
	return string(b)
}

func TestProbeControlContractExample1(t *testing.T) {
	s := Status{State: "busy", Phase: "P18", Minutes: 37, CostUSD: 1.92, FiveHour: 22, SevenDay: 60, SessionID: "s-1",
		Queue: QueueView{State: "running", Approved: "7b31dfe", Index: 1, Phases: 3, Current: "P18"},
		Repo:  github.Access{Checked: true, Reachable: true, Push: true, CheckedAt: time.Date(2026, 10, 8, 15, 0, 0, 0, time.UTC)}}
	testhelp.Equal(t, "example 1 json.Marshal(Status)", pCCJSON(t, s), `{"state":"busy","phase":"P18","minutes":37,"cost_usd":1.92,"five_hour":22,"seven_day":60,"session_id":"s-1","queue":{"state":"running","approved":"7b31dfe","index":1,"phases":3,"current":"P18","reason":""},"repo":{"checked":true,"reachable":true,"push":true,"reason":"","checked_at":"2026-10-08T15:00:00Z"}}`)
	// variant: the zero Status keeps every key but "stop"
	testhelp.Equal(t, "example 1 variant json.Marshal(Status{})", pCCJSON(t, Status{}), `{"state":"","phase":"","minutes":0,"cost_usd":0,"five_hour":0,"seven_day":0,"session_id":"","queue":{"state":"","approved":"","index":0,"phases":0,"current":"","reason":""},"repo":{"checked":false,"reachable":false,"push":false,"reason":"","checked_at":"0001-01-01T00:00:00Z"}}`)
}

func TestProbeControlContractExample2(t *testing.T) {
	s := Status{State: "waiting", Stop: &Stop{Kind: "not_pushed", Phase: "P17", Reason: "not pushed: HEAD bbb222, origin/main aaa111", At: time.Date(2026, 10, 8, 16, 0, 0, 0, time.UTC)}}
	got := pCCJSON(t, s)
	tail := `,"stop":{"kind":"not_pushed","phase":"P17","reason":"not pushed: HEAD bbb222, origin/main aaa111","at":"2026-10-08T16:00:00Z"}}`
	if !strings.HasSuffix(got, tail) {
		t.Errorf("example 2 json.Marshal(Status with Stop):\ngot:  %s\nwant the end: %s", got, tail)
	}
	testhelp.Equal(t, "example 2 holds no issue_url", strings.Contains(got, "issue_url"), false)
	// variant: a Stop with an issue URL carries it before "at"
	st := Stop{Kind: "emergency", Phase: "P5", Reason: "card red", IssueURL: "https://github.com/o/r/issues/12", At: time.Date(2027, 2, 3, 4, 5, 6, 0, time.UTC)}
	testhelp.Equal(t, "example 2 variant json.Marshal(Stop)", pCCJSON(t, st), `{"kind":"emergency","phase":"P5","reason":"card red","issue_url":"https://github.com/o/r/issues/12","at":"2027-02-03T04:05:06Z"}`)
}

func TestProbeControlContractExample3(t *testing.T) {
	q := Question{RequestID: "9f4ffa22-2676-4391-bfd9-bd7d16c3866c", Text: "Which option do you want: A or B?", Header: "A or B", Options: []string{"Option A", "Option B"}, AskedAt: time.Date(2026, 10, 8, 13, 9, 8, 0, time.UTC)}
	testhelp.Equal(t, "example 3 json.Marshal(Question)", pCCJSON(t, q), `{"request_id":"9f4ffa22-2676-4391-bfd9-bd7d16c3866c","text":"Which option do you want: A or B?","header":"A or B","options":["Option A","Option B"],"asked_at":"2026-10-08T13:09:08Z"}`)
	// variant: the other views of the contract, every tag
	pv := ProjectView{Name: "demo", Language: "go", RepoURL: "https://github.com/o/demo", State: "idle", CreatedAt: time.Date(2026, 10, 8, 15, 0, 0, 0, time.UTC)}
	testhelp.Equal(t, "example 3 variant json.Marshal(ProjectView)", pCCJSON(t, pv), `{"name":"demo","language":"go","repo_url":"https://github.com/o/demo","state":"idle","created_at":"2026-10-08T15:00:00Z"}`)
	testhelp.Equal(t, "example 3 variant json.Marshal(Milestone)", pCCJSON(t, Milestone{Kind: "run", Headline: "P4 green", Numbers: "10/10"}), `{"kind":"run","headline":"P4 green","numbers":"10/10"}`)
	tr := TokenResult{Access: github.Access{Checked: true, Reachable: true, Push: true, CheckedAt: time.Date(2026, 10, 8, 15, 0, 0, 0, time.UTC)}, Pushed: true}
	testhelp.Equal(t, "example 3 variant json.Marshal(TokenResult)", pCCJSON(t, tr), `{"access":{"checked":true,"reachable":true,"push":true,"reason":"","checked_at":"2026-10-08T15:00:00Z"},"pushed":true}`)
	tr = TokenResult{Pushed: false, PushError: "bootstrap: push failed (exit 1): denied"}
	testhelp.Equal(t, "example 3 variant json.Marshal(TokenResult with PushError)", pCCJSON(t, tr), `{"access":{"checked":false,"reachable":false,"push":false,"reason":"","checked_at":"0001-01-01T00:00:00Z"},"pushed":false,"push_error":"bootstrap: push failed (exit 1): denied"}`)
	ev := Events{Entries: []eventlog.Entry{{Seq: 3, T: 1.5, Dir: "out", Msg: json.RawMessage(`{"type":"x"}`)}}, Last: 3}
	testhelp.Equal(t, "example 3 variant json.Marshal(Events)", pCCJSON(t, ev), `{"entries":[{"seq":3,"t":1.5,"dir":"out","msg":{"type":"x"}}],"last":3}`)
}

func TestProbeControlContractExample4(t *testing.T) {
	in := []float64{0.22, 0.6, 0.005, 1.3, -0.1, 0.995}
	got := []int{}
	for _, u := range in {
		got = append(got, Percent(u))
	}
	testhelp.Equal(t, "example 4 Percent(0.22, 0.6, 0.005, 1.3, -0.1, 0.995)", got, []int{22, 60, 1, 100, 0, 100})
	// variant: other points of the rounding and the clamp
	in = []float64{0.994, 0.5, 0, 1, 0.004, 0.775, 2, -3}
	got = []int{}
	for _, u := range in {
		got = append(got, Percent(u))
	}
	testhelp.Equal(t, "example 4 variant Percent(0.994, 0.5, 0, 1, 0.004, 0.775, 2, -3)", got, []int{99, 50, 0, 100, 0, 78, 100, 0})
}

type pCCCode struct {
	Status int
	Code   string
}

func pCCOf(err error) pCCCode {
	s, c := Code(err)
	return pCCCode{s, c}
}

func TestProbeControlContractExample5(t *testing.T) {
	in := []error{nil, ErrUnknownProject, fmt.Errorf("wrap: %w", ErrBadOption), ErrBadToken, ErrExists, ErrNoSession, ErrBusy, ErrNoQuestion, ErrNotBusy, ErrNotWaiting, ErrBadInput, errors.New("disk full")}
	want := []pCCCode{{200, ""}, {404, "unknown_project"}, {400, "bad_option"}, {401, "bad_session_token"}, {409, "exists"}, {409, "no_session"}, {409, "busy"}, {409, "no_question"}, {409, "not_busy"}, {409, "not_waiting"}, {400, "bad_input"}, {500, "internal"}}
	for i, e := range in {
		name := "nil"
		if e != nil {
			name = e.Error()
		}
		testhelp.Equal(t, fmt.Sprintf("example 5 Code(%s)", name), pCCOf(e), want[i])
	}
	// variant: the texts of the errors; wrapped errors; the order of errors.Is on an error wrapping two
	testhelp.Equal(t, "example 5 variant error texts",
		[]string{ErrUnknownProject.Error(), ErrNoSession.Error(), ErrBusy.Error(), ErrNotWaiting.Error(), ErrBadToken.Error(), ErrBadInput.Error(), ErrExists.Error()},
		[]string{"unknown project", "no session running", "session is busy", "not waiting", "session token mismatch", "bad input", "project exists"})
	testhelp.Equal(t, "example 5 variant Code(wrapped ErrBusy)", pCCOf(fmt.Errorf("restart: %w", ErrBusy)), pCCCode{409, "busy"})
	testhelp.Equal(t, "example 5 variant Code(wrapped session.ErrNotBusy)", pCCOf(fmt.Errorf("x: %w", session.ErrNotBusy)), pCCCode{409, "not_busy"})
	testhelp.Equal(t, "example 5 variant Code(ErrBusy and ErrUnknownProject)", pCCOf(fmt.Errorf("%w; %w", ErrBusy, ErrUnknownProject)), pCCCode{404, "unknown_project"})
	testhelp.Equal(t, "example 5 variant Code(ErrNoSession and ErrBadInput)", pCCOf(fmt.Errorf("%w; %w", ErrNoSession, ErrBadInput)), pCCCode{400, "bad_input"})
	testhelp.Equal(t, "example 5 variant Code(ErrNotWaiting and ErrBadToken)", pCCOf(fmt.Errorf("%w; %w", ErrNotWaiting, ErrBadToken)), pCCCode{401, "bad_session_token"})
	testhelp.Equal(t, "example 5 variant Code(an error with the text of ErrBusy)", pCCOf(errors.New("session is busy")), pCCCode{500, "internal"})
}

var pCCMethods = [][2]string{
	{"Answer", "func(string, int) error"},
	{"Continue", "func(string) error"},
	{"CreateProject", "func(context.Context, string, string, string) (control.ProjectView, error)"},
	{"Events", "func(string, int64, int) (control.Events, error)"},
	{"Interrupt", "func(string) error"},
	{"Milestone", "func(string, string, control.Milestone) error"},
	{"Order", "func(string, string) (session.OrderResult, error)"},
	{"Pending", "func(string) (*control.Question, error)"},
	{"PhaseDone", "func(string, string, string, string) error"},
	{"PlanLoad", "func(string, queue.Plan) (control.QueueView, error)"},
	{"Projects", "func() []control.ProjectView"},
	{"PutGithubToken", "func(context.Context, string, string) (control.TokenResult, error)"},
	{"Restart", "func(string, bool) error"},
	{"Status", "func(string) (control.Status, error)"},
	{"StopCheck", "func(string) (*control.Stop, error)"},
	{"Usage", "func(string) (control.Usage, error)"},
	{"WaitOperator", "func(string, string, string, string, string) error"},
}

func TestProbeControlContractExample6(t *testing.T) {
	testhelp.Equal(t, "example 6 ErrNoQuestion is session.ErrNoQuestion", ErrNoQuestion == session.ErrNoQuestion, true)
	testhelp.Equal(t, "example 6 ErrBadOption is session.ErrBadOption", ErrBadOption == session.ErrBadOption, true)
	testhelp.Equal(t, "example 6 ErrNotBusy is session.ErrNotBusy", ErrNotBusy == session.ErrNotBusy, true)
	ty := reflect.TypeOf((*Control)(nil)).Elem()
	testhelp.Equal(t, "example 6 Control is an interface", ty.Kind(), reflect.Interface)
	testhelp.Equal(t, "example 6 Control method count", ty.NumMethod(), 17)
	got := [][2]string{}
	for i := 0; i < ty.NumMethod(); i++ {
		m := ty.Method(i)
		got = append(got, [2]string{m.Name, m.Type.String()})
	}
	testhelp.Equal(t, "example 6 Control methods (name, signature) sorted by name", got, pCCMethods)
	// variant: the field order of the views (json keys in declaration order)
	testhelp.Equal(t, "example 6 variant json.Marshal(QueueView{Reason})", pCCJSON(t, QueueView{Reason: "smoke", Current: "P2", Approved: "abc"}), `{"state":"","approved":"abc","index":0,"phases":0,"current":"P2","reason":"smoke"}`)
}

func TestProbeControlContractExample7(t *testing.T) {
	u := Usage{FiveHour: 22, SevenDay: 60, FiveHourResetsAt: 1791469200, SevenDayResetsAt: 1791853200, SessionCostUSD: 0.0064, StretchCostUSD: 5.2609, LimitsAt: time.Date(2026, 10, 8, 13, 9, 3, 0, time.UTC)}
	testhelp.Equal(t, "example 7 json.Marshal(Usage)", pCCJSON(t, u), `{"five_hour":22,"seven_day":60,"five_hour_resets_at":1791469200,"seven_day_resets_at":1791853200,"session_cost_usd":0.0064,"stretch_cost_usd":5.2609,"limits_at":"2026-10-08T13:09:03Z"}`)
	// variant: other values
	u = Usage{FiveHour: 100, SevenDay: 3, FiveHourResetsAt: 1800000000, SevenDayResetsAt: 1800600000, SessionCostUSD: 1.5, StretchCostUSD: 30, LimitsAt: time.Date(2027, 1, 1, 0, 0, 0, 0, time.UTC)}
	testhelp.Equal(t, "example 7 variant json.Marshal(Usage)", pCCJSON(t, u), `{"five_hour":100,"seven_day":3,"five_hour_resets_at":1800000000,"seven_day_resets_at":1800600000,"session_cost_usd":1.5,"stretch_cost_usd":30,"limits_at":"2027-01-01T00:00:00Z"}`)
}
