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

func (h *matchHandler) ListUpcoming(c *gin.Context) {
	page, pageSize := utils.GetPagination(c)

	matches, err := h.matchService.ListUpcoming(c.Request.Context(), page, pageSize)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, matches)
}

func (h *matchHandler) ListLive(c *gin.Context) {
	matches, err := h.matchService.ListLive(c.Request.Context())
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, matches)
}

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
