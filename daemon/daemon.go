package daemon

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"path/filepath"
	"sort"
	"sync"
	"time"

	"morphstudio/bootstrap"
	"morphstudio/claude"
	"morphstudio/control"
	"morphstudio/eventlog"
	"morphstudio/github"
	"morphstudio/gitrules"
	"morphstudio/queue"
	"morphstudio/registry"
	"morphstudio/runner"
	"morphstudio/session"
	"morphstudio/supervisor"
	"morphstudio/telegram"
)

// Deps collects the daemon's dependencies.
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

// Config holds the daemon's fixed configuration.
type Config struct {
	ClaudeBin   string
	MorphBin    string
	ProjectsDir string
	MCPBaseURL  string
	Model       string
	ExtraArgs   []string
	Defaults    queue.Caps
	MaxParallel int
	Loop        supervisor.Config
}

// State is the per-project persisted state.
type State struct {
	Loop supervisor.Loop `json:"loop"`
	Repo github.Access   `json:"repo"`
}

type proj struct {
	reg      registry.Project
	state    State
	machine  *session.Machine
	proc     claude.Process
	log      *eventlog.Log
	queuedAt time.Time
	pumps    map[claude.Process]chan struct{}
}

// Daemon is the running daemon.
type Daemon struct {
	deps     Deps
	ctx      context.Context
	mu       sync.Mutex
	projects map[string]*proj
}

// startPump is set by pump.go's init func.
var startPump func(d *Daemon, ctx context.Context, project string, p claude.Process, l *eventlog.Log)

var _ control.Control = (*Daemon)(nil)

// Open builds a daemon from deps and its already-registered projects.
func Open(ctx context.Context, d Deps) (*Daemon, error) {
	daemon := &Daemon{
		deps:     d,
		ctx:      ctx,
		projects: make(map[string]*proj),
	}
	for _, rp := range d.Registry.List() {
		var st State
		err := d.Registry.LoadState(rp.Name, &st)
		if err != nil {
			if errors.Is(err, registry.ErrNoState) {
				st = State{Loop: *supervisor.New(rp.Name, queue.Queue{})}
			} else {
				return nil, err
			}
		}
		daemon.projects[rp.Name] = &proj{
			reg:   rp,
			state: st,
			pumps: make(map[claude.Process]chan struct{}),
		}
	}
	for _, name := range daemon.names() {
		p := daemon.projects[name]
		switch p.state.Loop.State {
		case "running", "starting", "paused":
			daemon.execute(ctx, name, p.state.Loop.Exited(-2, "", d.Now(), d.Config.Loop))
		}
	}
	return daemon, nil
}

func (d *Daemon) names() []string {
	out := make([]string, 0, len(d.projects))
	for n := range d.projects {
		out = append(out, n)
	}
	sort.Strings(out)
	return out
}

func (d *Daemon) runningCount() int {
	n := 0
	for _, p := range d.projects {
		switch p.state.Loop.State {
		case "running", "starting", "paused":
			n++
		}
	}
	return n
}

func queueView(q *queue.Queue) control.QueueView {
	v := control.QueueView{
		State:    q.State,
		Approved: q.Approved,
		Index:    q.Index,
		Phases:   len(q.Phases),
		Reason:   q.Reason,
	}
	if cur, ok := q.Current(); ok {
		v.Current = cur.ID
	}
	return v
}

// Projects returns a view over every registered project.
func (d *Daemon) Projects() []control.ProjectView {
	d.mu.Lock()
	defer d.mu.Unlock()
	out := []control.ProjectView{}
	for _, name := range d.names() {
		p := d.projects[name]
		out = append(out, control.ProjectView{
			Name:      p.reg.Name,
			Language:  p.reg.Language,
			RepoURL:   p.reg.RepoURL,
			State:     p.state.Loop.State,
			CreatedAt: p.reg.CreatedAt,
		})
	}
	return out
}

