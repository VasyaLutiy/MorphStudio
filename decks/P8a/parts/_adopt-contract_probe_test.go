package control

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"morphstudio/internal/testhelp"
	"morphstudio/queue"
	"morphstudio/session"
)

// pACPlain implements only the 17 methods of Control.
type pACPlain struct{}

func (pACPlain) Projects() []ProjectView { return nil }
func (pACPlain) CreateProject(ctx context.Context, name, language, repoURL string) (ProjectView, error) {
	return ProjectView{}, nil
}
func (pACPlain) Status(project string) (Status, error)                       { return Status{}, nil }
func (pACPlain) Events(project string, since int64, max int) (Events, error) { return Events{}, nil }
func (pACPlain) Order(project, text string) (session.OrderResult, error) {
	return session.OrderResult{}, nil
}
func (pACPlain) Pending(project string) (*Question, error)                   { return nil, nil }
func (pACPlain) Answer(project string, option int) error                     { return nil }
func (pACPlain) Interrupt(project string) error                              { return nil }
func (pACPlain) Usage(project string) (Usage, error)                         { return Usage{}, nil }
func (pACPlain) Restart(project string, force bool) error                    { return nil }
func (pACPlain) PlanLoad(project string, plan queue.Plan) (QueueView, error) { return QueueView{}, nil }
func (pACPlain) Continue(project string) error                               { return nil }
func (pACPlain) StopCheck(project string) (*Stop, error)                     { return nil, nil }
func (pACPlain) PutGithubToken(ctx context.Context, project, token string) (TokenResult, error) {
	return TokenResult{}, nil
}
func (pACPlain) PhaseDone(project, sessionToken, phase, next string) error               { return nil }
func (pACPlain) WaitOperator(project, sessionToken, kind, reason, issueURL string) error { return nil }
func (pACPlain) Milestone(project, sessionToken string, m Milestone) error               { return nil }

// pACFake adds AdoptProject to the 17 methods and records its arguments.
type pACFake struct {
	pACPlain
	calls [][4]string
	view  ProjectView
	err   error
	ctxOK []bool
}

func (f *pACFake) AdoptProject(ctx context.Context, name, dir, repoURL, language string) (ProjectView, error) {
	f.calls = append(f.calls, [4]string{name, dir, repoURL, language})
	f.ctxOK = append(f.ctxOK, ctx != nil && ctx.Value(pACKey{}) == "k")
	return f.view, f.err
}

type pACKey struct{}

var _ Control = pACPlain{}
var _ Control = (*pACFake)(nil)

var pACStudio = ProjectView{Name: "studio", Language: "go", RepoURL: "https://github.com/VasyaLutiy/MorphStudio", State: "idle", CreatedAt: time.Date(2026, 10, 10, 18, 0, 0, 0, time.UTC)}

func pACCtx() context.Context { return context.WithValue(context.Background(), pACKey{}, "k") }

func TestProbeAdoptContractExample1(t *testing.T) {
	c := &pACFake{view: pACStudio}
	var _ Adopter = c
	v, err := Adopt(pACCtx(), c, "studio", "/home/morph/MorphStudio", "https://github.com/VasyaLutiy/MorphStudio", "go")
	testhelp.Equal(t, "example 1 view", v, pACStudio)
	testhelp.Equal(t, "example 1 error", err, nil)
	testhelp.Equal(t, "example 1 calls", c.calls, [][4]string{{"studio", "/home/morph/MorphStudio", "https://github.com/VasyaLutiy/MorphStudio", "go"}})
	// variant: the context reaches AdoptProject as given
	testhelp.Equal(t, "example 1 variant ctx passed through", c.ctxOK, []bool{true})
}

func TestProbeAdoptContractExample2(t *testing.T) {
	v, err := Adopt(context.Background(), pACPlain{}, "studio", "/x", "https://github.com/o/r", "go")
	testhelp.Equal(t, "example 2 view", v, ProjectView{})
	testhelp.Equal(t, "example 2 errors.Is ErrUnsupported", errors.Is(err, ErrUnsupported), true)
	if err != nil {
		testhelp.Equal(t, "example 2 text", err.Error(), "adopt not supported")
	} else {
		t.Errorf("example 2: error nil, want %q", "adopt not supported")
	}
	st, code := Code(ErrUnsupported)
	testhelp.Equal(t, "example 2 Code(ErrUnsupported)", fmt.Sprint(st, " ", code), "500 internal")
	// variant: a pointer to a plain Control is not an Adopter either
	_, err = Adopt(context.Background(), &pACPlain{}, "studio", "/x", "https://github.com/o/r", "go")
	testhelp.Equal(t, "example 2 variant *pACPlain", errors.Is(err, ErrUnsupported), true)
}

func TestProbeAdoptContractExample3(t *testing.T) {
	c := &pACFake{err: fmt.Errorf("%w: bootstrap: refused: not a git repository: /home/morph/nowhere", ErrBadInput)}
	v, err := Adopt(context.Background(), c, "gamma", "/home/morph/nowhere", "https://github.com/o/gamma", "go")
	testhelp.Equal(t, "example 3 view", v, ProjectView{})
	testhelp.Equal(t, "example 3 errors.Is ErrBadInput", errors.Is(err, ErrBadInput), true)
	if err != nil {
		testhelp.Equal(t, "example 3 text", err.Error(), "bad input: bootstrap: refused: not a git repository: /home/morph/nowhere")
	}
	st, code := Code(err)
	testhelp.Equal(t, "example 3 Code bad input", fmt.Sprint(st, " ", code), "400 bad_input")
	c.err = ErrExists
	_, err = Adopt(context.Background(), c, "demo", "/home/morph/demo", "https://github.com/o/demo", "python")
	testhelp.Equal(t, "example 3 errors.Is ErrExists", errors.Is(err, ErrExists), true)
	st, code = Code(err)
	testhelp.Equal(t, "example 3 Code exists", fmt.Sprint(st, " ", code), "409 exists")
	testhelp.Equal(t, "example 3 calls", c.calls, [][4]string{{"gamma", "/home/morph/nowhere", "https://github.com/o/gamma", "go"}, {"demo", "/home/morph/demo", "https://github.com/o/demo", "python"}})
	// variant: values and error are returned as is together
	c.view, c.err = pACStudio, ErrExists
	v, _ = Adopt(context.Background(), c, "studio", "/s", "https://github.com/o/s", "go")
	testhelp.Equal(t, "example 3 variant view returned with the error", v, pACStudio)
}
