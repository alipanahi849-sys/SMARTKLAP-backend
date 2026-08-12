package service

import (
	"context"
	"time"

	chantrepo "clap/internal/modules/chant/repository"
	lyricsdto "clap/internal/modules/lyricssync/dto"
	lyricssvc "clap/internal/modules/lyricssync/service"
	playbackrepo "clap/internal/modules/playback/repository"
	realtimeddto "clap/internal/modules/realtime/dto"
	sharederrors "clap/internal/shared/errors"

	"github.com/google/uuid"
)

// ReconnectionState is the complete snapshot returned to a reconnecting client.
// It allows the client to immediately render the correct UI state without
// waiting for the next event.
type ReconnectionState struct {
	ServerTimeMs int64                               `json:"server_time_ms"`
	ActiveSong   *ActiveSongSnapshot                 `json:"active_song,omitempty"`
	ActiveChant  *ActiveChantSnapshot                `json:"active_chant,omitempty"`
	CurrentLyric *lyricsdto.LyricAtTimestampResponse `json:"current_lyric,omitempty"`
}

// ActiveSongSnapshot describes the song currently scheduled or playing.
type ActiveSongSnapshot struct {
	ScheduleID  string    `json:"schedule_id"`
	SongID      string    `json:"song_id"`
	ScheduledAt time.Time `json:"scheduled_at"`
	DurationMs  int64     `json:"duration_ms"`
}

// ActiveChantSnapshot describes a chant currently in progress for a match.
type ActiveChantSnapshot struct {
	ChantID    string    `json:"chant_id"`
	SongID     string    `json:"song_id"`
	Title      string    `json:"title"`
	StartsAt   time.Time `json:"starts_at"`
	DurationMs int64     `json:"duration_ms"`
	OffsetMs   int64     `json:"offset_ms"`
}

// ReconnectionRecoveryService aggregates cross-module state for the reconnection
// endpoint.  It reads from repositories directly, bypassing service-layer auth,
// to stay lightweight and fast.
type ReconnectionRecoveryService interface {
	GetMatchState(ctx context.Context, matchID uuid.UUID) (*ReconnectionState, error)
}

type reconnectionRecoveryService struct {
	playbackRepo playbackrepo.PlaybackRepository
	chantRepo    chantrepo.ChantRepository
	lyricsSvc    lyricssvc.LyricsSyncService
}

// NewReconnectionRecoveryService constructs the service.
// lyricsSvc is optional — pass nil to skip lyrics recovery.
func NewReconnectionRecoveryService(
	playbackRepo playbackrepo.PlaybackRepository,
	chantRepo chantrepo.ChantRepository,
	lyricsSvc lyricssvc.LyricsSyncService,
) ReconnectionRecoveryService {
	return &reconnectionRecoveryService{
		playbackRepo: playbackRepo,
		chantRepo:    chantRepo,
		lyricsSvc:    lyricsSvc,
	}
}

func (s *reconnectionRecoveryService) GetMatchState(ctx context.Context, matchID uuid.UUID) (*ReconnectionState, error) {
	now := time.Now().UTC()
	state := &ReconnectionState{
		ServerTimeMs: now.UnixMilli(),
	}

	// Active chant takes priority — it drives the mobile countdown/live flow.
	if s.chantRepo != nil {
		if chant, err := s.chantRepo.FindActiveByMatch(ctx, matchID, now); err == nil && chant != nil {
			offsetMs := now.Sub(chant.ScheduledAt).Milliseconds()
			if offsetMs < 0 {
				offsetMs = 0
			}
			state.ActiveChant = &ActiveChantSnapshot{
				ChantID:    chant.ID.String(),
				SongID:     chant.SongID.String(),
				Title:      chant.Title,
				StartsAt:   chant.ScheduledAt,
				DurationMs: int64(chant.DurationSeconds) * 1000,
				OffsetMs:   offsetMs,
			}
			s.populateCurrentLyric(ctx, chant.SongID, offsetMs, state)
			return state, nil
		}
	}

	// Fallback: active scheduled song playback.
	upcoming, err := s.playbackRepo.FindUpcoming(ctx, matchID, now.Add(-time.Hour))
	if err == nil && len(upcoming) > 0 {
		var active *ActiveSongSnapshot
		for _, sch := range upcoming {
			if sch.ScheduledAt.Before(now) || sch.ScheduledAt.Equal(now) {
				active = &ActiveSongSnapshot{
					ScheduleID:  sch.ID.String(),
					SongID:      sch.SongID.String(),
					ScheduledAt: sch.ScheduledAt,
					DurationMs:  sch.DurationMs,
				}
			}
		}
		if active == nil {
			sch := upcoming[0]
			active = &ActiveSongSnapshot{
				ScheduleID:  sch.ID.String(),
				SongID:      sch.SongID.String(),
				ScheduledAt: sch.ScheduledAt,
				DurationMs:  sch.DurationMs,
			}
		}
		state.ActiveSong = active
		if active != nil {
			songOffsetMs := now.UnixMilli() - active.ScheduledAt.UnixMilli()
			if songOffsetMs >= 0 {
				if songID, parseErr := uuid.Parse(active.SongID); parseErr == nil {
					s.populateCurrentLyric(ctx, songID, songOffsetMs, state)
				}
			}
		}
	}

	return state, nil
}

func (s *reconnectionRecoveryService) populateCurrentLyric(
	ctx context.Context,
	songID uuid.UUID,
	offsetMs int64,
	state *ReconnectionState,
) {
	if s.lyricsSvc == nil || offsetMs < 0 {
		return
	}
	language, langErr := s.lyricsSvc.FirstAvailableLanguage(ctx, songID)
	if langErr != nil {
		return
	}
	if lyricResp, lyricErr := s.lyricsSvc.GetLyricsAtTimestamp(ctx, songID, language, offsetMs); lyricErr == nil {
		state.CurrentLyric = lyricResp
	}
}

// ─── EventEnvelopePublisher (used by lyric scheduler) ────────────────────────

// EnvelopePublisher delivers a pre-built EventEnvelope to a match channel.
// Implemented by WebSocketRealtimeGateway.
type EnvelopePublisher interface {
	PublishToMatch(ctx context.Context, matchID uuid.UUID, env *realtimeddto.EventEnvelope) error
}

func isNotFound(err error) bool {
	type coder interface{ StatusCode() int }
	if c, ok := err.(coder); ok {
		return c.StatusCode() == 404
	}
	if appErr, ok := err.(*sharederrors.AppError); ok {
		return appErr.StatusCode == 404
	}
	return false
}
