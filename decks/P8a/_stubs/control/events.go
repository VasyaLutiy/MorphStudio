package control

import "morphstudio/eventlog"

type EventExit struct {
	Code   int    `json:"code"`
	Stderr string `json:"stderr"`
}

type EventView struct {
	Seq            int64      `json:"seq"`
	Dir            string     `json:"dir"`
	Type           string     `json:"type"`
	Subtype        string     `json:"subtype,omitempty"`
	Text           string     `json:"text,omitempty"`
	Tools          []string   `json:"tools,omitempty"`
	CostUSD        float64    `json:"cost_usd,omitempty"`
	TerminalReason string     `json:"terminal_reason,omitempty"`
	Exit           *EventExit `json:"exit,omitempty"`
}

type EventsView struct {
	Events []EventView `json:"events"`
	Last   int64       `json:"last"`
}

func Cut(s string, max int) string { return "stub Cut" }

func View(e eventlog.Entry, max int) EventView { return EventView{Type: "stub View"} }

func Views(e Events, max int) EventsView { return EventsView{} }
