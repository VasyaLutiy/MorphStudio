package api

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"morphstudio/internal/testhelp"
	"morphstudio/project"
	"morphstudio/store"
)

var (
	probeT1 = time.Date(2026, 10, 8, 9, 0, 0, 0, time.UTC)
	probeT2 = time.Date(2026, 10, 9, 10, 30, 0, 0, time.UTC)
)

const probeP1 = `{"id":"p-1","name":"Alpha","slug":"alpha","description":"","status":"draft","createdAt":"2026-10-08T09:00:00Z","updatedAt":"2026-10-08T09:00:00Z"}` + "\n"

func probeH(s store.Store, id string, at time.Time) Handlers {
	return Handlers{Store: s, Now: func() time.Time { return at }, NewID: func() string { return id }}
}

func probeServe(h func(http.ResponseWriter, *http.Request), method, target, id, ct, body string) *httptest.ResponseRecorder {
	req := httptest.NewRequest(method, target, strings.NewReader(body))
	if ct != "" {
		req.Header.Set("Content-Type", ct)
	}
	if id != "" {
		req.SetPathValue("id", id)
	}
	rec := httptest.NewRecorder()
	h(rec, req)
	return rec
}

func probeResp(t *testing.T, what string, rec *httptest.ResponseRecorder, code int, body string) {
	t.Helper()
	testhelp.Equal(t, what+" status", rec.Code, code)
	testhelp.Equal(t, what+" body", rec.Body.String(), body)
	testhelp.Equal(t, what+" Content-Type", rec.Header().Get("Content-Type"), "application/json")
}

func probeErr(code string) string { return `{"error":{"code":"` + code + `"}}` + "\n" }

func probeAfter2(t *testing.T) Handlers {
	t.Helper()
	h := probeH(store.NewMem(), "p-1", probeT1)
	rec := probeServe(h.Create, "POST", "/v1/projects", "", "application/json; charset=utf-8", `{"name":" Alpha ","slug":"alpha"}`)
	testhelp.Equal(t, "setup create status", rec.Code, 201)
	return h
}

func probeCount(t *testing.T, s store.Store) int {
	t.Helper()
	l, _ := s.List()
	return len(l)
}

func TestProbeProjectHandlersExample1(t *testing.T) {
	h := probeH(store.NewMem(), "p-1", probeT1)
	probeResp(t, "GET list", probeServe(h.List, "GET", "/v1/projects", "", "", ""), 200, `{"items":[]}`+"\n")
}

func TestProbeProjectHandlersExample2(t *testing.T) {
	h := probeH(store.NewMem(), "p-1", probeT1)
	rec := probeServe(h.Create, "POST", "/v1/projects", "", "application/json; charset=utf-8", `{"name":" Alpha ","slug":"alpha"}`)
	probeResp(t, "POST", rec, 201, probeP1)
	testhelp.Equal(t, "Location", rec.Header().Get("Location"), "/v1/projects/p-1")
	got, err := h.Store.Get("p-1")
	testhelp.Equal(t, "stored", got, project.Project{ID: "p-1", Name: "Alpha", Slug: "alpha", Status: "draft", CreatedAt: probeT1, UpdatedAt: probeT1})
	testhelp.Equal(t, "stored error", err, error(nil))
}

func TestProbeProjectHandlersExample3(t *testing.T) {
	h := probeH(store.NewMem(), "p-1", probeT1)
	for _, ct := range []string{"text/plain", "", "application/xml"} {
		probeResp(t, "POST with Content-Type "+ct, probeServe(h.Create, "POST", "/v1/projects", "", ct, `{"name":"a","slug":"a"}`), 415, probeErr("unsupported_media_type"))
	}
	testhelp.Equal(t, "store size", probeCount(t, h.Store), 0)
}

func probeBody(size int) string {
	head, tail := `{"name":"a","slug":"a","description":"`, `"}`
	return head + strings.Repeat("x", size-len(head)-len(tail)) + tail
}

func TestProbeProjectHandlersExample4(t *testing.T) {
	h := probeH(store.NewMem(), "p-1", probeT1)
	bodies := []string{`{"name":`, `{"name":"a","slug":"a","owner":"x"}`, `{"name":"a","slug":"a"} {}`, "", probeBody(1<<20 + 1)}
	for i, b := range bodies {
		probeResp(t, "bad body "+string(rune('0'+i)), probeServe(h.Create, "POST", "/v1/projects", "", "application/json", b), 400, probeErr("bad_json"))
	}
	probeResp(t, "1 MiB body", probeServe(h.Create, "POST", "/v1/projects", "", "application/json", probeBody(1<<20)), 422,
		`{"error":{"code":"validation","fields":{"description":"too long"}}}`+"\n")
	testhelp.Equal(t, "store size", probeCount(t, h.Store), 0)
}

