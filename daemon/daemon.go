// Package daemon is the control surface over the project registry: it creates
// and adopts projects, drives each project's phase loop and executes the loop's
// actions over claude, git and the network.
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

// Deps is everything the daemon needs from the outside world.
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

// Config bounds the daemon and its loops. DoneWait is how long the daemon
// waits, after a phase_done, for the result of the turn that called it before
// it ends the session anyway; 0 ends the session at once.
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

// State is a project's persisted daemon state.
type State struct {
	Loop supervisor.Loop `json:"loop"`
	Repo github.Access   `json:"repo"`
}

// pendingDone is the end batch a phase_done left for the turn's result.
type pendingDone struct {
	proc     claude.Process
	acts     []supervisor.Action
	deadline time.Time
}

// proj is the daemon's per-project state.
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

// Daemon is the control surface over the project registry.
type Daemon struct {
	deps     Deps
	ctx      context.Context
	mu       sync.Mutex
	projects map[string]*proj
}

// startPump is set by pump.go's init; it pumps a process's stdout lines into
// the project's machine. It is nil until pump.go links itself in.
var startPump func(d *Daemon, ctx context.Context, project string, p claude.Process, l *eventlog.Log)

var _ control.Control = (*Daemon)(nil)
var _ control.Adopter = (*Daemon)(nil)

// Open loads every registered project into a new daemon. A loop that was live
// when the daemon went down counts as an exit and is restarted at once.
func Open(ctx context.Context, deps Deps) (*Daemon, error) {
	d := &Daemon{deps: deps, ctx: ctx, projects: map[string]*proj{}}
	for _, rp := range deps.Registry.List() {
		var st State
		if err := deps.Registry.LoadState(rp.Name, &st); err != nil {
			if errors.Is(err, registry.ErrNoState) {
				st = State{Loop: *supervisor.New(rp.Name, queue.Queue{})}
			} else {
				return nil, err
			}
		}
		d.projects[rp.Name] = &proj{
			reg:   rp,
			state: st,
			pumps: map[claude.Process]chan struct{}{},
		}
	}
	for _, name := range d.names() {
		pr := d.projects[name]
		switch pr.state.Loop.State {
		case "running", "starting", "paused":
			d.execute(ctx, name, pr.state.Loop.Exited(-2, "", deps.Now(), deps.Config.Loop))
		}
	}
	return d, nil
}

// names returns the project names in order. d.mu must be held.
func (d *Daemon) names() []string {
	out := make([]string, 0, len(d.projects))
	for name := range d.projects {
		out = append(out, name)
	}
	sort.Strings(out)
	return out
}

// runningCount counts projects whose loop is live. d.mu must be held.
func (d *Daemon) runningCount() int {
	n := 0
	for _, pr := range d.projects {
		switch pr.state.Loop.State {
		case "running", "starting", "paused":
			n++
		}
	}
	return n
}

// oldestQueued returns the idle project with a queued plan whose queuedAt is
// the oldest. d.mu must be held.
func (d *Daemon) oldestQueued() (string, bool) {
	var best string
	var bestAt time.Time
	found := false
	for _, name := range d.names() {
		pr := d.projects[name]
		if pr.state.Loop.State != "idle" {
			continue
		}
		if pr.state.Loop.Queue.State != "queued" {
			continue
		}
		if !found || pr.queuedAt.Before(bestAt) {
			best = name
			bestAt = pr.queuedAt
			found = true
		}
	}
	return best, found
}

// Projects returns one view per registered project in name order.
func (d *Daemon) Projects() []control.ProjectView {
	d.mu.Lock()
	defer d.mu.Unlock()
	out := make([]control.ProjectView, 0, len(d.projects))
	for _, name := range d.names() {
		pr := d.projects[name]
		out = append(out, control.ProjectView{
			Name:      pr.reg.Name,
			Language:  pr.reg.Language,
			RepoURL:   pr.reg.RepoURL,
			State:     pr.state.Loop.State,
			CreatedAt: pr.reg.CreatedAt,
		})
	}
	return out
}

