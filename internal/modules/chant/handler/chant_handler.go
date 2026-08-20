package handler

import (
	"strings"

	"clap/internal/modules/chant/dto"
	"clap/internal/modules/chant/models"
	"clap/internal/modules/chant/service"
	"clap/internal/shared/middleware"
	"clap/internal/shared/response"
	"clap/internal/shared/utils"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// ChantHandler serves the mobile Chants screens (contract §4).
type ChantHandler interface {
	List(c *gin.Context)
	Lyrics(c *gin.Context)
	Complete(c *gin.Context)
	Cancel(c *gin.Context)
	TodayStats(c *gin.Context)
	Program(c *gin.Context)

	GetPoints(c *gin.Context)
	UpdatePoints(c *gin.Context)
	SetOnlineChant(c *gin.Context)
	ListOnlineChants(c *gin.Context)
	UnsetOnlineChant(c *gin.Context)
}

type chantHandler struct {
	svc service.ChantService
}

func NewChantHandler(svc service.ChantService) ChantHandler {
	return &chantHandler{svc: svc}
}

// List chants godoc
//
//	@Summary		List chants
//	@Tags			chants
//	@Produce		json
//	@Security		BearerAuth
//	@Param			search		query	string	false	"Search query"
//	@Param			match_id	query	string	false	"Match ID filter"
//	@Param			cursor		query	string	false	"Chant ID of the last item from the previous page"
//	@Param			limit		query	int		false	"Items per page (default 20, max 100)"
//	@Success		200	{object}	response.Response
//	@Failure		401	{object}	response.Response
//	@Failure		400	{object}	response.Response
//	@Router			/api/v1/chants [get]
func (h *chantHandler) List(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == uuid.Nil {
		response.Unauthorized(c, "Invalid user")
		return
	}

	limit := utils.GetMobileCursorLimit(c)

	var matchID *uuid.UUID
	if raw, ok := c.GetQuery("match_id"); ok && raw != "" {
		parsed, err := uuid.Parse(raw)
		if err != nil {
			response.BadRequest(c, "Invalid match_id")
			return
		}
		matchID = &parsed
	}

	var cursor *uuid.UUID
	if raw := strings.TrimSpace(c.Query("cursor")); raw != "" {
		id, err := uuid.Parse(raw)
		if err != nil {
			response.BadRequest(c, "Invalid cursor")
			return
		}
		cursor = &id
	}

	result, err := h.svc.List(c.Request.Context(), userID, dto.ChantListFilters{
		MatchID: matchID,
		Search:  c.Query("search"),
		Cursor:  cursor,
		Limit:   limit,
	})
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, result)
}

// Chant lyrics godoc
//
//	@Summary		Chant lyrics
//	@Tags			chants
//	@Produce		json
//	@Security		BearerAuth
//	@Param			chant_id	path	string	true	"Chant ID, or song ID when source=catalog"
//	@Param			mode		query	string	false	"preview or main (logged only)"
//	@Param			source		query	string	false	"catalog or online (default online)"
//	@Success		200	{object}	response.Response
//	@Failure		401	{object}	response.Response
//	@Failure		400	{object}	response.Response
//	@Router			/api/v1/chants/{chant_id}/lyrics [get]
func (h *chantHandler) Lyrics(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == uuid.Nil {
		response.Unauthorized(c, "Invalid user")
		return
	}

	chantID, err := uuid.Parse(c.Param("chant_id"))
	if err != nil {
		response.BadRequest(c, "Invalid chant ID")
		return
	}

	mode := c.DefaultQuery("mode", "main")
	if mode != "main" && mode != "preview" {
		response.BadRequest(c, "mode must be 'preview' or 'main'")
		return
	}

	source, ok := parseSource(c)
	if !ok {
		return
	}

	result, svcErr := h.svc.Lyrics(c.Request.Context(), userID, chantID, mode, source)
	if svcErr != nil {
		response.Error(c, svcErr)
		return
	}
	response.Success(c, result)
}

