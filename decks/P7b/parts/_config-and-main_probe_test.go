package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"regexp"
	"sort"
	"strings"
	"testing"
	"time"

	"morphstudio/control"
	"morphstudio/github"
	"morphstudio/internal/testhelp"
	"morphstudio/queue"
	"morphstudio/session"
)

func pCMErr(err error) string {
	if err == nil {
		return "<nil>"
	}
	return err.Error()
}

func pCMEnv(m map[string]string) func(string) string {
	return func(k string) string { return m[k] }
}

func TestProbeConfigAndMainExample1(t *testing.T) {
	got := ParseDotenv("# comment\n\nMORPHD_TOKEN=abc\nexport TG_BOT_TOKEN=\"123:ABC\"\nTG_CHAT_ID='-100'\nBAD LINE\nMORPHD_PORT = 7181\n")
	testhelp.Equal(t, "example 1 ParseDotenv", got, map[string]string{"MORPHD_TOKEN": "abc", "TG_BOT_TOKEN": "123:ABC", "TG_CHAT_ID": "-100", "MORPHD_PORT": "7181"})
	// variants: mismatched quotes stay, the first "=" splits, an empty text is an empty map
	got = ParseDotenv("A=\"x'\nB=c=d\n  # c\n")
	testhelp.Equal(t, "example 1 variant quotes and the first =", got, map[string]string{"A": "\"x'", "B": "c=d"})
	testhelp.Equal(t, "example 1 variant empty text", ParseDotenv(""), map[string]string{})
}

func TestProbeConfigAndMainExample2(t *testing.T) {
	c, err := Load(pCMEnv(map[string]string{"HOME": "/home/morph"}), map[string]string{"MORPHD_TOKEN": "abc"})
	testhelp.Equal(t, "example 2 Load defaults", []any{c, pCMErr(err)}, []any{Config{Token: "abc", Port: 7080, StateDir: "/home/morph/.local/state/morphd",
		ProjectsDir: "/home/morph/projects", ClaudeBin: "claude", MorphBin: "morph", TGToken: "", TGChatID: "", Model: "", ExtraArgs: nil,
		Defaults: queue.Caps{ClaudeUSD: 30, Hours: 3, ExecutorUSD: 5}, StretchUSD: 30, MaxParallel: 1, ResumesPerHour: 3, StallMinutes: 30,
		UsageAlertPercent: 50}, "<nil>"})
}

func TestProbeConfigAndMainExample3(t *testing.T) {
	c, err := Load(pCMEnv(map[string]string{"HOME": "/h", "MORPHD_TOKEN": "env-tok", "MORPHD_PORT": "7181", "MORPHD_CLAUDE_EXTRA_ARGS": "--effort high",
		"MORPHD_CLAUDE_USD": "12.5", "MORPHD_MAX_PARALLEL": "2"}), map[string]string{"MORPHD_TOKEN": "file-tok", "MORPHD_HOURS": "1"})
	testhelp.Equal(t, "example 3 Load", []any{c.Token, c.Port, c.ExtraArgs, c.Defaults, c.MaxParallel, c.StateDir, pCMErr(err)},
		[]any{"env-tok", 7181, []string{"--effort", "high"}, queue.Caps{ClaudeUSD: 12.5, Hours: 1, ExecutorUSD: 5}, 2, "/h/.local/state/morphd", "<nil>"})
	// variant: every other key from the dotenv
	c, err = Load(pCMEnv(map[string]string{"HOME": "/h"}), map[string]string{"MORPHD_TOKEN": "t", "MORPHD_STATE_DIR": "/s", "MORPHD_PROJECTS_DIR": "/p",
		"MORPHD_CLAUDE_BIN": "/b/claude", "MORPHD_MORPH_BIN": "/b/morph", "TG_BOT_TOKEN": "1:A", "TG_CHAT_ID": "-5", "MORPHD_MODEL": "opus",
		"MORPHD_EXECUTOR_USD": "7", "MORPHD_STRETCH_USD": "40", "MORPHD_RESUMES_PER_HOUR": "4", "MORPHD_STALL_MINUTES": "20", "MORPHD_USAGE_ALERT": "70"})
	testhelp.Equal(t, "example 3 variant every key from the dotenv", []any{c, pCMErr(err)}, []any{Config{Token: "t", Port: 7080, StateDir: "/s",
		ProjectsDir: "/p", ClaudeBin: "/b/claude", MorphBin: "/b/morph", TGToken: "1:A", TGChatID: "-5", Model: "opus",
		Defaults: queue.Caps{ClaudeUSD: 30, Hours: 3, ExecutorUSD: 7}, StretchUSD: 40, MaxParallel: 1, ResumesPerHour: 4, StallMinutes: 20,
		UsageAlertPercent: 70}, "<nil>"})
}

