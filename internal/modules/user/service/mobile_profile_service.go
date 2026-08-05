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
	"clap/pkg/media/optimize"
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
	Leaderboard(ctx context.Context, filters dto.LeaderboardFilters) (*dto.LeaderboardResponse, error)
	UploadAvatar(ctx context.Context, userID uuid.UUID, file *multipart.FileHeader) (*dto.AvatarUploadResponse, error)
}

type mobileProfileService struct {
	userRepo    authrepo.UserRepository
	profileRepo repository.ProfileRepository
	storage     storage.StorageProvider
	optimizer   optimize.Optimizer
}

func NewMobileProfileService(
	userRepo authrepo.UserRepository,
	profileRepo repository.ProfileRepository,
	storageProvider storage.StorageProvider,
) MobileProfileService {
	return NewMobileProfileServiceWithOptimizer(userRepo, profileRepo, storageProvider, optimize.Noop{})
}

func NewMobileProfileServiceWithOptimizer(
	userRepo authrepo.UserRepository,
	profileRepo repository.ProfileRepository,
	storageProvider storage.StorageProvider,
	optimizer optimize.Optimizer,
) MobileProfileService {
	if optimizer == nil {
		optimizer = optimize.Noop{}
	}
	return &mobileProfileService{
		userRepo:    userRepo,
		profileRepo: profileRepo,
		storage:     storageProvider,
		optimizer:   optimizer,
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

	if err := s.userRepo.Update(ctx, user); err != nil {
		return nil, err
	}

	logger.Info().
		Str("user_id", userID.String()).
		Msg("mobile_profile_updated")

	return s.buildProfileResponse(ctx, user)
}

func (s *mobileProfileService) Leaderboard(ctx context.Context, filters dto.LeaderboardFilters) (*dto.LeaderboardResponse, error) {
	limit := filters.Limit
	if limit <= 0 {
		limit = 4
	}
	if limit > 50 {
		limit = 50
	}

	var after *authrepo.LeaderboardCursorAnchor
	startRank := 1
	if filters.Cursor != nil {
		cursorUser, err := s.userRepo.FindByID(ctx, *filters.Cursor)
		if err != nil {
			return nil, errors.NewBadRequest("Invalid cursor", nil)
		}
		after = &authrepo.LeaderboardCursorAnchor{
			Points:    cursorUser.Points,
			CreatedAt: cursorUser.CreatedAt,
			ID:        cursorUser.ID,
		}
		rank, rankErr := s.userRepo.LeaderboardRank(ctx, cursorUser.Points, cursorUser.CreatedAt, cursorUser.ID)
		if rankErr != nil {
			return nil, rankErr
		}
		startRank = rank + 1
	}

	users, err := s.userRepo.TopByPointsAfter(ctx, limit+1, after)
	if err != nil {
		return nil, err
	}

	hasMore := len(users) > limit
	if hasMore {
		users = users[:limit]
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
			Rank:      startRank + i,
			Name:      u.DisplayName(),
			Points:    u.Points,
			AvatarURL: avatar,
		}
	}

	meta := dto.LeaderboardMeta{
		Limit:   limit,
		HasMore: hasMore,
	}
	if hasMore && len(users) > 0 {
		lastID := users[len(users)-1].ID
		meta.NextCursor = &lastID
	}

	return &dto.LeaderboardResponse{Items: items, Meta: meta}, nil
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

	prepared, err := s.optimizer.OptimizeImage(ctx, src, ext, optimize.ImageProfileAvatar)
	if err != nil {
		return nil, errors.NewInternal("Failed to optimize avatar", err)
	}
	defer prepared.Cleanup()

	reader, err := prepared.Open()
	if err != nil {
		return nil, errors.NewInternal("Failed to read optimized avatar", err)
	}
	defer reader.Close()

	// Unique key per upload so clients never reuse a cached URL for a new image.
	key := fmt.Sprintf("avatars/%s/%s%s", userID.String(), uuid.NewString(), prepared.Extension)
	if err := s.storage.Upload(ctx, key, reader, prepared.ContentType, prepared.Size); err != nil {
		logger.Error().Err(err).Str("user_id", userID.String()).Str("key", key).Msg("avatar_store_failed")
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
		oldKey := profile.AvatarURL
		profile.AvatarURL = key
		if updateErr := s.profileRepo.Update(ctx, profile); updateErr != nil {
			return nil, updateErr
		}
		// Best-effort cleanup of the previous object (ignore failures).
		if oldKey != "" && oldKey != key &&
			!strings.HasPrefix(oldKey, "http://") && !strings.HasPrefix(oldKey, "https://") {
			_ = s.storage.Delete(ctx, oldKey)
		}
	}

	// Reload so CreatedAt/UpdatedAt match what we return (like video upload).
	if reloaded, reloadErr := s.profileRepo.FindByUserID(ctx, userID); reloadErr == nil && reloaded != nil {
		profile = reloaded
	}

	user, err := s.userRepo.FindByID(ctx, userID)
	if err != nil {
		return nil, err
	}

	logger.Info().
		Str("user_id", userID.String()).
		Str("storage_key", key).
		Msg("avatar_uploaded")

	return &dto.AvatarUploadResponse{
		ID:        user.ID,
		CreatedAt: user.CreatedAt,
		UpdatedAt: latestTime(user.UpdatedAt, profile.UpdatedAt),
		AvatarURL: s.resolveAvatarURL(ctx, key),
	}, nil
}

// ─── internals ────────────────────────────────────────────────────────────────

func (s *mobileProfileService) buildProfileResponse(ctx context.Context, user *authmodels.User) (*dto.MobileProfileResponse, error) {
	updatedAt := user.UpdatedAt
	avatarURL := ""
	if profile, err := s.profileRepo.FindByUserID(ctx, user.ID); err == nil && profile != nil {
		avatarURL = s.resolveAvatarURL(ctx, profile.AvatarURL)
		updatedAt = latestTime(user.UpdatedAt, profile.UpdatedAt)
	}

	rank := dto.ProfileRank{Position: 1, Total: 1}
	if higher, err := s.userRepo.CountWithMorePoints(ctx, user.Points); err == nil {
		rank.Position = int(higher) + 1
	}
	if total, err := s.userRepo.CountActive(ctx); err == nil {
		rank.Total = int(total)
	}

	return &dto.MobileProfileResponse{
		ID:        user.ID,
		CreatedAt: user.CreatedAt,
		UpdatedAt: updatedAt,
		Name:      user.DisplayName(),
		Email:     user.Email,
		AvatarURL: avatarURL,
		Points:    user.Points,
		Rank:      rank,
	}, nil
}

func latestTime(a, b time.Time) time.Time {
	if b.After(a) {
		return b
	}
	return a
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
