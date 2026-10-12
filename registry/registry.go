// Package registry stores the projects morphd knows about. The set is kept
// in one JSON file that is never edited in place; a JSON state blob can be
// stored per project.
package registry

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"time"

	"morphstudio/queue"
)

// Sentinel errors returned by the registry.
var (
	ErrNotFound = errors.New("project not found")
	ErrExists   = errors.New("project exists")
	ErrBadName  = errors.New("bad project name")
	ErrNoState  = errors.New("no state")
)

// Project describes one project morphd manages.
type Project struct {
	Name      string     `json:"name"`
	Language  string     `json:"language"`
	RepoURL   string     `json:"repo_url"`
	Dir       string     `json:"dir"`
	User      string     `json:"user"`
	Caps      queue.Caps `json:"caps"`
	CreatedAt time.Time  `json:"created_at"`
}

// nameRE is the only shape a project name may take.
var nameRE = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{0,39}$`)

// Registry is the set of known projects backed by a JSON file.
type Registry struct {
	base     string
	projects []Project
}

// Open loads the registry under base, creating base when it is missing.
func Open(base string) (*Registry, error) {
	if err := os.MkdirAll(base, 0o700); err != nil {
		return nil, err
	}
	r := &Registry{base: base, projects: []Project{}}
	path := filepath.Join(base, "projects.json")
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return r, nil
		}
		return nil, err
	}
	var ps []Project
	if err := json.Unmarshal(data, &ps); err != nil {
		return nil, fmt.Errorf("registry: read %s: %w", path, err)
	}
	if ps == nil {
		ps = []Project{}
	}
	sort.Slice(ps, func(i, j int) bool { return ps[i].Name < ps[j].Name })
	r.projects = ps
	return r, nil
}

// Base returns the directory the registry lives in.
func (r *Registry) Base() string { return r.base }

// Add stores a project, rejecting an unknown name shape and duplicates.
func (r *Registry) Add(p Project) error {
	if !nameRE.MatchString(p.Name) {
		return ErrBadName
	}
	for _, ex := range r.projects {
		if ex.Name == p.Name {
			return ErrExists
		}
	}
	next := make([]Project, 0, len(r.projects)+1)
	next = append(next, r.projects...)
	next = append(next, p)
	sort.Slice(next, func(i, j int) bool { return next[i].Name < next[j].Name })
	if err := r.write(next); err != nil {
		return err
	}
	r.projects = next
	return nil
}

// Get returns the named project or ErrNotFound.
func (r *Registry) Get(name string) (Project, error) {
	for _, p := range r.projects {
		if p.Name == name {
			return p, nil
		}
	}
	return Project{}, ErrNotFound
}

// List returns a fresh copy of the projects, sorted by name.
func (r *Registry) List() []Project {
	out := make([]Project, len(r.projects))
	copy(out, r.projects)
	return out
}

// SaveState writes a JSON state blob for a known project.
func (r *Registry) SaveState(name string, v any) error {
	if _, err := r.Get(name); err != nil {
		return err
	}
	dir := filepath.Join(r.base, name)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	return writeAtomic(filepath.Join(dir, "state.json"), data)
}

// LoadState reads a project's JSON state blob into v.
func (r *Registry) LoadState(name string, v any) error {
	if _, err := r.Get(name); err != nil {
		return err
	}
	path := filepath.Join(r.base, name, "state.json")
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return ErrNoState
		}
		return err
	}
	return json.Unmarshal(data, v)
}

// write replaces projects.json with the whole sorted list.
func (r *Registry) write(ps []Project) error {
	if ps == nil {
		ps = []Project{}
	}
	data, err := json.MarshalIndent(ps, "", "  ")
	if err != nil {
		return err
	}
	return writeAtomic(filepath.Join(r.base, "projects.json"), data)
}

// writeAtomic writes data to a temporary file beside path and renames it
// onto path, so a reader never sees a partial file.
func writeAtomic(path string, data []byte) error {
	dir := filepath.Dir(path)
	f, err := os.CreateTemp(dir, ".tmp-*")
	if err != nil {
		return err
	}
	tmp := f.Name()
	if _, err := f.Write(data); err != nil {
		f.Close()
		os.Remove(tmp)
		return err
	}
	if err := f.Chmod(0o600); err != nil {
		f.Close()
		os.Remove(tmp)
		return err
	}
	if err := f.Close(); err != nil {
		os.Remove(tmp)
		return err
	}
	if err := os.Rename(tmp, path); err != nil {
		os.Remove(tmp)
		return err
	}
	return nil
}
