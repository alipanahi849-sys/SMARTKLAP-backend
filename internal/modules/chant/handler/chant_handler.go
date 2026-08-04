package handler

import (
	"clap/internal/modules/chant/service"
	"clap/internal/shared/middleware"
	"clap/internal/shared/response"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// ChantHandler serves the mobile Chants screens (contract §4).
type ChantHandler interface {
	List(c *gin.Context)
	Lyrics(c *gin.Context)
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
//	@Param			search	query	string	false	"Search query"
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

	var matchID *uuid.UUID
	if raw, ok := c.GetQuery("match_id"); ok && raw != "" {
		parsed, err := uuid.Parse(raw)
		if err != nil {
			response.BadRequest(c, "Invalid match_id")
			return
		}
		matchID = &parsed
	}

	result, err := h.svc.List(c.Request.Context(), userID, matchID, c.Query("search"))
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
//	@Param			chant_id	path	string	true	"Chant ID"
//	@Success		200	{object}	response.Response
//	@Failure		401	{object}	response.Response
//	@Failure		400	{object}	response.Response
//	@Router			/api/v1/chants/{chant_id}/lyrics [get]
func (h *chantHandler) Lyrics(c *gin.Context) {
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

	result, svcErr := h.svc.Lyrics(c.Request.Context(), chantID, mode)
	if svcErr != nil {
		response.Error(c, svcErr)
		return
	}
	response.Success(c, result)
}
