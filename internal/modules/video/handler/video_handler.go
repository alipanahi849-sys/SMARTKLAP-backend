package handler

import (
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
}

type videoHandler struct {
	svc service.VideoService
}

func NewVideoHandler(svc service.VideoService) VideoHandler {
	return &videoHandler{svc: svc}
}

func (h *videoHandler) Feed(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == uuid.Nil {
		response.Unauthorized(c, "Invalid user")
		return
	}

	page, limit := utils.GetMobilePagination(c)
	result, err := h.svc.Feed(c.Request.Context(), userID, page, limit)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, result)
}

func (h *videoHandler) Mine(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == uuid.Nil {
		response.Unauthorized(c, "Invalid user")
		return
	}

	page, limit := utils.GetMobilePagination(c)
	result, err := h.svc.Mine(c.Request.Context(), userID, page, limit)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, result)
}

func (h *videoHandler) Upload(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == uuid.Nil {
		response.Unauthorized(c, "Invalid user")
		return
	}

	file, err := c.FormFile("media")
	if err != nil {
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
	response.Created(c, result)
}

func (h *videoHandler) Like(c *gin.Context) {
	h.toggleLike(c, true)
}

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
	response.NoContent(c)
}
