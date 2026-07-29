package service

import (
	"container/heap"
	"context"
	"sync"
	"time"

	sharederrors "clap/internal/shared/errors"
)

// Clock abstracts wall-clock access for deterministic unit tests.
type Clock interface {
	Now() time.Time
}

type realClock struct{}

func (realClock) Now() time.Time { return time.Now().UTC() }

// RealClock returns the production system clock.
func RealClock() Clock { return realClock{} }

// SchedulerItem is a single entry in the priority queue.
type SchedulerItem struct {
	ID          string
	SessionID   string
	EventType   string
	ExecuteAt   time.Time
	PayloadJSON string
}

// heapEntry wraps SchedulerItem with its heap array index for O(log n) removal.
type heapEntry struct {
	item  *SchedulerItem
	index int
}

// schedulerHeap implements container/heap (min-heap ordered by ExecuteAt).
type schedulerHeap []*heapEntry

func (h schedulerHeap) Len() int           { return len(h) }
func (h schedulerHeap) Less(i, j int) bool { return h[i].item.ExecuteAt.Before(h[j].item.ExecuteAt) }
func (h schedulerHeap) Swap(i, j int) {
	h[i], h[j] = h[j], h[i]
	h[i].index = i
	h[j].index = j
}
func (h *schedulerHeap) Push(x interface{}) {
	e := x.(*heapEntry)
	e.index = len(*h)
	*h = append(*h, e)
}
func (h *schedulerHeap) Pop() interface{} {
	old := *h
	n := len(old)
	e := old[n-1]
	old[n-1] = nil
	e.index = -1
	*h = old[:n-1]
	return e
}

// EventScheduler is the in-memory, thread-safe priority queue interface.
// It is intentionally stateless between restarts — callers hydrate it from
// the SchedulerEventRepository on startup.
type EventScheduler interface {
	// RegisterEvent adds an event. Returns conflict error if ID already registered.
	RegisterEvent(ctx context.Context, item *SchedulerItem) error
	// CancelEvent removes a pending event by ID.
	CancelEvent(ctx context.Context, id string) error
	// RescheduleEvent updates the execution time of a pending event and re-heapifies.
	RescheduleEvent(ctx context.Context, id string, newExecuteAt time.Time) error
	// GetPendingEvents returns all events due at or before upTo without removing them.
	// Callers are responsible for marking them executed/cancelled via the repository.
	GetPendingEvents(ctx context.Context, upTo time.Time) ([]*SchedulerItem, error)
	// Size returns the total number of registered events.
	Size() int
}

type inMemoryScheduler struct {
	mu    sync.RWMutex
	h     schedulerHeap
	index map[string]*heapEntry
	clock Clock
}

// NewInMemoryScheduler creates a production-ready, goroutine-safe event scheduler.
func NewInMemoryScheduler(clock Clock) EventScheduler {
	s := &inMemoryScheduler{
		h:     make(schedulerHeap, 0, 64),
		index: make(map[string]*heapEntry),
		clock: clock,
	}
	heap.Init(&s.h)
	return s
}

func (s *inMemoryScheduler) RegisterEvent(_ context.Context, item *SchedulerItem) error {
	if item.ID == "" {
		return sharederrors.NewBadRequest("event ID must not be empty", nil)
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.index[item.ID]; exists {
		return sharederrors.NewConflict("event already registered in scheduler", nil)
	}

	e := &heapEntry{item: item}
	heap.Push(&s.h, e)
	s.index[item.ID] = e
	return nil
}

func (s *inMemoryScheduler) CancelEvent(_ context.Context, id string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	e, exists := s.index[id]
	if !exists {
		return sharederrors.NewNotFound("event not found in scheduler", nil)
	}

	heap.Remove(&s.h, e.index)
	delete(s.index, id)
	return nil
}

func (s *inMemoryScheduler) RescheduleEvent(_ context.Context, id string, newExecuteAt time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	e, exists := s.index[id]
	if !exists {
		return sharederrors.NewNotFound("event not found in scheduler", nil)
	}

	e.item.ExecuteAt = newExecuteAt
	heap.Fix(&s.h, e.index)
	return nil
}

func (s *inMemoryScheduler) GetPendingEvents(_ context.Context, upTo time.Time) ([]*SchedulerItem, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var due []*SchedulerItem
	for _, e := range s.h {
		if !e.item.ExecuteAt.After(upTo) {
			due = append(due, e.item)
		}
	}
	return due, nil
}

func (s *inMemoryScheduler) Size() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.h.Len()
}
