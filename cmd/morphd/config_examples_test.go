package main

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"regexp"
	"strings"
	"testing"
	"time"

	"morphstudio/control"
	"morphstudio/github"
	"morphstudio/internal/testhelp"
	"morphstudio/queue"
	"morphstudio/session"
)

func TestConfigAndMainExample1(t *testing.T) {
	text := "# comment\n\nMORPHD_TOKEN=abc\nexport TG_BOT_TOKEN=\"123:ABC\"\nTG_CHAT_ID='-100'\nBAD LINE\nMORPHD_PORT = 7181\n"
	got := ParseDotenv(text)
	want := map[string]string{
		"MORPHD_TOKEN": "abc",
		"TG_BOT_TOKEN": "123:ABC",
		"TG_CHAT_ID":   "-100",
		"MORPHD_PORT":  "7181",
	}
	testhelp.Equal(t, "ParseDotenv", got, want)
}

func TestConfigAndMainExample2(t *testing.T) {
	env := map[string]string{"HOME": "/home/morph"}
	getenv := func(k string) string { return env[k] }
	dotenv := map[string]string{"MORPHD_TOKEN": "abc"}
	got, err := Load(getenv, dotenv)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	want := Config{
		Token:             "abc",
		Port:              7080,
		StateDir:          "/home/morph/.local/state/morphd",
		ProjectsDir:       "/home/morph/projects",
		ClaudeBin:         "claude",
		MorphBin:          "morph",
		TGToken:           "",
		TGChatID:          "",
		Model:             "",
		ExtraArgs:         nil,
		Defaults:          queue.Caps{ClaudeUSD: 30, Hours: 3, ExecutorUSD: 5},
		StretchUSD:        30,
		MaxParallel:       1,
		ResumesPerHour:    3,
		StallMinutes:      30,
		UsageAlertPercent: 50,
	}
	testhelp.Equal(t, "Config", got, want)
}

func TestConfigAndMainExample3(t *testing.T) {
	env := map[string]string{
		"HOME":                     "/h",
		"MORPHD_TOKEN":             "env-tok",
		"MORPHD_PORT":              "7181",
		"MORPHD_CLAUDE_EXTRA_ARGS": "--effort high",
		"MORPHD_CLAUDE_USD":        "12.5",
		"MORPHD_MAX_PARALLEL":      "2",
	}
	getenv := func(k string) string { return env[k] }
	dotenv := map[string]string{
		"MORPHD_TOKEN": "file-tok",
		"MORPHD_HOURS": "1",
	}
	got, err := Load(getenv, dotenv)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	testhelp.Equal(t, "Token", got.Token, "env-tok")
	testhelp.Equal(t, "Port", got.Port, 7181)
	testhelp.Equal(t, "ExtraArgs", got.ExtraArgs, []string{"--effort", "high"})
	testhelp.Equal(t, "Defaults", got.Defaults, queue.Caps{ClaudeUSD: 12.5, Hours: 1, ExecutorUSD: 5})
	testhelp.Equal(t, "MaxParallel", got.MaxParallel, 2)
}

func TestConfigAndMainExample4(t *testing.T) {
	cases := []struct {
		env    map[string]string
		dotenv map[string]string
		want   string
	}{
		{
			env:    map[string]string{},
			dotenv: map[string]string{},
			want:   "config: MORPHD_TOKEN is required",
		},
		{
			env:    map[string]string{"MORPHD_TOKEN": "x", "MORPHD_PORT": "80a"},
			dotenv: map[string]string{},
			want:   "config: MORPHD_PORT: not a number: 80a",
		},
		{
			env:    map[string]string{"MORPHD_TOKEN": "x", "MORPHD_STALL_MINUTES": "soon"},
			dotenv: map[string]string{},
			want:   "config: MORPHD_STALL_MINUTES: not a number: soon",
		},
	}
	for _, c := range cases {
		env := c.env
		getenv := func(k string) string { return env[k] }
		_, err := Load(getenv, c.dotenv)
		if err == nil {
			t.Fatalf("Load: expected %q, got nil", c.want)
		}
		testhelp.Equal(t, "Load error", err.Error(), c.want)
	}
}

// cmFake is the one fake control of this file: Status answers "demo" with the
// record of the example and refuses every other project with ErrUnknownProject;
// every other method is a no-op.
type cmFake struct{}

func (cmFake) Projects() []control.ProjectView { return nil }

func (cmFake) CreateProject(ctx context.Context, name, language, repoURL string) (control.ProjectView, error) {
	return control.ProjectView{}, nil
}

func (cmFake) Status(project string) (control.Status, error) {
	if project != "demo" {
		return control.Status{}, control.ErrUnknownProject
	}
	return control.Status{
		State:     "busy",
		Phase:     "P18",
		Minutes:   37,
		CostUSD:   1.92,
		FiveHour:  22,
		SevenDay:  60,
		SessionID: "s-1",
		Queue: control.QueueView{
			State:    "running",
			Approved: "7b31dfe",
			Index:    1,
			Phases:   3,
			Current:  "P18",
		},
		Repo: github.Access{
			Checked:   true,
			Reachable: true,
			Push:      true,
			CheckedAt: time.Date(2026, time.October, 8, 15, 0, 0, 0, time.UTC),
		},
	}, nil
}

func (cmFake) Events(project string, since int64, max int) (control.Events, error) {
	return control.Events{}, nil
}

func (cmFake) Order(project, text string) (session.OrderResult, error) {
	return session.OrderResult{}, nil
}

