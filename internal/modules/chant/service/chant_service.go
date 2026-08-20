package service

import (
	"context"
	"strings"
	"time"

	"clap/internal/modules/chant/dto"
	"clap/internal/modules/chant/models"
	"clap/internal/modules/chant/repository"
	lyricsdto "clap/internal/modules/lyricssync/dto"
	lyricssvc "clap/internal/modules/lyricssync/service"
	matchmodels "clap/internal/modules/match/models"
	matchrepo "clap/internal/modules/match/repository"
	settingsmodels "clap/internal/modules/settings/models"
	settingsrepo "clap/internal/modules/settings/repository"
	songmodels "clap/internal/modules/song/models"
	"clap/internal/shared/errors"
	"clap/internal/shared/logger"
	"clap/pkg/storage"

	"github.com/google/uuid"
)

// Fallbacks used when app_settings has not been configured yet.
const (
	DefaultDailyTarget = 500
	DefaultSongPoints  = 100
	// DefaultOnlinePoints is deliberately higher than DefaultSongPoints: singing
	// live with the crowd is worth more than previewing from the library.
	DefaultOnlinePoints = 200
)

const audioURLExpiry = 2 * time.Hour

// catalogSectionTitle heads the predefined song library, kept separate from the
// scheduled online chants above it.
const catalogSectionTitle = "All chants"

// Catalog songs have no per-chant feedback tuning, so they use the same
// defaults as the chants table.
const (
	defaultFlashDurationMs     = 500
	defaultVibrationDurationMs = 500
)

// listenGrace absorbs clock skew and the round trip between the last lyric and
// the complete call.
const listenGrace = 5 * time.Second

// programNewWindow is how long a freshly earned score stays highlighted.
const programNewWindow = 15 * time.Minute

// ChantService implements the mobile Chants screens (contract §4).
type ChantService interface {
	List(ctx context.Context, userID uuid.UUID, filters dto.ChantListFilters) (*dto.ChantListResponse, error)
	// Lyrics resolves either a catalog song or a scheduled chant depending on
	// source. Pass uuid.Nil for userID to skip personalization (the realtime
	// notifier broadcasts one payload for everyone).
	Lyrics(ctx context.Context, userID, id uuid.UUID, mode, source string) (*dto.ChantLyricsResponse, error)
	Complete(ctx context.Context, userID, id uuid.UUID, source string) (*dto.ChantCompleteResponse, error)
	// Cancel settles a live chant the user walked out of. Only online chants can
	// be cancelled: the catalog is a library to browse, not a scheduled event.
	Cancel(ctx context.Context, userID, id uuid.UUID) error
	TodayStats(ctx context.Context, userID uuid.UUID) (*dto.ChantTodayStatsResponse, error)
	// Program powers the Home "Chants Program" scoreboard.
	Program(ctx context.Context, userID uuid.UUID, limit int) (*dto.ChantProgramResponse, error)

	// Admin surface.
	GetPointsSettings(ctx context.Context) (*dto.ChantPointsSettings, error)
	UpdatePointsSettings(ctx context.Context, req dto.UpdateChantPointsRequest) (*dto.ChantPointsSettings, error)
	SetOnlineChant(ctx context.Context, adminID uuid.UUID, req dto.SetOnlineChantRequest) (*dto.OnlineChantResponse, error)
	ListOnlineChants(ctx context.Context, matchID *uuid.UUID, limit int) ([]dto.OnlineChantResponse, error)
	UnsetOnlineChant(ctx context.Context, id uuid.UUID) error
}

type chantService struct {
	chantRepo    repository.ChantRepository
	matchRepo    matchrepo.MatchRepository
	lyricsSvc    lyricssvc.LyricsSyncService
	settingsRepo settingsrepo.SettingsRepository
	storage      storage.StorageProvider
}

func NewChantService(
	chantRepo repository.ChantRepository,
	matchRepo matchrepo.MatchRepository,
	lyricsSvc lyricssvc.LyricsSyncService,
	settingsRepo settingsrepo.SettingsRepository,
	storageProvider storage.StorageProvider,
) ChantService {
	return &chantService{
		chantRepo:    chantRepo,
		matchRepo:    matchRepo,
		lyricsSvc:    lyricsSvc,
		settingsRepo: settingsRepo,
		storage:      storageProvider,
	}
}

