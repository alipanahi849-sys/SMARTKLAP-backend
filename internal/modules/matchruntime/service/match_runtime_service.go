package service

import (
	"context"
	"time"

	"clap/internal/modules/matchruntime/dto"
	"clap/internal/modules/matchruntime/models"
	"clap/internal/modules/matchruntime/repository"
	sharederrors "clap/internal/shared/errors"
	"clap/internal/shared/logger"
	"clap/internal/shared/utils"

	"github.com/google/uuid"
)

// Clock abstracts wall-clock access for deterministic testing.
type Clock interface {
	Now() time.Time
}

type realClock struct{}

func (realClock) Now() time.Time { return time.Now().UTC() }

// RealClock returns the production clock.
func RealClock() Clock { return realClock{} }

// MatchEventPublisher is the transport-agnostic interface for delivering
// match state changes to connected clients.  Business logic uses this interface
// and knows nothing about WebSockets, Redis, or any transport layer.
//
// The production implementation is WebSocketRealtimeGateway.
// Tests can supply a stub or nil (publishing is skipped when nil).
type MatchEventPublisher interface {
	PublishMatchEvent(ctx context.Context, matchID uuid.UUID, eventType string, payload any) error
}

// MatchRuntimeService controls the timer lifecycle for a match.
// All state is persisted to PostgreSQL — the service is safe to restart
// without losing timer accuracy.
//
// Transition rules (enforced via models.ValidateTransition):
//
//	pending  → running   (StartMatch)
//	running  → paused    (PauseMatch)
//	paused   → running   (ResumeMatch)
//	running  → ended     (EndMatch)
//	paused   → ended     (EndMatch)
type MatchRuntimeService interface {
	StartMatch(ctx context.Context, matchID uuid.UUID, authCtx *utils.AuthorizationContext) (*dto.MatchRuntimeResponse, error)
	PauseMatch(ctx context.Context, matchID uuid.UUID, authCtx *utils.AuthorizationContext) (*dto.MatchRuntimeResponse, error)
	ResumeMatch(ctx context.Context, matchID uuid.UUID, authCtx *utils.AuthorizationContext) (*dto.MatchRuntimeResponse, error)
	EndMatch(ctx context.Context, matchID uuid.UUID, authCtx *utils.AuthorizationContext) (*dto.MatchRuntimeResponse, error)
	GetState(ctx context.Context, matchID uuid.UUID) (*dto.MatchRuntimeResponse, error)
	CurrentMatchTime(ctx context.Context, matchID uuid.UUID) (*dto.MatchTimeResponse, error)
}

type matchRuntimeService struct {
	repo      repository.MatchRuntimeRepository
	clock     Clock
	publisher MatchEventPublisher // optional; nil disables realtime delivery
}

// NewMatchRuntimeService constructs the service without realtime publishing.
// Existing tests and routes that don't need event delivery use this constructor.
func NewMatchRuntimeService(repo repository.MatchRuntimeRepository, clock Clock) MatchRuntimeService {
	return &matchRuntimeService{repo: repo, clock: clock}
}

// NewMatchRuntimeServiceWithPublisher constructs the service with an event publisher.
// Used in production to deliver match.runtime.updated events to WebSocket clients.
func NewMatchRuntimeServiceWithPublisher(
	repo repository.MatchRuntimeRepository,
	clock Clock,
	publisher MatchEventPublisher,
) MatchRuntimeService {
	return &matchRuntimeService{repo: repo, clock: clock, publisher: publisher}
}

// publishEvent is a best-effort helper: log errors but never fail the caller.
func (s *matchRuntimeService) publishEvent(ctx context.Context, matchID uuid.UUID, status string, elapsedMs int64) {
	if s.publisher == nil {
		return
	}
	payload := map[string]any{
		"status":     status,
		"elapsed_ms": elapsedMs,
	}
	if err := s.publisher.PublishMatchEvent(ctx, matchID, "match.runtime.updated", payload); err != nil {
		// Non-fatal — realtime delivery failure must not abort the mutation, but
		// it MUST be observable (CR-10 / F-012).
		logger.Error().
			Str("match_id", matchID.String()).
			Str("event_type", "match.runtime.updated").
			Str("status", status).
			Int64("elapsed_ms", elapsedMs).
			Err(err).
			Msg("realtime publish failed for match.runtime.updated")
	}
}

