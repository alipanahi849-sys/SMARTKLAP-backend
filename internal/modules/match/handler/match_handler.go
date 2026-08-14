package handler

import (
	"strings"

	"clap/internal/modules/match/dto"
	"clap/internal/modules/match/service"
	"clap/internal/shared/response"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type MatchHandler interface {
	GetCurrent(c *gin.Context)
	GetByID(c *gin.Context)
	GetPlayer(c *gin.Context)
	SearchTeams(c *gin.Context)
	GetFeaturedClub(c *gin.Context)
	SetFeaturedClub(c *gin.Context)
	Sync(c *gin.Context)
}

type matchHandler struct {
	svc service.MatchService
}

func NewMatchHandler(svc service.MatchService) MatchHandler {
	return &matchHandler{svc: svc}
}

// GetCurrent godoc
//
//	@Summary		Current featured-club match
//	@Tags			matches
//	@Produce		json
//	@Security		BearerAuth
//	@Success		200	{object}	response.Response
//	@Failure		401	{object}	response.Response
//	@Router			/api/v1/matches/current [get]
func (h *matchHandler) GetCurrent(c *gin.Context) {
	result, err := h.svc.GetCurrent(c.Request.Context())
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, dto.CurrentMatchEnvelope{Match: result})
}

// GetByID godoc
//
//	@Summary		Match detail
//	@Tags			matches
//	@Produce		json
//	@Security		BearerAuth
//	@Param			match_id	path	string	true	"Match ID"
//	@Success		200	{object}	response.Response
//	@Failure		401	{object}	response.Response
//	@Failure		404	{object}	response.Response
//	@Router			/api/v1/matches/{match_id} [get]
func (h *matchHandler) GetByID(c *gin.Context) {
	id, err := uuid.Parse(c.Param("match_id"))
	if err != nil {
		response.BadRequest(c, "Invalid match ID")
		return
	}
	result, svcErr := h.svc.GetByID(c.Request.Context(), id)
	if svcErr != nil {
		response.Error(c, svcErr)
		return
	}
	response.Success(c, result)
}

// GetPlayer godoc
//
//	@Summary		Player detail
//	@Tags			stats
//	@Produce		json
//	@Security		BearerAuth
//	@Param			player_id	path	string	true	"Player ID"
//	@Success		200	{object}	response.Response
//	@Failure		401	{object}	response.Response
//	@Failure		404	{object}	response.Response
//	@Router			/api/v1/players/{player_id} [get]
func (h *matchHandler) GetPlayer(c *gin.Context) {
	id, err := uuid.Parse(c.Param("player_id"))
	if err != nil {
		response.BadRequest(c, "Invalid player ID")
		return
	}
	result, svcErr := h.svc.GetPlayer(c.Request.Context(), id)
	if svcErr != nil {
		response.Error(c, svcErr)
		return
	}
	response.Success(c, result)
}

// SearchTeams godoc
//
//	@Summary		Search football teams
//	@Tags			admin-matches
//	@Produce		json
//	@Security		BearerAuth
//	@Param			search	query	string	true	"Team name"
//	@Success		200	{object}	response.Response
//	@Failure		401	{object}	response.Response
//	@Failure		403	{object}	response.Response
//	@Router			/api/v1/admin/football/teams [get]
func (h *matchHandler) SearchTeams(c *gin.Context) {
	result, err := h.svc.SearchProviderTeams(c.Request.Context(), strings.TrimSpace(c.Query("search")))
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, result)
}

// GetFeaturedClub godoc
//
//	@Summary		Get featured club
//	@Tags			admin-matches
//	@Produce		json
//	@Security		BearerAuth
//	@Success		200	{object}	response.Response
//	@Failure		401	{object}	response.Response
//	@Failure		403	{object}	response.Response
//	@Router			/api/v1/admin/settings/featured-club [get]
func (h *matchHandler) GetFeaturedClub(c *gin.Context) {
	result, err := h.svc.GetFeaturedClub(c.Request.Context())
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, result)
}

// SetFeaturedClub godoc
//
//	@Summary		Set featured club
//	@Tags			admin-matches
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			body	body	dto.SetFeaturedClubRequest	true	"Featured club"
//	@Success		200	{object}	response.Response
//	@Failure		401	{object}	response.Response
//	@Failure		403	{object}	response.Response
//	@Router			/api/v1/admin/settings/featured-club [put]
func (h *matchHandler) SetFeaturedClub(c *gin.Context) {
	var req dto.SetFeaturedClubRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}
	result, err := h.svc.SetFeaturedClub(c.Request.Context(), req)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.SuccessWithMessage(c, result, "Featured club updated")
}

// Sync godoc
//
//	@Summary		Sync featured-club fixtures
//	@Tags			admin-matches
//	@Produce		json
//	@Security		BearerAuth
//	@Success		200	{object}	response.Response
//	@Failure		401	{object}	response.Response
//	@Failure		403	{object}	response.Response
//	@Router			/api/v1/admin/matches/sync [post]
func (h *matchHandler) Sync(c *gin.Context) {
	result, err := h.svc.SyncNow(c.Request.Context())
	if err != nil {
		response.Error(c, err)
		return
	}
	response.SuccessWithMessage(c, result, "Matches synced")
}
