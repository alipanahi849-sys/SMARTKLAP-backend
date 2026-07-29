package service

import (
	"context"
	"time"

	"clap/internal/modules/playback/dto"
	"clap/internal/modules/playback/models"
	"clap/internal/modules/playback/repository"
	sharederrors "clap/internal/shared/errors"
	"clap/internal/shared/logger"
	"clap/internal/shared/utils"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// minScheduleAdvanceSeconds is the minimum lead time before a song can be scheduled.
const minScheduleAdvanceSeconds = 5

// SongExistsChecker allows PlaybackService to validate songs without importing
// the song module (avoids circular dependencies).
type SongExistsChecker interface {
	Exists(ctx context.Context, songID uuid.UUID) (bool, error)
}

// MatchExistsChecker allows PlaybackService to validate matches without importing
// the match module.
type MatchExistsChecker interface {
	Exists(ctx context.Context, matchID uuid.UUID) (bool, error)
}

// PlaybackService manages song playback event scheduling.
// It does NOT stream audio — it only persists scheduling records and
// registers events with the EventScheduler for future dispatch.
type PlaybackService interface {
	ScheduleSong(ctx context.Context, req *dto.ScheduleSongRequest, authCtx *utils.AuthorizationContext) (*dto.PlaybackScheduleResponse, error)
	CancelSong(ctx context.Context, scheduleID uuid.UUID, authCtx *utils.AuthorizationContext) error
	GetUpcomingSongs(ctx context.Context, matchID uuid.UUID) (*dto.UpcomingPlaybackResponse, error)
}

type playbackService struct {
	repo           repository.PlaybackRepository
	songChecker    SongExistsChecker
	matchChecker   MatchExistsChecker
	eventScheduler SongEventScheduler // optional; nil disables realtime event scheduling
	clock          func() time.Time
}

func NewPlaybackService(
	repo repository.PlaybackRepository,
	songChecker SongExistsChecker,
	matchChecker MatchExistsChecker,
) PlaybackService {
	return &playbackService{
		repo:         repo,
		songChecker:  songChecker,
		matchChecker: matchChecker,
		clock:        func() time.Time { return time.Now().UTC() },
	}
}

// NewPlaybackServiceWithEvents constructs the service with realtime event
// scheduling enabled. When a song is scheduled it also persists the
// song.playback.started and lyrics.line.changed events for dispatch (CR-5).
func NewPlaybackServiceWithEvents(
	repo repository.PlaybackRepository,
	songChecker SongExistsChecker,
	matchChecker MatchExistsChecker,
	eventScheduler SongEventScheduler,
) PlaybackService {
	return &playbackService{
		repo:           repo,
		songChecker:    songChecker,
		matchChecker:   matchChecker,
		eventScheduler: eventScheduler,
		clock:          func() time.Time { return time.Now().UTC() },
	}
}

func (s *playbackService) ScheduleSong(ctx context.Context, req *dto.ScheduleSongRequest, authCtx *utils.AuthorizationContext) (*dto.PlaybackScheduleResponse, error) {
	if err := authCtx.RequireAdminOrClubAdmin(); err != nil {
		return nil, sharederrors.NewForbidden("Admin or club admin access required", err)
	}

	// Validate schedule window.
	now := s.clock()
	if req.ScheduledAt.Before(now.Add(minScheduleAdvanceSeconds * time.Second)) {
		return nil, sharederrors.NewBadRequest("scheduled_at must be at least 5 seconds in the future", nil)
	}

	// Validate song existence.
	songExists, err := s.songChecker.Exists(ctx, req.SongID)
	if err != nil {
		return nil, err
	}
	if !songExists {
		return nil, sharederrors.NewNotFound("Song not found", nil)
	}

	// Validate match existence.
	matchExists, err := s.matchChecker.Exists(ctx, req.MatchID)
	if err != nil {
		return nil, err
	}
	if !matchExists {
		return nil, sharederrors.NewNotFound("Match not found", nil)
	}

	// Overlap detection: prevent conflicting playback windows for the same match.
	if req.DurationMs > 0 {
		overlaps, err := s.repo.HasOverlap(ctx, req.MatchID, req.ScheduledAt, req.DurationMs, nil)
		if err != nil {
			return nil, err
		}
		if overlaps {
			return nil, sharederrors.NewConflict(
				"Playback window overlaps with an existing schedule for this match", nil,
			)
		}
	}

	userID := authCtx.UserID
	schedule := &models.PlaybackSchedule{
		MatchID:     req.MatchID,
		SongID:      req.SongID,
		ScheduledAt: req.ScheduledAt,
		DurationMs:  req.DurationMs,
		Status:      models.PlaybackStatusPending,
		CreatedBy:   &userID,
	}

	if err := s.repo.Create(ctx, schedule); err != nil {
		return nil, err
	}

	// Schedule realtime events (playback.started + lyric lines) for dispatch.
	// Best-effort: a delivery-scheduling failure must not invalidate the
	// persisted schedule, but it is logged for observability.
	if s.eventScheduler != nil {
		if err := s.eventScheduler.ScheduleSongEvents(ctx, schedule.MatchID, schedule.SongID, schedule.ID, schedule.ScheduledAt); err != nil {
			logger.Error().
				Str("match_id", schedule.MatchID.String()).
				Str("song_id", schedule.SongID.String()).
				Str("schedule_id", schedule.ID.String()).
				Err(err).
				Msg("failed to schedule realtime song events")
		}
	}

	return dto.ToPlaybackScheduleResponse(schedule), nil
}

func (s *playbackService) CancelSong(ctx context.Context, scheduleID uuid.UUID, authCtx *utils.AuthorizationContext) error {
	if err := authCtx.RequireAdminOrClubAdmin(); err != nil {
		return sharederrors.NewForbidden("Admin or club admin access required", err)
	}

	existing, err := s.repo.FindByID(ctx, scheduleID)
	if err != nil {
		return err
	}

	if existing.Status == models.PlaybackStatusCancelled {
		return sharederrors.NewConflict("Playback schedule is already cancelled", nil)
	}
	if existing.Status == models.PlaybackStatusCompleted {
		return sharederrors.NewBadRequest("Cannot cancel a completed playback", nil)
	}

	existing.Status = models.PlaybackStatusCancelled
	return s.repo.Update(ctx, existing)
}

func (s *playbackService) GetUpcomingSongs(ctx context.Context, matchID uuid.UUID) (*dto.UpcomingPlaybackResponse, error) {
	now := s.clock()
	schedules, err := s.repo.FindUpcoming(ctx, matchID, now)
	if err != nil {
		return nil, err
	}

	responses := make([]*dto.PlaybackScheduleResponse, len(schedules))
	for i, sch := range schedules {
		responses[i] = dto.ToPlaybackScheduleResponse(sch)
	}

	return &dto.UpcomingPlaybackResponse{
		Schedules: responses,
		Total:     len(responses),
	}, nil
}

// ─── Checker adapters ─────────────────────────────────────────────────────────

// GormSongChecker validates song existence via a raw GORM query.
type GormSongChecker struct {
	DB *gorm.DB
}

func (c *GormSongChecker) Exists(ctx context.Context, songID uuid.UUID) (bool, error) {
	var count int64
	err := c.DB.WithContext(ctx).
		Table("songs").
		Where("id = ? AND deleted_at IS NULL", songID).
		Count(&count).Error
	if err != nil {
		return false, sharederrors.NewInternal("Failed to check song existence", err)
	}
	return count > 0, nil
}

// GormMatchChecker validates match existence via a raw GORM query.
type GormMatchChecker struct {
	DB *gorm.DB
}

func (c *GormMatchChecker) Exists(ctx context.Context, matchID uuid.UUID) (bool, error) {
	var count int64
	err := c.DB.WithContext(ctx).
		Table("matches").
		Where("id = ? AND deleted_at IS NULL", matchID).
		Count(&count).Error
	if err != nil {
		return false, sharederrors.NewInternal("Failed to check match existence", err)
	}
	return count > 0, nil
}
