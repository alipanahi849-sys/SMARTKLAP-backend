package handler

import (
	"clap/internal/modules/song/dto"
	"clap/internal/modules/song/service"
	"clap/internal/shared/middleware"
	"clap/internal/shared/response"
	"clap/internal/shared/utils"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type SongHandler interface {
	Create(c *gin.Context)
	GetByID(c *gin.Context)
	List(c *gin.Context)
	Search(c *gin.Context)
	Update(c *gin.Context)
	Delete(c *gin.Context)
}

type songHandler struct {
	songService service.SongService
}

func NewSongHandler(songService service.SongService) SongHandler {
	return &songHandler{songService: songService}
}

// Create song godoc
//
//	@Summary		Create song
//	@Tags			songs
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			body	body		dto.CreateSongRequest	true	"Request body"
//	@Success		201	{object}	response.Response
//	@Failure		401	{object}	response.Response
//	@Failure		400	{object}	response.Response
//	@Router			/api/v1/songs [post]
func (h *songHandler) Create(c *gin.Context) {
	var req dto.CreateSongRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	authCtx := getAuthContext(c)
	song, err := h.songService.Create(c.Request.Context(), &req, authCtx)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Created(c, song)
}

// Get song godoc
//
//	@Summary		Get song
//	@Tags			songs
//	@Produce		json
//	@Param			id	path	string	true	"Song ID"
//	@Success		200	{object}	response.Response
//	@Failure		400	{object}	response.Response
//	@Router			/api/v1/songs/{id} [get]
func (h *songHandler) GetByID(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "Invalid ID")
		return
	}

	song, err := h.songService.GetByID(c.Request.Context(), id)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, song)
}

// List songs godoc
//
//	@Summary		List songs
//	@Tags			songs
//	@Produce		json
//	@Param			sort	query	string	false	"Sort field"
//	@Param			order	query	string	false	"asc|desc"
//	@Param			page	query	int	false	"Page number"
//	@Param			per_page	query	int	false	"Items per page"
//	@Param			q	query	string	false	"Search query"
//	@Success		200	{object}	response.Response
//	@Failure		400	{object}	response.Response
//	@Router			/api/v1/songs [get]
func (h *songHandler) List(c *gin.Context) {
	page, pageSize := utils.GetPagination(c)

	filters := make(map[string]string)
	for key, values := range c.Request.URL.Query() {
		if len(values) > 0 {
			filters[key] = values[0]
		}
	}

	sortBy := c.Query("sort")
	sortOrder := c.Query("order")

	songs, err := h.songService.List(c.Request.Context(), page, pageSize, filters, sortBy, sortOrder)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, songs)
}

// Search songs godoc
//
//	@Summary		Search songs
//	@Tags			songs
//	@Produce		json
//	@Param			q	query	string	true	"Search query"
//	@Success		200	{object}	response.Response
//	@Failure		400	{object}	response.Response
//	@Router			/api/v1/songs/search [get]
func (h *songHandler) Search(c *gin.Context) {
	query := c.Query("q")
	if query == "" {
		response.BadRequest(c, "Search query is required")
		return
	}

	page, pageSize := utils.GetPagination(c)

	songs, err := h.songService.Search(c.Request.Context(), query, page, pageSize)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, songs)
}

// Update song godoc
//
//	@Summary		Update song
//	@Tags			songs
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id	path	string	true	"Song ID"
//	@Param			body	body		dto.UpdateSongRequest	true	"Request body"
//	@Success		200	{object}	response.Response
//	@Failure		401	{object}	response.Response
//	@Failure		400	{object}	response.Response
//	@Router			/api/v1/songs/{id} [put]
func (h *songHandler) Update(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "Invalid ID")
		return
	}

	var req dto.UpdateSongRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	authCtx := getAuthContext(c)
	song, err := h.songService.Update(c.Request.Context(), id, &req, authCtx)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, song)
}

// Delete song godoc
//
//	@Summary		Delete song
//	@Tags			songs
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id	path	string	true	"Song ID"
//	@Success		200	{object}	response.Response
//	@Failure		401	{object}	response.Response
//	@Failure		400	{object}	response.Response
//	@Router			/api/v1/songs/{id} [delete]
func (h *songHandler) Delete(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "Invalid ID")
		return
	}

	authCtx := getAuthContext(c)
	if err := h.songService.Delete(c.Request.Context(), id, authCtx); err != nil {
		response.Error(c, err)
		return
	}

	response.SuccessWithMessage(c, nil, "Song deleted successfully")
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
