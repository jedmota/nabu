package ipc

import (
	"encoding/json"
	"errors"
	"net/http"
	"testing"
	"time"

	"proxy-tui/internal/model"
)

// --- FlowToWire ---

func TestFlowToWire_WithError(t *testing.T) {
	f := &model.Flow{
		ID:        1,
		StartTime: time.Now(),
		EndTime:   time.Now(),
		Error:     errors.New("connection reset"),
		Request:   &model.Request{Method: "GET", URL: "http://a.com", Host: "a.com"},
	}
	w := FlowToWire(f)
	if w.ErrorString != "connection reset" {
		t.Errorf("ErrorString = %q, want %q", w.ErrorString, "connection reset")
	}
}

func TestFlowToWire_WithoutError(t *testing.T) {
	f := &model.Flow{
		ID:        2,
		StartTime: time.Now(),
		Request:   &model.Request{Method: "GET", URL: "http://a.com", Host: "a.com"},
	}
	w := FlowToWire(f)
	if w.ErrorString != "" {
		t.Errorf("ErrorString = %q, want empty", w.ErrorString)
	}
}

// --- WireToFlow ---

func TestWireToFlow_WithErrorString(t *testing.T) {
	w := FlowWire{
		ID:          3,
		ErrorString: "timeout",
		StartTime:   time.Now(),
	}
	f := WireToFlow(w)
	if f.Error == nil || f.Error.Error() != "timeout" {
		t.Errorf("Error = %v, want timeout", f.Error)
	}
}

func TestWireToFlow_WithoutErrorString(t *testing.T) {
	w := FlowWire{ID: 4, StartTime: time.Now()}
	f := WireToFlow(w)
	if f.Error != nil {
		t.Error("Error should be nil")
	}
}

// --- Round-trip ---

func TestFlowWire_RoundTrip(t *testing.T) {
	now := time.Now().Truncate(time.Millisecond) // truncate for JSON round-trip
	original := &model.Flow{
		ID:        42,
		StartTime: now,
		EndTime:   now.Add(100 * time.Millisecond),
		Tunneled:  true,
		Error:     errors.New("some error"),
		Request: &model.Request{
			Method:  "POST",
			URL:     "http://example.com/api",
			Host:    "example.com",
			Path:    "/api",
			Proto:   "HTTP/1.1",
			Headers: http.Header{"Content-Type": []string{"application/json"}},
			Body:    []byte(`{"key":"value"}`),
		},
		Response: &model.Response{
			StatusCode: 200,
			Status:     "200 OK",
			Proto:      "HTTP/1.1",
			Headers:    http.Header{"X-Custom": []string{"yes"}},
			Body:       []byte(`{"result":true}`),
		},
	}

	wire := FlowToWire(original)
	restored := WireToFlow(wire)

	if restored.ID != original.ID {
		t.Errorf("ID: got %d, want %d", restored.ID, original.ID)
	}
	if restored.Tunneled != original.Tunneled {
		t.Errorf("Tunneled: got %v, want %v", restored.Tunneled, original.Tunneled)
	}
	if restored.Error == nil || restored.Error.Error() != "some error" {
		t.Errorf("Error: got %v, want 'some error'", restored.Error)
	}
	if restored.Request.Method != "POST" {
		t.Errorf("Request.Method: got %q", restored.Request.Method)
	}
	if restored.Response.StatusCode != 200 {
		t.Errorf("Response.StatusCode: got %d", restored.Response.StatusCode)
	}
}

// --- MarshalMessage / UnmarshalMessage ---

func TestMarshalUnmarshal_RoundTrip(t *testing.T) {
	tests := []struct {
		msgType string
		payload interface{}
	}{
		{"hello", HelloPayload{Port: 9090, BindAddress: "0.0.0.0", FlowCount: 5}},
		{"sync", SyncPayload{Flows: []FlowWire{{ID: 1}}, Batch: 1, TotalBatch: 1}},
		{"sync_done", SyncDonePayload{}},
		{"flow_event", FlowEventPayload{EventType: model.FlowEventResponse, Flow: FlowWire{ID: 2}}},
		{"config_reload", ConfigReloadPayload{}},
	}

	for _, tt := range tests {
		t.Run(tt.msgType, func(t *testing.T) {
			data, err := MarshalMessage(tt.msgType, tt.payload)
			if err != nil {
				t.Fatalf("MarshalMessage: %v", err)
			}

			msg, err := UnmarshalMessage(data)
			if err != nil {
				t.Fatalf("UnmarshalMessage: %v", err)
			}
			if msg.Type != tt.msgType {
				t.Errorf("Type = %q, want %q", msg.Type, tt.msgType)
			}
		})
	}
}

func TestUnmarshalMessage_HelloPayload(t *testing.T) {
	data, _ := MarshalMessage("hello", HelloPayload{Port: 8080, BindAddress: "127.0.0.1", FlowCount: 10})
	msg, _ := UnmarshalMessage(data)

	var hello HelloPayload
	if err := json.Unmarshal(msg.Payload, &hello); err != nil {
		t.Fatal(err)
	}
	if hello.Port != 8080 {
		t.Errorf("Port = %d, want 8080", hello.Port)
	}
	if hello.BindAddress != "127.0.0.1" {
		t.Errorf("BindAddress = %q", hello.BindAddress)
	}
}

func TestUnmarshalMessage_InvalidJSON(t *testing.T) {
	_, err := UnmarshalMessage([]byte("not json"))
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}