// CreateProject bootstraps a new project and registers it.
func (d *Daemon) CreateProject(ctx context.Context, name, language, repoURL string) (control.ProjectView, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if _, ok := d.projects[name]; ok {
		return control.ProjectView{}, control.ErrExists
	}
	spec := bootstrap.Spec{
		Name:            name,
		Language:        language,
		RepoURL:         repoURL,
		ProjectsDir:     d.deps.Config.ProjectsDir,
		CredentialsFile: d.deps.Registry.SecretPath(name, "git-credentials"),
		MorphBin:        d.deps.Config.MorphBin,
	}
	report, err := bootstrap.Create(ctx, d.deps.Runner, spec)
	if err != nil {
		if len(report.Steps) == 0 {
			return control.ProjectView{}, fmt.Errorf("%w: %v", control.ErrBadInput, err)
		}
		return control.ProjectView{}, err
	}
	rp := registry.Project{
		Name:      name,
		Language:  language,
		RepoURL:   repoURL,
		Dir:       report.Dir,
		Caps:      d.deps.Config.Defaults,
		CreatedAt: d.deps.Now(),
	}
	if err := d.deps.Registry.Add(rp); err != nil {
		if errors.Is(err, registry.ErrExists) {
			return control.ProjectView{}, control.ErrExists
		}
		if errors.Is(err, registry.ErrBadName) {
			return control.ProjectView{}, fmt.Errorf("%w: %v", control.ErrBadInput, err)
		}
		return control.ProjectView{}, err
	}
	d.projects[name] = &proj{
		reg:   rp,
		state: State{Loop: *supervisor.New(name, queue.Queue{})},
		pumps: make(map[claude.Process]chan struct{}),
	}
	return control.ProjectView{
		Name:      name,
		Language:  language,
		RepoURL:   repoURL,
		State:     "idle",
		CreatedAt: rp.CreatedAt,
	}, nil
}

// Status returns a project's status.
func (d *Daemon) Status(project string) (control.Status, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	p, ok := d.projects[project]
	if !ok {
		return control.Status{}, control.ErrUnknownProject
	}
	loop := &p.state.Loop
	st := control.Status{
		State:     loop.State,
		Phase:     loop.Phase,
		SessionID: loop.SessionID,
		Repo:      p.state.Repo,
		Stop:      loop.Stop,
		LastExit:  loop.LastExit,
	}
	if p.machine != nil {
		st.CostUSD = p.machine.CostUSD
		if p.machine.SessionID != "" {
			st.SessionID = p.machine.SessionID
		}
		if p.machine.Limits != nil {
			st.FiveHour = control.Percent(p.machine.Limits.FiveHour.Utilization)
			st.SevenDay = control.Percent(p.machine.Limits.SevenDay.Utilization)
			st.LimitsUnknown = false
		} else {
			st.LimitsUnknown = true
		}
	} else {
		st.LimitsUnknown = true
	}
	if loop.State == "running" && p.proc != nil && p.machine != nil {
		st.State = p.machine.State
	}
	if loop.State == "running" || loop.State == "paused" {
		st.Minutes = int(d.deps.Now().Sub(loop.PhaseStartedAt).Minutes())
	}
	st.Queue = queueView(&loop.Queue)
	return st, nil
}

// Events returns the current project log's window.
func (d *Daemon) Events(project string, since int64, max int) (control.Events, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	p, ok := d.projects[project]
	if !ok {
		return control.Events{}, control.ErrUnknownProject
	}
	if p.log == nil {
		return control.Events{Entries: []eventlog.Entry{}, Last: 0}, nil
	}
	return control.Events{Entries: p.log.Since(since, max), Last: p.log.Last()}, nil
}

// Order sends text to the current session.
func (d *Daemon) Order(project, text string) (session.OrderResult, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	p, ok := d.projects[project]
	if !ok {
		return session.OrderResult{}, control.ErrUnknownProject
	}
	if p.proc == nil || p.machine == nil {
		return session.OrderResult{}, control.ErrNoSession
	}
	res, acts := p.machine.Order(text, d.deps.Now())
	for _, a := range acts {
		if a.Kind == "write" {
			d.writeLine(project, a.Line)
		}
	}
	return res, nil
}

