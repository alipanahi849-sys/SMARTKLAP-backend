package service

import (
	"context"
	"encoding/json"
	"fmt"
	"sort"

	"clap/internal/modules/lyricssync/dto"
	realtimedto "clap/internal/modules/realtime/dto"
	realtimemodels "clap/internal/modules/realtime/models"
	sharederrors "clap/internal/shared/errors"
	"clap/pkg/lyrics"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// LyricsSyncService derives realtime sync events from stored lyric records.
// It consumes the existing SongLyric table via a GORM query to avoid
// importing the songlyric module (preventing circular dependencies).
type LyricsSyncService interface {
	// BuildLyricsTimeline parses the LRC content for a song/language pair and
	// returns an ordered, indexed timeline.
	BuildLyricsTimeline(ctx context.Context, songID uuid.UUID, language string) (*dto.LyricsTimeline, error)

	// GetLyricsAtTimestamp returns the current and next lyric line at the given
	// playback offset (milliseconds from song start).
	GetLyricsAtTimestamp(ctx context.Context, songID uuid.UUID, language string, offsetMs int64) (*dto.LyricAtTimestampResponse, error)

	// GenerateRealtimeEvents converts a lyric timeline into RealtimeEvent records
	// ready for batch-insert. ExecuteAtMs is anchored to sessionStartMs.
	GenerateRealtimeEvents(ctx context.Context, timeline *dto.LyricsTimeline, sessionID uuid.UUID, sessionStartMs int64) ([]*realtimemodels.RealtimeEvent, error)

	// BuildRealtimeEvents converts a lyrics timeline into ordered EventEnvelope objects
	// ready for dispatch through the realtime gateway.
	// Input: timeline from BuildLyricsTimeline, runtime timestamp anchor (ms).
	// Output: ordered slice of EventEnvelopes with type lyrics.line.changed.
	BuildRealtimeEvents(ctx context.Context, timeline *dto.LyricsTimeline, matchID uuid.UUID, sessionStartMs int64) ([]*realtimedto.EventEnvelope, error)

	// FirstAvailableLanguage returns a language code that has stored lyrics for
	// the given song. It is used to avoid hardcoding a language when resolving
	// the current lyric or scheduling lyric events. Returns NotFound when the
	// song has no lyrics in any language.
	FirstAvailableLanguage(ctx context.Context, songID uuid.UUID) (string, error)
}

// songLyricRecord is a minimal projection of the song_lyrics table.
type songLyricRecord struct {
	Lyrics string
}

type lyricsSyncService struct {
	db *gorm.DB
}

// NewLyricsSyncService creates the service. It uses a raw GORM DB handle
// so it can query song_lyrics without a dependency on the songlyric package.
func NewLyricsSyncService(db *gorm.DB) LyricsSyncService {
	return &lyricsSyncService{db: db}
}

func (s *lyricsSyncService) BuildLyricsTimeline(ctx context.Context, songID uuid.UUID, language string) (*dto.LyricsTimeline, error) {
	lines, err := s.fetchAndParse(ctx, songID, language)
	if err != nil {
		return nil, err
	}

	entries := make([]dto.LyricTimelineEntry, len(lines))
	for i, l := range lines {
		entries[i] = dto.LyricTimelineEntry{
			Index:       i,
			TimestampMs: l.TimestampMs,
			Text:        l.Text,
		}
	}

	return &dto.LyricsTimeline{
		SongID:   songID,
		Language: language,
		Entries:  entries,
		Total:    len(entries),
	}, nil
}

func (s *lyricsSyncService) GetLyricsAtTimestamp(ctx context.Context, songID uuid.UUID, language string, offsetMs int64) (*dto.LyricAtTimestampResponse, error) {
	lines, err := s.fetchAndParse(ctx, songID, language)
	if err != nil {
		return nil, err
	}
	if len(lines) == 0 {
		return &dto.LyricAtTimestampResponse{OffsetMs: offsetMs}, nil
	}

	// Binary search for the last line whose timestamp is <= offsetMs.
	idx := sort.Search(len(lines), func(i int) bool {
		return lines[i].TimestampMs > offsetMs
	}) - 1

	resp := &dto.LyricAtTimestampResponse{OffsetMs: offsetMs}

	if idx >= 0 {
		resp.Current = &dto.LyricTimelineEntry{
			Index:       idx,
			TimestampMs: lines[idx].TimestampMs,
			Text:        lines[idx].Text,
		}
	}
	if idx+1 < len(lines) {
		resp.Next = &dto.LyricTimelineEntry{
			Index:       idx + 1,
			TimestampMs: lines[idx+1].TimestampMs,
			Text:        lines[idx+1].Text,
		}
	}

	return resp, nil
}

func (s *lyricsSyncService) GenerateRealtimeEvents(
	_ context.Context,
	timeline *dto.LyricsTimeline,
	sessionID uuid.UUID,
	sessionStartMs int64,
) ([]*realtimemodels.RealtimeEvent, error) {
	events := make([]*realtimemodels.RealtimeEvent, 0, len(timeline.Entries))

	for _, entry := range timeline.Entries {
		payload := map[string]interface{}{
			"index":        entry.Index,
			"text":         entry.Text,
			"timestamp_ms": entry.TimestampMs,
		}
		payloadBytes, err := json.Marshal(payload)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal lyric payload at index %d: %w", entry.Index, err)
		}

		ev := &realtimemodels.RealtimeEvent{
			SessionID:   sessionID,
			EventType:   realtimemodels.EventTypeLyricSync,
			ExecuteAtMs: sessionStartMs + entry.TimestampMs,
			PayloadJSON: string(payloadBytes),
		}
		events = append(events, ev)
	}

	return events, nil
}

