package service

import (
	"context"
	"time"

	"clap/internal/modules/realtime/repository"
	"clap/internal/shared/logger"
)

// HeartbeatCleanupConfig holds the single tunable: how many days of heartbeat
// history to retain. Rows older than this are eligible for deletion.
type HeartbeatCleanupConfig struct {
	RetentionDays int
}

// DefaultHeartbeatRetentionDays is used when no explicit config is provided.
const DefaultHeartbeatRetentionDays = 30

// HeartbeatCleanupService deletes stale client_heartbeat rows beyond the
// configured retention window. No cron is used — cleanup is triggered
// manually via an admin HTTP endpoint or by an external scheduler.
type HeartbeatCleanupService interface {
	// Cleanup deletes heartbeats older than RetentionDays.
	// Returns the number of rows deleted.
	Cleanup(ctx context.Context) (int64, error)
}

type heartbeatCleanupService struct {
	repo   repository.ClientHeartbeatRepository
	config HeartbeatCleanupConfig
}

// NewHeartbeatCleanupService constructs the cleanup service with default retention.
func NewHeartbeatCleanupService(repo repository.ClientHeartbeatRepository) HeartbeatCleanupService {
	return NewHeartbeatCleanupServiceWithConfig(repo, HeartbeatCleanupConfig{
		RetentionDays: DefaultHeartbeatRetentionDays,
	})
}

// NewHeartbeatCleanupServiceWithConfig constructs the cleanup service with a custom config.
func NewHeartbeatCleanupServiceWithConfig(repo repository.ClientHeartbeatRepository, cfg HeartbeatCleanupConfig) HeartbeatCleanupService {
	if cfg.RetentionDays <= 0 {
		cfg.RetentionDays = DefaultHeartbeatRetentionDays
	}
	return &heartbeatCleanupService{repo: repo, config: cfg}
}

func (s *heartbeatCleanupService) Cleanup(ctx context.Context) (int64, error) {
	cutoff := time.Now().UTC().AddDate(0, 0, -s.config.RetentionDays)

	deleted, err := s.repo.DeleteOlderThan(ctx, cutoff)
	if err != nil {
		return 0, err
	}

	logger.Info().
		Int64("deleted", deleted).
		Int("retention_days", s.config.RetentionDays).
		Time("cutoff", cutoff).
		Msg("Heartbeat cleanup completed")

	return deleted, nil
}