// Pending returns the machine's pending question, if any.
func (d *Daemon) Pending(project string) (*control.Question, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	p, ok := d.projects[project]
	if !ok {
		return nil, control.ErrUnknownProject
	}
	if p.machine == nil || p.machine.Pending == nil {
		return nil, nil
	}
	q := p.machine.Pending
	return &control.Question{
		RequestID: q.RequestID,
		Text:      q.Question.Text,
		Header:    q.Question.Header,
		Options:   q.Question.Options,
		AskedAt:   q.AskedAt,
	}, nil
}

// Answer answers the pending question.
func (d *Daemon) Answer(project string, option int) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	p, ok := d.projects[project]
	if !ok {
		return control.ErrUnknownProject
	}
	if p.proc == nil || p.machine == nil {
		return control.ErrNoSession
	}
	acts, err := p.machine.Answer(option, d.deps.Now())
	if err != nil {
		return err
	}
	for _, a := range acts {
		if a.Kind == "write" {
			d.writeLine(project, a.Line)
		}
	}
	return nil
}

// Interrupt asks the running session to stop its turn.
func (d *Daemon) Interrupt(project string) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	p, ok := d.projects[project]
	if !ok {
		return control.ErrUnknownProject
	}
	if p.proc == nil || p.machine == nil {
		return control.ErrNoSession
	}
	requestID := "int-" + d.deps.NewID()
	acts, err := p.machine.Interrupt(requestID)
	if err != nil {
		return err
	}
	for _, a := range acts {
		if a.Kind == "write" {
			d.writeLine(project, a.Line)
		}
	}
	return nil
}

// Usage returns the project's usage figures.
func (d *Daemon) Usage(project string) (control.Usage, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	p, ok := d.projects[project]
	if !ok {
		return control.Usage{}, control.ErrUnknownProject
	}
	loop := &p.state.Loop
	u := control.Usage{StretchCostUSD: loop.StretchUSD + loop.SessionUSD}
	if p.machine != nil {
		u.SessionCostUSD = p.machine.CostUSD
		if p.machine.Limits != nil {
			u.FiveHour = control.Percent(p.machine.Limits.FiveHour.Utilization)
			u.SevenDay = control.Percent(p.machine.Limits.SevenDay.Utilization)
			u.FiveHourResetsAt = p.machine.Limits.FiveHour.ResetsAt
			u.SevenDayResetsAt = p.machine.Limits.SevenDay.ResetsAt
			u.LimitsAt = p.machine.LimitsAt
		}
	}
	return u, nil
}

// Restart kills the running session and starts its phase again.
func (d *Daemon) Restart(project string, force bool) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	p, ok := d.projects[project]
	if !ok {
		return control.ErrUnknownProject
	}
	if p.proc != nil && p.machine != nil && p.machine.State != "ready" && !force {
		return control.ErrBusy
	}
	acts, err := p.state.Loop.Restart(d.deps.Now())
	if err != nil {
		return err
	}
	d.execute(d.ctx, project, acts)
	return nil
}

// PlanLoad replaces the project's queue with plan's.
func (d *Daemon) PlanLoad(project string, plan queue.Plan) (control.QueueView, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	p, ok := d.projects[project]
	if !ok {
		return control.QueueView{}, control.ErrUnknownProject
	}
	if p.state.Loop.State != "idle" && p.state.Loop.State != "waiting" {
		return control.QueueView{}, control.ErrBusy
	}
	q, err := queue.Load(plan, p.reg.Caps)
	if err != nil {
		return control.QueueView{}, fmt.Errorf("%w: %v", control.ErrBadInput, err)
	}
	p.state.Loop = *supervisor.New(project, q)
	p.queuedAt = d.deps.Now()
	if d.runningCount() < d.deps.Config.MaxParallel {
		acts, berr := p.state.Loop.Begin(d.deps.Now())
		if berr == nil {
			d.execute(d.ctx, project, acts)
		}
	} else {
		_ = d.deps.Registry.SaveState(project, p.state)
	}
	return queueView(&p.state.Loop.Queue), nil
}

// Continue resumes a stopped plan.
func (d *Daemon) Continue(project string) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	p, ok := d.projects[project]
	if !ok {
		return control.ErrUnknownProject
	}
	acts, err := p.state.Loop.Continue(d.deps.Now())
	if err != nil {
		if errors.Is(err, control.ErrNotWaiting) {
			return err
		}
		return fmt.Errorf("%w: %v", control.ErrNotWaiting, err)
	}
	d.execute(d.ctx, project, acts)
	return nil
}

