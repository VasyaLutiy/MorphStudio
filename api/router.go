package api

import (
	"crypto/subtle"
	"net/http"
	"strings"
)

// Bearer guards next with a static bearer token. The scheme is compared
// case-insensitively; the token is compared in constant time. An empty
// configured token refuses every request.
func Bearer(token string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !authorized(token, r.Header.Get("Authorization")) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusUnauthorized)
			_, _ = w.Write([]byte(`{"error":{"code":"unauthorized","message":"bearer token required"}}` + "\n"))
			return
		}
		next.ServeHTTP(w, r)
	})
}

func authorized(token, header string) bool {
	const prefix = "Bearer "
	if len(header) < len(prefix) || !strings.EqualFold(header[:len(prefix)], prefix) {
		return false
	}
	if token == "" {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(header[len(prefix):]), []byte(token)) == 1
}

// NewRouter builds the HTTP surface: every route but GET /healthz and the MCP
// mount sits behind Bearer.
func NewRouter(h Handlers, token string, mcp http.Handler) http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		WriteJSON(w, http.StatusOK, struct {
			Status string `json:"status"`
		}{Status: "ok"})
	})

	guarded := func(next http.HandlerFunc) http.Handler {
		return Bearer(token, next)
	}

	mux.Handle("GET /projects", guarded(h.Projects))
	mux.Handle("POST /projects", guarded(h.CreateProject))
	mux.Handle("GET /projects/{project}/status", guarded(h.Status))
	mux.Handle("GET /projects/{project}/events", guarded(h.Events))
	mux.Handle("POST /projects/{project}/messages", guarded(h.Messages))
	mux.Handle("GET /projects/{project}/pending", guarded(h.Pending))
	mux.Handle("POST /projects/{project}/answer", guarded(h.Answer))
	mux.Handle("POST /projects/{project}/interrupt", guarded(h.Interrupt))
	mux.Handle("GET /projects/{project}/usage", guarded(h.Usage))
	mux.Handle("POST /projects/{project}/restart", guarded(h.Restart))
	mux.Handle("POST /projects/{project}/plan", guarded(h.Plan))
	mux.Handle("POST /projects/{project}/continue", guarded(h.Continue))
	mux.Handle("GET /projects/{project}/stop", guarded(h.StopCheck))
	mux.Handle("PUT /projects/{project}/github-token", guarded(h.GithubToken))

	if mcp != nil {
		mux.Handle("/mcp", mcp)
		mux.Handle("/mcp/", mcp)
	}

	return mux
}
