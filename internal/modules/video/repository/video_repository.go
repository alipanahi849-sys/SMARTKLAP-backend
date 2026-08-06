package repository

import (
	"context"
	"time"

	"clap/internal/modules/video/models"
	"clap/internal/shared/errors"

	"github.com/google/uuid"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

// VideoCursorAnchor is the list position after which the next page is fetched.
type VideoCursorAnchor struct {
	CreatedAt time.Time
	ID        uuid.UUID
}

type VideoRepository interface {
	Create(ctx context.Context, video *models.Video) error
	FindByID(ctx context.Context, id uuid.UUID) (*models.Video, error)
	// FeedAfter lists published videos newest-first with the author preloaded.
	FeedAfter(ctx context.Context, limit int, after *VideoCursorAnchor) ([]models.Video, error)
	// ByUserAfter lists a user's own videos (any status) newest-first.
	ByUserAfter(ctx context.Context, userID uuid.UUID, limit int, after *VideoCursorAnchor) ([]models.Video, error)
	// LikedVideoIDs returns which of the given videos the user has liked.
	LikedVideoIDs(ctx context.Context, userID uuid.UUID, videoIDs []uuid.UUID) (map[uuid.UUID]bool, error)
	// Like inserts a like and bumps the denormalized counter atomically.
	// Returns false when the video was already liked (idempotent).
	Like(ctx context.Context, videoID, userID uuid.UUID) (bool, error)
	// Unlike removes a like and decrements the counter. Returns false when no
	// like existed (idempotent).
	Unlike(ctx context.Context, videoID, userID uuid.UUID) (bool, error)
	// SeenVideoIDs returns which of the given videos the user has seen.
	SeenVideoIDs(ctx context.Context, userID uuid.UUID, videoIDs []uuid.UUID) (map[uuid.UUID]bool, error)
	// MarkSeen records a view and bumps the denormalized counter atomically.
	// Returns false when the video was already seen (idempotent).
	MarkSeen(ctx context.Context, videoID, userID uuid.UUID) (bool, error)
}

type videoRepository struct {
	db *gorm.DB
}

func NewVideoRepository(db *gorm.DB) VideoRepository {
	return &videoRepository{db: db}
}

func (r *videoRepository) Create(ctx context.Context, video *models.Video) error {
	if err := r.db.WithContext(ctx).Create(video).Error; err != nil {
		return errors.NewInternal("Failed to create video", err)
	}
	return nil
}

func (r *videoRepository) FindByID(ctx context.Context, id uuid.UUID) (*models.Video, error) {
	var video models.Video
	err := r.db.WithContext(ctx).Preload("User").First(&video, "id = ?", id).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.NewNotFound("Video not found", nil)
		}
		return nil, errors.NewInternal("Failed to find video", err)
	}
	return &video, nil
}

func (r *videoRepository) FeedAfter(ctx context.Context, limit int, after *VideoCursorAnchor) ([]models.Video, error) {
	q := r.db.WithContext(ctx).Model(&models.Video{}).
		Where("status = ?", models.StatusPublished)
	if after != nil {
		q = q.Where(
			"(created_at < ?) OR (created_at = ? AND id < ?)",
			after.CreatedAt, after.CreatedAt, after.ID,
		)
	}

	var videos []models.Video
	if err := q.Preload("User").
		Order("created_at DESC, id DESC").
		Limit(limit).
		Find(&videos).Error; err != nil {
		return nil, errors.NewInternal("Failed to load feed", err)
	}
	return videos, nil
}

func (r *videoRepository) ByUserAfter(ctx context.Context, userID uuid.UUID, limit int, after *VideoCursorAnchor) ([]models.Video, error) {
	q := r.db.WithContext(ctx).Model(&models.Video{}).Where("user_id = ?", userID)
	if after != nil {
		q = q.Where(
			"(created_at < ?) OR (created_at = ? AND id < ?)",
			after.CreatedAt, after.CreatedAt, after.ID,
		)
	}

	var videos []models.Video
	if err := q.Preload("User").
		Order("created_at DESC, id DESC").
		Limit(limit).
		Find(&videos).Error; err != nil {
		return nil, errors.NewInternal("Failed to load videos", err)
	}
	return videos, nil
}

