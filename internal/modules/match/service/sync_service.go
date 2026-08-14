package service

import (
	"context"
	"strconv"
	"time"

	clubmodels "clap/internal/modules/club/models"
	clubrepo "clap/internal/modules/club/repository"
	leaguemodels "clap/internal/modules/league/models"
	leaguerepo "clap/internal/modules/league/repository"
	"clap/internal/modules/match/dto"
	"clap/internal/modules/match/models"
	matchrepo "clap/internal/modules/match/repository"
	playermodels "clap/internal/modules/player/models"
	playerrepo "clap/internal/modules/player/repository"
	seasonmodels "clap/internal/modules/season/models"
	seasonrepo "clap/internal/modules/season/repository"
	settingsrepo "clap/internal/modules/settings/repository"
	"clap/internal/shared/errors"
	"clap/internal/shared/logger"
	"clap/pkg/football"

	"github.com/google/uuid"
)

type SyncService struct {
	provider football.Provider
	matches  matchrepo.MatchRepository
	details  matchrepo.MatchDetailsRepository
	clubs    clubrepo.ClubRepository
	leagues  leaguerepo.LeagueRepository
	seasons  seasonrepo.SeasonRepository
	players  playerrepo.PlayerRepository
	settings settingsrepo.SettingsRepository
}

func NewSyncService(
	provider football.Provider,
	matches matchrepo.MatchRepository,
	details matchrepo.MatchDetailsRepository,
	clubs clubrepo.ClubRepository,
	leagues leaguerepo.LeagueRepository,
	seasons seasonrepo.SeasonRepository,
	players playerrepo.PlayerRepository,
	settings settingsrepo.SettingsRepository,
) *SyncService {
	return &SyncService{
		provider: provider,
		matches:  matches,
		details:  details,
		clubs:    clubs,
		leagues:  leagues,
		seasons:  seasons,
		players:  players,
		settings: settings,
	}
}

func (s *SyncService) Run(ctx context.Context) {
	if s == nil || s.provider == nil || !s.provider.Enabled() {
		logger.Info().Msg("Football match sync disabled (no provider key)")
		return
	}

	tickerLive := time.NewTicker(60 * time.Second)
	tickerList := time.NewTicker(15 * time.Minute)
	defer tickerLive.Stop()
	defer tickerList.Stop()

	if _, err := s.SyncFeaturedClub(ctx); err != nil {
		logger.Warn().Err(err).Msg("initial football match sync failed")
	}

	for {
		select {
		case <-ctx.Done():
			return
		case <-tickerLive.C:
			if err := s.refreshLive(ctx); err != nil {
				logger.Warn().Err(err).Msg("live football match sync failed")
			}
		case <-tickerList.C:
			if _, err := s.SyncFeaturedClub(ctx); err != nil {
				logger.Warn().Err(err).Msg("football fixture list sync failed")
			}
		}
	}
}

func (s *SyncService) EnsureClubFromProvider(ctx context.Context, providerTeamID string) (*clubmodels.Club, error) {
	if s.provider == nil || !s.provider.Enabled() {
		return nil, errors.NewServiceUnavailable("Football data provider is not configured", nil)
	}
	team, err := s.provider.GetTeam(ctx, providerTeamID)
	if err != nil {
		return nil, err
	}
	return s.upsertClub(ctx, *team)
}

func (s *SyncService) SyncFeaturedClub(ctx context.Context) (*dto.SyncResponse, error) {
	if s.provider == nil || !s.provider.Enabled() {
		return nil, errors.NewServiceUnavailable("Football data provider is not configured", nil)
	}

	settings, err := s.settings.Get(ctx)
	if err != nil {
		return nil, err
	}
	if settings.FeaturedClubID == nil {
		return &dto.SyncResponse{}, nil
	}

	club, err := s.clubs.FindByID(ctx, *settings.FeaturedClubID)
	if err != nil {
		return nil, err
	}
	if club.ProviderTeamID == "" {
		return nil, errors.NewUnprocessable("Featured club is missing a football provider team id", nil)
	}

	fixtures, err := s.provider.ListFixtures(ctx, club.ProviderTeamID, 5, 3)
	if err != nil {
		return nil, err
	}

	result := &dto.SyncResponse{}
	for _, fixture := range fixtures {
		created, match, upsertErr := s.upsertFixture(ctx, fixture)
		if upsertErr != nil {
			logger.Warn().Err(upsertErr).Str("fixture", fixture.ProviderID).Msg("fixture upsert failed")
			continue
		}
		if created {
			result.Imported++
		} else {
			result.Updated++
		}
		// Finished-match stats/events are fetched on demand when the user
		// opens game detail, so a list sync does not burn provider quota.
		if (match.Status == "live" || match.Status == "halftime") && needsDetailsSync(match) {
			if err := s.refreshDetails(ctx, match); err != nil {
				logger.Warn().Err(err).Str("match", match.ID.String()).Msg("match details sync failed")
			}
		}
	}
	return result, nil
}

