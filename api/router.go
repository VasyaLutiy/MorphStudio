package api

import (
	"net/http"

	"morphstudio/auth"
)

// NewRouter builds the HTTP routes for the API. The /v1/ routes are guarded
// by HTTP Basic authentication against users; /healthz is public.
func NewRouter(h Handlers, users auth.Users) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})

	guarded := func(next http.HandlerFunc) http.Handler {
		return auth.Require(users, "morph-studio", next)
	}

	mux.Handle("GET /v1/projects", guarded(h.List))
	mux.Handle("POST /v1/projects", guarded(h.Create))
	mux.Handle("GET /v1/projects/{id}", guarded(h.Get))
	mux.Handle("PUT /v1/projects/{id}", guarded(h.Update))
	mux.Handle("DELETE /v1/projects/{id}", guarded(h.Delete))

	return mux
}
