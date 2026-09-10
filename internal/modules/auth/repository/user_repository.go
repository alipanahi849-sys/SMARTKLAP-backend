package repository

import (
	"context"
	"strings"
	"time"

	"clap/internal/modules/auth/models"
	"clap/internal/shared/database"
	"clap/internal/shared/errors"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// LeaderboardCursorAnchor is the list position after which the next page is fetched.
type LeaderboardCursorAnchor struct {
	Points    int
	CreatedAt time.Time
	ID        uuid.UUID
}

type UserRepository interface {
	Create(ctx context.Context, user *models.User) error
	FindByID(ctx context.Context, id uuid.UUID) (*models.User, error)
	FindByEmail(ctx context.Context, email string) (*models.User, error)
	FindByGoogleID(ctx context.Context, googleID string) (*models.User, error)
	FindByAppleID(ctx context.Context, appleID string) (*models.User, error)
	Update(ctx context.Context, user *models.User) error
	Delete(ctx context.Context, id uuid.UUID) error
	List(ctx context.Context, offset, limit int) ([]models.User, int64, error)
	// ListFiltered returns users matching optional search / active filters.
	ListFiltered(ctx context.Context, opts UserListOptions) ([]models.User, int64, error)
	AddRole(ctx context.Context, userID, roleID uuid.UUID) error
	RemoveRole(ctx context.Context, userID, roleID uuid.UUID) error
	ReplaceRoles(ctx context.Context, userID uuid.UUID, roleIDs []uuid.UUID) error
	GetUserRoles(ctx context.Context, userID uuid.UUID) ([]models.Role, error)
	// AddPoints atomically increments (or decrements) a user's points balance
	// and returns the new balance.
	AddPoints(ctx context.Context, userID uuid.UUID, delta int) (int, error)
	// SpendPoints deducts points when the user has sufficient balance.
	SpendPoints(ctx context.Context, userID uuid.UUID, amount int) (int, error)
	// CountActive returns the number of active users (leaderboard total).
	CountActive(ctx context.Context) (int64, error)
	// CountWithMorePoints returns how many active users outrank the given
	// points balance; rank position = result + 1.
	CountWithMorePoints(ctx context.Context, points int) (int64, error)
	// TopByPointsAfter returns active users ordered by points (desc), created_at (asc).
	// Pass after=nil for the first page.
	TopByPointsAfter(ctx context.Context, limit int, after *LeaderboardCursorAnchor) ([]models.User, error)
	// LeaderboardRank returns a user's 1-based position on the leaderboard.
	LeaderboardRank(ctx context.Context, points int, createdAt time.Time, id uuid.UUID) (int, error)
}

// UserListOptions filters admin user listing.
type UserListOptions struct {
	Query    string
	IsActive *bool
	Role     string
	Offset   int
	Limit    int
}

type userRepository struct {
	db *gorm.DB
}

func NewUserRepository() UserRepository {
	return &userRepository{
		db: database.GetDB(),
	}
}

func (r *userRepository) Create(ctx context.Context, user *models.User) error {
	if err := r.db.WithContext(ctx).Create(user).Error; err != nil {
		return errors.NewInternal("Failed to create user", err)
	}
	return nil
}

func (r *userRepository) FindByID(ctx context.Context, id uuid.UUID) (*models.User, error) {
	var user models.User
	err := r.db.WithContext(ctx).Preload("Roles").First(&user, "id = ?", id).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.ErrUserNotFound
		}
		return nil, errors.NewInternal("Failed to find user", err)
	}
	return &user, nil
}

func (r *userRepository) FindByEmail(ctx context.Context, email string) (*models.User, error) {
	var user models.User
	err := r.db.WithContext(ctx).Preload("Roles").First(&user, "email = ?", email).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.ErrUserNotFound
		}
		return nil, errors.NewInternal("Failed to find user", err)
	}
	return &user, nil
}

func (r *userRepository) FindByGoogleID(ctx context.Context, googleID string) (*models.User, error) {
	var user models.User
	err := r.db.WithContext(ctx).Preload("Roles").First(&user, "google_id = ?", googleID).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.ErrUserNotFound
		}
		return nil, errors.NewInternal("Failed to find user", err)
	}
	return &user, nil
}

