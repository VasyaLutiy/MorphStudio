package registry

import (
	"errors"
	"regexp"
	"sync"
	"time"

	"morphstudio/queue"
)

var (
	ErrNotFound = errors.New("project not found")
	ErrExists   = errors.New("project exists")
	ErrBadName  = errors.New("bad project name")
	ErrNoState  = errors.New("no state")
)

var nameRE = regexp.MustCompile(`^[a-z0-9][a-z0-9-]{0,39}$`)

type Project struct {
	Name      string     `json:"name"`
	Language  string     `json:"language"`
	RepoURL   string     `json:"repo_url"`
	Dir       string     `json:"dir"`
	Caps      queue.Caps `json:"caps"`
	User      string     `json:"user"`
	CreatedAt time.Time  `json:"created_at"`
}

type Registry struct {
	mu    sync.Mutex
	base  string
	items []Project
}

func Open(base string) (*Registry, error) {
	panic("stub Open")
}

func (r *Registry) Base() string {
	panic("stub Base")
}

func (r *Registry) Add(p Project) error {
	panic("stub Add")
}

func (r *Registry) Get(name string) (Project, error) {
	panic("stub Get")
}

func (r *Registry) List() []Project {
	panic("stub List")
}

func (r *Registry) SaveState(name string, v any) error {
	panic("stub SaveState")
}

func (r *Registry) LoadState(name string, v any) error {
	panic("stub LoadState")
}

func (r *Registry) write() error {
	panic("stub write")
}
