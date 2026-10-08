package stream

import (
	"encoding/json"
	"errors"
)

type requestBody struct {
	Subtype string `json:"subtype"`
}

type controlRequestLine struct {
	Type      string      `json:"type"`
	RequestID string      `json:"request_id"`
	Request   requestBody `json:"request"`
}

type userMessage struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type userLine struct {
	Type            string      `json:"type"`
	Message         userMessage `json:"message"`
	ParentToolUseID any         `json:"parent_tool_use_id"`
}

type innerResponse struct {
	Behavior     string          `json:"behavior"`
	UpdatedInput json.RawMessage `json:"updatedInput"`
}

type responseBody struct {
	Subtype   string        `json:"subtype"`
	RequestID string        `json:"request_id"`
	Response  innerResponse `json:"response"`
}

type controlResponseLine struct {
	Type     string       `json:"type"`
	Response responseBody `json:"response"`
}

// Initialize returns the control_request line that opens a claude stream.
func Initialize(requestID string) []byte {
	b, _ := json.Marshal(controlRequestLine{
		Type:      "control_request",
		RequestID: requestID,
		Request:   requestBody{Subtype: "initialize"},
	})
	return append(b, '\n')
}

// Interrupt returns the control_request line that interrupts a claude run.
func Interrupt(requestID string) []byte {
	b, _ := json.Marshal(controlRequestLine{
		Type:      "control_request",
		RequestID: requestID,
		Request:   requestBody{Subtype: "interrupt"},
	})
	return append(b, '\n')
}

// User returns the user line carrying text to claude.
func User(text string) []byte {
	b, _ := json.Marshal(userLine{
		Type:            "user",
		Message:         userMessage{Role: "user", Content: text},
		ParentToolUseID: nil,
	})
	return append(b, '\n')
}

// Allow returns the control_response line granting a tool request with input.
func Allow(requestID string, input json.RawMessage) ([]byte, error) {
	if !json.Valid(input) {
		return nil, errors.New("stream: bad input")
	}
	b, err := json.Marshal(controlResponseLine{
		Type: "control_response",
		Response: responseBody{
			Subtype:   "success",
			RequestID: requestID,
			Response: innerResponse{
				Behavior:     "allow",
				UpdatedInput: append(json.RawMessage(nil), input...),
			},
		},
	})
	if err != nil {
		return nil, err
	}
	return append(b, '\n'), nil
}

// Answer returns the control_response line granting an AskUserQuestion with the
// chosen label written into the input's answers map.
func Answer(requestID string, input json.RawMessage, question, label string) ([]byte, error) {
	var m map[string]any
	if err := json.Unmarshal(input, &m); err != nil || m == nil {
		return nil, errors.New("stream: input is not an object")
	}
	m["answers"] = map[string]string{question: label}
	updated, err := json.Marshal(m)
	if err != nil {
		return nil, err
	}
	return Allow(requestID, updated)
}
