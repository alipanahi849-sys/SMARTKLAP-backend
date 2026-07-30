package handler

import (
	"clap/internal/modules/media/dto"
	"clap/internal/modules/media/service"
	"clap/internal/shared/middleware"
	"clap/internal/shared/response"
	"clap/internal/shared/utils"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type MediaHandler interface {
	Upload(c *gin.Context)
	GetPlaybackURL(c *gin.Context)
	UploadSongAudio(c *gin.Context)
}

type mediaHandler struct {
	mediaService service.MediaService
}

func NewMediaHandler(mediaService service.MediaService) MediaHandler {
	return &mediaHandler{mediaService: mediaService}
}

// Upload media godoc
//
//	@Summary		Upload media
//	@Tags			media
//	@Accept			mpfd
//	@Produce		json
//	@Security		BearerAuth
//	@Param			file	formData	file	true	"Media file"
//	@Param			type	formData	string	false	"Media type"
//	@Success		200	{object}	response.Response
//	@Failure		401	{object}	response.Response
//	@Failure		400	{object}	response.Response
//	@Router			/api/v1/media/upload [post]
func (h *mediaHandler) Upload(c *gin.Context) {
	var req dto.MediaUploadRequest
	if err := c.ShouldBind(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	authCtx := getAuthContext(c)
	media, err := h.mediaService.Upload(c.Request.Context(), req.File, authCtx)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Created(c, media)
}

// Get playback URL godoc
//
//	@Summary		Get playback URL
//	@Tags			media
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id	path	string	true	"Media ID"
//	@Success		200	{object}	response.Response
//	@Failure		401	{object}	response.Response
//	@Failure		400	{object}	response.Response
//	@Router			/api/v1/media/{id}/playback-url [get]
func (h *mediaHandler) GetPlaybackURL(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "Invalid ID")
		return
	}

	url, err := h.mediaService.GetPlaybackURL(c.Request.Context(), id)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, url)
}

// Upload song audio godoc
//
//	@Summary		Upload song audio
//	@Tags			media
//	@Accept			mpfd
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id	path	string	true	"Song ID"
//	@Param			file	formData	file	true	"Audio file"
//	@Success		200	{object}	response.Response
//	@Failure		401	{object}	response.Response
//	@Failure		400	{object}	response.Response
//	@Router			/api/v1/songs/{id}/audio [post]
func (h *mediaHandler) UploadSongAudio(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "Invalid ID")
		return
	}

	var req dto.SongAudioUploadRequest
	if err := c.ShouldBind(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	authCtx := getAuthContext(c)
	result, err := h.mediaService.UploadSongAudio(c.Request.Context(), id, req.File, authCtx)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, result)
}

func getAuthContext(c *gin.Context) *utils.AuthorizationContext {
	userID := middleware.GetUserID(c)
	roles := middleware.GetUserRoles(c)

	var clubID *uuid.UUID
	if clubIDStr := c.GetHeader("X-Club-ID"); clubIDStr != "" {
		if id, err := uuid.Parse(clubIDStr); err == nil {
			clubID = &id
		}
	}

	return utils.NewAuthorizationContext(userID, roles, clubID)
}
