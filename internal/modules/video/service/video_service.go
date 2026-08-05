package service

import (
	"context"
	"encoding/json"
	"fmt"
	"mime"
	"mime/multipart"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	userrepo "clap/internal/modules/user/repository"
	"clap/internal/modules/video/dto"
	"clap/internal/modules/video/models"
	"clap/internal/modules/video/repository"
	"clap/internal/shared/errors"
	"clap/internal/shared/logger"
	"clap/pkg/storage"

	"github.com/google/uuid"
)

const mediaURLExpiry = 6 * time.Hour

// Allowed upload formats per media type (contract §8.3: 415 on others).
var allowedVideoMimes = map[string]string{
	"video/mp4":       ".mp4",
	"video/quicktime": ".mov",
	"video/webm":      ".webm",
}

var allowedImageMimes = map[string]string{
	"image/jpeg": ".jpg",
	"image/png":  ".png",
	"image/webp": ".webp",
}

var hashtagPattern = regexp.MustCompile(`#([\p{L}\p{N}_]+)`)

// VideoService implements the mobile Video screens (contract §8).
type VideoService interface {
	Feed(ctx context.Context, userID uuid.UUID, filters dto.VideoListFilters) (*dto.VideoFeedResponse, error)
	Mine(ctx context.Context, userID uuid.UUID, filters dto.VideoListFilters) (*dto.VideoFeedResponse, error)
	Upload(ctx context.Context, userID uuid.UUID, file *multipart.FileHeader, mediaType, caption string) (*dto.VideoUploadResponse, error)
	Like(ctx context.Context, userID, videoID uuid.UUID) error
	Unlike(ctx context.Context, userID, videoID uuid.UUID) error
}

type videoService struct {
	videoRepo   repository.VideoRepository
	profileRepo userrepo.ProfileRepository
	storage     storage.StorageProvider
	maxFileSize int64
}

func NewVideoService(
	videoRepo repository.VideoRepository,
	profileRepo userrepo.ProfileRepository,
	storageProvider storage.StorageProvider,
	maxFileSizeMB int,
) VideoService {
	if maxFileSizeMB <= 0 {
		maxFileSizeMB = 50
	}
	return &videoService{
		videoRepo:   videoRepo,
		profileRepo: profileRepo,
		storage:     storageProvider,
		maxFileSize: int64(maxFileSizeMB) * 1024 * 1024,
	}
}

func (s *videoService) Feed(ctx context.Context, userID uuid.UUID, filters dto.VideoListFilters) (*dto.VideoFeedResponse, error) {
	limit := filters.Limit
	if limit <= 0 {
		limit = 20
	}

	var after *repository.VideoCursorAnchor
	if filters.Cursor != nil {
		cursorVideo, err := s.videoRepo.FindByID(ctx, *filters.Cursor)
		if err != nil {
			return nil, errors.NewBadRequest("Invalid cursor", nil)
		}
		if cursorVideo.Status != models.StatusPublished {
			return nil, errors.NewBadRequest("Invalid cursor", nil)
		}
		after = &repository.VideoCursorAnchor{
			CreatedAt: cursorVideo.CreatedAt,
			ID:        cursorVideo.ID,
		}
	}

	videos, err := s.videoRepo.FeedAfter(ctx, limit+1, after)
	if err != nil {
		return nil, err
	}
	return s.buildFeedResponse(ctx, userID, videos, limit)
}

func (s *videoService) Mine(ctx context.Context, userID uuid.UUID, filters dto.VideoListFilters) (*dto.VideoFeedResponse, error) {
	limit := filters.Limit
	if limit <= 0 {
		limit = 20
	}

	var after *repository.VideoCursorAnchor
	if filters.Cursor != nil {
		cursorVideo, err := s.videoRepo.FindByID(ctx, *filters.Cursor)
		if err != nil {
			return nil, errors.NewBadRequest("Invalid cursor", nil)
		}
		if cursorVideo.UserID != userID {
			return nil, errors.NewBadRequest("Invalid cursor", nil)
		}
		after = &repository.VideoCursorAnchor{
			CreatedAt: cursorVideo.CreatedAt,
			ID:        cursorVideo.ID,
		}
	}

	videos, err := s.videoRepo.ByUserAfter(ctx, userID, limit+1, after)
	if err != nil {
		return nil, err
	}
	return s.buildFeedResponse(ctx, userID, videos, limit)
}

