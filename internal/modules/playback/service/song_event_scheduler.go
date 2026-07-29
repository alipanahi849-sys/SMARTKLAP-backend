package service

import (
	"context"
	"time"

	schedulersvc "clap/internal/modules/eventscheduler/service"
	lyricssvc "clap/internal/modules/lyricssync/service"
	realtimedto "clap/internal/modules/realtime/dto"
	realtimemodels "clap/internal/modules/realtime/models"
	realtimerepo "clap/internal/modules/realtime/repository"
	"clap/internal/shared/logger"

	"github.com/google/uuid"
)

// SongEventScheduler turns a scheduled song into durable, dispatchable realtime
// events: a song.playback.started event at the song's start time, followed by
// one lyrics.line.changed event per lyric line (CR-5).
//
// Events are persisted via the EventScheduler (DB-backed) so they survive
// restarts and are dispatched by the EventDispatcher → realtime gateway →
// WebSocket clients.
type SongEventScheduler interface {
	ScheduleSongEvents(ctx context.Context, matchID, songID, scheduleID uuid.UUID, scheduledAt time.Time) error
}

// lyricEventPayload is serialised into SchedulerEvent.PayloadJSON and mirrors
// the realtime EventDispatcher's expected payload shape.
type lyricEventPayload struct {
	MatchID     string `json:"match_id"`
	SongID      string `json:"song_id,omitempty"`
	ScheduleID  string `json:"schedule_id,omitempty"`
	EventType   string `json:"event_type"`
	Line        string `json:"line,omitempty"`
	Index       int    `json:"index,omitempty"`
	TimestampMs int64  `json:"timestamp_ms,omitempty"`
}

type songEventScheduler struct {
	sessionRepo realtimerepo.RealtimeSessionRepository
	lyricsSvc   lyricssvc.LyricsSyncService
	scheduler   schedulersvc.EventSchedulerService
}

// NewSongEventScheduler constructs the scheduler collaborator.
func NewSongEventScheduler(
	sessionRepo realtimerepo.RealtimeSessionRepository,
	lyricsSvc lyricssvc.LyricsSyncService,
	scheduler schedulersvc.EventSchedulerService,
) SongEventScheduler {
	return &songEventScheduler{
		sessionRepo: sessionRepo,
		lyricsSvc:   lyricsSvc,
		scheduler:   scheduler,
	}
}

func (s *songEventScheduler) ScheduleSongEvents(
	ctx context.Context,
	matchID, songID, scheduleID uuid.UUID,
	scheduledAt time.Time,
) error {
	session, err := s.getOrCreateSession(ctx, matchID)
	if err != nil {
		return err
	}

	// 1. song.playback.started at the song's start time.
	startedPayload := lyricEventPayload{
		MatchID:    matchID.String(),
		SongID:     songID.String(),
		ScheduleID: scheduleID.String(),
		EventType:  realtimedto.EventTypeSongPlaybackStarted,
	}
	if _, err := s.scheduler.RegisterEvent(ctx, &schedulersvc.RegisterEventRequest{
		SessionID: session.ID,
		EventType: realtimedto.EventTypeSongPlaybackStarted,
		ExecuteAt: scheduledAt,
		Payload:   startedPayload,
	}); err != nil {
		return err
	}

	// 2. lyrics.line.changed for each line, anchored to the song start time.
	language, err := s.lyricsSvc.FirstAvailableLanguage(ctx, songID)
	if err != nil {
		// No lyrics for this song — playback.started alone is a valid outcome.
		logger.Info().
			Str("match_id", matchID.String()).
			Str("song_id", songID.String()).
			Msg("song event scheduler: no lyrics found; scheduled playback.started only")
		return nil
	}

	timeline, err := s.lyricsSvc.BuildLyricsTimeline(ctx, songID, language)
	if err != nil {
		return err
	}

	scheduled := 0
	for _, entry := range timeline.Entries {
		executeAt := scheduledAt.Add(time.Duration(entry.TimestampMs) * time.Millisecond)
		payload := lyricEventPayload{
			MatchID:     matchID.String(),
			SongID:      songID.String(),
			ScheduleID:  scheduleID.String(),
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
		Str("match_id", matchID.String()).
		Str("song_id", songID.String()).
		Str("session_id", session.ID.String()).
		Str("language", language).
		Int("lyric_events", scheduled).
		Msg("song event scheduler: scheduled playback + lyric events")

	return nil
}

// getOrCreateSession returns the active realtime session for the match,
// creating one if none exists. The session id anchors all scheduled events and
// satisfies the scheduler_events.session_id foreign key (migration 026).
func (s *songEventScheduler) getOrCreateSession(ctx context.Context, matchID uuid.UUID) (*realtimemodels.RealtimeSession, error) {
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
