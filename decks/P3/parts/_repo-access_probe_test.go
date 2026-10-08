package github

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"morphstudio/internal/testhelp"
)

type pRABody struct {
	io.Reader
	closed bool
}

func (b *pRABody) Close() error { b.closed = true; return nil }

type pRARec struct {
	calls                      int
	method, url, auth, acc, ua string
	resp                       *pRABody
}

func (r *pRARec) do(status int, body string) func(*http.Request) (*http.Response, error) {
	return func(req *http.Request) (*http.Response, error) {
		r.calls++
		r.method, r.url = req.Method, req.URL.String()
		r.auth, r.acc, r.ua = req.Header.Get("Authorization"), req.Header.Get("Accept"), req.Header.Get("User-Agent")
		r.resp = &pRABody{Reader: strings.NewReader(body)}
		return &http.Response{StatusCode: status, Body: r.resp, Header: http.Header{}}, nil
	}
}

var pRANow = time.Date(2026, 10, 8, 15, 0, 0, 0, time.UTC)

func pRAParse(t *testing.T, what, in, owner, repo string) {
	t.Helper()
	o, r, err := Parse(in)
	testhelp.Equal(t, what+" Parse("+in+")", []string{o, r}, []string{owner, repo})
	testhelp.Equal(t, what+" Parse("+in+") error", err, error(nil))
}

func pRAParseErr(t *testing.T, what, in string) {
	t.Helper()
	o, r, err := Parse(in)
	testhelp.Equal(t, what+" Parse("+in+") owner, repo", []string{o, r}, []string{"", ""})
	want := "github: not a github repository url: " + in
	if err == nil {
		t.Errorf("%s Parse(%q): error nil, want %q", what, in, want)
	} else {
		testhelp.Equal(t, what+" Parse("+in+") error", err.Error(), want)
	}
}

func TestProbeRepoAccessExample1(t *testing.T) {
	pRAParse(t, "example 1", "https://github.com/VasyaLutiy/MorphStudio", "VasyaLutiy", "MorphStudio")
	pRAParse(t, "example 1", "https://github.com/VasyaLutiy/MorphStudio.git", "VasyaLutiy", "MorphStudio")
	pRAParse(t, "example 1", "https://github.com/o/r/", "o", "r")
	pRAParse(t, "example 1", "git@github.com:o/r.git", "o", "r")
	pRAParse(t, "example 1 variant", "https://github.com/acme-9/tool.x", "acme-9", "tool.x")
	pRAParse(t, "example 1 variant", "git@github.com:Zed/q-1.git", "Zed", "q-1")
}

func TestProbeRepoAccessExample2(t *testing.T) {
	for _, in := range []string{"https://gitlab.com/o/r", "https://github.com/o", "github.com/o/r", ""} {
		pRAParseErr(t, "example 2", in)
	}
	for _, in := range []string{"https://github.com/o/r/x", "https://github.com//r", "git@github.com:o/.git", "http://github.com/o/r"} {
		pRAParseErr(t, "example 2 variant", in)
	}
}

func TestProbeRepoAccessExample3(t *testing.T) {
	testhelp.Equal(t, "example 3 CredentialLine", CredentialLine("ghp_abc"), "https://x-access-token:ghp_abc@github.com\n")
	testhelp.Equal(t, "example 3 CredentialLine (other token)", CredentialLine("github_pat_11X"), "https://x-access-token:github_pat_11X@github.com\n")
}

func TestProbeRepoAccessExample4(t *testing.T) {
	rec := &pRARec{}
	a := Check(context.Background(), rec.do(200, `{"full_name":"o/r","permissions":{"admin":false,"push":true,"pull":true}}`), "ghp_abc", "https://github.com/o/r", pRANow)
	testhelp.Equal(t, "example 4 Access", a, Access{Checked: true, Reachable: true, Push: true, Reason: "", CheckedAt: pRANow})
	testhelp.Equal(t, "example 4 Do calls", rec.calls, 1)
	testhelp.Equal(t, "example 4 request", []string{rec.method, rec.url, rec.auth, rec.acc, rec.ua}, []string{"GET", "https://api.github.com/repos/o/r", "Bearer ghp_abc", "application/vnd.github+json", "morphd"})
	if rec.resp != nil {
		testhelp.Equal(t, "example 4 response body closed", rec.resp.closed, true)
	}
	// variant: another repository, token and time
	rec2 := &pRARec{}
	now2 := time.Date(2027, 1, 2, 3, 4, 5, 0, time.UTC)
	a = Check(context.Background(), rec2.do(200, `{"permissions":{"push":true}}`), "t-9", "git@github.com:acme/tool.git", now2)
	testhelp.Equal(t, "example 4 variant Access", a, Access{Checked: true, Reachable: true, Push: true, CheckedAt: now2})
	testhelp.Equal(t, "example 4 variant request", []string{rec2.url, rec2.auth}, []string{"https://api.github.com/repos/acme/tool", "Bearer t-9"})
}

