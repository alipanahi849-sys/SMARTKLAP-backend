package service

import (
	"context"
	"time"

	"clap/internal/modules/eventscheduler/repository"
	"clap/internal/shared/logger"
)

// DefaultStaleProcessingTimeout is the default age after which a processing
// event is considered orphaned (crashed dispatcher) and reset to pending.
const DefaultStaleProcessingTimeout = 5 * time.Minute

// DefaultWatchdogInterval is the default cadence for the periodic recovery sweep.
const DefaultWatchdogInterval = 5 * time.Minute

// StaleRecoveryObserver is notified of stale-event resets for metrics.
// Implementations must be safe for concurrent use.
type StaleRecoveryObserver interface {
	ObserveStaleReset(count int64)
}

// SchedulerRecoveryService hydrates the in-memory priority queue from the
// durable scheduler_events table and reclaims events orphaned in the
// processing state by a crashed dispatcher.
//
// Recovery flow (CR-2):
//  1. Reset stale processing events (processing + updated_at older than timeout) → pending
//  2. Load all pending events
//  3. Rehydrate the in-memory scheduler
//
// The service is idempotent: events already present in the queue are skipped.
type SchedulerRecoveryService interface {
	// RecoverPendingEvents performs one full recovery sweep and returns the
	// number of events (re)registered in the in-memory scheduler.
	RecoverPendingEvents(ctx context.Context) (int, error)
	// RunWatchdog runs RecoverPendingEvents on the given interval until ctx is
	// cancelled. interval <= 0 uses DefaultWatchdogInterval.
	RunWatchdog(ctx context.Context, interval time.Duration)
}

type schedulerRecoveryService struct {
	repo         repository.SchedulerEventRepository
	scheduler    EventScheduler
	staleTimeout time.Duration
	observer     StaleRecoveryObserver
}

// NewSchedulerRecoveryService constructs the recovery service with default
// stale-processing timeout.
func NewSchedulerRecoveryService(
	repo repository.SchedulerEventRepository,
	scheduler EventScheduler,
) SchedulerRecoveryService {
	return NewSchedulerRecoveryServiceWithConfig(repo, scheduler, DefaultStaleProcessingTimeout, nil)
}

// NewSchedulerRecoveryServiceWithConfig constructs the recovery service with a
// custom stale-processing timeout and an optional metrics observer.
func NewSchedulerRecoveryServiceWithConfig(
	repo repository.SchedulerEventRepository,
	scheduler EventScheduler,
	staleTimeout time.Duration,
	observer StaleRecoveryObserver,
) SchedulerRecoveryService {
	if staleTimeout <= 0 {
		staleTimeout = DefaultStaleProcessingTimeout
	}
	return &schedulerRecoveryService{
		repo:         repo,
		scheduler:    scheduler,
		staleTimeout: staleTimeout,
		observer:     observer,
	}
}

func (s *schedulerRecoveryService) RecoverPendingEvents(ctx context.Context) (int, error) {
	// Step 1: reclaim events orphaned in processing by a crashed dispatcher.
	cutoff := time.Now().UTC().Add(-s.staleTimeout)
	reset, err := s.repo.ResetStaleProcessing(ctx, cutoff)
	if err != nil {
		// Non-fatal: continue to load whatever pending events exist.
		logger.Error().Err(err).Msg("scheduler recovery: stale processing reset failed")
	} else if reset > 0 {
		if s.observer != nil {
			s.observer.ObserveStaleReset(reset)
		}
		logger.Warn().
			Int64("reset_to_pending", reset).
			Time("cutoff", cutoff).
			Msg("scheduler recovery: stale processing events reclaimed")
	}

	// Step 2: load all pending events (now including any just-reclaimed ones).
	events, err := s.repo.FindAllPending(ctx)
	if err != nil {
		return 0, err
	}

	// Step 3: rehydrate the in-memory scheduler (idempotent).
	recovered := 0
	skipped := 0
	for _, ev := range events {
		item := &SchedulerItem{
			ID:          ev.ID.String(),
			SessionID:   ev.SessionID.String(),
			EventType:   ev.EventType,
			ExecuteAt:   ev.ExecuteAt,
			PayloadJSON: ev.PayloadJSON,
		}
		if err := s.scheduler.RegisterEvent(ctx, item); err != nil {
			skipped++
			continue
		}
		recovered++
	}

	logger.Info().
		Int("recovered", recovered).
		Int("skipped_already_registered", skipped).
		Int("total_pending", len(events)).
		Int64("stale_reset", reset).
		Msg("Scheduler recovery complete")

	return recovered, nil
}

func (s *schedulerRecoveryService) RunWatchdog(ctx context.Context, interval time.Duration) {
	if interval <= 0 {
		interval = DefaultWatchdogInterval
	}
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	logger.Info().
		Dur("interval", interval).
		Dur("stale_timeout", s.staleTimeout).
		Msg("scheduler recovery watchdog started")

	for {
		select {
		case <-ctx.Done():
			logger.Info().Msg("scheduler recovery watchdog stopped")
			return
		case <-ticker.C:
			if _, err := s.RecoverPendingEvents(ctx); err != nil {
				logger.Error().Err(err).Msg("scheduler recovery watchdog sweep failed")
			}
		}
	}
}