func (s *SyncService) refreshLive(ctx context.Context) error {
	settings, err := s.settings.Get(ctx)
	if err != nil {
		return err
	}
	if settings.FeaturedClubID == nil {
		return nil
	}

	club, err := s.clubs.FindByID(ctx, *settings.FeaturedClubID)
	if err != nil {
		return err
	}
	if club.ProviderTeamID == "" {
		return nil
	}
	if !s.shouldPollLive(ctx, club.ID) {
		return nil
	}

	fixtures, err := s.provider.ListLiveFixtures(ctx, club.ProviderTeamID)
	if err != nil {
		return err
	}
	for _, fixture := range fixtures {
		if _, _, upsertErr := s.upsertFixture(ctx, fixture); upsertErr != nil {
			logger.Warn().Err(upsertErr).Str("fixture", fixture.ProviderID).Msg("live fixture upsert failed")
		}
	}
	return nil
}

func (s *SyncService) shouldPollLive(ctx context.Context, clubID uuid.UUID) bool {
	live, err := s.matches.FindLiveByClub(ctx, clubID)
	if err == nil && len(live) > 0 {
		return true
	}
	current, err := s.matches.FindCurrentForClub(ctx, clubID)
	if err != nil || current == nil {
		return false
	}
	if current.Status == "live" || current.Status == "halftime" {
		return true
	}
	if current.Status == "scheduled" && !current.MatchDateTime.After(time.Now().UTC().Add(30*time.Minute)) {
		return true
	}
	return false
}

func (s *SyncService) RefreshDetailsIfStale(ctx context.Context, match *models.Match, maxAge time.Duration) error {
	if s.provider == nil || !s.provider.Enabled() || match == nil || match.ProviderMatchID == "" {
		return nil
	}
	if !shouldRefreshDetails(match.Status) && match.Status != "scheduled" {
		return nil
	}
	if match.DetailsSyncedAt != nil && time.Since(*match.DetailsSyncedAt) < maxAge {
		return nil
	}
	return s.refreshDetails(ctx, match)
}

func (s *SyncService) upsertFixture(ctx context.Context, fixture football.Fixture) (bool, *models.Match, error) {
	home, err := s.upsertClub(ctx, fixture.HomeTeam)
	if err != nil {
		return false, nil, err
	}
	away, err := s.upsertClub(ctx, fixture.AwayTeam)
	if err != nil {
		return false, nil, err
	}
	league, err := s.upsertLeague(ctx, fixture)
	if err != nil {
		return false, nil, err
	}
	season, err := s.upsertSeason(ctx, league.ID, fixture)
	if err != nil {
		return false, nil, err
	}

	kickoff, err := time.Parse(time.RFC3339, fixture.Kickoff)
	if err != nil {
		kickoff, err = time.Parse("2006-01-02T15:04:05Z07:00", fixture.Kickoff)
		if err != nil {
			kickoff = time.Now().UTC()
		}
	}

	existing, err := s.matches.FindByProviderMatchID(ctx, s.provider.Name(), fixture.ProviderID)
	if err != nil {
		return false, nil, err
	}

	minute := football.FormatMinute(fixture.Elapsed, fixture.Extra, fixture.Status)
	if existing == nil {
		match := &models.Match{
			LeagueID:           league.ID,
			SeasonID:           season.ID,
			HomeClubID:         home.ID,
			AwayClubID:         away.ID,
			Provider:           s.provider.Name(),
			ProviderMatchID:    fixture.ProviderID,
			MatchDateTime:      kickoff,
			StadiumName:        fixture.StadiumName,
			CompetitionLogoURL: fixture.LeagueLogoURL,
			Status:             fixture.Status,
			HomeScore:          fixture.HomeScore,
			AwayScore:          fixture.AwayScore,
			CurrentMinute:      minute,
		}
		if err := s.matches.Create(ctx, match); err != nil {
			return false, nil, err
		}
		created, err := s.matches.FindByID(ctx, match.ID)
		return true, created, err
	}

	existing.LeagueID = league.ID
	existing.SeasonID = season.ID
	existing.HomeClubID = home.ID
	existing.AwayClubID = away.ID
	existing.MatchDateTime = kickoff
	existing.StadiumName = fixture.StadiumName
	existing.CompetitionLogoURL = fixture.LeagueLogoURL
	existing.Status = fixture.Status
	existing.HomeScore = fixture.HomeScore
	existing.AwayScore = fixture.AwayScore
	existing.CurrentMinute = minute
	if err := s.matches.Update(ctx, existing); err != nil {
		return false, nil, err
	}
	updated, err := s.matches.FindByID(ctx, existing.ID)
	return false, updated, err
}