func (s *videoService) Upload(ctx context.Context, userID uuid.UUID, file *multipart.FileHeader, mediaType, caption string) (*dto.VideoUploadResponse, error) {
	if file.Size > s.maxFileSize {
		return nil, errors.NewPayloadTooLarge(
			fmt.Sprintf("File exceeds the %d MB limit", s.maxFileSize/(1024*1024)), nil,
		)
	}

	mimeType := file.Header.Get("Content-Type")
	if mimeType == "" {
		mimeType = mime.TypeByExtension(filepath.Ext(file.Filename))
	}
	mimeType = strings.ToLower(mimeType)

	mediaType = strings.ToLower(strings.TrimSpace(mediaType))
	if mediaType == "" {
		switch {
		case strings.HasPrefix(mimeType, "video/"):
			mediaType = models.MediaTypeVideo
		case strings.HasPrefix(mimeType, "image/"):
			mediaType = models.MediaTypeImage
		default:
			return nil, errors.NewBadRequest("type must be 'image' or 'video'", nil)
		}
	}
	if mediaType != models.MediaTypeImage && mediaType != models.MediaTypeVideo {
		return nil, errors.NewBadRequest("type must be 'image' or 'video'", nil)
	}

	allowed := allowedImageMimes
	if mediaType == models.MediaTypeVideo {
		allowed = allowedVideoMimes
	}
	ext, ok := allowed[mimeType]
	if !ok {
		return nil, errors.NewUnsupportedMedia("Unsupported media format", nil)
	}

	if s.storage == nil {
		return nil, errors.NewInternal("Storage is not configured", nil)
	}

	src, err := file.Open()
	if err != nil {
		return nil, errors.NewInternal("Failed to open uploaded file", err)
	}
	defer src.Close()

	videoID := uuid.New()
	key := fmt.Sprintf("videos/%s/%s%s", userID.String(), videoID.String(), ext)
	if err := s.storage.Upload(ctx, key, src, mimeType, file.Size); err != nil {
		return nil, errors.NewInternal("Failed to store media", err)
	}

	tags, _ := json.Marshal(extractHashtags(caption))
	video := &models.Video{
		ID:         videoID,
		UserID:     userID,
		MediaType:  mediaType,
		Caption:    strings.TrimSpace(caption),
		Tags:       string(tags),
		StorageKey: key,
		MimeType:   mimeType,
		FileSize:   file.Size,
		// No async transcoding pipeline exists — posts publish immediately.
		Status: models.StatusPublished,
	}
	if err := s.videoRepo.Create(ctx, video); err != nil {
		_ = s.storage.Delete(ctx, key)
		return nil, err
	}

	logger.Info().
		Str("user_id", userID.String()).
		Str("video_id", video.ID.String()).
		Str("media_type", mediaType).
		Int64("file_size", file.Size).
		Msg("video_uploaded")

	url := s.signedURL(ctx, key)
	var videoURL *string
	if url != "" {
		videoURL = &url
	}

	return &dto.VideoUploadResponse{
		ID:        video.ID,
		CreatedAt: video.CreatedAt,
		UpdatedAt: video.UpdatedAt,
		Status:    video.Status,
		VideoURL:  videoURL,
	}, nil
}

func (s *videoService) Like(ctx context.Context, userID, videoID uuid.UUID) error {
	if _, err := s.videoRepo.FindByID(ctx, videoID); err != nil {
		return err
	}
	created, err := s.videoRepo.Like(ctx, videoID, userID)
	if err != nil {
		return err
	}
	if created {
		logger.Info().
			Str("user_id", userID.String()).
			Str("video_id", videoID.String()).
			Msg("video_liked")
	}
	return nil
}

