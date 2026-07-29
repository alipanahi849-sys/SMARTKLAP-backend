package service

import (
	"context"

	clubrepo "clap/internal/modules/club/repository"
	"clap/internal/modules/match/dto"
	"clap/internal/modules/match/models"
	statsmodels "clap/internal/modules/stats/models"
	statsrepo "clap/internal/modules/stats/repository"

	"github.com/google/uuid"
)

// MatchEventPublisher delivers match state changes to connected clients.
// The production implementation is the WebSocket realtime gateway; nil
// disables publishing (tests, admin-only deployments).
type MatchEventPublisher interface {
	PublishMatchEvent(ctx context.Context, matchID uuid.UUID, eventType string, payload any) error
}

// detailDeps holds the optional dependencies used to enrich GET /matches/:id
// with the mobile statistics payload (Mobile API Contract §9.1).
type detailDeps struct {
	clubRepo   clubrepo.ClubRepository
	statsRepo  statsrepo.MatchStatsRepository
	playerRepo statsrepo.PlayerRepository
}

// enrichDetail fills the mobile-only fields of MatchResponse. Any lookup
// failure degrades gracefully: the base match payload is always returned.
func (s *matchService) enrichDetail(ctx context.Context, match *models.Match, resp *dto.MatchResponse) {
	if s.detail == nil {
		return
	}

	resp.HomeScore = match.HomeScore
	resp.AwayScore = match.AwayScore
	resp.Minute = match.CurrentMinute

	if home, err := s.detail.clubRepo.FindByID(ctx, match.HomeClubID); err == nil {
		resp.HomeTeam = &dto.MatchTeamInfo{ID: home.ID, Name: home.Name, LogoURL: home.LogoURL}
	}
	if away, err := s.detail.clubRepo.FindByID(ctx, match.AwayClubID); err == nil {
		resp.AwayTeam = &dto.MatchTeamInfo{ID: away.ID, Name: away.Name, LogoURL: away.LogoURL}
	}

	if stats, err := s.detail.statsRepo.StatsByMatch(ctx, match.ID); err == nil {
		rows := make([]dto.MatchStatRow, len(stats))
		for i, st := range stats {
			rows[i] = dto.MatchStatRow{Label: st.Label, Home: st.HomeValue, Away: st.AwayValue}
		}
		resp.Stats = rows
	}

	if events, err := s.detail.statsRepo.TimelineByMatch(ctx, match.ID); err == nil {
		items := make([]dto.MatchTimelineItem, len(events))
		for i, ev := range events {
			items[i] = dto.MatchTimelineItem{
				Kind:        ev.Kind,
				Side:        ev.Side,
				EventType:   ev.EventType,
				PlayerName:  ev.PlayerName,
				Minute:      ev.Minute,
				Score:       ev.Score,
				Highlighted: ev.Highlighted,
			}
		}
		resp.Timeline = items
	}

	if players, err := s.detail.playerRepo.FindByClubIDs(ctx, []uuid.UUID{match.HomeClubID, match.AwayClubID}); err == nil && len(players) > 0 {
		resp.Squads = buildSquadGroups(players)
	}
}

// buildSquadGroups groups players by position, preserving the repository's
// position/jersey ordering.
func buildSquadGroups(players []statsmodels.Player) []dto.SquadGroup {
	var groups []dto.SquadGroup
	index := map[string]int{}
	for _, p := range players {
		title := p.Position
		if title == "" {
			title = "Squad"
		}
		i, ok := index[title]
		if !ok {
			groups = append(groups, dto.SquadGroup{Title: title})
			i = len(groups) - 1
			index[title] = i
		}
		groups[i].Players = append(groups[i].Players, dto.SquadPlayer{
			ID:           p.ID,
			Name:         p.Name,
			JerseyNumber: p.JerseyNumber,
			Position:     p.Position,
			PhotoURL:     p.PhotoURL,
		})
	}
	return groups
}
