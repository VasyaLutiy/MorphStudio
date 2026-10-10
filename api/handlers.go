// Package api exposes the daemon's control surface over HTTP.
package api

import (
	"encoding/json"
	"io"
	"mime"
	"net/http"
	"strconv"

	"morphstudio/control"
	"morphstudio/queue"
)

// Handlers turns the control surface into HTTP handlers.
type Handlers struct {
	Control control.Control
}

// WriteJSON writes v as an application/json body ending in a newline.
func WriteJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

// WriteError writes the stable error body for err.
func WriteError(w http.ResponseWriter, err error) {
	status, code := control.Code(err)
	msg := err.Error()
	if status == http.StatusInternalServerError {
		msg = "internal"
	}
	WriteJSON(w, status, errorBody(code, msg))
}

// errorBody builds the {"error":{...}} envelope.
func errorBody(code, message string) map[string]any {
	return map[string]any{
		"error": map[string]any{
			"code":    code,
			"message": message,
		},
	}
}

// okBody is the {"ok":true} acknowledgement.
func okBody() map[string]bool {
	return map[string]bool{"ok": true}
}

// readJSON enforces the JSON reading rules and decodes into v. It writes the
// error response and reports false when the body is unacceptable.
func readJSON(w http.ResponseWriter, r *http.Request, v any) bool {
	mt, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || mt != "application/json" {
		WriteJSON(w, http.StatusUnsupportedMediaType, errorBody(
			"unsupported_media_type", "Content-Type must be application/json"))
		return false
	}
	r.Body = http.MaxBytesReader(w, r.Body, 1<<20)
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	if err := dec.Decode(v); err != nil {
		WriteJSON(w, http.StatusBadRequest, errorBody("bad_json", "bad json"))
		return false
	}
	var extra any
	if err := dec.Decode(&extra); err != io.EOF {
		WriteJSON(w, http.StatusBadRequest, errorBody("bad_json", "bad json"))
		return false
	}
	return true
}

// Projects answers GET /projects.
func (h Handlers) Projects(w http.ResponseWriter, r *http.Request) {
	projects := h.Control.Projects()
	if projects == nil {
		projects = []control.ProjectView{}
	}
	WriteJSON(w, http.StatusOK, map[string]any{"projects": projects})
}

type createProjectRequest struct {
	Name     string `json:"name"`
	Language string `json:"language"`
	RepoURL  string `json:"repo_url"`
}

// CreateProject answers POST /projects.
func (h Handlers) CreateProject(w http.ResponseWriter, r *http.Request) {
	var req createProjectRequest
	if !readJSON(w, r, &req) {
		return
	}
	view, err := h.Control.CreateProject(r.Context(), req.Name, req.Language, req.RepoURL)
	if err != nil {
		WriteError(w, err)
		return
	}
	WriteJSON(w, http.StatusCreated, view)
}

type adoptProjectRequest struct {
	Name     string `json:"name"`
	Dir      string `json:"dir"`
	RepoURL  string `json:"repo_url"`
	Language string `json:"language"`
}

// AdoptProject answers POST /projects/adopt.
func (h Handlers) AdoptProject(w http.ResponseWriter, r *http.Request) {
	var req adoptProjectRequest
	if !readJSON(w, r, &req) {
		return
	}
	view, err := control.Adopt(r.Context(), h.Control, req.Name, req.Dir, req.RepoURL, req.Language)
	if err != nil {
		WriteError(w, err)
		return
	}
	WriteJSON(w, http.StatusCreated, view)
}

// Status answers GET /projects/{project}/status.
func (h Handlers) Status(w http.ResponseWriter, r *http.Request) {
	status, err := h.Control.Status(r.PathValue("project"))
	if err != nil {
		WriteError(w, err)
		return
	}
	WriteJSON(w, http.StatusOK, status)
}

// Events answers GET /projects/{project}/events.
func (h Handlers) Events(w http.ResponseWriter, r *http.Request) {
	query := r.URL.Query()
	var since int64
	if raw := query.Get("since"); raw != "" {
		n, err := strconv.ParseInt(raw, 10, 64)
		if err != nil {
			WriteJSON(w, http.StatusBadRequest, errorBody("bad_input", "since must be an integer"))
			return
		}
		since = n
	}
	var max int
	if raw := query.Get("max"); raw != "" {
		n, err := strconv.Atoi(raw)
		if err != nil {
			WriteJSON(w, http.StatusBadRequest, errorBody("bad_input", "max must be an integer"))
			return
		}
		max = n
	}
	events, err := h.Control.Events(r.PathValue("project"), since, max)
	if err != nil {
		WriteError(w, err)
		return
	}
	WriteJSON(w, http.StatusOK, events)
}

