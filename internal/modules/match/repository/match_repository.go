package repository

import (
	"context"

	"clap/internal/modules/match/models"
	sharederrors "clap/internal/shared/errors"
	"clap/internal/shared/utils"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type MatchRepository interface {
	Create(ctx context.Context, match *models.Match) error
	FindByID(ctx context.Context, id uuid.UUID) (*models.Match, error)
	FindAll(ctx context.Context, page, pageSize int, filters map[string]string, sortBy, sortOrder string) ([]models.Match, int64, error)
	FindBySeason(ctx context.Context, seasonID uuid.UUID, page, pageSize int) ([]models.Match, int64, error)
	FindByLeague(ctx context.Context, leagueID uuid.UUID, page, pageSize int) ([]models.Match, int64, error)
	FindByClub(ctx context.Context, clubID uuid.UUID, page, pageSize int) ([]models.Match, int64, error)
	FindUpcoming(ctx context.Context, page, pageSize int) ([]models.Match, int64, error)
	FindLive(ctx context.Context) ([]models.Match, error)
	FindByProviderMatchID(ctx context.Context, provider, providerMatchID string) (*models.Match, error)
	FindCurrentForClub(ctx context.Context, clubID uuid.UUID) (*models.Match, error)
	ListForClub(ctx context.Context, clubID uuid.UUID, limit int) ([]models.Match, error)
	FindLiveByClub(ctx context.Context, clubID uuid.UUID) ([]models.Match, error)
	Update(ctx context.Context, match *models.Match) error
	Delete(ctx context.Context, id uuid.UUID) error
}

type matchRepository struct {
	db *gorm.DB
}

func NewMatchRepository(db *gorm.DB) MatchRepository {
	return &matchRepository{db: db}
}

func (r *matchRepository) Create(ctx context.Context, match *models.Match) error {
	if err := r.db.WithContext(ctx).Create(match).Error; err != nil {
		return sharederrors.NewInternal("Failed to create match", err)
	}
	return nil
}

func (r *matchRepository) FindByID(ctx context.Context, id uuid.UUID) (*models.Match, error) {
	var match models.Match
	if err := r.db.WithContext(ctx).Preload("League").Preload("Season").Preload("HomeClub").Preload("AwayClub").Where("id = ?", id).First(&match).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, sharederrors.NewNotFound("Match not found", nil)
		}
		return nil, sharederrors.NewInternal("Failed to find match", err)
	}
	return &match, nil
}

func (r *matchRepository) FindAll(ctx context.Context, page, pageSize int, filters map[string]string, sortBy, sortOrder string) ([]models.Match, int64, error) {
	var matches []models.Match
	var total int64

	query := r.db.WithContext(ctx).Model(&models.Match{}).Preload("League").Preload("Season").Preload("HomeClub").Preload("AwayClub")

	// Apply filters
	if seasonID, ok := filters["season_id"]; ok {
		if id, err := uuid.Parse(seasonID); err == nil {
			query = query.Where("season_id = ?", id)
		}
	}

	if leagueID, ok := filters["league_id"]; ok {
		if id, err := uuid.Parse(leagueID); err == nil {
			query = query.Where("league_id = ?", id)
		}
	}

	if clubID, ok := filters["club_id"]; ok {
		if id, err := uuid.Parse(clubID); err == nil {
			query = query.Where("home_club_id = ? OR away_club_id = ?", id, id)
		}
	}

	if status, ok := filters["status"]; ok && status != "" {
		query = query.Where("status = ?", status)
	}

	// Count total
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, sharederrors.NewInternal("Failed to count matches", err)
	}

	// Apply sorting
	allowedSortFields := map[string]bool{
		"match_datetime": true,
		"created_at":     true,
		"updated_at":     true,
	}
	if !allowedSortFields[sortBy] {
		sortBy = "match_datetime"
	}
	if sortOrder != "ASC" && sortOrder != "DESC" {
		sortOrder = "ASC"
	}
	query = query.Order(sortBy + " " + sortOrder)

	// Apply pagination
	offset := utils.GetOffset(page, pageSize)
	if err := query.Offset(offset).Limit(pageSize).Find(&matches).Error; err != nil {
		return nil, 0, sharederrors.NewInternal("Failed to find matches", err)
	}

	return matches, total, nil
}

func (r *matchRepository) FindBySeason(ctx context.Context, seasonID uuid.UUID, page, pageSize int) ([]models.Match, int64, error) {
	var matches []models.Match
	var total int64

	query := r.db.WithContext(ctx).Model(&models.Match{}).Where("season_id = ?", seasonID).Preload("League").Preload("Season").Preload("HomeClub").Preload("AwayClub")

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, sharederrors.NewInternal("Failed to count matches", err)
	}

	offset := utils.GetOffset(page, pageSize)
	if err := query.Offset(offset).Limit(pageSize).Order("match_datetime ASC").Find(&matches).Error; err != nil {
		return nil, 0, sharederrors.NewInternal("Failed to find matches", err)
	}

	return matches, total, nil
}

func (r *matchRepository) FindByLeague(ctx context.Context, leagueID uuid.UUID, page, pageSize int) ([]models.Match, int64, error) {
	var matches []models.Match
	var total int64

	query := r.db.WithContext(ctx).Model(&models.Match{}).Where("league_id = ?", leagueID).Preload("League").Preload("Season").Preload("HomeClub").Preload("AwayClub")

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, sharederrors.NewInternal("Failed to count matches", err)
	}

	offset := utils.GetOffset(page, pageSize)
	if err := query.Offset(offset).Limit(pageSize).Order("match_datetime DESC").Find(&matches).Error; err != nil {
		return nil, 0, sharederrors.NewInternal("Failed to find matches", err)
	}

	return matches, total, nil
}

