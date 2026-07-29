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
