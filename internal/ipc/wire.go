package ipc

import (
	"encoding/json"
	"errors"
	"time"

	"proxy-tui/internal/model"
)

// IPCMessage is the envelope for all IPC messages.
type IPCMessage struct {
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"payload"`
}

// HelloPayload is sent by the server immediately after a client connects.
type HelloPayload struct {
	Port        int    `json:"port"`
	BindAddress string `json:"bind_address"`
	FlowCount   int    `json:"flow_count"`
}

// SyncPayload carries a batch of existing flows during initial sync.
type SyncPayload struct {
	Flows       []FlowWire `json:"flows"`
	Batch       int        `json:"batch"`
	TotalBatch  int        `json:"total_batches"`
}

// SyncDonePayload signals the end of sync.
type SyncDonePayload struct{}

// FlowEventPayload wraps a single flow event for real-time streaming.
type FlowEventPayload struct {
	EventType model.FlowEventType `json:"event_type"`
	Flow      FlowWire            `json:"flow"`
}

// FlowWire is the wire-format mirror of model.Flow.
// It replaces Error (error) with ErrorString (string).
type FlowWire struct {
	ID          model.FlowID    `json:"id"`
	Request     *model.Request  `json:"request,omitempty"`
	Response    *model.Response `json:"response,omitempty"`
	StartTime   time.Time       `json:"start_time"`
	EndTime     time.Time       `json:"end_time"`
	ErrorString string          `json:"error,omitempty"`
	Tunneled    bool            `json:"tunneled,omitempty"`
}

// FlowToWire converts a model.Flow to the wire format.
func FlowToWire(f *model.Flow) FlowWire {
	w := FlowWire{
		ID:        f.ID,
		Request:   f.Request,
		Response:  f.Response,
		StartTime: f.StartTime,
		EndTime:   f.EndTime,
		Tunneled:  f.Tunneled,
	}
	if f.Error != nil {
		w.ErrorString = f.Error.Error()
	}
	return w
}

// WireToFlow converts a wire-format flow back to a model.Flow.
func WireToFlow(w FlowWire) *model.Flow {
	f := &model.Flow{
		ID:        w.ID,
		Request:   w.Request,
		Response:  w.Response,
		StartTime: w.StartTime,
		EndTime:   w.EndTime,
		Tunneled:  w.Tunneled,
	}
	if w.ErrorString != "" {
		f.Error = errors.New(w.ErrorString)
	}
	return f
}

// MarshalMessage encodes an IPCMessage as JSON bytes (without trailing newline).
func MarshalMessage(msgType string, payload interface{}) ([]byte, error) {
	payloadBytes, err := json.Marshal(payload)
	if err != nil {
		return nil, err
	}
	msg := IPCMessage{
		Type:    msgType,
		Payload: payloadBytes,
	}
	return json.Marshal(msg)
}

// UnmarshalMessage decodes a JSON line into an IPCMessage.
func UnmarshalMessage(data []byte) (*IPCMessage, error) {
	var msg IPCMessage
	if err := json.Unmarshal(data, &msg); err != nil {
		return nil, err
	}
	return &msg, nil
}
