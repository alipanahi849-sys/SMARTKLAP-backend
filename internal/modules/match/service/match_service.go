package service

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	clubmodels "clap/internal/modules/club/models"
	clubrepo "clap/internal/modules/club/repository"
	"clap/internal/modules/match/dto"
	"clap/internal/modules/match/models"
	matchrepo "clap/internal/modules/match/repository"
	playermodels "clap/internal/modules/player/models"
	playerrepo "clap/internal/modules/player/repository"
	settingsrepo "clap/internal/modules/settings/repository"
	"clap/internal/shared/errors"
	"clap/internal/shared/logger"
	"clap/pkg/football"

	"github.com/google/uuid"
)

type MatchService interface {
	GetCurrent(ctx context.Context) (*dto.CurrentMatchResponse, error)
	GetByID(ctx context.Context, id uuid.UUID) (*dto.MatchDetailResponse, error)
	GetPlayer(ctx context.Context, id uuid.UUID) (*dto.PlayerDetailResponse, error)
	SearchProviderTeams(ctx context.Context, query string) ([]dto.ProviderTeam, error)
	GetFeaturedClub(ctx context.Context) (*dto.FeaturedClubResponse, error)
	SetFeaturedClub(ctx context.Context, req dto.SetFeaturedClubRequest) (*dto.FeaturedClubResponse, error)
	SyncNow(ctx context.Context) (*dto.SyncResponse, error)
}

type matchService struct {
	matches  matchrepo.MatchRepository
	details  matchrepo.MatchDetailsRepository
	clubs    clubrepo.ClubRepository
	players  playerrepo.PlayerRepository
	settings settingsrepo.SettingsRepository
	provider football.Provider
	syncer   *SyncService
}

func NewMatchService(
	matches matchrepo.MatchRepository,
	details matchrepo.MatchDetailsRepository,
	clubs clubrepo.ClubRepository,
	players playerrepo.PlayerRepository,
	settings settingsrepo.SettingsRepository,
	provider football.Provider,
	syncer *SyncService,
) MatchService {
	return &matchService{
		matches:  matches,
		details:  details,
		clubs:    clubs,
		players:  players,
		settings: settings,
		provider: provider,
		syncer:   syncer,
	}
}

func (s *matchService) GetCurrent(ctx context.Context) (*dto.CurrentMatchResponse, error) {
	club, err := s.featuredClub(ctx)
	if err != nil {
		return nil, err
	}
	if club == nil {
		return nil, nil
	}

	match, err := s.matches.FindCurrentForClub(ctx, club.ID)
	if err != nil {
		return nil, err
	}
	if match == nil {
		return nil, nil
	}
	return mapCurrentMatch(match), nil
}

func (s *matchService) GetByID(ctx context.Context, id uuid.UUID) (*dto.MatchDetailResponse, error) {
	match, err := s.matches.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if s.syncer != nil {
		if err := s.syncer.RefreshDetailsIfStale(ctx, match, 45*time.Second); err != nil {
			logger.Warn().Err(err).Str("match", match.ID.String()).Msg("on-demand match details sync failed")
		} else {
			refreshed, findErr := s.matches.FindByID(ctx, match.ID)
			if findErr == nil {
				match = refreshed
			}
		}
	}

	stats, err := s.details.ListStats(ctx, match.ID)
	if err != nil {
		return nil, err
	}
	timeline, err := s.details.ListTimeline(ctx, match.ID)
	if err != nil {
		return nil, err
	}
	lineup, err := s.details.ListLineup(ctx, match.ID)
	if err != nil {
		return nil, err
	}

	return mapMatchDetail(match, stats, timeline, lineup), nil
}

func (s *matchService) GetPlayer(ctx context.Context, id uuid.UUID) (*dto.PlayerDetailResponse, error) {
	player, err := s.players.FindByID(ctx, id)
	if err != nil {
		return nil, err
	}

	if s.provider != nil && s.provider.Enabled() && player.ProviderPlayerID != "" {
		season := strconv.Itoa(time.Now().UTC().Year())
		if profile, fetchErr := s.provider.GetPlayer(ctx, player.ProviderPlayerID, season); fetchErr == nil && profile != nil {
			applyPlayerProfile(player, profile)
			_ = s.players.Update(ctx, player)
		}
	}

	return mapPlayerDetail(player), nil
}