func (s *videoService) Unlike(ctx context.Context, userID, videoID uuid.UUID) error {
	if _, err := s.videoRepo.FindByID(ctx, videoID); err != nil {
		return err
	}
	removed, err := s.videoRepo.Unlike(ctx, videoID, userID)
	if err != nil {
		return err
	}
	if removed {
		logger.Info().
			Str("user_id", userID.String()).
			Str("video_id", videoID.String()).
			Msg("video_unliked")
	}
	return nil
}

// ─── internals ────────────────────────────────────────────────────────────────

func (s *videoService) buildFeedResponse(ctx context.Context, userID uuid.UUID, videos []models.Video, limit int) (*dto.VideoFeedResponse, error) {
	hasMore := len(videos) > limit
	if hasMore {
		videos = videos[:limit]
	}

	videoIDs := make([]uuid.UUID, len(videos))
	authorIDs := make([]uuid.UUID, len(videos))
	for i, v := range videos {
		videoIDs[i] = v.ID
		authorIDs[i] = v.UserID
	}

	liked, err := s.videoRepo.LikedVideoIDs(ctx, userID, videoIDs)
	if err != nil {
		return nil, err
	}
	profiles, err := s.profileRepo.FindByUserIDs(ctx, authorIDs)
	if err != nil {
		return nil, err
	}

	items := make([]dto.VideoItem, len(videos))
	for i, v := range videos {
		avatar := ""
		if p, ok := profiles[v.UserID]; ok {
			avatar = s.resolveURL(ctx, p.AvatarURL)
		}

		var tags []string
		if err := json.Unmarshal([]byte(v.Tags), &tags); err != nil || tags == nil {
			tags = []string{}
		}

		items[i] = dto.VideoItem{
			ID:           v.ID,
			CreatedAt:    v.CreatedAt,
			UpdatedAt:    v.UpdatedAt,
			VideoURL:     s.signedURL(ctx, v.StorageKey),
			ThumbnailURL: s.signedURL(ctx, v.ThumbnailKey),
			Author: dto.VideoAuthor{
				Name:      v.User.DisplayName(),
				AvatarURL: avatar,
			},
			PostedAt:   v.CreatedAt.UTC().Format(time.RFC3339),
			Tags:       tags,
			LikesCount: v.LikesCount,
			ViewsCount: v.ViewsCount,
			IsLiked:    liked[v.ID],
		}
	}

	meta := dto.VideoListMeta{
		Limit:   limit,
		HasMore: hasMore,
	}
	if hasMore && len(videos) > 0 {
		lastID := videos[len(videos)-1].ID
		meta.NextCursor = &lastID
	}

	return &dto.VideoFeedResponse{
		Items: items,
		Meta:  meta,
	}, nil
}

func (s *videoService) signedURL(ctx context.Context, key string) string {
	if key == "" || s.storage == nil {
		return ""
	}
	url, err := s.storage.GenerateSignedURL(ctx, key, mediaURLExpiry)
	if err != nil {
		return ""
	}
	return url
}

// resolveURL passes absolute URLs through and signs storage keys.
func (s *videoService) resolveURL(ctx context.Context, stored string) string {
	if stored == "" || strings.HasPrefix(stored, "http://") || strings.HasPrefix(stored, "https://") {
		return stored
	}
	return s.signedURL(ctx, stored)
}

// extractHashtags parses #tags out of the caption (the New Post screen has no
// separate tags field).
func extractHashtags(caption string) []string {
	matches := hashtagPattern.FindAllStringSubmatch(caption, -1)
	tags := make([]string, 0, len(matches))
	seen := map[string]bool{}
	for _, m := range matches {
		if !seen[m[1]] {
			seen[m[1]] = true
			tags = append(tags, m[1])
		}
	}
	return tags
}
