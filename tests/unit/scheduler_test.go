package unit_test

import (
	"context"
	"fmt"
	"testing"
	"time"

	schedulerservice "clap/internal/modules/eventscheduler/service"
)

// fixedClock is a test clock that always returns the same time.
type fixedClock struct{ t time.Time }

func (c fixedClock) Now() time.Time { return c.t }

func newFixedClock(t time.Time) schedulerservice.Clock { return fixedClock{t: t} }

func TestScheduler_RegisterAndGetPending(t *testing.T) {
	now := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	clock := newFixedClock(now)
	s := schedulerservice.NewInMemoryScheduler(clock)
	ctx := context.Background()

	item := &schedulerservice.SchedulerItem{
		ID:        "evt-1",
		EventType: "song_start",
		ExecuteAt: now.Add(-1 * time.Second), // already due
	}

	if err := s.RegisterEvent(ctx, item); err != nil {
		t.Fatalf("RegisterEvent failed: %v", err)
	}

	if s.Size() != 1 {
		t.Fatalf("expected size 1, got %d", s.Size())
	}

	pending, err := s.GetPendingEvents(ctx, now)
	if err != nil {
		t.Fatalf("GetPendingEvents failed: %v", err)
	}
	if len(pending) != 1 {
		t.Fatalf("expected 1 pending event, got %d", len(pending))
	}
	if pending[0].ID != "evt-1" {
		t.Errorf("unexpected event ID: %s", pending[0].ID)
	}
}

func TestScheduler_FutureEventNotPending(t *testing.T) {
	now := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	s := schedulerservice.NewInMemoryScheduler(newFixedClock(now))
	ctx := context.Background()

	_ = s.RegisterEvent(ctx, &schedulerservice.SchedulerItem{
		ID:        "future",
		EventType: "timer_sync",
		ExecuteAt: now.Add(10 * time.Minute), // not yet due
	})

	pending, _ := s.GetPendingEvents(ctx, now)
	if len(pending) != 0 {
		t.Errorf("expected 0 pending events, got %d", len(pending))
	}
}

func TestScheduler_CancelEvent(t *testing.T) {
	now := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	s := schedulerservice.NewInMemoryScheduler(newFixedClock(now))
	ctx := context.Background()

	_ = s.RegisterEvent(ctx, &schedulerservice.SchedulerItem{
		ID:        "to-cancel",
		EventType: "flash",
		ExecuteAt: now,
	})

	if err := s.CancelEvent(ctx, "to-cancel"); err != nil {
		t.Fatalf("CancelEvent failed: %v", err)
	}
	if s.Size() != 0 {
		t.Errorf("expected size 0 after cancel, got %d", s.Size())
	}
}

func TestScheduler_CancelNonExistent(t *testing.T) {
	s := schedulerservice.NewInMemoryScheduler(newFixedClock(time.Now()))
	err := s.CancelEvent(context.Background(), "ghost")
	if err == nil {
		t.Error("expected error cancelling non-existent event")
	}
}

func TestScheduler_RescheduleEvent(t *testing.T) {
	now := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	s := schedulerservice.NewInMemoryScheduler(newFixedClock(now))
	ctx := context.Background()

	_ = s.RegisterEvent(ctx, &schedulerservice.SchedulerItem{
		ID:        "reschedule-me",
		EventType: "lyric_sync",
		ExecuteAt: now.Add(-1 * time.Second), // currently due
	})

	// Move it into the future — it should no longer appear as pending.
	future := now.Add(5 * time.Minute)
	if err := s.RescheduleEvent(ctx, "reschedule-me", future); err != nil {
		t.Fatalf("RescheduleEvent failed: %v", err)
	}

	pending, _ := s.GetPendingEvents(ctx, now)
	if len(pending) != 0 {
		t.Errorf("expected 0 pending events after reschedule, got %d", len(pending))
	}
}

func TestScheduler_DuplicateRegistration(t *testing.T) {
	s := schedulerservice.NewInMemoryScheduler(newFixedClock(time.Now()))
	ctx := context.Background()

	item := &schedulerservice.SchedulerItem{ID: "dup", EventType: "vibrate", ExecuteAt: time.Now()}
	_ = s.RegisterEvent(ctx, item)

	err := s.RegisterEvent(ctx, item)
	if err == nil {
		t.Error("expected conflict error on duplicate registration")
	}
}

func TestScheduler_PriorityOrdering(t *testing.T) {
	now := time.Date(2026, 1, 1, 12, 0, 0, 0, time.UTC)
	s := schedulerservice.NewInMemoryScheduler(newFixedClock(now))
	ctx := context.Background()

	// Register events out of order.
	times := []time.Time{
		now.Add(-3 * time.Second),
		now.Add(-1 * time.Second),
		now.Add(-2 * time.Second),
	}
	for i, at := range times {
		_ = s.RegisterEvent(ctx, &schedulerservice.SchedulerItem{
			ID:        fmt.Sprintf("ev-%d", i),
			EventType: "timer_sync",
			ExecuteAt: at,
		})
	}

	pending, _ := s.GetPendingEvents(ctx, now)
	if len(pending) != 3 {
		t.Fatalf("expected 3 pending events, got %d", len(pending))
	}
}

func TestScheduler_EmptyID(t *testing.T) {
	s := schedulerservice.NewInMemoryScheduler(newFixedClock(time.Now()))
	err := s.RegisterEvent(context.Background(), &schedulerservice.SchedulerItem{
		ID:        "",
		EventType: "flash",
		ExecuteAt: time.Now(),
	})
	if err == nil {
		t.Error("expected error for empty ID")
	}
}
