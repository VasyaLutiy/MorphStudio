package stream

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
)

// Question is one parsed AskUserQuestion.
type Question struct {
	Text        string   `json:"text"`
	Header      string   `json:"header"`
	Options     []string `json:"options"`
	MultiSelect bool     `json:"multi_select"`
	ToolUseID   string   `json:"tool_use_id"`
}

// Result is one parsed result record.
type Result struct {
	Subtype        string   `json:"subtype"`
	IsError        bool     `json:"is_error"`
	NumTurns       int      `json:"num_turns"`
	TotalCostUSD   float64  `json:"total_cost_usd"`
	TerminalReason string   `json:"terminal_reason"`
	DurationMS     int64    `json:"duration_ms"`
	Errors         []string `json:"errors"`
}

// Window is one rate limit window.
type Window struct {
	Utilization float64 `json:"utilization"`
	ResetsAt    int64   `json:"resets_at"`
}

// Limits holds the rate limit windows.
type Limits struct {
	FiveHour Window `json:"five_hour"`
	SevenDay Window `json:"seven_day"`
}

// Event is one parsed output line of claude.
type Event struct {
	Type      string          `json:"type"`
	Subtype   string          `json:"subtype"`
	SessionID string          `json:"session_id"`
	RequestID string          `json:"request_id"`
	Tool      string          `json:"tool"`
	Text      string          `json:"text"`
	Question  *Question       `json:"question,omitempty"`
	Result    *Result         `json:"result,omitempty"`
	Limits    *Limits         `json:"limits,omitempty"`
	Input     json.RawMessage `json:"input,omitempty"`
	Raw       json.RawMessage `json:"-"`
}

// Parse decodes one line of claude stream output into an Event.
func Parse(line []byte) (Event, error) {
	trimmed := bytes.TrimSpace(line)
	if len(trimmed) == 0 {
		return Event{}, errors.New("stream: empty line")
	}

	var anyVal any
	if err := json.Unmarshal(trimmed, &anyVal); err != nil {
		return Event{}, fmt.Errorf("stream: bad json: %w", err)
	}
	if _, ok := anyVal.(map[string]any); !ok {
		return Event{}, errors.New("stream: not an object")
	}

	var m map[string]json.RawMessage
	if err := json.Unmarshal(trimmed, &m); err != nil {
		return Event{}, fmt.Errorf("stream: bad json: %w", err)
	}

	typ, ok := fieldString(m, "type")
	if !ok {
		return Event{}, errors.New("stream: no type")
	}

	ev := Event{
		Type: typ,
		Raw:  append(json.RawMessage(nil), trimmed...),
	}
	if sid, ok := fieldString(m, "session_id"); ok {
		ev.SessionID = sid
	}

	switch typ {
	case "system":
		ev.Subtype, _ = fieldString(m, "subtype")
	case "assistant":
		ev.Text = assistantText(m)
	case "result":
		ev.Subtype, _ = fieldString(m, "subtype")
		if s, ok := fieldString(m, "result"); ok {
			ev.Text = s
		}
		ev.Result = parseResult(m, ev.Subtype)
	case "rate_limit_event":
		ev.Limits = parseLimits(m)
	case "control_request":
		ev.RequestID, _ = fieldString(m, "request_id")
		parseControlRequest(m, &ev)
	case "control_response":
		var resp map[string]json.RawMessage
		if raw, ok := m["response"]; ok {
			_ = json.Unmarshal(raw, &resp)
		}
		ev.RequestID, _ = fieldString(resp, "request_id")
		ev.Subtype, _ = fieldString(resp, "subtype")
	}

	return ev, nil
}

func assistantText(m map[string]json.RawMessage) string {
	raw, ok := m["message"]
	if !ok {
		return ""
	}
	var msg map[string]json.RawMessage
	if json.Unmarshal(raw, &msg) != nil {
		return ""
	}
	craw, ok := msg["content"]
	if !ok {
		return ""
	}
	var parts []json.RawMessage
	if json.Unmarshal(craw, &parts) != nil {
		return ""
	}
	var texts []string
	for _, p := range parts {
		var pm map[string]json.RawMessage
		if json.Unmarshal(p, &pm) != nil {
			continue
		}
		pt, _ := fieldString(pm, "type")
		if pt != "text" {
			continue
		}
		if tx, ok := fieldString(pm, "text"); ok {
			texts = append(texts, tx)
		}
	}
	return strings.Join(texts, "\n")
}

func parseResult(m map[string]json.RawMessage, subtype string) *Result {
	r := &Result{Subtype: subtype}
	r.IsError, _ = fieldBool(m, "is_error")
	if v, ok := fieldInt64(m, "num_turns"); ok {
		r.NumTurns = int(v)
	}
	if v, ok := fieldFloat(m, "total_cost_usd"); ok {
		r.TotalCostUSD = v
	}
	r.TerminalReason, _ = fieldString(m, "terminal_reason")
	if v, ok := fieldInt64(m, "duration_ms"); ok {
		r.DurationMS = v
	}
	if raw, ok := m["errors"]; ok {
		var errs []string
		if json.Unmarshal(raw, &errs) == nil {
			r.Errors = errs
		}
	}
	return r
}