func TestProbeConfigAndMainExample4(t *testing.T) {
	_, err := Load(pCMEnv(map[string]string{"HOME": "/h"}), map[string]string{})
	testhelp.Equal(t, "example 4 no token", pCMErr(err), "config: MORPHD_TOKEN is required")
	_, err = Load(pCMEnv(map[string]string{"MORPHD_TOKEN": "x", "MORPHD_PORT": "80a"}), map[string]string{})
	testhelp.Equal(t, "example 4 a bad port", pCMErr(err), "config: MORPHD_PORT: not a number: 80a")
	_, err = Load(pCMEnv(map[string]string{"MORPHD_TOKEN": "x", "MORPHD_STALL_MINUTES": "soon"}), map[string]string{})
	testhelp.Equal(t, "example 4 bad stall minutes", pCMErr(err), "config: MORPHD_STALL_MINUTES: not a number: soon")
	// variant: a float key
	_, err = Load(pCMEnv(map[string]string{"MORPHD_TOKEN": "x"}), map[string]string{"MORPHD_CLAUDE_USD": "lots"})
	testhelp.Equal(t, "example 4 variant a bad float", pCMErr(err), "config: MORPHD_CLAUDE_USD: not a number: lots")
}

// pCMFake implements control.Control; Status answers the HTTP Handlers literal for "demo".
type pCMFake struct{ calls []string }

func (f *pCMFake) Projects() []control.ProjectView { f.calls = append(f.calls, "Projects"); return nil }
func (f *pCMFake) CreateProject(ctx context.Context, name, language, repoURL string) (control.ProjectView, error) {
	return control.ProjectView{}, nil
}
func (f *pCMFake) Status(project string) (control.Status, error) {
	f.calls = append(f.calls, "Status "+project)
	if project != "demo" {
		return control.Status{}, control.ErrUnknownProject
	}
	return control.Status{State: "busy", Phase: "P18", Minutes: 37, CostUSD: 1.92, FiveHour: 22, SevenDay: 60, SessionID: "s-1",
		Queue: control.QueueView{State: "running", Approved: "7b31dfe", Index: 1, Phases: 3, Current: "P18"},
		Repo:  github.Access{Checked: true, Reachable: true, Push: true, CheckedAt: time.Date(2026, 10, 8, 15, 0, 0, 0, time.UTC)}}, nil
}
func (f *pCMFake) Events(project string, since int64, max int) (control.Events, error) {
	return control.Events{}, nil
}
func (f *pCMFake) Order(project, text string) (session.OrderResult, error) {
	return session.OrderResult{}, nil
}
func (f *pCMFake) Pending(project string) (*control.Question, error) { return nil, nil }
func (f *pCMFake) Answer(project string, option int) error           { return nil }
func (f *pCMFake) Interrupt(project string) error                    { return nil }
func (f *pCMFake) Usage(project string) (control.Usage, error)       { return control.Usage{}, nil }
func (f *pCMFake) Restart(project string, force bool) error          { return nil }
func (f *pCMFake) PlanLoad(project string, plan queue.Plan) (control.QueueView, error) {
	return control.QueueView{}, nil
}
func (f *pCMFake) Continue(project string) error                   { return nil }
func (f *pCMFake) StopCheck(project string) (*control.Stop, error) { return nil, nil }
func (f *pCMFake) PutGithubToken(ctx context.Context, project, token string) (control.TokenResult, error) {
	return control.TokenResult{}, nil
}
func (f *pCMFake) PhaseDone(project, sessionToken, phase, next string) error { return nil }
func (f *pCMFake) WaitOperator(project, sessionToken, kind, reason, issueURL string) error {
	return nil
}
func (f *pCMFake) Milestone(project, sessionToken string, m control.Milestone) error { return nil }

var _ control.Control = (*pCMFake)(nil)

