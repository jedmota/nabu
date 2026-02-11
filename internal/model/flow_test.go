package model

import (
	"errors"
	"testing"
	"time"
)

func TestFlow_Duration_Complete(t *testing.T) {
	start := time.Now().Add(-2 * time.Second)
	end := start.Add(500 * time.Millisecond)
	f := &Flow{StartTime: start, EndTime: end}

	d := f.Duration()
	if d != 500*time.Millisecond {
		t.Errorf("Duration() = %v, want 500ms", d)
	}
}

func TestFlow_Duration_InProgress(t *testing.T) {
	f := &Flow{StartTime: time.Now().Add(-100 * time.Millisecond)}
	d := f.Duration()
	if d < 100*time.Millisecond {
		t.Errorf("Duration() = %v, want >= 100ms for in-progress flow", d)
	}
}

func TestFlow_IsComplete_WithResponse(t *testing.T) {
	f := &Flow{Response: &Response{StatusCode: 200}}
	if !f.IsComplete() {
		t.Error("flow with Response should be complete")
	}
}

func TestFlow_IsComplete_WithError(t *testing.T) {
	f := &Flow{Error: errors.New("fail")}
	if !f.IsComplete() {
		t.Error("flow with Error should be complete")
	}
}

func TestFlow_IsComplete_Neither(t *testing.T) {
	f := &Flow{}
	if f.IsComplete() {
		t.Error("flow with neither Response nor Error should not be complete")
	}
}

func TestFlowEventType_Iota(t *testing.T) {
	if FlowEventRequest != 0 {
		t.Errorf("FlowEventRequest = %d, want 0", FlowEventRequest)
	}
	if FlowEventResponse != 1 {
		t.Errorf("FlowEventResponse = %d, want 1", FlowEventResponse)
	}
	if FlowEventError != 2 {
		t.Errorf("FlowEventError = %d, want 2", FlowEventError)
	}
}
