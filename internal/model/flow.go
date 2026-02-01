package model

import (
	"net/http"
	"time"
)

// FlowID is a unique identifier for a flow
type FlowID uint64

// Flow represents a complete HTTP request/response cycle
type Flow struct {
	ID        FlowID
	Request   *Request
	Response  *Response
	StartTime time.Time
	EndTime   time.Time
	Error     error
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
	Method  string
	URL     string
	Host    string
	Path    string
	Proto   string
	Headers http.Header
	Body    []byte
}

// Response represents an HTTP response
type Response struct {
	StatusCode int
	Status     string
	Proto      string
	Headers    http.Header
	Body       []byte
}

// FlowEvent represents an event in the flow lifecycle
type FlowEvent struct {
	Type FlowEventType
	Flow *Flow
}

// FlowEventType indicates what happened to a flow
type FlowEventType int

const (
	FlowEventRequest FlowEventType = iota
	FlowEventResponse
	FlowEventError
)