func pCMServe(h http.Handler, method, target, auth, body string) (int, string) {
	var req *http.Request
	if body == "" {
		req = httptest.NewRequest(method, target, nil)
	} else {
		req = httptest.NewRequest(method, target, strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Accept", "application/json, text/event-stream")
	}
	if auth != "" {
		req.Header.Set("Authorization", auth)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec.Code, rec.Body.String()
}

func TestProbeConfigAndMainExample5(t *testing.T) {
	f := &pCMFake{}
	h := Build(f, func(p string) bool { return p == "demo" }, func(p string) (string, bool) { return "s", p == "demo" }, "tok-7")
	if h == nil {
		t.Fatalf("example 5 Build returned nil")
	}
	code, body := pCMServe(h, "GET", "/healthz", "", "")
	testhelp.Equal(t, "example 5 GET /healthz", []any{code, body}, []any{200, `{"status":"ok"}` + "\n"})
	code, body = pCMServe(h, "GET", "/projects/demo/status", "Bearer tok-7", "")
	testhelp.Equal(t, "example 5 GET /projects/demo/status", []any{code, body}, []any{200,
		`{"state":"busy","phase":"P18","minutes":37,"cost_usd":1.92,"five_hour":22,"seven_day":60,"session_id":"s-1","queue":{"state":"running","approved":"7b31dfe","index":1,"phases":3,"current":"P18","reason":""},"repo":{"checked":true,"reachable":true,"push":true,"reason":"","checked_at":"2026-10-08T15:00:00Z"}}` + "\n"})
	code, body = pCMServe(h, "POST", "/mcp/demo", "Bearer tok-7", `{"jsonrpc":"2.0","id":1,"method":"tools/list","params":{}}`)
	var m struct {
		Result struct {
			Tools []struct {
				Name string `json:"name"`
			} `json:"tools"`
		} `json:"result"`
	}
	_ = json.Unmarshal([]byte(body), &m)
	names := []string{}
	for _, tl := range m.Result.Tools {
		names = append(names, tl.Name)
	}
	sort.Strings(names)
	hasStatus := sort.SearchStrings(names, "status") < len(names) && names[sort.SearchStrings(names, "status")] == "status"
	testhelp.Equal(t, "example 5 POST /mcp/demo tools/list", []any{code, hasStatus, len(names)}, []any{200, true, 10})
	code, _ = pCMServe(h, "GET", "/projects/demo/status", "", "")
	testhelp.Equal(t, "example 5 GET /projects/demo/status without the token", code, 401)
	// variant: the API token is the one Build got, on the router and the MCP mount
	code, _ = pCMServe(h, "GET", "/projects/demo/status", "Bearer tok-1", "")
	testhelp.Equal(t, "example 5 variant another token on the router", code, 401)
	code, _ = pCMServe(h, "POST", "/mcp/demo", "Bearer tok-1", `{"jsonrpc":"2.0","id":1,"method":"tools/list","params":{}}`)
	testhelp.Equal(t, "example 5 variant another token on the MCP mount", code, 401)
	code, _ = pCMServe(h, "POST", "/mcp/demo/session", "Bearer s", `{"jsonrpc":"2.0","id":1,"method":"tools/list","params":{}}`)
	testhelp.Equal(t, "example 5 variant the session token from session()", code, 200)
}

func TestProbeConfigAndMainExample6(t *testing.T) {
	in := [][16]byte{
		{0x96, 0x7e, 0x8f, 0x4d, 0xa2, 0xeb, 0x0a, 0xa6, 0x16, 0x8b, 0xb9, 0x05, 0x4a, 0x1c, 0x3d, 0x7e},
		{},
		{0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff},
		{0x00, 0x11, 0x22, 0x33, 0x44, 0x55, 0x66, 0x77, 0x88, 0x99, 0xaa, 0xbb, 0xcc, 0xdd, 0xee, 0xff},
	}
	got := []string{}
	for _, b := range in {
		got = append(got, UUIDv4(b))
	}
	testhelp.Equal(t, "example 6 UUIDv4 of the probe's bytes, zero, all-ff, 00..ff", got, []string{
		"967e8f4d-a2eb-4aa6-968b-b9054a1c3d7e", "00000000-0000-4000-8000-000000000000",
		"ffffffff-ffff-4fff-bfff-ffffffffffff", "00112233-4455-4677-8899-aabbccddeeff"})
	// variant: the version and variant nibbles keep the low bits of bytes 6 and 8
	testhelp.Equal(t, "example 6 variant UUIDv4(byte 6 0xc3, byte 8 0x7f)", UUIDv4([16]byte{6: 0xc3, 8: 0x7f}), "00000000-0000-4300-bf00-000000000000")
}

func TestProbeConfigAndMainExample7(t *testing.T) {
	re := regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)
	a, b := newID(), newID()
	testhelp.Equal(t, "example 7 newID() matches the v4 form (a, b)", []bool{re.MatchString(a), re.MatchString(b)}, []bool{true, true})
	testhelp.Equal(t, "example 7 newID() length (a, b)", []int{len(a), len(b)}, []int{36, 36})
	testhelp.Equal(t, "example 7 two ids differ", a != b, true)
	testhelp.Equal(t, "example 7 no upper case", a == strings.ToLower(a) && b == strings.ToLower(b), true)
}
