package service

import (
	"clap/internal/modules/user/models"
	"clap/internal/modules/user/repository"
	"clap/internal/shared/errors"
	"context"

	"github.com/google/uuid"
)

type ProfileService interface {
	GetProfile(ctx context.Context, userID uuid.UUID) (*models.Profile, error)
	CreateProfile(ctx context.Context, userID uuid.UUID, profile *models.Profile) (*models.Profile, error)
	UpdateProfile(ctx context.Context, userID uuid.UUID, updates map[string]interface{}) (*models.Profile, error)
	DeleteProfile(ctx context.Context, userID uuid.UUID) error
}

type profileService struct {
	profileRepo repository.ProfileRepository
}

func NewProfileService(profileRepo repository.ProfileRepository) ProfileService {
	return &profileService{
		profileRepo: profileRepo,
	}
}

func (s *profileService) GetProfile(ctx context.Context, userID uuid.UUID) (*models.Profile, error) {
	return s.profileRepo.FindByUserID(ctx, userID)
}

func (s *profileService) CreateProfile(ctx context.Context, userID uuid.UUID, profile *models.Profile) (*models.Profile, error) {
	existing, _ := s.profileRepo.FindByUserID(ctx, userID)
	if existing != nil {
		return nil, errors.NewConflict("Profile already exists", nil)
	}

	profile.UserID = userID
	if err := s.profileRepo.Create(ctx, profile); err != nil {
		return nil, err
	}

	return s.profileRepo.FindByID(ctx, profile.ID)
}

func (s *profileService) UpdateProfile(ctx context.Context, userID uuid.UUID, updates map[string]interface{}) (*models.Profile, error) {
	profile, err := s.profileRepo.FindByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}

	if bio, ok := updates["bio"]; ok {
		profile.Bio = bio.(string)
	}
	if avatarURL, ok := updates["avatar_url"]; ok {
		profile.AvatarURL = avatarURL.(string)
	}
	if dateOfBirth, ok := updates["date_of_birth"]; ok {
		if dob, ok := dateOfBirth.(string); ok {
			profile.DateOfBirth = &dob
		}
	}
	if country, ok := updates["country"]; ok {
		profile.Country = country.(string)
	}
	if city, ok := updates["city"]; ok {
		profile.City = city.(string)
	}

	if err := s.profileRepo.Update(ctx, profile); err != nil {
		return nil, err
	}

	return s.profileRepo.FindByID(ctx, profile.ID)
}

func (s *profileService) DeleteProfile(ctx context.Context, userID uuid.UUID) error {
	profile, err := s.profileRepo.FindByUserID(ctx, userID)
	if err != nil {
		return err
	}
	return s.profileRepo.Delete(ctx, profile.ID)
}
