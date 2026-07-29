package handler

import (
	"clap/internal/modules/clubseason/dto"
	"clap/internal/modules/clubseason/service"
	"clap/internal/shared/middleware"
	"clap/internal/shared/response"
	"clap/internal/shared/utils"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type ClubSeasonHandler interface {
	AddClubToSeason(c *gin.Context)
	RemoveClubFromSeason(c *gin.Context)
	ListClubsInSeason(c *gin.Context)
	ListSeasonsForClub(c *gin.Context)
	UpdateStatus(c *gin.Context)
}

type clubSeasonHandler struct {
	clubSeasonService service.ClubSeasonService
}

func NewClubSeasonHandler(clubSeasonService service.ClubSeasonService) ClubSeasonHandler {
	return &clubSeasonHandler{clubSeasonService: clubSeasonService}
}

func (h *clubSeasonHandler) AddClubToSeason(c *gin.Context) {
	var req dto.CreateClubSeasonRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	authCtx := getAuthContext(c)
	clubSeason, err := h.clubSeasonService.AddClubToSeason(c.Request.Context(), &req, authCtx)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Created(c, clubSeason)
}

func (h *clubSeasonHandler) RemoveClubFromSeason(c *gin.Context) {
	clubID, err := uuid.Parse(c.Param("club_id"))
	if err != nil {
		response.BadRequest(c, "Invalid club ID")
		return
	}

	seasonID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "Invalid season ID")
		return
	}

	authCtx := getAuthContext(c)
	if err := h.clubSeasonService.RemoveClubFromSeason(c.Request.Context(), clubID, seasonID, authCtx); err != nil {
		response.Error(c, err)
		return
	}

	response.SuccessWithMessage(c, nil, "Club removed from season successfully")
}

func (h *clubSeasonHandler) ListClubsInSeason(c *gin.Context) {
	seasonID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "Invalid season ID")
		return
	}

	page, pageSize := utils.GetPagination(c)

	clubSeasons, err := h.clubSeasonService.ListClubsInSeason(c.Request.Context(), seasonID, page, pageSize)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, clubSeasons)
}

func (h *clubSeasonHandler) ListSeasonsForClub(c *gin.Context) {
	clubID, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "Invalid club ID")
		return
	}

	page, pageSize := utils.GetPagination(c)

	clubSeasons, err := h.clubSeasonService.ListSeasonsForClub(c.Request.Context(), clubID, page, pageSize)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, clubSeasons)
}

func (h *clubSeasonHandler) UpdateStatus(c *gin.Context) {
	id, err := uuid.Parse(c.Param("id"))
	if err != nil {
		response.BadRequest(c, "Invalid ID")
		return
	}

	var req dto.UpdateClubSeasonRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	authCtx := getAuthContext(c)
	clubSeason, err := h.clubSeasonService.UpdateStatus(c.Request.Context(), id, &req, authCtx)
	if err != nil {
		response.Error(c, err)
		return
	}

	response.Success(c, clubSeason)
}

func getAuthContext(c *gin.Context) *utils.AuthorizationContext {
	userID := middleware.GetUserID(c)
	roles := middleware.GetUserRoles(c)
	return utils.NewAuthorizationContext(userID, roles, nil)
}
