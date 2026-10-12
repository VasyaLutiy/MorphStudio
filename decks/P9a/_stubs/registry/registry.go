package registry

import (
	"errors"
	"time"

	"morphstudio/queue"
)

// ErrNotFound is returned when a project is not stored.
var ErrNotFound = errors.New("project not found")

// ErrExists is returned when a project name is already stored.
var ErrExists = errors.New("project exists")

// ErrBadName is returned when a project name is not acceptable.
var ErrBadName = errors.New("bad project name")

// ErrNoState is returned when a project has no stored state blob.
var ErrNoState = errors.New("no state")

// Project is one registered project.
type Project struct {
	Name      string     `json:"name"`
	Language  string     `json:"language"`
	RepoURL   string     `json:"repo_url"`
	Dir       string     `json:"dir"`
	User      string     `json:"user"`
	Caps      queue.Caps `json:"caps"`
	CreatedAt time.Time  `json:"created_at"`
}

// Registry is the project registry rooted at a base directory.
type Registry struct {
	base     string
	projects []Project
}

var errStubProjectRegistry = errors.New("stub project registry")

// Open creates base when missing and reads its projects.json.
func Open(base string) (*Registry, error) { return nil, errStubProjectRegistry }

// Base returns the directory the registry lives in.
func (r *Registry) Base() string { return "" }

// Add validates and stores a project, rewriting the whole list.
func (r *Registry) Add(p Project) error { return errStubProjectRegistry }

// Get returns the stored project with the given name.
func (r *Registry) Get(name string) (Project, error) { return Project{}, errStubProjectRegistry }

// List returns a new sorted copy of the stored projects.
func (r *Registry) List() []Project { return nil }

// SaveState writes a project's state blob.
func (r *Registry) SaveState(name string, v any) error { return errStubProjectRegistry }

// LoadState reads a project's state blob into v.
func (r *Registry) LoadState(name string, v any) error { return errStubProjectRegistry }