// chantPoints is the resolved, admin-configured scoring for one request.
type chantPoints struct {
	song        int
	online      int
	dailyTarget int
}

func (s *chantService) points(ctx context.Context) chantPoints {
	resolved := chantPoints{
		song:        DefaultSongPoints,
		online:      DefaultOnlinePoints,
		dailyTarget: DefaultDailyTarget,
	}
	if s.settingsRepo == nil {
		return resolved
	}

	settings, err := s.settingsRepo.Get(ctx)
	if err != nil || settings == nil {
		// Scoring must never take the screen down; fall back to the defaults.
		logger.Warn().Msg("chant_points_settings_unavailable")
		return resolved
	}
	if settings.ChantSongPoints > 0 {
		resolved.song = settings.ChantSongPoints
	}
	if settings.ChantOnlinePoints > 0 {
		resolved.online = settings.ChantOnlinePoints
	}
	if settings.ChantDailyTarget > 0 {
		resolved.dailyTarget = settings.ChantDailyTarget
	}
	return resolved
}

func (s *chantService) List(ctx context.Context, userID uuid.UUID, filters dto.ChantListFilters) (*dto.ChantListResponse, error) {
	limit := filters.Limit
	if limit <= 0 {
		limit = 20
	}
	points := s.points(ctx)

	sections, matchTitle, meta, err := s.onlineSections(ctx, userID, filters, limit, points.online)
	if err != nil {
		return nil, err
	}

	// The catalog is the same on every page, so it only rides along on the first.
	if filters.Cursor == nil {
		catalog, catalogErr := s.catalogSection(ctx, userID, filters.Search, limit, points.song)
		if catalogErr != nil {
			return nil, catalogErr
		}
		if catalog != nil {
			sections = append(sections, *catalog)
		}
	}

	return &dto.ChantListResponse{MatchTitle: matchTitle, Sections: sections, Meta: meta}, nil
}

// onlineSections builds the scheduled-chant part of the list: the day-grouped
// sections with sequential is_next unlocking that production already relies on.
func (s *chantService) onlineSections(
	ctx context.Context,
	userID uuid.UUID,
	filters dto.ChantListFilters,
	limit int,
	onlinePoints int,
) ([]dto.ChantSection, string, dto.ChantListMeta, error) {
	sections := make([]dto.ChantSection, 0, 4)
	meta := dto.ChantListMeta{Limit: limit, HasMore: false}

	match, err := s.resolveMatch(ctx, filters.MatchID)
	if err != nil {
		return nil, "", meta, err
	}
	if match == nil {
		return sections, "", meta, nil
	}

	var after *repository.ChantCursorAnchor
	if filters.Cursor != nil {
		cursorChant, cursorErr := s.chantRepo.FindByID(ctx, *filters.Cursor)
		if cursorErr != nil {
			return nil, "", meta, errors.NewBadRequest("Invalid cursor", nil)
		}
		if cursorChant.MatchID != match.ID {
			return nil, "", meta, errors.NewBadRequest("Invalid cursor", nil)
		}
		after = &repository.ChantCursorAnchor{
			ScheduledAt: cursorChant.ScheduledAt,
			ID:          cursorChant.ID,
		}
	}

	chants, err := s.chantRepo.FindByMatchAfter(ctx, match.ID, filters.Search, limit+1, after)
	if err != nil {
		return nil, "", meta, err
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
		return nil, "", meta, err
	}

	now := time.Now().UTC()
	today := now.Truncate(24 * time.Hour)
	tomorrow := today.Add(24 * time.Hour)

	var todayItems, upcomingItems, previousItems []dto.ChantItem
	for _, c := range chants {
		item := dto.ChantItem{
			ID:              c.ID,
			SongID:          c.SongID,
			Title:           c.Title,
			SongPoints:      onlinePoints,
			DurationSeconds: chantDuration(&c),
			IsDone:          done[c.ID],
			// Every defined online chant is playable from this screen whatever its
			// schedule says, so it is never the silent lyrics-only view. The live
			// synced run is still entered from the countdown, not from here.
			IsPreview: false,
			Source:    models.SourceOnline,
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
			return nil, "", meta, checkErr
		}
		if !incompleteBefore {
			markNextInActiveSection(todayItems, upcomingItems, previousItems)
		}
	}

	if len(todayItems) > 0 {
		sections = append(sections, dto.ChantSection{Title: "Todays chants", Items: todayItems})
	}
	if len(upcomingItems) > 0 {
		sections = append(sections, dto.ChantSection{Title: "Upcoming chants", Items: upcomingItems})
	}
	if len(previousItems) > 0 {
		sections = append(sections, dto.ChantSection{Title: "Previous chants", Items: previousItems})
	}

	meta.HasMore = hasMore
	if hasMore && len(chants) > 0 {
		lastID := chants[len(chants)-1].ID
		meta.NextCursor = &lastID
	}

	return sections, match.HomeClub.Name + " - " + match.AwayClub.Name + "'s chants", meta, nil
}