func (cmFake) Pending(project string) (*control.Question, error) { return nil, nil }

func (cmFake) Answer(project string, option int) error { return nil }

func (cmFake) Interrupt(project string) error { return nil }

func (cmFake) Usage(project string) (control.Usage, error) { return control.Usage{}, nil }

func (cmFake) Restart(project string, force bool) error { return nil }

func (cmFake) PlanLoad(project string, plan queue.Plan) (control.QueueView, error) {
	return control.QueueView{}, nil
}

func (cmFake) Continue(project string) error { return nil }

func (cmFake) StopCheck(project string) (*control.Stop, error) { return nil, nil }

func (cmFake) PutGithubToken(ctx context.Context, project, token string) (control.TokenResult, error) {
	return control.TokenResult{}, nil
}

func (cmFake) PhaseDone(project, sessionToken, phase, next string) error { return nil }

func (cmFake) WaitOperator(project, sessionToken, kind, reason, issueURL string) error {
	return nil
}

func (cmFake) Milestone(project, sessionToken string, m control.Milestone) error { return nil }

func TestConfigAndMainExample5(t *testing.T) {
	f := cmFake{}
	h := Build(f,
		func(p string) bool { return p == "demo" },
		func(p string) (string, bool) { return "s", p == "demo" },
		"tok-7",
	)

	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/healthz", nil))
	testhelp.Equal(t, "healthz code", rec.Code, http.StatusOK)
	testhelp.Equal(t, "healthz body", rec.Body.String(), `{"status":"ok"}`+"\n")

	statusReq := httptest.NewRequest(http.MethodGet, "/projects/demo/status", nil)
	statusReq.Header.Set("Authorization", "Bearer tok-7")
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, statusReq)
	testhelp.Equal(t, "status code", rec.Code, http.StatusOK)
	testhelp.Equal(t, "status body", rec.Body.String(),
		`{"state":"busy","phase":"P18","minutes":37,"cost_usd":1.92,"five_hour":22,"seven_day":60,"session_id":"s-1","queue":{"state":"running","approved":"7b31dfe","index":1,"phases":3,"current":"P18","reason":""},"repo":{"checked":true,"reachable":true,"push":true,"reason":"","checked_at":"2026-10-08T15:00:00Z"}}`+"\n")

	mcpReq := httptest.NewRequest(http.MethodPost, "/mcp/demo", strings.NewReader(`{"jsonrpc":"2.0","id":1,"method":"tools/list","params":{}}`))
	mcpReq.Header.Set("Content-Type", "application/json")
	mcpReq.Header.Set("Accept", "application/json, text/event-stream")
	mcpReq.Header.Set("Authorization", "Bearer tok-7")
	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, mcpReq)
	testhelp.Equal(t, "mcp code", rec.Code, http.StatusOK)

	var mcpBody struct {
		Result struct {
			Tools []struct {
				Name string `json:"name"`
			} `json:"tools"`
		} `json:"result"`
	}
	if err := json.Unmarshal(rec.Body.Bytes(), &mcpBody); err != nil {
		t.Fatalf("decode mcp body: %v (body %q)", err, rec.Body.String())
	}
	testhelp.Equal(t, "mcp tool count", len(mcpBody.Result.Tools), 10)
	hasStatus := false
	for _, tool := range mcpBody.Result.Tools {
		if tool.Name == "status" {
			hasStatus = true
		}
	}
	testhelp.Equal(t, "mcp has status", hasStatus, true)

	rec = httptest.NewRecorder()
	h.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/projects/demo/status", nil))
	testhelp.Equal(t, "unauthorized code", rec.Code, 401)
}

func TestConfigAndMainExample6(t *testing.T) {
	probe := [16]byte{0x96, 0x7e, 0x8f, 0x4d, 0xa2, 0xeb, 0x0a, 0xa6, 0x16, 0x8b, 0xb9, 0x05, 0x4a, 0x1c, 0x3d, 0x7e}
	zero := [16]byte{}
	ones := [16]byte{0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff}
	ramp := [16]byte{0x00, 0x11, 0x22, 0x33, 0x44, 0x55, 0x66, 0x77, 0x88, 0x99, 0xaa, 0xbb, 0xcc, 0xdd, 0xee, 0xff}

	cases := []struct {
		in   [16]byte
		want string
	}{
		{probe, "967e8f4d-a2eb-4aa6-968b-b9054a1c3d7e"},
		{zero, "00000000-0000-4000-8000-000000000000"},
		{ones, "ffffffff-ffff-4fff-bfff-ffffffffffff"},
		{ramp, "00112233-4455-4677-8899-aabbccddeeff"},
	}
	for _, c := range cases {
		got := UUIDv4(c.in)
		testhelp.Equal(t, "UUIDv4", got, c.want)
		testhelp.Equal(t, "UUIDv4 length", len(got), 36)
		testhelp.Equal(t, "UUIDv4 lower case", strings.ToLower(got), got)
	}
}

func TestConfigAndMainExample7(t *testing.T) {
	shape := regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-4[0-9a-f]{3}-[89ab][0-9a-f]{3}-[0-9a-f]{12}$`)
	a := newID()
	b := newID()
	for _, id := range []string{a, b} {
		testhelp.Equal(t, "id matches shape", shape.MatchString(id), true)
		testhelp.Equal(t, "id length", len(id), 36)
		testhelp.Equal(t, "id lower case", strings.ToLower(id), id)
	}
	testhelp.Equal(t, "ids differ", a != b, true)
}
