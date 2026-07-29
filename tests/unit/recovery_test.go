package unit_test

import (
	"context"
	"testing"
	"time"

	schedulermodels "clap/internal/modules/eventscheduler/models"
	schedulerrepo "clap/internal/modules/eventscheduler/repository"
	schedulersvc "clap/internal/modules/eventscheduler/service"
	sharederrors "clap/internal/shared/errors"

	"github.com/google/uuid"
)

// ─── Stub event repository ────────────────────────────────────────────────────

type stubSchedulerEventRepo struct {
	events []*schedulermodels.SchedulerEvent
}

func (r *stubSchedulerEventRepo) Create(_ context.Context, ev *schedulermodels.SchedulerEvent) error {
	if ev.ID == uuid.Nil {
		ev.ID = uuid.New()
	}
	r.events = append(r.events, ev)
	return nil
}
func (r *stubSchedulerEventRepo) FindByID(_ context.Context, id uuid.UUID) (*schedulermodels.SchedulerEvent, error) {
	for _, ev := range r.events {
		if ev.ID == id {
			return ev, nil
		}
	}
	return nil, sharederrors.NewNotFound("not found", nil)
}
func (r *stubSchedulerEventRepo) FindPendingUpTo(_ context.Context, upTo time.Time) ([]*schedulermodels.SchedulerEvent, error) {
	var out []*schedulermodels.SchedulerEvent
	for _, ev := range r.events {
		if ev.Status == schedulermodels.SchedulerEventPending && !ev.ExecuteAt.After(upTo) {
			out = append(out, ev)
		}
	}
	return out, nil
}
func (r *stubSchedulerEventRepo) FindAllPending(_ context.Context) ([]*schedulermodels.SchedulerEvent, error) {
	var out []*schedulermodels.SchedulerEvent
	for _, ev := range r.events {
		if ev.Status == schedulermodels.SchedulerEventPending {
			out = append(out, ev)
		}
	}
	return out, nil
}
func (r *stubSchedulerEventRepo) UpdateStatus(_ context.Context, id uuid.UUID, status schedulermodels.SchedulerEventStatus) error {
	for _, ev := range r.events {
		if ev.ID == id {
			ev.Status = status
			return nil
		}
	}
	return sharederrors.NewNotFound("not found", nil)
}
func (r *stubSchedulerEventRepo) UpdateExecuteAt(_ context.Context, id uuid.UUID, at time.Time) error {
	for _, ev := range r.events {
		if ev.ID == id {
			ev.ExecuteAt = at
			return nil
		}
	}
	return sharederrors.NewNotFound("not found", nil)
}
func (r *stubSchedulerEventRepo) ClaimForProcessing(_ context.Context, id uuid.UUID) (*schedulermodels.SchedulerEvent, error) {
	for _, ev := range r.events {
		if ev.ID == id && ev.Status == schedulermodels.SchedulerEventPending {
			ev.Status = schedulermodels.SchedulerEventProcessing
			return ev, nil
		}
	}
	return nil, sharederrors.NewNotFound("not available", nil)
}
func (r *stubSchedulerEventRepo) MarkExecuted(_ context.Context, id uuid.UUID) error {
	return r.UpdateStatus(context.Background(), id, schedulermodels.SchedulerEventExecuted)
}
func (r *stubSchedulerEventRepo) MarkFailed(_ context.Context, id uuid.UUID, reason string) error {
	for _, ev := range r.events {
		if ev.ID == id {
			ev.Status = schedulermodels.SchedulerEventFailed
			ev.FailReason = reason
			return nil
		}
	}
	return sharederrors.NewNotFound("not found", nil)
}
func (r *stubSchedulerEventRepo) ResetStaleProcessing(_ context.Context, olderThan time.Time) (int64, error) {
	var reset int64
	for _, ev := range r.events {
		if ev.Status == schedulermodels.SchedulerEventProcessing && ev.UpdatedAt.Before(olderThan) {
			ev.Status = schedulermodels.SchedulerEventPending
			ev.FailReason = ""
			reset++
		}
	}
	return reset, nil
}
func (r *stubSchedulerEventRepo) DeleteTerminalOlderThan(_ context.Context, olderThan time.Time) (int64, error) {
	var kept []*schedulermodels.SchedulerEvent
	var deleted int64
	for _, ev := range r.events {
		terminal := ev.Status == schedulermodels.SchedulerEventExecuted ||
			ev.Status == schedulermodels.SchedulerEventFailed ||
			ev.Status == schedulermodels.SchedulerEventCancelled
		if terminal && ev.UpdatedAt.Before(olderThan) {
			deleted++
			continue
		}
		kept = append(kept, ev)
	}
	r.events = kept
	return deleted, nil
}

var _ schedulerrepo.SchedulerEventRepository = (*stubSchedulerEventRepo)(nil)

// ─── Tests ────────────────────────────────────────────────────────────────────

func TestRecovery_LoadsPendingEventsIntoScheduler(t *testing.T) {
	repo := &stubSchedulerEventRepo{}

	// Pre-seed three pending events as if they were written before a crash.
	sessionID := uuid.New()
	for i := 0; i < 3; i++ {
		repo.events = append(repo.events, &schedulermodels.SchedulerEvent{
			ID:          uuid.New(),
			SessionID:   sessionID,
			EventType:   "timer_sync",
			ExecuteAt:   time.Now().Add(time.Duration(i+1) * time.Minute),
			PayloadJSON: `{}`,
			Status:      schedulermodels.SchedulerEventPending,
		})
	}

	scheduler := schedulersvc.NewInMemoryScheduler(schedulersvc.RealClock())
	recoverySvc := schedulersvc.NewSchedulerRecoveryService(repo, scheduler)

	recovered, err := recoverySvc.RecoverPendingEvents(context.Background())
	if err != nil {
		t.Fatalf("RecoverPendingEvents failed: %v", err)
	}
	if recovered != 3 {
		t.Errorf("expected 3 recovered events, got %d", recovered)
	}
	if scheduler.Size() != 3 {
		t.Errorf("expected scheduler size=3, got %d", scheduler.Size())
	}
}

