package service

import (
	"context"
	"time"

	chantmodels "clap/internal/modules/chant/models"
	schedulersvc "clap/internal/modules/eventscheduler/service"
	lyricssvc "clap/internal/modules/lyricssync/service"
	realtimedto "clap/internal/modules/realtime/dto"
	realtimemodels "clap/internal/modules/realtime/models"
	realtimerepo "clap/internal/modules/realtime/repository"
	"clap/internal/shared/logger"

	"github.com/google/uuid"
)

// ChantEventScheduler registers durable realtime events for a chant:
// chant.started at the scheduled time, then one lyrics.line.changed per line.
type ChantEventScheduler interface {
	ScheduleChantEvents(ctx context.Context, chant chantmodels.Chant) error
}

type chantEventPayload struct {
	MatchID     string `json:"match_id"`
	ChantID     string `json:"chant_id,omitempty"`
	SongID      string `json:"song_id,omitempty"`
	EventType   string `json:"event_type"`
	Line        string `json:"line,omitempty"`
	Index       int    `json:"index,omitempty"`
	TimestampMs int64  `json:"timestamp_ms,omitempty"`
	StartsAt    string `json:"starts_at,omitempty"`
}

type chantEventScheduler struct {
	sessionRepo realtimerepo.RealtimeSessionRepository
	lyricsSvc   lyricssvc.LyricsSyncService
	scheduler   schedulersvc.EventSchedulerService
}

// NewChantEventScheduler constructs the scheduler.
func NewChantEventScheduler(
	sessionRepo realtimerepo.RealtimeSessionRepository,
	lyricsSvc lyricssvc.LyricsSyncService,
	scheduler schedulersvc.EventSchedulerService,
) ChantEventScheduler {
	return &chantEventScheduler{
		sessionRepo: sessionRepo,
		lyricsSvc:   lyricsSvc,
		scheduler:   scheduler,
	}
}

func (s *chantEventScheduler) ScheduleChantEvents(ctx context.Context, chant chantmodels.Chant) error {
	session, err := s.getOrCreateSession(ctx, chant.MatchID)
	if err != nil {
		return err
	}

	startedPayload := chantEventPayload{
		MatchID:   chant.MatchID.String(),
		ChantID:   chant.ID.String(),
		SongID:    chant.SongID.String(),
		EventType: realtimedto.EventTypeChantStarted,
		StartsAt:  chant.ScheduledAt.UTC().Format(time.RFC3339Nano),
	}
	if _, err := s.scheduler.RegisterEvent(ctx, &schedulersvc.RegisterEventRequest{
		SessionID: session.ID,
		EventType: realtimedto.EventTypeChantStarted,
		ExecuteAt: chant.ScheduledAt,
		Payload:   startedPayload,
	}); err != nil {
		return err
	}

	language, err := s.lyricsSvc.FirstAvailableLanguage(ctx, chant.SongID)
	if err != nil {
		logger.Info().
			Str("chant_id", chant.ID.String()).
			Str("song_id", chant.SongID.String()).
			Msg("chant event scheduler: no lyrics found; scheduled chant.started only")
		return nil
	}

	timeline, err := s.lyricsSvc.BuildLyricsTimeline(ctx, chant.SongID, language)
	if err != nil {
		return err
	}

	scheduled := 0
	for _, entry := range timeline.Entries {
		executeAt := chant.ScheduledAt.Add(time.Duration(entry.TimestampMs) * time.Millisecond)
		payload := chantEventPayload{
			MatchID:     chant.MatchID.String(),
			ChantID:     chant.ID.String(),
			SongID:      chant.SongID.String(),
			EventType:   realtimedto.EventTypeLyricsLineChanged,
			Line:        entry.Text,
			Index:       entry.Index,
			TimestampMs: entry.TimestampMs,
		}
		if _, err := s.scheduler.RegisterEvent(ctx, &schedulersvc.RegisterEventRequest{
			SessionID: session.ID,
			EventType: realtimedto.EventTypeLyricsLineChanged,
			ExecuteAt: executeAt,
			Payload:   payload,
		}); err != nil {
			return err
		}
		scheduled++
	}

	logger.Info().
		Str("chant_id", chant.ID.String()).
		Str("match_id", chant.MatchID.String()).
		Str("session_id", session.ID.String()).
		Int("lyric_events", scheduled).
		Time("starts_at", chant.ScheduledAt).
		Msg("chant event scheduler: scheduled chant.started + lyric events")

	return nil
}

func (s *chantEventScheduler) getOrCreateSession(ctx context.Context, matchID uuid.UUID) (*realtimemodels.RealtimeSession, error) {
	session, err := s.sessionRepo.FindActiveByMatchID(ctx, matchID)
	if err == nil {
		return session, nil
	}

	now := time.Now().UTC()
	fresh := &realtimemodels.RealtimeSession{
		MatchID:   matchID,
		Status:    realtimemodels.SessionStatusRunning,
		StartedAt: &now,
	}
	if createErr := s.sessionRepo.Create(ctx, fresh); createErr != nil {
		return nil, createErr
	}
	return fresh, nil
}
