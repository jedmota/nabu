package proxy

import (
	"sync"
	"sync/atomic"

	"proxy-tui/internal/model"
)

// FlowStore manages the collection of flows
type FlowStore struct {
	flows       []*model.Flow
	flowMap     map[model.FlowID]*model.Flow
	mu          sync.RWMutex
	nextID      uint64
	maxFlows    int
	events      chan model.FlowEvent
	subscribers []chan model.FlowEvent
	subMu       sync.Mutex
}

// NewFlowStore creates a new flow store
func NewFlowStore(maxFlows int, eventChan chan model.FlowEvent) *FlowStore {
	if maxFlows <= 0 {
		maxFlows = 10000
	}
	return &FlowStore{
		flows:    make([]*model.Flow, 0, maxFlows),
		flowMap:  make(map[model.FlowID]*model.Flow),
		maxFlows: maxFlows,
		events:   eventChan,
	}
}

// Subscribe returns a new channel that receives all flow events.
// Used by the IPC server to fan out events to connected clients.
func (s *FlowStore) Subscribe() <-chan model.FlowEvent {
	ch := make(chan model.FlowEvent, 256)
	s.subMu.Lock()
	s.subscribers = append(s.subscribers, ch)
	s.subMu.Unlock()
	return ch
}

// Unsubscribe removes a subscriber channel.
func (s *FlowStore) Unsubscribe(ch <-chan model.FlowEvent) {
	s.subMu.Lock()
	defer s.subMu.Unlock()
	for i, sub := range s.subscribers {
		if sub == ch {
			close(sub)
			s.subscribers = append(s.subscribers[:i], s.subscribers[i+1:]...)
			return
		}
	}
}

// emit sends an event to the primary channel and all subscriber channels (non-blocking).
func (s *FlowStore) emit(event model.FlowEvent) {
	if s.events != nil {
		select {
		case s.events <- event:
		default:
		}
	}
	s.subMu.Lock()
	subs := make([]chan model.FlowEvent, len(s.subscribers))
	copy(subs, s.subscribers)
	s.subMu.Unlock()
	for _, ch := range subs {
		select {
		case ch <- event:
		default:
		}
	}
}

// Add adds a new flow and returns its ID
func (s *FlowStore) Add(flow *model.Flow) model.FlowID {
	id := model.FlowID(atomic.AddUint64(&s.nextID, 1))
	flow.ID = id

	s.mu.Lock()
	// Evict old flows if at capacity
	if len(s.flows) >= s.maxFlows {
		// Remove oldest 10%
		removeCount := s.maxFlows / 10
		for i := 0; i < removeCount; i++ {
			delete(s.flowMap, s.flows[i].ID)
		}
		s.flows = s.flows[removeCount:]
	}
	s.flows = append(s.flows, flow)
	s.flowMap[id] = flow
	s.mu.Unlock()

	s.emit(model.FlowEvent{Type: model.FlowEventRequest, Flow: flow})

	return id
}

// Update updates an existing flow (e.g., with response)
func (s *FlowStore) Update(flow *model.Flow, eventType model.FlowEventType) {
	s.mu.Lock()
	s.flowMap[flow.ID] = flow
	s.mu.Unlock()

	s.emit(model.FlowEvent{Type: eventType, Flow: flow})
}

// Get retrieves a flow by ID
func (s *FlowStore) Get(id model.FlowID) *model.Flow {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.flowMap[id]
}

// All returns all flows (copy of slice)
func (s *FlowStore) All() []*model.Flow {
	s.mu.RLock()
	defer s.mu.RUnlock()
	result := make([]*model.Flow, len(s.flows))
	copy(result, s.flows)
	return result
}

// Count returns the number of flows
func (s *FlowStore) Count() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.flows)
}

// Clear removes all flows
func (s *FlowStore) Clear() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.flows = make([]*model.Flow, 0, s.maxFlows)
	s.flowMap = make(map[model.FlowID]*model.Flow)
}

// Filter returns flows matching the filter
func (s *FlowStore) Filter(filter *model.FilterState) []*model.Flow {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if filter == nil {
		result := make([]*model.Flow, len(s.flows))
		copy(result, s.flows)
		return result
	}

	result := make([]*model.Flow, 0)
	for _, flow := range s.flows {
		if filter.Match(flow) {
			result = append(result, flow)
		}
	}
	return result
}

// AddWithID inserts a flow that already has an ID (used by IPC clients
// receiving flows from a primary instance). It does NOT emit events.
func (s *FlowStore) AddWithID(flow *model.Flow) {
	s.mu.Lock()
	if len(s.flows) >= s.maxFlows {
		removeCount := s.maxFlows / 10
		for i := 0; i < removeCount; i++ {
			delete(s.flowMap, s.flows[i].ID)
		}
		s.flows = s.flows[removeCount:]
	}
	s.flows = append(s.flows, flow)
	s.flowMap[flow.ID] = flow
	s.mu.Unlock()
}

// UpdateFromRemote updates an existing flow by ID, or inserts it if not found.
// Used by IPC clients. It does NOT emit events.
func (s *FlowStore) UpdateFromRemote(flow *model.Flow, eventType model.FlowEventType) {
	s.mu.Lock()
	if existing, ok := s.flowMap[flow.ID]; ok {
		// Update in place
		*existing = *flow
	} else {
		// Not found — insert it
		s.flows = append(s.flows, flow)
		s.flowMap[flow.ID] = flow
	}
	s.mu.Unlock()
}

// Last returns the last n flows
func (s *FlowStore) Last(n int) []*model.Flow {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if n <= 0 || len(s.flows) == 0 {
		return nil
	}
	if n > len(s.flows) {
		n = len(s.flows)
	}
	start := len(s.flows) - n
	result := make([]*model.Flow, n)
	copy(result, s.flows[start:])
	return result
}