func TestRecovery_IdempotentOnDoubleCall(t *testing.T) {
	repo := &stubSchedulerEventRepo{}
	repo.events = append(repo.events, &schedulermodels.SchedulerEvent{
		ID:          uuid.New(),
		SessionID:   uuid.New(),
		EventType:   "song_start",
		ExecuteAt:   time.Now().Add(time.Minute),
		PayloadJSON: `{}`,
		Status:      schedulermodels.SchedulerEventPending,
	})

	scheduler := schedulersvc.NewInMemoryScheduler(schedulersvc.RealClock())
	recoverySvc := schedulersvc.NewSchedulerRecoveryService(repo, scheduler)

	// First recovery call.
	first, err := recoverySvc.RecoverPendingEvents(context.Background())
	if err != nil {
		t.Fatalf("first RecoverPendingEvents failed: %v", err)
	}

	// Second recovery call (simulates restart of a graceful shutdown that
	// preserved the in-memory queue state — e.g., via a blue/green deploy).
	second, err := recoverySvc.RecoverPendingEvents(context.Background())
	if err != nil {
		t.Fatalf("second RecoverPendingEvents failed: %v", err)
	}

	if first != 1 {
		t.Errorf("expected first recovery count=1, got %d", first)
	}
	if second != 0 {
		t.Errorf("expected second recovery count=0 (already registered), got %d", second)
	}
	if scheduler.Size() != 1 {
		t.Errorf("expected scheduler size=1, got %d", scheduler.Size())
	}
}

func TestRecovery_ExecutedEventsAreSkipped(t *testing.T) {
	repo := &stubSchedulerEventRepo{}
	repo.events = append(repo.events,
		&schedulermodels.SchedulerEvent{
			ID:        uuid.New(),
			SessionID: uuid.New(),
			EventType: "song_start",
			ExecuteAt: time.Now().Add(time.Minute),
			Status:    schedulermodels.SchedulerEventExecuted, // already done
		},
		&schedulermodels.SchedulerEvent{
			ID:        uuid.New(),
			SessionID: uuid.New(),
			EventType: "song_stop",
			ExecuteAt: time.Now().Add(2 * time.Minute),
			Status:    schedulermodels.SchedulerEventPending, // needs recovery
		},
	)

	scheduler := schedulersvc.NewInMemoryScheduler(schedulersvc.RealClock())
	recoverySvc := schedulersvc.NewSchedulerRecoveryService(repo, scheduler)

	recovered, _ := recoverySvc.RecoverPendingEvents(context.Background())
	if recovered != 1 {
		t.Errorf("expected 1 recovered event (executed one must be skipped), got %d", recovered)
	}
}

// ─── Event execution safety tests ─────────────────────────────────────────────

func TestExecSafety_ClaimForProcessingMutatesStatus(t *testing.T) {
	repo := &stubSchedulerEventRepo{}
	evID := uuid.New()
	repo.events = append(repo.events, &schedulermodels.SchedulerEvent{
		ID:        evID,
		SessionID: uuid.New(),
		EventType: "timer_sync",
		ExecuteAt: time.Now(),
		Status:    schedulermodels.SchedulerEventPending,
	})

	claimed, err := repo.ClaimForProcessing(context.Background(), evID)
	if err != nil {
		t.Fatalf("ClaimForProcessing failed: %v", err)
	}
	if claimed.Status != schedulermodels.SchedulerEventProcessing {
		t.Errorf("expected processing status, got %s", claimed.Status)
	}
}

func TestExecSafety_DoubleClaimReturnNotFound(t *testing.T) {
	repo := &stubSchedulerEventRepo{}
	evID := uuid.New()
	repo.events = append(repo.events, &schedulermodels.SchedulerEvent{
		ID:        evID,
		SessionID: uuid.New(),
		EventType: "flash",
		ExecuteAt: time.Now(),
		Status:    schedulermodels.SchedulerEventPending,
	})

	_, _ = repo.ClaimForProcessing(context.Background(), evID) // first claim succeeds

	_, err := repo.ClaimForProcessing(context.Background(), evID) // second claim fails
	if err == nil {
		t.Error("expected error on double claim — event is no longer pending")
	}
}

func TestExecSafety_MarkFailedStoresReason(t *testing.T) {
	repo := &stubSchedulerEventRepo{}
	evID := uuid.New()
	repo.events = append(repo.events, &schedulermodels.SchedulerEvent{
		ID:        evID,
		SessionID: uuid.New(),
		EventType: "vibrate",
		ExecuteAt: time.Now(),
		Status:    schedulermodels.SchedulerEventProcessing,
	})

	if err := repo.MarkFailed(context.Background(), evID, "gateway timeout"); err != nil {
		t.Fatalf("MarkFailed failed: %v", err)
	}

	ev, _ := repo.FindByID(context.Background(), evID)
	if ev.Status != schedulermodels.SchedulerEventFailed {
		t.Errorf("expected failed status, got %s", ev.Status)
	}
	if ev.FailReason != "gateway timeout" {
		t.Errorf("expected fail_reason='gateway timeout', got %q", ev.FailReason)
	}
}
