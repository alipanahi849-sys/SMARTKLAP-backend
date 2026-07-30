package handler

import (
	"clap/internal/modules/playback/dto"
	"clap/internal/modules/playback/service"
	"clap/internal/shared/middleware"
	"clap/internal/shared/response"
	"clap/internal/shared/utils"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type PlaybackHandler interface {
	ScheduleSong(c *gin.Context)
	CancelSong(c *gin.Context)
	GetUpcomingSongs(c *gin.Context)
}

type playbackHandler struct {
	svc service.PlaybackService
}

func NewPlaybackHandler(svc service.PlaybackService) PlaybackHandler {
	return &playbackHandler{svc: svc}
}

// POST /api/v1/songs/schedule
// Schedule song playback godoc
//
//	@Summary		Schedule song playback
//	@Tags			playback
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			body	body		dto.ScheduleSongRequest	true	"Request body"
//	@Success		201	{object}	response.Response
//	@Failure		401	{object}	response.Response
//	@Failure		400	{object}	response.Response
//	@Router			/api/v1/songs/schedule [post]
func (h *playbackHandler) ScheduleSong(c *gin.Context) {
	var req dto.ScheduleSongRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	authCtx := getAuthContext(c)
	result, err := h.svc.ScheduleSong(c.Request.Context(), &req, authCtx)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Created(c, result)
}

// DELETE /api/v1/songs/schedule/:id
// Cancel scheduled song godoc
//
//	@Summary		Cancel scheduled song
//	@Tags			playback
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id	path	string	true	"Schedule ID"
//	@Success		200	{object}	response.Response
//	@Failure		401	{object}	response.Response
//	@Failure		400	{object}	response.Response
//	@Router			/api/v1/songs/schedule/{id} [delete]
func (h *playbackHandler) CancelSong(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "Invalid schedule ID")
		return
	}

	authCtx := getAuthContext(c)
	if err := h.svc.CancelSong(c.Request.Context(), id, authCtx); err != nil {
		response.Error(c, err)
		return
	}
	response.SuccessWithMessage(c, nil, "Playback schedule cancelled")
}

// GET /api/v1/songs/schedule/upcoming?match_id=<uuid>
// Upcoming scheduled songs godoc
//
//	@Summary		Upcoming scheduled songs
//	@Tags			playback
//	@Produce		json
//	@Security		BearerAuth
//	@Param			match_id	query	string	true	"Match ID"
//	@Success		200	{object}	response.Response
//	@Failure		401	{object}	response.Response
//	@Failure		400	{object}	response.Response
//	@Router			/api/v1/songs/schedule/upcoming [get]
func (h *playbackHandler) GetUpcomingSongs(c *gin.Context) {
	matchIDStr := c.Query("match_id")
	if matchIDStr == "" {
		response.BadRequest(c, "match_id query parameter is required")
		return
	}

	matchID, err := uuid.Parse(matchIDStr)
	if err != nil {
		response.BadRequest(c, "Invalid match_id")
		return
	}

	result, err := h.svc.GetUpcomingSongs(c.Request.Context(), matchID)
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
	if s := c.GetHeader("X-Club-ID"); s != "" {
		if id, err := uuid.Parse(s); err == nil {
			clubID = &id
		}
	}
	return utils.NewAuthorizationContext(userID, roles, clubID)
}