// CreateProject bootstraps a new project and registers it.
func (d *Daemon) CreateProject(ctx context.Context, name, language, repoURL string) (control.ProjectView, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if _, exists := d.projects[name]; exists {
		return control.ProjectView{}, control.ErrExists
	}
	rep, err := bootstrap.Create(ctx, d.deps.Runner, bootstrap.Spec{
		Name:            name,
		Language:        language,
		RepoURL:         repoURL,
		ProjectsDir:     d.deps.Config.ProjectsDir,
		CredentialsFile: d.deps.Registry.SecretPath(name, "git-credentials"),
		MorphBin:        d.deps.Config.MorphBin,
	})
	if err != nil {
		if len(rep.Steps) == 0 {
			return control.ProjectView{}, fmt.Errorf("%w: %v", control.ErrBadInput, err)
		}
		return control.ProjectView{}, err
	}
	created := d.deps.Now()
	rp := registry.Project{
		Name:      name,
		Language:  language,
		RepoURL:   repoURL,
		Dir:       rep.Dir,
		Caps:      d.deps.Config.Defaults,
		CreatedAt: created,
	}
	if err := d.deps.Registry.Add(rp); err != nil {
		switch {
		case errors.Is(err, registry.ErrExists):
			return control.ProjectView{}, control.ErrExists
		case errors.Is(err, registry.ErrBadName):
			return control.ProjectView{}, fmt.Errorf("%w: %v", control.ErrBadInput, err)
		default:
			return control.ProjectView{}, err
		}
	}
	d.projects[name] = &proj{
		reg:   rp,
		state: State{Loop: *supervisor.New(name, queue.Queue{})},
		pumps: map[claude.Process]chan struct{}{},
	}
	return control.ProjectView{
		Name:      name,
		Language:  language,
		RepoURL:   repoURL,
		State:     "idle",
		CreatedAt: created,
	}, nil
}

// AdoptProject registers a repository that already exists at dir.
func (d *Daemon) AdoptProject(ctx context.Context, name, dir, repoURL, language string) (control.ProjectView, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if _, exists := d.projects[name]; exists {
		return control.ProjectView{}, control.ErrExists
	}
	_, err := bootstrap.Adopt(ctx, d.deps.Runner, bootstrap.AdoptSpec{
		Name:            name,
		Language:        language,
		RepoURL:         repoURL,
		Dir:             dir,
		CredentialsFile: d.deps.Registry.SecretPath(name, "git-credentials"),
	})
	if err != nil {
		if errors.Is(err, bootstrap.ErrRefused) {
			return control.ProjectView{}, fmt.Errorf("%w: %v", control.ErrBadInput, err)
		}
		return control.ProjectView{}, err
	}
	created := d.deps.Now()
	rp := registry.Project{
		Name:      name,
		Language:  language,
		RepoURL:   repoURL,
		Dir:       dir,
		Caps:      d.deps.Config.Defaults,
		CreatedAt: created,
	}
	if err := d.deps.Registry.Add(rp); err != nil {
		switch {
		case errors.Is(err, registry.ErrExists):
			return control.ProjectView{}, control.ErrExists
		case errors.Is(err, registry.ErrBadName):
			return control.ProjectView{}, fmt.Errorf("%w: %v", control.ErrBadInput, err)
		default:
			return control.ProjectView{}, err
		}
	}
	d.projects[name] = &proj{
		reg:   rp,
		state: State{Loop: *supervisor.New(name, queue.Queue{})},
		pumps: map[claude.Process]chan struct{}{},
	}
	return control.ProjectView{
		Name:      name,
		Language:  language,
		RepoURL:   repoURL,
		State:     "idle",
		CreatedAt: created,
	}, nil
}

// Status reports a project's state.
func (d *Daemon) Status(project string) (control.Status, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	pr, ok := d.projects[project]
	if !ok {
		return control.Status{}, control.ErrUnknownProject
	}
	return d.status(pr), nil
}