func parseLimits(m map[string]json.RawMessage) *Limits {
	lim := &Limits{}
	raw, ok := m["rate_limit_info"]
	if !ok {
		return lim
	}
	var rli map[string]json.RawMessage
	if json.Unmarshal(raw, &rli) != nil {
		return lim
	}
	uwRaw, ok := rawField(rli, "unifiedWindows", "unified_windows")
	if !ok {
		return lim
	}
	var uw map[string]json.RawMessage
	if json.Unmarshal(uwRaw, &uw) != nil {
		return lim
	}
	if w, ok := rawField(uw, "five_hour", "fiveHour"); ok {
		lim.FiveHour = parseWindow(w)
	}
	if w, ok := rawField(uw, "seven_day", "sevenDay"); ok {
		lim.SevenDay = parseWindow(w)
	}
	return lim
}

func parseWindow(raw json.RawMessage) Window {
	var m map[string]json.RawMessage
	if json.Unmarshal(raw, &m) != nil {
		return Window{}
	}
	var w Window
	if v, ok := fieldFloat(m, "utilization"); ok {
		w.Utilization = v
	}
	if v, ok := fieldFloat(m, "resets_at"); ok {
		w.ResetsAt = int64(v)
	} else if v, ok := fieldFloat(m, "resetsAt"); ok {
		w.ResetsAt = int64(v)
	}
	return w
}

func parseControlRequest(m map[string]json.RawMessage, ev *Event) {
	var req map[string]json.RawMessage
	if raw, ok := m["request"]; ok {
		if json.Unmarshal(raw, &req) != nil {
			return
		}
	}
	ev.Subtype, _ = fieldString(req, "subtype")
	if ev.Subtype != "can_use_tool" {
		return
	}
	ev.Tool, _ = fieldString(req, "tool_name")
	if in, ok := req["input"]; ok {
		ev.Input = append(json.RawMessage(nil), in...)
	}
	if ev.Tool != "AskUserQuestion" || len(ev.Input) == 0 {
		return
	}
	var in map[string]json.RawMessage
	if json.Unmarshal(ev.Input, &in) != nil {
		return
	}
	qraw, ok := in["questions"]
	if !ok {
		return
	}
	var questions []json.RawMessage
	if json.Unmarshal(qraw, &questions) != nil || len(questions) == 0 {
		return
	}
	var qm map[string]json.RawMessage
	if json.Unmarshal(questions[0], &qm) != nil {
		return
	}
	q := &Question{}
	q.Text, _ = fieldString(qm, "question")
	q.Header, _ = fieldString(qm, "header")
	q.MultiSelect, _ = fieldBool(qm, "multiSelect", "multi_select")
	if oraw, ok := qm["options"]; ok {
		var opts []json.RawMessage
		if json.Unmarshal(oraw, &opts) == nil {
			for _, o := range opts {
				var om map[string]json.RawMessage
				if json.Unmarshal(o, &om) != nil {
					continue
				}
				if lbl, ok := fieldString(om, "label"); ok {
					q.Options = append(q.Options, lbl)
				}
			}
		}
	}
	q.ToolUseID, _ = fieldString(req, "tool_use_id")
	ev.Question = q
}

func rawField(m map[string]json.RawMessage, keys ...string) (json.RawMessage, bool) {
	for _, k := range keys {
		if v, ok := m[k]; ok {
			return v, true
		}
	}
	return nil, false
}

func fieldString(m map[string]json.RawMessage, keys ...string) (string, bool) {
	raw, ok := rawField(m, keys...)
	if !ok {
		return "", false
	}
	var s string
	if json.Unmarshal(raw, &s) != nil {
		return "", false
	}
	return s, true
}

func fieldBool(m map[string]json.RawMessage, keys ...string) (bool, bool) {
	raw, ok := rawField(m, keys...)
	if !ok {
		return false, false
	}
	var b bool
	if json.Unmarshal(raw, &b) != nil {
		return false, false
	}
	return b, true
}

func fieldFloat(m map[string]json.RawMessage, keys ...string) (float64, bool) {
	raw, ok := rawField(m, keys...)
	if !ok {
		return 0, false
	}
	var f float64
	if json.Unmarshal(raw, &f) != nil {
		return 0, false
	}
	return f, true
}

func fieldInt64(m map[string]json.RawMessage, keys ...string) (int64, bool) {
	raw, ok := rawField(m, keys...)
	if !ok {
		return 0, false
	}
	var i int64
	if json.Unmarshal(raw, &i) != nil {
		return 0, false
	}
	return i, true
}