type messageRequest struct {
	Text string `json:"text"`
}

// Messages answers POST /projects/{project}/messages.
func (h Handlers) Messages(w http.ResponseWriter, r *http.Request) {
	var req messageRequest
	if !readJSON(w, r, &req) {
		return
	}
	result, err := h.Control.Order(r.PathValue("project"), req.Text)
	if err != nil {
		WriteError(w, err)
		return
	}
	WriteJSON(w, http.StatusOK, result)
}

// Pending answers GET /projects/{project}/pending.
func (h Handlers) Pending(w http.ResponseWriter, r *http.Request) {
	question, err := h.Control.Pending(r.PathValue("project"))
	if err != nil {
		WriteError(w, err)
		return
	}
	WriteJSON(w, http.StatusOK, map[string]any{"question": question})
}

type answerRequest struct {
	Option int `json:"option"`
}

// Answer answers POST /projects/{project}/answer.
func (h Handlers) Answer(w http.ResponseWriter, r *http.Request) {
	var req answerRequest
	if !readJSON(w, r, &req) {
		return
	}
	if err := h.Control.Answer(r.PathValue("project"), req.Option); err != nil {
		WriteError(w, err)
		return
	}
	WriteJSON(w, http.StatusOK, okBody())
}

// Interrupt answers POST /projects/{project}/interrupt.
func (h Handlers) Interrupt(w http.ResponseWriter, r *http.Request) {
	if err := h.Control.Interrupt(r.PathValue("project")); err != nil {
		WriteError(w, err)
		return
	}
	WriteJSON(w, http.StatusOK, okBody())
}

// Usage answers GET /projects/{project}/usage.
func (h Handlers) Usage(w http.ResponseWriter, r *http.Request) {
	usage, err := h.Control.Usage(r.PathValue("project"))
	if err != nil {
		WriteError(w, err)
		return
	}
	WriteJSON(w, http.StatusOK, usage)
}

type restartRequest struct {
	Force bool `json:"force"`
}

// Restart answers POST /projects/{project}/restart. The body is optional: it
// is read only when a body is present.
func (h Handlers) Restart(w http.ResponseWriter, r *http.Request) {
	req := restartRequest{}
	if r.ContentLength != 0 {
		if !readJSON(w, r, &req) {
			return
		}
	}
	if err := h.Control.Restart(r.PathValue("project"), req.Force); err != nil {
		WriteError(w, err)
		return
	}
	WriteJSON(w, http.StatusOK, okBody())
}

// Plan answers POST /projects/{project}/plan.
func (h Handlers) Plan(w http.ResponseWriter, r *http.Request) {
	var plan queue.Plan
	if !readJSON(w, r, &plan) {
		return
	}
	view, err := h.Control.PlanLoad(r.PathValue("project"), plan)
	if err != nil {
		WriteError(w, err)
		return
	}
	WriteJSON(w, http.StatusOK, view)
}

// Continue answers POST /projects/{project}/continue.
func (h Handlers) Continue(w http.ResponseWriter, r *http.Request) {
	if err := h.Control.Continue(r.PathValue("project")); err != nil {
		WriteError(w, err)
		return
	}
	WriteJSON(w, http.StatusOK, okBody())
}

// StopCheck answers GET /projects/{project}/stop.
func (h Handlers) StopCheck(w http.ResponseWriter, r *http.Request) {
	stop, err := h.Control.StopCheck(r.PathValue("project"))
	if err != nil {
		WriteError(w, err)
		return
	}
	WriteJSON(w, http.StatusOK, map[string]any{"stop": stop})
}

type githubTokenRequest struct {
	Token string `json:"token"`
}

// GithubToken answers PUT /projects/{project}/github-token.
func (h Handlers) GithubToken(w http.ResponseWriter, r *http.Request) {
	var req githubTokenRequest
	if !readJSON(w, r, &req) {
		return
	}
	result, err := h.Control.PutGithubToken(r.Context(), r.PathValue("project"), req.Token)
	if err != nil {
		WriteError(w, err)
		return
	}
	WriteJSON(w, http.StatusOK, result)
}
