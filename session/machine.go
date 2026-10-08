package session

import (
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"morphstudio/stream"
)

// ErrNoQuestion is returned when there is no pending question to answer.
var ErrNoQuestion = errors.New("no pending question")

// ErrBadOption is returned when an answer names an option out of range.
var ErrBadOption = errors.New("option out of range")

// ErrNotBusy is returned when an action requires a turn in flight.
var ErrNotBusy = errors.New("session is not busy")

// Pending is an AskUserQuestion request awaiting an answer.
type Pending struct {
	RequestID string          `json:"request_id"`
	Question  stream.Question `json:"question"`
	Input     json.RawMessage `json:"input"`
	AskedAt   time.Time       `json:"asked_at"`
}

// Action is what the daemon must do after an event or a command.
type Action struct {
	Kind     string
	Line     []byte
	Headline string
	Numbers  string
}

// Machine is the session state and its transition on every parsed event.
type Machine struct {
	State         string         `json:"state"`
	SessionID     string         `json:"session_id"`
	Turns         int            `json:"turns"`
	CostUSD       float64        `json:"cost_usd"`
	Limits        *stream.Limits `json:"limits,omitempty"`
	LimitsAt      time.Time      `json:"limits_at"`
	Queue         []string       `json:"queue"`
	Pending       *Pending       `json:"pending,omitempty"`
	LastResult    *stream.Result `json:"last_result,omitempty"`
	TurnStartedAt time.Time      `json:"turn_started_at"`
	LastEventAt   time.Time      `json:"last_event_at"`
	Allowed       int            `json:"allowed"`
}

// New returns a Machine in the "ready" state with an empty, non-nil queue.
func New() *Machine {
	return &Machine{State: "ready", Queue: []string{}}
}

// Apply folds one parsed claude event into the machine and returns the actions
// the daemon must take as a result. It never returns an empty non-nil slice.
func (m *Machine) Apply(ev stream.Event, now time.Time) []Action {
	m.LastEventAt = now

	switch ev.Type {
	case "system":
		if ev.Subtype == "init" {
			m.SessionID = ev.SessionID
		}
	case "rate_limit_event":
		m.Limits = ev.Limits
		m.LimitsAt = now
		return []Action{{Kind: "limit"}}
	case "control_request":
		if ev.Subtype != "can_use_tool" {
			break
		}
		if ev.Tool == "AskUserQuestion" && ev.Question != nil {
			m.Pending = &Pending{
				RequestID: ev.RequestID,
				Question:  *ev.Question,
				Input:     append(json.RawMessage(nil), ev.Input...),
				AskedAt:   now,
			}
			m.State = "question"
			return []Action{{
				Kind:     "ask",
				Headline: ev.Question.Text,
				Numbers:  numberedOptions(ev.Question.Options),
			}}
		}
		in := ev.Input
		if len(in) == 0 || !json.Valid(in) {
			in = json.RawMessage("{}")
		}
		line, _ := stream.Allow(ev.RequestID, in)
		m.Allowed++
		return []Action{{Kind: "write", Line: line}}
	case "result":
		m.Turns++
		m.CostUSD = ev.Result.TotalCostUSD
		m.LastResult = ev.Result
		m.Pending = nil
		m.State = "ready"
		actions := []Action{{
			Kind:     "turn",
			Headline: ev.Text,
			Numbers:  fmt.Sprintf("%s · turns %d · $%.4f", ev.Result.Subtype, ev.Result.NumTurns, ev.Result.TotalCostUSD),
		}}
		if len(m.Queue) > 0 {
			text := m.Queue[0]
			m.Queue = m.Queue[1:]
			m.State = "busy"
			m.TurnStartedAt = now
			actions = append(actions, Action{Kind: "write", Line: stream.User(text)})
		}
		return actions
	}

	return nil
}

func numberedOptions(opts []string) string {
	if len(opts) == 0 {
		return ""
	}
	parts := make([]string, len(opts))
	for i, opt := range opts {
		parts[i] = fmt.Sprintf("%d %s", i+1, opt)
	}
	return strings.Join(parts, " · ")
}
