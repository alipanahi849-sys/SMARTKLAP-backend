package repository

import (
	"clap/internal/modules/auth/models"
	"clap/internal/shared/database"
	"clap/internal/shared/errors"
	"context"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type RoleRepository interface {
	Create(ctx context.Context, role *models.Role) error
	FindByID(ctx context.Context, id uuid.UUID) (*models.Role, error)
	FindByName(ctx context.Context, name string) (*models.Role, error)
	Update(ctx context.Context, role *models.Role) error
	Delete(ctx context.Context, id uuid.UUID) error
	List(ctx context.Context, offset, limit int) ([]models.Role, int64, error)
}

type roleRepository struct {
	db *gorm.DB
}

func NewRoleRepository() RoleRepository {
	return &roleRepository{
		db: database.GetDB(),
	}
}

func (r *roleRepository) Create(ctx context.Context, role *models.Role) error {
	if err := r.db.WithContext(ctx).Create(role).Error; err != nil {
		return errors.NewInternal("Failed to create role", err)
	}
	return nil
}

func (r *roleRepository) FindByID(ctx context.Context, id uuid.UUID) (*models.Role, error) {
	var role models.Role
	err := r.db.WithContext(ctx).First(&role, "id = ?", id).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.ErrUserNotFound
		}
		return nil, errors.NewInternal("Failed to find role", err)
	}
	return &role, nil
}

func (r *roleRepository) FindByName(ctx context.Context, name string) (*models.Role, error) {
	var role models.Role
	err := r.db.WithContext(ctx).First(&role, "name = ?", name).Error
	if err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, errors.ErrUserNotFound
		}
		return nil, errors.NewInternal("Failed to find role", err)
	}
	return &role, nil
}

func (r *roleRepository) Update(ctx context.Context, role *models.Role) error {
	if err := r.db.WithContext(ctx).Save(role).Error; err != nil {
		return errors.NewInternal("Failed to update role", err)
	}
	return nil
}

func (r *roleRepository) Delete(ctx context.Context, id uuid.UUID) error {
	if err := r.db.WithContext(ctx).Delete(&models.Role{}, "id = ?", id).Error; err != nil {
		return errors.NewInternal("Failed to delete role", err)
	}
	return nil
}

func (r *roleRepository) List(ctx context.Context, offset, limit int) ([]models.Role, int64, error) {
	var roles []models.Role
	var total int64

	if err := r.db.WithContext(ctx).Model(&models.Role{}).Count(&total).Error; err != nil {
		return nil, 0, errors.NewInternal("Failed to count roles", err)
	}

	if err := r.db.WithContext(ctx).Offset(offset).Limit(limit).Find(&roles).Error; err != nil {
		return nil, 0, errors.NewInternal("Failed to list roles", err)
	}

	return roles, total, nil
}
