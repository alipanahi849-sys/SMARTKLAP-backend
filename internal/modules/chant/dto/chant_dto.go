package dto

import "github.com/google/uuid"

// ChantItem is a single row in the chant list (contract §4.1).
type ChantItem struct {
	ID              uuid.UUID `json:"id"`
	Title           string    `json:"title"`
	SongPoints      int       `json:"song_points"`
	DurationSeconds int       `json:"duration_seconds"`
	IsDone          bool      `json:"is_done"`
	IsNext          bool      `json:"is_next"`
	IsPreview       bool      `json:"is_preview"`
}

// ChantSection groups chants under a display title.
type ChantSection struct {
	Title string      `json:"title"`
	Items []ChantItem `json:"items"`
}

// ChantListFilters are query params for GET /chants.
type ChantListFilters struct {
	MatchID *uuid.UUID
	Search  string
	Cursor  *uuid.UUID
	Limit   int
}

// ChantListMeta is cursor pagination meta for GET /chants.
type ChantListMeta struct {
	Limit      int        `json:"limit"`
	HasMore    bool       `json:"has_more"`
	NextCursor *uuid.UUID `json:"next_cursor,omitempty"`
}

// ChantListResponse is GET /chants.
type ChantListResponse struct {
	MatchTitle string         `json:"match_title"`
	Sections   []ChantSection `json:"sections"`
	Meta       ChantListMeta  `json:"meta"`
}

// ChantCompleteResponse is POST /chants/{id}/complete.
type ChantCompleteResponse struct {
	IsDone       bool `json:"is_done"`
	PointsEarned int  `json:"points_earned"`
	TotalPoints  int  `json:"total_points"`
}

// ChantLyricLine is one synced lyric line (contract §4.3).
type ChantLyricLine struct {
	ID                  int    `json:"id"`
	TimeSeconds         int64  `json:"time_seconds"`
	Text                string `json:"text"`
	FlashDurationMs     int    `json:"flash_duration_ms"`
	VibrationDurationMs int    `json:"vibration_duration_ms"`
}

// ChantLyricsResponse is GET /chants/{id}/lyrics.
type ChantLyricsResponse struct {
	Title    string           `json:"title"`
	AudioURL string           `json:"audio_url"`
	Lyrics   []ChantLyricLine `json:"lyrics"`
}
