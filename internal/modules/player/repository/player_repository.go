package repository

import (
	"context"

	"clap/internal/modules/player/models"
	sharederrors "clap/internal/shared/errors"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type PlayerRepository interface {
	Create(ctx context.Context, player *models.Player) error
	FindByID(ctx context.Context, id uuid.UUID) (*models.Player, error)
	FindByProviderPlayerID(ctx context.Context, provider, providerPlayerID string) (*models.Player, error)
	Update(ctx context.Context, player *models.Player) error
}

type playerRepository struct {
	db *gorm.DB
}

func NewPlayerRepository(db *gorm.DB) PlayerRepository {
	return &playerRepository{db: db}
}

func (r *playerRepository) Create(ctx context.Context, player *models.Player) error {
	if err := r.db.WithContext(ctx).Create(player).Error; err != nil {
		return sharederrors.NewInternal("Failed to create player", err)
	}
	return nil
}

func (r *playerRepository) FindByID(ctx context.Context, id uuid.UUID) (*models.Player, error) {
	var player models.Player
	if err := r.db.WithContext(ctx).Preload("Club").Where("id = ?", id).First(&player).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, sharederrors.NewNotFound("Player not found", nil)
		}
		return nil, sharederrors.NewInternal("Failed to find player", err)
	}
	return &player, nil
}

func (r *playerRepository) FindByProviderPlayerID(ctx context.Context, provider, providerPlayerID string) (*models.Player, error) {
	var player models.Player
	if err := r.db.WithContext(ctx).Preload("Club").Where("provider = ? AND provider_player_id = ?", provider, providerPlayerID).First(&player).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, sharederrors.NewInternal("Failed to find player by provider ID", err)
	}
	return &player, nil
}

func (r *playerRepository) Update(ctx context.Context, player *models.Player) error {
	if err := r.db.WithContext(ctx).Save(player).Error; err != nil {
		return sharederrors.NewInternal("Failed to update player", err)
	}
	return nil
}