// status folds a project's loop and machine into the outward Status.
func (d *Daemon) status(pr *proj) control.Status {
	loop := &pr.state.Loop
	st := control.Status{
		State:         loop.State,
		Phase:         loop.Phase,
		SessionID:     loop.SessionID,
		Repo:          pr.state.Repo,
		Stop:          loop.Stop,
		LastExit:      loop.LastExit,
		LimitsUnknown: true,
	}
	if loop.State == "running" && pr.proc != nil && pr.machine != nil {
		st.State = pr.machine.State
	}
	if loop.State == "running" || loop.State == "paused" {
		st.Minutes = int(d.deps.Now().Sub(loop.PhaseStartedAt).Minutes())
	}
	if pr.machine != nil {
		st.CostUSD = pr.machine.CostUSD
		if pr.machine.SessionID != "" {
			st.SessionID = pr.machine.SessionID
		}
		if pr.machine.Limits != nil {
			st.LimitsUnknown = false
			st.FiveHour = control.Percent(pr.machine.Limits.FiveHour.Utilization)
			st.SevenDay = control.Percent(pr.machine.Limits.SevenDay.Utilization)
		}
	}
	q := loop.Queue
	qv := control.QueueView{
		State:    q.State,
		Approved: q.Approved,
		Index:    q.Index,
		Phases:   len(q.Phases),
		Reason:   q.Reason,
	}
	if cur, ok := q.Current(); ok {
		qv.Current = cur.ID
	}
	st.Queue = qv
	return st
}

// Events returns a window over a project's event log.
func (d *Daemon) Events(project string, since int64, max int) (control.Events, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	pr, ok := d.projects[project]
	if !ok {
		return control.Events{}, control.ErrUnknownProject
	}
	if pr.log == nil {
		return control.Events{Entries: []eventlog.Entry{}, Last: 0}, nil
	}
	return control.Events{Entries: pr.log.Since(since, max), Last: pr.log.Last()}, nil
}

// Order routes a piece of text to the project's session.
func (d *Daemon) Order(project, text string) (session.OrderResult, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	pr, ok := d.projects[project]
	if !ok {
		return session.OrderResult{}, control.ErrUnknownProject
	}
	if pr.proc == nil || pr.machine == nil {
		return session.OrderResult{}, control.ErrNoSession
	}
	res, acts := pr.machine.Order(text, d.deps.Now())
	for _, a := range acts {
		if a.Kind == "write" {
			d.writeLine(project, a.Line)
		}
	}
	return res, nil
}

// Pending returns the project's pending question, if any.
func (d *Daemon) Pending(project string) (*control.Question, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	pr, ok := d.projects[project]
	if !ok {
		return nil, control.ErrUnknownProject
	}
	if pr.machine == nil || pr.machine.Pending == nil {
		return nil, nil
	}
	p := pr.machine.Pending
	return &control.Question{
		RequestID: p.RequestID,
		Text:      p.Question.Text,
		Header:    p.Question.Header,
		Options:   append([]string(nil), p.Question.Options...),
		AskedAt:   p.AskedAt,
	}, nil
}

// Answer answers the project's pending question.
func (d *Daemon) Answer(project string, option int) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	pr, ok := d.projects[project]
	if !ok {
		return control.ErrUnknownProject
	}
	if pr.proc == nil || pr.machine == nil {
		return control.ErrNoSession
	}
	acts, err := pr.machine.Answer(option, d.deps.Now())
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

// Interrupt asks the project's session to stop the turn in flight.
func (d *Daemon) Interrupt(project string) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	pr, ok := d.projects[project]
	if !ok {
		return control.ErrUnknownProject
	}
	if pr.proc == nil || pr.machine == nil {
		return control.ErrNoSession
	}
	requestID := "int-" + d.deps.NewID()
	acts, err := pr.machine.Interrupt(requestID)
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