func (s *SyncService) refreshDetails(ctx context.Context, match *models.Match) error {
	if match.ProviderMatchID == "" {
		return nil
	}
	details, err := s.provider.GetFixtureDetails(ctx, match.ProviderMatchID)
	if err != nil {
		return err
	}

	match.Status = details.Fixture.Status
	match.HomeScore = details.Fixture.HomeScore
	match.AwayScore = details.Fixture.AwayScore
	match.CurrentMinute = football.FormatMinute(details.Fixture.Elapsed, details.Fixture.Extra, details.Fixture.Status)
	if details.Fixture.StadiumName != "" {
		match.StadiumName = details.Fixture.StadiumName
	}
	if details.Fixture.LeagueLogoURL != "" {
		match.CompetitionLogoURL = details.Fixture.LeagueLogoURL
	}
	now := time.Now().UTC()
	match.DetailsSyncedAt = &now
	if err := s.matches.Update(ctx, match); err != nil {
		return err
	}

	stats := make([]models.MatchStat, 0, len(details.Stats))
	for i, item := range details.Stats {
		stats = append(stats, models.MatchStat{
			MatchID:   match.ID,
			Label:     item.Label,
			HomeValue: item.Home,
			AwayValue: item.Away,
			SortOrder: i,
		})
	}
	if err := s.details.ReplaceStats(ctx, match.ID, stats); err != nil {
		return err
	}

	timeline := buildTimeline(match, details)
	if err := s.details.ReplaceTimeline(ctx, match.ID, timeline); err != nil {
		return err
	}

	lineup, err := s.buildLineup(ctx, match, details.Lineups)
	if err != nil {
		return err
	}
	return s.details.ReplaceLineup(ctx, match.ID, lineup)
}

func (s *SyncService) buildLineup(ctx context.Context, match *models.Match, lineups []football.Lineup) ([]models.MatchLineupPlayer, error) {
	out := make([]models.MatchLineupPlayer, 0)
	for _, lineup := range lineups {
		side := "away"
		clubID := match.AwayClubID
		if lineup.TeamID == match.HomeClub.ProviderTeamID {
			side = "home"
			clubID = match.HomeClubID
		} else if lineup.TeamID == match.AwayClub.ProviderTeamID {
			side = "away"
			clubID = match.AwayClubID
		}

		for i, item := range lineup.Players {
			var playerID *uuid.UUID
			if item.ProviderPlayerID != "" {
				player, err := s.upsertLineupPlayer(ctx, clubID, lineup.Formation, item)
				if err != nil {
					return nil, err
				}
				playerID = &player.ID
			}
			out = append(out, models.MatchLineupPlayer{
				MatchID:      match.ID,
				ClubID:       clubID,
				PlayerID:     playerID,
				Side:         side,
				Name:         item.Name,
				Position:     football.MapPosition(item.Position),
				JerseyNumber: item.Number,
				PhotoURL:     item.PhotoURL,
				IsStarter:    item.Starter,
				SortOrder:    football.PositionSortOrder(item.Position)*100 + i,
			})
		}
	}
	return out, nil
}

