// Package supervisor drives one project's phase loop: it feeds the queue,
// starts and restarts sessions, and runs the git checks.
package supervisor

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"morphstudio/control"
	"morphstudio/gitrules"
	"morphstudio/queue"
)

// Action is what the daemon must do after a loop input.
type Action struct {
	Kind      string
	Phase     string
	SessionID string
	Resume    bool
	BudgetUSD float64
	Line      string
	Milestone control.Milestone
}

// Loop is the phase loop of one project.
type Loop struct {
	Project        string        `json:"project"`
	Queue          queue.Queue   `json:"queue"`
	State          string        `json:"state"`
	Phase          string        `json:"phase"`
	SessionID      string        `json:"session_id"`
	PhaseStartedAt time.Time     `json:"phase_started_at"`
	Restarts       []time.Time   `json:"restarts"`
	Stop           *control.Stop `json:"stop,omitempty"`
	PausedUntil    time.Time     `json:"paused_until"`
	StretchUSD     float64       `json:"stretch_usd"`
	SessionUSD     float64       `json:"session_usd"`
	LastEventAt    time.Time     `json:"last_event_at"`
	Nudged         bool          `json:"nudged"`
	Alerted        bool          `json:"alerted"`
	LastExit       *control.Exit `json:"last_exit,omitempty"`
}

// Config bounds a loop's resilience.
type Config struct {
	MaxResumesPerHour int
	StallMinutes      int
	StretchUSD        float64
	UsageAlertPercent int
}

// ErrPhaseMismatch reports a phase_done for a phase the loop is not running.
var ErrPhaseMismatch = errors.New("phase mismatch")

// New returns an idle loop walking the given queue.
func New(project string, q queue.Queue) *Loop {
	return &Loop{Project: project, Queue: q, State: "idle"}
}

// fnum formats a float at its shortest representation.
func fnum(v float64) string {
	return strconv.FormatFloat(v, 'f', -1, 64)
}

// currentCaps returns the caps of the phase the queue points at.
func (l *Loop) currentCaps() queue.Caps {
	if l.Queue.Index >= 0 && l.Queue.Index < len(l.Queue.Phases) {
		return l.Queue.Phases[l.Queue.Index].Caps
	}
	return queue.Caps{}
}

// nextPhaseID returns the id after the queue's current phase, or "".
func (l *Loop) nextPhaseID() string {
	if l.Queue.Index+1 < len(l.Queue.Phases) {
		return l.Queue.Phases[l.Queue.Index+1].ID
	}
	return ""
}

// Begin starts the queue's current phase and asks for a start check.
func (l *Loop) Begin(now time.Time) ([]Action, error) {
	cur, err := l.Queue.Start()
	if err != nil {
		return nil, err
	}
	l.Phase = cur.ID
	l.State = "starting"
	l.Stop = nil
	l.Nudged = false
	l.Alerted = false
	l.SessionUSD = 0
	l.SessionID = ""
	l.Restarts = nil
	l.LastExit = nil
	return []Action{{Kind: "start_check", Phase: cur.ID}}, nil
}

// StartChecked turns the outcome of the start phase check into actions.
func (l *Loop) StartChecked(s gitrules.Start, newID func() string, now time.Time) []Action {
	if l.State != "starting" {
		return nil
	}
	caps := l.currentCaps()
	switch s.Mode {
	case "fresh":
		l.SessionID = newID()
		l.PhaseStartedAt = now
		l.LastEventAt = now
		l.State = "running"
		numbers := "session " + l.SessionID + " · cap $" + fnum(caps.ClaudeUSD) + " · " + fnum(caps.Hours) + " h"
		if s.Reason != "" {
			numbers += " · " + s.Reason
		}
		return []Action{
			{Kind: "spawn", Phase: l.Phase, SessionID: l.SessionID, Resume: false, BudgetUSD: caps.ClaudeUSD},
			{Kind: "first_line", Line: "/morph-orchestrator " + l.Phase},
			{
				Kind: "post",
				Milestone: control.Milestone{
					Kind:     "start",
					Headline: l.Phase + " started",
					Numbers:  numbers,
				},
			},
		}
	case "resume":
		if l.SessionID != "" {
			l.PhaseStartedAt = now
			l.LastEventAt = now
			l.State = "running"
			return []Action{
				{Kind: "spawn", Phase: l.Phase, SessionID: l.SessionID, Resume: true, BudgetUSD: caps.ClaudeUSD},
				{
					Kind: "post",
					Milestone: control.Milestone{
						Kind:     "start",
						Headline: l.Phase + " resumed",
						Numbers:  "session " + l.SessionID + " · " + s.Reason,
					},
				},
			}
		}
		l.SessionID = newID()
		l.PhaseStartedAt = now
		l.LastEventAt = now
		l.State = "running"
		return []Action{
			{Kind: "spawn", Phase: l.Phase, SessionID: l.SessionID, Resume: false, BudgetUSD: caps.ClaudeUSD},
			{Kind: "first_line", Line: "/morph-orchestrator " + l.Phase},
			{
				Kind: "post",
				Milestone: control.Milestone{
					Kind:     "start",
					Headline: l.Phase + " resumed on a fresh session",
					Numbers:  "session " + l.SessionID + " · cap $" + fnum(caps.ClaudeUSD) + " · " + fnum(caps.Hours) + " h · " + s.Reason,
				},
			},
		}
	default:
		l.State = "waiting"
		l.Stop = &control.Stop{Kind: "git", Phase: l.Phase, Reason: s.Reason, At: now}
		l.Queue.Stop(s.Reason)
		return []Action{
			{
				Kind: "post",
				Milestone: control.Milestone{
					Kind:     "stop",
					Headline: l.Phase + " not started: git",
					Numbers:  s.Reason,
				},
			},
		}
	}
}

