package handler

import (
	"strings"

	"clap/internal/modules/video/dto"
	"clap/internal/modules/video/service"
	"clap/internal/shared/middleware"
	"clap/internal/shared/response"
	"clap/internal/shared/utils"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// VideoHandler serves the mobile Video screens (contract §8).
type VideoHandler interface {
	Feed(c *gin.Context)
	Mine(c *gin.Context)
	Upload(c *gin.Context)
	Like(c *gin.Context)
	Unlike(c *gin.Context)
	MarkSeen(c *gin.Context)
}

type videoHandler struct {
	svc service.VideoService
}

func NewVideoHandler(svc service.VideoService) VideoHandler {
	return &videoHandler{svc: svc}
}

// Video feed godoc
//
//	@Summary		Video feed
//	@Tags			videos
//	@Produce		json
//	@Security		BearerAuth
//	@Param			cursor	query	string	false	"Video ID of the last item from the previous page"
//	@Param			limit	query	int		false	"Items per page (default 20, max 100)"
//	@Success		200	{object}	response.Response
//	@Failure		401	{object}	response.Response
//	@Failure		400	{object}	response.Response
//	@Router			/api/v1/videos/feed [get]
func (h *videoHandler) Feed(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == uuid.Nil {
		response.Unauthorized(c, "Invalid user")
		return
	}

	limit := utils.GetMobileCursorLimit(c)

	var cursor *uuid.UUID
	if raw := strings.TrimSpace(c.Query("cursor")); raw != "" {
		id, err := uuid.Parse(raw)
		if err != nil {
			response.BadRequest(c, "Invalid cursor")
			return
		}
		cursor = &id
	}

	result, err := h.svc.Feed(c.Request.Context(), userID, dto.VideoListFilters{
		Cursor: cursor,
		Limit:  limit,
	})
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, result)
}

// My videos godoc
//
//	@Summary		My videos
//	@Tags			videos
//	@Produce		json
//	@Security		BearerAuth
//	@Param			cursor	query	string	false	"Video ID of the last item from the previous page"
//	@Param			limit	query	int		false	"Items per page (default 20, max 100)"
//	@Success		200	{object}	response.Response
//	@Failure		401	{object}	response.Response
//	@Failure		400	{object}	response.Response
//	@Router			/api/v1/videos/mine [get]
func (h *videoHandler) Mine(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == uuid.Nil {
		response.Unauthorized(c, "Invalid user")
		return
	}

	limit := utils.GetMobileCursorLimit(c)

	var cursor *uuid.UUID
	if raw := strings.TrimSpace(c.Query("cursor")); raw != "" {
		id, err := uuid.Parse(raw)
		if err != nil {
			response.BadRequest(c, "Invalid cursor")
			return
		}
		cursor = &id
	}

	result, err := h.svc.Mine(c.Request.Context(), userID, dto.VideoListFilters{
		Cursor: cursor,
		Limit:  limit,
	})
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, result)
}

// Upload video godoc
//
//	@Summary		Upload video
//	@Tags			videos
//	@Accept			mpfd
//	@Produce		json
//	@Security		BearerAuth
//	@Param			media	formData	file	true	"Media file (image or video)"
//	@Param			type	formData	string	false	"image or video (inferred from file when omitted)"
//	@Param			caption	formData	string	false	"Caption"
//	@Success		200	{object}	response.Response
//	@Failure		401	{object}	response.Response
//	@Failure		400	{object}	response.Response
//	@Router			/api/v1/videos [post]
func (h *videoHandler) Upload(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == uuid.Nil {
		response.Unauthorized(c, "Invalid user")
		return
	}

	file := utils.FirstFormFile(c, "media", "file", "video", "image")
	if file == nil {
		response.BadRequest(c, "media file is required")
		return
	}

	mediaType := c.PostForm("type")
	caption := c.PostForm("caption")

	result, svcErr := h.svc.Upload(c.Request.Context(), userID, file, mediaType, caption)
	if svcErr != nil {
		response.Error(c, svcErr)
		return
	}
	response.CreatedWithMessage(c, result, "Video uploaded successfully")
}

// Like video godoc
//
//	@Summary		Like video
//	@Tags			videos
//	@Produce		json
//	@Security		BearerAuth
//	@Param			video_id	path	string	true	"Video ID"
//	@Success		200	{object}	response.Response
//	@Failure		401	{object}	response.Response
//	@Failure		400	{object}	response.Response
//	@Router			/api/v1/videos/{video_id}/like [post]
func (h *videoHandler) Like(c *gin.Context) {
	h.toggleLike(c, true)
}

// Unlike video godoc
//
//	@Summary		Unlike video
//	@Tags			videos
//	@Produce		json
//	@Security		BearerAuth
//	@Param			video_id	path	string	true	"Video ID"
//	@Success		200	{object}	response.Response
//	@Failure		401	{object}	response.Response
//	@Failure		400	{object}	response.Response
//	@Router			/api/v1/videos/{video_id}/like [delete]
func (h *videoHandler) Unlike(c *gin.Context) {
	h.toggleLike(c, false)
}

func (h *videoHandler) toggleLike(c *gin.Context, like bool) {
	userID := middleware.GetUserID(c)
	if userID == uuid.Nil {
		response.Unauthorized(c, "Invalid user")
		return
	}

	videoID, err := uuid.Parse(c.Param("video_id"))
	if err != nil {
		response.BadRequest(c, "Invalid video ID")
		return
	}

	var svcErr error
	if like {
		svcErr = h.svc.Like(c.Request.Context(), userID, videoID)
	} else {
		svcErr = h.svc.Unlike(c.Request.Context(), userID, videoID)
	}
	if svcErr != nil {
		response.Error(c, svcErr)
		return
	}
	if like {
		response.SuccessWithMessage(c, response.EmptyObject, "Video liked successfully")
		return
	}
	response.SuccessWithMessage(c, response.EmptyObject, "Video unliked successfully")
}

// Mark video as seen godoc
//
//	@Summary		Mark video as seen
//	@Tags			videos
//	@Produce		json
//	@Security		BearerAuth
//	@Param			video_id	path	string	true	"Video ID"
//	@Success		200	{object}	response.Response
//	@Failure		401	{object}	response.Response
//	@Failure		400	{object}	response.Response
//	@Failure		404	{object}	response.Response
//	@Router			/api/v1/videos/{video_id}/seen [post]
func (h *videoHandler) MarkSeen(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == uuid.Nil {
		response.Unauthorized(c, "Invalid user")
		return
	}

	videoID, err := uuid.Parse(c.Param("video_id"))
	if err != nil {
		response.BadRequest(c, "Invalid video ID")
		return
	}

	result, svcErr := h.svc.MarkSeen(c.Request.Context(), userID, videoID)
	if svcErr != nil {
		response.Error(c, svcErr)
		return
	}
	msg := "Video marked as seen"
	if !result.FirstSeen {
		msg = "Video already marked as seen"
	}
	response.SuccessWithMessage(c, result, msg)
}