// Usage reports rate-limit utilization and session spend.
func (d *Daemon) Usage(project string) (control.Usage, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	pr, ok := d.projects[project]
	if !ok {
		return control.Usage{}, control.ErrUnknownProject
	}
	loop := &pr.state.Loop
	u := control.Usage{
		StretchCostUSD: loop.StretchUSD + loop.SessionUSD,
	}
	if pr.machine != nil {
		u.SessionCostUSD = pr.machine.CostUSD
		u.LimitsAt = pr.machine.LimitsAt
		if pr.machine.Limits != nil {
			u.FiveHour = control.Percent(pr.machine.Limits.FiveHour.Utilization)
			u.SevenDay = control.Percent(pr.machine.Limits.SevenDay.Utilization)
			u.FiveHourResetsAt = pr.machine.Limits.FiveHour.ResetsAt
			u.SevenDayResetsAt = pr.machine.Limits.SevenDay.ResetsAt
		}
	}
	return u, nil
}

// Restart kills the project's running session and re-runs the start check.
func (d *Daemon) Restart(project string, force bool) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	pr, ok := d.projects[project]
	if !ok {
		return control.ErrUnknownProject
	}
	if pr.proc != nil && pr.machine != nil && pr.machine.State != "ready" && !force {
		return control.ErrBusy
	}
	acts, err := pr.state.Loop.Restart(d.deps.Now())
	if err != nil {
		return err
	}
	d.execute(d.ctx, project, acts)
	return nil
}

// PlanLoad installs an approved plan and begins its first phase when the
// daemon has room for another running project.
func (d *Daemon) PlanLoad(project string, plan queue.Plan) (control.QueueView, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	pr, ok := d.projects[project]
	if !ok {
		return control.QueueView{}, control.ErrUnknownProject
	}
	if pr.state.Loop.State != "idle" && pr.state.Loop.State != "waiting" {
		return control.QueueView{}, control.ErrBusy
	}
	q, err := queue.Load(plan, pr.reg.Caps)
	if err != nil {
		return control.QueueView{}, fmt.Errorf("%w: %v", control.ErrBadInput, err)
	}
	pr.state.Loop = *supervisor.New(project, q)
	pr.queuedAt = d.deps.Now()
	if d.runningCount() < d.deps.Config.MaxParallel {
		acts, err := pr.state.Loop.Begin(d.deps.Now())
		if err == nil {
			d.execute(d.ctx, project, acts)
		}
	} else {
		_ = d.deps.Registry.SaveState(project, pr.state)
	}
	return d.queueView(pr), nil
}

// queueView builds the outward view of a project's queue.
func (d *Daemon) queueView(pr *proj) control.QueueView {
	q := pr.state.Loop.Queue
	qv := control.QueueView{
		State:    q.State,
		Approved: q.Approved,
		Index:    q.Index,
		Phases:   len(q.Phases),
		Reason:   q.Reason,
	}
	if cur, ok := q.Current(); ok {
		qv.Current = cur.ID
	}
	return qv
}

// Continue resumes a waiting project.
func (d *Daemon) Continue(project string) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	pr, ok := d.projects[project]
	if !ok {
		return control.ErrUnknownProject
	}
	acts, err := pr.state.Loop.Continue(d.deps.Now())
	if err != nil {
		if errors.Is(err, control.ErrNotWaiting) {
			return err
		}
		return fmt.Errorf("%w: %v", control.ErrNotWaiting, err)
	}
	d.execute(d.ctx, project, acts)
	return nil
}

// StopCheck returns the project's stop, if any.
func (d *Daemon) StopCheck(project string) (*control.Stop, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	pr, ok := d.projects[project]
	if !ok {
		return nil, control.ErrUnknownProject
	}
	if pr.state.Loop.Stop == nil {
		return nil, nil
	}
	cp := *pr.state.Loop.Stop
	return &cp, nil
}

