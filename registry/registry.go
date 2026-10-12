// Package registry stores the projects of one morphd instance in a single
// JSON file and a JSON state blob per project.
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

// ErrNotFound reports a project name the registry does not hold.
var ErrNotFound = errors.New("project not found")

// ErrExists reports a project name the registry already holds.
var ErrExists = errors.New("project exists")

// ErrBadName reports a project name that is not allowed.
var ErrBadName = errors.New("bad project name")

// ErrNoState reports that a project has no state file yet.
var ErrNoState = errors.New("no state")

// Project is one project of the registry.
type Project struct {
	Name      string     `json:"name"`
	Language  string     `json:"language"`
	RepoURL   string     `json:"repo_url"`
	Dir       string     `json:"dir"`
	Caps      queue.Caps `json:"caps"`
	CreatedAt time.Time  `json:"created_at"`
	User      string     `json:"user,omitempty"`
}

// Registry holds the projects of one base directory.
type Registry struct {
	base     string
	projects []Project
}

// nameRe is the rule a project name must match.
var nameRe = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{0,39}$`)

// Open opens the registry under base, creating the directory when missing.
func Open(base string) (*Registry, error) {
	if err := os.MkdirAll(base, 0o700); err != nil {
		return nil, err
	}
	r := &Registry{base: base, projects: []Project{}}
	path := filepath.Join(base, "projects.json")
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
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
	r.projects = ps
	return r, nil
}

// Base returns the directory the registry lives in.
func (r *Registry) Base() string {
	return r.base
}

// Add stores a new project and rewrites the registry file.
func (r *Registry) Add(p Project) error {
	if !nameRe.MatchString(p.Name) {
		return ErrBadName
	}
	for _, e := range r.projects {
		if e.Name == p.Name {
			return ErrExists
		}
	}
	r.projects = append(r.projects, p)
	sort.Slice(r.projects, func(i, j int) bool {
		return r.projects[i].Name < r.projects[j].Name
	})
	return r.write()
}

// Get returns the named project.
func (r *Registry) Get(name string) (Project, error) {
	for _, p := range r.projects {
		if p.Name == name {
			return p, nil
		}
	}
	return Project{}, ErrNotFound
}

// List returns a copy of every project, sorted by name.
func (r *Registry) List() []Project {
	out := make([]Project, len(r.projects))
	copy(out, r.projects)
	return out
}

// ListUser returns a copy of the user's projects, sorted by name.
func (r *Registry) ListUser(user string) []Project {
	out := []Project{}
	for _, p := range r.projects {
		if p.User == user {
			out = append(out, p)
		}
	}
	return out
}

// write rewrites projects.json atomically.
func (r *Registry) write() error {
	ps := r.projects
	if ps == nil {
		ps = []Project{}
	}
	data, err := json.MarshalIndent(ps, "", "  ")
	if err != nil {
		return err
	}
	tmp, err := os.CreateTemp(r.base, ".projects-*.tmp")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		os.Remove(tmpName)
		return err
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmpName)
		return err
	}
	if err := os.Rename(tmpName, filepath.Join(r.base, "projects.json")); err != nil {
		os.Remove(tmpName)
		return err
	}
	return nil
}

// SaveState writes the project's state blob.
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
	tmp, err := os.CreateTemp(dir, ".state-*.tmp")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		os.Remove(tmpName)
		return err
	}
	if err := tmp.Close(); err != nil {
		os.Remove(tmpName)
		return err
	}
	if err := os.Rename(tmpName, filepath.Join(dir, "state.json")); err != nil {
		os.Remove(tmpName)
		return err
	}
	return nil
}

// LoadState reads the project's state blob.
func (r *Registry) LoadState(name string, v any) error {
	if _, err := r.Get(name); err != nil {
		return err
	}
	path := filepath.Join(r.base, name, "state.json")
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return ErrNoState
		}
		return err
	}
	return json.Unmarshal(data, v)
}
