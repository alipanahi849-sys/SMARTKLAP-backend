package handler

import (
	"clap/internal/modules/mobilehome/service"
	"clap/internal/shared/middleware"
	"clap/internal/shared/response"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// HomeHandler serves the mobile Home aggregates (contract §3).
type HomeHandler interface {
	Stadium(c *gin.Context)
	Club(c *gin.Context)
}

type homeHandler struct {
	svc service.HomeService
}

func NewHomeHandler(svc service.HomeService) HomeHandler {
	return &homeHandler{svc: svc}
}

// Stadium home godoc
//
//	@Summary		Stadium home
//	@Tags			mobile-home
//	@Produce		json
//	@Security		BearerAuth
//	@Success		200	{object}	response.Response
//	@Failure		401	{object}	response.Response
//	@Failure		400	{object}	response.Response
//	@Router			/api/v1/mobile/home/stadium [get]
func (h *homeHandler) Stadium(c *gin.Context) {
	userID := middleware.GetUserID(c)
	if userID == uuid.Nil {
		response.Unauthorized(c, "Invalid user")
		return
	}

	result, err := h.svc.Stadium(c.Request.Context(), userID)
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, result)
}

// Club home godoc
//
//	@Summary		Club home
//	@Tags			mobile-home
//	@Produce		json
//	@Security		BearerAuth
//	@Success		200	{object}	response.Response
//	@Failure		401	{object}	response.Response
//	@Failure		400	{object}	response.Response
//	@Router			/api/v1/mobile/home/club [get]
func (h *homeHandler) Club(c *gin.Context) {
	result, err := h.svc.Club(c.Request.Context())
	if err != nil {
		response.Error(c, err)
		return
	}
	response.Success(c, result)
}
