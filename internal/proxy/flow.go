package proxy

import (
	"sync"
	"sync/atomic"

	"proxy-tui/internal/model"
)

// FlowStore manages the collection of flows
type FlowStore struct {
	flows    []*model.Flow
	flowMap  map[model.FlowID]*model.Flow
	mu       sync.RWMutex
	nextID   uint64
	maxFlows int
	events   chan model.FlowEvent
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

	// Send event
	if s.events != nil {
		select {
		case s.events <- model.FlowEvent{Type: model.FlowEventRequest, Flow: flow}:
		default:
			// Don't block if channel is full
		}
	}

	return id
}

// Update updates an existing flow (e.g., with response)
func (s *FlowStore) Update(flow *model.Flow, eventType model.FlowEventType) {
	s.mu.Lock()
	s.flowMap[flow.ID] = flow
	s.mu.Unlock()

	// Send event
	if s.events != nil {
		select {
		case s.events <- model.FlowEvent{Type: eventType, Flow: flow}:
		default:
		}
	}
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
