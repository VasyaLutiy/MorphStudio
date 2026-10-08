package telegram

import (
	"context"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"

	"morphstudio/internal/testhelp"
)

type pMPBody struct {
	io.Reader
	closed bool
}

func (b *pMPBody) Close() error { b.closed = true; return nil }

type pMPRec struct {
	calls                    int
	method, url, ctype, body string
	resp                     *pMPBody
}

func (r *pMPRec) do(status int, body string) func(*http.Request) (*http.Response, error) {
	return func(req *http.Request) (*http.Response, error) {
		r.calls++
		r.method, r.url, r.ctype = req.Method, req.URL.String(), req.Header.Get("Content-Type")
		if req.Body != nil {
			b, _ := io.ReadAll(req.Body)
			r.body = string(b)
		}
		r.resp = &pMPBody{Reader: strings.NewReader(body)}
		return &http.Response{StatusCode: status, Body: r.resp, Header: http.Header{}}, nil
	}
}

func pMPErr(t *testing.T, what string, err error, want string) {
	t.Helper()
	if err == nil {
		t.Errorf("%s: error nil, want %q", what, want)
		return
	}
	testhelp.Equal(t, what, err.Error(), want)
}

func TestProbeMilestonePostExample1(t *testing.T) {
	testhelp.Equal(t, "example 1 Text", Text("demo", "start", "P18 started", "$2.10 · 3 min"), "[demo] 🚀 <b>P18 started</b>\n$2.10 · 3 min")
	testhelp.Equal(t, "example 1 Text merge", Text("prj", "merge", "P3 merged", "8/8"), "[prj] 🔀 <b>P3 merged</b>\n8/8")
}

func TestProbeMilestonePostExample2(t *testing.T) {
	testhelp.Equal(t, "example 2 Text ask", Text("x", "ask", "a <b> & c", ""), "[x] 🔔 <b>a &lt;b&gt; &amp; c</b>")
	testhelp.Equal(t, "example 2 Text unknown kind", Text("x", "nope", "h", "n"), "[x] 💬 <b>h</b>\nn")
	testhelp.Equal(t, "example 2 Escape &lt; once", Escape("&lt; <"), "&amp;lt; &lt;")
}

func TestProbeMilestonePostExample3(t *testing.T) {
	got := Text("p", "info", strings.Repeat("é", 301), strings.Repeat("ж", 3501))
	testhelp.Equal(t, "example 3 Text (runes, not bytes)", got, "[p] 💬 <b>"+strings.Repeat("é", 300)+"</b>\n"+strings.Repeat("ж", 3500))
	testhelp.Equal(t, "example 3 short text uncut", Text("p", "info", "ééé", "жж"), "[p] 💬 <b>ééé</b>\nжж")
}

func TestProbeMilestonePostExample4(t *testing.T) {
	rec := &pMPRec{}
	c := Client{Token: "123:ABC", ChatID: "-100", Do: rec.do(200, `{"ok":true}`)}
	st, err := c.Post(context.Background(), "demo", "start", "P18 started", "$2.10 · 3 min")
	testhelp.Equal(t, "example 4 Status", st, Status{Sent: true, HTTPStatus: 200})
	testhelp.Equal(t, "example 4 error", err, error(nil))
	testhelp.Equal(t, "example 4 Do calls", rec.calls, 1)
	testhelp.Equal(t, "example 4 method", rec.method, "POST")
	testhelp.Equal(t, "example 4 URL", rec.url, "https://api.telegram.org/bot123:ABC/sendMessage")
	testhelp.Equal(t, "example 4 Content-Type", rec.ctype, "application/x-www-form-urlencoded")
	testhelp.Equal(t, "example 4 body", rec.body, "chat_id=-100&disable_web_page_preview=true&parse_mode=HTML&text=%5Bdemo%5D+%F0%9F%9A%80+%3Cb%3EP18+started%3C%2Fb%3E%0A%242.10+%C2%B7+3+min")
	if rec.resp != nil {
		testhelp.Equal(t, "example 4 response body closed", rec.resp.closed, true)
	}
}

