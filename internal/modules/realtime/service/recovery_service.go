package service

import (
	"context"
	"time"

	lyricsdto "clap/internal/modules/lyricssync/dto"
	lyricssvc "clap/internal/modules/lyricssync/service"
	matchruntimerepo "clap/internal/modules/matchruntime/repository"
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
	RuntimeState *RuntimeStateSnapshot               `json:"runtime_state,omitempty"`
	ActiveSong   *ActiveSongSnapshot                 `json:"active_song,omitempty"`
	CurrentLyric *lyricsdto.LyricAtTimestampResponse `json:"current_lyric,omitempty"`
}

// RuntimeStateSnapshot is a lightweight read of the current match runtime.
type RuntimeStateSnapshot struct {
	MatchID   string `json:"match_id"`
	Status    string `json:"status"`
	ElapsedMs int64  `json:"elapsed_ms"`
}

// ActiveSongSnapshot describes the song currently scheduled or playing.
type ActiveSongSnapshot struct {
	ScheduleID  string    `json:"schedule_id"`
	SongID      string    `json:"song_id"`
	ScheduledAt time.Time `json:"scheduled_at"`
	DurationMs  int64     `json:"duration_ms"`
}

// ReconnectionRecoveryService aggregates cross-module state for the reconnection
// endpoint.  It reads from repositories directly, bypassing service-layer auth,
// to stay lightweight and fast.
type ReconnectionRecoveryService interface {
	GetMatchState(ctx context.Context, matchID uuid.UUID) (*ReconnectionState, error)
}

type reconnectionRecoveryService struct {
	runtimeRepo  matchruntimerepo.MatchRuntimeRepository
	playbackRepo playbackrepo.PlaybackRepository
	lyricsSvc    lyricssvc.LyricsSyncService
}

// NewReconnectionRecoveryService constructs the service.
// lyricsSvc is optional — pass nil to skip lyrics recovery.
func NewReconnectionRecoveryService(
	runtimeRepo matchruntimerepo.MatchRuntimeRepository,
	playbackRepo playbackrepo.PlaybackRepository,
	lyricsSvc lyricssvc.LyricsSyncService,
) ReconnectionRecoveryService {
	return &reconnectionRecoveryService{
		runtimeRepo:  runtimeRepo,
		playbackRepo: playbackRepo,
		lyricsSvc:    lyricsSvc,
	}
}

func (s *reconnectionRecoveryService) GetMatchState(ctx context.Context, matchID uuid.UUID) (*ReconnectionState, error) {
	state := &ReconnectionState{
		ServerTimeMs: time.Now().UnixMilli(),
	}

	// 1. Runtime state.
	runtime, err := s.runtimeRepo.FindByMatchID(ctx, matchID)
	if err != nil {
		if isNotFound(err) {
			// Match exists but has no runtime — return partial state.
			return state, nil
		}
		return nil, err
	}

	var elapsedMs int64
	if runtime.StartedAt != nil {
		switch string(runtime.Status) {
		case "running":
			elapsedMs = time.Now().UnixMilli() - runtime.StartedAt.UnixMilli() - runtime.TotalPausedMs
		case "paused":
			if runtime.PausedAt != nil {
				elapsedMs = runtime.PausedAt.UnixMilli() - runtime.StartedAt.UnixMilli() - runtime.TotalPausedMs
			}
		case "ended":
			if runtime.EndedAt != nil {
				elapsedMs = runtime.EndedAt.UnixMilli() - runtime.StartedAt.UnixMilli() - runtime.TotalPausedMs
			}
		}
		if elapsedMs < 0 {
			elapsedMs = 0
		}
	}

	state.RuntimeState = &RuntimeStateSnapshot{
		MatchID:   matchID.String(),
		Status:    string(runtime.Status),
		ElapsedMs: elapsedMs,
	}

	// 2. Active song (currently playing or the next pending one).
	upcoming, err := s.playbackRepo.FindUpcoming(ctx, matchID, time.Now().Add(-time.Hour))
	if err == nil && len(upcoming) > 0 {
		// Take the most recent schedule that started in the past or is the next one.
		now := time.Now()
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
		if active == nil && len(upcoming) > 0 {
			// No song started yet — use the next upcoming one.
			sch := upcoming[0]
			active = &ActiveSongSnapshot{
				ScheduleID:  sch.ID.String(),
				SongID:      sch.SongID.String(),
				ScheduledAt: sch.ScheduledAt,
				DurationMs:  sch.DurationMs,
			}
		}
		state.ActiveSong = active
	}

	// 3. Current lyric (optional, best-effort). Language is resolved from stored
	//    lyrics rather than hardcoded (F-021). The lyric offset is measured from
	//    the song's start, not from match elapsed time.
	if s.lyricsSvc != nil && state.ActiveSong != nil {
		songID, parseErr := uuid.Parse(state.ActiveSong.SongID)
		if parseErr == nil {
			if language, langErr := s.lyricsSvc.FirstAvailableLanguage(ctx, songID); langErr == nil {
				songOffsetMs := time.Now().UnixMilli() - state.ActiveSong.ScheduledAt.UnixMilli()
				if songOffsetMs >= 0 {
					if lyricResp, lyricErr := s.lyricsSvc.GetLyricsAtTimestamp(ctx, songID, language, songOffsetMs); lyricErr == nil {
						state.CurrentLyric = lyricResp
					}
				}
			}
		}
	}

	return state, nil
}

// ─── EventEnvelopePublisher (used by lyric scheduler) ────────────────────────

// EnvelopePublisher delivers a pre-built EventEnvelope to a match channel.
// Implemented by WebSocketRealtimeGateway.
type EnvelopePublisher interface {
	PublishToMatch(ctx context.Context, matchID uuid.UUID, env *realtimeddto.EventEnvelope) error
}

// isNotFound checks whether err is a 404-style AppError.
func isNotFound(err error) bool {
	type coder interface{ StatusCode() int }
	if c, ok := err.(coder); ok {
		return c.StatusCode() == 404
	}
	// Also check shared errors package's pattern.
	if appErr, ok := err.(*sharederrors.AppError); ok {
		return appErr.StatusCode == 404
	}
	return false
}
