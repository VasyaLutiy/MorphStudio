package registry

import (
	"errors"
	"time"

	"morphstudio/queue"
)

var ErrNotFound = errors.New("project not found")
var ErrExists = errors.New("project exists")
var ErrBadName = errors.New("bad project name")
var ErrNoState = errors.New("no state")

var errStub = errors.New("stub registry")

type Project struct {
	Name      string     `json:"name"`
	Language  string     `json:"language"`
	RepoURL   string     `json:"repo_url"`
	Dir       string     `json:"dir"`
	Caps      queue.Caps `json:"caps"`
	CreatedAt time.Time  `json:"created_at"`
	User      string     `json:"user,omitempty"`
}

type Registry struct {
	base     string
	projects []Project
}

func Open(base string) (*Registry, error)              { return nil, errStub }
func (r *Registry) Base() string                       { return "" }
func (r *Registry) Add(p Project) error                { return errStub }
func (r *Registry) Get(name string) (Project, error)   { return Project{}, errStub }
func (r *Registry) List() []Project                    { return nil }
func (r *Registry) ListUser(user string) []Project     { return nil }
func (r *Registry) SaveState(name string, v any) error { return errStub }
func (r *Registry) LoadState(name string, v any) error { return errStub }
