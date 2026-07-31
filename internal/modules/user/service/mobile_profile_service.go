package service

import (
	"context"
	"fmt"
	"mime"
	"mime/multipart"
	"path/filepath"
	"strings"
	"time"

	authmodels "clap/internal/modules/auth/models"
	authrepo "clap/internal/modules/auth/repository"
	"clap/internal/modules/user/dto"
	"clap/internal/modules/user/models"
	"clap/internal/modules/user/repository"
	"clap/internal/shared/errors"
	"clap/internal/shared/logger"
	"clap/pkg/storage"

	"github.com/google/uuid"
)

const (
	maxAvatarSizeBytes = 5 * 1024 * 1024 // 5 MB (contract §2.2)
	avatarURLExpiry    = 24 * time.Hour
)

var allowedAvatarMimeTypes = map[string]string{
	"image/jpeg": ".jpg",
	"image/png":  ".png",
	"image/webp": ".webp",
}

// MobileProfileService implements the mobile Profile screens (contract §2).
type MobileProfileService interface {
	GetMe(ctx context.Context, userID uuid.UUID) (*dto.MobileProfileResponse, error)
	UpdateMe(ctx context.Context, userID uuid.UUID, req *dto.UpdateMobileProfileRequest) (*dto.MobileProfileResponse, error)
	Leaderboard(ctx context.Context, limit int) (*dto.LeaderboardResponse, error)
	UploadAvatar(ctx context.Context, userID uuid.UUID, file *multipart.FileHeader) (*dto.AvatarUploadResponse, error)
}

type mobileProfileService struct {
	userRepo    authrepo.UserRepository
	profileRepo repository.ProfileRepository
	storage     storage.StorageProvider
}

func NewMobileProfileService(
	userRepo authrepo.UserRepository,
	profileRepo repository.ProfileRepository,
	storageProvider storage.StorageProvider,
) MobileProfileService {
	return &mobileProfileService{
		userRepo:    userRepo,
		profileRepo: profileRepo,
		storage:     storageProvider,
	}
}

func (s *mobileProfileService) GetMe(ctx context.Context, userID uuid.UUID) (*dto.MobileProfileResponse, error) {
	user, err := s.userRepo.FindByID(ctx, userID)
	if err != nil {
		return nil, err
	}
	return s.buildProfileResponse(ctx, user)
}

func (s *mobileProfileService) UpdateMe(ctx context.Context, userID uuid.UUID, req *dto.UpdateMobileProfileRequest) (*dto.MobileProfileResponse, error) {
	user, err := s.userRepo.FindByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	if req.Name != nil {
		name := strings.TrimSpace(*req.Name)
		if name == "" {
			return nil, errors.NewBadRequest("Name cannot be empty", nil)
		}
		user.FirstName = name
		user.LastName = ""
	}

	if req.Email != nil {
		email := strings.ToLower(strings.TrimSpace(*req.Email))
		if email != user.Email {
			if existing, findErr := s.userRepo.FindByEmail(ctx, email); findErr == nil && existing != nil && existing.ID != userID {
				return nil, errors.NewUnprocessable("Email already in use", nil)
			}
			user.Email = email
		}
	}

	if err := s.userRepo.Update(ctx, user); err != nil {
		return nil, err
	}

	logger.Info().
		Str("user_id", userID.String()).
		Msg("mobile_profile_updated")

	return s.buildProfileResponse(ctx, user)
}

func (s *mobileProfileService) Leaderboard(ctx context.Context, limit int) (*dto.LeaderboardResponse, error) {
	if limit <= 0 {
		limit = 4
	}
	if limit > 50 {
		limit = 50
	}

	users, err := s.userRepo.TopByPoints(ctx, limit)
	if err != nil {
		return nil, err
	}

	// Batch-load avatars in a single query (no N+1).
	userIDs := make([]uuid.UUID, len(users))
	for i, u := range users {
		userIDs[i] = u.ID
	}
	profiles, err := s.profileRepo.FindByUserIDs(ctx, userIDs)
	if err != nil {
		return nil, err
	}

	items := make([]dto.LeaderboardItem, len(users))
	for i, u := range users {
		avatar := ""
		if p, ok := profiles[u.ID]; ok {
			avatar = s.resolveAvatarURL(ctx, p.AvatarURL)
		}
		items[i] = dto.LeaderboardItem{
			Rank:      i + 1,
			Name:      u.DisplayName(),
			Points:    u.Points,
			AvatarURL: avatar,
		}
	}

	return &dto.LeaderboardResponse{Items: items}, nil
}