func TestProbeMilestonePostExample5(t *testing.T) {
	rec := &pMPRec{}
	c := Client{Token: "9:Z", ChatID: "42", Do: rec.do(200, `{"ok":true}`)}
	st, err := c.Post(context.Background(), "x", "ask", "a <b> & c", "")
	testhelp.Equal(t, "example 5 Status", st, Status{Sent: true, HTTPStatus: 200})
	testhelp.Equal(t, "example 5 error", err, error(nil))
	testhelp.Equal(t, "example 5 URL", rec.url, "https://api.telegram.org/bot9:Z/sendMessage")
	testhelp.Equal(t, "example 5 body", rec.body, "chat_id=42&disable_web_page_preview=true&parse_mode=HTML&text=%5Bx%5D+%F0%9F%94%94+%3Cb%3Ea+%26lt%3Bb%26gt%3B+%26amp%3B+c%3C%2Fb%3E")
}

func TestProbeMilestonePostExample6(t *testing.T) {
	for _, c := range []Client{
		{Token: "", ChatID: "42", Do: func(*http.Request) (*http.Response, error) {
			t.Errorf("example 6: Do called with an empty Token")
			return nil, errors.New("called")
		}},
		{Token: "t", ChatID: "", Do: func(*http.Request) (*http.Response, error) {
			t.Errorf("example 6: Do called with an empty ChatID")
			return nil, errors.New("called")
		}},
	} {
		st, err := c.Post(context.Background(), "x", "info", "h", "n")
		testhelp.Equal(t, "example 6 Status (token "+`"`+c.Token+`"`+", chat "+`"`+c.ChatID+`"`+")", st, Status{Skipped: true})
		testhelp.Equal(t, "example 6 error", err, error(nil))
	}
}

func TestProbeMilestonePostExample7(t *testing.T) {
	rec := &pMPRec{}
	c := Client{Token: "123:ABC", ChatID: "1", Do: rec.do(403, `{"ok":false,"description":"Forbidden: bot was blocked"}`)}
	st, err := c.Post(context.Background(), "x", "info", "h", "")
	testhelp.Equal(t, "example 7 Status", st, Status{HTTPStatus: 403})
	pMPErr(t, "example 7 error", err, `telegram: status 403: {"ok":false,"description":"Forbidden: bot was blocked"}`)
	if rec.resp != nil {
		testhelp.Equal(t, "example 7 response body closed", rec.resp.closed, true)
	}
	// variant: only the first 200 bytes of a long body
	rec2 := &pMPRec{}
	c2 := Client{Token: "123:ABC", ChatID: "1", Do: rec2.do(500, strings.Repeat("x", 300))}
	st, err = c2.Post(context.Background(), "x", "info", "h", "")
	testhelp.Equal(t, "example 7 long body Status", st, Status{HTTPStatus: 500})
	pMPErr(t, "example 7 long body error (first 200 bytes)", err, "telegram: status 500: "+strings.Repeat("x", 200))
}

func TestProbeMilestonePostExample8(t *testing.T) {
	c := Client{Token: "123:ABC", ChatID: "1", Do: func(*http.Request) (*http.Response, error) {
		return nil, errors.New(`Post "https://api.telegram.org/bot123:ABC/sendMessage": dial tcp: refused`)
	}}
	st, err := c.Post(context.Background(), "x", "info", "h", "")
	testhelp.Equal(t, "example 8 Status", st, Status{})
	pMPErr(t, "example 8 error", err, `telegram: post failed: Post "https://api.telegram.org/bot***/sendMessage": dial tcp: refused`)
	// variant: every occurrence of the token is masked
	c2 := Client{Token: "77:Q", ChatID: "1", Do: func(*http.Request) (*http.Response, error) {
		return nil, errors.New("77:Q failed twice: 77:Q")
	}}
	_, err = c2.Post(context.Background(), "x", "info", "h", "")
	pMPErr(t, "example 8 every occurrence masked", err, "telegram: post failed: *** failed twice: ***")
}
