package service

import (
	"context"
	"time"

	schedulerrepo "clap/internal/modules/eventscheduler/repository"
	realtimerepo "clap/internal/modules/realtime/repository"
	"clap/internal/shared/logger"
)

// RetentionConfig controls how long terminal realtime data is retained before
// the cleanup job deletes it. Zero values fall back to defaults.
type RetentionConfig struct {
	SchedulerEventRetentionDays int
	RealtimeEventRetentionDays  int
}

// Defaults applied when a retention value is non-positive.
const (
	DefaultSchedulerEventRetentionDays = 7
	DefaultRealtimeEventRetentionDays  = 7
)

// RetentionResult reports how many rows each cleanup pass removed.
type RetentionResult struct {
	SchedulerEventsDeleted int64 `json:"scheduler_events_deleted"`
	RealtimeEventsDeleted  int64 `json:"realtime_events_deleted"`
}

// DataRetentionService deletes terminal scheduler_events and aged
// realtime_events beyond their configured retention windows (CR-14).
// No cron is used — cleanup is triggered via an admin endpoint or an external
// scheduler.
type DataRetentionService interface {
	// CleanupSchedulerEvents deletes executed/failed/cancelled scheduler_events
	// older than the configured retention. Returns rows deleted.
	CleanupSchedulerEvents(ctx context.Context) (int64, error)
	// CleanupRealtimeEvents deletes realtime_events older than the configured
	// retention. Returns rows deleted.
	CleanupRealtimeEvents(ctx context.Context) (int64, error)
	// CleanupAll runs both passes and returns a combined result.
	CleanupAll(ctx context.Context) (*RetentionResult, error)
}

type dataRetentionService struct {
	schedulerRepo schedulerrepo.SchedulerEventRepository
	realtimeRepo  realtimerepo.RealtimeEventRepository
	cfg           RetentionConfig
}

// NewDataRetentionService constructs the retention service, applying defaults
// to any non-positive config value.
func NewDataRetentionService(
	schedulerRepo schedulerrepo.SchedulerEventRepository,
	realtimeRepo realtimerepo.RealtimeEventRepository,
	cfg RetentionConfig,
) DataRetentionService {
	if cfg.SchedulerEventRetentionDays <= 0 {
		cfg.SchedulerEventRetentionDays = DefaultSchedulerEventRetentionDays
	}
	if cfg.RealtimeEventRetentionDays <= 0 {
		cfg.RealtimeEventRetentionDays = DefaultRealtimeEventRetentionDays
	}
	return &dataRetentionService{
		schedulerRepo: schedulerRepo,
		realtimeRepo:  realtimeRepo,
		cfg:           cfg,
	}
}

func (s *dataRetentionService) CleanupSchedulerEvents(ctx context.Context) (int64, error) {
	cutoff := time.Now().UTC().AddDate(0, 0, -s.cfg.SchedulerEventRetentionDays)
	deleted, err := s.schedulerRepo.DeleteTerminalOlderThan(ctx, cutoff)
	if err != nil {
		return 0, err
	}
	logger.Info().
		Int64("deleted", deleted).
		Int("retention_days", s.cfg.SchedulerEventRetentionDays).
		Time("cutoff", cutoff).
		Msg("scheduler_events retention cleanup completed")
	return deleted, nil
}

func (s *dataRetentionService) CleanupRealtimeEvents(ctx context.Context) (int64, error) {
	cutoff := time.Now().UTC().AddDate(0, 0, -s.cfg.RealtimeEventRetentionDays)
	deleted, err := s.realtimeRepo.DeleteOlderThan(ctx, cutoff)
	if err != nil {
		return 0, err
	}
	logger.Info().
		Int64("deleted", deleted).
		Int("retention_days", s.cfg.RealtimeEventRetentionDays).
		Time("cutoff", cutoff).
		Msg("realtime_events retention cleanup completed")
	return deleted, nil
}

func (s *dataRetentionService) CleanupAll(ctx context.Context) (*RetentionResult, error) {
	schedDeleted, err := s.CleanupSchedulerEvents(ctx)
	if err != nil {
		return nil, err
	}
	rtDeleted, err := s.CleanupRealtimeEvents(ctx)
	if err != nil {
		return nil, err
	}
	return &RetentionResult{
		SchedulerEventsDeleted: schedDeleted,
		RealtimeEventsDeleted:  rtDeleted,
	}, nil
}