func (r *userRepository) FindByAppleID(ctx context.Context, appleID string) (*models.User, error) {
	var user models.User
	err := r.db.WithContext(ctx).Preload("Roles").First(&user, "apple_id = ?", appleID).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.ErrUserNotFound
		}
		return nil, errors.NewInternal("Failed to find user", err)
	}
	return &user, nil
}

func (r *userRepository) Update(ctx context.Context, user *models.User) error {
	if err := r.db.WithContext(ctx).Save(user).Error; err != nil {
		return errors.NewInternal("Failed to update user", err)
	}
	return nil
}

func (r *userRepository) Delete(ctx context.Context, id uuid.UUID) error {
	if err := r.db.WithContext(ctx).Delete(&models.User{}, "id = ?", id).Error; err != nil {
		return errors.NewInternal("Failed to delete user", err)
	}
	return nil
}

func (r *userRepository) List(ctx context.Context, offset, limit int) ([]models.User, int64, error) {
	return r.ListFiltered(ctx, UserListOptions{Offset: offset, Limit: limit})
}

func (r *userRepository) ListFiltered(ctx context.Context, opts UserListOptions) ([]models.User, int64, error) {
	limit := opts.Limit
	if limit <= 0 {
		limit = 20
	}
	if limit > 100 {
		limit = 100
	}
	offset := opts.Offset
	if offset < 0 {
		offset = 0
	}

	q := r.db.WithContext(ctx).Model(&models.User{})
	if query := strings.TrimSpace(opts.Query); query != "" {
		like := "%" + strings.ToLower(query) + "%"
		q = q.Where(
			"LOWER(email) LIKE ? OR LOWER(first_name) LIKE ? OR LOWER(last_name) LIKE ? OR LOWER(CONCAT(COALESCE(first_name,''), ' ', COALESCE(last_name,''))) LIKE ?",
			like, like, like, like,
		)
	}
	if opts.IsActive != nil {
		q = q.Where("is_active = ?", *opts.IsActive)
	}
	if role := strings.TrimSpace(opts.Role); role != "" {
		q = q.Where(
			"EXISTS (SELECT 1 FROM user_roles ur JOIN roles ro ON ro.id = ur.role_id WHERE ur.user_id = users.id AND ro.name = ?)",
			role,
		)
	}

	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, errors.NewInternal("Failed to count users", err)
	}

	var users []models.User
	if err := q.Preload("Roles").
		Order("created_at DESC, id DESC").
		Offset(offset).
		Limit(limit).
		Find(&users).Error; err != nil {
		return nil, 0, errors.NewInternal("Failed to list users", err)
	}

	return users, total, nil
}

func (r *userRepository) AddRole(ctx context.Context, userID, roleID uuid.UUID) error {
	if err := r.db.WithContext(ctx).Exec("INSERT INTO user_roles (user_id, role_id) VALUES (?, ?)", userID, roleID).Error; err != nil {
		return errors.NewInternal("Failed to add role to user", err)
	}
	return nil
}

func (r *userRepository) RemoveRole(ctx context.Context, userID, roleID uuid.UUID) error {
	if err := r.db.WithContext(ctx).Exec("DELETE FROM user_roles WHERE user_id = ? AND role_id = ?", userID, roleID).Error; err != nil {
		return errors.NewInternal("Failed to remove role from user", err)
	}
	return nil
}

func (r *userRepository) ReplaceRoles(ctx context.Context, userID uuid.UUID, roleIDs []uuid.UUID) error {
	return r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec("DELETE FROM user_roles WHERE user_id = ?", userID).Error; err != nil {
			return errors.NewInternal("Failed to clear user roles", err)
		}
		for _, roleID := range roleIDs {
			if err := tx.Exec(
				"INSERT INTO user_roles (user_id, role_id) VALUES (?, ?) ON CONFLICT DO NOTHING",
				userID, roleID,
			).Error; err != nil {
				return errors.NewInternal("Failed to assign user role", err)
			}
		}
		return nil
	})
}

