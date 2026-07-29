package service

import (
	"context"
	"encoding/json"

	clubrepo "clap/internal/modules/club/repository"
	"clap/internal/modules/stats/dto"
	"clap/internal/modules/stats/repository"
	"clap/internal/shared/logger"

	"github.com/google/uuid"
)

// PlayerService implements the Players screen (contract §9.2).
type PlayerService interface {
	// PlayerDetail returns the player profile. matchID is optional and only
	// used for request correlation (the on-pitch highlight is client-side).
	PlayerDetail(ctx context.Context, playerID uuid.UUID, matchID *uuid.UUID) (*dto.PlayerDetailResponse, error)
}

type playerService struct {
	playerRepo repository.PlayerRepository
	clubRepo   clubrepo.ClubRepository
}

func NewPlayerService(playerRepo repository.PlayerRepository, clubRepo clubrepo.ClubRepository) PlayerService {
	return &playerService{playerRepo: playerRepo, clubRepo: clubRepo}
}

func (s *playerService) PlayerDetail(ctx context.Context, playerID uuid.UUID, matchID *uuid.UUID) (*dto.PlayerDetailResponse, error) {
	player, err := s.playerRepo.FindByID(ctx, playerID)
	if err != nil {
		return nil, err
	}

	clubName := ""
	if club, clubErr := s.clubRepo.FindByID(ctx, player.ClubID); clubErr == nil {
		clubName = club.Name
	}

	var radar []dto.RadarStat
	if err := json.Unmarshal([]byte(player.RadarStats), &radar); err != nil || radar == nil {
		radar = []dto.RadarStat{}
	}

	evt := logger.Info().Str("player_id", playerID.String())
	if matchID != nil {
		evt = evt.Str("match_id", matchID.String())
	}
	evt.Msg("player_detail_requested")

	return &dto.PlayerDetailResponse{
		ID:                 player.ID,
		Name:               player.Name,
		JerseyNumber:       player.JerseyNumber,
		Club:               clubName,
		Age:                player.Age,
		PreferredFoot:      player.PreferredFoot,
		Nationality:        player.Nationality,
		HeightCm:           player.HeightCm,
		WeightKg:           player.WeightKg,
		WeakFootPercentage: player.WeakFootPercentage,
		PhotoURL:           player.PhotoURL,
		RadarStats:         radar,
		Formation:          player.Formation,
	}, nil
}