// catalogSection lists the predefined song library. It is independent of any
// match or schedule, so the screen has content even with no fixture coming up.
func (s *chantService) catalogSection(
	ctx context.Context,
	userID uuid.UUID,
	search string,
	limit int,
	songPoints int,
) (*dto.ChantSection, error) {
	songs, err := s.chantRepo.FindCatalogSongs(ctx, search, limit)
	if err != nil {
		return nil, err
	}
	if len(songs) == 0 {
		return nil, nil
	}

	songIDs := make([]uuid.UUID, len(songs))
	for i, song := range songs {
		songIDs[i] = song.ID
	}
	done, err := s.chantRepo.CompletedSongIDs(ctx, userID, songIDs)
	if err != nil {
		return nil, err
	}

	items := make([]dto.ChantItem, len(songs))
	for i, song := range songs {
		items[i] = dto.ChantItem{
			ID:              song.ID,
			SongID:          song.ID,
			Title:           song.Title,
			SongPoints:      songPoints,
			DurationSeconds: song.Duration,
			IsDone:          done[song.ID],
			IsPreview:       false,
			Source:          models.SourceCatalog,
		}
	}

	return &dto.ChantSection{Title: catalogSectionTitle, Items: items}, nil
}

func (s *chantService) Lyrics(ctx context.Context, userID, id uuid.UUID, mode, source string) (*dto.ChantLyricsResponse, error) {
	source = models.NormalizeSource(source)
	points := s.points(ctx)

	resolved, err := s.resolveTarget(ctx, userID, id, source, points)
	if err != nil {
		return nil, err
	}

	// mode is log-only per the contract (§4.3) — it never changes the payload.
	logger.Info().
		Str("chant_id", id.String()).
		Str("mode", mode).
		Str("source", source).
		Msg("chant_lyrics_requested")

	timeline, err := s.loadTimeline(ctx, resolved.target.SongID)
	if err != nil {
		return nil, err
	}

	lines := make([]dto.ChantLyricLine, len(timeline.Entries))
	for i, entry := range timeline.Entries {
		lines[i] = dto.ChantLyricLine{
			ID:                  i + 1,
			TimeSeconds:         float64(entry.TimestampMs) / 1000,
			Text:                entry.Text,
			FlashDurationMs:     resolved.flashMs,
			VibrationDurationMs: resolved.vibrationMs,
		}
	}

	alreadyCompleted := false
	if userID != uuid.Nil {
		alreadyCompleted, err = s.chantRepo.IsCompleted(ctx, resolved.target)
		if err != nil {
			return nil, err
		}
		// Opening the lyrics starts the clock that POST /complete checks.
		if !alreadyCompleted {
			if sessionErr := s.chantRepo.StartListenSession(ctx, userID, resolved.target.SongID, source); sessionErr != nil {
				return nil, sessionErr
			}
		}
	}

	return &dto.ChantLyricsResponse{
		Title:            resolved.title,
		AudioURL:         s.resolveAudioURL(ctx, resolved.song),
		SongID:           resolved.target.SongID,
		Points:           resolved.target.Points,
		AlreadyCompleted: alreadyCompleted,
		Lyrics:           lines,
	}, nil
}