func (s *mobileProfileService) UploadAvatar(ctx context.Context, userID uuid.UUID, file *multipart.FileHeader) (*dto.AvatarUploadResponse, error) {
	if file.Size > maxAvatarSizeBytes {
		return nil, errors.NewPayloadTooLarge("Avatar must be at most 5 MB", nil)
	}

	mimeType := file.Header.Get("Content-Type")
	if mimeType == "" {
		mimeType = mime.TypeByExtension(filepath.Ext(file.Filename))
	}
	ext, ok := allowedAvatarMimeTypes[strings.ToLower(mimeType)]
	if !ok {
		return nil, errors.NewUnsupportedMedia("Avatar must be a JPEG, PNG or WebP image", nil)
	}

	if s.storage == nil {
		return nil, errors.NewInternal("Storage is not configured", nil)
	}

	src, err := file.Open()
	if err != nil {
		return nil, errors.NewInternal("Failed to open uploaded file", err)
	}
	defer src.Close()

	key := fmt.Sprintf("avatars/%s%s", userID.String(), ext)
	if err := s.storage.Upload(ctx, key, src, mimeType, file.Size); err != nil {
		return nil, errors.NewInternal("Failed to store avatar", err)
	}

	profile, err := s.profileRepo.FindByUserID(ctx, userID)
	if err != nil {
		// No profile row yet — create one holding the avatar key.
		profile = &models.Profile{UserID: userID, AvatarURL: key}
		if createErr := s.profileRepo.Create(ctx, profile); createErr != nil {
			return nil, createErr
		}
	} else {
		profile.AvatarURL = key
		if updateErr := s.profileRepo.Update(ctx, profile); updateErr != nil {
			return nil, updateErr
		}
	}

	logger.Info().
		Str("user_id", userID.String()).
		Str("storage_key", key).
		Msg("avatar_uploaded")

	return &dto.AvatarUploadResponse{AvatarURL: s.resolveAvatarURL(ctx, key)}, nil
}

// ─── internals ────────────────────────────────────────────────────────────────

func (s *mobileProfileService) buildProfileResponse(ctx context.Context, user *authmodels.User) (*dto.MobileProfileResponse, error) {
	return &dto.MobileProfileResponse{
		ID:        user.ID,
		CreatedAt: user.CreatedAt,
		UpdatedAt: user.UpdatedAt,
		Name:      user.DisplayName(),
		Email:     user.Email,
		AvatarURL: s.avatarURLForUser(ctx, user.ID),
		Points:    user.Points,
	}, nil
}

func (s *mobileProfileService) avatarURLForUser(ctx context.Context, userID uuid.UUID) string {
	profile, err := s.profileRepo.FindByUserID(ctx, userID)
	if err != nil || profile == nil {
		return ""
	}
	return s.resolveAvatarURL(ctx, profile.AvatarURL)
}

// resolveAvatarURL turns a stored value into a fetchable URL. Absolute URLs
// (legacy rows) pass through; storage keys are converted to signed URLs.
func (s *mobileProfileService) resolveAvatarURL(ctx context.Context, stored string) string {
	if stored == "" || strings.HasPrefix(stored, "http://") || strings.HasPrefix(stored, "https://") {
		return stored
	}
	if s.storage == nil {
		return ""
	}
	url, err := s.storage.GenerateSignedURL(ctx, stored, avatarURLExpiry)
	if err != nil {
		return ""
	}
	return url
}
