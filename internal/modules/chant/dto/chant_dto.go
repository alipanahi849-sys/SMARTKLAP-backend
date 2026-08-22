package dto

import (
	"time"

	"github.com/google/uuid"
)

// ChantItem is a single row in the chant list (contract §4.1).
//
// Source tells the client which endpoint family the ID belongs to: "catalog"
// items are songs from the predefined library and their ID is a song ID, while
// "online" items are scheduled chants and their ID is a chant ID.
type ChantItem struct {
	ID              uuid.UUID `json:"id"`
	SongID          uuid.UUID `json:"song_id"`
	Title           string    `json:"title"`
	SongPoints      int       `json:"song_points"`
	DurationSeconds int       `json:"duration_seconds"`
	IsDone          bool      `json:"is_done"`
	IsPreview       bool      `json:"is_preview"`
	Source          string    `json:"source"`
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
	ID int `json:"id"`
	// TimeSeconds must stay fractional. Clients fire the torch and haptics on
	// these offsets, so rounding to a whole second puts the flash up to a
	// second away from the beat — and by a different amount on every line.
	TimeSeconds         float64 `json:"time_seconds"`
	Text                string  `json:"text"`
	FlashDurationMs     int     `json:"flash_duration_ms"`
	VibrationDurationMs int     `json:"vibration_duration_ms"`
}

// ChantTodayStatsResponse is GET /chants/me/today.
type ChantTodayStatsResponse struct {
	TodayPoints int `json:"today_points"`
	TodayTarget int `json:"today_target"`
}

// ChantLyricsResponse is GET /chants/{id}/lyrics.
type ChantLyricsResponse struct {
	Title    string    `json:"title"`
	AudioURL string    `json:"audio_url"`
	SongID   uuid.UUID `json:"song_id"`
	// Points is what finishing this chant is worth right now, so the leave
	// confirmation can name the stake.
	Points           int              `json:"points"`
	AlreadyCompleted bool             `json:"already_completed"`
	Lyrics           []ChantLyricLine `json:"lyrics"`
}

// ChantProgramItem is one row of the Home "Chants Program" scoreboard: either
// a chant the user still has to sing, or points they earned today. The
// scoreboard is personal, so it never carries another fan's rows.
type ChantProgramItem struct {
	ID     string `json:"id"`
	Title  string `json:"title"`
	Points int    `json:"points"`
	IsDone bool   `json:"is_done"`
	// IsCancelled marks an attempt the fan walked out of. Such a row is settled
	// like a completion — it is simply worth nothing.
	IsCancelled bool `json:"is_cancelled"`
	IsNew       bool `json:"is_new"`
	// StartsAt is the schedule of a chant still to sing, so the card can say
	// when the crowd picks it up. Settled rows carry CompletedAt instead.
	StartsAt    *time.Time `json:"starts_at,omitempty"`
	CompletedAt *time.Time `json:"completed_at,omitempty"`
}

// ChantProgramResponse is GET /chants/program.
type ChantProgramResponse struct {
	TodayPoints int                `json:"today_points"`
	TodayTarget int                `json:"today_target"`
	Items       []ChantProgramItem `json:"items"`
	Meta        ChantListMeta      `json:"meta"`
}

// ChantPointsSettings is the admin-configurable point payload
// (GET/PUT /admin/settings/chant-points).
type ChantPointsSettings struct {
	ChantSongPoints   int `json:"chant_song_points"`
	ChantOnlinePoints int `json:"chant_online_points"`
	ChantDailyTarget  int `json:"chant_daily_target"`
}

// UpdateChantPointsRequest updates only the values that are supplied.
type UpdateChantPointsRequest struct {
	ChantSongPoints   *int `json:"chant_song_points"   binding:"omitempty,min=0,max=100000"`
	ChantOnlinePoints *int `json:"chant_online_points" binding:"omitempty,min=0,max=100000"`
	ChantDailyTarget  *int `json:"chant_daily_target"  binding:"omitempty,min=1,max=1000000"`
}

// SetOnlineChantRequest promotes a song from the predefined catalog into a
// scheduled online chant for a match (POST /admin/chants).
type SetOnlineChantRequest struct {
	SongID              uuid.UUID `json:"song_id"      binding:"required"`
	MatchID             uuid.UUID `json:"match_id"     binding:"required"`
	ScheduledAt         time.Time `json:"scheduled_at" binding:"required"`
	Title               string    `json:"title"`
	DurationSeconds     int       `json:"duration_seconds"      binding:"omitempty,min=0"`
	FlashDurationMs     *int      `json:"flash_duration_ms"     binding:"omitempty,min=0"`
	VibrationDurationMs *int      `json:"vibration_duration_ms" binding:"omitempty,min=0"`
}

// OnlineChantResponse describes a scheduled online chant for the admin panel.
type OnlineChantResponse struct {
	ID                  uuid.UUID `json:"id"`
	SongID              uuid.UUID `json:"song_id"`
	MatchID             uuid.UUID `json:"match_id"`
	Title               string    `json:"title"`
	Points              int       `json:"points"`
	DurationSeconds     int       `json:"duration_seconds"`
	ScheduledAt         time.Time `json:"scheduled_at"`
	FlashDurationMs     int       `json:"flash_duration_ms"`
	VibrationDurationMs int       `json:"vibration_duration_ms"`
	IsActive            bool      `json:"is_active"`
}