func (s *chantService) Complete(ctx context.Context, userID, id uuid.UUID, source string) (*dto.ChantCompleteResponse, error) {
	source = models.NormalizeSource(source)
	points := s.points(ctx)

	resolved, err := s.resolveTarget(ctx, userID, id, source, points)
	if err != nil {
		return nil, err
	}

	if err := s.assertFullListen(ctx, resolved); err != nil {
		return nil, err
	}

	totalPoints, created, err := s.chantRepo.Complete(ctx, resolved.target)
	if err != nil {
		return nil, err
	}

	pointsEarned := 0
	if created {
		pointsEarned = resolved.target.Points
		logger.Info().
			Str("user_id", userID.String()).
			Str("chant_id", id.String()).
			Str("source", source).
			Int("points_earned", pointsEarned).
			Msg("chant_completed")
	}

	return &dto.ChantCompleteResponse{
		IsDone:       true,
		PointsEarned: pointsEarned,
		TotalPoints:  totalPoints,
	}, nil
}

func (s *chantService) Cancel(ctx context.Context, userID, id uuid.UUID) error {
	resolved, err := s.resolveTarget(ctx, userID, id, models.SourceOnline, s.points(ctx))
	if err != nil {
		return err
	}

	recorded, err := s.chantRepo.Cancel(ctx, resolved.target)
	if err != nil {
		return err
	}
	// Not recorded means the chant was already settled — the fan finished it, or
	// backed out twice. Either way there is nothing to report.
	if recorded {
		logger.Info().
			Str("user_id", userID.String()).
			Str("chant_id", id.String()).
			Msg("chant_cancelled")
	}
	return nil
}

func (s *chantService) Program(ctx context.Context, userID uuid.UUID, limit int) (*dto.ChantProgramResponse, error) {
	if limit <= 0 {
		limit = 20
	}
	points := s.points(ctx)

	todayPoints, err := s.chantRepo.TodayPoints(ctx, userID)
	if err != nil {
		return nil, err
	}

	feed, err := s.chantRepo.TodayProgramFeed(ctx, limit)
	if err != nil {
		return nil, err
	}

	now := time.Now().UTC()
	items := make([]dto.ChantProgramItem, 0, limit)
	for _, row := range feed {
		completedAt := row.CreatedAt
		scorer := row.UserID
		cancelled := row.Status == models.StatusCancelled
		items = append(items, dto.ChantProgramItem{
			ID:       row.ID.String(),
			UserID:   &scorer,
			UserName: displayName(row.FirstName, row.LastName),
			Title:    row.Title,
			Points:   row.PointsEarned,
			// A cancelled attempt is settled but never sung, so it is not "done".
			IsDone:      !cancelled,
			IsCancelled: cancelled,
			IsNew:       now.Sub(completedAt.UTC()) <= programNewWindow,
			IsSelf:      row.UserID == userID,
			CompletedAt: &completedAt,
		})
	}

	// Fill the rest of the card with what this user still has to sing today.
	if remaining := limit - len(items); remaining > 0 {
		pending, pendingErr := s.chantRepo.PendingTodayChants(ctx, userID, remaining)
		if pendingErr != nil {
			return nil, pendingErr
		}
		for _, chant := range pending {
			items = append(items, dto.ChantProgramItem{
				ID:     chant.ID.String(),
				Title:  chant.Title,
				Points: points.online,
				IsDone: false,
				IsSelf: true,
			})
		}
	}

	return &dto.ChantProgramResponse{
		TodayPoints: todayPoints,
		TodayTarget: points.dailyTarget,
		Items:       items,
	}, nil
}

func (s *chantService) TodayStats(ctx context.Context, userID uuid.UUID) (*dto.ChantTodayStatsResponse, error) {
	todayPoints, err := s.chantRepo.TodayPoints(ctx, userID)
	if err != nil {
		return nil, err
	}
	return &dto.ChantTodayStatsResponse{
		TodayPoints: todayPoints,
		TodayTarget: s.points(ctx).dailyTarget,
	}, nil
}