func (s *matchService) SearchProviderTeams(ctx context.Context, query string) ([]dto.ProviderTeam, error) {
	if s.provider == nil || !s.provider.Enabled() {
		return nil, errors.NewServiceUnavailable("Football data provider is not configured", nil)
	}
	teams, err := s.provider.SearchTeams(ctx, query)
	if err != nil {
		return nil, err
	}
	out := make([]dto.ProviderTeam, 0, len(teams))
	for _, team := range teams {
		item := dto.ProviderTeam{
			ProviderTeamID: team.ProviderID,
			Name:           team.Name,
			ShortName:      team.Code,
			Country:        team.Country,
			LogoURL:        team.LogoURL,
			VenueName:      team.VenueName,
		}
		if existing, findErr := s.clubs.FindByProviderTeamID(ctx, s.provider.Name(), team.ProviderID); findErr == nil && existing != nil {
			item.ClubID = &existing.ID
		}
		out = append(out, item)
	}
	return out, nil
}

func (s *matchService) GetFeaturedClub(ctx context.Context) (*dto.FeaturedClubResponse, error) {
	settings, err := s.settings.Get(ctx)
	if err != nil {
		return nil, err
	}
	resp := &dto.FeaturedClubResponse{ClubID: settings.FeaturedClubID}
	if settings.FeaturedClub != nil {
		resp.Name = settings.FeaturedClub.Name
		resp.LogoURL = settings.FeaturedClub.LogoURL
		resp.ProviderTeamID = settings.FeaturedClub.ProviderTeamID
		resp.Provider = settings.FeaturedClub.Provider
	}
	return resp, nil
}

func (s *matchService) SetFeaturedClub(ctx context.Context, req dto.SetFeaturedClubRequest) (*dto.FeaturedClubResponse, error) {
	if s.syncer == nil {
		return nil, errors.NewInternal("Match sync is not configured", nil)
	}

	var club *clubmodels.Club
	var err error
	switch {
	case req.ClubID != nil && *req.ClubID != uuid.Nil:
		club, err = s.clubs.FindByID(ctx, *req.ClubID)
	case strings.TrimSpace(req.ProviderTeamID) != "":
		club, err = s.syncer.EnsureClubFromProvider(ctx, strings.TrimSpace(req.ProviderTeamID))
	default:
		return nil, errors.NewBadRequest("club_id or provider_team_id is required", nil)
	}
	if err != nil {
		return nil, err
	}

	settings, err := s.settings.Get(ctx)
	if err != nil {
		return nil, err
	}
	settings.FeaturedClubID = &club.ID
	settings.FeaturedClub = club
	if err := s.settings.Save(ctx, settings); err != nil {
		return nil, err
	}

	if _, syncErr := s.syncer.SyncFeaturedClub(ctx); syncErr != nil {
		return nil, syncErr
	}

	return s.GetFeaturedClub(ctx)
}

func (s *matchService) SyncNow(ctx context.Context) (*dto.SyncResponse, error) {
	if s.syncer == nil {
		return nil, errors.NewInternal("Match sync is not configured", nil)
	}
	return s.syncer.SyncFeaturedClub(ctx)
}

func (s *matchService) featuredClub(ctx context.Context) (*clubmodels.Club, error) {
	settings, err := s.settings.Get(ctx)
	if err != nil {
		return nil, err
	}
	if settings.FeaturedClubID == nil {
		return nil, nil
	}
	if settings.FeaturedClub != nil {
		return settings.FeaturedClub, nil
	}
	return s.clubs.FindByID(ctx, *settings.FeaturedClubID)
}

func mapCurrentMatch(match *models.Match) *dto.CurrentMatchResponse {
	return &dto.CurrentMatchResponse{
		ID:        match.ID,
		Status:    match.Status,
		Minute:    match.CurrentMinute,
		Stadium:   match.StadiumName,
		KickoffAt: match.MatchDateTime.UTC().Format(time.RFC3339),
		HomeTeam:  teamSummary(match.HomeClub, match.HomeScore),
		AwayTeam:  teamSummary(match.AwayClub, match.AwayScore),
	}
}