// StartMatch transitions a match from pending → running.
// If no runtime state exists yet a fresh pending record is created first.
// Calling StartMatch on a running or ended match returns an error.
func (s *matchRuntimeService) StartMatch(ctx context.Context, matchID uuid.UUID, authCtx *utils.AuthorizationContext) (*dto.MatchRuntimeResponse, error) {
	if err := authCtx.RequireAdmin(); err != nil {
		return nil, sharederrors.NewForbidden("Admin access required", err)
	}

	now := s.clock.Now()
	userID := authCtx.UserID

	existing, err := s.repo.FindByMatchID(ctx, matchID)
	if err == nil {
		// State exists — validate transition pending → running.
		if err := models.ValidateTransition(existing.Status, models.RuntimeStatusRunning); err != nil {
			return nil, err
		}
		existing.Status = models.RuntimeStatusRunning
		existing.StartedAt = &now
		existing.UpdatedBy = &userID
		if err := s.repo.Update(ctx, existing); err != nil {
			return nil, err
		}
		resp := dto.ToMatchRuntimeResponse(existing)
		s.publishEvent(ctx, matchID, string(models.RuntimeStatusRunning), s.computeElapsed(existing).ElapsedMs)
		return resp, nil
	}

	// No existing state — create fresh record at pending then immediately start.
	state := &models.MatchRuntimeState{
		MatchID:   matchID,
		Status:    models.RuntimeStatusRunning,
		StartedAt: &now,
		CreatedBy: &userID,
	}
	if err := s.repo.Create(ctx, state); err != nil {
		return nil, err
	}
	resp := dto.ToMatchRuntimeResponse(state)
	s.publishEvent(ctx, matchID, string(models.RuntimeStatusRunning), 0)
	return resp, nil
}

// PauseMatch transitions running → paused.
func (s *matchRuntimeService) PauseMatch(ctx context.Context, matchID uuid.UUID, authCtx *utils.AuthorizationContext) (*dto.MatchRuntimeResponse, error) {
	if err := authCtx.RequireAdmin(); err != nil {
		return nil, sharederrors.NewForbidden("Admin access required", err)
	}

	state, err := s.repo.FindByMatchID(ctx, matchID)
	if err != nil {
		return nil, err
	}
	if err := models.ValidateTransition(state.Status, models.RuntimeStatusPaused); err != nil {
		return nil, err
	}

	now := s.clock.Now()
	userID := authCtx.UserID
	state.Status = models.RuntimeStatusPaused
	state.PausedAt = &now
	state.UpdatedBy = &userID

	if err := s.repo.Update(ctx, state); err != nil {
		return nil, err
	}
	resp := dto.ToMatchRuntimeResponse(state)
	s.publishEvent(ctx, matchID, string(models.RuntimeStatusPaused), s.computeElapsed(state).ElapsedMs)
	return resp, nil
}

// ResumeMatch transitions paused → running.
func (s *matchRuntimeService) ResumeMatch(ctx context.Context, matchID uuid.UUID, authCtx *utils.AuthorizationContext) (*dto.MatchRuntimeResponse, error) {
	if err := authCtx.RequireAdmin(); err != nil {
		return nil, sharederrors.NewForbidden("Admin access required", err)
	}

	state, err := s.repo.FindByMatchID(ctx, matchID)
	if err != nil {
		return nil, err
	}
	if err := models.ValidateTransition(state.Status, models.RuntimeStatusRunning); err != nil {
		return nil, err
	}

	now := s.clock.Now()
	userID := authCtx.UserID

	// Accumulate the paused interval into the running total.
	if state.PausedAt != nil {
		state.TotalPausedMs += now.Sub(*state.PausedAt).Milliseconds()
	}

	state.Status = models.RuntimeStatusRunning
	state.PausedAt = nil
	state.UpdatedBy = &userID

	if err := s.repo.Update(ctx, state); err != nil {
		return nil, err
	}
	resp := dto.ToMatchRuntimeResponse(state)
	s.publishEvent(ctx, matchID, string(models.RuntimeStatusRunning), s.computeElapsed(state).ElapsedMs)
	return resp, nil
}

