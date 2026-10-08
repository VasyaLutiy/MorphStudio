package auth

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"morphstudio/internal/testhelp"
)

func nextHandlerForExample(called *bool) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if called != nil {
			*called = true
		}
		w.Header().Set("X-Next", "yes")
		w.WriteHeader(http.StatusNoContent)
	})
}

func TestBasicAuthExample1(t *testing.T) {
	testhelp.Equal(t, "HashPassword(secret, s1)", HashPassword("secret", "s1"), "sha256$s1$f9b1214b8df1807692f471b9644de543bee606254884c13ba2e18f598eb2924d")
	testhelp.Equal(t, "HashPassword(pa:ss, NaCl)", HashPassword("pa:ss", "NaCl"), "sha256$NaCl$6bec8f041f5c92bd702205eb9e39eaae99a019caf941ca18646f37868c220533")
	testhelp.Equal(t, "HashPassword(\"\", \"\")", HashPassword("", ""), "sha256$$e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855")
}

func TestBasicAuthExample2(t *testing.T) {
	users := Users{
		"alice": HashPassword("secret", "s1"),
		"dora":  HashPassword("pa:ss", "NaCl"),
	}
	testhelp.Equal(t, "Verify alice secret", Verify(users, "alice", "secret"), true)
	testhelp.Equal(t, "Verify alice wrong", Verify(users, "alice", "wrong"), false)
	testhelp.Equal(t, "Verify bob secret", Verify(users, "bob", "secret"), false)
	testhelp.Equal(t, "Verify dora pa:ss", Verify(users, "dora", "pa:ss"), true)
	testhelp.Equal(t, "Verify dora secret", Verify(users, "dora", "secret"), false)
}

func TestBasicAuthExample3(t *testing.T) {
	users := Users{
		"carol": "md5$s1$f9b1214b8df1807692f471b9644de543bee606254884c13ba2e18f598eb2924d",
		"erin":  "sha256$s1",
		"fay":   "sha256$s1$F9B1214B8DF1807692F471B9644DE543BEE606254884C13BA2E18F598EB2924D",
	}
	testhelp.Equal(t, "Verify carol secret", Verify(users, "carol", "secret"), false)
	testhelp.Equal(t, "Verify erin secret", Verify(users, "erin", "secret"), false)
	testhelp.Equal(t, "Verify fay secret", Verify(users, "fay", "secret"), false)
}

func TestBasicAuthExample4(t *testing.T) {
	users := Users{
		"alice": HashPassword("secret", "s1"),
		"dora":  HashPassword("pa:ss", "NaCl"),
	}
	h := Require(users, "test-realm", nextHandlerForExample(nil))
	req := httptest.NewRequest(http.MethodGet, "/v1/projects", nil)
	rec := httptest.NewRecorder()
	h.ServeHTTP(rec, req)
	testhelp.Equal(t, "status", rec.Code, http.StatusUnauthorized)
	testhelp.Equal(t, "WWW-Authenticate", rec.Header().Get("WWW-Authenticate"), `Basic realm="test-realm"`)
	testhelp.Equal(t, "Content-Type", rec.Header().Get("Content-Type"), "application/json")
	testhelp.Equal(t, "body", rec.Body.String(), "{\"error\":{\"code\":\"unauthorized\"}}\n")
	testhelp.Equal(t, "X-Next", rec.Header().Get("X-Next"), "")
}

func TestBasicAuthExample5(t *testing.T) {
	users := Users{
		"alice": HashPassword("secret", "s1"),
		"dora":  HashPassword("pa:ss", "NaCl"),
	}
	called := false
	h := Require(users, "studio", nextHandlerForExample(&called))
	headers := []string{
		"Bearer YWxpY2U6c2VjcmV0",
		"Basic !!!",
		"Basic YWxpY2U=",
		"Basic YWxpY2U6d3Jvbmc=",
		"Basic Ym9iOnNlY3JldA==",
	}
	for i, hv := range headers {
		req := httptest.NewRequest(http.MethodGet, "/v1/projects", nil)
		req.Header.Set("Authorization", hv)
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		testhelp.Equal(t, "status", rec.Code, http.StatusUnauthorized)
		testhelp.Equal(t, "WWW-Authenticate", rec.Header().Get("WWW-Authenticate"), `Basic realm="studio"`)
		testhelp.Equal(t, "Content-Type", rec.Header().Get("Content-Type"), "application/json")
		testhelp.Equal(t, "body", rec.Body.String(), "{\"error\":{\"code\":\"unauthorized\"}}\n")
		testhelp.Equal(t, "X-Next", rec.Header().Get("X-Next"), "")
		_ = i
	}
	testhelp.Equal(t, "next called", called, false)
}

func TestBasicAuthExample6(t *testing.T) {
	users := Users{
		"alice": HashPassword("secret", "s1"),
		"dora":  HashPassword("pa:ss", "NaCl"),
	}
	called := false
	h := Require(users, "studio", nextHandlerForExample(&called))
	headers := []string{
		"Basic YWxpY2U6c2VjcmV0",
		"basic YWxpY2U6c2VjcmV0",
		"Basic ZG9yYTpwYTpzcw==",
	}
	for i, hv := range headers {
		req := httptest.NewRequest(http.MethodGet, "/v1/projects", nil)
		req.Header.Set("Authorization", hv)
		rec := httptest.NewRecorder()
		h.ServeHTTP(rec, req)
		testhelp.Equal(t, "status", rec.Code, http.StatusNoContent)
		testhelp.Equal(t, "X-Next", rec.Header().Get("X-Next"), "yes")
		testhelp.Equal(t, "WWW-Authenticate", rec.Header().Get("WWW-Authenticate"), "")
		_ = i
	}
	testhelp.Equal(t, "next called", called, true)
}