// parseSource reads ?source=, defaulting to the scheduled online chants.
func parseSource(c *gin.Context) (string, bool) {
	raw := strings.TrimSpace(c.Query("source"))
	switch raw {
	case "":
		return models.SourceOnline, true
	case models.SourceCatalog, models.SourceOnline:
		return raw, true
	default:
		response.BadRequest(c, "source must be 'catalog' or 'online'")
		return "", false
	}
}

// Complete chant godoc
//
//	@Summary		Complete chant
//	@Tags			chants
//	@Produce		json
//	@Security		BearerAuth
//	@Param			chant_id	path	string	true	"Chant ID, or song ID when source=catalog"
//	@Param			source		query	string	false	"catalog or online (default online)"
//	@Success		200	{object}	response.Response
//	@Failure		401	{object}	response.Response
//	@Failure		400	{object}	response.Response
//	@Failure		404	{object}	response.Response
//	@Failure		422	{object}	response.Response
//	@Router			/api/v1/chants/{chant_id}/complete [post]
func (h *chantHandler) Complete(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == uuid.Nil {
		response.Unauthorized(c, "Invalid user")
		return
	}

	chantID, err := uuid.Parse(c.Param("chant_id"))
	if err != nil {
		response.BadRequest(c, "Invalid chant ID")
		return
	}

	source, ok := parseSource(c)
	if !ok {
		return
	}

	result, svcErr := h.svc.Complete(c.Request.Context(), userID, chantID, source)
	if svcErr != nil {
		response.Error(c, svcErr)
		return
	}
	response.SuccessWithMessage(c, result, "Chant completed successfully")
}

// Cancel chant godoc
//
//	@Summary		Cancel chant
//	@Description	Records that the user left a live chant before it finished. The
//	@Description	chant is settled as cancelled: it is worth no points and cannot
//	@Description	be earned later.
//	@Tags			chants
//	@Produce		json
//	@Security		BearerAuth
//	@Param			chant_id	path		string	true	"Chant ID"
//	@Success		200			{object}	response.Response
//	@Failure		400			{object}	response.Response
//	@Failure		401			{object}	response.Response
//	@Failure		404			{object}	response.Response
//	@Router			/api/v1/chants/{chant_id}/cancel [post]
func (h *chantHandler) Cancel(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == uuid.Nil {
		response.Unauthorized(c, "Invalid user")
		return
	}

	chantID, err := uuid.Parse(c.Param("chant_id"))
	if err != nil {
		response.BadRequest(c, "Invalid chant ID")
		return
	}

	if svcErr := h.svc.Cancel(c.Request.Context(), userID, chantID); svcErr != nil {
		response.Error(c, svcErr)
		return
	}
	response.SuccessWithMessage(c, nil, "Chant cancelled")
}

// Today stats godoc
//
//	@Summary		Today's chant points
//	@Tags			chants
//	@Produce		json
//	@Security		BearerAuth
//	@Success		200	{object}	response.Response
//	@Failure		401	{object}	response.Response
//	@Router			/api/v1/chants/me/today [get]
func (h *chantHandler) TodayStats(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == uuid.Nil {
		response.Unauthorized(c, "Invalid user")
		return
	}

	result, err := h.svc.TodayStats(c.Request.Context(), userID)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, result)
}

// Chant program godoc
//
//	@Summary		Chant scores
//	@Description	Today's points scoreboard for the Home "Chants Program" card.
//	@Tags			chants
//	@Produce		json
//	@Security		BearerAuth
//	@Param			limit	query	int	false	"Items to return (default 20, max 100)"
//	@Success		200	{object}	response.Response
//	@Failure		401	{object}	response.Response
//	@Router			/api/v1/chants/program [get]
func (h *chantHandler) Program(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == uuid.Nil {
		response.Unauthorized(c, "Invalid user")
		return
	}

	result, err := h.svc.Program(c.Request.Context(), userID, utils.GetMobileCursorLimit(c))
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, result)
}

