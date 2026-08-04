package service

import (
	"context"
	"time"

	"clap/internal/modules/chant/dto"
	"clap/internal/modules/chant/models"
	"clap/internal/modules/chant/repository"
	lyricsdto "clap/internal/modules/lyricssync/dto"
	lyricssvc "clap/internal/modules/lyricssync/service"
	matchmodels "clap/internal/modules/match/models"
	matchrepo "clap/internal/modules/match/repository"
	songmodels "clap/internal/modules/song/models"
	"clap/internal/shared/errors"
	"clap/internal/shared/logger"
	"clap/pkg/storage"

	"github.com/google/uuid"
)

// DefaultDailyTarget is the "today_target" points goal shown on the home screen.
// Configurable per deployment via NewChantServiceWithTarget.
const DefaultDailyTarget = 500

const audioURLExpiry = 2 * time.Hour

// ChantService implements the mobile Chants screens (contract §4).
type ChantService interface {
	List(ctx context.Context, userID uuid.UUID, matchID *uuid.UUID, search string) (*dto.ChantListResponse, error)
	Lyrics(ctx context.Context, chantID uuid.UUID, mode string) (*dto.ChantLyricsResponse, error)
	Complete(ctx context.Context, userID, chantID uuid.UUID) (*dto.ChantCompleteResponse, error)
	// TodayProgram powers the Home "chant program" card (contract §3.1).
	TodayProgram(ctx context.Context, userID uuid.UUID, recentLimit int) (todayPoints, todayTarget int, recent []models.ChantCompletion, chants map[uuid.UUID]models.Chant, err error)
}

type chantService struct {
	chantRepo   repository.ChantRepository
	matchRepo   matchrepo.MatchRepository
	lyricsSvc   lyricssvc.LyricsSyncService
	storage     storage.StorageProvider
	dailyTarget int
}

func NewChantService(
	chantRepo repository.ChantRepository,
	matchRepo matchrepo.MatchRepository,
	lyricsSvc lyricssvc.LyricsSyncService,
	storageProvider storage.StorageProvider,
) ChantService {
	return NewChantServiceWithTarget(chantRepo, matchRepo, lyricsSvc, storageProvider, DefaultDailyTarget)
}

func NewChantServiceWithTarget(
	chantRepo repository.ChantRepository,
	matchRepo matchrepo.MatchRepository,
	lyricsSvc lyricssvc.LyricsSyncService,
	storageProvider storage.StorageProvider,
	dailyTarget int,
) ChantService {
	if dailyTarget <= 0 {
		dailyTarget = DefaultDailyTarget
	}
	return &chantService{
		chantRepo:   chantRepo,
		matchRepo:   matchRepo,
		lyricsSvc:   lyricsSvc,
		storage:     storageProvider,
		dailyTarget: dailyTarget,
	}
}

func (s *chantService) List(ctx context.Context, userID uuid.UUID, matchID *uuid.UUID, search string) (*dto.ChantListResponse, error) {
	match, err := s.resolveMatch(ctx, matchID)
	if err != nil {
		return nil, err
	}
	if match == nil {
		return &dto.ChantListResponse{MatchTitle: "", Sections: []dto.ChantSection{}}, nil
	}

	chants, err := s.chantRepo.FindByMatch(ctx, match.ID, search)
	if err != nil {
		return nil, err
	}

	chantIDs := make([]uuid.UUID, len(chants))
	for i, c := range chants {
		chantIDs[i] = c.ID
	}
	done, err := s.chantRepo.CompletedChantIDs(ctx, userID, chantIDs)
	if err != nil {
		return nil, err
	}

	// The "next" chant is the earliest not-done chant that hasn't passed yet.
	now := time.Now().UTC()
	nextID := uuid.Nil
	for _, c := range chants {
		if !done[c.ID] && !c.ScheduledAt.Before(now.Add(-time.Duration(c.DurationSeconds)*time.Second)) {
			nextID = c.ID
			break
		}
	}

	today := now.Truncate(24 * time.Hour)
	tomorrow := today.Add(24 * time.Hour)

	var todayItems, upcomingItems, previousItems []dto.ChantItem
	for _, c := range chants {
		item := dto.ChantItem{
			ID:              c.ID,
			Title:           c.Title,
			SongPoints:      c.Points,
			DurationSeconds: chantDuration(&c),
			IsDone:          done[c.ID],
			IsNext:          c.ID == nextID,
			// The contract exposes is_liked but defines no like/unlike endpoint
			// for chants; it stays false until the contract adds one.
			IsLiked:   false,
			IsPreview: c.IsPreview,
		}
		switch {
		case c.ScheduledAt.Before(today):
			previousItems = append(previousItems, item)
		case c.ScheduledAt.Before(tomorrow):
			todayItems = append(todayItems, item)
		default:
			upcomingItems = append(upcomingItems, item)
		}
	}

	sections := make([]dto.ChantSection, 0, 3)
	if len(todayItems) > 0 {
		sections = append(sections, dto.ChantSection{Title: "Todays chants", Items: todayItems})
	}
	if len(upcomingItems) > 0 {
		sections = append(sections, dto.ChantSection{Title: "Upcoming chants", Items: upcomingItems})
	}
	if len(previousItems) > 0 {
		sections = append(sections, dto.ChantSection{Title: "Previous chants", Items: previousItems})
	}

	title := match.HomeClub.Name + " - " + match.AwayClub.Name + "'s chants"

	return &dto.ChantListResponse{MatchTitle: title, Sections: sections}, nil
}