// StopCheck returns a copy of the loop's stop.
func (d *Daemon) StopCheck(project string) (*control.Stop, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	p, ok := d.projects[project]
	if !ok {
		return nil, control.ErrUnknownProject
	}
	if p.state.Loop.Stop == nil {
		return nil, nil
	}
	cp := *p.state.Loop.Stop
	return &cp, nil
}

// PutGithubToken stores and checks a GitHub token.
func (d *Daemon) PutGithubToken(ctx context.Context, project, token string) (control.TokenResult, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	p, ok := d.projects[project]
	if !ok {
		return control.TokenResult{}, control.ErrUnknownProject
	}
	if token == "" {
		return control.TokenResult{}, control.ErrBadInput
	}
	if err := d.deps.Registry.PutSecret(project, "github-token", token); err != nil {
		return control.TokenResult{}, err
	}
	if err := d.deps.Registry.PutSecret(project, "git-credentials", github.CredentialLine(token)); err != nil {
		return control.TokenResult{}, err
	}
	acc := github.Check(ctx, d.deps.GitHubDo, token, p.reg.RepoURL, d.deps.Now())
	p.state.Repo = acc
	_ = d.deps.Registry.SaveState(project, p.state)
	res := control.TokenResult{Access: acc}
	if acc.Push {
		pushed, err := bootstrap.FirstPush(ctx, d.deps.Runner, p.reg.Dir)
		res.Pushed = pushed
		if err != nil {
			res.PushError = err.Error()
		}
	}
	return res, nil
}

// PhaseDone records a phase_done report from the project's current session.
func (d *Daemon) PhaseDone(project, sessionToken, phase, next string) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	p, ok := d.projects[project]
	if !ok {
		return control.ErrUnknownProject
	}
	stored, err := d.deps.Registry.ReadSecret(project, "session-token")
	if err != nil || stored != sessionToken {
		return control.ErrBadToken
	}
	acts, err := p.state.Loop.PhaseDone(phase, next)
	if err != nil {
		return err
	}
	d.execute(d.ctx, project, acts)
	return nil
}

// WaitOperator records a wait_operator report from the project's current session.
func (d *Daemon) WaitOperator(project, sessionToken, kind, reason, issueURL string) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	p, ok := d.projects[project]
	if !ok {
		return control.ErrUnknownProject
	}
	stored, err := d.deps.Registry.ReadSecret(project, "session-token")
	if err != nil || stored != sessionToken {
		return control.ErrBadToken
	}
	acts := p.state.Loop.WaitOperator(kind, reason, issueURL, d.deps.Now())
	d.execute(d.ctx, project, acts)
	return nil
}

// Milestone posts a milestone line for the project's current session.
func (d *Daemon) Milestone(project, sessionToken string, m control.Milestone) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	if _, ok := d.projects[project]; !ok {
		return control.ErrUnknownProject
	}
	stored, err := d.deps.Registry.ReadSecret(project, "session-token")
	if err != nil || stored != sessionToken {
		return control.ErrBadToken
	}
	if d.deps.Post != nil {
		_, _ = d.deps.Post(d.ctx, project, m.Kind, m.Headline, m.Numbers)
	}
	return nil
}

// Execute runs a batch of supervisor actions under the daemon lock.
func (d *Daemon) Execute(ctx context.Context, project string, acts []supervisor.Action) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.execute(ctx, project, acts)
}

// Tick drives the clocked checks and starts queued projects.
func (d *Daemon) Tick(ctx context.Context) {
	d.mu.Lock()
	defer d.mu.Unlock()
	names := d.names()
	for _, name := range names {
		p := d.projects[name]
		acts := p.state.Loop.Tick(d.deps.Now(), d.deps.Config.Loop)
		if len(acts) > 0 {
			d.execute(ctx, name, acts)
		}
	}
	for d.runningCount() < d.deps.Config.MaxParallel {
		best := ""
		var bestAt time.Time
		for _, name := range names {
			p := d.projects[name]
			if p.state.Loop.State == "idle" && p.state.Loop.Queue.State == "queued" {
				if best == "" || p.queuedAt.Before(bestAt) {
					best = name
					bestAt = p.queuedAt
				}
			}
		}
		if best == "" {
			break
		}
		p := d.projects[best]
		acts, err := p.state.Loop.Begin(d.deps.Now())
		if err != nil {
			break
		}
		d.execute(ctx, best, acts)
	}
}

