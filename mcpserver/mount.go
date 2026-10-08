package mcpserver

import (
	"crypto/subtle"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"

	"morphstudio/control"
)

// Tokens carries the bearer tokens the MCP mount checks: the daemon API
// token and a lookup for a project's current session token.
type Tokens struct {
	API     string
	Session func(project string) (string, bool)
}

// errorBody is the fixed JSON body of a 401 or 404 answer.
type errorBody struct {
	Error errorDetail `json:"error"`
}

// errorDetail is the code and message of an errorBody.
type errorDetail struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

// Handler returns the HTTP mount of the three MCP endpoints.
func Handler(c control.Control, known func(project string) bool, t Tokens) http.Handler {
	mux := http.NewServeMux()

	options := &mcp.StreamableHTTPOptions{Stateless: true, JSONResponse: true}

	userH := mcp.NewStreamableHTTPHandler(func(*http.Request) *mcp.Server {
		return UserServer(c)
	}, options)

	pmH := mcp.NewStreamableHTTPHandler(func(r *http.Request) *mcp.Server {
		return PMServer(c, r.PathValue("project"))
	}, options)

	sessionH := mcp.NewStreamableHTTPHandler(func(r *http.Request) *mcp.Server {
		project := r.PathValue("project")
		token, _ := t.Session(project)
		return SessionServer(c, project, token)
	}, options)

	mux.Handle("POST /mcp", requireBearer(t.API, userH))

	mux.Handle("POST /mcp/{project}", requireBearer(t.API, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !known(r.PathValue("project")) {
			writeError(w, http.StatusNotFound, "unknown_project", "unknown project")
			return
		}
		pmH.ServeHTTP(w, r)
	})))

	mux.Handle("POST /mcp/{project}/session", http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		project := r.PathValue("project")
		token, ok := t.Session(project)
		if !ok || !checkBearer(r, token) {
			writeError(w, http.StatusUnauthorized, "unauthorized", "bearer token required")
			return
		}
		if !known(project) {
			writeError(w, http.StatusNotFound, "unknown_project", "unknown project")
			return
		}
		sessionH.ServeHTTP(w, r)
	}))

	return mux
}

// requireBearer wraps next so a missing or mismatched bearer token is a 401.
func requireBearer(want string, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !checkBearer(r, want) {
			writeError(w, http.StatusUnauthorized, "unauthorized", "bearer token required")
			return
		}
		next.ServeHTTP(w, r)
	})
}

// checkBearer reports whether r carries exactly want as its bearer token. An
// empty want refuses; the comparison is constant time and untrimmed.
func checkBearer(r *http.Request, want string) bool {
	if want == "" {
		return false
	}
	h := r.Header.Get("Authorization")
	if len(h) < 7 || !strings.EqualFold(h[:7], "Bearer ") {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(h[7:]), []byte(want)) == 1
}

// writeError writes status, a JSON content type and one fixed error body.
func writeError(w http.ResponseWriter, status int, code, message string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(errorBody{Error: errorDetail{Code: code, Message: message}})
}
