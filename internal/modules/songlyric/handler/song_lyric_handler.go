package handler

import (
	"clap/internal/modules/songlyric/dto"
	"clap/internal/modules/songlyric/service"
	"clap/internal/shared/middleware"
	"clap/internal/shared/response"
	"clap/internal/shared/utils"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type SongLyricHandler interface {
	Create(c *gin.Context)
	GetByID(c *gin.Context)
	GetBySongID(c *gin.Context)
	ListBySongID(c *gin.Context)
	Update(c *gin.Context)
	Delete(c *gin.Context)
	ImportLyrics(c *gin.Context)
}

type songLyricHandler struct {
	lyricService service.SongLyricService
}

func NewSongLyricHandler(lyricService service.SongLyricService) SongLyricHandler {
	return &songLyricHandler{lyricService: lyricService}
}

// Create song lyric godoc
//
//	@Summary		Create song lyric
//	@Tags			song-lyrics
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			body	body		dto.CreateSongLyricRequest	true	"Request body"
//	@Success		201	{object}	response.Response
//	@Failure		401	{object}	response.Response
//	@Failure		400	{object}	response.Response
//	@Router			/api/v1/song-lyrics [post]
func (h *songLyricHandler) Create(c *gin.Context) {
	var req dto.CreateSongLyricRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	authCtx := getAuthContext(c)
	lyric, err := h.lyricService.Create(c.Request.Context(), &req, authCtx)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Created(c, lyric)
}

// Get song lyric godoc
//
//	@Summary		Get song lyric
//	@Tags			song-lyrics
//	@Produce		json
//	@Param			id	path	string	true	"Lyric ID"
//	@Success		200	{object}	response.Response
//	@Failure		400	{object}	response.Response
//	@Router			/api/v1/song-lyrics/{id} [get]
func (h *songLyricHandler) GetByID(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "Invalid ID")
		return
	}

	lyric, err := h.lyricService.GetByID(c.Request.Context(), id)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, lyric)
}

// Get lyrics by language godoc
//
//	@Summary		Get lyrics by language
//	@Tags			song-lyrics
//	@Produce		json
//	@Param			id	path	string	true	"Song ID"
//	@Param			language	path	string	true	"Language code"
//	@Success		200	{object}	response.Response
//	@Failure		400	{object}	response.Response
//	@Router			/api/v1/songs/{id}/lyrics/{language} [get]
func (h *songLyricHandler) GetBySongID(c *gin.Context) {
	songID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "Invalid song ID")
		return
	}

	language := c.Query("language")
	if language == "" {
		response.BadRequest(c, "Language query parameter is required")
		return
	}

	lyric, err := h.lyricService.GetBySongID(c.Request.Context(), songID, language)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, lyric)
}

// List lyrics for song godoc
//
//	@Summary		List lyrics for song
//	@Tags			song-lyrics
//	@Produce		json
//	@Param			id	path	string	true	"Song ID"
//	@Success		200	{object}	response.Response
//	@Failure		400	{object}	response.Response
//	@Router			/api/v1/songs/{id}/lyrics [get]
func (h *songLyricHandler) ListBySongID(c *gin.Context) {
	songID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "Invalid song ID")
		return
	}

	page, pageSize := utils.GetPagination(c)

	lyrics, err := h.lyricService.ListBySongID(c.Request.Context(), songID, page, pageSize)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, lyrics)
}

// Update song lyric godoc
//
//	@Summary		Update song lyric
//	@Tags			song-lyrics
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id	path	string	true	"Lyric ID"
//	@Param			body	body		dto.UpdateSongLyricRequest	true	"Request body"
//	@Success		200	{object}	response.Response
//	@Failure		401	{object}	response.Response
//	@Failure		400	{object}	response.Response
//	@Router			/api/v1/song-lyrics/{id} [put]
func (h *songLyricHandler) Update(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "Invalid ID")
		return
	}

	var req dto.UpdateSongLyricRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	authCtx := getAuthContext(c)
	lyric, err := h.lyricService.Update(c.Request.Context(), id, &req, authCtx)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, lyric)
}

// Delete song lyric godoc
//
//	@Summary		Delete song lyric
//	@Tags			song-lyrics
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id	path	string	true	"Lyric ID"
//	@Success		200	{object}	response.Response
//	@Failure		401	{object}	response.Response
//	@Failure		400	{object}	response.Response
//	@Router			/api/v1/song-lyrics/{id} [delete]
func (h *songLyricHandler) Delete(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "Invalid ID")
		return
	}

	authCtx := getAuthContext(c)
	if err := h.lyricService.Delete(c.Request.Context(), id, authCtx); err != nil {
		response.Error(c, err)
		return
	}

	response.SuccessWithMessage(c, nil, "Song lyric deleted successfully")
}

// Import lyrics godoc
//
//	@Summary		Import lyrics
//	@Tags			song-lyrics
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id	path	string	true	"Song ID"
//	@Param			body	body		dto.ImportLyricsRequest	true	"Request body"
//	@Success		200	{object}	response.Response
//	@Failure		401	{object}	response.Response
//	@Failure		400	{object}	response.Response
//	@Router			/api/v1/songs/{id}/lyrics/import [post]
func (h *songLyricHandler) ImportLyrics(c *gin.Context) {
	songID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "Invalid song ID")
		return
	}

	var req dto.ImportLyricsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	authCtx := getAuthContext(c)
	result, err := h.lyricService.ImportLyrics(c.Request.Context(), songID, &req, authCtx)
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
