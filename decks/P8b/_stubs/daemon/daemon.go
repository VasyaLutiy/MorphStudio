package daemon

import (
	"context"
	"errors"
	"net/http"
	"sync"
	"time"

	"morphstudio/claude"
	"morphstudio/control"
	"morphstudio/eventlog"
	"morphstudio/github"
	"morphstudio/queue"
	"morphstudio/registry"
	"morphstudio/runner"
	"morphstudio/session"
	"morphstudio/supervisor"
	"morphstudio/telegram"
)

type Deps struct {
	Registry *registry.Registry
	Runner   runner.Runner
	Spawn    func(ctx context.Context, l claude.Launch) (claude.Process, error)
	Post     func(ctx context.Context, project, kind, headline, numbers string) (telegram.Status, error)
	GitHubDo func(*http.Request) (*http.Response, error)
	Now      func() time.Time
	NewID    func() string
	Config   Config
}

type Config struct {
	ClaudeBin   string
	MorphBin    string
	ProjectsDir string
	MCPBaseURL  string
	Model       string
	ExtraArgs   []string
	Defaults    queue.Caps
	MaxParallel int
	DoneWait    time.Duration
	Loop        supervisor.Config
}

type State struct {
	Loop supervisor.Loop `json:"loop"`
	Repo github.Access   `json:"repo"`
}

type pendingDone struct {
	proc     claude.Process
	acts     []supervisor.Action
	deadline time.Time
}

type proj struct {
	reg      registry.Project
	state    State
	machine  *session.Machine
	proc     claude.Process
	log      *eventlog.Log
	queuedAt time.Time
	pumps    map[claude.Process]chan struct{}
	done     *pendingDone
}

type Daemon struct {
	deps     Deps
	ctx      context.Context
	mu       sync.Mutex
	projects map[string]*proj
}

var startPump func(d *Daemon, ctx context.Context, project string, p claude.Process, l *eventlog.Log)

var _ control.Control = (*Daemon)(nil)
var _ control.Adopter = (*Daemon)(nil)

var errStub = errors.New("stub daemon")

func Open(ctx context.Context, d Deps) (*Daemon, error) { return nil, errors.New("stub Open") }

func (d *Daemon) Projects() []control.ProjectView { return nil }
func (d *Daemon) CreateProject(ctx context.Context, name, language, repoURL string) (control.ProjectView, error) {
	return control.ProjectView{}, errStub
}
func (d *Daemon) AdoptProject(ctx context.Context, name, dir, repoURL, language string) (control.ProjectView, error) {
	return control.ProjectView{}, errStub
}
func (d *Daemon) Status(project string) (control.Status, error) { return control.Status{}, errStub }
func (d *Daemon) Events(project string, since int64, max int) (control.Events, error) {
	return control.Events{}, errStub
}
func (d *Daemon) Order(project, text string) (session.OrderResult, error) {
	return session.OrderResult{}, errStub
}
func (d *Daemon) Pending(project string) (*control.Question, error) { return nil, errStub }
func (d *Daemon) Answer(project string, option int) error           { return errStub }
func (d *Daemon) Interrupt(project string) error                    { return errStub }
func (d *Daemon) Usage(project string) (control.Usage, error)       { return control.Usage{}, errStub }
func (d *Daemon) Restart(project string, force bool) error          { return errStub }
func (d *Daemon) PlanLoad(project string, plan queue.Plan) (control.QueueView, error) {
	return control.QueueView{}, errStub
}
func (d *Daemon) Continue(project string) error                   { return errStub }
func (d *Daemon) StopCheck(project string) (*control.Stop, error) { return nil, errStub }
func (d *Daemon) PutGithubToken(ctx context.Context, project, token string) (control.TokenResult, error) {
	return control.TokenResult{}, errStub
}
func (d *Daemon) PhaseDone(project, sessionToken, phase, next string) error { return errStub }
func (d *Daemon) WaitOperator(project, sessionToken, kind, reason, issueURL string) error {
	return errStub
}
func (d *Daemon) Milestone(project, sessionToken string, m control.Milestone) error { return errStub }

func (d *Daemon) finishDone(ctx context.Context, project string, p claude.Process)      {}
func (d *Daemon) Execute(ctx context.Context, project string, acts []supervisor.Action) {}
func (d *Daemon) Tick(ctx context.Context)                                              {}
func (d *Daemon) execute(ctx context.Context, project string, acts []supervisor.Action) {}
func (d *Daemon) writeLine(project string, line []byte)                                 {}