// GetPoints godoc
//
//	@Summary		Get chant point values
//	@Tags			admin-chants
//	@Produce		json
//	@Security		BearerAuth
//	@Success		200	{object}	response.Response
//	@Failure		401	{object}	response.Response
//	@Failure		403	{object}	response.Response
//	@Router			/api/v1/admin/settings/chant-points [get]
func (h *chantHandler) GetPoints(c *gin.Context) {
	result, err := h.svc.GetPointsSettings(c.Request.Context())
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, result)
}

// UpdatePoints godoc
//
//	@Summary		Set chant point values
//	@Description	Catalog songs and online chants are scored separately.
//	@Tags			admin-chants
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			body	body	dto.UpdateChantPointsRequest	true	"Point values"
//	@Success		200	{object}	response.Response
//	@Failure		400	{object}	response.Response
//	@Failure		401	{object}	response.Response
//	@Failure		403	{object}	response.Response
//	@Router			/api/v1/admin/settings/chant-points [put]
func (h *chantHandler) UpdatePoints(c *gin.Context) {
	var req dto.UpdateChantPointsRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	result, err := h.svc.UpdatePointsSettings(c.Request.Context(), req)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.SuccessWithMessage(c, result, "Chant points updated")
}

// SetOnlineChant godoc
//
//	@Summary		Schedule an online chant
//	@Description	Promotes a song from the predefined chant list into a live chant for a match.
//	@Tags			admin-chants
//	@Accept			json
//	@Produce		json
//	@Security		BearerAuth
//	@Param			body	body	dto.SetOnlineChantRequest	true	"Online chant"
//	@Success		201	{object}	response.Response
//	@Failure		400	{object}	response.Response
//	@Failure		401	{object}	response.Response
//	@Failure		403	{object}	response.Response
//	@Failure		404	{object}	response.Response
//	@Router			/api/v1/admin/chants [post]
func (h *chantHandler) SetOnlineChant(c *gin.Context) {
	var req dto.SetOnlineChantRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	result, err := h.svc.SetOnlineChant(c.Request.Context(), middleware.GetUserID(c), req)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.CreatedWithMessage(c, result, "Online chant scheduled")
}

// ListOnlineChants godoc
//
//	@Summary		List scheduled online chants
//	@Tags			admin-chants
//	@Produce		json
//	@Security		BearerAuth
//	@Param			match_id	query	string	false	"Match ID filter"
//	@Param			limit		query	int		false	"Items to return (default 20, max 100)"
//	@Success		200	{object}	response.Response
//	@Failure		401	{object}	response.Response
//	@Failure		403	{object}	response.Response
//	@Router			/api/v1/admin/chants [get]
func (h *chantHandler) ListOnlineChants(c *gin.Context) {
	var matchID *uuid.UUID
	if raw := strings.TrimSpace(c.Query("match_id")); raw != "" {
		parsed, err := uuid.Parse(raw)
		if err != nil {
			response.BadRequest(c, "Invalid match_id")
			return
		}
		matchID = &parsed
	}

	result, err := h.svc.ListOnlineChants(c.Request.Context(), matchID, utils.GetMobileCursorLimit(c))
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, result)
}

// UnsetOnlineChant godoc
//
//	@Summary		Unschedule an online chant
//	@Tags			admin-chants
//	@Produce		json
//	@Security		BearerAuth
//	@Param			chant_id	path	string	true	"Chant ID"
//	@Success		200	{object}	response.Response
//	@Failure		401	{object}	response.Response
//	@Failure		403	{object}	response.Response
//	@Failure		404	{object}	response.Response
//	@Router			/api/v1/admin/chants/{chant_id} [delete]
func (h *chantHandler) UnsetOnlineChant(c *gin.Context) {
	chantID, err := uuid.Parse(c.Param("chant_id"))
	if err != nil {
		response.BadRequest(c, "Invalid chant ID")
		return
	}

	if svcErr := h.svc.UnsetOnlineChant(c.Request.Context(), chantID); svcErr != nil {
		response.Error(c, svcErr)
		return
	}
	response.SuccessWithMessage(c, response.EmptyObject, "Online chant unscheduled")
}