// ─── admin ────────────────────────────────────────────────────────────────────

func (s *chantService) GetPointsSettings(ctx context.Context) (*dto.ChantPointsSettings, error) {
	points := s.points(ctx)
	return &dto.ChantPointsSettings{
		ChantSongPoints:   points.song,
		ChantOnlinePoints: points.online,
		ChantDailyTarget:  points.dailyTarget,
	}, nil
}

func (s *chantService) UpdatePointsSettings(ctx context.Context, req dto.UpdateChantPointsRequest) (*dto.ChantPointsSettings, error) {
	if s.settingsRepo == nil {
		return nil, errors.NewInternal("Settings storage unavailable", nil)
	}

	settings, err := s.settingsRepo.Get(ctx)
	if err != nil {
		return nil, err
	}
	applyPointsRequest(settings, req)

	if err := s.settingsRepo.Save(ctx, settings); err != nil {
		return nil, err
	}
	return &dto.ChantPointsSettings{
		ChantSongPoints:   settings.ChantSongPoints,
		ChantOnlinePoints: settings.ChantOnlinePoints,
		ChantDailyTarget:  settings.ChantDailyTarget,
	}, nil
}

func applyPointsRequest(settings *settingsmodels.AppSettings, req dto.UpdateChantPointsRequest) {
	if req.ChantSongPoints != nil {
		settings.ChantSongPoints = *req.ChantSongPoints
	}
	if req.ChantOnlinePoints != nil {
		settings.ChantOnlinePoints = *req.ChantOnlinePoints
	}
	if req.ChantDailyTarget != nil {
		settings.ChantDailyTarget = *req.ChantDailyTarget
	}
}

func (s *chantService) SetOnlineChant(ctx context.Context, adminID uuid.UUID, req dto.SetOnlineChantRequest) (*dto.OnlineChantResponse, error) {
	song, err := s.chantRepo.FindSongByID(ctx, req.SongID)
	if err != nil {
		return nil, errors.NewNotFound("Song not found", nil)
	}
	if _, err := s.matchRepo.FindByID(ctx, req.MatchID); err != nil {
		return nil, err
	}

	title := strings.TrimSpace(req.Title)
	if title == "" {
		title = song.Title
	}
	duration := req.DurationSeconds
	if duration <= 0 {
		duration = song.Duration
	}
	flash := defaultFlashDurationMs
	if req.FlashDurationMs != nil {
		flash = *req.FlashDurationMs
	}
	vibration := defaultVibrationDurationMs
	if req.VibrationDurationMs != nil {
		vibration = *req.VibrationDurationMs
	}

	chant := &models.Chant{
		MatchID:             req.MatchID,
		SongID:              song.ID,
		Title:               title,
		Points:              s.points(ctx).online,
		DurationSeconds:     duration,
		ScheduledAt:         req.ScheduledAt.UTC(),
		FlashDurationMs:     flash,
		VibrationDurationMs: vibration,
		IsPreview:           false,
		IsActive:            true,
	}
	if adminID != uuid.Nil {
		chant.CreatedBy = &adminID
		chant.UpdatedBy = &adminID
	}

	if err := s.chantRepo.CreateChant(ctx, chant); err != nil {
		return nil, err
	}

	logger.Info().
		Str("chant_id", chant.ID.String()).
		Str("song_id", song.ID.String()).
		Str("match_id", req.MatchID.String()).
		Time("scheduled_at", chant.ScheduledAt).
		Msg("online_chant_scheduled")

	response := toOnlineChantResponse(*chant)
	return &response, nil
}

func (s *chantService) ListOnlineChants(ctx context.Context, matchID *uuid.UUID, limit int) ([]dto.OnlineChantResponse, error) {
	if limit <= 0 {
		limit = 20
	}
	chants, err := s.chantRepo.FindScheduled(ctx, matchID, limit)
	if err != nil {
		return nil, err
	}

	items := make([]dto.OnlineChantResponse, len(chants))
	for i, chant := range chants {
		items[i] = toOnlineChantResponse(chant)
	}
	return items, nil
}