func (r *videoRepository) LikedVideoIDs(ctx context.Context, userID uuid.UUID, videoIDs []uuid.UUID) (map[uuid.UUID]bool, error) {
	liked := make(map[uuid.UUID]bool, len(videoIDs))
	if len(videoIDs) == 0 {
		return liked, nil
	}

	var ids []uuid.UUID
	if err := r.db.WithContext(ctx).Model(&models.VideoLike{}).
		Where("user_id = ? AND video_id IN ?", userID, videoIDs).
		Pluck("video_id", &ids).Error; err != nil {
		return nil, errors.NewInternal("Failed to load likes", err)
	}
	for _, id := range ids {
		liked[id] = true
	}
	return liked, nil
}

func (r *videoRepository) Like(ctx context.Context, videoID, userID uuid.UUID) (bool, error) {
	var created bool
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		like := models.VideoLike{VideoID: videoID, UserID: userID}
		res := tx.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "video_id"}, {Name: "user_id"}},
			DoNothing: true,
		}).Create(&like)
		if res.Error != nil {
			return errors.NewInternal("Failed to like video", res.Error)
		}
		if res.RowsAffected == 0 {
			return nil // already liked — idempotent
		}
		created = true
		if err := tx.Model(&models.Video{}).
			Where("id = ?", videoID).
			UpdateColumn("likes_count", gorm.Expr("likes_count + 1")).Error; err != nil {
			return errors.NewInternal("Failed to update like counter", err)
		}
		return nil
	})
	return created, err
}

func (r *videoRepository) Unlike(ctx context.Context, videoID, userID uuid.UUID) (bool, error) {
	var removed bool
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		res := tx.Where("video_id = ? AND user_id = ?", videoID, userID).Delete(&models.VideoLike{})
		if res.Error != nil {
			return errors.NewInternal("Failed to unlike video", res.Error)
		}
		if res.RowsAffected == 0 {
			return nil // not liked — idempotent
		}
		removed = true
		if err := tx.Model(&models.Video{}).
			Where("id = ? AND likes_count > 0", videoID).
			UpdateColumn("likes_count", gorm.Expr("likes_count - 1")).Error; err != nil {
			return errors.NewInternal("Failed to update like counter", err)
		}
		return nil
	})
	return removed, err
}

func (r *videoRepository) SeenVideoIDs(ctx context.Context, userID uuid.UUID, videoIDs []uuid.UUID) (map[uuid.UUID]bool, error) {
	seen := make(map[uuid.UUID]bool, len(videoIDs))
	if len(videoIDs) == 0 {
		return seen, nil
	}

	var ids []uuid.UUID
	if err := r.db.WithContext(ctx).Model(&models.VideoView{}).
		Where("user_id = ? AND video_id IN ?", userID, videoIDs).
		Pluck("video_id", &ids).Error; err != nil {
		return nil, errors.NewInternal("Failed to load views", err)
	}
	for _, id := range ids {
		seen[id] = true
	}
	return seen, nil
}

func (r *videoRepository) MarkSeen(ctx context.Context, videoID, userID uuid.UUID) (bool, error) {
	var created bool
	err := r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		view := models.VideoView{VideoID: videoID, UserID: userID}
		res := tx.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "video_id"}, {Name: "user_id"}},
			DoNothing: true,
		}).Create(&view)
		if res.Error != nil {
			return errors.NewInternal("Failed to mark video as seen", res.Error)
		}
		if res.RowsAffected == 0 {
			return nil // already seen — idempotent
		}
		created = true
		if err := tx.Model(&models.Video{}).
			Where("id = ?", videoID).
			UpdateColumn("views_count", gorm.Expr("views_count + 1")).Error; err != nil {
			return errors.NewInternal("Failed to update view counter", err)
		}
		return nil
	})
	return created, err
}
