package handler

import (
	"clap/internal/modules/match/dto"
	"clap/internal/modules/match/service"
	"clap/internal/shared/middleware"
	"clap/internal/shared/response"
	"clap/internal/shared/utils"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type MatchHandler interface {
	Create(c *gin.Context)
	GetByID(c *gin.Context)
	List(c *gin.Context)
	ListBySeason(c *gin.Context)
	ListByLeague(c *gin.Context)
	ListByClub(c *gin.Context)
	ListUpcoming(c *gin.Context)
	ListLive(c *gin.Context)
	Update(c *gin.Context)
	Delete(c *gin.Context)
}

type matchHandler struct {
	matchService service.MatchService
}

func NewMatchHandler(matchService service.MatchService) MatchHandler {
	return &matchHandler{matchService: matchService}
}

// Create match godoc
//
//	@Summary		Create match
//	@Tags			matches
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			body	body		dto.CreateMatchRequest	true	"Request body"
//	@Success		201	{object}	response.Response
//	@Failure		401	{object}	response.Response
//	@Failure		400	{object}	response.Response
//	@Router			/api/v1/matches [post]
func (h *matchHandler) Create(c *gin.Context) {
	var req dto.CreateMatchRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	authCtx := getAuthContext(c)
	match, err := h.matchService.Create(c.Request.Context(), &req, authCtx)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Created(c, match)
}

// Get match godoc
//
//	@Summary		Get match
//	@Tags			matches
//	@Produce		json
//	@Param			id	path	string	true	"Match ID"
//	@Success		200	{object}	response.Response
//	@Failure		400	{object}	response.Response
//	@Router			/api/v1/matches/{id} [get]
func (h *matchHandler) GetByID(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "Invalid ID")
		return
	}

	match, err := h.matchService.GetByID(c.Request.Context(), id)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, match)
}

// List matches godoc
//
//	@Summary		List matches
//	@Tags			matches
//	@Produce		json
//	@Param			sort	query	string	false	"Sort field"
//	@Param			order	query	string	false	"asc|desc"
//	@Param			page	query	int	false	"Page number"
//	@Param			per_page	query	int	false	"Items per page"
//	@Success		200	{object}	response.Response
//	@Failure		400	{object}	response.Response
//	@Router			/api/v1/matches [get]
func (h *matchHandler) List(c *gin.Context) {
	page, pageSize := utils.GetPagination(c)

	filters := make(map[string]string)
	for key, values := range c.Request.URL.Query() {
		if len(values) > 0 {
			filters[key] = values[0]
		}
	}

	sortBy := c.Query("sort")
	sortOrder := c.Query("order")

	matches, err := h.matchService.List(c.Request.Context(), page, pageSize, filters, sortBy, sortOrder)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, matches)
}

// List matches by season godoc
//
//	@Summary		List matches by season
//	@Tags			matches
//	@Produce		json
//	@Param			id	path	string	true	"Season ID"
//	@Success		200	{object}	response.Response
//	@Failure		400	{object}	response.Response
//	@Router			/api/v1/seasons/{id}/matches [get]
func (h *matchHandler) ListBySeason(c *gin.Context) {
	seasonID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "Invalid season ID")
		return
	}

	page, pageSize := utils.GetPagination(c)

	matches, err := h.matchService.ListBySeason(c.Request.Context(), seasonID, page, pageSize)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, matches)
}

// List matches by league godoc
//
//	@Summary		List matches by league
//	@Tags			matches
//	@Produce		json
//	@Param			id	path	string	true	"League ID"
//	@Success		200	{object}	response.Response
//	@Failure		400	{object}	response.Response
//	@Router			/api/v1/leagues/{id}/matches [get]
func (h *matchHandler) ListByLeague(c *gin.Context) {
	leagueID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "Invalid league ID")
		return
	}

	page, pageSize := utils.GetPagination(c)

	matches, err := h.matchService.ListByLeague(c.Request.Context(), leagueID, page, pageSize)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, matches)
}

// List matches by club godoc
//
//	@Summary		List matches by club
//	@Tags			matches
//	@Produce		json
//	@Param			id	path	string	true	"Club ID"
//	@Success		200	{object}	response.Response
//	@Failure		400	{object}	response.Response
//	@Router			/api/v1/clubs/{id}/matches [get]
func (h *matchHandler) ListByClub(c *gin.Context) {
	clubID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "Invalid club ID")
		return
	}

	page, pageSize := utils.GetPagination(c)

	matches, err := h.matchService.ListByClub(c.Request.Context(), clubID, page, pageSize)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, matches)
}

// List upcoming matches godoc
//
//	@Summary		List upcoming matches
//	@Tags			matches
//	@Produce		json
//	@Success		200	{object}	response.Response
//	@Failure		400	{object}	response.Response
//	@Router			/api/v1/matches/upcoming [get]
func (h *matchHandler) ListUpcoming(c *gin.Context) {
	page, pageSize := utils.GetPagination(c)

	matches, err := h.matchService.ListUpcoming(c.Request.Context(), page, pageSize)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, matches)
}

// List live matches godoc
//
//	@Summary		List live matches
//	@Tags			matches
//	@Produce		json
//	@Success		200	{object}	response.Response
//	@Failure		400	{object}	response.Response
//	@Router			/api/v1/matches/live [get]
func (h *matchHandler) ListLive(c *gin.Context) {
	matches, err := h.matchService.ListLive(c.Request.Context())
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, matches)
}

// Update match godoc
//
//	@Summary		Update match
//	@Tags			matches
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id	path	string	true	"Match ID"
//	@Param			body	body		dto.UpdateMatchRequest	true	"Request body"
//	@Success		200	{object}	response.Response
//	@Failure		401	{object}	response.Response
//	@Failure		400	{object}	response.Response
//	@Router			/api/v1/matches/{id} [put]
func (h *matchHandler) Update(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "Invalid ID")
		return
	}

	var req dto.UpdateMatchRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	authCtx := getAuthContext(c)
	match, err := h.matchService.Update(c.Request.Context(), id, &req, authCtx)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, match)
}

// Delete match godoc
//
//	@Summary		Delete match
//	@Tags			matches
//	@Produce		json
//	@Security		BearerAuth
//	@Param			id	path	string	true	"Match ID"
//	@Success		200	{object}	response.Response
//	@Failure		401	{object}	response.Response
//	@Failure		400	{object}	response.Response
//	@Router			/api/v1/matches/{id} [delete]
func (h *matchHandler) Delete(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "Invalid ID")
		return
	}

	authCtx := getAuthContext(c)
	if err := h.matchService.Delete(c.Request.Context(), id, authCtx); err != nil {
		response.Error(c, err)
		return
	}

	response.SuccessWithMessage(c, nil, "Match deleted successfully")
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