func (s *chantService) UnsetOnlineChant(ctx context.Context, id uuid.UUID) error {
	return s.chantRepo.DeactivateChant(ctx, id)
}

func toOnlineChantResponse(chant models.Chant) dto.OnlineChantResponse {
	return dto.OnlineChantResponse{
		ID:                  chant.ID,
		SongID:              chant.SongID,
		MatchID:             chant.MatchID,
		Title:               chant.Title,
		Points:              chant.Points,
		DurationSeconds:     chantDuration(&chant),
		ScheduledAt:         chant.ScheduledAt,
		FlashDurationMs:     chant.FlashDurationMs,
		VibrationDurationMs: chant.VibrationDurationMs,
		IsActive:            chant.IsActive,
	}
}

// ─── internals ────────────────────────────────────────────────────────────────

// resolvedChant is everything Lyrics and Complete need about a requested
// chant, whichever source it came from.
type resolvedChant struct {
	target      repository.CompletionTarget
	song        *songmodels.Song
	title       string
	flashMs     int
	vibrationMs int
	duration    int
	// scheduledAt is set for online chants only.
	scheduledAt *time.Time
}

// resolveTarget turns an (id, source) pair into that shape. For catalog items
// the ID is a song ID; for online items it is a chant ID.
func (s *chantService) resolveTarget(
	ctx context.Context,
	userID, id uuid.UUID,
	source string,
	points chantPoints,
) (*resolvedChant, error) {
	if source == models.SourceCatalog {
		song, err := s.chantRepo.FindSongByID(ctx, id)
		if err != nil {
			return nil, err
		}
		return &resolvedChant{
			target: repository.CompletionTarget{
				UserID: userID,
				SongID: song.ID,
				Source: models.SourceCatalog,
				Points: points.song,
			},
			song:        song,
			title:       song.Title,
			flashMs:     defaultFlashDurationMs,
			vibrationMs: defaultVibrationDurationMs,
			duration:    song.Duration,
		}, nil
	}

	chant, err := s.chantRepo.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}
	chantID := chant.ID
	scheduledAt := chant.ScheduledAt
	return &resolvedChant{
		target: repository.CompletionTarget{
			UserID:  userID,
			ChantID: &chantID,
			SongID:  chant.SongID,
			Source:  models.SourceOnline,
			Points:  points.online,
		},
		song:        &chant.Song,
		title:       chant.Title,
		flashMs:     chant.FlashDurationMs,
		vibrationMs: chant.VibrationDurationMs,
		duration:    chantDuration(chant),
		scheduledAt: &scheduledAt,
	}, nil
}

// assertFullListen rejects awards that arrive before the track could have
// finished. The clock starts when the lyrics were fetched; for a live chant the
// scheduled start also counts, because clients may replay a cached payload
// instead of re-requesting the lyrics.
func (s *chantService) assertFullListen(ctx context.Context, resolved *resolvedChant) error {
	if resolved.duration <= 0 {
		return nil
	}

	startedAt, err := s.chantRepo.ListenStartedAt(
		ctx, resolved.target.UserID, resolved.target.SongID, resolved.target.Source)
	if err != nil {
		return err
	}
	if startedAt == nil && resolved.scheduledAt != nil {
		startedAt = resolved.scheduledAt
	}
	if startedAt == nil {
		return errors.NewUnprocessable("Open the chant before claiming its points", nil)
	}
	if resolved.scheduledAt != nil && resolved.scheduledAt.Before(*startedAt) {
		// Whichever reference is older gives the user the benefit of the doubt.
		startedAt = resolved.scheduledAt
	}

	required := time.Duration(resolved.duration)*time.Second - listenGrace
	if elapsed := time.Now().UTC().Sub(startedAt.UTC()); elapsed < required {
		return errors.NewUnprocessable("Listen to the whole song to earn its points", nil)
	}
	return nil
}

func displayName(firstName, lastName string) string {
	name := strings.TrimSpace(firstName)
	if last := strings.TrimSpace(lastName); last != "" {
		if name != "" {
			name += " "
		}
		name += last
	}
	return name
}

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
