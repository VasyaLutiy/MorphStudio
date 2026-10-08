package auth

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"morphstudio/internal/testhelp"
)

const (
	probeHashS1   = "sha256$s1$f9b1214b8df1807692f471b9644de543bee606254884c13ba2e18f598eb2924d"
	probeHashNaCl = "sha256$NaCl$6bec8f041f5c92bd702205eb9e39eaae99a019caf941ca18646f37868c220533"
)

func probeUsers() Users {
	return Users{"alice": HashPassword("secret", "s1"), "dora": HashPassword("pa:ss", "NaCl")}
}

var probeNext = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("X-Next", "yes")
	w.WriteHeader(http.StatusNoContent)
})

func probeServe(h http.Handler, authz string) *httptest.ResponseRecorder {
	req := httptest.NewRequest("GET", "/v1/projects", nil)
	if authz != "" {
		req.Header.Set("Authorization", authz)
	}
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	return rec
}

func probe401(t *testing.T, rec *httptest.ResponseRecorder, what, realm string) {
	t.Helper()
	testhelp.Equal(t, what+" status", rec.Code, 401)
	testhelp.Equal(t, what+" WWW-Authenticate", rec.Header().Get("WWW-Authenticate"), `Basic realm="`+realm+`"`)
	testhelp.Equal(t, what+" Content-Type", rec.Header().Get("Content-Type"), "application/json")
	testhelp.Equal(t, what+" body", rec.Body.String(), `{"error":{"code":"unauthorized"}}`+"\n")
	testhelp.Equal(t, what+" X-Next", rec.Header().Get("X-Next"), "")
}

func TestProbeBasicAuthExample1(t *testing.T) {
	testhelp.Equal(t, `HashPassword("secret", "s1")`, HashPassword("secret", "s1"), probeHashS1)
	testhelp.Equal(t, `HashPassword("pa:ss", "NaCl")`, HashPassword("pa:ss", "NaCl"), probeHashNaCl)
	testhelp.Equal(t, `HashPassword("", "")`, HashPassword("", ""), "sha256$$e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855")
}

func TestProbeBasicAuthExample2(t *testing.T) {
	u := probeUsers()
	cases := []struct {
		user, pw string
		want     bool
	}{{"alice", "secret", true}, {"alice", "wrong", false}, {"bob", "secret", false}, {"dora", "pa:ss", true}, {"dora", "secret", false}}
	for _, c := range cases {
		testhelp.Equal(t, fmt.Sprintf("Verify(%q, %q)", c.user, c.pw), Verify(u, c.user, c.pw), c.want)
	}
}

func TestProbeBasicAuthExample3(t *testing.T) {
	u := Users{
		"carol": "md5$s1$f9b1214b8df1807692f471b9644de543bee606254884c13ba2e18f598eb2924d",
		"erin":  "sha256$s1",
		"fay":   "sha256$s1$F9B1214B8DF1807692F471B9644DE543BEE606254884C13BA2E18F598EB2924D",
	}
	for _, name := range []string{"carol", "erin", "fay"} {
		testhelp.Equal(t, fmt.Sprintf("Verify(%q, \"secret\")", name), Verify(u, name, "secret"), false)
	}
}

func TestProbeBasicAuthExample4(t *testing.T) {
	probe401(t, probeServe(Require(probeUsers(), "test-realm", probeNext), ""), "no Authorization", "test-realm")
}

func TestProbeBasicAuthExample5(t *testing.T) {
	h := Require(probeUsers(), "studio", probeNext)
	for _, a := range []string{"Bearer YWxpY2U6c2VjcmV0", "Basic !!!", "Basic YWxpY2U=", "Basic YWxpY2U6d3Jvbmc=", "Basic Ym9iOnNlY3JldA=="} {
		probe401(t, probeServe(h, a), a, "studio")
	}
}

func TestProbeBasicAuthExample6(t *testing.T) {
	h := Require(probeUsers(), "studio", probeNext)
	for _, a := range []string{"Basic YWxpY2U6c2VjcmV0", "basic YWxpY2U6c2VjcmV0", "Basic ZG9yYTpwYTpzcw=="} {
		rec := probeServe(h, a)
		testhelp.Equal(t, a+" status", rec.Code, 204)
		testhelp.Equal(t, a+" X-Next", rec.Header().Get("X-Next"), "yes")
		testhelp.Equal(t, a+" WWW-Authenticate", rec.Header().Get("WWW-Authenticate"), "")
	}
}
