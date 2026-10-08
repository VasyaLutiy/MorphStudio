package api

import (
	"encoding/json"
	"errors"
	"io"
	"mime"
	"net/http"
	"time"

	"morphstudio/project"
	"morphstudio/store"
)

// Handlers exposes the CRUD HTTP handlers over an injected store, clock and id
// source.
type Handlers struct {
	Store store.Store
	Now   func() time.Time
	NewID func() string
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, code string) {
	writeJSON(w, status, map[string]any{
		"error": map[string]string{"code": code},
	})
}

func writeValidation(w http.ResponseWriter, fields map[string]string) {
	writeJSON(w, http.StatusUnprocessableEntity, map[string]any{
		"error": map[string]any{
			"code":   "validation",
			"fields": fields,
		},
	})
}

// readInput enforces the Content-Type and body rules for Create and Update. It
// writes the appropriate error response and returns ok=false on failure.
func readInput(w http.ResponseWriter, r *http.Request) (project.Input, bool) {
	mediaType, _, err := mime.ParseMediaType(r.Header.Get("Content-Type"))
	if err != nil || mediaType != "application/json" {
		writeError(w, http.StatusUnsupportedMediaType, "unsupported_media_type")
		return project.Input{}, false
	}

	body := http.MaxBytesReader(w, r.Body, 1<<20)
	dec := json.NewDecoder(body)
	dec.DisallowUnknownFields()

	var in project.Input
	if err := dec.Decode(&in); err != nil {
		writeError(w, http.StatusBadRequest, "bad_json")
		return project.Input{}, false
	}

	var extra any
	if err := dec.Decode(&extra); err != io.EOF {
		writeError(w, http.StatusBadRequest, "bad_json")
		return project.Input{}, false
	}

	return in, true
}

// List handles GET /v1/projects.
func (h Handlers) List(w http.ResponseWriter, r *http.Request) {
	items, err := h.Store.List()
	if err != nil {
		writeError(w, http.StatusInternalServerError, "internal")
		return
	}
	if items == nil {
		items = []project.Project{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": items})
}

// Create handles POST /v1/projects.
func (h Handlers) Create(w http.ResponseWriter, r *http.Request) {
	in, ok := readInput(w, r)
	if !ok {
		return
	}

	norm, errs := project.Validate(in)
	if len(errs) > 0 {
		writeValidation(w, errs)
		return
	}

	p := project.New(norm, h.NewID(), h.Now())
	if err := h.Store.Create(p); err != nil {
		if errors.Is(err, store.ErrSlugTaken) {
			writeError(w, http.StatusConflict, "slug_taken")
			return
		}
		writeError(w, http.StatusInternalServerError, "internal")
		return
	}

	w.Header().Set("Location", "/v1/projects/"+p.ID)
	writeJSON(w, http.StatusCreated, p)
}

// Get handles GET /v1/projects/{id}.
func (h Handlers) Get(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	p, err := h.Store.Get(id)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			writeError(w, http.StatusNotFound, "not_found")
			return
		}
		writeError(w, http.StatusInternalServerError, "internal")
		return
	}
	writeJSON(w, http.StatusOK, p)
}

// Update handles PUT /v1/projects/{id}.
func (h Handlers) Update(w http.ResponseWriter, r *http.Request) {
	in, ok := readInput(w, r)
	if !ok {
		return
	}

	id := r.PathValue("id")
	existing, err := h.Store.Get(id)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			writeError(w, http.StatusNotFound, "not_found")
			return
		}
		writeError(w, http.StatusInternalServerError, "internal")
		return
	}

	norm, errs := project.Validate(in)
	if len(errs) > 0 {
		writeValidation(w, errs)
		return
	}

	p := project.Apply(existing, norm, h.Now())
	if err := h.Store.Update(p); err != nil {
		switch {
		case errors.Is(err, store.ErrNotFound):
			writeError(w, http.StatusNotFound, "not_found")
		case errors.Is(err, store.ErrSlugTaken):
			writeError(w, http.StatusConflict, "slug_taken")
		default:
			writeError(w, http.StatusInternalServerError, "internal")
		}
		return
	}

	writeJSON(w, http.StatusOK, p)
}

// Delete handles DELETE /v1/projects/{id}.
func (h Handlers) Delete(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if err := h.Store.Delete(id); err != nil {
		if errors.Is(err, store.ErrNotFound) {
			writeError(w, http.StatusNotFound, "not_found")
			return
		}
		writeError(w, http.StatusInternalServerError, "internal")
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
