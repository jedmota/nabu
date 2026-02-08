package model

import (
	"net/http"
	"time"
)

// FlowID is a unique identifier for a flow
type FlowID uint64

// Flow represents a complete HTTP request/response cycle
type Flow struct {
	ID        FlowID    `json:"id"`
	Request   *Request  `json:"request,omitempty"`
	Response  *Response `json:"response,omitempty"`
	StartTime time.Time `json:"start_time"`
	EndTime   time.Time `json:"end_time"`
	Error     error     `json:"-"`
	Tunneled  bool      `json:"tunneled,omitempty"`
}

// Duration returns the time taken for the flow
func (f *Flow) Duration() time.Duration {
	if f.EndTime.IsZero() {
		return time.Since(f.StartTime)
	}
	return f.EndTime.Sub(f.StartTime)
}

// IsComplete returns true if the flow has a response or error
func (f *Flow) IsComplete() bool {
	return f.Response != nil || f.Error != nil
}

// Request represents an HTTP request
type Request struct {
	Method  string      `json:"method"`
	URL     string      `json:"url"`
	Host    string      `json:"host"`
	Path    string      `json:"path"`
	Proto   string      `json:"proto"`
	Headers http.Header `json:"headers,omitempty"`
	Body    []byte      `json:"body,omitempty"`
}

// Response represents an HTTP response
type Response struct {
	StatusCode int         `json:"status_code"`
	Status     string      `json:"status"`
	Proto      string      `json:"proto"`
	Headers    http.Header `json:"headers,omitempty"`
	Body       []byte      `json:"body,omitempty"`
}

// FlowEvent represents an event in the flow lifecycle
type FlowEvent struct {
	Type FlowEventType `json:"type"`
	Flow *Flow         `json:"flow"`
}

// FlowEventType indicates what happened to a flow
type FlowEventType int

const (
	FlowEventRequest FlowEventType = iota
	FlowEventResponse
	FlowEventError
)