func (s *SyncService) upsertLineupPlayer(ctx context.Context, clubID uuid.UUID, formation string, item football.LineupPlayer) (*playermodels.Player, error) {
	existing, err := s.players.FindByProviderPlayerID(ctx, s.provider.Name(), item.ProviderPlayerID)
	if err != nil {
		return nil, err
	}
	if existing == nil {
		player := &playermodels.Player{
			ClubID:           clubID,
			Name:             item.Name,
			JerseyNumber:     item.Number,
			Position:         football.MapPosition(item.Position),
			PhotoURL:         item.PhotoURL,
			Formation:        formation,
			Provider:         s.provider.Name(),
			ProviderPlayerID: item.ProviderPlayerID,
			RadarStats:       []playermodels.RadarStat{},
			IsActive:         true,
		}
		if err := s.players.Create(ctx, player); err != nil {
			return nil, err
		}
		return player, nil
	}
	existing.ClubID = clubID
	if item.Name != "" {
		existing.Name = item.Name
	}
	if item.Number > 0 {
		existing.JerseyNumber = item.Number
	}
	if item.Position != "" {
		existing.Position = football.MapPosition(item.Position)
	}
	if item.PhotoURL != "" {
		existing.PhotoURL = item.PhotoURL
	}
	if formation != "" {
		existing.Formation = formation
	}
	if err := s.players.Update(ctx, existing); err != nil {
		return nil, err
	}
	return existing, nil
}

func (s *SyncService) upsertClub(ctx context.Context, team football.Team) (*clubmodels.Club, error) {
	if team.ProviderID == "" {
		return nil, errors.NewBadRequest("provider team id is required", nil)
	}
	existing, err := s.clubs.FindByProviderTeamID(ctx, s.provider.Name(), team.ProviderID)
	if err != nil {
		return nil, err
	}
	shortName := team.Code
	if shortName == "" {
		shortName = team.Name
	}
	if existing == nil {
		club := &clubmodels.Club{
			Name:           team.Name,
			ShortName:      shortName,
			LogoURL:        team.LogoURL,
			Country:        team.Country,
			VenueName:      team.VenueName,
			Provider:       s.provider.Name(),
			ProviderTeamID: team.ProviderID,
			IsActive:       true,
		}
		if err := s.clubs.Create(ctx, club); err != nil {
			return nil, err
		}
		return club, nil
	}
	existing.Name = team.Name
	existing.ShortName = shortName
	existing.LogoURL = team.LogoURL
	existing.Country = team.Country
	if team.VenueName != "" {
		existing.VenueName = team.VenueName
	}
	if err := s.clubs.Update(ctx, existing); err != nil {
		return nil, err
	}
	return existing, nil
}

func (s *SyncService) upsertLeague(ctx context.Context, fixture football.Fixture) (*leaguemodels.League, error) {
	existing, err := s.leagues.FindByProviderLeagueID(ctx, s.provider.Name(), fixture.LeagueProviderID)
	if err != nil {
		return nil, err
	}
	if existing == nil {
		league := &leaguemodels.League{
			Name:             fixture.LeagueName,
			Country:          fixture.LeagueCountry,
			Provider:         s.provider.Name(),
			ProviderLeagueID: fixture.LeagueProviderID,
			LogoURL:          fixture.LeagueLogoURL,
			IsActive:         true,
		}
		if err := s.leagues.Create(ctx, league); err != nil {
			return nil, err
		}
		return league, nil
	}
	existing.Name = fixture.LeagueName
	existing.Country = fixture.LeagueCountry
	existing.LogoURL = fixture.LeagueLogoURL
	if err := s.leagues.Update(ctx, existing); err != nil {
		return nil, err
	}
	return existing, nil
}