// PhaseDone acknowledges the running phase and asks for an end check.
func (l *Loop) PhaseDone(phase, next string) ([]Action, error) {
	if (l.State == "running" || l.State == "paused") && phase == l.Phase {
		return []Action{{Kind: "end_check", Phase: l.Phase}}, nil
	}
	return nil, fmt.Errorf("%w: phase_done %q while %s %q", ErrPhaseMismatch, phase, l.State, l.Phase)
}

// EndChecked turns the outcome of the end phase check into actions.
func (l *Loop) EndChecked(e gitrules.End, now time.Time) []Action {
	if l.State != "running" && l.State != "paused" {
		return nil
	}
	finished := l.Phase
	if !e.OK {
		reason := strings.Join(e.Problems, "; ")
		l.State = "waiting"
		l.Stop = &control.Stop{Kind: "not_pushed", Phase: finished, Reason: reason, At: now}
		l.Queue.Stop(reason)
		return []Action{
			{
				Kind: "post",
				Milestone: control.Milestone{
					Kind:     "stop",
					Headline: finished + " not pushed",
					Numbers:  reason,
				},
			},
		}
	}
	l.StretchUSD += l.SessionUSD
	out, _ := l.Queue.Done(finished, l.nextPhaseID())
	actions := []Action{{Kind: "kill"}}
	switch out.Stop {
	case "smoke", "operator":
		l.State = "waiting"
		l.Stop = &control.Stop{Kind: out.Stop, Phase: finished, Reason: "stop after " + out.Stop, At: now}
		actions = append(actions, Action{
			Kind: "post",
			Milestone: control.Milestone{
				Kind:     "stop",
				Headline: finished + " done: stop after " + out.Stop,
				Numbers:  "next " + out.Next + " · waiting for continue",
			},
		})
		return actions
	case "end":
		l.State = "waiting"
		l.Stop = &control.Stop{Kind: "end", Phase: finished, Reason: "plan complete", At: now}
		actions = append(actions, Action{
			Kind: "post",
			Milestone: control.Milestone{
				Kind:     "end",
				Headline: finished + " done: plan complete",
				Numbers:  "stretch $" + strconv.FormatFloat(l.StretchUSD, 'f', 4, 64),
			},
		})
		return actions
	}
	if out.Next != "" {
		next, _ := l.Begin(now)
		actions = append(actions, next...)
	}
	return actions
}

// WaitOperator parks the loop until an operator resolves the reason.
func (l *Loop) WaitOperator(kind, reason, issueURL string, now time.Time) []Action {
	if l.State != "running" && l.State != "paused" {
		return nil
	}
	l.State = "waiting"
	l.Stop = &control.Stop{Kind: kind, Phase: l.Phase, Reason: reason, IssueURL: issueURL, At: now}
	l.Queue.Stop(kind)
	numbers := reason
	if issueURL != "" {
		numbers += " · " + issueURL
	}
	return []Action{
		{
			Kind: "post",
			Milestone: control.Milestone{
				Kind:     "stop",
				Headline: l.Phase + ": wait_operator " + kind,
				Numbers:  numbers,
			},
		},
	}
}

// Continue resumes a waiting loop.
func (l *Loop) Continue(now time.Time) ([]Action, error) {
	if l.State != "waiting" {
		return nil, control.ErrNotWaiting
	}
	if l.Stop != nil && l.Stop.Kind == "not_pushed" {
		if _, err := l.Queue.Continue(); err != nil {
			return nil, err
		}
		if _, err := l.Queue.Start(); err != nil {
			return nil, err
		}
		l.State = "running"
		l.Stop = nil
		return []Action{{Kind: "end_check", Phase: l.Phase}}, nil
	}
	if _, err := l.Queue.Continue(); err != nil {
		return nil, err
	}
	return l.Begin(now)
}

// Restart kills the running session and re-runs the start check.
func (l *Loop) Restart(now time.Time) ([]Action, error) {
	if l.State != "running" && l.State != "paused" {
		return nil, control.ErrNoSession
	}
	l.State = "starting"
	l.SessionID = ""
	l.Restarts = nil
	return []Action{
		{Kind: "kill"},
		{Kind: "start_check", Phase: l.Phase},
	}, nil
}