func mapMatchDetail(
	match *models.Match,
	stats []models.MatchStat,
	timeline []models.MatchTimelineEvent,
	lineup []models.MatchLineupPlayer,
) *dto.MatchDetailResponse {
	homeScore := derefScore(match.HomeScore)
	awayScore := derefScore(match.AwayScore)
	resp := &dto.MatchDetailResponse{
		ID:                 match.ID,
		Status:             match.Status,
		HomeTeam:           teamSummary(match.HomeClub, match.HomeScore),
		AwayTeam:           teamSummary(match.AwayClub, match.AwayScore),
		Score:              fmt.Sprintf("%d : %d", homeScore, awayScore),
		Minute:             match.CurrentMinute,
		Stadium:            match.StadiumName,
		CompetitionLogoURL: match.CompetitionLogoURL,
		Stats:              make([]dto.StatItem, 0, len(stats)),
		Timeline:           make([]dto.TimelineItem, 0, len(timeline)),
		HomeSquads:         groupSquads(lineup, "home"),
		AwaySquads:         groupSquads(lineup, "away"),
	}
	if resp.CompetitionLogoURL == "" {
		resp.CompetitionLogoURL = match.League.LogoURL
	}
	for _, stat := range stats {
		resp.Stats = append(resp.Stats, dto.StatItem{
			Label: stat.Label,
			Home:  stat.HomeValue,
			Away:  stat.AwayValue,
		})
	}
	for _, item := range timeline {
		entry := dto.TimelineItem{
			Kind:        item.Kind,
			Minute:      item.Minute,
			Score:       item.Score,
			Side:        item.Side,
			Type:        item.EventType,
			Name:        item.PlayerName,
			Sub:         item.SubPlayerName,
			Highlighted: item.Highlighted,
		}
		resp.Timeline = append(resp.Timeline, entry)
	}
	return resp
}

func groupSquads(players []models.MatchLineupPlayer, side string) []dto.SquadGroup {
	order := []string{"Forward", "Midfielder", "Defender", "Goaler"}
	buckets := map[string][]dto.SquadPlayer{}
	for _, player := range players {
		if player.Side != side {
			continue
		}
		title := football.MapPosition(player.Position)
		buckets[title] = append(buckets[title], dto.SquadPlayer{
			ID:       player.PlayerID,
			Name:     player.Name,
			Position: title,
			PhotoURL: player.PhotoURL,
		})
	}
	groups := make([]dto.SquadGroup, 0, len(order))
	for _, title := range order {
		if len(buckets[title]) == 0 {
			continue
		}
		groups = append(groups, dto.SquadGroup{Title: title, Players: buckets[title]})
	}
	return groups
}

func mapPlayerDetail(player *playermodels.Player) *dto.PlayerDetailResponse {
	radar := make([]dto.RadarStat, 0, len(player.RadarStats))
	for _, item := range player.RadarStats {
		radar = append(radar, dto.RadarStat{Label: item.Label, Value: item.Value})
	}
	if len(radar) == 0 {
		radar = []dto.RadarStat{
			{Label: "Attack", Value: 50},
			{Label: "Skill", Value: 50},
			{Label: "Defence", Value: 50},
			{Label: "Tactic", Value: 50},
			{Label: "Creativity", Value: 50},
		}
	}
	return &dto.PlayerDetailResponse{
		ID:                 player.ID,
		Name:               player.Name,
		JerseyNumber:       player.JerseyNumber,
		Club:               player.Club.Name,
		ClubLogoURL:        player.Club.LogoURL,
		Age:                player.Age,
		PreferredFoot:      player.PreferredFoot,
		Nationality:        player.Nationality,
		HeightCM:           player.HeightCM,
		WeightKG:           player.WeightKG,
		WeakFootPercentage: player.WeakFootPercentage,
		PhotoURL:           player.PhotoURL,
		RadarStats:         radar,
		Formation:          player.Formation,
	}
}

func applyPlayerProfile(player *playermodels.Player, profile *football.PlayerProfile) {
	if profile.Name != "" {
		player.Name = profile.Name
	}
	if profile.Number > 0 {
		player.JerseyNumber = profile.Number
	}
	if profile.Position != "" {
		player.Position = profile.Position
	}
	if profile.Age > 0 {
		player.Age = profile.Age
	}
	if profile.PreferredFoot != "" {
		player.PreferredFoot = profile.PreferredFoot
	}
	if profile.Nationality != "" {
		player.Nationality = profile.Nationality
	}
	if profile.HeightCM > 0 {
		player.HeightCM = profile.HeightCM
	}
	if profile.WeightKG > 0 {
		player.WeightKG = profile.WeightKG
	}
	if profile.PhotoURL != "" {
		player.PhotoURL = profile.PhotoURL
	}
	if len(profile.Radar) > 0 {
		radar := make([]playermodels.RadarStat, 0, len(profile.Radar))
		for _, item := range profile.Radar {
			radar = append(radar, playermodels.RadarStat{Label: item.Label, Value: item.Value})
		}
		player.RadarStats = radar
	}
}

func teamSummary(club clubmodels.Club, score *int) dto.TeamSummary {
	return dto.TeamSummary{
		Name:    club.Name,
		LogoURL: club.LogoURL,
		Score:   score,
	}
}

func derefScore(score *int) int {
	if score == nil {
		return 0
	}
	return *score
}
