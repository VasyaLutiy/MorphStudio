// Package api exposes the daemon's control surface over HTTP.
package api

import (
	"crypto/subtle"
	"net/http"
	"strings"
)

// Bearer wraps next so that every request must carry
// Authorization: "Bearer " + token. The scheme is matched
// case-insensitively and the token is compared in constant time. An empty
// configured token refuses every request the same way.
func Bearer(token string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !authorized(r, token) {
			WriteJSON(w, http.StatusUnauthorized,
				errorBody("unauthorized", "bearer token required"))
			return
		}
		next.ServeHTTP(w, r)
	})
}

// authorized reports whether r carries the configured bearer token.
func authorized(r *http.Request, token string) bool {
	if token == "" {
		return false
	}
	header := r.Header.Get("Authorization")
	const prefix = "Bearer "
	if len(header) < len(prefix) {
		return false
	}
	if !strings.EqualFold(header[:len(prefix)], prefix) {
		return false
	}
	got := header[len(prefix):]
	return subtle.ConstantTimeCompare([]byte(got), []byte(token)) == 1
}

// NewRouter builds the daemon's HTTP mux. Every route but GET /healthz and
// the MCP mount sits behind Bearer(token). A nil mcp leaves /mcp unmounted.
func NewRouter(h Handlers, token string, mcp http.Handler) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, _ *http.Request) {
		WriteJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})

	mux.Handle("GET /projects", Bearer(token, http.HandlerFunc(h.Projects)))
	mux.Handle("POST /projects", Bearer(token, http.HandlerFunc(h.CreateProject)))
	mux.Handle("POST /projects/adopt", Bearer(token, http.HandlerFunc(h.AdoptProject)))

	mux.Handle("GET /projects/{project}/status", Bearer(token, http.HandlerFunc(h.Status)))
	mux.Handle("GET /projects/{project}/events", Bearer(token, http.HandlerFunc(h.Events)))
	mux.Handle("POST /projects/{project}/messages", Bearer(token, http.HandlerFunc(h.Messages)))
	mux.Handle("GET /projects/{project}/pending", Bearer(token, http.HandlerFunc(h.Pending)))
	mux.Handle("POST /projects/{project}/answer", Bearer(token, http.HandlerFunc(h.Answer)))
	mux.Handle("POST /projects/{project}/interrupt", Bearer(token, http.HandlerFunc(h.Interrupt)))
	mux.Handle("GET /projects/{project}/usage", Bearer(token, http.HandlerFunc(h.Usage)))
	mux.Handle("POST /projects/{project}/restart", Bearer(token, http.HandlerFunc(h.Restart)))
	mux.Handle("POST /projects/{project}/plan", Bearer(token, http.HandlerFunc(h.Plan)))
	mux.Handle("POST /projects/{project}/continue", Bearer(token, http.HandlerFunc(h.Continue)))
	mux.Handle("GET /projects/{project}/stop", Bearer(token, http.HandlerFunc(h.StopCheck)))
	mux.Handle("PUT /projects/{project}/github-token", Bearer(token, http.HandlerFunc(h.GithubToken)))

	if mcp != nil {
		mux.Handle("/mcp", mcp)
		mux.Handle("/mcp/", mcp)
	}

	return mux
}
