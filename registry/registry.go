// Package registry stores projects in one JSON file and a JSON state blob
// per project.
package registry

import (
	"bytes"
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

// ErrNotFound reports a missing project.
var ErrNotFound = errors.New("project not found")

// ErrExists reports a duplicate project name.
var ErrExists = errors.New("project exists")

// ErrBadName reports a project name that does not match the allowed shape.
var ErrBadName = errors.New("bad project name")

// ErrNoState reports a project without a state file.
var ErrNoState = errors.New("no state")

// Project is one registered project.
type Project struct {
	Name      string     `json:"name"`
	Language  string     `json:"language"`
	RepoURL   string     `json:"repo_url"`
	Dir       string     `json:"dir"`
	Caps      queue.Caps `json:"caps"`
	User      string     `json:"user"`
	CreatedAt time.Time  `json:"created_at"`
}

// Registry is an open project registry.
type Registry struct {
	base     string
	projects []Project
}

var nameRE = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{0,39}$`)

// Open opens the registry under base, creating base when missing.
func Open(base string) (*Registry, error) {
	if err := os.MkdirAll(base, 0o700); err != nil {
		return nil, err
	}
	r := &Registry{base: base}
	path := filepath.Join(base, "projects.json")
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return r, nil
		}
		return nil, err
	}
	trimmed := bytes.TrimSpace(data)
	if len(trimmed) == 0 || trimmed[0] != '[' {
		return nil, fmt.Errorf("registry: read %s: not a JSON array", path)
	}
	var projects []Project
	if err := json.Unmarshal(data, &projects); err != nil {
		return nil, fmt.Errorf("registry: read %s: %w", path, err)
	}
	r.projects = projects
	return r, nil
}

// Base returns the directory the registry lives in.
func (r *Registry) Base() string { return r.base }

// Add registers p.
func (r *Registry) Add(p Project) error {
	if !nameRE.MatchString(p.Name) {
		return ErrBadName
	}
	for _, x := range r.projects {
		if x.Name == p.Name {
			return ErrExists
		}
	}
	r.projects = append(r.projects, p)
	sort.Slice(r.projects, func(i, j int) bool { return r.projects[i].Name < r.projects[j].Name })
	return r.write()
}

// Get returns the project named name.
func (r *Registry) Get(name string) (Project, error) {
	for _, p := range r.projects {
		if p.Name == name {
			return p, nil
		}
	}
	return Project{}, ErrNotFound
}

// List returns every project sorted by name.
func (r *Registry) List() []Project {
	out := make([]Project, len(r.projects))
	copy(out, r.projects)
	sort.Slice(out, func(i, j int) bool { return out[i].Name < out[j].Name })
	return out
}

// SaveState writes v as the state blob of the project named name.
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
	tmp, err := os.CreateTemp(dir, "state-*.json.tmp")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmpName, filepath.Join(dir, "state.json"))
}

// LoadState reads the state blob of the project named name into v.
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

func (r *Registry) write() error {
	list := r.projects
	if list == nil {
		list = []Project{}
	}
	data, err := json.MarshalIndent(list, "", "  ")
	if err != nil {
		return err
	}
	tmp, err := os.CreateTemp(r.base, "projects-*.json.tmp")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)
	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	return os.Rename(tmpName, filepath.Join(r.base, "projects.json"))
}