func (r *userRepository) AddPoints(ctx context.Context, userID uuid.UUID, delta int) (int, error) {
	res := r.db.WithContext(ctx).Model(&models.User{}).
		Where("id = ?", userID).
		UpdateColumn("points", gorm.Expr("points + ?", delta))
	if res.Error != nil {
		return 0, errors.NewInternal("Failed to update user points", res.Error)
	}
	if res.RowsAffected == 0 {
		return 0, errors.ErrUserNotFound
	}

	var points int
	if err := r.db.WithContext(ctx).Model(&models.User{}).
		Where("id = ?", userID).
		Pluck("points", &points).Error; err != nil {
		return 0, errors.NewInternal("Failed to read user points", err)
	}
	return points, nil
}

func (r *userRepository) SpendPoints(ctx context.Context, userID uuid.UUID, amount int) (int, error) {
	if amount <= 0 {
		return 0, errors.NewBadRequest("amount must be positive", nil)
	}

	res := r.db.WithContext(ctx).Model(&models.User{}).
		Where("id = ? AND points >= ?", userID, amount).
		UpdateColumn("points", gorm.Expr("points - ?", amount))
	if res.Error != nil {
		return 0, errors.NewInternal("Failed to spend points", res.Error)
	}
	if res.RowsAffected == 0 {
		var user models.User
		err := r.db.WithContext(ctx).Select("points").First(&user, "id = ?", userID).Error
		if err != nil {
			if err == gorm.ErrRecordNotFound {
				return 0, errors.ErrUserNotFound
			}
			return 0, errors.NewInternal("Failed to read user points", err)
		}
		return 0, errors.NewUnprocessable("Insufficient points balance", nil)
	}

	var points int
	if err := r.db.WithContext(ctx).Model(&models.User{}).
		Where("id = ?", userID).
		Pluck("points", &points).Error; err != nil {
		return 0, errors.NewInternal("Failed to read user points", err)
	}
	return points, nil
}

func (r *userRepository) CountActive(ctx context.Context) (int64, error) {
	var total int64
	if err := r.db.WithContext(ctx).Model(&models.User{}).
		Where("is_active = ?", true).
		Count(&total).Error; err != nil {
		return 0, errors.NewInternal("Failed to count users", err)
	}
	return total, nil
}

func (r *userRepository) CountWithMorePoints(ctx context.Context, points int) (int64, error) {
	var total int64
	if err := r.db.WithContext(ctx).Model(&models.User{}).
		Where("is_active = ? AND points > ?", true, points).
		Count(&total).Error; err != nil {
		return 0, errors.NewInternal("Failed to compute rank", err)
	}
	return total, nil
}

func (r *userRepository) TopByPointsAfter(ctx context.Context, limit int, after *LeaderboardCursorAnchor) ([]models.User, error) {
	q := r.db.WithContext(ctx).Where("is_active = ?", true)

	if after != nil {
		q = q.Where(
			"(points < ?) OR (points = ? AND created_at > ?) OR (points = ? AND created_at = ? AND id > ?)",
			after.Points, after.Points, after.CreatedAt, after.Points, after.CreatedAt, after.ID,
		)
	}

	var users []models.User
	if err := q.Order("points DESC, created_at ASC, id ASC").
		Limit(limit).
		Find(&users).Error; err != nil {
		return nil, errors.NewInternal("Failed to load leaderboard", err)
	}
	return users, nil
}

func (r *userRepository) LeaderboardRank(ctx context.Context, points int, createdAt time.Time, id uuid.UUID) (int, error) {
	higher, err := r.CountWithMorePoints(ctx, points)
	if err != nil {
		return 0, err
	}

	var tied int64
	if err := r.db.WithContext(ctx).Model(&models.User{}).
		Where(
			"is_active = ? AND points = ? AND ((created_at < ?) OR (created_at = ? AND id < ?))",
			true, points, createdAt, createdAt, id,
		).
		Count(&tied).Error; err != nil {
		return 0, errors.NewInternal("Failed to compute rank", err)
	}

	return int(higher+tied) + 1, nil
}

func (r *userRepository) GetUserRoles(ctx context.Context, userID uuid.UUID) ([]models.Role, error) {
	var roles []models.Role
	err := r.db.WithContext(ctx).Table("roles").
		Joins("INNER JOIN user_roles ON roles.id = user_roles.role_id").
		Where("user_roles.user_id = ?", userID).
		Find(&roles).Error

	if err != nil {
		return nil, errors.NewInternal("Failed to get user roles", err)
	}

	return roles, nil
}
