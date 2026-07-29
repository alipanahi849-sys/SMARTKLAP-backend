package service

import (
	"context"
	"encoding/json"
	"time"

	"clap/internal/modules/eventscheduler/models"
	"clap/internal/modules/eventscheduler/repository"
	sharederrors "clap/internal/shared/errors"

	"github.com/google/uuid"
)

// RegisterEventRequest is the input for registering a new scheduled event.
type RegisterEventRequest struct {
	SessionID uuid.UUID
	EventType string
	ExecuteAt time.Time
	Payload   interface{}
}

// EventSchedulerService orchestrates durable event scheduling:
// it persists to DB for crash recovery and registers in the in-memory
// priority queue for low-latency reads.
type EventSchedulerService interface {
	RegisterEvent(ctx context.Context, req *RegisterEventRequest) (*models.SchedulerEvent, error)
	CancelEvent(ctx context.Context, eventID uuid.UUID) error
	RescheduleEvent(ctx context.Context, eventID uuid.UUID, newExecuteAt time.Time) error
	GetPendingEvents(ctx context.Context, upTo time.Time) ([]*models.SchedulerEvent, error)
}

type eventSchedulerService struct {
	repo      repository.SchedulerEventRepository
	scheduler EventScheduler
}

func NewEventSchedulerService(
	repo repository.SchedulerEventRepository,
	scheduler EventScheduler,
) EventSchedulerService {
	return &eventSchedulerService{repo: repo, scheduler: scheduler}
}

func (s *eventSchedulerService) RegisterEvent(ctx context.Context, req *RegisterEventRequest) (*models.SchedulerEvent, error) {
	if req.EventType == "" {
		return nil, sharederrors.NewBadRequest("event_type is required", nil)
	}
	if req.ExecuteAt.IsZero() {
		return nil, sharederrors.NewBadRequest("execute_at is required", nil)
	}

	payloadBytes, err := json.Marshal(req.Payload)
	if err != nil {
		return nil, sharederrors.NewBadRequest("Invalid event payload", err)
	}

	ev := &models.SchedulerEvent{
		SessionID:   req.SessionID,
		EventType:   req.EventType,
		ExecuteAt:   req.ExecuteAt,
		PayloadJSON: string(payloadBytes),
		Status:      models.SchedulerEventPending,
	}

	if err := s.repo.Create(ctx, ev); err != nil {
		return nil, err
	}

	// Mirror into the in-memory queue. Non-fatal if queue already has it —
	// it can be re-hydrated from DB on restart.
	item := &SchedulerItem{
		ID:          ev.ID.String(),
		SessionID:   ev.SessionID.String(),
		EventType:   ev.EventType,
		ExecuteAt:   ev.ExecuteAt,
		PayloadJSON: ev.PayloadJSON,
	}
	_ = s.scheduler.RegisterEvent(ctx, item)

	return ev, nil
}

func (s *eventSchedulerService) CancelEvent(ctx context.Context, eventID uuid.UUID) error {
	if err := s.repo.UpdateStatus(ctx, eventID, models.SchedulerEventCancelled); err != nil {
		return err
	}
	// Best-effort removal from in-memory queue (may already be absent).
	_ = s.scheduler.CancelEvent(ctx, eventID.String())
	return nil
}

func (s *eventSchedulerService) RescheduleEvent(ctx context.Context, eventID uuid.UUID, newExecuteAt time.Time) error {
	if err := s.repo.UpdateExecuteAt(ctx, eventID, newExecuteAt); err != nil {
		return err
	}
	return s.scheduler.RescheduleEvent(ctx, eventID.String(), newExecuteAt)
}

func (s *eventSchedulerService) GetPendingEvents(ctx context.Context, upTo time.Time) ([]*models.SchedulerEvent, error) {
	return s.repo.FindPendingUpTo(ctx, upTo)
}