func TestProbeRepoAccessExample5(t *testing.T) {
	for _, body := range []string{`{"permissions":{"push":false}}`, `{"full_name":"o/r"}`} {
		a := Check(context.Background(), (&pRARec{}).do(200, body), "ghp_abc", "https://github.com/o/r", pRANow)
		testhelp.Equal(t, "example 5 Access for "+body, a, Access{Checked: true, Reachable: true, Push: false, CheckedAt: pRANow})
	}
}

func TestProbeRepoAccessExample6(t *testing.T) {
	for _, c := range []struct {
		status int
		reason string
	}{{401, "401 bad credentials"}, {403, "403 forbidden"}, {404, "404 not found or no access"}, {500, "500 unexpected"}, {502, "502 unexpected"}, {301, "301 unexpected"}} {
		rec := &pRARec{}
		a := Check(context.Background(), rec.do(c.status, `{"message":"x"}`), "ghp_abc", "https://github.com/o/r", pRANow)
		testhelp.Equal(t, "example 6 Access for status "+c.reason, a, Access{Checked: true, Reason: c.reason, CheckedAt: pRANow})
		if rec.resp != nil {
			testhelp.Equal(t, "example 6 response body closed ("+c.reason+")", rec.resp.closed, true)
		}
	}
}

func TestProbeRepoAccessExample7(t *testing.T) {
	rec := &pRARec{}
	a := Check(context.Background(), rec.do(200, `{}`), "", "https://github.com/o/r", pRANow)
	testhelp.Equal(t, "example 7 Access (no token)", a, Access{Checked: true, Reason: "no token", CheckedAt: pRANow})
	a = Check(context.Background(), rec.do(200, `{}`), "t", "https://gitlab.com/o/r", pRANow)
	testhelp.Equal(t, "example 7 Access (bad url)", a, Access{Checked: true, Reason: "github: not a github repository url: https://gitlab.com/o/r", CheckedAt: pRANow})
	testhelp.Equal(t, "example 7 Do calls", rec.calls, 0)
	// variant: the JSON tags the status JSON (P4) reads
	b, _ := json.Marshal(Check(context.Background(), rec.do(200, `{}`), "", "https://github.com/o/r", pRANow))
	testhelp.Equal(t, "example 7 variant JSON", string(b), `{"checked":true,"reachable":false,"push":false,"reason":"no token","checked_at":"2026-10-08T15:00:00Z"}`)
	// variant: a nil do never panics
	func() {
		defer func() {
			if r := recover(); r != nil {
				t.Errorf("example 7 variant: Check with a nil do panics: %v", r)
			}
		}()
		a = Check(context.Background(), nil, "t", "https://github.com/o/r", pRANow)
		testhelp.Equal(t, "example 7 variant Access (nil do)", a, Access{Checked: true, Reason: "network: no Do", CheckedAt: pRANow})
	}()
}

func TestProbeRepoAccessExample8(t *testing.T) {
	do := func(*http.Request) (*http.Response, error) {
		return nil, errors.New("Get \"https://api.github.com/repos/o/r\": dial tcp: refused ghp_abc")
	}
	a := Check(context.Background(), do, "ghp_abc", "https://github.com/o/r", pRANow)
	testhelp.Equal(t, "example 8 Access", a, Access{Checked: true, Reason: "network: Get \"https://api.github.com/repos/o/r\": dial tcp: refused ***", CheckedAt: pRANow})
	// variant: every occurrence masked
	do2 := func(*http.Request) (*http.Response, error) { return nil, errors.New("k-5 then k-5") }
	a = Check(context.Background(), do2, "k-5", "https://github.com/o/r", pRANow)
	testhelp.Equal(t, "example 8 variant Reason (every occurrence)", a.Reason, "network: *** then ***")
}