func (s *SyncService) upsertSeason(ctx context.Context, leagueID uuid.UUID, fixture football.Fixture) (*seasonmodels.Season, error) {
	year := fixture.Season
	if year == "" {
		year = strconv.Itoa(time.Now().UTC().Year())
	}
	existing, err := s.seasons.FindByLeagueAndProviderSeason(ctx, leagueID, year)
	if err != nil {
		return nil, err
	}
	startYear, _ := strconv.Atoi(year)
	if startYear == 0 {
		startYear = time.Now().UTC().Year()
	}
	start := time.Date(startYear, time.July, 1, 0, 0, 0, 0, time.UTC)
	end := time.Date(startYear+1, time.June, 30, 0, 0, 0, 0, time.UTC)
	if existing == nil {
		season := &seasonmodels.Season{
			LeagueID:         leagueID,
			Name:             year,
			StartDate:        start,
			EndDate:          end,
			ProviderSeasonID: year,
			IsActive:         true,
		}
		if err := s.seasons.Create(ctx, season); err != nil {
			return nil, err
		}
		return season, nil
	}
	return existing, nil
}

func buildTimeline(match *models.Match, details *football.FixtureDetails) []models.MatchTimelineEvent {
	homeScore := 0
	awayScore := 0
	events := make([]models.MatchTimelineEvent, 0, len(details.Events)+2)
	sortOrder := 0
	htInserted := false
	ftInserted := false

	for _, item := range details.Events {
		eventType := football.MapEventType(item.Type, item.Detail)
		if eventType == "" {
			continue
		}
		minuteNum := item.Elapsed
		if !htInserted && minuteNum > 45 {
			events = append(events, models.MatchTimelineEvent{
				MatchID:   match.ID,
				Kind:      "marker",
				Minute:    "45",
				Score:     formatDashScore(homeScore, awayScore),
				SortOrder: sortOrder,
			})
			sortOrder++
			htInserted = true
		}

		side := "away"
		if item.TeamID == match.HomeClub.ProviderTeamID {
			side = "home"
		}
		if eventType == "goal" {
			if side == "home" {
				homeScore++
			} else {
				awayScore++
			}
		}

		events = append(events, models.MatchTimelineEvent{
			MatchID:       match.ID,
			Kind:          "event",
			Side:          side,
			EventType:     eventType,
			PlayerName:    item.PlayerName,
			SubPlayerName: item.AssistName,
			Minute:        football.FormatMinute(item.Elapsed, item.Extra, "live") + `"`,
			SortOrder:     sortOrder,
		})
		sortOrder++
	}

	if match.Status == "finished" && !ftInserted {
		if !htInserted {
			events = append([]models.MatchTimelineEvent{{
				MatchID:   match.ID,
				Kind:      "marker",
				Minute:    "45",
				Score:     formatDashScore(homeScore, awayScore),
				SortOrder: -1,
			}}, events...)
		}
		finalHome := derefScore(match.HomeScore)
		finalAway := derefScore(match.AwayScore)
		events = append([]models.MatchTimelineEvent{{
			MatchID:   match.ID,
			Kind:      "marker",
			Minute:    football.FormatMinute(details.Fixture.Elapsed, details.Fixture.Extra, "finished"),
			Score:     formatDashScore(finalHome, finalAway),
			SortOrder: -2,
		}}, events...)
	}

	// Timeline UI is newest-first; keep provider chronological then reverse via sort_order.
	for i := range events {
		events[i].SortOrder = i
	}
	reversed := make([]models.MatchTimelineEvent, len(events))
	for i := range events {
		reversed[len(events)-1-i] = events[i]
		reversed[len(events)-1-i].SortOrder = i
	}
	return reversed
}

func formatDashScore(home, away int) string {
	return strconv.Itoa(home) + " - " + strconv.Itoa(away)
}

func needsDetailsSync(match *models.Match) bool {
	if match == nil {
		return false
	}
	if match.Status == "finished" && match.DetailsSyncedAt != nil {
		return false
	}
	return true
}

func shouldRefreshDetails(status string) bool {
	switch status {
	case "live", "halftime", "finished":
		return true
	default:
		return false
	}
}

func appendLiveUnique(all []football.Fixture, live []football.Fixture) []football.Fixture {
	seen := map[string]struct{}{}
	for _, item := range all {
		seen[item.ProviderID] = struct{}{}
	}
	for _, item := range live {
		if _, ok := seen[item.ProviderID]; ok {
			continue
		}
		all = append(all, item)
	}
	return all
}
