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
	List(ctx context.Context, userID uuid.UUID, filters dto.ChantListFilters) (*dto.ChantListResponse, error)
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

func (s *chantService) List(ctx context.Context, userID uuid.UUID, filters dto.ChantListFilters) (*dto.ChantListResponse, error) {
	match, err := s.resolveMatch(ctx, filters.MatchID)
	if err != nil {
		return nil, err
	}
	if match == nil {
		return &dto.ChantListResponse{
			MatchTitle: "",
			Sections:   []dto.ChantSection{},
			Meta:       dto.ChantListMeta{Limit: filters.Limit, HasMore: false},
		}, nil
	}

	limit := filters.Limit
	if limit <= 0 {
		limit = 20
	}

	var after *repository.ChantCursorAnchor
	if filters.Cursor != nil {
		cursorChant, cursorErr := s.chantRepo.FindByID(ctx, *filters.Cursor)
		if cursorErr != nil {
			return nil, errors.NewBadRequest("Invalid cursor", nil)
		}
		if cursorChant.MatchID != match.ID {
			return nil, errors.NewBadRequest("Invalid cursor", nil)
		}
		after = &repository.ChantCursorAnchor{
			ScheduledAt: cursorChant.ScheduledAt,
			ID:          cursorChant.ID,
		}
	}

	chants, err := s.chantRepo.FindByMatchAfter(ctx, match.ID, filters.Search, limit+1, after)
	if err != nil {
		return nil, err
	}

	hasMore := len(chants) > limit
	if hasMore {
		chants = chants[:limit]
	}

	chantIDs := make([]uuid.UUID, len(chants))
	for i, c := range chants {
		chantIDs[i] = c.ID
	}
	done, err := s.chantRepo.CompletedChantIDs(ctx, userID, chantIDs)
	if err != nil {
		return nil, err
	}

	now := time.Now().UTC()
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
			IsPreview:       c.IsPreview,
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

	if after == nil {
		markNextInActiveSection(todayItems, upcomingItems, previousItems)
	} else {
		incompleteBefore, checkErr := s.chantRepo.HasIncompleteAtOrBefore(ctx, userID, match.ID, filters.Search, after)
		if checkErr != nil {
			return nil, checkErr
		}
		if !incompleteBefore {
			markNextInActiveSection(todayItems, upcomingItems, previousItems)
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

	meta := dto.ChantListMeta{
		Limit:   limit,
		HasMore: hasMore,
	}
	if hasMore && len(chants) > 0 {
		lastID := chants[len(chants)-1].ID
		meta.NextCursor = &lastID
	}

	return &dto.ChantListResponse{MatchTitle: title, Sections: sections, Meta: meta}, nil
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

// markNextInActiveSection marks is_next only in the first section (in API order)
// that still has incomplete chants; other sections stay locked until it is finished.
func markNextInActiveSection(sections ...[]dto.ChantItem) {
	activeMarked := false
	for _, items := range sections {
		if len(items) == 0 {
			continue
		}
		if !activeMarked && sectionHasIncomplete(items) {
			markNextInSection(items)
			activeMarked = true
			continue
		}
		for i := range items {
			items[i].IsNext = false
		}
	}
}

func sectionHasIncomplete(items []dto.ChantItem) bool {
	for _, item := range items {
		if !item.IsDone {
			return true
		}
	}
	return false
}

// markNextInSection sets is_next on the first incomplete item that follows
// a completed prefix within the same section (parent group).
func markNextInSection(items []dto.ChantItem) {
	for i := range items {
		items[i].IsNext = false
		if items[i].IsDone {
			continue
		}
		allPrevDone := true
		for j := 0; j < i; j++ {
			if !items[j].IsDone {
				allPrevDone = false
				break
			}
		}
		if allPrevDone {
			items[i].IsNext = true
			return
		}
	}
}