func (r *matchRepository) FindByClub(ctx context.Context, clubID uuid.UUID, page, pageSize int) ([]models.Match, int64, error) {
	var matches []models.Match
	var total int64

	query := r.db.WithContext(ctx).Model(&models.Match{}).Where("home_club_id = ? OR away_club_id = ?", clubID, clubID).Preload("League").Preload("Season").Preload("HomeClub").Preload("AwayClub")

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, sharederrors.NewInternal("Failed to count matches", err)
	}

	offset := utils.GetOffset(page, pageSize)
	if err := query.Offset(offset).Limit(pageSize).Order("match_datetime DESC").Find(&matches).Error; err != nil {
		return nil, 0, sharederrors.NewInternal("Failed to find matches", err)
	}

	return matches, total, nil
}

func (r *matchRepository) FindUpcoming(ctx context.Context, page, pageSize int) ([]models.Match, int64, error) {
	var matches []models.Match
	var total int64

	query := r.db.WithContext(ctx).Model(&models.Match{}).Where("status = 'scheduled' AND match_datetime > NOW()").Preload("League").Preload("Season").Preload("HomeClub").Preload("AwayClub")

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, sharederrors.NewInternal("Failed to count matches", err)
	}

	offset := utils.GetOffset(page, pageSize)
	if err := query.Offset(offset).Limit(pageSize).Order("match_datetime ASC").Find(&matches).Error; err != nil {
		return nil, 0, sharederrors.NewInternal("Failed to find matches", err)
	}

	return matches, total, nil
}

func (r *matchRepository) FindLive(ctx context.Context) ([]models.Match, error) {
	var matches []models.Match

	if err := r.db.WithContext(ctx).Model(&models.Match{}).Where("status IN ('live', 'halftime')").Preload("League").Preload("Season").Preload("HomeClub").Preload("AwayClub").Order("match_datetime ASC").Find(&matches).Error; err != nil {
		return nil, sharederrors.NewInternal("Failed to find live matches", err)
	}

	return matches, nil
}

func (r *matchRepository) FindByProviderMatchID(ctx context.Context, provider, providerMatchID string) (*models.Match, error) {
	var match models.Match
	if err := r.db.WithContext(ctx).
		Preload("League").Preload("Season").Preload("HomeClub").Preload("AwayClub").
		Where("provider = ? AND provider_match_id = ?", provider, providerMatchID).
		First(&match).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, sharederrors.NewInternal("Failed to find match by provider ID", err)
	}
	return &match, nil
}

func (r *matchRepository) FindCurrentForClub(ctx context.Context, clubID uuid.UUID) (*models.Match, error) {
	var match models.Match
	err := r.db.WithContext(ctx).
		Preload("League").Preload("Season").Preload("HomeClub").Preload("AwayClub").
		Where("home_club_id = ? OR away_club_id = ?", clubID, clubID).
		Order(`
			CASE
				WHEN status IN ('live', 'halftime') THEN 0
				WHEN status = 'scheduled' AND match_datetime >= NOW() THEN 1
				WHEN status = 'finished' THEN 2
				ELSE 3
			END,
			CASE WHEN status = 'scheduled' THEN match_datetime END ASC,
			match_datetime DESC
		`).
		First(&match).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, nil
		}
		return nil, sharederrors.NewInternal("Failed to find current match", err)
	}
	return &match, nil
}

func (r *matchRepository) ListForClub(ctx context.Context, clubID uuid.UUID, limit int) ([]models.Match, error) {
	if limit <= 0 {
		limit = 8
	}
	var matches []models.Match
	err := r.db.WithContext(ctx).
		Preload("League").Preload("Season").Preload("HomeClub").Preload("AwayClub").
		Where("(home_club_id = ? OR away_club_id = ?) AND status <> ?", clubID, clubID, "cancelled").
		Order(`
			CASE
				WHEN status IN ('live', 'halftime') THEN 0
				WHEN status = 'scheduled' AND match_datetime >= NOW() THEN 1
				WHEN status = 'finished' THEN 2
				ELSE 3
			END,
			CASE WHEN status = 'scheduled' THEN match_datetime END ASC,
			match_datetime DESC
		`).
		Limit(limit).
		Find(&matches).Error
	if err != nil {
		return nil, sharederrors.NewInternal("Failed to list club matches", err)
	}
	return matches, nil
}

func (r *matchRepository) FindLiveByClub(ctx context.Context, clubID uuid.UUID) ([]models.Match, error) {
	var matches []models.Match
	if err := r.db.WithContext(ctx).
		Preload("League").Preload("Season").Preload("HomeClub").Preload("AwayClub").
		Where("(home_club_id = ? OR away_club_id = ?) AND status IN ('live', 'halftime')", clubID, clubID).
		Order("match_datetime ASC").
		Find(&matches).Error; err != nil {
		return nil, sharederrors.NewInternal("Failed to find live matches", err)
	}
	return matches, nil
}

func (r *matchRepository) Update(ctx context.Context, match *models.Match) error {
	if err := r.db.WithContext(ctx).Save(match).Error; err != nil {
		return sharederrors.NewInternal("Failed to update match", err)
	}
	return nil
}

func (r *matchRepository) Delete(ctx context.Context, id uuid.UUID) error {
	if err := r.db.WithContext(ctx).Delete(&models.Match{}, id).Error; err != nil {
		return sharederrors.NewInternal("Failed to delete match", err)
	}
	return nil
}