// PutGithubToken stores a project's GitHub token and checks its access.
func (d *Daemon) PutGithubToken(ctx context.Context, project, token string) (control.TokenResult, error) {
	d.mu.Lock()
	defer d.mu.Unlock()
	if token == "" {
		return control.TokenResult{}, control.ErrBadInput
	}
	pr, ok := d.projects[project]
	if !ok {
		return control.TokenResult{}, control.ErrUnknownProject
	}
	if err := d.deps.Registry.PutSecret(project, "github-token", token); err != nil {
		return control.TokenResult{}, err
	}
	if err := d.deps.Registry.PutSecret(project, "git-credentials", github.CredentialLine(token)); err != nil {
		return control.TokenResult{}, err
	}
	access := github.Check(ctx, d.deps.GitHubDo, token, pr.reg.RepoURL, d.deps.Now())
	pr.state.Repo = access
	_ = d.deps.Registry.SaveState(project, pr.state)
	res := control.TokenResult{Access: access}
	if access.Push {
		pushed, err := bootstrap.FirstPush(ctx, d.deps.Runner, pr.reg.Dir)
		res.Pushed = pushed
		if err != nil {
			res.PushError = err.Error()
		}
	}
	return res, nil
}

// PhaseDone records that the running session finished its phase.
func (d *Daemon) PhaseDone(project, sessionToken, phase, next string) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	pr, ok := d.projects[project]
	if !ok {
		return control.ErrUnknownProject
	}
	if tok, err := d.deps.Registry.ReadSecret(project, "session-token"); err != nil || tok != sessionToken {
		return control.ErrBadToken
	}
	acts, err := pr.state.Loop.PhaseDone(phase, next)
	if err != nil {
		return err
	}
	if d.deps.Config.DoneWait > 0 && pr.proc != nil && pr.machine != nil && pr.machine.State != "ready" {
		pr.done = &pendingDone{proc: pr.proc, acts: acts, deadline: d.deps.Now().Add(d.deps.Config.DoneWait)}
		return nil
	}
	d.execute(d.ctx, project, acts)
	return nil
}

// WaitOperator parks the project's loop for an operator.
func (d *Daemon) WaitOperator(project, sessionToken, kind, reason, issueURL string) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	pr, ok := d.projects[project]
	if !ok {
		return control.ErrUnknownProject
	}
	if tok, err := d.deps.Registry.ReadSecret(project, "session-token"); err != nil || tok != sessionToken {
		return control.ErrBadToken
	}
	d.execute(d.ctx, project, pr.state.Loop.WaitOperator(kind, reason, issueURL, d.deps.Now()))
	return nil
}

// Milestone posts a progress report from the project's session.
func (d *Daemon) Milestone(project, sessionToken string, m control.Milestone) error {
	d.mu.Lock()
	defer d.mu.Unlock()
	if _, ok := d.projects[project]; !ok {
		return control.ErrUnknownProject
	}
	if tok, err := d.deps.Registry.ReadSecret(project, "session-token"); err != nil || tok != sessionToken {
		return control.ErrBadToken
	}
	_, _ = d.deps.Post(d.ctx, project, m.Kind, m.Headline, m.Numbers)
	return nil
}

// Execute runs a batch of loop actions over d.mu, then saves the state.
func (d *Daemon) Execute(ctx context.Context, project string, acts []supervisor.Action) {
	d.mu.Lock()
	defer d.mu.Unlock()
	d.execute(ctx, project, acts)
}

// Tick drives the clocked work of every project: pending ends, the loop's own
// ticks, and starting queued projects while the parallel budget allows.
func (d *Daemon) Tick(ctx context.Context) {
	d.mu.Lock()
	defer d.mu.Unlock()
	for _, name := range d.names() {
		pr := d.projects[name]
		if pr.done != nil {
			if pr.done.proc != pr.proc {
				pr.done = nil
			} else if !d.deps.Now().Before(pr.done.deadline) {
				d.finishDone(ctx, name, pr.done.proc)
			}
		}
		acts := pr.state.Loop.Tick(d.deps.Now(), d.deps.Config.Loop)
		if len(acts) > 0 {
			d.execute(ctx, name, acts)
		}
	}
	for d.runningCount() < d.deps.Config.MaxParallel {
		name, ok := d.oldestQueued()
		if !ok {
			return
		}
		pr := d.projects[name]
		acts, err := pr.state.Loop.Begin(d.deps.Now())
		if err != nil {
			return
		}
		d.execute(ctx, name, acts)
	}
}

