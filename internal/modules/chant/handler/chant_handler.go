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
	Countdown(c *gin.Context)
	Lyrics(c *gin.Context)
	Complete(c *gin.Context)
}

type chantHandler struct {
	svc service.ChantService
}

func NewChantHandler(svc service.ChantService) ChantHandler {
	return &chantHandler{svc: svc}
}

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

func (h *chantHandler) Countdown(c *gin.Context) {
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

	result, svcErr := h.svc.Countdown(c.Request.Context(), userID, chantID)
	if svcErr != nil {
		response.Error(c, svcErr)
		return
	}
	response.Success(c, result)
}

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

	result, svcErr := h.svc.Complete(c.Request.Context(), userID, chantID)
	if svcErr != nil {
		response.Error(c, svcErr)
		return
	}
	response.Success(c, result)
}