// execute runs a batch of actions for project; d.mu must be held.
func (d *Daemon) execute(ctx context.Context, project string, acts []supervisor.Action) {
	p := d.projects[project]
	if p == nil {
		return
	}
	defer func() {
		_ = d.deps.Registry.SaveState(project, p.state)
	}()
	for _, act := range acts {
		if !d.executeAction(ctx, p, project, act) {
			return
		}
	}
}

func (d *Daemon) executeAction(ctx context.Context, p *proj, project string, act supervisor.Action) bool {
	switch act.Kind {
	case "start_check":
		start := gitrules.StartCheck(ctx, d.deps.Runner, p.reg.Dir)
		d.execute(ctx, project, p.state.Loop.StartChecked(start, d.deps.NewID, d.deps.Now()))
	case "spawn":
		if !d.spawnSession(ctx, p, project, act) {
			return false
		}
	case "first_line", "write":
		d.orderLine(project, act.Line)
	case "end_check":
		end := gitrules.EndCheck(ctx, d.deps.Runner, p.reg.Dir, p.state.Loop.Phase)
		d.execute(ctx, project, p.state.Loop.EndChecked(end, d.deps.Now()))
	case "kill":
		if p.proc != nil {
			_ = p.proc.Kill()
		}
	case "post":
		if d.deps.Post != nil {
			_, _ = d.deps.Post(ctx, project, act.Milestone.Kind, act.Milestone.Headline, act.Milestone.Numbers)
		}
	}
	return true
}

func (d *Daemon) spawnSession(ctx context.Context, p *proj, project string, act supervisor.Action) bool {
	if p.proc != nil {
		_ = p.proc.Kill()
		p.proc = nil
	}
	fail := func(err error) bool {
		d.execute(ctx, project, p.state.Loop.Exited(-3, err.Error(), d.deps.Now(), d.deps.Config.Loop))
		return false
	}
	token := d.deps.NewID()
	if err := d.deps.Registry.PutSecret(project, "session-token", token); err != nil {
		return fail(err)
	}
	base := d.deps.Registry.Base()
	dir := filepath.Join(base, project)
	mcpPath, err := claude.WriteMCPConfig(dir, "morphd", d.deps.Config.MCPBaseURL+"/mcp/"+project+"/session", token)
	if err != nil {
		return fail(err)
	}
	logPath := filepath.Join(dir, "sessions", act.SessionID+".jsonl")
	log, err := eventlog.Open(logPath)
	if err != nil {
		return fail(err)
	}
	proc, err := d.deps.Spawn(ctx, claude.Launch{
		Bin:           d.deps.Config.ClaudeBin,
		Dir:           p.reg.Dir,
		SessionID:     act.SessionID,
		Resume:        act.Resume,
		BudgetUSD:     act.BudgetUSD,
		Model:         d.deps.Config.Model,
		MCPConfigPath: mcpPath,
		Extra:         d.deps.Config.ExtraArgs,
	})
	if err != nil {
		return fail(err)
	}
	p.machine = session.New()
	p.proc = proc
	p.log = log
	if startPump != nil {
		startPump(d, ctx, project, proc, log)
	}
	return true
}

func (d *Daemon) orderLine(project, line string) {
	p := d.projects[project]
	if p == nil || p.machine == nil {
		return
	}
	_, acts := p.machine.Order(line, d.deps.Now())
	for _, a := range acts {
		if a.Kind == "write" {
			d.writeLine(project, a.Line)
		}
	}
}

// writeLine writes line to the current process and log; d.mu must be held.
func (d *Daemon) writeLine(project string, line []byte) {
	p := d.projects[project]
	if p == nil || p.proc == nil {
		return
	}
	_ = p.proc.Write(line)
	if p.log != nil {
		_, _ = p.log.Append("in", line, d.deps.Now())
	}
}
