package proxy

import (
	"sync"
	"testing"
	"time"

	"nabu/internal/model"
)

func makeFlow(method, url, host string) *model.Flow {
	return &model.Flow{
		StartTime: time.Now(),
		Request: &model.Request{
			Method: method,
			URL:    url,
			Host:   host,
			Path:   "/",
		},
	}
}

// --- Add ---

func TestFlowStore_Add(t *testing.T) {
	eventCh := make(chan model.FlowEvent, 100)
	store := NewFlowStore(1000, eventCh)

	f1 := makeFlow("GET", "http://a.com", "a.com")
	f2 := makeFlow("GET", "http://b.com", "b.com")
	id1 := store.Add(f1)
	id2 := store.Add(f2)

	if id1 >= id2 {
		t.Errorf("IDs should be sequential: %d >= %d", id1, id2)
	}
	if store.Count() != 2 {
		t.Errorf("Count() = %d, want 2", store.Count())
	}

	// Check event emission
	select {
	case evt := <-eventCh:
		if evt.Type != model.FlowEventRequest {
			t.Errorf("event type = %d, want FlowEventRequest", evt.Type)
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("expected event not received")
	}
}

// --- Get ---

func TestFlowStore_Get(t *testing.T) {
	store := NewFlowStore(1000, nil)
	f := makeFlow("GET", "http://a.com", "a.com")
	id := store.Add(f)

	got := store.Get(id)
	if got == nil {
		t.Fatal("Get returned nil")
	}
	if got.Request.URL != "http://a.com" {
		t.Errorf("URL = %q, want http://a.com", got.Request.URL)
	}
	if store.Get(999) != nil {
		t.Error("Get should return nil for unknown ID")
	}
}

// --- All ---

func TestFlowStore_All_ReturnsCopy(t *testing.T) {
	store := NewFlowStore(1000, nil)
	store.Add(makeFlow("GET", "http://a.com", "a.com"))

	all := store.All()
	all[0] = nil // mutate the copy
	if store.Get(1) == nil {
		t.Error("mutating All() result should not affect store")
	}
}

// --- Count, Clear ---

func TestFlowStore_Clear(t *testing.T) {
	store := NewFlowStore(1000, nil)
	store.Add(makeFlow("GET", "http://a.com", "a.com"))
	store.Add(makeFlow("GET", "http://b.com", "b.com"))
	store.Clear()

	if store.Count() != 0 {
		t.Errorf("Count after Clear = %d, want 0", store.Count())
	}
}

// --- Update ---

func TestFlowStore_Update(t *testing.T) {
	eventCh := make(chan model.FlowEvent, 100)
	store := NewFlowStore(1000, eventCh)

	f := makeFlow("GET", "http://a.com", "a.com")
	store.Add(f)
	// drain the Add event
	<-eventCh

	f.Response = &model.Response{StatusCode: 200}
	store.Update(f, model.FlowEventResponse)

	select {
	case evt := <-eventCh:
		if evt.Type != model.FlowEventResponse {
			t.Errorf("event type = %d, want FlowEventResponse", evt.Type)
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("expected update event not received")
	}
}

// --- Filter ---

func TestFlowStore_Filter(t *testing.T) {
	store := NewFlowStore(1000, nil)
	store.Add(makeFlow("GET", "http://a.com/path", "a.com"))
	store.Add(makeFlow("POST", "http://b.com/path", "b.com"))

	filter := model.NewFilterState()
	filter.Methods = []string{"GET"}

	result := store.Filter(filter)
	if len(result) != 1 {
		t.Errorf("Filter returned %d flows, want 1", len(result))
	}
}

func TestFlowStore_Filter_Nil(t *testing.T) {
	store := NewFlowStore(1000, nil)
	store.Add(makeFlow("GET", "http://a.com", "a.com"))

	result := store.Filter(nil)
	if len(result) != 1 {
		t.Errorf("Filter(nil) returned %d flows, want 1 (all)", len(result))
	}
}

// --- Eviction ---

func TestFlowStore_Eviction(t *testing.T) {
	store := NewFlowStore(10, nil) // capacity 10

	for i := 0; i < 10; i++ {
		store.Add(makeFlow("GET", "http://a.com", "a.com"))
	}
	if store.Count() != 10 {
		t.Fatalf("expected 10 flows, got %d", store.Count())
	}

	// Adding one more triggers eviction of oldest 10% (1 flow)
	store.Add(makeFlow("GET", "http://new.com", "new.com"))
	if store.Count() != 10 {
		t.Errorf("Count after eviction = %d, want 10", store.Count())
	}

	// The first flow should have been evicted
	if store.Get(1) != nil {
		t.Error("oldest flow should have been evicted")
	}
}

// --- Subscribe / Unsubscribe ---

func TestFlowStore_Subscribe(t *testing.T) {
	store := NewFlowStore(1000, nil)
	sub := store.Subscribe()

	store.Add(makeFlow("GET", "http://a.com", "a.com"))

	select {
	case evt := <-sub:
		if evt.Type != model.FlowEventRequest {
			t.Errorf("subscriber got type %d, want FlowEventRequest", evt.Type)
		}
	case <-time.After(100 * time.Millisecond):
		t.Error("subscriber did not receive event")
	}
}

func TestFlowStore_Unsubscribe(t *testing.T) {
	store := NewFlowStore(1000, nil)
	sub := store.Subscribe()
	store.Unsubscribe(sub)

	// Channel should be closed
	_, ok := <-sub
	if ok {
		t.Error("unsubscribed channel should be closed")
	}
}

// --- Non-blocking emit ---

func TestFlowStore_EmitNonBlocking(t *testing.T) {
	// Full channel should not block
	eventCh := make(chan model.FlowEvent, 1)
	store := NewFlowStore(1000, eventCh)

	// Fill the channel
	eventCh <- model.FlowEvent{}

	// This should not block
	done := make(chan struct{})
	go func() {
		store.Add(makeFlow("GET", "http://a.com", "a.com"))
		close(done)
	}()

	select {
	case <-done:
		// OK
	case <-time.After(time.Second):
		t.Fatal("Add blocked on full event channel")
	}
}

// --- IPC methods ---

func TestFlowStore_AddWithID(t *testing.T) {
	eventCh := make(chan model.FlowEvent, 100)
	store := NewFlowStore(1000, eventCh)

	f := makeFlow("GET", "http://a.com", "a.com")
	f.ID = 42
	store.AddWithID(f)

	if store.Get(42) == nil {
		t.Error("AddWithID should store with given ID")
	}

	// Should NOT emit events
	select {
	case <-eventCh:
		t.Error("AddWithID should not emit events")
	case <-time.After(50 * time.Millisecond):
		// OK
	}
}

func TestFlowStore_UpdateFromRemote_Existing(t *testing.T) {
	store := NewFlowStore(1000, nil)

	f := makeFlow("GET", "http://a.com", "a.com")
	f.ID = 10
	store.AddWithID(f)

	updated := makeFlow("GET", "http://a.com", "a.com")
	updated.ID = 10
	updated.Response = &model.Response{StatusCode: 200}
	store.UpdateFromRemote(updated, model.FlowEventResponse)

	got := store.Get(10)
	if got.Response == nil {
		t.Error("UpdateFromRemote should update existing flow")
	}
}

func TestFlowStore_UpdateFromRemote_Upsert(t *testing.T) {
	store := NewFlowStore(1000, nil)

	f := makeFlow("GET", "http://a.com", "a.com")
	f.ID = 99
	store.UpdateFromRemote(f, model.FlowEventRequest)

	if store.Get(99) == nil {
		t.Error("UpdateFromRemote should insert if not found")
	}
}

// --- Last ---

func TestFlowStore_Last(t *testing.T) {
	store := NewFlowStore(1000, nil)
	for i := 0; i < 5; i++ {
		store.Add(makeFlow("GET", "http://a.com", "a.com"))
	}

	got := store.Last(3)
	if len(got) != 3 {
		t.Errorf("Last(3) returned %d, want 3", len(got))
	}

	// Edge: 0
	if store.Last(0) != nil {
		t.Error("Last(0) should return nil")
	}

	// Edge: more than count
	got = store.Last(100)
	if len(got) != 5 {
		t.Errorf("Last(100) returned %d, want 5", len(got))
	}
}

// --- Concurrent access ---

func TestFlowStore_Concurrent(t *testing.T) {
	store := NewFlowStore(1000, nil)
	var wg sync.WaitGroup

	for i := 0; i < 100; i++ {
		wg.Add(3)
		go func() {
			defer wg.Done()
			store.Add(makeFlow("GET", "http://a.com", "a.com"))
		}()
		go func() {
			defer wg.Done()
			store.All()
		}()
		go func() {
			defer wg.Done()
			store.Filter(model.NewFilterState())
		}()
	}
	wg.Wait()
}