// finishDone runs the pending end batch when the process it belongs to has
// reached it. d.mu must be held; it does not lock.
func (d *Daemon) finishDone(ctx context.Context, project string, p claude.Process) {
	pr := d.projects[project]
	if pr == nil || pr.done == nil || pr.done.proc != p {
		return
	}
	acts := pr.done.acts
	pr.done = nil
	d.execute(ctx, project, acts)
}

// execute runs a batch of actions in order. d.mu must be held; it does not
// lock. The state is saved after the whole batch.
func (d *Daemon) execute(ctx context.Context, project string, acts []supervisor.Action) {
	pr := d.projects[project]
	if pr == nil {
		return
	}
	for _, a := range acts {
		if !d.step(ctx, project, pr, a) {
			break
		}
	}
	_ = d.deps.Registry.SaveState(project, pr.state)
}

// step runs one action. It reports false when a failed spawn must skip the
// rest of the batch. d.mu must be held.
func (d *Daemon) step(ctx context.Context, project string, pr *proj, a supervisor.Action) bool {
	switch a.Kind {
	case "start_check":
		start := gitrules.StartCheck(ctx, d.deps.Runner, pr.reg.Dir)
		d.execute(ctx, project, pr.state.Loop.StartChecked(start, d.deps.NewID, d.deps.Now()))
	case "spawn":
		if err := d.spawn(ctx, project, pr, a); err != nil {
			d.execute(ctx, project, pr.state.Loop.Exited(-3, err.Error(), d.deps.Now(), d.deps.Config.Loop))
			return false
		}
	case "first_line", "write":
		if pr.machine != nil {
			_, acts := pr.machine.Order(a.Line, d.deps.Now())
			for _, w := range acts {
				if w.Kind == "write" {
					d.writeLine(project, w.Line)
				}
			}
		}
	case "end_check":
		end := gitrules.EndCheck(ctx, d.deps.Runner, pr.reg.Dir, pr.state.Loop.Phase)
		d.execute(ctx, project, pr.state.Loop.EndChecked(end, d.deps.Now()))
	case "kill":
		if pr.proc != nil {
			_ = pr.proc.Kill()
		}
	case "post":
		_, _ = d.deps.Post(ctx, project, a.Milestone.Kind, a.Milestone.Headline, a.Milestone.Numbers)
	}
	return true
}

// spawn launches a claude session for a spawn action, killing and clearing
// whatever process the project still holds. d.mu must be held.
func (d *Daemon) spawn(ctx context.Context, project string, pr *proj, a supervisor.Action) error {
	if pr.proc != nil {
		_ = pr.proc.Kill()
		pr.proc = nil
	}
	token := d.deps.NewID()
	if err := d.deps.Registry.PutSecret(project, "session-token", token); err != nil {
		return err
	}
	path, err := claude.WriteMCPConfig(
		filepath.Join(d.deps.Registry.Base(), project),
		"morphd",
		d.deps.Config.MCPBaseURL+"/mcp/"+project+"/session",
		token,
	)
	if err != nil {
		return err
	}
	lg, err := eventlog.Open(filepath.Join(d.deps.Registry.Base(), project, "sessions", a.SessionID+".jsonl"))
	if err != nil {
		return err
	}
	p, err := d.deps.Spawn(ctx, claude.Launch{
		Bin:           d.deps.Config.ClaudeBin,
		Dir:           pr.reg.Dir,
		SessionID:     a.SessionID,
		Resume:        a.Resume,
		BudgetUSD:     a.BudgetUSD,
		Model:         d.deps.Config.Model,
		MCPConfigPath: path,
		Extra:         d.deps.Config.ExtraArgs,
	})
	if err != nil {
		return err
	}
	pr.machine = session.New()
	pr.proc = p
	pr.log = lg
	if startPump != nil {
		startPump(d, ctx, project, p, lg)
	}
	return nil
}

// writeLine writes one line to the project's current process and event log.
// d.mu must be held.
func (d *Daemon) writeLine(project string, line []byte) {
	pr := d.projects[project]
	if pr == nil || pr.proc == nil {
		return
	}
	_ = pr.proc.Write(line)
	if pr.log != nil {
		_, _ = pr.log.Append("in", line, d.deps.Now())
	}
}
