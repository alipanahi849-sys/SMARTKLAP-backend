package repository

import (
	"clap/internal/modules/auth/models"
	"clap/internal/shared/database"
	"clap/internal/shared/errors"
	"context"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type UserRepository interface {
	Create(ctx context.Context, user *models.User) error
	FindByID(ctx context.Context, id uuid.UUID) (*models.User, error)
	FindByEmail(ctx context.Context, email string) (*models.User, error)
	Update(ctx context.Context, user *models.User) error
	Delete(ctx context.Context, id uuid.UUID) error
	List(ctx context.Context, offset, limit int) ([]models.User, int64, error)
	AddRole(ctx context.Context, userID, roleID uuid.UUID) error
	RemoveRole(ctx context.Context, userID, roleID uuid.UUID) error
	GetUserRoles(ctx context.Context, userID uuid.UUID) ([]models.Role, error)
	// AddPoints atomically increments (or decrements) a user's points balance
	// and returns the new balance.
	AddPoints(ctx context.Context, userID uuid.UUID, delta int) (int, error)
	// CountActive returns the number of active users (leaderboard total).
	CountActive(ctx context.Context) (int64, error)
	// CountWithMorePoints returns how many active users outrank the given
	// points balance; rank position = result + 1.
	CountWithMorePoints(ctx context.Context, points int) (int64, error)
	// TopByPoints returns the highest-scoring active users for the leaderboard.
	TopByPoints(ctx context.Context, limit int) ([]models.User, error)
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
	var users []models.User
	var total int64

	if err := r.db.WithContext(ctx).Model(&models.User{}).Count(&total).Error; err != nil {
		return nil, 0, errors.NewInternal("Failed to count users", err)
	}

	if err := r.db.WithContext(ctx).Preload("Roles").Offset(offset).Limit(limit).Find(&users).Error; err != nil {
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

func (r *userRepository) TopByPoints(ctx context.Context, limit int) ([]models.User, error) {
	var users []models.User
	if err := r.db.WithContext(ctx).
		Where("is_active = ?", true).
		Order("points DESC, created_at ASC").
		Limit(limit).
		Find(&users).Error; err != nil {
		return nil, errors.NewInternal("Failed to load leaderboard", err)
	}
	return users, nil
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
