package api

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"morphstudio/auth"
	"morphstudio/internal/testhelp"
	"morphstudio/store"
)

func probeRouter() (http.Handler, store.Store) {
	s := store.NewMem()
	at := time.Date(2026, 10, 8, 9, 0, 0, 0, time.UTC)
	h := Handlers{Store: s, Now: func() time.Time { return at }, NewID: func() string { return "r-1" }}
	return NewRouter(h, auth.Users{"ann": auth.HashPassword("pw1", "x")}), s
}

func probeDo(r http.Handler, method, target, authz, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, target, strings.NewReader(body))
	if body != "" {
		req.Header.Set("Content-Type", "application/json")
	}
	if authz != "" {
		req.Header.Set("Authorization", authz)
	}
	rec := httptest.NewRecorder()
	r.ServeHTTP(rec, req)
	return rec
}

const probeAnn = "Basic YW5uOnB3MQ=="

func TestProbeRouterExample1(t *testing.T) {
	r, _ := probeRouter()
	rec := probeDo(r, "GET", "/healthz", "", "")
	testhelp.Equal(t, "healthz status", rec.Code, 200)
	testhelp.Equal(t, "healthz Content-Type", rec.Header().Get("Content-Type"), "application/json")
	testhelp.Equal(t, "healthz body", rec.Body.String(), `{"status":"ok"}`+"\n")
}

func TestProbeRouterExample2(t *testing.T) {
	r, s := probeRouter()
	for _, rec := range []*httptest.ResponseRecorder{
		probeDo(r, "GET", "/v1/projects", "", ""),
		probeDo(r, "POST", "/v1/projects", "Basic YW5uOndyb25n", `{"name":"A","slug":"a"}`),
	} {
		testhelp.Equal(t, "status", rec.Code, 401)
		testhelp.Equal(t, "WWW-Authenticate", rec.Header().Get("WWW-Authenticate"), `Basic realm="morph-studio"`)
		testhelp.Equal(t, "body", rec.Body.String(), `{"error":{"code":"unauthorized"}}`+"\n")
	}
	l, _ := s.List()
	testhelp.Equal(t, "store size", len(l), 0)
}

func TestProbeRouterExample3(t *testing.T) {
	r, _ := probeRouter()
	rec := probeDo(r, "POST", "/v1/projects", probeAnn, `{"name":"A","slug":"a"}`)
	testhelp.Equal(t, "POST status", rec.Code, 201)
	testhelp.Equal(t, "POST Location", rec.Header().Get("Location"), "/v1/projects/r-1")
	rec = probeDo(r, "GET", "/v1/projects", probeAnn, "")
	testhelp.Equal(t, "GET list status", rec.Code, 200)
	testhelp.Equal(t, "GET list body", strings.HasPrefix(rec.Body.String(), `{"items":[{"id":"r-1","name":"A"`), true)
	rec = probeDo(r, "GET", "/v1/projects/r-1", probeAnn, "")
	testhelp.Equal(t, "GET one status", rec.Code, 200)
	testhelp.Equal(t, "GET one name A", strings.Contains(rec.Body.String(), `"name":"A"`), true)
	rec = probeDo(r, "PUT", "/v1/projects/r-1", probeAnn, `{"name":"B","slug":"b"}`)
	testhelp.Equal(t, "PUT status", rec.Code, 200)
	testhelp.Equal(t, "PUT name B", strings.Contains(rec.Body.String(), `"name":"B"`), true)
	testhelp.Equal(t, "DELETE status", probeDo(r, "DELETE", "/v1/projects/r-1", probeAnn, "").Code, 204)
	testhelp.Equal(t, "GET after DELETE status", probeDo(r, "GET", "/v1/projects/r-1", probeAnn, "").Code, 404)
}

func TestProbeRouterExample4(t *testing.T) {
	r, _ := probeRouter()
	testhelp.Equal(t, "PATCH one", probeDo(r, "PATCH", "/v1/projects/r-1", probeAnn, "").Code, 405)
	testhelp.Equal(t, "DELETE collection", probeDo(r, "DELETE", "/v1/projects", probeAnn, "").Code, 405)
	testhelp.Equal(t, "POST healthz", probeDo(r, "POST", "/healthz", probeAnn, "").Code, 405)
	testhelp.Equal(t, "GET nowhere", probeDo(r, "GET", "/nowhere", probeAnn, "").Code, 404)
}
