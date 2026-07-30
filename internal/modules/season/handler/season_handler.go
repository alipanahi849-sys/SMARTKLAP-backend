package handler

import (
	"clap/internal/modules/season/dto"
	"clap/internal/modules/season/service"
	"clap/internal/shared/middleware"
	"clap/internal/shared/response"
	"clap/internal/shared/utils"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type SeasonHandler interface {
	Create(c *gin.Context)
	GetByID(c *gin.Context)
	List(c *gin.Context)
	ListByLeagueID(c *gin.Context)
	Update(c *gin.Context)
	Delete(c *gin.Context)
}

type seasonHandler struct {
	seasonService service.SeasonService
}

func NewSeasonHandler(seasonService service.SeasonService) SeasonHandler {
	return &seasonHandler{seasonService: seasonService}
}

// Create season godoc
//
//	@Summary		Create season
//	@Tags			seasons
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			body	body		dto.CreateSeasonRequest	true	"Request body"
//	@Success		201	{object}	response.Response
//	@Failure		401	{object}	response.Response
//	@Failure		400	{object}	response.Response
//	@Router			/api/v1/seasons [post]
func (h *seasonHandler) Create(c *gin.Context) {
	var req dto.CreateSeasonRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	authCtx := getAuthContext(c)
	season, err := h.seasonService.Create(c.Request.Context(), &req, authCtx)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Created(c, season)
}

// Get season godoc
//
//	@Summary		Get season
//	@Tags			seasons
//	@Produce		json
//	@Param			id	path	string	true	"Season ID"
//	@Success		200	{object}	response.Response
//	@Failure		400	{object}	response.Response
//	@Router			/api/v1/seasons/{id} [get]
func (h *seasonHandler) GetByID(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "Invalid ID")
		return
	}

	season, err := h.seasonService.GetByID(c.Request.Context(), id)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, season)
}

// List seasons godoc
//
//	@Summary		List seasons
//	@Tags			seasons
//	@Produce		json
//	@Param			sort	query	string	false	"Sort field"
//	@Param			order	query	string	false	"asc|desc"
//	@Param			page	query	int	false	"Page number"
//	@Param			per_page	query	int	false	"Items per page"
//	@Success		200	{object}	response.Response
//	@Failure		400	{object}	response.Response
//	@Router			/api/v1/seasons [get]
func (h *seasonHandler) List(c *gin.Context) {
	page, pageSize := utils.GetPagination(c)

	filters := make(map[string]string)
	for key, values := range c.Request.URL.Query() {
		if len(values) > 0 {
			filters[key] = values[0]
		}
	}

	sortBy := c.Query("sort")
	sortOrder := c.Query("order")

	seasons, err := h.seasonService.List(c.Request.Context(), page, pageSize, filters, sortBy, sortOrder)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, seasons)
}

// List seasons by league godoc
//
//	@Summary		List seasons by league
//	@Tags			seasons
//	@Produce		json
//	@Param			id	path	string	true	"League ID"
//	@Success		200	{object}	response.Response
//	@Failure		400	{object}	response.Response
//	@Router			/api/v1/leagues/{id}/seasons [get]
func (h *seasonHandler) ListByLeagueID(c *gin.Context) {
	leagueID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "Invalid league ID")
		return
	}

	page, pageSize := utils.GetPagination(c)

	seasons, err := h.seasonService.ListByLeagueID(c.Request.Context(), leagueID, page, pageSize)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, seasons)
}

// Update season godoc
//
//	@Summary		Update season
//	@Tags			seasons
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id	path	string	true	"Season ID"
//	@Param			body	body		dto.UpdateSeasonRequest	true	"Request body"
//	@Success		200	{object}	response.Response
//	@Failure		401	{object}	response.Response
//	@Failure		400	{object}	response.Response
//	@Router			/api/v1/seasons/{id} [put]
func (h *seasonHandler) Update(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "Invalid ID")
		return
	}

	var req dto.UpdateSeasonRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	authCtx := getAuthContext(c)
	season, err := h.seasonService.Update(c.Request.Context(), id, &req, authCtx)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, season)
}

// Delete season godoc
//
//	@Summary		Delete season
//	@Tags			seasons
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id	path	string	true	"Season ID"
//	@Success		200	{object}	response.Response
//	@Failure		401	{object}	response.Response
//	@Failure		400	{object}	response.Response
//	@Router			/api/v1/seasons/{id} [delete]
func (h *seasonHandler) Delete(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "Invalid ID")
		return
	}

	authCtx := getAuthContext(c)
	if err := h.seasonService.Delete(c.Request.Context(), id, authCtx); err != nil {
		response.Error(c, err)
		return
	}

	response.SuccessWithMessage(c, nil, "Season deleted successfully")
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