func (s *chantService) Lyrics(ctx context.Context, chantID uuid.UUID, mode string) (*dto.ChantLyricsResponse, error) {
	chant, err := s.chantRepo.FindByID(ctx, chantID)
	if err != nil {
		return nil, err
	}

	// mode is log-only per the contract (§4.3) — it never changes the payload.
	logger.Info().
		Str("chant_id", chantID.String()).
		Str("mode", mode).
		Msg("chant_lyrics_requested")

	timeline, err := s.loadTimeline(ctx, chant.SongID)
	if err != nil {
		return nil, err
	}

	lines := make([]dto.ChantLyricLine, len(timeline.Entries))
	for i, entry := range timeline.Entries {
		lines[i] = dto.ChantLyricLine{
			ID:                  i + 1,
			TimeSeconds:         entry.TimestampMs / 1000,
			Text:                entry.Text,
			FlashDurationMs:     chant.FlashDurationMs,
			VibrationDurationMs: chant.VibrationDurationMs,
		}
	}

	return &dto.ChantLyricsResponse{
		Title:    chant.Title,
		AudioURL: s.resolveAudioURL(ctx, &chant.Song),
		Lyrics:   lines,
	}, nil
}

func (s *chantService) Complete(ctx context.Context, userID, chantID uuid.UUID) (*dto.ChantCompleteResponse, error) {
	chant, err := s.chantRepo.FindByID(ctx, chantID)
	if err != nil {
		return nil, err
	}

	totalPoints, created, err := s.chantRepo.Complete(ctx, chantID, userID, chant.Points)
	if err != nil {
		return nil, err
	}

	pointsEarned := 0
	if created {
		pointsEarned = chant.Points
		logger.Info().
			Str("user_id", userID.String()).
			Str("chant_id", chantID.String()).
			Str("match_id", chant.MatchID.String()).
			Int("points_earned", chant.Points).
			Msg("chant_completed")
	}

	return &dto.ChantCompleteResponse{
		IsDone:       true,
		PointsEarned: pointsEarned,
		TotalPoints:  totalPoints,
	}, nil
}

func (s *chantService) TodayProgram(ctx context.Context, userID uuid.UUID, recentLimit int) (int, int, []models.ChantCompletion, map[uuid.UUID]models.Chant, error) {
	todayPoints, err := s.chantRepo.TodayPoints(ctx, userID)
	if err != nil {
		return 0, 0, nil, nil, err
	}
	recent, chants, err := s.chantRepo.TodayCompletions(ctx, userID, recentLimit)
	if err != nil {
		return 0, 0, nil, nil, err
	}
	return todayPoints, s.dailyTarget, recent, chants, nil
}

// ─── internals ────────────────────────────────────────────────────────────────

func (s *chantService) resolveMatch(ctx context.Context, matchID *uuid.UUID) (*matchmodels.Match, error) {
	if matchID != nil {
		m, err := s.matchRepo.FindByID(ctx, *matchID)
		if err != nil {
			return nil, err
		}
		return m, nil
	}

	// No match specified: prefer the live match, then the next upcoming one.
	if live, err := s.matchRepo.FindLive(ctx); err == nil && len(live) > 0 {
		return &live[0], nil
	}
	if upcoming, _, err := s.matchRepo.FindUpcoming(ctx, 1, 1); err == nil && len(upcoming) > 0 {
		return &upcoming[0], nil
	}
	return nil, nil
}

func (s *chantService) loadTimeline(ctx context.Context, songID uuid.UUID) (*lyricsdto.LyricsTimeline, error) {
	language, err := s.lyricsSvc.FirstAvailableLanguage(ctx, songID)
	if err != nil {
		return nil, errors.NewNotFound("No lyrics available for this chant", err)
	}
	timeline, err := s.lyricsSvc.BuildLyricsTimeline(ctx, songID, language)
	if err != nil {
		return nil, err
	}
	return timeline, nil
}

func (s *chantService) resolveAudioURL(ctx context.Context, song *songmodels.Song) string {
	if song == nil {
		return ""
	}
	if song.StorageKey != "" && s.storage != nil {
		if url, err := s.storage.GenerateSignedURL(ctx, song.StorageKey, audioURLExpiry); err == nil {
			return url
		}
	}
	return song.AudioURL
}

func chantDuration(c *models.Chant) int {
	if c.DurationSeconds > 0 {
		return c.DurationSeconds
	}
	return c.Song.Duration
}
