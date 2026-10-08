package session

import (
	"fmt"
	"strings"
	"time"

	"morphstudio/stream"
)

// OrderResult reports what Order did with a piece of text: Sent is true when
// the text went straight to a ready machine, Queued is the queue length after
// a text was appended to a busy or questioning machine.
type OrderResult struct {
	Sent   bool `json:"sent,omitempty"`
	Queued int  `json:"queued,omitempty"`
}

// Order routes text to a ready machine or queues it behind the turn in flight.
// State "ready" writes the user line and becomes busy; any other state appends
// the trimmed text to the queue. Empty text is a no-op.
func (m *Machine) Order(text string, now time.Time) (OrderResult, []Action) {
	text = strings.TrimSpace(text)
	if text == "" {
		return OrderResult{}, nil
	}
	if m.State == "ready" {
		m.State = "busy"
		m.TurnStartedAt = now
		return OrderResult{Sent: true}, []Action{{Kind: "write", Line: stream.User(text)}}
	}
	m.Queue = append(m.Queue, text)
	return OrderResult{Queued: len(m.Queue)}, nil
}

// Answer grants the pending AskUserQuestion with the option number chosen
// (1-based). It clears the pending question and marks the machine busy.
func (m *Machine) Answer(option int, now time.Time) ([]Action, error) {
	if m.Pending == nil {
		return nil, ErrNoQuestion
	}
	opts := m.Pending.Question.Options
	if option < 1 || option > len(opts) {
		return nil, fmt.Errorf("%w: %d of 1..%d", ErrBadOption, option, len(opts))
	}
	line, err := stream.Answer(m.Pending.RequestID, m.Pending.Input, m.Pending.Question.Text, opts[option-1])
	if err != nil {
		return nil, err
	}
	m.Pending = nil
	m.State = "busy"
	return []Action{{Kind: "write", Line: line}}, nil
}

// Interrupt asks claude to stop the turn in flight. A ready machine has
// nothing to interrupt; busy and question states write the control line and
// leave the state for the following result to settle.
func (m *Machine) Interrupt(requestID string) ([]Action, error) {
	if m.State == "ready" {
		return nil, ErrNotBusy
	}
	return []Action{{Kind: "write", Line: stream.Interrupt(requestID)}}, nil
}
