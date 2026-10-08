package github

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"morphstudio/internal/testhelp"
)

// fakeDo records the last request it is handed and answers a literal response,
// or fails with err.
type fakeDo struct {
	status int
	body   string
	err    error

	calls int
	req   *http.Request
}

func (f *fakeDo) do(req *http.Request) (*http.Response, error) {
	f.calls++
	f.req = req
	if f.err != nil {
		return nil, f.err
	}
	return &http.Response{
		StatusCode: f.status,
		Header:     http.Header{"Content-Type": []string{"application/json"}},
		Body:       io.NopCloser(strings.NewReader(f.body)),
	}, nil
}

func TestRepoAccessExample1(t *testing.T) {
	cases := []struct {
		in         string
		owner, rep string
	}{
		{"https://github.com/VasyaLutiy/MorphStudio", "VasyaLutiy", "MorphStudio"},
		{"https://github.com/VasyaLutiy/MorphStudio.git", "VasyaLutiy", "MorphStudio"},
		{"https://github.com/o/r/", "o", "r"},
		{"git@github.com:o/r.git", "o", "r"},
	}
	for _, c := range cases {
		owner, repo, err := Parse(c.in)
		testhelp.Equal(t, "owner for "+c.in, owner, c.owner)
		testhelp.Equal(t, "repo for "+c.in, repo, c.rep)
		testhelp.Equal(t, "err for "+c.in, err, nil)
	}
}

func TestRepoAccessExample2(t *testing.T) {
	for _, in := range []string{
		"https://gitlab.com/o/r",
		"https://github.com/o",
		"github.com/o/r",
		"",
	} {
		_, _, err := Parse(in)
		got := ""
		if err != nil {
			got = err.Error()
		}
		testhelp.Equal(t, "error for "+in, got, "github: not a github repository url: "+in)
	}
}

func TestRepoAccessExample3(t *testing.T) {
	got := CredentialLine("ghp_abc")
	testhelp.Equal(t, "CredentialLine", got, "https://x-access-token:ghp_abc@github.com\n")
}

func TestRepoAccessExample4(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 10, 8, 15, 0, 0, 0, time.UTC)
	f := &fakeDo{
		status: http.StatusOK,
		body:   `{"full_name":"o/r","permissions":{"admin":false,"push":true,"pull":true}}`,
	}

	got := Check(ctx, f.do, "ghp_abc", "https://github.com/o/r", now)
	want := Access{Checked: true, Reachable: true, Push: true, Reason: "", CheckedAt: now}
	testhelp.Equal(t, "access", got, want)

	if f.req == nil {
		t.Fatal("request: no request recorded")
	}
	testhelp.Equal(t, "request method", f.req.Method, http.MethodGet)
	testhelp.Equal(t, "request url", f.req.URL.String(), "https://api.github.com/repos/o/r")
	testhelp.Equal(t, "request authorization", f.req.Header.Get("Authorization"), "Bearer ghp_abc")
	testhelp.Equal(t, "request accept", f.req.Header.Get("Accept"), "application/vnd.github+json")
	testhelp.Equal(t, "request user-agent", f.req.Header.Get("User-Agent"), "morphd")
}

func TestRepoAccessExample5(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 10, 8, 15, 0, 0, 0, time.UTC)
	for _, body := range []string{
		`{"permissions":{"push":false}}`,
		`{"full_name":"o/r"}`,
	} {
		f := &fakeDo{status: http.StatusOK, body: body}
		got := Check(ctx, f.do, "ghp_abc", "https://github.com/o/r", now)
		testhelp.Equal(t, "reachable for "+body, got.Reachable, true)
		testhelp.Equal(t, "push for "+body, got.Push, false)
	}
}

func TestRepoAccessExample6(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 10, 8, 15, 0, 0, 0, time.UTC)
	cases := []struct {
		status int
		reason string
	}{
		{http.StatusUnauthorized, "401 bad credentials"},
		{http.StatusForbidden, "403 forbidden"},
		{http.StatusNotFound, "404 not found or no access"},
		{http.StatusInternalServerError, "500 unexpected"},
	}
	for _, c := range cases {
		f := &fakeDo{status: c.status, body: "any body"}
		got := Check(ctx, f.do, "ghp_abc", "https://github.com/o/r", now)
		testhelp.Equal(t, "reason for status "+c.reason, got.Reason, c.reason)
		testhelp.Equal(t, "reachable for status "+c.reason, got.Reachable, false)
		testhelp.Equal(t, "push for status "+c.reason, got.Push, false)
	}
}

func TestRepoAccessExample7(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 10, 8, 15, 0, 0, 0, time.UTC)
	f := &fakeDo{status: http.StatusOK, body: "{}"}

	got := Check(ctx, f.do, "", "https://github.com/o/r", now)
	want := Access{Checked: true, Reason: "no token", CheckedAt: now}
	testhelp.Equal(t, "no token access", got, want)
	testhelp.Equal(t, "no token calls", f.calls, 0)

	got = Check(ctx, f.do, "t", "https://gitlab.com/o/r", now)
	want = Access{
		Checked:   true,
		Reason:    "github: not a github repository url: https://gitlab.com/o/r",
		CheckedAt: now,
	}
	testhelp.Equal(t, "bad url access", got, want)
	testhelp.Equal(t, "bad url calls", f.calls, 0)
}

func TestRepoAccessExample8(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 10, 8, 15, 0, 0, 0, time.UTC)
	f := &fakeDo{err: errors.New(`Get "https://api.github.com/repos/o/r": dial tcp: refused ghp_abc`)}

	got := Check(ctx, f.do, "ghp_abc", "https://github.com/o/r", now)
	want := "network: Get \"https://api.github.com/repos/o/r\": dial tcp: refused ***"
	testhelp.Equal(t, "reason", got.Reason, want)
	testhelp.Equal(t, "checked", got.Checked, true)
}
