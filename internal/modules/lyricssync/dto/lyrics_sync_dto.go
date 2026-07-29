package dto

import "github.com/google/uuid"

// LyricTimelineEntry is a single ordered lyric line with its playback offset.
type LyricTimelineEntry struct {
	Index       int    `json:"index"`
	TimestampMs int64  `json:"timestamp_ms"`
	Text        string `json:"text"`
	SessionID   string `json:"session_id,omitempty"`
}

// LyricsTimeline is the full ordered sequence for a song.
type LyricsTimeline struct {
	SongID   uuid.UUID            `json:"song_id"`
	Language string               `json:"language"`
	Entries  []LyricTimelineEntry `json:"entries"`
	Total    int                  `json:"total"`
}

// LyricAtTimestampResponse is returned by GetLyricsAtTimestamp.
type LyricAtTimestampResponse struct {
	Current  *LyricTimelineEntry `json:"current"`
	Next     *LyricTimelineEntry `json:"next,omitempty"`
	OffsetMs int64               `json:"offset_ms"`
}