func TestProbeProjectHandlersExample5(t *testing.T) {
	s := store.NewMem()
	_ = s.Create(project.Project{ID: "p-0", Slug: "taken"})
	h := probeH(s, "x9", time.Date(2025, 1, 2, 3, 4, 5, 0, time.UTC))
	probeResp(t, "invalid", probeServe(h.Create, "POST", "/v1/projects", "", "application/json", `{"name":"","slug":"Bad Slug","status":"x"}`), 422,
		`{"error":{"code":"validation","fields":{"name":"required","slug":"invalid","status":"invalid"}}}`+"\n")
	probeResp(t, "taken", probeServe(h.Create, "POST", "/v1/projects", "", "application/json", `{"name":"N","slug":"taken"}`), 409, probeErr("slug_taken"))
}

func TestProbeProjectHandlersExample6(t *testing.T) {
	h := probeAfter2(t)
	probeResp(t, "GET p-1", probeServe(h.Get, "GET", "/v1/projects/p-1", "p-1", "", ""), 200, probeP1)
	probeResp(t, "GET nope", probeServe(h.Get, "GET", "/v1/projects/nope", "nope", "", ""), 404, probeErr("not_found"))
}

func TestProbeProjectHandlersExample7(t *testing.T) {
	h := probeAfter2(t)
	_ = h.Store.Create(project.Project{ID: "p-2", Slug: "beta", CreatedAt: time.Date(2026, 10, 8, 9, 30, 0, 0, time.UTC)})
	h.Now = func() time.Time { return probeT2 }
	put := func(id, body string) *httptest.ResponseRecorder {
		return probeServe(h.Update, "PUT", "/v1/projects/"+id, id, "application/json", body)
	}
	probeResp(t, "PUT p-1", put("p-1", `{"name":"Alpha 2","slug":"alpha-2","description":"d","status":"active"}`), 200,
		`{"id":"p-1","name":"Alpha 2","slug":"alpha-2","description":"d","status":"active","createdAt":"2026-10-08T09:00:00Z","updatedAt":"2026-10-09T10:30:00Z"}`+"\n")
	probeResp(t, "PUT zz invalid", put("zz", `{"name":""}`), 404, probeErr("not_found"))
	probeResp(t, "PUT zz bad json", put("zz", `{"name":`), 400, probeErr("bad_json"))
	probeResp(t, "PUT p-1 slug beta", put("p-1", `{"name":"X","slug":"beta"}`), 409, probeErr("slug_taken"))
	rec := put("p-1", `{"name":"Y","slug":"alpha-2"}`)
	testhelp.Equal(t, "PUT own slug status", rec.Code, 200)
	testhelp.Equal(t, "PUT own slug name", strings.Contains(rec.Body.String(), `"name":"Y"`), true)
}

func TestProbeProjectHandlersExample8(t *testing.T) {
	h := probeAfter2(t)
	rec := probeServe(h.Delete, "DELETE", "/v1/projects/p-1", "p-1", "", "")
	testhelp.Equal(t, "DELETE status", rec.Code, 204)
	testhelp.Equal(t, "DELETE body", rec.Body.String(), "")
	testhelp.Equal(t, "DELETE Content-Type", rec.Header().Get("Content-Type"), "")
	_, err := h.Store.Get("p-1")
	testhelp.Equal(t, "Get after DELETE", err, store.ErrNotFound)
	probeResp(t, "DELETE again", probeServe(h.Delete, "DELETE", "/v1/projects/p-1", "p-1", "", ""), 404, probeErr("not_found"))
}

type probeFail struct{}

var errProbeDisk = errors.New("disk full")

func (probeFail) List() ([]project.Project, error)    { return nil, errProbeDisk }
func (probeFail) Get(string) (project.Project, error) { return project.Project{}, errProbeDisk }
func (probeFail) Create(project.Project) error        { return errProbeDisk }
func (probeFail) Update(project.Project) error        { return errProbeDisk }
func (probeFail) Delete(string) error                 { return errProbeDisk }

func TestProbeProjectHandlersExample9(t *testing.T) {
	h := probeH(probeFail{}, "p-1", probeT1)
	body := `{"name":"a","slug":"a"}`
	probeResp(t, "List", probeServe(h.List, "GET", "/v1/projects", "", "", ""), 500, probeErr("internal"))
	probeResp(t, "Get", probeServe(h.Get, "GET", "/v1/projects/p-1", "p-1", "", ""), 500, probeErr("internal"))
	probeResp(t, "Create", probeServe(h.Create, "POST", "/v1/projects", "", "application/json", body), 500, probeErr("internal"))
	probeResp(t, "Update", probeServe(h.Update, "PUT", "/v1/projects/p-1", "p-1", "application/json", body), 500, probeErr("internal"))
	probeResp(t, "Delete", probeServe(h.Delete, "DELETE", "/v1/projects/p-1", "p-1", "", ""), 500, probeErr("internal"))
}
