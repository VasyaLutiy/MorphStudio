package telegram

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"morphstudio/internal/testhelp"
)

func TestMilestonePostExample1(t *testing.T) {
	got := Text("demo", "start", "P18 started", "$2.10 · 3 min")
	want := "[demo] 🚀 <b>P18 started</b>\n$2.10 · 3 min"
	testhelp.Equal(t, "Text", got, want)
}

func TestMilestonePostExample2(t *testing.T) {
	got := Text("x", "ask", "a <b> & c", "")
	want := "[x] 🔔 <b>a &lt;b&gt; &amp; c</b>"
	testhelp.Equal(t, "Text", got, want)

	got = Text("x", "nope", "h", "n")
	want = "[x] 💬 <b>h</b>\nn"
	testhelp.Equal(t, "Text", got, want)
}

func TestMilestonePostExample3(t *testing.T) {
	headline := strings.Repeat("é", 301)
	numbers := strings.Repeat("ж", 3501)
	got := Text("x", "info", headline, numbers)
	want := "[x] 💬 <b>" + strings.Repeat("é", 300) + "</b>\n" + strings.Repeat("ж", 3500)
	testhelp.Equal(t, "Text", got, want)
}

func TestMilestonePostExample4(t *testing.T) {
	var recorded *http.Request
	var body []byte
	c := Client{Token: "123:ABC", ChatID: "-100", Do: func(r *http.Request) (*http.Response, error) {
		recorded = r
		b, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatal(err)
		}
		body = b
		rec := httptest.NewRecorder()
		rec.WriteHeader(http.StatusOK)
		rec.WriteString(`{"ok":true}`)
		return rec.Result(), nil
	}}

	status, err := c.Post(context.Background(), "demo", "start", "P18 started", "$2.10 · 3 min")
	testhelp.Equal(t, "Post status", status, Status{Sent: true, HTTPStatus: 200})
	testhelp.Equal(t, "Post error", err, nil)

	testhelp.Equal(t, "request method", recorded.Method, http.MethodPost)
	testhelp.Equal(t, "request URL", recorded.URL.String(), "https://api.telegram.org/bot123:ABC/sendMessage")
	testhelp.Equal(t, "request Content-Type", recorded.Header.Get("Content-Type"), "application/x-www-form-urlencoded")
	testhelp.Equal(t, "request body", string(body), "chat_id=-100&disable_web_page_preview=true&parse_mode=HTML&text=%5Bdemo%5D+%F0%9F%9A%80+%3Cb%3EP18+started%3C%2Fb%3E%0A%242.10+%C2%B7+3+min")
}

func TestMilestonePostExample5(t *testing.T) {
	var recorded *http.Request
	var body []byte
	c := Client{Token: "9:Z", ChatID: "42", Do: func(r *http.Request) (*http.Response, error) {
		recorded = r
		b, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatal(err)
		}
		body = b
		rec := httptest.NewRecorder()
		rec.WriteHeader(http.StatusOK)
		rec.WriteString(`{"ok":true}`)
		return rec.Result(), nil
	}}

	status, err := c.Post(context.Background(), "x", "ask", "a <b> & c", "")
	testhelp.Equal(t, "Post status", status, Status{Sent: true, HTTPStatus: 200})
	testhelp.Equal(t, "Post error", err, nil)

	testhelp.Equal(t, "request URL", recorded.URL.String(), "https://api.telegram.org/bot9:Z/sendMessage")
	testhelp.Equal(t, "request body", string(body), "chat_id=42&disable_web_page_preview=true&parse_mode=HTML&text=%5Bx%5D+%F0%9F%94%94+%3Cb%3Ea+%26lt%3Bb%26gt%3B+%26amp%3B+c%3C%2Fb%3E")
}

func TestMilestonePostExample6(t *testing.T) {
	called := false
	fail := func(r *http.Request) (*http.Response, error) {
		called = true
		t.Errorf("Do called for a skipped post")
		return nil, nil
	}

	clients := []Client{
		{Token: "", ChatID: "42", Do: fail},
		{Token: "t", ChatID: "", Do: fail},
	}
	for _, c := range clients {
		status, err := c.Post(context.Background(), "p", "info", "h", "n")
		testhelp.Equal(t, "Post status", status, Status{Skipped: true})
		testhelp.Equal(t, "Post error", err, nil)
	}
	testhelp.Equal(t, "Do called", called, false)
}

func TestMilestonePostExample7(t *testing.T) {
	c := Client{Token: "123:ABC", ChatID: "1", Do: func(r *http.Request) (*http.Response, error) {
		return &http.Response{
			StatusCode: http.StatusForbidden,
			Body:       io.NopCloser(strings.NewReader(`{"ok":false,"description":"Forbidden: bot was blocked"}`)),
		}, nil
	}}

	status, err := c.Post(context.Background(), "demo", "start", "h", "n")
	testhelp.Equal(t, "Post status", status, Status{HTTPStatus: 403})
	testhelp.Equal(t, "Post error", err.Error(), `telegram: status 403: {"ok":false,"description":"Forbidden: bot was blocked"}`)
}

func TestMilestonePostExample8(t *testing.T) {
	c := Client{Token: "123:ABC", ChatID: "1", Do: func(r *http.Request) (*http.Response, error) {
		return nil, errors.New(`Post "https://api.telegram.org/bot123:ABC/sendMessage": dial tcp: refused`)
	}}

	status, err := c.Post(context.Background(), "demo", "start", "h", "n")
	testhelp.Equal(t, "Post status", status, Status{})
	testhelp.Equal(t, "Post error", err.Error(), `telegram: post failed: Post "https://api.telegram.org/bot***/sendMessage": dial tcp: refused`)
}