// EndMatch transitions running|paused → ended.
func (s *matchRuntimeService) EndMatch(ctx context.Context, matchID uuid.UUID, authCtx *utils.AuthorizationContext) (*dto.MatchRuntimeResponse, error) {
	if err := authCtx.RequireAdmin(); err != nil {
		return nil, sharederrors.NewForbidden("Admin access required", err)
	}

	state, err := s.repo.FindByMatchID(ctx, matchID)
	if err != nil {
		return nil, err
	}
	if err := models.ValidateTransition(state.Status, models.RuntimeStatusEnded); err != nil {
		return nil, err
	}

	now := s.clock.Now()
	userID := authCtx.UserID

	// Finalise any in-flight pause interval.
	if state.Status == models.RuntimeStatusPaused && state.PausedAt != nil {
		state.TotalPausedMs += now.Sub(*state.PausedAt).Milliseconds()
	}

	state.Status = models.RuntimeStatusEnded
	state.EndedAt = &now
	state.PausedAt = nil
	state.UpdatedBy = &userID

	if err := s.repo.Update(ctx, state); err != nil {
		return nil, err
	}
	resp := dto.ToMatchRuntimeResponse(state)
	s.publishEvent(ctx, matchID, string(models.RuntimeStatusEnded), s.computeElapsed(state).ElapsedMs)
	return resp, nil
}

func (s *matchRuntimeService) GetState(ctx context.Context, matchID uuid.UUID) (*dto.MatchRuntimeResponse, error) {
	state, err := s.repo.FindByMatchID(ctx, matchID)
	if err != nil {
		return nil, err
	}
	return dto.ToMatchRuntimeResponse(state), nil
}

func (s *matchRuntimeService) CurrentMatchTime(ctx context.Context, matchID uuid.UUID) (*dto.MatchTimeResponse, error) {
	state, err := s.repo.FindByMatchID(ctx, matchID)
	if err != nil {
		return nil, err
	}
	return s.computeElapsed(state), nil
}

// computeElapsed derives the running match clock from persisted state.
// This makes the timer immune to server restarts.
func (s *matchRuntimeService) computeElapsed(state *models.MatchRuntimeState) *dto.MatchTimeResponse {
	now := s.clock.Now()

	resp := &dto.MatchTimeResponse{
		MatchID:       state.MatchID,
		Status:        string(state.Status),
		TotalPausedMs: state.TotalPausedMs,
		StartedAt:     state.StartedAt,
		ServerTimeMs:  now.UnixMilli(),
	}

	if state.StartedAt == nil {
		return resp
	}

	var elapsed int64
	switch state.Status {
	case models.RuntimeStatusRunning:
		elapsed = now.Sub(*state.StartedAt).Milliseconds() - state.TotalPausedMs

	case models.RuntimeStatusPaused:
		if state.PausedAt != nil {
			elapsed = state.PausedAt.Sub(*state.StartedAt).Milliseconds() - state.TotalPausedMs
		}

	case models.RuntimeStatusEnded:
		if state.EndedAt != nil {
			elapsed = state.EndedAt.Sub(*state.StartedAt).Milliseconds() - state.TotalPausedMs
		}
	}

	if elapsed < 0 {
		elapsed = 0
	}
	resp.ElapsedMs = elapsed
	return resp
}