func (s *lyricsSyncService) BuildRealtimeEvents(
	_ context.Context,
	timeline *dto.LyricsTimeline,
	matchID uuid.UUID,
	sessionStartMs int64,
) ([]*realtimedto.EventEnvelope, error) {
	envelopes := make([]*realtimedto.EventEnvelope, 0, len(timeline.Entries))

	for _, entry := range timeline.Entries {
		payload := realtimedto.LyricsLinePayload{
			Line:        entry.Text,
			TimestampMs: sessionStartMs + entry.TimestampMs,
			Index:       entry.Index,
		}
		env := realtimedto.NewEnvelope(realtimedto.EventTypeLyricsLineChanged, &matchID, payload)
		envelopes = append(envelopes, env)
	}

	return envelopes, nil
}

func (s *lyricsSyncService) FirstAvailableLanguage(ctx context.Context, songID uuid.UUID) (string, error) {
	var language string
	err := s.db.WithContext(ctx).
		Table("song_lyrics").
		Select("language").
		Where("song_id = ? AND deleted_at IS NULL", songID).
		Order("created_at ASC").
		Limit(1).
		Scan(&language).Error
	if err != nil {
		return "", sharederrors.NewInternal("Failed to resolve lyric language", err)
	}
	if language == "" {
		return "", sharederrors.NewNotFound("No lyrics found for song", nil)
	}
	return language, nil
}

// fetchAndParse retrieves lyric content from the DB and parses it.
func (s *lyricsSyncService) fetchAndParse(ctx context.Context, songID uuid.UUID, language string) ([]lyrics.LyricLine, error) {
	var record songLyricRecord
	err := s.db.WithContext(ctx).
		Table("song_lyrics").
		Select("lyrics").
		Where("song_id = ? AND language = ? AND deleted_at IS NULL", songID, language).
		First(&record).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, sharederrors.NewNotFound("Lyrics not found for song/language", nil)
		}
		return nil, sharederrors.NewInternal("Failed to fetch lyrics", err)
	}

	format := lyrics.DetectFormat(record.Lyrics)

	var lines []lyrics.LyricLine
	if format == "lrc" {
		lines, err = lyrics.ParseLRC(record.Lyrics)
		if err != nil {
			return nil, sharederrors.NewInternal("Failed to parse LRC lyrics", err)
		}
	} else {
		lines = lyrics.ParsePlainText(record.Lyrics)
	}

	// Ensure ascending timestamp order for binary search.
	sort.Slice(lines, func(i, j int) bool {
		return lines[i].TimestampMs < lines[j].TimestampMs
	})

	return lines, nil
}
